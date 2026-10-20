# Evidence: two copies of tests/test-secret-scan.sh at once

Plan: `docs/plans/active/2026-10-20-secret-scan-test-tmp.md` (AC3, AC4,
the TERM edge case). Run on 2026-10-20 on macOS in the task worktree,
base e333200c.

## Commands and results

| Check | Command | Result |
|-------|---------|--------|
| Before the change, alone | `sh tests/test-secret-scan.sh` | exit 0; `PASS: 123`, `FAIL: 0` |
| After, alone (AC3) | the same | exit 0; `PASS: 123`, `FAIL: 0` |
| After, two at once (AC4) | `sh tests/test-secret-scan.sh > a.log 2>&1 & pa=$!; sh tests/test-secret-scan.sh > b.log 2>&1 & pb=$!; wait "$pa"; ra=$?; wait "$pb"; rb=$?` | `ra=0 rb=0`; both logs `PASS: 123`, `FAIL: 0` |
| The harness reports a failing copy | one copy of the script beside `sh -c 'exit 3'`, each with its own `wait` | `rc=0 rd=3`: the non-zero exit is reported |
| Before the change, two at once | the old script (`git show e333200c:tests/test-secret-scan.sh`) copied to a scratch `tests/` dir beside a symlink to `scripts/`, the second copy started 0.2 s after the first | one copy exit 0 with `FAIL: 0`, the other exit 1 with `FAIL: 2` |
| TERM mid-run | `TMPDIR=<scratch dir> sh tests/test-secret-scan.sh &`, `kill -TERM` after 3 s | before the kill, the scratch `TMPDIR` held `ralph-secret-scan-test.<6 chars>` and `ralph-secret-scan.<6 chars>`; after it, the dir was empty; the run exited 128, and a re-run exited 1 (see Notes) |
| run-verify | `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` | exit 0, `All verifiers passed` |

## Notes

- A first try of the edge case sent SIGINT to the backgrounded script; a
  non-interactive `sh` starts background jobs with SIGINT ignored, so the
  run went on to `PASS: 123`. TERM was used instead
- The TERM row shows that the `trap` removes both directories, not that
  the script stops on TERM. The `trap` runs no `exit`, the same as the
  base script's `trap` (line 41 at e333200c), so after it the script goes
  on until a command fails in the removed working directory and `set -e`
  ends it. The exit code depends on where TERM lands: 128 in the first run
  (a git command), 1 in a re-run (`rm: .gitallowed: No such file or
  directory` on stderr). The re-run's scratch `TMPDIR` was empty after it
  too. The script makes no other directory with `mktemp`, so a run that
  goes on after the `trap` leaves nothing new behind
- The suffixes that `mktemp` chose are left out
