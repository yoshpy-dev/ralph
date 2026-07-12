# unify-permission-mode

- Status: Approved (user selected Option A + env-priority fix, 2026-07-12)
- Owner: Claude Code
- Date: 2026-07-12
- Related request: Unify RALPH_PERMISSION_MODE default to `bypassPermissions` across shell/toml/Go; make `ralph run` env exports honor the documented env > toml priority
- Related issue: N/A
- Type: fix
- Branch: fix/unify-permission-mode-default

## Objective

Remove the entry-point-dependent permission posture: today shell paths default
to `bypassPermissions` while the Go binary `ralph run` exports the toml/Go
default `auto`, and `run.go` exports several values unconditionally so a
user's env var loses to ralph.toml — contradicting the documented priority
(env > toml > default; recorded as tech-debt in PR #119).

## Scope

1. **Default unification to `bypassPermissions`** (user decision, Option A):
   - `templates/base/ralph.toml` `[pipeline] permission_mode = "auto"` →
     `"bypassPermissions"`, with a short comment: unattended Loop runs cannot
     answer permission prompts; safety is provided by deterministic hooks
     (pre_bash_guard, commit-msg-guard), worktree isolation, and verify/test
     gates; set `auto` here or via env to run more conservatively.
   - `internal/config/config.go` `Default()` PermissionMode `"auto"` →
     `"bypassPermissions"`; adjust the Load() backfill only if it references
     the literal (it backfills from Default(), so no extra change expected).
   - `scripts/ralph-config.sh` already `bypassPermissions` — no change.
2. **Env-priority fix in `internal/cli/run.go` `runPipeline`**:
   - `RALPH_MODEL`, `RALPH_EFFORT`, `RALPH_PERMISSION_MODE` (lines ~58-62):
     unconditional `append` → `appendEnvIfMissing` (env > toml > default; no
     CLI flags exist for these).
   - `RALPH_MAX_ITERATIONS`, `RALPH_MAX_PARALLEL` (lines ~63-72): these have
     CLI flags. Honor documented priority CLI > env > toml: when the flag was
     explicitly passed (value != 0), export unconditionally (CLI wins over
     env); when not passed, resolve toml default and export via
     `appendEnvIfMissing` (env wins over toml).
3. **Tests** (`internal/cli/run_env_test.go`, `internal/config/config_test.go`):
   - defaults now `bypassPermissions` (config Default/Load + template toml
     round-trip via TestLoad_TemplateRalphToml if it asserts permission_mode);
   - env override wins under `ralph run` for MODEL/EFFORT/PERMISSION_MODE;
   - MAX_*: flag beats env; env beats toml when flag absent.
4. **Docs**:
   - `docs/recipes/ralph-loop.md` (+ template mirror): simplify the
     RALPH_PERMISSION_MODE row — one default (`bypassPermissions`) for all
     entry points; conservative override via env or ralph.toml.
   - `docs/tech-debt/README.md`: mark the permission-mode divergence row
     RESOLVED (existing strike-through convention, commit ref).
   - `.claude/rules/model-routing.md` untouched (no model semantics change).

## Non-goals

- No change to `/cross-review`'s inline `--permission-mode auto` for the
  claude reviewer (independent, deliberately conservative read-mostly seat).
- No change to hooks or permission enforcement itself.
- No renaming or new knobs.

## Assumptions

- `bypassPermissions` and `auto` are both accepted by `claude -p
  --permission-mode` (both already in production use here).
- Scaffolded projects inherit the new template default via `ralph upgrade`;
  operators preferring `auto` set it in ralph.toml or env (documented).

## Affected areas

- `templates/base/ralph.toml`, `internal/config/config.go`,
  `internal/cli/run.go`, `internal/cli/run_env_test.go`,
  `internal/config/config_test.go`, `docs/recipes/ralph-loop.md` (+ template
  mirror), `docs/tech-debt/README.md`

## Design decisions

1. **`bypassPermissions` as the single default** — user decision (Option A):
   unattended `claude -p` loops cannot answer prompts; a stalled/failed slice
   is the common failure, and ralph's guardrails are deterministic (hooks,
   gates, worktrees), not interactive.
2. **Priority repair matches the documented contract** (ralph-config.sh
   header "CLI argument > environment variable > default"; loop-vars comment
   in run.go already says "env > TOML > default") rather than inventing new
   semantics.
3. Critical forks: None remaining (resolved by user).

## Acceptance criteria

- [ ] AC1: All three sources state `bypassPermissions`
  (`grep` evidence: ralph-config.sh, templates/base/ralph.toml, config.go
  Default()); Load() backfill yields it when the toml key is absent.
- [ ] AC2: `ralph run` with pre-set env `RALPH_PERMISSION_MODE=auto` (or any
  value) exports the env value, not the toml value — asserted in
  run_env_test; same for RALPH_MODEL / RALPH_EFFORT.
- [ ] AC3: MAX_ITERATIONS/MAX_PARALLEL: explicit CLI flag wins over env;
  without the flag, env wins over toml — asserted in run_env_test.
- [ ] AC4: Recipe row shows the single unified default in both copies;
  tech-debt row marked RESOLVED.
- [ ] AC5: `go test ./...`, full `./scripts/run-test.sh < /dev/null`, and
  `./scripts/run-verify.sh < /dev/null` pass; check-sync/check-skill-sync
  pass.

## Implementation outline

Single slice (7 files): toml + Go defaults → run.go priority →
tests → docs → gates → commit.

## Verify plan

- Static: run-static-verify; AC1/AC4 grep evidence; doc drift: recipe vs new
  behavior, model-routing.md precedence statements unaffected.
- Evidence: docs/reports/verify-2026-07-12-unify-permission-mode.md.

## Test plan

- Unit: run_env_test (env-wins cases ×3 vars; MAX_* flag/env/toml matrix),
  config_test (default + backfill + template round-trip).
- Regression: full shell glob + go test.
- Edge: empty env value (`RALPH_PERMISSION_MODE=`) — appendEnvIfMissing
  treats a present-but-empty key as present; document expected behavior in
  the test (present key wins, exporting nothing extra).
- Evidence: docs/reports/test-2026-07-12-unify-permission-mode.md.

## Risks and mitigations

- **Scaffolded projects silently gain broader permissions** on upgrade →
  called out in the toml comment + PR body; conservative operators set
  `auto` (now honored from env too).
- **Someone relied on toml beating env under `ralph run`** → contradicted
  the documented contract everywhere; PR body notes the change explicitly.

## Rollout or rollback notes

- Rollback: revert PR, or set `[pipeline] permission_mode = "auto"` /
  `RALPH_PERMISSION_MODE=auto` (now honored from both entry points).

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (fix/unify-permission-mode-default)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
