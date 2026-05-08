# Verify report (cycle 4): Ralph Loop Codex driver (Phase 2)

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Verifier: `verifier` subagent (Claude Opus 4.7, 1M context)
- Branch: `feat/44/ralph-loop-codex-driver`
- Cycle: 4 (post-fix re-verify of cycle-3 cross-review P2 #1/#2/#3, cap raised 3 → 4)
- Cycle-1 baseline: `docs/reports/verify-2026-05-08-ralph-loop-codex-driver.md`
- Cycle-2 baseline: `docs/reports/verify-2026-05-08-ralph-loop-codex-driver-cycle2.md`
- Cycle-3 baseline: `docs/reports/verify-2026-05-08-ralph-loop-codex-driver-cycle3.md`
- Evidence: `docs/evidence/verify-2026-05-08-ralph-loop-codex-driver-cycle4.log`

## Scope (cycle 4)

Cycle 4 verifies a single delta commit on top of cycle-3:

| Commit | Subject |
| --- | --- |
| `f735299` | fix: address cycle-3 cross-review P2 findings (3) — TTY-guard for fake-CLI stubs (#1), checkLoopDriver fail when codex missing (#2), env-aware sandbox/approval/reviewer display via `pick` closure (#3); +2 focused tests; cycle-count.json bumped 3 → 4 |

The full AC-1..AC-12 walk from cycles 1/2/3 is preserved; cycle-4 deltas are flagged with **[Δ cycle-4]**. No other commit was made between cycle-3's `12e7698` and cycle-4's `f735299`.

## Cycle-4 fix verification (focus areas)

### 1. fake-CLI stubs are TTY-safe (P2 #1)

| Check | Result | Evidence |
| --- | --- | --- |
| `tests/fixtures/cli-stubs/codex` skips stdin drain on TTY | **Yes** | Lines 41–49: `if [ -t 0 ]; then stdin_buf=""; else stdin_buf="$(head -c 4096 \|\| true)"; fi`. Comment block at :40-44 names the bug ("interactive `tests/test-ralph-cli-driver.sh` run would hang ... waiting for the developer to type EOF") and the resolved finding (`#44 cycle-3 cross-review P2 #1`). |
| `tests/fixtures/cli-stubs/claude` skips stdin drain on TTY (symmetric) | **Yes** | Lines 40–47: identical TTY-guard pattern; comment cross-references the codex stub for the same defense. |
| Non-TTY callers (CI, sub-shells, `< /dev/null`) unaffected | **Yes (likely but unverified by this static pass)** | The `else` branch preserves the original `head -c 4096` drain verbatim, so any non-TTY input (pipe, redirected file, `/dev/null`) takes the same path it did before. Behavioural confirmation belongs to `/test`. |
| Both stubs in lock-step on the new pattern | **Yes** | The TTY guard appears in both files with the same block shape and comment style. Stubs are root-only by design (see §"Mirror parity") so no template mirror to keep aligned. |
| `sh -n` POSIX syntax check | **Pass** | `sh -n tests/fixtures/cli-stubs/{codex,claude}` exits 0 on both. |
| `shellcheck -S warning` | **Pass** | Empty output for both stubs — no new SC warnings introduced. |

### 2. `checkLoopDriver` fails when codex is missing (P2 #2)

| Check | Result | Evidence |
| --- | --- | --- |
| Function returns `fail` when `effective == "codex"` and `exec.LookPath("codex")` fails | **Yes** | `internal/cli/doctor.go:463-468`: `if effective == "codex" { if _, err := exec.LookPath("codex"); err != nil { r.Status = "fail"; r.Detail = fmt.Sprintf("%s (source: %s) — codex CLI not found in PATH; \`ralph run\` preflight will fail", effective, source); return r } }`. |
| Detail message names the source so operator can see how driver=codex got selected | **Yes** | The fail-detail format string includes both `effective` and `source`, so an operator who set `RALPH_LOOP_DRIVER=codex` and forgot to install codex sees `codex (source: env) — codex CLI not found in PATH ...` rather than a bare "missing codex" message. |
| Fail message is operator-actionable (names what will break next) | **Yes** | "`ralph run` preflight will fail" tells the operator exactly which downstream step blocks, matching the cross-review finding's stated motivation. |
| Pass-path for `effective == "claude"` unchanged | **Yes** | The codex-only fast-fail block guards `effective == "codex"`. The claude path falls through to the `r.Status = "pass"` branch. |
| New regression test pins the fail behaviour | **Yes** | `internal/cli/doctor_loop_test.go:97-111`, `TestCheckLoopDriver_FailsWhenCodexMissing`: empties PATH via `t.Setenv("PATH", t.TempDir())`, asserts `r.Status == "fail"` and `r.Detail` contains `"codex CLI not found"`. |
| Existing `TestCheckLoopDriver_PriorityAndSource` not broken by the new fast-fail | **Yes** | Lines 65-75: the "codex-driven" cases now plant a sham `codex` script on PATH (`t.TempDir()`) so the `exec.LookPath` succeeds and the test reaches the detail-string assertion as before. The `"default when nothing set"` and `"totally empty TOML falls back to default"` cases land in the claude path and bypass the lookup entirely. `go test ./internal/cli/...` (via `run-static-verify.sh`) reports `ok` — see §"Static analysis". |
| `os/exec` import already present, no new import added unnecessarily | **Yes** | `internal/cli/doctor.go:10` already imported `os/exec` for unrelated checks; the new `exec.LookPath` call reuses it. |

### 3. Env-aware sandbox/approval/reviewer display via `pick` closure (P2 #3)

| Check | Result | Evidence |
| --- | --- | --- |
| Driver, sandbox, approval, reviewer all resolved through identical priority logic | **Yes** | `doctor.go:446-461`: the `pick` closure encapsulates `getenv → tomlVal → defaultVal` priority and is invoked four times — once per knob (`RALPH_LOOP_DRIVER`, `RALPH_CODEX_SANDBOX`, `RALPH_CODEX_APPROVAL_POLICY`, `RALPH_CLAUDE_REVIEWER_MODEL`). Refactor preserves the AC-6 "env > TOML > default" contract. |
| Old switch-statement semantics preserved exactly | **Yes (logical walk-through)** | Old code: case `envVal != ""` → ("env"); case `tomlVal != "" && tomlVal != defaultVal` → ("toml"); case `tomlVal != ""` (matches default) → ("toml") with comment; default → ("default"). New code: env first; non-empty toml → "toml" (single case, comment retained explaining matched-default behaviour); else → "default". The two branches that the old code split on "tomlVal vs defaultVal" produced the same `"toml"` source — collapsing them into one case is a pure refactor with the same observable behaviour. |
| Effective sandbox value reflects env override in display | **Yes** | `doctor.go:473-474`: detail format string now consumes `sandbox` (the `pick` result) instead of `cfg.Loop.CodexSandbox`. With `RALPH_CODEX_SANDBOX=danger-full-access`, the display reads `sandbox: danger-full-access` instead of the silent TOML/default. |
| Effective approval value reflects env override | **Yes** | Same line: `approval` substituted for `cfg.Loop.CodexApprovalPolicy`. |
| Effective reviewer value reflects env override | **Yes** | Same line: `reviewer` (claude/`<model>`) substituted for `cfg.Loop.ClaudeReviewerModel`. The cycle-3 finding only called out sandbox + approval, but the symmetry has been extended to reviewer-model — a quiet over-fix that closes the same operator-confusion class for `RALPH_CLAUDE_REVIEWER_MODEL`. |
| New regression test pins env-priority display | **Yes** | `internal/cli/doctor_loop_test.go:117-148`, `TestCheckLoopDriver_EnvOverridesShownInDetail`: sets `RALPH_CODEX_SANDBOX=danger-full-access` + `RALPH_CODEX_APPROVAL_POLICY=never`, plants a sham `codex` on PATH so the function does not short-circuit to fail, then asserts both `sandbox: danger-full-access` and `approval: never` appear in the detail string. |

### 4. Six-case `pick` closure walk (AC-6 stress test)

The plan's AC-6 review asked for a six-case proof that the `pick` closure produces the same `(value, source)` triples as the old switch logic. Walked logically against the new code:

| # | Case | env | toml | default | Old result | New result | Match |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | claude default (Default() loaded) | `""` | `"claude"` | `"claude"` | `("claude","toml")` (matched-default branch) | `("claude","toml")` (non-empty toml branch) | ✓ |
| 2 | codex default (TOML alone) | `""` | `"codex"` | `"claude"` | `("codex","toml")` | `("codex","toml")` | ✓ |
| 3 | env override of driver | `"codex"` | `"claude"` | `"claude"` | `("codex","env")` | `("codex","env")` | ✓ |
| 4 | env override of sandbox | sandbox env=`danger-full-access`, toml=`workspace-write` | `"workspace-write"` | `"workspace-write"` | n/a (old code did not honour env for sandbox) | `("danger-full-access","env")` → detail shows `sandbox: danger-full-access` | ✓ (intended fix) |
| 5 | env override of approval | approval env=`never`, toml=`on-failure` | `"on-failure"` | `"on-failure"` | n/a (old code did not honour env for approval) | `("never","env")` → detail shows `approval: never` | ✓ (intended fix) |
| 6 | missing codex fail | env=`codex`, toml=anything, codex absent on PATH | any | `"claude"` | passed silently with detail showing codex (cross-review P2 #2 root cause) | `fail` with detail `... — codex CLI not found in PATH; \`ralph run\` preflight will fail` | ✓ (intended fix) |

All six cases produce the right answer. Cases 1–3 prove the refactor is non-regressive; cases 4–6 prove the cycle-4 fixes land.

### 5. Cycle counter rationale recorded

Cycle counter raised 3 → 4 in `.harness/state/standard-pipeline/cycle-count.json` per the commit message. (Not re-read in this verify pass — cycle-3 verify already documents the operator-evidence pattern; the new entry is a continuation of that audit trail.)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| **AC-1** `scripts/ralph-cli-driver.sh` exposes `run_agent`; fake-CLI assertions cover argv/stdin/cwd/output | **Met** | Wrapper unchanged in cycle 4. **[Δ cycle-4]** Stubs gained TTY-guard but the wire-level argv/stdin/cwd contract is unaffected. The non-TTY branch (which the test harness exercises via subshells / piped stdin) keeps the original `head -c 4096` capture verbatim. |
| **AC-2** `RALPH_LOOP_DRIVER=codex --preflight --dry-run` green; codex Probe 5 active; claude branch keeps `claude_md_readable`+`json_output_format` | **Met** | No cycle-4 change in this surface. |
| **AC-3** `<log>.json` synthesised for codex; sidecar contract unchanged | **Met** | No cycle-4 change. |
| **AC-4** `internal/cli/run.go` propagates `cfg.Loop.*` to env only when env unset | **Met** | No cycle-4 change. `go build ./...` / `go vet ./...` clean. |
| **AC-5** cross-review dispatcher: `driver=claude → codex exec review`, `driver=codex → claude -p` adversarial; reviewer/driver fields recorded | **Met** | No cycle-4 change to the dispatcher or its mirrored docs. The 6 doc-drift sites flagged in cycle-3 were resolved in cycle-3's `/sync-docs` (commit `564d971`); a fresh repo-wide grep confirms no `permission-mode plan` literals remain outside `docs/reports/`, `docs/evidence/`, `docs/plans/` (point-in-time snapshots), and `.claude/agent-memory/` (description text in feedback memos). See evidence log §"permission-mode plan grep". |
| **AC-6** `ralph status` / `ralph doctor` show effective driver with source | **Met (stronger)** | **[Δ cycle-4]** Implementation now matches AC-6 text more strictly than before. Per the AC-6 walk above, all four loop knobs (`Driver`, `CodexSandbox`, `CodexApprovalPolicy`, `ClaudeReviewerModel`) now resolve through identical env > TOML > default priority. The fast-fail when codex is missing converts a previously-silent inconsistency ("doctor pass + run fail") into an explicit, operator-actionable fail. Two new tests pin the new behaviour. |
| **AC-7** Mirrored skills in lock-step | **Met** | `./scripts/check-skill-sync.sh` → `13 skill(s) in lock-step`. Spot-checks on `loop/SKILL.md` and `cross-review/SKILL.md` mirrors all `cmp -s` IDENTICAL. |
| **AC-8** `docs/recipes/ralph-loop.md` Codex driver section | **Met** | No cycle-4 change. The cycle-3 drift (legacy `--permission-mode plan` mention) was already cleaned in commit `564d971`. |
| **AC-9** `./scripts/run-verify.sh` green; `tests/test-ralph-cli-driver.sh` green | **Likely met (static portion verified, behavioural deferred to /test)** | `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` → `All verifiers passed.` (which exercises gofmt, go vet, go build, `internal/cli` go test cache, check-skill-sync, check-pipeline-sync, check-sync). `bash -n tests/test-ralph-cli-driver.sh` syntactically clean. Behavioural test execution is `/test`'s scope. |
| **AC-10** `internal/config/config_test.go` covers `[loop] driver` allowlist | **Met** | No cycle-4 change. |
| **AC-11** Codex CLI walkthrough | **Deferred** | Same status as cycles 1+2+3. Operator/`/test` work, not a `/verify` blocker. |
| **AC-12** Backwards compat | **Met** | The cycle-4 doctor changes only activate when `effective == "codex"`. Default-driver users (env unset, TOML unset or `driver = "claude"`) hit the same code path as before, with the same detail format `claude (source: <s>)`. The TTY-guard in the stubs only changes behaviour for an operator running the test in a terminal directly — no production / CI path is touched. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `gofmt -l ./cmd ./internal` | **Pass** | empty output. |
| `go vet ./...` | **Pass** | empty output. |
| `go build ./...` | **Pass** | empty output. |
| `shellcheck -S warning tests/fixtures/cli-stubs/codex tests/fixtures/cli-stubs/claude` | **Pass** | empty output — no new warnings introduced by the TTY-guard. |
| `shellcheck -S warning scripts/ralph-pipeline.sh scripts/ralph-cli-driver.sh scripts/ralph-orchestrator.sh scripts/ralph-config.sh` | **Pass with pre-existing SC3045 in `ralph-orchestrator.sh:549`** | Identical finding to cycles 1+2+3; not a cycle-4 regression. |
| `sh -n tests/fixtures/cli-stubs/codex` | **Pass** | POSIX syntax valid. |
| `sh -n tests/fixtures/cli-stubs/claude` | **Pass** | POSIX syntax valid. |
| `sh -n scripts/ralph-pipeline.sh` | **Pass** | unchanged in cycle 4. |
| `./scripts/check-skill-sync.sh` | **Pass** | `13 skill(s) in lock-step`. |
| `./scripts/check-sync.sh` | **Pass** | `IDENTICAL: 145, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3`. |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | **Pass** | All verifiers green; `internal/cli` go test cached `ok`; check-pipeline-sync green for all 8 referenced files. |

## Mirror parity (skills + stubs)

| Surface | Result | Notes |
| --- | --- | --- |
| `.claude/skills/loop/SKILL.md` ↔ `.agents/skills/loop/SKILL.md` | **IDENTICAL** | `cmp -s` clean. |
| `.claude/skills/loop/SKILL.md` ↔ `templates/base/.claude/skills/loop/SKILL.md` | **IDENTICAL** | `cmp -s` clean. |
| `.agents/skills/loop/SKILL.md` ↔ `templates/base/.agents/skills/loop/SKILL.md` | **IDENTICAL** | `cmp -s` clean. |
| `.claude/skills/cross-review/SKILL.md` ↔ `.agents/skills/cross-review/SKILL.md` | **IDENTICAL** | `cmp -s` clean. |
| `.claude/skills/cross-review/SKILL.md` ↔ `templates/base/.claude/skills/cross-review/SKILL.md` | **IDENTICAL** | `cmp -s` clean. |
| `tests/fixtures/cli-stubs/{codex,claude}` template mirror | **N/A by design** | `find templates -path "*tests/fixtures*"` returns no hits. The plan body explicitly scopes `tests/` as root-only ("`tests/` is root-only by design"), so the cycle-4 stub edits do not require any template-side companion change. `check-sync.sh` confirms `IDENTICAL: 145 / DRIFTED: 0 / TEMPLATE_ONLY: 10` (the 10 are the well-known scaffold-only files like `docs/reports/.gitkeep`). |

## Documentation drift

None introduced by cycle 4. Specifically:
- The plan body was not edited (cycle-4 commit only touched code + stubs + tests). Per the plan-AC-checklist-lag pattern in agent memory, that is acceptable.
- The progress checklist on the plan still reads `cap (3 of 3)`; the actual state is `cap (4 of 4)` with cycle-4 raise. This is plan-checklist drift, not a behaviour drift; flagging it as informational and handing to `/sync-docs`.
- The 6 sites flagged in cycle-3 verify (`loop/SKILL.md` ×4, `docs/recipes/ralph-loop.md` ×2, Test 6b) were all cleaned in commit `564d971` (cycle-3 sync-docs). A fresh repo-wide `grep -RIn "permission-mode plan"` finds the literal only inside agent-memory feedback memos and verify/cross-review/walkthrough reports — both are out-of-scope point-in-time documents.

## Coverage gaps (handed off to /test)

1. **AC-9 behavioural** — `tests/test-ralph-cli-driver.sh` 48+ assertions still need an end-to-end run. With the TTY-guard in place, an interactive direct-test invocation will no longer hang, but `/test` should confirm that the non-TTY (CI) branch still captures stdin into the call log as before.
2. **AC-11 walkthrough** — Real Codex driver walkthrough still not recorded.
3. **`go test ./internal/cli/...`** — the two new tests (`TestCheckLoopDriver_FailsWhenCodexMissing`, `TestCheckLoopDriver_EnvOverridesShownInDetail`) and the augmented `TestCheckLoopDriver_PriorityAndSource` need an explicit non-cached run. `run-static-verify.sh` reports the package as `ok (cached)`; a fresh invocation by `/test` would produce the strongest signal.

## Verdict

- **Pass** — all three cycle-3 cross-review P2 findings are closed by cycle-4. The doctor.go refactor is logically equivalent to the prior switch statement on the (Driver) axis, and is a strict generalisation on the (Sandbox / Approval / Reviewer) axes. The fast-fail on missing codex converts a known operator-confusion mode into an explicit `fail` with an actionable message. The fake-CLI stubs gained a symmetric TTY-guard that makes `tests/test-ralph-cli-driver.sh` safe to run interactively without altering CI/sub-shell behaviour.
- **Verified**: AC-1 (TTY-guard preserves wire contract), AC-2..AC-5 (no cycle-4 change), AC-6 (now strictly stronger — six-case walk + 2 new tests pin the new behaviour), AC-7 (lock-step proven), AC-8 (no change), AC-10 (no change), AC-12 (backwards-compatible).
- **Likely but unverified (handed to /test)**: AC-9 behavioural — 48+ assertions; the two new doctor tests need a non-cached `go test`.
- **Deferred**: AC-11 — Codex walkthrough.
- **Documentation drift (handed to /sync-docs)**: Plan progress-checklist `cap (3 of 3)` line should advance to `cap (4 of 4)` and reference `cycle-3 cross-review P2 #1/#2/#3` as the raise reason. Informational only — not contract-breaking.

**Pipeline decision**: **Continue.** Verdict is `pass` for the cycle-4 delta. The plan-checklist drift is MEDIUM doc-drift only and should be cleaned up in `/sync-docs` of the same pipeline run.

## Smallest additional check that would raise confidence

A focused `go test -run 'TestCheckLoopDriver' ./internal/cli/...` (non-cached) would directly exercise:

```go
TestCheckLoopDriver_PriorityAndSource          // 4 sub-cases — refactor non-regression
TestCheckLoopDriver_FailsWhenCodexMissing      // P2 #2 fix lock
TestCheckLoopDriver_EnvOverridesShownInDetail  // P2 #3 fix lock
```

That single subset of `/test` would convert AC-6's six-case walk from "logical proof + cached test" to "logical proof + fresh test execution" and is the highest-leverage behavioural confirmation for the cycle-4 delta. It is also the only piece of behavioural evidence that uses the new code paths — the TTY-guard's "skipped read" path is harder to exercise in CI (which is by definition non-TTY) and is best confirmed via the existing manual-direct-test guidance.
