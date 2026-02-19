// Package prompt provides structured prompt templates for Cyro's LLM-powered
// log analysis features.
//
// # Overview
//
// The package defines a set of [PromptType] constants, each representing a
// distinct analysis task. Callers construct a [BuildOptions] value describing
// the log context and call [Build] to receive a fully-formed []llm.Message
// slice that can be sent directly to any [llm.Provider].
//
// # Prompt types
//
//   - [TypeSummarize]           — high-level narrative summary of the logs
//   - [TypeRootCause]           — evidence-based diagnosis of what caused errors
//   - [TypeAnomalyDetection]    — identify unusual patterns versus a baseline
//   - [TypeNaturalLanguageQuery] — answer a free-form user question
//   - [TypeStructuredOutput]    — two-pass pattern for reliable JSON from small models
//
// # Basic usage
//
//	opts := prompt.BuildOptions{
//	    Summary: preprocessOutput.Summary,
//	}
//	messages, err := prompt.Build(prompt.TypeSummarize, opts)
//	if err != nil {
//	    return err
//	}
//	// Pass messages directly to llm.Provider.ChatStream(ctx, messages, chatOpts)
//
// # Two-pass structured output
//
// Small models often struggle to produce well-formed JSON on the first attempt.
// [TypeStructuredOutput] implements a two-pass approach:
//
//  1. Call [Build] with an empty [BuildOptions.FirstPassResponse] to get the
//     initial analysis-request messages. Send them to the LLM and collect the
//     full response text.
//
//  2. Call [Build] again with [BuildOptions.FirstPassResponse] set to that
//     response. The returned message slice appends an assistant message
//     (prefilling the model's prior answer) followed by a user extraction
//     instruction. Send this second slice to the LLM to obtain structured JSON.
//
// Example:
//
//	// First pass — ask for analysis
//	opts := prompt.BuildOptions{Summary: summary}
//	firstMsgs, _ := prompt.Build(prompt.TypeStructuredOutput, opts)
//	firstResp, _ := provider.Chat(ctx, firstMsgs, chatOpts)
//
//	// Second pass — extract JSON
//	opts.FirstPassResponse = firstResp.Content
//	secondMsgs, _ := prompt.Build(prompt.TypeStructuredOutput, opts)
//	jsonResp, _ := provider.Chat(ctx, secondMsgs, chatOpts)
package prompt
