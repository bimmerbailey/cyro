# Cyro - CLI Log Analysis Tool

CLI tool for log file analysis powered by local LLMs. Written in Go, using Cobra for CLI and Viper for configuration.

See @README.md for project overview and @docs/DESIGN.md for architecture details.

## Quick Reference

| Command           | Purpose                             |
|-------------------|-------------------------------------|
| `make build`      | Compile binary to `bin/cyro`        |
| `make test`       | Run all tests with race detection   |
| `make test-cover` | Run tests with HTML coverage report |
| `make fmt`        | Format all Go files (`gofmt -s`)    |
| `make vet`        | Run `go vet` static analysis        |
| `make lint`       | Run `golangci-lint`                 |
| `make check`      | Run fmt + vet + test in sequence    |
| `make tidy`       | Tidy and verify Go modules          |
| `make clean`      | Remove build artifacts              |

## Project Structure

- `main.go` -- Entrypoint, calls `cmd.Execute()`
- `cmd/` -- Cobra command definitions (root, search, analyze, stats, tail, version)
- `internal/config/` -- Shared domain types (Config, LogEntry, LogLevel)
- `internal/parser/` -- Multi-format log parser (JSON, syslog, Apache, generic)
- `internal/analyzer/` -- Stats computation, filtering, pattern detection
- `internal/output/` -- Formatted output (text, JSON, table)
- `docs/` -- DESIGN.md (architecture) and ROADMAP.md (phases)
- `tmp/` -- Sample log files for manual testing

## Code Conventions

- **Packages:** Lowercase, single-word names (`config`, `parser`, `analyzer`, `output`)
- **Exported types:** PascalCase (`LogEntry`, `FilterOptions`, `Stats`)
- **Unexported functions:** camelCase (`parseLine`, `tryParseJSON`)
- **Constants:** PascalCase with prefix grouping (`LevelDebug`, `LevelInfo`, `FormatJSON`)
- **Constructors:** Use `New()` pattern (`parser.New()`, `analyzer.New()`, `output.New()`)
- **Errors:** Return errors, never panic. Use `RunE` for Cobra commands.
- **Documentation:** Every package must have a doc comment (`// Package config provides...`)
- **Internal packages:** All business logic goes in `internal/` (unexportable by convention)
- **One concern per file:** Each file handles a single responsibility

## Architecture

- Single-binary CLI using Cobra + Viper
- Configuration priority: CLI flags > env vars (`CYRO_` prefix) > config file (`~/.cyro.yaml`) > defaults
- Pipeline: CLI Interface -> Parse -> Pre-Process -> (future: LLM Layer) -> Output
- Flags are bound to Viper via `viper.BindPFlag()` for unified config
- Version info injected at build time via ldflags

## Development Status

- Active branch: `rewrite-log_analysis` (Go rewrite from original Python codebase)
- Phase 1: Local CLI foundation -- commands scaffolded, `RunE` functions contain TODO stubs
- Internal packages (`parser`, `analyzer`, `output`) have working implementations
- No tests exist yet -- test files need to be created
- No CI/CD pipeline yet
- No `.golangci.yml` config yet (referenced by Makefile but not created)

## Dependencies

Only two direct dependencies -- keep it minimal:

- `github.com/spf13/cobra` -- CLI framework
- `github.com/spf13/viper` -- Configuration management

## Testing Guidelines

- Use standard Go testing (`go test`)
- Tests run with `-race` and `-count=1` (no caching)
- Place test files alongside source: `internal/parser/parser_test.go`
- Use table-driven tests where applicable
- Run `make test` before committing
