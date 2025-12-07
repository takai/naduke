# naduke

CLI tool that guesses better file names from file content using a local Ollama model, then renames the files while keeping their extensions.

## Requirements
- Go 1.22+
- Ollama installed and running locally or remotely.
- The model `granite4:3b-h` available in Ollama (or another model you specify).

### Model download (granite4:3b-h)
```sh
ollama pull granite4:3b-h
```

## Installation
```sh
go install github.com/takai/naduke/cmd/naduke@latest
```

Or build locally:
```sh
go build -o naduke ./cmd/naduke
```

## Usage
```sh
naduke [options] FILE...
```

Options (from `naduke -h`):
- `-host` Ollama host (default: `localhost`)
- `-port` Ollama port (default: `11434`)
- `-server` Full Ollama server URL (overrides host/port)
- `-model` Model name (default: `granite4:3b-h`)
- `-temperature` Sampling temperature (default: `0.0`)
- `-top_k` Top-k sampling (default: `1`)
- `-top_p` Top-p sampling (default: `1.0`)
- `-repeat_penalty` Repeat penalty (default: `1.0`)
- `-dry-run` Show suggested names without renaming (note: actual rename run may produce a different suggestion because LLM outputs can vary)
- `-prefix` Prefix to prepend to the generated name
- `-dir` Destination directory for renamed files (default: same as source)
- `-h`, `-help` Show help

Examples:
```sh
# Basic rename with defaults
naduke notes/todo.txt

# Multiple files at once
naduke docs/*.md

# Custom server URL
naduke -server http://ollama.example.com:11434 draft.txt

# Add a prefix to suggestions
naduke -prefix meeting_ notes.txt

# Rename into another directory
naduke -dir out/ docs/*.md
```

## Behavior
- Reads the first 1,000 characters (up to ~4KB); aborts on NUL bytes or invalid UTF-8.
- Sends system/user prompts to `/api/chat` (no streaming).
- Sanitizes model output; if empty after sanitization, uses `file`.
- Keeps the original extension (e.g., `draft.md` -> `summary.md`).
- Allows choosing a different destination directory via `-dir`; source file must be reachable and destination dir must exist.
- Fails if the destination already exists.
- Dry-run prints suggestions only; due to LLM variability, a later non-dry run might produce a different name.
- Validates model output against naming rules (single token, lowercase a-z0-9_, max 30 chars, no extension).
- Applies an optional prefix as provided, then appends the model output.

Parameter notes (you do not usually need to change these):
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
