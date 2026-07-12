#!/usr/bin/env bash
set -euo pipefail

_WORKTREE_SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=ralph-common.sh
. "${_WORKTREE_SCRIPT_DIR}/ralph-common.sh"

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/ralph-worktree.sh default-branch
  ./scripts/ralph-worktree.sh state-root
  ./scripts/ralph-worktree.sh state-path <task-id>
  ./scripts/ralph-worktree.sh validate-clean-base [base-branch]
  ./scripts/ralph-worktree.sh ensure --id <task-id> --kind <kind> --branch <branch> --path <path> [--base <branch>] [--canonical-ref <ref>] [--plan-path <path>] [--cleanup-policy <policy>]
  ./scripts/ralph-worktree.sh current
  ./scripts/ralph-worktree.sh resume --id <task-id>
  ./scripts/ralph-worktree.sh cleanup --id <task-id> [--force-branch]
  ./scripts/ralph-worktree.sh gc [--prune]

Task worktree state is local orchestration state. It is stored under:
  $(git rev-parse --git-common-dir)/ralph/worktrees/
USAGE
}

die() {
  printf 'ralph-worktree: %s\n' "$*" >&2
  exit 1
}

require_git() {
  command -v git >/dev/null 2>&1 || die "git not found"
  git rev-parse --git-dir >/dev/null 2>&1 || die "not inside a git repository"
}

require_jq() {
  command -v jq >/dev/null 2>&1 || die "jq not found"
}

repo_root() {
  require_git
  git rev-parse --show-toplevel
}

git_common_dir() {
  require_git
  local dir
  dir="$(git rev-parse --git-common-dir)"
  case "$dir" in
    /*) ;;
    *) dir="$(pwd -P)/$dir" ;;
  esac
  (cd "$dir" && pwd -P)
}

state_root() {
  printf '%s/ralph/worktrees\n' "$(git_common_dir)"
}

sanitize_id() {
  printf '%s' "$1" |
    tr '[:upper:]' '[:lower:]' |
    sed -E 's/[^a-z0-9._-]+/-/g;s/-+/-/g;s/^-//;s/-$//'
}

state_path() {
  local id safe
  id="${1:-}"
  [ -n "$id" ] || die "state-path requires <task-id>"
  safe="$(sanitize_id "$id")"
  [ -n "$safe" ] || die "task id did not contain any safe characters: $id"
  printf '%s/%s.json\n' "$(state_root)" "$safe"
}

abs_from_repo_root() {
  local path root
  path="$1"
  case "$path" in
    /*) printf '%s\n' "$path" ;;
    *)
      root="$(repo_root)"
      printf '%s/%s\n' "$root" "$path"
      ;;
  esac
}

# default_branch is provided by ralph-common.sh (sourced above).

validate_clean_base() {
  local base current dirty
  base="${1:-$(default_branch)}"
  git rev-parse --verify "${base}^{commit}" >/dev/null 2>&1 ||
    die "base branch not found: $base"
  current="$(git branch --show-current)"
  [ "$current" = "$base" ] ||
    die "must start from clean default branch '$base' (current: ${current:-detached})"
  dirty="$(git status --porcelain)"
  [ -z "$dirty" ] || die "base branch '$base' has uncommitted changes"
}

json_get() {
  local file expr
  file="$1"
  expr="$2"
  jq -r "$expr // empty" "$file"
}

write_state() {
  require_jq
  local file tmp
  file="$1"
  tmp="${file}.tmp"
  mkdir -p "$(dirname "$file")"
  jq -n \
    --arg id "$STATE_ID" \
    --arg kind "$STATE_KIND" \
    --arg worktree_path "$STATE_WORKTREE_PATH" \
    --arg branch "$STATE_BRANCH" \
    --arg base_branch "$STATE_BASE_BRANCH" \
    --arg base_sha "$STATE_BASE_SHA" \
    --arg canonical_ref "$STATE_CANONICAL_REF" \
    --arg plan_path "$STATE_PLAN_PATH" \
    --arg cleanup_policy "$STATE_CLEANUP_POLICY" \
    --arg created_at "$STATE_CREATED_AT" \
    --arg last_seen_at "$STATE_LAST_SEEN_AT" \
    '{
      id: $id,
      kind: $kind,
      worktree_path: $worktree_path,
      branch: $branch,
      base_branch: $base_branch,
      base_sha: $base_sha,
      canonical_ref: $canonical_ref,
      plan_path: $plan_path,
      cleanup_policy: $cleanup_policy,
      created_at: $created_at,
      last_seen_at: $last_seen_at
    }' > "$tmp"
  mv "$tmp" "$file"
}

update_last_seen() {
  require_jq
  local file tmp now
  file="$1"
  now="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  tmp="${file}.tmp"
  jq --arg now "$now" '.last_seen_at = $now' "$file" > "$tmp"
  mv "$tmp" "$file"
}

ensure_worktree() {
  require_jq
  local id="" kind="" branch="" path="" base="" canonical_ref="" plan_path="" cleanup_policy="pr-success"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --id) id="${2:-}"; shift 2 ;;
      --kind) kind="${2:-}"; shift 2 ;;
      --branch) branch="${2:-}"; shift 2 ;;
      --path) path="${2:-}"; shift 2 ;;
      --base) base="${2:-}"; shift 2 ;;
      --canonical-ref) canonical_ref="${2:-}"; shift 2 ;;
      --plan-path) plan_path="${2:-}"; shift 2 ;;
      --cleanup-policy) cleanup_policy="${2:-}"; shift 2 ;;
      *) die "unknown ensure option: $1" ;;
    esac
  done

  [ -n "$id" ] || die "ensure requires --id"
  [ -n "$kind" ] || die "ensure requires --kind"
  [ -n "$branch" ] || die "ensure requires --branch"
  [ -n "$path" ] || die "ensure requires --path"
  base="${base:-$(default_branch)}"

  local abs_path file existing_path existing_branch existing_kind base_sha now
  abs_path="$(abs_from_repo_root "$path")"
  file="$(state_path "$id")"

  if [ -f "$file" ]; then
    existing_path="$(json_get "$file" '.worktree_path')"
    existing_branch="$(json_get "$file" '.branch')"
    existing_kind="$(json_get "$file" '.kind')"
    if [ "$existing_path" = "$abs_path" ] &&
       [ "$existing_branch" = "$branch" ] &&
       [ "$existing_kind" = "$kind" ] &&
       [ -d "$abs_path" ] &&
       git -C "$abs_path" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      update_last_seen "$file"
      printf '%s\n' "$abs_path"
      return 0
    fi
    die "state collision for id '$id': $file"
  fi

  validate_clean_base "$base"

  if [ -e "$abs_path" ]; then
    die "worktree path already exists without matching state: $abs_path"
  fi
  if git show-ref --verify --quiet "refs/heads/${branch}"; then
    die "branch already exists without matching state: $branch"
  fi

  mkdir -p "$(dirname "$abs_path")"
  git worktree add "$abs_path" -b "$branch" "$base" >&2

  base_sha="$(git rev-parse "${base}^{commit}")"
  now="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  STATE_ID="$id"
  STATE_KIND="$kind"
  STATE_WORKTREE_PATH="$abs_path"
  STATE_BRANCH="$branch"
  STATE_BASE_BRANCH="$base"
  STATE_BASE_SHA="$base_sha"
  STATE_CANONICAL_REF="$canonical_ref"
  STATE_PLAN_PATH="$plan_path"
  STATE_CLEANUP_POLICY="$cleanup_policy"
  STATE_CREATED_AT="$now"
  STATE_LAST_SEEN_AT="$now"
  write_state "$file"
  printf '%s\n' "$abs_path"
}

resume_worktree() {
  require_jq
  local id file path branch
  id=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --id) id="${2:-}"; shift 2 ;;
      *) die "unknown resume option: $1" ;;
    esac
  done
  [ -n "$id" ] || die "resume requires --id"
  file="$(state_path "$id")"
  [ -f "$file" ] || die "state not found for id: $id"
  path="$(json_get "$file" '.worktree_path')"
  branch="$(json_get "$file" '.branch')"
  [ -d "$path" ] || die "recorded worktree path is missing: $path"
  [ "$(git -C "$path" branch --show-current)" = "$branch" ] ||
    die "recorded worktree branch mismatch: $path"
  update_last_seen "$file"
  printf '%s\n' "$path"
}

current_state() {
  require_jq
  local root current file path
  root="$(state_root)"
  current="$(repo_root)"
  [ -d "$root" ] || return 1
  for file in "$root"/*.json; do
    [ -f "$file" ] || continue
    path="$(json_get "$file" '.worktree_path')"
    if [ "$path" = "$current" ]; then
      printf '%s\n' "$file"
      return 0
    fi
  done
  return 1
}

cleanup_worktree() {
  require_jq
  local id="" force_branch=0 file path branch common dirty
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --id) id="${2:-}"; shift 2 ;;
      --force-branch) force_branch=1; shift ;;
      *) die "unknown cleanup option: $1" ;;
    esac
  done
  [ -n "$id" ] || die "cleanup requires --id"
  file="$(state_path "$id")"
  [ -f "$file" ] || die "state not found for id: $id"
  path="$(json_get "$file" '.worktree_path')"
  branch="$(json_get "$file" '.branch')"
  common="$(git_common_dir)"

  if [ -d "$path" ]; then
    dirty="$(git -C "$path" status --porcelain)"
    [ -z "$dirty" ] || die "refusing cleanup; worktree has uncommitted changes: $path"
    git --git-dir="$common" worktree remove "$path"
  fi

  if git --git-dir="$common" show-ref --verify --quiet "refs/heads/${branch}"; then
    if [ "$force_branch" -eq 1 ]; then
      git --git-dir="$common" branch -D "$branch"
    else
      git --git-dir="$common" branch -d "$branch"
    fi
  fi

  rm -f "$file"
}

gc_worktrees() {
  require_jq
  local prune=0 root file path branch stale count=0
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --prune) prune=1; shift ;;
      *) die "unknown gc option: $1" ;;
    esac
  done
  root="$(state_root)"
  [ -d "$root" ] || return 0
  for file in "$root"/*.json; do
    [ -f "$file" ] || continue
    path="$(json_get "$file" '.worktree_path')"
    branch="$(json_get "$file" '.branch')"
    stale=0
    [ -d "$path" ] || stale=1
    if [ "$stale" -eq 1 ]; then
      count=$((count + 1))
      printf 'STALE %s branch=%s path=%s\n' "$file" "$branch" "$path"
      [ "$prune" -eq 1 ] && rm -f "$file"
    fi
  done
  if [ "$count" -eq 0 ]; then
    printf 'No stale ralph worktree state.\n'
  fi
  return 0
}

cmd="${1:-}"
case "$cmd" in
  default-branch)
    shift
    [ "$#" -eq 0 ] || die "default-branch takes no arguments"
    default_branch
    ;;
  state-root)
    shift
    [ "$#" -eq 0 ] || die "state-root takes no arguments"
    state_root
    ;;
  state-path)
    shift
    state_path "${1:-}"
    ;;
  validate-clean-base)
    shift
    # Pass the optional argument through as-is; validate_clean_base resolves
    # default_branch internally when no argument is given. Resolving here AND
    # in the function produced two identical error messages on failure.
    validate_clean_base "${1:-}"
    ;;
  ensure)
    shift
    ensure_worktree "$@"
    ;;
  resume)
    shift
    resume_worktree "$@"
    ;;
  current)
    shift
    [ "$#" -eq 0 ] || die "current takes no arguments"
    current_state
    ;;
  cleanup)
    shift
    cleanup_worktree "$@"
    ;;
  gc)
    shift
    gc_worktrees "$@"
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
