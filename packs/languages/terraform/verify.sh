#!/usr/bin/env sh
set -eu

# HARNESS_VERIFY_MODE is set by the caller (run-verify.sh).
# Supported values: static, test, all (default).
mode="${HARNESS_VERIFY_MODE:-all}"
case "$mode" in
  static|test|all) ;;
  *)
    echo "Unknown HARNESS_VERIFY_MODE: $mode" >&2
    exit 2
    ;;
esac

init_validate="${RALPH_TERRAFORM_INIT_VALIDATE:-false}"
case "$init_validate" in
  true|false|"") ;;
  *)
    echo "Unknown RALPH_TERRAFORM_INIT_VALIDATE: $init_validate (expected true|false)" >&2
    exit 2
    ;;
esac

# Marker detection: only run the pack if Terraform/OpenTofu sources exist.
roots_file="$(mktemp "${TMPDIR:-/tmp}/ralph-terraform-roots.XXXXXX")"
cleanup() {
  rm -f "$roots_file"
}
trap cleanup EXIT HUP INT TERM

find_project_roots() {
  find . \
    \( -type d \( \
      -name .git -o \
      -name .terraform -o \
      -name .terragrunt-cache -o \
      -name node_modules -o \
      -name .dart_tool -o \
      -name target -o \
      -name dist -o \
      -name build \
    \) -prune \) -o \
    -type f \( \
      -name '*.tf' -o \
      -name '*.tofu' -o \
      -name '.terraform.lock.hcl' \
    \) -print 2>/dev/null |
    while IFS= read -r marker; do
      dirname "$marker"
    done |
    sort -u
}

root_selected() {
  [ -z "${RALPH_VERIFY_PROJECT_ROOTS:-}" ] && return 0
  root="${1#./}"
  [ -n "$root" ] || root="."

  for selected in $RALPH_VERIFY_PROJECT_ROOTS; do
    selected="${selected#./}"
    [ -n "$selected" ] || selected="."
    [ "$root" = "$selected" ] && return 0
  done
  return 1
}

find_project_roots |
  while IFS= read -r project_root; do
    if root_selected "$project_root"; then
      printf '%s\n' "$project_root"
    fi
  done > "$roots_file"

if [ ! -s "$roots_file" ]; then
  echo "Skipping Terraform verifier: no .tf / .tofu / .terraform.lock.hcl files found."
  exit 0
fi

# Pick the IaC CLI. Prefer `tofu` (OpenTofu) over `terraform`.
IAC_CLI=""
if command -v tofu >/dev/null 2>&1; then
  IAC_CLI=tofu
elif command -v terraform >/dev/null 2>&1; then
  IAC_CLI=terraform
else
  echo "Terraform/OpenTofu sources detected but neither 'tofu' nor 'terraform' is on PATH." >&2
  echo "Install one of them or remove the IaC sources to skip this verifier." >&2
  exit 1
fi

echo "Using IaC CLI: $IAC_CLI"

status=0

has_terraform_tests() {
  test_marker="$(find . \
    \( -type d \( \
      -name .git -o \
      -name .terraform -o \
      -name .terragrunt-cache -o \
      -name node_modules -o \
      -name .dart_tool -o \
      -name target -o \
      -name dist -o \
      -name build \
    \) -prune \) -o \
    -type f -name '*.tftest.hcl' -print -quit 2>/dev/null)"
  [ -n "$test_marker" ]
}

run_static() {
  # Format check
  if ! "$IAC_CLI" fmt -check; then
    echo "$IAC_CLI fmt: formatting issues detected."
    status=1
  else
    echo "$IAC_CLI fmt: ok"
  fi

  # Optional backend-less init/validate for generic CI. Backend-backed plans
  # require repo-specific credentials and must live in trusted workflows.
  if [ "$init_validate" = "true" ]; then
    if "$IAC_CLI" init -backend=false -input=false; then
      "$IAC_CLI" validate || status=1
    else
      status=1
    fi
  elif [ -d .terraform ]; then
    "$IAC_CLI" validate || status=1
  else
    echo "Skipping $IAC_CLI validate: .terraform/ not found (run '$IAC_CLI init' first)."
  fi

  # tflint (optional)
  if command -v tflint >/dev/null 2>&1; then
    tflint || status=1
  else
    echo "Skipping tflint: command not found."
  fi

  # tfsec / trivy config (optional)
  if command -v tfsec >/dev/null 2>&1; then
    tfsec . || status=1
  elif command -v trivy >/dev/null 2>&1; then
    trivy config . || status=1
  else
    echo "Skipping tfsec / trivy config: neither command found."
  fi
}

run_tests() {
  if has_terraform_tests; then
    "$IAC_CLI" test || status=1
  else
    echo "Skipping $IAC_CLI test: no terraform tests found (*.tftest.hcl)."
  fi
}

verify_root() {
  project_root="$1"
  echo "==> Terraform project root: $project_root"

  status=0
  case "$mode" in
    static) run_static ;;
    test)   run_tests ;;
    all)    run_static; run_tests ;;
    *)
      echo "Unknown HARNESS_VERIFY_MODE: $mode" >&2
      return 2
      ;;
  esac

  return "$status"
}

overall_status=0
while IFS= read -r project_root; do
  [ -n "$project_root" ] || continue
  if ! (cd "$project_root" && verify_root "$project_root"); then
    overall_status=1
  fi
done < "$roots_file"

exit "$overall_status"
