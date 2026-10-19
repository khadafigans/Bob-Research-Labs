#!/usr/bin/env bash
# Bob Research Labs update: rebuild from THIS source (never pull autumn27/artex).
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }

arch_dir(){
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *) die "unsupported architecture: $(uname -m)" ;;
  esac
}

sync_repo(){
  [ -d .git ] && command -v git >/dev/null 2>&1 || { warn "not a git checkout — skip git pull"; return; }
  [ "$(ask 'git pull --ff-only? (y/n)' y)" = y ] || return
  if ! git pull --ff-only; then
    warn "git pull --ff-only failed; continuing with current tree"
  fi
}

build_embedded_binary(){
  local arch="$1"
  local out="dist/${arch}/artex"
  command -v go >/dev/null 2>&1 || die "Go not found: https://go.dev/dl/"
  command -v npm >/dev/null 2>&1 || die "npm required for English UI embed"
  info "building English frontend…"
  ( cd web && npm ci && NEXT_PUBLIC_LOCALE=en NEXT_EXPORT=1 npm run build:static )
  rm -rf server/webui/dist && mkdir -p server/webui && cp -a web/out server/webui/dist
  mkdir -p "dist/${arch}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" go build -tags embedui -trimpath -o "${out}" ./cmd/artex
  ok "binary → ${out}"
}

update_docker(){
  command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 \
    || die "docker / docker compose missing — run ./install.sh first"
  [ -f .env ] || die ".env missing — run ./install.sh first"

  local arch; arch="$(arch_dir)"
  build_embedded_binary "$arch"
  info "rebuilding local image (no Hub pull)…"
  docker compose build
  info "restarting artex (schema migrates on boot)…"
  docker compose up -d artex
  ok "updated → http://localhost:8787"
  info "logs: docker compose logs -f artex"
}

update_local(){
  command -v go >/dev/null 2>&1 || die "Go not found: https://go.dev/dl/"
  [ -f config.json ] || warn "config.json missing — use ./install.sh for first setup"
  ok "Go: $(go version)"

  if command -v npm >/dev/null 2>&1; then
    info "rebuilding English frontend…"
    ( cd web && npm ci && NEXT_PUBLIC_LOCALE=en NEXT_EXPORT=1 npm run build:static )
    rm -rf server/webui/dist && cp -a web/out server/webui/dist
    CGO_ENABLED=0 go build -tags embedui -trimpath -o artex ./cmd/artex
  else
    warn "npm missing — backend-only rebuild"
    CGO_ENABLED=0 go build -o artex ./cmd/artex
  fi
  ok "built → ./artex"
  warn "restart the running process to apply"
}

echo "=============================="
echo "  Bob Research Labs update"
echo "  1) Docker (rebuild local image)"
echo "  2) Local (go recompile)  [default]"
echo "=============================="
case "$(ask 'Choice' 2)" in
  1) sync_repo; update_docker ;;
  2) sync_repo; update_local ;;
  *) die "invalid choice" ;;
esac
