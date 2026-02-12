package llm

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/bimmerbailey/cyro/internal/config"
)

func TestNewProvider_AllProviders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name        string
		provider    string
		cfg         config.LLMConfig
		setupEnv    func(t *testing.T)
		expectError bool
		errorMsg    string
	}{
		{
			name:     "ollama - valid config",
			provider: "ollama",
			cfg: config.LLMConfig{
				Provider: "ollama",
				Ollama: config.OllamaConfig{
					Host:  "http://localhost:11434",
					Model: "llama3.2",
				},
			},
			setupEnv: func(t *testing.T) {},
		},
		{
			name:     "openai - with env var",
			provider: "openai",
			cfg: config.LLMConfig{
				Provider: "openai",
				OpenAI: config.OpenAIConfig{
					Model: "gpt-4",
				},
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("OPENAI_API_KEY", "sk-test-key")
			},
		},
		{
			name:     "openai - with config key",
			provider: "openai",
			cfg: config.LLMConfig{
				Provider: "openai",
				OpenAI: config.OpenAIConfig{
					APIKey: "sk-from-config",
					Model:  "gpt-4",
				},
			},
			setupEnv: func(t *testing.T) {},
		},
		{
			name:     "openai - missing api key",
			provider: "openai",
			cfg: config.LLMConfig{
				Provider: "openai",
				OpenAI: config.OpenAIConfig{
					Model: "gpt-4",
				},
			},
			setupEnv: func(t *testing.T) {
				// Explicitly unset the env var to ensure it's not set
				os.Unsetenv("OPENAI_API_KEY")
			},
			expectError: true,
			errorMsg:    "OPENAI_API_KEY",
		},
		{
			name:     "anthropic - with env var",
			provider: "anthropic",
			cfg: config.LLMConfig{
				Provider: "anthropic",
				Anthropic: config.AnthropicConfig{
					Model: "claude-3-7-sonnet-20250219",
				},
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
			},
		},
		{
			name:     "anthropic - missing api key",
			provider: "anthropic",
			cfg: config.LLMConfig{
				Provider: "anthropic",
				Anthropic: config.AnthropicConfig{
					Model: "claude-3-7-sonnet-20250219",
				},
			},
			setupEnv: func(t *testing.T) {
				// Explicitly unset the env var to ensure it's not set
				os.Unsetenv("ANTHROPIC_API_KEY")
			},
			expectError: true,
			errorMsg:    "ANTHROPIC_API_KEY",
		},
		{
			name:     "unknown provider",
			provider: "gemini",
			cfg: config.LLMConfig{
				Provider: "gemini",
			},
			setupEnv:    func(t *testing.T) {},
			expectError: true,
			errorMsg:    "unknown llm provider",
		},
		{
			name:     "empty provider",
			provider: "",
			cfg: config.LLMConfig{
				Provider: "",
			},
			setupEnv:    func(t *testing.T) {},
			expectError: true,
			errorMsg:    "not specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv(t)

			cfg := &config.Config{LLM: tt.cfg}

			provider, err := NewProvider(cfg, logger)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("error should contain %q, got: %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if provider == nil {
				t.Fatal("expected provider but got nil")
			}
		})
	}
}

func TestResolveAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		configKey  string
		envVarName string
		envVarVal  string
		expected   string
	}{
		{
			name:       "config key takes precedence",
			configKey:  "from-config",
			envVarName: "TEST_KEY",
			envVarVal:  "from-env",
			expected:   "from-config",
		},
		{
			name:       "fallback to env var",
			configKey:  "",
			envVarName: "TEST_KEY",
			envVarVal:  "from-env",
			expected:   "from-env",
		},
		{
			name:       "empty when neither set",
			configKey:  "",
			envVarName: "TEST_KEY",
			envVarVal:  "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVarVal != "" {
				t.Setenv(tt.envVarName, tt.envVarVal)
			}

			result := resolveAPIKey(tt.configKey, tt.envVarName)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
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
