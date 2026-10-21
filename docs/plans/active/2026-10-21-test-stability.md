# test-stability

- Status: Approved
- Approved: 2026-10-21 sha256:edae494b2ed1
- Owner: Claude Code
- Date: 2026-10-21
- Related request: the third PR of the series that fixes the stage 9 trial findings ("2. 修正可能なものは修正してください", 2026-10-21): テストの安定 (F9, F10, the evidence table's G1), plus the doctor-test row and the dry-run cut gap that PR #246's test report found
- Related issue: N/A
- Type: fix
- Branch: fix/test-stability

## Objective

Stop four kinds of test failure that come from the test or the runner, not
from the code under test, and close the test gaps the trial and PR #246
left, so a gate, a regate, or a local run-verify fails only on a real
regression.

- F9: `TestRegate_FetchFailures/the_fetch_runs_past_the_bound` races git
  against the 300 ms bound. Its stub upload-pack starts with
  `exec >/dev/null 2>&1 </dev/null`, which closes the pipe git reads, so
  git reads EOF and exits 128 (`Could not read from remote repository`)
  after git's and the shell's start-up. Measured by hand on this Mac with
  the same stub and an isolated git config: 0.19 to 0.28 s, close to the
  bound. With stdout left open (`exec 2>/dev/null </dev/null`) git waits
  for the stub (3.95 s with `sleep 3`)
- F10: `run_tests` in the Go pack (`packs/languages/golang/verify.sh` and
  its template copy) runs `go test ./...` with no `-timeout`, so each
  package gets the 10-minute default; `ralph org gate` and `ralph org
  regate` reach the same script through `scripts/run-test.sh`, so their
  result depends on whether the caller's shell sets `GOFLAGS`
- The evidence table's G1: no suite runs `scripts/check-sync.sh` with the
  two `ROOT_ONLY_EXCLUSIONS` entries that fixed F8
- Three doctor tests fail under heavy load because `probeBinary` gives a
  stub `--version` 5 s
- PR #246's test report G1: no test checks that the dry-run `sent` event
  cuts the body

## Scope

- S1 (F9): the stub upload-pack of `the fetch runs past the bound` keeps
  its stdout (git's pipe) open and sends only its stderr away, with a
  comment that says why; it sleeps 20 s and then writes a marker file, and
  the subtest checks that the marker does not exist when Regate returns,
  so a fetch that ended because the stub finished fails the subtest
- S2 (F10): `run_tests` runs `go test ./...` with `GOFLAGS` set to
  `-timeout=30m` followed by the effective `GOFLAGS` (what `go env GOFLAGS`
  prints: the environment variable, else the value `go env -w` wrote; the
  variable itself when `go env` fails). Go applies the entries in order, so
  a timeout the caller set, in any form Go accepts, comes later and wins.
  Both copies stay identical; the pack README says so; a shell suite
  checks it with a fake `go` and with the real one
- S3 (G1): a new suite `tests/test-check-sync.sh` runs a copy of
  `scripts/check-sync.sh` over a scratch tree
- S4 (doctor): the probe's bound becomes a package variable
  (`doctorProbeTimeout`, 5 s) that the error text names; `internal/cli`'s
  `TestMain` raises it for the package's tests; a unit test of the
  timeout path sets it small; `TestCheckScaffoldIntegrity_CorruptManifest_StrictFails`
  prints doctor's output when a run fails
- S5 (#246's G1): a test sends a Japanese body over 200 bytes with
  `DryRun: true` and checks the cut in the recorded `sent` event
- S6 records: the five tech-debt rows (F9, F10, the doctor tests, G1, the
  dry-run cut) are resolved with the convention of the rows around them;
  the evidence table's F9, F10, and G1 rows name this branch and its F12
  and F14 rows name #246; the spec's Status and the evidence doc's PR list
  add #246 and count 29 PRs, and the evidence doc's "NFR 入れ方" row counts
  it; FR-7, whose only contradicting finding is F10, becomes `[x]` with its
  note, and the counts (spec 21 and 11, evidence table 20 and 11, since the
  table leaves FR-12 out) and the lists of contradicting findings follow

## Non-goals

- Making the org and cli test packages faster (the F10 row's other
  trigger, "find the slowest tests with `go test -json`")
- Changing `probeBinary`'s 5 s bound for users of `ralph doctor`
- Changing `regateBoundedGit` or any regate behavior: F9 is in the test
- Splitting `regate_test.go` (the tech-debt row on file size)
- Re-running a trial or a live regate under load
- The other series PRs (座席の環境, 分割計画の前文, 見張りの escalation,
  busy な leader, 文書のまとめ)
- Cutting a release (the user cuts it)

## Assumptions

- git reads the server's first message before it writes on a local fetch
  (protocol v2 too), so an upload-pack that holds its stdout open and
  writes nothing keeps git waiting until the stub exits or git is killed.
  The fetch runs in ralph's process group (`regateRemoteGit`,
  `ownGroup=false`), so the bound's cancel kills git only and the stub's
  sleep outlives it; with its stderr on `/dev/null` and its stdout on the
  dead git's pipe it holds none of the Go side's pipes
- `go help environment`: flags on the command line are applied after
  `GOFLAGS` and override it, so an unconditional `-timeout 30m` would
  override a caller's `GOFLAGS=-timeout=60m`
- `go env GOFLAGS` prints the effective value: checked by hand on
  2026-10-21 (go 1.26) with a scratch `GOENV`, it printed nothing with
  neither set, `-timeout=60m` after `go env -w GOFLAGS=-timeout=60m` with
  the variable unset, and `-count=1` with `GOFLAGS=-count=1` in the
  environment over that file
- Go applies `GOFLAGS` entries in order and accepts quoted entries:
  checked by hand on 2026-10-21 on go1.26.0 and go1.25.8 with a test that
  reads `t.Deadline()`. With no `GOFLAGS` the deadline was 10 minutes away;
  `-timeout=30m` gave 30; `-timeout=30m` followed by `-timeout=45m`,
  `--timeout=45m`, `-test.timeout=45m`, `--test.timeout=45m`,
  `"-timeout=45m"`, or `'-timeout=45m'` gave 45; `-timeout=45m -timeout=30m`
  gave 30; `-timeout=30m` followed by the `go env -w` value `-timeout=45m`
  gave 45. With `GOFLAGS=-timeout=30m`, `go build`, `go vet`, `go list`,
  `go run`, `go mod tidy`, and `go env` ran with rc 0 (a go command that
  does not know a `GOFLAGS` flag skips it)
- `ralph upgrade` updates a project's `packs/languages/golang/verify.sh`
  from the template like any core file (`internal/upgrade/replaceplan.go`
  keeps only absent pack namespaces), so S2 reaches downstream Go projects
  at their next upgrade
- No `internal/cli` test calls `t.Parallel()`, so a package variable that
  `TestMain` sets and one test changes with `t.Cleanup` is not raced
- `tests/test-*.sh` are found by `scripts/verify.local.sh` and CI without a
  list to edit

## Affected areas

- `internal/org/regate_test.go` (S1)
- `packs/languages/golang/verify.sh`, `templates/packs/golang/verify.sh`,
  `packs/languages/golang/README.md`, `templates/packs/golang/README.md`
  (S2)
- `tests/test-golang-pack-test-timeout.sh` (new, S2),
  `tests/test-check-sync.sh` (new, S3)
- `internal/cli/doctor.go`, `internal/cli/main_test.go`,
  `internal/cli/cli_test.go` or a doctor test file, and
  `internal/cli/doctor_scaffold_test.go` (S4)
- `internal/org/verbs_test.go` (S5)
- `docs/tech-debt/README.md`, `docs/evidence/org-spec-evidence-2026-10-21.md`,
  `docs/specs/2026-10-07-org-multi-org-director.md` (S6)

## Visual review

- Page: `.harness/state/plan-visual/test-stability.html` (current form:
  two easy figures, `overview` and `easy-timeout`; in the fold
  `detail-overview`, `detail-fetch-race`, and `detail-timeout`)
- Lint: `./scripts/plan-visual.sh lint` exits 0
- Self-check: shot the full page and all five figures and read every PNG;
  the fetch-race figure's notes crossed the bound line and were moved right
  of it, then every PNG was clean. After the Codex plan advisory the three
  fold figures were redrawn (the marker of S1, `GOFLAGS` with the default
  first in S2), linted again (exit 0), and shot and read again: clean

## Design decisions

- The pack's test timeout is a default that `GOFLAGS` can replace (S2).
  Alternatives: always pass `-timeout 30m`, which overrides a caller's own
  `GOFLAGS=-timeout=...` because command-line flags win; or leave the pack
  alone and have `ralph org gate` and `regate` put `GOFLAGS=-timeout=30m`
  in the scripts' environment, which fixes the gate but not a local
  run-verify and puts Go knowledge in the org runtime. 30m is the bound
  this repo has run with since 2026-10-19 (the longest local package run
  on record, 1537 s under a load near 50, fits; CI's whole verify job takes
  about 4 minutes). A downstream project's hung test now fails after 30
  minutes instead of 10; the README says how to set another bound
- The default goes first in `GOFLAGS` and the script reads no flag. A
  first draft looked for a timeout in the value and passed `-timeout 30m`
  on the command line when it found none; the Codex plan advisory showed
  that a quoted entry (`"-timeout=45m"`, which Go accepts) would be missed
  and then overridden, since the command line wins. Putting the default
  before the effective value lets Go's own reading decide, for every form.
  The effective value comes from `go env GOFLAGS` (consult-plan-teststab:
  `$GOFLAGS` alone misses a value set with `go env -w`, and setting the
  variable hides that file's value), with `$GOFLAGS` when `go env` fails
- The test processes see the new `GOFLAGS` in their environment. A go
  command a test runs skips `-timeout` unless it is `go test`, which then
  gets the same default
- The F9 stub sleeps 20 s and then writes a marker, and the subtest fails
  if the marker exists when Regate returns (Codex plan advisory: the runner
  checks the expired bound before the exit code, so a fetch that the bound
  did not stop but that ended when the stub finished would still read "ran
  past ... was stopped"). 20 s leaves a wide margin between the 300 ms
  bound and the stub's own end; the orphaned stub holds no pipe of the test
  and its write into the removed temp dir fails silently
- The doctor tests raise the bound in `TestMain` (as it already sets the
  other doctor seams) rather than in each of the three tests, so a new
  doctor test with stub binaries is covered too; the timeout path keeps a
  test of its own with a small bound
- FR-7 becomes `[x]`: its only contradicting finding is F10, its live
  records are the approval, merge-guarantee, and trial records already in
  the evidence table, and the F10 fix is checked by the pack suite and by
  a run of `./scripts/run-test.sh` with no `GOFLAGS`. Its note says that
  the `-timeout=30m` under which the trial's second regate passed is now
  the pack's default, and that a regate under load was not re-run (the
  same treatment as F8 in #245: fixed and tested, no new trial)
- The F9 records state the cause as found: the stub closes git's pipe, so
  git reads EOF and exits 128, racing the bound. Why the trial's failures
  came under heavy load (where both sides slow down) is not explained; the
  records say so and keep any guess, such as a late Go timer, as a guess.
  The AC1 run with a 2 s bound shows the race without depending on load
- Critical forks: None. Each choice is local to one script or test and
  cheap to reverse

## Acceptance criteria

- [x] AC1: the stub of `the fetch runs past the bound` leaves stdout open
  (no `>/dev/null` on fd 1), sends stderr to `/dev/null`, sleeps 20 s, and
  then writes a marker; the subtest fails if the marker exists when Regate
  returns; the subtest passes 20 of 20 alone with `-count=20` on both
  toolchains. Regression evidence, in scratch copies: with
  `regateGitTimeout` set to 2 s for that subtest, the base stub gives
  "failed (exit 128)" and the new stub gives "ran past 2s and was
  stopped"; with `exec.Command` in place of `exec.CommandContext` in
  `regateBoundedGit` (no cancellation), the new subtest fails on the marker
- [x] AC2: with `HARNESS_VERIFY_MODE=test`, a fake `go` on `PATH` (which
  prints `$GOFLAGS` for `go env GOFLAGS`, like go) records what `go test`
  gets: with no `GOFLAGS`, `GOFLAGS=-timeout=30m` and `test ./...` in each
  Go root; with `GOFLAGS=-count=1`, `-timeout=30m -count=1`; with
  `GOFLAGS='"-timeout=45m"'`, `-timeout=30m "-timeout=45m"`; with no
  `GOFLAGS` and `go env GOFLAGS` printing `-timeout=45m` (a `go env -w`
  value), `-timeout=30m -timeout=45m`; with `go env` failing and
  `GOFLAGS=-count=1`, `-timeout=30m -count=1`; static mode makes no
  `go test` call. With the real `go` (skipped, with a line saying so, when
  `go` is not on `PATH`), a scratch module's test that writes its
  `t.Deadline()` sees 30 minutes with no `GOFLAGS` and 45 with
  `GOFLAGS=-timeout=45m`, with `GOFLAGS='"-timeout=45m"'`, and with
  `-timeout=45m` written by `go env -w` into a scratch `GOENV`. The two
  copies of `verify.sh` are identical (`scripts/check-sync.sh`)
- [x] AC3: `tests/test-check-sync.sh` runs a copy of `scripts/check-sync.sh`
  over a scratch tree: `docs/reports/review-x.md` and
  `docs/reports/org-manifest-x.md` give rc 0; adding
  `docs/reports/other-x.md` gives rc 1 with a `ROOT_ONLY` line naming it;
  `docs/reports/reviewer.md` and `docs/reports/x-review-y.md` (a prefix
  that matches only part of the name, or not at the start) are
  `ROOT_ONLY` too. The suite fails on base's script with either entry
  removed from `ROOT_ONLY_EXCLUSIONS` (checked in a scratch copy)
- [x] AC4: `probeBinary` reads `doctorProbeTimeout` (5 s by default) and
  its timeout error says the bound it used, which reads "timed out after
  5s" by default; `TestMain` in `internal/cli` raises it; a test with a
  stub that sleeps past a 100 ms bound gets "timed out after 100ms";
  `TestCheckScaffoldIntegrity_CorruptManifest_StrictFails` includes
  doctor's output in its failure messages. The three doctor tests of the
  row pass 10 of 10 alone
- [x] AC5: a test sends a Japanese body over 200 bytes with byte 200 inside
  a character and `DryRun: true`; the recorded dry-run `sent` event's
  Details are valid UTF-8, hold no U+FFFD, end in `...(truncated)`, and
  keep a prefix of at most 200 bytes. A mutant that stops cutting in the
  dry-run branch fails it
- [x] AC6: the five tech-debt rows are resolved with the convention of the
  rows around them, each naming what closes it; the evidence table's F9,
  F10, and G1 rows name this branch and its F12 and F14 rows name #246;
  the spec's marks count 21 `[x]` and 11 `[ ]` (FR-12 included), its
  Status says 11 items stay `[ ]`, lists #246 ("試行の所見 F12・F14 の直し"),
  says 29 PRs, and has F10 gone from its list of contradicting findings;
  FR-7 is `[x]` with a note; the evidence doc's FR-7 row reads `[x]`, its
  count line reads 20 `[x]` and 11 `[ ]` (FR-12 is not in its table) with
  FR-7 gone from the list, its PR list has #246 and says 29, and its "NFR
  入れ方" row counts #246 with the other fixes (26 PRs whose plan and four
  reports were checked by file name); #246 reads `MERGED` with
  `gh pr view` (read only)
- [x] AC7: `./scripts/run-verify.sh` exits 0 (with `GOFLAGS=-timeout=30m`),
  `GOTOOLCHAIN=go1.25.8 go test -timeout 30m ./...` passes, and
  `env -u GOFLAGS ./scripts/run-test.sh` exits 0 (the pack's own bound)

## Implementation outline

1. S1: edit the stub and its comment in `internal/org/regate_test.go`
2. S2: edit `run_tests` in both `verify.sh` copies and both READMEs; add
   `tests/test-golang-pack-test-timeout.sh` (a fake `go` on `PATH`, the
   style of `tests/test-language-pack-monorepo-roots.sh`, then the real
   `go` on a scratch module)
3. S3: add `tests/test-check-sync.sh` (a scratch tree with
   `templates/base/`, `templates/base/.ralph/core/AGENTS.core.md`,
   `AGENTS.md` and `templates/base/AGENTS.md` with the managed block,
   `templates/packs/`, `packs/languages/`, and `docs/reports/`)
4. S4: `doctorProbeTimeout` in `internal/cli/doctor.go`, `TestMain`, the
   timeout test, the scaffold test's output
5. S5: the dry-run test in `internal/org/verbs_test.go`
6. S6: records

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` (gofmt, vet,
  the shell syntax checks of `scripts/verify.local.sh`, check-sync,
  check-skill-sync); `sh -n` and `bash -n` on the new suites;
  `GOTOOLCHAIN=go1.25.8 go vet ./internal/...`
- Spec compliance criteria to confirm: AC1 to AC7; the five rows and the
  evidence table agree with the code; the spec's marks are 21 and 11, the
  evidence table's 20 and 11, and both docs' counts, lists, and PR numbers
  match them; FR-7's note names this branch
- Documentation drift to check: the pack READMEs, any doc that says the
  Go pack runs `go test ./...` with go's default bound or tells people to
  set `GOFLAGS=-timeout=30m` for run-verify or a gate (`git grep`),
  `docs/recipes/` for language packs
- Evidence to capture: `docs/evidence/verify-2026-10-21-test-stability.log`

## Test plan

- Unit tests: S4's timeout test, S5's dry-run test
- Integration tests: `tests/test-golang-pack-test-timeout.sh`,
  `tests/test-check-sync.sh`, run through `./scripts/run-test.sh`
- Regression tests: AC1's scratch run with a 2 s bound on base and HEAD and
  its `exec.Command` mutant; AC3's scratch runs with an exclusion removed;
  AC5's mutant; `-count=20` of the F9 subtest on both toolchains; the three
  doctor tests 10 of 10 alone
- Edge cases: `GOFLAGS` with another flag; a quoted entry; a timeout only
  in the `go env -w` file; `go env` failing; static mode makes no
  `go test` call; a prefix that matches only part of a file name
- Evidence to capture: `docs/evidence/test-2026-10-21-test-stability.log`,
  with the load average at the start and end of each long run

## Risks and mitigations

- A downstream project's hung test takes 30 minutes to fail instead of 10.
  The README says how to set a shorter bound with `GOFLAGS`, and the PR
  body says it for the release notes
- A later Go release could stop applying `GOFLAGS` entries in order, and
  the default would then override the caller's timeout. The suite's real
  `go` cases fail if that happens on the toolchain CI or a maintainer uses
- An invalid `GOFLAGS` makes `go env` fail; the script then uses the
  variable as it is, and `go test` fails on it as it does today
- The F9 fix could hide a real regression of the bound. The marker catches
  a fetch that the bound did not stop (AC1's `exec.Command` mutant), and the
  "ran past" assertion catches a bound that is not reported
- The orphaned `sleep 20` of the F9 stub outlives its test for up to 20 s.
  It holds no pipe of the test process; its marker write into the removed
  temp dir fails with its stderr on `/dev/null`
- Raising `doctorProbeTimeout` in `TestMain` makes a stub that really
  hangs cost a minute instead of 5 s. No current test uses a hanging stub
  except the new one, which sets its own bound

## Rollout or rollback notes

- Revert the PR to go back. The pack change reaches downstream projects at
  their next `ralph upgrade`; the PR body names it for the release notes

## Open questions

- None

## Progress checklist

- [x] Plan reviewed (consult-plan-teststab: fix and go, 3 points applied;
  Codex plan advisory: 2 MEDIUM findings, both applied, the quoted
  `GOFLAGS` entry and the cancellation the subtest could not see)
- [x] Plan approved (auto-approved under the user's standing instruction)
- [x] Branch created
- [x] Implementation started
  - S1 3f8ccd9b: the stub keeps stdout open, sleeps 20 s, then writes
    `upload-pack-ended`; a `t.Cleanup` of the case fails the subtest if it
    exists. 20/20 on go1.26.0 and go1.25.8; with a 2 s bound the base stub
    gives "failed (exit 128)" and the new one "ran past 2s and was
    stopped"; the `exec.Command` mutant fails on the marker
  - S2 db1e5ac7: `run_tests` and both READMEs; the suite has 34 checks
    (24 with a fake go, 10 with the real one), 34/34 on both toolchains,
    24 plus a SKIP line with no go on PATH; mutants m1 (default last), m2
    (no `go env`), m3 (base), m4 (command line) fail 11, 3, 11, 13 checks.
    Deviation from the Risks section: `go env GOFLAGS` does not fail on an
    invalid value (go1.26 prints `notaflag` with rc 0); `go test` then
    fails on it as it does today, so the behavior is the same and only that
    sentence of Risks was wrong
  - S3 dc58715a: the first implementer stopped on an API connection error
    after writing an untracked draft; a second one reviewed it, fixed its
    tree comment, made the other-x.md case add to the two excluded reports
    (AC3's "adding"), and committed it. 16/16; mutants m1 and m2 (an F8
    entry removed) fail 3 checks each, m3 (`docs/reports/` added) 6, m4
    (substring match) 2, the nested case only: the pattern is a whole path,
    so `x-review-y.md` does not contain it; the x-review-y case catches a
    match on the file name after the directory (m7, 2 checks). The script
    before F8 (9349914b^) fails 5
  - S4 439ae909: `doctorProbeTimeout` (5 s), TestMain raises it to a
    minute, `TestProbeBinary_TimeoutNamesTheBound` (stub `exec /bin/sleep
    2`: past 100 ms and short of 5 s, so a probe that ignores the variable
    fails), the scaffold test prints doctor's output. The three tests 10/10
    on both toolchains; `internal/cli` 246 s with a load average of 150 at
    the start. A fourth test failed the same way on 2026-10-13
    (`TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected`); the TestMain
    comment names it
  - S5 43df3185: `TestOrgSend_DryRun_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`
    sends the Japanese body of the real-send test with `DryRun: true` (no
    spawn: the dry-run branch returns before `findSeat`) and checks no
    driver call, a last event `sent` with `DryRun`, and the four Details
    conditions. Both toolchains pass it; mutant M1 (no cut in the dry-run
    branch) fails it on the suffix, M2 (a byte cut in the dry-run branch)
    on U+FFFD and on the 200-byte prefix, and no other test fails under
    either
  - S6 58f2624f and 4f83c84c: the five tech-debt rows resolved; the
    evidence table's F9, F10, G1 name this branch, F12 and F14 name #246,
    and F10's 食い違う cell reads いいえ(今は); FR-7 `[x]` with its note;
    spec 21 and 11, evidence table 20 and 11; #246 in both PR lists, 29
    PRs, all read `MERGED` with `gh pr view` on 2026-10-22; "NFR 入れ方"
    counts 26. The doctor row's opening now reads "Doctor tests such as".
    The evidence doc's sentence "(番号はまだない)" became one that writes
    merged PRs by number. 4f83c84c took out a clause that named the AC7
    run of `./scripts/run-test.sh` before it ran; the FR-7 notes name the
    pack suite only
  - `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` at 818bd4b1: rc 0 in
    13 min (19:30:48Z to 19:43:59Z, load average 4.7 at the start and 6.7
    at the end; `internal/cli` 240 s, `internal/org` 312 s)
  - Self-review LOW L1 (the new suites resumed after a signal) and L3 (two
    names for one scratch path) fixed in 6d990dfc; L2 (the duplicated
    checks of the two Japanese-body tests) kept
- [x] Review artifact created (docs/reports/self-review-2026-10-21-test-stability.md, c02647ca and 44c9f87a)
- [x] Verification artifact created (docs/reports/verify-2026-10-21-test-stability.md, 24df95dd)
  - /test (d4918ed9) pass: `env -u GOFLAGS ./scripts/run-test.sh` rc 0 in
    773 s (load 5.2 to 5.0; `internal/cli` 251.7 s, `internal/org`
    314.2 s), `GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./...`
    rc 0 in 332 s, the F9 subtest 20/20 and the doctor tests 40/40 on both
    toolchains, 20 of 22 mutants red (m6 is equivalent). Its G1 (no test
    pins the raise in `TestMain`, mutant D3 survived) closed in ec852215
    with `TestMain_RaisesTheDoctorProbeTimeout`; D3 then fails only that
    test (813 others pass)
- [x] Test artifact created (docs/reports/test-2026-10-21-test-stability.md, d4918ed9 and d34e60c6)
- [ ] PR created
