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

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
