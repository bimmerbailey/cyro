package cmd

import (
	"strings"
	"testing"

	"github.com/bimmerbailey/cyro/internal/preprocess"
)

func TestBuildAskSystemPrompt(t *testing.T) {
	prompt := buildAskSystemPrompt()

	// Check that the prompt contains expected content
	if !strings.Contains(prompt, "log analysis assistant") {
		t.Error("system prompt should identify as log analysis assistant")
	}

	if !strings.Contains(prompt, "Answer the user's specific question") {
		t.Error("system prompt should mention answering user's question")
	}

	if !strings.Contains(prompt, "Never invent or hallucinate") {
		t.Error("system prompt should warn against hallucination")
	}
}

func TestBuildAskUserPrompt(t *testing.T) {
	question := "what caused the error?"
	output := &preprocess.CompressedOutput{
		Summary: "Error occurred at 14:32",
	}

	prompt := buildAskUserPrompt(question, output)

	// Check that prompt contains the question
	if !strings.Contains(prompt, "Question: ") {
		t.Error("prompt should contain 'Question:' prefix")
	}
	if !strings.Contains(prompt, question) {
		t.Error("prompt should contain the actual question")
	}

	// Check that prompt contains the log summary
	if !strings.Contains(prompt, "Log Summary:") {
		t.Error("prompt should contain 'Log Summary:' header")
	}
	if !strings.Contains(prompt, output.Summary) {
		t.Error("prompt should contain the log summary content")
	}
}

func TestBuildAskUserPromptWithEmptySummary(t *testing.T) {
	question := "what happened?"
	output := &preprocess.CompressedOutput{
		Summary: "",
	}

	prompt := buildAskUserPrompt(question, output)

	if !strings.Contains(prompt, question) {
		t.Error("prompt should still contain the question even with empty summary")
	}
}

func TestBuildAskContext(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		pattern   string
		level     string
		timeRange string
		want      []string
		notWant   []string
	}{
		{
			name:      "single file",
			files:     []string{"app.log"},
			pattern:   "",
			level:     "",
			timeRange: "",
			want:      []string{"app.log"},
			notWant:   []string{"files:"},
		},
		{
			name:      "multiple files",
			files:     []string{"app.log", "api.log"},
			pattern:   "",
			level:     "",
			timeRange: "",
			want:      []string{"2 files", "app.log", "api.log"},
			notWant:   []string{},
		},
		{
			name:      "with pattern filter",
			files:     []string{"app.log"},
			pattern:   "error",
			level:     "",
			timeRange: "",
			want:      []string{"error", "pattern:"},
			notWant:   []string{},
		},
		{
			name:      "with level filter",
			files:     []string{"app.log"},
			pattern:   "",
			level:     "error",
			timeRange: "",
			want:      []string{"error", "level:"},
			notWant:   []string{},
		},
		{
			name:      "with time range",
			files:     []string{"app.log"},
			pattern:   "",
			level:     "",
			timeRange: "last 1h",
			want:      []string{"last 1h", "Time range:"},
			notWant:   []string{},
		},
		{
			name:      "with all filters",
			files:     []string{"app.log"},
			pattern:   "timeout",
			level:     "error",
			timeRange: "1h",
			want:      []string{"timeout", "error", "1h"},
			notWant:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := buildAskContext(tt.files, tt.pattern, tt.level, tt.timeRange)

			for _, want := range tt.want {
				if !strings.Contains(ctx, want) {
					t.Errorf("context should contain %q, got:\n%s", want, ctx)
				}
			}

			for _, notWant := range tt.notWant {
				if strings.Contains(ctx, notWant) {
					t.Errorf("context should NOT contain %q", notWant)
				}
			}
		})
	}
}

func TestBuildAskContextWithNoContext(t *testing.T) {
	// When no context is provided, should still include the header
	ctx := buildAskContext([]string{"app.log"}, "", "", "")

	if !strings.Contains(ctx, "Context:") {
		t.Error("context should contain 'Context:' header")
	}
	if !strings.Contains(ctx, "app.log") {
		t.Error("context should contain file name")
	}
}
