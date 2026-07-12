# Plan: Fix all bugs found by the post-merge regression audit

- Status: approved
- Flow: standard (/work)
- Branch: fix/audit-findings
- Base: main
- Date: 2026-07-13

## Objective

Fix the seven issues surfaced by the post-merge regression investigation of
the #124–#129 series (three pre-existing bugs, one narrow series-introduced
behavior change, three cosmetic/doc issues).

## Scope / acceptance criteria

1. **[HIGH, pre-existing] MAX_PARALLEL enforcement** —
   `scripts/ralph-orchestrator.sh:1530`: `grep -c 'running' slice-*.status`
   emits `file:count` lines when 2+ files match, breaking the integer
   comparison so the parallel cap silently never binds.
   AC: with 2+ running status files and MAX_PARALLEL=1, no new slice is
   scheduled; regression test proves it.
2. **[pre-existing] doctor pack detection** — `internal/cli/doctor.go`
   `checkInstalledPacks` matches un-namespaced manifest keys (`README.md`)
   while the manifest stores `packs/languages/<lang>/README.md`, so it always
   reports "none installed"; the verify.sh existence probe also points at the
   project root.
   AC: a scaffolded project with packs reports each installed pack; missing
   `packs/languages/<lang>/verify.sh` warns with the correct path; Go tests
   cover both.
3. **[LOW, pre-existing] pipeline status mislabel** —
   `scripts/ralph-pipeline.sh` return-2 handler unconditionally finalizes as
   `gh_unavailable`, overwriting the already-written `invalid_branch_name`.
   AC: an invalid PR branch name finalizes as `invalid_branch_name`.
4. **[LOW, pre-existing] DRY_RUN cross-review skip reason** — skip message
   claims "codex binary not available" when the actual gate is `DRY_RUN=1`.
   AC: DRY_RUN runs log a dry-run skip reason.
5. **[NOTE] validate-clean-base double error** — failure path prints the
   resolve error twice (dispatch arg + in-function default both resolve).
   AC: exactly one error line.
6. **[series-introduced, narrow] `ralph cleanup` eager base resolution** —
   `scripts/ralph:493` resolves `default_branch` up front under
   `set -euo pipefail`, so cleanup dies before doing anything in repos with
   no remote and no main/master, even when the base is not needed on the
   taken path.
   AC: cleanup completes (or fails only at the step that genuinely needs the
   base) in a trunk-only repo; regression test.
7. **[series-introduced] tech-debt evidence links** — `docs/tech-debt/README.md`
   references nine reports deleted by the 30-day retention GC; one more in
   `.claude/agent-memory/verifier/feedback_hook_path_existence_check.md`.
   AC: references annotated as GC'd with a git-history retrieval hint; no
   dangling bare links remain.

## Non-goals

- No redesign of the orchestrator scheduling loop.
- No doctor check additions beyond fixing the existing one.
- No restoration of GC'd reports.

## Verify plan

- shellcheck on changed scripts; gofmt/go vet/go test; templates mirror parity
  (check-sync); full `./scripts/run-verify.sh`.

## Test plan

- New shell regression test for the MAX_PARALLEL count (2+ status files).
- New/extended Go tests for doctor pack detection (installed, missing
  verify.sh, no manifest).
- Trunk-only-repo cleanup test.
- Existing suites as behavior lock.

## Risks

- R1: counting fix must preserve zero-file behavior (`_current_running=0`).
- R2: doctor fix must keep the manifest-less fallback path intact.

## Rollout

Single PR to `main`.
