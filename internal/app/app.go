package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/media"
)

type Config struct {
	Dir     string
	File    string
	Preview string
	Tools   media.Tools
}

type Model struct {
	cfg    Config
	width  int
	height int
}

func New(cfg Config) Model {
	return Model{cfg: cfg}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	return "video-archiver: scaffold (q to quit)\n"
}
