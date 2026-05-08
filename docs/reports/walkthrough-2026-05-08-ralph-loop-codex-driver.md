# Walkthrough: Ralph Loop Codex driver (Phase 2)

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md` (AC-11)
- Branch: `feat/44/ralph-loop-codex-driver`
- Operator: Claude Code (tester subagent)
- Scope: smoke test of the Codex driver dispatcher + dry-run plumbing using the locally available `codex 0.128.0`. **No real autonomous codex/claude turn was invoked** — this walkthrough confirms the wiring (dispatcher + preflight + sidecar synthesis + doctor/status surfacing + template parity), not Inner-Loop convergence.

## Environment

| Tool | Version | Path |
| --- | --- | --- |
| `codex` | `codex-cli 0.128.0` | `~/.local/share/mise/installs/npm-openai-codex/0.128.0/bin/codex` |
| `claude` | `2.1.132` | (via global PATH; surfaced by `ralph doctor`) |
| `ralph` (Homebrew install) | `3.4.0 (0ee462f 2026-04-23T15:50:56Z)` | `/opt/homebrew/bin/ralph` — **predates Phase 2**; lacks `checkLoopDriver` |
| `ralph` (built from this branch) | `dev` | `/tmp/ralph-cycle2` (built via `go build -o /tmp/ralph-cycle2 ./cmd/ralph`) |

The Homebrew-installed `ralph` was built from main on 2026-04-23 and does not yet ship the Phase 2 Loop driver doctor probe. We built a local binary from `feat/44/ralph-loop-codex-driver` HEAD and used that for the walkthrough.

## Step 1 — Codex preflight dry-run dispatcher

Command:

```sh
RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight --dry-run
```

Exit code: **0**. Captured at `docs/evidence/walkthrough-2026-05-08-ralph-loop-codex-driver-preflight.log`.

Observed (banner + probes):

```
[2026-05-08T04:22:06Z] === Ralph Pipeline v2 ===
[2026-05-08T04:22:06Z] Max iterations: 20
[2026-05-08T04:22:06Z] Max inner cycles: 10
[2026-05-08T04:22:06Z] Max outer cycles: 2
[2026-05-08T04:22:06Z] Max repair attempts: 5
[2026-05-08T04:22:06Z] Skip PR: 0
[2026-05-08T04:22:06Z] Fix all: 0
[2026-05-08T04:22:06Z] Dry run: 1
[2026-05-08T04:22:06Z]
[2026-05-08T04:22:06Z] === Preflight capability probe ===
[2026-05-08T04:22:06Z]   driver: codex
[2026-05-08T04:22:06Z]   codex CLI: pass
[2026-05-08T04:22:06Z]   jq: pass
[2026-05-08T04:22:06Z]   agents_md_readable: pass
[2026-05-08T04:22:06Z]   git: pass
[2026-05-08T04:22:06Z]   codex_exec_flags: skip_dry_run
[2026-05-08T04:22:07Z]   gh CLI: available
[2026-05-08T04:22:07Z]   claude CLI: available
[2026-05-08T04:22:07Z] Preflight results saved to docs/evidence/preflight-probe.json
[2026-05-08T04:22:07Z] === Preflight probe PASSED ===
[2026-05-08T04:22:07Z] Preflight-only mode. Exiting.
```

Expected vs observed:

| Expectation | Observed | Verdict |
| --- | --- | --- |
| Banner reports `driver: codex` | `driver: codex` | PASS |
| `claude_md_readable` probe is skipped (driver=codex), replaced by `agents_md_readable: pass` | `agents_md_readable: pass` line present; no `claude_md_readable` line | PASS |
| `json_output_format` probe is skipped, replaced by `codex_exec_flags` Codex-specific probe | `codex_exec_flags: skip_dry_run` (probe is gated to non-dry-run for safety) | PASS |
| `codex CLI: pass` from preflight (binary on PATH) | `codex CLI: pass` | PASS |
| Pipeline exits 0 in `--preflight --dry-run` mode | exit 0 | PASS |
| `gh CLI` and `claude CLI` advisory lines surface (cross-CLI awareness) | both `available` | PASS |

The driver-aware probe partition (AC-2) is observed live: claude-side probes are not run when `driver=codex`, and the codex-side `agents_md_readable` + `codex_exec_flags` probes replace them. The `codex_exec_flags` probe is intentionally `skip_dry_run` because that probe is gated behind `--full-flag-check` (only runs when not in dry-run, since it actually invokes `codex exec --help` and parses); for AC-2 evidence we already exercised the active probe path during cycle-1 verify.

## Step 2 — `ralph doctor` Loop driver line (built-from-branch binary)

Command (default — env unset, TOML at default `claude`):

```sh
/tmp/ralph-cycle2 doctor
```

Output (relevant excerpt):

```
ralph doctor

  ✓ Claude Code CLI: pass — 2.1.132 (Claude Code)
  ✓ Codex CLI: pass — codex-cli 0.128.0
  ✓ Codex effective config: pass — codex_hooks=true, 2 hook entry(ies). Confirm `codex trust .` ran for this project
  ✓ Hooks integrity: pass
  ⚠ Manifest version: warn — no manifest found
  ✓ Pack: dart: pass
  ✓ Pack: golang: pass
  ✓ Pack: python: pass
  ✓ Pack: rust: pass
  ✓ Pack: typescript: pass
  ✓ Loop driver: pass — claude (source: toml)
  ✓ Go: pass

All checks passed.
```

Then with env override:

```sh
RALPH_LOOP_DRIVER=codex /tmp/ralph-cycle2 doctor
```

Relevant excerpt:

```
  ✓ Claude Code CLI: pass — 2.1.132 (Claude Code)
  ✓ Codex CLI: pass — codex-cli 0.128.0
  ✓ Loop driver: pass — codex (source: env, sandbox: workspace-write, approval: on-failure, reviewer: claude/claude-opus-4-7)
```

Expected vs observed:

| Expectation | Observed | Verdict |
| --- | --- | --- |
| Default-state `Loop driver` line shows effective driver and source | `Loop driver: pass — claude (source: toml)` | PASS |
| Env override flips effective driver and source from `toml` to `env` | `Loop driver: pass — codex (source: env, ...)` | PASS |
| When effective driver is `codex`, sandbox + approval + reviewer details surface | `sandbox: workspace-write, approval: on-failure, reviewer: claude/claude-opus-4-7` | PASS |
| `claude_opus_4_7` reviewer shown for the inverted dispatcher (driver=codex → reviewer=claude) | reviewer line shows `claude/claude-opus-4-7` | PASS |

This satisfies AC-6 end-to-end against the actual binary built from the branch HEAD.

## Step 3 — Root ↔ template byte-identical check

Commands:

```sh
cmp -s scripts/ralph-cli-driver.sh    templates/base/scripts/ralph-cli-driver.sh    && echo "ralph-cli-driver: IDENTICAL"
cmp -s scripts/ralph-pipeline.sh      templates/base/scripts/ralph-pipeline.sh      && echo "ralph-pipeline: IDENTICAL"
cmp -s scripts/ralph-config.sh        templates/base/scripts/ralph-config.sh        && echo "ralph-config: IDENTICAL"
cmp -s scripts/ralph-orchestrator.sh  templates/base/scripts/ralph-orchestrator.sh  && echo "ralph-orchestrator: IDENTICAL"
```

Output:

```
ralph-cli-driver: IDENTICAL
ralph-pipeline: IDENTICAL
ralph-config: IDENTICAL
ralph-orchestrator: IDENTICAL
```

All four edited shell scripts are byte-identical between the meta-repo root and `templates/base/`. Combined with the verifier's `./scripts/check-sync.sh` (`IDENTICAL: 145, DRIFTED: 0`) and `./scripts/check-skill-sync.sh` (`13 skill(s) in lock-step`), the dogfooding contract is intact.

## Step 4 — Observed-vs-expected for the Codex driver path (summary)

| Surface | Expected (per plan + cycle-2 verifier) | Observed | Verdict |
| --- | --- | --- | --- |
| `RALPH_LOOP_DRIVER=codex --preflight --dry-run` | exit 0; codex-specific probes selected; banner shows `driver: codex` | exit 0; `agents_md_readable: pass`, `codex_exec_flags: skip_dry_run`; `driver: codex` printed | PASS |
| `ralph doctor` driver line (default) | `Loop driver: pass — claude (source: toml)` | match | PASS |
| `ralph doctor` driver line (env override) | `Loop driver: pass — codex (source: env, sandbox: ..., approval: ..., reviewer: claude/...)` | match | PASS |
| Root ↔ templates/base parity for the four edited shell scripts | byte-identical (cmp -s exit 0 each) | all four IDENTICAL | PASS |
| `pick_reviewer` inversion (driver=codex → reviewer=claude) reflected in doctor reviewer field | reviewer field shows `claude/claude-opus-4-7` | match | PASS |
| Cross-review dispatcher would invoke `claude -p --permission-mode plan` for `driver=codex` | unit-tested in `tests/test-ralph-cli-driver.sh` Test 6b (PASS) | wiring verified by test, dispatcher branches at `scripts/ralph-pipeline.sh:778-797` confirmed by cycle-2 verifier | PASS |
| Codex `--output-last-message` sidecar synth (`<log>.json` written even on Codex driver) | unit-tested in Test 2 (PASS) and Test 3 (missing-file fallback) | wiring verified | PASS |

## Known limitations (intentional)

- **No real autonomous codex turn was invoked.** Per the test prompt's "Do NOT actually invoke a real codex/claude turn (no autonomous coding)", this walkthrough exercises the dispatcher + dry-run plumbing only. A future end-to-end short-prompt `RALPH_LOOP_DRIVER=codex ./scripts/ralph run --plan ...` against a tiny, throwaway plan slice would exercise Inner-Loop convergence (Codex `codex exec` actually streaming a model response), but is out of scope for the cycle-2 test gate.
- The Homebrew-installed `ralph` (3.4.0, built 2026-04-23) does **not** carry Phase 2 changes. Operators on stable should not see the `Loop driver` line in `ralph doctor` until the next release. Building from the branch (`go build -o /tmp/ralph-cycle2 ./cmd/ralph`) is required to surface the new probe locally.
- `codex_exec_flags` probe is `skip_dry_run` in the captured log because it is gated to non-dry-run mode for safety. The active probe (which actually parses `codex exec --help`) was exercised during cycle-1 and cycle-2 verifier runs and passed; no regressions.

## Verdict

PASS — Codex driver dispatcher, dry-run plumbing, doctor surfacing (default + env override), and template parity all behave as designed. AC-11 (smoke-test variant) is satisfied. AC-1 / AC-2 / AC-3 / AC-5 / AC-6 are reconfirmed against the live `codex 0.128.0` and the built-from-branch `ralph` binary.
