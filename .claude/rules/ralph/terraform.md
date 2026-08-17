---
paths:
  - "**/*.tf"
  - "**/*.tofu"
  - "**/*.tftest.hcl"
  - "**/.terraform.lock.hcl"
---
# Terraform / OpenTofu rules

- Keep modules small and composable; group related resources into a module rather than dumping into the root.
- Never commit `terraform.tfstate`, `*.tfstate.backup`, `.terraform/`, or `*.auto.tfvars` that contain secrets.
- Every `variable` and `output` must have a `description`. Constrain types explicitly (`type = string`, not bare).
- Pin providers in `required_providers` with `source` and a version constraint. Pin `required_version` for the IaC CLI.
- Configure a remote `backend` for any state that other people will read; avoid local-only state on shared infrastructure.
- Run `terraform fmt -check` and `terraform validate` (or the `tofu` equivalents) from each Terraform/OpenTofu root before completion.
- Prefer `for_each` (map keys) over `count` (positional) when the set of resources can change — it avoids destructive re-creation.
