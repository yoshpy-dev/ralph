# Test report: add-terraform-language-pack — cycle 3/3

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md`
- Tester: Claude Code (`tester` subagent)
- Scope: Cycle-3 behavioral validation after `03c5598` (gitignore patterns for Terraform state/cache, root + templates/base mirror); promote the verifier's recommended walkthrough into a permanent shell test.
- Evidence:
  - `docs/evidence/test-2026-05-13-add-terraform-language-pack-cycle3-baseline.log` — pre-change baseline run (cycle 1+2 suites only)
  - `docs/evidence/test-2026-05-13-add-terraform-language-pack-cycle3.log` — final run including the new gitignore test

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | <1s |
| `tests/test-check-skill-sync.sh` | (suite-summary) | OK | 0 | 0 | <1s |
| `tests/test-ralph-cli-driver.sh` | 48 | 48 | 0 | 0 | <1s |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 | <1s |
| `tests/test-terraform-pack-verify.sh` | 32 | 32 | 0 | 0 | <1s |
| `tests/test-terraform-rule-frontmatter.sh` | 9 | 9 | 0 | 0 | <1s |
| `tests/test-terraform-gitignore.sh` *(NEW)* | 47 | 47 | 0 | 0 | <1s |
| Static gates (shellcheck / `sh -n` / jq / check-sync / check-pipeline-sync / check-skill-sync) | 20 | 20 | 0 | 0 | <2s |
| `go test ./...` (9 packages) | 9 pkgs | 9 ok | 0 | 0 | cached |
| **Total assertion count** | **155** | **155** | **0** | **0** | — |

Verdict line from `./scripts/run-verify.sh`: `==> All verifiers passed.`

## Coverage

- Statement / branch / function: not instrumented for shell scripts (project convention; coverage is measured by test-case scope).
- Go coverage: not re-run in this cycle because no Go code changed. The cached `internal/...` results are from the cycle-2 instrumented run.
- New surface covered:
  - 13 ignore patterns added by `03c5598` × 2 gitignore files (root + `templates/base/`) = 26 positive sentinel/pattern pairs asserted by exact `git check-ignore -v` pattern match.
  - 4 negative-control HCL source files (`main.tf`, `variables.tf`, `outputs.tf`, `modules/vpc/main.tf`) × 2 gitignore files = 8 "must not be ignored" assertions.
  - 13 cross-check assertions comparing `<source>:<line>:<pattern>` tuples between the root and mirror repos.

## New test added

`tests/test-terraform-gitignore.sh` (47 assertions). Wired into `scripts/verify.local.sh` `run_hook_tests()` alongside the existing terraform suites.

What it asserts, and why each assertion matters:

1. **Pattern-exact ignore matches** for every entry added by commit `03c5598`:
   - `.terraform/` (directory glob — dropping a file inside it must be ignored).
   - `*.tfstate`, `*.tfstate.backup`, `*.tfstate.*.backup` (canonical Terraform state files; state files commonly carry provider secrets, so a silent un-ignore here would be a real-world exposure).
   - `*.tfplan` (binary plan output; can contain sensitive resource attributes).
   - `*.auto.tfvars`, `*.auto.tfvars.json` (auto-loaded variable files; common place for tokens).
   - `override.tf`, `override.tf.json`, `*_override.tf`, `*_override.tf.json` (local-only overrides, by Terraform convention should never be committed).
   - `crash.log`, `crash.*.log` (CLI crash dumps that include local environment).
2. **Negative controls.** Plain `main.tf`, `variables.tf`, `outputs.tf`, and a nested `modules/vpc/main.tf` must NOT be ignored. This catches the failure mode where someone over-broadens the rule to e.g. `*.tf`, which would silently stop tracking actual Terraform source.
3. **Root ⇔ mirror behavioral parity.** For each sentinel, we compare the `<source>:<line>:<pattern>` tuple resolved against the root `.gitignore` against the same tuple resolved against `templates/base/.gitignore`. They must be identical. `scripts/check-sync.sh` already enforces byte equality between the two files, but byte equality is a syntactic check; this asserts the equality is also semantically meaningful for ignore matching, which guards against e.g. encoding glitches that the sync gate might miss.
4. **Hermeticity.** `HOME`, `GIT_CONFIG_GLOBAL`, and `GIT_CONFIG_SYSTEM` are pinned to throwaway paths so the user's `~/.gitignore_global` or `core.excludesFile` setting cannot influence the result. Each gitignore source gets its own fresh `git init` repo under `mktemp -d` so no parent gitignore leaks in.

How it fails informatively: when an assertion fails, the test prints both the expected pattern (e.g. `*.tfstate`) and the actual pattern resolved by `git check-ignore -v` (e.g. `*.tf` if a broadening regression were introduced). For negative controls, it prints the matching `<source>:<line>:<pattern>` so the reviewer can see exactly which line in the gitignore is over-ignoring.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures across 155 assertions.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Cycle-1 baseline (8 detect + 32 verify + 9 rule frontmatter = 49 terraform-pack assertions) | PASS | `test-...-cycle3-baseline.log` |
| Cycle-2 hermeticity hardening (PATH sandbox in `test-terraform-pack-verify.sh`) | PASS | 32/32 still green |
| Existing non-terraform suites (`test-check-mojibake.sh`, `test-check-skill-sync.sh`, `test-ralph-cli-driver.sh`) | PASS | individual suite OK lines |
| Go test packages (9 pkgs) | PASS (cached) | `ok` lines in evidence log |

No prior assertions were weakened or skipped.

## Test gaps

The new test takes the verifier's high-confidence follow-up off the gap list. Remaining (out-of-scope for this cycle, tracked in cycle-2 verify report):

- The `crash.log` / `crash.*.log` patterns are not specific to Terraform — they would also ignore e.g. a Node.js debugger crash log. The test asserts only that the canonical Terraform crash file names are ignored; it does not validate the broader semantics of the rule, since the rule was deliberately written to match Terraform's own documented behavior (`docs/recipes/adding-a-language-pack.md`).
- No test runs `git status --ignored` to inventory which files in a real Terraform repo would be ignored. That would require a fixture Terraform project; pattern-exact matching against synthetic sentinels gives stronger localization on failure.
- `tflint` / `tfsec` cache directories (e.g. `.tflint.d/`, `.trivy/`) are not yet in `.gitignore`; if those tools later write to-be-ignored artifacts, a follow-up cycle should extend both the gitignore and this test in lockstep.

## Verdict

- Pass: **YES** — 155/155 assertions, `./scripts/run-verify.sh` exit 0, `==> All verifiers passed.`
- Fail: none
- Blocked: none

Cycle-3 `/test` gate satisfied. The full pipeline status for issue #52 cycle 3:

- self-review: MERGE (cycle 3)
- verify: PASS (cycle 3)
- test: PASS (cycle 3, this report)

Ready to proceed to `/sync-docs` → `/cross-review` → `/pr` per the canonical pipeline order.
