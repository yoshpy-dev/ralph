#!/usr/bin/env sh
# tests/test-wip-checkpoint-hooks.sh — the two WIP-checkpoint hooks
# (.claude/hooks/session_end_summary.sh and precompact_checkpoint.sh, and their
# templates/base copies) skip only their WIP commit when RALPH_WIP_CHECKPOINT
# is exactly "off".
#
# Each case runs one hook inside a fresh temp git repo, never in this checkout:
# the hook runs git add -A and git commit in its cwd. Git is isolated from the
# user's config, and the repo ignores .harness/state/ and .harness/logs/ as a
# scaffolded project does. For each hook and each copy:
#   1. feature branch, dirty tree (a tracked change and an untracked file),
#      RALPH_WIP_CHECKPOINT unset, "on", "OFF", or empty: one wip commit with
#      the hook's subject that holds both files; the md has no skipped line
#   2. the same tree with "off": HEAD and the index unchanged, the untracked
#      file still untracked, the md ends with exactly one skipped line
#   3. main, dirty tree, unset or "off": no commit, no skipped line
#   4. feature branch, clean tree, unset or "off": no commit, no skipped line
# In every case the hook exits 0, writes its md (title first), and writes no
# .harness/logs/hook-failures.log.
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SKIP_LINE='- WIP commit: skipped (RALPH_WIP_CHECKPOINT=off)'

TMP_ROOT="$(mktemp -d)"
TMP_ROOT_REAL="$(cd "$TMP_ROOT" && pwd -P)"
trap 'rm -r -f "$TMP_ROOT"' EXIT
trap 'exit 1' HUP INT TERM

# Keep git away from the user's config and from any repo named by the caller's
# environment (a leaked GIT_DIR would point the hook's commit at that repo).
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY \
  GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_COMMON_DIR GIT_NAMESPACE GIT_PREFIX \
  GIT_TEMPLATE_DIR GIT_CONFIG GIT_CONFIG_PARAMETERS GIT_CONFIG_COUNT
: > "${TMP_ROOT}/no-ignore"
cat > "${TMP_ROOT}/gitconfig" <<EOF
[core]
	excludesFile = ${TMP_ROOT}/no-ignore
[init]
	defaultBranch = main
EOF
GIT_CONFIG_GLOBAL="${TMP_ROOT}/gitconfig"
GIT_CONFIG_NOSYSTEM=1
export GIT_CONFIG_GLOBAL GIT_CONFIG_NOSYSTEM

_pass=0
_fail=0
_total=0
_repo_count=0

pass() {
  _total=$((_total + 1))
  _pass=$((_pass + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  _total=$((_total + 1))
  _fail=$((_fail + 1))
  printf '  FAIL: %s\n' "$1"
}

# make_repo BRANCH — a fresh repo with one commit on main (.gitignore and
# tracked.txt), then on BRANCH. Sets _repo.
make_repo() {
  _repo_count=$((_repo_count + 1))
  # A counter, not the mode, names the repo: "off" and "OFF" are the same
  # directory on a case-insensitive file system.
  _repo="${TMP_ROOT}/repo${_repo_count}"
  mkdir -p "$_repo"
  git -C "$_repo" init -q
  git -C "$_repo" config user.name 'WIP Hook Test'
  git -C "$_repo" config user.email 'wip-hook-test@example.invalid'
  git -C "$_repo" config commit.gpgsign false
  printf '.harness/state/\n.harness/logs/\n' > "${_repo}/.gitignore"
  printf 'one\n' > "${_repo}/tracked.txt"
  git -C "$_repo" add .gitignore tracked.txt
  git -C "$_repo" commit -q -m 'initial'
  if [ "$1" != main ]; then
    git -C "$_repo" checkout -q -b "$1"
  fi
}

# make_dirty — a tracked change and an untracked file in _repo.
make_dirty() {
  printf 'two\n' >> "${_repo}/tracked.txt"
  printf 'new\n' > "${_repo}/untracked.txt"
}

# guard_temp_repo REPO — stop the whole test unless REPO is under the temp
# root and is the top of its own work tree.
guard_temp_repo() {
  _here="$(cd "$1" && pwd -P)"
  _top="$(cd "$1" && git rev-parse --show-toplevel 2>/dev/null || true)"
  case "$_here" in
    "$TMP_ROOT_REAL"/*) ;;
    *)
      printf 'ABORT: %s is not under the temp root %s\n' "$_here" "$TMP_ROOT_REAL" >&2
      exit 1
      ;;
  esac
  if [ "$_top" != "$_here" ]; then
    printf 'ABORT: %s is not the top of its own repo (top: %s)\n' "$_here" "$_top" >&2
    exit 1
  fi
}

# run_hook HOOK MODE — run HOOK from inside _repo with RALPH_WIP_CHECKPOINT
# unset, empty, or set to MODE. Sets _rc and _out.
run_hook() {
  guard_temp_repo "$_repo"
  _out="${_repo}.out"
  _rc=0
  case "$2" in
    unset) (cd "$_repo" && env -u RALPH_WIP_CHECKPOINT sh "$1") > "$_out" 2>&1 || _rc=$? ;;
    empty) (cd "$_repo" && env RALPH_WIP_CHECKPOINT='' sh "$1") > "$_out" 2>&1 || _rc=$? ;;
    *) (cd "$_repo" && env RALPH_WIP_CHECKPOINT="$2" sh "$1") > "$_out" 2>&1 || _rc=$? ;;
  esac
}

# mode_label MODE — how a case names its RALPH_WIP_CHECKPOINT value.
mode_label() {
  case "$1" in
    unset) printf '%s' 'RALPH_WIP_CHECKPOINT unset' ;;
    empty) printf '%s' "RALPH_WIP_CHECKPOINT=''" ;;
    *) printf '%s' "RALPH_WIP_CHECKPOINT=$1" ;;
  esac
}

show_file() {
  if [ -f "$1" ]; then
    sed 's/^/      | /' "$1"
  else
    printf '      | (%s does not exist)\n' "$1"
  fi
}

# expect_md DESC SKIPS — the hook wrote its md with its title on the first
# line and SKIPS skipped lines; with one, the skipped line is the last line.
expect_md() {
  _md_file="${_repo}/${_md}"
  _skips=0
  _first=''
  _last=''
  if [ -f "$_md_file" ]; then
    _skips="$(grep -c -F -x -- "$SKIP_LINE" "$_md_file" || true)"
    _first="$(head -n 1 "$_md_file")"
    _last="$(tail -n 1 "$_md_file")"
  fi
  if [ -f "$_md_file" ] && [ "$_first" = "$_title" ] && [ "$_skips" = "$2" ] \
    && { [ "$2" = 0 ] || [ "$_last" = "$SKIP_LINE" ]; }; then
    pass "$1: ${_md} written, ${2} skipped line(s)"
  else
    fail "$1: ${_md} first line '${_first}', ${_skips} skipped line(s), want ${2}"
    show_file "$_md_file"
  fi
}

# expect_no_failure_log DESC — the hook logged no WIP commit failure.
expect_no_failure_log() {
  if [ ! -e "${_repo}/.harness/logs/hook-failures.log" ]; then
    pass "$1: no hook-failures.log"
  else
    fail "$1: hook-failures.log written"
    show_file "${_repo}/.harness/logs/hook-failures.log"
  fi
}

# case_commits HOOK MODE — feature branch, dirty tree: one wip commit.
case_commits() {
  make_repo feature/wip-test
  make_dirty
  run_hook "$1" "$2"
  _desc="${_label}, $(mode_label "$2"), dirty feature branch"
  _count="$(git -C "$_repo" rev-list --count HEAD)"
  _got="$(git -C "$_repo" log -1 --format=%s)"
  _status="$(git -C "$_repo" status --porcelain)"
  if [ "$_rc" -eq 0 ] && [ "$_count" = 2 ] && [ "$_got" = "$_subject" ] \
    && [ -z "$_status" ] && git -C "$_repo" cat-file -e HEAD:untracked.txt 2>/dev/null; then
    pass "${_desc}: one commit '${_subject}' with both files"
  else
    fail "${_desc}: exit ${_rc}, ${_count} commits, last subject '${_got}', status '${_status}'"
    show_file "$_out"
  fi
  expect_md "$_desc" 0
  expect_no_failure_log "$_desc"
}

# case_off HOOK — feature branch, dirty tree, "off": no commit, no git add.
case_off() {
  make_repo feature/wip-test
  make_dirty
  _before="$(git -C "$_repo" rev-parse HEAD)"
  run_hook "$1" off
  _desc="${_label}, $(mode_label off), dirty feature branch"
  _after="$(git -C "$_repo" rev-parse HEAD)"
  _status="$(git -C "$_repo" status --porcelain)"
  _want_status="$(printf ' M tracked.txt\n?? untracked.txt')"
  if [ "$_rc" -eq 0 ] && [ "$_after" = "$_before" ] \
    && git -C "$_repo" diff --cached --quiet && [ "$_status" = "$_want_status" ]; then
    pass "${_desc}: HEAD and index unchanged, untracked file still untracked"
  else
    fail "${_desc}: exit ${_rc}, HEAD ${_before} -> ${_after}, status '${_status}'"
    show_file "$_out"
  fi
  expect_md "$_desc" 1
  expect_no_failure_log "$_desc"
}

# case_no_commit HOOK BRANCH DIRTY MODE — a case that commits nothing with any
# value: main with a dirty tree, or a feature branch with a clean tree.
case_no_commit() {
  make_repo "$2"
  if [ "$3" = dirty ]; then
    make_dirty
  fi
  _before="$(git -C "$_repo" rev-parse HEAD)"
  _status_before="$(git -C "$_repo" status --porcelain)"
  run_hook "$1" "$4"
  _desc="${_label}, $(mode_label "$4"), ${3} ${2}"
  _after="$(git -C "$_repo" rev-parse HEAD)"
  _status="$(git -C "$_repo" status --porcelain)"
  if [ "$_rc" -eq 0 ] && [ "$_after" = "$_before" ] \
    && git -C "$_repo" diff --cached --quiet && [ "$_status" = "$_status_before" ]; then
    pass "${_desc}: no commit, tree unchanged"
  else
    fail "${_desc}: exit ${_rc}, HEAD ${_before} -> ${_after}, status '${_status}'"
    show_file "$_out"
  fi
  expect_md "$_desc" 0
  expect_no_failure_log "$_desc"
}

for _hook in session_end_summary precompact_checkpoint; do
  case "$_hook" in
    session_end_summary)
      _subject='wip: checkpoint before session end'
      _md='.harness/state/session-end.md'
      _title='# Session end summary'
      ;;
    precompact_checkpoint)
      _subject='wip: checkpoint before context compaction'
      _md='.harness/state/precompact-checkpoint.md'
      _title='# Pre-compact checkpoint'
      ;;
  esac
  for _copy in root template; do
    case "$_copy" in
      root) _path="${PROJECT_ROOT}/.claude/hooks/${_hook}.sh" ;;
      template) _path="${PROJECT_ROOT}/templates/base/.claude/hooks/${_hook}.sh" ;;
    esac
    _label="${_copy} ${_hook}.sh"
    printf '==> %s (%s)\n' "$_label" "$_path"
    if [ ! -f "$_path" ]; then
      fail "${_label}: ${_path} does not exist"
      continue
    fi
    for _mode in unset on OFF empty; do
      case_commits "$_path" "$_mode"
    done
    case_off "$_path"
    case_no_commit "$_path" main dirty unset
    case_no_commit "$_path" main dirty off
    case_no_commit "$_path" feature/wip-test clean unset
    case_no_commit "$_path" feature/wip-test clean off
  done
done

printf '\nwip-checkpoint hook tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
