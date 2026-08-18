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

## Cycle 2 (final, 2/2)

- Date: 2026-08-18
- Doc maintainer: `doc-maintainer` subagent (Claude Code, `/sync-docs`)
- Scope: **delta only** — commits since cycle-1's `851e906`: `6ad6e16` (cross-review triage), `c51497e` (cross-review AR#1/AR#2/AR#3 fixes: `OpDeleteOldPathAdoptFork` `NewPath` validation, seed-collision advisory preserved through migration by omitting the path from the v3 manifest, adopted-fork diffs added to the migration report), `13adf85` (self-review cycle-2 section, 1 MEDIUM + 3 LOW), `aabca2d` (self-review C2-1..C2-4 fixes: cycle-qualified `AR#` citations, settings.json rewrite skipped when a prune removes nothing, a duplicate-warning doc comment, three doc-comment corrections for the new op kind), `0cc50e5`/`d686226` (verify/test cycle-2 report sections). HEAD is `d686226`, working tree clean.

### Verdict

**No drift.** All four delta commits change production behavior or internal documentation (Go doc comments, code citations) without changing any claim already made in user-facing docs (README, spec, tech-debt register, recipes).

### What was checked

1. **AR#1 (`OpDeleteOldPathAdoptFork` `NewPath` validation) and AR#3 (adopted-fork diffs added to the migration report's "Fork diffs" section)** — both tighten internal correctness of behavior the README already describes at the coarse classification level ("fork" outcome, "diffs for every forked file" in the "Migrating from a legacy layout" section, `README.md:149`). AR#3 in particular makes that existing README sentence *true* for a case (`OpDeleteOldPathAdoptFork`) it previously wasn't for — no wording change needed since the sentence was already written at the right granularity ("every forked file", not enumerating op kinds). Confirmed no README/spec text names `OpDeleteOldPathAdoptFork` or any other `MigrationOpKind` directly — the abstraction boundary the docs already use (classification categories, not Go type names) absorbs this change with zero edits.
2. **AR#2 (seed-collision advisory preserved through migration by omitting the untracked path from the v3 manifest, instead of recording a hash that would make the chained upgrade's `classifyUntracked` silently no-op)** — this closes the migration-path route to a guarantee AC-1 already states in the plan ("manifest 未登録 + desired に存在する seed パスにローカルファイルが既存する場合、drift にならず...advisory がレポートに載る") and that `docs/tech-debt/README.md`'s RESOLVED row for the untracked-seed-collision item already documents as closed by `ReplaceOptions.OwnerForPath` + `classifyUntracked`. AR#2 is the migration-path instance of that same fix reaching the same end state via a different code path; the tech-debt row's RESOLVED text does not scope itself to "direct `ralph upgrade` only" and needs no correction. No README passage describes untracked-seed-collision handling at all (it is `ralph doctor`/`ralph upgrade` internals, correctly left to the advisory-diff report per the existing sync-docs cycle-1 decision on documentation granularity). No edit needed.
3. **C2-2 (settings.json rewrite skipped when a hook prune finds candidates but removes nothing — the near-miss-only case)** — checked README's "Migrating from a legacy layout" section (`README.md:143-149`) and AC-13's spec/plan wording for any claim this refines. Neither mentions settings.json pruning at the write/no-write mechanic level; both stay at "known legacy ralph hook entries are pruned (report-recorded)" / equivalent, which remains true — this fix changes *when a file write occurs*, not *what gets reported as pruned or preserved*. Matches the cycle-1 sync-docs pass's own finding #5 (this same granularity decision, made before C2-2 existed) — re-confirmed rather than re-litigated. No edit needed.
4. **C2-1, C2-3, C2-4 (cycle-qualified `AR#` citations, a duplicate pack-warning doc comment, three `OpDeleteOldPathAdoptFork` doc-comment corrections)** — all four are Go source comments and one `docs/tech-debt/README.md` LOW-5 row count update (`aabca2d` touches `internal/cli/migrate.go` and `internal/cli/migrate_test.go` only per `git show aabca2d --stat`; the tech-debt row edit was already part of `c8b47bd`/`b1babe7`, not this delta). No versioned doc outside source comments changed.
5. **`docs/tech-debt/README.md` register consistency** — self-review cycle-2 committed to opening a new batched row only for whichever of C2-1..C2-4 the fix pass left unfixed ("This row should only be opened for the items actually left unfixed... If the fix pass takes all four..., no row is needed."). `aabca2d`'s commit message confirms all four were fixed. Grepped the register for a new cycle-2-sourced row: none exists — consistent with the report's own conditional. The three RESOLVED rows already closed in cycle 1 (seed-collision advisory, `.codex/AGENTS.override.md` reattribution, the batched hardcoded-permission count) are unaffected by this delta.
6. **Plan progress checklist and `docs/insights/events/2026-08-18-overlay-scaffold-v2-p4.jsonl`** — all seven checklist lines through "Sync-docs artifact created" are already `[x]` (set in cycle 1); only "PR created" remains open, correctly. No new checklist line needed for cycle 2 — the plan template still lacks a per-cycle sync-docs marker (same known gap noted in cycle 1; not reopened here since the existing single line already reflects "a sync-docs artifact exists," which remains true).

### Verification after this pass

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

$ grep -rn "manifest-less" README.md docs/ templates/
docs/plans/archive/2026-07-13-audit-findings.md:76: (unrelated, pre-existing, different feature)
```

Corroborated by `docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md`'s own Cycle 2 "Documentation drift" section, reached independently by `/verify`: "None found in the delta... README.md, docs/tech-debt/README.md, and the spec were not touched by c51497e/13adf85/aabca2d and remain in sync as established in cycle 1."

### Files changed

- `docs/reports/sync-docs-2026-08-18-overlay-scaffold-v2-p4.md` (this section)
- `docs/insights/events/2026-08-18-overlay-scaffold-v2-p4.jsonl` (appended `sync_docs` event, cycle 2)

### Known gaps

None. This is the final pipeline cycle (2/2); no documentation surface needed a change for the cycle-2 delta.
