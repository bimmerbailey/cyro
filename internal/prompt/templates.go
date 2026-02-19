package prompt

import (
	"fmt"
	"strings"

	"github.com/bimmerbailey/cyro/internal/llm"
)

// Build constructs a []llm.Message slice ready to be sent to any llm.Provider.
//
// The returned slice always begins with a system message whose content is
// determined by pt, followed by one or more user/assistant messages that
// encode the log context and task instruction.
//
// Required fields per PromptType:
//   - All types:                Summary must be non-empty
//   - TypeNaturalLanguageQuery: Question must be non-empty
//   - TypeStructuredOutput:     Summary must be non-empty; FirstPassResponse
//     selects which pass is built (empty = first pass, non-empty = second pass)
//
// Returns ErrMissingField if a required field is absent.
func Build(pt PromptType, opts BuildOptions) ([]llm.Message, error) {
	if opts.Summary == "" {
		return nil, missingField("Summary")
	}

	switch pt {
	case TypeNaturalLanguageQuery:
		return buildNaturalLanguageQuery(opts)
	case TypeStructuredOutput:
		return buildStructuredOutput(opts)
	default:
		return buildStandard(pt, opts)
	}
}

// buildStandard handles TypeSummarize, TypeRootCause, and TypeAnomalyDetection.
// All three share the same two-message structure: system + user.
func buildStandard(pt PromptType, opts BuildOptions) ([]llm.Message, error) {
	return []llm.Message{
		{Role: "system", Content: systemPrompt(pt)},
		{Role: "user", Content: buildStandardUserMessage(pt, opts)},
	}, nil
}

// buildStandardUserMessage assembles the user-turn content for standard types.
func buildStandardUserMessage(pt PromptType, opts BuildOptions) string {
	var sb strings.Builder

	// Task instruction varies by type
	switch pt {
	case TypeRootCause:
		sb.WriteString("Perform a root cause analysis on the following log summary:\n\n")
	case TypeAnomalyDetection:
		sb.WriteString("Identify anomalies in the following log summary:\n\n")
	default: // TypeSummarize
		sb.WriteString("Analyze the following log summary:\n\n")
	}

	appendLogContext(&sb, opts)
	appendFilterNotes(&sb, opts)

	return sb.String()
}

// buildNaturalLanguageQuery builds messages for TypeNaturalLanguageQuery.
// Requires opts.Question to be non-empty.
func buildNaturalLanguageQuery(opts BuildOptions) ([]llm.Message, error) {
	if opts.Question == "" {
		return nil, missingField("Question")
	}

	var sb strings.Builder
	sb.WriteString("Question: ")
	sb.WriteString(opts.Question)
	sb.WriteString("\n\n")
	sb.WriteString("Log Summary:\n")
	sb.WriteString(opts.Summary)

	appendFilterNotes(&sb, opts)

	return []llm.Message{
		{Role: "system", Content: systemPrompt(TypeNaturalLanguageQuery)},
		{Role: "user", Content: sb.String()},
	}, nil
}

// buildStructuredOutput implements the two-pass pattern for TypeStructuredOutput.
//
// First pass (opts.FirstPassResponse == ""):
//
//	[system, user(analysis request)]
//
// Second pass (opts.FirstPassResponse != ""):
//
//	[system, user(analysis request), assistant(first pass text), user(JSON extraction)]
//
// The caller is responsible for collecting the first-pass response and
// providing it as opts.FirstPassResponse for the second call.
func buildStructuredOutput(opts BuildOptions) ([]llm.Message, error) {
	// First-pass user message: same analysis instruction as TypeSummarize
	var firstUserSB strings.Builder
	firstUserSB.WriteString("Analyze the following log summary:\n\n")
	appendLogContext(&firstUserSB, opts)
	appendFilterNotes(&firstUserSB, opts)

	firstUser := llm.Message{Role: "user", Content: firstUserSB.String()}
	system := llm.Message{Role: "system", Content: systemPrompt(TypeStructuredOutput)}

	if opts.FirstPassResponse == "" {
		// First pass: ask for free-form analysis
		return []llm.Message{system, firstUser}, nil
	}

	// Second pass: prefill the assistant turn, then request JSON extraction
	extractInstruction := "Now extract your analysis into the JSON schema specified in the system prompt. " +
		"Output ONLY the JSON object — no markdown, no explanation."

	return []llm.Message{
		system,
		firstUser,
		{Role: "assistant", Content: opts.FirstPassResponse},
		{Role: "user", Content: extractInstruction},
	}, nil
}

// appendLogContext writes the compressed log summary and optional metadata
// (time range, file list) into sb.
func appendLogContext(sb *strings.Builder, opts BuildOptions) {
	if opts.TimeRange != "" {
		sb.WriteString(fmt.Sprintf("Time range: %s\n\n", opts.TimeRange))
	}

	if len(opts.Files) == 1 {
		sb.WriteString(fmt.Sprintf("Source file: %s\n\n", opts.Files[0]))
	} else if len(opts.Files) > 1 {
		sb.WriteString(fmt.Sprintf("Source files (%d): %s\n\n",
			len(opts.Files), strings.Join(opts.Files, ", ")))
	}

	sb.WriteString(opts.Summary)
	sb.WriteString("\n\n")
}

// appendFilterNotes appends human-readable notes about any filters that were
// applied before compression, so the model knows the data was pre-filtered.
func appendFilterNotes(sb *strings.Builder, opts BuildOptions) {
	var notes []string

	if opts.Pattern != "" {
		notes = append(notes, fmt.Sprintf("Filtered by pattern: %s", opts.Pattern))
	}
	if opts.Level != "" {
		notes = append(notes, fmt.Sprintf("Filtered by level: %s", opts.Level))
	}
	if opts.GroupBy != "" {
		notes = append(notes, fmt.Sprintf("Analysis grouped by: %s", opts.GroupBy))
	}
	if opts.Window != "" {
		notes = append(notes, fmt.Sprintf("Time window applied: %s", opts.Window))
	}

	if len(notes) > 0 {
		sb.WriteString("Note: ")
		sb.WriteString(strings.Join(notes, "; "))
		sb.WriteString(".\n")
	}
}
