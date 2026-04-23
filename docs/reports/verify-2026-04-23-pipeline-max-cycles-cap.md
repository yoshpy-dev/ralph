# Verify report: pipeline-max-cycles-cap

- Date: 2026-04-23
- Plan: `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md`
- Verifier: verifier subagent (/verify — spec compliance + static analysis; no tests)
- Scope: 6 commits on `feat/pipeline-max-cycles-cap` vs `main` (`5a42478`, `009428f`, `7568755`, `7b2e2e2`, `878f5d2`, `d287534`); `git diff main...HEAD --stat` = 18 files, +321/-43.
- Evidence: `docs/evidence/verify-2026-04-23-pipeline-max-cycles-cap.log`

## Spec compliance

Working through each bullet in the plan's **Acceptance criteria** section verbatim.

| # | Acceptance criterion | Status | Evidence |
| --- | --- | --- | --- |
| 1 | `RALPH_MAX_OUTER_CYCLES` default changed to `2` in both `scripts/ralph-config.sh` and `templates/base/scripts/ralph-config.sh`, and they match. | VERIFIED | `scripts/ralph-config.sh:23` `RALPH_MAX_OUTER_CYCLES="${RALPH_MAX_OUTER_CYCLES:-2}"`; identical at `templates/base/scripts/ralph-config.sh:23`. `cmp -s` confirms byte identity across the two files. |
| 2 | `RALPH_STANDARD_MAX_PIPELINE_CYCLES` added to `ralph-config.sh` with default `2` and covered by `validate_all_numeric`. | VERIFIED | `scripts/ralph-config.sh:27` sets default `2`; `:58` calls `validate_numeric "RALPH_STANDARD_MAX_PIPELINE_CYCLES" "$RALPH_STANDARD_MAX_PIPELINE_CYCLES"` inside `validate_all_numeric`. Mirrored at `templates/base/scripts/ralph-config.sh:27,58`. |
| 3 | `tests/test-ralph-config.sh` has new default / override / validation cases for the new variable and passes. | VERIFIED (static inspection) | `tests/test-ralph-config.sh:80-81` (default=2), `:106-107` (override=5), `:166-167` (rejects non-numeric), `:169-170` (rejects zero). Behavioral pass/fail is owned by `/test`; self-review report records 27/27 pass. |
| 4 | `codex-review/SKILL.md` Case A and Case B each contain a cycle-cap check. | VERIFIED | `.claude/skills/codex-review/SKILL.md:91-115`: Step 6 defines `CAP_REACHED = (cycle >= RALPH_STANDARD_MAX_PIPELINE_CYCLES)`; Case A at `:95-107` branches on `NOT CAP_REACHED` vs `CAP_REACHED`; Case B at `:109-115` branches the same way. Mirrored in template. |
| 5 | On cap-reached, the "修正する" option is dropped and "上限解除 / PR 作成 / 中止" is offered. | VERIFIED | `.claude/skills/codex-review/SKILL.md:102-107` — cap-reached Case A lists exactly three options: (1) 上限を一時的に引き上げて再実行, (2) 指摘は記録し PR を作成する, (3) 中止. The "修正する" option is absent. Case B cap-reached path at `:115` skips the re-run option and proceeds to /pr. |
| 6 | `work/SKILL.md` documents plan-path persistence to `active-plan.json`, cycle counter init (cycle=1), and `/pr` cleanup of both files. | VERIFIED | `.claude/skills/work/SKILL.md:15-23` (Step 0.5): mkdir, write `active-plan.json`, init `cycle-count.json` at `cycle: 1`. `:43` documents `/pr` deletes both state files on success. Plan-path persistence is `{plan_path, created_at}` per `:21`. |
| 7 | `codex-review/SKILL.md` and `pr/SKILL.md` read `active-plan.json` instead of rescanning `docs/plans/active/`. | VERIFIED | `.claude/skills/codex-review/SKILL.md:23,28,45` — Step 0 reads `active-plan.json`, the "Hard prohibition" clause at `:28` forbids rescan, and Step 3 at `:45` reads the plan via the persisted path (fixed in commit d287534 after the self-review MEDIUM finding). `.claude/skills/pr/SKILL.md:24-25,32` — Step 0 reads `active-plan.json`, Step 5 uses the resolved path for `archive-plan.sh`. |
| 8 | `/work` SKILL.md describes the multi-plan AskUserQuestion behavior. | VERIFIED | `.claude/skills/work/SKILL.md:16-19` — Step 0.5.a explicitly states: "If multiple candidates exist, ask via AskUserQuestion which plan this `/work` run targets, and use the selected path." The selected path is then persisted per :21. |
| 9 | `.claude/rules/post-implementation-pipeline.md` "Re-run after Codex ACTION_REQUIRED fix" section documents the cap rule (standard=2, Ralph Loop=2). | VERIFIED | `.claude/rules/post-implementation-pipeline.md:36-43` — a new "Pipeline cycle cap (default 2 total runs)" subsection placed under "Re-run after Codex ACTION_REQUIRED fix". Both variables, their persistence paths, and cap-reached behavior are documented. |
| 10 | `README.md` / `AGENTS.md` / `CLAUDE.md` / `docs/quality/definition-of-done.md` / `docs/recipes/ralph-loop.md` synced. | PARTIALLY VERIFIED — see Documentation drift | `docs/quality/definition-of-done.md:28` and `docs/recipes/ralph-loop.md:144,148` are updated. `CLAUDE.md`, `AGENTS.md`, `README.md` are NOT touched by this diff. Per the user's verify prompt this is acceptable because they do not mention cap defaults and therefore cannot contradict the new cap — confirmed via grep (no `cycle` / `cap` / `3` / `2` wording that refers to pipeline run count in those files). Flagged as non-blocking. |
| 11 | `ralph-pipeline.sh --help` default display updated 3 → 2. | VERIFIED | `scripts/ralph-pipeline.sh:37` `--max-outer-cycles N     Max Outer Loop regressions before escalation (default: 2)`. Mirrored in template (`cmp -s` identical). |
| 12 | `./scripts/run-verify.sh` exits 0. | VERIFIED | Ran during this verification; stdout ends `==> All verifiers passed.` Full output appended to `docs/evidence/verify-2026-04-23-pipeline-max-cycles-cap.log`. Covers shellcheck (skipped on this macOS — no binary), `sh -n` on all hooks, `jq -e .` on both `settings.json` copies, `scripts/check-sync.sh`, `tests/test-check-mojibake.sh` (11/11), `gofmt`, `golangci-lint`, `go test ./...` (all `ok`). |
| 13 | `./scripts/check-sync.sh` (or `check-pipeline-sync.sh`) confirms source/template sync. | VERIFIED | `check-sync.sh` stdout: `IDENTICAL: 107, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 9, KNOWN_DIFF: 3 — PASS: all files in sync.` |

**Summary:** 12 of 13 acceptance criteria **VERIFIED**, 1 **PARTIALLY VERIFIED** (AC #10 — the top-level docs not touched, but they also do not contradict the new cap; consistent with the verify prompt's guidance).

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-verify.sh` | exit 0 | All verifiers passed. Timestamp 2026-04-23T09:10:17Z captured inside the script's own evidence log. |
| `./scripts/check-sync.sh` | PASS | `IDENTICAL: 107, DRIFTED: 0`. All 8 mirrored pairs touched by this diff are byte-identical (explicit `cmp -s` confirmation per pair appended to the evidence log). |
| `bash tests/test-ralph-config.sh` | not run | Owned by `/test`, not `/verify`. Self-review report records 27/27 pass; the three new assertions are visible at `tests/test-ralph-config.sh:80-81,106-107,166-170`. |
| `sh -n` on all hooks (source + templates) | OK | 18 files each passed POSIX syntax check. |
| `jq -e .` on `.claude/settings.json` and `templates/base/.claude/settings.json` | OK | Both are valid JSON. |
| `gofmt` / `golangci-lint` / `go test ./...` | OK / 0 issues / all `ok` | Included in `run-verify.sh` output. `go test` was cache-hit — all tests are currently marked as cached `ok`. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/post-implementation-pipeline.md` | Yes | New "Pipeline cycle cap" subsection at lines 36-43 documents both variables, persistence files, cap-reached behavior, and rationale. |
| `docs/quality/definition-of-done.md:28` | Yes | Documents the 2-run cap with both variable names and a pointer to the rules file. |
| `docs/recipes/ralph-loop.md:144,148` | Yes | Env-var table lists `RALPH_MAX_OUTER_CYCLES=2` and the new `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`. |
| `.claude/skills/work/SKILL.md` | Yes | Step 0.5 + Step 9e/9f describe the full lifecycle. |
| `.claude/skills/codex-review/SKILL.md` | Yes | Step 0 reads state, Step 3 reads plan via pinned path (fixed in d287534), Step 6 branches on cap, Step 7 increments counter. |
| `.claude/skills/pr/SKILL.md` | Yes | Step 0 reads pinned path, Step 5 archives via pinned path, Step 6 clears state on success. |
| `CLAUDE.md`, `AGENTS.md`, `README.md` | No change — not contradictory | Grep confirms none of these files state a pipeline-run-count cap. The verify prompt explicitly allows "they may not mention it at all — that's fine if so." |
| `templates/base/` mirrors | Yes | `cmp -s` confirms all 8 mirrored pairs touched by this diff are byte-identical. `check-sync.sh` reports DRIFTED=0 overall. |
| `docs/tech-debt/README.md` | Drift (non-blocking) | The self-review LOW finding flagged that the deferred "scripting the JSON counter" item (Codex advisory #1) lives only in the plan's Open Questions, not in `docs/tech-debt/`. Not part of the AC list, but mentioned here for full coverage. |
| Plan progress checklist (`docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md`) | Drift (expected) | Acceptance-criteria checkboxes at `:68-80` are all still `[ ]` unchecked. Progress checklist at `:156-168` shows Review/Verify/Test/PR artifacts as `[ ]`. This is the normal "plan AC boxes lag implementation" pattern — flagged as doc drift but not a verify failure (consistent with memory: `feedback_plan_ac_checklist_drift`). |

## Observational checks

- **Grep audit — pipeline default `2` vs lingering `3`** (the user's requested cross-reference check): Ran `grep -rn 'RALPH_MAX_OUTER_CYCLES' scripts/ templates/ docs/ .claude/`. Every hit that specifies a default value (source, rules, DoD, recipe, template mirrors) reads `2`. The only `3`-related hits in the repo are `README.md:197` ("3 consecutive no-change iterations" — unrelated to the pipeline run cap; it describes `stuck` detection) and `docs/recipes/ralph-loop.md:124` (same concept). Neither contradicts the new cap. Evidence appended to log.
- **Grep audit — `RALPH_STANDARD_MAX_PIPELINE_CYCLES`**: Every hit uses the new variable consistently; no residual `2 cycle(s)` or `3 cycle(s)` wording in prose docs.
- **State-file naming consistency**: All 14 references to `.harness/state/standard-pipeline/{active-plan.json,cycle-count.json}` across 8 canonical files (+ their template mirrors) use the same path. No drift between `work`, `codex-review`, `pr`, rules, and DoD.
- **Cap-check symmetry**: `codex-review/SKILL.md` Step 6 flow tree (Case A/B × CAP_REACHED/not) enumerates 4 branches explicitly; the DISMISSED-only Case C skips the cap entirely and proceeds to /pr. No cap-unaware branch exists.
- **Step 3 contradiction from self-review is closed**: Commit `d287534` aligned Step 3 with Step 0's "Hard prohibition" — `/codex-review` no longer instructs a rescan anywhere in the file, and the template mirror matches.
- **archive-plan.sh compatibility**: `/pr` Step 5 passes the absolute plan path (not a slug) to `scripts/archive-plan.sh`. Self-review confirmed the script accepts absolute paths via its `[ -f "$arg" ] || [ -d "$arg" ]` branch, so the `/pr` change is backward-compatible.

## Coverage gaps

These remain **unverified** by `/verify` and belong to `/test` or are future work:

1. **Behavioral execution of `tests/test-ralph-config.sh`.** Static inspection confirms the three new assertions exist and are shaped correctly; actual pass/fail is `/test`'s responsibility. Self-review report cited 27/27 but that was a different run.
2. **Dry-run of `ralph-pipeline.sh` with `MAX_OUTER_CYCLES=2`** (plan's integration-test item). Belongs to `/test`.
3. **End-to-end cap-reached flow** in `/codex-review` (AskUserQuestion menu rendering, user picking each option). Belongs to `/test` and/or manual walkthrough; no harness for this exists yet.
4. **Concurrency/race conditions around `cycle-count.json`** if a user runs two `/work` sessions against different plans. Plan's Risks section acknowledges the JSON is written from prompt text (not a deterministic helper). Tech-debt, not a `/verify` blocker.
5. **`.harness/state/standard-pipeline/` gitignore behavior.** The plan claims the existing `.harness/state/` gitignore covers it. Not re-verified here because no `/work` run has created the directory yet. Could be confirmed by `git check-ignore -v .harness/state/standard-pipeline/active-plan.json` after a touch, but that would require creating real state.

### Smallest useful additional check

If you want to raise confidence with one more step, the smallest high-value check would be:

```sh
git check-ignore -v .harness/state/standard-pipeline/active-plan.json
```

after touching the file locally. This confirms AC-adjacent assumption that the new subdirectory is already covered by `.harness/state/` in `.gitignore`, without actually committing state.

## Verdict

**PASS (with one non-blocking documentation observation).**

- **Verified:** AC #1–#9, #11, #12, #13 — all 11 non-doc-sync criteria are satisfied with direct file/line evidence and passing deterministic checks (`run-verify.sh` exit 0, `check-sync.sh` DRIFTED=0, mirror `cmp -s` all identical).
- **Partially verified:** AC #10 (top-level docs sync). The three outer files (`CLAUDE.md`, `AGENTS.md`, `README.md`) are not touched but also do not state any pipeline-run cap, so no contradiction exists. The verify prompt explicitly accepts this outcome.
- **Not verified (deferred to /test):** behavioral test execution, dry-run of `ralph-pipeline.sh`, end-to-end cap-reached interaction flow. These are not `/verify` responsibilities.
- **Documentation drift flagged, not blocking:** (a) plan AC checkboxes are still unchecked (normal lag pattern), (b) `docs/tech-debt/README.md` does not yet record the "script the counter" debt item (self-review LOW — outside this plan's AC but worth a follow-up commit).

No fix-and-revalidate cycle required from `/verify`. Proceed to `/test`.

---

## Cycle 2 verify (2026-04-23, commit `e27102a`)

- Scope: Re-verify the two Codex ACTION_REQUIRED fixes only — `.claude/skills/work/SKILL.md` Step 0.5.d (preserve counter on plan_path match) and `.claude/skills/codex-review/SKILL.md` Case B cap-reached (AskUserQuestion with raise-cap/PR/abort). Both canonical files plus their `templates/base/` mirrors (4 paths total).
- Evidence appended to `docs/evidence/verify-2026-04-23-pipeline-max-cycles-cap.log`.

### Results

| Check | Result | Evidence |
| --- | --- | --- |
| `./scripts/run-verify.sh` | exit 0 — "All verifiers passed." | Fresh evidence log `docs/evidence/verify-2026-04-23-094453.log`. |
| `./scripts/check-sync.sh` | DRIFTED=0 (IDENTICAL=107, TEMPLATE_ONLY=9, KNOWN_DIFF=3) | PASS. |
| Mirror byte-identity | `cmp -s` OK for both `work/SKILL.md` and `codex-review/SKILL.md` pairs | See log. |
| AC #4 (Case A/B cycle-cap check) — spot | Still VERIFIED; Case B cap-reached now fully specified at `.claude/skills/codex-review/SKILL.md:115-120` | — |
| AC #5 (cap-reached drops "修正", offers 3 options) — spot | Still VERIFIED and **strengthened**: Case B cap-reached now has the same 3-option AskUserQuestion (raise cap / PR / 中止) as Case A, resolving the cycle-1 Case B gap | `:115-120`. |
| AC #6 (work Step 0.5 lifecycle) — spot | Still VERIFIED and **strengthened**: Step 0.5.d now branches on (i) missing, (ii) plan_path match → preserve, (iii) plan_path mismatch → AskUserQuestion | `.claude/skills/work/SKILL.md:22-26`. |
| Contradiction check vs `.claude/rules/post-implementation-pipeline.md:40` | None. The rule says cap-reached drops "fix" and offers (1) raise cap, (2) PR + known gaps, (3) abort. Case B cap-reached at `:115-120` matches exactly | — |
| Contradiction check vs plan line 40 | None. Plan specifies "上限解除 / PR 作成 / 中止"; both Case A and Case B cap-reached branches implement these three | — |

### Verdict

**PASS.** Cycle 2 fixes are correctly implemented, mirrored, and internally consistent with both `post-implementation-pipeline.md:40` and plan line 40. No new findings. Cycle-1 verdict stands.

### Notes

- Uncommitted edit in working tree: `docs/reports/self-review-2026-04-23-pipeline-max-cycles-cap.md` (cycle-2 re-review section appended by reviewer). Not a verify blocker — docs report only.
- Cycle-1 AC-checkbox drift (plan `:68-80` still `[ ]`) unchanged; still flagged as non-blocking doc drift per prior pattern.

---

## Cycle 3 verify (2026-04-23, commit `12b87ee`)

- Scope: Codex ACTION_REQUIRED fix — `/work` step reorder (0 resolve plan → 0.5 branch → 0.7 pin+counter) and removal of counter increment on `/codex-review` cap-override path.

### Results

| Check | Result | Evidence |
| --- | --- | --- |
| `./scripts/run-verify.sh` | exit 0 — "All verifiers passed." | `docs/evidence/verify-2026-04-23-100326.log`. |
| `./scripts/check-sync.sh` | DRIFTED=0 (IDENTICAL=107). | PASS. |
| Mirror byte-identity (work + codex-review) | `shasum` identical for both pairs | `0396287…` work, `007dc75…` codex-review. |
| `/work` step reorder | VERIFIED | `.claude/skills/work/SKILL.md:9` Step 0 resolve → `:14` Step 0.5 branch → `:20-27` Step 0.7 pin+counter. `:25` preserves counter on plan_path match. |
| Cap-override no increment | VERIFIED | `.claude/skills/codex-review/SKILL.md:127` explicitly: "Do **NOT** increment `cycle-count.json`". Case A non-cap path at `:126` still increments. |
| Contradiction vs `post-implementation-pipeline.md:40` | None | Rule says "raise the cap"; SKILL.md:127 raises via `export` and leaves counter unchanged so the raised cap grants one extra pass. Honored. |

### Verdict

**PASS.** Cycle 3 fix correctly reorders `/work` Steps 0/0.5/0.7 and preserves counter semantics on cap-override. Cycles 1 & 2 verdicts stand.
