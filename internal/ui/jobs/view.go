package jobs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"video-archiver/internal/media"
	"video-archiver/internal/timefmt"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	skipStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	runStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("splitting %d segment(s) → %s", len(m.jobs), m.outDir)) + "\n\n")

	for i, j := range m.jobs {
		b.WriteString(m.jobLine(i, j) + "\n")
		if m.errs[i] != "" && (m.states[i] == JobFailed) {
			b.WriteString("   " + failStyle.Render(truncate(m.errs[i], max(0, m.width-4))) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("h help · "))
	switch m.state {
	case StateConfirm:
		b.WriteString(skipStyle.Render("some output files already exist:"))
		b.WriteString(dimStyle.Render("  o) overwrite · s) skip existing · c) cancel"))
	case StateRunning:
		done, failed, _ := m.Counts()
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d/%d done", done+failed, len(m.jobs))))
		b.WriteString(dimStyle.Render(" · esc cancel"))
	case StateDone:
		done, failed, skipped := m.Counts()
		summary := fmt.Sprintf("finished: %d done", done)
		if failed > 0 {
			summary += fmt.Sprintf(", %d failed", failed)
		}
		if skipped > 0 {
			summary += fmt.Sprintf(", %d skipped", skipped)
		}
		if failed > 0 {
			b.WriteString(failStyle.Render(summary))
		} else {
			b.WriteString(okStyle.Render(summary))
		}
		b.WriteString(dimStyle.Render(" · esc back to editor"))
	}
	return b.String()
}

func (m Model) jobLine(i int, j media.Job) string {
	rangeStr := timefmt.FormatShort(j.Start) + "–" + timefmt.FormatShort(j.End)
	name := truncate(j.Title, 28)
	switch m.states[i] {
	case JobDone:
		return okStyle.Render(" ✓ ") + name + dimStyle.Render("  "+rangeStr)
	case JobFailed:
		return failStyle.Render(" ✗ ") + name + dimStyle.Render("  "+rangeStr)
	case JobSkipped:
		return skipStyle.Render(" ↓ ") + name + dimStyle.Render("  "+rangeStr)
	case JobRunning:
		bar := progressBar(m.progress[i], 24)
		return runStyle.Render(" ▶ ") + name + dimStyle.Render("  "+rangeStr+"  ") + bar +
			dimStyle.Render(fmt.Sprintf(" %3.0f%%", m.progress[i]*100))
	default:
		return dimStyle.Render(" · ") + name + dimStyle.Render("  "+rangeStr)
	}
}

func progressBar(frac float64, w int) string {
	if w <= 0 {
		return ""
	}
	filled := int(frac * float64(w))
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", w-filled) + "]"
}

func truncate(s string, w int) string {
	if w <= 0 || lipgloss.Width(s) <= w {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}
