package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bimmerbailey/cyro/internal/analyzer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newStatsTestCmd(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{Use: "stats"}
	cmd.SetOut(out)
	cmd.Flags().String("since", "", "only include logs since timestamp")
	cmd.Flags().String("until", "", "only include logs until timestamp")
	cmd.Flags().Int("top", 10, "number of top messages to show")
	return cmd
}

func TestStatsBasicText(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"second"}`,
		`{"timestamp":"2025-01-26T10:00:03Z","level":"error","message":"boom"}`,
		`{"timestamp":"2025-01-26T10:00:04Z","level":"warn","message":"warning"}`,
	})

	var out bytes.Buffer
	cmd := newStatsTestCmd(&out)

	if err := runStats(cmd, []string{file}); err != nil {
		t.Fatalf("runStats() error = %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "Total Lines: 5") {
		t.Errorf("expected Total Lines: 5, got:\n%s", output)
	}

	if !strings.Contains(output, "Error Rate: 40.00%") {
		t.Errorf("expected Error Rate: 40.00%%, got:\n%s", output)
	}

	if !strings.Contains(output, "boom [2]") && !strings.Contains(output, "[2] boom") {
		t.Errorf("expected top message 'boom' with count 2, got:\n%s", output)
	}
}

func TestStatsJSON(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"error1"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"error","message":"error2"}`,
	})

	var out bytes.Buffer
	cmd := newStatsTestCmd(&out)

	if err := runStats(cmd, []string{file}); err != nil {
		t.Fatalf("runStats() error = %v", err)
	}

	var stats analyzer.Stats
	if err := json.Unmarshal(out.Bytes(), &stats); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\noutput: %s", err, out.String())
	}

	if stats.TotalLines != 3 {
		t.Errorf("expected TotalLines=3, got %d", stats.TotalLines)
	}

	if stats.ErrorRate != 0.6666666666666666 {
		t.Errorf("expected ErrorRate=0.67, got %f", stats.ErrorRate)
	}

	if len(stats.TopMessages) != 3 {
		t.Errorf("expected 3 top messages, got %d", len(stats.TopMessages))
	}
}

func TestStatsTable(t *testing.T) {
	viper.Reset()
	viper.Set("format", "table")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
	})

	var out bytes.Buffer
	cmd := newStatsTestCmd(&out)

	if err := runStats(cmd, []string{file}); err != nil {
		t.Fatalf("runStats() error = %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "Total Lines: 2") {
		t.Errorf("expected Total Lines: 2 in output, got:\n%s", output)
	}

	if !strings.Contains(output, "ERROR") {
		t.Errorf("expected ERROR in level distribution, got:\n%s", output)
	}

	if !strings.Contains(output, "INFO") {
		t.Errorf("expected INFO in level distribution, got:\n%s", output)
	}
}

func TestStatsTimeRangeFilter(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"info","message":"second"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"third"}`,
		`{"timestamp":"2025-01-26T10:00:03Z","level":"info","message":"fourth"}`,
	})

	var out bytes.Buffer
	cmd := newStatsTestCmd(&out)
	if err := cmd.Flags().Set("since", "2025-01-26T10:00:01.5Z"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := cmd.Flags().Set("until", "2025-01-26T10:00:03Z"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runStats(cmd, []string{file}); err != nil {
		t.Fatalf("runStats() error = %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "Total Lines: 2") {
		t.Errorf("expected Total Lines: 2 (filtered), got:\n%s", output)
	}

	if !strings.Contains(output, "First Entry: 2025-01-26 10:00:02") {
		t.Errorf("expected first entry at 10:00:02, got:\n%s", output)
	}
}

func TestStatsTopN(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"unique1"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"info","message":"common"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"unique2"}`,
		`{"timestamp":"2025-01-26T10:00:03Z","level":"info","message":"common"}`,
		`{"timestamp":"2025-01-26T10:00:04Z","level":"info","message":"unique3"}`,
	})

	var out bytes.Buffer
	cmd := newStatsTestCmd(&out)
	if err := cmd.Flags().Set("top", "2"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runStats(cmd, []string{file}); err != nil {
		t.Fatalf("runStats() error = %v", err)
	}

	var stats analyzer.Stats
	if err := json.Unmarshal(out.Bytes(), &stats); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(stats.TopMessages) != 2 {
		t.Errorf("expected 2 top messages (top flag), got %d", len(stats.TopMessages))
	}

	if stats.TopMessages[0].Message != "common" || stats.TopMessages[0].Count != 2 {
		t.Errorf("expected first top message to be 'common' with count 2, got %s/%d",
			stats.TopMessages[0].Message, stats.TopMessages[0].Count)
	}
}

func TestStatsEmptyFile(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "empty.log", []string{})

	var out bytes.Buffer
	cmd := newStatsTestCmd(&out)

	if err := runStats(cmd, []string{file}); err != nil {
		t.Fatalf("runStats() error = %v", err)
	}

	var stats analyzer.Stats
	if err := json.Unmarshal(out.Bytes(), &stats); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if stats.TotalLines != 0 {
		t.Errorf("expected TotalLines=0 for empty file, got %d", stats.TotalLines)
	}

	if len(stats.LevelCounts) != 0 {
		t.Errorf("expected empty LevelCounts for empty file, got %v", stats.LevelCounts)
	}
}

func TestStatsErrorRateWithFatal(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"info1"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"error1"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"fatal","message":"fatal1"}`,
	})

	var out bytes.Buffer
	cmd := newStatsTestCmd(&out)

	if err := runStats(cmd, []string{file}); err != nil {
		t.Fatalf("runStats() error = %v", err)
	}

	var stats analyzer.Stats
	if err := json.Unmarshal(out.Bytes(), &stats); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	expectedRate := 2.0 / 3.0
	if stats.ErrorRate != expectedRate {
		t.Errorf("expected ErrorRate=%f (1 error + 1 fatal / 3 total), got %f", expectedRate, stats.ErrorRate)
	}
}

func TestStatsLevelDistribution(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"debug","message":"debug1"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"info","message":"info1"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"info2"}`,
		`{"timestamp":"2025-01-26T10:00:03Z","level":"warn","message":"warn1"}`,
		`{"timestamp":"2025-01-26T10:00:04Z","level":"error","message":"error1"}`,
		`{"timestamp":"2025-01-26T10:00:05Z","level":"fatal","message":"fatal1"}`,
	})

	var out bytes.Buffer
	cmd := newStatsTestCmd(&out)

	if err := runStats(cmd, []string{file}); err != nil {
		t.Fatalf("runStats() error = %v", err)
	}

	var stats analyzer.Stats
	if err := json.Unmarshal(out.Bytes(), &stats); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if stats.LevelCounts[0] != 1 { // DEBUG
		t.Errorf("expected 1 DEBUG, got %d", stats.LevelCounts[0])
	}
	if stats.LevelCounts[1] != 2 { // INFO
		t.Errorf("expected 2 INFO, got %d", stats.LevelCounts[1])
	}
	if stats.LevelCounts[2] != 1 { // WARN
		t.Errorf("expected 1 WARN, got %d", stats.LevelCounts[2])
	}
	if stats.LevelCounts[3] != 1 { // ERROR
		t.Errorf("expected 1 ERROR, got %d", stats.LevelCounts[3])
	}
	if stats.LevelCounts[4] != 1 { // FATAL
		t.Errorf("expected 1 FATAL, got %d", stats.LevelCounts[4])
	}
}
