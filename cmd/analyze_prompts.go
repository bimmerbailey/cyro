package cmd

import (
	"fmt"
	"strings"

	"github.com/bimmerbailey/cyro/internal/preprocess"
)

// buildAnalysisSystemPrompt creates the system prompt for log analysis.
func buildAnalysisSystemPrompt() string {
	return `You are an expert log analysis assistant. Your role is to analyze log data and provide clear, actionable insights.

Guidelines:
1. Only reference information present in the provided log summary
2. Distinguish observations ("the logs show...") from inferences ("this suggests...")
3. Never invent or hallucinate log entries
4. Focus on patterns, root causes, and actionable recommendations
5. Use specific timestamps and error messages when available
6. Structure your response clearly with sections

Your analysis should include:
- Summary: High-level overview of what the logs show
- Key Findings: Most important patterns or issues
- Timeline: When issues occurred (if timestamps available)
- Root Cause: Why issues happened (evidence-based)
- Recommendations: What to investigate or fix next`
}

// buildAnalysisUserPrompt creates the user prompt with the compressed log summary.
func buildAnalysisUserPrompt(output *preprocess.CompressedOutput, context AnalysisContext) string {
	var sb strings.Builder

	sb.WriteString("Analyze the following log summary:\n\n")
	sb.WriteString(output.Summary)
	sb.WriteString("\n\n")

	// Add context about filters applied
	if context.Pattern != "" {
		sb.WriteString(fmt.Sprintf("Note: Logs were filtered by pattern: %s\n", context.Pattern))
	}
	if context.GroupBy != "" {
		sb.WriteString(fmt.Sprintf("Note: Analysis focused on grouping by: %s\n", context.GroupBy))
	}
	if context.Window != "" {
		sb.WriteString(fmt.Sprintf("Note: Time window applied: %s\n", context.Window))
	}

	return sb.String()
}

// AnalysisContext provides metadata about the analysis request.
type AnalysisContext struct {
	Pattern string
	GroupBy string
	Window  string
	Files   []string
}
