# Terraform pack

Verifies Terraform / OpenTofu projects. The pack auto-selects the IaC CLI: it prefers `tofu` when available, falls back to `terraform`.

Default verification order:

- static mode (`HARNESS_VERIFY_MODE=static`):
  - `fmt -check` in each detected Terraform/OpenTofu root
  - `validate` (only when `.terraform/` exists — skipped with a warning otherwise, since `validate` requires `init`)
  - `tflint` (if available)
  - `tfsec` (if available) or `trivy config .` (fallback)
- test mode (`HARNESS_VERIFY_MODE=test`):
  - `test` subcommand when any `*.tftest.hcl` file exists (Terraform 1.6+ / OpenTofu)
  - prints "no terraform tests" and skips otherwise

Activation:

- The pack runs only when at least one `*.tf`, `*.tofu`, or `.terraform.lock.hcl` file exists in the project. In monorepos, each marker directory is treated as a root and cache directories such as `.terraform/` are pruned. With no markers, it exits 0 silently.
- If markers are present but neither `tofu` nor `terraform` is on `PATH`, the pack exits **1** to avoid fail-open verification.

Customize this pack if your repo uses:
- multiple root modules with non-default layouts
- a custom `tflint` config (`.tflint.hcl`)
- workspace overlays
- backend-specific init flags that need to run before `validate`
