# Self-review report: org-result-missing

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-result-missing.md`(issue #361)
- Reviewer: reviewer サブエージェント(`/self-review`)。cycle 1
- Scope: `git diff origin/main...HEAD`(origin/main = 2a069b50、HEAD = 22d2f0cf、main の取り込みは b0c56556)。30 ファイル、+2732 / -98。中心は `internal/org/watch_result_missing.go`(新規)、`internal/org/watch.go` の `evaluateSeat` と `watchSeatSnapshot`、`internal/org/reseat_progress.go` の `sentDetailsText`、座席の雛形 3 つ、rule、skill、仕様、テスト
- 走らせたもの: HEAD を `git archive` で scratchpad に写し、そこに調べ用のテストを 1 本足して `go test -count=1 ./internal/org/ -run TestProbe_ResultMissing_SecondTaskWhileFirstItemOpen -v` を 2 回(下の M1 の根拠)。worktree には何も書いていない。全体のテスト、静的解析、`run-test.sh` は走らせていない

## Evidence reviewed

- 計画の全文(Progress の S0、S1、計画の直し、S2、S2b、S3、main の取り込み)
- `watch_result_missing.go` の全文と、`watch.go` の `evaluateSeat`、`raiseOrClear`、`sendAlert`、`checkDeadman`、`leaderActedSinceAlert`、`escalateAlert`、`leaderHistoryFromField`、`historyLeaderLineCount`
- `watch_recurrence.go` の `recordRecurrence`、`foldIntoPendingAlert`、`reconcilePendingAlertsWithInbox`
- `verbs.go` の `Send` の終わり(`sent` の TS は Enter と確かめのあと、確かめは既定 3 秒まで)、`callWithTimeout`(既定 10 秒)
- `driver/agmsg.go` の `History`(`AGMSG_SELF_NAME=off` が history.sh にも付く)と `driver.go` の Runner(stderr は出力に混ざらない)
- 手元の agmsg の `scripts/history.sh` と `send.sh`(repo の外。読んだだけ)
- `protocol.Parse`(1 行目が `TYPE:` でなければ失敗)
- テスト: `watch_result_missing_test.go` の全関数、`prompts_seat_send_test.go`、`prompts_leader_result_missing_test.go` の構成、`prompts_verification_test.go` と `prompts_xreview_reviewer_test.go` の差分
- 雛形: `implementer.md:115-156`、`reviewer.md:424-465` と冒頭の書く場所の段落、`leader.md:1664-1689`
- rule と skill: `agent-messaging.md` の 2 つの新しい段落、`/director` と `/org` の skill の差分、`docs/tech-debt/README.md` の追記

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | 同じ座席の前の TASK の `result_missing` の受信箱の件が open か acked のあいだに、次の TASK の `result_missing` が立つと、その件の `recurred` になる。note は空で、leader には ALERT が届かない。新しい TASK_ID はどこにも残らない。条件の鍵は (org, 座席, 条件) だけで TASK を含まず、立ち上がりで Active になるので、件が resolve されたあとも、座席が入力待ちのままなら新しい TASK の ALERT は出ない。leader は届かない RESULT を待ち続けるので、#361 と同じ形が残る。rule の「Its deadman escalation and its re-firings follow the rules above」は既存の決まりに従うとだけ言い、別の TASK の立ち上がりが前の件に吸われることは書いていない。台帳の (a) の行は件に TASK_ID がないことだけを扱う | `internal/org/watch.go:1387`(`raiseOrClear(..., condResultMissing, ..., true, "")`、recurNote が空)、`internal/org/watch_recurrence.go:172-197`(`openAlertItemForKey` は鍵で引く)と `:267`(escalation の側も note は空)、`.claude/rules/ralph/agent-messaging.md:301`。調べ: t1 を上げて deadman で件 e2 にしたあと、t2 を送り、座席を 1 周 working にしてから入力待ちにした。t0+30m で ALERT は t1 の 1 件のまま、e2 に `recurrences=1`、stderr は `condition=result_missing fired again; recorded as a recurrence on inbox item e2`。e2 を resolve してさらに 30 周回しても ALERT は 1 件、条件は active のまま | 少なくとも、`evaluateSeat` が `raiseOrClear` に渡す recurNote に新しい TASK_ID を入れる(たとえば `TASK_ID <id>`)。件の `recurred` に TASK が残り、director が別の TASK だと分かる。できれば、条件を最後に上げた TASK(`resultMissingTaskKey`)を座席の記録に持ち、TASK が変わった立ち上がりは前の件の再発にせず新しい ALERT として leader に送る。台帳の (a)(件に TASK_ID を持たせる)と同じところを触るので、まとめて直すとよい。直さずに進めるなら、rule と `/director` の skill にこの穴を 1 文書き、台帳に行を足す |
| L1 | LOW | readability | ALERT の本文は「without sending the leader a RESULT or BLOCKED for it」と言う。しかし座席がその TASK の BLOCKED を送り、leader が答え、そのあと座席が入力待ちのまま `stall_minutes` を過ぎた道でも条件は立つ。この道では座席は BLOCKED を送っているので、本文は事実と違う。QUESTION の道でも、待ちは TASK からではなく leader の答えから数えているのに、本文は「received TASK %s and has been waiting」としか言わない。leader の雛形(「その TASK の RESULT を送っていない」)と rule(RESULT か REVIEW)はそれぞれ別の言い方をしている | `internal/org/watch_result_missing.go:181-192`(BLOCKED と QUESTION のあと `start` を leader の答えに移す)、`:198-203`(本文)、`internal/org/prompts/leader.md:1664-1667` | 本文を「has sent the leader no RESULT for it since the TASK (or since the leader's last answer to its QUESTION or BLOCKED)」のように、数え始めを含む形にする。`assertResultMissingAlert` の `head` も合わせて直す |
| L2 | LOW | maintainability | history.sh の 1 行を読む関数が 2 つになった。新しい `parseAgmsgHistoryLine` は最初の `[` と次の `] ` で切り、既存の `leaderHistoryFromField` は最初の `] ` で切る。どちらも doc comment で history.sh の形を別々に説明していて、互いを名指していない。`agmsgHistoryRecord`、`parseAgmsgHistory`、`historyEscapes`、`agmsgNoHistory` は agmsg の履歴一般の読み手なのに、条件のファイルに置かれている。history.sh の形が変わったとき、片方だけ直る余地がある | `internal/org/watch_result_missing.go:61-67`、`:336-397`、`internal/org/watch.go:1770-1804` | `leaderHistoryFromField` を `parseAgmsgHistoryLine` の `From` で書き直すか、少なくとも 2 つの doc comment で互いを名指す。履歴の読み手を `agmsg_history.go` のような小さなファイルに移すと、grep で見つけやすい |
| L3 | LOW | null-safety | 履歴の読み方の doc comment が、history.sh の保証より強いことを言う。(a) history.sh は本文の改行とタブを `\n` と `\t` にするが、バックスラッシュはそのまま出す(`replace(replace(body, char(10), '\n'), char(9), '\t')`)。`historyEscapes` は本文に書かれた文字どおりの `\n` も改行に戻すので、`SUMMARY:` のようなヘッダの値に `\n` という文字列があると、そこで行が割れ、それより後のヘッダは本文になる。1 行目の TYPE と 2 行目の TASK_ID には効かない。(b) history.sh は、どれかの欄が NULL の行(SQLite の `\|\|` が全体を NULL にする)を `[ -n "$ts$from$to$body" ] \|\| continue` で飛ばす。そういう記録があると、出力の行数が limit より少ないのに古い行が残っていて、`historyReachesBack` の「limit より少なければ全部読めた」が成り立たない。agmsg がそういう記録を書くかは確かめていない | `internal/org/watch_result_missing.go:65-67`(「which are undone」)、`:310-324`、`:359-369`。手元の agmsg の `scripts/history.sh` の rows の SQL と表示のループ | (a) と (b) を doc comment に書く(元に戻す処理は完全ではない、limit より少ないことは history.sh が飛ばす行がない前提)。(b) を気にするなら、limit より少ない場合も、最も古い行が `since` より前か、行が 1 つもないときだけ遡れたとする案がある。ただし短い履歴で「わからない」が増えるので、doc に書くだけでもよい |
| L4 | LOW | maintainability | 時計が切れたあとは、座席が入力待ちでいるかぎり、毎周(既定 30 秒)履歴を読み直す。答え待ち、条件が立っている、わからない、のどれでも読む。遡れない「わからない」は 50、200、1000 の 3 回を毎周読み、各回に 10 秒の期限がある。TASK のあとの行は増えるだけなので、1000 で遡れなかった TASK は、そのあとも遡れない。計画の S3 の注は「`result_missing` は座席ごとに履歴を最大 3 回読む(1 回 10 秒まで)」を範囲の外の発見として挙げているが、台帳にも rule にもない | `internal/org/watch_result_missing.go:163-169`、`:279-308`、`internal/org/watch_result_missing_test.go:604-633`(1 周だけを見ていて、次の周の読み直しは固定していない)、計画の Progress の S3 | 1000 でも遡れなかった TASK を `resultMissingTaskKey` で座席の記録に覚え、その TASK では読まない(`ResultSeen` と同じ形)。直さないなら台帳に行を足す(毎周の読みの数、期限、どれで止まるか) |
| L5 | LOW | maintainability | 座席の確かめと見張りの判定が、同じ契約を違う緩さで読む。雛形の手順 2 は「`{{SEAT_ID}} → leader:` の最後の行に `TYPE: <TYPE>` と `TASK_ID: <TASK_ID>` がある」ことを確かめる(行のどこかにあればよい)。見張りは `protocol.Parse` で読み、1 行目が `TYPE:` でなければ型のない行として数える。先頭に空行、BOM、コードの囲みがあるファイルを送ると、座席の確かめは通り、見張りは RESULT を数えず、`stall_minutes` 後に届いている RESULT について `result_missing` を上げる。`$(cat <ファイル>)` は先頭の空行を残す | `internal/org/prompts/implementer.md:141-151`、`internal/org/prompts/reviewer.md:450-460`、`internal/org/watch_result_missing.go:326-334`、`internal/org/protocol/protocol.go:100-106` | 手順 2 を「その行の `leader: ` の直後が `TYPE: <TYPE>` で始まり、`TASK_ID: <TASK_ID>` がある」と書き、ファイルの 1 行目を `TYPE:` の行にすることも雛形で言う。`prompts_seat_send_test.go` で固定する |

## Positive notes

- 台帳から読む TASK_ID は、切れたもの、空のもの、制御文字を含むものを外している(`seatLastTask`)。ALERT のヘッダに改行を差し込む道はない。ALERT は `sendAlert` の `ValidateText` も通る
- 履歴の行は、最初の `[`、次の `] `、次の ` → `、次の `:` で切る。どの区切りも本文より前にあるので、本文に `] ` や ` → ` があっても from と to はずれない(`TestResultMissing_ParseAgmsgHistoryLine` の 6 つ目の行が固定している)
- 読めない行、読めない履歴、遡れない履歴、切れた TASK_ID は、どれも上げない側に倒れている。誤った ALERT で leader を止めない計画の方針と合う
- 見張りが座席を名指して history.sh を打っても、`driver.Agmsg.run` が `AGMSG_SELF_NAME=off` を付けるので、agmsg の自分の名付けは走らない
- 時刻は台帳も agmsg も秒で比べている。同じ秒の RESULT は数え、leader の答えは秒で後のものだけを数える。どちらもテストで固定されている
- 見張りの状態ファイルの新しい 2 つの欄は `omitempty` で、古いファイルは `IdleSince` なしで読み込まれ、その周に今の時刻が入る。数え始めが遅れるだけで、誤って上げる側には倒れない
- テストは偽の時計で回し、sleep がない。偽の history.sh は件数の上限、時刻順、`\n` と `\t` の書き方、出力の trim まで本物に合わせてある
- 座席の送り直しは、終了コードが 0 でないときの 1 回だけで、履歴の確かめに失敗したときは送り直さない。送信と確かめが輪になる道はない。leader の頼みのあとも、2 回目の `SEND FAILED:` で BLOCKED に移る

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 追加なし。M1 と L1 から L5 は、直すか残すかの判断を実装の側に返す。残すものがあれば、M1 と L4 は台帳に行が要る(M1 は台帳の「Known gaps of fix/org-result-missing」の行の (c) として足すのがよい)。L2、L3、L5 は doc comment と雛形の文の直しで済む | | | | この report、`docs/plans/active/2026-11-22-org-result-missing.md`(Progress の S3) |

_(行は足していない。追記は不要。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 5)。M1 は、届かない RESULT を見張りが黙って見逃す道を 1 つ残す。条件は「前の TASK の件が open か acked」に限られるので止める理由にはしないが、直すことを勧める。直さずに進めるなら、PR の本文に known gap として書き、台帳に行を足す
- Follow-ups:
  - M1 の最小の直しは `evaluateSeat` の recurNote 1 か所と、それを固定するテスト 1 本(上の調べのテストの形でよい)。TASK が変わったら新しい ALERT にする直しは、台帳の (a) と同じ `watchPendingAlert` と件の本文に触れる
  - L1 と L5 は雛形、本文、テストの文の直しで、判定の論理は変わらない
  - L4 の「遡れない TASK を覚える」は `ResultSeen` と同じ形の欄を 1 つ足す直し
  - どれも Go か雛形に触れるので、記録だけの修正には当たらない。直したら `/self-review` から回し直す
