# Walkthrough: Pipeline robustness improvements

- Date: 2026-04-09
- Plan: docs/plans/active/2026-04-09-pipeline-robustness.md
- Branch: feat/ralph-loop-v2 (additional commits, b426fab onwards)
- PR: #5

## 変更概観

10コミット（b426fab〜a563a12）、スクリプト/プロンプト/ドキュメントに分散。PR #5 監査で発見された実行時リスク 3件と軽微な不整合 2件を改善する。中核の変更対象は `scripts/ralph-pipeline.sh` のみ。

## AC1, AC2, AC6: JSON 出力モード移行

### 変更前の問題

`run_claude()` は `--output-format text` を使い、結果テキストをそのまま log ファイルに書き込んでいた。session_id は `grep 'Session:' log` で抽出していた。このアプローチはテキスト形式に依存し、非 JSON 混在（警告行など）で壊れるリスクがあった。

### 変更後の動作

`run_claude()` が `--output-format json` に移行した。stdout と stderr を分離し、JSON 全体を `${_log_file}.json` に保存してから `.result` を jq で抽出する。

```sh
claude -p --output-format json < "$_prompt_file" \
    > "${_log_file}.json" 2>"${_log_file}.stderr"
jq -e -r '.result // empty' "${_log_file}.json" > "$_log_file" || {
    # jq 失敗時は raw JSON を log にコピーして警告
    cp "${_log_file}.json" "$_log_file"
    log "Warning: JSON parse failed, using raw output"
}
```

session_id も jq で取得する:

```sh
jq -r '.session_id // empty' "${_impl_log}.json"
```

Preflight probe（Probe 5）が `--output-format json` サポートを検証する。非対応の claude CLI では `--output-format text` にフォールバックしてパイプラインを継続する。フォールバック時は警告ログを出力する。

## AC3, AC4: シグナル検出 2層化 + サイドカーライフサイクル

### 変更前の問題

COMPLETE/ABORT 検出は agent 出力の grep のみだった。agent がマーカーを出力しなかった場合、または出力フォーマットが変わった場合に検出が失敗する可能性があった。また stale なサイドカーファイルが前サイクルの値を残すリスクがあった。

### 変更後の動作

COMPLETE と ABORT の両方について 2層検出を実装した:

- Layer 1: `.harness/state/pipeline/.agent-signal` ファイルを読み、値が `COMPLETE` または `ABORT` を含むか確認する
- Layer 2: ログファイル（result テキスト）内の `<promise>COMPLETE</promise>` / `<promise>ABORT</promise>` マーカーを grep する

どちらか一方で検出できれば該当シグナルとして扱う。

サイドカーファイルは Inner Loop サイクル開始時にクリアする:

```sh
rm -f "${PIPELINE_DIR}/.agent-signal" "${PIPELINE_DIR}/.pr-url"
```

プロンプトテンプレート（`pipeline-inner.md`）に sidecar 書き込み指示を追加した:

```sh
# COMPLETE 時
echo COMPLETE > .harness/state/pipeline/.agent-signal
# ABORT 時
echo ABORT > .harness/state/pipeline/.agent-signal
```

### Inner Loop COMPLETE ゲーティング (AC3 の発展)

Codex WORTH_CONSIDERING 指摘に対応して、テスト通過だけでは Outer Loop に進まないよう制御を強化した。

変更前: テストが通過すれば即座に Outer Loop へ遷移する。

変更後: `run_inner_loop()` はテスト通過 + COMPLETE シグナルの両方が揃った場合のみ `return 0`（Outer Loop へ）を返す。COMPLETE なしでテストが通過した場合は `return 6` を返し、main loop の `case 6` ハンドラが `_inner_cycle` をインクリメントしてイテレーションを継続する。

## AC5: PR URL 3層検出

### 変更前の問題

PR URL を agent 出力の grep のみで取得していた。agent が URL を出力しない場合やフォーマットが変わった場合に取得が失敗する可能性があった。

### 変更後の動作

3層フォールバックを実装した:

- Layer 1: `gh pr list --head <branch> --state open --json url --jq '.[0].url'` による外部検証。github CLI が open PR の URL を直接返す。
- Layer 2: `.harness/state/pipeline/.pr-url` サイドカーファイル。`pipeline-outer.md` の指示に従い agent が書き込む。
- Layer 3: agent 出力ログの URL grep（既存のフォールバック）。

最初に取得できた URL を採用する。

## 安全な JSON 更新

### 変更前の問題

`ckpt_update` の呼び出しで `_pr_url` や `_new_session` を文字列連結で jq フィルタに埋め込んでいた。URL に `"` や `\` が含まれる場合に jq フィルタが壊れる。

### 変更後の動作

外部入力の埋め込みには `--arg` を使う:

```sh
# session_id の更新
ckpt_update --arg sid "$_new_session" '.session_id = $sid'

# PR URL の更新
ckpt_update --arg url "$_pr_url" \
    '.pr_created = true | .pr_url = $url | .status = "complete"'
```

`report_event` の JSON 構築も jq を使って安全にする:

```sh
_pr_event="$(jq -n --argjson c "$_cycle" --arg u "$_pr_url" \
    '{"cycle":$c,"url":$u}')"
report_event "pr-created" "$_pr_event"
```

## プロンプトスコープ修正

### pipeline-outer.md

Codex WORTH_CONSIDERING 指摘に対応。変更前は codex-review と PR 作成の指示まで含まれており、スクリプト側の処理フェーズと重複していた。変更後は docs-sync のみに限定し、以下を明示した:

```
Do NOT create pull requests or run codex review — those are handled by the pipeline.
```

### pipeline-review.md

レポート出力先を `.harness/state/pipeline/` に統一した（変更前は `docs/reports/` への出力指示が混在していた）:

```
Write findings to .harness/state/pipeline/self-review.md
```

## 軽微な修正 (AC7, AC9)

- `ralph-pipeline.sh --help` の exit code を 1 から 0 に修正した（`usage()` 関数内の `exit 1` → `exit 0`）
- `docs/quality/definition-of-done.md` にパイプラインモードのレポート配置を明記した: Inner Loop の作業レポートは `.harness/state/pipeline/`、最終アーティファクトは `docs/reports/`

## ガードレール修正 (b426fab, 9bb5727)

- `docs/quality/definition-of-done.md`: pipeline モードのレポート出力先の明確化
- `.claude/rules/post-implementation-pipeline.md` 追加: `/self-review → /verify → /test → /sync-docs → /codex-review → /pr` の順序を単一の正規ソースとして定義
- `docs/reports/audit-harness-2026-04-09.md`: pipeline step skip バグ（codex-review SKILL.md の再実行フローで `/sync-docs` が省略されていた）のポストモーテム
- `docs/quality/quality-gates.md`: pipeline モードのゲートを追加

## 検証サマリ

| フェーズ | 結果 | 主要エビデンス |
|---------|------|--------------|
| Self-review r1 | HIGH 2件 → 修正 | `ckpt_update --arg` 移行、phase 順序修正 |
| Verify r1 | PASS (AC1-AC11) | dry-run 全フェーズ通過、sh -n 全スクリプト通過 |
| Test r1 | PASS 12/12 | 全ユニット + 回帰テスト通過 |
| Codex triage | WORTH_CONSIDERING 2件 → 修正 | COMPLETE ゲーティング、pipeline-outer.md スコープ修正 |
| Self-review r2 | HIGH 1件 → 修正 (d607cdd) | report_event jq 化 |
| Verify r2 | PASS (全 AC 維持) | r2 変更点の個別確認を含む |
| Test r2 | PASS | — |

## 既知の残存 tech debt

- `ckpt_update()` が生の jq フィルタ式を受け取る汎用インターフェースのため、呼び出し側が外部値を文字列連結で埋め込みやすい。次回リファクタリング時に専用ヘルパー（`ckpt_set_field`）への移行を検討する。
- `pipeline-outer.md` の `echo "https://..." > .pr-url` 例がハードコードされており、agent が例 URL を書き込む可能性がある。`gh pr view --json url --jq '.url'` に変更することを推奨する。
- `pipeline-review.md` の inline fallback に run-static-verify.sh が存在しない場合のフォールバック説明がない。
- `ralph-orchestrator.sh` へのプランパス渡し未実装（`__PLAN_PATH__` が空になる）。別タスクとして tech-debt に記録済み。
