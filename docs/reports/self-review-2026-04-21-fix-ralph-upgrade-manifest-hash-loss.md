# Review report: fix/ralph-upgrade-manifest-hash-loss

- Date: 2026-04-21
- Plan: `docs/plans/active/2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md`
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: Two commits on `fix/ralph-upgrade-manifest-hash-loss` (`git log main..HEAD`):
  - `6293000 fix(upgrade): preserve hash on skip and heal empty-hash entries`
  - `d473bf8 fix(upgrade): scope manifest by base/pack and preserve dropped packs`

## Evidence reviewed

- `git diff main...HEAD --stat` — 5 files changed, +524 / -16
- `internal/upgrade/diff.go` (full file + diff)
- `internal/cli/upgrade.go` (full file + diff)
- `internal/upgrade/diff_test.go` (diff)
- `internal/cli/cli_test.go` (diff)
- Cross-check: `internal/cli/init.go:158-161` (manifest key format), `internal/scaffold/manifest.go` (Manifest / ManifestFile shape)
- Call-site sweep: `ComputeDiffs`, `ComputeDiffsNoRemovals`, `ComputeDiffsWithManifest`

Review is diff-quality only. Spec compliance, doc drift, and test-execution validation are deferred to `/verify` and `/test`.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | `internal/cli/upgrade.go:45` | Constant `basePrefix` is misnamed. Its value and docstring describe "the namespace prefix reserved for language pack entries" and every use-site treats keys that match it as NOT being base entries (they are excluded from `splitManifestForBase`, included in `splitManifestForPack`, and identify pack entries in `preservePackEntries`). The identifier is actively misleading for anyone grepping/reading the file. | `internal/cli/upgrade.go:42-45` (doc/value mismatch with name), `:55` (used as "is pack entry" predicate), `:232` (used in `preservePackEntries` as the pack-prefix test — same semantic). | Rename to `packNamespacePrefix` (or `packPrefix`, but that collides with the local variable). Also consider having `packPrefixFor` reuse it: `return basePrefix + pack + "/"` → `return packNamespacePrefix + pack + "/"` to eliminate the two independent string-construction sites. |
| LOW | `internal/cli/upgrade.go:50-61, 65-76` | `splitManifestForBase` / `splitManifestForPack` call `scaffold.NewManifest(m.Meta.Version)` only to immediately overwrite both `out.Meta` and `out.Files`. The constructor's work (setting Created/Updated timestamps, initializing an empty Files map) is wasted. Minor readability cost: a reader has to follow three assignments to see what the final state is. | `internal/cli/upgrade.go:51-53` and `67-69`: `NewManifest(...)` → `out.Meta = m.Meta` → `out.Files = make(...)`. | Replace with a literal: `out := &scaffold.Manifest{Meta: m.Meta, Files: make(map[string]scaffold.ManifestFile, len(m.Files))}`. Single expression, no wasted allocations, same behavior. |
| LOW | `internal/cli/upgrade.go:119, 127` | Both failure branches (PackFS error and diff error) call `preservePackEntries(oldManifest, prefix, preservedPackEntries)` but only the diff-error warning is tagged with "diff failed". The PackFS warning reads `Warning: pack %s: %v` — ambiguous for a reader of stderr output. | `internal/cli/upgrade.go:117-120` vs `:125-128`. | Unify warning wording, e.g. `"Warning: pack %s load failed: %v"` vs `"Warning: pack %s diff failed: %v"`, so operators can distinguish the two failure modes in logs. |
| LOW | `internal/cli/upgrade.go:130-132` | Old code used a `packPrefix := filepath.Join("packs", "languages", pack)` local; the new code inlines `filepath.Join("packs", "languages", pack, packDiffs[i].Path)` inside the loop. Functionally identical, but now there are three separate string-construction sites for the same pack path concept (`packPrefixFor` — manifest key prefix; `filepath.Join("packs","languages",pack)` — disk path + namespaced diff path; `packDir` at line 122). Consolidating them would improve grep-ability. | `internal/cli/upgrade.go:38-40, 122, 131`. | Introduce a helper like `packRelDir(pack) = filepath.Join("packs", "languages", pack)` used at 122 and 131, and keep `packPrefixFor` for the slash-normalised manifest-key prefix. |
| LOW | `internal/cli/cli_test.go:216` (test name) | `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` exercises the `scaffold.PackFS()` failure branch (ghostpack is not in the embedded FS), not the post-PackFS `ComputeDiffsWithManifest` failure branch. The test name promises "DiffFailure" but there is no test case for an actual diff computation returning an error. | `internal/cli/cli_test.go:216-276`: "pack that used to be installed but whose FS no longer loads" — this is the PackFS-load branch at `upgrade.go:117-120`, not the diff-failure branch at `:125-128`. The diff-failure branch (`:125-128`) is therefore uncovered. | Either (a) rename to `TestRunUpgrade_PreservesOldPackEntriesOnPackFSFailure` and add a second test that forces `ComputeDiffsWithManifest` to fail, or (b) keep the broad name and flag the diff-failure branch explicitly for `/test` to verify coverage. |
| LOW | `internal/upgrade/diff.go` | `ComputeDiffsNoRemovals` is retained as a public API but has no remaining production caller (`upgrade.go` now uses `ComputeDiffsWithManifest` directly). Kept intentionally per the plan's Open questions section, but without a `// Deprecated:` comment a future reader cannot tell it is a no-longer-used compatibility shim. | `internal/upgrade/diff.go:41-50` (wrapper), `Grep ComputeDiffsNoRemovals` → only definition + plan + old codex triage report reference it. | Either add a `// Deprecated: use ComputeDiffsWithManifest.` comment or track as tech debt. The plan explicitly chose to defer removal; a deprecation marker clarifies that choice in-code. |
| LOW | `internal/cli/upgrade.go:102-106` | Line 102 (`baseManifest := splitManifestForBase(oldManifest)`) is immediately followed by `ComputeDiffsWithManifest(baseManifest, ...)`. If the base diff fails, the per-pack loop never runs, but the warning message ("computing diffs") does not distinguish base from pack failure. Pre-existing wording, but now one of three similar sites. | `internal/cli/upgrade.go:105` vs `:118, :126`. | Consider `"computing base diffs: %w"` for clarity. Low priority; purely cosmetic. |

## Positive notes

- Heal logic is conservative: the empty-hash branch only produces `ActionSkip` (no disk write) when disk matches template, and falls through to `ActionConflict` otherwise. No silent overwrite of user edits, which matches the stated risk mitigation (plan R2).
- The new public API (`ComputeDiffsWithManifest`) is named grep-ably and documented with the intended use (scoped/stripped manifest subsets). The existing two wrappers preserve the public signature — zero external-caller churn.
- `splitManifestForBase` / `splitManifestForPack` / `preservePackEntries` all normalize keys via `filepath.ToSlash` before prefix-checking, which is defensive against Windows-style manifest keys that `init.go` could write (pre-existing `filepath.Join`-based keys).
- `preservedPackEntries` is applied via `maps.Copy(manifest.Files, preservedPackEntries)` *before* the diff loop runs. Correct ordering — if a newly-computed entry later collides (same key), the fresh diff wins.
- No debug print / TODO / commented-out code / hardcoded secrets in the diff.
- `ActionSkip` now carries `OldHash` / `DiskHash` / `NewHash` consistently with other actions (previously only `Path` / `Action`) — reduces ad-hoc zero-value reading in callers.
- Tests cover the four documented branches (skip-preserves-hash, pack-prefixed-subset, heal-when-disk-matches, empty-hash-conflict-when-disk-differs) plus the two integration scenarios (idempotent upgrade, corrupted-manifest heal). Plus preservation-on-failure. Test names describe intent, not mechanics.

## Coverage gaps

These are flagged for `/verify` and `/test` to confirm, not for this review to adjudicate:

- The `ComputeDiffsWithManifest` failure branch inside `upgrade.go:125-128` has no direct unit or integration test. `fs.WalkDir` + `fs.ReadFile` + `scaffold.HashFile` all have to fail for the error to propagate, which is difficult to provoke from an embedded FS.
- `TestComputeDiffs_Skip_PreservesHash` asserts `diffs[0].NewHash` but does not assert that `OldHash` / `DiskHash` are also set on the `ActionSkip` path. Missing assertion, not a bug.
- `TestRunUpgrade_HealsCorruptedManifest` only zeroes `AGENTS.md` / `CLAUDE.md` hashes; it does not exercise the "empty hash + disk differs → ActionConflict" branch end-to-end through `runUpgrade`. The unit test covers the conflict branch, but the CLI-level interactive path (stdin closed → "skip") is not integration-tested for that specific cause.
- The dead-public `ComputeDiffsNoRemovals` still has a removal-semantics contract. No test enforces that contract now that the function has no production caller; deletion would be silent.

## Recommendation

- Merge: **approve** (no CRITICAL or HIGH findings). All findings are LOW/MEDIUM code-hygiene items that do not affect correctness or security.
- Follow-ups:
  1. Rename `basePrefix` → `packNamespacePrefix` (MEDIUM; readability only, no behavior change). Strongly recommended but not blocking.
  2. Inline the `NewManifest` calls in `splitManifestForBase` / `splitManifestForPack` into struct literals.
  3. Disambiguate the two pack-warning strings.
  4. Either rename `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` or add a true diff-failure case.
  5. Mark `ComputeDiffsNoRemovals` with a `// Deprecated:` comment (or add a tech-debt entry documenting the planned removal per plan's Open questions).

No tech-debt file addition is required for this review — all deferred items are small hygiene fixes that fit into follow-up commits rather than accumulated complexity.

## Round 2 (post-codex)

- Date: 2026-04-21
- Trigger: Re-review after two Codex ACTION_REQUIRED fixes landed in commit `d16cb4d`.
- Scope: `git show d16cb4d` only. Prior commits (`6293000`, `d473bf8`, `b01861f`, `af16b7e`, `9f5ccc8`, `f7ab8bf`) were approved in the earlier round.
- Reviewer: reviewer subagent (self-review, diff quality only).

### Evidence reviewed

- `git show d16cb4d --stat`: 3 files, +118 / -41
  - `internal/cli/upgrade.go` (+37 / −17, net +20)
  - `internal/cli/cli_test.go` (+59 / −17, net +42)
  - `docs/reports/codex-triage-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md` (triage rewrite — out of code-quality scope)
- Full re-read of `internal/cli/upgrade.go` (current HEAD).
- Re-read of `internal/upgrade/diff.go:170-183` (removal loop, to confirm re-prefix interaction).
- `scaffold.AvailablePacks()` at `internal/scaffold/embed.go:35-47`.
- `setupTestEmbedFS` at `internal/cli/cli_test.go:12-25` (to confirm the test fixture genuinely lacks `deprecated.sh` and `ghostpack`).

### What the commit does

1. Pack-sweep removal detection: flips `ComputeDiffsWithManifest(packManifest, packDir, packFS, false)` → `true`. A tracked-but-missing pack file now surfaces as `ActionRemove`, then gets re-prefixed to `packs/languages/<pack>/<file>` before hitting the switch. The `ActionRemove` branch (`upgrade.go:225-229`) preserves `OldHash` so next-upgrade is idempotent.
2. Disappeared-pack handling: adds an `AvailablePacks()` pre-check. Packs absent from current templates are dropped (not preserved) — both from the per-pack diff loop and from the new `retainedPacks` slice that replaces the blanket `manifest.Meta.Packs = installedPacks` assignment.
3. Warning text disambiguation: `"Warning: pack %s: %v"` → `"Warning: pack %s load failed: %v (preserving manifest entries)"` and `"Warning: pack %s diff failed: %v (preserving manifest entries)"`. Resolves the earlier round's LOW finding on ambiguous warning wording.
4. Tests: renames `TestRunUpgrade_PreservesOldPackEntriesOnDiffFailure` → `TestRunUpgrade_DropsPacksRemovedFromTemplates` to match the new contract; adds `TestRunUpgrade_ReportsDeletedPackFile` exercising the re-prefixed `ActionRemove` path.

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | `internal/cli/cli_test.go:297-321` — `TestRunUpgrade_ReportsDeletedPackFile` | The test asserts the deprecated entry is **retained** via `OldHash` preservation (idempotency on re-run), but it does not also assert the user actually saw the "removed from template" notice on stderr. The whole point of the fix is the user-facing signal; if a future refactor silently suppressed the `fmt.Printf("  ⚠ %s (removed from template — review and delete manually)\n", d.Path)` branch, this test would still pass because the manifest preservation behavior is independent of the print. | `internal/cli/cli_test.go:315-320` — only manifest-entry existence is asserted. `upgrade.go:228` emits the notice but to stdout, not stderr (`fmt.Printf` → stdout). The test doesn't capture stdout. | Capture `os.Stdout` during `runUpgrade` and assert the output contains the pack-prefixed path `packs/languages/golang/deprecated.sh`. Alternatively, flag for `/test` to confirm coverage. Not blocking. |
| LOW | `internal/cli/cli_test.go:256-296` — `TestRunUpgrade_DropsPacksRemovedFromTemplates` | The test asserts ghostpack is **removed** from `m2.Meta.Packs` but does not assert golang is **retained** in `m2.Meta.Packs`. If a regression accidentally emptied `retainedPacks` (e.g. someone moved the `append` back into an unreachable branch), this test would still pass. | `internal/cli/cli_test.go:289-293` — `for _, p := range m2.Meta.Packs { if p == "ghostpack" { ... } }` catches ghostpack's presence but an empty slice also satisfies the loop. | Add `if !slices.Contains(m2.Meta.Packs, "golang") { t.Error("golang pack dropped from Meta.Packs") }`. Not blocking. |
| LOW | `internal/cli/upgrade.go:129-131` | Message `"Notice: pack %q no longer exists in templates — manifest tracking dropped (files on disk left untouched)"` uses `%q` for the pack name while the adjacent warnings at lines 136 / 149 use `%s`. Minor inconsistency in stderr formatting. | `upgrade.go:130, 136, 149`. | Harmonize on `%q` (quoting the pack name reads better across all three messages). Cosmetic only. |
| LOW | `internal/cli/upgrade.go:121, 168` | `preservedPackEntries` map + `maps.Copy(manifest.Files, preservedPackEntries)` is now dead in practice for the common "pack disappeared" case (the most-likely reason `PackFS` would fail), because the `AvailablePacks` pre-check intercepts those packs before they reach the preservation path. The path is only live for true transient errors (disk-IO failure on `fs.Sub`, `ReadDir` error, etc.). Still correct, but the comment at `:118-120` should reflect that preservation is now scoped to genuinely transient `PackFS` / `ComputeDiffsWithManifest` failures — not to the disappeared-pack case. | `upgrade.go:118-120` comment: *"a transient error does not permanently drop their tracking. Packs that have been removed from the template release are explicitly NOT preserved."* — this now reads accurately; the prior round's concern was resolved. Noting here only to confirm the comment matches the code. | No change needed. Logged for verification. |
| LOW | `internal/cli/upgrade.go:113` | `available := make(map[string]bool, len(availablePacks))` uses `map[string]bool` for a set membership check. `map[string]struct{}{}` is the idiomatic Go zero-byte-value set; `map[string]bool` is fine but marginally wastes 1 byte per entry. Given the pack count is tiny (N<10 in practice), the perf difference is literally zero. Purely stylistic. | `upgrade.go:113-116`. | Either leave as-is (readable) or switch to `struct{}{}`. Non-blocking. |

No CRITICAL, HIGH, or MEDIUM findings.

### Cross-check against Round 1 follow-ups

The prior round flagged five LOW / one MEDIUM items. Commit `d16cb4d` resolves some and introduces no regressions:

- **MEDIUM `basePrefix` misnaming** — already resolved in `b01861f` (outside this commit). Current code uses `packNamespacePrefix`. ✓
- **LOW `splitManifestFor*` wasted `NewManifest` allocation** — not addressed (still at `upgrade.go:51-53, 67-69`). Still LOW, still non-blocking. Not in scope for this Round 2 commit.
- **LOW ambiguous warning wording** — resolved by `d16cb4d` (warnings now say "load failed" vs "diff failed"). ✓
- **LOW three pack-path construction sites** — unchanged. Still LOW. Out of scope for Round 2.
- **LOW test-name mismatch (`PreservesOldPackEntriesOnDiffFailure` vs actual PackFS branch)** — superseded: the test was replaced, not renamed. The old concern is moot. The new test (`DropsPacksRemovedFromTemplates`) exercises the `AvailablePacks` pre-check path; the PackFS-failure branch at `upgrade.go:134-140` and the diff-failure branch at `:147-153` are both now **uncovered** by direct tests (both require provoking a transient-error scenario not present in the embedded-FS fixtures). Coverage gap to flag for `/test`. ✓ (partial — gap shifted, not closed)
- **LOW `ComputeDiffsNoRemovals` dead-public shim** — unchanged. Still LOW. Out of scope.

### Correctness spot-checks

- **Re-prefix safety for `ActionRemove`**: walk in `ComputeDiffsWithManifest` emits `ActionRemove` with `Path = "deprecated.sh"` (stripped key). Line 155 re-prefixes to `"packs/languages/golang/deprecated.sh"`. Switch at line 225 calls `manifest.SetFile(fullPath, d.OldHash)`. No collision with a possible `ActionAdd` at the same stripped path, because `ActionAdd` only fires when the file is in `newFS` and not in `packManifest.Files`; `ActionRemove` only fires when the file is in `packManifest.Files` and not in `newFS`. Mutually exclusive on `path`. ✓
- **`retainedPacks` ordering**: maintains insertion order of `installedPacks`. No sorting regression. ✓
- **`available` map nil-safety**: `AvailablePacks` returns `nil, err` on failure; the `err` branch returns early, so `availablePacks` is never iterated when nil. ✓
- **`available[pack]`**: safe read on an initialized (possibly empty) map. Missing key yields `false`. ✓
- **No new debug prints, TODOs, commented code, or secrets** in the diff. ✓
- **No new exception swallowing**: every `err` path either returns with context or logs to stderr with a preservation fallback. ✓

### Coverage gaps (for `/test`, not blocking merge here)

- The genuine transient `PackFS` failure branch (`upgrade.go:134-140`) has no direct test now that the ghostpack scenario was repurposed. Triggering it would require injecting an `fs.Sub` error against a pack that IS in `AvailablePacks()` — hard to provoke from `fstest.MapFS`.
- The pack-scoped `ComputeDiffsWithManifest` failure branch (`upgrade.go:147-153`) similarly has no direct test.
- `TestRunUpgrade_ReportsDeletedPackFile` does not assert the stdout notice was emitted (see LOW finding above).

### Recommendation

- Merge: **approve** (no CRITICAL, HIGH, or MEDIUM findings).
- All Round 2 findings are LOW cosmetic / test-assertion completeness items. None block the fix.
- Follow-ups (non-blocking, can ship separately):
  1. Harmonize stderr formatting verb (`%q` vs `%s`) across the three pack-related messages.
  2. Strengthen `TestRunUpgrade_DropsPacksRemovedFromTemplates` to positively assert golang is retained in `Meta.Packs`.
  3. Add a stdout-capture assertion to `TestRunUpgrade_ReportsDeletedPackFile`, or flag for `/test` to confirm the pack-scoped ActionRemove notice actually surfaces to users.
  4. Consider covering the transient `PackFS` / `ComputeDiffsWithManifest` failure branches (may require a custom `fs.FS` wrapper that injects errors mid-walk).

No tech-debt entry needed — findings are small and directly actionable. The Codex ACTION_REQUIRED concerns are genuinely addressed by the code changes, not just papered over.

## Round 3 (post-codex-2)

- Date: 2026-04-21
- Trigger: Re-review after Round 2 Codex findings landed in commit `6f038de` (ACTION_REQUIRED: drop manifest entry on `ActionRemove`; WORTH_CONSIDERING: Windows-portable test path assertions; plus Round 2 self-review LOW: positive `golang` retention assertion; plus new stdout-capture test).
- Scope: `git show 6f038de` only. All prior commits were approved in Rounds 1–2.
- Reviewer: reviewer subagent (self-review, diff quality only).

### Evidence reviewed

- `git show 6f038de --stat`: 3 files, +86 / −29
  - `internal/cli/upgrade.go` (−5 / +7, net +2; only the `ActionRemove` branch at `:225-232` plus comment)
  - `internal/cli/cli_test.go` (+63 / −24, net +39)
  - `docs/reports/codex-triage-*.md` (triage append — out of code-quality scope)
- Full re-read of `internal/cli/upgrade.go:223-253` to confirm the `ActionRemove` drop is consistent with the surrounding switch arms (`ActionAutoUpdate` / `ActionConflict/skip` / `ActionAdd` / `ActionSkip`) and with the `notified` counter + summary line at `:247-249`.
- Cross-check of `internal/upgrade/diff.go:171-183` (`checkRemovals` loop) to confirm that dropping the entry in `upgrade.go` does NOT cause the diff engine to re-emit `ActionRemove` on the next run (since on a successful second upgrade the entry is gone from `manifest.Files`, the iteration never fires for that key).
- Inspection of `internal/cli/cli_test.go:1-12` (imports), `:300-359` (the renamed + hardened `TestRunUpgrade_ReportsDeletedPackFileOnceThenDrops`), `:253-296` (hardened `TestRunUpgrade_DropsPacksRemovedFromTemplates` with positive `golang` retention).
- Confirmed no `t.Parallel()` in `cli_test.go` (grep `-n "t.Parallel"` → none), so swapping `os.Stdout` is test-local and does not race with other tests in the package.

### What the commit does

1. `ActionRemove` no longer calls `manifest.SetFile(d.Path, d.OldHash)`. The entry is simply dropped from the new manifest on the same run where the user was notified. Fixes the perpetual re-notify bug.
2. Every hard-coded pack manifest key in the tests (`"packs/languages/golang/README.md"`, `"packs/languages/ghostpack/verify.sh"`, `"packs/languages/golang/deprecated.sh"`) is now constructed via `filepath.Join`. On Windows the generated separator is `\`, matching what `executeInit` writes. On Linux/macOS the value is byte-identical to the previous string literal, so CI behavior is unchanged.
3. `TestRunUpgrade_DropsPacksRemovedFromTemplates` now asserts both directions: `ghostpack` removed AND `golang` retained in `Meta.Packs`. Uses boolean flags + post-loop assertions instead of `t.Error` inside the iteration (which would otherwise fail only on a seen-then-not-retained scenario, not a never-seen scenario).
4. `TestRunUpgrade_ReportsDeletedPackFile` was renamed to `TestRunUpgrade_ReportsDeletedPackFileOnceThenDrops` and now:
   - Captures stdout via `os.Pipe()` around the first `runUpgrade` call.
   - Asserts the pack-scoped path appears in stdout (the "removed from template" user-facing notice is actually emitted).
   - Asserts the entry is dropped from the manifest after the first upgrade.
   - Runs a second same-version upgrade, captures its stdout, and asserts the string `"removed from template"` is NOT present — the idempotency contract.

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | `internal/cli/cli_test.go:324-330, 347-352` — stdout capture pattern | The stdout-capture uses `os.Pipe()` with a synchronous `runUpgrade` call between `os.Stdout = w` and `w.Close()`. If `runUpgrade` ever emits more than one pipe-buffer's worth of output (typically 16–64 KiB depending on kernel), the writer would block and the test would deadlock because no goroutine is draining the reader concurrently. Current output is ~1–2 KiB for this scenario (1 `ActionRemove` print + 3 summary lines), so the risk is theoretical. But if a future change adds per-file prints over many files, this test could hang intermittently. | `cli_test.go:324-336` — the reader `io.ReadAll(r)` runs AFTER `w.Close()` and after the upgrade completes. No draining goroutine. The `fmt.Printf` count in `upgrade.go` is 19 sites, most per-file. | Optional: drain the pipe concurrently (`go func() { out, _ = io.ReadAll(r) }()`) or use a `bytes.Buffer` swap via `log`-style redirection. Non-blocking for this PR. |
| LOW | `internal/cli/cli_test.go:324-336, 347-358` — `os.Stdout` restoration on panic | If `runUpgrade` panics between `os.Stdout = w` and the manual restore at line 329/351, subsequent tests in the package run with a broken `os.Stdout` pointing at a closed pipe end. There is no `t.Cleanup` or `defer` guarding the restore. | Lines 324-329 and 347-351 — no defer on the stdout swap. | Optional: wrap as `defer func() { os.Stdout = origStdout }()` or `t.Cleanup(func() { os.Stdout = origStdout })` immediately after the swap. Defensive, not blocking. |
| LOW | `internal/cli/cli_test.go:325, 347` — discarded `os.Pipe` errors | `r, w, _ := os.Pipe()` and `r2, w2, _ := os.Pipe()` both discard the error return. In practice `os.Pipe` can fail under `EMFILE` (file-descriptor exhaustion). Test would then proceed with nil `w`, panic on `os.Stdout = w` → nil `Write`. | `cli_test.go:325, 347`. | Optional: `r, w, err := os.Pipe(); if err != nil { t.Fatalf("pipe: %v", err) }`. Low-frequency failure mode, fine to ignore but easy to harden. |
| LOW | `internal/cli/cli_test.go:330, 352` — discarded `io.ReadAll` errors | `out, _ := io.ReadAll(r)` discards the error. If the read fails, the test falls back to asserting against an empty buffer and produces a misleading failure message ("first upgrade stdout missing pack-scoped remove notice"). The actual error (e.g., EBADF on a closed pipe) would be lost. | `cli_test.go:330, 352`. | Optional: capture and `t.Logf` the error on non-nil. Not blocking. |
| LOW | `internal/cli/upgrade.go:225-232` — `ActionRemove` comment is accurate but no longer tracks `notified` semantics | The comment now says "Drop the entry from the new manifest" but does not explain WHY the `notified` counter is still incremented (for the summary line at `:247-249`). A reader coming fresh might wonder if the removal and the count are linked. | `upgrade.go:225-232`. | Optional: add one line like `// notified++ still fires so the summary ("Removed from template: N files") is accurate.` Cosmetic only. |

No CRITICAL, HIGH, or MEDIUM findings.

### Cross-check against Round 2 follow-ups

- **LOW test misses stdout-capture for `ActionRemove` notice** → **resolved** by Round 3. The new `TestRunUpgrade_ReportsDeletedPackFileOnceThenDrops` captures stdout and asserts the `deprecated.sh` path appears. ✓
- **LOW golang not positively asserted as retained in `Meta.Packs`** → **resolved** by Round 3. The two-flag pattern explicitly asserts both directions. ✓
- **LOW `%q` vs `%s` inconsistency in stderr messages** → **unchanged**. Out of scope for this commit (no diff in those lines). Remains a non-blocking cosmetic item.

### Correctness spot-checks

- **`ActionRemove` drop interacts correctly with `maps.Copy(manifest.Files, preservedPackEntries)` at `upgrade.go:168`**: `preservedPackEntries` only contains entries for packs whose `PackFS` or `ComputeDiffsWithManifest` failed. In the happy-path test scenario (`golang` loads successfully, `deprecated.sh` is `ActionRemove`d), `preservedPackEntries` is empty, so the `maps.Copy` is a no-op and the entry is truly gone from the final manifest. ✓
- **Second-upgrade idempotency**: after the first upgrade drops `deprecated.sh`, the second `ReadManifest` does not have that key; `ComputeDiffsWithManifest` walks `packFS` (which also does not have `deprecated.sh`); the `checkRemovals` loop at `diff.go:173-183` iterates `manifest.Files` — since the key is absent, no `ActionRemove` is emitted. No notice printed. The test's stdout-absence assertion matches the code path. ✓
- **`notified` counter consistency**: the `ActionRemove` arm still increments `notified++`, so the summary `"Removed from template: N files"` line remains correct. The only thing that changed is whether the manifest persists the tracking — not the user-facing count. ✓
- **Windows-path portability of the stdout substring assertion**: on Windows, `deprecatedEntry = filepath.Join("packs", "languages", "golang", "deprecated.sh")` = `"packs\\languages\\golang\\deprecated.sh"`. The stdout comes from `fmt.Printf("  ⚠ %s ...", d.Path)` where `d.Path` is built at `upgrade.go:155` via `filepath.Join("packs", "languages", pack, packDiffs[i].Path)` — same `filepath.Join` producing the same separator. So `strings.Contains` matches on all platforms. ✓
- **No new debug prints, TODOs, commented-out code, or secrets.** ✓
- **No exception swallowing in production code.** Test-side error discards (LOW findings above) are localized to test scaffolding. ✓

### Coverage gaps (for `/test`, not blocking merge here)

- The stdout-capture pattern does not verify that the PRE-notice summary (`"Removed from template: 1 files (review manually)"` at `upgrade.go:247-248`) is also emitted. A `strings.Contains(out, "Removed from template")` assertion on the first run would catch a regression that stripped the summary while keeping the per-file notice.
- The second-upgrade assertion uses the string `"removed from template"` (lowercase, matching the per-file notice template). It would also match the summary line `"Removed from template:"` if the summary were ever emitted with N=0. The summary is guarded by `if notified > 0` at `:247`, so no false positive today — but the assertion string is not a perfect oracle.

### Recommendation

- Merge: **approve** (no CRITICAL, HIGH, or MEDIUM findings).
- All Round 3 findings are LOW defensiveness / test-scaffolding hygiene items. None block the fix, and several (pipe deadlock, stdout-restore-on-panic, discarded pipe errors) are generic patterns any future stdout-capture test in this repo could adopt uniformly later.
- The Round 2 Codex ACTION_REQUIRED ("drop entry on `ActionRemove`") is substantively resolved, with a regression test that exercises both the user-facing notice AND the idempotency contract on a second run.
- The Round 2 Codex WORTH_CONSIDERING ("Windows-portable path assertions") is resolved uniformly across every hard-coded pack-manifest key in `cli_test.go`.
- The Round 2 self-review LOW ("positive golang retention") is closed.

No tech-debt entry needed. Round 3 introduces no new accumulated complexity — it removes a latent idempotency bug and hardens test portability.
