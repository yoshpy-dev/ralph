# Self-review report: allow-go-and-repo-commands

- Date: 2026-04-17
- Plan: `docs/plans/active/2026-04-17-allow-go-and-repo-commands.md`
- Reviewer: reviewer subagent
- Scope: diff quality only for commit `7295c69` on branch `chore/allow-go-and-repo-commands`

## Evidence reviewed

- `git diff main...HEAD --stat` → 3 files, +237 / -2 lines
  - `.claude/settings.json` (+25 / -1)
  - `templates/base/.claude/settings.json` (+25 / -1)
  - `docs/plans/active/2026-04-17-allow-go-and-repo-commands.md` (+185, new)
- `git diff main...HEAD -- .claude/settings.json` and `git diff main...HEAD -- templates/base/.claude/settings.json` — byte-identical diffs (both add the same 25 entries in the same order).
- `diff .claude/settings.json templates/base/.claude/settings.json` → no output (files are byte-identical, as required by template sync).
- `jq -e . .claude/settings.json` and `jq -e . templates/base/.claude/settings.json` → both exit 0.
- `jq '.permissions.allow | length as $l | (unique | length)'` on project file → 60 total / 60 unique → no duplicate entries introduced.
- `jq '.permissions.allow | map(select(. == "Bash(sh:*)" or . == "Bash(bash:*)" or . == "Bash(xargs:*)"))'` → `[]` (plan AC5 honored; dangerous generic shell prefixes explicitly omitted).
- `.claude/hooks/pre_bash_guard.sh` reviewed to confirm the plan's security reasoning: deny list covers `sudo`, force push, hard reset, `rm -rf`, `.git/` redirects, `.env` redirects, and `git commit -m` command-substitution in double quotes. Adding `sh:*`/`bash:*`/`xargs:*` to `allow` would have been a trivial bypass of that deny list, so the plan's decision to exclude them is correct.
- `grep -E 'Bash\(go |Bash\(gofmt|settings\.json'` across `AGENTS.md`, `CLAUDE.md`, `.claude/rules/*.md` → no matches. No documentation lists specific allow entries, so no doc drift is introduced by this diff (the AC12 drift-check precondition is diff-quality-adjacent and I'm noting the observation here).
- `tests/` contents: three `.sh` files, matching the `./tests/*` prefix shape.

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | Three near-duplicate `ralph` entries (`./ralph:*`, `./bin/ralph:*`, `bin/ralph:*`) hard-code invocation shape. If `scripts/build-tui.sh` ever changes the output path or if a `bin/ralph-tui` legacy entry is reintroduced, this list will need to grow in lockstep. Grep-able but slightly noisy. | `.claude/settings.json:63-65`, `templates/base/.claude/settings.json:63-65` | Leave as-is for now — the three variants correspond to the three real invocation shapes users/scripts employ. Revisit only if a fourth shape appears. |
| LOW | maintainability | `Bash(go version)` is the sole exact entry among 15 Go entries; all its peers are `go <sub>:*`. Consistent with other exact entries already in the file (e.g., `Bash(sh -n:*)` uses the `:*` shape), but slightly asymmetric inside the Go block. | `.claude/settings.json:55` | Acceptable. `go version` has no meaningful arg surface (`go version [-m] [-v] [packages]`), so a `go version:*` prefix would also be fine and a touch more uniform. Not worth churn. |

## Positive notes

- Diff is a pure addition of 25 prefixes with one unchanged trailing-comma adjustment; zero mutations to existing entries, zero changes to `hooks`, zero changes to `env`. Reversible via `git revert` with no collateral.
- The two destinations (`.claude/settings.json` and `templates/base/.claude/settings.json`) are kept byte-identical, which is what `./scripts/check-template.sh` expects — no template drift is introduced by this diff.
- Codex [HIGH] advisory (exclude `sh:*`/`bash:*`/`xargs:*`) is visibly reflected both in the plan (Scope → Non-scope (X), AC5) and in the diff (absent from the added entries). Verified mechanically via jq.
- Ordering of added entries is logical and matches the plan's stated order (Syntax shell → Go toolchain → Go linters → shellcheck → ralph binaries → tests), making the diff easy to audit line by line.
- No secrets, tokens, absolute user paths, or debug markers leaked into the JSON. Added entries are all repo-local paths (`./ralph`, `./bin/ralph`, `bin/ralph`, `./tests/*`) or tool names.
- Commit message follows Conventional Commits (`chore:`) and honors the repo's safe-quoting rule (no backticks or `$(...)` in double-quoted `-m`).
- The 60-entry `allow` array is still a flat, grep-able list; no JSON schema change, no new keys, no comment hacks.

## Tech debt identified

None surfaced by this diff. The pre-existing redundancy between `.claude/settings.local.json` and the shared `.claude/settings.json` is explicitly held out of scope by the plan's Non-goals and is already tracked as follow-up in the plan's narrative — it is not introduced by this change and does not need a new tech-debt entry.

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(No rows added.)_

## Recommendation

- Merge: **YES** (no blocking findings; 0 CRITICAL, 0 HIGH, 0 MEDIUM, 2 LOW advisory notes only).
- Proceed to `/verify` to confirm AC1–AC12 spec compliance (out of scope here).
- Follow-ups (all optional, none blocking):
  - Consider normalizing `Bash(go version)` to `Bash(go version:*)` if a future diff already touches the Go block — not worth its own PR.
  - Revisit whether `./ralph:*` / `./bin/ralph:*` / `bin/ralph:*` can be collapsed if the repo standardizes on one invocation shape for the built binary.
