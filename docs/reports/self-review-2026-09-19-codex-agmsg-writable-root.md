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
