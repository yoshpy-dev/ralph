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

## Cycle 2

- Verdict: **complete** — 1 residual drift item fixed (plan Deviations gap), 0 remaining

### Scope

Cycle-2 delta since the cycle-1 sync-docs pass: `d1df46f` (doctor warns on a
non-boolean `[features] hooks` value, cross-review AR#1), `bced11a`
(self-review cycle-2 fixes C2-M1/C2-M2/C2-L1–L5 — six doc surfaces reworded
to "false or non-boolean", `.codex/hooks/README.md` escape-hatch rewording,
the spec AC-10 footnote citation trimmed, a tech-debt row parenthetical
correction), and `67f56d5` (verifier-found orphaned sentence fragment in
`.codex/config.toml`'s hooks comment, fixed). Checked three things per the
lead's handoff: (1) any remaining "explicit false"-only claim about
`[features] hooks` doctor behavior that the six-surface sweep missed, (2) the
plan's `## Deviations` section for a cycle-2 bullet, (3) the tech-debt
register's edited row still renders as a well-formed table row.

### Findings and fixes

1. **Plan's `## Deviations` section** — missing a cycle-2 entry (cycle-1's
   sync-docs pass added the section but only covered cycle-1 events). Added a
   fourth bullet recording: cross-review AR#1 fixed inline in `d1df46f` under
   the trivial-edit exception; self-review cycle-2's 7 findings (2 MEDIUM,
   5 LOW) fixed in `bced11a`, with the touched-file list; the verifier-found
   dangling fragment ("`ralph doctor` and the shell / flags that...", left by
   the C2-L3 wording fix) fixed in `67f56d5`.
   `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md`.

### Checked, no drift found

- **"Explicit false"-only claim sweep** — `grep -rn` across `*.md`/`*.go`/
  `*.toml` for `hooks = false`, `[features] hooks`, and related phrasing
  found all six shipped surfaces (`.codex/README.md`, `.codex/config.toml`,
  `docs/recipes/codex-setup.md`, + their three `templates/base/` twins,
  already `cmp`-identical to their root counterparts per verify cycle-2)
  correctly stating "explicitly disabled or malformed" / "false or a
  non-boolean value" — no residual closed "only explicit false" phrasing.
  `.codex/AGENTS.override.md` and `README.md`'s comparison table still say
  `[features] hooks = true` in a config-trust/example context (not a
  doctor-behavior claim), which is accurate and out of scope for this sweep.
- **Tech-debt register row formatting** — the edited pre_bash_guard row
  (`docs/tech-debt/README.md`) and its neighbors all keep 7 pipe-delimited
  fields (`awk -F'|' '/^\|/ {print NF}'`, same method the verifier's C2-L5
  finding used); the RESOLVED hooks row (previously touched in cycle 1) is
  unaffected by the cycle-2 parenthetical edit to the adjacent row. No
  glued/broken rows introduced.
- `.codex/config.toml`'s hooks comment block, post-`67f56d5`, reads cleanly
  ("...do not reintroduce a `[hooks]` table here — `ralph doctor` flags that
  as a stale duplicate representation.") — root and template copies
  confirmed identical (`diff` clean).

### Files changed

- `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md` (Deviations
  cycle-2 bullet added)
- `docs/reports/sync-docs-2026-08-20-codex-hooks-json-wiring.md` (this
  report, Cycle 2 section)

### Verification

Docs-only change (plan + report). No source files touched, so
`./scripts/run-verify.sh` was not re-run — verifier's and tester's cycle-2
passes (`1b5cfa2`, `c288a81`) already cover static/spec/behavioral checks for
the code changes on this branch through `67f56d5`. Spot-checked the
tech-debt table row count with the same `awk -F'|' '/^\|/ {print NF}'`
approach the verifier's C2-L5 finding used; confirmed 7 fields on every data
row in the edited region.
