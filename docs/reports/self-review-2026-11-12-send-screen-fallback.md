# Self-review report: send-screen-fallback

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-send-screen-fallback.md`(issue #308)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目の 2 回目の run(1 回目の指摘を直したあと。`cycle-count.json` は cycle 1 のまま)。前の report は 5b6a62d6(LOW 8、Merge: yes)
- Scope: `git diff origin/main...HEAD`(origin/main は `d2e5c6e9`、HEAD は `c31780b4`)。69 ファイル、5784 行追加、174 行削除。前の run からの差は 3 つ。`07e01081`(L1、L3、L8 の修正)、`deddebc9`(`Send` の doc comment に 1 文)、`c31780b4`(origin/main の取り込み。`internal/cli/org.go` の status の Long、org skill の status の行 4 コピー、`docs/tech-debt/README.md` が競合)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

読んだもの。

- `internal/org/send_fallback.go`(380 行、全部。上限の追加後の `sendWaitForSeat`、`checkSendScreen`、`sendReadIdleScreen`、`sendHerdrState`、`sendScreenCall`、`sendScreenReserve`)
- `07e01081`、`deddebc9` の差分の全部と、`c31780b4` の競合 6 ファイルの解決結果
- `internal/org/verbs.go` の `Send` の doc comment(`The screen fallback` の節)と本体(`ctx` の作成から `sendScreenUnderLock` の呼び出しまで)、`capMSToContext`、`waitOrCtxDone`、`driverCallTimeout`、`callWithTimeout`、`DefaultSendEnterDelayMS`
- `internal/org/send_screen.go` の `sendScreenBoxCheck` と、その呼び出し側。比べ先の `resume.go` の `resumeAgentStatus` と `resumeLook`、`status_live_test.go` の `TestLiveSeatStates_EachCallEndsAtTheDriverCallTimeout`
- `internal/cli/org.go` の `origin/main` との差分の全部(send の help とフラグ、status の Long と `--live`、`printStatusJSON`、`printStatusTable` と、その呼び出し元 2 か所。ほかに呼び出し元が無いこと)
- 4 つの org skill(`.claude/`、`.agents/`、`templates/base/` の 2 つ)の status の行と sha256。`docs/tech-debt/README.md` の差分
- 新しいテスト 3 本(`AgentGetHangs_EndsAtTheDriverCallTimeout`、`PaneReadHangs_HerdrsWaitGetsTheRest`、`TestSendScreenBoxCheck_ChoiceDialogWithNoMarkerLine`)と、`sendFallbackOrg` の時間の定数、fake herdr の `blockRead`・`blockGet`

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- 半分の上限の算術: `sendScreenReserve` は `reserve >= 2*gap + enterDelay` のときだけ 0 以外を返すので、`reserve/2 >= gap + enterDelay/2` で、確かめの ctx は 2 回の読みの間の待ち(`waitOrCtxDone(ctx, gap)`)を必ず入れられる。既定(`--timeout-ms` 30 秒、reserve 10 秒)は 5 秒で、gap 3 秒の残り 2 秒が 3 回の herdr 呼び出しに当たる。最小(20.25 秒、reserve 6.75 秒)は 3.375 秒で、残りは 375 ms(これは L10 に書いた)
- `--timeout-ms` を超えない: `checkCtx` は `ctx`(`--timeout-ms`)の子で、確かめのあとの `AgentWait` は親の `ctx` を `capMSToContext(ctx, timeout)` で丸めて渡す。最初の待ちが `timeout - reserve` を使い切るので、確かめが上限まで使っても残りは `reserve/2` 以上ある(テストの数字は 800 ms、200 ms、約 200 ms)。確かめの途中の `return` はなく、`cancelCheck()` は全経路で呼ばれる
- エラーの文の変更: 旧い文言 `ran out of --timeout-ms between` は repo のどこにも残っていない(`git grep`)。新しい文言 `ran out of time between its two reads` は `send_fallback.go:199` の 1 か所で、doc、help、skill、テストのどれも写していない。確かめの時間切れは `--timeout-ms` の切れではなくなったので、新しい文言のほうが事実に合う
- L3 の修正の挙動: `reason == resumeReasonChoiceDialog && last >= 0` が偽のとき、`sendDialogHintLine(lines[max(last, 0):])` を通り、それも無ければ `reason != ""` の `no input box`(unknown)に落ちる。追加の doc comment とテスト(claude、codex、`ClassifySendScreen` 経由)の期待と合う
- merge の解決: status の Long は main 側の `feature_digest` の文のあとに空行を挟んで `--live` の段落が続き、両方ある。4 つの skill は sha256 が同一(`0921a99d…`)で、status の行に `feature_digest` と `--live` が 1 回ずつ入る。`orgFeatureJSON.FeatureDigest` と `payload.Feature` の代入は残り、`printStatusTable` と `printStatusJSON` の呼び出し元はどちらも新しい引数に直っている(テストからの直接の呼び出しは無い)。`docs/tech-debt/README.md` は origin/main に対して 1 行(この PR の Known gaps)だけが増え、列は隣の行と同じ 7 つ
- 前の run で確かめた項目(ロックの解放、2 つ目のキーを押さない、`ctx` の扱い、エラーの文の契約、失敗側に倒れる既定、`--live` なしの出力、`sent` の `Details` の読み手)は、この 3 コミットの差分が触れていない

読んでいない・実行していないもの。テスト、`go build`、`go vet`、linter は走らせていない。L9 の本番での動きは、読んで判断しただけで、実行して確かめていない。新しいテストの本体は、時間に関わる部分と呼び出しの列だけを読んだ。

## 前の run の指摘の状況

| ID | 状況 | 確かめたこと |
| --- | --- | --- |
| L1 | 直った(残りは L9) | 確かめの ctx を `reserve/2` で区切り(`sendWaitForSeat`)、3 つの herdr 呼び出し(`agent get`、`pane read` 2 回)を `sendScreenCall` に通した。`PaneReadHangs_HerdrsWaitGetsTheRest` が AC11(詰まった呼び出しのあとに herdr の待ちが残る)を、done が来る場合と来ない場合で固定している。1 回ごとの上限は本番では効かない(L9) |
| L2 | 残る | `sendScreenPath.checked` はそのまま |
| L3 | 直った | `last >= 0` の守りと、`last == -1` のテスト |
| L4 | 残る | `HintStateDir: orgInboxHintStateDir(…)` はそのまま(行は `org.go:1429` に動いた) |
| L5 | 残る | `herdr_state=` に入れる値の検査は無いまま |
| L6 | 残る | `isSendBusyLine` は `region` の全行に当たるまま |
| L7 | 残る | rune 切りの写し 3 か所と、driver の規則の有無を見る 4 か所はそのまま |
| L8 | 直った | `doneAfter` は 750 ms(`sendFallbackTimeoutMS-300+150`)。最初の待ちの期限(600 ms)と全体の期限(900 ms)の両方から 150 ms。テストの doc comment も「150 ms 後、`--timeout-ms` の終わりの 150 ms 前」に直っている |

`send_fallback_test.go:518` の comment が `(self-review L1)` と ID で引いている。この report は同じファイル名で上書きされるので、L1 の行を上に残し、番号は付け替えていない。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L2: `sendScreenPath.checked` は `beforeEnter` の中で `true` にされ、`Send` が `SendProgressTextTyped` のエラーの文を選ぶために読む。「`beforeEnter` のエラーか、Enter の前の待ちの中断か」を、クロージャが書いたフィールドで伝えている。`beforeEnter` が返す 2 種類のエラー(`errSendTextNotInInput` と `pane read … failed`)をまとめて見分ける型が無いのが理由 | `send_fallback.go:52-54`(型とフィールド)、`:305`(`sp.checked = true`)、`verbs.go:619`(`screenPath != nil && screenPath.checked`) | 2 種類のエラーを 1 つの型(たとえば `errSendBeforeEnter` で包む)にして `errors.As` で分ければ、フィールドとクロージャの副作用が要らなくなる。今の形でも動く(クロージャは同じ goroutine で同期的に走る) |
| LOW | naming | L4: `send` の `HintStateDir` に `orgInboxHintStateDir(cmd, resolvedStateDir)` を使っているが、この関数の doc は `org.Org.InboxHintStateDir` と inbox のコマンド用だとだけ言う。同じ `RunE` の少し上に、同じ規則(`--state-dir` を渡したときだけ解決済みの値)を `stateDirFlagSet` と `orgReadCommandHint` で書いた形がすでにある。同じ規則の言い方が 1 つの関数に 2 つ、名前は inbox 専用に見える | `internal/cli/org.go:1416`(`stateDirFlagSet := cmd.Flags().Changed("state-dir")`)、`:1429`(`HintStateDir: orgInboxHintStateDir(…)`)、`:1431`(`orgReadCommandHint(…, stateDirFlagSet)`)、`org_inbox.go:347-354`(doc) | `HintStateDir` に `stateDirFlagSet` を使った 1 行(`if stateDirFlagSet { hint = resolvedStateDir }`)にそろえるか、`orgInboxHintStateDir` の doc に send の使い方を足して名前を中立にする(`orgHintStateDir`) |
| LOW | null-safety | L5: `sent` の `Details` に入る `herdr_state=<state>` は、herdr の JSON の `agent_status` をそのまま入れる(空でなければ検査しない)。`Details` は空白区切りの `key=value ` の連なりで、あとに `submit_unconfirmed=true ` が続く。herdr が空白を含む値を返す版が出ると、トークンの境目が崩れる。今の 5 値(idle、working、blocked、done、unknown)では起きない | `send_fallback.go:64`(`"via=screen herdr_state=" + sp.herdrState + " "`)、`resume.go:479`(`herdrAgentStatus` は空だけ弾く) | 5 値以外は `sendHerdrStateUnknown` に丸めるか、`strings.Fields` で 1 語にする。`status --live` の table 側(`HerdrText`)も同じ値を列に入れる |
| LOW | maintainability | L6: 動作中の行の判定(`isSendBusyLine`)は、`region` と footer の全部の行に当てる。`region` は最後の prompt のあとから入力欄までで、assistant の返事の本文も入る。`sendClaudeBusyLine`(`^[*·✢✳✶✻✽] \S+…(?:\s|$)`)は、行頭の `* word…` や `· word…` の形の返事の行(箇条書きなど)にも合い、busy が turn-end の確認(4 段目)より先に決まるので、手番が終わった画面を `busy` と読む。codex の `esc to interrupt` を行のどこでも拾う規則も同じ。打たない側に倒れるので安全だが、フォールバックが黙って効かなくなる側の誤りで、エラーの文には busy の行が出る。正規表現を読んだ推論で、実画面や実行では確かめていない | `send_screen.go:129`(`sendClaudeBusyLine`)、`:378`(`sendScreenBusyCheck`)と、`region`・footer の全行を回すループ `:387`、`:519`(`isSendBusyLine`) | 緊急ではない。起きたら busy の行を fixture にして、busy の規則を最後の turn-end の行より下(または最後の block)に限るかを決める。計画の Risks の「動作中の行の文が CLI の版で変わる」と同じ型なので、tech-debt の Known gaps(b)の行に 1 文足すとよい |
| LOW | readability | L7: 小さな写しが 2 種類。(a) 「rune で切って `…` を足す」が 3 か所(`send_fallback.go:372-374`、`status_live.go:129-131`、`status_live.go:151-153`)。`ScreenText` は 60 rune で切ってから `sendScreenText` に渡し、そこでも 120 rune の切りが走る(今は 2 度目が何もしない)。(b) 「その driver に画面の規則があるか」を `resumeScreenSentences[driver]` の存在で調べる箇所が 4 つ | (b)は `send_screen.go:185`(`ClassifySendScreen`)、`:214`(`SendScreenSubmitted`)、`send_fallback.go:105`(`sendScreenReserve`)、`status_live.go:109`(`liveScreen`)。resume 自身の 2 か所(`resume_screens.go:297,348`)は既存 | `cutRunes(s string, max int) string` と `hasSendScreenRules(driver string) bool` を 1 つずつ作る。driver を足す(claude、codex 以外)ときに 4 か所を探さずに済み、grep もしやすい |
| LOW | maintainability | L9: `sendScreenCall` の 1 回ごとの上限(`driverCallTimeout`、10 秒)は、本番では先に切れることがない。確かめの ctx は `reserve/2` で、reserve は `min(defaultSendScreenReserve = 10 秒, --timeout-ms/3)` なので、確かめは最長 5 秒。CLI は `Org.SendScreenReserve` も `Org.DriverCallTimeout` も設定しない(本番コードの代入は `git grep` でゼロ、テストだけが設定する)ので、`agent get` か `pane read` が詰まったとき先に切れるのはいつも確かめの ctx。1 回ごとの上限が先に切れるのは、`DriverCallTimeout` を半分の reserve より短くしたテストの中だけ。影響は 2 つ。(a) `AgentGetHangs_EndsAtTheDriverCallTimeout` の doc が書く「`agent get` が詰まっても確かめは画面を読み続け、入力待ちの座席には画面で届く」は、本番では起きない。本番では `agent get` が確かめの 5 秒を使い切り、2 回の読みは ctx がすでに切れた状態で始まって失敗し(`exec.CommandContext` は切れた ctx では起動せずに ctx のエラーを返す、はず。未確認)、エラーの文は `herdr state unknown, the screen could not be read (herdr pane read %s failed: context deadline exceeded)` になる。pane read は実際には走っていないのに「失敗した」と読める。(b) `Send` の doc comment と `checkSendScreen` の doc comment は「各 herdr 呼び出しは driver call timeout まで(`sendScreenCall`)。だから詰まった呼び出しは残りを herdr の待ちに残す」と書くが、残りを守っているのは半分の上限で、`sendScreenCall` ではない。L1 の「sibling が持つ 1 回ごとの上限を新しい読みも持つ」は形だけ満たし、効果は無い | `send_fallback.go:149`(`context.WithTimeout(ctx, reserve/2)`)、`:259-267`(`sendScreenCall`)、`:25`、`:30`(既定の gap と reserve の最大)、`verbs.go:1025`(`defaultDriverCallTimeout = 10 * time.Second`)、`verbs.go:357-363`(`Send` の doc の追加分)、`send_fallback_test.go:486-494`(`AgentGetHangs…` の doc と、`:494` の `o.DriverCallTimeout = 30 * time.Millisecond`)。比べ先は `resume.go:434`(`resumeLook` の pane read)と `:457`(`resumeAgentStatus`)。`callWithTimeout`(`verbs.go:1053`)は `context.Background()` から作るので、1 回ごとの上限が唯一の上限 | どちらかに寄せる。(1)`sendScreenCall` を外し、`blockGet` のテストを「確かめの ctx で切れ、herdr の state は unknown、画面は読めず、herdr の待ちに戻る」に直す。上限が 1 つになり、doc の説明も半分の上限だけで済む。(2)詰まった `agent get` が 2 回の読みを食わないことを残したいなら、1 回ごとの上限を確かめの時間に合わせて短くする(`min(driverCallTimeout, check/3)` など)。どちらでもない今の形を残すなら、`sendScreenCall` と `Send` の doc に「`DriverCallTimeout` を半分の reserve より短くしたときだけ効く」と書き、AC11 の説明を半分の上限に付け替える。あわせて、確かめの ctx がすでに切れているときの verdict を「pane read が失敗した」ではなく「時間切れ」にすると、エラーの文が事実に合う |
| LOW | maintainability | L10: reserve の算術を説明する doc が、確かめの上限(`reserve/2`)を足したあとの配分と合わない。`sendScreenReserve` の doc は最小の reserve(読みの間 2 回分と Enter の前の待ち)の内訳を「1 つは 2 回の読みの間、もう 1 つは残り。herdr の呼び出し(`agent get`、pane read 4 回、send-text、send-keys)と Enter のあとの確かめ」と言う。今は確かめが `reserve/2` = 最小で `gap + enterDelay/2` までなので、確かめ自身の `agent get` と pane read 2 回に使えるのは、gap を引いた `enterDelay/2`(既定で 375 ms)だけ。`sendWaitForSeat` の新しい comment(「gap と、Enter の前の待ちの半分」)はこちらを言っていて、同じ 6.75 秒に 2 通りの説明がある。`defaultSendScreenReserve` の doc(「確かめのために、herdr の最初の待ちから取る最大」)も、確かめは半分までという今の意味と違う。`--timeout-ms` が 20.25 秒から数秒のあたり(確かめが 3.4 秒前後)で、3 回の herdr 呼び出しが 375 ms に収まらないと、確かめは `ran out of time between its two reads` か `could not be read` で herdr の待ちに戻る。安全側だが、help と skill が言う「20250 未満は画面を読まない」の境目は、画面が実際に効く境目より低い。herdr 呼び出しの実測は証拠のファイルに見当たらず、375 ms で足りるかどうかは未確認 | `send_fallback.go:26-30`(`defaultSendScreenReserve` の doc)、`:91-100`(`sendScreenReserve` の doc)、`:109`(`reserve < 2*gap+enterDelay`)、`:146-149`(新しい comment と上限) | doc だけ直すのが安い。`sendScreenReserve` の内訳を「確かめは reserve の半分まで(読みの間の gap と、呼び出しに enterDelay の半分)。残りの半分が、ロックのあとの読み、send-text、Enter、確かめ」に直し、`defaultSendScreenReserve` の doc も合わせる。境目を実際に合わせるなら `reserve/2 >= gap + 呼び出しの余裕` を式にするが、help と skill の 20250 が変わるので、実測を取ってからにする |

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 7(前の run からの残り 5、新しい 2)。

## Positive notes

- 前の run の LOW のうち、L1(確かめの herdr 呼び出しが詰まっても `--timeout-ms` を食い尽くさない)、L3(`lines[last]` の守り)、L8(テストの余裕)は、修正のコミットが 1 つで、それぞれにテストが付いている。L1 の修正は半分の上限 1 行で AC11 を守り、テストが「詰まった呼び出し」と「done が来る・来ない」の両方を固定している
- 追加の doc comment が、半分の上限を `sendWaitForSeat`、`checkSendScreen`、`Send` の 3 か所に同じ言葉で書いている(書き分けの食い違いは L9 と L10 に出した範囲)。エラーの文の変更(`--timeout-ms` から `time` へ)も、旧い文言を repo に残していない
- merge の解決は、競合した 6 ファイルで両方の側を残している。status の Long と skill の行は、main の `feature_digest` と、この branch の `--live` の両方があり、4 つの skill のコピーは同一
- L3 の修正は、失敗側の既定(`unknown`、入力欄なし)に倒す。追加の doc comment が「今は起きない」ことと、起きたときの扱いの両方を言い、テストが 2 つの driver で `sendScreenBoxCheck` と `ClassifySendScreen` の両方を通す

観察として残すもの(指摘にはしない)。

- `sendScreenUnderLock`、`beforeEnter`、`confirm` の `PaneRead` は、1 回ごとの上限を持たず `--timeout-ms` 全体の `ctx` で動く。確かめのあとの道で、詰まっても `--timeout-ms` は超えず、何も打たない(または `SendProgressTextTyped`)で終わるので、L1 の AC11 の対象ではない。ただし `sendScreenCall` の doc が「resume と `status --live` と同じ」と言うので、この 3 つが違うことも L9 の整理のときに書くとよい
- `PaneReadHangs_HerdrsWaitGetsTheRest` の余裕は約 100 ms(2 回目の待ちの `got[1] < 100` に対し、残りは約 200 ms)。L8 と同じ種類だが、両側とも 100 ms 以上ある。CI で落ちるようなら `got[1] >= 1` まで緩めてよい(残りが正であることだけが契約)
- `LiveSeatStates` が座席ごとに最悪 20 秒、`sendScreenReserve` が遅い環境で最小付近の `--timeout-ms` を断りうる、という前の run の 2 つの観察は、この差分で変わっていない
- 画面から取ったエラーの文が seat の出力を含む点も同じ扱い(引用と長さの上限がある)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L2、L4、L5、L6、L7、L9、L10 の 7 件(すべて LOW)。`sendScreenPath.checked` の副作用(L2)、`orgInboxHintStateDir` の流用(L4)、`herdr_state` の検査(L5)、動作中の行の規則が返事の本文にも当たる(L6)、rune 切りと driver の規則の写し(L7)、本番で効かない 1 回ごとの上限(L9)、reserve の算術の doc(L10) | どれも失敗側に倒れるか、今の入力では起きない。L9 は doc とテストの説明が本番の動きと合わない。L6 は返事の本文の形によってフォールバックが効かない | LOW だけで Merge は yes。コードを直すとパイプラインを最初からやり直す(cycle 1 の上限は 2)ので、この PR では直さず、register に回すのが安い。この run で直らなかった LOW は、PR のあと plan が archive されるので、register に行がなければどこにも残らない | `send_fallback.go`、`send_screen.go`、`status_live.go` のどれかを次に触るとき(L2、L7、L9、L10)。`orgInboxHintStateDir` か send の `HintStateDir` を触るとき(L4)。herdr に `agent_status` の新しい値が出たとき(L5)。送信エラーに `busy` と出る画面を実機で見たとき(L6)。`DriverCallTimeout` か `SendScreenReserve` を CLI から設定できるようにするとき(L9) | `docs/plans/active/2026-11-12-send-screen-fallback.md`(D1、D4、Risks)、この report |

_`docs/tech-debt/README.md` への追記は、このコミットを report だけにする指示のため、この run では行っていない。register の行にするなら、オーケストレーターが上の行を 1 行にまとめて追記する。この PR がすでに足した Known gaps の行(a から c)とは別の内容で、重ならない。前の run の L1、L3、L8 は直ったので、行に入れない。_

## Recommendation

- Merge: yes
- Follow-ups: CRITICAL、HIGH、MEDIUM は無い。前の run の L1、L3、L8 は直り、S6 の修正(半分の上限、1 回ごとの上限、エラーの文の変更)は `--timeout-ms` を超える道を作らず、詰まった呼び出しのあとの herdr の待ちを残している。merge の解決も、status の Long と skill の行の両方で、main 側と branch 側の内容を残している。残る LOW 7 件はどれも merge を止めない。L9 だけは、doc とテストが本番で起きない動きを説明しているので、次にコードを触る機会(cross-review の指摘を直すときなど)があれば、一緒に直す。直さないなら上の行を register に 1 行で回す。この PR で直す場合は、post-implementation-pipeline.md の規則どおり `/self-review` から全体をやり直す
