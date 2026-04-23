# VHS tapes

Scripts for [charmbracelet/vhs](https://github.com/charmbracelet/vhs) that generate the demo GIFs referenced from the top-level `README.md`.

## Prerequisites

```sh
brew install vhs ttyd ffmpeg
```

Make sure `ralph` itself is on your `PATH` (`brew install yoshpy-dev/tap/ralph` or `go install ./cmd/ralph`).

## Generate

From the repo root:

```sh
vhs docs/assets/vhs/demo-init.tape       # -> docs/assets/demo-init.gif
vhs docs/assets/vhs/demo-status.tape     # -> docs/assets/demo-status.gif
```

`demo-status.tape` expects to run inside a project with an active Ralph Loop plan. Bootstrap one first:

```sh
cd $(mktemp -d)
ralph init sample-project
cd sample-project
./scripts/new-ralph-plan.sh sample N/A 3
# produce at least one iteration so the TUI has state to render
```

## Conventions

- Width/height and theme match each other across tapes so the GIFs align visually in the README.
- Keep tapes under ~30 seconds so the resulting GIFs stay under a few MB.
- Commit both the `.tape` (source) and the generated `.gif` (output) so GitHub can render the GIF without asking contributors to install VHS.
