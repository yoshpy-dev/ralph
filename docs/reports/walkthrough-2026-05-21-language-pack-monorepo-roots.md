# Walkthrough: language-pack-monorepo-roots

- Date: 2026-05-21
- Plan: `docs/plans/archive/2026-05-21-language-pack-monorepo-roots.md`
- Author: Codex

## What changed

- Each shipped language verifier now finds project roots recursively, prunes cache/dependency/build directories, and runs the existing verifier commands from each selected root.
- `scripts/detect-languages.sh` now uses pruned `find -print -quit` detection, detects nested markers such as `go.mod`, and no longer emits `jvm`.
- `scripts/detect-changed-languages.sh` now emits optional `<language>_roots` fields where a changed file can be mapped to a project root.
- `scripts/run-verify.sh` passes those root fields to verifiers through `RALPH_VERIFY_PROJECT_ROOTS`.
- Template copies under `templates/base/scripts/` and `templates/packs/` were kept byte-identical.

## Key files to read first

- `scripts/run-verify.sh`
- `scripts/detect-changed-languages.sh`
- `packs/languages/typescript/verify.sh`
- `packs/languages/golang/verify.sh`
- `tests/test-language-pack-monorepo-roots.sh`
- `tests/test-run-verify-scope.sh`

## Main control flow

1. `run-verify.sh` asks `detect-changed-languages.sh` for changed languages and optional project roots.
2. For each selected language, `run-verify.sh` invokes `packs/languages/<lang>/verify.sh`.
3. If `<lang>_roots` exists, `run-verify.sh` sets `RALPH_VERIFY_PROJECT_ROOTS` for that verifier process.
4. Each verifier discovers all project roots for its language, filters by `RALPH_VERIFY_PROJECT_ROOTS` when present, `cd`s into each root, and runs the existing static/test commands according to `HARNESS_VERIFY_MODE`.

## Risky code paths

- Shell `case` classification for nested marker paths in `detect-changed-languages.sh`.
- Root filtering in each verifier, because it must not accidentally skip all roots during full-scope runs.
- Terraform root semantics: `fmt -check` is now root-local and cache-pruned rather than repository-recursive.

## What a human reviewer should pay special attention to

- Whether the root marker sets are broad enough for each language without creating false positives.
- Whether `RALPH_VERIFY_PROJECT_ROOTS` should eventually support paths containing spaces; current behavior follows the existing space-separated scope contract.
- Whether Terraform per-directory root behavior matches the team's desired module validation model.

## Known limitations

- No new real-tool integration test was added for TypeScript, Python, Rust, Dart, or Terraform; fake tools verify dispatch and cwd behavior.
- Deleted project roots cannot always be narrowed; the detector falls back to language-level selection when no marker can be found on disk.
