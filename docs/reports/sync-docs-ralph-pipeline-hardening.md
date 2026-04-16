# Sync-docs report: Ralph Pipeline Hardening

- **Date:** 2026-04-15
- **Plan:** feat/ralph-pipeline-hardening
- **Base branch:** main
- **Reviewer:** doc-maintainer (subagent)
- **Round:** Codex Round 3 fixes

## Summary

Documentation sync completed after Codex Round 3 fixes. All implementation changes are properly reflected in product and harness documentation. No documentation drift detected.

## Changes made

| File | Type | Change |
|------|------|--------|
| `docs/tech-debt/README.md` | Documentation update | Removed stale exit code collision entry (FIXED in commit b951bd0) |
| `AGENTS.md` | Repository map sync | Added `ralph-config.sh` to scripts list |
| `README.md` | Safety rails documentation | Added slice timeout and signal handler features to Ralph Loop overview |
| `docs/recipes/ralph-loop.md` | Configuration documentation | Added full "Configuration via environment variables" section with all RALPH_* vars and CLI override example |

## Verification

### Product-level documentation

| Document | Status | Notes |
|----------|--------|-------|
| `README.md` | ✓ Current | Ralph Loop safety rails section updated with new features (slice timeout, signal handlers) |
| `AGENTS.md` | ✓ Current | Scripts list updated to include `ralph-config.sh` |
| `CLAUDE.md` | ✓ No changes needed | Unchanged — post-implementation pipeline order already correct |
| `.claude/rules/` | ✓ No changes needed | Rules do not reference implementation-specific exit codes or configuration |
| `docs/quality/` | ✓ No changes needed | Definition of done and quality gates already aligned with current workflow |

### Harness-internal consistency

| Item | Status | Notes |
|------|--------|-------|
| **Scripts added/removed** | ✓ Sync complete | `ralph-config.sh` added in commit 9f42cca; AGENTS.md line 56 updated |
| **Safety rails documented** | ✓ Complete | README.md and recipes/ralph-loop.md both document new features (timeouts, signal handlers, validation) |
| **Configuration documentation** | ✓ Complete | New section in recipes/ralph-loop.md covers all env vars, defaults, and CLI override priority |
| **Cross-references** | ✓ All valid | post-implementation-pipeline.md references in subagent-policy.md verified; exit code 2 handler in ralph-pipeline.sh (line 986) correctly calls `_finalize "gh_unavailable"` |
| **Tech debt tracking** | ✓ Updated | Stale exit code collision entry removed; two remaining entries documented with proper impact and triggers |

### No drift detected in

- Pipeline order (still `/self-review → /verify → /test → /sync-docs → /codex-review → /pr`)
- Subagent delegation policy (unchanged)
- PR workflow and checklist
- Definition of done criteria
- Hook configuration references

## Specific fix notes

### Exit code collision (FIXED)

**What was fixed:** Commit b951bd0 changed `gh_unavailable` to return exit code 2 instead of 1, eliminating the collision with codex `ACTION_REQUIRED` (which returns 1).

**Documentation impact:** The stale tech-debt entry documenting this collision as unresolved was removed from `docs/tech-debt/README.md`. The fix is evidenced in ralph-pipeline.sh:
- Line 778: `return 2` with comment "distinct from 1 (ACTION_REQUIRED)"
- Line 986: `case 2)` handler for "Terminal config error (e.g., gh_unavailable)"

### New configuration module

**What was added:** Commit 9f42cca introduced `scripts/ralph-config.sh` to centralize all pipeline settings (model, effort, permission mode, iteration caps, timeouts).

**Documentation impact:**
- AGENTS.md (line 56): Added `ralph-config.sh` to scripts list
- README.md (Ralph Loop section): Noted that "All pipeline settings... are configurable via environment variables through `scripts/ralph-config.sh`"
- docs/recipes/ralph-loop.md: Added new "Configuration via environment variables" section with full table of all settings

### Signal handling hardening

**What was fixed:** Commits 23a9e8a and others separated INT/TERM and EXIT traps with `_INTERRUPTED` flag for correct signal/exit discrimination.

**Documentation impact:** docs/recipes/ralph-loop.md safety rails section (line 128) now includes "Separate INT/TERM and EXIT traps with `_INTERRUPTED` flag for clean signal/exit discrimination"

## Cross-checks

- ✓ `grep ralph-config.sh` returns references in AGENTS.md, README.md, and recipes/ralph-loop.md (all expected)
- ✓ `grep post-implementation-pipeline.md` verifies references in subagent-policy.md are still valid
- ✓ Exit code 2 handler in ralph-pipeline.sh matches documentation claim in recipes/ralph-loop.md ("Terminal config error")
- ✓ All environment variables documented in recipes match defaults in ralph-config.sh
- ✓ No hardcoded exit codes or config values in documentation that differ from scripts

## Remaining gaps

None identified. All implementation changes are reflected in product and harness documentation.

## Next steps

1. Commit the tech-debt documentation update
2. Create PR from branch
3. CI verify passes (no code changes)
4. Human review and merge
