# secret-scan-test-tmp

- Status: Approved
- Approved: 2026-10-20 sha256:3ee6f70482f6
- Owner: Claude Code
- Date: 2026-10-20
- Related request: tech-debt row 170 (`tests/test-secret-scan.sh` writes to fixed `/tmp` paths); precursor of the stage-9 trial (`docs/plans/active/2026-10-20-org-trial.md` on branch `chore/org-trial`, D11)
- Related issue: N/A
- Type: fix
- Branch: fix/secret-scan-test-tmp

## Objective

Let two copies of `tests/test-secret-scan.sh` run at the same time
without failing each other. Today every `expect_exit` writes the command's
stdout and stderr to `/tmp/ralph-secret-scan-test.out` and
`/tmp/ralph-secret-scan-test.err`, the stderr checks read the same file,
and the `trap` deletes both on exit. Two copies overwrite each other's
files, and the first to exit deletes them (row 170 records both copies
failing two checks each on 2026-10-07). The stage-9 trial runs two
feature orgs whose gates both run `run-test.sh`, which runs every
`tests/test-*.sh`, so their gates can overlap.

## Scope

- `tests/test-secret-scan.sh`: write the captured stdout and stderr into
  a directory the script makes with `mktemp -d` for itself, separate from
  the fixture repository `$workdir`, and remove it in the same `trap`
- `docs/tech-debt/README.md`: mark row 170 resolved

## Non-goals

- Other tests: a search of `tests/*.sh` and `scripts/*.sh` for fixed
  `/tmp/<name>` paths found no other one in use (the hits in
  `tests/test-pre-bash-guard.sh` are command strings the guard is fed)
- `templates/base/`: the scaffold ships no `tests/test-secret-scan.sh`
- Changing what any check asserts

## Assumptions

- `tests/test-secret-scan.sh` is the only copy (no template mirror):
  `templates/base/tests/` does not exist

## Affected areas

- `tests/test-secret-scan.sh`
- `docs/tech-debt/README.md` (row 170)

## Visual review

None (a change to a single test file and its tech-debt row; no
component, state, format, or boundary changes)

## Design decisions

Critical forks: None

- D1: the capture files go in their own `mktemp -d` directory, not under
  `$workdir`. `$workdir` is the fixture git repository the checks scan
  (`git init -q` right after it is made), so files written there would
  show up as untracked files in the scanned repository and could change
  what a check sees. The new directory is made before `$workdir`, with
  the same `${TMPDIR:-/tmp}` template style, and the existing `trap`
  removes it too

## Acceptance criteria

- [ ] AC1: no line of `tests/test-secret-scan.sh` names
  `/tmp/ralph-secret-scan-test.out` or `/tmp/ralph-secret-scan-test.err`;
  `expect_exit`, `expect_stderr_contains`, `expect_stderr_not_contains`,
  and every other reader use the per-run capture directory
- [ ] AC2: the per-run capture directory is not inside `$workdir`, and
  the `trap` on EXIT, HUP, INT, and TERM removes it
- [ ] AC3: run alone, the script passes with the same pass count as
  before the change (`FAIL: 0`; the count is recorded in the test report)
- [ ] AC4: two copies started together both exit 0 with `FAIL: 0`
- [ ] AC5: row 170 of `docs/tech-debt/README.md` is marked resolved with
  this change, in the register's style

## Implementation outline

1. Inline (a single-file edit; the handoff would cost more than the
   change): add the capture directory, replace the eight uses of the two
   paths, extend the `trap`; mark row 170

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` (includes
  shellcheck when installed); `sh -n tests/test-secret-scan.sh`
- Spec compliance criteria to confirm: AC1 by
  `grep -n 'ralph-secret-scan-test' tests/test-secret-scan.sh` (no hit
  of the two fixed paths), AC2 by reading the `trap` line
- Documentation drift to check: row 170 says resolved; nothing else
  states the paths (`grep -rn 'ralph-secret-scan-test' .`)
- Evidence to capture: the grep outputs

## Test plan

- Unit tests: the script itself (`sh tests/test-secret-scan.sh`), alone
- Integration tests: two copies at once, each started in the background
  with its PID kept and its exit code taken by its own `wait`:
  `sh tests/test-secret-scan.sh > a.log 2>&1 & pa=$!; sh tests/test-secret-scan.sh > b.log 2>&1 & pb=$!; wait "$pa"; ra=$?; wait "$pb"; rb=$?`.
  AC4 holds only when `ra` and `rb` are both 0 and both logs say
  `FAIL: 0`. The same harness is checked once with a copy that is made
  to fail (a command that exits non-zero in place of one script), and
  must then report that failure
- Regression tests: `./scripts/run-test.sh`
- Edge cases: the script interrupted with SIGINT leaves no capture
  directory behind (the `trap`)
- Evidence to capture: the pass counts of the alone run and both
  parallel runs, and the parallel run's exit codes

## Risks and mitigations

- R1 a check depended on the old absolute path (for example a fixture
  that greps for `/tmp/`): the alone run's pass count must equal the
  count before the change
- R2 `mktemp -d` before `$workdir` changes the order of what is made in
  `TMPDIR`: the signal check near the end uses its own `sig_dir`, not
  the order, so it is unaffected; the alone run confirms

## Rollout or rollback notes

Revert the PR. Nothing else depends on the paths.

## Open questions

None

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created

Notes:

- 2026-10-20: Codex plan advisory gave 1 finding, applied before
  approval: MEDIUM the parallel check lost each copy's exit code (the
  Test plan's integration line). Auto-approved per the user's standing
  instruction
- 2026-10-20: the trial plan's consult (consult-plan-trial) kept this
  precursor (D11 there)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
