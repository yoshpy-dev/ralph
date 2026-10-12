# Self-review report: org-watch-gate

- Date: 2026-10-12
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み、digest de5ca6b7e6f0)
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2、上限 2)。test が実機で見つけた後退を fdfb5b23 で直したあとの、全段のやり直し。cycle 1 の再実行(HEAD 7d2e4bbb)と最初の結果(HEAD 976a0ca9)は末尾の付録に残してある
- Scope: 前回(264951e6)のあとのコードの変更 2 つ。bda15444(N1〜N3 の直し)と fdfb5b23(同時の `--detach` のあとの `--stop` で、ロックを待っていた負けた子があとを取って動き続けた後退の直しと、test の probe 5 本)。`git diff 264951e6..4893b7e9 -- internal/`(9 ファイル、+481/-41)。4893b7e9 は計画の記録だけ。そのうえでブランチ全体(`git diff dc41c4f5...HEAD`)を、前回のあとで新しく気になる点がないかだけ見直した。テスト・静的解析・仕様適合・文書の整合の検査は行っていない
- 番号の付け方: cycle 2 の finding は C2-1〜C2-4。N1〜N3 は付録の cycle 1 の再実行、L1〜L6 は付録の最初の結果の番号

## Latest run: cycle 2(HEAD 4893b7e9)

### 前の指摘の扱い

| 対象 | 状態 | 根拠 |
| --- | --- | --- |
| N1(テストのコメントの空白) | 直った | `internal/cli/org_start_watch_test.go:341` は「AC13 and AC14: a watch」に戻った |
| N2(`--detach` の拒否の「nothing was started」) | 一部 | `orgWatchDetachRefusal` の 3 つの文は「so no new watch runs」になった。doc も、子を起動したあとにも使われることと、子の拒否が watch.log に残ることを書く(`internal/cli/org_watch.go:563-581`)。呼び手の `runOrgWatchDetach` の doc は「is refused with nothing started」のまま(C2-2) |
| N3(CLI の接頭辞の文字列比較) | 直った | dry-run の記録の追記の失敗を `fmt.Errorf("org: send: %w", err)` で包み(`internal/org/verbs.go:314-321`)、CLI は `result.Err` をそのまま返す(`internal/cli/org.go:953-954`)。`Send` の戻り口をもう一度すべて読み、どれも `org: send: ` で始まることを確かめた(`Send` の doc の「Every SendResult.Err starts with "org: send: "」のとおり)。org 層の `TestOrgSend_DryRun_AppendFails_ErrorStartsWithTheSendPrefix` と CLI のテストが、接頭辞が 1 回であることと EISDIR が鎖に残ることを見る |
| L5・L6 | 開いたまま(/sync-docs) | `docs/tech-debt/README.md` と `README.md` には 976a0ca9 のあとのコミットがない(C2-4) |

### Evidence reviewed

- 全文を読んだ差分: bda15444 と fdfb5b23 の `internal/` の全ファイル(`internal/cli/org_watch.go`、`org.go`、`internal/org/verbs.go`、`watchalive.go`、`watch.go`、テスト 4 ファイル)
- 周りで読んだ今のコード: `launchAndAwaitOrgWatch`・`ensureOrgWatch`・`runOrgWatchDetach`・`orgWatchDetachRefusal`、`startDetachedProcess` の `stop`(`internal/cli/org_watch.go:385-394`)、`runOrgWatchAll` の signal の扱い(`:153-155`)、`AcquireWatchLock`、`RunWatchAll` の始まり、`wireOrgStartWatchGate`、`Send` のすべての戻り口
- 経緯の確認のために、計画の Progress checklist の最後の 3 項目と verify・test の報告を読んだ
- 機械的な確認: `defaultWatchLockWait` はコードに残っていない。「nothing was started」はコードとテストから消え、似た言い方は `runOrgWatchDetach` の doc の「nothing started」だけ(C2-2)。`orgWatchChildExitWait` を使うのは `launchAndAwaitOrgWatch` だけ。偽の子を返す launcher のうち、ほかの pid の running の印でロックを取るものは、どれも `exited` を閉じるか `orgWatchChildExitWait` を自分で縮める(grep)。追加行にデバッグ出力・TODO・secret らしい文字列はない
- 確かめていないもの: テストは走らせていない。実機の確認は test の報告と Progress の記録を読んだだけ

### Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | exception-handling | `exitBy` を過ぎて子を止めたあと、`launchAndAwaitOrgWatch` は止める前の probe の pid をそのまま返す。`child.stop()` は子が終わるまで返らない(SIGTERM のあと、終わらなければ `orgWatchChildStopGrace` の 5 秒で SIGKILL)。SIGTERM を受けた子は、`signal.NotifyContext` が ctx を取り消すだけなので、ctx を見ない `AcquireWatchLock` の待ちを最後まで続ける。そのため止めるのに 2 秒ほどかかることがある。その間に別の端末の `--stop` でほかの見張りが止まると、`--detach` は動いていない見張りについて「already running (pid N)」と出して終了コード 0 で返る。止めた子がその間にロックを取っても、ctx が取り消されているので 1 サイクルで止まる。だから後退(見張りが残る)は起きない。start は ensure のあとで検査(`NewWatchGate`)がもう一度 probe するので、この場合は not_running で断られ、誤らない。計画の Progress が書く「残る穴」(返る前の `--stop`)の、もう 1 つの現れ方である | `internal/cli/org_watch.go:490-502`(probe、`child.stop()`、その前の probe の pid を返す行)、`:385-394`、`:273`(`orgWatchChildStopGrace`)、`:153-155`(`signal.NotifyContext`) | `child.stop()` のあとで probe し直し、作業を受け付ける見張りがあればそれを返す。なければループに戻すか、watch.log を名指すエラーにする。直さないなら、残る穴の tech-debt の行に「`--detach` の `already running` が、待つ間に止まった見張りを指すことがある」と足す |
| C2-2 | LOW | readability | N2 の直しが呼び手の doc に届いていない。`runOrgWatchDetach` の doc は「A watch that holds watch.lock and accepts no work is refused with nothing started」のままである。また、`already running` を出すのが自分の子が終わってから(最大 `orgWatchChildExitWait`)になったことも書いていない | `internal/cli/org_watch.go:535-541` | 「is refused, and no new watch runs (orgWatchDetachRefusal)」とし、`already running` は自分の子が終わるか止められてから出す、と 1 文足す |
| C2-3 | LOW | readability | `--detach` の拒否と起動の失敗の文は、`--state-dir` を明示しても、案内するコマンドに付けない。`orgWatchDetachRefusal` の「stop it with ralph org watch --stop」と「run ralph org watch --all --detach again」、`launchAndAwaitOrgWatch` の「see … and ralph status」がそうで、`--state-dir` なしの `ralph status` は既定の台帳(別の台帳)を読む。検査の文(`WatchGateError`)は `HintStateDir` で付ける。verify が先に挙げた点で(verify 報告の「気づいたこと」)、前回の私のレビューの見落とし。stale の文は 6a からの形で、stopping と `--once` の文と起動のあとの使い方はこの PR で足した | `internal/cli/org_watch.go:569-581`、`:528` | `orgWatchDetachRefusal` と errPrefix の文に `orgInboxHintStateDir(cmd, stateDir)` を渡し、`WatchGateError` と同じ組み立て(`watchGateCommand` を org 層から出す)で `--state-dir` を付ける |
| C2-4 | LOW | maintainability | 今回が最後の cycle(2/2)なのに、台帳に書くと決めたものがまだブランチにない。計画の Progress は残る穴(`--detach` が返る前に別の端末から `--stop` を打つと、ロックを待つ子があとを取りうる。foreground の `watch --all` も同じ)を「tech-debt に書く」とし、L5(tech-debt の 3 行)と L6(README)は /sync-docs に回してある。/sync-docs がこれらを入れないと、`/pr` で計画が archive されたあとは、どこにも残らない | `docs/plans/active/2026-10-12-org-watch-gate.md:192`。`git log 976a0ca9..HEAD -- docs/tech-debt/README.md README.md` は空 | /sync-docs で、残る穴の行を足し(C2-1 の現れ方も書く)、L5 と L6 を直す。C2-1〜C2-3 を直さないなら、同じ行か 1 本にまとめた行に足す |

件数: CRITICAL 0、HIGH 0、MEDIUM 0、cycle 2 の LOW 4 件(C2-1〜C2-4)。開いているのは、ほかに L5・L6 の 2 件(/sync-docs に回す)で、合わせて LOW 6 件。

ブランチ全体(`git diff dc41c4f5...HEAD`)で前回のあとに新しく気になったのは C2-3 だけで、ほかにはない。

### 依頼された点の結論(fdfb5b23)

1. `exitBy` の数え方: ほかの見張りが作業を受け付けているのを初めて見たときに決まり、そのあとは変えない。初めて見るのは起動のあとなので、`exitBy` は起動から 3 秒(`DefaultWatchLockWait` の 2 秒 + 1 秒)より前にはならない。子のロックの待ちは、子が起動して `AcquireWatchLock` に着いてから 2 秒で終わる。起動に 1 秒より長くかからなければ、子は `exitBy` の前に自分で終わる。ほかの見張りの最初の印が遅れて見えたときは `exitBy` も遅れるが、延びるのは待ちだけで、長くても全体の期限(10 秒)のあと 3 秒ほどである。doc(`orgWatchChildExitWait` と `launchAndAwaitOrgWatch`)のとおり
2. 3 つの分岐の順: 印が子の pid なら `started`、子が終わっていればほかの見張り、`exitBy` を過ぎていれば子を止めてほかの見張り、の順で正しい。待つ間に子がロックを取って印を書けば、`exitBy` の判定より先に `started` になる。作業を受け付ける見張りがないときは、前と同じ else の枝(子が終わって stale でない、または期限)で抜ける。直す前からある判定の限界として、子が印を書いたあとに死に、ほかの見張りがロックを取ってから最初の印を書くまでの一瞬には、子の印が running と読まれて `started` になる。この直しで広がってはいない
3. `child.stop()` のあとに probe し直さないこと: start は検査がもう一度 probe するので誤らない。`--detach` は C2-1
4. start が同じ経路を通ること: 通る。`ensureOrgWatch` は検査の中から呼ばれるので、待ちは manifest にも worktree にも触れる前に終わる。`TestOrgStart_AnotherWatchRuns_GoesOnOnlyOnceItsChildHasExited` は、子が終わる直前に、herdr が呼ばれておらず worktree もないことを確かめる。`wentOnFirst` は close の前に書いて close のあとで読むので、競合はない。`--plan` の 2 回目の検査は最初の probe でほかの見張りを受け入れるので、子を起動しない
5. 取り込んだテストの時間への頼り方: 下限はどれも決まった順で成り立つ。子の終わりはチャンネルで見てから返り、印は sleep のあとで書き、`exitBy` は起動のあとで決まる。上限は、子の 300 ms・500 ms や印の 200 ms に対して、3 秒の `orgWatchChildExitWait` か `orgWatchEnsureDeadline` で見ており、余裕は 2.5 秒以上ある。本番の settle のテストは実際に 2 秒待ち、[1.5 秒, 6 秒) で見る。TestMain は `orgWatchChildExitWait` を縮めないので、ほかの見張りが動いていて偽の子が終わらないテストを足すと 3 秒待つ。今そうなるテストはない(launcher を grep した)

### Positive notes

- 後退の直しは、子のロックの待ちを短くする案や止める頼みを残す案ではなく、自分の子が終わるのを待つ形にした。org 層の規則(ロックの待ち、頼みの扱い)を変えずに済み、待ちの長さを org 層の定数から決めているので、片方だけが変わることもない
- 待つ間に子がロックを取った場合を `started` として扱い、テストで固定した。止めた見張りのあとを子が取る場面が、黙って残る見張りではなく、報告される起動になる
- N3 の直しで「`Send` のエラーはどれも `org: send: ` で始まる」という決まりが org 層の 1 か所にまとまり、`Send` の doc に書かれた

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しい行はない。C2-4 のとおり、残る穴の行は /sync-docs が足す。

### Coverage gaps

- テストは走らせていない(/test の担当)。fdfb5b23 の新しいテストが直す前のコードで落ちることは、コードを読んで確かめた。直す前は、ほかの見張りの印を見た最初の probe で返るので、子の終わりを待つテストと `started` のテストは落ちる
- 実機の確認(`--stop` が 1 秒ほどで通り、見張りが残らない)は、Progress の記録と test の報告を読んだだけ
- C2-1 の筋は読んで確かめただけで、再現していない
- 付録の 2 つの結果の Coverage gaps は、この 2 つのコミットで変わっていない

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。後退は直り、子の終わりを待つことと待つ間の `started` にテストがある。N1・N3 は直り、N2 は呼び手の doc が残る(C2-2)。cycle 2 の LOW は C2-1〜C2-4 の 4 件で、どれも見張りが残る種類の誤りではない。L5・L6 と残る穴の行は /sync-docs で入れる)
- Follow-ups: 最後の cycle なので、C2-1〜C2-3 を直すなら上限を上げて回し直すことになる。直さないなら、C2-4 のとおり /sync-docs が台帳の行に入れる

## 付録

### 付録: cycle 1 の再実行(HEAD 7d2e4bbb)

- 再実行の Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)の中の再実行。最初の結果(HEAD 976a0ca9)は末尾の付録に残してある
- 再実行の Scope: 最初の結果の L1〜L4 を直した 2 つのコミット、dbeb45ca(L1〜L4)と 7d2e4bbb(CLI の send の二重の接頭辞。L3 を調べたときに見つかった、この PR より前からある包み)。`git diff f7a6fe6c..7d2e4bbb`(10 ファイル、+231/-73)。L5・L6 は /sync-docs に回すので、直したかは見ていない。テスト・静的解析・仕様適合・文書の整合の検査は行っていない
- 再実行の番号の付け方: 今回の finding は N1〜N3。L1〜L6 は付録の最初の結果の番号

#### 最初の結果の指摘の扱い

| 対象 | 状態 | 根拠 |
| --- | --- | --- |
| L1(子が終わったあとのエラーが相手の見張りを言わない) | 直った | `launchAndAwaitOrgWatch` は子が終わったあと、最後の probe が `not_running` でなければ `e.refuse(probe)` を返す(`internal/cli/org_watch.go:486-496`)。その probe は子が終わってから取ったものである。select で `childExited` が立ったあと、ループの頭でもう一度 probe するので、子がまだロックを持っていた時点の probe は使われない。抜ける条件は前と同じで、子が終わったあとは stale の間だけ期限まで待つ。stopping と `--once` はすぐ、stale は期限で拒否の文になり、ロックが空いていれば今までどおり watch.log を名指す(`TestOrgWatchDetach_ChildExitsFirst_ExitsOneNamingTheLog` がこちらの枝を通る)。新しいテスト 2 本(`TestOrgStart_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefusedAsTheGateRefuses` は start の 2 つの形 × 3 状態、`TestOrgWatchDetach_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefused` は `--detach` × 3 状態)は、直す前のコードでは「exited before it was running」になって落ちる。条件を stopping だけにした場合も、`--once` と stale の行で落ちる |
| L2(not_running の文が既存の org に start を案内する) | 直った | 文を「or, for a new org, start it with ralph org start, which starts one」に絞り、理由をコメントに書いた(`internal/org/watchgate.go:52-56`)。`watchgate_test.go` が新しい言い回しをそのまま固定する。古い言い回しは、コード・skill・雛形のどこにも残っていない(grep) |
| L3(send の拒否の `org:` の重なり、何もしない `TrimSpace`) | 直った | `taskNotSentError`(`internal/org/verbs.go:225-231`)が `org: send: TASK not sent: ` に、`org: ` を外した検査のエラーをつなぐ。`Unwrap` で検査のエラーを返すので、`errors.Is(err, ErrNoRunningWatch)` も、`errors.As` で `*WatchGateError` を取り出すことも、今までどおりできる。`TrimSpace` を消し、doc は「Parse trims the spaces around the key and the value and keeps their case」と直した。`splitHeaderLine` は key と値の両方を `TrimSpace` するので、doc は正しい(`protocol.go:140`、`:144`)。org と CLI のテストは、文の頭と `org:` が 1 回だけであることを見る |
| L3 の続き(7d2e4bbb、CLI の `org: send: org: send:`) | 直った | CLI は、`org: send: ` で始まるエラーをそのまま返し、ほかだけを包む(`internal/cli/org.go:953-958`)。`Send` の戻り口をすべて読んだ。`org: send: ` で始まらないのは dry-run の記録の追記(`verbs.go:318`)だけで、コメントのとおりである。変わるのは、二重だった文から 1 つ目の `org: send: ` が消えることだけで、ほかの文は変わらない。包みの `%w` は残るので `errors.Is` の鎖も同じ。古い二重の文をテストや文書が前提にしている箇所はない(`grep -rn 'org: send: org:'` は新しいテストの 1 行と過去の報告だけ) |
| L4(a)(b)(c)(d) | 直った | (a)(b) `WatchPIDText` と `WatchKillHint` を org 層に置き(`watchgate.go:83`、`:95`)、検査の文、`--detach` の拒否、`--stop` の 3 か所が同じ helper を呼ぶ。検査の stale の文は CLI の言い回し(「kill N (the pid in …watch-alive.json; check that it is still a ralph org watch first)」)にそろった。(c) `orgWatchGateOverride` は `func() error` になった。(d) `orgWatchEnsureDeadline` に改名し、コードとテストに古い名前は残っていない(grep。過去の報告の 1 行だけ) |
| L5・L6 | 対象外(/sync-docs) | この 2 つのコミットは文書に触れていない。どちらも開いたまま |

#### Evidence reviewed

- 全文を読んだ差分: dbeb45ca と 7d2e4bbb の全ファイル(`internal/org/watchgate.go`、`verbs.go`、`internal/cli/org_watch.go`、`org.go`、`main_test.go`、テスト 5 ファイル)
- 周りで読んだ今のコード: `launchAndAwaitOrgWatch` と `ensureOrgWatch` の全体、`orgWatchDetachRefusal`、`runOrgWatchStop`、`Send` のすべての戻り口(`verbs.go:290-446`)、send の CLI の失敗の経路(`org.go:902-960`)、`splitHeaderLine`
- 機械的な確認: `grep -rn 'orgWatchDetachDeadline\|watchGatePIDText\|orgWatchPIDText\|orgWatchKillHint\|check that pid .* is a ralph org watch and kill\|start the org with ralph org start'` は過去の報告とこの報告にしか当たらない。`rt.Send(` を呼ぶのは send のコマンドだけ。追加行にデバッグ出力・TODO・secret らしい文字列はない
- 確かめていないもの: テストは走らせていない。新しいテストが直す前のコードで落ちることは、読んで確かめた

#### Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | typo | dbeb45ca で新しいテストを上に足したときに、すぐ下の既存のコメントのコロンの後ろの空白が消えた(「AC13 and AC14:a watch that holds watch.lock」) | `internal/cli/org_start_watch_test.go:341` | 「AC13 and AC14: a watch」に戻す |
| N2 | LOW | readability | L1 の直しで、`--detach` の拒否の文(`orgWatchDetachRefusal`)は子を起動したあとにも出るようになった。文は 3 通りとも「so nothing was started」と言い、関数の doc も同じ前提で書かれている。実際には子を起動していて、その子はロックを断られて終わり、watch.log にその子の拒否の行が残る。動いている見張りを増やしていない点は正しいので害は小さいが、watch.log を開いた人は「何も起動していない」と食い違う行を見る。start の側(`WatchGateError`)はこの言い方をしないので、`--detach` だけの話である | `internal/cli/org_watch.go:535-550`(doc と 2 つの文)、`:521`(`refuse` に渡す)、`:486-496`(起動のあとの呼び出し) | 「so no watch was started」のように動いている見張りの話にそろえるか、起動のあとの呼び出しのときだけ「the watch it launched (pid N) was refused the lock」を足す。直さないなら関数の doc に「起動した子が断られたときも同じ文」と書く |
| N3 | LOW | maintainability | 7d2e4bbb は CLI でエラーの文の頭を見て包むかを決める。「`Send` のエラーは dry-run の記録の追記を除いて `org: send: ` で始まる」という決まりが、org 層の戻り口と CLI のコメントの 2 か所に分かれた。新しい戻り口が接頭辞なしで足されても CLI が包むので壊れはしないが、コメントの「only the others (the dry-run event's append)」は古くなる。dry-run の 1 か所が接頭辞を持てば、CLI の包みと文字列の比較はいらない | `internal/cli/org.go:953-958`、`internal/org/verbs.go:318`(`return SendResult{Err: err}`) | `verbs.go:318` を `fmt.Errorf("org: send: record the dry-run event: %w", err)` のように包み、CLI は `result.Err` をそのまま返す。`TestOrgSend_AnErrorWithoutTheSendPrefixGetsItOnce` の期待もそれに合わせる。今の形のままでも挙動に問題はないので、次に send のエラーを触るときでよい |

件数: CRITICAL 0、HIGH 0、MEDIUM 0、新しい LOW 3 件(N1〜N3)。最初の結果から開いているのは L5・L6 の 2 件(/sync-docs に回す)で、開いている LOW は合わせて 5 件。

#### 依頼された点の結論

1. L1 の `e.refuse(probe)` の分岐: 正しい。分岐に入るのは子が終わったあとだけで、そのとき使う probe は子が終わってから取ったものである。ロックを持っている見張りが stopping か `--once` ならすぐ、stale なら期限で拒否の文になる。ロックが空いていれば今までどおり watch.log を名指す。start では `WatchGateError` が返るので、`StartFeature` は worktree を作らず、新しいテストがそれを start の 2 つの形で確かめる。気になるのは `--detach` の文の「nothing was started」だけ(N2)
2. 子が終わったあとの stale の待ち: 前と同じで、期限まで待つ。期限の前に stale から not_running に変われば抜けて watch.log の文、running に変われば成功、stopping か `--once` に変われば拒否になる。待ちが延びる経路は増えていない
3. L4 の `WatchKillHint` で文をそろえたこと: 検査の stale の文は「stop it with ralph org watch --stop, or kill 4302 (the pid in …/watch-alive.json; check that it is still a ralph org watch first)」になり、続けて読める。pid がないときは「find the process that holds …/watch.lock (…/watch-alive.json names no pid) and kill it」。leader の雛形は文の「no new work」と `ralph org watch --all` で拒否を見分けるので、この変更の影響を受けない。skill と README は kill の文を引いていない
4. 7d2e4bbb がほかの send のエラーを変えるか: 変えるのは、二重だった `org: send: org: send: …` が `org: send: …` になることだけである。dry-run の記録の追記の失敗は今までどおり `org: send: ` が付き(`org: send: org: open …`)、新しいテストがそれを固定する。読み直しの案内(note)は包みの前に出すので、出方は変わらない

#### Positive notes

- L1 の直しは、最初の probe のときと同じ `e.refuse` を通るので、start と `--detach` のどちらでも、出会った時点にかかわらず同じ文になる。新しいテストは start の 2 つの形と `--detach` の両方で 3 状態を並べ、「exited before it was running」が出ないことも見る
- `taskNotSentError` は `Unwrap` を持つので、文を短くしても `errors.Is` と `errors.As` が今までどおり通る
- 7d2e4bbb は、この PR より前からある二重の接頭辞を、`errors.Is` の鎖を保ったまま直し、接頭辞のない 1 つの経路をテストで固定した

#### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しい行はない。

#### Coverage gaps

- テストは走らせていない(/test の担当)。新しいテストが直す前のコードで落ちることは、コードを読んで確かめた
- `TestOrgSend_AnErrorWithoutTheSendPrefixGetsItOnce` は manifest の場所にディレクトリを置き、`errors.Is(err, syscall.EISDIR)` を見る。macOS と Linux では、ディレクトリを書き込みで開くと EISDIR になる。Linux では読んだだけで確かめていない
- 最初の結果の Coverage gaps(`CLAUDECODE` の受け継ぎ、確かめられないロックのときの stale の文、前の版の見張りとの共有、Linux)は、この 2 つのコミットで変わっていない

#### Recommendation

- 再実行の判定: yes(CRITICAL・HIGH・MEDIUM なし。L1〜L4 は直り、L1 と L3 には直す前のコードで落ちるテストがある。新しい LOW は N1〜N3 の 3 件で、どれも挙動の誤りではない。L5・L6 は /sync-docs で直す)
- Follow-ups: N1 はコメントの 1 文字、N2 は文か doc の 1 行、N3 は任意の整理。直すと /self-review からやり直しになる。L5・L6 は /sync-docs で直す

### 付録: 最初の結果(HEAD 976a0ca9)

- 最初の結果の Scope: diff の品質だけ。`git diff dc41c4f5...HEAD`(HEAD 976a0ca9、33 ファイル、+3373/-319)。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない。計画の Design decisions F1〜F10 と Progress checklist に記録された決定(検査を `checkSpawnInput` の直後に置くこと、それより後の検査で断られた start が起動した見張りを残すこと、Dir を変えたときの `PWD` の書き換え、期待を変えた 6a のテスト)は、コードが記録と合っているかだけを見た

#### Evidence reviewed(最初の結果)

- 全文を読んだ差分: `internal/org/watchgate.go`、`watchalive.go`、`watch.go`、`spawn.go`、`feature.go`、`verbs.go`、`internal/cli/org.go`、`org_watch.go`、`status.go`、`main_test.go`、`internal/org/prompts/leader.md`。テストは `watchgate_test.go`、`watchgate_positions_test.go`、`watchalive_test.go`、`watch_all_test.go`、`prompts_test.go`、`org_watch_gate_test.go`、`org_start_watch_test.go`、`org_watch_test.go` の差分を読み、`status_test.go` は足したテストの名前とコメントを読んだ
- 文書: `/org` skill の 4 面(`cmp` で 4 つとも同じ)、agent-messaging の 2 面(同じ)、quality-gates の 2 面(同じ行を足した)、README、AGENTS.md、仕様の注、tech-debt の差分
- 周りで読んだ今のコード: `Parse`・`splitHeaderLine`・`Validate`・`ValidateText`(`internal/org/protocol/protocol.go`)、`ResolveOrgStateDir`・`MainWorktreeRoot`・`LedgerMainWorktreeRoot`(`internal/org/statedir.go`)、`fileLockHeld`(`internal/org/lockfile.go`)、`runOrgWatchStop`・`waitOrgWatchReleased`・`orgWatchKillHint`(`internal/cli/org_watch.go`)、`StartFeature` の前半、`writeMark`・`finish`・`stopAndRelease`(`internal/org/watch.go`)、start の RunE と `runOrgStartPlan`
- 機械的な確認: 追加行にデバッグ出力・TODO・secret らしい文字列はない。tech-debt で足した行と書き換えた行の 5 本は、どれもエスケープしていない `|` が 6 個で、列は崩れていない。RESOLVED の注が挙げるテスト 13 本は、どれも HEAD に 1 つずつある。`orgWatchGateOverride` を使うのは TestMain と `useRealOrgWatchGate` だけで、`internal/cli` のテストに `t.Parallel` はない(package 変数の差し替えと並行しない)。消した `newOrgSpawnRuntime` を呼ぶコードは残っていない
- 確かめていないもの: テストもバイナリも走らせていない(「Coverage gaps」)

#### 依頼された点の結論(最初の結果)

1. `NewWatchGate` の settle の繰り返し(`watchgate.go:107-127`): 判定し直すのは stale のときだけで、ほかの状態はすぐ返す。期限は実時計、印の判定は `now()` で、待ちは `min(watchGateProbeInterval, left)` なので期限を越えない。閉包が共有する可変の状態はない。`WatchGateError` の文は 5 通りとも `ralph org watch --all` と `no new work` を含み、leader の雛形のテストがそれを全状態で確かめている。running の枝は `--once` を前提に書かれているが、`ProbeWatch` が running を返すときは印が nil でないので、`WatchAcceptsWork` が false になる running は `Once` だけで、前提は成り立つ
2. `ProbeWatch` の stopping と印の deadman(`watchalive.go:206-226`): stopping は running と同じ窓で判定し、窓の外は stale になる。印の `deadman_minutes` は nil でなければ呼んだ側の値より先に使い、1 未満を 10 に読み替える規則は変わらない。前の版の印(欄なし)は呼んだ側の値で読む
3. `drainBeforeStop`(`watch.go:782-806`): stopping を書くのは書き直しの goroutine を始める前で、goroutine は `TS` だけを書く。元の goroutine は `close(done)` と `<-refresherDone` のあとで stopped を書くので、データ競合はない。BeforeStop がないときは stopping を書かない(doc のとおり。CLI はいつも BeforeStop を設定する、`internal/cli/org_watch.go:201`)。`OnLocked` は `defer w.finish(lock)` のあとに呼ぶので、`OnLocked` が panic してもロックは放される
4. 検査の位置: `Spawn` は `checkSpawnInput` の直後で、manifest の読み、dry-run の枝、`withManifestLock` の前(`spawn.go:464-472`)。`StartFeature` は `checkSpawnInput` の直後で、`Manifest.Read`・`spawnPrecheckErr`・worktree の前(`feature.go:194-200`)。`Send` は検証の直後で、dry-run の追記、`findSeat`、ctx を作る前(`verbs.go:277-291`)。どれも断ったときに何も書かない。ctx を作る前なので、検査の最大 2 秒は `--timeout-ms` を使わない。`Send` の `Parse` と `Validate` への分け方は `ValidateText`(`protocol.go:174-180`)と同じ順で、型付きのメッセージの扱いは変わらない。raw の TYPE は `Parse` が値を `TrimSpace` する(`protocol.go:144`)ので、空白と大文字小文字の扱いは計画 F5 のとおり
5. CLI: `ensureOrgWatch` と `launchAndAwaitOrgWatch` は計画の図 2 と tech-debt 206 の直しのとおりに動く。子が終わったら `exited` を nil にして select から外す書き方は正しく、そのあとは stale の間だけ待つ。`orgWatchChild` は環境から `HERDR_*` の 4 つを外し、Dir を変えたときだけ `PWD` を差し替える。`--config` は `filepath.Abs` で絶対パスにする。state dir は `ResolveOrgStateDir` が絶対パスで返す(`mustAbs`)ので、子の作業ディレクトリが変わっても `--state-dir` はずれない。banner は `OnLocked` から出すので、断られた 2 つ目の見張りは出さない。`wireOrgStartWatchGate` は ensure のあとで本物の検査を呼び、TestMain の差し替えがあるときは ensure を通らない。start に `--dry-run` はないので、dry-run が見張りを起動することはない
6. 子の起動の安全性: shell を通さず `os.Executable()` を exec し、引数は固定の語と絶対パスだけである。ロックの fd は Go の既定で close-on-exec なので子に渡らない。ログは 0644 の追記で 6a から変わらない
7. `status.go`: stopping と `(--once)` の行、`--json` の `once`、kill された見張りが残した stopping の印の行を足した。`statusDeadmanMinutes` の doc は、印に値がないときだけ使うと直してある
8. 雛形と文書: leader.md の項目は `TestRenderRolePrompt_Leader_WatchRefusalEscalatesWithoutRetry` が語句と `WatchGateError` の文の両方から固定している

#### Findings(最初の結果)

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L1 | LOW | exception-handling | 起動した子が、先にロックを取った stopping か `--once` の見張り(または stale のままの見張り)に断られて終わると、start と `--detach` のエラーは「the watch (pid N) exited before it was running; see watch.log」になる。「その見張りが終わってから打ち直す」という次の手は出ない。最初の probe で同じ状態を見たときは `e.refuse(probe)` の文が出るので、相手の見張りにどの時点で出会ったかで答えが変わる。起こるのは、cron の `watch --all --once` が始まった直後に start を打ったとき(子はロックを 2 秒待ってから終わる)など。watch.log には相手の pid を含む拒否の 1 行が残るので、調べれば原因は分かる | `internal/cli/org_watch.go:472`(`childExited && probe.State != org.WatchLivenessStale` で抜ける)、`:482-483`(childExited の文)。最初の probe の扱いは `:439-441` | 子が終わったあとで、ほかの見張りがロックを持っている(`probe.State` が `not_running` でない)なら `e.refuse(probe)` を返す。テストは `TestOrgStart_AChildThatExitsWhileAnotherWatchRuns_StartGoesOn` と同じ作りで、launcher の中で `onceWatchMark` を書いてロックを取り、閉じた `exited` の子を返せば足せる |
| L2 | LOW | readability | 見張りがない(not_running)ときの文は、spawn にも TASK の send にも「or start the org with ralph org start, which starts one」と案内する。TASK を送る相手の座席は動いている org の中にあり、(A) 現セッション昇格の org には leader の座席がない。その org の人が案内どおり `ralph org start` を打つと、headless の leader の座席が新しく立ちうる。文の「start the org」は新しい org の話と読めるが、すでにある org への spawn と TASK には当てはまらない | `internal/org/watchgate.go:51-53`。`prompts_test.go` は not_running の文が `ralph org watch --all` と `no new work` を含むかだけを見るので、語を足しても雛形のテストは崩れない | 「or, for a new org, start it with ralph org start, which starts one」のように新しい org に限る。`WatchGateError` は呼び手を知らないので、文を絞るほうが手数は少ない |
| L3 | LOW | readability | send の拒否は `org: send: TASK not sent: org: no ralph org watch --all is running ...` と `org:` が 2 回出る。S3 は起動のエラーで内側の `org: watch --detach:` を外して二重にならないようにしたが、send は `WatchGateError` の頭の `org: ` にそのまま重ねている。また `verbs.go:287` の `strings.TrimSpace(msg.Type)` は、`Parse` が値をすでに `TrimSpace` している(`protocol.go:144`)ので何もしない。doc と計画の「空白を除いて」と並べて読むと、`Parse` は除かないと受け取りうる | `internal/org/verbs.go:287`、`:289`。`internal/cli/org_watch_gate_test.go:324` と `internal/org/watchgate_positions_test.go:204` が `org: send: TASK not sent: ` を部分一致で見る | 包みの文を `org: send: TASK not sent` から `org:` の重ならない形にするか、`WatchGateError` の文の頭の `org: ` を外す(spawn と start の `rejected: ` の行も短くなる)。`TrimSpace` は消すか、`Parse` が除くことを 1 行のコメントで書く |
| L4 | LOW | maintainability | 同じものが 2 か所にある。(a) `watchGatePIDText`(`internal/org/watchgate.go:83-88`)は `orgWatchPIDText`(`internal/cli/org_watch.go:592-597`)と本体が同じ。(b) stale の見張りを kill する案内が、検査の文(`watchgate.go:55-58`、「check that pid N is a ralph org watch and kill it」)と、`--detach`・`--stop` の `orgWatchKillHint`(`org_watch.go:603-609`、「kill N (the pid in watch-alive.json; check that it is still a ralph org watch first)」)の 2 通りで書かれている。(c) `orgWatchGateOverride` の引数(`stateDir`, `deadmanMinutes`)はどの実装も読まない(`internal/cli/org.go:268`、`:327`、`main_test.go:69`)。(d) `orgWatchDetachDeadline` は start の待ちにも使うようになり、doc は直したが名前は `--detach` のまま | 左の各行 | (a)(b) は org 層に exported の helper を 1 つずつ置き、cli から呼ぶ。(c) は引数のない型にするか、引数を読むテストを足す。(d) は改名するなら `orgWatchEnsureDeadline`。どれも挙動は変えないので、次にこのあたりを触るときでよい |
| L5 | LOW | maintainability | tech-debt の開いた行 2 本がこの PR で古くなり、閉じた行 1 本に「still open」が残る。(a) 183 行の (c) F-7 は「`newOrgSpawnRuntime` (`internal/cli/org.go`) copies the first two steps of `newOrgRuntime`」と書くが、この PR は `newOrgSpawnRuntime` を消した。写しは、spawn・`start <task>`・`start --plan` の 3 か所に並んだ `ResolveOrgStateDir` の呼び出し(`internal/cli/org.go:550`、`:726`、`:823`)と、`newOrgSpawnRuntimeAt` の guard に移っている。(b) 205 行の (b) は `newOrgSpawnRuntimeAt` の呼び手を「(spawn and `start --plan`)」と書くが、今は `start <task>` も直接呼ぶ。(c) 202 行(201 として閉じた行)の Impact の欄に「still open: a cycle that takes longer than the deadman of `./ralph.toml` but not the watch's own」が残る。印が見張り自身の deadman を持つようになったので、この残りもこの PR で閉じた | `docs/tech-debt/README.md:183`、`:205`、`:202` | (a) は F-7 の文を今の形(3 か所の解決と `newOrgSpawnRuntimeAt`)に書き直す。Trigger の「`newOrgRuntime` の次の変更」はこの PR で起きていないので、行は開いたままでよい。(b) は呼び手に `start <task>` を足す。(c) は「still open: ...」に取り消し線を引くか、RESOLVED の注に「この残りも閉じた」と足す。/sync-docs で直せる |
| L6 | LOW | readability | README の「Either way the watch runs detached in the ledger's main worktree root, so without `--config` it reads that root's `ralph.toml`.」は、台帳に main worktree がないとき(repo の外を指す `--state-dir`)には打った場所で動く、という条件を落としている。start と `watch --detach` の help は「when it has one」、skill は「求められなければ打った場所」と書く | `README.md:256`。help は `internal/cli/org.go:693-694`、`internal/cli/org_watch.go:63` | 「in the ledger's main worktree root (when the ledger has one)」のように条件を足す |

件数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6。

#### Positive notes(最初の結果)

- `Org.WatchGate` を「nil なら検査しない」差し込み口にしたので、org 層の既存のテストは書き換えずに済んでいる。検査の位置は、stub の呼び出し回数と記録の snapshot で固定してある(`watchgate_positions_test.go`)。入力の検査より後で manifest の読みより前、という位置は、読めない manifest のときに検査の拒否が先に返ることでも確かめている
- 子の `PWD` を Dir に合わせる理由(Env を明示すると os/exec が `PWD` を直さない)をコメントに書き、`/bin/sh` の子で `pwd -P` と環境を確かめるテストを足している
- 起動のエラーから内側の `org: watch --detach:` を外し、呼び手の `errPrefix` で包む形にしたので、start と `--detach` が同じ起動の経路を通っても文が二重にならない
- leader の雛形のテストは、雛形の語句に加えて、`WatchGateError` の全状態の文がその語を含むことも見る。エラーの文を言い換えて雛形と食い違えば、テストが落ちる
- `drainBeforeStop` は、stopping の書き込み、goroutine の開始、終わりの `close` と待ち、stopped の書き込みの順で、mutex なしで競合しない

#### Tech debt identified(最初の結果)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しい行はない。L5 は既存の行の書き直しで、新しい負債ではない。

#### Coverage gaps(最初の結果)

- テスト、静的解析、バイナリでの実機確認は行っていない(/verify と /test の担当)。L1 の筋はコードを読んで確かめただけで、再現していない
- 子の環境から外すのは `HERDR_*` の 4 つだけである(F6)。Claude Code の Bash から start を打つと、子は `CLAUDECODE` などの変数も受け継ぐ。見張りの on-demand watcher が起動する `claude -p` がその環境でどう動くかは確かめていない。6a の `--detach` から同じで、この PR で始まったことではない
- `ProbeWatch` は watch.lock を確かめられない(open が ENOENT 以外で失敗する)とき、ロックありとみなす(6a からの規則、`internal/org/watchalive.go:191-194`)。その台帳では、検査と start は stale の文(「the ralph org watch --all that holds the lock ... find the process that holds ... and kill it」)を出し、確かめられなかったことは言わない。権限の壊れた台帳でだけ起こるので finding にはしていない
- 前の版の見張り(印に `once` も `deadman_minutes` もない)とこの版の検査が台帳を共有すると、前の版の `--once` の見張りは検査を通る。計画の Risks と Rollout のとおり release のノートで扱う
- Linux での動きは読んでいない(setsid の経路は 6a から変わっていない)

#### Recommendation(最初の結果)

- 最初の結果の判定: yes(CRITICAL・HIGH・MEDIUM なし。LOW 6 件。検査の位置、止まりかけと `--once` の見張り、start による見張りの起動、印の `deadman_minutes` は、計画の F1〜F10 と Progress checklist の記録どおりに実装されている)
- Follow-ups: L1〜L4 は小さなコードの直しで、直すと /self-review からやり直しになる。L5 と L6 は文書だけなので /sync-docs で直せる
