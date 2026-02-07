package cmd

import (
	"fmt"

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
  cyro tail --pattern "request_id=abc" --level warn app.log`,
	Args: cobra.ExactArgs(1),
	RunE: runTail,
}

func init() {
	tailCmd.Flags().StringP("pattern", "p", "", "only show lines matching regex pattern")
	tailCmd.Flags().StringP("level", "l", "", "minimum log level to display (debug, info, warn, error, fatal)")
	tailCmd.Flags().IntP("lines", "n", 10, "number of initial lines to show")
	tailCmd.Flags().Bool("no-follow", false, "print last N lines and exit (don't follow)")

	rootCmd.AddCommand(tailCmd)
}

func runTail(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	level, _ := cmd.Flags().GetString("level")
	lines, _ := cmd.Flags().GetInt("lines")

	// TODO: Implement tail logic using internal/parser with fsnotify or polling
	fmt.Printf("Tailing %s (level=%q, lines=%d)\n", filePath, level, lines)

	return nil
}
