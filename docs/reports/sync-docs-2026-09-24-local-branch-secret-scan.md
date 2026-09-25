# Sync-docs report: local-branch-secret-scan

- Date: 2026-09-24
- Plan: `docs/plans/active/2026-09-20-local-branch-secret-scan.md` (issue #169)
- Agent: `doc-maintainer` subagent (Claude Code), pipeline cycle 1
- Scope: documentation sync only. Reference reports: `docs/reports/self-review-2026-09-21-local-branch-secret-scan.md` (12 findings, fixed in `3afac6e`), `docs/reports/verify-2026-09-22-local-branch-secret-scan.md` (pass), `docs/reports/test-2026-09-24-local-branch-secret-scan.md` (pass, two gaps named).
- Precondition confirmed: HEAD `9554a58` (docs: add test report for local-branch-secret-scan), `git status --porcelain` empty at start.

## Surfaces checked, no change needed

Every surface below was read and compared against the code at HEAD; none was stale.

- `README.md` — the "Deterministic hooks" table row's "commit-msg secret scan" names `commit-msg-guard.sh`, a per-commit git hook, a different mechanism from this branch's history-range scan; the `scripts/` tree summary already ends in "etc." and does not enumerate scripts elsewhere either.
- `templates/base/README.md` — does not exist. `README.md` documents the ralph meta-repo itself and is not part of what `ralph init` scaffolds (confirmed: no `templates/base/README.md`, and no scaffold-manifest reference to shipping one).
- `docs/quality/definition-of-done.md` — no mention of secret scanning at all (neither the pre-existing CI scan nor the new local one); this is a generic per-phase checklist that was already this abstract before this branch, not a regression.
- `.claude/rules/ralph/git-commit-strategy.md` — the "Enforcement" sentence under Safe Quoting names `pre_bash_guard.sh` and `commit-msg-guard.sh`, both per-commit guards for a different concern (safe quoting / leaked secrets in the commit message itself), not the branch-history range scan this plan adds. Still accurate and complete for what it describes.
- `.claude/rules/ralph/ralph-workflow.md` — "Run `./scripts/run-verify.sh` ... before claiming success" is generic by design; the branch scan is now included transparently as one of `run-verify.sh`'s steps, which is the whole point of wiring it there instead of updating every caller.
- `.claude/skills/work/SKILL.md` (steps referencing `run-verify.sh`, Completion gate) — same generic-reference reasoning; no enumeration of what `run-verify.sh` checks exists there to go stale.
- `docs/recipes/` — only `adding-a-language-pack.md` matches `secret`, and that hit is its own unrelated "secret-bearing files" `.gitignore`-block note, not in scope for this diff.
- `.ralph/core/AGENTS.core.md` — the scaffolded-`AGENTS.md` managed-block source. It has no `scripts/` enumeration or repo-map section at all (56 lines, Mission/Primary loop/Source of truth/Verification/Hard rules only); the detailed `scripts/` bullet with the new `secret-scan-branch.sh` mention lives in the meta-repo's own `AGENTS.md` "Repo map" section (a meta-repo addition outside the managed block), which was already updated per the plan's Deviation notes. Nothing to add to the managed-block source.
- `docs/quality/quality-gates.md` + template copy — both already carry the branch-scan bullet, identical in wording (root omits `(issue #169)`, matching the self-review LOW fix already applied), confirmed via `diff`.
- `.claude/skills/pr/SKILL.md` + 3 mirrors — Step 3 (strict scan before push), Step 8 (re-scan before the archive-commit push), and the Completion gate all read correctly against `scripts/secret-scan-branch.sh`'s actual exit-code contract at HEAD. `cmp` confirms byte-identity across all four faces.
- `AGENTS.md` `scripts/` bullet — already names `secret-scan-branch.sh` ("branch-history secret scan `secret-scan-branch.sh`").

## `.gitallowed` guidance (task item 2)

No standalone doc explains how to write an allowlist regex; the documentation is intentionally the tool itself, and it is now internally consistent:

- `scripts/secret-scan.sh`'s on-finding guidance text (added in this plan) tells the reader to write the key name as a bracket expression (e.g. `api_ke[y]`) so the allowlist line does not match a scanner pattern, and explains why (a branch-history scan also reads the commit that adds the allowlist line).
- `.gitallowed`'s own header comments already document the same bracket-expression convention with worked examples, and were written before this plan.

Both agree. No new doc was created — consistent with the plan's own framing that the scanner's finding-time output is the documentation.

## `docs/tech-debt/README.md` (task item 3)

Added one new row for (a): `scripts/check-template.sh`'s `required_files` list has no regression test — confirmed via the test report's mutation 6f (dropping `scripts/xreview-helpers.sh` from the list caused zero test failures anywhere in the suite, and `check-template.sh` itself still reported success). Framed as pre-existing and not introduced by this plan, per the test report's own judgment.

(b) — the symlinked-`.gitallowed` cosmetic notice quirk — was **not** added as a register row, per the handoff's own default: it is cosmetic, not security-relevant (scan results stay correct), and not tied to a named acceptance criterion. It is already recorded in the test report's "Edge cases explored" and "Test gaps" sections, which is sufficient traceability for something this narrow (nobody has a legitimate reason to symlink `.gitallowed`).

## Plan readability (task item 4)

Scope row 1 (`docs/plans/active/2026-09-20-local-branch-secret-scan.md`) now opens with a "現在の契約(要約)" (current contract, summarized) sentence stating the final exit-code rules in one place, before the "Codex advisory" / "self-review cycle 1" revision history that follows it. The revision history was left intact — nothing was deleted, and no AC text was touched. Progress checklist: ticked "Verification artifact created" and "Test artifact created" (both now exist and pass). Added one Deviation notes bullet each for the `/verify` pass, the `/test` pass, and this `/sync-docs` pass, so the plan's own history stays complete without requiring a reader to cross-reference the reports directory to know what happened after the self-review cycle.

## Insight events (task item 5)

Three per-day event files exist for this slug (`docs/insights/events/2026-09-{20,22,24}-local-branch-secret-scan.jsonl`) because `scripts/insights-append.sh` names its output file `<events-dir>/<UTC-date-of-the-append-call>-<slug>.jsonl` — the date is computed fresh on every invocation (`_date="$(date -u '+%Y-%m-%d')"`), not pinned to the task's start date. `docs/insights/README.md`'s "File layout" section documents exactly this naming pattern (`<UTC-date>-<slug>.jsonl`), and its "Why per-task files?" rationale (avoiding conflicting concurrent appends across branches) holds regardless of whether one task produces one file or several, since no two branches ever touch the same file either way. Left as three files; no merge and no doc change.

## Gate results

| Gate | Result |
| --- | --- |
| `./scripts/check-skill-sync.sh` | PASS — 13 skills in lock-step |
| `./scripts/check-sync.sh` | PASS — IDENTICAL 159 / DRIFTED 0 / ROOT_ONLY 0 / TEMPLATE_ONLY 11 / KNOWN_DIFF 5 (unchanged from the verify report; `docs/tech-debt/README.md` has no template counterpart, `.gitkeep` only) |
| `./scripts/check-template-purity.sh` | PASS — no meta-repo-specific references in templates |
| `./scripts/check-template.sh` | PASS (exit 0) — prints the same 7 pre-existing `FAIL: ... references missing hook` lines the verify report already reproduced against `main`'s own copy of the script; not introduced or affected by this change |
| `go test ./internal/org/ -run TestSendDefaults -count=1` | PASS |
| `./scripts/secret-scan.sh --staged` (before commit) | clean, exit 0 |
| `./scripts/secret-scan-branch.sh --strict` (after commit) | see commit section below |

No skill, script, `.gitallowed`, or `.github/workflows` file was touched — only `docs/plans/active/2026-09-20-local-branch-secret-scan.md` and `docs/tech-debt/README.md` — so `sync-skills.sh` and the template-copy step were not needed.

## Commit

`docs: sync docs for local-branch-secret-scan` — files: `docs/plans/active/2026-09-20-local-branch-secret-scan.md`, `docs/tech-debt/README.md`, this report.

## Cycle 2

- Date: 2026-09-25
- Plan: `docs/plans/active/2026-09-20-local-branch-secret-scan.md` (issue #169), AC-2d and AC-3b, Scope row 1's current contract
- Agent: `doc-maintainer` subagent (Claude Code), pipeline cycle 2/2
- Precondition confirmed: HEAD `0e7f164` (docs: add cycle-2 test report), `git status --porcelain` empty at start
- Scope: documentation sync for the cycle-2 delta. Reference reports: `docs/reports/self-review-2026-09-21-local-branch-secret-scan.md` `## Cycle 2` (973bd15, MEDIUM 1 / LOW 5, all fixed in fc7d2e4), `docs/reports/verify-2026-09-22-local-branch-secret-scan.md` `## Cycle 2` (7d46144, pass, drift: only the walkthrough), `docs/reports/test-2026-09-24-local-branch-secret-scan.md` `## Cycle 2` (0e7f164, pass, two open gaps: submodule and symlink-to-symlink `.gitallowed` targets confirmed fail-closed by hand but with no suite test — `check-template.sh` required_files gap carried from cycle 1)

### Doc drift check (task item 1)

Re-checked the same shipped-doc surfaces the cycle-2 verify report already covered, plus a couple the handoff added explicitly:

- `.claude/skills/pr/SKILL.md` and its three mirrors (`.agents/skills/pr/`, `templates/base/.claude/skills/pr/`, `templates/base/.agents/skills/pr/`) — byte-unchanged since `ca3d824` (cycle-1 verify's own check, re-confirmed via `git diff ca3d824..HEAD --stat` on all four paths: empty). Step 3's exit-code prose (0/1/3) already describes the contract, not the allowlist-resolution internals the cycle-2 fix rewired — no drift.
- `docs/quality/quality-gates.md` + `templates/base/docs/quality/quality-gates.md` — byte-unchanged since `ca3d824`; both still describe the local mirror at contract level.
- `AGENTS.md` scripts bullet — already names `secret-scan-branch.sh` (added cycle-1 `/sync-docs`); no change needed.
- `README.md`, `docs/quality/definition-of-done.md`, `docs/recipes/*`, `.ralph/core/AGENTS.core.md` — `grep -rl "secret-scan-branch\|gitallowed"` against all four returns nothing; none of them ever named this mechanism, so there is nothing for the cycle-2 internals to make stale.
- `scripts/secret-scan.sh`'s on-finding guidance lines — byte-unchanged since `ca3d824`; unaffected by the cycle-2 delta (which touched only `secret-scan-branch.sh`).
- `scripts/secret-scan-branch.sh`'s own header/usage comment (`:1-29`) — read in full. It already states the fully-qualified-base-ref rule and the committed-`.gitallowed`-with-symlink-resolution rule in its opening paragraph (added as part of the C2-L5 self-review fix in `fc7d2e4`), and the 0/1/2/3 exit-code table is unchanged by the cycle-2 fix (only the internal mechanism deciding which exit fires changed). No drift.

No shipped doc was stale. This matches the cycle-2 verify report's own drift finding (none in shipped docs; one already-known-stale line in the walkthrough report, handled below).

### `docs/tech-debt/README.md` (task item 2)

The cycle-1 row about `check-template.sh`'s `required_files` list (line 135, unchanged) still holds — the cycle-2 delta touched neither `check-template.sh` nor the required-files list itself.

Added one new row for the cycle-2 test gaps: `scripts/secret-scan-branch.sh`'s committed-`.gitallowed` mode dispatch has no regression test for a submodule target (mode `160000`) or a symlink-to-symlink target (`120000` → `120000`); both are confirmed fail-closed by hand (empty allowlist plus the `report_allowlist_unreadable` notice, `--strict` exit 1 on the fixture's own planted finding) per the cycle-2 test report's "Re-confirmations" section, but neither has a fixture in `tests/test-secret-scan-branch.sh` — a submodule fixture needs `git -c protocol.file.allow=always submodule add` plus `.gitmodules`/`.git/modules/` bookkeeping, which the tester judged out of proportion to the one mode-branch check it would pin.

### Walkthrough (task item 3)

Updated `docs/reports/walkthrough-2026-09-24-local-branch-secret-scan.md` for the cycle-2 delta:

- Header 差分規模 line — recomputed from `git diff main...HEAD --stat` (32 files, +3,194/−67) and `wc -l scripts/secret-scan-branch.sh` (298 lines, up from cycle 1's 183), with the two test files' own line counts (1,206 / 299) alongside their assertion counts (96 / 32).
- 何が変わったか item 1 — added that `merge-base` takes the validated full ref (`refs/remotes/origin/<base>` / `refs/heads/<base>`, never the short name), and described the committed-`.gitallowed` mode-dispatch rule (regular file read directly, symlink resolved one level with the absolute/`..` checks, anything else falls back to an empty allowlist plus a notice).
- 読む順番 item 1 — inserted the allowlist mode dispatch (`allowlist_ls_tree_mode` / `allowlist_target_is_safe` / `report_allowlist_unreadable`) between "HEAD の `.gitallowed` を一時ファイルへ" and "無視の通知"; item 4 — updated the assertion count to 96 and named the `test_ac2d_*` / `test_ac3b_*` groups.
- コミット単位 table — added rows for `dbba825` (cross-review cycle 1's AR-1/AR-2 fix, plus the two additional holes the orchestrator's own probe found and folded into the same commit by amend: cwd-relative `ls-tree` and a directory-target symlink with a trailing slash) and `fc7d2e4` (self-review cycle 2's MEDIUM 1 / LOW 5 fixes plus four new tests); extended the "その他" row to mention the cycle-2 reports.
- 設計判断 — added one bullet: symlink resolution goes one level deep only, falling back to an empty allowlist (fail-closed) when it cannot resolve.
- 注意して見てほしい点 — added one bullet about `allowlist_ls_tree_mode`'s single-line/exact-path check and the mode `case` rejecting different inputs (trailing-slash directory vs. bare directory/submodule).
- Known limitations — removed the now-stale symlink-notice-misfire line (closed in cycle 2, per the cycle-2 self-review and test reports); added two lines: the submodule/symlink-to-symlink test gap (same content as the new tech-debt row), and a note that the `./`-normalisation-order fix and the guarded regular-file `git show` produce no observable red/green in the suite (as the self-review predicted; the corrupted-object case was confirmed only by hand).

Tone and format kept consistent with the existing report: Japanese, factual, no headers added, no exaggerated adjectives.

### Plan (task item 4)

Added three Deviation-notes bullets to `docs/plans/active/2026-09-20-local-branch-secret-scan.md`, one per phase, in the same style as the existing entries: `/verify` cycle 2 (7d46144, pass), `/test` cycle 2 (0e7f164, pass, the two gaps above plus the carried-over `check-template.sh` gap), and this `/sync-docs` cycle-2 pass (dated 2026-09-25). No AC checkbox was touched.

### Gate results

| Gate | Result |
| --- | --- |
| `./scripts/check-skill-sync.sh` | PASS — 13 skills in lock-step |
| `./scripts/check-sync.sh` | PASS — IDENTICAL 159 / DRIFTED 0 / TEMPLATE_ONLY 11 / KNOWN_DIFF 5 (unchanged from cycle 1) |
| `./scripts/check-template-purity.sh` | PASS — no meta-repo-specific references in templates |
| `./scripts/secret-scan.sh --staged` (before commit) | see commit section below |
| `./scripts/secret-scan-branch.sh --strict` (after commit) | see commit section below |

No skill, script, `.gitallowed`, or `.github/workflows` file was touched this cycle either — only the plan, the tech-debt register, the walkthrough, and this report — so `sync-skills.sh` and the template-copy step were not needed.

### Insight event

`./scripts/insights-append.sh --slug local-branch-secret-scan --flow standard --phase sync_docs --cycle 2 --verdict complete --critical 0 --high 0 --medium 0 --low 0 --source skill` (best-effort, per the handoff).

### Cycle-2 commit

`docs: sync docs for local-branch-secret-scan (cycle 2)` — files: `docs/plans/active/2026-09-20-local-branch-secret-scan.md`, `docs/tech-debt/README.md`, `docs/reports/walkthrough-2026-09-24-local-branch-secret-scan.md`, this report.
