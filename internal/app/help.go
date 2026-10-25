package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) helpView() string {
	if m.width < 4 || m.height < 5 {
		return lipgloss.NewStyle().MaxWidth(max(1, m.width)).MaxHeight(max(1, m.height)).Render("Help · esc close")
	}
	title := "Help · browser"
	text := "↑/↓ or k/j: browse\nEnter or l: open file / directory\nBackspace or ←: parent directory\nHome/g, End/G: first / last entry\nPgUp/PgDown: page\nr: reload\nq: quit (confirm)"
	switch m.screen {
	case screenEditor:
		title = "Help · editor"
		text = "←/→: seek ±1s\nShift+←/→ or {/}: seek ±10s\n, / .: step one frame\nHome/End: start / end\n:: jump to timestamp\ni: set in-point\no: set out-point and name segment\nTab: switch transport / segments\n↑/↓: select segment · Enter: jump\nt/d: rename / delete segment\nI/O: set segment start / end at cursor\n[/]: nudge start · {/}: nudge end\nO: output directory · x: run split\ns: save segments\nEsc: back / close dialog\nq: quit (confirm)"
	case screenJobs:
		title = "Help · splitting"
		text = "When outputs already exist:\no: overwrite · s: skip existing · c: cancel\nWhile splitting:\nEsc: cancel the run\nWhen finished:\nEsc: back to editor\nq: quit (confirm)"
	}
	text += "\n\nh / ?: help · F1: help even while typing"
	w := min(72, m.width-2)
	wrapped := lipgloss.NewStyle().Width(w).Render(text)
	lines := strings.Split(wrapped, "\n")
	rows := max(1, m.height-6)
	offset := min(m.helpOffset, max(0, len(lines)-rows))
	body := strings.Join(lines[offset:min(len(lines), offset+rows)], "\n")
	lineStyle := lipgloss.NewStyle().MaxWidth(w)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(w).Render(
		lineStyle.Render(title) + "\n" + body + "\n" + lineStyle.Render("↑/↓ scroll · any other key closes"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
