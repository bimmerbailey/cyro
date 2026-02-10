package output

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/bimmerbailey/cyro/internal/config"
)

func TestColorizeLine(t *testing.T) {
	tests := []struct {
		name          string
		level         config.LogLevel
		line          string
		expectColor   bool
		expectedColor string
	}{
		{
			name:          "DEBUG level - gray",
			level:         config.LevelDebug,
			line:          "debug message",
			expectColor:   true,
			expectedColor: colorGray,
		},
		{
			name:        "INFO level - no color",
			level:       config.LevelInfo,
			line:        "info message",
			expectColor: false,
		},
		{
			name:          "WARN level - yellow",
			level:         config.LevelWarn,
			line:          "warning message",
			expectColor:   true,
			expectedColor: colorYellow,
		},
		{
			name:          "ERROR level - red",
			level:         config.LevelError,
			line:          "error message",
			expectColor:   true,
			expectedColor: colorRed,
		},
		{
			name:          "FATAL level - bold red",
			level:         config.LevelFatal,
			line:          "fatal message",
			expectColor:   true,
			expectedColor: colorBold + colorRed,
		},
		{
			name:        "UNKNOWN level - no color",
			level:       config.LevelUnknown,
			line:        "unknown message",
			expectColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ColorizeLine(tt.level, tt.line)

			if tt.expectColor {
				// Should contain color code
				if !strings.Contains(result, tt.expectedColor) {
					t.Errorf("Expected result to contain color code %q, got: %s", tt.expectedColor, result)
				}
				// Should contain reset code
				if !strings.Contains(result, colorReset) {
					t.Errorf("Expected result to contain reset code, got: %s", result)
				}
				// Should contain the original line
				if !strings.Contains(result, tt.line) {
					t.Errorf("Expected result to contain line %q, got: %s", tt.line, result)
				}
			} else {
				// Should be unchanged
				if result != tt.line {
					t.Errorf("Expected line to be unchanged, got: %s", result)
				}
			}
		})
	}
}

func TestColorizeLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         config.LogLevel
		text          string
		expectColor   bool
		expectedColor string
	}{
		{
			name:          "DEBUG",
			level:         config.LevelDebug,
			text:          "DEBUG",
			expectColor:   true,
			expectedColor: colorGray,
		},
		{
			name:        "INFO",
			level:       config.LevelInfo,
			text:        "INFO",
			expectColor: false,
		},
		{
			name:          "WARN",
			level:         config.LevelWarn,
			text:          "WARN",
			expectColor:   true,
			expectedColor: colorYellow,
		},
		{
			name:          "ERROR",
			level:         config.LevelError,
			text:          "ERROR",
			expectColor:   true,
			expectedColor: colorRed,
		},
		{
			name:          "FATAL",
			level:         config.LevelFatal,
			text:          "FATAL",
			expectColor:   true,
			expectedColor: colorBold + colorRed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorizeLevel(tt.level, tt.text)

			if tt.expectColor {
				if !strings.Contains(result, tt.expectedColor) {
					t.Errorf("Expected color code %q in result: %s", tt.expectedColor, result)
				}
			} else {
				if result != tt.text {
					t.Errorf("Expected unchanged text %q, got: %s", tt.text, result)
				}
			}
		})
	}
}

func TestFormatEntry(t *testing.T) {
	entry := config.LogEntry{
		Raw:   "test log line",
		Level: config.LevelError,
	}

	t.Run("with colorize", func(t *testing.T) {
		result := FormatEntry(entry, true)
		// Should contain color codes
		if !strings.Contains(result, colorRed) {
			t.Errorf("Expected red color in result: %s", result)
		}
		if !strings.Contains(result, "test log line") {
			t.Errorf("Expected original line in result: %s", result)
		}
	})

	t.Run("without colorize", func(t *testing.T) {
		result := FormatEntry(entry, false)
		// Should be unchanged
		if result != entry.Raw {
			t.Errorf("Expected raw line %q, got: %s", entry.Raw, result)
		}
		// Should not contain color codes
		if strings.Contains(result, "\033[") {
			t.Errorf("Expected no color codes, got: %s", result)
		}
	})
}

func TestShouldColorize(t *testing.T) {
	tests := []struct {
		name     string
		mode     ColorMode
		writer   interface{}
		expected bool
	}{
		{
			name:     "ColorAlways - any writer",
			mode:     ColorAlways,
			writer:   &bytes.Buffer{},
			expected: true,
		},
		{
			name:     "ColorNever - any writer",
			mode:     ColorNever,
			writer:   os.Stdout,
			expected: false,
		},
		{
			name:     "ColorAuto - non-file writer",
			mode:     ColorAuto,
			writer:   &bytes.Buffer{},
			expected: false,
		},
		{
			name:     "ColorAuto - file writer (stdout)",
			mode:     ColorAuto,
			writer:   os.Stdout,
			expected: isTerminal(os.Stdout), // Depends on test environment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldColorize(tt.mode, tt.writer)
			if result != tt.expected {
				t.Errorf("shouldColorize() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestWriteColoredEntry(t *testing.T) {
	entry := config.LogEntry{
		Raw:   "test error message",
		Level: config.LevelError,
	}

	t.Run("ColorNever mode", func(t *testing.T) {
		buf := &bytes.Buffer{}
		writer := New(buf, FormatText)

		err := writer.WriteColoredEntry(entry, ColorNever)
		if err != nil {
			t.Fatalf("WriteColoredEntry() error = %v", err)
		}

		output := buf.String()
		// Should not contain color codes
		if strings.Contains(output, "\033[") {
			t.Errorf("Expected no color codes, got: %s", output)
		}
		// Should contain the raw message
		if !strings.Contains(output, "test error message") {
			t.Errorf("Expected message in output, got: %s", output)
		}
	})

	t.Run("ColorAlways mode", func(t *testing.T) {
		buf := &bytes.Buffer{}
		writer := New(buf, FormatText)

		err := writer.WriteColoredEntry(entry, ColorAlways)
		if err != nil {
			t.Fatalf("WriteColoredEntry() error = %v", err)
		}

		output := buf.String()
		// Should contain color codes (red for error)
		if !strings.Contains(output, colorRed) {
			t.Errorf("Expected red color code, got: %s", output)
		}
	})

	t.Run("ColorAuto mode with buffer (not TTY)", func(t *testing.T) {
		buf := &bytes.Buffer{}
		writer := New(buf, FormatText)

		err := writer.WriteColoredEntry(entry, ColorAuto)
		if err != nil {
			t.Fatalf("WriteColoredEntry() error = %v", err)
		}

		output := buf.String()
		// Buffer is not a TTY, should not contain color codes
		if strings.Contains(output, "\033[") {
			t.Errorf("Expected no color codes for non-TTY, got: %s", output)
		}
	})
}

func TestColorModeConstants(t *testing.T) {
	// Verify ColorMode constants are distinct
	modes := []ColorMode{ColorAuto, ColorAlways, ColorNever}
	seen := make(map[ColorMode]bool)

	for _, mode := range modes {
		if seen[mode] {
			t.Errorf("Duplicate ColorMode value: %v", mode)
		}
		seen[mode] = true
	}
}

func TestANSIColorCodes(t *testing.T) {
	// Verify ANSI color codes are valid escape sequences
	codes := []struct {
		name  string
		value string
	}{
		{"reset", colorReset},
		{"red", colorRed},
		{"yellow", colorYellow},
		{"gray", colorGray},
		{"bold", colorBold},
	}

	for _, code := range codes {
		t.Run(code.name, func(t *testing.T) {
			if !strings.HasPrefix(code.value, "\033[") {
				t.Errorf("Color code %q should start with ANSI escape sequence", code.name)
			}
			if !strings.HasSuffix(code.value, "m") {
				t.Errorf("Color code %q should end with 'm'", code.name)
			}
		})
	}
}

func TestColorizeLine_PreservesContent(t *testing.T) {
	// Test that coloring doesn't modify the actual content
	testLines := []string{
		"simple line",
		"line with special chars: !@#$%^&*()",
		"line with numbers 12345",
		"line with unicode: 你好世界",
		"line with\ttabs\tand\tspaces",
	}

	for _, line := range testLines {
		t.Run(line, func(t *testing.T) {
			colored := ColorizeLine(config.LevelError, line)

			// Remove ANSI codes
			cleaned := strings.ReplaceAll(colored, colorRed, "")
			cleaned = strings.ReplaceAll(cleaned, colorReset, "")

			if cleaned != line {
				t.Errorf("Content was modified: expected %q, got %q", line, cleaned)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	// Test with stdout (may or may not be terminal depending on test environment)
	result := isTerminal(os.Stdout)
	// Just verify it returns without error - actual value depends on environment
	t.Logf("os.Stdout isTerminal: %v", result)

	// Test with stderr
	result = isTerminal(os.Stderr)
	t.Logf("os.Stderr isTerminal: %v", result)
}
