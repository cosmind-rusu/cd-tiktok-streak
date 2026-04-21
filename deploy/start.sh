#!/usr/bin/env sh
set -eu

DATA_DIR="${CDRUSU_DATA_DIR:-/data}"
CONFIG_PATH="${CDRUSU_CONFIG_PATH:-$DATA_DIR/config.json}"
COOKIES_PATH="${CDRUSU_COOKIES_PATH:-$DATA_DIR/cookies.json}"

mkdir -p "$DATA_DIR"
mkdir -p "$(dirname "$CONFIG_PATH")"
mkdir -p "$(dirname "$COOKIES_PATH")"

write_file_from_env() {
  target_path="$1"
  raw_var_name="$2"
  b64_var_name="$3"

  raw_value="$(printenv "$raw_var_name" || true)"
  b64_value="$(printenv "$b64_var_name" || true)"

  if [ -n "$b64_value" ]; then
    printf '%s' "$b64_value" | base64 -d > "$target_path"
    return
  fi

  if [ -n "$raw_value" ]; then
    printf '%s' "$raw_value" > "$target_path"
  fi
}

write_file_from_env "$CONFIG_PATH" "CDRUSU_CONFIG_JSON" "CDRUSU_CONFIG_JSON_B64"
write_file_from_env "$COOKIES_PATH" "CDRUSU_COOKIES_JSON" "CDRUSU_COOKIES_JSON_B64"

if [ ! -f "$CONFIG_PATH" ]; then
  if [ -f "/app/config.example.json" ]; then
    cp /app/config.example.json "$CONFIG_PATH"
    echo "created $CONFIG_PATH from config.example.json; set your values and redeploy" >&2
  else
    echo "missing config file at $CONFIG_PATH and no CDRUSU_CONFIG_JSON(_B64) variable was provided" >&2
    exit 1
  fi
fi

if [ ! -f "$COOKIES_PATH" ]; then
  echo "missing cookies file at $COOKIES_PATH and no CDRUSU_COOKIES_JSON(_B64) variable was provided" >&2
  exit 1
fi

exec cd-tiktok-streak -config "$CONFIG_PATH" "$@"
