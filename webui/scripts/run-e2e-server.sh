#!/bin/sh
set -eu

ROOT_DIR="$(pwd)"
DATA_DIR="${ROOT_DIR}/.tmp/e2e-data"
LISTEN_ADDR="${E2E_LISTEN_ADDR:-127.0.0.1:18080}"
GO_CACHE_DIR="${ROOT_DIR}/.cache/go-build-e2e"
GO_TMP_DIR="${ROOT_DIR}/.tmp/go-tmp-e2e"

rm -rf "${DATA_DIR}"
mkdir -p "${DATA_DIR}"
mkdir -p "${GO_CACHE_DIR}" "${GO_TMP_DIR}"

export INITIAL_ADMIN_USERNAME="${INITIAL_ADMIN_USERNAME:-admin}"
export INITIAL_ADMIN_PASSWORD="${INITIAL_ADMIN_PASSWORD:-securepassword123}"
export INITIAL_ADMIN_EMAIL="${INITIAL_ADMIN_EMAIL:-admin@test.local}"
export INITIAL_ADMIN_FULL_NAME="${INITIAL_ADMIN_FULL_NAME:-Test Admin}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}"
export LOGIN_RATE_LIMIT_REQUESTS="${LOGIN_RATE_LIMIT_REQUESTS:-1000}"
export LOGIN_RATE_LIMIT_WINDOW="${LOGIN_RATE_LIMIT_WINDOW:-1m}"
export GOCACHE="${GOCACHE:-${GO_CACHE_DIR}}"
export GOTMPDIR="${GOTMPDIR:-${GO_TMP_DIR}}"

exec go run . server --dev-mode --listen-addr "${LISTEN_ADDR}" --data-dir "${DATA_DIR}"
