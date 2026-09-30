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

## Cycle 2

- Date: 2026-09-30
- Plan: `docs/plans/active/2026-09-30-codex-config-rewrite-detect.md`
  (Deviation notes: cross-review AR-1 + Slice D `ee397e9` — indented
  headers in the appended region now fall to the generic message; self-review
  cycle 2 + Slice E `a2c721d` — an at-least-one-known-change guard, single-quote
  root quoting for non-ASCII paths, and doc corrections; `ae96cd2` repoints
  the plan/evidence/tech-debt references at `docs/plans/archive/`)
- Branch: `fix/codex-config-rewrite-detect`, HEAD `3186527` (cycle 1 was
  `9b68c43`)

### Checked

- `docs/tech-debt/README.md`'s row from cycle 1: diffed `9b68c43..HEAD`.
  Slice E already repointed the evidence cell's plan reference from
  `docs/plans/active/2026-09-30-codex-config-rewrite-detect.md` to
  `docs/plans/archive/2026-09-30-codex-config-rewrite-detect.md` (valid once
  `/pr` archives it) — the only change to this row. The "what"/"impact"/"why
  deferred"/"trigger" cells describe the observed symptom and the
  detect-only scope at a level that does not enumerate the matcher's
  internals (indented headers, the at-least-one-known-change guard), so
  cycle 2's narrower matching does not make any of those cells stale.
  Column count unchanged: 6 pipes / 5 cells, matching the header.
- `docs/recipes/codex-setup.md` + `templates/base/docs/recipes/codex-setup.md`
  (`cmp` PASS) and `docs/evidence/codex-config-rewrite-2026-09-30.md`:
  diffed `9b68c43..HEAD`. Both were edited in Slice E and now state the
  at-least-one-known-change guard ("with at least one comment/blank line
  actually dropped or at least one policy table actually appended") and its
  negative cases (mode-only change, trailing newline added/removed, only
  blank/whitespace-only lines appended), matching the function comment and
  logic in `scripts/ralph-worktree.sh:89-166` (`dropped`,
  `saw_policy_header`, the final `if (!dropped && !saw_policy_header)`
  guard) and the die message's unchanged wording
  (`scripts/ralph-worktree.sh:187-204`; only the root-quoting mechanism
  changed, to single-quote escaping, which the recipe's `git -C` examples
  already render generically without depending on the quoting form). The
  ECC description was also corrected (Slice E, C2-4) to name the actual
  tables/keys it inserts (`[features]`, `[profiles.*]`, `[agents.*]`, root
  keys) instead of "append-only" — matches `merge-codex-config.js:27-34,295,300,313`
  as cited in the evidence file. Confirmed only, not rewritten, per this
  cycle's task.
  One minor completeness note, not treated as drift: the evidence file's
  "この調査が変えたこと" paragraph dropped its earlier explicit mention of
  "改行コードの変更" (line-ending/CRLF change) as an example of what falls
  to the generic message, while the recipe still lists it explicitly. Both
  statements remain true (case 11, CRLF-only, still asserts the generic
  message; `git diff main...HEAD -- scripts/ralph-worktree.sh` shows no
  change to CR handling in cycle 2) — the evidence file is just terser, not
  wrong. Left as Slice E wrote it.
- Indented-header behavior (Slice D, the actual AR-1 fix): confirmed present
  in `docs/recipes/codex-setup.md`'s new sentence "an appended table that
  is not `shell_environment_policy`" paragraph is unchanged in scope, but
  the negative-case list does not name "indented header" explicitly either
  — it was already covered by "an appended table that is not
  `shell_environment_policy`" reading loosely, and by the script comment's
  own explicit statement that an indented header, policy or not, falls to
  the generic message. Not a contradiction; the recipe's list was not
  required to enumerate every internal branch, and cycle 1's self-review
  standard for this recipe was end-to-end symptom/recovery accuracy, which
  holds.
- Re-confirmed unchanged from cycle 1 (re-grepped at HEAD `3186527`):
  `docs/recipes/worktrees.md`, `README.md`, `AGENTS.md`,
  `.claude/skills/plan/SKILL.md`, `.claude/skills/spec/SKILL.md` (+
  `.agents/skills/` and `templates/base/.claude/skills/` mirrors),
  `docs/evidence/README.md`. Same grep results as cycle 1 — no new hits,
  nothing stale.
- Gates: `./scripts/check-skill-sync.sh` (13 skills in lock-step),
  `./scripts/check-sync.sh` (159 IDENTICAL / 0 DRIFTED / 0 ROOT_ONLY /
  11 TEMPLATE_ONLY / 5 KNOWN_DIFF, PASS), `./scripts/check-template-purity.sh`
  (PASS). `cmp` on `docs/recipes/codex-setup.md` and
  `scripts/ralph-worktree.sh` against their template copies: both PASS.

### Stale

None found.

### Changed

None. The one thing this cycle's task asked to verify (the tech-debt row's
plan reference) was already fixed by Slice E before this cycle started.

### Left

None outstanding. The evidence file's dropped CRLF example (noted above) is
a completeness nit, not inaccurate, and was left as-is since this cycle's
task was to confirm, not rewrite, the recipe/evidence prose.

### Verdict

No drift found this cycle. No files changed.
