# Walkthrough: deprecation-notice-self-detect (PR #135)

- Date: 2026-07-24
- Plan: docs/plans/archive/2026-07-24-deprecation-notice-self-detect.md
- Issue: #134

差分は 588 行だが、実装本体は約 30 行。残りはパイプライン成果物(プラン、レポート、insight イベント)。

## 読み順

1. **`scripts/ralph`(+5/-2)** — 本体。deprecation notice の条件を `command -v ralph` の成否のみから、「解決結果が存在し、かつ `$0` と同一 inode でない(`-ef`)」に変更。`scripts/` を PATH に載せて `ralph` として起動した場合の自己誤検出を除外する。
2. **`templates/base/scripts/ralph`(+5/-2)** — 1 と byte-identical(`cmp` exit 0)。scaffold 配布用ミラー。
3. **`tests/test-ralph-deprecation-notice.sh`(+20)** — 回帰ケース5(自己検出)を追加。実リポジトリの `scripts/` を PATH 先頭に載せて `ralph status` を起動し、(a) notice 不在、(b) sibling-source 失敗なし、(c) status ヘッダー到達、の3点を assert。単体コピー方式は sibling source 失敗で false green になるため不採用(プラン Design decisions 参照)。

## 成果物(レビュー補助、読み飛ばし可)

- `docs/plans/archive/2026-07-24-deprecation-notice-self-detect.md` — プラン
- `docs/reports/self-review-*.md` / `verify-*.md` / `test-*.md` / `sync-docs-*.md` / `cross-review-triage-*.md` — パイプラインレポート
- `docs/insights/events/2026-07-24-deprecation-notice-self-detect.jsonl` — insight イベント
- `docs/tech-debt/README.md`(+1/-1) — shell CLI 退役エントリの Impact 記述を精度補正
