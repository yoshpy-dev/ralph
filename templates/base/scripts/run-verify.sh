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
status_file=".harness/state/verify-exit-code"
scope_file=".harness/state/verify-scope"

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
      scope_tmp="$(mktemp "${TMPDIR:-/tmp}/ralph-verify-scope.XXXXXX")"
      if ./scripts/detect-changed-languages.sh > "$scope_tmp"; then
        cp "$scope_tmp" "$scope_file"
        selected_scope="$(sed -n 's/^scope=//p' "$scope_tmp" | sed -n '1p')"
        scope_reason="$(sed -n 's/^reason=//p' "$scope_tmp" | sed -n '1p')"
        scope_docs_only="$(sed -n 's/^docs_only=//p' "$scope_tmp" | sed -n '1p')"
        languages="$(sed -n 's/^languages=//p' "$scope_tmp" | sed -n '1p')"
        rm -f "$scope_tmp"
      else
        rm -f "$scope_tmp"
        selected_scope="full"
        scope_reason="changed_detector_failed"
        scope_docs_only="false"
        languages=""
        {
          echo "scope=full"
          echo "reason=$scope_reason"
          echo "docs_only=false"
          echo "languages="
        } > "$scope_file"
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
      } > "$scope_file"
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
        } > "$scope_file"
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
        } > "$scope_file"
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
    } > "$scope_file"
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
      if [ -f "$scope_file" ]; then
        lang_roots="$(sed -n "s/^${lang}_roots=//p" "$scope_file" | sed -n '1p')"
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
