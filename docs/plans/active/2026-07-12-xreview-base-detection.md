# xreview-base-detection

- Status: Approved (user requested fix + merge; tech-debt follow-up from PR #121)
- Owner: Claude Code
- Date: 2026-07-12
- Related request: Fix the HEAD@{upstream} base-detection weakness at the sites that actually gate cross-review
- Related issue: N/A
- Type: fix
- Branch: fix/xreview-base-detection

## Objective

On a feature branch pushed with `git push -u`, `git rev-parse --abbrev-ref
'HEAD@{upstream}'` resolves to `origin/<same-branch>`; after stripping
`origin/`, `git diff "<base>...HEAD"` compares the branch to itself, the
`--quiet` gate at `scripts/ralph-pipeline.sh:809` sees an empty diff, and
cross-review is silently skipped. Replace the tracking-ref detection with
repo-default-branch detection (the merge target) at the gating sites, via a
shared testable helper.

## Scope

1. **New helper `detect_base_branch` in `scripts/ralph-cli-driver.sh`**
   (next to `pick_reviewer` — dispatcher-adjacent, testable in isolation).
   Resolution order (Codex advisory findings 1-2):
   a. `RALPH_XREVIEW_BASE` env, when set and non-empty — the explicit review
      base (exported by the Loop orchestrator; also an operator override);
   b. `git symbolic-ref --quiet --short refs/remotes/origin/HEAD` with the
      leading `origin/` stripped (repo default branch);
   c. the same main/master fallback semantics as the existing
      `default_branch()` helpers (`scripts/ralph`, `ralph-worktree.sh`) —
      mirror their exact checks rather than blindly printing `main`.
   Update the file's header function list. Note: the pipeline's existing
   `git diff "$base"...HEAD --quiet` gate treats a failing diff (invalid
   base) as "has changes" and runs the review — fail-open-to-review is the
   safe direction and is kept.
2. **`scripts/ralph-orchestrator.sh`**: export
   `RALPH_XREVIEW_BASE="$_base_branch"` (the branch the Loop was launched
   from, already tracked at ~L1292 and used to create the integration
   branch) before spawning per-slice pipelines and the integration pipeline,
   so Loop cross-review diffs against the true merge target even when the
   Loop starts from `develop`/`release/*`. Mirror to templates/base.
3. **`scripts/ralph-pipeline.sh` ~L807-808**: replace the two-line
   `rev-parse HEAD@{upstream}` detection with `_base="$(detect_base_branch)"`.
   No other gate-logic changes (the `git diff "${_base}...HEAD" --quiet`
   guard now compares against the true merge target).
4. **cross-review SKILL.md line 51 (x4 copies)**: replace the documented
   BASE command with the same resolution order (explicit
   `RALPH_XREVIEW_BASE` if set, else symbolic-ref default, else main) —
   phrased as "source scripts/ralph-cli-driver.sh and use
   detect_base_branch" or the equivalent inline snippet.
5. **Tests** (`tests/test-ralph-cli-driver.sh`): fixture-based cases
   (Codex advisory finding 3 — prove the GATE, not just return values):
   (a) regression shape: pushed feature branch tracking itself — the OLD
   detection would yield the feature branch and an EMPTY
   `git diff base...HEAD`, while `detect_base_branch` yields the default
   branch and a NON-EMPTY diff (assert both sides);
   (b) `RALPH_XREVIEW_BASE=develop` wins over origin/HEAD=main;
   (c) origin/HEAD -> non-main default (`develop`) honored;
   (d) no origin/HEAD -> main/master fallback semantics;
   (e) a `git worktree add` fixture resolves origin/HEAD through the shared
   common dir. Existing xreview suites must stay green.
6. **tech-debt row** (added in #121, base-detection divergence): mark
   RESOLVED per file convention.
7. **Mirrors**: `templates/base/scripts/ralph-cli-driver.sh`,
   `templates/base/scripts/ralph-pipeline.sh`,
   `templates/base/scripts/ralph-orchestrator.sh`, SKILL x4 lock-step
   (check-sync / check-skill-sync / check-pipeline-sync).

## Non-goals

- No change to `scripts/ralph` `default_branch()` (already correct) and no
  consolidation of the two helpers across shell entry points (different
  sourcing contexts; grep-able duplication acceptable).
- No change to the `--quiet` empty-diff skip semantics itself (an empty diff
  against the true base legitimately skips review).
- No Codex CLI/runtime changes (the .agents SKILL mirrors are text sync only).

## Assumptions

- `refs/remotes/origin/HEAD` is present in normally cloned repos; worktrees
  created by `ralph-worktree.sh` share the main repo's refs, so it resolves
  there too. When absent, the `main` fallback preserves today's fallback
  behavior.

## Affected areas

- `scripts/ralph-cli-driver.sh` (+ template mirror)
- `scripts/ralph-pipeline.sh` (+ template mirror)
- `.claude/skills/cross-review/SKILL.md` + `.agents` mirror + both
  `templates/base` copies
- `tests/test-ralph-cli-driver.sh`
- `docs/tech-debt/README.md`

## Design decisions

1. **Shared helper in ralph-cli-driver.sh** rather than an inline fix:
   testable in isolation (matches `resolve_phase_model` / `pick_reviewer`
   precedent), grep-able, single place to fix if detection evolves.
2. **symbolic-ref (repo default branch) over any tracking-ref variant**: the
   cross-review scope is "what this branch changes relative to its merge
   target"; the tracking ref is the wrong concept entirely, not just an
   edge case.
3. Critical forks: None.

## Acceptance criteria

- [ ] AC1: `detect_base_branch` exists in ralph-cli-driver.sh; the pipeline
  gate uses it; `grep -rn 'HEAD@{upstream}'` over scripts/ralph-pipeline.sh
  and the 4 cross-review SKILL copies -> 0 hits.
- [ ] AC2: new driver tests pass incl. the end-to-end gate proof (old
  detection -> empty diff vs detect_base_branch -> non-empty diff on the
  same fixture), the RALPH_XREVIEW_BASE override, non-main default,
  main/master fallback, and the worktree fixture.
- [ ] AC2b: ralph-orchestrator.sh exports RALPH_XREVIEW_BASE from
  `_base_branch` before slice and integration pipeline invocations (grep
  evidence), so Loop runs from develop/release/* review against the launch
  branch, not the repo default.
- [ ] AC3: existing cross-review regression suites pass unchanged
  (`tests/test-xreview-gate-regression.sh`,
  `tests/test-xreview-prompt-render.sh`).
- [ ] AC4: tech-debt row RESOLVED; mirrors byte-identical; all sync gates +
  `./scripts/run-verify.sh < /dev/null` + `./scripts/run-test.sh < /dev/null`
  pass.

## Implementation outline

Single slice: helper + pipeline call site + SKILL x4 + tests + tech-debt +
mirrors -> gates -> commit.

## Verify plan

- Static: run-static-verify; AC1 greps; doc drift: SKILL wording vs helper
  semantics; model-routing.md untouched.
- Evidence: docs/reports/verify-2026-07-12-xreview-base-detection.md.

## Test plan

- Unit: the three detect_base_branch fixture cases (AC2).
- Regression: full run-test glob incl. both xreview suites.
- Edge: origin/HEAD pointing at a branch name containing `/` (e.g.
  `origin/release/1.0` -> `release/1.0`) — the strip must remove only the
  leading `origin/`.
- Evidence: docs/reports/test-2026-07-12-xreview-base-detection.md.

## Codex plan advisory (evidence)

4 findings, all adopted: (1) HIGH — Loop merge target is the launch branch,
not the repo default -> explicit `RALPH_XREVIEW_BASE` exported by the
orchestrator takes precedence; (2) MEDIUM — reuse the existing
main/master fallback semantics instead of a bare `main`, keep the gate's
fail-open-to-review behavior for invalid bases; (3) MEDIUM — tests must
prove the gate end-to-end (empty-diff-before vs non-empty-after) plus
worktree and Loop-base fixtures; (4) LOW — Non-goals wording fixed to "no
Codex CLI/runtime changes".

## Risks and mitigations

- Repos where the operator intends a non-default merge target -> unchanged
  risk from before; the helper at least matches the PR-target convention
  used by `/pr` and the orchestrator. An env override knob is out of scope
  (no current consumer).
- `origin/HEAD` unset in CI checkouts -> fallback `main` (same as today).

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (fix/xreview-base-detection)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
