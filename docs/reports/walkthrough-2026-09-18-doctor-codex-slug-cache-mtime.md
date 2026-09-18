# Walkthrough: doctor-codex-slug-cache-mtime

- Date: 2026-09-18
- Plan: docs/plans/archive/2026-09-18-doctor-codex-slug-cache-mtime.md(PR 作成時に active から移動)
- Issue: #159(Closes)。#156 の観測手順の記述も更新(Refs)
- Branch: feat/doctor-codex-slug-cache-mtime(base: main @ 26ca767)
- 差分規模: 13 files / +1042 −8(コードは Go 2 ファイル。残りは skill 4 面・spec・plan・5 本の pipeline レポート)

## 読む順番

1. `internal/cli/doctor_codex_models.go`
   - `codexCacheStaleAfter`(24h)と test seam `codexCacheBetweenReadAndStat`(非公開、本番 nil)。
   - `readCodexModelsCache`: `os.Open` した 1 ハンドルで `Stat` → `io.ReadAll` → `Stat`。前後の `ModTime` / `Size` が一致すれば mtime を採用、不一致なら `changed=true`(mtime ゼロ)、`Stat` 失敗なら mtime ゼロ + `changed=false`。rename で差し替えられた場合はハンドルが旧 inode を指すので内容と mtime は旧側で一貫、in-place 書き換えだけを前後不一致で検知(同サイズ・同 mtime tick の書き換えは検知しない。best-effort として許容)。seam は最初の `Stat` 失敗の return より後、2 回目の `Stat` の直前で呼ぶ。
   - `formatCacheAge`(`Nm` / `Nh` / `Nd`、負値は `0m`。表示粒度の `hoursPerDay` は閾値の定数と別)と `codexCacheFreshnessClause`(`changed` 優先 → ゼロ mtime は空 → `(cache written <UTC RFC3339>, <age> ago)` + `age > 24h` なら stale 注記)。
   - `checkCodexModelSlugs`: `os.ReadFile` を `readCodexModelsCache` に置換し、warn / pass の Detail 末尾に clause を付ける。info の 5 経路と Status は不変。
2. `internal/cli/doctor_org_test.go` — 新規 6 テスト: 既知 mtime(`os.Chtimes`、秒揃え)の RFC3339 完全一致(5 分前で pass、48h 前で warn + stale)、pass でも stale 注記、`TestFormatCacheAge` テーブル、`TestCodexCacheFreshnessClause` テーブル(注入した `now` で 24h ちょうど = 非 stale / 24h+1s = stale、`changed` 優先、ゼロ mtime = 空)、seam で読み取り中に cache を書き換えて `freshness unknown` になり判定は読んだバイト列に基づくこと。
3. `.claude/skills/org/SKILL.md`(+ 3 ミラー)と `docs/specs/2026-08-01-org-runtime.md` (d) — 「更新も鮮度確認もしない」→「更新はしない(書き込み時刻を UTC で Detail に出し、24 時間超は stale 注記)」。観測手順は `cache written`(UTC、JST +9h)の確認、更新直後は `<age> ago` が `0m` になることを明記。

## コミット単位

| SHA | 内容 |
|---|---|
| fd1be42 | Slice B: skill 4 面 + spec (d)(index 衝突回避のため先行) |
| 3b0fdf4 | Slice A: Go 実装 + テスト 5 件(implementer) |
| 8f3f72b | self-review M1・L2〜L5(UTC 補足、clause のテーブルテスト、doc の機構明記、seam 配置、`hoursPerDay`) |
| 5143e3b | self-review follow-up 4 点(テスト挿入位置、行幅 ×2、`stat` 句の位置) |
| その他 | plan の進捗・逸脱記録、self-review / verify / test / sync-docs / cross-review レポート |

## 設計判断(plan Design decisions より)

- Status は変えない(warn は warn、pass は pass)。best-effort な Check の性質を維持し、stale は注記に留める。pass にも stale 注記を付ける(古い cache は「今も存在する」証拠として弱い)。
- Codex plan advisory の 2 所見を採用: MEDIUM-1「`ReadFile` と `Stat` を分けると内容と鮮度が別バージョンになりうる」→ 1 ハンドル + 前後 `Stat`、MEDIUM-2「表示時刻が mtime である証明がない」→ 既知 mtime の完全一致 assert。
- self-review M1(観測手順が UTC 表示との時差を考慮していない)を受け、spec と skill に UTC と `<age> ago` の補足を追加。
- Known gap: 実際の `f.Stat()` 失敗経路は fs の fault-injection seam を入れないため未テスト(コードレビューで確認)。

## レビューで特に見てほしい箇所

- `readCodexModelsCache` の前後 `Stat` 比較(`ModTime` + `Size`)が best-effort として十分か。
- Detail 文言(`—` の使用は既存の info 文言と同じ慣例)。
- test seam が本番経路に影響しないこと(nil チェックのみ)。
