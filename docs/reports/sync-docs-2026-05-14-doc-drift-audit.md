# Sync-docs report: doc-drift-audit

- Date: 2026-05-14
- Scope: repository documentation drift audit across root docs, rules, skills, quality gates, templates, language packs, and report links
- Driver: Codex single-agent inline execution

## Drift found and fixed

| Area | Drift | Fix |
| --- | --- | --- |
| `README.md` language-pack inventory | Terraform pack existed in `packs/languages/terraform/` but two README feature rows still listed only TypeScript, Python, Rust, Go, and Dart. | Added Terraform to both rows. |
| `README.md` evidence wording | Feature row still said "codex pass" instead of the current `cross-review` terminology and omitted sync-docs/cross-review artifacts. | Reworded to self-review, verify, test, sync-docs, and cross-review triage artifacts. |
| Cross-review terminology | Current bidirectional flow can use Claude or Codex as reviewer, but docs still said `Codex ACTION_REQUIRED`, `Codex triage`, or `Codex findings` for cross-review gates. | Generalized current workflow docs to `cross-review ACTION_REQUIRED`, `cross-review triage`, and reviewer-CLI availability. |
| Reports index | `docs/reports/README.md` listed self-review, verify, test, and walkthrough reports but not sync-docs or cross-review triage. | Added both report classes and naming patterns; mirrored to `templates/base/`. |
| Repo map | `docs/architecture/repo-map.md` report and script inventory lagged current files. | Added sync-docs/cross-review report types and expanded script groups to include worktree, local verifier, secret guards, and CLI driver. |
| Historical report links | Two report links pointed at local sibling `template.md`, and one Terraform report link pointed at the now-archived plan through `docs/plans/active/`. | Repointed the links to real current paths. |

## Checks run

| Check | Result |
| --- | --- |
| `git diff --check` | PASS |
| `./scripts/check-skill-sync.sh` | PASS (`13 skill(s) in lock-step`) |
| `./scripts/check-sync.sh` | PASS (`DRIFTED: 0`, `ROOT_ONLY: 0`) |
| `./scripts/check-pipeline-sync.sh` | PASS |
| `CI=true ./scripts/check-template.sh` | PASS |
| Markdown relative-link scan | PASS (0 broken links after excluding code spans) |
| Current-doc script reference scan | PASS (216 tracked current docs, 32 script references, 0 missing scripts) |
| `docs/architecture/repo-map.md` script inventory scan | PASS (37 scripts on disk, 37 referenced, 0 missing) |
| Language pack/rule/verifier scan | PASS (`dart`, `golang`, `python`, `rust`, `terraform`, `typescript` all have rules and executable non-placeholder verifiers) |
| Loop prompt mirror scan | PASS (`.claude/skills/loop/prompts/` and `.agents/skills/loop/prompts/` match) |
| Stale terminology scan | PASS for tracked current workflow docs. Remaining `codex-review` hits are rename-history/spec context, and remaining `Codex findings` hits are plan-advisory context. |

## Remaining notes

- Plain `./scripts/check-template.sh` was not used as the final structural gate because this checkout does not have the local git secret hooks installed. Its CI-equivalent path passed with `CI=true`.
- No active plans exist under `docs/plans/active/`; only `.gitkeep` is present.
