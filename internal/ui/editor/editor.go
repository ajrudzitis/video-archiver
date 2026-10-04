package editor

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/media"
	"video-archiver/internal/render"
	"video-archiver/internal/segments"
	"video-archiver/internal/timefmt"
)

const previewDebounce = 120 * time.Millisecond

type focusArea int

const (
	focusTransport focusArea = iota
	focusList
)

type promptMode int

const (
	promptNone promptMode = iota
	promptTitle
	promptRename
	promptOutputDir
	promptJump
)

type BackMsg struct{}

type RunJobsMsg struct {
	Jobs   []media.Job
	OutDir string
}

type frameMsg struct {
	tick int
	at   float64
	data []byte
	err  error
}

type previewDueMsg struct{ tick int }

type Model struct {
	tools    media.Tools
	renderer render.Renderer
	info     media.Info
	file     *segments.File
	sidecar  string

	width  int
	height int

	pos      float64
	inPoint  float64
	outPoint float64
	hasIn    bool
	hasOut   bool

	focus focusArea
	sel   int

	prompt promptMode
	input  textinput.Model

	previewPNG []byte
	previewAt  float64
	tick       int

	showHelp bool
	status   string
	warning  string
}

func New(tools media.Tools, renderer render.Renderer, info media.Info) Model {
	sidecar := segments.SidecarPath(info.Path)
	file, err := segments.Load(sidecar)
	if err != nil || file == nil {
		file = &segments.File{Source: filepath.Base(info.Path)}
	}
	file.Duration = info.Duration
	if file.OutputDir == "" {
		file.OutputDir = segments.DefaultOutputDir(info.Path)
	}

	var warning string
	if err := file.Validate(); err != nil {
		warning = "segments need attention: " + err.Error()
	}

	ti := textinput.New()
	ti.CharLimit = 200

	m := Model{
		tools:    tools,
		renderer: renderer,
		info:     info,
		file:     file,
		sidecar:  sidecar,
		input:    ti,
		warning:  warning,
	}
	if len(file.Segments) > 0 {
		m.sel = 0
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return m.requestPreview()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) frameStep() float64 {
	fps := m.info.FPS
	if fps <= 0 {
		fps = 25
	}
	return 1 / fps
}

func (m *Model) setPos(v float64) {
	if v < 0 {
		v = 0
	}
	if v > m.info.Duration {
		v = m.info.Duration
	}
	m.pos = v
}

func (m *Model) save() {
	if err := m.file.Save(m.sidecar); err != nil {
		m.status = "save error: " + err.Error()
	}
}

func (m Model) sortedIndex(i int) int {
	// index in file.Segments of the i-th segment in sorted order
	sorted := m.file.Sorted()
	if i < 0 || i >= len(sorted) {
		return -1
	}
	want := sorted[i]
	for j, s := range m.file.Segments {
		if s == want {
			return j
		}
	}
	return -1
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case previewDueMsg:
		if msg.tick != m.tick || m.renderer.Kind() == render.KindNone {
			return m, nil
		}
		if m.previewPNG != nil && m.previewAt == m.pos {
			return m, nil
		}
		return m, m.frameCmd(msg.tick)

	case frameMsg:
		if msg.tick != m.tick {
			return m, nil
		}
		if msg.err != nil {
			m.status = "preview: " + msg.err.Error()
			return m, nil
		}
		m.previewPNG = msg.data
		m.previewAt = msg.at
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}
	if m.prompt != promptNone {
		return m.handlePromptKey(msg)
	}
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		if m.focus == focusList {
			m.focus = focusTransport
			return m, nil
		}
		return m, func() tea.Msg { return BackMsg{} }
	case "tab":
		if len(m.file.Segments) > 0 {
			m.focus = (m.focus + 1) % 2
		}
		return m, nil
	case "?":
		m.showHelp = true
		return m, nil
	}
	if m.focus == focusList {
		return m.handleListKey(msg)
	}
	return m.handleTransportKey(msg)
}

func (m Model) handleTransportKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	step := m.frameStep()
	switch msg.String() {
	case "left":
		m.setPos(m.pos - 1)
	case "right":
		m.setPos(m.pos + 1)
	case "shift+left", "{":
		m.setPos(m.pos - 10)
	case "shift+right", "}":
		m.setPos(m.pos + 10)
	case ",":
		m.setPos(m.pos - step)
	case ".":
		m.setPos(m.pos + step)
	case "home":
		m.setPos(0)
	case "end":
		m.setPos(m.info.Duration)
	case "i":
		m.inPoint = m.pos
		m.hasIn = true
		m.status = fmt.Sprintf("in point %s", timefmt.Format(m.inPoint))
		return m, nil
	case "o":
		m.outPoint = m.pos
		m.hasOut = true
		if !m.hasIn || m.outPoint <= m.inPoint {
			m.status = fmt.Sprintf("out point %s (in point must be earlier)", timefmt.Format(m.outPoint))
			return m, nil
		}
		return m, m.startPrompt(promptTitle, "")
	case ":":
		return m, m.startPrompt(promptJump, timefmt.Format(m.pos))
	case "O":
		return m, m.startPrompt(promptOutputDir, m.file.OutputDir)
	case "s":
		m.save()
		m.status = "saved " + m.sidecar
		return m, nil
	case "x":
		return m.runJobs()
	}
	return m, m.schedulePreview()
}

func (m Model) handleListKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	n := len(m.file.Segments)
	if n == 0 {
		m.focus = focusTransport
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		m.sel = max(0, m.sel-1)
		return m, nil
	case "down", "j":
		m.sel = min(n-1, m.sel+1)
		return m, nil
	case "enter":
		s := m.file.Sorted()[m.sel]
		m.setPos(s.Start)
		return m, m.schedulePreview()
	case "d":
		idx := m.sortedIndex(m.sel)
		if idx >= 0 {
			removed := m.file.Segments[idx]
			m.file.Segments = append(m.file.Segments[:idx], m.file.Segments[idx+1:]...)
			m.sel = min(m.sel, len(m.file.Segments)-1)
			m.save()
			m.status = "deleted " + removed.Title
		}
		return m, nil
	case "t":
		s := m.file.Sorted()[m.sel]
		return m, m.startPrompt(promptRename, s.Title)
	case "I":
		return m.setSelStart(m.pos)
	case "O":
		return m.setSelEnd(m.pos)
	case "[":
		return m.nudgeSel(true, -m.frameStep())
	case "]":
		return m.nudgeSel(true, m.frameStep())
	case "{":
		return m.nudgeSel(false, -m.frameStep())
	case "}":
		return m.nudgeSel(false, m.frameStep())
	}
	return m, nil
}

func (m Model) setSelStart(v float64) (Model, tea.Cmd) {
	idx := m.sortedIndex(m.sel)
	if idx < 0 {
		return m, nil
	}
	s := m.file.Segments[idx]
	if v >= s.End {
		m.status = "start must be before end"
		return m, nil
	}
	s.Start = v
	m.file.Segments[idx] = s
	m.save()
	return m, nil
}

func (m Model) setSelEnd(v float64) (Model, tea.Cmd) {
	idx := m.sortedIndex(m.sel)
	if idx < 0 {
		return m, nil
	}
	s := m.file.Segments[idx]
	if v <= s.Start {
		m.status = "end must be after start"
		return m, nil
	}
	if v > m.info.Duration {
		v = m.info.Duration
	}
	s.End = v
	m.file.Segments[idx] = s
	m.save()
	return m, nil
}

func (m Model) nudgeSel(start bool, delta float64) (Model, tea.Cmd) {
	idx := m.sortedIndex(m.sel)
	if idx < 0 {
		return m, nil
	}
	s := m.file.Segments[idx]
	if start {
		v := s.Start + delta
		if v < 0 || v >= s.End {
			return m, nil
		}
		s.Start = v
	} else {
		v := s.End + delta
		if v <= s.Start || v > m.info.Duration {
			return m, nil
		}
		s.End = v
	}
	m.file.Segments[idx] = s
	m.save()
	return m, nil
}

func (m *Model) startPrompt(mode promptMode, initial string) tea.Cmd {
	m.prompt = mode
	m.input.Reset()
	m.input.SetValue(initial)
	switch mode {
	case promptTitle:
		m.input.Placeholder = "segment title"
		m.input.Prompt = "title> "
	case promptRename:
		m.input.Placeholder = "segment title"
		m.input.Prompt = "rename> "
	case promptOutputDir:
		m.input.Placeholder = "output directory"
		m.input.Prompt = "output> "
	case promptJump:
		m.input.Placeholder = "HH:MM:SS.mmm"
		m.input.Prompt = "jump> "
	}
	return m.input.Focus()
}

func (m Model) handlePromptKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.prompt = promptNone
		m.input.Blur()
		return m, nil
	case "enter":
		value := m.input.Value()
		mode := m.prompt
		m.prompt = promptNone
		m.input.Blur()
		return m.confirmPrompt(mode, value)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) confirmPrompt(mode promptMode, value string) (Model, tea.Cmd) {
	switch mode {
	case promptTitle:
		title := segments.SanitizeTitle(value)
		if title == "" {
			m.status = "title must not be empty"
			return m, nil
		}
		if m.file.HasDuplicateTitle(title, -1) {
			m.status = "a segment titled " + title + " already exists"
			return m, nil
		}
		if !m.hasIn || !m.hasOut || m.outPoint <= m.inPoint {
			m.status = "mark in and out points first (i / o)"
			return m, nil
		}
		seg := segments.Segment{Start: m.inPoint, End: m.outPoint, Title: title}
		m.file.Segments = append(m.file.Segments, seg)
		m.sel = m.indexOfSorted(seg)
		m.hasIn, m.hasOut = false, false
		m.save()
		m.status = fmt.Sprintf("added %s (%s)", title, timefmt.Format(seg.Duration()))
		return m, nil

	case promptRename:
		title := segments.SanitizeTitle(value)
		idx := m.sortedIndex(m.sel)
		if idx < 0 {
			return m, nil
		}
		if title == "" {
			m.status = "title must not be empty"
			return m, nil
		}
		if m.file.HasDuplicateTitle(title, idx) {
			m.status = "a segment titled " + title + " already exists"
			return m, nil
		}
		m.file.Segments[idx].Title = title
		m.save()
		m.status = "renamed to " + title
		return m, nil

	case promptOutputDir:
		dir := value
		if dir == "" {
			dir = segments.DefaultOutputDir(m.info.Path)
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(filepath.Dir(m.info.Path), dir)
		}
		m.file.OutputDir = dir
		m.save()
		m.status = "output dir: " + dir
		return m, nil

	case promptJump:
		v, err := timefmt.Parse(value)
		if err != nil {
			m.status = "jump: " + err.Error()
			return m, nil
		}
		m.setPos(v)
		return m, m.schedulePreview()
	}
	return m, nil
}

func (m Model) indexOfSorted(seg segments.Segment) int {
	for i, s := range m.file.Sorted() {
		if s == seg {
			return i
		}
	}
	return 0
}

func (m Model) runJobs() (Model, tea.Cmd) {
	if len(m.file.Segments) == 0 {
		m.status = "no segments defined"
		return m, nil
	}
	if err := m.file.Validate(); err != nil {
		m.status = "fix invalid segments first: " + err.Error()
		return m, nil
	}
	outDir := m.file.OutputDir
	if outDir == "" {
		outDir = segments.DefaultOutputDir(m.info.Path)
	}
	jobs := BuildJobs(m.info.Path, outDir, m.file.Sorted())
	return m, func() tea.Msg { return RunJobsMsg{Jobs: jobs, OutDir: outDir} }
}

func BuildJobs(source, outDir string, segs []segments.Segment) []media.Job {
	jobs := make([]media.Job, 0, len(segs))
	for _, s := range segs {
		jobs = append(jobs, media.Job{
			Source: source,
			Output: filepath.Join(outDir, segments.OutputName(s.Title)),
			Start:  s.Start,
			End:    s.End,
			Title:  s.Title,
		})
	}
	return jobs
}

func (m *Model) schedulePreview() tea.Cmd {
	if m.renderer.Kind() == render.KindNone {
		return nil
	}
	m.tick++
	tick := m.tick
	return tea.Tick(previewDebounce, func(time.Time) tea.Msg {
		return previewDueMsg{tick}
	})
}

func (m Model) requestPreview() tea.Cmd {
	if m.renderer.Kind() == render.KindNone {
		return nil
	}
	return m.frameCmd(m.tick)
}

func (m Model) frameCmd(tick int) tea.Cmd {
	tools := m.tools
	path := m.info.Path
	at := m.pos
	width := m.previewPixelWidth()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		data, err := media.ExtractFrame(ctx, tools, path, at, width)
		return frameMsg{tick: tick, at: at, data: data, err: err}
	}
}

func helpText(f focusArea) string {
	if f == focusList {
		return "↑/↓ select · enter jump · t rename · d delete · I/O set start/end at cursor · [/] nudge start · {/} nudge end · tab transport"
	}
	return "←/→ ±1s · {/} or shift+←/→ ±10s · ,/. frame · i in · o out+title · : jump · O output dir · tab segments · x run · s save · ? help"
}
