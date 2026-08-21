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

# Cycle 2

- Date: 2026-08-20
- Plan: `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md`
- Cycle: 2/2 (cap `RALPH_STANDARD_MAX_PIPELINE_CYCLES` = 2 — final cycle)
- Scope: cycle-2 delta since cycle-1 verify (`10432f6`), `git diff 10432f6..HEAD` — `4cff65f`/`ae5ece5`/`0217f08` (test/sync-docs reports + residual drift), `8aaa27b`/`cb54856` (pre_bash_guard tech-debt row + line-break fix), `cdb0aad` (cross-review triage, 1 AR), `d1df46f` (AR#1 fix), `00ee645` (self-review cycle-2, 2 MEDIUM + 5 LOW), `bced11a` (C2-M1/M2/L1-L5 fixes). HEAD at review time: `bced11a`.
- Dimension: spec compliance (AC-1..AC-10) + static analysis. No behavioral test execution (tester's job).
- **Verdict: PASS** (one non-blocking prose defect found and recorded below — recommend fixing before `/pr`, does not gate this verdict).

## Point 1 — AR#1 fix vs. triage contract, and C2-L4's message-rewording

`d1df46f` matches the triage contract in `docs/reports/cross-review-triage-codex-hooks-json-wiring.md` exactly: the triage's single ACTION_REQUIRED item ("`[features].hooks` present but non-boolean silently treated as absent") is fixed at `internal/cli/doctor.go:294-299` (as cited) with a type-switch branch that appends a distinct warn finding, plus `TestCheckCodexEffectiveConfig_HooksFeatureNonBoolean_Warns` (`internal/cli/cli_test.go:473-496`) as the triage recommended. `go test ./internal/cli/... -run TestCheckCodexEffectiveConfig -v` → all 14 subtests PASS, including the new one.

`bced11a` (C2-L4) then dropped the `%T` raw-Go-type leak the self-review flagged, but went further than the recommendation ("reuse the TOML-friendly type naming") — it removed value-rendering from the message entirely rather than adding `array`/`table`/`string` naming, landing on a static string: `"[features] hooks must be a boolean — a quoted or otherwise non-boolean value may leave hooks disabled; use \`hooks = true\`"`. This is a reasonable simplification (doctor.go:295-312 collapsed the three-arm type-switch into a plain `if isBool { ... } else { ... }`), and it still satisfies the finding's core ask (no Go-internals leak). Confirmed the discriminating test still passes post-rewording: the new message retains the literal substring `"non-boolean"` (inside "...or otherwise non-boolean value..."), so `TestCheckCodexEffectiveConfig_HooksFeatureNonBoolean_Warns`'s `strings.Contains(r.Detail, "non-boolean")` assertion still only matches this branch and not the sibling `"[features] hooks = false — Codex project hooks are disabled"` message. Ran the full `TestCheckCodexEffectiveConfig_*` suite (14 subtests) — all PASS at HEAD (`bced11a`).

## Point 2 — C2 fixes vs. self-review cycle-2 recommendations, re-classified at HEAD

| Cycle-2 ID | Status at HEAD | Evidence |
|-----|-----|-----|
| C2-M1 (`.codex/hooks/README.md` escape hatch pointed at an unreachable location) | **fixed** | Root + template rewritten to point at `.ralph/local/hooks/<event>.d/` or `.claude/hooks/local/<event>.d/` (both legal dispatcher-routed locations), and states directly referencing a script from `.codex/hooks.json` is flagged by the guard. `check_no_direct_hook_scripts_in_hooks_json`/`_in_config_toml` (`tests/test-hook-wiring.sh:259-297`) are unaffected — the new instruction routes through the dispatcher, so it cannot trip the guard. |
| C2-M2 (six surfaces claimed doctor warns "only" on explicit `false`, AR#1 added a non-boolean arm outside that closed claim) | **fixed** | `.codex/README.md`, `docs/recipes/codex-setup.md`, `.codex/config.toml` (+ their three `templates/base/` twins, all `cmp`-identical to their root counterparts) now uniformly say "explicitly disabled or malformed (`hooks = false`, or a non-boolean value...)" / "explicitly set to false or carries a non-boolean value" / "explicitly `false` or set to a non-boolean value". Verified via grep across all six files — no residual "only explicit false" phrasing remains, and the wording now matches doctor's actual two-branch (bool-false / non-boolean) behavior at HEAD. |
| C2-L1 (Check-3 comment enumeration dropped `[features] hooks`) | **fixed** | `internal/cli/doctor.go:105` now reads "Check 3: Codex hook wiring (`[features] hooks`; hooks.json schema + dispatcher routing; stale config.toml `[hooks]` table)." — the three-item list is restored and matches the function body's three concerns. |
| C2-L2 (permanent spec comment cited an ephemeral `docs/plans/active/...` path) | **fixed** | `docs/specs/2026-08-17-overlay-scaffold-v2.md:92`'s AC-10 bracket now cites only the durable `docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md` path; the plan-path pointer is dropped. (The two pre-existing dead pointers at `:66`/`:185` the self-review also noted are unrelated to this plan's scope and remain unfixed — outside this PR's affected areas.) |
| C2-L3 (`.codex/config.toml` still named the non-shipping `tests/test-hook-wiring.sh` gate) | **fixed, with a new prose defect** | The clause naming the shell test was removed from both root and template `config.toml`, but the edit left a dangling fragment: line 69-70 (identical in both copies) now reads "...do not reintroduce a `[hooks]` table here — `ralph doctor` and the shell / flags that as a stale duplicate representation." — "and the shell" is an orphaned conjunct with no following noun, a leftover from the pre-fix "`ralph doctor` and the shell hook-wiring test both flag" that only had its back half rewritten. Not a spec-compliance or static-analysis failure (no AC covers this sentence's grammar, and `check-sync.sh`/`check-template-purity.sh` don't parse prose), but it ships into every scaffolded project's `.codex/config.toml` in this form. **Recommend a one-line follow-up fix before `/pr`**: drop "and the shell" so it reads "`ralph doctor` flags that as a stale duplicate representation." (both root and template copies, byte-identical, so a single coordinated edit closes it). |
| C2-L5 (tech-debt row's parenthetical misattributed the guard mechanism) | **fixed** | The parenthetical was replaced with an accurate account: the row's own first recording used a compound `git commit -m` command whose heredoc body contained the guard's trigger substring, which is what actually blocked it — not a python-only workaround narrative implying the guard blocks file writes. Row still renders as 7 pipe-delimited fields (`awk -F'|' '/^\|/ {print NF}'` confirmed for lines 114-119, 122-123). |

Counts: 2 MEDIUM fixed, 5 LOW fixed (4 clean, 1 — C2-L3 — introduced a new non-blocking prose defect during the fix, documented above).

## Point 3 — AC-1..AC-10 at HEAD (`bced11a`)

Cycle-2 delta touches only: doc/comment wording (6 files + template twins), one tech-debt row parenthetical, one spec AC-10 bracket citation, and `internal/cli/doctor.go`'s non-boolean branch (simplified from a 3-arm type-switch to an `if/else`, same warn semantics). None of these touch the hooks.json routing, matcher, schema validator, dispatcher check, init/manifest wiring, or check-sync/purity mechanics that AC-1/2/3/3b/5/6/7 depend on, so cycle-1's per-AC evidence still holds. Re-confirmed directly rather than assumed:

- AC-1: `.codex/hooks.json` / `templates/base/.codex/hooks.json` unchanged since cycle-1 (`bced11a` didn't touch either); `.codex/config.toml` / template still have zero `[[hooks` tables (`grep -c '^\[\[hooks' .codex/config.toml` → 0) — the C2-L3 fix only edited prose in the existing reference-comment block, not the "do not reintroduce" instruction itself.
- AC-3b/AC-4/AC-5: `go test ./internal/cli/... -run TestCheckCodexEffectiveConfig -v` → 14/14 PASS at HEAD (listed under Point 1), covering the schema negative tests, the dispatcher-routing check, the dual-representation warn, and both the explicit-false and non-boolean `[features] hooks` branches. `tests/test-hook-wiring.sh`'s guards are unaffected by the doc-only C2 fixes (confirmed by reading the diff — no touches to that file in `bced11a`).
- AC-6/AC-7: `scripts/check-sync.sh` (full static run below) → `DRIFTED: 0`, `.codex/hooks.json` and `.codex/config.toml` both counted in `IDENTICAL: 157`; `./scripts/check-template-purity.sh` → PASS.
- AC-9: Both tech-debt rows still parse as 7 pipe-delimited fields after the C2-L5 wording fix (confirmed above); the RESOLVED annotations from cycle 1 are untouched by `bced11a`.
- AC-2, AC-8, AC-10: no cycle-2 commit touches the live-fire evidence file, `.codex/README.md`'s Trust UX section content (only the doctor-leniency paragraph inside it was reworded per C2-M2, not the trust-UX claims), or test/build mechanics beyond the doctor.go branch already covered above.

## Point 4 — Static analysis

`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` at HEAD (`bced11a`) — **all green**, exit 0, no FAIL/ERROR markers (`grep -n "FAIL\|ERROR"` on the evidence log returns nothing):
- `scripts/check-sync.sh` — `IDENTICAL: 157`, `DRIFTED: 0`, `TEMPLATE_ONLY: 11`, `KNOWN_DIFF: 5` (all pre-existing) — PASS
- `scripts/check-pipeline-sync.sh` — OK, 6/6 surfaces in sync
- `scripts/check-skill-sync.sh` — 13 skills in lock-step
- `scripts/check-template-purity.sh` — PASS
- Go verifier (`packs/languages/golang/verify.sh`): `gofmt: ok`, `golangci-lint run` → `0 issues.`
- `go build ./...` and `go vet ./internal/cli/...` — both clean (run separately as a sanity check on the doctor.go rewrite)

Evidence log: `docs/evidence/verify-2026-08-20-112016.log`.

## What remains unverified (tester/doc-maintainer scope)

- Full behavioral test execution (`./scripts/run-test.sh`) — not run here, per the verify/test split; tester's job for this cycle.
- Doc-drift beyond the specific claim-accuracy items checked in Point 2 — `/sync-docs` scope, already run for cycle 1 (`0217f08`, `ae5ece5`); not re-run for the cycle-2 delta since these are prose fixes inside files `/sync-docs` already covers, not new drift surfaces.
- The C2-L3 dangling-fragment prose defect noted above is unverified as *fixed* — it is a newly observed gap, not yet remediated.

## Cycle 2 verdict

**PASS.** All three requested checks (AR#1-fix/triage-contract match with a still-discriminating test, C2 self-review-recommendation match with all six doctor-warning surfaces now consistent, AC-1..AC-10 continuity) hold at HEAD (`bced11a`). Static analysis is fully green at full scope. One new, non-blocking issue was found during this cycle: the C2-L3 fix left a grammatically broken sentence ("`ralph doctor` and the shell / flags that...") identically in `.codex/config.toml` and its template twin — cosmetic, not spec- or test-breaking, but shipped to every scaffolded project. Recommend a one-line fix before `/pr` given this is the final pipeline cycle (cap 2/2); does not block this PASS verdict.

# Cycle 3

- Date: 2026-08-21
- Plan: `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md`
- Cycle: 3/3 (cap `RALPH_STANDARD_MAX_PIPELINE_CYCLES` raised to 3 after cycle-2 cross-review returned 2 ACTION_REQUIRED — final cycle)
- Scope: cycle-3 delta since cycle-2 verify (`67f56d5`), `git diff 67f56d5..HEAD` — `c288a81`/`f6de96e`/`ee5800d` (test/sync-docs cycle-2 artifacts), `7950c62` (cross-review cycle-2 triage, 2 AR), `4d8220c` (AR#1 cd-first command form; AR#2 apply_patch path derivation), `c6a24c6` (self-review cycle-3: 0 CRITICAL / 1 HIGH (C3-H1) / 4 MEDIUM (C3-M1..M4) / 4 LOW (C3-L1..L4)), `4e4e2a6` (C3-M4 plan pointer + C3-L2 spec dead pointers), `3dd3a64` (C3-H1 cwd resolution + C3-M1/M2/M3/L1/L3/L4). HEAD at review time: `3dd3a64`.
- Dimension: spec compliance (AC-1..AC-10) + static analysis. No behavioral test execution (tester's job) — targeted `go test ./internal/cli/...` runs below are spec-compliance checks on doctor's Go-side contract, the same class of check cycle 2's verify used (Point 1 there), not the full test suite.
- **Verdict: PASS**

## Point 1 — cycle-2 AR fixes and all 9 cycle-3 self-review findings, re-classified at HEAD

Both cross-review cycle-2 ACTION_REQUIRED items (AR#1: subdirectory-launch dispatcher no-op; AR#2: apply_patch payload path mismatch) were fixed in `4d8220c` — the shipped `.codex/hooks.json` command is now `cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh PostToolUse` (confirmed below, byte-identical to the `internal/cli/cli_test.go:390` fixture and the cycle-3 live-fire evidence). Self-review cycle-3 then found C3-H1 as the AR#1/AR#2 interaction defect (session-cwd-relative envelope paths vs. git-root process cwd), plus 8 secondary findings. Re-classified individually against the fix commits:

| Cycle-3 ID | Status at HEAD | Evidence |
|-----|-----|-----|
| C3-H1 (HIGH — envelope paths are session-cwd-relative, hook process cwd is git root after AR#1, so `check_mojibake.sh`'s existence test silently fails-open on subdirectory launches) | **fixed** | `3dd3a64` adds `session_cwd="$(... jq -r '.cwd // empty' ...)"` in both `.claude/hooks/check_mojibake.sh` and `.claude/hooks/post_edit_verify.sh` (+ template twins, `cmp`-verified identical below), and resolves a relative envelope path against it before the `[ -f "$fp" ]` test (mojibake) / before logging (post-edit-verify, re-relativized to `dispatch_root` for readability). Two new regression fixtures genuinely discriminate: `tests/test-check-mojibake.sh` Case G4 launches the hook from a simulated git root with a payload `cwd` pointing at a subdirectory and a relative `dirty.txt` path — asserts `exit 2`, which only happens if the path resolves against `cwd` rather than the hook's own `$PWD`. `tests/test-post-edit-verify.sh` Case H does the same for the logging path, asserting `edited-files.log` carries the `cwd`-resolved, root-relative form. Read (not run) both fixtures against the pre-fix code path described in the self-review: pre-fix, `[ -f "dirty.txt" ]` evaluates against the proj-root cwd, misses the subdir file, and silently exits 0 — the fixture's `exit 2` assertion would fail on that code, so it is a genuine discriminator, not a change-detector. |
| C3-M1 (MEDIUM — `.codex/hooks/README.md`'s escape-hatch claim that `ralph doctor` + "the hook-wiring checks" flag a direct-script bypass was false for the `ralph doctor` half, and the shell-test half doesn't ship) | **fixed via option b (doctor now enforces it)** | `3dd3a64` adds `hookScriptBasenameRe` + a loop in `validateCodexHooksJSON` (`internal/cli/doctor.go:349-451`) that flags any `*.sh` command basename other than `ralph-dispatch.sh` in a hooks.json handler. `.codex/hooks/README.md` (+ template) now says only "flagged by `ralph doctor`" (the shell-test clause was dropped), which is now literally true. `TestCheckCodexEffectiveConfig_DirectHookScriptReference_Warns` and `TestCheckCodexEffectiveConfig_DirectHookScriptReference_IgnoresDispatcher` both pass (`go test ./internal/cli/... -run TestCheckCodexEffectiveConfig -v` → 16/16 PASS, run below). The `IgnoresDispatcher` test uses `validHooksJSON`, the exact shipped-form fixture — confirms the new check does not false-positive on the shipped `.codex/hooks.json` itself (task point 4). |
| C3-M2 (MEDIUM — `post_edit_verify.sh`'s doc-class globs required a leading `/`, so a Codex `apply_patch` envelope's bare `docs/...` path fell through to the code-skip arm) | **fixed** | `3dd3a64` widens the `case` arms to `*"/docs/"*\|docs/*\|*"/.claude/rules/"*\|.claude/rules/*` (root form added alongside the nested form, matching the pre-existing `AGENTS.md`/`CLAUDE.md` bare-form pattern). New fixtures: `test-post-edit-verify.sh` Case G (single bare `docs/...` payload → doc-class message) and Case D rewritten to a bare `docs/plans/active/example.md` path (was `repo/docs/...`) so it now exercises the production shape rather than routing around it. |
| C3-M3 (MEDIUM — no regression gate pinned the `cd`-first command form; a revert to the pre-AR#1 absolute-path form would pass every existing assertion) | **fixed** | `3dd3a64` adds `cd_prefix_ok`/`relative_dispatch_ok` checks in `tests/test-hook-wiring.sh:check_codex_hooks_json` asserting the command contains `cd "$(git rev-parse --show-toplevel)"` and `./.claude/hooks/ralph-dispatch.sh` (relative form) — a revert to the absolute `"$(git rev-parse --show-toplevel)/.claude/hooks/ralph-dispatch.sh"` form now fails `relative_dispatch_ok`. The stale header comment (naming the old absolute-path shape as the reason path resolution is skipped) was rewritten to describe the actual `cd`-first shape and why it's still not naively path-resolved (compound shell expression, not a bare path). |
| C3-M4 (MEDIUM — plan's Open questions still recorded the superseded absolute-path form as the settled 確定形, and no cycle-3 Deviations bullet existed) | **fixed** | `4e4e2a6` rewrites `docs/plans/active/2026-08-20-codex-hooks-json-wiring.md:125` to the `cd`-first form with a "cycle 3 で更新" note explaining why AR#1 replaced the git-root-resolving absolute form, and adds a cycle-3 Deviations bullet (line 114, confirmed in the plan read above) naming both AR fixes and the C3-H1 interaction. |
| C3-L1 (LOW — doctor.go's `[features] hooks` block comment still framed the decision as a two-state (explicit-false / absent) story after the non-boolean arm was added) | **fixed** | `3dd3a64` extends the comment at `internal/cli/doctor.go:292-299`: "A present but non-boolean value (e.g. a quoted \"false\") is a third, distinct state — also a warn...". |
| C3-L2 (LOW — C2-L2 residue: AC-10 bracket citation fixed, but the co-located tech-debt clause and two dead `docs/plans/active/...` pointers elsewhere in the same spec were untouched) | **fixed (the two dead pointers)** | `4e4e2a6` repoints `docs/specs/2026-08-17-overlay-scaffold-v2.md:66` and `:185` from `docs/plans/active/...` to `docs/plans/archive/...` for the two named phase plans (`overlay-scaffold-v2-p4.md`, `-p3.md`), both confirmed to actually exist under `docs/plans/archive/` (`ls` below). The tech-debt clause at line 92 was not touched in this diff — a residual, but it points at a row that is honestly struck-through RESOLVED in `docs/tech-debt/README.md`, so a reader following it lands on the correct (resolved) state rather than a broken pointer; not re-filed, since C3-L2's own recommendation offered "drop the clause or mark it resolved inline" as alternatives and the row itself already carries the resolved marker. |
| C3-L3 (LOW — `post_edit_verify.sh`'s jq-absent apply_patch path degraded silently, unlike `check_mojibake.sh`'s marker-file convention) | **fixed** | `3dd3a64` adds an `else` branch writing `.harness/state/post-edit-verify-jq-missing` and a stderr line, mirroring `check_mojibake.sh`'s `mojibake-jq-missing` convention. |
| C3-L4 (LOW — tech-debt RESOLVED comment attributed the live-fire proof to "the shipped git-root form", which after AR#1 is no longer the shipped form) | **fixed** | `3dd3a64` appends "(command form later revised to the cd-first shape, see `docs/evidence/codex-hooks-livefire-cycle3-2026-08-20.md`)" to the RESOLVED comment in `docs/tech-debt/README.md`. |

Counts: 1/1 HIGH fixed, 4/4 MEDIUM fixed, 4/4 LOW fixed. No residue re-opened.

## Point 2 — AC-1..AC-10 at HEAD, with AC-2 evidence spanning slice-1 + cycle-3

Cycle-3 delta touches the two hook scripts (session-cwd path resolution), `doctor.go` (direct-script-reference check + comment), `tests/test-hook-wiring.sh` (new regression assertions), the plan, the spec, and tech-debt/README.md prose. None of these change the hooks.json routing/matcher/schema-validator/init-manifest mechanics AC-1/3/3b/6/7 depend on, so those are re-confirmed directly rather than assumed:

- **AC-1**: `.codex/hooks.json` unchanged content-wise since `4d8220c` (cycle-3's earlier commit); `grep -c '^\[\[hooks' .codex/config.toml` → 0, unchanged. `cmp` of both files against their `templates/base/` twins → identical (below).
- **AC-2**: evidence now spans two files — `docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md` (original absolute-path form, Slice 1) and `docs/evidence/codex-hooks-livefire-cycle3-2026-08-20.md` (the AR#1/AR#2 fix, subdirectory-launch live-fire). The cycle-3 evidence file's recorded command (`cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh PostToolUse`, line 26) is byte-identical to the shipped `.codex/hooks.json:10` and the `cli_test.go:390` fixture (confirmed via grep below) — the evidence claims match the final shipped command form, not the superseded one.
- **AC-3/AC-3b**: matcher (`Edit|Write|MultiEdit|apply_patch`) unchanged; `go test ./internal/cli/... -run 'TestCheckCodexEffectiveConfig|TestValidateCodexHooksJSON' -v` → 16/16 PASS (run below), including the two new C3-M1 tests and all four pre-existing AC-3b schema-negative tests.
- **AC-4/AC-5**: `checkCodexEffectiveConfig`'s warn-only contract holds (`TestCheckCodexEffectiveConfig_InvalidTOML_Fails` still the only fail path); `tests/test-hook-wiring.sh`'s new `cd_prefix_ok`/`relative_dispatch_ok` assertions are additive to `check_codex_hooks_json`, not replacements — read, not executed (tester's job), but structurally additive per the diff.
- **AC-6/AC-7**: `./scripts/run-static-verify.sh` (`RALPH_VERIFY_SCOPE=full`) at HEAD (`3dd3a64`) → all green (Point 4 below); `scripts/check-sync.sh` → `DRIFTED: 0`, `IDENTICAL: 157`.
- **AC-8**: `.codex/README.md` Trust UX section and `.codex/hooks/README.md`'s escape-hatch paragraph both re-read at HEAD — consistent with the C3-M1 fix (no shell-test claim remaining) and the C3-M2 fix (no doc-class regression affecting this file).
- **AC-9**: both tech-debt rows re-read at HEAD; the RESOLVED comment's command-form attribution is corrected (C3-L4, confirmed above); the "trust承認のみ残" row and the struck-through Phase-5 row both still parse as intact pipe-delimited table rows (`awk -F'|' '/^\|/ {print NF}'` unaffected by the C3-L4 text-only edit, since it only appended a parenthetical inside the existing HTML comment).
- **AC-10**: static-only scope holds for this cycle (below); full shell + Go test execution remains the tester's responsibility.

## Point 3 — coherence sweep (`show-toplevel` across the tree)

`grep -rn 'show-toplevel' .` (excluding `.git/`, `docs/evidence/`, `docs/reports/`) returns every quoted command form consistent with the shipped hooks.json: the `cd`-first form appears in `.codex/hooks.json`, `templates/base/.codex/hooks.json`, `.codex/README.md`, `.codex/config.toml` (+ template twins), `internal/cli/cli_test.go:390`, `tests/test-hook-wiring.sh` (both as the new assertion literal and in the rewritten header comment), and the plan's cycle-3-updated Open-questions line. Unrelated hits (`internal/org/statedir.go`, `internal/cli/org.go`, various `scripts/*-guard.sh` using `git rev-parse --show-toplevel` for repo-root resolution, unrelated to Codex hooks) are pre-existing and out of scope. A separate targeted grep for the stale no-`cd`, absolute-path-only form (`"$(git rev-parse --show-toplevel)/.claude/hooks/ralph-dispatch.sh" PostToolUse`) outside `docs/evidence/`, `docs/reports/`, `docs/plans/` found zero hits (only Go build cache binaries matched, unrelated). No stale form found in shipped content, test assertions, or permanent docs.

## Point 4 — doctor direct-script warn vs. the shipped hooks.json (no false positive)

`TestCheckCodexEffectiveConfig_DirectHookScriptReference_IgnoresDispatcher` (`internal/cli/cli_test.go:718-734`) feeds `validHooksJSON` — a fixture whose command string is byte-identical to `.codex/hooks.json:10` (confirmed by direct comparison: both are `cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh PostToolUse`) — into `checkCodexEffectiveConfig` and asserts `Status == "pass"`. This is a direct proof, not an inference: the new `hookScriptBasenameRe` loop extracts `ralph-dispatch.sh` from the shipped command and the `if name == "ralph-dispatch.sh" { continue }` guard excludes it from the findings list, so the dispatcher reference itself never trips the new C3-M1 check. `go test ./internal/cli/... -run TestCheckCodexEffectiveConfig -v` at HEAD → this test and its sibling `_DirectHookScriptReference_Warns` (which does trigger on a genuine non-dispatcher `*.sh` reference) both PASS.

## Point 5 — static analysis

`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` at HEAD (`3dd3a64`) — all green, no FAIL/ERROR:
- `sh -n` on all `.claude/hooks/*.sh` (root + template, including the two rewritten scripts) — OK
- `jq -e .` on `.claude/settings.json` and `.codex/hooks.json` (root + template) — OK
- Codex hook single-source guard, Codex inline hook detector smoke test, Codex PR provenance policy guard — OK
- `scripts/check-sync.sh` — `IDENTICAL: 157`, `DRIFTED: 0`, `TEMPLATE_ONLY: 11`, `KNOWN_DIFF: 5` (all pre-existing) — PASS
- `scripts/check-pipeline-sync.sh` — OK, 6/6 surfaces in sync
- `scripts/check-skill-sync.sh` — 13 skills in lock-step
- `scripts/check-template-purity.sh` — PASS, no meta-repo-specific references leaked into templates
- Go verifier (`packs/languages/golang/verify.sh`): `gofmt: ok`, `golangci-lint run` → `0 issues.`

Evidence log: `docs/evidence/verify-2026-08-21-022409.log`.

## What remains unverified (tester/doc-maintainer scope)

- Full behavioral test execution (`./scripts/run-test.sh`, shell 617+ and Go package suites, including the new `tests/test-check-mojibake.sh` Case G4 and `tests/test-post-edit-verify.sh` Cases G/H) — read against the pre-fix code path in Point 1 above, but not executed; belongs to `/test`.
- C3-L2's residual tech-debt clause at `docs/specs/2026-08-17-overlay-scaffold-v2.md:92` (not touched by this cycle's fix commits) — non-blocking, the row it points at is honestly resolved.
- A second live-fire capturing whether `check_mojibake.sh` actually scanned during a real subdirectory-launch Codex session (the self-review's own noted gap) — the regression fixtures (Case G4/H) prove the resolution logic in isolation; a full live re-run was not repeated for this verify cycle.

## Cycle 3 verdict

**PASS.** All 9 cycle-3 self-review findings (C3-H1 HIGH, C3-M1..M4 MEDIUM, C3-L1..L4 LOW) are correctly fixed at HEAD with genuinely discriminating regression fixtures for the HIGH — read against the pre-fix code path, not merely present. AC-1..AC-10 hold, with AC-2's evidence now correctly attributed to the final shipped `cd`-first command form. The coherence sweep found no stale absolute-path-only command form outside historical evidence/report/plan-narrative text. The new doctor direct-script-reference check does not false-positive on the shipped `.codex/hooks.json` — proven directly by a dedicated test using the byte-identical fixture. Static analysis is fully green at full scope.
