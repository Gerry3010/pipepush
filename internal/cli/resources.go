package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/routing"
	"github.com/Gerry3010/pipepush/internal/session"
)

// --- projects ---

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "projects", Short: "Manage projects"}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List your projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			projects, err := s.API.ListProjects(context.Background())
			if err != nil {
				return err
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNAME\tCREATED")
			for _, p := range projects {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", p.ID, s.Decrypt(p.EncryptedName), p.CreatedAt.Format("2006-01-02"))
			}
			return tw.Flush()
		},
	})

	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			desc, _ := cmd.Flags().GetString("description")
			encName, err := s.Encrypt(args[0])
			if err != nil {
				return err
			}
			var encDesc string
			if desc != "" {
				if encDesc, err = s.Encrypt(desc); err != nil {
					return err
				}
			}
			p, err := s.API.CreateProject(context.Background(), encName, encDesc)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Created project %q\n  ID: %s\n", args[0], p.ID)
			return nil
		},
	}
	createCmd.Flags().String("description", "", "Project description")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			if err := s.API.DeleteProject(context.Background(), args[0]); err != nil {
				return err
			}
			fmt.Println("✓ Deleted")
			return nil
		},
	})

	return cmd
}

// --- pipelines ---

func newPipelinesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "pipelines", Short: "Manage pipelines"}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pipelines in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			projectID, _ := cmd.Flags().GetString("project")
			if projectID == "" {
				return fmt.Errorf("--project is required")
			}
			pipelines, err := s.API.ListPipelines(context.Background(), projectID)
			if err != nil {
				return err
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNAME\tCREATED")
			for _, p := range pipelines {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", p.ID, s.Decrypt(p.EncryptedName), p.CreatedAt.Format("2006-01-02"))
			}
			return tw.Flush()
		},
	}
	listCmd.Flags().String("project", "", "Project ID (required)")
	cmd.AddCommand(listCmd)

	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			projectID, _ := cmd.Flags().GetString("project")
			if projectID == "" {
				return fmt.Errorf("--project is required")
			}
			encName, err := s.Encrypt(args[0])
			if err != nil {
				return err
			}
			p, err := s.API.CreatePipeline(context.Background(), projectID, encName, routing.Key(args[0]))
			if err != nil {
				return err
			}
			fmt.Printf("✓ Created pipeline %q\n  ID: %s\n", args[0], p.ID)
			return nil
		},
	}
	createCmd.Flags().String("project", "", "Project ID (required)")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			if err := s.API.DeletePipeline(context.Background(), args[0]); err != nil {
				return err
			}
			fmt.Println("✓ Deleted")
			return nil
		},
	})

	return cmd
}

// --- tokens ---

func newTokensCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tokens", Short: "Manage notification tokens"}

	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a notification token for CI/CD",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			projectID, _ := cmd.Flags().GetString("project")
			pipelineID, _ := cmd.Flags().GetString("pipeline")
			if projectID == "" {
				return fmt.Errorf("--project is required")
			}
			encName, err := s.Encrypt(args[0])
			if err != nil {
				return err
			}
			resp, err := s.API.CreateToken(context.Background(), encName, projectID, pipelineID)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Created token %q\n\n", args[0])
			fmt.Printf("  %s\n\n", resp.PlaintextToken)
			fmt.Println("⚠  Copy it now — it is shown only once and stored only as a hash.")
			fmt.Print(session.TokenUsage(s.Cfg.ServerURL, pipelineID != ""))
			return nil
		},
	}
	createCmd.Flags().String("project", "", "Project ID (required)")
	createCmd.Flags().String("pipeline", "", "Pipeline ID (optional; omit for a project-wide token that routes by pipeline name)")
	cmd.AddCommand(createCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tokens in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			projectID, _ := cmd.Flags().GetString("project")
			if projectID == "" {
				return fmt.Errorf("--project is required")
			}
			tokens, err := s.API.ListTokens(context.Background(), projectID)
			if err != nil {
				return err
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNAME\tPIPELINE\tACTIVE\tLAST USED")
			for _, t := range tokens {
				lastUsed := "never"
				if t.LastUsedAt != nil {
					lastUsed = t.LastUsedAt.Format("2006-01-02 15:04")
				}
				pipe := t.PipelineID
				if pipe == "" {
					pipe = "(project)"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%t\t%s\n", t.ID, s.Decrypt(t.EncryptedName), pipe, t.Active, lastUsed)
			}
			return tw.Flush()
		},
	}
	listCmd.Flags().String("project", "", "Project ID (required)")
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke a token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			if err := s.API.RevokeToken(context.Background(), args[0]); err != nil {
				return err
			}
			fmt.Println("✓ Revoked")
			return nil
		},
	})

	return cmd
}

// --- runs ---

func newRunsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "runs", Short: "View pipeline runs"}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recent runs for a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := loadSession()
			if err != nil {
				return err
			}
			pipelineID, _ := cmd.Flags().GetString("pipeline")
			limit, _ := cmd.Flags().GetInt("limit")
			if pipelineID == "" {
				return fmt.Errorf("--pipeline is required")
			}
			runs, err := s.API.ListRuns(context.Background(), pipelineID, limit)
			if err != nil {
				return err
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "STATUS\tBRANCH\tCOMMIT\tMESSAGE\tRECEIVED")
			for _, r := range runs {
				var p models.RunPayload
				if plain, derr := s.DecryptPayload(r.EncryptedPayload); derr == nil {
					_ = json.Unmarshal([]byte(plain), &p)
				}
				commit := p.Commit
				if len(commit) > 8 {
					commit = commit[:8]
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
					statusIcon(r.Status)+r.Status, p.Branch, commit, p.Message, r.ReceivedAt.Format("01-02 15:04"))
			}
			return tw.Flush()
		},
	}
	listCmd.Flags().String("pipeline", "", "Pipeline ID (required)")
	listCmd.Flags().Int("limit", 20, "Max number of runs")
	cmd.AddCommand(listCmd)

	return cmd
}

func statusIcon(status string) string {
	switch status {
	case "success":
		return "✓ "
	case "failure":
		return "✗ "
	case "cancelled":
		return "⊘ "
	case "running":
		return "● "
	case "skipped":
		return "○ "
	default:
		return "  "
	}
}
