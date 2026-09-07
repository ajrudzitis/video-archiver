package jobs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/media"
)

type State int

const (
	StateConfirm State = iota
	StateRunning
	StateDone
)

type JobState int

const (
	JobPending JobState = iota
	JobRunning
	JobDone
	JobFailed
	JobSkipped
)

type BackMsg struct{}

type beginMsg struct{}

type progressMsg struct {
	job  int
	frac float64
}

type jobDoneMsg struct {
	job int
	err error
}

type SendFunc func(tea.Msg)

type Model struct {
	send   SendFunc
	tools  media.Tools
	outDir string
	jobs   []media.Job

	states   []JobState
	errs     []string
	progress []float64

	state    State
	existing []int
	aborted  bool
	cancel   context.CancelFunc

	width  int
	height int
}

func New(send SendFunc, tools media.Tools, outDir string, jobs []media.Job) Model {
	m := Model{
		send:     send,
		tools:    tools,
		outDir:   outDir,
		jobs:     jobs,
		states:   make([]JobState, len(jobs)),
		errs:     make([]string, len(jobs)),
		progress: make([]float64, len(jobs)),
		state:    StateRunning,
	}
	for i, j := range jobs {
		if _, err := os.Stat(j.Output); err == nil {
			m.existing = append(m.existing, i)
		}
	}
	if len(m.existing) > 0 {
		m.state = StateConfirm
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return beginMsg{} }
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case beginMsg:
		if m.state == StateRunning {
			m.startNext()
		}
		return m, nil

	case progressMsg:
		if msg.job >= 0 && msg.job < len(m.jobs) && m.states[msg.job] == JobRunning {
			m.progress[msg.job] = min(1, msg.frac)
		}
		return m, nil

	case jobDoneMsg:
		i := msg.job
		if i < 0 || i >= len(m.jobs) {
			return m, nil
		}
		if msg.err != nil {
			m.states[i] = JobFailed
			m.errs[i] = msg.err.Error()
			os.Remove(m.jobs[i].Output)
			if m.aborted {
				m.finish()
				return m, nil
			}
		} else {
			m.states[i] = JobDone
			m.progress[i] = 1
		}
		m.startNext()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.state {
	case StateConfirm:
		switch msg.String() {
		case "o":
			m.state = StateRunning
			m.startNext()
		case "s":
			for _, i := range m.existing {
				m.states[i] = JobSkipped
			}
			m.state = StateRunning
			m.startNext()
		case "c", "esc":
			return m, func() tea.Msg { return BackMsg{} }
		}
		return m, nil

	case StateRunning:
		switch msg.String() {
		case "esc":
			m.abort()
		}
		return m, nil

	case StateDone:
		switch msg.String() {
		case "esc", "enter":
			return m, func() tea.Msg { return BackMsg{} }
		}
	}
	return m, nil
}

func (m *Model) abort() {
	m.aborted = true
	if m.cancel != nil {
		m.cancel()
	}
	for i, s := range m.states {
		if s == JobPending {
			m.states[i] = JobSkipped
			m.errs[i] = "cancelled"
		}
	}
	if !anyRunning(m.states) {
		m.finish()
	}
}

func anyRunning(states []JobState) bool {
	for _, s := range states {
		if s == JobRunning {
			return true
		}
	}
	return false
}

func (m *Model) finish() {
	m.state = StateDone
	m.cancel = nil
}

func (m *Model) startNext() {
	if m.aborted {
		m.finish()
		return
	}
	for i, s := range m.states {
		if s != JobPending {
			continue
		}
		m.startJob(i)
		return
	}
	m.finish()
}

func (m *Model) startJob(i int) {
	if err := os.MkdirAll(m.outDir, 0o755); err != nil {
		m.states[i] = JobFailed
		m.errs[i] = "create output dir: " + err.Error()
		m.startNext()
		return
	}
	m.states[i] = JobRunning
	job := m.jobs[i]
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	send := m.send
	tools := m.tools
	go func() {
		defer cancel()
		cmd := media.CutCommand(ctx, tools, job)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			send(jobDoneMsg{i, err})
			return
		}
		var errBuf bytes.Buffer
		cmd.Stderr = &errBuf
		if err := cmd.Start(); err != nil {
			send(jobDoneMsg{i, err})
			return
		}
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if secs, ok := media.ParseProgressLine(sc.Text()); ok && job.Duration() > 0 {
				send(progressMsg{i, secs / job.Duration()})
			}
		}
		if err := cmd.Wait(); err != nil {
			detail := strings.TrimSpace(errBuf.String())
			if ctx.Err() != nil {
				detail = "cancelled"
			} else if detail == "" {
				detail = err.Error()
			}
			send(jobDoneMsg{i, fmt.Errorf("%s", detail)})
			return
		}
		send(jobDoneMsg{i, nil})
	}()
}

func (m Model) Counts() (done, failed, skipped int) {
	for _, s := range m.states {
		switch s {
		case JobDone:
			done++
		case JobFailed:
			failed++
		case JobSkipped:
			skipped++
		}
	}
	return
}
