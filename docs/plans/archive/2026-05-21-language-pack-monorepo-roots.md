# Language pack monorepo roots

- Status: Implemented
- Owner: Codex
- Date: 2026-05-21
- Related request: Make shipped language pack verifiers execute inside nested monorepo project roots.
- Related issue: #110
- Type: fix
- Branch: fix/110-language-pack-monorepo-roots

## Objective

Update shipped language pack verifiers so monorepo repositories with nested TypeScript, Python, Rust, Go, Dart, or Terraform project roots are verified root-by-root instead of being skipped because the repository root lacks that language's marker file.

## Scope

- `packs/languages/{typescript,python,rust,golang,dart,terraform}/verify.sh`
- Matching `templates/packs/*/verify.sh` mirror files
- `scripts/detect-languages.sh` and the template copy
- Regression tests for nested project root detection and no-`jvm` detection output

## Non-goals

- Adding a JVM language pack
- Installing missing language toolchains automatically
- Implementing changed-scope project-root filtering in CI end-to-end beyond preserving a root-aware verifier structure that can be narrowed later

## Assumptions

- A detected project root with a required tool missing should fail closed; absence of project roots should remain a skip.
- Root enumeration must prune dependency/cache/build directories such as `.git`, `node_modules`, `.terraform`, `.dart_tool`, `target`, `dist`, and `build`.
- Single-project repositories should continue to behave as before, with commands executed from `.`.

## Affected areas

- Language pack verifier command dispatch
- Language detection output
- Template pack synchronization checks
- Shell regression tests

## Design decisions

Critical forks: None

- Each verifier owns its marker-to-root logic rather than introducing a shared sourced helper. The pack files are copied independently into target repositories, and keeping each pack self-contained matches the existing distribution model.
- Terraform keeps repository-wide format checking semantics per root using the selected IaC CLI, while validate/test/tflint/security scans run from each root. Cache directories are pruned when finding roots and tests.

## Acceptance criteria

- [x] TypeScript, Python, Rust, Go, Dart, and Terraform verifiers execute from nested project roots.
- [x] Existing single-root behavior is preserved.
- [x] `scripts/detect-languages.sh` no longer emits `jvm`.
- [x] Root marker discovery prunes cache/build/dependency directories.
- [x] Pack changes are mirrored byte-for-byte under `templates/packs/`.
- [x] Format checks remain non-mutating in CI paths.

## Implementation outline

1. Add root discovery and root-loop execution to each shipped verifier.
2. Update Terraform source/test discovery to avoid `.terraform/` and other cache directories.
3. Remove `jvm` emission from language detection.
4. Add shell regression coverage for nested roots and detection parity.
5. Mirror files into templates and run sync gates.

## Verify plan

- Static analysis checks: `sh -n` on edited shell scripts, `./scripts/check-sync.sh`, `./scripts/run-static-verify.sh`.
- Spec compliance criteria to confirm: root loops execute nested projects, skip/fail-closed semantics are retained, `jvm` is not emitted.
- Documentation drift to check: language pack docs and quality docs for changed contracts.
- Evidence to capture: verify report and command output from targeted shell tests.

## Test plan

- Unit tests: N/A for shell packs.
- Integration tests: add/extend shell tests with fake toolchains to assert command dispatch from nested roots.
- Regression tests: `tests/test-verify-mode-split.sh`, `tests/test-terraform-pack-verify.sh`, `tests/test-detect-languages-terraform.sh`, `tests/test-detect-changed-languages.sh`, and aggregate wrappers.
- Edge cases: no markers, nested markers only, multiple roots, pruned cache directories, missing required tool after root detection, `jvm` markers present.
- Evidence to capture: `docs/reports/{self-review,verify,test,sync-docs}-2026-05-21-language-pack-monorepo-roots.md`.

## Risks and mitigations

- Risk: `find` expressions become non-portable across POSIX shells. Mitigation: use POSIX `find` constructs and test with `/bin/sh`.
- Risk: multiple nested roots cause duplicate command runs. Mitigation: only root markers define roots and use exact directory entries.
- Risk: Terraform root semantics differ from module semantics. Mitigation: run root-level validate/test only where Terraform markers exist, preserve explicit skip for uninitialized roots.

## Rollout or rollback notes

This is distributed through the normal pack/template update path. Rollback is reverting the verifier and detection script changes plus their tests.

## Open questions

None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
