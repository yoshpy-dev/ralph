# codex-env-scaffold

- Status: Draft
- Owner: Codex
- Date: 2026-05-13
- Related request: codex-env-scaffold
- Related issue: 56
- Branch: issue-56-codex-env-scaffold

## Objective

Add upstream-safe Codex agent role configuration for the standard ralph
post-implementation flow and make root/template sync detect the new Codex
surface.

## Scope

- `.codex/agents/{reviewer,verifier,tester,doc-maintainer}.toml`
- `templates/base/.codex/agents/...`
- Root/template Codex docs as needed
- Sync/scaffold tests that should know this surface exists
- Existing main dirty `.codex/agents/*` files, adapted for upstream
- Plan/review/verify/test/sync-docs artifacts

## Non-goals

- Porting downstream app-specific Go/Python stack skills into ralph root.
- Codex hook single-source cleanup; tracked by issue #57.
- Redesigning the full CI/PR automation pipeline.

## Assumptions

- Codex agent TOML files are configuration assets for future/available Codex
  agent role selection; the standard Codex flow still runs inline when no
  subagent mechanism is available.
- Language-specific guidance belongs in `packs/languages/` or project-local
  customization, not in root `.agents/skills/`.

## Affected areas

- Codex scaffold surface
- Template embedding/init manifest behavior
- Sync verification

## Design decisions

Critical forks: None. The issue explicitly allows adopting agent role files
when their responsibilities match ralph's standard flow.

## Acceptance criteria

- [ ] Codex role files exist for reviewer, verifier, tester, and doc-maintainer.
- [ ] Role responsibilities match the standard pipeline contracts.
- [ ] Matching files are shipped from `templates/base/.codex/agents/`.
- [ ] Sync/scaffold checks cover the new Codex agent surface.
- [ ] No private path, downstream repo name, or app-specific stack guidance is included.
- [ ] Existing dirty `.codex/agents/*` content is incorporated where suitable.

## Implementation outline

1. Add upstream-safe Codex role TOML files to root and template.
2. Update docs/tests to mention and verify `.codex/agents/`.
3. Extend sync checks so future root `.codex` drift is caught.
4. Run static and targeted Go tests.

## Verify plan

- Static analysis checks: `scripts/check-sync.sh`, `git diff --check`.
- Spec compliance criteria to confirm: each acceptance criterion above.
- Documentation drift to check: README/AGENTS/Codex README scaffold map.
- Evidence to capture: static verify and targeted Go test logs.

## Test plan

- Unit tests: `go test ./internal/scaffold ./internal/cli`.
- Integration tests: `./scripts/run-static-verify.sh`.
- Regression tests: `scripts/check-sync.sh`.
- Edge cases: `ralph init` manifest tracks Codex agent files.
- Evidence to capture: verify/test reports with pass verdicts.

## Risks and mitigations

- Risk: Codex role files imply subagent behavior that is not always available.
  Mitigation: document them as role configuration while preserving inline
  fallback semantics.
- Risk: root/template drift. Mitigation: update sync scan and template tests.

## Rollout or rollback notes

Rollback by removing the `.codex/agents/` files and related docs/tests.

## Open questions

None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created: https://github.com/yoshpy-dev/ralph/pull/60
