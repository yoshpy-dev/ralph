# Sync-docs report: mojibake-postedit-guard

- Date: 2026-04-17
- Plan: `docs/plans/active/2026-04-17-mojibake-postedit-guard.md`
- Branch: `chore/mojibake-postedit-guard`
- Author: `doc-maintainer` subagent
- Upstream reference: Claude Code Issue #43746 (SSE chunk-boundary U+FFFD injection)

## Scope of this diff

Additive, hook-only change. New files:

- `.claude/hooks/check_mojibake.sh` (+ `templates/base/` mirror)
- `.claude/hooks/mojibake-allowlist` (+ `templates/base/` mirror)
- `scripts/verify.local.sh`
- `tests/test-check-mojibake.sh`
- `tests/fixtures/payloads/{edit,write,multiedit}.json`

Modified files:

- `.claude/settings.json` and `templates/base/.claude/settings.json` — PostToolUse matcher expanded from `Edit|Write` to `Edit|Write|MultiEdit`, and `check_mojibake.sh` added as a second entry alongside the existing `post_edit_verify.sh`.
- `scripts/check-sync.sh` — `ROOT_ONLY_EXCLUSIONS` extended with repo-only paths (`scripts/verify.local.sh`, `tests/`).
- `AGENTS.md` — one-line nested bullet under the `.claude/hooks/` entry describing the temporary mitigation and retirement trigger (already covered by `/self-review` and `/verify`).

No skills, no rules, no language packs, no CI workflows, no script entrypoints, and no public contracts were changed by this diff.

## Files updated in this sync pass

| File | Change | Why |
| --- | --- | --- |
| `docs/plans/active/2026-04-17-mojibake-postedit-guard.md` | Progress checklist: "Review artifact created", "Verification artifact created", "Test artifact created" flipped from `[ ]` to `[x]` with the corresponding report paths appended. PR checkbox left unchecked (handled by `/pr`). | Plan checklist was stale; all three artifacts exist in `docs/reports/` with PASS verdicts. Matches the pattern used by `sync-docs-2026-04-17-allow-go-and-repo-commands.md`. |
| `docs/tech-debt/README.md` | New row added for the mojibake mitigation bundle (hook + allowlist + tests + fixtures + settings entry + AGENTS.md note + check-sync.sh exclusions). | Captures the retirement trigger: "Upstream Issue #43746 closes in a released Claude Code version AND no local recurrences observed for 1 week." Keeps the removal contract in the same place as other deferred work so a future reader finds it without re-reading the plan. |

## Files checked and left unchanged

| Doc / contract | Result | Evidence |
| --- | --- | --- |
| `AGENTS.md` Repo map | Already synced | `AGENTS.md:66` carries the one-line `check_mojibake.sh` + `mojibake-allowlist` note. Per user instruction, no further edits. `/verify` explicitly flagged this as "PASS (1-line consolidated form)". |
| `CLAUDE.md` | No change needed | Scoped to skill orchestration and always-on defaults; does not enumerate individual hooks. Per user instruction, not edited. |
| `templates/base/AGENTS.md` | No change needed | KNOWN_DIFF per `scripts/check-sync.sh:83`. The template intentionally omits repo-specific hook notes; this matches the plan's scope ("scaffolded projects get the hook; the scaffolded AGENTS.md stays generic"). |
| `templates/base/CLAUDE.md` | No change needed | Scoped to skill defaults; no hook enumeration. |
| `README.md` — "Hook configuration" section (L178-190) | No change needed | Describes hooks shipped in `settings.json` at a behavior level (session start, prompt gate, bash guard, edit/write verification reminders, tool failure feedback, compaction checkpoints, session end). Adding a "mojibake guard" bullet would expand scope beyond the plan and would need to be torn down when the hook retires. The existing "Edit/write verification reminders" bullet is generic enough to accommodate both hooks under the shared matcher. |
| `README.md` — Operating loop (L127-176) | No change needed | Pipeline order and skill roster unchanged; no drift. |
| `docs/architecture/repo-map.md` | No change needed | References `.claude/hooks/` generically as "deterministic hook scripts" and `.claude/settings.json` generically as "hook and permission configuration". Accurate at that granularity for an additive hook. |
| `docs/architecture/design-principles.md` | No change needed | `grep -n 'PostToolUse\|check_mojibake\|mojibake\|hooks/'` returns 0 hits. High-level principles file; no hook enumeration. |
| `docs/quality/definition-of-done.md` | No change needed | DoD checklist is pipeline-shaped (artifacts, plans, pipeline order). A new PostToolUse hook is not a DoD item. |
| `docs/quality/quality-gates.md` | No change needed | Lists gate policy (must-pass scripts, CI workflows, pipeline-mode gates). `scripts/verify.local.sh` is auto-invoked by `run-verify.sh` (the file is already under the "must pass locally" gate "`./scripts/run-verify.sh`") — no new verifier needs listing. |
| `.claude/rules/architecture.md` | No change needed | Grep-ability rules unchanged; new hook names are grep-able. |
| `.claude/rules/documentation.md` | No change needed | No new doc types introduced. |
| `.claude/rules/planning.md` | No change needed | Plan structure unchanged. |
| `.claude/rules/testing.md` | No change needed | Tests were added (`tests/test-check-mojibake.sh`, 11/11 PASS); rule is already satisfied. |
| `.claude/rules/git-commit-strategy.md` | No change needed | 5-commit slicing discipline respected; no new pattern. |
| `.claude/rules/post-implementation-pipeline.md` | No change needed | Pipeline order unchanged. |
| `.claude/rules/subagent-policy.md` | No change needed | Subagent set unchanged. |
| `.claude/rules/<lang>.md` (python, typescript, golang, rust, dart) | No change needed | No language-pack changes in this diff. |
| `.claude/skills/audit-harness/SKILL.md` | No change needed | Audit prompt already includes `.claude/hooks/` as an inspect target; the new hook will be surfaced in future audits without any SKILL.md change. |
| `.claude/skills/sync-docs/SKILL.md` | No change needed | Skill scope covers "hooks added/removed"; the additions are now reflected in settings.json + AGENTS.md + this report. |
| `scripts/check-sync.sh` | Already synced | `ROOT_ONLY_EXCLUSIONS` now covers `scripts/verify.local.sh` and the `tests/` prefix. `check-sync.sh` itself is also covered by the existing `docs/reports/sync-docs-` prefix exclusion, so this report is excluded from ROOT_ONLY detection. Verified via `/verify`'s PASS row (`IDENTICAL: 107, DRIFTED: 0, ROOT_ONLY: 0`). |
| `scripts/run-verify.sh` | No change needed | Auto-invokes `verify.local.sh` as documented. |

## Drift analysis: none of the sync-docs checklist items triggered a doc edit

Cross-referenced against `.claude/skills/sync-docs/SKILL.md`'s checklist:

- **Skills added/removed/renamed**: none.
- **Hooks added/removed**: one hook added; settings.json entries match (both root and template); audit-harness target already includes `.claude/hooks/`; AGENTS.md note already present.
- **Rules added/removed**: none.
- **Language packs added/removed**: none.
- **Scripts added/removed**: `scripts/verify.local.sh` added. README Quick Start still lists `./scripts/run-verify.sh` which transitively runs `verify.local.sh`; `docs/architecture/repo-map.md` lists scripts at a role level ("verification (`run-verify.sh`, `run-static-verify.sh`, `run-test.sh`), CI checks, commit safety, language detection...") — `verify.local.sh` fits under the existing "verification" umbrella and does not warrant a separate enumeration, since it is a repo-local inner ring not present in scaffolded projects (by design per `check-sync.sh` exclusions). **No change.**
- **Quality gates changed**: none (see unchanged table above).
- **PR skill consistency**: `/pr` pre-checks read `docs/reports/self-review-*.md`, `verify-*.md`, `test-*.md` — all present and PASS. No drift.

## Residual doc drift

| Item | Severity | Disposition |
| --- | --- | --- |
| Plan's acceptance-criteria checklist (plan L60-72) is still `- [ ]` on every row. | Low | Out of scope for `/sync-docs`. `/verify` flagged the same drift as advisory ("Recommendation: `/sync-docs` or `/pr` flips the AC checkboxes to `[x]` based on this verify report"). We updated only the pipeline-stage "Progress checklist" section (L167-175) per the task instructions; the AC list is a separate historical record and is not the canonical pipeline progress tracker. `/pr` archives the plan as-is; the AC checkbox state will be frozen in archive. Leaving it is acceptable because `/verify`'s PASS row already records AC satisfaction authoritatively. |
| Hook stderr wording differs from plan AC3 text ("Re-read the file and rewrite the corrupted section without the replacement character." vs. "Re-read and rewrite the corrupted sections."). | Low | Self-review LOW-7 and verify "text deviation noted" both captured this. Meaning preserved; plan wording was illustrative, not normative. Not worth churning either side. |
| Plan mentions "2 行注記" for AGENTS.md; implementation landed one bullet line. | Low | Self-review LOW-7 captured this. The one-line form fits AGENTS.md's "keep short" rule. Not a drift to fix. |

## Conclusion

Two files updated in this pass:

- `docs/plans/active/2026-04-17-mojibake-postedit-guard.md` — progress checklist flipped for the three post-implementation artifacts.
- `docs/tech-debt/README.md` — new row for the mojibake-hook bundle with the Issue #43746 retirement trigger.

Everything else in the repo (product docs, harness docs, templates, rules, skills, scripts, quality gates) was already in sync because this diff is a narrow, additive, well-scoped mitigation. AGENTS.md and CLAUDE.md were intentionally not touched (per user instruction); the AGENTS.md one-line note was in place before `/sync-docs` ran.

Proceed to `/codex-review` (optional) then `/pr`.

## Re-sync-docs after Codex fixes (commit 306b23a)

- Date: 2026-04-17
- Trigger: Codex-review produced 1 ACTION_REQUIRED (P3 matcher symmetry), 1 WORTH_CONSIDERING (P2 mode split), 1 DISMISSED with hardening (P1). Fix commit `306b23a` applied all three; re-self-review kept MERGE verdict, re-verify kept PASS delta, re-test kept PASS (88 assertions). Per `post-implementation-pipeline.md`, `/sync-docs` must re-run even when only `/self-review` through `/test` re-ran.

### Behavior changes in 306b23a that could touch docs

1. **`PostToolUseFailure` matcher expanded** (`Bash|Edit|Write` → `Bash|Edit|Write|MultiEdit`) in both `.claude/settings.json` and `templates/base/.claude/settings.json`. Symmetric with the `PostToolUse` matcher already changed in the original slice.
2. **`scripts/verify.local.sh` now honors `HARNESS_VERIFY_MODE`** (`static`/`test`/`all`). When `run-static-verify.sh` runs, `verify.local.sh` now executes only shellcheck/sh-n/jq/check-sync; when `run-test.sh` runs, it executes only `tests/test-check-mojibake.sh`. Default (`all`) is unchanged.

### Docs checked and left unchanged

| File | Lines checked | Result |
| --- | --- | --- |
| `docs/quality/quality-gates.md` | L23-28 (must-pass-locally gate list) | Description already refers to the wrappers generically — `run-static-verify.sh` "wrapper for `HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh`" and `run-test.sh` "wrapper for `HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh`". With 306b23a, `verify.local.sh` now honors the mode that `run-verify.sh` forwards, so the wrappers now actually deliver the documented split for the hook smoke tests. The text was already correct in intent; no edit needed. |
| `docs/quality/definition-of-done.md` | Full file | `verify.local.sh` and `HARNESS_VERIFY_MODE` are not mentioned; DoD is pipeline-shaped, not script-enumerating. No edit needed. |
| `README.md` | L104 | The single reference (`packs/languages/*/verify.sh` or `scripts/verify.local.sh`) treats `verify.local.sh` as a customization entry point at init time, with no behavior description to drift. No edit needed. |
| `AGENTS.md`, `CLAUDE.md` | Per task instruction, not edited. | — |
| `.claude/skills/loop/prompts/pipeline-verify.md`, `pipeline-test.md` | `HARNESS_VERIFY_MODE=static|test ./scripts/run-verify.sh` invocations | These were already the two callers that motivated the mode-respecting fix. The prompts stay identical; 306b23a makes their intent actually apply to `verify.local.sh`. No edit needed. |
| `docs/reports/codex-triage-mojibake-postedit-guard.md` | Full file | Kept as the historical triage snapshot that drove commit 306b23a. Re-classifying P2 (WORTH_CONSIDERING) or the P3 table after-the-fact would rewrite history. The resolution is recorded here instead: **P3 resolved** by the matcher-symmetry fix (both `.claude/settings.json` lines now read `Bash|Edit|Write|MultiEdit`), **P2 resolved** by the `HARNESS_VERIFY_MODE` case block in `verify.local.sh` L20-27, **P1 hardened** by adding `dirname`/`env`/`ln`/`test` to the linked tool set in `tests/test-check-mojibake.sh` Case E (still DISMISSED as root-cause). |

### Docs updated in this pass

None. The two behavior changes in 306b23a are **narrow enough that no external doc copy described them specifically**:

- The `PostToolUseFailure` matcher change is invisible at the docs layer — no document enumerates per-hook matchers beyond the settings.json itself.
- The `HARNESS_VERIFY_MODE` support in `verify.local.sh` brings it into compliance with what `quality-gates.md:26-27` was already promising; the doc was aspirational before this fix, descriptive after.

### Drift re-check against sync-docs SKILL.md checklist

- **Skills added/removed/renamed**: none.
- **Hooks added/removed**: none in 306b23a (only a matcher string changed inside existing `PostToolUseFailure` entry).
- **Rules added/removed**: none.
- **Language packs added/removed**: none.
- **Scripts added/removed**: `verify.local.sh` content changed (not added/removed); no new script entrypoints.
- **Quality gates changed**: Gate contract `HARNESS_VERIFY_MODE` is now actually honored by `verify.local.sh`. The doc text did not change; the runtime behavior now matches it.
- **PR skill consistency**: pipeline order and pre-checks unchanged; `/pr` still reads the same four report types.

### Conclusion

Re-sync pass made **zero doc edits**. The prior sync-docs pass's conclusion ("Two files updated… everything else already in sync") still holds for the whole branch including 306b23a. Proceed to `/pr`.

## Re-sync-docs after post_edit_verify fix (commit 29d71a2)

- Date: 2026-04-17
- Trigger: Codex re-review P3-new exposed a pre-existing silent-no-op in `post_edit_verify.sh` (extracted top-level `file_path` against a payload where the field nests under `tool_input`). Commit 29d71a2 fixes `lib_json.sh` (accept dotted paths, backward-compatible for `pre_bash_guard.sh`'s non-dotted `"command"`) and updates the `post_edit_verify.sh` call site to `tool_input.file_path`. Re-self-review / re-verify / re-test all PASS.

### Docs checked for drift

| File | Checked for | Result |
| --- | --- | --- |
| `.claude/hooks/lib_json.sh` header comment | "top-level only" language that would contradict the new dotted-path contract | **Already synced in 29d71a2 itself** — L4-6 now reads "The field argument accepts a dotted path (e.g. `tool_input.file_path`)…Top-level keys work without a dot." Inline contract is accurate. No edit needed. |
| `README.md` "Hook configuration" section (L178-190) | any description of edited-files.log behavior or `lib_json.sh` extraction contract | No mention of either — the section describes hook categories at behavior level ("Edit/write verification reminders"). No drift. |
| `AGENTS.md`, `CLAUDE.md` | any mention of `lib_json.sh` / `edited-files.log` | None. Per task instruction, not edited. |
| `docs/architecture/repo-map.md`, `docs/architecture/design-principles.md` | any mention of `lib_json.sh` / `edited-files.log` | None (`.claude/hooks/` referenced generically). No drift. |
| `docs/quality/*.md` | any dependency on `edited-files.log` being populated | None. Quality gates are pipeline-shaped, not log-shaped. No drift. |
| `.claude/rules/*.md` | any reference to the extraction contract or log | None. No drift. |
| `templates/base/.claude/hooks/lib_json.sh` + `post_edit_verify.sh` | mirror parity | `cmp` → exit 0 (verified via `/verify` delta pass). No drift. |

### Docs updated in this pass

**None.** The extraction contract is documented exactly where it should be — in the `lib_json.sh` header comment, which 29d71a2 updated in the same commit as the behavior change. `.harness/state/edited-files.log` is runtime state (per AGENTS.md: "not canonical truth"), and no user-facing doc previously described it, so no retrofitted prose is needed. The branch is now fully in sync across the three sync-docs passes.
