# spec-auto-invoke

- Status: Approved
- Owner: Claude Code
- Date: 2026-07-14
- Related request: `/spec` は現在 `disable-model-invocation: true` で手動トリガー専用。AIエージェントも自動トリガーできるように変更し、関連ドキュメントを整合させる。
- Related issue: N/A
- Type: feat
- Branch: feat/spec-auto-invoke

## Objective

`/spec` スキルをモデル自動起動可能にする(`disable-model-invocation: true` を撤去)。
リクエストが `/plan` に渡すには曖昧すぎる場合に、エージェントが自律的に `/spec` を
起動できるようにし、手動トリガー一覧・Primary loop 記述などの関連ドキュメントを
すべて新しい状態に揃える。

## Scope

- `.claude/skills/spec/SKILL.md` — frontmatter から `disable-model-invocation: true` を削除。
  description 末尾の「Manual trigger only.」を自動起動トリガー条件の記述
  (いつモデルが起動すべきか)に置き換える。
- `.agents/skills/spec/` — `scripts/sync-skills.sh` で再生成
  (SKILL.md の description 同期 + `agents/openai.yaml` の自動削除)。
- `templates/base/.claude/skills/spec/SKILL.md` — ルートと同一の変更
  (`scripts/check-sync.sh` がルート⇔テンプレートの同一性を強制)。
- `templates/base/.agents/skills/spec/` — テンプレート側ミラーの再生成
  (`CLAUDE_ROOT`/`CODEX_ROOT` を templates/base 配下に向けて `sync-skills.sh` を実行)。
- ドキュメント整合:
  - `CLAUDE.md` — 手動トリガー一覧から `/spec` を外し `/release` のみに。auto-invoked 一覧に spec を追加。
  - `AGENTS.md` — Primary loop の「Spec (manual, optional」→「Spec (auto, optional」。
  - `README.md` — 「`/spec` is the only manual trigger in the loop」と「**Spec** (manual, optional」の修正。
  - `docs/architecture/repo-map.md` — 「(manual trigger)」表記の修正。
  - `templates/base/CLAUDE.md` — 「`/spec` is the only manual-trigger skill」の修正(テンプレート側は `/release` を含まないため「manual-trigger skills はなし」相当の記述に)。
  - `templates/base/AGENTS.md` — ルート AGENTS.md と同じ Primary loop 修正。
- `tests/test-sync-skills.sh` — 「`disable-model-invocation` を削除して再同期すると
  `agents/openai.yaml` が削除される」回帰ケースを追加(Codex 勧告 #1)。

## Non-goals

- `/spec` スキル本文(Steps/Goals/ワークフロー)の変更 — トリガーポリシーのみ変更する。
- `/release` のトリガーポリシー変更(手動のまま)。
- `anti-bottleneck` の `user-invocable: false` の変更。
- sync-skills.sh / check-skill-sync.sh のロジック変更(既存の変換規則で対応可能)。
- Codex 側 `.codex/` 設定の変更(openai.yaml はミラー再生成で自動的に消える)。

## Assumptions

- `sync-skills.sh` は source から `disable-model-invocation` が消えると
  `agents/openai.yaml`(`allow_implicit_invocation: false`)を自動削除する(scripts/sync-skills.sh:143-156 で確認済み)。
- モデル自動起動の判断材料は frontmatter の `description` なので、
  「Manual trigger only.」を「いつ起動すべきか」を示す文に置き換えることが動作変更の一部である。
- `scripts/check-sync.sh` は `.claude/skills/spec/` をルート⇔templates/base で比較する
  (release のみ除外)ため、テンプレート側も同時変更が必須。
- scaffold manifest の SHA256 はレンダリング時に計算されるため、静的なハッシュ再生成は不要。

## Affected areas

- `.claude/skills/spec/SKILL.md`
- `.agents/skills/spec/SKILL.md`、`.agents/skills/spec/agents/openai.yaml`(削除)
- `templates/base/.claude/skills/spec/SKILL.md`
- `templates/base/.agents/skills/spec/SKILL.md`、同 `agents/openai.yaml`(削除)
- `CLAUDE.md`、`AGENTS.md`、`README.md`、`docs/architecture/repo-map.md`
- `templates/base/CLAUDE.md`、`templates/base/AGENTS.md`

## Design decisions

Critical forks: None

- description の書き換えは「Invoke when the request is too vague or abstract for /plan」系の
  トリガー条件記述とする(既存の CLAUDE.md「Use /spec when the request is too vague for /plan」に整合。
  reasonable default として採用)。
- **[Codex 勧告 #2 反映]** description には肯定条件と否定条件の両方を明記する:
  - 肯定: objective / scope / 受け入れ条件が欠けた抽象的なリポジトリ変更依頼
  - 否定: レビュー依頼、Q&A・説明依頼、既存プラン実行、trivial fix、他スキルの明示指定時は起動しない
- テンプレート側への波及は選択の余地なし(`check-sync.sh` の同一性強制による)。

## Acceptance criteria

- [x] `.claude/skills/spec/SKILL.md` に `disable-model-invocation` 行が存在しない
- [x] description が自動起動のトリガー条件(肯定条件+否定条件)を含み「Manual trigger only.」を含まない
- [x] `test ! -e .agents/skills/spec/agents/openai.yaml` かつ
      `test ! -e templates/base/.agents/skills/spec/agents/openai.yaml` が成立する
- [x] `./scripts/check-skill-sync.sh` がパスする(ルート)
- [x] `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/check-skill-sync.sh` がパスする(テンプレート側)
- [x] `./scripts/check-sync.sh` がパスする(ルート⇔templates/base)
- [x] `tests/test-sync-skills.sh` に「flag 削除→再同期で openai.yaml が削除される」ケースが追加され、パスする
- [x] `CLAUDE.md` / `AGENTS.md` / `README.md` / `docs/architecture/repo-map.md` /
      `templates/base/CLAUDE.md` / `templates/base/AGENTS.md` に
      spec を手動トリガーとする記述が残っていない
- [x] `./scripts/run-verify.sh` がパスする

## Implementation outline

1. `.claude/skills/spec/SKILL.md` の frontmatter を修正(flag 削除 + description 更新)。
2. `templates/base/.claude/skills/spec/SKILL.md` に同一修正を適用。
3. `./scripts/sync-skills.sh` を実行(ルート)。`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` を実行(テンプレート側)。両側で `test ! -e .../agents/openai.yaml` を確認。
4. `tests/test-sync-skills.sh` に flag 削除→再同期→openai.yaml 削除の回帰ケースを追加。
5. ドキュメント 6 ファイルの手動トリガー記述を修正。
6. `./scripts/run-verify.sh` + テンプレート側 `check-skill-sync.sh` + `./scripts/run-test.sh` で検証。

## Verify plan

- Static analysis checks: `./scripts/run-verify.sh`(shellcheck 等の既存ゲート含む)
- Spec compliance criteria to confirm: acceptance criteria の grep 検証
  (`grep -rn "disable-model-invocation" .claude/skills/spec templates/base/.claude/skills/spec` が 0 件、
  `grep -rn "manual" CLAUDE.md AGENTS.md README.md docs/architecture/repo-map.md templates/base/CLAUDE.md templates/base/AGENTS.md | grep -i spec` が 0 件)
- Documentation drift to check: docs/recipes/codex-setup.md ほかに spec=手動の残存記述がないか全文 grep
- Evidence to capture: check-skill-sync.sh / check-sync.sh / run-verify.sh の実行ログ

## Test plan

- Unit tests: `tests/test-sync-skills.sh`(flag 削除→再同期で openai.yaml が削除される新規ケースを追加。既存 Case D は生成のみ検証)、`tests/test-check-skill-sync.sh`(既存)
- Integration tests: `./scripts/run-test.sh`(Go テスト含む — `internal/scaffold/embed_test.go` の `.agents/skills/` 存在チェックが通ること)
- Regression tests: `./scripts/check-sync.sh` / `./scripts/check-skill-sync.sh`
- Edge cases: templates/base 側ミラーの openai.yaml が正しく削除されること(rmdir で agents/ が空なら消える)
- Evidence to capture: テスト実行ログをレポートに記録

## Risks and mitigations

- **自動起動により /spec が過剰に発火する** — description に「too vague for /plan」という
  明確な条件を書き、CLAUDE.md 側にも「/plan で足りる場合は /spec を挟まない」旨を維持。
- **ミラー再生成漏れ(templates 側)** — check-sync.sh / check-skill-sync.sh が CI で drift を検出。
- **embed_test.go の期待ファイルリストとの不整合** — openai.yaml 削除が期待リストに含まれていないか
  実装時に確認(`.agents/skills/.gitkeep` レベルの検査であれば影響なし)。

## Rollout or rollback notes

- 単一 PR で完結。ロールバックは revert のみで済む(状態・スキーマ変更なし)。
- 下流プロジェクトは `ralph upgrade` で新テンプレートを取り込む(hash diff により自動更新または conflict 提示)。

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created
