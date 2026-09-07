package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"video-archiver/internal/render"
	"video-archiver/internal/timefmt"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	markStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	selStyle    = lipgloss.NewStyle().Background(lipgloss.Color("237"))
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241"))
)

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// layout computes the preview pane size (including border) and the right
// pane width.
func (m Model) layout() (prevW, mainH, rightW int) {
	mainH = max(8, m.height-4)
	prevW = clampInt(m.width*4/10, 24, 80)
	if minRight := 40; m.width-prevW-1 < minRight {
		prevW = max(20, m.width-minRight-1)
	}
	rightW = max(0, m.width-prevW-2)
	return prevW, mainH, rightW
}

func (m Model) previewPixelWidth() int {
	prevW, _, _ := m.layout()
	return (prevW - 2) * 8
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.showHelp {
		return m.helpView()
	}
	prevW, mainH, rightW := m.layout()

	left := m.previewPane(prevW, mainH)
	right := m.rightPane(rightW, mainH)
	main := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)

	var b strings.Builder
	b.WriteString(m.headerLine() + "\n")
	b.WriteString(main + "\n")
	b.WriteString(m.promptOrStatusLine())
	return b.String()
}

func (m Model) helpView() string {
	lines := []string{
		headerStyle.Render("video-archiver help"),
		"",
		accentStyle.Render("transport"),
		"  ←/→             seek ±1s",
		"  shift+←/→, {/}  seek ±10s",
		"  , / .           step one frame back/forward",
		"  home / end      jump to start/end",
		"  :               jump to timestamp",
		"",
		accentStyle.Render("marking"),
		"  i               set in-point at cursor",
		"  o               set out-point, then name the segment",
		"",
		accentStyle.Render("segment list (tab to focus)"),
		"  ↑/↓             select segment",
		"  enter           jump to segment start",
		"  t / d           rename / delete",
		"  I / O           set start/end at cursor",
		"  [ ] / { }       nudge start / end by one frame",
		"",
		accentStyle.Render("output"),
		"  O               set output directory",
		"  x               run the split",
		"  s               save sidecar now",
		"",
		accentStyle.Render("other"),
		"  esc             back / close dialog",
		"  q               quit",
		"",
		dimStyle.Render("segments autosave to <video>.segments.json · outputs are H.264/AAC .mp4 named after titles"),
		dimStyle.Render("press any key to close"),
	}
	box := borderStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) headerLine() string {
	i := m.info
	return headerStyle.Render(truncate(i.Path, max(10, m.width-48))) + dimStyle.Render(fmt.Sprintf(
		"  %s · %dx%d @ %.2ffps · %s/%s",
		timefmt.Format(i.Duration), i.Width, i.Height, i.FPS, orDash(i.VideoCodec), orDash(i.AudioCodec),
	))
}

func (m Model) previewPane(w, h int) string {
	innerW, innerH := w-2, h-2
	var content string
	switch {
	case m.renderer.Kind() == render.KindNone:
		content = centerText("preview off (-preview flag)", innerW, innerH)
	case m.previewPNG == nil:
		content = centerText("extracting frame…", innerW, innerH)
	default:
		content = m.renderer.RenderPNG(m.previewPNG, innerW, innerH)
	}

	lines := strings.Split(content, "\n")
	padded := make([]string, 0, innerH)
	for i := 0; i < innerH; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		padded = append(padded, padRight(line, innerW))
	}
	return borderStyle.Width(w - 2).Render(strings.Join(padded, "\n"))
}

func (m Model) rightPane(w, h int) string {
	if w <= 0 {
		return ""
	}
	var lines []string
	add := func(s string) { lines = append(lines, padRight(s, w)) }

	add(accentStyle.Render("position ") + timefmt.Format(m.pos) + dimStyle.Render(" / "+timefmt.Format(m.info.Duration)))
	add(m.positionBar(w))
	add("")

	marks := dimStyle.Render("in  ") + formatMark(m.hasIn, m.inPoint) + "  " + dimStyle.Render("out ") + formatMark(m.hasOut, m.outPoint)
	add(marks)
	add("")

	add(headerStyle.Render("segments") + dimStyle.Render(fmt.Sprintf("  (%d)", len(m.file.Segments))))
	listRows := h - len(lines) - 3
	if listRows < 1 {
		listRows = 1
	}
	for _, row := range m.segmentRows(w, listRows) {
		add(row)
	}
	for len(lines) < h-2 {
		add("")
	}
	add(dimStyle.Render("output " + truncate(m.file.OutputDir, w-7)))
	add(dimStyle.Render(truncate(helpText(m.focus), w)))
	return strings.Join(lines[:min(len(lines), h)], "\n")
}

func formatMark(set bool, v float64) string {
	if !set {
		return dimStyle.Render("--:--")
	}
	return markStyle.Render(timefmt.Format(v))
}

func (m Model) positionBar(w int) string {
	if m.info.Duration <= 0 || w < 4 {
		return ""
	}
	frac := m.pos / m.info.Duration
	pos := int(frac * float64(w-1))
	var b strings.Builder
	for i := 0; i < w; i++ {
		switch {
		case i == pos:
			b.WriteString(accentStyle.Render("●"))
		case i < pos:
			b.WriteString(dimStyle.Render("─"))
		default:
			b.WriteString(dimStyle.Render("·"))
		}
	}
	return b.String()
}

func (m Model) segmentRows(w, maxRows int) []string {
	segs := m.file.Sorted()
	if len(segs) == 0 {
		return []string{dimStyle.Render("none yet — press i then o")}
	}
	off := 0
	if m.sel >= maxRows {
		off = m.sel - maxRows + 1
	}
	var rows []string
	for i := off; i < len(segs) && len(rows) < maxRows; i++ {
		s := segs[i]
		row := fmt.Sprintf("%2d %-8s %-8s %-8s %s",
			i+1,
			timefmt.FormatShort(s.Start),
			timefmt.FormatShort(s.End),
			timefmt.FormatShort(s.Duration()),
			truncate(s.Title, max(0, w-30)),
		)
		if i == m.sel && m.focus == focusList {
			row = selStyle.Render(padRight(row, w))
		} else if i == m.sel {
			row = accentStyle.Render(row)
		}
		rows = append(rows, row)
	}
	return rows
}

func (m Model) promptOrStatusLine() string {
	if m.prompt != promptNone {
		return m.input.View()
	}
	if m.status != "" {
		return statusStyle.Render(truncate(m.status, max(0, m.width)))
	}
	return ""
}

func padRight(s string, w int) string {
	if width := lipgloss.Width(s); width >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func truncate(s string, w int) string {
	if w <= 0 || lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return s[:w]
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

func centerText(s string, w, h int) string {
	if h <= 0 {
		return ""
	}
	pad := strings.Repeat("\n", h/2)
	line := dimStyle.Render(s)
	if lipgloss.Width(s) < w {
		line = strings.Repeat(" ", (w-lipgloss.Width(s))/2) + line
	}
	return pad + line
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
