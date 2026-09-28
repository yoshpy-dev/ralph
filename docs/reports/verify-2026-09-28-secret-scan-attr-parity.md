# Verify report: secret-scan-attr-parity (verify, cycle 1)

- Date: 2026-09-28
- Plan: docs/plans/active/2026-09-27-secret-scan-attr-parity.md
- Verifier: verifier subagent (Claude Code)
- Scope: spec compliance + static analysis only. No behavioral test verdict (the tester runs next), no diff-quality review (self-review already ran: docs/reports/self-review-2026-09-27-secret-scan-attr-parity.md, MEDIUM 3 / LOW 3, all confirmed fixed below).
- HEAD: `aab29be` (`fix/secret-scan-attr-parity`), `git status --porcelain` empty, confirmed at start and end of this run.

## Static analysis

| Check | Result |
| --- | --- |
| `./scripts/run-static-verify.sh` | PASS (changed-language scope: shell). `check-sync.sh` 159 identical / 0 drifted / 11 template-only / 5 known-diff. `check-pipeline-sync.sh` OK. `check-skill-sync.sh` 13 skills in lock-step. `check-template-purity.sh` PASS. `secret-scan-branch.sh` (no-scope run): scanned 03f3e8a..aab29be against origin/main: clean. Evidence: `docs/evidence/verify-2026-09-28-020915.log` |
| `sh -n` | Clean on `scripts/secret-scan.sh`, `scripts/secret-scan-branch.sh`, both `templates/base/scripts/` copies, `tests/test-secret-scan.sh`, `tests/test-secret-scan-branch.sh` |
| `shellcheck --severity=warning` | Zero warnings on both scanner scripts and both test files (exit 0) |
| `cmp` (root vs template) | `scripts/secret-scan.sh`, `scripts/secret-scan-branch.sh`, `.claude/skills/pr/SKILL.md`, `.agents/skills/pr/SKILL.md` all byte-identical. `docs/quality/quality-gates.md` differs only at the pre-existing KNOWN_DIFF line (root cites `docs/tech-debt/README.md` and two CI-check rows the template lacks — same pattern `run-static-verify.sh` already allowlists, not a regression) |

No static-analysis failures. Nothing to fix before proceeding.

## Acceptance criteria

Re-grepped against the code at `aab29be`, not recalled from the plan text.

| AC | Implementing code | Test evidence |
| --- | --- | --- |
| AC-1 (driver `.binary` pinned, incl. `.`/`=` in driver name) | `scripts/secret-scan.sh:298-307` lists `diff\..*\.binary$` keys; `:319-328` pins each with `-c <key>=auto` or, for a key holding `=`, `--config-env=<key>=RALPH_SECRET_SCAN_BINARY_AUTO` (`:325`) | `tests/test-secret-scan.sh:429,432-433,436,440,442` (single driver, two drivers, dot-named driver, `=`-named driver, pre-2.31 git exits 3) |
| AC-2 (`diff.<driver>.algorithm` no-op, confirmed not regressed) | No pin needed — `--diff-algorithm=default` (`secret-scan.sh:340`) already overrides a driver's own algorithm; this AC is test-only per plan | `tests/test-secret-scan.sh:512,535` (moved-token fixture under histogram/patience, both default and driver-scoped) |
| AC-3 (revised: refusal lives in `secret-scan-branch.sh`, not the scanner; scanner does not refuse on `--range`) | `scripts/secret-scan.sh` has no `info/attributes` check (confirmed absent by grep; it was added in slice A 7e0c529 and removed in f2ceeca). `scripts/secret-scan-branch.sh:191-222` (`check_info_attributes`, blank/comment-only lines pass, a rule line calls `cannot_scan`) | `tests/test-secret-scan.sh:553` (range scan with an unrelated info/attributes rule still finds the token) and `:569` (a real `git merge` under the guard with an unrelated rule commits, exit 0). Branch-side: `tests/test-secret-scan-branch.sh:1600-1652` (rule line, comments-only, empty, symlink-to-rules, symlink-to-missing, directory, unreadable) |
| AC-4 (base adds/removes `-diff` after fork; unchanged base skips merge-tree; nested `.gitattributes` counts) | `scripts/secret-scan-branch.sh:462-474` (root+glob pathspec diff, `diff.relative=false` pinned) dispatches to `use_merge_attributes` (`:396-460`) only when changed | `tests/test-secret-scan-branch.sh:1372,1399` (adds/removes both directions), `:1427-1432` (nested + subdirectory cwd), `:1460-1466,1474-1475` (unchanged / already-merged base skips merge-tree, no notice, no `RALPH_SECRET_SCAN_ATTR_SOURCE` inheritance) |
| AC-5 (conflict: HEAD's attributes + one notice, exit follows the scan) | `secret-scan-branch.sh:448-450` (merge-tree rc 1 branch of `use_merge_attributes`) | `tests/test-secret-scan-branch.sh:1503-1506` (strict and default both read HEAD's attributes and find the token; single notice) |
| AC-5b (local `merge.default`/`merge.<name>.driver` config: strict exit 3, default reads HEAD + notice) | `secret-scan-branch.sh:413-426` (`merge_drivers` regex `^merge\.(default|.*\.driver)$`) → `attributes_unguaranteed` | `tests/test-secret-scan-branch.sh:1508-1524`, looped over both `merge.default` and `merge.keep.driver` key shapes (Codex advisory point 2) |
| AC-5c (merge-tree non-1 failure, no `--write-tree`, git < 2.41, `--write-tree`/`GIT_ATTR_SOURCE` unsupported: strict exit 3, default reads HEAD + notice) | `secret-scan-branch.sh:427-459` (`git_version_at_least`, `--write-tree` capability probe, mktemp guard, rc classification) | `tests/test-secret-scan-branch.sh:1541-1573` (`stub_merge_tree_fails`, `stub_old_git`, `stub_no_write_tree`, `stub_merge_mktemp_fails`, plus a real read-only `.git/objects` reproduction) |
| AC-6 (`RALPH_SECRET_SCAN_ATTR_SOURCE` unresolved → exit 3; ignored off-HEAD ranges) | `scripts/secret-scan.sh:288-297` | `tests/test-secret-scan.sh:357-368` |
| AC-7 (info/attributes rule: `--strict` stops, never prints clean) | `secret-scan-branch.sh:203-222` (`cannot_scan` before the scanner is ever invoked) | `tests/test-secret-scan-branch.sh:1600-1607` — see **Spec drift** note below: the substance (strict exit 3, no `clean`, default exits 0) holds, but the literal message the AC-7 text names no longer applies |
| AC-8 (default-config range results unchanged) | No code change to the default (no-driver-config, no-info/attributes) path; `RALPH_SECRET_ALLOWLIST`/env-source logic is additive and gated | Deviation notes report a 220,084-line whole-history comparison matching `main`; self-review re-confirmed by reading rather than re-running (Coverage gaps §). Full regression re-run is `/test` scope, not re-executed here |
| AC-9 (docs match behavior; `scripts/` ≡ `templates/base/scripts/`; `run-verify.sh` green) | Confirmed via `cmp` (byte-identical) and `run-static-verify.sh` PASS above | — |

Non-goals respected: `.github/workflows/`, `scripts/run-verify.sh`, `scripts/prepare-commit-msg-secret-guard.sh`, and `scripts/xreview-helpers.sh` all show no diff against `main` (`git diff main...HEAD` for those paths is empty) — the scanner's CI-facing `--range` path and the hook wiring needed no change, as the plan's Non-goals require. Design decisions section (Codex advisory items 1–3) all traced to code: key-as-a-whole `-c` pinning (not name-extraction), `.gitattributes`-changed gating before `use_merge_attributes` runs, and rc-1-only fallback with every other rc/incapability going through `attributes_unguaranteed`.

### Spec drift: AC-7's literal wording is stale after the self-review M3 revision

AC-7 as written expects `secret-scan-branch.sh --strict` to stop with the message **"scanner failed with exit 3"** when `.git/info/attributes` holds a rule. That was accurate for the pre-self-review design (slice A put the refusal inside `secret-scan.sh`, so `secret-scan-branch.sh` would only see a propagated scanner exit 3). The self-review's M3 finding moved the refusal entirely into `secret-scan-branch.sh`'s own `check_info_attributes` (deviation notes call this out for AC-3 but not AC-7). The branch script now detects the condition itself, before ever calling the scanner, and reports `"cannot scan, .git/info/attributes holds attribute rules that CI does not read; move them to .gitattributes or remove the file"` — never `"scanner failed with exit 3"` for this specific cause. That exact string is still used, but only for a genuine scanner-side failure now (for example an unresolved `RALPH_SECRET_SCAN_ATTR_SOURCE`).

This is documentation/spec drift in the plan's own AC-7 text, not a behavioral gap: the substantive requirement (`--strict` exits 3, never prints `clean`; default mode exits 0 without scanning) is met and tested (`tests/test-secret-scan-branch.sh:1600-1607`). The `.claude/skills/pr/SKILL.md` mirrors already document both messages correctly and separately (line 29 in each of the 4 copies), so no product documentation is affected — only the plan's AC-7 sentence is out of date. Recommend a one-line correction to AC-7 in a future plan touch (not blocking this verify).

## Exit-code contract check

- `scripts/secret-scan.sh:40-48` header vs. code: exit 3 causes (`--file` unreadable, unparsable staged entry, non-commit range end, unresolved `RALPH_SECRET_SCAN_ATTR_SOURCE`, "or git failed") match every `exit 3` call site, including the new `git config --get-regexp` failure path (`:300-307`), which falls under the header's generic "or git failed" clause.
- `scripts/secret-scan-branch.sh:43-54` header vs. code: `cannot_scan`/`nothing_to_scan` (both via `report_and_exit`, strict→3/default→0) cover the base-ref, merge-base, `info/attributes`, and merge-computation-failure cases; any scanner exit besides 0/1 is propagated verbatim and labeled "scanner failed with exit `<rc>`" (`:487-492`), matching the header's closing sentence. No undocumented exit path found.
- No change needed to hook wrappers or `run-verify.sh` (see Non-goals check above) — both already call the scripts by their existing contract (`--range`, `--strict`).

## Documentation drift

- Scanner header (`scripts/secret-scan.sh:1-48`) Environment section documents `RALPH_SECRET_ALLOWLIST` and `RALPH_SECRET_SCAN_ATTR_SOURCE`; the pinned-settings list (top comment) now names `GIT_ATTR_NOSYSTEM` and the `auto` driver-binary pin — matches code.
- Branch header (`scripts/secret-scan-branch.sh:1-66`) documents the attribute-source rule (unchanged base skips merge-tree, changed base uses `merge-tree --write-tree`, conflict/incapability fallbacks, the merge/rebase remedy) and the `info/attributes` refusal — matches code.
- `docs/quality/quality-gates.md` (both copies): updated in the same commit, differ only at the pre-existing KNOWN_DIFF line noted above — not a regression.
- `docs/tech-debt/README.md`: the two rows named in the plan (local-only attribute gaps; CI-merge-commit attribute divergence) are rewritten to reflect what is now pinned/refused vs. what remains (replace refs, pre-2.41 `GIT_ATTR_SOURCE`, local non-driver merge settings such as `merge.renames`). The row citing the merge guard's `info/attributes` behavior correctly says only `secret-scan-branch.sh` refuses now. Two other rows' plan pointers were already repointed from `docs/plans/active/2026-09-25-secret-scan-git-config.md` to `docs/plans/archive/2026-09-25-secret-scan-git-config.md` inside this same diff — the deviation notes flagged this as a sync-docs follow-up, but it is already done.
- `.claude/skills/pr/SKILL.md` Step 3, and its 3 mirrors (`.agents/skills/pr/SKILL.md`, `templates/base/.claude/skills/pr/SKILL.md`, `templates/base/.agents/skills/pr/SKILL.md`): byte-identical (`cmp`/`grep` confirm), all four updated with the `info/attributes` and merge-computation exit-3 causes plus the merge/rebase remedy (L1 fix).
- Beyond the plan's explicit list: AC-7's own wording (see Spec drift above). No other drift found.

## Diff hygiene

`git diff main...HEAD --stat`: 16 files — `scripts/secret-scan{,-branch}.sh` + template mirrors, `tests/test-secret-scan{,-branch}.sh`, `docs/quality/quality-gates.md` + template mirror, `docs/tech-debt/README.md`, `.claude/skills/pr/SKILL.md` + 3 mirrors, plus the plan file, the self-review report, and the plan's insights jsonl. All within the plan's declared Affected areas plus the expected reports/insights. No unrelated files touched.

## Verdict

**PASS.** Static analysis clean. AC-1 through AC-6, AC-9 fully verified against code and tests. AC-5/5b/5c verified against code and a thorough stub/real-failure test matrix. AC-7's behavior is verified and tested; its literal wording is stale (see Spec drift, non-blocking). AC-8 (regression, whole-history parity) is evidenced by the implementer's/self-review's reported comparison but not re-run here — full behavioral confirmation is `/test` scope.

## Follow-ups

1. Update AC-7's wording in a future plan touch to match the M3-revised design (the message is now `secret-scan-branch.sh`'s own `check_info_attributes` reason, not a propagated "scanner failed with exit 3").
2. `/test` should re-run (or otherwise re-confirm) the full-history default-config comparison (AC-8) rather than relying solely on the self-review's prior 220,084-line count.

## Unverified / out of scope for this step

- Behavioral test suite execution (`tests/test-secret-scan.sh`, `tests/test-secret-scan-branch.sh`) — `/test`'s job.
- Diff-quality review — already completed by self-review cycle 1.
