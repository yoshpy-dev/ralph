# Sync-docs report: org-runtime-lead

- Date: 2026-08-02
- Plan: `docs/plans/active/2026-08-02-org-runtime-lead.md`
- Maintainer: `doc-maintainer` subagent (`/sync-docs`)
- Input: drift list from `docs/reports/verify-2026-08-02-org-runtime-lead.md` (Documentation drift section) + `docs/reports/test-2026-08-02-org-runtime-lead.md` (all-green verdict)

## Changes

1. **Plan (`docs/plans/active/2026-08-02-org-runtime-lead.md`)**
   - AC-1 through AC-12 (including AC-2b and AC-11b) ticked `[x]`, each with an evidence pointer into `docs/reports/verify-2026-08-02-org-runtime-lead.md` and/or `docs/reports/test-2026-08-02-org-runtime-lead.md`; AC-11/AC-11b also point at `docs/evidence/org-lead-smoke-2026-08-02.txt`.
   - AC-2b text extended to describe the improved audit semantics from the self-review fix commit (`9a22942`): the `--scope`-less autonomous rejection path now goes through `reject()`, adding a `rejected` manifest event + `honored=false` receipt on top of the unchanged fail-closed behavior. This matches the verify report's "Self-review fix-commit crosscheck" LOW row and closes the doc-drift item it flagged ("AC-2b wording ... diverges narrowly").
   - Progress checklist: ticked "Implementation started" (with the slice commit list: `c04186e` / `9fe2af1` / `4b7a2e6`+`7be4993` / `039b18e` / smoke fixes `d35f157`+`428177b` / self-review fix `9a22942`), "Review artifact created", "Verification artifact created", "Test artifact created". Left "Plan reviewed" and "PR created" unticked — those are `/pr`'s and the human reviewer's steps, not sync-docs'.
   - `Status: Draft` left unchanged. Checked convention across archived plans: mixed (`Done`, `In review`, `Implementation complete (awaiting post-implementation pipeline)`, `Approved` all appear), but the two directly preceding plans in this same PR series — `docs/plans/archive/2026-08-01-org-runtime-mechanism.md` and `docs/plans/archive/2026-08-02-org-runtime-seats.md` — both kept `Status: Draft` through archival. No series-specific convention to change it, so left as-is per the instruction's fallback rule.

2. **`docs/tech-debt/README.md`** — the evidence-redaction row (the standalone row below the bundled 4-LOW-findings row, both concerning `org-seats-smoke-2026-08-02.txt`) now also names `docs/evidence/org-lead-smoke-2026-08-02.txt` as carrying the same unredacted local-path/UUID pattern, in the finding text, the impact clause, the "when to fix" clause, and the evidence-links column (added a pointer to this plan's own verify report). This directly resolves the verify report's Documentation drift note: "the new evidence file is not yet covered by the existing evidence-redaction tech-debt row's literal text (though its intent clearly applies)."

3. **`AGENTS.md` repo map** (root + `templates/base/AGENTS.md`, kept byte-identical) — the `internal/org/` line extended by one clause each: `seat saga manifest` → `seat saga manifest (flock-serialized)`, and a new `permission-mode envelopes` item + trailing `org report generation` item, reflecting `internal/org/lockfile.go`, `internal/org/permissions.go`, and `internal/org/report.go` added by this plan. Confirmed `diff AGENTS.md templates/base/AGENTS.md` is empty after the edit (both mirrors identical).

4. **`.claude/rules/agent-messaging.md`** — reviewed, no change needed. It documents the typed message protocol (`TYPE`/`TASK_ID` enum, star topology, size cap) and does not reference permission modes, the manifest lockfile, or `org report`; none of those are in scope for this rule doc.

## Drift checks

- `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 14 skill(s) in lock-step` (PASS, non-zero exit is the script's own convention for "warnings printed but lock-step held" — same as the verify report's reading).
- `bash ./scripts/check-sync.sh` → `IDENTICAL: 179, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3` — PASS, no new drift introduced by this doc-sync pass.
- `/org` skill mirrors (`.claude/skills/org/`, `.agents/skills/org/`, both `templates/base/` copies) — unchanged by this pass; already confirmed 4-way byte-identical in the verify report and reconfirmed green by the two checks above.

## Not changed (reviewed, no action needed)

- `docs/specs/2026-08-01-org-runtime.md` FR-2/FR-5/FR-9/FR-10 checkboxes — the verify report flagged these as stale, but they span the whole PR①–⑤ series and are explicitly out of this single PR's unilateral control (per the verify report's own framing); left for the series' final PR.
- `README.md`, `docs/quality/definition-of-done.md`, `.claude/skills/org/SKILL.md`, `internal/org/prompts/lead.md` — no drift reported against these by verify/test; not touched.

## Drift clean

Yes — after this pass, `check-skill-sync.sh` and `check-sync.sh` are both green, `AGENTS.md`/`templates/base/AGENTS.md` are byte-identical, the plan's AC/Progress checklists match the completed work reported by verify/test, and the tech-debt evidence-redaction row now names both affected evidence files.

## Next step

Proceed to `/cross-review` (optional) then `/pr`.

---

## Cycle 2

- Date: 2026-08-02
- Maintainer: `doc-maintainer` subagent (`/sync-docs`, Cycle 2)
- Input: drift lists from `docs/reports/verify-2026-08-02-org-runtime-lead.md` (Cycle 2, "Documentation drift found") and `docs/reports/self-review-2026-08-02-org-runtime-lead.md` (Cycle 2, findings L-5/L-6 and "Tech debt identified (Cycle 2)")
- Scope: plan deviations note, tech-debt tracking for two deferred-finding groups, and a check that no other doc still describes the pre-fix gate-vs-idempotent ordering

### Changes

1. **Plan (`docs/plans/active/2026-08-02-org-runtime-lead.md`, "Implementation notes (deviations)")** — added a new bullet recording the Cycle-2 fix-and-revalidate commits: `de4de50` (cross-review ACTION_REQUIRED #1 fix — idempotent-before-scope-gate reorder in Phase 1's locked closure, regression test `TestOrgSpawn_Idempotent_NoScopeRetry_ReturnsExistingSeat`) and `69be944` (self-review Cycle-2 M-1/M-2 fixes — dry-run/real path gate-ordering alignment, regression test `TestOrgSpawn_DryRun_And_Real_AgreeOnRejectionCause_EnvelopeBeforeScopeGate`; Phase-2 idempotency recheck against the post-compensation snapshot, regression test `TestOrgSpawn_StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent`), with pointers into both reports' Cycle 2 sections for the full detail.

2. **`docs/tech-debt/README.md`** — added two new rows:
   - The `/org` skill's Japanese `description:` frontmatter, deferred a second time (Cycle-1 LOW, then Cycle-2 self-review L-5) without being fixed or previously tracked. Per the self-review's own escalation rule ("a deferred finding untracked is the one outcome the deferral discipline was trying to avoid"), a second deferral gets a row.
   - A single consolidated row for the three Cycle-2 self-review LOWs explicitly marked "batchable, no urgency" (L-2 `checkCapacityAndStart`'s zero-arg closure signature, L-3 the permission-mode enum's package placement, L-4 `newOrgRuntime`'s widened signature) — one row rather than three, since all three share the same defer rationale (bundling them into this cycle's fix-and-revalidate pass, already spent on the two MEDIUM findings, was judged unnecessary churn) and the same pay-down trigger (next touch to the same three files).
   - L-1 (`permArgs`'s stale doc comment) was checked against current code and found already fixed at HEAD (`internal/org/spawn.go:378-383` now reads "stays nil on any early-return path that precedes its own assignment", matching the self-review's suggested rewording) — not part of either new row, and not itself a drift item.

3. **No other doc changes needed** — grepped `.claude/skills/org/SKILL.md`, `internal/org/prompts/lead.md`, the plan, and `docs/specs/2026-08-01-org-runtime.md` for AC-2b/scope-gate/idempotent language (item 3 of this cycle's brief: confirm no doc still describes the pre-fix gate-before-idempotency ordering). All three surviving mentions (`SKILL.md:50`, `:89`, plan AC-2b line) describe only the gate's *existence* and the `--scope` requirement, not its ordering relative to the idempotent check — none needed correction.

### Drift checks

- Re-ran `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 14 skill(s) in lock-step` (PASS; unaffected by this cycle's plan/tech-debt-only changes).
- Re-ran `bash ./scripts/check-sync.sh` → `IDENTICAL: 179, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3` (PASS, unchanged from Cycle 1).
- No skill, mirror, `AGENTS.md`, or `templates/base/` files touched this cycle — Cycle 1's `AGENTS.md` repo-map edit and the mirror byte-identity checks still hold.

### Not changed (reviewed, no action needed)

- Plan `Status: Draft` line — self-review Cycle 2 L-6 and verify Cycle 2 both flag this as stale against the now-fully-ticked AC/Progress checklists, but it was not in this cycle's assigned drift-fix list (items 1–3 above); left for `/pr` or a follow-up to decide, consistent with Cycle 1's reasoning that the two directly preceding plans in this series also stayed `Draft` through archival.
- `docs/specs/2026-08-01-org-runtime.md` FR-2/5/9/10 checkboxes — still out of this single PR's unilateral control (spans the whole PR①–⑤ series); unchanged from the Cycle 1 assessment.

### Drift clean

Yes — after this pass, `check-skill-sync.sh` and `check-sync.sh` are both green with zero drift, the plan's deviations note now covers both Cycle-2 fix commits, and both Cycle-2 self-review deferred-finding groups (L-5 alone, L-2/L-3/L-4 consolidated) are now tracked in `docs/tech-debt/README.md` rather than aging out untracked.

### Next step

Proceed to `/cross-review` (optional) then `/pr`.
