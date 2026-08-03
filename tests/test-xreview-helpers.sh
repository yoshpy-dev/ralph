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

# Helper: create a bare "remote" repo and a local clone with one commit on main.
_make_repo_with_origin() {
  local dir="$1"
  local remote_dir="${dir}-remote"
  mkdir -p "$remote_dir"
  git -C "$remote_dir" init -b main --bare >/dev/null 2>&1 ||
    git -C "$remote_dir" init --bare >/dev/null 2>&1
  mkdir -p "$dir"
  git -C "$dir" init -b main >/dev/null 2>&1 ||
    git -C "$dir" init >/dev/null 2>&1
  git -C "$dir" config user.email "test@example.com"
  git -C "$dir" config user.name "Ralph Test"
  git -C "$dir" remote add origin "$remote_dir"
  printf 'initial\n' > "$dir/README.md"
  git -C "$dir" add README.md
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

# NEW detection: detect_base_branch() -> main (via origin/HEAD)
_new_base_3a="$(cd "$_REPO_3A" && RALPH_XREVIEW_BASE="" detect_base_branch)"
check "3a-iii. detect_base_branch yields 'main' (not the feature branch)" \
  test "$_new_base_3a" = "main"

# NEW gate: diff main...HEAD -> non-empty (the widget.sh commit is new)
_new_diff_3a=0
git -C "$_REPO_3A" diff "main...HEAD" --quiet 2>/dev/null || _new_diff_3a=$?
check "3a-iv. NEW gate sees NON-EMPTY diff (cross-review correctly fires)" \
  test "$_new_diff_3a" -ne 0

# 3b: RALPH_XREVIEW_BASE override wins over origin/HEAD.
echo
echo "  3b: RALPH_XREVIEW_BASE override wins"
_REPO_3B="$WORK_DIR/repo-3b"
_make_repo_with_origin "$_REPO_3B"
_got_3b="$(cd "$_REPO_3B" && RALPH_XREVIEW_BASE=develop detect_base_branch)"
check "3b. RALPH_XREVIEW_BASE=develop wins over origin/HEAD=main" \
  test "$_got_3b" = "develop"

# 3c: origin/HEAD pointing at a non-main default branch is honoured.
echo
echo "  3c: non-main origin/HEAD (develop) is honoured"
_REPO_3C="$WORK_DIR/repo-3c"
_make_repo_with_origin "$_REPO_3C"
git -C "$_REPO_3C" remote set-head origin -d >/dev/null 2>&1 || true
git -C "$_REPO_3C" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/develop >/dev/null 2>&1
_got_3c="$(cd "$_REPO_3C" && RALPH_XREVIEW_BASE="" detect_base_branch)"
check "3c. non-main origin/HEAD -> develop returned" \
  test "$_got_3c" = "develop"

# 3c-edge: origin/HEAD pointing at origin/release/1.0 — strip only leading "origin/".
echo
echo "  3c-edge: origin/HEAD -> origin/release/1.0 strips only leading origin/"
_REPO_3CE="$WORK_DIR/repo-3ce"
_make_repo_with_origin "$_REPO_3CE"
git -C "$_REPO_3CE" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/release/1.0 >/dev/null 2>&1
_got_3ce="$(cd "$_REPO_3CE" && RALPH_XREVIEW_BASE="" detect_base_branch)"
check "3c-edge. release/1.0 stripped of leading origin/ only" \
  test "$_got_3ce" = "release/1.0"

# 3d: no origin/HEAD -> falls back to main/master local branch detection.
echo
echo "  3d: no origin/HEAD -> main/master fallback"
_REPO_3D="$WORK_DIR/repo-3d"
mkdir -p "$_REPO_3D"
git -C "$_REPO_3D" init -b main >/dev/null 2>&1 || git -C "$_REPO_3D" init >/dev/null 2>&1
git -C "$_REPO_3D" config user.email "test@example.com"
git -C "$_REPO_3D" config user.name "Ralph Test"
printf 'hello\n' > "$_REPO_3D/README.md"
git -C "$_REPO_3D" add README.md
git -C "$_REPO_3D" commit -m 'chore: initial' >/dev/null 2>&1
_got_3d_main="$(cd "$_REPO_3D" && RALPH_XREVIEW_BASE="" detect_base_branch)"
check "3d-i. no origin/HEAD + refs/heads/main exists -> main" \
  test "$_got_3d_main" = "main"

# 3d-ii: repo with only master branch falls back to master.
_REPO_3DM="$WORK_DIR/repo-3dm"
mkdir -p "$_REPO_3DM"
git -C "$_REPO_3DM" init -b master >/dev/null 2>&1 || git -C "$_REPO_3DM" init >/dev/null 2>&1
git -C "$_REPO_3DM" config user.email "test@example.com"
git -C "$_REPO_3DM" config user.name "Ralph Test"
printf 'hello\n' > "$_REPO_3DM/README.md"
git -C "$_REPO_3DM" add README.md
git -C "$_REPO_3DM" commit -m 'chore: initial' >/dev/null 2>&1
_got_3d_master="$(cd "$_REPO_3DM" && RALPH_XREVIEW_BASE="" detect_base_branch)"
check "3d-ii. no origin/HEAD + no main + refs/heads/master exists -> master" \
  test "$_got_3d_master" = "master"

# 3e: git worktree shares origin/HEAD from the common git dir.
echo
echo "  3e: git worktree resolves origin/HEAD through shared common dir"
_REPO_3E="$WORK_DIR/repo-3e"
_make_repo_with_origin "$_REPO_3E"
git -C "$_REPO_3E" worktree add "$WORK_DIR/wt-3e" -b feat/wt-test >/dev/null 2>&1
_got_3e="$(cd "$WORK_DIR/wt-3e" && RALPH_XREVIEW_BASE="" detect_base_branch)"
check "3e. worktree resolves origin/HEAD -> main via shared common dir" \
  test "$_got_3e" = "main"

echo
echo "── Summary ──"
printf '  PASS: %d\n  FAIL: %d\n' "$PASS" "$FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
