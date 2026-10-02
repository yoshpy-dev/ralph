# sync-docs report: drop-gpt-5-5-default-pool

## Cycle 1

- Date: 2026-10-02
- Plan: `docs/plans/active/2026-10-02-drop-gpt-5-5-default-pool.md` (issue #156)
- Pipeline cycle: 1 of 2. Delta: `main` `7dd6911c` to branch HEAD `b470340c`.
- Prior reports (the commits that added them): `docs/reports/self-review-2026-10-02-drop-gpt-5-5-default-pool.md`
  (`26b35264`, addendum `436d7ea6`; recommend merge),
  `docs/reports/verify-2026-10-02-drop-gpt-5-5-default-pool.md` (`1a24d0ea`; Pass, D-1..D-3),
  `docs/reports/test-2026-10-02-drop-gpt-5-5-default-pool.md` (`694ddf2a`; Pass, MEDIUM 1 / LOW 2; G-1 fixed by `c6d50f69`)

### Summary

The documentation shipped with the change (spec revision note (c), operations
note (d) wording and the new (e), the `/org` skill's table and recovery
paragraph in four copies, the template `ralph.toml`) matches the code at HEAD.
No other surface lists the default pool, counts its entries, or names
`gpt-5.5` as a default, so no documentation edit was needed. The one change in
this pass is three tech-debt rows (verify D-2, test G-2, test G-4), which the
earlier reports deferred to this step. No code, test, plan, or workflow file
was touched.

### Delta reviewed

`git diff main...HEAD`, documentation-relevant parts:

- `internal/config/config.go`, `scripts/ralph-config.sh`,
  `templates/base/scripts/ralph-config.sh`, `templates/base/ralph.toml`: the
  default `[org].model_pool` loses `codex gpt-5.5` and has 8 entries (claude
  `fable`/`opus`/`sonnet`/`haiku`, codex `gpt-6-astra`/`gpt-5.6-sol`/
  `gpt-5.6-terra`/`gpt-5.6-luna`). The codex fallback head stays `gpt-6-astra`.
- `.claude/skills/org/SKILL.md` and its three copies
  (`.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md`): the table loses the
  `gpt-5.5` row; a version-independent recovery paragraph is added (effective
  pool must contain every `[org.roles]` model; `spawn`/`start --model` rejection
  text; fix `ralph.toml` or pass a fixed copy with `--config`; state dir
  precedence `--state-dir` → `RALPH_ORG_STATE_DIR` → git toplevel → cwd).
- `docs/specs/2026-08-01-org-runtime.md`: revision note (c) lists the 8
  entries and points to (e); operations note (d)'s record field reads "既定の
  codex スラッグの有無" instead of "既定 5 スラッグの有無"; new (e) records the
  2026-10-02 maintainer decision, that no tag contained the `gpt-5.5` default,
  and the affected population.
- `internal/config/config_test.go`, `internal/cli/org_test.go`: tests only.
  Not documentation, not touched in this pass.

Claims in the shipped text that were re-checked against the repo in this pass
(not taken from the earlier reports):

| Claim | Check | Result |
|-------|-------|--------|
| v5.1.0 default pool is claude `opus`/`sonnet`/`haiku` only; no tag contains 3f9b4a01 | `git show v5.1.0:internal/config/config.go`; `git tag --contains 3f9b4a01` | 3 claude entries; 0 tags |
| Config is read from `--config`, else cwd `./ralph.toml`, else built-in default | `internal/cli/org.go:218-232` (`resolveOrgConfig`) | matches |
| Every verb loads the config (so `status`/`stop`/`disband` stop on a validation error) | `newOrgRuntime`/`newOrgRuntimeAt` callers in `internal/cli/org.go` (spawn, start, send, wait, read, stop, status, disband, report, watch) | matches; load error is returned as `org: load config: ...` |
| State dir order `--state-dir` → `RALPH_ORG_STATE_DIR` → git toplevel → cwd | `internal/org/statedir.go:47-58` | matches |
| `TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel` exists | `internal/cli/org_test.go:900` | present |

### Surfaces checked for drift (none found)

Grepped for `gpt-5.5`, `gpt-5.6-*`, `gpt-6-astra`, `model_pool`, `ModelPool`,
`MODEL_POOL`, "org codex model slugs", and entry/slug counts (`5 スラッグ`,
`9 エントリ`, `5 codex`, ...) across `README.md`, `AGENTS.md`, `CLAUDE.md`,
`docs/architecture/`, `docs/recipes/` and `templates/base/docs/`,
`docs/quality/`, `docs/specs/`, `docs/tech-debt/README.md`, `.claude/rules/`,
`.claude/skills/`, `.claude/agents/`, `.codex/`, `templates/base/ralph.toml`,
and the `templates/base/` copies of each.

- `README.md`, `AGENTS.md`: the org runtime is described by verbs and the spec
  pointer only; no pool list, no entry count. Still accurate.
- `docs/architecture/` (including `repo-map.md`): no mention of the pool or its
  slugs. Still accurate.
- `docs/recipes/codex-seat-permissions.md:64` (and the template copy): says "or
  `[org].model_pool` nothing is needed" in the context of codex presence in the
  pool, not its membership. Still accurate. No other recipe names a pool slug
  or `gpt-5.5`.
- `docs/quality/quality-gates.md:77` (and template `:76`): "model pool / role
  pool / `max_seats` checked by `ralph org spawn`". Behavior unchanged. Still
  accurate.
- `.claude/rules/ralph/model-routing.md:100-101` (and template): "`[org].model_pool`
  carries codex model slugs, and `ralph doctor` warns when a slug is missing".
  No list, no count. `gpt-6-astra` appears only as the pinned reviewer slug
  (`:55`, `:71`), which this change does not touch. Still accurate.
- `/org` skill outside the table and the new paragraph (all four copies):
  `前提` fallback head "claude は `fable`、codex は `gpt-6-astra`" (`:73`) is
  still the head of the 8-entry pool; the `doctor` description (`:88-100`) has no
  slug list. Still accurate.
- `templates/base/ralph.toml:17-37, 49-54`: the `driver_pool` narrowing comment
  and the `[org.roles]` example (`reviewer = ["opus", "gpt-6-astra"]`) both use
  slugs that remain in the pool. Still accurate. (`ralph.toml` has no root
  counterpart; `check-sync.sh` lists it as TEMPLATE_ONLY.)
- `docs/specs/2026-08-01-org-runtime.md` outside (c)/(d)/(e): `:17` ("既定
  `[org].model_pool` の codex エントリはスラッグ指定で…") and the operations
  note (a) enumeration of surfaces carry no count and no `gpt-5.5`. The note-(a)
  enumeration of hard-coded-default tests (`config_test.go`,
  `internal/org/envelope_summary_test.go`, `internal/cli/org_test.go`) is still
  accurate; `internal/org/envelope_summary_test.go` needed no change.
- `docs/tech-debt/README.md`: no existing row names the default pool entries.
- Remaining `git grep -n 'gpt-5\.5'` hits outside history
  (`docs/insights/events/`, `docs/reports/`, `docs/plans/`, `docs/evidence/`) and
  `*_test.go`: `.codex/config.toml` and its template (codex CLI's own `model`,
  Non-goal and plan Open question; the plan says to leave it), the 2026-05-07
  codex-parity spec (history of that decision), the spec's own (c)/(e) records,
  `scripts/verify.local.sh:83,93` and `tests/test-hook-wiring.sh`,
  `tests/test-ralph-worktree.sh` (fixtures that write a `.codex/config.toml`
  body; unrelated to `model_pool`), and a flag-parsing example comment in
  `internal/cli/doctor_shell_alias.go:627`. None describes the default pool.

### Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | Three rows appended (lines 143-145). Text below. |
| `docs/insights/events/2026-10-02-drop-gpt-5-5-default-pool.jsonl` | `sync_docs` event, cycle 1, verdict pass, via `./scripts/insights-append.sh`. |
| `docs/reports/sync-docs-2026-10-02-drop-gpt-5-5-default-pool.md` | This report. |

Root/template doc pairs: no root or template doc file with a counterpart was
edited in this pass. `docs/tech-debt/README.md` has no template counterpart
(only `docs/tech-debt/.gitkeep`). `./scripts/check-sync.sh` is green with the
branch's existing pairs.

### Tech-debt rows added

All three are pre-existing issues that this task surfaced, not introduced by
it. The file's five columns are: debt item, impact, why deferred, trigger to
pay down, related plan/report.

1. **Config and state dir resolve by different rules; teardown verbs stop on
   any config validation error** (verify D-2, self-review probe C).
   - Item: config is `--config`, else cwd `./ralph.toml`, else built-in default
     (`resolveOrgConfig`, `internal/cli/org.go`); state dir is `--state-dir`,
     else `RALPH_ORG_STATE_DIR`, else git toplevel, else cwd (`ResolveOrgStateDir`,
     `internal/org/statedir.go`). From a subdirectory without `--config`, a verb
     uses the built-in default config against the repo's real state dir (probe
     C: `status` exited 0 and listed `seat-1`). Every verb, including
     `status`/`stop`/`disband`, goes through `newOrgRuntime` → `resolveOrgConfig`,
     so a `ralph.toml` validation error blocks stopping running seats.
   - Impact: wrong envelope applied silently from a subdirectory; a broken
     `ralph.toml` leaves running seats reachable only after a fix or `--config`.
     Mitigated by the `/org` recovery paragraph and
     `TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel`.
   - Why deferred: predates this plan; its Non-goals excluded loosening the
     teardown verbs' validation.
   - Trigger: an incident report, or any change to `resolveOrgConfig` /
     `ResolveOrgStateDir`. Cross-references the existing
     `checkCodexAgmsgWritableRoot` row (b), which has the same missing upward
     search in `runDoctorFull(".")`.
   - Related: the plan, verify D-2, self-review probe C.
2. **`/org` skill's default-pool table is not tied to `config.Default()`**
   (test G-2). Four copies; `defaults_sync_test.go` locks only the three value
   surfaces; mutation t (restore the `gpt-5.5` row in all four copies) survived
   every gate. Trigger: the next default-pool change; add a test in the shape
   of `internal/org/send_defaults_sync_test.go`. The spec's operations note (a)
   already says no gate catches it.
3. **`TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry` indexes
   `events[len(events)-1]` unguarded** (test G-4, `internal/cli/org_test.go:711`).
   A regression panics and aborts the `internal/cli` package run (it did under
   mutation e). Trigger: the next touch to that test.

Not added, as instructed: no row for `.codex/config.toml`'s `model = "gpt-5.5"`.
That is an open maintainer question (plan Open questions, verify D-3), not debt.

Reference path note: row 1 cites the plan as `docs/plans/archive/2026-10-02-drop-gpt-5-5-default-pool.md`,
not `docs/plans/active/...`, because `/pr` archives the plan in this same PR
(Step 8) and `scripts/archive-plan.sh` does not rewrite references. If the plan
is not archived before merge, that one path dangles and needs repointing.

### Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | exit 0. IDENTICAL 159, DRIFTED 0, ROOT_ONLY 0, TEMPLATE_ONLY 11, KNOWN_DIFF 5 (unchanged set: `model-routing.md`, `verify.yml`, `CLAUDE.md`, `quality-gates.md`, `adding-a-language-pack.md`) |
| `./scripts/check-skill-sync.sh` | exit 0. 13 skills in lock-step |
| `./scripts/check-template-purity.sh` | exit 0. No meta-repo-specific references in templates |
| `./scripts/insights-append.sh --slug drop-gpt-5-5-default-pool --flow standard --phase sync_docs --cycle 1 --verdict pass --source skill` | exit 0; event appended |
| `./scripts/secret-scan.sh --staged` | recorded in the hand-off message (run at commit time) |
| `./scripts/secret-scan-branch.sh --strict` | recorded in the hand-off message (run before push) |

### Files changed in this pass

- `docs/tech-debt/README.md`
- `docs/reports/sync-docs-2026-10-02-drop-gpt-5-5-default-pool.md` (this report)
- `docs/insights/events/2026-10-02-drop-gpt-5-5-default-pool.jsonl`
  (sync_docs, cycle 1, verdict pass, via `./scripts/insights-append.sh`)
