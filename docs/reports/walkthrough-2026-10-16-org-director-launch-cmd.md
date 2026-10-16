# Walkthrough: org-director-launch-cmd(8 段目の前半の 2 本目の後半、8a-2b)

- Date: 2026-10-16
- Plan: docs/plans/active/2026-10-16-org-director-launch-cmd.md(/pr で archive に移す)
- Branch: feat/org-director-launch-cmd(base 3723d900。この walkthrough を書いた時点で 25 commits、51 files、+8738/-516)
- PR: #232
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 43 files、+7722/-516 になる。そのうちテストが `internal/org/director_cmd_test.go`(+1733)、`internal/cli/org_director_headless_test.go`(+1245)、`internal/org/director_stop_test.go`(+530)、`internal/org/director_test.go`(+448)、`internal/config/config_test.go`(+261)ほか。ほかに `/director` と `/org` の skill の 4 面ずつ、README、AGENTS.md、repo-map、仕様 FR-9 の注、`templates/base/ralph.toml`、新しい recipe(2 面)、tech-debt がある
- 後工程は 1 回(cycle 1)で、cross-review の指摘は 0 件だった。self-review、verify、test で出た指摘は、どれも同じ cycle の中で直した(下の 7)

新しいファイルは `internal/org/director_cmd.go`(起動スクリプト、launch_cmd の実行、プロセスの確かめと SIGTERM)、`docs/recipes/director-launch-cmd.md`(と `templates/base/` の写し)、それぞれのテスト。次の順に読むと、依存の向きに沿って追える。

## 1. 設定(`internal/config/config.go`、4c3f5fb9)

- `[org.director].launch_cmd` は文字列の配列。既定は空で、空なら start は 8a-2 と同じく herdr で立てる(計画の D2)
- 置き換えは `{script}`、`{script_sh}`、`{name}`、`{cwd}` だけ(D1)。`ExpandDirectorLaunchArg` が要素ごとに 1 回だけ置き換えるので、置き換えた値の中の `{...}` はもう一度は読まれない
- `ValidateDirectorLaunchCmd` は、空の要素、知らない `{...}`、閉じていない `{`、`{script}` も `{script_sh}` もない値を断る。断るエラーはキーの名前と理由を言う。起動スクリプトが走らない値は、start が必ず時間切れになるため
- launch_cmd は main worktree の `ralph.toml` から読む(D10 に足した。task worktree から start を打っても、人が設定した端末を使う)

## 2. 登録の形と check-in(`internal/org/director.go`、`director_start.go` の store の関数、f8d4db99)

- `attached` の `launch` に `cmd` が増えた。`launched` は pid と `start_time` を持つ。`Headless()` は herdr でも cmd でも start が書いた登録を指し、herdr だけの処理は `HerdrLaunch()`、cmd だけの処理は `CmdLaunch()` で分ける(D11)
- 隠れた動詞 `ralph org director launched --id --token --pid --state-dir` は、起動スクリプトだけが打つ check-in(D4)。`MarkCmdLaunched` は `director.lock` の下で次をすべて確かめてから `launched` を書く
  - token が合う(token を知らない呼び手には、登録の中身を何も漏らさない順に確かめる。self-review の L3)
  - 今の登録で、`launch = "cmd"` で、`starting`
  - pid が正で、check-in の親の pid と同じ(スクリプトの `$$`)
  - その pid の起動時刻を読める
- どれかが違えば何も書かずに終了コード 1 で終わり、スクリプトは agent を exec しない。止めたと記録した起動に遅れて届いた check-in も、ここで断られる

## 3. start の launch_cmd の道(`internal/org/director_cmd.go`、`director_start.go`、2f7962c9)

- 何も書かずに断る場合は 8a-2 と同じものを当てる(D10)。`HERDR_PANE_ID` の断りだけは当てない(D9)。driver の agent が start の env の PATH になければ、何も書かずに断る
- 先に `attached` を書き、台帳の下に起動スクリプト `launch/director_<token>.sh`(0700、ディレクトリも 0700)を書く(D3)
  - 中身は `cd`、`RALPH_DIRECTOR_TOKEN` の export、`RALPH_ORG_AGMSG_HOME` の export(start の env にあるときだけ。相対なら start のディレクトリから絶対パスにする。self-review の M1)、check-in、agent の exec
  - 値はすべて単一引用符で囲む。check-in の ralph は start を打った ralph の絶対パスで、agent も `exec.LookPath` が見つけた絶対パス
- launch_cmd はシェルを通さずに打ち、`DriverCallTimeout`(既定 10 秒)で区切る。端末のデーモンが出力を持ったまま残っても、待ちの遅れのあとで戻る(D5)
- check-in は `--timeout-ms`(既定 60 秒)まで台帳を読み直して待つ。`launched` が書かれたら成功で、stdout に `started d<N> (... launch=cmd, name=director_<token>, pid=...)` と、話しかける端末の名前を出す
- 失敗したら後始末する(D6、`abortCmd`)
  - launch_cmd の終了コードが 0 でない、時間切れ、check-in がない、待つあいだの `attach --replace` のどれでも、stop と同じ手順(4)でプロセスを止めて `stopped` を書く
  - 後始末のあいだに check-in が届いたら、読み直して止める。待つあいだに人の stop が先に記録していたら、それを後始末の記録として扱う(6b9b27d7 のテストで固定)
  - プロセスの終わりを確かめられなければ、今の登録のあいだだけ `detached` を書いて登録を開いたまま残し、stop の打ち方を出す。今の登録でなくなっていれば何も足さない
  - 出力は、開いたかもしれない端末を人が閉じることを言う

## 4. stop の launch_cmd の道(`internal/org/director_stop.go`、`stopDirectorProcess`、4a9c44f8)

- signal を送るのは `stopDirectorProcess` の 1 か所だけ(D7)
  - 記録した pid を `LC_ALL=C TZ=UTC ps -o stat=,lstart= -p <pid>` で読む。生きていて起動時刻が `start_time` と同じときだけ SIGTERM を送り、プロセスグループには送らない
  - pid が消えた、zombie、起動時刻が違う(別のプロセスが pid を使っている)なら何も送らず、終わったとみなす
  - SIGTERM のあとは最大 10 秒、読み直して終わりを待つ。SIGKILL は送らない
- 終わらないときは何も書かずに終了コード 1 で断り、見るための `ps -o pid,lstart,command -p <pid>` と `--force` を出す。ps を読めないときも断る。SIGTERM の前に読めなければ何も送っておらず、SIGTERM のあとの待ちで読めなくなったときは SIGTERM は送ってある(`ErrDirectorProcessUnreadable`。verify の I1 で文面を分けた)
- `--force` は確かめたプロセスには普段どおり SIGTERM を送り、終わりを確かめられなくても `force` 付きで記録する
- env の `RALPH_DIRECTOR_TOKEN` が登録の token と同じなら、`--force` でも断る(D8)。director が自分に stop を打つと、記録の前に自分のプロセスが終わるため
- stop が読んだあとに check-in が pid を記録したら、何も書かずに断り、もう一度打つよう出す(`ErrDirectorCheckedInDuringStop`)
- 端末は閉じない。出力は、閉じるのは人だと言う

## 5. 表示(`internal/cli/org_director.go`、`status.go`、`org.go`)

- `ralph org director status`、`ralph status` の `director:` の行、`--json` は、launch `cmd`、状態、端末の名前、pid、model を出す。`terminal_name` は CLI の表示用の型にだけ置き、記録の型は変えていない
- `ralph org stop --all` の案内は、launch_cmd の閉じていない登録も数える。止めはしない

## 6. 雛形、skill、文書(a6dda336 ほか)

- `internal/org/prompts/director.md` に `{{IF_HERDR}}` と `{{IF_CMD}}` の行を足した。launch_cmd の director には、端末が人の設定で開かれ、止めるときは人に stop を頼むことを書く
- `/director` skill の 4 面に launch_cmd の立て方と止め方を、`/org` skill の 4 面に台帳を書く動詞 `director launched` を足した
- 新しい recipe `docs/recipes/director-launch-cmd.md`(2 面)は、tmux、Orca、WezTerm の例を 1 つずつ書く。tmux の例は実機で通した。Orca と WezTerm の例は `--help` から書いたもので、動かしていないと recipe に書いた
- README、AGENTS.md、repo-map、仕様 FR-9 の 8a-2b の注、`templates/base/ralph.toml` のコメント
- tech-debt に足した行
  - director が自分で打つ `ralph` は、agent の shell の PATH の先頭の版になる(実機で確かめた。既存の行に足した)
  - Linux で壁時計が step したあとの `ps -o lstart=`。生きた director を別のプロセスと読み、何も送らずに `stopped` を書くおそれがある(未確認)
  - `abortCmd` の 2 つ目の `ErrDirectorStopped` の枝にテストがない

## 7. cycle 1 の中の直し

- self-review(MEDIUM 1、LOW 8、98131af5)
  - M1: 相対の `RALPH_ORG_AGMSG_HOME` を、絶対パスにしてから起動スクリプトへ書く
  - LOW: 折り返し、テストの後始末の signal、check-in の確かめの順、ps を読めないときの案内、重複した値、環境変数の名前の定数を 1 つに(`driver.AgmsgHomeEnv`)、`abortCmd` の再試行の形、timeout の doc
  - 確認で出た LOW 2 件(コメントの折り返し)は 8ad4275d
- verify(LOW 1、情報 1、0f5fbf89 と e2667a6b): `orgLedgerMutating` の動詞の一覧、SIGTERM のあとで ps を読めなくなった場合の文面
- test(生き残った mutant 2 個、6b9b27d7): check-in を待つあいだの stop、`attach --replace` か `detach --force` のあとの、終わりを確かめられない後始末
- cross-review は 0 件。そのあと c19d4747 で evidence の launch token を伏せ字にした。GitGuardian が 3deaf148 の `--token '<値>'` の行を incident として報告したため。値は 1 回の起動を名指すだけで権限はない

## 8. 確かめたこと(evidence)

- 実機: `docs/evidence/director-launch-cmd-livefire-2026-10-16.md`。専用の socket の tmux(`tmux -L ralph-lf-8a2b`)と隔離した台帳で、claude 2.1.291 と codex-cli 0.160.1 の director の start、check-in、status、自分を止める断り、stop を通した。どちらも起動の確認画面の状態でも会話中でも SIGTERM で終わる。codex は npm の入口(node)が本体の子に SIGTERM を渡す。失敗の道は、端末を開かない launch_cmd と、終了コード 3 の launch_cmd を通した。実機は 88bb29fd の binary で、そのあとの直し(M1 ほか)はテストと verify の偽の端末での確認だけが見ている
- テスト: go1.26.0 と go1.25.8 の両方、`-race`、繰り返し、ubuntu:24.04 の container で通った(`docs/reports/test-2026-10-16-org-director-launch-cmd.md`)。テストは本物の herdr、osascript、端末のアプリ、人の tmux を打たず、signal はテストが起こしたプロセスにだけ送る
