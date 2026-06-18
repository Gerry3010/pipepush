// Package client is the HTTP client used by the CLI to talk to a pipepush server.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Gerry3010/pipepush/internal/models"
)

type Client struct {
	baseURL string
	jwt     string
	http    *http.Client
}

func New(baseURL, jwt string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		jwt:     jwt,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// BaseURL returns the configured server base URL.
func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.jwt != "" {
		req.Header.Set("Authorization", "Bearer "+c.jwt)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed (is the server reachable at %s?): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp models.ErrorResponse
		respBody, _ := io.ReadAll(resp.Body)
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// --- Auth ---

func (c *Client) Register(ctx context.Context, req models.RegisterRequest) (*models.LoginResponse, error) {
	var out models.LoginResponse
	err := c.do(ctx, http.MethodPost, "/api/auth/register", req, &out)
	return &out, err
}

func (c *Client) Login(ctx context.Context, email, password string) (*models.LoginResponse, error) {
	var out models.LoginResponse
	err := c.do(ctx, http.MethodPost, "/api/auth/login", models.LoginRequest{Email: email, Password: password}, &out)
	return &out, err
}

// --- Projects ---

func (c *Client) ListProjects(ctx context.Context) ([]*models.Project, error) {
	var out []*models.Project
	err := c.do(ctx, http.MethodGet, "/api/projects", nil, &out)
	return out, err
}

func (c *Client) CreateProject(ctx context.Context, encName, encDesc string) (*models.Project, error) {
	var out models.Project
	err := c.do(ctx, http.MethodPost, "/api/projects", models.CreateProjectRequest{EncryptedName: encName, EncryptedDescription: encDesc}, &out)
	return &out, err
}

func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/projects/"+id, nil, nil)
}

// --- Pipelines ---

func (c *Client) ListPipelines(ctx context.Context, projectID string) ([]*models.Pipeline, error) {
	var out []*models.Pipeline
	err := c.do(ctx, http.MethodGet, "/api/projects/"+projectID+"/pipelines", nil, &out)
	return out, err
}

func (c *Client) CreatePipeline(ctx context.Context, projectID, encName string) (*models.Pipeline, error) {
	var out models.Pipeline
	err := c.do(ctx, http.MethodPost, "/api/projects/"+projectID+"/pipelines", models.CreatePipelineRequest{EncryptedName: encName}, &out)
	return &out, err
}

func (c *Client) DeletePipeline(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/pipelines/"+id, nil, nil)
}

// --- Tokens ---

func (c *Client) ListTokens(ctx context.Context, projectID string) ([]*models.NotificationToken, error) {
	var out []*models.NotificationToken
	err := c.do(ctx, http.MethodGet, "/api/projects/"+projectID+"/tokens", nil, &out)
	return out, err
}

func (c *Client) CreateToken(ctx context.Context, encName, projectID, pipelineID string) (*models.CreateTokenResponse, error) {
	var out models.CreateTokenResponse
	err := c.do(ctx, http.MethodPost, "/api/tokens", models.CreateTokenRequest{
		EncryptedName: encName, ProjectID: projectID, PipelineID: pipelineID,
	}, &out)
	return &out, err
}

func (c *Client) RevokeToken(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/tokens/"+id, nil, nil)
}

// --- Runs ---

func (c *Client) ListRuns(ctx context.Context, pipelineID string, limit int) ([]*models.Run, error) {
	var out []*models.Run
	path := fmt.Sprintf("/api/pipelines/%s/runs?limit=%d", pipelineID, limit)
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// --- Webhook (send) ---

func (c *Client) Send(ctx context.Context, req models.WebhookRequest) error {
	return c.do(ctx, http.MethodPost, "/api/webhook", req, nil)
}
