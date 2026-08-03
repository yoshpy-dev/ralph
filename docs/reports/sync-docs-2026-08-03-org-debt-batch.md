# Sync-docs report: org-debt-batch

- Date: 2026-08-03
- Plan: `docs/plans/active/2026-08-03-org-debt-batch.md`
- Maintainer: `doc-maintainer` subagent (`/sync-docs`)
- Diff scope: `fde0e84...HEAD` (branch `chore/org-debt-batch`, HEAD `bf37b24`)
  — C2-3 path API centralization, C2-5 guard completion, C2-6 insights
  repoint to org receipts with new output contract, watchdog 4 fixes
  (incl. `status`/`watch` `(source: ...)` annotation and new
  `state_dir_source` JSON field), upgrade removal smoke test.
- Input: the `/verify` report (`docs/reports/verify-2026-08-03-org-debt-batch.md`)
  already did a thorough doc-drift sweep and flagged one wording gap plus
  confirmed several docs already in sync. This pass verifies those findings
  against current source, fixes the flagged gap, and does the requested
  targeted checks (2)-(6) below.

## Changes

1. **Plan AC-6 wording fix** (`docs/plans/active/2026-08-03-org-debt-batch.md`)
   — `/verify` flagged that AC-6 says "5 row" but only 4 rows in
   `docs/tech-debt/README.md` were pre-registered and resolved; the 5th
   named item (upgrade removal-path smoke) never had a register row and
   was instead closed via AC-5's alternate path (a new automated regression
   test). Added a verify-time correction note directly under AC-6 recording
   the actual outcome (4 rows RESOLVED + 1 new automated test for a
   previously-unregistered gap) so a future reader doesn't go looking for a
   5th row that was never registered. Left the original wording intact
   (struck nothing) — this is an additive clarification, not a rewrite of
   history.

2. **`.claude/rules/model-routing.md`** ("Org runtime model receipts"
   section) — verified the section's path/schema description
   (`.harness/state/org/model-receipts.jsonl`, `ts / org_id / seat_id /
   role / driver / commanded_model / reported_effective_model / honored /
   reason`) still matches `internal/org/receipts.go`; it does. Added one
   sentence noting that `ralph insights` now reads this file by default
   (same org state-dir precedence as `ralph org` verbs) and aggregates it
   by `org_id` x `seat_id` with tri-state `honored`, so the rule doc points
   at the new C2-6 consumer instead of leaving the receipts file undocumented
   as to who reads it.

3. **Plan progress checklist** — ticked "Sync-docs artifact created".

## Checked, no change needed

- **`docs/insights/README.md`** — does not describe the `--receipts`
  section at all (confirmed via `grep -i receipt`, zero hits); it documents
  the `docs/insights/events/` JSONL schema only, which C2-6 did not touch.
  The CLI's own `--help` text (`internal/cli/insights.go`'s `Long` string)
  is the authoritative source for the `--receipts` flag/path/schema
  description and was already updated as part of the implementation
  (confirmed present: default-path precedence, Receipts section
  description, tri-state honored semantics). No stale content found, so no
  edit made — matches `/verify`'s own "N/A" finding for this file.
- **`.claude/skills/org/SKILL.md`** (+ 3 mirrors: `.agents/skills/org/`,
  `templates/base/.claude/skills/org/`, `templates/base/.agents/skills/org/`)
  — grepped for `ralph status`, `ralph org watch`, `state-dir`, and
  `(source:` — no literal output/banner examples exist in any of the four
  copies (only prose references to `--state-dir` precedence and the
  `watch` verb). Per the instruction to update only exact-output claims
  that would now be wrong, not for cosmetic drift: there is no exact-output
  claim to fix, so no edit made. Since no skill body changed, `sync-skills.sh`
  / `check-skill-sync.sh` were not run for this reason (see Verification).
- **`README.md` / `AGENTS.md`** — grepped for `receipt`/`model-receipts`;
  `README.md` has zero hits, `AGENTS.md` has two hits, both in the
  `internal/org/` and `internal/insights/` Repo map one-liners describing
  what the packages contain generically ("receipts", "insight
  event/receipt readers") — neither names a path, so neither can go stale
  when the path changes. No edit made.
- **`docs/tech-debt/README.md`** — spot-checked all 5 rows this batch
  touches (watchdog-4-LOW, C2-5 guard, C2-3 footgun API, C2-6 insights
  repoint, plus the new unresolved `Cutoff` ratchet row) against current
  source (`internal/org/watch.go`, `tests/test-no-loop-references.sh`,
  `internal/org/manifest.go`/`receipts.go`, `internal/cli/insights.go`).
  All closure comments are accurate and cite real commits/line numbers.
  No edit made (this file was already updated by the implementation
  slices, not deferred to `/sync-docs`).
- **`docs/specs/2026-08-01-org-runtime.md`** — grepped for
  `Escalated`/`state_dir_source`/`zero-active-seats`/`scope-change`/
  `porcelain`/`truncat`; zero hits. This is expected: the spec documents
  FR-level watchdog design (two-layer architecture, deadman clause), not
  implementation-level details like a specific field name or a prune cap.
  The four watchdog fixes in this batch are within-FR bug fixes, not
  FR-level behavior changes, so no spec update is warranted.

## Verification

- `./scripts/check-sync.sh` → `PASS: all files in sync.` (`DRIFTED: 0`,
  `.claude/rules/model-routing.md` reported as the pre-existing
  `KNOWN_DIFF` between root and `templates/base/`, unaffected by this
  session's one-sentence addition since the diff was already tracked as a
  known, allowed divergence).
- `bash tests/test-no-loop-references.sh` → `PASS: no live references to
  the retired Ralph Loop execution system outside historical documents`.
- No `.claude/skills/` files were touched this session, so
  `scripts/sync-skills.sh` and `scripts/check-skill-sync.sh` were not run
  (nothing to regenerate or drift-check).

## Files changed

- `docs/plans/active/2026-08-03-org-debt-batch.md` (AC-6 correction note,
  progress checklist tick)
- `.claude/rules/model-routing.md` (one-sentence `ralph insights` consumer
  note in the "Org runtime model receipts" section)

## Known gaps

- None. All 6 requested checks were performed; 2 produced edits, 4 confirmed
  already in sync with no action needed.
