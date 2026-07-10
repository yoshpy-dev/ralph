# Self-review report: model-routing (upstream ralph port)

- Date: 2026-07-10
- Plan: `docs/plans/active/2026-07-10-model-routing.md`
- Branch: `chore/model-routing` (base `main`, merge-base `57757a0`)
- Diff scope: `git diff main...HEAD` — 2 commits, 23 files
- Reviewer scope: diff quality only (naming, readability, unnecessary changes,
  typos, mirror consistency, secrets, maintainability). No tests / static
  analysis were run.

## Verdict

**MERGE.** (Re-reviewed 2026-07-10 after fix commit `1e5f3d0`.) The original
HIGH — the stale Go scaffolding source of truth — is **resolved**. All
effective config paths now read `opus` / `high` / `opus`, the two follow-up
deltas (`check-sync.sh` KNOWN_DIFFS entry, root-only rule bullet) are clean and
correctly scoped, and the intentional root↔templates divergence is registered.
No remaining CRITICAL or HIGH findings.

### Re-verification (fix commit `1e5f3d0`)

- `internal/config/config.go` `Default()` now returns `Model: "opus"`,
  `Effort: "high"`, `ClaudeReviewerModel: "opus"` — the runtime override path
  (`run.go:58-60,81`) now propagates the intended defaults.
- Test fixtures updated in lock-step: `internal/config/config_test.go` (model
  `opus`, effort `high`, incl. the `want high` error-message string) and
  `internal/cli/doctor_loop_test.go` (`ClaudeReviewerModel: "opus"` ×3).
- `grep -e claude-opus-4-7 -e xhigh internal/ cmd/` → **NONE**. Whole-tree grep
  of `.go`/`.sh`/`.toml` outside `docs/` → **NONE** (only the intentional
  `xhigh`/`max` diminishing-returns prose in `model-routing.md` remains).
- New `check-sync.sh` KNOWN_DIFFS entry (`.claude/rules/model-routing.md`) has a
  clear rationale comment and follows the existing `CLAUDE.md`/`AGENTS.md`
  precedent. The registration is warranted: the root copy genuinely diverges
  now (extra Go-layer bullet), so this documents a real intentional diff rather
  than masking accidental drift.
- New root-only rule bullet is grammatical, accurate, and correctly scoped
  ("this repo only; not scaffolded downstream") — it closes the three-source
  lock-step gap the HIGH finding surfaced.

### Original verdict (superseded)

~~NO-MERGE (fix HIGH first).~~ The doc/shell/skill re-tiering was clean and the
root↔templates/base mirrors byte-identical, but the Go scaffolding source of
truth (`internal/config/config.go` `Default()`) was left holding
`claude-opus-4-7` / `xhigh`. Because the `ralph run` Go path exports
`cfg.Pipeline.*` into `RALPH_MODEL` / `RALPH_EFFORT` unconditionally, the Go
defaults overrode the shell fallbacks this PR changed — masking the new
`opus` / `high` defaults at runtime. **Resolved by `1e5f3d0`.**

## Evidence gathered

- md5 mirror check — all 6 mirrored pairs/quads byte-identical:
  - `model-routing.md` root == templates/base
  - `ralph-config.sh` root == templates/base
  - cross-review `SKILL.md` all 4 copies (`.claude`/`.agents` × root/templates) identical
  - `ralph-loop.md`, `subagent-policy.md`, `verifier.md`, `tester.md`, `doc-maintainer.md` root == templates/base
- reviewer agent kept `model: opus` in both copies (correct — judgment seat retained)
- `grep -e claude-opus-4-7 -e xhigh` outside docs/specs, docs/plans, docs/reports
  → the only *effective* (non-report, non-spec, non-plan) matches are in
  `internal/config/config.go` and its tests `internal/config/config_test.go`,
  `internal/cli/doctor_loop_test.go`. (The `.claude/rules/model-routing.md:42`
  `xhigh`/`max` match is intentional prose about diminishing returns — not a config value.)

## Findings

| Severity | Status | Category | Finding | Evidence | Recommendation |
|----------|--------|----------|---------|----------|----------------|
| HIGH | ✅ RESOLVED (`1e5f3d0`) | maintainability / correctness | **Go scaffold defaults left stale, and they override the shell defaults this PR changed.** `internal/config/config.go:60-61,74` returned `Model: "claude-opus-4-7"`, `Effort: "xhigh"`, `ClaudeReviewerModel: "claude-opus-4-7"`. `internal/cli/run.go:58-60,81` exports `cfg.Pipeline.Model`/`Effort` and `cfg.Loop.ClaudeReviewerModel` on every `ralph run`, and `config.Load` falls back to `Default()` when `ralph.toml` is absent/unset — so a project with no override ran on `claude-opus-4-7`/`xhigh` at runtime, masking the intended `opus`/`high`. **Fix `1e5f3d0`:** `Default()` now returns `opus`/`high`/`opus`; re-verified via grep (no `claude-opus-4-7`/`xhigh` in `internal/` or `cmd/`). | `internal/config/config.go:60-61,74`; `internal/cli/run.go:58-60,81` | Done in `1e5f3d0` (Go defaults + `config_test.go` + `doctor_loop_test.go` fixtures updated in lock-step). |
| MEDIUM | ✅ RESOLVED (`1e5f3d0`) | acceptance-criterion drift | **Plan criterion partially unmet.** Plan line 62-63 required no `claude-opus-4-7`/`xhigh` in config values (excluding docs/specs, docs/plans, docs/reports); `internal/config/config.go` was not excluded yet held both literals. **Fix `1e5f3d0`:** criterion now met — whole-tree grep of `.go`/`.sh`/`.toml` outside `docs/` returns none (bar the intentional `xhigh`/`max` prose in `model-routing.md`). | `docs/plans/active/2026-07-10-model-routing.md:62-63` vs `internal/config/config.go` | Done in `1e5f3d0`. |
| LOW | ✅ RESOLVED (`1e5f3d0`) | consistency | **`internal/scaffold/embed_test.go` was updated for `claude_reviewer_model` but the Go `config` defaults/tests were not.** The embedded-TOML pin (`embed_test.go:162`) moved to `opus` while the sibling Go-native defaults in `internal/config/` were skipped — a partial-consideration asymmetry. **Fix `1e5f3d0`:** the Go source + `config_test.go` are now updated, so the asymmetry is closed. | `internal/scaffold/embed_test.go:162` vs `internal/config/config.go` + `config_test.go` | Resolved as a byproduct of the HIGH fix. |

### New deltas in `1e5f3d0` (re-review, no findings)

| Severity | Category | Assessment | Evidence |
|----------|----------|------------|----------|
| — | maintainability | **`check-sync.sh` KNOWN_DIFFS entry for `.claude/rules/model-routing.md`.** Correct and warranted: the root copy now genuinely diverges (extra Go-layer bullet describing `internal/config/`, which does not exist in scaffolded projects), so the entry documents a real intentional diff rather than masking accidental drift. Comment is clear; follows the existing `CLAUDE.md`/`AGENTS.md` precedent. | `scripts/check-sync.sh:93-95`; `diff` of root vs templates `model-routing.md` confirms the 4-line divergence |
| — | readability | **New root-only rule bullet documenting the three lock-step sources (shell / toml / Go `Default()`).** Grammatical, accurate, correctly scoped ("this repo only; not scaffolded downstream"). Closes the exact lock-step gap the HIGH finding surfaced. No typos, no secrets. | `.claude/rules/model-routing.md:52-55` |

## No-finding checks (passed)

- **Mirror consistency (root ↔ templates/base):** after `1e5f3d0`, `model-routing.md` intentionally diverges (root carries the extra Go-layer bullet) and this divergence is registered in `check-sync.sh` KNOWN_DIFFS. All other mirrored files remain byte-identical.
- **Stale literals in effective doc/shell/skill/toml/Go paths:** none. All of `scripts/ralph-config.sh`, `templates/base/ralph.toml`, `docs/recipes/ralph-loop.md`, cross-review `SKILL.md` (×4), `tests/test-ralph-config.sh`, and (after `1e5f3d0`) `internal/config/config.go` now read `opus`/`high`/`opus`. Remaining `claude-opus-4-7`/`xhigh` hits are only in `docs/specs/` / `docs/plans/` / `docs/reports/` (intentionally excluded historical records) plus the intentional `xhigh`/`max` diminishing-returns prose.
- **reviewer seat:** correctly retained `model: opus` in both copies.
- **verifier/tester/doc-maintainer:** all three flipped `opus` → `sonnet` in both copies, matching the plan's judgment/procedural split.
- **Test expectation changes:** `tests/test-ralph-config.sh:54,57`, `internal/scaffold/embed_test.go:162`, and (after `1e5f3d0`) `internal/config/config_test.go` + `internal/cli/doctor_loop_test.go` all match the new defaults exactly (`opus`, `high`, `opus`), including the `want high` error-message string. No remaining Go-native test drift.
- **New file `model-routing.md`:** no `paths:` frontmatter (correct for always-on rule), no typos found, no secrets, prose accurate. `subagent-policy.md` cross-reference sentence is grammatical and points at the correct sibling file.
- **Secrets / credentials:** none. All changes are model aliases and effort levels.
- **Debug code / TODO / commented-out code:** none.
- **Exception handling / null safety:** N/A — no executable logic changed (doc, frontmatter, shell default literals, test string pins only).

## Tech-debt

No tech-debt entry needed: the HIGH finding was fixed in-branch (`1e5f3d0`)
rather than deferred. The Go↔shell↔toml lock-step requirement is now documented
as an always-on rule bullet in `.claude/rules/model-routing.md`, which prevents
recurrence without a separate tech-debt record.
