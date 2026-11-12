# Self-review report: org-reseat

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(issue #312 の受け入れ条件 2、`ralph org reseat`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目の中の 3 回目の実行(`cycle-count.json` は無く、`--cycle auto` は 1)。cross-review の前
- この report は、f597b3dd の 2 回目の report(M2 と L2 から L9)を置き換える。理由: 2 回目の指摘のうち M2・L7・L8・L9 を e5b2cb85 が直し(plan の P9・AC-20)、772da303 が test の helper の errcheck を直したので、直しが穴を残していないかを読み、残りの LOW を判定し直し、止めたまま立てない道をもう一度探した結果を 1 つの report にまとめるため。2 回目の本文は `git show f597b3dd:docs/reports/self-review-2026-11-12-org-reseat.md` で読める
- Scope: `git diff 0936aaea...HEAD`(HEAD `7ed1fd28`、22 ファイル)。今回は e5b2cb85 のコード(`internal/cli/org_reseat.go`、`internal/org/reseat.go`)と 2 つのテスト差分、772da303、7ed1fd28(plan と spec の差分。コードの主張と合うかだけ)を全部読み、`Reseat`・`stopAll`・`startAll`・`finish`・`takeReseatLock`・`reseatTargets`・`reseatCustomRoleErr`・`ResolveRoleModel`・`autonomousScopeGateErr`・`cmd/ralph/main.go` の error の出力を読んだ。skill の写し 8 本、evidence、spec と plan の本文(差分の外)は diff 品質の対象外として読んでいない。テスト、静的解析、仕様適合、文書の drift は見ていない。リポジトリのコードは実行していない。scratchpad に Go の小さな probe を 1 本置いて動かした(下の M2 の項)
- ID の付け方: 前回までの `M2`・`L2` から `L9` は、直ったものを「前回の指摘の判定」に残し、開いたままのものは番号を変えずに続ける。今回の新しい指摘は `L10`

## 前回の指摘の判定

| ID | 判定 | 根拠 |
| --- | --- | --- |
| M2 | 直った | `orgReseatHoldSignals`(`org_reseat.go:271-275`)が、SIGPIPE を専用の channel(`pipe`、読み手なし)で `signal.Notify` し、関数の終わりで `signal.Stop` する。下の「M2 の直しの読み」 |
| L7 | 直った(help に書く側の案) | help が「終了コードは reseat の結果のまま、signal のあとも」と言い(`org_reseat.go:77-79`)、help のテストが同じ文を固定する。上げ直しはしないままで、成功した reseat は signal のあとも 0 で終わる。sibling の `org wait --director` とは違うが、help が言っているので script の書き手は知れる |
| L8 | 直った | help(`org_reseat.go:33-37`)が「同じコマンドを同じ `--seat` で打ち直す」に変わり、`reseat.go` の 2 か所の doc comment(30-36 行、`Reseat` の 980-986 行)も合う。`TestPlanReseat_FinishesAReseatThatWasCutOff` が `--seat reviewer` の打ち直しで stop 0 件・start は reviewer だけを固定する。dry run の `start:` 行が cut-off の座席だと言わない点は残るが、help が言うので持ち越さない |
| L9 | 直った | 断りの文の `spawn` が `--driver`、`--model`(記録が `""` なら付けない)、`--scope` を持つ。`--scope` は `StringVar`(`org.go:982`)なので、`ShellWord` が引用した `a, b` 形の 1 語がそのまま 1 つの値になり、reseat 自身が渡す `Scope: scope` と同じ値。断りの `scope` は `PlanReseat` の先頭へ動かしただけで、`events` は同じ読み。`ResolveRoleModel` は `--driver` だけで `--model` 無しの spawn を、役割に `[org.role_models]` の entry があるときだけ通す(entry 無しの役割は両方の flag が要る)。test の `tally`(entry 無し、model `""`)は org 層の API でだけ作れる配置で、CLI で作れる「entry があり model が `""`」の配置は断りの文のとおり通る。持ち越さない |
| L2 | 残っている | `orgReseatHerdrServerProbe`(`org_reseat.go:326-342`)と `ensureOrgHerdrServer`(`org_herdr_server.go:127-143`)。warning の文字列も同じ(`org_reseat.go:334`、`org_herdr_server.go:134`) |
| L3 | (b) だけ残っている | `feature.go:413` の「the leader's task copies the [reviewer] values」が、leader を対象にしない reseat でも出る。`reseat.go:405` の `startFeatureInput` は変わっていない |
| L4 | 残っている | 関数 `reseatSeat`(`reseat.go:691`)と型 `ReseatSeat`(`reseat.go:77`) |
| L5 | 残っている | `PlanReseat` は `reseat.go:329` から 475 行で 147 行(前回 145 行)。`plan.Start[0]` が leader であることに頼る 2 か所(391 行、403 行)も同じ |
| L6 | 残っている | `docs/plans/active/2026-11-12-org-reseat.md` を指す注釈: `reseat.go:22`、`seat.go:190`、`org_reseat.go:19`、`reseat_test.go:19`、`org_reseat_test.go:26`。新しい注釈(`orgReseatSignals` の「plan P9」)も plan の番号を引く |

## Evidence reviewed

### M2 の直しの読み

- 受け止める期間: `orgReseatHoldSignals` に入ってから、note を書く goroutine が終わり `signal.Stop(ch)` を呼んだあと、関数が返るとき(`defer signal.Stop(pipe)`)まで。`len(sigs) == 0` の分岐(3 つとも無視されて始まった場合)でも `pipe` は受ける
- その間の stderr と stdout の書き込みは、note、2 つの gate の行(`org.go:545` 付近、`org_herdr_server.go:233`)、`printCodexModelMismatchWarning`、`printOrgReseatResult` の `Fprintf` のすべてが `_, _ =` で戻り値を捨てる。EPIPE が制御を変える経路は無い
- 受け止める前の書き込み(`printLegacyModelPoolNotice`、`printScaffoldVersionNotice`)は、何も止めていないうちに終わる。ここで落ちても失うものは無い
- 子の process(herdr、agmsg)への影響は無い。Go は継承した SIGPIPE の無視を残さず(`SIGHUP` と `SIGINT` だけ残す)自分の handler を入れ、exec で受け止めた signal は既定に戻るので、`Notify` の有無で子の扱いは変わらない。doc comment の「there is no ignored SIGPIPE to leave as it is」はこのとおり
- probe(`orgReseatHoldSignals` と同じ形: 専用の channel で SIGPIPE、SIGINT を受ける goroutine が stderr に 1 行、本体は 1.5 秒後に記録ファイルを書き、stdout と stderr に 1 行ずつ書く)。python から stdout と stderr を pipe にして、両方の読み側を閉じてから SIGINT を送った:

| 結果の種類 | 終了 | 記録ファイル |
| --- | --- | --- |
| 本体が正常に終わる | 0 | 書かれた |
| 本体のあと、`main.go` と同じく stderr に error を書いて `os.Exit(1)` | SIGPIPE(`-13`) | 書かれた |

  (それぞれ 3 回動かした。ビルドしたての binary の最初の 1 回は SIGINT が `Notify` より先に着いて `-2` になったので数えていない。実際の `ralph` の binary では動かしていない。)
- 下の行が残りの穴で、指摘にしない。reseat の結果が error のとき、`main.go` が hold のあとに `fmt.Fprintln(os.Stderr, err)` を書く。stderr の読み手が居ないと、`signal.Stop(pipe)` のあとなので process は SIGPIPE で終わる(exit 1 でなく、signal の終わり)。記録は書かれたあとで、どちらも 0 でなく、error の文はどのみち届かない。help の「終了コードは reseat の結果のまま」は成功のときを例に言っているので、help と食い違わない
- テスト(`org_reseat_test.go:606` 付近): 子の test binary の fd 2 を、読み側を閉じた pipe にする。子が自分に SIGINT を送り、note の書き込みと自分の書き込みの両方が EPIPE になり、hold が返り、`call` が決めた終了コード 3 で終わることを固定する。直しが無ければ子は SIGPIPE で終わるので、親の `ExitCode() != 3` で落ちる(走らせてはいない)。stdout が閉じた場合は、同じ channel が受けるが test は無い

### 止めたまま立てない道(3 回目の確認)

`Reseat` の流れ(`stopAll`、`startAll`、`finish`)を、止めた座席が残る終わり方ごとに追った。

| 終わり方 | 記録 | 案内 |
| --- | --- | --- |
| stop が失敗する | `attempted` が立つので `reseated` を書く | `Retry` が、立っていない全座席(止めた座席を含む)を `--seat` で並べる |
| 止める直前に座席の記録が変わった(`Changed`) | 先に止めた座席があれば `reseated`、無ければ何も止めていない | `Retry` は変わった座席を除いて並べる |
| 止める前の manifest の読み直しに失敗する | 同じ | `Err` に失敗の文。止めた座席があれば `Retry` |
| leader か非 leader の start が失敗する | `reseated` を書き、失敗の座席は `start_failed`、あとの座席は `not_started` | `Retry` が全部並べる |
| `reseated` の書き込みに失敗する | 失敗は `Err` に足される | `Retry` を出す。立った座席は active、立たなかった座席は `cutOff` なので次の既定の打ち直しが拾う |
| SIGINT・SIGTERM・SIGHUP・SIGPIPE | 記録まで続く | 上の行のとおり |
| SIGKILL・SIGQUIT・電源断 | `stopped` に `reseat=<id>` が残り `reseated` が無い | 同じコマンドを同じ `--seat` で打ち直すと `cutOff` で拾う。`reseat.lock` に残る印は古い pid のままだが、次に取る reseat が上書きし、読まれるのはロックが持たれているときだけ |
| 出力が読まれない(閉じた pipe、`| tee` の Ctrl-C、agent の Bash tool)で、reseat が一部失敗した | `reseated` は書かれる | 案内は読まれない stdout にしか無い。L10 |

L10 以外の道は、記録と案内のどちらかが必ず残る。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L10 | LOW | Exception handling | 出力を読む手が居ない実行(M2 の直しが生き延びるようにした閉じた pipe、`2>&1 \| tee` のあとの Ctrl-C、agent の Bash tool の中断)で reseat が一部失敗すると、残った座席と打ち直しのコマンド(`Retry`)は、読まれない stdout にしか無い。台帳の `reseated` には座席ごとの結果(`seats=<seat>:<outcome>`)が残るが、これを読むのは `reseatLifeSeats` だけで(テスト以外のコードを grep した。`ralph org status`・report・doctor に行は無い)、人には見えない。`reseated` があるので `cutOff` は立たず、次の `--seat` なしの reseat の対象は active な座席だけになる。leader が立ち、implementer の start が失敗した場合、implementer はどの出力にも名前が出ない。active な座席が 1 つも無ければ、AC-6 の断りが止まった座席と打ち方を出す(こちらは救われる)。失敗と読み手の喪失が重なったときだけ起きる。コードを読んで導いたもので、実行して確かめてはいない | `internal/org/reseat.go:1112-1135`(`finish`。`Retry` は 1121 行で `reseatRetryCommand`(1194 行)から作る)、`reseat.go:221-226`(`reseated` を読む唯一の場所)、`internal/cli/org_reseat.go:75-77`(help が「失われる」と言う) | 最小は help の 1 文: 出力を失った reseat のあとは `ralph org status --org-id <id>` で止まった座席を見て `--seat` で名指しする。もう 1 つの案は、`--seat` なしの reseat が、今の命で `reseated` の結果が `start_failed` か `not_started` の座席を note に出す |
| L2 | LOW | Maintainability | 前回のまま。dry run の herdr probe が gate の前半を写している | `org_reseat.go:326-342`、`org_herdr_server.go:127-143` | 「status を読んで running かを返す」関数を 1 つ出す |
| L3 | LOW | Error text | (b) だけ残っている。leader を対象にしない reseat で `[reviewer]` の値が不正だと、「the leader's task copies the [reviewer] values」と出る。検査は予約の比較に leader の起動の検査が要るので必要で、文だけが reseat に合わない | `feature.go:413`、`reseat.go:405` | `refuse` の文に「leader は起こさないが、予約の比較に leader の起動の検査を通す」の 1 句を足す |
| L4 | LOW | Naming | 前回のまま。関数 `reseatSeat` と型 `ReseatSeat` が頭文字の大小だけで違う | `reseat.go:77`、`reseat.go:691` | 関数を `newReseatSeat` にする |
| L5 | LOW | Readability | 前回のまま。`PlanReseat` が 147 行で、`plan.Start`・`stops`・`sim` を同時に持ち、`plan.Start[0]` が leader であることに 2 か所が頼る | `reseat.go:329-475`、391 行、403 行 | `leaderIdx` を返す形にする。leader 向けの組み立てと非 leader の `Spawn` の組み立てを別の関数にする |
| L6 | LOW | Maintainability | 前回のまま。注釈が archive へ動く plan を指す | 「前回の指摘の判定」の L6 の行の 5 か所 | plan の slug だけにするか、動かない `docs/specs/2026-11-12-org-reseat.md` を指す |

指摘にしなかったもの。テストの helper `assertReseatRefused`(戻り値なし)と `checkReseatRefused`(戻り値あり)は名前が前置きの違いだけだが、doc comment が両者の関係を書いていて、呼び手は 15 と 4 か所で迷わない。

## Positive notes

- M2 の直しは、SIGINT などを受ける `ch` と SIGPIPE の `pipe` を分け、`pipe` に読み手を置かない形にした。受け止める期間と、その間の書き込みがすべて戻り値を捨てることが doc comment に書いてある。
- doc comment が「Go は継承した SIGPIPE の無視を残さない」ので無視の分岐が要らない理由を言い、`notIgnoredSignals` の対象と分けた理由が読める。
- 子の process を使うテストが、直しが無いときに落ちる形(終了コード 3 と EPIPE 2 回の固定)で、親の `signal.Notify` で子が SIGINT を既定で始める前提も sibling のテストと同じに置いてある。
- 断りの文の `spawn` が、reseat 自身が渡す値(driver、model、scope、cwd)を並べ、記録が `""` の model は付けない。テストが両方の形を固定している。
- help の 3 か所(打ち直しの言い方、閉じた pipe、終了コード)の文が、`reseat.go` の doc comment と plan の P9 の文と食い違わない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L10: 出力を失った reseat の一部失敗の案内が台帳の `reseated` にしか残らず、どの verb も表示しない | 打ち直しの対象から外れた座席が、人に気づかれないまま止まる。失敗と読み手の喪失が重なったときだけ | help 1 文の変更で済むので、直す余地がある | 実機の director の Bash tool から reseat を打って中断した最初の報告 | 本 report の L10 |
| L2・L3・L4・L5・L6 | 前回までの report の通り | 前回までの通り | 前回までの通り | 1 回目の report(`git show 6aa5188f:docs/reports/self-review-2026-11-12-org-reseat.md`) |

_(上の表は `docs/tech-debt/README.md` へまだ足していない。直すか持ち越すかは orchestrator が決める。持ち越す指摘だけを足す。いまの `docs/tech-debt/README.md` に reseat の行は無い。パイプラインの上限(`max_pipeline_cycles` 2)には、この実行ではまだ届いていない。)_

## Recommendation

- Merge: yes。CRITICAL・HIGH・MEDIUM は無い。LOW 6 件(L10 と持ち越しの L2 から L6)は、どれも先へ進むのを止めない
- 2 回目の M2・L7・L8・L9 は、新しい穴なしに直っている。SIGPIPE を受けるあとの `main.go` の error の出力だけ、閉じた stderr で SIGPIPE の終わりになるが、記録は書かれたあとで、指摘にしない
- 止めたまま立てない道は、L10 の 1 つを除いて、記録と案内のどちらかが残る
- Follow-ups: L10 は help の 1 文で足りる。直すなら L2 から L6 と同じコミットで直せる。コードを変える直しなら、パイプラインは `/self-review` からやり直す(`post-implementation-pipeline.md`)。help の文言だけの直しも、help を固定するテストを動かす
