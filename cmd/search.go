package cmd

import (
	"fmt"
	"regexp"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/output"
	"github.com/bimmerbailey/cyro/internal/parser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var searchCmd = &cobra.Command{
	Use:   "search [flags] <file>",
	Short: "Search and filter log entries",
	Long: `Search through log files using patterns, log levels, and time ranges.

Supports regex patterns, level-based filtering, and time window queries.

Examples:
  cyro search --pattern "error|timeout" /var/log/app.log
  cyro search --level error --since "2024-01-01" /var/log/app.log
  cyro search --pattern "user_id=123" --level warn app.log`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().StringP("pattern", "p", "", "regex pattern to search for")
	searchCmd.Flags().StringP("level", "l", "", "filter by log level (debug, info, warn, error, fatal)")
	searchCmd.Flags().String("since", "", "show logs since timestamp (RFC3339 or relative like '1h')")
	searchCmd.Flags().String("until", "", "show logs until timestamp (RFC3339 or relative like '1h')")
	searchCmd.Flags().IntP("context", "C", 0, "number of context lines around matches")
	searchCmd.Flags().BoolP("count", "c", false, "only print count of matching lines")
	searchCmd.Flags().BoolP("invert", "V", false, "invert match (show non-matching lines)")

	_ = viper.BindPFlag("search.pattern", searchCmd.Flags().Lookup("pattern"))
	_ = viper.BindPFlag("search.level", searchCmd.Flags().Lookup("level"))

	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	pattern, _ := cmd.Flags().GetString("pattern")
	levelStr, _ := cmd.Flags().GetString("level")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	contextLines, _ := cmd.Flags().GetInt("context")
	countOnly, _ := cmd.Flags().GetBool("count")
	invert, _ := cmd.Flags().GetBool("invert")

	if invert && pattern == "" {
		return fmt.Errorf("--invert requires --pattern")
	}

	files, err := config.ExpandGlobs(args)
	if err != nil {
		return err
	}

	var re *regexp.Regexp
	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
	}

	levelFilter := config.LevelUnknown
	if levelStr != "" {
		levelFilter = config.ParseLevel(levelStr)
		if levelFilter == config.LevelUnknown {
			return fmt.Errorf("invalid level: %s", levelStr)
		}
	}

	var since time.Time
	if sinceStr != "" {
		since, err = config.ParseTimeRef(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
	}

	var until time.Time
	if untilStr != "" {
		until, err = config.ParseTimeRef(untilStr)
		if err != nil {
			return fmt.Errorf("invalid --until value: %w", err)
		}
	}

	format := output.ParseFormat(viper.GetString("format"))
	p := parser.New(viper.GetStringSlice("timestamp_formats"))
	multiFile := len(files) > 1

	if countOnly {
		return runSearchCount(cmd, p, files, searchFilterOptions{
			re:          re,
			invert:      invert,
			level:       levelFilter,
			since:       since,
			until:       until,
			levelActive: levelStr != "",
		}, multiFile)
	}

	if format == output.FormatJSON {
		return runSearchJSON(cmd, p, files, searchFilterOptions{
			re:          re,
			invert:      invert,
			level:       levelFilter,
			since:       since,
			until:       until,
			levelActive: levelStr != "",
		}, contextLines)
	}

	return runSearchTextOrTable(cmd, p, files, searchFilterOptions{
		re:          re,
		invert:      invert,
		level:       levelFilter,
		since:       since,
		until:       until,
		levelActive: levelStr != "",
	}, format, contextLines, multiFile)
}

type searchFilterOptions struct {
	re          *regexp.Regexp
	invert      bool
	level       config.LogLevel
	since       time.Time
	until       time.Time
	levelActive bool
}

func (opts searchFilterOptions) matches(entry config.LogEntry) bool {
	if opts.levelActive && entry.Level != opts.level {
		return false
	}

	if !opts.since.IsZero() && !entry.Timestamp.IsZero() && entry.Timestamp.Before(opts.since) {
		return false
	}

	if !opts.until.IsZero() && !entry.Timestamp.IsZero() && entry.Timestamp.After(opts.until) {
		return false
	}

	if opts.re != nil {
		matched := opts.re.MatchString(entry.Raw)
		if opts.invert {
			matched = !matched
		}
		if !matched {
			return false
		}
	}

	return true
}

type contextEmitter struct {
	context         int
	matchFn         func(config.LogEntry) bool
	emit            func(config.LogEntry) error
	emitSeparator   func() error
	lastEmittedLine int
	afterRemaining  int
	inContext       bool
	hasOutput       bool
	before          []config.LogEntry
}

func (c *contextEmitter) process(entry config.LogEntry) error {
	if c.context == 0 {
		if c.matchFn(entry) {
			if err := c.emit(entry); err != nil {
				return err
			}
			c.lastEmittedLine = entry.Line
			c.hasOutput = true
		}
		return nil
	}

	matched := c.matchFn(entry)
	if matched {
		if !c.inContext && c.hasOutput && c.emitSeparator != nil {
			if err := c.emitSeparator(); err != nil {
				return err
			}
		}

		for _, prev := range c.before {
			if prev.Line <= c.lastEmittedLine {
				continue
			}
			if err := c.emit(prev); err != nil {
				return err
			}
			c.lastEmittedLine = prev.Line
			c.hasOutput = true
		}

		if entry.Line > c.lastEmittedLine {
			if err := c.emit(entry); err != nil {
				return err
			}
			c.lastEmittedLine = entry.Line
			c.hasOutput = true
		}

		c.inContext = true
		c.afterRemaining = c.context
	} else if c.inContext {
		if entry.Line > c.lastEmittedLine {
			if err := c.emit(entry); err != nil {
				return err
			}
			c.lastEmittedLine = entry.Line
			c.hasOutput = true
		}
		c.afterRemaining--
		if c.afterRemaining <= 0 {
			c.inContext = false
		}
	}

	if c.context > 0 {
		c.before = append(c.before, entry)
		if len(c.before) > c.context {
			c.before = c.before[1:]
		}
	}

	return nil
}

func runSearchCount(cmd *cobra.Command, p *parser.Parser, files []string, opts searchFilterOptions, multiFile bool) error {
	for _, filePath := range files {
		count := 0
		err := p.ParseFileStream(filePath, func(entry config.LogEntry) error {
			if opts.matches(entry) {
				count++
			}
			return nil
		})
		if err != nil {
			return err
		}
		if multiFile {
			fmt.Fprintf(cmd.OutOrStdout(), "%s:%d\n", filePath, count)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d\n", count)
	}
	return nil
}

func runSearchJSON(cmd *cobra.Command, p *parser.Parser, files []string, opts searchFilterOptions, contextLines int) error {
	writer := output.New(cmd.OutOrStdout(), output.FormatJSON)

	if len(files) == 1 {
		entries, err := collectEntries(p, files[0], opts, contextLines)
		if err != nil {
			return err
		}
		return writer.WriteEntries(entries)
	}

	result := make(map[string][]config.LogEntry)
	for _, filePath := range files {
		entries, err := collectEntries(p, filePath, opts, contextLines)
		if err != nil {
			return err
		}
		result[filePath] = entries
	}

	return writer.WriteJSON(result)
}

func runSearchTextOrTable(cmd *cobra.Command, p *parser.Parser, files []string, opts searchFilterOptions, format output.Format, contextLines int, multiFile bool) error {
	if format == output.FormatTable {
		writer := output.New(cmd.OutOrStdout(), output.FormatTable)
		for _, filePath := range files {
			entries, err := collectEntries(p, filePath, opts, contextLines)
			if err != nil {
				return err
			}
			if multiFile {
				fmt.Fprintf(cmd.OutOrStdout(), "==> %s <==\n", filePath)
			}
			if err := writer.WriteEntries(entries); err != nil {
				return err
			}
		}
		return nil
	}

	prefix := func(filePath string, line string) string {
		if !multiFile {
			return line
		}
		return fmt.Sprintf("%s:%s", filePath, line)
	}

	for _, filePath := range files {
		emitter := &contextEmitter{
			context: contextLines,
			matchFn: opts.matches,
			emit: func(entry config.LogEntry) error {
				fmt.Fprintln(cmd.OutOrStdout(), prefix(filePath, entry.Raw))
				return nil
			},
			emitSeparator: func() error {
				fmt.Fprintln(cmd.OutOrStdout(), "--")
				return nil
			},
		}

		err := p.ParseFileStream(filePath, func(entry config.LogEntry) error {
			return emitter.process(entry)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func collectEntries(p *parser.Parser, filePath string, opts searchFilterOptions, contextLines int) ([]config.LogEntry, error) {
	var entries []config.LogEntry

	emitter := &contextEmitter{
		context: contextLines,
		matchFn: opts.matches,
		emit: func(entry config.LogEntry) error {
			entries = append(entries, entry)
			return nil
		},
	}

	err := p.ParseFileStream(filePath, func(entry config.LogEntry) error {
		return emitter.process(entry)
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}
