# Self-review report: send-screen-fallback

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-send-screen-fallback.md`(issue #308)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff d4bafaa7...HEAD`(HEAD `b5fde71c`、base は origin/main との merge-base `d4bafaa7`)。68 ファイル、5527 行追加、174 行削除(testdata の実機画面 23 枚と証拠の記録を含む)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

読んだもの。

- `internal/org/send_fallback.go`(349 行、全部)
- `internal/org/send_screen.go`(568 行、全部。正規表現 11 本、`ClassifySendScreen` の 4 段、`SendScreenSubmitted`、`sendInputHolds` の 3 つの規則)
- `internal/org/status_live.go`(156 行、全部)
- `internal/org/verbs.go` の差分の全部と、`Send` の本体(`ctx` の作成から `sent` の記録まで)、`typeAndSubmit`、`confirmSubmitted`、`capMSToContext`、`waitOrCtxDone`、`sendPaneLockWait`
- `internal/cli/org.go` の差分の全部(`send` の help とフラグ、`status --live`、`printStatusTable`、`printStatusJSON`、`printLiveMismatches`)と、比べ先の `orgInboxHintStateDir`(`org_inbox.go`)、`orgReadCommandHint`
- `internal/org/resume.go`、`resume_screens.go`、`pane_lock.go`、`spawn.go`、`prompts/leader.md` の差分の全部。比べ先として `resumeAgentStatus`、`herdrAgentStatus`、`findResumeInputBox`、`codexFooterStart`、`callWithTimeout`
- `docs/tech-debt/README.md` の差分(1 行追加。この PR の Known gaps)と、`internal/cli/org_send_screen_fallback_test.go` の全部、`send_fallback_test.go` の fake herdr と時間に関わるテスト、`status_live_test.go` のテスト一覧
- 追加行に不可視 Unicode(U+2028、U+200B、U+FEFF、U+FFFD)は無い。evidence、testdata、recipe に `/Users/`、`hiroki`、メールアドレス、`/home/` は無い

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- ロックの解放: `Send` は `defer unlock()` と `sync.OnceFunc` の `unlock()`(`typeAndSubmit` の直後)の組のまま。新しい経路 `sendScreenUnderLock` と `sendWaitUnderLock` はどちらも `error` を返すだけで、ロックを触らない。ロックを取ったあとの `return` は全部 deferred で解放される
- 2 つ目のキーを押さない: `PaneSendKeys` を呼ぶのは `verbs.go:840`(`typeAndSubmit` の Enter 1 回)と、無関係な `verbs.go:1532`(`C-c`、stop 側)だけ。`send_fallback.go`、`send_screen.go`、`status_live.go` に呼び出しは無い。Enter のあとの `confirm` は `PaneRead` を繰り返すだけ
- 時間の算術: `sendScreenReserve` の `reserve < 2*gap + enterDelay` は、doc comment の「既定で 6.75 秒、`--timeout-ms` 20.25 秒未満はフォールバックなし」と、help の「3 分の 1 が 6 秒に Enter の前の待ちを足した値より短ければ」「Enter の前の待ちが 4 秒を超えると」と合う(`10 ≥ 6 + enterDelay` が 4 秒)。`sendFirstWait` の期限は `timeout - reserve` で親の `ctx` の期限より前なので、`ranOut`(`waitCtx.Err()` が `DeadlineExceeded`)は親の期限切れと取り違えない。herdr 自身の `--timeout` は `waitCtx` を作ったあとにプロセスが起動するので、ctx のほうが先に切れる
- `ctx` の扱い: 画面の確かめの後ろの `AgentWait` は `capMSToContext(ctx, timeout)` で残りに丸める(最低 1 ms)。確かめの途中で `ctx` が切れても `waitOrCtxDone` が `ctx.Err()` を返し、`verdict` に理由が入る。`confirm` のループは `ctx` に `window` の期限を重ねていて、`PaneRead` が失敗しても `waitOrCtxDone` が返るので、止まらないループにはならない
- エラーの文: 新しい `SendResult.Err` はすべて `org: send: ` で始まる(契約どおり)。画面の行は `strconv.Quote`(制御文字をエスケープ)と 120 rune の切り詰めを通る。次の手のコマンドは `ShellWord` と `VerbCommand` で組む。`%w` の連鎖(`sendScreenWaitErr` → `sendWaitErr`)は `errors.Is` が通る
- 失敗側に倒れる既定: `ClassifySendScreen` は未知の driver、入力欄が無い、入力欄に文がある、判断できない、で `unknown`。`idle` を返すのは「終わりの行が最後の状態の行」か「バナーがあり prompt が無い」の 2 つだけ。`sendScreenReserve` は driver が無い座席で 0(今までの動き)。`checkSendScreen` は 2 回の読みが 1 バイトでも違えば入力待ちとしない。`sendInputHolds` は未知の driver、空白だけの文、入力欄が無い画面で false
- インデックスの範囲: `lines[box.footerStart:]`、`lines[:box.aboveEnd]`、`sendCodexTypedInputText` の `lines[start]`、`sendClaudePastesHold` の `loc[2]:loc[3]`(正規表現に捕捉グループが 1 つで必須)は、どれも範囲内。`Atoi` の失敗も見ている
- `--live` なしの出力: table の見出しは `live != nil` のときだけ列が増える。JSON の `Live` は `*orgSeatLiveJSON` に `omitempty` で、`--live` なしでは出ない。`status` の runtime は `newOrgRuntimeAt` で `Herdr` を持つが、`--live` なしでは呼ばない(`TestOrgStatus_WithoutLiveTheOutputIsUnchangedAndHerdrIsNotCalled` が固定している)
- `sent` の `Details`: 読む側は `watch.go` の `leaderActivityEventCount`(`Event == EventSent` だけを数え、`Details` は読まない)で、先頭に `via=screen herdr_state=<state> ` が入って困る消費者は repo に無い

読んでいない・実行していないもの。テスト、`go build`、`go vet`、linter は走らせていない。正規表現は読んで判断しただけで、実行して試していない(`/verify`、`/test` の担当)。`send_screen_test.go` と `send_fallback_test.go` の本体の大半は、時間に関わる部分以外は読んでいない。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | exception-handling | L1: 画面の確かめの herdr 呼び出し(`agent get`、`pane read` 2 回)は、`--timeout-ms` 全体の `ctx` で動き、1 回ごとの上限も、確かめの分(reserve)での上限も無い。resume と `status --live` は同じ呼び出しを `callWithTimeout`(`driverCallTimeout`、10 秒)で 1 回ずつ区切っている。1 つが詰まると、残りの `--timeout-ms` を全部使い、AC11 の「herdr の待ちに戻る」分が残らない。`sendHerdrState` の doc は「as Resume reads it」と書くが、Resume は呼び出しごとに区切る点が違う。`status --live` には `TestLiveSeatStates_EachCallEndsAtTheDriverCallTimeout` があるが、send の確かめには同じ種類のテストが無い | `send_fallback.go:223-228` の `sendHerdrState`(`probe.AgentGet(ctx, name)`)、`:209` の `sendReadIdleScreen`(`o.Herdr.PaneRead(ctx, …)`)、`:244` の `sendScreenUnderLock`。比べ先は `resume.go:434,457`(`callWithTimeout`)、`status_live.go:liveScreen` と `liveHerdrState`、`status_live_test.go:204` | `sendWaitForSeat` で `checkSendScreen` に渡す `ctx` を `context.WithTimeout(ctx, reserve)` で区切るのが 1 行で済む。1 回ごとの区切りにするなら、3 つの呼び出しを `callWithTimeout` の形(`min(driverCallTimeout, 残り)`)に揃える。実害は herdr が `agent get` か `pane read` だけ詰まる場合で、そのとき今の動きは、確かめの前と同じ時間切れのエラーになるだけ |
| LOW | maintainability | L2: `sendScreenPath.checked` は `beforeEnter` の中で `true` にされ、`Send` が `SendProgressTextTyped` のエラーの文を選ぶために読む。「`beforeEnter` のエラーか、Enter の前の待ちの中断か」を、クロージャが書いたフィールドで伝えている。`checked` は `beforeEnter` が走った印でしかなく、`beforeEnter` が返す 2 種類のエラー(`errSendTextNotInInput` と `pane read … failed`)をまとめて見分ける型が無いのが理由 | `send_fallback.go:52-54`(型とフィールド)、`:275`(`sp.checked = true`)、`verbs.go:616`(`screenPath != nil && screenPath.checked`)| 2 種類のエラーを 1 つの型(たとえば `errSendBeforeEnter` で包む)にして `errors.As` か `errors.Is` で分ければ、フィールドとクロージャの副作用が要らなくなる。今の形でも動く(クロージャは同じ goroutine で同期的に走る) |
| LOW | null-safety | L3: `sendScreenBoxCheck` の `strings.TrimSpace(lines[last])` は、`last`(`sendLastMarkerLine`)が -1 のとき範囲外になる。今は `reason == resumeReasonChoiceDialog` なら marker 行が必ずあるので起きないが、その保証は `findResumeInputBox` の中の同じ走査(marker で始まる最後の行)に頼っていて、`sendLastMarkerLine` はその走査の写し。直後の `lines[max(last, 0):]` は -1 を考えて書いているので、`lines[last]` だけ守りが抜けている | `send_screen.go:353-356`(`sendScreenBoxCheck`)、`:493`(`sendLastMarkerLine`)、`resume_screens.go:379-396`(`findResumeInputBox` の走査と choice の分岐) | `if reason == resumeReasonChoiceDialog && last >= 0` にして、`last < 0` なら下の「入力欄が無い」へ落とす(失敗側に倒れる)。または `findResumeInputBox` が marker の行を返すようにして、走査を 1 か所にする |
| LOW | naming | L4: `send` の `HintStateDir` に `orgInboxHintStateDir(cmd, resolvedStateDir)` を使っているが、この関数の doc は `org.Org.InboxHintStateDir` と inbox のコマンド用だとだけ言う。同じ `RunE` の 2 行上に、同じ規則(`--state-dir` を渡したときだけ解決済みの値)を `stateDirFlagSet` と `orgReadCommandHint` で書いた形がすでにある。同じ規則の言い方が 1 つの関数に 2 つ、名前は inbox 専用に見える | `internal/cli/org.go:1405`(`stateDirFlagSet := cmd.Flags().Changed("state-dir")`)、`:1418`(`HintStateDir: orgInboxHintStateDir(…)`)、`:1420`(`orgReadCommandHint(…, stateDirFlagSet)`)、`org_inbox.go:347-360`(doc) | `HintStateDir` に `stateDirFlagSet` を使った 1 行(`if stateDirFlagSet { hint = resolvedStateDir }`)にそろえるか、`orgInboxHintStateDir` の doc に send の使い方を足して名前を中立にする(`orgHintStateDir`) |
| LOW | null-safety | L5: `sent` の `Details` に入る `herdr_state=<state>` は、herdr の JSON の `agent_status` をそのまま入れる(空でなければ検査しない)。`Details` は空白区切りの `key=value ` の連なりで、あとに `submit_unconfirmed=true ` が続く。herdr が空白を含む値を返す版が出ると、トークンの境目が崩れる。今の 5 値(idle、working、blocked、done、unknown)では起きない | `send_fallback.go:64`(`"via=screen herdr_state=" + sp.herdrState + " "`)、`resume.go:479-494`(`herdrAgentStatus`は空だけ弾く) | 5 値以外は `sendHerdrStateUnknown` に丸めるか、`strings.Fields` で 1 語にする。`status --live` の table 側(`HerdrText`)も同じ値を列に入れる |
| LOW | maintainability | L6: 動作中の行の判定(`isSendBusyLine`)は、`region` と footer の全部の行に当てる。`region` は最後の prompt のあとから入力欄までで、assistant の返事の本文も入る。`sendClaudeBusyLine`(`^[*·✢✳✶✻✽] \S+…(?:\s|$)`)は、行頭の `* word…` や `· word…` の形の返事の行(箇条書きなど)にも合い、busy が turn-end の確認(4 段目)より先に決まるので、手番が終わった画面を `busy` と読む。codex の `esc to interrupt` を行のどこでも拾う規則も同じ。打たない側に倒れるので安全だが、フォールバックが黙って効かなくなる側の誤りで、エラーの文には busy の行が出る。正規表現を読んだ推論で、実画面や実行では確かめていない | `send_screen.go:129`(`sendClaudeBusyLine`)、`:376-395`(`sendScreenBusyCheck` と、`region`・footer の全行を回すループ `:385`)、`:517`(`isSendBusyLine`) | 緊急ではない。起きたら busy の行を fixture にして、busy の規則を最後の turn-end の行より下(または最後の block)に限るかを決める。計画の Risks の「動作中の行の文が CLI の版で変わる」と同じ型なので、tech-debt の Known gaps(b)の行に 1 文足すとよい |
| LOW | readability | L7: 小さな写しが 2 種類。(a) 「rune で切って `…` を足す」が 3 か所(`send_fallback.go:344`、`status_live.go:131`、`status_live.go:153`)。`ScreenText` は 60 rune で切ってから `sendScreenText` に渡し、そこでも 120 rune の切りが走る(今は 2 度目が何もしない)。(b) 「その driver に画面の規則があるか」を `resumeScreenSentences[driver]` の存在で調べる箇所が 4 つ(`ClassifySendScreen`、`SendScreenSubmitted`、`sendScreenReserve`、`liveScreen`) | (b)は `send_screen.go:185`(`ClassifySendScreen`)、`:214`(`SendScreenSubmitted`)、`send_fallback.go:105`(`sendScreenReserve`)、`status_live.go:109`(`liveScreen`)。resume 自身の 2 か所(`resume_screens.go:297,348`)は既存 | `cutRunes(s string, max int) string` と `hasSendScreenRules(driver string) bool` を 1 つずつ作る。driver を足す(claude、codex 以外)ときに 4 か所を探さずに済み、grep もしやすい |
| LOW | maintainability | L8: `TestOrgSendScreen_ScreenBusyThenHerdrDone_SendsByHerdrsPath` の余裕が 50 ms。herdr の最初の待ちは ctx の期限 600 ms で切れなければならず、fake の `done`(`doneAfter` = 650 ms、`time.After`)がそれより遅いことで順序を決めている。ctx の期限は `AfterFunc` の goroutine が `Done` を閉じて届くのに対し、fake の `time.After` は timer が直接送るので、負荷や `-race` で goroutine の起動が 50 ms 以上遅れると順序が入れ替わりうる。入れ替わると最初の待ちが `done` を返して herdr の道に直行し、`get`/`read` が呼ばれないので `want` と合わず落ちる。もう一方の余裕(2 回目の待ちは 900 ms の期限まで 250 ms)と釣り合っていない。負荷をかけて確かめてはいない | `send_fallback_test.go:430`(`h.doneAfter = time.Duration(sendFallbackTimeoutMS-300+50) * time.Millisecond`)、`:458`(同じ式。こちらは入れ替わっても `readCount() == 0` が通るので落ちない)、fake は `:37-57` | `doneAfter` を 2 つの限界の真ん中(`sendFallbackTimeoutMS-300+150` = 750 ms)にする。余裕は両側で約 150 ms になる |

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 8。

## Positive notes

- 画面のフォールバックの手順が、`Send` の中の長い手続きとして増えず、`sendWaitForSeat`(待ちと確かめ)、`sendScreenUnderLock`(ロックの下の読み直し)、`sendScreenSubmitChecks`(Enter の前後の確かめ)、`sendWaitUnderLock`(今までの道)に分かれた。`Send` は「経路を 1 つ選ぶ」だけを持ち、`screenPath == nil` が「今までの道」で、`details()` が nil の受け手で `""` を返すため、herdr の道の `Details` は今までと同じ
- `NoScreenFallback` と、reserve が 0 になる 3 つの条件(driver が claude/codex でない、reserve が短すぎる、フラグ)が、すべて「今までの動き」(herdr の待ちを `--timeout-ms` 全部)に落ちる。新しい挙動が入るのは、確かめられたときだけ
- 時間の定数(`defaultSendScreenReadGap`、`defaultSendScreenReserve`、`sendScreenConfirmInterval`、`sendScreenLineMaxRunes`、`liveScreenLineMaxRunes`、`liveErrMaxRunes`)に名前と、根拠(実機の採取の節)が付いている。テスト用に `Org.SendScreenReadGap`/`SendScreenReserve` で縮められる
- `sendScreenText` と `ScreenText` が同じ書式を共有し、`send` のエラーと `status --live` の列で、同じ画面が同じ言葉で出る。エラーの末尾の次の手(`ralph org read`、`status --live`、`herdr agent explain`)は、`--state-dir` を渡したときだけ付く
- `SendProgressTextTyped` の意味の拡張が、doc comment(`SendProgress`、`Send`、`typeAndSubmit`)と CLI の `note:`(「入力欄に文が無い画面なら Enter は押さない」)の両方に書かれ、Enter を押さない側へ誘導している
- 既存の fixture に対する `ClassifyResumeScreen` の判定を変えずに、`resumeCodexFooterLine` の正規表現だけを広げた(`(?: · .+)?$` を足した 1 か所)。doc comment に codex 0.166.0 の実機の行が例として入っている

観察として残すもの(指摘にはしない)。

- `LiveSeatStates` は座席を 1 つずつ、`agent get` と `pane read` を各 10 秒の上限で呼ぶ。herdr が詰まっていると、座席が N なら最悪 N × 20 秒になる。読み取り専用の診断コマンドで、resume も同じ作りなので、今は直さない。座席が 10 を超える org を使い始めたら、全体の上限を足す
- `sendScreenReserve` の最小の reserve(6.75 秒)は、「残りの herdr 呼び出し約 9 回」を 1 つの read gap(3 秒)で見積もる。呼び出しが遅い環境では、`--timeout-ms` が 20 から 25 秒のあたりで、画面の道に入っても予算の確認(`remaining <= enterDelay`)で打たずに断られうる。断られる側に倒れ、エラーの文が `--timeout-ms` を上げる案内を出すので、指摘にはしない
- 画面から取ったエラーの文には、判定を決めた行の先頭 120 rune が入る(入力欄に文がある場合は入力欄の文)。leader の LLM が読むエラーに seat の出力が入るが、`ralph org read` が全画面を渡す設計と同じ信頼の扱いで、引用と長さの上限が付いている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L1 から L8 の 8 件(すべて LOW)。画面の確かめの herdr 呼び出しに 1 回ごとの上限が無い(L1)、`sendScreenPath.checked` の副作用(L2)、`lines[last]` の守り(L3)、`orgInboxHintStateDir` の流用(L4)、`herdr_state` の検査(L5)、動作中の行の規則が返事の本文にも当たる(L6)、rune 切りと driver の規則の写し(L7)、テストの余裕 50 ms(L8) | どれも失敗側に倒れるか、今の入力では起きない。L1 は herdr が一部の呼び出しだけ詰まるときに、AC11 の分が残らない。L6 は返事の本文の形によってフォールバックが効かない。L8 は負荷でテストが不安定になる恐れ | LOW だけで Merge は yes。コードを直すとパイプラインを最初からやり直す(cycle 1 の上限は 2)ので、この PR では直さず、register に回すのが安い | `send_fallback.go`、`send_screen.go`、`status_live.go` のどれかを次に触るとき(L1、L2、L3、L7)。`orgInboxHintStateDir` か send の `HintStateDir` を触るとき(L4)。herdr に `agent_status` の新しい値が出たとき(L5)。送信エラーに `busy` と出る画面を実機で見たとき(L6)。`send_fallback_test.go` が CI で落ちたとき(L8) | `docs/plans/active/2026-11-12-send-screen-fallback.md`(D1、D4、Risks)、この report |

_`docs/tech-debt/README.md` への追記は、このコミットを report だけにする指示のため、この run では行っていない。register の行にするなら、オーケストレーターが上の行を 1 行にまとめて追記する。この PR がすでに足した Known gaps の行(a から c)とは別の内容で、重ならない。_

## Recommendation

- Merge: yes
- Follow-ups: CRITICAL、HIGH、MEDIUM は無い。LOW 8 件はどれも merge を止めない。L1(確かめの呼び出しを reserve か `driverCallTimeout` で区切る)と L8(`doneAfter` を 750 ms にする)は 1 行から数行の直しで、ほかの 6 件と一緒に register へ回すか、次に同じファイルを触るときに直す。この PR で直す場合は、post-implementation-pipeline.md の規則どおり `/self-review` から全体をやり直す
