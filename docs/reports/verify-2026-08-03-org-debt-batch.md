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

---

# Cycle 2 — delta re-verify (AR-1 + cycle-2 self-review doc-fix commits)

- Date: 2026-08-03
- Verifier: `verifier` subagent (Claude Code, `/verify`, cycle 2/2)
- Scope: **delta only** — `git diff 4a5744e..2b71fd1` (branch `chore/org-debt-batch`, HEAD `2b71fd1`), i.e. everything after the Cycle-1 verify report's pinned HEAD (`3410621`). Cycle 1 above is not re-run; only the delta commits are evaluated: `bf37b24` (test report), `5147f2a` (sync-docs: AC-6 wording fix + model-routing.md receipts-consumer note), `0a40991` (cross-review triage), `53145ea` (AR-1 doctor-overclaim wording fix), `562541d` (cycle-2 self-review section), `2b71fd1` (fix commit: register cell C2-1, deferred-LOW row C2-2, checkboxes C2-5, plus C2-3/C2-4 comment precision folded in).
- Evidence: `docs/evidence/verify-2026-08-03-org-debt-batch-cycle2.log` (full `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` re-run against the `2b71fd1` tree)

## Static analysis (re-run)

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS | check-sync (0 drifted, 3 known-diff, 10 template-only — unchanged from cycle 1), check-pipeline-sync (in sync), check-skill-sync (13 skills), gofmt (`gofmt: ok`), `go vet` (silent = pass, consistent with cycle-1's own observation), golangci-lint (`0 issues.`), `staticcheck` (silent = pass). No behavior-affecting code changed in the delta (only comments in `internal/org/watch.go` and doc/register files), so a green re-run was expected and confirms no static regression was introduced by the fix commits. |
| `git status --porcelain` | clean | No uncommitted changes in the worktree at HEAD `2b71fd1`. |
| `cmp scripts/xreview-helpers.sh templates/base/scripts/xreview-helpers.sh` | Identical | Mirror lock-step still holds (delta touched no mirrored files). |
| `bash tests/test-no-loop-references.sh` | PASS (exit 0) | Re-spot-checked; delta did not touch this guard or its target files, PASS is expected continuity, not new evidence. |

## AR-1 fix crosscheck (commit `53145ea`)

| Finding | Fixed? | Evidence |
| --- | --- | --- |
| AR-1 (cross-review): watchdog-4-LOW closure note overclaimed that `doctor` "prints the org state check line when applicable and otherwise notes it as not applicable" | **Fixed** | `docs/tech-debt/README.md` (watchdog-4-LOW RESOLVED note) now reads "`doctor` was left unchanged -- it has no org state-dir diagnostic line to annotate ..., so the doctor half of sub-item (1) was closed as not-applicable, not implemented." `grep -n "ResolveOrgStateDir\|state-dir\|StateDir" internal/cli/doctor.go` → 0 hits, confirming the corrected claim (no diagnostic line exists at all) is literally true, unlike the prior "prints when applicable" phrasing which implied conditional behavior that was never implemented. |

## Cycle-2 self-review finding crosscheck (commit `2b71fd1`, findings C2-1..C2-5)

| Finding | Required before `/pr`? | Fixed? | Evidence |
| --- | --- | --- | --- |
| C2-1 (MEDIUM): C2-3 tech-debt row's **Related** column still cited the deleted `orgManifestPath` symbol | Yes | **Fixed** | `docs/tech-debt/README.md` line 55 (struck C2-3 row, Related column) now reads `` `internal/cli/org.go` (`newOrgRuntimeAt`), `internal/cli/status.go` (`runStatus`) `` — exactly the recommended replacement. Both are confirmed-live call sites of `org.ManifestPathIn` (`org.go:121`, `status.go:58`), matching cycle-1's own AC-1 evidence. `orgManifestPath` no longer appears anywhere in the file. |
| C2-2 (MEDIUM): six cycle-1 LOW findings were unfixed and unregistered, and would fall off the edge at the cycle cap | Yes | **Fixed, with one item silently dropped from the batch — see gap below** | A new, correctly-unstruck (not marked RESOLVED) row was added at `docs/tech-debt/README.md:87`: "6 batchable LOW findings from org-debt-batch cycle-1 self-review, deferred at merge". Verified the six named items against current source: (1) `status.Escalated` write-only — confirmed no reader outside decl/init/write (`grep -n "Escalated\b" internal/org/watch.go` → lines 272/528/1195 only, no `checkDeadman` read); (2) `filepath.Join(dir, "receipts.jsonl")` test-side asymmetry — confirmed present at exactly the four cited lines (`report_test.go:137,216`, `spawn_test.go:193`, `watch_test.go:276`); (3) `SEAT: %s` empty-value fallback — confirmed at `watch.go:917` verbatim; (4) `printStateDirLine` single-site adoption + `state_dir_source,omitempty` — confirmed at `status.go:263`; (5)/(6) not independently re-verified (test-hygiene / cosmetic, low-risk claims, consistent with self-review's own evidence). **Gap**: the batched row's six items map onto self-review's Escalated / ReceiptsPathIn / SEAT-header / printStateDirLine+omitempty / t.Cleanup / cosmetic-nits LOW findings, but **omit** the seventh cycle-1 LOW finding — "Slice 2 rewrote more of the `xreview-helpers.sh` header than the guard required" (self-review report, Cycle 1 LOW row, "unnecessary-change") — which C2-2's own recommendation text explicitly named for inclusion ("over-wide `xreview-helpers.sh` header rewording"). Confirmed still unfixed and now unregistered: `scripts/xreview-helpers.sh:9-16` still reads "per-slice driver script" / "per-slice + multi-worktree driver scripts" rather than the original "pipeline"/"orchestrator" wording, and no tech-debt row anywhere mentions it. This is a real, small completeness gap in the C2-2 fix, not a fabricated one — see Documentation drift below. |
| C2-3 (LOW): `evaluateTotalBudget`'s new comment cited a stale line number (`watch.go:838`) for `raiseOrClear`'s fresh-`FirstTS` behavior, invalidated by the same commit's own earlier hunk | Folded in (not required, but recommended) | **Fixed** | `internal/org/watch.go:640-641`: the citation now reads "matching `raiseOrClear`'s own re-raise path, which always writes a fresh FirstTS" — function name plus behavior, no line number. Grep-stable per `.claude/rules/architecture.md`'s naming guidance. |
| C2-4 (LOW): AR-1's fixed wording ("its only org check, `checkOrgEnvelope`, reports `[org]` config") was itself imprecise — `doctor` has two more org-runtime checks (`checkHerdrAvailable`, `checkAgmsgAvailable`) plus opt-in model probes | Folded in (not required, but recommended) | **Fixed** | Same `docs/tech-debt/README.md` note now reads "its org-related checks (`checkOrgEnvelope`, herdr/agmsg availability, opt-in model probes) report config and tool availability, not state-dir resolution" — matches `internal/cli/doctor.go`'s actual Check 8/9/10 + probes inventory (verified at `doctor.go:88-99`, `:473-481`, `:511`, `:529-538` per the self-review's own citations). |
| C2-5 (LOW): plan checkbox asymmetry — "Self-review artifact created" / "Verify artifact created" left unticked despite both artifacts existing since `d9563d6`/`4a5744e` | Yes | **Fixed** | `docs/plans/active/2026-08-03-org-debt-batch.md:116-117`: both boxes now `[x]`. All four post-implementation-pipeline checkboxes plus all five slice checkboxes are now consistently ticked. |
| C2-6 (LOW): missing `verify`/`sync_docs` insight events for cycle 1 | Not required before `/pr` (self-review's own "Fix before /pr" list named only C2-1/C2-2/C2-5) | **Not fixed, correctly out of scope for this commit** | `docs/insights/events/2026-08-03-org-debt-batch.jsonl` still holds only `self_review`(×2)/`test`/`cross_review` phase events — no `verify` or `sync_docs` entries. This is an accepted, self-review-acknowledged gap (best-effort insight backfill), not a compliance failure; flagging as a known remaining gap below rather than a finding, consistent with self-review's own scoping. |

## Documentation drift (Cycle 2 delta)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/plans/active/2026-08-03-org-debt-batch.md` AC-6 wording | In sync | `5147f2a` added the "verify 時点の訂正" caveat matching cycle-1's own recommendation almost verbatim — resolves the cycle-1 non-blocking gap. |
| `.claude/rules/model-routing.md` (org receipts / `ralph insights` consumer note) | In sync | New sentence ("`ralph insights` reads this file by default … tri-state `honored`") matches `internal/cli/insights.go:47-49` and `internal/insights/aggregate.go`'s actual schema, cross-checked in cycle 1. |
| `docs/tech-debt/README.md`, C2-2 batched LOW row | Minor drift (new, this cycle) | See C2-2 crosscheck above — the row omits the "over-wide `xreview-helpers.sh` header rewording" LOW finding that C2-2's recommendation explicitly asked to include. Non-blocking: it is a cosmetic wording issue in a comment header, no behavior or correctness impact, and the pattern is otherwise unchanged from what Slice 2 shipped (guard-completion itself, AC-2, remains correctly PASS — this is unrelated to the guard's function). Recommend either a one-line addendum to the existing batched row or a follow-up note at `/pr` time; does not block this verify cycle's PASS verdict. |
| Plan progress checklist | In sync | `2b71fd1` ticks the two remaining boxes (C2-5); all pipeline-artifact and slice checkboxes are now `[x]`. |

## Verdict (Cycle 2)

**PASS.**

- Verified this cycle: static analysis re-run green on the `2b71fd1` tree (no regression from the delta), AR-1 fix (doctor-overclaim wording), C2-1 (register Related-column cell), C2-3/C2-4 (comment precision, folded into the same commit), C2-5 (plan checkboxes) — all confirmed fixed exactly as their recommendations specified, checked against current source rather than trusting the self-review's own summary.
- Partially verified: C2-2 (batched deferred-LOW tech-debt row) — five of the six self-review-recommended items are correctly captured and line-reference-accurate; one (`xreview-helpers.sh` header over-rewording) was silently dropped from the batch and remains both unfixed and unregistered. Non-blocking doc-completeness gap, not a code defect.
- Confirmed not regressed: C2-6 (missing `verify`/`sync_docs` insight events) remains open by design — self-review did not require it before `/pr`, and it is unchanged by this delta.
- Not verified (out of `/verify` scope, unchanged from Cycle 1): AC-7's full-suite gate and behavioral test execution — that is `/test`'s job; the delta's own `docs/reports/test-2026-08-03-org-debt-batch.md` (commit `bf37b24`) already reports a green run predating this fix commit, and the fix commit changed no test-affecting code.
- **Recommended smallest next check**: append the missing `xreview-helpers.sh` item to the C2-2 batched tech-debt row (one-cell edit) before or shortly after `/pr` — cheapest way to close the completeness gap this cycle found.
