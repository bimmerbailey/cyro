package llm

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/llm/ollama"
)

// TestOllamaImplementsProvider verifies that ollama.Provider implements the Provider interface.
func TestOllamaImplementsProvider(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	ollamaProvider, err := ollama.New(ollama.Config{
		Host:  "http://localhost:11434",
		Model: "llama3.2",
	}, logger)
	if err != nil {
		t.Fatalf("Failed to create ollama provider: %v", err)
	}

	// Wrap in adapter
	adapter := &ollamaProviderAdapter{provider: ollamaProvider}

	// This will fail at compile time if the interface is not satisfied
	var _ Provider = adapter
}

// TestNewProviderOllama verifies the factory creates an Ollama provider.
func TestNewProviderOllama(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Ollama: config.OllamaConfig{
				Host:           "http://localhost:11434",
				Model:          "llama3.2",
				EmbeddingModel: "nomic-embed-text",
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := NewProvider(cfg, logger)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	// Verify provider has all required methods (compile-time check)
	var _ Provider = provider
}

// TestNewProviderNilConfig verifies that nil config is rejected.
func TestNewProviderNilConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	_, err := NewProvider(nil, logger)
	if err == nil {
		t.Error("NewProvider() should reject nil config")
	}
}

// TestNewProviderNilLogger verifies that nil logger is rejected.
func TestNewProviderNilLogger(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Ollama: config.OllamaConfig{
				Host:  "http://localhost:11434",
				Model: "llama3.2",
			},
		},
	}

	_, err := NewProvider(cfg, nil)
	if err == nil {
		t.Error("NewProvider() should reject nil logger")
	}
}

// TestNewProviderUnknown verifies that unknown provider is rejected.
func TestNewProviderUnknown(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "unknown-provider",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	_, err := NewProvider(cfg, logger)
	if err == nil {
		t.Error("NewProvider() should reject unknown provider")
	}
}

// TestNewProviderEmptyProvider verifies that empty provider string is rejected.
func TestNewProviderEmptyProvider(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	_, err := NewProvider(cfg, logger)
	if err == nil {
		t.Error("NewProvider() should reject empty provider")
	}
}

// TestMessageTypes verifies Message type structure.
func TestMessageTypes(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello",
	}

	if msg.Role != "user" {
		t.Errorf("Message.Role = %q, want %q", msg.Role, "user")
	}
	if msg.Content != "Hello" {
		t.Errorf("Message.Content = %q, want %q", msg.Content, "Hello")
	}
}

// TestChatOptions verifies ChatOptions type structure.
func TestChatOptions(t *testing.T) {
	opts := &ChatOptions{
		Model:       "llama3.2",
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	if opts.Model != "llama3.2" {
		t.Errorf("ChatOptions.Model = %q, want %q", opts.Model, "llama3.2")
	}
	if opts.Temperature != 0.7 {
		t.Errorf("ChatOptions.Temperature = %f, want %f", opts.Temperature, 0.7)
	}
	if opts.MaxTokens != 1000 {
		t.Errorf("ChatOptions.MaxTokens = %d, want %d", opts.MaxTokens, 1000)
	}
}

// TestResponse verifies Response type structure.
func TestResponse(t *testing.T) {
	resp := &Response{
		Content:      "Test response",
		Model:        "llama3.2",
		TokensPrompt: 10,
		TokensTotal:  30,
	}

	if resp.Content != "Test response" {
		t.Errorf("Response.Content = %q, want %q", resp.Content, "Test response")
	}
	if resp.Model != "llama3.2" {
		t.Errorf("Response.Model = %q, want %q", resp.Model, "llama3.2")
	}
	if resp.TokensPrompt != 10 {
		t.Errorf("Response.TokensPrompt = %d, want %d", resp.TokensPrompt, 10)
	}
	if resp.TokensTotal != 30 {
		t.Errorf("Response.TokensTotal = %d, want %d", resp.TokensTotal, 30)
	}
}

// TestStreamEvent verifies StreamEvent type structure.
func TestStreamEvent(t *testing.T) {
	event := StreamEvent{
		Content: "chunk",
		Done:    false,
		Error:   nil,
	}

	if event.Content != "chunk" {
		t.Errorf("StreamEvent.Content = %q, want %q", event.Content, "chunk")
	}
	if event.Done {
		t.Error("StreamEvent.Done should be false")
	}
	if event.Error != nil {
		t.Errorf("StreamEvent.Error should be nil, got %v", event.Error)
	}
}

// TestErrorTypes verifies custom error types.
func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrProviderUnavailable", ErrProviderUnavailable},
		{"ErrModelNotFound", ErrModelNotFound},
		{"ErrInvalidResponse", ErrInvalidResponse},
		{"ErrStreamClosed", ErrStreamClosed},
		{"ErrContextCanceled", ErrContextCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s should have an error message", tt.name)
			}
		})
	}
}

// TestProviderInterfaceMethods verifies all Provider interface methods are callable.
func TestProviderInterfaceMethods(t *testing.T) {
	// This test verifies that all methods exist at compile time
	// Runtime behavior is tested in ollama_test.go

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Ollama: config.OllamaConfig{
				Host:  "http://localhost:11434",
				Model: "llama3.2",
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := NewProvider(cfg, logger)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}

	ctx := context.Background()

	// Verify all methods exist (will fail at compile time if missing)
	_ = provider.Heartbeat
	_ = provider.ModelAvailable
	_ = provider.Chat
	_ = provider.ChatStream

	// Quick smoke test (these will fail if Ollama isn't running, which is fine)
	_ = provider.Heartbeat(ctx)
	_, _ = provider.ModelAvailable(ctx, "llama3.2")
	_, _ = provider.Chat(ctx, []Message{{Role: "user", Content: "test"}}, nil)
	_, _ = provider.ChatStream(ctx, []Message{{Role: "user", Content: "test"}}, nil)
}
