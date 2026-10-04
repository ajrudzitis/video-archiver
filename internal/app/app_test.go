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

func TestCtrlCQuitsWhenJobsIdle(t *testing.T) {
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

	_, cmd := m.Update(ctrlC())
	if cmd == nil {
		t.Fatal("ctrl+c should quit when jobs are idle")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c produced %T, want tea.QuitMsg", cmd())
	}
}

func TestQuitConfirmationBrowser(t *testing.T) {
	m := Model{screen: screenBrowser}

	armedModel, cmd := m.Update(keyMsg("q"))
	if cmd != nil {
		t.Fatal("q should arm confirmation, not quit")
	}
	armed := armedModel.(Model)
	if !armed.confirmQuit {
		t.Fatal("expected confirmation flag after q")
	}

	_, cmd = armed.Update(keyMsg("y"))
	if cmd == nil {
		t.Fatal("y should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("y produced %T, want tea.QuitMsg", cmd())
	}

	// A non-y key cancels without quitting.
	armedModel, _ = m.Update(keyMsg("q"))
	armed = armedModel.(Model)
	afterModel, cmd := armed.Update(keyMsg("n"))
	if cmd != nil {
		t.Error("n should cancel, not quit")
	}
	if after := afterModel.(Model); after.confirmQuit {
		t.Error("confirmation flag should be cleared after cancel")
	}
}

func TestQuitConfirmationJobs(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "a.mp4")
	if err := os.WriteFile(out, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	jobsModel := jobs.New(func(tea.Msg) {}, media.Tools{}, dir,
		[]media.Job{{Output: out, Start: 0, End: 1}})
	m := Model{screen: screenJobs, hasJobs: true, jobsModel: jobsModel}

	armedModel, cmd := m.Update(keyMsg("q"))
	if cmd != nil {
		t.Fatal("q should arm confirmation when idle, not quit")
	}
	armed := armedModel.(Model)
	if !armed.confirmQuit {
		t.Fatal("expected confirmation flag after q")
	}

	_, cmd = armed.Update(keyMsg("y"))
	if cmd == nil {
		t.Fatal("y should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("y produced %T, want tea.QuitMsg", cmd())
	}
}
