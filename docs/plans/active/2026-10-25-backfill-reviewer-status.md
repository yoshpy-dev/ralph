# backfill: read the cross-review Reviewer status

- Status: Approved
- Approved: 2026-10-25 sha256:1d3a60a7cd8b
- Owner: Claude Code (org backfill-na leader)
- Date: 2026-10-25
- Related request: split plan recheck, feature backfill-na (/private/tmp/ralph-iso/ralph/.harness/state/org/splits/recheck.md, digest 4f955a6f4269)
- Related issue: docs/tech-debt/README.md row on `parseCrossReviewAllCycles` / `crossReviewVerdict`
- Type: fix
- Branch: fix/backfill-na

## Objective

`ralph insights` backfill derives a cross-review cycle's verdict only from the
`After triage: …` line. A report whose reviewer did not finish
(`Reviewer status: incomplete (<reason>)`) still carries
`After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0`, so
`crossReviewVerdict` returns `pass`. Make backfill read the
`Reviewer status:` line of each cycle and give an incomplete cycle the verdict
`n/a`, the same as an unreadable report.

## Scope

- `internal/insights/backfill.go`: `parseCrossReviewAllCycles` records, per
  cycle, whether the most recent `Reviewer status:` line before that cycle's
  `After triage:` line starts with `incomplete`; the cross_review case sets
  `Verdict = "n/a"` for such a cycle.
- `internal/insights/backfill_test.go`: tests for the three cases below.

## Non-goals

- Editing `docs/tech-debt/README.md` (the recheck record PR marks the row RESOLVED).
- Changing other phases' verdicts, the dedupe key, or the event schema.
- Setting `ParseMiss` for an incomplete cycle (the report is readable; only
  the verdict is not known).

## Assumptions

- In real reports the `- Reviewer status:` line comes before the cycle's
  `- After triage:` line in the same header block (see
  `docs/reports/cross-review-triage-check-template-scaffold-green.md` lines 9/13 and 52/56).
- A status line belongs to the next `After triage:` line; it is reset after
  each triage line, like the pending `## Cycle N` heading.

## Affected areas

- internal/insights/backfill.go
- internal/insights/backfill_test.go

## Visual review

.harness/state/plan-visual/backfill-reviewer-status.html (lint and shot results are in the approval QUESTION).

## Design decisions

Critical forks: None. Matching is case-insensitive on `Reviewer status:\s*incomplete`; any other value (`complete`, missing line) keeps the counts-based verdict.

## Acceptance criteria

- [ ] A cycle whose `Reviewer status:` starts with `incomplete` gets verdict `n/a`, not `pass` (same as an unreadable report)
- [ ] A cycle with `Reviewer status: complete` and 0/0/0 keeps verdict `pass`
- [ ] In a report with two cycles where only the later cycle is incomplete, only that cycle is `n/a`
- [ ] The three cases above are checked by tests (files written under `t.TempDir()`)

## Implementation outline

1. Add an `incomplete bool` to `crossReviewCycleEntry`; in the scanner, match `(?i)Reviewer status:\s*incomplete` (and any `Reviewer status:` line to reset), attach it to the next triage entry, reset after the triage line.
2. In the `cross_review` case, set `ev.Verdict = "n/a"` when `entry.incomplete`, else `crossReviewVerdict(entry.triage)`.
3. Add table or individual tests in `backfill_test.go` for the three ACs.

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` (gofmt, go vet, lint)
- Spec compliance criteria to confirm: the four ACs
- Documentation drift to check: comments on `parseCrossReviewAllCycles` mention the status line
- Evidence to capture: reviewer report under docs/reports/

## Test plan

- Unit tests: `go test ./internal/insights/...`
- Integration tests: none
- Regression tests: existing backfill tests stay green
- Edge cases: two cycles, later one incomplete; complete with 0/0/0
- Evidence to capture: `./scripts/run-test.sh` output in the reviewer report

## Risks and mitigations

- A status line in a later cycle could leak into an earlier one: reset after each triage line and test the two-cycle case.

## Rollout or rollback notes

Revert the commit; existing committed insight events are not rewritten.

## Open questions

None.

## 全機能の座席に適用する決まり

```
## 座席の決まり

- 座席はどれも claude で動かす。implementer は `--model opus`、reviewer は `--model sonnet` で spawn する
- 予約(`- Reserve:`)の外のファイルは編集しない
- `docs/tech-debt/README.md` は編集しない。直した行の RESOLVED は、この確かめの記録の PR が書く
- テストは本物の herdr、agmsg、gh、osascript、network を叩かない
```

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [ ] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
