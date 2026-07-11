# cli-stub-stdin-hang

- Status: Approved (autonomous goal session; user requested this fix PR explicitly)
- Owner: Claude Code
- Date: 2026-07-12
- Related request: Fix the CLI-stub stdin hang discovered during PR #116 (recorded in docs/tech-debt/README.md)
- Related issue: N/A
- Type: fix
- Branch: fix/cli-stub-stdin-hang

## Objective

`tests/fixtures/cli-stubs/{codex,claude}` drain stdin via `head -c 4096` for
call-log capture. The existing guard only skips the read for TTY stdin
(`[ -t 0 ]`). When stdin is a non-TTY descriptor that never reaches EOF (an
inherited open pipe — observed with a Claude Code background Bash task), the
stub blocks forever and the whole verify chain
(`test-ralph-cli-driver.sh` → `verify.local.sh` → `run-verify.sh`) hangs at
0% CPU with no output. Make the stubs hang-proof, harden the direct
invocation call sites, and add a regression test.

## Scope

1. **Stub fix (both stubs, same change)** — replace the TTY-only guard with a
   positive condition: drain stdin ONLY when it is a regular file
   (`[ -f /dev/stdin ]`), where EOF is guaranteed; otherwise set
   `stdin_buf=""` and skip. Rationale: every stdin-content assertion in the
   test suite goes through `run_agent ... < "$prompt_file"` (regular file);
   TTYs and pipes are exactly the descriptors that can block indefinitely.
   Update the stub header comments to document both hazard cases (TTY typing
   wait — #44 cycle-3 P2; open-pipe EOF wait — this fix).
2. **Call-site hardening** — in `tests/test-ralph-cli-driver.sh`, add
   `< /dev/null` to direct stub invocations that replicate the dispatcher
   shape without a stdin redirect (test 6a `codex exec review`, and the 6b
   claude-reviewer line if it lacks a redirect). Defense in depth: the test
   must not depend on the harness's inherited stdin.
3. **Regression test** — new test in `test-ralph-cli-driver.sh`: run each
   stub with stdin attached to a deliberately never-closing-within-margin
   pipe (`sleep 5 | stub ...`), assert the stub exits well before the writer
   does (elapsed < 3s) and still produces its call log / last-message file.
   A regressed stub would block until the writer exits (~5s) and fail the
   elapsed assertion deterministically.
4. **Tech-debt closure** — mark the corresponding row in
   `docs/tech-debt/README.md` as RESOLVED (strike-through + commit ref),
   matching the file's existing convention for resolved rows.

## Non-goals

- No production-script changes (`ralph-pipeline.sh` invokes the real codex
  CLI, which does not read stdin for `exec review`; adding redirects there is
  unnecessary churn and would require template mirrors).
- No general timeout wrapper in the stubs (macOS lacks `timeout` by default;
  a watchdog subshell adds complexity the positive `-f /dev/stdin` condition
  avoids).
- No changes to templates/base (tests are root-only; not scaffolded).

## Assumptions

- `/dev/stdin` resolves on macOS and Linux and `test -f` follows it to the
  underlying descriptor (regular file iff redirected from one). If
  `/dev/stdin` is absent the condition is false → skip drain → safe default.
- All existing stdin-content assertions use file-redirected stdin, so
  narrowing the drain condition does not break any current test (verified by
  running the full suite).

## Affected areas

- `tests/fixtures/cli-stubs/codex`
- `tests/fixtures/cli-stubs/claude`
- `tests/test-ralph-cli-driver.sh`
- `docs/tech-debt/README.md`

## Design decisions

1. **Positive condition (`drain iff regular file`) over negative
   (`skip iff TTY or pipe`)**: unlisted descriptor types default to the safe
   branch instead of the blocking one.
2. **Belt and braces**: stub-level fix (root cause) + call-site `< /dev/null`
   (defense) + elapsed-time regression test (detection). Matches the harness
   principle of promoting repeated mistakes into tests.
3. Critical forks: None (single reasonable approach; alternatives recorded in
   Non-goals).

## Acceptance criteria

- [ ] AC1: Both stubs no longer read stdin unless it is a regular file; the
  header comments document both hazard cases.
- [ ] AC2: `printf x | tests/fixtures/cli-stubs/codex exec review --base main`
  (open-pipe shape) exits immediately; regression test asserts elapsed < 3s
  for both stubs and passes.
- [ ] AC3: Direct dispatcher-shape invocations in the test file carry
  `< /dev/null`.
- [ ] AC4: Full `bash tests/test-ralph-cli-driver.sh` passes (all existing
  stdin-capture assertions still green); `./scripts/run-test.sh` passes.
- [ ] AC5: `./scripts/run-verify.sh < /dev/null` passes; the previously
  hanging invocation shape (`run-verify.sh` with inherited open-pipe stdin)
  completes — demonstrated by the regression test rather than by reproducing
  the background-task environment.
- [ ] AC6: tech-debt row marked RESOLVED with commit reference.

## Implementation outline

Single slice (4 files): stub edits → call-site redirects → regression test →
tech-debt row → run suite + run-verify → commit.

## Verify plan

- Static: `./scripts/run-verify.sh < /dev/null` (shellcheck covers the stubs
  if in scope; sync gates unaffected).
- Spec compliance: AC1–AC6 against the diff.
- Doc drift: tech-debt row consistency.
- Evidence: docs/reports/verify-2026-07-12-cli-stub-stdin-hang.md.

## Test plan

- Unit/regression: new elapsed-time cases + full test-ralph-cli-driver.sh.
- Full regression: `./scripts/run-test.sh`.
- Edge cases: stdin = /dev/null (drain skipped, no hang), stdin = regular
  file (drain still captures content — existing assertions), stdin = pipe
  (new regression case).
- Evidence: docs/reports/test-2026-07-12-cli-stub-stdin-hang.md.

## Risks and mitigations

- `test -f /dev/stdin` portability quirk on an untested platform → safe
  default is skip-drain (no hang; only the call-log stdin field is empty,
  which no assertion requires outside file-redirected cases).
- Elapsed-time regression test flakiness on a loaded machine → 3s threshold
  vs 5s writer gives a 2s margin; the assertion direction (must be FAST)
  cannot false-pass on a slow machine when the bug is present.

## Rollout or rollback notes

- Test-only change; revert the commit to roll back. No scaffold impact.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (fix/cli-stub-stdin-hang)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
