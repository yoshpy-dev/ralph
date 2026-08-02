# Sync-docs report: org-runtime-watchdog

- Date: 2026-08-03
- Plan: `docs/plans/active/2026-08-02-org-runtime-watchdog.md`
- Spec: `docs/specs/2026-08-01-org-runtime.md` (FR-8 Watchdog 二層)
- Agent: `doc-maintainer` subagent (Claude Code, `/sync-docs`)
- Commit under review: `255e79e` (docs: add test report for org-runtime-watchdog)
- Base: `9abaaed` (main)

## Summary

PR④ (Watchdog 二層: パルス層タイマー + オンデマンドウォッチャー) implemented
`ralph org watch`, the `ALERT` protocol type, `[org.watchdog]` config, the
deadman escalation path, state-dir cwd-relative resolution, and closed 3
tech-debt rows carried from PR③. Self-review found 2 HIGH + 6 MEDIUM
findings; all HIGH and 4 of 6 MEDIUM were fixed in-cycle (commit `79d9f40`),
with M-4 (deadman activity source) and its own follow-up (`AC-5` verify
caveat) left as a narrower-than-recommended fix. Verify report: PASS with two
follow-up items (AC-5/M-4 partial fix, AC-10 state-dir live-smoke line — the
latter closed by the cross-cwd addendum, commit `6a09e64`). Test report: full
pass, zero failures. This sync-docs pass closes the plan's own deferred spec
revision, backfills plan AC/progress evidence, and converts two verify/test
follow-up items into tracked tech-debt rows.

## Drift check results

### Spec FR-8 wording (`常駐座席` / `既定 60 秒`) — stale, corrected

**Drift detected. Resolved.**

Two mismatches between `docs/specs/2026-08-01-org-runtime.md` FR-8 and the
shipped implementation:

1. **Watcher form.** FR-8 described the watcher layer as an "LLM 常駐座席"
   (resident LLM seat). The plan's own Design decisions record a deliberate,
   user-confirmed deviation to an on-demand `claude -p` invocation (async
   single-flight, fired only on a pulse-layer semantic trigger) instead —
   rationale: zero resident cost, a hung judge recovers on the next
   invocation, and it matches the "LLM assist only at trigger time" intent
   from the original spec discussion. The plan explicitly deferred reconciling
   this wording to `/sync-docs`.
2. **Pulse interval.** FR-8 said "既定 60 秒" (default 60s); the shipped
   default is 30s (`internal/config/config.go` `Default()`,
   `templates/base/ralph.toml:122-126`, plan's own Scope bullet). The verify
   report flagged this as a second, previously-unflagged FR-8/implementation
   mismatch and asked `/sync-docs` to reconcile it alongside the resident/
   on-demand wording.

Both fixed in `docs/specs/2026-08-01-org-runtime.md` FR-8: the watcher clause
now reads "パルス層トリガーのオンデマンド LLM 判定" with a one-line pointer
to the plan's Design decisions for the rationale, and the interval now reads
"既定 30 秒". No other spec line references "常駐" in a watchdog-specific
context (the general "常駐 LLM 座席"/"常駐対話セッション" phrasing elsewhere
— Summary, FR-3, Alternatives table — describes the general org-runtime seat
model from FR-3 and is unrelated to FR-8's watcher; `watchdog` there is
already documented as a mechanism identity, not a spawned seat, per
`.claude/rules/agent-messaging.md`).

### Plan Acceptance criteria — stale (unticked despite Met/Pass verdicts)

**Drift detected. Resolved.**

All 13 AC checkboxes (AC-1 through AC-11, including AC-3b/AC-3c) were
unticked even though the verify report scored every one Met or Partially Met
and the test report scored AC-1..AC-9/AC-11 behaviorally verified. Ticked all
13 and appended an evidence pointer (file/function + covering test names) to
each, including the two known-open caveats inline: AC-5 now notes the
`leadActivityEventCount` scoping gap (tech-debt row added, see below), and
AC-7/AC-10 note that the cross-cwd live confirmation is the addendum commit
`6a09e64`, not the original `2026-08-02` evidence capture.

### Plan Progress checklist — stale (artifacts existed but weren't ticked)

**Drift detected. Resolved.**

"Review artifact created" and "Verification artifact created" were unticked
even though `docs/reports/self-review-2026-08-02-org-runtime-watchdog.md` and
`docs/reports/verify-2026-08-02-org-runtime-watchdog.md` both exist and are
referenced elsewhere in the plan's own Implementation notes. Ticked both with
a one-line summary of each report's verdict. "PR created" remains unticked —
correct, since `/pr` has not run yet.

### `docs/tech-debt/README.md` — 2 new rows added for verify/test follow-ups

**Gap identified by verify/test reports. Recorded.**

Neither the verify report's AC-5/M-4 caveat nor its 3 still-open LOW findings
(source-return discard, oversized scope-change ALERT truncation, unreachable
`Escalated` guard + unbounded map growth) had a tech-debt row — both reports
explicitly asked for one rather than letting the findings age out silently
(the same escalation discipline the repo already applies to L-5 in the prior
PR's tech-debt history). Added two new open rows:

1. **`leadActivityEventCount` deadman-activity scoping (M-4 partial fix).**
   Not scoped to `org_id` or lead-attributable seats, so an unrelated seat's
   (or, in a shared state dir, a different org's) manifest event can
   prematurely clear a pending deadman alert. Trigger: PR⑤, or the first live
   multi-seat/multi-org deadman incident.
2. **3 remaining deferred LOW findings** (`source`-return discard in
   `ResolveOrgStateDir` call sites, oversized scope-change ALERT truncation
   with no `SEAT` fallback header, unreachable `Escalated` guard + unbounded
   `Escalated` map growth). Trigger: next touch to `internal/org/watch.go` or
   the watch-banner code in `internal/cli/org.go`.

Both rows cite the self-review, verify, and test reports and the exact
function/line locations. Neither blocks any AC.

### Cosmetic M-numbering drift in fix-commit comments — corrected (trivially greppable)

**Drift detected. Resolved.**

The verify report noted (Self-review fix-commit cross-check, closing note)
that commit `79d9f40`'s inline code comments label two fixes "self-review
M-7"/"M-8" while the self-review report's own table numbers the same findings
M-5/M-6 — an internal off-by-two drift inside the fix commit's own comments,
not a doc-vs-code compliance gap. Confirmed via `grep -n "M-[0-9]"` across
`internal/org/watch.go`, `internal/org/watcher.go`, `internal/cli/org.go`,
and their `_test.go` files: code comments used `M-3`→`M-5` (Honored
tri-state), `M-4`→`M-6` (deadman activity source), `M-5`→`M-7` (`--once`
`WaitGroup`), `M-6`→`M-8` (stall time term), consistently shifted by +2
relative to the self-review report's row order. This was trivially greppable
(all occurrences match `self-review M-[5-8]` with no ambiguous context), so
realigned all 16 comment occurrences to the report's own M-3/M-4/M-5/M-6
numbering (comment-only change; `gofmt`/`go vet`/`go build`/package tests all
re-confirmed green after the edit — see Verification below). Did not touch
`internal/cli/org_test.go:1148`'s unrelated "self-review MEDIUM-2" comment,
which references a different PR's (org-runtime-lead) self-review finding
under its own numbering scheme.

### No other drift found

- `AGENTS.md` `internal/org/` repo-map line already names "two-layer
  watchdog (pulse watch + on-demand watcher)" — matches shipped scope, no
  change needed.
- `.claude/rules/agent-messaging.md` `ALERT` row + 3 mirrors — `cmp`
  byte-identical, confirmed again in this pass.
- `.claude/skills/org/SKILL.md` `watch` verb row + cwd-convention wording +
  4 mirrors — `check-skill-sync.sh` green, confirmed again in this pass.
- `docs/quality/definition-of-done.md` redaction convention line — present,
  no change needed.
- `docs/tech-debt/README.md`'s 3 PR④-targeted rows from the plan's own Scope
  (state-dir cwd-split, `/org` skill description language, batchable-3) are
  already struck through with `RESOLVED 2026-08-02` annotations — no change
  needed beyond the 2 new rows above.
- `CLAUDE.md` / `README.md` / `docs/quality/definition-of-done.md` pipeline
  references — this PR does not touch pipeline order or skill set; no update
  needed.

## Verification

- `./scripts/check-sync.sh` — PASS (0 DRIFTED, 179 IDENTICAL, 3 KNOWN_DIFF).
- `./scripts/check-skill-sync.sh` — PASS (14 skills in lock-step).
- `./scripts/run-static-verify.sh` — PASS (`gofmt: ok`, `0 issues.`, mirror
  gates OK); full-scope fallback for the same reason the verify/test reports
  noted (`scripts/ralph-config.sh` classified "unclassified").
- `go build ./...` — clean.
- `go vet ./internal/org/... ./internal/cli/...` — silent (pass).
- `go test ./internal/org/... ./internal/cli/... ./internal/config/...` —
  all green (`internal/org` 7.9s, `internal/cli` 23.2s, `internal/org/driver`
  and `internal/org/protocol` and `internal/config` cached-pass), re-run
  after the comment-only M-numbering fix to confirm no behavioral change.

## Files changed in this pass

- `docs/specs/2026-08-01-org-runtime.md` — FR-8 wording (watcher form,
  interval default).
- `docs/plans/active/2026-08-02-org-runtime-watchdog.md` — AC-1..AC-11 ticked
  with evidence pointers; Progress checklist rows ticked.
- `docs/tech-debt/README.md` — 2 new rows (deadman activity scoping, 3
  remaining deferred LOWs).
- `internal/org/watch.go`, `internal/org/watch_test.go`,
  `internal/org/watcher.go`, `internal/org/watcher_test.go`,
  `internal/cli/org.go`, `internal/cli/org_test.go` — comment-only
  M-numbering realignment (no behavior change).

## Recommendation

**Proceed to `/cross-review`.** All doc drift closed or explicitly tracked;
static/mirror gates and the affected Go packages remain green after the
comment-only fix. Carry the 2 new tech-debt rows forward as named follow-ups
(not blocking `/pr`), consistent with the verify and test reports'
recommendation.
