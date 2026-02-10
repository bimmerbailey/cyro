// Package tail provides log file tailing capabilities with live filtering.
//
// It implements "tail -f" like functionality with support for pattern matching,
// level filtering, and log rotation detection.
package tail

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/parser"
	"github.com/fsnotify/fsnotify"
)

// Options configures the tailer behavior.
type Options struct {
	FilePath     string                      // Path to the log file
	Lines        int                         // Number of initial lines to show
	Follow       bool                        // Whether to follow the file for new content
	FollowRotate bool                        // Whether to follow through log rotations
	Pattern      *regexp.Regexp              // Optional regex pattern to filter lines
	LevelFilter  config.LogLevel             // Minimum log level to display
	OutputFunc   func(config.LogEntry) error // Function called for each matching entry
}

// Tailer handles tailing a log file with filtering.
type Tailer struct {
	opts    Options
	parser  *parser.Parser
	file    *os.File
	offset  int64
	watcher *fsnotify.Watcher
}

// New creates a new Tailer with the given options.
func New(opts Options) *Tailer {
	return &Tailer{
		opts:   opts,
		parser: parser.New(nil),
	}
}

// Run starts the tailing process. It blocks until context is cancelled or an error occurs.
func (t *Tailer) Run(ctx context.Context) error {
	// Open the file
	if err := t.openFile(); err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer t.close()

	// Read initial N lines
	if t.opts.Lines > 0 {
		if err := t.readInitialLines(); err != nil {
			return fmt.Errorf("failed to read initial lines: %w", err)
		}
	}

	// If not following, we're done
	if !t.opts.Follow {
		return nil
	}

	// Set up file watcher
	if err := t.setupWatcher(); err != nil {
		return fmt.Errorf("failed to setup watcher: %w", err)
	}
	defer t.watcher.Close()

	// Watch for changes
	return t.watch(ctx)
}

// openFile opens the log file and seeks to the end if following.
func (t *Tailer) openFile() error {
	f, err := os.Open(t.opts.FilePath)
	if err != nil {
		return err
	}
	t.file = f

	// Get initial file position at the end
	if t.opts.Follow {
		// We'll seek back for initial lines, then return to end
		stat, err := f.Stat()
		if err != nil {
			return err
		}
		t.offset = stat.Size()
	}

	return nil
}

// readInitialLines reads and displays the last N lines from the file.
func (t *Tailer) readInitialLines() error {
	// Get file size
	stat, err := t.file.Stat()
	if err != nil {
		return err
	}
	fileSize := stat.Size()

	// If file is empty, nothing to read
	if fileSize == 0 {
		return nil
	}

	// Try to find a good starting position
	// We use a heuristic: average line length of ~300 bytes (generous for JSON logs)
	// We multiply by 2 to ensure we get enough lines
	estimatedBytesNeeded := int64(t.opts.Lines * 300 * 2)
	startPos := fileSize - estimatedBytesNeeded
	if startPos < 0 {
		startPos = 0
	}

	// Seek to estimated position
	if _, err := t.file.Seek(startPos, io.SeekStart); err != nil {
		return err
	}

	// Create scanner
	scanner := bufio.NewScanner(t.file)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	// If we're not at the start, skip the first partial line
	if startPos > 0 {
		if scanner.Scan() {
			// Discard first partial line
		}
	}

	// Read all lines from this position to end
	var entries []config.LogEntry
	linesRead := 0
	for scanner.Scan() {
		linesRead++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		entry := t.parseLine(line, linesRead)

		if t.shouldDisplay(entry) {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Keep only the last N entries
	if len(entries) > t.opts.Lines {
		entries = entries[len(entries)-t.opts.Lines:]
	}

	// Output the entries
	for _, entry := range entries {
		if err := t.opts.OutputFunc(entry); err != nil {
			return err
		}
	}

	// Update offset to current position (end of file)
	t.offset, err = t.file.Seek(0, io.SeekEnd)
	return err
}

// setupWatcher initializes the fsnotify watcher.
func (t *Tailer) setupWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	t.watcher = watcher

	// Add file to watcher
	if err := watcher.Add(t.opts.FilePath); err != nil {
		return err
	}

	return nil
}

// watch monitors the file for changes and outputs new lines.
func (t *Tailer) watch(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-t.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed unexpectedly")
			}

			if err := t.handleEvent(ctx, event); err != nil {
				return err
			}

		case err, ok := <-t.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

// handleEvent processes a file system event.
func (t *Tailer) handleEvent(ctx context.Context, event fsnotify.Event) error {
	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		// File was written to, read new content
		return t.readNewContent()

	case event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename:
		// File was removed or renamed (log rotation)
		return t.handleRotation(ctx)

	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		// Ignore chmod events
		return nil
	}

	return nil
}

// readNewContent reads and outputs new content added to the file.
func (t *Tailer) readNewContent() error {
	// Seek to last known position
	if _, err := t.file.Seek(t.offset, io.SeekStart); err != nil {
		return err
	}

	// Read new lines
	scanner := bufio.NewScanner(t.file)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		entry := t.parseLine(line, lineNum)

		if t.shouldDisplay(entry) {
			if err := t.opts.OutputFunc(entry); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Update offset
	var err error
	t.offset, err = t.file.Seek(0, io.SeekCurrent)
	return err
}

// handleRotation handles log file rotation.
func (t *Tailer) handleRotation(ctx context.Context) error {
	if !t.opts.FollowRotate {
		// Exit gracefully
		fmt.Fprintf(os.Stderr, "\nFile rotated. Exiting. Use --follow-rotate to follow through rotations.\n")
		return fmt.Errorf("file rotated")
	}

	// Close current file
	if t.file != nil {
		t.file.Close()
		t.file = nil
	}

	// Wait for new file to appear (with timeout)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timeout:
			return fmt.Errorf("timeout waiting for rotated file to reappear")
		case <-ticker.C:
			// Try to open the file
			f, err := os.Open(t.opts.FilePath)
			if err == nil {
				// File exists again
				t.file = f
				t.offset = 0

				// Re-add file to watcher
				if err := t.watcher.Add(t.opts.FilePath); err != nil {
					return fmt.Errorf("failed to watch rotated file: %w", err)
				}

				fmt.Fprintf(os.Stderr, "\n==> File rotated, following new file <==\n")
				return nil
			}
		}
	}
}

// parseLine parses a single log line into a LogEntry.
func (t *Tailer) parseLine(line string, lineNum int) config.LogEntry {
	entry := config.LogEntry{
		Raw:    line,
		Line:   lineNum,
		Level:  config.LevelUnknown,
		Fields: make(map[string]interface{}),
	}

	// Use parser's internal logic by creating a temporary reader with newline
	// The parser expects complete lines ending with newline
	lineWithNewline := line + "\n"

	// Create a minimal parser instance and parse the line
	p := parser.New(nil)
	entries, err := p.Parse(strings.NewReader(lineWithNewline))
	if err == nil && len(entries) > 0 {
		entry = entries[0]
		entry.Line = lineNum // Preserve our line number
		entry.Raw = line     // Use original line without newline
	}

	return entry
}

// shouldDisplay checks if an entry matches the filter criteria.
func (t *Tailer) shouldDisplay(entry config.LogEntry) bool {
	// Check level filter
	if t.opts.LevelFilter != config.LevelUnknown {
		// Level filter: show entries at or above the specified level
		// Exception: Unknown level entries are always shown (can't filter what we can't classify)
		if entry.Level != config.LevelUnknown {
			entryLevel := levelToInt(entry.Level)
			filterLevel := levelToInt(t.opts.LevelFilter)
			if entryLevel < filterLevel {
				return false
			}
		}
	}

	// Check pattern filter
	if t.opts.Pattern != nil {
		if !t.opts.Pattern.MatchString(entry.Raw) {
			return false
		}
	}

	return true
}

// levelToInt converts a log level to an integer for comparison.
func levelToInt(level config.LogLevel) int {
	switch level {
	case config.LevelDebug:
		return 0
	case config.LevelInfo:
		return 1
	case config.LevelWarn:
		return 2
	case config.LevelError:
		return 3
	case config.LevelFatal:
		return 4
	default:
		return -1 // Unknown levels are shown
	}
}

// close closes all resources.
func (t *Tailer) close() {
	if t.file != nil {
		t.file.Close()
	}
	if t.watcher != nil {
		t.watcher.Close()
	}
}
