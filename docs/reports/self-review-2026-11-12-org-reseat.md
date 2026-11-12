# Self-review report: org-reseat

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(issue #312 の受け入れ条件 2、`ralph org reseat`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目の中の 2 回目の実行(`cycle-count.json` は無く、`--cycle auto` は 1)。cross-review の前
- この report は、6aa5188f の 1 回目の report(M1 と L1 から L6)を置き換える。理由: 1 回目の指摘のうち M1・L1・L3(a) を 93395521 が直した(plan の P9、P10、AC-20、AC-21)ので、直しが穴を残していないかを見て、残りの LOW を判定し直した結果を 1 つの report にまとめるため。1 回目の本文は `git show 6aa5188f:docs/reports/self-review-2026-11-12-org-reseat.md` で読める
- Scope: `git diff 0936aaea...HEAD`(HEAD `f0c53cde`、22 ファイル)。今回は 93395521 のコード全部(`internal/cli/org_reseat.go`、`internal/org/reseat.go`)と、そこから呼ぶ `Stop`・`noteDetails`・`Spawn` の prompt の扱い・`autonomousScopeGateErr`・`RenderRolePrompt`・`FeatureRepoRoot`・`notIgnoredSignals`・`signalCauseContext`・`reraiseSignal` を読んだ。テストは P9・P10 の部分(`reseat_test.go` の 318 から 513 行、`org_reseat_test.go` の 455 から 580 行)だけ。skill の写し 8 本、evidence、spec と plan の本文(P9・P10・AC の行を除く)は diff 品質の対象外として読んでいない。テスト、静的解析、仕様適合、文書の drift は見ていない。リポジトリのコードは実行していない。scratchpad に Go の小さな probe を 2 本置いて動かした(下の M2)
- ID の付け方: 前回の `M1`、`L1` から `L6` は、直ったものを「前回の指摘の判定」に残し、開いたままの `L2`・`L3`・`L4`・`L5`・`L6` は番号を変えずに続ける。今回の新しい指摘は `M2`、`L7` から `L9`

## 前回の指摘の判定

| ID | 判定 | 根拠 |
| --- | --- | --- |
| M1 | 直った。ただし直しの 1 層目に穴があり、M2 に書く | 1 層目は CLI の signal の受け止め(`org_reseat.go` の `orgReseatHoldSignals`)、2 層目は `--seat` なしの対象に足した `cutOff`(`reseat.go` の `reseatLifeSeats`、`reseatTargets`)。2 層目の確認は下の「cut-off の検出」 |
| L1 | 直った(止めの前に断る) | `reseatCustomRoleErr`(`reseat.go:626`)が `RenderRolePrompt(role, RolePromptVars{})` の `ok` を、Spawn が読むのと同じ判定として使う。`PlanReseat` の中で、対象が決まったあと、何かを組む前に呼ばれる(`reseat.go:356`)。残りは L9 と、下の「`--prompt` の残り」 |
| L3 | (a) は直った。(b) は残っている | (a) `orgReseatNoMainWorktreeErr` が reseat の文にした。`FeatureRepoRoot` の失敗は `statedir.go` の 2 つの分岐だけで、`orgMainWorktreeMissing` の文と合う。(b) `feature.go:413` の「the leader's task copies the [reviewer] values」が、leader を対象にしない reseat でも出る。`reseat.go:401` から 404 行は変わっていない |
| L2 | 残っている | `orgReseatHerdrServerProbe`(`org_reseat.go:306`)と `ensureOrgHerdrServer` の「`herdr status server --json` が無い」の warning は同じ文字列のまま(`org_reseat.go:314`、`org_herdr_server.go:134`) |
| L4 | 残っている | `reseatSeat`(`reseat.go:674`)と型 `ReseatSeat`(`reseat.go:76`) |
| L5 | 残っていて、少し悪くなった | `PlanReseat` は `reseat.go:328` から 472 行で 145 行(前回は 142 行)。`plan.Start[0]` が leader であることに頼る 2 か所(387 行、399 行)も同じ。93395521 は `cutOff` の seat も `plan.Start[0]` に入りうるようにしたが、`Active` の判定(387 行)があるので止めの側は崩れない |
| L6 | 残っている | `docs/plans/active/2026-11-12-org-reseat.md` を指す注釈は、コードに 3 か所(`reseat.go:22`、`seat.go:190`、`org_reseat.go:19`)とテストに 2 か所。93395521 の新しい注釈は plan の番号 `P9`・`P10`・`AC-21` を引くので、archive へ動いたあとは plan の slug で探すことになる |

## Evidence reviewed

lead に頼まれた 3 点と、残りの LOW を、コードを読んで追った。

### cut-off の検出(人や leader が意図して止めた座席を起こさないか)

`cutOff` が立つ条件は 3 つが全部そろうとき。(1) 座席の最新の state event が `stopped`(`stop_failed` は state event ではないので数えない。dry run の event は `ev.DryRun` で読み飛ばす)。(2) その `Details` の最後の語が `reseat=<id>`(`reseatNoteID`)。(3) 今の命に `id=<id>` の `reseated` が無い(`reseatedID`)。

- `reseat=<id>` を `Details` の最後に書くのは `noteDetails` だけで、呼ぶのは `Stop` と `Spawn` の `Note` 欄。`Note` を値で受ける CLI の flag は無い(`internal/cli` の非テストで `Note:` を設定するのは `org_followup.go:90` の別の型だけ)。人の `ralph org stop` や leader の stop の `Details` は `leave=ok`・`leave=failed: ...`・`leave=skipped: ...`・`model_observed=...` のどれかで終わる。`leave=failed: <エラー文>` の最後の語が `reseat=` で始まる偽造は、エラー文に依存するので現実の入力では起きない
- 人が reseat の前に止めた座席は active でも `cutOff` でもないので、`--seat` なしの対象にならない(D2 のまま)
- cut-off した reseat が止めた座席を、人が止め直すと、新しい `stopped` が `Note` なしで書かれ、`cutOff` が消える。`Stop` は座席の今の状態を見ずに書く(`verbs.go:1330-1336` の `seatFromEvents` は存在だけを見る)ので、pane が閉じていても `pane=already closed` で成功する。この道はテストに無い。`TestPlanReseat_FinishesAReseatThatWasCutOff` の「a finished reseat's seat stopped later by hand」は、終わった reseat の座席を人が止める場合で、cut-off した座席を止め直す場合ではない。doc comment(「a later stop by hand, which carries no note」)はこの 2 つ目を指して読める。コードを読む限り成り立つが、証拠はテストにない(テストは `/test` の担当なので指摘にはしない)
- 人が止めた座席を、あとから別の reseat が止めることは無い。reseat が止めるのは active な座席だけ(`reseat.go:383`)
- reseat id の照合は、`map[string]bool` の完全一致。id は `YYYYMMDDThhmmss-xxxx` の 20 文字(`newReseatID`)で固定幅なので、ある id が別の id の接頭辞になることは無い。`reseatNoteID` は最後の語、`reseatedID` は最初の語しか読まない。`TestReseatNoteAndReseatedID` が「語が別の位置にあると数えない」を固定している
- `reseated` の書き込みに失敗した reseat(`finish` の `appendEvent`)は、止めた座席が `reseated` なしで残るが、始めた座席は active で `cutOff` にならない。止めて始められなかった座席だけが `cutOff` になり、次の reseat が拾う。これは意図どおり
- 終わった reseat が途中で止まった座席(`stopped`、`not_started`)は、`reseated` に id があるので `cutOff` にならない。AC-6 の「`--seat` を付けた打ち方を出す」の道のまま
- 起動の途中の失敗は `spawn_failed`・`rejected` が最新になるので、`cutOff` から外れる。そのとき reseat は `reseated` を書き、`Retry` が `--seat` を出す。起動の途中で kill されて `spawn_started` で止まった座席は active として扱われ、次の reseat が止めてから起動する

結論: 人や leader が意図して止めた座席が、cut-off の検出で起きることは無い。残る穴は L8(cut-off した reseat が `--seat` つきだった場合の、打ち直しの言い方)。

### `reseat=<id>` の偽造と接頭辞

上の通り。偽造は、`Note` を設定できる呼び手(`Reseat` だけ)以外では起きない。接頭辞の一致も起きない。

### signal の受け止めと終了コード・出力

`orgReseatHoldSignals`(`org_reseat.go:254`)を、sibling の `signalCauseContext`・`reraiseSignal`(`org_wait_director.go`)と並べて読んだ。

- 無視されて始まった signal は受けない(`notIgnoredSignals`)。全部無視なら `signal.Notify` を呼ばずに `call()` を走らせる。sibling と同じ作りで合っている
- 最初の 1 つで 1 行を stderr に出し、あとの signal は buffer 1 の channel に入れるか捨てる。`signal.Stop` は `call()` が返って、goroutine が `<-noted` で終わってから呼ぶ
- `err` は `runOrgReseat` の外側の変数に入り(`org_reseat.go:207-210`)、dry run は `orgReseatThroughSignals` を通らない。`TestOrgReseat_ARealReseatRunsThroughTheSignalHold` がこの 2 点を固定している
- 終了コードは `printOrgReseatResult` の返す値のまま。signal を受けても、全部立てば 0。この扱いは L7
- stderr の note の書き込みが、pipe の読み手が居ないと process を殺す。これが M2

### M2 の probe

`orgReseatHoldSignals` と同じ形(SIGINT を `signal.Notify` で受け、goroutine が最初の 1 つで stderr に 1 行書き、本体は 1.5 秒後に記録ファイルを書く)の Go のプログラムを scratchpad に置き、python から stderr を pipe にして、読み側を閉じてから SIGINT を送った。

| 条件 | 終了 | 記録ファイル |
| --- | --- | --- |
| stderr の pipe の読み手が居る | 0(note は読み手に届いた) | 書かれた |
| 読み手を閉じてある | SIGPIPE(`-13`) | 書かれない |
| 読み手を閉じてあり、`signal.Notify` に SIGPIPE も加えた | 0 | 書かれた |

(1 つ目と 2 つ目の条件は 2 回ずつ、3 つ目は 1 回動かした。ビルドしたての binary は最初の 1 回の起動が遅く、SIGINT が `Notify` より先に着いて `-2` になったので、その回は数えていない。実際の `ralph` の binary では動かしていない。)

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M2 | MEDIUM | Exception handling | signal を受けたときの stderr の note が、pipe の読み手が居ないと process を SIGPIPE で殺す。Go の runtime は、SIGPIPE を `Notify` していない process が fd 1 か 2 への書き込みで broken pipe に当たると、その process を SIGPIPE で終える。`orgReseatHoldSignals` の goroutine は最初の signal で stderr に 1 行書くので、`ralph org reseat ... 2>&1 \| tee log` で Ctrl-C を押す(tee が同じ前景の process group で先に死ぬ)、director の Bash tool のように出力を pipe で受ける呼び手が中断して pipe を閉じる、のどちらでも、記録(`reseated`)を書く前に reseat が終わりうる。M1 が閉じようとした穴が、この条件で開く。止めた座席は `cutOff` で次の reseat が拾うので、被害は「打ち直すと終わる」ところで止まる。stdout の最後の print も同じ道だが、そこは記録のあとで台帳は無事。help の「A real reseat does not end at SIGINT, SIGTERM, or SIGHUP」(`org_reseat.go:71-72`)と、テストの名前「KeepsTheProcess」は、この条件で成り立たない。テストの stderr は `regateCLIStderr`(buffer)で、閉じた pipe の場合が無い | `internal/cli/org_reseat.go:254-280`(`orgReseatHoldSignals`、goroutine の `fmt.Fprintf(stderr, ...)`)、`org_reseat_test.go:509-553`。上の probe(M2 の probe)。probe は同じ形のプログラムで、実際の `ralph` の binary ではない | `signal.Notify` の対象に `syscall.SIGPIPE` を加え(受けた値は捨てる)、書き込みを EPIPE で返す形にする。probe では、これで SIGINT のあとも記録まで進んだ。note を捨てる案でもよい。どちらでも、閉じた pipe に stderr をつないだ child で note の経路を通すテストを足す |
| L2 | LOW | Maintainability | 前回のまま。dry run の herdr probe が gate の前半を写している | `internal/cli/org_reseat.go:306-322`、`org_herdr_server.go:127-143` | 「status を読んで running かを返す」関数を 1 つ出す |
| L3 | LOW | Error text | (b) だけ残っている。leader を対象にしない reseat(implementer だけ、など)で `[reviewer]` の値が不正だと、「the leader's task copies the [reviewer] values」と出る。検査自体は予約の比較に leader の起動の検査が要るので必要で、文だけが reseat に合わない | `internal/org/feature.go:413`、`reseat.go:401-404` | `refuse` の文に「leader は起こさないが、予約の比較に leader の起動の検査を通す」の 1 句を足す |
| L4 | LOW | Naming | 前回のまま。`reseatSeat`(関数)と `ReseatSeat`(型)が頭文字の大小だけで違う | `reseat.go:76`、`reseat.go:674` | 関数を `newReseatSeat` にする |
| L5 | LOW | Readability | 前回のまま。`PlanReseat` が 145 行で、`plan.Start`・`stops`・`sim` を同時に持ち、`plan.Start[0]` が leader であることに 2 か所が頼る | `reseat.go:328-472`、387 行、399 行 | `leaderIdx` を返す形にする。leader 向けの組み立てと非 leader の `Spawn` の組み立てを別の関数にする |
| L6 | LOW | Maintainability | 前回のまま。注釈が archive へ動く plan を指す。93395521 が足した注釈も plan の番号(`P9`、`P10`、`AC-21`)を引く | `reseat.go:22`、`seat.go:190`、`org_reseat.go:19` と、テスト 2 か所 | plan の slug だけにするか、動かない `docs/specs/2026-11-12-org-reseat.md` を指す |
| L7 | LOW | Exception handling | signal を受けて最後まで走った reseat は、signal で終わらず、`Reseat` の結果の終了コード(全部立てば 0)で終わる。sibling の `ralph org wait --director` は、終わりを記録したあとで同じ signal を上げ直し(`orgDirectorWaitReraise`)、呼び手に「signal で終わった」と見せる。reseat は Ctrl-C のあとも 0 で終わるので、`ralph org reseat ...; ralph org send ...` のような shell の列は、Ctrl-C のあとも次へ進む。直前に stderr の note があるので人は気づくが、script は気づかない | `org_reseat.go:254-280`(上げ直しが無い)、`org_wait_director.go:82`、`:98-199`(`reraiseSignal`) | 最初に受けた signal を覚えておき、出力のあとで `reraiseSignal` で上げ直す。しないなら、help に「signal を受けても終了コードは reseat の結果のまま」と書く |
| L8 | LOW | Maintainability | help が、cut-off した reseat は「`--seat` なしで打ち直すと終わる」と言う。cut-off した reseat が `--seat implementer` のようにつきの reseat(leader は動き続けるという AC-3 の形)だった場合、`--seat` なしの打ち直しは、cut-off した座席を足したうえで、動いている leader を含む active な座席を全部止めて立て直す。この場合の打ち直しは同じ `--seat` を付けた reseat。dry run の `start:` 行も、active でない座席が cut-off した reseat の座席だからだとは言わない(`stop:` 行が無い、で読み取るしかない)。全部の座席を対象にした reseat が leader の起動のあとで切れた場合は、今動いている leader も止めて立て直す(1 回起こし直した直後なので失うものは少ない) | `org_reseat.go:34-35`(help)、`reseat.go:539-551`(`reseatTargets`)、`reseat.go:382-389`(`stops`)、`org_reseat.go:336-349`(`printOrgReseatDryRun`) | help を「同じコマンドをもう一度打つ」に変える(`--seat` つきならそれが cut-off の座席に当たる)。dry run の `start:` 行に、cut-off した reseat の id を 1 語足す |
| L9 | LOW | Error text | 独自の役割の座席を断る文が出す手での立て直しの `spawn` は、reseat 自身が渡す `--scope`・`--driver`・`--model` を持たない。既定の `[org.permissions].default = "autonomous"` では、`--scope` も `--reserve` も無い `spawn` は `autonomousScopeGateErr` に断られる。`--driver` と `--model` は、独自の役割には `[org.role_models]` の entry が無いので、台帳に記録された値(テストでは haiku)ではなく built-in の既定に落ちる。実行して確かめてはいない(コードを読んだ)。断りの文は `<its prompt>` の置き換えも要求するので、人が 1 度は書き直す文ではある | `reseat.go:626-660`(`reseatCustomRoleErr`、`spawn := ...`)、`spawn.go:1805`(`autonomousScopeGateErr`)、`reseat.go:438-441`(reseat が渡す `Scope`、`Driver`、`Model`) | 文に、記録された `--driver`・`--model` と、reseat が使う `--scope` の値(または `--scope <the org's scope>`)を足す |

### `--prompt` の残り(指摘にしない)

known role(template がある役割)の座席に `--prompt` が付いている場合、Spawn は template のあとにそれを足す(`spawn.go:1097-1107`)。台帳が `--prompt` を持たないので、reseat した座席はその足した分を失う。`reseatCustomRoleErr` は template が無い役割だけを断り、この場合を断らない。spec(143 行)が `--prompt` を読み戻せないものと認めており、出荷されている role prompt(`internal/org/prompts/*.md`)に `--prompt` を使う箇所が無く、失うのは template に足された分だけで TASK と同じく leader が送り直せる。そのため指摘にしない。`reseatCustomRoleErr` の doc comment の「so reseat would start it with no instructions」は template が無い場合だけの話で、合っている。

## Positive notes

- `cutOff` の 3 条件が、書く側(`Stop` の `Note`)と読む側(`reseatNoteID`、`reseatedID`)で同じ形に決まっている。id の照合は完全一致で、接頭辞や偽造の道が無い。
- cut-off した座席は active でないので、`Reseat` が止めない(`stops` は `Active` だけ)。起動の順、`Retry`、`Decision` の組み立ては、cut-off した座席と leader が混ざっても前の形のまま通る(`finish` の leader の扱い、`reseatedDetails`)。
- `reseatCustomRoleErr` が「template があるか」を、Spawn が読む `RenderRolePrompt` の `ok` で決める。二重の判定を持たない。
- `orgReseatHoldSignals` が、無視されて始まった signal を受けない(`notIgnoredSignals`)理由を、sibling と同じ言葉で書いている。signal を受けない分岐(`len(sigs) == 0`)で `signal.Notify` が全 signal を中継する罠を避けている。
- `orgReseatNoMainWorktreeErr` の doc comment が、`FeatureRepoRoot` の失敗が 2 つの分岐だけであることを前提として書いている。分岐が増えたときに文が合わなくなるが、`org_cleanup.go`・`org_merge.go` と同じ作りで一貫している。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| M2: signal の note が閉じた pipe で process を殺す | pipe の読み手が先に死ぬ実行(`2>&1 \| tee`、agent の Bash tool)で、受け止めた signal のあとも途中で終わる。被害は `cutOff` で拾えるところまで | 直す案は 1 行(`Notify` に SIGPIPE を足す)で、deferral の理由は今のところ無い | 直さずに持ち越すなら、実機の director から reseat を打って中断した最初の報告 | 本 report の M2 |
| L7: signal を受けたあとも終了コードが 0 | script の列が Ctrl-C のあとも進む | 上げ直しを足す小さな変更 | reseat を script から呼ぶ運用が出たとき | 本 report の L7 |
| L8: cut-off した `--seat` つきの reseat の打ち直しの言い方 | `--seat` なしで打ち直すと、動いている leader も立て直す | help と dry run の文言の変更 | cut-off の実例の報告 | 本 report の L8 |
| L9: 独自の役割の座席の手での立て直しの文が、そのままでは断られる | 文を書き直す手間 | 文言の変更 | 独自の役割の座席を feature org で使う報告 | 本 report の L9、`docs/specs/2026-11-12-org-reseat.md` の 143 行 |
| L2・L3・L4・L5・L6 | 前回の report の通り | 前回の通り | 前回の通り | 1 回目の report(`git show 6aa5188f:docs/reports/self-review-2026-11-12-org-reseat.md`) |

_(上の表は `docs/tech-debt/README.md` へまだ足していない。直すか持ち越すかは orchestrator が決める。持ち越す指摘だけを足す。いまの `docs/tech-debt/README.md` に reseat の行は無い。パイプラインの上限(`max_pipeline_cycles` 2)には届いていないので、直す余地がある。)_

## Recommendation

- Merge: yes。CRITICAL と HIGH は無い。MEDIUM 1 件(M2)は、1 行の直しと閉じた pipe のテストで済むので、直してから先へ進むことを勧める
- 前回の M1 は、signal の受け止め(M2 の穴つき)と cut-off の検出の 2 層で直っていて、人や leader が意図して止めた座席を起こす道は無い。L1 は止めの前に断る形で直っている
- Follow-ups: M2 の直し。L7 から L9 と、残りの LOW(L2 から L6)は、同じコミットで直せる。直したら、パイプラインは `/self-review` からやり直す(`post-implementation-pipeline.md`)
