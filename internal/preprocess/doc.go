// Package preprocess provides log compression and redaction for LLM consumption.
//
// The preprocessing pipeline reduces raw logs by ~50x using three stages:
//
//  1. Template Extraction (Drain algorithm) - Groups similar log messages
//  2. Secret Redaction - Removes PII/credentials with correlation-preserving hashes
//  3. Token Budget Enforcement - Prioritizes errors and frequent patterns
//
// Basic usage:
//
//	preprocessor := preprocess.New(
//	    preprocess.WithTokenLimit(8000),
//	    preprocess.WithRedaction(true),
//	)
//	compressed, err := preprocessor.Process(logEntries)
//
// The output is optimized for LLM consumption and fits within context windows.
//
// Configuration via ~/.cyro.yaml:
//
//	redaction:
//	  enabled: true
//	  patterns:
//	    - ipv4
//	    - ipv6
//	    - email
//	    - api_key
package preprocess
