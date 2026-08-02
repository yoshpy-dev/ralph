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

# Cycle 2 (fix-and-revalidate re-run)

- Date: 2026-08-02
- Agent: `doc-maintainer` subagent (Claude Code, `/sync-docs`), pipeline cycle 2 of 2
- Delta reviewed: `f0cbf11` (cross-review ACTION_REQUIRED #1 fix — reserve `_`
  as the org/seat namespace separator, tighten `identifierPattern` to
  `^[a-z][a-z0-9-]{0,29}$`, add the `len(org)+1+len(seat) ≤ 32` combined-length
  guard) and `011a0a7` (cycle-2 self-review LOW fixes: `failStepWithNote` doc
  comment now names both callers, the 30-vs-32 length rationale moved from a
  test comment into `identifierPattern`'s own doc comment, `qa.md`'s
  branch-specific `EVIDENCE` example replaced with a synthetic placeholder,
  `{{PLAN_PATH}}` restored to the prompt replacer), plus the cycle-2
  self-review/verify/test report addenda (`63c7a62`, `d0e0a55`, `35af821`).
- Scope: confirm no drift remains after the charset/separator change and the
  comment-only follow-up fixes; nothing in this delta changes CLI surface,
  skills, hooks, or scripts.

## Drift check results (cycle 2)

### Item 1 — old identifier charset / hyphen-join wording in shipped docs

**Checked. No drift found; already closed by the code-fix commits.**

`.claude/rules/agent-messaging.md` never documented the identifier charset or
join rule in the first place (it defines the message *protocol*, not
org/seat id shape), so there was nothing to correct there. The role-prompt
templates (`internal/org/prompts/reviewer.md`, `qa.md`) contain no id-shape
examples or hyphen-join illustrations — grepped both for
`0,63`/`A-Za-z0-9_-`/hyphen-join patterns, no hits. The only place the old
charset (`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`) and the old `-` join still
appear is inside historical narration that is correctly framed as
historical: the plan's "Implementation notes (deviations)" cross-review
bullet ("ハイフン連結は...一意でない"), the cross-review triage report, and
the self-review/verify/test report prose describing what the fix changed
*from*. None of these state the old shape as current behavior. Re-confirmed
mirror byte-identity: `cmp .claude/rules/agent-messaging.md
templates/base/.claude/rules/agent-messaging.md` → identical.

### Item 2 — plan deviations section / Open questions

**Checked. No drift found.**

The plan's "Implementation notes (deviations)" already carries the cycle-2
cross-review fix bullet ("cross-review 起因の修正(cycle 2)"), added by the
`f0cbf11` unit of work itself, and it accurately states the current charset,
the reserved `_` separator, and the combined-length check. "Open questions"
has no stale references to the old `-` join — its remaining entries (PR③
permission-mode follow-up) are unrelated to the identifier change. No edit
needed.

### Item 3 — tech-debt row consistency

**Checked. No drift found; rows are consistent, no duplicates.**

`docs/tech-debt/README.md` carries exactly one row for the cycle-2
self-review's "derived, not persisted" herdr-name finding (added by `63c7a62`
ahead of this sync-docs pass) and exactly one consolidated row for the four
deferred LOW findings (added in the cycle-1 sync-docs pass, cross-checked
against the self-review's cycle-2 regression table — all four are still
correctly marked "Deferred, now tracked" and none were silently re-opened or
duplicated by the cycle-2 fixes). The TOCTOU `max_seats` race row from PR①
is untouched by this delta and remains a single row. No contradictions
between rows.

### Cycle-2 self-review deferred findings (L4, L6/L7, INFO) — confirmed not silently dropped

The cycle-2 self-review explicitly batched/deferred three LOW-severity
findings not named in this task's known items (`lastAttempt` redundant
variable, role-neutral `defaultScopeText` wording, package-level verb
gating for PR③). The verify report's cycle-2 section already confirmed these
are consistent with the self-review's own "batch or defer" call and not
silent gaps requiring a tech-debt row (they are code-quality nits below the
tech-debt-worthy bar, not deferred correctness risk). No action taken —
consistent with the verify report's own conclusion.

## Confirmed clean (no change needed, cycle 2)

- `.claude/rules/agent-messaging.md` — no identifier-shape content, nothing
  to update; mirror re-confirmed identical.
- `internal/org/prompts/reviewer.md`, `qa.md` — no stale id-shape examples.
- `AGENTS.md`, `CLAUDE.md`, `README.md`, `docs/quality/definition-of-done.md`
  — unaffected by a Go-internal charset/separator change; cycle-1's
  "confirmed clean" verdicts still hold (delta touches no skill, hook,
  script, or CLI-surface file).
- `docs/tech-debt/README.md` — one row per finding, no duplicates, no
  contradictions (verified above).

## Harness-internal sync (cycle 2)

No skills, hooks, scripts, or language packs changed in the `f0cbf11`/
`011a0a7` delta (confirmed by `git diff a550e17..HEAD --stat`: only
`internal/org/`, `internal/cli/org_test.go`, and `docs/`). Ran the mechanical
gates anyway:

- `./scripts/check-sync.sh` → `DRIFTED: 0`, `PASS: all files in sync.`
- `./scripts/check-skill-sync.sh` → `13 skill(s) in lock-step`

## Verification (cycle 2)

- `git status` in the worktree — clean before this pass began; no doc edits
  were needed, so no working-tree changes from this pass beyond this report.
- Grepped for stale charset/hyphen-join wording across
  `.claude/rules/agent-messaging.md`, `internal/org/prompts/*.md`, and the
  plan body — no hits outside correctly-historical narration.
- Re-read `docs/tech-debt/README.md` rows 55–57 in full to confirm no
  duplicate or contradictory entries.
- `cmp .claude/rules/agent-messaging.md
  templates/base/.claude/rules/agent-messaging.md` → identical.
- `./scripts/check-sync.sh` and `./scripts/check-skill-sync.sh` → both green.

## Outcome (cycle 2)

No documentation drift found in the `f0cbf11`/`011a0a7` delta. All three
known items from the task are confirmed already closed — the separator/
charset fix and its cycle-2 self-review follow-ups carried their own doc
updates (tech-debt row, plan deviations bullet) in the same commits that
introduced the behavior change, and no shipped doc referenced the old shape
as current. Ready to proceed to `/cross-review`.
