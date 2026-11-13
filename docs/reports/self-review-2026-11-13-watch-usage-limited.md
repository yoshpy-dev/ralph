# Self-review report: watch-usage-limited

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-watch-usage-limited.md`(issue #312 の受け入れ条件 1)
- Branch: `feat/watch-usage-limited`、base は `origin/main` の `4220ea7c`(ブランチは `78611dbc` で取り込み済み)。最初のレビューの HEAD は `c1ea2017`。更新(下の「Update after e25def99」)は `e25def99`(L1、L2、L4、L6、L7 の直しと L3 のコメント)と `cc9d802b`(計画 D4 の直し)を読んだもので、HEAD は `cc9d802b`。cycle 2(下の「Update for cycle 2」)は cycle 1 の cross-review の ACTION_REQUIRED の直し `34e5bd86` と `193b044a` を読んだもので、HEAD は `193b044a`。最初の表と指摘の行番号は `c1ea2017` のもの。cycle 2 で足した行番号は `193b044a` のもの
- Reviewer: reviewer サブエージェント(`/self-review`)。**Cycle: 2**(パイプラインの 2 回目、`cycle-count.json` は `cycle: 2`、triage の見出しは `Cycle: 1/2` なので上限に当たる回)。cycle 1 の run のうちに直しのコミットを見て更新したもの(cycle は進めていない)を残し、cycle 2 の節を足した
- Scope: `git diff origin/main...HEAD` のうち Go の本体(`internal/org/watch_usage_limit.go`、`usage_limit_screen.go`、`watch.go`、`escalate.go`、`resume_screens.go`、`internal/cli/org_watch.go`)を全部読んだ。周辺として `watch_recurrence.go`、`inbox.go`、`protocol.go`、`status_live.go`、`spawn.go` の `HerdrClient` を読んだ。テスト(`watch_usage_limit_test.go`、`usage_limit_screen_test.go`)は関数の一覧と、座席が増えたときの表と画面の端の表だけ読んだ。skill、仕様、AGENTS.md、README の差分は diff の質の面で一度見ただけで、drift は見ていない。diff の質だけを見た。テストの実行、静的解析、gofmt、仕様適合、文書の drift は見ていない

## Evidence reviewed

依頼で挙げられた点を、コードを読んで追った結果。

| 点 | 結果 | 根拠 |
| --- | --- | --- |
| 状態ファイルの形と後方互換 | 問題なし。`usage_limited_seats` と pending の `usage_limited` はどちらも `omitempty` で、`loadWatchStatus` は `json.Unmarshal` のみ(`DisallowUnknownFields` なし)なので、古いファイルは空として読め、新しいファイルを古い binary が読んでも落ちない | `watch.go:270`、`watch.go:329`、`watch.go:371-384` |
| stall の判定と leader の待ちの除外の順序 | 意図どおり。`stalled` を計算し、立っているときだけ画面を読み、上限なら `stalled=false`、そのあとで leader にだけ `leaderIsWaiting` を当てる。上限で止まった leader は除外の前に集合へ入る(AC11) | `watch.go:1316-1324` |
| 集合の増加の決め方(open / acked / resolved) | 意図どおり。増えたときだけ `grown` で、open なら recurred、acked は `false` を返して新しい件、resolve 済みは `openAlertItemForKey` が拾わず新しい件。`openAlertItemForKey` は ID が大きい方を返すので、e1(acked)と e2(新しい open)が並んだあとの増加は e2 に当たる | `watch_usage_limit.go:141-169`、`:199-229`、`watch_recurrence.go:75-83` |
| `ErrInboxResolved` の競合 | 意図どおり。`beforeAppendRecurred` の隙間で resolve されると `errors.Is(err, ErrInboxResolved)` で stderr なしの `false`、新しい件になる。テストの表に行がある | `watch_usage_limit.go:218-225`、`watch_usage_limit_test.go:462` |
| `escalateToInbox` が usage_limited を acked の件に畳まない | 守られている。`!isUsageLimitedAlert(alertID)` で `escalateAsRecurrence` の道を閉じている。`alertID` と同じ件、`recurred` に同じ `alert_id` がある件は先の 2 つの分岐が拾うので、再起動のリプレイは二重にならない | `watch.go:1999-2010` |
| 理由の選び方 | 条件が先。`watchEscalationReason` は usage_limited なら `usage_limit`、それ以外は今までの Subject の規則で、他の条件の結果は変わらない | `watch.go:1929-1937` |
| 本文の上限と制御文字 | 問題なし。上限は rune 数(`protocol.Validate`)で、予算は見出しと前置きも引いた数なので安全側。画面の行は取り込み時と本文の組み立て時の 2 回 `usageLimitLine` に通る(空白類は空白、その他の制御文字は除去、不正な UTF-8 は U+FFFD、300 rune で切る)。座席と driver は `ValidateIdentifier`、`PrintableInboxText` で見出しに改行が入らない | `usage_limit_screen.go:81-96`、`watch_usage_limit.go:296-421` |
| pane の読みの限定 | 問題なし。`stalled &&` の短絡で、stall の判定が立たない座席は読まない。1 回の読みは `callWithTimeout`(ドライバ呼び出しの上限)の中で、`resumeReadLines`(60 行)だけ | `watch.go:1317`、`watch_usage_limit.go:98-116` |
| 他の条件への影響 | なし。`sendAlert` は戻り値を足しただけ(`raiseOrClear` は捨てる)。`checkWatchEscalationValues`、`watchEscalationBody` の新しい分岐はどちらも `isUsageLimitedAlert` で入る | `watch.go:1382`、`:2058`、`:2136` |
| `usage_limit_screen.go`(印と Stopped の表、resume の挙動) | 問題なし。`ClassifyResumeScreen` は読み取りのみ。`matchedStopSentence` は `findResumeStop` に 1 文ずつの表を渡して再現していて、`findResumeStop` が `prompt=false` の塊だけを返すので、`resumeBlock{text: block}` で結果が食い違わない。印と Stopped の突き合わせはテストで固定されている | `usage_limit_screen.go:42-71`、`resume_screens.go:620-640`、`usage_limit_screen_test.go:314` |

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW(直った: e25def99) | readability | L1. `seatStoppedOnUsageLimit` は bool を返す問い合わせの名前だが、`cycle.usageLimited` へ追記する副作用がある。呼び出し側は `stalled = false` しか見えず、座席が org の集合に入ることは上のコメントと関数の doc を読まないと分からない | `watch_usage_limit.go:98-116`(`cycle.usageLimited = append(...)`)、`watch.go:1317-1319` | `(watchUsageLimitedSeat, bool)` を返して呼び出し側で追記する。または `collectUsageLimitedSeat` のように動詞の名前にする |
| LOW(直った: e25def99) | maintainability | L2. `watchPaneReader` とその型アサートは要らない。`HerdrClient` が `PaneRead` を持つので、nil でない `Org.Herdr` では常に成功する(doc 自身もそう書いている)。兄弟の呼び出しは `o.Herdr.PaneRead` を直接呼んでいる。`watchHerdrProbe` が要るのは `AgentGet` が `HerdrClient` にないからで、同じ理由は `PaneRead` にない | `watch_usage_limit.go:73-80`、`spawn.go:133`、`status_live.go:115`、`send_fallback.go:222` | `w.org.Herdr == nil` の確認だけにして interface を消す。計画 D4 が「型アサート」と書いているので、直すなら D4 の文も合わせる |
| LOW(取り下げ: 私の誤り。`watch_test.go:3967` に使い手がある。e25def99 でコメントが理由を書いた) | maintainability | L3. コメント「it is a pointer so that two pending alerts can still be compared with ==」の使い手が木の中に見つからない。`watchPendingAlert` を `==` で比べる箇所はなく(`*a != *b` の形と、`reflect.DeepEqual(after, before)` の `watch_test.go:1348` は slice があっても動く)、ポインタの `==` は中身ではなくアドレスの比較なので、座席が同じ 2 つの alert でも別のポインタなら等しくならない。理由が空のまま、包みの構造体、ポインタ、`usageLimitedSeats()` の nil 確認が残っている | `watch.go:259-261`、`:270`、`watch_usage_limit.go:57-71` | `UsageLimited []watchUsageLimitedSeat` を直接のフィールドにして包みを消す。`==` で比べる場所が実際にあるなら、その場所をコメントに書く(grep では見つからなかった。見落としならこの指摘は取り下げる) |
| LOW(直った: e25def99。届く先が限られる点は N1) | exception-handling | L4. `PaneRead` の失敗は理由を残さず捨てる(doc も「nothing written」)。pane id が古い、herdr の server が違う、などで読みが毎回失敗すると、watch は今までどおり stall を上げ続け、#312 の症状と見分けがつかず、読めていないことが stderr にも status にも出ない。`agent get` の失敗は liveness の条件として上がる(`watch.go:1284-1289`)のと非対称 | `watch_usage_limit.go:104-110` | 失敗を座席ごとに 1 行 stderr へ。同じ座席の同じ失敗を毎周期出さないよう、直前の周期と同じなら黙る |
| LOW(未解決: 計画どおり。cycle 2 の時点で `docs/tech-debt/README.md` に行がある) | maintainability | L5. 集合にいた座席の読みが 1 回失敗(deadline 切れを含む)すると、その座席は stall になり、集合から外れ、次に読めた周期で「新しい座席」として recurred か新しい件になる。leader が上限で動けない org では、その間に出た stall の ALERT が deadman のあとに stall の件にもなる。計画の AC4 と AC6 の読みどおりだが、herdr が 1 回遅れただけで同じ事故の記録が増える | `watch_usage_limit.go:152-164`、`watch.go:1317-1325` | 集合にいた座席の読みの失敗は、その周期だけ集合に残して stall を立てない。計画の挙動として残すなら、PR の Known gaps に書く |
| LOW(直った: e25def99) | exception-handling | L6. inbox に載せられない道(`escalateWithoutInbox`)では、usage_limited の banner は `subject=leader reason=usage_limit`、`escalations.jsonl` は `subject: leader`、デスクトップ通知は `(usage_limit)` だけで、止まった座席が出ない。計画のリスク「SEAT: leader を leader 自身の問題と読み違える」の対策は件の本文だけで、この道には効かない。この道に入るのは inbox が読めない・書けない、または `checkWatchEscalationValues` が落ちたときに限る | `watch.go:2119-2122`、`:2156-2168`、`:2058-2060` | usage_limited のときだけ banner に `seats=<PrintableInboxText を通した座席の並び>` を足す。`checkUsageLimitedSeats` が落ちた理由でこの道に来ることもあるので、escape が要る |
| LOW(直った: e25def99) | naming | L7. `usageLimitedSeats()`(メソッド)と `UsageLimitedSeats`(status のフィールド)は大文字小文字しか違わず、中身も違う(座席の構造体の列と、id の列)。`watchUsageLimitedSeat`、`watchUsageLimitedSeats`、`usageLimitedSeatIDs`、`usageLimitedSeatList` も近く、grep の結果が混ざる(architecture.md の「grep-able な名前」) | `watch_usage_limit.go:66`、`watch.go:329`、`watch_usage_limit.go:51-62`、`:381-398` | status のフィールドを `UsageLimitedSeatIDs` にする(json タグは `usage_limited_seats` のまま)。L3 の直しで `watchUsageLimitedSeats` が消えれば、紛らわしい名前は 1 つ減る |
| LOW(未解決。e25def99 で出た。cycle 2 の時点で `docs/tech-debt/README.md` に行がある) | exception-handling | N1. L4 の直しは「読めなかった理由」を stall の ALERT のメッセージの最後の 1 行にしたが、届く先は leader への agmsg と `OnSemanticTrigger` だけ。stall が deadman で件になるときの本文(`watchEscalationBody`)は固定の文で、メッセージも pending も持たないので、director と人は読めなかったことを知れない。watch 自身の stderr にも出ない。理由が出るのは立ち上がりの周期の読みだけで、stall が先に立ち、あとから読みが失敗し続ける座席には付かない。L4 の場面(読みが毎回失敗し、leader が止まっている)では、メッセージを読む leader が動けない | `watch_usage_limit.go:131-137`、`watch.go:1323-1339`、`watch.go:2147-2157`(`watchEscalationBody` の stall の分岐は `why` が固定) | 出口を 1 つ足す。watch の stderr に座席ごとに 1 行(前の周期と同じ理由なら黙る)、または理由を pending に持たせて stall の件の本文に足す。足さないなら、PR の Known gaps に「読めなかった理由は leader への ALERT にだけ載る」と書く |
| LOW(直った: c8681f0d) | maintainability | N2. 計画 D4 の新しい文が「self-review の L2 で外した」「(self-review の L4)」「(L6)」と、このレポートの指摘 ID を指している。レポートは保存期間のあとで掃除される(`docs/tech-debt/README.md` に GC 済みの行がある)が、計画は archive に残るので、ID の指す先がなくなる。D4 の事実の文(型アサートをしない、読めなかった理由を stall の ALERT に足す、inbox なしの道の banner に座席を出す)はコードと合っている | `docs/plans/active/2026-11-13-watch-usage-limited.md` D4(`cc9d802b`) | ID を外して理由だけ書く(「`HerdrClient` が `PaneRead` を持つので」「読めなかったとき、確かめられなかったことを伝えるため」)。Progress の行は commit を併記しているので ID があってもよい |
| LOW(未解決。cycle 2 で出た) | maintainability | N3. 座席が増えたときの「件が acked なら記録しない」を決める場所が 2 つになった。読んだ時点の `grown && item.State != InboxStateOpen` と、ロックの中の `AppendRecurredIfOpen` の `ErrInboxAcked`。あとの方は前の方が断るものを全部断る(acked から open へ戻る出来事はない)ので、前の方を消しても結果は変わらない。読んだ時点で acked の件は、`beforeAppendRecurred()` と `AppendRecurredIfOpen` を通って `ErrInboxAcked` になり、stderr なしの `false` になる。これは前の方の早い `return false` と同じ結果である。表の「e1 acked」の行も、競合のテスト(`acks != 1` は件が読んだ時点で open だったことを確かめる)も、前の方が無くても通るので、前の方を消して落ちるテストはない。コメントは前の方を「fast path」と呼ぶが、通るのは立ち上がりの周期だけで、省けるのはロック 1 回と追記の試し 1 回であり、読み手が 2 か所を突き合わせる費用の方が大きい。同じ直しで、cycle 1 の sync-docs が足した 2 行(`docs/tech-debt/README.md` の Test gaps の行(c)「`AppendRecurred` が `ErrInboxResolved` 以外で失敗する枝」と(d)「M2 と M3 の guard は『e1 acked』の 1 行」)と test report の M2(「e1 acked で死んだ」)が、この直しの前の形のまま残る。M2 は今は等価な変異になり、(c) の枝は `AppendRecurred` と `AppendRecurredIfOpen` の 2 つになり、`ErrInboxAcked` 以外の失敗もそこを通る | `watch_usage_limit.go:259`(`if !ok \|\| (grown && item.State != InboxStateOpen)`)、`:263-265`(`recur` の差し替え)、`inbox.go:434`(`case openOnly && item.State == InboxStateAcked`)、`watch_usage_limit.go` の `recordUsageLimitRecurrence` の doc(「The state of the read is a fast path」)、`docs/tech-debt/README.md` の Test gaps の行の (c) と (d)、`docs/reports/test-2026-11-13-watch-usage-limited.md` の M2 の行 | 前の方の条件(`\|\| (grown && item.State != InboxStateOpen)`)を消して `if !ok {` にし、doc の「fast path」の文を「acked かどうかは `AppendRecurredIfOpen` がロックの中で決める」に直す。残すなら、doc に「前の方を消しても結果は変わらない」と書き、Test gaps の行の (d) に M2 が等価な変異になったことを足す。どちらを選んでも (c) の `AppendRecurred` を「`AppendRecurred` と `AppendRecurredIfOpen`」にする |
| LOW(未解決。cycle 2 で出た) | maintainability | N4. cycle 2 の直しが足した 2 か所が、`/cross-review` が同じ名前のまま書き直す報告の中の番号を指す。tech-debt の行は `docs/reports/cross-review-triage-watch-usage-limited.md`(WORTH_CONSIDERING 2)、競合のテストのコメントは「cross-review cycle 1 of feat/watch-usage-limited, finding 1」。triage の名前に cycle は入らず(見出しの `Cycle: 1/2` で区別する)、cross-review の cycle 2 がこの報告を書き直すか足すと、WORTH_CONSIDERING の 2 番と finding 1 が同じ指摘を指す保証はない(私は step の書き方までは読んでいない。このリポジトリの前の PR で同じ形が起きたことは記録にある)。2 か所とも文の中に指摘の中身を全部書いているので、番号が外れても読めるが、番号は確かめられない出所を足している | `docs/tech-debt/README.md` の `recurred` の note の行(「WORTH_CONSIDERING 2」)、`watch_usage_limit_race_test.go:9`、`docs/reports/cross-review-triage-watch-usage-limited.md` の見出し(`Cycle: 1/2`) | 行の「Related plan/report」の欄を、番号なしの「cross-review cycle 1 の triage(WORTH_CONSIDERING の 1 件)」にする。コメントは「cross-review cycle 1 of feat/watch-usage-limited」だけにして、中身の説明(すでにある)に任せる。cross-review の cycle 2 のあとに番号が合っていることを見るなら、それでもよい |

## Update after e25def99

`e25def99`(`watch.go`、`watch_usage_limit.go`、`watch_usage_limit_test.go`)と `cc9d802b`(計画 D4)を読んだ結果。コードを読んで追ったもので、テストは走らせていない。

| ID | 結果 | 確認 |
| --- | --- | --- |
| L1 | 直った | `readUsageLimitScreen` は `(watchUsageLimitedSeat, bool, error)` を返し、何も変えない(`watch_usage_limit.go:101`)。追記と `stalled=false` は `evaluateSeat`(`watch.go:1325-1330` 付近)に出た。`seatStoppedOnUsageLimit` への参照は木に残っていない(grep) |
| L2 | 直った | `watchPaneReader` を消し、`w.org.Herdr == nil` の確認のあと `w.org.Herdr.PaneRead` を直接呼ぶ。`context` と `errors` の import は残りの使い道がある。`Herdr` が nil の Org は読まず、エラーもなく、stall のまま。計画 D4 の文も同じ内容に直っている |
| L3 | 取り下げ | 私の誤り。最初の grep の形(`*got != want` など)が `*p != want` を拾わず、`watch_test.go:3967`(`TestWatch_Recurrence_APendingAlertTheInboxDoesNotHoldIsKept`、`want := *status.PendingAlerts[alertID]` のあと `*p != want`)の使い手を見落とした。構造体にスライスを持たせるとコンパイルが通らないので、ポインタの構造体には理由がある。`want` は自分の写しで、ポインタのアドレスが同じなので、この比べ方は成り立つ。`e25def99` のコメントはそのテスト名と理由を書いていて、実際のテストと合っている |
| L4 | 直った(届く先は N1) | `usageLimitUnreadNote` は読めなかった理由を 1 行にし(制御文字は除去、200 rune で切る)、stall の ALERT の最後に足す。読みの失敗、期限切れ、pane id なしの 3 つで出て、読めた画面と `Herdr` なしでは付かない。テストの表が行ごとに最後の行を固定し、長い誤りと制御文字の行も持つ。`protocol.ValidateText` も通している。`heartbeat stalled for over` の文を見るテストは、新しい `assertStallNote` の 2 行だけ(grep)なので、足した行で既存のテストが落ちる形は見つからなかった |
| L5 | 未解決 | 計画どおり。Known gaps に書く(計画の Progress に、そうすると書いてある) |
| L6 | 直った | `escalateWithoutInbox` は `pending` を受け、banner に usage_limited のときだけ ` seats=a,b` を足す(`usageLimitedBannerSeats`、`watch.go:2170-2172`)。座席は `PrintableInboxText` を通る。座席なし、不正な座席の pending も扱い(空、エスケープ)、テストが 3 行の banner を固定する。呼び出し元は `escalateAlert` の 1 か所のみで、旧シグネチャは残っていない。`escalations.jsonl` と desktop 通知には座席が出ないが、指摘が求めたのは banner だった |
| L7 | 直った | メソッドを `limitedSeatEntries()` に改名。status のフィールド `UsageLimitedSeats` とは別の語になり、大文字小文字だけの違いは消えた |
| D4(`cc9d802b`) | 事実は合う | 型アサートをやめたこと、`Herdr` なしなら読まず stall のまま、読めなかったとき stall の ALERT に 1 行、inbox なしの道の banner に座席、はコードと一致する。残る点は N2 |

新しい指摘は N1 と N2(どちらも LOW)。`e25def99` の diff に CRITICAL、HIGH、MEDIUM はない。

確認して指摘にしなかった点(`e25def99`):

- `usageLimitUnreadNote` に入るエラーの文は herdr の CLI 由来で、leader が読む agmsg に載る。1 行に直され、制御文字は除去され、200 rune で切られるので、新たな経路は増えていない(`watch_usage_limit.go:131-137`)
- `evaluateSeat` は stall の読みを `if stalled { ... }` の中で 1 回に保ち、読みの回数は前と同じ。`unread` は stall のメッセージにだけ使われ、leader の待ちの除外で stall が消えるときは捨てられる
- `escalateWithoutInbox` の引数が `subject` から `pending` に変わったことで、`escalations.jsonl` の `Subject` は `pending.Subject` で、前と同じ値

## Update for cycle 2

cycle 1 の cross-review の ACTION_REQUIRED(`docs/reports/cross-review-triage-watch-usage-limited.md`: ACTION_REQUIRED 1、WORTH_CONSIDERING 1)の直し、`34e5bd86`(`inbox.go`、`watch_usage_limit.go`、`inbox_test.go`、新しい `watch_usage_limit_race_test.go`、tech-debt の 1 行)と `193b044a`(`watch_recurrence.go` の `beforeAppendRecurred` のコメント)を読んだ。`e25def99` から HEAD までに変わった Go のファイルはこの 5 つだけ(`git diff e25def99 HEAD --stat -- '*.go'`)。ブランチ全体は `git diff origin/main...HEAD --stat`(37 ファイル)で見て、cycle 1 で読んだ Go の本体は変わっていない。コードを読んで追ったもので、テストは走らせていない。

### cycle 1 の指摘の状態

| ID | 状態 | 確認 |
| --- | --- | --- |
| L1、L2、L4、L6、L7 | 直った(変わらず) | `e25def99` から Go の本体(`watch.go`、`usage_limit_screen.go`)は動いていない |
| L3 | 取り下げ(変わらず) | |
| L5 | 未解決、計画どおり。tracked | cycle 1 の sync-docs が `docs/tech-debt/README.md` に行を足した(「A seat in the set of the org condition `usage_limited` whose screen read fails once …」、`self-review L5` と名前を書いている)。行の中のコードの名前(`readUsageLimitScreen`、`raiseUsageLimited`、`recordUsageLimitRecurrence`、`evaluateSeat`)は木にある。`34e5bd86` は `recordUsageLimitRecurrence` の中を変えたが、行の主張(座席が 1 回の読みの失敗で集合を出る)には触れていない |
| N1 | 未解決。tracked | 同じく行がある(「The reason a screen could not be read (`usageLimitUnreadNote`) reaches only the stall ALERT …」、`self-review N1`) |
| N2 | 直った | `c8681f0d`。計画 D4 の文(`- D4 画面を読むのは…`)に L2、L4、L6 の ID は残っていない。ID が残るのは Progress の行(commit を併記)だけ |

### 直しの確認(`34e5bd86`、`193b044a`)

| 点 | 結果 | 根拠 |
| --- | --- | --- |
| ロックの中で open を確かめる | 意図どおり。`appendRecurred(…, openOnly)` は `update` の decide の中で、resolved(`ErrInboxResolved`)、`openOnly` かつ acked(`ErrInboxAcked`)、ALERT でない型、の順に断る。acked の確かめは resolved のあと、型の確かめの前にある。acked で型が ALERT でない件も `ErrInboxAcked` になるが、呼び出し元は ALERT の件しか渡さない。断った append は何も書かない(テストが `inbox.jsonl` の写しで確かめる)。`AppendRecurred` は `openOnly=false` で、今までと同じ | `inbox.go:402-441` |
| 断られた道 | 閉じている。`errors.Is(err, ErrInboxResolved)` と同じ扱いで、stderr なしの `false` → `sendAlert` が全座席(`limited`)を持つ新しい pending を作り、同じ周期の deadman の掃きで件になる。`escalateToInbox` は `!isUsageLimitedAlert(alertID)` で acked の e1 への畳み込みを閉じているので、必ず新しい件。競合のテストが e1 は acked のまま recurred なし、e2 が 3 座席、ALERT は 2 つ、と固定している | `watch_usage_limit.go:259-273`、`watch.go:2020`(`escalateToInbox` の `!isUsageLimitedAlert`) |
| stderr の静けさ | 新しい穴ではない。読んだ時点で acked(今までの道)も、append のあいだに resolve(`ErrInboxResolved`)も、今まで stderr なしで新しい件になる。新しい件は `escalateToInbox` が 1 行を出し(件の ID と director 向けの文)、競合のテストは「recorded as a recurrence」と「not recorded」の 2 行が無いことを見ている。acked で断られたことは書かれないが、同じ周期の e2 の本文と `SEATS` に増えた座席が出る | `watch_usage_limit.go:269`、競合のテスト 53-57 行 |
| 新しい件ができたあとの次の周期 | 追加の記録は出ない。`status.UsageLimitedSeats` は append の前に 3 座席に置き換わっていて(`raiseUsageLimited` の `status.UsageLimitedSeats = usageLimitedSeatIDs(limited)`)、次の周期は `wasActive` かつ `added` が空で dedupe の `return`。競合のテストの最後の周期がこれを固定している(e1 と e2 に recurred なし、ALERT は 2 つのまま) | `watch_usage_limit.go:199-202`、競合のテスト 59-62 行 |
| `AppendRecurred` の他の呼び出し元に open だけの形が要るか | 要らない。5 か所。`recordRecurrence`(`watch_recurrence.go:190`)と `escalateAsRecurrence`(`:267`)は他の条件で、open か acked に recurred を足す規則は `agent-messaging.md` が定めている。`escalateAsRecurrence` へは usage_limited は来ない(`escalateToInbox` の `!isUsageLimitedAlert`)。`escalateToInbox` が新しい件を作ったあとの `AppendRecurred(id, alertID, "", pending.Recurrences)`(`watch.go:2032`)は note なしの数だけで、件の本文の `SEATS` と行に増えた座席が入っている(`recordUsageLimitRecurrence` が pending の座席を先に置き換える)ので、件がその隙間で acked になっても座席の情報は失われない。`recordUsageLimitRecurrence` の条件が下りたあとの立ち上がり(`grown` が false)は計画の AC5 と `agent-messaging.md` の規則(open か acked)で、`AppendRecurred` のまま | grep の結果(非テストの呼び出し元はこの 5 つ) |
| コメントと名前 | `ErrInboxUnknownID`、`ErrInboxResolved` の doc に新しいメソッドが入っている。`recordUsageLimitRecurrence` の doc(「The state of the read is a fast path …」「or, for a grown condition, acked meanwhile (ErrInboxAcked) with no line」)はコードと合う。`193b044a` のコメント(`beforeAppendRecurred` は acked にも効く、`recordUsageLimitRecurrence` が `AppendRecurredIfOpen` で足す)も合い、`escalateToInbox` のコメントの「an acked item, say」も今の挙動と食い違わない。`appendRecurred(…, openOnly bool)` の bool は非公開で呼び出しは 2 か所なので許せる。`usageLimitRecurNote` の「so AppendRecurred takes it」と `InboxNoteMaxRunes` の「the longest recurred note AppendRecurred accepts」は、`AppendRecurredIfOpen` も同じ note の確かめを共有するので偽にならない | |
| テストが直しの前に落ちるか | 競合のテストは落ちる。`acks != 1` で seam が 1 回走ったこと(件が読んだ時点で open だったこと、つまり断ったのが ロックの中の確かめであること)を見て、直しの前の `AppendRecurred` なら e1(acked)に recurred が付くので `len(recurredEvents(e1)) != 0` と `len(items) != 2` で落ちる。seam が走らない空の通過ではない。`inbox_test.go` の 3 つ: open で正確な 1 行を足す、acked と resolved は正確な文と `errors.Is` の両向きで断って `inbox.jsonl` が変わらない、他の 5 つの断り(alert_id 空、count 0、改行入りの note、QUESTION の件、未知の ID)は 3 つの sentinel のうち包むのが期待した 1 つだけ(未知の ID は `ErrInboxUnknownID`、他は無し)であることを両向きで見る、ロックを持つ別の holder が ack するあいだ待って `ErrInboxAcked` になる(150 ms の待ちは `TestInboxAppendRecurred_DecidesUnderTheInboxLock` と同じ書き方) | `watch_usage_limit_race_test.go`、`inbox_test.go:1369-1505`(`TestInboxAppendRecurredIfOpen_*` の 3 つ) |
| tech-debt の追加行 | 列は 5 つで、`\|` を含まない。行の主張は木と合う: 500 rune の上限(`InboxNoteMaxRunes`)、`fitUsageLimitedSeats` の最後の `cutUsageLimitText(usageLimitedMore(seats), budget)`(「+N more」の並びまで切る)、`max_seats` の既定 5(`config.go`、`templates/base/ralph.toml`)、`SEATS` の見出しは全座席、watch の stderr の行は `usageLimitedSeatList(added)` で全 id、状態ファイルの `usage_limited_seats`。座席の id は最長 30 文字(`identifierPattern`)で、note の予算は 500 から前置きの 26 を引いた 474 rune なので、「+N more」の並びまで切れるのは id が最長のとき 15 座席前後から(私の計算。走らせていない)。指摘は N4 の点だけ | `docs/tech-debt/README.md` の最後の行 |

確認して指摘にしなかった点(cycle 2)。

- recurred を足した直後、director が recurred を読む前の ack で件が acked になる窓は残る。ack は読んだ版を持たないので、この修正では閉じられず、cycle 1 の指摘が言った窓(append の前の ack)とは別。受け入れ条件の「open の件に足す」は、足した時点で open を指す
- `recur := w.org.Inbox.AppendRecurred` を `grown` で差し替える書き方は、メソッド値で型が同じ。分岐は 1 か所で、読める
- `watch_usage_limit_race_test.go` の名前は、このリポジトリで唯一の `_race_test.go` で、Go の `-race` を思わせる。中身は「append の前の ack」の 1 つのテストで、表のテストの 4 行目(「e1 resolved between the read and the append」)の ack 版にあたる。`watch_usage_limit_test.go` が 936 ある(tech-debt の行が分割を決めている)ので別の file にした理由は分かる。次にこの file を触るときの分割に任せる

cycle 2 の新しい指摘は N3 と N4(どちらも LOW)。`34e5bd86` と `193b044a` に CRITICAL、HIGH、MEDIUM はない。

## Positive notes

- (`c1ea2017` 時点。`e25def99` で `evaluateSeat` の差し込みは約 10 行に増えた)新しい条件の部品が `watch_usage_limit.go` に集まり、`watch.go` への差し込みは、定数、構造体のフィールド 2 つ、`evaluateSeat` の 3 行、`evaluateCycle` の 1 行、`sendAlert` の戻り値、escalate 系の分岐 4 つ(`escalateToInbox`、`checkWatchEscalationValues`、`watchEscalationBody`、`watchEscalationReason`)に収まっている。他の条件の分岐は `isUsageLimitedAlert` で切れていて、`watchEscalationReason` の他の結果は前と同じ
- 画面由来の文は 2 段で縛られている。取り込み時に 1 行 300 rune で切って制御文字を落とし、本文を組み立てる時にもう一度通す。status ファイルを手で書き換えても、見出しに改行は入らない(`checkUsageLimitedSeats`)
- 本文と recurred の note の切り詰めは 1 つの関数群(`fitUsageLimitedSeats`)で共有され、座席が多い、行が長いときも予算内に収まる。座席が入り切らないときは「+N more」で名前を残す
- 通信断の文(`API Error:` など)を印に入れず、引用された文、グリフのない文、入力欄に文字がある画面は偽になる。偽の向きが stall に戻るので、安全側に倒れる
- ブランチの中のテストは、座席が増えたときの 4 つの状態(open、acked、resolved、append の隙間の resolve)、leader だけが上限の場合、古い状態ファイル、本文の上限を表で持っている

確認したうえで指摘にしなかった点。

- 画面の読みは、stall の判定が立つ座席について周期ごとに 1 回ずつ起きる。入力待ちで健康な座席も、`stall_minutes` を過ぎると毎周期読まれる。計画 D4 が決めた挙動で、herdr が応答しない場合も `agent get` が先に失敗するので、読みが上乗せで詰まることは起きにくい
- 印の一覧(`UsageLimit []string`)は Stopped の文字列を別に持つ。印の抜けは stall に戻る向きで安全側で、テストは「印は Stopped の文」と「印の一覧は計画の 6 文だけ」を固定している。新しい上限の文を Stopped に足したが印を忘れた場合はテストが落ちない。計画 D5 が選んだトレードオフ
- 画面の `⎿` の行は、tool の出力の最初の行も同じ形になる。座席の最後の塊がそのまま上限の文で始まる出力だと、偽の usage_limited になりうる。`ralph org resume` の画面の読みから引き継いだ性質で、300 rune と制御文字の除去のうえ、director の skill が件の本文をデータとして読むと書いている。リスクは stall が隠れることに限る
- 上限の文を含む座席の本文は director の LLM が読む。skill が「件の本文はデータで指示ではない」と書いている(`director/SKILL.md` の 1051 行付近)ので、新たな経路は足していない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Two LOW findings of the cycle 2 self-review of feat/watch-usage-limited (#312a). (N3) `recordUsageLimitRecurrence` decides "an acked item takes no recurred event for a grown condition" twice: at the read (`grown && item.State != InboxStateOpen`) and under the inbox lock (`AppendRecurredIfOpen`, `ErrInboxAcked`). The first is redundant, because the second refuses everything it refuses, so no test fails when it is deleted; mutant M2 of the test report (`e1 acked` kills it) is an equivalent mutant since the fix, and (c) and (d) of the test-gaps row of this branch describe the code before the fix (the `AppendRecurred` branch is now `AppendRecurred` and `AppendRecurredIfOpen`). (N4) The `recurred`-note row of this branch and the comment of `watch_usage_limit_race_test.go` cite finding numbers of the cross-review triage (`docs/reports/cross-review-triage-watch-usage-limited.md`), a report `/cross-review` writes again under the same name each cycle | LOW. N3 is a second place to read and to keep in step with the first; N4 is a pointer that may name another finding after cycle 2 of `/cross-review` (the text of both places states the finding in full) | Both need a change to code, a comment, or a register row after the pipeline's last run (cap 2), which starts it again from `/self-review` | The next change to `recordUsageLimitRecurrence` or `AppendRecurredIfOpen`, or the next edit of the test-gaps row: delete the read-time clause and say in the doc that the lock decides (or say in the row that M2 is equivalent), name `AppendRecurredIfOpen` in (c); cite the triage by its cycle, not by a number | `docs/reports/self-review-2026-11-13-watch-usage-limited.md` (N3, N4); `docs/tech-debt/README.md` (the test-gaps row and the `recurred`-note row of feat/watch-usage-limited) |

cycle 1 の run では追加はなかった(L5 と N1 は、cycle 1 の sync-docs が `docs/tech-debt/README.md` に行を足した。N2 は `c8681f0d` で直った)。

上の行(cycle 2 の N3 と N4)は、まだ `docs/tech-debt/README.md` に書いていない。この更新の commit は報告だけにする依頼なので、足していない。今回の run は triage の見出し(`Cycle: 1/2`)から上限の 2 回目で、N3 と N4 の直しはコードかコメントを動かすので、新しい run を始め、cap を上げない限りできない。直さずに進めるなら 2 件は deferral で、上の行を `docs/tech-debt/README.md` に足す(orchestrator か `/sync-docs`)。直すなら、N3 は `|| (grown && item.State != InboxStateOpen)` の削除と doc の文、N4 は行の欄とコメントの番号の書き換えで、行は要らない。

_(If any rows were added above, also append them to `docs/tech-debt/`. The row above is not appended yet; see the paragraph before this line.)_

## Recommendation

- Merge: yes。cycle 2、`193b044a` まで。CRITICAL、HIGH、MEDIUM はない。数は、未解決の LOW が 4 件(L5、N1、N3、N4。L5 と N1 は `docs/tech-debt/README.md` に行がある)、直ったもの 6 件(L1、L2、L4、L6、L7、N2)、取り下げ 1 件(L3)。cycle 1 の cross-review の ACTION_REQUIRED(座席が増えたときの open の確かめがロックの外にあった)は `34e5bd86` で直っている。断られた道(新しい件と 2 つ目の ALERT)、stderr の静けさ、新しい件ができたあとの次の周期、`AppendRecurred` の他の 4 つの呼び出し元(open だけの形は要らない)を追って、直しが開く新しい窓は見つからなかった(コードを読んで追った。テストは走らせていない)。N3 と N4 はどちらも LOW で、マージを止める理由ではない
- Cycle 1 の記録(`cc9d802b` まで、同じ報告の更新): 最初の run の LOW 7 件のうち 6 件が片づき、直しで N1 と N2 が出た。依頼で挙げられた点(状態ファイルの互換、stall と待ちの除外の順序、集合の増加と `ErrInboxResolved`、acked の件に畳まないこと、理由の選び方、本文の上限と制御文字、pane の読みの限定、他の条件への影響、印と Stopped の同期、resume の挙動)は最初の run で追って問題がなく、`e25def99` 以降 `watch.go` と `usage_limit_screen.go` は動いていない
- Follow-ups:
  - N3 と N4 は、直さずに進めるなら「Tech debt identified」の 1 行を `docs/tech-debt/README.md` に足す。この run は triage の見出しから上限の 2 回目で、直しは新しい run になる。L5 と N1 は行がある。PR の Known gaps には L5、N1、N3、N4 と、triage の WORTH_CONSIDERING 2(recurred の note の切り詰め)を並べる
  - N1 の直し方(watch の stderr に座席ごとに 1 行、または pending に理由を持たせて件の本文に足す)は、tech-debt の行にある
  - 最初の run で L3 を誤って挙げた(使い手を grep で見落とした)。使い手は `watch_test.go:3967` を読んで確かめた。ほかに `watchPendingAlert` を比べる箇所があるかは探し直していない
  - `/verify` と `/sync-docs` へ。計画の `## Affected areas` に `internal/org/inbox.go` と `inbox_test.go` が無く(`34e5bd86` が足した)、AC4 の文は「recurred を足すあいだに件が resolve されたとき」だけで、ack のときを書いていない。cycle 1 の test report の M2(「e1 acked」の行で死んだ)は直しの前の形で、`docs/tech-debt/README.md` の Test gaps の行の (c) と (d) も同じ(N3)。ここでは diff の質だけを見たので、spec と文書の合わせは判断していない
  - 実行結果、コンパイル、gofmt の整列、skill 4 つの写しの一致、D4 の承認 digest(`0746c70f0d2b`)が計画の本文と合っているかは `/verify` と `/test`。この更新では確かめていない
  - insight event: cycle 1 の自己レビューの event(`docs/insights/events/2026-11-12-watch-usage-limited.jsonl`)は最初の run の数(LOW 7)のまま。cycle 2 の event を `2026-11-13-watch-usage-limited.jsonl` に追記した(LOW 4、`--cycle auto` で 2)。追記した行は、この報告の commit には入れていない
