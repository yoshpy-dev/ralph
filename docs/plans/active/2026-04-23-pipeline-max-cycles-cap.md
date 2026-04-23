# pipeline-max-cycles-cap

- Status: Draft
- Owner: Claude Code
- Date: 2026-04-23
- Related request: ポスト実装パイプラインの最大ループ回数をデフォルト2回（総実行）に制限する
- Related issue: N/A
- Branch: feat/pipeline-max-cycles-cap

## Objective

ポスト実装パイプライン（`/self-review → /verify → /test → /sync-docs → /codex-review → /pr`）の**総実行回数をデフォルト2回まで**に制限する。標準フローと Ralph Loop の両方に適用する。初回実行 + 再実行1回で上限。2回目の `/codex-review` で ACTION_REQUIRED が残っていても自動再帰はせず、ユーザー判断に委ねる（標準フロー）か、`_finalize` でエスカレーションする（Ralph Loop）。

## Scope

- Ralph Loop：`RALPH_MAX_OUTER_CYCLES` のデフォルト値を `3` から `2` に変更（既存変数はそのまま活用）
- 標準フロー：新しい独立変数 `RALPH_STANDARD_MAX_PIPELINE_CYCLES`（デフォルト `2`）を `ralph-config.sh` に追加
- **プランパスの永続化**：`/work` 開始時点で対象プランの絶対パス（`docs/plans/active/<date>-<slug>.md`）を `.harness/state/standard-pipeline/active-plan.json` に 1 回だけ書き込む。`/codex-review` と `/pr` は `docs/plans/active/` を再スキャンせず、この永続化された識別子を参照する（Codex 指摘 2 対応）
- 標準フロー用のサイクルカウント永続化：`.harness/state/standard-pipeline/cycle-count.json` に保存。キーは永続化されたプランパス。`/codex-review` 実行ごとに increment、`/pr` 成功時にリセット
- `.claude/skills/codex-review/SKILL.md` の Case A / Case B 再実行分岐に「サイクル上限チェック」ステップを追加
- `.claude/skills/work/SKILL.md` にプランパス永続化（Step 0 直後）、サイクルカウンタ初期化、`/pr` 時クリーンアップを追記
- ドキュメント同期：`post-implementation-pipeline.md`, `subagent-policy.md`, `README.md`, `AGENTS.md`, `CLAUDE.md`, `docs/quality/definition-of-done.md`, `docs/recipes/ralph-loop.md`
- テンプレート側（`templates/base/scripts/ralph-config.sh` など）も同期

## Non-goals

- Inner Loop の回数制限変更（`RALPH_MAX_INNER_CYCLES=10` はそのまま）
- `RALPH_MAX_ITERATIONS=20` の変更
- 新しい CLI フラグの追加（将来の拡張余地として残すが本プランでは不要）
- Codex トリアージロジック自体の変更（分類アルゴリズムは既存のまま）
- ユーザーが明示的に上書きするメカニズムの UI 刷新（環境変数で上書き可能な現状を維持）

## Assumptions

- `RALPH_MAX_OUTER_CYCLES=2` にすることで Outer Loop は最大 2 サイクルまで（`_outer_cycle=1, 2` まで許可、`3` でエスカレーション）となり、ユーザーの意図する「総実行 2 回」と一致する（`ralph-pipeline.sh:968-971` のロジックを精査済み）
- 標準フローのサイクル数は Claude Code セッション間で持続するため、ファイルベースのカウンタ（`.harness/state/standard-pipeline/cycle-count.json`）が適切。session memory は context compaction で失われる可能性があるため不採用
- プランの同一性は `/work` 開始時に 1 回だけ固定する。`/codex-review` と `/pr` は `.harness/state/standard-pipeline/active-plan.json` に記録された絶対パスを参照し、`docs/plans/active/` を再スキャンしない（Codex 指摘 2 対応）
- サイクルカウンタのキーは永続化されたプランの絶対パス（または SHA256）を使う。スラグ抽出や session_start_context.sh の辞書順ピックに依存しない
- 複数 active plan が並存する場合、`/work` は「どのプランを対象にするか」を AskUserQuestion で 1 回だけ確認し、決定したパスを永続化する。以降の skill はその固定パスのみを使う
- 標準フローで上限到達時は `/codex-review` の Case A/B で表示される選択肢から「修正する」オプションを除外し、代わりに「上限到達の通知 + 継続 / PR 作成 / 中止」を提示する
- `ralph init` で配布される `templates/base/` 配下のスクリプトも同時に更新する必要がある（source と destination の両方）

## Affected areas

- `scripts/ralph-config.sh`（`RALPH_MAX_OUTER_CYCLES` デフォルト変更、`RALPH_STANDARD_MAX_PIPELINE_CYCLES` 新規追加）
- `scripts/ralph-pipeline.sh`（`--help` 文言のデフォルト値更新）
- `templates/base/scripts/ralph-config.sh`（source と同期）
- `templates/base/scripts/ralph-pipeline.sh`（source と同期）
- `.claude/skills/codex-review/SKILL.md`（Case A/B にサイクル上限チェック追加、DISMISSED 以外の分岐を再構成）
- `.claude/skills/work/SKILL.md`（サイクルカウンタ初期化と `/pr` 前のクリーンアップ記述）
- `.claude/rules/post-implementation-pipeline.md`（「Re-run after Codex ACTION_REQUIRED fix」に上限記述追加）
- `README.md`, `AGENTS.md`, `CLAUDE.md`, `docs/quality/definition-of-done.md`, `docs/recipes/ralph-loop.md`（仕様記述の同期）
- `tests/test-ralph-config.sh`（新デフォルト値の検証、新変数の validation）
- 新規：`.harness/state/standard-pipeline/`（.gitignore 登録の既存パターン `.harness/state/` 配下のため追加作業不要）

## Design decisions

<!-- Critical forks resolved with the user. Each entry: 判断・採用した選択肢・理由（rationale）。 -->

- **「最大ループ回数2回」の意味**：パイプライン**総実行 2 回**を採用（初回 + 再実行 1 回）。2 回目の `/codex-review` で ACTION_REQUIRED が残っても自動再帰はしない。Rationale: ユーザー選択どおり。無制限ループ抑止が主目的であり、総実行数で制御する方が直感的。
- **Ralph Loop の既存変数との関係**：`RALPH_MAX_OUTER_CYCLES` は Ralph Loop 専用のまま残し、デフォルトを `3 → 2` に変更。標準フローは別変数 `RALPH_STANDARD_MAX_PIPELINE_CYCLES` で管理。Rationale: ユーザー選択どおり。Ralph Loop の Outer Loop セマンティクス（regression カウント）と標準フローのパイプライン総実行回数はスコープが異なるため、変数を分けることで将来の独立調整を可能にする。
- **上限到達時の標準フロー挙動**：ユーザーに AskUserQuestion で判断を委ねる。選択肢：「上限解除して再実行」「PR 作成」「中止」。Rationale: ユーザー選択どおり。標準フローは元来インタラクティブであり、強制中断より選択肢提示が harness のポリシーに合致する。
- **カウンタ永続化方式**：ファイルベース（`.harness/state/standard-pipeline/cycle-count.json`）。Rationale: Claude Code session memory は context compaction で失われるため、プラン単位で永続化。`/pr` 成功時にクリーンアップ、中断時は次回 `/work` 開始時に残存カウンタを検出して継続可能。
- **プラン識別の固定化**（Codex 指摘 2 採用）：`/work` 開始時にプランの絶対パスを `.harness/state/standard-pipeline/active-plan.json` に記録し、以降の skill は再スキャンせずこのファイルを参照する。Rationale: 現状の `docs/plans/active/` 再スキャンや session hook の辞書順ピックは、複数 active plan やリネーム下で同一性保証にならず、カウンタ誤紐付けの温床になる。永続化により `/work → /codex-review → /pr` の全ステップで同じプランを指すことを保証する。

## Acceptance criteria

- [ ] `RALPH_MAX_OUTER_CYCLES` のデフォルト値が `2` に変更され、`scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` の両方で一致する
- [ ] `RALPH_STANDARD_MAX_PIPELINE_CYCLES` が `ralph-config.sh` に追加され、デフォルト `2`、`validate_all_numeric` で検証される
- [ ] `tests/test-ralph-config.sh` に新変数の default / override / validation ケースが追加され、パスする
- [ ] `.claude/skills/codex-review/SKILL.md` の Case A と Case B の各選択肢に「サイクル上限チェック」が記述されている
- [ ] 上限到達時に「修正する」オプションが除外され、「上限解除」「PR 作成」「中止」が提示される挙動が SKILL.md に明記されている
- [ ] `.claude/skills/work/SKILL.md` にプランパス永続化（`.harness/state/standard-pipeline/active-plan.json` への書き込み）、サイクルカウンタ初期化（cycle=1 の登録）、`/pr` 完了時のカウンタ・プランパス両方のクリア手順が記載されている
- [ ] `.claude/skills/codex-review/SKILL.md` と `.claude/skills/pr/SKILL.md` が `docs/plans/active/` を再スキャンせず、`active-plan.json` の識別子を参照するよう明記されている
- [ ] 複数 active plan 存在時の `/work` の挙動（AskUserQuestion で 1 プランを選択、以降固定）が SKILL.md に記載されている
- [ ] `.claude/rules/post-implementation-pipeline.md` の「Re-run after Codex ACTION_REQUIRED fix」セクションに上限ルール（標準=2, Ralph Loop=2）が追記されている
- [ ] `README.md` / `AGENTS.md` / `CLAUDE.md` / `docs/quality/definition-of-done.md` / `docs/recipes/ralph-loop.md` の該当記述が同期される
- [ ] `ralph-pipeline.sh --help` のデフォルト表示が `3 → 2` に更新されている
- [ ] `./scripts/run-verify.sh` が 0 で終了する
- [ ] `./scripts/check-sync.sh`（もしくは `check-pipeline-sync.sh`）が source / template の同期を確認して pass する

## Implementation outline

1. **`scripts/ralph-config.sh` 更新**：`RALPH_MAX_OUTER_CYCLES="${RALPH_MAX_OUTER_CYCLES:-2}"` に変更。`RALPH_STANDARD_MAX_PIPELINE_CYCLES="${RALPH_STANDARD_MAX_PIPELINE_CYCLES:-2}"` を追加。`validate_all_numeric` にも追加。
2. **`scripts/ralph-pipeline.sh` 更新**：`--help` 文言のデフォルト値 `3 → 2` を更新。
3. **`templates/base/scripts/` 同期**：上記 2 ファイルをテンプレート側にもコピー／同期。`./scripts/check-sync.sh` で検証。
4. **`tests/test-ralph-config.sh` 拡張**：新 default 値テスト、`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の override / 不正値検証。
5. **`.claude/skills/codex-review/SKILL.md` 改訂**：
   - Step 0 を追加：`.harness/state/standard-pipeline/active-plan.json` を読みプラン絶対パスを取得（存在しなければ警告して従来挙動にフォールバック）。`.harness/state/standard-pipeline/cycle-count.json` を読み、現在のサイクル番号を確定（存在しなければ 1）
   - Step 7（新）：サイクル番号が上限未満なら既存の Case A/B を実行、上限到達なら上限到達フローを提示
   - 上限到達フロー：AskUserQuestion で「上限解除して再実行（環境変数 `RALPH_STANDARD_MAX_PIPELINE_CYCLES=N` を提示）」「PR 作成」「中止」の 3 択
   - **禁則事項**を追記：`docs/plans/active/` の再スキャンによるプラン特定を禁止。必ず `active-plan.json` を参照
6. **`.claude/skills/work/SKILL.md` 追記**：
   - Step 0.5 を追加：プランパス確定（複数 active plan 時は AskUserQuestion で選択、決定したパスを `.harness/state/standard-pipeline/active-plan.json` に書き込み）、`cycle-count.json` を `{<plan-path>: 1}` で初期化
   - Step 10（新）：`/pr` 成功後に `active-plan.json` と `cycle-count.json` を削除
7. **`.claude/skills/pr/SKILL.md` 同期**：プラン識別を `active-plan.json` 参照に変更。アーカイブ対象も同じパスを使用。
8. **`.claude/rules/post-implementation-pipeline.md` 更新**：「Re-run after Codex ACTION_REQUIRED fix」に上限ルール説明を追加。「Where this order is referenced」のリストに新たな記述箇所を追加。
9. **`Affected areas` の .gitignore 確認**：`.harness/state/` 配下が既に無視対象であることを確認（新規 `.harness/state/standard-pipeline/` は自動で無視される）。
10. **ドキュメント同期**：`README.md`, `AGENTS.md`, `CLAUDE.md`, `docs/quality/definition-of-done.md`, `docs/recipes/ralph-loop.md` の該当記述を更新。
11. **検証**：`./scripts/run-verify.sh`, `./scripts/check-sync.sh`, `bash tests/test-ralph-config.sh` を実行してエビデンスを収集。

## Verify plan

- Static analysis checks: `./scripts/run-verify.sh`（`go vet`, `golangci-lint`, doc-sync チェックを含む）
- Spec compliance criteria to confirm: Acceptance criteria の全項目が満たされること。特に「Where this order is referenced」で列挙された全ファイルの同期
- Documentation drift to check: `CLAUDE.md`, `AGENTS.md`, `README.md`, `docs/quality/definition-of-done.md`, `docs/recipes/ralph-loop.md`, `post-implementation-pipeline.md` の記述が新デフォルト（2 回上限）と整合
- Evidence to capture: `docs/reports/verify-2026-04-23-pipeline-max-cycles-cap.md` に `run-verify.sh` 出力、`check-sync.sh` 出力、新変数の grep 結果を保存

## Test plan

- Unit tests:
  - `tests/test-ralph-config.sh`：`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の default=2、override 値、不正値での exit エラーを検証
  - `tests/test-ralph-config.sh`：`RALPH_MAX_OUTER_CYCLES` の default が `2` であることを検証（既存テストがあれば更新）
- Integration tests:
  - Ralph Loop の dry-run（`ralph-pipeline.sh --dry-run --max-outer-cycles=2`）で `MAX_OUTER_CYCLES=2` が出力され、3 回目の Outer Loop が escalate する挙動を確認
- Regression tests:
  - 既存の `tests/test-ralph-status.sh` が新デフォルトで継続してパス
  - `scripts/check-pipeline-sync.sh` が source / template の同期を検出
- Edge cases:
  - `RALPH_STANDARD_MAX_PIPELINE_CYCLES=1` のとき、1 回目の ACTION_REQUIRED で即上限到達フローに入ること
  - `RALPH_STANDARD_MAX_PIPELINE_CYCLES=0` や非数値は validation エラー
  - プランスラグを特定できないとき（`docs/plans/active/` が空）、カウンタなしで警告のみ出して従来挙動にフォールバック
  - `/pr` 失敗時にカウンタが残ること（再実行で続きから数えられること）
- Evidence to capture: `docs/reports/test-2026-04-23-pipeline-max-cycles-cap.md` に上記テスト結果と手動 dry-run ログを保存

## Risks and mitigations

- **Risk**: 既存ユーザーが `RALPH_MAX_OUTER_CYCLES=3` を前提にしている
  - Mitigation: ドキュメント（`README.md`, `docs/recipes/ralph-loop.md`）で変更を明記。環境変数で従来値に戻せることを案内。
- **Risk**: 標準フローのサイクルカウンタが session 間で正しく消えず、別プランで混線
  - Mitigation: `active-plan.json` に永続化した絶対パスをカウンタのキーに使うため、`docs/plans/active/` 再スキャンに起因する誤紐付けを排除。`/pr` 成功時に `active-plan.json` / `cycle-count.json` 両方をクリア。
- **Risk**: テンプレート側との同期漏れ（`templates/base/scripts/`）
  - Mitigation: `scripts/check-sync.sh` で自動検知。verify 時に実行。
- **Risk**: 既存の Ralph Loop 実行中プランが 3 サイクル目で中断される
  - Mitigation: Rollout Note に「in-flight Ralph Loop には影響しないが、再開時は新デフォルト適用」を記載。必要なら `--max-outer-cycles=3` を暫定指定可能。

## Rollout or rollback notes

- Rollout: PR マージ後、新規 `/work` / `/loop` セッションから自動適用。既存の in-flight プランには影響しない（カウンタは新規生成）。
- Rollback: `RALPH_MAX_OUTER_CYCLES` と `RALPH_STANDARD_MAX_PIPELINE_CYCLES` のデフォルト値を戻し、`.claude/skills/codex-review/SKILL.md` のキャップチェック Step を削除するコミットで revert 可能。永続化ファイルは自然に消える（手動削除不要）。

## Open questions

- `.harness/state/standard-pipeline/` ディレクトリ名は合意済みか（提案）。他候補：`.harness/state/pipeline-standard/`, `.harness/state/cycle/`
- `/codex-review` 実行時に `active-plan.json` が存在しない場合（古いセッション継続 or 手動作業）のフォールバック挙動：警告のみで続行 vs エラーで停止（推奨: 警告 + 最新の `docs/plans/active/*.md` を fallback 候補として提示）
- 上限到達時の「上限解除」選択で、環境変数変更をユーザーに促すか、セッションスコープで一時的に引き上げるか
- Codex 指摘 1（スクリプト化）は本プランでは採用しないが、将来のハードニングとして `docs/tech-debt/` に記録すべきか

## Codex plan advisory

- 2026-04-23 実行：HIGH × 2（決定論的スクリプト化 / プラン識別の永続化）
- 採用：指摘 2（プラン永続化）。`.harness/state/standard-pipeline/active-plan.json` を導入し、`/work → /codex-review → /pr` 全ステップで同一識別子を参照
- 見送り：指摘 1（ヘルパースクリプト化）。skill docs ベースの最小変更を優先するユーザー判断。将来のハードニングとして Open questions に記録

## Progress checklist

- [ ] Plan reviewed
- [ ] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
