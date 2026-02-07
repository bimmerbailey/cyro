# Cyro Design Document

## LLM Integration Strategy

There are three viable approaches for integrating LLMs into Cyro's Go CLI:

### Option A: Direct Provider SDKs

Use each provider's official Go SDK behind a common `Provider` interface in `internal/llm/`.

| Provider  | SDK                                      | Stars | Status              |
|-----------|------------------------------------------|-------|---------------------|
| Ollama    | `github.com/ollama/ollama/api`           | 162K  | Official, Go-native |
| OpenAI    | `github.com/openai/openai-go/v3`         | 2.9K  | Official, v3.18.0   |
| Anthropic | `github.com/anthropics/anthropic-sdk-go` | 767   | Official, v1.21.0   |
| Gemini    | `google.golang.org/genai`                | 1K    | Official, v1.45.0   |

Define a shared interface and implement per-SDK:

```go
type Provider interface {
Chat(ctx context.Context, messages []Message, opts ChatOptions) (*Response, error)
ChatStream(ctx context.Context, messages []Message, opts ChatOptions) (<-chan StreamEvent, error)
}
```

**Pros:** Full control, type-safe, only import what you use, each SDK is production-grade and officially maintained.

**Cons:** More boilerplate — you maintain the abstraction layer yourself.

### Option B: LangChainGo

Use `github.com/tmc/langchaingo` (8.6K stars, 187 contributors, MIT) for a unified `llms.Model` interface across all
providers, plus chains, agents, memory, tool calling, and RAG primitives.

```go
llm, _ := ollama.New(ollama.WithModel("llama3.2"))
// swap to: openai.New() or anthropic.New() — same interface
completion, _ := llms.GenerateFromSinglePrompt(ctx, llm, "analyze this log")
```

**Pros:** Instant multi-provider support, streaming built in, idiomatic Go, rich abstractions for agents/tool-calling.

**Cons:** Pre-v1 (v0.1.14), heavy dependency tree, abstracts away provider-specific features.

### Option C: OpenAI SDK as Universal Client

Since Ollama and many other providers expose OpenAI-compatible endpoints, use a single SDK for everything:

```go
// Ollama via OpenAI-compatible endpoint
client := openai.NewClient(option.WithBaseURL("http://localhost:11434/v1"))
// OpenAI directly
client := openai.NewClient() // uses OPENAI_API_KEY
```

This is how [charmbracelet/mods](https://github.com/charmbracelet/mods) (4.4K stars) works — one SDK, config-driven base
URLs.

**Pros:** Simplest code, single dependency.

**Cons:** Loses Ollama-specific features (model pulling, thinking mode) and Anthropic-specific features (extended
thinking, Bedrock).

### Assessment

Given Cyro's goals (Ollama-first, multi-provider flexibility, subagent system):

- **Option A** is the most robust long-term choice — full access to each provider's capabilities, clean architecture, no
  framework lock-in. More upfront work but pays off as complexity grows.
- **Option B** is the fastest path if you want chains/agents/RAG built in and don't mind the dependency weight.
- **Option C** is pragmatic for shipping fast when provider-specific features aren't needed.

A hybrid of A and C is also viable — use the Ollama native client for full Ollama features (model management, thinking
mode), and the OpenAI SDK for OpenAI/other compatible providers.

### Open Questions

1. Which providers are needed immediately? Just Ollama, or OpenAI/Anthropic from the start?
2. Are agent/tool-calling capabilities needed (LLM deciding to run shell commands, query files), or is this more about
   sending log data to an LLM for analysis?
3. How important is keeping dependencies minimal?

---

## LLM-Powered Log Analysis

### Core Architecture

The dominant pattern in the ecosystem is a **layered pipeline** — never send raw logs directly to an LLM. Pre-process
with traditional tools, compress the output, then feed the distilled context to the LLM for semantic understanding.

```
┌─────────────────────────────────────────────────────────┐
│                    CLI Interface                        │
│  cyro analyze /var/log/app.log                          │
│  cyro ask "what caused the OOM at 3pm?" --file app.log  │
│  cyro tail /var/log/app.log --level error               │
└────────┬────────────────────────────────────────────────┘
         │
    ┌────▼────┐
    │ Ingest  │  ← Parse timestamps, detect format (JSON/syslog/plain)
    └────┬────┘    Time-filter, severity-filter
         │
    ┌────▼────────────┐
    │ Pre-Process     │  ← Drain-style template extraction
    │ (LOCAL, no LLM) │    Frequency counting, deduplication
    │                 │    Secret redaction
    └────┬────────────┘    Token budget enforcement
         │
    ┌────▼─────────┐
    │ LLM Layer    │  ← Summarize, answer questions, detect anomalies
    │ (API call)   │    Structured output (JSON for machine consumption)
    └────┬─────────┘    Supports: Ollama, OpenAI, Anthropic
         │
    ┌────▼────────┐
    │ Output      │  ← Markdown for terminal, JSON for piping
    │ Formatting  │    Color-coded severity in TTY
    └─────────────┘
```

### Key Design Principles

1. **Never send raw logs to LLMs.** Always pre-process with template extraction + frequency counting. This reduces
   cost ~50x and improves output quality.
2. **LLMs are best for interpretation, not detection.** Use traditional tools to find signals, then use LLMs to explain
   what they mean.
3. **Secret redaction is non-negotiable.** Logs contain credentials, PII, IPs. Redact before any LLM call using
   correlation-preserving hashes (same entity -> same placeholder).
4. **Context window allocation matters.** Budget ~50% for surrounding temporal context, not just the error itself.

### Use Cases

#### 1. Log Summarization

Pre-filter by severity/time, extract templates with frequency counts, then have the LLM summarize:

```
Raw Logs (500K lines)
  → Pre-filter (severity, time range, dedup)
  → Template extraction (Drain algorithm)  →  47 unique patterns
  → Frequency counting                     →  "Pattern X occurred 5,234 times"
  → Token budget enforcement               →  8,000 token output
  → LLM receives compressed summary
```

For files exceeding context windows, use hierarchical map-reduce: chunk -> summarize each chunk -> consolidate
summaries.

#### 2. Natural Language Querying (RAG)

Users ask questions in plain English about their logs:

```
User: "why did the server crash at 3pm?"
  → Embed the question via Ollama
  → Vector similarity search over indexed log chunks
  → Retrieve top 5-10 relevant chunks
  → Build prompt with retrieved context + question
  → LLM generates answer referencing specific timestamps and errors
```

#### 3. Anomaly Detection

LLMs excel at **novel/unknown anomalies** that rules haven't been written for. Best as a hybrid:

```
Layer 1: Regex/rules       → Fast, catches known patterns
Layer 2: Template analysis → Catches frequency shifts
Layer 3: LLM              → Catches novel patterns, explains semantic anomalies
```

LLMs can spot things like "this sequence of individually normal-looking logs represents an unusual state transition"
that rule-based systems miss.

#### 4. Root Cause Analysis

Feed the LLM structured error context:

- The error line itself (~10-15% of token budget)
- Preceding logs within a time window (~40-50% of budget)
- Related errors from same/adjacent components (~20-30%)
- System metadata and instructions (~10-15%)

### RAG Architecture for Log Q&A

#### Chunking Strategy

Use hybrid chunking — create overlapping chunks from multiple strategies for best retrieval:

| Strategy           | Best For               | Parameters                                            |
|--------------------|------------------------|-------------------------------------------------------|
| **Time-window**    | Incident investigation | 2-5 min windows (high-volume), 15-30 min (low-volume) |
| **Error-anchored** | Error diagnosis        | Error line ± 20 lines before, 10 after                |
| **Sliding window** | General search         | 30 lines with 10-line overlap                         |

Create chunks from **both** time-windows and error-anchoring. Some log lines appearing in multiple chunks is
intentional — it improves retrieval for different query types.

#### Embedding

Don't embed individual lines (too slow, too little context). Embed chunks of 10-50 lines.

Build a semantic representation for embedding, not raw text:

```go
// Embedding-friendly chunk representation
"Log chunk: 47 entries, 3 errors, 12 warnings
Time range: 2024-01-15T14:30:00Z to 2024-01-15T14:35:00Z
[ERROR] connection refused to upstream:8080 after 30s timeout
[WARN] retry attempt 3/5 for service-auth
..."
```

Recommended Ollama embedding models:

| Model               | Dimensions | Size   | Notes                                   |
|---------------------|------------|--------|-----------------------------------------|
| `nomic-embed-text`  | 768        | ~274MB | Best quality/speed tradeoff, 8K context |
| `all-minilm`        | 384        | ~23MB  | Tiny and fast, 512 token limit          |
| `mxbai-embed-large` | 1024       | ~670MB | Highest quality, slower                 |

#### Vector Store

**Recommended: `chromem-go`** (849 stars, pure Go, zero transitive dependencies, built-in Ollama support, file-based
persistence).

```go
import "github.com/philippgille/chromem-go"

db := chromem.NewPersistentDB("~/.cyro/vectors", false)
embeddingFunc := chromem.NewEmbeddingFuncOllama("nomic-embed-text", "http://localhost:11434/api")
collection, _ := db.GetOrCreateCollection("server-logs", nil, embeddingFunc)

// Index chunks
collection.AddDocuments(ctx, documents, runtime.NumCPU())

// Query
results, _ := collection.Query(ctx, "why did the server crash", 5, nil, nil)
```

Performance: queries 100K documents in ~40ms. For log analysis this is more than adequate.

Upgrade path if SQL integration is needed later: `sqlite-vec` (6.8K stars, CGO-free WASM option available).

#### Query Flow

```
cyro ask "why did the server crash at 3pm?" --file /var/log/app.log

1. Check if log file is already indexed in ~/.cyro/vectors/
   - If not: parse → chunk (hybrid) → embed via Ollama → store in chromem-go
2. Embed the user's question via Ollama /api/embed
3. Vector similarity search → top 5-10 relevant log chunks
4. Build prompt: system prompt + retrieved chunks + user question
5. Send to LLM (Ollama /api/chat) → stream answer to terminal
```

### Pre-Processing: Template Extraction

The Drain algorithm is the standard for log template mining. It extracts patterns like:

```
Input lines:
  "Connected to 10.0.0.1:5432"
  "Connected to 10.0.0.2:5432"
  "Connected to 192.168.1.1:3306"

Extracted template:
  "Connected to <*>:<*>" (occurred 3 times)
```

No pure-Go implementation of Drain exists — this is a gap worth filling in Cyro's `internal/` packages. The algorithm is
straightforward: fixed-depth parse tree that groups log messages by token similarity.

### Prompt Engineering for Log Analysis

Key patterns that work well (including with smaller 7B-13B local models):

**System prompt essentials:**

- Instruct to reference specific timestamps and log lines
- Distinguish observations ("the logs show...") from inferences ("this suggests...")
- Never invent log data (small models hallucinate log content readily)
- Format responses with Summary → Timeline → Root Cause → Recommendations

**Two-pass pattern for structured output (critical for small models):**

1. First LLM call: natural language analysis
2. Second LLM call: convert analysis to structured JSON

This is dramatically more reliable than asking small models to analyze *and* format simultaneously.

**Effective temperature:** 0 for all log analysis tasks. Creativity is not helpful here.

### Practical Considerations

#### Cost

| Approach                  | Daily Cost (1MB log file) | Annual      |
|---------------------------|---------------------------|-------------|
| With template compression | ~$0.015/day               | ~$5.50/year |
| Without compression       | ~$0.75/day                | ~$274/year  |
| Local Ollama              | $0 (hardware cost)        | $0          |

#### Latency

| Operation                          | Latency    |
|------------------------------------|------------|
| Template extraction (10K lines)    | ~100ms     |
| Embedding (100 chunks, local)      | ~500ms     |
| Vector similarity search           | ~5-40ms    |
| LLM call (8K tokens, cloud)        | ~2-5s      |
| LLM call (8K tokens, local Ollama) | ~5-30s     |
| **Total pipeline (local)**         | **~6-31s** |

Acceptable for interactive CLI use. Not suitable for real-time alerting on high-throughput streams.

#### Accuracy

- **Hallucination risk:** LLMs can invent plausible root causes not supported by logs. Prompts must instruct: "only
  reference information present in the provided logs."
- **Missing context:** If the root cause is in a different service's logs, the LLM will give an incomplete answer.
  Multi-file support matters.
- **Small model limitations:** 7B models handle classification and simple summarization well. Root cause analysis across
  complex multi-service incidents benefits from 13B+ or cloud models.

### Recommended Dependency Stack

Minimal viable RAG pipeline for Cyro:

```
github.com/philippgille/chromem-go    # Vector store + Ollama embeddings (0 transitive deps)
github.com/ollama/ollama/api          # Ollama client for chat/generate
```

That's two dependencies for the full pipeline: parse → chunk → embed → store → retrieve → answer.

### Reference Projects

| Project                                                                     | Language | What It Does                                                         |
|-----------------------------------------------------------------------------|----------|----------------------------------------------------------------------|
| [logpai/Drain3](https://github.com/logpai/Drain3)                           | Python   | Log template mining — the foundation for LLM log pipelines           |
| [charmbracelet/mods](https://github.com/charmbracelet/mods)                 | Go       | Multi-provider CLI LLM tool with streaming, conversation persistence |
| [olegiv/logwatch-ai-go](https://github.com/olegiv/logwatch-ai-go)           | Go       | Go-based log analyzer using Claude/Ollama with prompt caching        |
| [Apsion/log-essence](https://github.com/Apsion/log-essence)                 | Python   | Full pipeline: Drain3 + embeddings + secret redaction + CLI          |
| [ashwin-jagadeesha/lograven](https://github.com/ashwin-jagadeesha/lograven) | Python   | RAG-based log Q&A for massive logs, FAISS + TinyLlama                |

---

## Scaling

The local-first design doesn't preclude scaling. Each layer (parsing, embedding, storage, LLM) can be swapped
independently thanks to the interface-based architecture.

### Single User, Large Logs (10GB+ files)

No infrastructure changes needed. The current design handles this with implementation discipline:

- **Streaming parser** — `bufio.Scanner` reads line-by-line without loading the entire file into memory.
- **Batch embedding** — Process and embed chunks in batches rather than holding all in memory.
- **Incremental indexing** — Track file offsets so re-indexing only processes new lines. chromem-go supports adding
  documents incrementally.
- **Template extraction** — The Drain algorithm is O(n) per line with constant memory regardless of file size.

### Team Scale (5-50 users, shared logs)

Multiple users analyzing the same logs shouldn't each re-index independently.

**What changes:**

- **Shared Ollama server** — Point to a team-hosted instance via `OLLAMA_HOST` env var or Viper config. Already
  supported.
- **Shared vector store** — Replace chromem-go's file-based persistence with PostgreSQL + pgvector or Qdrant. The
  interface-based design makes this a backend swap.
- **Shared index cache** — A central service indexes log files once and serves queries to multiple users.

**Infrastructure added:**

- A shared Ollama instance (or cloud API keys)
- PostgreSQL with pgvector, or Qdrant (single container)
- Optional: a thin API service fronting the RAG pipeline

### Organization Scale (50-500 users, centralized log infrastructure)

Cyro becomes a **client** that talks to backend services rather than doing everything locally.

**What changes:**

- **Log ingestion moves server-side** — Logs flow from existing infrastructure (ELK, Loki, Datadog) into a
  pre-processing pipeline. Template extraction and embedding happen on ingest, not on query.
- **The CLI becomes a query client** — `cyro ask "why did auth fail?"` sends the question to an API server that handles
  embedding, retrieval, and LLM calls.
- **Caching becomes critical** — A server-side cache of template embeddings avoids redundant compute.
- **LLM calls go through a gateway** — Rate limiting, cost tracking, prompt caching, audit logging (e.g., LiteLLM or a
  custom proxy).

**Infrastructure added:**

- API server (Go — could be a `cyro serve` subcommand)
- Message queue for async indexing (NATS, Redis Streams)
- PostgreSQL + pgvector or Qdrant cluster
- LLM gateway/proxy for cost control
- Object storage for raw log archives (S3, MinIO)

```
┌────────┐     ┌──────────────┐     ┌──────────────┐     ┌─────────┐
│ cyro   │────▶│ Cyro API     │────▶│ Vector Store │     │ Ollama  │
│ CLI    │     │ Server       │     │ (pgvector)   │     │ / Cloud │
└────────┘     │              │     └──────────────┘     │ LLM     │
               │  - embed     │                          └─────────┘
               │  - retrieve  │                               ▲
               │  - prompt    │───────────────────────────────┘
               └──────┬───────┘
                      │
               ┌──────▼───────┐
               │ Log Sources  │
               │ (Loki, ELK,  │
               │  S3, files)  │
               └──────────────┘
```

### Platform Scale (500+ users, multi-tenant)

At this point Cyro is a product, not a tool:

- **Multi-tenancy** — Isolated vector namespaces per team/org
- **RBAC** — Users can only query logs they have access to
- **Horizontal scaling** — Multiple API server replicas, vector store sharding
- **Async pipelines** — Log ingestion fully decoupled from querying
- **Observability** — Trace every query: what was retrieved, what the LLM saw, what it answered
- **Cost allocation** — Track LLM token usage per team

### Component Scaling Summary

| Component    | Local (now)        | Team                  | Org                  |
|--------------|--------------------|-----------------------|----------------------|
| Parser       | In-process         | In-process            | Server-side pipeline |
| Embeddings   | Ollama local       | Shared Ollama         | Dedicated GPU pool   |
| Vector store | chromem-go (files) | pgvector / Qdrant     | Qdrant cluster       |
| LLM          | Ollama local       | Shared Ollama / cloud | LLM gateway          |
| Interface    | CLI direct         | CLI → API             | CLI → API + Web UI   |
