# sync-docs report: org-envelope-low-findings

- Date: 2026-09-17
- Plan: `docs/plans/active/2026-09-17-org-envelope-low-findings.md`
- Prior reports: `docs/reports/self-review-2026-09-17-org-envelope-low-findings.md` (Merge,
  no open findings), `docs/reports/verify-2026-09-17-org-envelope-low-findings.md` (Pass),
  `docs/reports/test-2026-09-17-org-envelope-low-findings.md` (PASS)

## Summary

This PR is a cosmetic sweep of 10 deferred LOW findings from PR #152 plus
in-cycle self-review fixes (L1-L8, N1-N3). The only behavior change is in
`ralph doctor`'s codex slug check (`internal/cli/doctor_codex_models.go`):
`codexModelsCachePath` now returns `(string, error)`, uses a non-empty
`CODEX_HOME` literally (no trim), and reports a home-resolution failure as
an `info` result instead of silently falling back to a cwd-relative path.
I audited every doc surface the task brief named plus the repo-wide grep
sweeps below and found no additional drift — the plan's own implementation
slices (314b89f, 746c70d, 3a9362c, ff30ee2, b677a95) had already fixed every
comment, table row, and tech-debt row this PR touches. No files were changed
in this pass.

## Files checked (no changes needed — already correct)

- `internal/cli/doctor_codex_models.go` — doc comment now says "Six
  deterministic outcomes" and lists the new home-resolution-failure case;
  matches the code (`checkCodexModelSlugs` returns `info` on that path).
  `grep -n TrimSpace internal/cli/doctor_codex_models.go` is empty (AC-1).
- `docs/recipes/codex-setup.md` (+ `templates/base/docs/recipes/codex-setup.md`,
  confirmed byte-identical by `check-sync.sh`) — neither file mentions
  `CODEX_HOME` resolution detail or the model-slug check's outcome count;
  nothing to update.
- `README.md` — no mention of `Leaded`, `codex slug`/`model slug`, or the
  org formation pattern table (`grep -n` empty); no drift.
- `docs/specs/2026-08-01-org-runtime.md` — its `Leaded` mentions (FR-10
  checklist item, line 46) name the pattern generically ("Solo / Leaded /
  Parallel の型") and don't reproduce the contradictory delegation sentence
  this PR fixed. Its FR-2 line about the codex-slug doctor check is a
  feature-level description ("軽量 Check を追加") unaffected by the
  `(string, error)` signature change, which is an implementation detail one
  level below spec granularity. No revision-note edit needed — this PR
  changes no spec-level behavior (Non-goals confirms org runtime behavior
  itself is out of scope).
- `.claude/skills/org/SKILL.md` (+ `.agents/skills/org/SKILL.md`,
  `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md`) — the Leaded row now reads
  "実装は完了済みか既存フロー(`/work`)で進める前提で、Lead 自身は実装しない",
  consistent with `internal/org/prompts/lead.md`'s implementer-first
  delegation. All four mirrors verified byte-identical (`diff` pairwise);
  `grep -rn 'Lead 自身か既存フロー' .claude .agents templates/base` is empty (AC-4).
- `docs/plans/active/2026-09-17-org-envelope-low-findings.md` — Progress
  checklist and Status (`In progress`) accurately reflect that the PR has
  not been created yet; nothing to tick before `/pr` runs.
- `docs/tech-debt/README.md` — the closed row's table structure is intact
  (6 pipes / 5 columns, matching the header) and its content matches the
  plan's Design decisions and Deviation notes (untrimmed `CODEX_HOME`
  dismissed on codex-source evidence, items 3-10 fixed as listed with
  correct commit SHAs). The forward reference to
  `docs/plans/archive/2026-09-17-org-envelope-low-findings.md` (not yet
  created — the plan is still under `docs/plans/active/`) is intentional
  per the task brief and left as-is; no other row was touched.
- `AGENTS.md`, `.claude/rules/ralph/*.md` — no mention of
  `codexModelsCachePath`, `checkCodexModelSlugs`, `DefaultModelForDriver`,
  `pruneRetiredConditions`, or "deterministic outcomes" anywhere in these
  files (`grep -rln` empty). `model-routing.md`'s org-receipts paragraph
  ("Codex seats have no aliases; `[org].model_pool` carries codex model
  slugs, and `ralph doctor` warns when a slug is missing from codex's local
  model cache") describes behavior at a level this PR does not change
  (still warns; only the home-resolution-failure edge case changed from
  silent to `info`). No edit needed.
- `internal/config/config.go`, `internal/org/prompts.go`,
  `internal/org/prompts/lead.md`, `internal/org/watch.go`,
  `internal/org/spawn.go` — doc comments already match the plan's items 3,
  6, 7, 8 and L1 fixes; `grep -n 'later slice' internal/config/config.go`
  and `grep -n 'PR③' internal/org/prompts.go` are both empty. The only
  remaining hits for "later slice", "PR③", and "Lead 自身か既存フロー"
  repo-wide are inside the plan file itself, quoting the pre-fix text as
  historical description of what the findings were — expected, not drift.

## Drift found and fixed

None. No documentation surface needed a change beyond what the implementer
slices and self-review fix rounds had already committed.

## Verification commands run

```
./scripts/check-skill-sync.sh     # [ok] check-skill-sync: 13 skill(s) in lock-step
./scripts/check-sync.sh           # PASS: all files in sync. (DRIFTED: 0, KNOWN_DIFF: 5, all pre-existing)
./scripts/check-pipeline-sync.sh  # [ok] all 6 referencing docs list every pipeline step
sh tests/test-no-loop-references.sh  # PASS: no live references to the retired Ralph Loop system
```

```
diff .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md              # identical
diff .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md  # identical
diff .agents/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md  # identical
```

```
grep -rn "later slice" --include='*.go' --include='*.md' .   # only in docs/plans/active/2026-09-17-org-envelope-low-findings.md (expected)
grep -rn "PR③" --include='*.go' --include='*.md' .            # no hits
grep -rn "Lead 自身か既存フロー" .                              # only in the plan file (expected)
grep -n "TrimSpace" internal/cli/doctor_codex_models.go        # no hits
grep -rln "codexModelsCachePath\|checkCodexModelSlugs\|DefaultModelForDriver\|pruneRetiredConditions\|deterministic outcomes" \
  .claude/rules AGENTS.md CLAUDE.md docs/quality docs/specs    # no hits
grep -n "Leaded\|codex slug\|model slug\|Solo\|Parallel" README.md  # no hits
```

All referenced test function names (`TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace`,
`TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder`,
`TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors`,
`TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation`,
`TestRenderRolePrompt_Lead_DelegatesToImplementer`,
`TestRolePrompts_SeatTemplatesContainFanOutSection`,
`TestMarkdownSection_AnchorsHeaderAndBoundsBody`) were confirmed present via
`grep -n 'func <name>'` against their respective `_test.go` files.

## Files changed by this sync-docs pass

None. This report is the only new file.
