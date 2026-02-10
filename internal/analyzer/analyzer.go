// Package analyzer provides log analysis capabilities including
// pattern detection, frequency analysis, and anomaly identification.
package analyzer

import (
	"fmt"
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

// GroupedResult represents entries grouped by a field value.
type GroupedResult struct {
	Key     string  `json:"key"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// TimeWindowStats holds statistics for a time window.
type TimeWindowStats struct {
	Start         time.Time               `json:"start"`
	End           time.Time               `json:"end"`
	Count         int                     `json:"count"`
	LevelCounts   map[config.LogLevel]int `json:"level_counts"`
	ErrorCount    int                     `json:"error_count"`
	ErrorPercent  float64                 `json:"error_percent"`
	ChangePercent float64                 `json:"change_percent"` // Change from previous window
}

// AnalysisResult contains the full analysis output.
type AnalysisResult struct {
	TotalLines  int               `json:"total_lines"`
	GroupBy     string            `json:"group_by"`
	Groups      []GroupedResult   `json:"groups"`
	TimeWindows []TimeWindowStats `json:"time_windows,omitempty"`
	Pattern     string            `json:"pattern,omitempty"`
	FilePath    string            `json:"file_path,omitempty"`
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
		if opts.MinLevel != config.LevelUnknown {
			if opts.ExactLevel {
				if e.Level != opts.MinLevel {
					continue
				}
			} else if e.Level < opts.MinLevel {
				continue
			}
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
	Pattern    string
	MinLevel   config.LogLevel
	Since      time.Time
	Until      time.Time
	Invert     bool
	ExactLevel bool
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

// GroupBy groups entries by a specific field and returns the top N groups.
// Supported fields: "level", "message", "source".
func (a *Analyzer) GroupBy(entries []config.LogEntry, field string, topN int) ([]GroupedResult, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	groups := make(map[string]int)

	for _, e := range entries {
		var key string
		switch field {
		case "level":
			key = e.Level.String()
		case "message":
			key = e.Message
		case "source":
			key = e.Source
		default:
			return nil, fmt.Errorf("unsupported group-by field: %s (must be 'level', 'message', or 'source')", field)
		}

		if key == "" && field == "source" {
			key = "(unknown)"
		}

		groups[key]++
	}

	// Convert to slice and sort
	result := make([]GroupedResult, 0, len(groups))
	total := len(entries)
	for key, count := range groups {
		result = append(result, GroupedResult{
			Key:     key,
			Count:   count,
			Percent: float64(count) * 100 / float64(total),
		})
	}

	// Sort by count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	// Return top N
	if len(result) > topN {
		result = result[:topN]
	}

	return result, nil
}

// AnalyzeByWindow splits entries into time windows and calculates statistics.
func (a *Analyzer) AnalyzeByWindow(entries []config.LogEntry, window time.Duration) []TimeWindowStats {
	if len(entries) == 0 {
		return nil
	}

	// Find time range
	var minTime, maxTime time.Time
	for _, e := range entries {
		if !e.Timestamp.IsZero() {
			if minTime.IsZero() || e.Timestamp.Before(minTime) {
				minTime = e.Timestamp
			}
			if maxTime.IsZero() || e.Timestamp.After(maxTime) {
				maxTime = e.Timestamp
			}
		}
	}

	if minTime.IsZero() || maxTime.IsZero() {
		return nil
	}

	// Align to window boundaries
	windowStart := minTime.Truncate(window)
	var windows []TimeWindowStats

	for current := windowStart; current.Before(maxTime) || current.Equal(maxTime); current = current.Add(window) {
		windows = append(windows, TimeWindowStats{
			Start:       current,
			End:         current.Add(window),
			LevelCounts: make(map[config.LogLevel]int),
		})
	}

	// Assign entries to windows
	for _, e := range entries {
		if e.Timestamp.IsZero() {
			continue
		}

		// Find the window this entry belongs to
		windowIdx := int(e.Timestamp.Sub(windowStart) / window)
		if windowIdx >= 0 && windowIdx < len(windows) {
			windows[windowIdx].Count++
			windows[windowIdx].LevelCounts[e.Level]++
			if e.Level >= config.LevelError {
				windows[windowIdx].ErrorCount++
			}
		}
	}

	// Calculate error percentages and changes
	for i := range windows {
		if windows[i].Count > 0 {
			windows[i].ErrorPercent = float64(windows[i].ErrorCount) * 100 / float64(windows[i].Count)
		}

		// Calculate change from previous window
		if i > 0 {
			if windows[i-1].Count > 0 {
				windows[i].ChangePercent = float64(windows[i].Count-windows[i-1].Count) * 100 / float64(windows[i-1].Count)
			}
		}
	}

	return windows
}
