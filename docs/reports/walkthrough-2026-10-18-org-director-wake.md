# Walkthrough: org-director-wake(8 段目の後半の 1 本目、8b-1)

- Date: 2026-10-18
- Plan: docs/plans/active/2026-10-18-org-director-wake.md(/pr で archive に移す)
- Branch: feat/org-director-wake(base 9e92d928。この walkthrough を書いた時点で 27 commits、60 files、+7939/-328)
- PR: #236
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 50 files、+6433/-328 になる。そのうちテストは `internal/org/director_wake_test.go`(+1391)、`internal/cli/org_director_wake_test.go`(+810)、`internal/config/config_test.go`(+435)、`internal/org/inbox_test.go`(+302)、`watch_all_test.go`(+280)、`internal/cli/org_watch_test.go`(+176)、`org_inbox_test.go`(+112)ほかで、14 files、+3678/-12、足した `func Test` は 72 本。ほかに `/director` と `/org` の skill の 4 面ずつ、recipe `docs/recipes/director-wake.md` の 2 面(各 358 行)、README、`AGENTS.md`、仕様 FR-10 の注、tech-debt の 7 行(新しい行 1、書き換えた行 6)がある
- 後工程は 1 回まわした(`cycle-count.json` は 1、上限は 2)。cross-review は 0 件で、上限には届いていない(下の 11)

新しいファイルは `internal/org/director_wake.go`(576 行)、`internal/cli/org_director_wake.go`(106 行)と、それぞれのテスト、recipe。次の順に読むと、escalate された 1 件が push で director に届くまでを 1 本で追える。

```
escalate → inbox.jsonl に open の件 → 見張りの次のサイクル → 期限の送り
  → WakeDirector: 件を選ぶ → 送り手を選ぶ → 1 行を送る → 件ごとに woken を書く
```

## 1. 設定 `[org.director].wake` と `wake_cmd`(`internal/config/config.go`、S1、2f08cfe7)

- `OrgDirectorConfig.Wake`(文字列の配列、`Default()` は `["pull"]`)と `WakeCmd`(argv、既定は空)。`Load` が `ValidateDirectorWake` と `ValidateDirectorWakeCmd` を呼ぶ
  - wake: 空の配列、`pull` と `push` 以外、重複を断る。順は問わない
  - wake_cmd: 最初の要素が空白だけ、最初の要素が placeholder を持つ、`{target}` と `{text}` 以外の `{...}` を断る。エラーは `[org.director].wake_cmd:` のようにキーで始まる
  - `wake = ["push"]` で wake_cmd が空でも `Load` は通す。送れるかは登録で決まるので、attach と start が確かめる(D2、D4)
- 最初の要素に placeholder を許さない。`{target}` は leader も書ける `director.jsonl` から来るので、登録が打つ実行ファイルを選べないようにする(計画の Codex 助言 HIGH、D2)
- `ValidateDirectorWakeNamed` は attach の `--wake` が同じ規則でフラグ名つきのエラーを出すために分けた。`ExpandDirectorWakeArg` は 1 回の置換で全 placeholder を置くので、値の中の `{...}` は置き直さない
- launch_cmd と部品を共有する。`directorLaunchPlaceholderPattern` を `directorCmdPlaceholderPattern` に改め、`checkDirectorCmdProgram`、`checkDirectorCmdPlaceholders`、`expandDirectorCmdArg` に切り出した。launch_cmd のエラーの文と順は変わらない(cycle 1 の self-review が突き合わせた)
- 既定値の 3 面: Go の `Default()`、`scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` の `RALPH_ORG_DIRECTOR_WAKE`(配列なのでコンマ区切り)、`templates/base/ralph.toml` の `wake = ["pull"]`。`defaults_sync_test.go` の `org.director.wake` が揃いを見る。wake_cmd に環境変数はない。template の wake_cmd の例はコメントで、`TestTemplateRalphToml_WakeCmdExampleIsValid` が読み込めることを見る
- `{target}` は `;`、`$`、バッククォート、空白を通す(tmux の id が `$1`、`%3`、`@2` の形のため)。シェルが読む文字列に入れず、argv の 1 要素にする。この注意は `WakeCmd` の doc、`templates/base/ralph.toml`、recipe の Security にある(self-review M1)。`{target_sh}` の案は tech-debt の 1 行

## 2. 登録(`internal/org/director.go`、`director_start.go`、`director_wake.go`、`internal/cli/org_director*.go`、S2、0d49b2a0)

- `director.jsonl` の `attached` に `wake_target`(omitempty)を足した。`DirectorRegistration.WakeTarget` も同じ。前の行は `wake_target` を持たず、そのまま読める
- 送り先は `ResolvedWakeTarget` が決める。herdr の headless は pane の id(`PaneID`)、launch_cmd の headless は端末の名前 `DirectorAgentName(token)`(`director_<token>`)、昇格したセッションと知らない launch は `WakeTarget`(attach の `--wake-target`)。fold は `wake_target` を昇格したセッションの分しか読まず、headless では空に戻す(`registrationOf`)
- attach: `--wake <pull|push|pull,push>`(省くと設定の `wake`)と `--wake-target`。`newOrgDirectorAttachCmd` の流れは次のとおりで、どの断りも何も書く前に起きる。`--replace` でも今の登録は残る
  1. `checkOrgDirectorAttachWakeFlags`: フラグの形だけ。`--wake` を `,` で割って `ValidateDirectorWakeNamed("--wake", ...)`、`--wake-target` は与えられていれば空でも `ValidateDirectorWakeTarget`
  2. `rt.attachWake`: 設定を読んで wake を決め、`org.CheckDirectorAttachWake` で送れるかを確かめる
  3. `store.Attach`
- `CheckDirectorAttachWake` が断る順: push で wake_cmd が空(`ErrDirectorWakeCmdNeeded`)、push で wake_cmd が `{target}` を含むのに `--wake-target` がない(`ErrDirectorWakeTargetNeeded`)、push がないのに `--wake-target` がある(`ErrDirectorWakeTargetWithoutPush`)、送り先の形(`ValidateDirectorWakeTarget`)。送り先の形は、空でない、有効な UTF-8、制御文字なし、256 文字以下(`MaxDirectorWakeTargetRunes`)、`-` で始まらない
- store の `Attach` も、空でない不正な wake と不正な送り先を断る(CLI の後ろの 2 段目)。空の `Wake` は断らず `["pull"]` を記録する。範囲外のテストが `Wake` なしで `DirectorAttach` を組むための逸脱で、CLI はいつも解決済みの値を渡す(計画の Progress)
- start: `AttachHeadless` が設定の `Wake` を書き、`wake_target` は書かない。`checkDirectorStartWake` は `checkDirectorStart` の入力の検査で、`checkDirectorLaunchCmd`、`director.jsonl` の最初の読み(今の登録の有無)、watch gate より前に走る。launch_cmd の start で push かつ wake_cmd が空なら `ErrDirectorWakeCmdNeeded` で断り、台帳に書かず、launch_cmd も herdr も打たない。herdr の start は wake_cmd が空でも通す(pane が送り先になる)
- 表示: attach の成功行は push のときだけ `wake=push`(と `wake_target=...`)を足す(`orgDirectorAttachWakeDetails`)。pull だけの行は 8a のまま。`director:` の行は push のとき `wake=<wake> to <送り先>`(`orgDirectorPushDetails`)。`director status` は `wake:` の行に `wake_target:` を、値があるときだけ足す。`ralph status --json` と `director status --json` は同じ `orgDirectorReport` を通るので `wake` と `wake_target` が載る。`internal/cli/status.go` の変更はコメントだけである。値は `inboxField` で escape する

## 3. 設定を読む場所と wake_cmd を打つ場所(`internal/cli/org.go`、`org_watch.go`、S2、S3b)

- 規則は launch_cmd と同じ。`--config` がなく、台帳が main worktree のものなら、main worktree の `ralph.toml` から読む。leader が feature worktree の `ralph.toml` を書き換えても、attach、start、`watch --all` の使う値は変わらない(AC4)
- 読み口は 3 つ
  - attach: `withMainWorktreeDirectorWakeConfig`(`rt.attachWake` が呼ぶ)
  - start: 既存の `withMainWorktreeDirectorStartConfig` が `Wake` と `WakeCmd` も取る
  - watch: `orgWatchDirectorWake` が `withMainWorktreeDirectorWakeConfigFrom` を呼ぶ。この関数は読んだ `ralph.toml` のパスも返す
- `ralph status` と `director status` は wake と wake_cmd の設定を読まず、登録に記録された wake を出す。設定の `wake` を変えても今の登録には効かず、attach か start をし直す(D3)
- wake_cmd を打つ場所は、それを読んだ `ralph.toml` のディレクトリである。`orgWatchDirectorWake` が返す
  - main の規則で読んだなら main worktree のルート
  - `--config` ならそのファイルの絶対パスのディレクトリ
  - どちらでもなければ見張りを起動したときの作業ディレクトリ
- この値は `WatchAllParams.DirectorWakeDir`、`watchAllRun.wakeDir`、`WakeDirector` の `wakeCmdDir`、`runDirectorCmd` の `dir` と渡る。feature worktree から前景で立てた見張りでも、feature 側の同名のスクリプトは打たない(計画の Codex 助言 HIGH への対応)
- 最初の要素が `/` を含む相対パスなら、`runDirectorWakeCmd` が上のディレクトリから解く。`/` を含まない名前は PATH で探す。ディレクトリが空か相対なら、何も打たずに失敗する
- `--detach` では、親が先に `orgWatchDirectorWake` を呼ぶ。main の `ralph.toml` が読めなければ、watch.log ではなく親のエラーになる。子は自分でディレクトリを決める
- 見張りは起動時に wake_cmd を読むので、変えたら見張りを起動し直す。help と recipe に書いた

## 4. 見張りのサイクル(`internal/org/watch.go`、S3b、b716d041)

- `watchAllRun.cycle()` の順は、期限の送り(`SweepInboxDeadlines`)、起こす(`wake()`)、各 org の pulse、生存の印の書き換え
- 期限の送りを先に置く。起こす手順が wake_cmd や herdr で詰まっても、そのサイクルの人への送りは遅れない(D6)。遅れるのは pulse と印で、tech-debt の「herdr が固まるとサイクルが遅れる」行に、この手順を書き足した
- `wake()` は `WakeDirector` を 1 回呼ぶ。`DirectorWakeMemory` は `watchAllRun` が 1 つ持つので、サイクルをまたいで残る。見張りを起動し直すと忘れる
- `WakeDirector` が読めなかった(`director.jsonl` か受信箱が読めない)ときは `watchdog: director wake skipped this cycle: ...` の 1 行を出し、pulse と印は続ける
- `WakeDirector` は `[org.director].enabled` を読まない。enabled が false だと attach は断られるが、前からある push の登録は動く(verify I2、D5 と D6 は enabled で止めることを求めていない)
- `DirectorWakeDir` が空だと、wake_cmd の送りだけが失敗の行になる。herdr の既定の送り方は動く

## 5. 起こす件と 1 行(`internal/org/director_wake.go`、D5、D7)

- `directorToWake` が前半を受け持つ。`director.jsonl` を読み、次のどれかなら行を書かずに返す(`DirectorWakeSkipped*`)
  - 今の登録がない(`no_registration`)
  - 登録の wake に push がない(`no_push`、`WakesByPush`)
  - headless で `running` でない(`not_running`)
  - 受信箱を読んで、起こす件がない(`nothing_to_wake`)
- `DirectorWakeItems` が選ぶのは、状態が open で `Woken` が false の件の ID(ID 順)。acked と resolved は選ばない。delivery は見ないので、escalate が人にすぐ送った `now` の件も 1 回起こす。director が登録される前からある open の件も、`woken` がなければ 1 回起こす
- `DirectorWakeText` は固定の 1 行を作る。文は `New org inbox items for the director: <ID の列>. Read them with ralph org inbox and handle each as the director skill says.` である
  - ID は fold が通した `e<N>` の形だけ。20 件(`directorWakeListMax`)まで並べ、残りは ` plus N more` と数だけを書く
  - 本文、TYPE、org_id、送り先は載せない。シェルの記号(`;`、`|`、`&`、`$`、バッククォート、`<`、`>`、括弧、`*`、`?`、引用符)を含まず、先頭は `/`、`!`、`#` でない。`TestDirectorWakeText` が文を固定する
- 送るのは 1 サイクルに 1 回。同じサイクルに 2 件あれば、1 行に 2 つの ID が載る

## 6. 送り手の選び方と送り方(`director_wake.go`、`director_cmd.go`、D8)

- `directorWakeSender` が登録ごとに送り手を選ぶ
  1. wake_cmd が空でなければ、どの登録にも wake_cmd(`via=wake_cmd`)。まず `config.ValidateDirectorWakeCmd` を通す(`Org` は `config.Load` を通さずに組めるため。落ちれば `wake_cmd_invalid`)。wake_cmd が `{target}` を含むときだけ、`ResolvedWakeTarget` を `ValidateDirectorWakeTarget` に通す(手で書き換えた `director.jsonl` への備え。落ちれば `target_invalid`)
  2. wake_cmd が空で、登録が herdr の headless(`HerdrLaunch`)なら、herdr の pane(`via=herdr`)。pane の id も同じ形の検査を通す
  3. どちらでもなければ `no_sender`(昇格したセッションと launch_cmd の director で wake_cmd が空)
- `runDirectorWakeCmd`: 全要素に `ExpandDirectorWakeArg` で `{target}` と `{text}` を置き、シェルを通さずに打つ。`callWithTimeout`(`driverCallTimeout`、10 秒)の下で `runDirectorCmd` を呼ぶ。stdin は空で、stdout と stderr はまとめて末尾 512 バイト(`directorWakeOutputLimit`)だけを失敗の行に残す
- `runDirectorCmd` は launch_cmd の runner(`runDirectorLaunchCmd`)を一般化したもの。エラーは渡された key(`launch_cmd` か `wake_cmd`)で始まる。launch_cmd の出力は 4096 バイトのまま。`defaultDirectorLaunchWaitDelay` は `defaultDirectorCmdWaitDelay` になった
- `typeDirectorWakeLine`(herdr の既定)の順
  1. `confirmSeatPane` で、pane が登録のものか確かめる。stop と同じ確かめで、`PaneGet`、`TabGet`、`WorkspaceGet` の 3 回の呼び出しで tab と workspace の label が `DirectorTabLabel` と `DirectorWorkspaceLabel` に合うかを見る。pane がない、合わない、確かめの呼び出しが失敗したときは、何も打たない
  2. `PaneSendText`
  3. `sendEnterDelay` だけ待つ(既定 750 ms、`DefaultSendEnterDelayMS`。`ralph org send` と同じ。Enter が貼り付けの一部に読まれるのを避ける)
  4. `PaneSendKeys` で Enter
- herdr の各呼び出しは別々に 10 秒で区切る

## 7. 失敗と保留(`DirectorWakeMemory`、D6、D10)

- 送れなかった(wake_cmd の終了コードが 0 でない、10 秒の時間切れ、herdr の失敗)ときは、`woken` を書かず、`watchdog: director d1 not woken for e3 e5: ...` の 1 行を stderr(`--detach` なら `watch.log`)に出し、その登録の失敗の時刻を覚える。次の試しは `directorWakeRetry`(5 分)あける。保留の判定は受信箱を読んだあとなので、起こす件がないサイクルは何も出さない
- 送れたのに `woken` を書けなかったときも、保留に入れて行に書く。壊れた受信箱に毎サイクル打たないため
- 送れる手段がない登録は、失敗でなく skip として扱う。`wake_cmd_invalid`、`target_invalid`、`no_sender` の 3 つで、`warnOnce` が登録の id と理由ごとに 1 度だけ `... is not woken: ... (said once)` を書く。登録が入れ替われば id が変わるので、新しい登録にはもう一度言う
- メモリは見張りのプロセスの中だけ。送ったあと `woken` を書く前に見張りが落ちると、次の見張りが同じ件をもう 1 回起こす。重なりは 1 回までとして受け入れた(D6)
- herdr で send-text が通り Enter だけ失敗すると、1 行は入力欄に残り、5 分後の再試行がもう 1 度打つ(self-review L1)。コードは変えず、recipe の失敗の節に書いた
- stderr の行は、1 行まるごと `PrintableInboxText` を通す(`writeDirectorWakeLine`)。値が `director.jsonl`、herdr、wake_cmd の出力から来るため
- ack の期限は残る(D10)。`InboxDeadlineDue` は `Woken` も `WokenAt` も読まず、doc にそう書いた。期限の送りは起こす手順より先に走る。起こせた件も起こせなかった件も、ack しなければ `ack_minutes` で人に届く。`TestInboxDeadlines_AWokenItemComesDueAsOneNotWoken` と `TestRunWatchAll_WokenAndNotWokenItemsBothReachTheHumanAtTheAckDeadline` が固定する

## 8. 受信箱の `woken`(`internal/org/inbox.go`、S3a、0f9b457f)

- 新しい event `InboxEventWoken`。`InboxEvent` に `director_id`(`d<N>`)と `via`(`InboxWokenViaWakeCmd` は `wake_cmd`、`InboxWokenViaHerdr` は `herdr`)が増えた
- `AppendWoken(ids, directorID, via)`: 1 回のロックで受信箱を読み直し、件ごとに 1 行を同じ ts で、1 回の write で追記する。次のときは何も書かずに断る
  - ids が空、director_id が `d<N>` でない、via が 2 つのどちらでもない、ID の重複
  - 受信箱にない ID(`ErrInboxUnknownID`)。受信箱の file がなければ、state dir もロックも作らない
- 1 回の write のため、`appendLocked` を複数の event を受ける形にした。末尾が改行でない行を別の壊れた行として残す既存の守りは、複数 event でも保つ
- fold は `Woken` と `WokenAt` を載せるだけで、状態を変えない。どの状態の件にも付く(起こしているあいだに ack された件も、起こした事実として書く)。知らない件への `woken` は無視して数える。この段より前の ralph は `woken` を知らない event として数え、無視する
- `woken` は名指した件でなく、起こす対象の全件に書く。1 行が名指すのは 20 件までだが、`AppendWoken` には全件の ID を渡す。mutant C4(名指した 20 件だけに書く)は `TestWakeDirector_OneRunNamesEveryNewItem` が落とす
- 表示: `ralph org inbox show` の events に `<ts> woken director_id=d1 via=herdr`、`--json` の件に `woken` と `woken_at`。一覧の列は変えていない

## 9. 文書(S4、78f8acb3 ほか)

- `/director` skill の 4 面(同一)
  - 昇格の手順に、wake の説明と attach の例と断りを足した。人が打ち込むセッションは pull のままにする。push だと見張りの 1 行と Enter が人の入力欄に入り、打ちかけの文と一緒に送られるため
  - 「受信箱の処理」を、wake ごとの扱い、wait で待つ、見張りが送る 1 行、件の処理に分けた。「まだないもの」の節は消した
- `/org` skill の 4 面は `watch`、`inbox`、`director` の行と受信箱の節。雛形 `internal/org/prompts/director.md` は、herdr と launch_cmd の 2 通りで、1 行が来ること、人の指示でないこと、`ralph org inbox` で読むことを書く(`TestRenderDirectorPrompt_TellsWhatTheWakeLineIs`)
- README(動詞の表、watch の段落、director の段落、新しい wake の段落)、`AGENTS.md` の `internal/org/` の行、仕様 FR-10 の注(8b を 2 本に分けたこと、見張りが打つこと、送り先の読み替え、受け入れ条件 120 行と 121 行の読み替え)、`templates/base/ralph.toml` のコメント。help は attach、start、status、watch、escalate、`inbox show`、`--config`
- recipe `docs/recipes/director-wake.md` と `templates/base/docs/recipes/director-wake.md`: herdr(wake_cmd なし)、herdr の中の昇格したセッション、Orca、tmux の 2 通り、launch_cmd の director の例。pull を勧める場面、エージェントが 1 行をどう扱うか、送れないとき、Security、確かめたことも書く。`director-launch-cmd.md` の 2 面にも、start の断りと wake の案内を足した。`docs/architecture/repo-map.md` の recipe の一覧は sync-docs が足した 1 行
- tech-debt: 新しい行は `{target}` が shell-safe の文字に制限されないこと(`{target_sh}` の案)。書き換えた 6 行は、サイクルの遅れの行に起こす手順を足したもの、watch が死んだことに気づかない行(push の起こしは見張り自身が送るので気づけない)、登録の生存と `starting` の期限の 2 行(8b を 8b-2 に絞った)、`attach --replace` と `detach --force` の tripwire の行(push で登録が送り先を決めるようになったので、書き換えで送り先を変えられる。歯止めは 1 行が ID だけなことと ack の期限)、解消済みの行 1 つ(`defaultDirectorCmdWaitDelay` への改名の追従)

## 10. テストと evidence

- 主なテストと固定するもの
  - `director_wake_test.go`(org): `TestValidateDirectorWakeTarget`、`TestCheckDirectorAttachWake`、登録の記録と断り、start の wake。`TestWakeDirector_*` は wake_cmd の 1 回の送り、1 行に全 ID、skip の条件、5 分の保留(境目の 4 分 59 秒と 5 分)、said-once、相対パスと PATH、woken の書き込み失敗、入れ替わった登録、herdr の順
  - `org_director_wake_test.go`(cli): attach の断りの表(何も書かれないこと)、main worktree の読み方、start の断りと通り、status の表示と escape、`TestStatusJSON_CarriesTheWakeAndTheWakeTarget`、help
  - `watch_all_test.go`: サイクルの順(`TestRunWatchAll_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse`)、pull は起こさない、`director.jsonl` が読めなくても pulse は続く、escalate から `WaitInbox` と見張りまで通す `TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem`
  - `org_watch_test.go`: main と feature worktree に同じ相対パスの別のスクリプトを置き、main 側だけが main のルートで打たれること(`TestOrgWatch_RunsTheWakeCmdOfTheMainWorktreeInItsRoot`)
- テストは本物の herdr、osascript、端末、Orca、tmux を打たない。偽の herdr と、テストが書く wake_cmd(引数を印のファイルに書くスクリプト)を使い、台帳は一時の dir に置く。/test は PATH の先頭に canary を置いて、6 つのコマンドが 1 回も呼ばれないことを確かめた
- mutation(`docs/evidence/test-2026-10-18-org-director-wake-mutation.txt`): 83 個のうち 77 個が赤。生き残りは同値 2 個(O3、G5)と穴 4 個(G7、O4、H6、M11)。穴は e50bed12 で閉じた。サイクルをまたぐメモリ(`TestRunWatchAll_HoldsAFailedWakeBackAcrossCycles`、`TestRunWatchAll_SaysTheNoSenderLineOnceAcrossCycles`)、使えない pane の id(`TestWakeDirector_AnUnusableHerdrPaneIsSkippedWithoutCallingHerdr`)、PATH で引く名前(`TestWakeDirector_ABareProgramNameIsLookedUpInPath`)、`ralph status --json`。揺れた時間切れの行の `DriverCallTimeout` は 1 秒から 3 秒にした
- 実機(`docs/evidence/director-wake-livefire-2026-10-18.md`): 分離した herdr server(XDG の 3 変数)と隔離した台帳で、codex 0.160.1 の headless director を 3 回起こした。`$HOME` は `~`、token と pane の id は伏せてある
  - Run 1(`["push"]`): 待っていない codex は 1 行を新しい手番として受け、受信箱を読んで ack し、QUESTION を人に聞いた。本文の頼みを自分では実行しなかった
  - Run 2(`["pull", "push"]`): wait が拾った件を、同じ秒に 1 行が名指した。`woken` が `acked` のあとに書かれた。手番の途中に届いた 1 行と Enter は、次の手番まで待たされず今の手番に差し込まれた(同じ turn_id)。失われず、ack は 1 回だった
  - Run 3(`["push"]`、S4 のあとの skill と prompt): escalate の 1 秒後に起こされ、新しい手番で ack して人に聞いた。wait は始めなかった
  - Run 1 と 2 は S4 より前の skill と prompt だった。そのため、evidence と recipe に前提を書き、Run 3 を足した(verify V3)

## 11. 後工程の 1 回(cycle 1)

- 実装: S1 2f08cfe7、S2 0d49b2a0、S3a 0f9b457f、S3b b716d041。S5(実機)を S4(文書)より先にして b17b58da、S4 を 78f8acb3 と 7b8b5449 に置いた。codex が作業中の入力をどう扱うかの結果を、skill と recipe に書くため
- self-review(2129f422): MEDIUM 1(M1、`{target}` をシェルの文字列に入れる危険の注意が文書にない)と LOW 7(L1〜L7)。S6(705e56bf)で直した。L5 で attach の検査と登録の部品を `director_wake.go`、CLI の部品を `org_director_wake.go` に移し、`WakeDirector` を `directorToWake` と `sendDirectorWake` に分けた。再レビュー(3a95cfc8)の LOW 4(R1〜R4、コメントと doc)は a0170256 で直した。再レビューは移動を、行の文字列の多重集合で突き合わせて純粋な移動と確かめた
- verify(abf295f9、pass): MEDIUM 1(V1、tech-debt の行の Trigger が発火したのに残っている)、LOW 2(V2 recipe に herdr の昇格したセッションの例がない、V3 実機が S4 の前)。4331358c で直した。再確認の LOW 1(N1)と情報 2 件は ea55f731
- test(6a99b1a8、pass): mutant 83 個。テストの穴 5 件を e50bed12 で閉じ、implementer が scratch の写しで 4 つの mutant が新しいテストで赤になることを確かめた。implementer が見つけた LOW(`-` で始まる送り先の理由の文が、herdr の既定の道でも wake_cmd と書く)は d2a740d3 で直した。d2a740d3 で `./scripts/run-verify.sh` は rc 0
- sync-docs(157c0274): ずれは 1 点で、`docs/architecture/repo-map.md` の recipe の一覧に director-wake がなかった。足した
- cross-review(9ff6bd04): codex(`gpt-6-astra`、reasoning effort `xhigh`、read-only、`--ignore-rules`)の指摘は 0 件。codex はテストを走らせていない(sandbox のため)ので、テストは /test と d2a740d3 の run-verify の結果による
- e50bed12 と d2a740d3 は /test のあとに入った。self-review、verify、test はこの 2 つのあとで走り直していない。d2a740d3 のコード変更は、理由の文 1 つ(`ValidateDirectorWakeTarget` の `-` で始まる値の文)である。sync-docs と cross-review は、この 2 つを含む範囲を読んだ

## 12. 試していないことと、先に読む場所

- 試していないこと
  - Claude Code の director、Orca の `orca terminal send`、tmux の `send-keys`、herdr の中の昇格したセッションの例。recipe と evidence が実行していないと書いている。Claude Code が手番の途中に届いた 1 行をどう扱うかも、skill が「確かめていない」と書いている
  - 現在の skill での `["pull", "push"]` と pull だけ(Run 2 は S4 より前の skill)
  - 本物の herdr で wake が失敗する場合と、その後の 5 分の保留。偽の herdr のテストだけである
  - 実機の Run 3 の binary は abf295f9 で、そのあとの非テストの Go の変更は d2a740d3 の理由の文 1 つだけである。HEAD の binary で実機は打っていない
  - CLI の binary を通して escalate、`wait --inbox`、`watch --all` を 1 本で打つテストはない。org の層と CLI の層に分かれて固定されている。`watch --all --detach` の子が打つ場所は、コードを読んだだけである
  - e50bed12 の 4 つの mutant は、implementer の報告で、/test は走らせ直していない
  - 時間切れの行(`no_return_in_time`)は、1 秒の設定で 40 回に 1 回落ちた。3 秒に直したあと、20 回の繰り返しは load average が 4〜5 の環境で確かめただけで、負荷の高い環境では確かめていない。この差分の外の `TestRegate_FetchFailures` も、並列の負荷で落ちることがある(org-auto-merge の /test と同じ形、原因は未確認)
- 先に読む場所
  - `internal/cli/org_watch.go` の `orgWatchDirectorWake`: wake_cmd を打つディレクトリの規則で、安全の中心はここ
  - `internal/org/director_wake.go` の `WakeDirector`、`directorToWake`、`sendDirectorWake`、`directorWakeSender`
  - `internal/config/config.go` の `ValidateDirectorWakeCmd`: 最初の要素に placeholder を許さない規則
  - `internal/org/inbox.go` の `AppendWoken` と fold の `woken`
  - `.claude/skills/director/SKILL.md` の「見張りが送る 1 行」。director が 1 行を受けたときの手順である
