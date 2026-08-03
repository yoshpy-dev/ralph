# Verify report: org runtime 残債バッチのクローズ (org-debt-batch)

- Date: 2026-08-03
- Plan: `docs/plans/active/2026-08-03-org-debt-batch.md`
- Verifier: `verifier` subagent (Claude Code, `/verify`)
- Scope: spec compliance (AC-1..AC-6; AC-7's final gate is `run-verify.sh`, out of this phase's scope per the plan's own note) + static analysis (`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`) on `git diff fde0e84...3410621` (branch `chore/org-debt-batch`). No behavioral test suite was run — that is `/test`'s job.
- Evidence: `docs/evidence/verify-2026-08-03-org-debt-batch.log` (full `run-static-verify.sh` output)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: `NewManifestStore`/`ManifestRelPath`/`NewReceiptStore`/`ReceiptsRelPath` — zero non-test hits, path derivation centralized | Met | `grep -rn 'NewManifestStore\b\|ManifestRelPath\|NewReceiptStore\b\|ReceiptsRelPath' --include='*.go' .` → 0 hits. Confirmed both surviving call sites — `newOrgRuntimeAt` (write, `internal/cli/org.go:121`) and `runStatus` (read, `internal/cli/status.go:58`) — call `org.ManifestPathIn`/`org.ReceiptsPathIn` directly; `orgManifestPath` itself no longer exists in the repo (`grep -rn orgManifestPath --include='*.go' .` → 0 hits) |
| AC-2: `tests/test-no-loop-references.sh` `PATTERN` = 9 knob tokens; file-wide `xreview-helpers.sh` exclusion removed; PASS | Met | `PATTERN` (line 28) lists `RALPH_(FORCE\|IMPLEMENT\|SELF_REVIEW\|VERIFY\|TEST\|SYNC_DOCS\|PR\|PROBE\|ESCALATION)_MODEL` — 9 tokens. `EXCLUDE_REGEX` (line 56) no longer names `xreview-helpers.sh`; only `insights_test.go` and `upgrade_retired_loop_artifacts_test.go` are exempted (both are retirement-proving fixtures, same rationale). Ran `bash tests/test-no-loop-references.sh` directly as a static pattern-guard spot-check (not a behavioral test suite run) → `PASS`, exit 0 |
| AC-3: `ralph insights` org-receipts output contract (group-by org_id×seat_id, honored tri-state, honored-rate excludes unknown, empty-vs-populated same JSON schema, old path gone from help/doc) | Met | `internal/cli/insights.go:225` text line matches the plan's example format verbatim (`ORG %s  SEAT %s  commanded=%s  honored: true=%d false=%d unknown=%d  rate=%s (unknown %d excluded)`). `internal/insights/aggregate.go` `ReceiptsSummary`/`ReceiptOrgStats`/`ReceiptSeatStats` field names/JSON tags match the plan's schema exactly (`path`, `orgs[].org_id`, `orgs[].seats[].{seat_id,commanded_models,honored_true,honored_false,honored_unknown}`, `skipped_lines`). `AggregateReceipts` always initializes `Orgs: []ReceiptOrgStats{}` (aggregate.go:238) — confirmed non-nil empty slice keeps the JSON shape identical whether or not receipts exist. `HonoredRate()` divides `true/(true+false)`, unknown excluded (aggregate.go:73-77). No live reference to the retired `.harness/state/pipeline/model-receipts.jsonl` path remains outside historical/explanatory comments (`internal/insights/receipts.go:11-14`, test doc comments) and archived docs — `ralph insights --help`'s flag text and Long description now describe only the org-receipts default |
| AC-4: watchdog 4 findings each have a green regression test | Met (tests exist and target the right mechanism; not independently re-run — that is `/test`'s job) | (a) `TestStatusCmd_ShowsStateDirSource`, `TestStatusCmd_EmptyStateDirShowsSourceToo` (status_test.go), `TestOrgWatch_Once_BannerShowsStateDirSource` (org_test.go); (b) `TestWatch_ScopeChange_TruncatesOversizedPorcelainOutput`, `TestTruncateScopeOutput_LineCountThenByteBudget`, `TestSendAlert_ProtocolValidationFailureFallback_IncludesSeatHeader` (watch_test.go); (c) `TestPruneEscalated_CapsAtMaxEntries_DropsOldestByEmbeddedTimestamp`, `TestPruneEscalated_AtExactlyMaxEntries_NoPruning`, `TestWatch_EscalateAlert_PrunesAt101stEscalation` (watch_test.go); (d) `TestWatch_TotalBudgetCutoff_ResumeAfterAllSeatsStop_ReAlertsOnNewCutoff` (watch_test.go), extended by the fix commit with `FirstTS` assertions |
| AC-5: upgrade removal-path automated test green is the RESOLVED condition; if automation is too heavy, row updated honestly instead | Met, with a plan-wording caveat (see Documentation drift) | `internal/cli/upgrade_retired_loop_artifacts_test.go` (`TestRunUpgrade_RemovesRetiredLoopArtifactsFromManifest`) is a real automated test using the actual embedded `TemplatesFS`, not a mock — this satisfies AC-5's "automated test green" branch (structural review only here; running it is `/test`'s job). The test's own doc comment and the plan's deviation note both honestly record that `ActionRemove` is notify-only (manifest entry dropped, disk file untouched) rather than literal file deletion — this is a deliberate, correctly-documented reinterpretation of "prove removal", not silent scope-narrowing. Caveat: this plan item never had a pre-existing `docs/tech-debt/README.md` row (self-review's own "known gaps" section flags this explicitly) — see AC-6 |
| AC-6: 5 target tech-debt rows RESOLVED (strikethrough + date + PR ref) | Partially met — 4 of 5 named items are rows that exist and are now RESOLVED; the 5th (upgrade removal-path smoke) never had a corresponding row to resolve | `grep` + manual read of `docs/tech-debt/README.md` confirms 4 rows struck through with `RESOLVED 2026-08-03 in chore/org-debt-batch` and closure comments citing the right commits: watchdog-4-LOW (line 80, Slice 4/`ab4e9da`), C2-5 guard PATTERN (line 88, Slice 2/`57effc3`), C2-3 footgun API (line 90, Slice 1/`3b91875`+`0e510a7`), C2-6 insights repoint (line 92, Slice 3/`e00e301`+`4ddd8e7`). No row exists (or was added) for "upgrade removal-path smoke" — it was only ever a known-gap note in `docs/reports/self-review-2026-08-03-org-runtime-retire-loop.md` and the retire-loop walkthrough, never a `docs/tech-debt/README.md` entry. The diff also adds one *new*, unresolved row (`Cutoff` ratchet scope, line 93) documenting a boundary Slice 4 correctly declined to widen — that addition is honest and expected, not a compliance gap |
| AC-7: final gate `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` + `go test ./... -count=1` green | Out of scope for this phase per the plan's own note ("`run-static-verify.sh` 単体は /verify フェーズの部分ゲートであり AC の対象外") | Not evaluated here; `run-static-verify.sh` (the static half) passed — see Static analysis below. Full `run-verify.sh` (adds tests) is `/test`'s responsibility |

## Self-review fix commit crosscheck (M1/M2/M3, commit `3410621`)

| Finding | Fixed? | Evidence |
| --- | --- | --- |
| MEDIUM-1: `escalateAlert` doc comment still attributed dedupe to `status.Escalated` | Fixed | `internal/org/watch.go:1171-1176` now attributes dedupe to `delete(status.PendingAlerts, alertID)` and states `Escalated` is a bounded, write-only audit trail |
| MEDIUM-2: tech-debt closure note claimed `orgManifestPath` "delegates to" `ManifestPathIn` when it was deleted | Fixed | `docs/tech-debt/README.md` line 89 now reads "`orgManifestPath` was deleted, and both the write path (`newOrgRuntimeAt`) and the read path (`runStatus`) now call `org.ManifestPathIn` directly" — verified accurate against current `org.go`/`status.go` (see AC-1 evidence above) |
| MEDIUM-3: zero-active-seats clear left `FirstTS` stale, diverging from `raiseOrClear`'s fresh-`FirstTS` re-raise convention | Fixed | `internal/org/watch.go:648` adds `rec.FirstTS = ""` alongside `rec.Active = false`, with an expanded comment explaining the `conditionFirstTS` interaction. `internal/org/watch_test.go`'s `TestWatch_TotalBudgetCutoff_ResumeAfterAllSeatsStop_ReAlertsOnNewCutoff` gained three new assertions: non-empty pre-resume `FirstTS`, cleared `FirstTS` after the guard fires, and a fresh (different) `FirstTS` on the resumed re-raise |

All three self-review MEDIUM findings are fixed as recommended; no regressions observed in the fix commit's diff (`docs/tech-debt/README.md` +1/-1 line, `internal/org/watch.go` +19/-5, `internal/org/watch_test.go` +17).

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS | check-sync (0 drifted, 3 known-diff, 10 template-only — none new), check-pipeline-sync (all referencing docs in sync), check-skill-sync (13 skills in lock-step), gofmt (`gofmt: ok`), `go vet` (silent = pass), `golangci-lint` (`0 issues.`), `staticcheck` (silent = pass). Full output: `docs/evidence/verify-2026-08-03-org-debt-batch.log` |
| `grep -rn 'NewManifestStore\b\|ManifestRelPath\|NewReceiptStore\b\|ReceiptsRelPath' --include='*.go' .` | 0 hits | AC-1 evidence |
| `bash tests/test-no-loop-references.sh` | PASS (exit 0) | Spot-checked as a deterministic pattern guard (same category as check-sync.sh), not a behavioral test run |
| `cmp scripts/xreview-helpers.sh templates/base/scripts/xreview-helpers.sh` | Identical | Mirror lock-step confirmed |
| Evidence-file redaction sweep on `docs/evidence/org-debt-batch-2026-08-03.txt` | Clean | `grep -c "/Users/\|/home/"` → 0; `~/...` redaction convention applied consistently (4 occurrences) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/model-routing.md` (org receipts section) | In sync | Describes `.harness/state/org/model-receipts.jsonl` / `internal/org/receipts.go`; no stale pipeline-receipts reference |
| `docs/insights/README.md` | N/A | No receipts-path content to update (file doesn't reference the receipts default) |
| `.claude/skills/org/SKILL.md` | In sync | No stale `--receipts` default references found |
| `internal/insights/receipts.go`, `internal/cli/insights_test.go` retired-path comments | In sync (historical) | Comments explain *why* the old schema reader was deleted rather than presenting it as live — correct use, not drift |
| Plan's Canonical ref / AC-6 wording ("5 row(C2-3 / C2-5 / C2-6 / watchdog deferred LOW 4 件 / upgrade 削除経路スモーク)") | Minor drift | Only 4 of the 5 named items map to an actual `docs/tech-debt/README.md` row; the plan's own framing implies a 5th resolvable row that never existed. Self-review already flagged this ("whether AC-6's '5 rows RESOLVED' is satisfied is /verify's call... the fifth plan item... had no pre-existing register row"). Non-blocking: AC-5's alternate path ("automated test green" without a pre-existing row to resolve) covers the substance; recommend a one-line plan/AC-6 wording fix at `/sync-docs` time ("4 tech-debt rows RESOLVED + 1 new automated regression test for the previously-untracked upgrade removal-path gap") so a future reader doesn't go looking for a 5th row that was never registered |
| New `Cutoff` ratchet tech-debt row (line 93) | In sync | Correctly scoped as a new, honest, unresolved row documenting the boundary of Slice 4's fix; not a compliance gap |

## Observational checks

- Manually re-derived the AC-1 grep, the AC-2 `PATTERN`/`EXCLUDE_REGEX` state, and the AC-3 output-contract format string/JSON schema against current source rather than trusting the self-review's summary — all matched.
- Traced the tech-debt closure comment's two named call sites (`newOrgRuntimeAt`, `runStatus`) to their actual line numbers in `internal/cli/org.go` and `internal/cli/status.go` to confirm the fix-commit wording (M2) is literally true, not just plausible.
- Confirmed `docs/evidence/org-debt-batch-2026-08-03.txt` (the plan's designated evidence target) exists, is redacted per convention, and is referenced correctly from the RESOLVED tech-debt rows it supports.

## Coverage gaps

- Behavioral correctness of the watchdog regression tests (AC-4), the upgrade smoke test (AC-5), and `go vet`'s silence covering all new code paths were confirmed structurally (tests exist, target the right assertions, static analysis is green) but were **not executed** here — full `go test ./... -count=1` and the shell test suite are `/test`'s responsibility per the pipeline split.
- AC-3's "実ファイル手動確認 evidence" (real-file manual confirmation) — `docs/evidence/org-debt-batch-2026-08-03.txt` was read and its redaction/format checked, but the underlying `ralph insights --receipts ~/tmp-scratch/demo-receipts.jsonl` invocation was not independently re-run in this session; treat the transcript as likely-accurate but unverified by re-execution.
- `docs/insights/README.md` was checked for stale content and found to have none, but was not exhaustively diffed against the plan's full doc-drift list beyond the receipts-path concern.

## Verdict

**PASS**, with one non-blocking documentation-wording gap.

- Verified: AC-1, AC-2, AC-3, AC-4 (structural), AC-5, self-review M1/M2/M3 fixes, static analysis (full scope), mirror lock-step, evidence redaction.
- Partially verified: AC-6 (4 of 5 named items are real, correctly-RESOLVED rows; the 5th was never a row — recommend a plan-wording fix, not a code fix, at `/sync-docs`).
- Not verified (out of `/verify` scope): AC-7 (full `run-verify.sh` + `go test ./... -count=1`), behavioral execution of the new/updated Go and shell tests — hand off to `/test`.
