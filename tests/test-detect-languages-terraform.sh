#!/usr/bin/env sh
# test-detect-languages-terraform.sh —
# integration tests for the terraform branch of scripts/detect-languages.sh.
#
# Covers the four marker scenarios from the plan's test plan (issue #52):
#   1. *.tf present              → emits `terraform`
#   2. *.tofu only (OpenTofu)    → emits `terraform`
#   3. .terraform.lock.hcl only  → emits `terraform`
#   4. .terraform/ noise only    → does NOT emit `terraform` (prune)
#
# Plus a guard that terraform does not leak into a clean empty dir.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DETECT="$PROJECT_ROOT/scripts/detect-languages.sh"

if [ ! -x "$DETECT" ]; then
  echo "FAIL: detect-languages.sh not found or not executable at $DETECT" >&2
  exit 1
fi

_pass=0
_fail=0
_total=0

assert_contains() {
  _desc="$1"
  _needle="$2"
  _haystack="$3"
  _total=$((_total + 1))
  case "$_haystack" in
    *"$_needle"*)
      _pass=$((_pass + 1))
      printf '  PASS: %s\n' "$_desc"
      ;;
    *)
      _fail=$((_fail + 1))
      printf '  FAIL: %s\n' "$_desc"
      printf '    expected to contain: %s\n' "$_needle"
      printf '    actual output:       %s\n' "$_haystack"
      ;;
  esac
}

assert_not_contains() {
  _desc="$1"
  _needle="$2"
  _haystack="$3"
  _total=$((_total + 1))
  case "$_haystack" in
    *"$_needle"*)
      _fail=$((_fail + 1))
      printf '  FAIL: %s\n' "$_desc"
      printf '    expected NOT to contain: %s\n' "$_needle"
      printf '    actual output:           %s\n' "$_haystack"
      ;;
    *)
      _pass=$((_pass + 1))
      printf '  PASS: %s\n' "$_desc"
      ;;
  esac
}

run_detect_in() {
  # $1 = directory to run inside; prints stdout of detect-languages.sh
  (cd "$1" && "$DETECT" 2>/dev/null)
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/detect-tf-test.XXXXXX")"
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT

# ── Case 1: a single .tf file ──────────────────────────────────────────
case1="$workdir/case1-tf"
mkdir -p "$case1"
cat >"$case1/main.tf" <<'TF'
terraform { required_version = ">= 1.6.0" }
TF
out1="$(run_detect_in "$case1")"
assert_contains "1. *.tf present → emits terraform" "terraform" "$out1"

# ── Case 2: .tofu only (OpenTofu) ─────────────────────────────────────
case2="$workdir/case2-tofu"
mkdir -p "$case2"
cat >"$case2/main.tofu" <<'TF'
terraform { required_version = ">= 1.6.0" }
TF
out2="$(run_detect_in "$case2")"
assert_contains "2. *.tofu only → emits terraform (OpenTofu support)" "terraform" "$out2"

# ── Case 3: .terraform.lock.hcl only ───────────────────────────────────
case3="$workdir/case3-lock"
mkdir -p "$case3"
cat >"$case3/.terraform.lock.hcl" <<'LOCK'
provider "registry.terraform.io/hashicorp/null" { version = "3.2.1" }
LOCK
out3="$(run_detect_in "$case3")"
assert_contains "3. .terraform.lock.hcl only → emits terraform" "terraform" "$out3"

# ── Case 4: .terraform/ directory noise only ───────────────────────────
# init's output dir should be pruned: if a user has only an init-leftover
# .terraform/ with no .tf / .tofu / lockfile, we must not claim this is a
# terraform repo.
case4="$workdir/case4-prune"
mkdir -p "$case4/.terraform"
echo '{}' >"$case4/.terraform/terraform.tfstate"
# A .tf file *inside* .terraform should be pruned too.
echo 'terraform {}' >"$case4/.terraform/leftover.tf"
out4="$(run_detect_in "$case4")"
assert_not_contains "4. only .terraform/ noise → does NOT emit terraform (prune)" "terraform" "$out4"

# ── Case 5: clean empty dir → no terraform leak ────────────────────────
case5="$workdir/case5-empty"
mkdir -p "$case5"
out5="$(run_detect_in "$case5")"
assert_not_contains "5. empty dir → does NOT emit terraform" "terraform" "$out5"

# ── Case 6: mixed go + terraform → both emitted, no duplication ────────
case6="$workdir/case6-mixed"
mkdir -p "$case6"
cat >"$case6/main.tf" <<'TF'
terraform {}
TF
cat >"$case6/go.mod" <<'GO'
module example.com/mixed

go 1.22
GO
out6="$(run_detect_in "$case6")"
assert_contains "6a. mixed go+tf → emits terraform" "terraform" "$out6"
assert_contains "6b. mixed go+tf → emits golang"    "golang"    "$out6"

# Guard: terraform should appear exactly once even with multiple markers.
case7="$workdir/case7-multi-marker"
mkdir -p "$case7"
echo 'terraform {}' >"$case7/main.tf"
echo 'terraform {}' >"$case7/aux.tofu"
echo '# lock'      >"$case7/.terraform.lock.hcl"
out7="$(run_detect_in "$case7")"
count="$(printf '%s\n' "$out7" | grep -c '^terraform$' || true)"
_total=$((_total + 1))
if [ "$count" = "1" ]; then
  _pass=$((_pass + 1))
  printf '  PASS: 7. all three markers present → emits terraform exactly once\n'
else
  _fail=$((_fail + 1))
  printf '  FAIL: 7. all three markers present → expected 1 emit, got %s\n' "$count"
  printf '    raw output: %s\n' "$out7"
fi

printf '\n── Summary ──\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
