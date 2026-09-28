# Cross-review triage report: secret-scan-attr-parity

- Date: 2026-09-28
- Plan: docs/plans/active/2026-09-27-secret-scan-attr-parity.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-09-27-secret-scan-attr-parity.md
- Self-review report: docs/reports/self-review-2026-09-27-secret-scan-attr-parity.md (`## Cycle 2`: MEDIUM 1 / LOW 5; C2-M1 and C2-L2 to C2-L5 fixed in 8443a03; C2-L1 deferred by decision and recorded in docs/tech-debt/README.md)
- Verify report: docs/reports/verify-2026-09-28-secret-scan-attr-parity.md (`## Cycle 2`: pass)
- Test report: docs/reports/test-2026-09-28-secret-scan-attr-parity.md (`## Cycle 2`: pass; 246 / 123 / 32 assertions on git 2.49, the same under dash, docker git 2.40.4 and 2.43.7 with 0 FAIL, six pin mutations each caught by the suite)
- Sync-docs report: docs/reports/sync-docs-2026-09-28-secret-scan-attr-parity.md (`## Cycle 2`: no drift)
- Reviewed HEAD: 8cbb918. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the three targeted suites (401 assertions, pass on git 2.49) and compared the scanner template copies. Its closing statement: "No actionable regressions were found. All 401 assertions across the three targeted test suites passed on Git 2.49, and the scanner template copies match. Full-repository verification and older Git versions were not rerun."
- Implementation context summary: since the cycle-1 review (HEAD af29652), Slice E (fe4f383) runs `git merge-tree` isolated from local config (eight pins, `GIT_ATTR_SOURCE` set to HEAD's commit) and gates the branch suite on the real git version (AR-1, AR-2); Slice F (8443a03) closes the cycle-2 self-review findings (a committed-filter renormalize test, the allowlist mktemp guard, merge-tree environment and directoryRenames tests, the remedy only on causes that merging gets past, doc pointers).
- Outcome: Case C (no findings), so the flow proceeds to `/pr`. The older-git runs the reviewer did not repeat are covered by the cycle-2 test report (docker alpine:3.18 with git 2.40.4 and alpine:3.19 with git 2.43.7, 0 FAIL).

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 1
- Date: 2026-09-28
- Plan: docs/plans/active/2026-09-27-secret-scan-attr-parity.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0
- User decision (2026-09-28, AskUserQuestion): fix both findings (AR-1, AR-2) and re-run the full pipeline as cycle 2/2

### Triage context

- Active plan: docs/plans/active/2026-09-27-secret-scan-attr-parity.md
- Self-review report: docs/reports/self-review-2026-09-27-secret-scan-attr-parity.md (MEDIUM 3 / LOW 3, all fixed in f2ceeca)
- Verify report: docs/reports/verify-2026-09-28-secret-scan-attr-parity.md (pass)
- Test report: docs/reports/test-2026-09-28-secret-scan-attr-parity.md (pass; the git 2.40 test-portability gap was recorded there and in docs/tech-debt/README.md)
- Reviewed HEAD: af29652. Reviewer command: `codex -m gpt-6-astra -c model_reasoning_effort=xhigh exec review --base main` with stdin closed. The reviewer ran the targeted suites (345 assertions, pass on git 2.49) and the branch suite on git 2.40.4 (19 failures), and reproduced the renormalize regression.
- Implementation context summary: when the base changed `.gitattributes` since the merge-base, `secret-scan-branch.sh` runs `git merge-tree --write-tree <base> HEAD` and passes the resulting tree as the scanner's attribute source. The merge-tree call carries no `-c` pins (`use_merge_attributes`), while `scan_range` pins its own git invocation; the local merge-driver guard covers `merge.default` / `merge.<name>.driver` only.

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 | [P2] Disable local renormalization when computing merge attributes. With `merge.renormalize=true` and a user attributes file selecting a local clean filter for `.gitattributes`, `git merge-tree` runs that filter despite the merge-driver guard; a filter emitting `*.txt -diff` made `--strict` return clean where the previous scanner and a default-config merge detect the token. | Real by code reading: the merge-tree invocation has no config pins, so any local setting that changes how `.gitattributes` is merged (renormalize with a clean filter, a `merge=` attribute for `.gitattributes` from the user or system attributes file or attr.tree, rename settings) can change the merged attributes CI never sees. Fix: run merge-tree with the same isolation the scanner uses for its own git call (`GIT_ATTR_NOSYSTEM=1`, `-c core.attributesFile=/dev/null`, `-c attr.tree=`) plus `-c merge.renormalize=false`, and pin the merge's rename settings to git's defaults (`merge.renames=true`, `merge.directoryRenames=conflict`, `merge.renameLimit=1000`); tests: the reviewer's reproduction (renormalize + user attributes file + local clean filter emitting `*.txt -diff`) must give exit 1 under `--strict`, and a `*.gitattributes merge=union` line in a user attributes file must not change the result. | `scripts/secret-scan-branch.sh` (+ template copy), `tests/test-secret-scan-branch.sh` |
| AR-2 | [P2] Gate merge-attribute tests on the installed git version. On git older than 2.41 the branch suite expects merge-derived scanning although production deliberately exits 3; on Alpine 3.18 (git 2.40.4) 19 assertions fail, and `verify.local.sh` runs this suite, so verification fails on such hosts despite correct production behavior. | Real: the /test run found the same 19 failures and recorded them as a gap, but the suite is part of `run-test.sh` / `run-verify.sh`, so a host with git 2.40 or older cannot pass verification. Fix: add a real-git-version capability gate to `tests/test-secret-scan-branch.sh` like `tests/test-secret-scan.sh` has (SKIP the merge-derived cases below 2.41 and assert the documented strict exit 3 there instead), and keep the stubbed-old-git test. | `tests/test-secret-scan-branch.sh`, `docs/tech-debt/README.md` (drop the gap once closed) |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
