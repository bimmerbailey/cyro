// Package parser provides log file parsing capabilities.
//
// It detects common log formats (syslog, JSON, Apache, generic) and extracts
// structured fields from each line.
package parser

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
)

// Format represents a detected log format.
type Format string

const (
	FormatJSON    Format = "json"
	FormatSyslog  Format = "syslog"
	FormatApache  Format = "apache"
	FormatGeneric Format = "generic"
)

// Parser reads and parses log files into structured entries.
type Parser struct {
	timestampFormats []string
}

// DetectFormat attempts to detect the log format from a line.
func DetectFormat(line string) Format {
	// Try JSON
	if len(line) > 0 && line[0] == '{' {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err == nil {
			return FormatJSON
		}
	}

	// Try syslog pattern
	if syslogPattern.MatchString(line) {
		return FormatSyslog
	}

	// Try Apache pattern
	if apachePattern.MatchString(line) {
		return FormatApache
	}

	// Default to generic
	return FormatGeneric
}

// New creates a new Parser with the given timestamp format patterns.
func New(timestampFormats []string) *Parser {
	if len(timestampFormats) == 0 {
		timestampFormats = []string{
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05",
			"Jan 02 15:04:05",
			"02/Jan/2006:15:04:05 -0700",
		}
	}
	return &Parser{timestampFormats: timestampFormats}
}

// ParseFile opens a file and parses all log entries from it.
func (p *Parser) ParseFile(path string) ([]config.LogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return p.Parse(f)
}

// Parse reads log entries from the given reader.
func (p *Parser) Parse(r io.Reader) ([]config.LogEntry, error) {
	var entries []config.LogEntry
	err := p.ParseStream(r, func(entry config.LogEntry) error {
		entries = append(entries, entry)
		return nil
	})
	return entries, err
}

// ParseStream reads log entries from the given reader and calls fn for each entry.
// The callback can return an error to stop parsing early.
func (p *Parser) ParseStream(r io.Reader, fn func(config.LogEntry) error) error {
	scanner := bufio.NewScanner(r)

	// Increase buffer size to handle long lines (default is 64KB, we use 1MB)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		entry := p.parseLine(line, lineNum)
		if err := fn(entry); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// ParseFileStream opens a file and calls fn for each parsed log entry.
func (p *Parser) ParseFileStream(path string, fn func(config.LogEntry) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return p.ParseStream(f, fn)
}

// parseLine attempts to parse a single log line into a LogEntry.
func (p *Parser) parseLine(line string, lineNum int) config.LogEntry {
	entry := config.LogEntry{
		Raw:    line,
		Line:   lineNum,
		Level:  config.LevelUnknown,
		Fields: make(map[string]interface{}),
	}

	// Try JSON first
	if p.tryParseJSON(line, &entry) {
		return entry
	}

	// Try syslog format
	if p.tryParseSyslog(line, &entry) {
		return entry
	}

	// Try Apache/Nginx Combined Log Format
	if p.tryParseApache(line, &entry) {
		return entry
	}

	// Generic format: try to extract timestamp and level, then clean up message
	cleanedLine := line

	// Try to extract timestamp
	entry.Timestamp = p.extractTimestamp(line)

	// Try to extract log level
	levelMatch := levelPattern.FindString(line)
	if levelMatch != "" {
		entry.Level = config.ParseLevel(levelMatch)
		// Remove the level from the cleaned line
		cleanedLine = strings.Replace(cleanedLine, levelMatch, "", 1)
	} else {
		entry.Level = config.LevelUnknown
	}

	// Try to remove common timestamp patterns from message
	timestampPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\[?\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?\]?\s*`),
		regexp.MustCompile(`^\[?\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?\]?\s*`),
		regexp.MustCompile(`^\[?\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}\]?\s*`),
	}
	for _, pattern := range timestampPatterns {
		cleanedLine = pattern.ReplaceAllString(cleanedLine, "")
	}

	// Remove common prefixes like [INFO], (ERROR), etc.
	cleanedLine = regexp.MustCompile(`^\s*[\[\(]?(DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|CRITICAL)[\]\)]?\s*[-:]?\s*`).ReplaceAllString(cleanedLine, "")

	// Trim whitespace
	entry.Message = strings.TrimSpace(cleanedLine)
	if entry.Message == "" {
		entry.Message = line // Fallback to raw line if we cleaned too much
	}

	return entry
}

// tryParseJSON attempts to parse the line as a JSON log entry.
func (p *Parser) tryParseJSON(line string, entry *config.LogEntry) bool {
	if len(line) == 0 || line[0] != '{' {
		return false
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return false
	}

	// Extract common JSON log fields
	for _, key := range []string{"msg", "message", "text"} {
		if v, ok := data[key].(string); ok {
			entry.Message = v
			break
		}
	}

	for _, key := range []string{"level", "severity", "lvl"} {
		if v, ok := data[key].(string); ok {
			entry.Level = config.ParseLevel(v)
			break
		}
	}

	for _, key := range []string{"time", "timestamp", "ts", "@timestamp"} {
		if v, ok := data[key].(string); ok {
			entry.Timestamp = p.parseTimestamp(v)
			break
		} else if v, ok := data[key].(float64); ok {
			// Handle numeric timestamps (epoch seconds or milliseconds)
			if v > 1e12 {
				// Likely milliseconds
				entry.Timestamp = time.Unix(0, int64(v)*int64(time.Millisecond))
			} else {
				// Likely seconds
				entry.Timestamp = time.Unix(int64(v), 0)
			}
			break
		}
	}

	if v, ok := data["source"].(string); ok {
		entry.Source = v
	}

	// Store remaining fields
	for k, v := range data {
		switch k {
		case "msg", "message", "text", "level", "severity", "lvl",
			"time", "timestamp", "ts", "@timestamp", "source":
			continue
		default:
			entry.Fields[k] = v
		}
	}

	return true
}

// syslogPattern matches BSD syslog format: Jan 02 15:04:05 hostname process[pid]: message
// Optionally with priority: <N>Jan 02 15:04:05 hostname process[pid]: message
var syslogPattern = regexp.MustCompile(`^(?:<(\d+)>)?(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+(\S+?)(?:\[(\d+)\])?:\s+(.*)$`)

// tryParseSyslog attempts to parse the line as a syslog entry.
func (p *Parser) tryParseSyslog(line string, entry *config.LogEntry) bool {
	matches := syslogPattern.FindStringSubmatch(line)
	if matches == nil {
		return false
	}

	// matches[1] = priority (optional)
	// matches[2] = timestamp
	// matches[3] = hostname
	// matches[4] = process name
	// matches[5] = pid (optional)
	// matches[6] = message

	// Parse timestamp (syslog format: Jan 02 15:04:05)
	// Note: syslog doesn't include year, we'll use current year
	timestampStr := matches[2]
	currentYear := time.Now().Year()
	fullTimestamp := timestampStr + " " + time.Now().Format("2006")
	for _, format := range []string{
		"Jan 02 15:04:05 2006",
		"Jan  2 15:04:05 2006",
	} {
		if t, err := time.Parse(format, fullTimestamp); err == nil {
			entry.Timestamp = t
			break
		}
	}
	_ = currentYear // unused for now, keeping for future use

	// Extract hostname as source
	entry.Source = matches[3]

	// Store process name and optional PID in fields
	if matches[4] != "" {
		entry.Fields["process"] = matches[4]
	}
	if matches[5] != "" {
		entry.Fields["pid"] = matches[5]
	}

	// Extract priority-based level if present
	if matches[1] != "" {
		// Syslog priority = facility * 8 + severity
		// Severity: 0=emerg, 1=alert, 2=crit, 3=error, 4=warning, 5=notice, 6=info, 7=debug
		priority := matches[1]
		if len(priority) > 0 {
			// Parse the priority number to get severity
			var priorityNum int
			if _, err := regexp.MatchString(`^\d+$`, priority); err == nil {
				for _, ch := range priority {
					priorityNum = priorityNum*10 + int(ch-'0')
				}
				severity := priorityNum % 8 // Last 3 bits are severity
				switch severity {
				case 7:
					entry.Level = config.LevelDebug
				case 6, 5:
					entry.Level = config.LevelInfo
				case 4:
					entry.Level = config.LevelWarn
				case 3:
					entry.Level = config.LevelError
				case 2, 1, 0:
					entry.Level = config.LevelFatal
				}
			}
		}
	}

	// Message
	entry.Message = matches[6]

	// Try to extract level from message if not set by priority
	if entry.Level == config.LevelUnknown {
		entry.Level = p.extractLevel(matches[6])
	}

	return true
}

// apachePattern matches Apache Combined Log Format:
// 127.0.0.1 - user [02/Jan/2006:15:04:05 -0700] "GET /path HTTP/1.1" 200 1234 "referer" "user-agent"
var apachePattern = regexp.MustCompile(`^(\S+) (\S+) (\S+) \[([^\]]+)\] "(\S+) (\S+)(?: (\S+))?" (\d{3}) (\d+|-) "([^"]*)" "([^"]*)"`)

// tryParseApache attempts to parse the line as an Apache/Nginx Combined Log Format entry.
func (p *Parser) tryParseApache(line string, entry *config.LogEntry) bool {
	matches := apachePattern.FindStringSubmatch(line)
	if matches == nil {
		return false
	}

	// matches[1] = remote host
	// matches[2] = identity (usually -)
	// matches[3] = user (usually -)
	// matches[4] = timestamp
	// matches[5] = method
	// matches[6] = path
	// matches[7] = protocol (optional)
	// matches[8] = status code
	// matches[9] = size
	// matches[10] = referer
	// matches[11] = user agent

	// Remote host as source
	entry.Source = matches[1]

	// Parse timestamp (Apache format: 02/Jan/2006:15:04:05 -0700)
	if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[4]); err == nil {
		entry.Timestamp = t
	}

	// Build message from HTTP request
	protocol := matches[7]
	if protocol == "" {
		protocol = "HTTP/1.0"
	}
	entry.Message = matches[5] + " " + matches[6] + " " + protocol + " -> " + matches[8]

	// Store request details in fields
	entry.Fields["method"] = matches[5]
	entry.Fields["path"] = matches[6]
	entry.Fields["protocol"] = protocol
	entry.Fields["status_code"] = matches[8]
	if matches[9] != "-" {
		entry.Fields["size"] = matches[9]
	}
	if matches[10] != "-" && matches[10] != "" {
		entry.Fields["referer"] = matches[10]
	}
	if matches[11] != "" {
		entry.Fields["user_agent"] = matches[11]
	}
	if matches[3] != "-" {
		entry.Fields["user"] = matches[3]
	}

	// Derive log level from HTTP status code
	statusCode := matches[8]
	if len(statusCode) == 3 {
		switch statusCode[0] {
		case '2', '3':
			entry.Level = config.LevelInfo
		case '4':
			entry.Level = config.LevelWarn
		case '5':
			entry.Level = config.LevelError
		default:
			entry.Level = config.LevelUnknown
		}
	}

	return true
}

// levelPattern matches common log level strings.
var levelPattern = regexp.MustCompile(`(?i)\b(DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|CRITICAL)\b`)

// extractLevel extracts the log level from a line.
func (p *Parser) extractLevel(line string) config.LogLevel {
	match := levelPattern.FindString(line)
	if match == "" {
		return config.LevelUnknown
	}
	return config.ParseLevel(match)
}

// extractTimestamp tries all known timestamp formats against the line.
// It searches for timestamps at the beginning, inside brackets, or elsewhere in the line.
func (p *Parser) extractTimestamp(line string) time.Time {
	// Try common timestamp patterns with regex first for better detection
	timestampPatterns := []struct {
		regex  *regexp.Regexp
		format string
	}{
		// ISO 8601 / RFC3339
		{regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`), "2006-01-02T15:04:05Z07:00"},
		{regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z`), "2006-01-02T15:04:05Z"},
		// Common datetime
		{regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`), "2006-01-02 15:04:05"},
		{regexp.MustCompile(`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}`), "01/02/2006 15:04:05"},
	}

	for _, tp := range timestampPatterns {
		if match := tp.regex.FindString(line); match != "" {
			if t, err := time.Parse(tp.format, match); err == nil {
				return t
			}
			// Try with milliseconds
			if t, err := time.Parse(tp.format+".999999999", match); err == nil {
				return t
			}
		}
	}

	// Fallback to original format-based extraction
	for _, format := range p.timestampFormats {
		if t := p.tryTimestampFormat(line, format); !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

// parseTimestamp parses a known timestamp string.
func (p *Parser) parseTimestamp(s string) time.Time {
	for _, format := range p.timestampFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// tryTimestampFormat attempts to parse a timestamp from a line using a specific format.
func (p *Parser) tryTimestampFormat(line string, format string) time.Time {
	// Try from the beginning of the line (most common)
	fmtLen := len(format)
	if len(line) >= fmtLen {
		if t, err := time.Parse(format, line[:fmtLen]); err == nil {
			return t
		}
	}
	return time.Time{}
}
