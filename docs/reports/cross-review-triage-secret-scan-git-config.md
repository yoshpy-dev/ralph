# Cross-review triage report: secret-scan-git-config

- Date: 2026-09-25 (cycle 2; cycle 1 is kept below under `## Cycle 1`)
- Plan: docs/plans/active/2026-09-25-secret-scan-git-config.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 1 (cycle 2)
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0
- User decision (cycle 1, 2026-09-25, AskUserQuestion): fix AR-1 and re-run the full pipeline as cycle 2/2
- User decision (cycle 2, 2026-09-26, AskUserQuestion, cap reached): raise the cap to 3 and fix AR-2 in a third and final pipeline run

## Triage context

- Active plan: docs/plans/active/2026-09-25-secret-scan-git-config.md
- Self-review report: docs/reports/self-review-2026-09-25-secret-scan-git-config.md (cycle 2: LOW 4, all fixed in 443ffb7; the fix for finding 2 added `GIT_ATTR_SOURCE=HEAD`, the line this finding is about, and as the last cycle it was not re-reviewed by the self-review)
- Verify report: docs/reports/verify-2026-09-25-secret-scan-git-config.md (cycle 2: pass)
- Test report: docs/reports/test-2026-09-25-secret-scan-git-config.md (cycle 2: pass)
- Reviewed HEAD: dca6c2f. The reviewer ran the three targeted suites (pass) and reproduced the finding with the merge guard.
- Implementation context summary: `GIT_ATTR_SOURCE=HEAD` makes `--range` read attributes from HEAD's tree, which matches CI for the callers whose range ends at HEAD (CI's own `base..HEAD`, `secret-scan-branch.sh`'s `merge-base..HEAD`). `prepare-commit-msg-secret-guard.sh` is the one caller whose range ends elsewhere: during a merge it scans `HEAD..<merge head>` before HEAD advances, and the attributes that apply to the resulting merge (the working tree, or the incoming side) can differ from HEAD's.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-2 | [P2] Preserve merge-result attributes for incoming-history scans. During a non-fast-forward merge, `prepare-commit-msg-secret-guard.sh` scans `HEAD..$merge_head` before HEAD advances. If HEAD contains `*.txt -diff`, but the incoming branch removes that attribute and adds then deletes a token, forcing attributes from HEAD suppresses the incoming history's text diffs. Reproduced: the original merge guard exits 1, the patched guard exits 0, and the post-merge CI scan exits 1. The staged guard also passes because the token was subsequently deleted. Make attribute-source selection account for merge-state callers rather than unconditionally using HEAD. | Real by code reading: `scan_range` sets `GIT_ATTR_SOURCE=HEAD` for every range, and the merge guard's range does not end at HEAD. It is a regression versus main in the local merge guard only (main read the working-tree attributes, i.e. the merge in progress). The trigger is narrow: HEAD must carry a `-diff`/`binary` attribute for the file type, the incoming branch must remove it, and the token must be added and removed again on the incoming side; CI still detects it once the merge is pushed through a PR. Candidate fix: take the attribute source from the range's right-hand end (the commit the scan is about) instead of always HEAD, i.e. `<merge head>` for the merge guard and HEAD for CI and the branch scan, falling back to HEAD when the range has no parsable right-hand end; plus a test with the reviewer's merge reproduction. | `scripts/secret-scan.sh` (+ template copy), `tests/test-secret-scan.sh` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 1 (2026-09-25, reviewed HEAD f6d1b87)

- Total reviewer findings: 1. After triage (cycle 1): ACTION_REQUIRED 1 (AR-1), fixed in ca8a212 and validated in cycle 2.

### Triage context

- Active plan: docs/plans/active/2026-09-25-secret-scan-git-config.md
- Self-review report: docs/reports/self-review-2026-09-25-secret-scan-git-config.md (MEDIUM 1 / LOW 6; LOW-5, an added line whose content starts with "++ " being dropped by the `+++ ` header rule, was deferred to tech-debt as a pre-existing gap shared with CI)
- Verify report: docs/reports/verify-2026-09-25-secret-scan-git-config.md (pass)
- Test report: docs/reports/test-2026-09-25-secret-scan-git-config.md (pass; CI parity 4/4 on ordinary fixtures)
- Reviewed HEAD: f6d1b87. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the three focused suites (201 assertions, pass) and reproduced the finding. The triager reproduced it too: a committed line whose content is ESC `[32m++ ` followed by a token is detected by the main-branch scanner (rc 1) and missed by this branch's scanner (rc 0).
- Implementation context summary: this branch makes the scanner read the same added lines as CI regardless of local git config. The ANSI strip in `scan_diff_stream` was added as a second defence for colored input (range output already uses `--no-color`). Because the strip runs before the `+++ ` header rule, it changes the meaning of file content that contains escape sequences, in CI as well as locally.

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 | [P2] Preserve added content when removing ANSI sequences. A committed line starting with a literal ESC `[32m++ ` followed by a recognized secret becomes `+++ ...` after the strip and is discarded as a file header; `--range` returned 1 before this patch and 0 after. Range output already uses `--no-color`, so these escapes are file content; this is a new false negative in CI and pre-push scanning. Preserve raw range content or restrict header skipping to actual diff headers. | Real, reproduced by the triager, and a regression this branch introduces (the main-branch scanner detects the line). It is also the same root cause as self-review LOW-5 (content starting with "++ " is dropped even without any escape), which was deferred as pre-existing. Fix: make the header rule state-aware so `+++ ` is skipped only inside a diff header (after a `diff --git` line and before the first `@@` hunk line), and treat every `+` line inside a hunk as added content; keep the ANSI strip for `--diff` input but make sure it cannot turn hunk content into a header (with the state-aware rule it no longer can). Tests: the reviewer's reproduction (range, rc 1), a hunk line whose content starts with "++ " (range and `--diff`, rc 1), the existing synthetic `--diff` input without a hunk header still detected, colored `--diff` input still detected. This also closes the "++ " item of the CI-wide tech-debt row. | `scripts/secret-scan.sh` (+ template copy), `tests/test-secret-scan.sh`, `docs/tech-debt/README.md` |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
