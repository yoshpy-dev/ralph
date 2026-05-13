# Verify report: scoped-verify-test

- Date: 2026-05-14
- Plan: `docs/plans/active/2026-05-14-scoped-verify-test.md`
- Scope: spec compliance + static analysis + documentation drift
- Evidence: `docs/evidence/verify-2026-05-13-173931.log`
- Verdict: pass

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| Post-implementation `/verify` uses changed-language static verification by default. | met | `scripts/run-static-verify.sh` now exports `RALPH_VERIFY_SCOPE=changed` by default; `.agents/skills/verify/SKILL.md` and mirrored Claude/template skill docs describe changed-language scope. |
| Post-implementation `/test` uses changed-language behavioral tests by default. | met | `scripts/run-test.sh` now exports `RALPH_VERIFY_SCOPE=changed` by default; `tests/test-run-verify-scope.sh` confirms the wrapper runs only the changed Go pack in the fixture. |
| Language-independent gates continue to run on every post-implementation pass. | met | `scripts/run-verify.sh` still runs `scripts/verify.local.sh` before language-pack selection; `tests/test-run-verify-scope.sh` asserts `local:*:changed` runs in changed and full-fallback cases. |
| Ambiguous or shared changes fall back to full verification and explain why. | met | `scripts/detect-changed-languages.sh` emits `scope=full` with reasons such as `shared:<path>` and `unclassified:<path>`; detector and runner tests cover shared CI and unknown-file fallback. |
| Ralph Loop integration pipeline remains full-scope. | met | `scripts/ralph-pipeline.sh` exports `RALPH_VERIFY_SCOPE=full` for `--skip-pr --fix-all` unless the caller overrides it. |
| CI remains full-scope. | met | `.github/workflows/verify.yml` sets `RALPH_VERIFY_SCOPE: full` for the verification job; `run-verify.sh` also defaults to full. |
| PR creation does not run an additional full local verify/test solely as a pre-PR gate. | met | PR skill pre-checks still require reports/evidence and do not add a full verify/test command; no new PR-before-create full gate was added. |
| Tests cover single-language changes, multi-language changes, docs-only changes, shared config changes, and ambiguous fallback behavior. | met | `tests/test-detect-changed-languages.sh` covers all listed cases; `tests/test-run-verify-scope.sh` covers runner integration and full fallback. |
| Docs describe the scoped local loop versus full CI/integration gates. | met | Updated `docs/quality/quality-gates.md`, `docs/quality/definition-of-done.md`, `.claude/rules/post-implementation-pipeline.md`, `.claude/rules/subagent-policy.md`, and mirrors. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `sh -n scripts/detect-changed-languages.sh templates/base/scripts/detect-changed-languages.sh scripts/run-verify.sh templates/base/scripts/run-verify.sh tests/test-detect-changed-languages.sh tests/test-run-verify-scope.sh` | pass | Shell syntax is valid. |
| `scripts/check-sync.sh` | pass | Root/template mirrors are in sync; known diffs only for expected files. |
| `scripts/check-skill-sync.sh` | pass | 13 skill bodies remain in lock-step. |
| `./scripts/run-static-verify.sh` | pass | Changed scope correctly fell back to full because this PR changes `.github/workflows/verify.yml`; Go verifier passed. |
| `go test ./...` | pass | Re-run with elevated permissions because Go cache writes are outside the sandbox. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.agents/skills/verify` / `.claude/skills/verify` | yes | Changed-scope default documented. |
| `.agents/skills/test` / `.claude/skills/test` | yes | Changed-scope default documented. |
| `.claude/rules/post-implementation-pipeline.md` | yes | Phase responsibility and integration full-scope behavior documented. |
| `docs/quality/quality-gates.md` | yes | Local changed scope versus CI full scope documented. |
| `docs/recipes/adding-a-language-pack.md` | yes | New language packs must update both full and changed-language detectors. |

## Coverage gaps

- No live Ralph Loop run was performed; behavior is verified by `scripts/ralph-pipeline.sh` static inspection and shell tests around the verify/test scripts.

## Verdict

- Verified: acceptance criteria, static checks, docs sync, skill sync.
- Partially verified: Ralph Loop integration full-scope behavior by code inspection rather than a full orchestrator run.
- Not verified: CI result after PR creation.
