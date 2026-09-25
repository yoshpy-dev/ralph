# Cross-review triage report: local-branch-secret-scan

- Date: 2026-09-25 (cycle 2; cycle 1 was 2026-09-24, see `## Cycle 1` below)
- Plan: docs/plans/active/2026-09-20-local-branch-secret-scan.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2 (cycle 2)
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0
- User decision (2026-09-24, AskUserQuestion, cycle 1): fix both findings (AR-1, AR-2) and re-run the full pipeline as cycle 2/2
- User decision (2026-09-25, AskUserQuestion, cycle 2, cap reached): create the PR with AR-3 and AR-4 recorded as known gaps and fix them in a follow-up issue

## Triage context

- Active plan: docs/plans/active/2026-09-20-local-branch-secret-scan.md
- Self-review report: docs/reports/self-review-2026-09-21-local-branch-secret-scan.md (`## Cycle 2`: MEDIUM 1 / LOW 5, all fixed in fc7d2e4)
- Verify report: docs/reports/verify-2026-09-22-local-branch-secret-scan.md (`## Cycle 2`: pass)
- Test report: docs/reports/test-2026-09-24-local-branch-secret-scan.md (`## Cycle 2`: pass; 96 assertions, six mutations, open gaps recorded in docs/tech-debt/README.md)
- Reviewed HEAD: f720cc2. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the added tests and the targeted scaffold test (pass) and reproduced both findings in scratch repos. The triager reproduced both in fixture repos as well: (P1) with a repo-local `color.ui=always`, `--strict` exits 0 `clean` while the same history exits 1 without it; the first added line of `git log -p` starts with an escape sequence before the `+`. (P2) with `.gitallowed` a committed symlink whose target ends in a newline, and both `rules` (holding the exception) and the newline-suffixed name committed, `--strict` exits 0 (with the "ignoring uncommitted edits" notice, since the checked-out link resolves to the other file) while the scanner run on the checked-out tree exits 1.
- Implementation context summary: cycle 1 fixed the two bypasses the reviewer found then (symlinked `.gitallowed`, short base ref). Both cycle-2 findings are again of the one kind this plan is meant to rule out: `--strict` reports clean while CI would report a finding. P1 is a weakness of the existing scanner (`scripts/secret-scan.sh` runs `git log -p` without `--no-color`) that the new local gate inherits; CI runners use the default `color.ui=auto`, so CI itself is unaffected. P2 needs a committed file name and a link target that both end in a newline, which no ordinary repository has.
- Cap: the pipeline cycle cap (`RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`) is reached, so the fix-and-re-run option is not offered; the user chooses between raising the cap, creating the PR with the findings recorded as known gaps, or aborting.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-3 | [P1] Disable Git diff coloring before running the local scanner. With `color.ui=always`, the scanner's `git log -p` output is colored; its added-line parser only accepts lines beginning with `+`, so escape-prefixed additions are silently discarded. Reproduced: `--strict` reports clean and exits 0, while disabling coloring makes the same history exit 1. | Real, and reproduced by the triager. `scripts/secret-scan.sh` runs `git log --format='commit %H' --no-ext-diff -p` without `--no-color`; `color.ui=always` or `color.diff=always` at any config level colors non-tty output, the `+` is preceded by an escape sequence, and `scan_diff_stream` sees no added lines. Pre-existing scanner weakness surfaced by the new gate (also affects `run-verify.sh`'s use of the scanner), direction is the bad one (local clean, CI finding). Fix: add `--no-color` to the scanner's `git log` call (and its template copy), optionally also run the scanner from `secret-scan-branch.sh` with color forced off via `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_0=color.ui`/`GIT_CONFIG_VALUE_0=false`; tests: a fixture repo with `git config color.ui always` must still exit 1 in `tests/test-secret-scan.sh` and `tests/test-secret-scan-branch.sh`. | `scripts/secret-scan.sh` (+ template copy), `scripts/secret-scan-branch.sh` (+ template copy, optional), `tests/test-secret-scan.sh`, `tests/test-secret-scan-branch.sh` |
| AR-4 | [P2] Preserve trailing newlines when reading symlink targets. Command substitution strips trailing newlines from the stored symlink target. If `.gitallowed` points to a name ending in a newline and both that name and the stripped name exist, the script resolves the wrong file. Reproduced: with the exception only in the stripped-name file, `--strict` exits 0 while CI follows the actual link and exits 1. | Real but contrived (needs a committed file name and a link target that both end in a newline), reproduced by the triager. `$(git show HEAD:.gitallowed)` drops trailing newlines, so the resolved name differs from the link's real target; the "ignoring uncommitted edits" notice fires (the checked-out link resolves elsewhere) but does not block. Direction is the bad one. Fix: compare the blob size (`git cat-file -s HEAD:.gitallowed`) with the byte length of the stripped target (`printf '%s' "$t" \| wc -c`); on mismatch treat the target as unresolvable (empty allowlist plus the existing notice, fail closed). Test: fixture with both files committed and the exception only in the stripped-name file must exit 1 with the notice. | `scripts/secret-scan-branch.sh` (+ template copy), `tests/test-secret-scan-branch.sh` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 1 (2026-09-24, reviewed HEAD 286a53c)

- Total reviewer findings: 2. After triage (cycle 1): ACTION_REQUIRED 2 (AR-1, AR-2), WORTH_CONSIDERING 0, DISMISSED 0. Both fixed in dbba825 and validated in cycle 2.

### Triage context

- Active plan: docs/plans/active/2026-09-20-local-branch-secret-scan.md
- Self-review report: docs/reports/self-review-2026-09-21-local-branch-secret-scan.md (12 findings, all fixed in 3afac6e)
- Verify report: docs/reports/verify-2026-09-22-local-branch-secret-scan.md (pass)
- Test report: docs/reports/test-2026-09-24-local-branch-secret-scan.md (pass; it noticed the symlinked `.gitallowed` case only as a cosmetic notice misfire)
- Reviewed HEAD: 286a53c. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the new assertions, the related shell suites and the targeted scaffold test (pass) and reproduced both findings in scratch repos.
- Implementation context summary: `scripts/secret-scan-branch.sh` exists to make the local strict gate say "clean" only when CI's scanner would also say clean. Both findings are cases where strict mode reports clean while CI would report a finding, which is the one outcome this plan is meant to rule out. Both are edge cases that the /pr flow does not reach in this repository (no tag named `main`, no symlinked `.gitallowed`), but scaffolded projects get the same script.

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 | [P2] Resolve symlink targets before loading the committed allowlist. When `.gitallowed` is a committed symlink, `git show HEAD:.gitallowed` prints the link's target path, not the allowlist CI reads. That path becomes an unintended regex, so a value containing the target's name passes `--strict` while CI's scanner exits 1; exceptions in the real target are also ignored. | Real: the script writes `git show "HEAD:.gitallowed"` to the temp allowlist without checking the tree entry's mode (`scripts/secret-scan-branch.sh`, the `git cat-file -e` / `git show` lines). A symlink blob's content is its target path. CI's scanner reads the checked-out file, which follows the link. Direction of the error is the bad one (local clean, CI finding). Fix: read the entry's mode with `git ls-tree HEAD -- .gitallowed`; for `120000` resolve one level inside the committed tree (`git show HEAD:<target>` with the target taken relative to the repo root; reject absolute or `..` targets); if the target cannot be read, or the entry is any other non-blob type, use an EMPTY allowlist and print a notice (fail closed: fewer exceptions can only add findings, never hide one). Test with a committed symlink pointing at a committed allowlist file that suppresses a fixture, and with a dangling symlink. | `scripts/secret-scan-branch.sh` (+ template copy), `tests/test-secret-scan-branch.sh` |
| AR-2 | [P2] Preserve the fully qualified base ref for merge-base resolution. The existence checks validate `refs/remotes/origin/<base>` and `refs/heads/<base>`, but `base_ref` is set to the SHORT name (`origin/<base>` or `<base>`) before `git merge-base`. With no remote base and a tag named `main`, git resolves `main` to the tag; the reviewer reproduced a secret-adding commit tagged `main` followed by a deletion commit: strict scans only the deletion and reports clean. A local branch literally named `origin/main` can shadow the remote-tracking ref the same way. | Real: git's refname disambiguation tries `refs/tags/<name>` before `refs/heads/<name>`, and `refs/<name>` (a local branch named `origin/main` lives at `refs/heads/origin/main`, which `origin/main` also matches before `refs/remotes/origin/main`). The script validated one ref and then asked git to resolve a different string. Fix: keep the validated full ref (`refs/remotes/origin/<base>` / `refs/heads/<base>`) for `git merge-base`, `git rev-list` and the range, and use the short form only in the printed line. Tests: a tag named like the base that points at a commit which would make the range skip the leaking commit; a local branch named `origin/<base>` while the remote-tracking ref exists. | `scripts/secret-scan-branch.sh` (+ template copy), `tests/test-secret-scan-branch.sh` |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
