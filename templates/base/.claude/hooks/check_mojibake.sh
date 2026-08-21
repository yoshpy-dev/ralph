#!/usr/bin/env sh
# check_mojibake.sh — PostToolUse guard against U+FFFD injection.
#
# Background:
#   Claude Code's Write/Edit/MultiEdit tools can split multi-byte characters
#   at SSE chunk boundaries, leaving U+FFFD (replacement character, UTF-8
#   bytes EF BF BD) inside the written file. This hook scans the edited
#   file and, if it finds U+FFFD that is not allowlisted, exits 2 so
#   Claude is prompted to re-read and rewrite the corrupted section.
#
#   Temporary mitigation — remove once Claude Code upstream Issue #43746
#   (and related) is fixed in a released version and we verify no
#   regressions for a week.
#
# Contract:
#   - Reads PostToolUse JSON payload from stdin.
#   - Extracts .tool_input.file_path via jq (jq is required).
#   - If .tool_input.file_path is empty (Codex apply_patch payloads have
#     no file_path — the patch body lives in .tool_input.command instead),
#     derives the touched-file list from that patch body's "*** Add File:",
#     "*** Update File:", and "*** Move to:" envelope lines. "*** Delete
#     File:" targets are intentionally excluded — the file no longer
#     exists on disk after the patch, so there is nothing left to scan.
#   - apply_patch envelope paths are relative to the Codex SESSION's cwd
#     (the payload's top-level "cwd" field), not to this hook's own cwd
#     (the git root, since ralph-dispatch.sh's callers cd there first — see
#     .codex/hooks.json). A relative envelope path is resolved against the
#     payload's "cwd" before the existence test below; an absolute envelope
#     path passes through unchanged, and a missing/empty "cwd" falls back
#     to resolving against this hook's own cwd (the pre-fix behavior).
#   - Scans every derived path that exists on disk; the first one carrying
#     an unallowlisted U+FFFD exits 2 (first violation wins, matching the
#     single-file contract below).
#   - If jq is missing, warn to stderr, write marker, and exit 0
#     (fail-open-with-warning — we do not want to block every edit in
#     minimal environments).
#   - If the payload is malformed or yields no path to scan, exit 0
#     (quiet no-op — nothing to scan).
#   - If a scanned file does not exist, is empty, or has no U+FFFD, it is
#     skipped without affecting the others.
#   - If a scanned file matches a glob in .claude/hooks/mojibake-allowlist,
#     it is skipped even when U+FFFD is present.
#   - Otherwise print an actionable message to stderr and exit 2.
#
# Environment:
#   HOOK_REPO_ROOT can be set to override the repo root used for
#   allowlist lookup and relative-path matching (used by tests).

set -eu

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="${HOOK_REPO_ROOT:-$(cd "$HOOK_DIR/../.." && pwd)}"

payload="$(cat)"

if ! command -v jq >/dev/null 2>&1; then
  mkdir -p "$REPO_ROOT/.harness/state" 2>/dev/null || true
  : > "$REPO_ROOT/.harness/state/mojibake-jq-missing" 2>/dev/null || true
  printf 'check_mojibake.sh: jq not found; skipping U+FFFD scan (install jq to enable).\n' >&2
  exit 0
fi

file_path="$(printf '%s' "$payload" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)"
session_cwd="$(printf '%s' "$payload" | jq -r '.cwd // empty' 2>/dev/null || true)"

scan_paths=""
if [ -n "$file_path" ]; then
  scan_paths="$file_path"
else
  patch_text="$(printf '%s' "$payload" | jq -r '.tool_input.command // empty' 2>/dev/null || true)"
  if [ -n "$patch_text" ]; then
    scan_paths="$(printf '%s\n' "$patch_text" | sed -n \
      -e 's/^\*\*\* Add File: \(.*\)$/\1/p' \
      -e 's/^\*\*\* Update File: \(.*\)$/\1/p' \
      -e 's/^\*\*\* Move to: \(.*\)$/\1/p')"
  fi
fi

if [ -z "$scan_paths" ]; then
  exit 0
fi

allowlist_file="$REPO_ROOT/.claude/hooks/mojibake-allowlist"

# Build the U+FFFD byte sequence at runtime so the source file itself
# contains only ASCII (prevents the hook from flagging its own source).
FFFD="$(printf '\357\277\275')"

# is_allowlisted <file_path> — matches against both the repo-relative and
# absolute forms, same as the single-file version this replaced.
is_allowlisted() {
  _fp="$1"
  [ -f "$allowlist_file" ] || return 1
  _rel="$_fp"
  case "$_fp" in
    "$REPO_ROOT"/*) _rel="${_fp#"$REPO_ROOT"/}" ;;
  esac
  while IFS= read -r pattern || [ -n "$pattern" ]; do
    # Skip blank lines and comments.
    case "$pattern" in
      ''|\#*) continue ;;
    esac
    # Shell case-glob matches *, ?, [ ] but not **; normalise ** to *.
    normalised="$(printf '%s' "$pattern" | sed 's#\*\*#*#g')"
    # shellcheck disable=SC2254  # intentional glob match
    case "$_rel" in
      $normalised) return 0 ;;
    esac
    # shellcheck disable=SC2254
    case "$_fp" in
      $normalised) return 0 ;;
    esac
  done < "$allowlist_file"
  return 1
}

while IFS= read -r fp; do
  [ -z "$fp" ] && continue
  # Resolve a relative apply_patch envelope path against the session cwd
  # (see the Contract note above); an absolute path passes through
  # unchanged, and an empty session_cwd leaves it relative (pre-fix
  # fallback, evaluated against this hook's own cwd by the [ -f ] below).
  case "$fp" in
    /*) : ;;
    *)
      if [ -n "$session_cwd" ]; then
        fp="$session_cwd/$fp"
      fi
      ;;
  esac
  [ -f "$fp" ] || continue
  if is_allowlisted "$fp"; then
    continue
  fi
  if LC_ALL=C grep -q "$FFFD" "$fp" 2>/dev/null; then
    printf 'check_mojibake.sh: U+FFFD detected in %s. Re-read the file and rewrite the corrupted section without the replacement character.\n' "$fp" >&2
    exit 2
  fi
done <<EOF
$scan_paths
EOF

exit 0
