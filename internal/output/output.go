// Package output provides formatted output rendering for log entries
// and analysis results. It supports text, JSON, and table formats.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/bimmerbailey/cyro/internal/config"
)

// Format represents an output format type.
type Format string

const (
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// ParseFormat converts a string to a Format, defaulting to text.
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "table":
		return FormatTable
	default:
		return FormatText
	}
}

// Writer handles writing formatted output.
type Writer struct {
	w      io.Writer
	format Format
}

// New creates a new output Writer.
func New(w io.Writer, format Format) *Writer {
	return &Writer{w: w, format: format}
}

// WriteEntries outputs a slice of log entries in the configured format.
func (wr *Writer) WriteEntries(entries []config.LogEntry) error {
	switch wr.format {
	case FormatJSON:
		return wr.writeJSON(entries)
	case FormatTable:
		return wr.writeTable(entries)
	default:
		return wr.writeText(entries)
	}
}

// WriteJSON outputs any value as indented JSON.
func (wr *Writer) WriteJSON(v interface{}) error {
	enc := json.NewEncoder(wr.w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (wr *Writer) writeJSON(entries []config.LogEntry) error {
	return wr.WriteJSON(entries)
}

func (wr *Writer) writeText(entries []config.LogEntry) error {
	for _, e := range entries {
		fmt.Fprintln(wr.w, e.Raw)
	}
	return nil
}

func (wr *Writer) writeTable(entries []config.LogEntry) error {
	tw := tabwriter.NewWriter(wr.w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "LINE\tLEVEL\tTIMESTAMP\tMESSAGE")
	fmt.Fprintln(tw, "----\t-----\t---------\t-------")

	for _, e := range entries {
		ts := ""
		if !e.Timestamp.IsZero() {
			ts = e.Timestamp.Format("15:04:05")
		}

		msg := e.Message
		if len(msg) > 80 {
			msg = msg[:77] + "..."
		}

		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", e.Line, e.Level, ts, msg)
	}

	return tw.Flush()
}
