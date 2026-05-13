# Verify report: add-terraform-language-pack (cycle 2/2)

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md` (issue #52)
- Verifier: verifier subagent (Claude Code)
- Scope: cycle-2 fix `f27e1a2` (one file: `tests/test-terraform-pack-verify.sh`, +30/−4). Cycle-1 baseline already PASS (`docs/reports/verify-2026-05-13-add-terraform-language-pack.md`); the cycle-2 fix targets the cross-review ACTION_REQUIRED P2 hermeticity finding, internal to test infra. Acceptance criteria unaffected.
- Evidence: `docs/evidence/verify-2026-05-13-add-terraform-language-pack-cycle2.log`
- Cycle: 2/2 (final under default `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`)

## Spec compliance

The 13 acceptance criteria are unchanged from cycle 1 (cycle-2 touches only `tests/test-terraform-pack-verify.sh`, an internal regression suite — no AC references test hermeticity).

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| `packs/languages/terraform/{README.md,verify.sh}` exist and executable | Verified (carried) | Cycle-1 verify §Spec compliance row 1; pack files untouched in cycle 2. |
| `templates/packs/terraform/` byte-identical to `packs/languages/terraform/` | Verified | `check-sync.sh` PASS (148 identical, 0 drifted) — see static analysis below. |
| `.claude/rules/terraform.md` with the four `paths:` globs | Verified (carried) | Cycle-1 verify §Spec compliance row 3; rule file untouched. |
| `templates/base/.claude/rules/terraform.md` byte-identical mirror | Verified | `check-sync.sh` PASS, no ROOT_ONLY entries. |
| `scripts/detect-languages.sh` emits `terraform` for `.tf`/`.tofu`/`.terraform.lock.hcl` (prunes `.terraform/`) | Verified (carried) | Cycle-1 verify §Spec compliance row 5; script untouched. |
| `internal/scaffold.PackFS("terraform")` resolves; `ralph pack list` includes `terraform` | Verified (carried) | Cycle-1 verify §Spec compliance row 6; no Go code or templates touched. |
| `HARNESS_VERIFY_MODE=static` invokes `fmt -check -recursive` and gated `validate` | Verified (carried) | `packs/languages/terraform/verify.sh:41-55` unchanged. |
| `HARNESS_VERIFY_MODE=test` gates `$IAC_CLI test` on `*.tftest.hcl` | Verified (carried) | `packs/languages/terraform/verify.sh:74-80` unchanged. |
| `tflint`/`tfsec`/`trivy config` skip when missing | Verified (carried) | `packs/languages/terraform/verify.sh:58-71` unchanged. |
| No markers → warning + `exit 0` | Verified (carried) | `verify.sh:20-23`; behavior unchanged. |
| Markers + neither CLI → error + `exit 1` (fail-open avoidance) | Verified (carried) | `verify.sh:26-35`; behavior unchanged. |
| Recipe doc uses `terraform`, hand-edit + mirror-checklist | Verified (carried) | Cycle-1 verify §Spec compliance row 12. |
| `scripts/check-skill-sync.sh` green | Verified | `./scripts/check-skill-sync.sh` → exit 0, 13 skills in lock-step. |

All 13 ACs: **verified at static-analysis level**, unchanged from cycle 1.

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/check-sync.sh` | PASS (exit 0) | `IDENTICAL: 148, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3`. Cycle-2 fix is test-only and cannot introduce template drift; this confirms it. |
| `./scripts/check-skill-sync.sh` | PASS (exit 0) | 13 skills in lock-step. |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | PASS (exit 0) | Composite gate: hook shellcheck + `sh -n`, `jq -e` on both settings.json files, `check-sync.sh`, `check-pipeline-sync.sh` (all 8 referenced files reference all pipeline steps), `check-skill-sync.sh`, gofmt, `go vet`, `go test ./...` cached green. Evidence: `docs/evidence/verify-2026-05-13-add-terraform-language-pack-cycle2.log`. |
| `shellcheck tests/test-terraform-pack-verify.sh` | PASS (exit 0) | No findings on the cycle-2 hermetic-PATH block. |
| `sh -n tests/test-terraform-pack-verify.sh` | PASS (exit 0) | POSIX syntax valid (matters because the file uses `#!/usr/bin/env sh`). |

## Cycle-2 fix verification (Codex advisory alignment)

The cycle-1 cross-review ACTION_REQUIRED #1 (`docs/reports/cross-review-triage-2026-05-13-add-terraform-language-pack.md:25`) flagged that `clean_path="/usr/bin:/bin"` exposed host-installed `terraform / tofu / tflint / tfsec / trivy` on common CI images, breaking the hermeticity of 8 stub-CLI scenarios.

**Fix in `f27e1a2` (lines 83–112):**

1. Fresh `$workdir/.coreutils` directory.
2. Symlinks to host-resolved paths for the minimal coreutils set (`sh find grep sed cat chmod mkdir rm printf ls test true false head tr`).
3. Fail-loud per-tool guard if a required coreutil is missing on the host.
4. Leak-guard that aborts if any of `terraform tofu tflint tfsec trivy` appears in the coreutils dir.
5. `clean_path="$coreutils_dir"` — the banned IaC tool names are **excluded by construction**, not shadowed by stubs.

This matches the Codex-recommended approach 1:1 ("temp PATH with required coreutils only, not shadowing tool names"). **Verified at static level.**

**Repo-wide residue scan:** `grep -rn 'clean_path\|"/usr/bin:/bin"\|/usr/bin:/bin' tests/ scripts/ packs/` returned zero hits for the literal `/usr/bin:/bin`; all `clean_path` hits resolve to the new `coreutils_dir`-backed variable (line 112). No other test (`test-detect-languages-terraform.sh`, `test-terraform-rule-frontmatter.sh`) uses a PATH-narrowing idiom, so the cycle-2 fix scope is exhaustive.

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| Plan acceptance criteria | In sync | No AC references test hermeticity; cycle-2 fix is internal to test infra. |
| Plan Progress checklist | Stale (`[ ] PR created`) | Expected end-of-flow state; `/pr` will tick. Not a verify failure. |
| `docs/recipes/adding-a-language-pack.md` | In sync | Cycle 1 verified; no behavior change to surface. |
| `AGENTS.md` / `CLAUDE.md` / `README.md` | In sync | None enumerate test fixtures. |
| `.claude/rules/` index | In sync | No rule change. |
| `templates/base/` mirrors | In sync | `check-sync.sh` PASS; cycle-2 fix is outside the mirror scope (tests are not mirrored). |
| `docs/tech-debt/README.md` | In sync | Cycle-1 sync-docs already appended the Terraform-related entries (`tfsec` archival, `pack add` pathing). Cycle-2 self-review identified one new LOW row (broken self-referential symlinks for builtins `printf/test/true/false`) which `/sync-docs` should pick up next; it does not block verify. |
| Test header comment (`tests/test-terraform-pack-verify.sh:1–17`) | Decision: leave as-is | The hermetic-PATH rationale is documented inline at lines 83–110, immediately adjacent to the construct it explains. Adding a duplicate paragraph to the file header would split the explanation across two locations and create a maintenance hazard. Self-review classified the existing inline block as INFO: "the right level of inline documentation for a security-relevant test fixture." **No doc drift; explicit no-op.** |

## Observational checks

- **Hermeticity property re-established (static).** Cycle-2 self-review confirmed empirically (via direct probe under `/bin/sh`): with `PATH="$coreutils_dir"`, `command -v terraform/tofu/tflint/tfsec/trivy` all return non-found, while a fake binary placed under the old `/usr/bin:/bin` PATH was resolvable. The fix demonstrably blocks the leak. /verify confirms this **statically** via the leak-guard pattern at lines 106–110 and the by-construction exclusion of IaC tool names from the symlink loop at line 95.
- **Coreutils completeness check.** Cross-referenced the symlink loop's tool list against every external command invoked by `packs/languages/terraform/verify.sh`. Required external commands (`find`, `grep`, `sed`) are all in the loop. Builtins (`echo`, `printf`, `test`, `true`, `false`) work via `/bin/sh` builtin dispatch regardless of PATH.
- **Cycle-2 self-review LOW findings.** Two LOW findings (broken self-referential symlinks for builtins; optional `[ -e ]` → `[ -L ]` tightening of the leak-guard) are non-blocking and have a tech-debt row queued for `/sync-docs`. They do not affect spec compliance or static analysis.
- **Untracked report files.** `docs/reports/cross-review-triage-2026-05-13-add-terraform-language-pack.md` and `docs/reports/self-review-2026-05-13-add-terraform-language-pack-cycle2.md` are untracked. They will be picked up at commit time before `/pr`. Not a verify failure.

## Coverage gaps

- **Behavioral confirmation that the 32 tests still pass under the cycle-2 hermetic PATH** — delegated to `/test`. Static analysis cannot run them.
- **Behavioral confirmation that the hermetic PATH does not hide a required coreutil on the CI image** (e.g. GitHub Actions `ubuntu-latest`) — delegated to `/test`.
- **No additional deterministic verifier is needed.** The existing gates (`check-sync.sh`, `check-skill-sync.sh`, `run-static-verify.sh`, `shellcheck`, `sh -n`) cover the static surface of the cycle-2 fix completely. If hardened further: a unit-style assertion that `command -v terraform` returns empty under `PATH="$coreutils_dir"` would belong in `/test`, not `/verify`.

## Verdict

- **Verified.** All 13 acceptance criteria remain met. `check-sync.sh`, `check-skill-sync.sh`, `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh`, `shellcheck`, and `sh -n` all PASS (exit 0). The cycle-2 fix is internal to test infra, does not touch any AC-covered surface, matches the Codex-recommended approach 1:1, and the residue scan confirms no `/usr/bin:/bin` literal remains anywhere in `tests/ scripts/ packs/`.
- **Likely but unverified:** Behavioral pass of the 32 assertions under the cycle-2 PATH; CI-host coreutils availability. Both delegated to `/test`.
- **Not verified:** Same as cycle 1 — hidden-file glob runtime behavior in Claude Code's editor matcher (consumer is outside this repo).
- **Documentation drift:** None blocking. Plan Progress checklist `[ ] PR created` will be ticked by `/pr`. Decision on test-header expansion: no-op (inline rationale already adjacent to the construct).

**Pass.** Proceed to `/test` for cycle 2.
