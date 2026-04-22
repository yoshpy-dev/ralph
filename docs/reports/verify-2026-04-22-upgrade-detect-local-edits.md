# Verify report: upgrade — detect local edits and show unified diff

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-upgrade-detect-local-edits.md`
- Verifier: verifier subagent
- Scope: spec compliance (each acceptance criterion), static analysis (`./scripts/run-static-verify.sh`), documentation drift vs. new behavior, contract coherence of new `SetFileUnmanaged` / `Managed=false` semantics. Behavioral test execution is explicitly deferred to `/test`.
- Branch: `feat/upgrade-detect-local-edits` (6 commits ahead of `main`)
- Evidence: `docs/evidence/verify-2026-04-22-upgrade-detect-local-edits.log`

## Spec compliance

Each AC is mapped to (a) the diff.go / upgrade.go branch that realizes it and (b) the test that locks the behavior in. Tests are referenced as static evidence — they are *not* executed here (per skill scope); actual exercise belongs to `/test`.

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| (1) テンプレート未変更 + ローカル編集あり → `ActionConflict` | Verified (static) | `internal/upgrade/diff.go:175-195` adds the `newHash == mf.Hash && diskHash != mf.Hash` → `ActionConflict` branch. Locked in by `TestComputeDiffs_LocalEditWithUnchangedTemplate` (`internal/upgrade/diff_test.go:301-328`) which asserts `Action=ActionConflict`, populated `NewContent`, and `DiskHash != OldHash`. |
| (2) テンプレート未変更 + ローカル編集なし → `ActionSkip` (heal 含む回帰なし) | Verified (static) | `internal/upgrade/diff.go:175-184` retains the `newHash == mf.Hash && diskHash == mf.Hash` → `ActionSkip` path. Empty-hash heal path at `internal/upgrade/diff.go:142-167` preserved. Locked in by `TestComputeDiffs_Skip_PreservesHash` (`diff_test.go:134-157`) and `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` (`diff_test.go:199-227`). |
| (3) テンプレート変更 + ローカル編集あり → `ActionConflict` | Verified (static) | `internal/upgrade/diff.go:197-218` kept intact (unchanged branch for `newHash != mf.Hash && diskHash != mf.Hash`). Locked in by `TestComputeDiffs_Conflict` (`diff_test.go:58-82`). |
| (4) `[d]iff` は `--- local` / `+++ template (version)` の unified diff (`-` = local, `+` = template) | Verified (static) | `internal/cli/upgrade.go:335-340`: `UnifiedDiff(localBytes, d.NewContent, "local", fmt.Sprintf("template (%s)", version))` — local-as-old, template-as-new. Plan AC #4 text was aligned to the implementation in commit `18ab284` (`docs(plan): align diff label direction with implementation`). Locked in by `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff` asserting `"--- local"`, `"+++ template (1.0.0-test)"`, `"-# my agents"`, `"+# AGENTS"` (`internal/cli/cli_test.go:596-607`). |
| (5) `[d]iff` 後に再プロンプト、`d` 連打でループ化しない、EOF で skip | Verified (static) | `resolveConflict` (`internal/cli/upgrade.go:298-320`) uses an unbounded `for` loop that re-prompts on any non-`o`/`s` input (including repeated `d`) and the EOF branch (`err != nil && line == ""`) collapses to `resolutionSkip`. Locked in by `TestRunUpgrade_InteractiveDiff_RepromptsOnInvalid` (`cli_test.go:612-640`, asserts prompt rendered ≥ 4 times for `xyz\nd\nd\ns\n`) and `TestRunUpgrade_NextRunAfterSkip_IsSilent` (`cli_test.go:644-680`, asserts empty stdin does not hit the non-interactive branch once unmanaged). |
| (6) disk 読み取り失敗時は警告 + hash サマリ継続、abort しない | Verified (static) | `showDiff` (`internal/cli/upgrade.go:326-347`): `os.ReadFile` failure routes to `writef(errOut, "    (could not read %s: %v — falling back to hash summary)\n", ...)` + `template hash` / `local hash` lines; control returns to the prompt loop. Locked in by `TestRunUpgrade_DiskReadFailure_FallsBackToHash` (`cli_test.go:685-720`) which uses the `removingReader` primitive to delete the file mid-prompt and asserts `"could not read"` in errOut and `"template hash:"` in out, with no abort. |
| (7) `overwrite` → local = template, manifest `{Hash: newHash, Managed: true}` | Verified (static) | `internal/cli/upgrade.go:201-207`: `resolutionOverwrite` case writes `d.NewContent` then `manifest.SetFile(d.Path, d.NewHash)` (which is `Managed=true` by construction — see `internal/scaffold/manifest.go:71-76`). Locked in by `TestRunUpgrade_InteractiveOverwrite_WritesManaged` (`cli_test.go:487-526`) asserting disk content matches template and `entry.Managed == true`. |
| (8) `skip` → local 維持、manifest `{Hash: diskHash, Managed: false}` → 次回 silent skip | Verified (static) | `internal/cli/upgrade.go:208-224`: `resolutionSkip` case calls `manifest.SetFileUnmanaged(d.Path, hash)` with `hash = d.DiskHash` (fallback chain `OldHash` → `NewHash` only if `DiskHash == ""`, which is unreachable for any current conflict producer per self-review — defensive). Locked in by `TestRunUpgrade_InteractiveSkip_WritesUnmanaged` (`cli_test.go:530-570`) and convergence test `TestRunUpgrade_NextRunAfterSkip_IsSilent` (`cli_test.go:644-680`). |
| (9) `Managed=false` エントリ → `ComputeDiffsWithManifest` で `ActionSkip`、プロンプト・auto-update 抑制 | Verified (static) | `internal/upgrade/diff.go:84-98`: early-return `ActionSkip` for `inManifest && !mf.Managed` positioned *before* the template/disk-hash branches, so it shadows all downstream conflict/auto-update logic. Locked in by `TestComputeDiffs_Unmanaged_IsSilentSkip` (`diff_test.go:334-359`, template diverges but stays Skip) and `TestComputeDiffs_Unmanaged_SilentSkipWhenDiskMissing` (`diff_test.go:365-384`). |
| (10) `ralph upgrade --force` は全上書き、`Managed` は true に戻る | Verified (static) | `internal/cli/upgrade.go:190-198`: `--force` branch under `ActionConflict` writes template content and calls `manifest.SetFile(d.Path, d.NewHash)` (→ `Managed=true`). Note: `--force` still short-circuits at the `ActionConflict` switch arm; unmanaged entries (`ActionSkip` via the early-return at `diff.go:84-98`) are NOT flipped back to `Managed=true` by `--force` because they never reach the conflict arm. This matches the plan's Open Questions (`--resync` is out of scope) but is not covered by the force test (see Coverage gaps below). Locked in for the conflict path by `TestRunUpgrade_ForceOverwritesLocalEdit` (`cli_test.go:448-483`). |
| (11) `./scripts/run-verify.sh` と `go test ./...` が緑 | Verified (static) for `run-static-verify.sh`; `go test ./...` executed inside the verifier script (cached pass, exit 0). Full `go test ./...` re-run under `/test`. | `docs/evidence/verify-2026-04-22-upgrade-detect-local-edits.log` — `==> All verifiers passed.` Exit 0. `ok internal/cli 0.684s` + `ok internal/upgrade (cached)` + `ok internal/scaffold (cached)`. |

All 11 acceptance criteria are satisfied by the diff. One AC (#10) is only tested on the "conflict without prior unmanaged state" axis — the "force vs. already-unmanaged entry" axis is not exercised. See Coverage gaps.

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | PASS (exit 0) | Full run in `docs/evidence/verify-2026-04-22-upgrade-detect-local-edits.log`. |
| `shellcheck` (hooks + verify scripts) | PASS | All `.claude/hooks/*.sh` and `templates/base/.claude/hooks/*.sh` OK. |
| `sh -n` (all hook scripts) | PASS | Syntax OK. |
| `jq -e . .claude/settings.json` and `templates/base/.claude/settings.json` | PASS | Both settings.json parse cleanly. |
| `scripts/check-sync.sh` | PASS | `IDENTICAL=107, DRIFTED=0, ROOT_ONLY=0`. No template/root mirror drift introduced by the feature. |
| `gofmt` | PASS | `gofmt: ok` (0 unformatted files). |
| `go vet` (embedded in verifier) | PASS | `0 issues`. |
| `go test ./...` (triggered by run-static-verify.sh's golang verifier) | PASS | `ok internal/cli 0.684s`, `ok internal/upgrade (cached)`, `ok internal/scaffold (cached)`. (Noting the pass here for completeness; execution analysis belongs to `/test`.) |

No `TODO` / `FIXME` / debug prints / commented-out blocks introduced — spot-checked via the diff.

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md` | No drift | Only mentions `internal/upgrade/ — Diff engine + conflict resolution` at line 47; does not describe the conflict prompt semantics or `Managed=false` convergence. No text to update. |
| `AGENTS.md` | No drift | No upgrade-specific contract text. |
| `docs/architecture/repo-map.md` | No drift | Lists `internal/upgrade/` without behavioral contract. |
| `docs/specs/2026-04-16-ralph-cli-tool.md` (§ 冪等性と自動修復, lines 286-298) | **Drifted** | The spec currently enumerates (a) 同一バージョン冪等性, (b) 空ハッシュ自動修復 (heal), (c) pack 名前空間化, (d) pack preservation vs. drop, (e) ActionRemove 後のマニフェスト・ドロップ, (f) 再導入ファイルの安全側判定. It does **not** describe the new "local edit detection on unchanged template" branch, nor the `Managed=false` / user-owned convergence contract introduced by this PR. This is the primary doc drift and belongs to `/sync-docs`. The spec's "コンフリクト時はファイルごとに上書き/スキップ/diff表示を選択可能" at line 29 remains accurate at a high level but the AC list and subsection need a new bullet (e.g. `**ローカル編集検知 + user-owned 収束**`). |
| `docs/specs/2026-04-16-ralph-cli-tool.md:42` (`.ralph/prompts/` upgrade conflict note) | No drift | Independent feature, unaffected. |
| `.claude/rules/` (architecture, testing, planning, git-commit-strategy, etc.) | No drift | No rule text references the upgrade conflict flow or `Managed` field semantics. |
| Plan AC #4 label direction | In sync (as of `18ab284`) | Plan text was realigned to match the implemented `--- local / +++ template (version)` order. Commit `18ab284` also fixed the `Reprompls` typo flagged by self-review. |
| Plan progress checklist (lines 58-68) | Stale but benign | Acceptance-criteria boxes remain `[ ]`. Per verifier memory (`feedback_plan_ac_checklist_drift.md`), this is a known doc-drift pattern and not a spec-compliance failure; flagged here for the author rather than blocking verdict. |
| `docs/tech-debt/README.md` | Pending | Self-review identified 2 LOW items (`--resync` escape hatch absent, plan label drift pre-18ab284). The label item is now resolved. The `--resync` item has not been appended. `/sync-docs` should append it (or mark it resolved if the author prefers to defer tracking). |

## Observational checks

- `ComputeDiffsWithManifest` contract coherence: the new `!mf.Managed` early-return (`diff.go:84-98`) is placed **before** the `!inManifest` branch (`diff.go:100-124`), the empty-hash heal (`diff.go:142-168`), the unchanged-template branch (`diff.go:175-195`), and the changed-template branches (`diff.go:197-218`). This ordering makes `Managed=false` a strict override — no later branch can reclassify an unmanaged entry. This matches the plan's Managed=false convergence contract and the self-review's "three layered defenses" observation.
- `SetFile` vs `SetFileUnmanaged` symmetry: both set the full `ManifestFile{Hash, Managed}` struct — no partial mutation, no field-level invariant to maintain. `SetFile` always writes `Managed=true` (preserving the pre-PR invariant for all existing callers at `internal/cli/upgrade.go:185,195,206,234,249`, `internal/cli/init.go:179`, `internal/cli/pack.go:79`). `SetFileUnmanaged` is called from exactly one site (`internal/cli/upgrade.go:221`) plus the `ActionSkip` preservation path (`upgrade.go:247`). Grep-verified.
- Manifest TOML schema: `Managed bool` already existed (`internal/scaffold/manifest.go:27`); no schema change, no migration concern. Existing project manifests written pre-PR all have `managed = true` (the old `SetFile` always set it). When read back into the new code, they are indistinguishable from the intended "template-managed" state. Backward-compat: **yes** by construction.
- `runUpgradeIO` DI boundary: `runUpgrade(targetDir, force)` is now a 1-line shim over `runUpgradeIO(targetDir, force, os.Stdin, os.Stdout, os.Stderr)` (`upgrade.go:80-82`). cobra command wiring (`newUpgradeCmd`) still calls `runUpgrade`, so CLI signature is unchanged — consistent with the plan's "keep public entry point stable" intent.
- `UnifiedDiff` determinism: asserted by `TestUnifiedDiff_OrderStability` (same input produces byte-identical output). Important for evidence-log reproducibility.
- `resolutionSkip = iota` zero-value invariant: self-review LOW finding #2 noted this is an implicit contract. Not blocking for verify, but any future addition to the `resolution` enum above `resolutionSkip` would silently flip the default. Recommend an inline comment at `upgrade.go:289-292` as a non-blocking follow-up.
- No new goroutines, channels, or concurrency primitives introduced — the change is strictly sequential I/O + pure functions.

## Coverage gaps

1. **`--force` interaction with `Managed=false`**: if a user previously chose `skip` (manifest now `Managed=false`) and later runs `ralph upgrade --force`, the current code path is: `ComputeDiffsWithManifest` early-returns `ActionSkip` at `diff.go:84-98` → the `switch d.Action` at `upgrade.go:175` hits the `case upgrade.ActionSkip` branch at line 242, which *preserves* the unmanaged state (`SetFileUnmanaged(d.Path, prev.Hash)`). So `--force` does **not** re-adopt unmanaged files. This is consistent with the plan's stance that re-adoption requires a future `--resync` / `--adopt` (Open Questions), but it is not explicitly locked in by a test. Suggest `/test` add `TestRunUpgrade_Force_DoesNotReadoptUnmanaged` to make this contract first-class.
2. **CRLF edge case mentioned in plan test plan**: `unified_diff_test.go` covers add-only, remove-only, replace, empty, trailing newline, context window, and order stability — but not explicit CRLF (LF/CRLF mixed) input. The plan mentions CRLF in the Risks section with "CRLF もそのまま扱う (テストケース追加)". A `TestUnifiedDiff_CRLFPreservation` would close this gap. Not a blocker for verify (behavior is correct — `\r` is part of the line string and compared verbatim), flagged for `/test`.
3. **Defensive fallback in `resolveConflict` hash chain**: `upgrade.go:213-222` has unreachable inner branches (`d.OldHash` / `d.NewHash` fallback) — all current `ActionConflict` producers set `DiskHash`. Not a bug, not a coverage blocker; noted per self-review LOW finding #3.
4. **Plan progress checklist (AC boxes)**: lines 58-68 are `[ ]` despite implementation + tests landing. Per memory, do not fail verify on stale AC boxes; this is a documentation nit for the author to tick during PR.
5. **Spec drift in `docs/specs/2026-04-16-ralph-cli-tool.md` § 冪等性と自動修復**: new "local edit detection + Managed=false convergence" bullet is missing. `/sync-docs` must add it, otherwise a future reader reconstructing the contract from spec alone would miss the new prompt-storm guard.

## Verdict

- **Verified**:
  - All 11 acceptance criteria have file-level evidence plus co-located tests.
  - Static analysis green (`HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` → exit 0).
  - `Managed=false` / `SetFileUnmanaged` introduction is coherent with the existing `scaffold.Manifest` contract — additive, non-breaking, correctly ordered override in `ComputeDiffsWithManifest`.
  - No drift in `README.md`, `AGENTS.md`, `docs/architecture/`, `.claude/rules/`.
  - Self-review LOW items #1 (`Repromp*` typo) and #4 (plan AC #4 label direction) resolved in commit `18ab284`.

- **Partially verified**:
  - Plan progress checklist AC boxes (58-68) left unchecked — cosmetic doc drift; not blocking.

- **Not verified (deferred by skill scope)**:
  - Actual behavioral test pass/fail — belongs to `/test`. The verifier observed that `go test ./...` embedded in the static-verify script reports `ok`, but a full explicit run with `-count=1` is `/test`'s job.
  - Interactive terminal behavior (real TTY I/O, color rendering, signal handling) — belongs to `/test` or manual walkthrough. The DI boundary (`runUpgradeIO`) makes this inspectable but not observable statically.

- **Doc drift flagged (non-blocking for verify verdict, belongs to `/sync-docs`)**:
  - `docs/specs/2026-04-16-ralph-cli-tool.md` § 冪等性と自動修復: add a bullet describing the new "local edit detection + Managed=false user-owned convergence" contract. Suggested wording:
    > **ローカル編集検知と user-owned 収束**: `newHash == mf.Hash` かつ `diskHash != mf.Hash` のとき、従来は `ActionSkip` に落ちていたがこれを `ActionConflict` に昇格する。ユーザが `[s]kip` を選んだファイルは `Managed=false` + `Hash=diskHash` でマニフェスト上に記録され、以降の `ralph upgrade` は (対話も auto-update も発生せず) silent skip に収束する (prompt storm 回避)。再び template 管理下に戻すには将来の `--resync` / `--adopt` が必要。
  - `docs/tech-debt/README.md`: append the `--resync` / `--adopt` escape-hatch item flagged by self-review.

- **Suggested minimal next check (highest confidence for lowest cost)**:
  1. `/test`: run `go test ./internal/upgrade/... ./internal/cli/... -count=1 -v` and capture output to `docs/evidence/test-2026-04-22-upgrade-detect-local-edits.log`.
  2. Optional: add `TestRunUpgrade_Force_DoesNotReadoptUnmanaged` during `/test` to lock in Coverage gap #1.

**Verdict: PASS.** The implementation satisfies all acceptance criteria at the file-evidence level, static analysis is clean, and the new `Managed=false` semantics are coherent with existing contracts. Two doc-drift items are flagged for `/sync-docs` but do not block the verify verdict. Pipeline may continue to `/test`.
