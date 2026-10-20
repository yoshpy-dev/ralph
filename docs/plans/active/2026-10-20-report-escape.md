# Escape `|` in org report event details

- Status: Draft
- Approved: TBD
- Owner: Claude Code (org leader report-escape)
- Date: 2026-10-20
- Related request: split plan trial (feature report-escape, digest 0ffd32661606)
- Related issue: tech-debt "Small leftovers of org-plan-approval" (a)
- Type: fix
- Branch: fix/report-escape

## Objective

`ralph org report` writes each event's Details verbatim into a Markdown table
row (`internal/org/report.go`, the `| ts | seat_id | event | details |` table).
A `|` inside Details breaks the row's column count. Escape `|` as `\|` so each
row keeps exactly 4 columns.

## Scope

- `internal/org/report.go`: escape `|` in Details when writing the event table row.
- `internal/org/report_test.go`: test for a Details value containing `|`.

## Non-goals

- Escaping other columns (ts, seat_id, event) or other Markdown characters.
- Changing any other section of the report.

## Assumptions

- Details are single-line strings; newline handling is out of scope.

## Affected areas

- `internal/org/report.go`, `internal/org/report_test.go` only (reserved paths).

## Visual review

- `.harness/state/plan-visual/report-escape.html` (lint and shot results in the approval QUESTION).

## Design decisions

Critical forks: None

## Acceptance criteria

- [ ] A Details value containing `|` is escaped and its row keeps 4 columns
- [ ] A test in `internal/org/report_test.go` checks that row
- [ ] Rows whose Details contain no `|` are unchanged
- [ ] `go test ./internal/org/` passes (also with `GOTOOLCHAIN=go1.25.8`)

## Implementation outline

1. Replace `|` with `\|` in `details` before `fmt.Fprintf` of the event row.
2. Add a table-driven or focused test asserting the escaped row and an unchanged plain row.

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` (via `ralph org gate`)
- Spec compliance criteria to confirm: the 4 acceptance criteria above
- Documentation drift to check: none expected
- Evidence to capture: reviewer report under `docs/reports/`

## Test plan

- Unit tests: new test in `internal/org/report_test.go`
- Integration tests: none
- Regression tests: existing report tests unchanged
- Edge cases: Details with multiple `|`; Details with no `|`; empty Details (`-`)
- Evidence to capture: `go test ./internal/org/` output, also with `GOTOOLCHAIN=go1.25.8`

## Risks and mitigations

- An already-escaped `\|` would become `\\|`; accepted, Details are not pre-escaped today.

## Rollout or rollback notes

Revert the single commit.

## Open questions

None.

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [ ] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
