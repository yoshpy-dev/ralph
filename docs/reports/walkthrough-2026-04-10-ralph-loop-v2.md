# Walkthrough: Ralph Loop プランシステム再設計 (v2)

Date: 2026-04-10
Branch: feat/ralph-loop-v2
Diff: 74 files, +7496 / -65 lines

## 変更の全体像

本 PR は Ralph Loop の並列スライス実行基盤を構築する。主な変更は 4 領域:

1. **パイプライン基盤** — `ralph-pipeline.sh` (Inner/Outer Loop 自律実行)
2. **並列オーケストレータ** — `ralph-orchestrator.sh` (マルチワークツリー並列実行)
3. **ディレクトリベースプラン** — テンプレート + 生成スクリプト + CLI 統合
4. **レガシー廃止** — インラインスライスモードの完全除去

## 1. パイプライン基盤 (`scripts/ralph-pipeline.sh`)

894 行の新規スクリプト。`claude -p` を Inner/Outer Loop で呼び出す自律パイプライン。

### Inner Loop
```
implement → self-review → verify → test
→ テスト失敗: retry (max-inner-cycles まで)
```

### Outer Loop
```
sync-docs → codex-review → PR
→ ACTION_REQUIRED: Inner Loop に差し戻し
```

### 主要機能
- **Preflight probe**: Claude CLI の JSON 出力対応、codex 可用性を事前検査
- **Checkpoint**: `checkpoint.json` で全状態を永続化。`ralph status` で確認可能
- **Stuck detection**: HEAD コミットハッシュ比較で 3 回連続無進捗を検出
- **Agent signal protocol**: サイドカーファイル + stdout マーカーの 2 層シグナル
- **Failure triage**: テスト失敗時のエラー分類 + リペア試行

## 2. 並列オーケストレータ (`scripts/ralph-orchestrator.sh`)

763 行の新規スクリプト。ディレクトリベースプランからスライスをパースし、各スライスをワークツリーで並列実行。

### フロー
```
parse_slices() → create_integration_branch() → create_worktree() × N
  → run_slice() × N (並列、依存待ち)
  → integration_merge() (依存順で sequential merge)
  → create_unified_pr() (--unified-pr 時)
```

### 主要機能
- **ディレクトリベースパース**: `slice-*.md` からスラグ・目的・依存・影響ファイルを抽出
- **デュアルフォーマット対応**: インラインフィールド (`- Objective: ...`) とセクションヘッダ (`## Objective`) の両方
- **Shared-file locklist**: `_manifest.md` から読み取り + 自動検出 (重複ファイル)
- **依存解決**: スライス間依存を尊重し、完了待ちで並列度を最大化
- **Integration branch**: `integration/<slug>` に sequential merge、コンフリクト時は即中止
- **依存スラグ正規化**: `slice-1-foo` → `1-foo` (Codex P1 修正)

## 3. ディレクトリベースプラン

### 構造
```
docs/plans/active/<date>-<slug>/
  ├── _manifest.md       ← メタデータ, locklist, 依存グラフ
  ├── slice-1-<name>.md  ← 自己完結スライスプラン
  ├── slice-2-<name>.md
  └── ...
```

### 新規ファイル
- `docs/plans/templates/ralph-loop-manifest.md` — マニフェストテンプレート
- `docs/plans/templates/ralph-loop-slice.md` — スライステンプレート
- `scripts/new-ralph-plan.sh` — ディレクトリ生成スクリプト

### CLI 統合
- `scripts/ralph run --plan <directory>` — ディレクトリ自動検出で `--slices` 有効化
- `scripts/archive-plan.sh` — ファイル/ディレクトリ両対応

## 4. レガシー廃止

- `parse_slices_inline()` 削除 (旧: 単一ファイル内 `### Slice N:` ヘッダ)
- `integration_merge_check()` 削除 (旧: ドライマージチェック)
- `docs/plans/templates/ralph-loop-plan.md` 削除
- Integration branch を常時作成 (旧: `--unified-pr` 時のみ)
- 全ドキュメントからインライン参照を除去

## スキル・ルール変更

| ファイル | 変更 |
|---------|------|
| `.claude/skills/plan/SKILL.md` | Step 2.7 にフロー選択追加 (標準/単一/並列) |
| `.claude/skills/loop/SKILL.md` | 並列スライス実行コマンド追記 |
| `.claude/rules/subagent-policy.md` | オーケストレータモード文書化 |
| `docs/quality/definition-of-done.md` | 並列スライスモード完了条件追記 |
| `AGENTS.md` | Repo map にディレクトリベースプラン記載 |

## Codex 修正 (2件)

1. **依存スラグ正規化** (`ralph-orchestrator.sh:638`): `sed 's/^slice //'` → `sed 's/^slice[- ]*//'`
2. **base branch フォールバック** (`ralph-pipeline.sh:621`): パイプ分離 + `${_base:-main}`

## 検証結果

- Self-review: CRITICAL/HIGH なし (2 回実行)
- Verify: PASS (2 回実行)
- Test: 20/20 PASS
- Codex: ACTION_REQUIRED 1件修正済み、WORTH_CONSIDERING 1件修正済み
- `sh -n`: 全17スクリプト PASS
- `run-verify.sh`: exit 0

## 既知のギャップ

- `claude -p` のライブ API パスは未テスト (dry-run のみ)
- マルチワークツリー並列実行のランタイムは未テスト
- `shellcheck` 未インストール
