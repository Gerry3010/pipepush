# Self-hosting pipepush

pipepush is a single Go binary + PostgreSQL. The Docker image also serves the
web app, so one container + a database is all you need.

## With Docker Compose (recommended)

```bash
git clone https://github.com/Gerry3010/pipepush.git
cd pipepush
cp .env.example .env
```

Fill in `.env`:

```bash
# JWT secret
openssl rand -base64 32

# VAPID keys for Web Push
docker compose run --rm pipepush pipepush-server -gen-vapid-keys
```

Paste those values into `.env`, set `POSTGRES_PASSWORD` and `BASE_URL`, then:

```bash
docker compose up -d
docker compose logs -f pipepush     # watch migrations + startup
```

The server listens on `:8080` (web UI + API). Migrations run automatically on
start.

## TLS / reverse proxy (required for Web Push)

Browsers only allow Web Push over HTTPS. Put pipepush behind a TLS proxy.
Example with Caddy:

```
pipepush.example.com {
    reverse_proxy localhost:8080
}
```

Set `BASE_URL=https://pipepush.example.com` in `.env` to match.

## Configuration reference

| Env var             | Required | Description                                          |
| ------------------- | -------- | ---------------------------------------------------- |
| `DATABASE_URL`      | yes      | PostgreSQL DSN                                        |
| `JWT_SECRET`        | yes      | HS256 signing secret (`openssl rand -base64 32`)     |
| `PORT`              | no       | Listen port (default `8080`)                         |
| `BASE_URL`          | no       | Public URL (links/CORS)                              |
| `VAPID_PUBLIC_KEY`  | no\*     | Web Push public key (\*required for browser push)    |
| `VAPID_PRIVATE_KEY` | no\*     | Web Push private key                                 |
| `VAPID_EMAIL`       | no       | Contact email sent to push services                  |
| `STATIC_DIR`        | no       | Directory to serve the web app from (set in image)   |

## Backups

All state is in PostgreSQL. Back up the `postgres_data` volume or use
`pg_dump`. Note: pipeline data is end-to-end encrypted — a database dump contains
only ciphertext, which is safe at rest but **useless without users' passwords**.

## Upgrading

```bash
git pull            # or pull a new image tag
docker compose build
docker compose up -d
```

Migrations are applied automatically and are forward-only.

## Without Docker

```bash
go build -o pipepush-server ./cmd/pipepush-server
cd web && npm install && npm run build && cd ..
export DATABASE_URL=... JWT_SECRET=... STATIC_DIR=$PWD/web/dist
./pipepush-server
```
