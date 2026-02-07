package config

import (
	"testing"
	"time"
)

func TestParseTimeRefAbsolute(t *testing.T) {
	got, err := ParseTimeRef("2025-01-26T10:00:01Z")
	if err != nil {
		t.Fatalf("ParseTimeRef() error = %v", err)
	}
	if !got.Equal(time.Date(2025, 1, 26, 10, 0, 1, 0, time.UTC)) {
		t.Fatalf("unexpected time: %v", got)
	}

	got, err = ParseTimeRef("2025-01-26 10:00:01")
	if err != nil {
		t.Fatalf("ParseTimeRef() error = %v", err)
	}
	if !got.Equal(time.Date(2025, 1, 26, 10, 0, 1, 0, time.UTC)) {
		t.Fatalf("unexpected time: %v", got)
	}

	got, err = ParseTimeRef("2025-01-26")
	if err != nil {
		t.Fatalf("ParseTimeRef() error = %v", err)
	}
	if !got.Equal(time.Date(2025, 1, 26, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected time: %v", got)
	}
}

func TestParseTimeRefRelative(t *testing.T) {
	start := time.Now()
	got, err := ParseTimeRef("1h30m")
	if err != nil {
		t.Fatalf("ParseTimeRef() error = %v", err)
	}
	end := time.Now()

	duration := 90 * time.Minute
	maxSkew := 2 * time.Second
	if end.Sub(got) < duration-maxSkew {
		t.Fatalf("expected duration >= %v, got %v", duration-maxSkew, end.Sub(got))
	}
	if start.Sub(got) > duration+maxSkew {
		t.Fatalf("expected duration <= %v, got %v", duration+maxSkew, start.Sub(got))
	}

	got, err = ParseTimeRef("1d2h")
	if err != nil {
		t.Fatalf("ParseTimeRef() error = %v", err)
	}
	if end.Sub(got) < 26*time.Hour-maxSkew {
		t.Fatalf("expected duration >= %v, got %v", 26*time.Hour-maxSkew, end.Sub(got))
	}
}

func TestParseTimeRefInvalid(t *testing.T) {
	_, err := ParseTimeRef("banana")
	if err == nil {
		t.Fatal("expected error for invalid time reference")
	}
}
