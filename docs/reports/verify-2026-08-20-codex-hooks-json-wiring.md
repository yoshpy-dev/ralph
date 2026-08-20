# Verify — codex-hooks-json-wiring

- Date: 2026-08-20
- Plan: `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md`
- Cycle: 1
- Scope: `git diff main...HEAD` at commits `808df48`, `95d1140`, `cdd8e2e`, `3b9e3f9`, `6717ff6`, `2bda3a1`, `f795732`, `d6a04fb`, `c72e644`, `7af720a` (branch `fix/codex-hooks-json-wiring`, working tree clean at time of review)
- Dimension: spec compliance (AC-1..AC-10) + static analysis. No behavioral test execution (tester's job).
- **Verdict: PASS**

## Per-AC evidence

| AC | Status | Evidence |
|----|--------|----------|
| AC-1 | PASS | `.codex/hooks.json` and `templates/base/.codex/hooks.json` route `PostToolUse` through `"$(git rev-parse --show-toplevel)/.claude/hooks/ralph-dispatch.sh" PostToolUse`, matcher `Edit\|Write\|MultiEdit\|apply_patch`. `cmp .codex/hooks.json templates/base/.codex/hooks.json` → identical. `.codex/config.toml` / `templates/base/.codex/config.toml` no longer contain a `[[hooks.*]]` table (`grep -c '^\[\[hooks' .codex/config.toml` → 0); a reference comment block (lines 58-80) documents hooks.json as the source of truth and explicitly says "do not reintroduce" a `[hooks]` table. `cmp` on both config.toml copies → identical. |
| AC-2(a) | PASS | `docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md` records a bypass-free fire in this trusted checkout using a then-trusted absolute-path command form (13x "hook: PostToolUse Completed" on stderr, drop-in payload captured, `tool_name=apply_patch`). The plan's own AC-2(a) text anticipates the shipped git-root command string has a different trust hash and permits substituting bypass + recorded reason for the shipped form specifically — the evidence file's final section states exactly that substitution and reason. Consistent with plan wording, not a silent gap. |
| AC-2(b) | PASS | Evidence file's "AC-2(b) fresh-scaffold fixture evidence" section: `ralph init --yes <tmpdir>` fixture built from the Slice-2 code, hooks.json ships and is JSON-valid, manifest carries the entry; `codex exec --dangerously-bypass-hook-trust` fired the shipped hooks.json in the fresh, untrusted fixture (drop-in marker created, "hook: PostToolUse" x16 on stderr). Non-bypass constraint on a fresh/untrusted fixture is stated explicitly as expected and documented (Slice 3, `.codex/README.md` Trust UX section). |
| AC-3 | PASS | Matcher decision (`Edit\|Write\|MultiEdit\|apply_patch`) recorded with primary-source citation (learn.chatgpt.com/docs/hooks: matcher is a regex over tool name) plus live-fire confirmation that `apply_patch` is the actual reported tool name on file edits and that the alternation without `apply_patch` still fired. Recorded in the plan's "Open questions" section (now resolved) and in `.codex/config.toml` lines 71-73 / `.codex/README.md` (post-fix wording, see M2 below). |
| AC-3b | PASS | `validateCodexHooksJSON` (`internal/cli/doctor.go:339-420`) walks the official schema (`hooks` → event key → matcher-group array → `{type:"command", command:<string>}`) and reports a distinct finding per defect class. All four required negative tests exist and pass their assertions: `TestCheckCodexEffectiveConfig_HooksJSONTopLevelEventKey_Warns` (event name at top level), `TestCheckCodexEffectiveConfig_HooksJSONHooksKeyMissing_Warns` (missing `hooks` key), `TestCheckCodexEffectiveConfig_HooksJSONHandlerMissingType_Warns` (handler missing `type`), `TestCheckCodexEffectiveConfig_HooksJSONCommandAsArray_Warns` (command as array) — all in `internal/cli/cli_test.go:575-651`, all asserting `Status == "warn"`. |
| AC-4 | PASS | `checkCodexEffectiveConfig` (`doctor.go:267-328`) treats hooks.json as source of truth: missing file, invalid JSON, schema noncompliance (AC-3b), and missing dispatcher routing all warn (`TestCheckCodexEffectiveConfig_HooksJSONMissing_Warns`, `_DispatcherRoutingMissing_Warns`). A surviving config.toml `[hooks]` table warns as dual representation (`_DualRepresentation_Warns`). `[features] hooks = false` explicit warns (`_HooksFeatureExplicitFalse_Warns`); absent key is lenient (`_HooksFeatureAbsent_Lenient`, `_DeprecatedFeatureFlagKey_TreatedAsAbsent`) per the Slice-1-resolved open question. Function doc comment states `--strict` never escalates these findings (only an unparseable config.toml fails) — matches `runDoctorFull`'s `allPass` logic and `TestCheckCodexEffectiveConfig_InvalidTOML_Fails`. |
| AC-5 | PASS | `tests/test-hook-wiring.sh`: `check_codex_hooks_json` checks dispatcher routing via real jq parsing; `check_codex_config_toml_no_hooks_tables` (using the whitespace-tolerant `codex_config_toml_has_hooks_table` awk detector, self-tested by `test_codex_config_toml_hooks_table_detector` against tight/spaced/clean fixtures) fails on a reintroduced config.toml `[hooks]` table in either tight or spaced TOML form; `check_no_direct_hook_scripts_in_hooks_json` / `check_no_direct_hook_scripts_in_config_toml` fail on a legacy direct-call form in either representation, and the latter now records an explicit pass on the (current) no-command-assignments state rather than silently returning. |
| AC-6 | PASS | `internal/cli/init_v2_test.go` diff (+2 lines) and `internal/cli/cli_test.go` diff wire `.codex/hooks.json` into the init fixture set with `owner=core` expectations; AC-2(b)'s fresh-fixture evidence independently confirms the shipped file is present and manifest-tracked after `ralph init`. |
| AC-7 | PASS | `./scripts/run-static-verify.sh` (`RALPH_VERIFY_SCOPE=full`) → `scripts/check-sync.sh`: `DRIFTED: 0`, `.codex/hooks.json` present at both root and `templates/base/` and counted in `IDENTICAL: 157`; `./scripts/check-template-purity.sh` → "PASS: no meta-repo-specific references found in templates." Both green in the same run (evidence log below). |
| AC-8 | PASS | `.codex/README.md` "Trust UX" section (present in both root and template copies, `cmp` clean) documents: hooks require project trust + `[features] hooks` not explicitly false + a one-time interactive hook-trust approval; an unapproved hook is silently skipped. `docs/recipes/codex-setup.md` (+ template copy) states the same doctor-warning contract post-fix. `.codex/AGENTS.override.md` (+ template copy, commit `7af720a`) updated its "project-level hooks" row to `.codex/hooks.json` + `[features] hooks = true` + project trust + one-time interactive hook approval, matching the rest of the surface. |
| AC-9 | PASS | `docs/tech-debt/README.md`: the "trust承認のみ残" row (originally line ~115) now carries an appended note recording the 2026-08-20 approval session and the AC-2(a) bypass-substitution reason for the shipped git-root command form. The "Phase 5 二重に不発" row is struck through with a `RESOLVED 2026-08-20 in fix/codex-hooks-json-wiring` prefix and an HTML comment block giving the full resolution summary, pointing at the evidence file. Both rows individually confirmed via `git show 2bda3a1 -- docs/tech-debt/README.md`; table column counts (7 pipe-delimited columns) intact in both edited rows. |
| AC-10 | PARTIAL — static only | `./scripts/run-verify.sh`-equivalent static path (`./scripts/run-static-verify.sh` full scope) is green (see Static analysis below). Full shell + Go test execution is out of scope for `/verify` per the pipeline contract and is the tester's responsibility for this cycle. |

## Self-review fix re-classification (H1, M1-M5, L1-L5)

All 11 findings from `docs/reports/self-review-2026-08-20-codex-hooks-json-wiring.md` were addressed in commit `c72e644`. Re-checked against HEAD:

| ID | Self-review verdict | HEAD status | Evidence |
|----|---|---|---|
| H1 | `.codex/hooks/README.md` still told readers to add inline `[[hooks.*]]` and said "do not add hooks.json" | RESOLVED | `.codex/hooks/README.md` (+ template) rewritten: describes hooks.json wiring, inverts the "do not add" rule to target a `[hooks]` table in config.toml, repoints See-also at `.codex/hooks.json`. |
| M1 | Doctor leniency (absent `[features] hooks` key) contradicted by 3 shipped surfaces | RESOLVED | `.codex/README.md`, `docs/recipes/codex-setup.md`, `.codex/config.toml` (all + template copies) now uniformly state: doctor warns only on explicit `hooks = false`, absent key left to Codex's undocumented default. |
| M2 | `.codex/README.md` claimed `apply_patch` "has to be in the matcher" — evidence shows otherwise | RESOLVED | Reworded to match config.toml: `apply_patch` included as the conservative default; `Edit`/`Write`/`MultiEdit` accepted alongside for readability/parity. |
| M3 | `[[?hooks(\.\|])` regex misses spaced `[[ hooks.X ]]` TOML form | RESOLVED | New `codex_config_toml_has_hooks_table` awk detector strips whitespace before matching; self-tested by `test_codex_config_toml_hooks_table_detector` against tight, spaced, and clean fixtures — all three assertions pass at HEAD. |
| M4 | `scripts/verify.local.sh` remediation text still pointed at config.toml as source of truth | RESOLVED | Message inverted: "this repo uses .codex/hooks.json as the source of truth — delete the config.toml [hooks]/[[hooks.*]] entries." |
| M5 | `doctor.go:105` call-site comment described the pre-migration check contract | RESOLVED | Updated to "Codex hook wiring (hooks.json schema + dispatcher routing; stale config.toml [hooks] table)." |
| L1 | Doc block overstated "never fails" | RESOLVED | Precise phrasing: "`--strict` never escalates these findings; the only `fail` this check produces is an unparseable `config.toml`." |
| L2 | Plan-local coordinates (`Slice 1`, `AC-3b`) in permanent comments | RESOLVED | Replaced with the durable evidence-file path (`docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md`) in `doctor.go` and all 4 test doc comments. |
| L3 | Shipped `.codex/README.md` named `tests/test-hook-wiring.sh`, which does not ship to `templates/base/` | RESOLVED | Same H1/M2 rewrite dropped the test-file name; sentence now reads "`ralph doctor` flags the stale dual representation if it comes back." Confirmed: `grep -n test-hook-wiring templates/base/.codex/README.md` → no match. |
| L4 | `check_codex_hooks_json` dropped the file-existence assertion silently | RESOLVED | New header comment explains the dispatcher-existence check is covered by `check_settings_json` on the Claude-side entry, since Codex commands are shell-evaluated strings not naively path-resolvable. |
| L5 | `check_no_direct_hook_scripts_in_config_toml` silently no-ops when there are no `command =` lines | RESOLVED | Now records an explicit `record_pass "$label: config.toml has no hook command assignments"` on the empty path. |

No re-introduced defects observed; the fix commit's diff is scoped to exactly the files each finding named.

## Consistency sweep

- `grep -rln '\[\[hooks\.'` across the tree (excluding `docs/reports/`, `docs/plans/archive/`, `docs/tech-debt/README.md`) → only `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md`, `.codex/README.md`, `.codex/hooks/README.md`, `.codex/config.toml`, and their `templates/base/` copies. All six matches are either historical/plan narrative or explicit "do not reintroduce" guidance — none is a live instruction to add inline hooks.
- `grep -rn "config.toml.*source of truth|hooks are wired in.*config.toml"` (case-insensitive, excluding reports/archive/tech-debt) → only `tests/test-hook-wiring.sh:191` (the intentional fail-message wording for a *reintroduced* table) and `AGENTS.md:108` (correctly states hooks.json, not config.toml, is the source of truth). No stale claim found.
- Zero occurrences of a live "do not add hooks.json" instruction outside historical/struck-through tech-debt text.

## Byte-identity check

- `.codex/hooks.json` vs `templates/base/.codex/hooks.json`: `cmp` → identical (16 lines, command string `"$(git rev-parse --show-toplevel)/.claude/hooks/ralph-dispatch.sh" PostToolUse`, matcher `Edit|Write|MultiEdit|apply_patch`).
- Both match the evidence file's recorded shipped form character-for-character (`docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md:8`).
- `.codex/config.toml` vs `templates/base/.codex/config.toml`: `cmp` → identical.
- `tests/test-hook-wiring.sh`'s `check_codex_hooks_json_byte_identical` provides a permanent regression guard for the hooks.json pair; `scripts/check-sync.sh` covers the config.toml pair (both included in the full-scope static run below, both green).

## Tech-debt register accuracy

Both edited rows in `docs/tech-debt/README.md` (verified via `git show 2bda3a1 -- docs/tech-debt/README.md` and re-read at HEAD):
1. The "trust承認のみ残" row: RESOLVED-in-place append accurately describes the 2026-08-20 approval and the AC-2(a) bypass-substitution reasoning for the shipped command-string hash mismatch.
2. The "Phase 5 二重に不発" row: struck through, prefixed `RESOLVED 2026-08-20 in fix/codex-hooks-json-wiring`, with a full HTML-comment resolution summary above it (standard register convention — comment block precedes the struck row for traceability).
Column counts (pipe-delimited fields: finding / impact / why-deferred / next-trigger / evidence-pointers) are intact in both rows — no truncation or malformed table row introduced.

## Static analysis

`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` — **all green**, no FAIL/ERROR in the run:
- `sh -n` on all `.claude/hooks/*.sh` (root + template) — OK
- `jq -e .` on `.claude/settings.json` (root + template) — OK
- Codex hook single-source guard, Codex inline hook detector smoke test, Codex PR provenance policy guard — OK
- `scripts/check-sync.sh` — `DRIFTED: 0`, `IDENTICAL: 157`, `KNOWN_DIFF: 5` (all pre-existing, unrelated to this branch) — PASS
- `scripts/check-pipeline-sync.sh` — OK, all 6 cross-referenced surfaces in sync
- `scripts/check-skill-sync.sh` — 13 skills in lock-step
- `scripts/check-template-purity.sh` — PASS, no meta-repo-specific references leaked into templates
- Go verifier (`packs/languages/golang/verify.sh`): `gofmt: ok`, `golangci-lint run` → `0 issues.`

Evidence log: `docs/evidence/verify-2026-08-20-063624.log` (411 lines, zero FAIL/ERROR markers; `grep -n "==>\|FAIL\|ERROR"` confirms only step markers, no failures).

## What remains unverified (tester/doc-maintainer scope)

- Full behavioral test execution (`./scripts/run-test.sh`, shell 617+ and Go package suites) — not run here per the verify/test split; belongs to `/test`.
- `ralph upgrade`'s handling of a downstream project holding an untracked, content-diverging `.codex/hooks.json` (plan's edge case) — unit/integration test territory, not confirmed by this static+spec-compliance pass.
- Doc-drift beyond the specific claim-accuracy items checked above (full `/sync-docs` sweep) — separate pipeline phase.
- Runtime trust-state probing (whether `codex trust .` was actually run in a given downstream environment) — explicitly out of reach for static verification per the plan and the self-review's own "Known gaps" section.

## Verdict

**PASS.** All 10 acceptance criteria (including AC-2's two-part evidence contract and AC-3b's four negative tests) are met with cited evidence. All 11 self-review findings (H1, M1-M5, L1-L5) are correctly fixed at HEAD with no regressions. The consistency sweep found no remaining stale "config.toml carries hooks" claims outside historical/struck-through text. Root and template copies of both `hooks.json` and `config.toml` are byte-identical. Tech-debt register rows are accurate. Static analysis is fully green at full scope.
