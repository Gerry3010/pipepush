# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

pipepush is an **end-to-end-encrypted CI/CD notification service**. A CI step POSTs a token + run
status to the server; the server encrypts the run details to the user's public key, stores only
ciphertext, and dispatches the encrypted blob over SSE (live) and Web Push (browser/phone). Plaintext
run details only ever exist on the user's device. It ships three things from one Go module: an HTTP
server, a CLI/TUI client, and a React PWA.

## Commands

Go (server + CLI, module `github.com/Gerry3010/pipepush`):

```bash
go build ./...                  # build everything
go vet ./...                    # what CI lints with
go test ./...                   # unit tests incl. crypto
go test ./internal/crypto -run TestName   # single test
go run ./cmd/pipepush-server    # run server (migrations run automatically on boot)
go run ./cmd/pipepush           # run CLI/TUI
pipepush-server -gen-vapid-keys # generate Web Push VAPID key pair for .env
```

Web (`web/`, Vite + React + TypeScript):

```bash
cd web && npm install
npm run dev      # dev server
npm run build    # tsc -b && vite build  (output in web/dist, served by the Go server via STATIC_DIR)
```

Two tests need extra setup and are the ones most likely to bite — they are exercised in CI:

```bash
# End-to-end (needs a running server + Postgres):
DATABASE_URL=postgres://... JWT_SECRET=x PORT=8088 go run ./cmd/pipepush-server &
PIPEPUSH_E2E_URL=http://localhost:8088 go test ./internal/client -run EndToEnd

# Go <-> browser crypto wire-compatibility (no DB needed):
PIPEPUSH_INTEROP=1 go test ./internal/crypto -run Interop
```

Local Postgres for development:

```bash
docker run -d --name pp-pg -e POSTGRES_DB=pipepush -e POSTGRES_USER=pipepush \
  -e POSTGRES_PASSWORD=secret -p 5433:5432 postgres:17-alpine
export DATABASE_URL='postgres://pipepush:secret@localhost:5433/pipepush?sslmode=disable'
```

Full stack via Docker: `cp .env.example .env`, fill in secrets, `docker compose up -d`.

## Architecture

`docs/ARCHITECTURE.md` is the authoritative reference for the crypto and threat model — read it before
touching anything encryption-related. Key points:

- **Two binaries, one module.** `cmd/pipepush-server` (chi HTTP server, all routes wired in
  `internal/server/router.go`) and `cmd/pipepush` (Cobra CLI + Bubble Tea TUI under `internal/tui`,
  `internal/cli`).
- **E2E encryption (`internal/crypto`, mirrored in `web/src/crypto`).** Every user has an X25519 key
  pair generated client-side. The private key is wrapped with AES-256-GCM under an Argon2id-derived key
  and stored as an opaque blob the server can't unwrap. Run payloads are encrypted *to the user's public
  key* via ECIES (X25519 ECDH + HKDF-SHA256 + XChaCha20-Poly1305). **The Go and browser implementations
  are byte-for-byte wire-compatible** — this is the project's central invariant. Any change to the ECIES
  scheme or key-wrapping must be made in both `internal/crypto` and `web/src/crypto` and verified with
  the interop test, or CLI and web stop interoperating.
- **The server stores ciphertext only.** It can see email, public key, wrapped-private-key blob, token
  SHA-256 hashes, and run status+timing (plaintext, for routing/UI coloring). It cannot see passwords,
  private keys, plaintext tokens, or pipeline/branch/commit/message details.
- **Tokens.** Generated server-side (32 random bytes), returned once, stored only as `SHA-256(token)`.
  Bound to a project and (to receive runs) a pipeline. Revoke flips `active=false`; hash lookup ignores
  inactive tokens.
- **Project-scoped token routing (`internal/routing`).** Because pipeline names are stored encrypted,
  the server can't match an incoming webhook's plaintext pipeline name against them. Instead both sides
  hash the normalized name (`routing.Key` = SHA-256 of trim+lowercase name; same logic in
  `web/src/crypto/routing.ts`) and match on that hash, letting one project-wide token route to the right
  pipeline. Pipeline-bound tokens skip this.
- **Delivery.** `internal/server/handlers/webhook.go` encrypts + persists + fans out to the `SSEHub`
  (`/api/events`) and the Web Push `Dispatcher` (`internal/push`). Push uses VAPID Web Push only — there
  is **no Firebase/FCM**; the `FCM_SERVICE_ACCOUNT_JSON` config field is an unused stub. Browsers'
  built-in push services handle delivery, so no third-party push account is required.
- **Migrations** live in `migrations/` (golang-migrate, embedded via `migrations/embed.go`) and run
  automatically on server startup.
- **Web app** is a React Router SPA. The Go server optionally serves the built `web/dist` from
  `STATIC_DIR` with history-API fallback (`serveSPA` in `router.go`).

## Conventions

- Web Push **requires HTTPS** except on `localhost` — production needs a TLS reverse proxy.
- The CLI caches the decrypted private key in `~/.config/pipepush/config.json` (0600); the browser keeps
  it in memory only. Losing the password makes historical encrypted runs unrecoverable.
- Targets Go 1.26.
