# Self-review report: model-routing (upstream ralph port)

- Date: 2026-07-10
- Plan: `docs/plans/active/2026-07-10-model-routing.md`
- Branch: `chore/model-routing` (base `main`, merge-base `57757a0`)
- Diff scope: `git diff main...HEAD` — 1 commit, 22 files
- Reviewer scope: diff quality only (naming, readability, unnecessary changes,
  typos, mirror consistency, secrets, maintainability). No tests / static
  analysis were run.

## Verdict

**NO-MERGE (fix HIGH first).** The doc/shell/skill re-tiering is clean and the
root↔templates/base mirrors are byte-identical, but the Go scaffolding source of
truth (`internal/config/config.go` `Default()`) was left holding
`claude-opus-4-7` / `xhigh`. Because the `ralph run` Go path exports
`cfg.Pipeline.*` into `RALPH_MODEL` / `RALPH_EFFORT` unconditionally, the Go
defaults *override* the shell fallbacks this PR just changed — so the new
`opus` / `high` defaults are masked at runtime for any project without an
explicit `ralph.toml` pipeline override. This also violates the plan's own
acceptance criterion "`claude-opus-4-7`/`xhigh` が設定値として残存しない" (the
`internal/config/` path is not in the plan's exclusion set).

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

| Severity | Category | Finding | Evidence | Recommendation |
|----------|----------|---------|----------|----------------|
| HIGH | maintainability / correctness | **Go scaffold defaults left stale, and they override the shell defaults this PR changed.** `internal/config/config.go:60-61,74` still returns `Model: "claude-opus-4-7"`, `Effort: "xhigh"`, `ClaudeReviewerModel: "claude-opus-4-7"`. `internal/cli/run.go:58-60` exports `cfg.Pipeline.Model`/`Effort` into `RALPH_MODEL`/`RALPH_EFFORT` on every `ralph run`, and `run.go:81` exports `cfg.Loop.ClaudeReviewerModel`. Since `config.Load` falls back to `Default()` when `ralph.toml` is absent or the field is unset (`config.go:107,123,126,154`), a project with no `[pipeline] model` override runs on `claude-opus-4-7`/`xhigh` at runtime through the Go CLI — the shell fallback (`opus`/`high`) this PR just set is never reached. The intended default change is masked for the primary entry path. This is the same Go↔shell drift the sibling opus-4-7 PR flagged (self-review H-1, `docs/reports/self-review-2026-04-21-ralph-default-opus-4-7.md:32`), now re-opened in the opposite direction. | `internal/config/config.go:60-61,74`; `internal/cli/run.go:58-60,81`; `scripts/ralph-config.sh:18-19,38` | Update `Default()` to `opus` / `high` / `opus` (mirrors the shell + ralph.toml change) and update the pinned expectations in `internal/config/config_test.go:11,14,73-74,106` and `internal/cli/doctor_loop_test.go:42,101,130`. OR, if intentionally deferred, add a Non-goal line + tech-debt record explaining why the Go source of truth keeps the old defaults. Neither is currently done. |
| MEDIUM | acceptance-criterion drift | **Plan criterion partially unmet.** Plan line 62-63: "`claude-opus-4-7`/`xhigh` が設定値として残存しない（docs/specs/・docs/plans/・docs/reports/ を除く）". `internal/config/config.go` is a config value and is not in the excluded set, yet it still holds both literals. Either the code or the criterion/Non-goals must move. | `docs/plans/active/2026-07-10-model-routing.md:62-63` vs `internal/config/config.go:60-61,74` | Fold into the HIGH fix, or amend the plan's Non-goals + acceptance criteria to explicitly carve out the Go layer with a stated reason. |
| LOW | consistency | **`internal/scaffold/embed_test.go` was updated for `claude_reviewer_model` but the Go `config` defaults/tests were not.** The diff picks up the embedded-TOML pin (`embed_test.go:162` → `opus`) because it asserts against `templates/base/ralph.toml` content, but the sibling Go-native defaults in `internal/config/` that feed the same value were skipped. This asymmetry (one Go test updated, the Go source + its own tests untouched) is a readability/consistency wart that hints the Go layer was only partially considered. | `internal/scaffold/embed_test.go:162` (updated) vs `internal/config/config.go:74` + `config_test.go` (not updated) | Resolved automatically once the HIGH finding is addressed. |

## No-finding checks (passed)

- **Mirror consistency (root ↔ templates/base):** all mirrored files byte-identical (md5). No divergence introduced.
- **Stale literals in effective doc/shell/skill/toml paths:** none. All of `scripts/ralph-config.sh`, `templates/base/ralph.toml`, `docs/recipes/ralph-loop.md`, cross-review `SKILL.md` (×4), and `tests/test-ralph-config.sh` now read `opus`/`high`/`opus`. Remaining `claude-opus-4-7`/`xhigh` hits are in `docs/specs/` / `docs/plans/` / `docs/reports/` (intentionally excluded historical records) — except the Go paths flagged above.
- **reviewer seat:** correctly retained `model: opus` in both copies.
- **verifier/tester/doc-maintainer:** all three flipped `opus` → `sonnet` in both copies, matching the plan's judgment/procedural split.
- **Test expectation changes:** `tests/test-ralph-config.sh:54,57` and `internal/scaffold/embed_test.go:162` match the new shell/ralph.toml defaults exactly (`opus`, `high`, `opus`). (The remaining test drift is the Go-native `internal/config/*_test.go`, covered by the HIGH finding.)
- **New file `model-routing.md`:** no `paths:` frontmatter (correct for always-on rule), no typos found, no secrets, prose accurate. `subagent-policy.md` cross-reference sentence is grammatical and points at the correct sibling file.
- **Secrets / credentials:** none. All changes are model aliases and effort levels.
- **Debug code / TODO / commented-out code:** none.
- **Exception handling / null safety:** N/A — no executable logic changed (doc, frontmatter, shell default literals, test string pins only).

## Tech-debt

No new tech-debt entry written: the HIGH finding is a fixable in-scope gap, not
an accepted deferral. If the author consciously chooses to defer the Go-layer
update (option B under the HIGH finding), a `docs/tech-debt/README.md` entry
should be added at that time recording the Go↔shell default divergence and the
reason.
