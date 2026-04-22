---
name: release
description: Cut a new release tag so `brew upgrade ralph` picks up the latest build. Verifies main/clean state, runs the verify script, picks a semver bump, pushes the tag, monitors the Release workflow, and confirms the Homebrew tap was updated. Manual trigger only. Repo-specific — not distributed via template.
disable-model-invocation: true
allowed-tools: Bash, Read, Grep, AskUserQuestion
---
Cut a release tag for this repository so Homebrew users can `brew upgrade ralph` to the new version.

## How the release pipeline works

1. Pushing a `vX.Y.Z` tag triggers `.github/workflows/release.yml`.
2. The workflow runs `goreleaser release --clean` using `.goreleaser.yml`.
3. goreleaser builds cross-platform archives, creates the GitHub Release, and updates the Homebrew tap at `yoshpy-dev/homebrew-tap`.
4. Once the tap is updated, `brew update && brew upgrade ralph` distributes the new build.

This skill automates steps around pushing the tag — it does not mutate `.goreleaser.yml` or the workflow.

## Pre-checks

Stop and report which check failed if any of these are false:

1. `gh` CLI is authenticated (`gh auth status`).
2. Current branch is `main`.
3. Working tree is clean (`git status --porcelain` is empty).
4. Local `main` is up to date with `origin/main` (`git fetch origin main` then compare `HEAD` with `origin/main`).
5. No uncommitted `v*` tags exist locally that are missing on remote (`git push --tags --dry-run`).

## Steps

1. **Discover current version.** Run `git tag --sort=-v:refname | head -1` to find the latest `vX.Y.Z`. If no tag exists, start from `v0.1.0`.
2. **Run the quality gate.** Run `./scripts/run-verify.sh`. If it fails, stop and surface the error — do not proceed to tagging.
3. **Preview the changeset.** Run `git log <latest-tag>..HEAD --oneline` and show it to the user. This is the raw material goreleaser will turn into release notes (filtered by `.goreleaser.yml` `changelog.filters`).
4. **Select the version bump.** Use `AskUserQuestion` with three options:
   - `patch` — bug fixes, docs, chore (default)
   - `minor` — new backwards-compatible features
   - `major` — breaking changes
   Compute the next version from the latest tag. Confirm the computed `vX.Y.Z` with the user before tagging.
5. **Create and push the tag.**
   - `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
   - `git push origin vX.Y.Z`
6. **Monitor the Release workflow.**
   - Find the run: `gh run list --workflow=release.yml --limit 1 --json databaseId,status,conclusion,headBranch`.
   - Watch it: `gh run watch <id> --exit-status`. If the workflow fails, surface the logs (`gh run view <id> --log-failed`) and stop. The tag remains on origin; the user must decide whether to delete it (`git push --delete origin vX.Y.Z` + `git tag -d vX.Y.Z`) or re-run the workflow.
7. **Verify the GitHub Release.** `gh release view vX.Y.Z` — confirm all four archives exist (`darwin_amd64`, `darwin_arm64`, `linux_amd64`, `linux_arm64`) plus `checksums.txt`.
8. **Verify the Homebrew tap update.** Fetch the tap Formula and confirm the version bumped:
   - `gh api repos/yoshpy-dev/homebrew-tap/contents/Formula/ralph.rb --jq '.content' | base64 -d | grep -E 'version|url'`
   - The `version` line must match the new `vX.Y.Z` (without the leading `v`).
   - If the Formula does not yet reflect the new version, wait ~30s and retry once; goreleaser pushes to the tap as the final step.
9. **Report completion.** Show the user:
   - Release URL (`gh release view vX.Y.Z --json url --jq '.url'`)
   - The command to install the new build: `brew update && brew upgrade ralph` (or `brew install yoshpy-dev/tap/ralph` for fresh installs)

## Completion gate

Do NOT declare the release complete until ALL of the following are true:

- [ ] Tag `vX.Y.Z` exists on `origin`
- [ ] Release workflow run finished with `conclusion: success`
- [ ] GitHub Release `vX.Y.Z` has all 5 assets (4 archives + checksums.txt)
- [ ] Homebrew tap Formula `version` matches `X.Y.Z`
- [ ] User was shown the `brew upgrade` command

## Failure recovery

- **Verify script fails before tagging.** No cleanup needed. Fix the issue on `main` first.
- **Workflow fails after tag push.** Decide with the user:
  - Re-run: `gh run rerun <id>` (keeps the same tag).
  - Abandon: delete the remote and local tag, then fix and retry with a new patch version. Do **not** overwrite a pushed tag — goreleaser and Homebrew consumers treat tags as immutable.
- **Homebrew tap not updated but GitHub Release succeeded.** Check `HOMEBREW_TAP_GITHUB_TOKEN` secret in the repo settings. Manually re-run the workflow after fixing.
