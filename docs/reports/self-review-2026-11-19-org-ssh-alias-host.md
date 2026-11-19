# Self-review report: org-ssh-alias-host

- Date: 2026-11-20(3 回目。パイプライン cycle 2。cross-review の ACTION_REQUIRED #1 を直した `240324fe` のあと。`cycle-count.json` は `cycle: 2`)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`
- Branch: `fix/org-ssh-alias-host`、HEAD `e5cad0e5`(計画だけの commit)、コードの最後は `240324fe`、merge base は `main` の `e6d5d0ed`
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff main...HEAD` の全体(40 ファイル、+2864 −216)の diff の質だけ。重点は `240324fe`(doctor の Merge guarantee の検査を signal で止める。`doctor.go`、`doctor_merge_guarantee.go`、新しい `doctor_merge_guarantee_signal_test.go`)。テストと静的解析は走らせていない
- 番号: 前回までの L1〜L10 の番号は変えない。Go の test のコメントがこの report の `L1`・`L2`・`L3` を引いているため。新しい指摘は L11 から振る

## Evidence reviewed

- `240324fe` の全 diff(4 ファイル、+361 −23)。変えた関数は HEAD の全文で読んだ: `runDoctorStateDir` の Check 11c から 14、`printDoctorResults`、`doctorInterrupt`、`checkMergeGuarantee`、`runMergeGuaranteeCheck`、`doctorMergeGuaranteeSignalContext`。新しい test ファイルは 266 行を通して読んだ
- 抽出の前後比べ: `git show 240324fe^:internal/cli/doctor.go` の印字の塊と `printDoctorResults` を突き合わせた。Node の検査の印字は 1 文字も変わらない: `Interrupted: the Node toolchain check did not finish. Run ralph doctor again.` と、2 つの error 文(`...while the Node toolchain check measured the tools` に `; %d check(s) failed` が付く場合と付かない場合)は、`nodeToolchainCheckName` と `"measured the tools"` を `%s` に入れると元の文字列になる
- `ssh -G` を打つ道の全数: 非 test で `"ssh"` を実行するのは `mergeorigin.go:186` の 1 か所。`resolveMergeAPIHost` の呼び手は `ReadMergeOrigin`(`mergeorigin.go:140`)と doctor の `readOrigin`(`mergeguarantee.go:311`)の 2 つ。`ReadMergeOrigin` の呼び手は `merge.go:429`、`regate_ci.go:94`、`internal/cli/org_cleanup.go:206` の 3 つ。`CheckMergeGuarantee` の呼び手は doctor だけ
- その 4 つの入口の ctx: `ralph org merge`(`org_merge.go:274`)、`ralph org cleanup`(`org_cleanup.go:168`)、`ralph org regate`(`org_regate.go:254`)は `signal.NotifyContext(parent, os.Interrupt, SIGTERM, SIGHUP)` で ctx を作る。`rt.Merge(`・`rt.Regate(` の呼び出しはこの 2 か所だけで、`MergeOrderParams{Origin: ...}` を渡すのは cleanup だけ。`ralph status` の `ReadMergeGuarantee` と `ralph org merge-order` は ssh を打たない(`resolveMergeAPIHost` の呼び手の全数に入らない)。merge と regate(ci)が signal のあとも走らせる `context.WithoutCancel` の呼び出しは gh、manifest の lock、git の tracking ref で、ssh はない
- test の安全の確かめ: `signal.Notify(caught, sig)` を `Kill` の前に登録している。`os/signal` の `process` は希望する全 channel へ 1 回の呼び出しの中で送る(`$GOROOT/src/os/signal/signal.go:245-262`)ので、NotifyContext が ctx を終わらせた時点で `caught` にも届いている。`internal/cli` に `t.Parallel()` はない。`signalSelf` と `doctor_node_test.go:765` が同じ手順で本物の signal を打つ先例
- 実プロセスの kill の確かめ: `realMergeDetachedExec`(`mergeorder.go:1407`)は `Setpgid` と `kill(-pid, SIGKILL)`。`doctorMergeGuaranteeSignalContext` の doc の「the group gets SIGKILL」と合う
- register: `git diff main...HEAD -- docs/tech-debt/README.md` を 240324fe の変更と突き合わせた(両方向)。L4・L6・L7・L9 を名指す行は register にない
- 掃き出し: `internal/cli/doctor*.go` と `mergeguarantee.go` の「exit code」「never fail」の文、`checkMergeGuarantee(`・`runMergeGuaranteeCheck(`・`printDoctorResults(` の呼び手の全数

## 前回の指摘の扱い

| 前回 | 結果 | 根拠 |
| --- | --- | --- |
| L1(入力の host の検査が出力より緩い) | 解決(前回から変わらない) | 入力と出力が同じ述語 `mergeSSHHostName` を通る。`ssh` を打つのは `mergeorigin.go:186` の 1 か所で、その前に必ずこの述語を通る。許可リストは空でなく、`-` で始まらず、全バイトが `A-Z a-z 0-9 . _ : -` |
| L2(merge と cleanup の `--help` が古い) | 解決(L8 の分も `3c369ff4` で直った) | `org_merge.go` と `org_cleanup.go` の help に、ssh の origin の host は `ssh -G <host>` の答えで、`Match exec` の命令が走ると書いてある。`Help` の test が文を固定する |
| L3(doctor が「ssh -G が走ったか」を条件から写す) | 解決(前回から変わらない) | `mergeSSHResolveRuns(originURL, host)` を解く関数と doctor が共有する(`mergeorigin.go:218`、`mergeguarantee.go:312`)。`mgDoneOnceHostRead` は旧い条件なら落ちる |
| L5(最後の `switch` の 3 つ目の条件が重い) | 解決(前回から変わらない) | `!mergeSSHHostName(resolved)` の 1 つに畳まれている |
| L8(L2 の直しが regate の help と doctor のコメントに届いていない) | 解決 | `org_regate.go` の ci の段落が「`ssh -G` connects to nothing, but it runs the commands of any Match exec in the ssh config」と言う。`doctor.go` の `runDoctorOpts` の上のコメントが `ssh -G <host>` を、自分のプロセスグループで、期限 5 秒で、host が普通の名前の ssh の origin のときだけ、と数え上げる。`240324fe` がこのコメントに signal の 1 句を足したが、数え上げは崩れていない |
| L10(`which gh cannot take as a host` が ralph の規則を gh のものと言う) | 解決 | `mergeSSHHostNameWhy` が入力側と出力側の 2 つの理由の末尾を同じ語で作り、「ralph の規則であって gh の規則ではない」と doc に書く。`-` で始まる場合だけ別の語になる |
| L4(`resolveMergeAPIHost` の形) | 残した。許容できる(今回も変わらない) | `resolveMergeAPIHost`(`mergeorigin.go:169`)の引数 7 つと `originURL, host string` の並び、`sshRun == nil` の既定が `ReadMergeOrigin`(`:119`)と中(`:177`)の 2 か所、「was interrupted」の枝は本番の呼び手が読まないこと、`mergeSSHResolveRuns(originURL, host string)`(`:218`)が同じ形を 1 つ足していること。`240324fe` は触らない |
| L6(偽の ssh のプロセスグループの確かめが 3 か所) | 残した。許容できる(今回も変わらない) | `mergeorigin_test.go`、`mergeguarantee_alias_test.go`、`regate_ci_ssh_alias_test.go` に同じ `ps -o pgid=` の確かめがある。`240324fe` の test は偽の exec で ctx の終わりを待つだけで、この手順を使わないので 4 つ目にはなっていない |
| L7(test の表の読みにくさ) | 残した。許容できる(今回も変わらない) | 位置指定 7 欄の表と、`map[bool]int{true: 1, false: 0}[tc.sshRuns]` が `mergeguarantee_alias_test.go:167` と `:360` の 2 か所 |
| L9(`%q` にした文の隣で、hint が同じ host を `%s` で出す) | 残した(利用者の設定の host なので危険は低い)。HEAD でも同じ | `mergeSSHAliasHint`(`mergeorigin.go:275-281`)が `apiHost` と `originHost` を `%s` で出し、同じ文の `why` は `%q` で host を引く |

## Findings

CRITICAL、HIGH、MEDIUM はない。LOW は 6 件。前回から残した 4 件(L4、L6、L7、L9)と、新しい 2 件(L11、L12)。

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L11: 新しい test が「自分のプロセスに signal を送る」手順を 2 回、手で書いた。`signal.Notify(caught, sig)` で test 自身の channel を先に登録し、`t.Cleanup` か `defer` で `signal.Stop` し、`syscall.Kill(os.Getpid(), sig)` を打つ。同じ手順は `signalSelf(sig)` にあり、doc が同じ理由(登録がなければ、捕まえ損ねた signal が test の binary を終わらせる)を書き、`org_regate_test.go` と `org_wait_director_test.go` が使っている。手書きは `doctor_node_test.go:765` ほかにもう複数あり、そこへ 2 つ足した。順序の問題はない(NotifyContext を先に作り、`caught` を後で登録しても、1 回の `process` が両方へ送る) | `doctor_merge_guarantee_signal_test.go:116-119` と `:246-253`。`org_gate_test.go:817`(`signalSelf`)、`org_regate_test.go:639,722`、`org_wait_director_test.go:592,622` | 表の `setup` の `interrupt` を `func() { signalSelf(syscall.SIGINT)(t, nil) }` に、2 つ目の test の `Notify` から `Kill` までの塊を `signalSelf(sig)(t, nil)` に置き換える。直さなくても挙動は変わらない |
| LOW | exception-handling | L12: `runMergeGuaranteeCheck` は、検査が返った時点で ctx が終わっていれば原因を返す(`doctor_merge_guarantee.go:115`)。signal が検査の最後の命令のあとに来た場合、検査は本物の結果を出し終えている(`checkMergeGuarantee` の doc も「unless the signal came after its last command」と書く)のに、doctor は `Interrupted: the Merge guarantee check did not finish.` と出し(`doctor.go:320`)、行は `pass` や `warn` のまま、後の検査を打たずに error で終わる。Node の検査は probe が割り込まれたときだけ `interrupted` を返すので、2 つの検査で「割り込まれた」の意味が違う。窓は `CheckMergeGuarantee` が返ってから `Err()` を読むまでの数命令で、後の検査を打たないのは Ctrl-C として正しい。ずれるのは「did not finish」の 1 句だけ | `doctor_merge_guarantee.go:111-119`、`doctor.go:244-249`、`:320`。Node 側は `doctor_node.go` の `checkNodeToolchain` の doc(probe の error を `interrupted` として返す) | 残してよい。直すなら banner を「did not finish」に頼らない形(例: `Interrupted while the %s check ran.`)にする。2 つの検査で共有する文なので、Node の test の期待文も一緒に変わる。ほかの案(`interrupted` を、検査の結果が割り込みの unknown のときだけ返す)は `org` 側に印が要り、大きい |
| LOW | maintainability | L4(残した): 上の表のとおり。`resolveMergeAPIHost` の形(引数 7 つ、`originURL, host string` の並び、既定の二重) | `mergeorigin.go:119,169,177,218` | 動きは変わらない。直すなら host を `originURL` から中で出し、`sshRun` の既定を片方に置く |
| LOW | maintainability | L6(残した): 偽の ssh を PATH に置いてプロセスグループを確かめる手順が 3 つの test ファイルに複製 | `mergeorigin_test.go`、`mergeguarantee_alias_test.go`、`regate_ci_ssh_alias_test.go` | `soFakeSSH` の隣に共通の確認を置く |
| LOW | readability | L7(残した): 位置指定の 7 欄の表と、`map[bool]int{...}[tc.sshRuns]` が 2 か所 | `mergeguarantee_alias_test.go:167,360` | 欄名つきの struct literal と、`want := 0; if tc.sshRuns { want = 1 }` |
| LOW | security | L9(残した): `mergeSSHAliasHint` が host を `%s` で出す(同じ文の `why` は `%q`)。host は利用者の git config の値で、制御文字を持てるのは scp 形の origin だけ | `mergeorigin.go:275-281` | `apiHost` と `originHost` も `%q` にし、期待文を直す。または生で出すと決めて、`%q` と混ぜない |

## 依頼された点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| 直しは正しいか | 正しい | 検査だけを `signal.NotifyContext` の ctx で走らせ(`doctor_merge_guarantee.go:111-119`)、`defer stop()` が検査の返りで signal を元に戻す。`SaveMergeGuarantee` と後の検査は handler の外。ctx が終わると `CheckMergeGuarantee` の `stopped` が親の ctx を見て「the check was interrupted (<原因>) while <段> ran」の unknown を作り(`mergeguarantee.go:670-674`)、`resolveMergeAPIHost` の `sshCtx` は同じ親から派生するので、`realMergeDetachedExec` が ssh のグループを SIGKILL する。原因は `context.Cause` でそのまま伝わる(test が `interrupt signal received` を ssh 側で確かめる) |
| 完全か(signal の ctx を持たず `ssh -G` を打つ道が他にあるか) | 他にない | `ssh -G` の呼び手の全数は上の Evidence のとおり。merge・cleanup・regate の 3 つの入口は元から `NotifyContext`、doctor が最後の 1 つで、`240324fe` で閉じた。`status` と `merge-order` は ssh を打たない |
| 本物の SIGINT の test は test binary を殺す、または handler を漏らすか | どちらもしない | `caught` を `Kill` の前に登録するので、ralph 側の handler がなくなる回帰があっても binary は終わらない(その場合、偽の ssh は 5 秒の期限で戻り、原因の文字列が合わず test が落ちる。待ちは止まらない)。NotifyContext の `stop` は `runMergeGuaranteeCheck` の `defer` と、test の `released` の数(2 回)で確かめられる。seam の差し替えは `t.Cleanup` が LIFO で元に戻す。`taken`・`released` は doctor を 2 回走らせた数と合う |
| Node の検査の動きは変わらないか | 変わらない | 印字の文字列が 3 つとも同じ(Evidence)。`checkNodeToolchain` と `doctorNodeSignalContext` は `240324fe` が触らない。`allPass`・`countFailed` の読み方も同じ |
| `printDoctorResults` / `doctorInterrupt` の抽出は動きを変えないか | 変えない | 印字の塊が移っただけ。違いは 2 つで、どちらも意図どおり: (1) 割り込みの文が `%s` で組まれる(上の比較)、(2) Merge guarantee の割り込みでは Check 12 から 14 を打たずにここへ来る。`defer` は `runDoctorStateDir` にない(`defer` は `:367` と `:1225` の別の関数だけ)ので、早い return が取りこぼす片付けもない |
| test は revert で落ちるか | 落ちる(読んで確かめた。走らせていない) | seam を使わなければ `taken != 2` と原因の文字列で落ちる。`mergeGuaranteeInterrupted` の早い return を消すと Node の行が出て、行の列が一致しない。`context.Cause` を `Err()` にすると原因が `context canceled` になる。`defer stop()` を消すと `released` が合わない。3 つの signal のどれかを `NotifyContext` から外すと 2 つ目の test が落ちる |

## 確かめたが挙げなかったもの

- 割り込みの unknown が cache に書かれる: `SaveMergeGuarantee` の doc は「unknown included」「古い kind を残さない」で、計画の Progress も「キャッシュの規則は変えていない」と言う。timeout の unknown と同じ扱いで、設計どおり。Ctrl-C が良い cache を unknown に置き換えることは、この規則の帰結
- test のコメントが `docs/reports/cross-review-triage-<slug>.md` の `ACTION_REQUIRED #1` を引く(`doctor_merge_guarantee_signal_test.go:19-20`): `internal/` の Go に triage の path を引くコメントが多数ある(`migrate.go`、`upgrade_v2.go`、`doctor.go`、`spawn_test.go` の `ACTION_REQUIRED #1` の形も)。path つきで引いているので、この repo の慣習の範囲
- `signal.NotifyContext(parent, os.Interrupt, SIGTERM, SIGHUP)` の並びが 6 か所になった(org の 4 つの入口と、Node と Merge guarantee の 2 つの seam)。先例の形に従っていて、seam を分けているので test が検査ごとに差し替えられる
- `checkMergeGuarantee`・`runMergeGuaranteeCheck`・`org.CheckMergeGuarantee` と `mergeGuaranteeResult`・ローカル変数 `mergeGuaranteeResults` が近い名前: 各関数の doc が層の違いを言っている。`runMergeGuaranteeCheck` を切り出した理由(`defer stop()` を検査だけにかける)は doc から読める
- 名前つき戻り値 `(results []checkResult, interrupted error)` の `results` は使われず、`c, interrupted := ...` は同じ block の `interrupted` に代入する(名前つき戻り値と同じ scope で、`c` が新しいので `:=` が通る): 読みにくいが正しい

## Positive notes

- 割り込みの扱いを Node の検査と同じ形に揃えた(seam の名前 `doctorMergeGuaranteeSignalContext` と `doctorNodeSignalContext`、doc の構成、「Tests replace it」)。次に doctor へ長い外部コマンドの検査を足す人が写す先が 2 つになった
- 抽出した `printDoctorResults` は、`runDoctorStateDir` の末尾にあった印字を、割り込みの 2 つの経路が同じ形で使える関数にした。Node の経路の文字列は変えず、test(`doctor_node_test.go`)が固定する期待文も触らない
- 直しが自分の文を直した: `checkMergeGuarantee` の doc の「doctor's exit code never changes because of it」を「because of the kind」に変えた。割り込みで exit 1 になるので、元の文は偽になっていた。Check 11c のコメントと `runDoctorOpts` の上のコメントにも signal の 1 句がある
- test は、signal が来たら ssh -G の ctx がその原因で終わること、コマンドが `git`・`ssh` で止まり gh を打たないこと、行の列(割り込みの行の前は signal なしの run と同じ)、cache、`Interrupted:` の行、error の文、`taken`・`released` の数を、1 つの表で 3 通り(seam で終わらせる、台帳が解けない run で終わらせる、本物の SIGINT)確かめる。signal なしの run を先に走らせて同じ fixture の基準にしてある
- `TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP` が 3 つの signal を別々の subtest で、原因の文字列つきで確かめる。`sig.String() + " signal received"` は `org_gate_test.go` の期待文と同じ形

## Tech debt identified

この run はパイプラインの 2 回目で、cap は 2(`cycle-count.json` に `cap` はなく、`[standard].max_pipeline_cycles` の既定のまま)なので最後の run になる。直さずに進める LOW は、そのまま延期になる。L4・L6・L7・L9 は前の run から register にない(`docs/tech-debt/README.md` にこれらを名指す行はない)。次の行を `/sync-docs` が足す(直した文字は消す)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small leftovers of fix/org-ssh-alias-host (#340), found by its self-review and left in place at the last pipeline run; all LOW. (a) `resolveMergeAPIHost` in `internal/org/mergeorigin.go` takes seven arguments with `originURL, host string` side by side, its `sshRun == nil` default is set both there and in `ReadMergeOrigin`, and its clause for a host that ran no ssh is chosen by elimination after `mergeOriginIsSSH` (self-review L4). (b) The check that a fake `ssh` on PATH ends with its children (`ps -o pgid=` and the three-way test of the pgid) is written out in `mergeorigin_test.go`, `mergeguarantee_alias_test.go`, and `regate_ci_ssh_alias_test.go` (L6). (c) Those tests' tables have seven positional fields, and `map[bool]int{true: 1, false: 0}[tc.sshRuns]` appears twice in `mergeguarantee_alias_test.go` (L7). (d) `mergeSSHAliasHint` prints `apiHost` and `originHost` with `%s` in a sentence whose `why` quotes the host with `%q` (L9). (e) doctor's `Interrupted: the Merge guarantee check did not finish.` is also printed for a signal that came after the check's last command, when the check had its real result (L12). (f) `internal/cli/doctor_merge_guarantee_signal_test.go` writes the steps that send a signal to the test process by hand twice, where `signalSelf` in `internal/cli/org_gate_test.go` does it (L11) | LOW. Long or repeated test code and two strings that are a little off; no behavior is wrong, and (d) matters only for a host with a control character in a git config the user wrote | The branch fixed the findings that changed what a user reads or what ran (L1, L2, L3, L5, L8, L10) and the cross-review's one finding; these are in code and tests that behave correctly, and the pipeline is at its last run, where a fix of code starts it again | The next change to `resolveMergeAPIHost` or to the tests of `mergeorigin.go` and `mergeguarantee.go` for (a) to (d); the next change to doctor's interrupt banner (`printDoctorResults`) for (e); the next signal test of `internal/cli` for (f) | `docs/plans/active/2026-11-19-org-ssh-alias-host.md`; self-review of org-ssh-alias-host (L4, L6, L7, L9, L11, L12), `docs/reports/self-review-2026-11-19-org-ssh-alias-host.md` |

この run は、依頼された範囲(report だけの commit)に合わせて `docs/tech-debt/README.md` を変えていない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 6 件。前回の 6 件のうち L8・L10 が解決し、L4・L6・L7・L9 が残り、新しい 2 件を足した)
- Follow-ups:
  - L11 と L12 は小さく、直すと code の変更になる。cap が 2/2 で、code の直しは「記録だけの修正」の例外に入らないので、直せば full pipeline の再走になる。直さずに上の register の行へ入れるほうが小さい
  - `/sync-docs` へ(数えない。文書のずれ): (1) register の「Five files went past the 800-line guide, or grew past it, with script-local-config」の行の fix/org-ssh-alias-host の注は、`internal/cli/doctor.go` が「1651 → 1653 lines」で、変えた commit は `3c369ff4` だけと言う。`240324fe` が `printDoctorResults`・`doctorInterrupt` と signal の句を足して 1683 行になり、変えた commit も 2 つになったので、数と commit を直す。(2) 計画の Progress が申し送った「doctor の検査が signal で止まるときの動きが README にない(Node の検査も)」は `/sync-docs` の判断。(3) README の doctor の行の exit code の文に、割り込まれたときに exit 1 になることを足すかどうか
  - 計画に doctor の signal の AC はなく、Progress の行が「計画からのずれ」として記録している。digest で固定された計画を触らないやり方として妥当で、`/pr` が archive する記録に残る
- Known gaps:
  - doctor の既定の経路(`NotifyContext` の ctx から `realMergeDetachedExec` まで)を本物の `ssh` のプロセスグループで通した確かめはしていない。test は偽の exec が ctx の終わりを待つ。グループごと止まることは org 層の実プロセスの test(AC5b)が、ctx が終わるまでの道は doctor の test が、それぞれ見る
  - 端末を閉じたとき(SIGHUP)の doctor の動きを本物の端末では見ていない。signal の種類ごとの原因の文字列は test が見る
  - `signal` の順序の議論(`caught` と NotifyContext の channel がどちらも届く)は `os/signal` のソースを読んだだけで、走らせていない
  - テスト、静的解析、spec の合致、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の範囲)
