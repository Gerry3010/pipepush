package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gerry3010/pipepush/internal/client"
	"github.com/Gerry3010/pipepush/internal/crypto"
	"github.com/Gerry3010/pipepush/internal/models"
)

func loginCmd(api *client.Client, email, password string, register bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var resp *models.LoginResponse
		var err error
		if register {
			kp, kerr := crypto.GenerateKeyPair()
			if kerr != nil {
				return errMsg{kerr}
			}
			encPriv, salt, perr := crypto.EncryptPrivateKey(kp.PrivateKey.Bytes(), password)
			if perr != nil {
				return errMsg{perr}
			}
			resp, err = api.Register(ctx, models.RegisterRequest{
				Email:               email,
				Password:            password,
				PublicKey:           crypto.PublicKeyToBase64(kp.PublicKey),
				EncryptedPrivateKey: encPriv,
				KDFSalt:             salt,
			})
		} else {
			resp, err = api.Login(ctx, email, password)
		}
		if err != nil {
			return errMsg{err}
		}
		return loginOKMsg{resp: resp, email: email, password: password}
	}
}

func loadProjectsCmd(api *client.Client) tea.Cmd {
	return func() tea.Msg {
		projects, err := api.ListProjects(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return projectsLoadedMsg{projects}
	}
}

func loadPipelinesCmd(api *client.Client, projectID string) tea.Cmd {
	return func() tea.Msg {
		pipelines, err := api.ListPipelines(context.Background(), projectID)
		if err != nil {
			return errMsg{err}
		}
		return pipelinesLoadedMsg{pipelines}
	}
}

func loadRunsCmd(api *client.Client, pipelineID string) tea.Cmd {
	return func() tea.Msg {
		runs, err := api.ListRuns(context.Background(), pipelineID, 50)
		if err != nil {
			return errMsg{err}
		}
		return runsLoadedMsg{runs}
	}
}

func loadTokensCmd(api *client.Client, projectID string) tea.Cmd {
	return func() tea.Msg {
		tokens, err := api.ListTokens(context.Background(), projectID)
		if err != nil {
			return errMsg{err}
		}
		return tokensLoadedMsg{tokens}
	}
}

func createProjectCmd(api *client.Client, encName string) tea.Cmd {
	return func() tea.Msg {
		if _, err := api.CreateProject(context.Background(), encName, ""); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"project created"}
	}
}

func createPipelineCmd(api *client.Client, projectID, encName string) tea.Cmd {
	return func() tea.Msg {
		if _, err := api.CreatePipeline(context.Background(), projectID, encName); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"pipeline created"}
	}
}

func createTokenCmd(api *client.Client, encName, projectID, pipelineID string) tea.Cmd {
	return func() tea.Msg {
		resp, err := api.CreateToken(context.Background(), encName, projectID, pipelineID)
		if err != nil {
			return errMsg{err}
		}
		return tokenCreatedMsg{plaintext: resp.PlaintextToken}
	}
}

func deleteProjectCmd(api *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := api.DeleteProject(context.Background(), id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"project deleted"}
	}
}

func deletePipelineCmd(api *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := api.DeletePipeline(context.Background(), id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"pipeline deleted"}
	}
}

func revokeTokenCmd(api *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := api.RevokeToken(context.Background(), id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"token revoked"}
	}
}

// listenSSECmd subscribes to the SSE stream and forwards events into the program.
func listenSSECmd(api *client.Client, p *tea.Program) {
	_ = api.StreamEvents(context.Background(), func(e models.SSEEvent) {
		p.Send(sseEventMsg{event: e})
	})
}
