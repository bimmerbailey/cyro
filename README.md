# Cyro

A CLI tool for log analysis, powered by local LLMs.

Cyro parses, searches, and analyzes log files from the terminal. It combines traditional log tooling (pattern matching,
filtering, statistics) with AI-powered analysis via [Ollama](https://ollama.com) for summarization, root cause analysis,
and natural language querying over logs of any size.

## Features

- **Search** — Filter logs by regex pattern, level, and time range
- **Analyze** — Detect patterns, frequency shifts, and anomalies
- **Stats** — Line counts by level, error rates, top messages
- **Tail** — Live-follow log files with real-time filtering
- **AI Analysis** — Summarize logs, ask questions in plain English, identify root causes (planned)
- **RAG** — Query logs of any size using retrieval-augmented generation (planned)
- **Multi-format** — JSON, syslog, Apache/Nginx, and generic plaintext log parsing
- **Multiple output formats** — Text, JSON, and table output

## Requirements

- Go 1.25+
- [Ollama](https://ollama.com) (for AI features, optional for basic log analysis)

## Installation

### From source

```sh
git clone https://github.com/bimmerbailey/cyro.git
cd cyro
make build
```

The binary is built to `bin/cyro`.

### Install to GOPATH

```sh
make install
```

## Usage

```sh
# Search for errors in a log file
cyro search --level error /var/log/app.log

# Search with a regex pattern and time range
cyro search --pattern "timeout|refused" --since 1h /var/log/app.log

# Show log statistics
cyro stats /var/log/app.log

# Analyze patterns and top messages
cyro analyze --top 20 /var/log/app.log

# Live-tail with level filtering
cyro tail --level warn /var/log/app.log
```

### Global Flags

```
--config string   config file (default is $HOME/.cyro.yaml)
-f, --format string   output format: text, json, table (default "text")
-v, --verbose         enable verbose output
```

### Commands

| Command      | Description                                                     |
|--------------|-----------------------------------------------------------------|
| `search`     | Search and filter log entries by pattern, level, and time range |
| `analyze`    | Analyze log files for patterns, anomalies, and frequency        |
| `stats`      | Show aggregate statistics for a log file                        |
| `tail`       | Live-tail a log file with filtering                             |
| `version`    | Print version, commit, and build date                           |
| `completion` | Generate shell completion scripts (bash, zsh, fish, powershell) |

Run `cyro <command> --help` for details on each command.

## Configuration

Cyro reads configuration from `~/.cyro.yaml`, the current directory, or a path specified with `--config`. Environment
variables with the `CYRO_` prefix are also supported.

```yaml
# ~/.cyro.yaml
format: text
verbose: false
timestamp_formats:
  - "2006-01-02T15:04:05Z07:00"
  - "2006-01-02 15:04:05"
  - "Jan 02 15:04:05"
  - "02/Jan/2006:15:04:05 -0700"
```

## Project Structure

```
cyro/
├── main.go                        # Entrypoint
├── cmd/                           # Cobra command definitions
│   ├── root.go                    # Root command, Viper config init
│   ├── search.go                  # Search/filter logs
│   ├── analyze.go                 # Pattern analysis
│   ├── stats.go                   # Aggregate statistics
│   ├── tail.go                    # Live-tail with filtering
│   └── version.go                 # Version info (injected via ldflags)
├── internal/                      # Private business logic
│   ├── config/                    # Shared types (LogEntry, LogLevel, Config)
│   ├── parser/                    # Multi-format log parser
│   ├── analyzer/                  # Stats, filtering, pattern detection
│   └── output/                    # Formatted output (text, JSON, table)
├── go.mod
├── Makefile
├── DESIGN.md                      # Architecture and LLM integration design
└── ROADMAP.md                     # Phased roadmap from CLI to platform
```

## Development

```sh
make help        # Show all available targets
make build       # Compile to bin/cyro
make run         # Build and run
make test        # Run tests with race detection
make fmt         # Format code
make vet         # Run go vet
make lint        # Run golangci-lint
make check       # Format + vet + test
make tidy        # Tidy and verify modules
make clean       # Remove build artifacts
```

## Roadmap

See [ROADMAP.md](docs/ROADMAP.md) for the full phased plan. The high-level progression:

| Phase               | What Ships                                        |
|---------------------|---------------------------------------------------|
| 1 — Local CLI       | Functional log tool: search, stats, analyze, tail |
| 2 — LLM Integration | AI-powered analysis via Ollama                    |
| 3 — RAG Pipeline    | Natural language querying over large log files    |
| 4 — Multi-Provider  | OpenAI/Anthropic support, anomaly detection, v1.0 |
| 5 — Team Scale      | Shared server mode, networked vector store        |
| 6 — Org Scale       | Async ingestion, log source connectors            |
| 7 — Platform        | Multi-tenant, RBAC, web UI                        |

## License

MIT
