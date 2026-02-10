package ollama

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNew verifies provider creation with various configurations.
func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with host",
			config: Config{
				Host:           "http://localhost:11434",
				Model:          "llama3.2",
				EmbeddingModel: "nomic-embed-text",
			},
			wantErr: false,
		},
		{
			name: "empty config uses defaults",
			config: Config{
				Host: "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name: "invalid host URL",
			config: Config{
				Host: "://invalid-url",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := New(tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("New() returned nil provider without error")
			}
			if !tt.wantErr {
				// Check defaults were applied
				if provider.config.Model == "" {
					t.Error("Model should have default value")
				}
				if provider.config.EmbeddingModel == "" {
					t.Error("EmbeddingModel should have default value")
				}
			}
		})
	}
}

// TestNewNilLogger verifies that nil logger is rejected.
func TestNewNilLogger(t *testing.T) {
	_, err := New(Config{Host: "http://localhost:11434"}, nil)
	if err == nil {
		t.Error("New() should reject nil logger")
	}
}

// TestChat verifies the Chat method with a mock Ollama server.
func TestChat(t *testing.T) {
	// Create a mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/chat" {
			// Parse request to echo back the model
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Send mock response
			response := map[string]interface{}{
				"model":             req["model"],
				"message":           map[string]string{"role": "assistant", "content": "Test response"},
				"done":              true,
				"prompt_eval_count": 10,
				"eval_count":        20,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := New(Config{Host: server.URL, Model: "test-model"}, logger)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	resp, err := provider.Chat(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Chat() failed: %v", err)
	}

	if resp.Content != "Test response" {
		t.Errorf("Chat() content = %q, want %q", resp.Content, "Test response")
	}
	if resp.Model != "test-model" {
		t.Errorf("Chat() model = %q, want %q", resp.Model, "test-model")
	}
	if resp.TokensPrompt != 10 {
		t.Errorf("Chat() TokensPrompt = %d, want 10", resp.TokensPrompt)
	}
	if resp.TokensTotal != 30 {
		t.Errorf("Chat() TokensTotal = %d, want 30", resp.TokensTotal)
	}
}

// TestChatEmptyMessages verifies that Chat rejects empty message list.
func TestChatEmptyMessages(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := New(Config{Host: "http://localhost:11434"}, logger)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.Chat(ctx, []Message{}, nil)
	if err == nil {
		t.Error("Chat() should reject empty messages")
	}
}

// TestChatStream verifies the ChatStream method with a mock server.
func TestChatStream(t *testing.T) {
	// Create a mock Ollama server that streams responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/chat" {
			w.Header().Set("Content-Type", "application/x-ndjson")

			// Send three streaming chunks
			chunks := []map[string]interface{}{
				{"message": map[string]string{"content": "Hello "}, "done": false},
				{"message": map[string]string{"content": "World"}, "done": false},
				{"message": map[string]string{"content": "!"}, "done": true, "prompt_eval_count": 5, "eval_count": 15},
			}

			encoder := json.NewEncoder(w)
			for _, chunk := range chunks {
				if err := encoder.Encode(chunk); err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := New(Config{Host: server.URL, Model: "test-model"}, logger)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	stream, err := provider.ChatStream(ctx, messages, nil)
	if err != nil {
		t.Fatalf("ChatStream() failed: %v", err)
	}

	var content strings.Builder
	var doneCount int
	for event := range stream {
		if event.Error != nil {
			t.Fatalf("Stream error: %v", event.Error)
		}
		content.WriteString(event.Content)
		if event.Done {
			doneCount++
		}
	}

	expectedContent := "Hello World!"
	if content.String() != expectedContent {
		t.Errorf("ChatStream() content = %q, want %q", content.String(), expectedContent)
	}
	if doneCount != 1 {
		t.Errorf("ChatStream() done events = %d, want 1", doneCount)
	}
}

// TestChatStreamCancellation verifies that context cancellation stops the stream.
func TestChatStreamCancellation(t *testing.T) {
	// Create a server that would stream forever
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/chat" {
			w.Header().Set("Content-Type", "application/x-ndjson")
			encoder := json.NewEncoder(w)

			// Send chunks until client disconnects
			for i := 0; i < 100; i++ {
				chunk := map[string]interface{}{
					"message": map[string]string{"content": "chunk"},
					"done":    false,
				}
				if err := encoder.Encode(chunk); err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := New(Config{Host: server.URL}, logger)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cleaned up

	messages := []Message{{Role: "user", Content: "Hello"}}

	stream, err := provider.ChatStream(ctx, messages, nil)
	if err != nil {
		t.Fatalf("ChatStream() failed: %v", err)
	}

	// Cancel after receiving a few chunks
	eventCount := 0
	for event := range stream {
		eventCount++
		if eventCount == 3 {
			cancel()
		}
		if event.Error != nil {
			// Should get a cancellation error
			if !strings.Contains(event.Error.Error(), "canceled") {
				t.Errorf("Expected cancellation error, got: %v", event.Error)
			}
			break
		}
	}

	if eventCount == 0 {
		t.Error("Should have received at least one event")
	}
}

// TestHeartbeat verifies the Heartbeat method.
func TestHeartbeat(t *testing.T) {
	// Create a mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"version": "0.1.0"})
		} else if r.URL.Path == "/" {
			// Ollama's heartbeat endpoint
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ollama is running"))
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := New(Config{Host: server.URL}, logger)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	err = provider.Heartbeat(ctx)
	if err != nil {
		t.Errorf("Heartbeat() should succeed, got error: %v", err)
	}
}

// TestModelAvailable verifies the ModelAvailable method.
func TestModelAvailable(t *testing.T) {
	// Create a mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			response := map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.2:latest", "model": "llama3.2"},
					{"name": "codellama:latest", "model": "codellama"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := New(Config{Host: server.URL}, logger)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		model     string
		available bool
	}{
		{"llama3.2", true},
		{"llama3.2:latest", true},
		{"codellama", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			available, err := provider.ModelAvailable(ctx, tt.model)
			if err != nil {
				t.Fatalf("ModelAvailable() error: %v", err)
			}
			if available != tt.available {
				t.Errorf("ModelAvailable(%q) = %v, want %v", tt.model, available, tt.available)
			}
		})
	}
}

// TestChatWithOptions verifies that ChatOptions are properly applied.
func TestChatWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/chat" {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			// Verify temperature was set
			if options, ok := req["options"].(map[string]interface{}); ok {
				if temp, ok := options["temperature"].(float64); !ok || temp != 0.7 {
					t.Errorf("Temperature not set correctly, got %v", temp)
				}
			}

			response := map[string]interface{}{
				"model":   req["model"],
				"message": map[string]string{"content": "response"},
				"done":    true,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := New(Config{Host: server.URL, Model: "default-model"}, logger)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "test"}}
	opts := &ChatOptions{
		Model:       "custom-model",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := provider.Chat(ctx, messages, opts)
	if err != nil {
		t.Fatalf("Chat() failed: %v", err)
	}

	if resp.Model != "custom-model" {
		t.Errorf("Model override not applied, got %q", resp.Model)
	}
}
