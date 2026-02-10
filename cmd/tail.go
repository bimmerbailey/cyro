package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/output"
	"github.com/bimmerbailey/cyro/internal/tail"
	"github.com/spf13/cobra"
)

var tailCmd = &cobra.Command{
	Use:   "tail [flags] <file>",
	Short: "Live-tail a log file with filtering",
	Long: `Watch a log file in real-time, similar to 'tail -f' but with
built-in filtering by log level, pattern matching, and formatted output.

Examples:
  cyro tail /var/log/app.log
  cyro tail --level error /var/log/app.log
  cyro tail --pattern "request_id=abc" --level warn app.log
  cyro tail --follow-rotate /var/log/app.log`,
	Args: cobra.ExactArgs(1),
	RunE: runTail,
}

func init() {
	tailCmd.Flags().StringP("pattern", "p", "", "only show lines matching regex pattern")
	tailCmd.Flags().StringP("level", "l", "", "minimum log level to display (debug, info, warn, error, fatal)")
	tailCmd.Flags().IntP("lines", "n", 10, "number of initial lines to show")
	tailCmd.Flags().Bool("no-follow", false, "print last N lines and exit (don't follow)")
	tailCmd.Flags().Bool("follow-rotate", false, "follow through log rotations (continue when file is renamed/removed)")
	tailCmd.Flags().Bool("no-color", false, "disable colored output")

	rootCmd.AddCommand(tailCmd)
}

func runTail(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	levelStr, _ := cmd.Flags().GetString("level")
	lines, _ := cmd.Flags().GetInt("lines")
	noFollow, _ := cmd.Flags().GetBool("no-follow")
	followRotate, _ := cmd.Flags().GetBool("follow-rotate")
	noColor, _ := cmd.Flags().GetBool("no-color")
	patternStr, _ := cmd.Flags().GetString("pattern")

	// Validate file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Parse pattern if provided
	var pattern *regexp.Regexp
	var err error
	if patternStr != "" {
		pattern, err = regexp.Compile(patternStr)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
	}

	// Parse level filter
	levelFilter := config.LevelUnknown
	if levelStr != "" {
		levelFilter = config.ParseLevel(levelStr)
		if levelFilter == config.LevelUnknown {
			return fmt.Errorf("invalid level: %s", levelStr)
		}
	}

	// Determine color mode
	colorMode := output.ColorAuto
	if noColor {
		colorMode = output.ColorNever
	}

	// Create output function
	outputFunc := func(entry config.LogEntry) error {
		return output.New(os.Stdout, output.FormatText).WriteColoredEntry(entry, colorMode)
	}

	// Create tailer
	tailer := tail.New(tail.Options{
		FilePath:     filePath,
		Lines:        lines,
		Follow:       !noFollow,
		FollowRotate: followRotate,
		Pattern:      pattern,
		LevelFilter:  levelFilter,
		OutputFunc:   outputFunc,
	})

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run tailer in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- tailer.Run(ctx)
	}()

	// Wait for either completion or signal
	select {
	case <-sigChan:
		// Signal received, cancel context and wait for tailer to finish
		cancel()
		// Give it a moment to clean up
		<-errChan
		return nil
	case err := <-errChan:
		// Tailer finished (or errored)
		if err != nil && err.Error() != "file rotated" {
			return err
		}
		return nil
	}
}
