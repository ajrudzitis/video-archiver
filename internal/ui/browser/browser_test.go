package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsVideoFile(t *testing.T) {
	for _, name := range []string{"a.mp4", "b.MKV", "c.m2ts", "movie.DV"} {
		if !IsVideoFile(name) {
			t.Errorf("%s should be a video file", name)
		}
	}
	for _, name := range []string{"a.txt", "b.mp4.json", "c", "d.srt"} {
		if IsVideoFile(name) {
			t.Errorf("%s should not be a video file", name)
		}
	}
}

func TestLoadOrderingAndSidecar(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.mp4", "A.avi", "notes.txt", ".hidden.mp4"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.mp4.segments.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New(dir)
	if m.err != nil {
		t.Fatal(m.err)
	}
	var names []string
	for _, e := range m.entries {
		names = append(names, e.name)
	}
	want := []string{"..", "subdir", "A.avi", "b.mp4"}
	if len(names) != len(want) {
		t.Fatalf("entries = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("entries = %v, want %v", names, want)
		}
	}
	last := m.entries[len(m.entries)-1]
	if !last.hasSidecar {
		t.Error("b.mp4 should be marked with sidecar")
	}
}

func TestHumanSize(t *testing.T) {
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{5 * 1024 * 1024 * 1024, "5.0 GiB"},
	} {
		if got := humanSize(tc.in); got != tc.want {
			t.Errorf("humanSize(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
