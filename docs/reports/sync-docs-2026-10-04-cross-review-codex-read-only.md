# sync-docs report: cross-review-codex-read-only

## Cycle 1

- Date: 2026-10-04
- Plan: `docs/plans/active/2026-10-03-cross-review-codex-read-only.md` (issue #197, security)
- Pipeline cycle: 1 of 2. Delta: `main` `c9da0a27` to branch HEAD `e8507e87`.
- Prior reports (the commits that added them):
  `docs/reports/self-review-2026-10-03-cross-review-codex-read-only.md` (`3c11bf74`, addendum `1280d797`; merge ok),
  `docs/reports/verify-2026-10-03-cross-review-codex-read-only.md` (`407da9f6`; Pass, O-3 deferred to this step),
  `docs/reports/test-2026-10-03-cross-review-codex-read-only.md` (`2a31ea9f`, addendum `698d0d5c`; Pass, G-1 closed by Slice D `2d385bc3`, G-7 closed by Slice E `2b2fac0c`)

### Summary

The documentation that shipped with the change (the skill explanations in
`/cross-review` Step 4 and `/plan` step 11.c, the call-shape paragraph in
`docs/recipes/codex-setup.md`, the `.codex/config.toml` comments, and the
tech-debt rows) matches the skills' invocation lines at HEAD. No other doc
surface states the codex call's sandbox flags or its argument order, so the
only edits are the three the hand-off asked for: a short sentence on where the
read-only `-c` must sit, the O-3 rephrase, and the extension of the
residual-risk row. No plan, other report, skill invocation line, test,
workflow, `.gitallowed`, or scanner file was touched.

### Delta reviewed

`git diff main...HEAD` excluding `docs/plans`, `docs/reports`, `docs/insights`
(14 files before this pass):

- Skills, four copies each (`.claude/skills`, `.agents/skills`,
  `templates/base/.claude/skills`, `templates/base/.agents/skills`):
  `/cross-review` Step 4 (`:58`) and the CLI execution modes table (`:167`) run
  `-c sandbox_mode=read-only exec review --ignore-rules`; `/plan` step 11.c
  (`:71`) runs `exec --sandbox read-only --ignore-rules`. Both carry an
  explanatory paragraph (`/cross-review` `:60`, `/plan` `:73`).
- `.codex/config.toml`, `templates/base/.codex/config.toml` (byte-identical):
  comments only. The `sandbox_mode` and `approval_policy` comments say `codex
  exec` ignores `approval_policy`, the `"untrusted"` option is gone, and the
  `[profiles.review]` comment lists the explicit flags the reviewer passes.
- `docs/recipes/codex-setup.md`, `templates/base/docs/recipes/codex-setup.md`
  (byte-identical): the "Running `codex exec` from an agent's Bash tool"
  section (`:84-115`) now describes the read-only sandbox, `--ignore-rules`,
  and the MCP and read limits.
- `tests/test-codex-exec-invocation.sh`: 96 to 144 checks. Test only; not
  touched in this pass.
- `docs/tech-debt/README.md`: the #197 row resolved (strikethrough plus a
  `RESOLVED` comment) and a residual-risk row added (`:149`).

Claims in the shipped text re-checked in this pass (not taken from the earlier
reports):

| Claim | Check | Result |
|-------|-------|--------|
| All four copies of each skill are in lock-step; root and template config and recipe are identical | `./scripts/check-skill-sync.sh`, `./scripts/check-sync.sh`, `cmp` on the pairs | green, byte-identical |
| The call shape in the recipe (`--sandbox read-only` for the advisory, `-c sandbox_mode=read-only` for the reviewer, `--ignore-rules` on both, `exec review` has no `--sandbox`) matches the skill lines | `.claude/skills/cross-review/SKILL.md:58`, `:167`; `.claude/skills/plan/SKILL.md:71` | matches |
| Parsed config is unchanged by the comment edits | `tomllib` load of both `.codex/config.toml` copies against `main:` | equal |
| The approval wording in the config comment is supported | `codex --help` (codex-cli 0.154.0, scratch `HOME`/`CODEX_HOME`): `-a, --ask-for-approval`, "Configure when the model requires human approval before executing a command"; `approval: never` in the run headers recorded in the plan's Deviation notes (AC-3) | supported (see O-3 below) |

### Surfaces checked for drift

Searched `exec review`, `model_reasoning_effort`, `danger-full-access`,
`sandbox`, `read-only`, and `cross-review` across the files below.

| File | Lines | Finding |
|------|-------|---------|
| `docs/recipes/codex-setup.md` (+ template) | `:93-106` | Describes the call shape and the sandbox. Did not say where the read-only `-c` must sit. **Fixed** (see Changes made). |
| `.claude/skills/cross-review/SKILL.md` (+ 3 copies) | `:60` | Same gap in the Step 4 explanation. **Fixed.** The new sentence avoids the string `command codex ` so the invocation test does not read it as a call line. |
| `.claude/skills/plan/SKILL.md` (+ 3 copies) | `:73` | Advisory uses the `--sandbox read-only` flag, which survives a late `-c`, and points to `/cross-review` Step 4 for the limits. No `-c` placement claim. Not changed. |
| `.codex/config.toml` (+ template) | `:19-25` | O-3. **Fixed.** |
| `.codex/config.toml` (+ template) | `:69-74` (`[profiles.review]` comment) | Names `-m`, `-c model_reasoning_effort`, `-c sandbox_mode=read-only`, `--ignore-rules`. Matches the skill. Parsed TOML equal to main. |
| `.codex/README.md` (+ template) | `:19`, `:63-64` | "calls `codex exec review`"; trust note lists `sandbox_mode`. No flag or argument-order claim. |
| `.claude/rules/ralph/post-implementation-pipeline.md` (+ template) | `:24` | "inline; calls `codex exec review`". No sandbox claim. |
| `.claude/rules/ralph/model-routing.md` (+ template, KNOWN_DIFF) | `:52-58`, `:131` | Covers `RALPH_CODEX_REVIEWER_MODEL` and effort fallbacks only; the fallbacks in the skill lines are unchanged (`internal/config/defaults_sync_test.go` green). Not affected. |
| `scripts/ralph-config.sh` (+ template) | `:37-42` | Says the model and effort are passed as `-m` / `-c model_reasoning_effort=`. Still true; says nothing about the sandbox. |
| `README.md` | `:174`, `:224`, `:285-286` | "model, sandbox, approval" for `config.toml`; "calls `codex exec review`". No flag claim. |
| `AGENTS.md` | `:37`, `:73`, `:111` | Pipeline map and file list. No flag claim. |
| `docs/quality/` | none | No hit for any search term. |
| `docs/recipes/codex-seat-permissions.md` (+ template) | `:256` | About org seats and `--sandbox workspace-write`; a different mechanism. Not affected. |
| `docs/specs/2026-05-07-codex-cli-parity.md` | `:16`, `:40`, `:67`, `:134` | Dated spec records of `codex exec review --base ...`; no sandbox claim. History; left as written. |

Drift found: one (the missing `-c` placement sentence in the recipe and in the
`/cross-review` Step 4 explanation). Fixed. Plus the O-3 verify finding.

### Changes made

| File | Change |
|------|--------|
| `docs/recipes/codex-setup.md`, `templates/base/docs/recipes/codex-setup.md` | One sentence added after the sandbox rationale (`:101-103`): "Keep every `-c` before `exec`: on codex-cli 0.154.0 a `-c` placed after `exec` makes codex discard all the `-c` options given before it (the sandbox and the reasoning effort included)." It does not name `tests/test-codex-exec-invocation.sh`, because the test is not scaffolded into downstream projects. |
| `.claude/skills/cross-review/SKILL.md`, `.agents/skills/cross-review/SKILL.md`, `templates/base/.claude/skills/cross-review/SKILL.md`, `templates/base/.agents/skills/cross-review/SKILL.md` | The same sentence added to the `:60` explanation after "(`exec review` has no `--sandbox` flag)", worded for the skill: "... this one and the reasoning effort included." Invocation lines untouched. |
| `.codex/config.toml`, `templates/base/.codex/config.toml` | O-3 comment rephrase (below). Comments only. |
| `docs/tech-debt/README.md` | Residual-risk row (`:149`) extended with item (e) (below). |
| `docs/insights/events/2026-10-04-cross-review-codex-read-only.jsonl` | `sync_docs` event, cycle 1, verdict pass, via `./scripts/insights-append.sh`. |
| `docs/reports/sync-docs-2026-10-04-cross-review-codex-read-only.md` | This report. |

Root/template doc pairs: every file with a template counterpart was edited in
both places and compared with `cmp` (config, recipe, and both skill pairs).
`docs/tech-debt/` has no template counterpart.

#### O-3: `.codex/config.toml` comment

Before (`:20-21`):

> In an interactive session, approval_policy below gates destructive shell calls.

After (`:20-22`):

> approval_policy below sets when an interactive session asks for approval
> before running a command (`codex --help`, --ask-for-approval).

The old sentence said the policy "gates destructive shell calls", which no codex
help or doc checked here supports. The new wording follows the `codex --help`
text for `-a, --ask-for-approval` ("Configure when the model requires human
approval before executing a command"). The sentence after it was kept and now
cites the run header (`approval: never`) that the plan's AC-3 run recorded.
Parsed TOML is equal to `main` for both copies.

#### Tech-debt row: item (e)

The residual-risk row (`:149`, the one about MCP, reads outside the repo, the
reverse claude reviewer, and the doctor probe) keeps its format and gains one
item in each cell:

- Item: (e) `tests/test-codex-exec-invocation.sh` reads one physical line the way
  sh splits it into words, and these edits pass it: a backslash-newline
  continuation (only the first physical line of an invocation is scanned); an
  invocation that does not start a simple command with `command codex ...`
  (subshell, brace group, bare or absolute `codex`, `env`, `sh -c`, `eval`,
  `time`, `if`); a `#` comment that hides flags; a shell expansion that yields
  an option (`$(...)`, backticks, `${}`, brace expansion, `$'..'`, glob). Each
  takes a deliberate edit, and the `#` form also comments out the launch line's
  redirect, watchdog, and `echo "codex rc="`, so the skill visibly stops working.
  A root-level `-c mcp_servers.<name>.command=...` passes every check too and
  starts a program outside the sandbox while the header still reads
  `sandbox: read-only` (same class as (a)); a `-c notify=[...]` passes the same
  way, but whether codex runs it is unverified.
- Impact: such an edit can drop the read-only selector or `--ignore-rules` from
  a codex call while the test stays green.
- Why deferred: (e) is adversarial-only. Closing it means modelling more of sh or
  rejecting the characters the tokenizer does not model; the test already
  catches the realistic regressions (a late `-c`, a dropped or widened sandbox
  word, a missing `--ignore-rules`).
- Trigger: the next change to the test, where (e) can be closed by rejecting `$`,
  backtick, `{`, `(`, `#`, and a trailing backslash on invocation lines (apart
  from `$BASE` and the `${RALPH_...}` defaults), or at least listed in the
  test's header comment.
- Related: `docs/reports/test-2026-10-03-cross-review-codex-read-only.md`
  (G-2 to G-6 and the Slice D addendum).

G-1 (a `-c` after `exec`) and G-7 (the `e` alias) are not in the row: Slices D
and E closed them and the test now fails both.

### Not done in this pass

- The plan's checkboxes and the tech-debt rows' `docs/plans/archive/...` paths
  (verify O-1, O-2) are `/pr`'s job: `scripts/archive-plan.sh` moves the plan
  and does not rewrite references. Until `/pr` archives the plan in this PR,
  the two tech-debt rows that cite `docs/plans/archive/2026-10-03-cross-review-codex-read-only.md`
  point at a path that does not exist yet.
- `ralph doctor --probe-models` still runs `codex exec` without a sandbox
  override (residual-risk (d)); unchanged, out of scope for #197.
- `docs/tech-debt/README.md` has 30 lines that cite `docs/plans/active/`; how many of
  those plans are archived now was not checked. The pattern pre-dates this
  change, and only the rows this task touches were reviewed.

### Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | exit 0. IDENTICAL 159, DRIFTED 0, ROOT_ONLY 0, TEMPLATE_ONLY 11, KNOWN_DIFF 5 (unchanged set) |
| `./scripts/check-skill-sync.sh` | exit 0. 13 skills in lock-step |
| `./scripts/check-template-purity.sh` | exit 0. No meta-repo-specific references in templates |
| `sh tests/test-codex-exec-invocation.sh` | 144/144 (also 144/144 under `dash`), after the skill prose edit |
| `go test ./internal/config/ ./internal/scaffold/` | ok (skill fallbacks and embedded templates) |
| Parsed `.codex/config.toml` vs `main:` (both copies, `tomllib`) | equal |
| `./scripts/insights-append.sh --slug cross-review-codex-read-only --flow standard --phase sync_docs --cycle 1 --verdict pass --source skill` | exit 0; event appended |
| `./scripts/secret-scan.sh --staged` | recorded in the hand-off message (run at commit time) |
| `./scripts/secret-scan-branch.sh --strict` | recorded in the hand-off message (run before push) |

### Files changed in this pass

- `docs/recipes/codex-setup.md`, `templates/base/docs/recipes/codex-setup.md`
- `.claude/skills/cross-review/SKILL.md`, `.agents/skills/cross-review/SKILL.md`,
  `templates/base/.claude/skills/cross-review/SKILL.md`,
  `templates/base/.agents/skills/cross-review/SKILL.md`
- `.codex/config.toml`, `templates/base/.codex/config.toml`
- `docs/tech-debt/README.md`
- `docs/reports/sync-docs-2026-10-04-cross-review-codex-read-only.md` (this report)
- `docs/insights/events/2026-10-04-cross-review-codex-read-only.jsonl`
  (sync_docs, cycle 1, verdict pass, via `./scripts/insights-append.sh`)
