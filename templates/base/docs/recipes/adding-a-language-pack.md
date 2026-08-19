# Adding a language pack

A language pack is a self-contained set of language-specific depth for
`ralph`: a `verify.sh` script that runs static analysis and tests for that
language, activation markers so it only runs when the language is actually
present, and a `rule.md` that renders into `.claude/rules/ralph/<lang>.md` so
agents get language-specific guidance without bloating `AGENTS.md`.

## Enabling a pack in this project

List what is available, then add the one you need:

```sh
ralph pack list
ralph pack add <language>
```

`ralph pack add <language>` writes the pack's files under
`packs/languages/<language>/` (verification script, README, activation
markers) and renders its rule content to `.claude/rules/ralph/<language>.md`.
It also records the pack in `.ralph/manifest.toml` so `ralph doctor` and
`ralph upgrade` track it going forward.

`ralph pack add` requires a v2-layout project (`.ralph/manifest.toml` with
`meta.layout = "v2"`). If your project predates that layout, run `ralph
upgrade` first — it performs a one-time, confirmed migration to v2 (preview,
git-clean precondition, `y`/`N` confirmation — `--yes`/`--dry-run` available)
before you can add a pack.

Packs can also be selected up front, at `ralph init` time, if you already
know which languages the project needs.

## How the rule content works

Every pack's `rule.md` lands at `.claude/rules/ralph/<language>.md` alongside
the core ralph rules in the same directory — one file per active language,
scoped to that language's own conventions (verification order, common
pitfalls, naming/structure conventions). This keeps `AGENTS.md` a short map
instead of an encyclopedia: language-specific detail lives in its own rule
file, not folded into the project-wide guidance every agent reads regardless
of what it is touching.

`.claude/rules/ralph/<language>.md` is scaffold-owned content — `ralph
upgrade` keeps it in sync with the pack's current template. If you need to
diverge from a pack's shipped rule content, `ralph eject
.claude/rules/ralph/<language>.md` before editing it, so `ralph upgrade`
stops overwriting your local edits and instead reports it as a fork.

## Verification

Once a pack is installed, `./scripts/run-verify.sh` picks it up
automatically: `scripts/detect-languages.sh` (and, for changed-scope runs,
`scripts/detect-changed-languages.sh`) detect the language from marker files
already present in your project, then run `packs/languages/<language>/verify.sh`
as part of the normal verify pipeline. No extra wiring is required after
`ralph pack add` — verification and the rule content are both active as soon
as the pack's own marker files exist in your project.

## Contributing a new pack

Adding a brand-new language pack (one that is not yet available via `ralph
pack list`) is a `ralph` CLI contributor workflow, not something you do
inside your own scaffolded project — a new pack ships to every project only
once it is merged and embedded in a released `ralph` binary. If your project
needs a language pack that does not exist yet, contribute it in the upstream
`ralph` repository.
