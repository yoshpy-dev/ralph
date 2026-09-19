# Verify report: codex-agmsg-writable-root

- Date: 2026-09-19
- Plan: `docs/plans/active/2026-09-19-codex-agmsg-writable-root.md` (issue #164)
- Verifier: `verifier` subagent (Claude Code, standard flow)
- Branch: `feat/codex-agmsg-writable-root`, HEAD `cf5464a` (base `main`)
- Scope: spec compliance (AC-1..AC-8) + static analysis via `./scripts/run-static-verify.sh` (changed-language scope: Go + markdown/skills) + range secret scan. No behavioral tests run (`/test` owns that); no source/plan/doc edits made.
- Evidence: `docs/evidence/verify-2026-09-19-042648.log` (gitignored per `.gitignore:58`, same as prior verify runs in this repo)
- Pipeline cycle: 1 (of cap 2).

## Overall verdict: PASS

AC-1 through AC-7 are met, all backed by code read plus live-binary reproduction where applicable. AC-8 (`Closes #164`) is a `/pr`-time criterion, out of scope here.

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: Check appears right after the codex-slug Check | Met | `internal/cli/doctor.go:137-148` registers `checkCodexAgmsgWritableRoot` as "Check 11b" immediately after Check 11 (`checkCodexModelSlugs`), passing `cfg.Org`, `driver.ResolveAgmsgHome(cfg.Org.AgmsgHome)`, and the `doctorCodexSandboxEnv` seam. Live-binary run confirms adjacency in printed output: `"Org codex model slugs: pass — ..."` immediately followed by `"Codex sandbox (agmsg writable root): pass — ..."`. |
| AC-2: Scope-5 test matrix + Detail contents + path-element ancestor rule | Met | Every named scenario in Scope row 5 maps to a test in `internal/cli/doctor_codex_writable_root_test.go` (see table below). `TestCheckCodexAgmsgWritableRoot_CodexVerifiedNoRoots_Warn` (line 133) asserts the warn Detail contains the agmsg `db` dir, the config path, `docs/recipes/codex-seat-permissions.md`, and the `"attempt to write a readonly database"` phrase. `TestPathCovers` (line 742) and `TestCheckCodexAgmsgWritableRoot_PrefixSiblingNotCovered` (line 207) pin the path-element boundary (`/a/b` does not cover `/a/bc`; `…/db` vs `…/dbx`). |
| AC-3: full outcome matrix incl. `driver_pool`/role conditions, `AGMSG_STORAGE_PATH`, decode/type/home failures, `countFailed` exclusion | Met | `checkCodexAgmsgWritableRoot` (`internal/cli/doctor_codex_writable_root.go:558-621`) implements outcomes 1-7 in the documented order (agmsg missing → info; `driver_pool` gate → pass; resolveEnv error → info; unreadable/undecodable config or non-array `writable_roots` → info; no reason → pass; covering root → pass; otherwise → warn). It only ever sets `Status` to `"pass"`, `"warn"`, or `"info"` — never `"fail"` — and `countFailed` (`doctor.go:195-200`) only counts `"fail"`, so this Check never affects exit code (also proven live: both scratch warn and pass scenarios below exited 0). |
| AC-4: existing `runDoctor*` tests stay hermetic against the real `~/.codex/config.toml` | Met | `internal/cli/main_test.go:19-27` pins `doctorCodexSandboxEnv` in `TestMain` to a non-existent config path under `os.TempDir()`, mirroring the existing `doctorShellAliasEnv` seam. `TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault` (line 938) proves the seam is wired end to end: with the default project config it reads `pass — not needed`, independent of the developer's real `~/.codex/config.toml`. |
| AC-5: recipe + `/org` skill mentions, 3 sync gates pass | Met | `docs/recipes/codex-seat-permissions.md` and `templates/base/docs/recipes/codex-seat-permissions.md` are byte-identical (`cmp` exit 0) and both gained the same paragraph describing the Check, its two trigger conditions, and its read scope. All four `SKILL.md` mirrors (`.claude/skills/org`, `.agents/skills/org`, `templates/base/.claude/skills/org`, `templates/base/.agents/skills/org`) carry the identical added sentence (`git diff` confirms byte-for-byte match across all four). `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`. `./scripts/check-sync.sh` → `DRIFTED: 0`. `./scripts/check-template-purity.sh` → PASS. |
| AC-6: `gofmt`/`go vet`/golangci-lint clean; range secret scan exit 0 | Met (static half only — `go test` is `/test`'s job) | `./scripts/run-static-verify.sh` → `gofmt: ok`, `go vet` silent (clean), `golangci-lint run` → `0 issues.`, `staticcheck` silent (clean, confirmed installed at `~/go/bin/staticcheck`) → overall `All verifiers passed.`, exit 0. `./scripts/secret-scan.sh --range "$(git merge-base HEAD main)..HEAD"` → exit 0, no findings. `go test ./internal/cli/... -count=1` and `./scripts/run-verify.sh` are explicitly out of scope for `/verify`; the plan's Deviation notes record the implementer's own green run of both after each slice, most recently after Slice D (`33ac5a8`). |
| AC-7: evidence lines recorded in the plan match current behavior | Met | Reproduced all three lines live with a freshly built binary (see "Observational checks"); byte-for-byte match against the plan's Deviation notes (lines 109/112 for the real-config line, updated wording explicitly noted at line 112 for the scratch lines). |
| AC-8: PR body `Closes #164` | Pending `/pr` | Out of scope for `/verify`. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass (exit 0) | shellcheck / `sh -n` / `jq -e` / hook-guard checks all OK; `check-sync.sh`, `check-pipeline-sync.sh`, `check-skill-sync.sh`, `check-template-purity.sh` all OK; golang verifier: `gofmt: ok`, `go vet` clean, `golangci-lint run` → `0 issues.`; scope = changed languages (golang). Evidence: `docs/evidence/verify-2026-09-19-042648.log`. |
| `./scripts/secret-scan.sh --range "$(git merge-base HEAD main)..HEAD"` | Pass (exit 0) | History-range scan across all 10 commits on this branch (merge-base `2c511a4`); no findings. Matches the plan's fixture-hygiene risk mitigation (no `api_key=`-shaped strings in test fixtures; canary strings `CANARYLEAK123` used instead and asserted absent from Detail). |
| `./scripts/check-skill-sync.sh` | Pass | `13 skill(s) in lock-step`. |
| `./scripts/check-sync.sh` | Pass | `DRIFTED: 0`, `ROOT_ONLY: 0`; the 4 changed files (recipe + 3 skill mirrors, minus the root `.claude/skills/org` which check-sync doesn't compare — see check-skill-sync above) all land in `IDENTICAL: 158`. |
| `./scripts/check-template-purity.sh` | Pass | No meta-repo-specific references in templates. |
| `cmp` recipe root vs template | Pass | Byte-identical. |
| `staticcheck ./...` (via verify.sh) | Pass (clean, silent) | Confirmed binary present (`command -v staticcheck` → `~/go/bin/staticcheck`); a missing binary would print a "Skipping staticcheck" line, which is absent from the log. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/recipes/codex-seat-permissions.md` (+ template) | Yes | New paragraph states both trigger conditions ((a) `codex_verified = true` + edits/autonomous role, (b) `sandbox_mode = "workspace-write"` + guarded role), the `driver_pool` gate, the `AGMSG_STORAGE_PATH` override, and that only the user-level `config.toml` is read (no profiles/project config/`-c`). All match the code (`internal/cli/doctor_codex_writable_root.go:239-303`) and `internal/org/permissions.go` (`ResolvePermissionMode`, `permissionArgsForDriver`: codex `autonomous`/`edits` under `CodexVerified` get `--sandbox workspace-write`; `guarded` gets no flag and inherits the user's own `sandbox_mode`). |
| `.claude/skills/org/SKILL.md` permission-作法 codex bullet (4 mirrors) | Yes | Identical added sentence across all four mirror files, same two conditions and read-scope caveat as the recipe. |
| Doc comments in `internal/cli/doctor_codex_writable_root.go` | Yes | The `checkCodexAgmsgWritableRoot` doc comment's 7-outcome list matches the actual `if`/`switch` chain read line-by-line; the `codexSeatModesPossible` comment's description of the "effective default from a Roles-nilled copy" matches the NEW-2 fix (`orgCfg.Permissions.Roles = nil` before calling `ResolvePermissionMode`); the trailing paragraph about `runDoctorFull` calling this check with `config.Load`'s returned `cfg.Org` even on a load error (NEW-5) was independently reproduced live below. |
| `internal/cli/doctor.go` Check 11b comment | Yes | Matches the registration call and the check's actual behavior (static read, `driver_pool`/permissions inputs, no process spawn). |
| Plan's Deviation notes (AC-7 evidence quotes) | Yes | See "Observational checks" — reproduced verbatim, including the plan's own note that the scratch reason text changed to "...and a role resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats)". |
| README.md / AGENTS.md doctor check enumeration | No mention (expected, not drift) | Neither file enumerates individual `ralph doctor` checks by name (README.md's doctor line is a generic one-liner); this matches the plan's own note that this was "confirmed in #162" — no update needed. |

## Observational checks

Live-binary reproduction of AC-7's three recorded evidence lines, using a binary built from this worktree's HEAD (`go build ./cmd/ralph`):

1. **This repo, real `~/.codex/config.toml`, default project config** (`codex_verified = false`, default permission mode `autonomous`, no guarded role):
   ```
   ✓ Codex sandbox (agmsg writable root): pass — not needed: [org.permissions].codex_verified is false and no role resolves to guarded
   ```
   Exact match to the plan's recorded line (line 112). Output-order check also confirms AC-1: this line immediately follows `"Org codex model slugs: pass — ..."`.

2. **Scratch project** (`ralph init` scaffold + `[org.permissions] codex_verified = true` in `ralph.toml`) **+ scratch `CODEX_HOME` with an empty `config.toml`** (no `writable_roots`), against a scratch agmsg home with `scripts/send.sh` present:
   ```
   ⚠ Codex sandbox (agmsg writable root): warn — no writable root in <scratch>/ch-none/config.toml covers the
   agmsg store <scratch>/agmsg-home/db, so a codex seat under workspace-write cannot send RESULT to lead
   ("attempt to write a readonly database"); needed because [org.permissions].codex_verified = true and a role
   resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats); add
   <scratch>/agmsg-home/db to [sandbox_workspace_write].writable_roots (docs/recipes/codex-seat-permissions.md)
   ```
   Matches the plan's recorded wording template and its explicitly-noted reason-text update (line 112). `ralph doctor` exit code: 0.

3. **Same scratch project + scratch `CODEX_HOME` with `writable_roots = ["<agmsg-home>/db"]`**:
   ```
   ✓ Codex sandbox (agmsg writable root): pass — writable root <scratch>/agmsg-home/db in <scratch>/ch-root/config.toml
   covers the agmsg store <scratch>/agmsg-home/db; needed because [org.permissions].codex_verified = true and a role
   resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats)
   ```
   Matches the plan's recorded line. `ralph doctor` exit code: 0.

One construction mistake was caught and corrected during reproduction: appending a second `[org.permissions]` table to the scratch `ralph.toml` produced `toml: table permissions already exists`, which `config.Load` surfaced as `⚠ ralph.toml: warn — parse error: ... using defaults` — the check then graded against the built-in defaults (`codex_verified = false`), not the intended override. This is itself a live confirmation of the NEW-5 doc-comment claim (the check trusts `cfg.Org` as loaded, including a defaults fallback after a load error) rather than a defect; editing the existing key instead of duplicating the table produced the expected warn/pass lines above.

## Scope-5 clause → test mapping (AC-2/AC-3)

| Scope-5 clause | Test(s) |
| --- | --- |
| 不要(理由なし)→ pass | `TestCheckCodexAgmsgWritableRoot_NotNeeded_CodexVerifiedFalse_NoGuardedRole`, `_NotNeeded_GuardedDefaultConfigAbsent`, `_NotNeeded_GuardedDefaultConfigExists`, `_NotNeeded_SandboxModeSetButNoGuardedRole` |
| `codex_verified = true` で root なし → warn | `TestCheckCodexAgmsgWritableRoot_CodexVerifiedNoRoots_Warn` |
| 両方の理由が同時に立つ場合の番号付け | `TestCheckCodexAgmsgWritableRoot_BothReasons_WarnDetailNumbersEachReason` |
| root が `db` と同一 → pass | `TestCheckCodexAgmsgWritableRoot_RootEqualsStore_Pass` |
| root が祖先(agmsg home)→ pass | `TestCheckCodexAgmsgWritableRoot_RootIsAncestor_Pass` |
| 接頭辞が同じだけの別ディレクトリ → warn | `TestCheckCodexAgmsgWritableRoot_PrefixSiblingNotCovered`, `TestPathCovers` |
| symlink 経由で同一 → pass | `TestCheckCodexAgmsgWritableRoot_SymlinkedRoot_Pass` |
| store 自体が symlink で root の外 → warn (self-review MEDIUM-3) | `TestCheckCodexAgmsgWritableRoot_StoreSymlinkedOutsideConfiguredRoot_Warn` |
| store の symlink 先も列挙されている → pass | `TestCheckCodexAgmsgWritableRoot_StoreSymlinkedRealTargetAlsoListed_Pass` |
| `codex_verified = false` でも `sandbox_mode = workspace-write` なら warn | `TestCheckCodexAgmsgWritableRoot_GuardedSeatSandboxModeNoRoots_Warn` |
| 空文字キーの role が default を覆い隠さない (self-review NEW-2) | `TestCheckCodexAgmsgWritableRoot_EmptyStringRoleKeyDoesNotSuppressWarn`, `TestCodexSeatModesPossible` |
| config なし + `codex_verified = true` → warn(存在しない旨を明示、self-review LOW-4/NEW-3) | `TestCheckCodexAgmsgWritableRoot_MissingConfigCodexVerified_WarnNamesConfigAbsent` |
| `~` 表示(self-review MEDIUM-2) | `TestCheckCodexAgmsgWritableRoot_HomeSubstitution_TildeInDetail` |
| 壊れた TOML → info(内容を出さない、self-review MEDIUM-1) | `TestCheckCodexAgmsgWritableRoot_MalformedTOML_InfoWithoutContent`, `_DuplicateKey_InfoWithoutKeyNameOrMessage`, `_TypeMismatch_InfoWithPosition` (self-review LOW-5) |
| `writable_roots` が配列でない → info | `TestCheckCodexAgmsgWritableRoot_WritableRootsWrongType_Info` |
| `driver_pool` に codex なし → pass(self-review LOW-6) | `TestCheckCodexAgmsgWritableRoot_CodexNotInDriverPool_Pass` |
| agmsg 未導入 → info | `TestCheckCodexAgmsgWritableRoot_AgmsgMissing_Info` |
| 導入済みでバージョン違いでも root なしなら warn | `TestCheckCodexAgmsgWritableRoot_AgmsgVersionMismatch_StillGraded` |
| `AGMSG_STORAGE_PATH`: 既定 `db` だけ覆う root → warn / override 先を覆う root → pass / 末尾スラッシュ除去 | `TestCheckCodexAgmsgWritableRoot_StorageOverride` (3 subtests) |
| `profile` 設定時の注記 | `TestCheckCodexAgmsgWritableRoot_ProfileSuffix` |
| config がディレクトリ → info | `TestCheckCodexAgmsgWritableRoot_ConfigIsDirectory_Info` |
| config が FIFO → info(開かずに完了、self-review LOW-7) | `TestCheckCodexAgmsgWritableRoot_ConfigIsFIFO_InfoAndCompletes` (unix-only build tag file) |
| config が 1 MiB 超 → info | `TestCheckCodexAgmsgWritableRoot_ConfigTooLarge_Info` |
| 相対パス・`~` の root は一致しない | `TestCheckCodexAgmsgWritableRoot_RelativeAndTildeRoots_NeverMatch`, `TestPathCovers`, `TestCoveringWritableRoot_StoreNotCreatedYetUnderSymlinkedParent` |
| resolveEnv 失敗 → info | `TestCheckCodexAgmsgWritableRoot_ResolveEnvError_Info` |
| `CODEX_HOME` がリテラルに使われる / `HOME` フォールバック / `AGMSG_STORAGE_PATH` | `TestCodexSandboxEnvFromOS` (4 subtests) |
| store が未作成でも祖先で覆われる / symlink 経由の親でも一致 | `TestCoveringWritableRoot_StoreNotCreatedYet_AncestorCovered`, `_StoreNotCreatedYetUnderSymlinkedParent` |
| `codexNotNeededWhyA`/`codexNotNeededWhyB` の分岐網羅 | `TestCodexNotNeededWhyA`, `TestCodexNotNeededWhyB` |
| 理由の番号付け(self-review NEW-1)の純粋関数テスト | `TestCodexSandboxReasonClause` |
| `runDoctorOpts` 統合テスト + `TestMain` 既定での pass(AC-4) | `TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault`, `_WarnsThroughTheSeam` |

No Scope-5 clause was found without a pinning test. Fixture strings were grepped for anything resembling a secret literal (`api_key=`, etc.) — none found; the file instead uses `CANARYLEAK123` canary strings, consistent with the plan's stated fixture-hygiene mitigation.

## Coverage gaps

- **Not verified here (by design):** `go test ./internal/cli/... -count=1` was not executed — that is `/test`'s responsibility per the skill contract. The Scope-5 mapping above establishes that a pinning test exists for every named clause, but whether each test currently *passes* is unconfirmed by this report.
- **Windows FIFO-equivalent path:** `doctor_codex_writable_root_unix_test.go` is `!windows`-tagged; there is no Windows-specific regression test for "config path resolves to something unopenable" on that platform, matching the plan's stated non-goal scope (this repo's `internal/cli` test suite is already unix-only via `internal/org/lockfile.go`'s unconditional `syscall.Flock`, so this is consistent with existing precedent, not a new gap).
- **Codex's own interpretation of `writable_roots`** (relative-path/`~` expansion, `-c` override merge-vs-replace semantics) remains genuinely unconfirmed at the codex-binary level, as the plan's Assumptions and Non-goals both state explicitly; this report did not attempt to re-verify that against a real `codex` binary, since the plan scopes it out and the Deviation notes record the `codex sandbox` subcommand probe that was already tried and found inconclusive.

## Verdict

- Verified: AC-1, AC-2, AC-3, AC-4, AC-5, AC-6 (static half), AC-7 — code read plus live static-verify/secret-scan/sync-gate runs plus live-binary reproduction of the recorded doctor output (all three AC-7 lines and the AC-1 adjacency).
- Partially verified: AC-6 as a whole — `go test`/`run-verify.sh` intentionally not re-run here; the plan's Deviation notes record the implementer's own green runs after the latest commit (`33ac5a8`), which `/test` should independently confirm.
- Not verified: AC-8 (PR-time only, not yet applicable).
