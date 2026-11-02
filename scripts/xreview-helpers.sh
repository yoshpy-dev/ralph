#!/usr/bin/env sh
# xreview-helpers.sh — driver-agnostic helpers for the /cross-review skill.
#
# Exposes these functions:
#   detect_base_branch
#   pick_reviewer [driver]
#   count_triage_findings <triage_report_path> <category>
#
# Extracted from the retired driver dispatcher script (Ralph Loop
# execution system) when Ralph Loop was removed: /cross-review is the
# consumer of all three functions and survives as part of the standard
# development harness. scripts/secret-scan-branch.sh also sources this
# file, for detect_base_branch alone (base-branch resolution for the
# branch-history secret scan), so a future /cross-review refactor that
# changes detect_base_branch's contract must check that second consumer
# too. The loop-only pieces of that dispatcher script (run_agent,
# resolve_phase_model, write_model_receipt, the codex/claude dispatch
# wrappers) were deleted along with the rest of the Ralph Loop execution
# system — they had no remaining consumer once the Loop's per-slice driver
# script was removed.

# detect_base_branch — print the repo's true merge-target branch name.
#
# Resolution order:
#   1. [standard].base_branch of ralph.local.toml, when it is not empty: read
#      with ralph_setting_or_fail of scripts/ralph-config.sh, which runs
#      `ralph config get standard.base_branch`, both in the root of the
#      repository of the current directory (git rev-parse --show-toplevel).
#      This file is sourced (by the /cross-review skill and by
#      scripts/secret-scan-branch.sh), and POSIX sh gives a sourced file no
#      way to find its own path, so that root stands in for the directory
#      above this file. Outside a git work tree, or in a repository with no
#      scripts/ralph-config.sh (one scaffolded before it existed), there is
#      no setting to read, and this step is skipped silently.
#   2. git symbolic-ref --quiet --short refs/remotes/origin/HEAD with the leading
#      "origin/" stripped — the repo's actual default branch.
#   3. Fallback: "main" if refs/heads/main exists, else "master".
#
# When ralph_setting_or_fail fails in step 1 (ralph config get exits 3 for
# an invalid ralph.local.toml or ralph.toml, for example), this function
# writes one line on stderr after the line ralph_setting_or_fail writes,
# prints nothing, and returns 1 instead of going on to step 2: a base that
# silently changed would change what the review and the secret scan cover.
# /cross-review step 4 stops on it, and secret-scan-branch.sh reports
# "cannot scan" (exit 3 under --strict). With no ralph on PATH, or one older
# than `config get`, ralph_setting_or_fail gives the empty default, so step 2
# follows silently. No environment variable chooses the base.
#
# Note: the /cross-review gate treats a failing diff (invalid base) as "has
# changes" and runs the review — fail-open-to-review is the safe direction
# and is kept.
# No side effects beyond reading git refs and running `ralph config get`:
# ralph-config.sh is sourced in a subshell, so the caller's shell gets no
# function or variable from it.
detect_base_branch() {
  # 1. [standard].base_branch, when it is set; a failure to read it stops here.
  if _dbb_root="$(git rev-parse --show-toplevel 2>/dev/null)" &&
    [ -f "$_dbb_root/scripts/ralph-config.sh" ]; then
    # shellcheck source=ralph-config.sh
    if ! _dbb_setting="$(CDPATH='' cd -- "$_dbb_root" && . ./scripts/ralph-config.sh &&
      ralph_setting_or_fail standard.base_branch)"; then
      printf 'xreview-helpers.sh: [standard].base_branch could not be read (the line above says why; a mistake in any key of ralph.toml or ralph.local.toml makes every key fail); not falling back to the automatic base branch\n' >&2
      return 1
    fi
    if [ -n "$_dbb_setting" ]; then
      printf '%s\n' "$_dbb_setting"
      return 0
    fi
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

# pick_reviewer [driver] — return the *opposite* CLI of the driver, so
# /plan's plan advisory, /cross-review, a feature org leader's plan advisory,
# and a feature org reviewer seat's cross-review always use a different model
# than the one doing the work. Prints "codex" or "claude" on stdout.
#
# Args: $1 (optional) = driver ("claude"|"codex", case-insensitive). When
# omitted, falls back to $RALPH_PRIMARY_CLI, then to "claude" if that is also
# unset. All four callers pass a driver. The first three pass the driver they
# resolved: the CLI running the skill (for a feature org leader, the CLI the
# leader names for itself), which RALPH_PRIMARY_CLI (case-insensitive)
# overrides (see .claude/skills/plan/SKILL.md step 11.b,
# .claude/skills/cross-review/SKILL.md Step 2, and the plan advisory
# subsection of internal/org/prompts/leader.md in the ralph repository
# source). The fourth, a feature org reviewer seat's cross-review, passes the
# implementer's driver from the leader's TASK (IMPLEMENTER_DRIVER:), not its
# own CLI, and relies on this function not reading RALPH_PRIMARY_CLI when a
# driver is given, since the reviewer's pane holds the reviewer's own driver
# (see the cross-review TASK section of internal/org/prompts/reviewer.md in
# the ralph repository source). The driver value is
# normalized to lowercase before the case branch so "CODEX"/"Claude"/etc.
# resolve to the same reviewer as their lowercase form (AR-2 fix) instead of
# silently falling through to the "unrecognized driver" default.
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
