# Walkthrough: org-director-liveness(8 段目の後半の 2 本目の前半、8b-2a)

- Date: 2026-10-19
- Plan: docs/plans/active/2026-10-18-org-director-liveness.md(/pr で archive に移す)
- Branch: feat/org-director-liveness(base 88e37b1d、HEAD 2da83a76。この walkthrough を書いた時点で 29 commits、57 files、+7755/-335)
- PR: #237
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 48 files、+6588/-335 になる。そのうちテストは `internal/org/director_check_test.go`(+857)、`director_start_cleanup_test.go`(+804)、`internal/cli/org_director_cleanup_test.go`(+502)、`internal/org/director_test.go`(+445)、`watch_all_test.go`(+434)、`internal/cli/org_director_check_line_test.go`(+314)ほかで、17 files、+4408/-30、足した `func Test` は 70 本。非テストの Go は 15 files、+1632/-236。残りの 16 files は文書で、`/director` skill の 4 面(各 +106/-8)、`/org` skill の 4 面(各 +2/-2)、recipe `director-launch-cmd.md` と `director-wake.md` の 2 面ずつ、README、`AGENTS.md`、仕様の注 2 つ、tech-debt の 10 行(新しい行 3、書き換えた行 6、解決の注の行 1)
- 後工程は 1 回まわした(cross-review の triage は `Cycle: 1/2`)。cross-review は 0 件で、上限には届いていない(下の 9)

新しいファイルは `internal/org/director_check.go`(508 行)、`internal/org/director_start_cleanup.go`(147 行)、`internal/cli/org_director_check.go`(169 行)と、それぞれのテスト。次の順に読むと、headless の director が終わってから、それが status に出て、次の start が片付けて立て直すまでを 1 本で追える。

```
start が attached に自分の pid を書く → 見張りの cycle: 期限 → 確かめ → 起こす → pulse → 印
  → status と attach の断りが印を読む
  → 次の start: その場で確かめ直す → ended なら確かめた版の登録だけを片付ける → 記録 → 立てる
```

## 1. starter の記録と check-in の断り(`internal/org/director.go`、`director_start.go`、`internal/cli/org_director_headless.go`、S1、d48c0d23)

- `DirectorEvent` と `DirectorRegistration` に `starter_pid`(`StarterPID`)と `starter_start_time`(`StarterStartTime`)を足した。どちらも omitempty。`registrationOf` は headless の `attached` からだけ、正の pid と空白でない起動時刻の組のときに読む。8b-1 までの行、手書きの行、片方だけの行は「starter なし」になり、`attached` は今までどおり効く
- `StartDirector` は `checkDirectorStart` のあとで `directorStarter` を呼び、自分の pid と、`readDirectorProcess` で読んだその起動時刻を `AttachHeadless` に渡す。alive で起動時刻が読めたときだけ両方を書き、読めなければ両方とも書かずに進む。断られた start は ps を打たない(D1)。`AttachHeadless` は、片方だけの組と負の pid を、読み書きの前に断る
- check-in: `MarkCmdLaunched` は、starter が gone、zombie、起動時刻の違いのどれかなら `ErrDirectorCheckInRefused` で断り、何も書かない(`directorCheckInStarterEnded`。読みは stop の `directorProcessEnded` と同じ)。ps が失敗したとき、知らない state のとき、starter がないときは今までどおり受け付ける。起動スクリプトは check-in が失敗すると agent を起動せずに終わるので、遅れて開いた端末の agent は立たない。CLI は `orgDirectorCheckInReadProcess` を `ReadStarter` に渡す。`launched` の help に断りの条件を足した
- D2 からの逸脱: 計画は「ロックの下で starter を読む」だった。ps は最大 10 秒(`directorCheckInStarterTimeout`)かかり、`director.lock` を待つ側は 5 秒(`manifestLockTimeout`)で諦める。self-review L1 の指摘で、ロックの前に登録を読んで同じ検査を通ったときだけ ps を読み、ロックの下では検査をやり直して、starter が読んだときと同じ(`StarterPID` と `StarterStartTime`)ときだけ ps の結果を使う形にした(e85853c5)。starter が変わるのは `director.jsonl` を手で書き換えたときだけ。token が合わない呼び出しには ps を打たせない。D2 の本文は承認済みなので書き換えず、逸脱は計画の Progress と仕様の注に書いた

## 2. 確かめ(`internal/org/director_check.go`、`internal/org/driver/herdr.go`、`verbs.go`、S2a、8f608232)

- `CheckDirector(store, now)` は、今の登録が headless で stopped でないときだけ確かめる。昇格したセッション、stopped、登録なしは `Checked=false` で返す。結果は `DirectorCheck`(state、how、checked_at、detail)。state は `alive`、`ended`、`unknown`。how は `DirectorCheckHow*` の 16 個で、alive 3、ended 4、unknown 9
- 行は 3 つ(`checkDirectorOnce`)
  - `starting`(herdr も launch_cmd も): `checkDirectorStarter` が starter を ps で読む。starter なしは `no_starter`、gone か zombie は `start_ended`、起動時刻の違いは `starter_pid_reused`、同じなら `start_running`、ps の失敗は `starter_unreadable`
  - herdr の `running`: `checkDirectorAgent` が `herdr agent get director_<token>` を打つ。成功は `agent_found`。`agent_not_found` のときだけ、記録した pane を stop と同じ `confirmSeatPane` の label の照合で読む。label の合う pane があれば `agent_ended`、pane がなければ(pane の記録がない場合も)`pane_closed`、label の不一致と get の失敗は `pane_unconfirmed`。`agent get` のほかの失敗は `agent_get_failed`、client が `AgentGet` を持たなければ `no_agent_get`。各呼び出しは `callWithTimeout`(`DriverCallTimeout`、10 秒)で区切る
  - launch_cmd の `running`: `checkDirectorProcess` が記録した pid を読む。gone か zombie は `process_ended`、起動時刻の違いは `pid_reused`、同じなら `process_found`、ps の失敗は `process_unreadable`
- 起動時刻の違いは ended にせず unknown にする。pid の再利用か、Linux の時計の変更か見分けられないため。stop が使う `directorProcessEnded` は違いを ended と読むので、確かめは別の関数で読む(D3、計画の非ゴール)
- `driver/herdr.go`: `AgentGet` の失敗で、herdr の `agent_not_found` の envelope を `%w` で包むようにした(`checkHerdrAgentGetResult`)。`IsAgentNotFound` が読める。エラーの文は変わらない。`AgentStart` と `AgentWait` は `checkHerdrEnvelopeError` のまま。`IsNotFound` は `agent_not_found` に反応しない(agent がないことは、pane が閉じたことではない)。org 側は `verbs.go` の `agentNotFoundError` と `isAgentNotFound` で、driver を import せずに読む
- 確かめは何も書かない(`director.jsonl`、herdr の close、キー、signal のどれも)。herdr に打つのは `agent get` と `pane get`、`tab get`、`workspace get` だけ

## 3. 確かめを登録の版に結びつける(`director_check.go` の `directorVersion`、`CheckDirector` の試行、S2a)

- Codex の計画助言 HIGH への対応。2 つの start が重なると、後の start が `starting` を読んだあとで先の start が `launched` を書いて終わる。後の start の ps は start のプロセスが消えたのを見て ended と読み、立ったばかりの director を止めうる
- 1 回の試行は次の順: ロックなしで登録を読む(版 A)、`directorCheckAfterRead`(テスト用の seam)、ps か herdr を読む、`director.lock` の下で `readDirectorVersion` が版を読み直す。等しければ結果を版 A の登録とともに返す。違えば結果を捨ててもう 1 回やり、2 回(`directorCheckAttempts`)続けて変われば unknown の `changed` にする
- `directorVersion` は 4 つの値: 確かめる id、今の登録の id(`currentID`)、その id の event の数(`events`、parse できる行のうち `ev.ID == id`)、fold が読んだ登録の `%#v`(`folded`、手で書き換えた file への 2 つ目の守り)。`director.jsonl` は追記だけなので、event が増えれば `events` が増える。別の登録の event は、どれが current かを変えない限り版を変えない
- ロックの下では ps も herdr も打たない。ロックは読み直しだけに取り、書かない

## 4. 見張りのサイクルと印、起こす手順(`watch.go`、`director_check.go`、`director_wake.go`、`internal/cli/org_watch.go`、S2b、0756a73a)

- `watchAllRun.cycle()` の順は、期限の送り(`SweepInboxDeadlines`)、`checkDirector()`、`wake(check)`、各 org の pulse、生存の印。確かめを期限の送りのあとに置くので、確かめが herdr か ps で詰まっても、そのサイクルの人への送りは遅れない(8b-1 の D6 と同じ扱い)。遅れるのは起こす手順と pulse で、tech-debt の 203 行に確かめの分を書き足した(sync-docs)
- `checkDirector()` は `CheckDirector` が `Checked` を返したときだけ `director-alive.json` を書く。中身は `DirectorAlive`(id、attached_at、state、how、checked_at、detail の 6 キー)で、`writeJSONFileAtomic` で上書きする。今の headless の登録がなければ書かず、前の印をそのまま残す。読む側は `DirectorAliveFor` で、id と `attached_at` の両方が今の登録と同じ印だけを使う(`director.jsonl` を消すと id が振り直されるため。`attached_at` は S2b で足した)
- 確かめが読めなかった(`director.jsonl` が読めない、`director.lock` が取れない)ときは `watchdog: director check skipped this cycle: ...` の 1 行を出し、結果なしで起こす手順と pulse が進む。印が書けないときは `watchdog: director check mark not written: ...` の 1 行で、結果は起こす手順に渡る。見張りは `[org.director].enabled` を読まない(起こす手順と同じ)
- 起こす手順: `WakeDirector` に引数 `check *DirectorCheckResult` が増えた。`directorCheckFoundEnded` が true になるのは、同じ id と `attached_at` の登録を、同じサイクルの確かめが ended と読んだときだけ。そのとき `DirectorWakeSkippedEnded`(`ended`)で何も送らず、`woken` も書かず、登録ごとに 1 度だけ `watchdog: director d1 is not woken: the watch's check found it ended (<how>) (said once)` を出す。agent が終わった herdr の pane は shell に戻っていて、1 行がコマンドとして読まれるため。unknown、alive、別の登録の確かめ、nil は今までどおり送る(送り先は 8b-1 の label の照合が守る)。手順の番号が 3 つずれた
- `org watch` の Short と Long の help を、サイクルの順に直した

## 5. start の片付け(`director_start.go`、`director_start_cleanup.go`、`director_stop.go`、S3、3ece3deb)

- `checkDirectorStart` の最初の確かめ(今の登録を見る位置で、見張りの検査より前)が `checkCurrentDirector` を呼ぶ。昇格したセッションの登録は確かめず、今までどおり断る。headless なら印ではなくその場で `CheckDirector` を打つ
  - 確かめるものがない(stopped のまま current、など)ときは今までどおり断る
  - alive か unknown は `*DirectorAttachedError` に `Check` を付けて断る。何も書かない
  - ended なら `cleanUpEndedDirector`
- `cleanUpEndedDirector` が打つのは、確かめが読んだ版 A の登録(`chk.Registration`)だけで、登録を読み直して閉じる先を決めない
  - herdr: `closeHerdrDirector(reg)`。`StopDirector` の本体から取り出した関数で、自分の pane と workspace を閉じる断り、label の照合、C-c、pane と workspace の close を打つ。`StopDirector` の挙動は変えない(エラーの文の `org: director: stop <id>: ` の前置きだけ呼び出し側が付ける)
  - launch_cmd: `stopDirectorProcess(reg)`。pid がない(`starting`)か gone なら何も送らない。alive で起動時刻が記録と同じときだけ SIGTERM を送り、SIGKILL は送らない。director は終わっているので `RALPH_DIRECTOR_TOKEN` の拒否はかけない
  - 最後に `MarkStoppedAtVersion(reg, chk.version, who)` で記録する。`director.lock` の下で、ほかの何より先に `directorVersionOf` を版 A と比べ、違えば何も書かず `ErrDirectorChangedSinceCheck` で断る。同じなら、`MarkStoppedIfUnchanged` と同じ 2 行(`detached`、`stopped`)を 1 回の write で書く(`TestMarkStoppedAtVersion_UnchangedWritesWhatMarkStoppedIfUnchangedWrites` が同じ byte を固定する)
- 記録が書けたら `p.Notice` に 1 行(`cleaned up d1, which had ended (agent_ended): ctrl_c=ok, pane=closed, workspace=closed, and recorded it stopped`)を出して進む。そのあと watch gate が断っても、この 1 行は残る。記録まで届かなければ `*DirectorCleanupError`(verb `start`)で、`stop --id` を言って断る。`ErrDirectorChangedSinceCheck` のときは、start をやり直すか、まだ開いていて終わっているなら stop で記録するよう言う
- fe773bfa: 確かめと片付けのあいだで登録を読み直す変異は、seam がなく mutation で捕まらなかった。`directorStartAfterCheck` と、片付けが版 A だけに打つことのテストを足した。読み直す変異 2 つが赤になる
- 片付けは人が `director start` を打ったときだけ走る。見張りは閉じない(consult の判定。生きている director を誤って外さないため)

## 6. attach の断りと表示(`internal/cli/org_director_check.go`、`org_director.go`、`org_director_headless.go`、`status.go`、S3、S4 の 0b9b7ed3)

- attach(`--replace` なし)の断り: 現在の登録が headless のとき、`attachHeadlessChecks` が文を足す。印が今の登録のもの(`DirectorAliveFor`)なら `the watch checked d1 at <時刻>: <state> (<how>)`。`starting` で starter があれば `its start (pid N), checked just now: ...`(`CheckDirectorStarter`。ps だけで herdr は打たない)。どちらかが ended なら、start が片付けてから立てると案内する。attach は何も閉じず、何も書かない。昇格したセッションの断りの文は変わらない
- start の断りには `start checked d1 just now: alive (agent_found)` が付く(`startAttachedError`)。片付けの 1 行は stdout の先頭に出る(`DirectorStartParams.Notice` に `cmd.OutOrStdout()` を渡す)。`startError` は `ErrDirectorChangedSinceCheck` のとき `ralph org director status` を言う
- 表示: `readOrgDirectorCheck` が印を読み、`orgDirectorLine` の headless の行に `checked by the watch at <時刻>: alive (<how>)`、`ended (...)`、`unknown (<how>: <detail>)` のどれかを出す。印がない、または id か `attached_at` が違うときは `not checked yet: ralph org watch --all checks it each cycle`(印が読めなければ理由を足す)。ended なら start と stop の 2 つの出口を言い、`[org.director].enabled = false` のときは start が断られるので stop だけを言う。昇格したセッションと、stopped のまま current の登録は、前の「the registration does not tell whether its session is still running」の文のまま。`ralph status` と `director status` は同じ `orgDirectorLine` を通る
- `--json`: `director.check`(`orgDirectorReportCheck`: state、how、checked_at、detail の 4 キー。印がなければ省く)。headless の登録の JSON には `starter_pid` と `starter_start_time` が載る。同じ field を持つ型が 3 つ(`org.DirectorCheck`、`org.DirectorAlive`、`orgDirectorReportCheck`)あるので、doc でどれがどの JSON かを言い分けた(self-review L2)
- 印の値は text では `inboxField`(`PrintableInboxText`)を通す。JSON は登録の値と同じく生のまま

## 7. 文書(S5 5e76f820、S5b 84ab7d1e、622c204e)

- `/director` skill の 4 面(同一): 「確かめを読む」の節に how の分類(alive 3、ended 4、unknown 9。`DirectorCheckHow*` と同数)を置き、start の片付けと、見張りが送る 1 行の ended の扱いを足した。`/org` skill の 4 面は `watch` と `director` の行(S5b)
- recipe: `director-launch-cmd.md`(start の片付けの 3 つの場合、check-in を断る条件)と `director-wake.md`(ended には送らない)を、root と `templates/base/` の 2 面とも(S5b)
- README の 5 行(動詞の表、`ralph status` の行、watch の段落、director の段落、wake の段落)、`AGENTS.md` の `internal/org/` の行(`director_check.go` と `director_start_cleanup.go`)、仕様の注 2 つ(FR-9 の 8b-2a の注、FR-10 の 8b-1 の注に 8b-2 を 2 本に分けたこと)。help は attach、start、launched、`director status`、`ralph status`、`org watch`
- tech-debt の 10 行
  - 新しい行 3: codex 0.160.1 の app-server の daemon を stop も start の片付けも止めないこと、確かめが別の herdr server を見ると ended と読むこと、`director_start.go` が 1203 行で 800 行を超え、テスト 2 つも超えること(self-review R1)
  - 書き換えた行 6: 203(確かめの呼び出しによる遅れ、sync-docs)、209 と 249(きっかけを 8b-2b に)、247(昇格したセッションの分に絞る)、254(start の片付けの案内)、256(解決した分を取り消し線にし、残りの unknown の how を同じ行の Debt に `Still open:` として書いた)。256 の上に RESOLVED の注の行が 1 行入った

## 8. テストと evidence

- 主なテストと固定するもの
  - `director_check_test.go`(org): `TestCheckDirector_StartingRows`、`_HerdrRunningRows`、`_CmdRunningRows`、`_NoAgentGet` が D3 の表を偽の herdr と偽の ps で固定する。`_LaunchedAfterTheReadIsNotReadAsStartEnded`(herdr と launch_cmd)と `_ChangedOnBothTriesIsUnknown` は、差し込みで順を決める。印の読み書きは `TestDirectorAlive_*`、`TestReadDirectorAlive_*`、`TestDirectorAliveFor`
  - `director_start_cleanup_test.go` と `director_start_cleanup_refusal_test.go`(org): ended の片付け(`agent_ended`、launch_cmd、`start_ended`)、alive と unknown の断り、記録まで届かない片付け、重なる 2 つの start(`_AStartThatFinishedDuringTheCheckIsNotCleanedUp`)、片付けが版 A だけに打つこと、`MarkStoppedAtVersion`
  - `director_test.go`、`director_start_test.go`、`director_cmd_test.go`(org): starter の fold と記録、check-in の断りの表、ロックの外で ps を読むこと
  - `watch_all_test.go`、`director_wake_test.go`(org): サイクルの順、確かめが印だけを書くこと(3 サイクルで `director.jsonl` が byte 単位で変わらず、herdr は get だけ)、ended に 1 行を送らず stderr に 1 回だけ書くこと
  - `org_director_cleanup_test.go`、`org_director_check_line_test.go`(cli): start の片付けと断り、attach の断りの文、`director:` の行と `--json`(印なし、別の id、別の `attached_at`、alive、ended、unknown、制御文字、壊れた印)
- テストは本物の herdr、osascript、端末、Orca、tmux、gh、network を打たない。ps は自分の pid と、テストが起こして待った子にだけ読み、signal はテストが起こした sleep にだけ送る。/test は herdr、gh、tmux、orca、osascript、open などの canary を PATH の先頭に置いて新しいテスト 67 本を流し、呼び出しは 0 回だった。stub を外す overlay では 5 回呼ばれ、canary が効くことを確かめた
- mutation(`docs/evidence/test-2026-10-19-org-director-liveness.md`): 31 個すべてが赤。版の読み直しを消す M01 は 7 本が落とす。片付けを読み直した登録に打つ M03、確かめを起こす手順のあとに置く M04、check-in が終わった starter を受け付ける M07、表示の印の照合を id だけにする M12 と C01 も赤。1 本だけが落とす mutant は M02(読み直しのロックを外す)と M26(`folded` を空にする)。M02 は 7ef22637 で、本物の flock を持たせたテスト `TestCheckDirector_TheReadUnderTheLockWaitsForAWriterThatHoldsIt` を足した。M26 は white-box の `TestDirectorVersionOf` だけが捕まえ、INFO として残した
- 実機(`docs/evidence/director-liveness-livefire-2026-10-18.md`): 分離した herdr server と隔離した台帳で、codex 0.160.1 の headless director を使った。binary は 0b9b7ed3。`$HOME` は `~`、token と pane の id と start の pid は伏せてある
  - (a) `/quit` で agent を終わらせた: 見張りの次のサイクルで `ended (agent_ended)`。codex が folder trust の dialog と最初の手番にいたあいだは `alive (agent_found)` だった。attach は印の結果つきで断られ、何も書かなかった。次の start が片付けの 1 行を出して d2 を立てた
  - (b) `pane_created` のあとで start を SIGKILL した: 見張りが `ended (start_ended)`。次の start が d3 を片付けて d4 を立てた。pane の中で agent は始まっていなかった
  - 生きている d4 への start は `start checked d4 just now: alive (agent_found)` を添えて断り、`director.jsonl` に何も書かなかった。見張りは `director.jsonl` に書かず、`watch.log` は起動の 1 行だけだった
  - 見つけたこと: codex の app-server の daemon が、turn の `ralph org wait --inbox` を、director を止めたあとも 5 分の時間切れまで抱えていた。stop も start の片付けも止めない。tech-debt の daemon の行に書いた

## 9. 後工程の 1 回(cycle 1)

- 実装: S1 d48c0d23、S2a 8f608232、S2b 0756a73a、S3 3ece3deb、S4 fe773bfa と 0b9b7ed3、S5 5e76f820、S6(実機)fc215a2b、S5b 84ab7d1e。a6da6a3a は S4 の報告にあった doc の食い違いの修正
- self-review(4d47285f): Merge: yes、LOW 4。L1 は check-in の ps がロックを待つ側の 5 秒より長いこと、L2 は同じ field の型 3 つと効かない JSON の説明、L3 は 800 行を超えたファイル、L4 は stub herdr の helper が前提を確かめないこと。L1・L2・L4 を e85853c5、L3 を 80dd2982 で直した(`org_director_check.go` と `director_start_cleanup.go` を作り、`org_director.go` は 945 から 793 行(sync-docs の 1 行で今は 794 行)、`director_stop.go` は 871 から 782 行)。再レビュー(68ed6d2f)は、移動を行の多重集合の突き合わせで純粋な移動と確かめ、LOW 1(R1: `director_start.go` が 1203 行のまま)を tech-debt の 1 行にした(3f5e8409)
- verify(1171515f、pass): LOW 1(V1、計画の Progress の S5 の行の文言)と INFO 3(V2 チェックボックス、V3 D2 の本文、V4 tech-debt の行番号のずれ)。7e43f062 で V1 を直し、AC1〜AC9 に印を付けた。D2 の本文と、計画が名指す行番号は承認済みの本文なので変えず、逸脱は Progress と仕様の注に置いた
- test(eb16551a、pass): 新しいテスト 67 本、mutant 31 個がすべて赤、`-race` で DATA RACE 0。go1.25.8 の全体の回で `TestRegate_FetchFailures` が 1 件落ちた(`regate*.go` はこの差分で変わっていない。単独の `-count=10` は 10 回とも PASS、org パッケージの再実行も PASS)。LOW 2(start の確かめで台帳が読めない道と、launch_cmd の片付けで終わりを確かめられない道が通らない。確かめの読み直しのロックの mutant M02 が 1 本でしか捕まらない)は、テストだけのコミット 7ef22637 の 3 本で埋めた
- sync-docs(622c204e): ずれは 3 点。`ralph org director --help` のグループの help が、昇格したセッションと headless の director を分けずに「Nor does it mean the session is still running」と書いていた(計画の Verify plan の探し文言では当たらなかった)。tech-debt の 203 行に確かめの呼び出しがなかった。file size の行が `driver/herdr_test.go` の 852 行を挙げていなかった。help は 1 文と pin を直し、tech-debt は 2 行を直した
- cross-review(2da83a76): codex(`gpt-6-astra`、reasoning effort `xhigh`、read-only、`--ignore-rules`、rc 0)の指摘は 0 件。codex は read-only の sandbox でビルド用の dir を作れず、テストを走らせていない。テストは /test、7ef22637 のあとの確認、sync-docs の `go test ./internal/cli` の結果による
- 7ef22637 と 622c204e は /test のあとに入った。self-review、verify、test はこの 2 つのあとで走り直していない。7ef22637 はテストだけ(`director_check_lock_test.go` と `director_start_cleanup_refusal_test.go`、+252)。622c204e のコードの変更は `ralph org director --help` の 1 文とそのテストの pin(`TestOrgDirectorHelp_DescribesTheRegistration`)だけで、残りは tech-debt の 2 行と sync-docs の report である。sync-docs と cross-review は、この 2 つを含む範囲を読んだ(cross-review の HEAD は 7e1b6718)。最終の HEAD での `./scripts/run-verify.sh` の結果は PR の Verification を見る

## 10. 試していないことと、先に読む場所

- 試していないこと
  - launch_cmd の director の確かめ(`process_ended`、`pid_reused`)と、遅れた check-in の断り。実機はなく、テストが自分で起こしたプロセスで見ている
  - Claude Code の director
  - `wake = ["push"]` の登録が ended のときに送らないこと(D5)。実機は pull で、`director_wake_test.go` のテストだけである
  - 2 つの start の重なり、見張りと start の同時。差し込みの関数で順を決めるテストだけで、本当に同時に走らせるテストはない
  - unknown の各結果(`agent get` の失敗、label の違う pane、ps の失敗)。偽の herdr と偽の ps のテストだけである
  - Linux(procps)の `ps -o stat=,lstart=`。手元で読んでいない。本物の ps を読むテストは、ps のない環境では落ちる。`deadChildPID` は pid が使い回されないことを前提にしており、おそらく起きないが未確認
  - 実機の binary は 0b9b7ed3 で、HEAD の binary では打っていない。そのあとの非テストの Go の変更は、e85853c5 の check-in の starter の読み(launch_cmd で、実機の範囲の外)、80dd2982 のファイルの移動(行の突き合わせで純粋な移動と確かめた)、a6da6a3a と 622c204e のコメントと help の文である
  - 見張りと director と start が別の herdr server を見ている場合は、`agent_not_found` を ended と読み、start が何も閉じずに stopped を記録して 2 つ目の director を立てうる。tech-debt に書いた(座席の pulse と stop も同じ前提)
  - 知られた揺れ `TestRegate_FetchFailures` は、go1.25.8 の全体の回で 1 回落ち、単独では 10 回とも PASS した。この差分の外である
- 先に読む場所
  - `internal/org/director_check.go` の `CheckDirector`、`directorVersionOf`、`readDirectorVersion`: 確かめの結果を登録の版に結びつける規則
  - `internal/org/director_start_cleanup.go` の `checkCurrentDirector`、`cleanUpEndedDirector` と、`director_start.go` の `MarkStoppedAtVersion`: 片付けが版 A の登録だけに打つこと
  - `internal/org/director_start.go` の `MarkCmdLaunched`: ロックの外の ps と、ロックの下の starter の照合(D2 からの逸脱)
  - `internal/org/watch.go` の `cycle` と `checkDirector`、`director_wake.go` の `directorCheckFoundEnded`
  - `.claude/skills/director/SKILL.md` の「確かめを読む」: director が ended や unknown を見たときの手順
