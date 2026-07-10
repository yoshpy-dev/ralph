# standard-flow-orchestrator

- Status: Approved (autonomous goal session; decisions self-selected and recorded below)
- Owner: Claude Code
- Date: 2026-07-11
- Related request: Use the session model (Fable 5) as orchestrator brain and delegate execution to cheaper models (PR2 of 2; PR1 = #115 Ralph Loop per-phase routing)
- Related issue: N/A
- Type: feat
- Branch: feat/standard-flow-orchestrator

## Objective

Bring the orchestrator–worker discipline to the standard flow (/work): the
main session (session model, e.g. Fable 5) plans, delegates, adjudicates
reports, and reviews; implementation slices run on a new `implementer`
subagent pinned to `sonnet` via a structured handoff. Also fix the
`--model opus` hardcode in the cross-review skill so the standard flow
respects `RALPH_CLAUDE_REVIEWER_MODEL`.

## Scope

1. **New `implementer` subagent** — `.claude/agents/implementer.md`
   (frontmatter: `model: sonnet`, tools Read/Grep/Glob/Bash/Write/Edit,
   `memory: project` matching existing agents). System prompt: scoped slice
   implementer that (a) works from a structured handoff (plan path, slice
   objective, acceptance criteria, exact verification commands, files in
   scope, commit message format), (b) confirms a clean baseline
   (`git status --porcelain`) before starting, (c) runs the stated
   verification before committing, (d) stages ONLY the handoff-listed paths
   (never `git add -A`/`-u`) and includes `git status --porcelain` +
   `git show --stat HEAD` output in the report as commit-boundary evidence,
   (e) returns a report contract (changed files, decisions/deviations,
   verification evidence, commit SHA), (f) refuses to widen scope
   (out-of-scope discoveries go back to the orchestrator). Codex parity:
   `.codex/agents/implementer.toml` with equivalent `developer_instructions`.
   Mirrors: `templates/base/.claude/agents/implementer.md` and
   `templates/base/.codex/agents/implementer.toml` (root ↔ template
   byte-identical per `check-sync.sh`).
2. **/work orchestrator discipline** — `.claude/skills/work/SKILL.md`:
   amend the implementation steps so slices are delegated to the
   `implementer` subagent with the structured handoff above; the main
   session stays on decomposition, handoff authoring, report adjudication,
   and plan upkeep. Inline implementation remains allowed for (a) trivial
   single-file edits where a handoff costs more than the change, and
   (b) dispatch failure (fallback, noted in the report — same convention as
   the post-impl pipeline). Mirror to `.agents/skills/work/SKILL.md` and
   both `templates/base/` copies; `check-skill-sync.sh` must pass.
3. **cross-review model hardcode fix** — `.claude/skills/cross-review/SKILL.md`
   lines ~55 and ~154: `claude -p --model opus` →
   `claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"`. Narrow claim
   (Codex advisory finding 3): the skill honors the **exported env var** with
   an `opus` fallback — the variable reaches a manual `/cross-review` run via
   the operator's environment or by sourcing `scripts/ralph-config.sh` (the
   skill already sources it in a subshell for the cycle cap; the reviewer
   step will note the same sourcing for the model). No claim that ralph.toml
   is read directly by the skill. Same 4-copy mirror discipline.
4. **model-routing.md** — add a "Standard flow delegation (/work)" section
   documenting: implementation slices → `implementer` (sonnet) with
   structured handoff; session model reserved for planning/decomposition/
   adjudication/final review; inline-implementation exceptions; note that
   `.claude/skills/cross-review/SKILL.md` now reads
   `RALPH_CLAUDE_REVIEWER_MODEL` (closing the sync-target gap). Template
   mirror follows the existing KNOWN_DIFF policy.
5. **subagent-policy.md** — new section "Implementation (/work) — delegate to
   `implementer`": when to delegate vs inline, the handoff contract fields,
   the fallback convention, and Codex custom-agent parity. Mirror to
   `templates/base/`.

## Non-goals

- No shell/Go changes (`ralph-pipeline.sh`, `internal/`) — Loop routing
  shipped in PR #115.
- No new env vars: the implementer tier is pinned in agent frontmatter per
  the existing model-routing rule.
- No enforcement hook that blocks inline implementation — this is rule/skill
  guidance, consistent with how the rest of the pipeline discipline works.
- No changes to reviewer/verifier/tester/doc-maintainer agents.

## Assumptions

- Claude Code discovers `.claude/agents/implementer.md` as
  `Task(subagent_type="implementer")` without code changes; Codex likewise
  picks up `.codex/agents/implementer.toml`.
- `${RALPH_CLAUDE_REVIEWER_MODEL:-opus}` in the documented command is safe:
  when the variable is unset the `:-opus` fallback preserves today's
  behavior exactly.

## Affected areas

- `.claude/agents/implementer.md` (new) + `templates/base/.claude/agents/`
- `.codex/agents/implementer.toml` (new) + `templates/base/.codex/agents/`
- `.claude/skills/work/SKILL.md`, `.agents/skills/work/SKILL.md` + both
  `templates/base/` copies
- `.claude/skills/cross-review/SKILL.md`,
  `.agents/skills/cross-review/SKILL.md` + both `templates/base/` copies
- `.claude/rules/model-routing.md`, `.claude/rules/subagent-policy.md` + both
  `templates/base/` copies
- Possibly `CLAUDE.md`/`AGENTS.md` one-liners if a statement becomes stale
  (keep map-sized)

## Design decisions

Critical forks resolved autonomously (goal session; pre-authorized):

1. **Implementer tier pinned in frontmatter (`model: sonnet`), no env knob.**
   Matches the existing rule ("always pin model: in agent frontmatter") and
   keeps /work symmetrical with reviewer/verifier/tester/doc-maintainer.
   Escalation for a judgment-heavy slice stays an orchestrator choice
   (explicit Task `model` parameter), not a new variable.
2. **Guidance, not enforcement.** Inline implementation stays legal for
   trivial edits and dispatch failures; a hard hook would fight legitimate
   use and diverge from how the pipeline's other disciplines are specified.
3. **Merge-conflict avoidance with PR #115**: model-routing.md gains the new
   section directly after the tier-table paragraph (before "## Rules"), and
   "Where the values live" is NOT edited (the cross-review sync note lives in
   the new section instead) — #115 rewrote the post-Rules region and the
   values list, so PR2 avoids those hunks entirely. subagent-policy.md edit
   goes near the /work pipeline section, which #115 did not touch beyond the
   /loop paragraph.
4. **Structured handoff contract mirrors the plan-artifact doctrine**
   (quality is preserved by plan precision, not model tier): the handoff must
   carry acceptance criteria, exact verification commands, file scope, and a
   report contract — the standard-flow analogue of what `ralph-pipeline.sh`
   prompts already encode.

## Acceptance criteria

- [ ] AC1: `.claude/agents/implementer.md` exists with `model: sonnet`
  frontmatter and the handoff/baseline/staging-allowlist/report prompt
  (incl. "stage only handoff-listed paths, never `git add -A`", baseline
  `git status --porcelain` check, and status/diff evidence in the report);
  `.codex/agents/implementer.toml` carries equivalent instructions; all four
  copies (root + templates/base) exist and `check-sync.sh` passes.
- [ ] AC1b (dispatch smoke — AMENDED, see deviation note): dispatch of
  `Task(subagent_type="implementer")` was attempted for Slice 2 and failed
  with "Agent type 'implementer' not found" — the subagent registry loads at
  session start from the project root, and the new definition exists only in
  this task worktree until merge. The documented fallback convention was
  exercised instead (inline-role fallback, noted in slice reports), which
  itself validates the skill's exception path. Runtime dispatch smoke for
  BOTH Claude Code and Codex is therefore recorded as a known gap: verify in
  a fresh session after merge (`Task(subagent_type="implementer")` appears in
  the agent list). This gap and the failed-dispatch evidence must appear in
  the test report and PR body.
- [ ] AC2: `/work` SKILL.md (all 4 copies) instructs delegating
  implementation slices to `implementer` with the structured handoff, states
  the two inline exceptions, and `check-skill-sync.sh` passes.
- [ ] AC3: no bare `claude -p --model opus` remains for the claude reviewer
  path: both occurrences read `"${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"`;
  `grep -rn 'claude -p --model opus' .claude .agents templates/base` → 0 hits.
  (Grep-verified only; the env-var-honoring claim is scoped per Scope 3.)
- [ ] AC4: model-routing.md documents standard-flow delegation without
  touching "Where the values live" or the post-Rules region;
  subagent-policy.md documents the implementer policy. Because
  model-routing.md is a KNOWN_DIFF in `check-sync.sh` (gate cannot catch a
  missing section), explicitly verify:
  `grep -l 'Standard flow delegation' .claude/rules/model-routing.md
  templates/base/.claude/rules/model-routing.md` → both files hit.
- [ ] AC5: `./scripts/run-verify.sh` passes (includes check-sync,
  check-skill-sync, shellcheck, go gates) and full `./scripts/run-test.sh`
  stays green (regression only — no code change).

## Implementation outline

1. **Slice 1 — implementer agent (4 files)**: author
   `.claude/agents/implementer.md` + `.codex/agents/implementer.toml`,
   mirror both to `templates/base/`, verify, commit.
2. **Slice 2 — skills (work + cross-review, 8 files)**: /work delegation
   steps + cross-review model var fix, mirrors, `check-skill-sync.sh`,
   verify, commit.
3. **Slice 3 — rules (4 files)**: model-routing.md + subagent-policy.md and
   template mirrors; drift sweep of CLAUDE.md/AGENTS.md; verify, commit.

## Verify plan

- Static analysis: `./scripts/run-verify.sh` (sync gates, shellcheck, etc.).
- Spec compliance: AC1–AC5 against the diff.
- Documentation drift: model-routing/subagent-policy/work/cross-review must
  tell one consistent story; grep for stale claims that /work implements
  everything inline.
- Evidence: docs/reports/verify-2026-07-11-standard-flow-orchestrator.md.

## Test plan

- Docs/agent-definition-only PR — behavioral coverage = regression: full
  `./scripts/run-test.sh` (shell glob + go test) must stay green.
- Edge cases via grep/review: no residual `--model opus` (AC3), agent
  frontmatter well-formed (matches existing agent file shape).
- Evidence: docs/reports/test-2026-07-11-standard-flow-orchestrator.md.

## Risks and mitigations

- **Over-delegation churn** (trivial edits routed through subagents) →
  explicit inline exceptions in the skill text.
- **Weaker implementation quality on sonnet** → structured handoff contract
  + unchanged post-impl pipeline (reviewer=opus, verify, test, cross-review).
- **Merge conflict with #115** → Design decision 3 (disjoint hunks); if #115
  merges first, rebase is trivial.
- **4-copy drift** → deterministic sync gates already fail CI.

## Rollout or rollback notes

- Rollout: merge after (or independently of) #115; scaffolded projects pick
  it up via `ralph upgrade`.
- Rollback: revert the PR; no state or config migration. Operators can also
  simply keep implementing inline — the discipline is guidance.

## Codex plan advisory (evidence)

Codex (codex-cli 0.139.0, read-only) returned 4 findings; all adopted:

1. [HIGH] Delegation could pass ACs as paper-only → AC1b added: Slices 2–3
   of this plan dispatch `Task(subagent_type="implementer")` as runtime
   smoke; Codex-side dispatch recorded as known gap.
2. [HIGH] Delegated commit boundary underspecified → Scope 1 + AC1 now
   require baseline check, staging allowlist (no `git add -A`), and
   status/diff evidence in the report.
3. [MEDIUM] Reviewer-model claim broader than verification → Scope 3 and AC3
   narrowed to "exported env var with opus fallback"; no ralph.toml claim.
4. [MEDIUM] model-routing.md is KNOWN_DIFF so the sync gate cannot catch a
   missing template section → AC4 adds an explicit both-copies grep.

## Open questions

- None blocking. Follow-up candidates (recorded, not in scope): a haiku
  "explorer" agent definition for bulk read-only sweeps; measuring actual
  token savings from receipts + session usage; runtime smoke for Codex
  custom-agent dispatch of `implementer`.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (feat/standard-flow-orchestrator)
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
