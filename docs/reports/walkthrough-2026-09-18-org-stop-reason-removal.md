# Walkthrough: org-stop-reason-removal

- Date: 2026-09-18
- Plan: docs/plans/archive/2026-09-17-org-stop-reason-removal.md(PR 作成時に active から移動)
- Issue: #153
- Branch: refactor/org-stop-reason-removal(base: main @ c6b8127)
- 差分規模: 10 files / +835 −47(うちコード・テストは 3 files。残りは plan、tech-debt 行、5 本の pipeline レポート)

## 読む順番

1. `internal/org/verbs.go` — `StopParams.Reason` フィールドと doc comment、`Stop` の `" reason=<Reason>"` Details 追記を削除。残る `StopParams{...}` リテラル(`ralph org stop`、`Disband`)はいずれも Reason を使っていなかったので、`stopped` イベントの Details は従来と byte 単位で同一。
2. `internal/org/watch.go` — **コードは不変**(`git diff main -- internal/org/watch.go` はコメント行のみ)。`leadActivityEventCount` の doc comment (b) を書き換え、`reason=watchdog_` 除外を「dormant / 将来の enforcement 用」ではなく「PR #152 以前の watchdog が書いた manifest との互換ガード」と位置づけた。ガードが必要なのは、ベースラインに無く recount にだけ現れる cutoff がある 2 ケース((a) 永続化された pending alert の `ManifestLen` をまたぐアップグレード境界、(b) 新旧バイナリ混在窓)であること、alert 前の cutoff はベースラインと recount で相殺されることを明記。
3. `internal/org/watch_test.go` — 合成 `stopped` を `Stop(Reason)` で作っていた 2 テストを、旧 watchdog が書いた形の Details を `Manifest.Append` で直接書く legacy fixture に改名・書き換え(`…LegacyWatchdogStopEvent…`、`…_LegacyWatchdogStopDoesNot`)。手動停止テストから `Reason` を外した。新規 `TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop` は手書きの status JSON で「保存済みベースライン + 旧 cutoff + 新規イベントなし」を再現し、subtest (i) でガードを外した実装なら alert が誤って消えて fail する構造(ベースラインは legacy イベント追加の **前** に計算)。subtest (ii) は `sent` が alert を消す sanity check。
4. `docs/tech-debt/README.md` — `StopParams.Reason` の行を `~~` + `(RESOLVED 2026-09-18 in refactor/org-stop-reason-removal)` でクローズ。Trigger セルに「プロデューサ側のみ削除、reader 側は legacy 互換で残す」理由(Codex plan advisory HIGH-1)を記録。

## コミット単位

| SHA | 内容 |
|---|---|
| bce892c | Slice A: 項目 1・2・3a〜3d(Go 3 ファイル) |
| cb70f13 | Slice B: tech-debt 行クローズ |
| 72b89d5 | self-review M1・L1〜L5 修正 |
| ec9a226 | self-review N1〜N4(コメント一句 ×4) |
| その他 | plan の進捗・逸脱記録、self-review / verify / test / sync-docs / cross-review レポート |

## 設計判断(plan Design decisions より)

- フォーク 1: `--reason` を配線せず削除(本番プロデューサなし、spec に enforcement の記述なし)。
- フォーク 2: Codex plan advisory の HIGH-1 を受け、削除範囲をプロデューサ側に限定。`checkDeadman` は「alert 時に保存した `ManifestLen`」と「manifest 全件の数え直し」を比較するため、数え方の規則を変えると、旧 cutoff を含む manifest で保留中 alert がアップグレード後に誤って消える(dedupe で再 alert も抑止、revert で戻らない)。
- issue の受け入れ条件「`grep 'Reason' … watch.go` が空」は、無関係な escalation レコードの `Reason` JSON フィールドに当たるため文字どおりには満たせない。StopParams 由来の参照に限定した grep に読み替えた(plan AC-1)。

## レビューで特に見てほしい箇所

- `watch.go` のコメントが実装と一致しているか(除外は lifecycle 5 種すべてに掛かる、call site は sendAlert/checkDeadman の 2 箇所)。
- 新規テストの fixture で `lead_agent_get: ""` と `history_lead_lines: -1` が probe/history 経路を無効化し、manifest の数え直しだけで判定が決まること。
