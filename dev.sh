#!/usr/bin/env bash
#
set -euo pipefail
cd "$(dirname "$0")"

cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

# ( go run,  ).  work agent  " " .
go run ./cmd/artex -addr :8787 -proxy 127.0.0.1:8788 &

#   (Vite/Next dev server, /api  :8787   ).
( cd web && npm run dev ) &

echo "[dev]  :8787 /  :8788 /  http://localhost:5173  (Ctrl-C  )"
wait
