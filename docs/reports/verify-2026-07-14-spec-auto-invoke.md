# Verify report: spec-auto-invoke

- Date: 2026-07-14
- Plan: docs/plans/active/2026-07-14-spec-auto-invoke.md
- Verifier: verifier subagent (spec compliance + static analysis)
- Scope: changed-language scope (shell + golang detected via diff; see static analysis section)
- Prior artifact: docs/reports/self-review-2026-07-14-spec-auto-invoke.md

## Acceptance criteria

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| AC1 | `.claude/skills/spec/SKILL.md` に `disable-model-invocation` 行が存在しない | PASS | `grep -n "disable-model-invocation" .claude/skills/spec/SKILL.md` → exit 1 (0 matches). `templates/base/.claude/skills/spec/SKILL.md` も同様 exit 1 |
| AC2 | description が自動起動のトリガー条件(肯定条件+否定条件)を含み「Manual trigger only.」を含まない | PASS | description に "Invoke when a repository-change request is too vague or abstract for /plan" (肯定) および "Do not invoke for reviews, Q&A or explanations, execution of an existing plan, trivial fixes, or when the user explicitly requests another skill." (否定) を確認。"Manual trigger only." の文字列なし |
| AC3 | `test ! -e .agents/skills/spec/agents/openai.yaml` かつ `test ! -e templates/base/.agents/skills/spec/agents/openai.yaml` が成立する | PASS | 両コマンドとも exit 0 (ファイル不在を確認) |
| AC4 | `./scripts/check-skill-sync.sh` がパスする(ルート) | PASS | `[ok] check-skill-sync: 13 skill(s) in lock-step` / exit 0 |
| AC5 | `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/check-skill-sync.sh` がパスする(テンプレート側) | PASS | `[ok] check-skill-sync: 12 skill(s) in lock-step` / exit 0 |
| AC6 | `./scripts/check-sync.sh` がパスする | PASS | `PASS: all files in sync.` IDENTICAL:176, DRIFTED:0 / exit 0 |
| AC7 | `tests/test-sync-skills.sh` に「flag 削除→再同期で openai.yaml が削除される」ケースが追加されパスする | PASS (static only) | Case G (lines 192-231) が追加済み。コメントヘッダー 14行目に G を記載。2-phase: 初期 sync で openai.yaml 作成を確認→flag 削除して再 sync で openai.yaml 削除+agents/ dir 削除を確認。行動テストは /test スコープ |
| AC8 | `CLAUDE.md` / `AGENTS.md` / `README.md` / `docs/architecture/repo-map.md` / `templates/base/CLAUDE.md` / `templates/base/AGENTS.md` に spec を手動トリガーとする記述が残っていない | PASS | 詳細はドキュメントドリフトセクション参照 |
| AC9 | `./scripts/run-verify.sh` がパスする | PASS | `==> All verifiers passed.` / exit 0 |

## Plan-specific checks

| Check | Result | Evidence |
|-------|--------|----------|
| `disable-model-invocation` 行の不在(root SKILL.md) | PASS | `grep` → exit 1 (0件) |
| `disable-model-invocation` 行の不在(template SKILL.md) | PASS | `grep` → exit 1 (0件) |
| description に肯定条件("Invoke when") | PASS | description 内に "Invoke when a repository-change request is too vague or abstract for /plan" 確認 |
| description に否定条件("Do not invoke") | PASS | description 内に "Do not invoke for reviews, Q&A or explanations…" 確認 |
| "Manual trigger only." が description に含まれない | PASS | 文字列なし |
| `test ! -e .agents/skills/spec/agents/openai.yaml` | PASS | exit 0 |
| `test ! -e templates/base/.agents/skills/spec/agents/openai.yaml` | PASS | exit 0 |

## Static analysis

`./scripts/run-static-verify.sh` を実行 (2026-07-14T03:20:11Z):

- Shell syntax (`sh -n`): 全 hook ファイルおよびテンプレート側 OK
- `jq -e .` (`.claude/settings.json`, `templates/base/.claude/settings.json`): OK
- Codex hook 単一ソースガード: OK
- `scripts/check-sync.sh`: PASS (DRIFTED:0)
- `scripts/check-pipeline-sync.sh`: all 8 reference files OK
- `scripts/check-skill-sync.sh`: 13 skill(s) in lock-step
- Language scope: golang verifier (gofmt + staticcheck: 0 issues)
- 全体結果: **exit 0 / All verifiers passed**

証跡ログ: `docs/evidence/verify-2026-07-14-032011.log`

## Documentation drift

対象ファイルの手動トリガー記述を個別確認:

| ファイル | 確認結果 |
|----------|---------|
| `CLAUDE.md` | `Manual-trigger skills: /release only` — spec は auto-invoked 一覧に追記済み。PASS |
| `AGENTS.md` | Primary loop: `Spec (auto, optional —` に変更済み。PASS |
| `README.md` | 「Every step in the loop, including /spec, is auto-invoked. /release is the only manual-trigger skill」に変更済み。PASS |
| `docs/architecture/repo-map.md` | `.claude/skills/spec/`: `auto-invoked when a request is too vague for /plan` に変更済み。`/release` が `manual trigger; repo-only` と明記 — 正しい記述として期待される。PASS |
| `templates/base/CLAUDE.md` | `All skills (…) are auto-invoked. The scaffold ships no manual-trigger skill.` に変更済み。PASS |
| `templates/base/AGENTS.md` | Primary loop: `Spec (auto, optional —` に変更済み。PASS |

広域ドリフトスキャン:

- `docs/recipes/codex-setup.md:82` / `templates/base/docs/recipes/codex-setup.md:82`: `disable-model-invocation ⇔ policy.allow_implicit_invocation` の技術説明文 — spec を手動トリガーと述べる記述ではない。問題なし。
- `.claude/skills/release/SKILL.md` / `.agents/skills/release/SKILL.md`: `Manual trigger only.` — /release 自体の記述であり正しい。
- `docs/plans/`, `docs/reports/`, `docs/specs/` 配下: スキャン対象外(アーカイブ)。

残存ドリフト: **なし**

## Self-review findings (from prior report)

| Severity | Finding | Impact on verify |
|----------|---------|-----------------|
| LOW | test-sync-skills.sh: case G が source 上 case F より前に配置 (comment 順と不一致) | 機能的影響なし。verify の判定に影響しない |
| LOW | 中間 trap (line 196) が直後の最終 trap (line 237) に上書きされるが、既存イディオムと一致 | 機能的影響なし |

HIGH/CRITICAL 所見なし。

## Known gaps

- **AC7 の行動テスト**: `tests/test-sync-skills.sh` Case G の実際の実行確認は `/test` スコープ。静的確認(コード存在・構造)は完了。
- **自動起動しきい値の実効性**: description の記述がモデルの実際の自動起動判断に正しく反映されるかは runtime/behavioral 特性であり静的検証では確認不能。
- **`/release` の `disable-model-invocation: true` 維持**: 変更対象外であることは確認済み(grep で残存確認)。

## Verdict

**PASS**

全 9 受け入れ基準をパス。静的解析 exit 0。ドキュメントドリフトなし。
