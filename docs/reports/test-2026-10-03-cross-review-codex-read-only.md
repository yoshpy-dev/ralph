# Test report: cross-review-codex-read-only

- Date: 2026-10-03
- Plan: docs/plans/active/2026-10-03-cross-review-codex-read-only.md (issue #197, security)
- Tester: tester subagent (Claude Code), cycle 1 of 2
- Scope: behavioral tests only. Branch `security/cross-review-codex-read-only`, HEAD `85208137`, base `main` `c9da0a27`. Static analysis is `/verify`'s job (PASS in `docs/reports/verify-2026-10-03-cross-review-codex-read-only.md`). The change is skill text, docs, comments and `tests/test-codex-exec-invocation.sh`; no Go or production shell code changed, so the Go suite is a no-regression check only.
- Evidence: `docs/evidence/test-2026-10-03-cross-review-codex-read-only.log` (gitignored; full run output, mutation results, stub argv results); `docs/evidence/verify-2026-10-03-022706.log` (default scope) and `docs/evidence/verify-2026-10-03-023116.log` (full scope), written by `run-verify.sh` in test mode. All scratch work (mutant trees, probes, stubs) lived outside the repo.

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` (foreground) | 32 shell test files (1268 `PASS:` lines, 22 `FAIL: 0` summaries, no other `FAIL`) + 8 Go packages | all | 0 | 0 | 3m45s wall, exit 0, `All verifiers passed.` |
| `./scripts/run-test.sh` (default scope) | identical set to the full run | all | 0 | 0 | 3m55s wall, exit 0 |
| `sh tests/test-codex-exec-invocation.sh` x3 | 132 each | 132 | 0 | 0 | ~0.43s each; the 3 outputs are byte-identical |
| `dash tests/test-codex-exec-invocation.sh` x3 | 132 each | 132 | 0 | 0 | ~0.40s each; byte-identical to the `sh` output |
| `go test ./... -count=1` (uncached, extra) | 8 packages | 8 ok | 0 | 0 | cli 45.3s, config 2.7s, insights 1.3s, org 10.9s, org/driver 1.6s, org/protocol 2.2s, scaffold 1.6s, upgrade 3.0s |

What the default `./scripts/run-test.sh` selects: `Requested scope: changed` resolved to `Language scope: full fallback (unclassified:.codex/config.toml)`, `Language packs selected: golang`. `.codex/config.toml` is in the diff and has no language pack, so the #195 merge-base detector falls back to full instead of `no_changes`. Both runs therefore executed the same 32 shell files and the Go suite. `internal/org` ran uncached in both (9.3s, 7.4s); the other Go packages came from the build cache (the diff touches no Go), which the extra `-count=1` run covers.

Per-suite totals printed by the full run: branch-name 26, check-template 54, codex-exec-invocation 132 (main had 96), ensure-pr-ready 7, ensure-pr-title-prefix 13, gc-artifacts 11, insights-append 39, ralph-config 19, ralph-worktree 143; the other suites print PASS lines only.

## Linux / mawk (CI parity)

CI runs on ubuntu, where `awk` is mawk. The new check relies on a 50-line awk tokenizer (`CODEX_ARGS_AWK`), so awk dialect is the main portability risk.

Environment: `ubuntu:24.04` container (aarch64 Linux, Docker Desktop on this Mac), `awk` -> `/usr/bin/mawk` (mawk 1.3.4 20240123), `/bin/sh` -> dash, bash 5.2.21. The tree was piped in with `git archive HEAD | docker run --rm -i ubuntu:24.04 bash -c '... tar -x ...'` (no `-v` mounts).

| Shell in container | Runs | Result |
| --- | --- | --- |
| `sh` (dash) | 3 | 132/132, exit 0, 0 `FAIL:` lines each |
| `dash` | 3 | 132/132, exit 0 |
| `bash` | 3 | 132/132, exit 0 |

The nine outputs are identical to each other and to the macOS output. All 247 mutant trees (below) were then run in the same container under sh, dash and bash (741 runs). Exit code, `Results:` counts and the set of failing assertion names are identical to the macOS run (BSD awk 20200816) for every mutant, and identical across the three shells. Caught/survived is 102/145 mutants (41/52 distinct mutations) on both platforms. Not reproduced: x86_64 (mawk is the same program).

## Mutation hunt

Method. A `git archive HEAD` mirror lives in scratch. For each mutation a fresh tree is built from it (the 8 skill files, the test, `scripts/ralph-config.sh`), one invocation line is edited, and the suite is run under `sh` and `dash` (and sh/dash/bash on Linux). Line types: P (`/plan` step 11.c), S (`/cross-review` Step 4 fenced block), T (the "CLI execution modes" table row), applied to `.claude/skills` (the suite applies one tokenizer to all four faces). 93 distinct mutations, 247 mutants (site and shape variants). The mirror was never edited in place: its checksum at the end equals the one taken at creation, `cmp` of the worktree's test and skill files against the mirror is clean, and `git status --porcelain` is empty in the worktree and the main checkout.

For every survivor two independent checks were made:
1. Does the shell hand codex that argv? The mutated Step 4 / step 11.c line was executed under `bash` and `zsh` with a stub `codex` first on PATH that records `"$@"`.
2. Does codex 0.154.0 honor it? No-auth header probe: scratch `HOME` and `CODEX_HOME` whose `config.toml` sets `sandbox_mode = "danger-full-access"` (standing in for the project config), scratch git repo with `main` + a feature branch, `codex exec review ...` stopped as soon as the `sandbox:` header line is printed. For the highest-risk class the real mutated skill line was run end to end (zsh, watchdog and all) against the real binary in that scratch HOME. The user's real `~/.codex` was not read or written by any probe.

Controls: the unmutated trees and equivalent spellings stay green at 132/132 (`ctl-baseline`, `ctl-spaced-readonly` = `-c 'sandbox_mode = "read-only"'`, `root-benign-c`).

### Caught (41 mutations, all sites; each trips 1 assertion unless noted)

| Mutation | Result |
| --- | --- |
| Remove the read-only selector (S/T and P); remove `--ignore-rules` (P/S/T) | caught: "selects no sandbox" / "missing --ignore-rules" |
| `--sandbox danger-full-access`, `--sandbox=workspace-write`, `-s workspace-write`, `-sworkspace-write` (P) | caught: "selects a sandbox other than read-only" |
| `sandbox_mode=workspace-write` in place of read-only; later `-c sandbox_mode=danger-full-access` (S/T and P); `--config sandbox_mode=...`; `--config=sandbox_mode=...`; glued `-csandbox_mode=...`; TOML `sandbox_mode='workspace-write'`; TOML with spaces around `=`; quoted TOML key; profile-scoped `profiles.x.sandbox_mode`; uppercase value; extra spaces; tab separator | caught: "selects a sandbox other than read-only" |
| `-c sandbox_workspace_write={network_access=true}`; `-c approval_policy=never`; `--yolo`; `--dangerously-bypass-approvals-and-sandbox`; `--dangerously-bypass-hook-trust`; `--add-dir /`; `--approve-for-me`; `-p x`; `--yolo` after `</dev/null`; `--yo''lo`, `--yo\lo`, `"--yolo"` | caught: "names an option that widens or bypasses ..." |
| `command codex --yolo ...` appended after `&&`, `;`, `|` | caught: reported as `[codex #2]`, 3 assertions |
| A second physical line holding another invocation | caught: invocation count 3 != 2 plus 5 more assertions |
| Continuation that moves `</dev/null` / `-o` / `--ignore-rules` to the next physical line (`--sandbox \<nl>danger-full-access`; split before `-o`) | caught: 4 and 2 assertions |
| `--ignore-rule` (abbreviation) or `--ignore-user-config` in place of `--ignore-rules`; `env FOO=1 command codex` shape | caught |

### Survived (52 distinct mutations; 28 are in bypass-capable classes)

| Class | Mutations | Suite | Shell hands codex that argv | codex 0.154.0 honors it | Assessment |
| --- | --- | --- | --- | --- | --- |
| G-1 `-c` placement: any `-c` after `exec` (after `review`, after `--base`, between `exec` and `review`) | `late-c-after-ignore`, `late-c-after-base`, `late-c-between-exec-review` at S and T; also `ctl-harmless-extra-c` (`-c foo=bar`) at S and T | green (132/132) | yes (bash, zsh) | **yes: the root-level `-c` options are discarded.** Header of the real mutated line: `sandbox: danger-full-access`, `reasoning effort: none` (baseline `read-only`, `xhigh`). Even `-c model_reasoning_effort=high` after `exec review` does it. A `-c` before `exec` (`root-benign-c`) is safe. | **Real gap, realistic regression.** A benign key=value appended after the subcommand silently removes the sandbox and the suite stays green. At P the `--sandbox read-only` flag survives (header `read-only`) but the root `-c model_reasoning_effort` is lost (`effort: none`). |
| G-2 backslash-newline continuation after `</dev/null` with the widening on the next physical line | `cont-after-devnull`, `cont-after-devnull-yolo` at S and P | green | yes (argv ends `... -o <f> --yolo` / `-c sandbox_mode=danger-full-access`) | `--yolo`: yes (header `danger-full-access` at S and P). `-c sandbox_mode=danger-full-access` after `-o`: yes at S, **no at P** (the `--sandbox read-only` flag wins) | Gap: only physical line 1 of an invocation is scanned, and the count check does not see the continuation line (no ` exec `). Needs a deliberate edit. |
| G-3 second invocation not shaped `command codex ...` at the start of a simple command | `( command codex --yolo ...)`, `(command ...)`, `{ ...; }`, bare `codex`, `/usr/local/bin/codex`, `env codex`, `sh -c '...'`, `eval "..."`, `time command codex`, `if ...; then command codex ...; fi` at P, S, T | green | yes (2 stub calls, the 2nd with `--yolo`; `/usr/local/bin/codex` does not exist here, same class, not executed) | yes: `--yolo` gives `danger-full-access` at root and subcommand level | Gap, adversarial only: the scanner reads only simple commands that begin with `command codex`. `&&`, `;`, `|`, `||`, `&` work as separators (caught). |
| G-4 `#` shell comment hides the flags | `comment-hides-flags` (P, S: `--ignore-rules` after `# `), `comment-hides-sandbox` (S) | green | yes: stub argv lacks `--ignore-rules` / `-c sandbox_mode=read-only` | yes (no flag, no override) | Gap, but the same edit comments out `> log 2>&1 &`, the watchdog and `echo "codex rc="`, so the skill visibly stops working. |
| G-5 shell expansion that produces an option | `$(echo --yolo)`, backtick, `${Y:---yolo}`, variable inside the `-c` key (`${K:-sandbox_mode}=...`), brace expansion `--yol{o,}`, ANSI-C quote `$'\x2d\x2dyolo'` at P, S, T (backtick: P, S); glob `--yol[o]` | green | yes in bash and zsh (stub). Glob: bash passes it literally, zsh stops with `no matches found` | yes once expanded (real line header `danger-full-access`); the literal glob is rejected by clap | Known class in the plan ("`$(...)` and unquoted expansions are not read"), but `$(...)` is **not** the only form. An expansion in the *value* of a recognised key is caught (the value must equal `read-only` literally). |
| G-6 `-c` keys outside the sandbox/approval allowlist that start programs | `-c mcp_servers.x.command=touch -c 'mcp_servers.x.args=["<path>"]'` (root level and after `exec`), `-c 'notify=["touch","<path>"]'` | green | yes | MCP: **yes**, header stays `sandbox: read-only` and the marker file was created before any model call (codex starts the program outside the command sandbox). notify: not verified (needs a credentialed turn) | Outside the suite's declared scope; residual risk (a) in `docs/tech-debt/README.md`, reached through a different door. |
| Not honored by codex 0.154.0 (17 mutations) | uppercase / mixed-case `-c SANDBOX_MODE=...`, `-c Sandbox_Mode=...` (root level: header stays `read-only`); `--yol`, `--dangerously-bypass-approvals` (clap: `unexpected argument`); `--ignore-rules=false` (`unexpected value 'false'`), `--no-ignore-rules` (`unexpected argument`); `-c ignore_rules=false` (`unknown configuration field` under `--strict-config`, not a key); `--cd /` (`unexpected argument` for `exec review`); `--enable foo` (`Unknown feature flag`); env prefixes `CODEX_PERMISSION_PROFILE=:danger-no-sandbox`, `CODEX_SANDBOX=none`, `CODEX_EXEC_SERVER_URL=...`, `CODEX_NON_INTERACTIVE=1` (header unchanged); `CODEX_HOME=<missing dir>` (codex exits: `Error finding codex home`) | green | n/a | no | No action. Fail-closed or no-op. (`CODEX_EXEC_SERVER_URL`: remote-exec behaviour during a turn is unverified.) |
| Permission-related feature flags | `--enable request_permissions_tool`, `--enable exec_permission_approvals` (both "under development") | green | n/a | header unchanged (`read-only`); effect during a turn unverified | Low; `--enable` is not on the suite's denylist. |
| Equivalent or narrowing, survive by design (6) | `ctl-baseline`, `ctl-spaced-readonly`, `root-benign-c`, quoted `'--ignore-rules'` (same argument), `--strict-config`, `--ignore-user-config` (drops user config, keeps `-c` and `--ignore-rules`) | green | n/a | n/a | Expected. |

Also surviving: `late-c-after-exec-plan@P` (`exec -c model_verbosity=low --sandbox read-only`). Benign for the sandbox (the flag wins), but it drops the root-level effort `-c`.

## Real cross-review run on final HEAD (optional step 4)

The exact Step 4 line from `.claude/skills/cross-review/SKILL.md` at HEAD `85208137` was run under zsh with the 20-minute watchdog, `</dev/null`, and `-o` / log in scratch. The cwd was this worktree and `BASE=main` came from `detect_base_branch`. Main checkout not used for codex.

- `codex rc=0`; the `-o` file is 257 bytes (non-empty): "No actionable regressions found. The focused test passed 132/132 checks under dash; mutation checks, mirrored-file comparisons, shell syntax, and ShellCheck also passed. Full-suite and live reviewer execution were not verified in this read-only environment."
- log header: `workdir: .../.claude/worktrees/cross-review-codex-read-only`, `model: gpt-6-astra`, `approval: never`, `sandbox: read-only`, `reasoning effort: xhigh`.
- 21 `Operation not permitted` lines: 12 are `tests/test-codex-exec-invocation.sh: line 200: cannot create temp file for here document` (the reviewer tried to run the suite and the write was refused), 5 are the nested-codex "could not create PATH aliases" warning, 4 are reads of docs that quote the phrase. A refused write did not stop the review (rc 0 and `-o` written): the plan's edge case holds.
- afterwards `git status --porcelain` is empty in the worktree and in the main checkout.
- The AC-3b rules probe was not re-run (per instruction).

## Coverage

- Statement / branch / function: not measured. Shell has no instrumented coverage here and no Go file is in the diff.
- Notes: the assertion-level picture is the mutation table. The new check is strong on what AC-2 names (a read-only selector must exist and every selection must be `read-only`; `--ignore-rules` must exist; widening or bypass words must be absent), across all 12 invocation lines, under sh/dash/bash and under BSD awk and mawk. It is blind to *where* in the argv an option sits (G-1) and to anything its tokenizer does not parse the way a shell does (G-2 to G-5).

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| none | - | - | - |

No test failed in any run. No flake observed (the known doctor/watcher timing tests did not fail in the uncached Go run).

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #184: codex invocations without `</dev/null`, with an alias-injected `-m`, or with drifted model/effort fallbacks | still guarded | 132/132 includes the five #184 token checks and the fallback-value checks for all four faces |
| AC-8: "Reviewer status: incomplete" documented in 4 cross-review faces | still guarded | 4 PASS lines |
| AC-2 red cases: removing the read-only selector or `--ignore-rules` from any single site | goes red as required | mutation table, first rows (S, T, P) |
| Skill/template parity suites (`test-check-skill-sync.sh`, `test-sync-skills.sh`, `test-template-purity.sh`, `test-hook-wiring.sh`) | green | full run, `docs/evidence/verify-2026-10-03-023116.log` |

## Test gaps

1. G-1 (recommend closing in this cycle). Codex 0.154.0 discards the root-level `-c` set when any `-c` follows `exec`. `check_sandbox_tokens` only asks that `sandbox_mode=read-only` appears somewhere. A rule that no `-c` / `--config` token may follow the `exec` token (the awk output already keeps argument order) catches all `late-c-*` mutants and also keeps `-c model_reasoning_effort=...` effective. HEAD's 12 lines are correct (every `-c` precedes `exec`; real run header `sandbox: read-only`, `reasoning effort: xhigh`).
2. G-2 to G-5 are tokenizer-versus-shell gaps. They need an edit that is hard to make by accident. The plan lists `$(...)` and unquoted variable expansion as known; the confirmed set is larger (continuation lines, non-`command codex` second invocations, `#`, backticks, `${}`, braces, `$'..'`). Suggest listing them in the test's header comment, or rejecting the characters the tokenizer does not model (`$`, backtick, `{`, `(`, `#`, trailing `\`) on invocation lines apart from the known `$BASE` and `${RALPH_...}` forms.
3. G-6: MCP and `notify` through `-c` are outside the allowlist by design (tech-debt (a)). `notify` honoring was not verified.
4. Not run: the AC-3b rules probe (explicitly excluded); a credentialed turn that exercises `notify`.
5. busybox ash and zsh as the suite's own interpreter were not run (macOS sh = bash posix, dash, and bash 5.2 on Linux were).

## Verdict

- Pass: yes. Full-scope `run-test.sh` (32 shell files + 8 Go packages) green; `test-codex-exec-invocation.sh` 132/132 under macOS sh and dash and under ubuntu:24.04 sh, dash, bash with mawk, 3 runs each; no failure, no flake. AC-2 holds as written (removing the read-only selector or `--ignore-rules` from any single site goes red; 41 of 93 mutations caught identically on BSD awk and mawk). The real Step 4 line at HEAD runs with `sandbox: read-only`, `codex rc=0`, non-empty `-o`.
- Fail: none.
- Blocked: none.
- Open item: G-1 is a real hole in a security guard and a realistic edit. Recommend one "no `-c` after `exec`" assertion in this cycle (the `late-c-*` mutants are its red/green script); G-2 to G-6 can be recorded as known gaps. Tests do not block `/pr`.
