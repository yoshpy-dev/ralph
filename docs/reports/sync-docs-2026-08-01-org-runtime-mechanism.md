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

---

## Cycle 2 (fix-and-revalidate re-run)

- Date: 2026-08-02
- Agent: `doc-maintainer` subagent (Claude Code, `/sync-docs`), pipeline cycle 2 of 2
- Scope: the delta since the cycle-1 reviewed state — two fix commits landed in response to the cross-review ACTION_REQUIRED finding (`docs/reports/cross-review-triage-org-runtime-mechanism.md` #1): `4dcfc03` (fix: return idempotent spawn before envelope validation) and `e6a162c` (fix: reject stateless envelope violations before stale compensation), plus their cycle-2 self-review/verify/test report addenda.
- Worktree: `docs/spec-org-runtime`, HEAD `9f68587`

### Drift check results

#### docs/plans/active/2026-08-01-org-runtime-mechanism.md — Implementation notes section (stale)

**Drift detected, flagged by the cycle-2 verify report. Resolved.**

The "Implementation notes (deviations from initial outline)" section did not
mention either cycle-2 fix commit or the resulting spawn branch ordering.
Per `.claude/rules/planning.md` ("record meaningful deviations from the
original plan instead of silently drifting"), added one entry naming both
commits, summarizing what each fixed (cross-review ACTION_REQUIRED #1 —
at-cap idempotent retry incorrectly rejected; cycle-2 self-review MEDIUM —
envelope-invalid respawn triggered destructive compensation before
rejection), and stating the final 4-step ordering confirmed by the verify
report's Cycle 2 section: idempotent early return → stateless envelope
validation (`ValidateSpawnEnvelope`) → stale-in-flight compensation
(`compensateStale`) → capacity validation (`ValidateSpawnCapacity`) → saga
side effects. Cross-referenced the verify report Cycle 2 section and the
cross-review triage report for evidence pointers rather than re-deriving
line numbers here.

#### docs/tech-debt/README.md — TOCTOU and verb-coverage-gap rows

**No drift. Confirmed still accurate.**

Re-checked both rows against the cycle-2 delta (`git diff --stat
9bfe07e...HEAD -- internal/ scripts/ templates/` — only
`internal/org/{driver/driver,driver/herdr,driver/probe,envelope,manifest,spawn,spawn_test}.go`
touched):

- `max_seats` read-then-append race row (line 51): the two fix commits
  reorder *which* validations run before/after stale-in-flight compensation;
  neither adds locking around the `Read → ActiveSeatCount → validate →
  appendEvent` window. The row's description and its deferred-to-PR③
  resolution plan remain accurate — left unchanged.
- `Verbs.Send`/`Wait`/`Read` 0% coverage row (line 53): neither fix commit
  touches `internal/org/verbs.go` or `internal/cli/org.go` (confirmed by the
  `git diff --stat` above — only `spawn.go`/`spawn_test.go` and the
  driver/envelope/manifest files changed). The coverage gap is unaffected —
  left unchanged.

#### Consistency sweep — old validation ordering references

**No drift found.**

`grep -rn "ValidateSpawn\|idempotent.*envelope\|envelope.*idempotent" docs/
AGENTS.md CLAUDE.md .claude/rules/` — the only hits outside the
already-current verify/self-review/test reports are:

- `docs/reports/cross-review-triage-org-runtime-mechanism.md` — describes the
  *pre-fix* bug as the original reviewer finding (ACTION_REQUIRED #1). This
  is a point-in-time triage artifact (same class as verify/self-review/test
  reports before their own Cycle 2 addenda), not a description of current
  behavior; left unchanged, consistent with how cycle-1 triage findings are
  never rewritten after the fix lands.
- `docs/specs/2026-08-01-org-runtime.md` FR-1 — describes the verb-level
  contract ("herdr pane 作成 → worktree 用意 → agmsg チーム参加 → 役割プロン
  プト投入 → エンベロープ検証", spawn's external step sequence), not the
  internal idempotent/capacity split. The cycle-2 verify report already
  confirmed neither fix commit touches spawn's external contract (CLI flags,
  manifest field names, exit codes) — only internal validation ordering — so
  no spec update is implied. No other doc references the old ordering.

#### AGENTS.md, CLAUDE.md, README.md, .claude/rules/

**No drift.** Same conclusion as cycle 1 — this delta is confined to
`internal/org/` internals; no behavior, contract, or workflow described by
these files changed.

### Files changed (Cycle 2)

| File | Change |
|------|--------|
| `docs/plans/active/2026-08-01-org-runtime-mechanism.md` | Added cycle-2 fix commits (`4dcfc03`, `e6a162c`) and the final 4-step spawn ordering to "Implementation notes (deviations from initial outline)" |
| `docs/reports/sync-docs-2026-08-01-org-runtime-mechanism.md` | This Cycle 2 section |

### Files with no change needed (Cycle 2)

| File | Reason |
|------|--------|
| `docs/tech-debt/README.md` | TOCTOU row (max_seats race) and verb-coverage-gap row (send/wait/read) both re-checked against the cycle-2 diff and remain accurate — neither fix commit touches the code paths they describe |
| `docs/reports/cross-review-triage-org-runtime-mechanism.md` | Point-in-time triage artifact describing the pre-fix finding; not rewritten after the fix lands, same convention as other pipeline reports |
| `docs/specs/2026-08-01-org-runtime.md` | FR-1's external verb contract is unchanged by this delta; internal validation-ordering refactor doesn't rise to a spec-level change |
| `AGENTS.md`, `CLAUDE.md`, `README.md`, `.claude/rules/*.md` | No references to spawn's internal validation ordering; no workflow/contract changed |

### Verdict — Cycle 2

Documentation is in sync for the cycle-2 delta. One drift item closed (plan's
Implementation notes section now records both fix commits and the final
spawn ordering, per the cycle-2 verify report's flag). Tech-debt rows and
consistency sweep confirmed clean — no other doc references the superseded
validation ordering. Proceed to `/cross-review`.
