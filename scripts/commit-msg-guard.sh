#!/usr/bin/env sh
# commit-msg-guard.sh — git commit-msg hook
# Last line of defense: scans commit messages for leaked secrets.
# Install: cp scripts/commit-msg-guard.sh .git/hooks/commit-msg
# Bypass: git commit --no-verify
set -eu

MSG_FILE="${1:?Usage: commit-msg-guard.sh <commit-msg-file>}"

if [ ! -f "$MSG_FILE" ]; then
  echo "commit-msg-guard: message file not found: $MSG_FILE" >&2
  exit 1
fi

msg="$(cat "$MSG_FILE")"
errors=""

# --- AWS Access Key ---
if printf '%s' "$msg" | grep -qE 'AKIA[0-9A-Z]{16}'; then
  errors="${errors}\n  - AWS Access Key (AKIA...)"
fi

# --- GitHub tokens ---
if printf '%s' "$msg" | grep -qE '(ghp|gho|ghs|ghu|ghr)_[A-Za-z0-9_]{30,}'; then
  errors="${errors}\n  - GitHub token (ghp_/gho_/ghs_/ghu_/ghr_)"
fi

# --- Generic secret patterns (key=value with secret-like key names) ---
if printf '%s' "$msg" | grep -qiE '(api_key|api_secret|secret_key|access_token|auth_token|private_key|client_secret)\s*[=:]\s*\S+'; then
  errors="${errors}\n  - Secret key-value pair (api_key=, secret=, token=, etc.)"
fi

# --- Private key header ---
if printf '%s' "$msg" | grep -qE 'BEGIN [A-Z ]*(PRIVATE KEY|RSA PRIVATE|EC PRIVATE|DSA PRIVATE)'; then
  errors="${errors}\n  - Private key header (BEGIN ... PRIVATE KEY)"
fi

# --- Environment variable dump detection (5+ lines of KEY=value) ---
env_dump_count="$(printf '%s' "$msg" | grep -cE '^[A-Z_][A-Z0-9_]*=.+' || true)"
if [ "$env_dump_count" -ge 5 ]; then
  errors="${errors}\n  - Environment variable dump detected (${env_dump_count} KEY=value lines)"
fi

# --- Abnormally long message warning (>2000 chars) ---
msg_len="$(printf '%s' "$msg" | wc -c | tr -d ' ')"
if [ "$msg_len" -gt 2000 ]; then
  errors="${errors}\n  - Abnormally long commit message (${msg_len} chars > 2000)"
fi

if [ -n "$errors" ]; then
  printf '\n=== commit-msg-guard: BLOCKED ===\n' >&2
  printf 'コミットメッセージにシークレットまたは疑わしいパターンを検出しました:\n' >&2
  printf '%b\n' "$errors" >&2
  printf '\nコミットをブロックしました。メッセージを修正してください。\n' >&2
  printf 'バイパスが必要な場合: git commit --no-verify\n\n' >&2
  exit 1
fi

exit 0
