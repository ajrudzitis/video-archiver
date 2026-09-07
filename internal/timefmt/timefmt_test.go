package timefmt

import (
	"math"
	"testing"
)

func TestFormat(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{0, "00:00.000"},
		{0.0004, "00:00.000"},
		{0.0006, "00:00.001"},
		{61.5, "01:01.500"},
		{3599.999, "59:59.999"},
		{3600, "1:00:00.000"},
		{3661.25, "1:01:01.250"},
		{-5, "00:00.000"},
	} {
		if got := Format(tc.in); got != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatShort(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{0, "00:00.0"},
		{61.54, "01:01.5"},
		{3599.94, "59:59.9"},
		{3599.96, "1:00:00"},
		{3661.25, "1:01:01"},
	} {
		if got := FormatShort(tc.in); got != tc.want {
			t.Errorf("FormatShort(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
	}{
		{"0", 0},
		{"12.5", 12.5},
		{"89.9", 89.9},
		{"3661", 3661},
		{"01:02.5", 62.5},
		{"90:30", 5430},
		{"1:01:01.250", 3661.25},
		{" 02:03 ", 123},
	} {
		got, err := Parse(tc.in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tc.in, err)
			continue
		}
		if math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("Parse(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	for _, in := range []string{"", "abc", "1:2:3:4", "-1", "1:60", "1:00:60", "1:60:00", "12:"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) expected error", in)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	for _, v := range []float64{0, 0.001, 12.345, 62.5, 3661.25, 35999.999} {
		got, err := Parse(Format(v))
		if err != nil {
			t.Fatalf("Parse(Format(%v)) error: %v", v, err)
		}
		if math.Abs(got-v) > 0.0005 {
			t.Errorf("round trip %v -> %q -> %v", v, Format(v), got)
		}
	}
}
