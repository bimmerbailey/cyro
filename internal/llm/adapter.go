package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// langchainAdapter implements the Provider interface using langchaingo.
// This adapter translates between our Provider interface and langchaingo's llms.Model.
type langchainAdapter struct {
	model        llms.Model
	defaultModel string
	providerType string
	logger       *slog.Logger
}

// Chat sends messages and returns a complete response.
func (a *langchainAdapter) Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*Response, error) {
	lcMessages := convertMessages(messages)
	lcOpts := convertOptions(opts, a.defaultModel)

	resp, err := a.model.GenerateContent(ctx, lcMessages, lcOpts...)
	if err != nil {
		return nil, wrapError(err)
	}

	return convertResponse(resp, a.defaultModel), nil
}

// ChatStream sends messages and returns a channel of streaming events.
func (a *langchainAdapter) ChatStream(ctx context.Context, messages []Message, opts *ChatOptions) (<-chan StreamEvent, error) {
	lcMessages := convertMessages(messages)
	lcOpts := convertOptions(opts, a.defaultModel)

	eventChan := make(chan StreamEvent, 10)

	go func() {
		defer close(eventChan)

		// Add streaming callback to options
		streamOpts := append(lcOpts, llms.WithStreamingFunc(
			func(ctx context.Context, chunk []byte) error {
				select {
				case eventChan <- StreamEvent{
					Content: string(chunk),
					Done:    false,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			},
		))

		_, err := a.model.GenerateContent(ctx, lcMessages, streamOpts...)

		if err != nil {
			eventChan <- StreamEvent{Error: wrapError(err), Done: true}
		} else {
			eventChan <- StreamEvent{Done: true}
		}
	}()

	return eventChan, nil
}

// Heartbeat checks if the provider is reachable (cloud providers only).
// Ollama provider overrides this with HTTP health check.
func (a *langchainAdapter) Heartbeat(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Simple ping with minimal token usage
	_, err := a.Chat(ctx, []Message{
		{Role: "user", Content: "ping"},
	}, &ChatOptions{
		MaxTokens: 1,
	})

	return err
}

// ModelAvailable checks if model is available (cloud providers assume yes).
// Ollama provider overrides this with actual model list check.
func (a *langchainAdapter) ModelAvailable(ctx context.Context, model string) (bool, error) {
	// For cloud providers, assume model is available
	// They'll fail at request time with clear error messages
	return true, nil
}

// --- Conversion Helpers ---

func convertMessages(messages []Message) []llms.MessageContent {
	result := make([]llms.MessageContent, len(messages))
	for i, msg := range messages {
		result[i] = llms.TextParts(convertRole(msg.Role), msg.Content)
	}
	return result
}

func convertRole(role string) llms.ChatMessageType {
	switch role {
	case "system":
		return llms.ChatMessageTypeSystem
	case "user":
		return llms.ChatMessageTypeHuman
	case "assistant":
		return llms.ChatMessageTypeAI
	default:
		return llms.ChatMessageTypeGeneric
	}
}

func convertOptions(opts *ChatOptions, defaultModel string) []llms.CallOption {
	result := []llms.CallOption{}

	// Model selection
	if opts != nil && opts.Model != "" {
		result = append(result, llms.WithModel(opts.Model))
	} else {
		result = append(result, llms.WithModel(defaultModel))
	}

	// Temperature
	if opts != nil {
		result = append(result, llms.WithTemperature(float64(opts.Temperature)))
	}

	// Max tokens
	if opts != nil && opts.MaxTokens > 0 {
		result = append(result, llms.WithMaxTokens(opts.MaxTokens))
	}

	return result
}

func convertResponse(lcResp *llms.ContentResponse, defaultModel string) *Response {
	if len(lcResp.Choices) == 0 {
		return &Response{Model: defaultModel}
	}

	choice := lcResp.Choices[0]

	return &Response{
		Content:      choice.Content,
		Model:        getStringFromInfo(choice.GenerationInfo, "Model", defaultModel),
		TokensPrompt: getIntFromInfo(choice.GenerationInfo, "PromptTokens"),
		TokensTotal:  getIntFromInfo(choice.GenerationInfo, "TotalTokens"),
	}
}

func getIntFromInfo(info map[string]any, key string) int {
	if v, ok := info[key].(int); ok {
		return v
	}
	if v, ok := info[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getStringFromInfo(info map[string]any, key string, defaultVal string) string {
	if v, ok := info[key].(string); ok {
		return v
	}
	return defaultVal
}

// wrapError converts langchaingo errors to our error types.
func wrapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case llms.IsRateLimitError(err):
		return fmt.Errorf("%w: rate limit exceeded", ErrProviderUnavailable)
	case llms.IsAuthenticationError(err):
		return fmt.Errorf("authentication failed (check API key): %w", err)
	case llms.IsTokenLimitError(err):
		return fmt.Errorf("%w: context too long", ErrInvalidResponse)
	case llms.IsProviderUnavailableError(err):
		return ErrProviderUnavailable
	case llms.IsCanceledError(err):
		return ErrContextCanceled
	default:
		return err
	}
}
