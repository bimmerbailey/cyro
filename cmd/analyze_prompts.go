package cmd

// AnalysisContext provides metadata about the analysis request.
// It is passed alongside log entries to build the LLM prompt via
// the internal/prompt package.
type AnalysisContext struct {
	Pattern string
	GroupBy string
	Window  string
	Files   []string
}
