# Self-review report: gate-tool-versions

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-gate-tool-versions.md`(issue #311)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(cycle 1)の 2 回目の review。1 回目(`245b85f9`、HEAD `d65f301e` を見た)は CRITICAL・HIGH・MEDIUM 0、LOW 7、Merge: yes で、修正の commit `b0b3bfe2` が LOW 1・2・3・5・6・7 を直し、LOW 4(`ProbeEnv` を 2 回当てる)は害がないとして残した
- Scope: `git diff origin/main...HEAD`(HEAD `3d89cfe7`、base は `origin/main` の `4220ea7c`。merge コミット `a734cfb6` で入った main 側の変更は含まない)。52 ファイル、6944 行追加、301 行削除(計画、1 回目の report、insight の行を含む)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

1 回目の指摘の行ごとに、直ったか、残したか、半分だけ直ったかを確かめた。

| 1 回目 | 結果 | 確かめたこと |
| --- | --- | --- |
| LOW 1(doctor の probe が Ctrl-C で止まらない) | 直った。ただし新しい LOW 1 を生んだ | `probeNodeToolchain`(`internal/cli/doctor_node.go:173`)が `doctorNodeSignalContext` の下で probe を打ち、probe が戻ると `stop()` で登録を外す。signal を受けるのは probe の間だけで、その前の `git ls-files` の間の Ctrl-C は今までどおり ralph を終わらせ、git は同じ process group なので一緒に止まる。テスト(`TestDoctorNodeToolchain_InterruptStopsTheProbe`)は seam と実際の SIGINT の 2 通りで、warn の 1 行と、偽の node が sleep のあとに書くはずのファイルが無いこと(group が止まったこと)を見る。SIGINT の subtest はテスト自身のチャネルを先に登録してから自分に送るので、check が受け損ねてもテストの binary は落ちない |
| LOW 2(login shell の cwd) | 直った。ただし新しい LOW 4 を生んだ | `BuildScript(root, ...)` は場所ごとの subshell の頭で `cd '<root>'` か `cd '<root>'/'<dir>'` を打ち、`mise current` も root に cd する subshell に入れた(`internal/toolchain/probe.go:128-176`)。root も dir も `shellQuote` 済みで、絶対パスなので `CDPATH` も効かない。`Probe` は `filepath.Abs` で Dir を絶対にしてから script と `Run` の cwd の両方に使う。`Report.Mise` と `ProbeOptions.Dir` の doc も直っている。`TestProbe_LoginShellThatCdsElsewhereMeasuresEachPlace` は、別のディレクトリへ cd してから script を打つ包みの shell で、3 か所と `mise current` がそれぞれの場所で測られることを見る |
| LOW 3(`FindLayout` の error を捨てる) | 直った。ただし新しい LOW 2 を生んだ | `doctorNodeLayout` は `findErr` を返し、info の行と note の両方が `nodeErrText(findErr)` を出す。時間切れ(`context deadline exceeded`)と git の exit が見分けられる。名前に改行のあるディレクトリでも 1 行に収まる(テストあり) |
| LOW 4(`ProbeEnv` を 2 回当てる) | 残した(orchestrator の判断) | `internal/cli/doctor_node.go:152` はまだ `toolchain.ProbeEnv(os.Environ())` を渡し、`ProbeOptions.Env` の doc(`internal/toolchain/probe.go:370`)も変わっていない。結果は同じなので、下の表に「残すと決めたもの」として 1 行で残す |
| LOW 5(「15 seconds in all」) | 直った | gate の help は「The probe's shell has 15 seconds」と、列挙(10 秒まで)と前後の `git status` が「come on top of that, all while gate.lock is held」と書く。regate の help、`/org` skill(4 つの写し)、手順書(2 つの写し)も同じ中身。`KillGrace` の分は書いていないが、「shell の予算」と言っているので嘘にはならない |
| LOW 6(「installs and writes nothing」) | help・手順書・skill は直った。package の doc に同じ形が残る(新しい LOW 3) | help は「runs with the variables that turn off ...」と「A probe that times out or fails does not change the verdict」に、手順書は「which keeps go from switching ...」と「records a pass as `tree_changed`」になった。旧い文言を `grep` すると、Go と md のどこにも残っていない(1 回目の report を除く) |
| LOW 7(中断の文面) | 直った | `gateRun.started` は最初のスクリプトの直前に立ち、`abort` は error のときに `started` だけを返す。`gateRunInterrupted` は log なし、log ありでスクリプト前、スクリプト後の 3 通りを書き分け、gate と regate で同じ本体を使う(prefix は `gateWords.prefix` と `regatePrefix`、どちらも前の文字列と同じ)。`gateCheckAfter` と記録の失敗は `true` を渡し、そこまで来ればスクリプトは必ず 1 つ走っている。log を作る前の呼び出しは全部 `""` と `false` で、`regate_merge.go` の 7 か所も揃っている。leader と reviewer の prompt と skill はこの文で分岐していない(`grep 'scripts were stopped\|before the scripts ran'` が prompt と skill で 0 件) |

ほかに見たもの。

- `internal/cli/doctor_node.go` の 1〜260 行(変わった部分の全部)と、`doctor.go` の結果の表示と終了コード(`:254-284`)
- `internal/toolchain/probe.go` の `probeFunc`、`BuildScript`、`writeCD`、`ParseProbe`、`Probe`。`internal/toolchain/version.go` の `cleanText`、`projects.go` の `trackedFiles`
- `internal/org/gate.go` の `Gate` と中断の 3 関数、`gate_run.go` の `runGateScripts` の全部、`regate.go` の `regateGate` と `regateInterrupted`、`regate_merge.go` の差分、`gate_tools.go` の `gateToolsOneLine` と probe の呼び出し
- 修正の commit のテスト(doctor の中断、`nodeErrText`、`BuildScript` の固定、包みの shell、相対の Dir、regate の「測る間の中断」)
- `templates/base/` の 5 つの写しは `cmp` で同じ、`.claude` と `.agents` の org skill も同じ。`git diff --check origin/main...HEAD` は rc 0
- tech-debt の 4 行(1 回目のまま)と、`internal/toolchain/doc.go`、AGENTS.md、reviewer の prompt の追加行

1 回目で確かめて今回は繰り返さなかったこと: 場所の名前の濾しと引用、記録の形、`Run` の止め方と `cappedBuffer`、記録の値の確かめ直し、doctor が fail を返さないこと(1 回目の report の Evidence reviewed を参照)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | exception-handling | doctor は probe の間の Ctrl-C を飲み込み、終了コード 0 で「All checks passed.」と出す。`signal.NotifyContext` が SIGINT・SIGTERM・SIGHUP の既定の動き(終わる)を止め、check は warn の 1 行を返すだけなので、Check 14 が最後の check である doctor はそのまま全部の結果を表示する。warn は `allPass` を落とさない。打った人が止めたつもりでも、`ralph doctor && 次のコマンド` は次へ進む(bash は子が signal で死ななかったとき、子が SIGINT を処理したとみなして続ける)。gate、regate、merge、cleanup は中断を error にして終了コード 1 で終わるので、doctor だけ違う。最初の Ctrl-C のあとも登録は `stop()` まで残るので、group を止める間(最大で `KillGrace` の 2 回分)の 2 回目の Ctrl-C も効かない | `internal/cli/doctor_node.go:49-51`、`:156-159`、`:173-177`。`internal/cli/doctor.go:258-281`(`allPass` は `fail` でしか落ちない)。比べる先は `internal/cli/org_regate.go:250` と `internal/org/gate.go` の `gateRunInterrupted` | probe が signal で止まったことを `checkNodeToolchain` の呼び手に返し(戻り値に cause を足す、など)、`runDoctorStateDir` は結果を表示したあとに `doctor: interrupted (<cause>)` の error を返して終了コード 1 にする。表示を残すかどうかは好みでよいが、0 で終わらせない |
| LOW | maintainability | 修正で足した `nodeErrText` は `toolchain.cleanText` と同じ処理の写しで、同じ `FindLayout` の error を gate はさらに別の `gateToolsOneLine` で 1 行にする。3 つの濾しのうち 2 つは判定が同じで、`gateToolsOneLine` は C1 を通し、不正な UTF-8 は `strings.Map` が置換文字(U+FFFD)にし、上限も持たない。`cleanText(s, 240)` と `nodeErrText` の出力は切り方まで同じになる(どちらも 237 バイトに `...`)。片方の規則を直すと、残りが黙ってずれる | `internal/cli/doctor_node.go:208-228`、`internal/toolchain/version.go:86-105`、`internal/org/gate_tools.go:144`、`:209-216` | `toolchain` に `CleanText(s string, limit int) string` を出して doctor と gate の両方から使い、`nodeErrText` を消す。gate の log の行にも同じ上限がかかる |
| LOW | readability | 1 回目の LOW 6 と同じ言い切りが package の doc に残っている。`internal/toolchain/doc.go` は「Measuring changes nothing (AC10). The package installs nothing and writes no file.」と書き、同じ段落の最後で「A tool that ignores those variables can still write」と言う。修正は help、手順書、skill を直したが、この doc は 1 回目の指摘にも名前がなく、直っていない | `internal/toolchain/doc.go:15-27`。`BuildScript` の doc(`internal/toolchain/probe.go:128-140`)は「never installs or writes anything itself」で、`itself` が付いているので合っている | 「The probe script itself installs nothing and writes no file」とし、「Measuring changes nothing」は「is set up to change nothing」のように弱めるか消す |
| LOW | maintainability | `mise current` を root に cd する別の subshell に移したので、root の nocd の記録が 2 回出うるようになった。`ParseProbe` は nocd を場所の名前だけで覚え(`noCD[dir]`)、`case noCD[t.Dir]` を `st.exited` より先に見るので、root の subshell で測れたツールも、後の mise の subshell の cd が落ちれば「cannot cd into the directory」の unknown になる。起きるのは 2 つの subshell の間に root に入れなくなったときだけで、gate の worktree ではまず起きない。ただ、「場所ごとに subshell は 1 つ」という `ParseProbe` の前提を、修正が黙って崩した | `internal/toolchain/probe.go:155-158`(mise の subshell の `writeCD(&b, root, RootDir)`)、`:279`(`noCD[dir] = true`)、`:325`(`case noCD[t.Dir]`) | `ParseProbe` の switch で `st != nil && st.exited` の場合を `noCD` より先に見る(場所の subshell の cd が落ちたなら、その場所のツールの state は無いので結果は変わらない)。あるいは mise の nocd の記録に `miseLabel` を入れ、`noCD` を (dir, label) で引く |
| LOW | readability | (1 回目の LOW 4、残すと決めたもの)doctor は `toolchain.ProbeEnv(os.Environ())` を渡し、`Probe` がもう 1 回 `ProbeEnv` を当てる。冪等なので結果は同じ | `internal/cli/doctor_node.go:152`、`internal/toolchain/probe.go:370` | 残すなら、`ProbeOptions.Env` の doc に「ProbeEnv を当て済みの環境でもよい」と 1 行足すと、次に読む人が迷わない。直さなくてもよい |

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5(新しい 4 件と、1 回目から残した 1 件)。

## Positive notes

- 中断の文面を 3 通りに分け、gate と regate で 1 つの本体(`gateRunInterrupted`)に寄せた。`started` を error のときにも返す `abort` の閉包で、ループの 6 か所の return が同じ形になった
- 包みの shell に `cd` させてから script を打たせるテストは、まず `pwd` で包みが本当に別の場所へ動くことを確かめ(対照)、そのうえで 3 か所と `mise current` を見る。対照があるので、`cd` を外しても通ってしまうテストになっていない
- doctor の中断のテストは、偽の node が sleep のあとに書くファイルの有無で「group が止まった」ことを見ていて、warn の文面だけを見るテストより強い
- `nodeErrText` のテストは、文字の途中での切り、上限ちょうど、C1(`\u0085`)、不正な UTF-8 を並べ、出力が上限以下で正しい UTF-8 であることを全件で確かめている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

この review で新しく足す行はない。4 件の新しい LOW はどれも小さく直せるので、直すか残すかは orchestrator が決める。cycle 1 のまま `/pr` に進んで残す場合は、残した LOW を 1 行にまとめて `docs/tech-debt/README.md` に書く(計画は `/pr` で archive され、ほかに持ち越す場所がない)。1 回目の LOW 4 は orchestrator が「直さない」と決めたもので、あとで払う作業がないので行にしない。差分が足した 4 行は 1 回目のままで、修正の commit と食い違わない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 5 件)
- Follow-ups:
  - doctor の中断を終了コード 1 にする(1 行目)のが、人の操作に直接かかわるので、直す価値がいちばん高い
  - `toolchain.CleanText` を出して 3 つの濾しを 1 つにする(2 行目)と、`doc.go` の言い切り(3 行目)、`ParseProbe` の nocd の順(4 行目)は、同じ commit で直せる
