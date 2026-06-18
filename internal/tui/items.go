package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/Gerry3010/pipepush/internal/models"
)

// projectItem adapts a Project to list.Item.
type projectItem struct {
	id   string
	name string
	desc string
}

func (i projectItem) Title() string       { return i.name }
func (i projectItem) Description() string { return i.desc }
func (i projectItem) FilterValue() string { return i.name }

// pipelineItem adapts a Pipeline to list.Item.
type pipelineItem struct {
	id   string
	name string
}

func (i pipelineItem) Title() string       { return i.name }
func (i pipelineItem) Description() string { return "pipeline" }
func (i pipelineItem) FilterValue() string { return i.name }

// runItem adapts a Run + decrypted payload to list.Item.
type runItem struct {
	status  string
	payload models.RunPayload
	when    string
}

func (i runItem) Title() string {
	badge := lipgloss.NewStyle().Foreground(statusColor(i.status)).Bold(true).
		Render(statusGlyph(i.status) + " " + i.status)
	t := badge
	if i.payload.Branch != "" {
		t += "  " + i.payload.Branch
	}
	if i.payload.Commit != "" {
		c := i.payload.Commit
		if len(c) > 8 {
			c = c[:8]
		}
		t += "  " + c
	}
	return t
}

func (i runItem) Description() string {
	d := i.when
	if i.payload.Message != "" {
		d += "  " + i.payload.Message
	}
	if i.payload.Duration != "" {
		d += "  (" + i.payload.Duration + ")"
	}
	return d
}
func (i runItem) FilterValue() string { return i.status + " " + i.payload.Branch }

// tokenItem adapts a NotificationToken to list.Item.
type tokenItem struct {
	id       string
	name     string
	active   bool
	lastUsed string
}

func (i tokenItem) Title() string {
	mark := "●"
	if !i.active {
		mark = "○"
	}
	return mark + " " + i.name
}
func (i tokenItem) Description() string { return "last used: " + i.lastUsed }
func (i tokenItem) FilterValue() string { return i.name }
