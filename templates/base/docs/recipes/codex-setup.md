# Codex setup recipe

Operate ralph from OpenAI Codex in a project that was scaffolded with
`ralph init`. Both Claude Code and Codex are first-class drivers — pick the
one you prefer and the standard flow is identical.

## Prerequisites

- Codex **>= 0.128.0** on `$PATH` (`codex --version`).
- A project that has been initialized with a recent `ralph` (`ralph init` or
  `ralph upgrade` produces `.codex/` and `.agents/skills/`).
- Ralph CLI for `ralph doctor` (optional but recommended).

## One-time setup

```sh
cd <project>
codex trust .                  # required: only then are .codex/config.toml
                               # and [hooks] actually loaded
ralph doctor                   # confirms Claude Code/Codex presence,
                               # hooks=true, hook entries
```

`ralph doctor` warns if the project is unwritten / untrusted, if
`[features] hooks = true` is missing, or if no `[hooks]` entries are
visible. Resolve every warning before relying on hook-driven safety.

## Daily flow

Inside a `codex` session, kick off the standard flow with skill mentions:

```
$spec    # optional; uses an isolated spec worktree for issue/spec outputs
$plan    # ensures a clean-base task worktree, then creates docs/plans/active/<date>-<slug>.md
$work    # resumes the task worktree, implements, runs the post-impl pipeline, then hands off to PR
```

After `$work` completes implementation, the standard pipeline runs
`self-review → verify → test → sync-docs` through Codex custom agents,
then runs `$cross-review` and `$pr` inline. Invoke individual phase skills
manually only when recovering from a failed or interrupted run.

Important: do **not** type `/spec`, `/plan`, etc. `/plan`, `/review`, and
`/status` are Codex built-in slash commands — leading-slash invocation will
trigger the wrong handler. Use `$skill-name` mention syntax or pick from the
`/skills` menu.

## Cross-review pairing

Codex-driven flow uses Claude as the cross-reviewer:

1. `$cross-review` resolves the driver via `RALPH_PRIMARY_CLI=codex` (or
   auto-detects when only `codex` is on `$PATH`).
2. The skill calls `claude -p` with an adversarial reviewer prompt and feeds
   the diff between `origin/$BASE` and `HEAD`.
3. Findings are triaged in the same Codex session and written to
   `docs/reports/cross-review-triage-<slug>.md`. The triage report header
   carries `Driver: codex  Reviewer: claude` so the artifact is
   self-describing.

If the user wants Codex to review a Claude-driven flow, set
`RALPH_PRIMARY_CLI=claude` before running the post-implementation pipeline.

## Subagents

Codex supports subagents and project-scoped custom agents under
`.codex/agents/`. The standard ralph post-implementation pipeline uses those
agents by default: `reviewer` → `verifier` → `tester` → `doc-maintainer`.
Keep that order so phase boundaries, report writes, and the cycle cap stay
predictable.

If Codex cannot dispatch a subagent, run that step inline and record the
fallback in the report. The cycle cap (`RALPH_STANDARD_MAX_PIPELINE_CYCLES`)
still applies, so a fix-and-revalidate run cannot exceed two passes by
default.

## Drift safety

`scripts/check-skill-sync.sh` compares `.claude/skills/<name>/SKILL.md` and
`.agents/skills/<name>/SKILL.md` on six axes: inventory, body,
frontmatter `name`, frontmatter `description`, implicit-invocation
policy (`disable-model-invocation` ⇔ `policy.allow_implicit_invocation`),
and `prompts/` directory parity (every file must exist on both sides and
be byte-identical).
CI fails on drift, so always edit both sides whenever you touch a skill.

## Recovery from a half-applied upgrade

`ralph upgrade` represents the `codex-review` → `cross-review` rename as an
`add` plus a `remove`. If a run is interrupted between the two:

1. Re-run `ralph upgrade` — it is idempotent and will finish the rename.
2. Run `git status` to confirm both `cross-review/` is present and any leftover
   `codex-review/` residue is gone.
3. Run `./scripts/check-skill-sync.sh`. A clean run prints
   `[ok] check-skill-sync: N skill(s) in lock-step`.

If `ralph upgrade` is unavailable or refuses to converge, restore from a
pre-upgrade commit (`git restore --source=<sha> -- .claude .codex .agents`)
and retry once you can run the upgrade end-to-end.
