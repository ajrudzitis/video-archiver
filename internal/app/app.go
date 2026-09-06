package app

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/media"
	"video-archiver/internal/render"
	"video-archiver/internal/timefmt"
	"video-archiver/internal/ui/browser"
)

type screen int

const (
	screenBrowser screen = iota
	screenEditor
)

type Config struct {
	Dir     string
	File    string
	Preview string
	Tools   media.Tools
}

type Model struct {
	cfg      Config
	width    int
	height   int
	screen   screen
	browser  browser.Model
	renderer render.Renderer
	info     media.Info
	status   string
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

	case browser.OpenMsg:
		m.status = "probing " + msg.Path
		return m, m.probeCmd(msg.Path)

	case probeDoneMsg:
		m.info = msg.info
		m.status = ""
		m.screen = screenEditor

	case probeErrMsg:
		m.status = "error: " + msg.err.Error()

	case tea.KeyMsg:
		switch m.screen {
		case screenBrowser:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Update(msg)
			return m, cmd
		case screenEditor:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.screen = screenBrowser
				m.browser.Load()
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.screen {
	case screenEditor:
		return m.editorPlaceholderView()
	default:
		v := m.browser.View()
		if m.status != "" {
			v += "\n" + m.status
		}
		return v
	}
}

func (m Model) editorPlaceholderView() string {
	i := m.info
	return fmt.Sprintf(
		"%s\n\nduration %s · %dx%d @ %.2f fps · video %s · audio %s\n\n(editor coming soon — esc to go back, q to quit)",
		i.Path, timefmt.Format(i.Duration), i.Width, i.Height, i.FPS, i.VideoCodec, orNone(i.AudioCodec),
	)
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
