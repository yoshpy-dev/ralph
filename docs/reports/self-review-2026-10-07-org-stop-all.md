# Self-review report: org-stop-all

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Branch: feat/org-stop-all(HEAD 05977322)
- Reviewer: reviewer subagent (Claude)、S7 後の再レビュー(初回は a103a05f、HEAD e2189d79)
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、typo、null 安全、デバッグ用コード、秘密情報、例外処理、安全性、保守性)。対象は `git diff origin/main...HEAD` の全体(28 ファイル、+5678/-165)で、S7(05977322、18 ファイル)は全行読んだ。重点は (1) C-c を送る・閉じる経路がすべて持ち主の確認を通るか、(2) herdr の get の失敗が「閉じ済み」と読まれうるか、(3) テストの fake や stub が本物の herdr の挙動を隠していないか。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査は実行していない。

## 前回からの変化

| 前回の指摘 | 今回 | 根拠 |
| --- | --- | --- |
| F-1 MEDIUM(台帳の id が別の pane / workspace を指しうる) | 解消 | C-c と close の呼び出し 5 か所すべてが、直前に `confirmSeatPane` / `confirmOrgWorkspace` を通る(下の表)。id の振り直しは evidence の Run 4 で実機確認済み(保存ファイルがないと `w1` から振り直される)。残る点は F-1 の下の「残余」 |
| F-2 LOW(`ErrWaitDelay` を成功の exit 0 でも失敗にする) | 解消 | `internal/org/driver/driver.go:60` が `errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil` で成功扱いにし、`TestExecRunner_Run_SuccessNotFailedByGrandchildHoldingPipes` が成功側を見る |
| F-3 LOW(`--all` 全体に上限がない) | 継続(小さくなった) | herdr が返事をしないとき、1 座席の費用が C-c + close の約 20 秒から、最初の get の 10 秒に減った(確認が失敗すると C-c も close も送らない)。全体の上限は今もない |
| F-4 LOW(`Disband` の読み取りから `disbanded` までの窓) | 継続 | 1 座席の呼び出しが get 3 回ぶん増え、窓はわずかに広がった |
| F-5 LOW(`withPrefixOnce`) | 継続 | S7 は触っていない |
| F-6 LOW(`Stop` の長さ、自分の判定が複数箇所) | 継続 | `Stop` は 117 行から 109 行(`verbs.go:831-939`)。自分の pane / workspace の判定は 5 か所のまま |
| F-7 LOW(`NewHerdrError` の export) | 継続 | S7 は触っていない。fake が使う型は `driver.NewHerdrError` のまま |
| F-8 LOW(skill 表のセルが長い) | 継続(悪化) | S7 が `stop` と `disband` の行に 1 文ずつ足し、`stop` のセルは 1394 字から 1725 字になった |

新しく見つけたのは LOW 2 件(F-9、F-10)。CRITICAL、HIGH、MEDIUM はない。

## S7 についての 3 つの確認

### 1. 持ち主の確認を通らずに C-c を送る・閉じる経路があるか

`PaneSendKeys`、`PaneClose`、`WorkspaceClose` の非テストの呼び出しを `grep` で全部出して、直前の確認を見た。

| 呼び出し | 場所 | 直前の確認 |
| --- | --- | --- |
| C-c | `verbs.go:965`(`stopSeatPane`) | `confirmSeatPane`(`:953`)。失敗なら C-c も close も送らずに戻る |
| pane close | `verbs.go:971`(`stopSeatPane`) | 同上 |
| 自分の pane の close | `verbs.go:1017`(`CloseDeferredSelfPane`) | 台帳から座席を引き直して `confirmSeatPane`(`:1010`) |
| 自分の workspace の close | `verbs.go:1059`(`CloseDeferredSelfWorkspace`) | 台帳から org を引き直して `confirmOrgWorkspace`(`:1052`) |
| workspace close | `verbs.go:1633`(`closeOrgWorkspace`) | `confirmOrgWorkspace`(`:1623`) |

`--force` の枝(`notClosedNote(..., true)`)はどれも記録だけで、閉じる呼び出しを足していない。`--dry-run` は driver を呼ばない。`Stop` の自分の pane の判定(`:959`)は確認の後ろに置かれたので、`HERDR_PANE_ID` と一致するだけの id は先送りにもならない。

この diff の外に 1 件残っている。`internal/org/spawn.go:1845`(`compensatePaneCtx`)は、`compensateStale`(`:1811`)から、前回の spawn が終わらなかった座席の台帳上の pane id に C-c を送る。この経路は main にもあり、S7 は触っていないが、台帳の id に確認なしで C-c を送る点は同じ。起きる条件は、終わらなかった spawn の記録が残っていて、その間に herdr の id が振り直され、同じ座席を spawn し直すこと。C-c だけで close はしない。この PR で直す必要はなく、tech-debt に 1 行足すのが妥当と考える。

### 2. get の失敗が「閉じ済み」と読まれうるか

「閉じ済み」(`gone`)になるのは、`PaneGet` / `WorkspaceGet` が、envelope として読めて code が `*_not_found` の応答を返したときだけ(`verbs.go:677`、`:723`)。それ以外は失敗として返り、C-c も close も送られない。

- 期限切れ、接続拒否、code が別の envelope、exit 0 の空出力や非 JSON、途中で切れた JSON、`result.pane.tab_id` などの欠落は、すべて `herdrGetResult` でエラーになる(`herdr_test.go:557-617` の表に 11 ケース。bare-id に落とす `WorkspaceCreate` 型の寛容さを持たない)。
- `ExecRunner` が `ErrWaitDelay` を成功にして途中までの stdout を返しても、途中で切れた JSON は `parseHerdrEnvelope` が malformed として返し、`herdrGetResult` の `case envErr != nil` でエラーになる。F-2 の修正が get の失敗側を緩めることはない。
- pane は見つかったのに tab が `tab_not_found` のときは、`confirmSeatPane` が `%v` で包み直して `%w` の連鎖を切る(`verbs.go:693`)ため、後段の `isNotFound` が「閉じ済み」と読めない。`TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` の `tab gone although the pane was found` が `isNotFound(result.CloseErr) == false` を確かめている(`verbs_test.go:2215`)。この守りは `%v` 1 文字に依っている(F-9)。
- pane は見つかったのに workspace が `workspace_not_found` のときも失敗になる(`verbs.go:704`)。この枝を通るテストはない(下の 3)。

残る読み違いは、本物の `pane_not_found` が「別の herdr から見た答え」だった場合。`HERDR_SESSION` の違う shell から `stop` を打つと、本物の pane が動いているのに自分のセッションには id がなく、`stopped` が書かれる。S7 より前から close 側にあった挙動で、S7 が増やした経路ではない。herdr のセッション間で id の空間が分かれているかは試していない(未確認)。

### 3. テストの fake や stub が実際の挙動を隠していないか

- `fakeHerdr`(`spawn_test.go:27-211`)は、`TabCreate` と `WorkspaceCreate` が label を記録し、get がそこから答える。spawn を通したテストは、確認を作りで通る。確認が効くことは、負のテスト(`verbs_test.go:2165-2476`)が map を書き換えて見ている。本物の herdr の JSON の形は、fake ではなく `herdr_test.go:41-45` の 3 つの定数(evidence の Run 4 の応答と同じ文字列)が受け持ち、形は一致している。
- `fakeWatchHerdr`(`watch_test.go:82-95`)は、get をすべて not-found で返し、`Stop` を「閉じ済み」の枝に通す。コメントにその旨がある。watch のテストが C-c と close を見ていないので、隠れる挙動はない。
- CLI の `herdrStub`(`org_test.go`)は、`workspace create` と `tab create` が label をファイルに残し、get がそれを返す。`seedSeat` は pane の workspace を `ws-of-<orgID>` に固定し、`seedWorkspace` が記録した id とは別にする(コメントに「label だけ見る」とある)。本番の確認は pane の workspace id と台帳の workspace id を比べないので、この切り離しで通るテストが本番で落ちることはない。
- 通っていない枝: `confirmSeatPane` の `wsGone`(`verbs.go:704-705`)と、`CloseDeferredSelfPane` の `no seat recorded on it`(`:1008`)。`/test` で拾える。

## Evidence reviewed

- `git show 05977322` の全行(driver.go、herdr.go、spawn.go、verbs.go、verbs_all.go、4 つの skill ミラー、evidence、テスト 8 ファイル)と、`internal/org/verbs.go` の `confirmSeatPane`、`confirmOrgWorkspace`、`Stop`、`stopSeatPane`、`CloseDeferredSelfPane`、`CloseDeferredSelfWorkspace`、`lastSeatOnPane`、`lastOrgOfWorkspace`、`Disband`、`disbandOwnLast`、`closeOrgWorkspace` の現在の全文。`verbs_all.go` の `StopAll`。`herdr.go` の `parseHerdrEnvelope`、`herdrGetResult`、`wrapHerdrRunError`、`PaneGet`、`TabGet`、`WorkspaceGet`。
- `closeOrgWorkspace` の分岐(確認失敗、gone、self、close 成功、not-found、close 失敗、`Force`)ごとに、台帳に書く値と `DisbandResult` の入る先を追った。`self && gone` は `ClosedWorkspaces` に入り、`self && !gone` だけが `DeferredSelfWorkspaceID` を立てて `DeferredSelfPaneID` を空にする。
- `spawn.go` の `TabCreate(ctx, workspaceID, p.Cwd, p.SeatID)` と `WorkspaceCreate(ctx, p.Cwd, p.OrgID)` は main にもあり、label の付け方は S7 で変わっていない。
- `strings` で読んだ `herdr` のバイナリ(起動はしていない)に `custom_name`(tab と workspace のスナップショット)がある。おそらく `--label` は利用者が付けた名前として保たれる。未確認です(下の F-1 の残余)。
- デバッグ用コード、`TODO` / `FIXME`、`/Users/` などの固定パスを追加行から `grep` した。実コードの追加行にはない。
- ミラー: `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面は、追加・削除の行の md5 が 4 面とも同じ(77f971fe…)。`docs/recipes/codex-seat-permissions.md` と `templates/base/` の写しも同じ(f4f85df9…)。追加行に `internal/`、`cmd/`、`issue #`、`PR #` はない。evidence に `$HOME` の絶対パスはない。
- `docs/tech-debt/README.md` は差分に含まれていない。F-3〜F-10 の行は `/sync-docs` が足す計画(下の「Tech debt identified」)。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 解消 | security | 前回は MEDIUM。台帳の id を、それが今も ralph の作ったものを指すか確かめずに閉じていた。S7 で C-c と close の全経路が label の確認を通るようになり、確認が失敗した座席は `stop_failed`、workspace は失敗として、`--force` でも閉じない。残余が 3 つある。(a) 確認は tab の label が座席 id、workspace の label が org_id であることだけを見る。利用者が herdr で tab や workspace の名前を変えると、本物の座席でも `stop` が失敗し続ける(計画の「判断」節で受け入れ済み)。このとき `--force` は `stopped` を記録するだけで pane は閉じないが、失敗の文言(`workspace "w1" is not org_id "orgA"'s: herdr labels it "x"`)は `--force` に触れず、退路が `--help` の Long にしかない。(b) evidence の Run 4 は、tab の label が `tab create --label` のとおり返ることを、エージェントが動いていない tab で見ている。`herdr agent start` のあとも label が保たれるかは確かめていない(未確認)。保たれないなら、本物の座席がすべて確認で止まる。閉じる側に倒れるのでデータは失われないが、`stop` と `disband` が `--force` なしでは使えなくなる。(c) 上の「2」の末尾のとおり、別セッションからの `pane_not_found` は「閉じ済み」と読まれる | `internal/org/verbs.go:658-731`(確認)、`:953-981`、`:1010`、`:1052`、`:1623`。`docs/evidence/herdr-pane-close-2026-10-07.md` の Run 4。`docs/plans/active/2026-10-07-org-stop-all.md:74` | merge の前に、(b) を隔離したサーバーで 1 回確かめて evidence に足す(`ralph org spawn` した座席の tab に `herdr tab get` を打つ)。直すとしたら、失敗の文言の末尾に「`--force` で記録だけ残せる」を 1 文足す。(a)(c) は受け入れてよい |
| F-2 解消 | exception-handling | 前回は LOW。`ErrWaitDelay` の成功扱いは入った | `internal/org/driver/driver.go:55-62`。`driver_test.go` の `TestExecRunner_Run_SuccessNotFailedByGrandchildHoldingPipes` | なし。`Run` の doc comment(`:37-40`)は失敗側だけを説明しているので、次に触るときに 1 文足す程度でよい |
| F-3 LOW | maintainability | `--all` の全体に上限がない。herdr が返事をしないとき、1 座席は最初の get の 10 秒で `stop_failed` になり(前回は約 20 秒)、座席の上限 30 で約 5 分、出力は最後にまとめて出る。接続拒否なら即座に失敗するので、返事をしない場合だけの問題 | `internal/org/verbs.go:603`、`:949-981`。`internal/org/verbs_all.go:129-141`。`internal/cli/org.go:844`(`printStopAllResult` は最後に出力) | 受け入れてよいなら、`stop --all` の help に「herdr が応答しないと 1 座席あたり約 10 秒かかる」と 1 行。直すなら、連続して期限切れになった回数が一定を超えたら残りを打ち切る |
| F-4 LOW | exception-handling | `Disband` は台帳を 1 回読んだあとで座席を止め、最後に `disbanded` を書く。読んだあとに別プロセスの `spawn` が足した座席は、止められないまま非 active になる。1 座席あたりの呼び出しが get 3 回ぶん増え、窓は広がった(前回と同じ経路) | `internal/org/verbs.go:1541`(`Manifest.Read`)、`:1578`(`appendDisbanded`)。`internal/org/manifest.go:184` | `disbanded` を書く直前に台帳を読み直して、最初の読みになかった active な座席があれば書かずに失敗として返す。直さないなら台帳に 1 行 |
| F-5 LOW | maintainability | `withPrefixOnce` が文字列の先頭を見て接頭辞を足す。原因は `Stop` の error の包み方のそろわなさで、台帳への追記失敗だけが接頭辞なしで返る | `internal/cli/org.go:762`、`:793-801`。`internal/org/verbs.go:934`(`result.Err = err`)と `:930`(`org: stop: record %s: %w`) | `result.Err = err` を `fmt.Errorf("org: stop: record %s: %w", event, err)` にそろえ、`withPrefixOnce` をやめる |
| F-6 LOW | maintainability | (a) `Stop` は 109 行(コメントと空行を除くと 73 行)。agmsg の Leave を `stopSeatPane` と同じ形の関数に出せる。(b) 「自分の pane / workspace か」の判定が 5 か所に別々にある。1 か所だけ直すと、自分の pane を後回しにする順序が静かに壊れる。S7 で `Stop` 側の判定が確認の後ろに移り、順序を決める側(`Disband`、`StopAll`)と、先送りを決める側(`stopSeatPane`)が別の場所になった。(c) `DisbandResult.recordStop` / `failSeat` と `StopAllResult.recordStop` / `failSeat` が同じ分岐の写し | `internal/org/verbs.go:831-939`、`:959`、`:1553`、`:1565`。`internal/org/verbs_all.go:134`、`:352`。`verbs.go:1444-1461` と `verbs_all.go:73-93` | (b) は `o.ownPaneID()` / `o.ownWorkspaceID()` のような読み取りと比較の 1 組の関数にまとめる。(a)(c) は急がない |
| F-7 LOW | maintainability | `NewHerdrError` は、別パッケージのテスト(`internal/org` の `fakeHerdr`、`fakeWatchHerdr`)が「見つからない」の応答を作るためだけに export されている | `internal/org/driver/herdr.go:64`。`internal/org/spawn_test.go:180`、`watch_test.go:86-94` | 急がない。直すなら、`NotFound() bool` を持つ小さな型を fake 側に置く |
| F-8 LOW | readability | `/org` skill の表の `stop` と `disband` の行が、1 つのセルに 8〜11 の挙動を詰めている。S7 で `stop` は 1394 字から 1725 字、`disband` は 1494 字から 1585 字になり、表のセルとして読みにくい(4 面の写しで同じ) | `.claude/skills/org/SKILL.md:167-168` | 表のセルは 1〜2 文にして、詳細は表の下の節に出す。4 面の写しは `scripts/sync-skills.sh` で同期する |
| F-9 LOW | maintainability | S7 は `tab_not_found` を `NotFound()` と `IsNotFound` の対象に加えた。この述語は `PaneClose` / `WorkspaceClose` の「閉じ済み」判定にも使われる。一方で、`confirmSeatPane` は tab の not-found を失敗として扱いたいので、`%v` で包み直して連鎖を切っている。`%v` を `%w` に直すと、pane は見つかったのに tab が消えた状態が「閉じ済み」と読まれ、動いている座席に `stopped` が書かれる。今は `verbs_test.go:2184-2186` と `:2215` がこれを固定しているので、壊れればテストが落ちる。述語の意味が呼び出しごとに違うことが保守の負担になっている | `internal/org/driver/herdr.go:49-74`(`HerdrCodeTabNotFound`、`NotFound`)。`internal/org/verbs.go:690-694`。`internal/org/verbs_test.go:2215` | `NotFound()` を pane と workspace の 2 つに戻し、`TabGet` が返す tab の not-found には別の判定(`TabNotFound()`)を持たせる案がある。急がない |
| F-10 LOW | readability | コメントを書き換えたとき、折り返しを直し切っていない箇所がある。`spawn.go` の `DriverCallTimeout` のコメントに 98 字の行(周りは 70 字前後)、`verbs.go` の `DisbandResult` と `Disband` の doc に 32 字、55 字、13 字の行、`herdr.go` の `IsNotFound` の doc に 24 字の行が残る。gofmt も vet も通るので、機械では拾えない | `internal/org/spawn.go:214`。`internal/org/verbs.go:1428`、`:1504`、`:1520`。`internal/org/driver/herdr.go:80` | 次に `internal/` を直すときに、その段落だけ折り返し直す。この指摘だけのために再実行はしない |

## Positive notes

- 確認の置き場所が 1 つの関数対(`confirmSeatPane`、`confirmOrgWorkspace`)にまとまり、`Stop`、`Disband`、後回しの close の 3 つが同じ判定を使う。呼び出し 5 か所すべてに直前の確認があり、確認を飛ばせる枝がない。
- 失敗が閉じる側に倒れる。get の期限切れ、接続拒否、読めない応答、tab や workspace の消失、label の不一致は、どれも C-c も close も送らず、`stop_failed` か workspace の失敗になる。`--force` でも閉じずに記録だけで、`Details` に `(forced)` と理由が残る。
- 後回しの close(`CloseDeferredSelfPane` / `CloseDeferredSelfWorkspace`)が、台帳から座席と org を引き直して確認をやり直す。`Stop` が返したあとに label が変わっても閉じない。引き直せない(座席の記録がない、台帳が読めない)ときも閉じずに、pane の id を名指ししたエラーにする。
- driver の get が、成功の応答も JSON の envelope と必須項目の存在を要求する。「読めない応答を成功として通さない」という理由がコメントに書かれている(`herdr.go` の `herdrGetResult`)。
- `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` は 4 つの不一致 x `Force` の有無を表で見て、C-c と close の呼び出しがないこと、`isNotFound` が偽であること、`Details` の中身、Leave の有無、座席が active のままかを確かめる。
- evidence の Run 4 が、id の振り直しと get の応答を実機で確かめ、driver のテストの定数がその応答と同じ形になっている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (この diff は台帳に行を足していない。F-3〜F-10 は `/sync-docs` が足す計画。F-3、F-4、F-6、F-7、F-9 は 1 行ずつ、F-5 と F-8 と F-10 は 1 行にまとめてよい。あわせて、`spawn.go:1845` の `compensateStale` が台帳の pane id に確認なしで C-c を送る既存の経路を 1 行) | | | | |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。前回の MEDIUM(F-1)と F-2 は解消した。
- Follow-ups: F-1 の残余(b)(`herdr agent start` のあとも tab の label が保たれるか)は、確かめれば閉じる。隔離したサーバーで 1 回確かめる手順で、結果を evidence に足せば AC14 の根拠にもなる。F-3〜F-10 は blocking ではない。直す場合は `internal/` の変更になり、`/self-review` から回し直しになる(パイプラインの上限は 2 回で、今回が 2 回目)ので、直さずに台帳へ送るのが妥当と考える。
- `/sync-docs` への引き継ぎ(私は文書のずれを判定しない): `internal/cli/org.go:724-734`(`stop` の Long)と `:994-1004`(`disband` の Long)は、「C-c を送り、pane を閉じる」とだけ書き、確認の失敗と `--force` が閉じないことに触れていない。skill の表は更新済み。
