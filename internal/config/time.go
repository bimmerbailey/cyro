package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseTimeRef parses an absolute timestamp or a relative duration.
// Relative values are subtracted from the current time (e.g. "1h", "30m", "1d2h").
func ParseTimeRef(s string) (time.Time, error) {
	input := strings.TrimSpace(s)
	if input == "" {
		return time.Time{}, fmt.Errorf("time reference is empty")
	}

	if t, err := parseAbsoluteTime(input); err == nil {
		return t, nil
	}

	d, err := parseRelativeDuration(input)
	if err != nil {
		return time.Time{}, err
	}

	return time.Now().Add(-d), nil
}

func parseAbsoluteTime(input string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, input); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid absolute time: %s", input)
}

// ParseDuration parses a duration string supporting standard Go durations and extended units (d for days).
// Examples: "5m", "1h", "1h30m", "2d"
func ParseDuration(s string) (time.Duration, error) {
	return parseRelativeDuration(s)
}

func parseRelativeDuration(input string) (time.Duration, error) {
	if d, err := time.ParseDuration(input); err == nil {
		return d, nil
	}

	re := regexp.MustCompile(`(\d+)([dhms])`)
	matches := re.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid relative duration: %s", input)
	}

	totalLen := 0
	total := time.Duration(0)

	for _, match := range matches {
		totalLen += match[1] - match[0]
		valueStr := input[match[2]:match[3]]
		unit := input[match[4]:match[5]]

		value, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid relative duration: %s", input)
		}

		switch unit {
		case "d":
			total += time.Hour * 24 * time.Duration(value)
		case "h":
			total += time.Hour * time.Duration(value)
		case "m":
			total += time.Minute * time.Duration(value)
		case "s":
			total += time.Second * time.Duration(value)
		default:
			return 0, fmt.Errorf("invalid relative duration: %s", input)
		}
	}

	if totalLen != len(input) {
		return 0, fmt.Errorf("invalid relative duration: %s", input)
	}

	return total, nil
}
