// Package config provides configuration types and helpers for cyro.
package config

import (
	"encoding/json"
	"strings"
	"time"
)

// Config holds the application-wide configuration.
type Config struct {
	Format           string    `mapstructure:"format"`
	Verbose          bool      `mapstructure:"verbose"`
	TimestampFormats []string  `mapstructure:"timestamp_formats"`
	LogDir           string    `mapstructure:"log_dir"`
	LLM              LLMConfig `mapstructure:"llm"`
}

// LLMConfig holds configuration for LLM providers.
type LLMConfig struct {
	Provider    string       `mapstructure:"provider"`
	Temperature float32      `mapstructure:"temperature"`
	TokenBudget int          `mapstructure:"token_budget"`
	Ollama      OllamaConfig `mapstructure:"ollama"`
}

// OllamaConfig holds Ollama-specific configuration.
type OllamaConfig struct {
	Host           string `mapstructure:"host"`
	Model          string `mapstructure:"model"`
	EmbeddingModel string `mapstructure:"embedding_model"`
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
