# Codex triage report: Ralph Pipeline Hardening

- Date: 2026-04-15
- Plan: feat/ralph-pipeline-hardening
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Codex model: gpt-5.4 (v0.120.0)
- Rounds: 4
- Total findings: 8 (ACTION_REQUIRED: 4, WORTH_CONSIDERING: 4, DISMISSED: 0)

## Round 1 — Total findings: 3, After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=1

### ACTION_REQUIRED (fixed)

| # | Codex finding | Fix |
|---|---------------|-----|
| 1 | [P1] INT/TERM trap does not call `exit` — orchestrator continues after cleanup | Added `_on_signal()` with `_INTERRUPTED` flag and `exit 1` |
| 2 | [P1] EXIT trap overwrites "partial" status with "interrupted" on non-zero exit | EXIT trap now guards with `_INTERRUPTED == 1` |

### WORTH_CONSIDERING (upgraded to ACTION_REQUIRED in Round 2)

| # | Codex finding | Triage rationale |
|---|---------------|------------------|
| 3 | [P2] Missing `gh` returns exit code 1, misinterpreted as "codex ACTION_REQUIRED" | Exit code collision. Upgraded after Round 2 confirmed orchestrator-side impact. |

## Round 2 — Total findings: 2, After triage: ACTION_REQUIRED=2

### ACTION_REQUIRED (fixed)

| # | Codex finding | Fix |
|---|---------------|-----|
| 4 | [P1] `gh_unavailable` not in orchestrator terminal status patterns | Added `gh_unavailable` to skip-launch, failure-count, and status display patterns |
| 5 | [P2] `run_outer_loop` exit code collision for `gh_unavailable` and ACTION_REQUIRED | Changed to `return 2` with new case handler: `_finalize "gh_unavailable"` |

## Round 3 — Total findings: 2, After triage: WORTH_CONSIDERING=2

### WORTH_CONSIDERING (fixed)

| # | Codex finding | Fix |
|---|---------------|-----|
| 6 | [P1] `_CHILD_PIDS` never pruned — PID reuse kill risk on long sessions | Removed `_CHILD_PIDS` loop; `cleanup_on_exit` uses `.pid` files exclusively |
| 7 | [P3] `timeout` status missing from status display patterns | Added `timeout` to `resolve_display_phase`, `status_icon`, and detail column |

## Round 4 — Total findings: 1, After triage: WORTH_CONSIDERING=1

### WORTH_CONSIDERING (fixed)

| # | Codex finding | Fix |
|---|---------------|-----|
| 8 | [P1] Timeout kill deletes `.pid` file before process exits — cleanup can't re-kill | Kept `.pid` file after timeout kill; EXIT trap handles cleanup |

## DISMISSED

(none across all 4 rounds)
