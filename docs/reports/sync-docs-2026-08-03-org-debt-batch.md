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

## Cycle 2 (residual sweep)

- Date: 2026-08-03
- Worktree: `.claude/worktrees/org-debt-batch`, branch `chore/org-debt-batch`, HEAD `8cdd5da`
- Trigger: residual sweep after the cycle-2 fix-and-revalidate pass (`5147f2a`..`8cdd5da`) —
  doc/comment-only delta: `docs/tech-debt/README.md` wording fixes (AR-1 doctor-overclaim
  correction in `53145ea`, register-cell correction + new 7-item batched deferred-LOW row in
  `2b71fd1`/`bc058a1`), one `internal/org/watch.go` comment edit (stale self-reference removed,
  `2b71fd1`), plan progress-checklist ticks (`2b71fd1`), and the cycle-2 self-review/verify/test
  report sections themselves.
- Scope: confirm the cycle-2 doc/comment fixes did not introduce new drift — tech-debt README
  internal consistency (citations still point at real code) and plan-file coherence (checklist
  state matches actual artifacts on disk).

### Checks performed

1. **`docs/tech-debt/README.md` doctor-overclaim fix (`53145ea`)** — verified against
   `internal/cli/doctor.go`: the corrected wording ("`doctor` was left unchanged — it has no org
   state-dir diagnostic line to annotate") matches current source; `checkOrgEnvelope` (doctor.go:534)
   reports `[org]` envelope config and herdr/agmsg/model-probe availability, not state-dir
   resolution. No stale claim remains.
2. **Register-cell correction (`2b71fd1`)** — the citation now reads `internal/cli/org.go`
   (`newOrgRuntimeAt`) / `internal/cli/status.go` (`runStatus`); both functions confirmed present
   at those names (`grep -n 'func newOrgRuntimeAt\|func runStatus'`). Matches Slice 1's actual
   AC-1 refactor.
3. **New 7-item batched deferred-LOW row (`2b71fd1` + `bc058a1`)** — cross-checked against the
   verify cycle-2 report's own C2-2 crosscheck (`docs/reports/verify-2026-08-03-org-debt-batch.md`
   §"Documentation drift (Cycle 2 delta)"), which had flagged item 7 (`xreview-helpers.sh` header
   over-rewording) as omitted; confirmed `bc058a1` added it. All 7 items now present and the gap
   verify flagged is closed.
4. **`internal/org/watch.go` comment edit (`2b71fd1`)** — removed a self-referential line-number
   citation (`watch.go:838`) that pointed at no function of that name in the current file (no
   `raiseOrClear` symbol exists); the removal eliminates a stale pointer rather than introducing
   one. No other doc or comment references the removed citation.
5. **Plan progress checklist (`2b71fd1`)** — `docs/plans/active/2026-08-03-org-debt-batch.md`'s
   "Self-review artifact created" / "Verify artifact created" boxes are now `[x]`, matching the
   actual report files on disk (`docs/reports/self-review-2026-08-03-org-debt-batch.md`,
   `docs/reports/verify-2026-08-03-org-debt-batch.md`, both present with Cycle 2 sections).
6. **Insight-event log consistency (`docs/insights/events/2026-08-03-org-debt-batch.jsonl`)** —
   found and fixed one genuine mislabel: the `verify` event appended by `cee1579` ("record verify
   cycle-2 insight event") recorded `"cycle":1`, but the verify report it documents is explicitly
   titled "Cycle 2 — delta re-verify" and covers the `4a5744e..2b71fd1` delta. Corrected to
   `"cycle":2` so the event log matches the report it represents and so `ralph insights`
   aggregation doesn't undercount the fix-and-revalidate cycle for the `verify` phase (the
   `self_review` and `test` phases already had matching cycle-1/cycle-2 pairs in this file).

### Residual drift found (pre-existing, not introduced by cycle-2 fixes)

- `docs/tech-debt/README.md`'s `Cutoff`-ratchet row cites `watch.go:635-637` and `watch.go:654`
  for its supporting comment/early-return; current line numbers are `~634-644` and `662`
  respectively (function `evaluateTotalBudget` still contains both, just offset by a handful of
  lines). This imprecision predates the cycle-2 `watch.go` comment edit — the one-line removal in
  `2b71fd1` shifted the offset by only 1 line, not the ~8-line gap observed. Non-blocking
  (approximate line pointers are the row's existing convention elsewhere in this file); left
  as-is rather than opening a new fix cycle over a citation still resolvable by function name.

### Cycle 2 verdict

No new documentation drift from the cycle-2 doc/comment fixes. One pre-existing data-integrity
issue found and corrected (insight-event cycle mislabel, item 6 above); one pre-existing citation
imprecision noted but left unfixed (non-blocking, out of the requested scope). Plan progress
checklist and tech-debt README are internally coherent with current source as of `8cdd5da`.
