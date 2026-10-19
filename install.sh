#!/usr/bin/env bash
# Bob Research Labs install: (1) Docker from THIS source  (2) local Go build
# Never pulls upstream autumn27/artex (that image is Korean ARTEX).
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }
rand(){ head -c 18 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24; }

arch_dir(){
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *) die "unsupported architecture: $(uname -m) (need amd64 or arm64)" ;;
  esac
}

ensure_docker(){
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    ok "docker + docker compose found"; return
  fi
  warn "docker / docker compose not found"
  case "$(uname -s)" in
    Linux)
      if [ "$(ask 'Install Docker automatically? (y/n)' y)" = y ]; then
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker "$USER" || true
        ok "Docker installed (re-login may be required for non-sudo docker)"
      else
        die "install docker, then re-run"
      fi ;;
    Darwin) die "install Docker Desktop: https://www.docker.com/products/docker-desktop/" ;;
    *)      die "install docker, then re-run" ;;
  esac
}

# Build English embedded UI + linux binary into dist/<arch>/artex for the Dockerfile.
build_embedded_binary(){
  local arch="$1"
  local out="dist/${arch}/artex"
  command -v go >/dev/null 2>&1 || die "Go not found (>=1.21): https://go.dev/dl/"
  command -v npm >/dev/null 2>&1 || die "npm/Node not found — required to embed the English UI"
  ok "Go: $(go version)"
  info "building English frontend (NEXT_PUBLIC_LOCALE=en)…"
  ( cd web && npm ci && NEXT_PUBLIC_LOCALE=en NEXT_EXPORT=1 npm run build:static )
  rm -rf server/webui/dist
  mkdir -p server/webui
  cp -a web/out server/webui/dist
  info "compiling embedui binary → ${out}"
  mkdir -p "dist/${arch}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" go build -tags embedui -trimpath -o "${out}" ./cmd/artex
  ok "binary ready: ${out}"
}

# ── (1) Docker from this repo (no Hub pull of autumn27/artex) ──
install_docker(){
  ensure_docker
  if [ ! -f .env ]; then
    cp .env.example .env 2>/dev/null || true
    local pw key
    pw="$(ask 'Postgres password (Enter = random)' "$(rand)")"
    key="$(ask 'ANTHROPIC_API_KEY (optional, set later in UI)' '')"
    if grep -q '^POSTGRES_PASSWORD=' .env 2>/dev/null; then
      sed -i.bak "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${pw}|" .env
    else
      printf 'POSTGRES_PASSWORD=%s\n' "$pw" >> .env
    fi
    if grep -q '^ANTHROPIC_API_KEY=' .env 2>/dev/null; then
      sed -i.bak "s|^ANTHROPIC_API_KEY=.*|ANTHROPIC_API_KEY=${key}|" .env
    else
      printf 'ANTHROPIC_API_KEY=%s\n' "$key" >> .env
    fi
    rm -f .env.bak
    ok ".env created"
  else
    info "using existing .env"
  fi

  local arch; arch="$(arch_dir)"
  build_embedded_binary "$arch"

  info "building Docker image from THIS source (bob-research-labs:local)…"
  docker compose build
  info "starting…"
  docker compose up -d
  ok "up → http://localhost:8787"
  info "logs: docker compose logs -f artex"
}

# ── (2) Local compile + run ──
install_local(){
  echo "Database:"
  echo "  1) Connect to existing PostgreSQL"
  echo "  2) Start PostgreSQL via Docker"
  case "$(ask 'Choice' 1)" in
    2)
      ensure_docker
      local pw; pw="$(ask 'Postgres password (Enter = random)' "$(rand)")"
      docker run -d --name artex-pg -p 5432:5432 \
        -e POSTGRES_USER=artex -e POSTGRES_PASSWORD="$pw" -e POSTGRES_DB=artex \
        -v artex-pg:/var/lib/postgresql/data postgres:16-alpine
      DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=artex DB_PASS="$pw" DB_NAME=artex DB_SSL=disable ;;
    *)
      DB_HOST="$(ask 'DB host' 127.0.0.1)"
      DB_PORT="$(ask 'port' 5432)"
      DB_USER="$(ask 'user' artex)"
      DB_PASS="$(ask 'password' '')"
      DB_NAME="$(ask 'database' artex)"
      DB_SSL="$(ask 'sslmode (disable/require)' disable)" ;;
  esac

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
  ok "wrote config.json"

  command -v go >/dev/null 2>&1 || die "Go not found (>=1.21): https://go.dev/dl/"
  ok "Go: $(go version)"

  if command -v npm >/dev/null 2>&1; then
    info "building English frontend…"
    ( cd web && npm ci && NEXT_PUBLIC_LOCALE=en NEXT_EXPORT=1 npm run build:static )
    rm -rf server/webui/dist && mkdir -p server/webui && cp -a web/out server/webui/dist
    info "compiling embedui binary…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o artex ./cmd/artex
  else
    warn "npm not found: backend-only build (no embedded UI)"
    CGO_ENABLED=0 go build -o artex ./cmd/artex
  fi
  ok "built → ./artex"

  info "starting… (Ctrl-C to stop)"
  ./artex
}

echo "=============================="
echo "  Bob Research Labs install"
echo "  (English UI — builds from this tree)"
echo "  1) Docker (build local image)"
echo "  2) Local run (go compile)  [default]"
echo "=============================="
case "$(ask 'Choice' 2)" in
  1) install_docker ;;
  2) install_local ;;
  *) die "invalid choice" ;;
esac
