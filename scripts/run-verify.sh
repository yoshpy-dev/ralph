#!/usr/bin/env sh
set -eu

# HARNESS_VERIFY_MODE controls which checks to run:
#   static — linters, type checks, static analysis only
#   test   — tests only
#   all    — both static and test (default, backward-compatible)
HARNESS_VERIFY_MODE="${HARNESS_VERIFY_MODE:-all}"
export HARNESS_VERIFY_MODE

# RALPH_VERIFY_SCOPE controls which language packs to run:
#   full    — all languages detected in the repository (default)
#   changed — only language packs affected by the current git diff, with
#             conservative fallback to full for shared or ambiguous changes
RALPH_VERIFY_SCOPE="${RALPH_VERIFY_SCOPE:-full}"
case "$RALPH_VERIFY_SCOPE" in
  full|changed) ;;
  *)
    echo "run-verify.sh: unknown RALPH_VERIFY_SCOPE=$RALPH_VERIFY_SCOPE (expected full|changed)" >&2
    exit 2
    ;;
esac
export RALPH_VERIFY_SCOPE

mkdir -p .harness/state .harness/logs docs/evidence

ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
evidence_file="docs/evidence/verify-$(date -u '+%Y-%m-%d-%H%M%S').log"
# Left behind at the end of the run for readers; the run itself never reads it.
scope_file=".harness/state/verify-scope"

# Per-run handoff files. Two runs in the same worktree (a background gate and
# a reviewer's run, say) must not see each other's exit code, scope, or
# docs-only marker, so each run gets its own temp files instead of shared
# paths under .harness/state/. The EXIT trap removes them; an EXIT trap does
# not change the exit status (POSIX keeps $? across a trap action). The HUP,
# INT, and TERM traps exit with 128 + the signal number, which runs the EXIT
# trap, but sh runs a trap only after the foreground command it is waiting
# for ends. A signal sent to the whole process group (what `ralph org gate`
# and Ctrl-C in a terminal send) also ends the { } | tee pipeline below, so
# the run exits at once. A signal sent to this script's pid alone runs the
# trap only after that pipeline has finished, verifiers included.
run_status_file=""
run_scope_file=""
run_non_docs_file=""
scope_publish_tmp=""
# shellcheck disable=SC2329  # run by the EXIT trap; shellcheck 0.11 misses it when a script ends in exit
cleanup_run_files() {
  for run_file in "$run_status_file" "$run_scope_file" "$run_non_docs_file" "$scope_publish_tmp"; do
    if [ -n "$run_file" ]; then
      rm -f "$run_file" 2>/dev/null || true
    fi
  done
}
trap cleanup_run_files EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
run_status_file="$(mktemp "${TMPDIR:-/tmp}/ralph-verify-status.XXXXXX")"
run_scope_file="$(mktemp "${TMPDIR:-/tmp}/ralph-verify-scope.XXXXXX")"
run_non_docs_file="$(mktemp "${TMPDIR:-/tmp}/ralph-verify-non-docs.XXXXXX")"

# NOTE: The { } | tee pipeline runs the block in a subshell (POSIX sh).
# Variables set inside (ran_any, status, docs_only) do NOT propagate to the
# outer shell. The block writes its exit code to this run's $run_status_file
# as its last step, and the outer shell reads it back; a missing, empty, or
# non-numeric value counts as a failure, never as success. The scope goes to
# this run's $run_scope_file (the block reads ${lang}_roots back from it) and
# is copied to .harness/state/verify-scope after the block.
# The while-read loop inside also runs in a sub-subshell, so docs_only is
# communicated via this run's $run_non_docs_file marker (non-empty = a
# non-docs change).
# Do not refactor these to rely on variable propagation across pipes, or back
# to shared paths under .harness/state/: a second run in the same worktree
# would overwrite or delete this run's files.
{
  ran_any=0
  status=0

  echo "# Verification run"
  echo "- Timestamp: $ts"
  echo "- Mode: $HARNESS_VERIFY_MODE"
  echo "- Requested scope: $RALPH_VERIFY_SCOPE"
  echo ""

  if [ -x ./scripts/verify.local.sh ]; then
    echo "==> Running local verifier"
    ran_any=1
    if ! ./scripts/verify.local.sh; then
      status=1
    fi
  fi

  scope_docs_only=""
  if [ "$RALPH_VERIFY_SCOPE" = "changed" ]; then
    if [ -x ./scripts/detect-changed-languages.sh ]; then
      if ./scripts/detect-changed-languages.sh > "$run_scope_file"; then
        selected_scope="$(sed -n 's/^scope=//p' "$run_scope_file" | sed -n '1p')"
        scope_reason="$(sed -n 's/^reason=//p' "$run_scope_file" | sed -n '1p')"
        scope_docs_only="$(sed -n 's/^docs_only=//p' "$run_scope_file" | sed -n '1p')"
        languages="$(sed -n 's/^languages=//p' "$run_scope_file" | sed -n '1p')"
      else
        selected_scope="full"
        scope_reason="changed_detector_failed"
        scope_docs_only="false"
        languages=""
        {
          echo "scope=full"
          echo "reason=$scope_reason"
          echo "docs_only=false"
          echo "languages="
        } > "$run_scope_file"
      fi
    else
      selected_scope="full"
      scope_reason="changed_detector_missing"
      scope_docs_only="false"
      languages=""
      {
        echo "scope=full"
        echo "reason=$scope_reason"
        echo "docs_only=false"
        echo "languages="
      } > "$run_scope_file"
    fi

    case "$selected_scope" in
      changed)
        echo "==> Language scope: changed ($scope_reason)"
        ;;
      full)
        echo "==> Language scope: full fallback ($scope_reason)"
        languages="$(./scripts/detect-languages.sh || true)"
        {
          echo "scope=full"
          echo "reason=$scope_reason"
          echo "docs_only=$scope_docs_only"
          echo "languages=$languages"
        } > "$run_scope_file"
        ;;
      *)
        echo "==> Language scope: full fallback (invalid_detector_output)"
        scope_reason="invalid_detector_output"
        scope_docs_only="false"
        languages="$(./scripts/detect-languages.sh || true)"
        {
          echo "scope=full"
          echo "reason=$scope_reason"
          echo "docs_only=false"
          echo "languages=$languages"
        } > "$run_scope_file"
        ;;
    esac
  else
    echo "==> Language scope: full"
    languages="$(./scripts/detect-languages.sh || true)"
    {
      echo "scope=full"
      echo "reason=requested_full"
      echo "docs_only=false"
      echo "languages=$languages"
    } > "$run_scope_file"
  fi

  if [ -n "$languages" ]; then
    echo "==> Language packs selected: $languages"
  else
    echo "==> Language packs selected: none"
  fi

  for lang in $languages; do
    verifier="packs/languages/$lang/verify.sh"
    if [ -x "$verifier" ]; then
      echo "==> Running $lang verifier"
      ran_any=1
      lang_roots=""
      if [ -f "$run_scope_file" ]; then
        lang_roots="$(sed -n "s/^${lang}_roots=//p" "$run_scope_file" | sed -n '1p')"
      fi
      if [ -n "$lang_roots" ]; then
        echo "==> ${lang} project roots selected: $lang_roots"
        if ! RALPH_VERIFY_PROJECT_ROOTS="$lang_roots" "$verifier"; then
          status=1
        fi
      elif ! "$verifier"; then
        status=1
      fi
    fi
  done

  # .ralph/local/ drop-in extension points, run after core verification.
  # HARNESS_VERIFY_MODE selects which local dir(s) apply: static ->
  # verify.d, test -> test.d, all (the run-verify.sh default) -> both.
  case "$HARNESS_VERIFY_MODE" in
    static) local_drop_in_dirs=".ralph/local/verify.d" ;;
    test) local_drop_in_dirs=".ralph/local/test.d" ;;
    *) local_drop_in_dirs=".ralph/local/verify.d .ralph/local/test.d" ;;
  esac

  for local_drop_in_dir in $local_drop_in_dirs; do
    if [ -d "$local_drop_in_dir" ]; then
      for local_drop_in in "$local_drop_in_dir"/*.sh; do
        [ -e "$local_drop_in" ] || continue
        [ -x "$local_drop_in" ] || continue
        echo "==> Running local drop-in: $local_drop_in"
        ran_any=1
        if ! "$local_drop_in"; then
          status=1
        fi
      done
    fi
  done

  changed_files=""
  if command -v git >/dev/null 2>&1; then
    changed_files="$( (git diff --name-only 2>/dev/null; git diff --name-only --cached 2>/dev/null) | sort -u )"
  fi

  docs_only=1
  if [ -n "$scope_docs_only" ]; then
    case "$scope_docs_only" in
      true) docs_only=1 ;;
      false) docs_only=0 ;;
    esac
  elif [ -n "$changed_files" ]; then
    printf '%s\n' "$changed_files" | while IFS= read -r file; do
      case "$file" in
        ""|docs/*|README.md|AGENTS.md|CLAUDE.md|.claude/*)
          ;;
        *)
          echo "$file" > "$run_non_docs_file"
          ;;
      esac
    done
    if [ -s "$run_non_docs_file" ]; then
      docs_only=0
    fi
  fi

  # Branch-history secret scan: mirrors the CI step
  # (.github/workflows/verify.yml) so a leak surfaces here, before push,
  # instead of only after CI reads the pushed history. Runs in static and
  # all modes (not test mode, to keep run-test.sh behavioral-only); does
  # not count toward ran_any -- it is not a language verifier.
  branch_scan_failed=0
  case "$HARNESS_VERIFY_MODE" in
    static|all)
      if [ "${RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN:-}" = "1" ]; then
        echo "==> Skipping branch secret scan (RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1)"
      elif [ -x ./scripts/secret-scan-branch.sh ]; then
        echo "==> Running branch secret scan"
        if ! ./scripts/secret-scan-branch.sh; then
          status=1
          branch_scan_failed=1
        fi
      else
        echo "==> Branch secret scan skipped: ./scripts/secret-scan-branch.sh missing or not executable"
      fi
      ;;
  esac

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

  # branch_scan_failed is tracked separately from status: the docs_only=0
  # arm above overwrites status with 2 ("no verifier for code-like
  # changes"), which would otherwise erase the fact that the branch scan
  # itself also failed. Print this unconditionally, independent of
  # ran_any/docs_only, so a docs-only-looking run with a real finding never
  # closes on a purely reassuring summary.
  if [ "$branch_scan_failed" -eq 1 ]; then
    echo "==> Branch secret scan failed: see \"secret-scan-branch:\" above."
  fi

  printf '%s' "$status" > "$run_status_file"
} 2>&1 | tee "$evidence_file"

echo ""
echo "Evidence saved to: $evidence_file"

# Prune old evidence logs: keep newest 20 verify-*.log files.
# Use a temp file instead of process substitution (POSIX sh compatibility).
if [ -d "docs/evidence" ]; then
  ev_list_tmp="$(mktemp "${TMPDIR:-/tmp}/ralph-ev-prune.XXXXXX")"
  find docs/evidence -maxdepth 1 -name "verify-*.log" -type f 2>/dev/null | sort -r > "$ev_list_tmp"
  ev_count=0
  while IFS= read -r ev_file; do
    ev_count=$((ev_count + 1))
    if [ "$ev_count" -gt 20 ]; then
      rm -f "$ev_file"
    fi
  done < "$ev_list_tmp"
  rm -f "$ev_list_tmp"
fi

# Leave this run's scope behind for readers of .harness/state/verify-scope.
# Write it under a per-run name in .harness/state/ and rename it into place,
# so a reader never sees a half-written file. A block that died before
# writing the scope leaves the previous file as it was.
if [ -s "$run_scope_file" ]; then
  scope_publish_tmp=".harness/state/verify-scope.$$.tmp"
  cat "$run_scope_file" > "$scope_publish_tmp"
  mv -f "$scope_publish_tmp" "$scope_file"
  scope_publish_tmp=""
fi

# Read this run's exit code from its status file. The block writes it last,
# so a missing, unreadable, empty, or non-numeric value means the block did
# not finish (killed, or a command failed under set -e): a failure, never a
# success. Only a plain exit status 0-255 is taken as is.
verify_status=""
if [ -r "$run_status_file" ]; then
  verify_status="$(cat "$run_status_file" 2>/dev/null || true)"
fi
case "$verify_status" in
  0|[1-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5]) ;;
  *)
    echo "run-verify.sh: could not read this run's exit code; treating it as a failure" >&2
    verify_status=1
    ;;
esac

# Clear needs-verify marker on success
if [ "$verify_status" = "0" ] && [ -f .harness/state/needs-verify ]; then
  rm -f .harness/state/needs-verify
fi

exit "$verify_status"
