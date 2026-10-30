#!/usr/bin/env bash
# test-xreview-helpers.sh — behavioural tests for scripts/xreview-helpers.sh,
# the /cross-review-only helpers extracted from the retired driver
# dispatcher script (Ralph Loop execution system removal).
#
# Covers the three surviving functions: pick_reviewer, count_triage_findings,
# detect_base_branch. These test cases are carried over unchanged in intent
# from that dispatcher script's own test suite, Test 5 / Test 7 / Test 14
# (renumbered here); the loop-only functions (run_agent, resolve_phase_model,
# write_model_receipt) and their tests were deleted along with the rest of
# the Ralph Loop execution system (per-slice + multi-worktree driver
# scripts).
#
# detect_base_branch reads [standard].base_branch with `ralph config get` in
# the fixture repository's root, through the copy of scripts/ralph-config.sh
# each fixture commits. A stand-in ralph (tests/lib/fake-ralph.sh) first on
# PATH answers it, so the host's ralph is never asked; the cases with no
# ralph at all put only wrappers of the tools they need on PATH.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

WORK_DIR="$(mktemp -d -t xreview-helpers-XXXXXX)"
trap 'rm -rf "$WORK_DIR"' EXIT

PASS=0
FAIL=0

check() {
  local label="$1"
  shift
  if "$@"; then
    printf '  PASS  %s\n' "$label"
    PASS=$((PASS + 1))
  else
    printf '  FAIL  %s\n' "$label"
    FAIL=$((FAIL + 1))
  fi
}

# shellcheck source=/dev/null
. "$REPO_ROOT/scripts/xreview-helpers.sh"

# ── Test 1: pick_reviewer returns the opposite CLI ──────────────────────
echo
echo "── Test 1: pick_reviewer returns the opposite CLI"

got="$(pick_reviewer claude)"
check "1a. arg=claude -> reviewer=codex (got '$got')" test "$got" = "codex"

got="$(pick_reviewer codex)"
check "1b. arg=codex -> reviewer=claude (got '$got')" test "$got" = "claude"

got="$(RALPH_PRIMARY_CLI=claude pick_reviewer)"
check "1c. env RALPH_PRIMARY_CLI=claude -> reviewer=codex (got '$got')" test "$got" = "codex"

got="$(RALPH_PRIMARY_CLI=codex pick_reviewer)"
check "1d. env RALPH_PRIMARY_CLI=codex -> reviewer=claude (got '$got')" test "$got" = "claude"

got="$(unset RALPH_PRIMARY_CLI; pick_reviewer)"
check "1e. no arg, no env -> reviewer=codex (safe default) (got '$got')" test "$got" = "codex"

got="$(RALPH_PRIMARY_CLI=codex pick_reviewer claude)"
check "1f. explicit arg wins over env (got '$got')" test "$got" = "codex"

# 1g-1i: driver value is case-insensitive (AR-2 fix, cross-review-triage-
# org-runtime-retire-loop.md) -- SKILL.md documents RALPH_PRIMARY_CLI as
# case-insensitive, so an uppercase/mixed-case value must still map to the
# correct opposite reviewer, not fall through to the "unrecognized driver"
# default.
got="$(pick_reviewer CODEX)"
check "1g. arg=CODEX (uppercase) -> reviewer=claude (got '$got')" test "$got" = "claude"

got="$(pick_reviewer Claude)"
check "1h. arg=Claude (mixed case) -> reviewer=codex (got '$got')" test "$got" = "codex"

got="$(RALPH_PRIMARY_CLI=CODEX pick_reviewer)"
check "1i. env RALPH_PRIMARY_CLI=CODEX (uppercase) -> reviewer=claude (got '$got')" test "$got" = "claude"

# ── Test 2: count_triage_findings parser ─────────────────────────────────
echo
echo "── Test 2: count_triage_findings respects table rows, not headings"

# 2a. Empty triage with only the template scaffolding (no findings)
TRIAGE_EMPTY="$WORK_DIR/triage-empty.md"
cat > "$TRIAGE_EMPTY" <<'EOF'
# Cross-review triage report: smoke

- Date: 2026-05-08
- Driver: claude
- Reviewer: codex
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
EOF

got_a="$(count_triage_findings "$TRIAGE_EMPTY" ACTION_REQUIRED)"
got_w="$(count_triage_findings "$TRIAGE_EMPTY" WORTH_CONSIDERING)"
got_d="$(count_triage_findings "$TRIAGE_EMPTY" DISMISSED)"
check "2a-i. clean report -> ACTION_REQUIRED=0 (got '$got_a')" test "$got_a" = "0"
check "2a-ii. clean report -> WORTH_CONSIDERING=0 (got '$got_w')" test "$got_w" = "0"
check "2a-iii. clean report -> DISMISSED=0 (got '$got_d')" test "$got_d" = "0"

# 2b. Real triage with findings — counts via the summary header line
TRIAGE_REAL="$WORK_DIR/triage-real.md"
cat > "$TRIAGE_REAL" <<'EOF'
# Cross-review triage report: example

- Total reviewer findings: 5
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=1, DISMISSED=2

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | foo | bar | a.go |
| 2 | baz | qux | b.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | maybe | optional | c.go |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
| 1 | nope | false-positive | x |
| 2 | nope2 | already-addressed | y |
EOF

check "2b-i. real report -> ACTION_REQUIRED=2"   test "$(count_triage_findings "$TRIAGE_REAL" ACTION_REQUIRED)" = "2"
check "2b-ii. real report -> WORTH_CONSIDERING=1" test "$(count_triage_findings "$TRIAGE_REAL" WORTH_CONSIDERING)" = "1"
check "2b-iii. real report -> DISMISSED=2"        test "$(count_triage_findings "$TRIAGE_REAL" DISMISSED)" = "2"

# 2c. Report missing the summary line (fallback path counts table rows)
TRIAGE_NOSUM="$WORK_DIR/triage-no-summary.md"
cat > "$TRIAGE_NOSUM" <<'EOF'
# Cross-review triage report: legacy

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | only finding | rationale | a.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
EOF

check "2c-i. no-summary fallback -> ACTION_REQUIRED=1" test "$(count_triage_findings "$TRIAGE_NOSUM" ACTION_REQUIRED)" = "1"
check "2c-ii. no-summary fallback -> WORTH_CONSIDERING=0" test "$(count_triage_findings "$TRIAGE_NOSUM" WORTH_CONSIDERING)" = "0"

# 2d. Reviewer prose mentions "ACTION_REQUIRED=2" but no canonical summary
# header — must NOT match the summary path.
TRIAGE_PROSE="$WORK_DIR/triage-prose.md"
cat > "$TRIAGE_PROSE" <<'EOF'
# Cross-review triage report: prose-only

The reviewer pointed out that historically, ACTION_REQUIRED=2 issues
have produced spurious regressions, so this report avoids that header.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
EOF
check "2d. prose mention not picked as summary -> ACTION_REQUIRED=0" test "$(count_triage_findings "$TRIAGE_PROSE" ACTION_REQUIRED)" = "0"

# 2e. Missing file -> 0 (must not error)
check "2e. missing file -> 0" test "$(count_triage_findings "$WORK_DIR/does-not-exist.md" ACTION_REQUIRED)" = "0"

# ── Test 3: detect_base_branch ───────────────────────────────────────────
echo
echo "── Test 3: detect_base_branch — resolution order and gate proof"

# shellcheck source=lib/fake-ralph.sh
. "$REPO_ROOT/tests/lib/fake-ralph.sh"
FAKE_BIN="$WORK_DIR/fake-bin"
fake_ralph_install "$FAKE_BIN"
FAKE_PATH="$FAKE_BIN:$PATH"
# What detect_base_branch and ralph-config.sh run, for the cases with no
# ralph on PATH (exit 127): the host's ralph must not be found.
TOOLS="$WORK_DIR/tools"
mkdir -p "$TOOLS"
for _tool in git mktemp cat sed grep rm; do
  printf '#!/bin/sh\nexec "%s" "$@"\n' "$(command -v "$_tool")" > "$TOOLS/$_tool"
  chmod +x "$TOOLS/$_tool"
done

# base_value <value> -- the stand-in answers standard.base_branch with
# <value> ("" for the empty default), and its log starts empty.
base_value() {
  fake_ralph_reset "$FAKE_BIN"
  fake_ralph_value "$FAKE_BIN" standard.base_branch "$1"
}

# run_dbb <dir> <PATH> -- runs detect_base_branch in <dir> with PATH=<PATH>,
# in a subshell, and sets DBB_OUT, DBB_ERR, and DBB_RC.
run_dbb() {
  DBB_OUT="$(cd "$1" && PATH="$2" && detect_base_branch 2>"$WORK_DIR/dbb.err")"
  DBB_RC=$?
  DBB_ERR="$(cat "$WORK_DIR/dbb.err")"
}

# root_call <dir> -- the stand-in's log line for one read of
# standard.base_branch in <dir>.
root_call() {
  printf '%s|config get standard.base_branch' "$(cd "$1" && pwd -P)"
}

# Helper: create a bare "remote" repo and a local clone with one commit on
# main, which holds scripts/ralph-config.sh as a scaffolded project does.
_make_repo_with_origin() {
  local dir="$1"
  local remote_dir="${dir}-remote"
  mkdir -p "$remote_dir"
  git -C "$remote_dir" init -b main --bare >/dev/null 2>&1 ||
    git -C "$remote_dir" init --bare >/dev/null 2>&1
  mkdir -p "$dir/scripts"
  git -C "$dir" init -b main >/dev/null 2>&1 ||
    git -C "$dir" init >/dev/null 2>&1
  git -C "$dir" config user.email "test@example.com"
  git -C "$dir" config user.name "Ralph Test"
  git -C "$dir" remote add origin "$remote_dir"
  printf 'initial\n' > "$dir/README.md"
  cp "$REPO_ROOT/scripts/ralph-config.sh" "$dir/scripts/ralph-config.sh"
  git -C "$dir" add README.md scripts/ralph-config.sh
  git -C "$dir" commit -m 'chore: initial' >/dev/null 2>&1
  git -C "$dir" push -u origin main >/dev/null 2>&1
  # Ensure origin/HEAD is set (git clone does this automatically; we set it explicitly).
  git -C "$dir" remote set-head origin main >/dev/null 2>&1
}

# 3a: End-to-end gate proof — pushed feature branch tracking itself.
#
# On a feature branch pushed with `git push -u`, the OLD detection
# (rev-parse HEAD@{upstream} | sed 's|origin/||') returns the feature branch
# name, making `git diff <feature>...HEAD` empty — gate silently passes.
# detect_base_branch() must return "main" (via origin/HEAD) so the diff
# against main is NON-EMPTY — gate correctly fires.
echo
echo "  3a: end-to-end gate proof (pushed feature branch)"
_REPO_3A="$WORK_DIR/repo-3a"
_make_repo_with_origin "$_REPO_3A"

git -C "$_REPO_3A" checkout -b feature/add-widget >/dev/null 2>&1
printf 'widget\n' > "$_REPO_3A/widget.sh"
git -C "$_REPO_3A" add widget.sh
git -C "$_REPO_3A" commit -m 'feat: add widget' >/dev/null 2>&1
git -C "$_REPO_3A" push -u origin feature/add-widget >/dev/null 2>&1

# OLD detection: HEAD@{upstream} -> origin/feature/add-widget -> strip "origin/" -> feature/add-widget
_old_base_3a="$(git -C "$_REPO_3A" rev-parse --abbrev-ref 'HEAD@{upstream}' 2>/dev/null | sed 's|origin/||' || true)"
check "3a-i. OLD detection yields feature branch (not main)" \
  test "$_old_base_3a" = "feature/add-widget"

# OLD gate: diff feature/add-widget...HEAD -> empty (comparing branch to itself)
_old_diff_3a=0
git -C "$_REPO_3A" diff "feature/add-widget...HEAD" --quiet 2>/dev/null || _old_diff_3a=$?
check "3a-ii. OLD gate sees EMPTY diff (cross-review would be skipped)" \
  test "$_old_diff_3a" -eq 0

# NEW detection: detect_base_branch() -> main (via origin/HEAD), with
# [standard].base_branch empty (the default).
base_value ""
run_dbb "$_REPO_3A" "$FAKE_PATH"
check "3a-iii. detect_base_branch yields 'main' (not the feature branch) (got '$DBB_OUT', rc $DBB_RC)" \
  test "$DBB_RC:$DBB_OUT" = "0:main"
check "3a-iii. an empty [standard].base_branch is read once, in the root (log '$(fake_ralph_calls "$FAKE_BIN")')" \
  test "$(fake_ralph_calls "$FAKE_BIN")" = "$(root_call "$_REPO_3A")"
check "3a-iii. an empty [standard].base_branch writes nothing on stderr (got '$DBB_ERR')" \
  test -z "$DBB_ERR"

# NEW gate: diff main...HEAD -> non-empty (the widget.sh commit is new)
_new_diff_3a=0
git -C "$_REPO_3A" diff "main...HEAD" --quiet 2>/dev/null || _new_diff_3a=$?
check "3a-iv. NEW gate sees NON-EMPTY diff (cross-review correctly fires)" \
  test "$_new_diff_3a" -ne 0

# 3b: [standard].base_branch wins over origin/HEAD, read in the repository
# root from any directory below it; RALPH_XREVIEW_BASE changes nothing.
echo
echo "  3b: [standard].base_branch wins; RALPH_XREVIEW_BASE changes nothing"
_REPO_3B="$WORK_DIR/repo-3b"
_make_repo_with_origin "$_REPO_3B"
mkdir -p "$_REPO_3B/sub/dir"
base_value develop
run_dbb "$_REPO_3B" "$FAKE_PATH"
check "3b-i. [standard].base_branch=develop wins over origin/HEAD=main (got '$DBB_OUT', rc $DBB_RC)" \
  test "$DBB_RC:$DBB_OUT" = "0:develop"
check "3b-i. a set value writes nothing on stderr (got '$DBB_ERR')" \
  test -z "$DBB_ERR"
fake_ralph_clear_calls "$FAKE_BIN"
run_dbb "$_REPO_3B/sub/dir" "$FAKE_PATH"
check "3b-ii. from a subdirectory, the value is still develop (got '$DBB_OUT', rc $DBB_RC)" \
  test "$DBB_RC:$DBB_OUT" = "0:develop"
check "3b-ii. from a subdirectory, ralph is asked in the repository root (log '$(fake_ralph_calls "$FAKE_BIN")')" \
  test "$(fake_ralph_calls "$FAKE_BIN")" = "$(root_call "$_REPO_3B")"
base_value ""
_got_3b_env="$(cd "$_REPO_3B" && PATH="$FAKE_PATH" && RALPH_XREVIEW_BASE=develop && export RALPH_XREVIEW_BASE && detect_base_branch 2>/dev/null)"
check "3b-iii. RALPH_XREVIEW_BASE=develop with an empty setting -> main from origin/HEAD (got '$_got_3b_env')" \
  test "$_got_3b_env" = "main"
base_value release
_got_3b_both="$(cd "$_REPO_3B" && PATH="$FAKE_PATH" && RALPH_XREVIEW_BASE=develop && export RALPH_XREVIEW_BASE && detect_base_branch 2>/dev/null)"
check "3b-iv. RALPH_XREVIEW_BASE=develop does not override [standard].base_branch=release (got '$_got_3b_both')" \
  test "$_got_3b_both" = "release"

# 3c: origin/HEAD pointing at a non-main default branch is honoured.
echo
echo "  3c: non-main origin/HEAD (develop) is honoured"
_REPO_3C="$WORK_DIR/repo-3c"
_make_repo_with_origin "$_REPO_3C"
git -C "$_REPO_3C" remote set-head origin -d >/dev/null 2>&1 || true
git -C "$_REPO_3C" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/develop >/dev/null 2>&1
base_value ""
run_dbb "$_REPO_3C" "$FAKE_PATH"
check "3c. non-main origin/HEAD -> develop returned (got '$DBB_OUT')" \
  test "$DBB_OUT" = "develop"

# 3c-edge: origin/HEAD pointing at origin/release/1.0 — strip only leading "origin/".
echo
echo "  3c-edge: origin/HEAD -> origin/release/1.0 strips only leading origin/"
_REPO_3CE="$WORK_DIR/repo-3ce"
_make_repo_with_origin "$_REPO_3CE"
git -C "$_REPO_3CE" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/release/1.0 >/dev/null 2>&1
base_value ""
run_dbb "$_REPO_3CE" "$FAKE_PATH"
check "3c-edge. release/1.0 stripped of leading origin/ only (got '$DBB_OUT')" \
  test "$DBB_OUT" = "release/1.0"

# 3d: no origin/HEAD -> falls back to main/master local branch detection.
echo
echo "  3d: no origin/HEAD -> main/master fallback"
_REPO_3D="$WORK_DIR/repo-3d"
mkdir -p "$_REPO_3D/scripts"
git -C "$_REPO_3D" init -b main >/dev/null 2>&1 || git -C "$_REPO_3D" init >/dev/null 2>&1
git -C "$_REPO_3D" config user.email "test@example.com"
git -C "$_REPO_3D" config user.name "Ralph Test"
printf 'hello\n' > "$_REPO_3D/README.md"
cp "$REPO_ROOT/scripts/ralph-config.sh" "$_REPO_3D/scripts/ralph-config.sh"
git -C "$_REPO_3D" add README.md scripts/ralph-config.sh
git -C "$_REPO_3D" commit -m 'chore: initial' >/dev/null 2>&1
base_value ""
run_dbb "$_REPO_3D" "$FAKE_PATH"
check "3d-i. no origin/HEAD + refs/heads/main exists -> main (got '$DBB_OUT')" \
  test "$DBB_OUT" = "main"

# 3d-ii: repo with only master branch falls back to master.
_REPO_3DM="$WORK_DIR/repo-3dm"
mkdir -p "$_REPO_3DM/scripts"
git -C "$_REPO_3DM" init -b master >/dev/null 2>&1 || git -C "$_REPO_3DM" init >/dev/null 2>&1
git -C "$_REPO_3DM" config user.email "test@example.com"
git -C "$_REPO_3DM" config user.name "Ralph Test"
printf 'hello\n' > "$_REPO_3DM/README.md"
cp "$REPO_ROOT/scripts/ralph-config.sh" "$_REPO_3DM/scripts/ralph-config.sh"
git -C "$_REPO_3DM" add README.md scripts/ralph-config.sh
git -C "$_REPO_3DM" commit -m 'chore: initial' >/dev/null 2>&1
base_value ""
run_dbb "$_REPO_3DM" "$FAKE_PATH"
check "3d-ii. no origin/HEAD + no main + refs/heads/master exists -> master (got '$DBB_OUT')" \
  test "$DBB_OUT" = "master"

# 3e: git worktree shares origin/HEAD from the common git dir.
echo
echo "  3e: git worktree resolves origin/HEAD through shared common dir"
_REPO_3E="$WORK_DIR/repo-3e"
_make_repo_with_origin "$_REPO_3E"
git -C "$_REPO_3E" worktree add "$WORK_DIR/wt-3e" -b feat/wt-test >/dev/null 2>&1
base_value ""
run_dbb "$WORK_DIR/wt-3e" "$FAKE_PATH"
check "3e. worktree resolves origin/HEAD -> main via shared common dir (got '$DBB_OUT')" \
  test "$DBB_OUT" = "main"
check "3e. in a worktree, ralph is asked in the worktree's root (log '$(fake_ralph_calls "$FAKE_BIN")')" \
  test "$(fake_ralph_calls "$FAKE_BIN")" = "$(root_call "$WORK_DIR/wt-3e")"

# 3f: no ralph on PATH (exit 127, as in CI): the automatic base, silently.
echo
echo "  3f: no ralph on PATH -> automatic base, silently"
base_value develop
run_dbb "$_REPO_3B" "$TOOLS"
check "3f. no ralph on PATH -> main from origin/HEAD (got '$DBB_OUT', rc $DBB_RC)" \
  test "$DBB_RC:$DBB_OUT" = "0:main"
check "3f. no ralph on PATH writes nothing on stderr (got '$DBB_ERR')" \
  test -z "$DBB_ERR"

# 3g: a ralph older than `config get`: the automatic base, silently.
echo
echo "  3g: an older ralph -> automatic base, silently"
base_value develop
fake_ralph_old "$FAKE_BIN"
run_dbb "$_REPO_3B" "$FAKE_PATH"
check "3g. an older ralph -> main from origin/HEAD (got '$DBB_OUT', rc $DBB_RC)" \
  test "$DBB_RC:$DBB_OUT" = "0:main"
check "3g. an older ralph writes nothing on stderr (got '$DBB_ERR')" \
  test -z "$DBB_ERR"
check "3g. the older ralph was asked in the root (log '$(fake_ralph_calls "$FAKE_BIN")')" \
  test "$(fake_ralph_calls "$FAKE_BIN")" = "$(root_call "$_REPO_3B")"

# 3h: ralph fails on the settings (exit 3: a ralph.local.toml that fails
# validation; exit 2): no automatic base. detect_base_branch prints nothing,
# names the key on stderr, and returns non-zero, although origin/HEAD and
# refs/heads/main both exist.
echo
echo "  3h: a settings error -> failure, not the automatic base"
for _code in 3 2; do
  fake_ralph_reset "$FAKE_BIN"
  fake_ralph_fail "$FAKE_BIN" standard.base_branch "$_code" "config get: ralph.local.toml: plan_visual.browser: not an absolute path"
  run_dbb "$_REPO_3B" "$FAKE_PATH"
  check "3h. exit $_code -> detect_base_branch returns non-zero (rc $DBB_RC)" \
    test "$DBB_RC" -ne 0
  check "3h. exit $_code -> nothing on stdout (got '$DBB_OUT')" \
    test -z "$DBB_OUT"
  case "$DBB_ERR" in
    *"ralph config get standard.base_branch exited $_code"*"[standard].base_branch could not be read from ralph.local.toml; not falling back to the automatic base branch"*)
      check "3h. exit $_code -> stderr names the key, ralph's exit, and no fallback" true ;;
    *) check "3h. exit $_code -> stderr names the key, ralph's exit, and no fallback (got '$DBB_ERR')" false ;;
  esac
done

# 3i: a repository with no scripts/ralph-config.sh (scaffolded before it
# existed): the automatic base, and ralph is not asked.
echo
echo "  3i: no scripts/ralph-config.sh -> automatic base, ralph not asked"
_REPO_3I="$WORK_DIR/repo-3i"
_make_repo_with_origin "$_REPO_3I"
git -C "$_REPO_3I" rm -q scripts/ralph-config.sh
git -C "$_REPO_3I" commit -q -m 'chore: drop ralph-config.sh' >/dev/null 2>&1
fake_ralph_reset "$FAKE_BIN"
fake_ralph_fail "$FAKE_BIN" standard.base_branch 3 "config get: not expected"
run_dbb "$_REPO_3I" "$FAKE_PATH"
check "3i. no scripts/ralph-config.sh -> main from origin/HEAD (got '$DBB_OUT', rc $DBB_RC)" \
  test "$DBB_RC:$DBB_OUT" = "0:main"
check "3i. no scripts/ralph-config.sh -> ralph is not asked (log '$(fake_ralph_calls "$FAKE_BIN")')" \
  test -z "$(fake_ralph_calls "$FAKE_BIN")"

# 3j: outside a git work tree: the last fallback, and ralph is not asked.
echo
echo "  3j: outside a git work tree -> master, ralph not asked"
mkdir -p "$WORK_DIR/nogit/dir"
fake_ralph_clear_calls "$FAKE_BIN"
_got_3j="$(cd "$WORK_DIR/nogit/dir" && PATH="$FAKE_PATH" && GIT_CEILING_DIRECTORIES="$WORK_DIR/nogit" && export GIT_CEILING_DIRECTORIES && detect_base_branch 2>/dev/null)"
check "3j. outside a git work tree -> master (got '$_got_3j')" \
  test "$_got_3j" = "master"
check "3j. outside a git work tree -> ralph is not asked (log '$(fake_ralph_calls "$FAKE_BIN")')" \
  test -z "$(fake_ralph_calls "$FAKE_BIN")"

# 3k: detect_base_branch sources ralph-config.sh in a subshell, so the
# caller's shell gets none of its functions or variables (a caller that only
# wants pick_reviewer sources this file as before).
echo
echo "  3k: the caller's shell gets nothing from ralph-config.sh"
base_value develop
_leak_3k="$(cd "$_REPO_3B" && PATH="$FAKE_PATH" && detect_base_branch >/dev/null 2>&1 &&
  if command -v ralph_setting_or_fail >/dev/null 2>&1 || [ -n "${_rs_default+set}" ]; then echo leaked; else echo clean; fi)"
check "3k. no ralph_setting_or_fail or _rs_ variable after detect_base_branch (got '$_leak_3k')" \
  test "$_leak_3k" = "clean"

echo
echo "── Summary ──"
printf '  PASS: %d\n  FAIL: %d\n' "$PASS" "$FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
