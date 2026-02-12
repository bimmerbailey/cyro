// Package config provides configuration types and helpers for cyro.
package config

import (
	"encoding/json"
	"strings"
	"time"
)

// Config holds the application-wide configuration.
type Config struct {
	Format           string          `mapstructure:"format"`
	Verbose          bool            `mapstructure:"verbose"`
	TimestampFormats []string        `mapstructure:"timestamp_formats"`
	LogDir           string          `mapstructure:"log_dir"`
	LLM              LLMConfig       `mapstructure:"llm"`
	Redaction        RedactionConfig `mapstructure:"redaction"`
}

// LLMConfig holds configuration for LLM providers.
type LLMConfig struct {
	// Provider selects which LLM to use: "ollama", "openai", "anthropic"
	Provider string `mapstructure:"provider"`

	// Global settings applied to all providers
	Temperature float32 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`

	// Provider-specific configuration
	Ollama    OllamaConfig    `mapstructure:"ollama"`
	OpenAI    OpenAIConfig    `mapstructure:"openai"`
	Anthropic AnthropicConfig `mapstructure:"anthropic"`
}

// OllamaConfig holds Ollama-specific settings.
type OllamaConfig struct {
	Host      string `mapstructure:"host"`       // API endpoint
	Model     string `mapstructure:"model"`      // Default model name
	KeepAlive string `mapstructure:"keep_alive"` // e.g., "5m"
	NumCtx    int    `mapstructure:"num_ctx"`    // Context window size
	NumGPU    int    `mapstructure:"num_gpu"`    // GPU layers to offload
}

// OpenAIConfig holds OpenAI-specific settings.
type OpenAIConfig struct {
	APIKey  string `mapstructure:"api_key"`  // Optional: read from OPENAI_API_KEY if empty
	Model   string `mapstructure:"model"`    // e.g., "gpt-4o", "gpt-4"
	BaseURL string `mapstructure:"base_url"` // Optional: for compatible endpoints
	OrgID   string `mapstructure:"org_id"`   // Optional: organization ID
}

// AnthropicConfig holds Anthropic/Claude-specific settings.
type AnthropicConfig struct {
	APIKey string `mapstructure:"api_key"` // Optional: read from ANTHROPIC_API_KEY if empty
	Model  string `mapstructure:"model"`   // e.g. "claude-3-7-sonnet-20250219"
}

// RedactionConfig holds configuration for secret redaction in preprocessing.
type RedactionConfig struct {
	// Enabled controls whether redaction is active
	Enabled bool `mapstructure:"enabled"`

	// Patterns specifies which redaction patterns to use
	// Available: ipv4, ipv6, email, api_key, aws_key, jwt, private_key, mac_address, credit_card, uuid
	Patterns []string `mapstructure:"patterns"`
}

// LogLevel represents a standard log severity level.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelUnknown
)

// String returns the string representation of a LogLevel.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON implements json.Marshaler for LogLevel.
func (l LogLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// UnmarshalJSON implements json.Unmarshaler for LogLevel.
func (l *LogLevel) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*l = ParseLevel(s)
	return nil
}

// ParseLevel converts a string to a LogLevel.
func ParseLevel(s string) LogLevel {
	switch strings.ToLower(s) {
	case "debug", "dbg":
		return LevelDebug
	case "info", "inf":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error", "err":
		return LevelError
	case "fatal", "critical", "crit":
		return LevelFatal
	default:
		return LevelUnknown
	}
}

// LogEntry represents a single parsed log line.
type LogEntry struct {
	Raw       string                 `json:"raw"`
	Timestamp time.Time              `json:"timestamp,omitempty"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Line      int                    `json:"line"`
}
