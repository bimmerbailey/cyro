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

func newAnalyzeTestCmd(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{Use: "analyze"}
	cmd.SetOut(out)
	cmd.Flags().Int("top", 10, "number of top results to show")
	cmd.Flags().String("group-by", "message", "group results by field")
	cmd.Flags().StringP("pattern", "p", "", "focus analysis on entries matching pattern")
	cmd.Flags().String("window", "", "time window for trend analysis")
	return cmd
}

func TestAnalyzeBasicText(t *testing.T) {
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
	cmd := newAnalyzeTestCmd(&out)

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "5 entries") {
		t.Errorf("expected '5 entries', got:\n%s", output)
	}

	if !strings.Contains(output, "Grouped by: message") {
		t.Errorf("expected 'Grouped by: message', got:\n%s", output)
	}

	// Should show "boom" as top message with count 2
	if !strings.Contains(output, "boom") {
		t.Errorf("expected 'boom' in top results, got:\n%s", output)
	}
}

func TestAnalyzeGroupByLevel(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"second"}`,
		`{"timestamp":"2025-01-26T10:00:03Z","level":"error","message":"crash"}`,
		`{"timestamp":"2025-01-26T10:00:04Z","level":"warn","message":"warning"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("group-by", "level"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "Grouped by: level") {
		t.Errorf("expected 'Grouped by: level', got:\n%s", output)
	}

	// Should show INFO and ERROR with 2 entries each
	if !strings.Contains(output, "INFO") {
		t.Errorf("expected 'INFO' in results, got:\n%s", output)
	}

	if !strings.Contains(output, "ERROR") {
		t.Errorf("expected 'ERROR' in results, got:\n%s", output)
	}
}

func TestAnalyzeWithPattern(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"login successful"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"login failed"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"logout"}`,
		`{"timestamp":"2025-01-26T10:00:03Z","level":"error","message":"login timeout"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("pattern", "login"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "Pattern: login") {
		t.Errorf("expected 'Pattern: login', got:\n%s", output)
	}

	// Should have 3 entries matching "login"
	if !strings.Contains(output, "3 entries") {
		t.Errorf("expected '3 entries' (matching pattern), got:\n%s", output)
	}

	// Should not contain "logout"
	if strings.Contains(output, "logout") {
		t.Errorf("should not contain 'logout' (doesn't match pattern), got:\n%s", output)
	}
}

func TestAnalyzeWithWindow(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"msg1"}`,
		`{"timestamp":"2025-01-26T10:00:30Z","level":"info","message":"msg2"}`,
		`{"timestamp":"2025-01-26T10:01:00Z","level":"error","message":"error1"}`,
		`{"timestamp":"2025-01-26T10:01:30Z","level":"error","message":"error2"}`,
		`{"timestamp":"2025-01-26T10:02:00Z","level":"info","message":"msg3"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("window", "1m"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := cmd.Flags().Set("group-by", "level"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	output := out.String()

	// Should have trend analysis section
	if !strings.Contains(output, "Trend Analysis:") {
		t.Errorf("expected 'Trend Analysis:' section, got:\n%s", output)
	}

	// Should show time windows
	if !strings.Contains(output, "10:00:00") {
		t.Errorf("expected first window at 10:00:00, got:\n%s", output)
	}
}

func TestAnalyzeJSON(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"second"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("group-by", "level"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	var result analyzer.AnalysisResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\noutput: %s", err, out.String())
	}

	if result.TotalLines != 3 {
		t.Errorf("expected TotalLines=3, got %d", result.TotalLines)
	}

	if result.GroupBy != "level" {
		t.Errorf("expected GroupBy='level', got %s", result.GroupBy)
	}

	// Should have 2 groups: INFO and ERROR
	if len(result.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(result.Groups))
	}
}

func TestAnalyzeTable(t *testing.T) {
	viper.Reset()
	viper.Set("format", "table")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("group-by", "level"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "RANK") {
		t.Errorf("expected 'RANK' in table header, got:\n%s", output)
	}

	if !strings.Contains(output, "COUNT") {
		t.Errorf("expected 'COUNT' in table header, got:\n%s", output)
	}

	if !strings.Contains(output, "PERCENT") {
		t.Errorf("expected 'PERCENT' in table header, got:\n%s", output)
	}

	if !strings.Contains(output, "Total entries: 2") {
		t.Errorf("expected 'Total entries: 2', got:\n%s", output)
	}
}

func TestAnalyzeEmptyFile(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "empty.log", []string{})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "No matching entries found") {
		t.Errorf("expected 'No matching entries found' for empty file, got:\n%s", output)
	}
}

func TestAnalyzeInvalidGroupBy(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("group-by", "invalid"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err := runAnalyze(cmd, []string{file})
	if err == nil {
		t.Fatal("expected error for invalid group-by, got nil")
	}

	if !strings.Contains(err.Error(), "invalid --group-by value") {
		t.Errorf("expected error about invalid group-by, got: %v", err)
	}
}

func TestAnalyzeInvalidPattern(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("pattern", "[invalid("); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err := runAnalyze(cmd, []string{file})
	if err == nil {
		t.Fatal("expected error for invalid pattern, got nil")
	}

	if !strings.Contains(err.Error(), "invalid pattern") {
		t.Errorf("expected error about invalid pattern, got: %v", err)
	}
}

func TestAnalyzeInvalidWindow(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("window", "invalid"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err := runAnalyze(cmd, []string{file})
	if err == nil {
		t.Fatal("expected error for invalid window, got nil")
	}

	if !strings.Contains(err.Error(), "invalid --window value") {
		t.Errorf("expected error about invalid window, got: %v", err)
	}
}

func TestAnalyzeTopN(t *testing.T) {
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
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("top", "2"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	var result analyzer.AnalysisResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(result.Groups) != 2 {
		t.Errorf("expected 2 groups (top flag), got %d", len(result.Groups))
	}

	if result.Groups[0].Key != "common" || result.Groups[0].Count != 2 {
		t.Errorf("expected first group to be 'common' with count 2, got %s/%d",
			result.Groups[0].Key, result.Groups[0].Count)
	}
}

func TestAnalyzeMultiFile(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	fileA := writeTempFile(t, dir, "a.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"error","message":"error1"}`,
	})
	fileB := writeTempFile(t, dir, "b.log", []string{
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"error2"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("group-by", "level"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{fileA, fileB}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	output := out.String()

	// Should indicate analysis of 2 files
	if !strings.Contains(output, "2 files") {
		t.Errorf("expected '2 files' in output, got:\n%s", output)
	}

	// Should have 2 ERROR entries total
	if !strings.Contains(output, "ERROR") {
		t.Errorf("expected 'ERROR' in results, got:\n%s", output)
	}
}

func TestAnalyzeGroupBySource(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"msg1","source":"service-a"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"msg2","source":"service-b"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"msg3","source":"service-a"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("group-by", "source"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	var result analyzer.AnalysisResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if result.GroupBy != "source" {
		t.Errorf("expected GroupBy='source', got %s", result.GroupBy)
	}

	// Should have 2 groups: service-a and service-b
	if len(result.Groups) != 2 {
		t.Errorf("expected 2 groups (service-a, service-b), got %d", len(result.Groups))
	}

	// service-a should have 2 entries
	foundServiceA := false
	for _, g := range result.Groups {
		if g.Key == "service-a" && g.Count == 2 {
			foundServiceA = true
			break
		}
	}
	if !foundServiceA {
		t.Errorf("expected service-a with count 2, got: %v", result.Groups)
	}
}

func TestAnalyzeWindowWithErrors(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"msg1"}`,
		`{"timestamp":"2025-01-26T10:00:30Z","level":"error","message":"error1"}`,
		`{"timestamp":"2025-01-26T10:01:00Z","level":"error","message":"error2"}`,
		`{"timestamp":"2025-01-26T10:01:30Z","level":"fatal","message":"fatal1"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("window", "1m"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	var result analyzer.AnalysisResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(result.TimeWindows) == 0 {
		t.Fatalf("expected time windows, got none")
	}

	// Check that error counts are tracked
	hasErrors := false
	for _, w := range result.TimeWindows {
		if w.ErrorCount > 0 {
			hasErrors = true
			break
		}
	}

	if !hasErrors {
		t.Errorf("expected some time windows to have errors, got: %v", result.TimeWindows)
	}
}

func TestAnalyzeNoTimestampWindow(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	// Logs without timestamps
	file := writeTempFile(t, dir, "app.log", []string{
		`{"level":"info","message":"msg1"}`,
		`{"level":"error","message":"msg2"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("window", "1m"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	output := out.String()

	// Should indicate no timestamp information
	if !strings.Contains(output, "No timestamp information available") {
		t.Errorf("expected message about missing timestamps, got:\n%s", output)
	}
}

func TestAnalyzePatternWithWindow(t *testing.T) {
	viper.Reset()
	viper.Set("format", "json")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"login success"}`,
		`{"timestamp":"2025-01-26T10:00:30Z","level":"error","message":"login failed"}`,
		`{"timestamp":"2025-01-26T10:01:00Z","level":"info","message":"logout"}`,
		`{"timestamp":"2025-01-26T10:01:30Z","level":"error","message":"login failed"}`,
	})

	var out bytes.Buffer
	cmd := newAnalyzeTestCmd(&out)
	if err := cmd.Flags().Set("pattern", "login"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := cmd.Flags().Set("window", "1m"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runAnalyze(cmd, []string{file}); err != nil {
		t.Fatalf("runAnalyze() error = %v", err)
	}

	var result analyzer.AnalysisResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Should have pattern in result
	if result.Pattern != "login" {
		t.Errorf("expected Pattern='login', got %s", result.Pattern)
	}

	// Should have 3 total entries matching the pattern
	if result.TotalLines != 3 {
		t.Errorf("expected TotalLines=3 (matching pattern), got %d", result.TotalLines)
	}

	// Should have time windows
	if len(result.TimeWindows) == 0 {
		t.Errorf("expected time windows when using --window, got none")
	}
}
