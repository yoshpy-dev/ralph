# Plan: Fix known breakage — pack add layout + checkpoint triage key migration

- Status: approved
- Flow: standard (/work)
- Branch: fix/known-breakage
- Base: develop
- Date: 2026-07-12

## Objective

Fix the two confirmed-broken items tracked in `docs/tech-debt/README.md`:

1. `ralph pack add <lang>` renders pack files into the project root instead of
   `packs/languages/<lang>/`, clobbering root files (`internal/cli/pack.go:64-67`).
2. `internal/state.ReadPipelineCheckpoint` silently drops triage counts for
   checkpoints written before the `codex_review_triage` → `cross_review_triage`
   key rename (no migration path).

## Scope

- `internal/cli/pack.go` — render into `packRelDir(lang)`, namespace manifest
  hashes, handle `rule.md` control file, update `Meta.Packs`.
- `internal/cli/init.go` — extract the shared per-pack render logic if the
  extraction is clean; otherwise mirror it with a comment pointing at init.go.
- `internal/state/reader.go` — legacy-key fallback on checkpoint read.
- Tests for both fixes.
- `docs/tech-debt/README.md` — strike through both resolved entries.

## Non-goals

- No changes to the upgrade diff engine.
- No changes to shell scripts.

## Acceptance criteria

- AC1: `ralph pack add <lang>` in a tempdir project places all pack files under
  `packs/languages/<lang>/`, renders `rule.md` to `.claude/rules/<lang>.md`,
  and never writes pack payload files to the project root.
- AC2: manifest entries for pack files are namespaced with
  `packs/languages/<lang>/`, and `Meta.Packs` includes the added language.
- AC3: a checkpoint.json containing only the legacy `codex_review_triage` key
  loads with triage counts intact; a file with the new key is unaffected; a
  file with both keys prefers the new key.
- AC4: `go test ./...` passes; `./scripts/run-verify.sh` passes.
- AC5: tech-debt entries for both items are marked resolved with date.

## Verify plan

- `gofmt -l internal/ cmd/` (empty output)
- `go vet ./...`
- `go test ./... -count=1`
- `./scripts/run-verify.sh`

## Risks

- R1: pack add behavior change could break users relying on the (broken) root
  layout — accepted; the old layout is documented as broken.
- R2: init.go refactor could regress `ralph init` pack rendering — mitigated by
  existing init tests + new pack add test.

## Rollout

Single PR to `develop`. No migration needed; checkpoint fallback is read-time only.
