# Verify report: org-director-launch-cmd

- Date: 2026-10-16
- Plan: docs/plans/active/2026-10-16-org-director-launch-cmd.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 2aeecc48d0ba で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-9 の起動 (B) のうち `[org.director].launch_cmd` の部分、8 段目の 8a-2b)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 1)
- Scope: `git diff 3723d900...HEAD`(HEAD 8ad4275d、15 コミット、46 ファイル、+8210/-513)。仕様適合(AC1〜AC10、D1〜D12、FR-9 の起動 (B) とセキュリティの節)、self-review の M1・L1〜L8・RC-L1・RC-L2 の直し(98131af5、8ad4275d)、文書のずれ、静的解析、HEAD の binary と偽の端末での観察。テストは走らせていない(/test の受け持ち)。テストはコードを読み、どの AC をどう固定しているかを確かめた。行番号は HEAD 8ad4275d のもの
- Evidence: 生の出力は `docs/evidence/verify-2026-10-16-org-director-launch-cmd.log`(run-static-verify の要約、Go の検査、probe 1〜4 のスクリプトと出力。`$PROBE` は scratchpad の probe のディレクトリ、`~` は `$HOME`)、wrapper の全ログは `docs/evidence/verify-2026-10-16-025354.log`(どちらも gitignore の対象)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 launch_cmd の既定は空で、空なら herdr。3 種の値を読み込みで断り、エラーはキーと理由を言う | 満たす(binary でも確認) | `OrgDirectorConfig.LaunchCmd`(`internal/config/config.go:202`)は Default で nil。`Load` が `ValidateDirectorLaunchCmd`(`:251`)を呼ぶ(`:570`)。start の分かれは `checkDirectorStart` の `cmdLaunch: len(o.Config.Director.LaunchCmd) > 0`(`internal/org/director_start.go:741`)。probe 1: `ralph org director status --config` で、空白だけの最初の要素、`{home}`、`{ a; }`、`{script}` も `{script_sh}` もない値の 4 つが終了コード 1 で、どれも `[org.director].launch_cmd:` で始まり理由を言う。対照の `sh {script_sh}` の値と `launch_cmd = []` は通る。断ったとき state dir は作られない。テスト(読んだだけ): `TestLoad_OrgDirectorLaunchCmd`、`TestLoad_OrgDirectorLaunchCmdWrongType`、`TestValidateDirectorLaunchCmd_EmptyIsValid`、`TestTemplateRalphToml_LaunchCmdExampleIsValid`。空のとき herdr の道が 8a-2 のままであることは既存の herdr のテストが固定する(/test で確かめる) |
| AC2 start の順(attached → 0700 の起動スクリプト → シェルなしの argv → check-in を待つ → stdout)、agmsg home の export、manifest に書かない | 満たす(binary でも確認) | `startCmd`(`internal/org/director_cmd.go:534`)が prompt、起動スクリプト(`writeDirectorLaunchScript`、`:334`)、argv(`directorLaunchArgv`、`:375`)、`runDirectorLaunchCmd`(`:433`、`exec.CommandContext(ctx, argv[0], argv[1:]...)` の `:440` でシェルを通さない)、`waitForCheckIn`(`:575`)の順に進む。probe 2(下の観察 B2): `director.jsonl` は `attached`(`launch":"cmd"`・token・model・`permission_mode`)、次に `launched`(`pid`・`start_time`)。起動スクリプトは `-rwx------`、`launch/` は `drwx------`。中身は `cd '<main root>' \|\| exit 1`、`RALPH_DIRECTOR_TOKEN` の export、`RALPH_ORG_AGMSG_HOME` の export、check-in、agent の絶対パスへの `exec` で、どの値も単一引用符。stdout は `started d1 (driver=claude, model=sonnet, permission=guarded, launch=cmd, name=director_<token>, pid=<pid>)`、端末で話す行、stop の打ち方の 3 行。repo/sub で `RALPH_ORG_AGMSG_HOME=rel/agmsg` を付けて打つと、スクリプトと偽の agent の env は `$PROBE/repo/sub/rel/agmsg`(start を打った場所が基準。director の cwd は `$PROBE/repo`)。env がない probe 4 のスクリプトには `RALPH_ORG_AGMSG_HOME` が 0 回で、偽の agent には `<unset>` が届いた。台帳に `manifest.jsonl` と `model-receipts.jsonl` はない |
| AC3 check-in(`ralph org director launched`)は条件がそろうときだけ `launched` を書き、ほかは終了コード 1 で何も書かず agent を起動しない。`ralph org --help` に出ない | 満たす(一部はテストのコードで確認) | store の `MarkCmdLaunched`(`internal/org/director_start.go:322`)はロックの下で nil、token、launch cmd、stopped、current、starting の順に断る。CLI の `directorLaunched`(`internal/cli/org_director_headless.go:700`)が先に親の pid(`os.Getppid`)と ps の `alive` を確かめる。`Hidden: true`(`:672`)。probe 1: `ralph org --help` の `launched` は 0 件、`ralph org director --help` の Available Commands に出ない、`director launched --help` は読める、フラグなしは終了コード 1。probe 2 と probe 3 F2b: stop か start の後始末のあとに起動スクリプトを手で打つと `is already recorded stopped` で終了コード 1、`director.jsonl` の行数も偽の agent の印の数も変わらない。親の pid が違う場合と ps を読めない場合の断りは `TestOrgDirectorLaunched_Refusals`(9 行)だけが固定する(/test で確かめる) |
| AC4 launch_cmd の失敗、時間切れ、check-in の時間切れ、待つあいだの `attach --replace` で終了コード 1、端末を人が閉じると言い、記録は 3 通り | 満たす(一部はテストのコードで確認) | `abortCmd`(`internal/org/director_cmd.go:622`)と `detachCmdOnly`(`:657`)。probe 3: F1(`exit 3`)、F2(スクリプトを打たない launch_cmd と `--timeout-ms 1500`)、F5(戻らない launch_cmd、10 秒で `did not return in time and was killed`)はどれも終了コード 1 で、`attached` → `detached` → `stopped` の順に書かれ、出力は `the terminal that [org.director].launch_cmd may have opened (name director_<token>) is left open for you to close` で終わる。F4(check-in のあとで `exit 3`)は `launched` のあとに後始末が偽の agent に SIGTERM を送り(`start sent SIGTERM to the launch script or agent that checked in (pid N), which ended`)、`detached` と `stopped` を書き、そのプロセスは残っていない。`detached` だけを書く道、待つあいだの replace、check-in の直後で止めた起動スクリプト、後始末と重なる check-in は `TestStartDirector_ACleanupThatCannotConfirmTheEndLeavesItDetachedOnly`、`TestStartDirector_AReplaceWhileWaitingForTheCheckInEndsItsOwnLaunch`、`TestStartDirector_TheCleanupStopsALaunchScriptPausedAfterItsCheckIn`、`TestStartDirector_ACheckInDuringTheCleanupIsReadAgainAndStopped`、CLI の `TestOrgDirectorStart_ALaunchCmdCleanupErrorNamesTheStopAndTheTerminal` が固定する(/test で確かめる) |
| AC5 8a-2 の断りを launch_cmd でも当て、何も書かず launch_cmd を打たない。agent が PATH にないときも断る。`HERDR_PANE_ID` では断らない | 満たす(一部はテストのコードで確認) | `checkDirectorStart`(`internal/org/director_start.go:740`)の `!c.cmdLaunch && o.getenv(herdrPaneIDEnv) != ""`(`:751`)。`checkDirectorLaunchCmd`(`internal/org/director_cmd.go:475`)は最初の読みと見張りの関門より前に走る(`director_start.go:778`)。probe 3 F3: `--driver codex` で codex が PATH にないと `the codex agent is not on PATH` で終了コード 1、`director.jsonl` は 0 行増、launch_cmd が作るはずの印のファイルはない。probe 2: `HERDR_PANE_ID=probe-pane` を付けた start が通った。ほかの断り(enabled = false、codex の autonomous と edits、promoted と starting の登録、見張りの関門、読めない `director.jsonl`)は `TestStartDirector_LaunchCmdKeepsTheOtherRefusals`(7 行)、`--config` の相対の agmsg_home は CLI の `TestOrgDirectorStart_ARelativeAgmsgHomeIsRefusedWithALaunchCmd` が固定する(/test で確かめる) |
| AC6 stop は起動時刻が同じときだけ SIGTERM を送って待ち、`stopped`(今の登録なら `detached` も同じ write)。終わったプロセスと pid のない登録には何も送らない。端末は人が閉じる | 満たす(一部はテストのコードで確認) | `StopDirector` が launch cmd を `stopCmdDirector`(`internal/org/director_stop.go:226`、`:314`)に分け、`stopDirectorProcess`(`internal/org/director_cmd.go:199`)が `directorProcessEnded` のあとで pid だけに SIGTERM を送る(`:211`)。probe 2: stop が `stopped d1 (process=ended after SIGTERM)` を出し、偽の agent(exec のあとの sleep)は消え、`detached` と `stopped` が同じ時刻に書かれ、`the terminal ... is left open for you to close` が出た。gone、zombie、別の起動時刻は `TestStopDirector_LaunchCmdSendsNothingToAProcessThatHasEnded` の 3 つの subtest、pid のない `starting` は `TestStopDirector_ALaunchCmdStillStartingIsRecordedStoppedWithNothingSent`、exec の前の起動スクリプトは `TestStopDirector_LaunchCmdStopsALaunchScriptPausedAfterItsCheckIn` が固定する(/test で確かめる) |
| AC7 断る 3 つの場合と `--force` | 満たす(一部はテストのコードで確認) | probe 2: `RALPH_DIRECTOR_TOKEN=<token>` の stop は `--force` があってもなくても終了コード 1 で、`director.jsonl` は 2 行のまま、偽の agent は動いたまま。probe 4: SIGTERM を無視する偽の agent に stop を打つと、10 秒待って `pid N did not end within 10s of its SIGTERM; nothing was recorded`、`look with ps -o pid,lstart,command -p N`、`--force` の打ち方を出して終了コード 1、行数は変わらない。`--force` は本人と確かめたプロセスにもう一度 SIGTERM を送って 10 秒待ち、`"force":true` の `stopped` を書いて警告を出し、プロセスは残った(SIGKILL は送られていない。probe が自分で KILL した)。ps を読めない場合と、stop の読みのあとの check-in は `TestStopDirector_ALaunchCmdWhosePSCannotBeReadIsSentNothing`、`TestStopDirector_ALaunchCmdCheckInAfterTheReadRecordsNothing`、`TestDirectorMarkStoppedIfUnchanged_RefusesACheckInAfterTheRead` が固定する(/test で確かめる)。`--force` の読み方は下の I2 |
| AC8 status、`ralph status` の `director:` の行、`--json`、`stop --all` の案内 | 満たす(binary でも確認) | `orgDirectorLaunchDetails`(`internal/cli/org_director.go:607`)と `newOrgDirectorReportRegistration`(`:504`)。probe 2: `director status` と `ralph status` の行は `headless running, model=sonnet, launch cmd, name director_<token>, pid <pid>`。field の表示と `--json` は `launch`・`state`・`model`・`pid`・`start_time`・`terminal_name` を持つ。`ralph org stop --all` は `note: ... the headless director d1 (running) is still open: stop it with ralph org director stop --id d1 --state-dir ...` を出した。herdr の登録の出力が 8a-2 のままであることは既存のテストが固定する(/test で確かめる) |
| AC9 skill の 4 面、同期の検査、help・README・仕様の注・雛形のコメント・recipe の 2 面・tech-debt、「まだない」の消し忘れ | 満たす(LOW 1 件) | `./scripts/check-skill-sync.sh` は 14 skill が lock-step、`./scripts/check-sync.sh` は DRIFTED 0(run-static-verify の中)。`/director` の 4 面の md5 は `fdcb69df0bf6872770bf5f38112299f4`、`/org` の 4 面は `372383619c81d3052a6c12634bed3c91`。recipe の 2 面は `cmp` で同じ。`/director` の「まだないもの」から launch_cmd の項が消え、tech-debt の「launch_cmd is not implemented」の行も消えた。`launch_cmd` を「まだない」と書く箇所は、計画と報告を除く repo に残っていない(grep)。ずれは下の V1(Go のコメントの動詞の一覧) |
| AC10 テストは本物の herdr・osascript・端末のアプリ・人の tmux を打たず、repo の台帳に書かず、自分で起こしたプロセスにだけ signal を送る。実機の確認を evidence に残す | 満たす(テストの側はコードを読んだ範囲) | 足したテストの行で `tmux` が出るのは argv の値と TOML の文字列だけで、`runDirectorLaunchCmd` に渡すのは `/bin/sh`、`sleep`、存在しないパスだけ。CLI の `TestMain`(`internal/cli/main_test.go`)は agent の探索と ralph の実行ファイルを失敗に固定する。テストの signal は `killCheckedInProcesses` と `stopOwnBackgroundChild`(`internal/org/director_cmd_test.go:1056`、`:840`)がどちらも ps の起動時刻を確かめてから送る。実機の確認は `docs/evidence/director-launch-cmd-livefire-2026-10-16.md`(専用の socket の tmux 3.5a、claude 2.1.291 と codex-cli 0.160.1)。その binary は 88bb29fd のもので、98131af5 の直しより前(下の I4) |

FR-9 とセキュリティの節:

- 起動 (B) の「`[org.director].launch_cmd` で別の端末の起動コマンドに差し替えられる」: AC1〜AC7 と probe 2〜4、実機の evidence で満たす
- 「シェルに解釈させない」: launch_cmd は argv のまま `exec.CommandContext` に渡り(`internal/org/director_cmd.go:440`)、置き換えの値は起動スクリプトのパス、`director_<token>`、cwd だけ(`directorLaunchArgv`、`:375`)。受信箱の本文は入らない。起動スクリプトの値はすべて `shellQuote`(`text`、`:283`)
- 「Orca などの外部の端末は必須にしない」: 既定は空で、`templates/base/ralph.toml` の例はコメントのまま

### self-review の直し(98131af5、8ad4275d)

| 指摘 | Status | Evidence |
| --- | --- | --- |
| M1 相対の `RALPH_ORG_AGMSG_HOME` | 直った(binary でも確認) | `startDirector` が launch_cmd のときだけ `agmsgHomeIsRelative` で見分けて `filepath.Abs` を掛ける(`internal/cli/org_director_headless.go:319`)。probe 2 で、repo/sub から打った `rel/agmsg` が `$PROBE/repo/sub/rel/agmsg` になってスクリプトと agent に届いた(director の cwd は `$PROBE/repo`) |
| L3 `MarkCmdLaunched` の順 | 直った | token の case が launch の case より前(`internal/org/director_start.go` の switch)。probe 3 F2b の断りの文は `headless stopped, launch cmd` を名指す(token が合う場合) |
| L4 ps を読めない枝の案内 | 直った | `ErrDirectorProcessUnreadable` と `orgDirectorPSLook`(`internal/cli/org_director_headless.go:620`)。probe 4 の「終わらない」枝の文は `again once pid N has ended` と ps の打ち方を出す。読めない枝の文はテストだけが固定する |
| L5 `DirectorStartResult.Launch` | 直った | start の出力は `reg.CmdLaunch()` で分かれ、probe 2 の 3 行は launch cmd の形 |
| L1、L2、L6〜L8、RC-L1、RC-L2 | 直った | 読んで確かめた。`internal/org/verbs.go:737-744` は 72〜76 桁、`internal/org/director_cmd_test.go:877-886` は 77 桁以内 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | scope は `full fallback (unclassified:templates/base/ralph.toml)`、言語は golang。gofmt ok、golangci-lint `0 issues`。hook の shellcheck と `sh -n`、guard の awk、settings の jq、Codex hook の検査、`check-sync.sh`(PASS、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`(PASS)、tech-debt の plan の参照、`secret-scan-branch: scanned 3723d900..8ad4275d against origin/main: clean` |
| `go vet ./...`(go1.26.0) | pass | 出力なし、rc 0 |
| `gofmt -l .` | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod の Go。`%w: %w` を含めて通る |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし、rc 0 |
| `git diff --check 3723d900...HEAD` | pass | 出力なし |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | 2aeecc48d0ba |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md`(124、125、258 行) | yes | `ralph org` の行、`ralph status` の行(`launch cmd`、端末の名前、pid、`terminal_name`)、director の段落(シェルなしの argv、check-in、SIGTERM、SIGKILL なし、`RALPH_DIRECTOR_TOKEN`、recipe へのリンク)。probe の出力と合う |
| `AGENTS.md`、`docs/architecture/repo-map.md` | yes | `director_cmd.go` と launch_cmd の端末を足した。AGENTS.md の managed block は check-sync で同期 |
| 仕様の FR-9 の注(86 行) | yes | D1〜D11 の要約。85 行(8a-2 の注)の「8a-2b に送った」は当時の記録で、86 行がその続き |
| `templates/base/ralph.toml` | yes | 置き換えの 4 つ、`{script}` か `{script_sh}` が要ること、端末は人が閉じること。例は `TestTemplateRalphToml_LaunchCmdExampleIsValid` が読む |
| `docs/recipes/director-launch-cmd.md` と template の写し | yes(I1 を除く) | `cmp` で同じ。エラーの文(`did not check in within ... (--timeout-ms N)`、`again once ps can read pid <pid>`、`run ralph org director stop from a terminal other than the director's`)、start の 3 行、10 秒、stdin が空であることはコードと probe の出力に合う |
| `/director` skill の 4 面 | yes(I1 を除く) | 同じ md5。「まだないもの」から launch_cmd を外した。start、check-in、失敗、stop、`RALPH_DIRECTOR_TOKEN`、`--force` の手順はコードと合う |
| `/org` skill の 4 面 | yes | 同じ md5。書き換える動詞の一覧に `director launched`、`director` の行に launch_cmd |
| help(start、stop、status、`ralph status`、attach、detach、`ralph org`、`--config`、`stop --all`、launched) | yes(I1 を除く) | probe 1〜4 の出力と合う。start の「for up to 10 s」は `DriverCallTimeout` の既定で、CLI から変える設定はない |
| `docs/tech-debt/README.md` | yes | launch_cmd が未実装の行を消し、253・259・189 行に launch_cmd の場合を足し、267〜271 行を足した。どれも 5 列(表の見出しと同じ)。271 行の「13 件、12 ファイル」は `grep -rn 'plans/active/2026-10-15-org-director-start' internal` の 13 件・12 ファイルと一致。新しい行は計画を active のパスで指し、/pr の `archive-plan.sh` が書き換える |
| `.claude/rules/ralph/agent-messaging.md`、`ralph-workflow.md` | 変更不要 | 前者は message の形で、launch_cmd に触れない。後者の「`ralph org director start` ... `director stop`」は herdr に限らない書き方で、今も正しい |
| V1 `internal/cli/org_legacy_ledger.go:21-25` | no(LOW) | `orgLedgerMutating` の doc comment は書き換える動詞を並べるが、このブランチの隠れた動詞 `director launched` がない。`newOrgDirectorLaunchedCmd` はこの access で runtime を作る(`internal/cli/org_director_headless.go:678`)。同じ一覧の `/org` skill の 4 面(59 行)は 8d183220 で `director launched` を足した。挙動は変わらない(古い台帳に動く座席があれば check-in は断られ、start は時間切れで `stopped` を書く)。直すなら一覧に `director launched` を足す |
| 計画の AC と Progress の checkbox | 遅れている(情報) | AC1〜AC10 はすべて `[ ]`、Progress の「Verification artifact created」も `[ ]`。前の段と同じく /sync-docs か orchestrator が付ける |

情報(ずれに数えない):

- I1 ps を読めないときの「何も送らない」: stop の help(`internal/cli/org_director_headless.go:196`)、recipe(172 行)、`/director` skill(291〜292 行)は「ps を読めないとき(何も送らない)」と書く。最初の読みで ps が失敗したときはそのとおり。SIGTERM のあとの読み直しで ps が失敗したときは、SIGTERM は送ったあとで(`internal/org/director_cmd.go:224` の note `process=sent SIGTERM; ps could not be read after it`)、出力の note はそれを正しく言う。読み直しのあいだだけ ps が失敗する場合に限られ、文書が書いていないだけで、記録はどちらでも書かない
- I2 `--force` の読み方: AC7 と D7 の「まだ送っていない SIGTERM を送らずに」を、実装は 8a-2 の D11(label が合えば `--force` でも通常どおり閉じる)と同じ意味に読み、本人と確かめたプロセスには `--force` でも SIGTERM を送って待つ。Progress の S3 の注、help、recipe、skill、仕様の注(「それ以上は何も送らずに記録する」)が同じ読み方で、probe 4 の挙動とも合う
- I3 tech-debt の新しい行と 189・253・259 行が計画を `docs/plans/active/` で指すのは、/pr の前は正しい(Progress の S4 の注のとおり)
- I4 実機の evidence は 88bb29fd の binary で、そのあとの 98131af5 が start の出力(L5)、stop の案内(L4)、agmsg home(M1)、check-in の断りの順(L3)を変えた。HEAD の binary の probe 2〜4 で、evidence が引く start の行、stop の 3 行、`RALPH_DIRECTOR_TOKEN` の断り、check-in の断り、`exit 3` と時間切れの start のエラーが同じ形で出ることを確かめた。HEAD で本物の claude と codex は打っていない

## Observational checks

HEAD の binary(`go build -o $PROBE/bin/ralph ./cmd/ralph`)を、`env -i` と `$PROBE/bin` だけの PATH(ralph、git、sh、ps、nohup、sleep の symlink、偽の `claude`、呼ばれたら記録して失敗する偽の `herdr`)で打った。台帳は `$PROBE/repo/.harness/state/org`(git init した scratch の repo の main worktree のもの)、launch_cmd は main worktree の `ralph.toml` の `["sh", "-c", "nohup sh \"$1\" >/dev/null 2>&1 &", "sh", "{script}"]`(計画の Test plan の偽の端末)。偽の agent は受け取った引数、cwd、2 つの env を印のファイルに書き、`exec /bin/sleep 300` で同じ pid のまま眠る。本物の herdr、tmux、osascript、端末のアプリ、claude、codex は打っていない。

- B1(probe 1): AC1 の 4 つの断りと 2 つの対照、hidden の動詞の help。上の AC1 と AC3
- B2(probe 2): repo/sub から `RALPH_ORG_AGMSG_HOME=rel/agmsg`、`HERDR_PANE_ID=probe-pane`、`--config` なしで start。repo/sub の `./ralph.toml` には `["false", "{script}"]` を置いたが、start は main worktree の launch_cmd で通った(D10)。そのあと status、`--json`、`ralph status`、`stop --all`、`RALPH_DIRECTOR_TOKEN` の断り 2 つ、stop、遅れた check-in を打った。上の AC2、AC3、AC5〜AC8
- B3(probe 3): F1〜F5。上の AC4 と AC5
- B4(probe 4): SIGTERM を無視する偽の agent(`trap '' TERM` のあとの exec)。上の AC7
- herdr の呼び出しは probe 1〜4 を通して 0 回(見張りの分も含む)。描かれた director の雛形には `{{IF_` の残りも herdr の pane の行もなく、端末の行(19、32、37、47〜49 行)がある
- 後片付け: start が起こした見張りを 2 回とも `ralph org watch --stop` で止め、`pgrep -fl "$PROBE/"` と `pgrep -fl 'sleep 300'` が空であることを確かめた

## Coverage gaps

- テストの実行(go1.26.0 と go1.25.8、`-race`、mutation)は /test の受け持ちで、走らせていない。上の表の「テストのコードで確認」の部分(AC3 の親の pid と ps の断り、AC4 の `detached` だけの道・replace・check-in の直後の停止・後始末と重なる check-in、AC5 のほかの断り、AC6 の zombie・pid の再利用・exec の前の起動スクリプト、AC7 の ps を読めない場合と読みのあとの check-in、AC1 と AC8 の herdr の道の回帰、AC10 のテストの隔離)は、テストが通って赤になることを /test で確かめる
- Linux の `ps -o stat=,lstart=` の形と、同じプロセスを 2 回読んで同じ秒を返すか(計画の Assumptions と self-review の Not checked)。手元の macOS では probe の check-in と stop がそろって通った。CI の verify.yml は ubuntu-latest で `RALPH_VERIFY_SCOPE=full` の run-verify を走らせるので、PR の CI で見る
- HEAD の binary での本物の claude と codex(I4)、Orca と WezTerm の例(tech-debt 270 行)
- 最小の追加の確かめ: PR の CI の Linux で `TestReadDirectorProcess_AnAliveProcessKeepsItsStartTime` と `TestReadDirectorProcess_RunsPSInCAndUTC` が通ることを見れば、Linux の ps の前提(check-in と stop の本人の確かめの土台)を確かめられる

## Verdict

- Verdict: pass(V1 は Go のコメントの一覧の LOW のずれで、/sync-docs で直せる。merge を止めるものではない)
- Verified: AC1〜AC9 の本体と、AC10 の実機の evidence の存在。FR-9 の起動 (B)、シェルを通さないこと、外部の端末を必須にしないこと。self-review の M1・L1〜L8・RC-L1・RC-L2 の直し。静的解析(run-static-verify、go vet、gofmt、go1.25.8 の build と vet、`git diff --check`、skill の 4 面、recipe の 2 面、check-sync)。HEAD の binary と偽の端末での start、check-in、status、`--json`、`ralph status`、`stop --all` の案内、`RALPH_DIRECTOR_TOKEN` の断り、stop、遅れた check-in の断り、start の 4 つの失敗の道、agent が PATH にないときの断り、SIGTERM を無視するプロセスへの stop と `--force`、herdr の呼び出しが 0 回であること
- Partially verified: AC3〜AC8 のうちテストだけが固定する場合(上の Coverage gaps)、AC10 のテストの隔離(コードを読んだ範囲)
- Not verified: テストの実行、Linux の ps、HEAD の binary での本物の claude と codex、Orca と WezTerm
