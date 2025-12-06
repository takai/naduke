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

## Testing
```sh
GOCACHE=$(pwd)/.cache/go-build go test ./...
```

## License
MIT
