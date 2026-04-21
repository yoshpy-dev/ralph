# Test report: fix-ralph-upgrade-manifest-hash-loss

- Date: 2026-04-21
- Plan: `docs/plans/active/2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md`
- Tester: tester subagent (Claude)
- Scope: Behavioral tests only (unit + integration + regression) for the upgrade manifest hash heal / pack namespacing fix. Static analysis and verify handled upstream by `/verify`.
- Evidence: `docs/evidence/test-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.log`
- Branch: `fix/ralph-upgrade-manifest-hash-loss`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go test ./... -count=1` (full repo) | 167 | 165 | 0 | 2 | ~22s wall (parallel packages) |
| `go test -v -run 'TestComputeDiffs_.*' ./internal/upgrade/...` (targeted) | 8 | 8 | 0 | 0 | 0.226s |
| `go test -v -run 'TestRunUpgrade_.*' ./internal/cli/...` (targeted) | 4 | 4 | 0 | 0 | 0.369s |

### Package breakdown (from `go test ./... -count=1`)

| Package | Result | Duration |
| --- | --- | --- |
| `internal/action` | ok | 3.855s |
| `internal/cli` | ok | 0.813s |
| `internal/config` | ok | 2.840s |
| `internal/scaffold` | ok | 0.966s |
| `internal/state` | ok | 1.319s |
| `internal/ui` | ok | 2.237s |
| `internal/ui/panes` | ok | 3.268s |
| `internal/upgrade` | ok | 1.474s |
| `internal/watcher` | ok | 4.121s |
| root, `cmd/ralph`, `cmd/ralph-tui` | no test files | — |

### Plan-targeted tests (all PASS)

| Test | Package | Status |
| --- | --- | --- |
| `TestComputeDiffs_Skip_PreservesHash` | `internal/upgrade` | PASS |
| `TestComputeDiffsWithManifest_PackPrefixedSubset` | `internal/upgrade` | PASS |
| `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` | `internal/upgrade` | PASS |
| `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers` | `internal/upgrade` | PASS |
| `TestRunUpgrade_SameVersionIsIdempotent` | `internal/cli` | PASS |
| `TestRunUpgrade_HealsCorruptedManifest` | `internal/cli` | PASS |
| `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` | `internal/cli` | PASS |

### Regression tests (pre-existing)

| Test | Package | Status |
| --- | --- | --- |
| `TestComputeDiffs_AutoUpdate` | `internal/upgrade` | PASS |
| `TestComputeDiffs_Conflict` | `internal/upgrade` | PASS |
| `TestComputeDiffs_AddNewFile` | `internal/upgrade` | PASS |
| `TestComputeDiffs_RemoveFile` | `internal/upgrade` | PASS |
| `TestRunUpgrade_AutoUpdate` | `internal/cli` | PASS |

## Coverage

- Statement (internal/upgrade): **80.9%**
- Statement (internal/cli): **31.1%**
- Branch: Not measured (Go tooling default does not report branch coverage).
- Function: Not measured separately (statement coverage is the proxy).
- Notes:
  - `internal/upgrade` is the main surface of this change and is well-covered at 80.9%. The four new diff tests exercise the three new/changed branches (ActionSkip with NewHash, empty-hash heal → skip, empty-hash conflict) plus pack-prefixed subset processing.
  - `internal/cli` reports 31.1%; this package is a CLI orchestration layer dominated by cobra wiring, stdout/stderr formatting, and interactive prompts. The new integration tests cover the targeted `runUpgrade` paths (idempotency, corrupted-manifest heal, pack diff failure preservation). Overall package coverage is limited by untested non-upgrade subcommands rather than gaps in this change.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Same-version `ralph upgrade` marking untouched base files as `modified locally` on the 2nd run (missing `NewHash` on `ActionSkip`) | Fixed | `TestRunUpgrade_SameVersionIsIdempotent` asserts no empty-hash entries remain in manifest after two upgrades; `TestComputeDiffs_Skip_PreservesHash` asserts `NewHash` is populated on skip |
| Pack files appearing simultaneously as `removed from template` and `new file` (monolithic `ComputeDiffs` saw pack-prefixed manifest keys vs. root-relative pack FS) | Fixed | `TestComputeDiffsWithManifest_PackPrefixedSubset` (no double-classification with subset manifest + pack FS) |
| Already-corrupted manifests (`hash = ""`) needing forced overwrite to recover | Fixed | `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` + `TestRunUpgrade_HealsCorruptedManifest` (heals in a single same-version upgrade, no write) |
| User-edited files with corrupted hash wrongly auto-healed | Preserved safety | `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers` keeps them as conflict (no silent overwrite) |
| Pack FS / diff failure wiping old manifest pack entries | Fixed | `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` (unknown pack triggers warning but old entries persist) |

## Acceptance criteria → test mapping

| AC | Covering test(s) | Status |
| --- | --- | --- |
| AC1: Same-version upgrade ×2 shows no modified/removed/new-file entries | `TestRunUpgrade_SameVersionIsIdempotent` | PASS |
| AC2: Manifest base entries never carry empty hash after upgrade | `TestRunUpgrade_SameVersionIsIdempotent`, `TestComputeDiffs_Skip_PreservesHash` | PASS |
| AC3: `hash = ''` + disk==template heals to ActionSkip without forced overwrite | `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate`, `TestRunUpgrade_HealsCorruptedManifest` | PASS |
| AC4: Pack file not classified as both `removed` and `new file` | `TestComputeDiffsWithManifest_PackPrefixedSubset` | PASS |
| AC5: Failed pack FS/diff preserves old manifest entries | `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` | PASS |
| AC6: Existing `TestComputeDiffs_AutoUpdate/_Conflict/_AddNewFile/_RemoveFile` still green | All four pass | PASS |
| AC7 (a): ActionSkip has non-empty `NewHash` | `TestComputeDiffs_Skip_PreservesHash` | PASS |
| AC7 (b): Namespaced manifest + pack FS → pack files not double-classified | `TestComputeDiffsWithManifest_PackPrefixedSubset` | PASS |
| AC7 (c): Empty-hash + disk match → ActionSkip with heal | `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` | PASS |
| AC7 (d): Pack diff failure → old entry preserved | `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` | PASS |
| AC8: `go test ./...` green (verify handles `run-verify.sh`) | Full `go test ./... -count=1` green | PASS |

Every acceptance criterion maps to at least one passing test.

## Skipped tests

| Test | Reason (from source) | Assessment |
| --- | --- | --- |
| `TestBaseFS_WithMockFS` | Placeholder skip in `internal/scaffold` (mock FS not wired) | Unrelated to this change; pre-existing |
| `TestAvailablePacks_WithMockFS` | Placeholder skip in `internal/scaffold` (mock FS not wired) | Unrelated to this change; pre-existing |

Both skips predate this branch and do not affect the upgrade fix.

## Test gaps

- `internal/cli` coverage at 31.1% reflects untested non-upgrade subcommands (`doctor`, `abort`, `retry`, parts of `pack`, `version`, etc.). These are outside the scope of this plan but represent long-term coverage debt for the CLI layer.
- No end-to-end test exercises the real `scaffold.PackFS` failure path — the integration test synthesizes failure by injecting an unknown pack name into `installedPacks`. This is sufficient because the production failure mode is exactly that (missing pack) and it triggers the same code path; a hypothetical "valid pack name with corrupted embed FS" case is not realistically reachable without patching embed state.
- Interactive prompt paths (stdin-attached confirmation for conflicts) are not exercised by tests; the plan scopes to non-interactive runs. Follow-up work could add a `*testing.T` harness over the prompt, tracked as future CLI coverage.
- No flaky tests observed in this run. Re-running the targeted suites twice produced identical results.

## Verdict

- Pass: yes
- Fail: no
- Blocked: no

**Tests are green. Safe to proceed to `/sync-docs` → `/codex-review` → `/pr` per the post-implementation pipeline.**
