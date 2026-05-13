# sync-docs report — add-terraform-language-pack

- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md`
- Branch: `feat/52/add-terraform-language-pack`
- Date: 2026-05-13
- Pipeline cycle: 1

## Summary

Reconciled documentation drift after the Terraform pack implementation. No behavior or contract drift detected — only enumerated lists and tracking ledgers needed updating.

## Changes

| File | Change | Reason |
|------|--------|--------|
| `README.md` | Added `terraform/` to the language-pack roster sentence in the "Language packs" section | The README enumerates included starter packs by name; the new pack must appear there to be discoverable. README.md is a KNOWN_DIFF in `scripts/check-sync.sh`, so `templates/base/README.md` does not need to match. |
| `docs/tech-debt/README.md` | Added 2 rows: (1) `ralph pack add <lang>` pathing bug (`internal/cli/pack.go:64-67`) — Codex HIGH#1, out of scope. (2) `tfsec` upstream-archived — long-term plan to drop in favor of `trivy config` only. | Both are explicit Non-goals / Risks in the plan, and the self-review report flagged them as tech debt to track. |
| `docs/plans/active/2026-05-13-add-terraform-language-pack.md` | Ticked "Review artifact created", "Verification artifact created", "Test artifact created"; added "Sync-docs artifact created" row | The artifacts now exist on disk; the checklist must reflect that before `/pr`. |

## Not touched (verified non-drift)

- `AGENTS.md` — Repo map enumerates `packs/languages/` but does not list individual languages; the new pack is auto-discoverable. No edit needed.
- `CLAUDE.md` — Generic and small; nothing pack-specific.
- `docs/architecture/*` — Stack-agnostic; no per-pack references.
- `docs/quality/definition-of-done.md` — Workflow-level only.
- `.claude/rules/*` and `.claude/skills/*` — Untouched by this PR; `scripts/check-skill-sync.sh` passes.
- `templates/base/.claude/rules/terraform.md` — Already mirrored in slice 2.
- `templates/base/docs/recipes/adding-a-language-pack.md` — Already mirrored in slice 4.
- `templates/base/scripts/detect-languages.sh` — Already mirrored in slice 3.
- `internal/scaffold/embed.go` — Auto-discovers from `templates/packs/`; no code change needed for new packs.

## Gates

- `./scripts/check-sync.sh` → PASS (148 identical, 0 drifted, 0 root-only)
- `./scripts/check-skill-sync.sh` → PASS (13 skills in lock-step)

## Verdict

PASS. Documentation is now aligned with implementation. Ready for `/cross-review`.
