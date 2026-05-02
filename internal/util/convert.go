// Package util provides shared conversion helpers used across cmd files.
package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseSize parses a human-readable size string (e.g., "10MB", "1.5GB", "500KB")
// into bytes. Supported suffixes: B, KB/K, MB/M, GB/G, TB/T (case-insensitive).
// Returns an error for malformed input.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))

	// Extract number and unit.
	var numStr string
	var unit string
	for i, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			numStr += string(r)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no number found in size string")
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}

	multiplier := int64(1)
	switch unit {
	case "B", "":
		multiplier = 1
	case "KB", "K":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}

	return int64(num * float64(multiplier)), nil
}

// FormatDuration formats a time.Duration as a human-readable relative age
// (e.g., "3 days", "2 months", "1 year").
func FormatDuration(d time.Duration) string {
	if d < time.Hour {
		return "less than 1 hour"
	}

	hours := int(d.Hours())
	if hours == 1 {
		return "1 hour"
	}
	if hours < 24 {
		return fmt.Sprintf("%d hours", hours)
	}

	days := hours / 24
	if days == 1 {
		return "1 day"
	}
	if days < 30 {
		return fmt.Sprintf("%d days", days)
	}

	months := days / 30
	if months == 1 {
		return "1 month"
	}
	if months < 12 {
		return fmt.Sprintf("%d months", months)
	}

	years := months / 12
	if years == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", years)
}
