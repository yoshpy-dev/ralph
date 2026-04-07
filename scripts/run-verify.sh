#!/usr/bin/env sh
set -eu

# HARNESS_VERIFY_MODE controls which checks to run:
#   static — linters, type checks, static analysis only
#   test   — tests only
#   all    — both static and test (default, backward-compatible)
HARNESS_VERIFY_MODE="${HARNESS_VERIFY_MODE:-all}"
export HARNESS_VERIFY_MODE

mkdir -p .harness/state .harness/logs docs/evidence

ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
evidence_file="docs/evidence/verify-$(date -u '+%Y-%m-%d-%H%M%S').log"
status_file=".harness/state/verify-exit-code"

# NOTE: The { } | tee pipeline runs the block in a subshell (POSIX sh).
# Variables set inside (ran_any, status, docs_only) do NOT propagate to the
# outer shell. We use a status_file to pass the exit code back out.
# The while-read loop inside also runs in a sub-subshell, so docs_only is
# communicated via the .harness/state/non_docs_change marker file.
# Do not refactor these to rely on variable propagation across pipes.
{
  ran_any=0
  status=0

  echo "# Verification run"
  echo "- Timestamp: $ts"
  echo "- Mode: $HARNESS_VERIFY_MODE"
  echo ""

  if [ -x ./scripts/verify.local.sh ]; then
    echo "==> Running local verifier"
    ran_any=1
    if ! ./scripts/verify.local.sh; then
      status=1
    fi
  fi

  languages="$(./scripts/detect-languages.sh || true)"
  for lang in $languages; do
    verifier="packs/languages/$lang/verify.sh"
    if [ -x "$verifier" ]; then
      echo "==> Running $lang verifier"
      ran_any=1
      if ! "$verifier"; then
        status=1
      fi
    fi
  done

  changed_files=""
  if command -v git >/dev/null 2>&1; then
    changed_files="$( (git diff --name-only 2>/dev/null; git diff --name-only --cached 2>/dev/null) | sort -u )"
  fi

  docs_only=1
  if [ -n "$changed_files" ]; then
    printf '%s\n' "$changed_files" | while IFS= read -r file; do
      case "$file" in
        ""|docs/*|README.md|AGENTS.md|CLAUDE.md|.claude/*)
          ;;
        *)
          echo "$file" > .harness/state/non_docs_change
          ;;
      esac
    done
    if [ -f .harness/state/non_docs_change ]; then
      docs_only=0
      rm -f .harness/state/non_docs_change
    fi
  fi

  if [ "$ran_any" -eq 0 ]; then
    if [ "$docs_only" -eq 1 ]; then
      echo "No language verifier ran. This appears to be docs or scaffold-level work only."
    else
      echo "No verifier ran for code-like changes."
      echo "Add a real verifier in ./scripts/verify.local.sh or packs/languages/<name>/verify.sh."
      status=2
    fi
  else
    echo ""
    if [ "$status" -eq 0 ]; then
      echo "==> All verifiers passed."
    else
      echo "==> Some verifiers failed."
    fi
  fi

  printf '%s' "$status" > "$status_file"
} 2>&1 | tee "$evidence_file"

echo ""
echo "Evidence saved to: $evidence_file"

# Read exit code from status file
verify_status=0
if [ -f "$status_file" ]; then
  verify_status="$(cat "$status_file")"
  rm -f "$status_file"
fi

# Clear needs-verify marker on success
if [ "$verify_status" = "0" ] && [ -f .harness/state/needs-verify ]; then
  rm -f .harness/state/needs-verify
fi

exit "$verify_status"
