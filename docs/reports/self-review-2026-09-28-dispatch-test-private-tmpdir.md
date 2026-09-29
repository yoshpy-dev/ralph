# Review report: dispatch-test-private-tmpdir (self-review, cycle 1)

- Date: 2026-09-29
- Plan: docs/plans/active/2026-09-28-dispatch-test-private-tmpdir.md (issue #182)
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for `git diff main...HEAD -- tests/ docs/tech-debt/` at 0a1b0a8 (code commit 84c412f: `tests/test-ralph-dispatch.sh` case I, one deleted row in `docs/tech-debt/README.md`). Read in full: `tests/test-ralph-dispatch.sh`, `.claude/hooks/ralph-dispatch.sh`. No spec-compliance verdict, no test verdict, no documentation-drift audit.

## Evidence reviewed

- `git log main..HEAD`: 4 commits (3bc69eb, efd4ec1 plan; 84c412f code; 0a1b0a8 plan record). HEAD 0a1b0a8 and `git status --porcelain` empty at review start and after the probes.
- `git diff main...HEAD --stat`: `tests/test-ralph-dispatch.sh` +139/-23 (net +116), `docs/tech-debt/README.md` -1, plan +120.
- Callers of `TMPDIR` inside the suite: the only `mktemp` in the file is `workdir` at line 81, before the export at 106. `templates/base/scripts/run-verify.sh:59` and `:265` (reached by case H) use `${TMPDIR:-/tmp}`, so they now write `ralph-verify-scope.*` / `ralph-ev-prune.*` under `$workdir/shared-tmp`; neither matches `ralph-dispatch-*`.
- Removed variables: a grep for `i_tmpdir_base`, `i_tmp_before`, `i_tmp_after`, `i_leaked_tmp` in the test file exits 1 (none left).
- `docs/tech-debt/README.md`: a grep for `test-ralph-dispatch`, `pgrep`, `pkill`, `10-slow` exits 1 after the deletion (no dangling reference, and no row for the plan's pgrep/pkill non-goal either).
- Probes (a minimal fixture under the session scratchpad, `TMPDIR` pointed at the scratchpad, hook name `10-probe-r182.sh` so no `pkill`/`pgrep` pattern of the real suite can match it; nothing under the host temp directory was listed or deleted; the orphan was reaped by its own PID):
  - Process table while the dispatcher runs a `Stop.d` hook:
    ```
    22823 22771 sh ./.claude/hooks/ralph-dispatch.sh Stop
    22867 22823 sh ./.claude/hooks/Stop.d/10-probe-r182.sh
    22873 22867 sleep 30
    ```
    The hook's command line is the relative path the dispatcher builds from `./.claude/hooks/${event}.d` (`.claude/hooks/ralph-dispatch.sh`, `dirs=` line). It carries no `$workdir` component.
  - After `kill -TERM <dispatcher>`: dispatcher rc 143, the three `ralph-dispatch-*` files are gone, and `22873 1 sleep 30` is still alive (reparented to PID 1).
  - The same run wrapped as `inner.sh | cat` returned in 1 s. `lsof` on the orphan shows only fd 0 (the stdin buffer), fd 1 (the out temp file) and fd 2 (the dispatcher's stderr log). It does not hold the caller's pipe.
  - `/bin/bash` 3.2.57 runs an `EXIT` trap when the script is killed with TERM (rc 143, trap output written), so the cleanup comment's "signal" case holds for TERM/INT/HUP.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | robustness (concurrency) | The new `pkill -f "Stop.d/10-concurrent-slow.sh"` matches the concurrent fixture's hook in **every** run of this suite on the host, not just this one. The dispatcher runs hooks by relative path, so the process command line is `sh ./.claude/hooks/Stop.d/10-concurrent-slow.sh` with nothing per-run in it. The EXIT trap now runs this `pkill` on every suite exit. When suite A exits while suite B is between its started-marker poll and its presence `find`, A kills B's hook. B's dispatcher then takes the non-zero-rc branch, exits 143 and removes its temp files. B then records "concurrent dispatcher's temp files missing ... (fixture not effective)". The window is short, but this is a new false failure from the same "another session on this machine" class that #182 closes. The plan's own risk mitigation ("hook の path は一意にする") is not achieved. The `pkill` also targets the `sh` wrapper, which the dispatcher's `kill_child` has already stopped, while the process that actually outlives the stop is the `sleep` (see the next row). | `tests/test-ralph-dispatch.sh:92` (EXIT trap), `:517` (explicit stop), `:401` (hook path), `:427`/`:431` (the check that goes red); probe process table above | Put a per-run token into the hook file name and use the same string in both `pkill` calls (for example `10-concurrent-slow-$$.sh`). Alternatively, drop both `pkill` lines and rely on `kill -TERM "$concurrent_pid"` plus `wait`, which already stops the hook through the dispatcher's TERM trap. |
| LOW | robustness (process hygiene) | The concurrent hook's `sleep 30` outlives both the explicit stop and the EXIT trap. `kill_child` TERMs the hook's `sh`, which dies without forwarding TERM to its foreground `sleep`. The `sleep` is reparented to PID 1, and neither `pkill` pattern matches `sleep 30`. Each suite run therefore leaves a process alive for up to 30 s, holding fds to two unlinked temp files and to `$concurrent_out`. It does not hold the caller's pipe (probe: the pipeline returned in 1 s), so nothing hangs. Case I's own `sleep 5` has the same shape, bounded to 5 s, and predates this diff. | `tests/test-ralph-dispatch.sh:405`; probe `22873 1 sleep 30` after dispatcher rc 143 | Use `exec sleep 30` in the fixture hook, so TERM from `kill_child` reaches the `sleep` itself. With the MEDIUM fix, the `pkill` safety net can then go. |
| LOW | robustness / readability | The presence check at 427 and the absence check at 519 read `$TMPDIR`, and nothing establishes that the directory held no `ralph-dispatch-*` before the fixture started. Because the export at 106 is suite-wide, every dispatcher run in cases A-H writes to the same directory. A regression that leaves an A-H temp file behind would make the "fixture started effectively" PASS vacuous and the absence FAIL blame the concurrent dispatcher. `find "$TMPDIR"` also means the claim that `find` only targets `$workdir` depends on the export staying above it, while `find "$shared_tmp"` states it directly. The concurrent subshell (407-410) inherits `TMPDIR` implicitly, where case I's subshell sets its own at 449-450. | `tests/test-ralph-dispatch.sh:106`, `:427`, `:519`, `:407-410` | Record and assert an empty baseline of `$shared_tmp` before launching the fixture. Use `$shared_tmp` in both `find` calls. Optionally set `TMPDIR="$shared_tmp"` explicitly in the concurrent subshell, mirroring case I. |
| LOW | robustness | The private-dir leak check still filters `-name 'ralph-dispatch-*'`, a filter carried over from the shared-dir design. `$case_i_tmpdir` belongs to case I's dispatcher alone, so any entry left in it is a leak. The filter would miss a future temp file with another prefix. Nothing is missed today, because `$out_tmp.first` also matches. `2>/dev/null` also turns a missing directory into an empty result, which reads as PASS. The exit-code assertion catches that case today, because `mktemp` fails and rc is not 143. | `tests/test-ralph-dispatch.sh:487` | `find "$case_i_tmpdir" -mindepth 1 -maxdepth 1`, and fail explicitly if the directory is missing. |
| LOW | readability (output) | The presence PASS line embeds three absolute temp paths with random suffixes, plus a trailing space left by `tr '\n' ' '`. No other PASS line in the file carries a path. The line changes on every run, so two green results blocks never match, and any test report that pastes the block carries host temp paths. The two leftover FAIL messages (490, 523) use raw newline-separated `find` output, while 427 uses `sort \| tr`. | `tests/test-ralph-dispatch.sh:427`, `:429`, `:490`, `:523` | Print a count on PASS (for example `(3 files)`). Keep the names on FAIL only, formatted the same way in all three places. |
| LOW | comments | Four comment statements do not match the file. (1) Lines 97-98 and 101 cite "AC-3 in the plan" and "(AC-7)", but the file never names the plan, and the plan moves to `archive/` at `/pr`. "Run separately from scratch" reads as "from zero" rather than "from a scratch copy", and describes a one-off procedure that is not in the repo. Line 21's "(AC-5)" is precedent, but the issue number (#182) survives archival. (2) The same block says the concurrent fixture uses this directory. It omits that the export is suite-wide, so every A-H dispatcher run and case H's `run-verify.sh` `mktemp` calls land there too. That suite-wide reach is what makes "a leaked temp file can never land outside $workdir" true. (3) Line 387, "This reproduces the real flake", overstates the fixture. The fixture reproduces the condition behind the flake, and against the new check it stays green by design. (4) Lines 85-86 say `concurrent_pid` "is set only while that fixture's dispatcher is actually running". In fact it is set from `$!` at 411 before anything confirms a start, and it keeps a dead PID if the fixture exited early, until line 518 clears it. | `tests/test-ralph-dispatch.sh:85-87`, `:97-103`, `:387` | Replace the AC references with `#182`. Say that the export applies to the whole suite. Use "reproduces the condition behind the flake". Use "set between launch and case I's explicit stop". |

No CRITICAL or HIGH findings.

## Positive notes

- The private `TMPDIR` is set inside the backgrounded subshell before `exec`, so `$!` still holds the dispatcher's own PID and the existing "signal the dispatcher, not a wrapper" design of case I is preserved (`:447-452`).
- Deleting `rm -f $i_leaked_tmp` removes the path by which a leak check could delete another session's live temp files. The deleted line also expanded its argument unquoted.
- The fixture validates itself. The presence assertion fails when the second dispatcher's files are missing, so a fixture that silently stopped working cannot keep the suite green. The dispatcher creates all three temp files before it runs the hook, so "started marker exists" implies "files exist", and that check is deterministic.
- The separate event (`Stop`) and hook name keep the fixture out of case I's `pgrep -f "PreCompact.d/10-slow.sh"` and `finished_marker` checks.
- `concurrent_pid=""` is initialized before the trap is installed, so the EXIT trap is `set -u`-safe on every path.

## Coverage gaps

- The suite was not run. Behavior claims come from reading the file plus the scratchpad probes above. The 29/0 result, five consecutive passes, and the AC-3 red reproduction are the verifier's and tester's to confirm.
- The orphan and the process command line were probed on macOS (`/bin/sh` is bash 3.2). Under dash the TERM-to-`sh` semantics are the same by POSIX, but this was not re-probed on Linux.
- The cross-run `pkill` race in the MEDIUM row was established from the command-line probe and the code path. It was not reproduced with two live suites, because doing so would signal other agents' concurrent runs on this machine.
- The pre-existing absence of a `mktemp -d` success check (line 81) now also feeds the exported `TMPDIR` (`/shared-tmp` if `workdir` were empty). This predates the diff and was not raised as a finding.

## Answers to the requested checks

1. Suite-wide export: cases A-H keep their assertions. Their dispatcher temp files, and case H's `run-verify.sh` temp files, now land in `$workdir/shared-tmp`. No other `mktemp` in the file runs after the export. The comment explains the shared directory but not its suite-wide reach (LOW, comments).
2. Fixture ordering is correct: start, started-marker poll, presence, case I on the private directory, TERM plus `wait`, `pkill`, clear the PID, absence. If the trap fires before the PID is set or after it is cleared, only the `pkill` runs. If the started marker never appears, the FAIL text names the cause ("did not start within 5s (started marker never appeared)"), and the suite still stops the fixture. The absence check can then double-report if TERM landed before the dispatcher installed its traps, but the first FAIL names the cause. The `pkill` itself is the MEDIUM finding.
3. `find "$case_i_tmpdir"` is the right replacement for the old set difference, and the `rm -f` is gone. Every `rm` target derives from `$workdir`. Both `find "$TMPDIR"` calls resolve under `$workdir` today, but only because of line 106 (LOW, robustness / readability).
4. The PASS line with paths is noise; the FAIL lines should keep the paths (LOW, output).
5. The header (22-34) and the case I banner (351-357) match the code. Four comment statements elsewhere do not (LOW, comments).
6. Timing: the 5 s poll is generous for a process that only has to `mktemp` three files and `touch` a marker. `sleep 30` covers case I's worst case (the 5 s poll plus a 5 s deferred TERM on a regression) with more than 2x margin. TERM plus `wait` is deterministic, because the dispatcher's TERM trap removes its files before exiting. The one remaining flake source introduced here is the cross-run `pkill` (MEDIUM).

## Recommendation

- Merge: yes after the MEDIUM is addressed or consciously accepted. No CRITICAL or HIGH findings. The MEDIUM and the `sleep 30` orphan are a few lines each: a per-run token in the hook name, and `exec sleep 30`.
- Follow-ups:
  1. Fix or accept the cross-run `pkill` (MEDIUM) in this cycle, since it reintroduces the concurrent-session flake class this PR closes.
  2. The plan's Non-goal says the pre-existing `pgrep`/`pkill -f "PreCompact.d/10-slow.sh"` collision between concurrent suite runs is "記録にとどめる", but `docs/tech-debt/README.md` has no row for it. Hand this to `/sync-docs` to decide, and mention the `Stop.d` pattern too if the MEDIUM is not fixed.
