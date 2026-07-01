// Command pipepush-mcp is a Model Context Protocol server that exposes pipepush
// as tools, so an MCP client (e.g. Claude Code) can create projects, pipelines
// and CI tokens and query runs without shelling out to the CLI.
//
// It reuses the same on-disk session as the CLI (~/.config/pipepush/config.json,
// written by `pipepush login`), so the one-time interactive login is the only
// step that needs a TTY. All names/payloads are E2E-encrypted client-side via
// internal/session; the server only ever sees ciphertext.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/routing"
	"github.com/Gerry3010/pipepush/internal/session"
)

// version is overridden at build time: -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "pipepush", Version: version}, nil)
	registerTools(server)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "pipepush-mcp:", err)
		os.Exit(1)
	}
}

// text wraps a plain-text tool result.
func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

// load returns an authenticated session or a clear "not logged in" error. It is
// called per request so a login performed after the server starts is picked up.
func load() (*session.Session, error) {
	return session.Load()
}

// resolveProjectID accepts either a project ID or a plaintext project name and
// returns the ID. Names are matched case-insensitively against decrypted names.
func resolveProjectID(ctx context.Context, s *session.Session, ref string) (string, error) {
	projects, err := s.API.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.ID == ref {
			return ref, nil
		}
	}
	for _, p := range projects {
		if strings.EqualFold(s.Decrypt(p.EncryptedName), ref) {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("no project matches %q (by id or name)", ref)
}

// resolvePipelineID accepts a pipeline ID or a plaintext pipeline name within a
// project and returns the ID.
func resolvePipelineID(ctx context.Context, s *session.Session, projectID, ref string) (string, error) {
	pipelines, err := s.API.ListPipelines(ctx, projectID)
	if err != nil {
		return "", err
	}
	for _, p := range pipelines {
		if p.ID == ref {
			return ref, nil
		}
	}
	for _, p := range pipelines {
		if strings.EqualFold(s.Decrypt(p.EncryptedName), ref) {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("no pipeline matches %q in project (by id or name)", ref)
}

// --- tool input types ---

type emptyIn struct{}

type projectCreateIn struct {
	Name        string `json:"name" jsonschema:"name of the project to create"`
	Description string `json:"description,omitempty" jsonschema:"optional project description"`
}

type projectRefIn struct {
	Project string `json:"project" jsonschema:"project ID or plaintext project name"`
}

type pipelineCreateIn struct {
	Name    string `json:"name" jsonschema:"name of the pipeline to create (also its routing name)"`
	Project string `json:"project" jsonschema:"project ID or plaintext project name"`
}

type tokenCreateIn struct {
	Name     string `json:"name" jsonschema:"a label for the token, e.g. 'GitHub (my-repo)'"`
	Project  string `json:"project" jsonschema:"project ID or plaintext project name"`
	Pipeline string `json:"pipeline,omitempty" jsonschema:"optional pipeline ID or name; omit for a project-wide token that routes by pipeline name in each request"`
}

type runsListIn struct {
	Pipeline string `json:"pipeline" jsonschema:"pipeline ID"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max number of runs (default 20)"`
}

type sendIn struct {
	Token    string `json:"token" jsonschema:"the plaintext pp_ token"`
	Status   string `json:"status" jsonschema:"CI status, e.g. success|failure|cancelled|running|skipped (normalized server-side)"`
	Pipeline string `json:"pipeline,omitempty" jsonschema:"pipeline name; required for project-wide tokens"`
	RunID    string `json:"runId,omitempty" jsonschema:"CI run ID/number"`
	Commit   string `json:"commit,omitempty" jsonschema:"commit SHA"`
	Branch   string `json:"branch,omitempty" jsonschema:"branch name"`
	Duration string `json:"duration,omitempty" jsonschema:"run duration, e.g. 3m12s"`
	Message  string `json:"message,omitempty" jsonschema:"free-form message"`
}

func registerTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "pipepush_whoami",
		Description: "Show the logged-in pipepush account (email) and the configured server URL.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyIn) (*mcp.CallToolResult, any, error) {
		sess, err := load()
		if err != nil {
			return nil, nil, err
		}
		return text(fmt.Sprintf("Logged in as %s\nServer: %s", sess.Cfg.Email, sess.Cfg.ServerURL)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pipepush_projects_list",
		Description: "List your pipepush projects with their IDs (names decrypted locally).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyIn) (*mcp.CallToolResult, any, error) {
		sess, err := load()
		if err != nil {
			return nil, nil, err
		}
		projects, err := sess.API.ListProjects(ctx)
		if err != nil {
			return nil, nil, err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%d project(s):\n", len(projects))
		for _, p := range projects {
			fmt.Fprintf(&b, "  %s  %s\n", p.ID, sess.Decrypt(p.EncryptedName))
		}
		return text(b.String()), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pipepush_project_create",
		Description: "Create a pipepush project. The name is encrypted client-side before upload.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in projectCreateIn) (*mcp.CallToolResult, any, error) {
		sess, err := load()
		if err != nil {
			return nil, nil, err
		}
		if in.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		encName, err := sess.Encrypt(in.Name)
		if err != nil {
			return nil, nil, err
		}
		var encDesc string
		if in.Description != "" {
			if encDesc, err = sess.Encrypt(in.Description); err != nil {
				return nil, nil, err
			}
		}
		p, err := sess.API.CreateProject(ctx, encName, encDesc)
		if err != nil {
			return nil, nil, err
		}
		return text(fmt.Sprintf("Created project %q\nID: %s", in.Name, p.ID)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pipepush_pipelines_list",
		Description: "List pipelines in a project (accepts a project ID or name).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in projectRefIn) (*mcp.CallToolResult, any, error) {
		sess, err := load()
		if err != nil {
			return nil, nil, err
		}
		projectID, err := resolveProjectID(ctx, sess, in.Project)
		if err != nil {
			return nil, nil, err
		}
		pipelines, err := sess.API.ListPipelines(ctx, projectID)
		if err != nil {
			return nil, nil, err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%d pipeline(s):\n", len(pipelines))
		for _, p := range pipelines {
			fmt.Fprintf(&b, "  %s  %s\n", p.ID, sess.Decrypt(p.EncryptedName))
		}
		return text(b.String()), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pipepush_pipeline_create",
		Description: "Create a pipeline in a project. The name doubles as the routing key.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in pipelineCreateIn) (*mcp.CallToolResult, any, error) {
		sess, err := load()
		if err != nil {
			return nil, nil, err
		}
		if in.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		projectID, err := resolveProjectID(ctx, sess, in.Project)
		if err != nil {
			return nil, nil, err
		}
		encName, err := sess.Encrypt(in.Name)
		if err != nil {
			return nil, nil, err
		}
		p, err := sess.API.CreatePipeline(ctx, projectID, encName, routing.Key(in.Name))
		if err != nil {
			return nil, nil, err
		}
		return text(fmt.Sprintf("Created pipeline %q\nID: %s", in.Name, p.ID)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "pipepush_token_create",
		Description: "Create a CI/CD notification token. Returns the plaintext pp_ token ONCE " +
			"(store it as the CI secret PIPEPUSH_TOKEN) plus a ready-to-paste setup snippet. " +
			"Omit 'pipeline' for a project-wide token that routes by the pipeline name in each request.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in tokenCreateIn) (*mcp.CallToolResult, any, error) {
		sess, err := load()
		if err != nil {
			return nil, nil, err
		}
		if in.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		projectID, err := resolveProjectID(ctx, sess, in.Project)
		if err != nil {
			return nil, nil, err
		}
		var pipelineID string
		if in.Pipeline != "" {
			if pipelineID, err = resolvePipelineID(ctx, sess, projectID, in.Pipeline); err != nil {
				return nil, nil, err
			}
		}
		encName, err := sess.Encrypt(in.Name)
		if err != nil {
			return nil, nil, err
		}
		resp, err := sess.API.CreateToken(ctx, encName, projectID, pipelineID)
		if err != nil {
			return nil, nil, err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Created token %q\n\n  %s\n\n", in.Name, resp.PlaintextToken)
		b.WriteString("⚠  This token is shown only once and stored only as a hash. Store it as the CI secret PIPEPUSH_TOKEN.\n")
		b.WriteString(session.TokenUsage(sess.Cfg.ServerURL, pipelineID != ""))
		return text(b.String()), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pipepush_runs_list",
		Description: "List recent runs for a pipeline, decrypted locally (status, branch, commit, message).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in runsListIn) (*mcp.CallToolResult, any, error) {
		sess, err := load()
		if err != nil {
			return nil, nil, err
		}
		if in.Pipeline == "" {
			return nil, nil, fmt.Errorf("pipeline is required")
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}
		runs, err := sess.API.ListRuns(ctx, in.Pipeline, limit)
		if err != nil {
			return nil, nil, err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%d run(s):\n", len(runs))
		for _, r := range runs {
			var p models.RunPayload
			if plain, derr := sess.DecryptPayload(r.EncryptedPayload); derr == nil {
				_ = json.Unmarshal([]byte(plain), &p)
			}
			commit := p.Commit
			if len(commit) > 8 {
				commit = commit[:8]
			}
			fmt.Fprintf(&b, "  %s  %s  %s  %s  %s\n",
				r.Status, p.Branch, commit, p.Message, r.ReceivedAt.Format("01-02 15:04"))
		}
		return text(b.String()), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pipepush_send",
		Description: "Send a pipeline status using a token (no login needed for the send itself). Useful for end-to-end verification.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sendIn) (*mcp.CallToolResult, any, error) {
		sess, err := load()
		if err != nil {
			return nil, nil, err
		}
		if in.Token == "" || in.Status == "" {
			return nil, nil, fmt.Errorf("token and status are required")
		}
		err = sess.API.Send(ctx, models.WebhookRequest{
			Token:    in.Token,
			Status:   session.NormalizeStatus(in.Status),
			Pipeline: in.Pipeline,
			RunID:    in.RunID,
			Commit:   in.Commit,
			Branch:   in.Branch,
			Duration: in.Duration,
			Message:  in.Message,
		})
		if err != nil {
			return nil, nil, err
		}
		return text(fmt.Sprintf("✓ Sent %s notification", session.NormalizeStatus(in.Status))), nil, nil
	})
}
