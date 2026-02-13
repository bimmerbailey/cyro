package preprocess

import (
	"regexp"
)

// RedactionPattern defines a built-in pattern for secret detection.
type RedactionPattern struct {
	Name        string
	Regex       *regexp.Regexp
	Type        string // Used for placeholder prefix: [IPV4:hash], [EMAIL:hash], etc.
	Description string
}

// Built-in redaction patterns for common sensitive data.
// These patterns detect PII, credentials, and other sensitive information in logs.
var (
	// IPv4 addresses: 192.168.1.1
	ipv4Regex = regexp.MustCompile(`\b(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)

	// IPv6 addresses: 2001:db8::1, fe80::1%eth0
	ipv6Regex = regexp.MustCompile(`(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,7}:|(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}|(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}|(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}|(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(?::[0-9a-fA-F]{1,4}){1,6}|:(?::[0-9a-fA-F]{1,4}){1,7}|::(?:[fF]{4}(?::0{1,4}){0,1}:){0,1}(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)|(?:[0-9a-fA-F]{1,4}:){1,4}:(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`)

	// Email addresses: user@example.com
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// API keys and tokens (common patterns)
	// AWS Access Key ID: AKIAIOSFODNN7EXAMPLE
	awsAccessKeyRegex = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)

	// Generic API keys: api_key=..., apikey=..., token=...
	// Matches various common API key formats
	apiKeyRegex = regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|token|secret|password|passwd|pwd)["\s]*[:=]["\s]*[a-zA-Z0-9_\-]{8,}`)

	// JWT tokens: eyJhbGciOiJIUzI1NiIs...
	jwtRegex = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*\b`)

	// Private keys (BEGIN RSA PRIVATE KEY, etc.)
	privateKeyRegex = regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`)

	// MAC addresses: 00:1B:44:11:3A:B7 or 00-1B-44-11-3A-B7
	macAddressRegex = regexp.MustCompile(`\b(?:[0-9A-Fa-f]{2}[:-]){5}(?:[0-9A-Fa-f]{2})\b`)

	// Credit card numbers (basic pattern, detects common formats)
	creditCardRegex = regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`)

	// UUIDs: 550e8400-e29b-41d4-a716-446655440000
	// Note: These are often not sensitive but included for completeness
	uuidRegex = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
)

// BuiltInPatterns contains all available redaction patterns.
// These can be selectively enabled/disabled via configuration.
var BuiltInPatterns = map[string]RedactionPattern{
	"ipv4": {
		Name:        "ipv4",
		Regex:       ipv4Regex,
		Type:        "IPV4",
		Description: "IPv4 addresses",
	},
	"ipv6": {
		Name:        "ipv6",
		Regex:       ipv6Regex,
		Type:        "IPV6",
		Description: "IPv6 addresses",
	},
	"email": {
		Name:        "email",
		Regex:       emailRegex,
		Type:        "EMAIL",
		Description: "Email addresses",
	},
	"api_key": {
		Name:        "api_key",
		Regex:       apiKeyRegex,
		Type:        "SECRET",
		Description: "API keys and tokens",
	},
	"aws_key": {
		Name:        "aws_key",
		Regex:       awsAccessKeyRegex,
		Type:        "AWS_KEY",
		Description: "AWS Access Key IDs",
	},
	"jwt": {
		Name:        "jwt",
		Regex:       jwtRegex,
		Type:        "JWT",
		Description: "JWT tokens",
	},
	"private_key": {
		Name:        "private_key",
		Regex:       privateKeyRegex,
		Type:        "PRIVATE_KEY",
		Description: "Private key headers",
	},
	"mac_address": {
		Name:        "mac_address",
		Regex:       macAddressRegex,
		Type:        "MAC",
		Description: "MAC addresses",
	},
	"credit_card": {
		Name:        "credit_card",
		Regex:       creditCardRegex,
		Type:        "CC",
		Description: "Credit card numbers",
	},
	"uuid": {
		Name:        "uuid",
		Regex:       uuidRegex,
		Type:        "UUID",
		Description: "UUIDs",
	},
}

// DefaultPatterns returns the recommended set of patterns to enable by default.
// This includes the most common sensitive data types while avoiding
// patterns that might have too many false positives.
func DefaultPatterns() []string {
	return []string{
		"ipv4",
		"ipv6",
		"email",
		"api_key",
		"aws_key",
		"jwt",
		"private_key",
	}
}

// GetPatterns returns the patterns matching the given names.
// Unknown pattern names are silently ignored.
func GetPatterns(names []string) []RedactionPattern {
	patterns := make([]RedactionPattern, 0, len(names))
	for _, name := range names {
		if pattern, ok := BuiltInPatterns[name]; ok {
			patterns = append(patterns, pattern)
		}
	}
	return patterns
}
