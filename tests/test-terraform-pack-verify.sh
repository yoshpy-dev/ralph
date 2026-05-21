#!/usr/bin/env sh
# test-terraform-pack-verify.sh —
# integration tests for packs/languages/terraform/verify.sh.
#
# Tests every branch the plan calls out (issue #52):
#   - No markers              → warning + exit 0  (pack not applicable)
#   - Markers + no CLI on PATH → error   + exit 1  (fail-open regression guard)
#   - Markers + fake CLI + no .terraform/ dir
#                              → validate is skipped, fmt runs against fake CLI
#   - HARNESS_VERIFY_MODE=static / test / all dispatch
#   - tofu preferred over terraform when both exist on PATH
#   - tflint / tfsec / trivy missing → skipped with explicit message
#
# We never invoke a real terraform/tofu binary. Instead, we drop a
# stub script into a sandboxed PATH and assert which subcommands it
# is asked to run.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERIFY="$PROJECT_ROOT/packs/languages/terraform/verify.sh"

if [ ! -x "$VERIFY" ]; then
  echo "FAIL: verify.sh not found or not executable at $VERIFY" >&2
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

assert_exit() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  if [ "$_expected" = "$_actual" ]; then
    record_pass "$_desc (exit $_actual)"
  else
    record_fail "$_desc (expected exit $_expected, got $_actual)"
  fi
}

assert_stdout_contains() {
  _desc="$1"
  _needle="$2"
  _logfile="$3"
  if grep -q -- "$_needle" "$_logfile" 2>/dev/null; then
    record_pass "$_desc"
  else
    record_fail "$_desc (looked for '$_needle')"
    printf '    log tail:\n'
    sed -n '$p' "$_logfile" 2>/dev/null | sed 's/^/      /'
  fi
}

assert_stdout_not_contains() {
  _desc="$1"
  _needle="$2"
  _logfile="$3"
  if grep -q -- "$_needle" "$_logfile" 2>/dev/null; then
    record_fail "$_desc (unexpected '$_needle' in output)"
  else
    record_pass "$_desc"
  fi
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/tf-pack-test.XXXXXX")"
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT

# Build a hermetic PATH that contains ONLY the POSIX coreutils we need,
# never the IaC tools. We must not point at /usr/bin or /bin directly,
# because CI hosts often have terraform/tofu/tflint/tfsec/trivy installed
# system-wide (apt-get install terraform, etc.) — exposing them would
# make the "no CLI" / "optional tool absent" branches non-hermetic.
#
# Strategy: create a fresh directory and populate it with symlinks to
# the host's resolved paths for each coreutil we depend on. If a
# required coreutil is missing on the host, fail loudly rather than
# silently degrade.
coreutils_dir="$workdir/.coreutils"
mkdir -p "$coreutils_dir"
for _tool in sh find grep sed cat chmod mkdir rm printf ls test true false head tr dirname sort mktemp; do
  _resolved="$(command -v "$_tool" 2>/dev/null || true)"
  if [ -z "$_resolved" ]; then
    echo "FAIL: required coreutil '$_tool' not found on host PATH" >&2
    exit 1
  fi
  ln -s "$_resolved" "$coreutils_dir/$_tool"
done
# Guard: refuse to start if any IaC tool somehow leaked into the
# coreutils dir. This prevents a future maintainer from accidentally
# adding 'terraform' or 'tofu' to the symlink list above.
for _banned in terraform tofu tflint tfsec trivy; do
  if [ -e "$coreutils_dir/$_banned" ]; then
    echo "FAIL: hermetic PATH leak — '$_banned' present in $coreutils_dir" >&2
    exit 1
  fi
done
clean_path="$coreutils_dir"

make_stub_dir() {
  # $1 = directory to populate; $2..$N = names of stubs to create.
  _dir="$1"; shift
  mkdir -p "$_dir"
  for name in "$@"; do
    stub="$_dir/$name"
    cat >"$stub" <<EOF
#!/usr/bin/env sh
# stub for $name — records subcommand under $_dir/log.\$\$ and exits 0.
{
  printf '%s' "$name"
  for arg in "\$@"; do printf ' %s' "\$arg"; done
  printf '\n'
} >>"$_dir/calls.log"
exit 0
EOF
    chmod +x "$stub"
  done
}

# Run verify.sh inside $1 (working dir) with mode $2 and PATH $3.
# Writes combined output to $4 and exit code is returned via $?.
run_verify_in() {
  _wd="$1"; _mode="$2"; _path="$3"; _log="$4"
  (
    cd "$_wd"
    HARNESS_VERIFY_MODE="$_mode" \
      PATH="$_path" \
      "$VERIFY"
  ) >"$_log" 2>&1
}

run_verify_init_validate_in() {
  _wd="$1"; _mode="$2"; _path="$3"; _log="$4"
  (
    cd "$_wd"
    HARNESS_VERIFY_MODE="$_mode" \
      RALPH_TERRAFORM_INIT_VALIDATE=true \
      PATH="$_path" \
      "$VERIFY"
  ) >"$_log" 2>&1
}

# ── Case A: no markers → warning + exit 0 ──────────────────────────────
caseA="$workdir/caseA-no-markers"
mkdir -p "$caseA"
logA="$caseA/out.log"
set +e
run_verify_in "$caseA" "all" "$clean_path" "$logA"
rcA=$?
set -e
assert_exit "A. no markers → exit 0 (pack not applicable)" 0 "$rcA"
assert_stdout_contains "A. no markers → skip message printed" \
  "Skipping Terraform verifier" "$logA"

# ── Case B: markers present + no CLI on PATH → exit 1 (fail-open guard) ─
caseB="$workdir/caseB-no-cli"
mkdir -p "$caseB"
echo 'terraform {}' >"$caseB/main.tf"
logB="$caseB/out.log"
set +e
run_verify_in "$caseB" "all" "$clean_path" "$logB"
rcB=$?
set -e
assert_exit "B. markers + no CLI → exit 1 (fail-open avoidance)" 1 "$rcB"
assert_stdout_contains "B. error message names both CLIs" \
  "neither 'tofu' nor 'terraform'" "$logB"

# ── Case C: markers + fake terraform + no .terraform/ → validate skipped ─
caseC="$workdir/caseC-no-init"
mkdir -p "$caseC"
echo 'terraform {}' >"$caseC/main.tf"
stubsC="$workdir/stubsC"
make_stub_dir "$stubsC" "terraform"
logC="$caseC/out.log"
set +e
run_verify_in "$caseC" "static" "$stubsC:$clean_path" "$logC"
rcC=$?
set -e
assert_exit "C. markers + stub terraform + no .terraform/ → exit 0" 0 "$rcC"
assert_stdout_contains "C. announces using terraform CLI" \
  "Using IaC CLI: terraform" "$logC"
assert_stdout_contains "C. fmt was invoked on terraform stub" \
  "terraform fmt -check" "$stubsC/calls.log"
assert_stdout_not_contains "C. validate NOT invoked when .terraform/ absent" \
  "terraform validate" "$stubsC/calls.log"
assert_stdout_contains "C. explicit skip message for validate" \
  "Skipping terraform validate" "$logC"
assert_stdout_contains "C. tflint missing → skip message" \
  "Skipping tflint" "$logC"
assert_stdout_contains "C. tfsec/trivy missing → skip message" \
  "Skipping tfsec / trivy config" "$logC"

# ── Case C2: opt-in backend-less init validate runs without .terraform/ ─
caseC2="$workdir/caseC2-init-validate"
mkdir -p "$caseC2"
echo 'terraform {}' >"$caseC2/main.tf"
stubsC2="$workdir/stubsC2"
make_stub_dir "$stubsC2" "terraform"
logC2="$caseC2/out.log"
set +e
run_verify_init_validate_in "$caseC2" "static" "$stubsC2:$clean_path" "$logC2"
rcC2=$?
set -e
assert_exit "C2. opt-in init validate without .terraform/ → exit 0" 0 "$rcC2"
assert_stdout_contains "C2. backend-less init invoked" \
  "terraform init -backend=false -input=false" "$stubsC2/calls.log"
assert_stdout_contains "C2. validate invoked after opt-in init" \
  "terraform validate" "$stubsC2/calls.log"
assert_stdout_not_contains "C2. no default validate skip message" \
  "Skipping terraform validate" "$logC2"

# ── Case D: tofu preferred over terraform when both exist ──────────────
caseD="$workdir/caseD-tofu-first"
mkdir -p "$caseD"
echo 'terraform {}' >"$caseD/main.tofu"
stubsD="$workdir/stubsD"
make_stub_dir "$stubsD" "terraform" "tofu"
logD="$caseD/out.log"
set +e
run_verify_in "$caseD" "static" "$stubsD:$clean_path" "$logD"
rcD=$?
set -e
assert_exit "D. tofu+terraform on PATH → exit 0" 0 "$rcD"
assert_stdout_contains "D. tofu wins over terraform (OpenTofu preference)" \
  "Using IaC CLI: tofu" "$logD"
# terraform stub should not have been called at all
assert_stdout_not_contains "D. terraform stub was NOT invoked" \
  "terraform " "$stubsD/calls.log"
assert_stdout_contains "D. tofu fmt invoked" \
  "tofu fmt -check" "$stubsD/calls.log"

# ── Case E: HARNESS_VERIFY_MODE=test with no *.tftest.hcl → skip ───────
caseE="$workdir/caseE-test-mode-no-tests"
mkdir -p "$caseE"
echo 'terraform {}' >"$caseE/main.tf"
stubsE="$workdir/stubsE"
make_stub_dir "$stubsE" "terraform"
logE="$caseE/out.log"
set +e
run_verify_in "$caseE" "test" "$stubsE:$clean_path" "$logE"
rcE=$?
set -e
assert_exit "E. mode=test + no *.tftest.hcl → exit 0" 0 "$rcE"
assert_stdout_contains "E. announces no terraform tests" \
  "no terraform tests" "$logE"
assert_stdout_not_contains "E. mode=test must NOT invoke fmt" \
  "fmt -check" "$stubsE/calls.log"
assert_stdout_not_contains "E. mode=test must NOT invoke test (no fixtures)" \
  "terraform test" "$stubsE/calls.log"

# ── Case F: HARNESS_VERIFY_MODE=test with *.tftest.hcl → test runs ─────
caseF="$workdir/caseF-test-mode-with-tests"
mkdir -p "$caseF"
echo 'terraform {}' >"$caseF/main.tf"
cat >"$caseF/main.tftest.hcl" <<'T'
run "smoke" { command = plan }
T
stubsF="$workdir/stubsF"
make_stub_dir "$stubsF" "terraform"
logF="$caseF/out.log"
set +e
run_verify_in "$caseF" "test" "$stubsF:$clean_path" "$logF"
rcF=$?
set -e
assert_exit "F. mode=test + *.tftest.hcl → exit 0 (stub succeeds)" 0 "$rcF"
assert_stdout_contains "F. terraform test was invoked" \
  "terraform test" "$stubsF/calls.log"
assert_stdout_not_contains "F. mode=test must NOT invoke fmt" \
  "fmt -check" "$stubsF/calls.log"

# ── Case G: HARNESS_VERIFY_MODE=all runs both static and test ──────────
caseG="$workdir/caseG-all-mode"
mkdir -p "$caseG"
echo 'terraform {}' >"$caseG/main.tf"
cat >"$caseG/aux.tftest.hcl" <<'T'
run "smoke" { command = plan }
T
stubsG="$workdir/stubsG"
make_stub_dir "$stubsG" "terraform"
logG="$caseG/out.log"
set +e
run_verify_in "$caseG" "all" "$stubsG:$clean_path" "$logG"
rcG=$?
set -e
assert_exit "G. mode=all → exit 0" 0 "$rcG"
assert_stdout_contains "G. mode=all invokes fmt" \
  "terraform fmt -check" "$stubsG/calls.log"
assert_stdout_contains "G. mode=all invokes test" \
  "terraform test" "$stubsG/calls.log"

# ── Case H: unknown HARNESS_VERIFY_MODE → exit 2 ───────────────────────
caseH="$workdir/caseH-bogus-mode"
mkdir -p "$caseH"
echo 'terraform {}' >"$caseH/main.tf"
stubsH="$workdir/stubsH"
make_stub_dir "$stubsH" "terraform"
logH="$caseH/out.log"
set +e
run_verify_in "$caseH" "bogus" "$stubsH:$clean_path" "$logH"
rcH=$?
set -e
assert_exit "H. unknown mode → exit 2" 2 "$rcH"
assert_stdout_contains "H. error names the bad mode value" \
  "Unknown HARNESS_VERIFY_MODE: bogus" "$logH"

# ── Case I: .terraform/ present → validate IS invoked ──────────────────
# Sanity check for the inverse of Case C.
caseI="$workdir/caseI-init-present"
mkdir -p "$caseI/.terraform"
echo '{}' >"$caseI/.terraform/terraform.tfstate"
echo 'terraform {}' >"$caseI/main.tf"
stubsI="$workdir/stubsI"
make_stub_dir "$stubsI" "terraform"
logI="$caseI/out.log"
set +e
run_verify_in "$caseI" "static" "$stubsI:$clean_path" "$logI"
rcI=$?
set -e
assert_exit "I. markers + .terraform/ present → exit 0" 0 "$rcI"
assert_stdout_contains "I. validate IS invoked when .terraform/ exists" \
  "terraform validate" "$stubsI/calls.log"
assert_stdout_not_contains "I. no 'Skipping ... validate' message" \
  "Skipping terraform validate" "$logI"

# ── Case J: marker via .terraform.lock.hcl alone is enough ─────────────
# Catches a regression where has_markers() only sees *.tf / *.tofu.
caseJ="$workdir/caseJ-lock-only"
mkdir -p "$caseJ"
cat >"$caseJ/.terraform.lock.hcl" <<'LOCK'
provider "registry.terraform.io/hashicorp/null" { version = "3.2.1" }
LOCK
stubsJ="$workdir/stubsJ"
make_stub_dir "$stubsJ" "terraform"
logJ="$caseJ/out.log"
set +e
run_verify_in "$caseJ" "static" "$stubsJ:$clean_path" "$logJ"
rcJ=$?
set -e
assert_exit "J. lock-only repo → pack runs (no early-exit-0 false negative)" 0 "$rcJ"
assert_stdout_contains "J. terraform fmt invoked from lockfile-only repo" \
  "terraform fmt" "$stubsJ/calls.log"

printf '\n── Summary ──\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
