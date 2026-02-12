package preprocess

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
)

// Test Patterns
func TestBuiltInPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
		want    bool
	}{
		{
			name:    "IPv4 address",
			pattern: "ipv4",
			text:    "Connection from 192.168.1.1 to server",
			want:    true,
		},
		{
			name:    "Email address",
			pattern: "email",
			text:    "User user@example.com logged in",
			want:    true,
		},
		{
			name:    "AWS Access Key",
			pattern: "aws_key",
			text:    "Key: AKIAIOSFODNN7EXAMPLE",
			want:    true,
		},
		{
			name:    "JWT token",
			pattern: "jwt",
			text:    "Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			want:    true,
		},
		{
			name:    "No sensitive data",
			pattern: "ipv4",
			text:    "This is a normal log message",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, ok := BuiltInPatterns[tt.pattern]
			if !ok {
				t.Fatalf("Pattern %s not found", tt.pattern)
			}

			got := pattern.Regex.MatchString(tt.text)
			if got != tt.want {
				t.Errorf("MatchString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultPatterns(t *testing.T) {
	patterns := DefaultPatterns()
	if len(patterns) == 0 {
		t.Error("DefaultPatterns() returned empty slice")
	}

	// Check that all default patterns exist
	for _, name := range patterns {
		if _, ok := BuiltInPatterns[name]; !ok {
			t.Errorf("Default pattern %s not found in BuiltInPatterns", name)
		}
	}
}

func TestGetPatterns(t *testing.T) {
	patterns := GetPatterns([]string{"ipv4", "email", "nonexistent"})
	if len(patterns) != 2 {
		t.Errorf("GetPatterns() returned %d patterns, want 2", len(patterns))
	}
}

// Test Redaction
func TestRedactorBasic(t *testing.T) {
	redactor := NewRedactor(true, []string{"ipv4"})

	text := "Connection from 192.168.1.1 to 10.0.0.1"
	got := redactor.Redact(text)

	// Should have redacted both IPs
	if strings.Contains(got, "192.168.1.1") {
		t.Error("Redactor did not redact 192.168.1.1")
	}
	if strings.Contains(got, "10.0.0.1") {
		t.Error("Redactor did not redact 10.0.0.1")
	}

	// Should have placeholders
	if !strings.Contains(got, "[IPV4:") {
		t.Error("Redactor did not add IPV4 placeholder")
	}
}

func TestRedactorCorrelation(t *testing.T) {
	redactor := NewRedactor(true, []string{"ipv4"})

	// Same IP should always get same placeholder
	text1 := "Connection from 192.168.1.1"
	text2 := "Disconnection from 192.168.1.1"

	got1 := redactor.Redact(text1)
	got2 := redactor.Redact(text2)

	// Extract placeholders
	placeholder1 := extractPlaceholder(got1, "IPV4")
	placeholder2 := extractPlaceholder(got2, "IPV4")

	if placeholder1 == "" || placeholder2 == "" {
		t.Fatal("Failed to extract placeholders")
	}

	if placeholder1 != placeholder2 {
		t.Errorf("Same IP got different placeholders: %s vs %s", placeholder1, placeholder2)
	}
}

func TestRedactorDisabled(t *testing.T) {
	redactor := NewRedactor(false, []string{"ipv4"})

	text := "Connection from 192.168.1.1"
	got := redactor.Redact(text)

	if got != text {
		t.Error("Disabled redactor modified the text")
	}
}

func TestRedactorCount(t *testing.T) {
	redactor := NewRedactor(true, []string{"ipv4", "email"})

	text := "User user@example.com from 192.168.1.1 contacted admin@example.com at 10.0.0.1"
	got, count := redactor.RedactAndCount(text)

	if count != 4 {
		t.Errorf("RedactAndCount() count = %d, want 4", count)
	}

	if strings.Contains(got, "192.168.1.1") || strings.Contains(got, "user@example.com") {
		t.Error("Redaction did not replace all sensitive values")
	}
}

func TestRedactorGetUniqueValues(t *testing.T) {
	redactor := NewRedactor(true, []string{"ipv4"})

	redactor.Redact("IP: 192.168.1.1")
	redactor.Redact("IP: 192.168.1.1") // Same IP
	redactor.Redact("IP: 10.0.0.1")    // Different IP

	values := redactor.GetUniqueValues()
	if len(values) != 2 {
		t.Errorf("GetUniqueValues() returned %d values, want 2", len(values))
	}
}

// Test Drain Algorithm
func TestDrainExtractorBasic(t *testing.T) {
	extractor := NewDrainExtractor(4, 0.5, 100)

	// First pair: IPs are variables, should group
	extractor.Extract("Connection from 192.168.1.1 established")
	extractor.Extract("Connection from 10.0.0.1 established")

	// Second pair: IDs are variables, should group
	extractor.Extract("User 12345 logged in")
	extractor.Extract("User 67890 logged in")

	templates := extractor.GetTemplates()
	if len(templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(templates))
	}

	// Check that similar messages were grouped
	for _, tmpl := range templates {
		if tmpl.Count != 2 {
			t.Errorf("Template %s has count %d, want 2", tmpl.Pattern, tmpl.Count)
		}
	}
}

func TestDrainExtractorVariableTokens(t *testing.T) {
	extractor := NewDrainExtractor(4, 0.5, 100)

	// These should be grouped together because IPs and user IDs are variables
	messages := []string{
		"User 12345 logged in from 192.168.1.1",
		"User 67890 logged in from 10.0.0.1",
		"User 11111 logged in from 172.16.0.1",
	}

	for _, msg := range messages {
		extractor.Extract(msg)
	}

	templates := extractor.GetTemplates()
	if len(templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(templates))
	}

	if templates[0].Count != 3 {
		t.Errorf("Template count = %d, want 3", templates[0].Count)
	}

	// Pattern should have wildcards
	if !strings.Contains(templates[0].Pattern, "<*>") {
		t.Errorf("Template pattern %s should contain wildcards", templates[0].Pattern)
	}
}

func TestDrainExtractorSimilarity(t *testing.T) {
	extractor := NewDrainExtractor(4, 0.5, 100)

	// Different messages that should NOT be grouped
	messages := []string{
		"Database connection established",
		"User authentication successful",
		"Cache miss for key abc123",
	}

	for _, msg := range messages {
		extractor.Extract(msg)
	}

	templates := extractor.GetTemplates()
	if len(templates) != 3 {
		t.Errorf("Expected 3 templates, got %d", len(templates))
	}
}

func TestDrainExtractorReset(t *testing.T) {
	extractor := NewDrainExtractor(4, 0.5, 100)

	extractor.Extract("Test message 1")
	extractor.Extract("Test message 2")

	if extractor.GetTemplateCount() != 1 {
		t.Errorf("Expected 1 template before reset, got %d", extractor.GetTemplateCount())
	}

	extractor.Reset()

	if extractor.GetTemplateCount() != 0 {
		t.Error("Expected 0 templates after reset")
	}
}

func TestDrainCalculateSimilarity(t *testing.T) {
	extractor := NewDrainExtractor(4, 0.5, 100)

	tests := []struct {
		tokens1 []string
		tokens2 []string
		want    float64
	}{
		{
			tokens1: []string{"User", "logged", "in"},
			tokens2: []string{"User", "logged", "in"},
			want:    1.0,
		},
		{
			tokens1: []string{"User", "logged", "in"},
			tokens2: []string{"User", "logged", "out"},
			want:    0.67, // 2/3
		},
		{
			tokens1: []string{"User", "logged", "in"},
			tokens2: []string{"Database", "connected"},
			want:    0.0,
		},
		{
			tokens1: []string{"User", "<*>", "in"},
			tokens2: []string{"User", "12345", "in"},
			want:    1.0, // Wildcard matches anything
		},
	}

	for _, tt := range tests {
		got := extractor.calculateSimilarity(tt.tokens1, tt.tokens2)
		diff := got - tt.want
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.01 {
			t.Errorf("calculateSimilarity() = %f, want %f", got, tt.want)
		}
	}
}

// Test Compression
func TestCompressorBasic(t *testing.T) {
	compressor := NewCompressor(1000)

	entries := []config.LogEntry{
		{Message: "Error: Connection failed", Level: config.LevelError, Timestamp: time.Now()},
		{Message: "Warning: High memory usage", Level: config.LevelWarn, Timestamp: time.Now()},
		{Message: "Info: Server started", Level: config.LevelInfo, Timestamp: time.Now()},
	}

	// Create templates that actually match the entries - each message becomes a template
	templates := []*Template{
		{ID: "T1", Pattern: "Error: Connection failed", Tokens: []string{"Error:", "Connection", "failed"}, Count: 1, Examples: []string{"Error: Connection failed"}},
		{ID: "T2", Pattern: "Warning: High memory usage", Tokens: []string{"Warning:", "High", "memory", "usage"}, Count: 1, Examples: []string{"Warning: High memory usage"}},
		{ID: "T3", Pattern: "Info: Server started", Tokens: []string{"Info:", "Server", "started"}, Count: 1, Examples: []string{"Info: Server started"}},
	}

	output, err := compressor.Compress(entries, templates, 0)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if output.TotalLines != 3 {
		t.Errorf("TotalLines = %d, want 3", output.TotalLines)
	}

	if output.TotalTemplates != 3 {
		t.Errorf("TotalTemplates = %d, want 3", output.TotalTemplates)
	}

	if output.TokenCount <= 0 {
		t.Error("TokenCount should be positive")
	}
}

func TestCompressorEmpty(t *testing.T) {
	compressor := NewCompressor(1000)

	output, err := compressor.Compress([]config.LogEntry{}, []*Template{}, 0)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if output.TotalLines != 0 {
		t.Errorf("TotalLines = %d, want 0", output.TotalLines)
	}

	if !strings.Contains(output.Summary, "No log entries") {
		t.Error("Empty summary should indicate no entries")
	}
}

func TestCompressorTokenBudget(t *testing.T) {
	compressor := NewCompressor(500) // Small budget

	// Create entries and matching templates - use simple messages that match exactly
	entries := make([]config.LogEntry, 20)
	templates := make([]*Template, 0, 20)

	for i := 0; i < 20; i++ {
		msg := fmt.Sprintf("Log message %d", i)
		entries[i] = config.LogEntry{
			Message:   msg,
			Level:     config.LevelInfo,
			Timestamp: time.Now(),
		}
		// Create a template for each entry
		templates = append(templates, &Template{
			ID:       fmt.Sprintf("T%d", i),
			Pattern:  msg,
			Tokens:   []string{"Log", "message", fmt.Sprintf("%d", i)},
			Count:    1,
			Examples: []string{msg},
		})
	}

	output, err := compressor.Compress(entries, templates, 0)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	// Should be within budget (or close to it)
	if output.TokenCount > output.TokenLimit+100 {
		t.Errorf("TokenCount %d exceeds TokenLimit %d by more than 100",
			output.TokenCount, output.TokenLimit)
	}

	// Should have included some but not all templates due to budget
	if len(output.Templates) == 0 {
		t.Error("Should have included at least some templates")
	}
}

func TestCompressorPriority(t *testing.T) {
	compressor := NewCompressor(1000)

	entries := []config.LogEntry{
		{Message: "Error critical", Level: config.LevelError, Timestamp: time.Now()},
		{Message: "Warning minor", Level: config.LevelWarn, Timestamp: time.Now()},
		{Message: "Info message 1", Level: config.LevelInfo, Timestamp: time.Now()},
		{Message: "Info message 2", Level: config.LevelInfo, Timestamp: time.Now()},
	}

	templates := []*Template{
		{ID: "T1", Pattern: "Error <*>", Count: 1},
		{ID: "T2", Pattern: "Warning <*>", Count: 1},
		{ID: "T3", Pattern: "Info <*>", Count: 2},
	}

	output, err := compressor.Compress(entries, templates, 0)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	// Check that errors come before warnings in output
	errorIdx := strings.Index(output.Summary, "Error")
	warnIdx := strings.Index(output.Summary, "Warning")

	if errorIdx == -1 || warnIdx == -1 {
		t.Error("Summary should contain Error and Warning sections")
	}

	if errorIdx > warnIdx {
		t.Error("Errors should appear before warnings in summary")
	}
}

// Test Preprocessor Integration
func TestPreprocessorBasic(t *testing.T) {
	preprocessor := New(
		WithTokenLimit(8000),
		WithRedaction(true),
	)

	entries := []config.LogEntry{
		{
			Raw:       "192.168.1.1 - User login success",
			Message:   "192.168.1.1 - User login success",
			Level:     config.LevelInfo,
			Timestamp: time.Now(),
		},
		{
			Raw:       "10.0.0.1 - User login success",
			Message:   "10.0.0.1 - User login success",
			Level:     config.LevelInfo,
			Timestamp: time.Now(),
		},
		{
			Raw:       "192.168.1.1 - Database connection failed",
			Message:   "192.168.1.1 - Database connection failed",
			Level:     config.LevelError,
			Timestamp: time.Now(),
		},
	}

	output, err := preprocessor.Process(entries)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// Should have redacted IPs
	if output.RedactedCount < 2 {
		t.Errorf("RedactedCount = %d, want at least 2", output.RedactedCount)
	}

	// Should have extracted templates
	if output.TotalTemplates < 1 {
		t.Errorf("TotalTemplates = %d, want at least 1", output.TotalTemplates)
	}

	// Summary should contain compressed info
	if !strings.Contains(output.Summary, "Log Analysis Summary") {
		t.Error("Summary should contain header")
	}

	// Should be within budget
	if !output.IsWithinBudget() {
		t.Errorf("Output exceeds budget: %d/%d tokens", output.TokenCount, output.TokenLimit)
	}
}

func TestPreprocessorWithOptions(t *testing.T) {
	preprocessor := New(
		WithTokenLimit(4000),
		WithRedaction(true),
		WithRedactionPatterns([]string{"ipv4"}),
		WithDrainConfig(4, 0.5, 100),
		WithDebug(true),
	)

	entries := []config.LogEntry{
		{
			Message:   "Server started on 192.168.1.1",
			Level:     config.LevelInfo,
			Timestamp: time.Now(),
		},
	}

	output, err := preprocessor.Process(entries)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if output.TokenLimit != 4000 {
		t.Errorf("TokenLimit = %d, want 4000", output.TokenLimit)
	}

	// Debug mode should add metadata
	if _, ok := output.Metadata["templates_debug"]; !ok {
		t.Error("Debug mode should add template debug info")
	}
}

func TestPreprocessorReset(t *testing.T) {
	preprocessor := New()

	preprocessor.Process([]config.LogEntry{
		{Message: "Test 1", Level: config.LevelInfo, Timestamp: time.Now()},
	})

	if preprocessor.GetTemplateCount() != 1 {
		t.Error("Expected 1 template before reset")
	}

	preprocessor.Reset()

	if preprocessor.GetTemplateCount() != 0 {
		t.Error("Expected 0 templates after reset")
	}
}

func TestPreprocessorEmpty(t *testing.T) {
	preprocessor := New()

	output, err := preprocessor.Process([]config.LogEntry{})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if output.TotalLines != 0 {
		t.Errorf("TotalLines = %d, want 0", output.TotalLines)
	}
}

func TestProcessWithStats(t *testing.T) {
	preprocessor := New()

	// Use messages with variable tokens so Drain can group them
	entries := []config.LogEntry{
		{Message: "Error code 1001 occurred", Level: config.LevelError, Timestamp: time.Now()},
		{Message: "Error code 1002 occurred", Level: config.LevelError, Timestamp: time.Now()},
		{Message: "Warning: High memory 85%", Level: config.LevelWarn, Timestamp: time.Now()},
		{Message: "Server started on port 8080", Level: config.LevelInfo, Timestamp: time.Now()},
	}

	_, stats, err := preprocessor.ProcessWithStats(entries)
	if err != nil {
		t.Fatalf("ProcessWithStats() error = %v", err)
	}

	if stats.InputLines != 4 {
		t.Errorf("InputLines = %d, want 4", stats.InputLines)
	}

	// Should have some templates (exact count depends on grouping)
	if stats.OutputTemplates == 0 {
		t.Error("Should have at least some templates")
	}

	if stats.CompressionRatio <= 0 {
		t.Error("CompressionRatio should be positive")
	}

	// Test String() method
	statsStr := stats.String()
	if !strings.Contains(statsStr, "Processed") {
		t.Error("Stats String() should contain summary")
	}
}

// Test QuickCompress convenience functions
func TestQuickCompress(t *testing.T) {
	entries := []config.LogEntry{
		{Message: "Test message", Level: config.LevelInfo, Timestamp: time.Now()},
	}

	output, err := QuickCompress(entries, 2000, true)
	if err != nil {
		t.Fatalf("QuickCompress() error = %v", err)
	}

	if output.TokenLimit != 2000 {
		t.Errorf("TokenLimit = %d, want 2000", output.TokenLimit)
	}
}

func TestQuickCompressWithDefaults(t *testing.T) {
	entries := []config.LogEntry{
		{Message: "Test message", Level: config.LevelInfo, Timestamp: time.Now()},
	}

	output, err := QuickCompressWithDefaults(entries)
	if err != nil {
		t.Fatalf("QuickCompressWithDefaults() error = %v", err)
	}

	if output.TokenLimit != DefaultTokenLimit {
		t.Errorf("TokenLimit = %d, want %d", output.TokenLimit, DefaultTokenLimit)
	}
}

// Helper function to extract placeholder value
func extractPlaceholder(text, typ string) string {
	prefix := fmt.Sprintf("[%s:", typ)
	start := strings.Index(text, prefix)
	if start == -1 {
		return ""
	}
	end := strings.Index(text[start:], "]")
	if end == -1 {
		return ""
	}
	return text[start : start+end+1]
}

// Benchmark tests
func BenchmarkDrainExtractor(b *testing.B) {
	extractor := NewDrainExtractor(4, 0.5, 100)
	messages := []string{
		"Connection from 192.168.1.1 established",
		"User 12345 logged in from 10.0.0.1",
		"Database query took 45ms",
		"Cache hit for key abc123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.Extract(messages[i%len(messages)])
	}
}

func BenchmarkRedactor(b *testing.B) {
	redactor := NewRedactor(true, DefaultPatterns())
	text := "User user@example.com from 192.168.1.1 with API key sk-abc123def456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		redactor.Redact(text)
	}
}

func BenchmarkPreprocessor(b *testing.B) {
	preprocessor := New()
	entries := make([]config.LogEntry, 100)
	for i := range entries {
		entries[i] = config.LogEntry{
			Message:   fmt.Sprintf("Log message %d with variable %d", i, i*10),
			Level:     config.LevelInfo,
			Timestamp: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		preprocessor.Process(entries)
		preprocessor.Reset()
	}
}
