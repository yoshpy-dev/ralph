# Verify report: overlay-scaffold-v2 Phase 2

- Date: 2026-08-17
- Plan: `docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- Verifier: `verifier` subagent (Claude Code, Sonnet 5)
- Scope: spec compliance (AC-1 through AC-11) + static analysis via
  `./scripts/run-static-verify.sh`, on `feat/overlay-scaffold-v2-p2`, base
  `main`, HEAD `c2d501e` (126 files changed vs. `main`). No behavioral test
  execution beyond what static analysis and the plan's own AC-8 explicitly
  name — the full behavioral suite is `/test`'s job.

## Verdict: PASS

No spec-compliance gaps and no static-analysis failures. One pre-existing,
self-review-acknowledged test-coverage gap noted below (non-blocking).

## Self-review fix-commit crosscheck

Commit `c2d501e` ("fix: address phase-2 self-review findings") was checked
line-by-line against `docs/reports/self-review-2026-08-17-overlay-scaffold-v2-p2.md`'s
findings table:

| Finding | Fixed in c2d501e? | Evidence |
| --- | --- | --- |
| HIGH — `org_test.go:850,902` stale negative-guard substrings | Yes | Both now check `.claude/rules/ralph/agent-messaging.md` (the string the templates actually emit today), restoring the guards' ability to fire |
| MEDIUM — dispatcher EXIT-only trap leaks payload on signal kill | Yes | `trap cleanup EXIT INT TERM HUP` added, moved above the first `mktemp` so an early failure doesn't leak |
| MEDIUM — Codex hook surface not migrated to dispatcher | Deferred (tech debt row, correctly not silent) | `docs/tech-debt/README.md` new row: "The Codex hook surface... was not migrated to ralph-dispatch.sh in Phase 2" |
| MEDIUM — `.gitignore` block runs to EOF, no stated free area | Partially fixed + tech debt row | `.gitignore`/`templates/base/.gitignore` gained a trailing `# Project-specific ignores go below this line.` signpost after `END RALPH MANAGED`; tech-debt row still correctly notes the block itself still ends at EOF (mitigation, not full closure) |
| MEDIUM — `check-pipeline-sync.sh` comment claims a mirror with the canonical doc's own reference list that doesn't hold (`CLAUDE.md`) | Yes | `CLAUDE.md` removed from both the canonical doc's "Where this order is referenced" list and the script's `REFS`; comment and code agree again |
| MEDIUM — `lib_json.sh` sourced but unused, widens `set -e` blast radius | Yes | `. "$HOOK_DIR/lib_json.sh"` and the `HOOK_DIR` var removed; header comment rewritten to describe the actual `command -v jq` probe |
| MEDIUM — `test-hook-wiring.sh` grep not anchored to `command = [...]` lines | Yes | `grep -E '^[[:space:]]*command[[:space:]]*=' "$config" \| grep -oE '"\./[^"]+"'` |
| LOW — `ownerForScaffoldPath` uses OS separator against slash-keyed manifest paths | Yes | `filepath.ToSlash(relPath)` added with a doc comment explaining why |
| LOW — unquoted `$event` word-splits/glob-expands | Yes | `case "$event" in *[!A-Za-z]*) ... exit 2 ;; esac` guard added |
| LOW — `awk` blank-collapse duplicated in both merge branches | Not fixed | Still duplicated verbatim; cosmetic, correctly left as a LOW follow-up |
| LOW — `cp`'d single-script output inherits umask instead of 0600 | Yes | `chmod 600 "$first_output"` added |
| LOW — `check-sync.sh` `ROOT_ONLY` label misnames the missing-`templates/base/AGENTS.md` case | Not fixed | Still prints `ROOT_ONLY templates/base/AGENTS.md (file missing)`; cosmetic, correctly left |
| LOW — block-aware `AGENTS.md` comparison double-counts the shared `identical` counter | Not fixed | Correctly left; cosmetic |
| LOW — free-area heading renders as an H3 subsection of "Hard rules", stale "Steps 4–7" cross-reference | Yes | Promoted to `## Org runtime pointers...`, reworded to name the steps instead of referencing block-internal numbering |
| LOW — 8 `<event>.d` shims don't forward `"$@"` | Yes | All 8 root + 8 template shims now `exec ... "$@"` (spot-checked `.claude/hooks/PostToolUse.d/10-post-edit-verify.sh`) |

Every fix matches its finding's recommendation; nothing was overclaimed. The
two MEDIUM items intentionally left open (Codex wiring, `.gitignore` block
extent) are recorded as new, accurately-worded rows in
`docs/tech-debt/README.md` rather than silently dropped — consistent with
the self-review's own recommendation to defer them. Two additional rows were
also added during Slice 6 for issues discovered during implementation
(a `pre_bash_guard.sh` payload-shape mismatch under `jq`, and the legacy
upgrade engine's lack of rename detection for the pack-rule path move) — both
correctly scoped as out-of-scope-for-Phase-2 discoveries, not silent gaps.

## Acceptance criteria

| AC | Verdict | Evidence |
| --- | --- | --- |
| AC-1 (fresh `ralph init` generates v2 layout) | PASS | `internal/cli/init_v2_test.go:TestExecuteInit_V2_FreshInit_LayoutAndOwners` asserts `Meta.Layout == "v2"`, per-path owners, and a well-formed `AGENTS.md` block; passed in the full test run (see Static analysis section) |
| AC-2 (manifest v3, owner attribution) | PASS | Same test asserts `AGENTS.md`/`.gitignore` = block, `CLAUDE.md`/`ralph.toml`/`docs/**`/`.github/workflows/verify.yml` = seed, rest = core; `internal/cli/init.go`'s `ownerForScaffoldPath` implements the mapping |
| AC-3 (existing-file init: seed untouched, block appended, block-exterior bytes preserved) | PASS | `TestExecuteInit_V2_PreExistingFiles_AppendsBlockPreservesSeed` — byte-exact `CLAUDE.md`, `AGENTS.md`/`.gitignore` = original prefix + appended block, `hasPrefixBytes` assertion; `TestExecuteInit_V2_MalformedBlock_LeftUntouchedWithWarning` covers the malformed-block edge case named in the plan's Test plan |
| AC-4 (dispatcher order + stdin + exit propagation + JSON merge semantics on real payloads) | PASS, with one noted gap | `tests/test-ralph-dispatch.sh` Cases A–H cover: PreToolUse `permissionDecision` deny-wins-and-short-circuits (A), PostToolUse `additionalContext` aggregation (B), byte-exact single-script passthrough (C), non-zero exit propagation with later scripts skipped (D), stdin reaching every script (E), core→`.ralph/local/hooks` ordering (F), missing-dir exit 0 (G), and run-verify.sh/run-static-verify.sh/run-test.sh mode-selecting drop-in execution (H, = AC-5). Dispatcher code (`dirs="./.claude/hooks/${event}.d ./.ralph/local/hooks/${event}.d ./.claude/hooks/local/${event}.d"`) wires all 3 layers, but only the first two are exercised by an ordering test — the third (`.claude/hooks/local/<event>.d/`, gitignored) layer's *ordering* is untested (self-review already flagged this as a known gap, not a regression from the fix commit) |
| AC-5 (`run-verify.sh`/`run-test.sh` drop-in extension points, fixture-tested) | PASS | `tests/test-ralph-dispatch.sh` Case H1–H4: `run-static-verify.sh` runs only `verify.d`, `run-test.sh` runs only `test.d`, plain `run-verify.sh` (mode=all) runs both, and a failing `verify.d` drop-in fails the run |
| AC-6 (pack rule path `→ .claude/rules/ralph/<lang>.md`, owner=core, `check-sync.sh` pack_rule_source updated) | PASS | `internal/cli/language_pack.go:packRuleRelPath` returns `.claude/rules/ralph/<pack>.md`; `TestExecuteInit_V2_LanguagePack_RuleUnderRalphSubdir` asserts the new path exists, the old path does not, owner=core, and doctor's pack check still passes; `scripts/check-sync.sh`'s `pack_rule_source()` matches `.claude/rules/ralph/*.md` |
| AC-7 (block-aware `check-sync.sh`, `AGENTS.md` out of `KNOWN_DIFFS`, DRIFTED=0) | PASS | `./scripts/check-sync.sh` run from the worktree: `KNOWN_DIFF` list is now `{model-routing.md, verify.yml, CLAUDE.md}` (no `AGENTS.md`); Sync Summary `DRIFTED: 0`, `PASS: all files in sync.`; `docs/tech-debt/README.md`'s prior `AGENTS.md` KNOWN_DIFFS row is marked `RESOLVED 2026-08-17` |
| AC-8 (meta-repo harness works under v2: `run-verify.sh` exit 0 incl. drop-ins + hook command smoke) | PASS | `./scripts/run-verify.sh` (full mode, run inside the worktree) exited 0 — "All verifiers passed", evidence `docs/evidence/verify-2026-08-17-101740.log`; hook-command executability is asserted by `tests/test-hook-wiring.sh` (part of the same green run) which now anchors its `.codex/config.toml` grep to `command = ` lines |
| AC-9 (existing shell+Go suite passes; upgrade wiring change limited to pack-path + v2 guard) | PASS | Same full `run-verify.sh` run: all shell test files `OK`, `go vet`/`golangci-lint` = `0 issues.`, `go test ./...` all `ok`/cached; `internal/cli/upgrade.go`'s only new logic is the `LayoutV2` fail-closed guard plus the pack-rule path constant follow-through — confirmed by reading `runUpgradeIO`'s diff-computation path, unchanged apart from the guard |
| AC-10 (legacy `ralph upgrade` fail-closed on `meta.layout="v2"`, zero writes, names Phase 3) | PASS | `internal/cli/upgrade.go:185` guard runs before any diff computation, applies to both `--force` and `--dry-run`; `TestRunUpgradeIO_V2Layout_FailsClosedWithoutWrites` asserts the error mentions "v2" and "Phase 3" and that `snapshotDirHashes` before/after are equal for both code paths |
| AC-11 (no meta-repo-specific content in `templates/base/AGENTS.md`) | PASS | `templates/base/AGENTS.md` contains no `internal/`, `cmd/ralph`, or "Repo map" text; `diff templates/base/AGENTS.md .ralph/core/AGENTS.core.md` shows only the expected non-managed header difference; block-aware `check-sync.sh` (AC-7) independently proves the managed-block interior is byte-identical to `.ralph/core/AGENTS.core.md` in both files |

## Static analysis

Two runs, both from inside the task worktree (a first attempt was
mistakenly launched via absolute path from the meta-repo root and produced a
misleading `check-sync.sh` FAIL against the wrong tree — re-run correctly
below; the stray run is not evidence of anything in this branch):

- `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` (default
  `RALPH_VERIFY_SCOPE=changed`, fell back to full Go scope because
  `.claude/hooks/PostToolUse.d/10-post-edit-verify.sh` is unclassifiable by
  the changed-language detector — the documented conservative-fallback
  behavior). Exit 0. Evidence: `docs/evidence/verify-2026-08-17-102004.log`.
  Covers: shellcheck across every hook + verify script (root and
  `templates/base/` mirrors, plus the new `ralph-dispatch.sh` and all 8
  `<event>.d/` shims), `sh -n` syntax checks on the same set, `jq -e .` on
  both `settings.json` files, Codex hook single-source/inline-detector/PR-provenance
  guards, `check-sync.sh` (PASS, DRIFTED=0), `check-pipeline-sync.sh` (PASS,
  `CLAUDE.md` correctly absent from `REFS`), `check-skill-sync.sh` (PASS, 13
  skills in lock-step), and the Go verifier (`gofmt: ok`, `go vet` silent,
  golangci-lint `0 issues.`).
- `./scripts/run-verify.sh` (full mode = static + test, `RALPH_VERIFY_SCOPE=full`)
  as broader evidence given the PR's meta-repo-wide reach (AC-8/AC-9
  explicitly ask for this). Exit 0, "All verifiers passed." Evidence:
  `docs/evidence/verify-2026-08-17-101740.log`. This includes the full shell
  test suite (`tests/*.sh`, all `OK`) and `go test ./...` (all packages
  `ok`/cached) — full behavioral confirmation belongs to `/test`, but this
  run is offered as supporting evidence since it was already required by
  AC-8's own wording.

No FAIL lines in either evidence log.

## Documentation drift

- `AGENTS.md` (root): repo map bullets for `.claude/rules/ralph/`, `.ralph/core/`,
  `.ralph/local/`, and the dispatcher's 3-layer fan-out are present and match
  the shipped layout (spot-checked against `.claude/hooks/ralph-dispatch.sh`'s
  actual `dirs` list). The self-review's H3-vs-H2 heading finding and the
  stale "Steps 4–7" cross-reference are both fixed in `c2d501e`.
- Rule-path references: `.claude/rules/ralph/*.md` cross-references
  (`agent-messaging.md`, `subagent-policy.md`, `post-implementation-pipeline.md`,
  `model-routing.md`, etc.) are consistent across `AGENTS.md`, `CLAUDE.md`,
  and the rule files themselves; `check-pipeline-sync.sh` mechanically proves
  this for the pipeline-order references specifically.
- `scripts/check-pipeline-sync.sh`'s `REFS` list and the canonical doc's own
  "Where this order is referenced" list are back in agreement (`CLAUDE.md`
  removed from both, with a comment explaining why).
- `docs/tech-debt/README.md`: the Phase-1 `AGENTS.md` KNOWN_DIFFS row is
  correctly marked `RESOLVED 2026-08-17`; the fix commit's own scope (signal
  trap, `lib_json.sh`, event-name guard, etc.) does not touch any tech-debt
  row that was already RESOLVED before this commit — no self-resolving-row
  drift.
- Plan progress checklist (`docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md`):
  "Verification artifact created" is still unchecked as of this report — expected,
  since this report is what creates it. Not a drift finding.

## Later-phase scope explicitly out of this verify

Per the plan's Non-goals, the following spec FRs are deliberately unimplemented
here and were not checked against AC-1..11: `ralph upgrade` wiring for the v2
layout / interactive-engine retirement / baseline removal (Phase 3), old-layout
downstream migration logic (Phase 4), `eject`/`adopt`/`doctor --strict`/mixed-layout
guard (Phase 5), and `.codex/` dispatcher parity (Phase 5, tracked as a new
tech-debt row above rather than silently deferred).

## What remains unverified

- Real Claude Code / Codex runtime execution of the dispatcher against a live
  hook invocation (as opposed to the shell-level fixture harness in
  `tests/test-ralph-dispatch.sh`) — inherently unverifiable statically.
- The third dispatcher layer's (`.claude/hooks/local/<event>.d/`) execution
  *order* relative to the other two — code confirms it's wired, but no test
  exercises its ordering (pre-existing, self-review-acknowledged gap, not
  introduced by this PR).
- Behavioral test *coverage percentage* and any edge cases beyond what static
  analysis surfaces — `/test`'s scope, not re-run here beyond the full
  `run-verify.sh` pass already reported above.

## Minimal next check for highest confidence gain

Add a Case F2 to `tests/test-ralph-dispatch.sh` exercising all three layers
(`core .d` → `.ralph/local/hooks/<event>.d/` → `.claude/hooks/local/<event>.d/`)
in one fixture, asserting the full three-part order string. This is the one
gap in the self-review's own acknowledged list that is cheap to close and
would remove the last "wired but unordered-tested" caveat in AC-4.

---

## Cycle 2

- Date: 2026-08-17
- Scope: delta since this cycle-1 verify (`4d4bdfc..bd2d867`): test report +
  dispatcher 3-layer order test + hermeticity fix (`0132d9f`), sync-docs
  (`5283ee0`), cross-review triage cycle 1 (`6e915f5`), cross-review
  ACTION_REQUIRED fixes (`f80e60f`), cycle-2 self-review (`98d184f`),
  cycle-2 self-review fixes (`bd2d867`, current HEAD)
- Pipeline cycle: 2/2 (cap `RALPH_STANDARD_MAX_PIPELINE_CYCLES`, default 2)

### Verdict: PASS

No new spec-compliance gaps and no new static-analysis failures. Both
cross-review ACTION_REQUIRED fixes match the triage contract; all cycle-2
self-review findings that were flagged "fix before merge" are fixed, and the
two flagged "cheap and worth doing" bookkeeping items are done. Deferred
items are correctly recorded, not silently dropped.

### Cross-review ACTION_REQUIRED fix crosscheck (`f80e60f`)

| # | Triage contract | Delivered | Evidence |
| --- | --- | --- | --- |
| AR#1 | `.ralph/local/**` must classify as owner=seed (L3 overlay, create-once/advisory, not core-replaceable) | Yes | `internal/cli/init.go`: `ownerForScaffoldPath` gained `if strings.HasPrefix(relPath, ".ralph/local/") { return scaffold.OwnerSeed }`, ordered before the catch-all `return scaffold.OwnerCore`; doc comment explains the Phase-3-replace-planner risk it closes. `init_v2_test.go`'s `TestExecuteInit_V2_FreshInit_LayoutAndOwners` gained a `.ralph/local/verify.d/.gitkeep` → `OwnerSeed` assertion (fixture embed FS updated to match) |
| AR#2 | `ralph pack add` on a v2 manifest must assign owner=core to every entry it writes (pack payload + `.claude/rules/ralph/<lang>.md`) so Phase 3's planner doesn't treat them as legacy-skipped/ownerless | Yes | `internal/cli/pack.go`: `addPack` now loops `pr.hashes` and calls `manifest.SetOwner(path, ...)` for every entry when `manifest.Meta.Layout == scaffold.LayoutV2`; legacy (no layout) manifests are left ownerless. Two new tests: `TestAddPack_V2Manifest_AssignsOwnerCore` (asserts owner=core on pack verify.sh/README.md/rule.md) and `TestAddPack_LegacyManifest_StaysOwnerless` (asserts no owner written and `Meta.Layout` isn't silently upgraded to v2) |

`f80e60f`'s pack.go fix originally hardcoded `scaffold.OwnerCore` (a correct
but duplicated classification); cycle-2 self-review (C2-5) flagged the
duplication as a latent-divergence risk, and `bd2d867` replaced it with a
direct call to `ownerForScaffoldPath(path)` — confirmed at
`internal/cli/pack.go:84-95`, same package, no import needed. The two entry
points (`ralph init`, `ralph pack add`) now share one classifier and cannot
diverge on a future pack path (e.g. under `docs/` or `.ralph/local/`).

### Cycle-2 self-review (C2-1..C2-5) fix crosscheck (`bd2d867`)

| # | Self-review finding | Fixed? | Evidence |
| --- | --- | --- | --- |
| C2-1 (MEDIUM, "fix before merge") | `trap cleanup INT TERM HUP` had no `exit`, so POSIX `sh` **resumes** after a fatal signal instead of terminating; `cleanup`'s unconditional `rm -f "$out_tmp.first"` could also expand to a bare, cwd-relative `.first` if the trap fired before `out_tmp` was assigned | Yes | `.claude/hooks/ralph-dispatch.sh` (byte-identical `templates/base/` mirror, `diff` confirms): `cleanup()` now guards every removal with `if [ -n "$var" ]; then rm -f "$var"; fi` instead of an unconditional `rm -f`; `trap cleanup EXIT` stays bare, but `trap 'cleanup; exit 130' INT`, `'cleanup; exit 143' TERM`, `'cleanup; exit 129' HUP` were added — matching the self-review's own recommended fix verbatim. `tests/test-ralph-dispatch.sh` gained Case I: starts a slow hook, sends `SIGTERM` mid-run, asserts exit 143 (not resumed) and no stray `.first` left in the fixture cwd |
| C2-2 (MEDIUM) | "Hooks dispatcher" claims in `AGENTS.md`, `docs/architecture/repo-map.md`, and `README.md` were unqualified — Codex still calls hooks directly, so `.ralph/local/hooks/<event>.d/` drop-ins silently don't run under Codex | Yes | All three sites now carry "(Claude Code today; Codex wiring is Phase 3)" or equivalent wording (verified via grep — `AGENTS.md:106`, `docs/architecture/repo-map.md:20`, `README.md:229`) |
| C2-3 (MEDIUM, "cheap, worth doing") | `.gitignore` tech-debt row asserted "no free area below the block", which `c2d501e` (two commits earlier, same PR) had already fixed by appending a signpost line | Yes | `docs/tech-debt/README.md` row struck through with `~~...~~ (RESOLVED 2026-08-17 in c2d501e)`, recommendation line struck through and replaced with a pointer to the fix commit. `tail -1` of both `.gitignore` and `templates/base/.gitignore` confirms the "Project-specific ignores go below this line." signpost is present and byte-identical between root and template |
| C2-4 (MEDIUM, "cheap, worth doing") | Four cycle-1 LOW items (dispatcher `awk` duplication, `check-sync.sh` `ROOT_ONLY` mislabel, `identical` double-count, `packRuleRelPath` OS-separator manifest keys) were about to vanish untracked at the pipeline cap | Yes | One new batched tech-debt row covers the three still-open items (`awk` duplication, `ROOT_ONLY` label, `identical` counter) and explicitly notes the fourth (`packRuleRelPath`) was actually fixed in this same cleanup slice rather than deferred — confirmed: `internal/cli/language_pack.go:121` now reads `path.Join(".claude", "rules", "ralph", pack+".md")` (was `filepath.Join`), with a doc comment explaining the slash-safe-manifest-key rationale |
| C2-5 (MEDIUM) | `pack.go`'s `f80e60f` fix hardcoded `scaffold.OwnerCore` instead of calling the classifier it named in its own comment (`ownerForScaffoldPath`) — same-package, one-line fix | Yes | `internal/cli/pack.go:91`: `manifest.SetOwner(path, ownerForScaffoldPath(path))`, comment rewritten to say classification is "shared... rather than mirrored" (see above) |

Also delivered in the same commit and consistent with the report's "LOW
quick wins" / "Process gaps" sections: `templates/base/AGENTS.md` gained a
`<!-- Project-specific notes go below this line. -->` signpost after its
managed block (mirroring the `.gitignore` convention; AC-11 grep for
`internal/`, `cmd/ralph`, "Repo map" in `templates/base/AGENTS.md` still
returns nothing — no meta-repo leakage introduced); README scaffold-tree
comment column realignment (cosmetic, not spot-checked byte-for-byte); the
plan's "Plan reviewed" checkbox is now ticked; and a cycle-1 `phase:verify`
insight event was appended to
`docs/insights/events/2026-08-17-overlay-scaffold-v2-p2.jsonl` (was missing
— confirmed present, `ts: 2026-08-17T13:32:51Z`, `verdict: pass`).

Deferred-and-correctly-recorded (not "fix before merge" per the report's own
recommendation): C2-2's follow-up scope (only the wording was in scope for
this cycle, not the actual Codex dispatcher migration — that's Phase 3), and
the LOW list (`C2-6`/awk/`ROOT_ONLY`/`identical`, all captured in the C2-4
batched row).

### Static analysis (re-run, HEAD `bd2d867`)

`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` from inside the
worktree: exit 0. `sh -n` on both `ralph-dispatch.sh` mirrors and
`tests/test-ralph-dispatch.sh` — all OK. shellcheck across every hook +
verify script — OK. `check-sync.sh` — `DRIFTED: 0`, `KNOWN_DIFF: 3`
(`.claude/rules/ralph/model-routing.md`, `.github/workflows/verify.yml`,
`CLAUDE.md`; `AGENTS.md` still correctly absent), `PASS: all files in
sync.`. `check-pipeline-sync.sh` — all 6 references OK. `check-skill-sync.sh`
— 13 skills in lock-step. Go verifier — `gofmt: ok`, `go vet` silent,
`golangci-lint` `0 issues.`, `staticcheck` present on `PATH` (silent on
success, consistent with a clean pass). No FAIL lines. Evidence:
`docs/evidence/verify-2026-08-17-133840.log`.

### AC re-check (delta-relevant only)

AC-2/AC-6 (owner attribution) and AC-7 (`check-sync.sh` block-aware,
`AGENTS.md` out of `KNOWN_DIFFS`, `DRIFTED=0`) were re-confirmed above and
still hold after the cycle-2 fixes; AC-1/AC-3/AC-4/AC-5/AC-8/AC-9/AC-10/AC-11
are untouched by the cycle-2 delta (no files in their evidence paths changed
between `4d4bdfc` and `HEAD` except the dispatcher script covered under
AC-4/AC-8, whose behavior for the already-tested cases is unchanged — only
the signal-handling path, newly covered by Case I, changed). Plan AC
checkboxes (`docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md`) remain
unticked — pre-existing doc-drift noted in the cycle-1 section above, not
introduced or worsened by cycle 2.

### What remains unverified (cycle 2 delta)

- Real Claude Code / Codex runtime execution of the reworked signal traps —
  `tests/test-ralph-dispatch.sh` Case I exercises the POSIX `sh` semantics
  directly (background process + `kill -TERM` + `wait`), which is the
  correct level for this claim, but a live agent-triggered hook timeout was
  not separately reproduced.
- The third dispatcher layer's execution *order* (`.claude/hooks/local/<event>.d/`)
  is still untested for ordering, per the cycle-1 section — unchanged by this
  delta and not part of what cycle 2 targeted.
- Behavioral coverage percentage and the actual `go test ./...` / shell-suite
  pass/fail outcome for this exact HEAD — that is `/test`'s scope for cycle
  2; this report only re-ran static analysis.

### Minimal next check for highest confidence gain (cycle 2)

Run `/test` (behavioral) against HEAD `bd2d867` to confirm
`tests/test-ralph-dispatch.sh` Case I and the two new Go ownership tests
(`TestAddPack_V2Manifest_AssignsOwnerCore`,
`TestAddPack_LegacyManifest_StaysOwnerless`) actually pass, not just parse —
static analysis (`sh -n`, shellcheck, `go vet`/`gofmt`/`golangci-lint`)
cannot execute a signal-handling probe or a Go test body.

---

## Cycle 3

- Date: 2026-08-17
- Scope: delta since this cycle-2 verify (`54f97ac..HEAD`): cycle-2 test +
  sync-docs artifacts (`d21273b`, `0be339a`), cycle-2 cross-review triage
  (`d5b478e`), cycle-2 cross-review ACTION_REQUIRED fixes (`c289875` —
  `doctor` `checkHooks` command-token split, `init` symlinked-block-surface
  refusal), cycle-3 self-review (`3580c23`), cycle-3 self-review fixes
  (`2b1a855`, current HEAD)
- Pipeline cycle: 3/3 (cap raised from the default 2 to 3 for this PR)

### Verdict: PASS

No new spec-compliance gaps and no new static-analysis failures. Both
cycle-2 cross-review ACTION_REQUIRED fixes match the triage contract, and
the cycle-3 self-review's two "strongly recommended before /pr" findings
(C3-1, C3-2) are genuinely fixed with regression tests, not just patched
around. The remaining cycle-3 LOW findings are either fixed outright or
correctly batched into one new tech-debt row — this is the last pipeline
cycle, so nothing here was left silently unrecorded.

### Cross-review ACTION_REQUIRED fix crosscheck (`c289875`)

| # | Triage contract | Delivered | Evidence |
| --- | --- | --- | --- |
| AR#1 | `ralph doctor`'s hooks-integrity check must stat only the executable token of a `command = "./path/to/script.sh EventName"` string, not the whole string, so a v2 scaffold's argument-carrying dispatcher command does not false-fail | Yes | `internal/cli/doctor.go:341-352` (`checkHooks`): `strings.Fields(cmd)` splits on whitespace, `exe := fields[0]` is stat'd instead of `cmd`; empty-fields guarded with `continue`. `internal/cli/doctor_hooks_test.go` adds `TestCheckHooks_DispatcherCommandWithArgs_ScriptPresentReportsPass` (present script → pass) plus a missing-script → fail case and an arg-less command → pass case (per the self-review's cycle-3 positive notes), so both directions of the regression are pinned, not just the reviewer's original repro |
| AR#2 | `init`'s block-append path (non-`--force`) must refuse to write through a symlinked pre-existing `AGENTS.md`/`.gitignore`, leaving it untouched with a warning (same non-destructive posture as the malformed-block case) | Yes | `internal/cli/init.go:365-380` (`reconcileBlockSurfaces`): `os.Lstat(diskPath)` before the existing `os.ReadFile`; a symlink mode bit warns `"is a symlink; left untouched"` and `continue`s, a non-regular mode warns `"is not a regular file; left untouched"` and `continue`s. `internal/cli/init_v2_test.go`'s `TestExecuteInit_V2_SymlinkedBlockSurface_LeftUntouchedWithWarning` (pre-existing in this commit's parent per the self-review's C3-1 evidence trail) asserts the link survives untouched and the external target's bytes are unchanged |

Both fixes are exactly the "small fix + test" shape the triage report asked
for. Neither widened scope beyond the two named files (`doctor.go`,
`init.go`) plus their test files.

### Cycle-3 self-review finding crosscheck (`2b1a855`)

| # | Self-review finding | Fixed? | Evidence |
| --- | --- | --- | --- |
| C3-1 (MEDIUM, "strongly recommended before /pr") | The AR#2 symlink guard only covers `reconcileBlockSurfaces` (paths `RenderFS` already classified as `Skipped`); `RenderFS`'s own existence check uses `os.Stat`, which follows symlinks, so a **dangling** symlink at a scaffold path stats as absent, gets classified `Created`, and `os.WriteFile` writes scaffold content through the link to wherever it resolves — outside `TargetDir`. Applies to every rendered path, not just block surfaces. | Yes | `internal/scaffold/render.go:74-85`: `os.Stat(target)` replaced with `os.Lstat(target)`, with a doc comment explaining the Stat-follows-symlinks containment gap and why `--force` is treated as explicit user consent to follow a link. Two new regression tests: `internal/scaffold/render_test.go:TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed` (asserts the dangling link is `Skipped` not `Created`, the link itself is untouched, and nothing is written at the external target) and `internal/cli/init_v2_test.go:TestExecuteInit_V2_DanglingSymlinkBlockSurface_LeftUntouchedWithWarning` (same assertion at the `executeInit` integration level, confirming the AR#2 warning path and the render-layer fix compose correctly for a block surface specifically). Both `Lstat` guards (render-layer for dangling links, `reconcileBlockSurfaces`-layer for resolvable links reaching the skipped/append path) are now in place, closing the gap the self-review identified as "the fix commit adopted the threat model and left the sibling gate open" |
| C3-2 (MEDIUM, "strongly recommended before /pr") | Case I's SIGTERM regression test (added for cycle-2's C2-1 trap fix) backgrounds a wrapping subshell (`( cd ... \| dispatcher ) &`) and signals `$!` — the subshell's PID, not the dispatcher's. The subshell dies from the default TERM action, the dispatcher is never signaled and keeps running to completion reparented to PID 1, so the test measures POSIX subshell-kill semantics, not the trap the fix added; a trap-free dispatcher passes unchanged. | Yes | `tests/test-ralph-dispatch.sh:333-364`: the wrapping subshell is replaced with a backgrounded `( cd "$fixture" \|\| exit 1; exec ./.claude/hooks/ralph-dispatch.sh PreCompact < "$i_stdin" > "$i_out" 2>&1 ) &`, with a comment explaining that `exec` after `cd` replaces the backgrounded subshell's own process image with the dispatcher, so `$!` is the dispatcher's real PID — matching the self-review's own recommended fix shape exactly (`sh -c` whose `exec` replaces the shell). The test also now snapshots `$TMPDIR`'s `ralph-dispatch-*` entries before and after via `find`+`comm -13`, asserting no leaked temp file rather than only the `.first` check, and reaps the orphaned `10-slow.sh` sleep with `pkill` so it does not outlive the test run |

Both fixes match their finding's recommended shape verbatim, not just the
letter of "make the assertion pass" — C3-1's fix is layered (render.go
*and* the existing `reconcileBlockSurfaces` guard both hold), and C3-2's
fix targets the actual process the trap protects, with an independent
temp-file-leak check added on top of what the finding asked for.

### Cycle-3 LOW findings disposition

| # | Finding | Status | Evidence |
| --- | --- | --- | --- |
| C3-3 | `tests/test-ralph-dispatch.sh`'s header case list (`a.`–`h.`) is one entry short of the body (Case I exists, not listed) | Fixed | `tests/test-ralph-dispatch.sh:19-25`: an `i.` entry describing the reworked Case I was added to the header enumeration in the same commit |
| C3-4 | Two insight events in `docs/insights/events/2026-08-17-overlay-scaffold-v2-p2.jsonl` are mis-stamped `cycle:1` although appended by cycle-2 fix commits; cycle 2 has no `self_review` event | Registered as tech debt, not fixed | `docs/tech-debt/README.md`'s new batched row (below) names both mis-stamped events and the missing `self_review` event, with the correction path spelled out (`scripts/insights-append.sh`) |
| C3-5 | `cli_test.go:2416`'s `t.Fatalf("precondition failed: ...")` misattributes the `addPack`-must-not-upgrade-`Meta.Layout` regression as a broken fixture; the triage report's re-resolved `init.go:373` pointer lands on a closing brace, not `reconcileBlockSurfaces` itself | Registered as tech debt, not fixed | Same batched row names both the `Fatalf` wording and the stale triage pointer/header |

This matches the self-review's own contingency exactly: it named C3-1 and
C3-2 as the two items worth a fix in this cycle, and said C3-3 through C3-5
"need one batched `docs/tech-debt/README.md` row" if deferred. C3-3 was
fixed instead of deferred (cheaper than expected); C3-4 and C3-5 landed in
one new row:

`docs/tech-debt/README.md` (new row, appended in `2b1a855`): "Two cycle-3
self-review findings for overlay-scaffold-v2 Phase 2 were left unfixed at
the pipeline cap (3/3, cap raised from 2)" — names both mis-stamped insight
events plus the missing `self_review` event (C3-4) and both the `Fatalf`
wording and the stale triage-report pointer/header (C3-5), with trigger and
evidence-path columns filled in per the existing register format. No LOW
finding from this cycle vanished untracked at the pipeline's last cycle.

### Static analysis (re-run, HEAD `2b1a855`)

`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` from inside the
worktree: exit 0. Covers shellcheck + `sh -n` across every hook and verify
script (including the reworked `tests/test-ralph-dispatch.sh` Case I),
`jq -e .` on both `settings.json` files, the three Codex hook guards,
`check-sync.sh` (`DRIFTED: 0`, `KNOWN_DIFF: 3` —
`.claude/rules/ralph/model-routing.md`, `.github/workflows/verify.yml`,
`CLAUDE.md`; `AGENTS.md` still correctly absent, `PASS: all files in
sync.`), `check-pipeline-sync.sh` (all 6 references OK),
`check-skill-sync.sh` (13 skills in lock-step), and the Go verifier
(`gofmt: ok`, `go vet` silent, `golangci-lint` `0 issues.`). No FAIL lines.
Evidence: `docs/evidence/verify-2026-08-17-143744.log`.

### AC re-check (delta-relevant only)

- **AC-1 / AC-3** (fresh-init v2 layout generation; existing-file init
  behavior for regular files) still hold: both `c289875`'s and `2b1a855`'s
  `Lstat`/mode-bit guards only add new `continue` branches for symlink /
  non-regular targets — for a regular file, `info.Mode().IsRegular()` is
  true and both functions fall through to the pre-existing code path
  unchanged (`reconcileBlockSurfaces`'s `os.ReadFile` call, `RenderFS`'s
  existing `Skipped`/`Created` branching). No behavior change for the
  regular-file case AC-1/AC-3 actually test.
- **AC-8/AC-9** (hook-command executability smoke test; existing suite
  green): `checkHooks`'s fix directly targets the AC-8 smoke-test claim —
  before this cycle, `ralph doctor` on any fresh v2-layout project would
  have falsely reported the dispatcher's hook scripts as missing, which is
  the exact failure mode AC-8's "移設後のパス欠損検出" (post-move path-loss
  detection) clause exists to catch, now inverted into a false positive it
  did not anticipate. The fix restores the intended semantics: only the
  executable token is checked, so a present dispatcher script reports
  green and a genuinely absent one still reports missing (both directions
  pinned by `doctor_hooks_test.go`). AC-9's "existing suite passes" and the
  new tests' actual pass/fail outcome for this exact HEAD is `/test`'s
  scope, not re-run here beyond the static analysis above.
- **AC-2/AC-6/AC-7/AC-10/AC-11** are untouched by the cycle-3 delta (no
  files in their evidence paths changed between `bd2d867` and `HEAD` except
  where already covered above).
- Plan AC checkboxes (`docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md:86-96`)
  remain unticked even though `- [x] Plan reviewed` was ticked in cycle 2 —
  pre-existing doc-drift noted in the cycle-1 and cycle-2 sections above,
  unchanged by cycle 3, and separately named as a process gap in the
  cycle-2 self-review section (not re-litigated here).

### What remains unverified (cycle 3 delta)

- Real Claude Code / Codex runtime execution of `checkHooks`'s new token-split
  parsing against a live `ralph doctor` invocation — the regression tests
  exercise the function directly with a fixture `settings.json`, which is
  the correct unit-level check, but an end-to-end `ralph init --yes && ralph
  doctor` was not separately reproduced here (the self-review's own C3-crosscheck
  section above notes this was the reviewer's original live repro, not
  re-run by this verify pass).
- The two new Go tests (`TestCheckHooks_DispatcherCommandWithArgs_...`,
  `TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed`,
  `TestExecuteInit_V2_DanglingSymlinkBlockSurface_LeftUntouchedWithWarning`)
  and the reworked `tests/test-ralph-dispatch.sh` Case I were read and
  reasoned about, not executed — confirming they actually pass (not just
  compile/parse) is `/test`'s scope for this cycle.
- The third dispatcher layer's execution order
  (`.claude/hooks/local/<event>.d/`) remains untested for ordering,
  unchanged from cycles 1 and 2.

### Minimal next check for highest confidence gain (cycle 3)

Run `/test` (behavioral) against HEAD `2b1a855` to confirm the three new Go
tests and the reworked `tests/test-ralph-dispatch.sh` Case I actually pass —
in particular Case I's own correctness now depends on `exec` replacing the
backgrounded subshell's process image as documented; a shell that does not
support that construct as expected would silently fall back to signaling
the wrong process again, which only a real `sh` execution (not static
`sh -n`/shellcheck) can catch.

---

## Cycle 4

- Date: 2026-08-18
- Scope: delta since this cycle-3 verify (`a7c857b..HEAD`, HEAD `1e03cd5`):
  cycle-3 test + sync-docs artifacts (`bb22a24`, `c5771af`), cycle-3
  cross-review triage (`756aac8`), cycle-3 cross-review ACTION_REQUIRED
  fixes (`28842a0` — dispatcher child-kill on signal +
  `renderMappedFile` Lstat guard, plus an insight-event correction bundled
  in the same commit), a follow-up insight cycle-stamp fix (`88ba334`),
  cycle-4 self-review (`ef9e8e2`), and a tech-debt row refresh (`1e03cd5`)
- Pipeline cycle: 4/4 (final — this is the last cycle regardless of
  outcome per the standing pipeline-cap convention for this series)

### Verdict: PASS

No new spec-compliance gaps and no new static-analysis failures. Both
cycle-3 cross-review ACTION_REQUIRED fixes in `28842a0` match the triage
contract exactly, and the cycle-4 self-review's single MEDIUM finding
(C4-1, a tech-debt row asserting states this PR itself had already
superseded) is fixed as recommended, with no code change required.

### Cross-review ACTION_REQUIRED fix crosscheck (`28842a0`)

| # | Triage contract | Delivered | Evidence |
| --- | --- | --- | --- |
| AR#1 | Dispatcher signal traps (TERM/INT/HUP) must propagate to the currently-running hook child, not just kill the dispatcher itself, so a signalled dispatcher does not leave an in-flight hook script running detached and free to keep mutating the repo | Yes | `.claude/hooks/ralph-dispatch.sh:72-101` (and the byte-identical `templates/base/` mirror): `child=""` tracked alongside the existing cleanup vars; the hook invocation backgrounds itself (`"$script" < "$stdin_buf" > "$out_tmp" & child=$!; wait "$child"; rc=$?; ... child=""`) instead of running as a foreground command; a new `kill_child()` sends `TERM` to `$child` and `wait`s it, called first in each of the three signal traps (`trap 'kill_child; cleanup; exit N' INT/TERM/HUP`) before the pre-existing `cleanup`. `rc`/first-failure semantics are preserved — the backgrounding only wraps the same `"$script" ... ; rc=$?` sequence in `wait`, it does not change what `rc` captures. `tests/test-ralph-dispatch.sh` Case I is extended (not replaced) with a `finished_marker` that only appears if the 5s hook script ran to completion (the assertion that actually falsifies an unfixed dispatcher, per the comment at its declaration — POSIX `sh` defers a trapped signal until the current foreground command returns, so an unfixed dispatcher's `wait` doesn't unblock until the child exits on its own, at which point a bare `pgrep` check would find nothing to fail on either way) plus a `<=3s` elapsed-time bound and a `pgrep`-based defense-in-depth check |
| AR#2 | `renderMappedFile` (the pack-rule `.claude/rules/ralph/<lang>.md` write path) must apply the same `Lstat`-not-`Stat` guard C3-1 added to `scaffold.RenderFS`, closing the same dangling-symlink containment gap on the sibling write path the cycle-3 self-review named as still open | Yes | `internal/cli/language_pack.go:151` (`renderMappedFile`): `os.Stat(target)` replaced with `os.Lstat(target)`, with a doc comment explicitly citing `scaffold.RenderFS` as the pattern being mirrored and restating why `Stat` following a dangling symlink would defeat the `isInsideDir` boundary check above it. New regression test `internal/cli/cli_test.go` `TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed` (added in the same commit) asserts the rule path is `Skipped` not `Created`, the symlink itself survives untouched, and nothing is written at the external dangling target — the same three-part assertion shape as C3-1's own `RenderFS` regression test |

Both fixes land exactly where the triage named them (`ralph-dispatch.sh` +
its template mirror, `language_pack.go`), each with its own regression
test, and neither widens scope beyond the two named files plus their test
files. `diff .claude/hooks/ralph-dispatch.sh templates/base/.claude/hooks/ralph-dispatch.sh`
is clean — mirror discipline held.

`28842a0` also bundles the cycle-3 triage's WORTH_CONSIDERING #3 fix: the
two cycle-2 events mis-stamped `cycle:1` (`verify` at `13:42:10Z`,
`cross_review` at `14:02:20Z`) are corrected to `cycle:2`, and the missing
cycle-2 `self_review` event (`13:38:47Z`) is inserted in timestamp order —
confirmed by re-diffing `docs/insights/events/2026-08-17-overlay-scaffold-v2-p2.jsonl`
at `28842a0` directly, not just trusting the self-review's own account.
`88ba334` is a second, separate correction the first commit missed: the
cycle-3 `cross_review` event itself (`15:05:50Z`) was still stamped
`cycle:1`; `88ba334` corrects it to `cycle:3` (confirmed present at `:16`
in the current file). Both corrections are consistent with the same
underlying WORTH_CONSIDERING #3 intent (fix insight-event cadence data
this PR generated, inside the same PR) even though `88ba334` landed as a
follow-up rather than in the same commit.

### Cycle-4 self-review finding crosscheck (`1e03cd5`)

| # | Self-review finding | Fixed? | Evidence |
| --- | --- | --- | --- |
| C4-1 (MEDIUM) | The batched cycle-3 tech-debt row (written by `2b1a855`) asserted three things that were each true only at the moment it was written and false two commits later: (a) `docs/reports/cross-review-triage-overlay-scaffold-v2-p2.md`'s `internal/cli/init.go:373` pointer "lands on a closing brace" — that whole triage report was rewritten wholesale by `756aac8` and no longer references `init.go` at all; (b) the same report's header "still reads `Cycle: 2/2 (cap reached)`" — `756aac8`'s rewritten header reads `Cycle: 3/3`; (c) `internal/cli/cli_test.go:2416` names the misworded `Fatalf` — `28842a0` inserted 65 lines above that point, shifting the real `Fatalf` to `:2481` and leaving `:2416` pointing at an unrelated, correctly-worded `Fatalf` | Yes | `docs/tech-debt/README.md` (row diff in `1e03cd5`): clauses (a)/(b) replaced with a single strikethrough-annotated `(3) stale triage-report pointer/header~~ (RESOLVED 2026-08-18: 756aac8 rewrote the triage report; its header now reads Cycle: 3/3 and the decayed init.go:373 pointer no longer exists)`, consistent with how the same row already marks its other now-resolved clause (the mis-stamped insight events); the Impact cell's matching "stale triage pointer and header decay further" clause is removed; the Files cell drops `docs/reports/cross-review-triage-overlay-scaffold-v2-p2.md` and replaces the decaying `internal/cli/cli_test.go:2416` line-number pointer with the test name `internal/cli/cli_test.go` (`TestAddPack_LegacyManifest_StaysOwnerless`), which resolves to `:2456` today (confirmed via `grep -n`) and does not decay under future unrelated insertions to the same file the way a bare line number does |

The row edit matches C4-1's recommendation exactly ("one row edit, no code
change... replace `internal/cli/cli_test.go:2416` with the test name...
so the pointer stops decaying"), using the row's own established
strikethrough-plus-RESOLVED-annotation convention (already used for the
`.gitignore` clause and the insight-event clause in the same row) rather
than silently deleting the stale clauses — a reasonable, precedent-consistent
choice, not a deviation.

### Static analysis (re-run, HEAD `1e03cd5`)

`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` from inside the
worktree: exit 0. Covers shellcheck + `sh -n` across every hook and verify
script (including the reworked `ralph-dispatch.sh` signal-trap/child-kill
logic in both the root and `templates/base/` mirrors), `jq -e .` on both
`settings.json` files, the three Codex hook guards, `check-sync.sh`
(`DRIFTED: 0`, `KNOWN_DIFF: 3` — `.claude/rules/ralph/model-routing.md`,
`.github/workflows/verify.yml`, `CLAUDE.md`; `AGENTS.md` still correctly
absent, `PASS: all files in sync.`), `check-pipeline-sync.sh` (all 6
references OK), `check-skill-sync.sh` (13 skills in lock-step), and the Go
verifier (`gofmt: ok`, `go vet` silent, `golangci-lint` `0 issues.`;
`staticcheck` present on `PATH`, silent on success — consistent with a
clean pass, not a skip). No FAIL lines. Evidence:
`docs/evidence/verify-2026-08-17-160407.log`. `git status --porcelain` is
clean at `1e03cd5` (no uncommitted changes).

### AC re-check (delta-relevant only)

- **AC-4 / AC-8** (dispatcher event fan-out; hook-command executability):
  the child-kill change only adds signal-path behavior (backgrounding the
  hook script and killing it on TERM/INT/HUP) — the non-signalled
  execution path (`"$script" < "$stdin_buf" > "$out_tmp" & child=$!; wait
  "$child"; rc=$?`) still captures the same `rc` the prior foreground call
  captured, and the existing non-signal test cases (a.–h.) are unchanged
  by this delta, only Case I is extended. No regression to AC-4's
  documented JSON-merge-semantics cases or AC-8's smoke-test claim.
- **AC-6** (pack rule rendering to `.claude/rules/ralph/<lang>.md`,
  tracked with owner=core): `renderMappedFile`'s `Lstat` swap only adds a
  new `continue`-equivalent (`Skipped`) branch for symlink/dangling
  targets; for the regular-file case AC-6 actually exercises,
  `info.Mode().IsRegular()` (implicit in the existing `statErr == nil`
  branch) is unchanged, so the happy path AC-6 tests is unaffected.
- **AC-1/AC-2/AC-3/AC-5/AC-7/AC-9/AC-10/AC-11** are untouched by the
  cycle-4 delta — no files in their evidence paths changed between
  `2b1a855` and `HEAD` except where already covered above (dispatcher +
  `language_pack.go`) or the tech-debt/insight doc-only commits, which
  carry no code-behavior claim.
- Plan AC checkboxes (`docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md:86-96`)
  remain unticked — pre-existing doc-drift noted in every prior cycle
  section above, unchanged by cycle 4.

### What remains unverified (cycle 4 delta)

- The two new Go regression tests
  (`TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed`) and
  the extended `tests/test-ralph-dispatch.sh` Case I (`finished_marker`,
  the `<=3s` elapsed bound, and the `pgrep` defense-in-depth check) were
  read and reasoned about, not executed — confirming they actually pass
  (not just compile/parse), and confirming the child-kill fix actually
  prevents the hook child from outliving the dispatcher on a live signal,
  is `/test`'s scope for this cycle.
- The "Accepted residue" items the cycle-4 self-review recorded (partial
  process-group kill coverage — `kill_child` signals only the direct
  child, not grandchildren spawned by the hook script itself; a possible
  `set -e`/`child=""` assignment race on a signal landing mid-statement;
  non-fixture-scoped `pgrep`/`pkill` patterns in Case I) are deliberate,
  documented deferrals, not verified-safe by this pass — they were not
  independently re-derived here, only cross-checked against the
  self-review's own reasoning for internal consistency.
- The third dispatcher layer's execution order
  (`.claude/hooks/local/<event>.d/`) remains untested for ordering,
  unchanged from cycles 1–3.

### Minimal next check for highest confidence gain (cycle 4)

Run `/test` (behavioral) against HEAD `1e03cd5` to confirm
`TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed` and the
extended Case I actually pass — in particular that `finished_marker`
genuinely stays absent under a live `kill -TERM` against the real
dispatcher (not just that the test's own logic is sound on paper), since
that is the assertion this cycle's fix depends on to prove the child-kill
behavior works, not merely that the dispatcher itself terminates.
