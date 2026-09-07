package jobs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/media"
)

func fakeSend(ch chan tea.Msg) SendFunc {
	return func(msg tea.Msg) { ch <- msg }
}

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func waitMsg(t *testing.T, ch chan tea.Msg) tea.Msg {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for job message")
		return nil
	}
}

func TestRunFailuresFinish(t *testing.T) {
	if _, err := os.Stat("/bin/false"); err != nil {
		t.Skip("/bin/false not available")
	}
	ch := make(chan tea.Msg, 8)
	outDir := filepath.Join(t.TempDir(), "out")
	jobs := []media.Job{
		{Source: "/nonexistent-a", Output: filepath.Join(outDir, "a.mp4"), Start: 0, End: 1},
		{Source: "/nonexistent-b", Output: filepath.Join(outDir, "b.mp4"), Start: 0, End: 1},
	}
	m := New(fakeSend(ch), media.Tools{FFmpeg: "/bin/false", FFprobe: "/bin/false"}, outDir, jobs)
	if m.state != StateRunning {
		t.Fatalf("state = %v, want StateRunning", m.state)
	}

	m, _ = m.Update(beginMsg{})
	for range jobs {
		msg := waitMsg(t, ch).(jobDoneMsg)
		if msg.err == nil {
			t.Fatalf("job %d: expected failure", msg.job)
		}
		m, _ = m.Update(msg)
	}
	if m.state != StateDone {
		t.Errorf("state = %v, want StateDone", m.state)
	}
	done, failed, skipped := m.Counts()
	if done != 0 || failed != 2 || skipped != 0 {
		t.Errorf("counts = %d/%d/%d, want 0/2/0", done, failed, skipped)
	}
}

func TestConfirmSkipExisting(t *testing.T) {
	ch := make(chan tea.Msg, 8)
	outDir := t.TempDir()
	existing := filepath.Join(outDir, "a.mp4")
	if err := os.WriteFile(existing, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	jobs := []media.Job{
		{Source: "/nonexistent", Output: existing, Start: 0, End: 1},
	}
	m := New(fakeSend(ch), media.Tools{FFmpeg: "/bin/false", FFprobe: "/bin/false"}, outDir, jobs)
	if m.state != StateConfirm {
		t.Fatalf("state = %v, want StateConfirm", m.state)
	}
	m, _ = m.Update(keyMsg("s"))
	if m.states[0] != JobSkipped {
		t.Errorf("state of existing job = %v, want JobSkipped", m.states[0])
	}
	if m.state != StateDone {
		t.Errorf("state = %v, want StateDone (nothing left to run)", m.state)
	}
	if _, err := os.Stat(existing); err != nil {
		t.Error("skipped job must not touch existing file")
	}
}

func TestConfirmCancel(t *testing.T) {
	ch := make(chan tea.Msg, 8)
	outDir := t.TempDir()
	existing := filepath.Join(outDir, "a.mp4")
	if err := os.WriteFile(existing, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	jobs := []media.Job{{Source: "/nonexistent", Output: existing, Start: 0, End: 1}}
	m := New(fakeSend(ch), media.Tools{FFmpeg: "/bin/false", FFprobe: "/bin/false"}, outDir, jobs)
	var got tea.Msg
	m, cmd := m.Update(keyMsg("c"))
	if cmd == nil {
		t.Fatal("expected a command")
	}
	got = cmd()
	if _, ok := got.(BackMsg); !ok {
		t.Errorf("command produced %T, want BackMsg", got)
	}
}
