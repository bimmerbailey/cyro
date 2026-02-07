package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/bimmerbailey/cyro/internal/config"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Format
	}{
		{
			name:  "JSON log",
			input: `{"timestamp": "2025-01-26T10:00:01Z", "level": "info", "message": "test"}`,
			want:  FormatJSON,
		},
		{
			name:  "Syslog with PID",
			input: "Jan 26 10:00:01 web-01 sshd[1234]: Connection closed",
			want:  FormatSyslog,
		},
		{
			name:  "Syslog without PID",
			input: "Jan 26 10:00:01 web-01 kernel: Out of memory",
			want:  FormatSyslog,
		},
		{
			name:  "Apache Combined Log Format",
			input: `192.168.1.100 - user [26/Jan/2025:10:00:01 -0500] "GET /index.html HTTP/1.1" 200 1234 "https://example.com" "Mozilla/5.0"`,
			want:  FormatApache,
		},
		{
			name:  "Generic log",
			input: "2025-01-26 10:00:01 ERROR Something went wrong",
			want:  FormatGeneric,
		},
		{
			name:  "Generic with brackets",
			input: "[2025-01-26T10:00:01Z] INFO: Application started",
			want:  FormatGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.input)
			if got != tt.want {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ParseJSON(t *testing.T) {
	p := New(nil)

	tests := []struct {
		name          string
		input         string
		wantLevel     config.LogLevel
		wantMessage   string
		wantTimestamp bool
		checkFields   map[string]interface{}
	}{
		{
			name:          "Basic JSON with string fields",
			input:         `{"timestamp": "2025-01-26T10:00:01Z", "level": "info", "message": "test message", "user": "admin"}`,
			wantLevel:     config.LevelInfo,
			wantMessage:   "test message",
			wantTimestamp: true,
			checkFields:   map[string]interface{}{"user": "admin"},
		},
		{
			name:          "JSON with numeric fields",
			input:         `{"timestamp": "2025-01-26T10:00:01Z", "level": "error", "message": "connection failed", "status_code": 500, "retry_count": 3}`,
			wantLevel:     config.LevelError,
			wantMessage:   "connection failed",
			wantTimestamp: true,
			checkFields:   map[string]interface{}{"status_code": float64(500), "retry_count": float64(3)},
		},
		{
			name:          "JSON with boolean fields",
			input:         `{"timestamp": "2025-01-26T10:00:01Z", "level": "warn", "message": "check this", "blocked": true}`,
			wantLevel:     config.LevelWarn,
			wantMessage:   "check this",
			wantTimestamp: true,
			checkFields:   map[string]interface{}{"blocked": true},
		},
		{
			name:          "JSON with epoch seconds timestamp",
			input:         `{"ts": 1706270401, "level": "debug", "message": "debug log"}`,
			wantLevel:     config.LevelDebug,
			wantMessage:   "debug log",
			wantTimestamp: true,
		},
		{
			name:          "JSON with epoch milliseconds timestamp",
			input:         `{"timestamp": 1706270401000, "level": "fatal", "message": "fatal error"}`,
			wantLevel:     config.LevelFatal,
			wantMessage:   "fatal error",
			wantTimestamp: true,
		},
		{
			name:        "JSON with alternative field names",
			input:       `{"time": "2025-01-26T10:00:01Z", "severity": "warning", "msg": "alternative fields"}`,
			wantLevel:   config.LevelWarn,
			wantMessage: "alternative fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := p.parseLine(tt.input, 1)

			if entry.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", entry.Level, tt.wantLevel)
			}
			if entry.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", entry.Message, tt.wantMessage)
			}
			if tt.wantTimestamp && entry.Timestamp.IsZero() {
				t.Error("Expected non-zero timestamp")
			}
			if entry.Raw != tt.input {
				t.Errorf("Raw = %q, want %q", entry.Raw, tt.input)
			}

			// Check specific fields if provided
			for key, expectedValue := range tt.checkFields {
				if val, ok := entry.Fields[key]; !ok {
					t.Errorf("Expected field %q not found in Fields", key)
				} else if val != expectedValue {
					t.Errorf("Field %q = %v, want %v", key, val, expectedValue)
				}
			}
		})
	}
}

func TestParser_ParseSyslog(t *testing.T) {
	p := New(nil)

	tests := []struct {
		name        string
		input       string
		wantLevel   config.LogLevel
		wantMessage string
		wantSource  string
		checkFields map[string]interface{}
	}{
		{
			name:        "Syslog with PID",
			input:       "Jan 26 10:00:01 web-01 sshd[1234]: Accepted password for admin",
			wantLevel:   config.LevelUnknown,
			wantMessage: "Accepted password for admin",
			wantSource:  "web-01",
			checkFields: map[string]interface{}{"process": "sshd", "pid": "1234"},
		},
		{
			name:        "Syslog without PID",
			input:       "Jan 26 10:00:01 db-01 postgres: ERROR: deadlock detected",
			wantLevel:   config.LevelError,
			wantMessage: "ERROR: deadlock detected",
			wantSource:  "db-01",
			checkFields: map[string]interface{}{"process": "postgres"},
		},
		{
			name:        "Syslog with priority (error)",
			input:       "<27>Jan 26 10:00:01 app-01 myapp[999]: Connection established",
			wantLevel:   config.LevelError,
			wantMessage: "Connection established",
			wantSource:  "app-01",
		},
		{
			name:        "Syslog with priority (info)",
			input:       "<30>Jan 26 10:00:01 web-01 nginx: Server started",
			wantLevel:   config.LevelInfo,
			wantMessage: "Server started",
			wantSource:  "web-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := p.parseLine(tt.input, 1)

			if entry.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", entry.Level, tt.wantLevel)
			}
			if entry.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", entry.Message, tt.wantMessage)
			}
			if entry.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", entry.Source, tt.wantSource)
			}

			for key, expectedValue := range tt.checkFields {
				if val, ok := entry.Fields[key]; !ok {
					t.Errorf("Expected field %q not found", key)
				} else if val != expectedValue {
					t.Errorf("Field %q = %v, want %v", key, val, expectedValue)
				}
			}
		})
	}
}

func TestParser_ParseApache(t *testing.T) {
	p := New(nil)

	tests := []struct {
		name        string
		input       string
		wantLevel   config.LogLevel
		wantSource  string
		checkFields map[string]interface{}
	}{
		{
			name:       "Apache 200 OK",
			input:      `192.168.1.100 - user123 [26/Jan/2025:10:00:01 -0500] "GET /index.html HTTP/1.1" 200 1234 "https://example.com" "Mozilla/5.0"`,
			wantLevel:  config.LevelInfo,
			wantSource: "192.168.1.100",
			checkFields: map[string]interface{}{
				"method":      "GET",
				"path":        "/index.html",
				"status_code": "200",
				"user":        "user123",
			},
		},
		{
			name:       "Apache 404 Not Found",
			input:      `10.0.0.50 - - [26/Jan/2025:10:01:15 -0500] "GET /missing HTTP/1.1" 404 567 "-" "curl/7.68.0"`,
			wantLevel:  config.LevelWarn,
			wantSource: "10.0.0.50",
			checkFields: map[string]interface{}{
				"method":      "GET",
				"path":        "/missing",
				"status_code": "404",
			},
		},
		{
			name:       "Apache 500 Server Error",
			input:      `172.16.0.25 - - [26/Jan/2025:10:02:30 -0500] "POST /api/process HTTP/1.1" 500 89 "https://app.example.com" "React Native"`,
			wantLevel:  config.LevelError,
			wantSource: "172.16.0.25",
			checkFields: map[string]interface{}{
				"method":      "POST",
				"status_code": "500",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := p.parseLine(tt.input, 1)

			if entry.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", entry.Level, tt.wantLevel)
			}
			if entry.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", entry.Source, tt.wantSource)
			}
			if entry.Timestamp.IsZero() {
				t.Error("Expected non-zero timestamp")
			}

			for key, expectedValue := range tt.checkFields {
				if val, ok := entry.Fields[key]; !ok {
					t.Errorf("Expected field %q not found", key)
				} else if val != expectedValue {
					t.Errorf("Field %q = %v, want %v", key, val, expectedValue)
				}
			}
		})
	}
}

func TestParser_ParseGeneric(t *testing.T) {
	p := New(nil)

	tests := []struct {
		name          string
		input         string
		wantLevel     config.LogLevel
		wantMessage   string
		wantTimestamp bool
	}{
		{
			name:          "Generic with ISO timestamp",
			input:         "2025-01-26T10:00:01Z ERROR Connection failed",
			wantLevel:     config.LevelError,
			wantMessage:   "Connection failed",
			wantTimestamp: true,
		},
		{
			name:          "Generic with bracketed timestamp",
			input:         "[2025-01-26 10:00:01] INFO Application started",
			wantLevel:     config.LevelInfo,
			wantMessage:   "Application started",
			wantTimestamp: true,
		},
		{
			name:          "Generic with level in brackets",
			input:         "2025-01-26 10:00:01 [WARN] Low disk space",
			wantLevel:     config.LevelWarn,
			wantMessage:   "Low disk space",
			wantTimestamp: true,
		},
		{
			name:        "Generic without timestamp",
			input:       "FATAL: System crash detected",
			wantLevel:   config.LevelFatal,
			wantMessage: "System crash detected",
		},
		{
			name:        "Generic with mixed case level",
			input:       "Debug: Testing feature X",
			wantLevel:   config.LevelDebug,
			wantMessage: "Testing feature X",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := p.parseLine(tt.input, 1)

			if entry.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", entry.Level, tt.wantLevel)
			}
			if !strings.Contains(entry.Message, tt.wantMessage) {
				t.Errorf("Message = %q, want to contain %q", entry.Message, tt.wantMessage)
			}
			if tt.wantTimestamp && entry.Timestamp.IsZero() {
				t.Error("Expected non-zero timestamp")
			}
		})
	}
}

func TestParser_ParseStream(t *testing.T) {
	p := New(nil)

	input := `{"timestamp": "2025-01-26T10:00:01Z", "level": "info", "message": "line 1"}
{"timestamp": "2025-01-26T10:00:02Z", "level": "error", "message": "line 2"}
{"timestamp": "2025-01-26T10:00:03Z", "level": "warn", "message": "line 3"}`

	t.Run("Collect all entries", func(t *testing.T) {
		var entries []config.LogEntry
		err := p.ParseStream(strings.NewReader(input), func(entry config.LogEntry) error {
			entries = append(entries, entry)
			return nil
		})

		if err != nil {
			t.Fatalf("ParseStream() error = %v", err)
		}

		if len(entries) != 3 {
			t.Fatalf("got %d entries, want 3", len(entries))
		}

		if entries[0].Message != "line 1" {
			t.Errorf("entry[0].Message = %q, want %q", entries[0].Message, "line 1")
		}
		if entries[1].Level != config.LevelError {
			t.Errorf("entry[1].Level = %v, want %v", entries[1].Level, config.LevelError)
		}
		if entries[2].Line != 3 {
			t.Errorf("entry[2].Line = %d, want 3", entries[2].Line)
		}
	})

	t.Run("Early termination", func(t *testing.T) {
		count := 0
		err := p.ParseStream(strings.NewReader(input), func(entry config.LogEntry) error {
			count++
			if count >= 2 {
				return errors.New("stop")
			}
			return nil
		})

		if err == nil {
			t.Error("Expected error from early termination")
		}
		if count != 2 {
			t.Errorf("callback called %d times, want 2", count)
		}
	})
}

func TestParser_Parse(t *testing.T) {
	p := New(nil)

	input := `{"timestamp": "2025-01-26T10:00:01Z", "level": "info", "message": "test 1"}
{"timestamp": "2025-01-26T10:00:02Z", "level": "error", "message": "test 2"}`

	entries, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
}

func TestParser_SkipBlankLines(t *testing.T) {
	p := New(nil)

	input := `{"timestamp": "2025-01-26T10:00:01Z", "level": "info", "message": "line 1"}

{"timestamp": "2025-01-26T10:00:02Z", "level": "error", "message": "line 2"}
   
{"timestamp": "2025-01-26T10:00:03Z", "level": "warn", "message": "line 3"}`

	entries, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3 (blank lines should be skipped)", len(entries))
	}
}

func TestParser_CustomTimestampFormats(t *testing.T) {
	customFormats := []string{"01/02/2006 15:04:05"}
	p := New(customFormats)

	entry := p.parseLine("01/26/2025 10:00:01 ERROR Custom timestamp format", 1)

	if entry.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp with custom format")
	}
	if entry.Level != config.LevelError {
		t.Errorf("Level = %v, want %v", entry.Level, config.LevelError)
	}
}

func TestParser_LongLine(t *testing.T) {
	p := New(nil)

	// Create a line longer than the default bufio.Scanner buffer (64KB)
	longMessage := strings.Repeat("x", 100*1024) // 100KB
	input := `{"timestamp": "2025-01-26T10:00:01Z", "level": "info", "message": "` + longMessage + `"}`

	entries, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error on long line = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	if len(entries[0].Message) != len(longMessage) {
		t.Errorf("message length = %d, want %d", len(entries[0].Message), len(longMessage))
	}
}

func TestExtractTimestamp(t *testing.T) {
	p := New(nil)

	tests := []struct {
		name      string
		input     string
		wantFound bool
	}{
		{
			name:      "ISO 8601 at start",
			input:     "2025-01-26T10:00:01Z ERROR test",
			wantFound: true,
		},
		{
			name:      "ISO 8601 with offset",
			input:     "2025-01-26T10:00:01-05:00 INFO test",
			wantFound: true,
		},
		{
			name:      "Datetime format",
			input:     "2025-01-26 10:00:01 WARN test",
			wantFound: true,
		},
		{
			name:      "Bracketed timestamp",
			input:     "[2025-01-26T10:00:01Z] ERROR test",
			wantFound: true,
		},
		{
			name:      "No timestamp",
			input:     "ERROR no timestamp here",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := p.extractTimestamp(tt.input)
			if tt.wantFound && ts.IsZero() {
				t.Error("Expected to find timestamp, got zero")
			}
			if !tt.wantFound && !ts.IsZero() {
				t.Errorf("Expected no timestamp, got %v", ts)
			}
		})
	}
}

func BenchmarkParser_ParseJSON(b *testing.B) {
	p := New(nil)
	line := `{"timestamp": "2025-01-26T10:00:01Z", "level": "error", "message": "test message", "user": "admin", "status_code": 500}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.parseLine(line, 1)
	}
}

func BenchmarkParser_ParseSyslog(b *testing.B) {
	p := New(nil)
	line := "Jan 26 10:00:01 web-01 sshd[1234]: Accepted password for admin from 192.168.1.100"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.parseLine(line, 1)
	}
}

func BenchmarkParser_ParseApache(b *testing.B) {
	p := New(nil)
	line := `192.168.1.100 - user123 [26/Jan/2025:10:00:01 -0500] "GET /index.html HTTP/1.1" 200 1234 "https://example.com" "Mozilla/5.0"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.parseLine(line, 1)
	}
}
