# org-director-launch-cmd

- Status: Approved
- Approved: 2026-10-16 sha256:2aeecc48d0ba
- Owner: Claude Code
- Date: 2026-10-16
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 8 段目の前半(8a)の 2 本目の後半(8a-2b)。FR-9 の起動 (B) の「`[org.director].launch_cmd` で別の端末の起動コマンドに差し替えられる」。8a-2 は PR #231(3723d900)で merge 済みで、その計画(`docs/plans/archive/2026-10-15-org-director-start.md`)の D1 が launch_cmd をこの段に分けた。分かれ目は consult(consult-plan-launchcmd)の判定による。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-director-launch-cmd

## Objective

`[org.director].launch_cmd` を設定すると、`ralph org director start` は herdr の代わりにそのコマンドで端末を開き、そこで headless の director を立てる。登録は 8a-2 と同じく先に書く。起動は、端末の中の起動スクリプトが ralph に知らせた時点で記録する。`ralph org director stop --id` は、記録したプロセスがこの起動の director であることを確かめてから止める。herdr を使わない人も headless の director を持てるようになる。

## Scope

- 設定 `[org.director].launch_cmd`(文字列の配列。空なら今の herdr の起動)と、その検証
- start の launch_cmd の道:
  - 先に `attached`(`launch = "cmd"`)を書く
  - 台帳の下に起動スクリプトを書く
  - launch_cmd をシェルを通さずに打つ
  - 起動スクリプトの知らせ(check-in)を待つ
  - 失敗したら後始末する
- 隠れた動詞 `ralph org director launched`(起動スクリプトだけが打つ check-in)
- stop の launch_cmd の道: プロセスを確かめてから SIGTERM を送り、`--force` もある
- `ralph org director status`、`ralph status` の `director:` の行、`--json`、`stop --all` の案内に launch_cmd の登録も出す
- 文書: `/director` skill の 4 面、README、仕様 FR-9 の注、`templates/base/ralph.toml` のコメント、新しい recipe `docs/recipes/director-launch-cmd.md`(2 面)、tech-debt

## Non-goals

- 端末(タブ、ウィンドウ)を閉じること。stop が止めるのは director のプロセスだけで、端末は人が閉じる
- 利用者が書く止めるコマンド(`stop_cmd`)
- push の起こし口(`wake_cmd`)と director の生存の印(8b)。登録が `running` のまま agent が先に終わった場合も 8b で扱う
- 落ちた start が残した `starting` の登録の期限(8b、tech-debt の既存の行)
- herdr の起動の振る舞いの変更。launch_cmd が空なら 8a-2 のまま
- `auto_merge` と、merge のあとの gate の取り直し(8a-3)
- Windows

## Assumptions

- 起動スクリプトの最後の `exec` で、スクリプトのプロセスがそのまま agent になる。そのため check-in が記録する pid(`$$`)は agent の pid になり、exec の前と後で pid も起動時刻も変わらない(consult の指摘)
- `ps -o lstart= -p <pid>` は macOS の ps と Linux の procps の ps の両方で、プロセスの起動時刻を秒の精度で返す。出力は TZ とロケールで変わるので、ralph は `LC_ALL=C TZ=UTC` を付けて読む(手元の macOS で TZ によって変わることを確かめた。Linux は CI で確かめる)
- claude と codex は SIGTERM で終わる。未確認なので、実機で確かめる(Risks)
- launch_cmd は、端末を開いたらすぐ戻るコマンドを書いてもらう。例は `tmux -L <socket> new-session -d`、`orca terminal create`、`wezterm cli spawn`。戻らないコマンドは時間切れで止める

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`(`OrgDirectorConfig.LaunchCmd` と検証)
- `internal/org/director.go`(`DirectorLaunchCmd`、`launched` の `pid`、fold、`Headless()` の意味)
- `internal/org/director_start.go`(`StartDirector` の分かれ、`AttachHeadless` の launch、check-in の書き手、`abort`)
- `internal/org/director_cmd.go`(新規。起動スクリプト、launch_cmd の置き換えと実行、プロセスの確かめ、SIGTERM と待ち)
- `internal/org/director_stop.go`(`StopDirector` の分かれ、自分を止める断り)
- テスト: `internal/org/director_cmd_test.go`(新規)、`director_test.go`、`director_start_test.go`、`director_stop_test.go`、`internal/cli/org_director_headless_test.go`、`internal/cli/org_director_test.go`、`internal/cli/status_test.go`
- `internal/cli/org_director_headless.go`(start の出力、stop の出力、隠れた動詞 `launched`)、`internal/cli/org_director.go`(status の出力)、`internal/cli/status.go`
- `internal/org/prompts/director.md`(launch_cmd の端末で動く場合の 1 文が要るときだけ)
- `.claude/skills/director/SKILL.md` と写しの 3 面、`README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`、`templates/base/ralph.toml`、`docs/recipes/director-launch-cmd.md` と `templates/base/docs/recipes/director-launch-cmd.md`、`docs/tech-debt/README.md`、必要なら `AGENTS.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-director-launch-cmd.html`(かんたん版 3 枚: overview、easy-stop、easy-late。折りたたみに detail-overview、detail-start、detail-states)
- 自己点検: `plan-visual.sh lint` が exit 0。全体と 6 枚の図を撮って読んだ。detail-states の `(--force)` の書き方と下の枠の余白を直し、撮り直して確かめた。Codex の計画助言で D4・D6・D7 を直したあと、easy-stop、detail-start、detail-states の文言を起動時刻の確かめと `detached` だけの後始末に合わせ、lint と撮り直しで確かめた

## Design decisions

- D1 launch_cmd の形: 文字列の配列(argv)で、シェルを通さずに exec する。置き換えは決まった集合だけにし、ほかの `{...}` は設定の検証エラーにする(consult の F1)
  - `{script}`: 起動スクリプトの絶対パス
  - `{script_sh}`: 同じパスをシェル用に単一引用符で囲んだもの。tmux の shell-command と Orca の `--command` は文字列をシェルが読むので要る
  - `{name}`: `director_<token>`
  - `{cwd}`: director を動かすディレクトリ(台帳の main worktree のルート)

  `{script}` か `{script_sh}` のどちらもなければ検証エラーにする。起動スクリプトが走らなければ、start は必ず時間切れになるため。仕様のセキュリティの節(「シェルに解釈させない」)に従う。受信箱の本文は雛形に入らない
- D2 選び方: launch_cmd が空でなければ、start はいつもそれを使う(consult の F4。仕様の「差し替えられる」)。登録の `launch` は `cmd` になる。`--launch` のフラグは足さない
- D3 起動スクリプト: start は台帳の下の `launch/director_<token>.sh`(0700)に、次の 3 つを書く。どの値も単一引用符で囲む(既存の `directorShellWord` の形)
  - `{cwd}` に `cd` し、`RALPH_DIRECTOR_TOKEN=<token>` を export する。start を打った env に `RALPH_ORG_AGMSG_HOME` があれば、その値も export する。相対の値は、start を打ったディレクトリから絶対パスにしてから書く(見張りと同じ agmsg を指すため。self-review の M1)
  - check-in: start を打った ralph の絶対パス(`os.Executable`)で `ralph org director launched --id <id> --token <token> --pid $$ --state-dir <台帳の絶対パス>` を打つ。失敗したら exec せずに終わる。このパスは `Org` のフィールドで差し替えられ、テストは check-in のあとで止まる helper process を使う
  - `exec <agent の絶対パス> <permission の引数> --model <m> <雛形を指す 1 文>`。agent の絶対パスは、start の env で `exec.LookPath` が見つけたもの。見つからなければ start は何も書かずに断る。引数の並びは herdr の agent start に渡すものと同じ

  launch_cmd の端末の env と cwd は ralph からは決められないので、スクリプトで決める。8a-2 の D13 で herdr の pane に届かなかった `RALPH_ORG_AGMSG_HOME` は、launch_cmd の道では届く
- D4 check-in: 隠れた動詞 `ralph org director launched` は、`director.lock` の下で次のすべてを確かめたときだけ、`launched`(`pid` と `start_time` 付き)を書く。どれかが違えば何も書かずに終了コード 1 で終わり、スクリプトは exec しない(consult の F2)
  - 登録が今の登録で、`launch = "cmd"`、`starting` のまま
  - token が合う
  - pid が正の数で、check-in の親のプロセス(`os.Getppid()`)の pid と同じ
  - その pid の起動時刻を D7 の読み方で読める。読んだ値を `start_time` に記録する

  stop や start の後始末が先に `stopped` を書いていれば、遅れて開いた端末の check-in は断られるので、agent は起動しない。ralph の絶対パスがあとで消えたとき(brew upgrade など)も check-in が失敗するだけで、start が時間切れで `stopped` を書くので、記録は事実と食い違わない(consult)
- D5 start の時間: launch_cmd の実行は `DriverCallTimeout`(既定 10 秒)で区切る。check-in を待つのは `--timeout-ms`(既定 60 秒、herdr の agent start と同じ値)で区切る。新しい設定は足さない(consult の F7)。待つあいだは台帳を短い間隔で読み直す。`launched` が書かれたら成功、登録が今の登録でなくなったか `stopped` になったら後始末に進む
- D6 start の後始末(launch_cmd の道): launch_cmd の失敗(終了コードが 0 でない、時間切れ)、check-in の時間切れ、待つあいだの `attach --replace` のどれでも、start は stop と同じ手順で後始末する
  - 登録に pid があれば(check-in が先に済んでいれば)、D7 の確かめのあとで SIGTERM を送り、終わったと確かめる。check-in と exec のあいだの起動スクリプトも、exec のあとの agent も、同じ確かめで本人とみなす(Codex の計画助言の指摘)
  - pid がないか、終わったと確かめたら、ロックの下で、読んだ pid と今の pid を比べて `stopped`(今の登録なら `detached` も)を書く。check-in が後始末と重なって pid が増えていたら、もう 1 回だけ読み直してその pid を止める(pid は空から値にしか変わらない)
  - 終わったと確かめられないとき(ps を読めない、時間内に終わらない)は `stopped` を書かない。8a-2 の `abort` が閉じきれなかったときと同じく `detached` だけを書いて id を開いたままにし、`*DirectorCleanupError` に `ralph org director stop --id <id>` の打ち方を載せて終了コード 1 で終わる。start と attach は通り、stop はあとでその id を見つけられる(consult)
  - 出力では、開いたかもしれない端末は人が閉じること、その端末のスクリプトは check-in が断られて自分で終わることを言う
- D7 stop の確かめ方(launch_cmd の道。8a-2 の D9 の label の確かめの代わり): 記録した pid のプロセスが生きていて、その起動時刻が `launched` の `start_time` と同じときだけ、この起動の director とみなす。起動時刻は check-in と同じく `LC_ALL=C TZ=UTC ps -o lstart= -p <pid>` で読み、前後の空白を落として比べる。exec は pid も起動時刻も変えないので、exec の前の起動スクリプトにも後の agent にも同じ確かめが効き、agent が process title を書き換えても影響を受けない(Codex の指摘と consult の判定。argv の照合は exec の前に合わないので使わない)
  - 確かめたら SIGTERM を送り、プロセスが終わるのを決まった時間(新しい定数 10 秒。テストは `Org` のフィールドで縮める)だけ待ってから `stopped` を書く
  - プロセスがもうないか、zombie(`ps -o stat=` が `Z` で始まる)なら、終わったとみなして進む(herdr が知らない pane と同じ)。同じ pid に起動時刻の違うプロセスがいるときも、記録したプロセスは終わったとみなす(pid の再利用)。ただし後始末の確かめ(D6)と stop のどちらでも、そのプロセスには何も送らない
  - ps を読めないとき、時間内に終わらないときは、`--force` がなければ断る(終了コード 1)。SIGKILL は送らない
  - `--force` は、まだ送っていない SIGTERM を送らずに、`force` 付きの `stopped` を書く(8a-2 の D11 と同じ意味)
  - まだ pid のない登録(`starting`)には何も送らず、`stopped` を書く。遅れた check-in は断られる
  - ロックの下で、読んだ pid と今の pid を比べ、違えば何も書かずに断る(8a-2 の D14 の比べ方を pid に当てる)
  - 端末は閉じない。出力で、人が閉じることを言う
- D8 自分を止める断り: env の `RALPH_DIRECTOR_TOKEN` が登録の token と同じなら、stop は断る。D3 のスクリプトがこの env を渡すので、director が自分の登録に stop を打つと、記録の前に自分のプロセスが終わってしまう。8a-2 の `HERDR_PANE_ID` の断りと同じ考え方
- D9 start の `HERDR_PANE_ID` の断りは、launch_cmd の道では当てない(consult の F5)。8a-2 で断ったのは、herdr の pane の中から herdr を打つと後始末で困るためで、launch_cmd の道は herdr を打たない
- D10 そのまま使うもの: codex の director は guarded だけ(8a-2 の D12)、claude の permission の写し方、`--config` の相対の `[org].agmsg_home` の断り(8a-2 の D13。規則は 1 つのままにする。consult)、見張りの確かめと起動(8a-2 の D5)、manifest と receipts に書かないこと(8a-2 の D2、D8)。launch_cmd は `[org.director].enabled` と同じ規則で読む: `--config` がなく台帳が main worktree のものなら、main worktree の `ralph.toml` から取る(`withMainWorktreeDirectorStartConfig`)。feature org の worktree の `ralph.toml` を書き換えても、ralph が打つコマンドは変わらない(S1 の implementer の指摘で足した)
- D11 `Headless()` の意味: herdr でも cmd でも、`ralph org director start` が書いた登録を headless とする。stop、`stop --all` の案内、status は両方を扱う。herdr だけの処理(label の確かめ、pane と workspace を閉じる)は `launch = "herdr"` で分ける
- D12 範囲: recipe は端末ごとに 1 例だけ書く(tmux、Orca、WezTerm)。スライスは「設定の検証 → start と check-in → stop → 文書」の順にする(consult の F6)

Critical forks: F1〜F7 を consult(consult-plan-launchcmd)に聞き、判定は「直してから進める」だった。stop の確かめを雛形のパスで行うこと(D7)と、時間の値を既存の値から取ること(D5)を直して、ほかは自分の案のまま決めた(ユーザーの指示「計画の分かれ目も自分で選ぶ」)。そのあと Codex の計画助言が「check-in と exec のあいだの起動スクリプトを、雛形のパスの照合では本人と認められない」と指摘した(HIGH)。同じ consult に聞いて、本人の確かめを pid と起動時刻に替え(D4、D7)、終わったと確かめられない後始末は `detached` だけを書く形にした(D6)。

## Acceptance criteria

- [x] AC1 `[org.director].launch_cmd` は既定で空で、空なら start は 8a-2 と同じく herdr で起動する。空でない値のうち次のものは、`ralph.toml` の読み込みで断る。断るエラーはキーの名前と理由を言う
  - 最初の要素が空白だけ
  - 決まった集合にない `{...}` を含む
  - `{script}` も `{script_sh}` も含まない
- [x] AC2 偽の launch_cmd での start は、次の順に進む
  - まず `director.jsonl` に `launch = "cmd"`・`token`・`model`・`permission_mode` 付きの `attached` を書く
  - 台帳の `launch/director_<token>.sh` を 0700 で書く。中身は `cd`、`RALPH_DIRECTOR_TOKEN` の export、check-in、agent の絶対パスへの `exec` で、どの値も引用する
  - launch_cmd の置き換えた argv をシェルを通さずに打つ
  - check-in の `launched`(pid と `start_time` 付き)を待ち、stdout に `started d<N>`、launch、端末の名前 `director_<token>`、pid、stop の打ち方を出す

  start を打った env に `RALPH_ORG_AGMSG_HOME` があれば、スクリプトはその値を export し、なければ export しない。相対の値は start を打ったディレクトリから絶対パスにしてから書く。manifest には何も書かない
- [x] AC3 `ralph org director launched` は、登録が今の登録で、`launch = "cmd"`、`starting`、token が合い、pid が正で check-in の親の pid と同じで、その起動時刻を読めたときだけ、pid と `start_time` 付きの `launched` を書く。そうでなければ何も書かずに終了コード 1 で終わる。断られたとき、起動スクリプトは agent を起動しない(テストでは偽の agent が走らないことを見る)。`ralph org --help` には出ない
- [x] AC4 launch_cmd の終了コードが 0 でないとき、時間切れのとき、`--timeout-ms` のうちに check-in がないとき、待つあいだに `attach --replace` されたときは、start は終了コード 1 で終わり、出力は開いたかもしれない端末を人が閉じることを言う。記録は次のとおり
  - pid がなければ `stopped`(今の登録なら `detached` も)を書く
  - check-in が後始末と重なって pid が記録されていたら、D7 の確かめのあとでそのプロセスに SIGTERM を送り、終わったと確かめてから同じく記録する。check-in と exec のあいだの起動スクリプトにも同じく送る(テストは check-in の直後で止めた helper で確かめ、偽の agent が起動しないことを見る)
  - 終わったと確かめられなければ(ps を読めない、時間内に終わらない)`detached` だけを書き、出力に `ralph org director stop --id <id>` の打ち方を出す
- [x] AC5 start は、8a-2 の断り(`enabled = false`、codex の autonomous と edits、今の登録がある、見張りの関門、`--config` の相対の agmsg_home など)を launch_cmd の道でも当て、何も書かず、launch_cmd を打たない。driver の agent が start の env の PATH にないときも、同じく何も書かずに断る。`HERDR_PANE_ID` があっても、launch_cmd の道では断らない
- [x] AC6 launch_cmd の登録への `ralph org director stop --id <id>` は次のように動く
  - 記録した pid のプロセスの起動時刻が `start_time` と同じときだけ SIGTERM を送り、終わるのを待ってから `stopped`(今の登録なら `detached` も同じ write で)を書く。exec の前の起動スクリプトにも後の agent にも同じく効く
  - プロセスがもうないか、zombie か、同じ pid で起動時刻の違うプロセス(テストでは別のプロセスの pid と、ずらした `start_time`)なら、何も送らずに `stopped` を書く
  - pid のない `starting` の登録には何も送らずに `stopped` を書き、そのあとの check-in は断られる
  - 出力は、端末を人が閉じることを言う
- [x] AC7 launch_cmd の登録への stop は、次のときに断り(終了コード 1)、何も書かない
  - `--force` がなく、ps を読めない、決まった時間のうちに終わらない。ps を読めなければ SIGTERM を送らず、SIGKILL はどの場合も送らない
  - env の `RALPH_DIRECTOR_TOKEN` が登録の token と同じ
  - stop が読んだあとに check-in が pid を記録した(D14 と同じく、もう一度 stop を打つよう出す)

  `--force` は、まだ送っていない SIGTERM を送らずに、`force` 付きの `stopped` を書く
- [x] AC8 `ralph org director status`、`ralph status` の `director:` の行、`--json` は、launch_cmd の登録なら launch `cmd`、状態(`starting` か `running`)、端末の名前、pid、model を出す。`ralph org stop --all` の案内は launch_cmd の閉じていない登録も数える。herdr の登録の出力は 8a-2 のまま
- [x] AC9 `/director` skill の 4 面が同じ中身で、`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る。help、README、仕様 FR-9 の注、`templates/base/ralph.toml` のコメント、recipe の 2 面、tech-debt が launch_cmd の動きを説明し、「launch_cmd はまだない」と書いた箇所が残っていない
- [x] AC10 テストは本物の herdr、osascript、端末のアプリ、人の tmux を打たず、repo の台帳に書かない。プロセスを確かめて止めるテストは、テストが自分で起こしたプロセスにだけ signal を送る。実機の確認は、専用の socket の tmux server(`tmux -L <専用の名前>`)と隔離した台帳で、claude と codex の start と stop を打つ(evidence に残す)

## Implementation outline

1. S1 設定: `OrgDirectorConfig.LaunchCmd`、検証(AC1)、`templates/base/ralph.toml` のコメント。ralph-config.sh には足さない(シェルの側は読まない)
2. S2 start と check-in: `DirectorLaunchCmd`、`launched` の `pid`、fold、`Headless()`(D11)、`director_cmd.go` の起動スクリプトと launch_cmd の実行、check-in の書き手と隠れた動詞、待ちと後始末、start の出力(AC2〜AC5)
3. S3 stop: プロセスの確かめ、SIGTERM と待ち、`--force`、D14 の pid の比べ、自分を止める断り、status と `stop --all` の表示(AC6〜AC8)
4. S4 文書: skill の 4 面、README、仕様の注、recipe の 2 面、tech-debt(AC9)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、golangci-lint、secret scan)、`go vet ./...`、`GOTOOLCHAIN=go1.25.8` の build と vet、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: FR-9 の起動 (B) の「`[org.director].launch_cmd` で別の端末の起動コマンドに差し替えられる」を AC1〜AC7 と実機の確認で、セキュリティの節の「シェルに解釈させない」を AC1 と AC2 で、原則の「Orca などの外部の端末は必須にしない」を AC1(既定は空)で確かめる
- Documentation drift to check: `/director` skill の「まだないもの」から launch_cmd を外したか。tech-debt の launch_cmd の行を消したか、書き換えたか。README と仕様の注が herdr だけを書いていないか
- Evidence to capture: 実機の確認(専用の socket の tmux と隔離した台帳)で打ったコマンドと出力、`director.jsonl` の行、`ps` の出力。`$HOME` は `~` に置き換える

## Test plan

- Unit tests: launch_cmd の検証の表(AC1)、置き換え(`{script_sh}` の引用、空白と単一引用符を含むパス)、起動スクリプトの中身(引用、export の有無)、check-in の書き手の断りの表(AC3。親の pid と違う pid を含む)、fold の `launched` の pid と `start_time`、`Headless()`、起動時刻の読み方(`LC_ALL=C TZ=UTC`、前後の空白、zombie)
- Integration tests:
  - start: 偽の launch_cmd(`sh -c 'nohup sh "$1" >/dev/null 2>&1 &' sh {script}`)と偽の agent(起動したら印のファイルを書いて眠るシェルスクリプト)で、start、check-in、stop を通す
  - stop が本物の `ps` と SIGTERM で偽の agent を止めること(exec のあと)
  - check-in の直後で止まる helper(`Org` のフィールドで差し替えた check-in。書いたあと解除のファイルを待つ)のあいだに stop を打ち、解除しても偽の agent の印ができないこと。同じ形で、start の後始末(launch_cmd の終了コードが 0 でない)でも確かめる
  - 別のプロセスの pid と、ずらした `start_time` では SIGTERM を送らないこと
- Regression tests: launch_cmd が空なら herdr の道が 8a-2 のテストのまま通ること。promoted session の登録の扱いが変わらないこと
- Edge cases:
  - check-in と後始末の重なり(後始末のあいだに check-in が来る)
  - stop が読んだあとの check-in
  - 遅れて開いた端末の check-in が断られる
  - SIGTERM を無視する偽の agent(時間切れで断る、`--force` で記録する)
  - 終わっているプロセス
  - pid の再利用(起動時刻が合わない)
  - `{script_sh}` のパスに単一引用符がある
  - launch_cmd が戻らない(時間切れ)
- Evidence to capture: `go test` の出力(go1.26.0 と go1.25.8)、`-race`、mutation の結果

## Risks and mitigations

- pid の再利用で別のプロセスを止める: 起動時刻が check-in で記録したものと同じときだけ送る(D7)。起動時刻は秒の精度なので、同じ秒の中で pid が使い回されると見分けられないが、記録したプロセスが終わってから同じ秒のうちに同じ pid が出る必要があり、実際上は起きないと見る(consult)
- check-in と exec のあいだに後始末や stop が来る: pid と起動時刻は exec で変わらないので、起動スクリプトのうちに SIGTERM で止まり、agent は起動しない(Codex の計画助言。check-in の直後で止めたテストで確かめる)
- 後始末で終わったと確かめられない: `stopped` を書かずに `detached` だけを書き、stop の打ち方を出す。記録が事実より先に「止めた」と言うことはない
- `ps -o lstart=` の出力の形が環境で違う: `LC_ALL=C TZ=UTC` を付けて読み、同じ機械の同じ読み方の値どうしを比べる。Linux の形は CI で確かめる。busybox の ps は `lstart` を持たないが、その環境は対象にしない(読めなければ check-in が断り、start は時間切れで止める)
- claude や codex が SIGTERM で終わらない: 終わらなければ断るだけで、記録は事実と食い違わない。実機で確かめる
- launch_cmd が戻らない(`wezterm start` など): `DriverCallTimeout` で止めて後始末する。recipe には戻る形だけを書く
- ralph の絶対パスが check-in の前に消える: check-in が失敗し、start が時間切れで `stopped` を書く
- agent が check-in のあとで起動に失敗する(exec の失敗): 登録は `running` のまま残り、stop はプロセスがないので `stopped` を書く。生存の印は 8b
- 端末が開いたまま残る: 出力と recipe で、人が閉じることを言う
- launch_cmd は利用者が書いたコマンドを ralph が打つ: 受信箱の本文や座席の出力は雛形に入らない。値は argv として渡し、シェルを通さない(D1)

## Rollout or rollback notes

- 既定は空なので、launch_cmd を書かない人の動きは 8a-2 と同じ
- 戻すときは、launch_cmd の登録を `director stop` で止めてから revert する。8a-2 の読み手は `launch = "cmd"` の登録を headless とみなさない

## Open questions

- なし(F1〜F7 は consult の判定で決めた)

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved(ユーザーの指示「作成したプランは自動承認します」による自動承認。Codex の計画助言 1 件(HIGH)は D4・D6・D7 に反映済み)
- [x] Branch created
- [x] Implementation started
  - S1 設定と検証: 4c3f5fb9。config が置き換えの規則(`ExpandDirectorLaunchArg`)と検証(`ValidateDirectorLaunchCmd`)も公開した
  - S2 は大きいので 2 つのコミットに分けた。S2a(記録の形、check-in の書き手と隠れた動詞、起動時刻の読み方、SIGTERM と待ち、起動スクリプトと argv の組み立て): f8d4db99。S2a のあいだ、StopDirector は launch cmd の登録を断る(S3 で置き換える)。S2b は start の launch_cmd の道、後始末、CLI の出力、設定の読み方、雛形
  - D10 に「launch_cmd は main worktree の `ralph.toml` から読む」を足した(S1 の implementer の指摘)。図に関わらないので図はそのまま、digest を取り直して自動承認し直した(3edb71508e34 → af95b6db6074)
  - S2b(start の launch_cmd の道、check-in の待ち、後始末、CLI の出力、main worktree からの launch_cmd の読み込み、雛形の `{{IF_HERDR}}` / `{{IF_CMD}}` の行): 2f7962c9。CLI のテストは TestMain で agent の探索と ralph の実行ファイルの取得を失敗に固定し、本物の claude や codex を起動スクリプトから動かさない。偽の launch_cmd には `env -u RALPH_ORG_AGMSG_HOME` を足した(テストの env が偽の端末に漏れて、agmsg home を渡さない変異が残ったため)
  - S3(stop の launch_cmd の道、`RALPH_DIRECTOR_TOKEN` の断り、status と `ralph status` と `--json` と `stop --all` の案内、start と stop の help): 4a9c44f8。`--force` は、本人と確かめたプロセスには普段どおり SIGTERM を送って待ち、終わらなければ force 付きで記録する。SIGTERM の前に ps を読めなければ何も送らない(D11 の「確かめられれば `--force` でも普段どおり閉じる」と同じ意味)。JSON の `terminal_name` は CLI の表示用の型にだけ置き、記録の型は変えていない。このブランチのコメントが指す計画のパスを、計画 ID の形に直した
  - S4(文書: `/director` と `/org` の skill の 4 面、README、仕様の注、recipe の 2 面、tech-debt、AGENTS.md、repo-map、`ralph org stop` の help): a6dda336。tech-debt の行は計画を active のパスで指す(`verify.local.sh` が存在しないパスで落ちるため。archive への書き換えは /pr の archive-plan.sh が行う)。そのあと `ralph status` の help の「which closes it」と、`/org` skill の書き換える動詞の一覧の `director launched` の抜けを直した
  - 実機の確認(AC10): `docs/evidence/director-launch-cmd-livefire-2026-10-16.md`。専用の socket の tmux 3.5a と隔離した台帳で、claude 2.1.291 と codex-cli 0.160.1 の start、check-in、status、`RALPH_DIRECTOR_TOKEN` の断り、stop を通した。どちらも、起動の確認画面の状態でも会話中でも SIGTERM で終わる。codex は npm の入口(node)が本体の子に SIGTERM を渡す。失敗の道は、端末を開かない launch_cmd(check-in の時間切れ、あとから打った起動スクリプトは断られる)と、終了コード 3 の launch_cmd。director が自分で打つ `ralph` は agent の shell の PATH で決まり、開発用のビルドで start を打つと古い版になる(既存の tech-debt の行に足した)
- [x] Review artifact created
  - self-review cycle 1: `docs/reports/self-review-2026-10-16-org-director-launch-cmd.md`(MEDIUM 1、LOW 8)。全件を 98131af5 で直した。M1 は、相対の `RALPH_ORG_AGMSG_HOME` を start を打ったディレクトリから絶対パスにしてから起動スクリプトへ書くもの。org 層の起動スクリプトも、絶対パスでも `~/` 始まりでもない値を断る。L4 で ps を読めない枝の案内を分けた(`ErrDirectorProcessUnreadable`)。D3 と AC2 にこの絶対化を書き足したので、digest を取り直して自動承認し直した(af95b6db6074 → 2aeecc48d0ba)。`defaultDriverCallTimeout` の doc に director の呼び出しを足した(implementer が範囲外で見つけたもの)
  - reviewer の確認で cycle 1 の 9 件は直ったと判定され、新しい LOW 2 件(コメントの折り直し)は 8ad4275d で直した
- [x] Verification artifact created
  - verify cycle 1: `docs/reports/verify-2026-10-16-org-director-launch-cmd.md`(pass、LOW 1、情報 3)。V1(`orgLedgerMutating` の動詞の一覧に `director launched`)と I1(SIGTERM のあとの待ちで ps を読めなくなったときは、SIGTERM はもう送ってある。help、recipe、skill の文)は 0f5fbf89 で直した。同じ誤りの `ErrDirectorProcessNotEnded` の doc と、この Progress の S3 の文も直した
- [x] Test artifact created
  - test cycle 1: `docs/reports/test-2026-10-16-org-director-launch-cmd.md`(pass)。go1.26 と go1.25.8、`-race`、繰り返し、ubuntu の container で通った。mutant 81 個のうち生き残った O33 と O35 は、6b9b27d7 のテスト 2 本で閉じた。Linux で時計が step したあとの lstart は確かめていない(tech-debt に足す)
- [ ] PR created
