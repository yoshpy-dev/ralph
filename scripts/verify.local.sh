#!/usr/bin/env sh
# verify.local.sh — repo-local static + hook tests, invoked automatically
# by scripts/run-verify.sh.
#
# Honors HARNESS_VERIFY_MODE to match the documented split in
# docs/quality/quality-gates.md:
#   - static: shellcheck, sh -n, jq validity, template sync
#   - test  : hook smoke tests (tests/test-check-mojibake.sh)
#   - all   : everything (default; what run-verify.sh sets)
#
# This file is NOT shipped to scaffolded projects; scaffolded projects
# should write their own verify.local.sh. See scripts/check-sync.sh
# ROOT_ONLY_EXCLUSIONS for the sync policy.

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

mode="${HARNESS_VERIFY_MODE:-all}"
case "$mode" in
  static|test|all) ;;
  *)
    printf 'verify.local.sh: unknown HARNESS_VERIFY_MODE=%s (expected static|test|all)\n' "$mode" >&2
    exit 2
    ;;
esac

status=0

run() {
  label="$1"
  shift
  printf '==> %s\n' "$label"
  if "$@"; then
    printf '    OK\n'
  else
    printf '    FAIL\n'
    status=1
  fi
}

run_static_checks() {
  # 1. Shellcheck on hook and verification shell scripts (if available).
  if command -v shellcheck >/dev/null 2>&1; then
    # Build the argument list via positional parameters so shellcheck
    # does not flag unquoted expansion.
    set --
    for f in .claude/hooks/*.sh templates/base/.claude/hooks/*.sh scripts/verify.local.sh tests/test-check-mojibake.sh; do
      [ -f "$f" ] || continue
      set -- "$@" "$f"
    done
    if [ "$#" -gt 0 ]; then
      run "shellcheck hook + verify scripts" shellcheck "$@"
    fi
  else
    printf '==> shellcheck not installed; skipping (install for stricter checks)\n'
  fi

  # 2. Syntax check every .sh in .claude/hooks/.
  for f in .claude/hooks/*.sh templates/base/.claude/hooks/*.sh; do
    [ -f "$f" ] || continue
    run "sh -n $f" sh -n "$f"
  done

  # 3. JSON validity for settings.json.
  if command -v jq >/dev/null 2>&1; then
    for f in .claude/settings.json templates/base/.claude/settings.json; do
      [ -f "$f" ] || continue
      run "jq -e . $f" jq -e . "$f"
    done
  else
    printf '==> jq not installed; skipping JSON validity check\n'
  fi

  # 4. Template sync.
  if [ -x scripts/check-sync.sh ]; then
    run "scripts/check-sync.sh" scripts/check-sync.sh
  fi
}

run_hook_tests() {
  if [ -x tests/test-check-mojibake.sh ]; then
    run "tests/test-check-mojibake.sh" tests/test-check-mojibake.sh
  fi
}

case "$mode" in
  static) run_static_checks ;;
  test)   run_hook_tests ;;
  all)    run_static_checks; run_hook_tests ;;
esac

exit "$status"
