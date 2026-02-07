package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ExpandGlobs expands file paths and glob patterns into a sorted unique list.
func ExpandGlobs(patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no file patterns provided")
	}

	files := make([]string, 0)
	seen := make(map[string]struct{})

	for _, pattern := range patterns {
		if hasGlobMeta(pattern) {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return nil, err
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no matches for pattern %q", pattern)
			}
			for _, match := range matches {
				if _, ok := seen[match]; ok {
					continue
				}
				seen[match] = struct{}{}
				files = append(files, match)
			}
			continue
		}

		if _, err := os.Stat(pattern); err != nil {
			return nil, err
		}
		if _, ok := seen[pattern]; ok {
			continue
		}
		seen[pattern] = struct{}{}
		files = append(files, pattern)
	}

	sort.Strings(files)
	return files, nil
}

func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}
