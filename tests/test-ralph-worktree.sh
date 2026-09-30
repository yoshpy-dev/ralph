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

printf '==> ralph-worktree.sh codex config external-rewrite detection\n'

assert_contains() {
  _desc="$1"
  _needle="$2"
  _haystack="$3"
  _total=$((_total + 1))
  if printf '%s' "$_haystack" | grep -qF -- "$_needle"; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n    missing: %s\n' "$_desc" "$_needle"
  fi
}

assert_not_contains() {
  _desc="$1"
  _needle="$2"
  _haystack="$3"
  _total=$((_total + 1))
  if printf '%s' "$_haystack" | grep -qF -- "$_needle"; then
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n    unexpected: %s\n' "$_desc" "$_needle"
  else
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  fi
}

# _cx_new_repo — create a fresh git repo on main with the fixture
# .codex/config.toml committed; prints the repo's physical (symlink-free)
# path, matching the pwd -P resolution done for $_repo above so stderr
# text built from `git rev-parse --show-toplevel` compares equal to it.
_cx_fixture="$_tmp/codex-fixture.toml"
cat > "$_cx_fixture" <<'FIXTURE'
# .codex/config.toml — test fixture.
#
# A leading comment block explaining the file.

model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
# hooks comment
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "on-request"
FIXTURE

_cx_new_repo() {
  _cx_dir="$(mktemp -d "$_tmp/codex-repo.XXXXXX")"
  mkdir -p "$_cx_dir/.codex"
  cp "$_cx_fixture" "$_cx_dir/.codex/config.toml"
  (
    cd "$_cx_dir"
    git init -b main >/dev/null
    git config user.email test@example.com
    git config user.name "Ralph Test"
    printf 'hello\n' > README.md
    git add README.md .codex/config.toml
    git commit -m 'chore: initial' >/dev/null
  )
  cd "$_cx_dir" && pwd -P
}

# _cx_run <repo> — run validate-clean-base main in <repo>; sets $_cx_stderr
# and $_cx_exit.
_cx_run() {
  set +e
  _cx_stderr="$(cd "$1" && "$RALPH_WORKTREE" validate-clean-base main 2>&1 >/dev/null)"
  _cx_exit=$?
  set -e
}

# _cx_assert_generic <label> — assert the last _cx_run result is the
# pre-existing generic "has uncommitted changes" message (no
# rewrite-specific text).
_cx_assert_generic() {
  assert_eq "$1: exits non-zero" 1 "$_cx_exit"
  assert_contains "$1: mentions uncommitted changes" "has uncommitted changes" "$_cx_stderr"
  assert_not_contains "$1: omits external-rewrite text" "external rewrite" "$_cx_stderr"
}

# _cx_assert_specific <label> <root> — assert the last _cx_run result is the
# rewrite-specific message with the three recovery commands, and omits
# template guidance that a scaffolded (template-free) project cannot use.
_cx_assert_specific() {
  assert_eq "$1: exits non-zero" 1 "$_cx_exit"
  assert_contains "$1: mentions external rewrite" "external rewrite" "$_cx_stderr"
  assert_contains "$1: has diff command" "git -C $2 diff -- .codex/config.toml" "$_cx_stderr"
  assert_contains "$1: has checkout command" "git -C $2 checkout -- .codex/config.toml" "$_cx_stderr"
  assert_contains "$1: has status command" "git -C $2 status --porcelain" "$_cx_stderr"
  assert_not_contains "$1: omits templates/" "templates/" "$_cx_stderr"
  assert_not_contains "$1: omits cmp" "cmp" "$_cx_stderr"
}

# Case 1: comments stripped + [shell_environment_policy] table appended ->
# specific message (also via ensure), then the printed recovery commands
# actually restore a clean, passing state.
_cx_repo1="$(_cx_new_repo)"
cat > "$_cx_repo1/.codex/config.toml" <<'CASE1'
model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "on-request"

[shell_environment_policy]
inherit = "core"

[shell_environment_policy.set]
SOME_VAR = "1"
CASE1
_cx_run "$_cx_repo1"
_cx_assert_specific "case 1 (match shape)" "$_cx_repo1"

set +e
_cx_ensure_stderr="$(cd "$_cx_repo1" && "$RALPH_WORKTREE" ensure --id cx-case1 --kind standard --branch feat/cx-case1 --path .claude/worktrees/cx-case1 2>&1 >/dev/null)"
_cx_ensure_exit=$?
set -e
assert_eq "case 1 via ensure: exits non-zero" 1 "$_cx_ensure_exit"
assert_contains "case 1 via ensure: mentions external rewrite" "external rewrite" "$_cx_ensure_stderr"

git -C "$_cx_repo1" diff -- .codex/config.toml >/dev/null
git -C "$_cx_repo1" checkout -- .codex/config.toml
assert_eq "case 1 recovery: status is clean" "" "$(git -C "$_cx_repo1" status --porcelain)"
_cx_run "$_cx_repo1"
assert_eq "case 1 recovery: validate-clean-base now passes" 0 "$_cx_exit"

# Case 2: comments stripped only (no table appended) -> specific message.
_cx_repo2="$(_cx_new_repo)"
cat > "$_cx_repo2/.codex/config.toml" <<'CASE2'
model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "on-request"
CASE2
_cx_run "$_cx_repo2"
_cx_assert_specific "case 2 (comments stripped only)" "$_cx_repo2"

# Case 3: table appended only, comments kept -> specific message.
_cx_repo3="$(_cx_new_repo)"
cat "$_cx_fixture" > "$_cx_repo3/.codex/config.toml"
cat >> "$_cx_repo3/.codex/config.toml" <<'CASE3'

[shell_environment_policy]
inherit = "core"
CASE3
_cx_run "$_cx_repo3"
_cx_assert_specific "case 3 (table appended only)" "$_cx_repo3"

# Case 4: match shape plus one changed value -> generic message.
_cx_repo4="$(_cx_new_repo)"
cat > "$_cx_repo4/.codex/config.toml" <<'CASE4'
model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "never"

[shell_environment_policy]
inherit = "core"
CASE4
_cx_run "$_cx_repo4"
_cx_assert_generic "case 4 (value changed alongside match shape)"

# Case 5: match shape plus another dirty tracked file -> generic message.
_cx_repo5="$(_cx_new_repo)"
cat > "$_cx_repo5/.codex/config.toml" <<'CASE5'
model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "on-request"

[shell_environment_policy]
inherit = "core"
CASE5
printf 'dirty\n' >> "$_cx_repo5/README.md"
_cx_run "$_cx_repo5"
_cx_assert_generic "case 5 (another dirty tracked file)"

# Case 6: match shape but the appended table is not shell_environment_policy
# -> generic message.
_cx_repo6="$(_cx_new_repo)"
cat > "$_cx_repo6/.codex/config.toml" <<'CASE6'
model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "on-request"

[features2]
extra_key = "x"
CASE6
_cx_run "$_cx_repo6"
_cx_assert_generic "case 6 (non-shell_environment_policy table appended)"

# Case 7: only an unrelated untracked file -> generic message.
_cx_repo7="$(_cx_new_repo)"
printf 'scratch\n' > "$_cx_repo7/scratch.txt"
_cx_run "$_cx_repo7"
_cx_assert_generic "case 7 (unrelated untracked file only)"
rm -f "$_cx_repo7/scratch.txt"

# Case 8: match shape fully staged -> generic message.
_cx_repo8="$(_cx_new_repo)"
cat > "$_cx_repo8/.codex/config.toml" <<'CASE8'
model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "on-request"

[shell_environment_policy]
inherit = "core"
CASE8
git -C "$_cx_repo8" add .codex/config.toml
_cx_run "$_cx_repo8"
_cx_assert_generic "case 8 (staged rewrite)"

# Case 9: match shape partially staged (MM) -> generic message.
_cx_repo9="$(_cx_new_repo)"
cat > "$_cx_repo9/.codex/config.toml" <<'CASE9A'
model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "on-request"

[shell_environment_policy]
inherit = "core"
CASE9A
git -C "$_cx_repo9" add .codex/config.toml
printf '\n[shell_environment_policy.set]\nSOME_VAR = "1"\n' >> "$_cx_repo9/.codex/config.toml"
_cx_run "$_cx_repo9"
_cx_assert_generic "case 9 (partially staged rewrite)"

# Case 10: HEAD has a multi-line string whose body contains a '#' line, and
# the working copy drops that line -> generic message (Codex advisory
# counter-example; the guard must not treat it as a comment removal).
_cx_repo10="$(mktemp -d "$_tmp/codex-repo.XXXXXX")"
mkdir -p "$_cx_repo10/.codex"
cat > "$_cx_repo10/.codex/config.toml" <<'CASE10HEAD'
description = """
Line one
# looks like a comment but is not
Line three
"""
model = "gpt-5.5"
CASE10HEAD
(
  cd "$_cx_repo10"
  git init -b main >/dev/null
  git config user.email test@example.com
  git config user.name "Ralph Test"
  printf 'hello\n' > README.md
  git add README.md .codex/config.toml
  git commit -m 'chore: initial' >/dev/null
)
_cx_repo10="$(cd "$_cx_repo10" && pwd -P)"
cat > "$_cx_repo10/.codex/config.toml" <<'CASE10WORK'
description = """
Line one
Line three
"""
model = "gpt-5.5"
CASE10WORK
_cx_run "$_cx_repo10"
_cx_assert_generic "case 10 (multi-line string guard)"

# Case 11: match shape with CRLF line endings -> specific message.
_cx_repo11="$(_cx_new_repo)"
awk '{printf "%s\r\n", $0}' <<'CASE11' > "$_cx_repo11/.codex/config.toml"
model = "gpt-5.5"
sandbox_mode = "danger-full-access"

[features]
hooks = true

[profiles.work]
model = "gpt-5.5"
approval_policy = "on-request"

[shell_environment_policy]
inherit = "core"
CASE11
_cx_run "$_cx_repo11"
_cx_assert_specific "case 11 (CRLF line endings)" "$_cx_repo11"

# Case 12: .codex/config.toml absent from HEAD (untracked new file) ->
# generic message.
_cx_repo12="$(mktemp -d "$_tmp/codex-repo.XXXXXX")"
(
  cd "$_cx_repo12"
  git init -b main >/dev/null
  git config user.email test@example.com
  git config user.name "Ralph Test"
  printf 'hello\n' > README.md
  git add README.md
  git commit -m 'chore: initial' >/dev/null
)
_cx_repo12="$(cd "$_cx_repo12" && pwd -P)"
mkdir -p "$_cx_repo12/.codex"
cp "$_cx_fixture" "$_cx_repo12/.codex/config.toml"
_cx_run "$_cx_repo12"
_cx_assert_generic "case 12 (.codex/config.toml untracked/new)"

# Case 13: clean base -> validate-clean-base still passes.
_cx_repo13="$(_cx_new_repo)"
_cx_run "$_cx_repo13"
assert_eq "case 13 (clean base) exits 0" 0 "$_cx_exit"

printf '\nralph-worktree tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
