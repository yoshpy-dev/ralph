# Verify report: pr-ready-branch-type-enforcement

- Date: 2026-05-14
- Plan: `docs/plans/archive/2026-05-14-pr-ready-branch-type-enforcement.md`
- Verifier: Codex
- Evidence: `docs/evidence/verify-2026-05-14-pr-ready-branch-type-enforcement.log` (ignored raw log, local evidence)
- Verdict: **PASS**

## Acceptance criteria

| Criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: Claude Code and Codex standard PR instructions require ready-for-review PRs by default. | Verified | `.claude/skills/pr/SKILL.md` and `.agents/skills/pr/SKILL.md` require `gh pr create` without `--draft` unless explicitly requested and require `ensure-pr-ready.sh` before completion. |
| AC-2: Ralph Loop/pipeline PR paths verify draft state after PR creation. | Verified | `scripts/ralph-pipeline.sh` and `scripts/ralph-orchestrator.sh` call `scripts/ensure-pr-ready.sh` and fail closed if ready-state verification fails. |
| AC-3: Plan templates include branch type metadata and plan creation scripts can set it. | Verified | `docs/plans/templates/*` include `Type`; `new-feature-plan.sh` and `new-ralph-plan.sh` accept `--type`; regression tests cover generated metadata. |
| AC-4: Standard `/work` and `/loop` branch creation uses a shared generator. | Verified | Claude and Codex `work`/`loop` skills call `scripts/branch-name.sh from-plan` and validate generated branches. |
| AC-5: Branch validation accepts only controlled type prefixes for PR-facing branches. | Verified | `scripts/branch-name.sh validate` restricts prefixes to `feat fix docs chore refactor test ci build perf release security`; tests reject old `issue-*`, `codex-*`, `integration/*`, and `slice/*` shapes. |
| AC-6: Regression tests cover branch generation/validation and PR ready-state enforcement without network calls. | Verified | `tests/test-branch-name.sh` and `tests/test-ensure-pr-ready.sh` use temp fixtures and a stubbed `gh`. |
| AC-7: Root and `templates/base/` scaffold copies remain in sync. | Verified | `scripts/check-sync.sh` reports `DRIFTED: 0`; `scripts/check-skill-sync.sh` reports `13 skill(s) in lock-step`. |

## Static checks

| Command | Result |
| --- | --- |
| `./scripts/run-static-verify.sh > docs/evidence/verify-2026-05-14-pr-ready-branch-type-enforcement.log 2>&1` | PASS |
| `./scripts/check-sync.sh` | PASS |
| `./scripts/check-skill-sync.sh` | PASS |
| `sh -n scripts/branch-name.sh scripts/ensure-pr-ready.sh scripts/new-feature-plan.sh scripts/new-ralph-plan.sh scripts/ralph-pipeline.sh scripts/ralph-orchestrator.sh tests/test-branch-name.sh tests/test-ensure-pr-ready.sh` | PASS |
| `git diff --check` | PASS |

## Notes

- Raw evidence logs are intentionally ignored by `.gitignore`; this report records the local evidence path and pass verdict.
- `run-static-verify.sh` produced a timestamped nested evidence log as part of the standard verifier output: `docs/evidence/verify-2026-05-14-042643.log`.

## Verdict

All static verification and spec compliance checks passed.
