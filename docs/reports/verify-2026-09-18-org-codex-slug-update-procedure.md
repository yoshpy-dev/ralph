# Verify report: org-codex-slug-update-procedure

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-org-codex-slug-update-procedure.md` (issue #156)
- Verifier: `verifier` subagent (Claude Code, standard flow)
- Diff scope: `git diff 93cf3bb...HEAD` on `docs/org-codex-slug-update-procedure` (HEAD `411ecf2`), 8 files, +228/-0. Docs-only: `.claude/skills/org/SKILL.md` + 3 mirrors, `docs/specs/2026-08-01-org-runtime.md`, plus plan/self-review/insight-event bookkeeping. No Go changes.
- Scope note: AC-1, AC-2, AC-4 are in scope for this pass. AC-3 (issue comment) and AC-5 (`Refs #156` in the PR body) are deferred to `/pr` per the task's instructions and the plan's own Implementation outline (Slice C runs at `/pr`) — not evaluated as fail here. No behavioral test suite was run; `git status --porcelain` confirmed a clean working tree before and after this pass.

## Overall verdict: Pass

AC-1, AC-2, and AC-4 all confirmed with direct evidence. Static analysis is clean across all sub-gates including the two sync gates the plan specifically calls out. Every technical claim named in the task brief (9 items) was independently verified against the current code, not just trusted from the self-review report — all 9 check out exactly as stated. No new doc drift found beyond the two already-recorded, deliberate deviations.

## Static analysis

`./scripts/run-static-verify.sh` (changed-language scope resolved to `docs_only`, no language pack selected):

- `scripts/check-sync.sh` → `PASS: all files in sync.` (`IDENTICAL: 157`, `DRIFTED: 0`, `ROOT_ONLY: 0`)
- `scripts/check-pipeline-sync.sh` → OK, all 6 canonical-order references in sync
- `scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`
- `scripts/check-template-purity.sh` → `PASS: no meta-repo-specific references found in templates.`
- Overall: `==> All verifiers passed.`
- Evidence: `docs/evidence/verify-2026-09-18-041855.log` (gitignored per `docs/evidence/*.log`)

## Acceptance criteria

| AC | Verdict | Evidence |
| --- | --- | --- |
| AC-1 | Pass | `grep -c 'seed' .claude/skills/org/SKILL.md` → 2; `grep -c 'brew upgrade' .claude/skills/org/SKILL.md` → 1 (both ≥1 as required). The shipped paragraph (`.claude/skills/org/SKILL.md:61-76`) states the doctor-warn-driven `ralph.toml` edit, that `ralph.toml` is seed-once ("初回 `ralph init` で生成されたあと `ralph upgrade` は触らない"), and that following the default requires binary update → `ralph version` → `ralph upgrade` (correctly scoped to "上流の新しい既定は…確認できる", not overstated as a hard precondition — see self-review M2 resolution). 4-way `cmp`: `.claude/skills/org/SKILL.md` vs `.agents/skills/org/SKILL.md`, vs `templates/base/.claude/skills/org/SKILL.md`, vs `templates/base/.agents/skills/org/SKILL.md` — all three exit 0 (byte-identical, no diff output). `./scripts/check-skill-sync.sh` and `./scripts/check-sync.sh` both pass (see Static analysis). |
| AC-2 | Pass | `docs/specs/2026-08-01-org-runtime.md` has `### 運用ノート: 既定 codex スラッグの更新手順(2026-09-18、issue #156)` placed directly after the `### 2026-09-16 改訂` section and before `## Background and problem`. `grep -c` for each of the 16 required terms (`config.go`, `templates/base/ralph.toml`, `ralph-config.sh`, `SKILL.md`, `run-verify.sh`, `ralph doctor`, `ralph upgrade`, `brew upgrade`, `ralph version`, `EmbeddedFS`, `mtime`, `codex --version`, `config_test.go`, `defaults_sync_test.go`, `envelope_summary_test.go`, `org_test.go`) returned ≥1 for every term (counts ranged 1–7; long-line markdown means some multi-occurrence terms still count as 1 line-match, which satisfies the AC's "each ≥1" wording). The two-stage distribution rollout ((c): binary update first, `ralph upgrade` second) and the pre-observation cache-refresh procedure ((d)) are both spelled out in full, matching the plan's Deviation-notes-recorded fix from self-review L4 (see Documentation drift below). |
| AC-3 | Deferred to `/pr` | Not evaluated in this pass. Plan Implementation outline assigns the issue comment to "Slice C (`/pr` 段階)"; `[ ]` unchecked in the plan's Acceptance criteria section is expected at this point in the pipeline, not a fail. |
| AC-4 | Pass | `./scripts/run-static-verify.sh` ran clean end-to-end (see Static analysis above) — this is the deterministic check the plan names for AC-4 ("文書のみの変更だが、sync 系ゲートを含むため"). Both sync gates the plan calls out (`check-skill-sync.sh`, `check-sync.sh`) are part of that run and pass. |
| AC-5 | Deferred to `/pr` | Not evaluated in this pass; the PR body (`Refs #156`, not `Closes`) is authored at `/pr` time per the plan's Implementation outline. |

## Technical-claim verification (against current repo state, not the self-review's word)

All 9 claims named in the task brief were checked directly:

1. `checkCodexModelSlugs` (`internal/cli/doctor_codex_models.go:72-98`) resolves `$CODEX_HOME/models_cache.json` else `~/.codex/models_cache.json` (`:29-38`) and does a single `os.ReadFile` (`:98`) — no `exec.Command`, no `ModTime` call anywhere in the file. Confirmed.
2. `ralph doctor --probe-models` exists (`internal/cli/doctor.go:36`); a codex probe failure is explicitly labeled `advisory: codex --model support on exec is best-effort upstream` (`:811`). Confirmed.
3. `cmd/ralph/main.go:18` sets `scaffold.EmbeddedFS = ralph.TemplatesFS`; `classifySeed` in `internal/upgrade/replaceplan.go` appends a seed `AdvisoryEntry` when the template hash differs from the recorded one (`:417`, `:465`), rendered under `docs/reports/upgrade-<version>-<date>.md` (`internal/upgrade/report.go:202`). Confirmed.
4. `docs/specs/2026-08-17-overlay-scaffold-v2.md:45` lists `.claude/skills/` and `scripts/` under L1 core (full replace); `:49` lists `ralph.toml` and `docs/recipes/` under L5 seed-once (generate-if-missing, advisory-diff only). Confirmed.
5. `internal/config/defaults_sync_test.go` opens exactly `scripts/ralph-config.sh` (root, `:95`) and `templates/base/ralph.toml` (`:102`), comparing both to `config.Default()` — no third file read. `templates/base/scripts/ralph-config.sh` is not named by the test; it is covered generically by `scripts/check-sync.sh`'s whole-repo diff walk (it is not in `ROOT_ONLY_EXCLUSIONS` or `KNOWN_DIFFS`, so any drift there shows up as `DRIFTED`). Confirmed — matches the self-review's L1-fix attribution exactly.
6. `driver_pool`-only configs: `internal/config/config.go:251` computes `cfg.Org.ModelPool = filterModelPoolByDrivers(Default().Org.ModelPool, cfg.Org.DriverPool)` when `model_pool` is unset — the inherited default is narrowed to declared drivers. Confirmed.
7. `gpt-6-astra` is hard-coded in all three named test files: `internal/config/config_test.go:127,743`; `internal/org/envelope_summary_test.go:84,95-96,153,155`; `internal/cli/org_test.go:775,777`. Confirmed.
8. Both "outside the enumeration" surfaces the spec (a) names still contain the slug: `.claude/skills/org/SKILL.md:39` ("プール先頭…codex は `gpt-6-astra`", under the `## 前提` section at `:19`, which matches the spec's post-fix "「前提」節" pointer — the earlier "冒頭" wording from cycle 1 was corrected in the 07cdd0d follow-up) and `templates/base/ralph.toml:55` (`# reviewer = ["opus", "gpt-6-astra"]` inside the `[org.roles]` comment example). Confirmed.
9. `docs/specs/` is in `scripts/check-sync.sh`'s `ROOT_ONLY_EXCLUSIONS` array (`:56`), so the new maintainer-only spec section correctly avoids creating template drift. Confirmed.

No inaccurate claim was found in either the shipped prose or the self-review's evidence trail.

## Documentation drift

- **Two deliberate, plan-recorded deviations — both confirmed as shipped, not stale.** (1) Scope item 2 (d)'s original "2〜4 週間続けて消えたままの場合だけ" wording was superseded by a firm criterion after self-review L4: the shipped spec text reads "2 週間連続して消えたままなら…判断に入り、迷う場合は 4 週間まで観測を延長する。それ未満では外さない。" (`docs/specs/2026-08-01-org-runtime.md:22`). The plan's Deviation notes (last two entries) record this change and its rationale; no stale range remains anywhere in the shipped copy. (2) The self-review's follow-up round at `07cdd0d` (probe-failure asymmetry, `driver_pool`-only narrowing, "冒頭"→"前提" pointer fix) is fully landed in the current SKILL.md and spec text — verified directly against the file contents (see claim 8 above and the shipped paragraph at `.claude/skills/org/SKILL.md:66-68`), not merely trusted from the self-review's own revalidation table.
- **Skill paragraph (operator audience) and spec section (maintainer audience) agree on the load-bearing facts**, as the plan's Design decisions require: both state the cache is read-only/no-refresh, both describe the same two-stage distribution (binary update → `ralph upgrade`), and neither overstates "順が要る" as a hard precondition post-fix (M2's resolution: "バイナリを差し替えた時点で新既定に切り替わる" / "`ralph upgrade` は…実効プールは変えない" appear on both surfaces).
- **The spec's own `### 2026-09-16 改訂` note (c) remains internally consistent** with the new section: (c) states the current default codex slugs (`gpt-6-astra`/`gpt-5.6-sol`/`gpt-5.6-terra`/`gpt-5.6-luna`/`gpt-5.5`), and the new "運用ノート" section neither contradicts nor duplicates that list — it only adds the update procedure, immediately following (e) and before `## Background and problem`, so the section ordering the plan specifies is intact.
- **No unintended files touched.** `git diff 93cf3bb...HEAD --name-only` is limited to the 8 files the plan's Scope table names (2 prose files + their mirrors, plus plan/self-review/insight-event bookkeeping); no Go source, no `ralph.toml`, no `docs/recipes/` file appears in the diff, consistent with the plan's Non-goals.

## Known gaps / out of scope for this pass

- AC-3 (issue #156 comment) and AC-5 (`Refs #156` PR body) are intentionally unverified here — they are produced at `/pr` time per the plan's own Implementation outline, not a gap in this diff.
- Behavioral tests (`go test ./internal/config/... -count=1`, the plan's named regression check) were not run — that is `/test`'s job, not `/verify`'s.
- `./scripts/run-verify.sh` (the unscoped, test-inclusive wrapper) was not run for the same reason; `./scripts/run-static-verify.sh` was used instead, per this task's instructions and the repo's verifier/tester split.
- I did not independently re-execute the self-review's `codex exec --sandbox read-only 'echo ok' </dev/null` cache-refresh probe or re-derive the 12:31→12:40:07 mtime measurement; that observation is plan-recorded evidence from earlier in the task, not a claim this diff makes about code behavior, so it was out of scope for this pass's code-vs-prose verification.
