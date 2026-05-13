# Verify report: add-terraform-language-pack

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md` (issue #52)
- Verifier: verifier subagent (Claude Code)
- Scope: spec compliance + static analysis for `feat/52/add-terraform-language-pack` (5 commits vs `main`, 11 files, +547/-18). Behavioral tests are delegated to `/test`.
- Evidence: `docs/evidence/verify-2026-05-13-add-terraform-language-pack.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| `packs/languages/terraform/README.md` and `verify.sh` exist; `verify.sh` is executable | Verified | `ls -la packs/languages/terraform/` shows both files; `test -x` returns 0; `git ls-files --stage` mode 100755 (per self-review). |
| `templates/packs/terraform/` is byte-identical to `packs/languages/terraform/` | Verified | `diff packs/languages/terraform/README.md templates/packs/terraform/README.md` → empty; same for `verify.sh`. `scripts/check-sync.sh` PASS (148 identical, 0 drifted). |
| `.claude/rules/terraform.md` exists; `paths:` frontmatter scopes `**/*.tf`, `**/*.tofu`, `**/*.tftest.hcl`, `**/.terraform.lock.hcl` | Verified | `Read` of `.claude/rules/terraform.md` lines 2-7 shows exactly those four globs in `paths:` array. |
| `templates/base/.claude/rules/terraform.md` byte-identical to `.claude/rules/terraform.md` | Verified | `diff` returns empty; `scripts/check-sync.sh` PASS with no `ROOT_ONLY` entries. |
| `scripts/detect-languages.sh` emits `terraform` for `*.tf` / `*.tofu` / `.terraform.lock.hcl` (and prunes `.terraform/`) | Verified (static) | `grep -n terraform scripts/detect-languages.sh` shows line 41 with the prune + extension predicate and `emit terraform`. Behavioral fixture exercise belongs to `/test` (self-review smoke-tested empty `.terraform/` → no emit). |
| `internal/scaffold.PackFS("terraform")` resolves; `ralph pack list` includes `terraform` | Verified | `go run ./cmd/ralph pack list` output: `dart, golang, python, rust, terraform, typescript` (exit 0). |
| `HARNESS_VERIFY_MODE=static` invokes `fmt -check -recursive` and (if `.terraform/` present) `validate`, skipping uninitialised state | Verified (static) | `Read` of `verify.sh:41-55`: `run_static` calls `$IAC_CLI fmt -check -recursive`, then gated `validate` on `[ -d .terraform ]`. Behavioral execution path is `/test`'s scope. |
| `HARNESS_VERIFY_MODE=test` runs `$IAC_CLI test` only when `*.tftest.hcl` exists | Verified (static) | `verify.sh:74-80` shows `find ... -name '*.tftest.hcl'` gate before `$IAC_CLI test`; else "Skipping ... test" message. |
| `tflint` / `tfsec` / `trivy config` skip when missing (do not fail CI) | Verified (static) | `verify.sh:58-71` uses `command -v` guards with `echo "Skipping ..."` fallbacks. |
| No markers → warning + `exit 0` | Verified (static) | `verify.sh:20-23`: `if ! has_markers; then echo "Skipping ..."; exit 0; fi`. Self-review smoke test confirmed exit 0. |
| Markers present + neither `terraform` nor `tofu` → error + `exit 1` (fail-open avoidance) | Verified (static) | `verify.sh:26-35`: empty `IAC_CLI` branch prints two-line stderr message and `exit 1`. Self-review smoke test confirmed exit 1. |
| `docs/recipes/adding-a-language-pack.md` uses `terraform` as the worked example, mentions hand-edit to `detect-languages.sh`, includes mirror-checklist section | Verified | `grep -n terraform docs/recipes/adding-a-language-pack.md`: line 3 "Worked example: adding a Terraform / OpenTofu pack"; line 8 `new-language-pack.sh terraform`; line 18 mirror `cp -r`; line 25 mirror to `templates/base/.claude/rules/terraform.md`; line 31 `detect-languages.sh` hand-edit. |
| `scripts/check-skill-sync.sh` green | Verified | `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`, exit 0. |

All 13 acceptance criteria: verified at static-analysis level.

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/check-sync.sh` | PASS (exit 0) | Summary: `IDENTICAL: 148, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3`. No new ROOT_ONLY entries from terraform pack — both the pack pair and the rule pair are mirrored correctly. |
| `./scripts/check-skill-sync.sh` | PASS (exit 0) | 13 skills in lock-step; this PR did not touch skills, sanity-only. |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | PASS (exit 0) | Composite gate: `check-sync.sh`, `check-skill-sync.sh`, post-impl pipeline reference check (CLAUDE.md / AGENTS.md / README.md / loop / cross-review / subagent-policy / definition-of-done all reference all pipeline steps), gofmt, go vet, `go test ./...` cached green. Evidence: `docs/evidence/verify-2026-05-13-add-terraform-language-pack.log`. |
| `shellcheck packs/languages/terraform/verify.sh templates/packs/terraform/verify.sh` | PASS (exit 0) | No findings. |
| `sh -n packs/languages/terraform/verify.sh` | PASS | POSIX syntax valid. |
| `go run ./cmd/ralph pack list` | PASS (exit 0) | Output lists `terraform` alongside `dart, golang, python, rust, typescript`. Confirms `internal/scaffold.AvailablePacks()` enumerates terraform via embedded FS auto-discovery (no code change needed; per plan assumption). |
| `go test ./internal/scaffold/... -run 'AvailablePacks|PackFS'` | PASS (exit 0) | `TestAvailablePacks_WithMockFS` SKIP (host EmbeddedFS not initialised in unit context — expected, pre-existing); `TestAvailablePacksExcludesTemplate` PASS. The live `pack list` is the load-bearing check for AvailablePacks enumeration. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/recipes/adding-a-language-pack.md` | In sync | Rewritten end-to-end with terraform as the worked example, including a "Mirror checklist" section new contributors must follow. |
| `AGENTS.md` repo map | In sync | The map references `packs/languages/` and `templates/packs/` generically without enumerating language names — no update needed. |
| `CLAUDE.md` | In sync | Does not enumerate language packs. |
| `README.md` | In sync | Does not enumerate language packs. |
| `.claude/rules/` index | In sync | New `terraform.md` follows the exact frontmatter and prose conventions of `golang.md` / `python.md` / `rust.md` / `dart.md` / `typescript.md`. No rules-policy file lists language names. |
| `templates/base/.claude/rules/` mirror | In sync | `terraform.md` mirrored byte-identically; verified by `check-sync.sh`. |
| `.codex/` parity | In sync (n/a) | Codex-side config is shape-mirrored from `templates/base/.codex/`; no Codex changes were required by this plan. |
| Plan progress checklist | Mostly in sync | Acceptance criteria checkboxes are all `[x]`. Progress checklist still has `[ ] Review artifact created`, `[ ] Verification artifact created`, `[ ] Test artifact created`, `[ ] PR created` — these are routine end-of-flow updates (the review artifact does exist at `docs/reports/self-review-2026-05-13-add-terraform-language-pack.md` but the plan was not updated to tick that box). Doc drift, not a verify failure; `/sync-docs` will reconcile. |

## Observational checks

- Self-review follow-up (a) — **hidden-file glob `**/.terraform.lock.hcl` in `.claude/rules/terraform.md` frontmatter**: At static analysis, the frontmatter is well-formed YAML and the literal is the conventional doublestar form (peer rules use `**/*.tf`-style globs; no peer uses a leading-dot file pattern, so terraform is the first). The repo does **not** ship a programmatic consumer for `paths:` frontmatter — only `.claude/skills/sync-docs/SKILL.md:27` mentions it as a doc-drift checklist item, and Claude Code's editor-side path matcher consumes it at runtime. Whether the runtime matcher fires on hidden files is a behavioral question that cannot be verified deterministically in this skill; it should be exercised in `/test` (or accepted as "likely correct" given the doublestar literal matches the same dot-leading file in `detect-languages.sh` line 41 and `verify.sh` line 10 with no issue). Marked as **likely but unverified** at this stage.
- Self-review follow-up (b) — **`internal/scaffold.AvailablePacks()` enumerates `terraform`**: Confirmed live via `go run ./cmd/ralph pack list` (output includes `terraform`). The `AvailablePacks` implementation reads directly from `templates/packs/` via `go:embed`, so the new `templates/packs/terraform/` directory is auto-discovered with no Go code change. **Verified.**
- Marker-detection idiom (`find . -type d -name .terraform -prune -o -type f \( -name '*.tf' -o -name '*.tofu' \)`) is duplicated across `scripts/detect-languages.sh:41` and `packs/languages/terraform/verify.sh:11-12,75`. Self-review flagged this as LOW maintainability. Not a verify failure; acceptable for a three-site duplication.
- `tfsec` upstream archival risk is tracked in self-review's "Tech debt identified" section and will be appended to `docs/tech-debt/README.md` during `/sync-docs`.

## Coverage gaps

- **Behavioral exercise of `verify.sh` branches** (no markers, markers + missing CLI, markers + CLI + missing `.terraform/`, mode dispatch): out of `/verify` scope. Self-review performed manual smoke tests; `/test` should formalise these.
- **`scripts/detect-languages.sh` integration test** for the `.tofu`-only and `.terraform.lock.hcl`-only edge cases: behavioral, delegated to `/test`.
- **Hidden-file glob matcher behavior**: not deterministically verifiable from repo files alone (depends on the Claude Code editor's runtime path matcher).
- No new deterministic verifier needed. The existing gates (`check-sync.sh`, `check-skill-sync.sh`, `run-static-verify.sh`, `go test ./...`) cover the static surface area of this PR completely.

## Verdict

- **Verified**: All 13 acceptance criteria pass at static-analysis level. `check-sync.sh`, `check-skill-sync.sh`, `run-static-verify.sh` (HARNESS_VERIFY_MODE=static), `shellcheck`, `sh -n`, and `ralph pack list` all PASS (exit 0). `internal/scaffold.AvailablePacks()` enumerates `terraform`. Mirror parity holds across `packs/languages/terraform/` ↔ `templates/packs/terraform/` and `.claude/rules/terraform.md` ↔ `templates/base/.claude/rules/terraform.md`. Documentation is in sync.
- **Likely but unverified**: Runtime behavior of Claude Code's editor path matcher on the `**/.terraform.lock.hcl` hidden-file glob (consumer is outside this repo; idiom is conventional).
- **Not verified**: Behavioral exercise of `verify.sh` branches (delegated to `/test`).
- **Documentation drift**: Plan's Progress checklist has stale checkboxes for review/verify/test/PR artifacts — `/sync-docs` will reconcile. Not a verify failure.

**Pass.** Proceed to `/test`.
