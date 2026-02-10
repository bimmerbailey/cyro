package tail

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
)

// Helper function to create a temporary log file
func createTempLogFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.log")

	// Ensure content ends with newline for proper parsing
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return filePath
}

// Helper function to count output entries (thread-safe)
func countingOutputFunc(t *testing.T) (func(config.LogEntry) error, func() []config.LogEntry) {
	var mu sync.Mutex
	entries := []config.LogEntry{}

	outputFunc := func(entry config.LogEntry) error {
		mu.Lock()
		defer mu.Unlock()
		entries = append(entries, entry)
		return nil
	}

	getEntries := func() []config.LogEntry {
		mu.Lock()
		defer mu.Unlock()
		// Return a copy to avoid race conditions
		result := make([]config.LogEntry, len(entries))
		copy(result, entries)
		return result
	}

	return outputFunc, getEntries
}

func TestTailer_ReadInitialLines(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		lines         int
		expectedCount int
	}{
		{
			name: "last 3 lines from 5 line file",
			content: `line 1
line 2
line 3
line 4
line 5`,
			lines:         3,
			expectedCount: 3,
		},
		{
			name: "request more lines than exist",
			content: `line 1
line 2`,
			lines:         10,
			expectedCount: 2,
		},
		{
			name:          "single line",
			content:       `single line`,
			lines:         1,
			expectedCount: 1,
		},
		{
			name:          "empty file",
			content:       "",
			lines:         10,
			expectedCount: 0,
		},
		{
			name: "json log entries",
			content: `{"level": "info", "message": "msg1"}
{"level": "error", "message": "msg2"}
{"level": "warn", "message": "msg3"}`,
			lines:         2,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTempLogFile(t, tt.content)
			outputFunc, entries := countingOutputFunc(t)

			tailer := New(Options{
				FilePath:   filePath,
				Lines:      tt.lines,
				Follow:     false,
				OutputFunc: outputFunc,
			})

			ctx := context.Background()
			if err := tailer.Run(ctx); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if len(entries()) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(entries()))
			}
		})
	}
}

func TestTailer_LevelFiltering(t *testing.T) {
	content := `{"level": "debug", "message": "debug msg"}
{"level": "info", "message": "info msg"}
{"level": "warn", "message": "warn msg"}
{"level": "error", "message": "error msg"}
{"level": "fatal", "message": "fatal msg"}`

	tests := []struct {
		name          string
		levelFilter   config.LogLevel
		expectedCount int
		expectLevels  []config.LogLevel
	}{
		{
			name:          "filter error and above",
			levelFilter:   config.LevelError,
			expectedCount: 2,
			expectLevels:  []config.LogLevel{config.LevelError, config.LevelFatal},
		},
		{
			name:          "filter warn and above",
			levelFilter:   config.LevelWarn,
			expectedCount: 3,
			expectLevels:  []config.LogLevel{config.LevelWarn, config.LevelError, config.LevelFatal},
		},
		{
			name:          "filter info and above",
			levelFilter:   config.LevelInfo,
			expectedCount: 4,
			expectLevels:  []config.LogLevel{config.LevelInfo, config.LevelWarn, config.LevelError, config.LevelFatal},
		},
		{
			name:          "no filter (unknown level)",
			levelFilter:   config.LevelUnknown,
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTempLogFile(t, content)
			outputFunc, entries := countingOutputFunc(t)

			tailer := New(Options{
				FilePath:    filePath,
				Lines:       10,
				Follow:      false,
				LevelFilter: tt.levelFilter,
				OutputFunc:  outputFunc,
			})

			ctx := context.Background()
			if err := tailer.Run(ctx); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if len(entries()) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(entries()))
			}

			// Verify the levels of returned entries
			if tt.expectLevels != nil {
				allEntries := entries()
				for i, entry := range allEntries {
					if i < len(tt.expectLevels) {
						expectedLevel := tt.expectLevels[i]
						if entry.Level != expectedLevel {
							t.Errorf("Entry %d: expected level %s, got %s", i, expectedLevel, entry.Level)
						}
					}
				}
			}
		})
	}
}

func TestTailer_PatternFiltering(t *testing.T) {
	content := `Line with user_id=123
Line with user_id=456
Line with product_id=789
Line with user_id=123 again
No special content here`

	tests := []struct {
		name          string
		pattern       string
		expectedCount int
	}{
		{
			name:          "match user_id",
			pattern:       "user_id",
			expectedCount: 3,
		},
		{
			name:          "match specific user",
			pattern:       "user_id=123",
			expectedCount: 2,
		},
		{
			name:          "match product",
			pattern:       "product_id",
			expectedCount: 1,
		},
		{
			name:          "regex pattern",
			pattern:       "user_id=\\d+",
			expectedCount: 3,
		},
		{
			name:          "no matches",
			pattern:       "nonexistent",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTempLogFile(t, content)
			outputFunc, entries := countingOutputFunc(t)

			pattern, err := regexp.Compile(tt.pattern)
			if err != nil {
				t.Fatalf("Invalid pattern: %v", err)
			}

			tailer := New(Options{
				FilePath:   filePath,
				Lines:      10,
				Follow:     false,
				Pattern:    pattern,
				OutputFunc: outputFunc,
			})

			ctx := context.Background()
			if err := tailer.Run(ctx); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if len(entries()) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(entries()))
			}
		})
	}
}

func TestTailer_CombinedFiltering(t *testing.T) {
	content := `{"level": "info", "message": "user login", "user_id": "123"}
{"level": "error", "message": "user failed", "user_id": "123"}
{"level": "error", "message": "database error", "component": "db"}
{"level": "warn", "message": "user timeout", "user_id": "123"}
{"level": "info", "message": "system healthy"}`

	filePath := createTempLogFile(t, content)
	outputFunc, entries := countingOutputFunc(t)

	// Filter: error level AND contains "user"
	pattern, _ := regexp.Compile("user")
	tailer := New(Options{
		FilePath:    filePath,
		Lines:       10,
		Follow:      false,
		LevelFilter: config.LevelError,
		Pattern:     pattern,
		OutputFunc:  outputFunc,
	})

	ctx := context.Background()
	if err := tailer.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should only match: error level with "user" in message
	// Expected: 1 (the "user failed" error)
	expectedCount := 1
	if len(entries()) != expectedCount {
		t.Errorf("Expected %d entries, got %d", expectedCount, len(entries()))
	}

	if len(entries()) > 0 {
		entry := entries()[0]
		if entry.Level != config.LevelError {
			t.Errorf("Expected level ERROR, got %s", entry.Level)
		}
		if !strings.Contains(entry.Raw, "user") {
			t.Errorf("Expected entry to contain 'user', got: %s", entry.Raw)
		}
	}
}

func TestTailer_FollowMode(t *testing.T) {
	// Create initial file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "follow.log")

	initialContent := "line 1\nline 2\n"
	if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	outputFunc, entries := countingOutputFunc(t)
	tailer := New(Options{
		FilePath:   filePath,
		Lines:      2,
		Follow:     true,
		OutputFunc: outputFunc,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start tailing in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- tailer.Run(ctx)
	}()

	// Wait for initial lines to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify initial 2 lines were read
	if len(entries()) != 2 {
		t.Errorf("Expected 2 initial entries, got %d", len(entries()))
	}

	// Append new content
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for append: %v", err)
	}

	if _, err := f.WriteString("line 3\n"); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}
	f.Close()

	// Wait for new line to be detected and processed
	time.Sleep(300 * time.Millisecond)

	// Should now have 3 entries total
	if len(entries()) != 3 {
		t.Errorf("Expected 3 entries after append, got %d", len(entries()))
	}

	// Verify the third entry
	if len(entries()) >= 3 {
		if !strings.Contains(entries()[2].Raw, "line 3") {
			t.Errorf("Expected third entry to contain 'line 3', got: %s", entries()[2].Raw)
		}
	}

	// Cancel context to stop tailing
	cancel()

	// Wait for tailer to finish
	select {
	case <-errCh:
		// Tailer stopped successfully
	case <-time.After(2 * time.Second):
		t.Error("Tailer did not stop within timeout")
	}
}

func TestTailer_NoFollowMode(t *testing.T) {
	content := "line 1\nline 2\nline 3\n"
	filePath := createTempLogFile(t, content)

	outputFunc, entries := countingOutputFunc(t)
	tailer := New(Options{
		FilePath:   filePath,
		Lines:      2,
		Follow:     false, // Don't follow
		OutputFunc: outputFunc,
	})

	ctx := context.Background()
	start := time.Now()

	if err := tailer.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	duration := time.Since(start)

	// Should complete quickly (< 500ms)
	if duration > 500*time.Millisecond {
		t.Errorf("No-follow mode took too long: %v", duration)
	}

	// Should have read last 2 lines
	if len(entries()) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries()))
	}
}

func TestTailer_MultipleLogFormats(t *testing.T) {
	tests := []struct {
		name    string
		content string
		lines   int
	}{
		{
			name: "JSON format",
			content: `{"timestamp": "2025-01-26T10:00:00Z", "level": "info", "message": "test"}
{"timestamp": "2025-01-26T10:01:00Z", "level": "error", "message": "error"}`,
			lines: 2,
		},
		{
			name: "Syslog format",
			content: `Jan 26 10:00:00 host app[123]: INFO: test message
Jan 26 10:01:00 host app[123]: ERROR: error message`,
			lines: 2,
		},
		{
			name: "Apache format",
			content: `192.168.1.1 - - [26/Jan/2025:10:00:00 -0500] "GET / HTTP/1.1" 200 1234 "-" "Mozilla"
192.168.1.2 - - [26/Jan/2025:10:01:00 -0500] "POST /api HTTP/1.1" 500 567 "-" "curl"`,
			lines: 2,
		},
		{
			name: "Generic text",
			content: `2025-01-26 10:00:00 INFO: test message
2025-01-26 10:01:00 ERROR: error message`,
			lines: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTempLogFile(t, tt.content)
			outputFunc, entries := countingOutputFunc(t)

			tailer := New(Options{
				FilePath:   filePath,
				Lines:      tt.lines,
				Follow:     false,
				OutputFunc: outputFunc,
			})

			ctx := context.Background()
			if err := tailer.Run(ctx); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if len(entries()) != tt.lines {
				t.Errorf("Expected %d entries, got %d", tt.lines, len(entries()))
			}

			// Verify all entries have non-empty Raw field
			allEntries := entries()
			for i, entry := range allEntries {
				if entry.Raw == "" {
					t.Errorf("Entry %d has empty Raw field", i)
				}
			}
		})
	}
}

func TestShouldDisplay(t *testing.T) {
	tests := []struct {
		name        string
		entry       config.LogEntry
		levelFilter config.LogLevel
		pattern     *regexp.Regexp
		expected    bool
	}{
		{
			name: "no filters - should display",
			entry: config.LogEntry{
				Raw:   "test message",
				Level: config.LevelInfo,
			},
			levelFilter: config.LevelUnknown,
			pattern:     nil,
			expected:    true,
		},
		{
			name: "level filter - matches",
			entry: config.LogEntry{
				Raw:   "error message",
				Level: config.LevelError,
			},
			levelFilter: config.LevelError,
			pattern:     nil,
			expected:    true,
		},
		{
			name: "level filter - below threshold",
			entry: config.LogEntry{
				Raw:   "info message",
				Level: config.LevelInfo,
			},
			levelFilter: config.LevelError,
			pattern:     nil,
			expected:    false,
		},
		{
			name: "pattern filter - matches",
			entry: config.LogEntry{
				Raw:   "test user_id=123",
				Level: config.LevelInfo,
			},
			levelFilter: config.LevelUnknown,
			pattern:     regexp.MustCompile("user_id"),
			expected:    true,
		},
		{
			name: "pattern filter - no match",
			entry: config.LogEntry{
				Raw:   "test message",
				Level: config.LevelInfo,
			},
			levelFilter: config.LevelUnknown,
			pattern:     regexp.MustCompile("nonexistent"),
			expected:    false,
		},
		{
			name: "both filters - both match",
			entry: config.LogEntry{
				Raw:   "error user_id=123",
				Level: config.LevelError,
			},
			levelFilter: config.LevelError,
			pattern:     regexp.MustCompile("user_id"),
			expected:    true,
		},
		{
			name: "both filters - level fails",
			entry: config.LogEntry{
				Raw:   "info user_id=123",
				Level: config.LevelInfo,
			},
			levelFilter: config.LevelError,
			pattern:     regexp.MustCompile("user_id"),
			expected:    false,
		},
		{
			name: "both filters - pattern fails",
			entry: config.LogEntry{
				Raw:   "error message",
				Level: config.LevelError,
			},
			levelFilter: config.LevelError,
			pattern:     regexp.MustCompile("user_id"),
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tailer := &Tailer{
				opts: Options{
					LevelFilter: tt.levelFilter,
					Pattern:     tt.pattern,
				},
			}

			result := tailer.shouldDisplay(tt.entry)
			if result != tt.expected {
				t.Errorf("shouldDisplay() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestLevelToInt(t *testing.T) {
	tests := []struct {
		level    config.LogLevel
		expected int
	}{
		{config.LevelDebug, 0},
		{config.LevelInfo, 1},
		{config.LevelWarn, 2},
		{config.LevelError, 3},
		{config.LevelFatal, 4},
		{config.LevelUnknown, -1},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			result := levelToInt(tt.level)
			if result != tt.expected {
				t.Errorf("levelToInt() = %d, expected %d for %s", result, tt.expected, tt.level.String())
			}
		})
	}
}

func TestTailer_EmptyLines(t *testing.T) {
	// File with empty lines should skip them (parser skips blank lines)
	content := `line 1

line 3


line 6`

	filePath := createTempLogFile(t, content)
	outputFunc, entries := countingOutputFunc(t)

	tailer := New(Options{
		FilePath:   filePath,
		Lines:      10,
		Follow:     false,
		OutputFunc: outputFunc,
	})

	ctx := context.Background()
	if err := tailer.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should have 3 non-empty lines
	if len(entries()) != 3 {
		t.Errorf("Expected 3 entries (empty lines skipped), got %d", len(entries()))
	}
}

func TestTailer_LargeFile(t *testing.T) {
	// Create a file with many lines to test the seeking heuristic
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large.log")

	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Write 1000 lines
	for i := 1; i <= 1000; i++ {
		f.WriteString("This is log line number " + string(rune(i)) + " with some content to make it longer\n")
	}
	f.Close()

	outputFunc, entries := countingOutputFunc(t)
	tailer := New(Options{
		FilePath:   filePath,
		Lines:      10,
		Follow:     false,
		OutputFunc: outputFunc,
	})

	ctx := context.Background()
	if err := tailer.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should get exactly 10 lines (last 10)
	if len(entries()) != 10 {
		t.Errorf("Expected 10 entries, got %d", len(entries()))
	}
}
