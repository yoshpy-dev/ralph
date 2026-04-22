# Test report: plan-critical-forks-step

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-plan-critical-forks-step.md` (docs-only)
- Tester: tester subagent
- Scope: Docs-only change adding Step 4.5 "Critical forks" to `/plan` SKILL and a "Design decisions" section to plan templates (`feature-plan.md`, `ralph-loop-manifest.md`) in both repo root and `templates/base/`. No code or runtime behavior touched.
- Evidence: `docs/evidence/test-2026-04-22-plan-critical-forks-step.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go test -count=1 ./...` (9 packages with tests) | all ok | all | 0 | pre-existing SKIPs only | ~4.4s slowest pkg |
| `./tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | <1s |

Per-package Go results (all `ok`, no `FAIL`):
- `internal/action` (4.236s)
- `internal/cli` (1.165s)
- `internal/config` (1.010s)
- `internal/scaffold` (1.366s)
- `internal/state` (1.724s)
- `internal/ui` (2.044s)
- `internal/ui/panes` (3.536s)
- `internal/upgrade` (2.398s)
- `internal/watcher` (4.378s)

No-test packages (expected, docs-only project root and cmd entrypoints): repo root, `cmd/ralph`, `cmd/ralph-tui`.

## Coverage

- Statement: not measured this run (no `-coverprofile` — docs-only change adds no new code paths)
- Branch: n/a
- Function: n/a
- Notes: This change modifies only Markdown skill files and plan templates. No new Go code, no new shell code, no new hook logic. Coverage is unchanged relative to prior baseline. Running the existing Go and shell suites confirms the docs change did not disturb any test fixtures, embedded templates, or `go:embed` paths.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `templates/base/` drift from root (sync-check) | Guarded by prior `/verify` run (`docs/evidence/verify-2026-04-22-plan-critical-forks-step.log` exit 0) | Verify report PASS |
| Mojibake guard on hook edits | Pass | `test-check-mojibake.sh` 11/11 PASS |
| Go template embedding unaffected by doc edits | Pass | `internal/scaffold` ok (1.366s) |

## Test gaps

No new behavioral tests are required for this change. Rationale:

1. The diff touches only `.md` files under `.claude/skills/plan/`, `docs/plans/templates/`, `.claude/rules/subagent-policy.md`, `CLAUDE.md`, and their `templates/base/` mirrors.
2. Plan-template content is consumed by the `/plan` skill at authoring time (read by an LLM), not parsed by code. There is no schema, no parser, no runtime contract to assert.
3. The root/templates-base sync invariant is already covered by `scripts/check-sync.sh` (exercised via `/verify`, PASS today).
4. The mojibake guard covers the risk that doc edits introduce U+FFFD — PASS today.
5. Adding unit tests that merely grep for the new heading text would be a tautology (they would assert the very string we just wrote) and would not catch any class of regression the existing sync-check and mojibake guard do not already catch.

If this skill change later grows a programmatic contract (e.g., a linter that asserts plans contain a "Design decisions" section), that would be the right moment to add tests.

## Verdict

- Pass: yes
- Fail: no
- Blocked: no

Safe to proceed to `/sync-docs` → `/codex-review` (optional) → `/pr`.
