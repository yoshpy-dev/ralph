# Self-review report: org-stop-all

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Branch: feat/org-stop-all(HEAD 672172be)
- Reviewer: reviewer subagent (Claude)、パイプライン 2 回目(cycle 2、既定の上限の最後の 1 回)。1 回目は a103a05f と eb7bf5e2(HEAD 05977322)
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、typo、null 安全、デバッグ用コード、秘密情報、例外処理、安全性、保守性)。対象は cycle 1 のレビュー以降の差分 `git diff eb7bf5e2..HEAD -- ':!docs'`(10 ファイル、+981/-63)で、f63ae025(`/test` が足したテスト)、223258eb(`/sync-docs` が書いた stop / disband の Long と skill の disband 行)、53807a82(S8)を全行読んだ。重点は S8 の補償が誤った記録を書きうるか、再試行が止まらなくなるか、`--force` の扱い、エラー文の案内、doc comment の食い違い、ヘルプの正確さ。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査は実行していない。実機の herdr も使っていないので、挙動の判断はコードと fake のテストの読みによる
- この報告書の ID: 今回の指摘は `C2-1`〜`C2-6`。cycle 1 の `F-1`〜`F-10` は、`docs/tech-debt/README.md` の 163〜168 行が `(F-3, F-4)` のように指しているため、末尾の「付録」に原文のまま残した。この報告書は cycle ごとに上書きされるので、登録簿から指すときは ID の前に cycle を付ける

## 前回(cycle 1)からの変化

| 前回の指摘 | 今回 | 根拠 |
| --- | --- | --- |
| F-1(持ち主の確認) | 解消のまま。残余 (b) も解消 | `docs/evidence/herdr-pane-close-2026-10-07.md` の Run 5(801ca77f)が、`herdr agent start` のあとも tab と workspace の label が保たれることを隔離サーバーで確かめている。(a)(c) は受け入れ済みのまま。S8 のエラー文も `--force` の退路には触れない(手で閉じる案内は `--force` のときと、記録を戻せないときだけに付く。C2-1 の下の表) |
| F-2(`ErrWaitDelay`) | 解消のまま | S8 は `driver.go` を触っていない |
| F-3(`--all` に全体の上限がない) | 継続 | S8 の補償は台帳への追記だけで、driver 呼び出しを増やしていない |
| F-4(`Disband` の読み取りから `disbanded` までの窓) | 継続 | 補償にも同じ形の窓がある(C2-1) |
| F-5(`withPrefixOnce`) | 継続 | 変化なし(S8 で `org.go:802-806` に移った) |
| F-6(「自分の」の判定が複数箇所) | 継続、悪化 | S8 が 6 か所目を足した(C2-2) |
| F-7(`NewHerdrError` の export) | 継続 | 変化なし |
| F-8(skill の表のセルが長い) | 継続、わずかに悪化 | S8 が disband の行に 1 文(約 70 字)足した |
| F-9(`%v` で連鎖を切る) | 継続 | S8 の `closeSelfPane` は `confirmSeatPane` の error を `%w` で包むが、その先で `isNotFound` を掛けないので、F-9 の守りには触れない(`verbs.go:1039-1043`) |
| F-10(コメントの折り返し) | 継続 | S8 が同じ種類の行を 1 つ足した(C2-6) |

CRITICAL、HIGH、MEDIUM はない。新しい指摘は LOW 6 件。

## S8 についての確認

### 1. 補償が誤った記録を書きうるか

- 座席の特定(pane 側): `lastSeatOnPane(rr.Events, paneID)`(`verbs.go:1019`)は、`Stop` が直前に書いた `stopped` を引くので、呼んだ本人の座席になる。`reactivateSeat` は、その座席の最後の状態が `stopped` で pane_id が同じときだけ書く(`:1177`)。
- 座席の特定(workspace 側): pane id の一致だけで決める。持ち主の確認を通らない(C2-2)。
- 別の org の workspace を開き直すか: 開き直さない。`reopenWorkspace` は、確認に使ったのと同じ `last.OrgID` の記録を書く(`:1092`、`:1102`)。確認が label の不一致で落ちたときも、開き直すのは disband の前に台帳が持っていた状態で、`resolveWorkspace`(`spawn.go:1705`)が label を見ずに再利用するのは S8 より前からの挙動。S8 が増やした経路ではない。
- 閉じられていたのに補償する場合: close が時間切れになっても、herdr のサーバー側では閉じていることがある。プロセスは pane と一緒に終わるので補償は走らないか、書いた直後に終わる。台帳に active が残っても、次の `stop` は `pane_not_found` を閉じ済みと読んで `stopped` を書き直す(`verbs.go:677`、workspace は `:723`)。自己修復する。実機では試していない。
- 古い値のコピー: Role、Driver、Model、Worktree、PaneID、AgmsgTeam は、Roster の最後の状態の `stopped` から写す(`:1181-1183`)。HerdrAgentName は `lastHerdrAgentName`(`:1226`)が、その座席の最新の名前つきの記録から引く。名前を持つ記録は `spawned` だけ(`spawn.go:939`)なので、元の `spawned` か、前回の補償が写した `spawned` で、どちらも同じ名前になる。Details は写さない(元の `spawned` の `permission_mode=` や `scope=` はその記録にだけ残る)。Details を読むのは `report.go:150` の permission 列だけで、`stopped` の座席もすでに `-` と出るので、悪くなる点はない。写しの古さは C2-1。
- コードを読んだかぎりでは、`codexSpawnCorrelation`(`verbs.go:1307-1313`)は `spawn_started` を見て `spawned` を見ないので、補償の `spawned` が codex の観測の突き合わせをずらすことはない。`leaderActivityEventCount`(`watch.go:827`)は `spawned` を leader の活動に数えるが、`stopped` も数えているので、見える差は出ない。

### 2. 再試行が止まらなくなるか、補償が 2 回入るか

- `CloseDeferredSelfPane` と `CloseDeferredSelfWorkspace` の呼び出し元は `internal/cli/org.go:895`、`:898` の 1 か所で、1 回のコマンドで 1 回しか呼ばれない。内部に再試行のループはない。
- 補償は、台帳が実際に食い違うときだけ書く。座席がまだ active のまま、workspace がまだ open のままの表(`TestOrgCloseDeferredSelfPane_Outcomes`、`TestOrgCloseDeferredSelfWorkspace_Outcomes` の追加の確認)が、追記 0 件を見ている。同じ補償が 1 回の呼び出しで 2 回入る経路は見つからなかった。
- 打ち直しのたびに台帳は増える。pane 側は 1 回につき 2 件(`stopped`、`spawned`)、workspace 側は最大 5 件(`stopped`、`org_workspace_closed`、`disbanded`、`org_workspace_created`、`spawned`)。操作する人が打った回数に比例し、自動では増えない。

### 3. `--force`

- 補償は 1 件も書かない(`verbs.go:1028-1030`、`:1097-1099` で返る)。CLI は警告を出し(`org.go:903-905`)、`runErr` をそのまま返すので、ほかに失敗がなければ終了コード 0、あれば 1。
- `TestOrgCloseDeferredSelf_Force_NoCompensation` が追記 0 件と、座席が stopped のまま、org が disbanded のままを見る。CLI の `TestOrgStopDisband_Force_OwnCloseFails_WarnsExitsZero` が終了コード 0、stderr の警告、台帳に `reactivated:` と `reopened:` がないことを見る。

### 4. 手で閉じる案内が付く場所

| 場合 | 案内 | 場所 |
| --- | --- | --- |
| 台帳が読めない | あり | `verbs.go:1017`、`:1086` |
| その pane、workspace を記録した座席、org がない | あり | `:1021`、`:1090` |
| close が失敗、`--force` | あり | `:1029`、`:1098` |
| close が失敗、補償の追記が失敗 | あり(`errorFor` が失敗を並べたあとに付ける) | `:1151` 以下 |
| close が失敗、補償を書けた | なし(「打ち直すと閉じ直す」だけ) | `errorFor` |
| close が失敗、台帳がすでに食い違っていない | なし(元のエラーのまま) | `errorFor` |

「台帳を直せない場合」と案内の付く場所は一致している。それぞれ `TestOrgCloseDeferredSelf_NothingRestored_NamesTheManualClose`、`TestOrgCloseDeferredSelf_Force_NoCompensation`、`_Outcomes` の表が文言まで見ている。残る 2 行のうち、最後の「すでに食い違っていない」は、実際には `reactivateSeat` が "" を返す 3 条件(座席がない、最後の状態が `stopped` でない、pane が違う)を 1 つにまとめている(C2-1)。

### 5. 古い挙動を書いた doc comment

`Stop`(`verbs.go:819`)、`CloseDeferredSelfPane`(`:1000-1009`)、`CloseDeferredSelfWorkspace`(`:1069-1078`)、`Disband`(`:1678-1687`)、`closeDeferredSelf`(`org.go:876-889`)は、補償の追加に合わせて書き直されていて、コードと食い違わない。`StopResult.DeferredSelfPaneID`(`:766-772`)、`StopAllResult.DeferredSelfPaneID`(`verbs_all.go:57-60`)、`DisbandResult` の同名の欄(`:1581-1591`)は、「最後の操作として閉じる」としか言っていないので、そのまま正しい。`lastOrgOfWorkspace` は `lastWorkspaceEvent` に変わり、Go のコードと doc comment に旧名は残っていない(旧名は計画書の 155 行と `/test` の報告書の 98 行に、経緯として残る)。

### 6. ヘルプの正確さ

- `stop` の Long(`org.go:724-743`)は、コードと食い違わない。「`stop --all` を別の pane から」は、自分の pane を閉じる経路を通らずに、通常の C-c と close で閉じ直せるという意味で正しい。
- `disband` の Long(`org.go:1011-1034`)の「`--force` でも、label で org のものと確かめた workspace は閉じ、tab の確認に落ちた座席の pane も一緒に終わる」は、`closeOrgWorkspace`(`verbs.go:1780-1823`)が `Force` で確認を飛ばさないことと合う。ただし最後の文は、pane だけを後回しにした場合を言い落としている(C2-4)。

## Evidence reviewed

- `git diff 223258eb..53807a82` の `internal/org/verbs.go` と `internal/cli/org.go` の全行、`git diff eb7bf5e2..HEAD` の `internal/org/verbs_test.go`(全部)、`internal/cli/org_stop_all_test.go`、`internal/cli/org_legacy_ledger_test.go`、`internal/org/prompts_test.go`。
- `Roster`(`manifest.go:171-237`)で、`disbanded` が seat の `Event` を変えず `Active` だけを落とすこと(補償の `spawned` が `disbanded` のあとに入ると active に戻る)、`orgsToDisband`(`verbs_all.go:307-331`)が補償のあとの org を拾うこと、`openOrgWorkspaces`(`spawn.go:1729`)が補償の `org_workspace_created` で workspace を開いたと数えることを読んだ。
- `lastSeatOnPane`(`verbs.go:1213`)は `spawn_failed` など状態でない記録も引くが、`reactivateSeat` が Roster の状態で絞り直す。
- `Disband`、`disbandOwnLast`、`closeOrgWorkspace`、`StopAll`、`DisbandAll`、`splitOwnOrgs` の自分の pane、workspace の扱いを、補償が前提にする「呼んだ本人の座席」の取り方と突き合わせた。
- `.claude/skills/org/SKILL.md` の 4 面の写し: 223258eb と 53807a82 のどちらも、追加・削除の行の md5 が 4 面で一致する。`cmp` で root と `templates/base/` の 2 組が同一。
- 追加行にデバッグ用コード、`TODO`、固定パス、秘密情報はない。
- `docs/tech-debt/README.md` の 163〜168 行が引く `verbs.go`、`org.go` の行番号を、HEAD と 223258eb(S8 の前)の両方で `sed -n` で引き直した(C2-3)。
- `docs/insights/events/2026-10-07-org-stop-all.jsonl` は self_review 2 件、verify、test、sync_docs、cross_review が各 1 件。今回で self_review の cycle 2 を足す。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| C2-1 LOW | exception-handling | 補償は、driver を呼ぶ前に読んだ台帳の写しで判断する。補償が走るのは herdr が応答しなかったときで、その場合は読んでから 10 秒(最初の get が応答しない場合)から最大 40 秒(get 3 回と close 1 回、各 10 秒)たっている。この間に同じ座席が別の pane で spawn し直されると、`reactivateSeat` は写しの `stopped` を見て古い pane の `spawned` を足し、roster の pane が古い方に戻る。逆に写しが古くて条件に合わないと、`reactivateSeat` は座席なし、最後の状態が `stopped` でない、pane が違うの 3 つをすべて "" で返し、`errorFor` はそれを「すでに active」と読んで元のエラーのまま返す(手で閉じる案内も付かない)。`errorFor` の doc comment(「すでに active で workspace が open」)は 3 条件のうち 1 つしか言っていない。起きるには、補償の窓の中で同じ座席 id の `spawn` か、`rejected` / `spawn_failed` の記録が入る必要があり、狭い | `internal/org/verbs.go:1015`(読み取り)、`:1023`(最大 4 回の呼び出し)、`:1032`(写しで `reactivateSeat`)、`:1084`、`:1092`、`:1102-1105`、`:1177`、`:1199`、`:1146-1150`(`errorFor` の doc) | 補償の直前に台帳を読み直し、その写しで `reactivateSeat` と `reopenWorkspace` を呼ぶ(窓は数ミリ秒になる)。直さないなら `reactivateSeat` の doc comment に「呼び出しの前に読んだ写しで判断する」と書き、`errorFor` の doc を 3 条件に合わせる |
| C2-2 LOW | security | `CloseDeferredSelfWorkspace` が補償で戻す自分の座席を、`HERDR_PANE_ID` と一致する最新の記録(`lastSeatOnPane`)と、その座席が同じ org の `stopped` であることだけで選ぶ。持ち主の確認(tab の label)を通らない。この PR の前提は「台帳の pane id は別のものを指しうる」(F-1)で、evidence の Run 4 は、herdr の保存ファイルがないと id が振り直されることを確かめている。そのあとで、同じ org に古い `stopped` の座席と、同じ pane id で動く leader が並び、leader の自分の workspace の close が失敗すると、古い座席が active に戻る。disband を打ち直すと、その座席は自分の pane の座席として `Stop` に回り、tab の label が座席 id でないため `stop_failed` になり、`--force` が要る。pane 側(`:1019`)は `Stop` が直前に書いた `stopped` を引くので、この問題はない。F-6(b) の「自分の pane か」を決める箇所に、`o.getenv(herdrPaneIDEnv)` を読み直す 6 か所目も足している。実機では確かめていない | `internal/org/verbs.go:1103-1106`。比較: `Disband` の `ownSeats`(`:1712`)は id の一致で順番を決めるだけで、持ち主の判断は `Stop` の `confirmSeatPane`(`:954`)が下す | 補償の前に `confirmSeatPane(orgID, seatID, ownPane)` を通し、通らなければ座席を戻さない。あるいは `Disband` が自分の座席 id を `DisbandResult` で返し、`CloseDeferredSelfWorkspace` に渡す(`getenv` の読み直しがなくなる)。直さないなら、台帳に 1 行 |
| C2-3 LOW | maintainability | S8 が `verbs.go` と `org.go` の行をずらし、登録簿の 163〜168 行が引く行番号の多くが別の場所に当たる。たとえば F-4 の行の `verbs.go:1541`(`Disband` の `Manifest.Read`)は HEAD では空行、`:1578`(`appendDisbanded` の呼び出し)は `DisbandResult` の欄、`:1444-1461`(`recordStop` / `failSeat` の写し)は `observeStopModelReceipt` の途中。写しは `:1601-1618` にある。`:1003`、`:1045`、`:1542`、`:1553`、`:1565`、`:1680`、`:1683`、`:1428`、`:1504`、`:1520` も同じ。`:831-939`、`:929`、`:930`、`:934`、`:949-981`、`:959` は 1 行ずれ、`org.go` の `:768`、`:799-803`、`:850` は 3 行ずれた。S8 の前(223258eb)では `sed -n 1541p` が `rr, err := o.Manifest.Read()` を返すので、行は S8 の前の番号で書かれている。もう 1 点、168 行目の (a)「台帳が読めない失敗経路(`:838`、`:1003`、`:1045`、`:1542`)と `org_workspace_created` の追記失敗に届くテストがない」のうち `:1003`、`:1045`(`CloseDeferredSelfPane`、`CloseDeferredSelfWorkspace` の読み取り)は、S8 の `TestOrgCloseDeferredSelf_NothingRestored_NamesTheManualClose` の「manifest unreadable」の 2 件が届くようになり、成り立たなくなった(HEAD では `:1015`、`:1084`)。`org_workspace_created` の追記失敗として行が挙げる `spawn.go:1716` は `resolveWorkspace` の追記で、S8 の補償の追記(`reopenWorkspace`)とは別なので、そちらは今も届いていない | `docs/tech-debt/README.md:163-168`。HEAD での行き先は上の本文。S8 の前は `git show 223258eb:internal/org/verbs.go \| sed -n 1541p` | `/sync-docs` で、行番号を関数名と動作(`Disband` の最初の `Manifest.Read`、`appendDisbanded` など)に置き換える。168 行目の (a) から `CloseDeferredSelfPane` と `CloseDeferredSelfWorkspace` の読み取り失敗(`:1003`、`:1045`)を外す(`Stop`、`Disband` の読み取りと、`stop_failed`、`disbanded`、`resolveWorkspace` の追記失敗は残る) |
| C2-4 LOW | readability | `disband` の Long の最後の段落は「失敗したら、座席の pane を active に、workspace を open に戻す」と書くが、後回しにしたのが pane だけのとき(自分の pane が org の workspace の外にある)は workspace は戻らない。`TestOrgCloseDeferredSelfPane_AfterDisband_ReactivatedPastDisbanded` がその経路で、戻るのは座席だけ。読み手は workspace も開き直されると受け取る | `internal/cli/org.go:1030-1033`。`verbs_test.go` の上記テスト | 「座席を active に戻し、workspace を後回しにしていたときはその workspace も open に戻す」のように、条件を 1 句足す。直すと `internal/` の変更になるので、直さないなら 1 行にまとめて台帳へ |
| C2-5 LOW | maintainability | `verbs.go` は main の 1050 行から cycle 1 で 1688 行、S8 で 1847 行になった(共通のレビュー基準の目安は 800 行)。S8 の自分の pane、workspace を最後に閉じる塊(`CloseDeferredSelfPane`、`CloseDeferredSelfWorkspace`、`closeSelfPane`、`closeSelfWorkspace`、`selfCompensation`、`reactivateSeat`、`reopenWorkspace`、`lastSeatOnPane`、`lastHerdrAgentName`、`lastWorkspaceEvent`)は 984〜1250 行目に連続していて、`Stop` / `Disband` の本体とは呼び出しの向きが一方向(それらが返した id を受けるだけ)なので、そのままファイルに切り出せる | `internal/org/verbs.go:984-1248` | 次に `verbs.go` を触るときに `verbs_selfclose.go`(仮)へ移す。この指摘だけのために再実行はしない |
| C2-6 LOW | readability | 小さい点が 4 つ。(a) `selfCompensation` のレシーバが混在する(`add` はポインタ、`errorFor` は値)。(b) `errorFor` は新しいエラーを作るのではなく元のエラーに文を足すので、名前が内容と合わない(`describe` や `annotate` の方が近い)。(c) `CloseDeferredSelfPane` の `err` が、最初は台帳の読み取りエラー、途中から包んだ close の失敗を指す(`:1015`、`:1027`)。`failure` との使い分けも読みにくい。(d) `Stop` の doc comment に括弧書きを挿したあと、折り返しが直っておらず、前後が 73〜77 字の中で 1 行だけ 61 字になっている(`:820`)。F-10 と同じ種類 | `internal/org/verbs.go:1137`、`:1151`、`:1015`、`:1027`、`:819-822` | 次に触るときに直す。この指摘だけのために再実行はしない |

## Positive notes

- 補償の書く条件が狭い。座席は最後の状態が `stopped` で pane が同じときだけ、workspace は最後が `org_workspace_closed` のときだけで、台帳がすでに合っているときは何も書かず、そのことを表のテストが見ている。
- close が時間切れでも実際には閉じていた場合は、補償が active を残しても、次の `stop` と `disband` が not-found を閉じ済みと読んで直る。誤った記録が自己修復する。
- 補償の失敗(追記できない、台帳が読めない)で、手で閉じる herdr のコマンドがエラー文に載る。`--force` のときも同じコマンドが警告に載る。
- 補償の `spawned` を、元の `spawned` と TS と Details 以外が一致するか、`assertReactivatedCopy` が欄ごとに見る。`HerdrAgentName` が空でないことも見ているので、`stopped` から写し漏らす退行は落ちる。
- `closeDeferredSelf` の `force` の分岐は、成功と同じ `runErr` を返すので、`--force` が終了コードの意味を変えない。
- `/test` が足したテストは、台帳に書き込めない場合(`readOnlyManifest`)を `t.Cleanup` で戻し、root での実行は skip する。`TestOrgDisband_WorkspaceGoneAtCheck_CountsAsClosed` は、実機の herdr の返し方(確認の get がすでに not-found)で、閉じる呼び出しが 0 回であることを見ている。
- skill の 4 面の写しは同一で、追加行に `internal/`、`cmd/`、`issue #`、`PR #` はない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C2-1、C2-2、C2-5、C2-6 を 1 行にまとめる。補償が driver 呼び出しの前の台帳の写しで判断する(`verbs.go:1015-1032`、`:1084-1105`)こと、workspace 側が自分の座席を pane id の一致だけで選ぶ(`:1103-1106`)こと、`verbs.go` の自分の pane / workspace を最後に閉じる塊が 1847 行のファイルに入っていること、`selfCompensation` の細かい点 | herdr が応答しないときに、同じ座席の別 pane への spawn し直しと重なると、古い pane の `spawned` が入る。herdr の id が振り直された org で、自分の workspace の close が失敗すると、古い `stopped` の座席が active に戻って disband に `--force` が要る。どちらも狭い条件 | cycle の上限(2 回)に達していて、直すと `internal/` の変更になり、`/self-review` から回し直しになる | `CloseDeferredSelfPane` / `CloseDeferredSelfWorkspace` / `reactivateSeat` / `reopenWorkspace` の次の変更。または、補償のあとの座席が disband で `stop_failed` になる報告 | `docs/reports/self-review-2026-10-07-org-stop-all.md`(C2-1、C2-2、C2-5、C2-6) |
| C2-4 を 1 行(直さない場合)。`disband` の Long が、pane だけを後回しにしたときは workspace が戻らないことを言い落とす | ヘルプの読み手が workspace も開き直されると受け取る | 同上 | `disband` の Long を次に直すとき | 同上(C2-4) |

C2-3 は台帳そのものの修正なので、行を足さず `/sync-docs` で既存の行を直す。F-3〜F-10 の行は `/sync-docs` がすでに足している(163〜168 行)。

## Recommendation

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。S8 の補償は、誤った座席や別の org の workspace を戻す経路が見つからず(C2-2 は pane id の再利用が前提の狭い条件)、`--force` では何も書かず、再試行は自動では回らない。
- Follow-ups: C2-3 は `/sync-docs` の直しで足りる(台帳の行番号を関数名に、168 行目の (a) から 2 か所を外す)。C2-1、C2-2、C2-5、C2-6、C2-4 は blocking ではなく、上の「Tech debt identified」の 2 行に送る。直すと `internal/` の変更になり、パイプラインの上限(2 回)を超えるので、この PR では直さないのが妥当と考える。
- `/sync-docs` への引き継ぎ(私は文書のずれを判定しない): (1) 台帳の 163〜168 行の行番号(C2-3)。(2) `.claude/skills/org/SKILL.md` の `stop` の行(167 行目)と `disband` の行(168 行目)は、「自分の pane、workspace を最後に閉じる close が失敗したら、座席を active、workspace を open に戻して終了コード 1、`--force` なら戻さず警告で終了コード 0」を書いていない(`--help` の Long にはある)。4 面の写しで同じ。`disband` の行の最後の「解散した org_id でまた `spawn` すると新しい workspace を作る」は、補償で開き直した workspace があるときは、その workspace を再利用する(`TestOrgCloseDeferredSelfWorkspace_CloseFails_ReopenedAndReactivated` の末尾)ので、例外が 1 つある。(3) 計画書の 155 行と `/test` の報告書の 98 行にある `lastOrgOfWorkspace` は、経緯として残すなら変更不要。

## 付録 A: S7 についての 3 つの確認(cycle 1 の原文)

登録簿(`docs/tech-debt/README.md` の 163〜168 行)は、この報告書の `F-3`、`F-1 residual (c)`、「`compensateStale` の記録(「1.」の下)」のように、cycle 1 の番号と節を指している。cycle 2 の上書きで指し先が消えないよう、cycle 1(HEAD 05977322)の節を原文のまま残す。行番号は 05977322 のもので、HEAD ではずれている(C2-3)。

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

## 付録 B: cycle 1 の Findings(F-1〜F-10 の原文)

F-1 の残余 (b) は、cycle 2 の時点で `docs/evidence/herdr-pane-close-2026-10-07.md` の Run 5 により解消している。F-3〜F-10 の登録簿の行は 163〜168 行。

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
