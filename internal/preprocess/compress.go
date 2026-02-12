package preprocess

import (
	"fmt"
	"strings"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
)

// DefaultTokenLimit is the default maximum tokens for LLM input.
const DefaultTokenLimit = 8000

// Token estimation: roughly 1 token per 4 characters for English text.
const charsPerToken = 4

// CompressedOutput represents the final compressed log summary ready for LLM consumption.
type CompressedOutput struct {
	Summary        string                 // Human-readable summary
	TimeRange      TimeRange              // First and last timestamps
	TotalLines     int                    // Original line count
	TotalTemplates int                    // Number of unique templates
	Templates      []TemplateSummary      // Extracted patterns with frequencies
	RedactedCount  int                    // Number of sensitive values redacted
	Metadata       map[string]interface{} // Additional context
	TokenCount     int                    // Estimated token count
	TokenLimit     int                    // Maximum allowed tokens
}

// TimeRange represents the time span covered by the logs.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// TemplateSummary represents a template in the compressed output.
type TemplateSummary struct {
	Pattern   string          // Template pattern with wildcards
	Count     int             // Frequency
	Level     config.LogLevel // Highest severity level for this template
	Examples  []string        // Sample messages
	FirstSeen time.Time       // First occurrence timestamp
	LastSeen  time.Time       // Last occurrence timestamp
}

// Compressor handles the compression and formatting of preprocessed logs.
type Compressor struct {
	tokenLimit int
}

// NewCompressor creates a new compressor with the specified token limit.
// Use 0 or negative for default limit (8000 tokens).
func NewCompressor(tokenLimit int) *Compressor {
	if tokenLimit <= 0 {
		tokenLimit = DefaultTokenLimit
	}
	return &Compressor{
		tokenLimit: tokenLimit,
	}
}

// Compress processes Drain templates and creates a compressed output
// that fits within the token budget.
func (c *Compressor) Compress(
	entries []config.LogEntry,
	templates []*Template,
	redactedCount int,
) (*CompressedOutput, error) {
	if len(entries) == 0 {
		return &CompressedOutput{
			Summary:    "No log entries to analyze.",
			TimeRange:  TimeRange{},
			TotalLines: 0,
			Templates:  []TemplateSummary{},
			TokenCount: 0,
			TokenLimit: c.tokenLimit,
		}, nil
	}

	// Calculate time range
	timeRange := c.calculateTimeRange(entries)

	// Convert Drain templates to template summaries with severity info
	templateSummaries := c.createTemplateSummaries(entries, templates)

	// Sort templates by priority (errors first, then by frequency)
	templateSummaries = c.prioritizeTemplates(templateSummaries)

	// Build output respecting token limit
	output := c.buildOutput(
		entries,
		templateSummaries,
		timeRange,
		redactedCount,
	)

	return output, nil
}

// calculateTimeRange finds the first and last timestamps in the entries.
func (c *Compressor) calculateTimeRange(entries []config.LogEntry) TimeRange {
	var start, end time.Time

	for _, entry := range entries {
		if entry.Timestamp.IsZero() {
			continue
		}
		if start.IsZero() || entry.Timestamp.Before(start) {
			start = entry.Timestamp
		}
		if end.IsZero() || entry.Timestamp.After(end) {
			end = entry.Timestamp
		}
	}

	return TimeRange{Start: start, End: end}
}

// createTemplateSummaries enhances Drain templates with severity and timing info.
func (c *Compressor) createTemplateSummaries(
	entries []config.LogEntry,
	drainTemplates []*Template,
) []TemplateSummary {
	// Build a map of template ID -> entries matching it
	templateEntries := make(map[string][]config.LogEntry)

	// Create a simple matcher for Drain templates
	for _, entry := range entries {
		for _, template := range drainTemplates {
			if c.matchesTemplate(entry.Message, template) {
				templateEntries[template.ID] = append(templateEntries[template.ID], entry)
				break
			}
		}
	}

	summaries := make([]TemplateSummary, 0, len(drainTemplates))
	for _, template := range drainTemplates {
		entries := templateEntries[template.ID]
		if len(entries) == 0 {
			continue
		}

		summary := TemplateSummary{
			Pattern:  template.Pattern,
			Count:    template.Count,
			Examples: template.Examples,
		}

		// Find highest severity level and time range
		maxLevel := config.LevelUnknown
		var firstSeen, lastSeen time.Time
		for _, entry := range entries {
			if entry.Level > maxLevel {
				maxLevel = entry.Level
			}
			if !entry.Timestamp.IsZero() {
				if firstSeen.IsZero() || entry.Timestamp.Before(firstSeen) {
					firstSeen = entry.Timestamp
				}
				if lastSeen.IsZero() || entry.Timestamp.After(lastSeen) {
					lastSeen = entry.Timestamp
				}
			}
		}
		summary.Level = maxLevel
		summary.FirstSeen = firstSeen
		summary.LastSeen = lastSeen

		summaries = append(summaries, summary)
	}

	return summaries
}

// matchesTemplate checks if a message matches a Drain template pattern.
func (c *Compressor) matchesTemplate(message string, template *Template) bool {
	// Simple matching: check if message could match the template pattern
	// This is a simplified version - in practice, we'd need proper tokenization
	messageTokens := strings.Fields(message)
	templateTokens := strings.Fields(template.Pattern)

	if len(messageTokens) != len(templateTokens) {
		return false
	}

	for i, templateToken := range templateTokens {
		if templateToken == "<*>" {
			continue // Wildcard matches anything
		}
		if templateToken != messageTokens[i] {
			return false
		}
	}

	return true
}

// prioritizeTemplates sorts templates by severity and frequency.
// Errors come first, then warnings, then by frequency.
func (c *Compressor) prioritizeTemplates(templates []TemplateSummary) []TemplateSummary {
	// Simple bubble sort for clarity
	for i := 0; i < len(templates)-1; i++ {
		for j := i + 1; j < len(templates); j++ {
			scoreI := c.templatePriorityScore(templates[i])
			scoreJ := c.templatePriorityScore(templates[j])
			if scoreJ > scoreI {
				templates[i], templates[j] = templates[j], templates[i]
			}
		}
	}
	return templates
}

// templatePriorityScore calculates a priority score for sorting.
// Higher score = higher priority.
func (c *Compressor) templatePriorityScore(t TemplateSummary) float64 {
	// Severity weight: FATAL=5, ERROR=4, WARN=3, INFO=2, DEBUG=1, UNKNOWN=0
	severityWeight := float64(t.Level) * 1000
	// Frequency weight: log(count + 1) to prevent outliers from dominating
	freqWeight := float64(t.Count)
	return severityWeight + freqWeight
}

// buildOutput constructs the final compressed output respecting token limit.
func (c *Compressor) buildOutput(
	entries []config.LogEntry,
	templates []TemplateSummary,
	timeRange TimeRange,
	redactedCount int,
) *CompressedOutput {
	output := &CompressedOutput{
		TimeRange:      timeRange,
		TotalLines:     len(entries),
		TotalTemplates: len(templates),
		Templates:      []TemplateSummary{},
		RedactedCount:  redactedCount,
		Metadata:       make(map[string]interface{}),
		TokenLimit:     c.tokenLimit,
	}

	// Start building the summary
	var sb strings.Builder

	// Header section (always included)
	c.writeHeader(&sb, output)

	// Template section (respect token limit)
	includedTemplates := c.writeTemplates(&sb, templates, c.tokenLimit)
	output.Templates = includedTemplates

	// Footer with statistics
	c.writeFooter(&sb, output)

	output.Summary = sb.String()
	output.TokenCount = estimateTokens(output.Summary)

	return output
}

// writeHeader writes the summary header.
func (c *Compressor) writeHeader(sb *strings.Builder, output *CompressedOutput) {
	sb.WriteString("=== Log Analysis Summary ===\n\n")

	if !output.TimeRange.Start.IsZero() {
		sb.WriteString(fmt.Sprintf("Time Range: %s to %s\n",
			output.TimeRange.Start.Format(time.RFC3339),
			output.TimeRange.End.Format(time.RFC3339)))
	}

	sb.WriteString(fmt.Sprintf("Total Lines: %d\n", output.TotalLines))
	sb.WriteString(fmt.Sprintf("Unique Patterns: %d\n", output.TotalTemplates))
	if output.RedactedCount > 0 {
		sb.WriteString(fmt.Sprintf("Sensitive Values Redacted: %d\n", output.RedactedCount))
	}
	sb.WriteString("\n")
}

// writeTemplates writes templates respecting the token budget.
func (c *Compressor) writeTemplates(
	sb *strings.Builder,
	templates []TemplateSummary,
	tokenLimit int,
) []TemplateSummary {
	var included []TemplateSummary
	currentTokens := estimateTokens(sb.String())

	// Reserve tokens for header/footer (estimated)
	reservedTokens := 200
	availableTokens := tokenLimit - reservedTokens

	// Separate by severity
	errors := []TemplateSummary{}
	warnings := []TemplateSummary{}
	others := []TemplateSummary{}

	for _, t := range templates {
		switch t.Level {
		case config.LevelFatal, config.LevelError:
			errors = append(errors, t)
		case config.LevelWarn:
			warnings = append(warnings, t)
		default:
			others = append(others, t)
		}
	}

	// Always try to include all errors
	if len(errors) > 0 {
		sb.WriteString("=== Error Summary ===\n")
		for _, t := range errors {
			templateStr := c.formatTemplate(t)
			templateTokens := estimateTokens(templateStr)

			if currentTokens+templateTokens > availableTokens {
				break
			}

			sb.WriteString(templateStr)
			currentTokens += templateTokens
			included = append(included, t)
		}
		sb.WriteString("\n")
	}

	// Include warnings if space permits
	if len(warnings) > 0 && currentTokens < availableTokens {
		sb.WriteString("=== Warning Summary ===\n")
		for _, t := range warnings {
			templateStr := c.formatTemplate(t)
			templateTokens := estimateTokens(templateStr)

			if currentTokens+templateTokens > availableTokens {
				break
			}

			sb.WriteString(templateStr)
			currentTokens += templateTokens
			included = append(included, t)
		}
		sb.WriteString("\n")
	}

	// Include other templates (INFO, DEBUG) if space permits
	if len(others) > 0 && currentTokens < availableTokens {
		sb.WriteString("=== Top Info Patterns ===\n")
		for _, t := range others {
			templateStr := c.formatTemplate(t)
			templateTokens := estimateTokens(templateStr)

			if currentTokens+templateTokens > availableTokens {
				break
			}

			sb.WriteString(templateStr)
			currentTokens += templateTokens
			included = append(included, t)
		}
		sb.WriteString("\n")
	}

	return included
}

// formatTemplate formats a single template for output.
func (c *Compressor) formatTemplate(t TemplateSummary) string {
	var sb strings.Builder

	// Severity prefix
	levelStr := t.Level.String()
	sb.WriteString(fmt.Sprintf("[%s] %s (%d occurrences)\n", levelStr, t.Pattern, t.Count))

	// Examples (if space permits and we have them)
	if len(t.Examples) > 0 {
		sb.WriteString("  Examples:\n")
		for _, ex := range t.Examples {
			// Truncate long examples
			if len(ex) > 120 {
				ex = ex[:117] + "..."
			}
			sb.WriteString(fmt.Sprintf("    - %s\n", ex))
		}
	}

	return sb.String()
}

// writeFooter writes the summary footer with statistics.
func (c *Compressor) writeFooter(sb *strings.Builder, output *CompressedOutput) {
	output.Metadata["included_templates"] = len(output.Templates)
	output.Metadata["compression_ratio"] = float64(output.TotalLines) / float64(len(output.Templates)+1)

	sb.WriteString(fmt.Sprintf("Token Count: ~%d / %d\n", output.TokenCount, output.TokenLimit))
}

// estimateTokens provides a rough estimate of token count.
// Assumes ~1 token per 4 characters for English text.
func estimateTokens(text string) int {
	return len(text) / charsPerToken
}

// GetCompressionRatio calculates the compression achieved.
func (c *CompressedOutput) GetCompressionRatio() float64 {
	if c.TotalTemplates == 0 {
		return 1.0
	}
	return float64(c.TotalLines) / float64(c.TotalTemplates)
}

// IsWithinBudget returns true if the output is within the token limit.
func (c *CompressedOutput) IsWithinBudget() bool {
	return c.TokenCount <= c.TokenLimit
}

// GetTemplatesByLevel returns templates filtered by severity level.
func (c *CompressedOutput) GetTemplatesByLevel(level config.LogLevel) []TemplateSummary {
	var result []TemplateSummary
	for _, t := range c.Templates {
		if t.Level == level {
			result = append(result, t)
		}
	}
	return result
}
