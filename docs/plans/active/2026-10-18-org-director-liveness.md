# org-director-liveness

- Status: Approved
- Approved: 2026-10-18 sha256:d9e4345287da
- Owner: Claude Code
- Date: 2026-10-18
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 8 段目の後半の 2 本目(8b-2)の前半(8b-2a)。headless の director の生死の確かめと、落ちた start が残す `starting` の登録の片付け(tech-debt の 255 行と、247 行の headless の分)。8b-1 は PR #236 で merge 済み(main は 88e37b1d)。8b-2 の分け方と、見張りは印だけを書き片付けは start に寄せる形は consult(consult-plan-liveness、2026-10-18)の判定による。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-director-liveness

## Objective

headless の director(`ralph org director start` が立てたもの)が死んだとき、それが分かり、次の `director start` が人の手を待たずに片付けて進めるようにする。横断の見張り(`ralph org watch --all`)は各サイクルで今の登録の生死を確かめて印のファイルに書き、`ralph status` と `ralph org director status` はそれを出す。`director start` は今の登録が headless で、その場で確かめて終わっていれば、`stop` と同じ片付けをしてから立てる。start は自分の pid と起動時刻を登録に書き、start が死んだあとの `starting` は「start が終わった」と読む。遅れて来た launch_cmd の check-in は断る。見張りは director.jsonl を書かず、何も閉じない。

## Scope

- 登録: headless の `attached` に start のプロセスの pid と起動時刻(`starter_pid`、`starter_start_time`)を書く。fold がそれを登録に載せる
- check-in: start のプロセスが終わっていれば `ralph org director launched` を断る
- 確かめ(新しい `internal/org/director_check.go`): 今の headless の登録が生きているかを、herdr の director は `herdr agent get` と pane の label の照合で、launch_cmd の director は記録した pid で、`starting` は start のプロセスで読む。結果は alive・ended・unknown と、どの確かめで分かったか
- 見張り: サイクルの順を「受信箱の期限 → director の確かめ → 起こす → パルス → 生存の印」にし、確かめの結果を `director-alive.json` に上書きで書く。ended と読んだ登録には、そのサイクルで起こす 1 行を送らない
- start: 今の登録が headless で、その場の確かめが ended なら、`StopDirector` と同じ片付けを確かめた版の登録に打ってから進む。alive か unknown なら今どおり断り、確かめの結果を文に足す
- attach(`--replace` なし)の断り: 印と start のプロセスの確かめの結果を文に足す。片付けはしない
- 表示: `ralph status` の `director:` の行、`ralph org director status`、どちらの `--json` にも確かめの結果を出す
- 文書: help、README、`/director` skill の 4 面、仕様の注(8b-1 の注と、この段の注)、tech-debt、AGENTS.md
- 実機の確認: 分離した herdr server で、codex の headless director の agent を終わらせる場合と、start を途中で kill する場合

## Non-goals

- 昇格したセッション(`/director` で attach した対話のセッション)の生存の印と、見張りの停止に director が気づく仕組み(8b-2b。247 行の昇格したセッションの分と 209 行の半分)。8b-2b は、Claude Code の background の wait がセッションの終了で止まるかの実験のあとで計画する
- 見張りが登録を stopped にすること、見張りが pane や workspace を閉じること(consult の判定。生きている director を誤って外さないため)
- attach が片付けること(`--replace` がすでに逃げ道として通る)
- 起動時刻が記録と違う pid を「終わった」とみなす自動の片付け(pid の再利用か、Linux の時計の変更か見分けられないので unknown にする。tech-debt 270・271 行)
- start が workspace create と記録のあいだで死んだときの、記録のない workspace(254 行)を閉じること
- codex 0.160.1 の app-server の daemon(TUI が終わっても残る)を止めること
- 9 段目の試行

## Assumptions

- herdr 0.7.5 では、agent が `/quit` で終わったときも SIGKILL で落ちたときも、`herdr agent get director_<token>` は 1 秒以内に `agent_not_found` を返し、pane は shell に戻って残る。pane を閉じると pane も tab も `not_found` になり、workspace は残る。25 秒の前景のコマンドの最中は 18 回とも見つかった(2026-10-18 の分離した server での確かめ。実機の確認の evidence に写す)
- 見張りと director と start は同じ herdr server を見ている。座席のパルス(`agent get`)と今の `stop --id`(pane_not_found を閉じたとみなす)も同じ前提に立つ
- 見張りは台帳ごとに 1 つ(`watch.lock`)なので、`director-alive.json` の書き手は 1 つになる
- `ps -o stat=,lstart= -p <pid>` は macOS と procps で読める(`ReadDirectorProcess`。busybox は対象外)

## Affected areas

- `internal/org/director.go`(`DirectorEvent` と `DirectorRegistration` の starter の field、fold)
- `internal/org/director_start.go`(`AttachHeadless` が starter を書く、`MarkCmdLaunched` の断り、`checkDirectorStart` の片付け)
- `internal/org/director_check.go`(新規。確かめ、`director-alive.json` の読み書き)
- `internal/org/director_cmd.go`(`ReadDirectorProcess` を確かめから使う。変えるのは必要なら公開の範囲だけ)
- `internal/org/director_stop.go`(start の片付けのために、登録を読み直さず D3 の版 A の登録で閉じて記録する入口を足す。`StopDirector` と `stop --id` の挙動は変えない)
- `internal/org/director_wake.go`(確かめが ended と読んだ登録に送らない)
- `internal/org/watch.go`(サイクルの順)
- `internal/cli/org_director.go`(attach の断りの文、`director status` の表示、`orgDirectorLine`)、`internal/cli/org_director_headless.go`(start の出力と help)、`internal/cli/status.go`(help、`--json`)
- テスト: `internal/org/director_check_test.go`(新規)、`director_test.go`、`director_start_test.go`、`director_cmd_test.go`、`director_wake_test.go`、`watch_all_test.go`、`internal/cli/org_director_test.go`、`org_director_headless_test.go`、`status_test.go`
- 文書: `README.md`、`.claude/skills/director/SKILL.md` と写しの 3 面、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/tech-debt/README.md`、`AGENTS.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-director-liveness.html`(かんたん版 2 枚: overview、easy-who。折りたたみに detail-overview、detail-cycle、detail-check、detail-start)
- 自己点検: `plan-visual.sh lint` が exit 0。全体と 6 枚の図を撮って読んだ。detail-overview で「書く」の線が director_stop.go の箱の文字を横切り、detail-start で alive・unknown の曲線が片付けの箱を横切り、その箱の文字が右の矢印に重なっていたので、箱を動かして線をまっすぐにし、箱を広げ、撮り直して確かめた
- Codex の計画助言 1 件(HIGH、2 つの start が重なると立ったばかりの director を止めうる)を D3・D4・D6・AC3・AC6 に入れたあと、detail-start に「結果は版 A に結ぶ」「版 A の pane・workspace・pid だけ」「版 A と比べて記録」「版が変わったら断る」を足し、detail-check と detail-cycle の注に読み直しと `changed` を足した。lint は exit 0、detail-start を撮り直して読み、重なりがないことを確かめた

## Design decisions

- D1 start のプロセスを記録する: `AttachHeadless` は、start のプロセス(`os.Getpid()`)の pid と、`ReadDirectorProcess` で読んだその起動時刻を、`attached` の `starter_pid` と `starter_start_time` に書く。起動時刻を読めなければ両方とも書かず、start は今どおり進む(その登録の `starting` は D3 で unknown になる)。古い行(field がない)も同じく unknown
- D2 check-in の断り: `MarkCmdLaunched` は、ロックの下で登録の starter を `ReadDirectorProcess` で読み、gone、zombie、起動時刻の違いのどれかなら `ErrDirectorCheckInRefused` で断る(理由は「start のプロセスが終わった」)。ps が読めなければ今どおり受け付ける。起動スクリプトは check-in が失敗すると `exit 1` で終わり、agent を exec しないので、遅れて来た agent は立たない
- D3 確かめ(`CheckDirector`、見張りと start が共有する): 対象は今の登録で、headless で stopped でないもの。昇格したセッションの登録は確かめない
  - `starting`(herdr でも launch_cmd でも): starter が記録されていなければ unknown(`no_starter`)。gone か zombie なら ended(`start_ended`)。起動時刻が違えば unknown(`starter_pid_reused`: pid の再利用か時計の変更)。生きていれば alive(`start_running`)。ps が読めなければ unknown
  - herdr の `running`: `herdr agent get director_<token>` が成功すれば alive(`agent_found`)。`agent_not_found` なら、記録した pane を stop と同じ label の照合(`confirmSeatPane`)で読み、label の合う pane があれば ended(`agent_ended`)、pane がなければ ended(`pane_closed`)、label の不一致や get の失敗は unknown。`agent get` のほかの失敗(時間切れ、server がない)は unknown。各呼び出しは `DriverCallTimeout`(10 秒)で区切る
  - launch_cmd の `running`: 記録した pid を `ReadDirectorProcess` で読み、gone か zombie なら ended(`process_ended`)、起動時刻が違えば unknown(`pid_reused`)、生きていれば alive(`process_found`)、ps が読めなければ unknown
  - 結果は state(alive・ended・unknown)、how(上の括弧の語)、checked_at、detail(unknown の理由の短い文)を持つ
  - 結果は、確かめに使った登録の版に結びつける(Codex の計画助言 HIGH)。確かめは登録を読み(版 A)、ps と herdr を読み、そのあと `director.lock` の下で登録を読み直す。その登録の event が 1 つでも増えていれば(`launched`、`workspace_created`、`pane_created`、`stopped`、`detached` など)、結果を捨てて確かめ直し、2 回続けて変わっていれば unknown(`changed`)にする。たとえば、後の start が `starting` を読んだあと、先の start が `launched` を書いて正しく終わったなら、start のプロセスが消えていても `start_ended` にはならない。結果は版 A の登録とともに返す
- D4 見張りの印: 見張りは各サイクル、受信箱の期限のあと、起こす手順の前に、今の登録を D3 で確かめて `director-alive.json`(台帳の下、一時ファイルと rename で上書き)に `id` と D3 の結果を書く。D3 の読み直しのため、見張りも `director.lock` を短く取る(読むだけで、書かない)。今の headless の登録がなければ書かない(読む側は id が今の登録と違う印を無視する)。director.jsonl は読むだけで書かない。確かめに失敗しても、そのサイクルのほかの手順は進む(stderr に 1 行)
- D5 起こす手順: そのサイクルの確かめが ended と読んだ登録には、起こす 1 行を送らず、stderr に登録ごとに 1 回だけ書く。unknown なら今どおり送る(8b-1 の label の照合が送り先を守る)
- D6 start の片付け: `checkDirectorStart` の最初の確かめ(今の登録を見る位置。見張りの検査より前)で、今の登録が headless なら、印ではなくその場で D3 を打つ
  - ended なら `StopDirector`(`--force` なし)と同じ片付けをする。herdr は label の照合のあとで pane と workspace を閉じ、launch_cmd は pid がないか終わっているので何も送らない。stopped を記録できたら、片付けたことを 1 行出して進む(進んだ先で start がほかの理由で断られても、片付けた事実は 1 行残る)。片付けが記録まで届かなければ、stop のエラーと `stop --id` を言って断る
  - 片付けは D3 の版 A の登録で打つ。stop のように登録を読み直して閉じる先を決めず、閉じる pane と workspace、送る先の pid は版 A のものに限る。最後の記録(`MarkStoppedIfUnchanged`)も版 A と比べ、版 A のあとに event が増えていれば記録せずに断る。版 A で ended と分かった登録が閉じるまでのあいだに生き返る道はない。`starting` は start のプロセスが消えたことを確かめたあとで版が変わっていないので、`launched` を書ける者が残っていない(herdr は start だけが書き、launch_cmd の check-in は D2 で断られる)。`running` の ended(agent が見つからない、pid が消えた)は戻らない
  - alive か unknown なら今どおり断り(`DirectorAttachedError`)、確かめの結果を文に足す
  - 今の登録が昇格したセッションなら今どおり断る
  - 2 つの start が同時に片付けても、`MarkStoppedIfUnchanged` と `AttachHeadless` のロックの下の判定が 1 つに決める
- D7 attach の断り: `--replace` なしの attach が今の登録で断られるとき、印の id が今の登録と同じなら印の結果(時刻つき)を、登録が `starting` で starter が記録されていれば start のプロセスの確かめの結果を、文に足す。ended なら「`ralph org director start` が片付けてから立てる、または `stop --id`、または `attach --replace`」を案内する。attach は herdr を打たず、片付けもしない
- D8 表示: `ralph status` の `director:` の行と `ralph org director status` は、headless の登録で印の id が今の登録と同じなら「checked by the watch at <時刻>: alive / ended (<how>) / unknown (<detail>)」を出し、ended なら片付けの案内を足す。印がない、または id が違えば「not checked yet」と出す。昇格したセッションの文(登録はセッションの生死を表さない)は変えない。`--json` は `director.check`(state、how、checked_at、detail)を足す。印の値は今の表示と同じく escape する
- D9 実機: 分離した herdr server と隔離した台帳で、(a) codex の headless director の agent を `/quit` で終わらせ、見張りの次のサイクルで status が ended を出し、次の `director start` が片付けて d2 を立てる、(b) `director start` を agent の起動中に kill し、status が `starting` の ended(`start_ended`)を出し、次の start が片付けて立てる、を確かめる

Critical forks: 8b-2 の分け方(8b-2a = この計画、8b-2b = 昇格したセッションと見張りの停止)、見張りが記録するか印だけを書くか、片付けをどこに置くか、attach に片付けを入れるか、を consult(consult-plan-liveness)に聞いた。判定は「印だけを書く B' を採る、片付けは start、attach は文だけ」で、D4・D6・D7 に入れた。起動時刻の違いを unknown にすること(D3)、確かめを起こす手順の前に置いて ended に送らないこと(D5、consult のリスク 2)は、ユーザーの指示「計画の分かれ目も自分で選ぶ」に従って自分で選んだ。そのあと Codex の計画助言が「2 つの start が重なると、後の start が `starting` を読んだあとで先の start が `running` にして終わり、後の start は start のプロセスが消えたのを見て ended と読み、立ったばかりの director を止める。stop は登録を読み直して閉じる先を決め、記録の比較は閉じたあとなので防げない」と指摘した(HIGH)。ユーザーの指示どおり直し、確かめの結果を登録の版に結びつけ(D3)、片付けは版 A の登録だけに打ち、記録も版 A と比べる形にした(D6、AC3、AC6)。

## Acceptance criteria

- [ ] AC1 starter: `director start`(herdr と launch_cmd のどちらも)が書く `attached` に `starter_pid`(start のプロセスの pid)と `starter_start_time` が載る。起動時刻を読めないときは両方とも載らず、start は進む。fold は field のない古い行を今どおり読む
- [ ] AC2 check-in: 登録の starter が gone、zombie、起動時刻の違いのどれかなら、`ralph org director launched` は断り(終了コード 1)、`launched` を書かない。starter が生きているか、ps が読めないか、starter が記録されていなければ今どおり受け付ける
- [ ] AC3 確かめ: D3 の表のすべての行(`no_starter`、`start_ended`、`starter_pid_reused`、`start_running`、`agent_found`、`agent_ended`、`pane_closed`、label の不一致、`agent get` の失敗、`process_ended`、`pid_reused`、`process_found`、ps の失敗、`changed`)が、偽の herdr と偽の ps で、決めた state と how を返す。昇格したセッションと stopped の登録は確かめない。確かめの途中で登録に event が増えたとき(読んだあと、ps を読む前に `launched` が書かれ、start のプロセスが終わる)、結果は `start_ended` にならない(herdr と launch_cmd の両方、テストの中の差し込みで順を決める)
- [ ] AC4 見張り: 1 サイクルの順が「受信箱の期限 → 確かめ → 起こす → パルス → 生存の印」になり、確かめの結果が `director-alive.json` に id つきで書かれる。今の headless の登録がなければ書かない。見張りは director.jsonl に何も書かず、herdr の close も send-keys も打たない(確かめの分。起こす手順の send-text と Enter は 8b-1 のまま)。確かめの失敗は 1 行で、そのサイクルの起こす手順とパルスは進む
- [ ] AC5 起こす手順: 同じサイクルの確かめが ended なら、push の登録にも 1 行を送らず `woken` を書かない。stderr には登録ごとに 1 回だけ書く。unknown なら今どおり送る
- [ ] AC6 start の片付け: 今の登録が headless で ended なら、start は stop と同じ片付けをして stopped を記録し、片付けたことを 1 行出してから新しい登録を立てる。herdr の登録では label の合う pane と workspace だけを閉じる。alive か unknown なら今どおり断って何も書かず、文に確かめの結果が入る。昇格したセッションの登録では今どおり断る。片付けが記録まで届かなければ、新しい登録を書かずに断り、`stop --id` を言う。2 つの start が重なり、後の start が `starting` を読んだあとで先の start が `running` にして終わったとき、後の start は signal も close も送らず、新しい登録も書かずに断る(herdr と launch_cmd の両方、差し込みで順を決めるテスト)。片付けが閉じる pane・workspace と signal を送る pid は版 A のものだけ
- [ ] AC7 attach の断りと表示: `--replace` なしの attach の断りの文に、印の結果(id が同じとき)と start のプロセスの確かめ(`starting` のとき)が入る。`ralph status` の `director:` の行、`ralph org director status`、両方の `--json` に確かめの結果が出る(印がないとき、id が違うとき、alive、ended、unknown の各行)。昇格したセッションの文は変わらない
- [ ] AC8 文書: `/director` skill の 4 面が同じ中身で、headless の director の確かめの読み方を書く。`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る。help、README、仕様の注(8b-1 の注に 8b-2 を 2 本に分けたこと、この段の注)、tech-debt(255 行を閉じるか残りに書き換え、247 行を昇格したセッションの分に絞る、209 行のきっかけを 8b-2b に、254 行に start の片付けの案内、codex の daemon と別の herdr server の行を足す)、AGENTS.md が新しい動きを説明する
- [ ] AC9 テストは本物の herdr、osascript、端末、Orca、人の tmux、gh、network を打たず、repo の台帳に書かない。signal は自分で起こしたプロセスにだけ送る。実機の確認(D9 の (a) と (b))を分離した herdr server で行い、evidence に残す(`$HOME` は `~`、token と pane の値は伏せる)

## Implementation outline

1. S1 登録と check-in: starter の field、`AttachHeadless`、fold、`MarkCmdLaunched` の断り(AC1、AC2)
2. S2 確かめと印: `director_check.go`(`CheckDirector`、印の読み書き)、見張りのサイクル、起こす手順の飛ばし(AC3〜AC5)
3. S3 start と attach: start の片付け、attach の断りの文、start の出力と help(AC6、AC7 の attach の分)
4. S4 表示: `orgDirectorLine`、`director status`、`--json`(AC7 の表示の分)
5. S5 文書(AC8)
6. S6 実機の確認と evidence(AC9)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、golangci-lint、secret scan)、`go vet ./...`、`GOTOOLCHAIN=go1.25.8` の build と vet、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: ユーザーストーリー 3(director が止まっても期限で人に届く)が変わらないこと(確かめは受信箱の期限を読まず、期限の送りはサイクルの最初のまま)。FR-9 の headless の director の start と stop が今どおり動くこと。FR-10 の起こし口が ended の登録に送らないこと(D5)。8b-1 の注が 8b-2 を 2 本に分けたことを書き、この段の注が D3〜D8 と非ゴールを書くこと
- Documentation drift to check: 「registration does not tell whether its session is still running」が headless の director について残っていないか(昇格したセッションには残す)。tech-debt の 255 行と 247 行が、この段で閉じた分と 8b-2b に残す分を書き分けているか。start の help が片付けを書いているか
- Evidence to capture: 実機の確認で打ったコマンドと出力、director.jsonl と `director-alive.json` の行、`ralph status` の `director:` の行。`$HOME` は `~`、token と pane の値は伏せる

## Test plan

- Unit tests:
  - D3 の確かめの表(偽の herdr の `AgentGet`・`PaneGet`・`TabGet`・`WorkspaceGet`、偽の `DirectorProcessReader`)
  - 印の書き込みと読み取り(id、壊れた印、ない印、一時ファイルと rename)
  - starter の field の fold(新しい行、古い行)
  - check-in の断りの表(gone、zombie、起動時刻の違い、生きている、ps の失敗、starter なし)
- Integration tests:
  - 見張りのサイクルの順と印(偽の herdr。確かめの呼び出しが起こす手順の前に来る)
  - ended の登録に起こす 1 行を送らないこと、stderr の 1 回だけの行
  - start の片付け(herdr の agent_ended、pane_closed、launch_cmd の process_ended、`starting` の start_ended)、alive と unknown の断り、昇格したセッションの断り、片付けが記録まで届かないときの断り
  - attach の断りの文、`ralph status` と `director status` の表示と `--json`
- Regression tests: 8a と 8b-1 のテスト(attach、start、stop、check-in、escalate、期限、起こす手順)がそのまま通る。見張りは確かめのために director.jsonl を書かず、herdr の close を打たない
- Edge cases:
  - 確かめと start が同じ登録を同時に見る(start の片付けのあとで見張りが書いた印は、id が違うので読まれない)
  - 2 つの start の重なり: 後の start の読みと ps のあいだに、先の start が `launched` を書いて終わる(Codex の計画助言 HIGH。差し込みの関数で順を決め、signal、close、新しい登録がないことを確かめる)
  - 見張りの確かめの途中で `launched` が書かれる(印は `start_ended` を書かない)
  - 片付けのあいだに別の start が登録を立てる(ロックの下の判定)
  - 印が壊れている、印の id が古い
  - 見張りが止まっていて印が古い(時刻を出すので読める)
  - `agent get` が成功するのに pane の label が違う(alive のまま。名前は token で一意)
- Evidence to capture: `go test` の出力(go1.26 と go1.25.8)、`-race`、mutation の結果

## Risks and mitigations

- 見張りと director が別の herdr server を見ていると、`agent_not_found` と `pane_not_found` が出て ended と読み、start が何も閉じずに stopped を記録して 2 つ目の director を立てる: 座席のパルスと今の stop も同じ前提に立つ。印と start の出力に how を出し、tech-debt に 1 行足す
- Linux で時計を動かすと、生きているプロセスの起動時刻が変わって読める(271 行): D3 は起動時刻の違いを unknown にし、自動の片付けは gone と zombie と herdr の not_found だけで打つ。check-in(D2)は起動時刻の違いでも断るが、断られた start は自分の片付けで終わるだけ
- 2 つの start が重なると、後の start が、先の start の立てたばかりの director を止めうる(Codex の計画助言 HIGH): 確かめの結果を登録の版に結びつけ、読み直して版が変わっていれば捨てる(D3)。片付けは版 A の登録だけを閉じ、記録も版 A と比べる(D6)。順を決めたテストで確かめる(AC3、AC6)
- start の片付けは、死んだ director の pane の画面を消す: 片付けるのは人が `director start` を打ったときだけで、見張りは閉じない。status が先に ended を出すので、人は先に pane を見られる
- 確かめが見張りのサイクルに herdr の呼び出しを 1 回(agent が見つからなければ 4 回まで)足す: 各 10 秒で区切り、期限の送りより後に置く(tech-debt 203 行の遅れと同じ扱い)
- 古い ralph に戻すと: `attached` の starter の field と `director-alive.json` は読まれずに無視される
- codex 0.160.1 は TUI が終わっても app-server の daemon を残し、「running work continues」と言う: stop も start の片付けもこの daemon を止めない。tech-debt に 1 行足す

## Rollout or rollback notes

- 既定の動きの変化: 見張りが今の headless の登録の確かめを始める(herdr の呼び出しが増える)。`director start` は、死んだ headless の登録を片付けてから立てるようになる(今は断る)。release のノートに書く
- 戻すときは、そのまま revert してよい。前の ralph は starter の field と印を無視し、`starting` の登録は今までどおり stop まで残る

## Open questions

- なし(分かれ目は consult の判定と、Design decisions の自分の選択で決めた)

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved(ユーザーの指示「作成したプランは自動承認します」による自動承認、2026-10-18)
- [x] Branch created(feat/org-director-liveness)
- [x] Implementation started
  - S1 d48c0d23(starter の field と check-in の断り)
  - S2 は 2 つのコミットに分けた。S2a 8f608232(`CheckDirector` と版の結びつけ、driver の `agent_not_found`)、S2b 0756a73a(見張りの確かめ、`director-alive.json`、ended の登録を起こさない)
  - S2b の印は id に加えて `attached_at` も照合する(director.jsonl を消すと id が振り直されるため)。確かめは起こす手順と同じく `[org.director].enabled` を見ない
  - S3 3ece3deb(start の片付け、attach の断りの文)。記録は新しい `MarkStoppedAtVersion` と `ErrDirectorChangedSinceCheck` で版 A と比べる。start の通知は `DirectorStartParams.Notice` に 1 行。attach が start のプロセスを確かめるため `CheckDirectorStarter` を公開した。launch_cmd の片付けは director が終わっているので `RALPH_DIRECTOR_TOKEN` の拒否をかけない。確かめと片付けのあいだで登録を読み直す変異は seam がなく捕まらなかったので、S4 の前のコミットで seam とテストを足す
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
