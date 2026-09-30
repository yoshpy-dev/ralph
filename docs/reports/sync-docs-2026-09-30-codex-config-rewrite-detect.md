# Sync-docs report: codex-config-rewrite-detect

- Date: 2026-09-30
- Plan: `docs/plans/active/2026-09-30-codex-config-rewrite-detect.md` (issue #185)
- Agent: `doc-maintainer` subagent (cycle 1)
- Branch: `fix/codex-config-rewrite-detect`, HEAD `6500eec`

## Checked

- `docs/recipes/codex-setup.md` and its template copy (`templates/base/docs/recipes/codex-setup.md`,
  confirmed present, `cmp` PASS): read the new "If `.codex/config.toml`
  changes on its own" section in full against the final code
  (`scripts/ralph-worktree.sh:89-186`). The symptom, the shape recognized
  (comment/blank-line deletion, trailing `[shell_environment_policy]`
  table), the negative cases that fall back to the generic message, the
  recovery commands (`git diff` / `checkout` / `status --porcelain`), and
  the ruled-out/remaining-candidates summary all match the shipped
  detector and `docs/evidence/codex-config-rewrite-2026-09-30.md`. No
  statement claims CRLF or an edited/added comment is recognized (self-review
  finding #1's fix, verified present: "a changed or added comment,
  re-indentation, a line-ending change ... is reported as an ordinary
  uncommitted change instead").
- `docs/evidence/codex-config-rewrite-2026-09-30.md`: read in full. Matches
  the plan's investigation section and the self-review revisions (finding
  #8: "15 回以上... 関与を否定する根拠にはならない" — observation-scoped,
  not a negative proof; finding #9: distinguishes the project-level file
  from `everything-claude-code`'s user-level write; finding #10: `stat -f`
  command now carries a UTC-offset format `%Y-%m-%dT%H:%M:%S%z` with a
  macOS/BSD note and a GNU coreutils alternative).
- `docs/recipes/worktrees.md`: only generic language ("Refuse to start if
  the default branch checkout is dirty") with no specific error-message
  text or recovery path described — nothing to point at the new section,
  and nothing it says is now inaccurate.
- `README.md`, `AGENTS.md`, `.claude/skills/plan/SKILL.md`,
  `.claude/skills/spec/SKILL.md` (+ `.agents/skills/` and
  `templates/base/.claude/skills/` mirrors): grepped for
  `uncommitted|clean-base|clean base|validate_clean_base|validate-clean-base`.
  All hits are either name-only (`clean-base task worktree`, the `ensure`
  invocation itself) or describe an unrelated code path (`README.md:150`'s
  "uncommitted changes" is `internal/cli/migrate.go`'s legacy-migration
  abort, not `ralph-worktree.sh`'s `validate_clean_base`). None quote or
  describe the generic/specific error-message text this change touched, so
  none are stale.
- `docs/tech-debt/README.md`: no existing row named issue #185,
  `.codex/config.toml`'s rewrite, or `validate_clean_base` (confirmed by
  grep before editing). Added one row (see Changed).
- `docs/evidence/README.md`: a general "what counts as evidence" explainer,
  not a per-file index — none of the other 15 evidence files in the
  directory are listed there either. No entry needed for the new file.
- Gates: `./scripts/check-skill-sync.sh` (13 skills in lock-step),
  `./scripts/check-sync.sh` (159 IDENTICAL / 0 DRIFTED / 0 ROOT_ONLY /
  11 TEMPLATE_ONLY / 5 KNOWN_DIFF, PASS), `./scripts/check-template-purity.sh`
  (PASS, no meta-repo-specific references in templates).

## Stale

None found, apart from the tech-debt gap below (a missing row, not a wrong
one).

## Changed

- `docs/tech-debt/README.md`: added one row for the `.codex/config.toml`
  external rewrite (unidentified root cause, issue #185) — what happened,
  the impact before/after this PR (generic stop → specific message,
  detection only, not a fix), why the remaining candidates are deferred
  (interactive-only, out of this PR's scope per the plan's Non-goals), the
  trigger (the next occurrence, pointing at the evidence checklist in
  `docs/evidence/codex-config-rewrite-2026-09-30.md`), and the related
  evidence/plan/issue references. Column count matches the header (6 pipes
  / 5 cells in both); no literal `|` characters appear in the new cell text,
  so no escaping was needed. `docs/tech-debt/.gitkeep` is `TEMPLATE_ONLY`
  in `check-sync.sh` (this file is meta-repo-only, no template mirror to
  update).

## Left

None outstanding from this cycle's scope. `docs/recipes/worktrees.md`
mentions the dirty-base refusal only in the abstract; adding a pointer to
`codex-setup.md`'s new section there was considered but not done, since the
task's guidance was to add a pointer only where a recovery path is
described, and this recipe describes none.

## Verdict

Drift found and fixed: one missing `docs/tech-debt/README.md` row. All
other checked locations were already in sync with the shipped detector and
evidence record.
