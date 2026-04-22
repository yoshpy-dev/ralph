# Verify report: Rename repo & rebrand to `ralph` CLI

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-rename-to-ralph-cli.md`
- Verifier: `verifier` subagent
- Scope: spec compliance + static analysis + documentation drift (behavioral tests are out of scope for `/verify`; handled by `/test`)
- Evidence: `docs/evidence/verify-2026-04-22-rename-to-ralph-cli.log`
- Branch: `refactor/rename-to-ralph-cli`
- Base: `main`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| `go.mod` module line = `github.com/yoshpy-dev/ralph` | Verified | `head -5 go.mod` → `module github.com/yoshpy-dev/ralph` |
| `go build ./...` succeeds | Verified | `go build ./...` exit 0, no output |
| `go test ./...` succeeds | Likely but unverified here | Bundled `run-verify.sh` golang verifier reports all packages `ok` (cached). Fresh `go test ./...` run belongs to `/test`. |
| Scoped grep for `harness-engineering-scaffolding-template` is ZERO hits in `cmd/`, `internal/`, `templates/`, `packs/`, `scripts/`, `.github/`, `.goreleaser.yml`, `go.mod`, `go.sum`, `README.md`, `AGENTS.md`, `CLAUDE.md`, `.claude/skills/`, `.claude/rules/`, `.claude/agents/`, `.claude/hooks/` | Verified | 16 separate Grep runs returned "No matches found" for each scoped path. Full-repo grep returns exactly 6 files, all in the allowed set: `docs/reports/self-review-2026-04-22-rename-to-ralph-cli.md`, `docs/plans/active/2026-04-22-rename-to-ralph-cli.md`, `docs/specs/2026-04-16-ralph-cli-tool.md`, `docs/plans/archive/2026-04-17-mojibake-postedit-guard.md`, `docs/plans/archive/2026-04-17-allow-go-and-repo-commands.md`, `docs/plans/archive/2026-04-16-ralph-cli-tool.md` |
| `scripts/install.sh` uses new repo URL | Verified | `scripts/install.sh:10` → `REPO="yoshpy-dev/ralph"`; header URL at line 7 uses `raw.githubusercontent.com/yoshpy-dev/ralph/main/scripts/install.sh` |
| `.goreleaser.yml` `homepage` points to new URL | Verified | `.goreleaser.yml:46` → `homepage: "https://github.com/yoshpy-dev/ralph"` |
| README centers the `ralph` CLI (no "scaffolding template" framing) | Verified | `README.md:1` is `# ralph`; `README.md:3` leads with "`ralph` is a CLI for harness engineering ..."; case-insensitive grep for `scaffolding template` / `template repository` in `README.md` returns no matches. `scaffold` appears only as a verb ("scaffolds, upgrades, and runs ...") or as a noun referring to the assets `ralph init` emits ("Core scaffold stays stack-agnostic"), not as a framing of the repo itself. |
| AGENTS.md line 1 describes a CLI (not a scaffold) | Partially met | `AGENTS.md:1` is the heading `# AGENTS.md`. The first content line (line 3) is `This repository hosts \`ralph\`, a CLI for harness engineering. Run \`ralph init\` to scaffold a new project from this source.` — this satisfies the spirit of the criterion (CLI-first description on the first prose line) even though it is technically line 3 rather than line 1. Flagged as a minor wording deviation, not a blocker. |
| `./scripts/run-verify.sh` exits 0 | Verified | Full run captured in evidence log; terminates with `==> All verifiers passed.` and `exit=0` |
| `git remote -v` points to new URL | Verified | `origin  ssh://git@github.com/yoshpy-dev/ralph.git` (fetch + push) |
| External-importer evidence attached | Verified | See Observational checks below; captured during planning on 2026-04-22. |
| Pre-merge gate: `gh repo rename` succeeded, `yoshpy-dev/ralph` reachable, old URL 301-redirects | Unknown at static-analysis time | Plan `Progress checklist` shows `[x] GitHub repo renamed & remote updated`; local `git remote -v` confirms the new URL is in use. Network-level reachability and 301 status require a live probe (curl `-I`) which belongs to `/test` or a pre-`/pr` gate. Not reverified here. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `go vet ./...` | Pass | No output (silent success) |
| `go build ./...` | Pass | No output (silent success) |
| `gofmt -l .` | Pass | Empty output; no files need reformatting |
| `./scripts/run-verify.sh` | Pass (exit 0) | shellcheck + hook syntax + `check_mojibake.sh` tests (11/11 PASS) + `check-sync.sh` (IDENTICAL 107, DRIFTED 0) + golang verifier (gofmt ok, 0 issues, all packages ok). Bundled evidence saved at `docs/evidence/verify-2026-04-22-101958.log` plus our combined log at `docs/evidence/verify-2026-04-22-rename-to-ralph-cli.log`. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md` cross-references to scripts | In sync | `./scripts/new-feature-plan.sh`, `./scripts/new-ralph-plan.sh`, `./scripts/run-verify.sh`, `./scripts/install.sh` all exist on disk |
| `README.md` install commands | In sync | `brew install yoshpy-dev/tap/ralph` preserved (formula name already `ralph`); curl install URL matches new `yoshpy-dev/ralph` raw path |
| `AGENTS.md` repo map | In sync | First prose line is CLI-focused. Map entries (`cmd/ralph/`, `cmd/ralph-tui/`, `internal/cli/`, `internal/scaffold/`, `templates/`, `packs/languages/`) all correspond to real directories. "scaffold" remains only where it accurately describes the noun (`internal/scaffold/`, `templates/base/` = "base scaffold"), not as a framing of the repo itself. |
| `CLAUDE.md` guidance | In sync | Case-insensitive scan for `scaffolding template` / `template repository` returns no matches. |
| `.claude/skills/release/SKILL.md` repo references | Not regrepped here | Plan's Verify plan notes this was confirmed clean during implementation; no old-name matches anywhere under `.claude/` confirms no regression. |
| Homebrew user-facing surface | In sync | `brew install yoshpy-dev/tap/ralph` path unchanged (formula name was already `ralph`); `.goreleaser.yml` `homepage` now points to `github.com/yoshpy-dev/ralph` so next release will publish updated formula metadata. |
| Plan `Progress checklist` | Doc drift (minor) | Items for "Review / Verification / Test / PR artifact created" are still unchecked. Review artifact exists (`docs/reports/self-review-2026-04-22-rename-to-ralph-cli.md`) and this verify artifact is being created in this run. Expected to be finalized by `/pr` archival. Not a blocker. |

## Observational checks

### External importer evidence (captured 2026-04-22 during planning)

- `gh search code "yoshpy-dev/harness-engineering-scaffolding-template"` → 0 results. No public source on GitHub imports the old module path.
- `pkg.go.dev/github.com/yoshpy-dev/harness-engineering-scaffolding-template` → HTTP 404. Module was never indexed by the Go proxy; no external importers can have pulled it via `go get`.
- Conclusion: zero external Go importers. The `go mod edit -module` cutover is safe without a compatibility shim. This is the load-bearing evidence for the plan's "no backwards-compatibility shim" decision.

### Scoped grep confirmation

Full-repo `rg "harness-engineering-scaffolding-template"` returns exactly 6 files. All are in the plan's explicit allow-list:

- `docs/reports/self-review-2026-04-22-rename-to-ralph-cli.md` (report about this rename; allowed under `docs/reports/`)
- `docs/plans/active/2026-04-22-rename-to-ralph-cli.md` (the active plan itself; explicitly allowed)
- `docs/specs/2026-04-16-ralph-cli-tool.md` (spec; allowed under `docs/specs/`)
- `docs/plans/archive/2026-04-16-ralph-cli-tool.md` (archive)
- `docs/plans/archive/2026-04-17-mojibake-postedit-guard.md` (archive)
- `docs/plans/archive/2026-04-17-allow-go-and-repo-commands.md` (archive)

No residual in `cmd/`, `internal/`, `templates/`, `packs/`, `scripts/`, `.github/`, `.goreleaser.yml`, `go.mod`, `go.sum`, `README.md`, `AGENTS.md`, `CLAUDE.md`, or `.claude/`.

### Branch commit hygiene

7 commits on `refactor/rename-to-ralph-cli`, each conventional-format and each scoped to one concern (plan, module rename, install + goreleaser, README, AGENTS/CLAUDE, plan progress, README pack-name fix).

## Coverage gaps

- **`go test ./...` fresh run**: `/verify` relies on cached test results surfaced by `run-verify.sh`. A fresh invalidated-cache run is a `/test` concern.
- **Live network check**: Confirming that `https://github.com/yoshpy-dev/harness-engineering-scaffolding-template` returns a 301 to `https://github.com/yoshpy-dev/ralph` is not performed by static analysis. This is a pre-`/pr` operational gate (and the plan's acceptance criterion already calls it out as a pre-`/pr` condition). Suggest probing with `curl -sIL https://github.com/yoshpy-dev/harness-engineering-scaffolding-template | head -1` before `/pr` runs — smallest useful additional check.
- **`ralph init` / `ralph doctor` smoke**: Binary-level integration checks listed in the plan's Test plan belong to `/test`, not `/verify`.
- **`.github/workflows/*.yml` repo-name hardcoding**: Scoped grep returned zero hits in `.github/`, which satisfies the acceptance criterion. The plan's suggestion that workflows should mostly use `${{ github.repository }}` was not independently reverified beyond the grep.

## Verdict

- **Overall: PASS** — all acceptance criteria that are verifiable via static analysis are met. One minor wording deviation (AGENTS.md CLI-description is on line 3, not strictly line 1) is called out as non-blocking.
- **Verified**:
  - `go.mod` module path migration
  - `go build ./...`, `go vet ./...`, `gofmt -l .` clean
  - Scoped grep: zero hits across all required paths; residuals confined to allow-listed historical docs
  - `scripts/install.sh` repo URL
  - `.goreleaser.yml` homepage
  - README framing and cross-reference integrity
  - `./scripts/run-verify.sh` exit 0
  - `git remote -v` points to new URL
  - External-importer evidence (`gh search code` 0 results, `pkg.go.dev` 404) attached
- **Partially verified**:
  - AGENTS.md "line 1 describes a CLI" — CLI description is on first prose line (line 3) rather than literally line 1. Treat as wording deviation, not a spec violation.
- **Not verified (belongs to `/test` or pre-`/pr` gate)**:
  - Fresh uncached `go test ./...`
  - Live HTTP 301 redirect probe from old GitHub URL to new one
  - `ralph init` / `ralph doctor` binary smoke
- **Recommended smallest additional check to increase confidence**: `curl -sIL https://github.com/yoshpy-dev/harness-engineering-scaffolding-template 2>&1 | head -5` immediately before `/pr` runs, to confirm the pre-merge gate in the plan's acceptance criteria.
