# Self-review report: add-terraform-language-pack (cycle 2/2)

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md` (issue #52)
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality of commit `f27e1a2` (single file: `tests/test-terraform-pack-verify.sh`, +30/−4). This commit is the fix for the cycle-1 cross-review ACTION_REQUIRED P2 finding. Spec compliance, test coverage, and doc drift are delegated to `/verify` and `/test`.

## Evidence reviewed

- `git show f27e1a2` and `git diff f47b653..f27e1a2` — exactly one file changed.
- Full surrounding context: `tests/test-terraform-pack-verify.sh` lines 1–334 (entire file).
- Producer being tested: `packs/languages/terraform/verify.sh` (lines 1–93).
- Cycle-1 self-review: `docs/reports/self-review-2026-05-13-add-terraform-language-pack.md`.
- Cycle-1 cross-review triage: `docs/reports/cross-review-triage-2026-05-13-add-terraform-language-pack.md` (the originating finding for this cycle).
- Test execution: `bash tests/test-terraform-pack-verify.sh` → `PASS: 32 / 32, FAIL: 0` (confirmed).
- Hermeticity probes (under `/bin/sh`, the test's actual interpreter via `#!/usr/bin/env sh`):
  - Built the hermetic dir exactly as the script does (loop over `sh find grep sed cat chmod mkdir rm printf ls test true false head tr`, `command -v` + `ln -s`).
  - With `PATH="$coreutils_dir"` only, probed `command -v terraform/tofu/tflint/tfsec/trivy` → all five **not found** (correct).
  - Cross-check by simulating a leaked binary: with `PATH="$fakebin:/usr/bin:/bin"` (old behavior), `command -v terraform` resolved to the fake; with `PATH="$coreutils_dir"` (new behavior), it did not resolve. **The fix demonstrably blocks the leak.**
- Coreutils completeness check: grepped `verify.sh` for every command it invokes and cross-referenced against the symlinked list. All non-builtin externals used by the producer (`find`, `grep`, `sed`) are present in the loop. `echo`, `test`, `true`, `false`, `printf` are `/bin/sh` builtins and need no PATH entry (verified via `type echo` etc. in `/bin/sh`).
- Leak-guard banned list `terraform tofu tflint tfsec trivy` was matched against every CLI invocation site in `verify.sh:27–69` — 1:1 match, no missing entries.
- `[ -e symlink ]` semantics verified: returns **false** for broken symlinks (`ln -s /nonexistent x; [ -e x ]` → false). Returns **true** for symlinks to existing files. This determines the leak-guard's effective threat model — see Finding LR-3.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | The symlink-creation loop calls `command -v` for shell builtins (`printf`, `test`, `true`, `false`) which under `/bin/sh` returns the bare builtin name (e.g. literally `"printf"`) rather than an absolute path. This produces a *relative* symlink — `printf -> printf` — which is broken (self-referential, resolves to nothing). On the current host, four of the fifteen entries are broken symlinks. Tests pass anyway because these are all shell builtins that the test stubs and `verify.sh` invoke as builtins, not via PATH lookup. | Direct probe under `/bin/sh`: `command -v printf` → `printf`; `command -v test` → `test`; `command -v true` → `true`; `command -v false` → `false`. After the script runs, `ls -la $coreutils_dir` shows `printf -> printf`, `test -> test`, `true -> true`, `false -> false`. | Either (a) drop these four entries from the loop with a comment that they are builtins, or (b) add a guard: `if printf '%s' "$_resolved" | grep -q "^/" ; then ln -s ...; fi` — i.e. only symlink when the resolution is an absolute path. Not blocking — the broken symlinks are harmless today, and the more honest fix is option (a). Worth a follow-up. |
| LOW | maintainability | The leak-guard catches a symlink-to-existing-binary leak (correct, primary case) but does **not** catch a *broken* symlink leak (e.g. if a future maintainer added `terraform` to the loop but the host has no `terraform`, `command -v terraform` returns empty, the per-tool `[ -z "$_resolved" ]` check fires first with `FAIL: required coreutil 'terraform' not found`, so the leak-guard never even runs). This is the intended threat model — a host where terraform leaks **is** a host where terraform is installed, so the symlink will be non-broken and `[ -e ]` will return true. | `[ -e symlink_to_nonexistent ]` → false (POSIX `test -e` does not follow broken symlinks). `[ -e symlink_to_/bin/sh ]` → true. | Acceptable as written. Optionally tighten by using `[ -L ]` (true for any symlink, even broken) instead of `[ -e ]`, which would catch the future-maintainer typo case **before** the per-tool "required coreutil not found" check. Not required — current behavior fails fast either way. |
| INFO | readability | The comment block (lines 83–92) is excellent — explains *why* `/usr/bin:/bin` is wrong, *what* the new strategy is, and *what happens* if a coreutil is missing. The leak-guard's "future maintainer's typo" rationale (lines 103–105) is also crisp. This is the right level of inline documentation for a security-relevant test fixture. | `tests/test-terraform-pack-verify.sh:83–110` | None — keep as-is. |
| INFO | maintainability | `set -eu` is in effect (line 18), so `_resolved="$(command -v "$_tool" 2>/dev/null || true)"` correctly suppresses the non-zero exit from `command -v` on miss. The `|| true` is load-bearing here, not redundant. | line 96 vs. line 18 | None. |

No CRITICAL, HIGH, or MEDIUM findings. The fix is targeted, addresses the cycle-1 finding correctly, and does not introduce new defects.

## Positive notes

- **The fix actually closes the hermeticity gap.** Empirical proof: a fake `terraform` placed under `PATH="$fakebin:/usr/bin:/bin"` was resolvable by `command -v terraform` (old behavior), but under `PATH="$coreutils_dir"` (new behavior) it was not. The eight stub-CLI scenarios cited in the cross-review triage can now actually exercise the absence branches they claim to test.
- **Leak-guard's banned list is complete and matches the producer.** `terraform tofu tflint tfsec trivy` is exactly the set of CLIs that `verify.sh:27–69` invokes via PATH. No additions, no omissions.
- **Defensive fail-loud semantics.** Both the "required coreutil missing" check (line 97–100) and the "leak detected" check (line 106–110) emit to stderr and `exit 1`. There is no silent degradation path.
- **Cleanup trap remains intact.** The trap on EXIT (line 81) is registered before the hermetic dir is built, so a `exit 1` from either guard correctly triggers `rm -rf "$workdir"` (which removes the partial `$workdir/.coreutils` too).
- **Cleanup runs with the outer PATH, not `clean_path`.** Verified by tracing: `clean_path` is only injected via the inline `PATH="$_path"` assignment in `run_verify_in` (line 141), not exported to the trap. So `rm -rf` in cleanup is not at risk of failing because of a broken `rm` symlink. (Today's `rm` symlink is correct anyway.)
- **Diff is tightly scoped to one file.** No drive-by edits, no formatting churn elsewhere in the test file or in `verify.sh`. Cycle-1's positive notes about unrelated-change discipline carry over.
- **Tests still pass: 32/32.** No regression from the hardening.
- **Alpine/busybox portability is fine.** Symlinking `/bin/find` (which on busybox is itself a symlink to `/bin/busybox`) is safe because the kernel `execve()` keeps `argv[0]` as the basename of the symlink (`find`), which busybox uses to dispatch. The symlink-of-symlink chain resolves correctly.
- **No race conditions.** The `coreutils_dir` is inside the freshly-mktemp'd `$workdir`. No other process knows the path; no TOCTOU window between symlink creation and use.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Builtin entries (`printf`, `test`, `true`, `false`) in the coreutils symlink loop produce broken self-referential symlinks under `/bin/sh`. | Cosmetic noise only — these are builtins, so no test actually depends on the symlinks resolving. Could mislead a future debugger who `ls -la`s the hermetic dir and thinks the test fixture is corrupted. | Tests pass; not blocking; fixing is a 5-minute follow-up. | Next time someone touches this loop, or if `ls -la` output appears in a CI log and confuses a reader. | This report (Finding LR-1) |

_(One row added above; should be appended to `docs/tech-debt/README.md` during `/sync-docs` if not already covered by cycle-1's tech-debt entries.)_

## Recommendation

- Merge: **yes** — proceed to `/verify` for cycle 2.
- Verdict: no CRITICAL or HIGH findings. The two LOW findings concern (a) cosmetic broken symlinks for builtins and (b) a redundancy-vs-tightening choice in the leak-guard's `[ -e ]` vs `[ -L ]`. Neither blocks merge.
- Follow-ups (non-blocking):
  1. Drop `printf test true false` from the coreutils symlink loop and add a one-line comment that they are `/bin/sh` builtins, or guard the `ln -s` on `case "$_resolved" in /*) ln -s ...;; esac`. (Finding LR-1.)
  2. Consider tightening the leak-guard from `[ -e ... ]` to `[ -L ... ]` so it catches even broken-symlink leaks. (Finding LR-2.) Optional.
  3. Let `/verify` and `/test` confirm: (a) the 32 tests still pass under the cycle-2 verifier, (b) the hermetic PATH does not unexpectedly hide a required coreutil on the actual CI image (e.g. GitHub Actions ubuntu-latest), (c) no other test file (e.g. `tests/test-terraform-pack-render.sh`) needs the same hardening if it uses a similar `clean_path` idiom.
