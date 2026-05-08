---
name: pr
description: Create a pull request after self-review, verify, and test pass. Handles branch push, PR creation, plan archiving, and hand-off. Invoke automatically after /cross-review completes (or is skipped), when self-review, verify, and test reports all exist with passing verdicts.
allowed-tools: Read, Grep, Glob, Bash, Write
---
Create a PR to hand off completed work for human review and merge.

## Pre-checks

Before creating a PR, confirm ALL of the following:

1. A self-review report exists in `docs/reports/` with no CRITICAL findings.
2. A verify report exists in `docs/reports/` with pass or partial-pass verdict.
3. A test report exists in `docs/reports/` with pass verdict. **If tests failed, do NOT create the PR.**
4. Raw evidence is saved in `docs/evidence/`.
5. Branch name follows `<type>/<issue>/<slug>` or `<type>/<slug>` format.
6. You are NOT on main or master.
7. `gh` CLI is available (if not, provide manual commands instead).

If any pre-check fails, stop and explain what is missing.

## Steps

1. **Resolve pinned plan identity** (standard flow):
   Read `.harness/state/standard-pipeline/active-plan.json` to obtain the exact plan path persisted by `/work`. Use this path for archival in Step 6 instead of rescanning `docs/plans/active/`. If the file is absent (e.g. Ralph Loop or legacy session), fall back to the single file under `docs/plans/active/` or ask the user which plan to archive.
2. Check for uncommitted changes with `git status --porcelain`.
   - **If uncommitted changes exist**: Stage with `git add` (prefer specific files over `-A`) and create a conventional commit: `<type>: <description>`. If a GitHub issue is linked, append `Refs #<number>` to the commit body.
   - **If working tree is clean** (intermediate commits already exist): Skip staging and committing — proceed directly to push.
3. Push the branch: `git push -u origin HEAD`.
4. Create the PR with `gh pr create` using [template.md](template.md) for the body structure. **PR title and body must be written in Japanese.**
5. For large diffs (>500 changed lines), create a walkthrough in `docs/reports/walkthrough-<date>-<slug>.md`.
6. Archive the plan using the path resolved in Step 1: `./scripts/archive-plan.sh <absolute-plan-path>`.
7. **Clear standard-pipeline state** (on successful PR creation):
   `rm -f .harness/state/standard-pipeline/active-plan.json .harness/state/standard-pipeline/cycle-count.json`.
   If PR creation fails, leave the state files in place so the user can resume.

## Completion gate

Do NOT present the PR as complete unless ALL of the following are true:

- [ ] PR created and URL displayed to the user
- [ ] Plan archived from `docs/plans/active/` to `docs/plans/archive/`
- [ ] For large diffs: walkthrough report exists
- [ ] Commit follows conventional commit format
- [ ] `.harness/state/standard-pipeline/` state files removed on success

## CLI 別実行ガイダンス

このスキルは Claude Code と Codex の両方で動作する。実行モードは AGENTS.md と
`.codex/AGENTS.override.md` の規約に従う。

| 観点 | Claude Code | Codex |
|------|-------------|-------|
| Skill 起動 | `/skill-name` slash command | `$skill-name` mention または `/skills` メニュー (`/skill-name` 形式は built-in 衝突のため使わない) |
| Skill 本体パス | `.claude/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` |
| サブエージェント | `Task(subagent_type=...)` で並列呼び出し可 | 順次 inline 実行 — 単一 agent 内で連続実行 |
| 構造化対話 | `AskUserQuestion` | 番号付き選択肢を stdout に出して数字を待機 |
| 成果物 | `docs/reports/`, `docs/plans/`, `docs/specs/` 共通 | 同左 (CLI 非依存) |

drift check (`./scripts/check-skill-sync.sh`) は両側の本文と起動メタデータを
照合する — 片側だけ編集すると CI で fail する。
