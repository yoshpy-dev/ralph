# Verify report: fix-ralph-upgrade-manifest-hash-loss

- Date: 2026-04-21
- Plan: docs/plans/active/2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md
- Verifier: verifier subagent (Claude Code)
- Scope: Spec compliance (plan acceptance criteria) + static analysis on branch `fix/ralph-upgrade-manifest-hash-loss` vs `main`. Tests NOT executed as part of this step (handled by `/test`). However, `./scripts/run-verify.sh` internally runs `go test ./...` via the golang verifier, and that run is captured as supporting evidence.

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `git rev-parse --abbrev-ref HEAD` | PASS | `fix/ralph-upgrade-manifest-hash-loss` |
| `git diff main...HEAD --stat` | PASS | 5 files changed (plan + 2 impl + 2 test), +524/-16 |
| `go vet ./...` | PASS | No output (clean) |
| `gofmt -l internal/` | PASS | No output (no unformatted files) |
| `./scripts/run-verify.sh` | PASS | `EXIT=0`. All shell/hook syntax checks, settings.json jq parse, check-sync, mojibake tests, and the golang verifier (gofmt + go vet + go test) passed. Evidence: `docs/evidence/verify-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.log` |

## Spec compliance (Acceptance criteria walkthrough)

| # | Acceptance criterion | Status | Evidence |
| --- | --- | --- | --- |
| 1 | `ralph init` 直後の同一バージョン `ralph upgrade` 2 回連続で `modified/removed/new` 表示なし (内部 ActionSkip) | Verified | `internal/upgrade/diff.go:131-140` now sets `NewHash` on the unchanged branch; `internal/cli/upgrade.go:206-210` writes `d.NewHash` into the manifest on `ActionSkip`. Regression test `TestRunUpgrade_SameVersionIsIdempotent` (`internal/cli/cli_test.go:153-187`) executes init → upgrade → upgrade and asserts no empty-hash manifest entries. |
| 2 | upgrade 後の base エントリが空文字ハッシュを持たない | Verified | Same test, loop at `internal/cli/cli_test.go:175-179` checks `v.Hash == ""` for every entry. |
| 3 | `hash = ''` 破損マニフェストが 1 回の同一バージョン `runUpgrade` で非対話的に ActionSkip 扱いで復旧 | Verified | Heal branch in `internal/upgrade/diff.go:102-127` (disk == newHash → ActionSkip + NewHash; disk != newHash → ActionConflict). Tests: `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` (`internal/upgrade/diff_test.go:199-227`), `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers` (`internal/upgrade/diff_test.go:231-253`), and integration `TestRunUpgrade_HealsCorruptedManifest` (`internal/cli/cli_test.go:192-233`) which runs without `--force` and with closed stdin — any reach of the conflict prompt would flip skip to "skip" or surface an error, and the test asserts healed hashes. |
| 4 | pack ファイルが同一 upgrade 内で `removed` と `new file` に同時分類されない | Verified (by design + indirect test) | `splitManifestForBase` (`internal/cli/upgrade.go:50-61`) excludes `packs/languages/*` keys from the base sweep so they cannot be flagged removed. `splitManifestForPack` (`internal/cli/upgrade.go:65-76`) strips the namespace so pack FS keys match and produce ActionSkip instead of ActionAdd. `ComputeDiffsWithManifest(..., checkRemovals=false)` is used for pack calls. `TestComputeDiffsWithManifest_PackPrefixedSubset` (`internal/upgrade/diff_test.go:162-194`) asserts ActionSkip (not Add) and `TestRunUpgrade_SameVersionIsIdempotent` asserts the pack path `packs/languages/golang/README.md` exists exactly once and no unprefixed leak occurs. No test explicitly enumerates both-actions-absent, but the split-manifest construction makes co-occurrence unreachable. |
| 5 | 旧マニフェストの pack エントリが `scaffold.PackFS` / diff 失敗時にも新マニフェストで保持 | Verified | `preservePackEntries` helper (`internal/cli/upgrade.go:230-236`) is called on both `scaffold.PackFS` error and `ComputeDiffsWithManifest` error (`upgrade.go:116-128`). `maps.Copy(manifest.Files, preservedPackEntries)` at line 143 merges preserved entries into the new manifest. Integration test `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` (`internal/cli/cli_test.go:237-279`) exercises the path via a synthetic `ghostpack` in `Meta.Packs`. |
| 6 | 既存 `TestComputeDiffs_AutoUpdate/_Conflict/_AddNewFile/_RemoveFile` 緑 | Verified | `./scripts/run-verify.sh` golang verifier reported `ok internal/upgrade` and `ok internal/cli` with `go test ./...`. |
| 7 | 新規テスト (a)(b)(c)(d) 追加 | Verified | (a) `TestComputeDiffs_Skip_PreservesHash` (`diff_test.go:134-157`); (b) `TestComputeDiffsWithManifest_PackPrefixedSubset` (`diff_test.go:162-194`); (c) `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` (`diff_test.go:199-227`); (d) `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` (`cli_test.go:237-279`). Plus bonus `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers`, `TestRunUpgrade_SameVersionIsIdempotent`, `TestRunUpgrade_HealsCorruptedManifest`. |
| 8 | `go test ./...` 緑 / `./scripts/run-verify.sh` 緑 | Verified | `EXIT=0` from run-verify.sh, which includes `go test ./...` across all packages. Explicit `/test` run will re-validate in the next pipeline step. |

All 8 acceptance criteria are **Verified**.

### Test-name specificity check

All new test names clearly describe the behavior under verification:

- `TestComputeDiffs_Skip_PreservesHash` — skip path preserves NewHash.
- `TestComputeDiffsWithManifest_PackPrefixedSubset` — pack-subset manifest avoids Add misclassification.
- `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` — heal branch when disk matches.
- `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers` — heal branch conflict guard.
- `TestRunUpgrade_SameVersionIsIdempotent` — end-to-end idempotency.
- `TestRunUpgrade_HealsCorruptedManifest` — end-to-end heal without prompt.
- `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` — preservation on pack diff failure.

None rely on vague mechanics-only names. Passes the `.claude/rules/testing.md` specificity guidance.

## Observational checks

- `go vet ./...` — clean, no output.
- `gofmt -l internal/` — clean, no output.
- Static analysis run via `./scripts/run-verify.sh` completed with `EXIT=0`. Full log saved to `docs/evidence/verify-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.log`.
- Commit history is logically structured (3 commits: heal + hash preservation → pack scoping and preservation → prefix rename refactor). Small and focused.
- `splitManifestForBase` and `splitManifestForPack` preserve `Meta.Packs` via `out.Meta = m.Meta`, so nested pack lookups continue to work across the scoped subsets.

## Documentation drift

Plan step 8 explicitly defers doc updates to the `/sync-docs` stage. Flags (no edits required now):

- `docs/specs/2026-04-16-ralph-cli-tool.md`:
  - Lines 260–284 describe the `upgrade` flow but do not mention:
    - Same-version upgrade idempotency (no `modified/removed/new` output when nothing changed).
    - Heal behavior for empty-hash manifest entries (`hash = ''` repaired automatically).
    - Pack namespacing (`packs/languages/<pack>/…`) and separate base/pack diff scopes.
    - Preservation of old pack entries when a pack FS fails to load.
  - Recommendation: Add a short "Idempotency & heal" subsection under `### upgrade フロー` in `/sync-docs`.

- `docs/recipes/*`:
  - Only one `upgrade`-related hit (`docs/recipes/ralph-loop.md:84`: `migration | Backward-compatible migration steps | Upgrades`) — unrelated, it is a label-definition table. No recipe currently documents `ralph upgrade`; no drift to correct.

- `AGENTS.md` / `CLAUDE.md`: No behavioral contracts exposed in these map files are changed by the fix; no update needed.

Verdict on doc drift: **expected deferral** — flag passed to `/sync-docs`.

## Coverage gaps

- AC4 is satisfied structurally (split-manifest design eliminates the double-classification path) rather than by a direct assertion over a full diff slice. If stronger confidence is desired, the smallest useful additional check would be a single assertion in `TestRunUpgrade_SameVersionIsIdempotent` (or a dedicated test) that collects every diff action emitted during the second upgrade and verifies no path appears as both `ActionRemove` and `ActionAdd`. This is a nice-to-have; not a blocker.
- Static analysis did not include `staticcheck` or `golangci-lint`; the project relies on `go vet` + `gofmt`. Consistent with `.claude/rules/golang.md` and existing norms for this repo.

## Verdict

- Verified: AC1, AC2, AC3, AC4 (by design), AC5, AC6, AC7 (a–d), AC8; static analysis (`go vet`, `gofmt`, `run-verify.sh`); test-name specificity.
- Partially verified: AC4 has no direct diff-set assertion but the code path is unreachable by construction.
- Not verified: N/A.

**Overall verdict: PASS**

No blockers for `/test`. Documentation drift items are tracked for `/sync-docs` and do not gate this step.

## Artifacts

- Raw verification output: `docs/evidence/verify-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.log`
- This report: `docs/reports/verify-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md`
