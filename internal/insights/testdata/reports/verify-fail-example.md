# Verify report: some-failing-task

- Date: 2026-07-11
- Plan: docs/plans/active/2026-07-11-some-failing-task.md

## Verdict: FAIL

One or more acceptance criteria not met.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|-----------|--------|----------|
| AC1 | gofmt clean | PASS | exit 0 |
| AC2 | go vet clean | FAIL | 2 issues found |
