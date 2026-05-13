#!/usr/bin/env sh
set -eu

# HARNESS_VERIFY_MODE is set by the caller (run-verify.sh).
# Supported values: static, test, all (default).
mode="${HARNESS_VERIFY_MODE:-all}"

# Marker detection: only run the pack if Terraform/OpenTofu sources exist.
has_markers() {
  if [ -f .terraform.lock.hcl ]; then
    return 0
  fi
  if find . -type d -name .terraform -prune -o -type f \
      \( -name '*.tf' -o -name '*.tofu' \) -print 2>/dev/null | grep -q .; then
    return 0
  fi
  return 1
}

if ! has_markers; then
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

run_static() {
  # Format check
  if ! "$IAC_CLI" fmt -check -recursive; then
    echo "$IAC_CLI fmt: formatting issues detected."
    status=1
  else
    echo "$IAC_CLI fmt: ok"
  fi

  # Validate (only if init has been run)
  if [ -d .terraform ]; then
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
  if find . -type d -name .terraform -prune -o -type f -name '*.tftest.hcl' -print 2>/dev/null | grep -q .; then
    "$IAC_CLI" test || status=1
  else
    echo "Skipping $IAC_CLI test: no terraform tests found (*.tftest.hcl)."
  fi
}

case "$mode" in
  static) run_static ;;
  test)   run_tests ;;
  all)    run_static; run_tests ;;
  *)
    echo "Unknown HARNESS_VERIFY_MODE: $mode" >&2
    exit 2
    ;;
esac

exit "$status"
