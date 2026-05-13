# fix-xreview-placeholder-substitution

- Status: In review
- Owner: Claude Code
- Date: 2026-05-13
- Related request: Cross-review gate silently bypassed under codex driver due to unexpanded `${BASE_BRANCH}` / `${REPORTS_DIR}` placeholders
- Related issue: #50
- Branch: fix/50/xreview-placeholder-substitution

## Objective

Stop the cross-review gate from silently passing under the `RALPH_LOOP_DRIVER=codex` + claude-reviewer path. The adversarial-claude prompt must be invoked with `BASE_BRANCH` and `REPORTS_DIR` expanded to real values so the reviewer writes its triage report to a path the downstream parser can find. After the fix, an `ACTION_REQUIRED` finding from claude reviewer must regress the Inner Loop instead of being silently dropped.

## Scope

- `scripts/ralph-pipeline.sh` cross-review phase (around lines 787–818): pre-render the adversarial-claude prompt with `BASE_BRANCH` / `REPORTS_DIR` expanded before piping to `claude -p`.
- `.claude/skills/cross-review/prompts/adversarial-claude.md`: keep `${BASE_BRANCH}` / `${REPORTS_DIR}` placeholders, but document (in a comment near the top) that the surrounding shell must expand them before invocation.
- Regression coverage: add a test under `tests/` that verifies pipeline-side prompt rendering substitutes both placeholders so no literal `${BASE_BRANCH}` / `${REPORTS_DIR}` strings remain.
- Doc touch-ups: short note in `.claude/skills/cross-review/SKILL.md` describing the substitution contract so future edits to the prompt do not reintroduce un-rendered placeholders.

## Non-goals

- Redesigning the cross-review triage report format.
- Changing the `pick_reviewer` / `count_triage_findings` behavior in `scripts/ralph-cli-driver.sh`.
- Touching the Codex-side reviewer path (`codex exec review --base "$_base"`) — only the claude-reviewer-under-codex-driver path is broken.
- Adding telemetry beyond the existing `report_event "cross-review"` line.
- Backporting to 3.5.0 / 3.5.1; the fix lands on `main` and ships in the next tag.

## Assumptions

- `sed` and `awk` are available in every environment the pipeline runs in (POSIX baseline; already used elsewhere in `scripts/`). `envsubst` is NOT assumed (not present in minimal Linux containers).
- Git refs CAN contain `#`, `&`, and `/`; `REPORTS_DIR` is configurable and could in theory contain regex / replacement metacharacters. The renderer must therefore avoid `sed`-pattern coupling and treat the values as literal strings.
- `_base` may be unset / empty if `git rev-parse` failed upstream; the renderer must error out clearly rather than silently substituting an empty string.
- A failure mode where the reviewer LLM hallucinates a different output path is out of scope for this fix; we only ensure that the *prompt* it receives names a real, expanded path.

## Affected areas

- `scripts/ralph-pipeline.sh` — cross-review phase (read-write).
- `.claude/skills/cross-review/prompts/adversarial-claude.md` — comment / contract note (read-write).
- `.claude/skills/cross-review/SKILL.md` — short note on the rendering contract (read-write).
- `tests/test-xreview-prompt-render.sh` *(new)* — regression test.
- Mirrored skill body under `.agents/skills/cross-review/` if `scripts/check-skill-sync.sh` flags drift.

## Design decisions

**Critical fork — how to expand placeholders before invoking `claude -p`?**

Chosen option: **Option A — pre-render the prompt into a per-cycle temp file under `${PIPELINE_DIR}/` using a metacharacter-safe renderer, then redirect that file into `claude -p`.**

Rationale:
- Portable: stays POSIX (`awk` or `bash` parameter expansion); no `envsubst` / gettext dependency, which is missing on slim Linux containers and some CI images.
- Keeps the prompt body as a separate, version-controlled file (Option B in the issue couples the substitution logic to the prompt content).
- Localized blast radius: change lives in ~15–25 shell lines and one new test file; no orchestrator-wide refactor.
- Cleanup is free: `${PIPELINE_DIR}/` is already per-cycle ephemeral state.

**Renderer implementation correction (per Codex finding #1):** The naive `sed -e "s#\${BASE_BRANCH}#${_base}#g"` form **does not work** under `set -u` (the unset `BASE_BRANCH` triggers an early error, as Codex verified empirically), and `sed` replacement metacharacters (`&`, `\`, the delimiter) collide with valid git refs (`feature#1`, `feature&1`, `release/3.5` all pass `git check-ref-format`). The renderer will therefore:

- Use `awk` with `ENVIRON["VAR"]` lookup (treats replacement value as a literal string — no metacharacter interpretation), OR a small Bash function using `${var//pattern/replacement}` parameter expansion which is also literal.
- Quote the placeholder pattern in single quotes so the shell never expands it.
- Fail loudly if `_base` is empty or any required value is unset.
- Validate the rendered output (see acceptance criterion below) so silent-bypass cannot recur.

Trade-offs considered:
- `envsubst` is cleaner syntactically but introduces a runtime dependency we have explicitly avoided elsewhere.
- Hardcoding `docs/reports/` in the prompt (Option B in the issue) would let us drop the substitution entirely, but couples the prompt to one config value and would break if `REPORTS_DIR` ever becomes user-configurable.
- Heredoc inside the script would duplicate the prompt text in two places and defeat the purpose of the dedicated prompt file.

**Operator note (no fork):** The auto-invoked flow is `/work` (single-slice bug fix in one script + one prompt file + one new test — well below the Ralph Loop threshold).

## Acceptance criteria

- [ ] When `RALPH_LOOP_DRIVER=codex` and the claude reviewer is invoked, the prompt that reaches `claude -p` contains the literal `_base` value (e.g. `main`) and the literal `REPORTS_DIR` value (e.g. `docs/reports`), with no `${BASE_BRANCH}` / `${REPORTS_DIR}` strings remaining.
- [ ] The rendered prompt file is written under `${PIPELINE_DIR}/` (per-cycle, ephemeral) and not committed.
- [ ] If the prompt file is missing, the existing warning path (`Warning: adversarial-claude prompt missing at ...`) is preserved.
- [ ] **Unresolved-placeholder guard (per Codex finding #3):** After rendering, the renderer scans the output for any remaining `${[A-Z_][A-Z0-9_]*}` token. If a token outside the allowlist (`BASE_BRANCH`, `REPORTS_DIR`) is found, the renderer logs the offending token and treats the cross-review gate as failed (not bypassed). The allowlist lives alongside the renderer so adding a new placeholder is a deliberate edit, not a silent introduction.
- [ ] If rendering fails (non-zero exit, unresolved placeholder, or empty `_base`), the pipeline logs the failure and does NOT silently pass the cross-review gate. ACTION_REQUIRED defaults to 0 only when the diff against `_base` is empty; otherwise the gate must surface the rendering failure rather than auto-passing.
- [ ] **End-to-end gate-regression test (mandatory, per Codex finding #2):** A new test fakes the `claude` CLI (reusing `tests/fixtures/fake-claude` or an equivalent), captures the rendered prompt stdin, writes a fresh triage report under `REPORTS_DIR` with `After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0`, drives the cross-review phase, and asserts the function returns non-zero (Inner Loop regression). This proves the bug is fixed end-to-end, not only at the substitution layer.
- [ ] **Renderer unit test:** asserts no `${BASE_BRANCH}` / `${REPORTS_DIR}` literals remain and both substituted values appear at least once; parameterized over `main`, `release/3.5`, `feature#1`, `feature&1` to cover metacharacter edge cases.
- [ ] `./scripts/run-verify.sh` (or its delegate) and `./scripts/check-skill-sync.sh` stay green.
- [ ] The adversarial-claude prompt file documents the rendering contract in a top-of-file comment so future edits cannot reintroduce un-rendered placeholders without noticing.

## Implementation outline

1. Add a metacharacter-safe renderer helper in `scripts/ralph-pipeline.sh` (cross-review phase). Sketch using `awk`:
   ```sh
   _rendered_prompt="${PIPELINE_DIR}/outer-${_cycle}-adversarial-claude.md"
   if [ -z "${_base:-}" ]; then
     log_error "cross-review: empty _base; cannot render adversarial prompt"
     # treat as gate failure, not silent pass
   fi
   BASE_BRANCH="$_base" REPORTS_DIR="$REPORTS_DIR" \
     awk '{
       line = $0
       gsub(/\$\{BASE_BRANCH\}/, ENVIRON["BASE_BRANCH"], line)
       gsub(/\$\{REPORTS_DIR\}/,  ENVIRON["REPORTS_DIR"],  line)
       print line
     }' "$_adv_prompt" > "$_rendered_prompt"
   ```
   `awk gsub` replacements are literal — no `sed`-style `&` / `\` interpretation. Single-quoting the awk program means the shell never expands `${...}` in the pattern.
2. Validate the rendered output (allowlist enforcement):
   ```sh
   if leftover="$(grep -oE '\$\{[A-Z_][A-Z0-9_]*\}' "$_rendered_prompt" | sort -u)"; then
     if [ -n "$leftover" ]; then
       log_error "cross-review: unresolved placeholders in rendered prompt: $leftover"
       # treat as gate failure
     fi
   fi
   ```
3. Replace `< "$_adv_prompt"` in the `claude -p` invocation with `< "$_rendered_prompt"`. Drop or comment the now-redundant env-var prefix (left in only if useful for downstream debugging logs).
4. Add a brief comment at the top of `.claude/skills/cross-review/prompts/adversarial-claude.md`: "Placeholders `${BASE_BRANCH}` and `${REPORTS_DIR}` are pre-rendered by `scripts/ralph-pipeline.sh` before invocation. Adding a new `${...}` placeholder requires updating the renderer allowlist."
5. Add a short note to `.claude/skills/cross-review/SKILL.md` (in the "Reviewer inversion inside Ralph Loop" section) about the rendering + allowlist contract.
6. Write `tests/test-xreview-prompt-render.sh` (renderer unit test):
   - Run the renderer block in isolation, parameterized over `_base ∈ {main, release/3.5, feature#1, feature&1}` and `REPORTS_DIR ∈ {docs/reports, docs/reports#1}`.
   - Assert no `${BASE_BRANCH}` / `${REPORTS_DIR}` literals remain; both substituted values appear unchanged.
   - Negative case: inject a fake placeholder `${UNKNOWN}` into a tmp prompt and confirm the allowlist guard surfaces it.
7. Write `tests/test-xreview-gate-regression.sh` (mandatory end-to-end gate test, per Codex finding #2):
   - Stand up a sandbox with a fake `claude` binary (reuse `tests/fixtures/fake-claude` patterns) that, when invoked, writes a freshly-dated triage report under `REPORTS_DIR` containing `- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0` and an `## ACTION_REQUIRED` row.
   - Invoke the cross-review phase function (extracted or sourced from `ralph-pipeline.sh`) with `RALPH_LOOP_DRIVER=codex`.
   - Assert: the function returns non-zero (signal to regress to Inner Loop); the rendered prompt file exists under `${PIPELINE_DIR}/`; the triage report was discovered by the `find` parser.
8. Mirror the skill body changes to `.agents/skills/cross-review/` if the drift gate complains; otherwise leave it alone.
9. Run `./scripts/check-skill-sync.sh`, the new tests, the existing test suite, and `./scripts/run-verify.sh`.

## Verify plan

- Static analysis checks: `shellcheck scripts/ralph-pipeline.sh` (matches existing repo discipline), `./scripts/check-skill-sync.sh`.
- Spec compliance criteria to confirm:
  - The cross-review phase still records `driver` / `reviewer` in `ckpt_update` and `report_event` JSONL (no telemetry regression).
  - `ACTION_REQUIRED > 0` from the rendered-prompt path triggers `return 1` (Inner Loop regression) exactly as before.
  - The `--fix-all` WORTH_CONSIDERING regression branch is untouched.
  - Cycle cap behavior (`RALPH_MAX_OUTER_CYCLES`) is untouched.
- Documentation drift to check:
  - `.claude/skills/cross-review/SKILL.md` mentions the rendering step.
  - `AGENTS.md` cross-review section (if present) still matches reality — likely no change needed because it speaks at the contract level, not the implementation level.
  - The mirrored Codex-side skill body matches (`scripts/check-skill-sync.sh`).
- Evidence to capture: shellcheck output, skill-sync exit code, sample rendered prompt path under `docs/reports/` for the verify artifact.

## Test plan

- Unit tests:
  - `tests/test-xreview-prompt-render.sh` — substitution behavior parameterized over `_base ∈ {main, release/3.5, feature#1, feature&1}` and `REPORTS_DIR ∈ {docs/reports, docs/reports#1}`. Also covers the unresolved-placeholder allowlist guard (negative case with `${UNKNOWN}` injected).
- Integration tests (mandatory, per Codex finding #2):
  - `tests/test-xreview-gate-regression.sh` — fake `claude` binary writes a triage report with `ACTION_REQUIRED=1`; the cross-review phase function MUST return non-zero (regress to Inner Loop) and MUST NOT silently proceed to PR.
  - Manual smoke under `DRY_RUN=1` to confirm the rendered prompt file is created at the expected path without invoking the live reviewer.
- Regression tests:
  - Re-run existing `tests/test-ralph-cli-driver.sh` to confirm reviewer inversion + `count_triage_findings` still pass.
  - Re-run `tests/test-check-skill-sync.sh`.
- Edge cases:
  - Prompt file missing → existing warning path preserved.
  - Renderer exits non-zero (simulate by making `_rendered_prompt` path unwritable) → pipeline logs and does NOT silently pass.
  - `_base` empty / unset → renderer errors out before `claude -p` is invoked.
  - `_base` contains `#`, `&`, or `/` → renderer treats them literally (awk `gsub` replacement is metacharacter-free).
  - Unsupported placeholder `${UNKNOWN}` smuggled into the prompt → allowlist guard surfaces it; gate fails closed.
- Evidence to capture: shell test outputs, sample rendered prompt under per-cycle dir, the triage-report contents captured by the gate-regression test.

## Risks and mitigations

- **Risk:** Sed-style metacharacter collision (`#`, `&`, `\`) in valid git refs or future `REPORTS_DIR` values.
  - *Mitigation:* Use `awk gsub` (or Bash `${var//pat/repl}`), which treats replacement values as literal strings — no metacharacter interpretation. Renderer unit test parameterizes over `feature#1`, `feature&1`, `release/3.5`.
- **Risk:** Substitution-only fix passes its own test while the silent-bypass gate remains broken (e.g., later regression).
  - *Mitigation:* Mandatory end-to-end gate-regression test (`tests/test-xreview-gate-regression.sh`) drives the cross-review phase with a fake `claude` and asserts the function returns non-zero when ACTION_REQUIRED findings exist. This guards the actual contract, not the substitution mechanic.
- **Risk:** Future contributor adds a new `${...}` placeholder to the prompt and forgets to update the renderer.
  - *Mitigation:* Allowlist-based unresolved-placeholder guard fails the gate if any `${[A-Z_][A-Z0-9_]*}` token outside `{BASE_BRANCH, REPORTS_DIR}` remains in the rendered output. Documented in the prompt's top-of-file comment.
- **Risk:** Reviewer LLM ignores the expanded path and writes elsewhere.
  - *Mitigation:* Out of scope. The existing `find "$REPORTS_DIR" -name 'cross-review-triage-*' -newer "${PIPELINE_DIR}/checkpoint.json"` parser still returns empty in that case, but the renderer no longer hides a real path from the reviewer, so the most common cause is removed. Tracked as a follow-up if it manifests in practice.
- **Risk:** Skill body drift causes CI to fail.
  - *Mitigation:* Run `./scripts/check-skill-sync.sh` locally and mirror any changes to `.agents/skills/cross-review/`.
- **Risk:** Empty / unset `_base` (upstream `git rev-parse` failed) silently renders an empty branch name into the prompt.
  - *Mitigation:* Renderer errors out when `_base` is empty before `claude -p` is invoked, surfacing the upstream failure instead of masking it.

## Rollout or rollback notes

- Rollout: standard `/work` → post-implementation pipeline → PR → CI verify → human merge into `main`. Ships in the next ralph tag.
- Rollback: revert the single commit on `main` if regressions appear. The change is additive (new test + ~10 lines in the cross-review phase + comment lines) and has no migration step.
- No state-file or schema change; existing `.harness/state/pipeline/` artifacts remain compatible.

## Open questions

- Should the rendered prompt file be retained on disk after the cycle for auditability, or unlinked at end-of-cycle? Default: keep under `${PIPELINE_DIR}/` (already ephemeral by convention) and let the existing pipeline cleanup handle it. Revisit if it bloats state dirs.
- Worth adding a `RALPH_DEBUG_KEEP_RENDERED_PROMPTS` flag later? Out of scope for this PR; mention as a follow-up note in the verify report if interesting.

## Progress checklist

- [x] Plan reviewed (Codex advisory absorbed; 3 findings → plan updates)
- [x] Branch created (`fix/50/xreview-placeholder-substitution`)
- [x] Implementation started
- [x] Renderer + allowlist guard in `scripts/ralph-pipeline.sh` (commit `0304686`)
- [x] Renderer unit test `tests/test-xreview-prompt-render.sh` (54/54, commit `4f15681`)
- [x] End-to-end gate-regression test `tests/test-xreview-gate-regression.sh` (16/16, commit `d2dd875`)
- [x] templates/base/ mirrors restored (commit `f3363b6`)
- [x] `./scripts/run-verify.sh` green — evidence `docs/evidence/verify-2026-05-13-024222.log`
- [x] `./scripts/check-skill-sync.sh` green (13 skills in lock-step)
- [x] `./scripts/check-sync.sh` green (145 identical, 0 drifted)
- [x] Review artifact created (`docs/reports/self-review-2026-05-13-fix-xreview-placeholder-substitution.md`, cycle 2: `…-cycle2.md`)
- [x] Verification artifact created (`docs/reports/verify-2026-05-13-fix-xreview-placeholder-substitution.md`)
- [x] Test artifact created (`docs/reports/test-2026-05-13-fix-xreview-placeholder-substitution.md`)
- [ ] PR created
