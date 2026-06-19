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
      curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \\
        -H "Content-Type: application/json" \\
        -d "{\\"token\\":\\"$PIPEPUSH_TOKEN\\",\\"status\\":\\"$CI_JOB_STATUS\\",\\"pipeline\\":\\"$CI_PIPELINE_NAME\\",\\"branch\\":\\"$CI_COMMIT_REF_NAME\\",\\"commit\\":\\"$CI_COMMIT_SHA\\"}"`;
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
              -d "{\\"token\\":\\"$PIPEPUSH_TOKEN\\",\\"status\\":\\"$STATUS\\",\\"pipeline\\":\\"$BITBUCKET_REPO_SLUG\\",\\"branch\\":\\"$BITBUCKET_BRANCH\\",\\"commit\\":\\"$BITBUCKET_COMMIT\\",\\"runId\\":\\"$BITBUCKET_BUILD_NUMBER\\"}"`;
}

function otherSnippet(token: string, server: string): string {
  return `curl -sf -X POST "${server}/api/webhook" \\
  -H "Content-Type: application/json" \\
  -d '{"token":"${token}","status":"success","pipeline":"CI","branch":"main","commit":"abc1234"}'

# status accepts CI-native values (passed, failed, aborted, …) — normalized server-side.
# Fields: token (required), status (required), pipeline, branch, commit, runId, duration, message.`;
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
          <button className="copy-snippet link-btn" onClick={() => copy(snippet, "snippet")}>
            {copied === "snippet" ? "copied ✓" : "copy"}
          </button>
          <pre className="snippet">{snippet}</pre>
        </div>
      </div>
    </div>
  );
}
