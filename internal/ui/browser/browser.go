package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"video-archiver/internal/segments"
)

var videoExtensions = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
	".mpeg": true, ".mpg": true, ".m2ts": true, ".mts": true,
	".ts": true, ".wmv": true, ".dv": true, ".webm": true,
	".m4v": true, ".3gp": true, ".vob": true, ".flv": true,
}

func IsVideoFile(name string) bool {
	return videoExtensions[strings.ToLower(filepath.Ext(name))]
}

type entry struct {
	name       string
	isDir      bool
	size       int64
	modTime    time.Time
	hasSidecar bool
}

type OpenMsg struct{ Path string }

type Model struct {
	dir     string
	entries []entry
	cursor  int
	offset  int
	width   int
	height  int
	err     error
}

func New(dir string) Model {
	m := Model{dir: dir}
	m.Load()
	return m
}

func (m *Model) Load() {
	items, err := os.ReadDir(m.dir)
	if err != nil {
		m.err = err
		m.entries = nil
		return
	}
	m.err = nil
	entries := make([]entry, 0, len(items)+1)
	if filepath.Dir(m.dir) != m.dir {
		entries = append(entries, entry{name: "..", isDir: true})
	}
	for _, it := range items {
		name := it.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if it.IsDir() {
			entries = append(entries, entry{name: name, isDir: true})
			continue
		}
		if !IsVideoFile(name) {
			continue
		}
		e := entry{name: name}
		if fi, err := it.Info(); err == nil {
			e.size = fi.Size()
			e.modTime = fi.ModTime()
		}
		if _, err := os.Stat(segments.SidecarPath(filepath.Join(m.dir, name))); err == nil {
			e.hasSidecar = true
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})
	m.entries = entries
	if m.cursor >= len(m.entries) {
		m.cursor = max(0, len(m.entries)-1)
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) selectedPath() string {
	if len(m.entries) == 0 {
		return ""
	}
	e := m.entries[m.cursor]
	if e.name == ".." {
		return filepath.Dir(m.dir)
	}
	return filepath.Join(m.dir, e.name)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-m.pageSize())
	case "pgdown":
		m.move(m.pageSize())
	case "home", "g":
		m.cursor = 0
		m.offset = 0
	case "end", "G":
		m.cursor = max(0, len(m.entries)-1)
	case "enter", "l", "right":
		if len(m.entries) == 0 {
			return m, nil
		}
		e := m.entries[m.cursor]
		path := m.selectedPath()
		if e.isDir {
			m.dir = path
			m.cursor = 0
			m.offset = 0
			m.Load()
			return m, nil
		}
		return m, func() tea.Msg { return OpenMsg{Path: path} }
	case "backspace", "h", "left":
		parent := filepath.Dir(m.dir)
		if parent != m.dir {
			m.dir = parent
			m.cursor = 0
			m.offset = 0
			m.Load()
		}
	case "r":
		m.Load()
	}
	return m, nil
}

func (m *Model) move(delta int) {
	if len(m.entries) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.entries) {
		m.cursor = len(m.entries) - 1
	}
}

func (m Model) pageSize() int {
	return max(1, m.height-4)
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	dirStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selStyle   = lipgloss.NewStyle().Background(lipgloss.Color("237"))
	markStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func (m Model) View() string {
	var b strings.Builder
	header := titleStyle.Render("video-archiver") + dimStyle.Render("  "+m.dir)
	b.WriteString(header + "\n")

	if m.err != nil {
		b.WriteString(errStyle.Render("error: "+m.err.Error()) + "\n")
	}

	page := m.pageSize()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+page {
		m.offset = m.cursor - page + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}

	if len(m.entries) == 0 && m.err == nil {
		b.WriteString(dimStyle.Render("no video files here") + "\n")
	}
	for i := m.offset; i < len(m.entries) && i < m.offset+page; i++ {
		b.WriteString(m.renderLine(i) + "\n")
	}
	for i := len(m.entries) - m.offset; i < page; i++ {
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render("↑/↓ browse · enter open · ⌫ up · r reload · q quit"))
	return b.String()
}

func (m Model) renderLine(i int) string {
	e := m.entries[i]
	var line string
	if e.isDir {
		line = dirStyle.Render(e.name + "/")
	} else {
		name := e.name
		if e.hasSidecar {
			name += " " + markStyle.Render("◆")
		}
		line = name + strings.Repeat(" ", max(0, 46-lipgloss.Width(name))) +
			dimStyle.Render(fmt.Sprintf("%10s  %s", humanSize(e.size), e.modTime.Format("2006-01-02")))
	}
	if i == m.cursor {
		line = selStyle.Render("❯ " + line)
	} else {
		line = "  " + line
	}
	if m.width > 0 {
		line = lipgloss.NewStyle().MaxWidth(m.width).Render(line)
	}
	return line
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
