package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/llm"
	"github.com/bimmerbailey/cyro/internal/output"
	"github.com/bimmerbailey/cyro/internal/parser"
	"github.com/bimmerbailey/cyro/internal/preprocess"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var askCmd = &cobra.Command{
	Use:   "ask <question> --file <logfile>",
	Short: "Ask natural language questions about log files using AI",
	Long: `Ask natural language questions about log files using AI-powered analysis.

The ask command preprocesses log files to extract patterns and reduce token usage,
then sends a question along with the compressed log context to an LLM for answers.

Examples:
  cyro ask "what caused the errors?" --file app.log
  cyro ask "when did the database timeout occur?" --file /var/log/app.log --level error
  cyro ask "what happened between 2pm and 3pm?" --file app.log --since 2h --until 1h
  cyro ask "why did authentication fail?" --file auth.log --file api.log --pattern "auth"`,
	Args: cobra.ExactArgs(1),
	RunE: runAsk,
}

func init() {
	askCmd.Flags().StringSliceP("file", "F", []string{}, "log file(s) to analyze (required, repeatable)")
	askCmd.Flags().StringP("pattern", "p", "", "pre-filter logs matching regex pattern")
	askCmd.Flags().String("level", "", "filter by log level (debug, info, warn, error, fatal)")
	askCmd.Flags().String("since", "", "filter logs after this time (relative duration like '1h', '30m', or absolute like '2024-02-13T14:00:00')")
	askCmd.Flags().String("until", "", "filter logs before this time (relative duration or absolute timestamp)")

	_ = askCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(askCmd)
}

func runAsk(cmd *cobra.Command, args []string) error {
	question := args[0]
	files, _ := cmd.Flags().GetStringSlice("file")
	pattern, _ := cmd.Flags().GetString("pattern")
	levelStr, _ := cmd.Flags().GetString("level")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")

	format := output.ParseFormat(viper.GetString("format"))
	verbose := viper.GetBool("verbose")
	ctx := context.Background()

	// Expand file globs
	expandedFiles, err := config.ExpandGlobs(files)
	if err != nil {
		return err
	}

	// Compile pattern if provided
	var re *regexp.Regexp
	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
	}

	// Parse level filter if provided
	var levelFilter config.LogLevel
	if levelStr != "" {
		levelFilter = config.ParseLevel(levelStr)
		if levelFilter == config.LevelUnknown && levelStr != "unknown" {
			return fmt.Errorf("invalid level filter: %s (must be one of: debug, info, warn, error, fatal)", levelStr)
		}
	}

	// Parse time range filters
	var sinceTime, untilTime time.Time
	if sinceStr != "" {
		sinceTime, err = config.ParseTimeRef(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
	}
	if untilStr != "" {
		untilTime, err = config.ParseTimeRef(untilStr)
		if err != nil {
			return fmt.Errorf("invalid --until value: %w", err)
		}
	}

	// Parse all files and collect entries
	p := parser.New(viper.GetStringSlice("timestamp_formats"))
	var allEntries []config.LogEntry
	multiFile := len(expandedFiles) > 1

	for _, file := range expandedFiles {
		err = p.ParseFileStream(file, func(entry config.LogEntry) error {
			// Apply pattern filter
			if re != nil && !re.MatchString(entry.Raw) {
				return nil
			}

			// Apply level filter
			if levelStr != "" && entry.Level != levelFilter {
				return nil
			}

			// Apply since filter
			if !sinceTime.IsZero() && !entry.Timestamp.IsZero() && entry.Timestamp.Before(sinceTime) {
				return nil
			}

			// Apply until filter
			if !untilTime.IsZero() && !entry.Timestamp.IsZero() && entry.Timestamp.After(untilTime) {
				return nil
			}

			allEntries = append(allEntries, entry)
			return nil
		})
		if err != nil {
			return fmt.Errorf("error parsing %s: %w", file, err)
		}
	}

	if len(allEntries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No log entries matched your filters. Try broader criteria.")
		return nil
	}

	// Preprocessing
	if format == output.FormatText && verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Preprocessing %d log entries...\n\n", len(allEntries))
	}

	preprocessor := preprocess.New(
		preprocess.WithTokenLimit(8000),
		preprocess.WithRedaction(viper.GetBool("redaction.enabled")),
		preprocess.WithRedactionPatterns(viper.GetStringSlice("redaction.patterns")),
	)

	preprocessOutput, stats, err := preprocessor.ProcessWithStats(allEntries)
	if err != nil {
		return fmt.Errorf("preprocessing failed: %w", err)
	}

	// Initialize LLM provider
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	if verbose {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	cfg := &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	provider, err := llm.NewProvider(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w\n\nTroubleshooting:\n- Ensure Ollama is running: ollama serve\n- Check provider config in ~/.cyro.yaml\n- For cloud providers, verify API keys are set", err)
	}

	// Health check
	if err := provider.Heartbeat(ctx); err != nil {
		if cfg.LLM.Provider == "ollama" {
			return fmt.Errorf("cannot connect to Ollama at %s: %w\n\nStart Ollama with: ollama serve",
				cfg.LLM.Ollama.Host, err)
		}
		return fmt.Errorf("LLM provider %s unavailable: %w", cfg.LLM.Provider, err)
	}

	// Build prompts
	messages := []llm.Message{
		{Role: "system", Content: buildAskSystemPrompt()},
		{Role: "user", Content: buildAskUserPrompt(question, preprocessOutput)},
	}

	chatOpts := &llm.ChatOptions{
		Temperature: float32(cfg.LLM.Temperature),
		MaxTokens:   cfg.LLM.MaxTokens,
	}

	// Set model based on provider
	switch cfg.LLM.Provider {
	case "ollama":
		chatOpts.Model = cfg.LLM.Ollama.Model
	case "openai":
		chatOpts.Model = cfg.LLM.OpenAI.Model
	case "anthropic":
		chatOpts.Model = cfg.LLM.Anthropic.Model
	}

	// Stream LLM response
	stream, err := provider.ChatStream(ctx, messages, chatOpts)
	if err != nil {
		return fmt.Errorf("failed to start LLM stream: %w", err)
	}

	// Print header for text format
	if format == output.FormatText {
		fmt.Fprintln(cmd.OutOrStdout(), "=== Answer ===")
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Stream tokens
	var fullResponse strings.Builder
	for event := range stream {
		if event.Error != nil {
			if fullResponse.Len() > 0 {
				fmt.Fprintf(os.Stderr, "\n\nError during streaming: %v\n", event.Error)
			}
			return event.Error
		}

		if event.Content != "" {
			if format == output.FormatText {
				fmt.Fprint(cmd.OutOrStdout(), event.Content)
			}
			fullResponse.WriteString(event.Content)
		}
	}

	// Handle JSON output
	if format == output.FormatJSON {
		var filesField interface{}
		if multiFile {
			filesField = expandedFiles
		} else if len(expandedFiles) == 1 {
			filesField = expandedFiles[0]
		}

		askResult := map[string]interface{}{
			"question":       question,
			"files":          filesField,
			"total_lines":    len(allEntries),
			"templates":      preprocessOutput.TotalTemplates,
			"redacted_count": preprocessOutput.RedactedCount,
			"time_range": map[string]interface{}{
				"start": preprocessOutput.TimeRange.Start,
				"end":   preprocessOutput.TimeRange.End,
			},
			"answer": fullResponse.String(),
			"metadata": map[string]interface{}{
				"provider": cfg.LLM.Provider,
				"model":    chatOpts.Model,
				"filters": map[string]string{
					"pattern": pattern,
					"level":   levelStr,
					"since":   sinceStr,
					"until":   untilStr,
				},
			},
		}

		writer := output.New(cmd.OutOrStdout(), output.FormatJSON)
		if err := writer.WriteJSON(askResult); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}
		return nil
	}

	// Show verbose stats for text format
	if verbose {
		fmt.Fprintln(cmd.OutOrStdout(), "\n\n=== Preprocessing Statistics ===")
		fmt.Fprintf(cmd.OutOrStdout(), "Input: %d lines\n", stats.InputLines)
		fmt.Fprintf(cmd.OutOrStdout(), "Templates extracted: %d\n", stats.OutputTemplates)
		fmt.Fprintf(cmd.OutOrStdout(), "Compression ratio: %.1fx\n", stats.CompressionRatio)
		fmt.Fprintf(cmd.OutOrStdout(), "Secrets redacted: %d\n", stats.RedactedCount)
		fmt.Fprintf(cmd.OutOrStdout(), "Tokens sent to LLM: %d/%d (%.1f%%)\n",
			stats.TokenCount, stats.TokenLimit,
			float64(stats.TokenCount)/float64(stats.TokenLimit)*100)
	}

	return nil
}
