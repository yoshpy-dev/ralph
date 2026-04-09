# Test report: Ralph Loop v2 — Plan System Redesign

- Date: 2026-04-10
- Plan: docs/plans/archive/2026-04-09-ralph-loop-v2.md
- Tester: tester subagent (claude-sonnet-4-6)
- Scope: feat/ralph-loop-v2 branch — new-ralph-plan.sh, ralph-orchestrator.sh (directory plan mode), archive-plan.sh (directory mode), ralph-loop-init.sh (pipeline mode), ralph-pipeline.sh (regression), ralph CLI
- Evidence: `docs/evidence/test-2026-04-10-ralph-loop-v2.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (canonical) | 0 | 0 | 0 | 0 (scaffold-only) | < 1s |
| Unit: new-ralph-plan.sh directory structure | 1 | 1 | 0 | 0 | < 1s |
| Unit: new-ralph-plan.sh content substitution | 1 | 1 | 0 | 0 | < 1s |
| Unit: ralph-orchestrator.sh --dry-run --plan \<dir\> | 1 | 1 | 0 | 0 | < 1s |
| Unit: archive-plan.sh \<directory\> | 1 | 1 | 0 | 0 | < 1s |
| Unit: ralph-loop-init.sh --pipeline \<fullpath\> sets plan_path | 1 | 1 | 0 | 0 | < 1s |
| Regression: ralph-pipeline.sh --dry-run --max-iterations 3 | 1 | 1 | 0 | 0 | ~2s |
| Regression: ralph --help exits 0 | 1 | 1 | 0 | 0 | < 1s |
| Regression: ralph-pipeline.sh --help exits 0 | 1 | 1 | 0 | 0 | < 1s |
| Edge: ralph-loop-init.sh empty plan_slug | 1 | 1 | 0 | 0 | < 1s |
| Edge: ralph-orchestrator.sh with 0 slice-*.md files | 1 | 1 | 0 | 0 | < 1s |
| Edge: ralph-orchestrator.sh --plan \<single-file\> rejected | 1 | 1 | 0 | 0 | < 1s |
| **TOTAL** | **11** | **11** | **0** | **0** | **~5s** |

## Coverage

- Statement: N/A (shell scripts, no coverage tool)
- Branch: Key branches manually exercised: directory vs. file plan resolution, 0-slice error path, single-file rejection, pipeline mode vs. standard mode, empty plan_slug
- Function: All major entry points tested: new-ralph-plan.sh, ralph-orchestrator.sh (dry-run), archive-plan.sh (directory), ralph-loop-init.sh (--pipeline, with and without plan_slug), ralph-pipeline.sh (--dry-run), ralph (--help)
- Notes: Live `claude -p` paths are untestable without API access; covered by prior dry-run evidence

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures. All 11 tests passed.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `ralph --help` exits 0 (fixed 2026-04-09) | PASS | exit code 0 observed |
| `ralph-pipeline.sh --help` exits 0 (fixed in pipeline-robustness) | PASS | exit code 0 observed |
| `ralph-pipeline.sh --dry-run --max-iterations 3` completes successfully | PASS | exit code 0, status: complete |

## Test gaps

1. **Live API paths**: `ralph-pipeline.sh` Inner Loop and Outer Loop with real `claude -p` calls — untestable without API access. Covered by prior integration evidence (`docs/evidence/test-2026-04-09-pipeline-robustness-r2.log`).
2. **Multi-worktree execution**: `ralph-orchestrator.sh` actual worktree creation and parallel execution — dry-run validates plan parsing; live worktree creation requires non-trivial git state setup.
3. **Dependency-ordered slice execution**: The orchestrator's sequential execution logic for slices with `dependencies:` defined is not exercised by dry-run — slice dependency parsing produces correct output but execution sequencing is untested at runtime.
4. **Stuck detection 3-cycle threshold**: Already noted in MEMORY.md as not fully exercisable in dry-run. Not applicable to this test plan's scope (no new stuck detection code in ralph-loop-v2).
5. **`new-ralph-plan.sh --help` exits 1**: By design (usage() exits 1). Not in scope for this test plan, but noted for completeness.
6. **`archive-plan.sh` overwrite guard**: `archive-plan.sh` rejects if archive already has same name — not tested here (covered implicitly by prior behavior).

## Observations

- Test 8 (0 slice edge case) initially appeared to exit 0 due to `$?` capture timing in chained command. Confirmed on direct re-run: `exit 1` is correctly triggered when 0 slice files are found in the plan directory.
- Test 2 (dry-run) side effect: `ralph-orchestrator.sh --dry-run` creates an integration branch despite being a dry run. This is expected behavior as documented in the orchestrator (integration branch creation precedes the dry-run check), but is worth noting for test cleanup hygiene.
- `run-test.sh` canonical runner returns exit 0 with "No language verifier ran" because the repo is shell-script scaffold level. All behavioral tests are manual shell execution.

## Verdict

- Pass: 11 / 11
- Fail: 0 / 11
- Blocked: 0 / 11

**PASS — all test plan items verified. Branch feat/ralph-loop-v2 is clear for PR creation.**
