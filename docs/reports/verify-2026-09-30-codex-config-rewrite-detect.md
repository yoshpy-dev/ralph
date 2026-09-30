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
