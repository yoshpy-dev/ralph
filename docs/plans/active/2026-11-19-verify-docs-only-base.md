# verify-docs-only-base

- Status: Approved
- Approved: 2026-11-19 sha256:37bcd60d52cb
- Owner: Claude Code
- Date: 2026-11-19
- Related request: run-verify.sh: 言語 pack が当たらない repo では、CI がコードの PR も「docs だけの変更」と判定して exit 0 になる
- Related issue: #341
- Type: fix
- Branch: fix/verify-docs-only-base

## Objective

In a repository where no language pack and no local verifier runs,
`scripts/run-verify.sh` decides whether the change is docs only from the same
change set in every scope and in CI: the merge-base with the base branch plus
uncommitted and untracked files. A code change then exits 2 in CI as it does
locally, a docs-only change still exits 0, and a pull request that only adds
or updates ralph's scaffold (`ralph init`, `ralph upgrade`) still exits 0.

## Scope

- `scripts/detect-changed-languages.sh` (and its byte-identical copy under
  `templates/base/scripts/`):
  - base: `GITHUB_BASE_REF`, when set, comes first, read the same way as
    `scripts/secret-scan-branch.sh` (`refs/remotes/origin/<name>`, else
    `refs/heads/<name>`); then `[verify].base`; then the automatic base as
    today. The header line "No environment variable chooses the base." is
    rewritten to say that no setting comes from an environment variable and
    that CI's own `GITHUB_BASE_REF` is read first, as `secret-scan-branch.sh`
    does.
  - output: `docs_only` becomes `true|false|unknown`. `true`: every file in the
    change set is docs or ralph scaffold. `false`: at least one is not.
    `unknown`: the change set could not be read (no git, no repository, a
    settings error, no base, no merge-base, no remote default, a missing
    `GITHUB_BASE_REF` ref, or any of the four commands that list the change
    set exiting non-zero). A new key `first_non_docs=<path>` names the first
    file that is neither, when `docs_only=false`.
  - the four commands that list the change set (`git diff --name-only
    <merge-base> HEAD`, `git diff --name-only`, `git diff --name-only
    --cached`, `git ls-files --others --exclude-standard`) lose their
    `|| true`: a failure of any one of them, even after another printed
    docs-only paths, gives `reason=diff_failed` and `docs_only=unknown`, so a
    failed read can never become `true` (and the changed scope no longer
    turns a failed read into `no_changes`).
  - the loop no longer exits at the first shared or unclassified file: it
    remembers the first full-scope reason, reads every file to compute
    `docs_only`, and then emits `scope=full` with that reason. Which scope and
    which languages it reports is unchanged.
  - scaffold: a path listed as a `[files.<key>]` table of
    `.ralph/manifest.toml` (single-quoted, double-quoted without backslashes,
    or bare key), plus `.ralph/manifest.toml` itself and `.ralph/baseline/*`.
    A changed path that still exists is checked against the working tree's
    manifest; a path that no longer exists (a deletion, such as a core file
    `ralph upgrade` removes together with its manifest entry) is checked
    against that manifest and the one at the merge-base
    (`git show <merge-base>:.ralph/manifest.toml`).
  - the docs table stays the detector's only table:
    `.harness/*|docs/*|templates/base/docs/*|README.md|AGENTS.md|CLAUDE.md|*.md`.
- `scripts/run-verify.sh` (and its byte-identical template copy):
  - full scope also runs the detector, into a per-run temp file of its own,
    and takes only `docs_only`, `reason`, and `first_non_docs` from it. The
    languages of the full scope still come from `scripts/detect-languages.sh`
    (#190: CI runs full), and no `<lang>_roots` is passed.
  - the `git diff --name-only` / `--cached` block and its docs table
    (`docs/*|README.md|AGENTS.md|CLAUDE.md|.claude/*`) are removed.
  - when no verifier ran: `true` exits 0 with a line saying the changes are
    docs or ralph scaffold files only; `false` exits 2 naming the first
    code-like path; `unknown` (also a missing or failed detector) exits 2
    saying the changes could not be read and why. Both exit-2 cases say how to
    add a verifier for the current mode (`scripts/verify.local.sh`,
    `.ralph/local/verify.d/*.sh` for static and all, `.ralph/local/test.d/*.sh`
    for test and all, or `ralph pack add <language>`).
  - `.harness/state/verify-scope` in full scope records the detector's
    `docs_only` instead of a fixed `false`.
- Tests: a new `tests/test-run-verify-docs-only.sh`; new cases in
  `tests/test-detect-changed-languages.sh`; existing tests updated for the new
  messages, the three-valued `docs_only`, the detector being needed in full
  scope, and `unset GITHUB_BASE_REF`.
- Docs: `docs/quality/quality-gates.md` (root and template), README's
  `[verify].base` sentence, the comments in both scripts.

## Non-goals

- Changing which language packs the full scope runs, or making CI use the
  changed scope.
- Changing `templates/base/.github/workflows/verify.yml` or the root
  `.github/workflows/verify.yml`: the detector reads `GITHUB_BASE_REF` itself,
  so the seed workflow already in downstream repositories works unchanged.
- A `--base` argument to `run-verify.sh` or the detector.
- Treating only unmodified scaffold files (hash match against the manifest) as
  scaffold.
- Shipping the release note itself (`internal/upgrade/notes/<X.Y.Z>.md` is
  written at `/release`); the candidate text goes to lane A.

## Assumptions

- `actions/checkout` with `fetch-depth: 0` on `pull_request` fetches every
  branch into `refs/remotes/origin/*` and checks out the merge commit detached;
  the existing secret-scan step of `verify.yml` already depends on
  `origin/${GITHUB_BASE_REF}` existing. Whether it writes
  `refs/remotes/origin/HEAD` does not matter once `GITHUB_BASE_REF` comes first.
- CI has no `ralph` binary, so `[verify].base` is never read there.
- `ralph init` writes `.ralph/manifest.toml` with `[files.'<path>']` keys
  (checked with a scratch `ralph init --yes` on a8c5e289: 160 entries, every
  file init writes except the manifest itself). `templates/base/.gitignore`
  does not ignore `.ralph/`, so CI's checkout holds it.
- The two scripts are core files, upgraded together; a repository with
  `run-verify.sh` and without `detect-changed-languages.sh` is not supported
  (it gets `unknown`).

## Affected areas

- `scripts/detect-changed-languages.sh`, `templates/base/scripts/detect-changed-languages.sh`
- `scripts/run-verify.sh`, `templates/base/scripts/run-verify.sh`
- `tests/test-run-verify-docs-only.sh` (new)
- `tests/test-detect-changed-languages.sh`
- `tests/test-run-verify-scope.sh`, `tests/test-run-verify-concurrency.sh`,
  `tests/test-run-verify-branch-secret-scan.sh`, `tests/test-ralph-dispatch.sh`
  (fixtures and expected messages)
- `internal/org/gate_runverify_test.go` (deviation found in S2: this Go test
  runs the real scripts and pins the `ralph config get` calls; the full scope
  now also asks `verify.base`, and its fixture environment must drop
  `GITHUB_BASE_REF`, which CI sets)
- `docs/quality/quality-gates.md`, `templates/base/docs/quality/quality-gates.md`
- `README.md` (the `[verify].base` sentence only)

## Visual review

- Page: `.harness/state/plan-visual/verify-docs-only-base.html` (easy figures
  `overview` and `easy-pass`; fold figures `detail-overview`, `detail-base`,
  `detail-exit`)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: full page and all five figures shot and read; fixed a label
  touching a box (overview), a copy edge running through a label
  (detail-overview), and a sub line overflowing its node (detail-base)

## Design decisions

Critical forks: None asked of the user. The three decisions the user handed to
`/consult` were chosen autonomously (consult `consult-plan-docsonly`,
2026-11-19, verdict "fix the test environment, then proceed"; all four
proposals kept):

- D1. CI base — chosen autonomously: the detector reads `GITHUB_BASE_REF`
  first. Rationale: `scripts/secret-scan-branch.sh` (run by `run-verify.sh`)
  already does exactly this, README line 154 already says "in CI, the pull
  request's base comes first", and `verify.yml` is a seed file that
  `ralph upgrade` does not rewrite, so a `--base` argument passed from the
  workflow would not reach existing downstream repositories; one whose default
  branch is neither main nor master would then fail docs-only PRs too. The
  2026-10-30 rule "No environment variable chooses the base" was about user
  settings (moved to `ralph.local.toml`); `GITHUB_BASE_REF` is a fact CI sets.
- D2. Scaffold — chosen autonomously: paths listed in `.ralph/manifest.toml`
  plus the manifest and `.ralph/baseline/*`. Rationale: the manifest is the
  record of what ralph wrote, and it is in the PR's own HEAD. A hash match
  would make an edit to `ralph.toml` exit 2 and needs sha256 in portable sh.
  An edited core file (say `scripts/run-verify.sh`) also counts as scaffold;
  a pack-less repository has no checker for it anyway.
- D3. Docs table — chosen autonomously: the detector's table is the only one;
  `run-verify.sh` takes `docs_only` from the detector in both scopes. This
  removes the drift by construction. `.claude/*` is no longer docs as such:
  scaffolded `.claude/*` files are covered by the manifest and `.md` files by
  `*.md`, so only a user's own non-`.md` file under `.claude/` (a local hook
  script) now counts as code.
- D4. Unreadable change set — `unknown` exits 2 like `false`, with its own
  line. Today the changed scope already reports `docs_only=false` (exit 2) for
  the same cases; the full scope said "docs only" there, which is the bug.

## Acceptance criteria

- [ ] AC1: In a repository with no language pack and no local verifier, a
  CI-like checkout (detached at a PR merge commit, branches fetched into
  `refs/remotes/origin/*`, no `refs/remotes/origin/HEAD`, default branch
  `develop`, `GITHUB_BASE_REF=develop`) of a commit that adds a code file makes
  `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` exit 2, and its output
  says code-like changes with the file's path and does not say docs only.
- [ ] AC2: The same checkout of a commit that changes only docs exits 0.
- [ ] AC3: The same checkout of a commit that adds only ralph scaffold (files
  listed in its `.ralph/manifest.toml`, the manifest itself, and a
  `.gitignore` listed there) exits 0.
- [ ] AC4: In the same repository, a local clone on the feature branch with a
  clean tree and no `GITHUB_BASE_REF` gives the same exit code as AC1, AC2,
  and AC3 for the same commits, in the full scope and in the changed scope.
- [ ] AC5: With `GITHUB_BASE_REF` naming a branch that has no ref, or no
  merge-base, the run exits 2 and says the changes could not be read, with the
  reason; it never says docs only.
- [ ] AC6: The exit-2 output names how to add a verifier for the mode
  (`scripts/verify.local.sh`; `.ralph/local/verify.d/` in static and all;
  `.ralph/local/test.d/` in test and all; `ralph pack add <language>`).
- [ ] AC7: The detector emits `docs_only=true|false|unknown` and, for `false`,
  `first_non_docs=<path>`; its scope, reason, languages, and roots are as
  before for every existing case of `tests/test-detect-changed-languages.sh`.
- [ ] AC8: With `GITHUB_BASE_REF` set, the detector uses
  `refs/remotes/origin/<name>` (else `refs/heads/<name>`) and does not ask
  `ralph` for `[verify].base`.
- [ ] AC9: `scripts/` and `templates/base/scripts/` copies of both scripts are
  byte-identical (`./scripts/check-sync.sh` passes).
- [ ] AC10: Every test that runs the detector or `run-verify.sh` unsets
  `GITHUB_BASE_REF`, so the suite passes the same in CI (where
  `GITHUB_BASE_REF=main`) and locally.
- [ ] AC11: When any one of the four commands that list the change set exits
  non-zero (checked one at a time with a `git` stand-in on PATH, including a
  case where an earlier command printed only docs paths), the detector
  reports `docs_only=unknown` with `reason=diff_failed`, and a pack-less
  `run-verify.sh` exits 2 in the full and the changed scope.
- [ ] AC12: A commit that deletes a non-`.md` scaffold file and removes its
  entry from `.ralph/manifest.toml` (what `ralph upgrade` does to an obsolete
  core file) exits 0 in the CI-like checkout and in the local clone.

## Implementation outline

1. Detector: `GITHUB_BASE_REF` first, three-valued `docs_only`,
   `first_non_docs`, no early exit on shared or unclassified files, scaffold
   from the manifest (deleted paths also from the merge-base's manifest),
   `diff_failed` for a failed listing command; header and comments; template
   copy; new and updated
   cases in `tests/test-detect-changed-languages.sh` (with
   `unset GITHUB_BASE_REF`). — verify: `cmp scripts/detect-changed-languages.sh templates/base/scripts/detect-changed-languages.sh && shellcheck scripts/detect-changed-languages.sh tests/test-detect-changed-languages.sh && sh tests/test-detect-changed-languages.sh`
2. `run-verify.sh`: detector in full scope, the git-diff block and its table
   removed, the three outcomes and their lines, the scope file; template copy;
   the new `tests/test-run-verify-docs-only.sh` (AC1–AC6, AC11, AC12); fixtures and
   expected lines of the existing run-verify tests updated, each with
   `unset GITHUB_BASE_REF`. — verify: `cmp scripts/run-verify.sh templates/base/scripts/run-verify.sh && shellcheck scripts/run-verify.sh tests/test-run-verify-docs-only.sh tests/test-run-verify-scope.sh tests/test-run-verify-concurrency.sh tests/test-run-verify-branch-secret-scan.sh tests/test-ralph-dispatch.sh && for t in tests/test-run-verify-docs-only.sh tests/test-run-verify-scope.sh tests/test-run-verify-concurrency.sh tests/test-run-verify-branch-secret-scan.sh tests/test-ralph-dispatch.sh; do sh "$t" || exit 1; done`
2b. (Deviation found in S2.) `internal/org/gate_runverify_test.go`: the gate's
   full-scope runs now ask `verify.base` too, so the pinned call lists gain
   `verify.base:off`; `gateRalphStubEnv` drops `GITHUB_BASE_REF` so the
   control's changed scope reads `[verify].base` in CI as well. Production
   code of the gate is unchanged. — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && go test ./internal/org/ -run 'TestGate_RealRunVerify' -count=1 && GITHUB_BASE_REF=main go test ./internal/org/ -run 'TestGate_RealRunVerify' -count=1`
3. Docs: `docs/quality/quality-gates.md` (both copies) and README's
   `[verify].base` sentence. — verify: `./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` (shellcheck over
  `scripts/*.sh` and `tests/*.sh`, `bash -n`, sync checks).
- Spec compliance criteria to confirm: AC1–AC10; the two decisions of the
  user (pack-less code PR exits 2 in CI and locally, committed or not;
  scaffold-only PR does not fail).
- Documentation drift to check: `docs/quality/quality-gates.md` both copies,
  README line 154, the detector header, `run-verify.sh` comments,
  `./scripts/check-sync.sh`, `./scripts/check-template.sh`.
- Evidence to capture: verify report under `docs/reports/`; a manual run in a
  scratch `ralph init` repository (pack-less) for code, docs, and scaffold
  commits, with and without `GITHUB_BASE_REF`.

## Test plan

- Unit tests (detector): `docs_only` for docs + scaffold mixes; a shared
  scaffold file (`scripts/run-verify.sh` listed in the manifest) gives
  `scope=full`, `reason=shared:...`, `docs_only=true`; a shared non-scaffold
  file plus a code file gives `docs_only=false` and `first_non_docs`; the
  three manifest key forms; a manifest path with a backslash is not scaffold;
  `GITHUB_BASE_REF` with origin ref, with only a local branch, with neither
  (`unknown`); `GITHUB_BASE_REF` set does not ask the stand-in ralph; every
  fallback that could not read the diff gives `docs_only=unknown`; each of
  the four listing commands failing in turn through a `git` stand-in
  (`diff_failed`, AC11); a deleted scaffold path found only in the
  merge-base's manifest is scaffold, and a deleted path in neither manifest
  is not (AC12).
- Integration tests (`tests/test-run-verify-docs-only.sh`): AC1–AC6 on a
  pack-less fixture with a bare remote whose default branch is `develop`, a
  CI-like clone (fetch `+refs/heads/*:refs/remotes/origin/*`, no set-head, a
  PR merge commit checked out detached), and a local clone; static, test, and
  all modes for the guidance line.
- Regression tests: the existing cases of `test-detect-changed-languages.sh`,
  `test-run-verify-scope.sh`, `test-run-verify-concurrency.sh` (case 5 lines
  and the scope file), `test-run-verify-branch-secret-scan.sh` (docs-only and
  finding cases), `test-ralph-dispatch.sh` case H.
- Edge cases: `GITHUB_BASE_REF` set but empty (treated as unset); no manifest;
  an untracked code file only; a deleted scaffold file; a rename from a
  scaffold path to a code path (`git diff --name-only` lists the new path).
- Evidence to capture: test report under `docs/reports/`.

## Risks and mitigations

- R1: Downstream pack-less repositories start failing CI on every code PR
  (exit 2) until they add a verifier. This is the user's decision 1.
  Mitigation: the output names what to add; release-note candidate sent to
  lane A.
- R2: A downstream CI with a shallow checkout (`fetch-depth` 1) or without the
  base branch fetched gets `unknown` and exits 2 even on docs-only PRs (only
  where no verifier runs). Mitigation: the `unknown` line says to fetch the
  base branch; release-note candidate.
- R3: A user's own non-`.md` file under `.claude/` now counts as code (D3).
  Mitigation: release-note candidate.
- R4: A local shell that still has `GITHUB_BASE_REF` set changes the base, as
  it already does for `secret-scan-branch.sh`. Mitigation: documented in the
  header and quality-gates.
- R5: The full scope now reads `[verify].base`, so a broken `ralph.local.toml`
  makes a pack-less full run exit 2 (`unknown`, `config_error:verify.base`).
  Repositories with a language pack are unaffected (a verifier ran).
  Mitigation: documented in quality-gates.
- R6: The meta-repository's own CI sets `GITHUB_BASE_REF=main`; a test that
  forgets to unset it would pass locally and fail in CI. Mitigation: AC10, and
  `test-run-verify-docs-only.sh` sets it only per case.
- R7: Adding a path to the manifest lets a change pass as scaffold. The check
  is not a security boundary; noted in the detector header.
- R8 (plan advisory 1): a `git` listing command that fails after printing
  part of the change set would otherwise leave a docs-only partial list.
  Mitigation: every listing command's exit status is checked (AC11).
- R9 (plan advisory 2): `ralph upgrade` deletes obsolete core files and their
  manifest entries together, so the working tree's manifest alone would call
  the deletion code. Mitigation: deleted paths also match the merge-base's
  manifest (AC12).
- R10: The changed scope now reports `full` with `diff_failed` where a failed
  listing used to pass silently as `no_changes`; a repository with packs then
  runs every pack. This is the intended direction (a failed read is not a
  clean tree).

## Rollout or rollback notes

Ships with the next ralph release; downstream repositories get both scripts
with `ralph upgrade` (core files). Rollback: revert the PR; the scripts return
to the full scope's `git diff` check. No state or format migration.

## Open questions

None.

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex (gpt-6-astra, xhigh). 1 [HIGH] the four listing commands' `|| true` can turn a failed read into `docs_only=true` → applied (scope bullet, AC11, R8, test plan). 2 [MEDIUM] an upgrade that deletes a core file and its manifest entry would read as code → applied (scaffold bullet, AC12, R9, test plan). Applied without asking, per the user's standing instruction for this series.
- [x] Plan approved — auto-approved under the user's standing instruction for the #340〜#344 series (「計画は推す案で自動承認する」)
- [x] Branch created
- [x] Implementation started
  - S1 (detector) committed 930e6975 by the implementer: 221/221 in
    `tests/test-detect-changed-languages.sh`, also with `GITHUB_BASE_REF=main`
    exported, under dash, and on ubuntu:24.04; self-check clean (one
    copy-paste fix). The slice commands run shellcheck as
    `shellcheck --severity=warning` (the level of `scripts/verify.local.sh`):
    plain shellcheck 0.11 exits 1 on the info-level SC1091 that HEAD already
    had for each `# shellcheck source=` line. Deviation within scope: a
    docs+scaffold mix whose scaffold file is unclassified (say
    `.claude/hooks/x.sh`) keeps `scope=full`/`unclassified:<path>` and gets
    `docs_only=true`, because scope and reason stay as before (AC7).
    Follow-up for /sync-docs: `tests/test-detect-changed-languages.sh` grew
    from 848 to about 1290 lines; note it in the 800-line row of
    `docs/tech-debt/README.md`.
  - S2 (run-verify.sh) committed ec269cb7 by the implementer: docs-only
    66/66, scope 25/25, concurrency 37/37, branch-secret-scan 55/0,
    dispatch 33/0, detector 221/221, the same with `GITHUB_BASE_REF=main`;
    the new test fails 37 checks on the old run-verify.sh; self-check clean.
    Decisions: the branch-secret-scan cases moved their secret to a docs file
    (`notes.md`) so their exit codes still tell skip from no-skip, and a new
    case keeps a code-file secret (exit 2 with the finding line); that
    fixture excludes its untracked `scripts/` in `.git/info/exclude`. git
    2.49's fetch writes `refs/remotes/origin/HEAD` even with an explicit
    refspec, so the CI-like checkout deletes it and checks that it is gone.
  - S2b (deviation, inline, trivial single-file edit) committed 672e5ded:
    `internal/org/gate_runverify_test.go` failed after S2 (the full scope
    asks `verify.base`) and, with `GITHUB_BASE_REF=main`, already after S1.
    Verify: gofmt/vet clean; `go test ./internal/org/ -run
    TestGate_RealRunVerify` ok, also with `GITHUB_BASE_REF=main`. A wider
    `go test ./internal/org/ -run 'Gate|Regate'` hit go test's 10-minute
    timeout on the loaded machine (no failure reported before the timeout);
    the full suite runs in /test. Self-check: unnecessary changes clean,
    typos clean, debug leftovers clean, secrets clean, files outside the
    slice clean (the plan note committed separately). Plan re-approved
    (auto) after the deviation note: sha256:37bcd60d52cb; figures unchanged
    (the detail overview's "existing tests" node covers the file).
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
