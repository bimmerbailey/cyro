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
	scanner := bufio.NewScanner(r)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		entry := p.parseLine(line, lineNum)
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return entries, err
	}

	return entries, nil
}

// parseLine attempts to parse a single log line into a LogEntry.
func (p *Parser) parseLine(line string, lineNum int) config.LogEntry {
	entry := config.LogEntry{
		Raw:    line,
		Line:   lineNum,
		Level:  config.LevelUnknown,
		Fields: make(map[string]string),
	}

	// Try JSON first
	if p.tryParseJSON(line, &entry) {
		return entry
	}

	// Try to extract timestamp
	entry.Timestamp = p.extractTimestamp(line)

	// Try to extract log level
	entry.Level = p.extractLevel(line)

	// The remainder is the message
	entry.Message = line

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
			if s, ok := v.(string); ok {
				entry.Fields[k] = s
			}
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
func (p *Parser) extractTimestamp(line string) time.Time {
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
