package segments

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSegmentValid(t *testing.T) {
	if err := (Segment{0, 10, "a"}).Valid(); err != nil {
		t.Errorf("valid segment rejected: %v", err)
	}
	for _, s := range []Segment{
		{0, 10, ""},
		{-1, 10, "a"},
		{10, 10, "a"},
		{10, 5, "a"},
	} {
		if err := s.Valid(); err == nil {
			t.Errorf("expected error for %+v", s)
		}
	}
}

func TestSidecarRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "movie.mp4.segments.json")
	in := &File{
		Source:    "movie.mp4",
		Duration:  123.456,
		OutputDir: filepath.Join(dir, "out"),
		Segments: []Segment{
			{0, 10.5, "opening"},
			{20, 30, "the end"},
		},
	}
	if err := in.Save(path); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != SidecarVersion {
		t.Errorf("version = %d, want %d", out.Version, SidecarVersion)
	}
	if !reflect.DeepEqual(in.Segments, out.Segments) {
		t.Errorf("segments mismatch: %+v vs %+v", in.Segments, out.Segments)
	}
	if in.Source != out.Source || in.Duration != out.Duration || in.OutputDir != out.OutputDir {
		t.Errorf("metadata mismatch: %+v vs %+v", in, out)
	}
}

func TestLoadMissing(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Error("expected error for missing sidecar")
	}
}

func TestValidate(t *testing.T) {
	good := &File{Segments: []Segment{{0, 10, "a"}, {20, 30, "b"}}}
	if err := good.Validate(); err != nil {
		t.Errorf("valid segments rejected: %v", err)
	}
	for _, f := range []*File{
		{Segments: []Segment{{10, 5, "a"}}},             // end before start
		{Segments: []Segment{{-1, 5, "a"}}},             // negative start
		{Segments: []Segment{{0, 5, ""}}},               // empty title
		{Segments: []Segment{{0, 5, "..."}}},            // empty after sanitizing
		{Segments: []Segment{{0, 5, "A"}, {6, 9, "a"}}}, // duplicate output name
	} {
		if err := f.Validate(); err == nil {
			t.Errorf("expected error for %+v", f.Segments)
		}
	}
}

func TestSorted(t *testing.T) {
	f := &File{Segments: []Segment{{30, 40, "b"}, {0, 10, "a"}, {15, 20, "c"}}}
	got := f.Sorted()
	for i := 1; i < len(got); i++ {
		if got[i].Start < got[i-1].Start {
			t.Fatalf("not sorted: %+v", got)
		}
	}
	if len(f.Segments) != 3 || f.Segments[0].Title != "b" {
		t.Error("Sorted must not mutate receiver")
	}
}

func TestHasDuplicateTitle(t *testing.T) {
	f := &File{Segments: []Segment{{0, 1, "Opening"}, {2, 3, "middle"}}}
	if !f.HasDuplicateTitle("opening", -1) {
		t.Error("case-insensitive duplicate not detected")
	}
	if f.HasDuplicateTitle("opening", 0) {
		t.Error("skip index not honored")
	}
	if f.HasDuplicateTitle("fresh", -1) {
		t.Error("false positive")
	}
}

func TestSanitizeTitle(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"opening credits", "opening credits"},
		{"a/b\\c:d*e?f\"g<h>i|j", "a b c d e f g h i j"},
		{"  spaced  ", "spaced"},
		{".hidden.", "hidden"},
		{"bad\x01ctrl", "bad ctrl"},
		{"...", ""},
	} {
		if got := SanitizeTitle(tc.in); got != tc.want {
			t.Errorf("SanitizeTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestOutputName(t *testing.T) {
	if got := OutputName("my title"); got != "my title.mp4" {
		t.Errorf("OutputName = %q", got)
	}
}

func TestRejectNonfiniteBoundaries(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, segment := range []Segment{{Start: value, End: 10, Title: "test"}, {Start: 0, End: value, Title: "test"}} {
			if err := segment.Valid(); err == nil {
				t.Errorf("accepted %+v", segment)
			}
			f := File{Segments: []Segment{segment}}
			if err := f.Validate(); err == nil {
				t.Errorf("file accepted %+v", segment)
			}
		}
	}
}
