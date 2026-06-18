package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Gerry3010/pipepush/internal/client"
	"github.com/Gerry3010/pipepush/internal/config"
	"github.com/Gerry3010/pipepush/internal/models"
)

// normalizeStatus maps common CI status strings to pipepush statuses.
func normalizeStatus(s string) string {
	switch s {
	case "success", "passed", "ok", "succeeded":
		return "success"
	case "failure", "failed", "error", "broken":
		return "failure"
	case "cancelled", "canceled", "aborted":
		return "cancelled"
	case "running", "started", "in_progress", "pending":
		return "running"
	case "skipped":
		return "skipped"
	default:
		return s
	}
}

func newSendCmd() *cobra.Command {
	var token, status, serverFlag, pipeline, runID, commit, branch, duration, message string

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a pipeline status from CI/CD (uses a token, no login required)",
		Long: `Send a pipeline status notification.

This is the command you call from a CI/CD step. It only needs a token, so it
works without logging in. The token is created with 'pipepush tokens create'.

Status is normalized, so CI-native values like "passed"/"failed" work directly
(e.g. GitHub Actions ${{ job.status }}).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				token = os.Getenv("PIPEPUSH_TOKEN")
			}
			if token == "" {
				return fmt.Errorf("--token is required (or set PIPEPUSH_TOKEN)")
			}
			if status == "" {
				return fmt.Errorf("--status is required")
			}

			serverURL := serverFlag
			if serverURL == "" {
				serverURL = os.Getenv("PIPEPUSH_SERVER")
			}
			if serverURL == "" {
				// fall back to stored config
				if cfg, err := config.LoadClientConfig(); err == nil {
					serverURL = cfg.ServerURL
				}
			}
			if serverURL == "" {
				return fmt.Errorf("server URL required: use --server, PIPEPUSH_SERVER, or 'pipepush server set'")
			}

			api := client.New(serverURL, "")
			err := api.Send(context.Background(), models.WebhookRequest{
				Token:    token,
				Status:   normalizeStatus(status),
				Pipeline: pipeline,
				RunID:    runID,
				Commit:   commit,
				Branch:   branch,
				Duration: duration,
				Message:  message,
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Sent %s notification\n", normalizeStatus(status))
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Notification token (or PIPEPUSH_TOKEN env)")
	cmd.Flags().StringVar(&status, "status", "", "Status: success|failure|cancelled|running|skipped (required)")
	cmd.Flags().StringVar(&serverFlag, "server", "", "Server URL (or PIPEPUSH_SERVER env)")
	cmd.Flags().StringVar(&pipeline, "pipeline", "", "Pipeline name (informational)")
	cmd.Flags().StringVar(&runID, "run-id", "", "CI run ID/number")
	cmd.Flags().StringVar(&commit, "commit", "", "Commit SHA")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name")
	cmd.Flags().StringVar(&duration, "duration", "", "Run duration (e.g. 3m12s)")
	cmd.Flags().StringVar(&message, "message", "", "Free-form message")

	return cmd
}
