# Sync-docs report: org-runtime-mechanism

- Date: 2026-08-01
- Plan: `docs/plans/active/2026-08-01-org-runtime-mechanism.md`
- Spec: `docs/specs/2026-08-01-org-runtime.md` (PR① 機構層)
- Agent: `doc-maintainer` subagent (Claude Code, `/sync-docs`)
- Commit under review: `2e0364c` (docs: add test report for org-runtime-mechanism)
- Base: `9abaaed` (main)

## Summary

PR① (org runtime mechanism layer) is purely additive: `[org]` config envelope,
`ralph org` verb set (spawn/send/wait/read/stop/status/disband), `internal/org`
+ `internal/org/driver` packages, and `ralph doctor` org checks. Self-review,
verify, and test reports all confirm scope discipline (zero diff to
`scripts/ralph-orchestrator.sh`, `scripts/ralph-pipeline.sh`, `internal/ui/`,
`internal/state/`, `internal/action/`, `.claude/skills/`) and AC-1..AC-10
passing. This sync-docs pass closes three drift items flagged by the verify
and test reports and confirms the rest of the doc surface needs no change for
this PR.

## Drift check results

### docs/tech-debt/README.md — herdr-agent-namespace row (stale, pre-fix state)

**Drift detected. Resolved.**

The row added by self-review commit `d3e36a7` described the herdr
agent-naming bug as still open ("registers the herdr agent under the bare
`seat_id`", "`org wait` ... discards" `--org-id`). Commit `9bfe07e` (same PR,
later in the branch) fixed this: `herdrAgentName(orgID, seatID)` was added and
wired into `AgentStart` (`spawn.go:189`), `Send`'s and `Wait`'s `AgentWait`
calls (`verbs.go:83,138`), and `WaitParams` gained an `OrgID` field wired from
the CLI's `--org-id` (`internal/cli/org.go:239`). The verify report's
"Documentation drift" table flagged this row as stale and pointed to the
fix-verification table as evidence. Confirmed the fix by reading the
`9bfe07e` diff directly (not just trusting the reports) before editing.

Marked resolved using the file's existing `~~strikethrough~~ (RESOLVED
<date> in <commit>)` + preceding HTML-comment convention (matching the
"Cross-review base detection" and `PipelineCheckpoint` migration rows). The
other two rows added alongside it in the same self-review — the `max_seats`
read-then-append race and the unverified agmsg CLI flag-shape assumption —
were re-checked against the current diff and remain accurate; left
unchanged.

### docs/tech-debt/README.md — new entry: send/wait/read verb test coverage gap

**Gap identified by /test. Recorded.**

The test report's Coverage and Test gaps sections flag `Verbs.Send`,
`Verbs.Wait`, `Verbs.Read` (`internal/org/verbs.go:54,131,156`) and their CLI
wiring as 0.0% covered under both a package-local and a combined
`-coverpkg=./internal/org/...,./internal/cli/...` run — confirmed via
`grep -n "func Test" internal/cli/org_test.go` in the test report (only
spawn/status/disband are covered). This is a real, non-trivial gap: 3 of the
7 verbs in PR①'s scope ship with no behavioral proof. Added a new tech-debt
row recommending it be closed alongside PR② (seat operation phase,
`docs/specs/2026-08-01-org-runtime.md`), which will exercise these verbs live
once seats run resident sessions — matching the test report's own
recommendation.

### docs/plans/active/2026-08-01-org-runtime-mechanism.md — Progress checklist

**Three items were stale.**

Updated:
- `[ ] Review artifact created` → `[x] Review artifact created (docs/reports/self-review-2026-08-01-org-runtime-mechanism.md)`
- `[ ] Verification artifact created` → `[x] Verification artifact created (docs/reports/verify-2026-08-01-org-runtime-mechanism.md)`
- `[ ] Test artifact created` → `[x] Test artifact created (docs/reports/test-2026-08-01-org-runtime-mechanism.md)`

`[ ] PR created` left unchecked — that is `/pr`'s job, not yet run.

### docs/plans/active/2026-08-01-org-runtime-mechanism.md — Acceptance criteria checkboxes (AC-1..AC-10)

**Additional drift beyond the assigned known-items list, closed while in this file.**

The verify report explicitly flagged these as "stale but expected... flagging
for `/sync-docs` or plan owner to tick, not treating as a failure": all ten
AC checkboxes were still `[ ]` even though verify found them met (static) and
test found them proven (behavioral, all named tests passing under `-race`).
Ticked all ten with a short evidence pointer (named test or report reference)
per box, since both /verify and /test independently confirmed each one for
PR①'s scope and the verify report treated ticking them as in-scope for this
step.

### docs/specs/2026-08-01-org-runtime.md — FR checkboxes

**No change. Correctly deferred.**

FR-1/FR-2/FR-9 cover only their "mechanism part" in PR①; FR-3 through FR-11
describe later PRs (座席化, Lead 自律編成, Watchdog, 旧系撤去) that are
explicit Non-goals of this plan. Left all FR checkboxes unchecked — the spec
is a multi-PR umbrella document, not this PR's own progress tracker. Ticking
them now would misrepresent scope not yet shipped and would preempt PR⑤'s
full doc overhaul, which this plan explicitly defers.

### AGENTS.md / templates/base/AGENTS.md — repo map

**No drift.** Already carries the `internal/org/` repo-map line
(`internal/org/ — org runtime mechanism layer: envelope validation, seat saga
manifest, receipts, herdr/agmsg driver adapters (spec:
docs/specs/2026-08-01-org-runtime.md)`), byte-identical in both root and
`templates/base/` mirrors (`diff AGENTS.md templates/base/AGENTS.md` — no
output).

### CLAUDE.md, README.md, .claude/rules/

**No drift. No update required.**

`grep -n "internal/org\|ralph org\|org runtime" README.md` — no matches.
`grep -rl "internal/org\|ralph org " .claude/rules/` — no matches. Per the
plan's own scope ("ドキュメント最小更新... 全面改稿は PR⑤"), the pipeline/
skill/rules rewrite for the full org-runtime architecture is explicitly out
of scope here; PR① does not change any behavior, contract, or workflow that
these files describe today (`/work`, `/loop`, model routing, subagent
policy are all untouched — confirmed by the verify report's do-not-touch
diff check).

### docs/quality/quality-gates.md, docs/quality/definition-of-done.md

**No drift.** The only "org" substring match in `docs/quality/quality-gates.md`
is the generic phrase "org or repo-specific policy checks" — unrelated to
`ralph org`/`internal/org`. `definition-of-done.md` has no references at all.
Neither describes a quality gate that PR① changes.

## Files changed

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | Marked herdr-agent-namespace row RESOLVED (with HTML-comment provenance, matching existing convention); added new row for `Verbs.Send/Wait/Read` 0% coverage gap, tied to PR② |
| `docs/plans/active/2026-08-01-org-runtime-mechanism.md` | Ticked "Review/Verification/Test artifact created" in Progress checklist; ticked AC-1..AC-10 with evidence pointers |
| `docs/reports/sync-docs-2026-08-01-org-runtime-mechanism.md` | This report |

## Files with no change needed

| File | Reason |
|------|--------|
| `docs/specs/2026-08-01-org-runtime.md` | FR checkboxes span PR①–PR⑤; PR① covers only the mechanism part of FR-1/FR-2/FR-9. Left unchecked — spec is a multi-PR umbrella, full doc overhaul is PR⑤'s job |
| `AGENTS.md` / `templates/base/AGENTS.md` | Repo map already carries the `internal/org/` line, byte-identical in both mirrors |
| `CLAUDE.md` | No org-runtime references; no workflow/contract this PR changes |
| `README.md` | No org-runtime references |
| `.claude/rules/*.md` | No org-runtime references; pipeline/skill/rules rewrite is explicit PR⑤ scope |
| `docs/quality/quality-gates.md`, `docs/quality/definition-of-done.md` | No relevant references; no quality gate changed by this PR |
| `docs/tech-debt/README.md` — `max_seats` race row, agmsg flag-shape row | Re-checked against current diff; both remain accurate and unresolved, left unchanged |

## Verdict

Documentation is in sync for PR①'s scope. No blocking drift. The full
architecture-doc overhaul (AGENTS.md/CLAUDE.md/README/.claude/rules rewrite
for the org-runtime replacement of the pipeline flow) is correctly deferred
to PR⑤ per the plan's own scope and non-goals. Proceed to `/cross-review`.
