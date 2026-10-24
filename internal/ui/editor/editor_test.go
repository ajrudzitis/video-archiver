package editor

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestNewBacksUpMalformedSidecar(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "movie.mp4")
	sidecar := segments.SidecarPath(src)
	if err := os.WriteFile(sidecar, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New(media.Tools{}, render.New(render.KindNone), media.Info{Path: src, Duration: 100})

	if m.warning == "" {
		t.Error("expected a warning for a malformed sidecar")
	}
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Error("malformed sidecar should have been moved aside")
	}
	if _, err := os.Stat(sidecar + ".corrupt"); err != nil {
		t.Errorf("expected backup at %s.corrupt: %v", sidecar, err)
	}
	if len(m.file.Segments) != 0 {
		t.Error("expected no segments after backing up a malformed sidecar")
	}
}

func TestBackupSidecarUnique(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "movie.mp4.segments.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".corrupt", []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	backup, err := backupSidecar(path)
	if err != nil {
		t.Fatal(err)
	}
	if backup == path+".corrupt" {
		t.Error("should not overwrite an existing backup")
	}
	if _, err := os.Stat(backup); err != nil {
		t.Errorf("backup should exist: %v", err)
	}
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestQuitRequiresConfirmation(t *testing.T) {
	m := New(media.Tools{}, render.New(render.KindNone), media.Info{Path: "/movie.mp4", Duration: 100})

	// First q only arms the confirmation; it must not quit.
	armed, cmd := m.Update(key("q"))
	if cmd != nil {
		t.Fatal("first q should prompt, not quit")
	}
	if !armed.confirmQuit {
		t.Fatal("expected confirmation flag after q")
	}

	// y confirms and quits.
	_, cmd = armed.Update(key("y"))
	if cmd == nil {
		t.Fatal("y should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("y produced %T, want tea.QuitMsg", cmd())
	}

	// Any other key cancels without quitting.
	armed, _ = m.Update(key("q"))
	after, cmd := armed.Update(key("n"))
	if cmd != nil {
		t.Error("n should cancel, not quit")
	}
	if after.confirmQuit {
		t.Error("confirmation flag should be cleared after cancel")
	}
}

func TestSaveFailureStaysVisibleAndBlocksLeaving(t *testing.T) {
	m := New(media.Tools{}, render.New(render.KindNone), media.Info{Path: filepath.Join(t.TempDir(), "missing", "movie.mp4"), Duration: 10})
	m, _ = m.Update(key("s"))
	if !m.dirty || !strings.HasPrefix(m.status, "save error:") {
		t.Fatalf("hidden failure: %s", m.status)
	}
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatal("left editor with unsaved changes")
	}
	if err := os.MkdirAll(filepath.Dir(m.sidecar), 0755); err != nil {
		t.Fatal(err)
	}
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil || m.dirty {
		t.Fatal("expected successful retry and exit")
	}
}

func TestOutputDirectoryPromptPreservesDefault(t *testing.T) {
	m := New(media.Tools{}, render.New(render.KindNone), media.Info{Path: "videos/movie.mp4", Duration: 10})
	original := m.file.OutputDir
	if !filepath.IsAbs(original) {
		t.Fatal("output directory must be absolute")
	}
	m.sidecar = filepath.Join(t.TempDir(), "sidecar.json")
	m, _ = m.confirmPrompt(promptOutputDir, original)
	if m.file.OutputDir != original {
		t.Fatalf("changed %s to %s", original, m.file.OutputDir)
	}
	m, _ = m.confirmPrompt(promptOutputDir, "custom")
	want, _ := filepath.Abs("videos/custom")
	if m.file.OutputDir != want {
		t.Fatalf("got %s, want %s", m.file.OutputDir, want)
	}
}

func TestEndPreviewsLastFrameAndPreservesEndpoint(t *testing.T) {
	tools, err := media.FindTools()
	if err != nil {
		t.Skip(err)
	}
	src := filepath.Join(t.TempDir(), "source.mp4")
	if out, err := exec.Command(tools.FFmpeg, "-v", "error", "-f", "lavfi", "-i", "testsrc2=duration=1:size=160x120:rate=25", "-c:v", "libx264", "-y", src).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	m := New(tools, render.New(render.KindHalfBlock), media.Info{Path: src, Duration: 1, FPS: 25})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	frame := m.frameCmd(m.tick)().(frameMsg)
	if frame.err != nil || !bytes.HasPrefix(frame.data, []byte("\x89PNG")) {
		t.Fatalf("end preview: %v", frame.err)
	}
	if m.pos != 1 || frame.at != 1 {
		t.Fatal("endpoint changed")
	}
}
