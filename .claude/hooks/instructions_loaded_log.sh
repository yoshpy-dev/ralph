#!/usr/bin/env sh
set -eu

payload="$(cat | tr '\n' ' ')"
file_path="$(printf '%s' "$payload" | sed -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
load_reason="$(printf '%s' "$payload" | sed -n 's/.*"load_reason"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

mkdir -p .harness/logs
printf '%s\t%s\n' "$load_reason" "$file_path" >> .harness/logs/instructions-loaded.log

exit 0
