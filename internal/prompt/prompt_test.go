package prompt_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/bimmerbailey/cyro/internal/llm"
	"github.com/bimmerbailey/cyro/internal/prompt"
)

const testSummary = `=== Error Summary ===
[ERROR] database connection refused (42 occurrences)
  Examples:
    - dial tcp 127.0.0.1:5432: connect: connection refused

=== Warning Summary ===
[WARN] retry attempt <*> of <*> (18 occurrences)
  Examples:
    - retry attempt 1 of 3`

// TestBuild_RequiresSummary verifies that ErrMissingField is returned when
// Summary is empty, for every PromptType.
func TestBuild_RequiresSummary(t *testing.T) {
	types := []prompt.PromptType{
		prompt.TypeSummarize,
		prompt.TypeRootCause,
		prompt.TypeAnomalyDetection,
		prompt.TypeNaturalLanguageQuery,
		prompt.TypeStructuredOutput,
	}

	for _, pt := range types {
		t.Run(string(pt), func(t *testing.T) {
			opts := prompt.BuildOptions{Question: "does not matter"}
			_, err := prompt.Build(pt, opts)
			if !errors.Is(err, prompt.ErrMissingField) {
				t.Errorf("expected ErrMissingField, got %v", err)
			}
		})
	}
}

// TestBuild_NaturalLanguageQuery_RequiresQuestion ensures Question is enforced.
func TestBuild_NaturalLanguageQuery_RequiresQuestion(t *testing.T) {
	opts := prompt.BuildOptions{Summary: testSummary}
	_, err := prompt.Build(prompt.TypeNaturalLanguageQuery, opts)
	if !errors.Is(err, prompt.ErrMissingField) {
		t.Errorf("expected ErrMissingField for missing Question, got %v", err)
	}
}

// TestBuild_MessageStructure verifies message count and roles for each type.
func TestBuild_MessageStructure(t *testing.T) {
	tests := []struct {
		name         string
		pt           prompt.PromptType
		opts         prompt.BuildOptions
		wantMsgCount int
		wantRoles    []string
	}{
		{
			name:         "summarize",
			pt:           prompt.TypeSummarize,
			opts:         prompt.BuildOptions{Summary: testSummary},
			wantMsgCount: 2,
			wantRoles:    []string{"system", "user"},
		},
		{
			name:         "root_cause",
			pt:           prompt.TypeRootCause,
			opts:         prompt.BuildOptions{Summary: testSummary},
			wantMsgCount: 2,
			wantRoles:    []string{"system", "user"},
		},
		{
			name:         "anomaly_detection",
			pt:           prompt.TypeAnomalyDetection,
			opts:         prompt.BuildOptions{Summary: testSummary},
			wantMsgCount: 2,
			wantRoles:    []string{"system", "user"},
		},
		{
			name: "natural_language_query",
			pt:   prompt.TypeNaturalLanguageQuery,
			opts: prompt.BuildOptions{
				Summary:  testSummary,
				Question: "why did the database fail?",
			},
			wantMsgCount: 2,
			wantRoles:    []string{"system", "user"},
		},
		{
			name:         "structured_output_first_pass",
			pt:           prompt.TypeStructuredOutput,
			opts:         prompt.BuildOptions{Summary: testSummary},
			wantMsgCount: 2,
			wantRoles:    []string{"system", "user"},
		},
		{
			name: "structured_output_second_pass",
			pt:   prompt.TypeStructuredOutput,
			opts: prompt.BuildOptions{
				Summary:           testSummary,
				FirstPassResponse: "The logs show 42 connection refused errors...",
			},
			wantMsgCount: 4,
			wantRoles:    []string{"system", "user", "assistant", "user"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgs, err := prompt.Build(tc.pt, tc.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(msgs) != tc.wantMsgCount {
				t.Errorf("message count: got %d, want %d", len(msgs), tc.wantMsgCount)
			}

			for i, role := range tc.wantRoles {
				if i >= len(msgs) {
					break
				}
				if msgs[i].Role != role {
					t.Errorf("msgs[%d].Role = %q, want %q", i, msgs[i].Role, role)
				}
			}
		})
	}
}

// TestBuild_SystemPromptDistinctPerType checks that different PromptTypes
// produce different system prompts (no copy-paste accident).
func TestBuild_SystemPromptDistinctPerType(t *testing.T) {
	types := []prompt.PromptType{
		prompt.TypeSummarize,
		prompt.TypeRootCause,
		prompt.TypeAnomalyDetection,
		prompt.TypeNaturalLanguageQuery,
		prompt.TypeStructuredOutput,
	}

	seen := make(map[string]prompt.PromptType)
	for _, pt := range types {
		opts := prompt.BuildOptions{
			Summary:  testSummary,
			Question: "q", // satisfy NLQ requirement
		}
		msgs, err := prompt.Build(pt, opts)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", pt, err)
		}

		systemContent := msgs[0].Content
		if prior, exists := seen[systemContent]; exists {
			t.Errorf("prompt type %q has identical system prompt to %q", pt, prior)
		}
		seen[systemContent] = pt
	}
}

// TestBuild_UserPromptContainsSummary verifies the compressed summary appears
// in the user message for all types.
func TestBuild_UserPromptContainsSummary(t *testing.T) {
	summarySnippet := "database connection refused"
	opts := prompt.BuildOptions{
		Summary:  testSummary,
		Question: "test question",
	}

	types := []prompt.PromptType{
		prompt.TypeSummarize,
		prompt.TypeRootCause,
		prompt.TypeAnomalyDetection,
		prompt.TypeNaturalLanguageQuery,
		prompt.TypeStructuredOutput,
	}

	for _, pt := range types {
		t.Run(string(pt), func(t *testing.T) {
			msgs, err := prompt.Build(pt, opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Find the first user message
			var userMsg *llm.Message
			for i := range msgs {
				if msgs[i].Role == "user" {
					userMsg = &msgs[i]
					break
				}
			}
			if userMsg == nil {
				t.Fatal("no user message found")
			}

			if !strings.Contains(userMsg.Content, summarySnippet) {
				t.Errorf("user message does not contain summary snippet %q", summarySnippet)
			}
		})
	}
}

// TestBuild_FilterNotes verifies that optional filter fields appear in the
// user message when set.
func TestBuild_FilterNotes(t *testing.T) {
	opts := prompt.BuildOptions{
		Summary:   testSummary,
		Pattern:   "error|fail",
		Level:     "error",
		GroupBy:   "message",
		Window:    "5m",
		TimeRange: "2024-01-01 14:00 → 2024-01-01 15:00",
		Files:     []string{"/var/log/app.log"},
	}

	msgs, err := prompt.Build(prompt.TypeSummarize, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userContent := msgs[1].Content

	checks := []string{
		"error|fail",
		"error",
		"message",
		"5m",
		"2024-01-01 14:00",
		"/var/log/app.log",
	}
	for _, needle := range checks {
		if !strings.Contains(userContent, needle) {
			t.Errorf("user message does not contain %q\ncontent:\n%s", needle, userContent)
		}
	}
}

// TestBuild_MultipleFiles verifies that multi-file context is included.
func TestBuild_MultipleFiles(t *testing.T) {
	opts := prompt.BuildOptions{
		Summary: testSummary,
		Files:   []string{"app.log", "auth.log", "db.log"},
	}

	msgs, err := prompt.Build(prompt.TypeSummarize, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userContent := msgs[1].Content
	if !strings.Contains(userContent, "3") {
		t.Errorf("expected file count (3) in user message, got:\n%s", userContent)
	}
	for _, f := range opts.Files {
		if !strings.Contains(userContent, f) {
			t.Errorf("expected file %q in user message", f)
		}
	}
}

// TestBuild_NaturalLanguageQuery_QuestionPlacement verifies the question
// appears before the log summary in the user message.
func TestBuild_NaturalLanguageQuery_QuestionPlacement(t *testing.T) {
	q := "why did authentication fail?"
	opts := prompt.BuildOptions{
		Summary:  testSummary,
		Question: q,
	}

	msgs, err := prompt.Build(prompt.TypeNaturalLanguageQuery, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	user := msgs[1].Content
	qIdx := strings.Index(user, q)
	sIdx := strings.Index(user, testSummary[:20])

	if qIdx < 0 {
		t.Error("question not found in user message")
	}
	if sIdx < 0 {
		t.Error("summary not found in user message")
	}
	if qIdx > sIdx {
		t.Error("question should appear before the log summary in the user message")
	}
}

// TestBuild_StructuredOutput_SecondPass_AssistantPrefill verifies that the
// second-pass assistant message contains the first-pass response verbatim.
func TestBuild_StructuredOutput_SecondPass_AssistantPrefill(t *testing.T) {
	firstPassText := "The logs reveal 42 connection refused errors starting at 14:32..."

	opts := prompt.BuildOptions{
		Summary:           testSummary,
		FirstPassResponse: firstPassText,
	}

	msgs, err := prompt.Build(prompt.TypeStructuredOutput, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages for second pass, got %d", len(msgs))
	}

	assistantMsg := msgs[2]
	if assistantMsg.Role != "assistant" {
		t.Errorf("msgs[2].Role = %q, want %q", assistantMsg.Role, "assistant")
	}
	if assistantMsg.Content != firstPassText {
		t.Errorf("assistant message content mismatch\ngot:  %q\nwant: %q",
			assistantMsg.Content, firstPassText)
	}

	// Final user message must mention JSON
	finalUser := msgs[3].Content
	if !strings.Contains(strings.ToLower(finalUser), "json") {
		t.Errorf("final user message should reference JSON, got: %q", finalUser)
	}
}

// TestBuild_StructuredOutput_SystemPromptMentionsJSON verifies the system
// prompt for TypeStructuredOutput references JSON output.
func TestBuild_StructuredOutput_SystemPromptMentionsJSON(t *testing.T) {
	opts := prompt.BuildOptions{Summary: testSummary}
	msgs, err := prompt.Build(prompt.TypeStructuredOutput, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	systemContent := msgs[0].Content
	if !strings.Contains(systemContent, "JSON") {
		t.Errorf("TypeStructuredOutput system prompt should mention JSON\ncontent: %s", systemContent)
	}
}
