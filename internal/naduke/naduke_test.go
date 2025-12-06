package naduke

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"Title With Spaces\nand more", "title_with_spaces"},
		{"Hello-World!", "hello_world"},
		{strings.Repeat("a", 40), strings.Repeat("a", 30)},
		{"__!__", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := SanitizeName(tt.in)
			if got != tt.want {
				t.Fatalf("SanitizeName(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEnsureTextSample(t *testing.T) {
	t.Parallel()

	_, err := EnsureTextSample("hi\x00", "sample.txt")
	if err == nil {
		t.Fatalf("expected error on NUL byte")
	}

	invalidUTF8 := string([]byte{0xff, 0xfe})
	_, err = EnsureTextSample(invalidUTF8, "sample.txt")
	if err == nil {
		t.Fatalf("expected error on invalid UTF-8")
	}

	out, err := EnsureTextSample("ok text", "sample.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok text" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestReadSample(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	content := strings.Repeat("x", readBytes+10)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	sample, err := ReadSample(path)
	if err != nil {
		t.Fatalf("ReadSample error: %v", err)
	}
	if len(sample) != readBytes {
		t.Fatalf("expected sample length %d, got %d", readBytes, len(sample))
	}
	if sample != content[:readBytes] {
		t.Fatalf("sample does not match expected prefix")
	}
}

func TestRenameFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := RenameFile(src, "renamed"); err != nil {
		t.Fatalf("rename failed: %v", err)
	}

	dst := filepath.Join(dir, "renamed.txt")
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination missing: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be gone")
	}

	// Destination already exists
	if err := os.WriteFile(src, []byte("again"), 0o644); err != nil {
		t.Fatalf("write source again: %v", err)
	}
	if err := os.WriteFile(dst, []byte("exists"), 0o644); err != nil {
		t.Fatalf("write existing dst: %v", err)
	}
	if err := RenameFile(src, "renamed"); err == nil {
		t.Fatalf("expected error when destination exists")
	}
}

func TestDestinationPath(t *testing.T) {
	t.Parallel()

	path := "/tmp/example/note.txt"
	dest := DestinationPath(path, "suggested_name")

	want := "/tmp/example/suggested_name.txt"
	if dest != want {
		t.Fatalf("DestinationPath = %q; want %q", dest, want)
	}
}

func TestGenerateName(t *testing.T) {
	t.Parallel()

	fakeTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		var payload chatRequest
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "test-model" {
			t.Fatalf("unexpected model: %s", payload.Model)
		}
		if payload.Options.Temperature != 0.5 {
			t.Fatalf("unexpected temperature: %v", payload.Options.Temperature)
		}
		if payload.Options.TopK != 3 {
			t.Fatalf("unexpected top_k: %v", payload.Options.TopK)
		}
		if payload.Options.TopP != 0.9 {
			t.Fatalf("unexpected top_p: %v", payload.Options.TopP)
		}
		if payload.Options.RepeatPenalty != 1.2 {
			t.Fatalf("unexpected repeat_penalty: %v", payload.Options.RepeatPenalty)
		}
		resp := chatResponse{
			Message: &chatMessage{
				Role:    "assistant",
				Content: "suggested_name",
			},
		}
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(resp); err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&buf),
			Header:     make(http.Header),
		}, nil
	})

	client := &client{
		http: &http.Client{Transport: fakeTransport},
		uri:  &url.URL{Scheme: "http", Host: "example.com", Path: "/api/chat"},
	}

	name, err := client.GenerateName("test-model", 0.5, 3, 0.9, 1.2, "hello")
	if err != nil {
		t.Fatalf("GenerateName error: %v", err)
	}
	if name != "suggested_name" {
		t.Fatalf("unexpected name: %q", name)
	}
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
