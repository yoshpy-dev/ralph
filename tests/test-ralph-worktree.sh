#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RALPH_WORKTREE="${PROJECT_ROOT}/scripts/ralph-worktree.sh"

_pass=0
_fail=0
_total=0
_tmp=""

cleanup() {
  [ -n "$_tmp" ] && rm -rf "$_tmp"
}
trap cleanup EXIT HUP INT TERM

assert_eq() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  _total=$((_total + 1))
  if [ "$_expected" = "$_actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n    expected: %s\n    actual:   %s\n' "$_desc" "$_expected" "$_actual"
  fi
}

assert_exit() {
  _desc="$1"
  _expected="$2"
  shift 2
  _total=$((_total + 1))
  set +e
  "$@" >/dev/null 2>&1
  _actual="$?"
  set -e
  if [ "$_expected" -eq "$_actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s (expected exit %s, got %s)\n' "$_desc" "$_expected" "$_actual"
  fi
}

_tmp="$(mktemp -d)"
_repo="$_tmp/repo"
mkdir -p "$_repo"
(
  cd "$_repo"
  git init -b main >/dev/null
  git config user.email test@example.com
  git config user.name "Ralph Test"
  printf 'hello\n' > README.md
  git add README.md
  git commit -m 'chore: initial' >/dev/null
)
_repo="$(cd "$_repo" && pwd -P)"

printf '==> ralph-worktree.sh state paths\n'
_state_root="$(cd "$_repo" && "$RALPH_WORKTREE" state-root)"
assert_eq "state root uses git common dir" "$_repo/.git/ralph/worktrees" "$_state_root"
_state_path="$(cd "$_repo" && "$RALPH_WORKTREE" state-path 'Spec Issue #77')"
assert_eq "state path sanitizes id" "$_repo/.git/ralph/worktrees/spec-issue-77.json" "$_state_path"
assert_eq "default branch resolves local main" "main" "$(cd "$_repo" && "$RALPH_WORKTREE" default-branch)"
assert_exit "clean main passes validation" 0 sh -c 'cd "$1" && "$2" validate-clean-base main' sh "$_repo" "$RALPH_WORKTREE"

printf '==> ralph-worktree.sh ensure/resume\n'
_wt_path="$(cd "$_repo" && "$RALPH_WORKTREE" ensure \
  --id issue-77 \
  --kind standard \
  --branch feat/worktree-first \
  --path .claude/worktrees/worktree-first \
  --canonical-ref https://github.com/example/repo/issues/77)"
assert_eq "ensure prints absolute worktree path" "$_repo/.claude/worktrees/worktree-first" "$_wt_path"
assert_exit "created worktree is usable" 0 git -C "$_wt_path" rev-parse --is-inside-work-tree
assert_eq "created worktree branch" "feat/worktree-first" "$(git -C "$_wt_path" branch --show-current)"
assert_exit "state file exists" 0 test -f "$_repo/.git/ralph/worktrees/issue-77.json"
assert_eq "resume returns worktree path" "$_wt_path" "$(cd "$_repo" && "$RALPH_WORKTREE" resume --id issue-77)"
assert_eq "ensure resumes matching state" "$_wt_path" "$(cd "$_repo" && "$RALPH_WORKTREE" ensure \
  --id issue-77 \
  --kind standard \
  --branch feat/worktree-first \
  --path .claude/worktrees/worktree-first)"
assert_eq "current returns state path from inside worktree" "$_repo/.git/ralph/worktrees/issue-77.json" "$(cd "$_wt_path" && "$RALPH_WORKTREE" current)"

printf '==> ralph-worktree.sh collision and dirty-base checks\n'
(
  cd "$_repo"
  git branch feat/collision main
)
assert_exit "branch collision without state fails" 1 sh -c 'cd "$1" && "$2" ensure --id collision --kind standard --branch feat/collision --path .claude/worktrees/collision' sh "$_repo" "$RALPH_WORKTREE"
printf 'dirty\n' >> "$_repo/README.md"
assert_exit "dirty default branch fails closed" 1 sh -c 'cd "$1" && "$2" ensure --id dirty --kind standard --branch feat/dirty --path .claude/worktrees/dirty' sh "$_repo" "$RALPH_WORKTREE"
git -C "$_repo" checkout -- README.md

printf '==> ralph-worktree.sh cleanup\n'
assert_exit "cleanup removes worktree and local branch" 0 sh -c 'cd "$1" && "$2" cleanup --id issue-77 --force-branch' sh "$_repo" "$RALPH_WORKTREE"
assert_exit "worktree path removed" 1 test -d "$_wt_path"
assert_exit "local branch removed" 128 git --git-dir="$_repo/.git" show-ref --verify refs/heads/feat/worktree-first
assert_exit "state removed" 1 test -f "$_repo/.git/ralph/worktrees/issue-77.json"

printf '==> ralph-worktree.sh gc\n'
_gc_state_dir="$_repo/.git/ralph/worktrees"
mkdir -p "$_gc_state_dir"

# (a) no state files -> exit 0 + "No stale" message
set +e
_gc_out="$(cd "$_repo" && "$RALPH_WORKTREE" gc 2>&1)"; rc=$?
set -e
assert_eq "gc with no state exits 0" 0 "$rc"
assert_eq "gc with no state prints No stale message" "No stale ralph worktree state." "$_gc_out"

# (b) one stale state (path missing) -> gc exits 0, lists STALE, file NOT deleted
_stale_json="$_gc_state_dir/stale-task.json"
printf '{"id":"stale-task","branch":"feat/stale","worktree_path":"%s/nonexistent-path","kind":"standard"}\n' "$_repo" > "$_stale_json"
set +e
_gc_out="$(cd "$_repo" && "$RALPH_WORKTREE" gc 2>&1)"; rc=$?
set -e
assert_eq "gc with stale entry exits 0" 0 "$rc"
assert_eq "gc lists stale entry" "STALE $_stale_json branch=feat/stale path=$_repo/nonexistent-path" "$_gc_out"
assert_exit "gc does not delete state file without --prune" 0 test -f "$_stale_json"

# (c) gc --prune -> exits 0, file deleted, second run reports "No stale" with exit 0
set +e
_gc_prune_out="$(cd "$_repo" && "$RALPH_WORKTREE" gc --prune 2>&1)"; rc=$?
set -e
assert_eq "gc --prune exits 0" 0 "$rc"
assert_exit "gc --prune deletes stale state file" 1 test -f "$_stale_json"
set +e
_gc_second_out="$(cd "$_repo" && "$RALPH_WORKTREE" gc 2>&1)"; rc=$?
set -e
assert_eq "gc after prune exits 0" 0 "$rc"
assert_eq "gc after prune prints No stale message" "No stale ralph worktree state." "$_gc_second_out"

# (d) non-stale state (worktree path exists) -> not listed, not deleted
_live_wt_path="$_tmp/live-worktree"
mkdir -p "$_live_wt_path"
_live_json="$_gc_state_dir/live-task.json"
printf '{"id":"live-task","branch":"feat/live","worktree_path":"%s","kind":"standard"}\n' "$_live_wt_path" > "$_live_json"
set +e
_gc_live_out="$(cd "$_repo" && "$RALPH_WORKTREE" gc 2>&1)"; rc=$?
set -e
assert_eq "gc with non-stale entry exits 0" 0 "$rc"
assert_eq "gc with non-stale entry prints No stale message" "No stale ralph worktree state." "$_gc_live_out"
assert_exit "gc does not delete non-stale state file" 0 test -f "$_live_json"
rm -f "$_live_json"

printf '\nralph-worktree tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
