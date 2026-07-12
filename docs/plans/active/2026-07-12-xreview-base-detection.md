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
   (next to `pick_reviewer` — same placement rationale: dispatcher-adjacent,
   testable in isolation): prints the repo default branch via
   `git symbolic-ref --quiet --short refs/remotes/origin/HEAD` with the
   leading `origin/` stripped; falls back to `main` when unset (fresh
   clones, no remote). Same semantics as `scripts/ralph` `default_branch()`
   (L488-492) and the pipeline-outer.md prompt snippet shipped in #121.
   Update the file's header function list.
2. **`scripts/ralph-pipeline.sh` ~L807-808**: replace the two-line
   `rev-parse HEAD@{upstream}` detection with `_base="$(detect_base_branch)"`.
   No other gate-logic changes (the `git diff "${_base}...HEAD" --quiet`
   guard now compares against the true merge target).
3. **cross-review SKILL.md line 51 (x4 copies)**: replace the documented
   BASE command with the symbolic-ref form:
   `BASE=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|^origin/||'); BASE=${BASE:-main}`.
4. **Tests** (`tests/test-ralph-cli-driver.sh`): new cases for
   `detect_base_branch` using temp git fixtures: (a) origin/HEAD set to
   `main` while the current branch is a pushed feature branch tracking
   itself -> returns `main` (the exact regression shape); (b) origin/HEAD
   set to a non-`main` default (e.g. `develop`) -> returns `develop`;
   (c) no origin/HEAD -> falls back to `main`. Existing xreview suites must
   stay green.
5. **tech-debt row** (added in #121, base-detection divergence): mark
   RESOLVED per file convention.
6. **Mirrors**: `templates/base/scripts/ralph-cli-driver.sh`,
   `templates/base/scripts/ralph-pipeline.sh`, SKILL x4 lock-step
   (check-sync / check-skill-sync / check-pipeline-sync).

## Non-goals

- No change to `scripts/ralph` `default_branch()` (already correct) and no
  consolidation of the two helpers across shell entry points (different
  sourcing contexts; grep-able duplication acceptable).
- No change to the `--quiet` empty-diff skip semantics itself (an empty diff
  against the true base legitimately skips review).
- No Codex-side changes.

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
- [ ] AC2: new driver tests pass: pushed-feature-branch fixture returns the
  default branch (regression shape), non-main default honored, no-remote
  fallback = main.
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
