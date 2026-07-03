---
name: pipepush-setup
description: >-
  Set up pipepush CI/CD pipeline notifications in a repository. Use when the user
  wants to add build/deploy notifications, get notified when pipelines finish or
  fail, or wire pipepush into GitHub Actions, GitLab CI, or any CI/CD provider.
---

# pipepush setup

This skill wires [pipepush](https://github.com/Gerry3010/pipepush) into a repo so
the user gets an end-to-end encrypted push notification when a pipeline finishes.

## Prerequisites — check first

1. Is the `pipepush` CLI installed? Run `pipepush --version`.
   - If not: `go install github.com/Gerry3010/pipepush/cmd/pipepush@latest`
     (or download the release binary for the platform).
2. Is a server configured and logged in? Run `pipepush whoami`.
   - If "Not logged in": ask the user for their server URL, then run
     `pipepush server set <url>` and tell them to run `pipepush login` themselves
     (it prompts for a password — **do not** ask for or handle their password).
   - Never attempt to type the password on their behalf; the prompt needs a TTY.

## Steps

Once `pipepush whoami` succeeds:

1. **Create or pick a project** (one per app/repo is typical):
   ```bash
   pipepush projects list
   pipepush projects create "<repo name>"     # if none fits
   ```
   Capture the project ID from the output.

2. **Create a pipeline** (one per workflow, e.g. "CI", "Deploy"):
   ```bash
   pipepush pipelines create "<workflow name>" --project <project-id>
   ```
   Capture the pipeline ID.

3. **Create a token.** Pick the scope based on how the user wants to wire it up
   — `tokens create` prints a ready-to-paste GitHub Actions snippet tailored to
   the scope, so relay that output to the user.

   - **Pipeline-bound** (one token per workflow; runs always go to this pipeline):
     ```bash
     pipepush tokens create "<provider> (<repo>)" --project <project-id> --pipeline <pipeline-id>
     ```
   - **Project-wide** (one token for the whole repo; each run is routed — and the
     pipeline auto-created — by the `pipeline` name in the request, so the
     workflow name decides the pipeline). Omit `--pipeline`:
     ```bash
     pipepush tokens create "<repo>" --project <project-id>
     ```

   The plaintext token (`pp_…`) is printed **once**. Do NOT commit it. Instead:
   - Tell the user to store it as a CI secret named `PIPEPUSH_TOKEN`.
   - For GitHub: `gh secret set PIPEPUSH_TOKEN` (the user pastes the value), and
     `gh variable set PIPEPUSH_SERVER --body "<server url>"`.

4. **Detect the CI provider** by inspecting the repo, then add a notification
   step. Always run it on both success and failure. For a project-wide token the
   request **must** include a non-empty `pipeline` name (e.g. `github.workflow`).

   **GitHub Actions** — add to the relevant job in `.github/workflows/*.yml`:
   ```yaml
   - name: Notify pipepush
     if: always()
     run: |
       curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \
         -H "Content-Type: application/json" \
         -d "{\"token\":\"$PIPEPUSH_TOKEN\",\"status\":\"${{ job.status }}\",\"pipeline\":\"${{ github.workflow }}\",\"branch\":\"${{ github.ref_name }}\",\"commit\":\"${{ github.sha }}\",\"runId\":\"${{ github.run_number }}\"}"
     env:
       PIPEPUSH_TOKEN: ${{ secrets.PIPEPUSH_TOKEN }}
       PIPEPUSH_SERVER: ${{ vars.PIPEPUSH_SERVER }}
   ```

   **GitLab CI** — add a job in `.gitlab-ci.yml`. GitLab's `$CI_JOB_STATUS` is
   `success`/`failed`/`canceled`; the webhook only accepts `success`/`failure`/
   `cancelled`, so map it first:
   ```yaml
   notify_pipepush:
     stage: .post
     when: always
     image: curlimages/curl:latest
     script:
       - |
         case "$CI_JOB_STATUS" in
           success) STATUS=success ;;
           canceled) STATUS=cancelled ;;
           *) STATUS=failure ;;
         esac
         curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \
           -H "Content-Type: application/json" \
           -d "{\"token\":\"$PIPEPUSH_TOKEN\",\"status\":\"$STATUS\",\"pipeline\":\"$CI_PIPELINE_NAME\",\"branch\":\"$CI_COMMIT_REF_NAME\",\"commit\":\"$CI_COMMIT_SHA\"}"
   ```
   (Set `PIPEPUSH_TOKEN` and `PIPEPUSH_SERVER` as masked CI/CD variables.)

   **Bitbucket Pipelines** — Bitbucket has no job-status variable, so notify from
   `after-script` using `$BITBUCKET_EXIT_CODE`. Add to `bitbucket-pipelines.yml`:
   ```yaml
   pipelines:
     default:
       - step:
           name: Build
           script:
             - echo "your build/test/deploy steps"
           after-script:
             - |
               if [ "$BITBUCKET_EXIT_CODE" = "0" ]; then STATUS=success; else STATUS=failure; fi
               curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \
                 -H "Content-Type: application/json" \
                 -d "{\"token\":\"$PIPEPUSH_TOKEN\",\"status\":\"$STATUS\",\"pipeline\":\"$BITBUCKET_REPO_SLUG\",\"branch\":\"$BITBUCKET_BRANCH\",\"commit\":\"$BITBUCKET_COMMIT\",\"runId\":\"$BITBUCKET_BUILD_NUMBER\"}"
   ```
   (Set `PIPEPUSH_TOKEN` and `PIPEPUSH_SERVER` as Secured repository variables.)

   **Codemagic** (Flutter/iOS/Android) — no per-step status var exists in the
   build steps, so notify from `publishing:` `scripts:` (they run regardless of
   outcome) using `$CM_BUILD_STEP_STATUS`. Add `PIPEPUSH_TOKEN` (Secure) +
   `PIPEPUSH_SERVER` as env vars, then to each workflow:
   ```yaml
   publishing:
     scripts:
       - name: Notify pipepush
         script: |
           if [ "$CM_BUILD_STEP_STATUS" = "success" ]; then STATUS=success; else STATUS=failure; fi
           curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \
             -H "Content-Type: application/json" \
             -d "{\"token\":\"$PIPEPUSH_TOKEN\",\"status\":\"$STATUS\",\"pipeline\":\"$CM_WORKFLOW_ID\",\"branch\":\"${CM_BRANCH:-$CM_TAG}\",\"commit\":\"$CM_COMMIT\",\"runId\":\"$BUILD_NUMBER\"}"
   ```
   (Full notes in `docs/CODEMAGIC.md`. Codemagic = Apple/Flutter CI — verify on a real build.)

   **Other providers** — the contract is just `POST $PIPEPUSH_SERVER/api/webhook`
   with JSON `{ token, status, pipeline?, branch?, commit?, runId?, duration?, message?, logs? }`.
   `status` must be one of `success`, `failure`, `cancelled`, `running`, `skipped`
   (the webhook does **not** normalize — only the `pipepush send` CLI and the
   `pipepush_send` MCP tool map CI-native values like `passed`/`failed`/`aborted`
   before sending; raw curl must send a valid value). `pipeline` is optional for a
   pipeline-bound token but **required** for a project-wide token (it selects/creates
   the pipeline by name). `logs` is optional build output — see below.

   **Attaching build logs (optional)** — logs are stored end-to-end encrypted (like
   every other run detail) and shown in the run detail view; the server keeps the
   last 16 KB. Nothing captures logs automatically: pipe your build step's output to
   a file first (e.g. `your-build 2>&1 | tee build.log`), then send the tail. Two ways:

   - **Raw curl + jq** (jq escapes multi-line log text safely; preinstalled on GitHub
     runners, `apk add jq` on the GitLab curl image):
     ```bash
     LOGS=$(tail -c 16000 build.log 2>/dev/null || true)
     jq -nc --arg token "$PIPEPUSH_TOKEN" --arg status "$STATUS" --arg pipeline "CI" \
           --arg branch "$BRANCH" --arg commit "$COMMIT" --arg logs "$LOGS" \
           '{token:$token,status:$status,pipeline:$pipeline,branch:$branch,commit:$commit,logs:$logs}' \
       | curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" -H "Content-Type: application/json" -d @-
     ```
   - **The `pipepush send` binary** (no jq needed — Go escapes + truncates for you):
     ```bash
     pipepush send --token "$PIPEPUSH_TOKEN" --status "$STATUS" --pipeline CI \
       --branch "$BRANCH" --commit "$COMMIT" --logs-file build.log
     ```

5. **Verify** end-to-end without waiting for a real build:
   ```bash
   pipepush send --token <pp_token> --status success --branch test --message "setup check"
   pipepush runs list --pipeline <pipeline-id>   # the test run should appear, decrypted
   ```
   Then tell the user to delete the test run expectation and they're done.

## Important rules

- The plaintext token is shown once and stored only as a hash — never echo it
  into a file the user might commit; route it to a secret store.
- Never handle the user's password; `pipepush login` is theirs to run.
- Prefer the `curl` form in CI (no binary download); use the `pipepush send`
  binary only if the user wants status normalization or a smaller command.
- Keep `if: always()` / `when: always` so failures (the important case) notify.
