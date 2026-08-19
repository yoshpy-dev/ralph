#!/usr/bin/env sh
# xreview-helpers.sh — driver-agnostic helpers for the /cross-review skill.
#
# Exposes these functions:
#   detect_base_branch
#   pick_reviewer [driver]
#   count_triage_findings <triage_report_path> <category>
#
# Extracted from the retired driver dispatcher script (Ralph Loop
# execution system) when Ralph Loop was removed: /cross-review is the only
# consumer of these three functions and it survives as part of the standard
# development harness. The loop-only pieces of that dispatcher script
# (run_agent, resolve_phase_model, write_model_receipt, the codex/claude
# dispatch wrappers) were deleted along with the rest of the Ralph Loop
# execution system — they had no remaining consumer once the Loop's
# per-slice driver script was removed.

# detect_base_branch — print the repo's true merge-target branch name.
#
# Resolution order:
#   1. $RALPH_XREVIEW_BASE — explicit override (exported by the operator).
#   2. git symbolic-ref --quiet --short refs/remotes/origin/HEAD with the leading
#      "origin/" stripped — the repo's actual default branch.
#   3. Fallback: "main" if refs/heads/main exists, else "master".
#
# Note: the /cross-review gate treats a failing diff (invalid base) as "has
# changes" and runs the review — fail-open-to-review is the safe direction
# and is kept.
# Pure function: no side effects beyond reading git refs.
detect_base_branch() {
  # 1. Explicit override wins.
  if [ -n "${RALPH_XREVIEW_BASE:-}" ]; then
    printf '%s\n' "$RALPH_XREVIEW_BASE"
    return 0
  fi

  # 2. Symbolic-ref: repo default branch (correct merge target).
  _dbb_remote_head="$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null || true)"
  if [ -n "$_dbb_remote_head" ]; then
    printf '%s\n' "${_dbb_remote_head#origin/}"
    return 0
  fi

  # 3. Local branch fallback — same main/master semantics as default_branch()
  #    in scripts/ralph-common.sh and scripts/ralph-worktree.sh.
  if git show-ref --verify --quiet refs/heads/main 2>/dev/null; then
    printf 'main\n'
    return 0
  fi
  printf 'master\n'
}

# pick_reviewer [driver] — return the *opposite* CLI of the active driver, so
# /cross-review always uses a different model than the one doing the
# implementation work. Prints "codex" or "claude" on stdout.
#
# Args: $1 (optional) = driver ("claude"|"codex", case-insensitive). When
# omitted, falls back to $RALPH_PRIMARY_CLI, then to "claude" if that is also
# unset — matching the /cross-review auto-detect default (see
# .claude/skills/cross-review/SKILL.md Step 2, which documents
# RALPH_PRIMARY_CLI as case-insensitive). The driver value is normalized to
# lowercase before the case branch so "CODEX"/"Claude"/etc. resolve to the
# same reviewer as their lowercase form (AR-2 fix) instead of silently
# falling through to the "unrecognized driver" default.
pick_reviewer() {
  _pr_driver="${1:-${RALPH_PRIMARY_CLI:-claude}}"
  _pr_driver="$(printf '%s' "$_pr_driver" | tr '[:upper:]' '[:lower:]')"
  case "$_pr_driver" in
    claude) printf 'codex\n'  ;;
    codex)  printf 'claude\n' ;;
    *)      printf 'codex\n'  ;;  # safe fallback for an unrecognized driver value
  esac
}

# count_triage_findings — print the count for one triage category from a
# cross-review-triage report. Prefers the canonical summary header line
# (`After triage: ACTION_REQUIRED=N, ...`) and falls back to counting the
# `|` table rows under each `## <CATEGORY>` heading. Lives here so the
# /cross-review triage parser is testable in isolation: an inline
# `grep -c '<CATEGORY>' "$file"` overcounted the literal headings (#44
# cross-review P1).
#
# Args: $1 = triage report path
#       $2 = category (ACTION_REQUIRED | WORTH_CONSIDERING | DISMISSED)
count_triage_findings() {
  _file="$1"
  _category="$2"
  if [ ! -s "$_file" ]; then
    printf '0\n'
    return 0
  fi
  # Anchor the summary match at the start of a line so a reviewer's prose
  # ("the diff includes ACTION_REQUIRED=2 as an example") cannot trigger the
  # summary path. The triage template (docs/reports/templates/cross-review-triage-report.md)
  # writes the canonical line as `- After triage: ACTION_REQUIRED=N, ...`.
  _summary="$(grep -m1 -E '^[- ]*After triage: ACTION_REQUIRED=[0-9]+' "$_file" 2>/dev/null || true)"
  if [ -n "$_summary" ]; then
    _n="$(printf '%s' "$_summary" | grep -oE "${_category}=[0-9]+" | head -1 | cut -d= -f2)"
    printf '%s\n' "${_n:-0}"
    return 0
  fi
  awk -v cat="## ${_category}" '
    $0 == cat { f = 1; next }
    /^## / { f = 0 }
    f && /^\|/ && !/^\| *# / && !/^\| *-+/ { n++ }
    END { print n+0 }
  ' "$_file" 2>/dev/null || printf '0\n'
}
