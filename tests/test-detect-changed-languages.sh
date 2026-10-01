#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DETECT="$PROJECT_ROOT/scripts/detect-changed-languages.sh"

if [ ! -x "$DETECT" ]; then
  echo "FAIL: detect-changed-languages.sh not found or not executable at $DETECT" >&2
  exit 1
fi

# The default-base cases must not inherit an explicit base from the caller.
unset RALPH_VERIFY_BASE

_pass=0
_fail=0
_total=0

record_pass() {
  _pass=$((_pass + 1))
  _total=$((_total + 1))
  printf '  PASS: %s\n' "$1"
}

record_fail() {
  _fail=$((_fail + 1))
  _total=$((_total + 1))
  printf '  FAIL: %s\n' "$1"
}

get_field() {
  _field="$1"
  _file="$2"
  sed -n "s/^${_field}=//p" "$_file" | sed -n '1p'
}

assert_field() {
  _desc="$1"
  _field="$2"
  _expected="$3"
  _file="$4"
  _actual="$(get_field "$_field" "$_file")"
  if [ "$_actual" = "$_expected" ]; then
    record_pass "$_desc"
  else
    record_fail "$_desc (expected $_field=$_expected, got $_actual)"
    sed 's/^/    out: /' "$_file"
  fi
}

assert_field_contains() {
  _desc="$1"
  _field="$2"
  _needle="$3"
  _file="$4"
  _actual="$(get_field "$_field" "$_file")"
  case " $_actual " in
    *" $_needle "*) record_pass "$_desc" ;;
    *)
      record_fail "$_desc (expected $_field to contain $_needle, got $_actual)"
      sed 's/^/    out: /' "$_file"
      ;;
  esac
}

assert_field_prefix() {
  _desc="$1"
  _field="$2"
  _prefix="$3"
  _file="$4"
  _actual="$(get_field "$_field" "$_file")"
  case "$_actual" in
    "$_prefix"*) record_pass "$_desc" ;;
    *)
      record_fail "$_desc (expected $_field prefix $_prefix, got $_actual)"
      sed 's/^/    out: /' "$_file"
      ;;
  esac
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/detect-changed-languages.XXXXXX")"
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT HUP INT TERM

# Pin git away from the developer's global and system configuration.
hermetic_home="$workdir/.home"
mkdir -p "$hermetic_home"
HOME="$hermetic_home"
GIT_CONFIG_GLOBAL="$hermetic_home/.gitconfig"
GIT_CONFIG_SYSTEM=/dev/null
GIT_CONFIG_NOSYSTEM=1
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_CONFIG_NOSYSTEM GIT_TERMINAL_PROMPT

make_repo() {
  _repo="$(mktemp -d "$workdir/repo.XXXXXX")"
  (
    cd "$_repo"
    git init -q
    git checkout -q -B main
    git config user.name "Ralph Test"
    git config user.email "ralph-test@example.com"
    printf '# test repo\n' > README.md
    git add README.md
    git commit -q -m "init"
  )
  printf '%s\n' "$_repo"
}

run_detect() {
  _repo="$1"
  _out="$2"
  (cd "$_repo" && "$DETECT") > "$_out"
}

run_detect_base() {
  _repo="$1"
  _base="$2"
  _out="$3"
  (cd "$_repo" && RALPH_VERIFY_BASE="$_base" "$DETECT") > "$_out"
}

# Repo on <branch> whose only remote <remote> is a bare repo under $workdir,
# with <branch> pushed and tracked.
make_repo_with_remote() {
  _remote="$1"
  _branch="$2"
  _bare="$(mktemp -d "$workdir/bare.XXXXXX")"
  git init -q --bare "$_bare"
  _repo="$(mktemp -d "$workdir/repo.XXXXXX")"
  (
    cd "$_repo"
    git init -q
    git checkout -q -B "$_branch"
    git config user.name "Ralph Test"
    git config user.email "ralph-test@example.com"
    printf '# test repo\n' > README.md
    git add README.md
    git commit -q -m "init"
    git remote add "$_remote" "$_bare"
    git push -q -u "$_remote" "$_branch"
  )
  printf '%s\n' "$_repo"
}

# Commit a Go module file in the current directory.
commit_go_module() {
  printf 'module example.com/test\n\ngo 1.22\n' > go.mod
  git add go.mod
  git commit -q -m "add go module"
}

# Make branch <branch> track <remote>, a remote that was never fetched (no
# refs/remotes/<remote>/* exist).
track_unfetched_remote() {
  git remote add "$1" "$workdir/never-fetched.git"
  git config "branch.$2.remote" "$1"
  git config "branch.$2.merge" "refs/heads/$2"
}

# Guard against a vacuous fixture: <name> must resolve to HEAD as a short name.
assert_short_name_shadowed() {
  _desc="$1"
  _repo="$2"
  _name="$3"
  _head="$(cd "$_repo" && git rev-parse HEAD)"
  _resolved="$(cd "$_repo" && git rev-parse --verify --quiet "$_name^{commit}" 2>/dev/null || true)"
  if [ -n "$_head" ] && [ "$_head" = "$_resolved" ]; then
    record_pass "$_desc"
  else
    record_fail "$_desc (HEAD=$_head $_name=$_resolved)"
  fi
}

# Guard against a vacuous fixture: the pushed branch's upstream must be HEAD.
assert_upstream_is_head() {
  _desc="$1"
  _repo="$2"
  _head="$(cd "$_repo" && git rev-parse HEAD)"
  _upstream="$(cd "$_repo" && git rev-parse '@{upstream}' 2>/dev/null || true)"
  if [ -n "$_head" ] && [ "$_head" = "$_upstream" ]; then
    record_pass "$_desc"
  else
    record_fail "$_desc (HEAD=$_head upstream=$_upstream)"
  fi
}

# 1. Single-language uncommitted change.
repo="$(make_repo)"
printf 'package main\n' > "$repo/main.go"
out="$workdir/go.out"
run_detect "$repo" "$out"
assert_field "go change uses changed scope" scope changed "$out"
assert_field "go change is not docs-only" docs_only false "$out"
assert_field "go change selects golang" languages golang "$out"

# 2. Multi-language change.
repo="$(make_repo)"
mkdir -p "$repo/src"
printf 'export const x = 1\n' > "$repo/src/app.ts"
printf 'print(\"x\")\n' > "$repo/tool.py"
out="$workdir/multi.out"
run_detect "$repo" "$out"
assert_field "multi-language uses changed scope" scope changed "$out"
assert_field_contains "multi-language selects typescript" languages typescript "$out"
assert_field_contains "multi-language selects python" languages python "$out"

# 3. Docs-only change does not select a language pack.
repo="$(make_repo)"
mkdir -p "$repo/docs"
printf '# note\n' > "$repo/docs/note.md"
out="$workdir/docs.out"
run_detect "$repo" "$out"
assert_field "docs-only uses changed scope" scope changed "$out"
assert_field "docs-only reason" reason docs_only "$out"
assert_field "docs-only flag" docs_only true "$out"
assert_field "docs-only selects no languages" languages "" "$out"

# 4. Shared CI config falls back to full.
repo="$(make_repo)"
mkdir -p "$repo/.github/workflows"
printf 'name: verify\n' > "$repo/.github/workflows/verify.yml"
out="$workdir/shared.out"
run_detect "$repo" "$out"
assert_field "shared config falls back to full" scope full "$out"
assert_field_prefix "shared config records reason" reason "shared:.github/workflows/verify.yml" "$out"
assert_field "shared config is not docs-only" docs_only false "$out"

# 5. Unclassified code-like files fall back to full.
repo="$(make_repo)"
printf 'opaque\n' > "$repo/unknown.xyz"
out="$workdir/unknown.out"
run_detect "$repo" "$out"
assert_field "unknown file falls back to full" scope full "$out"
assert_field_prefix "unknown file records reason" reason "unclassified:unknown.xyz" "$out"

# 6. No git repository is full fallback.
plain="$workdir/plain"
mkdir -p "$plain"
out="$workdir/no-git.out"
run_detect "$plain" "$out"
assert_field "non-git directory falls back to full" scope full "$out"
assert_field "non-git reason" reason no_git_repository "$out"

# 7. Committed branch changes are detected against main.
repo="$(make_repo)"
(
  cd "$repo"
  git checkout -q -b feature
  printf 'module example.com/test\n\ngo 1.22\n' > go.mod
  git add go.mod
  git commit -q -m "add go module"
)
out="$workdir/branch.out"
run_detect "$repo" "$out"
assert_field "branch diff uses changed scope" scope changed "$out"
assert_field "branch diff selects golang" languages golang "$out"

# 8. JVM markers are not emitted because no JVM pack is shipped.
repo="$(make_repo)"
printf 'plugins {}\n' > "$repo/build.gradle"
out="$workdir/jvm.out"
run_detect "$repo" "$out"
assert_field "JVM marker falls back instead of selecting missing pack" scope full "$out"
assert_field_prefix "JVM marker records unclassified reason" reason "unclassified:build.gradle" "$out"

# 9. Nested project roots are emitted for changed-scope narrowing.
repo="$(make_repo)"
mkdir -p "$repo/service"
printf 'module example.com/service\n\ngo 1.22\n' > "$repo/service/go.mod"
printf 'package main\n' > "$repo/service/main.go"
out="$workdir/go-root.out"
run_detect "$repo" "$out"
assert_field "nested go change selects golang" languages golang "$out"
assert_field "nested go change emits project root" golang_roots service "$out"

# 10. A pushed branch (upstream == HEAD) still reports what it changed since it
#     left the default branch; the upstream is not the diff base.
repo="$(make_repo_with_remote origin main)"
(
  cd "$repo"
  git checkout -q -B feature
  commit_go_module
  git push -q -u origin feature
)
assert_upstream_is_head "pushed branch fixture has upstream equal to HEAD" "$repo"
out="$workdir/pushed.out"
run_detect "$repo" "$out"
assert_field "pushed branch uses changed scope" scope changed "$out"
assert_field "pushed branch is not docs-only" docs_only false "$out"
assert_field "pushed branch selects golang" languages golang "$out"

# 11. An explicit RALPH_VERIFY_BASE still wins over the default base.
out="$workdir/pushed-explicit.out"
run_detect_base "$repo" origin/feature "$out"
assert_field "explicit base equal to HEAD uses changed scope" scope changed "$out"
assert_field "explicit base equal to HEAD reports no_changes" reason no_changes "$out"
assert_field "explicit base equal to HEAD selects no languages" languages "" "$out"

# 12. A nonexistent explicit base falls back to full, as before.
out="$workdir/explicit-missing.out"
run_detect_base "$repo" nope "$out"
assert_field "missing explicit base falls back to full" scope full "$out"
assert_field "missing explicit base records no_merge_base" reason "no_merge_base:nope" "$out"

# 13. origin/HEAD naming a default branch other than main/master (trunk), with
#     no main or master anywhere, is the diff base.
repo="$(make_repo_with_remote origin trunk)"
(
  cd "$repo"
  git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/trunk
  git checkout -q -B feature
  commit_go_module
  git push -q -u origin feature
)
out="$workdir/trunk.out"
run_detect "$repo" "$out"
assert_field "trunk default branch uses changed scope" scope changed "$out"
assert_field "trunk default branch selects golang" languages golang "$out"

# 14. origin/HEAD naming a ref that does not exist is skipped; origin/main wins.
repo="$(make_repo_with_remote origin main)"
(
  cd "$repo"
  git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/gone
  git checkout -q -B feature
  commit_go_module
  git push -q -u origin feature
)
out="$workdir/dangling-head.out"
run_detect "$repo" "$out"
assert_field "dangling origin/HEAD uses changed scope" scope changed "$out"
assert_field "dangling origin/HEAD falls through to origin/main" languages golang "$out"

# 15. On the default branch with nothing unpushed and a clean tree, there is
#     nothing to verify.
repo="$(make_repo_with_remote origin main)"
out="$workdir/default-clean.out"
run_detect "$repo" "$out"
assert_field "clean default branch uses changed scope" scope changed "$out"
assert_field "clean default branch reports no_changes" reason no_changes "$out"
assert_field "clean default branch selects no languages" languages "" "$out"

# 16. An unpushed commit on the default branch is still detected against
#     origin/main.
repo="$(make_repo_with_remote origin main)"
(cd "$repo" && commit_go_module)
out="$workdir/default-unpushed.out"
run_detect "$repo" "$out"
assert_field "unpushed commit on default branch selects golang" languages golang "$out"

# 17. The only remote is not called origin: the branch's tracked remote supplies
#     the default branch, so an unpushed commit on main is not hidden.
repo="$(make_repo_with_remote central main)"
(cd "$repo" && commit_go_module)
out="$workdir/central.out"
run_detect "$repo" "$out"
assert_field "central-only remote uses changed scope" scope changed "$out"
assert_field "central-only remote selects golang for unpushed commit" languages golang "$out"

# 18. Detached HEAD has no tracked remote to consult, so the local main is the
#     base: central/main (older than main) is not used, and main's own python
#     commit stays out of the result.
repo="$(make_repo_with_remote central main)"
(
  cd "$repo"
  printf 'print("x")\n' > tool.py
  git add tool.py
  git commit -q -m "add python tool"
  git checkout -q -b feature
  commit_go_module
  git checkout -q --detach
)
out="$workdir/detached.out"
run_detect "$repo" "$out"
assert_field "detached HEAD uses changed scope" scope changed "$out"
assert_field "detached HEAD diffs against local main only" languages golang "$out"

# 19. A branch tracking a local branch (remote ".") still selects golang.
repo="$(make_repo)"
(
  cd "$repo"
  git checkout -q -b feature
  commit_go_module
  git branch -q --set-upstream-to=main
)
out="$workdir/local-tracking.out"
run_detect "$repo" "$out"
assert_field "local-tracking branch (remote .) uses changed scope" scope changed "$out"
assert_field "local-tracking branch (remote .) still selects golang" languages golang "$out"

# 20. No origin, no tracked remote, and neither main nor master: no base.
repo="$(make_repo)"
(cd "$repo" && git branch -q -m main trunk)
out="$workdir/no-base.out"
run_detect "$repo" "$out"
assert_field "repo without main or master falls back to full" scope full "$out"
assert_field "repo without main or master records no_diff_base" reason no_diff_base "$out"

# 21. A local branch literally named origin/main must not shadow
#     refs/remotes/origin/main: defaults reach merge-base as full ref names.
repo="$(make_repo_with_remote origin main)"
(
  cd "$repo"
  git checkout -q -b feature
  commit_go_module
  git branch origin/main
)
assert_short_name_shadowed "shadow fixture: short name origin/main resolves to HEAD" "$repo" origin/main
out="$workdir/shadow-branch.out"
run_detect "$repo" "$out"
assert_field "branch named origin/main does not hide the diff" scope changed "$out"
assert_field "branch named origin/main does not hide golang" languages golang "$out"

# 22. A tag named main must not shadow the local branch main.
repo="$(make_repo)"
(
  cd "$repo"
  git checkout -q -b feature
  commit_go_module
  git tag main
)
assert_short_name_shadowed "shadow fixture: short name main resolves to HEAD" "$repo" main
out="$workdir/shadow-tag.out"
run_detect "$repo" "$out"
assert_field "tag named main does not hide the diff" scope changed "$out"
assert_field "tag named main does not hide golang" languages golang "$out"

# 23. main tracks a remote that was never fetched, so there is no remote default
#     branch and the local base is main itself: committed changes are invisible,
#     which must fall back to full instead of reporting no_changes.
repo="$(make_repo)"
(
  cd "$repo"
  track_unfetched_remote central main
  commit_go_module
)
out="$workdir/unfetched-central.out"
run_detect "$repo" "$out"
assert_field "unfetched tracked remote falls back to full" scope full "$out"
assert_field "unfetched tracked remote records no_remote_default" reason "no_remote_default:central" "$out"

# 24. The same holds when the tracked remote is origin.
repo="$(make_repo)"
(
  cd "$repo"
  track_unfetched_remote origin main
  commit_go_module
)
out="$workdir/unfetched-origin.out"
run_detect "$repo" "$out"
assert_field "unfetched origin falls back to full" scope full "$out"
assert_field "unfetched origin records no_remote_default" reason "no_remote_default:origin" "$out"

# 25. On another branch the local main is a different branch than HEAD's, so the
#     diff against it is still meaningful.
repo="$(make_repo)"
(
  cd "$repo"
  git checkout -q -b feature
  track_unfetched_remote central feature
  commit_go_module
)
out="$workdir/unfetched-feature.out"
run_detect "$repo" "$out"
assert_field "unfetched remote on a feature branch uses changed scope" scope changed "$out"
assert_field "unfetched remote on a feature branch diffs against local main" languages golang "$out"

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
