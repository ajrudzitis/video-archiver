package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/media"
	"video-archiver/internal/render"
	"video-archiver/internal/ui/browser"
	"video-archiver/internal/ui/editor"
	"video-archiver/internal/ui/jobs"
)

type screen int

const (
	screenBrowser screen = iota
	screenEditor
	screenJobs
)

type Config struct {
	Dir     string
	File    string
	Preview string
	Tools   media.Tools
	Runtime *Runtime
}

type Model struct {
	cfg       Config
	width     int
	height    int
	screen    screen
	browser   browser.Model
	editor    editor.Model
	hasEditor bool
	jobsModel jobs.Model
	hasJobs   bool
	renderer  render.Renderer
	status    string
}

type probeDoneMsg struct{ info media.Info }
type probeErrMsg struct{ err error }

func New(cfg Config) Model {
	return Model{
		cfg:      cfg,
		browser:  browser.New(cfg.Dir),
		renderer: render.New(render.Detect(cfg.Preview)),
		screen:   screenBrowser,
	}
}

func (m Model) Init() tea.Cmd {
	if m.cfg.File != "" {
		m.status = "probing " + m.cfg.File
		return m.probeCmd(m.cfg.File)
	}
	return nil
}

func (m Model) probeCmd(path string) tea.Cmd {
	tools := m.cfg.Tools
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		info, err := media.Probe(ctx, tools, path)
		if err != nil {
			return probeErrMsg{err}
		}
		return probeDoneMsg{info}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browser.SetSize(msg.Width, msg.Height-2)
		m.editor.SetSize(msg.Width, msg.Height)
		m.jobsModel.SetSize(msg.Width, msg.Height)

	case browser.OpenMsg:
		m.status = "probing " + msg.Path
		return m, m.probeCmd(msg.Path)

	case probeDoneMsg:
		m.editor = editor.New(m.cfg.Tools, m.renderer, msg.info)
		m.editor.SetSize(m.width, m.height)
		m.hasEditor = true
		m.screen = screenEditor
		m.status = ""
		return m, m.editor.Init()

	case probeErrMsg:
		m.status = "error: " + msg.err.Error()

	case editor.BackMsg:
		m.screen = screenBrowser
		m.browser.Load()

	case editor.RunJobsMsg:
		m.jobsModel = jobs.New(m.cfg.Runtime.Send, m.cfg.Tools, msg.OutDir, msg.Jobs)
		m.jobsModel.SetSize(m.width, m.height)
		m.hasJobs = true
		m.screen = screenJobs
		return m, m.jobsModel.Init()

	case jobs.BackMsg:
		m.screen = screenEditor

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.screen {
		case screenBrowser:
			if msg.String() == "q" {
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		case screenEditor:
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		case screenJobs:
			if msg.String() == "q" {
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.jobsModel, cmd = m.jobsModel.Update(msg)
			return m, cmd
		}
	default:
		if m.screen == screenEditor && m.hasEditor {
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}
		if m.screen == screenJobs && m.hasJobs {
			var cmd tea.Cmd
			m.jobsModel, cmd = m.jobsModel.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.screen {
	case screenEditor:
		if m.hasEditor {
			return m.editor.View()
		}
	case screenJobs:
		if m.hasJobs {
			return m.jobsModel.View()
		}
	}
	v := m.browser.View()
	if m.status != "" {
		v += "\n" + m.status
	}
	if c := m.renderer.Clear(); c != "" {
		v += "\n" + c
	}
	return v
}
