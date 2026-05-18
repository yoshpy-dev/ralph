# Sync-docs report: upgrade edit conflict markers

- Date: 2026-05-19
- Scope: `ralph upgrade` file-level conflict resolution docs
- Verdict: PASS

## Updated Surfaces

| Surface | Result |
| --- | --- |
| `README.md` | Documents the baseline-backed `[a]pply template file / [k]eep local file / [e]dit file` prompt, Git-style marker edit flow, and unresolved-marker rejection. |
| `docs/specs/2026-04-16-ralph-cli-tool.md` | Documents file-level `apply / keep / edit`, managed partial state for `keep local file`, marker-based editing, and summary confirmation semantics. |
| Current PR reports | Self-review, verify, and test reports now reflect file-level choices, managed partial keep behavior, empty-editor regression coverage, and terminology cleanup. |

## Drift Check

- Current implementation and current user-facing docs agree on `[a]pply template file / [k]eep local file / [e]dit file`.
- Baseline-missing v1 fallback remains documented as `[o]verwrite / [s]kip / [d]iff`.
- Historical archived plans and reports were left unchanged as evidence of prior work, not current behavior.

## Remaining Gaps

None.
