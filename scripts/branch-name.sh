#!/usr/bin/env sh
set -eu

ALLOWED_TYPES="feat fix docs chore refactor test ci build perf release security"
ALLOWED_TYPES_RE="feat|fix|docs|chore|refactor|test|ci|build|perf|release|security"

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/branch-name.sh from-plan <plan-path>
  ./scripts/branch-name.sh validate <branch-name>
  ./scripts/branch-name.sh type <branch-name>
  ./scripts/branch-name.sh title-prefix <branch-name>
  ./scripts/branch-name.sh allowed-types

Branch names must follow:
  <type>/<slug>
  <type>/<issue>/<slug>
USAGE
}

trim() {
  printf '%s' "$1" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//'
}

is_allowed_type() {
  _type="$1"
  for _allowed in $ALLOWED_TYPES; do
    if [ "$_type" = "$_allowed" ]; then
      return 0
    fi
  done
  return 1
}

field_value() {
  _field="$1"
  _file="$2"
  awk -v want="$(printf '%s' "$_field" | tr '[:upper:]' '[:lower:]')" '
    {
      line = $0
      sub(/^[[:space:]]*[-*][[:space:]]*/, "", line)
      key = line
      sub(/:.*/, "", key)
      key = tolower(key)
      if (key == want) {
        sub(/^[^:]*:[[:space:]]*/, "", line)
        print line
        exit
      }
    }
  ' "$_file"
}

plan_manifest() {
  _path="$1"
  if [ -d "$_path" ]; then
    _path="${_path%/}/_manifest.md"
  fi
  if [ ! -f "$_path" ]; then
    printf 'branch-name: plan file not found: %s\n' "$_path" >&2
    exit 1
  fi
  printf '%s\n' "$_path"
}

plan_slug() {
  _path="$1"
  _base="$(basename "$_path" .md)"
  if [ "$_base" = "_manifest" ]; then
    _base="$(basename "$(dirname "$_path")")"
  fi
  printf '%s' "$_base" |
    sed -E 's/^[0-9]{4}-[0-9]{2}-[0-9]{2}-//' |
    tr '[:upper:]' '[:lower:]' |
    sed -E 's/[^a-z0-9._-]+/-/g;s/-+/-/g;s/^-//;s/-$//'
}

issue_number() {
  _raw="$(trim "$1")"
  case "$_raw" in
    ""|N/A|n/a|none|None|NONE) return 0 ;;
  esac
  printf '%s' "$_raw" | grep -oE '#?[0-9]+' | head -1 | tr -d '#'
}

from_plan() {
  _plan="$(plan_manifest "$1")"
  _type="$(field_value "Type" "$_plan" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_-].*$//' || true)"
  _type="$(trim "$_type")"
  if ! is_allowed_type "$_type"; then
    printf 'branch-name: invalid or missing Type in %s: %s\n' "$_plan" "${_type:-<missing>}" >&2
    printf 'branch-name: allowed types: %s\n' "$ALLOWED_TYPES" >&2
    exit 1
  fi

  _slug="$(plan_slug "$_plan")"
  if [ -z "$_slug" ]; then
    printf 'branch-name: could not derive slug from %s\n' "$_plan" >&2
    exit 1
  fi

  _issue_raw="$(field_value "Related issue" "$_plan" || true)"
  _issue="$(issue_number "$_issue_raw")"
  if [ -n "$_issue" ]; then
    printf '%s/%s/%s\n' "$_type" "$_issue" "$_slug"
  else
    printf '%s/%s\n' "$_type" "$_slug"
  fi
}

validate() {
  _branch="$1"
  if printf '%s' "$_branch" | grep -Eq "^(${ALLOWED_TYPES_RE})/([0-9]+/)?[a-z0-9][a-z0-9._-]*$"; then
    return 0
  fi
  printf 'branch-name: invalid branch name: %s\n' "$_branch" >&2
  printf 'branch-name: expected <type>/<slug> or <type>/<issue>/<slug>; allowed types: %s\n' "$ALLOWED_TYPES" >&2
  return 1
}

branch_type() {
  _branch="$1"
  validate "$_branch" >/dev/null || return 1
  printf '%s\n' "${_branch%%/*}"
}

title_prefix() {
  _type="$(branch_type "$1")" || return 1
  printf '%s:\n' "$_type"
}

cmd="${1:-}"
case "$cmd" in
  from-plan)
    [ "${2:-}" != "" ] || { usage >&2; exit 1; }
    from_plan "$2"
    ;;
  validate)
    [ "${2:-}" != "" ] || { usage >&2; exit 1; }
    validate "$2"
    ;;
  type)
    [ "${2:-}" != "" ] || { usage >&2; exit 1; }
    branch_type "$2"
    ;;
  title-prefix)
    [ "${2:-}" != "" ] || { usage >&2; exit 1; }
    title_prefix "$2"
    ;;
  allowed-types)
    printf '%s\n' "$ALLOWED_TYPES"
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
