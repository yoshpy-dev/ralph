#!/usr/bin/env sh
set -eu

# Detect which language packs are relevant to the current git diff.
#
# Output is a small key=value contract for scripts/run-verify.sh:
#   scope=changed|full
#   reason=<machine-readable reason>
#   docs_only=true|false|unknown
#   languages=<space-separated language pack names>
#   first_non_docs=<path>, only with docs_only=false
#   <language>_roots=<space-separated project roots, when changed scope can narrow safely>
#
# The change set is every path that differs between HEAD and its merge-base
# with the diff base (below), plus unstaged, staged, and untracked files
# (not ignored ones). docs_only describes the whole change set, in either
# scope:
#   true     every path is a docs file (the table of is_docs_file) or a
#            ralph scaffold file
#   false    at least one path is neither; first_non_docs names the first
#            such path in sorted order
#   unknown  the change set was not read: no git, no repository, a settings
#            error, no ref for GITHUB_BASE_REF, no remote default, no base,
#            no merge-base, or a failed listing (diff_failed)
#
# A ralph scaffold file is .ralph/manifest.toml, a path under
# .ralph/baseline/, or a path that the manifest lists as a [files.<key>]
# table, its key single-quoted, double-quoted with no backslash, or bare.
# These are the files ralph init and ralph upgrade write, so a change that
# only adds or updates them is docs only. A path that still exists is looked
# up in the working tree's manifest. A path that no longer exists is looked
# up there and in the manifest at the merge-base, because ralph upgrade
# removes an obsolete core file together with its entry. Adding a path to
# the manifest makes a change to it pass as scaffold: the rule keeps
# scaffold-only changes from failing, and it is not a security boundary.
#
# The four git commands that list the change set are each checked. When one
# of them exits non-zero, even after another printed paths, the result is
# full with reason diff_failed and docs_only=unknown, so a failed read never
# passes as docs only or as no_changes. no_changes means that all four ran
# and printed nothing.
#
# The four listings run as git -c core.quotePath=false, so a path with
# non-ASCII characters is printed as it is and can match the docs table and
# the manifest; git's default prints it in double quotes with octal escapes
# ("docs/\346\227\245..."), which matches neither. git still quotes a path
# that holds a control character, a double quote, or a backslash; that
# quoted form matches neither, so such a path counts as non-docs.
#
# A "full" scope is intentionally conservative. Shared harness files,
# unclassified code-like files, or missing diff context fall back to the
# repository-wide language detector. The first shared or unclassified path
# gives the reason, and languages and roots stop growing there; the rest of
# the change set is still read for docs_only.
#
# The diff base comes first from GITHUB_BASE_REF when it is not empty:
# refs/remotes/origin/<name>, else refs/heads/<name>. With neither ref the
# result is full with reason no_base_ref:<label>, and with a ref that has no
# merge-base with HEAD, no_merge_base:refs/remotes/origin/<label> or
# no_merge_base:refs/heads/<label>. <label> is the value, or the word
# GITHUB_BASE_REF when the value holds a character other than
# A-Z a-z 0-9 . _ / -, so such a value never reaches the output. A local
# shell that still exports GITHUB_BASE_REF changes the base as well.
#
# Otherwise the diff base is [verify].base of ralph.local.toml when it is not empty:
# read with ralph_setting_or_fail of scripts/ralph-config.sh, which runs
# `ralph config get verify.base`, both in the root of the repository of the
# current directory (git rev-parse --show-toplevel), as detect_base_branch
# of scripts/xreview-helpers.sh does. In a repository with no
# scripts/ralph-config.sh (one scaffolded before it existed) there is no
# setting to read. With no ralph on PATH, or one older than `config get`,
# the value is the empty default. When the read fails (ralph config get
# exits 3 for an invalid ralph.local.toml or ralph.toml, for example), a
# line on stderr follows the one ralph_setting_or_fail writes, and the
# result is full with reason config_error:verify.base, never the automatic
# base below: on the default branch that base can be HEAD itself, and a
# mistake in the settings would then end in no_changes and verify nothing.
# No setting comes from an environment variable. GITHUB_BASE_REF, the pull
# request's base that GitHub Actions sets, comes first (above), read as
# scripts/secret-scan-branch.sh reads it, so ralph is not asked then.
#
# With GITHUB_BASE_REF and [verify].base empty, the base is the first
# existing ref among: origin's default branch (the target of
# refs/remotes/origin/HEAD when that is an existing ref under
# refs/remotes/origin/, then origin/main, origin/master); the same three for
# the remote the current branch tracks (skipped when HEAD is detached or that
# remote is origin or "."); then local main, master. Defaults are handed to
# git merge-base as full ref names, so a tag or branch sharing a short name
# cannot take their place. The branch's own @{upstream} is never consulted:
# after a push it equals HEAD and would hide every change committed since the
# branch left the default branch. When branch.<current>.remote is set (a
# remote name, "." for a local branch, or a URL), neither origin nor that
# remote has a default branch ref locally, and the local fallback is the
# current branch itself, committed changes cannot be seen, so the result is
# full with reason no_remote_default:<remote> (no_remote_default:. for a local
# branch, no_remote_default:url when the value is not a configured remote name,
# so a URL, which may carry credentials, never reaches the output). Where the
# base resolves to HEAD itself (no remote default ref and no
# branch.<current>.remote, on local main or master or detached at its tip),
# committed changes are not visible, only uncommitted and untracked files
# count, and the result can be no_changes; set [verify].base in
# ralph.local.toml (or RALPH_VERIFY_SCOPE=full in the wrappers) to cover
# them.

languages=""
typescript_roots=""
python_roots=""
rust_roots=""
golang_roots=""
dart_roots=""
terraform_roots=""
first_non_docs=""

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
  [ -n "$first_non_docs" ] && printf 'first_non_docs=%s\n' "$first_non_docs"
  [ -n "$typescript_roots" ] && printf 'typescript_roots=%s\n' "$typescript_roots"
  [ -n "$python_roots" ] && printf 'python_roots=%s\n' "$python_roots"
  [ -n "$rust_roots" ] && printf 'rust_roots=%s\n' "$rust_roots"
  [ -n "$golang_roots" ] && printf 'golang_roots=%s\n' "$golang_roots"
  [ -n "$dart_roots" ] && printf 'dart_roots=%s\n' "$dart_roots"
  [ -n "$terraform_roots" ] && printf 'terraform_roots=%s\n' "$terraform_roots"
  return 0
}

# fallback_unread <reason> -- the change set was not read: the full scope
# with docs_only=unknown, then exit.
fallback_unread() {
  emit_result full "$1" unknown ""
  exit 0
}

if ! command -v git >/dev/null 2>&1; then
  fallback_unread "git_not_found"
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  fallback_unread "no_git_repository"
fi

# The root of the repository of the current directory: [verify].base is read
# there, and the working tree's .ralph/manifest.toml is found there.
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"

# Print the default branch of remote $1 as a full ref name: the branch that
# refs/remotes/<remote>/HEAD points to (only when it is an existing ref under
# refs/remotes/<remote>/), else refs/remotes/<remote>/main, else
# refs/remotes/<remote>/master. Print nothing when none exists.
remote_default_ref() {
  remote="$1"
  head_target="$(git symbolic-ref --quiet "refs/remotes/$remote/HEAD" 2>/dev/null || true)"
  case "$head_target" in
    "refs/remotes/$remote/"?*)
      if git show-ref --verify --quiet "$head_target"; then
        printf '%s\n' "$head_target"
        return 0
      fi
      ;;
  esac
  for candidate in main master; do
    if git show-ref --verify --quiet "refs/remotes/$remote/$candidate"; then
      printf 'refs/remotes/%s/%s\n' "$remote" "$candidate"
      return 0
    fi
  done
  return 0
}

base_ref=""
# base_ref_label is what the no_merge_base reason shows of a base taken from
# GITHUB_BASE_REF; empty for any other base, whose reason shows base_ref.
base_ref_label=""
if [ -n "${GITHUB_BASE_REF:-}" ]; then
  # GITHUB_BASE_REF (see the header), as scripts/secret-scan-branch.sh reads
  # it; ralph is not asked. base_label is the only form of the value a
  # reason may show.
  case "$GITHUB_BASE_REF" in
    *[!ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._/-]*)
      base_label="GITHUB_BASE_REF"
      ;;
    *)
      base_label="$GITHUB_BASE_REF"
      ;;
  esac
  if git show-ref --verify --quiet "refs/remotes/origin/$GITHUB_BASE_REF"; then
    base_ref="refs/remotes/origin/$GITHUB_BASE_REF"
    base_ref_label="refs/remotes/origin/$base_label"
  elif git show-ref --verify --quiet "refs/heads/$GITHUB_BASE_REF"; then
    base_ref="refs/heads/$GITHUB_BASE_REF"
    base_ref_label="refs/heads/$base_label"
  else
    fallback_unread "no_base_ref:$base_label"
  fi
elif [ -n "$repo_root" ] && [ -f "$repo_root/scripts/ralph-config.sh" ]; then
  # [verify].base (see the header). ralph-config.sh is sourced in a
  # subshell, so none of its functions or variables stay in this shell.
  # shellcheck source=ralph-config.sh
  if ! base_ref="$(CDPATH='' cd -- "$repo_root" && . ./scripts/ralph-config.sh &&
    ralph_setting_or_fail verify.base)"; then
    printf 'detect-changed-languages.sh: [verify].base could not be read (the line above says why; a mistake in any key of ralph.toml or ralph.local.toml makes every key fail); taking the full scope, not the automatic base\n' >&2
    fallback_unread "config_error:verify.base"
  fi
fi
if [ -z "$base_ref" ]; then
  current_ref="$(git symbolic-ref --quiet HEAD 2>/dev/null || true)"
  configured_remote=""
  case "$current_ref" in
    refs/heads/?*)
      configured_remote="$(git config --get "branch.${current_ref#refs/heads/}.remote" 2>/dev/null || true)"
      ;;
  esac
  # configured_remote is the raw branch.<b>.remote (it decides the guard
  # below); tracked_remote is the remote name step 2 consults (empty for "."
  # or a URL); remote_label is the only form of the value the reason may show.
  tracked_remote=""
  remote_label=""
  case "$configured_remote" in
    "") ;;
    .) remote_label="." ;;
    *)
      if git config --get "remote.$configured_remote.url" >/dev/null 2>&1; then
        tracked_remote="$configured_remote"
        remote_label="$configured_remote"
      else
        remote_label="url"
      fi
      ;;
  esac

  base_ref="$(remote_default_ref origin)"
  if [ -z "$base_ref" ] && [ -n "$tracked_remote" ] && [ "$tracked_remote" != "origin" ]; then
    base_ref="$(remote_default_ref "$tracked_remote")"
  fi
  if [ -z "$base_ref" ]; then
    local_default=""
    if git show-ref --verify --quiet refs/heads/main; then
      local_default="refs/heads/main"
    elif git show-ref --verify --quiet refs/heads/master; then
      local_default="refs/heads/master"
    fi
    if [ -n "$configured_remote" ] && [ "$local_default" = "$current_ref" ]; then
      fallback_unread "no_remote_default:$remote_label"
    fi
    base_ref="$local_default"
  fi
fi

if [ -z "$base_ref" ]; then
  fallback_unread "no_diff_base"
fi

merge_base="$(git merge-base HEAD "$base_ref" 2>/dev/null || true)"
if [ -z "$merge_base" ]; then
  fallback_unread "no_merge_base:${base_ref_label:-$base_ref}"
fi

# Every temporary file of this run lives in tmp_dir. The EXIT trap removes
# it; an EXIT trap does not change the exit status. HUP, INT, and TERM exit
# with 128 + the signal number, which runs the EXIT trap, as
# scripts/run-verify.sh does, so the run ends there instead of going on
# without its files.
tmp_dir=""
cleanup_tmp_dir() {
  if [ -n "$tmp_dir" ]; then
    rm -rf "$tmp_dir" || true
  fi
}
trap cleanup_tmp_dir EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-changed-files.XXXXXX")"

# The change set (see the header). Each listing command is checked on its
# own, so one that fails after another printed paths leaves no partial list.
# core.quotePath=false prints a non-ASCII path as it is (see the header).
if ! git -c core.quotePath=false diff --name-only "$merge_base" HEAD > "$tmp_dir/committed" 2>/dev/null ||
  ! git -c core.quotePath=false diff --name-only > "$tmp_dir/unstaged" 2>/dev/null ||
  ! git -c core.quotePath=false diff --name-only --cached > "$tmp_dir/staged" 2>/dev/null ||
  ! git -c core.quotePath=false ls-files --others --exclude-standard > "$tmp_dir/untracked" 2>/dev/null; then
  fallback_unread "diff_failed"
fi
sort -u "$tmp_dir/committed" "$tmp_dir/unstaged" "$tmp_dir/staged" "$tmp_dir/untracked" > "$tmp_dir/changed"

if [ ! -s "$tmp_dir/changed" ]; then
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

# manifest_paths <manifest> -- prints, one per line, the path of each
# [files.<key>] table of <manifest>: a line that is exactly [files.'<path>'],
# [files."<path>"] with no backslash in <path> (a backslash starts an escape
# there, so the text is not the path), or [files.<bare key>].
manifest_paths() {
  LC_ALL=C sed -n \
    -e "s/^\[files\.'\([^'][^']*\)'\]\$/\1/p" \
    -e 's/^\[files\."\([^"\\][^"\\]*\)"\]$/\1/p' \
    -e 's/^\[files\.\([A-Za-z0-9_-][A-Za-z0-9_-]*\)\]$/\1/p' \
    "$1"
}

# The scaffold lists (see the header): the paths of the working tree's
# manifest, and those of the manifest at the merge-base. A manifest that is
# missing or cannot be read gives an empty list.
: > "$tmp_dir/scaffold-here"
: > "$tmp_dir/scaffold-base"
if [ -n "$repo_root" ] && [ -f "$repo_root/.ralph/manifest.toml" ]; then
  manifest_paths "$repo_root/.ralph/manifest.toml" > "$tmp_dir/scaffold-here" 2>/dev/null || true
fi
if git show "$merge_base:.ralph/manifest.toml" > "$tmp_dir/manifest-base.toml" 2>/dev/null; then
  manifest_paths "$tmp_dir/manifest-base.toml" > "$tmp_dir/scaffold-base" 2>/dev/null || true
fi

# is_scaffold_file <path> -- <path> is a ralph scaffold file (see the
# header). A path that still exists counts only when the working tree's
# manifest lists it.
is_scaffold_file() {
  file="$1"
  case "$file" in
    .ralph/manifest.toml|.ralph/baseline/*)
      return 0
      ;;
  esac
  if grep -Fqx -- "$file" "$tmp_dir/scaffold-here"; then
    return 0
  fi
  if [ -z "$repo_root" ] || [ -e "$repo_root/$file" ] || [ -L "$repo_root/$file" ]; then
    return 1
  fi
  grep -Fqx -- "$file" "$tmp_dir/scaffold-base"
}

# full_reason is the reason of the first shared or unclassified path, where
# the scope, reason, languages, and roots are settled; the loop goes on only
# while all_docs (docs_only) can still change.
full_reason=""
all_docs=true
while IFS= read -r file; do
  [ -n "$file" ] || continue

  if [ "$all_docs" = true ] && ! is_docs_file "$file" && ! is_scaffold_file "$file"; then
    all_docs=false
    first_non_docs="$file"
  fi

  if [ -n "$full_reason" ]; then
    [ "$all_docs" = true ] || break
    continue
  fi

  if is_shared_full_file "$file"; then
    full_reason="shared:$file"
    continue
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

  full_reason="unclassified:$file"
done < "$tmp_dir/changed"

if [ -n "$full_reason" ]; then
  emit_result full "$full_reason" "$all_docs" ""
elif [ -n "$languages" ]; then
  emit_result changed "changed_languages" "$all_docs" "$languages"
else
  emit_result changed "docs_only" "$all_docs" ""
fi
