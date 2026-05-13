# Cross-review triage report: add-terraform-language-pack

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md`
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes (cycle 1 + cycle 2)
- Cycle: 2/2 (cap reached — `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`)
- Total reviewer findings (cycle 2): 2 (P1, P2)
- After triage (cycle 2): ACTION_REQUIRED=0, WORTH_CONSIDERING=1, DISMISSED=1
- Cycle 1 findings (previous run, preserved for traceability): ACTION_REQUIRED=1 (RESOLVED in commit `f27e1a2`), DISMISSED=3

## Triage context

- Active plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md`
- Self-review reports: cycle 1 + cycle 2 in `docs/reports/`
- Verify reports: cycle 1 + cycle 2 in `docs/reports/`
- Test reports: cycle 1 + cycle 2 in `docs/reports/`
- Implementation context summary: Cycle 1 shipped the terraform pack body, rule, detection, and recipe doc. Cycle 1 cross-review flagged a test-hermeticity issue (P2) which was fixed in `f27e1a2`. Cycle 2 (this run) targets the post-fix diff. The plan explicitly defers `ralph pack add <lang>` pathing bug (Codex /plan HIGH#1) and tracks it in `docs/tech-debt/README.md`.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

(none)

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | **[P2] Add Terraform state files to scaffold gitignore.** `.claude/rules/terraform.md:11` warns "never commit `terraform.tfstate`, `*.tfstate.backup`, `.terraform/`" but neither root `.gitignore` nor `templates/base/.gitignore` was updated. A routine `git add .` in a scaffolded project that runs terraform locally can stage state files containing secrets. | Real issue (Axis 1 YES) — Terraform state files routinely contain provider credentials, resource ARNs, and other secrets; secret-in-git-history incidents from tfstate are well documented. Worth fixing (Axis 2 DEBATABLE) — the change is 3 lines and fits the existing pattern (templates/base/.gitignore already enumerates `node_modules/`, `target/`, `.venv/`, etc., one set per ecosystem). Scope is adjacent but not strictly inside "add a language pack". Cap-reached, so the user must decide: extend cycle, accept inline, or record in Known gaps. | `templates/base/.gitignore`, optionally root `.gitignore` |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
| 1 | [P1] `ralph pack add terraform` installs the pack at the project root instead of `packs/languages/terraform/` and can overwrite an existing `README.md`. Codex empirically reproduced this (built ralph, ran `pack add` in a tempdir, found `README.md`/`verify.sh` landed at the root). | Pre-existing bug in `internal/cli/pack.go:64-67`, identical to Codex /plan HIGH#1 which the user explicitly accepted as out-of-scope. Affects ALL existing packs (golang/python/rust/dart/typescript) equally, not specific to terraform. Already recorded in `docs/tech-debt/README.md` with a concrete fix recipe (use `filepath.Join(absDir, "packs", "languages", lang)` to match `init.go:158`). The plan's documented mitigation is to ship packs via `ralph init` with `packs:` array, which the recipe doc reflects. | already-addressed |

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle-1 findings (preserved for traceability)

| Cycle | Severity | Finding | Status |
|-------|----------|---------|--------|
| 1 | P2 | Non-hermetic test PATH (`clean_path="/usr/bin:/bin"`) | **RESOLVED** in commit `f27e1a2` (symlink-only coreutils dir + leak-guard) |
| 1 | LOW | Mode case validation order | DISMISSED (style-preference, peer-pack parity) |
| 1 | LOW | Duplicated `find` expression | DISMISSED (out-of-scope, three callers only) |
| 1 | LOW | Hidden-file glob in rule frontmatter | DISMISSED (already-addressed in cycle-1 verify/test triage) |
