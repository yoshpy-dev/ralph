# Walkthrough: 座席を別のアカウントへ移すときの herdr のサーバーの立て直しを、`ralph org reseat --restart-herdr` と `--claude-config-dir` の 1 回で済ませる(org-reseat-restart-herdr)

- 計画: `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md`(PR の作成のあとに archive へ移す)
- issue: #354
- 差分: `git diff main...HEAD`(この walkthrough を足す前で 41 ファイル、+5,302 / −167。base は main の 29dd00fd で、merge base も同じ。branch には origin/main を d05fa43e(main の 26b9925c)と b7c41aa3(main の 29dd00fd、#358)で取り込んである)。Go の本体は 7 ファイルで +1,198 / −62、Go のテストは 9 ファイルで +2,675 / −4。残りは skill の写し 8 つ・README・AGENTS.md・仕様 4 つ・quality-gates 2 つ・tech-debt(17 ファイル、+695 / −101)と、計画・報告・insight・実機の記録(8 ファイル、+734)
- パイプライン: 1 周(`cycle-count.json` は cycle 1)。self-review は 3 回で、どれも Merge: yes(run 1 の M1 と L1〜L4、run 2 の L5〜L7、run 3 の L8。L6 は直さず doc に書き、残りは直した)。verify と test は pass、sync-docs は verify の文書の指摘を直し、cross-review(codex、HEAD 7a2d45ab)は指摘 0 件
- 読む順は下の節の番号のとおり。1 は頼まれたこと、2〜5 が機構の本体、6 が CLI の出力、7 がテスト、8 が実機の記録、9 が文書、10 が残したもの。判断が要るのは 10 の最初の項(窓)

## 1. issue が頼んだことと、仕様が選んだこと

- 座席の Claude のアカウントが週の上限に当たったとき、別のアカウントで続けるには、新しいアカウントの環境(別の `CLAUDE_CONFIG_DIR`)から herdr のサーバーを起動し直してから `ralph org reseat` を打つ。座席の pane は herdr のサーバーの環境を引き継ぐためである。`/director` skill はサーバーを止めない決まりなので、人が別の端末で `herdr server stop` を打ち、済んだことを director に伝えていた
- issue の困りごとは 2 つ。止めたあとの最初の reseat を新しい環境から打つ必要があるのに、動いているサーバーがどの環境で起動されたかが人にも director にも見えない。人が止める頃合いを見て済んだことを伝える待ちが生まれる
- 提案は 3 つ。専用の動詞(`ralph org herdr restart`)、reseat の `--restart-herdr`、`ralph status` か `ralph doctor` での `CLAUDE_CONFIG_DIR` の表示。仕様 `docs/specs/2026-11-20-org-reseat-restart-herdr.md` は 2 番目を選んだ(D1。止める・起動する・座席を立てるを 1 回の `reseat.lock` の中で済ませ、間に別の動詞を入れない)。3 番目は reseat の出力に留め(D4)、`ralph status` と `ralph doctor` は範囲外にした(仕様の Out of scope)。専用の動詞は作っていない
- 断る条件は issue の提案より広い。提案は「ほかに active な座席を持つ org か、herdr の中の headless の director があれば断る」だが、仕様はサーバーにその org の座席の pane のほかの pane が 1 つでもあれば断る(D2)。台帳は repo ごとで、サーバーはユーザーに 1 つなので、台帳ではほかの repo の org や ralph の外の shell の pane が見えない。仕様は、このマシンのサーバーに別の repo の org の pane 3 つと shell の pane があったと書く(2026-11-20 の実測)
- `--claude-config-dir <dir>` は issue にない。director のセッションの環境は古いアカウントのままなので、呼んだ側の環境で起動するだけでは、director に頼んだときにアカウントが移らない(D3)。D5 で、この値を打ち直しの命令にも写す。`HOME` は変えない
- 仕様の Status は Approved(consult の指摘 3 点を入れたあと)。FR-1〜FR-8、NFR-1〜NFR-3、AC1〜AC8 は sync-docs の pass で `[x]` にした

## 2. 再起動の順(`internal/org/reseat.go`、読む価値が一番高い)

`--restart-herdr` を付けた本当の reseat は、次の順に進む。`ReseatParams` の新しい欄は `RestartHerdr`(`:92`)と `ClaudeConfigDir`・`ClaudeConfigDirGiven`(`:97-98`)で、結果の `HerdrStopped`・`HerdrRestarted` は `:1145-1149`。サーバーの状態を読む probe とゲートは関数の欄(`ReseatParams.HerdrServerProbe`、`Org.HerdrServerGate`)で受けるので、org の層は lsof も herdr の起動も知らない。

1. `PlanReseat` の確かめ 9、`reseatRestartHerdrErr`(呼び出し `:570-575`、本体 `:649-683`)。次のどれかで、何も止めずに断る。呼んだ側が herdr の pane の中(`HERDR_PANE_ID`、`:650-653`)。`--seat` が active な座席の全部を含まない(`:654-664`。サーバーと一緒に選ばれなかった座席も終わる)。probe かゲートがない(`:665-672`)。サーバーが動いていない(`:673-681`。止める必要がなく、`--restart-herdr` なしの reseat がサーバーを起動する)
   - 仕様の FR-1 は確かめを 1〜9 と書くが、実装では 9 を `--restart-herdr` 自身の確かめ、ゲートを 10 にした(`:425-436`、仕様 FR-1 の「実装の順」)。ゲートは止まったサーバーを起動するので、断られる restart が起動しないよう確かめを先に置いた
2. ゲート(確かめ 10、`:586-591`)。見張りを起動することがある。サーバーは 1 で動いていると確かめてあるので、ここでは起動しない(`:428-433`)。dry run は起動せず、状態を読むだけ(`reseatDryRunGates`、`:581`、`:1027`)
3. 1 回目の pane の確かめ、`reseatFirstPanesErr`(`Reseat` の `:1366-1370`、dry run は `:1335-1339`、本体 `:730-741`)。`herdr pane list` を workspace を絞らずに読み、台帳の active な座席の pane(`reseatOwnPanes`、`:690`)でないものが 1 つでもあれば断る
   - 断りは各 pane の id・label・cwd を並べ(`reseatOtherPanesText`、`:746-763`)、前の `herdr server stop` のあとに herdr が戻した記録だけの pane の見分け方(`herdr pane process-info <pane_id>`)と閉じ方を案内する(`reseatOtherPanesHint`、`:769-772`)
   - 最新の state event に pane id がない座席(spawn_started)は自分の pane に数えない(`:687-689`)ので、その pane があれば断る側に倒れる。一覧が読めなくても断る(`:731-735`)
   - ここまでは何も止めていないので、断っても台帳に何も書かない
4. 座席を止める、`stopAll`(呼び出し `:1389`、本体 `:1419-1441`)。今の reseat と同じ
5. サーバーの立て直し、`restartHerdr`(`:1495-1541`)
   - 2 回目の pane の確かめ(`:1500-1512`)。読めない、またはほかの pane ができていれば、サーバーを止めずに終わる。座席は止めたままで、`reseated` は記録する(`attempted` は `stopAll` で立つ、`:1429`。仕様 FR-3 の注)
   - `herdr server stop`(`:1514`)。2 回目の一覧とこの停止のあいだに別の呼び出しはない(`:1500` から `:1514`。`reseat_restart_test.go:237-240` が、停止の直前の呼び出しが 2 回目の一覧であることを固める)。停止が失敗したら probe を 1 回読み、止まっていれば `HerdrStopped` を立てる(`:1515-1529`)
   - 止まるまでの待ち、`waitHerdrStopped`(`:1530`、本体 `:1548-1568`)。100 ms ごとに probe を読み、10 秒で諦める(`:1072-1073`)。諦めたときは、サーバーはまだ止まるかもしれないと出し、起動し直さず座席も立てない
   - `HerdrStopped` を立てたあと(`:1533`)、ゲートがサーバーを起動する(`:1534-1538`)。起動は呼んだ側の環境で、`--claude-config-dir` があれば `CLAUDE_CONFIG_DIR` が置き換わる(4 節)。失敗したら座席を立てずに終わる。通れば `HerdrRestarted`(`:1539`)
6. 座席を立てる、`startAll`(呼び出し `:1390`、本体 `:1578`)。今の reseat と同じ。pane は新しいサーバーの環境を引き継ぐ
7. `finish`(`:1669`)が `reseated` を記録し(`:1671`)、打ち直しの命令を作る(`:1681-1683`)

打ち直しの命令の規則(`reseatRetryCommandWithHerdr`、`:1884-1903`。flag の部分は `reseatRetryHerdrFlags`、`:1867-1876`)は、`finish` が `restartHerdr = RestartsHerdr && !HerdrStopped` で決める(`:1682`)。

- サーバーが止まったのを見ていない(2 回目の確かめで断った、`herdr server stop` が失敗して状態が running か読めない、10 秒待っても止まらない)ときは `--restart-herdr` を残す。サーバーがまだ動いているかもしれない
- 止まったのを見たあと(ゲートの起動が失敗した、leader が立って別の座席が失敗した)は `--restart-herdr` を外す。付けたままにすると、立ち直った leader をもう 1 度止める。サーバーが止まったままなら、打ち直しの確かめ 10 のゲートが `--claude-config-dir` つきで起動する
- `--claude-config-dir` は、渡されていれば必ず写す(`''` は `ShellWord` が引用符 2 つにする)。打ち直しが呼んだ側の古いアカウントに戻らないため(FR-5b)
- 取り違えると、立ち直った座席を止めるか、止まったサーバーに断られる。AC4(`reseat_restart_test.go:458`)と AC4b(`:496`。出た命令を実際に打つ)が固める

## 3. herdr のドライバ(`internal/org/driver/herdr.go`)

- `PaneListAll`(`:484`)は `herdr pane list` を `--workspace` なしで打つ。読み方は `PaneList` と同じ `herdrPaneListResult`(`:491`)に括り出した。`result.panes` がない、`pane_id`・`tab_id`・`workspace_id` を欠く pane は error。空の一覧は error にしない
- `HerdrPane` に `Cwd` を足した(`:288`、JSON の欄は `:522`。null か欠けていれば `""`)。断りに cwd を出すため
- `ServerStop`(`:1006`)は `herdr server stop` を打つだけで、成功の応答を読まない。herdr 0.7.5 は成功で何も出さなかった(8 節)ので、exit 0 で error envelope でなければ成功とし、`checkHerdrCloseResult` の規約に従う。止まったかどうかは呼び出し側が `ServerStatus` で読む
- `HerdrClient`(`internal/org/spawn.go:153-161`)に 2 つの method が増えたので、テストの fake の `fakeHerdr`(`spawn_test.go`)と `fakeWatchHerdr`(`watch_test.go`)も 2 つずつ持つ。`fakeHerdr.PaneListAll` は、`WorkspaceCreate`・`TabCreate`・`PaneSplit` で作った cwd を返す
- テストは `herdr_test.go:2379`(`PaneListAll`)、`:2441`(cwd の読み)、`:2463`(`ServerStop`)

## 4. ゲートの環境の上書き(`internal/cli/org_herdr_server_env.go`、`org_herdr_server.go`)

- 上書きは `orgHerdrServerEnvOverride`(`org_herdr_server_env.go:43-46`)の `apply`(`:53-65`)。ゼロ値は何も変えない。`setClaudeConfigDir` が立っていれば、環境から `CLAUDE_CONFIG_DIR` の項目をすべて外し、値が `""` でなければ末尾に足す。元の slice は変えない
- flag の値は `checkOrgClaudeConfigDir`(`:74-90`)が見る。`""` は外す指定。そうでなければ絶対パスにし、存在するディレクトリでなければ断る(symlink は辿る)。サーバーは `HOME` で動き、打ち直しの命令は別のディレクトリから打たれうるので、絶対パスにしてから渡す
- 配線: `runOrgReseat`(`internal/cli/org_reseat.go:286-295`)は、何よりも先にこの確かめをする。`--claude-config-dir` が付いたかは `cmd.Flags().Changed` で見る(`:214`)ので、`''` も「渡した」になる。上書きを持つゲートは本当の reseat にだけ入る(`wireOrgHerdrServerGateWithEnv`、呼び出し `:323`、定義 `org_herdr_server.go:105`)。ゲートの本体は `ensureOrgHerdrServerWithEnv`(`:147`)で、`orgHerdrServerLaunchRequest` が `env.apply(orgHerdrServerEnv(os.Environ(), dir))` を使う(`:312`)
- 今の `wireOrgHerdrServerGate`(`:96-98`)と `ensureOrgHerdrServer`(`:139-141`)は、ゼロの env で新しい関数を呼ぶだけの入れ物にしたので、start・spawn・director start の挙動は変わらない
- 上書きが効くのは、ゲートがサーバーを起動するときだけ。動いているサーバーはそのまま通す(`:146`)。`--restart-herdr` なしの `--claude-config-dir` が動いているサーバーに効かないのはこのためで、reseat は stderr でそう知らせる(6 節)
- 変えるのは `CLAUDE_CONFIG_DIR` だけ。herdr の pane の変数と `CLAUDECODE`・`CLAUDE_CODE_*` を外す今の規則(`orgHerdrServerDropsEnv`)、`HOME`、`PWD` はそのまま。2026-11-12 の reseat の仕様の D6 が退けたのは pane ごとに `--env` で渡す案で、ここで変えるのはサーバーの環境そのものである(仕様 D3)
- テストは `org_herdr_server_env_test.go:45`(`apply`)、`:84`(ゲートが上書きの値でサーバーを起動する)、`:148`(配線)、`:213`(flag の値の確かめ)、CLI 側は `org_reseat_restart_test.go:278`(確かめが最初)と `:306`(本物のゲートを fake の seam につなぎ、flag の値が起動する環境に届く)

## 5. 動いているサーバーの `CLAUDE_CONFIG_DIR` を読む(`internal/cli/org_herdr_server_env.go`)

- `readOrgHerdrServerClaudeConfigDir`(`:173-212`)の順は 3 つ。`herdr status server --json` で socket を得る(動いていない、読めない、socket がなければ「読めない」)。`lsof -t <socket>` で pid を引く(`orgHerdrServerPID`、`:234-272`。2 つ以上なら `ps -o comm=` の末尾が `herdr` のものに絞り、1 つにならなければ「読めない」)。環境を読む。Linux は `/proc/<pid>/environ`(`orgEnvironValue`、`:323-334`)、ほかは `ps eww -o command= -p <pid>`(`orgPSEnvValue`、`:303-317`)
- 結果は set・unset・unreadable の 3 状態(`:96-106`)。unreadable がゼロ値で、何も入れていない結果が unset に読めないようにしてある。環境の項目が 1 つもない `ps eww` の出力も unset にせず unreadable にする(`:208-209`。Apple のプラットフォームのバイナリは `ps eww` で環境を出さない)。失敗は error にせず、理由つきの unreadable にする。表示のための読みで、読めなくても reseat は断らない(仕様 D4)
- entry の始まりの pattern(`orgPSEnvKey = " [^ =]+="`、`:289`)。`ps eww` は引用符を付けないので、値の中の空白と次の項目の前の空白を区別できない。そこで値は ` CLAUDE_CONFIG_DIR=` の後ろから、次の ` <語>=` の前までとする。語は空白と `=` 以外の 1 字以上
  - トレードオフは `:291-301` に書いた。値の中に ` <語>=` があるとそこで切れて短く読むが、長く読むことはなく、別の項目の値は混ざらない
  - 最初の版は項目の名前を識別子の形に限っていた。そのため `x-y=...` のような名前の項目が後ろにあると、その値が `CLAUDE_CONFIG_DIR` の値に入って stderr に出た(self-review の M1。53c12663 で直し、`org_herdr_server_env_test.go:572` が固める)
- `ps eww` だけは `LC_ALL=en_US.UTF-8` で走らせる(`orgHerdrServerEnvCommandEnv`、`:336-358`)。`LC_ALL=C` だと macOS の ps は 0x80 以上のバイトをエスケープし、日本語のパスが別のパスに読める(self-review の L5、f620a6e8。Darwin 25.5.0 で確かめた)。その locale がない系は C のままで、エスケープされる。lsof と `ps -o comm=` は C のまま
- 呼び出しはどれも期限つき(NFR-3)。lsof と ps は `orgHerdrServerEnvCallTimeout`(5 秒、`:150`)、status は `orgHerdrStatusTimeout`。失敗した run の stdout は、環境が入りうるのでエラーに入れない(`:364-369`)
- 直さずに残した L6: 引数に `=` を持つ語を環境の項目と数えるので、環境を出さない process を読むと「読めない」ではなく unset になる(`:163-168` に書いた)。ralph が起こすサーバーの引数は `server` だけなので起きない。test report が本物の ps で確かめた

## 6. CLI の出力(`internal/cli/org_reseat.go`、`org_reseat_herdr_server.go`)

- flag は `org_reseat.go:230-235`、help の新しい段落は `:132-175`
- stdout: 本当の reseat は、座席の `stopped:` の行のあとに `stopped: herdr server`(`res.HerdrStopped`)、`started:` の行の前に `started: herdr server`(`res.HerdrRestarted`)を出す(`:598-603`)。`--dry-run` は `stop: herdr server` と `start: herdr server` を座席の行の間に出す(`printOrgReseatDryRun`、`:544-551`)。dry run も確かめ 9 と 1 回目の pane の確かめを走らせ、何も止めない(`reseat.go:1335-1339`、`reseat_restart_test.go:430`)
- stderr、reseat の前(`printOrgReseatHerdrServerNotes`、呼び出し `org_reseat.go:331`、本体 `org_reseat_herdr_server.go:39-67`)
  - `--restart-herdr` ならまず TUI の 1 行(`:23-24`)。サーバーを止めると herdr の TUI も閉じる。確かめに通ったあとに止める言い方にして、断られる run でも偽にならないようにした
  - `herdr server: CLAUDE_CONFIG_DIR=<値>`、`=unset`、`cannot be read: <理由>` の 1 行(`:73-81`)
  - `--restart-herdr` なしで、読めた値が座席に使わせたい値(`--claude-config-dir`、なければ呼んだ側の環境)と違えば、座席はサーバーの環境で立つのでそのアカウントにならない、という `note:`(`:117-127`)。読めず、`--claude-config-dir` が付いていて、サーバーが動いているときは、flag は動いているサーバーに効かない、という `note:`(`:132-134`)。どれも断らない
- stderr、reseat のあと: `--restart-herdr` がサーバーを起動し直した本当の reseat だけ、結果を出したあとに同じ行をもう 1 度読んで出す(呼び出し `org_reseat.go:344`、本体 `org_reseat_herdr_server.go:150-165`)。座席に使わせたい値と違えば `note:` が続く(`:171-180`)。止めてからゲートが起動するまでに別のプロセスがサーバーを起動したのかもしれず、立てた座席はそのサーバーの環境で動いている、という内容
  - これは計画の Scope にない追加である。verify の I6(止めたのを見てからゲートが起動するまでに別のプロセスがサーバーを起動すると、ゲートの上書きは効かない)を受けて 562e1ab5 で足し、`note:` の文は b1fd40e8 で、座席が 1 つも立たなかった場合でも偽にならない言い方に直した
- `orgReseatRestartHerdrServerProbe`(`org_reseat.go:509-524`)は org の層に渡す probe。今の `orgReseatHerdrServerProbe` と違い、`herdr status server --json` を持たない herdr を「動いている」に数えず error にする。止まったかどうかを読めないため
- CLI のテストでは、`TestMain` が読み(`orgHerdrServerClaudeConfigDirReader`)を nil にする(`main_test.go:211`)。nil は何も読まず何も出さない。この machine の herdr のサーバーに lsof や ps を打たないためと、reseat の stderr 全体を固定する既存の 4 本のテスト(`codex_hook_trust_notice_test.go`)を変えないため。その結果、本番の probe は CLI のテストのどれからも通らない。7 節の本物のプロセスのテストがこの穴を塞ぐ

## 7. 先に読むテスト

- `internal/org/reseat_restart_test.go:218` `TestReseat_RestartHerdr_StopsTheServerBetweenTheStopsAndTheStarts`(AC1)。呼び出しの列を固める(`:229-230`)。status の読み、ゲート、1 回目の一覧、座席の停止、2 回目の一覧、`server_stop`、status の読み 2 回、ゲート、座席の起動の順。停止の直前が 2 回目の一覧であること(`:237-240`)、停止のあとに座席の停止も `server_stop` もないこと(`:234-236`)も見る
- `:458`(AC4。止めているあいだに pane ができると、サーバーを止めず、座席を止めたまま、`--restart-herdr` と `--claude-config-dir` を持つ打ち直しを出す)、`:496`(AC4b。leader が立って別の座席が失敗したあと、出た命令は `--restart-herdr` を持たない。その命令を実際に打つと、status を読まず、一覧を読まず、サーバーを止めず、残りの座席だけを立てる)、`:734`(`reseatRetryHerdrFlags` の表)
- `:255`(AC1b)、`:292`(AC2。断りの全文を実行と dry run の両方で固める)、`:339`(AC3)、`:371`(動いていない・読めないサーバー)、`:539`・`:578`・`:621`・`:665`(止まらない、停止の失敗、停止後のゲートの失敗、一覧が読めない)、`:708`(`--restart-herdr` なしでは pane の一覧にもサーバーにも触らない)
- CLI: `org_reseat_restart_test.go:110`(AC1 を CLI から)、`:360`(読みの行と `note:` の各場合。stderr 全体を固める)、`:481`(立て直したあとの行)、`:656`(出た打ち直しの命令を打つ)、`:732` と `:769`(help の pin)
- 読みのテスト: `org_herdr_server_env_test.go:509`(値が次の項目の前で終わる)、`:572`(次の項目が識別子の形でない名前でも値に入らない。M1 の変異で 6 件落ちる)、`:616`(Linux の `/proc`)、`:705`(UTF-8 の ps)
- `org_herdr_server_env_process_test.go:89` `TestOrgHerdrServerReadClaudeConfigDir_ReadsARealProcess`。本番の probe(`orgHerdrServerEnvProbeNow`)が本物の lsof と ps(Linux は `/proc`)で、実在のプロセスの `CLAUDE_CONFIG_DIR` を読むことを見る唯一のテスト。status の読みだけを fake にし、テストのバイナリ自身を子として起動して unix socket で待たせる(macOS の ps は Apple のプラットフォームのバイナリの環境を出さないため、`:26-28`)。値は空白と日本語を含む `<tmp>/日本語 dir`、次の項目は `X-Y=next`。`-short`、darwin と linux 以外、lsof がない環境では skip する。test report の穴 1 を塞ぐために 261df0c9 で足した
- 変異(test report): 依頼した 5 種(8 個)は名前の挙がった AC のテストで落ちた。追加の変異 6(本番の probe の `goos` を linux にする)は既存のテストで生き残り、上の本物のプロセスのテストで落ちる。変異 1b(2 回目の一覧の結果を見ない)は org の AC4 のテストだけが落とし、CLI のテストは落とさない(test report の穴 3)

## 8. 実機の記録(`docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md`)

- このマシンの本物のサーバー(別の repo の org の pane がある)は止めず、`docs/recipes/isolated-herdr-server.md` の決まり(XDG の 3 変数と `HERDR_SOCKET_PATH` を `/tmp/rrh` の下へ、`env -i`)で分離したサーバーを使った。macOS 26.5.2、herdr 0.7.5。座席は立てていない
- 確かめたこと: `herdr pane list`(workspace なし)が shell の pane を cwd つきで返す(`PaneListAll` が読む形)。`lsof -t <socket>` が pid を 1 つ返し、`ps -o comm=` が `herdr` を返す。`ps eww` から `CLAUDE_CONFIG_DIR` が読める。`herdr server stop` は標準出力に何も出さずに終わり、1 秒後の status は `not_running` で、pid もソケットのファイルもなくなる。本物のサーバーには触れていない
- この記録は命令を手で打ったもので、ralph の読みの道は通っていない。道を通す確かめは 7 節の本物のプロセスのテストが担う(test report の穴 1)

## 9. 文書

- `/director` skill: 「座席を立て直す」の 3(`.claude/skills/director/SKILL.md:1515` から。10 行が 91 行になった)に、打つ命令、人に頼まれたときだけ打つこと、移す先の dir は人に聞くこと、断りの 4 つ、2 回目の確かめと 10 秒、窓、`stopped: herdr server` の有無で変わる打ち直しの flag、`herdr server:` の行と `note:` を人に伝えること、director が打てないときは人が止めること。「してはいけないこと」(`:2463`)に、`herdr server stop` は自分で打たないままの例外を 1 つ足した
- `/org` skill: 前提の文に `--claude-config-dir` の例外(`.claude/skills/org/SKILL.md:32`)、コマンド表の `reseat` の行(`:387`)、「座席の立て直し」の節(`:1675` から。別のアカウントへ移す段落は `:1845` 以降)
- skill の写しは 8 つ(`.claude` と `.agents`、root と `templates/base`)で、`.agents` の側は `scripts/sync-skills.sh` で作った。root と template の `.claude` は同じ内容
- `README.md:304`(reseat の段落)、`AGENTS.md:92`(1 句)、`docs/quality/quality-gates.md:99`(Herdr server gate の行。template にも同じ文)
- 仕様: `docs/specs/2026-08-01-org-runtime.md:135-142` の 2026-11-20 改訂 (a)〜(d)(本文の記述は書き換えず、`:84` に印だけ足した)、`docs/specs/2026-10-07-org-multi-org-director.md:167` に director の例外、`docs/specs/2026-11-12-org-reseat.md:70` に追記、`docs/specs/2026-11-20-org-reseat-restart-herdr.md` に FR-3 の注(2 回目の確かめで断っても `reseated` を記録する、`:41`)と `- 実装:` の行(`:7`)
- `docs/tech-debt/README.md`: 窓の新しい行(`:545`)と、この branch が Trigger に当たった 12 行の持ち越しの注。`internal/org/reseat.go` は 1,574 → 1,933 行で、この branch の前から 800 行を超えていた(`:520` の注)。この branch が足したテストファイルは、どれも 800 行を超えない(749・762・781 行)

## 10. 残したもの

- 最後の pane の確かめと `herdr server stop` の窓(`reseat.go:1500` から `:1514`)。ここで作られた pane は、どちらの確かめにも映らずサーバーと一緒に終わる。レビューで一番判断が要る項である
  - 経緯: Codex の計画助言の HIGH 2(計画の Progress、`:119`)は、サーバー側で pane の作成を締め出すか、条件つきの停止を前提にし、できなければ自動の立て直しを断れ、と求めた。計画は、herdr 0.7.5 にどちらもないと判断した(`api schema` の `server.stop` は `EmptyParams` だけ、計画の Design decisions 3、`:59`)。文字どおりに入れると機能がなくなるので、窓を残して Known gaps と tech-debt に書く、と決めた(consult の判定も同じ)。cross-review の codex も、この窓を「explicitly accepted limitation」と読んだ
  - 窓で終わりうるもの(tech-debt `:545`): 別の repo の ralph がちょうどそのとき立てた座席(ゲートを通った直後で、サーバーが落ちればその spawn は失敗して pane を閉じ、打ち直しで戻る)、同じ repo の別の shell で人が打った `start` や `spawn`(`reseat.lock` を取るのは reseat だけ)、人が herdr で開いた pane
  - 窓の長さは測っていない。計画の Design decisions 3 と Risks の「1 秒未満」は測った値ではなく、tech-debt の行の書き方(「測っていない。最後の一覧と停止のあいだに別の呼び出しは走らない」)を取る。計画の本文は承認の digest に入るので直していない(計画の Progress に記録)
  - 緩和は 2 つ。`--restart-herdr` は人が頼んだときだけ打つと skill に書いた。reseat が終わるまで、ほかの repo でも同じ repo の別の shell でも座席を立てず、herdr で pane を開かないよう、打つ前に人へ伝えることも skill に書いた
  - ralph の動詞に共有のロック(herdr の socket のパスで鍵を作る user 単位の flock)を取らせる案は、tech-debt の行に設計メモとして書き、入れていない。start・spawn・reseat・director start の 4 つに触れる。同じプロセスの中でゲートが取る共有ロックを排他に変える必要があり、flock はそれを原子的にできない、socket のパスを返さない herdr では鍵が作れない、書けないロックのディレクトリでは黙って弱まる、という 3 つの穴がある。人が herdr を直接触る分は、どの案でも防げない
- 座席のある org での通しは打っていない。順と断る形は fake の herdr で確かめ、herdr の呼び出しは分離したサーバーで手で確かめただけである(8 節)
- 新しいアカウントの資格で座席が動くかは確かめていない。`--claude-config-dir` は、人がその値の shell から起動したときと同じ環境を作るだけで、ログインの状態や keychain は見ない。2 つのアカウントを持つ人の手でしか確かめられない(仕様の Open questions、計画の Risks)。reseat の出力が人に頼むのは、座席の画面でどのアカウントで動くかを見ることである
- Linux は実機で確かめていない。`/proc/<pid>/environ` の道は、fake の `readFile` のテスト(`org_herdr_server_env_test.go:616`)と、CI(`ubuntu-latest`、`.github/workflows/verify.yml:6`)で走る本物のプロセスのテストが見る。仕様の FR-7 はこの道を「案」と書く。`lsof` がない環境では読みは「読めない」になり、reseat は断らない
- 止めてからゲートが起動するまでに別のプロセスがサーバーを起動した場合(verify の I6)は、防がず、起動し直したあとの読み直しと `note:` で知らせるだけにした(6 節)
- 計画の本文に古い名前が残る。Scope の `reseatRetryCommand` は、53c12663 で消えて `reseatRetryCommandWithHerdr` だけになった(`TestReseatRetryCommand`、`reseat_test.go:1962` も名前が古いまま)
- `ServerStop` の成功の応答は、herdr 0.7.5 が何も出さないので読んでいない。将来の herdr が成功で何かを出しても、exit 0 で error envelope でなければ成功と読む
- test report は d05fa43e で pass(`./scripts/run-test.sh` が rc 0、shell 46 本、Go 11 パッケージ、1,359 秒)。そのあとに origin/main の 29dd00fd を取り込み(b7c41aa3)、テストだけの 261df0c9 と、help の 1 文と pin の 5799b25c を足した。この 2 つのあとは `./scripts/run-test.sh` を回しておらず、cross-review の triage は 5799b25c のあとの `go test -run 'Reseat|Help'` が ok だったと書く。`internal/` は 5799b25c から HEAD まで変わっていない(`git diff --quiet 5799b25c HEAD -- internal/` で確かめた)
- 私はテストを回していない。上の結果は報告から読んだもの
