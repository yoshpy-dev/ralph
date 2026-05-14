#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/ensure-pr-ready.sh [<pr-url-or-number-or-branch>]

If no target is provided, the current git branch is used.
USAGE
}

target="${1:-}"
if [ "${target:-}" = "-h" ] || [ "${target:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ -z "$target" ]; then
  target="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
fi

if [ -z "$target" ]; then
  printf 'ensure-pr-ready: no PR target provided and current branch could not be detected\n' >&2
  exit 1
fi

url="$(gh pr view "$target" --json url --jq '.url')"
is_draft="$(gh pr view "$target" --json isDraft --jq '.isDraft')"

case "$is_draft" in
  true)
    printf 'ensure-pr-ready: PR is draft, marking ready: %s\n' "$url" >&2
    gh pr ready "$target" >/dev/null
    ;;
  false)
    printf 'ensure-pr-ready: PR already ready: %s\n' "$url" >&2
    exit 0
    ;;
  *)
    printf 'ensure-pr-ready: unexpected isDraft value for %s: %s\n' "$target" "$is_draft" >&2
    exit 1
    ;;
esac

is_draft_after="$(gh pr view "$target" --json isDraft --jq '.isDraft')"
if [ "$is_draft_after" != "false" ]; then
  printf 'ensure-pr-ready: PR is still draft after gh pr ready: %s\n' "$url" >&2
  exit 1
fi

printf 'ensure-pr-ready: PR marked ready: %s\n' "$url" >&2
