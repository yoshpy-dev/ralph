# sync-docs report: org-codex-slug-update-procedure

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-org-codex-slug-update-procedure.md` (issue #156)
- Prior reports: `docs/reports/self-review-2026-09-18-org-codex-slug-update-procedure.md`
  (Merge, no open findings), `docs/reports/verify-2026-09-18-org-codex-slug-update-procedure.md`
  (Pass), `docs/reports/test-2026-09-18-org-codex-slug-update-procedure.md` (Pass)

## Summary

This PR is docs-only: one operator paragraph added to `.claude/skills/org/SKILL.md`
(+ 3 mirrors, `.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`,
`templates/base/.agents/skills/org/SKILL.md`) under "### 既定の model_pool", and one
new `### 運用ノート: 既定 codex スラッグの更新手順` section added to
`docs/specs/2026-08-01-org-runtime.md`. No code, CLI, or default-value change.
`/self-review`, `/verify`, and `/test` already checked internal accuracy against
the code (mechanism claims, lock-step surfaces, mirror byte-identity) and found
zero open issues. This pass's job was the wider sweep: other docs that describe
the same mechanisms and might now be stale, inconsistent, or missing a pointer
to the new procedure. **No files were changed in this pass** — every candidate
surface was already consistent with the new prose, or correctly out of scope.

## Surfaces checked (no changes needed)

- **`docs/recipes/codex-setup.md`** (+ `templates/base/docs/recipes/codex-setup.md`,
  confirmed byte-identical by `diff`) — no mention of `model_pool`, model slugs,
  or `models_cache.json` at all (`grep -n 'model_pool\|models_cache\|slug'` finds
  nothing on-topic). Matches the plan's own Non-goals, which explicitly excludes
  this recipe: it is seed-once and a changeable procedure would not reach
  downstream projects if placed there.
- **`README.md`** — the `ralph doctor` row in the Commands table describes the
  check generically (core-hash/block/settings-key/manifest drift); it does not
  enumerate individual doctor sub-checks (codex slugs, hooks, etc.), so there is
  no per-check detail to go stale. No mention of `model_pool` or codex slugs
  anywhere in the file.
- **`AGENTS.md`** repo map — the `docs/specs/2026-08-01-org-runtime.md` entry is
  a one-line pointer ("spec: docs/specs/2026-08-01-org-runtime.md"), consistent
  with "AGENTS.md is a map, not an encyclopedia." No update needed for a new
  subsection inside a spec it already points at.
- **`.claude/rules/ralph/model-routing.md`** — "Org runtime model receipts"
  already carries a generic pointer ("see `docs/specs/2026-08-01-org-runtime.md`
  for the org runtime's own model selection rules") immediately before the
  sentence naming codex slugs and the doctor warn. That existing pointer already
  covers the new section since it lives in the same target file. The doc's own
  "Where the values live" list is scoped to the Claude model-tier table
  (`.claude/agents/*.md`, `ralph-config.sh` reviewer/cycle-cap vars), a
  different mechanism from org's `model_pool` lock-step, so it correctly does
  not enumerate `templates/base/ralph.toml` or `RALPH_ORG_MODEL_POOL`.
- **`docs/tech-debt/README.md`** — searched for any open row about codex slug
  staleness or issue #156 (`grep -n '156\|codex slug'`). The only two rows that
  ever discussed codex-slug-adjacent mechanics are both struck and marked
  `RESOLVED` (line 130, `org-implementer-seat-envelope` batch, closed
  2026-09-17; an unrelated `Verbs.Send/Wait/Read` coverage row at line 70). No
  open row exists to cross-reference from the new procedure, and this PR does
  not close #156 (per plan Non-goals), so no new row was added either — the
  plan's Design decisions already record that the observation loop and
  default-update PR are follow-on work tracked on the issue itself.
- **`templates/base/ralph.toml`** comments (lines 23-27) — already say codex
  slugs "go stale when codex retires a [model]" and point at
  `~/.codex/models_cache.json`; this is a *reason* callout, not a *procedure*,
  and does not contradict the new spec section's procedure. Left as is per the
  plan's own assumption that this file is seed-once and out of scope for a
  procedure that must reach downstream.
- **`docs/specs/2026-08-17-overlay-scaffold-v2.md`** — the L5 seed-once row
  (`ralph.toml` "欠落時のみ生成。テンプレート側が変わった場合は advisory diff
  の表示・レポート出力のみ(自動適用しない)") is exactly the seed-once/advisory-diff
  mechanism the new (c) distribution note in `docs/specs/2026-08-01-org-runtime.md`
  relies on. Confirmed consistent; not edited, per the task's explicit
  instruction to leave this file alone.
- **2026-09-16 revision note (c)** in `docs/specs/2026-08-01-org-runtime.md`
  (lines 7-13, immediately above the new section) states the current default
  pool contents and that `ralph doctor` gained the slug Check; the new
  "運用ノート" section (lines 15-22) documents *how to change* that default. No
  overlap or contradiction — (c) is a design decision record, the new section
  is an operating procedure for a future change.
- **`docs/plans/README.md`** — the only plans index in the repo; describes
  directory layout and naming conventions only, does not enumerate individual
  plans or specs. Nothing to update.
- **Plan's own Status/Progress checklist**
  (`docs/plans/active/2026-09-18-org-codex-slug-update-procedure.md:107-115`) —
  already accurate: everything through "Test artifact created" is checked, "PR
  created" is correctly unchecked (no PR exists yet). AC-3 (issue comment) and
  AC-5 (`Refs #156` in the PR body) are correctly unchecked, matching the
  plan's own Implementation outline (both belong to the `/pr` step). No edit
  needed.

## Mirror/sync verification

No skill-mirror content was changed in this pass, so `sync-skills.sh` was not
re-run. Re-confirmed the state left by `/work` and `/self-review` directly in
this worktree rather than trusting prior reports' word:

```
$ diff .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md            # exit 0, no output
$ diff .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md   # exit 0, no output
$ diff .claude/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md  # exit 0, no output
$ ./scripts/check-skill-sync.sh   # [ok] check-skill-sync: 13 skill(s) in lock-step
$ ./scripts/check-sync.sh         # PASS: all files in sync. IDENTICAL: 157, DRIFTED: 0, ROOT_ONLY: 0
```

All four skill copies remain byte-identical; both sync gates pass.

## Files changed in this pass

None. Every documentation surface the change could plausibly affect was
already consistent, or is correctly excluded (seed-once recipe, unrelated
generic doctor description, no open tech-debt row to cross-reference, a
maintainer-only spec left untouched per instruction).

## Known gaps

- Did not re-run `./scripts/run-verify.sh` or `go test ./internal/config/...`;
  that is `/verify`'s and `/test`'s completed job (both cited above, Pass).
- Did not post the issue #156 observation comment (AC-3) or author the PR body
  (AC-5) — both are explicitly `/pr`-step work per the plan's Implementation
  outline, not `/sync-docs` scope.
