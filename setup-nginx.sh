#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# pipepush — nginx + Let's Encrypt reverse-proxy setup
#
# Sets up system nginx as a hardened TLS reverse proxy in front of the pipepush
# server (127.0.0.1:PORT). Web Push REQUIRES HTTPS, so this is the recommended
# production front end. Safe to re-run (idempotent).
#
# It solves the cert bootstrap problem properly: an HTTP-only vhost is brought up
# first, the certificate is obtained via the webroot challenge, and only then is
# the hardened HTTPS vhost written — so nginx never references a cert that does
# not exist yet.
#
# Usage:
#   sudo bash setup-nginx.sh --domain pipepush.example.com --email you@example.com
#   STAGING=1 bash setup-nginx.sh ...      # Let's Encrypt staging (testing)
#
# Run it on the server after the stack is up (docker compose up -d).
# Tip: bind the backend to localhost in docker-compose.yml for production:
#   ports: ["127.0.0.1:8080:8080"]
# ──────────────────────────────────────────────────────────────────────────────

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Colours / output helpers ──────────────────────────────────────────────────
RED='\033[0;31m'; GRN='\033[0;32m'; YLW='\033[1;33m'; BLU='\033[0;34m'; BLD='\033[1m'; RST='\033[0m'
info()  { echo -e "${GRN}[+]${RST} $*"; }
warn()  { echo -e "${YLW}[!]${RST} $*"; }
error() { echo -e "${RED}[✗]${RST} $*" >&2; exit 1; }
ok()    { echo -e "${GRN}  ✓${RST}  $*"; }
hint()  { echo -e "${BLU}      ↳${RST} $*"; }
step()  { echo -e "\n${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RST}\n${BLD}  $*${RST}\n${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RST}"; }

# ── Defaults / config ─────────────────────────────────────────────────────────
DOMAIN="${DOMAIN:-}"
ADMIN_EMAIL="${ADMIN_EMAIL:-}"
SERVER_PORT="${PORT:-8080}"
WEBROOT="/var/www/certbot"
STAGING="${STAGING:-0}"
MAX_BODY="${MAX_BODY:-25M}"

# Pull defaults from an existing .env (pipepush uses PORT and BASE_URL)
ENV_FILE="${ENV_FILE:-$SCRIPT_DIR/.env}"
if [[ -f "$ENV_FILE" ]]; then
  PORT_FROM_ENV=$(grep -E '^PORT=' "$ENV_FILE" | cut -d= -f2- || true)
  [[ -n "${PORT_FROM_ENV:-}" ]] && SERVER_PORT="$PORT_FROM_ENV"
  if [[ -z "$DOMAIN" ]]; then
    BASE_URL=$(grep -E '^BASE_URL=' "$ENV_FILE" | cut -d= -f2- || true)
    DOMAIN=$(printf '%s' "$BASE_URL" | sed -E 's#^https?://##; s#/.*$##; s#:.*$##')
    [[ "$DOMAIN" == "localhost" ]] && DOMAIN=""
  fi
  [[ -z "$ADMIN_EMAIL" ]] && ADMIN_EMAIL=$(grep -E '^VAPID_EMAIL=' "$ENV_FILE" | cut -d= -f2- || true)
fi

# ── Argument parsing ──────────────────────────────────────────────────────────
usage() {
  echo -e "${BLD}pipepush nginx setup${RST}

  --domain   <host>   Public domain (e.g. pipepush.example.com)
  --email    <addr>   Email for Let's Encrypt expiry notices
  --port     <port>   Backend port to proxy to (default: ${SERVER_PORT})
  --staging           Use Let's Encrypt staging (no rate limits, untrusted cert)
  -h, --help          Show this help

Values fall back to \$DOMAIN / \$ADMIN_EMAIL / .env when flags are omitted."
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

maybe_sudo() { if [[ "$(id -u)" -eq 0 ]]; then "$@"; else sudo "$@"; fi; }

ask() {
  local var="$1" prompt="$2"
  [[ -n "${!var:-}" ]] && { info "$prompt: ${!var}"; return; }
  local val; read -rp "  $prompt: " val
  [[ -n "$val" ]] || error "$prompt is required."
  printf -v "$var" '%s' "$val"
}

echo
echo -e "${BLD}  pipepush — nginx + TLS reverse proxy${RST}"
echo -e "  Hardened HTTPS in front of the pipepush server (needed for Web Push)"
echo

# ── 1. Prerequisites & packages ───────────────────────────────────────────────
step "1/7  Prerequisites"

command -v curl &>/dev/null || error "'curl' not found — install it first."
[[ "$(id -u)" -eq 0 ]] || command -v sudo &>/dev/null || error "need root or sudo"

PKG=""
for c in apt-get dnf pacman; do command -v "$c" &>/dev/null && { PKG="$c"; break; }; done
[[ -n "$PKG" ]] || warn "No known package manager found — nginx/certbot must be preinstalled."

pkg_install() {
  case "$PKG" in
    apt-get) maybe_sudo apt-get update -qq && maybe_sudo apt-get install -y "$@" ;;
    dnf)     maybe_sudo dnf install -y "$@" ;;
    pacman)  maybe_sudo pacman -Sy --noconfirm "$@" ;;
    *)       error "Cannot auto-install: $* — install manually and re-run." ;;
  esac
}

if ! command -v nginx &>/dev/null; then
  info "Installing nginx…"; pkg_install nginx; ok "nginx installed"
else ok "nginx present"; fi

if ! command -v certbot &>/dev/null; then
  info "Installing certbot…"; pkg_install certbot; ok "certbot installed"
else ok "certbot present"; fi

maybe_sudo systemctl enable --now nginx &>/dev/null || true

# ── 2. Inputs ─────────────────────────────────────────────────────────────────
step "2/7  Configuration"
ask DOMAIN      "Domain (e.g. pipepush.example.com)"
ask ADMIN_EMAIL "Admin e-mail (Let's Encrypt notices)"
info "Proxy target: http://127.0.0.1:${SERVER_PORT}"
[[ "$STAGING" == "1" ]] && warn "STAGING mode: certificate will NOT be browser-trusted."

# ── 3. DNS preflight (soft) ───────────────────────────────────────────────────
step "3/7  DNS & connectivity check"

resolve_ip() {
  if command -v dig &>/dev/null; then dig +short A "$1" | tail -n1
  elif command -v getent &>/dev/null; then getent hosts "$1" | awk '{print $1}' | tail -n1
  else echo ""; fi
}
PUBLIC_IP=$(curl -fsS --max-time 8 https://api.ipify.org 2>/dev/null || echo "")
DOMAIN_IP=$(resolve_ip "$DOMAIN")

if [[ -n "$PUBLIC_IP" && -n "$DOMAIN_IP" ]]; then
  if [[ "$PUBLIC_IP" == "$DOMAIN_IP" ]]; then
    ok "$DOMAIN → $DOMAIN_IP (matches this server)"
  else
    warn "$DOMAIN resolves to $DOMAIN_IP but this server is $PUBLIC_IP"
    hint "If the A record is wrong, certbot's HTTP-01 challenge will fail."
    read -rp "  Continue anyway? [y/N] " yn; [[ "${yn,,}" == "y" ]] || error "Aborted — fix DNS first."
  fi
else
  warn "Could not verify DNS automatically — make sure $DOMAIN points here."
fi

# ── 4. Backend reachability (soft) ────────────────────────────────────────────
step "4/7  Backend health"
if curl -fsS --max-time 5 "http://127.0.0.1:${SERVER_PORT}/health" &>/dev/null; then
  ok "Backend healthy on 127.0.0.1:${SERVER_PORT}"
else
  warn "Backend not reachable on 127.0.0.1:${SERVER_PORT}/health."
  hint "Start it first:  docker compose up -d"
  hint "Continuing — nginx will return 502 until the backend is up."
fi

# ── 5. HTTP bootstrap vhost (serves ACME + proxy) ─────────────────────────────
step "5/7  nginx vhost (HTTP bootstrap)"

if [[ -d /etc/nginx/sites-available ]]; then
  VHOST="/etc/nginx/sites-available/pipepush"
  VHOST_LINK="/etc/nginx/sites-enabled/pipepush"
else
  VHOST="/etc/nginx/conf.d/pipepush.conf"
  VHOST_LINK=""
fi
info "vhost file: $VHOST"

maybe_sudo mkdir -p "$WEBROOT"

maybe_sudo tee "$VHOST" >/dev/null <<NGINX
# Managed by setup-nginx.sh — HTTP bootstrap (pre-certificate)
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};

    location /.well-known/acme-challenge/ {
        root ${WEBROOT};
    }

    location / {
        proxy_pass         http://127.0.0.1:${SERVER_PORT};
        proxy_http_version 1.1;
        proxy_set_header   Host              \$host;
        proxy_set_header   X-Real-IP         \$remote_addr;
        proxy_set_header   X-Forwarded-For   \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;
    }
}
NGINX

[[ -n "$VHOST_LINK" && ! -L "$VHOST_LINK" ]] && maybe_sudo ln -s "$VHOST" "$VHOST_LINK"
maybe_sudo rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true

maybe_sudo nginx -t || error "nginx config test failed (HTTP bootstrap)"
maybe_sudo systemctl reload nginx
ok "HTTP vhost live — ACME challenge path ready"

# ── 6. Firewall + certificate ─────────────────────────────────────────────────
step "6/7  Firewall & certificate"

if command -v ufw &>/dev/null && maybe_sudo ufw status 2>/dev/null | grep -q "Status: active"; then
  maybe_sudo ufw allow 80/tcp  &>/dev/null || true
  maybe_sudo ufw allow 443/tcp &>/dev/null || true
  ok "ufw: opened ports 80 and 443"
fi

CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
if maybe_sudo test -f "${CERT_DIR}/fullchain.pem"; then
  ok "Certificate already exists for ${DOMAIN} — skipping issuance"
else
  info "Requesting certificate via webroot challenge…"
  CB_ARGS=(certonly --webroot -w "$WEBROOT" -d "$DOMAIN"
           --non-interactive --agree-tos -m "$ADMIN_EMAIL" --no-eff-email)
  [[ "$STAGING" == "1" ]] && CB_ARGS+=(--staging)
  maybe_sudo certbot "${CB_ARGS[@]}" \
    || error "certbot failed. Common causes: DNS not pointing here, or port 80 blocked."
  ok "Certificate obtained"
fi

RENEW_HOOK="/etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh"
maybe_sudo mkdir -p "$(dirname "$RENEW_HOOK")"
maybe_sudo tee "$RENEW_HOOK" >/dev/null <<'HOOK'
#!/usr/bin/env bash
systemctl reload nginx
HOOK
maybe_sudo chmod +x "$RENEW_HOOK"
ok "Auto-renewal reload hook installed"

# ── 7. Hardened HTTPS vhost ───────────────────────────────────────────────────
step "7/7  nginx vhost (hardened HTTPS)"

SSL_OPTS_INC=""
maybe_sudo test -f /etc/letsencrypt/options-ssl-nginx.conf \
  && SSL_OPTS_INC="    include /etc/letsencrypt/options-ssl-nginx.conf;"
DHPARAM_INC=""
maybe_sudo test -f /etc/letsencrypt/ssl-dhparams.pem \
  && DHPARAM_INC="    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;"

maybe_sudo tee "$VHOST" >/dev/null <<NGINX
# Managed by setup-nginx.sh — hardened HTTPS reverse proxy
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

    client_max_body_size ${MAX_BODY};
    gzip on;
    gzip_types application/json text/css application/javascript;

    location / {
        proxy_pass         http://127.0.0.1:${SERVER_PORT};
        proxy_http_version 1.1;
        proxy_set_header   Host              \$host;
        proxy_set_header   X-Real-IP         \$remote_addr;
        proxy_set_header   X-Forwarded-For   \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;

        # SSE stream (/api/events) — disable buffering, keep alive
        proxy_set_header   Connection        "";
        proxy_buffering    off;
        proxy_read_timeout 1h;
    }
}
NGINX

maybe_sudo nginx -t || error "nginx config test failed (HTTPS vhost)"
maybe_sudo systemctl reload nginx
ok "HTTPS vhost live"

# ── Verify end-to-end ─────────────────────────────────────────────────────────
echo
VERIFY_FLAG=""; [[ "$STAGING" == "1" ]] && VERIFY_FLAG="-k"
if curl -fsS $VERIFY_FLAG --max-time 10 "https://${DOMAIN}/health" &>/dev/null; then
  ok "https://${DOMAIN}/health responding 🎉"
else
  warn "Could not verify https://${DOMAIN}/health yet (backend down, or DNS still propagating)."
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo
echo -e "${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RST}"
echo -e "${BLD}  Done — pipepush is reverse-proxied over TLS${RST}"
echo -e "${BLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RST}"
echo
echo "  Web UI:    https://${DOMAIN}/"
echo "  Webhook:   https://${DOMAIN}/api/webhook"
echo "  Health:    https://${DOMAIN}/health"
echo
echo "  Point the CLI at it:  pipepush server set https://${DOMAIN}"
echo "  Renewal:   certbot renew --dry-run   (auto-reloads nginx on success)"
echo
[[ "$STAGING" == "1" ]] && warn "Re-run WITHOUT --staging to get a browser-trusted certificate."
echo
