# sync-docs report: org-stop-reason-removal

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-17-org-stop-reason-removal.md` (issue #153)
- Prior reports: `docs/reports/self-review-2026-09-18-org-stop-reason-removal.md`
  (Merge, no open findings), `docs/reports/verify-2026-09-18-org-stop-reason-removal.md`
  (Pass), `docs/reports/test-2026-09-18-org-stop-reason-removal.md` (PASS)

## Summary

This is a small, self-contained refactor (removes the producer-less
`StopParams.Reason` field and its `Stop` Details append; rewrites one doc
comment in `internal/org/watch.go` with no code change; reshapes three tests
and adds one). The plan's own Non-goals and Verify plan already asserted "no
docs/specs/recipes/rules update needed, confirmed by grep", and `/verify`
independently re-confirmed that claim (AC-7, Documentation drift section).
This pass re-ran the same and additional greps directly against the worktree
to confirm no doc surface was missed, and found none. **No files were
changed in this pass.**

## Files/surfaces checked (no changes needed — already correct)

- `docs/specs/2026-08-01-org-runtime.md` — describes the watchdog's two-layer
  design and the deadman clause at the requirements level (FR-8, AC,
  Background) but never names `StopParams`, `Stop`'s `reason=` Details
  suffix, or `leadActivityEventCount`. `grep -n 'StopParams\|reason=watchdog\|leadActivityEventCount'`
  against the spec is empty, so per this task's own instruction (add a
  revision note only if the spec already documented the removed seam) none
  is added. The spec's existing budget-removal revision markers (e.g. the
  inline "(2026-09-16 改訂で撤去)" annotations on FR-8) are the file's
  convention for genuinely-affected requirements; this change does not
  qualify.
- `.claude/rules/ralph/agent-messaging.md` — documents the `STOP` message
  *type* in the agmsg typed protocol, an unrelated concept (a message kind,
  not the Go `StopParams`/`Stop` verb or its Details string). No mention of
  `StopParams`, `Reason`, or `reason=watchdog_`.
- `.claude/skills/org/SKILL.md` and its three mirrors
  (`.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md`) — the verbs table's `stop`
  row reads only "座席を停止。" / `ralph org stop --org-id X --seat reviewer-1`;
  it never documented a `--reason` flag (none exists), so there is nothing
  to retire. No watchdog/deadman description in this skill references the
  removed field. `git diff c6b8127...HEAD -- .claude/skills/org/SKILL.md
  .agents/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md
  templates/base/.agents/skills/org/SKILL.md` is empty, confirming no
  mirror drifted out of lock-step because of this change (no sync-skills
  regeneration needed).
- `docs/recipes/*.md`, `README.md`, `AGENTS.md`, `CLAUDE.md` — no mention of
  `ralph org stop`, `StopParams`, or the deadman/watchdog exclusion at any
  level of detail. `grep -rn 'org stop\|StopParams'` against `README.md`/`AGENTS.md`/`CLAUDE.md`
  is empty.
- `docs/tech-debt/README.md` — the `StopParams.Reason` row (line 129) is
  correctly closed by the implementation slice: struck, `(RESOLVED 2026-09-18
  in refactor/org-stop-reason-removal)` in the debt-item cell, 6 pipes intact,
  Related cell carries the archive-plan and self-review-report pointers.
  Read the full file (all 14 rows) to confirm no *other* row still describes
  `StopParams.Reason` or "the reader-side exclusion" as open — the only other
  rows mentioning `leadActivityEventCount`/watchdog-cutoff-stop (lines
  70/79/83) are unrelated, already-`RESOLVED` historical rows about
  `org_id`/seat scoping and probe-outage handling from earlier PRs; none of
  them claims the exclusion is dormant or unreachable, so none is stale
  because of this change.
- Plan's own Progress checklist (`docs/plans/active/2026-09-17-org-stop-reason-removal.md:120-128`) —
  already accurate against reality: everything through "Test artifact
  created" is checked, and "PR created" is correctly unchecked (no PR exists
  yet). No edit needed.
- Plan's Deviation notes — already carries the note this task would
  otherwise have added: the last entry ("2026-09-18 plan drift") records
  that the shipped `watch.go` comment wording supersedes the plan's original
  Scope item 2 text, per self-review M1. No further note needed.

## Verification of "no stale reference" claims

Re-ran the sweeps independently in this worktree (not just trusting the
prior reports' citations):

- `grep -rn 'StopParams\|reason=watchdog\|leadActivityEventCount' docs/specs
  docs/recipes README.md AGENTS.md CLAUDE.md .claude/rules .claude/skills
  .agents/skills templates/base` → empty.
- `grep -rn 'WatchdogsOwnStopEvent\|_WatchdogStopDoesNot\b' --include='*.md'
  .` → only the plan's own scope table (historical record of the rename,
  expected to remain) and the self-review/verify/test reports (historical,
  expected to remain). Nothing in a live/current-state doc.
- `grep -n 'stop\|watchdog\|Reason' .claude/skills/org/SKILL.md` → only the
  `stop` verb row and two "stop座席→disband" workflow steps, none mentioning
  `Reason` or `--reason`.

## Files changed in this pass

None. No documentation, skill, spec, recipe, or tech-debt edit was needed —
the implementation slice's own Slice B (`docs/tech-debt/README.md`) and the
plan/report artifacts already committed cover every doc surface this
change touches. Sync scripts were not run since no mirrored file was
touched.

## Known gaps

- Did not re-verify `./scripts/run-verify.sh`/tests; that is `/verify`'s and
  `/test`'s completed job (both reports cited above already Pass/PASS).
- Did not inspect every file in the repo for the string "Reason" (e.g.
  `internal/org/watch.go`'s unrelated escalation-record `Reason` field) —
  that grep-precision question is `/verify`'s AC-1 scope, already answered
  there (Pass, three unrelated hits, no `StopParams` residue).
