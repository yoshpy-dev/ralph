# Self-review report: codex-agmsg-writable-root

- Date: 2026-09-19
- Plan: `docs/plans/active/2026-09-19-codex-agmsg-writable-root.md` (issue #164)
- Reviewer: `reviewer` subagent (Claude Code), pipeline cycle 1 of cap 2
- Scope: diff quality only for `git diff main...HEAD` on `feat/codex-agmsg-writable-root`
  (HEAD `f51147f`, 6 commits, 11 files, +1212/-6). No spec-compliance, test-coverage,
  or documentation-drift judgement — those belong to `/verify` and `/test`.

## Evidence reviewed

- `git diff main...HEAD` in full (all 11 files), plus `git log --oneline main..HEAD`.
- New source read line by line: `internal/cli/doctor_codex_writable_root.go` (386 lines),
  `internal/cli/doctor_codex_writable_root_test.go` (664 lines).
- Registration and seam: `internal/cli/doctor.go:141-148`, `internal/cli/main_test.go:12-28`.
- Truthfulness cross-checks against the producer: `internal/org/permissions.go`
  (`codexEditsArgs`/`codexAutonomousArgs` = `--sandbox workspace-write`;
  `permissionArgsForDriver`'s `guarded` arm returns `nil, nil`),
  `internal/org/driver/driver.go:82` (`AgmsgAvailable`),
  `internal/org/driver/agmsg.go:123` (`ResolveAgmsgHome`),
  `internal/config/config.go:76-93` (`OrgPermissionsConfig`).
- Sibling check for convention parity: `internal/cli/doctor_shell_alias.go` (its
  `displayPath` closure at line 732, built from `env.Home`; its `syscall.ENOTDIR`
  handling at line 105; its herdr gate at `internal/cli/doctor.go:119-125`).
- Cited external evidence verified: `docs/evidence/codex-seat-permissions-2026-09-18.md`
  P3 (line 45-47) confirms the "attempt to write a readonly database (8)" failure and
  the `writable_roots` fix; `~/.agents/skills/agmsg/scripts/lib/storage.sh:19-38`
  confirms the `AGMSG_STORAGE_PATH` > `<skill_dir>/db` priority and the
  `${AGMSG_STORAGE_PATH%/}` single-trailing-slash trim that `agmsgStoreDir` mirrors.
- Library behaviour probed with throwaway `go test` files under the scratchpad
  (written into `internal/cli/`, run, then deleted; `git status --porcelain` is empty):
  go-toml v2.3.0 error shapes, and the check's end-to-end Detail strings for absent /
  malformed / wrong-typed configs, symlinked stores, and four root spellings.
- One environment probe: `TMPDIR="$HOME/..." go test ./internal/cli/ -run
  TestCheckCodexAgmsgWritableRoot` (see MEDIUM-2).

No test suite, static analysis, linter, or formatter was run as a verdict input; the
`go test` invocations above exist only to produce the evidence quoted in the findings.

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | exception-handling | **MEDIUM-1. A TOML parse error that is not a `*toml.DecodeError` prints go-toml's raw message, which embeds a key or table name taken from the user's config — and is labelled "could not read", which is false (the file read fine, it failed to parse).** The doc comment two functions up states the opposite guarantee. | `internal/cli/doctor_codex_writable_root.go:302-307` promises "A TOML parse/decode error reports only the `*toml.DecodeError`'s line/column — never its message text or any other part of the config, which can hold secrets." But go-toml v2.3.0 raises duplicate-key and table-conflict errors from `internal/tracker/seen.go:200,216,268,293` and wraps them at `unmarshaler.go:124` with a plain `fmt.Errorf`, so `errors.As(readErr, &decodeErr)` at `:334` is **false** and control falls to `:342`. Probe output: input `token = "sk-AAA"\ntoken = "sk-BBB"` → `asDecodeError=false Error()="toml: key token is already defined"`; input with a repeated `[sandbox_workspace_write]` header → `"toml: table sandbox_workspace_write already exists"`. Both render verbatim into the Detail via `codexConfigReadReason` (`:141-146`). `TestCheckCodexAgmsgWritableRoot_MalformedTOML_InfoWithoutContent` (test:179) only covers the syntax shape, which *is* a `DecodeError`, so its `CANARYLEAK123` assertion never reaches this branch. Values do not leak with the current struct (go-toml's value-bearing messages at `unmarshaler.go:1064-1112` are int-range errors, and `codexUserConfig` has no integer fields) — key and table **names** do. | Make `readCodexUserConfig` wrap decode failures in a distinct error type (e.g. `errCodexConfigParse{err}`) so `checkCodexAgmsgWritableRoot` can branch on parse-vs-read. Keep the line/column rendering for `*toml.DecodeError`; for any other parse error emit `"%s could not be parsed as TOML — writable roots not checked"` with no message text. Then correct `codexConfigReadReason`'s doc at `:136-140`, whose "is exactly 'not a regular file' or 'larger than 1 MiB'" enumeration is already incomplete. Add a duplicate-key fixture to the canary test. |
| MEDIUM | maintainability | **MEDIUM-2. `codexConfigDisplayPath` reads the real `os.UserHomeDir()` instead of the injected seam, so six of the new tests fail on any machine whose `TMPDIR` sits under `$HOME`.** The doc comment claims parity with the sibling, but the sibling is immune precisely because it does not do this. | `internal/cli/doctor_codex_writable_root.go:76-90` calls `os.UserHomeDir()` directly; `codexSandboxEnv` (`:22-25`) has no `Home` field, so `TestMain` cannot pin it. The sibling's `displayPath` (`internal/cli/doctor_shell_alias.go:731-737`) builds `homePrefix` from `env.Home`, i.e. from the seam. Reproduced: `TMPDIR="$HOME/ralph-tmpdir-probe" go test ./internal/cli/ -run TestCheckCodexAgmsgWritableRoot` → `FAIL: TestCheckCodexAgmsgWritableRoot_CodexVerifiedNoRoots_Warn … expected detail to contain "/Users/<user>/ralph-tmpdir-probe/…/config.toml", got: no writable root in ~/ralph-tmpdir-probe/…/config.toml covers …`. Every assertion of the form `strings.Contains(r.Detail, cfgPath)` (test:64, and the same shape elsewhere) carries the same dependency. | Add `Home string` to `codexSandboxEnv`, fill it in `codexSandboxEnvFromOS`, pin it in `TestMain` and `codexSandboxTestEnv`, and change `codexConfigDisplayPath(path string)` to `codexConfigDisplayPath(home, path string)`. That both fixes the tests and makes the redaction observable from the seam, matching the doc comment's claim. |
| MEDIUM | security | **MEDIUM-3. The unresolved first pass in `coveringWritableRoot` returns `pass` for a store directory that is a symlink pointing out of every configured root** — the exact configuration the check exists to catch, reported clean. | `internal/cli/doctor_codex_writable_root.go:267-272` compares the cleaned, as-written strings and returns on the first textual hit, before `resolveNearestExisting` is ever consulted at `:273`. Probe: `<agmsgHome>/db` created as a symlink to `<base>/elsewhere`, `writable_roots = ["<agmsgHome>"]` → `status=pass … covers the agmsg store <agmsgHome>/db`. codex's sandbox evaluates the real path, so the seat would still fail with "attempt to write a readonly database". | Delete the first loop (`:268-272`) and keep only the resolving pass. It is strictly stronger and loses nothing: `resolveNearestExisting` falls back to `filepath.Clean(path)` when nothing resolves (`:243-265`), so every case the text pass accepts is still accepted, while the symlinked-store false pass turns into the correct `warn`. Verified unaffected by the change: trailing-slash, `.`-segment, `..`-segment and `//` root spellings all already pass through `filepath.Clean`; `/tmp` → `/private/tmp` on macOS resolves on both sides. |
| LOW | readability | **LOW-4. The "not needed" all-clear asserts a fact about a config file that may not exist.** `readCodexUserConfig` computes and returns `exists`, and the only caller discards it. | `internal/cli/doctor_codex_writable_root.go:332` (`cfg, _, readErr := …`) and the Detail at `:356`. Probe with an absent path: `pass — not needed: [org.permissions].codex_verified is false and <dir>/nope.toml does not set sandbox_mode = "workspace-write"`. `TestCheckCodexAgmsgWritableRoot_NotNeeded_Pass` (test:38-51) deliberately passes an absent config and asserts only the substring `not needed`, so it pins the wording rather than catching it. This is also the shape the pinned `TestMain` seam produces for every pre-existing `runDoctor*` test. | Keep `exists` and branch the sentence: `"not needed: [org.permissions].codex_verified is false and there is no codex config at %s"` when `!exists`. |
| LOW | typo | **LOW-5. A syntactically valid TOML file with a wrong-typed value is reported as "is not valid TOML".** | `internal/cli/doctor_codex_writable_root.go:338`. Probe: `sandbox_mode = 123` → `info — <path> is not valid TOML (line 1, column 16)`. go-toml returns a `*toml.DecodeError` for type mismatches too (`asDecodeError=true`, `Error()="toml: cannot decode TOML integer into struct field …"`), so the branch is correct but the wording is not. Same for `sandbox_workspace_write = "x"`. | Reword to `"%s could not be decoded (line %d, column %d) — writable roots not checked"`. Folds naturally into the MEDIUM-1 fix. |
| LOW | maintainability | **LOW-6. The reason clause fires on `codex_verified` alone and ignores `[org.permissions].default`/`roles` and `[org].driver_pool`, so it can describe seats that cannot exist.** | `internal/cli/doctor_codex_writable_root.go:182-193`; the registration at `internal/cli/doctor.go:148` passes only `cfg.Org.Permissions.CodexVerified` although the whole `cfg` is in hand. With `[org.permissions].default = "guarded"` and no role override, `permissionArgsForDriver`'s guarded arm returns `nil, nil` (`internal/org/permissions.go`), so ralph passes no `--sandbox` flag to any codex seat, yet the Detail still reads "ralph passes `--sandbox workspace-write` to edits and autonomous codex seats". Same with `driver_pool = ["claude"]`. The sibling gates its codex clauses on `herdrResult.Status == "pass"` (`internal/cli/doctor.go:119-125`) for the analogous "is the org runtime even in play" reason, and `docs/tech-debt/README.md` already carries an open row recording this exact gap for `checkShellAliases`. | Cheapest correct fix: soften to "ralph **would pass** `--sandbox workspace-write` to any edits or autonomous codex seat". Precise fix: pass `cfg.Org` and drop reason (a) when every resolved mode is `guarded`. Recommend the wording fix now plus the tech-debt row below, since the default `[org.permissions].default` is `autonomous` (`internal/config/config.go:157`), making the false positive narrow. |
| LOW | maintainability | **LOW-7. The `runtime.GOOS == "windows"` skips in the new test file are unreachable — the file cannot compile on Windows at all.** | `internal/cli/doctor_codex_writable_root_test.go:340` skips, then line 347 calls `syscall.Mkfifo`, which is not defined on Windows; a runtime skip cannot rescue a missing symbol. `TestCodexSandboxEnvFromOS`'s skip at line 420 is in the same file and inherits the problem. The module is already unix-only (`internal/org/lockfile.go:67,79` use `syscall.Flock` with no build tag) and `.goreleaser.yaml` builds `darwin`/`linux` only, so nothing is broken — the skips just advertise a portability guarantee the package does not have. | Either split the FIFO test into a `//go:build unix` file, or drop both skips and state "unix-only package" in the file header. Lowest priority item in this report. |
| LOW | maintainability | **LOW-8. `TestCoveringWritableRoot_StoreNotCreatedYet_AncestorCovered` is satisfied by the textual first pass, so it does not exercise the helper its name and comment credit.** | `internal/cli/doctor_codex_writable_root_test.go:523-531`: `coveringWritableRoot([]string{agmsgHome}, filepath.Join(agmsgHome, "db"))` matches at `doctor_codex_writable_root.go:268-272` without reaching `resolveNearestExisting`. Delete `resolveNearestExisting` entirely and this test still passes. The sibling test at line 646 (`…UnderSymlinkedParent`) does exercise it and would fail. | If MEDIUM-3's fix lands, this test becomes meaningful as written and needs no change. If MEDIUM-3 is declined, retitle it so it does not claim coverage it lacks. |
| LOW | typo | **LOW-9. `agmsgStoreDir`'s doc cites the evidence file for agmsg's priority order, but that file never mentions `AGMSG_STORAGE_PATH`.** | `internal/cli/doctor_codex_writable_root.go:195-201` cites "docs/evidence/codex-seat-permissions-2026-09-18.md, agmsg 1.1.13's storage.sh". `grep -n AGMSG_STORAGE_PATH docs/evidence/codex-seat-permissions-2026-09-18.md` returns nothing; the evidence file supports only the `<home>/.agents/skills/agmsg/db` default (line 14, line 47). The behavioural claim itself is correct — `~/.agents/skills/agmsg/scripts/lib/storage.sh:20-23` does `${AGMSG_STORAGE_PATH%/}`, matching the Go `strings.TrimSuffix(storageOverride, "/")` exactly. | Cite `storage.sh` alone for the override and trim, and keep the evidence citation for the default `<agmsgHome>/db`. |

### Checked and clean

- **Secrets / credentials:** no keys, tokens, or credential-shaped literals anywhere in the
  diff. Fixtures are paths, `sandbox_mode = "workspace-write"`, and the deliberate
  `CANARYLEAK123` canary (not secret-shaped); the plan's CI secret-scan risk is addressed.
- **Debug code:** no `fmt.Print`/`log`/`TODO`/`FIXME`/commented-out code in the new files.
- **Path traversal / injection:** the check only reads a path derived from `CODEX_HOME`, and
  never executes anything. `driver.AgmsgAvailable` is a `stat`, not a spawn.
- **Unbounded reads / hangs:** `os.Stat` + `Mode().IsRegular()` precedes `os.Open`
  (`:103-113`), so a FIFO cannot block the doctor run — the class the sibling PR #168 hit.
  The 1 MiB cap uses the `LimitReader(f, max+1)` + `len(data) > max` idiom correctly.
- **Termination:** `resolveNearestExisting`'s walk terminates on `parent == dir` (`:256-258`).
- **Exit code:** the check never returns `fail`, so `countFailed` is unaffected; confirmed
  by `TestRunDoctorOpts_CodexSandboxCheck_WarnsThroughTheSeam` (test:587).
- **Path-element matching:** probed `trailing slash`, `.` segment, `..` segment and `//`
  spellings of a covering root — all `pass`; `…/dbx` vs `…/db` and `/a/b` vs `/a/bc` — `warn`.
  Relative and `~`-prefixed roots are rejected on both passes (`:216-220`, `:275-277`).
- **Unnecessary changes:** none. The `main_test.go` edit is two lines plus a comment; the
  `doctor.go` edit is one registration plus a comment; the four skill mirrors and two recipe
  copies are byte-identical additions of the same passage.
- **Mirror discipline:** `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`, and
  both `templates/base/` copies carry the identical hunk; the recipe root and template copies
  likewise. No `internal/`- or `cmd/`-path leakage into the template copies.
- **Line widths:** the new skill paragraph measures 65-78 display columns (CJK counted as 2),
  inside the file's 63-78 norm. No reflow needed.
- **Doc accuracy:** the recipe and skill passages both state "user-level `config.toml` only;
  profiles, project-level config and `-c` overrides are not evaluated", which matches the
  code and the plan's Non-goals. No overstatement found in either doc hunk.

## Positive notes

- The two Codex plan-advisory findings were genuinely implemented, not just recorded:
  `agmsgStoreDir` mirrors agmsg's own resolution order including the single-trailing-slash
  trim (verified against `storage.sh` line by line), and installation is decided by
  `driver.AgmsgAvailable` rather than the sibling agmsg check's `Status` — with
  `TestCheckCodexAgmsgWritableRoot_AgmsgVersionMismatch_StillGraded` (test:246) asserting
  the sibling really is `info` for a version mismatch before asserting the new check warns.
  That test would catch a regression to the `Status`-based shortcut.
- The FIFO hazard from the sibling PR's review history was pre-empted, not repeated: the
  regular-file check happens before `os.Open`, and the test asserts completion inside a 5s
  select rather than just asserting the status.
- `codexWritableRoots` typing the field as `any` to distinguish "wrong shape" from "absent"
  instead of letting a bad array silently fail the whole decode is the right call, and the
  reason is written down at the struct (`:55-58`).
- The `doctorCodexSandboxEnv` seam is pinned in `TestMain` with the reason recorded, so the
  13-pre-existing-tests hermeticity problem from PR #168 did not recur (modulo MEDIUM-2,
  which is a different axis: the home used for *display*, not the config path).

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `checkCodexAgmsgWritableRoot` grades on `[org.permissions].codex_verified` alone: it never reads `[org.permissions].default` / `[org.permissions].roles` or `[org].driver_pool`, so it warns on a project where every codex seat resolves to `guarded` (`permissionArgsForDriver` returns no flags) or where codex is not even in the driver pool. The reason clause "ralph passes `--sandbox workspace-write` to edits and autonomous codex seats" then describes seats that cannot exist. Same class as the open `checkShellAliases` row above, one step narrower: this check does read `codex_verified`, but not the resolved modes. | A warn that no action can clear on a guarded-only or claude-only project; erodes trust in the check | `[org.permissions].default` defaults to `autonomous` (`internal/config/config.go:157`), so the false positive needs an explicit guarded default plus no role override; the full fix means threading `cfg.Org` and duplicating `ResolvePermissionMode`'s precedence into doctor | A user reports the warn on a project with `default = "guarded"`, or the next change to either doctor codex check | `docs/plans/active/2026-09-19-codex-agmsg-writable-root.md`, this report LOW-6 |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_ — the row above is
deferred work identified by this review; it is not yet in `docs/tech-debt/README.md`.

## Recommendation

- **Merge: yes, after MEDIUM-1 through MEDIUM-3.** No CRITICAL and no HIGH findings.
  Per `code-review.md`'s approval criteria this already clears the bar, but all three
  MEDIUMs are small, local edits and this is pipeline cycle 1 of 2 — fixing them now
  costs one revalidation round and avoids shipping a documented security property that
  the code does not hold (MEDIUM-1), a test that fails on a legitimate `TMPDIR`
  (MEDIUM-2), and a `pass` on the misconfiguration the check was written to catch
  (MEDIUM-3). MEDIUM-3's fix is a five-line deletion.
- Suggested single fix commit: MEDIUM-1 + LOW-5 together (one error-classification
  change), MEDIUM-2 (seam field), MEDIUM-3 (delete the text pass), LOW-4 (`exists`),
  LOW-6 wording, LOW-9 citation. LOW-7 and LOW-8 are optional polish.
- Follow-ups: append the tech-debt row above to `docs/tech-debt/README.md`. If MEDIUM-3
  is declined, LOW-8's test title must change so it stops claiming coverage it lacks.
- Known gaps in this review: severity judgements are diff-quality only. Spec compliance
  against AC-1..AC-8, static analysis, and behavioural test results are `/verify`'s and
  `/test`'s to report; nothing here was run as a gate.

---

## Revalidation (cycle 1, fix round)

- Date: 2026-09-19
- Fix range reviewed: `git diff 03d9e93..HEAD`, HEAD `3856643` (`636da6a` plan,
  `0811070` Go fixes, `65a6f86` parameter drop, `5bacc46` docs, `3856643` plan notes).
  12 files, +785/-219.
- Verdict: **merge**. All nine original findings are resolved. Six new findings, all
  LOW, none blocking.

### Evidence reviewed for the fix round

- `internal/cli/doctor_codex_writable_root.go` re-read in full (now 572 lines), plus
  the rewritten test file, the new `internal/cli/doctor_codex_writable_root_unix_test.go`,
  and the `doctor.go` / `main_test.go` / docs / plan hunks.
- New logic cross-checked against the producer: `internal/org/permissions.go`
  (`ResolvePermissionMode` precedence, `permissionArgsForDriver`'s codex arms),
  `internal/org/envelope.go:53,77` (driver rejected when outside `[org].driver_pool`),
  `internal/config/config.go:140,157,320-331` (defaults and `[org.permissions]`
  validation), `internal/cli/org.go:210-214` (`--role` is required and non-blank, so
  a seat's role is never the empty string).
- A 15-case matrix driven through the real `checkCodexAgmsgWritableRoot` in a throwaway
  test, printing `codexSeatModesPossible`'s two flags plus the status and Detail for each,
  next to `permissionArgsForDriver`'s actual output for all six (codex_verified × mode)
  pairs. Probe files removed; `git status --porcelain` is empty.
- `TMPDIR="$HOME/ralph-tmpdir-probe" go test ./internal/cli/ -run
  'TestCheckCodexAgmsgWritableRoot|TestCodex|TestCovering|TestPathCovers|TestAgmsgStoreDir'
  -count=1` → `ok`. This is the exact command that failed before the fix.
- `GOOS=windows go vet ./internal/cli/` → only the pre-existing
  `internal/org/lockfile.go` `syscall.Flock` errors; nothing from the writable-root files.
- Mirror parity re-checked with `cmp`: all four `org/SKILL.md` copies byte-identical,
  both `codex-seat-permissions.md` copies byte-identical. Revised skill paragraph
  measures 71-77 display columns, the recipe 60-73.

### Per-finding status

| Finding | Status | Evidence |
| --- | --- | --- |
| MEDIUM-1 (go-toml message text leaks a config key name) | **Resolved** | Every `toml.Unmarshal` failure is now wrapped in `codexConfigDecodeError` (`internal/cli/doctor_codex_writable_root.go:125-139`, wrapped at `:186-197`), which carries only `Line`/`Column`/`HasPosition` and no text. The caller branches on it at `:531-536` and renders through `codexConfigDecodeDetail` (`:142-152`). `codexConfigReadReason` (`:207-212`) is now reachable only for read failures, and its doc comment says so. `TestCheckCodexAgmsgWritableRoot_DuplicateKey_InfoWithoutKeyNameOrMessage` (test:396) asserts the absence of `note`, `already defined`, and the canary value; it passed in the hostile-`TMPDIR` run above. |
| MEDIUM-2 (real `os.UserHomeDir` in the display helper) | **Resolved** | `codexSandboxEnv.Home` added (`:26`), filled in `codexSandboxEnvFromOS` (`:45-64`), pinned in `TestMain` (`internal/cli/main_test.go:25-27`). `codexConfigDisplayPath(home, path)` (`:103-111`) no longer calls `os.UserHomeDir`; `grep -n os.UserHomeDir` finds exactly one occurrence in the production file, inside `codexSandboxEnvFromOS`. The `TMPDIR` reproduction now passes, and `TestCheckCodexAgmsgWritableRoot_HomeSubstitution_TildeInDetail` (test:345) proves the `~` came through the seam by using a temp-dir home. |
| MEDIUM-3 (textual first pass returns a false `pass`) | **Resolved** | `coveringWritableRoot` (`:405-416`) has a single pass, resolving both sides. `TestCheckCodexAgmsgWritableRoot_StoreSymlinkedOutsideConfiguredRoot_Warn` (test:257) pins the exact case I reported, and `..._StoreSymlinkedRealTargetAlsoListed_Pass` (test:278) pins that the legitimate match still passes and names the real-target root. Restoring the deleted loop turns the first test red. |
| LOW-4 (all-clear about an absent config) | **Resolved** | `exists` is consumed at `:527` and reaches both `codexNotNeededWhyB` (`:308-317`, "…does not exist") and `codexWritableRootDetail`'s warn branch (`:448-452`). Pinned by `..._NotNeeded_GuardedDefaultConfigAbsent` (test:77) and `..._MissingConfigCodexVerified_WarnNamesConfigAbsent` (test:318), both asserting the full expected string. |
| LOW-5 ("is not valid TOML" for a type mismatch) | **Resolved** | Wording is now "could not be decoded as a codex config" (`:142-152`); `..._TypeMismatch_InfoWithPosition` (test:421) asserts the position is present and the old phrase is absent. |
| LOW-6 (reason driven by `codex_verified` alone) | **Resolved in code, not deferred** | `checkCodexAgmsgWritableRoot` now takes `config.OrgConfig` (`:502`), short-circuits on `!slices.Contains(orgCfg.DriverPool, "codex")` (`:510-514`), and gates each reason on `codexSeatModesPossible` (`:252-266`, `:277-291`). I verified the gating against the producer for all six (codex_verified × mode) pairs: codex + guarded returns no args either way, codex + edits/autonomous errors when unverified and returns `--sandbox workspace-write` when verified. The 15-case matrix agrees with that table in every row. See NEW-2 and NEW-5 for two residual edges. |
| LOW-7 (unreachable Windows skip) | **Resolved** | The FIFO test moved to `internal/cli/doctor_codex_writable_root_unix_test.go` behind `//go:build !windows`; `syscall` is no longer imported by `doctor_codex_writable_root_test.go`. `GOOS=windows go vet ./internal/cli/` reports only the pre-existing `internal/org/lockfile.go` failures, and the new file's header comment names that pre-existing constraint accurately. |
| LOW-8 (test did not exercise the helper it credited) | **Resolved** | With the textual pass gone, `TestCoveringWritableRoot_StoreNotCreatedYet_AncestorCovered` (test:760) now reaches `resolveNearestExisting` on both sides, so its name matches what it runs. |
| LOW-9 (evidence file credited for `AGMSG_STORAGE_PATH`) | **Resolved** | `agmsgStoreDir`'s comment (`:321-331`) now credits `scripts/lib/storage.sh` for the override and its trim, keeps the evidence file only for the default location, and states outright that the evidence file does not describe the variable. |

### New findings introduced by the fix round

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | **NEW-1. When both reasons fire, the reason clause reads as a flat four-clause `and` chain.** Each reason string now contains its own " and " (the role condition), and `strings.Join(reasons, " and ")` adds a third, so the reader cannot see where one reason ends and the next begins. | `internal/cli/doctor_codex_writable_root.go:279-289` build the two reasons; `:553` joins them. Matrix case 9 output: `needed because [org.permissions].codex_verified = true and a role resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats) and sandbox_mode = "workspace-write" in <cfg> and a role resolves to guarded (guarded codex seats inherit it)`. `TestCheckCodexAgmsgWritableRoot_BothReasons_WarnDetailJoinsWithAnd` (test:153) pins this shape as intended, so nothing will flag it later. This was unambiguous before the fix, when each reason was a single clause. | Join with `"; "` instead of `" and "`, or wrap each reason in parentheses. Update the test's name and assertion to match. |
| LOW | maintainability | **NEW-2. `codexSeatModesPossible` derives the effective default through `ResolvePermissionMode(orgCfg, "")`, which a `[org.permissions.roles]` entry keyed with the empty string silently intercepts — turning a should-warn into a `pass`.** | `internal/cli/doctor_codex_writable_root.go:261`. `org.ResolvePermissionMode` returns `Roles[role]` first, so with `default = "guarded"` and `roles = {"": "edits"}` the default is never applied. Probed (matrix case 12): `modes(ww=true, guarded=false)` and, with `sandbox_mode = "workspace-write"` in the codex config, the check returns `pass — not needed: … and no role resolves to guarded` where it should warn. `config.Load` accepts such a document, because `internal/config/config.go:326-330` validates only the mode value, never the role key. Likelihood is low: `--role` is required and non-blank at spawn (`internal/cli/org.go:210-214`), so `""` is not a real role — it is a config typo that disables reason (b). | Two lines at `:261`: copy `orgCfg`, nil out `Permissions.Roles` on the copy, and resolve the default from that copy, so no role entry can intercept it. The loop over `orgCfg.Permissions.Roles` below already covers every real role. |
| LOW | readability | **NEW-3. In the config-absent warn, the "does not exist" parenthetical is attached to the store path, not to the config path it describes.** | `internal/cli/doctor_codex_writable_root.go:448-452` renders `no writable root covers the agmsg store <store> (<config> does not exist), so a codex seat …`. The two paths differ, so a careful reader recovers, but at a glance the parenthetical reads as a statement about the store directory. `..._MissingConfigCodexVerified_WarnNamesConfigAbsent` (test:318) asserts this exact adjacency. | Reword to `no writable root covers the agmsg store %s — there is no codex config at %s — so a codex seat …`, or move the clause next to the `add … in %s` fragment that already names the config. |
| LOW | unnecessary-change | **NEW-4. The plan's Scope row 1 still enumerates the check's input as `codex_verified` only, contradicting the same table's row 2, AC-3, and the shipped signature.** | `docs/plans/active/2026-09-19-codex-agmsg-writable-root.md` Scope row 1 reads "入力: `codex_verified`(`cfg.Org.Permissions.CodexVerified`)、解決済みの agmsg home…", while row 2 and AC-3 were both revised for `driver_pool` and roles, and `internal/cli/doctor_codex_writable_root.go:502` takes `config.OrgConfig`. Scope row 5's test enumeration likewise predates the driver-pool and role tests. | One-line edit to Scope row 1 ("入力: `[org]` envelope(`cfg.Org`)、…"). `/verify` owns the acceptance-criteria judgement; this is flagged only as an internal contradiction inside the diff. |
| LOW | maintainability | **NEW-5. The check now consumes `cfg.Org` even when `config.Load` returned an error, so an invalid `[org.permissions].default` produces a confident `pass` whose two stated reasons are both false.** Optional to fix. | `internal/cli/doctor.go:148` passes `cfg.Org` unconditionally; `runDoctorFull` keeps going after a load error (`internal/cli/doctor.go:89-96`, which prints its own `ralph.toml: warn`). `config.Load` returns the partially-populated `cfg` alongside a validation error (`internal/config/config.go:323-331`). Probed (matrix case 13): `default = "typo"` yields `modes(ww=false, guarded=false)` and `pass — not needed: no role resolves to edits or autonomous and no role resolves to guarded`, although the document does set a default. The verdict is harmless (a project whose `ralph.toml` does not load cannot spawn any seat), only the reasons are wrong, and doctor prints the parse warning on its own line. Before the fix the check read a single bool and could not hit this. | Cheapest option: add a sentence to `checkCodexAgmsgWritableRoot`'s doc comment saying it trusts the `[org]` envelope as loaded and that an unloadable `ralph.toml` is reported by the separate `ralph.toml` check. A behavioural fix would need `cfgErr` threaded to the check, which is more churn than the case warrants. |
| LOW | typo | **NEW-6. Two small comment inaccuracies.** (a) `codexConfigDisplayPath`'s doc says `Home` is "itself only ever set by `codexSandboxEnvFromOS`", but `TestMain` and one test also set it — which is the entire point of the MEDIUM-2 fix. (b) Two test comments call the driver-pool branch "Step 0", a label the implementation's own enumeration does not use; the source numbers it outcome 2. | (a) `internal/cli/doctor_codex_writable_root.go:95-96` versus `internal/cli/main_test.go:26` and `internal/cli/doctor_codex_writable_root_test.go:352`. (b) `internal/cli/doctor_codex_writable_root_test.go:42,468` versus `internal/cli/doctor_codex_writable_root.go:478-480`. | (a) "set by `codexSandboxEnvFromOS` in production and by the seam in tests". (b) Say "outcome 2" so the label is greppable against the source. |

### Positive notes on the fix round

- The LOW-6 fix went to the producer rather than to prose: I checked
  `permissionArgsForDriver`'s output for all six (codex_verified × mode) pairs and every
  row of a 15-case config matrix agrees with it, including the two directions that matter
  most — `codex_verified = false` with an edits role warns about nothing (the spawn would
  error), and `default = "guarded"` with `sandbox_mode = "workspace-write"` warns on
  reason (b) alone.
- `codexConfigDecodeError` fixes the class rather than the instance: it wraps *every*
  `toml.Unmarshal` failure, so a future go-toml release that adds a new plain-error shape
  cannot reopen MEDIUM-1.
- The three `codexNotNeededWhyB` branches, the two `codexNotNeededWhyA` branches, and
  `codexSeatModesPossible`'s five mode combinations each have a direct table test
  (test:794, test:829, test:847), and four of the not-needed cases assert the full Detail
  string with `!=` rather than `strings.Contains`, so a wording regression cannot slip past.
- The windows fix used a build tag rather than a broader refactor, and its comment states
  the pre-existing unix-only constraint instead of claiming portability the package
  does not have.

### Updated recommendation

- **Merge: yes.** All three MEDIUMs and all six LOWs from the initial review are resolved,
  each with a test that goes red on a revert. The six new findings are LOW: NEW-1 through
  NEW-3 are wording or a two-line guard, NEW-4 is a one-line plan edit, NEW-5 is optional
  and documented, NEW-6 is two comments.
- Follow-ups if the round has room: NEW-2 (the two-line default-resolution guard) is the
  only one with a behavioural consequence; NEW-1 and NEW-3 are the two that a user actually
  reads. NEW-4 is worth doing before `/verify` reads the plan.
- The tech-debt row proposed in the initial review is **withdrawn**: LOW-6 was fixed in
  code rather than deferred, so there is nothing to record in `docs/tech-debt/README.md`.
- Known gaps unchanged: this is diff quality only. Acceptance criteria, static analysis,
  and the behavioural test run belong to `/verify` and `/test`.

---

## Cycle 2

- Date: 2026-09-19
- Range reviewed: `git diff 3e43a3c..HEAD`, HEAD `fea7fc9` (21 commits, 18 files,
  +1582/-137), with `git diff main...HEAD` as context.
- Trigger: cross-review cycle 1 found AR-1, WC-1, WC-2; the user chose to fix all
  three and re-run the pipeline. This is the cycle-2 self-review, the last automatic
  run under the default cap.
- Verdict: **merge**. 1 MEDIUM, 7 LOW. No CRITICAL, no HIGH.

### Evidence reviewed

- `internal/cli/doctor_codex_writable_root.go` re-read in full (893 lines), the test
  file (1562 lines), the unix test file, and every docs / plan / report / `.gitallowed`
  hunk in the range.
- New logic cross-checked against the producers: `internal/org/envelope.go`
  (`ValidateSpawnEnvelope`, `modelInPool`, `modelAllowedForRole`),
  `internal/org/permissions.go`, `internal/config/config.go`.
- **Role/model matrix, seven configurations**, each driven through
  `codexSeatModesPossible` and, independently, through `org.ValidateSpawnEnvelope` for
  three role names x two models as an oracle. Included the lead's three requested cases:
  an `[org.roles]` entry present but empty for the role; a role listed in `[org.roles]`
  but absent from `[org.permissions.roles]`; and a pool whose codex entries are excluded
  for every listed role while an unlisted role could still use them. **All seven agree
  with the oracle**, including the two directions WC-2 was about.
- **Implicit-root matrix, five configurations** of the two exclusion keys against
  `TMPDIR` set to and differing from the injected fixed temp root.
- **Symlink matrix** for the protected-directory rule: a store genuinely under
  `.agents`, and a store reached through a `.agents` symlink whose target is outside it.
- `TMPDIR=/tmp go test ./internal/cli/ -count=1` → `ok` (35.6s), confirming the
  hermeticity fix in `91a5542`.
- **Secret scanner probed with seven crafted lines** through
  `./scripts/secret-scan.sh --file`, plus a scan of `.gitallowed` itself and of the
  three reports that prompted the new allowlist entry (all four clean).
- All probe files removed; `git status --porcelain` is empty.

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | security | **C2-1. The protected-directory rule is applied only to the resolved spelling, so a symlink that leads out of a protected directory turns AR-1's warn back into a `pass`.** This is the one direction the triage says the check must never get wrong. | `internal/cli/doctor_codex_writable_root.go:593-601` resolves both sides with `resolveNearestExisting` and then calls `pathCrossesCodexProtectedDir` on the resolved pair only. Probed: with the agmsg home spelled through a `.agents` symlink whose target is a sibling directory, and the home directory as the sole writable root, the check returns **pass** — while the control (the store genuinely under `.agents`) correctly warns and names the blocked ancestor. The same probe prints the two verdicts side by side: the test on the configured spelling is `true`, on the resolved spelling `false`. The rule's own comment (`:487-494`) states the opposite policy — "not documented precisely, so … treats any occurrence between root and target as protected — the warn-biased reading". Codex's behaviour here (path-string or realpath) is undocumented, so the warn-biased reading should cover both spellings. | In `coveringWritableRoot`, treat a root as blocked when `pathCrossesCodexProtectedDir` is true on **either** the resolved pair or the cleaned configured pair. I checked the three legitimate shapes this must not break — root equal to the store, root equal to the agmsg home, root equal to the `.agents` directory itself — and all three produce a relative path with no protected element on both spellings, so ORing adds only the symlink case. |
| LOW | typo | **C2-2. The implicit-root pass names the wrong exclusion key when the seat's temp directory equals codex's fixed temp root and the fixed root is excluded — it tells the operator a key "is not set" that they did set.** | `internal/cli/doctor_codex_writable_root.go:647-652` derives the key by comparing the matched root against `slashTmpDir`, but `codexImplicitWritableRoots` (`:628-640`) can return that same path from the `TMPDIR` branch after the fixed-root branch was skipped. Probed: with the fixed-root exclusion set true and `TMPDIR` equal to the fixed temp root, the implicit-roots list contains exactly one entry (from `TMPDIR`), yet the Detail reads "…which workspace-write keeps writable by default (`exclude_slash_tmp` is not set in `<cfg>`)". The converse case (the other exclusion set, `TMPDIR` equal to the fixed root) names the key correctly. | Have `codexImplicitWritableRoots` return the key alongside each root, or pass `cfg` to `codexImplicitRootKey` and return `exclude_tmpdir_env_var` whenever the fixed-root exclusion is set. Either removes the identity guess the long doc comment currently defends. |
| LOW | security | **C2-3. The new secret-scan allowlist entry is well-narrowed on its own terms, but `is_allowed` exempts the WHOLE line, so any line containing the hygiene idiom is exempt from every scanner rule.** | The rule is at `.gitallowed:21`; `scripts/secret-scan.sh`'s `is_allowed` tests the entire matched line against each allowlist expression and suppresses the finding on a match. I probed seven crafted lines: the intended idiom is allowed; a backticked key name next to a real-looking value is still blocked; a value inside the backticks is still blocked; a real assignment with no idiom is blocked; the idiom without the trailing "-shaped" is blocked — all as the fix intended. But a line carrying the idiom **and** an AWS-access-key-shaped literal produced no finding at all. This is not hypothetical here: the two report lines the entry was added for (`docs/reports/test-2026-09-19-…md:113` and `docs/reports/verify-2026-09-19-…md:33`) are ~250-character prose lines that are now wholly exempt. | The exemption is inherent to `is_allowed`'s line scope, not to this entry, and narrowing it further inside one expression is not practical. Cheapest honest fix: one sentence in the `.gitallowed` comment block recording that an allowlisted line is exempt from every rule, so the next author keeps hygiene sentences on their own short line. |
| LOW | maintainability | **C2-4. The new tech-debt row's "Two further limits are unverified" enumeration is falsified by this same PR, which added a third.** | `docs/tech-debt/README.md:131` lists exactly two unverified limits (whether codex expands a non-absolute root; whether a guarded seat inherits the user config's sandbox mode). The protected-directory rule added in `ab09dbc` is a third of the same kind: `internal/cli/doctor_codex_writable_root.go:487-494` says whether codex protects only a root's top-level entries or every nested occurrence "is not documented precisely", and the code picks the warn-biased reading. A closed-list phrasing that the same commit invalidates is the pattern this register has been bitten by before. | Change "Two further limits" to an open phrasing and add the any-element reading as a third item, naming `pathCrossesCodexProtectedDir`. |
| LOW | typo | **C2-5. `codexWritableRootDetail`'s doc claims the absent-config form "never carries a blockedAncestor (an absent config has no roots to speak of)", but implicit roots are still evaluated when the config is absent, and the note is then silently dropped.** | `internal/cli/doctor_codex_writable_root.go:727-728` makes the claim; `:736-741` is the `!exists` branch, which ignores its `blockedAncestor` argument. With the config absent, `cfg` is the zero value, so `codexImplicitWritableRoots` still returns the temp roots. Probed with a store under a `.agents` directory beneath the injected fixed temp root: the check computes a blocked ancestor and the Detail omits it. The verdict (warn) is right; only the explanation the operator would act on is missing. | Either include the clause in the absent-config sentence, or reword the comment to say the absent-config form deliberately drops it because there is no `[sandbox_workspace_write]` table to point at. The comment as written asserts something the code does not guarantee. |
| LOW | maintainability | **C2-6. The test that proves a genuinely covering root beats a blocked ancestor asserts something true either way.** | `internal/cli/doctor_codex_writable_root_test.go:428-443` checks `strings.Contains(r.Detail, store)`, but the pass Detail always ends with "covers the agmsg store `<store>`". I rendered the regression it claims to catch by calling `codexWritableRootDetail` with the home directory as the root: the assertion still passes. Only the `Status != "pass"` check discriminates, and it would not catch the blocked ancestor being named as the covering root. | Assert `strings.Contains(r.Detail, "writable root "+store+" in ")`, or assert the blocked home path is not named as the root. |
| LOW | typo | **C2-7. A test's doc comment opens with two sentences that both introduce it.** | `internal/cli/doctor_codex_writable_root_test.go:1157-1158`: "TestCodexImplicitWritableRoots is cross-review WC-1's pure unit test." immediately followed by "TestCodexImplicitWritableRoots is a pure test of codexImplicitWritableRoots' string logic …". A second doc sentence was added without merging the first. | Merge into one sentence. |
| LOW | readability | **C2-8. `checkCodexAgmsgWritableRoot` is now 86 lines, above the repository's 50-line function guideline, and recomputes the codex model list twice per run.** | The function body runs `internal/cli/doctor_codex_writable_root.go:809-894`; `code-review.md`'s checklist asks for functions under 50 lines. `codexModelPoolModels(orgCfg)` is called at `:821` for the early return and again inside `codexSeatModesPossible` at `:346`. | Informational. The body is a flat sequence of guard clauses with every non-trivial step already extracted into a named, table-tested helper, which is the readable shape for a nine-outcome check; splitting it further would hide the outcome order the doc comment enumerates. Passing the already-computed model list into `codexSeatModesPossible` would remove the duplicate work and the redundant empty-list guard inside it. |

### Checked and clean

- **Cross-review AR-1, WC-1, WC-2 are each addressed at the producer, not in prose.**
  The role/model gate agrees with `ValidateSpawnEnvelope` on all seven matrix
  configurations; the protected-directory rule warns for `.git`, `.agents`, and
  `.codex` and correctly does not fire when the root *is* the protected directory or
  the agmsg home; the implicit temp roots flip pass/warn with each exclusion key.
- **`codexModelPermittedForRole` mirrors `modelAllowedForRole` exactly**, including the
  nil-map case: the sibling's `len(cfg.Roles) == 0` early return is subsumed by the
  lookup-on-nil-map returning not-ok, so the two agree for every input. It correctly
  generalizes "is this model allowed" to "is any codex model allowed".
- **Hermeticity.** The only `/tmp` literal in an assertion is in the unit test for the
  production resolver, which is deterministic and independent of the machine's
  temp directory. Every integration test injects a fixture path through the seam, and
  `TMPDIR=/tmp go test ./internal/cli/ -count=1` passes. The seam comment explains why,
  naming the CI runner case.
- **No duplicated pane-environment sentence.** Probed the case the guard exists for — a
  temp-directory-derived implicit root together with a store-location override, with and
  without a configured profile — and the clause appears exactly once in both.
- **The numbered outcome list in the check's doc comment matches the code's order**,
  all nine branches, including the two new ones inserted as outcomes 3 and 8.
- **Docs.** The recipe and the four skill mirrors are byte-identical to their
  counterparts, and every claim in them matches the code: the model-eligibility
  condition, both pool gates, the broad-root caveat, and the two implicit roots with
  their exclusion keys. Skill paragraph measures 71-78 display columns, inside the
  file's norm. The evidence-file addendum correctly says no live re-verification was
  done.
- **Secrets.** `.gitallowed`, and the three reports whose hygiene sentences prompted it,
  all scan clean; the allowlist comment no longer self-matches the scanner pattern.
- **Debug code, swallowed errors, path traversal, unbounded reads, termination:**
  unchanged from cycle 1 and still clean. The check still never spawns a process and
  never returns `fail`.

### Positive notes

- The WC-1 follow-up is the best part of this round: the implementer found that a
  hard-coded fixed temp root made roughly a dozen tests flip on any machine whose own
  temp directory equals it, and fixed the *class* by routing that path through the same
  seam as the rest of the environment. The seam comment names the CI runner where it
  would have bitten.
- Every new decision function is pure and table-tested: the protected-element rule has
  eleven cases including two lookalike names, the implicit-root builder nine, the
  model-eligibility gate its own table mirroring the producer's shape.
- The reason-numbering fix from the revalidation round held up: with both reasons firing
  the clause now reads "(1) … and (2) …", and a single reason is left unnumbered.
- The new unreadable-config test asserts the config path appears exactly once, which is
  the assertion that actually pins the "reason must not repeat the path" contract.

### Recommendation

- **Merge: yes.** No CRITICAL or HIGH. C2-1 is the only finding that changes a verdict
  the operator sees, and its fix is a single additional condition in one `if`.
- Fix order if the cap is not raised: C2-1 (false pass), then C2-2 (false sentence in a
  pass), then C2-4 and C2-5 (two claims the code does not support), then C2-3, C2-6,
  C2-7. C2-8 is informational and needs no change.
- Known gaps: diff quality only. Acceptance criteria, static analysis, and the test run
  are `/verify`'s and `/test`'s, all of which already reported PASS for the pre-cycle-2
  state; the cycle-2 code changes post-date those reports.
