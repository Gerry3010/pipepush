package tui

import (
	"github.com/Gerry3010/pipepush/internal/models"
)

// Async messages used by the TUI update loop.

type errMsg struct{ err error }

type loginOKMsg struct {
	resp     *models.LoginResponse
	email    string
	password string
}

type projectsLoadedMsg struct{ projects []*models.Project }

type pipelinesLoadedMsg struct{ pipelines []*models.Pipeline }

type runsLoadedMsg struct{ runs []*models.Run }

type tokensLoadedMsg struct{ tokens []*models.NotificationToken }

type tokenCreatedMsg struct{ plaintext string }

type actionDoneMsg struct{ message string }

// sseEventMsg is delivered when a realtime run update arrives.
type sseEventMsg struct{ event models.SSEEvent }
