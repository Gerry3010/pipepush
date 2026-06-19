package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Gerry3010/pipepush/internal/client"
	"github.com/Gerry3010/pipepush/internal/crypto"
	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/routing"
)

// TestEndToEnd exercises the full flow against a running server.
// Enable with: PIPEPUSH_E2E_URL=http://localhost:8088 go test ./internal/client/ -run E2E -v
func TestEndToEnd(t *testing.T) {
	baseURL := os.Getenv("PIPEPUSH_E2E_URL")
	if baseURL == "" {
		t.Skip("set PIPEPUSH_E2E_URL to run the end-to-end test")
	}

	ctx := context.Background()
	email := fmt.Sprintf("e2e-%d@example.com", time.Now().UnixNano())
	password := "test-password-1234"

	// 1. Register: generate keypair, encrypt private key with password.
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	encPriv, salt, err := crypto.EncryptPrivateKey(kp.PrivateKey.Bytes(), password)
	if err != nil {
		t.Fatal(err)
	}

	api := client.New(baseURL, "")
	resp, err := api.Register(ctx, models.RegisterRequest{
		Email:               email,
		Password:            password,
		PublicKey:           crypto.PublicKeyToBase64(kp.PublicKey),
		EncryptedPrivateKey: encPriv,
		KDFSalt:             salt,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	t.Logf("registered %s", email)

	// 2. Verify we can unlock the private key returned by the server.
	privBytes, err := crypto.DecryptPrivateKey(resp.EncryptedPrivateKey, resp.KDFSalt, password)
	if err != nil {
		t.Fatalf("unlock private key: %v", err)
	}
	priv, err := crypto.PrivateKeyFromBytes(privBytes)
	if err != nil {
		t.Fatal(err)
	}

	authed := client.New(baseURL, resp.JWT)

	// 3. Create a project (name encrypted client-side).
	encName, _ := crypto.EncryptString(kp.PublicKey, "My Test Project")
	proj, err := authed.CreateProject(ctx, encName, "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	t.Logf("created project %s", proj.ID)

	// 4. List projects and verify decryption.
	projects, err := authed.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	gotName, err := crypto.DecryptString(priv, projects[0].EncryptedName)
	if err != nil || gotName != "My Test Project" {
		t.Fatalf("project name decrypt failed: %q err=%v", gotName, err)
	}

	// 5. Create a pipeline.
	encPipeName, _ := crypto.EncryptString(kp.PublicKey, "Deploy to Prod")
	pipe, err := authed.CreatePipeline(ctx, proj.ID, encPipeName, routing.Key("Deploy to Prod"))
	if err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	t.Logf("created pipeline %s", pipe.ID)

	// 6. Create a token bound to the pipeline.
	encTokName, _ := crypto.EncryptString(kp.PublicKey, "GitHub Actions")
	tokResp, err := authed.CreateToken(ctx, encTokName, proj.ID, pipe.ID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if tokResp.PlaintextToken == "" {
		t.Fatal("expected plaintext token")
	}
	t.Logf("created token (len=%d)", len(tokResp.PlaintextToken))

	// 7. Send a webhook (simulating CI/CD) — unauthenticated, token only.
	webhookClient := client.New(baseURL, "")
	err = webhookClient.Send(ctx, models.WebhookRequest{
		Token:    tokResp.PlaintextToken,
		Status:   "success",
		Pipeline: "Deploy to Prod",
		RunID:    "42",
		Commit:   "abc123def456",
		Branch:   "main",
		Duration: "3m12s",
		Message:  "All tests passed",
	})
	if err != nil {
		t.Fatalf("send webhook: %v", err)
	}
	t.Log("webhook sent")

	// 8. List runs and verify the E2E-encrypted payload decrypts correctly.
	runs, err := authed.ListRuns(ctx, pipe.ID, 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Status != "success" {
		t.Errorf("expected status success, got %s", runs[0].Status)
	}

	plain, err := crypto.DecryptString(priv, runs[0].EncryptedPayload)
	if err != nil {
		t.Fatalf("decrypt run payload: %v", err)
	}
	var payload models.RunPayload
	if err := json.Unmarshal([]byte(plain), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Branch != "main" || payload.Commit != "abc123def456" || payload.Message != "All tests passed" {
		t.Errorf("payload mismatch: %+v", payload)
	}
	t.Logf("decrypted run payload: %+v", payload)

	// 9. Invalid token must be rejected.
	err = webhookClient.Send(ctx, models.WebhookRequest{Token: "pp_invalid", Status: "success"})
	if err == nil {
		t.Error("expected invalid token to be rejected")
	}

	t.Log("✓ full end-to-end flow passed")
}
