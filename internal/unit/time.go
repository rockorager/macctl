package unit

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func parseSeconds(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty time span")
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return seconds, nil
	}
	total := 0.0
	for _, field := range strings.Fields(value) {
		seconds, err := parseTimeField(field)
		if err != nil {
			return 0, err
		}
		total += seconds
	}
	return int(total), nil
}

func parseTimeField(field string) (float64, error) {
	idx := 0
	for idx < len(field) && (unicode.IsDigit(rune(field[idx])) || field[idx] == '.') {
		idx++
	}
	if idx == 0 {
		return 0, fmt.Errorf("invalid time span %q", field)
	}
	n, err := strconv.ParseFloat(field[:idx], 64)
	if err != nil {
		return 0, err
	}
	unit := field[idx:]
	if unit == "" {
		unit = "s"
	}
	switch unit {
	case "us", "µs":
		return n / 1_000_000, nil
	case "ms":
		return n / 1_000, nil
	case "s", "sec", "second", "seconds":
		return n, nil
	case "m", "min", "minute", "minutes":
		return n * 60, nil
	case "h", "hr", "hour", "hours":
		return n * 60 * 60, nil
	case "d", "day", "days":
		return n * 24 * 60 * 60, nil
	case "w", "week", "weeks":
		return n * 7 * 24 * 60 * 60, nil
	default:
		return 0, fmt.Errorf("unsupported time unit %q", unit)
	}
}
