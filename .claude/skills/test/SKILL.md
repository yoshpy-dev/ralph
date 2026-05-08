---
name: test
description: Run behavioral tests (unit, integration, regression) and produce a test report. Tests must pass before PR creation. Invoke automatically after /verify completes.
allowed-tools: Read, Grep, Glob, Bash, Write
---
Run tests and write a report to `docs/reports/`.

## Preferred flow

1. Read the active plan and its test plan section.
2. Run `./scripts/run-test.sh` (tests only) unless there is a stronger project-specific test runner.
3. Capture test results, coverage, and failure analysis in a report from [template.md](template.md).
4. Save raw test output to `docs/evidence/test-<date>-<slug>.log`.
5. If no tests exist or test infrastructure is missing, say so explicitly and propose the smallest useful test to add.
6. Distinguish:
   - passing
   - failing (with root cause analysis)
   - skipped (with reason)

## Test categories

- **Normal path**: Expected inputs produce expected outputs
- **Error path**: Invalid inputs, missing dependencies, boundary conditions
- **Regression**: Previously broken behavior stays fixed

## Gate

**Tests must pass before PR creation.** If any test fails:
- Record the failure in the report
- Do NOT proceed to /pr
- Propose a fix or flag the failure for human decision

## What /test does NOT do

- **Static analysis**: That is the responsibility of `/verify`.
- **Diff quality**: That is the responsibility of `/self-review`.
- **Spec compliance**: That is the responsibility of `/verify`.

## Output

- `docs/reports/test-<date>-<slug>.md` — human-readable summary
- `docs/evidence/test-<date>-<slug>.log` — raw test output
- clear pass/fail verdict
- explicit test gaps

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
