package prompt

import (
	"errors"
	"fmt"
)

// PromptType identifies the analysis task a prompt is designed to perform.
// Each type produces a distinct system persona and user message structure.
type PromptType string

const (
	// TypeSummarize produces a high-level narrative summary of the logs.
	// It is the default mode used by `cyro analyze --ai`.
	TypeSummarize PromptType = "summarize"

	// TypeRootCause instructs the model to diagnose the root cause of errors
	// found in the log data, following an evidence-based chain of reasoning.
	TypeRootCause PromptType = "root_cause"

	// TypeAnomalyDetection instructs the model to identify patterns that deviate
	// from normal behaviour — frequency spikes, new error classes, unexpected
	// sequences — and classify their severity.
	TypeAnomalyDetection PromptType = "anomaly_detection"

	// TypeNaturalLanguageQuery instructs the model to answer a specific user
	// question about the log data. Used by `cyro ask`.
	TypeNaturalLanguageQuery PromptType = "natural_language_query"

	// TypeStructuredOutput implements a two-pass pattern for reliable JSON
	// extraction from small models. On the first pass (FirstPassResponse == "")
	// it requests a free-form analysis. On the second pass
	// (FirstPassResponse set) it prefills the assistant turn and appends a
	// JSON extraction instruction.
	TypeStructuredOutput PromptType = "structured_output"
)

// BuildOptions holds all contextual information required to build a prompt.
// Not all fields are required for every [PromptType]; see the documentation
// for each type to understand which fields are used.
type BuildOptions struct {
	// Summary is the compressed log text produced by internal/preprocess.
	// Required for all prompt types.
	Summary string

	// Question is the user's natural language question.
	// Required for [TypeNaturalLanguageQuery].
	Question string

	// Files is the list of log file paths being analysed.
	// Optional: included as context when non-empty.
	Files []string

	// Pattern is the regex used to pre-filter the log entries.
	// Optional: appended as a context note when non-empty.
	Pattern string

	// Level is the log level filter applied before compression.
	// Optional: appended as a context note when non-empty.
	Level string

	// GroupBy is the field used for grouping in statistical analysis.
	// Optional: appended as a context note when non-empty.
	GroupBy string

	// Window is the time window applied during trend analysis (e.g. "5m").
	// Optional: appended as a context note when non-empty.
	Window string

	// TimeRange is a human-readable description of the log time span
	// (e.g. "2024-01-01 14:00 → 2024-01-01 15:00").
	// Optional: included in the header when non-empty.
	TimeRange string

	// FirstPassResponse is used only with [TypeStructuredOutput].
	// When empty, Build returns the first-pass messages (free-form analysis
	// request). When set to the model's first-pass reply, Build returns the
	// second-pass messages (assistant prefill + JSON extraction instruction).
	FirstPassResponse string
}

// ErrMissingField is returned by [Build] when a required field for the
// requested [PromptType] is absent from [BuildOptions].
var ErrMissingField = errors.New("prompt: missing required field")

// missingField wraps [ErrMissingField] with the specific field name.
func missingField(field string) error {
	return fmt.Errorf("%w: %s", ErrMissingField, field)
}
