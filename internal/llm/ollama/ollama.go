// Package ollama provides an Ollama implementation of the llm.Provider interface.
//
// Note: To avoid import cycles, this package defines its own types that match
// the llm.Provider interface. The parent llm package imports this package and
// uses type assertions to ensure compatibility.
package ollama

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
)

// Provider implements the LLM provider interface for Ollama.
type Provider struct {
	client *api.Client
	config Config
	logger *slog.Logger
}

// Config holds Ollama-specific configuration.
type Config struct {
	// Host is the Ollama API endpoint (e.g., "http://localhost:11434")
	Host string

	// Model is the default model to use (e.g., "llama3.2")
	Model string

	// EmbeddingModel is the model to use for embeddings (e.g., "nomic-embed-text")
	// This will be used in Phase 3 for RAG functionality
	EmbeddingModel string
}

// Message represents a single message in a conversation.
type Message struct {
	Role    string
	Content string
}

// ChatOptions configures chat behavior.
type ChatOptions struct {
	Model       string
	Temperature float32
	MaxTokens   int
}

// Response represents a complete LLM response.
type Response struct {
	Content      string
	Model        string
	TokensPrompt int
	TokensTotal  int
}

// StreamEvent represents a single event in a streaming response.
type StreamEvent struct {
	Content string
	Done    bool
	Error   error
}

// Common errors
var (
	ErrProviderUnavailable = errors.New("llm provider is not reachable")
	ErrContextCanceled     = errors.New("operation was canceled")
)

// New creates a new Ollama provider.
// If cfg.Host is empty, it uses the OLLAMA_HOST environment variable or defaults to http://localhost:11434.
func New(cfg Config, logger *slog.Logger) (*Provider, error) {
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	// Start with environment-based client (respects OLLAMA_HOST)
	client, err := api.ClientFromEnvironment()
	if err != nil {
		logger.Error("failed to create ollama client from environment", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}

	// Override with explicit config if provided
	if cfg.Host != "" {
		parsedURL, err := url.Parse(cfg.Host)
		if err != nil {
			logger.Error("invalid ollama host URL", "host", cfg.Host, "error", err)
			return nil, fmt.Errorf("invalid ollama host: %w", err)
		}

		client = api.NewClient(parsedURL, http.DefaultClient)
		logger.Debug("created ollama client with explicit host", "host", cfg.Host)
	} else {
		logger.Debug("created ollama client from environment")
	}

	// Set default model if not specified
	if cfg.Model == "" {
		cfg.Model = "llama3.2"
		logger.Debug("using default model", "model", cfg.Model)
	}

	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "nomic-embed-text"
		logger.Debug("using default embedding model", "model", cfg.EmbeddingModel)
	}

	return &Provider{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// Chat sends messages to Ollama and returns a complete response.
func (p *Provider) Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*Response, error) {
	if len(messages) == 0 {
		return nil, errors.New("messages cannot be empty")
	}

	// Use options or defaults
	model := p.config.Model
	temperature := float32(0)
	if opts != nil {
		if opts.Model != "" {
			model = opts.Model
		}
		temperature = opts.Temperature
	}

	p.logger.Debug("sending chat request", "model", model, "messages", len(messages), "temperature", temperature)

	// Convert messages to Ollama format
	ollamaMessages := make([]api.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Create request
	req := &api.ChatRequest{
		Model:    model,
		Messages: ollamaMessages,
		Options: map[string]interface{}{
			"temperature": temperature,
		},
		Stream: new(bool), // false - we want complete response
	}

	// Send request
	var response api.ChatResponse
	err := p.client.Chat(ctx, req, func(resp api.ChatResponse) error {
		response = resp
		return nil
	})

	if err != nil {
		p.logger.Error("chat request failed", "error", err, "model", model)
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("%w: %v", ErrContextCanceled, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}

	p.logger.Debug("chat request completed",
		"model", response.Model,
		"prompt_tokens", response.PromptEvalCount,
		"total_tokens", response.EvalCount)

	return &Response{
		Content:      response.Message.Content,
		Model:        response.Model,
		TokensPrompt: response.PromptEvalCount,
		TokensTotal:  response.PromptEvalCount + response.EvalCount,
	}, nil
}

// ChatStream sends messages to Ollama and returns a channel of streaming events.
func (p *Provider) ChatStream(ctx context.Context, messages []Message, opts *ChatOptions) (<-chan StreamEvent, error) {
	if len(messages) == 0 {
		return nil, errors.New("messages cannot be empty")
	}

	// Use options or defaults
	model := p.config.Model
	temperature := float32(0)
	maxTokens := 0
	if opts != nil {
		if opts.Model != "" {
			model = opts.Model
		}
		temperature = opts.Temperature
		maxTokens = opts.MaxTokens
	}

	p.logger.Debug("starting chat stream", "model", model, "messages", len(messages), "temperature", temperature)

	// Convert messages to Ollama format
	ollamaMessages := make([]api.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Create request
	req := &api.ChatRequest{
		Model:    model,
		Messages: ollamaMessages,
		Options: map[string]interface{}{
			"temperature": temperature,
		},
		Stream: ptrBool(true), // Enable streaming
	}

	if maxTokens > 0 {
		req.Options["num_predict"] = maxTokens
	}

	// Create channel for streaming events
	eventChan := make(chan StreamEvent, 10)

	// Start streaming in a goroutine
	go func() {
		defer close(eventChan)

		err := p.client.Chat(ctx, req, func(resp api.ChatResponse) error {
			// Check if context was canceled
			select {
			case <-ctx.Done():
				p.logger.Debug("chat stream canceled by context")
				eventChan <- StreamEvent{
					Error: fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err()),
					Done:  true,
				}
				return ctx.Err()
			default:
			}

			// Send content chunk if present
			if resp.Message.Content != "" {
				eventChan <- StreamEvent{
					Content: resp.Message.Content,
					Done:    resp.Done,
				}
			}

			// Log final response
			if resp.Done {
				p.logger.Debug("chat stream completed",
					"model", resp.Model,
					"prompt_tokens", resp.PromptEvalCount,
					"total_tokens", resp.EvalCount)
			}

			return nil
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			p.logger.Error("chat stream failed", "error", err, "model", model)
			eventChan <- StreamEvent{
				Error: fmt.Errorf("%w: %v", ErrProviderUnavailable, err),
				Done:  true,
			}
		}
	}()

	return eventChan, nil
}

// Heartbeat checks if the Ollama service is reachable and healthy.
func (p *Provider) Heartbeat(ctx context.Context) error {
	p.logger.Debug("checking ollama heartbeat")

	err := p.client.Heartbeat(ctx)
	if err != nil {
		p.logger.Error("ollama heartbeat failed", "error", err)
		return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}

	p.logger.Debug("ollama heartbeat successful")
	return nil
}

// ModelAvailable checks if a specific model is available (i.e., has been pulled).
func (p *Provider) ModelAvailable(ctx context.Context, model string) (bool, error) {
	p.logger.Debug("checking model availability", "model", model)

	// List all available models
	listResp, err := p.client.List(ctx)
	if err != nil {
		p.logger.Error("failed to list models", "error", err)
		return false, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}

	// Check if requested model is in the list
	for _, modelInfo := range listResp.Models {
		if modelInfo.Name == model || modelInfo.Model == model {
			p.logger.Debug("model is available", "model", model)
			return true, nil
		}
	}

	p.logger.Debug("model not found", "model", model, "available_count", len(listResp.Models))
	return false, nil
}

// ptrBool returns a pointer to a bool value.
func ptrBool(b bool) *bool {
	return &b
}
