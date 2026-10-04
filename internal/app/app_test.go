package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/media"
	"video-archiver/internal/ui/jobs"
)

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func ctrlC() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlC}
}

func TestQuitGuardWhileJobsRunning(t *testing.T) {
	dir := t.TempDir()
	jobsModel := jobs.New(func(tea.Msg) {}, media.Tools{}, dir,
		[]media.Job{{Output: filepath.Join(dir, "a.mp4"), Start: 0, End: 1}})
	if !jobsModel.Busy() {
		t.Fatal("setup: expected jobs model to be running")
	}
	m := Model{screen: screenJobs, hasJobs: true, jobsModel: jobsModel}

	for _, msg := range []tea.KeyMsg{keyMsg("q"), ctrlC()} {
		if _, cmd := m.Update(msg); cmd != nil {
			t.Errorf("%q should not quit while jobs are running", msg.String())
		}
	}
}

func TestQuitWhenJobsIdle(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "a.mp4")
	if err := os.WriteFile(out, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	jobsModel := jobs.New(func(tea.Msg) {}, media.Tools{}, dir,
		[]media.Job{{Output: out, Start: 0, End: 1}})
	if jobsModel.Busy() {
		t.Fatal("setup: expected jobs model to be idle (confirm state)")
	}
	m := Model{screen: screenJobs, hasJobs: true, jobsModel: jobsModel}

	for _, msg := range []tea.KeyMsg{keyMsg("q"), ctrlC()} {
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Errorf("%q should quit when jobs are idle", msg.String())
			continue
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%q produced %T, want tea.QuitMsg", msg.String(), cmd())
		}
	}
}
