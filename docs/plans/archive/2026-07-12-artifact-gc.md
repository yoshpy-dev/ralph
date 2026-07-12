# Plan: Artifact retention policy + GC + stale-state detection

- Status: approved
- Flow: standard (/work)
- Branch: chore/artifact-gc
- Base: develop
- Date: 2026-07-12

## Objective

Stop unbounded artifact growth and detect zombie runtime state:

1. `docs/reports/` has 256 committed files (~85/month) with no retention policy.
2. `docs/evidence/` accumulates local verify logs without pruning (516 on disk).
3. `.harness/state/orchestrator/orchestrator.json` can be left `status: "running"`
   forever by a crashed/abandoned run; nothing detects it (a 2026-05-13 zombie
   pointing at a deleted tempdir exists locally today).

## Scope

- New `scripts/gc-artifacts.sh`: dry-run by default, `--apply` to act,
  `--days N` (default 30). Deletes `docs/reports/` files older than N days
  (by filename date, falling back to mtime); prunes local `docs/evidence/`
  logs beyond the newest 20. Never touches README/.gitkeep files.
- `scripts/run-verify.sh`: self-prune evidence logs (keep newest 20).
- Retention policy documented in `docs/reports/README.md`.
- `ralph doctor` (internal/cli/doctor.go): warn when
  `.harness/state/orchestrator/orchestrator.json` has `status == "running"`
  but its mtime is older than 24h — suggest `ralph abort`/manual cleanup.
- Run the GC once in this PR (`--apply`) so the cleanup lands here.
- Mirror script changes to `templates/base/scripts/` (check-sync gate).
- Tests: shell test for gc-artifacts.sh (tempdir fixture); Go test for the
  doctor stale-state check.

## Non-goals

- No changes to report generation or naming.
- No remote/CI-side deletion; GC is a local maintenance command.
- No orchestrator behavior change (detection only).

## Acceptance criteria

- AC1: `gc-artifacts.sh` dry-run lists candidates without deleting; `--apply`
  deletes only matching files; README/.gitkeep always survive.
- AC2: reports newer than N days and all non-report dirs are untouched.
- AC3: `run-verify.sh` leaves at most 20 evidence logs after a run.
- AC4: `ralph doctor` flags a stale running orchestrator state (>24h) and
  passes on fresh/completed states.
- AC5: one-time cleanup applied in this PR; `docs/reports/` only retains
  files newer than 30 days.
- AC6: `./scripts/run-verify.sh` and `./scripts/check-sync.sh` pass.

## Verify plan

- `bash tests/test-gc-artifacts.sh`
- `go test ./internal/cli/... -count=1`
- `./scripts/check-sync.sh`
- `./scripts/run-verify.sh`

## Risks

- R1: deleting committed reports loses easy browsing of old artifacts —
  accepted: git history and merged PRs retain them; policy documented.
- R2: filename-date parsing must not misfire on unusual names — fallback to
  mtime and always keep non-matching files.

## Rollout

Single PR to `develop`. GC is manual (`--apply`) thereafter.
