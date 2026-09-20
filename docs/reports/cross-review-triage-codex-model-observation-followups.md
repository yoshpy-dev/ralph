# Cross-review triage report: codex-model-observation-followups

- Date: 2026-09-20 (cycle 1 and cycle 2)
- Plan: docs/plans/active/2026-09-20-codex-model-observation-followups.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: cycle 1 = 1, cycle 2 = 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0
- Note: the summary line reflects cycle 2 (current state). Cycle 1 was ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0; it was fixed (row kept below as history, marked resolved) and re-verified in cycle 2.
- User decision (2026-09-20, AskUserQuestion): fix AR-1 and re-run the full pipeline as cycle 2/2

## Triage context

- Active plan: docs/plans/active/2026-09-20-codex-model-observation-followups.md
- Self-review report: docs/reports/self-review-2026-09-20-codex-model-observation-followups.md (cycle 1: 2 MEDIUM + 5 LOW, all fixed in 6564680)
- Verify report: docs/reports/verify-2026-09-20-codex-model-observation-followups.md (pass)
- Test report: docs/reports/test-2026-09-20-codex-model-observation-followups.md (pass; 50x / 20x race / 200x repeats were clean on this machine)
- Reviewed HEAD: 7c95d3f. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the org and CLI suites (pass) and then repeated one test under the race detector; it did not start a live seat. No finding on the production change itself.
- Implementation context summary: this branch makes the codex session observer honour a deadline inside a pass (cooperative `ctx.Err()` checks while collecting candidates, before opening a record, once per line). Before this branch the observe timeout only bounded the waits BETWEEN passes, so the test helper `testOrg()` could pin `CodexModelObserveTimeout` to 1 ms for every test without affecting a pass that finds its record. With the deadline now enforced inside the pass, the same 1 ms also bounds the filesystem scan of tests that EXPECT a successful observation.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 (cycle 1, resolved in 5e239b0, 711cd9d, a1afc0c) | [P2] Increase the budget for successful observation tests. Under race instrumentation `TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver` intermittently fails (7 of 300 runs for the reviewer) because `codexGuardedOrg` inherits `testOrg`'s 1 ms observation timeout, which this patch now enforces during Stop's filesystem scan; a valid fixture can then yield an empty receipt. | Real: `testOrg()` (`internal/org/spawn_test.go:259-260`) sets `CodexModelObserveTimeout: time.Millisecond`, and both `observeCodexSpawnReceipt` and `observeStopModelReceipt` now derive their scan ctx from that value, so every test that expects `found` races a 1 ms deadline against ReadDir + Lstat + open + a few line reads. My own 300-run race repeat on this machine was clean (0 failures), and the tester's repeats were clean too, which only shows the margin is machine-dependent: a loaded two-core CI runner is slower than this laptop, and the repo's CI has been broken before by timing that passed locally. The flakiness is introduced by this patch, not pre-existing. Worth fixing before merge: an intermittently red CI on main costs more than one more pipeline run. The fix is test-only: tests that expect a successful observation get a generous budget (a found record returns at once, so this does not slow them), set AFTER the spawn in the Stop tests whose spawn is meant to come back unknown (otherwise that spawn would wait the whole budget); tests that check timeout behaviour keep their tiny budgets. | `internal/org/spawn_test.go`, `internal/org/verbs_test.go` (and `internal/cli` tests if they rely on a tiny budget for a successful observation) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 2 (2026-09-20, after the cycle 1 fix)

- Reviewed HEAD: 5bf9886. Same reviewer command as cycle 1 (explicit model and effort, stdin closed).
- Reviewer verdict, verbatim: "No actionable regressions were found. All Go tests, targeted race tests repeated 10 times, org/CLI vet checks, and template/skill synchronization checks passed. Live Codex/herdr integration was not exercised."
- Findings: 0. Nothing to triage; Case C (proceed to /pr).
- How the cycle 1 finding was resolved (test-only): tests that expect a found record run with `testCodexObserveGenerousBudget` (30 s; a match returns at once); Stop tests whose Spawn is meant to stay unknown keep the tiny budget for the Spawn and raise it before Stop; the read-error-reason test, whose budget must run out only after one pass completed, has 200 ms; timeout tests keep tiny budgets; internal/cli uses a per-test override. Self-review cycle 2 found the read-error test as the one remaining member of the class (HIGH) and it was fixed in the same cycle.
- Between the two reviews the full pipeline re-ran: self-review cycle 2 (merge after the HIGH; 1 HIGH + 3 LOW, all fixed), verify cycle 2 (pass, no drift), test cycle 2 (pass: 280 race iterations under three contention shapes, injected scan delays of 3 / 10 / 25 / 60 ms all clean, red/green checks on the budget constant and the raise-before-stop helper), sync-docs cycle 2 (no doc change needed).
- Known gap carried to the PR: neither reviewer nor pipeline exercised a live codex seat; the flake the reviewer reported in cycle 1 was never reproduced on this machine, so the fix is validated by delay injection rather than by reproducing the original failure.
