package cmd

import (
	"fmt"

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
	Args: cobra.ExactArgs(1),
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
	filePath := args[0]
	pattern, _ := cmd.Flags().GetString("pattern")
	level, _ := cmd.Flags().GetString("level")

	// TODO: Implement search logic using internal/parser and internal/analyzer
	fmt.Printf("Searching %s (pattern=%q, level=%q)\n", filePath, pattern, level)

	return nil
}
