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
<!-- The command is the narrowest check that covers the files the slice touches. For a Go package, its format, vet, and tests: `test -z "$(gofmt -l internal/foo)" && go vet ./internal/foo/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/foo/` (gofmt -l exits 0 even when it lists files, hence test -z). The -timeout=30m first in GOFLAGS is the Go language pack's default, so a package whose tests take over ten minutes is not cut off by go test's 10-minute default, and a timeout the caller set (in GOFLAGS or with go env -w) comes after it and wins. For a package in a nested module (its own go.mod below the repository root), run the command inside that module's directory with paths relative to it, because without a root go.work Go does not enter a nested module from the root: `(cd service && test -z "$(gofmt -l pkg)" && go vet ./pkg/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./pkg/)`. For other languages, the language pack's per-package checks. -->
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
