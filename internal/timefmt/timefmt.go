package timefmt

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func Format(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	ms := int64(seconds*1000 + 0.5)
	h := ms / 3_600_000
	ms -= h * 3_600_000
	m := ms / 60_000
	ms -= m * 60_000
	s := ms / 1000
	ms -= s * 1000
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, ms)
	}
	return fmt.Sprintf("%02d:%02d.%03d", m, s, ms)
}

// FormatShort renders with tenth-of-second precision: "MM:SS.d" or
// "H:MM:SS" for durations of an hour or more.
func FormatShort(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	ds := int64(seconds*10 + 0.5)
	h := ds / 36_000
	ds -= h * 36_000
	m := ds / 600
	ds -= m * 600
	s := ds / 10
	ds -= s * 10
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d.%d", m, s, ds)
}

func Parse(text string) (float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("empty timestamp")
	}
	parts := strings.Split(text, ":")
	if len(parts) > 3 {
		return 0, fmt.Errorf("invalid timestamp %q", text)
	}
	var hours, mins, secs float64
	switch len(parts) {
	case 3:
		h, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp %q", text)
		}
		hours = h
		fallthrough
	case 2:
		m, err := strconv.ParseFloat(parts[len(parts)-2], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp %q", text)
		}
		mins = m
	}
	s, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid timestamp %q", text)
	}
	secs = s
	if hours < 0 || mins < 0 || secs < 0 {
		return 0, fmt.Errorf("negative timestamp %q", text)
	}
	// A bare number is total seconds and may exceed 60. Otherwise only the
	// leading component may exceed 60 (e.g. 90:30 or 2:00:00 style inputs
	// with large leading units).
	if len(parts) >= 2 && secs >= 60 {
		return 0, fmt.Errorf("seconds must be below 60 in %q", text)
	}
	if len(parts) == 3 && mins >= 60 {
		return 0, fmt.Errorf("minutes must be below 60 in %q", text)
	}
	total := hours*3600 + mins*60 + secs
	if math.IsNaN(total) || math.IsInf(total, 0) {
		return 0, fmt.Errorf("timestamp must be finite: %q", text)
	}
	return total, nil
}
