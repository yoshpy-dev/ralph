# Release behavior notes

`ralph upgrade` lists these notes in its report
(`docs/reports/upgrade-<version>-<date>.md`), below the
`Version: <from> → <to>` line: the note of every release after the version
the project upgrades from, up to and including the version it upgrades to,
oldest first. The notes are embedded into the `ralph` binary
(`//go:embed notes/*.md` in `internal/upgrade/behavior_notes.go`), so a
project sees the notes of the binary it upgrades with.

## One file per release

- Name: `<X.Y.Z>.md`, the release's version with no leading `v`
  (`8.2.0.md`; a prerelease such as `8.2.0-rc.1.md` also works). No build
  metadata (`+...`).
- Body: a short Markdown bullet list of what changes in the behavior of the
  scaffold's skills, rules, agent definitions, hooks, scripts, or role
  prompts that an upgrading project should know, such as a step a skill now
  takes or no longer takes, or a hook that now blocks a command.
- No headings: no line may start with `#`, indented or not. The report adds
  `### <X.Y.Z>` above each note.
- A release with no such change has no file.

## When to write one

During `/release`, after the version is decided and before the tag (see
`.claude/skills/release/SKILL.md`). The note goes into `main` through a pull
request, and `/release` then runs again and tags.

## Checks

`go test ./internal/upgrade/` runs `ValidateBehaviorNotes` on the embedded
notes and fails when a `.md` file here other than this README is not named
`<X.Y.Z>.md`, has an empty body, or has a line that starts with `#`. This
README is not a note: `ralph upgrade` reads only files named as a version.
