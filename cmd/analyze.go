package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bimmerbailey/cyro/internal/analyzer"
	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/llm"
	"github.com/bimmerbailey/cyro/internal/output"
	"github.com/bimmerbailey/cyro/internal/parser"
	"github.com/bimmerbailey/cyro/internal/preprocess"
	"github.com/bimmerbailey/cyro/internal/prompt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [flags] <file>",
	Short: "Analyze log files for patterns and anomalies",
	Long: `Perform deep analysis on log files to detect patterns, anomalies,
error frequency spikes, and recurring issues.

With --ai flag, uses an LLM to provide natural language analysis and insights.
Without --ai, provides statistical analysis of log patterns.

Examples:
  cyro analyze /var/log/app.log
  cyro analyze --ai /var/log/app.log
  cyro analyze --ai --pattern "error" /var/log/app.log
  cyro analyze --top 20 --group-by level /var/log/app.log
  cyro analyze --window 5m app.log`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().Bool("ai", false, "enable AI-powered analysis using LLM")
	analyzeCmd.Flags().Int("top", 10, "number of top results to show")
	analyzeCmd.Flags().String("group-by", "message", "group results by field (level, message, source)")
	analyzeCmd.Flags().StringP("pattern", "p", "", "focus analysis on entries matching pattern")
	analyzeCmd.Flags().String("window", "", "time window for trend analysis (e.g., 5m, 1h)")

	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	aiEnabled, _ := cmd.Flags().GetBool("ai")
	topN, _ := cmd.Flags().GetInt("top")
	groupBy, _ := cmd.Flags().GetString("group-by")
	pattern, _ := cmd.Flags().GetString("pattern")
	windowStr, _ := cmd.Flags().GetString("window")

	// Validate group-by field
	validGroupFields := map[string]bool{"level": true, "message": true, "source": true}
	if !validGroupFields[groupBy] {
		return fmt.Errorf("invalid --group-by value: %s (must be 'level', 'message', or 'source')", groupBy)
	}

	// Expand file globs
	files, err := config.ExpandGlobs(args)
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

	// Parse window duration if provided
	var window time.Duration
	if windowStr != "" {
		window, err = config.ParseDuration(windowStr)
		if err != nil {
			return fmt.Errorf("invalid --window value: %w", err)
		}
		if window <= 0 {
			return fmt.Errorf("window duration must be positive")
		}
	}

	format := output.ParseFormat(viper.GetString("format"))
	p := parser.New(viper.GetStringSlice("timestamp_formats"))
	anlz := analyzer.New()

	// Collect all matching entries
	var allEntries []config.LogEntry
	multiFile := len(files) > 1

	for _, file := range files {
		err = p.ParseFileStream(file, func(entry config.LogEntry) error {
			// Apply pattern filter
			if re != nil && !re.MatchString(entry.Raw) {
				return nil
			}
			allEntries = append(allEntries, entry)
			return nil
		})
		if err != nil {
			return err
		}
	}

	if len(allEntries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No matching entries found.")
		return nil
	}

	// Route to AI analysis if requested
	if aiEnabled {
		return runAIAnalyze(cmd, allEntries, files, pattern, groupBy, windowStr)
	}

	// Build analysis result
	result := analyzer.AnalysisResult{
		TotalLines: len(allEntries),
		GroupBy:    groupBy,
		Pattern:    pattern,
	}

	// If window analysis requested
	if window > 0 {
		result.TimeWindows = anlz.AnalyzeByWindow(allEntries, window)
		if len(result.TimeWindows) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No timestamp information available for window analysis.")
			return nil
		}
	}

	// Group results
	result.Groups, err = anlz.GroupBy(allEntries, groupBy, topN)
	if err != nil {
		return err
	}

	// Output based on format
	switch format {
	case output.FormatJSON:
		return outputAnalysisJSON(cmd, result, files, multiFile)
	case output.FormatTable:
		return outputAnalysisTable(cmd, result, window > 0)
	default:
		return outputAnalysisText(cmd, result, files, window > 0, multiFile)
	}
}

func outputAnalysisJSON(cmd *cobra.Command, result analyzer.AnalysisResult, files []string, multiFile bool) error {
	if multiFile {
		result.FilePath = strings.Join(files, ", ")
	} else if len(files) == 1 {
		result.FilePath = files[0]
	}

	writer := output.New(cmd.OutOrStdout(), output.FormatJSON)
	return writer.WriteJSON(result)
}

func outputAnalysisTable(cmd *cobra.Command, result analyzer.AnalysisResult, hasWindow bool) error {
	// Print grouped results
	fmt.Fprintf(cmd.OutOrStdout(), "Analysis Results (Grouped by %s):\n\n", result.GroupBy)
	fmt.Fprintln(cmd.OutOrStdout(), "RANK\tCOUNT\tPERCENT\tKEY")
	fmt.Fprintln(cmd.OutOrStdout(), "----\t-----\t-------\t---")

	for i, group := range result.Groups {
		key := group.Key
		if len(key) > 50 {
			key = key[:47] + "..."
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d\t%d\t%.1f%%\t%s\n", i+1, group.Count, group.Percent, key)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nTotal entries: %d\n", result.TotalLines)

	// Print time window analysis if present
	if hasWindow && len(result.TimeWindows) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nTrend Analysis (Time Windows):")
		fmt.Fprintln(cmd.OutOrStdout(), "START\t\tEND\t\tCOUNT\tERRORS\tCHANGE")
		fmt.Fprintln(cmd.OutOrStdout(), "-----\t\t---\t\t-----\t------\t------")

		for _, win := range result.TimeWindows {
			changeStr := "-"
			if win.ChangePercent != 0 {
				if win.ChangePercent > 0 {
					changeStr = fmt.Sprintf("↑ %.1f%%", win.ChangePercent)
				} else {
					changeStr = fmt.Sprintf("↓ %.1f%%", -win.ChangePercent)
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d\t%d\t%s\n",
				win.Start.Format("15:04:05"),
				win.End.Format("15:04:05"),
				win.Count,
				win.ErrorCount,
				changeStr)
		}
	}

	return nil
}

func outputAnalysisText(cmd *cobra.Command, result analyzer.AnalysisResult, files []string, hasWindow bool, multiFile bool) error {
	// Header
	if multiFile {
		fmt.Fprintf(cmd.OutOrStdout(), "Analysis of %d files (%d entries)\n", len(files), result.TotalLines)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Analysis of %s (%d entries)\n", files[0], result.TotalLines)
	}

	if result.Pattern != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Pattern: %s\n", result.Pattern)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Grouped by: %s\n\n", result.GroupBy)

	// Top results
	fmt.Fprintln(cmd.OutOrStdout(), "Top Results:")
	for i, group := range result.Groups {
		key := group.Key
		if len(key) > 80 {
			key = key[:77] + "..."
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %2d. %-8s entries (%.1f%%) - %s\n",
			i+1,
			fmt.Sprintf("%d", group.Count),
			group.Percent,
			key)
	}

	// Time window analysis
	if hasWindow && len(result.TimeWindows) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nTrend Analysis:")
		for i, win := range result.TimeWindows {
			if win.Count == 0 {
				continue
			}

			changeStr := ""
			if i > 0 {
				if win.ChangePercent > 0 {
					changeStr = fmt.Sprintf(" ↑ %.1f%%", win.ChangePercent)
				} else if win.ChangePercent < 0 {
					changeStr = fmt.Sprintf(" ↓ %.1f%%", -win.ChangePercent)
				}
			}

			timeStr := fmt.Sprintf("%s - %s", win.Start.Format("15:04:05"), win.End.Format("15:04:05"))
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: %d entries%s\n", timeStr, win.Count, changeStr)
			if win.ErrorCount > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "    Errors: %d (%.1f%%)\n", win.ErrorCount, win.ErrorPercent)
			}
		}
	}

	return nil
}

// runAIAnalyze handles AI-powered log analysis.
func runAIAnalyze(
	cmd *cobra.Command,
	entries []config.LogEntry,
	files []string,
	pattern string,
	groupBy string,
	windowStr string,
) error {
	format := output.ParseFormat(viper.GetString("format"))
	verbose := viper.GetBool("verbose")
	ctx := context.Background()

	// 1. Validate format
	if format == output.FormatTable {
		return fmt.Errorf("table format not supported with --ai (use 'text' or 'json')")
	}

	// Print preprocessing message for text format
	if format == output.FormatText && verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Preprocessing %d log entries...\n\n", len(entries))
	}

	// 2. Initialize preprocessing
	preprocessor := preprocess.New(
		preprocess.WithTokenLimit(8000),
		preprocess.WithRedaction(viper.GetBool("redaction.enabled")),
		preprocess.WithRedactionPatterns(viper.GetStringSlice("redaction.patterns")),
	)

	preprocessOutput, stats, err := preprocessor.ProcessWithStats(entries)
	if err != nil {
		return fmt.Errorf("preprocessing failed: %w", err)
	}

	// 3. Initialize LLM provider
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

	// 4. Build prompts
	messages, err := prompt.Build(prompt.TypeSummarize, prompt.BuildOptions{
		Summary: preprocessOutput.Summary,
		Pattern: pattern,
		GroupBy: groupBy,
		Window:  windowStr,
		Files:   files,
	})
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
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

	// 5. Stream LLM response
	stream, err := provider.ChatStream(ctx, messages, chatOpts)
	if err != nil {
		return fmt.Errorf("failed to start LLM stream: %w", err)
	}

	// Print header for text format
	if format == output.FormatText {
		fmt.Fprintln(cmd.OutOrStdout(), "=== AI-Powered Log Analysis ===")
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

	// 6. Handle JSON output
	if format == output.FormatJSON {
		analysisResult := map[string]interface{}{
			"files":          files,
			"total_lines":    len(entries),
			"templates":      preprocessOutput.TotalTemplates,
			"redacted_count": preprocessOutput.RedactedCount,
			"time_range": map[string]interface{}{
				"start": preprocessOutput.TimeRange.Start,
				"end":   preprocessOutput.TimeRange.End,
			},
			"ai_analysis": fullResponse.String(),
			"metadata":    preprocessOutput.Metadata,
		}

		writer := output.New(cmd.OutOrStdout(), output.FormatJSON)
		return writer.WriteJSON(analysisResult)
	}

	// 7. Show verbose stats if requested (text format only)
	if verbose && format == output.FormatText {
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
