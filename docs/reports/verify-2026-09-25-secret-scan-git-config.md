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
