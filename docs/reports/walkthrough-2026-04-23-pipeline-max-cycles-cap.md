# Walkthrough: pipeline-max-cycles-cap

- Date: 2026-04-23
- Plan: `docs/plans/archive/2026-04-23-pipeline-max-cycles-cap.md`
- Branch: `feat/pipeline-max-cycles-cap`
- Diff: 22 files changed, 777 insertions(+), 49 deletions(-)
- Commits: 12

## ゴール

ポスト実装パイプライン（`/self-review → /verify → /test → /sync-docs → /codex-review → /pr`）の**総実行回数をデフォルト2回まで**に制限する。初回 + 再実行1回で上限。標準フローと Ralph Loop の両方に適用。

## 全体像

```
Before: 無制限ループ（標準フロー）、Ralph Loop は MAX_OUTER_CYCLES=3
After:  標準フロー = RALPH_STANDARD_MAX_PIPELINE_CYCLES=2、Ralph Loop = RALPH_MAX_OUTER_CYCLES=2
```

## 変更内訳（ファイル別）

### 1. Shell config（デフォルト値）
- `scripts/ralph-config.sh`: `RALPH_MAX_OUTER_CYCLES` 3→2、新変数 `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2` 追加、`validate_all_numeric` で検証。
- `scripts/ralph-pipeline.sh`: `--help` のデフォルト表示を 3→2 に更新。
- `templates/base/scripts/ralph-config.sh` / `ralph-pipeline.sh`: 上記を同期。

### 2. 標準フロー skill（プラン識別 + カウンタ）
- `.claude/skills/work/SKILL.md`:
  - **Step 0**: プラン解決を最優先。`.md` ファイルのみ受理、ディレクトリは `/loop` へ誘導。単一/複数/0 件のケースで分岐。
  - **Step 0.5**: Step 0 で解決したパスを使って branch 作成。plan file の `Branch: TBD` 書き換え。
  - **Step 0.7**: `.harness/state/standard-pipeline/active-plan.json` に絶対パスを永続化。`cycle-count.json` を 3 分岐で初期化（missing/match→保持/mismatch→AskUserQuestion）。
  - **Step 9.e/f**: `/codex-review` と `/pr` のキャップ挙動を説明。

- `.claude/skills/codex-review/SKILL.md`:
  - **Step 0**: 永続化モード（active-plan.json 有）とフォールバックモード（無）で分岐。フォールバックモードは `cycle-count.json` を触らない（orphan state 防止）。
  - **Step 4**: トリアージレポートヘッダに `Cycle: X/Y` を記載。
  - **Step 6 Case A / Case B**: `CAP_REACHED` で分岐。キャップ到達時は「修正」オプションを除外し、「上限解除」「PR 作成」「中止」を提示。両 Case で対称。
  - **Step 7**: 非キャップ再実行は cycle を increment、キャップ解除再実行は increment しない（ユーザーに `RALPH_STANDARD_MAX_PIPELINE_CYCLES=<cycle+1>` を export するよう指示）。
  - **Step 3 triage context**: `active-plan.json` のパスを必ず参照、再スキャン禁止。

- `.claude/skills/pr/SKILL.md`:
  - **Step 0**: `active-plan.json` からプラン識別。
  - **Step 5**: 永続化されたパスでアーカイブ。
  - **Step 6**: 成功時に `active-plan.json` と `cycle-count.json` を削除。

### 3. ルール・ドキュメント同期
- `.claude/rules/post-implementation-pipeline.md`: 「Pipeline cycle cap (default 2 total runs)」節を新設。標準フロー = `RALPH_STANDARD_MAX_PIPELINE_CYCLES`、Ralph Loop = `RALPH_MAX_OUTER_CYCLES`、エスカレーション動作を明記。
- `docs/quality/definition-of-done.md`: 2 回上限への参照追加。
- `docs/recipes/ralph-loop.md`: 環境変数表に新デフォルトと新変数を反映。
- 上記 3 ファイルとも `templates/base/` 配下にミラー。

### 4. テスト
- `tests/test-ralph-config.sh`: default=2 検証、override=5 検証、非数値/0 で validation エラーを確認。4 アサーション追加（合計 27/27 PASS）。

### 5. Design decisions
- **「最大ループ回数2回」 = 総実行2回**（初回 + 再実行1回）。
- Ralph Loop は既存変数 `RALPH_MAX_OUTER_CYCLES` を継続使用（名前を変えない）。標準フローは新規独立変数で管理。将来の独立調整余地を残す。
- 上限到達時は **ユーザーに判断を委ねる**（強制中断より選択肢提示）。
- プラン識別はファイル永続化（`.harness/state/standard-pipeline/active-plan.json`）。session memory は context compaction で失われるため採用せず（**Codex 指摘 2 採用**）。

## 合計 4 回の Codex レビューサイクル

| サイクル | 指摘 | 採用 | コミット |
|----|----|----|----|
| Plan advisory | HIGH×2（スクリプト化 / プラン永続化） | 永続化のみ採用 | — |
| Cycle 1 | P2×2（カウンタリセット / Case B で AskUserQuestion skip） | 両方修正 | `e27102a` |
| Cycle 2 | P1 + P2（Step 順序 / cap-override increment） | 両方修正 | `12b87ee` |
| Cycle 3 | P2×2（ディレクトリ受入 / fallback で cycle-count 触る） | 両方修正 | `ebb926d` |

詳細: `docs/reports/codex-triage-2026-04-23-pipeline-max-cycles-cap.md`

## 検証エビデンス

- `./scripts/run-verify.sh`: exit 0（全サイクル）。最新: `docs/evidence/verify-2026-04-23-123746.log`
- `./scripts/check-sync.sh`: DRIFTED=0、IDENTICAL=107（source/template 完全同期）
- `bash tests/test-ralph-config.sh`: 27/27 PASS
- `bash tests/test-ralph-status.sh`: 40/40 PASS（regression なし）

## 既知ギャップ / フォローアップ候補

- **スクリプト化（Codex advisory 指摘 1）**: 標準フローのキャップ追跡は skill prompt 駆動のまま。決定論的なヘルパースクリプト化は本 PR では見送り。E2E テスト不在。将来のハードニング候補。
- **AC チェックボックス未チェック**: プランの Acceptance criteria は `[ ]` のまま（`feedback_plan_ac_checklist_drift` 方針により PR 作成時点で未更新）。レビュー時にマージ直前で更新可。
- **in-flight Ralph Loop**: 実行中のプランには影響しない（新規 `/loop` から新デフォルト適用）。`--max-outer-cycles=3` で従来値に戻せる。

## ロールバック

環境変数上書きで即時ロールバック可:
```sh
export RALPH_MAX_OUTER_CYCLES=3
export RALPH_STANDARD_MAX_PIPELINE_CYCLES=10  # 実質無制限
```

コードレベルの revert は `scripts/ralph-config.sh` のデフォルト値戻し + skill docs の Step 追加分削除。永続化ファイル（`.harness/state/standard-pipeline/`）は gitignore 配下なので自然消滅。
