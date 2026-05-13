#!/usr/bin/env sh
# test-terraform-gitignore.sh —
# pins the Terraform / OpenTofu ignore patterns in the project root
# .gitignore and the templates/base/.gitignore mirror.
#
# Background: cycle-3 of the issue #52 pipeline added explicit ignore
# entries for Terraform state, plan output, auto-loaded tfvars, override
# files, and crash logs to both .gitignore files (commit 03c5598).
# The cycle-3 verifier flagged that the walkthrough used a one-off
# `git check-ignore -v` invocation in a scratch repo and recommended
# promoting it into a permanent shell test. Without this test, a future
# refactor of either .gitignore could silently un-ignore Terraform state
# (the very files that often carry provider secrets) and no other gate
# would catch it: shellcheck doesn't read .gitignore; check-sync.sh
# enforces byte equality between the two files but not their semantics;
# the language-pack verifier never touches git.
#
# What this test does:
#   1. For each new Terraform-related pattern, drop a sentinel file into
#      a hermetic temp git repo and assert `git check-ignore -v` reports
#      the expected line in .gitignore as the matching rule.
#   2. Negative control: a plain `main.tf` Terraform source file must NOT
#      be ignored (we want to track HCL sources, not skip them).
#   3. Cross-check both .gitignore files (root + templates/base mirror)
#      produce identical behavior for the same sentinel files. The byte
#      equality of the two files is enforced by check-sync.sh; this test
#      verifies the equality is also semantically meaningful for ignore
#      matching, which guards against e.g. encoding glitches that the
#      sync gate might miss.
#
# Hermeticity: we set HOME and GIT_CONFIG_GLOBAL to an isolated dir so
# the user's `~/.gitignore_global` cannot influence results, and we use
# `git init` with a brand-new repo so no parent .gitignore leaks in.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_GITIGNORE="$PROJECT_ROOT/.gitignore"
MIRROR_GITIGNORE="$PROJECT_ROOT/templates/base/.gitignore"

if [ ! -f "$ROOT_GITIGNORE" ]; then
  echo "FAIL: missing .gitignore at $ROOT_GITIGNORE" >&2
  exit 1
fi
if [ ! -f "$MIRROR_GITIGNORE" ]; then
  echo "FAIL: missing mirror .gitignore at $MIRROR_GITIGNORE" >&2
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

assert_eq() {
  _desc="$1"; _expected="$2"; _actual="$3"
  if [ "$_expected" = "$_actual" ]; then
    record_pass "$_desc"
  else
    record_fail "$_desc"
    printf '    expected: %s\n' "$_expected"
    printf '    actual:   %s\n' "$_actual"
  fi
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/tf-gitignore-test.XXXXXX")"
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT

# Isolate git from the host's global config / global gitignore.
hermetic_home="$workdir/.home"
mkdir -p "$hermetic_home"
HOME="$hermetic_home"
GIT_CONFIG_GLOBAL="$hermetic_home/.gitconfig"
GIT_CONFIG_SYSTEM=/dev/null
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_TERMINAL_PROMPT

# Sentinel table:
#   <sentinel-path>|<expected pattern in .gitignore>
#
# The pattern column is what `git check-ignore -v` reports in its 3rd
# tab-separated field. It must match the .gitignore line text exactly.
# We pick sentinel files that exercise every glob: directory match,
# bare extension, double-extension backup variant, dot-prefixed files,
# and dot-injection crash logs.
sentinels='
.terraform/providers/.keep|.terraform/
terraform.tfstate|*.tfstate
terraform.tfstate.backup|*.tfstate.backup
terraform.tfstate.20260513.backup|*.tfstate.*.backup
terraform.tfplan|*.tfplan
production.auto.tfvars|*.auto.tfvars
staging.auto.tfvars.json|*.auto.tfvars.json
override.tf|override.tf
override.tf.json|override.tf.json
custom_override.tf|*_override.tf
custom_override.tf.json|*_override.tf.json
crash.log|crash.log
crash.20260513.log|crash.*.log
'

# Files that MUST NOT be ignored (we still want to track HCL sources).
# Format: <sentinel-path>
negative_sentinels='
main.tf
variables.tf
outputs.tf
modules/vpc/main.tf
'

run_section() {
  _label="$1"
  _gitignore_src="$2"

  printf '\n── %s ──\n' "$_label"

  _repo="$workdir/$_label.repo"
  mkdir -p "$_repo"
  # Use a separate isolated repo per gitignore source.
  ( cd "$_repo" && git init -q --initial-branch=main )
  cp "$_gitignore_src" "$_repo/.gitignore"

  # Materialize every sentinel so check-ignore has something to resolve.
  # We need real paths because some patterns (.terraform/) are directory
  # globs and git check-ignore evaluates against the on-disk file path.
  while IFS='|' read -r _sentinel _expected_pattern; do
    [ -z "$_sentinel" ] && continue
    case "$_sentinel" in
      */*) mkdir -p "$_repo/$(dirname "$_sentinel")" ;;
    esac
    : > "$_repo/$_sentinel"
  done <<EOF
$sentinels
EOF

  # Negative-control source files too.
  for _src in $negative_sentinels; do
    [ -z "$_src" ] && continue
    case "$_src" in
      */*) mkdir -p "$_repo/$(dirname "$_src")" ;;
    esac
    : > "$_repo/$_src"
  done

  # Positive assertions: each sentinel must be ignored by the expected
  # pattern. `git check-ignore -v` exits 0 when a pattern matches, 1
  # when no pattern matches; output is `<source>:<line>:<pattern>\t<path>`.
  while IFS='|' read -r _sentinel _expected_pattern; do
    [ -z "$_sentinel" ] && continue
    _output="$( ( cd "$_repo" && git check-ignore -v -- "$_sentinel" ) 2>/dev/null || true)"
    if [ -z "$_output" ]; then
      record_fail "$_label: $_sentinel should be ignored by '$_expected_pattern' (check-ignore reported no match)"
      continue
    fi
    # Extract the pattern column (3rd tab-separated value before the path).
    # `<source>:<line>:<pattern>\t<path>` — pattern is everything after
    # the 2nd ':' and before the tab.
    _actual_pattern="$(printf '%s' "$_output" | sed -E 's/^[^:]+:[0-9]+:([^	]+)	.*/\1/')"
    assert_eq "$_label: $_sentinel ignored by expected pattern" "$_expected_pattern" "$_actual_pattern"
  done <<EOF
$sentinels
EOF

  # Negative assertions: HCL sources must NOT be ignored.
  for _src in $negative_sentinels; do
    [ -z "$_src" ] && continue
    if ( cd "$_repo" && git check-ignore -q -- "$_src" ); then
      _why="$( ( cd "$_repo" && git check-ignore -v -- "$_src" ) 2>/dev/null || true)"
      record_fail "$_label: $_src must NOT be ignored (matched by: $_why)"
    else
      record_pass "$_label: $_src is tracked (not ignored)"
    fi
  done
}

run_section "root-gitignore" "$ROOT_GITIGNORE"
run_section "templates-base-mirror" "$MIRROR_GITIGNORE"

# Cross-check: both .gitignore files must produce identical ignore
# behavior. Byte equality is already enforced by scripts/check-sync.sh,
# but we additionally assert behavioral equality: for each sentinel,
# the (source-line-pattern) tuple resolved by each repo must be equal
# (the source filename may differ — both repos use ".gitignore" — but
# the line number and pattern must match).
printf '\n── cross-check: root vs templates/base produce identical ignore behavior ──\n'

while IFS='|' read -r _sentinel _expected_pattern; do
  [ -z "$_sentinel" ] && continue
  _root_out="$( ( cd "$workdir/root-gitignore.repo" && git check-ignore -v -- "$_sentinel" ) 2>/dev/null || true)"
  _mirror_out="$( ( cd "$workdir/templates-base-mirror.repo" && git check-ignore -v -- "$_sentinel" ) 2>/dev/null || true)"
  # Normalize: strip the path suffix and keep only `source:line:pattern`.
  _root_norm="$(printf '%s' "$_root_out" | sed -E 's/	.*$//')"
  _mirror_norm="$(printf '%s' "$_mirror_out" | sed -E 's/	.*$//')"
  assert_eq "cross-check: $_sentinel resolves to same line+pattern in both gitignores" "$_root_norm" "$_mirror_norm"
done <<EOF
$sentinels
EOF

printf '\n── Summary ──\n'
printf '  PASS: %d\n' "$_pass"
printf '  FAIL: %d\n' "$_fail"
printf '  TOTAL: %d\n' "$_total"

if [ "$_fail" -ne 0 ]; then
  exit 1
fi
exit 0
