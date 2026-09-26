# Verify report: secret-scan-git-config (cycle 1)

- Date: 2026-09-25
- Plan: docs/plans/active/2026-09-25-secret-scan-git-config.md
- Verifier: verifier subagent (Claude Code)
- HEAD: 95a02f9 (worktree `git status --porcelain` empty at start and end)
- Scope: spec compliance (AC-1..AC-17) + static analysis. No behavioral test verdict (tester runs next), no diff-quality review (self-review already passed).

## Static analysis

| Check | Result |
|---|---|
| `./scripts/run-static-verify.sh` | PASS — full-scope fallback (`secret-scan-branch.sh` has no language classifier extension); gofmt ok, go vet 0 issues, `check-sync.sh` DRIFTED=0, `check-pipeline-sync.sh` ok, `check-skill-sync.sh` ok, `check-template-purity.sh` PASS, branch secret scan clean. Evidence: `docs/evidence/verify-2026-09-25-073053.log` |
| `sh -n` | PASS on all 6: `scripts/secret-scan.sh`, `scripts/secret-scan-branch.sh`, both `templates/base/scripts/` copies, `tests/test-secret-scan.sh`, `tests/test-secret-scan-branch.sh` |
| `shellcheck --severity=warning` | 0 issues on all 4: both scripts, both test files |
| `./scripts/check-sync.sh` | PASS — IDENTICAL 159 / DRIFTED 0 / KNOWN_DIFF 5 (none touch the scanner scripts) |
| `./scripts/check-template-purity.sh` | PASS — no meta-repo-specific references in templates |
| `cmp scripts/secret-scan.sh templates/base/scripts/secret-scan.sh` | byte-identical |
| `cmp scripts/secret-scan-branch.sh templates/base/scripts/secret-scan-branch.sh` | byte-identical |

No static findings.

## Acceptance criteria

| AC | Verdict | Evidence |
|---|---|---|
| AC-1 (color.ui / color.diff) | PASS | `scripts/secret-scan.sh:219` `--no-color` in `scan_range`; `tests/test-secret-scan.sh:163,166` |
| AC-2 (`diff.relative=true`, subdirectory) | PASS | `secret-scan.sh:212` `-c diff.relative=false`; `tests/test-secret-scan.sh:171` |
| AC-3 (committed textconv) | PASS | `secret-scan.sh:216,219` `core.attributesFile=/dev/null` + `--no-textconv`; `tests/test-secret-scan.sh:206` |
| AC-4 (`diff.renames=copies`, rename-only unchanged) | PASS | `secret-scan.sh:213-214` `diff.renames=true` / `diff.renameLimit=1000`; `tests/test-secret-scan.sh:223,228,231` |
| AC-5 (diff algorithm pinned) | PASS | `secret-scan.sh:220` `--diff-algorithm=default`; `tests/test-secret-scan.sh:272,275` (moved-line fixture, not just "pin exists") |
| AC-6 (default-config range results unchanged) | PASS | Ran `sh tests/test-secret-scan.sh` once as team-lead-authorized evidence: 61/61 PASS, including default-config assertions at lines 228, 253, 342. Corroborated by self-review's own 50/50 run pre-Slice-D and the product run below. |
| AC-7 (staged: non-ASCII/space/tab/quote/backslash names) | PASS | `secret-scan.sh:131-171` `scan_staged` reads by blob id, name is label-only; `tests/test-secret-scan.sh:362,369` |
| AC-8 (staged, `diff.relative=true`, subdirectory) | PASS | `secret-scan.sh:134` `-c diff.relative=false`; `tests/test-secret-scan.sh:388` |
| AC-9 (unreadable staged blob fails; gitlink skipped) | PASS | `secret-scan.sh:152-168` (`is_object_id` validation exits 3, `160000` gitlink `continue`, `cat-file` failure exits 3); `tests/test-secret-scan.sh:434,435,441` |
| AC-10 (`--diff` colored input) | PASS | `secret-scan.sh:182` ANSI `gsub` in `scan_diff_stream`; `tests/test-secret-scan.sh:348` |
| AC-11 (symlink target trailing newline) | PASS | `secret-scan-branch.sh:252-266` sentinel-preserving read + `allowlist_target_is_safe` newline rejection; `tests/test-secret-scan-branch.sh:1181-1182` |
| AC-12 (non-ASCII symlink target resolves) | PASS | `secret-scan-branch.sh:206` `core.quotePath=false` on `ls-tree`; `tests/test-secret-scan-branch.sh:1212-1214` |
| AC-13 (`color.ui=always`, end-to-end) | PASS | Relies on AC-1's `scan_range` pin, exercised through `secret-scan-branch.sh`; `tests/test-secret-scan-branch.sh:1238-1239` |
| AC-15 (`git log` failure before/mid output) | PASS | `secret-scan.sh:207-225` writes to a temp file, checks `log_rc` before parsing, exits 3 with a reason; `tests/test-secret-scan.sh:185-191,293-294` |
| AC-16 (`core.bigFileThreshold` small) | PASS | `secret-scan.sh:215` `core.bigFileThreshold=512m`; `tests/test-secret-scan.sh:176` |
| AC-17 (branch script propagates scanner's own exit 3, both modes) | PASS | `secret-scan-branch.sh:304-320` propagates `scan_rc` (not folded into 0/1) regardless of `--strict`; `tests/test-secret-scan-branch.sh:1263-1268` (both strict and default mode assertions) |
| AC-14 (byte-identical templates, `run-verify.sh` green, own `--strict` clean) | PASS | `cmp` above; `run-static-verify.sh` green; `./scripts/secret-scan-branch.sh --strict` on this HEAD → `scanned c1785b8..95a02f9 against origin/main: clean`, exit 0 |

All 17 acceptance criteria map cleanly to implementing code and a labeled pinning test.

## Non-goals check

- No diff to `.gitallowed`, `.github/workflows/`, or any `record_matches` pattern line (`git diff main...HEAD -- .gitallowed .github/workflows/` empty; grepped pattern lines unchanged). Scanner patterns and allowlist content untouched, as the plan requires.
- `-diff`/`binary` attribute handling and merge-commit diffs are explicitly out of scope and not touched.
- Newline-in-filename: the diff does not attempt general newline-in-filename handling; the one place it matters for this PR (the `.gitallowed` symlink target, AC-11) fails closed as designed. Staged-scan paths are read by blob id and used only as a label, so a raw-output line's C-quoting (git always quotes a path containing a literal newline/tab/backslash/double-quote, independent of `core.quotePath`) never causes a parse split.
- Windows: not addressed, matches Non-goals.

## Exit-code contract check

`secret-scan.sh`: 0 (clean) / 1 (found) / 2 (usage) / 3 (could not scan) / 129,130,143 (signals) — consistent between the script's own header (`scripts/secret-scan.sh:18-24`) and its behavior (traps at lines 50-52 exit with the conventional codes; `--file` missing/unreadable exits 3 at line 232-235; `scan_staged`/`scan_range` exit 3 on git or parse failure).

`secret-scan-branch.sh`: header (`scripts/secret-scan-branch.sh:20-29`) now correctly documents that exit 3 has two sources — this script's own `cannot_scan`/`nothing_to_scan` (strict-only) and the scanner's own exit 3 propagated in either mode (lines 304-320). This is internally consistent with AC-17.

Hook wrappers (`pre-commit-secret-guard.sh`, `pre-merge-commit-secret-guard.sh` via `exec`; `prepare-commit-msg-secret-guard.sh`, `commit-msg-guard.sh` under `set -eu` with no conditional wrapping the scanner call) all propagate any non-zero scanner exit — including the new exit-3 cases — as a hook failure that blocks the commit/merge. `scripts/run-verify.sh:207-227` treats any non-zero `secret-scan-branch.sh` exit (including a propagated scanner exit 3) as `status=1`. All four wrapper sites plus `run-verify.sh` were re-read against the new exit-3 semantics; none need code changes — the fail-closed contract already covers the new case because it was written as "any non-zero, not just 1."

## Documentation drift (not fixed here — flagging for `/sync-docs`)

1. **`.claude/skills/pr/SKILL.md` Step 3 (and its 3 mirrors: `.agents/skills/pr/SKILL.md`, `templates/base/.claude/skills/pr/SKILL.md`, `templates/base/.agents/skills/pr/SKILL.md`) undersells what exit 3 can now mean.** The text says "Exit 3 covers two different problems: 'could not scan' ... and 'nothing to scan' ...", both from `secret-scan-branch.sh`'s own logic. As of this PR, exit 3 can also arrive from the underlying scanner failing to read the range (e.g. `git log` failure, an unreadable staged/range blob) and being propagated verbatim by the existing "scanner failed with exit `<rc>`" path (`scripts/secret-scan-branch.sh:307-320`). Before this PR that path was effectively unreachable for exit 3, because `secret-scan.sh` had no exit-3 case at all (confirmed: `git show main:scripts/secret-scan.sh` documents no exit code beyond 0/1/2, and the old header has no exit-code table). It is reachable now, and the operator-facing text doesn't mention it or its distinct stderr line (`secret-scan-branch: scanner failed with exit <rc> on <range>` vs. the `cannot scan, ...` / `nothing to scan, ...` phrasing Step 3 describes).
2. **`docs/tech-debt/README.md` has no new rows yet** for the plan's own Non-goals items (committed `-diff`/`binary` handling, merge-commit diffs) or for self-review LOW-5 (the `++ ` prefix in an added line, deferred per the plan's Deviation notes). This is expected — the plan's Implementation outline assigns this to `/sync-docs` (point 4) — not a defect, just noting it's still pending.
3. **Watch for a stale tech-debt candidate**: the plan's work-slice Deviation notes (line 124 of the plan) list `diff.renameLimit`(未確認) among the out-of-scope items to record as tech-debt. That item was subsequently reproduced and *fixed* by self-review's MEDIUM-1 (Slice D, commit `aecee05`: `-c diff.renameLimit=1000` is now pinned, `tests/test-secret-scan.sh` covers it). `/sync-docs` should not carry it forward as an open tech-debt row — the plan's later self-review section already supersedes that earlier deviation note.
4. **Pre-existing, out of scope**: `docs/architecture/repo-map.md`'s scripts bullet lists `secret-scan.sh` under "secret and commit safety" but not `secret-scan-branch.sh` (unlike `AGENTS.md`, which lists both). `repo-map.md` was last touched by an unrelated commit (`1bc74cc`) and is untouched by this PR's diff — flagging only as a pre-existing gap, not a regression from this change.

`docs/quality/quality-gates.md` (both the root and `templates/base/` copies) describe `--range` / `--strict` modes only, with no exit-code table — no drift found there.

## Diff hygiene

`git diff main...HEAD --stat`: 9 files, 1196(+)/85(-) — `scripts/secret-scan.sh`, `scripts/secret-scan-branch.sh`, both `templates/base/scripts/` copies, `tests/test-secret-scan.sh`, `tests/test-secret-scan-branch.sh`, the plan, the self-review report, and one insight-events jsonl. All within the plan's affected areas plus reports/insights.

## Verdict: PASS

No static findings, no failed acceptance criteria, no Non-goals violations. Three documentation-drift items are flagged above for `/sync-docs` (none block this verdict).

## Follow-ups

- `/sync-docs`: fix the `pr/SKILL.md` Step 3 exit-3 description (×4 mirrors), add the two Non-goals tech-debt rows, and check whether the `diff.renameLimit` deviation-note item should be dropped or marked resolved rather than filed as new debt.
- `/test`: run the full behavioral suites (`tests/test-secret-scan.sh`, `tests/test-secret-scan-branch.sh`, plus `tests/test-run-verify-branch-secret-scan.sh` per the plan's Test plan) and confirm the mutation-based red/green claims from the self-review still hold.

## Insight event

Appended: `docs/insights/events/2026-09-25-secret-scan-git-config.jsonl` (phase=verify, cycle=1, verdict=pass, critical=0, high=0, medium=0, low=0).

## Cycle 2

- Date: 2026-09-25
- HEAD: 5774b5f (worktree `git status --porcelain` empty at start and end)
- Delta since cycle 1 (b76d9da): `237dc14` test report, `f6d1b87` sync-docs, `f413a58` cycle-1 cross-review triage (AR-1), `ca8a212` fix AR-1 (state-aware `+++ ` header rule), `45ce71f` plan notes, `750d247` cycle-2 self-review, `443ffb7` Slice F (`GIT_ATTR_SOURCE=HEAD` + doc corrections), `5774b5f` plan notes

### Static analysis (re-run)

| Check | Result |
|---|---|
| `./scripts/run-static-verify.sh` | PASS — full-scope fallback (unchanged reason); gofmt ok, go vet 0 issues, `check-sync.sh` DRIFTED=0, `check-pipeline-sync.sh` ok, `check-skill-sync.sh` ok, `check-template-purity.sh` PASS, branch secret scan clean (`scanned c1785b8..5774b5f against origin/main: clean`). Evidence: `docs/evidence/verify-2026-09-25-133512.log` |
| `sh -n` | PASS on all 6 files (both scripts, both template copies, both test files) |
| `shellcheck --severity=warning` | 0 issues on both scripts + both test files |
| `cmp` templates | `scripts/secret-scan.sh` and `scripts/secret-scan-branch.sh` still byte-identical to their `templates/base/scripts/` copies |

No new static findings. `scripts/secret-scan-branch.sh` is unchanged in this delta (confirmed via `git diff b76d9da..5774b5f --stat` — only `scripts/secret-scan.sh` + template copy touched among the scanners), so its own static checks carry over unchanged from cycle 1.

### AR-1 fix (cross-review)

Cross-review triage (`docs/reports/cross-review-triage-secret-scan-git-config.md`) found AR-1: the ANSI-strip added by this PR ran a global `gsub` before the `/^\+\+\+ /` file-header rule, so a *committed* line whose content is an escape sequence followed by `++ ` (e.g. `ESC[32m++ <token>`) collapsed to `+++ ...` after stripping and was silently dropped as a header — a regression against `main`, which detects that same line (main has no ANSI-stripping at all, so `+ESC[...]++ ` never literally starts with `+++ `). This is a different, narrower case than self-review cycle-1's LOW-5 (plain, uncolored content starting with `++ `, already present pre-PR and deferred as CI-shared tech debt).

Fix `ca8a212` (`scripts/secret-scan.sh:185-198`) replaces the global rule with a state machine: `in_hunk` is reset to 0 on a `diff ` line and set to 1 on a `@@` line; `+++ ` is skipped only when `!in_hunk`; the ANSI strip is now anchored (`while (sub(/^\033\[[0-9;:]*m/, "")) {}`, line 191) so it only removes *leading* escape codes, never ones embedded in content. This fixes AR-1 and, as a side effect proven correct by the self-review's oracle comparison (a Python hunk-counter run against 215,825 lines of this repo's own `git log -p` history plus a synthetic fixture covering every header-look-alike shape), also closes LOW-5 — it is "a strict superset of main's reads: the only lines it adds are hunk content that starts with `++ `" (self-review, Positive notes).

Evidence re-checked at HEAD:
- `tests/test-secret-scan.sh:406-407` (escaped `++ ` content, `--range`), `:412` (plain `++ ` content, `--range`), `:423` (token-looking file name in a `+++ ` header not reported), `:427,429,431,433,435` (the same four shapes plus a colored hunk, under `--diff`).
- Ran `sh tests/test-secret-scan.sh` once at HEAD (same narrow AC-6-evidence carve-out as cycle 1): 74/74 PASS (up from 61/61 in cycle 1, consistent with the ~13 new assertions for AR-1 + `GIT_ATTR_SOURCE`).
- Cross-review's own full-history oracle run (215,825 lines, byte-identical across main's awk, the new awk, and the oracle) is stronger evidence than any single fixture that the new rule reads a strict superset of `main`'s lines with no unrelated behavior change.

### AC re-map (current code, cycle-2 line numbers)

AC-1, AC-2, AC-3, AC-4, AC-5, AC-7, AC-8, AC-9, AC-11, AC-12, AC-13, AC-15, AC-16, AC-17 — unchanged in substance from cycle 1; re-grepped and still PASS at the current line numbers (`scan_staged`/`scan_range` pin block now at `scripts/secret-scan.sh:203-236` after the `GIT_ATTR_SOURCE=HEAD` comment insertion; `is_object_id` still `:113-120`; `scan_staged` still `:132-172`). Only the two ACs below have cycle-2-specific behavior to re-verify:

- **AC-6 (default-config range results unchanged)**: still PASS, with one intentional, plan-acknowledged addition. The state-aware header rule (above) now scans hunk lines whose content starts with `++ ` — previously dropped, including under the default config — as findings. This is a widening in the "finds more" direction only (a strict superset per the oracle comparison), and it is the explicit purpose of fixing AR-1/LOW-5, not a spec violation: the plan's Design decisions and both self-review cycles treat "CI and local agree" as the contract, and a line CI's own `git log -p` already contains was previously under-scanned locally in the same way it was under-scanned by `main`. `tests/test-secret-scan.sh:228,274,318,364` (unrelated default-config assertions carried over from cycle 1, still passing) plus the new `:407,412,423` assertions confirm the widened set is exactly the `++ `-prefixed hunk-content case, nothing broader.
- **AC-16 / range scan config-independence**: unchanged (`core.bigFileThreshold=512m` still pinned, `scripts/secret-scan.sh:211`); no interaction with this cycle's fixes.

**New in cycle 2, not covered by cycle-1's AC list**: `GIT_ATTR_SOURCE=HEAD` (Slice F, `443ffb7`) closes a gap the plan's own Deviation notes had flagged as "unconfirmed" (an uncommitted working-tree `.gitattributes` edit, or a local `attr.tree`, marking a file `-diff` so the range scan misses it while CI's checkout of HEAD would not). `scripts/secret-scan.sh:226` sets `GIT_ATTR_SOURCE=HEAD` on the `scan_range` git invocation (git 2.41+; ignored by an older git, which still reads the working tree — documented at `:203-206` and in the header at `:8`). This was not one of the plan's original 17 ACs (it is a self-review cycle-2 fix, not a plan acceptance criterion), so it does not get an AC number, but it is fully tested: `tests/test-secret-scan.sh:184-236` (version-gated on git ≥ 2.41 via `git_version`/`git_major`/`git_minor` parsing at `:185-195`, with an explicit `SKIP:` print on an older git) covers an uncommitted `-diff` edit, a local `attr.tree`, a committed `.gitattributes` read correctly from HEAD even with the working-tree copy removed (matching CI's checkout), and an unborn-HEAD edge case correctly exiting 3. Cross-review verified the git-version claim directly: "git 2.32.7 ... skips inexact rename detection ... and git 2.34.8 ... detects all 500," and separately confirmed `GIT_ATTR_SOURCE`'s effect on git 2.49; neither self-review nor cross-review checked which git release *introduced* `GIT_ATTR_SOURCE` itself (recorded as a coverage gap in the self-review, not contradicted by anything found here).

### Non-goals (re-check)

Still respected: `git diff b76d9da..5774b5f -- .gitallowed .github/workflows/` is empty (not re-touched in this delta either), and the `record_matches` pattern lines are unchanged in the cycle-2 diff. The AR-1 fix and `GIT_ATTR_SOURCE=HEAD` are both scanner-mechanism changes, not pattern or allowlist changes.

### Documentation drift (re-check against cycle-1's flagged items)

All three cycle-1 items are resolved:

1. **`pr/SKILL.md` exit-3 description (×4 mirrors)** — now reads "Exit 3 covers three different problems: ... and the scanner itself failing to read the range (printed as `scanner failed with exit 3`; for example a git error)" (`.claude/skills/pr/SKILL.md:29`, and identically in `.agents/skills/pr/SKILL.md`, `templates/base/.claude/skills/pr/SKILL.md`, `templates/base/.agents/skills/pr/SKILL.md`). Matches the actual passthrough behavior confirmed in cycle 1.
2. **`docs/tech-debt/README.md` Non-goals rows** — now present: line 138 (CI-shared blind spots: committed `-diff`/binary attribute or a real binary file, and merge commits — no longer lists the `++ ` item, confirmed absent) and line 139 (local-only gaps: exactly two unpinnable — `.git/info/attributes` `-diff` and a local `diff.<driver>.binary=true` for a committed-`.gitattributes`-named driver — plus the one closable-but-not-yet-done item, a local replace ref on `secret-scan-branch.sh`'s own git calls, closable with `GIT_NO_REPLACE_OBJECTS=1`; the row also correctly notes the uncommitted-`.gitattributes` case is now pinned by `GIT_ATTR_SOURCE=HEAD` rather than counting it among the two unpinnable gaps).
3. **Stale `diff.renameLimit` deviation-note watch item** — resolved as anticipated: `docs/tech-debt/README.md` carries no open row for it (MEDIUM-1 fixed it in cycle 1, before cycle 1's `/sync-docs` even ran), so there was nothing for `/sync-docs` to file.

New drift, found in cycle-2 self-review and fixed by Slice F (spot-checked at current HEAD, all confirmed fixed, no re-open):

4. `docs/quality/quality-gates.md:46` (both root and `templates/base/` copies) previously overclaimed "regardless of local git config"; now reads "except for a few local-only attribute sources it cannot pin (listed in the `scripts/secret-scan.sh` header ...)" — matches the header's own "Known local-only gaps" list (`scripts/secret-scan.sh:10-13`). Confirmed no remaining claim anywhere in the diff that the scan is independent of *all* local git settings.
5. `docs/architecture/repo-map.md:61` now lists `secret-scan-branch.sh` alongside `secret-scan.sh` under "secret and commit safety" — closes the pre-existing gap cycle 1 flagged as out-of-scope.
6. `docs/tech-debt/README.md` row 140 (test-gaps) now attributes the untested `--no-show-signature` observation to "the cycle-1 self-review" rather than the implementer-facing "self-review Slice D" (Slice D is an implementer slice, not a self-review) — matches self-review cycle-2 finding (c).

No remaining open documentation-drift items found in this scope.

### Diff hygiene (cycle 2)

`git diff b76d9da..5774b5f --stat`: 17 files touched, all within plan/report/tech-debt/doc-mirror scope plus the two scanner-mechanism files and their tests (listed above). No unexpected files.

### Verdict: PASS

No static findings, no failed acceptance criteria (all 17 plan ACs plus the un-numbered `GIT_ATTR_SOURCE=HEAD` fix), no Non-goals violations, and all documentation drift from cycle 1 is resolved with no new open items.

### Follow-ups

- `/test`: `docs/reports/test-2026-09-25-secret-scan-git-config.md` on file is a **cycle-1** report (HEAD `b76d9da`, 61/61) that predates `ca8a212` (AR-1 fix) and `443ffb7` (`GIT_ATTR_SOURCE=HEAD`) — it does not cover either fix. `/test` needs a genuine cycle-2 pass at `5774b5f`: full behavioral suites (now 74/74 for `tests/test-secret-scan.sh` per the AC-6-evidence run above), plus the cycle-2 self-review's mutation-based red/green claims (state-machine mutations, the `while`-loop-to-single-`sub` non-discriminating mutation already disclosed as a coverage gap).
- No verifier-level follow-ups remain; ready for `/cross-review` (cycle 2) → `/pr` per the pipeline once `/test` confirms.

### Insight event (cycle 2)

Appended: `docs/insights/events/2026-09-25-secret-scan-git-config.jsonl` (phase=verify, cycle=2, verdict=pass, critical=0, high=0, medium=0, low=0).

## Cycle 3

- Date: 2026-09-26
- HEAD: c2ad097 (worktree `git status --porcelain` empty at start and end)
- This is run 3 of 3 (final; the user raised the pipeline cap to 3 after cross-review cycle 2 found AR-2)
- Delta since cycle-2 verify (092c5e3): `b403b9f` cycle-2 test report, `dca6c2f` cycle-2 sync-docs, `df3193e` cycle-2 cross-review triage (AR-2, cap raised to 3), `004d99e` Slice G (attributes from the range end — this is what introduced C3-H1, see below), `63461d6` plan notes, `9e297e4` self-review run 3 (HIGH C3-H1, LOW ×3), `0e136fc` Slice H (the fix for C3-H1 and the three LOWs), `c2ad097` plan notes

### Static analysis (re-run)

| Check | Result |
|---|---|
| `./scripts/run-static-verify.sh` | PASS — full-scope fallback (unchanged reason); gofmt ok, go vet 0 issues, `check-sync.sh` DRIFTED=0, `check-pipeline-sync.sh` ok, `check-skill-sync.sh` ok, `check-template-purity.sh` PASS, branch secret scan clean (`scanned c1785b8..c2ad097 against origin/main: clean`). Evidence: `docs/evidence/verify-2026-09-26-100534.log` |
| `sh -n` | PASS on all 6 files |
| `shellcheck --severity=warning` | 0 issues on both scripts + both test files |
| `cmp` templates | both scanners still byte-identical to their `templates/base/scripts/` copies |
| `sh tests/test-secret-scan.sh` (AC-6/CI-parity evidence, same carve-out as prior cycles) | 86/86 PASS at HEAD (up from 74/74 in cycle 2, matching the plan's own count) |
| `./scripts/secret-scan-branch.sh --strict` (AC-14 evidence) | `scanned c1785b8..c2ad097 against origin/main: clean`, exit 0 |

No new static findings.

### The AR-2 -> C3-H1 -> Slice H arc

This cycle's core mechanism change is in three steps, all now landed:

1. **AR-2 (cross-review cycle 2, found against 443ffb7's `GIT_ATTR_SOURCE=HEAD`)**: `prepare-commit-msg-secret-guard.sh` calls `secret-scan.sh --range "HEAD..$merge_head"` during a merge — a range that does *not* end at HEAD. Cycle 2's unconditional `GIT_ATTR_SOURCE=HEAD` forced *every* range, including this one, to read HEAD's (pre-merge) attributes. If HEAD carries a `-diff` that the incoming side had already dropped, and the incoming side adds-then-deletes a secret, the merge guard now misses it — a genuine regression against `main` (which read the working tree's live attributes, i.e. the in-progress merge result, and caught it).
2. **Slice G (`004d99e`)** tried "attribute source = the range's own end commit" (HEAD for CI/branch-scan, `$merge_head` for the guard). Self-review run 3 (`9e297e4`) found this only **moves** the misread, doesn't remove it (**C3-H1**, HIGH): the guard's `$merge_head` is the *incoming* side's own tree, not the actual merge result. Constructed the mirror case — HEAD's side drops `-diff` (forked from a base that had it), the incoming side keeps it and adds-then-deletes a token — and confirmed Slice G's guard misses it while `main`'s guard and the post-merge/CI scan both catch it.
3. **Slice H (`0e136fc`, current HEAD)** implements C3-H1's own recommendation exactly: pin `GIT_ATTR_SOURCE` to HEAD's commit *only when the range's resolved end equals HEAD*; for any other end (the merge guard), explicitly `unset GIT_ATTR_SOURCE` (including one inherited from the caller's environment) and add `-c attr.tree=` so a local `attr.tree` config doesn't step in either — falling back to the working tree's attributes exactly as `main` did. Verified in code: `scripts/secret-scan.sh:240-277` (`end_commit`/`head_commit` comparison, lines 253-258 are the branch).

Evidence re-checked at HEAD:
- `tests/test-secret-scan.sh` new section at `:239-321` (the `expect_merge_guard_finds` helper plus four merge scenarios): "merge where the incoming side drops -diff" (the original AR-2 shape) and "merge where HEAD's side drops -diff" (the C3-H1 mirror) both run a *real* `git merge --no-ff --no-commit` and invoke the *real* `prepare-commit-msg-secret-guard.sh` (not just `--range` directly), confirming the actual caller is fixed, not just the library function. Also: "an inherited GIT_ATTR_SOURCE does not choose the merge guard's attributes" and "a local attr.tree does not replace the merge guard's working-tree attributes" (`:305-309`).
- The two `A...B` tests (`:255-274`, "A...B with B other than HEAD reads the working tree's attributes" / "A...B with B at HEAD reads HEAD's attributes") replace the earlier "a symmetric A...B range reads attributes from B" test that C3-L1 asked to be dropped or rewritten — they now pin the exact, disclosed HEAD-only-pin behavior instead of a blanket per-end-tree pin.
- Ran `sh tests/test-secret-scan.sh` once at HEAD (same AC-6-evidence carve-out as prior cycles): 86/86 PASS.

I did not independently re-run the merge-fixture reproduction by hand — it has already been reproduced three times by three independent parties (cross-review's own fixture for AR-2, self-review run 3's own mirror fixture for C3-H1, and now a hermetic, committed regression test for both directions plus the real guard script), which is stronger evidence than a fourth manual repro would add.

### AC re-map (current code, cycle-3 line numbers)

All 17 plan ACs are unaffected in substance by this cycle's fix (the attribute-source logic only changes *which git config governs .gitattributes*, not which added lines a range's diff contains) and still PASS at re-grepped line numbers: `is_object_id` `:120-125`, `scan_staged` `:137-177`, `scan_diff_stream` `:190-203`, `scan_range`'s pin block `:240-277`. AC-13 (color.ui=always end-to-end via `secret-scan-branch.sh`) is unaffected — that script's range still ends at HEAD, so it still takes the `GIT_ATTR_SOURCE=$head_commit` branch, same as cycle 2.

New in this cycle, not part of the plan's original 17 ACs (same status as cycle 2's `GIT_ATTR_SOURCE=HEAD` — a self-review/cross-review-driven fix, not a plan acceptance criterion):
- **Merge-guard attribute correctness** (AR-2/C3-H1, above): fixed and tested in both directions.
- **Range-end resolution and its own exit-3 case** (`scripts/secret-scan.sh:242-247`): `range_end=${range##*..}` (last `..`, matching the header's documented forms `A..B`, `A...B`, `A..`, `..B`, or a single revision), defaulting to `HEAD` when empty, then `git rev-parse --verify --quiet "$range_end^{commit}"` — a form that doesn't resolve to a commit (`X^!`, `X^@`, `X^-`, a `:/regex` end) exits 3 with "`<end> does not name a commit`" rather than being guessed. This is a *new* exit-3 trigger, additive to AC-15's existing "git log failed" case (it fires *before* `git log` even runs), and it's covered by `tests/test-secret-scan.sh:217-219` ("a range whose end does not resolve exits 3"). Per cross-review's own forms probe (`docs/reports/cross-review-triage-secret-scan-git-config.md`, "Range forms" table), no real caller in the repo passes an unsupported form — verify.yml, `secret-scan-branch.sh`, and `prepare-commit-msg-secret-guard.sh` all pass plain `A..B`.
- **`A...B` accuracy is a disclosed, tested, narrow residual limitation, not a bug**: for a triple-dot range whose end happens to equal HEAD, attributes are still pinned to HEAD even though a symmetric-difference scan also prints commits reachable only from the *other* side — those commits' own diffs get HEAD's attributes rather than their own tree's. No caller in the repo uses `A...B`; both directions are pinned by name in the two new tests above rather than left as silent behavior.

### AC-6 / CI parity (re-confirmed for this cycle's specific claim)

Team lead's framing for this cycle — "verify.yml's `$base..HEAD` and the branch scan's `merge-base..HEAD` (both end at HEAD)" — holds: both ranges resolve `range_end` to `HEAD` literally, so both take the `end_commit = head_commit` branch and get `GIT_ATTR_SOURCE=$head_commit`, identical to cycle 2's intent for these two callers. Only `prepare-commit-msg-secret-guard.sh`'s range does not end at HEAD, and that is exactly the caller Slice H's `else` branch targets. `sh tests/test-secret-scan.sh` at 86/86 (above) includes the full set of default-config and attribute-source assertions carried over from cycles 1-2 plus this cycle's additions, none regressed.

### Exit-code contract (re-check)

`scripts/secret-scan.sh`'s header (`:23-29`) now lists four exit-3 causes: a missing/unreadable `--file`, an unparsable staged entry, **a `--range` end that is not a commit** (new this cycle), or `git` failing. All four are still just "any non-zero, not 1" to every caller (the four hook wrappers, `secret-scan-branch.sh`, `run-verify.sh`'s branch-scan call) — none of those call sites special-case by exit value beyond the existing 0/1/other split, so the new exit-3 trigger needed no caller-side changes, same conclusion as cycle 1/2 for the earlier exit-3 additions. Confirmed no caller file was touched in this delta (`git diff 092c5e3..c2ad097 --stat` lists only `scripts/secret-scan.sh` + template copy among scanner-adjacent scripts).

### Non-goals (re-check)

Still respected: `git diff 092c5e3..c2ad097 -- .gitallowed .github/workflows/` is empty, and `record_matches` pattern lines are unchanged in this delta too.

### Documentation drift (re-check)

All items from self-review run 3 (C3-H1's own doc angle, C3-L1, C3-L2, C3-L3) are fixed by Slice H, confirmed at current HEAD:

1. **C3-H1 doc angle** — the `scan_range` comment's "incoming side's" framing (which stated the rationale that C3-H1 disproved) is gone; the current comment (`scripts/secret-scan.sh:229-236`) correctly says a range not ending at HEAD "reads the working tree's attributes, which during a merge are the merge result's, the tree CI reads once the merge is pushed" — matching Slice H's actual (corrected) behavior, not Slice G's.
2. **C3-L1 (range-form documentation)** — usage text now states supported forms inline (`:39`: "A..B, A...B, A.., ..B, or one revision"), and the `scan_range` comment (`:229-236`) states the resolution rule and names the unsupported forms explicitly. The `A...B` residual (above) is now disclosed via the two dedicated tests rather than left implicit.
3. **C3-L2 (header wording)** — both sentences flagged are corrected: attributes are now described as pinned "for a range ending at HEAD" (not "to git's defaults", `:8-10`), and the git-version-gating sentence now correctly singles out `GIT_ATTR_SOURCE` as needing git 2.41 separately from the `-c` keys (`:18-21`), matching `docs/tech-debt/README.md`'s test-gaps row.
4. **C3-L3 (CI's PR-merge-commit attribute divergence)** — now has its own dedicated tech-debt row (`docs/tech-debt/README.md`, the new row added in this delta, right after the "Local-only ways" row) and `docs/quality/quality-gates.md` (both root and `templates/base/` copies, line 46) now says "except for a few attribute differences it cannot pin, local-only attribute sources and CI's checkout of the PR merge commit" — broadened from cycle 2's "local-only attribute sources" alone, per C3-L3's exact request.

`pr/SKILL.md` (×4) and `docs/architecture/repo-map.md` are untouched in this delta (not in `git diff 092c5e3..c2ad097 --stat`) — consistent with self-review run 3's own "Status of the cycle-2 findings at HEAD" section, which found no new issues with either.

No open documentation-drift items found in this scope.

### Diff hygiene (cycle 3)

`git diff 092c5e3..c2ad097 --stat`: 13 files, all within plan/report/tech-debt/doc scope plus `scripts/secret-scan.sh` + template copy + `tests/test-secret-scan.sh`. No unexpected files. `secret-scan-branch.sh` and its test file remain untouched across all three cycles of this plan.

### Verdict: PASS

No static findings, no failed acceptance criteria, no Non-goals violations. The HIGH finding from self-review run 3 (C3-H1) is fixed and tested in both directions (not deferred), and all three LOWs are fixed. No documentation drift remains open.

### Follow-ups

- `/test`: needs a genuine cycle-3 pass at `c2ad097` (same caution as cycle 2 — `docs/reports/test-2026-09-25-secret-scan-git-config.md`'s cycle-2 section is itself only current through `b403b9f`/`5774b5f`-era HEAD, predating Slices G and H). In particular, confirm the merge-guard mutation set (dropping the `end_commit = head_commit` branch, dropping the `unset GIT_ATTR_SOURCE`/`-c attr.tree=` else-branch pins) each turns red the corresponding new test.
- This is the final planned pipeline run (cap 3, user-approved) — no further verifier-level follow-ups; ready for `/cross-review` (cycle 3) → `/pr` once `/test` confirms.

### Insight event (cycle 3)

Appended: `docs/insights/events/2026-09-26-secret-scan-git-config.jsonl` (phase=verify, cycle=3, verdict=pass, critical=0, high=0, medium=0, low=0).
