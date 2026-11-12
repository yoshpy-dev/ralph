# Self-review report: gate-tool-versions

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-gate-tool-versions.md`(issue #311)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `d65f301e`、base は `origin/main` の `14531858`。merge コミット `4af8276b` で入った main 側の変更は含まない)。50 ファイル、6406 行追加、245 行削除(計画ファイルを含む)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

読んだもの。

- 新しいパッケージの本体の全部: `internal/toolchain/probe.go`、`run.go`、`projects.go`、`version.go`、`npmrange.go`、`doc.go`
- `internal/org/gate_tools.go` の全部と、差分の前後: `gate_run.go`(`runGateScripts` の全部)、`gate_record.go`(`withTools`、`toolDetails`、`gateRecordFromDetails`)、`gate.go`(`gateDirtyLines`、`gateGit`、`gateInterrupted`、ロックの前の汚れた worktree の断り)、`regate.go`(`regateGate`、`regateInterrupted`)、`regate_merge.go`
- `internal/cli/doctor_node.go` の全部、`doctor.go` の 1 行、`org_gate.go` と `org_regate.go` の help と `printGateTools`、2 つの `main_test.go` の固定
- テストのうち、実際の probe を回すもの(`gate_tools_test.go` 300〜500 行、`doctor_node_test.go` 545〜680 行)と、時間を測るテストの余裕(`run_test.go`、`probe_test.go` の上限はどれも予算 + 猶予 + 5 秒か 10 秒以上)
- reviewer の役割 prompt、`/org` skill と director skill の追加行(4 つの写しは `cmp` で同じ)、手順書の追加節(2 つの写しは `cmp` で同じ)、tech-debt の追加 4 行
- `git diff --check origin/main...HEAD` は rc 0

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- スクリプトに入る場所の名前: `safePathName` が制御文字(C0、DEL、C1)と不正な UTF-8 の名前を落とし(`SkippedPaths` に数える)、残りは `shellQuote`(`'` を `'\''` に)で 1 語にし、`cd` には `./` を前に付ける。`-` で始まる名前、`$`、`` ` ``、空白、`'` を含む名前でも、コマンドにも option にもならない。ツールの名前と引数は定数だけ。`git ls-files -z` なので quotePath の影響もない
- 記録の形: 1 行は `ralph-toolchain` と 4 つのタブ区切りの欄で、`ParseProbe` は欄が 5 つの行だけを記録として読む。場所の名前にタブと改行は入らない(上の濾し)。ツールの出力は begin と exit の記録の間に流し、どの記録も先頭に `\n` を付けるので、改行のない出力のあとでも記録が行の頭から始まる。出力は `$(...)` に入れないので、stdout を握る shim の子がいてもコマンド置換で止まらない
- 実行: `Setpgid`、時間切れと ctx の終わりで group に SIGTERM、`KillGrace` のあと SIGKILL、`WaitDelay = KillGrace`、shell が自分で終わったあとの残りも `stopStrays` で止める。`cappedBuffer` は上限のあとも `len(p)` を返すので書き手を止めない。stdin は空。gate.lock の fd は probe に渡らない(`ExtraFiles` なし、Go の fd は CLOEXEC)。時間切れの道の最悪は 15 + 2 秒、shell が先に終わり子が stdout を握る道は 15 + 2 + 2 秒で、AC4 の「15 秒と猶予」に収まる
- ctx: `Probe` は ctx が終わったときだけ error を返し、`measureGateTools` は FindLayout、前の status、probe、後の status のそれぞれのあとで `ctx.Err()` を見て `# tools: interrupted` を書いて返す。`runGateScripts` は probe の前に `defer load.sampler.stop()` を登録済みなので、probe の失敗の道でも負荷の goroutine が止まる。中断では何も記録しない(`TestGate_InterruptDuringTheProbeRecordsNothing`)
- 環境: gate は scripts の環境(`gateScriptEnv(env)`)を probe に渡し、`Probe` が `ProbeEnv` で 7 つの変数を最後に置く。scripts に渡す `env` は変えない(`ProbeEnv` は新しい slice を作る)
- 記録の値: `toolDetails` は鍵を `gateToolNames` の定数から作り、値を `gateToolValuePattern`(`^[A-Za-z0-9._+-]+$`)で確かめ直して合わなければ `unknown` にし、`tool_dirs` は `max(n, 0)` の整数。空白と `=` を含む値は出ないので `gateRecordFromDetails` の `ok=false` にならず、` log=` の切り分けも崩れない。古い binary は `tool_*` を知らない鍵として読み飛ばす。値は `gateToolValue` と `toolDetails` の 2 か所で確かめる(record の不変条件を org の側でも守る)
- worktree の比べ: gate は scripts の前に汚れた worktree を断るので、probe の前の `before` はふつう空で、`slices.Contains` の 2 重の走査が大きくならない。`gateGit` は `--no-optional-locks`
- doctor の Check が失敗しないこと: 返す status は warn、info、pass だけで、`--strict` も上げない(`scaffoldViolationStatus` は FR-9 の Check だけ)。`$SHELL` が sh の仲間でなければ `/bin/sh` にし、見つからない shell は `Run` の "could not start" から warn になる。読む file は `os.Stat` で regular file に限り、FIFO と device を開かない。値の表示は `nodeShow`(`%q`)か `nodePlain`(印字できる ASCII だけ)、`packageManager` は `ParsePackageManager` が名前の文字と semver を確かめたあとの値だけを出す。`m.Path` と `m.Reason` は `cleanText` 済み
- 測る場所が 0 個の repo では shell を打たない(`NodeTargets` が空なら info で返る)

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | exception-handling | doctor の probe は Ctrl-C で止まらない。`checkNodeToolchain` は `context.Background()` で probe を打ち、`toolchain.Run` は shell を別の process group(`Setpgid`)で起こす。doctor には signal の扱いがないので、端末の Ctrl-C は前面の group の ralph だけを終わらせ、`$SHELL -lc` とその子は 20 秒の時間切れもないまま走り続ける。gate と regate は SIGINT・SIGTERM・SIGHUP を ctx につないでいるので、Run の Cancel が group を止める | `internal/cli/doctor_node.go:114`(`ctx := context.Background()`)、`internal/toolchain/run.go:74`(`Setpgid: true`)。`internal/cli/doctor.go` に `signal.NotifyContext` なし(`internal/cli/org_regate.go:245` と `org_cleanup.go:165` にはある) | `checkNodeToolchain` で `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)` を作って probe に渡す。止まった場合は今の「was stopped before it finished measuring」の warn がそのまま使える |
| LOW | null-safety | doctor の root の測定は、login shell の startup files が残した cwd で走る。`BuildScript` は root の subshell では `cd` せず、ほかの場所には `./<dir>` の相対の `cd` をする。`$SHELL -lc` の `.zprofile` や `.bash_profile` が `cd` すると、root の node と pnpm は別のディレクトリ(mise なら global の版)で測られ、ほかの場所は `nocd` になり「cannot cd into the directory」の warn が出る。gate の `/bin/sh -c` は startup files を読まないので当たらない | `internal/toolchain/probe.go:135-140`(`if dir != RootDir` のときだけ `cd`、引数は `"./"+dir`)、`internal/cli/doctor_node.go:131`(`Argv: []string{shell, "-lc"}`) | `BuildScript` に root の絶対パスを渡し、各 subshell の頭で `cd <shellQuote(abs)>`(場所は `cd <shellQuote(abs + "/" + dir)>`)にする。`ProbeOptions.Dir` が既にその値を持つ |
| LOW | exception-handling | `doctorNodeLayout` は `FindLayout` の error を捨てる。git の repo で `git ls-files` が失敗した、あるいは `LsFilesTimeout`(10 秒)を超えた場合も、top-level の file だけを読む道に入り、detail は「not a git repository, or git ls-files failed」としか書かない。サブディレクトリの lockfile や版の file を読まなかった理由を、人が見分けられない | `internal/cli/doctor_node.go:156`(`if l, findErr := toolchain.FindLayout(ctx, root); findErr == nil`)、`:143` の note | `findErr` を返し、note に `cleanText` 相当で 1 行にした理由を足す。gate の側は既に `layoutErr` を log に書いている(`internal/org/gate_tools.go:143-146`) |
| LOW | readability | probe の環境の作り方が 2 つの呼び手で違う。`ProbeOptions.Env` は「呼び手の環境、スクリプトには `ProbeEnv(Env)` が渡る」と書かれ、`Probe` が `ProbeEnv` を当てる。gate はそのとおり生の環境を渡すが、doctor は `toolchain.ProbeEnv(os.Environ())` を渡すので 2 回当たる(冪等なので結果は同じ)。AC10 の doctor のテストは偽の probe が受けた `opts.Env` を調べるので、確かめているのは doctor の余分な 1 回のほうで、`Probe` が当てることは `TestDoctorNodeToolchain_RealShellProbe` だけが見ている | `internal/cli/doctor_node.go:133`、`internal/toolchain/probe.go:351-352,371`、`internal/org/gate_tools.go:290`、`internal/cli/doctor_node_test.go:554-581` | doctor も `os.Environ()` を渡し、偽の probe のテストは `toolchain.ProbeEnv(opts.Env)` を調べる形にする。残すなら `ProbeOptions.Env` の doc に「当て済みでもよい」と書く |
| LOW | maintainability | help の「It has 15 seconds in all」は probe の sh の予算で、gate.lock を持つ間に足される時間の全部ではない。probe の前に `git ls-files`(`LsFilesTimeout` 10 秒、別の予算)と `git status` が 1 回、後に `git status` が 1 回走り、probe 自体も止めるのに `KillGrace` を最大 2 回(計 4 秒)足す。最悪は 10 + 19 秒と `git status` 2 回。AC4 が縛るのは sh なので計画の上限は超えないが、help を読んだ人は 15 秒で済むと受け取る | `internal/cli/org_gate.go:115`、`internal/cli/org_regate.go:120`、`internal/toolchain/projects.go:45`、`internal/org/gate_tools.go:277-303` | `FindLayout` と probe を同じ deadline(`gateToolsTimeout` の ctx)の下に置き、probe には残りを渡す。あるいは help を「the probe has 15 seconds (listing the tracked files has 10 more)」のように直す |
| LOW | readability | help と手順書の言い切りが、すぐ次の文と食い違う。regate の help は「installs and writes nothing, has 15 seconds in all, and never changes the verdict」の直後に「When it changed the worktree, stderr gets ...」と書く。gate の help の「the verdict never depends on it」も、probe が worktree に書けば pass が `tree_changed` になる次の段落と合わない。手順書は「so measuring installs and writes nothing」のあとに「A tool those variables do not reach can still write」と書く。計画の Assumptions も、mise と corepack の変数の効き目は実機で確かめていないとしている | `internal/cli/org_regate.go:120-121`、`internal/cli/org_gate.go:113-115`、`docs/recipes/isolated-herdr-server.md:348-349`(templates の写しも同じ)、計画の Assumptions の 5 つ目(39 行) | 「is set up not to install, pin, or download」と「whether the probe finished never changes the verdict」の形にし、書き込みの検出は次の文に任せる |
| LOW | readability | probe の中断で出る error が、走っていない scripts を止めたと言う。`gateInterrupted` と `regateInterrupted` は「logPath があれば scripts が始まっていた」を前提に「the scripts were stopped and nothing was recorded」と書く。この差分で log を作ってから scripts の前に probe が走るようになり、その前提が崩れた。`gateInterrupted` の doc の「names the log when the scripts had started (logPath set)」も同じ | `internal/org/gate.go:275-287`、`internal/org/regate.go:668-677`、`internal/org/gate_tools_test.go:492`(この文面を固定している) | 文面を「the scripts were stopped, or never started, and nothing was recorded」か「the run was stopped」にし、doc の前提も直す。leader や reviewer がこの文で分岐していないことは prompt で確かめてから |

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 7。

## Positive notes

- 場所の名前を shell の語と記録の欄に入れる前に、制御文字と不正な UTF-8 を落とし、落とした数を log に出す。名前の濾し、引用、`./` の前置きの 3 つが揃っていて、git が追跡するどんな名前でもスクリプトの構文を変えられない
- ツールの出力をコマンド置換に入れず、begin と exit の記録の間に流す設計で、shim の子が stdout を握る場面(issue の場面)でも `$(...)` で止まらない。その場面を `run_test.go` と `probe_test.go` が実際の process で確かめている
- 記録の書き手が値を自分で確かめ直すので、`toolchain` の版のトークンの規則が将来ゆるんでも、gate の記録は読めなくならない
- 実際の probe を通す gate のテスト(`TestGate_RecordsTheToolVersionsWithTheRealProbe`)が、probe の環境に 7 つの変数が入ること、scripts の環境に入らないこと、worktree に何も書かれないことを同じ実行で見ている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

この review で新しく足す行はない。7 件の LOW はどれも小さく直せるので、直すか残すかは orchestrator が決める。残す場合は、2 回目の review で残った LOW を先送りとして tech-debt に書く。差分が足した 4 行(shim の子の版、doctor の shell、7 つの変数の外のツール、10 か所の上限)は、計画の D1、D6、D7、D2 と R2、R4、R7、R8 にそれぞれ対応し、差分と食い違わない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 7 件)
- Follow-ups:
  - doctor の probe を signal につなぐ(1 行目)と、login shell の cwd に頼らない `cd`(2 行目)は、doctor が人の shell を相手にするので、直す価値が高い
  - help と手順書の 3 か所の言い切り(5、6 行目)と、中断の文面(7 行目)は、同じ commit で直せる
