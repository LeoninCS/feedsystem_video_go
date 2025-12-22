#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
cd "$ROOT_DIR"

RUN_DIR="$ROOT_DIR/.run"
mkdir -p "$RUN_DIR"

# Start Redis (optional).
# - If you already run Redis elsewhere, this will detect it (when redis-cli exists) and skip.
REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_CONF="${REDIS_CONF:-}"

start_redis() {
  if ! command -v redis-server >/dev/null 2>&1; then
    echo "[start.sh] redis-server not found; skip starting Redis"
    return 0
  fi

  if command -v redis-cli >/dev/null 2>&1; then
    if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
      echo "[start.sh] Redis already running at $REDIS_HOST:$REDIS_PORT"
      return 0
    fi
  fi

  echo "[start.sh] Starting Redis at $REDIS_HOST:$REDIS_PORT"
  if [ -n "$REDIS_CONF" ]; then
    nohup redis-server "$REDIS_CONF" >"$RUN_DIR/redis.log" 2>&1 &
  else
    nohup redis-server --bind "$REDIS_HOST" --port "$REDIS_PORT" >"$RUN_DIR/redis.log" 2>&1 &
  fi
  echo $! >"$RUN_DIR/redis.pid"
}

start_redis

echo "[start.sh] Starting app"
go run ./cmd
