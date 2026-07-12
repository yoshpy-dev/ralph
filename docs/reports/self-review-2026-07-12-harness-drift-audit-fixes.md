# Self-Review: harness-drift-audit-fixes

- Date: 2026-07-12
- Branch: `docs/harness-drift-audit-fixes` (base `main` @ 4d80723)
- Commit reviewed: `fa811a3` (docs-only, 8 files, +11/-5)
- Scope: diff quality only — factual accuracy of each statement vs the code it describes, wording, mirror consistency. No tests/static/spec/doc-drift/broad-audit.

## Summary

Docs-only change fixing six harness-drift audit findings. Every factual claim
introduced by the diff was cross-checked against the code it references. All
claims are accurate, both root+template mirrors are byte-identical, and the new
tech-debt row matches the table's 5-column shape.

Verdict: MERGE. No CRITICAL, HIGH, MEDIUM, or LOW findings.

## Findings

None.

## Evidence — claim-by-claim verification

| # | Claim in diff | Verification | Result |
|---|---------------|--------------|--------|
| 1 | Recipe note: `RALPH_PERMISSION_MODE` default is `bypassPermissions` via shell scripts (`scripts/ralph-config.sh`) | `scripts/ralph-config.sh:20` → `RALPH_PERMISSION_MODE="${RALPH_PERMISSION_MODE:-bypassPermissions}"` | Accurate |
| 2 | Recipe note: `auto` when launched via `ralph run` (value from `ralph.toml` `[pipeline] permission_mode`) | `internal/config/config.go:102` `Default()` → `PermissionMode: "auto"`; `templates/base/ralph.toml:9` → `permission_mode = "auto"`; `internal/cli/run.go:61` exports `RALPH_PERMISSION_MODE=`+`cfg.Pipeline.PermissionMode`, so `ralph run` overrides the shell default | Accurate |
| 3 | tech-debt row: shell default `bypassPermissions` vs toml/Go default `auto` divergence, with cited paths | Same three sources as #1/#2; all paths (`scripts/ralph-config.sh:20`, `templates/base/ralph.toml`, `internal/config/config.go Default()`) resolve correctly | Accurate |
| 4 | tech-debt row shape (5 columns) | Header at README.md:14 is 5 cells; new row has 5 cells (Debt item / Impact / Why deferred / Trigger / Related), no stray or escaped pipes | Matches |
| 5 | CLAUDE.md note: `anti-bottleneck` is `user-invocable: false`, belongs to neither the manual-trigger nor auto-invoked list | `.claude/skills/anti-bottleneck/SKILL.md` frontmatter has `user-invocable: false` and no `disable-model-invocation` key | Accurate |
| 6 | AGENTS.md: `docs/reports/` holds self-review, verify, test, sync-docs, cross-review triage, walkthrough artifacts | `docs/reports/` contains `self-review-*`, `test-*`, `sync-docs-*`, `cross-review-triage-*` files present | Accurate |
| 7 | repo-map.md: new `## Tests` section — `tests/` holds `tests/test-*.sh` + `tests/fixtures/` incl. `cli-stubs/` | `tests/` contains 29 `test-*.sh` files; `tests/fixtures/cli-stubs/` exists | Accurate |
| 8 | ralph.toml comment: pipeline reads prompts from `.claude/skills/loop/prompts/`; `dir` key reserved for future override | `ralph-pipeline.sh` / `ralph-orchestrator.sh` / `ralph-loop-init.sh` hardcode `.claude/skills/loop/prompts/`; the `.ralph/prompts` `dir` value is parsed into config (`config.go:104`) but not consumed by the shell pipeline | Accurate |

## Mirror consistency

- `AGENTS.md` vs `templates/base/AGENTS.md`: `cmp` → identical.
- `docs/recipes/ralph-loop.md` vs `templates/base/docs/recipes/ralph-loop.md`: `cmp` → identical.
- `CLAUDE.md`, `docs/architecture/repo-map.md`, `docs/tech-debt/README.md`, `templates/base/ralph.toml`: no mirror obligation (repo-map has no template counterpart; the others are edited only where they exist). Verified `templates/base/docs/architecture/repo-map.md` does not exist, so no missing mirror.

## Diff-quality checklist

- Unnecessary changes: none — every hunk maps to a named audit finding.
- Naming / grep-ability: n/a (prose); cited symbols (`RALPH_PERMISSION_MODE`, `permission_mode`, `Default()`) are grep-able and correct.
- Typos / copy-paste: none observed.
- Debug code / secrets / exception handling / security: n/a (docs-only, no executable change).

## Recommendation

MERGE. Docs-only, all factual claims verified against source, mirrors intact.
The permission-mode divergence itself is (correctly) recorded as tech debt
rather than resolved here, since choosing the canonical default is an
operator/product decision out of scope for a docs-drift fix.

## Known gaps

- Report confirms accuracy of statements only; whether the divergence *should*
  be resolved (making shell and Go defaults agree) is a product decision left
  to `/verify` and the tech-debt trigger, not this diff-quality pass.
