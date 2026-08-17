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
