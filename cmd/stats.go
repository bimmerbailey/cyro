package cmd

import (
	"fmt"
	"time"

	"github.com/bimmerbailey/cyro/internal/analyzer"
	"github.com/bimmerbailey/cyro/internal/config"
	"github.com/bimmerbailey/cyro/internal/output"
	"github.com/bimmerbailey/cyro/internal/parser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var statsCmd = &cobra.Command{
	Use:   "stats [flags] <file>",
	Short: "Show log file statistics",
	Long: `Display statistical summary of a log file including line counts,
level distribution, time range, error rates, and top messages.

Examples:
  cyro stats /var/log/app.log
  cyro stats --format json /var/log/app.log
  cyro stats --since "2024-01-01" app.log`,
	Args: cobra.ExactArgs(1),
	RunE: runStats,
}

func init() {
	statsCmd.Flags().String("since", "", "only include logs since timestamp (RFC3339 or relative like '1h')")
	statsCmd.Flags().String("until", "", "only include logs until timestamp (RFC3339 or relative like '1h')")
	statsCmd.Flags().Int("top", 10, "number of top messages to show")

	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	topN, _ := cmd.Flags().GetInt("top")

	var since time.Time
	var err error
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

	p := parser.New(viper.GetStringSlice("timestamp_formats"))

	var entries []config.LogEntry
	err = p.ParseFileStream(filePath, func(entry config.LogEntry) error {
		if !since.IsZero() && !entry.Timestamp.IsZero() && entry.Timestamp.Before(since) {
			return nil
		}
		if !until.IsZero() && !entry.Timestamp.IsZero() && entry.Timestamp.After(until) {
			return nil
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return err
	}

	anlz := analyzer.New()
	stats := anlz.ComputeStats(entries, topN)

	format := output.ParseFormat(viper.GetString("format"))

	switch format {
	case output.FormatJSON:
		return outputStatsJSON(cmd, stats)
	case output.FormatTable:
		return outputStatsTable(cmd, stats)
	default:
		return outputStatsText(cmd, filePath, stats)
	}
}

func outputStatsJSON(cmd *cobra.Command, stats analyzer.Stats) error {
	writer := output.New(cmd.OutOrStdout(), output.FormatJSON)
	return writer.WriteJSON(stats)
}

func outputStatsTable(cmd *cobra.Command, stats analyzer.Stats) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Total Lines: %d\n\n", stats.TotalLines)

	fmt.Fprintln(cmd.OutOrStdout(), "Level Distribution:")
	fmt.Fprintln(cmd.OutOrStdout(), "LEVEL\tCOUNT\tPERCENTAGE")
	fmt.Fprintln(cmd.OutOrStdout(), "-----\t-----\t----------")
	levels := []config.LogLevel{
		config.LevelFatal,
		config.LevelError,
		config.LevelWarn,
		config.LevelInfo,
		config.LevelDebug,
		config.LevelUnknown,
	}
	for _, level := range levels {
		count := stats.LevelCounts[level]
		if count > 0 {
			percent := float64(count) * 100 / float64(stats.TotalLines)
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\t%.1f%%\n", level, count, percent)
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nError Rate: %.2f%%\n\n", stats.ErrorRate*100)

	if !stats.FirstEntry.IsZero() {
		fmt.Fprintf(cmd.OutOrStdout(), "Time Range: %s to %s\n\n",
			stats.FirstEntry.Format("2006-01-02 15:04:05"),
			stats.LastEntry.Format("2006-01-02 15:04:05"))
	}

	if len(stats.TopMessages) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Top Messages:")
		fmt.Fprintln(cmd.OutOrStdout(), "COUNT\tMESSAGE")
		fmt.Fprintln(cmd.OutOrStdout(), "-----\t-------")
		for _, msg := range stats.TopMessages {
			msgStr := msg.Message
			if len(msgStr) > 60 {
				msgStr = msgStr[:57] + "..."
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\n", msg.Count, msgStr)
		}
	}

	return nil
}

func outputStatsText(cmd *cobra.Command, filePath string, stats analyzer.Stats) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Statistics for %s:\n", filePath)
	fmt.Fprintf(cmd.OutOrStdout(), "  Total Lines: %d\n", stats.TotalLines)

	fmt.Fprintln(cmd.OutOrStdout(), "\n  Level Distribution:")
	levels := []config.LogLevel{
		config.LevelFatal,
		config.LevelError,
		config.LevelWarn,
		config.LevelInfo,
		config.LevelDebug,
		config.LevelUnknown,
	}
	for _, level := range levels {
		count := stats.LevelCounts[level]
		if count > 0 {
			percent := float64(count) * 100 / float64(stats.TotalLines)
			fmt.Fprintf(cmd.OutOrStdout(), "    %s: %d (%.1f%%)\n", level, count, percent)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n  Error Rate: %.2f%%\n", stats.ErrorRate*100)

	if !stats.FirstEntry.IsZero() {
		fmt.Fprintf(cmd.OutOrStdout(), "\n  Time Range:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "    First Entry: %s\n", stats.FirstEntry.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(cmd.OutOrStdout(), "    Last Entry:  %s\n", stats.LastEntry.Format("2006-01-02 15:04:05"))
		duration := stats.LastEntry.Sub(stats.FirstEntry)
		fmt.Fprintf(cmd.OutOrStdout(), "    Duration:    %s\n", duration)
	}

	if len(stats.TopMessages) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\n  Top Messages:")
		for i, msg := range stats.TopMessages {
			fmt.Fprintf(cmd.OutOrStdout(), "    %d. [%d] %s\n", i+1, msg.Count, msg.Message)
		}
	}

	return nil
}
