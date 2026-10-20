# Review report: escape `|` in org report event details

- Date: 2026-10-20
- Plan: docs/plans/active/2026-10-20-report-escape.md
- Reviewer: reviewer seat (org report-escape)
- Scope: commit 3ea9ff3e (internal/org/report.go, internal/org/report_test.go)
- HEAD: 3ea9ff3ecdb434fe3eb23b99827c844cdb638289
- BASE: f0668004015323a2366ac7983a50e2d27017dfa2 (merge-base with origin/main)

## Gate

GATE: pass

- `./scripts/run-static-verify.sh`: exit 0. gofmt ok, 0 vet issues, branch secret scan clean. staticcheck was skipped (command not found). Evidence: docs/evidence/verify-2026-10-20-045606.log
- `./scripts/run-test.sh`: exit 0. `internal/org` ok (762s), `internal/cli` ok (541s), all other packages ok. Evidence: docs/evidence/verify-2026-10-20-045718.log

## Evidence reviewed

- `git show 3ea9ff3e` (report.go +2, report_test.go +18)
- report.go:168-181 (event table), :140-160 (roster table), :190-200 (receipts table)

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | test | The new test uses one `\|` in one Details value. The plan's edge cases list "multiple `\|`" and empty Details (`-`); neither is asserted. `ReplaceAll` covers multiple pipes by construction, and the empty case runs before the escape, so no defect results. | internal/org/report_test.go:95-111 | Optional: add a second pipe to the input and an empty-Details row. |
| LOW | plan | AC 4 asks for `go test ./internal/org/` also with `GOTOOLCHAIN=go1.25.8`. The gate ran only with the default toolchain. | plan AC 4 | leader runs the pinned-toolchain pass, or records it as a known gap. |
| LOW | scope | Roster and receipts tables write seat_id, role, driver and model unescaped. This is a non-goal in the plan (only Details), so it is not a defect here. | report.go:158, :197 | Follow-up candidate only. |

No CRITICAL or HIGH findings. Acceptance criteria 1-3 are met: the escape sits before the row `Fprintf`, the test asserts both the escaped 4-cell row and an unchanged plain row, and rows without `|` pass through the same code unchanged. The `\\|` double-escape risk is the one the plan accepts.

## Positive notes

- The change is two lines plus a comment that says why.
- The test asserts the full row string, so a wrong column count would fail it.

## Coverage gaps

- GOTOOLCHAIN=go1.25.8 run not performed (see LOW above).
- staticcheck not installed in this environment.

## Recommendation

- Merge: yes (no CRITICAL/HIGH)
- Follow-ups: pinned-toolchain test run; optional extra test cases.
