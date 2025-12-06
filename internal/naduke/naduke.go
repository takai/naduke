package naduke

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	DefaultModel         = "granite4:3b-h"
	DefaultHost          = "localhost"
	DefaultPort          = 11434
	DefaultTemperature   = 0.0
	DefaultTopK          = 1
	DefaultTopP          = 1.0
	DefaultRepeatPenalty = 1.0
	readBytes            = 8 * 1024
)

var (
	systemPrompt = strings.TrimSpace(`
You are a tool that generates file names.
You MUST follow these rules:
- Output only a single file name without extension.
- Do not add an extension.
- Use only lowercase letters a-z, digits 0-9, and underscores.
- No spaces, no hyphens, no other characters.
- Less than or equal than 30 characters.
- Make it concise but descriptive of the text content.
`)
	userPrompt = strings.TrimSpace(`
Generate an appropriate file name for this text file content.

<content>
%s
</content>
`)
	invalidChars = regexp.MustCompile(`[^a-z0-9_]`)
)

type Options struct {
	Host          string
	Port          int
	Server        string
	Model         string
	Temperature   float64
	TopK          int
	TopP          float64
	RepeatPenalty float64
}

type client struct {
	http *http.Client
	uri  *url.URL
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  chatOptions   `json:"options"`
}

type chatOptions struct {
	Temperature   float64 `json:"temperature"`
	TopK          int     `json:"top_k"`
	TopP          float64 `json:"top_p"`
	RepeatPenalty float64 `json:"repeat_penalty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message *chatMessage `json:"message"`
	// Some Ollama responses use "response" instead.
	Response string `json:"response"`
}

func NewClient(opts Options) (*client, error) {
	uri, err := buildURI(opts)
	if err != nil {
		return nil, err
	}
	return &client{
		http: &http.Client{},
		uri:  uri,
	}, nil
}

func buildURI(opts Options) (*url.URL, error) {
	if opts.Server != "" {
		parsed, err := url.Parse(opts.Server)
		if err != nil {
			return nil, fmt.Errorf("invalid server URL: %w", err)
		}
		if parsed.Scheme == "" {
			parsed.Scheme = "http"
		}
		parsed.Path = "/api/chat"
		return parsed, nil
	}
	return &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		Path:   "/api/chat",
	}, nil
}

func (c *client) GenerateName(model string, temperature float64, topK int, topP float64, repeatPenalty float64, content string) (string, error) {
	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf(userPrompt, content)},
		},
		Stream: false,
		Options: chatOptions{
			Temperature:   temperature,
			TopK:          topK,
			TopP:          topP,
			RepeatPenalty: repeatPenalty,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.uri.String(), bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("request model: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("model request failed (%d): %s", resp.StatusCode, string(body))
	}

	var decoded chatResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	switch {
	case decoded.Message != nil && decoded.Message.Content != "":
		return decoded.Message.Content, nil
	case decoded.Response != "":
		return decoded.Response, nil
	default:
		return "", errors.New("empty response from model")
	}
}

func ReadSample(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, readBytes)
	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(buf[:n]), nil
}

func EnsureTextSample(sample string, path string) (string, error) {
	if sample == "" {
		return "", nil
	}
	if strings.ContainsRune(sample, '\x00') {
		return "", fmt.Errorf("%s does not look like a text file (NUL byte found)", path)
	}
	if !utf8.ValidString(sample) {
		return "", fmt.Errorf("%s is not valid UTF-8 text", path)
	}
	return sample, nil
}

func SanitizeName(raw string) string {
	name := strings.TrimSpace(raw)
	if idx := strings.IndexByte(name, '\n'); idx >= 0 {
		name = name[:idx]
	}
	name = strings.ToLower(name)
	name = invalidChars.ReplaceAllString(name, "_")
	if len(name) > 30 {
		name = name[:30]
	}
	name = strings.Trim(name, "_")
	if name == "" {
		return "file"
	}
	return name
}

func RenameFile(path, newName string) error {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	destination := filepath.Join(dir, newName+ext)

	absSrc, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("absolutize source: %w", err)
	}
	absDst, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("absolutize destination: %w", err)
	}
	if absSrc == absDst {
		fmt.Printf("%s -> %s\n", path, destination)
		return nil
	}
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("destination already exists - %s", destination)
	}

	if err := os.Rename(path, destination); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	fmt.Printf("%s -> %s\n", path, destination)
	return nil
}
