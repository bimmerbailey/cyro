package output

import (
	"fmt"
	"os"

	"github.com/bimmerbailey/cyro/internal/config"
	"golang.org/x/term"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// ColorMode determines when to use colored output.
type ColorMode int

const (
	ColorAuto   ColorMode = iota // Auto-detect based on TTY
	ColorAlways                  // Always use colors
	ColorNever                   // Never use colors
)

// isTerminal checks if the given file is a terminal.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// shouldColorize determines if output should be colorized based on mode and TTY detection.
func shouldColorize(mode ColorMode, w interface{}) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	case ColorAuto:
		// Check if writer is a file and if it's a terminal
		if f, ok := w.(*os.File); ok {
			return isTerminal(f)
		}
		return false
	}
	return false
}

// colorizeLevel adds color to a log level string based on severity.
func colorizeLevel(level config.LogLevel, text string) string {
	switch level {
	case config.LevelDebug:
		return colorGray + text + colorReset
	case config.LevelInfo:
		return text // Default color
	case config.LevelWarn:
		return colorYellow + text + colorReset
	case config.LevelError:
		return colorRed + text + colorReset
	case config.LevelFatal:
		return colorBold + colorRed + text + colorReset
	default:
		return text
	}
}

// ColorizeLine applies color to an entire log line based on its level.
func ColorizeLine(level config.LogLevel, line string) string {
	switch level {
	case config.LevelDebug:
		return colorGray + line + colorReset
	case config.LevelWarn:
		return colorYellow + line + colorReset
	case config.LevelError:
		return colorRed + line + colorReset
	case config.LevelFatal:
		return colorBold + colorRed + line + colorReset
	default:
		return line // INFO and UNKNOWN use default color
	}
}

// FormatEntry formats a single log entry with optional coloring.
func FormatEntry(entry config.LogEntry, colorize bool) string {
	if colorize {
		return ColorizeLine(entry.Level, entry.Raw)
	}
	return entry.Raw
}

// WriteColoredEntry writes a log entry to the writer with color based on ColorMode.
func (wr *Writer) WriteColoredEntry(entry config.LogEntry, mode ColorMode) error {
	colorize := shouldColorize(mode, wr.w)
	line := FormatEntry(entry, colorize)
	_, err := fmt.Fprintln(wr.w, line)
	return err
}
