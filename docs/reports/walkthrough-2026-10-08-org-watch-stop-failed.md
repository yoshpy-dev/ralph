# Walkthrough: org-watch-stop-failed

- Date: 2026-10-08
- Plan: docs/plans/archive/2026-10-08-org-watch-stop-failed.md(この PR の最後のコミットで archive に移す)
- Branch: fix/org-watch-stop-failed(base main da4dccb0、#208 のマージ)
- Diff(HEAD 6f8c2dbb、この walkthrough を除く): 10 files、+660 / -11。うち `internal/` は 2 files、+209 / -9(テスト以外は `watch.go` だけ)。報告・insight・plan が +448
- PR: #209

## 何を変えたか

`ralph org watch` の deadman が、閉じられなかった stop(`stop_failed`)も leader の活動として数えるようにした。#208 で、閉じられなかった `stop` は `stopped` の代わりに `stop_failed` を書くようになった。ところが `leaderActivityEventCount` がこれを数えず、herdr が応答しない間に leader が座席を止めようとしても、deadman が leader の無活動として人に上げていた。

## 読む順

1. `2765f002` 本体。
   - `internal/org/watch.go` の `leaderActivityEventCount` の `case` に `EventStopFailed` を足した。今までの種類と同じく、`reason=watchdog_` の除外と org の絞り込みを通る。doc comment の (b) に、書き手と数える理由を足した
   - `internal/org/watch_test.go` の `fakeWatchHerdr` に `PaneGetErr` を足した。設定すると `PaneGet` が失敗し、本物の `Stop` が持ち主の確認で止まって `stop_failed` を書く。テストを 3 本足した
2. `3b219fc7` doc comment の直し(self-review の F-1・F-2)。「除外は `case` を共有するので掛かる」と「互換のための除外は別に要らない(v5.1.0 までは `stop_failed` を書かない)」を分け、残る窓を 2 文で書いた
3. `14ac4611` テストを 1 本追加(test の段)。警告の前からある `stop_failed` では警告が消えないこと。「警告を出すときの数え方だけ `stop_failed` を除く」書き換えを検出する
4. `b603603a` tech-debt。org-stop-all のテストの抜けの行の (c) を、`PaneGetErr` のある今の fake に合わせて直した。`leaderActivityEventCount` の行は `2765f002` で解決済みにしてある
5. 実装後のパイプライン: `290e7efb`・`1b609add` self-review(Merge 可、再確認で解消)、`841b0585` verify(pass)、`2d98e7be` test(pass)、`6f8c2dbb` cross-review(指摘 0 件)

## 計画からの逸脱

- なし。Codex plan advisory の指摘(更新をまたぐと、保存済みの警告が黙って消えうる)は、ユーザーが「軽く計画を直す」を選んだ。そのため前提と rollout の注意に書いた
- 実装の途中で、前提の 3 つ目では塞げない窓が 1 つ見つかった。v5.1.0 の watch が動き続けたまま、新しい stop が `stop_failed` を書く場合で、計画の本文は承認の digest の範囲なので直していない。進捗の記録、doc comment、PR 本文の既知の穴に書いた

## 変えていないもの

- deadman の判定の仕組み(警告を出したときの数と、数え直した数の差で見る)
- ほかに数えるイベントと、台帳の形

## 下流への届き方

| 所有権 | ファイル | upgrade での扱い |
|---|---|---|
| バイナリ | `internal/org/watch.go` | バイナリを更新すると効く |

#208 と同じ release に入れる。更新の前に `ralph org watch` を止めておくと、残る窓も起きない。

## 確かめたこと

- `./scripts/run-static-verify.sh` は rc 0(golangci-lint 0 件)。`./scripts/run-test.sh` は shell 2,047 件、Go 8 パッケージで、すべて pass。`go test -race` も pass
- mutation 9 件すべてをテストが検出した(この直しを戻す、`reason=watchdog_` の判定を外す、org の絞り込みを外す、偽の herdr が `PaneGetErr` を無視する、ほか 5 件)
- #208 を含む release はまだない(`git tag --contains da4dccb0` は空、最新は v5.1.0)。v5.1.0 のコードに `stop_failed` はない

## 確かめていないこと

- 更新をまたぐ窓の再現と、v5.1.0 の watch が `stop_failed` を含む台帳をどう扱うか
- Disband・StopAll が書く `stop_failed` と、pane の close の失敗で書く `stop_failed` を watch のテストで作ること(数え方はイベントの種類だけで決まる)
- 本物の herdr での動作
