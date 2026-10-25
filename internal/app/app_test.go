package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"video-archiver/internal/media"
	"video-archiver/internal/render"
	"video-archiver/internal/ui/editor"
	"video-archiver/internal/ui/jobs"
)

func TestJobsClearPreviewThroughCompletion(t *testing.T) {
	m := New(Config{Dir: t.TempDir(), Preview: "kitty", Runtime: &Runtime{}})
	m.width, m.height = 80, 24
	m.screen, m.hasEditor = screenEditor, true
	updated, cmd := m.Update(editor.RunJobsMsg{OutDir: t.TempDir()})
	m = updated.(Model)
	if m.screen != screenJobs || cmd == nil {
		t.Fatal("expected transition to jobs")
	}
	if !strings.HasPrefix(m.View(), m.renderer.Clear()) {
		t.Fatal("jobs screen must clear the editor preview")
	}
	// An empty run completes immediately, without launching ffmpeg.
	updated, _ = m.Update(cmd())
	m = updated.(Model)
	v := m.View()
	if !strings.HasPrefix(v, m.renderer.Clear()) || !strings.Contains(v, "finished: 0 done") {
		t.Fatal("finished message must be drawn after clearing the preview")
	}
	updated, _ = m.Update(jobs.BackMsg{})
	m = updated.(Model)
	if strings.HasPrefix(m.View(), m.renderer.Clear()) {
		t.Fatal("returning to editor must allow the preview to be displayed")
	}
}

func TestHelpClearsPreview(t *testing.T) {
	m := New(Config{Dir: t.TempDir(), Preview: "kitty"})
	m.width, m.height = 80, 24
	m.screen, m.hasEditor = screenEditor, true
	updated, _ := m.Update(keyMsg("h"))
	m = updated.(Model)
	if !strings.HasPrefix(m.View(), m.renderer.Clear()) || !strings.Contains(m.View(), "Help") {
		t.Fatal("help must clear the preview before drawing")
	}
}

func TestHelpAvailableOnEveryScreen(t *testing.T) {
	for _, screen := range []screen{screenBrowser, screenEditor, screenJobs} {
		m := Model{screen: screen, width: 80, height: 24}
		opened, cmd := m.Update(keyMsg("h"))
		m = opened.(Model)
		if cmd != nil || !m.showHelp || !strings.Contains(m.View(), "Help") {
			t.Fatalf("screen %d: h did not open help", screen)
		}
		closed, cmd := m.Update(keyMsg("q"))
		if cmd != nil || closed.(Model).showHelp || closed.(Model).confirmQuit {
			t.Fatalf("screen %d: closing help activated an underlying action", screen)
		}
	}
}

func TestHelpFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {40, 12}, {20, 8}, {4, 5}} {
		m := Model{screen: screenEditor, showHelp: true, width: size[0], height: size[1]}
		view := m.View()
		if lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
			t.Errorf("help exceeds %dx%d: %dx%d", m.width, m.height, lipgloss.Width(view), lipgloss.Height(view))
		}
	}
}

func TestHelpPreservesPromptTyping(t *testing.T) {
	e := editor.New(media.Tools{}, render.New(render.KindNone), media.Info{Path: filepath.Join(t.TempDir(), "movie.mp4")})
	e, _ = e.Update(keyMsg("O"))
	m := Model{screen: screenEditor, hasEditor: true, editor: e}
	updated, _ := m.Update(keyMsg("h"))
	m = updated.(Model)
	if m.showHelp || !m.editor.EditingText() {
		t.Fatal("h should remain a text input key in prompts")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyF1})
	if !updated.(Model).showHelp {
		t.Fatal("F1 should open help from a prompt")
	}
}

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
