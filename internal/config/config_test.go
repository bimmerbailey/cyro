package config

import (
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  LogLevel
	}{
		// Lowercase
		{"debug lowercase", "debug", LevelDebug},
		{"info lowercase", "info", LevelInfo},
		{"warn lowercase", "warn", LevelWarn},
		{"warning lowercase", "warning", LevelWarn},
		{"error lowercase", "error", LevelError},
		{"fatal lowercase", "fatal", LevelFatal},
		{"critical lowercase", "critical", LevelFatal},

		// Uppercase
		{"DEBUG uppercase", "DEBUG", LevelDebug},
		{"INFO uppercase", "INFO", LevelInfo},
		{"WARN uppercase", "WARN", LevelWarn},
		{"WARNING uppercase", "WARNING", LevelWarn},
		{"ERROR uppercase", "ERROR", LevelError},
		{"FATAL uppercase", "FATAL", LevelFatal},
		{"CRITICAL uppercase", "CRITICAL", LevelFatal},

		// Mixed case
		{"Debug mixed", "Debug", LevelDebug},
		{"Info mixed", "Info", LevelInfo},
		{"Warning mixed", "Warning", LevelWarn},
		{"Error mixed", "Error", LevelError},
		{"Fatal mixed", "Fatal", LevelFatal},

		// Abbreviations
		{"dbg abbrev", "dbg", LevelDebug},
		{"inf abbrev", "inf", LevelInfo},
		{"err abbrev", "err", LevelError},
		{"crit abbrev", "crit", LevelFatal},

		// Unknown
		{"empty string", "", LevelUnknown},
		{"invalid", "invalid", LevelUnknown},
		{"random", "random", LevelUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  string
	}{
		{"debug", LevelDebug, "DEBUG"},
		{"info", LevelInfo, "INFO"},
		{"warn", LevelWarn, "WARN"},
		{"error", LevelError, "ERROR"},
		{"fatal", LevelFatal, "FATAL"},
		{"unknown", LevelUnknown, "UNKNOWN"},
		{"invalid value", LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("LogLevel.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLogLevel_MarshalJSON(t *testing.T) {
	level := LevelError
	got, err := level.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	want := `"ERROR"`
	if string(got) != want {
		t.Errorf("MarshalJSON() = %s, want %s", got, want)
	}

	// Verify it's NOT an integer
	if string(got) == "3" {
		t.Error("MarshalJSON() should produce string \"ERROR\", not integer 3")
	}
}

func TestLogLevel_UnmarshalJSON(t *testing.T) {
	var level LogLevel
	err := level.UnmarshalJSON([]byte(`"ERROR"`))
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if level != LevelError {
		t.Errorf("UnmarshalJSON() got %v, want %v", level, LevelError)
	}

	// Test case-insensitivity
	var level2 LogLevel
	err = level2.UnmarshalJSON([]byte(`"info"`))
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if level2 != LevelInfo {
		t.Errorf("UnmarshalJSON() got %v, want %v", level2, LevelInfo)
	}
}
