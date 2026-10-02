# sync-docs report: codex-config-model-gpt-5-6-sol

## Cycle 1

- Date: 2026-10-02
- Plan: `docs/plans/active/2026-10-02-codex-config-model-gpt-5-6-sol.md` (refs #156)
- Pipeline cycle: 1 of 2. Delta: `main` `47345820` to branch HEAD `c3b65ccc`.
- Prior reports (the commits that added them): `docs/reports/self-review-2026-10-02-codex-config-model-gpt-5-6-sol.md`
  (`f7cb8c0c`, addendum `dbd3002b`; merge ok, M-1 / M-2 deferred to this step),
  `docs/reports/verify-2026-10-02-codex-config-model-gpt-5-6-sol.md` (`57c24384`; Pass, D-3 and D-4 deferred to this step),
  `docs/reports/test-2026-10-02-codex-config-model-gpt-5-6-sol.md` (`a695e85a`; Pass, G-1 closed by Slice C `21291931`)

### Summary

The documentation that shipped with the change (the rewritten comments in the
two `.codex/config.toml` copies, the spec notes in F-6, the `:180` open-questions
line, and (e) of the org-runtime spec) matches the code at HEAD. No other doc
surface states the project's codex model, its reasoning effort, or that project
profiles take effect, so no drift edit was needed outside the spec note the
verifier asked for (D-4). This pass adds one spec note and three tech-debt
rows. No plan, other report, code, config, test, workflow, `.gitallowed`, or
scanner file was touched.

### Delta reviewed

`git diff main...HEAD` excluding `docs/plans`, `docs/reports`, `docs/insights`:

- `.codex/config.toml`, `templates/base/.codex/config.toml` (byte-identical):
  `model` `gpt-5.5` to `gpt-5.6-sol` in three places; the top comment now says
  the Codex-side pipeline agents (`.codex/agents/`, no `model` set) inherit the
  model and that no effort is set (user-level setting, else the model default,
  low for `gpt-5.6-sol` as of 2026-10-02); the Profiles block says codex
  ignores project-local `[profiles.*]` (codex-cli 0.154.0 and 0.159.2), that the
  tables are examples for the user-level config, that `--profile review` here is
  not read-only, and that `/cross-review` does not use `--profile`.
- `docs/specs/2026-05-07-codex-cli-parity.md`: F-6 (`:45`) and the open-questions
  line (`:180`) carry "2026-10-02 に `gpt-5.6-sol` へ変更。#156".
- `docs/specs/2026-08-01-org-runtime.md`: (e) records that its last sentence is
  as of #196 and that the setting changed on 2026-10-02.
- `internal/scaffold/embed_test.go`: new `TestTemplateBaseCodexTomlFilesParse`.
  Test only; not touched in this pass.

Claims in the shipped text re-checked in this pass (not taken from the earlier
reports):

| Claim | Check | Result |
|-------|-------|--------|
| Root and template `.codex/config.toml` are identical | `./scripts/check-sync.sh` | IDENTICAL |
| `.codex/agents/*.toml` set no `model` | `git grep -n -w model -- .codex/agents templates/base/.codex/agents` | 0 hits |
| `/cross-review` passes no `--profile` and no sandbox flag; `/plan` advisory passes `--sandbox read-only` | `.claude/skills/cross-review/SKILL.md:58`, `:167`; `.claude/skills/plan/SKILL.md:71`; `.agents/` and `templates/base/` copies each contain the `exec review` call twice | matches |
| The Profiles comment's tables and keys are unchanged from main | `git diff main...HEAD -- .codex/config.toml` | only `model` values and comment lines changed |
| `templates/base/.codex/agents/implementer.toml` exists but is not in `TestTemplateBaseCodexAssetsExist` | `internal/scaffold/embed_test.go:216-226` (`required` list), `find templates/base/.codex/agents` | confirmed |

### Surfaces checked for drift

Searched `gpt-5.5`, `gpt-5.6`, `profile`, `--profile`, `reasoning.effort`,
`model_reasoning`, and `config.toml` across `README.md`, `AGENTS.md`, `CLAUDE.md`,
`docs/recipes/` (+ `templates/base/docs/recipes/`), `docs/quality/`,
`docs/specs/`, `.codex/` (+ `templates/base/.codex/`), `.claude/rules/` (+ template),
and the skills. Only the lines below touch the project codex model, its effort,
or project profiles.

| File | Lines | Finding |
|------|-------|---------|
| `docs/recipes/codex-setup.md` (+ template) | `:85-100` | Says `codex exec` calls pass explicit `-m` and `-c model_reasoning_effort=` so they do not depend on the project's model. Still accurate; `/cross-review` and `/plan` do exactly that. |
| `docs/recipes/codex-setup.md` (+ template) | `:127-175` | The `[profiles.*]` mention is about the user-level config that the `everything-claude-code` plugin edits, and the #185 rewrite recovery. Neither claims project profiles apply. Still accurate. |
| `docs/recipes/codex-seat-permissions.md` (+ template) | `:66` | "reads only the user-level `config.toml`; profiles, project-level config, and `-c` overrides are not evaluated". Consistent with codex dropping project profiles. |
| `.codex/README.md` (+ template) | `:74` | "no per-agent model is pinned ... Codex runs follow the session/config model". Consistent with the new top comment. No profile or effort statement. |
| `.codex/AGENTS.override.md` (+ template) | `:7-30` | Execution-model bullets name no model, effort, or profile. |
| `.claude/rules/ralph/model-routing.md` (+ template, KNOWN_DIFF) | `:54-60`, `:100-130` | Covers Claude tiers, `RALPH_CODEX_REVIEWER_MODEL` (`gpt-6-astra`) and the org `model_pool`; says nothing about the project codex model. Not affected by this change. |
| `README.md` | `:174`, `:272`, `:287` | Describes `config.toml` as "model, sandbox, approval" and the trust requirement. No slug, effort, or profile claim. |
| `AGENTS.md` | `:108` | Lists `config.toml` among `.codex/` files; no model claim. |
| `docs/quality/` | none | No hit for any search term. |
| `docs/specs/2026-05-07-codex-cli-parity.md` | `:45`, `:47`, `:180` | F-6 already annotated; "profiles 定義例" stays accurate because the tables are kept as examples. `:180` already annotated. |
| `docs/specs/2026-05-07-codex-cli-parity.md` | `:77` | User story 4 expects project profiles to switch flows. Contradicts the observed codex behavior. **Fixed** (see Changes made). |
| `docs/evidence/*` | various | Dated records of runs on `gpt-5.5`. History; left as written. |

Remaining `git grep -n 'gpt-5\.5'` hits outside `docs/reports`, `docs/plans`,
`docs/insights` are the spec/evidence records above, `docs/tech-debt/README.md`
(the earlier default-pool row), `*_test.go` fixtures and mocks, and the shell
fixtures in `scripts/verify.local.sh`, `tests/test-hook-wiring.sh`,
`tests/test-ralph-worktree.sh`. None describes the project's codex default.

Drift found: one (user story 4). Fixed.

### Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-05-07-codex-cli-parity.md` | Dated note appended to user story 4 (`:77`). Text below. The story itself is not rewritten. |
| `docs/tech-debt/README.md` | Three rows appended (lines 146-148). Text below. |
| `docs/insights/events/2026-10-02-codex-config-model-gpt-5-6-sol.jsonl` | `sync_docs` event, cycle 1, verdict pass, via `./scripts/insights-append.sh`. |
| `docs/reports/sync-docs-2026-10-02-codex-config-model-gpt-5-6-sol.md` | This report. |

Root/template doc pairs: no file with a root/template counterpart was edited in
this pass. `docs/specs/` and `docs/tech-debt/` have no template counterpart.

#### Spec note (verify D-4)

Appended to `docs/specs/2026-05-07-codex-cli-parity.md:77`, after the unchanged
user-story sentence:

> (2026-10-02 注記: codex は project の `.codex/config.toml` の `[profiles.*]` を読まずに捨てる(codex-cli 0.154.0 と 0.159.2 で確認)。この要望は project の profile だけでは満たせない。経緯は `docs/tech-debt/README.md` の該当行)

#### Tech-debt rows added

The file's five columns are: debt item, impact, why deferred, trigger to pay
down, related plan/report. All three are pre-existing issues that this task
surfaced.

1. **codex ignores project-local `[profiles.*]`** (self-review M-1, verify D-3).
   - Item: codex prints `Ignored unsupported project-local config keys ...:
     profiles` (codex-cli 0.154.0 and 0.159.2), so `[profiles.work]` and
     `[profiles.review]` in both `.codex/config.toml` copies never apply; they
     work only if copied into the user-level `config.toml`. The comment now says
     so and the tables stay as examples. Open decision: delete the tables, or
     keep them as examples.
   - Impact: `codex --profile review` here is not read-only (it runs under
     `danger-full-access`, probe P3) and an undefined profile name is not an
     error (probe P2). The tables ship to every scaffolded project, and spec F-6
     and user story 4 still read as if project profiles switched flows (story 4
     now carries the dated note).
   - Why deferred: this change only edits the model value and comments;
     deleting the tables changes a core file that `ralph upgrade` replaces
     downstream.
   - Trigger: the next change to `.codex/config.toml`, a codex release that
     reads project-local profiles (then the comment must change), or someone
     relying on `--profile review` being read-only.
   - Related: self-review M-1 and probes P1-P4, verify D-3 and D-4, the plan's
     Deviation notes.
2. **`/cross-review`'s codex reviewer runs with `danger-full-access` and
   `approval: never`** (self-review addendum M-2). `codex exec review` has no
   `--sandbox` flag and the skill passes no `-c sandbox_mode=...`
   (`.claude/skills/cross-review/SKILL.md:58`, `:167`, four skill copies);
   `/plan`'s advisory passes `--sandbox read-only` (`plan/SKILL.md:71`). Fix
   tracker: issue #197 (https://github.com/yoshpy-dev/ralph/issues/197).
   Impact: if the reviewer's codex follows an instruction inside the diff under
   review, it can run commands without approval. Probe P6 showed
   `-c sandbox_mode=read-only` gives `sandbox: read-only`; whether `-o` is still
   written and the review completes under read-only is unverified (probes ran
   without credentials). Trigger: issue #197, before the next change to the
   codex path of `/cross-review`.
3. **`TestTemplateBaseCodexAssetsExist` does not list
   `templates/base/.codex/agents/implementer.toml`** (Slice C implementer's
   out-of-scope note). The new parse test globs `agents/*.toml`, so the file is
   parsed but not asserted to exist. A one-sided delete of the template copy is
   caught by `check-sync.sh` (ROOT_ONLY); deleting both copies is not caught by
   either Go test. Trigger: the next touch to that test; add
   `.codex/agents/implementer.toml` to its `required` list.

Reference path note: rows 1 and 3 cite the plan as
`docs/plans/archive/2026-10-02-codex-config-model-gpt-5-6-sol.md`, not
`docs/plans/active/...`, because `/pr` archives the plan in this same PR
(`scripts/archive-plan.sh` does not rewrite references). If the plan is not
archived before merge, those two paths dangle and need repointing.

### Not done in this pass

- Self-review L-3 (effort drops to `low`; `chore:` commits are left out of the
  changelog, so a line on #186 is needed) is an orchestrator action after merge
  and is not a doc edit.
- The PR body still needs the effort and `--profile review` statements
  (self-review follow-up 3). That is `/pr`'s job.
- No doc states a default effort for the project codex, so there was nothing
  to correct for the `medium` to `low` change beyond the config comment.

### Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | exit 0. IDENTICAL 159, DRIFTED 0, ROOT_ONLY 0, TEMPLATE_ONLY 11, KNOWN_DIFF 5 (unchanged set) |
| `./scripts/check-skill-sync.sh` | exit 0. 13 skills in lock-step |
| `./scripts/check-template-purity.sh` | exit 0. No meta-repo-specific references in templates |
| `./scripts/insights-append.sh --slug codex-config-model-gpt-5-6-sol --flow standard --phase sync_docs --cycle 1 --verdict pass --source skill` | exit 0; event appended |
| `./scripts/secret-scan.sh --staged` | recorded in the hand-off message (run at commit time) |
| `./scripts/secret-scan-branch.sh --strict` | recorded in the hand-off message (run before push) |

### Files changed in this pass

- `docs/specs/2026-05-07-codex-cli-parity.md`
- `docs/tech-debt/README.md`
- `docs/reports/sync-docs-2026-10-02-codex-config-model-gpt-5-6-sol.md` (this report)
- `docs/insights/events/2026-10-02-codex-config-model-gpt-5-6-sol.jsonl`
  (sync_docs, cycle 1, verdict pass, via `./scripts/insights-append.sh`)
