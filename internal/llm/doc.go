// Package llm provides a unified interface for interacting with multiple LLM providers.
//
// Cyro uses langchaingo (github.com/tmc/langchaingo) to support Ollama, OpenAI,
// and Anthropic through a single Provider interface. The package handles provider
// selection, API key resolution, and error normalization.
//
// # Architecture
//
//	┌──────────┐
//	│ Provider │  Interface
//	│Interface │  (Chat, ChatStream, Heartbeat, ModelAvailable)
//	└────┬─────┘
//	     │
//	     ├──────────────┬──────────────┬───────────────┐
//	     │              │              │               │
//	┌────▼────┐   ┌─────▼─────┐  ┌────▼──────┐  ┌─────▼──────┐
//	│ Ollama  │   │  OpenAI   │  │ Anthropic │  │  (future)  │
//	│Provider │   │ Provider  │  │ Provider  │  │            │
//	└────┬────┘   └─────┬─────┘  └────┬──────┘  └────────────┘
//	     │              │              │
//	     └──────────────┴──────────────┘
//	                    │
//	           ┌────────▼─────────┐
//	           │  langchaingo     │
//	           │  llms.Model      │
//	           └──────────────────┘
//
// # Configuration
//
// Provider selection is controlled by the llm.provider config value:
//
//	llm:
//	  provider: ollama  # or "openai", "anthropic"
//	  temperature: 0.0
//
//	  ollama:
//	    host: http://localhost:11434
//	    model: llama3.2
//
//	  openai:
//	    model: gpt-4o
//	    # api_key: read from OPENAI_API_KEY env var
//
//	  anthropic:
//	    model: claude-3-7-sonnet-20250219
//	    # api_key: read from ANTHROPIC_API_KEY env var
//
// # API Key Resolution
//
// API keys are resolved in this order:
//  1. Config file (llm.openai.api_key or llm.anthropic.api_key)
//  2. Native environment variable (OPENAI_API_KEY or ANTHROPIC_API_KEY)
//
// If a provider requires an API key and neither source provides one, NewProvider
// returns an error.
//
// # Example Usage
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
// # Switching Providers
//
// Switch between providers using environment variables:
//
//	# Use Ollama (default)
//	cyro chat /var/log/app.log
//
//	# Use OpenAI
//	CYRO_LLM_PROVIDER=openai OPENAI_API_KEY=sk-... cyro chat /var/log/app.log
//
//	# Use Anthropic
//	CYRO_LLM_PROVIDER=anthropic ANTHROPIC_API_KEY=sk-ant-... cyro chat /var/log/app.log
//
// # Thread Safety
//
// All Provider implementations must be safe for concurrent use.
// Multiple goroutines may call methods on the same provider simultaneously.
package llm
