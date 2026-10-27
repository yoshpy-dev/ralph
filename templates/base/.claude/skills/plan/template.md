# __TITLE__

- Status: Draft
- Approved: TBD
- Owner: Claude Code
- Date: __DATE__
- Related request: __REQUEST__
- Related issue: __ISSUE__
- Type: __TYPE__
- Branch: TBD

## Objective

## Scope

## Non-goals

## Assumptions

## Affected areas

## Visual review

<!-- Visual review page path (.harness/state/plan-visual/<slug>.html), the lint result, and the self-check result. See .claude/skills/plan/diagrams.md. -->
<!-- No page? Write: "None (<reason>)" -->

## Design decisions

<!-- Critical forks resolved with the user. Each entry: decision, chosen option, rationale. -->
<!-- No critical forks? Write: "Critical forks: None" -->

## Acceptance criteria

- [ ]

## Implementation outline

<!-- One numbered item per slice. End each item with that slice's verification command, written as "— verify: `<command>`". /implement copies the command into the implementer handoff. -->
<!-- The command is the narrowest check that covers the files the slice touches. For a Go package, its format, vet, and tests: `test -z "$(gofmt -l internal/foo)" && go vet ./internal/foo/ && go test ./internal/foo/` (gofmt -l exits 0 even when it lists files, hence test -z). For other languages, the language pack's per-package checks. -->
<!-- Do not narrow with RALPH_VERIFY_SCOPE=changed: it only selects which language packs run, and inside a pack (Go) it still runs the whole module (`go test ./...`). The full static and test checks run once, in /verify and /test. -->

1. <slice> — verify: `<command>`

## Verify plan

- Static analysis checks:
- Spec compliance criteria to confirm:
- Documentation drift to check:
- Evidence to capture:

## Test plan

- Unit tests:
- Integration tests:
- Regression tests:
- Edge cases:
- Evidence to capture:

## Risks and mitigations

## Rollout or rollback notes

## Open questions

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [ ] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
