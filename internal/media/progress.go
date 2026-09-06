package media

import (
	"strconv"
	"strings"
)

func ParseProgressLine(line string) (seconds float64, ok bool) {
	if v, found := strings.CutPrefix(line, "out_time_us="); found {
		if us, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return us / 1_000_000, true
		}
		return 0, false
	}
	if v, found := strings.CutPrefix(line, "out_time_ms="); found {
		if ms, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return ms / 1000, true
		}
	}
	return 0, false
}
