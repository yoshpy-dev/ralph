# Self-review report: add-terraform-language-pack

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md` (issue #52)
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality of `feat/52/add-terraform-language-pack` vs `main` (5 commits, 11 files, +547/−18). Spec compliance, test coverage, and doc drift are out of scope (delegated to `/verify` and `/test`).

## Evidence reviewed

- `git diff main...HEAD` (all 11 files)
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md`
- Peer packs for stylistic comparison:
  - `packs/languages/golang/verify.sh` (lines 1-12: shebang + `set -eu` + marker + CLI gates)
  - `packs/languages/python/verify.sh` and `rust/verify.sh` (same idiom)
- Peer rule files: `.claude/rules/golang.md`, `python.md`, `rust.md` (frontmatter conventions)
- Embedded-pack auto-discovery contract: `internal/scaffold/embed.go` lines 21-35 (`PackFS` / `AvailablePacks` read directly from `templates/packs/`)
- Mirror parity verified with `cmp` on all five mirror pairs:
  - `packs/languages/terraform/verify.sh` ↔ `templates/packs/terraform/verify.sh` — identical
  - `packs/languages/terraform/README.md` ↔ `templates/packs/terraform/README.md` — identical
  - `.claude/rules/terraform.md` ↔ `templates/base/.claude/rules/terraform.md` — identical
  - `scripts/detect-languages.sh` ↔ `templates/base/scripts/detect-languages.sh` — identical
  - `docs/recipes/adding-a-language-pack.md` ↔ `templates/base/docs/recipes/adding-a-language-pack.md` — identical
- Executable bit verified via `git ls-files --stage` — both `verify.sh` paths committed as `100755`.
- Behavioral smoke tests (no terraform/tofu installed on review host):
  - Empty dir → "Skipping Terraform verifier" + exit 0
  - Dir with `main.tf`, no CLI on PATH → "neither 'tofu' nor 'terraform' is on PATH" + exit 1 (fail-open correctly blocked)
  - Dir with only `.terraform.lock.hcl` → marker detected, exit 1 due to missing CLI (correct)
  - Dir with `.terraform/modules/something.tf` only → "Skipping" + exit 0 (prune works; the dot dir's contents do not count as markers)
  - Unknown `HARNESS_VERIFY_MODE=oops` without marker → exits 0 via early return (acceptable; mode validation only kicks in once markers exist)
- `scripts/detect-languages.sh` smoke-tested with a bare `.terraform/` dir: emits nothing (prune correctly suppresses output).

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `verify.sh`'s mode `case` validates the mode only after marker detection and CLI selection. A user setting `HARNESS_VERIFY_MODE=foo` in an empty dir gets a silent exit 0 instead of the configuration error. This is a minor footgun but matches no other pack's behavior (peer packs also early-exit on missing markers before mode handling), so it is consistent. | `packs/languages/terraform/verify.sh:5-23` (mode read at line 5, marker check at line 21, mode `case` at line 84) | Acceptable as-is for parity with peer packs. If you want stricter feedback, move the `case "$mode" in static\|test\|all)` validation above the marker check — but this would diverge from peers and is not required. |
| LOW | readability | The marker-detection `find` expression `find . -type d -name .terraform -prune -o -type f \( -name '*.tf' -o -name '*.tofu' \) -print` is correct but non-obvious. Same expression is duplicated in `scripts/detect-languages.sh:41` and `verify.sh:11-12` and again (with `*.tftest.hcl`) at `verify.sh:67`. | `verify.sh:11-12, 67`; `scripts/detect-languages.sh:41` | Acceptable for a small pack. If a fourth copy appears, extract a `has_iac_sources` helper. No action required for this PR. |
| LOW | maintainability | `.claude/rules/terraform.md` declares `paths: - "**/.terraform.lock.hcl"`. Whether the editor's path-matcher treats hidden-file globs the same as visible ones is non-trivial and not exercised by any existing rule file in the repo. The other three globs (`*.tf`, `*.tofu`, `*.tftest.hcl`) are unambiguous. | `.claude/rules/terraform.md:2-6` | Leave as written; treat verification of hidden-file glob behavior as `/verify`'s problem, not a diff-quality blocker. |
| INFO | maintainability | `tfsec` is officially archived (announced by Aqua in 2024) and `trivy config` is the current upstream recommendation. The pack hits both with `tfsec` preferred, then `trivy` fallback. This matches plan R2 explicitly. Worth a tracking note for a future "drop tfsec" cleanup. | `verify.sh:53-59`; plan R2 | No action this PR. Track in `docs/tech-debt/` (see below). |
| INFO | naming | The two `.tof` extensions in the find expressions are correctly `.tofu` (not `.tf` with a typo). Confirmed across all three find sites. No typos found. | grep `\.tofu` across changed files | None. |

No CRITICAL, HIGH, or MEDIUM findings.

## Positive notes

- The diff is tightly scoped to the stated objective. No unrelated files touched, no formatting-only churn outside the new pack.
- `set -eu` plus `if ! cmd; then`/`cmd || status=1` is the correct idiom for collecting failures without short-circuiting; consistent with peer packs.
- Fail-open hardening (Codex HIGH#2) is implemented exactly as the plan specified: markers + missing CLI → exit 1 with two-line guidance to stderr (the human-friendly hint about removing IaC sources is a nice touch).
- The OpenTofu dispatch (`tofu` preferred over `terraform`) is the right default — OpenTofu users explicitly self-select via the `tofu` binary's presence.
- Mirror discipline is clean: byte-identical across all five pairs, executable bit preserved on both `verify.sh` copies, and the recipe doc now includes an explicit "Mirror checklist" section that names the three pair locations new contributors must remember. This is a small but high-leverage docs improvement.
- The `validate` step is correctly gated on `.terraform/` existence, avoiding a known noise source ("validate before init") that would otherwise generate false failures.
- The recipe rewrite (`docs/recipes/adding-a-language-pack.md`) materially raises the floor for the next pack author: it spells out `detect-languages.sh` as a hand-edit, calls out the byte-identical mirror requirement, and lists the gate scripts. Worth keeping.
- No debug code, no commented-out blocks, no leftover TODO/FIXME, no hardcoded paths or secrets in the diff.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `tfsec` upstream archive | Optional check will eventually 404 / refuse new platforms. Pack already falls back to `trivy config`. | tfsec still works for the install base; ripping it out today would be churn. | When `tfsec` install fails on a supported OS or trivy ergonomics catch up enough that the fallback can become the default. | This PR (`packs/languages/terraform/verify.sh:53-59`) |
| `ralph pack add <lang>` writes to project root, not `packs/languages/<lang>/` | `ralph pack add terraform` will not produce a usable pack layout. Tracked in plan Non-goals and R6. | Pre-existing bug independent of Terraform pack. Plan deliberately scopes acceptance to `PackFS("terraform")` resolution, not the `pack add` CLI path. | Open a follow-up issue at `/pr` time; fix `internal/cli/pack.go:64-67` to pass the `packs/languages/<lang>/` subpath instead of `absDir`. | This plan's Non-goals and Risks (R6) |

Both rows should be appended to `docs/tech-debt/README.md` (or a dedicated file under `docs/tech-debt/`) per the skill's instruction.

## Recommendation

- Merge: yes — proceed to `/verify`. No CRITICAL or HIGH findings; LOW/INFO notes are documentation/follow-up only.
- Follow-ups:
  1. At `/pr` time, file a follow-up issue for the pre-existing `ralph pack add` pathing bug (Codex HIGH#1, plan Non-goals).
  2. Append the two tech-debt rows above to `docs/tech-debt/README.md` during `/sync-docs`.
  3. Let `/verify` confirm: (a) hidden-file glob matching for `**/.terraform.lock.hcl` in the rule frontmatter, (b) `scripts/check-sync.sh` and `scripts/check-skill-sync.sh` PASS, (c) `internal/scaffold.AvailablePacks()` enumerates `terraform`.
  4. Let `/test` exercise the verify.sh branches (marker absent, marker + missing CLI, marker + CLI + missing `.terraform/`, mode dispatch) — at minimum manually, per the plan's test plan.
