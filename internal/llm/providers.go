package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// resolveAPIKey checks config first, then falls back to environment variable.
// Returns empty string if neither is set.
func resolveAPIKey(configKey, envVarName string) string {
	if configKey != "" {
		return configKey
	}
	return os.Getenv(envVarName)
}

// newOllamaProvider creates an Ollama provider with health check capabilities.
func newOllamaProvider(cfg *config.Config, logger *slog.Logger) (Provider, error) {
	opts := []ollama.Option{
		ollama.WithModel(cfg.LLM.Ollama.Model),
		ollama.WithServerURL(cfg.LLM.Ollama.Host),
	}

	if cfg.LLM.Ollama.KeepAlive != "" {
		opts = append(opts, ollama.WithKeepAlive(cfg.LLM.Ollama.KeepAlive))
	}

	if cfg.LLM.Ollama.NumCtx > 0 {
		opts = append(opts, ollama.WithRunnerNumCtx(cfg.LLM.Ollama.NumCtx))
	}

	if cfg.LLM.Ollama.NumGPU > 0 {
		opts = append(opts, ollama.WithRunnerNumGPU(cfg.LLM.Ollama.NumGPU))
	}

	model, err := ollama.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama provider: %w", err)
	}

	logger.Info("initialized ollama provider",
		"host", cfg.LLM.Ollama.Host,
		"model", cfg.LLM.Ollama.Model,
	)

	adapter := &langchainAdapter{
		model:        model,
		defaultModel: cfg.LLM.Ollama.Model,
		providerType: "ollama",
		logger:       logger,
	}

	return &ollamaProvider{
		langchainAdapter: adapter,
		host:             cfg.LLM.Ollama.Host,
	}, nil
}

// newOpenAIProvider creates an OpenAI provider.
func newOpenAIProvider(cfg *config.Config, logger *slog.Logger) (Provider, error) {
	apiKey := resolveAPIKey(cfg.LLM.OpenAI.APIKey, "OPENAI_API_KEY")

	if apiKey == "" {
		return nil, fmt.Errorf(
			"openai api key not configured: set OPENAI_API_KEY environment variable or llm.openai.api_key in config",
		)
	}

	opts := []openai.Option{
		openai.WithToken(apiKey),
		openai.WithModel(cfg.LLM.OpenAI.Model),
	}

	if cfg.LLM.OpenAI.BaseURL != "" {
		opts = append(opts, openai.WithBaseURL(cfg.LLM.OpenAI.BaseURL))
	}

	if cfg.LLM.OpenAI.OrgID != "" {
		orgID := resolveAPIKey(cfg.LLM.OpenAI.OrgID, "OPENAI_ORG_ID")
		if orgID != "" {
			opts = append(opts, openai.WithOrganization(orgID))
		}
	}

	model, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create openai provider: %w", err)
	}

	logger.Info("initialized openai provider",
		"model", cfg.LLM.OpenAI.Model,
		"base_url", cfg.LLM.OpenAI.BaseURL,
	)

	return &langchainAdapter{
		model:        model,
		defaultModel: cfg.LLM.OpenAI.Model,
		providerType: "openai",
		logger:       logger,
	}, nil
}

// newAnthropicProvider creates an Anthropic/Claude provider.
func newAnthropicProvider(cfg *config.Config, logger *slog.Logger) (Provider, error) {
	apiKey := resolveAPIKey(cfg.LLM.Anthropic.APIKey, "ANTHROPIC_API_KEY")

	if apiKey == "" {
		return nil, fmt.Errorf(
			"anthropic api key not configured: set ANTHROPIC_API_KEY environment variable or llm.anthropic.api_key in config",
		)
	}

	opts := []anthropic.Option{
		anthropic.WithToken(apiKey),
		anthropic.WithModel(cfg.LLM.Anthropic.Model),
	}

	model, err := anthropic.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create anthropic provider: %w", err)
	}

	logger.Info("initialized anthropic provider",
		"model", cfg.LLM.Anthropic.Model,
	)

	return &langchainAdapter{
		model:        model,
		defaultModel: cfg.LLM.Anthropic.Model,
		providerType: "anthropic",
		logger:       logger,
	}, nil
}

// ollamaProvider extends langchainAdapter with Ollama-specific health checks.
type ollamaProvider struct {
	*langchainAdapter
	host string
}

// Heartbeat checks Ollama server health via /api/tags endpoint.
func (p *ollamaProvider) Heartbeat(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", p.host+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrProviderUnavailable, resp.StatusCode)
	}

	return nil
}

// ModelAvailable checks if a specific model has been pulled to Ollama.
func (p *ollamaProvider) ModelAvailable(ctx context.Context, model string) (bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", p.host+"/api/tags", nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	for _, m := range result.Models {
		if m.Name == model || m.Model == model {
			return true, nil
		}
	}

	return false, nil
}
