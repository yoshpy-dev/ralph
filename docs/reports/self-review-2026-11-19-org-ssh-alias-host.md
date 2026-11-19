# Self-review report: org-ssh-alias-host

- Date: 2026-11-19(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`
- Branch: `fix/org-ssh-alias-host`、HEAD `c36f511e`、base は `origin/main` の `a8c5e289`
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff origin/main...HEAD` の 24 ファイル(+1734 −178)の diff の質だけ。コードは `554d04aa`(`resolveMergeAPIHost`、`ReadMergeOrigin` の ssh 実行関数、merge・regate ci・cleanup の呼び手、merge の案内)と `40afa91e`(doctor の Merge guarantee、キャッシュの `api_host`、別名の注記、取り消しの文)。文書の commit は `44a56862`、`4ec556e4`、`d19a2fa6`。テストと静的解析は走らせていない

## Evidence reviewed

- コード: `internal/org/mergeorigin.go` 全体(`ReadMergeOrigin`、`resolveMergeAPIHost`、`mergeOriginIsSSH`、`mergeSSHAliasHint`)、`mergeguarantee.go` の差分と `stopped`・`getMergeOriginURL`・`ReadMergeGuarantee`、`merge.go` の `readOrigin`・`ghAccess`・`ghMissing`・`mergeReady`、`regate_ci.go` の `newRegateCI`、`mergeenforce.go` の `CheckMergeEnforcement`(`sshRun` を持たない `mergeGuaranteeRun`)、`internal/cli/org_cleanup.go`
- 周辺の既存コード: `mergeOriginFromURL`(`mergeorder.go:945`)、`realMergeDetachedExec` と `runMergeOrderCommand`(`mergeorder.go:1407-1438`)
- テスト(読んだだけ): `mergeorigin_test.go`(新規、412 行)、`mergeguarantee_alias_test.go`(新規、359 行)、`merge_test.go`・`mergeguarantee_test.go`・`regate_ci_test.go`・`mergeorder_test.go`・`merge_fake_test.go` の差分、`internal/cli/org_cleanup_steps_test.go`
- 文書: README の 2 か所(語単位の差分)、org skill の差分、spec の 7c-2 の 1 文、tech-debt の 3 行(RESOLVED の註、打ち消しの行、新しい director skill の行)、`docs/evidence/ssh-alias-host-livefire-2026-11-19.md`
- 探したもの: `cannot resolve|not resolved|ssh alias` と `get-url origin ran` を非 test の Go と文書で検索(古い言い回しの残り)。`mergeOriginFromURL(`・`getMergeOriginURL(`・`--hostname` の使い手(host を直さずに gh へ渡す所の残り)。`internal/cli` のテストで ssh 形の origin を使う所(本物の `ssh` が走る道)
- probe: URL の解析が host に何を通すかを、scratch の 20 行の `go run` で確かめた(L1)

## Findings

CRITICAL、HIGH、MEDIUM はない。LOW が 7 件。

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | security | L1: `ssh -G <host>` に渡す host の入力側の検査は先頭の `-` だけで、出力側の hostname の検査(空白、制御文字、`/`、不正な UTF-8)より緩い。host は origin の URL から来る。URL の解析は `ssh://git@$(id)/o/r.git` の host を `$(id)` として、`ssh://git@a;touch/o/r.git` を `a;touch` として通し、scp 形はコロンの前なら何でも host になる(scratch の `go run` で確かめた)。`~/.ssh/config` に `Match exec "... %h"` があると、この host が展開されてシェルに渡る。OpenSSH 9.6 より前の既知の問題(CVE-2023-51385)で、gh も同じ。ただし origin は利用者自身の git config で、`Match exec` に `%h` を書く設定が要るので、危険は低い。CVE の記憶はこの repo では未確認 | `mergeorigin.go:175`(入力は `-` だけ)、`:207-210`(出力は文字を見る)、`mergeOriginFromURL`(`mergeorder.go:945-975`) | 出力と同じ検査を入力にも掛ける(許可リストなら `[A-Za-z0-9._:\[\]-]`)。掛けた host は、今の `-` の枝と同じく ssh を打たず、理由つきで origin の host のまま進む。ssh の `--` は先頭の `-` にしか効かないので、代わりにならない |
| LOW | maintainability | L2: `ralph org merge` と `ralph org cleanup` の `--help` は、host を `git remote get-url origin` から読み、gh へ `-R <host>/<owner>/<repo>` と `--hostname <host>` で渡すと言ったままで、ssh の origin では `ssh -G` が host を決めることも、`Match exec` の命令が走ることも書いていない。README、skill、doc comment は直っているが、利用者が最初に読む help が古い。計画の Risks は「README と doc comment に書く」で、help は挙げていない | `internal/cli/org_merge.go:58-62`、`internal/cli/org_cleanup.go:70-72`(どちらも `git diff` に入っていない) | 各 1 文足す(ssh の origin の host は `ssh -G` の答え、5 秒の期限、`Match exec` が走る)。Go の文字列なので、`/verify` の文書ずれの対象にもなる。help の文字列を固定するテストがあれば一緒に直す |
| LOW | maintainability | L3: doctor の `readOrigin` が「ssh -G が走ったか」の条件 `mergeOriginIsSSH(line) && !strings.HasPrefix(host, "-")` を、`resolveMergeAPIHost` の中の分岐から写している。つなぐのは行末の `// ssh -G ran` だけ。解く関数に「走らない分岐」が増えたとき(L1 の host 検査がまさにそれ)、ここは古いまま、走っていない `ssh -G <host>` を、ctx が終わったときの理由に名指しする | `mergeguarantee.go:311-316`、`mergeorigin.go:172-176` | `resolveMergeAPIHost` が「ssh を打ったか」を返す(3 つ目の戻り値)。呼び手の条件を消せる |
| LOW | maintainability | L4: `resolveMergeAPIHost` の形。(a) 引数 7 つのうち `originURL, host string` が並び、`host` は `originURL` から `mergeOriginFromURL` で出る値なのに、呼び手 2 か所が両方を渡す(取り違えてもコンパイルが通る)。(b) `timeout` は本番の 2 か所とも `mergeSSHResolveTimeout` を渡すので、引数はテストの口と 0 以下の既定にしか効かない。(c) `sshRun == nil` の既定が `ReadMergeOrigin` と `resolveMergeAPIHost` の両方にあり、前者を通る限り後者は動かない。(d) `err != nil && ctx.Err() != nil` の枝(「was interrupted」)の文は、`ReadMergeOrigin` が(`ctx.Err()` を見て)捨て、doctor も `g.stopped` が先に返すので、本番では読まれない。`ReadMergeOrigin(ctx, "", nil, nil, 0)` のように同じ型の runner が並ぶ点も同じ系統 | `mergeorigin.go:117,121-123,142,170,178-183,189-190`、`mergeguarantee.go:311-316`、`internal/cli/org_cleanup.go:203` | 動きは変わらないので、直さなくてもよい。直すなら、host を `originURL` から中で出す、`sshRun` の既定を片方にだけ置く。(d) は、本番で読まれない枝だとコメントに書くか、一般の「failed」に畳む(畳むとテスト 2 つの期待文も変わる) |
| LOW | readability | L5: `resolveMergeAPIHost` の最後の `switch` の 3 つ目の `case` が、3 つの条件を `||` でつなぎ、2 つ目の条件の中に 3 行のクロージャを置く。条件を足すとき(L1 の入力検査と同じ検査を使い回したい場面)に括弧の中を探す形になる | `mergeorigin.go:207-210` | 名前のある関数(例: `mergeHostIsUsable(s string) bool`)に抜き出し、L1 の入力検査と共有する |
| LOW | maintainability | L6: 偽の `ssh` を PATH に置いてプロセスグループを確かめる手順(スクリプトの 1 行、pid と pgid の読み取り、`pgid != pid \|\| pgid == test` の判定とそのエラー文)が、3 つのテストファイルに同じ形で複製されている。`syscall.Getpgrp()` の値を入れる変数の名前が `test` で、テストファイルの中で読むと紛らわしい | `mergeorigin_test.go:398-411`、`mergeguarantee_alias_test.go:343-358`、`regate_ci_test.go:819-832` | `soFakeSSH` の隣に `soAssertOwnProcessGroup(t, groupFile)` を置き、3 か所から呼ぶ。変数名は `testGroup` |
| LOW | readability | L7: テストの表の読みにくさ。(a) `TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes` の表は位置指定で 7 欄、文字列が 4 つ並び(`wantHost, wantWhy, wantSSH` とタイムアウトの `0`)、行を読んでも `""` がどの欄か分からない。(b) `map[bool]int{true: 1, false: 0}[tc.sshRuns]`。tech-debt の org-merge-order の行の (e) が、同じ形の `map[bool]string{...}[ok]` を既に残債に挙げている | (a) `mergeorigin_test.go:96` 以降の表、(b) `mergeguarantee_alias_test.go:157` | (a) は欄名つきの struct literal にする。(b) は `want := 0; if tc.sshRuns { want = 1 }` |

## 依頼された点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| `ssh -G` の子プロセスの扱いと、プロセスグループごとの kill | 問題なし | 本番の既定は `realMergeDetachedExec`(`Setpgid` と `cmd.Cancel` の `kill(-pid, SIGKILL)`、`mergeorder.go:1413-1423`)。stdin は設定しないので null device。`WaitDelay` が、孫が握る pipe を閉じる。実プロセスのテストが、専用の期限と親の ctx の取り消しのどちらでも、呼び出しが戻り孫の `sleep` が終わることを見る(`mergeorigin_test.go` の `RealSSHThatHangs...`)。doctor と regate ci の既定の道、`ReadMergeOrigin` の `sshRun` が nil の既定(cleanup が使う)も、偽の ssh が自分のグループで走ることを見る(cleanup の呼び出しそのものが nil を渡すことは、テストで固定していない)。`setsid` した孫は残る(計画の Risks が認めている) |
| 解いた値を gh の引数に入れる前の検査 | 問題なし(入力側は L1) | 空、`-` で始まる、空白・制御文字・`/`・不正な UTF-8 は、どれも origin の host のまま進み、理由つき(`mergeorigin.go:196-211`)。小文字にしてから `ssh.github.com` の読み替えを重ねる順は、計画と gh の順に合う。表のテストが各行を押さえる |
| ssh の出力・エラーが Detail に入る道 | 問題なし | ssh の stderr は `mergeGuaranteeErrText` を通る(空白の畳み、制御文字の escape、長さの上限)。hostname は `%q` で出る |
| ctx と専用期限の区別 | 問題なし | `ctx` と `sshCtx` を見分けて 3 つの文(取り消し、期限切れ、失敗)に分ける。`ReadMergeOrigin` は ssh のあとに `ctx.Err()` を見て `MergeOriginInterrupted` を返し、doctor は `g.stopped("ssh -G " + host)` で止まる。どちらもテストがある |
| キャッシュの互換 | 問題なし | `api_host` は `omitempty`、`host` は origin に書かれたまま、版は 1。`ReadMergeGuarantee` の照合は `host` のまま。`api_host` のない古いキャッシュを読むテストがある |
| 古い言い回しの残り | 1 件(L2) | `get-url origin ran` を含む取り消しの文は、merge・regate ci・cleanup とそのテストで直っている(残る `mergeorder.go:882` は merge-order 自身の読みで、ssh を打たない)。「doctor は別名を解けない」と言う非 test の文は、残っているものがない |
| 本物の `ssh` が CLI のテストで走る道 | ない | `internal/cli` のテストで ssh 形の origin を使うのは merge-order と status のテストだけで、どちらも `ReadMergeOrigin` も ssh も通らない |
| tech-debt の register を diff と突き合わせた(両方向) | 問題なし | 250 行の行は打ち消しと RESOLVED の註で閉じ、この PR が開いたままにして矛盾する行はない。`ssh.github.com` と `github.com` の間の `origin_changed` の行は、`host` が書かれたままなので変わらない。新しい director skill の行の「about line 772」は、skill の grep の行(772)に合う |

## Positive notes

- 解く関数が `mergeorigin.go` の 1 か所で、doctor・merge・regate ci・cleanup が同じ関数を通る。`MergeEnforcementParams.Host` は解いた値を受け取るだけで、ssh を打たないことがコメントに書いてある
- 解けなかったときの理由(`unresolvedWhy`)が、失敗の種類ごとにコマンド名つきの 1 句になり、gh が失敗したときだけ案内に出る。gh が成功する別名でない origin の案内は変わらない(表のテストが「別名でない」行を押さえる)
- 偽の ssh を使うテストが、プロセスグループという本番の約束を実プロセスで見る(fake の Exec だけでは見えない所)。`soExec` は ctx の期限も記録するので、5 秒の期限と親の期限の短いほうが効くことまで確かめる
- `docs/evidence/ssh-alias-host-livefire-2026-11-19.md` が、8.2.1 と新しい binary の同じ場所の行を並べ、別名でない origin の行が 1 文字も変わらないことと、ssh を PATH から外した案内まで残す。token や鍵のパスは写っていない
- 既存の fake に ssh を別の記録(`sshCalls`)として足し、呼び出しの番号で動く `after` を壊さない作りにした

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(今は) | - | L1 から L7 は、この PR の中で直せる。cycle 1 の推奨は「直す」で、先送りは決めていない | 最終の cycle まで直されずに残った行は、1 行にまとめて register に足す(候補は L2、L4、L6、L7) | - |

この run は、依頼された範囲(report だけの commit)に合わせて `docs/tech-debt/README.md` を変えていない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 7 件)
- Follow-ups:
  - L1、L3、L4、L5 は Go のコード、L2 は利用者に見える help の文字列、L6、L7 はテストの直し。どれも「記録だけの修正」の例外に入らない。直すなら `/verify` の前にまとめて直すと、パイプラインの再走を足さずに済む。直さずに進めるなら、PR 本文の Known gaps に書く
  - L1 と L5 は同じ関数に触る(host の検査を 1 つの関数に出し、入力と出力の両方に使う)ので、一緒に直すと 1 回で済む
  - 計画の Progress checklist の「Review artifact created」はこの report で埋まる。印は呼び出し側が付ける
  - この run は、依頼された範囲(report だけの commit)に合わせて、insight event を書いていない。skill の手順のとおりに残すなら `./scripts/insights-append.sh --slug org-ssh-alias-host --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 7 --cycle auto --source skill`
- Known gaps:
  - `ssh -G` は既定の ssh の設定を読み、`core.sshCommand` と `GIT_SSH_COMMAND` の `-F` は見ない(gh も同じ)。別名が `-F` の設定にだけある checkout では、解けないまま案内に進む
  - gh の `Translator` との読み方の一致(最後の `hostname` の行、解いたあとの読み替え)は、計画の Assumptions を信じて、go-gh のソースを読み直していない
  - 本物の `Match exec` を書いた ssh の設定での実走は、evidence も言うとおり、していない(実プロセスのテストは偽の ssh)
  - テスト、静的解析、spec の合致、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の範囲)
