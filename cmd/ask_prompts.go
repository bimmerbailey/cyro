package cmd

import (
	"fmt"
	"strings"

	"github.com/bimmerbailey/cyro/internal/preprocess"
)

// buildAskSystemPrompt creates the system prompt for the ask command.
// The prompt is designed to let the LLM respond naturally to the user's
// specific question, rather than forcing a structured format.
func buildAskSystemPrompt() string {
	return `You are a helpful log analysis assistant. Your role is to answer questions
about log data based on the provided context. Answer the user's specific question
accurately and directly using only the information present in the log summary.

Guidelines:
- Focus on answering the user's specific question
- Reference timestamps and specific log entries when relevant
- Distinguish observations from inferences
- Never invent or hallucinate log entries that aren't in the provided data
- Provide actionable insights when appropriate
- Be concise but thorough - match the level of detail the user is asking for`
}

// buildAskUserPrompt creates the user prompt combining the question and compressed log data.
func buildAskUserPrompt(question string, output *preprocess.CompressedOutput) string {
	var sb strings.Builder

	sb.WriteString("Question: ")
	sb.WriteString(question)
	sb.WriteString("\n\n")
	sb.WriteString("Log Summary:\n")
	sb.WriteString(output.Summary)

	return sb.String()
}

// buildAskContext provides additional context about the log source.
// This can be included in the prompt if needed for better answers.
func buildAskContext(files []string, pattern string, level string, timeRange string) string {
	var sb strings.Builder

	sb.WriteString("Context:\n")

	if len(files) > 1 {
		sb.WriteString(fmt.Sprintf("- Analyzing %d files: %s\n", len(files), strings.Join(files, ", ")))
	} else if len(files) == 1 {
		sb.WriteString(fmt.Sprintf("- Analyzing file: %s\n", files[0]))
	}

	if pattern != "" {
		sb.WriteString(fmt.Sprintf("- Filtered by pattern: %s\n", pattern))
	}

	if level != "" {
		sb.WriteString(fmt.Sprintf("- Filtered by level: %s\n", level))
	}

	if timeRange != "" {
		sb.WriteString(fmt.Sprintf("- Time range: %s\n", timeRange))
	}

	return sb.String()
}
