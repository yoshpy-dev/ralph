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

---

## Cycle 2

- Date: 2026-09-19
- Verifier: `verifier` subagent (Claude Code, standard flow), pipeline cycle 2 of cap 2
- Range: `eec559b..HEAD` (21 commits) since the cycle-1 verify above: cross-review cycle 1 fixes (`ab09dbc`, `91a5542`, `c03c21e` + docs `4c891ec`), cycle-2 self-review (`4c48ae3`) and its two fix commits with **no reviewer pass** (`735fc45`, `5fcf070`), sync-docs/secret-scan-allowlist churn (`04b6df5`…`e2c64c3`), and plan bookkeeping (`fea7fc9`, `7184b2c`).
- Branch: `feat/codex-agmsg-writable-root`, HEAD `7184b2c` (base `main`); `git status --porcelain` empty before and after this pass.
- Scope: spec compliance re-check of AC-1–AC-7 against HEAD, plus static analysis (`./scripts/run-static-verify.sh`) and a fresh history-range secret scan. No behavioral tests run.
- Evidence: `docs/evidence/verify-2026-09-19-090307.log` (gitignored, same convention as cycle 1); live-binary reproduction below.

### Overall verdict: PASS

No blocking finding. Two non-blocking documentation-drift items (AC-3, AC-7 — both wording/enumeration, not code defects) should be closed before `/pr`; see below.

### Spec compliance (cycle 2)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: adjacency | Met (reconfirmed) | Rebuilt the binary from HEAD `7184b2c`; `ralph doctor` from the repo root still prints `"Codex sandbox (agmsg writable root)"` immediately after `"Org codex model slugs"`. Unaffected by the cycle-2 range (registration in `doctor.go:142-148` untouched). |
| AC-2: Scope-5 matrix + Detail contents + path-element boundary | Met | The cycle-1 mapping (this report, "Scope-5 clause → test mapping") still holds — none of the cycle-2 commits touched `pathCovers` or removed a pinning test. The cycle-2 range *adds* tests beyond Scope-5's literal enumeration, each read for assertion strength, not just presence: `TestCheckCodexAgmsgWritableRoot_RootCrossesDotAgentsViaSymlinkTarget_WarnNamesBlockedAncestor`, `_SymlinkedDotAgentsControls_StayPass`, `_StoreUnderDotGitOrDotCodex_Warn`, `_MissingConfigWithBlockedImplicitAncestor_WarnNamesBoth`, `_BlockedAncestorAndExplicitStoreBothListed_PassNamingExplicit` (self-review C2-6's own regression test — asserts `"writable root "+store+" in "` is present and `"writable root "+home+" in "` is absent, not a bare `strings.Contains(r.Detail, store)`), `_SlashTmpDirEqualsTmpDirExcludedSlashTmp_NamesTmpdirEnvVar`, `_SlashTmpDirEqualsTmpDirExcludedTmpdirEnvVar_NamesSlashTmp` (C2-2's regression pair), and `TestCodexRootCoverage_MixedSpellings_ProtectedSymlinkStillBlocks` (`5fcf070`'s own new unit test, with an explicit `covers=true, blocked=false` control case). |
| AC-3: outcome matrix incl. `driver_pool`/`model_pool`/role conditions, decode/type/home failures, `countFailed` exclusion | Met in code; **plan enumeration is behind the code (drift, non-blocking)** | `checkCodexAgmsgWritableRoot`'s 9-outcome doc comment (`internal/cli/doctor_codex_writable_root.go:828-862`) matches the actual `if`/`switch` chain line-by-line, including the two outcomes added since cycle 1 (outcome 3: no codex model in `[org].model_pool` → pass; outcome 8: an implicit temp root covers the store → pass). `Status` is still only ever `"pass"`/`"warn"`/`"info"`, so `countFailed` (`doctor.go`) remains unaffected — confirmed by re-reading the full outcome chain, no new `"fail"` path exists. **Gap:** the plan's own AC-3 text (`docs/plans/active/2026-09-19-codex-agmsg-writable-root.md:61`) does not name the "no codex model in the pool → pass", "store covered by an implicit temp root → pass", or "blocked-ancestor named on warn" outcomes — these were introduced by Scope row 2's cross-review revision passage but AC-3's own enumeration was never updated to match. The behavior itself is correct and tested; only the plan's checklist text is incomplete. |
| AC-4: hermeticity | Met (reconfirmed) | `internal/cli/main_test.go:24-27` unchanged in this range; `TestMain` still pins `doctorCodexSandboxEnv` to a non-existent `ConfigPath` and leaves `TmpDir`/`SlashTmpDir` at their Go zero value (`""`), which `codexImplicitWritableRoots` treats as "no implicit root" — matches the doc comment's claim at `:94-96`. None of the cycle-2 fix commits touched the seam shape. |
| AC-5: recipe + skill mentions, 3 sync gates | Met | Re-ran all three gates this cycle: `./scripts/check-skill-sync.sh` → `13 skill(s) in lock-step`; `./scripts/check-sync.sh` → `DRIFTED: 0`; `./scripts/check-template-purity.sh` → PASS. `cmp` confirms all three mirror pairs are still byte-identical: `docs/recipes/codex-seat-permissions.md` vs its template, `.claude/skills/org/SKILL.md` vs its template, `.agents/skills/org/SKILL.md` vs its template. Both doc surfaces were re-read against the current code (see Documentation drift below); content matches. |
| AC-6: static clean; range secret scan exit 0 | Met (static half only — `go test` is `/test`'s job) | `./scripts/run-static-verify.sh` → `gofmt: ok`, `go vet` silent (clean), `golangci-lint run` → `0 issues.`, `staticcheck` silent (clean) → `All verifiers passed.`, exit 0. `./scripts/secret-scan.sh --range "$(git merge-base HEAD main)..HEAD"` → exit 0 across all 21 commits in the cycle-2 range (merge-base unchanged at `2c511a4`). `go test ./internal/cli/... -count=1` / `./scripts/run-verify.sh` remain out of scope for `/verify`; the plan's Deviation notes record the implementer's `TMPDIR=/tmp go test ./internal/cli/... -count=1` → `ok` after `5fcf070` (the last code-changing commit; `7184b2c` is docs-only). |
| AC-7: evidence lines match current behavior | **Drift found (non-blocking)** — see Observational checks | The plan's last-recorded real-config evidence line (line 112) no longer matches HEAD's output literally: commit `c03c21e` (cycle 2, cross-review WC-2 fix) added the clause "that can use a codex model" to every reason string `codexSandboxReasons` produces and to both `codexNotNeededWhyA`/`codexNotNeededWhyB`, *after* line 112 was written, and no later Deviation note re-records the line. Reproduced live below. The pass/warn/info **verdicts** themselves are unaffected — only the quoted wording is stale. |
| AC-8: PR body `Closes #164` | Pending `/pr` | Unchanged, out of scope. |

### Static analysis (cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass (exit 0) | Scope auto-selected `golang` (changed-language default); `gofmt: ok`, `go vet` clean, `golangci-lint run` → `0 issues.`, `staticcheck` silent/clean. Evidence: `docs/evidence/verify-2026-09-19-090307.log`. |
| `./scripts/secret-scan.sh --range "$(git merge-base HEAD main)..HEAD"` | Pass (exit 0) | History-range scan across all 21 commits on this branch (merge-base `2c511a4`); no findings, including the `.gitallowed` narrowing commits (`90dc4be`, `ee70868`, `e2c64c3`) and the report-hygiene sentences they were added for. |
| `./scripts/check-skill-sync.sh` | Pass | `13 skill(s) in lock-step`. |
| `./scripts/check-sync.sh` | Pass | `DRIFTED: 0`. |
| `./scripts/check-template-purity.sh` | Pass | No meta-repo-specific references in templates. |
| `cmp` recipe root vs template, both skill mirrors vs their templates | Pass (3/3) | Byte-identical. |

### Documentation drift (cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `internal/cli/doctor.go` Check 11b comment (`:142-147`) | Yes (accurate, intentionally abbreviated) | States the `driver_pool`/permissions inputs and the core warn condition, then points to `checkCodexAgmsgWritableRoot` for detail; does not itself enumerate the `model_pool` gate, the protected-directory rule, or the implicit temp roots added since cycle 1 — but it never claims completeness, and the referenced function's own doc comment does carry all of them. Not drift. |
| `checkCodexAgmsgWritableRoot`'s 9-outcome doc comment | Yes | Matches the `if`/`switch` chain read line-by-line, including outcomes 3 and 8 added in the cycle-2 range. |
| `docs/recipes/codex-seat-permissions.md` (+ template) | Yes | States both trigger conditions, the `driver_pool`/`model_pool` gates, "only roles that may use a codex model count," the `/tmp`/`$TMPDIR` implicit roots with their exclusion keys, and the user-level-config-only scope. All match the code (`codexSandboxReasons`, `codexModelPermittedForRole`, `codexImplicitWritableRoots`). |
| `.claude/skills/org/SKILL.md` (4 mirrors) | Yes, with one completeness gap (informational) | States the same trigger conditions, the `driver_pool`/`model_pool` gates, "the model-eligible roles only" restriction, the `.agents`-is-protected caveat (name the `db` directory directly), and the user-level-config-only scope — all match the code. Unlike the recipe, it does not mention the `/tmp`/`$TMPDIR` implicit-root exemption; this is an omission, not a false claim, and AC-5 only requires "a mention of the Check," which is met. Not blocking. |
| `.gitallowed` comment (self-review C2-3 fix) | Yes | States explicitly that the allowlist is evaluated per line and that once a rule matches, nothing else on that line is scanned — matches `secret-scan.sh`'s `is_allowed` behavior and C2-3's recommendation verbatim. |
| `docs/tech-debt/README.md` row (self-review C2-4 fix) | Yes | Now names three (not two) unverified limits — `~`/relative expansion, the protected-directory scope (`pathCrossesCodexProtectedDir`, named explicitly), and guarded-seat `sandbox_mode` inheritance — closing the closed-list overclaim C2-4 flagged. |
| `docs/evidence/codex-seat-permissions-2026-09-18.md` P3 addendum | Yes | States the doctor Check now catches the gap statically, that no live seat re-verification was done, and points at the plan's Non-goals — accurate and unchanged this cycle. |
| Plan AC-3 (acceptance criteria text) | **No — drift (non-blocking)** | See Spec compliance table above: AC-3's own enumeration predates the `model_pool` gate and the implicit-temp-root/blocked-ancestor outcomes. |
| Plan AC-7 evidence quotes (Deviation notes line 112) | **No — drift (non-blocking)** | See Observational checks below. |

### Observational checks (cycle 2)

1. **Live reproduction of the real-config AC-7 line.** Built `./cmd/ralph` from HEAD `7184b2c` into a scratch directory outside the repo, then ran it from the repo root against the real `~/.codex/config.toml` (content never printed, per task instructions):
   ```
   ✓ Codex sandbox (agmsg writable root): pass — not needed: [org.permissions].codex_verified is false and no role that can use a codex model resolves to guarded
   ```
   immediately preceded by `✓ Org codex model slugs: pass — ...` (AC-1 adjacency, reconfirmed). Exit code 0.

   This does **not** literally match the plan's last-recorded quote (Deviation notes, line 112): `"pass — not needed: [org.permissions].codex_verified is false and no role resolves to guarded"` — missing the clause "that can use a codex model" before "resolves to guarded". `git show c03c21e -- internal/cli/doctor_codex_writable_root.go` confirms this exact clause was added to `codexSandboxReasons`, `codexNotNeededWhyA`, and `codexNotNeededWhyB` in that commit, which post-dates the plan line it was supposed to update (`c03c21e` is a cycle-2 cross-review fix; line 112 was written during the cycle-1 revalidation round, well before cross-review cycle 1 even ran). No later Deviation note re-quotes this line with the corrected wording.

   The two scratch-project (warn / pass) AC-7 lines recorded at the same plan line 112 were not independently rebuilt this cycle, but both render their "needed because ..." clause through the exact same `codexSandboxReasons`/`codexSandboxReasonClause` functions the real-config line does — confirmed by code read (`internal/cli/doctor_codex_writable_root.go:369-381`, `:924-931`) — so they carry the identical wording staleness by construction, not by separate reproduction.

   **Recommendation:** append a fresh AC-7 evidence entry (all three lines) to the plan's Deviation notes before `/pr`; not done here since `/verify` does not edit the plan.

2. **Read `735fc45` and `5fcf070` in full** (the two commits the self-review Cycle 2 section never reviewed) against findings C2-1 and C2-2 and the plan's own account (Deviation notes, lines 122-123):
   - **C2-1** (protected-directory rule checked only on the resolved spelling, letting a symlink hide a protected element): `735fc45` introduces `codexRootCoverage`, checking the `resolved/resolved` and `configured/configured` pairs (an improvement over the single resolved-only check C2-1 found, but the commit's own doc comment at the time only claimed "EITHER" of two pairings). `5fcf070` extends the same function to all four pairings (adding `resolved-root/configured-target` and `configured-root/resolved-target`), matching the plan's own account of a residual gap the implementer found manually testing a `.agents`-symlink shape that neither same-spelling pairing could see. The new test `TestCodexRootCoverage_MixedSpellings_ProtectedSymlinkStillBlocks` constructs exactly that shape (root spelled through a symlink; store spelled with the resolved prefix but reaching a protected `.agents` symlink deeper down) and asserts `covers=true, blocked=true`, plus a control case (`covers=true, blocked=false`) for a store no spelling routes through a protected directory. By inspection of the four-term `||` in `codexRootCoverage` (`:601-604`), the test can only pass with all four pairings present — dropping the mixed-pair terms makes the primary assertion fail, since neither same-spelling pair sees the protected element in this fixture.
   - **C2-2** (implicit-root pass names the wrong exclusion key when the seat's `$TMPDIR` equals codex's fixed temp root): `735fc45` replaces the `[]string` implicit-roots list with `[]codexImplicitRoot{Dir, ExcludeKey, FromTmpdirEnv}`, each entry's key set once at construction in `codexImplicitWritableRoots` (`:679-687`) rather than re-derived post hoc by comparing the matched directory string against `slashTmpDir`. Two new tests — `TestCheckCodexAgmsgWritableRoot_SlashTmpDirEqualsTmpDirExcludedSlashTmp_NamesTmpdirEnvVar` and its `_ExcludedTmpdirEnvVar_NamesSlashTmp` counterpart — construct the exact ambiguous case (`TMPDIR` == the fixed temp root) for both exclusion combinations and assert the correct key is present while the other is explicitly absent, plus (for the `TMPDIR`-sourced match) that the pane-environment note appears exactly once. Read in full; no string-identity comparison remains anywhere in the matched-entry rendering path (`codexImplicitRootDetail`, `:754-766`).
   - Both fix commits' new tests were read for assertion strength (specific field/phrase, not a bare status check) before being counted as verifying the fix.

### Coverage gaps (cycle 2)

- **Not verified here (by design):** `go test ./internal/cli/... -count=1` was not run — that is `/test`'s responsibility. The plan's Deviation notes record the implementer's own `TMPDIR=/tmp` green run after `5fcf070` (the last code-changing commit before this report; `7184b2c` is docs-only).
- **AC-3 plan enumeration** (see above) — behavior correct and tested, plan checklist text incomplete. Non-blocking.
- **AC-7 plan evidence quotes** (see above) — verdicts correct, literal wording stale post-`c03c21e`. A fresh evidence entry should be appended before `/pr`. Non-blocking.
- **Skill bullet's `/tmp`/`$TMPDIR` omission** relative to the recipe — informational, does not affect AC-5.
- **Scratch-project warn/pass AC-7 lines** were not independently rebuilt this cycle (see Observational checks item 1) — their staleness is inferred from the shared producer function, not from a fresh live run. If a fresh AC-7 evidence entry is added to the plan, it should include a live rebuild of both scratch scenarios, not just the real-config line.
- **Unchanged from cycle 1, still out of scope:** the Windows FIFO-equivalent gap; codex's own interpretation of `writable_roots` (relative/`~` expansion, `-c` merge-vs-replace semantics).

### Verdict (cycle 2)

- Verified: AC-1, AC-2, AC-4, AC-5, AC-6 (static half) — reconfirmed against the full cycle-2 range with live rebuild, static-verify/secret-scan/sync-gate reruns, and a full read of the two previously-unreviewed fix commits (`735fc45`, `5fcf070`) against their target findings.
- Verified in code, with a non-blocking plan-drift gap: AC-3 (the doc comment / `if`-chain / tests are all correct and complete; the plan's own AC-3 text has not been updated for the two newest outcomes), AC-7 (the pass/warn/info verdicts are correct and reconfirmed live; the plan's quoted wording is stale after `c03c21e` and needs a fresh evidence entry before `/pr`).
- Partially verified: AC-6 as a whole — `go test`/`run-verify.sh` intentionally not re-run here; the plan's Deviation notes record the implementer's own green run after `5fcf070`, which `/test` should independently confirm for the cycle-2 range.
- Not verified: AC-8 (PR-time only, not yet applicable).
- **Overall: PASS.** No CRITICAL/blocking finding. Recommend closing the two documentation-drift items (AC-3 enumeration, AC-7 evidence refresh) in the plan before `/pr` — both are wording gaps, not behavioral defects.
