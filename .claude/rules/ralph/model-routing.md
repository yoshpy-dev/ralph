# Model routing

How to assign Claude models when delegating work to subagents or background
tasks. Delegation timing/order lives in `subagent-policy.md`.

## Tier table

| Seat group | Model | Agents and typical work |
|------------|-------|-------------------------|
| Orchestrator (main session) | session model (user's choice) | planning, decomposition, arbitration, final review |
| Implementation and verification seats | `opus` | scoped implementation (`implementer`), spec/static verification (`verifier`), test execution (`tester`), design trade-offs, ambiguous root-cause debugging |
| Review and doc seats | `sonnet` | diff-quality review (`reviewer`), doc sync (`doc-maintainer`) |
| Bulk mechanical work | `haiku` | grep/file inventory, log scanning, large read-only sweeps |

Quality is preserved by plan artifacts, not by model tier: when delegating,
the prompt must carry acceptance criteria, exact verification commands, and a
report contract (changed files, key decisions, verification evidence, the
mechanical self-check result). A cheaper model with a precise plan beats an
expensive model with a vague one.

## Standard flow delegation (/implement)

Implementation slices in `/implement` are delegated to the `implementer` subagent
(`model: opus` pinned in frontmatter; Codex: `.codex/agents/implementer.toml`).
The orchestrator (session model) retains planning, decomposition, handoff
authoring, report adjudication, and final review — it does not write slice code
itself.

**Structured handoff must carry:**

| Field | Required content |
|-------|-----------------|
| Plan path | worktree-relative path to the active plan |
| Slice objective | single-sentence goal |
| Acceptance criteria | the ACs this slice addresses |
| Files in scope | exact list; implementer stages only these paths |
| Exact verification commands | the slice's command from the plan's `## Implementation outline` (the narrowest check that covers the slice's files), copy-paste ready |
| Commit message format | conventional format string |
| Resume (only when dispatching a stopped slice again) | the uncommitted changes the stopped slice left in the files in scope, named as the slice's own (`/implement` step 7) |

**Report contract (implementer → orchestrator):** changed files, decisions/deviations,
verification evidence, the mechanical self-check result, commit-boundary
evidence (`git status --porcelain` + `git show --stat HEAD` output), commit SHA.
The mechanical self-check has five items, and this is the canonical list that
every other mention repeats word for word: unnecessary changes, typos and
copy-paste errors, debug leftovers, secrets, and files outside the slice. The
result says, for each item, clean, fixed (what), or found and not fixed (what
and why), and names a secret by its file and line, never by its value. When
an item is found and not fixed, the orchestrator has it fixed
as its own slice before the next slice (`/implement` step 7). For the last
slice, that happens before step 13. A secret that cannot be removed within
scope is never committed: the implementer stops without a commit, and the
orchestrator first takes the secret out of the worktree, narrows the slice if
needed, and dispatches it again as a resume, whose handoff names the
uncommitted changes the stopped slice left in the files in scope as the
slice's own (`/implement` step 7).

**Inline exceptions** (no subagent dispatch needed):

- Trivial single-file edits where the handoff overhead exceeds the change cost.
- Dispatch failure → inline fallback, noted in the report (same convention as
  the post-implementation pipeline fallback).

**Overriding a seat's default:** pass an explicit `model` on the Task call —
no new env knob. Typical cases: `opus` on the `reviewer` call for a
security-sensitive diff, or `sonnet` on an `implementer` call for a mechanical
slice.

**Cross-review sync note:** `.claude/skills/cross-review/SKILL.md` and the
plan advisory of `.claude/skills/plan/SKILL.md` take the other CLI's reviewer
model and effort from the four keys of `[reviewer]` in `ralph.toml`:
`claude_model` and `claude_effort` for a claude reviewer
(`claude -p --model <claude_model> --effort <claude_effort>`), and
`codex_model` and `codex_effort` for a codex reviewer
(`codex -m <codex_model> -c model_reasoning_effort=<codex_effort>`). A
`ralph.local.toml` in the main worktree's root overrides them on one machine.
Both skills read a key with `ralph_setting <key>`, the internal helper in
`scripts/ralph-config.sh`, which runs `ralph config get <key>` and falls back
to its own table of the same defaults when that fails. The plan advisory of a
feature org's leader (its role prompt, `internal/org/prompts/leader.md` in the
`ralph` repository source, not part of a scaffolded project) reads no
setting: `ralph org start --plan` writes the values it read into the leader's
task as two lines, `- reviewer の codex: <model> <effort>` and
`- reviewer の claude: <model> <effort>`, and the leader uses those, so an org
keeps the values it started with. No environment variable changes these
values. When a default changes, change `Default()` in
`internal/config/config.go`, the helper's table, and the comments of
`templates/base/ralph.toml` together; `internal/config/defaults_sync_test.go`
fails until they agree.

## Rules

- **Always pin `model:` in agent frontmatter.** Omitted `model:` means
  `inherit` — a subagent spawned from an expensive main session silently runs
  on that expensive model, multiplied by parallel fan-out.
- **Use stable aliases (`fable`, `opus`, `sonnet`, `haiku`), not full model IDs**, in
  agent frontmatter, `[reviewer].claude_model` of `ralph.toml` (and its
  built-in default), and skill docs. Full IDs go stale and can break
  `claude -p` at runtime. When a reviewer run must be reproducible, pin a full
  ID in `[reviewer].claude_model` of `ralph.toml`, or of `ralph.local.toml` to
  pin it on your machine only; the key accepts the characters of a full ID
  (`A-Z a-z 0-9 . _ : @ / + -`). Agent frontmatter keeps its alias.
- **Codex has no model aliases.** `[reviewer].codex_model` holds a literal
  codex model slug (`gpt-6-astra` by default, from `config.Default()` in
  `internal/config/config.go`), unlike the Claude-side aliases above. Change
  it when codex retires the slug: `ralph doctor` checks the slugs of
  `[org].model_pool`, whose default holds the same slug, and reports one that
  is retiring or retired.
- **Do not export `CLAUDE_CODE_SUBAGENT_MODEL`.** It silently overrides every
  frontmatter `model:` and per-call `model` parameter. Treat it as an
  emergency-only blunt instrument.
- **Ad-hoc Task/Agent calls should pass `model` explicitly** when the work is
  bulk search or mechanical (e.g. Explore fan-outs → `haiku`); otherwise they
  inherit the session model.
- **Scheduled, cron, and background executions must state their model
  explicitly.** They otherwise run on whatever model is currently saved as
  the session default.
- **Switch models at task boundaries, not mid-conversation.** A mid-session
  model or effort switch invalidates the prompt cache and re-reads the full
  history uncached on the next response.
- Keep a subagent's `effort` at the default (`high`) unless measured evidence
  justifies a change; `xhigh`/`max` have documented diminishing returns. The
  other CLI's reviewer (`/plan`'s plan advisory, `/cross-review`, and a
  feature org leader's plan advisory) is set apart: `[reviewer].claude_effort`
  (one of `low`, `medium`, `high`, `xhigh`, `max`) and `[reviewer].codex_effort`
  set its effort, `xhigh` by default for both. The claude reviewer gets it as
  `claude -p --effort`, which `-p` honors
  (`docs/evidence/claude-effort-probe-2026-10-30.md`). Lower either key in
  `ralph.toml` or `ralph.local.toml` when measured evidence shows the reviews
  do not need it.

## Org runtime model receipts

`ralph org spawn` seats are commanded with an explicit model per seat, not
via the tier table above. Each spawn appends a JSON line to
`.harness/state/org/model-receipts.jsonl` (`internal/org/receipts.go`) with
`ts / org_id / seat_id / role / driver / commanded_model /
reported_effective_model / honored / reason`. This is a separate mechanism
from the `/implement` subagent tiers documented above — see
`docs/specs/2026-08-01-org-runtime.md` for the org runtime's own model
selection rules. `ralph insights` reads this file by default (resolved via
the same org state-dir precedence as `ralph org` verbs) and aggregates it
by `org_id` x `seat_id` with tri-state `honored` (true/false/unknown) in
its Receipts section. Codex seats have no aliases; `[org].model_pool`
carries codex model slugs, and `ralph doctor` warns when a slug is missing
from codex's local model cache.

For a codex seat the receipt carries an observation, not only the command:
after the seat starts, `ralph org spawn` looks for up to 8 s for the seat's
codex session record (`$CODEX_HOME/sessions/`, else `~/.codex/sessions/`;
`internal/org/codex_session.go`) and records its `turn_context.model` as
`reported_effective_model`, with `honored=true` when it equals the commanded
model and `honored=false` when it does not (codex switches a retiring model
to its replacement on its own; `ralph doctor` lists pool slugs that carry
such a notice). The record is matched by the sentence ralph passes to the
seat to point it at its role-prompt file and by a session start time not
before the spawn, and only the model is read from it. A seat whose record
cannot be identified stays `unknown`: no turn yet (for example a
model-retirement dialog is open), an inline or empty initial prompt (no
role-prompt file to match), or a herdr server whose `CODEX_HOME` differs
from ralph's. `ralph org stop` looks once more for a seat that has no
observed receipt since its spawn and, when it finds the record, appends
one (that single look is itself bounded by the same up-to-8s budget as
spawn's poll), so a seat can have an `unknown` receipt from spawn and an
observed one from stop. Stop also covers a session that started days after
the spawn (a startup dialog answered late), up to about a month. claude
seats are not observed and stay `unknown`.

## Where the values live

- `.claude/agents/*.md` — pipeline subagent tiers (frontmatter `model:`)
- `ralph.toml`, and `ralph.local.toml` in the main worktree's root for one
  machine — the values a person sets: `[standard].max_pipeline_cycles` (the
  pipeline cycle cap) and the four `[reviewer]` keys (`claude_model`,
  `claude_effort`, `codex_model`, `codex_effort`). A key neither file sets
  keeps its built-in default, `Default()` in `internal/config/config.go`, which
  `templates/base/ralph.toml` writes as comments. ralph no longer reads
  `RALPH_STANDARD_MAX_PIPELINE_CYCLES`, `RALPH_CLAUDE_REVIEWER_MODEL`,
  `RALPH_CODEX_REVIEWER_MODEL`, or `RALPH_CODEX_REASONING_EFFORT`, and
  `ralph doctor` warns when one is still set.
- `scripts/ralph-config.sh` — the internal helper that reads a key with
  `ralph config get`. The skills source it to read those five keys with
  `ralph_setting <key>`; `scripts/plan-visual.sh`,
  `scripts/xreview-helpers.sh`, `scripts/detect-changed-languages.sh`, and
  `scripts/run-verify.sh` source it to read five keys only `ralph.local.toml`
  may set (`[plan_visual].browser` and `opener`, `[standard].base_branch`,
  `[verify].base` and `skip_branch_secret_scan`), the two bases with
  `ralph_setting_or_fail <key>`. It holds no value a person sets, only a
  table of the defaults of those ten keys. Both functions print the default
  silently when `ralph` is not on PATH or has no `config get`. On any other
  failure `ralph_setting` prints the default after one line on stderr, and
  `ralph_setting_or_fail` writes that line, prints nothing, and fails, so a
  mistake in the settings does not quietly change the base a verify, a
  review, or a secret scan uses. It reads no environment variable.
- `internal/config/defaults_sync_test.go` (`TestDefaultsLockStep`; the checks of the commented keys, the helper's table, and the skills live in `defaults_sync_settings_test.go`) — asserts that `config.Default()`'s `[org]` values match `templates/base/ralph.toml` for the `[org]` keys `orgTemplateDrift` reads (each key set, same value; not `[org.roles]`, `[org.permissions].roles`, `launch_cmd`, or `wake_cmd`; `[org.review]` and `[org.gate]` are covered), that `standardTemplateDrift` finds the two `[standard]` switches set in the template with `Default()`'s values, that `commentedTemplateDrift` finds the ten keys of the helper's table (`[standard].max_pipeline_cycles`, the four `[reviewer]` keys, and the five keys only `ralph.local.toml` may set) only as comments in the template, each once and with `Default()`'s value, that `[org.review].max_cross_review_cycles` defaults to the same value as `[standard].max_pipeline_cycles` in `Default()`, that the table of `scripts/ralph-config.sh` holds exactly those ten keys with `Default()`'s values, and that the four copies of the `/plan`, `/cross-review`, and `/implement` SKILL.md expand none of the four variables above, read the settings with `ralph_setting` and only keys of its table, pass `--effort` to each claude reviewer launch, and state the cap rule of `cycle-count.json`'s `cap`
- `internal/org/prompts_leader_plan_test.go` — asserts that the feature org leader's plan advisory takes the reviewer's model and effort from the task's two reviewer lines, which `featureLeaderTask` writes from `[reviewer]`, and that the role prompt names none of `scripts/ralph-config.sh`, `ralph config get reviewer`, and the four variables above (`TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryTakesTheReviewerFromTheTask`), and that its codex and claude commands hold the placeholders those lines fill in, at `-m` and `model_reasoning_effort` (`TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexTurnsMCPOff`) and at `--model` and `--effort` (`TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeReadsOnly`)
- `tests/test-agent-models.sh` — asserts the tier table and the implementer
  pin sentence above match `.claude/agents/*.md` frontmatter, on both the root
  and `templates/base/` sides
