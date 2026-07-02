#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# pipepush — one-shot server deployment
#
# Installs Docker (if missing), brings up the full stack (pipepush + PostgreSQL)
# from the published image with freshly generated secrets, then sets up nginx +
# Let's Encrypt TLS in front of it. Safe to re-run (idempotent).
#
# Run on the target server as a user with sudo (or as root):
#   bash deploy.sh --domain pipepush.example.com --email you@example.com
#   STAGING=1 bash deploy.sh ...     # Let's Encrypt staging (testing)
# ──────────────────────────────────────────────────────────────────────────────

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
DOMAIN="${DOMAIN:-}"
ADMIN_EMAIL="${ADMIN_EMAIL:-}"
INSTALL_DIR="${INSTALL_DIR:-/opt/pipepush}"
IMAGE="${IMAGE:-ghcr.io/gerry3010/pipepush:latest}"
SERVER_PORT="${PORT:-8080}"
WEBROOT="/var/www/certbot"
STAGING="${STAGING:-0}"

# ── Output helpers ────────────────────────────────────────────────────────────
RED='\033[0;31m'; GRN='\033[0;32m'; YLW='\033[1;33m'; BLU='\033[0;34m'; BLD='\033[1m'; RST='\033[0m'
info()  { echo -e "${GRN}[+]${RST} $*"; }
warn()  { echo -e "${YLW}[!]${RST} $*"; }
error() { echo -e "${RED}[✗]${RST} $*" >&2; exit 1; }
ok()    { echo -e "${GRN}  ✓${RST}  $*"; }
hint()  { echo -e "${BLU}      ↳${RST} $*"; }
step()  { echo -e "\n${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RST}\n${BLD}  $*${RST}\n${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RST}"; }

maybe_sudo() { if [[ "$(id -u)" -eq 0 ]]; then "$@"; else sudo "$@"; fi; }

# True if something is already listening on the given TCP port (host side).
port_in_use() { ss -ltnH "sport = :$1" 2>/dev/null | grep -q .; }

pkg_install() {
  if   command -v apt-get &>/dev/null; then maybe_sudo apt-get update -qq && maybe_sudo apt-get install -y "$@"
  elif command -v dnf     &>/dev/null; then maybe_sudo dnf install -y "$@"
  elif command -v pacman  &>/dev/null; then maybe_sudo pacman -Sy --noconfirm "$@"
  else error "Cannot auto-install: $* — install manually and re-run."; fi
}

ask() {
  local var="$1" prompt="$2"
  [[ -n "${!var:-}" ]] && { info "$prompt: ${!var}"; return; }
  local val; read -rp "  $prompt: " val
  [[ -n "$val" ]] || error "$prompt is required."
  printf -v "$var" '%s' "$val"
}

usage() {
  echo -e "${BLD}pipepush deployment${RST}

  --domain  <host>   Public domain (e.g. pipepush.example.com)
  --email   <addr>   Email for Let's Encrypt + VAPID contact
  --port    <port>   Host port to bind the backend to (default: 8080)
  --staging          Let's Encrypt staging (no rate limits, untrusted cert)
  -h, --help         Show this help"
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain) DOMAIN="$2"; shift 2 ;;
    --email)  ADMIN_EMAIL="$2"; shift 2 ;;
    --port)   SERVER_PORT="$2"; shift 2 ;;
    --staging) STAGING=1; shift ;;
    -h|--help) usage ;;
    *) error "Unknown argument: $1 (try --help)" ;;
  esac
done

echo
echo -e "${BLD}  pipepush — server deployment${RST}"
echo -e "  Docker stack + nginx + TLS, end to end"
echo

# ── 1. Docker ─────────────────────────────────────────────────────────────────
step "1/6  Docker"

if ! command -v docker &>/dev/null; then
  warn "Docker not found."
  read -rp "  Install Docker now via get.docker.com? [Y/n] " yn
  if [[ "${yn,,}" != "n" ]]; then
    curl -fsSL https://get.docker.com | maybe_sudo sh
    maybe_sudo systemctl enable --now docker
    # Allow the current non-root user to use docker without sudo
    [[ "$(id -u)" -ne 0 ]] && maybe_sudo usermod -aG docker "$USER" || true
    ok "Docker installed"
    hint "If 'docker' still needs sudo, log out/in once (group change) — this run uses sudo as needed."
  else
    error "Docker is required. Install it and re-run."
  fi
else
  ok "Docker present"
fi

# docker compose plugin
DC=(docker compose)
if ! docker compose version &>/dev/null; then
  if command -v docker-compose &>/dev/null; then
    DC=(docker-compose)
  else
    error "docker compose plugin not found."
  fi
fi

# Wrap docker in sudo if the daemon isn't reachable as this user yet
DOCKER_SUDO=()
if ! docker info &>/dev/null; then
  if maybe_sudo docker info &>/dev/null; then DOCKER_SUDO=(sudo); else error "Cannot talk to the Docker daemon."; fi
fi
dc() { "${DOCKER_SUDO[@]}" "${DC[@]}" "$@"; }
dkr() { "${DOCKER_SUDO[@]}" docker "$@"; }

require_curl() { command -v curl &>/dev/null || error "'curl' not found."; }
require_curl
command -v openssl &>/dev/null || error "'openssl' not found."

# ── 2. Inputs ─────────────────────────────────────────────────────────────────
step "2/6  Configuration"
ask DOMAIN      "Domain (e.g. pipepush.example.com)"
ask ADMIN_EMAIL "Admin e-mail (Let's Encrypt + VAPID)"

# On re-run, reuse the host port the existing stack already binds — otherwise the
# in-use guard below trips on our own running container and would pick a new port.
REUSE_PORT=0
if [[ -f "${INSTALL_DIR}/docker-compose.yml" ]]; then
  EXISTING_PORT=$(sed -nE 's/.*127\.0\.0\.1:([0-9]+):8080.*/\1/p' "${INSTALL_DIR}/docker-compose.yml" | head -1)
  if [[ -n "${EXISTING_PORT}" ]]; then
    [[ -z "${PORT:-}" ]] && SERVER_PORT="${EXISTING_PORT}"
    [[ "${SERVER_PORT}" == "${EXISTING_PORT}" ]] && REUSE_PORT=1
  fi
fi

# Host port: default 8080, but detect conflicts and offer a free alternative.
if [[ "${REUSE_PORT}" -eq 0 ]] && port_in_use "$SERVER_PORT"; then
  warn "Host port ${SERVER_PORT} is already in use on this server."
  SUGGEST="$SERVER_PORT"
  for p in 8090 8091 8092 8180 8280 9090; do port_in_use "$p" || { SUGGEST="$p"; break; }; done
  read -rp "  Pick a free host port [${SUGGEST}]: " ans
  SERVER_PORT="${ans:-$SUGGEST}"
  port_in_use "$SERVER_PORT" && error "Port ${SERVER_PORT} is also in use — pick another with --port."
fi
ok "Host port: ${SERVER_PORT}  (nginx → 127.0.0.1:${SERVER_PORT} → container :8080)"
[[ "$STAGING" == "1" ]] && warn "STAGING mode: certificate will NOT be browser-trusted."

maybe_sudo mkdir -p "$INSTALL_DIR"
maybe_sudo chown "$USER":"$USER" "$INSTALL_DIR" 2>/dev/null || true
cd "$INSTALL_DIR"

# ── 3. Compose + secrets ──────────────────────────────────────────────────────
step "3/6  Stack configuration"

cat > docker-compose.yml <<COMPOSE
services:
  pipepush:
    image: ${IMAGE}
    ports:
      - "127.0.0.1:${SERVER_PORT}:8080"
    environment:
      DATABASE_URL: postgres://pipepush:\${POSTGRES_PASSWORD}@postgres:5432/pipepush?sslmode=disable
      JWT_SECRET: \${JWT_SECRET}
      BASE_URL: https://${DOMAIN}
      VAPID_PUBLIC_KEY: \${VAPID_PUBLIC_KEY}
      VAPID_PRIVATE_KEY: \${VAPID_PRIVATE_KEY}
      VAPID_EMAIL: ${ADMIN_EMAIL}
    labels:
      com.centurylinklabs.watchtower.scope: pipepush
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  # Auto-deploy: watch ghcr for a new :latest (published by the release workflow
  # on every v* tag) and pull + restart pipepush automatically. Outbound-only, so
  # it works behind a firewall that blocks inbound SSH. Scoped so it only ever
  # touches the pipepush container, never postgres.
  watchtower:
    # Maintained fork: the original containrrr/watchtower is unmaintained and
    # ships a Docker API client too old for modern engines (it fails with
    # "client version 1.25 is too old; minimum supported API version is 1.40").
    image: ghcr.io/nicholas-fedor/watchtower:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: --cleanup --scope pipepush --interval 120
    labels:
      com.centurylinklabs.watchtower.scope: pipepush
    restart: unless-stopped

  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_DB: pipepush
      POSTGRES_USER: pipepush
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U pipepush"]
      interval: 5s
      timeout: 5s
      retries: 10
    restart: unless-stopped

volumes:
  postgres_data:
COMPOSE
ok "docker-compose.yml written"

if [[ -f .env ]]; then
  ok ".env already exists — keeping existing secrets"
else
  info "Generating secrets + VAPID keys…"
  VAPID_OUT=$(dkr run --rm "$IMAGE" -gen-vapid-keys) || error "VAPID keygen failed"
  {
    echo "# pipepush — generated by deploy.sh on $(date -Iseconds)"
    echo "POSTGRES_PASSWORD=$(openssl rand -base64 32 | tr -d '/+=\n')"
    echo "JWT_SECRET=$(openssl rand -base64 48 | tr -d '/+=\n')"
    echo "$VAPID_OUT"
  } > .env
  chmod 600 .env
  ok ".env written with random secrets"
fi

# ── 4. Start stack ────────────────────────────────────────────────────────────
step "4/6  Start stack"
info "Pulling images…"; dc pull
info "Starting…"; dc up -d

printf "  Waiting for backend"
HEALTHY=0
for _ in $(seq 1 45); do
  if curl -sf "http://127.0.0.1:${SERVER_PORT}/health" &>/dev/null; then HEALTHY=1; echo; break; fi
  printf "."; sleep 2
done
if [[ $HEALTHY -eq 0 ]]; then
  echo; dc logs --tail=30 pipepush || true
  error "Backend did not become healthy. Logs above."
fi
ok "Backend healthy on 127.0.0.1:${SERVER_PORT}"

# ── 5. nginx + certificate ────────────────────────────────────────────────────
step "5/6  nginx + TLS"

command -v nginx   &>/dev/null || { info "Installing nginx…";   pkg_install nginx; }
command -v certbot &>/dev/null || { info "Installing certbot…"; pkg_install certbot; }

if [[ -d /etc/nginx/sites-available ]]; then
  VHOST="/etc/nginx/sites-available/pipepush"; VHOST_LINK="/etc/nginx/sites-enabled/pipepush"
else
  VHOST="/etc/nginx/conf.d/pipepush.conf"; VHOST_LINK=""
fi
maybe_sudo mkdir -p "$WEBROOT"

# HTTP bootstrap vhost (serves ACME + proxy) so nginx is valid before the cert exists
maybe_sudo tee "$VHOST" >/dev/null <<NGINX
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};
    location /.well-known/acme-challenge/ { root ${WEBROOT}; }
    location / {
        proxy_pass http://127.0.0.1:${SERVER_PORT};
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
NGINX
[[ -n "$VHOST_LINK" && ! -L "$VHOST_LINK" ]] && maybe_sudo ln -s "$VHOST" "$VHOST_LINK"
maybe_sudo rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true
maybe_sudo systemctl enable --now nginx &>/dev/null || true
maybe_sudo nginx -t || error "nginx config test failed (HTTP bootstrap)"
maybe_sudo systemctl reload nginx
ok "HTTP vhost live"

if command -v ufw &>/dev/null && maybe_sudo ufw status 2>/dev/null | grep -q "Status: active"; then
  maybe_sudo ufw allow 80/tcp &>/dev/null || true
  maybe_sudo ufw allow 443/tcp &>/dev/null || true
  ok "ufw: opened 80 and 443"
fi

CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
if maybe_sudo test -f "${CERT_DIR}/fullchain.pem"; then
  ok "Certificate already exists — skipping issuance"
else
  info "Requesting certificate via webroot…"
  CB_ARGS=(certonly --webroot -w "$WEBROOT" -d "$DOMAIN" --non-interactive --agree-tos -m "$ADMIN_EMAIL" --no-eff-email)
  [[ "$STAGING" == "1" ]] && CB_ARGS+=(--staging)
  maybe_sudo certbot "${CB_ARGS[@]}" || error "certbot failed (DNS not pointing here, or port 80 blocked?)."
  ok "Certificate obtained"
fi

RENEW_HOOK="/etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh"
maybe_sudo mkdir -p "$(dirname "$RENEW_HOOK")"
printf '#!/usr/bin/env bash\nsystemctl reload nginx\n' | maybe_sudo tee "$RENEW_HOOK" >/dev/null
maybe_sudo chmod +x "$RENEW_HOOK"

# ── 6. Hardened HTTPS vhost ───────────────────────────────────────────────────
step "6/6  Hardened HTTPS"

SSL_OPTS_INC=""; maybe_sudo test -f /etc/letsencrypt/options-ssl-nginx.conf \
  && SSL_OPTS_INC="    include /etc/letsencrypt/options-ssl-nginx.conf;"
DHPARAM_INC="";  maybe_sudo test -f /etc/letsencrypt/ssl-dhparams.pem \
  && DHPARAM_INC="    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;"

maybe_sudo tee "$VHOST" >/dev/null <<NGINX
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};
    location /.well-known/acme-challenge/ { root ${WEBROOT}; }
    location / { return 301 https://\$host\$request_uri; }
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name ${DOMAIN};

    ssl_certificate     ${CERT_DIR}/fullchain.pem;
    ssl_certificate_key ${CERT_DIR}/privkey.pem;
${SSL_OPTS_INC}
${DHPARAM_INC}

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header Referrer-Policy strict-origin-when-cross-origin always;

    client_max_body_size 25M;
    gzip on;
    gzip_types application/json text/css application/javascript;

    location / {
        proxy_pass         http://127.0.0.1:${SERVER_PORT};
        proxy_http_version 1.1;
        proxy_set_header   Host              \$host;
        proxy_set_header   X-Real-IP         \$remote_addr;
        proxy_set_header   X-Forwarded-For   \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;
        # SSE stream (/api/events)
        proxy_set_header   Connection        "";
        proxy_buffering    off;
        proxy_read_timeout 1h;
    }
}
NGINX
maybe_sudo nginx -t || error "nginx config test failed (HTTPS vhost)"
maybe_sudo systemctl reload nginx
ok "HTTPS vhost live"

# ── Verify + done ─────────────────────────────────────────────────────────────
echo
VERIFY_FLAG=""; [[ "$STAGING" == "1" ]] && VERIFY_FLAG="-k"
if curl -fsS $VERIFY_FLAG --max-time 10 "https://${DOMAIN}/health" &>/dev/null; then
  ok "https://${DOMAIN}/health responding 🎉"
else
  warn "Could not verify https://${DOMAIN}/health yet (DNS still propagating?)."
fi

echo
echo -e "${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RST}"
echo -e "${BLD}  pipepush is live${RST}"
echo -e "${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RST}"
echo
echo "  Web UI:   https://${DOMAIN}/"
echo "  Stack:    cd ${INSTALL_DIR} && ${DC[*]} ps"
echo "  Logs:     cd ${INSTALL_DIR} && ${DC[*]} logs -f pipepush"
echo "  CLI:      pipepush server set https://${DOMAIN}  &&  pipepush login --register"
echo
[[ "$STAGING" == "1" ]] && warn "Re-run WITHOUT --staging for a browser-trusted certificate."
echo
