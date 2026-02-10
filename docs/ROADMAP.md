# Cyro Roadmap

A phased roadmap from local CLI tool to scalable platform. Each phase builds on the previous and delivers a usable
product at every stage.

---

## Phase 1: Local CLI Foundation

**Goal:** A fully functional local log analysis tool that works without any AI.
**Infrastructure:** None. Single binary, local filesystem only.
**Users:** Single developer on their machine.

### 1.1 Core Log Parsing (existing scaffolding in `internal/parser`)

- [x] Finalize multi-format detection: JSON, syslog, Apache/Nginx, generic plaintext
- [x] Robust timestamp extraction across all configured formats
- [x] Level extraction for structured and unstructured logs
- [x] Field extraction for JSON logs (key-value pairs into `LogEntry.Fields`)
- [x] Streaming parser that handles files of any size (line-by-line, never loads full file)

### 1.2 Search Command (`cmd/search.go`)

- [x] Regex pattern matching (`--pattern`)
- [x] Level filtering (`--level`)
- [x] Time range filtering (`--since`, `--until`) with relative time support (e.g., `1h`, `30m`)
- [x] Context lines around matches (`--context`)
- [x] Inverted matching (`--invert`)
- [x] Match count mode (`--count`)
- [x] Multi-file support (glob patterns)

### 1.3 Stats Command (`cmd/stats.go`)

- [x] Line counts by log level
- [x] Time range (first and last entry)
- [x] Error rate calculation
- [x] Top N most frequent messages
- [x] Output in text, JSON, and table formats

### 1.4 Analyze Command (`cmd/analyze.go`)

- [ ] Group-by support (level, message, source)
- [ ] Top N results with frequency counts
- [ ] Pattern-focused analysis (`--pattern`)
- [ ] Time window trend detection (`--window`)

### 1.5 Tail Command (`cmd/tail.go`)

- [ ] Follow mode with file watching (fsnotify or polling)
- [ ] Live filtering by level and pattern
- [ ] Initial N lines display
- [ ] Graceful shutdown on SIGINT

### 1.6 Output Formatting (`internal/output`)

- [ ] Text output (raw log lines, default)
- [ ] JSON output (structured, pipeable)
- [ ] Table output (tabwriter with aligned columns)
- [ ] Color-coded severity when output is a TTY
- [ ] No color when piped (detect `os.Stdout` is not a terminal)

### 1.7 Build & Distribution

- [ ] Version injection via ldflags (`make build`)
- [ ] Shell completion generation (bash, zsh, fish, powershell — Cobra built-in)
- [ ] Cross-compilation targets (Linux amd64/arm64, macOS amd64/arm64)
- [ ] Goreleaser config for release automation

### Milestone: `v0.1.0`

At this point Cyro is a useful `grep`/`jq`-like tool for logs — no AI required. All subsequent phases are additive.

---

## Phase 2: LLM Integration (Local, Ollama-First)

**Goal:** Add AI-powered analysis using local Ollama models.
**Infrastructure:** Ollama running locally (`http://localhost:11434`).
**Users:** Single developer with Ollama installed.

### 2.1 LLM Provider Interface (`internal/llm/`)

- [ ] Define `Provider` interface: `Chat()`, `ChatStream()`, `Embed()`
- [ ] Define shared types: `Message`, `ChatOptions`, `Response`, `StreamEvent`
- [ ] Ollama provider implementation using `github.com/ollama/ollama/api`
- [ ] Streaming support (tokens printed as they arrive)
- [ ] Connection health check (`Heartbeat()`)
- [ ] Model availability check (is the required model pulled?)
- [ ] Configurable endpoint via Viper (`llm.ollama.host`)
- [ ] Configurable model via Viper (`llm.ollama.model`, default `llama3.2`)

### 2.2 Pre-Processing Pipeline (`internal/preprocess/`)

- [ ] Drain algorithm implementation in pure Go (log template extraction)
    - Fixed-depth parse tree
    - Wildcard token replacement for variable fields
    - Template frequency counting
- [ ] Secret redaction with correlation-preserving hashes
    - IP addresses → `[IPV4:a3f2]`
    - Email addresses → `[EMAIL:b7c1]`
    - API keys / tokens (common patterns) → `[SECRET:d4e5]`
    - Same entity always maps to same placeholder
- [ ] Token budget enforcement (default 8K tokens, configurable)
- [ ] Log compression: templates + frequency counts + time range summary

### 2.3 AI-Powered Analyze (`cmd/analyze.go` enhancement)

- [ ] `--ai` flag to enable LLM-powered analysis
- [ ] Pipeline: parse → pre-process (Drain + redaction) → compress → LLM summarization
- [ ] System prompt for log analysis (temperature 0, structured output)
- [ ] Hierarchical map-reduce for files exceeding context window
- [ ] Streaming LLM response to terminal

### 2.4 Ask Command (`cmd/ask.go` — new)

- [ ] Natural language questions about log files: `cyro ask "what caused the errors?" --file app.log`
- [ ] Direct prompting mode (pre-compress logs, send with question)
- [ ] Streaming response output
- [ ] Follow-up questions with conversation context

### 2.5 Prompt Templates (`internal/prompt/`)

- [ ] System prompt: log analyst persona
- [ ] Summarization prompt template
- [ ] Root cause analysis prompt template
- [ ] Anomaly detection prompt template
- [ ] Natural language query prompt template
- [ ] Structured output prompt (two-pass pattern for small models)

### 2.6 Configuration

- [ ] `~/.cyro.yaml` schema for LLM settings:
  ```yaml
  llm:
    provider: ollama          # ollama | openai | anthropic
    ollama:
      host: http://localhost:11434
      model: llama3.2
      embedding_model: nomic-embed-text
    temperature: 0
    token_budget: 8000
  redaction:
    enabled: true
    patterns:
      - ipv4
      - email
      - api_key
  ```
- [ ] Environment variable overrides (`CYRO_LLM_PROVIDER`, etc.)

### Milestone: `v0.2.0`

Cyro can analyze logs with AI. `cyro analyze --ai app.log` gives a natural language summary.
`cyro ask "why did auth fail?" --file app.log` answers questions.

---

## Phase 3: RAG Pipeline (Local Vector Search)

**Goal:** Enable natural language querying over large log files using retrieval-augmented generation.
**Infrastructure:** Ollama (chat + embedding models). Local filesystem for vector persistence.
**Users:** Single developer, handling logs too large for direct LLM context.

### 3.1 Chunking Engine (`internal/chunker/`)

- [ ] Time-window chunking (configurable window size)
- [ ] Error-anchored chunking (error ± N context lines)
- [ ] Sliding window chunking (N lines with M overlap)
- [ ] Hybrid mode: produce chunks from multiple strategies simultaneously
- [ ] Semantic chunk text builder (summary header + deduplicated messages)
- [ ] Minimum/maximum chunk size enforcement

### 3.2 Vector Store Integration (`internal/vectorstore/`)

- [ ] chromem-go integration for persistent local vector storage
- [ ] Ollama embedding via `nomic-embed-text` (default) or configurable model
- [ ] Batch embedding with progress indication
- [ ] Persistence in `~/.cyro/vectors/<file-hash>/`
- [ ] Incremental indexing (detect file changes, only re-index new content)
- [ ] Index management: list indexed files, delete stale indexes
- [ ] Metadata filtering (by file, time range, severity)

### 3.3 RAG Query Pipeline (`internal/rag/`)

- [ ] End-to-end pipeline: embed question → retrieve chunks → build prompt → LLM answer
- [ ] Configurable top-K retrieval (default 5)
- [ ] Time-aware retrieval (boost chunks near mentioned timestamps)
- [ ] Token budget management across retrieved chunks
- [ ] Source attribution in responses (line numbers, timestamps)

### 3.4 Ask Command Enhancement (`cmd/ask.go`)

- [ ] Automatic indexing on first query per log file
- [ ] `--reindex` flag to force re-indexing
- [ ] Progress bar for indexing (parsing, chunking, embedding)
- [ ] Display source chunks used for the answer (`--show-sources`)
- [ ] Multi-file querying: `cyro ask "..." --file app.log --file auth.log`

### 3.5 Index Command (`cmd/index.go` — new)

- [ ] `cyro index /var/log/app.log` — pre-index a file for fast querying
- [ ] `cyro index --list` — show all indexed files with stats
- [ ] `cyro index --clean` — remove stale/orphaned indexes
- [ ] `cyro index --stats <file>` — show chunk count, vector dimensions, disk usage

### Milestone: `v0.3.0`

Cyro handles logs of any size via RAG. Users can ask questions about 10GB+ log files and get answers referencing
specific timestamps and errors.

---

## Phase 4: Multi-Provider & Polish

**Goal:** Support cloud LLM providers, improve UX, harden for daily use.
**Infrastructure:** Ollama (default) + optional cloud API keys.
**Users:** Individual developers and power users.

### 4.1 Additional LLM Providers

- [ ] OpenAI provider (`github.com/openai/openai-go/v3`)
- [ ] Anthropic provider (`github.com/anthropics/anthropic-sdk-go`)
- [ ] Provider selection via config/flag (`--provider openai`)
- [ ] API key management via env vars (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`)
- [ ] Prompt caching support (Anthropic — 16-30% cost savings)

### 4.2 Anomaly Detection Command (`cmd/anomalies.go` — new)

- [ ] `cyro anomalies /var/log/app.log` — detect unusual patterns
- [ ] Baseline comparison mode: `cyro anomalies --baseline good.log --suspect bad.log`
- [ ] Rate-of-change detection (frequency spikes in template patterns)
- [ ] Severity classification (LOW/MEDIUM/HIGH/CRITICAL)
- [ ] JSON output for integration with alerting systems

### 4.3 Diff Command (`cmd/diff.go` — new)

- [ ] `cyro diff --before app-yesterday.log --after app-today.log`
- [ ] Compare log patterns between two time periods or files
- [ ] Identify new patterns, disappeared patterns, frequency changes
- [ ] LLM-powered explanation of what changed

### 4.4 Terminal UX

- [ ] Colored output with severity-appropriate colors
- [ ] Progress bars for long operations (indexing, analysis)
- [ ] Spinner for LLM calls
- [ ] Markdown rendering in terminal (glamour or similar)
- [ ] Interactive mode consideration (Bubbletea TUI)

### 4.5 Testing & Quality

- [ ] Unit tests for parser (all log formats)
- [ ] Unit tests for Drain algorithm
- [ ] Unit tests for chunker (all strategies)
- [ ] Integration tests for RAG pipeline (mock Ollama)
- [ ] Test fixtures: sample log files in various formats
- [ ] CI pipeline (GitHub Actions: lint, test, build)
- [ ] golangci-lint configuration

### 4.6 Documentation

- [ ] Man page generation (Cobra built-in)
- [ ] LLM-ready docs generation (Cobra `GenMarkdownTree`)
- [ ] Example workflows in README
- [ ] Configuration reference

### Milestone: `v1.0.0`

Stable, tested, documented. Works with Ollama, OpenAI, and Anthropic. Handles files of any size. Ready for daily use.

---

## Phase 5: Team Scale

**Goal:** Multiple users share log indexes and LLM infrastructure.
**Infrastructure:** Shared Ollama or cloud LLM, networked vector store, optional API server.
**Users:** 5-50 engineers on a team.

### 5.1 Server Mode (`cmd/serve.go` — new)

- [ ] `cyro serve` — start an HTTP API server exposing the RAG pipeline
- [ ] REST endpoints: `/api/ask`, `/api/index`, `/api/analyze`, `/api/search`
- [ ] Authentication (API key or JWT)
- [ ] Request/response logging
- [ ] Health check endpoint
- [ ] Graceful shutdown

### 5.2 Client Mode (`cmd/` updates)

- [ ] `--server` flag to point CLI at a remote Cyro API server
- [ ] All existing commands work transparently in client mode
- [ ] Fallback to local mode if server is unreachable
- [ ] Config: `server.url` in `~/.cyro.yaml`

### 5.3 Networked Vector Store

- [ ] PostgreSQL + pgvector backend for `internal/vectorstore/`
- [ ] Qdrant backend as alternative
- [ ] Vector store backend selection via config
- [ ] Connection pooling and retry logic
- [ ] Shared index: multiple users query the same indexed logs

### 5.4 Log Source Integrations

- [ ] Read from S3/MinIO (`cyro analyze s3://bucket/path/app.log`)
- [ ] Read from stdin (`kubectl logs pod | cyro analyze --ai -`)
- [ ] Compressed file support (.gz, .zst)
- [ ] Directory/glob ingestion (`cyro index /var/log/app/*.log`)

### 5.5 Shared Ollama / LLM Gateway

- [ ] Documentation for deploying shared Ollama server
- [ ] LiteLLM proxy support for multi-provider routing
- [ ] Rate limiting per user
- [ ] Usage tracking (tokens consumed per query)

### 5.6 Deployment

- [ ] Docker image for `cyro serve`
- [ ] Docker Compose file (Cyro API + PostgreSQL/pgvector + Ollama)
- [ ] Helm chart for Kubernetes deployment
- [ ] Environment-based configuration for containers

### Milestone: `v2.0.0`

A team can deploy Cyro as a shared service. Engineers query logs from their terminals; indexing and LLM calls happen on
shared infrastructure.

---

## Phase 6: Organization Scale

**Goal:** Integrate with existing log infrastructure, handle high-volume ingestion, support hundreds of users.
**Infrastructure:** API server cluster, async ingestion pipeline, LLM gateway, centralized vector store.
**Users:** 50-500 engineers across multiple teams.

### 6.1 Async Ingestion Pipeline

- [ ] Background workers that consume logs from external sources
- [ ] Message queue integration (NATS or Redis Streams) for decoupled ingestion
- [ ] Real-time ingestion from log streams (Loki, Elasticsearch, CloudWatch)
- [ ] Template extraction and embedding on ingest (not on query)
- [ ] Deduplication: skip re-indexing unchanged log segments

### 6.2 Log Source Connectors

- [ ] Elasticsearch / OpenSearch connector
- [ ] Grafana Loki connector
- [ ] AWS CloudWatch Logs connector
- [ ] Kubernetes log collection (via log files or API)
- [ ] Generic webhook receiver for custom log sources

### 6.3 LLM Gateway

- [ ] Central LLM proxy with rate limiting and cost tracking
- [ ] Prompt caching layer (cache system prompts and common template embeddings)
- [ ] Model routing: different models for different task types
- [ ] Fallback chain: local Ollama → cloud provider if local is overloaded
- [ ] Cost allocation per team/project
- [ ] Audit log: what was queried, what context was sent, what was answered

### 6.4 API Server Scaling

- [ ] Stateless API server (horizontal scaling behind load balancer)
- [ ] Connection pooling to vector store and LLM gateway
- [ ] Request queuing for expensive operations (indexing, analysis)
- [ ] Caching layer (Redis) for repeated queries and template embeddings
- [ ] Metrics and observability (Prometheus, OpenTelemetry)

### 6.5 Administration

- [ ] Admin API/CLI for managing indexes, users, usage
- [ ] Index retention policies (auto-delete indexes older than N days)
- [ ] Storage usage monitoring and alerts
- [ ] Backup and restore for vector store

### Milestone: `v3.0.0`

Cyro integrates with existing log infrastructure. Logs are indexed automatically as they arrive. Hundreds of engineers
query without managing infrastructure.

---

## Phase 7: Platform Scale

**Goal:** Multi-tenant SaaS-ready platform with enterprise features.
**Infrastructure:** Full distributed system with isolation, RBAC, and horizontal scaling.
**Users:** 500+ across multiple organizations.

### 7.1 Multi-Tenancy

- [ ] Isolated vector namespaces per organization/team
- [ ] Tenant-scoped API keys and configuration
- [ ] Resource quotas per tenant (storage, queries/day, tokens/day)
- [ ] Data isolation guarantees (no cross-tenant data leakage)

### 7.2 Access Control

- [ ] RBAC: who can query which log sources
- [ ] SSO integration (OIDC, SAML)
- [ ] Audit trail for all queries and data access
- [ ] Secret redaction enforcement (cannot be disabled by users)

### 7.3 Horizontal Scaling

- [ ] Vector store sharding (partition by tenant, time range, or log source)
- [ ] Multiple API server replicas with shared-nothing architecture
- [ ] Worker pool scaling based on ingestion queue depth
- [ ] Auto-scaling policies (Kubernetes HPA)

### 7.4 Observability & Operations

- [ ] Full distributed tracing (OpenTelemetry) across the pipeline
- [ ] Dashboard: query latency, index freshness, LLM usage, error rates
- [ ] Alerting: failed ingestion, LLM errors, storage thresholds
- [ ] SLO tracking: query latency p50/p99, availability

### 7.5 Web Interface

- [ ] Web UI for querying logs (complement to CLI)
- [ ] Shareable query results and analysis reports
- [ ] Log source management
- [ ] Usage dashboards per team

### Milestone: `v4.0.0`

Cyro is a platform. Multiple organizations use it as their AI-powered log analysis layer.

---

## Summary

| Phase | Version | Users  | Infrastructure           | Key Deliverable                       |
|-------|---------|--------|--------------------------|---------------------------------------|
| 1     | v0.1.0  | 1      | None                     | Functional CLI log tool (no AI)       |
| 2     | v0.2.0  | 1      | Ollama local             | AI-powered analysis and Q&A           |
| 3     | v0.3.0  | 1      | Ollama + local vectors   | RAG over large log files              |
| 4     | v1.0.0  | 1      | Ollama + optional cloud  | Multi-provider, polished, tested      |
| 5     | v2.0.0  | 5-50   | Shared server + pgvector | Team-shared service                   |
| 6     | v3.0.0  | 50-500 | Distributed pipeline     | Org-wide with log source integrations |
| 7     | v4.0.0  | 500+   | Full platform            | Multi-tenant SaaS                     |
