package editor

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/media"
	"video-archiver/internal/render"
	"video-archiver/internal/segments"
)

func TestBuildJobs(t *testing.T) {
	segs := []segments.Segment{
		{Start: 0, End: 10.5, Title: "opening"},
		{Start: 20, End: 30, Title: "bad/name"},
	}
	jobs := BuildJobs("/movies/holiday.avi", "/out", segs)
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs", len(jobs))
	}
	if jobs[0].Output != filepath.Join("/out", "opening.mp4") {
		t.Errorf("job 0 output = %s", jobs[0].Output)
	}
	if jobs[1].Output != filepath.Join("/out", "bad name.mp4") {
		t.Errorf("job 1 output = %s", jobs[1].Output)
	}
	if jobs[0].Source != "/movies/holiday.avi" || jobs[0].Duration() != 10.5 {
		t.Errorf("job 0 = %+v", jobs[0])
	}
}

func TestNewWarnsOnInvalidSegments(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "movie.mp4")
	sidecar := segments.SidecarPath(src)
	f := &segments.File{
		Source:   "movie.mp4",
		Segments: []segments.Segment{{Start: 10, End: 5, Title: "bad"}},
	}
	if err := f.Save(sidecar); err != nil {
		t.Fatal(err)
	}

	m := New(media.Tools{}, render.New(render.KindNone), media.Info{Path: src, Duration: 100})
	if m.warning == "" {
		t.Error("expected a validation warning for invalid segments")
	}

	// Running with invalid segments must be refused.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil {
		t.Error("run should be refused while segments are invalid")
	}
}
