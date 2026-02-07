// Package config provides configuration types and helpers for cyro.
package config

import "time"

// Config holds the application-wide configuration.
type Config struct {
	Format           string   `mapstructure:"format"`
	Verbose          bool     `mapstructure:"verbose"`
	TimestampFormats []string `mapstructure:"timestamp_formats"`
	LogDir           string   `mapstructure:"log_dir"`
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

// ParseLevel converts a string to a LogLevel.
func ParseLevel(s string) LogLevel {
	switch s {
	case "debug", "DEBUG", "dbg":
		return LevelDebug
	case "info", "INFO", "inf":
		return LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return LevelWarn
	case "error", "ERROR", "err":
		return LevelError
	case "fatal", "FATAL", "critical", "CRITICAL":
		return LevelFatal
	default:
		return LevelUnknown
	}
}

// LogEntry represents a single parsed log line.
type LogEntry struct {
	Raw       string            `json:"raw"`
	Timestamp time.Time         `json:"timestamp,omitempty"`
	Level     LogLevel          `json:"level"`
	Message   string            `json:"message"`
	Source    string            `json:"source,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	Line      int               `json:"line"`
}
