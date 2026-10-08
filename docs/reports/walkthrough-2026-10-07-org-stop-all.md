# Walkthrough: org-stop-all

- Date: 2026-10-08
- Plan: docs/plans/archive/2026-10-07-org-stop-all.md(この PR の最後のコミットで archive に移す)
- Branch: feat/org-stop-all(base main f423f230)
- Diff(HEAD 48ce82d0、この walkthrough を除く): 35 files、+7,492 / -165。報告・insight・plan・evidence・ミラー(`.agents/`、`templates/`)を除くと 23 files、+6,376 / -138。そのうちテストが +4,508、テスト以外の Go が +1,807
- PR: #208

## 何を変えたか

`ralph org stop --all` と `ralph org disband --all` を足した。1 座席の `stop` は C-c のあと herdr の pane を閉じてプロセスを終わらせ、`disband` は org の herdr workspace も閉じる。閉じる前に、tab と workspace の label で、台帳の id が今も ralph の作ったものを指すかを確かめる。閉じられなかった座席は `stopped` にせず `stop_failed` を書き、打ち直しで拾う。コマンドを打った pane と workspace は最後に閉じ、閉じるのに失敗したら台帳を「動いている」に戻す。

## 読む順

1. `0f2e9dfd` S1 driver(`internal/org/driver/herdr.go`)。`PaneClose`・`WorkspaceClose` と、herdr の `*_not_found` を `NotFound()` で区別するエラーの型
2. `815f5fc8` S2 Stop(`internal/org/verbs.go`)。C-c → pane を閉じる → agmsg から外す → 記録、の順。閉じられないときは `stop_failed`(`internal/org/seat.go` に足した、状態を変えないイベント)。呼び出しごとの期限(`callWithTimeout`、既定 10 秒)と `--force`
3. `7853d8a9` S3 Disband。座席が 1 つでも残れば workspace を閉じない。`org_workspace_closed` を書き、次の spawn は `resolveWorkspace`(`internal/org/spawn.go`)で workspace を作り直す。ExecRunner の `WaitDelay`
4. `47f36dad` S4 全 org(`internal/org/verbs_all.go`、新規)。`StopAll`・`DisbandAll`・`orgsToDisband`。自分の座席と org は、ほかがすべて成功したときだけ最後に処理する
5. `9f8c2783` S5 CLI(`internal/cli/org.go`)。`--all`・`--force` のフラグ、同時指定の検査(state dir を解決する前に判定する)、出力と終了コード、自分の pane と workspace を最後に閉じる `closeDeferredSelf`
6. `e2189d79` S6 文書。leader の雛形の締めの順(stop → report → disband)、`/org` skill、README、仕様の FR-2、recipe、`docs/evidence/herdr-pane-close-2026-10-07.md`
7. `05977322` S7 持ち主の確認(`confirmSeatPane`・`confirmOrgWorkspace`、driver の `PaneGet`・`TabGet`・`WorkspaceGet`)と、`ErrWaitDelay` を成功として扱うこと。1 回目の self-review の F-1・F-2 を受けて足した
8. `53807a82` S8 自分の close の補償(`CloseDeferredSelfPane`・`CloseDeferredSelfWorkspace`、`reactivateSeat`・`reopenWorkspace`)。1 回目の cross-review の ACTION_REQUIRED を受けて足した
9. 実装後のパイプライン(2 回)
   - 1 回目: `a103a05f`・`eb7bf5e2` self-review(S7 の前と後)、`de154798` verify(pass)、`f63ae025` mutation で見つけたテストの穴、`ad3baa72` test(pass)、`223258eb`・`2c97407c` /sync-docs、`643b200e` cross-review(ACTION_REQUIRED 1)
   - 2 回目: `8bfc0f40` self-review(Merge 可、LOW のみ)、`e3bef918` verify(pass)、`a6cc5fbf` S8 のテストの穴、`8e3d951e` test(pass)、`3f52d676`・`de3461ae` /sync-docs、`48ce82d0` cross-review(ACTION_REQUIRED 1、上限に達したので既知の穴として PR に進んだ)

## 計画からの逸脱(すべてユーザーが承認し直した)

- S7: 台帳の id が、herdr のセッションの保存ファイルが失われたあとに別の pane や workspace を指しうる(隔離サーバーで確認)。閉じる前に label で持ち主を確かめる段を足した(`e5de53d2`、digest 15eb2a79e2f6)
- S8: 自分の pane や workspace の最後の close が失敗すると、打ち直しが効かなかった。補償のイベントを書き足して台帳を戻す形にした。詳細の文字列で「未確認」を表す案は、`Roster` の決まりの外に第 2 の状態を作るので採らなかった(`5f26af07`、digest 2f2cfde39e9f)

## 変えていないもの

- 台帳の形。イベントの種類(`stop_failed`、`org_workspace_closed`)と詳細の文字列が増えるだけ
- `--all` を付けない `stop` / `disband` は、今までどおり `--org-id` が要る
- spawn が失敗したときの補償(C-c だけ)、agmsg の team の削除、`ralph status`、`ralph org watch`

## 下流への届き方

| 所有権 | ファイル | upgrade での扱い |
|---|---|---|
| バイナリ | `internal/org/`、`internal/cli/`、leader の雛形(`internal/org/prompts/leader.md`、go:embed) | バイナリを更新すると効く |
| core | `/org` skill(`.claude/skills/`、`.agents/skills/`)、`docs/recipes/codex-seat-permissions.md` | 置き換わる |

更新すると `stop` が pane を閉じるようになるので、止める座席の画面のログが要るときは、先に `ralph org read` で読む。古いバイナリは `org_workspace_closed` を読まない。この PR のバイナリで disband した org_id を古いバイナリから spawn すると、閉じた workspace に tab を作ろうとして失敗するので、別の org_id を使う。

## 確かめたこと

- `./scripts/run-static-verify.sh` は rc 0(golangci-lint 0 件)。`./scripts/run-test.sh` は shell 2,047 件、Go 8 パッケージで、すべて pass。`go test -race` も pass
- mutation: 1 回目は 81 件中 80 件、2 回目は 64 件中 63 件をテストが検出した。残りはどちらも等価な書き換え
- herdr 0.7.5 の隔離したサーバーで、pane と workspace の close、見つからない id の応答、id の振り直し、`herdr agent start` のあとも label が保たれることを確かめた(`docs/evidence/herdr-pane-close-2026-10-07.md` の Run 1〜5)
- HEAD のバイナリで、`--all` と `--org-id` の同時指定の拒否、空の台帳への `--all --dry-run`、補償の前後での `status` と `--all` の対象を確かめた(verify の報告)

## 確かめていないこと

- 本物の herdr で `ralph org spawn` から stop / disband までを通しで流すこと。close の失敗は偽の driver と stub で起こしている
- codex の座席の C-c の動き(pane を閉じればプロセスは終わるので、結果は driver によらない)
- 2 回目の cross-review の 3 件(PR 本文の「既知の穴」と tech-debt の 167 行・169 行)。watchdog が `stop_failed` を数えない件は main からの後退なので、続く PR で直す
- `3f52d676` の文字列の変更は、2 回目の self-review・verify・test のあとに入った(2 回目の cross-review は見ている)
