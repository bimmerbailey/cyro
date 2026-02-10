// Package llm provides an abstraction layer for Large Language Model interactions.
//
// The package defines a Provider interface that enables swapping between different
// LLM providers (Ollama, OpenAI, Anthropic) without changing consuming code.
//
// Example usage:
//
//	provider, err := llm.NewProvider(cfg, logger)
//	if err != nil {
//	    return err
//	}
//
//	messages := []llm.Message{
//	    {Role: "system", Content: "You are a log analysis expert."},
//	    {Role: "user", Content: "Analyze these logs..."},
//	}
//
//	// Streaming response
//	stream, err := provider.ChatStream(ctx, messages, &llm.ChatOptions{
//	    Model:       "llama3.2",
//	    Temperature: 0,
//	})
//	for event := range stream {
//	    if event.Error != nil {
//	        return event.Error
//	    }
//	    fmt.Print(event.Content)
//	}
package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/llm/ollama"
)

// Provider defines the interface for LLM interactions.
// Implementations must be safe for concurrent use.
type Provider interface {
	// Chat sends messages and returns a complete response.
	// The context can be used to cancel the request.
	Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*Response, error)

	// ChatStream sends messages and returns a channel of streaming events.
	// The channel will be closed when the stream completes or encounters an error.
	// The context can be used to cancel the stream.
	ChatStream(ctx context.Context, messages []Message, opts *ChatOptions) (<-chan StreamEvent, error)

	// Heartbeat checks if the provider is reachable and healthy.
	// Returns nil if the provider is available, otherwise returns an error.
	Heartbeat(ctx context.Context) error

	// ModelAvailable checks if a specific model is available for use.
	// Returns true if the model is ready, false if it needs to be pulled/downloaded.
	ModelAvailable(ctx context.Context, model string) (bool, error)
}

// Message represents a single message in a conversation.
type Message struct {
	// Role identifies the message sender: "system", "user", or "assistant"
	Role string

	// Content is the message text
	Content string
}

// ChatOptions configures chat behavior.
// All fields are optional; nil opts uses provider defaults.
type ChatOptions struct {
	// Model specifies which model to use (e.g., "llama3.2", "gpt-4")
	Model string

	// Temperature controls randomness (0.0 = deterministic, 2.0 = very random)
	// For log analysis, 0 is recommended for consistent output
	Temperature float32

	// MaxTokens limits the response length (0 = unlimited/provider default)
	MaxTokens int
}

// Response represents a complete LLM response.
type Response struct {
	// Content is the generated text
	Content string

	// Model is the name of the model that generated the response
	Model string

	// TokensPrompt is the number of tokens in the prompt
	TokensPrompt int

	// TokensTotal is the total number of tokens (prompt + completion)
	TokensTotal int
}

// StreamEvent represents a single event in a streaming response.
type StreamEvent struct {
	// Content is the incremental text chunk (token or group of tokens)
	Content string

	// Done indicates if this is the final event in the stream
	Done bool

	// Error contains any error that occurred during streaming
	// When Error is non-nil, the stream should be considered terminated
	Error error
}

// Common errors returned by LLM providers.
var (
	// ErrProviderUnavailable indicates the LLM provider is not reachable
	ErrProviderUnavailable = errors.New("llm provider is not reachable")

	// ErrModelNotFound indicates the requested model is not available
	ErrModelNotFound = errors.New("requested model is not available")

	// ErrInvalidResponse indicates the provider returned an invalid response
	ErrInvalidResponse = errors.New("provider returned invalid response")

	// ErrStreamClosed indicates the stream was closed unexpectedly
	ErrStreamClosed = errors.New("stream was closed unexpectedly")

	// ErrContextCanceled indicates the operation was canceled via context
	ErrContextCanceled = errors.New("operation was canceled")
)

// NewProvider creates an LLM provider based on the configuration.
// The logger is used for debug and error messages.
// Returns an error if the provider type is unknown or initialization fails.
func NewProvider(cfg *config.Config, logger *slog.Logger) (Provider, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	providerType := strings.ToLower(cfg.LLM.Provider)
	logger.Debug("creating llm provider", "type", providerType)

	switch providerType {
	case "ollama":
		ollamaProvider, err := ollama.New(ollama.Config{
			Host:           cfg.LLM.Ollama.Host,
			Model:          cfg.LLM.Ollama.Model,
			EmbeddingModel: cfg.LLM.Ollama.EmbeddingModel,
		}, logger)
		if err != nil {
			return nil, err
		}
		return &ollamaProviderAdapter{provider: ollamaProvider}, nil

	case "":
		return nil, errors.New("llm provider not specified in configuration")

	default:
		return nil, fmt.Errorf("unknown llm provider: %s (supported: ollama)", providerType)
	}
}

// ollamaProviderAdapter adapts the ollama.Provider to the llm.Provider interface.
// This is needed to avoid import cycles between llm and ollama packages.
type ollamaProviderAdapter struct {
	provider *ollama.Provider
}

func (a *ollamaProviderAdapter) Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*Response, error) {
	// Convert llm.Message to ollama.Message
	ollamaMessages := make([]ollama.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Convert options
	var ollamaOpts *ollama.ChatOptions
	if opts != nil {
		ollamaOpts = &ollama.ChatOptions{
			Model:       opts.Model,
			Temperature: opts.Temperature,
			MaxTokens:   opts.MaxTokens,
		}
	}

	// Call ollama provider
	resp, err := a.provider.Chat(ctx, ollamaMessages, ollamaOpts)
	if err != nil {
		return nil, err
	}

	// Convert response
	return &Response{
		Content:      resp.Content,
		Model:        resp.Model,
		TokensPrompt: resp.TokensPrompt,
		TokensTotal:  resp.TokensTotal,
	}, nil
}

func (a *ollamaProviderAdapter) ChatStream(ctx context.Context, messages []Message, opts *ChatOptions) (<-chan StreamEvent, error) {
	// Convert llm.Message to ollama.Message
	ollamaMessages := make([]ollama.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Convert options
	var ollamaOpts *ollama.ChatOptions
	if opts != nil {
		ollamaOpts = &ollama.ChatOptions{
			Model:       opts.Model,
			Temperature: opts.Temperature,
			MaxTokens:   opts.MaxTokens,
		}
	}

	// Get ollama stream
	ollamaStream, err := a.provider.ChatStream(ctx, ollamaMessages, ollamaOpts)
	if err != nil {
		return nil, err
	}

	// Convert stream events
	eventChan := make(chan StreamEvent, 10)
	go func() {
		defer close(eventChan)
		for ollamaEvent := range ollamaStream {
			eventChan <- StreamEvent{
				Content: ollamaEvent.Content,
				Done:    ollamaEvent.Done,
				Error:   ollamaEvent.Error,
			}
		}
	}()

	return eventChan, nil
}

func (a *ollamaProviderAdapter) Heartbeat(ctx context.Context) error {
	return a.provider.Heartbeat(ctx)
}

func (a *ollamaProviderAdapter) ModelAvailable(ctx context.Context, model string) (bool, error) {
	return a.provider.ModelAvailable(ctx, model)
}
