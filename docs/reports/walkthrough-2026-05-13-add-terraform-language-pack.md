# Walkthrough: add Terraform / OpenTofu language pack

- Branch: `feat/52/add-terraform-language-pack`
- Issue: #52
- Plan: `docs/plans/archive/2026-05-13-add-terraform-language-pack.md`
- Diff stats: 33 files, +2603/-19 lines (large diff justifies this walkthrough)
- Pipeline cycles: 3 (cap extended from 2 → 3 to fix gitignore safety net)

## Reading order for reviewers

If you only have 10 minutes, start at **§1 → §2 → §3** and skip the rest.

1. **Pack body** — verifier and its README (`packs/languages/terraform/`)
2. **Wiring** — `scripts/detect-languages.sh` change + the rule file scoping
3. **Safety net** — `.gitignore` patterns and the regression test that locks them in
4. (optional) **Mirrors** — `templates/packs/...`, `templates/base/...` (byte-identical copies)
5. (optional) **Pipeline artifacts** — `docs/reports/*-cycle{1,2,3}.md` (process trail, not implementation)

## §1 — Pack body (the actual verifier)

`packs/languages/terraform/verify.sh` (92 lines):

- `#!/usr/bin/env sh` + `set -eu` per peer-pack convention.
- **Marker detection**: scans for `*.tf` / `*.tofu` / `.terraform.lock.hcl`. The `find` expression prunes `.terraform/` so transient init artifacts don't trigger.
- **CLI dispatch**: `IAC_CLI="$(command -v tofu || command -v terraform || true)"` — OpenTofu preferred, with terraform fallback. If both are missing AND markers present → `exit 1` (Codex /plan HIGH#2 fail-open avoidance).
- **Mode switch** (`HARNESS_VERIFY_MODE` = `static` / `test` / `all`):
  - `static`: `fmt -check -recursive` → `validate` (only if `.terraform/` exists) → `tflint` (optional) → `tfsec` or `trivy config` (optional).
  - `test`: `*.tftest.hcl` present → `$IAC_CLI test`; otherwise skip with "no terraform tests".
- **Activation contract** documented in the README.

Why: the pack lives in `packs/languages/`, which `scripts/run-verify.sh:42` discovers via `detect-languages.sh`. The cycle-1 P2 finding (test hermeticity) is closed by `tests/test-terraform-pack-verify.sh` using a symlink-only PATH sandbox.

## §2 — Wiring (detection + rule scoping)

`scripts/detect-languages.sh`: emits `terraform` when any of the markers exist. Mirror at `templates/base/scripts/detect-languages.sh`.

`.claude/rules/terraform.md`: minimal HCL conventions (module sizing, state hygiene, provider pinning, `for_each` vs `count`). `paths:` frontmatter scopes to `**/*.tf` / `**/*.tofu` / `**/*.tftest.hcl` / `**/.terraform.lock.hcl`. Mirror at `templates/base/.claude/rules/terraform.md`.

`internal/scaffold/embed.go` requires NO code change — `AvailablePacks()` auto-discovers `templates/packs/terraform/`. Verified live: `ralph pack list` now includes `terraform`.

## §3 — Safety net (gitignore + regression test)

This was added in cycle 3 in response to Codex cross-review WORTH_CONSIDERING.

`.gitignore` (mirrored to `templates/base/.gitignore`, byte-identical): adds `terraform.tfstate*`, `.terraform/`, `*.tfplan`, `*.auto.tfvars*`, override files, and crash logs. Without these, the rule in §2 was prose-only; a `git add .` could stage state files with provider credentials.

`tests/test-terraform-gitignore.sh` (47 assertions): builds a hermetic temp git repo with throwaway `HOME` / `GIT_CONFIG_*`, materializes a sentinel file for each pattern, asserts `git check-ignore -v` returns the expected `<file>:<line>:<pattern>`. Negative control: `main.tf`, `variables.tf`, `outputs.tf`, `modules/vpc/main.tf` must NOT be ignored. Root vs `templates/base/` parity is asserted per pattern.

## §4 — Mirrors (skip if you trust check-sync.sh)

`scripts/check-sync.sh` enforces byte-equality on 4 mirror pairs introduced here:

- `packs/languages/terraform/` ↔ `templates/packs/terraform/`
- `.claude/rules/terraform.md` ↔ `templates/base/.claude/rules/terraform.md`
- `scripts/detect-languages.sh` ↔ `templates/base/scripts/detect-languages.sh`
- `.gitignore` ↔ `templates/base/.gitignore`
- `docs/recipes/adding-a-language-pack.md` ↔ `templates/base/docs/recipes/adding-a-language-pack.md`

All PASS (148 identical / 0 drifted / 0 root-only).

## §5 — Pipeline artifacts (process trail)

Three pipeline cycles produced 11 report files under `docs/reports/`. Summary:

| Cycle | Trigger | self-review | verify | test | sync-docs | cross-review |
|-------|---------|-------------|--------|------|-----------|--------------|
| 1 | Initial impl | MERGE | PASS | PASS (114/114) | done | Codex ACTION_REQUIRED (test hermeticity) |
| 2 | Hermeticity fix (`f27e1a2`) | MERGE | PASS | PASS (147 incl. cycle-1) | done | Codex WORTH_CONSIDERING (gitignore) |
| 3 | Gitignore fix (`03c5598`) + regression test (`68cc41f`) | MERGE | PASS | PASS (155/155) | done | Case C (review timed out during inspection; no structured findings emitted; cycle 1-2 issues already RESOLVED) |

Final test count: 155 assertions (114 cycle-1 + 9 cycle-3 baseline re-run accounted in test runner + 47 gitignore behavioral test — see cycle-3 test report for exact accounting).

## §6 — Out of scope (deferred)

Tracked in `docs/tech-debt/README.md`:

1. **`ralph pack add <lang>` pathing bug** (`internal/cli/pack.go:64-67`) — pre-existing, affects all packs equally, surfaced by Codex /plan HIGH#1 and re-confirmed by cycle-2 cross-review. Fix recipe documented in tech-debt. Workaround: ship packs via `ralph init` with `packs:` array.
2. **`tfsec` upstream-archived** — Aqua deprecated 2024, `trivy config` is the successor. Pack already falls back to trivy.
3. **Minor canonical-list divergences from `github/gitignore/Terraform.gitignore`** (4 LOW points, consolidated as one row): `*.tfstate.*.backup` vs canonical `*.tfstate.*`, narrower `*.auto.tfvars*` vs `*.tfvars`, missing `.terraformrc`/`terraform.rc`, block comment header understates scope.

## §7 — Risks and rollback

- Risk: new pack auto-activates if any `*.tf` file appears anywhere in the repo. Mitigation: marker detection prunes `.terraform/`, marker absence → silent exit 0.
- Risk: `tofu` preference may surprise users who only have `terraform` and a stale `tofu` shim. Mitigation: README documents the dispatch order.
- Rollback: pack is additive. Reverting this branch removes all new files; the only meta-repo change is detect-languages.sh, which is a 4-line block. No data migration, no destructive operations.
