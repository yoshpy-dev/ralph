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
