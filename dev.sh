#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

go run ./cmd/blacksand -addr :8787 -proxy 127.0.0.1:8788 &

( cd web && npm run dev ) &

echo "[dev] Backend :8787 / proxy :8788 / frontend http://localhost:5173 (Ctrl-C to stop)"
wait
