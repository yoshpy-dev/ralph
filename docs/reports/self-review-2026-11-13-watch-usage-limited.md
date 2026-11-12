# Self-review report: watch-usage-limited

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-watch-usage-limited.md`(issue #312 の受け入れ条件 1)
- Branch: `feat/watch-usage-limited`、base は `origin/main` の `4220ea7c`(ブランチは `78611dbc` で取り込み済み)。レビューした HEAD は `c1ea2017`
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの 1 回目(`cycle-count.json` は cycle 1)
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
| LOW | readability | L1. `seatStoppedOnUsageLimit` は bool を返す問い合わせの名前だが、`cycle.usageLimited` へ追記する副作用がある。呼び出し側は `stalled = false` しか見えず、座席が org の集合に入ることは上のコメントと関数の doc を読まないと分からない | `watch_usage_limit.go:98-116`(`cycle.usageLimited = append(...)`)、`watch.go:1317-1319` | `(watchUsageLimitedSeat, bool)` を返して呼び出し側で追記する。または `collectUsageLimitedSeat` のように動詞の名前にする |
| LOW | maintainability | L2. `watchPaneReader` とその型アサートは要らない。`HerdrClient` が `PaneRead` を持つので、nil でない `Org.Herdr` では常に成功する(doc 自身もそう書いている)。兄弟の呼び出しは `o.Herdr.PaneRead` を直接呼んでいる。`watchHerdrProbe` が要るのは `AgentGet` が `HerdrClient` にないからで、同じ理由は `PaneRead` にない | `watch_usage_limit.go:73-80`、`spawn.go:133`、`status_live.go:115`、`send_fallback.go:222` | `w.org.Herdr == nil` の確認だけにして interface を消す。計画 D4 が「型アサート」と書いているので、直すなら D4 の文も合わせる |
| LOW | maintainability | L3. コメント「it is a pointer so that two pending alerts can still be compared with ==」の使い手が木の中に見つからない。`watchPendingAlert` を `==` で比べる箇所はなく(`*a != *b` の形と、`reflect.DeepEqual(after, before)` の `watch_test.go:1348` は slice があっても動く)、ポインタの `==` は中身ではなくアドレスの比較なので、座席が同じ 2 つの alert でも別のポインタなら等しくならない。理由が空のまま、包みの構造体、ポインタ、`usageLimitedSeats()` の nil 確認が残っている | `watch.go:259-261`、`:270`、`watch_usage_limit.go:57-71` | `UsageLimited []watchUsageLimitedSeat` を直接のフィールドにして包みを消す。`==` で比べる場所が実際にあるなら、その場所をコメントに書く(grep では見つからなかった。見落としならこの指摘は取り下げる) |
| LOW | exception-handling | L4. `PaneRead` の失敗は理由を残さず捨てる(doc も「nothing written」)。pane id が古い、herdr の server が違う、などで読みが毎回失敗すると、watch は今までどおり stall を上げ続け、#312 の症状と見分けがつかず、読めていないことが stderr にも status にも出ない。`agent get` の失敗は liveness の条件として上がる(`watch.go:1284-1289`)のと非対称 | `watch_usage_limit.go:104-110` | 失敗を座席ごとに 1 行 stderr へ。同じ座席の同じ失敗を毎周期出さないよう、直前の周期と同じなら黙る |
| LOW | maintainability | L5. 集合にいた座席の読みが 1 回失敗(deadline 切れを含む)すると、その座席は stall になり、集合から外れ、次に読めた周期で「新しい座席」として recurred か新しい件になる。leader が上限で動けない org では、その間に出た stall の ALERT が deadman のあとに stall の件にもなる。計画の AC4 と AC6 の読みどおりだが、herdr が 1 回遅れただけで同じ事故の記録が増える | `watch_usage_limit.go:152-164`、`watch.go:1317-1325` | 集合にいた座席の読みの失敗は、その周期だけ集合に残して stall を立てない。計画の挙動として残すなら、PR の Known gaps に書く |
| LOW | exception-handling | L6. inbox に載せられない道(`escalateWithoutInbox`)では、usage_limited の banner は `subject=leader reason=usage_limit`、`escalations.jsonl` は `subject: leader`、デスクトップ通知は `(usage_limit)` だけで、止まった座席が出ない。計画のリスク「SEAT: leader を leader 自身の問題と読み違える」の対策は件の本文だけで、この道には効かない。この道に入るのは inbox が読めない・書けない、または `checkWatchEscalationValues` が落ちたときに限る | `watch.go:2119-2122`、`:2156-2168`、`:2058-2060` | usage_limited のときだけ banner に `seats=<PrintableInboxText を通した座席の並び>` を足す。`checkUsageLimitedSeats` が落ちた理由でこの道に来ることもあるので、escape が要る |
| LOW | naming | L7. `usageLimitedSeats()`(メソッド)と `UsageLimitedSeats`(status のフィールド)は大文字小文字しか違わず、中身も違う(座席の構造体の列と、id の列)。`watchUsageLimitedSeat`、`watchUsageLimitedSeats`、`usageLimitedSeatIDs`、`usageLimitedSeatList` も近く、grep の結果が混ざる(architecture.md の「grep-able な名前」) | `watch_usage_limit.go:66`、`watch.go:329`、`watch_usage_limit.go:51-62`、`:381-398` | status のフィールドを `UsageLimitedSeatIDs` にする(json タグは `usage_limited_seats` のまま)。L3 の直しで `watchUsageLimitedSeats` が消えれば、紛らわしい名前は 1 つ減る |

## Positive notes

- 新しい条件の部品が `watch_usage_limit.go` に集まり、`watch.go` への差し込みは、定数、構造体のフィールド 2 つ、`evaluateSeat` の 3 行、`evaluateCycle` の 1 行、`sendAlert` の戻り値、escalate 系の分岐 4 つ(`escalateToInbox`、`checkWatchEscalationValues`、`watchEscalationBody`、`watchEscalationReason`)に収まっている。他の条件の分岐は `isUsageLimitedAlert` で切れていて、`watchEscalationReason` の他の結果は前と同じ
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

追加はない。L1 から L7 はすべて LOW で、直す場合は full re-run になる。直さずに進める場合の扱いは `/sync-docs` か `/pr` の Known gaps で決める(この run では `docs/tech-debt/` に書いていない)。

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。LOW が 7 件(L1 から L7)。依頼で挙げられた点(状態ファイルの互換、stall と待ちの除外の順序、集合の増加と `ErrInboxResolved`、acked の件に畳まないこと、理由の選び方、本文の上限と制御文字、pane の読みの限定、他の条件への影響、印と Stopped の同期、resume の挙動)は、コードを読んで追い、問題は見つからなかった(テストは走らせていない)
- Follow-ups:
  - 直す価値があるのは L4(読めないことが見えない)と L6(inbox なしの道で座席が出ない)。どちらも #312 の症状を観測する場面で効く。L5 は計画の挙動なので、直すか Known gaps に書くかを決める
  - L1、L2、L3、L7 は読みやすさの整理で、まとめて 1 コミットで直せる。L2 を直すなら計画 D4 の「型アサート」の文も合わせる
  - 計画の Progress に S3 の範囲外の所見(`quality-gates.md` の見張りの行、multi-org の仕様の理由の一覧)が残っている。`/sync-docs` で扱う
  - 実行結果、コンパイル、gofmt の整列、skill 4 つの写しの一致は `/verify` と `/test`
