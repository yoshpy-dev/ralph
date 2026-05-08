# Codex setup recipe

Operate ralph from the OpenAI Codex CLI in a project that was scaffolded with
`ralph init`. Both Claude Code and Codex are first-class drivers — pick the
one you prefer and the standard flow is identical.

## Prerequisites

- Codex CLI **>= 0.128.0** on `$PATH` (`codex --version`).
- A project that has been initialized with a recent `ralph` (`ralph init` or
  `ralph upgrade` produces `.codex/` and `.agents/skills/`).
- Ralph CLI for `ralph doctor` (optional but recommended).

## One-time setup

```sh
cd <project>
codex trust .                  # required: only then are .codex/config.toml
                               # and [hooks] actually loaded
ralph doctor                   # confirms claude/codex CLI presence,
                               # codex_hooks=true, hook entries
```

`ralph doctor` warns if the project is unwritten / untrusted, if
`[features] codex_hooks = true` is missing, or if no `[hooks]` entries are
visible. Resolve every warning before relying on hook-driven safety.

## Daily flow

Inside a `codex` session, kick off the standard flow with skill mentions:

```
$spec    # optional, refines vague requests into docs/specs/<date>-<slug>.md
$plan    # creates docs/plans/active/<date>-<slug>.md
$work    # creates the feature branch and starts implementation
$self-review
$verify
$test
$sync-docs
$cross-review   # optional — calls `claude -p` for a Claude second opinion
$pr
```

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

Codex does not have a `Task(subagent_type=...)` mechanism. The
post-implementation pipeline (`self-review → verify → test → sync-docs`) runs
**sequentially inline in the single agent** — each step writes its own report
to `docs/reports/`. The cycle cap (`RALPH_STANDARD_MAX_PIPELINE_CYCLES`) still
applies, so a fix-and-revalidate run cannot exceed two passes by default.

## Drift safety

`scripts/check-skill-sync.sh` compares `.claude/skills/<name>/SKILL.md` and
`.agents/skills/<name>/SKILL.md` on five axes: inventory, body,
frontmatter `name`, frontmatter `description`, and implicit-invocation
policy (`disable-model-invocation` ⇔ `policy.allow_implicit_invocation`).
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
