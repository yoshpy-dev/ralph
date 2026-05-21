#!/usr/bin/env sh
set -eu

# Detect which language packs are relevant to the current git diff.
#
# Output is a small key=value contract for scripts/run-verify.sh:
#   scope=changed|full
#   reason=<machine-readable reason>
#   docs_only=true|false
#   languages=<space-separated language pack names>
#   <language>_roots=<space-separated project roots, when changed scope can narrow safely>
#
# A "full" scope is intentionally conservative. Shared harness files,
# unclassified code-like files, or missing diff context fall back to the
# repository-wide language detector.

languages=""
typescript_roots=""
python_roots=""
rust_roots=""
golang_roots=""
dart_roots=""
terraform_roots=""

emit_lang() {
  lang="$1"
  case " $languages " in
    *" $lang "*) ;;
    *)
      if [ -n "$languages" ]; then
        languages="$languages $lang"
      else
        languages="$lang"
      fi
      ;;
  esac
}

emit_root() {
  lang="$1"
  root="$2"
  [ -n "$root" ] || return 0
  root="${root#./}"
  [ -n "$root" ] || root="."

  case "$lang" in
    typescript)
      case " $typescript_roots " in
        *" $root "*) ;;
        *) typescript_roots="${typescript_roots:+$typescript_roots }$root" ;;
      esac
      ;;
    python)
      case " $python_roots " in
        *" $root "*) ;;
        *) python_roots="${python_roots:+$python_roots }$root" ;;
      esac
      ;;
    rust)
      case " $rust_roots " in
        *" $root "*) ;;
        *) rust_roots="${rust_roots:+$rust_roots }$root" ;;
      esac
      ;;
    golang)
      case " $golang_roots " in
        *" $root "*) ;;
        *) golang_roots="${golang_roots:+$golang_roots }$root" ;;
      esac
      ;;
    dart)
      case " $dart_roots " in
        *" $root "*) ;;
        *) dart_roots="${dart_roots:+$dart_roots }$root" ;;
      esac
      ;;
    terraform)
      case " $terraform_roots " in
        *" $root "*) ;;
        *) terraform_roots="${terraform_roots:+$terraform_roots }$root" ;;
      esac
      ;;
  esac
}

emit_result() {
  scope="$1"
  reason="$2"
  docs_only="$3"
  langs="${4:-}"

  printf 'scope=%s\n' "$scope"
  printf 'reason=%s\n' "$reason"
  printf 'docs_only=%s\n' "$docs_only"
  printf 'languages=%s\n' "$langs"
  [ -n "$typescript_roots" ] && printf 'typescript_roots=%s\n' "$typescript_roots"
  [ -n "$python_roots" ] && printf 'python_roots=%s\n' "$python_roots"
  [ -n "$rust_roots" ] && printf 'rust_roots=%s\n' "$rust_roots"
  [ -n "$golang_roots" ] && printf 'golang_roots=%s\n' "$golang_roots"
  [ -n "$dart_roots" ] && printf 'dart_roots=%s\n' "$dart_roots"
  [ -n "$terraform_roots" ] && printf 'terraform_roots=%s\n' "$terraform_roots"
  return 0
}

fallback_full() {
  reason="$1"
  emit_result full "$reason" false ""
  exit 0
}

if ! command -v git >/dev/null 2>&1; then
  fallback_full "git_not_found"
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  fallback_full "no_git_repository"
fi

base_ref="${RALPH_VERIFY_BASE:-}"
if [ -z "$base_ref" ]; then
  upstream="$(git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null || true)"
  if [ -n "$upstream" ]; then
    base_ref="$upstream"
  elif git show-ref --verify --quiet refs/remotes/origin/main; then
    base_ref="origin/main"
  elif git show-ref --verify --quiet refs/heads/main; then
    base_ref="main"
  elif git show-ref --verify --quiet refs/remotes/origin/master; then
    base_ref="origin/master"
  elif git show-ref --verify --quiet refs/heads/master; then
    base_ref="master"
  fi
fi

if [ -z "$base_ref" ]; then
  fallback_full "no_diff_base"
fi

merge_base="$(git merge-base HEAD "$base_ref" 2>/dev/null || true)"
if [ -z "$merge_base" ]; then
  fallback_full "no_merge_base:$base_ref"
fi

tmp_files="$(mktemp "${TMPDIR:-/tmp}/ralph-changed-files.XXXXXX")"
trap 'rm -f "$tmp_files"' EXIT HUP INT TERM

{
  git diff --name-only "$merge_base" HEAD 2>/dev/null || true
  git diff --name-only 2>/dev/null || true
  git diff --name-only --cached 2>/dev/null || true
  git ls-files --others --exclude-standard 2>/dev/null || true
} | sort -u > "$tmp_files"

if [ ! -s "$tmp_files" ]; then
  emit_result changed "no_changes" true ""
  exit 0
fi

is_docs_file() {
  file="$1"
  case "$file" in
    .harness/*|docs/*|templates/base/docs/*|README.md|AGENTS.md|CLAUDE.md|*.md)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

is_shared_full_file() {
  file="$1"
  case "$file" in
    scripts/run-verify.sh|scripts/run-static-verify.sh|scripts/run-test.sh|scripts/detect-languages.sh|scripts/detect-changed-languages.sh)
      return 0
      ;;
    templates/base/scripts/run-verify.sh|templates/base/scripts/run-static-verify.sh|templates/base/scripts/run-test.sh|templates/base/scripts/detect-languages.sh|templates/base/scripts/detect-changed-languages.sh)
      return 0
      ;;
    packs/languages/*|templates/packs/*)
      return 0
      ;;
    .github/workflows/*|templates/base/.github/workflows/*)
      return 0
      ;;
    Makefile|Taskfile|Taskfile.*|Justfile|Dockerfile|Dockerfile.*|docker-compose.yml|docker-compose.yaml|compose.yml|compose.yaml)
      return 0
      ;;
    .tool-versions|mise.toml|.mise.toml|lefthook.yml|lefthook.yaml)
      return 0
      ;;
    *.proto|*.graphql|*.graphqls|openapi.*|schemas/*|schema/*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

classify_language() {
  file="$1"
  base="${file##*/}"
  case "$base" in
    *.go|go.mod|go.sum)
      printf 'golang\n'
      ;;
    *.ts|*.tsx|package.json|package-lock.json|pnpm-lock.yaml|yarn.lock|tsconfig.json|tsconfig.*.json|vite.config.*|vitest.config.*|jest.config.*|next.config.*|eslint.config.*|.eslintrc|.eslintrc.*)
      printf 'typescript\n'
      ;;
    *.py|pyproject.toml|requirements.txt|requirements-*.txt|setup.cfg|tox.ini|poetry.lock)
      printf 'python\n'
      ;;
    *.rs|Cargo.toml|Cargo.lock)
      printf 'rust\n'
      ;;
    *.dart|pubspec.yaml|pubspec.lock|analysis_options.yaml)
      printf 'dart\n'
      ;;
    *.tf|*.tofu|*.tftest.hcl|.terraform.lock.hcl)
      printf 'terraform\n'
      ;;
  esac
}

file_dir() {
  file="$1"
  case "$file" in
    */*) dirname "$file" ;;
    *) printf '.\n' ;;
  esac
}

has_name_in_dir() {
  dir="$1"
  pattern="$2"
  marker="$(find "$dir" -maxdepth 1 -type f -name "$pattern" -print -quit 2>/dev/null)"
  [ -n "$marker" ]
}

has_project_marker() {
  lang="$1"
  dir="$2"

  case "$lang" in
    typescript)
      [ -f "$dir/package.json" ] ||
        [ -f "$dir/package-lock.json" ] ||
        [ -f "$dir/pnpm-lock.yaml" ] ||
        [ -f "$dir/yarn.lock" ] ||
        [ -f "$dir/tsconfig.json" ] ||
        has_name_in_dir "$dir" 'tsconfig.*.json'
      ;;
    python)
      [ -f "$dir/pyproject.toml" ] ||
        [ -f "$dir/requirements.txt" ] ||
        [ -f "$dir/setup.py" ] ||
        [ -f "$dir/tox.ini" ] ||
        has_name_in_dir "$dir" 'requirements-*.txt'
      ;;
    rust)
      [ -f "$dir/Cargo.toml" ]
      ;;
    golang)
      [ -f "$dir/go.mod" ]
      ;;
    dart)
      [ -f "$dir/pubspec.yaml" ]
      ;;
    terraform)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

nearest_project_root() {
  lang="$1"
  file="$2"
  dir="$(file_dir "$file")"

  if [ "$lang" = "terraform" ]; then
    printf '%s\n' "$dir"
    return 0
  fi

  while :; do
    if has_project_marker "$lang" "$dir"; then
      printf '%s\n' "$dir"
      return 0
    fi

    [ "$dir" = "." ] && return 1
    dir="$(dirname "$dir")"
  done
}

while IFS= read -r file; do
  [ -n "$file" ] || continue

  if is_shared_full_file "$file"; then
    fallback_full "shared:$file"
  fi

  lang="$(classify_language "$file")"
  if [ -n "$lang" ]; then
    emit_lang "$lang"
    root="$(nearest_project_root "$lang" "$file" || true)"
    if [ -n "$root" ]; then
      emit_root "$lang" "$root"
    fi
    continue
  fi

  if is_docs_file "$file"; then
    continue
  fi

  fallback_full "unclassified:$file"
done < "$tmp_files"

if [ -n "$languages" ]; then
  emit_result changed "changed_languages" false "$languages"
else
  emit_result changed "docs_only" true ""
fi
