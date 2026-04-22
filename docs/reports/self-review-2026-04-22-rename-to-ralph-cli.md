# Self-review report: rename-to-ralph-cli

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-rename-to-ralph-cli.md`
- Reviewer: reviewer subagent (Claude Opus 4.7)
- Scope: Diff quality only (naming, readability, unnecessary changes, typos, null safety, debug code, secrets, exception handling, security, maintainability). Spec compliance, test coverage, and doc drift are out of scope for this step.

## Evidence reviewed

- `git diff main...HEAD --stat` (34 files, +411 / -287)
- `git log main..HEAD --oneline` (6 commits)
- Full diff of `go.mod`, `cmd/**`, `internal/**`, `scripts/install.sh`, `.goreleaser.yml`, `AGENTS.md`, `CLAUDE.md`
- Full current content of `README.md`
- Scoped grep: `git grep -l "harness-engineering-scaffolding-template"` — only hits are in `docs/plans/archive/` and `docs/specs/` (explicitly non-goals per plan §Non-goals)
- Build/test sanity: `go build ./...` succeeds, `go test ./...` all packages pass
- `git remote -v` → `ssh://git@github.com/yoshpy-dev/ralph.git` (fetch + push)
- Cross-checked `.goreleaser.yml` tap section (`owner: yoshpy-dev`, `name: homebrew-tap`) against README's `brew install yoshpy-dev/tap/ralph`
- Cross-checked `scripts/install.sh` REPO URL against `api.github.com/repos/${REPO}/releases/latest` call
- Cross-checked testdata JSON `pr_url` values against `internal/state/reader_test.go` expectations
- Verified README-referenced paths exist: `scripts/new-feature-plan.sh`, `scripts/new-ralph-plan.sh`, `scripts/build-tui.sh`, `scripts/ralph`, `scripts/run-verify.sh`, `docs/recipes/ralph-loop.md`, `docs/roadmap/harness-maturity-model.md`
- Enumerated embedded packs: `templates/packs/{_template,dart,golang,python,rust,typescript}` and `packs/languages/{_template,dart,golang,python,rust,typescript}`
- Inspected `internal/scaffold/embed.go` `PackFS("<lang>")` resolution (`templates/packs/<lang>`)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | typo | README "Language packs" section instructs `ralph pack add go`, but the embedded pack is named `golang`. `PackFS("go")` will fail with `language pack "go" not found` because `fs.Sub(EmbeddedFS, "templates/packs/go")` does not resolve. This is a net-new copy-paste bug introduced by the README rewrite — the previous README only showed `./scripts/new-language-pack.sh go` (which is a scaffolder for creating a *new* pack of any name, so that line was technically harmless; the new `ralph pack add go` is not). | `README.md:146-150`; `internal/scaffold/embed.go:23-31` (`return fs.Sub(EmbeddedFS, "templates/packs/"+lang)`); `ls templates/packs/` → `_template dart golang python rust typescript`; `internal/cli/pack.go:60-62` (`return fmt.Errorf("language pack %q not found", lang)`) | Change the example to `ralph pack add golang` (or add a `go → golang` alias in `PackFS` if a shorter name is desired). The `./scripts/new-language-pack.sh go` example on the next line is separately fine since it creates a brand-new pack directory by whatever name the user passes. |
| LOW | naming | `.goreleaser.yml` `description` was changed from "Harness engineering scaffold and autonomous pipeline CLI" to "ralph — a CLI for harness engineering with Claude Code". The em-dash and the leading-lowercase `ralph` work fine on the Homebrew formula page, but the description contains the project name as a prefix, which is slightly redundant with the formula name `ralph` displayed above it. Not a defect; noting for stylistic consideration. | `.goreleaser.yml:46-47` | Optional: drop the leading `ralph — ` so the description reads "A CLI for harness engineering with Claude Code." Keep if the current wording is intentional for SEO/discovery. |
| LOW | readability | `cmd/ralph/main.go` now has two imports that differ only by path depth: `ralph "github.com/yoshpy-dev/ralph"` (root package alias) and `"github.com/yoshpy-dev/ralph/internal/cli"`. The `ralph` alias on the root module is load-bearing (it exposes `ralph.Version`, `ralph.Commit`, etc. via ldflags) but a reader unfamiliar with the codebase may momentarily think it shadows the `ralph` CLI name. Pre-existing pattern (the alias existed on the old module path too), so not introduced by this diff — the rename just made the alias and the module leaf visually identical. | `cmd/ralph/main.go:7-9` | No action required for this PR. If it becomes confusing in review, a follow-up can rename the alias to `meta` or `ralphpkg`. Flagging only so it is not lost. |
| LOW | maintainability | Testdata JSONs (`internal/state/testdata/checkpoint-complete.json`, `orchestrator-complete.json`) and the test assertion in `internal/state/reader_test.go` both bake the full PR URL `https://github.com/yoshpy-dev/ralph/pull/{7,42}` as string literals. If the repo is ever renamed again, three files must be updated in lockstep or the test silently drifts. | `internal/state/reader_test.go:73-74`; `internal/state/testdata/checkpoint-complete.json:31`; `internal/state/testdata/orchestrator-complete.json:9` | Optional follow-up (not this PR): have the test assert on a structural predicate (`strings.HasSuffix(s.PRUrl, "/pull/42") && strings.Contains(s.PRUrl, "/yoshpy-dev/")`) or derive the expected URL from a constant, so the JSON fixture is the single source of truth. Not blocking — the current assertions are correct and localized. |

## Positive notes

- **Surgical diff.** All 25 Go-file edits are pure import-path swaps — no incidental reformatting, no drive-by refactors, no debug code, no stray TODOs. Exactly what a mechanical rename should look like.
- **No secrets or credentials** introduced. No `.env`, no tokens, no hardcoded API keys. The only URL-shaped strings are public GitHub URLs and the Homebrew tap path, which are already public.
- **No debug code.** No leftover `fmt.Println`, no `log.Printf("DEBUG ...")`, no commented-out blocks.
- **Null safety is unchanged.** No pointer dereference patterns were altered; `internal/state/reader.go` `checkpoint.PRUrl != nil` guard remains intact (`reader.go:218-219`).
- **Exception handling unchanged.** No `err` swallowing, no new generic `recover()`. Import-only changes do not touch control flow.
- **Security posture unchanged.** No new input-handling code paths, no new file I/O, no new exec/shell invocations. `scripts/install.sh` still uses `set -eu`, still verifies via GitHub Releases API, still only changes the `REPO` constant.
- **testdata + test assertion updated in lockstep.** `internal/state/testdata/checkpoint-complete.json`, `orchestrator-complete.json`, and `internal/state/reader_test.go` all moved to the new URL together. (See LOW finding for a structural improvement, but correctness is preserved.)
- **README rewrite is cohesive** and reads as a CLI README rather than a repo-template README. Structure (Install → Quick start → Commands → Scaffolded layout → Operating loop → Ralph Loop → Hooks → Language packs → Portability → Adoption → Defaults → This-repo layout → License) is consistent with comparable Go CLI projects.
- **AGENTS.md + CLAUDE.md edits are minimal** (1 line each) and do not expand the "map" beyond its mandated size.
- **Scope discipline.** Plan §Non-goals explicitly lists `docs/plans/archive/`, `docs/reports/`, `docs/specs/`, and the active plan file itself as untouched. The diff respects that: the only remaining `harness-engineering-scaffolding-template` hits outside the diff are in those excluded paths.
- **`.github/workflows/` was left alone**, which is correct — the plan predicted workflows use `${{ github.repository }}` and manual inspection confirms no repo-name literal exists there.
- **Commit decomposition is clean.** Six commits cleanly separate plan, Go module rename, distribution scripts, README rewrite, AGENTS/CLAUDE reframe, and progress update. Bisect-friendly.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Testdata PR URLs are string-literal-coupled across `reader_test.go` and two JSON fixtures | Future repo/owner renames require 3 synchronized edits; silent drift possible | Out of scope for this rename PR (cosmetic coupling, not a correctness issue) | Next time the repo URL changes, or when adding more PR-URL-bearing fixtures | This report; `internal/state/reader_test.go:73-74` |

_(This is the same item as the LOW "maintainability" finding above. It is deferred intentionally — the current diff does not amplify the coupling, it just demonstrates the cost in passing. No row needs to be appended to `docs/tech-debt/README.md` for this one PR; noting inline is sufficient. If the same pattern recurs in another PR, promote it to the register.)_

## Recommendation

- **Merge: yes, after fixing the MEDIUM finding.** The `ralph pack add go` example is a real user-facing breakage (users copying the README will see a "language pack not found" error on first try). Change `go` → `golang` in `README.md:147`. Once that one-character fix lands, the PR is clean.
- **No CRITICAL findings.** No security, secrets, null-safety, or exception-handling issues. The diff is a disciplined, mechanical rename plus a well-scoped README rewrite.
- **Follow-ups** (not blocking):
  - LOW: consider trimming `.goreleaser.yml` `description` to remove the redundant `ralph — ` prefix.
  - LOW: consider making `reader_test.go` PR-URL assertions structural rather than string-equality, if more PR-URL fixtures are added later.
  - LOW: the `ralph "github.com/yoshpy-dev/ralph"` root-package alias in `cmd/ralph/main.go` is visually ambiguous now that module leaf == alias; leave as-is unless it trips up a future reader.
- **Known gaps** (deferred to `/verify` and `/test`, per self-review scope rules):
  - Spec compliance against acceptance criteria → `/verify`
  - Whether `./scripts/run-verify.sh` passes end-to-end → `/verify`
  - Whether `ralph init`, `ralph doctor`, `ralph pack add golang` behaviorally work after the module rename → `/test`
  - Documentation drift across `.claude/skills/`, `docs/recipes/`, `docs/roadmap/` → `/sync-docs`
