package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newSearchTestCmd(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{Use: "search"}
	cmd.SetOut(out)
	cmd.Flags().StringP("pattern", "p", "", "regex pattern to search for")
	cmd.Flags().StringP("level", "l", "", "filter by log level (debug, info, warn, error, fatal)")
	cmd.Flags().String("since", "", "show logs since timestamp (RFC3339 or relative like '1h')")
	cmd.Flags().String("until", "", "show logs until timestamp (RFC3339 or relative like '1h')")
	cmd.Flags().IntP("context", "C", 0, "number of context lines around matches")
	cmd.Flags().BoolP("count", "c", false, "only print count of matching lines")
	cmd.Flags().BoolP("invert", "V", false, "invert match (show non-matching lines)")
	return cmd
}

func writeTempFile(t *testing.T, dir string, name string, lines []string) string {
	path := filepath.Join(dir, name)
	content := []byte("" + joinLines(lines))
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func joinLines(lines []string) string {
	buf := bytes.Buffer{}
	for i, line := range lines {
		buf.WriteString(line)
		if i < len(lines)-1 {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func TestSearchContext(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"after1"}`,
		`{"timestamp":"2025-01-26T10:00:03Z","level":"info","message":"between"}`,
		`{"timestamp":"2025-01-26T10:00:04Z","level":"error","message":"boom2"}`,
		`{"timestamp":"2025-01-26T10:00:05Z","level":"info","message":"after2"}`,
	})

	var out bytes.Buffer
	cmd := newSearchTestCmd(&out)
	if err := cmd.Flags().Set("pattern", "boom"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := cmd.Flags().Set("context", "1"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runSearch(cmd, []string{file}); err != nil {
		t.Fatalf("runSearch() error = %v", err)
	}

	expected := joinLines([]string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"after1"}`,
		"--",
		`{"timestamp":"2025-01-26T10:00:03Z","level":"info","message":"between"}`,
		`{"timestamp":"2025-01-26T10:00:04Z","level":"error","message":"boom2"}`,
		`{"timestamp":"2025-01-26T10:00:05Z","level":"info","message":"after2"}`,
	}) + "\n"

	if out.String() != expected {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestSearchCountInvert(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"info","message":"after"}`,
	})

	var out bytes.Buffer
	cmd := newSearchTestCmd(&out)
	if err := cmd.Flags().Set("pattern", "error"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := cmd.Flags().Set("invert", "true"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := cmd.Flags().Set("count", "true"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runSearch(cmd, []string{file}); err != nil {
		t.Fatalf("runSearch() error = %v", err)
	}

	if out.String() != "2\n" {
		t.Fatalf("unexpected count output: %s", out.String())
	}
}

func TestSearchLevelExact(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"warn","message":"warn"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"error"}`,
	})

	var out bytes.Buffer
	cmd := newSearchTestCmd(&out)
	if err := cmd.Flags().Set("level", "warn"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runSearch(cmd, []string{file}); err != nil {
		t.Fatalf("runSearch() error = %v", err)
	}

	expected := `{"timestamp":"2025-01-26T10:00:00Z","level":"warn","message":"warn"}` + "\n"
	if out.String() != expected {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestSearchTimeRange(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	file := writeTempFile(t, dir, "app.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"info","message":"first"}`,
		`{"timestamp":"2025-01-26T10:00:01Z","level":"info","message":"second"}`,
		`{"timestamp":"2025-01-26T10:00:04Z","level":"info","message":"third"}`,
	})

	var out bytes.Buffer
	cmd := newSearchTestCmd(&out)
	if err := cmd.Flags().Set("since", "2025-01-26T10:00:03Z"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := runSearch(cmd, []string{file}); err != nil {
		t.Fatalf("runSearch() error = %v", err)
	}

	expected := `{"timestamp":"2025-01-26T10:00:04Z","level":"info","message":"third"}` + "\n"
	if out.String() != expected {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestSearchMultiFileCount(t *testing.T) {
	viper.Reset()
	viper.Set("format", "text")

	dir := t.TempDir()
	fileA := writeTempFile(t, dir, "a.log", []string{
		`{"timestamp":"2025-01-26T10:00:00Z","level":"error","message":"boom"}`,
	})
	fileB := writeTempFile(t, dir, "b.log", []string{
		`{"timestamp":"2025-01-26T10:00:01Z","level":"error","message":"boom"}`,
		`{"timestamp":"2025-01-26T10:00:02Z","level":"error","message":"boom"}`,
	})

	var out bytes.Buffer
	cmd := newSearchTestCmd(&out)
	if err := cmd.Flags().Set("pattern", "error"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := cmd.Flags().Set("count", "true"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	pattern := filepath.Join(dir, "*.log")
	if err := runSearch(cmd, []string{pattern}); err != nil {
		t.Fatalf("runSearch() error = %v", err)
	}

	expected := fileA + ":1\n" + fileB + ":2\n"
	if out.String() != expected {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
