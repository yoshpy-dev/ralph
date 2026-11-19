# Walkthrough: verify-docs-only-base

- Date: 2026-11-19
- Plan: docs/plans/archive/2026-11-19-verify-docs-only-base.md (#341)
- Author: hiroki-yoshioka

## What changed

In a repository where no language pack, no `scripts/verify.local.sh`, and no
`.ralph/local/` drop-in runs, `scripts/run-verify.sh` used to decide "docs
only" in the full scope from `git diff` and `git diff --cached` alone. CI's
clean checkout has neither, so a pull request that added code passed CI as
docs-only work (exit 0) while the same change failed locally before it was
committed (exit 2).

Now both scopes, locally and in CI, take the decision from
`scripts/detect-changed-languages.sh`, which reads the change set against the
merge-base with the base branch (committed, staged, unstaged, untracked):

- `docs_only=true` (every path is docs or ralph scaffold) exits 0;
- `docs_only=false` exits 2 and names the first code-like path;
- `docs_only=unknown` (the change set could not be read) exits 2 with the
  reason.

Both exit-2 summaries say how to add a verifier for the mode. In CI the
detector reads `GITHUB_BASE_REF` first (as `secret-scan-branch.sh` already
did), so the seed `verify.yml` of existing downstream repositories works
unchanged. `ralph org gate` and `regate` drop the caller's `GITHUB_BASE_REF`
from the scripts' environment, as they drop `RALPH_VERIFY_*`.

## Key files to read first

1. `scripts/detect-changed-languages.sh` header: the output contract
   (`docs_only=true|false|unknown`, `first_non_docs`), the scaffold rule, the
   `diff_failed` rule, and the base order.
2. `scripts/run-verify.sh`: the detector block before the scope branch, and
   the `ran_any=0` summary at the end.
3. `tests/test-run-verify-docs-only.sh`: the CI-like checkout (fetched
   branches, no `origin/HEAD`, a detached PR merge commit, default branch
   `develop`) and the local clone, AC1–AC6, AC11, AC12.
4. `internal/org/gate_run.go`: `gateStrippedEnvNames`.
5. `docs/evidence/verify-docs-only-live-2026-11-19.md`: the same cases on a
   repository scaffolded by this branch's `ralph init`.

## Main control flow

```
run-verify.sh
  ├─ verify.local.sh (if any)
  ├─ detect-changed-languages.sh  → scope, reason, docs_only, first_non_docs, languages
  │     base: GITHUB_BASE_REF → [verify].base → origin default → tracked remote → local main/master
  │     change set: 4 git listings (core.quotePath=false), any failure → diff_failed / unknown
  │     every path: docs table or scaffold (.ralph/manifest.toml, deleted paths also at the merge-base)
  ├─ changed scope: languages from the detector; full scope: languages from detect-languages.sh
  ├─ language verifiers, .ralph/local drop-ins
  ├─ branch secret scan (static/all)
  └─ ran_any=0 → true: exit 0 / false: exit 2 + hint / unknown: exit 2 + reason + hint
```

## Risky code paths

- The detector's loop no longer exits at the first shared or unclassified
  file. It remembers the first reason and stops collecting languages and
  roots from there, so scope, reason, languages, and roots stay as before;
  the rest of the files only feed `docs_only`. The 221 existing checks of
  `tests/test-detect-changed-languages.sh` pin that.
- `unknown` exits 2 like `false`. A pack-less repository whose base cannot be
  read (a shallow CI checkout, a local repository with no origin and no
  main/master, a broken `ralph.local.toml`) now fails even a docs-only change,
  with the reason and the fix in the output. Repositories where a verifier
  runs are not affected: `docs_only` is read only when no verifier ran.
- Scaffold is what `.ralph/manifest.toml` lists. An edited core file still
  counts as scaffold, and adding a path to the manifest makes it pass; the
  header says it is not a security boundary.

## What a human reviewer should pay special attention to

- The release-note impact: downstream pack-less repositories start failing CI
  on code PRs until they add a verifier (the user's decision 1 of the
  series). The candidate text was sent to lane A.
- `GITHUB_BASE_REF` read first by the detector, against the old header line
  "No environment variable chooses the base": the rule was about user
  settings, and `GITHUB_BASE_REF` is a fact CI sets (D1 of the plan).
- The gate change (`gateStrippedEnvNames`): a whole-name match, so
  `GITHUB_BASE_REFX` passes.

## Known limitations

- The real GitHub Actions checkout was not observed; this repository's own CI
  runs the golang pack, so its log does not show the docs-only decision.
- The full scope runs the detector even when a verifier runs (one more
  detector run; a settings error prints the detector's line).
- A rename from a scaffold path to a code path is not tested.
- These and the other leftovers are recorded in `docs/tech-debt/README.md`.
