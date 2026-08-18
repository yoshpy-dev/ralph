# Sync-docs report: overlay-scaffold-v2 Phase 4 (residual pass)

- Date: 2026-08-18
- Plan: `docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- Doc maintainer: `doc-maintainer` subagent (`/sync-docs`)
- Scope: `git diff main...HEAD` on `feat/overlay-scaffold-v2-p4` (base `main` `1025d13`, HEAD `b1515ea`). Slice 5 (commit `c8b47bd`) already performed the main doc sweep (README migration section, `docs/recipes/adding-a-language-pack.md` + `templates/base/` twin, spec FR-8 addendum, `AGENTS.md` repo map, tech-debt RESOLVED rows). This pass checks the self-review fix commit (`b1babe7`) that landed after slice 5 for doc drift it introduced or left unreconciled.

## Verdict

**No drift found.** The self-review fix commit already corrected the one doc issue its own review raised (README's "manifest-less" wording), and no other doc surface needed a change for the behavior `b1babe7` added (`OpDeleteOldPathAdoptFork`, the drift-sentinel pass-through, the parent-chain guard on delete/dir ops, and prune-candidate/prune-actual reconciliation). One plan-hygiene gap was found and fixed (see below), matching the same gap closed in the Phase 2 and Phase 3 sync-docs passes.

## What was checked

1. **README "manifest-less" wording (self-review MEDIUM-3)** — self-review flagged both new README passages (`### ralph upgrade`, `#### Migrating from a legacy layout`) for claiming migration covers "legacy (pre-v2 or manifest-less)" projects, when `runUpgradeIOWithOptions` actually returns a `run 'ralph init' first` error before any layout dispatch for a manifest-less directory. `b1babe7` dropped "or manifest-less" from both sentences. Grepped `manifest-less` across README, `docs/`, and `templates/`: the only remaining hits are the verify/self-review reports documenting the fix (expected) and an unrelated line in an archived Phase-0 audit plan (`docs/plans/archive/2026-07-13-audit-findings.md`, a different feature's doctor fallback). No orphaned wording.
2. **`docs/recipes/adding-a-language-pack.md` twin consistency** — root and `templates/base/` copies are still byte-identical after slice 5's edit (`b1babe7` did not touch either); `./scripts/check-sync.sh` confirms (0 DRIFTED).
3. **`docs/tech-debt/README.md` row coherence** — three rows are RESOLVED in this branch: (a) the batched cycle-1 LOW row's `in io.Reader` sub-item (now read by `runMigrateLegacy`'s confirmation prompt) and its hardcoded-permission item (3) count, which `b1babe7` updated a second time from "now-5" to "now-6" sites to include `writeMigrationFile`'s `0755` — count and file list both correctly say 6 in the current file; (b) the `.codex/AGENTS.override.md` re-attribution row; (c) the untracked-seed-collision `classifyUntracked` owner-blindness row. All three read as fully struck through with a `RESOLVED 2026-08-18 in feat/overlay-scaffold-v2-p4` marker and a concrete pointer to the closing code (`classifyLegacyPath`'s `pathCodexOverride` case, `ReplaceOptions.OwnerForPath` + `classifyUntracked`'s seed branch). No stale counts or dangling strikethrough found.
4. **Spec FR-8 addendum vs. `b1babe7`'s executor changes** — `b1babe7` added `OpDeleteOldPathAdoptFork` (fork-attribution recovery for the rerun and collision-matrix paths) and the `ErrUpgradeDriftRemaining` pass-through. Neither touches the CLAUDE.md/AGENTS.md/`.gitignore` special-face behavior FR-8's Phase-4 addendum describes (unmodified → seed/block replace, modified → left in place + chained block append). Confirmed FR-8's addendum text still matches current `classifyLegacyPath`/`classifyUnmodifiedGeneric` special-casing for those three paths; no edit needed.
5. **README migration-section detail level vs. `PrunedHookNearMisses`** — `b1babe7` added near-miss reporting for argument-carrying legacy hook variants (a command is only pruned on an exact match; a substring-matching near-miss is left in place and separately reported in `docs/reports/ralph-migration-<date>.md`). README's "Migrating from a legacy layout" section does not mention settings.json hook pruning at any level of detail (it covers only the four coarse op kinds: replace/relocate/fork/delete) — this predates `b1babe7` and was a Slice-5 scope choice, not something the fix commit introduced. Judged: not worth a half-sentence addition. README stays at path-classification granularity throughout; the settings-specific pruning mechanics (including the near-miss distinction) are exactly what the generated migration report exists to surface, consistent with how README already defers other special-cased paths (CLAUDE.md content, `.codex/AGENTS.override.md` reattribution) to the report rather than enumerating them. No change made.
6. **AGENTS.md repo map** — `internal/upgrade/` entry (`AGENTS.md:88`, mirrored `templates/base/AGENTS.md`) already reads "legacy layouts migrate via a confirmed one-time flow (internal/cli/migrate.go) before chaining into this engine," written at slice 5. Still accurate after `b1babe7`; no symbol-level claims to go stale.

## Fixed in this pass

- **Plan progress checklist** (`docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md`) — "Review artifact created" was still unchecked even though `docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md` exists and is committed (`8fc6260`). Checked it. Added and checked a "Sync-docs artifact created" line (the shared `docs/plans/templates/feature-plan.md` doesn't carry this line either — the same gap the Phase 2 and Phase 3 sync-docs passes each closed locally in their own plan files).

## Verification after doc edits

```
$ ./scripts/check-sync.sh
=== Sync Summary ===
  IDENTICAL:      158
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     3
PASS: all files in sync.

$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
```

## Files changed

- `docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md` (checklist only)
- `docs/reports/sync-docs-2026-08-18-overlay-scaffold-v2-p4.md` (this report)
- `docs/insights/events/2026-08-18-overlay-scaffold-v2-p4.jsonl` (appended `sync_docs` event)

## Known gaps

None. This was a residual/confirmatory pass; slice 5's doc sweep already covered the substantive drift from Phase 4's behavior, and the self-review fix commit's own README correction closed the one doc issue it raised.
