package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.Color("#7D56F4")
	colorSuccess = lipgloss.Color("#43BF6D")
	colorError   = lipgloss.Color("#F2495C")
	colorWarning = lipgloss.Color("#FF9830")
	colorMuted   = lipgloss.Color("#6C7086")
	colorRunning = lipgloss.Color("#5794F2")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 1)

	breadcrumbStyle = lipgloss.NewStyle().Foreground(colorMuted)

	helpStyle = lipgloss.NewStyle().Foreground(colorMuted).MarginTop(1)

	errorStyle = lipgloss.NewStyle().Foreground(colorError).Bold(true)

	successStyle = lipgloss.NewStyle().Foreground(colorSuccess)

	docStyle = lipgloss.NewStyle().Margin(1, 2)

	inputLabelStyle = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
)

// statusColor returns the lipgloss color for a run status.
func statusColor(status string) lipgloss.Color {
	switch status {
	case "success":
		return colorSuccess
	case "failure":
		return colorError
	case "cancelled":
		return colorMuted
	case "running":
		return colorRunning
	case "skipped":
		return colorWarning
	default:
		return colorMuted
	}
}

func statusGlyph(status string) string {
	switch status {
	case "success":
		return "✓"
	case "failure":
		return "✗"
	case "cancelled":
		return "⊘"
	case "running":
		return "●"
	case "skipped":
		return "○"
	default:
		return "•"
	}
}
