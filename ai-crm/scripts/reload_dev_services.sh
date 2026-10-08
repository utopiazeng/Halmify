#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKEND_DIR="$ROOT_DIR/ai-crm"
FRONTEND_DIR="$ROOT_DIR/ai-crm-frontend"
FILESERVER_DIR="$ROOT_DIR/fileserver"
LOG_DIR="$ROOT_DIR/.scratch/dev-services"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"
FILESERVER_LOG="$LOG_DIR/fileserver.log"
BACKEND_BIN="$LOG_DIR/ai-crm-dev"
FILESERVER_BIN="$LOG_DIR/fileserver-dev"
BACKEND_PID_FILE="$LOG_DIR/backend.pid"
FRONTEND_PID_FILE="$LOG_DIR/frontend.pid"
FILESERVER_PID_FILE="$LOG_DIR/fileserver.pid"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-3000}"
FILESERVER_PORT="${FILESERVER_PORT:-18080}"
RELOAD_FILESERVER="${RELOAD_FILESERVER:-0}"
FRONTEND_URL=""

mkdir -p "$LOG_DIR"

port_pids() {
  local port="$1"
  ss -ltnp "( sport = :$port )" 2>/dev/null |
    sed -n 's/.*pid=\([0-9]\+\).*/\1/p' |
    sort -u
}

port_busy() {
  local port="$1"
  ss -ltn "( sport = :$port )" 2>/dev/null | awk 'NR > 1 { found = 1 } END { exit found ? 0 : 1 }'
}

stop_port() {
  local port="$1"
  local pids
  pids="$(port_pids "$port" || true)"
  if [[ -z "$pids" ]]; then
    return
  fi
  while read -r pid; do
    [[ -z "$pid" ]] && continue
    local owner
    owner="$(ps -o user= -p "$pid" 2>/dev/null | tr -d ' ')"
    if [[ "$owner" == "$(id -un)" ]]; then
      kill "$pid" 2>/dev/null || true
    else
      echo "port $port is held by pid $pid owned by $owner; cannot stop it as $(id -un)"
    fi
  done <<< "$pids"
  sleep 1
}

stop_pid_file() {
  local pid_file="$1"
  [[ -f "$pid_file" ]] || return 0

  local pid
  pid="$(cat "$pid_file" 2>/dev/null || true)"
  rm -f "$pid_file"
  [[ -n "$pid" ]] || return 0

  local owner
  owner="$(ps -o user= -p "$pid" 2>/dev/null | tr -d ' ')"
  if [[ "$owner" == "$(id -un)" ]]; then
    kill "$pid" 2>/dev/null || true
    sleep 1
  fi
}

stop_project_frontends() {
  local escaped_dir
  escaped_dir="$(printf '%s\n' "$FRONTEND_DIR" | sed 's/[.[\*^$()+?{}|]/\\&/g')"

  ps -u "$(id -un)" -o pid=,command= |
    awk -v dir="$escaped_dir" '$0 ~ dir && $0 ~ /vite/ { print $1 }' |
    while read -r pid; do
      [[ -n "$pid" ]] || continue
      kill "$pid" 2>/dev/null || true
    done || true
  sleep 1
}

find_free_port() {
  local port="$1"
  while port_busy "$port"; do
    port=$((port + 1))
  done
  echo "$port"
}

append_env_if_set() {
  local -n target="$1"
  local key="$2"
  if [[ -n "${!key:-}" ]]; then
    target+=("$key=${!key}")
  fi
}

config_value() {
  local file="$1"
  local key="$2"
  awk -F= -v key="$key" '
    $1 ~ "^[[:space:]]*" key "[[:space:]]*$" {
      value = $2
      sub(/^[[:space:]]+/, "", value)
      sub(/[[:space:]]+$/, "", value)
      print value
      exit
    }
  ' "$file" 2>/dev/null || true
}

mysql_database_exists() {
  local user="$1"
  local password="$2"
  local database="$3"
  local socket="${DB_SOCKET:-/var/run/mysqld/mysqld.sock}"
  [[ "$database" =~ ^[A-Za-z0-9_]+$ ]] || return 1
  command -v mysql >/dev/null 2>&1 || return 1

  local args=(--protocol=socket "-u${user}")
  if [[ -n "$password" ]]; then
    args+=("-p${password}")
  fi
  if [[ -S "$socket" ]]; then
    args+=("-S" "$socket")
  fi
  mysql "${args[@]}" -Nse "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = '${database}'" 2>/dev/null |
    grep -qx "$database"
}

apply_backend_database_fallback() {
  if [[ -n "${DB_NAME:-}" ]]; then
    return
  fi

  local config="$BACKEND_DIR/conf/app.conf"
  local configured_db configured_user configured_password
  configured_db="$(config_value "$config" db_name)"
  configured_user="$(config_value "$config" db_user)"
  configured_password="$(config_value "$config" db_password)"
  configured_user="${DB_USER:-${configured_user:-root}}"
  configured_password="${DB_PASSWORD:-$configured_password}"

  if [[ -n "$configured_db" ]] && mysql_database_exists "$configured_user" "$configured_password" "$configured_db"; then
    return
  fi
  if mysql_database_exists "$configured_user" "$configured_password" "ai_crm"; then
    DB_NAME="ai_crm"
  fi
}

wait_http() {
  local url="$1"
  local name="$2"
  for _ in $(seq 1 40); do
    local code
    code="$(curl -sS -o /dev/null -w '%{http_code}' "$url" 2>/dev/null || true)"
    if [[ "$code" != "000" ]]; then
      echo "$name ready: $url"
      return
    fi
    sleep 0.5
  done
  echo "$name failed to become ready: $url" >&2
  return 1
}

stop_pid_file "$BACKEND_PID_FILE"
stop_pid_file "$FRONTEND_PID_FILE"
stop_project_frontends
stop_port "$BACKEND_PORT"
stop_port "$FRONTEND_PORT"

if [[ "$RELOAD_FILESERVER" == "1" ]]; then
  stop_pid_file "$FILESERVER_PID_FILE"
  stop_port "$FILESERVER_PORT"

  if port_busy "$FILESERVER_PORT"; then
    echo "fileserver port $FILESERVER_PORT is still occupied; not starting another fileserver"
  else
    (cd "$FILESERVER_DIR" && go build -o "$FILESERVER_BIN" ./cmd/server)
    fileserver_env=(
      FILESERVER_ADDR=":${FILESERVER_PORT}"
      FILESERVER_STORAGE_ROOT="${FILESERVER_STORAGE_ROOT:-$ROOT_DIR/.scratch/fileserver-storage}"
    )
    for key in \
      FILESERVER_CONFIG \
      FILESERVER_DB_HOST \
      FILESERVER_DB_PORT \
      FILESERVER_DB_USER \
      FILESERVER_DB_PASSWORD \
      FILESERVER_DB_NAME \
      FILESERVER_DB_PARAMS \
      FILESERVER_DB_SOCKET \
      FILESERVER_REDIS_HOST \
      FILESERVER_REDIS_PORT \
      FILESERVER_REDIS_PASSWORD \
      FILESERVER_REDIS_DB \
      FILESERVER_APP_ID \
      FILESERVER_APP_SECRET \
      FILESERVER_MAX_UPLOAD_BYTES
    do
      append_env_if_set fileserver_env "$key"
    done
    setsid -f bash -c 'cd "$1" && shift && exec "$@"' bash "$FILESERVER_DIR" env \
      "${fileserver_env[@]}" \
      "$FILESERVER_BIN" >"$FILESERVER_LOG" 2>&1 &
  fi
else
  echo "fileserver reload skipped; set RELOAD_FILESERVER=1 to restart it"
fi

if port_busy "$BACKEND_PORT"; then
  echo "backend port $BACKEND_PORT is still occupied; not starting another backend"
else
  (cd "$BACKEND_DIR" && go build -o "$BACKEND_BIN" .)
  apply_backend_database_fallback
  backend_env=()
  for key in \
    DB_HOST \
    DB_PORT \
    DB_USER \
    DB_PASSWORD \
    DB_NAME \
    DB_SOCKET \
    DB_RUN_MIGRATIONS \
    DB_MIGRATION_LOCK_NAME \
    DB_MIGRATION_LOCK_TIMEOUT_SEC \
    REDIS_ADDR \
    REDIS_PASSWORD \
    REDIS_DB \
    FILESERVER_NODES \
    FILESERVER_APP_ID \
    FILESERVER_APP_SECRET \
    FILESERVER_DISK_PROTECTION_ENABLED \
    FILESERVER_MIN_FREE_BYTES \
    FILESERVER_HIGH_WATERMARK_PERCENT \
    FILESERVER_UPLOAD_MAX_FILE_BYTES \
    AI_API_KEY \
    DEEPSEEK_API_KEY
  do
    append_env_if_set backend_env "$key"
  done
  setsid -f bash -c 'cd "$1" && shift && exec "$@"' bash "$BACKEND_DIR" env \
    "${backend_env[@]}" \
    "$BACKEND_BIN" >"$BACKEND_LOG" 2>&1 &
fi

if port_busy "$FRONTEND_PORT"; then
  next_frontend_port="$(find_free_port "$((FRONTEND_PORT + 1))")"
  echo "frontend port $FRONTEND_PORT is still occupied; starting frontend on $next_frontend_port"
  FRONTEND_PORT="$next_frontend_port"
  setsid -f bash -c 'cd "$1" && npm run dev -- --host 0.0.0.0 --port "$2"' bash "$FRONTEND_DIR" "$FRONTEND_PORT" >"$FRONTEND_LOG" 2>&1
else
  setsid -f bash -c 'cd "$1" && npm run dev -- --host 0.0.0.0 --port "$2"' bash "$FRONTEND_DIR" "$FRONTEND_PORT" >"$FRONTEND_LOG" 2>&1
fi

FRONTEND_URL="http://127.0.0.1:$FRONTEND_PORT/"

wait_http "http://127.0.0.1:$BACKEND_PORT/api/v1/auth/me" "backend" || {
  tail -n 80 "$BACKEND_LOG" >&2 || true
  exit 1
}
port_pids "$BACKEND_PORT" | head -n 1 >"$BACKEND_PID_FILE"
if [[ "$RELOAD_FILESERVER" == "1" ]]; then
  wait_http "http://127.0.0.1:$FILESERVER_PORT/internal/v1/health" "fileserver" || {
    tail -n 80 "$FILESERVER_LOG" >&2 || true
    exit 1
  }
  port_pids "$FILESERVER_PORT" | head -n 1 >"$FILESERVER_PID_FILE"
fi
wait_http "$FRONTEND_URL" "frontend" || {
  tail -n 80 "$FRONTEND_LOG" >&2 || true
  exit 1
}
port_pids "$FRONTEND_PORT" | head -n 1 >"$FRONTEND_PID_FILE"

echo "logs:"
echo "  backend:  $BACKEND_LOG"
echo "  frontend: $FRONTEND_LOG"
echo "  fileserver: $FILESERVER_LOG"
echo "urls:"
echo "  backend:  http://127.0.0.1:$BACKEND_PORT"
echo "  frontend: $FRONTEND_URL"
echo "  fileserver: http://127.0.0.1:$FILESERVER_PORT"
