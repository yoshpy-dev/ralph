# Test report (cycle 2): add-terraform-language-pack

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md` (issue #52)
- Tester: tester subagent (Claude Code), cycle 2 / 2
- Cycle 1 reference: `docs/reports/test-2026-05-13-add-terraform-language-pack.md` (114/114 PASS, all green)
- Scope: regression re-run after Codex cross-review P2 fix in commit `f27e1a2` ("test: harden terraform pack verify tests with hermetic PATH"). Confirms all 114 cycle-1 assertions still pass under the new symlink-only coreutils PATH, and explicitly exercises the three hermeticity guarantees added by `f27e1a2`:
  1. Production fake `terraform` on the outer `$PATH` is invisible under `clean_path`.
  2. Leak-guard fires if `terraform`/`tofu`/`tflint`/`tfsec`/`trivy` symlink appears in `$coreutils_dir`.
  3. Missing required coreutil triggers `FAIL: required coreutil ...` and exit 1.
- Evidence: `docs/evidence/test-2026-05-13-add-terraform-language-pack-cycle2.log`

## Cycle 2 deltas vs cycle 1

| Change | Detail |
| --- | --- |
| `tests/test-terraform-pack-verify.sh` | `clean_path="/usr/bin:/bin"` (host-leaky) → freshly-built `$workdir/.coreutils` dir of symlinks-only POSIX coreutils + leak-guard for the 5 banned IaC tool names. +30 / -4 lines, single file. |
| Production code (`packs/languages/terraform/verify.sh`, `scripts/detect-languages.sh`, `.claude/rules/terraform.md`, `templates/...`) | **No change** between cycle 1 and cycle 2. |
| Other test files | **No change** (the other two new suites — `test-detect-languages-terraform.sh`, `test-terraform-rule-frontmatter.sh` — remain byte-identical to cycle 1). |

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | <1s |
| `tests/test-check-skill-sync.sh` | 6 | 6 | 0 | 0 | <1s |
| `tests/test-ralph-cli-driver.sh` | 48 | 48 | 0 | 0 | ~6s |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 | <1s |
| `tests/test-terraform-pack-verify.sh` (now hermetic) | 32 | 32 | 0 | 0 | <1s |
| `tests/test-terraform-rule-frontmatter.sh` | 9 | 9 | 0 | 0 | <1s |
| `go test ./... -count=1` (uncached) | 9 pkgs | 9 | 0 | 0 SKIP at suite-level (1 pre-existing `TestAvailablePacks_WithMockFS` t.Skip inside `internal/scaffold`, unchanged) | ~44s |
| `go run ./cmd/ralph pack list` (integration smoke) | 1 | 1 | 0 | 0 | <1s |
| **Total** | **114** | **114** | **0** | **0** | **~52s** |

`./scripts/run-test.sh` exit code: **0**. Full evidence at `docs/evidence/test-2026-05-13-add-terraform-language-pack-cycle2.log`.

### Per-suite confirmation: every cycle-1 assertion still passes

The hermeticity hardening did not regress any case. Per-suite `PASS: n/n` counters from the cycle-2 log:

```
PASS: 11      (test-check-mojibake.sh)
PASS: 6       (test-check-skill-sync.sh)
PASS: 48      (test-ralph-cli-driver.sh)
PASS: 8 / 8   (test-detect-languages-terraform.sh)
PASS: 32 / 32 (test-terraform-pack-verify.sh) ← hardened
PASS: 9 / 9   (test-terraform-rule-frontmatter.sh)
```

Note: the previously suspect cases under the leaky PATH were **Case B** (markers + no CLI → exit 1, fail-open guard) and **Case C / E / F / G / H / I / J** "tflint/tfsec/trivy missing → skip" assertions. Under the cycle-1 PATH of `/usr/bin:/bin`, a CI host with `apt-get install terraform` would have silently invoked the real binary instead of the stub or asserted-absent path. With the cycle-2 hermetic `$coreutils_dir`, these cases now exercise the absence branches as the assertion names claim.

## Hermeticity regression checks (cycle-2 explicit verifications)

These three checks directly validate the Codex P2 fix beyond just "the suite is green". They were run ad-hoc against `tests/test-terraform-pack-verify.sh` as it sits at commit `f27e1a2`; reproduction commands are in the verdict section.

### Check 1: outer-$PATH `terraform` is invisible under `clean_path`

Reproducer (`/tmp/hermetic-regression/`):

1. Build `.coreutils/` symlink dir exactly as the test does (subset of POSIX tools).
2. Place a fake executable `terraform` at `./fakebin/terraform` that prints `FAKE_TERRAFORM_INVOKED` and exits 0.
3. Confirm fake is found when fakebin is on outer PATH; confirm `command -v terraform/tofu/tflint/tfsec/trivy` all return empty + exit 1 when PATH is the symlink dir only.

Result:

```
--- Sanity: with OUTER PATH including fakebin ---
/tmp/hermetic-regression/fakebin/terraform        ← fake visible if exposed
--- Hermetic: PATH=$coreutils_dir only ---
exit=1                                            ← terraform NOT found
exit=1                                            ← tofu NOT found
exit=1                                            ← tflint NOT found
exit=1                                            ← tfsec NOT found
exit=1                                            ← trivy NOT found
```

PASS — the Codex P2 hermeticity fix actually works. Even if the host had `terraform` installed at `/usr/local/bin/terraform`, the test's `clean_path` would not expose it. This is the regression guard for Codex finding P2.

### Check 2: leak-guard fires when a banned IaC tool slips into the coreutils dir

Reproducer: copy `tests/test-terraform-pack-verify.sh` to a scratch copy, inject `ln -s "$(command -v sh)" "$coreutils_dir/terraform"` between the populate loop and the guard for-loop, and run.

Result:

```
EXIT=1
FAIL: hermetic PATH leak — 'terraform' present in /var/folders/.../tf-pack-test.KINDKJ/.coreutils
```

PASS — guard fires with explicit error naming the banned tool and the directory, then `exit 1` before any cases run. Future maintainers cannot silently add `terraform` (or `tofu`, `tflint`, `tfsec`, `trivy`) to the symlink list without the suite immediately failing.

Edge note (recorded, not blocking): the guard uses `[ -e ]`, which is **false for broken symlinks** (target does not exist). An attacker / future bug could in theory create a `terraform` symlink with an empty/nonexistent target and skip the guard. In practice this would also defeat the leak — a broken symlink to nowhere does not expose a real binary — so the guard's protection of the actual threat model (real IaC binary exposure) holds. Tightening to `[ -L ] || [ -e ]` would catch the cosmetic edge but is optional and out of scope for this cycle.

### Check 3: missing required coreutil triggers `FAIL: required coreutil ...` + exit 1

Reproducer: copy the test, replace the populate loop's tool list with one that includes a deliberately nonexistent name (`ralph_nonexistent_tool_xyz`), and run.

Result:

```
EXIT=1
FAIL: required coreutil 'ralph_nonexistent_tool_xyz' not found on host PATH
```

PASS — exit code 1, message names the missing tool, no cases run. The fail-loud-rather-than-degrade contract from the commit message is honored.

## Coverage

- Statement: unchanged from cycle 1 (no production code change). Every branch in `packs/languages/terraform/verify.sh` remains exercised by at least one of the 32 cases in `test-terraform-pack-verify.sh`. The cycle-2 hardening only changes **how** the absence branches are exercised (truly hermetic) without changing **which** branches are covered.
- Branch: unchanged from cycle 1. Plus, cycle-2 ad-hoc checks above newly exercise three previously-implicit branches of the **test harness itself**: outer-PATH isolation, leak-guard true branch, and missing-coreutil exit branch.
- Function: unchanged from cycle 1.
- Notes:
  - Cycle-1 report's coverage section remains accurate. The hermetic PATH delta is purely an isolation strengthening — no case was added, removed, or weakened.
  - One cosmetic LOW (flagged by cycle-2 self-review) is now confirmed by direct inspection of the test workspace: `command -v` resolves shell builtins (`printf`, `test`, `true`, `false`) to bare names, not absolute paths, so the loop creates 4 self-referential broken symlinks for them:

    ```
    lrwxr-xr-x  printf -> printf
    lrwxr-xr-x  test   -> test
    lrwxr-xr-x  true   -> true
    lrwxr-xr-x  false  -> false
    ```

    These are harmless because `verify.sh` runs under `#!/usr/bin/env sh`, where those names resolve as shell builtins inside the child process and never go through PATH. The hermetic absolute-path symlinks for the remaining 13 tools (`sh`, `find`, `grep`, `sed`, `cat`, `chmod`, `mkdir`, `rm`, `ls`, `head`, `tr`) suffice. **Not blocking**; recorded for future polish.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| _none_ | — | — | — |

No failures. All 114 assertions pass; all three hermeticity guarantees verified.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Cycle-1 "all 114 pass" baseline must still hold after Codex P2 fix (`f27e1a2`) | PASS | `./scripts/run-test.sh` exit 0; per-suite counters match cycle 1. |
| Codex cross-review P2 hermeticity: host-installed `terraform`/`tofu`/`tflint`/`tfsec`/`trivy` must NOT leak into the stub-CLI absence branches | PASS | Direct outer-PATH check (Check 1). The 8 stub-CLI cases (B + C+E+F+G+H+I+J's "optional missing" arms) now exercise true absence. |
| Test-harness self-protection: leak-guard fires on banned-tool symlink | PASS | Check 2 reproduces guard + exit 1 + explicit message. |
| Test-harness self-protection: missing coreutil aborts loudly | PASS | Check 3 reproduces `FAIL: required coreutil ...` + exit 1. |
| `./scripts/check-sync.sh` byte-identity (packs/languages/terraform/ ↔ templates/packs/terraform/, .claude/rules/terraform.md ↔ templates/base/.claude/rules/terraform.md) | PASS | Invoked indirectly via `verify.local.sh` during `./scripts/run-test.sh` static phase; no drift reported in log. |
| `go test ./...` uncached whole-suite green | PASS | All 9 test packages `ok` (`internal/action`, `internal/cli`, `internal/config`, `internal/scaffold`, `internal/state`, `internal/ui`, `internal/ui/panes`, `internal/upgrade`, `internal/watcher`). |
| `ralph pack list` integration: terraform discoverable via PackFS auto-discovery | PASS | `go run ./cmd/ralph pack list` output includes `terraform` alongside dart/golang/python/rust/typescript. |

## Test gaps

Inherited from cycle 1 — all four cycle-1 gaps remain explicitly out of scope and unchanged:

1. **`ralph pack add terraform` end-to-end placement** — still not tested; pre-existing `addPack` bug, plan Non-goal R6, recommended for follow-up issue at `/pr` time.
2. **Live `terraform` / `tofu` invocation against a real fixture** — still stubbed; correctness of dispatch fully covered by stub-based assertions.
3. **`tflint` / `tfsec` / `trivy config` PASS paths** — only the "missing → skip" arms are tested; optional linters per plan design decision #5.
4. **Hidden-file glob (`**/.terraform.lock.hcl`) runtime path matching by Claude Code editor** — frontmatter contract pinned; runtime correctness external.

Cycle 2 introduces no new gaps. Three minor records (non-blocking):

5. **Self-referential broken symlinks for shell builtins** (`printf`, `test`, `true`, `false`) — confirmed cosmetic LOW from self-review. Could be polished by either (a) skipping builtins explicitly in the populate loop, or (b) verifying each `command -v` result is an absolute path before symlinking. Not required for the Codex P2 fix; deferred.
6. **`[ -e ]` does not catch broken-symlink leaks in the leak-guard** — recorded under Check 2. The realistic attack/regression case (real IaC binary symlink) is caught; the broken-symlink-to-nowhere case is not. Optional tightening to `[ -L ] || [ -e ]`. Deferred.
7. **No persistent regression test for the three meta-checks themselves** — Checks 1, 2, 3 above were run ad-hoc, not committed as a new suite. Rationale below.

None of the above block PR creation.

## Why no new committed regression slice this cycle

The user instruction stated *"If new regression test cases are added, commit them as a separate slice with `test:` prefix."* This cycle adds zero new committed test files. Rationale:

- The Codex P2 fix (`f27e1a2`) repairs the **existing** 32 cases in `test-terraform-pack-verify.sh` so they truly exercise their named absence branches. Those 32 cases ARE the regression test for the P2 finding.
- Checks 1, 2, 3 above are **ad-hoc verifications of the test harness's own contracts**, not behaviors of the production code under test. Promoting them to permanent files would mean writing a "test of a test", which adds harness complexity without protecting any product behavior beyond what the existing 32 cases plus the inline leak-guard already encode.
- The leak-guard and missing-coreutil aborts are **self-enforcing at every test-suite run** (they fire before any case if conditions hold), so they protect themselves; no external assertion is needed.

If the user wants Checks 1, 2, 3 codified, a follow-up `test:` slice could add a small `tests/test-terraform-pack-harness.sh` that runs the three reproducers above. This is offered as an optional follow-up, not blocking.

## Verdict

- **Pass.** All 114 assertions across all suites passed under the cycle-2 hermetic PATH. Go suite is green (uncached). All three hermeticity guarantees from commit `f27e1a2` are independently verified.
- **Fail:** none.
- **Blocked:** none.

Proceed to `/cross-review` (cycle 2), then `/pr`.

### Reproduction summary

```
# Full suite (matches the report's headline number)
./scripts/run-test.sh
# Go suite uncached
go test ./... -count=1
# Integration smoke
go run ./cmd/ralph pack list

# Check 1 — outer-PATH terraform invisible under clean_path
#   (snippet captured in body; rebuild .coreutils/ + fakebin/terraform)

# Check 2 — leak-guard fires on banned-tool symlink
cp tests/test-terraform-pack-verify.sh /tmp/leak.sh
# inject: ln -s "$(command -v sh)" "$coreutils_dir/terraform"
#   between the populate loop and the for-loop "for _banned in ..."
cp /tmp/leak.sh tests/.tmp-leak.sh && chmod +x tests/.tmp-leak.sh
tests/.tmp-leak.sh; echo $?    # expect: 1, with "hermetic PATH leak — 'terraform'"
rm tests/.tmp-leak.sh

# Check 3 — missing-coreutil aborts loudly
cp tests/test-terraform-pack-verify.sh /tmp/missing.sh
# inject "ralph_nonexistent_tool_xyz" into the _tool list
cp /tmp/missing.sh tests/.tmp-missing.sh && chmod +x tests/.tmp-missing.sh
tests/.tmp-missing.sh; echo $?  # expect: 1, with "required coreutil 'ralph_nonexistent_tool_xyz'"
rm tests/.tmp-missing.sh
```
