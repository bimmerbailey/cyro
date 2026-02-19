package cmd

// Prompt construction tests for the ask command have moved to
// internal/prompt/prompt_test.go, which tests prompt.Build(TypeNaturalLanguageQuery, ...)
// directly.
//
// This file is reserved for integration-level tests of the runAsk command
// that do not require a live LLM (e.g. flag validation, file parsing,
// preprocessing path).
