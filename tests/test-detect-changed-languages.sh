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

# The detector takes GITHUB_BASE_REF as the base before anything else, and
# this suite's own CI run sets it for real (the pull request's base): a
# leaked value would pick another base for these ad hoc repos. The cases
# that test it pass it explicitly.
unset GITHUB_BASE_REF

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

# run_detect_fake_git <dir> <out> <pattern> -- runs the detector in <dir>
# with the stand-in git of case 52 ($fake_git_bin) first on PATH, failing
# the calls whose arguments match <pattern> (logged to $fake_git_log, which
# starts empty), and TMPDIR set to the empty $tmp_check; its stderr goes to
# $err.
run_detect_fake_git() {
  : > "$fake_git_log"
  (cd "$1" && PATH="$fake_git_bin:$PATH" FAKE_GIT_FAIL="$3" FAKE_GIT_LOG="$fake_git_log" TMPDIR="$tmp_check" "$DETECT") > "$2" 2> "$err"
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
assert_field "go change names main.go as the first non-docs path" first_non_docs main.go "$out"
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
assert_output_lacks "docs-only prints no first_non_docs" first_non_docs= "$out"

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
assert_field "non-git directory: docs_only is unknown" docs_only unknown "$out"

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
assert_field "missing [verify].base: docs_only is unknown" docs_only unknown "$out"

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
assert_field "repo without main or master: docs_only is unknown" docs_only unknown "$out"

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
assert_field "unfetched tracked remote: docs_only is unknown" docs_only unknown "$out"

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
assert_field "unfetched origin: docs_only is unknown" docs_only unknown "$out"

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
assert_field "main tracking a local branch: docs_only is unknown" docs_only unknown "$out"

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
assert_field "URL as tracked remote: docs_only is unknown" docs_only unknown "$out"
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
assert_field "ghost remote: docs_only is unknown" docs_only unknown "$out"

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
assert_field "unfetched dotted and slashed remote: docs_only is unknown" docs_only unknown "$out"

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
  assert_field "exit $code: docs_only is unknown (the change set was not read)" docs_only unknown "$out"
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

# write_manifest <file> <table header>... -- writes a .ralph/manifest.toml in
# the shape ralph init writes (go-toml v2): [meta], [files], then one table
# per header line given, each with an owner key.
write_manifest() {
  _manifest="$1"
  shift
  mkdir -p "$(dirname "$_manifest")"
  {
    printf "[meta]\nversion = 'dev'\nlayout = 'v2'\n\n[files]\n"
    for _header in "$@"; do
      printf "%s\nowner = 'core'\n\n" "$_header"
    done
  } > "$_manifest"
}

# assert_dir_empty <desc> <dir> -- <dir> holds no file.
assert_dir_empty() {
  _left="$(ls -A "$2")"
  if [ -z "$_left" ]; then
    record_pass "$1"
  else
    record_fail "$1 (left: $_left)"
  fi
}

# 43. A change of only docs and ralph scaffold files is docs only. The scope
#     and the reason stay as before: a scaffold file that is neither docs nor
#     a language file is unclassified, so the scope is full. A file under
#     .claude/ that the manifest does not list is not scaffold.
repo="$(make_repo)"
(
  cd "$repo"
  mkdir -p .claude/hooks .claude/skills/demo
  printf '#!/bin/sh\n' > .claude/hooks/check.sh
  printf '# demo\n' > .claude/skills/demo/SKILL.md
  write_manifest .ralph/manifest.toml \
    "[files.'.claude/hooks/check.sh']" \
    "[files.'.claude/skills/demo/SKILL.md']"
  git add -A
  git commit -q -m "scaffold"
  git checkout -q -b feature
  printf 'exit 0\n' >> .claude/hooks/check.sh
  printf 'more\n' >> .claude/skills/demo/SKILL.md
  printf 'more\n' >> README.md
)
out="$workdir/scaffold-mix.out"
run_detect "$repo" "$out"
assert_field "docs and scaffold files: docs_only is true" docs_only true "$out"
assert_field "docs and scaffold files: an unclassified scaffold file keeps the full scope" scope full "$out"
assert_field "docs and scaffold files: the reason names that file as before" reason "unclassified:.claude/hooks/check.sh" "$out"
assert_output_lacks "docs and scaffold files: no first_non_docs" first_non_docs= "$out"

printf '#!/bin/sh\n' > "$repo/.claude/hooks/mine.sh"
out="$workdir/scaffold-mix-own.out"
run_detect "$repo" "$out"
assert_field "an unlisted file under .claude/ is not docs only" docs_only false "$out"
assert_field "an unlisted file under .claude/ is the first non-docs path" first_non_docs .claude/hooks/mine.sh "$out"
assert_field "an unlisted file under .claude/ leaves the reason as before" reason "unclassified:.claude/hooks/check.sh" "$out"

# 44. A scaffold file that is a language file keeps the changed scope and its
#     language and root, and is still docs only.
repo="$(make_repo)"
(
  cd "$repo"
  mkdir -p .claude/hooks
  printf '{}\n' > .claude/hooks/package.json
  write_manifest .ralph/manifest.toml "[files.'.claude/hooks/package.json']"
  git add -A
  git commit -q -m "scaffold"
  printf '{"name":"x"}\n' > .claude/hooks/package.json
)
out="$workdir/scaffold-language.out"
run_detect "$repo" "$out"
assert_field "scaffold language file uses changed scope" scope changed "$out"
assert_field "scaffold language file reports changed_languages" reason changed_languages "$out"
assert_field "scaffold language file selects typescript" languages typescript "$out"
assert_field "scaffold language file keeps its root" typescript_roots .claude/hooks "$out"
assert_field "scaffold language file is docs only" docs_only true "$out"

# 45. The three forms of a [files.<key>] table: single-quoted (what ralph init
#     writes), double-quoted, and bare. Each scaffold file, changed next to
#     README.md, is docs only; an unlisted file is not.
repo="$(make_repo)"
(
  cd "$repo"
  printf 'x = 1\n' > ralph.toml
  printf 'tool\n' > quoted-tool
  printf 'tool\n' > ralph-tool
  write_manifest .ralph/manifest.toml \
    "[files.'ralph.toml']" \
    '[files."quoted-tool"]' \
    '[files.ralph-tool]'
  git add -A
  git commit -q -m "scaffold"
)
for key_form in single-quoted double-quoted bare unlisted; do
  case "$key_form" in
    single-quoted) changed_path=ralph.toml ;;
    double-quoted) changed_path=quoted-tool ;;
    bare) changed_path=ralph-tool ;;
    unlisted) changed_path=own-tool ;;
  esac
  (
    cd "$repo"
    git checkout -q -- .
    printf 'more\n' >> "$changed_path"
    printf 'more\n' >> README.md
  )
  out="$workdir/key-form-$key_form.out"
  run_detect "$repo" "$out"
  assert_field "$key_form key ($changed_path): the reason stays unclassified" reason "unclassified:$changed_path" "$out"
  if [ "$key_form" = unlisted ]; then
    assert_field "unlisted file ($changed_path) is not docs only" docs_only false "$out"
    assert_field "unlisted file ($changed_path) is the first non-docs path" first_non_docs "$changed_path" "$out"
  else
    assert_field "$key_form key ($changed_path) is scaffold: docs_only is true" docs_only true "$out"
  fi
done

# 46. A double-quoted key with a backslash is not taken. The key below is
#     tool, then the TOML escape of the dot (a backslash, u, 002E), then sh:
#     TOML reads it as tool.sh, but the detector does not decode escapes, so
#     tool.sh is not scaffold. (git quotes a path that holds a backslash, so
#     a changed path never holds one either.) The backslash is put in at run
#     time, so the escape stays six characters in the manifest.
backslash="\\"
repo="$(make_repo)"
(
  cd "$repo"
  printf '#!/bin/sh\n' > tool.sh
  write_manifest .ralph/manifest.toml "[files.\"tool${backslash}u002Esh\"]"
  git add -A
  git commit -q -m "scaffold"
  printf 'exit 0\n' >> tool.sh
)
if grep -Fqx "[files.\"tool${backslash}u002Esh\"]" "$repo/.ralph/manifest.toml"; then
  record_pass "backslash fixture: the manifest holds the escape as written"
else
  record_fail "backslash fixture: the manifest holds the escape as written"
fi
out="$workdir/key-backslash.out"
run_detect "$repo" "$out"
assert_field "double-quoted key with a backslash is not scaffold" docs_only false "$out"
assert_field "double-quoted key with a backslash: tool.sh is the first non-docs path" first_non_docs tool.sh "$out"

# 47. .ralph/manifest.toml itself and a path under .ralph/baseline/ are
#     scaffold, with no manifest entry for them.
repo="$(make_repo)"
write_manifest "$repo/.ralph/manifest.toml"
out="$workdir/scaffold-manifest.out"
run_detect "$repo" "$out"
assert_field "the manifest itself is scaffold: docs_only is true" docs_only true "$out"
assert_field "the manifest itself: the reason stays unclassified" reason "unclassified:.ralph/manifest.toml" "$out"
rm -rf "$repo/.ralph"
mkdir -p "$repo/.ralph/baseline"
printf 'x\n' > "$repo/.ralph/baseline/x"
out="$workdir/scaffold-baseline.out"
run_detect "$repo" "$out"
assert_field "a path under .ralph/baseline/ is scaffold: docs_only is true" docs_only true "$out"
assert_field "a path under .ralph/baseline/: the reason stays unclassified" reason "unclassified:.ralph/baseline/x" "$out"

# 48. A shared harness file that the manifest lists keeps the full scope and
#     its reason, and is docs only. A Go file after it in the change set is
#     still read for docs_only, but adds no language and no root (go.mod is
#     in the root, so a root would show).
repo="$(make_repo)"
(
  cd "$repo"
  commit_go_module
  mkdir -p scripts
  printf '#!/bin/sh\n' > scripts/run-verify.sh
  write_manifest .ralph/manifest.toml "[files.'scripts/run-verify.sh']"
  git add -A
  git commit -q -m "scaffold"
  git checkout -q -b feature
  printf 'exit 0\n' >> scripts/run-verify.sh
  printf 'more\n' >> README.md
)
out="$workdir/shared-scaffold.out"
run_detect "$repo" "$out"
assert_field "shared scaffold file falls back to full" scope full "$out"
assert_field "shared scaffold file records its shared reason" reason "shared:scripts/run-verify.sh" "$out"
assert_field "shared scaffold file next to docs is docs only" docs_only true "$out"

printf 'package main\n' > "$repo/zz.go"
out="$workdir/shared-scaffold-then-go.out"
run_detect "$repo" "$out"
assert_field "a Go file after the shared file keeps the shared reason" reason "shared:scripts/run-verify.sh" "$out"
assert_field "a Go file after the shared file makes docs_only false" docs_only false "$out"
assert_field "a Go file after the shared file is the first non-docs path" first_non_docs zz.go "$out"
assert_field "a Go file after the shared file adds no language" languages "" "$out"
assert_output_lacks "a Go file after the shared file adds no root" golang_roots= "$out"

# 49. A shared file that is not scaffold, after a Go file: the scope, reason,
#     and the root collected before the shared file are as before, and the
#     Go file is the first non-docs path.
repo="$(make_repo)"
(
  cd "$repo"
  commit_go_module
  printf 'package main\n' > app.go
  mkdir -p schemas
  printf '{}\n' > schemas/api.json
)
out="$workdir/shared-after-go.out"
run_detect "$repo" "$out"
assert_field "shared file after a Go file falls back to full" scope full "$out"
assert_field "shared file after a Go file records its reason" reason "shared:schemas/api.json" "$out"
assert_field "shared file after a Go file is not docs only" docs_only false "$out"
assert_field "shared file after a Go file names the Go file first" first_non_docs app.go "$out"
assert_field "shared file after a Go file selects no languages, as before" languages "" "$out"
assert_field "shared file after a Go file keeps the root collected before it" golang_roots . "$out"

# 50. GITHUB_BASE_REF names the base first, read as secret-scan-branch.sh
#     reads it. origin has main and develop; develop holds a python commit
#     that main does not, and the feature branch adds a Go commit on top of
#     develop. The local develop is moved back to main, so only
#     refs/remotes/origin/develop gives exactly golang: the automatic base
#     (origin/main) and the local develop both give "golang python".
repo="$(make_repo_with_remote origin main)"
(
  cd "$repo"
  git checkout -q -b develop
  commit_python_tool
  git push -q -u origin develop
  git checkout -q -b feature
  commit_go_module
  git branch -q -f develop main
  git branch -q localbase refs/remotes/origin/develop
)
out="$workdir/base-ref-unset.out"
run_detect_setting "$repo" "$out"
assert_field "control: without GITHUB_BASE_REF the base is origin/main" languages "golang python" "$out"

out="$workdir/base-ref-origin.out"
run_detect_setting "$repo" "$out" GITHUB_BASE_REF=develop
assert_field "GITHUB_BASE_REF=develop uses changed scope" scope changed "$out"
assert_field "GITHUB_BASE_REF=develop diffs against origin/develop, not the local develop" languages golang "$out"

out="$workdir/base-ref-local.out"
run_detect_setting "$repo" "$out" GITHUB_BASE_REF=localbase
assert_field "GITHUB_BASE_REF with only a local branch uses changed scope" scope changed "$out"
assert_field "GITHUB_BASE_REF with only a local branch diffs against it" languages golang "$out"

out="$workdir/base-ref-missing.out"
run_detect_setting "$repo" "$out" GITHUB_BASE_REF=nope
assert_field "GITHUB_BASE_REF with no ref falls back to full" scope full "$out"
assert_field "GITHUB_BASE_REF with no ref records no_base_ref" reason "no_base_ref:nope" "$out"
assert_field "GITHUB_BASE_REF with no ref: docs_only is unknown" docs_only unknown "$out"
assert_field "GITHUB_BASE_REF with no ref selects no languages" languages "" "$out"

out="$workdir/base-ref-odd.out"
run_detect_setting "$repo" "$out" "GITHUB_BASE_REF=bad name;x"
assert_field "GITHUB_BASE_REF with odd characters records no_base_ref:GITHUB_BASE_REF" reason "no_base_ref:GITHUB_BASE_REF" "$out"
assert_field "GITHUB_BASE_REF with odd characters: docs_only is unknown" docs_only unknown "$out"
assert_output_lacks "GITHUB_BASE_REF with odd characters leaves the value out of the output" "bad name" "$out"

# A ref with no merge-base with HEAD gives no_merge_base, and the value shows
# by the same rule: an orphan commit on origin as orphan and rel+v1#x, and
# only locally as rel+v2#x (git allows + and # in a branch name).
(
  cd "$repo"
  orphan="$(git commit-tree "$(git mktree < /dev/null)" -m "orphan")"
  git push -q origin "${orphan}:refs/heads/orphan" "${orphan}:refs/heads/rel+v1#x"
  git branch -q 'rel+v2#x' "$orphan"
)
out="$workdir/base-ref-orphan.out"
run_detect_setting "$repo" "$out" GITHUB_BASE_REF=orphan
assert_field "GITHUB_BASE_REF with no merge-base falls back to full" scope full "$out"
assert_field "GITHUB_BASE_REF with no merge-base records no_merge_base with the name" reason "no_merge_base:refs/remotes/origin/orphan" "$out"
assert_field "GITHUB_BASE_REF with no merge-base: docs_only is unknown" docs_only unknown "$out"

out="$workdir/base-ref-odd-orphan-origin.out"
run_detect_setting "$repo" "$out" "GITHUB_BASE_REF=rel+v1#x"
assert_field "GITHUB_BASE_REF with odd characters and no merge-base on origin records no_merge_base:refs/remotes/origin/GITHUB_BASE_REF" reason "no_merge_base:refs/remotes/origin/GITHUB_BASE_REF" "$out"
assert_output_lacks "GITHUB_BASE_REF with odd characters and no merge-base on origin leaves the value out of the output" "rel+v1" "$out"

out="$workdir/base-ref-odd-orphan-local.out"
run_detect_setting "$repo" "$out" "GITHUB_BASE_REF=rel+v2#x"
assert_field "GITHUB_BASE_REF with odd characters and no merge-base locally records no_merge_base:refs/heads/GITHUB_BASE_REF" reason "no_merge_base:refs/heads/GITHUB_BASE_REF" "$out"
assert_output_lacks "GITHUB_BASE_REF with odd characters and no merge-base locally leaves the value out of the output" "rel+v2" "$out"

out="$workdir/base-ref-empty.out"
run_detect_setting "$repo" "$out" GITHUB_BASE_REF=
assert_field "empty GITHUB_BASE_REF is as unset: the base is origin/main" languages "golang python" "$out"

# 51. With GITHUB_BASE_REF set, ralph is not asked for [verify].base, not even
#     one that would fail; with it empty, [verify].base is read as before.
#     The fixture is case 11's.
fake_ralph_reset "$FAKE_BIN"
fake_ralph_fail "$FAKE_BIN" verify.base 3 "config get: not expected"
out="$workdir/base-ref-no-ralph.out"
run_detect_setting "$cfg_repo" "$out" GITHUB_BASE_REF=main
assert_calls "GITHUB_BASE_REF set: ralph is not asked" ""
assert_field "GITHUB_BASE_REF=main diffs against origin/main" languages "golang python" "$out"
assert_file_empty "GITHUB_BASE_REF set: nothing on stderr" "$err"

base_value HEAD~1
out="$workdir/base-ref-empty-setting.out"
run_detect_setting "$cfg_repo" "$out" GITHUB_BASE_REF=
assert_calls "empty GITHUB_BASE_REF: [verify].base is read, in the root" "$(root_call "$cfg_repo")"
assert_field "empty GITHUB_BASE_REF: [verify].base HEAD~1 applies" languages golang "$out"

# 52. A failed listing never leaves a partial list (AC11). A git first on
#     PATH runs the real git, and for a call whose arguments match the
#     pattern $FAKE_GIT_FAIL it logs the call and exits 128 after the real
#     git printed its output. In the fixture each of the four listing
#     commands prints docs paths only (a committed README.md change, then an
#     unstaged, a staged, and an untracked docs file), so without the check
#     each failing run below would be docs only. The detector runs each
#     listing as git -c core.quotePath=false, so the patterns start with it.
real_git="$(command -v git)"
fake_git_bin="$workdir/fake-git-bin"
fake_git_log="$workdir/fake-git.log"
mkdir -p "$fake_git_bin"
{
  printf '#!/bin/sh\n'
  printf 'real_git="%s"\n' "$real_git"
  cat <<'EOF'
case "$*" in
  $FAKE_GIT_FAIL)
    printf '%s\n' "$*" >> "$FAKE_GIT_LOG"
    "$real_git" "$@"
    exit 128
    ;;
esac
exec "$real_git" "$@"
EOF
} > "$fake_git_bin/git"
chmod +x "$fake_git_bin/git"

tmp_check="$workdir/tmp-check"
mkdir -p "$tmp_check"

repo="$(make_repo)"
(
  cd "$repo"
  mkdir -p docs
  printf '# a\n' > docs/a.md
  printf '# b\n' > docs/b.md
  git add docs
  git commit -q -m "docs"
  git checkout -q -b feature
  printf 'committed\n' >> README.md
  git add README.md
  git commit -q -m "docs change"
  printf 'unstaged\n' >> docs/a.md
  printf 'staged\n' >> docs/b.md
  git add docs/b.md
  printf '# new\n' > docs/new.md
)
out="$workdir/diff-failed-control.out"
run_detect_fake_git "$repo" "$out" none
assert_field "control: with no failing call the fixture is docs only" reason docs_only "$out"
assert_file_empty "control: the fake git failed no call" "$fake_git_log"

listing_git='-c core.quotePath=false'
for listing in committed unstaged staged untracked; do
  case "$listing" in
    committed) fail_args="$listing_git diff --name-only * HEAD" ;;
    unstaged) fail_args="$listing_git diff --name-only" ;;
    staged) fail_args="$listing_git diff --name-only --cached" ;;
    untracked) fail_args="$listing_git ls-files --others --exclude-standard" ;;
  esac
  out="$workdir/diff-failed-$listing.out"
  run_detect_fake_git "$repo" "$out" "$fail_args"
  failed_calls="$(cat "$fake_git_log")"
  failed_count="$(grep -c '' "$fake_git_log" || true)"
  case "$failed_count:$listing:$failed_calls" in
    "1:committed:$listing_git diff --name-only "[0-9a-f]*" HEAD" | "1:unstaged:$listing_git diff --name-only" | "1:staged:$listing_git diff --name-only --cached" | "1:untracked:$listing_git ls-files --others --exclude-standard")
      record_pass "fake git fixture ($listing): that listing command alone failed, once"
      ;;
    *)
      record_fail "fake git fixture ($listing): that listing command alone failed, once (log '$failed_calls')"
      ;;
  esac
  assert_field "failed $listing listing falls back to full" scope full "$out"
  assert_field "failed $listing listing records diff_failed" reason diff_failed "$out"
  assert_field "failed $listing listing: docs_only is unknown" docs_only unknown "$out"
  assert_field "failed $listing listing selects no languages" languages "" "$out"
  assert_dir_empty "failed $listing listing leaves no temporary file" "$tmp_check"
done

# 53. A deleted path is scaffold when the manifest at the merge-base lists
#     it (AC12): ralph upgrade removes an obsolete core file together with
#     its entry. A deleted path in neither manifest is not scaffold, and a
#     path that still exists is looked up in the working tree's manifest
#     only.
repo="$(make_repo)"
(
  cd "$repo"
  mkdir -p scripts
  printf '#!/bin/sh\n' > scripts/old-tool.sh
  printf '#!/bin/sh\n' > scripts/own-tool.sh
  printf '#!/bin/sh\n' > scripts/kept-tool.sh
  write_manifest .ralph/manifest.toml \
    "[files.'scripts/kept-tool.sh']" \
    "[files.'scripts/old-tool.sh']"
  git add -A
  git commit -q -m "scaffold"
  git checkout -q -b feature
  git rm -q scripts/old-tool.sh
  write_manifest .ralph/manifest.toml "[files.'scripts/kept-tool.sh']"
  git add .ralph/manifest.toml
  git commit -q -m "remove old-tool.sh and its entry"
)
out="$workdir/deleted-scaffold.out"
(cd "$repo" && TMPDIR="$tmp_check" "$DETECT") > "$out"
assert_field "deleted scaffold file with its entry removed is docs only" docs_only true "$out"
assert_field "deleted scaffold file: the reason stays unclassified" reason "unclassified:.ralph/manifest.toml" "$out"
assert_dir_empty "a run that reads both manifests leaves no temporary file" "$tmp_check"

(cd "$repo" && git rm -q scripts/own-tool.sh)
out="$workdir/deleted-own.out"
run_detect "$repo" "$out"
assert_field "deleted file in neither manifest is not docs only" docs_only false "$out"
assert_field "deleted file in neither manifest is the first non-docs path" first_non_docs scripts/own-tool.sh "$out"

(
  cd "$repo"
  git checkout -q HEAD -- scripts/own-tool.sh
  write_manifest .ralph/manifest.toml
  printf 'exit 0\n' >> scripts/kept-tool.sh
)
out="$workdir/kept-unlisted.out"
run_detect "$repo" "$out"
assert_field "an existing file listed only at the merge-base is not docs only" docs_only false "$out"
assert_field "an existing file listed only at the merge-base is the first non-docs path" first_non_docs scripts/kept-tool.sh "$out"

# 54. No git on PATH: full with reason git_not_found, docs_only unknown. PATH
#     holds only sh.
no_git_tools="$workdir/no-git-tools"
mkdir -p "$no_git_tools"
printf '#!/bin/sh\nexec "%s" "$@"\n' "$(command -v sh)" > "$no_git_tools/sh"
chmod +x "$no_git_tools/sh"
if (PATH="$no_git_tools" && command -v git >/dev/null 2>&1); then
  record_fail "no-git fixture: the sh-only PATH has no git"
else
  record_pass "no-git fixture: the sh-only PATH has no git"
fi
out="$workdir/no-git-on-path.out"
(cd "$cfg_repo" && PATH="$no_git_tools" "$DETECT") > "$out" 2> "$err"
assert_field "no git on PATH falls back to full" scope full "$out"
assert_field "no git on PATH records git_not_found" reason git_not_found "$out"
assert_field "no git on PATH: docs_only is unknown" docs_only unknown "$out"

# 55. The listings run as git -c core.quotePath=false, so a path with
#     non-ASCII characters reaches the docs table and the manifest as it is:
#     a committed docs/日本語.md is docs only, a manifest-listed
#     scripts/日本語.sh is scaffold, and a non-ASCII code file is the first
#     non-docs path, printed raw. (git's default prints each in double
#     quotes as "...\346\227\245...", which matches neither.) A path that
#     holds a double quote stays quoted, so it is not a docs file.
repo="$(make_repo)"
(
  cd "$repo"
  write_manifest .ralph/manifest.toml "[files.'scripts/日本語.sh']"
  git add -A
  git commit -q -m "scaffold"
  git checkout -q -b feature
  mkdir -p docs
  printf '# 日本語\n' > docs/日本語.md
  git add docs
  git commit -q -m "a non-ASCII docs file"
)
out="$workdir/non-ascii-docs.out"
run_detect "$repo" "$out"
assert_field "committed docs/日本語.md: docs_only is true" docs_only true "$out"
assert_field "committed docs/日本語.md: the reason is docs_only" reason docs_only "$out"

(
  cd "$repo"
  mkdir -p scripts
  printf '#!/bin/sh\n' > scripts/日本語.sh
  git add scripts
  git commit -q -m "a manifest-listed non-ASCII script"
)
out="$workdir/non-ascii-scaffold.out"
run_detect "$repo" "$out"
assert_field "manifest-listed scripts/日本語.sh is scaffold: docs_only is true" docs_only true "$out"
assert_field "manifest-listed scripts/日本語.sh: the reason names it raw" reason "unclassified:scripts/日本語.sh" "$out"

mkdir -p "$repo/src"
printf 'export {};\n' > "$repo/src/日本語.js"
out="$workdir/non-ascii-code.out"
run_detect "$repo" "$out"
assert_field "non-ASCII code file is not docs only" docs_only false "$out"
assert_field "non-ASCII code file: first_non_docs names it raw" first_non_docs "src/日本語.js" "$out"

rm -f "$repo/src/日本語.js"
printf '# hi\n' > "$repo/docs/say\"hi\".md"
out="$workdir/quoted-docs.out"
run_detect "$repo" "$out"
assert_field "docs path with a double quote stays quoted: docs_only is false" docs_only false "$out"
assert_field "docs path with a double quote: first_non_docs is the quoted form" first_non_docs '"docs/say\"hi\".md"' "$out"

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
