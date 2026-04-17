# Sync-docs report: allow-go-and-repo-commands

- Date: 2026-04-17
- Plan: `docs/plans/active/2026-04-17-allow-go-and-repo-commands.md`
- Branch: `chore/allow-go-and-repo-commands`
- Commit: `7295c69 chore: allow go toolchain and repo binaries in shared settings.json`
- Author: doc-maintainer subagent

## Scope of this diff

Pure additive change to `permissions.allow` in both `.claude/settings.json` and `templates/base/.claude/settings.json` (byte-identical). Entries added: Go toolchain (15 prefixes), linters (`gofmt:*`, `golangci-lint:*`, `staticcheck:*`, `goimports:*`, `shellcheck:*`), ralph binary (`./ralph:*`, `./bin/ralph:*`, `bin/ralph:*`), `./tests/*`, and `bash -n:*`. No hooks, scripts, skills, rules, language packs, workflows, or contracts changed.

## Drift check results

| Area checked | Result | Evidence |
| --- | --- | --- |
| `AGENTS.md` enumeration of `Bash(...)` entries | No drift | `grep -R "Bash(go \|gofmt\|golangci-lint\|staticcheck\|goimports\|shellcheck\|./ralph\|bin/ralph\|./tests\|bash -n"` returns no hits (already captured in verify report AC12). |
| `CLAUDE.md` (root + template) settings enumeration | No drift | No `Bash(...)` enumeration; defaults-behavior section is skill-oriented. |
| `.claude/rules/*.md` settings enumeration | No drift | Same grep is empty; rules are topic-/path-scoped, not permissions-scoped. |
| `README.md` — "Hook configuration" section | No drift | Section describes hooks shipped in `settings.json` by name (session start, prompt gate, bash guard, etc.), not permission entries. Adding a new "Allow-list" subsection would expand scope beyond the plan and create drift surface. Kept as-is. |
| `docs/architecture/repo-map.md` | No drift | Only references `.claude/settings.json` generically as "hook and permission configuration". Accurate at that granularity. |
| `docs/architecture/design-principles.md` | No drift | High-level principles, no permissions content. |
| `docs/quality/quality-gates.md` | No drift | File scopes verification gates (`run-verify.sh`, `check-*.sh`, pipeline-mode gates). Permission allow-listing is not a verification gate and does not belong here. |
| `docs/quality/definition-of-done.md` | No drift (not re-read — unchanged area). Allow-list baseline is not part of the DoD checklist. |
| `docs/tech-debt/README.md` | No entry added (see below) |
| `scripts/check-template.sh` | No drift | References `.claude/settings.json` only to require its existence and to validate referenced hook scripts exist. No permission enumeration. |
| `internal/cli/doctor.go` | No drift | Loads `settings.json` for JSON validity and hook presence, not permission entries. |
| `scripts/bootstrap.sh` | No drift | Only checks existence of `.claude/settings.json`. |
| `.claude/skills/sync-docs/SKILL.md` | No drift | Skill scope is unchanged. |
| `.claude/skills/loop/prompts/pipeline-outer.md` | No drift | References `settings.json` only as an example touch point for hook sync. |

## Tech-debt decision

The user asked whether `docs/tech-debt/README.md` should get an entry for the local-only commands in `.claude/settings.local.json` that are candidates for future promotion into the shared baseline (mentioned as Non-goals in the plan).

**Decision: not added.** Rationale:
- `.claude/settings.local.json` is per-developer and gitignored (no shared cost, no shared risk).
- The Non-goals section of the plan already records this deferral; the plan itself is archived at `/pr` time (`archive-plan.sh`) and remains grep-able from `docs/plans/archive/`.
- Existing `docs/tech-debt/README.md` entries are pipeline/behavior debt with concrete paydown triggers (e.g., "if false-negative CRITICAL findings slip through to merge"). A "some local entries could be promoted later" item has no concrete trigger and no shared impact — adding it would be noise, not a useful pointer.
- If future developers find themselves repeatedly needing a prefix that is already in many local files, that pressure naturally produces its own follow-up plan. Recording it now is premature.

## "Go toolchain is allow-listed" narrative — not added

The user asked whether any guidance doc should advertise the new baseline (e.g., "Go toolchain is allow-listed in shared settings" vs. "developer-local only"). No doc was updated:
- No existing doc distinguishes between shared and local settings at a narrative level — the only mentions of `settings.local.json` are in `README.md:190` ("Use for personal overrides") and the plan. Introducing a new narrative here would be net-new scope.
- The plan's Scope section (A–F) and the diff itself are the durable records. Anyone debugging why a `go ...` call no longer prompts can `git log -p .claude/settings.json` to reach the plan via the commit message.

## Files touched in this sync pass

| File | Change | Why |
| --- | --- | --- |
| `docs/plans/active/2026-04-17-allow-go-and-repo-commands.md` | Status: "Draft (revised after Codex advisory)" → "In review (post-implementation pipeline)"; Progress checklist: `Review artifact created`, `Verification artifact created`, `Test artifact created` marked `[x]` with report paths appended. | Progress checklist was stale. Review/verify/test artifacts exist in `docs/reports/`. PR creation is the next (and final) step, still unchecked. |

No other files were changed.

## Conclusion

Documentation is in sync with this diff. The plan's AC12 ("No allow-entry enumerations in AGENTS.md / CLAUDE.md / `.claude/rules/`") was designed to prevent exactly this drift and has held: there is no doc that enumerates `Bash(...)` entries outside the plan file and its reports. The only update required was the plan's own progress checklist.

Proceed to `/codex-review` (optional) then `/pr`.
