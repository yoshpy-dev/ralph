# Test report: org-runtime-seats

- Date: 2026-08-02
- Plan: `docs/plans/active/2026-08-02-org-runtime-seats.md`
- Tester: `tester` subagent (Claude Code, `/test`)
- Scope: behavioral tests only (static analysis already passed in `/verify`, see `docs/reports/verify-2026-08-02-org-runtime-seats.md`). `./scripts/run-test.sh` requested `changed` scope but resolved to the `golang` full fallback (`scripts/ralph-config.sh` is an unclassified path), so every shell suite plus every Go package ran. Additionally ran the plan-specified explicit command: `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1`.
- Evidence: `docs/evidence/test-2026-08-02-org-runtime-seats.log` (full `./scripts/run-test.sh` output, 1390 lines)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` — shell suites (46 files under `tests/`) | 346+ (assertion-counted suites) + all-PASS suites (`FAIL: 0` gates) | 346 (counted) + all remaining PASS | 0 | 0 | ~90s (background run) |
| `./scripts/run-test.sh` — golang verifier (`go test ./...`, full scope) | 13 packages with test files | 13 | 0 | 0 | ~18s (cached where unchanged) |
| `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1` | 5 packages | 5 | 0 | 0 | ~26s |
| `go test ./internal/org/... -coverprofile=... -covermode=atomic -count=1` | 3 packages | 3 | 0 | 0 | ~3.5s |
| Targeted re-run of the 22 named tests cited in the verify report's AC table (`-run '<regex>' -v`) | 22 | 22 | 0 | 0 | <2s |

Full suite list executed by `run-test.sh` (all green, `FAIL: 0` or explicit `N passed, 0 failed`): `test-agent-phase-boundaries.sh`, `test-branch-name.sh`, `test-check-mojibake.sh`, `test-check-skill-sync.sh`, `test-detect-changed-languages.sh`, `test-detect-languages-terraform.sh`, `test-ensure-pr-ready.sh`, `test-ensure-pr-title-prefix.sh`, `test-gc-artifacts.sh`, `test-insights-append.sh`, `test-insights-pipeline-events.sh`, `test-language-pack-monorepo-roots.sh`, `test-model-routing.sh`, `test-ralph-cleanup-no-remote.sh`, `test-ralph-cli-driver.sh`, `test-ralph-config.sh`, `test-ralph-deprecation-notice.sh`, `test-ralph-dry-run-side-effects.sh`, `test-ralph-orchestrator-branch-names.sh`, `test-ralph-orchestrator-parsers.sh`, `test-ralph-orchestrator-pr-strategy.sh`, `test-ralph-pipeline-functions.sh`, `test-ralph-run-options.sh`, `test-ralph-signals.sh`, `test-ralph-slice-skip-pr.sh`, `test-ralph-status.sh`, `test-ralph-worktree.sh`, `test-run-verify-scope.sh`, `test-secret-scan.sh`, `test-self-review-scope.sh`, `test-sync-skills.sh`, `test-terraform-gitignore.sh`, `test-terraform-pack-verify.sh`, `test-terraform-rule-frontmatter.sh`, `test-verify-mode-split.sh`, `test-xreview-gate-regression.sh`, `test-xreview-prompt-render.sh`.

Go packages (`go test ./...`, full scope): `internal/action`, `internal/cli`, `internal/config`, `internal/insights`, `internal/org`, `internal/org/driver`, `internal/org/protocol`, `internal/scaffold`, `internal/state`, `internal/ui`, `internal/ui/panes`, `internal/upgrade`, `internal/watcher` — all `ok`. `-race` re-run confirmed no data races in the org-runtime-seats packages (`internal/org`, `internal/org/driver`, `internal/org/protocol`, `internal/cli`, `internal/config`).

## Coverage

- Package `internal/org`: **86.7%** of statements (`go test ./internal/org/... -coverprofile -covermode=atomic`), meeting the plan's ~86%+ expectation.
- Package `internal/org/driver`: 92.0% of statements.
- Package `internal/org/protocol`: 97.9% of statements.
- `internal/org/verbs.go` per-function coverage (via `go tool cover -func`):

  | Function | Coverage |
  | --- | --- |
  | `findSeat` | 85.7% |
  | `Send` | 83.3% |
  | `truncateForDetails` | 66.7% |
  | `Wait` | 100.0% |
  | `Read` | 91.7% |
  | `Stop` | 92.0% |
  | `Status` | 88.9% |
  | `Disband` | 78.6% |
  | **file total** | **88.7%** |

  These numbers match `docs/tech-debt/README.md`'s resolved-row note exactly (`Send 83.3%, Wait 100.0%, Read 91.7%, Stop 92.0%`, up from `Send 70.0%, Wait 0.0%, Read 0.0%, Stop 80.0%` before Slice 4) — confirms AC-6's tech-debt closure claim is accurate, not just asserted.
- Branch: not separately instrumented (Go's `-cover` reports statement coverage only; no branch-coverage tool is wired into this repo's Go toolchain).
- Notes: `internal/cli` and `internal/config` were exercised with `-race` per the task but not separately profiled for coverage — out of the explicit ask (only `internal/org` + `verbs.go` per-func numbers were requested).

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | No failures observed across any suite or package. | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| PR① org-runtime-mechanism test suite (spawn saga, status, dry-run, config lockstep) | Still green | `internal/org`, `internal/org/driver`, `internal/cli` all `ok` under both `run-test.sh`'s full golang verifier and the explicit `-race` re-run |
| `internal/config/defaults_sync_test.go` (3-surface model/effort/agmsg_home lockstep drift tripwire) | Still green | `internal/config` package `ok` in both runs; specifically includes the AC-2 `agmsg_home` check added this PR |
| `scripts/check-sync.sh` / `scripts/check-skill-sync.sh` (root ↔ template mirror drift) | Still green | `test-check-skill-sync.sh` all-PASS in `run-test.sh`; `check-sync.sh` itself is a `/verify`-scope static check (already confirmed green in the verify report), not re-run here per the `/test` vs `/verify` boundary |
| Verbs 0% coverage (Send/Wait/Read) — `docs/tech-debt/README.md` closed row | Still resolved, numbers re-verified independently | `go tool cover -func` output above matches the tech-debt row's recorded post-change numbers exactly |

## Test gaps

- **AC-8 (live smoke)** is intentionally not automated — it requires a real herdr + agmsg install and an interactive claude seat. Verified as evidence-only per the plan (`docs/evidence/org-seats-smoke-2026-08-02.txt`), already confirmed in `/verify`; nothing for `/test` to execute here.
- **`truncateForDetails` (66.7%)** and **`Disband` (78.6%)** are the two lowest-covered functions in `verbs.go`. Not blocking (both are inside the 86.7%+ package figure and neither is named in AC-6's tech-debt closure, which scoped Send/Wait/Read/Stop specifically), but a future slice adding a `Disband`-with-partial-failure case or a `truncateForDetails` boundary-length case would close the remaining gap.
- No branch-coverage tooling is configured for Go in this repo; statement coverage is the only instrumented signal.
- Deferred LOW self-review findings noted in the verify report (`leadIdentity` string duplication, doctor error detail, `dryRunSpawn` swallowed errors, evidence-redaction convention) are pre-existing scope decisions, not test gaps — no AC references them and no plan test-case names them.

## Verdict

- Pass: 346+ shell assertions across 36 suites, 0 failed; 13 Go packages via full-scope `go test ./...`, 0 failed; 5 packages re-run with `-race -count=1` (org-runtime-seats-affected scope), 0 failed, no races; all 22 test names cited in the verify report's AC table individually confirmed to exist and pass.
- Fail: none.
- Blocked: none.

**Overall: PASS.** AC-1, AC-3, AC-4, AC-6, AC-9, and AC-11's behavioral proof (delegated by `/verify` to `/test`) is now confirmed: every named covering test exists and passes, `internal/org` coverage is 86.7% (meets the ~86%+ target), and `verbs.go`'s Send/Wait/Read/Stop coverage matches the tech-debt row's resolved figures exactly. Combined with `/verify`'s PASS verdict on AC-2/AC-5/AC-7/AC-8/AC-10 and static analysis, AC-9 (`go test ./...` green) is now fully verified. Safe to proceed to `/sync-docs`.

## AC → test mapping summary

| AC | Behavioral test(s) | Result |
| --- | --- | --- |
| AC-1 (agmsg adapter argv + `AgmsgAvailable`) | `internal/org/driver/agmsg_test.go`: `TestAgmsg_Send/Join/TeamMembers/History/Leave/Whoami/DeliverySet(_InvalidModeRejectedBeforeRun)`, `TestResolveAgmsgHome_Precedence`, `TestAgmsgVersion_RoundTrip/MissingFile`; `internal/org/driver/driver_test.go`: `TestAgmsgAvailable_HomeWithSendScript_NoError`, `TestAgmsgAvailable_HomeWithoutSendScript_ErrNotInstalled`, `TestAgmsgAvailable_PathBootstrapperIgnored` | PASS |
| AC-2 (`agmsg_home` 3-way lockstep) | `internal/config` package (`defaults_sync_test.go` drift check) + `config_test.go` | PASS |
| AC-3 (spawn saga join/HELLO + failure injection) | `internal/org/spawn_test.go`: `TestOrgSpawn_FailureInjection_AgmsgJoin_SeatJoinFails_CompensatesExistingPane`, `TestOrgSpawn_EnsureLeadJoined_ErrorDoesNotFailSaga_WhenSeatJoinAndSendSucceed`, `TestOrgSpawn_FailureInjection_AgmsgSend_DetailsIncludeLeadJoinError` | PASS |
| AC-4 (`--role` templates + variable substitution) | `internal/org/prompts_test.go`: `TestRenderRolePrompt_Reviewer_AllKnownVarsSubstituted`, `TestRenderRolePrompt_QA_AllKnownVarsSubstituted`, `TestRenderRolePrompt_UnknownRole_NoTemplate`, `TestRenderRolePrompt_UnknownPlaceholder_PassesThroughUnchanged`, `TestRenderRolePrompt_EmptyScope_SubstitutesDefaultText`; `internal/cli/org_test.go`: `TestOrgSpawn_UnknownRole_NoTemplateApplied` | PASS |
| AC-5 (`Leave` best-effort on stop/disband) | `internal/org/verbs_test.go`: `TestOrgStop_ExistingSeat_RecordsPaneAndLeaveOutcomes` | PASS |
| AC-6 (Verbs Send/Wait/Read coverage + tech-debt closure) | `internal/org/verbs_test.go` Send/Wait/Read suites; `internal/cli/org_test.go`: `TestOrgWait_UnknownSeat_StillSucceeds_PassthroughToHerdr`, `TestOrgRead_UnknownSeat_NonZeroExit`; coverage numbers cross-checked against `docs/tech-debt/README.md`'s resolved row | PASS |
| AC-7 (`agent-messaging.md` mirror + check-sync) | Covered by `/verify` (static/doc-sync scope); no dedicated behavioral test needed beyond `test-check-skill-sync.sh` (PASS) | PASS (verify-scope) |
| AC-8 (live smoke) | Manual, evidence-only (`docs/evidence/org-seats-smoke-2026-08-02.txt`) — verified in `/verify`, not re-executed here | Not applicable to `/test` |
| AC-9 (`go test ./...` green) | Full-scope `go test ./...` via `run-test.sh` (13 packages, all `ok`) + explicit `-race` re-run of the 5 affected packages | PASS |
| AC-10 (unknown-seat stop/disband guards) | `internal/org/verbs_test.go`: `TestOrgStop_UnknownSeat_ErrorsWithoutAppendingEvent`, `TestOrgStop_UnknownSeat_DryRun_AlsoErrorsWithoutAppendingEvent`, `TestOrgDisband_OnlyStopsExistingActiveSeats_UnknownNeverAppears`; `internal/cli/org_test.go`: `TestOrgStop_UnknownSeat_NonZeroExit` | PASS |
| AC-11 (protocol validator + `--raw` bypass) | `internal/org/protocol/protocol_test.go` (13 `func Test*`, incl. `TestValidate_BodySizeCap_CountsRunesNotBytes`); `internal/cli/org_test.go`: `TestOrgSend_RawFlag_BypassesValidation` | PASS |

(Path-traversal MEDIUM-2 fix, verified alongside AC-3/AC-11 scope: `internal/org/identifier_test.go`: `TestValidateIdentifier`, `TestOrgSpawn_TraversalSeatID_RejectedBeforeAnyFilesystemOrManifestWrite`, `TestOrgSpawn_InvalidOrgID_RejectedBeforeAnyManifestWrite`, `TestOrgSpawn_ValidIdentifiers_PassValidationAndSpawn`; `internal/cli/org_test.go`: `TestOrgSpawn_TraversalSeatID_NonZeroExit_NoDriverCalls`, `TestOrgSpawn_InvalidOrgID_NonZeroExit`, `TestOrgSend_TraversalTo_NonZeroExit` — all PASS.)

## Cycle 2

- Date: 2026-08-02
- Pipeline cycle: 2/2 (`RALPH_STANDARD_MAX_PIPELINE_CYCLES` default cap)
- Tester: `tester` subagent (Claude Code, `/test`)
- Scope: behavioral tests only; re-run after fix-and-revalidate for cross-review triage `ACTION_REQUIRED #1` (`docs/reports/cross-review-triage-org-runtime-seats.md`).
- Delta since Cycle 1 (HEAD `d0e0a55`):
  - `f0cbf11` — fix: reserve underscore as org/seat namespace separator (addresses ACTION_REQUIRED #1: hyphen-joined `<org_id>-<seat_id>` was not unique — `a-b`+`c` and `a`+`b-c` both collapsed to `a-b-c`). Also tightens `identifierPattern` to lowercase-only (matches herdr's own `^[a-z][a-z0-9_-]{0,31}$` agent-name pattern) and adds a combined-length guard against herdr's 32-char agent-name limit.
  - `011a0a7` — fix: address cycle-2 self-review LOW findings (comments, template examples, `PLAN_PATH` replacer restoration). No test-visible behavior change; not re-verified separately beyond the full suite run below (no LOW finding named a test gap).

### Test execution (Cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` — shell suites (all `tests/*.sh`, full scope) | Every suite reports `FAIL: 0` or `N passed, 0 failed` | All | 0 | 0 | ~90s |
| `./scripts/run-test.sh` — golang verifier (`go test ./...`, full scope) | 13 packages with test files | 13 | 0 | 0 | cached/fast |
| `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1` (plan-specified) | 5 packages | 5 | 0 | 0 | ~26s, 0 races |
| Targeted `-run` re-execution of the 5 new/changed `f0cbf11` regression tests, `-v -count=1` | 5 named tests | 5 | 0 | 0 | <1s |

`run-test.sh` resolved `changed` scope to the `golang` full fallback again (`scripts/ralph-config.sh` is still an unclassified path under `detect-languages.sh`), so the same full shell-suite + full-Go-package run executed as in Cycle 1. Evidence: `docs/evidence/test-2026-08-02-org-runtime-seats-cycle2.log` (combined `run-test.sh` output + the explicit `-race` command output, 1396 lines).

### f0cbf11 regression tests — confirmed present and passing

Mapped to cross-review `ACTION_REQUIRED #1` (`docs/reports/cross-review-triage-org-runtime-seats.md` line 25: hyphen-join collision between herdr agent names and prompt file paths):

| Test | File | Purpose | Result |
| --- | --- | --- | --- |
| `TestHerdrAgentNameAndPromptFilePath_UnambiguousAcrossSplit` | `internal/org/identifier_test.go` | New. Proves `herdrAgentName("a-b","c")` ≠ `herdrAgentName("a","b-c")` and the equivalent `promptFilePath` pair, under the `_`-join fix (`a-b_c` vs `a_b-c`) — the exact collision case the reviewer named. | PASS |
| `TestOrgSpawn_CombinedIdentifierLength_RejectedBeforeAnyManifestWrite` | `internal/org/identifier_test.go` | New. Two individually-valid (≤30 char) ids whose `_`-joined herdr name exceeds herdr's 32-char limit must be rejected before any driver call or manifest write. | PASS |
| `TestOrgSpawn_CombinedIdentifierLength_ExactlyAtLimit_Spawns` | `internal/org/identifier_test.go` | New. Boundary case: combined length exactly 32 chars spawns successfully (no off-by-one over-rejection). | PASS |
| `TestValidateIdentifier_LengthBoundary` | `internal/org/identifier_test.go` | New. Pins the 30-char single-identifier cutoff (30 valid / 31 rejected) introduced alongside the lowercase-only charset tightening. | PASS |
| `TestValidateIdentifier` (updated cases) | `internal/org/identifier_test.go` | Updated. Adds `has_underscore` and mixed/upper-case identifiers to the invalid set — underscore is now reserved as the join separator and must never be legal inside a single id; charset is now lowercase-only to match herdr's own pattern. | PASS |

Also re-verified as part of the same commit (existing tests whose expected literals changed from `-` to `_` joins, not new regression coverage but confirmed non-regressed): `TestHerdrAgentName_NamespacesBySeatAndOrg` (`internal/org/spawn_test.go`), `TestOrgSpawn_RoleAndScopeFlags_ExpandTemplateAndRecordScope` and `TestOrgWait_UnknownSeat_StillSucceeds_PassthroughToHerdr` (`internal/cli/org_test.go`) — all PASS, individually re-run with `-v`.

All five new/changed tests were also confirmed present via `git show --stat f0cbf11` (touches `internal/cli/org_test.go`, `internal/org/identifier.go`, `internal/org/identifier_test.go`, `internal/org/spawn.go`, `internal/org/spawn_test.go`) and via a direct `grep` for `TestHerdrAgentNameAndPromptFilePath_UnambiguousAcrossSplit` confirming it exists in `internal/org/identifier_test.go:150`.

### Failure analysis (Cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | No failures observed across any suite or package. | — |

### Regression checks (Cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Cycle 1 PASS baseline (346+ shell assertions, 13 Go packages, `internal/org` 86.7% coverage) | Still green | Full `run-test.sh` re-run, identical suite list, all `FAIL: 0` |
| Hyphen-join collision (`a-b`+`c` vs `a`+`b-c`) — cross-review `ACTION_REQUIRED #1` | Fixed and regression-covered | `TestHerdrAgentNameAndPromptFilePath_UnambiguousAcrossSplit` PASS |
| Combined-length overflow against herdr's 32-char agent-name cap | New guard, regression-covered | `TestOrgSpawn_CombinedIdentifierLength_RejectedBeforeAnyManifestWrite` / `_ExactlyAtLimit_Spawns` PASS |
| `011a0a7` LOW fixes (comments, template examples, `PLAN_PATH` replacer restoration) | No test-visible regression | Full suite green; no LOW finding named a specific test case |

### Test gaps (Cycle 2)

- No new test gaps introduced by `f0cbf11` or `011a0a7`. The Cycle 1 gaps (AC-8 live smoke, `truncateForDetails`/`Disband` coverage, no Go branch-coverage tooling) are unchanged and still apply.
- The `_`-join fix is now the single source of truth for herdr agent naming and prompt file paths; no other call site in the diff still builds a `-`-joined name (confirmed by the full `internal/org` and `internal/cli` package runs passing with the updated literals).

### Verdict (Cycle 2)

- Pass: full shell suite (all `FAIL: 0`), 13 Go packages via full-scope `go test ./...`, 5 packages under `-race -count=1` (0 races), 5 new/changed `f0cbf11` regression tests individually re-run and confirmed present + passing.
- Fail: none.
- Blocked: none.

**Overall: PASS.** Cross-review `ACTION_REQUIRED #1` is confirmed fixed with dedicated regression coverage (`TestHerdrAgentNameAndPromptFilePath_UnambiguousAcrossSplit` plus the combined-length boundary tests). Combined with the unchanged Cycle 1 PASS verdict on all other ACs, the pipeline is clear to proceed to `/sync-docs` (already run, `docs/reports/sync-docs-2026-08-02-org-runtime-seats.md`) and `/cross-review` re-run.
