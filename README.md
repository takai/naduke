# naduke

CLI tool that guesses better file names from file content using a local Ollama model, then renames the files while keeping their extensions.

## Requirements
- Go 1.22+
- Running Ollama server with access to `granite4:3b-h` (or a model you specify).

## Installation
```sh
go build -o naduke ./cmd/naduke
```

## Usage
```sh
naduke [options] FILE...
```

Options:
- `--host` Ollama host (default: `localhost`)
- `--port` Ollama port (default: `11434`)
- `--server` Full Ollama server URL (overrides host/port)
- `--model` Model name (default: `granite4:3b-h`)
- `--temperature` Sampling temperature (default: `0.0`)
- `--top_k` Top-k sampling (default: `1`)
- `--top_p` Top-p sampling (default: `1.0`)
- `--repeat_penalty` Repeat penalty (default: `1.0`)
- `-h`, `--help` Show help

Examples:
```sh
# Basic rename with defaults
naduke notes/todo.txt

# Multiple files at once
naduke docs/*.md

# Custom server URL
naduke --server http://ollama.example.com:11434 draft.txt
```

## Behavior
- Reads first 8KB; aborts on NUL bytes or invalid UTF-8.
- Sends system/user prompts to `/api/chat` (no streaming).
- Sanitizes model output; if empty after sanitization, uses `file`.
- Keeps the original extension (e.g., `draft.md` -> `summary.md`).
- Fails if the destination already exists.

Parameter notes (for general users):
- `temperature`: Controls randomness/creativity. Higher = more varied suggestions; lower = safer/more deterministic.
- `top_k`: Limits candidates to the top-K tokens before sampling. Lower = conservative; higher = more diverse.
- `top_p`: Nucleus sampling; keeps the smallest set of tokens whose cumulative probability ≥ `top_p`. Higher = more diverse; lower = more focused.
- `repeat_penalty`: Penalizes repeating tokens. Higher than 1 discourages repetition; keep near 1 for normal behavior.

## Testing
```sh
GOCACHE=$(pwd)/.cache/go-build go test ./...
```

## License
MIT
