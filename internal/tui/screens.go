package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gerry3010/pipepush/internal/client"
	"github.com/Gerry3010/pipepush/internal/crypto"
	"github.com/Gerry3010/pipepush/internal/routing"
)

// --- Login screen ---

func (m model) updateLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+r":
		m.registerMode = !m.registerMode
		return m, nil
	case "tab", "down":
		m.loginFocus = (m.loginFocus + 1) % 2
		return m.refocusLogin(), nil
	case "shift+tab", "up":
		m.loginFocus = (m.loginFocus + 1) % 2
		return m.refocusLogin(), nil
	case "enter":
		email := m.emailInput.Value()
		password := m.passwordInput.Value()
		if email == "" || password == "" {
			m.err = fmt.Errorf("email and password are required")
			return m, nil
		}
		m.err = nil
		m.status = "authenticating…"
		return m, loginCmd(m.api, email, password, m.registerMode)
	}

	var cmd tea.Cmd
	if m.loginFocus == 0 {
		m.emailInput, cmd = m.emailInput.Update(msg)
	} else {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m model) refocusLogin() model {
	if m.loginFocus == 0 {
		m.emailInput.Focus()
		m.passwordInput.Blur()
	} else {
		m.emailInput.Blur()
		m.passwordInput.Focus()
	}
	return m
}

func (m model) handleLoginOK(msg loginOKMsg) (tea.Model, tea.Cmd) {
	privBytes, err := crypto.DecryptPrivateKey(msg.resp.EncryptedPrivateKey, msg.resp.KDFSalt, msg.password)
	if err != nil {
		m.err = fmt.Errorf("could not unlock private key: %w", err)
		return m, nil
	}
	priv, err := crypto.PrivateKeyFromBytes(privBytes)
	if err != nil {
		m.err = err
		return m, nil
	}

	m.cfg.JWT = msg.resp.JWT
	m.cfg.Email = msg.email
	m.cfg.PublicKey = msg.resp.PublicKey
	m.cfg.PrivateKey = crypto.PrivateKeyToBase64(priv)
	if err := m.cfg.Save(); err != nil {
		m.err = err
		return m, nil
	}

	m.api = client.New(m.cfg.ServerURL, m.cfg.JWT)
	_ = m.loadKeys()
	m.status = ""
	m.err = nil

	if prog != nil {
		go listenSSECmd(m.api, prog)
	}
	return m, loadProjectsCmd(m.api)
}

func (m model) viewLogin() string {
	title := "Log in"
	if m.registerMode {
		title = "Create account"
	}
	b := titleStyle.Render("pipepush") + "\n\n"
	b += inputLabelStyle.Render(title) + "  " + breadcrumbStyle.Render("server: "+m.cfg.ServerURL) + "\n\n"
	b += "Email\n" + m.emailInput.View() + "\n\n"
	b += "Password\n" + m.passwordInput.View() + "\n\n"
	if m.status != "" {
		b += breadcrumbStyle.Render(m.status) + "\n"
	}
	if m.err != nil {
		b += errorStyle.Render("error: "+m.err.Error()) + "\n"
	}
	b += helpStyle.Render("tab: switch field · enter: submit · ctrl+r: toggle login/register · ctrl+c: quit")
	return docStyle.Render(b)
}

// --- Navigation (lists) ---

func (m model) updateNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Let the list handle filtering input first.
	if m.list.FilterState().String() == "filtering" {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		return m.goBack()
	case "enter":
		return m.drillDown()
	case "n":
		return m.startCreate()
	case "d":
		return m.deleteSelected()
	case "t":
		if m.screen == screenProjects {
			if it, ok := m.list.SelectedItem().(projectItem); ok {
				m.curProjectID = it.id
				m.curProjectName = it.name
				return m, loadTokensCmd(m.api, it.id)
			}
		}
	case "r":
		if m.screen == screenRuns {
			return m, loadRunsCmd(m.api, m.curPipelineID)
		}
		if m.screen == screenTokens {
			return m.deleteSelected() // 'r' = revoke on tokens screen
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) drillDown() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenProjects:
		if it, ok := m.list.SelectedItem().(projectItem); ok {
			m.curProjectID = it.id
			m.curProjectName = it.name
			return m, loadPipelinesCmd(m.api, it.id)
		}
	case screenPipelines:
		if it, ok := m.list.SelectedItem().(pipelineItem); ok {
			m.curPipelineID = it.id
			m.curPipeName = it.name
			return m, loadRunsCmd(m.api, it.id)
		}
	}
	return m, nil
}

func (m model) goBack() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenPipelines, screenTokens:
		return m, loadProjectsCmd(m.api)
	case screenRuns:
		return m, loadPipelinesCmd(m.api, m.curProjectID)
	}
	return m, nil
}

func (m model) deleteSelected() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenProjects:
		if it, ok := m.list.SelectedItem().(projectItem); ok {
			return m, deleteProjectCmd(m.api, it.id)
		}
	case screenPipelines:
		if it, ok := m.list.SelectedItem().(pipelineItem); ok {
			return m, deletePipelineCmd(m.api, it.id)
		}
	case screenTokens:
		if it, ok := m.list.SelectedItem().(tokenItem); ok {
			return m, revokeTokenCmd(m.api, it.id)
		}
	}
	return m, nil
}

// --- Creation prompt ---

func (m model) startCreate() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenProjects:
		m.inputMode = inputNewProject
		m.textInput.Placeholder = "Project name"
	case screenPipelines:
		m.inputMode = inputNewPipeline
		m.textInput.Placeholder = "Pipeline name"
	case screenTokens:
		m.inputMode = inputNewToken
		m.textInput.Placeholder = "Token name (e.g. GitHub Actions)"
	default:
		return m, nil
	}
	m.textInput.SetValue("")
	m.textInput.Focus()
	return m, textinput.Blink
}

func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = inputNone
		return m, nil
	case "enter":
		name := m.textInput.Value()
		if name == "" {
			m.inputMode = inputNone
			return m, nil
		}
		encName, err := crypto.EncryptString(m.pub, name)
		if err != nil {
			m.err = err
			m.inputMode = inputNone
			return m, nil
		}
		mode := m.inputMode
		m.inputMode = inputNone
		switch mode {
		case inputNewProject:
			return m, createProjectCmd(m.api, encName)
		case inputNewPipeline:
			return m, createPipelineCmd(m.api, m.curProjectID, encName, routing.Key(name))
		case inputNewToken:
			// Token is bound to the current pipeline if we have one in context.
			return m, createTokenCmd(m.api, encName, m.curProjectID, m.curPipelineID)
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) viewInput() string {
	var label string
	switch m.inputMode {
	case inputNewProject:
		label = "New project"
	case inputNewPipeline:
		label = "New pipeline in " + m.curProjectName
	case inputNewToken:
		label = "New token in " + m.curProjectName
	}
	b := titleStyle.Render("pipepush") + "\n\n"
	b += inputLabelStyle.Render(label) + "\n\n"
	b += m.textInput.View() + "\n\n"
	b += helpStyle.Render("enter: create · esc: cancel")
	return docStyle.Render(b)
}

func (m model) viewTokenShown() string {
	b := titleStyle.Render("pipepush") + "\n\n"
	b += successStyle.Render("✓ Token created") + "\n\n"
	b += "Copy this token now — it is shown only once:\n\n"
	b += inputLabelStyle.Render("  "+m.shownToken) + "\n\n"
	b += breadcrumbStyle.Render("Use it in CI/CD: pipepush send --token <token> --status success") + "\n\n"
	b += helpStyle.Render("press any key to continue")
	return docStyle.Render(b)
}
