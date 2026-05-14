#!/usr/bin/env sh
# commit-msg-guard.sh — git commit-msg hook
# Last line of defense: scans commit messages for leaked secrets.
# Install: ralph init/upgrade installs this into .git/hooks/commit-msg when safe.
# Manual fallback: cp scripts/commit-msg-guard.sh .git/hooks/commit-msg
# Bypass: git commit --no-verify
set -eu

MSG_FILE="${1:?Usage: commit-msg-guard.sh <commit-msg-file>}"

if [ ! -f "$MSG_FILE" ]; then
  echo "commit-msg-guard: message file not found: $MSG_FILE" >&2
  exit 1
fi

msg="$(cat "$MSG_FILE")"

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
"$REPO_ROOT/scripts/secret-scan.sh" --file "$MSG_FILE" "commit message"

# --- Conventional Commit format validation ---
# Extract the first line (subject) of the commit message
subject="$(printf '%s' "$msg" | head -n 1)"

# Allow merge commits and amend commits
case "$subject" in
  Merge\ *|Revert\ *) exit 0 ;;
esac

# Validate conventional commit format: <type>: <description>
# Allowed types: feat, fix, refactor, docs, test, chore, perf, ci, wip
if ! printf '%s' "$subject" | grep -qE '^(feat|fix|refactor|docs|test|chore|perf|ci|wip)(\(.+\))?: .+'; then
  printf '\n=== commit-msg-guard: FORMAT WARNING ===\n' >&2
  printf 'コミットメッセージが Conventional Commits 形式に従っていません。\n' >&2
  printf '期待される形式: <type>: <description>\n' >&2
  printf '許可されるtype: feat, fix, refactor, docs, test, chore, perf, ci, wip\n' >&2
  printf '例: feat: add user authentication\n' >&2
  printf '現在のメッセージ: %s\n\n' "$subject" >&2
  exit 1
fi

exit 0
