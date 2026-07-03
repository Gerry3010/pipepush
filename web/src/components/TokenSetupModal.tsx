import { useState } from "react";

type Provider = "github" | "gitlab" | "bitbucket" | "other";

interface Props {
  token: string;
  serverUrl: string;
  pipelineBound: boolean;
  onClose: () => void;
}

const PROVIDERS: { id: Provider; label: string }[] = [
  { id: "github", label: "GitHub Actions" },
  { id: "gitlab", label: "GitLab CI" },
  { id: "bitbucket", label: "Bitbucket" },
  { id: "other", label: "Other (curl)" },
];

function githubSnippet(token: string, server: string): string {
  return `# 1. Store the secret and server URL once:
gh secret set PIPEPUSH_TOKEN --body "${token}"
gh variable set PIPEPUSH_SERVER --body "${server}"

# 2. Add this step to your workflow job (notifies on success AND failure):
      - name: Notify pipepush
        if: always()
        run: |
          curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \\
            -H "Content-Type: application/json" \\
            -d "{\\"token\\":\\"$PIPEPUSH_TOKEN\\",\\"status\\":\\"\${{ job.status }}\\",\\"pipeline\\":\\"\${{ github.workflow }}\\",\\"branch\\":\\"\${{ github.ref_name }}\\",\\"commit\\":\\"\${{ github.sha }}\\",\\"runId\\":\\"\${{ github.run_number }}\\"}"
        env:
          PIPEPUSH_TOKEN: \${{ secrets.PIPEPUSH_TOKEN }}
          PIPEPUSH_SERVER: \${{ vars.PIPEPUSH_SERVER }}

# Optional: also attach build logs (stored end-to-end encrypted, shown in the run
# detail view). Pipe your build to a file — "your-build 2>&1 | tee build.log" —
# then build the JSON with jq (preinstalled on GitHub runners) and send the tail:
      - name: Notify pipepush
        if: always()
        run: |
          LOGS=$(tail -c 16000 build.log 2>/dev/null || true)
          jq -nc --arg token "$PIPEPUSH_TOKEN" --arg status "\${{ job.status }}" \\
            --arg pipeline "\${{ github.workflow }}" --arg branch "\${{ github.ref_name }}" \\
            --arg commit "\${{ github.sha }}" --arg runId "\${{ github.run_number }}" --arg logs "$LOGS" \\
            '{token:$token,status:$status,pipeline:$pipeline,branch:$branch,commit:$commit,runId:$runId,logs:$logs}' \\
            | curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" -H "Content-Type: application/json" -d @-
        env:
          PIPEPUSH_TOKEN: \${{ secrets.PIPEPUSH_TOKEN }}
          PIPEPUSH_SERVER: \${{ vars.PIPEPUSH_SERVER }}`;
}

function gitlabSnippet(token: string, server: string): string {
  return `# Set PIPEPUSH_TOKEN and PIPEPUSH_SERVER as masked CI/CD variables:
#   PIPEPUSH_TOKEN  = ${token}
#   PIPEPUSH_SERVER = ${server}
# Then add this job to .gitlab-ci.yml:
notify_pipepush:
  stage: .post
  when: always
  image: curlimages/curl:latest
  script:
    - |
      # $CI_JOB_STATUS is success/failed/canceled; the webhook wants success/failure/cancelled:
      case "$CI_JOB_STATUS" in
        success) STATUS=success ;;
        canceled) STATUS=cancelled ;;
        *) STATUS=failure ;;
      esac
      curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \\
        -H "Content-Type: application/json" \\
        -d "{\\"token\\":\\"$PIPEPUSH_TOKEN\\",\\"status\\":\\"$STATUS\\",\\"pipeline\\":\\"$CI_PIPELINE_NAME\\",\\"branch\\":\\"$CI_COMMIT_REF_NAME\\",\\"commit\\":\\"$CI_COMMIT_SHA\\"}"

# Optional: also attach build logs (E2E-encrypted, shown in the run detail view).
# The curl image has no jq, so download the pipepush binary — it escapes + caps the
# log for you. Pipe your build to build.log ("your-build 2>&1 | tee build.log") in
# the build job, cache/artifact it into this job, then:
#   pipepush send --token "$PIPEPUSH_TOKEN" --status "$STATUS" \\
#     --pipeline "$CI_PIPELINE_NAME" --branch "$CI_COMMIT_REF_NAME" \\
#     --commit "$CI_COMMIT_SHA" --logs-file build.log`;
}

function bitbucketSnippet(token: string, server: string): string {
  return `# Set PIPEPUSH_TOKEN and PIPEPUSH_SERVER as repository variables (Secured):
#   PIPEPUSH_TOKEN  = ${token}
#   PIPEPUSH_SERVER = ${server}
# Then in bitbucket-pipelines.yml, notify from after-script (runs on success AND failure):
pipelines:
  default:
    - step:
        name: Build
        script:
          - echo "your build/test/deploy steps"
        after-script:
          - |
            if [ "$BITBUCKET_EXIT_CODE" = "0" ]; then STATUS=success; else STATUS=failure; fi
            curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \\
              -H "Content-Type: application/json" \\
              -d "{\\"token\\":\\"$PIPEPUSH_TOKEN\\",\\"status\\":\\"$STATUS\\",\\"pipeline\\":\\"$BITBUCKET_REPO_SLUG\\",\\"branch\\":\\"$BITBUCKET_BRANCH\\",\\"commit\\":\\"$BITBUCKET_COMMIT\\",\\"runId\\":\\"$BITBUCKET_BUILD_NUMBER\\"}"

# Optional: also attach build logs (E2E-encrypted, shown in the run detail view).
# Pipe your build step to a file — "your-build 2>&1 | tee build.log" — then build
# the JSON with jq (available on Bitbucket's default images) and send the tail:
            LOGS=$(tail -c 16000 build.log 2>/dev/null || true)
            jq -nc --arg token "$PIPEPUSH_TOKEN" --arg status "$STATUS" \\
              --arg pipeline "$BITBUCKET_REPO_SLUG" --arg branch "$BITBUCKET_BRANCH" \\
              --arg commit "$BITBUCKET_COMMIT" --arg runId "$BITBUCKET_BUILD_NUMBER" --arg logs "$LOGS" \\
              '{token:$token,status:$status,pipeline:$pipeline,branch:$branch,commit:$commit,runId:$runId,logs:$logs}' \\
              | curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" -H "Content-Type: application/json" -d @-`;
}

function otherSnippet(token: string, server: string): string {
  return `curl -sf -X POST "${server}/api/webhook" \\
  -H "Content-Type: application/json" \\
  -d '{"token":"${token}","status":"success","pipeline":"CI","branch":"main","commit":"abc1234"}'

# status must be one of: success, failure, cancelled, running, skipped (the raw
# webhook does NOT normalize — the pipepush CLI/MCP map passed/failed/aborted for you).
# Fields: token (required), status (required), pipeline, branch, commit, runId, duration, message, logs.

# Optional: attach build logs (stored end-to-end encrypted, shown in the run
# detail view). Build the JSON with jq so arbitrary log text is escaped safely:
LOGS=$(tail -c 16000 build.log 2>/dev/null || true)
jq -nc --arg token "${token}" --arg status "success" --arg pipeline "CI" \\
      --arg branch "main" --arg commit "abc1234" --arg logs "$LOGS" \\
      '{token:$token,status:$status,pipeline:$pipeline,branch:$branch,commit:$commit,logs:$logs}' \\
  | curl -sf -X POST "${server}/api/webhook" -H "Content-Type: application/json" -d @-`;
}

export function TokenSetupModal({ token, serverUrl, pipelineBound, onClose }: Props) {
  const [provider, setProvider] = useState<Provider>("github");
  const [copied, setCopied] = useState<"" | "token" | "snippet">("");

  const builders: Record<Provider, (t: string, s: string) => string> = {
    github: githubSnippet,
    gitlab: gitlabSnippet,
    bitbucket: bitbucketSnippet,
    other: otherSnippet,
  };
  const snippet = builders[provider](token, serverUrl);

  async function copy(text: string, what: "token" | "snippet") {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(what);
      setTimeout(() => setCopied(""), 1500);
    } catch {
      /* clipboard blocked — user can select manually */
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Token created — wire it into your CI</h3>
          <button className="link-btn" onClick={onClose}>
            close
          </button>
        </div>

        <p className="scope-note">
          {pipelineBound ? (
            <>
              <strong>Pipeline-bound token.</strong> Runs always go to this one
              pipeline; the <code>pipeline</code> field below is shown in
              notifications only.
            </>
          ) : (
            <>
              <strong>Project-wide token.</strong> The <code>pipeline</code> field
              routes each run to a pipeline by name (created automatically on
              first use), so it is required.
            </>
          )}
        </p>

        <label>Token — copy it now, it is shown only once</label>
        <div className="token-row">
          <code className="token-value">{token}</code>
          <button className="link-btn" onClick={() => copy(token, "token")}>
            {copied === "token" ? "copied ✓" : "copy"}
          </button>
        </div>

        <div className="tabs">
          {PROVIDERS.map((p) => (
            <button
              key={p.id}
              className={`tab${provider === p.id ? " active" : ""}`}
              onClick={() => setProvider(p.id)}
            >
              {p.label}
            </button>
          ))}
        </div>

        <div className="snippet-wrap">
          <button className="copy-snippet" onClick={() => copy(snippet, "snippet")}>
            {copied === "snippet" ? "copied ✓" : "copy"}
          </button>
          <pre className="snippet">{snippet}</pre>
        </div>
      </div>
    </div>
  );
}
