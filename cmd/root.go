package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "cyro",
	Short: "A powerful log analysis tool",
	Long: `Cyro is a CLI tool for analyzing, searching, and monitoring log files.

It supports multiple log formats and provides powerful filtering,
statistical analysis, and real-time tailing capabilities.

Examples:
  cyro search --level error /var/log/app.log
  cyro analyze --pattern "timeout" /var/log/app.log
  cyro stats /var/log/app.log
  cyro tail --level warn /var/log/app.log`,
}

// Execute is called by main.main(). It runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cyro.yaml)")
	rootCmd.PersistentFlags().StringP("format", "f", "text", "output format (text, json, table)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")

	_ = viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error finding home directory:", err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".cyro")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("CYRO")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("format", "text")
	viper.SetDefault("verbose", false)
	viper.SetDefault("timestamp_formats", []string{
		"2006-01-02T15:04:05Z07:00",  // RFC3339
		"2006-01-02 15:04:05",        // Common datetime
		"Jan 02 15:04:05",            // Syslog
		"02/Jan/2006:15:04:05 -0700", // Apache/Nginx
	})
	viper.SetDefault("log_dir", filepath.Join(".", "logs"))

	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
