# Review: backfill-na reviewer (t-2)

- Date: 2026-10-25
- Commit: 5babe70226ddbd159bf6bcd3f091638e131c8ef5
- Plan: docs/plans/active/2026-10-25-backfill-reviewer-status.md
- GATE: pass
- Merge: OK (CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 2)

## Deterministic gates

- `./scripts/run-static-verify.sh`: exit 0, "All verifiers passed". gofmt ok, go vet 0 issues, branch secret scan clean. staticcheck was skipped (command not found). Evidence: docs/evidence/verify-2026-10-25-023524.log
- `./scripts/run-test.sh`: exit 0, all packages ok (internal/insights 2.099s, internal/cli 342.843s, internal/org 428.606s). Evidence: docs/evidence/verify-2026-10-25-023622.log

## Acceptance criteria

| AC | Result | Evidence |
|----|--------|----------|
| `Reviewer status: incomplete...` gives `n/a` | met | internal/insights/backfill.go:159-165, test `TestParseReport_CrossReview_IncompleteReviewerIsNA`; ParseMiss stays false (non-goal honored) |
| `complete` with 0/0/0 stays `pass` | met | `TestParseReport_CrossReview_CompleteReviewerZeroCountsIsPass` |
| Only the later of two cycles is `n/a` | met | `TestParseReport_CrossReview_OnlyLaterCycleIncomplete` (also covers case-insensitive `Incomplete`) |
| Tests use `t.TempDir()` | met | `writeCrossReviewReport` in backfill_test.go |

Scope: the commit touches only internal/insights/backfill.go and backfill_test.go. docs/tech-debt/README.md is untouched. The comment on `parseCrossReviewAllCycles` documents the status line (verify-plan drift check).

## Findings

### LOW 1: status regex is not anchored to a list item
- Evidence: internal/insights/backfill.go (`reviewerStatusRe`, `(?i)Reviewer status:\s*(\S*)`)
- The pattern matches anywhere in a line. A prose line such as "... Reviewer status: incomplete (<reason>) ..." that sits between a cycle's status line and its triage line would override the real status. The one such prose line in the repo (docs/reports/cross-review-triage-codex-exec-stdin-and-model.md:24) comes after the last triage line, so no current report is affected.

### LOW 2: a markdown-decorated value is read as not incomplete
- Evidence: same regex; `\S*` captures `**incomplete**` and `HasPrefix(..., "incomplete")` is false.
- All 67 existing `Reviewer status:` lines read `complete`, so no current report is affected. A future decorated value would fall back to the counts-based verdict (`pass`), the pre-fix behavior.

## Notes

- The status line is consumed with `continue`, so a line holding both a status and an `After triage:` is not parsed. Real reports keep them on separate lines.
- The reset after each triage line matches the plan's Risk mitigation (two-cycle test).
