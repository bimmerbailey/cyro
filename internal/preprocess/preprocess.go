package preprocess

import (
	"fmt"

	"github.com/bimmerbailey/cyro/internal/config"
)

// Preprocessor orchestrates the complete log preprocessing pipeline.
//
// The pipeline consists of three stages:
//  1. Redaction - Remove sensitive data with correlation-preserving hashes
//  2. Drain Template Extraction - Group similar messages into templates
//  3. Compression - Apply token budget and format output
//
// Usage:
//
//	preprocessor := preprocess.New(
//	    preprocess.WithTokenLimit(8000),
//	    preprocess.WithRedaction(true),
//	    preprocess.WithRedactionPatterns([]string{"ipv4", "email", "api_key"}),
//	)
//
//	output, err := preprocessor.Process(logEntries)
//	if err != nil {
//	    return err
//	}
//
//	fmt.Println(output.Summary)
//	fmt.Printf("Compression ratio: %.1fx\n", output.GetCompressionRatio())
type Preprocessor struct {
	redactor   *Redactor
	drain      *DrainExtractor
	compressor *Compressor
	tokenLimit int
	debug      bool
}

// Option configures a Preprocessor.
type Option func(*Preprocessor)

// WithTokenLimit sets the maximum token limit for output.
// Default is 8000 tokens.
func WithTokenLimit(limit int) Option {
	return func(p *Preprocessor) {
		p.tokenLimit = limit
	}
}

// WithRedaction enables or disables secret redaction.
// Default is enabled.
func WithRedaction(enabled bool) Option {
	return func(p *Preprocessor) {
		if p.redactor != nil {
			p.redactor.enabled = enabled
		}
	}
}

// WithRedactionPatterns sets which redaction patterns to use.
// Default patterns are used if not specified.
func WithRedactionPatterns(patterns []string) Option {
	return func(p *Preprocessor) {
		p.redactor = NewRedactor(p.redactor.enabled, patterns)
	}
}

// WithDrainConfig configures the Drain algorithm parameters.
func WithDrainConfig(depth int, simThreshold float64, maxChildren int) Option {
	return func(p *Preprocessor) {
		p.drain = NewDrainExtractor(depth, simThreshold, maxChildren)
	}
}

// WithDebug enables debug mode which includes template lineage information.
func WithDebug(enabled bool) Option {
	return func(p *Preprocessor) {
		p.debug = enabled
	}
}

// New creates a new Preprocessor with the specified options.
//
// Example with all options:
//
//	preprocessor := preprocess.New(
//	    preprocess.WithTokenLimit(4000),
//	    preprocess.WithRedaction(true),
//	    preprocess.WithRedactionPatterns([]string{"ipv4", "email"}),
//	    preprocess.WithDrainConfig(4, 0.5, 100),
//	)
func New(opts ...Option) *Preprocessor {
	// Default configuration
	p := &Preprocessor{
		redactor:   NewRedactor(true, DefaultPatterns()),
		drain:      NewDrainExtractor(0, 0, 0), // Uses defaults
		compressor: NewCompressor(DefaultTokenLimit),
		tokenLimit: DefaultTokenLimit,
		debug:      false,
	}

	// Apply options
	for _, opt := range opts {
		opt(p)
	}

	// Re-create compressor with correct token limit
	p.compressor = NewCompressor(p.tokenLimit)

	return p
}

// Process runs the complete preprocessing pipeline on the given log entries.
//
// The pipeline stages:
//  1. Redaction: Scans all messages for sensitive patterns
//  2. Drain: Extracts templates from redacted messages
//  3. Compression: Formats output respecting token budget
//
// Returns a CompressedOutput containing the formatted summary and metadata.
func (p *Preprocessor) Process(entries []config.LogEntry) (*CompressedOutput, error) {
	if len(entries) == 0 {
		return p.compressor.Compress(entries, nil, 0)
	}

	// Stage 1: Redaction
	redactedEntries, redactedCount := p.redactEntries(entries)

	// Stage 2: Drain Template Extraction
	templates := p.extractTemplates(redactedEntries)

	// Stage 3: Compression
	output, err := p.compressor.Compress(redactedEntries, templates, redactedCount)
	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	// Add debug info if enabled
	if p.debug {
		p.addDebugInfo(output, templates, redactedEntries)
	}

	return output, nil
}

// ProcessBatch processes multiple batches of log entries.
// This is useful when processing large files in chunks.
//
// Note: Template extraction happens across all batches, so patterns
// from earlier batches will be recognized in later batches.
func (p *Preprocessor) ProcessBatch(entries []config.LogEntry) (*CompressedOutput, error) {
	// For now, just process as a single batch
	// In the future, this could support incremental processing
	return p.Process(entries)
}

// Reset clears all state, including extracted templates and redaction mappings.
// Call this when switching to a completely different log file where
// correlations should not persist.
func (p *Preprocessor) Reset() {
	p.drain.Reset()
	p.redactor.Reset()
}

// GetRedactedValues returns a map of all redacted values and their placeholders.
// This is useful for debugging to see what was redacted.
func (p *Preprocessor) GetRedactedValues() map[string]string {
	return p.redactor.GetUniqueValues()
}

// GetTemplateCount returns the number of templates currently extracted.
func (p *Preprocessor) GetTemplateCount() int {
	return p.drain.GetTemplateCount()
}

// redactEntries applies redaction to all log entries.
// Returns the redacted entries and count of redacted values.
func (p *Preprocessor) redactEntries(entries []config.LogEntry) ([]config.LogEntry, int) {
	redactedEntries := make([]config.LogEntry, len(entries))
	totalRedacted := 0

	for i, entry := range entries {
		redactedEntries[i] = entry

		// Redact the message
		redactedMsg, count := p.redactor.RedactAndCount(entry.Message)
		redactedEntries[i].Message = redactedMsg
		totalRedacted += count

		// Also redact raw if it differs
		if entry.Raw != entry.Message {
			redactedRaw, count := p.redactor.RedactAndCount(entry.Raw)
			redactedEntries[i].Raw = redactedRaw
			totalRedacted += count
		}
	}

	return redactedEntries, totalRedacted
}

// extractTemplates extracts Drain templates from redacted entries.
func (p *Preprocessor) extractTemplates(entries []config.LogEntry) []*Template {
	for _, entry := range entries {
		p.drain.Extract(entry.Message)
	}

	return p.drain.GetTemplates()
}

// addDebugInfo adds debug information to the output.
func (p *Preprocessor) addDebugInfo(output *CompressedOutput, templates []*Template, entries []config.LogEntry) {
	if output.Metadata == nil {
		output.Metadata = make(map[string]interface{})
	}

	// Template lineage
	templateInfo := make([]map[string]interface{}, 0, len(templates))
	for _, t := range templates {
		info := map[string]interface{}{
			"id":      t.ID,
			"pattern": t.Pattern,
			"count":   t.Count,
		}
		templateInfo = append(templateInfo, info)
	}
	output.Metadata["templates_debug"] = templateInfo

	// Redaction stats
	redactedValues := p.redactor.GetUniqueValues()
	output.Metadata["unique_redacted_values"] = len(redactedValues)
	output.Metadata["redacted_types"] = p.countRedactedTypes(redactedValues)
}

// countRedactedTypes counts how many values were redacted by type.
func (p *Preprocessor) countRedactedTypes(redactedValues map[string]string) map[string]int {
	counts := make(map[string]int)
	for _, placeholder := range redactedValues {
		// Extract type from placeholder [TYPE:hash]
		if len(placeholder) > 2 {
			// Find the first colon
			for i := 1; i < len(placeholder)-1; i++ {
				if placeholder[i] == ':' {
					typ := placeholder[1:i]
					counts[typ]++
					break
				}
			}
		}
	}
	return counts
}

// ProcessWithStats processes entries and returns detailed statistics.
// This is useful for understanding the preprocessing results.
func (p *Preprocessor) ProcessWithStats(entries []config.LogEntry) (*CompressedOutput, *ProcessStats, error) {
	output, err := p.Process(entries)
	if err != nil {
		return nil, nil, err
	}

	stats := &ProcessStats{
		InputLines:       len(entries),
		OutputTemplates:  len(output.Templates),
		RedactedCount:    output.RedactedCount,
		TokenCount:       output.TokenCount,
		TokenLimit:       output.TokenLimit,
		CompressionRatio: output.GetCompressionRatio(),
		WithinBudget:     output.IsWithinBudget(),
	}

	// Count by severity
	for _, t := range output.Templates {
		switch t.Level {
		case config.LevelFatal:
			stats.FatalCount++
		case config.LevelError:
			stats.ErrorCount++
		case config.LevelWarn:
			stats.WarnCount++
		case config.LevelInfo:
			stats.InfoCount++
		case config.LevelDebug:
			stats.DebugCount++
		}
	}

	return output, stats, nil
}

// ProcessStats contains detailed statistics about the preprocessing run.
type ProcessStats struct {
	InputLines       int
	OutputTemplates  int
	RedactedCount    int
	TokenCount       int
	TokenLimit       int
	CompressionRatio float64
	WithinBudget     bool
	FatalCount       int
	ErrorCount       int
	WarnCount        int
	InfoCount        int
	DebugCount       int
}

// String returns a human-readable summary of the processing statistics.
func (s *ProcessStats) String() string {
	return fmt.Sprintf(
		"Processed %d lines into %d templates (%.1fx compression)\n"+
			"Redacted %d sensitive values\n"+
			"Token usage: %d/%d (%.1f%%)\n"+
			"Severity distribution: %d FATAL, %d ERROR, %d WARN, %d INFO, %d DEBUG",
		s.InputLines,
		s.OutputTemplates,
		s.CompressionRatio,
		s.RedactedCount,
		s.TokenCount,
		s.TokenLimit,
		float64(s.TokenCount)/float64(s.TokenLimit)*100,
		s.FatalCount,
		s.ErrorCount,
		s.WarnCount,
		s.InfoCount,
		s.DebugCount,
	)
}

// QuickCompress is a convenience function for one-off preprocessing without
// creating a Preprocessor instance.
//
// Example:
//
//	output, err := preprocess.QuickCompress(logEntries, 8000, true)
func QuickCompress(entries []config.LogEntry, tokenLimit int, redaction bool) (*CompressedOutput, error) {
	preprocessor := New(
		WithTokenLimit(tokenLimit),
		WithRedaction(redaction),
	)
	return preprocessor.Process(entries)
}

// QuickCompressWithDefaults is a convenience function using default settings.
// Token limit: 8000, Redaction: enabled
func QuickCompressWithDefaults(entries []config.LogEntry) (*CompressedOutput, error) {
	return QuickCompress(entries, DefaultTokenLimit, true)
}
