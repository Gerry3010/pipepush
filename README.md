# ⚡ pipepush

**End-to-end encrypted CI/CD pipeline notifications — provider-agnostic, self-hostable, open source.**

Drop one step into any pipeline (GitHub Actions, GitLab CI, Jenkins, Drone, …) and
get a push notification the moment it succeeds or fails — on the CLI, in the
browser, or as a Web Push notification on your phone. Pipeline details are
**encrypted on your device** with a key only you hold; the server only ever
stores ciphertext.

```
CI/CD step ──POST /api/webhook──▶ pipepush server ──┬─▶ SSE  ──▶ CLI / Web (live)
   (token)                        (encrypts with     └─▶ Web Push ──▶ browser / phone
                                   your public key)
```

---

## Why pipepush?

- **Provider-agnostic** — it's just an HTTP webhook + a tiny `send` binary. Works anywhere.
- **End-to-end encrypted** — X25519 + XChaCha20-Poly1305. The server can't read your
  pipeline names, branches, commits, or messages. [How it works](docs/ARCHITECTURE.md).
- **Self-hosted** — one `docker compose up`. Your data stays on your VPS.
- **Three clients** — a Bubble Tea **TUI**, a **CLI** for scripting/CI, and a **React PWA**
  with browser push.
- **Agent-friendly** — ships with a [Claude Code skill](.claude/skills/pipepush-setup/SKILL.md)
  so an AI agent can wire pipepush into your repo for you. See [below](#-for-ai-agents-claude-etc).

---

## Quick start (self-hosting)

```bash
git clone https://github.com/Gerry3010/pipepush.git
cd pipepush
cp .env.example .env

# Generate the secrets the .env asks for:
openssl rand -base64 32                         # -> JWT_SECRET
docker compose run --rm pipepush pipepush-server -gen-vapid-keys   # -> VAPID keys
# (paste all of these into .env)

docker compose up -d
```

The app is now at `http://localhost:8080` (web UI + API). Put it behind a TLS
reverse proxy (Caddy/Traefik/nginx) for production — Web Push **requires HTTPS**.

## Quick start (CLI)

```bash
go install github.com/Gerry3010/pipepush/cmd/pipepush@latest

pipepush server set https://pipepush.example.com
pipepush login --register            # create an account (generates your E2E keys)
pipepush                             # launch the interactive TUI
```

Or scripted:

```bash
pipepush projects create "My App"
pipepush pipelines create "Deploy" --project <project-id>
pipepush tokens create "GitHub Actions" --project <project-id> --pipeline <pipeline-id>
# -> copy the pp_… token (shown once)
```

## Wire it into CI/CD

The notification step needs nothing but the token. With the `pipepush` binary:

```yaml
# GitHub Actions
- name: Notify pipepush
  if: always()
  run: |
    curl -sL https://github.com/Gerry3010/pipepush/releases/latest/download/pipepush-linux-amd64 \
      -o /usr/local/bin/pipepush && chmod +x /usr/local/bin/pipepush
    pipepush send --token "$PIPEPUSH_TOKEN" --status "${{ job.status }}" \
      --pipeline "${{ github.workflow }}" --branch "${{ github.ref_name }}" \
      --commit "${{ github.sha }}" --run-id "${{ github.run_number }}"
  env:
    PIPEPUSH_TOKEN: ${{ secrets.PIPEPUSH_TOKEN }}
    PIPEPUSH_SERVER: ${{ vars.PIPEPUSH_SERVER }}
```

Or with plain `curl` (no binary):

```bash
curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" -H "Content-Type: application/json" \
  -d "{\"token\":\"$PIPEPUSH_TOKEN\",\"status\":\"success\",\"branch\":\"main\"}"
```

More examples: [GitHub Actions](docs/GITHUB_ACTIONS.md) · [GitLab CI](docs/GITLAB_CI.md)

`--status` accepts CI-native values too (`passed`, `failed`, `aborted`, …) — they're
normalized automatically.

---

## 🤖 For AI agents (Claude etc.)

pipepush ships with a **Claude Code skill** that teaches an agent how to set
itself up as a notification step in your repository:

➡️ **[.claude/skills/pipepush-setup/SKILL.md](.claude/skills/pipepush-setup/SKILL.md)**

If you use [Claude Code](https://claude.com/claude-code), the skill is picked up
automatically when this repo (or the skill) is on your path. Just ask:

> "Use the pipepush-setup skill to add pipeline notifications to this repo."

The agent will create a project/pipeline/token via the CLI and add the correct
notification step to your CI config. A provider-neutral walkthrough is in
[docs/AGENT_SETUP.md](docs/AGENT_SETUP.md).

---

## Components

| Path                    | What                                                       |
| ----------------------- | ---------------------------------------------------------- |
| `cmd/pipepush`          | CLI + TUI (Bubble Tea)                                     |
| `cmd/pipepush-server`   | HTTP server (chi) + webhook receiver + Web Push dispatcher |
| `internal/crypto`       | X25519 ECIES + Argon2id key wrapping (Go)                  |
| `web/`                  | React PWA, wire-compatible crypto via `@noble/*`           |
| `migrations/`           | PostgreSQL schema (auto-applied on server start)           |

## Development

```bash
# Backend + DB
docker run -d --name pp-pg -e POSTGRES_DB=pipepush -e POSTGRES_USER=pipepush \
  -e POSTGRES_PASSWORD=secret -p 5433:5432 postgres:17-alpine
export DATABASE_URL='postgres://pipepush:secret@localhost:5433/pipepush?sslmode=disable'
export JWT_SECRET=dev-secret
go run ./cmd/pipepush-server          # migrations run automatically

# Web (dev server proxies /api -> :8080)
cd web && npm install && npm run dev

# Tests
go test ./...                                            # unit (incl. crypto)
PIPEPUSH_E2E_URL=http://localhost:8088 go test ./internal/client -run EndToEnd
PIPEPUSH_INTEROP=1 go test ./internal/crypto -run Interop   # Go<->browser crypto
```

## Security

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the threat model. In short:
the server stores your public key and ciphertext only; your private key is
wrapped with an Argon2id key derived from your password and is decrypted only on
your device. CLI sessions cache the decrypted key in `~/.config/pipepush/`
(mode 0600) — treat that file like an SSH key.

## License

MIT © Gerald Hofbauer
