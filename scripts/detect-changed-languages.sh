#!/usr/bin/env sh
set -eu

# Detect which language packs are relevant to the current git diff.
#
# Output is a small key=value contract for scripts/run-verify.sh:
#   scope=changed|full
#   reason=<machine-readable reason>
#   docs_only=true|false
#   languages=<space-separated language pack names>
#
# A "full" scope is intentionally conservative. Shared harness files,
# unclassified code-like files, or missing diff context fall back to the
# repository-wide language detector.

languages=""

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

emit_result() {
  scope="$1"
  reason="$2"
  docs_only="$3"
  langs="${4:-}"

  printf 'scope=%s\n' "$scope"
  printf 'reason=%s\n' "$reason"
  printf 'docs_only=%s\n' "$docs_only"
  printf 'languages=%s\n' "$langs"
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
  case "$file" in
    *.go|go.mod|go.sum)
      printf 'golang\n'
      ;;
    *.ts|*.tsx|package.json|package-lock.json|pnpm-lock.yaml|yarn.lock|tsconfig.json|tsconfig.*.json|vite.config.*|vitest.config.*|jest.config.*|next.config.*|eslint.config.*|.eslintrc|.eslintrc.*)
      printf 'typescript\n'
      ;;
    *.py|pyproject.toml|requirements.txt|requirements-*.txt|setup.py|setup.cfg|tox.ini|poetry.lock)
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
    pom.xml|build.gradle|build.gradle.kts)
      printf 'jvm\n'
      ;;
  esac
}

while IFS= read -r file; do
  [ -n "$file" ] || continue

  if is_shared_full_file "$file"; then
    fallback_full "shared:$file"
  fi

  lang="$(classify_language "$file")"
  if [ -n "$lang" ]; then
    emit_lang "$lang"
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
