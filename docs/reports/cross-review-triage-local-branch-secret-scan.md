# Cross-review triage report: local-branch-secret-scan

- Date: 2026-09-24 (cycle 1)
- Plan: docs/plans/active/2026-09-20-local-branch-secret-scan.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-09-20-local-branch-secret-scan.md
- Self-review report: docs/reports/self-review-2026-09-21-local-branch-secret-scan.md (12 findings, all fixed in 3afac6e)
- Verify report: docs/reports/verify-2026-09-22-local-branch-secret-scan.md (pass)
- Test report: docs/reports/test-2026-09-24-local-branch-secret-scan.md (pass; it noticed the symlinked `.gitallowed` case only as a cosmetic notice misfire)
- Reviewed HEAD: 286a53c. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the new assertions, the related shell suites and the targeted scaffold test (pass) and reproduced both findings in scratch repos.
- Implementation context summary: `scripts/secret-scan-branch.sh` exists to make the local strict gate say "clean" only when CI's scanner would also say clean. Both findings are cases where strict mode reports clean while CI would report a finding, which is the one outcome this plan is meant to rule out. Both are edge cases that the /pr flow does not reach in this repository (no tag named `main`, no symlinked `.gitallowed`), but scaffolded projects get the same script.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 | [P2] Resolve symlink targets before loading the committed allowlist. When `.gitallowed` is a committed symlink, `git show HEAD:.gitallowed` prints the link's target path, not the allowlist CI reads. That path becomes an unintended regex, so a value containing the target's name passes `--strict` while CI's scanner exits 1; exceptions in the real target are also ignored. | Real: the script writes `git show "HEAD:.gitallowed"` to the temp allowlist without checking the tree entry's mode (`scripts/secret-scan-branch.sh`, the `git cat-file -e` / `git show` lines). A symlink blob's content is its target path. CI's scanner reads the checked-out file, which follows the link. Direction of the error is the bad one (local clean, CI finding). Fix: read the entry's mode with `git ls-tree HEAD -- .gitallowed`; for `120000` resolve one level inside the committed tree (`git show HEAD:<target>` with the target taken relative to the repo root; reject absolute or `..` targets); if the target cannot be read, or the entry is any other non-blob type, use an EMPTY allowlist and print a notice (fail closed: fewer exceptions can only add findings, never hide one). Test with a committed symlink pointing at a committed allowlist file that suppresses a fixture, and with a dangling symlink. | `scripts/secret-scan-branch.sh` (+ template copy), `tests/test-secret-scan-branch.sh` |
| AR-2 | [P2] Preserve the fully qualified base ref for merge-base resolution. The existence checks validate `refs/remotes/origin/<base>` and `refs/heads/<base>`, but `base_ref` is set to the SHORT name (`origin/<base>` or `<base>`) before `git merge-base`. With no remote base and a tag named `main`, git resolves `main` to the tag; the reviewer reproduced a secret-adding commit tagged `main` followed by a deletion commit: strict scans only the deletion and reports clean. A local branch literally named `origin/main` can shadow the remote-tracking ref the same way. | Real: git's refname disambiguation tries `refs/tags/<name>` before `refs/heads/<name>`, and `refs/<name>` (a local branch named `origin/main` lives at `refs/heads/origin/main`, which `origin/main` also matches before `refs/remotes/origin/main`). The script validated one ref and then asked git to resolve a different string. Fix: keep the validated full ref (`refs/remotes/origin/<base>` / `refs/heads/<base>`) for `git merge-base`, `git rev-list` and the range, and use the short form only in the printed line. Tests: a tag named like the base that points at a commit which would make the range skip the leaking commit; a local branch named `origin/<base>` while the remote-tracking ref exists. | `scripts/secret-scan-branch.sh` (+ template copy), `tests/test-secret-scan-branch.sh` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
