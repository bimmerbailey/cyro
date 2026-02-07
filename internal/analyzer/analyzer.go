// Package analyzer provides log analysis capabilities including
// pattern detection, frequency analysis, and anomaly identification.
package analyzer

import (
	"regexp"
	"sort"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
)

// Stats holds aggregate statistics for a set of log entries.
type Stats struct {
	TotalLines  int                     `json:"total_lines"`
	LevelCounts map[config.LogLevel]int `json:"level_counts"`
	FirstEntry  time.Time               `json:"first_entry,omitempty"`
	LastEntry   time.Time               `json:"last_entry,omitempty"`
	TopMessages []MessageCount          `json:"top_messages,omitempty"`
	ErrorRate   float64                 `json:"error_rate"`
}

// MessageCount tracks a message and how often it appears.
type MessageCount struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// Analyzer performs analysis on parsed log entries.
type Analyzer struct{}

// New creates a new Analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// ComputeStats calculates aggregate statistics from a set of log entries.
func (a *Analyzer) ComputeStats(entries []config.LogEntry, topN int) Stats {
	stats := Stats{
		TotalLines:  len(entries),
		LevelCounts: make(map[config.LogLevel]int),
	}

	if len(entries) == 0 {
		return stats
	}

	messageCounts := make(map[string]int)

	for _, e := range entries {
		stats.LevelCounts[e.Level]++

		if !e.Timestamp.IsZero() {
			if stats.FirstEntry.IsZero() || e.Timestamp.Before(stats.FirstEntry) {
				stats.FirstEntry = e.Timestamp
			}
			if stats.LastEntry.IsZero() || e.Timestamp.After(stats.LastEntry) {
				stats.LastEntry = e.Timestamp
			}
		}

		messageCounts[e.Message]++
	}

	// Calculate error rate
	errorCount := stats.LevelCounts[config.LevelError] + stats.LevelCounts[config.LevelFatal]
	if stats.TotalLines > 0 {
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalLines)
	}

	// Get top messages
	stats.TopMessages = topMessages(messageCounts, topN)

	return stats
}

// Filter returns entries matching the given criteria.
func (a *Analyzer) Filter(entries []config.LogEntry, opts FilterOptions) []config.LogEntry {
	var result []config.LogEntry

	var re *regexp.Regexp
	if opts.Pattern != "" {
		re, _ = regexp.Compile(opts.Pattern)
	}

	for _, e := range entries {
		if opts.MinLevel != config.LevelUnknown && e.Level < opts.MinLevel {
			continue
		}

		if !opts.Since.IsZero() && !e.Timestamp.IsZero() && e.Timestamp.Before(opts.Since) {
			continue
		}

		if !opts.Until.IsZero() && !e.Timestamp.IsZero() && e.Timestamp.After(opts.Until) {
			continue
		}

		if re != nil {
			matched := re.MatchString(e.Raw)
			if opts.Invert {
				matched = !matched
			}
			if !matched {
				continue
			}
		}

		result = append(result, e)
	}

	return result
}

// FilterOptions defines the criteria for filtering log entries.
type FilterOptions struct {
	Pattern  string
	MinLevel config.LogLevel
	Since    time.Time
	Until    time.Time
	Invert   bool
}

// topMessages extracts the N most frequent messages.
func topMessages(counts map[string]int, n int) []MessageCount {
	msgs := make([]MessageCount, 0, len(counts))
	for msg, count := range counts {
		msgs = append(msgs, MessageCount{Message: msg, Count: count})
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Count > msgs[j].Count
	})

	if len(msgs) > n {
		msgs = msgs[:n]
	}

	return msgs
}
