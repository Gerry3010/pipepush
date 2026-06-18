# Agent Setup Guide (Claude & other AI agents)

This is a provider-neutral walkthrough for an AI agent (or a human) to add
pipepush notifications to a repository. If you use **Claude Code**, prefer the
packaged skill at [`.claude/skills/pipepush-setup/SKILL.md`](../.claude/skills/pipepush-setup/SKILL.md)
— it encodes these steps with guardrails.

## 0. Context the agent needs

- The pipepush **server URL** (self-hosted, e.g. `https://pipepush.example.com`).
- The user must be **logged in** on the CLI (`pipepush login`). The agent must
  **not** handle the password — that prompt is the user's to complete.

## 1. Confirm the CLI is ready

```bash
pipepush --version      # install via: go install github.com/Gerry3010/pipepush/cmd/pipepush@latest
pipepush whoami         # must show an email + server; else have the user log in
```

## 2. Create the resource hierarchy

```bash
pipepush projects create "My Repo"                 # → note the project ID
pipepush pipelines create "CI" --project <proj-id> # → note the pipeline ID
pipepush tokens create "GitHub Actions" \
  --project <proj-id> --pipeline <pipe-id>         # → prints pp_… ONCE
```

The hierarchy is: **Project → Pipeline → Runs**, with **Tokens** scoped to a
project (optionally bound to a pipeline). A token bound to a pipeline is what
lets incoming webhooks be filed under that pipeline.

## 3. Store the token as a CI secret (never commit it)

```bash
# GitHub example
gh secret set PIPEPUSH_TOKEN          # paste the pp_… value when prompted
gh variable set PIPEPUSH_SERVER --body "https://pipepush.example.com"
```

## 4. Add the notification step

The only hard requirement is a single HTTP POST, run on **both success and
failure**. See [GITHUB_ACTIONS.md](GITHUB_ACTIONS.md) and [GITLAB_CI.md](GITLAB_CI.md)
for copy-paste snippets. The webhook contract:

```
POST $PIPEPUSH_SERVER/api/webhook
Content-Type: application/json

{
  "token":    "pp_…",          // required
  "status":   "success",        // required; CI-native values like "passed" are normalized
  "pipeline": "Deploy",         // optional, informational
  "branch":   "main",           // optional
  "commit":   "abc123",         // optional
  "runId":    "42",             // optional
  "duration": "3m12s",          // optional
  "message":  "All green"       // optional
}
```

## 5. Verify without a real build

```bash
pipepush send --token <pp_…> --status success --branch test --message "agent setup check"
pipepush runs list --pipeline <pipe-id>   # the run appears, fully decrypted
```

If the run shows up decrypted, the wiring is correct. Done.

## Guardrails for agents

- The plaintext token is shown **once** and stored only as a hash — route it to a
  secret store, never into a committed file.
- Do not ask for, store, or type the user's password.
- Keep the step unconditional on outcome (`if: always()` / `when: always`) so the
  important case — failures — actually notifies.
