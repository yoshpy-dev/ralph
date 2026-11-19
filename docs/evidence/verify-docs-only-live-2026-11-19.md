# Live check of #341: a pack-less repository scaffolded by `ralph init`

- Date: 2026-11-19
- Plan: docs/plans/active/2026-11-19-verify-docs-only-base.md
- Branch head checked: 09e63c43 (ralph built from it with `go build ./cmd/ralph`)
- git: 2.49.0

## Setup

A bare remote whose default branch is `develop` (no main, no master). Its
first commit holds `index.html`, `README.md`, and `data.json` only, so no
language pack applies. Branch `scaffold` adds everything `ralph init --yes .`
writes (scripts, `.claude/`, `.ralph/manifest.toml`, `.gitignore`, ...) in
one commit; `develop` then moves to it. Branch `code` adds `app.js`; branch
`docs` edits `README.md`.

- CI-like checkout: `git init`, `git fetch origin '+refs/heads/*:refs/remotes/origin/*'`,
  `git symbolic-ref --delete refs/remotes/origin/HEAD` (git 2.49 writes it on
  fetch; actions/checkout is assumed not to), detach at the base and
  `git merge --no-ff` the branch, then `./scripts/run-verify.sh` (the
  template `verify.yml` runs it with no scope, so full) with
  `GITHUB_BASE_REF=<base>`. The scaffold PR's base is `develop` before the
  scaffold commit.
- Local clone: `git clone`, check out the branch, clean tree, no
  `GITHUB_BASE_REF`, `RALPH_VERIFY_SCOPE=full` and `=changed`; the base is
  the automatic one, `origin/develop` through the clone's `origin/HEAD`.
  Before the scaffold's clone, the remote's `develop` is moved back to the
  commit before the scaffold, so that clone diffs the scaffold commit, not
  nothing (`no_changes`).
- Re-run by the verifier at d55e9cb1 (after the self-review fixes): the
  same ten results.
- The scratch repositories run with `core.hooksPath` pointed at an empty
  directory (a throwaway fixture; no commit of this repository is affected).

## Result

```
scaffold  ci(with GITHUB_BASE_REF) full    rc=0  No language verifier ran. The changes are docs or ralph scaffold files only.
code      ci(with GITHUB_BASE_REF) full    rc=2  No verifier ran for code-like changes (for example: app.js).
docs      ci(with GITHUB_BASE_REF) full    rc=0  No language verifier ran. The changes are docs or ralph scaffold files only.
docs      ci(without GITHUB_BASE_REF) full    rc=2  No verifier ran, and the changes could not be read (no_diff_base), so they are not taken as docs only. The changes are read against GITHUB_BASE_REF
code      local                    full    rc=2  No verifier ran for code-like changes (for example: app.js).
code      local                    changed rc=2  No verifier ran for code-like changes (for example: app.js).
docs      local                    full    rc=0  No language verifier ran. The changes are docs or ralph scaffold files only.
docs      local                    changed rc=0  No language verifier ran. The changes are docs or ralph scaffold files only.
scaffold  local                    full    rc=0  No language verifier ran. The changes are docs or ralph scaffold files only.
scaffold  local                    changed rc=0  No language verifier ran. The changes are docs or ralph scaffold files only.
```

The line for CI without `GITHUB_BASE_REF` ends: `The changes are read against
GITHUB_BASE_REF in CI (fetch the base branch, as actions/checkout does with
fetch-depth: 0), else [verify].base of ralph.local.toml, else the default
branch.` It shows why the detector reads `GITHUB_BASE_REF` first: with no
`origin/HEAD`, no `main` or `master`, and a detached HEAD, no automatic base
exists.

Before this branch, issue #341 reported exit 0 for the CI run of a commit
that adds code to such a repository.
