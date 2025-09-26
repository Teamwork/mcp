#!/usr/bin/env bash
set -euo pipefail

PORT="18080"
LOGFILE="/tmp/mcp_http_smoke.log"

cleanup() {
  set +e
  if lsof -ti tcp:"${PORT}" >/dev/null 2>&1; then
    lsof -ti tcp:"${PORT}" | xargs -r kill
    sleep 1
  fi
}
trap cleanup EXIT

cd "$(dirname "$0")/.."

# Ensure port free
cleanup

echo "[smoke] building modules..."
go build ./...

echo "[smoke] starting HTTP server on :${PORT}..."
TW_MCP_SERVER_ADDRESS=":${PORT}" TW_MCP_LOG_LEVEL=debug \
  go run cmd/mcp-http/main.go >"${LOGFILE}" 2>&1 &
SRV_PID=$!
sleep 2

if ! lsof -ti tcp:"${PORT}" >/dev/null 2>&1; then
  echo "[smoke] server failed to start; logs:" >&2
  sed -n '1,200p' "${LOGFILE}" >&2 || true
  exit 1
fi

echo "[smoke] checking health endpoint..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:${PORT}/api/health" || true)
if [[ "${STATUS}" != "204" ]]; then
  echo "[smoke] unexpected health status: ${STATUS}" >&2
  sed -n '1,200p' "${LOGFILE}" >&2 || true
  exit 1
fi

echo "[smoke] listing tools via CLI..."
go run cmd/mcp-http-cli/main.go -mcp-url="http://127.0.0.1:${PORT}" list-tools >/dev/null

echo "[smoke] success"
exit 0
