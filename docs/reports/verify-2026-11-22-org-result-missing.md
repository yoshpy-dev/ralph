# Verify report: org-result-missing

- Date: 2026-11-22(Run 2、cycle 2。`--cycle auto` は worktree の `.harness/state/standard-pipeline/cycle-count.json` の `cycle` 2 を読む)
- Plan: `docs/plans/active/2026-11-22-org-result-missing.md`(issue #361)。`- Approved:` の digest `698af1d189e8` は HEAD `f9476afb` でも一致する(cycle 2 の計画の変更は Progress の行だけ)
- Verifier: verifier サブエージェント(`/verify`)
- Scope(Run 2): Run 1 の report のコミット `46c5040b` から HEAD `f9476afb` までの 7 コミット(`git diff --stat 46c5040b..HEAD` は 30 ファイル、+955 −188)。Go を変えたのは `e976f232`(テストだけ)、`31d69757`(sync-docs: `--help` の文字列、leader の雛形、テスト)、`cc04cb0a`(cross-review の ACTION_REQUIRED 1 の直し)、`f9476afb`(self-review Run 3 の R3-L1〜L3 の直し)。branch 全体の比べ元は merge base `2a069b50`。origin/main は `b263b1f3`(#364)に進んだが、取り込んでいない
- Evidence: `docs/evidence/verify-2026-11-22-org-result-missing.log` の末尾に Run 2 の出力を足した(Run 1 の中身は頭にバイト単位で残っている。`head -c <前の大きさ> | cmp` で確かめた)。wrapper 自身のログは `docs/evidence/verify-2026-11-22-054504.log`。どちらも `.gitignore` の `docs/evidence/*.log` に当たる
- テストは走らせていない(`./scripts/run-test.sh` も narrow の `go test` も)。テストの対応は本文を読んで確かめ、rule の pin だけは python の写しで HEAD と前の版に当てた
- 構成: Run 2 を上に置き、`## Verdict` は Run 2 の 1 つだけにした。Run 1 は下の「Run 1」に見出しを 1 段下げて残し、文は変えていない。所見の番号は Run 1 の続き(V12 から)

cross-review の ACTION_REQUIRED 1 は閉じた。10 秒の幅は、`ralph org send` が Enter から `sent` の記録までにかける時間のコードの上限と合い、2 つの広げ方はどちらも上げない側にしか動かない。self-review Run 3 の R3-L1〜L3 は `f9476afb` で頼まれたとおり入った。full の static verify は rc=0。残りは台帳の数と注(V12、V13、LOW)と INFO 6 件で、どれも /sync-docs に渡す。

## Run 2 (cycle 2)

### cross-review の指摘の行き先

| # | 指摘 | HEAD | 根拠 |
| --- | --- | --- | --- |
| 1(ACTION_REQUIRED) | `Send` は `sent` の TS を Enter と送信の確かめのあとに取るので、手番の短い座席の RESULT や QUESTION が TASK の時刻より前に見えて捨てられ、誤った `result_missing` が上がる | 閉じた。直し方は triage の案の 1 つ目(Enter から記録までの最大の幅だけ TASK の時刻より前から数え、QUESTION の答えの比べも同じ幅で上げない側に寄せる)のとおりで、台帳の形と `verbs.go` は変えていない | `resultMissing` は `repliesFrom := task.At.Add(-resultMissingSentLag)`(`internal/org/watch_result_missing.go:249`)を 1 回だけ計算し、履歴の読み(`seatLinesToLeader` の `since`、`historyReachesBack`)と行の数え(`l.At.Before(repliesFrom)`)の両方に渡す。答えは `firstSentAfter(..., last.At.Add(resultMissingSentLag))`(`:270`)で、秒に切った時刻が厳密に後(`ts.After(at)`)のものだけ。テスト(読んで確かめた): `TheClockStartsAtTheTaskOrAtTheLeadersAnswer` の表に、T−2 s の RESULT(上げない)、T−10 s ちょうどの RESULT(上げない)、T−11 s の古い RESULT(6 分で上げる)、T−2 s の QUESTION(上げない)、答えが Q+2 s と Q+10 s の行(上げない)、Q+2 s の送信のあと 3 分の答え(8 分で上げる)。`ReadsBackToTheSentLagBeforeTheTask` は T−5 s の RESULT を 50 行の外から 200 の読みで拾う |

### 10 秒の導き方とコード

`Send` が Enter を押してから `sent` の時刻を取るまで(`internal/org/verbs.go`):

| 段 | コード | 長さ |
| --- | --- | --- |
| Enter | `typeAndSubmit` の `PaneSendKeys(ctx, paneID, "Enter")`(`verbs.go:918`) | herdr の呼び出し 1 回。キーは呼び出しが戻る前に pane に届くので、座席の返事はここから始まりうる |
| herdr の道の確かめ | `confirmSubmitted`(`verbs.go:956-960`)が `AgentWait` に herdr の `--timeout` として `capMSToContext(ctx, 3 s)` を渡す | herdr が自分の `--timeout` で終わる限り 3 s まで |
| 画面の道の確かめ | `sendScreenSubmitChecks` の `confirm`(`internal/org/send_fallback.go:316-330`)が `context.WithTimeout(ctx, 3 s)` の中で `PaneRead` を 250 ms ごとに読む | 3 s。窓の終わりに走っていた読みは ctx で kill され、`execRun` の `cmd.WaitDelay = execWaitDelay`(`internal/org/driver/driver.go:46`、`:85`)の 2 s まで待たれる |
| 記録 | `unlock()`(`verbs.go:674`)のあと `appendEvent(ManifestEvent{TS: o.now(), ...})`(`:723`)。`o.now()` は UTC の RFC3339 で秒まで(`internal/org/spawn.go:412-418`)。TS は `Manifest.Append` のロックを待つ前に決まる | 手元の処理だけ |

3 s と 2 s に、負荷のある機械で herdr の呼び出しが始まって戻るまでの余白 5 s を足して 10 s になり、定数の doc の言い分と合う。`SendSubmitConfirmTimeout` を変えるのはテストだけで(`internal/org/spawn.go:245` の欄を設定するコードは `_test.go` の外にない)、本番の窓は 3 s。秒への切り捨ても、Enter から記録までが 10 s 以内なら、Enter 以後の座席の行の秒は「記録の秒 − 10 s」より前にならない(床関数の差で確かめた)。`TestResultMissing_SentLagCoversSendsSubmitCheck` は `defaultSendSubmitConfirmTimeout + driver.ExecWaitDelay() + 5 s ≤ resultMissingSentLag` と秒の整数を確かめ、`driver.ExecWaitDelay()` は `execWaitDelay` をそのまま返すので、driver の値の変更もこのテストに届く。

確かめていないこと: herdr の道は herdr が自分の `--timeout` を守ることに頼る(守らなければ、上限は Send の ctx の `--timeout-ms` になり 10 s を超えうる。そのときは誤って上げる側に倒れる。定数の doc が「too short」の場合として書いている)。余白の 5 s は測っていない。

### 2 つの広げ方の向き

- 読みの遡り: `since` が T−10 s になったので、読み直しが増えるか、1,000 行でも遡れずに「わからない」(`ResultOutOfReach`、上げない)になるだけ
- 数える窓: [T−10 s, T) の行が加わる。同じ TASK_ID の RESULT か REVIEW なら `ResultSeen`(上げない)。`last` を変えるのは T 以後に座席の行がないときだけで、それが QUESTION か同じ TASK_ID の BLOCKED なら答え待ちになり、時計は T 以後の答えから始まるか、答えがなければ上げない。ほかの型なら時計は前と同じく T から始まる
- 答えの比べ: Q+10 s より後に記録された `sent` だけを答えにするので、時計の始まりは同じか遅くなるか、答えがなく上げない
- 上げなくなる道はどれも `resultMissingNo` で、`evaluateSeat` は座席の stall を前どおり決める(`resultMissingNo` の doc、`internal/org/watch_result_missing.go:34-45`)。stall の保留の時計(`resultMissingClockRuns`)は `task.At` から数えたままで、lag に関わらない
- `ResultSeen` と `ResultOutOfReach` は TASK の鍵で覚えるので、記憶の意味は変わらない。`resultMissingAlertOfTask` の比べ(alert_id の時刻が TASK の秒より後)も lag に関わらない

self-review Run 3 が挙げた 3 つ目の道(T の前 10 s にある QUESTION が `last` になる)も 2 点目に入り、上げない側に倒れる。

### self-review Run 3 の 4 件の行き先(`f9476afb`)

| Run 3 | 頼まれたこと | HEAD |
| --- | --- | --- |
| R3-L1 | 境目のテストの行を `t0.Add(-resultMissingSentLag)` に置き、名前と doc を直す | 直った。`TestWatch_ResultMissing_AReadWhoseOldestLineIsInTheReachBacksFirstSecondDoesNotReachBack`(`internal/org/watch_result_missing_test.go:689`)は RESULT と 50 行を `since := t0.Add(-resultMissingSentLag)` に置き、読みが 50 と 200、ALERT が 0 であることを見る。変異 Rc(`historyReachesBack` の `r.At.Before(since)` を `!r.At.After(since)` にする)では最初の読みで遡れたとみなして RESULT を落とし、6 分で上げるので落ちる(読んで確かめた。走らせていない) |
| R3-L2 | (a) driver の値を読む関数、(b) rule の「10 seconds」を定数と結ぶ | 両方入った。(a) `driver.ExecWaitDelay()`(`internal/org/driver/driver.go:48-55`)を足し、テストの下限は herdr の呼び出しの 5 s を含む。(b) rule のテストは `resultMissingSentLag` が秒の整数であることを確かめ、定数から `"10 seconds"` を作って断片 5 つに入れた。仕様の「10 秒」は結ばれていない(V15) |
| R3-L3 | rule の窓を 1 回だけ言う。`resultMissingAlertMessage` の doc。折り返し | 直った。rule は「do not reach back to 10 seconds before the `TASK`'s time」「at or after 10 seconds before the `TASK`'s time」と言い、後ろの文は理由と前の TASK の RESULT の扱いだけになった。段落の行はどれも 80 字以内(awk で数えた)。`--help` の「ALERTs sent」だけの行もなくなった |
| R3-L4 | 台帳 565 行目の数と注 | 残っている(V12) |

rule の pin の python の写し(`TestAgentMessagingRule_ResultMissingAndTheSeatsSendCheck` の Escalation の節の最初の 12 断片を、`skillOrderedFragmentProblems` と同じく空白を除いて順に探す)は、HEAD の 2 写しで 12 断片すべてを見つけ、`31d69757` の rule では新しい 5 断片を見つけない。

### Spec compliance (Run 2)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(S0) | Met(変わらず) | cycle 2 は証拠のファイルにも S0 にも触れていない(`git diff --stat 46c5040b..HEAD -- docs/evidence/org-result-missing-hook-probe-2026-11-22.md` は空) |
| AC2(S1) | Met(変わらず) | `implementer.md`、`reviewer.md`、`prompts_seat_send_test.go` は cycle 2 で変わっていない(同じ `git diff --stat` が空) |
| AC3(S2) | Met | ALERT のヘッダ(`CONDITION: result_missing`、`SEAT`、`TASK_ID`)と上げる条件の形は同じで、数える窓が T−10 s に広がっただけ |
| AC4(S2) | Met | 一致・作業中・時間がまだ・読めない・leader・消える・stall と重ねない、の各テストは cycle 2 で変わっていない。上げない側の行が増えた |
| AC6(S2) | Met(読みが 1 か所狭まった、V18) | (a) は答えが Q+10 s より後に記録されたときに上げる(表の (a) の行は答えが 2 分後。足した行は Q+2 s の送信を答えにせず、3 分の答えから 8 分で上げる)。(b) に 10 s のうちの送信を答えにしない行が足された。(d) の境目は T−10 s の秒に置き直された(R3-L1)。(f) の同じ秒の RESULT は前どおり数える。(c)(e)(g)(h)(i) のテストは変わっていない |
| AC5 | Met | leader の雛形、skill、rule は `31d69757` のあとも今のコードと合う。`check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh`(14 skill)は wrapper の中で通った |

計画の Verify plan の「今の ALERT の escalate、recurred、usage_limited の扱いと食い違わないこと」: cycle 2 は `escalateToInbox`、`recordResultMissingRecurrence`、`raiseResultMissing`、`usageLimited` の判定に触れていない。`git diff -U0 31d69757..HEAD -- internal/org/watch_result_missing.go` の hunk は、履歴の上限の doc、新しい定数、`resultMissing` の doc と本体、`resultMissingAlertMessage` の doc、`firstSentAfter` の doc だけで、`watch.go` と `watch_recurrence.go` は `b0a9b28a` から変わっていない。Run 1 の判断のまま。

### Static analysis (Run 2)

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | Pass(rc=0) | `Requested scope: full`。shellcheck と hook の `sh -n`、settings の `jq -e`、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、台帳の計画の参照、golang の verifier(`gofmt: ok`、golangci-lint `0 issues.`)、branch の secret scan(`2a069b50..f9476afb`、clean) |
| `./scripts/plan-visual.sh digest` | `698af1d189e8` | 計画の `- Approved:` と一致 |
| `cmp`(rule 2 写し、`/org` と `/director` の 4 写しずつ、recipe 2 写し) | 同じ | quality-gates はファイル全体では KNOWN_DIFF で、Watchdog の行は 2 写しで同じ |
| rule の pin の python の写し | HEAD 12/12、`31d69757` 7/12 | 上の「self-review Run 3 の 4 件の行き先」の下 |

### Documentation drift (Run 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/ralph/agent-messaging.md`(2 写し) | Yes | 窓を 1 回だけ言い、遡る理由、前の TASK の RESULT を数えること、答えの「more than 10 seconds after that line, to the second」がコード(秒に切って `ts.After(q+10s)`)と合う |
| `docs/specs/2026-08-01-org-runtime.md` の 2026-11-22 改訂 | ほぼ | (a)(b) は 10 秒とその理由を書き、コードと合う。「10 秒」はテストで結ばれていない(V15)。(c) の「履歴が TASK まで遡れなかった」は要約(V16) |
| `/org` skill(4 写し) | ほぼ | 要約。「履歴が TASK まで遡れなかった」(V16)。`watch` の行が RESULT だけを言うのは Run 1 の V7 の残り |
| `/director` skill(4 写し) | Yes | 窓を言わない要約。V7 の残り |
| `ralph org watch --help` | Yes | `f9476afb` の変更は折り返しだけ。窓を言わない要約 |
| README、quality-gates(2 写し)、AGENTS.md | Yes | 窓を言わない要約 |
| ALERT の本文(`resultMissingAlertMessage`) | ほぼ | V17 |
| `internal/org/watch.go` の doc | ほぼ | V16 |
| `docs/tech-debt/README.md` | 一部 | V12、V13、V14 |
| `docs/reports/sync-docs-2026-11-22-org-result-missing.md` | 一部 | V19 |
| 計画の Progress | 一部 | V18 |

### Findings (Run 2)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| V12 | LOW | tech-debt register | 565 行目(`watch_result_missing_test.go` の大きさ)の Trigger「The next change to `internal/org/watch_result_missing.go` or to its tests」に `cc04cb0a` と `f9476afb` が当たり、注がない(self-review R3-L4 を確かめ、数を HEAD に合わせた)。1,672 行 → 1,754 行(最後に変えたのは `f9476afb`)、`watch_result_missing.go` 675 → 726 行、テスト関数 31 → 33、「`TestResultMissing_ParseAgmsgHistoryLine` and the three after it」は今は 4 つ(`SentLagCoversSendsSubmitCheck` が足された)。最後の欄の「Test gaps 1, 2, and 4, closed by e976f232」の 1 は、`cc04cb0a` で境目を固定しなくなり、`f9476afb` の改名したテストで閉じ直した。1,672 + 300 = 1,972 には届いていない | `wc -l` と `grep -c '^func Test'` を `2a069b50`、`b0a9b28a`、`e976f232`、`cc04cb0a`、`f9476afb` で数えた(証拠のログ)、`git log -1 -- internal/org/watch_result_missing*.go` は `f9476afb` | 数と最後のコミットを直し、「hit this trigger ... did not pay it」の注を日付つきで足す。Test gaps 1 は `f9476afb` の `AReadWhoseOldestLineIsInTheReachBacksFirstSecondDoesNotReachBack` で閉じ直したと書く |
| V13 | LOW | tech-debt register | この branch の注のうち 3 行の数が cycle 2 で古くなった。362 行目の「`internal/org/watch_result_missing.go` (675 lines …)」と「`internal/org/watch_result_missing_test.go` (1,672 lines …)」は 726 と 1,754。370 行目と 431 行目の「`prompts_leader_result_missing_test.go` … 275」は 289(`f9476afb` が rule のテストの断片を変えた)。370 の Trigger「the leader template tests」は、`f9476afb` がそのファイルの中の rule のテスト(`TestAgentMessagingRule_ResultMissingAndTheSeatsSendCheck`)だけを変えたので、字面では当たる。362 と 431 の Trigger には cycle 2 は当たらない(エスカレーションのコードにも `leader.md` の節にも触れていない)。88、362、467 行目の「`watch.go` … 2,328 … at b0a9b28a, the last commit of the branch that changed it」は今も正しい | 同上の `wc -l`、`git diff --stat b0a9b28a..HEAD -- internal/org/watch.go internal/org/watch_recurrence.go` は空 | 3 行の数を直す。370 は当たったとして注に続けるかを /sync-docs が決める |
| V14 | INFO | tech-debt register | 字面では当たるが判断の要る行が 3 つ。181 行目の F-7「when the driver API is next touched」: `f9476afb` が package driver に公開の関数 `ExecWaitDelay()` を足した(`driver.go` 139 → 148 行)。F-7 は herdr のクライアントの NotFound の型の話で、この関数はそれと関わらない。251 行目(外部コマンドの runner が 6 つあり、それぞれ wait delay の定数を持つ): 新しい runner は足していないので当たらないが、`resultMissingSentLag` が `ExecRunner` の `execWaitDelay` に頼るようになった。runner をまとめるときにその値を変えると `TestResultMissing_SentLagCoversSendsSubmitCheck` が落ちる。517 行目(「the paragraph of `newOrgWatchCmd`」): `f9476afb` は pulse の条件の並び(`usage_limited /` の行を含む)を折り返し直しただけで、`31d69757` の注と同じ境目 | `git show f9476afb -- internal/org/driver/driver.go internal/cli/org_watch.go` | 181 と 517 は当たったとみなすかを /sync-docs が決める。251 には任意で 1 文(`resultMissingSentLag` がこの wait delay を含むこと)を足す |
| V15 | INFO | doc drift | 仕様の 2026-11-22 改訂 (a)(b) の「10 秒」はどのテストとも結ばれていない。rule の「10 seconds」は定数から作った断片で固定された。定数の doc の「change with it」の文が名指すのは submit の確かめと rule のテストだけ | `docs/specs/2026-08-01-org-runtime.md:155-156`、`internal/org/watch_result_missing.go:98-103`。この仕様を読むテストは `internal` にない(`spawn_test.go:1494` のコメントだけ) | 任意。定数の doc に「仕様の 2026-11-22 改訂 (a)(b) の 10 秒も合わせて直す」を足す |
| V16 | INFO | doc drift | 遡れなかった TASK を「TASK まで遡れなかった」と書く要約が残る。遡る先は今は TASK の時刻の 10 秒前。仕様 (c)、`/org` skill(`.claude/skills/org/SKILL.md:720`)、`watchSeatSnapshot` の doc の「whose history did not reach back to it with the last limit」(`internal/org/watch.go:319`)、`evaluateSeat` の doc の「A TASK whose history did not reach back to it」(`:1310-1311`)。正確な言い方は rule、仕様 (b)、`resultMissingHistoryLimits` の doc にある | 上の各行 | 次にその段落を直すとき「TASK の時刻の 10 秒前まで」にそろえる |
| V17 | INFO | wording | ALERT の本文の QUESTION・BLOCKED の分岐は「its last line to the leader since then is a QUESTION」と言うが、その行は表示する TASK の時刻より最大 10 s 前のことがある(Enter から記録までのあいだに座席が送った QUESTION)。1 文目の「no RESULT or REVIEW … since that TASK was sent at T」は、確かめた範囲のほうが広いので正しい。`resultMissingAlertMessage` の doc は正しく書いている | `internal/org/watch_result_missing.go:298-311`、本文の pin は `internal/org/watch_result_missing_test.go:1070-1078` | 任意。直すなら pin も合わせる。leader は画面を読むので害は小さい |
| V18 | INFO | AC | AC6 (a)「TASK → QUESTION → leader の DECISION → 入力待ちのまま RESULT を送らない、で上げる」は、DECISION が QUESTION の 10 s より後に記録されたときだけになった。10 s のうちに記録された DECISION は答えにならず、そのあと何も送られなければ上げない(stall は前どおり立つ)。rule と仕様 (a) は書いているが、計画の Progress には cross-review の直し(`cc04cb0a`)と `f9476afb` の項がない(self-review Run 3 の項が `cc04cb0a` に触れるだけ) | 計画の Progress checklist、`TheClockStartsAtTheTaskOrAtTheLeadersAnswer` の Q+2 s・Q+10 s の行 | Progress に cross-review の直しの項を足し、AC6 (a)(b) が 10 s の幅で読まれることを書く(計画の本文の AC は変えない) |
| V19 | INFO | record | sync-docs の report の 72 行目が、`f9476afb` で改名したテストの古い名前 `TestWatch_ResultMissing_AReadWhoseOldestLineIsInTheTasksSecondDoesNotReachBack` を挙げる。self-review の R3-L1 の名前は履歴なのでそのままでよい | `docs/reports/sync-docs-2026-11-22-org-result-missing.md:72` | cycle 2 の sync-docs の report で新しい名前を書く |

### Run 1 の所見の行き先

| Run 1 | HEAD | 直したコミット |
| --- | --- | --- |
| V1(MEDIUM) | 直った。`--help`(`internal/cli/org_watch.go:118-123`)、quality-gates(2 写し)、README、`/org` skill の受信箱の段落と `watch` の行、`/director` skill の `recurred` の段落、仕様 (b) が、`result_missing` は TASK ごとに数えると言う | `31d69757` |
| V2 | 直った。`--help` と quality-gates の条件の並びに `result_missing`、stall の保留の文 | `31d69757`(`--help` の折り返しは `f9476afb`) |
| V3 | 直った。仕様 (b) に 5 つの欄と探し直し、(c) と `/org` skill に遡れなかった TASK の除外 | `31d69757` |
| V4 | 直った。rule の失敗の道の文、台帳 563 行目の (c) | `31d69757` |
| V5 | 直った(注が足された)。362 行目の注の数は cycle 2 でまた古くなった(V13) | `31d69757` |
| V6 | 直った(recipe 2 写し) | `31d69757` |
| V7 | 一部。README と leader の雛形は「RESULT(か REVIEW)」。`/org` skill の `watch` の行と `/director` skill の項は RESULT だけ(sync-docs が INFO として残した) | `31d69757` |
| V8、V9 | 開いたまま(INFO。次にその段落、その構造体に触るとき) | — |
| V10 | PR の本文に書く(INFO) | — |
| V11 | 書いていない(530 行目の Trigger に当たっていない。任意) | — |

### /sync-docs に渡すもの (Run 2)

1. 台帳 565 行目(V12、self-review R3-L4): 数、最後のコミット、テスト関数の数、末尾のユニットテストの数、Test gaps 1 の閉じ直し、日付つきの注
2. 台帳 362、370、431 行目(V13): 注の数(726、1,754、289)。370 の Trigger に当たったとみなすかの判断
3. 台帳 181 と 517 行目(V14): 字面で当たる。注を足すかの判断。251 行目は任意で 1 文
4. 計画の Progress(V18): cross-review の直し(`cc04cb0a`)と R3 の直し(`f9476afb`)の項、AC6 (a)(b) の 10 s の幅
5. sync-docs の report(V19): 改名したテストの名前
6. 任意(INFO): 定数の doc に仕様の 10 秒を足す(V15)、遡る先の言い方をそろえる(V16)、ALERT の本文(V17)
7. 当たらなかった行(確かめた): 563 行目の (a)〜(d)(`watchPendingAlert`、`watchEscalationBody`、`raiseResultMissing`、`recordResultMissingRecurrence`、`escalateToInbox` は cycle 2 で変わっていない。self-review Run 3 が「直さないなら (e)」とした R3-L1・L2 は直ったので (e) は要らない)、467 行目と 88 行目(`watch.go` と `watch_recurrence.go` に触れていない)、267、376、118、389、190、182 行目(cycle 2 のコードのコミットは `/director` skill、recipe、quality-gates、`leader.md`、`/org` skill に触れていない。`31d69757` の分は sync-docs が注を足した)

### Coverage gaps (Run 2)

- テストは走らせていない(/test の役目)。R3-L1 の変異 Rc が新しいテストで落ちることは、テストの本文と `historyReachesBack` を読んで確かめただけ
- 10 秒のうちの余白 5 s は測っていない。herdr が `agent wait --timeout` を守るかは実地で確かめていない
- 本物の herdr と agmsg の座席での動きは確かめていない(Run 1 と同じ)
- 台帳の掃き出しは、cycle 2 で変わったファイルと関数の名前(`driver`、`ExecWaitDelay`、`watch_result_missing`、`resultMissing`、`firstSentAfter`、`historyReachesBack`、`org_watch.go`、`agent-messaging`、`prompts_leader_result_missing`、`the spec` など)で 4 つ目の欄を拾い、当たった行を読んだ。`Send` や `verbs.go` で拾った行は、cycle 2 が `verbs.go` に触れていないので外した

## Verdict

- Verified: cross-review の ACTION_REQUIRED 1 が閉じたこと(読みと数えの窓、答えの比べ、テストの行)。10 秒の導き方がコードの時間の流れと合うこと。2 つの広げ方が上げない側にしか動かず、上げなくなる道では stall が前どおり決まること。self-review Run 3 の R3-L1〜L3 の直し。rule の pin(python の写しで HEAD と前の版に当てた)。AC1〜AC6(静的に)。`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` が rc=0。計画の digest の一致。rule、skill、recipe の写しの一致
- Partially verified: 文書と台帳の同期。rule と仕様 (a)(b) は今のコードと合うが、台帳の数と注(V12、V13)、判断の要る行(V14)、計画の Progress(V18)、sync-docs の report(V19)が追いついていない
- Not verified: 振る舞い(/test に渡す)、余白の 5 s と herdr の `--timeout` の実地、本物の座席と herdr での動き、#361 の原因
- Verdict: pass(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2、INFO 6)。受け入れ条件と静的解析は通った。V12 と V13 は /sync-docs で直すこと

---

## Run 1 (cycle 1、履歴)

Run 1(HEAD `b0a9b28a`)の report。見出しを 1 段下げたほかは、書いたときのまま残している。

- Date: 2026-11-22(cycle 1。worktree に `.harness/state/standard-pipeline/` がないので `--cycle auto` は 1)
- Plan: `docs/plans/active/2026-11-22-org-result-missing.md`(issue #361)。`- Approved:` の digest `698af1d189e8` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する(承認の直しの `6ced9a6c` のあとの計画の変更は Progress の行だけ)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff origin/main...HEAD`(origin/main `2a069b50`、HEAD `b0a9b28a`、main の取り込みは `b0c56556`)。33 ファイル、+3773 −101。self-review Run 1 の直し `f57cb92d` と Run 2 の直し `b0a9b28a` を含む
- Evidence: `docs/evidence/verify-2026-11-22-org-result-missing.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。wrapper 自身のログは `docs/evidence/verify-2026-11-22-033407.log`
- テストは走らせていない(`./scripts/run-test.sh` も narrow の `go test` も)。受け入れ条件のテストの対応は、テストの本文を読んで確かめた

受け入れ条件はすべて満たしている。静的解析は通った。文書には、`f57cb92d` で変わった「別の TASK の立ち上がりは新しい ALERT と件になる」という決まりが rule の外に届いていないずれがある(V1、MEDIUM)。ほかは LOW 5 件と INFO 5 件で、どれも文書と台帳の直しなので /sync-docs に渡す(下の「/sync-docs に渡すもの」)。

### Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(S0) | Met | `docs/evidence/org-result-missing-hook-probe-2026-11-22.md` に 1 段目(dispatcher に座席の形の 18 コマンドを 66 回、exit 1 は 0 回、1 になるのは `TMPDIR` が無いか書けないときだけ)と 2 段目(codex の sandbox で agmsg の保存先に書けないと `send.sh` が出力なしで exit 1、何も保存しない)がある。原因は見込みで、確かめられていない。hook は変えず、計画の外の 2 点は台帳の 561 行目(dispatcher の `mktemp`)と 562 行目(agmsg の黙った失敗)にある。新しい issue は起こしていない(V10) |
| AC2(S1) | Met | implementer と reviewer の雛形(`internal/org/prompts/implementer.md:115-160`、`reviewer.md` の同じ節)が、送信のコマンドの形 `bash {{AGMSG_SCRIPTS}}/send.sh {{TEAM}} {{SEAT_ID}} leader "$(cat <ファイル>)"`、終了コードの確かめ、1 回の送り直し、`SEND FAILED:` の行(stderr が空のときの要点も)、終了コード 0 のあとの `history.sh` の確かめ、TASK に答えずに手番を終えないことを言う。`prompts_seat_send_test.go` の 7 本が固定している(`NamesTheSendCommand`、`ChecksTheExitCodeAndRetriesOnce`、`TheMessageFileStartsWithTheTypeLine`、`ChecksTheHistoryAfterExitZero`、`DoesNotEndTheTurnWithoutAReply`、ほか 2 本) |
| AC3(S2) | Met | `TestWatch_ResultMissing_RaisesForAnIdleSeatWithoutTheResultAndClearsOnIt` が fake の herdr と fake の history.sh で、`stall_minutes` の手前で上げず、過ぎたら 1 通上げることを見る。`assertResultMissingAlert` が `CONDITION: result_missing`、`SEAT`、`TASK_ID` を見る。BLOCKED と QUESTION は、leader の答えのあとは時計を数え直して上げる形に計画で直された(Progress の「計画の直し」、AC6 (a)) |
| AC4(S2) | Met | 一致するメッセージ・作業中・時間がまだ・履歴を読めない・切れた TASK_ID・STOP は `TestWatch_ResultMissing_IsNotRaisedWhenTheSeatAnsweredOrItCannotBeTold` の表、leader は `IsNotRaisedForTheLeader`、消えることは `RaisesFor...ClearsOnIt` と `ANewTaskClearsItAndIsWatchedOnItsOwn`、stall と重ねないことは `TheSeatGetsNoStallWhileItHolds` と `HoldsTheSeatsStallWhileItsClockRuns` |
| AC6(S2) | Met | (a)(b)(c)(e) は `TheClockStartsAtTheTaskOrAtTheLeadersAnswer` の表の行(名前に (a) などが付いている)、(d) は `ReadsFurtherBackForAResultTheFirstLimitLeftOut` と `IsNotRaisedWhenTheHistoryDoesNotReachBackToTheTask`(1000 行で上げない、999 行で上げる、覚えたあとは読まない)、(f) は「the RESULT came in the TASK's own second」、(g) は「a RESULT the leader answered with a DECISION」、(h) は `WaitsForTheSeatToStayWaitingForStallMinutes`、(i) は「a STOP followed the TASK」 |
| AC5 | Met | leader の雛形の運用規律の末尾(`internal/org/prompts/leader.md:1664-1689`)、rule の 2 写し(`cmp` で同じ)、`/org` と `/director` の skill(`.claude` と `.agents` は `diff` で同じ、`templates/base` の写しとも `cmp` で同じ)が `result_missing` を言う。`check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh`(14 skill)は wrapper の中で通った |

計画の Verify plan の「見張りの新しい条件が、今の ALERT の escalate、recurred、usage_limited の扱いと食い違わないこと」:

- escalate: `escalateToInbox` は `result_missing` の ALERT を鍵の open の件に畳まない(`!isResultMissingAlert(alertID)`)。これは食い違いではなく、別の TASK の件に吸われないための意図した違いで、rule に書いてある
- recurred: 同じ TASK の立ち上がりは `recordResultMissingRecurrence` が保留中の ALERT か alert_id が同じ件にだけ畳む。別の TASK は新しい ALERT。状態ファイルを失ったときは `resultMissingAlertOfTask` が alert_id の時刻(TASK の秒より後)で同じ TASK の ALERT を探し直す。`evaluateCycle` は `now` を取ってから台帳を読む(`internal/org/watch.go:1074-1076`)ので、前の TASK の ALERT の時刻は次の TASK の秒より後にならない、という doc の言い分は読みの順と合う
- usage_limited: 利用上限で止まった座席は `result_missing` を評価しない(`usageLimited`、`TestWatch_ResultMissing_ASeatStoppedOnAUsageLimitGetsNone`)

#### self-review Run 2 の 4 件の行き先(`b0a9b28a`)

| Run 2 | 頼まれたこと | HEAD |
| --- | --- | --- |
| R2-L1 | `evaluateSeat` と `resultMissingNo` の doc に、遡れなかった TASK は stall を抑えないことを足す | 直った。`evaluateSeat` の括弧に「neither known to be done nor out of the history's reach (ResultOutOfReach)」、後ろに「holds the stall in no later cycle either」の文。`resultMissingNo` の doc にも同じ文 |
| R2-L2 | 少なくとも rule に 1 文と `escalateToInbox` の括弧。安い守りは任意 | 安い守り(`resultMissingAlertOfTask`)を入れ、テスト 3 本(`TheSameTaskAfterALostStatusFileIsARecurrenceOfItsItem`、`ANewTaskAfterALostStatusFileIsNotFoldedIntoTheEarlierTasksItem`、`TestResultMissing_AlertOfTask`)。`escalateToInbox` と `raiseResultMissing` の doc は、残る道(台帳か受信箱が読めない、append の失敗)を書いた。rule の文と台帳 563 行目の (c) はない(V4) |
| R2-L3 | `recordRecurrence` の doc とファイル冒頭から写しを指す | 直った(`internal/org/watch_recurrence.go:17-20` と `:176-182`)。467 行目の (7) のファイルの一覧は /sync-docs へ(V5) |
| R2-L4 | 2 つの doc を狭める。そろえ切るなら雛形とテストの断片も | 両方入った。`resultMissingLineOf` の doc はヘッダの範囲を書き、雛形は「`TASK_ID:` の行は、例のとおり 2 行目」と「`leader: ` の直後が `TYPE: <TYPE>\nTASK_ID: <TASK_ID>` で始まる」を言い、`prompts_seat_send_test.go` の断片も合わせた |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | Pass(rc=0) | `Requested scope: full`。shellcheck と hook の `sh -n`、settings の `jq -e`、Codex の hook の guard 3 つ、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、台帳の計画の参照、golang の verifier(`gofmt: ok`、golangci-lint `0 issues.`)、branch の secret scan(`2a069b50..b0a9b28a`、clean) |
| `./scripts/plan-visual.sh digest` | `698af1d189e8` | 計画の `- Approved:` と一致 |
| `cmp` / `diff`(rule、skill、recipe の写し) | 同じ | rule 2 写し、`/org` と `/director` の 4 写しずつ、`docs/recipes/isolated-herdr-server.md` の 2 写し |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/ralph/agent-messaging.md`(2 写し) | ほぼ | 「How a seat sends to the leader」と `result_missing` の段落は今のコードと合う(TASK ごとの ALERT、同じ TASK だけを畳む、遡れなかった TASK の記憶と stall への効き方)。残る道の 1 文がない(V4)。座席の確かめの書き方が雛形より緩い(V8) |
| `ralph org watch --help`(`internal/cli/org_watch.go`) | No | V1、V2 |
| `docs/quality/quality-gates.md`(2 写し) | No | V1、V2 |
| `README.md:284` | 一部 | `result_missing` の説明はある。同じ行の再発の決まりが例外を書かない(V1)。RESULT だけを言う(V7) |
| `AGENTS.md` の repo map | Yes | `watch_result_missing.go` の説明は「no RESULT or REVIEW」と stall の保留を言い、コードと合う |
| `/org` skill(4 写し) | 一部 | `result_missing` の項はある。ALERT の件の再発の段落(`.claude/skills/org/SKILL.md:677-688`)と `watch` の行の再発の文が例外を書かない(V1)。stall の保留の文が遡れなかった TASK を除かない(V3、INFO 寄り) |
| `/director` skill(4 写し) | 一部 | `result_missing` の件の扱いはある。`recurred` の段落(`:1082-1086`)の「立つたびに新しい件は増えない。続いている 1 つの問題として読み」が例外を書かない(V1)。RESULT だけを言う(V7) |
| `docs/specs/2026-08-01-org-runtime.md` の 2026-11-22 改訂 | 一部 | (b) の「立ち直しは `recurred` になる」が `f57cb92d` より前の書き方(V1)。(c) が遡れなかった TASK を抑えの外に置かない(V3) |
| `internal/org/prompts/leader.md` | ほぼ | `result_missing` の項は RESULT だけを言う(V7) |
| `docs/recipes/isolated-herdr-server.md`(2 写し) | No | V6 |
| `docs/tech-debt/README.md` | 一部 | V5 |
| `internal/config/config.go` のコメント | 一部 | V9(INFO) |

### Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| V1 | MEDIUM | doc drift | `f57cb92d` で、別の TASK の `result_missing` の立ち上がりは、同じ座席の件が open か acked のあいだも新しい ALERT と新しい件になった。rule はそう書いたが、再発の一般の決まりを書くほかの場所は例外を書かず、「同じ org・座席・条件の件が resolve されていなければ再発は `recurred` で、新しい ALERT も件もない」と言い切っている。director は `/director` skill のとおり、1 つの座席の `result_missing` の件が 2 つあれば 1 つの問題として読みうる | `internal/cli/org_watch.go:93-97`(`TestOrgWatch_HelpDescribesAlertRecurrencesLeaderStallAndReservedScope` が固定している文)、`docs/quality/quality-gates.md:97` と写しの `:96`、`README.md:284`、`.claude/skills/org/SKILL.md:677-688`(「見張りを起動し直して記録を失っても…`recurred` を足す」も `result_missing` には当たらない)と `:394` の `watch` の行、`.claude/skills/director/SKILL.md:1082-1086`、仕様の 2026-11-22 改訂 (b) | どの場所にも「`result_missing` は TASK ごと: 別の TASK の立ち上がりは新しい ALERT と件、畳まれるのは同じ TASK だけ」を 1 文足す。`--help` は Go の文字列なので、固定された文を残して後ろに文を足せば pin は変わらない。直したら `go test ./internal/cli/ -run OrgWatch_Help` を走らせる |
| V2 | LOW | doc drift | 見張りの条件を並べる 2 か所が `result_missing` を挙げず、leader 以外の座席の stall が時計のあいだ保留されることも言わない | `internal/cli/org_watch.go:71-72`(「heartbeat-stall / process-liveness / worktree-scope-change / usage_limited ALERTs」)、`docs/quality/quality-gates.md:97` と写し(「stall / liveness / scope-change / usage_limited ALERT」) | 並びに `result_missing` を足し、stall の保留を 1 文で書く(V1 と同じ直しでよい) |
| V3 | LOW | doc drift | 仕様の 2026-11-22 改訂 (c) は「入力待ちの座席に終わったと分からない TASK があるあいだ」stall を立てないと言う。履歴が 1000 行でも遡れなかった TASK も「終わったと分からない」に入るが、コードはその TASK の stall を抑えない(`resultMissing` の頭で `resultMissingNo`)。`/org` skill の「その時計が回るあいだ」も時計の定義に例外を書かない。(b) は状態ファイルの欄を `result_seen` と `idle_since` だけ挙げ、`result_out_of_reach`、`result_missing_task`、`result_missing_alert` を挙げない | `docs/specs/2026-08-01-org-runtime.md:156-157`、`internal/org/watch_result_missing.go:198-200`、`.claude/skills/org/SKILL.md:719`、rule の「neither known to be done nor out of the history's reach」 | (c) に「履歴が遡れなかった TASK と、結果を見つけた TASK は除く」を足す。(b) に 3 つの欄と、状態ファイルを失ったときの探し直しを足す。`/org` skill は rule と同じ言い方にする |
| V4 | LOW | doc drift | rule の「A rise for the same `TASK` follows the rules above, with that `ALERT` and its own item only.」は、`b0a9b28a` のあとも失敗の道では成り立たない。台帳か受信箱が読めない、`AppendRecurred` が resolve 以外で失敗した、のどれかで `recordResultMissingRecurrence` が false を返すと、同じ TASK に新しい ALERT ができ、`escalateToInbox` は `result_missing` を鍵の件に畳まないので 2 つ目の件になる。コードの doc はこの道を書いているが、rule と台帳にはない(self-review R2-L2 の残り) | `internal/org/watch_result_missing.go:302-305` と `:350-354`、`internal/org/watch.go:2057-2064`、`.claude/rules/ralph/agent-messaging.md` の `result_missing` の段落 | rule に「台帳か受信箱が読めないとき、記録に失敗したときは、同じ TASK でも新しい ALERT と件になる(黙る側には倒れない)」を足し、台帳 563 行目に (c) を足す |
| V5 | LOW | tech-debt register | 台帳の Trigger と持ち越しの注が、`f57cb92d` と `b0a9b28a` と main の取り込みに追いついていない。下の「/sync-docs に渡すもの」の 1 から 7 | `docs/tech-debt/README.md` の 88、182、362、377、467、563 行目(と候補の 389、431 行目) | 各行に日付つきの注を足す |
| V6 | LOW | doc drift | recipe の「Optional: a private agmsg and a notification shim」は、claude の座席は自分の agmsg の skill で送るので、座席が leader に報告する run では私用の写しを使わず「ralph が座席にパスを渡すまで」待て、と言う。S1 で座席の雛形は `{{AGMSG_SCRIPTS}}` から送るようになった。私用の写しで RESULT が届くかの実地の確かめはまだない(台帳 377 行目) | `docs/recipes/isolated-herdr-server.md:415-431` と `:770-778`(2 写し、`cmp` で同じ) | S1 で雛形がパスを渡すようになったこと、実地では確かめていないことを書き直す。377 行目の「then the recipe bullet」と合わせる |
| V7 | INFO | wording | ALERT の本文と rule は「RESULT or REVIEW」と言うが、leader の雛形、`/director` skill、`/org` skill の `watch` の行、README は RESULT だけを言う | `internal/org/prompts/leader.md:1666`、`.claude/skills/director/SKILL.md:1151`、`.claude/skills/org/SKILL.md:394`、`README.md:284` | 「RESULT(か REVIEW)」にそろえる。leader の雛形はテストの断片(`prompts_leader_result_missing_test.go`)も合わせる |
| V8 | INFO | wording | rule は座席の確かめを「the last line it sent the leader … carries the `TYPE` and `TASK_ID` it sent」と書き、仕様 (d) も「その TYPE と TASK_ID があること」と書く。`b0a9b28a` のあとの雛形は、ファイルの 2 行目を `TASK_ID:` にし、`leader: ` の直後が `TYPE: <TYPE>\nTASK_ID: <TASK_ID>` で始まることを確かめる | rule の「How a seat sends to the leader」、仕様の (d)、`internal/org/prompts/implementer.md:121-123` と `:147-150` | 要約として誤りではない。次に段落を直すとき雛形の形に合わせる |
| V9 | INFO | comment | `OrgWatchdogConfig` の `IntervalSeconds` のコメントは条件を「heartbeat stall, process liveness, scope change」とだけ並べ(`usage_limited` もない、前から)、`StallMinutes` は「the heartbeat-stall threshold」とだけ言う。`stall_minutes` は `result_missing` の待ち時間にもなった | `internal/config/config.go:339-344` | 次にこの構造体に触るとき直す |
| V10 | INFO | AC1 | AC1 の「原因が見つかったなら…tech-debt と issue に書いてある」について、原因は見込み(確かめていない)で、新しい issue はない。見込みは証拠と台帳 562 行目にあり、agmsg はこの repo の外(Non-goals) | 計画の Progress の S0、証拠の「結論」 | PR の本文で、#361 の原因は見込みのままで、届かないことを見えるようにした、と書く |
| V11 | INFO | tech-debt register | 530 行目は「`reseatProgressSentTask` と `sentSubmitUnconfirmed` が呼ぶ 1 つの読み手」を求める。S2 の `sentDetailsText` はその候補になった(今は `reseatProgressSentTask` と `seatLastTask` が呼ぶ)。Trigger(承認の条件、`checkApprovalDecision`、`Send` の Details の語)には当たらない | `internal/org/reseat_progress.go:291-307`、台帳 530 行目 | 任意。次に 530 行目を払うとき `sentDetailsText` を使うと書き添える |

### /sync-docs に渡すもの

頼まれた 8 件はどれも確かめた(1、2、4、5、8、9、10、11 に入れた)。足したものは 3、6、7、12、13、14。

1. 台帳 467 行目(確かめた): この branch の注は「`watch_recurrence.go`、…、`escalateToInbox`、…、`recordRecurrence` … were not touched」と言うが、`f57cb92d` が `escalateToInbox` のコード(鍵の件への畳み込みから `result_missing` を外した)を変え、`b0a9b28a` が `watch_recurrence.go` の冒頭と `recordRecurrence` の doc を変えた。行の Trigger(recurrence code)と (9) の Trigger(`escalateToInbox`、`recordRecurrence`)に当たり、払っていない。(7) の直しの対象に `watch_result_missing.go` の `recordResultMissingRecurrence` を足す(R2-L3)。行数は `watch.go` 2,249 → 2,328、`watch_recurrence.go` 277 → 288(`wc -l`、origin/main 2a069b50 と b0a9b28a)
2. 台帳 88 行目(確かめた): 注の「2,249 to 2,300 lines … at 43b7777e」と欄の `IdleSince` と `ResultSeen` は古い。今は 2,328 行(b0a9b28a)、`watchSeatSnapshot` の欄は 5 つ(`ResultOutOfReach`、`ResultMissingTask`、`ResultMissingAlert` を足す)
3. 台帳 362 行目(新しく見つけた): Trigger「The next change to the watch's escalation or to its tests」に `f57cb92d` の `escalateToInbox` が当たり、この branch の注がない。新しい `internal/org/watch_result_missing_test.go` は 1,580 行で、800 行の目安を超えるのに、台帳のどこにも行数の記録がない(`watch_result_missing.go` は 675 行で目安の内)
4. 台帳 563 行目(確かめた): (a) の Impact の「finds the TASK on the seat's screen or in the ledger」は、座席ごとに件が 1 つの書き方。M1 の直しのあと 1 つの座席に TASK_ID のない件がいくつも並びうるので、前の件の TASK は `ALERT_TS` の前にその座席へ送られた最後の TASK として台帳で探す、と足す。(b) の注は M1 の直しだけを書くが、`b0a9b28a` も `raiseResultMissing` を変えた(状態ファイルを失ったときの探し直し)ので Trigger に当たり、払っていない。(c) を足す: 残る 2 つ目の件の道(台帳か受信箱が読めない、append の失敗。V4)と、状態ファイルを失ったときは `resultMissingAlertOfTask` で閉じたこと
5. `README.md:284`(確かめた): V1 と V7
6. 台帳 182 行目(新しく見つけた): (a)「The next edit to the `/org` skill's command table」に `watch` の行の追記が当たり、注がない。表の行は 3,410 → 3,609 文字(python で数えた、origin/main 2a069b50 と HEAD)。4 写しとも同じ
7. 台帳 377 行目(新しく見つけた): main の取り込みで、#360 の注がこの branch の注の後ろに来た。#360 の注の「`reviewer.md` still has no `{{AGMSG_SCRIPTS}}`」は main の上では正しかったが、この branch では誤り(`reviewer.md` に 2 か所ある)。打ち消し線の中の「see the 2026-11-22 note at the end of this cell」も、もう末尾を指さない。「the 2026-11-22 note of fix/org-result-missing」と書き直し、#360 の文がこの branch の前の状態を言っていることを添える
8. leader の雛形の `result_missing` の項が REVIEW を言わない(確かめた): V7
9. 仕様の (c) と `/org` skill の stall の文(確かめた): V3。仕様の (b) の「立ち直しは `recurred`」も V1
10. `docs/recipes/isolated-herdr-server.md` の「Optional: a private agmsg」(確かめた): V6
11. 「その座席の記録がないときや読み書きの失敗のあとは同じ TASK に 2 つ目の件」: 状態ファイルを失ったときは `b0a9b28a` で閉じた。読み書きの失敗の道は残り、rule と台帳にない(V4)
12. `ralph org watch --help` と `docs/quality/quality-gates.md`(2 写し)(新しく見つけた): V1 と V2。`--help` を直すなら `internal/cli` の narrow test を走らせる
13. `/org` skill の ALERT の件の再発の段落と `/director` skill の `recurred` の段落(新しく見つけた、4 写しずつ): V1
14. 候補(Trigger に当たるかは判断): 431 行目(「adds a section to `leader.md` or `reviewer.md`」。見出しは足していないが、`leader.md` 1,663 → 1,689、`reviewer.md` 517 → 567、`implementer.md` 186 → 226 の規則の追記で、#312c-2 と #363 は同じ種類の追記に注を残した)。389 行目(「the rule that stops the seats (item 6 of the mission and the discipline section)」。運用規律に「この ALERT を理由に座席を stop も spawn し直しもしない」の項が入った。止める規則そのものは変わっていない)

### Observational checks

- 台帳の Trigger の掃き出し: 各行を ` | ` で切って 4 つ目の欄(Trigger)を python で読み、この branch が変えたファイルと関数の名前を含む行を挙げ、注の有無と Trigger の文を 1 行ずつ読んだ(scratchpad の `sweep2.py` と `cell.py`)。結果は上の 1、2、3、4、6、7、14。ほかの行(161、369、372、382、393、394、462、468、491、513、518 から 533、544、553、556、560 など)は、Trigger が名指す関数や節にこの branch が触れていない
- `resultMissingAlertOfTask` の言い分(前の TASK の ALERT の時刻は次の TASK の秒より後にならない)を、`evaluateCycle` の読みの順(`now` のあとに台帳を読む)と、`Send` が `sent` の TS を append の直前に `o.now()` で取ること(`internal/org/verbs.go:721-723`、`appendEvent` はロックを取らない)で確かめた。TS を取ってから書くまでのあいだに見張りの周が始まる窓はマイクロ秒の幅で、そこで取り違えても、leader が TASK を送ったことで保留中の ALERT は消えるので、新しい ALERT の側に倒れる(読んだだけで、走らせていない)
- `--help` の固定の文(`internal/cli/org_watch_test.go:1940-1970`)を読み、V1 の直しを文の追加にすれば pin が変わらないことを確かめた

### Coverage gaps

- テストは走らせていない(/test の役目)。AC2 から AC6 の「固定」は、テストの関数の本文を読んで対応を確かめたもの
- herdr の pane に立てた本物の座席での送信の確かめと `result_missing` の実地の動き、私用の agmsg の写しでの RESULT の届き方(V6、台帳 377 行目)は確かめていない
- issue の環境で agmsg の保存先に書けなかったかは確かめていない(証拠の「確かめていないこと」)
- 台帳の Trigger の掃き出しは、名前の一致で拾った行だけを読んだ。名前を出さずに振る舞いで書かれた Trigger は拾えていないかもしれない

### Verdict

- Verified: AC1 から AC6(静的に、テストの本文と照らして)。`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` が rc=0。計画の digest が一致。self-review Run 2 の R2-L1、R2-L3、R2-L4 は頼まれたとおり入り、R2-L2 は安い守りが入った。rule、skill、recipe の写しはそろっている
- Partially verified: 文書の同期。rule と AGENTS.md は今のコードと合うが、再発の一般の決まり(V1)、条件の並び(V2)、仕様 (b)(c)(V3)、失敗の道(V4)、台帳(V5)、recipe(V6)が追いついていない。どれも /sync-docs で直せる
- Not verified: 振る舞い(/test に渡す)、本物の座席と herdr での動き、#361 の原因
- Verdict: pass(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 5、INFO 5)。受け入れ条件と静的解析は通った。V1 から V6 は /sync-docs で直すこと
