# Walkthrough: Codex CLI parity

- Date: 2026-05-07
- Plan: docs/plans/active/2026-05-07-codex-cli-parity.md
- Branch: feat/codex-cli-parity
- Status: deterministic gates green; live Codex smoke test deferred

## What was verified deterministically

The full canonical pipeline (`./scripts/run-verify.sh`) is green on this branch:

- Shell static checks (shellcheck, `sh -n`, `jq`).
- `scripts/check-sync.sh` — root ↔ templates parity, including the new
  `.codex/`, `.agents/skills/`, `docs/recipes/codex-setup.md`, and
  `scripts/check-skill-sync.sh` mirrors.
- `scripts/check-pipeline-sync.sh` — canonical post-implementation order
  references `cross-review` consistently across SKILL.md / rules / CLAUDE.md /
  AGENTS.md / README.md / definition-of-done.md.
- `scripts/check-skill-sync.sh` — 13 skills in lock-step on body, name,
  description, and implicit-invocation policy.
- `tests/test-check-mojibake.sh` — 11/11 hook test cases pass.
- `tests/test-check-skill-sync.sh` — 6/6 fixtures (parity + four drift
  modes + dual-side forbid) pass.
- `go test ./...` — including the new
  `TestExecuteInit_RendersCodexSurfaces`,
  `TestCheckCodexEffectiveConfig_*`, `TestLoad_RequireCodexCLI`, and
  `TestTemplateBaseCodexAssetsExist` cases.

Last verify evidence: `docs/evidence/verify-2026-05-07-091630.log`.

## What requires a manual smoke test

The Codex flow itself (running `codex` end-to-end against this branch) is a
manual sign-off. Steps for a maintainer with the Codex CLI installed:

1. `git checkout feat/codex-cli-parity && go install ./cmd/ralph` (or build
   the binary).
2. `mkdir /tmp/ralph-codex-smoke && cd /tmp/ralph-codex-smoke && ralph init --yes`.
3. `codex trust .` — required, otherwise `.codex/config.toml` and `[hooks]`
   are silently dropped.
4. `ralph doctor` — confirm Codex CLI detected, `codex_hooks=true`, and at
   least one hook entry visible. Capture the output for evidence.
5. `codex` then drive the standard flow with skill mentions:

   ```
   $spec → $plan → $work → $self-review → $verify → $test → $sync-docs → $cross-review → $pr
   ```

   Confirm `docs/specs/`, `docs/plans/active/`, and
   `docs/reports/{self-review,verify,test,sync-docs,cross-review-triage}-*.md`
   are populated by the Codex-driven run.
6. Re-run `./scripts/check-skill-sync.sh` and `./scripts/run-verify.sh`
   inside the smoke project — both must exit 0.

Record the run in a follow-up walkthrough report when executed.

## Why the smoke test is not run in CI

- The Codex CLI is not pre-installed in the GitHub Actions runner image used
  by `.github/workflows/verify.yml`.
- A live `codex exec` invocation requires authenticated OpenAI credentials
  that should not live in CI.
- The deterministic gates above already enforce every codified contract
  (config layout, manifest entries, drift detection, doctor probes). The
  smoke test exists to catch behavioural surprises in the Codex CLI itself,
  which a CI runner cannot exercise without the binary.

## Known gaps recorded

- `tests/upgrade_downgrade_test.go` — explicit new→old→new round-trip
  fixture. Postponed: requires generating a pre-rename manifest and template
  snapshot that exercises the `codex-review` → `cross-review` rename. The
  existing `internal/cli` upgrade tests cover the diff engine; the round-
  trip-specific scenario lands as follow-up work.
- Live Codex smoke test (above) — must be run manually before the next
  release tag.
