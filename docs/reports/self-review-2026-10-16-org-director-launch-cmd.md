# Self-review report: org-director-launch-cmd

- Date: 2026-10-16
- Plan: docs/plans/active/2026-10-16-org-director-launch-cmd.md(承認済み、digest af95b6db6074。8 段目の 8a-2b)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)
- Scope: diff の品質だけ。`git diff 3723d900...HEAD`(HEAD 3deaf148、11 コミット、40 ファイル、+7845/-488)。テストの実行、静的解析、仕様適合、文書のずれの検査は行っていない(/test・/verify・/sync-docs の担当)。計画の D1〜D12 は意図の確認にだけ使った

## Evidence reviewed

- 読んだ差分(コード、全体): `internal/org/director_cmd.go`(708 行すべて)、`director_start.go`・`director_stop.go`・`director.go`・`spawn.go`・`prompts.go` の diff すべて、`internal/cli/org_director_headless.go`・`org_director.go`・`org.go`・`status.go` の diff すべて、`internal/config/config.go` の diff すべて、`internal/org/prompts/director.md` の diff
- 読んだ差分(文書): `/director` skill の diff すべて(4 面は `md5` が同じ)、`/org` skill の 4 面(同じ)、`docs/recipes/director-launch-cmd.md` 全文(2 面は `cmp` が同じ)、`templates/base/ralph.toml`、README・AGENTS.md・repo-map・仕様 FR-9 の追記(`-U0`)、`docs/tech-debt/README.md` の追加 5 行、書き換え 3 行、削除 1 行(`|` で分けた列はすべて 7、「13 件のコメント・12 ファイル」の数は `grep` で一致)、証跡 `docs/evidence/director-launch-cmd-livefire-2026-10-16.md`(`$HOME` と秘密の grep は空)
- 読んだテスト: `internal/org/director_cmd_test.go` の helper 部(1〜120 行、470〜520 行、790〜1060 行、1330〜1360 行、1480〜1530 行)、`internal/cli/main_test.go` の diff、`org_director_headless_test.go` の helper(1840〜1945 行、2180〜2290 行)、`org_director_test.go`・`org_stop_all_test.go` の diff。実行はしていない
- 手で確かめたこと: macOS の `ps -o stat=,lstart= -p` が、生きている pid で `Ss <lstart>`(後ろに空白)、存在しない妥当な pid で終了コード 1 と出力なしを返すこと。日が 1 桁の lstart(`Thu Oct  1 ...`)を Fields と Join で 1 つの空白にしても、`time.Parse("Mon Jan _2 15:04:05 2006")` が通ること(scratchpad の小さな Go 片で確かめた)
- 追加で grep: 破壊的な呼び出し(`syscall.Kill`)の全箇所、`.Headless()`・`HerdrLaunch()`・`CmdLaunch()` の判別の全箇所、`RALPH_ORG_AGMSG_HOME` と `RALPH_DIRECTOR_TOKEN` の全箇所、`LaunchCmd =` を設定するテスト、`tmux`・`osascript`・`herdr` を打つテスト、追加行の pointer 崩れ・debug・秘密の語、追加コメント行の幅

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | null-safety | start が打たれた場所の `RALPH_ORG_AGMSG_HOME` を、相対のままでも絶対にせずに起動スクリプトへ書く。スクリプトは `cd <台帳の main worktree のルート>` のあとにその値を export するので、相対の値は start を打った場所とは別のディレクトリから解かれる。同じ start が起こす見張りは、同じ変数の相対の値を start の cwd で絶対にして受け取る。start を main worktree の下のディレクトリから `RALPH_ORG_AGMSG_HOME=./agmsg` で打つと、director と見張りが別の agmsg を使う。8a-2 の D13 が `--config` の相対の `agmsg_home` を断った理由と同じで、`--config` 経由では防いだこの分かれ目が、env 経由では開いている。気づかずに別の agmsg を使うので、失敗が見えない | `internal/cli/org_director_headless.go:312` が `os.Getenv(orgAgmsgHomeEnv)` をそのまま `AgmsgHome` に渡す。`internal/org/director_cmd.go:301` の検査は NUL だけで、`:312` が `cd` のあとに export する。見張りの側は `internal/cli/org_watch.go:369-385`(`orgWatchChildAgmsgHome`、`agmsgHomeIsRelative` なら `filepath.Abs`)。テストの値は絶対パスだけ(`internal/cli/org_director_headless_test.go:1972` の `/agmsg home/it's`、`internal/org/director_cmd_test.go:426`)で、相対の case がない | `startDirector` で `agmsgHomeIsRelative(v)` のとき `filepath.Abs` で絶対にして渡す(見張りと同じ規則)。または D13 と同じ断りにする。どちらでも、相対の値の case を CLI のテストに足す(絶対化の基準を main root にする変異が赤になるように、start を linked worktree か下のディレクトリで打つ) |
| L1 | LOW | readability | 整形の崩れが 4 か所ある。挿入した文のあとに段落を折り直していない形 | `internal/org/director_start.go:663` は 125 桁の 1 行(同じ段落の他の行は 78 桁前後)。`director_start.go:188-190` は `None of them writes / anything. An id not of the form d<N> and an / empty workspaceID` と、短い行で折れる。`docs/recipes/director-launch-cmd.md:242` は 133 桁(他の段落は 80 桁以内)。`.claude/skills/director/SKILL.md:266` は表示幅 90(中央値 69)で、4 面とも同じ | 4 か所を折り直す。skill は `scripts/sync-skills.sh` で 4 面を揃える |
| L2 | LOW | security(テストの隔離) | 後始末が、起動時刻を確かめずに、ファイルから読んだ pid へ SIGTERM を送る。AC10 は「テストが自分で起こしたプロセスにだけ signal」で、同じファイルの `killCheckedInProcesses` は ps の起動時刻が記録と同じときだけ送る。`sleep 5` が終わって pid が使い回される窓は小さく、実害は出にくいが、このブランチが製品側に入れた確かめと揃っていない | `internal/org/director_cmd_test.go:810-826`(`child.pid` を読んで `syscall.Kill(pid, SIGTERM)`)。比較: `:1013-1041`(`same()` で `StartTime` を比べてから SIGKILL) | pid を読んだ直後に ps で起動時刻を読んで控え、cleanup では同じ起動時刻のときだけ送る。または孫を起こさず、`startTestProcess` で起こした自分の子だけにする |
| L3 | LOW | maintainability | `MarkCmdLaunched` のコメントと switch の順が合わない。コメントは「token の比較が state より先なので、token を持たない呼び出し側は登録について何も知れない」と言うが、`!reg.CmdLaunch()` の断りが token の比較より前にあり、その文は `reg.describe()`(driver、attach 時刻、launch、state)を返す。launch cmd でない登録の driver と attach 時刻が、token なしで見える。台帳は同じ利用者が読めるので実害は小さく、コメントの言い過ぎ | `internal/org/director_start.go:349-357`(`:349` のコメント、`:354` の `case !reg.CmdLaunch()`、`:356` の token の case) | token の case を `!reg.CmdLaunch()` の前に移す(`reg == nil` の次)。または、コメントを「launch cmd の登録かどうかは token なしでも分かる」に直す |
| L4 | LOW | readability | 終わったと確かめられない stop の hint が、ps を読めなかった枝でも同じ文を出す。`once pid N has ended` と `look with ps -o pid,lstart,command -p N` は、プロセスが生きている枝の案内で、ps が読めない原因の枝では、ps で見るよう案内することになる。help はこの案内を両方の枝に約束している(`stop --help`)ので、挙動は意図どおりで、文面だけの話 | `internal/org/director_stop.go:507-508`(`again once pid %d has ended`)、`internal/cli/org_director_headless.go:497`(`may still be running: look with ps ...`) | 原因が `ps could not be read`(`directorProcessStopUnreadable`)のときは「ps を読めるようにしてから打ち直す」と言い分けるか、help と文面を「ps で見る」を書かない形に揃える。直さないなら今のまま(LOW) |
| L5 | LOW | maintainability | `DirectorStartResult.Launch` は `Registration.Launch` と同じ値を 2 つの場所に持つ。CLI の start の出力はこの 2 つ目の `res.Launch` で herdr と cmd を分け、stop の出力と status は `Registration.CmdLaunch()` で分ける。分ける元が 2 つあるので、将来の道が `Launch:` を書き忘れると、herdr の 2 行(空の pane と workspace)が出る | `internal/org/director_start.go:586`(フィールド)、`:714` と `internal/org/director_cmd.go:557`(定数で書く)、`internal/cli/org_director_headless.go:323`(`res.Launch == org.DirectorLaunchCmd`) | フィールドをやめて `reg.CmdLaunch()` で分ける(stop と同じ) |
| L6 | LOW | maintainability | `RALPH_ORG_AGMSG_HOME` の名前の定数が 3 つになった。`driver/agmsg.go:124` の文字列、`cli/org_watch.go:309` の `orgAgmsgHomeEnv`、新しい `org/director_cmd.go:242` の `directorAgmsgHomeEnv`。名前を読む側(`ResolveAgmsgHome`)が 1 つ、書く側が 2 つあり、変えるときは 3 か所を探す | `grep -rn RALPH_ORG_AGMSG_HOME internal --include='*.go'`(テストを除く 3 件) | `driver` に export した定数を 1 つ置いて 3 か所が使う。または、新しい定数の doc に `cli/org_watch.go:309` と `driver/agmsg.go:124` と同じ名前だと書く |
| L7 | LOW | readability | `abortCmd` の再読み込みが `for reread := false; ; reread = true` で、最大 2 周の再試行を「後置文で true にする無限ループ」で書いている。ループの先頭の `switch` が見る `err` は、前の周の `continue` 直前の代入のもので、`continue` から読み直すと追いにくい。ロジックは合っている(2 周目で `ErrDirectorCheckedInDuringStop` が出れば cleanup error に落ちる) | `internal/org/director_cmd.go:621-641` | `for attempt := 0; attempt < 2; attempt++` にして、2 周目の `ErrDirectorCheckedInDuringStop` を `attempt == 1` で明示する。または、1 回の読みと記録を `markCmdStoppedOnce(reg)` に切り出して 2 回呼ぶ |
| L8 | LOW | maintainability | `DriverCallTimeout` と `defaultDriverCallTimeout` の doc は、この deadline が掛かる呼び出しを列挙しているが、このブランチの 2 つ(launch_cmd の実行、`ReadDirectorProcess` の ps 読み)が入っていない(8a-2 の director の herdr の呼び出しも元から入っていない)。テストが ps を守るために `DriverCallTimeout = 30s` と書く場所がある(`director_cmd_test.go:1483`、`:1524`)ので、結合は実在する | `internal/org/spawn.go:213-222`、`internal/org/verbs.go:737-743`、`internal/org/director_cmd.go:116-127`(`readDirectorProcess`)、`:421`(`runDirectorLaunchCmd` の doc は「start gives DriverCallTimeout」) | 2 つの doc に director の launch_cmd と ps を足す。8a-2 の分も足すと、列挙が一度で閉じる |

## Positive notes

- シェルを通さない組み立てが一貫している。launch_cmd は argv で打ち、置き換えは 1 回の走査で、値の中の `{name}` を再び置き換えない(`ExpandDirectorLaunchArg`)。起動スクリプトは値をすべて `shellQuote` し、NUL・絶対パス・id・token の形を書く前に検査し、temp file に 0700 で書いて rename する。`launch/` も 0700 にする。`{script_sh}` は同じ `shellQuote` を通る。ここから注入できる道は見つからなかった
- signal の安全が 2 重になっている。SIGTERM は `stopDirectorProcess` の 1 か所にだけあり、`pid <= 0` を先に断り(0 と -1 は group と全体に飛ぶ)、ps の起動時刻が記録と同じ生きたプロセスにだけ送り、zombie・pid の再利用・読めないときは何も送らない。director の stop と start の後始末に SIGKILL はない(製品側の SIGKILL は `internal/org/gate_run.go:238` の gate の実行だけで、このブランチの外)。pid を単独で指定し、group には送らない。`ESRCH` は終了と数え、それ以外の失敗は「終わっていない」側に倒す
- 記録が事実より先に進まない形が、start の後始末(`abortCmd` → `detachCmdOnly`)と stop(`recordCmdDirectorStop`)の両方で同じ。終わったと確かめられなければ `stopped` を書かず、`--force` だけが書く。check-in が読みと記録のあいだに来たときは `MarkStoppedIfUnchanged` が pid と起動時刻を比べて断り、打ち直す stop を出す。ロックの下で外部呼び出しをしない
- check-in の書き手が `director.lock` の下で、id・launch・token・state・current・pid・起動時刻を検査し、fold も `launched` の pid と start_time の有無を launch ごとに検査する(`director.go:744-745`)。8a-2 の行は変わらず読める。hand edit した `director.jsonl` から出る値は `inboxField` か `PrintableInboxText` を通る
- テストの隔離が実際に効いている。CLI は `TestMain` で agent の探索と ralph の実行ファイルを失敗に固定し、org は `cmdStartFixture` だけが偽を設定する。`LaunchCmd` を設定するテストは全部その fixture 経由で、本物の `claude`・`codex`・`tmux`・`osascript`・`herdr` を打つ道は見つからなかった。signal を送るテストは、自分で起こしたプロセスか、ps の起動時刻が check-in の記録と同じプロセスだけを対象にする(L2 の 1 箇所を除く)
- 8a-2 の読み手を壊さない作りになっている。`Headless()` が両方の launch を含み、herdr 専用の処理は `HerdrLaunch()` で分ける。`recordHerdrID` と `MarkLaunched` が launch cmd の登録を断る(`ErrDirectorNotHerdrLaunch`)。fold が launch cmd の登録への `workspace_created` と `pane_created` を無視する
- 文書の 4 面と recipe の 2 面が byte 一致で、tech-debt の追加行が 7 列に揃い、数の主張が `grep` と一致する。証跡は `$HOME` を `~` に置き換え済みで、秘密の語はない。コメントの pointer は plan の ID(`plan 2026-10-16-org-director-launch-cmd, D7`)で書かれ、cycle や finding の番号を指す形がない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし。M1 は直す前提の指摘で、L1〜L8 は直すコストが小さい。このブランチが `docs/tech-debt/README.md` に足した 5 行(起動スクリプトの残り、秒精度の起動時刻、端末を閉じられないこと、Orca と WezTerm の例が未実行、8a-2 のコメントの archive パス)は、内容と列数を確かめ、重複もなかった | - | - | - | - |

_(行を足していないので、`docs/tech-debt/` への追記はない。)_

## Not checked

- Linux の `ps -o lstart` が、同じプロセスを 2 回読んで同じ秒を返すか。macOS では確かめた。procps が起動時刻を boot 時刻から組み立てる版で、時計の補正のあとに 1 秒ずれると、stop は「別のプロセスが pid を使っている」と読んで何も送らずに `stopped` を書く(生きた director が記録の上では止まったことになる)。計画の Assumptions が「Linux は CI で確かめる」としているので、/test の Linux の結果で見る
- テストの実行結果、`-race`、mutation、`go vet`、gofmt、golangci-lint、Go 1.25.8 の build

## Recommendation

- Merge: yes(CRITICAL・HIGH はなし。MEDIUM 1 件は、相対の `RALPH_ORG_AGMSG_HOME` を絶対にせずに起動スクリプトへ書く入力検査の穴で、既定の経路(絶対パスか未設定)は壊れない。LOW 8 件は整形、テストの cleanup、コメントと switch の順、hint の文面、重複した値、doc の列挙。signal の安全、シェルを通さない組み立て、記録と事実の順序に欠陥は見つからなかった)
- Follow-ups:
  1. M1: `startDirector` で相対の値を絶対にして `AgmsgHome` に渡し(見張りの `orgWatchChildAgmsgHome` と同じ規則)、CLI のテストに相対の値の case を足す。Go に触れるので、直すなら cycle 1 の fix としてパイプラインを回し直す
  2. L1(文書の 2 か所と skill の 4 面)は文書なので、直すなら /sync-docs で入れてよい。Go のコメント(`director_start.go:663`、`:188-190`)は L2〜L8 と同じ commit に入れる
  3. L2〜L8 は、直すなら M1 の fix と同じ commit にまとめる。直さずに進むなら、cap 2 の最後の回に当たるので、`/sync-docs` が L3・L6・L8(コメントの食い違いと重複)だけ登録簿に足す
  4. /test への引き継ぎ: Linux の `ps -o lstart` の安定(上の Not checked)と、相対の `RALPH_ORG_AGMSG_HOME` の case(M1 を直した場合)
