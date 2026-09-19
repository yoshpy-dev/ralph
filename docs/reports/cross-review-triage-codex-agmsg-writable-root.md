# Cross-review triage report: codex-agmsg-writable-root

- Date: 2026-09-19 (cycle 1 and cycle 2)
- Plan: docs/plans/active/2026-09-19-codex-agmsg-writable-root.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: cycle 1 = 3, cycle 2 = 1
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=1, DISMISSED=0
- Note: the summary line reflects cycle 2 (current state). Cycle 1 was ACTION_REQUIRED=1, WORTH_CONSIDERING=2, DISMISSED=0; all three were fixed (rows kept below as history, marked resolved) and re-verified in cycle 2.
- User decision (2026-09-19, AskUserQuestion): fix all three findings (AR-1, WC-1, WC-2) and re-run the full pipeline as cycle 2/2

## Triage context

- Active plan: docs/plans/active/2026-09-19-codex-agmsg-writable-root.md (pinned in `.harness/state/standard-pipeline/active-plan.json`)
- Self-review report: docs/reports/self-review-2026-09-19-codex-agmsg-writable-root.md (cycle 1: 3 MEDIUM + 6 LOW, all fixed; revalidation: 6 LOW, all fixed without a second reviewer pass)
- Verify report: docs/reports/verify-2026-09-19-codex-agmsg-writable-root.md (PASS, no drift)
- Implementation context summary: `ralph doctor` gained a static check that reads the user-level codex `config.toml` and warns when a codex seat of this project could run under `workspace-write` but no `[sandbox_workspace_write].writable_roots` entry covers agmsg's message store. The user chose "detect and point to the recipe" over having ralph add a writable root to seats automatically. The check is deliberately biased towards warning: a false `pass` hides the failure it exists to catch (a seat that cannot send RESULT), while a false `warn` costs one look at the recipe. The reasons were made config-aware in self-review (driver pool, role-resolved permission modes). The reviewer ran `codex exec review --base main` (codex-cli 0.154.0, `-m gpt-6-astra -c model_reasoning_effort=xhigh`, stdin closed) at HEAD 792bce5; its findings are from static inspection plus codex's own documentation. Each finding was checked before classification: the protected-path rule against the documentation page the reviewer cites (it states that `.git`, `.agents`, and `.codex` directories inside writable roots are read-only, recursively), the role/model gate against `internal/org/envelope.go` (`ValidateSpawnEnvelope` → `modelAllowedForRole`), and the `/tmp` claim against `docs/evidence/codex-seat-permissions-2026-09-18.md` P4.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 | **(resolved in ab09dbc + 735fc45 + 5fcf070, cycle 2)** [P2] An ancestor root is accepted even when the path from it to the store passes through a directory codex protects. With the home directory in `writable_roots`, doctor reports that it covers the default store `~/.agents/skills/agmsg/db`, but codex keeps `.agents` beneath a writable root read-only, recursively, so the seat still cannot write the database. | Real, and it is the one direction this check must never get wrong: a `pass` for a configuration in which RESULT delivery still fails. Listing the home directory as a writable root is a plausible thing for an operator to do, and the default agmsg home lives under `.agents`. The live evidence only covers the explicit `db` directory as the root (#155 Run A2), which is unaffected. Fix: a root covers the store only when no path element between the root and the store is `.git`, `.agents`, or `.codex`; the Detail recommends the explicit store directory. Whether codex protects only the top-level entry of a root or nested ones too is not documented precisely, so the check treats any such element as protected (warn-biased). | `internal/cli/doctor_codex_writable_root.go` (`pathCovers` / `coveringWritableRoot`), recipe + skill wording if they mention ancestor roots |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| WC-1 | **(resolved in ab09dbc + 91a5542, cycle 2)** [P2] Implicit temporary roots are ignored: when `AGMSG_STORAGE_PATH` points into `/tmp` (writable by default under workspace-write), the check still warns that RESULT delivery is impossible. | Real but it errs on the safe side (a false warn, not a false pass) and needs an unusual setup (a message store under a temp directory). The repository's own evidence (P4) confirms `/tmp` is a writable root. Cheap fix: treat `/tmp` and `$TMPDIR` as implicit roots unless `[sandbox_workspace_write].exclude_slash_tmp` / `exclude_tmpdir_env_var` is true, and say in the Detail when the store is covered by an implicit root. Real issue: yes (small). Worth fixing: debatable on its own, reasonable alongside AR-1 because it touches the same comparison. | `internal/cli/doctor_codex_writable_root.go` (config struct, coverage) |
| WC-2 | **(resolved in ab09dbc + c03c21e, cycle 2)** [P2] Permission overrides are not intersected with model eligibility: with a guarded default, an autonomous override for one role, and `[org.roles]` restricting that role to claude models, the check counts an autonomous codex seat as possible although `ValidateSpawnEnvelope` rejects every codex model for that role. | Real (confirmed in `internal/org/envelope.go`), again a false warn rather than a false pass, and it needs a specific configuration. It is the same class as self-review LOW-6, which was fixed in code, so leaving this half of it out is inconsistent. Fix stays pure and table-testable: a role's mode counts only if some codex model in `[org].model_pool` is permitted for that role; the default mode counts only if the pool has a codex model at all; no codex model in the pool means no codex seat, like the driver-pool gate. Real issue: yes (small). Worth fixing: debatable. | `internal/cli/doctor_codex_writable_root.go` (`codexSeatModesPossible`) |
| WC-3 | (cycle 2) [P2] Workspace coverage is not considered: when the agmsg home or the `AGMSG_STORAGE_PATH` directory lies inside the seat's working directory, workspace-write already permits the database write, yet the check warns that the seat cannot send RESULT and recommends a persistent writable root that is not needed. The reviewer reproduced the warning with codex-cli 0.154.0 while a sandboxed SQLite write to the same project-local store succeeded. | Real, and in the safe direction (an unneeded warn, not a false pass). It needs a project-local message store, which is not the default (`~/.agents/skills/agmsg`), and one that is not under the project's own `.agents`, `.git`, or `.codex` directory, because codex keeps those read-only inside the workspace too. The check cannot know a seat's `--cwd`: `ralph org spawn` sets it per seat, and seats often run in task worktrees below the project root, where a store at the project root is NOT inside the seat's workspace. Treating the project directory as a writable root would therefore risk the false pass this check must avoid. The honest fix is to qualify the warn when the store lies under the project directory (a seat whose working directory contains the store can already write it) and add a regression test; it changes only wording. Real issue: yes (small). Worth fixing in this PR: debatable under the cycle cap. | `internal/cli/doctor_codex_writable_root.go` (warn Detail), recipe + skill wording |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 2 (2026-09-19, after the cycle 1 fixes)

- Re-run after the user chose "fix all three and re-run the full pipeline": self-review cycle 2 (merge; 1 MEDIUM + 7 LOW, all addressed in 235abb0 / 735fc45 / 5fcf070), verify cycle 2 (PASS, d2495e4; three non-blocking notes applied in a16e3f6), test cycle 2 (PASS, 5e939d1: 202 targeted tests, `TMPDIR=/tmp` CI simulation ok, race-clean, 17 built-binary probes; two uncovered branches closed in c83ce78), sync-docs cycle 2 (d18fc36 / aaf876b: tech-debt row extended).
- Reviewer run: `codex exec review --base main` (codex-cli 0.154.0, `-m gpt-6-astra -c model_reasoning_effort=xhigh`, stdin closed) at HEAD aaf876b. The reviewer confirmed the CLI test suite passes and reproduced its one finding with a fixture.
- Findings: 1 (WC-3 above), WORTH_CONSIDERING. None of the cycle 1 findings was re-reported.
- Case B with the cap reached (cycle 2/2): no automatic re-run. The user decides between raising the cap and re-running, creating the PR with WC-3 recorded as a known gap, and aborting. The decision is appended below.

