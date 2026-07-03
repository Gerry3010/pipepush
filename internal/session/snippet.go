package session

import (
	"fmt"
	"strings"
)

// TokenUsage returns copy-paste CI setup for a freshly created token. The
// guidance differs by token scope: a pipeline-bound token routes by itself
// (the pipeline name is informational), while a project-scoped token routes —
// and auto-creates — a pipeline by the "pipeline" name in each request.
//
// It returns the text as a string so both the CLI (which prints it) and the MCP
// server (which returns it in a tool result) emit the identical snippet.
func TokenUsage(serverURL string, pipelineScoped bool) string {
	server := serverURL
	if server == "" {
		server = "https://your-pipepush-server"
	}

	var b strings.Builder
	b.WriteString("\n── GitHub Actions setup ──────────────────────────────\n")
	if pipelineScoped {
		b.WriteString("Scope: pipeline-bound — runs go to this one pipeline.\n")
		b.WriteString("       The \"pipeline\" field below is shown in notifications only.\n")
	} else {
		b.WriteString("Scope: project-wide — one token for every workflow in the repo.\n")
		b.WriteString("       The \"pipeline\" field routes each run to a pipeline by name\n")
		b.WriteString("       (created automatically on first use), so it is required.\n")
	}
	b.WriteString("\n1. Store the secret and server URL (run once):\n\n")
	b.WriteString("   gh secret set PIPEPUSH_TOKEN          # paste the pp_… token above\n")
	fmt.Fprintf(&b, "   gh variable set PIPEPUSH_SERVER --body %q\n", server)
	b.WriteString("\n2. Add this step to your workflow job (notifies on success AND failure):\n\n")
	b.WriteString("   - name: Notify pipepush\n")
	b.WriteString("     if: always()\n")
	b.WriteString("     run: |\n")
	b.WriteString("       curl -sf -X POST \"$PIPEPUSH_SERVER/api/webhook\" \\\n")
	b.WriteString("         -H \"Content-Type: application/json\" \\\n")
	b.WriteString("         -d \"{\\\"token\\\":\\\"$PIPEPUSH_TOKEN\\\",\\\"status\\\":\\\"${{ job.status }}\\\",\\\"pipeline\\\":\\\"${{ github.workflow }}\\\",\\\"branch\\\":\\\"${{ github.ref_name }}\\\",\\\"commit\\\":\\\"${{ github.sha }}\\\",\\\"runId\\\":\\\"${{ github.run_number }}\\\"}\"\n")
	b.WriteString("     env:\n")
	b.WriteString("       PIPEPUSH_TOKEN: ${{ secrets.PIPEPUSH_TOKEN }}\n")
	b.WriteString("       PIPEPUSH_SERVER: ${{ vars.PIPEPUSH_SERVER }}\n\n")
	b.WriteString("Optional: also attach build logs (stored end-to-end encrypted, shown in the\n")
	b.WriteString("run detail view). Pipe your build step's output to a file, e.g.\n")
	b.WriteString("  run: your-build-command 2>&1 | tee build.log\n")
	b.WriteString("then build the JSON with jq (preinstalled on GitHub runners) so log text is\n")
	b.WriteString("escaped safely, and send the last 16 KB:\n\n")
	b.WriteString("   - name: Notify pipepush\n")
	b.WriteString("     if: always()\n")
	b.WriteString("     run: |\n")
	b.WriteString("       LOGS=$(tail -c 16000 build.log 2>/dev/null || true)\n")
	b.WriteString("       jq -nc --arg token \"$PIPEPUSH_TOKEN\" --arg status \"${{ job.status }}\" \\\n")
	b.WriteString("         --arg pipeline \"${{ github.workflow }}\" --arg branch \"${{ github.ref_name }}\" \\\n")
	b.WriteString("         --arg commit \"${{ github.sha }}\" --arg runId \"${{ github.run_number }}\" --arg logs \"$LOGS\" \\\n")
	b.WriteString("         '{token:$token,status:$status,pipeline:$pipeline,branch:$branch,commit:$commit,runId:$runId,logs:$logs}' \\\n")
	b.WriteString("         | curl -sf -X POST \"$PIPEPUSH_SERVER/api/webhook\" -H \"Content-Type: application/json\" -d @-\n")
	b.WriteString("     env:\n")
	b.WriteString("       PIPEPUSH_TOKEN: ${{ secrets.PIPEPUSH_TOKEN }}\n")
	b.WriteString("       PIPEPUSH_SERVER: ${{ vars.PIPEPUSH_SERVER }}\n\n")
	if pipelineScoped {
		b.WriteString("Tip: for a project-wide token (one secret for all workflows), create a\n")
		b.WriteString("     token without --pipeline.\n")
	} else {
		b.WriteString("Tip: ${{ github.workflow }} becomes the pipeline name — give each\n")
		b.WriteString("     workflow a distinct name to keep runs in separate pipelines.\n")
	}
	return b.String()
}
