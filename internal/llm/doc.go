// Package llm provides a unified interface for interacting with Large Language Models.
//
// # Overview
//
// This package defines a Provider interface that abstracts different LLM providers
// (Ollama, OpenAI, Anthropic) behind a common API. This allows the rest of the
// application to use LLMs without being coupled to a specific provider.
//
// # Architecture
//
// The package uses a factory pattern with provider-specific implementations in
// subpackages. To avoid import cycles, subpackages (like ollama) define their own
// types that match the Provider interface, and the parent package uses adapter
// types to bridge between them.
//
//	┌──────────────┐
//	│ llm package  │  ← Defines Provider interface
//	│              │  ← Factory: NewProvider()
//	│              │  ← Adapters for each provider
//	└──────┬───────┘
//	       │
//	       ├─────────────┐
//	       │             │
//	┌──────▼──────┐  ┌──▼───────────┐
//	│ llm/ollama  │  │ llm/openai   │ (future)
//	│ package     │  │ package      │
//	└─────────────┘  └──────────────┘
//
// # Usage
//
// Create a provider using the factory function with your configuration:
//
//	import (
//	    "log/slog"
//	    "github.com/bimmerbailey/cyro/internal/config"
//	    "github.com/bimmerbailey/cyro/internal/llm"
//	)
//
//	cfg := &config.Config{
//	    LLM: config.LLMConfig{
//	        Provider: "ollama",
//	        Ollama: config.OllamaConfig{
//	            Host:  "http://localhost:11434",
//	            Model: "llama3.2",
//	        },
//	    },
//	}
//
//	logger := slog.Default()
//	provider, err := llm.NewProvider(cfg, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check health
//	if err := provider.Heartbeat(ctx); err != nil {
//	    log.Fatal("LLM provider not available:", err)
//	}
//
//	// Check if model is available
//	available, err := provider.ModelAvailable(ctx, "llama3.2")
//	if err != nil || !available {
//	    log.Fatal("Model llama3.2 not found. Run: ollama pull llama3.2")
//	}
//
// # Synchronous Chat
//
// Send messages and receive a complete response:
//
//	messages := []llm.Message{
//	    {Role: "system", Content: "You are a log analysis expert."},
//	    {Role: "user", Content: "Analyze these logs..."},
//	}
//
//	response, err := provider.Chat(ctx, messages, &llm.ChatOptions{
//	    Model:       "llama3.2",
//	    Temperature: 0,  // Deterministic for log analysis
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(response.Content)
//	fmt.Printf("Tokens: %d prompt + %d total\n",
//	    response.TokensPrompt,
//	    response.TokensTotal)
//
// # Streaming Chat
//
// Stream tokens as they're generated (better UX for slow local models):
//
//	stream, err := provider.ChatStream(ctx, messages, &llm.ChatOptions{
//	    Model:       "llama3.2",
//	    Temperature: 0,
//	    MaxTokens:   8000,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for event := range stream {
//	    if event.Error != nil {
//	        log.Fatal(event.Error)
//	    }
//	    fmt.Print(event.Content)  // Print tokens as they arrive
//	    if event.Done {
//	        fmt.Println("\n[Stream complete]")
//	    }
//	}
//
// # Error Handling
//
// The package defines custom error types for common failure modes:
//
//   - ErrProviderUnavailable: LLM service is not reachable
//   - ErrModelNotFound: Requested model is not available
//   - ErrInvalidResponse: Provider returned malformed data
//   - ErrStreamClosed: Stream ended unexpectedly
//   - ErrContextCanceled: Operation was canceled via context
//
// Use errors.Is() to check for specific error types:
//
//	if errors.Is(err, llm.ErrProviderUnavailable) {
//	    // Handle connection failure
//	}
//
// # Configuration
//
// Configuration is loaded from ~/.cyro.yaml or environment variables:
//
//	llm:
//	  provider: ollama
//	  temperature: 0
//	  token_budget: 8000
//	  ollama:
//	    host: http://localhost:11434
//	    model: llama3.2
//	    embedding_model: nomic-embed-text
//
// Environment variables (prefix: CYRO_):
//
//   - CYRO_PROVIDER=ollama
//   - CYRO_OLLAMA_HOST=http://localhost:11434
//   - CYRO_OLLAMA_MODEL=llama3.2
//
// # Adding New Providers
//
// To add support for a new LLM provider:
//
// 1. Create a new subpackage (e.g., internal/llm/openai)
//
// 2. Define types matching the Provider interface:
//
//	type Provider struct { ... }
//	type Message struct { Role, Content string }
//	type ChatOptions struct { Model, Temperature, MaxTokens }
//	type Response struct { Content, Model, TokensPrompt, TokensTotal }
//	type StreamEvent struct { Content, Done, Error }
//
// 3. Implement all Provider methods:
//
//	func (p *Provider) Chat(ctx, messages, opts) (*Response, error)
//	func (p *Provider) ChatStream(ctx, messages, opts) (<-chan StreamEvent, error)
//	func (p *Provider) Heartbeat(ctx) error
//	func (p *Provider) ModelAvailable(ctx, model) (bool, error)
//
// 4. Add a case to the NewProvider() factory in llm.go:
//
//	case "openai":
//	    openaiProvider, err := openai.New(...)
//	    return &openaiProviderAdapter{provider: openaiProvider}, nil
//
// 5. Create an adapter type that converts between llm types and provider types
//
// 6. Add configuration structs to internal/config/config.go
//
// 7. Add default values to cmd/root.go
//
// # Thread Safety
//
// All Provider implementations must be safe for concurrent use.
// Multiple goroutines may call methods on the same provider simultaneously.
package llm
