package preprocess

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// Redactor removes sensitive data from log messages while preserving
// correlation between identical values.
//
// The same sensitive value will always be replaced with the same placeholder,
// allowing the LLM to understand relationships (e.g., "the same IP address
// appears in multiple error messages") without exposing actual values.
type Redactor struct {
	enabled  bool
	patterns []RedactionPattern
	hashMap  map[string]string // Original value -> placeholder
	mu       sync.RWMutex      // Protects hashMap
}

// NewRedactor creates a new Redactor with the specified configuration.
// If enabled is false, Redact() will return text unchanged.
func NewRedactor(enabled bool, patternNames []string) *Redactor {
	patterns := GetPatterns(patternNames)
	if len(patterns) == 0 {
		patterns = GetPatterns(DefaultPatterns())
	}

	return &Redactor{
		enabled:  enabled,
		patterns: patterns,
		hashMap:  make(map[string]string),
	}
}

// Redact scans the text for sensitive patterns and replaces them with
// correlation-preserving placeholders.
//
// Example:
//
//	"Connection from 192.168.1.1 failed" → "Connection from [IPV4:a3f2] failed"
//	"Connection from 192.168.1.1 succeeded" → "Connection from [IPV4:a3f2] succeeded"
//
// The same IP address always gets the same placeholder [IPV4:a3f2].
func (r *Redactor) Redact(text string) string {
	if !r.enabled || len(r.patterns) == 0 {
		return text
	}

	result := text

	// Apply each pattern
	for _, pattern := range r.patterns {
		result = r.redactPattern(result, pattern)
	}

	return result
}

// redactPattern applies a single redaction pattern to the text.
func (r *Redactor) redactPattern(text string, pattern RedactionPattern) string {
	return pattern.Regex.ReplaceAllStringFunc(text, func(match string) string {
		return r.getPlaceholder(match, pattern.Type)
	})
}

// getPlaceholder returns the placeholder for a given value.
// The same value always produces the same placeholder, enabling correlation.
func (r *Redactor) getPlaceholder(value, patternType string) string {
	r.mu.RLock()
	if placeholder, ok := r.hashMap[value]; ok {
		r.mu.RUnlock()
		return placeholder
	}
	r.mu.RUnlock()

	// Generate new placeholder
	hash := r.hashValue(value)
	placeholder := fmt.Sprintf("[%s:%s]", patternType, hash)

	// Store in map for future lookups
	r.mu.Lock()
	r.hashMap[value] = placeholder
	r.mu.Unlock()

	return placeholder
}

// hashValue generates a short, deterministic hash for a value.
// Uses first 4 hex characters of SHA256 for readability.
func (r *Redactor) hashValue(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:2]) // First 2 bytes = 4 hex chars
}

// GetUniqueValues returns a map of all unique values that were redacted
// and their corresponding placeholders. Useful for debugging.
func (r *Redactor) GetUniqueValues() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy
	result := make(map[string]string, len(r.hashMap))
	for k, v := range r.hashMap {
		result[k] = v
	}
	return result
}

// Reset clears the hash map, removing all remembered value->placeholder mappings.
// This is useful when processing a new file where correlations shouldn't
// persist across different contexts.
func (r *Redactor) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hashMap = make(map[string]string)
}

// IsEnabled returns whether redaction is enabled.
func (r *Redactor) IsEnabled() bool {
	return r.enabled
}

// RedactEntries applies redaction to a slice of LogEntry messages.
// This is a convenience function for batch processing.
func RedactEntries(redactor *Redactor, entries []LogEntry) []LogEntry {
	if redactor == nil || !redactor.IsEnabled() {
		return entries
	}

	result := make([]LogEntry, len(entries))
	for i, entry := range entries {
		result[i] = entry
		result[i].Message = redactor.Redact(entry.Message)
		// Also redact raw line if it differs from message
		if entry.Raw != entry.Message {
			result[i].Raw = redactor.Redact(entry.Raw)
		}
	}
	return result
}

// LogEntry is a local copy of config.LogEntry to avoid import cycles.
// This represents a single parsed log line with redacted content.
type LogEntry struct {
	Raw       string
	Timestamp interface{} // Can be time.Time or string
	Level     string
	Message   string
	Source    string
	Line      int
}

// SimpleRedact is a convenience function for one-off redaction without
// maintaining state. Each call creates a new Redactor, so correlations
// are only preserved within the single text.
func SimpleRedact(text string, patternNames []string) string {
	redactor := NewRedactor(true, patternNames)
	return redactor.Redact(text)
}

// RedactAndCount redacts text and returns the count of replacements made.
// This is useful for metrics and logging.
func (r *Redactor) RedactAndCount(text string) (string, int) {
	if !r.enabled || len(r.patterns) == 0 {
		return text, 0
	}

	count := 0
	result := text

	for _, pattern := range r.patterns {
		matches := pattern.Regex.FindAllString(result, -1)
		count += len(matches)
		result = r.redactPattern(result, pattern)
	}

	return result, count
}

// IsSensitive checks if text contains any sensitive patterns.
// This is a quick check that doesn't do redaction.
func (r *Redactor) IsSensitive(text string) bool {
	if !r.enabled || len(r.patterns) == 0 {
		return false
	}

	for _, pattern := range r.patterns {
		if pattern.Regex.MatchString(text) {
			return true
		}
	}
	return false
}

// CountSensitive returns the number of sensitive patterns found in text.
func (r *Redactor) CountSensitive(text string) int {
	if !r.enabled || len(r.patterns) == 0 {
		return 0
	}

	count := 0
	seen := make(map[string]bool)

	for _, pattern := range r.patterns {
		matches := pattern.Regex.FindAllString(text, -1)
		for _, match := range matches {
			if !seen[match] {
				count++
				seen[match] = true
			}
		}
	}

	return count
}

// NormalizeValue normalizes a value for consistent hashing.
// This handles variations like case differences in email addresses.
func NormalizeValue(value, patternType string) string {
	switch patternType {
	case "EMAIL":
		// Emails are case-insensitive for the domain part
		parts := strings.Split(value, "@")
		if len(parts) == 2 {
			return strings.ToLower(parts[0]) + "@" + strings.ToLower(parts[1])
		}
		return strings.ToLower(value)
	case "IPV4", "IPV6":
		// Normalize IP formatting
		return strings.ToLower(value)
	default:
		return value
	}
}
