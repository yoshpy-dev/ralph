# Self-review report: add-terraform-language-pack (cycle 3)

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md`
- Reviewer: reviewer subagent (Claude Code)
- Scope: Diff quality only for cycle-3 fix commit `03c5598` (raised pipeline cap 2→3 to address Codex cycle-2 WORTH_CONSIDERING P2: missing Terraform ignores in `.gitignore` / `templates/base/.gitignore`)

## Evidence reviewed

- `git show --stat 03c5598` — only two files modified, both `.gitignore` files
- `git show 03c5598 -- .gitignore` and `git show 03c5598 -- templates/base/.gitignore` — both diffs are byte-identical 15-line insertions (14 content lines + 1 trailing blank)
- `cmp /Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/.gitignore /Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/templates/base/.gitignore` — exit 0 (IDENTICAL), mirror discipline satisfied
- Full root `.gitignore` (53 lines) read to confirm the insertion is positioned cleanly between the language-cache block (lines 1–13) and the `.claude/settings.local.json` block (line 30→), with one blank line on each side; no existing pattern was overwritten or relocated
- `.claude/rules/terraform.md` line 11 ("Never commit `terraform.tfstate`, `*.tfstate.backup`, `.terraform/`, or `*.auto.tfvars`…") — the rule the fix is enforcing
- Canonical reference: `https://raw.githubusercontent.com/github/gitignore/main/Terraform.gitignore` fetched live for cross-check
- Cycle-1 self-review (`self-review-2026-05-13-add-terraform-language-pack.md`) and cycle-2 self-review (`*-cycle2.md`) findings re-read; no new diff overlaps with their files (cycle-3 touches `.gitignore` only; cycle-1/2 touched `packs/`, `templates/packs/`, `scripts/detect-languages.sh`, `internal/scaffold/embed.go`, verify-test hardening)

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `*.tfstate.*.backup` is narrower than the canonical `*.tfstate.*` glob and will miss adjacent transient state artifacts such as `terraform.tfstate.d/<workspace>/*.tfstate.backup` (workspace state) and `.terraform.tfstate.lock.info` (apply lock file). The current set covers the two literal forms the rules file calls out (`*.tfstate`, `*.tfstate.backup`) but not the broader family. | `.gitignore:17-19`; canonical Terraform.gitignore uses `*.tfstate.*` as a single line. Workspace state lives in `terraform.tfstate.d/` which is *also* not matched by `.terraform/` (different prefix). | Non-blocking. Either tighten the comment in `.gitignore` to acknowledge the scope, or in a follow-up replace `*.tfstate.backup` + `*.tfstate.*.backup` with `*.tfstate.*` and add `terraform.tfstate.d/` and `.terraform.tfstate.lock.info` lines. Defer to tech-debt; rules-file claim is still satisfied. |
| LOW | maintainability | Divergence from canonical on `.tfvars`: the canonical Terraform.gitignore ignores all `*.tfvars` / `*.tfvars.json` (broad-and-safe), whereas this fix only ignores `*.auto.tfvars*`. Explicit-variable files passed via `-var-file=` are not auto-loaded and so genuinely live in the repo as fixtures, so narrowing is *defensible*, but a contributor who copies a secret into `prod.tfvars` won't be protected. | `.gitignore:21-22`; canonical lines `*.tfvars` / `*.tfvars.json`; rules file `terraform.md:11` only names `*.auto.tfvars` so the fix is consistent with the rule, just not with upstream consensus. | Non-blocking. Mention in tech-debt: if we ever add a CI scanner for committed secrets, prefer broadening to `*.tfvars` and using `!example.tfvars` negation for intentional fixtures. |
| LOW | maintainability | `.terraformrc` and `terraform.rc` (CLI configuration files — may contain `credentials "app.terraform.io" { token = "..." }` blocks for Terraform Cloud) are not ignored. The canonical list includes them. These usually live in `$HOME` rather than the repo, so risk is low, but a contributor running `terraform login` with a project-local config file would commit the token. | `.gitignore:15-28` (no `.terraformrc` / `terraform.rc` lines); canonical lines 38-40. | Non-blocking. Add in a follow-up if/when we see this pattern in scaffolded projects. |
| LOW | naming | The block header comment is fine ("Terraform / OpenTofu state and cache (often contain provider secrets)") but slightly understates scope — the block also ignores plan output, override files, and crash logs, none of which are strictly "state and cache". | `.gitignore:15` | Optional: rename comment to "Terraform / OpenTofu (state, plans, overrides, crash logs)". Not worth its own commit. |

No CRITICAL, HIGH, or MEDIUM findings.

## Verifications performed

1. **Patch is purely additive** — insertion at line 14→ in both files; the pre-existing 13-line language-cache block (`.DS_Store` through `.ruff_cache/`) and the trailing blocks (`.claude/settings.local.json`, `.harness/`, `coverage.out`, `docs/evidence/`, `docs/reports/pipeline-execution-*.json`) are unchanged. Confirmed by inspecting the diff hunk header `@@ -12,6 +12,21 @@` and reading the full file. No regression of existing ignores.
2. **`.terraform/` is ignored as a directory** — trailing slash on `.gitignore:16` makes git treat it as a directory-only match, which is correct (the provider binaries cache is a directory). Files literally named `.terraform` would still be tracked, which is the desired behavior.
3. **No source-file masking** — confirmed none of the added globs match `.tf` source files. Patterns added: `*.tfstate`, `*.tfstate.backup`, `*.tfstate.*.backup`, `*.tfplan`, `*.auto.tfvars`, `*.auto.tfvars.json`, `override.tf`, `override.tf.json`, `*_override.tf`, `*_override.tf.json`, `crash.log`, `crash.*.log`. The only patterns that match `*.tf` files are the four override-file lines, which is the intended behavior (developer-local overrides are not source artifacts).
4. **Mirror discipline** — `cmp` exit 0 between root and `templates/base/.gitignore`. The cycle-3 fix correctly mirrors to both sides, consistent with the project's mirror-discipline pattern (root + templates/base byte-identical).
5. **Cycle-1 / cycle-2 findings remain resolved** — cycle-3 only touches `.gitignore` files. None of the cycle-1 (pack scaffolding, detect script, rules file) or cycle-2 (hermetic PATH in tests) files are re-edited, so prior fixes are not reverted. No overlap with the cycle-2 hermetic-PATH harness work (`scripts/check-skill-sync.sh` semantics, test file).
6. **No debug code, secrets, exception handling, or path-traversal concerns** — patch is data-only (gitignore patterns), no code paths added.
7. **Comment header readability** — single-line block header on `.gitignore:15` explains intent; readable in a `cat` of the file.

## Positive notes

- Both files are kept byte-identical with no manual drift; the commit follows the established mirror pattern correctly.
- Commit message clearly cites the Codex finding it addresses, lists each pattern with the reason, and ends with `Refs #52` — high-quality provenance.
- Inserting the new block *between* logical groups (language caches above, Claude/harness configs below) with blank-line separators preserves readability of the file.
- Patterns chosen are conservative-but-correct: no source-file masking, no over-broad globs that would hide intentional fixtures (`example.tf`, hand-written tests), and the trailing-slash convention on `.terraform/` is right.
- Header comment is honest about *why* (provider secrets), not just *what*.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Narrower-than-canonical `*.tfstate.*.backup` (misses `terraform.tfstate.d/<workspace>/...` and `.terraform.tfstate.lock.info`) | Low — workspace state files for multi-environment Terraform users could leak; uncommon in scaffold-time setups | Cycle-3 was a single-commit fix; broadening the glob set was out of scope for the pipeline-cap-raise scenario | Adoption of `terraform workspace` in any scaffolded project, or a security finding from a real user repo | This report; `.gitignore:17-19` |
| `*.tfvars` (broad) vs `*.auto.tfvars*` (narrow) divergence from canonical | Low — explicit `-var-file=prod.tfvars` files with credentials could be committed | Defensible narrow choice matches the rule file at `terraform.md:11`; broadening would require `!fixture.tfvars` negation patterns | If we add a secret-scanning CI hook, revisit and broaden with negations | This report; `.gitignore:21-22` |
| `.terraformrc` / `terraform.rc` not ignored (Terraform Cloud token risk) | Very low — usually `$HOME`-scoped | Almost never seen in repo root; out of scope for cycle-3 minimal fix | First report of a contributor accidentally committing a project-local `.terraformrc` | This report |

(All three rows are advisory only; none rise to MEDIUM. I recommend appending them as one consolidated entry to `docs/tech-debt/README.md` during `/sync-docs` rather than as three separate entries.)

## Recommendation

- **Merge: YES** — no CRITICAL or HIGH findings.
- The diff is a minimal, well-targeted fix for the Codex cycle-2 WORTH_CONSIDERING P2 finding. It enforces the policy the rules file already documents, mirrors correctly to both root and template, and does not regress existing ignores.
- The four LOW findings are upstream-divergence nits and a minor comment-naming suggestion; none of them justify another pipeline cycle.
- **Verdict**: MERGE. Cycle-3 fix is accepted as final; recommend proceeding to `/verify` → `/test` → `/sync-docs` (consolidate the three tech-debt rows into `docs/tech-debt/README.md`) → `/cross-review` (cycle 3, final allowed) → `/pr`.
- Cycle-2 ACTION_REQUIRED hermeticity fix and cycle-1 implementation findings remain resolved — cycle-3 touched none of those files.

## Follow-ups (non-blocking, defer to a separate issue if desired)

- Consider a future PR that aligns the Terraform ignore set more closely with GitHub's canonical Terraform.gitignore, specifically: broaden `*.tfstate.*.backup` → `*.tfstate.*`, add `terraform.tfstate.d/` and `.terraform.tfstate.lock.info`, add `.terraformrc` / `terraform.rc`, and discuss broadening to `*.tfvars` with explicit `!example.tfvars` negations. Bundle with any future Terraform pack hardening; not urgent.
