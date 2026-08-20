# Cycle 3 live-fire evidence: codex-hooks-json-wiring (2026-08-20, codex-cli 0.147.0)

Re-verification for cross-review cycle 2's two ACTION_REQUIRED findings
(`docs/reports/cross-review-triage-codex-hooks-json-wiring.md`):

- AR#1: dispatcher must run from the repo root even when the Codex session
  is launched from a subdirectory.
- AR#2: `post_edit_verify.sh` / `check_mojibake.sh` must derive edited paths
  from `apply_patch` payloads (`tool_input.command`), not just
  `tool_input.file_path`.

## Fixture setup

- `go build -o <tmp>/ralph ./cmd/ralph` from this worktree, then
  `<tmp>/ralph init --yes <tmp>/proj` (fresh git repo, `git init` + `ralph
  init` — ships the fixed `.codex/hooks.json` and the updated
  `post_edit_verify.sh`/`check_mojibake.sh` via the manifest).
- Probe drop-in added at `<tmp>/proj/.claude/hooks/local/PostToolUse.d/99-probe.sh`
  (third dispatcher layer): appends the raw stdin payload to
  `.harness/state/probe-payloads.log` and exits 0. Committed so the fixture
  has a clean tree (`git commit -m "chore: add probe drop-in for live-fire
  evidence"`).
- Shipped `.codex/hooks.json` command under test (unchanged from the fixed
  root/template copies):
  ```
  cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh PostToolUse
  ```

## Invocation

Run from **`<tmp>/proj/docs`** — a subdirectory of the fixture's git root,
not the root itself — to test the AR#1 fix under the condition that
previously caused a silent no-op:

```
cd <tmp>/proj/docs
codex exec --skip-git-repo-check --dangerously-bypass-hook-trust \
  "Create a new file named livefire_probe.txt in the current working \
   directory with the exact single line of text: hello from cycle3 livefire"
```

(`--dangerously-bypass-hook-trust` supplements the interactive hook-trust
approval gate for this fresh, non-interactive fixture — consistent with the
Slice 1 evidence file's AC-2(b) methodology; it does not touch the AR#1/AR#2
mechanics under test, which are about dispatcher cwd resolution and payload
parsing, not the trust gate itself.)

Codex used its `apply_patch` tool (not `Edit`/`Write`) to create the file,
confirmed by the transcript's `tool_name: apply_patch` and the
`*** Begin Patch / *** Add File: livefire_probe.txt / *** End Patch` body —
so this run exercises both AR#1 (subdirectory launch) and AR#2 (apply_patch
payload shape) in a single live invocation.

## Result

- `hook: PostToolUse` / `hook: PostToolUse Completed` appeared multiple
  times in the exec transcript (three groups, once per PostToolUse-firing
  Codex action during the run).
- **AR#1 proof (subdirectory-cwd dispatch):** `.harness/state/` was created
  and populated at `<tmp>/proj/.harness/state/` — the fixture's **git
  root** — even though the session's cwd was `<tmp>/proj/docs`. No
  `<tmp>/proj/docs/.harness/` directory was created (confirmed absent). If
  the pre-fix command form (absolute dispatcher path, no leading `cd`) were
  still in place, `ralph-dispatch.sh` would have resolved its `./.claude/hooks/PostToolUse.d/`
  etc. layers relative to `docs/`, found nothing, and run zero scripts —
  the probe would not have fired and no state directory would exist under
  the repo root.
- `<tmp>/proj/.harness/state/probe-payloads.log` contains the raw
  PostToolUse payload the third dispatcher layer received, proving the
  full three-layer fan-out (`./.claude/hooks/PostToolUse.d/` core →
  `./.ralph/local/hooks/PostToolUse.d/` → `./.claude/hooks/local/PostToolUse.d/`)
  reached the probe drop-in. Payload excerpt:
  ```json
  {"tool_name":"apply_patch","tool_input":{"command":"*** Begin Patch\n*** Add File: livefire_probe.txt\n+hello from cycle3 livefire\n*** End Patch"}, ...}
  ```
  `tool_input` has no `file_path` key (`has file_path key: False`,
  confirmed via a JSON parse of the captured payload) — this is a genuine
  `apply_patch`-shaped payload, not the `Edit`/`Write` shape.
- **AR#2 proof (apply_patch path derivation):**
  `<tmp>/proj/.harness/state/edited-files.log` contains exactly one line:
  `livefire_probe.txt` — the path derived from the payload's
  `*** Add File: livefire_probe.txt` envelope line by
  `post_edit_verify.sh`'s new jq-based patch-body parser, since
  `tool_input.file_path` was absent. Before this fix, this branch had no
  fallback and `post_edit_verify.sh` would have logged nothing for this
  payload.
- The target file `<tmp>/proj/docs/livefire_probe.txt` was created with the
  exact requested content (`hello from cycle3 livefire`), confirming the
  edit itself succeeded independent of the hook wiring under test.

## Cleanup

The fixture directory (`<tmp>` under the session scratchpad) was removed
after evidence capture; no fixture artifacts were committed to this repo.
