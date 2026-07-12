# Verify report: xreview-base-detection

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-xreview-base-detection.md
- Verifier: verifier subagent (Claude Code, /verify)
- Scope: spec compliance + static analysis; no behavioral test execution (belongs to /test)
- Evidence: `docs/evidence/verify-2026-07-12-121032.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `detect_base_branch` exists in ralph-cli-driver.sh; pipeline gate uses it; `grep -rn 'HEAD@{upstream}'` over scripts/ralph-pipeline.sh and 4 SKILL copies -> 0 hits | PASS | `scripts/ralph-cli-driver.sh:194` defines `detect_base_branch`. `scripts/ralph-pipeline.sh:807` calls `_base="$(detect_base_branch)"`. Grep over pipeline + 4 SKILL files returns 0 hits. Template mirrors (`templates/base/scripts/ralph-pipeline.sh:807`) also use `detect_base_branch`. |
| AC2: new driver tests exist incl. end-to-end gate proof (14a: old detection -> empty diff vs detect_base_branch -> non-empty diff), RALPH_XREVIEW_BASE override (14b), non-main default (14c), `/`-containing default (14c-edge), main/master fallback (14d-i/ii), worktree fixture (14e) | PASS (existence + assertions; execution belongs to /test) | `tests/test-ralph-cli-driver.sh:613–760` — Test 14 (a–e) present. 14a asserts OLD yields feature branch (14a-i) + empty diff (14a-ii) AND `detect_base_branch` yields main (14a-iii) + non-empty diff (14a-iv). 14b: `RALPH_XREVIEW_BASE=develop` override. 14c: `origin/HEAD -> develop`. 14c-edge: `origin/release/1.0` strips only leading `origin/`. 14d-i: no origin/HEAD + `refs/heads/main` -> main. 14d-ii: only `refs/heads/master` -> master. 14e: git worktree resolves via shared common dir. |
| AC2b: ralph-orchestrator.sh exports RALPH_XREVIEW_BASE from `_base_branch` before slice and integration pipeline invocations | PASS | `scripts/ralph-orchestrator.sh:1297` — `export RALPH_XREVIEW_BASE="$_base_branch"` at L1297, which precedes `run_slice` call at L1526 and `run_integration_pipeline` at L1597. Template mirror is byte-identical. |
| AC3: existing xreview suites exist unchanged (`tests/test-xreview-gate-regression.sh`, `tests/test-xreview-prompt-render.sh`) | PASS (unchanged; execution belongs to /test) | `git diff main...HEAD -- tests/test-xreview-gate-regression.sh tests/test-xreview-prompt-render.sh` — empty output (no modifications). Both files present. |
| AC4: tech-debt row RESOLVED; mirrors byte-identical; sync gates pass; `./scripts/run-static-verify.sh < /dev/null` passes | PASS | Tech-debt row at `docs/tech-debt/README.md:44-45` uses `~~...~~` strikethrough + `<!-- RESOLVED 2026-07-12 in fix/xreview-base-detection: ... -->` HTML comment, matching the existing convention (see cli-stub-stdin-hang row directly above). All 8 mirrors byte-identical (cmp: cli-driver, pipeline, orchestrator, AGENTS.md, both SKILL surfaces root vs template, .claude vs .agents). `run-static-verify.sh < /dev/null` exits 0 (check-sync: DRIFTED=0, check-skill-sync: 13 in lock-step, check-pipeline-sync: all OK, gofmt: ok, 0 golangci-lint issues). |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh < /dev/null` (worktree) | EXIT 0 | All gates passed. check-sync: IDENTICAL=171, DRIFTED=0. check-pipeline-sync: all 8 consumer files reference pipeline steps. check-skill-sync: 13 skills in lock-step. gofmt: ok. golangci-lint: 0 issues. |
| `sh -n scripts/ralph-cli-driver.sh` | PASS | No POSIX syntax errors. |
| `sh -n scripts/ralph-pipeline.sh` | PASS | No POSIX syntax errors. |
| `sh -n scripts/ralph-orchestrator.sh` | PASS | No POSIX syntax errors. |
| `shellcheck -S warning scripts/ralph-cli-driver.sh` | PASS (0 warnings) | No shellcheck warnings on the new `detect_base_branch` function or surrounding code. |
| `grep -rn 'HEAD@{upstream}' scripts/ralph-pipeline.sh templates/base/scripts/ralph-pipeline.sh .claude/skills/cross-review/SKILL.md .agents/skills/cross-review/SKILL.md templates/base/.claude/skills/cross-review/SKILL.md templates/base/.agents/skills/cross-review/SKILL.md` | 0 hits | Confirmed: old broken detection removed from all production sites. Remaining occurrences are only inside `tests/test-ralph-cli-driver.sh` (Test 14a-i/ii, intentionally proving old behavior). |
| Mirror parity (cmp) for all 8 synced pairs | PASS | cli-driver, pipeline, orchestrator (scripts + templates/base), both SKILL surfaces (.claude/.agents × root/template), AGENTS.md — all byte-identical. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| SKILL.md ×4 (`.claude/skills/cross-review/SKILL.md` + `.agents` mirror + both `templates/base` copies) — base detection wording at line 51 | In sync | All 4 copies updated to `. scripts/ralph-cli-driver.sh; BASE=$(detect_base_branch)` with the 3-step resolution order matching the helper exactly: (1) `$RALPH_XREVIEW_BASE`, (2) symbolic-ref origin/HEAD with leading `origin/` stripped, (3) `main`/`master` fallback. All 4 copies byte-identical. |
| AGENTS.md function list (root + template mirror) | In sync | `scripts/ralph-cli-driver.sh` description at `AGENTS.md:80` now lists `detect_base_branch` between `pick_reviewer` and `count_triage_findings`. Both copies byte-identical. |
| `docs/tech-debt/README.md` RESOLVED row | In sync | Row marked RESOLVED with HTML comment + strikethrough, following the same format as the `cli-stub-stdin-hang` precedent. Details summarize what was fixed. |
| Helper comment accuracy (LOW finding from self-review) | Accepted as-is | Self-review noted the comment says "mirrors `default_branch()` in `scripts/ralph` **and** `ralph-worktree.sh`" but the two helpers diverge at the final fallback. The wording is slightly imprecise but the behavior (unconditional `master` fallback) is correct and intentional. Self-review accepted this as optional. No change needed. |
| `model-routing.md` | Not modified — no drift | Confirmed unchanged in the diff. |

## Observational checks

- **Export-by-value robustness (AC2b)**: L1297 `export RALPH_XREVIEW_BASE="$_base_branch"` snapshots the value before `create_worktree` (L1450) clobbers the global `_base_branch` variable. This is correct in a no-`local` POSIX-sh script. The export value is frozen at the launch branch regardless of later variable mutation.
- **Codex advisory findings — all 4 adopted**:
  - Finding 1 (HIGH): `RALPH_XREVIEW_BASE` exported by orchestrator takes priority — confirmed at `ralph-cli-driver.sh:196` (resolution step 1).
  - Finding 2 (MEDIUM): reuses existing main/master fallback semantics (`git show-ref --verify --quiet refs/heads/main`) + fail-open-to-review behavior documented in helper comment at L191-192.
  - Finding 3 (MEDIUM): tests prove the gate end-to-end (empty-diff-before vs non-empty-after) plus worktree (14e) and Loop-base fixtures (14b).
  - Finding 4 (LOW): Non-goals section reads "No Codex CLI/runtime changes (the .agents SKILL mirrors are text sync only)."
- **`detect_base_branch` is pure**: no top-level execution in `ralph-cli-driver.sh` — safe to source. All `2>/dev/null` guards are on read-only git ref queries with intended fallbacks.
- **prefix-only strip** `${_dbb_remote_head#origin/}` is correct for `release/1.0`: removes only the leading `origin/`, leaving the rest intact. Confirmed by 14c-edge.

## Coverage gaps

- **Test execution (AC2, AC3)**: fixture tests at Test 14 and the unchanged xreview suites are defined and structurally correct. Whether they actually pass at runtime belongs to /test.
- **Runtime resolution of `detect_base_branch` in the pipeline** is verified at the source level only; live execution with an actual git remote belongs to /test.

## Verdict

PASS — all acceptance criteria met at the static/structural level.

- **Verified**: AC1 (detect_base_branch defined + HEAD@{upstream} removed from all 6 production files), AC2 (test assertions structure + gate-proof shape), AC2b (export position at L1297, before L1526 run_slice and L1597 run_integration_pipeline), AC3 (xreview suites unchanged), AC4 (tech-debt RESOLVED row + mirrors byte-identical + run-static-verify exits 0). All 4 Codex advisory findings adopted.
- **Likely but unverified**: test execution (AC2 pass/fail at runtime) and AC3 (xreview suites still green) — delegated to /test.
- **Not verified**: live cross-review base detection against a real remote (belongs to /test or manual smoke test).
