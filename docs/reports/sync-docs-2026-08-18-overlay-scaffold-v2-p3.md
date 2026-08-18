# Sync-docs report: overlay-scaffold-v2 Phase 3 (residual pass)

- Date: 2026-08-18
- Plan: `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- Doc maintainer: `doc-maintainer` subagent (`/sync-docs`)
- Scope: `git diff main...HEAD` on `feat/overlay-scaffold-v2-p3` (base `main`, HEAD `62631aa`). Slice 5 (commit `895ba8d`) already performed the main doc sweep (spec exit-code resolution, `AGENTS.md` repo map, README upgrade section, `docs/recipes/adding-a-language-pack.md`, `docs/tech-debt/README.md`). This pass checks the four fix-slice commits landed after slice 5 (`3d8461d`, `6b617dc`, `c785300`, `62631aa`) for doc drift they may have introduced or left unreconciled.

## Verdict

**No drift found.** Slice 5's doc sweep and the fix slices' own `docs/tech-debt/README.md` row updates (commit `6b617dc`) already account for every surface this residual pass was asked to check. One plan-hygiene gap was found and fixed (see below).

## What was checked

1. **`AGENTS.md` repo map after M4/M5 API deletions** — the repo map entry for `internal/upgrade/` (line ~89) names concepts (replace planner, block engine, settings 3-way merge, advisory diff, report writer), not the specific Go symbols (`SetFileUnmanaged`, `WithTemplateHash`, `FileStatePartial`, `BaselineStatusAvailable`) that verify's MEDIUM 5 finding had flagged as unreachable and that the fix slice (`3d8461d`) deleted. Confirmed: no doc references any of those four symbol names. Repo map still accurate.
2. **README's upgrade section vs. final flag set** — `## \`ralph upgrade\`` (README.md:126-141) documents exactly `--dry-run`, `--diff` (implying `--dry-run`), and `--pager auto|always|never`; states plainly "There is no `--force` flag." Cross-checked against `internal/cli/upgrade.go`/`upgrade_v2.go` flag registration: matches. No stale `--force` mentions found anywhere in README, `docs/recipes/`, `docs/quality/`, `.claude/rules/`, or `.claude/skills/` (the only `--force` hits are the unrelated `ralph-worktree.sh cleanup --force-branch` flag).
3. **Manifest API doc references** — grepped `SetFileUnmanaged`, `WithTemplateHash`, `SetFileWithBaseline`, `SetFileResolvedWithBaseline`, `addBaselineOnly` across `docs/`, `.claude/rules/`, `.claude/skills/`, `templates/`. All remaining hits are in archived plans and dated self-review/verify reports (`docs/plans/archive/2026-04-22-upgrade-detect-local-edits.md`, `docs/reports/self-review-2026-08-1{7,8}-*.md`, `docs/reports/verify-2026-08-1{7,8}-*.md`) — historical artifacts, correctly left untouched. No living doc (README, AGENTS.md, rules, skills, recipes, quality docs) references any of these deleted/dead symbols.
4. **`docs/quality/definition-of-done.md`** — no claims about upgrade interactivity, conflict resolution, or baselines; only references the post-implementation pipeline and generic evidence-redaction rules. Nothing to reconcile.
5. **`docs/recipes/codex-setup.md`** — re-confirmed the "Recovery from a half-applied upgrade" section's "represents ... as an `add` plus a `remove`" wording (lines 87-100). This describes the *net effect* of a template path rename under the current engine (`internal/upgrade/replaceplan.go`'s `OpCreate` + `OpDelete`, confirmed present; `diff.go`/`merge.go`/`baseline.go` confirmed deleted), not the deleted legacy engine's per-file `[a]pply/[k]eep/[e]dit` conflict-resolution vocabulary. Still accurate; no edit needed.
6. **Stale `baseline` references in living docs** — grepped `README.md`, `AGENTS.md`, `docs/recipes/`, `docs/quality/`, `.claude/rules/`, `.claude/skills/` (and their `templates/base/`/`.agents/skills/` twins) for `baseline`. Only hit: `.claude/skills/work/SKILL.md`'s "so the implementer starts from an unambiguous baseline" — generic English usage (starting point), unrelated to the deleted `.ralph/baseline/` scaffold mechanism. No change needed.
7. **`docs/tech-debt/README.md` fix-slice reconciliation** — the fix slices (`3d8461d`, `6b617dc`, `c785300`) already rewrote every tech-debt row that referenced the deleted legacy engine (the pack-rule-rename-detection row now describes `PlanCoreReplaceDesired`'s successor behavior with an explicit `<!-- UPDATED 2026-08-18 in feat/overlay-scaffold-v2-p3 -->` marker; the Codex-hooks-parity and `pre_bash_guard.sh` rows retargeted their "next touch" trigger from "Phase 3" to "next config change" since Phase 3 landed without touching either surface; a new row records the Phase-3 coverage loss from deleting the legacy test suite). No further edits needed.

## Fixed in this pass

- **Plan progress checklist** (`docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`) — "Review artifact created" and "Verification artifact created" were still unchecked even though `docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md` and `docs/reports/verify-2026-08-18-overlay-scaffold-v2-p3.md` both exist and are committed (`618f55a`, `c785300`). Checked both. Added and checked a "Sync-docs artifact created" line (the shared `docs/plans/templates/feature-plan.md` doesn't carry this line either, matching the same gap noted and closed in the Phase 2 plan's own sync-docs pass).

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

- `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md` (checklist only)
- `docs/reports/sync-docs-2026-08-18-overlay-scaffold-v2-p3.md` (this report)
- `docs/insights/events/2026-08-18-overlay-scaffold-v2-p3.jsonl` (appended `sync_docs` event)

## Known gaps

None. This was a residual/confirmatory pass; slice 5's doc sweep already covered the substantive drift from Phase 3's behavior changes.
