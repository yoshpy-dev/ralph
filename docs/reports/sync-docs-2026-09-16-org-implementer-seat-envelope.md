# sync-docs report: org-implementer-seat-envelope

- Date: 2026-09-16
- Plan: `docs/plans/active/2026-09-16-org-implementer-seat-envelope.md`
- Prior reports: `docs/reports/self-review-2026-09-16-org-implementer-seat-envelope.md`,
  `docs/reports/verify-2026-09-16-org-implementer-seat-envelope.md` (PASS),
  `docs/reports/test-2026-09-16-org-implementer-seat-envelope.md` (PASS)

## Summary

The implementation's own Slice 5 (`docs: align org runtime docs`, part of HEAD)
had already performed the bulk of the documentation sync for this plan's
three behavior changes (budget removal, model_pool refresh + doctor codex
check + `--model` fallback rule, implementer seat + fan-out). This pass
audited every doc surface listed in the sync-docs task brief against the
current diff (`git diff main...HEAD`) and found no additional drift. The
only change made in this pass is ticking the plan's own Progress checklist.

## Files checked (no changes needed — already correct)

- `AGENTS.md` — repo map `internal/org/` line: no budget mention, "two-layer
  watchdog" still accurate (stall/liveness/scope-change/deadman unchanged).
- `README.md` — Org runtime section already says "a `lead` plus
  `implementer`, `reviewer`, and `qa` seats" (updated in this branch's Slice
  5 diff). Commands table's `ralph doctor` row is a general description and
  does not enumerate individual Check numbers, so the new "Org codex model
  slugs" check needs no row edit. No `--model`/`--probe-models` flag text in
  README to update.
- `docs/recipes/*.md` — `worktrees.md`'s "Org runtime" section and
  `agent-teams.md` reference org runtime only at the manifest/worktree level;
  no budget or model_pool enumeration to drift.
- `docs/quality/definition-of-done.md` — org runtime section points at the
  spec by reference only; no budget/model_pool content to drift.
- `.claude/rules/ralph/*.md` — `agent-messaging.md` (seat-id example now
  `implementer`, `reviewer`, `qa`) and `model-routing.md` (`fable` alias
  added to the stable-alias list; org receipts paragraph now notes codex
  seats have no aliases and `ralph doctor` warns on missing slugs) already
  reflect the new state. `post-implementation-pipeline.md`,
  `subagent-policy.md`, `git-commit-strategy.md`, `testing.md`,
  `architecture.md`, `ralph-workflow.md` are unaffected by this plan's scope
  (standard `/work` pipeline and `.claude/agents/` were explicit non-goals).
- `.codex/README.md` / `.codex/agents/*.toml` — these define the standard
  `/work` pipeline's Codex custom agents (`implementer`, `reviewer`,
  `tester`, `verifier`, `doc-maintainer`), a distinct concept from the org
  runtime's `internal/org/prompts/*.md` seat templates. Explicitly out of
  scope per the plan's non-goals ("標準フロー…や `.claude/agents/` の変更").
  No drift.
- `docs/insights/README.md` — does not document budget-related events; no
  drift.
- `.ralph/core/AGENTS.core.md` — org runtime line ("coordinating `lead` plus
  role seats") is generic and does not enumerate specific roles or budget;
  no drift.
- `internal/cli/doctor.go` — the new `checkCodexModelSlugs` Check has no
  CLI flag (unconditional, no `--help` text to keep in sync); the
  `--probe-models` flag help text is unchanged. Nothing in README's Commands
  table needs updating in response.
- `docs/architecture/repo-map.md` — does not enumerate individual Go files
  under `internal/cli/` or `internal/org/`, so `doctor_codex_models.go` and
  `prompts/implementer.md` need no entry.
- `docs/specs/2026-08-01-org-runtime.md`, `docs/quality/quality-gates.md`,
  `docs/tech-debt/README.md` — already carry the 2026-09-16 revision notes
  (FR-2/7/8/10/NFR annotated in place, budget-enforcement gate row replaced
  with a watchdog-pulse-layer row, tech-debt row for the Cutoff ratchet
  marked RESOLVED, and a new open row added for `StopParams.Reason` having
  no production producer after the removal — self-review MEDIUM-3).
- `templates/base/**` mirrors of all of the above — verified in sync (see
  Verification below); the two pre-existing root/template KNOWN_DIFFs
  (`docs/quality/quality-gates.md`'s extra `check-sync.sh` bullet,
  `.claude/rules/ralph/model-routing.md`'s extra
  `internal/org/receipts.go`/`defaults_sync_test.go` path references) are
  meta-repo-only additions unrelated to this plan and were not widened by
  it — both files' budget/model_pool content is identical between root and
  template.

## Drift found and fixed

- Plan Progress checklist: ticked "Review artifact created", "Verification
  artifact created", "Test artifact created" (all three report files exist
  under `docs/reports/`). "PR created" left unticked per task instructions.

No other drift found.

## Drift intentionally left in place

- `templates/base/README.md` and `templates/base/AGENTS.md` do not exist as
  separate files (only `templates/base/AGENTS.md` exists; there is no
  `templates/base/README.md` — the scaffold's README is the repo root
  README, not templated). The task brief's grep command referencing
  `templates/base/README.md` therefore errors (`No such file or directory`)
  rather than finding a hit; this is a non-existent path, not a drift
  finding.
- Root/template KNOWN_DIFFs in `docs/quality/quality-gates.md` and
  `.claude/rules/ralph/model-routing.md` (see above) predate this plan and
  are unrelated to budget/model_pool/implementer content — `./scripts/check-sync.sh`
  reports them as `KNOWN_DIFF`, not `DRIFTED`, confirming they are the
  expected meta-repo-only deltas.

## Verification commands run

```
./scripts/sync-skills.sh          # [sync-skills] done: 13 skill(s) mirrored to .agents/skills
./scripts/check-skill-sync.sh     # [ok] check-skill-sync: 13 skill(s) in lock-step
./scripts/check-sync.sh           # PASS: all files in sync. (DRIFTED: 0, KNOWN_DIFF: 5 all pre-existing/unrelated)
./scripts/check-pipeline-sync.sh  # [ok] all 7 referencing docs list every pipeline step
sh tests/test-no-loop-references.sh  # PASS: no live references to the retired Ralph Loop system
```

```
grep -rn -i 'budget' README.md AGENTS.md docs/recipes docs/quality \
  .claude/rules .codex templates/base/README.md templates/base/AGENTS.md \
  templates/base/docs 2>/dev/null
# no hits; exits 2 only because templates/base/README.md does not exist
# (confirmed with `ls`), not because of a grep match anywhere
```

```
grep -rln -i 'budget' --include='*.md' --include='*.toml' --include='*.sh' . \
  | grep -v -E '^\./(docs/plans/archive|docs/reports|docs/specs|docs/tech-debt/README.md|\.git)/'
# ./.agents/skills/audit-harness/SKILL.md            -- "AGENTS.md size budget", unrelated concept
# ./.claude/skills/audit-harness/SKILL.md             -- same, unrelated concept
# ./docs/plans/active/2026-09-16-org-implementer-seat-envelope.md  -- this plan's own body (expected)
# ./docs/tech-debt/README.md                          -- resolved + new rows, expected (annotated above)
# ./templates/base/.agents/skills/audit-harness/SKILL.md  -- mirror of the above, unrelated
# ./templates/base/.claude/skills/audit-harness/SKILL.md  -- mirror of the above, unrelated
```

All hits outside `docs/plans/`, `docs/reports/`, `docs/specs/`, and
`docs/tech-debt/README.md` are the `audit-harness` skill's "AGENTS.md size
budget" concept (token/line budget for the AGENTS.md file), unrelated to the
org runtime `[org.budget]` concept removed by this plan. Left in place.

## Files changed by this sync-docs pass

- `docs/plans/active/2026-09-16-org-implementer-seat-envelope.md` (Progress checklist)
- `docs/reports/sync-docs-2026-09-16-org-implementer-seat-envelope.md` (this report)

## Cycle 2 (2026-09-17)

- Reconciled behavior change from commits `0d41553` (`fix: keep driver_pool-only overrides loadable with codex default models`) and `79bcb96` (`fix: name driver_pool when an inherited model_pool filters to empty`): `config.Load()` now, when `[org].driver_pool` is set but `[org].model_pool` is not, filters the inherited default `model_pool` down to the declared drivers; if that leaves it empty, `Load()` errors with `[org].driver_pool <list> has no default [org].model_pool entries; set [org].model_pool explicitly` naming the key the document actually wrote. An explicit `model_pool` skips the filter and is validated strictly as before (unchanged).

### Files checked, no drift found

- `templates/base/ralph.toml` — the `[org].driver_pool` comment block (added by `0d41553`) already states the filtering behavior and matches the current error text and `Load()` logic verbatim (`internal/config/config.go:250-256`). No edit needed.
- `.claude/skills/org/SKILL.md` (+ 3 mirrors) — the 既定 model_pool table documents the shipped defaults only; it makes no claim about partial-override filtering, so there is nothing in it to contradict the new behavior. Left as-is.
- `docs/recipes/*.md`, `README.md` — no `driver_pool`/`model_pool` mentions (`grep -rn` empty); no drift.
- `.claude/rules/ralph/model-routing.md` — already carries the codex-slug/no-alias note from cycle 1; it does not describe the `driver_pool`/`model_pool` relationship, so cycle-2's filter behavior does not touch it. Left as-is.

### Drift found and fixed

- `docs/specs/2026-08-01-org-runtime.md` — the 2026-09-16 revision block (bullets a-d) predates the cycle-2 fix and didn't record the driver_pool-only override compatibility behavior. Added bullet `(e)` describing the filter-then-validate behavior, the empty-pool error naming `driver_pool`, and both commit SHAs.

### Tech debt

- Appended one open row to `docs/tech-debt/README.md` batching the deferred LOW findings still open per `docs/reports/self-review-2026-09-16-org-implementer-seat-envelope.md`: cycle-1 LOW findings (`doctor_codex_models.go` home-dir error, `DefaultModelForDriver` stale doc, `RolePromptVars.PlanPath` missing `implementer.md`, `/org` skill Leaded-row contradiction, two over-claiming test assertions, `lead.md` 常体/敬体 mix, `OrgModelPoolEntry` "later slice" doc, `pruneRetiredConditions` redundant delete, stale 3-entry envelope fixture) plus cycle-2 C2-L2 (two under-named/under-asserted `driver_pool` filter regression tests). C2-M1 and C2-L1 are not included — both were already fixed in commit `79bcb96`. This closes the cycle-2 verify follow-up referencing C2-L3.

### Verification commands run (cycle 2)

```
./scripts/sync-skills.sh          # [sync-skills] done: 13 skill(s) mirrored to .agents/skills
./scripts/check-skill-sync.sh     # [ok] check-skill-sync: 13 skill(s) in lock-step
./scripts/check-sync.sh           # PASS: all files in sync. (DRIFTED: 0, KNOWN_DIFF: 5, all pre-existing)
./scripts/check-pipeline-sync.sh  # [ok] all 7 referencing docs list every pipeline step
sh tests/test-no-loop-references.sh  # PASS: no live references to the retired Ralph Loop system
```

### Files changed by this cycle-2 sync-docs pass

- `docs/specs/2026-08-01-org-runtime.md` (revision block bullet (e))
- `docs/tech-debt/README.md` (new deferred-LOW-findings batch row)
- `docs/plans/active/2026-09-16-org-implementer-seat-envelope.md` (Cycle 2 section note, Commits)
- `docs/reports/sync-docs-2026-09-16-org-implementer-seat-envelope.md` (this section)
