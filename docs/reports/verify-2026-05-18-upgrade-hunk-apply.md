# Verify report: upgrade-hunk-apply

- Date: 2026-05-18
- Branch: `feat/99/upgrade-hunk-apply`
- Plan: `docs/plans/active/2026-05-18-upgrade-hunk-apply.md`
- Verdict: PASS

## Static Verification

- `git diff --check`: PASS
- `./scripts/run-static-verify.sh`: PASS on escalated final rerun.
- Evidence: `docs/evidence/verify-2026-05-18-040005.log`

Two sandboxed attempts failed before the escalated pass:

- `docs/evidence/verify-2026-05-18-035628.log`: `staticcheck` could not write to `~/Library/Caches/staticcheck`.
- `docs/evidence/verify-2026-05-18-035655.log`: same cache restriction after moving Go and golangci-lint caches.
- `docs/evidence/verify-2026-05-18-035734.log`: changing `HOME` moved the module cache and network was unavailable, so dependencies could not be downloaded.

These were environment restrictions, not code findings. The final verify run used the normal development environment and passed.

## Acceptance Criteria

- Baseline-missing / v1 fallback remains legacy `overwrite / skip / diff`: PASS.
- Baseline-available conflicts expose hunk-level `apply / keep / edit / skip file`: PASS.
- `keep local hunk` records managed partial state instead of unmanaged file ownership: PASS.
- `skip file` preserves local content and marks unmanaged after confirmation: PASS.
- Prompt output excludes `next` and `quit`: PASS.
- Hunk choices are staged until pre-apply summary confirmation: PASS.
- Summary `N` / EOF leaves target file, manifest, and baseline unchanged: PASS.
- Summary `y` writes resolved content and updates manifest/baseline metadata: PASS.
- Partial/all-template states are recorded as `partial` / `managed`: PASS.
- Normal diff UI omits hunk headers and hash summaries: PASS.
- Missing editor / editor failure is non-destructive and re-prompts: PASS.

## Documentation Drift

No remaining drift found for current behavior. Updated README, CLI spec, tech-debt, and this plan's progress checklist.
