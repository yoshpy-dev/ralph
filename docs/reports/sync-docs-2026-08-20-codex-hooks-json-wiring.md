# Sync-docs report — codex-hooks-json-wiring (cycle 1)

- Plan: `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md`
- Verdict: **complete** — 3 residual drift items fixed, 0 remaining

## Scope

Slice 3 (2bda3a1) and the self-review fix pass (c72e644, 7af720a) already
updated the deliberate doc set for this task: `.codex/README.md`,
`.codex/hooks/README.md`, `docs/recipes/codex-setup.md`, `README.md`
Codex-vs-Claude table, `CLAUDE.md`, `AGENTS.md`, `ralph-workflow.md`,
`AGENTS.override.md`, and both tech-debt rows. The verifier's consistency
sweep (`docs/reports/verify-2026-08-20-codex-hooks-json-wiring.md`,
commit `10432f6`) found zero residual stale claims in its scope. This pass
covers what those sweeps didn't: the historical spec, `docs/architecture/`,
`docs/quality/`, `docs/insights/README.md`, `.claude/rules/ralph/*.md`, and
the plan's own Deviations bookkeeping.

## Findings and fixes

1. **`docs/architecture/repo-map.md:25`** — the `.codex/` file listing
   omitted `hooks.json` (still described the surface as `config.toml,
   agents/, hooks/, AGENTS.override.md, README.md`), even though `AGENTS.md`
   in this worktree already lists `hooks.json` as the hooks source of truth.
   Fixed: added `hooks.json` to the listing plus a clause noting it routes
   `PostToolUse` through `ralph-dispatch.sh` while `config.toml` keeps only a
   reference comment.
2. **`README.md:272`** — the "Codex-native" file-set bullet in the
   Portability section had the same omission (`hooks.json` missing from the
   Codex-native file list). Fixed: added `.codex/hooks.json` to the list.
3. **`docs/specs/2026-08-17-overlay-scaffold-v2.md:92`** (AC-10 footnote) —
   the parenthetical still read "Codex 側は dispatcher 配線済みだが
   project-scoped hooks の実発火確認が trust 制約で未了", which this task's
   live-fire evidence now contradicts (non-bypass fire confirmed in this
   meta-repo, bypass fire confirmed in a fresh `ralph init` fixture). Left
   the historical claim in place (this is a completed spec, not live
   documentation) and appended a bracketed 2026-08-20 pointer to the
   resolving evidence (`docs/plans/active/2026-08-20-codex-hooks-json-wiring.md`
   AC-2, `docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md`) rather
   than rewriting the AC-10 text itself.
4. **Plan's `## Deviations` section** — missing (per known template gap;
   `docs/plans/templates/feature-plan.md` has no `Deviations` heading).
   Added one between "Rollout or rollback notes" and "Design decisions"
   recording: (a) the 11 self-review findings (0 CRITICAL/1 HIGH/5
   MEDIUM/5 LOW) fixed in `c72e644`, (b) the `.codex/AGENTS.override.md`
   follow-up fix in `7af720a` (outside the plan's original Affected areas
   list), (c) AC-2(b) live-fire evidence turning out stronger than planned
   — the fresh, untrusted `ralph init` fixture fired under
   `--dangerously-bypass-hook-trust`, not just the pre-trusted meta-repo
   path.

## Checked, no drift found

- `docs/quality/` — no `config.toml` hooks claims, no `[[hooks.*]]` or
  `[hooks]` mentions.
- `docs/insights/README.md` — no hook-wiring mentions (out of scope for this
  doc; confirmed no stale drift).
- `.claude/rules/ralph/*.md` — no residual TOML hook-wiring claims (the
  `ralph-workflow.md` and root `documentation.md` copies already say
  `.codex/hooks.json` per the Slice 3 pass).
- `docs/tech-debt/README.md` rows 104/115/121/122 — already RESOLVED with
  accurate resolution text from the Slice 3 + self-review fix commits; no
  further changes needed.

## Files changed

- `docs/architecture/repo-map.md`
- `README.md`
- `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md` (Deviations
  section added)
- `docs/reports/sync-docs-2026-08-20-codex-hooks-json-wiring.md` (this
  report)

## Verification

Docs-only change; no code/behavior touched. Spot-checked with `git diff`
that each edit is a targeted addition (no unrelated rewrites). Did not
re-run `./scripts/run-verify.sh` (no source files changed) — verifier's
cycle-1 pass (`10432f6`) already covers static/spec compliance for the code
changes on this branch.
