package tui

import (
	"crypto/ecdh"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Gerry3010/pipepush/internal/client"
	"github.com/Gerry3010/pipepush/internal/config"
	"github.com/Gerry3010/pipepush/internal/crypto"
	"github.com/Gerry3010/pipepush/internal/models"
)

type screen int

const (
	screenLogin screen = iota
	screenProjects
	screenPipelines
	screenRuns
	screenTokens
	screenTokenShown
)

// inputMode is the kind of creation prompt currently active.
type inputMode int

const (
	inputNone inputMode = iota
	inputNewProject
	inputNewPipeline
	inputNewToken
)

type model struct {
	cfg *config.ClientConfig
	api *client.Client

	priv *ecdh.PrivateKey
	pub  *ecdh.PublicKey

	screen screen
	width  int
	height int

	// login
	emailInput    textinput.Model
	passwordInput textinput.Model
	loginFocus    int
	registerMode  bool

	// lists
	list list.Model

	// creation prompt
	inputMode  inputMode
	textInput  textinput.Model
	shownToken string

	// navigation context
	curProjectID   string
	curProjectName string
	curPipelineID  string
	curPipeName    string

	status string
	err    error
}

func initialModel(cfg *config.ClientConfig) model {
	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.Focus()
	email.CharLimit = 255
	email.Width = 40

	pass := textinput.New()
	pass.Placeholder = "password"
	pass.EchoMode = textinput.EchoPassword
	pass.EchoCharacter = '•'
	pass.CharLimit = 255
	pass.Width = 40

	ti := textinput.New()
	ti.CharLimit = 255
	ti.Width = 40

	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowHelp(false)

	m := model{
		cfg:           cfg,
		api:           client.New(cfg.ServerURL, cfg.JWT),
		emailInput:    email,
		passwordInput: pass,
		textInput:     ti,
		list:          l,
		screen:        screenLogin,
	}

	if cfg.Email != "" {
		m.emailInput.SetValue(cfg.Email)
	}
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// loadKeys decrypts the locally cached private key into the model.
func (m *model) loadKeys() error {
	priv, err := crypto.PrivateKeyFromBase64(m.cfg.PrivateKey)
	if err != nil {
		return err
	}
	pub, err := crypto.PublicKeyFromBase64(m.cfg.PublicKey)
	if err != nil {
		return err
	}
	m.priv = priv
	m.pub = pub
	return nil
}

func (m model) decrypt(ct string) string {
	if m.priv == nil {
		return "<locked>"
	}
	s, err := crypto.DecryptString(m.priv, ct)
	if err != nil {
		return "<decryption failed>"
	}
	return s
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-4)
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		// When a creation prompt is open, route keys there.
		if m.inputMode != inputNone {
			return m.updateInput(msg)
		}
		switch m.screen {
		case screenLogin:
			return m.updateLogin(msg)
		default:
			return m.updateNav(msg)
		}

	case errMsg:
		m.err = msg.err
		return m, nil

	case loginOKMsg:
		return m.handleLoginOK(msg)

	case projectsLoadedMsg:
		items := make([]list.Item, 0, len(msg.projects))
		for _, p := range msg.projects {
			items = append(items, projectItem{id: p.ID, name: m.decrypt(p.EncryptedName), desc: m.decrypt(p.EncryptedDescription)})
		}
		m.setList("Projects", items)
		m.screen = screenProjects
		return m, nil

	case pipelinesLoadedMsg:
		items := make([]list.Item, 0, len(msg.pipelines))
		for _, p := range msg.pipelines {
			items = append(items, pipelineItem{id: p.ID, name: m.decrypt(p.EncryptedName)})
		}
		m.setList("Pipelines · "+m.curProjectName, items)
		m.screen = screenPipelines
		return m, nil

	case runsLoadedMsg:
		m.setRuns(msg.runs)
		m.screen = screenRuns
		return m, nil

	case tokensLoadedMsg:
		items := make([]list.Item, 0, len(msg.tokens))
		for _, t := range msg.tokens {
			lu := "never"
			if t.LastUsedAt != nil {
				lu = t.LastUsedAt.Format("2006-01-02 15:04")
			}
			items = append(items, tokenItem{id: t.ID, name: m.decrypt(t.EncryptedName), active: t.Active, lastUsed: lu})
		}
		m.setList("Tokens · "+m.curProjectName, items)
		m.screen = screenTokens
		return m, nil

	case tokenCreatedMsg:
		m.shownToken = msg.plaintext
		m.screen = screenTokenShown
		return m, nil

	case actionDoneMsg:
		m.status = msg.message
		return m, m.reloadCurrent()

	case sseEventMsg:
		// Live update: if we're viewing runs, prepend the new one.
		if m.screen == screenRuns {
			m.prependRun(msg.event)
		}
		m.status = "● live update received"
		return m, nil
	}

	// Delegate to active component
	var cmd tea.Cmd
	if m.screen == screenLogin {
		m.emailInput, cmd = m.emailInput.Update(msg)
	} else {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	if m.screen == screenLogin {
		return m.viewLogin()
	}
	if m.inputMode != inputNone {
		return m.viewInput()
	}
	if m.screen == screenTokenShown {
		return m.viewTokenShown()
	}

	var b string
	b += titleStyle.Render("pipepush") + "  " + breadcrumbStyle.Render(m.breadcrumb()) + "\n\n"
	b += m.list.View() + "\n"
	if m.status != "" {
		b += successStyle.Render(m.status) + "\n"
	}
	if m.err != nil {
		b += errorStyle.Render("error: "+m.err.Error()) + "\n"
	}
	b += helpStyle.Render(m.helpLine())
	return docStyle.Render(b)
}

func (m *model) setList(title string, items []list.Item) {
	m.list.Title = title
	m.list.SetItems(items)
	m.list.ResetSelected()
	m.err = nil
}

func (m *model) setRuns(runs []*models.Run) {
	items := make([]list.Item, 0, len(runs))
	for _, r := range runs {
		items = append(items, m.makeRunItem(r))
	}
	m.setList("Runs · "+m.curPipeName, items)
}

func (m model) makeRunItem(r *models.Run) runItem {
	var p models.RunPayload
	if plain, err := crypto.DecryptString(m.priv, r.EncryptedPayload); err == nil {
		_ = json.Unmarshal([]byte(plain), &p)
	}
	return runItem{status: r.Status, payload: p, when: r.ReceivedAt.Format("2006-01-02 15:04")}
}

func (m *model) prependRun(e models.SSEEvent) {
	var p models.RunPayload
	if plain, err := crypto.DecryptString(m.priv, e.EncryptedPayload); err == nil {
		_ = json.Unmarshal([]byte(plain), &p)
	}
	item := runItem{status: p.Status, payload: p, when: "just now"}
	items := append([]list.Item{item}, m.list.Items()...)
	m.list.SetItems(items)
}

func (m model) reloadCurrent() tea.Cmd {
	switch m.screen {
	case screenProjects:
		return loadProjectsCmd(m.api)
	case screenPipelines:
		return loadPipelinesCmd(m.api, m.curProjectID)
	case screenRuns:
		return loadRunsCmd(m.api, m.curPipelineID)
	case screenTokens:
		return loadTokensCmd(m.api, m.curProjectID)
	}
	return nil
}

func (m model) breadcrumb() string {
	switch m.screen {
	case screenPipelines:
		return m.curProjectName
	case screenRuns:
		return m.curProjectName + " / " + m.curPipeName
	case screenTokens:
		return m.curProjectName + " / tokens"
	default:
		return "projects"
	}
}

func (m model) helpLine() string {
	switch m.screen {
	case screenProjects:
		return "enter: pipelines · t: tokens · n: new · d: delete · q: quit"
	case screenPipelines:
		return "enter: runs · n: new · d: delete · esc: back · q: quit"
	case screenRuns:
		return "r: refresh · esc: back · q: quit  (live updates on)"
	case screenTokens:
		return "n: new · r: revoke · esc: back · q: quit"
	default:
		return "q: quit"
	}
}

var _ = lipgloss.NewStyle
var _ = fmt.Sprintf

// prog holds the running program so background goroutines (e.g. the SSE
// listener started after an in-TUI login) can deliver messages.
var prog *tea.Program

// Run starts the TUI program.
func Run(cfg *config.ClientConfig) error {
	m := initialModel(cfg)

	// If already logged in, load keys and jump straight to projects.
	var startCmd tea.Cmd
	if cfg.IsLoggedIn() {
		if err := m.loadKeys(); err == nil {
			m.screen = screenProjects
			startCmd = loadProjectsCmd(m.api)
		}
	}

	prog = tea.NewProgram(m, tea.WithAltScreen())

	// Start SSE listener in the background once logged in.
	if cfg.IsLoggedIn() {
		go listenSSECmd(m.api, prog)
	}

	if startCmd != nil {
		go func() { prog.Send(startCmd()) }()
	}

	_, err := prog.Run()
	return err
}
