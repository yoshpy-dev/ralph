# scoped-verify-test

- Status: Draft
- Owner: Claude Code
- Date: 2026-05-14
- Related request: Scope post-implementation verify/test to changed languages
- Related issue: #71
- Branch: codex-scoped-verify-test

## Objective

Make post-implementation `/verify` and `/test` faster for multi-language
repositories by running only language packs affected by the current diff, while
retaining conservative full-scope gates for ambiguous changes, Ralph Loop
integration, and CI.

## Scope

- Add changed-file language detection with machine-readable scope output.
- Add `RALPH_VERIFY_SCOPE=full|changed` routing to `run-verify.sh`.
- Make `/verify` and `/test` wrappers default to changed-language scope.
- Keep `run-verify.sh` default and CI behavior full-scope.
- Keep Ralph Loop integration pipeline full-scope.
- Update tests, template mirrors, and docs.

## Non-goals

- Do not make PR creation run a full local verification gate.
- Do not replace per-language pack verification commands.
- Do not add a new language pack.
- Do not change CI beyond making full scope explicit.

## Assumptions

- `git` history is available for normal post-implementation runs.
- If no reliable diff base exists, full-scope verification is safer than
  skipping language packs.
- Project-local gates such as `scripts/verify.local.sh` are language-independent
  enough to run every time.

## Affected areas

- `scripts/run-verify.sh`, `scripts/run-static-verify.sh`, `scripts/run-test.sh`
- `scripts/detect-changed-languages.sh`
- `scripts/ralph-pipeline.sh`
- `templates/base/scripts/`
- `.claude/skills/`, `.agents/skills/`, `.claude/rules/`, `.claude/agents/`
- `docs/quality/`, `docs/recipes/`, `docs/architecture/`
- `tests/`
- `.github/workflows/verify.yml`

## Design decisions

Critical forks: None. The user already chose scoped local post-implementation
gates with CI and Ralph Loop integration retaining full-scope coverage.

## Acceptance criteria

- [ ] Post-implementation `/verify` uses changed-language static verification by default.
- [ ] Post-implementation `/test` uses changed-language behavioral tests by default.
- [ ] Language-independent gates continue to run on every post-implementation pass.
- [ ] Ambiguous or shared changes fall back to full verification and explain why.
- [ ] Ralph Loop integration pipeline remains full-scope.
- [ ] CI remains full-scope.
- [ ] PR creation does not run an additional full local verify/test solely as a pre-PR gate.
- [ ] Tests cover single-language changes, multi-language changes, docs-only changes, shared config changes, and ambiguous fallback behavior.
- [ ] Docs describe the scoped local loop versus full CI/integration gates.

## Implementation outline

1. Add a changed-language detector that emits scope, reason, docs-only flag, and selected languages.
2. Teach `run-verify.sh` to honor `RALPH_VERIFY_SCOPE`.
3. Make `/verify` and `/test` wrappers use changed scope unless callers override it.
4. Make Ralph Loop integration export full scope for `--skip-pr --fix-all`.
5. Add shell integration tests for detector and runner behavior.
6. Sync template, skill, rule, and quality-gate docs.

## Verify plan

- Static analysis checks: `sh -n` for changed shell scripts, `./scripts/run-static-verify.sh`.
- Spec compliance criteria to confirm: every acceptance criterion above with file and test evidence.
- Documentation drift to check: root/template mirrors, `.claude`/`.agents` skill sync, quality docs, language-pack recipe.
- Evidence to capture: `docs/evidence/verify-<date>-scoped-verify-test.log`.

## Test plan

- Unit tests: `tests/test-detect-changed-languages.sh`.
- Integration tests: `tests/test-run-verify-scope.sh`, `scripts/check-sync.sh`, `scripts/check-skill-sync.sh`.
- Regression tests: `tests/test-verify-mode-split.sh`, `go test ./...`.
- Edge cases: docs-only changes, shared CI files, unclassified files, committed branch diffs, no git repository.
- Evidence to capture: `docs/evidence/test-<date>-scoped-verify-test.log`.

## Risks and mitigations

- Risk: changed-language detection misses cross-language coupling. Mitigation:
  conservative full fallback for shared/unclassified files and full CI/integration gates.
- Risk: wrapper default surprises direct users. Mitigation: `run-verify.sh` remains
  backward-compatible full by default and docs call out `RALPH_VERIFY_SCOPE=full`.
- Risk: template drift. Mitigation: mirror changes and run sync checks.

## Rollout or rollback notes

- Rollout through normal PR and CI. Existing `./scripts/run-verify.sh` callers
  keep full-scope behavior by default.
- Rollback by reverting the PR; CI remains full-scope throughout.

## Open questions

- None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
