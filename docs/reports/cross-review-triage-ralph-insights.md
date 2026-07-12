# Cross-review triage — ralph-insights

- Date: 2026-07-13
- Reviewer: codex (cross-model second opinion)
- Driver: claude
- Diff: feat/ralph-insights vs origin/develop
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=1, DISMISSED=0

## ACTION_REQUIRED

| # | Severity | Finding | Triage rationale | Fix |
|---|----------|---------|------------------|-----|
| 1 | HIGH | Backfill collapses multi-cycle reports into one cycle-1 event; `parseCrossReviewTriage` takes the FIRST "After triage" line, so a report with cycle-1 AR=2 / cycle-2 AR=0 records the wrong (non-final) result and loses escalation history. Verified: `internal/insights/backfill.go:152-155` hardcodes cycle=1 with a comment claiming multi-cycle reports do not exist, while `docs/reports/cross-review-triage-loop-model-routing.md:12,63` has two triage lines. Violates plan AC5 / Design decision "Backfill dedupe key includes the source and cycle". | The plan folded Codex plan-advisory MEDIUM#3 precisely to prevent this; the implementation missed body-level cycle sections. | Parse all "After triage" occurrences (and cycle/addendum section markers) → one event per cycle; regression test uses a real-shaped single report with two cycles. |
| 2 | HIGH | Appender emits `cycle: null` when `--cycle` omitted; the skill snippets omit it, so all standard-flow events have null cycle (committed example: `docs/insights/events/2026-07-12-ralph-insights.jsonl`). Go reader maps null→0, excluding these events from escalation analysis and contradicting README field expectations. | Schema-consistency bug introduced by the skill wiring; cheap deterministic fix at the appender level beats trusting LLM-written snippets. | `insights-append.sh` defaults `--cycle` to 1 when omitted; README documents cycle default and marks routing fields optional for `source:skill|backfill`. |
| 3 | MEDIUM | Backfill dedupe key uses `source_report_path` verbatim; relative first run + absolute second run duplicates every event. Verified: `backfill.go` stores the path as passed while the CLI default is relative. | Data-corruption class — exactly what cross-review was asked to catch. | Normalize with `filepath.Abs` before storing/deduping; regression test covers rel-then-abs double apply. |

## WORTH_CONSIDERING

| # | Severity | Finding | Triage rationale | Decision |
|---|----------|---------|------------------|----------|
| 4 | LOW | `ralph insights --json` prints the human "no data yet" text in the zero-data case instead of JSON (AC3 wants valid JSON from --json). | Real but minor; fix is one branch reorder. | Fix along with #1-#3 in the same cycle. |

## DISMISSED

None.

## Consequence

Per `.claude/rules/post-implementation-pipeline.md`, ACTION_REQUIRED triggers a
fix followed by a full pipeline re-run (self-review → verify → test →
sync-docs → cross-review). This is pipeline cycle 2 of the default cap 2.
