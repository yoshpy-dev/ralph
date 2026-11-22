# Walkthrough: org-result-missing

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-org-result-missing.md
- Issue: #361
- Base: main b263b1f3

差分は 44 ファイル、+5,207 −161 行。約 1,300 行は計画、報告、evidence、insight で、Go は 15 ファイル(+3,364 −66、うちテスト 6 ファイルが約 2,450 行)。残りは座席と leader の雛形 3 つ、rule、org と director の skill、recipe、quality-gates(`templates/base` と `.agents` の写しを含む)、仕様、README、AGENTS.md、tech-debt。計画は S0(調べ)、S1(座席の雛形)、S2 と S2b(見張り)、S3(leader の雛形と文書)の順に入れ、そのあと self-review から cross-review までを 2 周した。下の順に読むと早い。

## 1. 何が足りなかったか

issue #361 には、座席の RESULT が leader に届かず、leader が待ち続けた例が 3 つある。agmsg の `messages` の表に RESULT の行はなかった。気づけたのは、leader の停滞の ALERT を見た director が座席の画面を読んだからだった。

| org | 座席(driver) | TASK | 座席の画面 |
|---|---|---|---|
| docs-followup | reviewer(codex) | gate-1、gate-2 | gate の結果のあとに `Hook failed … exited with code 1` |
| monorepo | reviewer(codex) | gate-1 | 「gate が通過しました」のあとに同じ `Hook failed` |
| dataform | implementer(claude) | s11 | RESULT の中身を書いて手番を終えた |

足りないものは 2 つあった。座席には、送信の失敗に気づく手順がなかった。見張りには、TASK を受けた座席が結果を送らずに入力待ちでいることを、座席と TASK_ID を名指して知らせる条件がなかった。今は leader の停滞としか出ないので、leader は何を待っていて何が届いていないのかが分からない。3 つ目の受け入れ条件は、codex の座席で hook が失敗しても RESULT が届く、または届かないことが分かる、だった。

## 2. 調べ: hook は送信を止めていなかった(S0)

`Hook failed` が送信を落としたのかを、モデルを呼ばずに手元で調べた。記録は `docs/evidence/org-result-missing-hook-probe-2026-11-22.md`(b42383cb)。

- 1 段目: 座席が打ちそうな 18 のコマンドを、codex の形と Claude Code の形の入力にして `ralph-dispatch.sh PreToolUse` に渡した。66 回のうち終了コード 1 は 0 回だった。1 を返したのは `TMPDIR` が無いか書けないときだけで、原因は dispatcher の `mktemp` の失敗(`.claude/hooks/ralph-dispatch.sh:108`)
- codex 0.166.0 のソースでは、PreToolUse の hook が 1 で終わっても Failed と表示するだけで、コマンドは走る。止めるのは Blocked だけ
- 2 段目: hook だけでは届かなかった理由にならないので、codex の sandbox の中で agmsg の `send.sh` を走らせた。agmsg の DB のディレクトリに書けない sandbox では、`send.sh` は標準出力も標準エラーも空のまま終了コード 1 で終わり、何も保存しない。`history.sh` も 1 になる
- 結論: hook はこの計画で変えない。届かなかった原因の見込みは、座席の sandbox から agmsg の保存先に書けなかったこと。issue の環境でそうだったかは確かめていない(未確認)
- 2 段目の結果は S1 に引き継いだ。送信の失敗は標準エラーが空のことがあり、`history.sh` が失敗したときは送れたと数えない

## 3. 座席の側: 送ったかを確かめ、失敗を画面に残す(S1)

implementer と reviewer の雛形に、leader へ送るコマンドの形を書いた(`internal/org/prompts/implementer.md:128`、`internal/org/prompts/reviewer.md:437`)。今までは agmsg の skill に任せ、コマンドを名指していなかった。

`bash {{AGMSG_SCRIPTS}}/send.sh {{TEAM}} {{SEAT_ID}} leader "$(cat <ファイル>)"`

RESULT、BLOCKED、QUESTION を送ったら、次の順に確かめる。

1. 終了コードを見る。0 でなければ標準エラーを読み、1 回だけ送り直す。それでも 0 でなければ、画面に `SEND FAILED: <TYPE> <TASK_ID> <本文のファイルのパス>: <エラーの要点>` を 1 行書いて止まる(`internal/org/prompts/implementer.md:141`、`internal/org/prompts/reviewer.md:450`)。標準エラーが空なら、要点は `stderr なし、exit <終了コード>` と書く
2. 終了コードが 0 なら、`history.sh` で履歴を読む(`internal/org/prompts/implementer.md:145`)。履歴には座席が受けた行も入るので、`<座席> → leader:` の行に絞り、最後の行が送った `TYPE:` と `TASK_ID:` で始まることを確かめる。`history.sh` が失敗したときと行がないときは、送れたと数えない。`SEND FAILED:` を書いて止まり、ここでは送り直さない
3. TASK を受けたら、RESULT、BLOCKED、QUESTION のどれかを送って確かめ終えるか、`SEND FAILED:` を書くまで手番を終えない。`run_in_background` のコマンドの終わりを待つあいだは除く(`internal/org/prompts/implementer.md:157`)

- メッセージのファイルは `.harness/state/msg-<org>-<座席>-<TASK_ID>.md` で、1 行目を `TYPE:`、2 行目を `TASK_ID:` にする。TASK について送る QUESTION にも TASK_ID を付ける。見張りが履歴をこの形で読むため
- 座席は `ralph org send` を使わない。agent-messaging.md の「typed but not submitted のときは送り直さない」とはぶつからない
- `{{AGMSG_SCRIPTS}}` は leader の雛形だけが使う変数だった。座席の雛形にも渡るようになり、`[org].agmsg_home` が別の agmsg を指していても、ralph が書く保存先に座席も送る。`internal/org/prompts.go`、`internal/org/spawn.go`、`internal/cli/org.go` のコメントはこれに合わせて直した。`docs/recipes/isolated-herdr-server.md` の「Optional: a private agmsg」も書き直した

## 4. 見張りの側: `result_missing`(S2、S2b)

`internal/org/watch_result_missing.go`(728 行)が新しい。座席ごとの条件を 1 つ足し、leader 以外の座席だけが対象になる。読む順は次のとおり。

1. 呼び出し口: `internal/org/watch.go:1394` の `updateIdleSince`(`internal/org/watch_result_missing.go:180`)が、herdr が working と言わない周の最初の時刻を座席の記録に残す。working を見たら消す。そのあと `resultMissing`(`internal/org/watch_result_missing.go:231`)を呼ぶ。利用上限で止まった座席には呼ばない(org の `usage_limited` が知らせる)
2. TASK を台帳から読む: `seatLastTask`(`internal/org/watch_result_missing.go:513`)が、座席の今の life(最後の `spawn_started` か `spawned` より後)で最後の TASK の TASK_ID と時刻を、台帳の `sent` から読む。そのあとに STOP を送っていたら対象にしない。TASK_ID が切れている、空、制御文字を含む、時刻が読めない TASK も対象にしない(読めないものは上げない)
3. 入力待ちの時計: TASK の時刻と入力待ちの始まりの遅いほうから `[org.watchdog].stall_minutes` が過ぎるまでは、履歴を読まずに `resultMissingClockRuns` を返す(`internal/org/watch_result_missing.go:248`、`waitedFor` は `:497`)。新しい設定の鍵は足していない
4. 履歴の読みと遡り: `seatLinesToLeader`(`internal/org/watch_result_missing.go:574`)が `history.sh <team> <座席> <件数>` を 50、200、1000 件の順に読み、`<座席> → leader` の行だけを残す。座席を名指した履歴は受けた行も返すので、送り手と宛先で絞る。読みが TASK の時刻まで遡れたかは `historyReachesBack`(`internal/org/watch_result_missing.go:619`)が見る。1000 件でも遡れなければ「わからない」として上げず、その TASK は `ResultOutOfReach` に覚えて二度と読まない。読みの失敗や読めない行は上げず、次の周にまた読む
5. 済みの覚え: その TASK_ID の RESULT か REVIEW があれば済みで、`ResultSeen` に覚える(`internal/org/watch_result_missing.go:264`)。あとで件数の上限から行が落ちても上げない。済みの数えは TASK の時刻(下の 10 のとおり、その 10 秒前)から始める。RESULT のあとに leader が返事の DECISION を送っても、数え直さない(計画の直し。前の形は RESULT を範囲の外に置いて誤って上げた)
6. QUESTION・BLOCKED の答え待ち: 座席の最後の行が QUESTION か、同じ TASK_ID の BLOCKED なら、座席は leader の答えを待っている。その行のあとに leader が座席へ送った最初の `sent`(`firstSentAfter`、`internal/org/watch_result_missing.go:546`)から時計を数え直す。送っていなければ上げない(`internal/org/watch_result_missing.go:270`)。時計がそろうと、`CONDITION: result_missing`、`SEAT`、`TASK_ID` を持つ ALERT を leader に送る
7. ALERT の本文: `resultMissingAlertMessage`(`internal/org/watch_result_missing.go:300`)は失敗と決めつけない。見張りが見たことと時計の始まりを言い、座席の画面を `ralph org read` で読むよう頼み、バックグラウンドのコマンドを待っているなら待つよう添える
8. TASK ごとの ALERT: `raiseResultMissing`(`internal/org/watch_result_missing.go:357`)。最初は `raiseOrClear` を使っていたが、前の TASK の件が open か acked のあいだに次の TASK で立つと、空の note の `recurred` として前の件に畳まれ、leader に届かなかった(self-review Run 1 の M1)。今は座席の記録が最後の ALERT の TASK と alert_id を持ち(`internal/org/watch.go:339`)、別の TASK の立ち上がりは必ず新しい ALERT と新しい件になる。同じ TASK は自分の ALERT か件にだけ畳む(`recordResultMissingRecurrence`、`internal/org/watch_result_missing.go:406`)。`escalateToInbox` は result_missing を別の件の再発にしない(`internal/org/watch.go:2102`)。状態ファイルを失ったあとは、alert_id の時刻から TASK の ALERT を探し直す(`resultMissingAlertOfTask`、`internal/org/watch_result_missing.go:466`)
9. stall の保留(S2b): `internal/org/watch.go:1399` は、`result_missing` が立っている周と時計が回っている周に、leader 以外の座席の stall を上げない。S2 の最初の形では、長く作業した座席が入力待ちになると stall が先に上がり、あとから `result_missing` も上がって、同じ座席に 2 つの ALERT が出た。時計が切れて履歴が「済み」「答え待ち」「わからない」なら、stall は今までどおり決める。代わりに、RESULT を送ってから入力待ちになった座席の stall も、最大 `stall_minutes` 遅れる(テストで固定)
10. 10 秒の幅(cycle 1 の cross-review): `Send` は台帳の `sent` の時刻を、Enter と送信の確かめのあとに取る(`internal/org/verbs.go:723`)。手番の短い座席の返事は、記録された TASK の時刻より前に見えて捨てられ、誤った `result_missing` が上がる。`resultMissingSentLag`(10 秒、`internal/org/watch_result_missing.go:115`)で、返事の数えと履歴の遡りを TASK の 10 秒前から始め(`internal/org/watch_result_missing.go:251`)、QUESTION・BLOCKED の答えはその行の 10 秒より後に記録された `sent` だけにした(`internal/org/watch_result_missing.go:272`)。どちらも上げない側に動く。10 秒は、送信の確かめの 3 s、driver の WaitDelay の 2 s、herdr の呼び出しの余白 5 s の和。`driver.ExecWaitDelay()`(`internal/org/driver/driver.go:53`)は、テストが WaitDelay の値を読むために足した

## 5. leader と文書

- leader の雛形(`internal/org/prompts/leader.md:1664`): `result_missing` の ALERT を受けたら、座席を stop も spawn し直しもしない。座席の画面を `ralph org read` で読む。`SEND FAILED:` の行か、書いたのに送っていない結果が見えたら、その座席に DECISION を送って送り直しを頼む。TASK は送り直さない(作業のやり直しになり、per-plan では `PLAN:` のヘッダも要る)。同じ TASK でまた `SEND FAILED:` が出たら、BLOCKED を上げる。座席が作業中か、バックグラウンドのコマンドを待っているなら待つ
- rule: `.claude/rules/ralph/agent-messaging.md` に「How a seat sends to the leader」の節(`.claude/rules/ralph/agent-messaging.md:196`)と、`result_missing` の段落(`.claude/rules/ralph/agent-messaging.md:274`)を足した。`templates/base` に同じ写しがある
- skill: `/org` の `watch` の行(`.claude/skills/org/SKILL.md:394`)と `result_missing` の項(`.claude/skills/org/SKILL.md:715`)、`/director` の ALERT の件の扱い(`.claude/skills/director/SKILL.md:1305`)。4 写しずつ
- 仕様: `docs/specs/2026-08-01-org-runtime.md:155` から 158 行目の 2026-11-22 の改訂 (a) から (d)
- README、AGENTS.md の `internal/org/` の行、quality-gates の見張りの行、`ralph org watch --help`(`internal/cli/org_watch.go`)に、条件の名前を足した。TASK ごとに数えることは、AGENTS.md を除く 3 か所に書いた。AGENTS.md の行は、stall の保留も言う
- 台帳(`docs/tech-debt/README.md`): 新しい行を 5 つ足した。範囲外の発見 2 つ(dispatcher の `mktemp` の失敗で全 hook が 1 になる、agmsg の `send.sh` の黙った失敗)、この branch の既知の穴の行、claude の reviewer の gate、テストの大きさ。既存の約 20 行には、持ち越しの注か数の直しを足した

## 6. 2 周の経過

cycle 1 の経過。

- self-review Run 1: MEDIUM 1(M1)、LOW 5、Merge: yes。`f57cb92d` で直した。M1 の直しが上の 8
- self-review Run 2: LOW 4。`b0a9b28a` で直した。状態ファイルを失ったあとの探し直し(R2-L2)を含む
- verify Run 1(HEAD `b0a9b28a`): pass。MEDIUM 1(V1: 別の TASK は新しい ALERT になる決まりが、help、quality-gates、README、skill、仕様に届いていない)。sync-docs の `31d69757` で足した
- test Run 1: pass。変異 13 個のうち 2 個が落ちず、`e976f232` のテストで閉じた
- cross-review: codex が ACTION_REQUIRED 1 件(P2)。上の 10 の `sent` の時刻の遅れ。cycle の上限 2 のうち 1 周目なので、直して回し直した

cycle 2 の経過。

- `cc04cb0a` で `resultMissingSentLag` を足し、`f9476afb` で境目のテストと rule を置き直し、`0bf2a099` で `firstSentAfter` の絞り込みを固定した
- self-review Run 3: LOW 4。R3-L1 から L3 は `f9476afb` で直し、R3-L4(台帳の数)は sync-docs で直した
- verify Run 2: pass(MEDIUM 0、LOW 2、INFO 6)
- test Run 2: pass。変異 16 個のうち 5 個が落ちなかった。5 個は `firstSentAfter` の絞り込みを固定するテストがなかったためで、`0bf2a099` のあと、`firstSentAfter` の絞り込みを外す変異 G1 から G5 の 5 個はすべて新しいテストで落ちる
- cross-review: codex の指摘なし(cycle 2 / 2、上限に届いた)

main は途中で 2 回取り込んだ(`b0c56556` は #365 と #360、`45633318` は #364 の PR #375)。1 回目の衝突は org skill、AGENTS.md、台帳、`spawn.go` のコメントで、2 回目は台帳だけだった。どちらも main の行を先に置き、両側の追記を残して解いた。

## 7. 先に読むテスト

- `internal/org/watch_result_missing_test.go`: `TestWatch_ResultMissing_RaisesForAnIdleSeatWithoutTheResultAndClearsOnIt`(`:304`)、`TestWatch_ResultMissing_TheClockStartsAtTheTaskOrAtTheLeadersAnswer`(`:481`。TASK から数える、QUESTION の答えから数え直す、10 秒の境目の表)、`TestWatch_ResultMissing_ReadsFurtherBackForAResultTheFirstLimitLeftOut`(`:710`)
- `internal/org/watch_result_missing_test.go` の stall の保留: `TestWatch_ResultMissing_TheSeatGetsNoStallWhileItHolds`(`:872`)
- `internal/org/watch_result_missing_test.go` の TASK ごとの ALERT(M1): `TestWatch_ResultMissing_ANewTaskAlertsWhileTheEarlierTasksItemIsOpen`(`:1207`)、`TestWatch_ResultMissing_ANewTaskIsANewAlertWhileTheConditionIsStillActive`(`:1247`)
- `internal/org/watch_result_missing_test.go` の 10 秒の幅: `TestWatch_ResultMissing_OnlyARealSendToTheSeatAnswersItsQuestion`(`:616`)、`TestResultMissing_SentLagCoversSendsSubmitCheck`(`:1808`。送信の確かめの窓、WaitDelay、余白 5 s の和を下限として持つ)
- `internal/org/prompts_seat_send_test.go`: `TestRenderRolePrompt_SeatSend_ChecksTheExitCodeAndRetriesOnce`(`:103`)、`TestRenderRolePrompt_SeatSend_ChecksTheHistoryAfterExitZero`(`:169`)、`TestRenderRolePrompt_SeatSend_DoesNotEndTheTurnWithoutAReply`(`:206`)
- `internal/org/prompts_leader_result_missing_test.go`: `TestRenderRolePrompt_Leader_ResultMissingAlertSaysWhatTheLeaderDoes`(`:61`)、`TestAgentMessagingRule_ResultMissingAndTheSeatsSendCheck`(`:167`)
- `internal/cli/org_watch_result_missing_test.go`: `TestOrgWatch_HelpDescribesResultMissingAndItsRecurrencePerTask`(`:18`)

## 8. 確かめたこと

- `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` rc 0 が 2 回(1 周目は HEAD 46c5040b で 1,574 s、2 周目は HEAD 09f92f2e で 1,348 s)。足したテストは `-v -count=1` で SKIP 0
- 変異: 雛形と見張りの各段で 10、24、9、8 個(S1、S2、S2b、S3)のうち、落ちなかったのは結果の変わらない 2 個だけ。test の 1 周目は 13 個中 11 個、2 周目は 16 個中 11 個が落ち、残りの穴は `e976f232`、`f9476afb`、`0bf2a099` で閉じた
- `./scripts/run-static-verify.sh`(full)は verify の 2 回とも rc 0。sync-docs では `check-sync.sh`(DRIFTED 0)、`check-skill-sync.sh`(14 skill が lock-step)、gofmt、vet、`internal/org` の絞ったテストが通った
- 計画の承認の digest は `698af1d189e8` のまま変わっていない
- S0 の probe は、座席の形の 18 コマンドで dispatcher の終了コード 1 が 0 回、sandbox の中の `send.sh` が書けない条件で黙って終了コード 1、まで確かめた。モデルも本物の agmsg の DB も herdr のサーバーも使っていない
- cross-review の codex は、read-only の sandbox で Go の一時ディレクトリを作れず、絞ったテストを走らせていない。指摘なしは、diff を読んだ結果

## 9. 残したもの

- 原因は未確認。見込みは、座席の sandbox から agmsg の保存先に書けなかったこと(S0)
- 本物の座席と herdr での実行は、まだない。座席の手順、`SEND FAILED:` の行、`result_missing` の発生は、fake の herdr と agmsg の履歴、雛形の文の固定でだけ確かめた
- 10 秒の幅のうち、herdr の呼び出しの余白 5 s は測っていない。herdr が自分の `--timeout` を守ることに頼る。足りないと、誤った `result_missing` が 1 件多く上がる側に倒れる(黙る側ではない)
- 台帳の読み、受信箱の読み、`recurred` の追記が失敗したとき、同じ TASK に 2 つ目の件ができる。この道を走らせるテストはない
- 受信箱の ALERT の件には TASK_ID が入らない。同じ座席の件が並ぶと、古い件の TASK は台帳でしか分からない
- 新しい ALERT のたびに on-demand watcher が起動する(stall と同じで、`watcher_enabled` が既定で on)
- claude の reviewer が gate を `run_in_background` で待つあいだは、stall の代わりに `result_missing` が上がる(知らせの数は増えない)。本文と leader の雛形は、待っているなら待つと言う
- `internal/org/watch_result_missing_test.go` は 1,820 行で、800 行の目安を超える
- 動いている org は、座席を立て直すまで前の雛形のまま。見張りは新しい binary で動く
- origin/main(#364 の PR #375)を取り込んだ `45633318` のあと、full の `./scripts/run-test.sh` は回していない。絞ったテストと静的な確認は通った。通しは CI に任せる
- ALERT の本文の「its last line to the leader since then」は、記録された TASK の時刻より最大 10 秒前の QUESTION を指すことがある。`internal/org/watch.go` の 2 つの doc(`watchSeatSnapshot`、`evaluateSeat`)の「did not reach back to it with the last limit」は、「TASK の時刻の 10 秒前まで」にそろっていない。10 秒の数字は rule のテストと定数の和のテストが固定するが、仕様の改訂 (a)(b) と `watch.go` の doc は読むテストがない
