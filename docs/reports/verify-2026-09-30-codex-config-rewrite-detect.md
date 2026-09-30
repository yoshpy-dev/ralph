# Verify report: codex-config-rewrite-detect

- Date: 2026-09-30
- Plan: docs/plans/active/2026-09-30-codex-config-rewrite-detect.md
- Verifier: verifier subagent (Claude), cycle 1
- Scope: `git diff main...HEAD` at commit `72756be` (7 files besides the plan
  itself: `scripts/ralph-worktree.sh` + template, `tests/test-ralph-worktree.sh`,
  `docs/recipes/codex-setup.md` + template, `docs/evidence/codex-config-rewrite-2026-09-30.md`).
  Spec compliance (AC-1..AC-7) and static analysis only; no independent
  behavioral test authoring — `tests/test-ralph-worktree.sh` was run once,
  unmodified, to read its own pass/fail output as AC evidence (the tester
  owns the behavioral pass/fail verdict).

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `cmp scripts/ralph-worktree.sh templates/base/scripts/ralph-worktree.sh` | PASS | byte-identical |
| `cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md` | PASS | byte-identical |
| `bash -n scripts/ralph-worktree.sh` | PASS | no syntax errors |
| `sh -n tests/test-ralph-worktree.sh` | PASS | no syntax errors |
| `shellcheck -S warning scripts/ralph-worktree.sh` | PASS | 0 warnings |
| `shellcheck -S warning tests/test-ralph-worktree.sh` | PASS | 0 warnings |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS | full scope (per issue #190, default scope would skip packs on this pushed branch); evidence: `docs/evidence/verify-2026-09-30-060451.log`; includes `check-sync.sh` (DRIFTED=0), `check-pipeline-sync.sh`, `check-skill-sync.sh`, `check-template-purity.sh`, golang verifier, branch secret scan — all `[ok]`/`OK` |
| `./scripts/check-sync.sh` (standalone) | PASS | `IDENTICAL: 159, DRIFTED: 0, ROOT_ONLY: 0` |
| `sh tests/test-ralph-worktree.sh` (read once for AC evidence, not authored/altered) | PASS | `105 passed, 0 failed, 105 total` |
| Mutation: `print "MATCH"` -> `print "NOMATCH"` (detector always false) | Confirms AC-5 red evidence | 20/105 assertions fail (all "specific message" cases, incl. case 1/18/19), run in an isolated scratch copy, not committed |
| Mutation: `is_sep_header` forced to always return true (any table name accepted) | Confirms AC-5 red evidence | fails case 6 and case 17 |
| Mutation: `has_multiline` forced to always return 0 (multi-line-string guard removed) | Confirms AC-5 red evidence | fails case 10 |
| Mutation: exact-porcelain guard loosened to substring match on `.codex/config.toml` | Confirms AC-2(b)/(e) evidence | fails case 5 (other dirty file), 8 (staged), 9 (partially staged) |

All four mutations were applied to throwaway copies under the scratch
directory and run against the unmodified test file; the worktree's tracked
files were never touched (`git status --porcelain` stayed empty throughout,
confirmed before and after).

## Spec compliance (AC-1..AC-7)

- **AC-1** (rewrite shape -> stop with specific message + recovery commands): PASS.
  `scripts/ralph-worktree.sh:174-186` gates on `dirty` being exactly
  `" M .codex/config.toml"` (scripts/ralph-worktree.sh:174) before calling
  `codex_config_external_rewrite_only` (scripts/ralph-worktree.sh:103-162).
  On match, the message names the rewrite (`:177`) and prints
  `git -C ${qroot} diff -- .codex/config.toml` (`:179`),
  `git -C ${qroot} checkout -- .codex/config.toml` (`:181`), and
  `git -C ${qroot} status --porcelain` (`:183`), with `qroot` produced via
  `printf -v qroot '%q' "$root"` (`:176`) — no `templates/` or `cmp` mention.
  `ensure_worktree` calls `validate_clean_base "$base"` (`:286`), so `ensure`
  goes through the same path. Test case 1 (`tests/test-ralph-worktree.sh:296-315`)
  and case 19 (space-in-path, `:558-571`) both execute the printed checkout
  command verbatim and then confirm `validate-clean-base` passes; both PASS
  in the live run (105/0 total).
- **AC-2** (all six negative shapes fall back to the generic message): PASS.
  Verified in code and by the live test run: (a) value change alongside the
  match shape — case 4 (`:344-348`), guarded by the byte-exact comparison in
  the `while` loop (`:136-146`); (b) another dirty tracked file — case 5
  (`:350-355`), guarded by the exact-porcelain string check (`:174`), also
  confirmed by the mutation above; (c) non-policy table appended — case 6
  (`:357-375`), guarded by `is_sep_header` (`:113-115`, `:152-156`), also
  confirmed by mutation; (d) untracked-only — case 7 (`:377-382`), excluded
  by the porcelain check; (e) staged / partially staged — cases 8-9
  (`:384-397`), excluded by the porcelain check, also confirmed by mutation;
  (f) HEAD-side multi-line string — case 10 (`:399-420`), guarded by
  `has_multiline` (`:116-118`, `:124`, `:129`), also confirmed by mutation.
  All six cases PASS in the live run.
- **AC-3** (revised: raw-line comparison; only HEAD's comment/blank lines are
  droppable): PASS. Comment-only deletion (case 2, `:317-331`) and
  table-only addition with comments kept (case 3, `:333-342`) both give the
  specific message. Comment content changed (case 14), comment added (case
  15), indentation-only change (case 16), and a trailing-comment header
  appended after a real policy table (`[features2] # c`, case 17) all give
  the generic message, matching the plan's self-review-driven revision
  (deviation note, self-review finding #1). CRLF-only (case 11) now gives
  the generic message per the raw-line, no-CR-stripping comparison
  (docstring `scripts/ralph-worktree.sh:94-95`) — this differs from the plan's
  original AC-3 wording (which predates the revision) but matches the
  revised AC-3 text and the deviation notes exactly. Empty HEAD file (case
  18, `:545-556`) is handled via the `FILENAME == "-"` branch (`:123-127`,
  `:119-122` comment) rather than `FNR == NR`, and gets the specific message
  as required. All PASS in the live run.
- **AC-4** (clean base passes; byte identity): PASS. Case 13 (`:440-443`)
  confirms a clean base still passes `validate-clean-base`. `cmp` confirms
  script/recipe byte identity with their templates (table above). `ensure`
  reuses `validate_clean_base` (`:286`), so it is not a separate code path.
- **AC-5** (named cases exist and cover AC-1..AC-4; mutation evidence
  recorded): PASS. 19 named cases (1-19) exist in
  `tests/test-ralph-worktree.sh:296-571` and jointly cover AC-1 (cases 1, 19),
  AC-2 (cases 4-10), AC-3 (cases 2, 3, 11, 14-18), and AC-4 (case 13). The
  plan's 2026-09-30 work (Slice B) deviation note records mutation evidence
  (5 mutation kinds against the pre-Slice-B-narrowing test set, 105/0 total);
  this verify pass independently reran 4 comparable mutations against the
  final code and confirmed each kills the case(s) the plan claims (see
  Deterministic checks above).
- **AC-6** (recipe + evidence doc content): PASS.
  `docs/recipes/codex-setup.md:127-171` ("If `.codex/config.toml` changes on
  its own") has symptom (`:129-140`), check (`:142-146`, `git diff`), restore
  (`:148-153`, checkout + `git status --porcelain` confirmation; `cmp` with
  the template is explicitly scoped to "developing ralph itself" at
  `:155-156`), and an investigation-status paragraph (`:158-169`) that
  states ruled-out candidates with their evidence strength ("neither writes
  to the *project's* `.codex/config.toml`"; "in an isolated probe it writes
  the user-level config regardless of whether the project is marked
  trusted, and never touches the project one") and marks remaining
  candidates as such ("Remaining candidates: ..."). The template copy is
  byte-identical (table above).
  `docs/evidence/codex-config-rewrite-2026-09-30.md` has the investigation
  record (symptom, ruled-out hypotheses with reasoning, `:19-103`), probe
  steps and results including the untrusted-vs-trusted re-run (`:68-95`),
  remaining candidates (`:105-116`), and next-time evidence capture
  including a `stat` command with a timezone offset plus a GNU alternative
  (`:123-129`), `ps` (`:131-132`), and codex logs with an explicit
  instruction not to auto-read the real `~/.codex` (`:134-136`). Neither
  document contains content copied from a real `~/.codex` — both explicitly
  scope every probe to a fake `HOME`/`CODEX_HOME` (recipe `:158` "an isolated
  probe"; evidence doc `:68`, `:70` "偽の HOME / CODEX_HOME").
- **AC-7** (static analysis clean): PASS — see Deterministic checks table.

## Observational checks

- Consistency across the three descriptions of the narrowed rule (function
  doc comment `scripts/ralph-worktree.sh:89-102`, the `die` message text
  `:177`, and the recipe `docs/recipes/codex-setup.md:134-140`): all three
  describe the same shape (HEAD's comment/blank lines droppable, a trailing
  `[shell_environment_policy(.name)?]` table plus its `key = value` lines
  addable, everything else byte-identical) and all three now list the same
  exclusions (changed/added comment, re-indentation, CRLF, non-policy table,
  a comment appended after one). No drift found.
- Grepped `has uncommitted changes` repo-wide (docs `*.md` + `*.sh`): the
  only prose descriptions of the generic message are in the diff itself, the
  plan, the self-review report, and the new recipe section — all consistent
  with the current code. `docs/recipes/worktrees.md` and `README.md`/`AGENTS.md`
  describe "refuse to start if dirty" / "clean-base task worktree" at a
  level that does not name the specific error text, so they are not made
  stale by this change.
- Self-review findings 1-10 (report:
  `docs/reports/self-review-2026-09-30-codex-config-rewrite-detect.md`) were
  cross-checked against Slice B (commit `0bb6aae`) rather than re-reviewed
  for diff quality: #1 (MEDIUM, comment/whitespace/CRLF over-matched) is
  fixed by the raw-line rewrite and covered by cases 11/14-17; #2 (LOW,
  trailing-comment header absorbed as a key line) is fixed and covered by
  case 17; #3 (LOW, `FNR == NR` empty-HEAD bug) is fixed via
  `FILENAME == "-"` and covered by case 18; #4 (LOW, unquoted `$root` in the
  guidance) is fixed via `%q` (`:176`) and covered by case 19; #5 (LOW,
  temp-file cleanup) is fixed by removing the temp file entirely (`git show
  | awk ... -`, `:109`); #6 (LOW, misplaced test comment) — confirmed
  `_cx_fixture` now precedes the `_cx_new_repo` doc comment
  (`tests/test-ralph-worktree.sh:183-209`); #7 (LOW, fixture duplication) —
  confirmed `_cx_write_rewrite_shape` (`:236-263`) and the optional
  `target-dir` parameter on `_cx_new_repo` (`:201-234`) factor out the
  previously-duplicated shapes; #8/#9 (LOW, evidence-doc wording) —
  confirmed the evidence doc now uses observed-range language ("15 回以上で
  再現せず" instead of a blanket "原因ではない", `:99-103`) and states the
  probe's project-trust state explicitly for both runs (`:68`, `:87-89`);
  #10 (LOW, `stat` timezone) — confirmed the evidence doc's next-time
  capture command now includes `-t '%Y-%m-%dT%H:%M:%S%z'` (`:128`).

## Coverage gaps

- The actual external rewrite (as observed 2026-09-17/18) was never
  reproduced in this repo or during this verify pass; correctness of the
  detector rests on the plan's recorded observations of that shape, not on
  a live repro. This is a stated plan non-goal, not a gap in this verify
  pass.
- This pass read `tests/test-ralph-worktree.sh`'s own output once and reran
  4 mutations in scratch copies for AC-5 confidence; it did not otherwise
  execute or author behavioral tests. Full behavioral test ownership
  (including any tests beyond `tests/test-ralph-worktree.sh`) is `/test`'s
  responsibility.
- Did not verify the claim that `everything-claude-code`'s
  `merge-codex-config.js:313` writes only to the user-level config path —
  took the plan/self-review's citation at face value rather than reading
  that plugin's source in this pass (out of scope: third-party plugin code,
  not part of this diff).

## Verdict

- Verified: AC-1, AC-2, AC-3 (revised text), AC-4, AC-5, AC-6, AC-7; all
  static analysis checks (shellcheck, bash/sh syntax, `check-sync.sh`, full
  `run-static-verify.sh`); doc/comment/message consistency; self-review
  finding-by-finding resolution in Slice B.
- Partially verified: AC-5's mutation coverage — reran 4 of the plan's
  claimed mutation kinds independently (all confirmed) rather than every
  mutation variant the implementer may have tried during development.
- Not verified: the detector's fidelity against the real, unreproduced
  external rewrite (plan non-goal); third-party plugin source cited in the
  evidence doc.

**Overall verdict: PASS.** No blocking findings.

## Cycle 2

- Date: 2026-09-30
- Verifier: verifier subagent (Claude), cycle 2 (pipeline cap 2/2)
- Reviewed HEAD: `ae96cd2` (worktree confirmed clean at this SHA before
  starting)
- Scope: `git diff 8e7a678...HEAD` — cycle-1 test (`d84cdab`, cases 20-21),
  cross-review triage AR-1 fix (`ee397e9`, cases 22-24), cycle-2 self-review
  (`f49ceb9`, findings C2-1..C2-5), and the C2-1..C2-5 fix (`a2c721d`, cases
  25-28 plus the "at least one known change" guard and single-quote root
  quoting), plus doc/tech-debt reference repointing to the archive path
  (`ae96cd2`)

### Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `cmp scripts/ralph-worktree.sh templates/base/scripts/ralph-worktree.sh` | PASS | byte-identical |
| `cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md` | PASS | byte-identical |
| `bash -n scripts/ralph-worktree.sh` | PASS | no syntax errors |
| `sh -n tests/test-ralph-worktree.sh` | PASS | no syntax errors |
| `shellcheck -S warning scripts/ralph-worktree.sh` | PASS | 0 warnings |
| `shellcheck -S warning tests/test-ralph-worktree.sh` | PASS | 0 warnings |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS | full scope (issue #190); evidence: `docs/evidence/verify-2026-09-30-095033.log`; all sub-checks `[ok]`/`OK`, golang verifier clean, branch secret scan clean (`5efd3d6..ae96cd2`) |
| `./scripts/check-sync.sh` (standalone) | PASS | `IDENTICAL: 159, DRIFTED: 0, ROOT_ONLY: 0` |
| `sh tests/test-ralph-worktree.sh` (read once for AC evidence, not authored/altered) | PASS | `143 passed, 0 failed, 143 total` |

### Spec compliance delta (AC-1..AC-7 at HEAD)

- **AC-1**: PASS, still holds, with the recovery-command quoting now
  POSIX single-quote form (`scripts/ralph-worktree.sh:189-195`, `sq`/`qroot`)
  instead of bash `%q` — the self-review cycle-2 finding C2-2 fix. Test
  case 19 (space in path) and case 28 (Japanese path component plus a
  literal `'`) both round-trip the printed checkout command through
  `bash -c` (`tests/test-ralph-worktree.sh:590-603`, `:707-726`), and case
  28 additionally round-trips through `zsh -c` when available
  (`:728-744`). `_cx_assert_specific` now computes its expected quoted
  root via `_cx_shquote`, which runs the literal substitution script
  (`:278-295`) rather than assuming an unquoted root — closes self-review
  C2-3. Still no `templates/`/`cmp` mention in the message. All PASS live.
- **AC-2 / AC-3 (revised, further narrowed)**: PASS. New generic-message
  cases added this cycle: a comment injected inside the appended policy
  table (case 20, `:605-629`), an array-of-tables header
  `[[shell_environment_policy]]` (case 21, `:631-651`), an indented
  non-policy header with spaces (case 22) and with a tab (case 23) after
  the policy table (`:653-671`, the cross-review AR-1 fix), an indented
  *policy* header itself (case 24, `:673-680`), a mode-only change with
  byte-identical content (case 25, `:682-689`), a removed trailing newline
  (case 26, `:691-697`), and blank/whitespace-only appended lines (case 27,
  `:699-705`). Cases 25-27 exercise the new "at least one known change"
  guard: the appended-region header check is now `nl ~ /^[ \t]*\[/` so an
  indented header of any kind is recognized as a header and rejected
  unless it is the exact unindented policy form
  (`scripts/ralph-worktree.sh:164-168`), and the function tracks
  `dropped`/`saw_policy_header` and requires at least one to be true before
  printing `MATCH` (`:145`, `:153`, `:159`, `:165`, `:170`) — this is the
  self-review C2-1 fix. All 8 new cases PASS live, and the two mutations
  the plan's cycle-1-test report flagged as uncaught (comment inside the
  appended table; `[[shell_environment_policy]]`) are now covered by cases
  20-21 per the Slice C/cycle-1-test deviation notes.
- **AC-4**: PASS, unchanged in substance; byte identity confirmed above;
  `ensure` still shares `validate_clean_base`.
- **AC-5**: PASS. Case count grew from 19 to 28 across this delta (cases
  20-21 in Slice C/`d84cdab`, 22-24 in Slice D/`ee397e9`, 25-28 in Slice
  E/`a2c721d`), each targeting exactly one of the gaps found by the prior
  test/self-review/cross-review cycle. The plan's deviation notes record
  red evidence for each slice (reverting the header regex breaks
  22/23/24; reverting the "at least one known change" guard breaks
  25/26/27; reverting `%q`→single-quote breaks 1/2/3/18/28; unquoted
  test expectations break 1/2/3/18/28). Did not independently rerun new
  mutations this cycle (cycle-1 verify already reran 4; the delta here is
  additive narrowing on top of an already-mutation-tested core, and the
  self-review cycle 2 report independently probed 13 additional shapes
  — M1-M5, G1-G8 — by hand, which is a stronger independent check than a
  mechanical mutation for this delta).
- **AC-6**: PASS, with the C2-4 wording fix confirmed against the actual
  plugin source (not just re-read as prose): `merge-codex-config.js`
  (`~/.claude/plugins/cache/everything-claude-code/everything-claude-code/1.9.0/scripts/codex/merge-codex-config.js:26-35`)
  defines `TABLE_PATHS` as `features`, `profiles.strict`, `profiles.yolo`,
  `agents(.explorer/.reviewer/.docs_researcher)` — matching the evidence
  doc's revised claim (`docs/evidence/codex-config-rewrite-2026-09-30.md:38-40`)
  — and never includes `shell_environment_policy`; root keys are spliced
  in before the first table and missing table keys are appended
  in-place (`:293-300`, add-only, matching "既存のコメントを剥がすこともない"),
  confirming the doc's "insert/append, not full re-serialize" framing
  (`docs/recipes/codex-setup.md:163-166`). The write target is
  `$CODEX_HOME/config.toml` with `CODEX_HOME` defaulting to `$HOME/.codex`
  (`sync-ecc-to-codex.sh:23-25`), confirming "the user-level config" and
  "a different file from the project's" (recipe `:166`). This was a
  read-only check of the installed plugin's own source, not the user's
  `~/.codex` config; no real `~/.codex` content appears in either doc.
  Recipe and evidence doc symptom text both now list the new negative
  cases (mode-only, trailing newline, blank/whitespace-only appended) —
  `docs/recipes/codex-setup.md:133-143`, evidence doc `:156-159` — and the
  function comment, die message, and both docs describe the same final
  rule (raw lines; HEAD may lose only blank/whole-line-comment lines;
  appended region: blank lines, exact unindented policy headers, key
  lines under them; at least one dropped line or one appended policy
  header required).
- **AC-7**: PASS — see Deterministic checks table above.

### Reference-path check

`docs/evidence/codex-config-rewrite-2026-09-30.md:4`, `:163` and the new
`docs/tech-debt/README.md` row (`grep -n "codex-config-rewrite-detect"
docs/tech-debt/README.md`) now cite
`docs/plans/archive/2026-09-30-codex-config-rewrite-detect.md`, while the
plan file itself is still at `docs/plans/active/2026-09-30-codex-config-rewrite-detect.md`
(confirmed via `ls docs/plans/active/`). This is a deliberate forward
reference per the plan's Slice E deviation note — the path becomes valid
once `/pr` archives the plan — not a broken link. The tech-debt row's cell
count matches the table header (6 pipes, 5 columns, same as the header row
and other existing rows).

### Cross-review triage cross-check

`docs/reports/cross-review-triage-codex-config-rewrite-detect.md` records
one ACTION_REQUIRED finding (AR-1: indented non-policy headers after a
policy table absorbed as key lines) against HEAD `9b68c43`, with the
user's decision to fix and re-run as cycle 2/2. Confirmed AR-1's exact fix
is present: `is_sep_header` itself is unchanged (still requires the exact
unindented form), but the appended-region loop's header test widened from
`/^\[/` to `/^[ \t]*\[/` (`scripts/ralph-worktree.sh:164`), so an indented
header of any kind is now recognized and rejected unless it is the exact
policy form — matching the triage's prescribed fix and covered by cases
22-24.

### Coverage gaps

- Same non-goal gap as cycle 1: the real external rewrite has still never
  been reproduced; this cycle's narrowing (cases 20-28) is validated
  against constructed fixtures and the self-review's hand-probed shapes,
  not a live repro.
- Did not independently rerun mutation testing for this cycle's delta
  (see AC-5 note above) — relied on the plan's recorded red evidence per
  slice plus the self-review cycle 2 report's independent 13-shape probe
  table (M1-M5, G1-G8), which exercises the same code paths a mutation
  would target.
- `gawk` portability of `FILENAME == "-"` remains unconfirmed (noted as a
  gap in the self-review cycle 2 report itself); this repo's CI runs on
  Linux where the default `awk` may or may not be gawk — not independently
  checked in this pass.
- Did not re-verify the self-review cycle 2 report's own claim about
  `core.autocrlf=true` environments (stated there as a safe-side
  miss-in-the-safe-direction, not re-derived here).

### Verdict (Cycle 2)

- Verified: AC-1..AC-4, AC-6, AC-7 at the new HEAD; the cross-review AR-1
  fix; self-review C2-1..C2-5 fixes (cross-checked against code, not
  re-reviewed for diff quality); the archive-path forward-reference is
  intentional, not a doc-drift bug; tech-debt row structure.
- Partially verified: AC-5 — case coverage and the plan's own recorded red
  evidence confirmed; no independent mutation rerun this cycle (see gap
  above).
- Not verified: real-world rewrite fidelity (plan non-goal, unchanged from
  cycle 1); `gawk` portability of the HEAD/working-tree split.

**Overall verdict: PASS.** No blocking findings. This is the pipeline's
2nd and final cycle per `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default 2);
no further fix-and-revalidate re-run is expected after this cycle.
