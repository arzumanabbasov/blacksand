#!/usr/bin/env bash
# Blacksand install script: ① all-in-Docker  ② build and run locally
#
# Messages are English by default. For Korean, set BLACKSAND_LANGUAGE=ko
# (BLACKSAND_LANGUAGE=ko also works) before running, e.g. BLACKSAND_LANGUAGE=ko ./install.sh
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

# msg EN KO → print the string for the selected language.

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }
rand(){ head -c 18 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24; }

# ── docker environment detection / auto-install ─────────────
ensure_docker(){
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    ok "Found docker and docker compose"; return
  fi
  warn "docker / docker compose not found"
  case "$(uname -s)" in
    Linux)
      if [ "$(ask "Install Docker automatically? (y/n)" y)" = y ]; then
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker "$USER" || true
        ok "Docker installed (group change needs re-login to use without sudo)"
      else
        die "Please install docker yourself and retry"
      fi ;;
    Darwin) die "On macOS install Docker Desktop: https://www.docker.com/products/docker-desktop/" ;;
    *)      die "Please install docker yourself and retry" ;;
  esac
}

# ── ① all-in-Docker ─────────────────────────────────────────
install_docker(){
  ensure_docker
  if [ ! -f .env ]; then
    cp .env.example .env 2>/dev/null || true
    local pw key
    pw="$(ask "Postgres password (Enter for a random one)" "$(rand)")"
    key="$(ask "ANTHROPIC_API_KEY (optional, can set later in the UI)" '')"
    sed -i.bak "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${pw}|" .env
    sed -i.bak "s|^ANTHROPIC_API_KEY=.*|ANTHROPIC_API_KEY=${key}|" .env
    rm -f .env.bak
    ok ".env generated (POSTGRES_PASSWORD set)"
  else
    info "Reusing existing .env"
  fi
  info "Building Blacksand from this checkout…"
  docker compose build blacksand
  docker compose up -d
  ok "Started → http://localhost:8787"
  info "View logs: docker compose logs -f blacksand"
}

# ── ② build and run locally ─────────────────────────────────
install_local(){
  echo "Database setup:"
  echo "  1) Connect to an existing PostgreSQL"
  echo "  2) Start a PostgreSQL with Docker (needs docker)"
  case "$(ask "Choose" 1)" in
    2)
      ensure_docker
      local pw; pw="$(ask "Postgres password (Enter for a random one)" "$(rand)")"
      docker run -d --name blacksand-pg -p 5432:5432 \
        -e POSTGRES_USER=blacksand -e POSTGRES_PASSWORD="$pw" -e POSTGRES_DB=blacksand \
        -v blacksand-pg:/var/lib/postgresql/data postgres:16-alpine
      DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=blacksand DB_PASS="$pw" DB_NAME=blacksand DB_SSL=disable ;;
    *)
      DB_HOST="$(ask "Database host" 127.0.0.1)"
      DB_PORT="$(ask "Port" 5432)"
      DB_USER="$(ask "User" blacksand)"
      DB_PASS="$(ask "Password" '')"
      DB_NAME="$(ask "Database name" blacksand)"
      DB_SSL="$(ask "sslmode (disable/require)" disable)" ;;
  esac

  # Generate config.json
  cat > config.json <<JSON
{
  "database": {
    "host": "${DB_HOST}",
    "port": ${DB_PORT},
    "user": "${DB_USER}",
    "password": "${DB_PASS}",
    "dbname": "${DB_NAME}",
    "sslmode": "${DB_SSL}"
  }
}
JSON
  ok "config.json generated"

  # go environment check
  command -v go >/dev/null 2>&1 || die "Go not found, install Go (>=1.26): https://go.dev/dl/"
  ok "Go: $(go version)"

  # Embedding the frontend needs node to produce the static output
  if command -v npm >/dev/null 2>&1; then
    info "Building the frontend static output…"
    ( cd web && npm ci && npm run build:static )
    rm -rf server/webui/dist && cp -r web/out server/webui/dist
    info "Compiling the single binary with the frontend embedded…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o blacksand ./cmd/blacksand
  else
    warn "npm not found: building the backend WITHOUT the embedded frontend (run npm run dev for the frontend)"
    CGO_ENABLED=0 go build -o blacksand ./cmd/blacksand
  fi
  ok "Build complete → ./blacksand"

  info "Starting… (Ctrl-C to quit)"
  ./start.sh
}

echo "=============================="
echo "  Blacksand install"
echo "  1) All-in-Docker install"
echo "  2) Local run (go build)"
echo "=============================="
case "$(ask "Choose" 1)" in
  1) install_docker ;;
  2) install_local ;;
  *) die "Invalid choice" ;;
esac
