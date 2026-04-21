# Sync-docs report: fix-ralph-upgrade-manifest-hash-loss

- Date: 2026-04-21
- Plan: `docs/plans/active/2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md`
- Branch: `fix/ralph-upgrade-manifest-hash-loss`
- Maintainer: `doc-maintainer` subagent (Claude Code)
- Upstream trigger: Documentation drift flagged in `docs/reports/verify-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md` (lines 56–72).

## Scope

Align product documentation with the implemented behavior of `ralph upgrade` idempotency, empty-hash heal, pack manifest namespacing, and pack diff-failure entry preservation. No implementation or test files were touched.

## Files updated

| File | Change | Reason |
| --- | --- | --- |
| `docs/specs/2026-04-16-ralph-cli-tool.md` | Added `#### 冪等性と自動修復 (idempotency & heal)` subsection under `### upgrade フロー` (after line 284, before `## セキュリティ考慮事項`). | Spec section described only the interactive diff UI output; it did not mention same-version idempotency (`ActionSkip` now carries `NewHash`), empty-hash self-heal (`hash = ''` repaired when disk matches template; conflict otherwise), pack namespacing (`packs/languages/<pack>/<rel>` keys with separate base/pack diff scopes), or pack-entry preservation on diff failure. All four items are behaviors implemented on this branch (see `internal/upgrade/diff.go` heal branch + skip-NewHash branch, and `internal/cli/upgrade.go` `splitManifestForBase`, `splitManifestForPack`, `preservePackEntries`). |
| `docs/plans/active/2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md` | Flipped Progress checklist items: Review / Verification / Test / Sync-docs artifact boxes now checked. | Brings plan progress in line with artifacts already on disk (`docs/reports/self-review-…md`, `verify-…md`, `test-…md`, and this report). |

## Intentionally left alone

| Area | Why not changed |
| --- | --- |
| `AGENTS.md` | The fix is a bug fix inside the `upgrade` command; it does not change any contract surfaced in the repo map (no new file/module, no new skill, no renamed script). Keeping this map file short per `.claude/rules/documentation.md`. |
| `CLAUDE.md` | No default-behavior or contract-level change. |
| `README.md` | Upgrade command is referenced only as a cobra subcommand in the package overview (line 45–47); no user-facing command surface or flag changed. |
| `docs/recipes/*` | Grep confirmed no recipe documents `ralph upgrade` behavior (only a passing `migration | Upgrades` label reference in `docs/recipes/ralph-loop.md:84`, which is unrelated). Nothing to resync. |
| `.claude/rules/*` | No rule references `ralph upgrade` idempotency or manifest heal semantics. |
| `docs/quality/*` | DoD / quality gates unchanged — the fix adds no new gate. |
| Implementation and test files | Explicitly out of scope for `/sync-docs`. |

## Additional drift checks performed

- `grep -l "ralph upgrade" docs/**/*` — only archived plans, active plan itself, and this branch's reports referenced the command. Archived plans are frozen by convention; no edits.
- `grep "upgrade" README.md` — two hits, both in the `internal/cli/` / `internal/upgrade/` repo-map enumeration. Still accurate.
- `AGENTS.md` repo map — `internal/upgrade/` description ("hash-based diff engine, conflict resolution (auto-update, conflict, add, remove)") still correctly describes the public-facing action set; the new "skip-with-NewHash" and "heal" behaviors are refinements within existing actions and do not require map-level expansion.

## Evidence

- Verify report drift recommendation: `docs/reports/verify-2026-04-21-fix-ralph-upgrade-manifest-hash-loss.md` §Documentation drift (lines 56–72).
- Implementation anchors cited in the new subsection:
  - `internal/upgrade/diff.go` — skip-with-`NewHash` branch and empty-hash heal branch.
  - `internal/cli/upgrade.go` — `splitManifestForBase`, `splitManifestForPack`, `preservePackEntries`, and the `maps.Copy` merge of preserved pack entries into the new manifest.

## Verdict

Documentation now matches implementation for the four behaviors flagged by `/verify`. No further doc drift detected outside the spec. Ready for `/codex-review` and `/pr`.
