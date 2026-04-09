# Self-review report: Ralph Loop v2 — Codex fix re-run

- Date: 2026-04-10
- Plan: docs/plans/archive/2026-04-09-ralph-loop-v2.md
- Reviewer: reviewer subagent (claude-sonnet-4-6)
- Scope: Focused re-run after Codex finding fixes (commit 5f852b4). Reviews the two fix changes plus checks for any new issues introduced. Diff quality only — no spec compliance or coverage evaluation.

## Evidence reviewed

- `git show 5f852b4` — fix commit: two-line change to ralph-orchestrator.sh, two-line change to ralph-pipeline.sh, plus new codex-triage report
- `scripts/ralph-orchestrator.sh:638` — dependency slug normalization (Codex P1 fix target)
- `scripts/ralph-pipeline.sh:621–622` — base branch fallback (Codex P2 fix target)
- Prior self-review report: `docs/reports/self-review-2026-04-10-ralph-loop-v2.md` (findings already recorded)
- `docs/tech-debt/README.md` — existing deferred items
- MEMORY.md patterns cross-checked

---

## Fix 1: Dependency slug normalization (ralph-orchestrator.sh:638)

**Before:**
```sh
_dep_slug="$(echo "$dep" | tr -d ' []' | tr '[:upper:]' '[:lower:]' | sed 's/^slice //')"
```

**After:**
```sh
_dep_slug="$(echo "$dep" | tr -d ' []' | tr '[:upper:]' '[:lower:]' | sed 's/^slice[- ]*//')"
```

**Analysis:** The old pattern `sed 's/^slice //'` failed after `tr -d ' []'` had already removed all spaces — so the literal string `"slice "` (with trailing space) could never match. The new pattern `sed 's/^slice[- ]*//'` correctly strips both `slice-` (the dash form used in filenames) and `slice` (space form after tr processing), and handles zero or more trailing separator characters. Verified with test inputs:
- `"slice 1-auth-api"` → tr removes spaces → `"slice1-auth-api"` → old sed: no match → stays `"slice1-auth-api"` (WRONG); new sed: strips `"slice"` prefix → `"1-auth-api"` (CORRECT)
- `"slice-2-db"` → tr has no effect → `"slice-2-db"` → new sed: strips `"slice-"` → `"2-db"` (CORRECT)
- Matches the slug format produced by `parse_slices`: `basename "slice-1-auth-api.md" .md | sed 's/^slice-//'` → `"1-auth-api"`

The fix is correct. No new issues introduced at this call site.

**Residual consideration:** If a dependency field contains `"none"` (e.g., from a manually written slice file), `tr -d ' []'` returns `"none"`, `sed` has no match, and `check_slice_status "none"` is called. Since no status file for `"none"` exists, it returns `"pending"`, causing the dependent slice to wait forever. This is a pre-existing edge case — the `parse_slices` function does filter `"none"` for both inline format (line 111) and section-body format (line 137), so `"none"` should not appear in the `$d` field under correct usage. No action needed from this fix.

---

## Fix 2: Base branch fallback (ralph-pipeline.sh:621–622)

**Before:**
```sh
_base="$(git rev-parse --abbrev-ref HEAD@{upstream} 2>/dev/null | sed 's|origin/||' || echo main)"
```

**After:**
```sh
_base="$(git rev-parse --abbrev-ref HEAD@{upstream} 2>/dev/null | sed 's|origin/||')"
_base="${_base:-main}"
```

**Analysis:** The old pattern placed `|| echo main` at the pipeline level. When `git rev-parse` fails (no upstream configured), `sed` receives empty stdin and exits 0 — the `||` branch is never taken, leaving `_base=""`. The new pattern separates the assignment and uses POSIX parameter expansion `${_base:-main}` which correctly handles the empty-string case. Verified with local test:
- With no upstream: old pattern → `""`, new pattern → `"main"`
- With upstream set: both patterns → correct remote branch name (e.g., `"feat/ralph-loop-v2"`)

The fix is correct. No new issues introduced at this call site.

**Adjacent code note:** The `codex exec review --base "$_base"` on line 624 correctly uses `$_base` after the fallback is applied. The surrounding `git diff "${_base}...HEAD"` guard on line 623 also benefits from the non-empty `_base`.

---

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | The fix commit does not introduce new findings. All findings below are carry-overs from the prior review, included for completeness. | — | — |
| LOW | readability | `scripts/ralph-pipeline.sh:670` PR prompt still uses `echo "https://github.com/..." > .harness/state/pipeline/.pr-url` as an example. Agent may copy the literal text. Layer 1 (`gh pr list`) and Layer 3 (log grep) provide fallback. Unchanged from prior review. | `scripts/ralph-pipeline.sh:659–671` | Replace with `gh pr view --json url --jq '.url' > .harness/state/pipeline/.pr-url`. Already in tech-debt. |
| LOW | readability | `scripts/ralph-orchestrator.sh:441` uses unquoted HEREDOC delimiter `MERGE_EOF` inside `git merge -m "$(cat <<MERGE_EOF...)"`, which expands `$_slice_branch` and `$_int_branch`. Harmless in content (branch names have no shell-special chars) but violates repo convention for single-quoted delimiters. Unchanged from prior review. | `scripts/ralph-orchestrator.sh:441–443`; `.claude/rules/git-commit-strategy.md` | Change to `'MERGE_EOF'`. Already noted in prior report. |

---

## Positive notes

- Fix 1 (dependency slug) addresses the root cause correctly: the ordering of `tr -d ' []'` before `sed` was the actual failure mode, and the new regex handles all expected separator variants.
- Fix 2 (base branch fallback) uses idiomatic POSIX parameter expansion rather than a pipe-level `||`, which is the correct fix for the "sed exits 0 on empty input" class of bugs.
- The fix commit is minimal (4 lines changed across 2 files plus a new report). No extraneous changes, no formatting drift, no accidental includes.
- No debug code, secrets, or TODO markers introduced.
- The codex-triage report (`docs/reports/codex-triage-2026-04-10-ralph-loop-v2.md`) correctly documents the Codex findings and triage decisions before the fixes were applied.

---

## Tech debt identified

No new deferred items. The two carry-over items remain in `docs/tech-debt/README.md`:
- PR prompt example URL (LOW)
- MERGE_EOF unquoted delimiter (LOW, convention violation)

---

## Recommendation

- **Merge: YES** — The two Codex findings are correctly fixed. No new CRITICAL or HIGH issues introduced. No regression in previously-resolved findings. The LOW carry-over items are unchanged from the prior approved state.
- **Follow-ups (pre-existing, non-blocking):**
  - Replace PR prompt example URL with `gh pr view` command (LOW)
  - Use single-quoted `'MERGE_EOF'` delimiter (LOW, convention)
