# Walkthrough: fix-xreview-placeholder-substitution (issue #50)

PR diff is ~1,400 lines across 17 files. The implementation footprint in production code is small (~80 lines in `scripts/ralph-pipeline.sh`); the rest is test scaffolding, the templates/base/ mirror, and pipeline reports.

## TL;DR

Under `RALPH_LOOP_DRIVER=codex`, `ralph-pipeline.sh` piped the adversarial-claude prompt to `claude -p` with `${BASE_BRANCH}` / `${REPORTS_DIR}` placeholders intact (claude does not template-substitute stdin). The reviewer wrote its triage report to an unparseable literal path, the downstream `find` parser returned empty, `_action_required` stayed at its initial `0`, and the cross-review gate silently proceeded to PR creation — exactly the failure shape reported in issue #50.

After the fix:
1. The prompt is **pre-rendered** with `awk index()/substr()` (literal-string replacement) into a per-cycle copy under `${PIPELINE_DIR}/outer-N-adversarial-claude.md` before `claude -p` consumes it.
2. An **allowlist guard** fails the cross-review gate closed if any `${...}` token survives rendering (defence against future placeholder drift).
3. A new **`_render_failed` flag** is consulted by the gate decision **before** `_action_required`, so render failures regress the Inner Loop instead of silently falling through.
4. The `render_failed` field is recorded in the cross-review checkpoint and `report_event` JSONL line for operator visibility.

## Tour of the diff

### 1. Production code — `scripts/ralph-pipeline.sh` cross-review phase

The change is localized to the `claude` arm of the `case "$_reviewer"` block. Walk through it in order:

**`scripts/ralph-pipeline.sh:772` — flag init**
```sh
_render_failed=0
```
Sits alongside `_action_required=0` / `_worth_considering=0` / `_dismissed=0`. Initialized once per cross-review phase entry.

**`scripts/ralph-pipeline.sh:802-826` — the renderer (awk index/substr)**
```sh
_rendered_prompt="${PIPELINE_DIR}/outer-${_cycle}-adversarial-claude.md"
if ! BASE_BRANCH="$_base" REPORTS_DIR="$REPORTS_DIR" \
     awk '
       function lreplace(s, needle, repl,    out, idx) {
         out = ""
         while ((idx = index(s, needle)) > 0) {
           out = out substr(s, 1, idx - 1) repl
           s = substr(s, idx + length(needle))
         }
         return out s
       }
       {
         line = $0
         line = lreplace(line, "${BASE_BRANCH}", ENVIRON["BASE_BRANCH"])
         line = lreplace(line, "${REPORTS_DIR}", ENVIRON["REPORTS_DIR"])
         print line
       }
     ' "$_adv_prompt" > "$_rendered_prompt"; then
  log_error "cross-review: failed to render adversarial prompt to ${_rendered_prompt}"
  echo "render_failed_awk" > "$_xreview_log"
  _render_failed=1
else
```

Key choices:
- **`index()` + `substr()` not `gsub()`**: `awk gsub` interprets `&` in the replacement string as "matched text" — so a `REPORTS_DIR=docs/reports&backup` value would produce `docs/reports${REPORTS_DIR}backup` (Codex caught this during plan review). `index()` + `substr()` is true literal replacement.
- **Single-quoted awk program**: the shell never expands `${BASE_BRANCH}` in the pattern; awk sees the literal `${BASE_BRANCH}` string.
- **`ENVIRON["BASE_BRANCH"]` lookup**: awk reads from its own environment instead of shell interpolation, so `set -u` cannot fire on an unset placeholder name.

**`scripts/ralph-pipeline.sh:831-841` — allowlist guard**
```sh
_leftover_placeholders="$(grep -oE '\$\{[A-Z_][A-Z0-9_]*\}' "$_rendered_prompt" 2>/dev/null | sort -u || true)"
if [ -n "$_leftover_placeholders" ]; then
  log_error "cross-review: unresolved placeholders in rendered prompt: ${_leftover_placeholders}"
  echo "render_failed_unresolved_placeholders" > "$_xreview_log"
  _render_failed=1
fi
```
Defence-in-depth: if someone adds `${PLAN_PATH}` (or any new placeholder) to the prompt and forgets to extend the renderer, the rendered output will still contain that token. The grep catches it and fails the gate closed.

**`scripts/ralph-pipeline.sh:844-853` — reviewer invocation (now guarded)**
```sh
if [ "$_render_failed" -eq 0 ]; then
  claude -p --model "$RALPH_CLAUDE_REVIEWER_MODEL" \
    --permission-mode auto --output-format text \
    < "$_rendered_prompt" 2>&1 | tee "$_xreview_log" || true
fi
```
The only structural change to the reviewer call: read from `$_rendered_prompt` (not `$_adv_prompt`) and skip if rendering failed.

**`scripts/ralph-pipeline.sh:884-885` — telemetry**
The `cross_review_triage` checkpoint object and the `report_event "cross-review"` JSONL line both gain a `render_failed` field, so operators can distinguish "reviewer ran clean" from "renderer broke" in pipeline logs.

**`scripts/ralph-pipeline.sh:893-896` — gate decision (the fix for the self-review CRITICAL)**
```sh
if [ "$_render_failed" -ne 0 ]; then
  log "Adversarial prompt render failed — regressing to Inner Loop (gate fails closed)"
  return 1
fi
```
This check runs **before** `_action_required > 0`. Without it, the original commit (`0304686`) would have set `_adv_prompt=""` on render failure, skipped the reviewer, then fallen through to the parser which would see no triage report, leaving `_action_required=0` — reproducing the exact silent-bypass shape of issue #50 inside the fix meant to close it. The reviewer caught this in cycle 1; the cycle-2 commit (`12a1984`) wired the flag through.

### 2. Prompt contract — `.claude/skills/cross-review/prompts/adversarial-claude.md`

Top-of-file HTML comment block documenting that `${BASE_BRANCH}` and `${REPORTS_DIR}` are pre-rendered by `ralph-pipeline.sh` and that adding a new placeholder requires updating both the renderer AND the allowlist guard. The cycle-2 commit added `\` to the metacharacter list (the renderer is safe against `\` but the comment originally only mentioned `#`, `&`, `/`).

### 3. Skill body — `.claude/skills/cross-review/SKILL.md` (+ Codex mirror)

New "Prompt rendering contract (claude reviewer path)" subsection under "Reviewer inversion inside Ralph Loop", with pointers to the regression tests. Mirrored byte-identical into `.agents/skills/cross-review/SKILL.md` (`scripts/check-skill-sync.sh` enforces this).

### 4. Regression tests — `tests/test-xreview-prompt-render.sh` (renderer)

54 assertions across 7 parameterized cases:

| Case | `_base` | `REPORTS_DIR` | Tests |
|------|---------|---------------|-------|
| 1 | `main` | `docs/reports` | baseline |
| 2 | `release/3.5` | `docs/reports` | `/` in ref |
| 3 | `feature#1` | `docs/reports` | `#` in ref |
| 4 | `feature&1` | `docs/reports` | `&` in ref (the awk `gsub` trap) |
| 5 | `feature\back` | `docs/reports` | `\` in ref |
| 6 | `main` | `docs/reports#1` | `#` in REPORTS_DIR |
| 7 | `main` | `docs/reports&backup` | `&` in REPORTS_DIR |

Each case asserts: no `${BASE_BRANCH}` / `${REPORTS_DIR}` literals remain, both substituted values appear unchanged, unrelated `$` tokens (`$0.00`, `${arr[0]}`) are preserved, and the allowlist guard finds no leftovers. Negative case: a hand-injected `${UNKNOWN_PLACEHOLDER}` MUST trip the allowlist guard while supported placeholders MUST NOT.

A drift assertion (`grep -q 'function lreplace(s, needle, repl' "$PIPELINE_SH"`) keeps the test renderer in lock-step with the production one.

### 5. Regression tests — `tests/test-xreview-gate-regression.sh` (end-to-end gate)

21 assertions across 5 phases, sourcing `count_triage_findings` directly from `scripts/ralph-cli-driver.sh` (the production parser, not a copy):

- **Phase 1 — render the real prompt**: renders `.claude/skills/cross-review/prompts/adversarial-claude.md` with `main` + a sandbox `REPORTS_DIR`; asserts no placeholders remain and the allowlist guard accepts the output.
- **Phase 2 — `ACTION_REQUIRED=1` triage MUST regress**: writes a synthetic triage report with one ACTION_REQUIRED row, drives `count_triage_findings`, asserts the gate-decision function returns non-zero.
- **Phase 3 — clean triage MUST proceed**: same shape with zero findings; gate returns zero.
- **Phase 4 — `--fix-all` interaction**: WORTH_CONSIDERING=1 regresses under `--fix-all`, does not regress without it.
- **Phase 5 — render-failure path MUST regress end-to-end**: simulates `RENDER_FAILED=1` plus a non-existent triage file; gate MUST regress without consulting the parser. Plus drift assertions that the pipeline (a) initializes `_render_failed=0`, (b) sets it to 1 in at least one error branch, (c) gates on it **before** `_action_required`.

### 6. `templates/base/` mirror

`scripts/check-sync.sh` enforces byte-identical copies in `templates/base/` so `ralph init` ships the same harness ralph itself dogfoods. Four files mirrored: `scripts/ralph-pipeline.sh`, `.claude/skills/cross-review/SKILL.md`, `.agents/skills/cross-review/SKILL.md`, `.claude/skills/cross-review/prompts/adversarial-claude.md`.

### 7. Pipeline reports (`docs/reports/`)

- Cycle-1 self-review (caught the CRITICAL render-failure bypass)
- Cycle-2 self-review (verified the fix, verdict: merge)
- Verify (all 9 ACs verified, static analysis clean)
- Test (54/54 + 21/21 + existing regressions all green)
- Sync-docs (audit of AGENTS.md, CLAUDE.md, README.md, recipes/, types.go; no drift)
- Cross-review triage (Codex returned one P2, triaged WORTH_CONSIDERING)

### 8. Tech-debt entry

The awk renderer body is duplicated three places (pipeline + 2 tests) with a name-based drift guard. Tech-debt entry logged in `docs/tech-debt/README.md` for the next time the renderer grows — the cleanup path is to extract `render_adversarial_prompt` into `scripts/ralph-cli-driver.sh` next to `count_triage_findings`.

## Commits in order

| SHA | Subject |
|-----|---------|
| `0304686` | fix: pre-render cross-review prompt placeholders before claude -p (#50) |
| `4f15681` | test: add renderer unit test for cross-review prompt placeholders (#50) |
| `d2dd875` | test: add end-to-end cross-review gate-regression test (#50) |
| `f3363b6` | chore: mirror cross-review prompt rendering fix into templates/base/ (#50) |
| `fd3e958` | docs: update plan progress checklist for xreview placeholder fix (#50) |
| `12a1984` | fix: fail cross-review gate closed on render failure (#50 self-review CRITICAL) |
| `7f38512` | docs: sync-docs after issue #50 fix — tick artifact boxes + audit report |
| `61a9aed` | docs: record cross-review triage for issue #50 fix |

## Known follow-up

Codex cross-review surfaced one WORTH_CONSIDERING [P2]: if a future `_base` or `REPORTS_DIR` legitimately contains literal `${SOMETHING}` text (e.g. a git ref named `release-${YEAR}`), the post-render allowlist scan would misclassify it as an unresolved placeholder. The canonical fix is to scan the **template** (input file) for unsupported placeholders before injecting replacement values. Deferred per triage: real but low-probability, loud failure mode (clear error log, gate fails closed — not silent), manual workaround (rename branch). Tracked in `docs/reports/cross-review-triage-2026-05-13-fix-xreview-placeholder-substitution.md`.
