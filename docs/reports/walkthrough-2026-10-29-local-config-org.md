# Walkthrough: local-config-org(個人用の設定ファイル `ralph.local.toml` を足し、`RALPH_ORG_AGMSG_HOME` を読まなくする)

- Date: 2026-10-30(plan の日付は 2026-10-29)
- Plan: docs/plans/active/2026-10-29-local-config-org.md(/pr で archive に移す)
- Issue: #276(`Refs`。#276 を 3 本に分けた 1 本目 A1。A2 は `RALPH_ORG_STATE_DIR`、B は標準フローの値とスクリプトで、B が `Closes #276`)
- PR: #287
- Branch: feat/local-config-org(この walkthrough の前の HEAD 6460c420。merge を除く 32 commits と、origin/main を取り込んだ merge commit d5386fb7。`git diff origin/main...HEAD` で 78 files、+3272/-988。内訳は Go の本番コード 19 files(+559/-359)、Go のテスト 29 files(+1519/-356)、残りが skill の写し・文書・設定・記録)
- Base: 5eba4013(main、「Merge pull request #286」= leader-plan-depth)。分岐元は 7522835c で、S5 のあとに main を d5386fb7 で取り込んだ
- 読む範囲: 動きを決めるのは新しい 2 つの package(`internal/gitroot/gitroot.go`、`internal/config/local.go`)、`config.Load`(`internal/config/config.go:695`)、`driver.ResolveAgmsgHome`(`internal/org/driver/agmsg.go:157`)、watch の起動の拒否(`internal/cli/org_watch.go:405`)、doctor の warn(`internal/cli/doctor.go:853`)。テストの差分の大半は 2 種類で、新しいテスト(`internal/config/local_test.go` 570 行、`internal/gitroot/gitroot_test.go` 264 行、`internal/cli/doctor_legacy_agmsg_home_test.go` 122 行など)と、`RALPH_ORG_AGMSG_HOME` を `t.Setenv` していたテストを差し替え口に付け替える機械的な置き換え。残りは skill と recipe の写し、README、`templates/base/` の設定
- 後工程は cycle 1 の中で回した(`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の上限は上げていない)。self-review は 3 回(初回 LOW 4、S6 のあと LOW 6、V3 の直しのあと LOW 4)、verify は 3 回(LOW 2、4、3)、test は 2 回(初回は既知の不安定なテスト 1 本で rc 1、打ち直して rc 0。S6 と V3 のあとの再実行は rc 0)、sync-docs、cross-review(codex、所見 0 件)。記録だけの修正の例外は使っていない

```
何を変えるか → git の helper(S1) → 個人用のファイルの読み込み(S2) → 無視と隔離(S3) → テストの差し替え口(S4a) → env の撤去・拒否・warn(S4b) → 文書(S5) → 不安定なテスト(S6、V3) → レビューと main の取り込みで足した直し → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで:
  - 設定は `ralph.toml`(commit され、全員に効く)と環境変数に分かれていた。agmsg の場所は `RALPH_ORG_AGMSG_HOME` > `[org].agmsg_home` > 既定の順で決まり、機械ごとの値を書く git が無視するファイルはなかった。
  - テストは `RALPH_ORG_AGMSG_HOME` を `t.Setenv` して本物の agmsg から離していた。`ralph org gate` のスクリプトは、`RALPH_VERIFY_*` と `HARNESS_VERIFY_*` を除いた呼んだ側の環境をそのまま受け継いでいた。
- この PR から:
  - `ralph.local.toml` を足す。`ralph.toml` と同じ形で、キーごとに上書きする(表はキーごと、配列は置き換え)。`ralph.toml` がなくても読む。git の中では、どの worktree やサブディレクトリから打っても主 worktree の root のものを読む。`.gitignore` の管理ブロックが無視する。
  - ralph は `RALPH_ORG_AGMSG_HOME` を読まない。agmsg の場所は `[org].agmsg_home`(`ralph.local.toml` で上書き)だけで決まる。環境に変数が残っていれば `ralph doctor` が warn を出す。
  - `--config` の相対の `[org].agmsg_home` は、`ralph org watch --all --detach` と `ralph org start` が見張りを起動する前にも断る(`ralph org director start` は前から断る)。個人用のファイルの相対の値は `config.Load` が断る。
  - `ralph org gate` と `ralph org regate` のスクリプトは `RALPH_LOCAL_CONFIG=off` で走る。`internal/cli`・`internal/org`・`internal/config` のテストも同じ値に固定し、`internal/cli` と `internal/org` は agmsg の場所も存在しないパスに固定する。
- issue の文との違い(issue は #276 の本文。計画の Design decisions の D1〜D10 から書く):
  - issue の表は変数を一度に移す。この PR はそのうち `RALPH_ORG_AGMSG_HOME` の 1 行と、個人用のファイルの仕組みだけ(D1)。`RALPH_ORG_STATE_DIR` は A2、標準フローの値とスクリプトは B。
  - issue は個人用のファイルの置き場所と読み方を決めていない(「名前は計画で決める」)。名前は `ralph.local.toml`。読む場所は config のパスの dir から決め、cwd から決めない(D2)。dir が git の中なら主 worktree の root にする。git が無視するファイルは linked worktree に現れず、`--config` なしでサブディレクトリから打つと `./ralph.toml` の dir がサブディレクトリになるため(Codex の計画助言の HIGH)。
  - issue の「残す内部の受け渡し」の一覧に `RALPH_LOCAL_CONFIG` はない。この PR が足した(D4)。ちょうど `off` のときだけ意味があり、gate とテストが開発機の個人用のファイルを読まないための受け渡しで、人の設定を上書きする値ではない。
  - issue にない決めが 3 つある。個人用のファイルの `agmsg_home` は絶対パスか `~/` 始まりだけ(D5)、watch の起動でも相対の `--config` の値を断る(D7)、環境に `RALPH_ORG_AGMSG_HOME` が残っていれば doctor が warn する(D8)。どれも、読むプロセスの cwd が主 worktree でないと値が別の agmsg を指しうること、黙って効かなくなると座席が別の agmsg に向くことが理由。
  - `.gitignore` の無視の規則は先頭の `/` なし(D10。Codex の計画助言の MEDIUM)。
- 計画の本文の読み替え(承認の digest `1609f175543b` を変えないため本文は直さず、Progress checklist に読み替えを書いた):
  - Rollout の「`.gitignore` の管理ブロックに `/ralph.local.toml`」は D10 の前の書き方。実装と README は先頭の `/` なしで、どの階層でも無視する。
  - Assumptions の「`internal/org` の TestMain は `statedir_test.go` にある」は外れ。origin/main の `internal/org` に TestMain はなく、`internal/org/main_test.go` を新しく作った(verify の I4)。
  - Scope S3 にある `internal/cli` の TestMain の `RALPH_LOCAL_CONFIG=off` は S2 に前倒しした(08d1d949)。
  - S5 の verify のコマンドは、README を grep の対象から外して読む形に改めた(README は「もう読まない」と書くので 1 件残る)。`go test ./internal/cli/ -run 'Gate|Status'` も足した。
  - tech-debt の `resolvedOrClean` の呼び出しは「7 か所」ではなく、本番 4 呼び出し(3 ファイル)とテスト 5 呼び出しだった。
  - S6(不安定なテストの直し)は本文の Scope と Implementation outline にない。Progress checklist に判断(consult の案 C)と理由を書いて足した。
- release のノート: sync-docs の report の「Release notes」の節。要点は、`RALPH_ORG_AGMSG_HOME` を読まなくなるので、値を `ralph.local.toml` の `[org].agmsg_home` に移し、更新のあとに `ralph doctor` を打つこと。
- 戻し方: この PR を revert する。個人用のファイルがなければ、動きは `ralph.toml` だけのときと同じ。revert すると `ralph.local.toml` は読まれなくなり、`RALPH_ORG_AGMSG_HOME` がまた効く。

## 2. git の helper(S1、ab012b7d)

- `internal/gitroot/gitroot.go` を新設した(130 行)。`internal/org/statedir.go` にあった `gitMainWorktree`・`parseMainWorktreeRecord`・`gitToplevel`・`samePath` を移し、作業ディレクトリを引数で取る形にした: `MainWorktree`(`:35`)、`parseMainWorktreeRecord`(`:71`)、`Toplevel`(`:94`)、`SamePath`(`:105`)、`ResolvedOrClean`(`:112`)、`command`(`:125`)。本体は `git -C dir` を足した以外は同じ。
- `command` は dir が空のとき `-C` を付けない(`:119-124` の注釈)。`statedir_test.go` の git stub が引数列の全体で照合することと、`TestStatusCmd_MergeGuaranteeRunsNoGH` が `rev-parse` の呼び出しを数えることを壊さないため。
- `internal/org/statedir.go` は `gitroot.MainWorktree("")`(`:67`、`:114`、`:133`)、`gitroot.Toplevel("")`(`:70`、`:136`、`:163`)、`gitroot.SamePath`(`:115`、`:175`)を呼ぶ。`feature.go:236` と `:576` は `samePath` を `gitroot.SamePath` に替えただけ。org の ledger の決まり方(AC11)は変えていない。
- `resolvedOrClean`(`statedir.go:193`)は 1 行の転送関数として残した。呼び出しは `feature.go:197`、`permissions.go:209`、`split.go:183` と `feature_test.go` の 5 か所。名前の付け替えだけのために 3 つの本番ファイルを触らないためで、tech-debt に 1 行ある(self-review の L1)。
- テスト: `TestParseMainWorktreeRecord` の 7 ケースを `gitroot_test.go:218` に移した。新しいのは `TestMainWorktree_RunsGitInDir`(`:86`)、`TestMainWorktree_NoneToName`(`:114`)、`TestToplevel_RunsGitInDir`(`:139`)、`TestEmptyDirIsTheWorkingDirectory`(`:167`)、`TestSamePath`(`:190`)。bare の linked worktree と git の外も見る。
- 検証: gitroot 4 秒、org 655 秒、rc 0。

## 3. 個人用のファイルの読み込み(S2、9ed3793d と 08d1d949)

- `internal/config/local.go`(新規、106 行):
  - `LocalConfigFileName`(`:19`)と `LocalConfigEnv`(`:25`)。
  - `LocalPath`(`:42`): config のパスの dir を絶対にしてから(`git -C` に相対の dir を渡すと cwd 基準になるため)、`gitroot.MainWorktree(dir)` の root、なければ `gitroot.Toplevel(dir)`(主 worktree を名指せない bare の linked worktree)、それも失敗すれば dir の `ralph.local.toml`。
  - `readLocalConfig`(`:67`): `RALPH_LOCAL_CONFIG` がちょうど `off` なら git も呼ばずに nil。`fs.ErrNotExist` だけを「ファイルなし」とし、ディレクトリや権限のエラーはパスつきで返す。
  - `checkLocalAgmsgHome`(`:91`): 個人用の文書だけを見て、相対の `[org].agmsg_home` を断る。絶対、`~/` 始まり、空は通す。`ralph.toml` の相対の値は今までどおり通る。
- `config.Load`(`internal/config/config.go:695`)は `ralph.toml` を読んだあと、同じ `Config` にもう一度 `toml.Unmarshal` する(`:726`)。go-toml v2.3.0 は表をキーごとに重ね、配列と `[[org.model_pool]]` を置き換える(テストで固定)。`[org.roles]` もキーごとに重なる。
- `[org].driver_pool` と `[org].model_pool` の有無(`orgPoolKeysPresent`、`config.go:660`)は 2 つの文書のどちらかにあれば「ある」とする。検査は重ねたあとに `validateLoaded`(`:754`)で 1 回。個人用のファイルのエラーは、parse なら `<パス>: …`、検査なら `(with <パス> applied)` でパスを名指す。
- `ralph config get` は重ねた値を返す。help に 1 段落(`internal/cli/config.go:57-62`)。読めない・正しくない個人用のファイルは exit 3。
- テスト(`internal/config/local_test.go`): `TestLocalPath_*`(`:114`〜)、`TestLoad_LocalFileOverridesKeyByKey`(`:198`)、`TestLoad_LocalFileReplacesArrays`(`:235`)、`TestLoad_LocalFileWithoutRalphToml`(`:264`)、`TestLoad_LocalFileThroughASymlink`(`:294`)、`TestLoad_LocalFileErrorsNameItsPath`(`:315`)、`TestLoad_LocalConfigEnv`(`:388`)、`TestLoad_LocalAgmsgHome`(`:452`)、`TestLoad_LocalPoolKeys`(`:501`)。AC2 の回帰は `internal/cli/config_test.go:245`(主 worktree の root、linked worktree の root、サブディレクトリから打った `config get org.agmsg_home` が同じ値)。
- 計画とのずれ(08d1d949): Load が個人用のファイルを探す `git rev-parse` を打つようになり、git の呼び出しを数える cli のテスト 2 本(`TestStatusCmd_MergeGuaranteeRunsNoGH`、`TestOrgMergeOrder_NoLedgerPrintsTheNoBranchesNoteWithoutRunningGit`)が落ちた。cli の TestMain に `RALPH_LOCAL_CONFIG=off` を入れ(`internal/cli/main_test.go:127`)、S3 から S2 に前倒しした。
- 検証: rc 0。変異 4 つをテストが落とした。

## 4. 無視と隔離(S3、08926b11)

- `.gitignore:25` と `templates/base/.gitignore:25` の管理ブロックに `ralph.local.toml`(先頭の `/` なし)。2 つの管理ブロックは同じ中身で、`ralph upgrade` で配布先にも入る。
- gate の環境: `internal/org/gate_run.go` の `gateLocalConfigEnv`(`:53`)と `gateScriptEnv`(`:71`)。受け継いだ `RALPH_LOCAL_CONFIG` を名前の完全一致で落とし(`:75`)、`RALPH_VERIFY_SCOPE=full` のあとに `RALPH_LOCAL_CONFIG=off` を足す(`:83`)。`Gate`(`gate.go:177`)と `Regate`(`regate.go:659`)は同じ関数を通る。`GateParams.Env` の doc(`gate.go:66-72`)と help(`internal/cli/org_gate.go:54-58`)に 1 文ずつ。help は `TestOrgGateHelp`(`internal/cli/org_gate_test.go:789`)が固定しているので 1 行足した。
- TestMain: `internal/org/main_test.go:29`、`internal/config/main_test.go:18`(新規)。`internal/cli` は S2 で済んでいる。
- テスト: `TestGate_ScriptEnvironmentIsFixed`(`gate_test.go:1305`)は `Gate` を本物のスクリプトで走らせ、呼んだ側の `RALPH_LOCAL_CONFIG=on` がちょうど 1 つの `=off` として届くことを見る。`TestGateScriptEnv`(`:1342`)は継承の 4 形(`=on`、`=`、`=off`、`=` なし)と、名前が前方一致・部分一致するだけの変数を見る。
- built binary を走らせるシェルテストには、今のところ設定を読むものがない。B でスクリプトが `ralph config get` を呼ぶようになったら、B で `RALPH_LOCAL_CONFIG=off` を渡す(計画の Scope に書いた)。

## 5. テストの差し替え口(S4a、35e612f9)

- `driver.AgmsgHomeOverride`(`internal/org/driver/agmsg.go:148`)を足した。`ResolveAgmsgHome`(`:157`)が最初に見る。production のコードは設定しない。`driver` に置くのは、`internal/org/prompts.go` が `ResolveAgmsgHome("")` を自分で呼ぶため(D6)。
- `internal/cli/main_test.go:135` と `internal/org/main_test.go:33` の TestMain が、存在しないパス(`os.TempDir()` の下)に固定する。`setAgmsgHomeOverride`(`internal/cli/main_test.go:177`、`internal/org/main_test.go:41`)が `t.Cleanup` で戻す。`setupOrgStubPATH`(`internal/cli/org_test.go:241`)は `:262` でスタブに向ける。
- `RALPH_ORG_AGMSG_HOME` を `t.Setenv` して隔離していたテストを差し替え口に移した(17 files、+122/-33)。env の効き方そのものを確かめるテストは、この時点では env を読むので差し替えを空にして残した。`func Test` の数は前と同じ 2173 で、増えたのは driver のサブテスト 2 つ。
- 進め方: 実装担当が利用上限で止まり、別の担当が残った変更を引き継いで再開した。
- 検証: handoff のコマンドは rc 0。`-count=1` の通しを 2 回回した(load 11〜15)。落ちたのは時間の条件で決まる既知のテストなど(agmsg の解決を通らない)で、単独で回し直すと通った。

## 6. env の撤去、D7 の拒否、doctor の warn(S4b、1c530966 と b1550af0)

- driver: `ResolveAgmsgHome` は env を読まない(`agmsg.go:157-166`、順は差し替え口 > `[org].agmsg_home` > 既定)。`AgmsgHomeEnv` は消えた。
- watch の起動(D7): `orgWatchChild`(`internal/cli/org_watch.go:405`)の先頭で、`--config` が明示され `agmsgHomeIsRelative`(`:441`)なら `relativeAgmsgHomeError`(`:450`)で断る。呼ぶのは `ralph org watch --all --detach`(`org_watch.go:680`)と、start 系の動詞が見張りを起こすときの `wireOrgStartWatchGate`(`org.go:489`。`ralph org start` が使う)で、どちらも `launchAndAwaitOrgWatch`(`org_watch.go:598`)の最初に子を組み立てる。`ensureOrgWatch` は見張りがすでに動いていれば子を組み立てないので、その場合は断らない。`orgWatchChildAgmsgHome`(env を絶対パスにして子に渡す関数)は消えた。help に 1 文ずつ(`org.go:957`、`org_watch.go:94`)。
- director start(D13): `checkStartAgmsgHome`(`internal/cli/org_director_headless.go:375`)から `cmdLaunch` の引数と env の警告を消し、拒否の文を watch と共通の `relativeAgmsgHomeError`(`:378`)にした。拒否の条件(`Changed("config") && configPath != "" && agmsgHomeIsRelative(...)`)と、`startDirector` の拒否の順(`enabled` が先)は変えていない。D13 の理由は「director と watch は ledger の主 worktree の root で動く」に書き直した。
- 起動スクリプト: `DirectorStartParams`(`internal/org/director_start.go:748`)と `directorLaunchScript`(`internal/org/director_cmd.go:259`)から `AgmsgHome` を消した。スクリプトが export するのは `RALPH_DIRECTOR_TOKEN`(`:317`)と `driver.SeatPaneEnv()`(`:318`)だけ。
- doctor(D8): `checkLegacyAgmsgHomeEnv`(`internal/cli/doctor.go:853`、呼び出しは `:142`)は、環境の `RALPH_ORG_AGMSG_HOME` が空でなければ warn を 1 行出す。値には使わず、fail にもしない。warn は「もう読まない。`ralph.local.toml` の `[org].agmsg_home` に絶対パスか `~/` 始まりで書く」と言う。
- shell: `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` から `[org]` の段と `RALPH_ORG_AGMSG_HOME` の既定の行(27 行ずつ)を消した。`defaults_sync_test.go` から shell 側の照合を消した(config と `templates/base/ralph.toml` の照合は残る、`:170`)。`tests/test-ralph-config.sh` も 5 行消えた。
- テスト: env の効き方を確かめていたものを新しい決まり方に書き直した(`driver/agmsg_test.go`、`director_cmd_test.go`、`org_director_headless_test.go`、`org_watch_test.go`、`org_runtime_env_test.go`、`prompts_leader_inbox_test.go`、`spawn_test.go` など)。新しいのは `TestCheckLegacyAgmsgHomeEnv`(`doctor_legacy_agmsg_home_test.go:16`)、`TestRunDoctorOpts_LegacyAgmsgHomeEnv`(`:40`)、`TestOrgWatchDetach_ARelativeAgmsgHomeInConfigIsRefusedBeforeTheLaunch`(`org_watch_test.go:1087`)、`TestOrgStart_ARelativeAgmsgHomeInConfigIsRefusedAndNothingLaunched`(`org_start_watch_test.go:691`)、`TestOrgDirectorStart_TheAgmsgHomeVariableGetsNoWarning`(`org_director_headless_test.go:723`)、`TestOrgDirectorStart_LaunchCmdPassesTheDirectorNoAgmsgHomeVariable`(`:2196`)。`func Test` は 2449 から 2452。消したサブテストは、消えた env の振る舞いだけを見ていたもの。
- b1550af0: `internal/org/main_test.go` の注釈 3 か所から env の記述を消しただけ。
- 検証: 1 回目は負荷(load 40〜65)で org と cli が 30 分の timeout に当たり rc 1、org の 6 本が時間の条件で落ちた。6 本は単独で全部通り、規則どおり 1 回だけ打ち直した 2 回目が rc 0。overlay の変異 4 つ(env を読み戻す、D7 の拒否を外す、doctor の行を外す、起動スクリプトに export を戻す)は、新しいテストがすべて落とした。production のコードで `RALPH_ORG_AGMSG_HOME` を読むのは doctor の warn だけで、ほかに注釈が 1 行ある(`agmsg.go:154`)。

## 7. 文書(S5、dd1d0c0a)

- README に `### ralph.local.toml` の節(`README.md:142`〜`:146`): 形、キーごとの上書き、配列の置き換え、主 worktree の root、linked worktree やサブディレクトリの写しは読まれず何も言われないこと、`.gitignore` と `ralph upgrade`、パスを名指すエラー、絶対か `~/` の `agmsg_home`、`RALPH_ORG_AGMSG_HOME` をもう読まないこと。`config get`・status・watch・director・gate・merge-order の段落も合わせた。
- `templates/base/ralph.toml:2` の見出しと `:89` の `agmsg_home` の注釈。
- `/org` skill(`.claude/skills/org/SKILL.md:58` の前提の節、command table の `watch`・`start`・`gate`・`director` の行)と `/director` skill(`.claude/skills/director/SKILL.md:254`、`:289-292`)。`.claude/` と `templates/base/.claude/` を直し、`.agents/` の 2 つは `./scripts/sync-skills.sh` で作り直した。4 つの写しは一致(`check-skill-sync` 14 本)。
- recipe 2 本(`docs/recipes/isolated-herdr-server.md`、`director-launch-cmd.md`)と `templates/base/docs/recipes/` の写し。
- `AGENTS.md:89`(`internal/config/` の行に個人用のファイル)と `:90`(`internal/gitroot/` の行)。
- コードの注釈と help: `internal/cli/org.go`、`org_gate.go`、`org_watch.go`、`internal/org/gate.go`、`seat.go`。help を固定する `internal/cli/org_start_watch_test.go` も合わせた。
- tech-debt: 270 行の `RALPH_ORG_AGMSG_HOME` の行を RESOLVED にし、273 行は (b) だけを取り消し、`samePath` の参照を `gitroot.SamePath` に直し、`resolvedOrClean` の行を足した。
- 検証: `sync-skills`、`check-skill-sync`、`check-sync` が通り、対象の grep は 0 件。`go test ./internal/cli/ -run 'Gate|Status|Help'` と `go test ./internal/org/ -run 'Gate'` が ok。

## 8. 不安定なテスト(S6 の 321203dc と 140ed1d2、V3 の c7095378)

どちらも test と verify が tech-debt の Trigger に当たったことを見つけ、持ち越しの理由(その PR は変えていない)が使えなかったので、この PR の中で直した。判断は consult(案 C)に聞いた。

- S6: `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`(`internal/org/director_cmd_test.go:299`)は、`sh -c "trap '' TERM; exec sleep 30"` を起こして固定の 100 ms 待ってから SIGTERM を送っていた。負荷が高いと trap の前に SIGTERM が届いて sh が終わる。test の初回の `run-test.sh` で 5 回目の失敗が出た。
  - 直し: `sh` が trap の直後に作る marker のファイルを `waitForFileWithin`(`:1110`)で待つ(`:307`)。marker ができた時点で TERM は無視されていて、無視は `exec` を越える。`ps` で `sleep` を待つ案は、`comm` の出し方が macOS と Linux で違うので採らなかった。
  - 140ed1d2: 待ちの上限を 5 s から 10 s にして、同じ package の 10 s の待ち(`cmdStartFixture.release`、`startFakeAgent` など)にそろえた(self-review の L5)。tech-debt の閉じた行の Trigger の欄も取り消した(L6)。
  - 検証: `-count=30` ok。待ちを外すと load 538〜562 で 50 回中 7 回落ち、今の版は 50 回中 50 回通った。33b567e7 の test の再実行では `-count=20` で 20/20。
- V3: `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog`(`internal/cli/org_watch_test.go:1336`)は、子のスクリプトが `stdin: eof` を書いてから `trap '' TERM` を置くので、テストがその行を見て SIGTERM を送ると trap の前に届きうる。S6 の行と同じ形の競合で、S4b がこのファイルを変えていた。
  - 直し: 子のスクリプトの 1 行目で trap を張る(`:1343`、`if [ "$2" = ignore-term ]; then trap '' TERM; fi`)。ログに出る行の順は変わらない。
  - 検証: `-count=20` ok。tech-debt の行の記録では、`sleep 0.3` を挟むと 3 回中 3 回失敗し、trap を先頭に動かすと 10 回中 10 回通る。33b567e7 の再実行では 20/20。
- 残っているもの(新しい tech-debt の行): 負荷に弱い固定の期限を持つ 2 本 `TestStartDirector_ACleanupThatCannotConfirmTheEndWritesNothingOnceItIsNotCurrent/an_attach_--replace`(20 s)と `TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time`(3 s)。S6 の検証で load 126〜428 のときに落ち、単独と load が下がってからは通った。この PR は 2 本のテストのコードを変えていない。

## 9. レビューと main の取り込みで足した直し

- plan advisory(codex): 指摘 2 件をどちらも反映した。(1) [HIGH] サブディレクトリから打つと個人用のファイルが読まれない → D2 を「dir が git の中なら主 worktree の root」に変え、AC2 と回帰テストを足した。(2) [MEDIUM] `/ralph.local.toml` はルートしか無視しない → 先頭の `/` をやめた(D10)。
- 08d1d949(S2 のずれ): 上の S2 の節。
- d5386fb7(main の取り込み、5eba4013 = #286): 衝突は 7 本で、`scripts/ralph-config.sh` の 2 つの写し、`/org` skill の 4 つの写し、README。ralph-config.sh の見出しは、こちらが消した `[org]` の段の言及を落とし、main の「機能ごとの org の leader の計画助言」を残した。`/org` skill は main の `plan_advisory` の文のあとにこちらの `ralph.local.toml` の文を続け、`.claude` と `templates/base/.claude` を直して `sync-skills.sh` で写しを作り直した。README はこちらの版を取り、main の 2 か所を入れ直して、main の差分と語の単位で一致することを確かめた。
- 27f43d67(self-review の初回): LOW 4 件、Merge: yes。L1 は sync-docs で tech-debt の理由を直した。L2(相対 `agmsg_home` の判定が `config` と `cli` に 2 か所)、L3(doctor が個人用のファイルのエラーを `Name: "ralph.toml"` で出す)、L4(`AgmsgHomeOverride` と env の名前が戻らないことを固定するテストがない)は直さず、PR の Known gaps にした。
- 64a20f69(verify の初回): pass。V1(tech-debt の D13 の変異の行の Trigger に当たった)と V4(Trigger に当たったのに注がない行)は sync-docs で注を足した。V2(`config get` のエラーの頭が `ralph.toml:` になる)は PR の Known gaps。
- 148d542e(test の初回): pass。隔離の確かめ(clone に壊れた `ralph.local.toml` と偽の agmsg を置いたカナリア。6 package で呼び出し 0 件)と、感度の確かめ 3 つ。Test gaps の 2 つ(regate のスクリプトに `RALPH_LOCAL_CONFIG=off` が届くことを見るテストがない、agmsg の隔離を外しても `TestOrgSend*` が通る)は tech-debt の新しい行にした。
- 321203dc、140ed1d2、c7095378: 上の「不安定なテスト」の節。S6 のあとは self-review(d7f09f81)と verify(77a415c9)を、V3 の直しのあとは self-review(fa769eea)と verify(33b567e7)を回し直し、test は両方を含めて 33b567e7 で打ち直した(report は c9e83864。どれも cycle 1 のまま)。
- 10423da9(sync-docs): tech-debt の 16 行(Trigger に当たった 12 行に持ち越しの注、L1 の理由の書き直し、RESOLVED の注 2 行の直し、`isolateGitEnv` の行に写し 2 つの注、177 行の (b) の取り消し)と新しい 2 行、multi-org spec の FR-6 と FR-9 の日付つきの箇条、org-runtime spec の FR-2 の注、README の 1 文(個人用のファイルの空の `agmsg_home`)、計画の AC の checkbox。
- 6460c420(cross-review の triage report): codex(gpt-6-astra、reasoning effort xhigh、read-only)が HEAD 10423da9 に対して `codex exec review --base main` を回し、所見は 0 件。前後の git の状態は同じ(`cmp`)。codex は sandbox が一時ファイルを許さないので Go のテストを打てていない。

## 10. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(verify の 3 回と、sync-docs の run。gofmt、go vet、golangci-lint 0 件、staticcheck、shellcheck、`check-sync`(IDENTICAL 172)、`check-skill-sync`(14 本)、tech-debt の plan 参照、branch の secret scan)
  - `GOFLAGS=-count=1 ./scripts/run-test.sh` rc 0(HEAD 33b567e7、shell 45 本、Go 10 package。初回の 64a20f69 は既知の不安定なテスト 1 本で rc 1、打ち直して rc 0)。`func Test` は origin/main の 2,442 から 2,467
  - カバレッジ(64a20f69 の test): `internal/config` 96.6%、`internal/gitroot` 92.9%、`internal/org/driver` 95.3%
  - HEAD 27f43d67 から build した binary の probe(verify の初回): 個人用のファイルがキーごとに上書きすること、`ralph.toml` なしで読むこと、主 worktree の root・サブディレクトリ・linked worktree の root・その下のサブディレクトリの 4 か所から打った `config get org.agmsg_home` が同じ値になること、壊れた個人用のファイルが exit 3 でパスを名指すこと、`RALPH_LOCAL_CONFIG` が未設定・`off`・`OFF`・空のときの読み方、`RALPH_ORG_AGMSG_HOME` を設定しても `config get` と doctor の agmsg の check が変わらず、doctor の差が warn の 1 行だけであること
  - `git check-ignore` が `ralph.local.toml` と `config/ralph.local.toml` の両方で通ること、2 つの `.gitignore` の管理ブロックが同じであること
  - テストが本物の agmsg と開発機の個人用のファイルに触れないこと(test の隔離の確かめ。カナリア 0 件)。固定を外すと落ちる、または届くことも感度の確かめで見た: config の TestMain の `off` を外すと 2 本、cli では 11 本が落ち、agmsg の固定を外すと既定の agmsg に 35 回届いた
  - 変異: S2 の 4 つと S4b の 4 つはどれもテストが落とした
- 確かめていないこと・残っていること:
  - 実際の `ralph org gate` と `ralph org regate`。gate の環境は `TestGate_ScriptEnvironmentIsFixed` が `Gate` の本物のスクリプトで見ているが、`Regate` のスクリプトに `RALPH_LOCAL_CONFIG=off` が届くことを見るテストはない(tech-debt の新しい行)
  - detach した watch と `ralph org director start` の実際の起動(テストでは `orgWatchLaunch` を差し替えている)、本物の herdr と agmsg での通し
  - `ralph upgrade` が管理ブロックに `ralph.local.toml` の行を足すこと。管理ブロックが upgrade の面であること(`internal/cli/init.go:498`)を読んだだけで、打っていない。おそらく足される。未確認です
  - `./scripts/run-verify.sh` の既定のモードを 1 回で通すこと(static の半分と test の半分は、それぞれ rc 0)
  - git の `safe.directory` で断られる環境と、`GIT_DIR` を継承する git hook の中から `ralph` を打つ場合(self-review の N2。コードを読んだだけ)
  - Known gaps(コードとテストの LOW で、この PR では直していない): self-review の L2・L3・L4、verify の V2
  - 仕様として残るもの: `RALPH_LOCAL_CONFIG` は `off` だけに意味があり、ほかの値(パスを含む)は何も言わずに個人用のファイルを読む(self-review の N1)。linked worktree やサブディレクトリに置いた `ralph.local.toml` は読まれず、知らせる所はない(README に書いてある)
  - cross-review の codex は Go のテストを打っていない。Go のテストは `/test` の結果による
