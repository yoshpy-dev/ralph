#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DETECT="$PROJECT_ROOT/scripts/detect-changed-languages.sh"

if [ ! -x "$DETECT" ]; then
  echo "FAIL: detect-changed-languages.sh not found or not executable at $DETECT" >&2
  exit 1
fi

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

assert_output_lacks() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if grep -F -q -- "$_needle" "$_file"; then
    record_fail "$_desc (output contains $_needle)"
    sed 's/^/    out: /' "$_file"
  else
    record_pass "$_desc"
  fi
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

# assert_file_empty <desc> <file> -- <file> (a run's stderr) is empty.
assert_file_empty() {
  if [ -s "$2" ]; then
    record_fail "$1"
    sed 's/^/    err: /' "$2"
  else
    record_pass "$1"
  fi
}

# assert_file_contains <desc> <needle> <file> -- <file> holds <needle>.
assert_file_contains() {
  if grep -F -q -- "$2" "$3"; then
    record_pass "$1"
  else
    record_fail "$1 (missing $2)"
    sed 's/^/    err: /' "$3"
  fi
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

# The stand-in ralph (tests/lib/fake-ralph.sh) that answers
# `ralph config get verify.base` for the fixtures that commit
# scripts/ralph-config.sh (commit_ralph_config). The other fixtures hold no
# ralph-config.sh, so the detector asks no ralph there.
# shellcheck source=lib/fake-ralph.sh
. "$PROJECT_ROOT/tests/lib/fake-ralph.sh"
FAKE_BIN="$workdir/fake-bin"
fake_ralph_install "$FAKE_BIN"
err="$workdir/detect.err"

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

# run_detect_setting <dir> <out> [<name=value>...] -- runs the detector in
# <dir> with the stand-in ralph first on PATH and the given variables added
# to its environment; its stderr goes to $err.
run_detect_setting() {
  _repo="$1"
  _out="$2"
  shift 2
  (cd "$_repo" && env PATH="$FAKE_BIN:$PATH" "$@" "$DETECT") > "$_out" 2> "$err"
}

# base_value <value> -- the stand-in answers verify.base with <value> ("" for
# the empty default), and its log starts empty.
base_value() {
  fake_ralph_reset "$FAKE_BIN"
  fake_ralph_value "$FAKE_BIN" verify.base "$1"
}

# root_call <repo> -- the stand-in's log line for one read of verify.base in
# the root of <repo>.
root_call() {
  printf '%s|config get verify.base' "$(cd "$1" && pwd -P)"
}

# assert_calls <desc> <want> -- the stand-in's log since its last reset is
# <want>.
assert_calls() {
  _got="$(fake_ralph_calls "$FAKE_BIN")"
  if [ "$_got" = "$2" ]; then
    record_pass "$1"
  else
    record_fail "$1 (log '$_got', want '$2')"
  fi
}

# Commit a copy of scripts/ralph-config.sh in the current directory, as a
# scaffolded project holds it, so the detector reads [verify].base.
commit_ralph_config() {
  mkdir -p scripts
  cp "$PROJECT_ROOT/scripts/ralph-config.sh" scripts/ralph-config.sh
  git add scripts/ralph-config.sh
  git commit -q -m "add ralph-config.sh"
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

# Commit a python file in the current directory.
commit_python_tool() {
  printf 'print("x")\n' > tool.py
  git add tool.py
  git commit -q -m "add python tool"
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

# 11. [verify].base of ralph.local.toml wins over the default base. The
#     fixture commits scripts/ralph-config.sh on main, and its pushed
#     feature branch adds a python and then a Go commit on top of
#     origin/main.
repo="$(make_repo_with_remote origin main)"
(
  cd "$repo"
  commit_ralph_config
  git push -q origin main
  git checkout -q -B feature
  commit_python_tool
  commit_go_module
  git push -q -u origin feature
)
cfg_repo="$repo"
base_value origin/feature
out="$workdir/setting-head.out"
run_detect_setting "$repo" "$out"
assert_field "[verify].base equal to HEAD uses changed scope" scope changed "$out"
assert_field "[verify].base equal to HEAD reports no_changes" reason no_changes "$out"
assert_field "[verify].base equal to HEAD selects no languages" languages "" "$out"
assert_calls "[verify].base is read once, in the root" "$(root_call "$repo")"
assert_file_empty "[verify].base equal to HEAD writes nothing on stderr" "$err"

base_value HEAD~1
out="$workdir/setting-parent.out"
run_detect_setting "$repo" "$out"
assert_field "[verify].base HEAD~1 uses changed scope" scope changed "$out"
assert_field "[verify].base HEAD~1 selects only the last commit's golang" languages golang "$out"

# 12. A nonexistent [verify].base falls back to full, as before.
base_value nope
out="$workdir/setting-missing.out"
run_detect_setting "$repo" "$out"
assert_field "missing [verify].base falls back to full" scope full "$out"
assert_field "missing [verify].base records no_merge_base" reason "no_merge_base:nope" "$out"

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
  commit_python_tool
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

# 26. origin is consulted before the tracked remote: a stale fork that the
#     feature branch tracks must not become the base (its older main would pull
#     main's python commit into the result).
repo="$(make_repo_with_remote fork main)"
origin_bare="$(mktemp -d "$workdir/bare.XXXXXX")"
git init -q --bare "$origin_bare"
(
  cd "$repo"
  commit_python_tool
  git remote add origin "$origin_bare"
  git push -q origin main
  git checkout -q -b feature
  commit_go_module
  git push -q -u fork feature
)
out="$workdir/origin-before-tracked.out"
run_detect "$repo" "$out"
assert_field "origin plus a tracked stale fork uses changed scope" scope changed "$out"
assert_field "origin default branch wins over the tracked fork (exact languages)" languages golang "$out"

# 27. A remote whose only default branch is master (no main anywhere, no remote
#     HEAD) is the base.
repo="$(make_repo_with_remote origin master)"
(
  cd "$repo"
  git checkout -q -b feature
  commit_go_module
  git push -q -u origin feature
  git branch -q -D master
)
out="$workdir/remote-master.out"
run_detect "$repo" "$out"
assert_field "remote master-only default uses changed scope" scope changed "$out"
assert_field "remote master-only default selects golang" languages golang "$out"

# 28. A repo with no remote and only a local master uses it as the base.
repo="$(make_repo)"
(
  cd "$repo"
  git branch -q -m main master
  git checkout -q -b feature
  commit_go_module
)
out="$workdir/local-master.out"
run_detect "$repo" "$out"
assert_field "local master-only default uses changed scope" scope changed "$out"
assert_field "local master-only default selects golang" languages golang "$out"

# 29. When a remote has both main and master at different tips, main is the base
#     (the stale master would pull main's python commit into the result).
repo="$(make_repo_with_remote origin master)"
(
  cd "$repo"
  git checkout -q -b main
  commit_python_tool
  git push -q -u origin main
  git checkout -q -b feature
  commit_go_module
  git push -q -u origin feature
)
out="$workdir/remote-main-over-master.out"
run_detect "$repo" "$out"
assert_field "remote main wins over a stale master" languages golang "$out"

# 30. The same order holds for local branches: main wins over a stale master.
repo="$(make_repo)"
(
  cd "$repo"
  git branch -q master
  commit_python_tool
  git checkout -q -b feature
  commit_go_module
)
out="$workdir/local-main-over-master.out"
run_detect "$repo" "$out"
assert_field "local main wins over a stale master" languages golang "$out"

# 31. A refs/remotes/origin/HEAD that points outside refs/remotes/origin/ (here
#     at the current branch) is ignored; origin/main is the base.
repo="$(make_repo_with_remote origin main)"
(
  cd "$repo"
  git checkout -q -b feature
  commit_go_module
  git symbolic-ref refs/remotes/origin/HEAD refs/heads/feature
)
out="$workdir/head-outside-remote.out"
run_detect "$repo" "$out"
assert_field "origin/HEAD outside the remote uses changed scope" scope changed "$out"
assert_field "origin/HEAD outside the remote falls through to origin/main" languages golang "$out"

# 32. main tracks a local branch (remote ".") and there is no remote default:
#     the local base is main itself, so committed changes are invisible and the
#     result falls back to full, naming "." as the remote.
repo="$(make_repo)"
(
  cd "$repo"
  git branch -q base
  git branch -q --set-upstream-to=base main
  commit_go_module
)
out="$workdir/local-upstream-main.out"
run_detect "$repo" "$out"
assert_field "main tracking a local branch falls back to full" scope full "$out"
assert_field "main tracking a local branch records no_remote_default" reason "no_remote_default:." "$out"

# 33. main's tracked "remote" is a URL, not a remote name, and there is no
#     origin: the guard still falls back to full, but the reason names "url"
#     instead of the value (a URL can carry credentials) and the URL appears
#     nowhere in the output.
repo="$(make_repo)"
url_bare="$(mktemp -d "$workdir/bare.XXXXXX")"
git init -q --bare "$url_bare"
url_scheme="file"
url_value="${url_scheme}://${url_bare}"
(
  cd "$repo"
  git config branch.main.remote "$url_value"
  git config branch.main.merge refs/heads/main
  commit_go_module
)
out="$workdir/url-remote.out"
run_detect "$repo" "$out"
assert_field "URL as tracked remote falls back to full" scope full "$out"
assert_field "URL as tracked remote records no_remote_default:url" reason "no_remote_default:url" "$out"
assert_output_lacks "URL as tracked remote leaves the scheme out of the output" "${url_scheme}://" "$out"
assert_output_lacks "URL as tracked remote leaves the path out of the output" "$url_bare" "$out"

# 34. main tracks a "ghost" remote: its config section is gone but its
#     refs/remotes/<name>/* remain. It is not a configured remote name, so the
#     reason says "url" and step 2 must not consult the leftover refs (using
#     them would report a changed-scope golang diff instead of the full
#     fallback).
repo="$(make_repo)"
ghost_bare="$(mktemp -d "$workdir/bare.XXXXXX")"
git init -q --bare "$ghost_bare"
(
  cd "$repo"
  git remote add ghost "$ghost_bare"
  git push -q -u ghost main
  git config --remove-section remote.ghost
  commit_go_module
)
if (cd "$repo" && git show-ref --verify --quiet refs/remotes/ghost/main &&
  ! git config --get remote.ghost.url >/dev/null 2>&1 &&
  [ "$(git config --get branch.main.remote)" = ghost ]); then
  record_pass "ghost fixture keeps refs/remotes/ghost/main without a remote.ghost section"
else
  record_fail "ghost fixture keeps refs/remotes/ghost/main without a remote.ghost section"
fi
out="$workdir/ghost-remote.out"
run_detect "$repo" "$out"
assert_field "ghost remote falls back to full" scope full "$out"
assert_field "ghost remote records no_remote_default:url" reason "no_remote_default:url" "$out"

# 35. A remote name with both a dot and a slash is a configured remote name: the
#     fetched one supplies the base, the unfetched one is named in the reason.
dotted_remote="team/central.eu"
repo="$(make_repo_with_remote "$dotted_remote" main)"
(cd "$repo" && commit_go_module)
out="$workdir/dotted-remote-fetched.out"
run_detect "$repo" "$out"
assert_field "dotted and slashed remote name uses changed scope" scope changed "$out"
assert_field "dotted and slashed remote name selects golang" languages golang "$out"

repo="$(make_repo)"
(
  cd "$repo"
  track_unfetched_remote "$dotted_remote" main
  commit_go_module
)
out="$workdir/dotted-remote-unfetched.out"
run_detect "$repo" "$out"
assert_field "unfetched dotted and slashed remote falls back to full" scope full "$out"
assert_field "unfetched dotted and slashed remote is named in the reason" reason "no_remote_default:$dotted_remote" "$out"

# 36. An empty [verify].base is the default: the automatic base
#     (origin/main), silently. The fixture is case 11's.
base_value ""
out="$workdir/setting-empty.out"
run_detect_setting "$cfg_repo" "$out"
assert_field "empty [verify].base diffs against origin/main" languages "golang python" "$out"
assert_file_empty "empty [verify].base writes nothing on stderr" "$err"

# 37. RALPH_VERIFY_BASE, which earlier versions read, changes nothing: not
#     with [verify].base empty, and not against a value.
base_value ""
out="$workdir/env-with-empty-setting.out"
run_detect_setting "$cfg_repo" "$out" RALPH_VERIFY_BASE=HEAD~1
assert_field "RALPH_VERIFY_BASE=HEAD~1 with [verify].base empty still diffs against origin/main" languages "golang python" "$out"
base_value HEAD~1
out="$workdir/env-with-setting.out"
run_detect_setting "$cfg_repo" "$out" RALPH_VERIFY_BASE=origin/feature
assert_field "RALPH_VERIFY_BASE=origin/feature does not override [verify].base HEAD~1" languages golang "$out"

# 38. From a subdirectory, ralph is asked in the repository's root.
mkdir -p "$cfg_repo/sub/dir"
base_value ""
out="$workdir/setting-subdir.out"
run_detect_setting "$cfg_repo/sub/dir" "$out"
assert_calls "from a subdirectory, verify.base is read in the root" "$(root_call "$cfg_repo")"

# 39. No ralph on PATH (exit 127, as in CI): the default, silently. PATH
#     holds only the tools the detector and ralph-config.sh run.
tools="$workdir/tools"
mkdir -p "$tools"
for tool in sh git mktemp sort sed find dirname rm cat grep; do
  printf '#!/bin/sh\nexec "%s" "$@"\n' "$(command -v "$tool")" > "$tools/$tool"
  chmod +x "$tools/$tool"
done
if (PATH="$tools" && command -v ralph >/dev/null 2>&1); then
  record_fail "tools fixture: the tools-only PATH has no ralph"
else
  record_pass "tools fixture: the tools-only PATH has no ralph"
fi
out="$workdir/no-ralph.out"
(cd "$cfg_repo" && PATH="$tools" "$DETECT") > "$out" 2> "$err"
assert_field "no ralph on PATH diffs against origin/main" languages "golang python" "$out"
assert_file_empty "no ralph on PATH writes nothing on stderr" "$err"

# 40. A ralph older than `config get`: the default, silently.
base_value HEAD~1
fake_ralph_old "$FAKE_BIN"
out="$workdir/old-ralph.out"
run_detect_setting "$cfg_repo" "$out"
assert_field "an older ralph diffs against origin/main" languages "golang python" "$out"
assert_file_empty "an older ralph writes nothing on stderr" "$err"
assert_calls "the older ralph is asked once, in the root" "$(root_call "$cfg_repo")"

# 41. D7 of plan 2026-10-30-script-local-config: on main with no remote and
#     a clean tree the automatic base is main itself, so the changed scope
#     reports no_changes although main holds a Go commit (the control). When
#     ralph cannot read the settings (exit 3 for a ralph.local.toml that
#     fails validation, or exit 2), the result is full with reason
#     config_error:verify.base, never no_changes.
repo="$(make_repo)"
(cd "$repo" && commit_ralph_config && commit_go_module)
base_value ""
out="$workdir/config-ok-main.out"
run_detect_setting "$repo" "$out"
assert_field "control: with [verify].base empty, clean main reports no_changes" reason no_changes "$out"
for code in 3 2; do
  fake_ralph_reset "$FAKE_BIN"
  fake_ralph_fail "$FAKE_BIN" verify.base "$code" "config get: ralph.local.toml: [plan_visual].browser must be an absolute path"
  out="$workdir/config-error-$code.out"
  run_detect_setting "$repo" "$out"
  assert_field "exit $code falls back to full" scope full "$out"
  assert_field "exit $code records config_error:verify.base" reason config_error:verify.base "$out"
  assert_field "exit $code is not docs-only" docs_only false "$out"
  assert_output_lacks "exit $code does not report no_changes" no_changes "$out"
  assert_file_contains "exit $code: ralph-config.sh names the key and the exit code" \
    "ralph config get verify.base exited $code" "$err"
  assert_file_contains "exit $code: the detector says it takes the full scope" \
    "detect-changed-languages.sh: [verify].base could not be read (the line above says why; a mistake in any key of ralph.toml or ralph.local.toml makes every key fail); taking the full scope, not the automatic base" "$err"
done

# 42. A repository with no scripts/ralph-config.sh (scaffolded before it
#     existed) has no setting to read: ralph is not asked, not even one that
#     would fail, and the automatic base applies.
repo="$(make_repo)"
(cd "$repo" && commit_go_module)
fake_ralph_reset "$FAKE_BIN"
fake_ralph_fail "$FAKE_BIN" verify.base 3 "config get: not expected"
out="$workdir/no-config.out"
run_detect_setting "$repo" "$out"
assert_field "no ralph-config.sh: clean main reports no_changes" reason no_changes "$out"
assert_calls "no ralph-config.sh: ralph is not asked" ""

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
