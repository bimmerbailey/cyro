package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bimmerbailey/cyro/internal/analyzer"
	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/output"
	"github.com/bimmerbailey/cyro/internal/parser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [flags] <file>",
	Short: "Analyze log files for patterns and anomalies",
	Long: `Perform deep analysis on log files to detect patterns, anomalies,
error frequency spikes, and recurring issues.

Examples:
  cyro analyze /var/log/app.log
  cyro analyze --top 20 --group-by level /var/log/app.log
  cyro analyze --pattern "connection refused" --window 5m app.log`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().Int("top", 10, "number of top results to show")
	analyzeCmd.Flags().String("group-by", "message", "group results by field (level, message, source)")
	analyzeCmd.Flags().StringP("pattern", "p", "", "focus analysis on entries matching pattern")
	analyzeCmd.Flags().String("window", "", "time window for trend analysis (e.g., 5m, 1h)")

	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
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
