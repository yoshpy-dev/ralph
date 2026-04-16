# Codex triage report: ralph-tui integration

- Date: 2026-04-16
- Plan: docs/plans/active/2026-04-15-ralph-tui/_manifest.md
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes (docs/reports/self-review-2026-04-16-integration-ralph-tui.md)
- Total Codex findings: 5
- After triage: ACTION_REQUIRED=4, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-04-15-ralph-tui/
- Self-review report: docs/reports/self-review-2026-04-16-integration-ralph-tui.md
- Verify report: N/A (run-verify.sh used directly)
- Implementation context: Sub-model wiring was just added in previous commit (8533385). Codex reviewed the wired state and found gaps in the wiring.

## ACTION_REQUIRED

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 1 | SliceSelectedMsg not forwarded to ActionsModel | Real bug: actions pane never knows which slice is selected, making r/a/L/e unusable | cmd/ralph-tui/main.go |
| 2 | No Tailer created; log pane has no log source | Real bug: watcher.Watch() emits StateChangedMsg, not LogLineMsg. Tailer needed for log streaming | cmd/ralph-tui/main.go |
| 3 | sliceNameRe truncates to `slice-N`, mismatching real names like `1-ralph-tui` | Real bug: dependency graph nodes never match SliceState.Name, deps pane shows unknown statuses | internal/state/reader.go |
| 4 | LogPath and WorktreePath never populated in ReadFullStatus | Real bug: HasLogs() and HasWorktree() always false, L/e actions permanently disabled | internal/state/reader.go |

## WORTH_CONSIDERING

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 5 | Retry doesn't reset `.started` timestamp, causing timeout carry-over | Valid edge case: a retried slice could inherit old start time and timeout immediately | scripts/ralph |

## DISMISSED

(none)
