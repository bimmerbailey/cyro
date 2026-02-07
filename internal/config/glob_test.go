package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandGlobs(t *testing.T) {
	dir := t.TempDir()

	fileA := filepath.Join(dir, "a.log")
	fileB := filepath.Join(dir, "b.log")
	fileC := filepath.Join(dir, "c.txt")

	for _, path := range []string{fileA, fileB, fileC} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	files, err := ExpandGlobs([]string{filepath.Join(dir, "*.log")})
	if err != nil {
		t.Fatalf("ExpandGlobs() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	files, err = ExpandGlobs([]string{fileA, filepath.Join(dir, "*.log")})
	if err != nil {
		t.Fatalf("ExpandGlobs() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestExpandGlobsNoMatch(t *testing.T) {
	dir := t.TempDir()

	_, err := ExpandGlobs([]string{filepath.Join(dir, "*.missing")})
	if err == nil {
		t.Fatal("expected error for unmatched glob")
	}
}
