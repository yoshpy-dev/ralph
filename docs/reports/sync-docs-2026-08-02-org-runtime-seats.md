# Sync-docs report: org-runtime-seats

- Date: 2026-08-02
- Plan: `docs/plans/active/2026-08-02-org-runtime-seats.md`
- Spec: `docs/specs/2026-08-01-org-runtime.md` (PR② 座席の常駐セッション化)
- Agent: `doc-maintainer` subagent (Claude Code, `/sync-docs`)
- Commit under review: `a550e17` (docs: add test report for org-runtime-seats)
- Base: `9abaaed` (main)

## Summary

PR② (座席の常駐セッション化) rewrote the agmsg adapter to the verified real
script interface, integrated `join`/`leave` into the spawn/stop saga, added
role-prompt templates and a typed-message protocol validator, closed the
Verbs 0% coverage gap, and ran a live herdr+agmsg smoke that surfaced and
fixed 4 real-CLI integration bugs (documented in the plan's "Implementation
notes (deviations)"). Self-review, verify, and test reports all confirm
AC-1..AC-11 met and no CRITICAL/blocking findings. This sync-docs pass closes
the three drift items flagged by the pipeline reports and confirms the rest
of the doc surface needs no change for this PR.

## Drift check results

### Plan AC-8 evidence-path wording (`.log` vs `.txt`) — stale, corrected

**Drift detected. Resolved.**

The plan's AC-8 text specified `docs/evidence/org-seats-smoke-*.log`, but
`docs/evidence/*.log` is gitignored (`.gitignore:55`), so a `.log`-named file
could never be the committed durable evidence. The actual committed artifact
is `docs/evidence/org-seats-smoke-2026-08-02.txt` (a stray untracked `.log`
duplicate exists in the working tree from the original capture — expected,
per the verify report's note). The verify report flagged this as a "minor
wording drift, not a functional gap" and asked `/sync-docs` to reconcile it.
Corrected the AC-8 wording in the plan to `*.txt` and added a one-line
parenthetical explaining the gitignore reason, so a future reader doesn't
reintroduce the `.log` extension.

### Plan Progress checklist — stale (artifacts existed but weren't ticked)

**Drift detected. Resolved.**

"Review artifact created" / "Verification artifact created" / "Test artifact
created" were unchecked even though all three reports already existed
(`docs/reports/self-review-2026-08-02-org-runtime-seats.md`,
`verify-2026-08-02-org-runtime-seats.md`, `test-2026-08-02-org-runtime-seats.md`).
The verify report flagged this as a known, non-blocking gap ("plan AC/checklist
lags implementation" pattern) and explicitly named it as `/sync-docs`'s job.
Ticked all three with pointers to the report paths.

### Plan Acceptance criteria — stale (all 11 ACs unchecked despite PASS verdicts)

**Drift detected. Resolved.**

None of AC-1..AC-11 were checked off in the plan even though the verify
report's Verdict is PASS on all 11 (5 fully verified behaviorally, 6 with
spec-shape + static analysis confirmed and behavioral proof delegated to
`/test`, which then confirmed all of them PASS). Checked all 11 boxes with a
short evidence pointer per AC (verify report AC row + test report AC→test
mapping, cross-referenced against `docs/tech-debt/README.md`'s RESOLVED rows
for AC-6). AC-5's wording was also updated in place to name the actual
mechanism (`Leave`/`leave.sh`) instead of the plan's original `despawn.sh`
assumption, which the plan's own "Implementation notes (deviations)" section
already documents as superseded by a live-smoke discovery — the AC text had
not been updated to match.

### docs/tech-debt/README.md — new entry: 4 deferred LOW self-review findings

**Gap identified by /verify and /test. Recorded.**

The self-review report's "Recommendation" section named 8 LOW follow-ups as
"safe to batch or defer" alongside the 5 MEDIUM findings that were fixed in
commit `1520b96`. The verify report's fix-verification table confirms 2 of
those 8 LOWs were also fixed in that same commit (`agmsgTeam` shadowing,
retry-count/off-by-one) and explicitly lists the remaining 4 as "Not fixed,
not newly tracked as tech debt" — then flags this as a gap in its own
"Documentation drift" and "Coverage gaps" sections, asking `/sync-docs` to
record them before they silently age out. The test report's Test gaps section
confirms these are pre-existing scope decisions, not test gaps.

Added **one consolidated tech-debt row** (per task instructions, not 4
separate rows) covering:
- (a) `"lead"` identity as a bare string literal + `agmsgTypeForDriver("claude")`
  hardcoded for the lead's agmsg type (`internal/org/spawn.go:378,404`)
- (b) `ralph doctor`'s `checkAgmsgAvailable` not surfacing the resolved
  `agmsg_home` in its error detail (`internal/cli/doctor.go:625-631`,
  `internal/org/driver/driver.go:82-86`)
- (c) `dryRunSpawn` silently swallowing `RenderRolePrompt`/`promptFilePath`
  errors instead of failing the dry-run trail (`internal/org/spawn.go:610-624`)
- (d) no redaction convention for `docs/evidence/` committed smoke logs
  (`docs/evidence/org-seats-smoke-2026-08-02.txt` embeds the developer's home
  path and a session UUID)

Trigger to pay down: PR③ (Lead 自律編成) makes (a) and (c) load-bearing
(multi-identity leads, LLM-driven dry-run pre-flight); (b)/(d) pay down
opportunistically or on first operator complaint. Referenced both the
self-review report (original findings) and the verify report (drift
surfacing) per the task instructions.

## Confirmed clean (no change needed)

- **`AGENTS.md` repo map** — the `internal/org/` line already reads "herdr/agmsg
  driver adapters, role prompt templates (go:embed), typed message protocol"
  (added in PR①'s sync-docs pass); this PR's changes (script-shape rewrite,
  join/leave integration, protocol validator, role templates) are all inside
  that same one-line description — no new noun needs surfacing at map
  granularity.
- **`CLAUDE.md`** — no skill/command/pipeline behavior changed; nothing to
  update.
- **`README.md`** — org runtime is still pre-`/org`-skill (no `ralph org
  start`, no Lead autonomy); the plan's own Non-goals defer "旧系撤去・ドキュ
  メント全面改稿" to PR⑤. Grepped for `internal/org`/`agmsg`/`herdr` — no
  hits, consistent with README not yet documenting org runtime at all (by
  design, per the spec's phased rollout).
- **`.claude/rules/agent-messaging.md` ↔ `templates/base/.claude/rules/agent-messaging.md`
  mirror** — already synced by the implementation itself (self-review and
  verify reports both confirm byte-identical via `cmp`); re-confirmed here:
  `cmp` → identical.
- **`docs/specs/2026-08-01-org-runtime.md`** — the verify report already
  confirmed the PR② boundary note (scope-outside-write detection deferred to
  PR④'s Watchdog pulse layer) is in sync with the plan's Non-goals; no
  further edit needed.
- **`docs/quality/definition-of-done.md`** — no pipeline-order or quality-gate
  change in this PR.

## Harness-internal sync

No skills, hooks, scripts, or language packs changed in this PR (confirmed by
`git diff ffda48f...HEAD --stat`: only `internal/org/`, `internal/cli/org.go`,
`internal/cli/doctor.go`, `internal/config/`, `.claude/rules/agent-messaging.md`
+ mirror, `docs/`, `scripts/ralph-config.sh` + mirror). Ran the two mechanical
gates anyway as a final check:

- `./scripts/check-sync.sh` → `DRIFTED: 0`, `PASS: all files in sync.`
- `./scripts/check-skill-sync.sh` → `13 skill(s) in lock-step`

## Verification

- `git status` in the worktree — clean before this pass began.
- Plan edits: AC-8 wording, all 11 AC checkboxes + evidence, 3 Progress
  checklist rows — verified by re-reading the edited sections after writing.
- Tech-debt edit: verified the new row sits directly above the
  `Verbs.Send/Wait/Read` RESOLVED row (chronological insertion point) and
  does not disturb any existing row's table-cell structure.
- `cmp .claude/rules/agent-messaging.md templates/base/.claude/rules/agent-messaging.md` → identical.
- `./scripts/check-sync.sh` and `./scripts/check-skill-sync.sh` → both green (see above).

## Outcome

Documentation is in sync with implementation. No CRITICAL or blocking drift
remains. Ready to proceed to `/cross-review`.
