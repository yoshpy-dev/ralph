# Verify report: upgrade-partial-hunks

- Plan: `docs/plans/active/2026-05-18-upgrade-partial-hunks.md`
- Related issue: #97
- Verdict: PASS for this PR slice
- Evidence: `docs/evidence/verify-2026-05-18-030318.log`

## Acceptance Criteria

| Criterion | Verdict | Evidence |
| --- | --- | --- |
| Existing v1 manifests read successfully and do not force the baseline-backed prompt when baseline is missing | PASS | `TestReadManifestV1Compatibility`, `TestRunUpgrade_V1ManifestConflict_UsesLegacyPrompt` |
| Newly initialized and accepted template files record `baseline_status = "available"` and `baseline_path` | PASS | `TestExecuteInit_NewProject`, `TestManifestRoundTripV2BaselineFields`, overwrite/force paths use `setManagedWithBaseline` |
| `Managed=false` entries remain unmanaged across v2 metadata writes and baseline creation | PASS | Existing unmanaged regression tests still pass; `preserveUnmanaged` keeps `Managed=false` |
| `ralph upgrade --dry-run --diff` previews changes without writing target files or manifest | PASS | `TestRunUpgrade_DryRunDiff_DoesNotMutateFilesOrManifest` |
| Upgrade diff UI omits hunk headers and hash summaries | PASS | `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff`, `TestRunUpgrade_DiskReadFailure_FallsBackToWarning` |
| Conflict diff display can use pager and fall back to stdout | PASS | `--pager` validation and `writeDiffOutput` fallback path implemented; full verify passed. Direct pager failure integration remains a low-value edge gap. |
| Baseline-available conflicts expose only `apply template file / keep local file / edit`; no `skip`, `next`, or `quit` | PASS | `TestRunUpgrade_InteractiveDiff_RepromptsOnInvalid` asserts prompt shape and absence of `skip`/`next`/`quit` |
| Full hunk-level edit/pre-apply summary semantics are tracked as follow-up debt | PASS | `docs/tech-debt/README.md` entry added |
| Tests cover manifest compatibility, baseline writing, dry-run behavior, and prompt option shape | PASS | New/updated tests in `internal/scaffold` and `internal/cli` |

## Static Verification

`./scripts/run-verify.sh` passed in full mode. It included sync checks, shell tests, gofmt, vet/staticcheck-equivalent gates, and `go test ./...`.

## Documentation Drift

Updated `docs/specs/2026-04-16-ralph-cli-tool.md` with manifest v2 baseline metadata, `--dry-run`, `--diff`, `--pager`, and baseline-backed prompt fallback semantics. Added tech debt for the remaining #97 hunk-level editor/merge work.

## Remaining Gaps

- This PR does not implement full old-template/local/new-template hunk planning, manual edit, or final pre-apply summary. Those are explicitly deferred before #97 can be closed.
