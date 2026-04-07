#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

GO_CMD=""
if command -v go >/dev/null 2>&1; then
	GO_CMD="go"
elif command -v go.exe >/dev/null 2>&1; then
	GO_CMD="go.exe"
elif [ -x "/mnt/c/Program Files/Go/bin/go.exe" ]; then
	GO_CMD="/mnt/c/Program Files/Go/bin/go.exe"
fi

if [ -z "$GO_CMD" ]; then
	echo "go is required but was not found in PATH or in the default Windows install location" >&2
	exit 1
fi

PIDS=()

free_port() {
	local port="$1"
	if command -v powershell.exe >/dev/null 2>&1; then
		powershell.exe -NoProfile -Command "Get-NetTCPConnection -LocalPort ${port} -State Listen -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -Unique | ForEach-Object { Stop-Process -Id \$_ -Force -ErrorAction SilentlyContinue }" >/dev/null 2>&1 || true
	fi
}

cleanup() {
	local exit_code=$?
	if [ ${#PIDS[@]} -gt 0 ]; then
		kill "${PIDS[@]}" >/dev/null 2>&1 || true
		wait "${PIDS[@]}" >/dev/null 2>&1 || true
	fi
	exit "$exit_code"
}
trap cleanup EXIT INT TERM

start_service() {
	local name="$1"
	local port="$2"
	shift 2

	echo "Starting ${name} on ${port}..."
	"$@" >"${name}.log" 2>&1 &
	PIDS+=("$!")
}

free_port 8080
free_port 8081
free_port 8082

start_service stock 8082 env PORT=8082 "$GO_CMD" run ./services/stock
start_service payment 8081 env PORT=8081 STOCK_URL=http://localhost:8082 "$GO_CMD" run ./services/payment
start_service order 8080 env PORT=8080 PAYMENT_URL=http://localhost:8081 "$GO_CMD" run ./services/order

echo "All services started. Logs: order.log, payment.log, stock.log"
echo "Press Ctrl+C to stop."

wait
