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
