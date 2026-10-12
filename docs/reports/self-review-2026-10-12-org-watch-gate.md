# Self-review report: org-watch-gate

- Date: 2026-10-12
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み、digest de5ca6b7e6f0)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)の中の再実行。最初の結果(HEAD 976a0ca9)は末尾の付録に残してある
- Scope: 最初の結果の L1〜L4 を直した 2 つのコミット、dbeb45ca(L1〜L4)と 7d2e4bbb(CLI の send の二重の接頭辞。L3 を調べたときに見つかった、この PR より前からある包み)。`git diff f7a6fe6c..7d2e4bbb`(10 ファイル、+231/-73)。L5・L6 は /sync-docs に回すので、直したかは見ていない。テスト・静的解析・仕様適合・文書の整合の検査は行っていない
- 番号の付け方: 今回の finding は N1〜N3。L1〜L6 は付録の最初の結果の番号

## Latest run: 再実行(HEAD 7d2e4bbb)

### 最初の結果の指摘の扱い

| 対象 | 状態 | 根拠 |
| --- | --- | --- |
| L1(子が終わったあとのエラーが相手の見張りを言わない) | 直った | `launchAndAwaitOrgWatch` は子が終わったあと、最後の probe が `not_running` でなければ `e.refuse(probe)` を返す(`internal/cli/org_watch.go:486-496`)。その probe は子が終わってから取ったものである。select で `childExited` が立ったあと、ループの頭でもう一度 probe するので、子がまだロックを持っていた時点の probe は使われない。抜ける条件は前と同じで、子が終わったあとは stale の間だけ期限まで待つ。stopping と `--once` はすぐ、stale は期限で拒否の文になり、ロックが空いていれば今までどおり watch.log を名指す(`TestOrgWatchDetach_ChildExitsFirst_ExitsOneNamingTheLog` がこちらの枝を通る)。新しいテスト 2 本(`TestOrgStart_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefusedAsTheGateRefuses` は start の 2 つの形 × 3 状態、`TestOrgWatchDetach_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefused` は `--detach` × 3 状態)は、直す前のコードでは「exited before it was running」になって落ちる。条件を stopping だけにした場合も、`--once` と stale の行で落ちる |
| L2(not_running の文が既存の org に start を案内する) | 直った | 文を「or, for a new org, start it with ralph org start, which starts one」に絞り、理由をコメントに書いた(`internal/org/watchgate.go:52-56`)。`watchgate_test.go` が新しい言い回しをそのまま固定する。古い言い回しは、コード・skill・雛形のどこにも残っていない(grep) |
| L3(send の拒否の `org:` の重なり、何もしない `TrimSpace`) | 直った | `taskNotSentError`(`internal/org/verbs.go:225-231`)が `org: send: TASK not sent: ` に、`org: ` を外した検査のエラーをつなぐ。`Unwrap` で検査のエラーを返すので、`errors.Is(err, ErrNoRunningWatch)` も、`errors.As` で `*WatchGateError` を取り出すことも、今までどおりできる。`TrimSpace` を消し、doc は「Parse trims the spaces around the key and the value and keeps their case」と直した。`splitHeaderLine` は key と値の両方を `TrimSpace` するので、doc は正しい(`protocol.go:140`、`:144`)。org と CLI のテストは、文の頭と `org:` が 1 回だけであることを見る |
| L3 の続き(7d2e4bbb、CLI の `org: send: org: send:`) | 直った | CLI は、`org: send: ` で始まるエラーをそのまま返し、ほかだけを包む(`internal/cli/org.go:953-958`)。`Send` の戻り口をすべて読んだ。`org: send: ` で始まらないのは dry-run の記録の追記(`verbs.go:318`)だけで、コメントのとおりである。変わるのは、二重だった文から 1 つ目の `org: send: ` が消えることだけで、ほかの文は変わらない。包みの `%w` は残るので `errors.Is` の鎖も同じ。古い二重の文をテストや文書が前提にしている箇所はない(`grep -rn 'org: send: org:'` は新しいテストの 1 行と過去の報告だけ) |
| L4(a)(b)(c)(d) | 直った | (a)(b) `WatchPIDText` と `WatchKillHint` を org 層に置き(`watchgate.go:83`、`:95`)、検査の文、`--detach` の拒否、`--stop` の 3 か所が同じ helper を呼ぶ。検査の stale の文は CLI の言い回し(「kill N (the pid in …watch-alive.json; check that it is still a ralph org watch first)」)にそろった。(c) `orgWatchGateOverride` は `func() error` になった。(d) `orgWatchEnsureDeadline` に改名し、コードとテストに古い名前は残っていない(grep。過去の報告の 1 行だけ) |
| L5・L6 | 対象外(/sync-docs) | この 2 つのコミットは文書に触れていない。どちらも開いたまま |

### Evidence reviewed

- 全文を読んだ差分: dbeb45ca と 7d2e4bbb の全ファイル(`internal/org/watchgate.go`、`verbs.go`、`internal/cli/org_watch.go`、`org.go`、`main_test.go`、テスト 5 ファイル)
- 周りで読んだ今のコード: `launchAndAwaitOrgWatch` と `ensureOrgWatch` の全体、`orgWatchDetachRefusal`、`runOrgWatchStop`、`Send` のすべての戻り口(`verbs.go:290-446`)、send の CLI の失敗の経路(`org.go:902-960`)、`splitHeaderLine`
- 機械的な確認: `grep -rn 'orgWatchDetachDeadline\|watchGatePIDText\|orgWatchPIDText\|orgWatchKillHint\|check that pid .* is a ralph org watch and kill\|start the org with ralph org start'` は過去の報告とこの報告にしか当たらない。`rt.Send(` を呼ぶのは send のコマンドだけ。追加行にデバッグ出力・TODO・secret らしい文字列はない
- 確かめていないもの: テストは走らせていない。新しいテストが直す前のコードで落ちることは、読んで確かめた

### Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | typo | dbeb45ca で新しいテストを上に足したときに、すぐ下の既存のコメントのコロンの後ろの空白が消えた(「AC13 and AC14:a watch that holds watch.lock」) | `internal/cli/org_start_watch_test.go:341` | 「AC13 and AC14: a watch」に戻す |
| N2 | LOW | readability | L1 の直しで、`--detach` の拒否の文(`orgWatchDetachRefusal`)は子を起動したあとにも出るようになった。文は 3 通りとも「so nothing was started」と言い、関数の doc も同じ前提で書かれている。実際には子を起動していて、その子はロックを断られて終わり、watch.log にその子の拒否の行が残る。動いている見張りを増やしていない点は正しいので害は小さいが、watch.log を開いた人は「何も起動していない」と食い違う行を見る。start の側(`WatchGateError`)はこの言い方をしないので、`--detach` だけの話である | `internal/cli/org_watch.go:535-550`(doc と 2 つの文)、`:521`(`refuse` に渡す)、`:486-496`(起動のあとの呼び出し) | 「so no watch was started」のように動いている見張りの話にそろえるか、起動のあとの呼び出しのときだけ「the watch it launched (pid N) was refused the lock」を足す。直さないなら関数の doc に「起動した子が断られたときも同じ文」と書く |
| N3 | LOW | maintainability | 7d2e4bbb は CLI でエラーの文の頭を見て包むかを決める。「`Send` のエラーは dry-run の記録の追記を除いて `org: send: ` で始まる」という決まりが、org 層の戻り口と CLI のコメントの 2 か所に分かれた。新しい戻り口が接頭辞なしで足されても CLI が包むので壊れはしないが、コメントの「only the others (the dry-run event's append)」は古くなる。dry-run の 1 か所が接頭辞を持てば、CLI の包みと文字列の比較はいらない | `internal/cli/org.go:953-958`、`internal/org/verbs.go:318`(`return SendResult{Err: err}`) | `verbs.go:318` を `fmt.Errorf("org: send: record the dry-run event: %w", err)` のように包み、CLI は `result.Err` をそのまま返す。`TestOrgSend_AnErrorWithoutTheSendPrefixGetsItOnce` の期待もそれに合わせる。今の形のままでも挙動に問題はないので、次に send のエラーを触るときでよい |

件数: CRITICAL 0、HIGH 0、MEDIUM 0、新しい LOW 3 件(N1〜N3)。最初の結果から開いているのは L5・L6 の 2 件(/sync-docs に回す)で、開いている LOW は合わせて 5 件。

### 依頼された点の結論

1. L1 の `e.refuse(probe)` の分岐: 正しい。分岐に入るのは子が終わったあとだけで、そのとき使う probe は子が終わってから取ったものである。ロックを持っている見張りが stopping か `--once` ならすぐ、stale なら期限で拒否の文になる。ロックが空いていれば今までどおり watch.log を名指す。start では `WatchGateError` が返るので、`StartFeature` は worktree を作らず、新しいテストがそれを start の 2 つの形で確かめる。気になるのは `--detach` の文の「nothing was started」だけ(N2)
2. 子が終わったあとの stale の待ち: 前と同じで、期限まで待つ。期限の前に stale から not_running に変われば抜けて watch.log の文、running に変われば成功、stopping か `--once` に変われば拒否になる。待ちが延びる経路は増えていない
3. L4 の `WatchKillHint` で文をそろえたこと: 検査の stale の文は「stop it with ralph org watch --stop, or kill 4302 (the pid in …/watch-alive.json; check that it is still a ralph org watch first)」になり、続けて読める。pid がないときは「find the process that holds …/watch.lock (…/watch-alive.json names no pid) and kill it」。leader の雛形は文の「no new work」と `ralph org watch --all` で拒否を見分けるので、この変更の影響を受けない。skill と README は kill の文を引いていない
4. 7d2e4bbb がほかの send のエラーを変えるか: 変えるのは、二重だった `org: send: org: send: …` が `org: send: …` になることだけである。dry-run の記録の追記の失敗は今までどおり `org: send: ` が付き(`org: send: org: open …`)、新しいテストがそれを固定する。読み直しの案内(note)は包みの前に出すので、出方は変わらない

### Positive notes

- L1 の直しは、最初の probe のときと同じ `e.refuse` を通るので、start と `--detach` のどちらでも、出会った時点にかかわらず同じ文になる。新しいテストは start の 2 つの形と `--detach` の両方で 3 状態を並べ、「exited before it was running」が出ないことも見る
- `taskNotSentError` は `Unwrap` を持つので、文を短くしても `errors.Is` と `errors.As` が今までどおり通る
- 7d2e4bbb は、この PR より前からある二重の接頭辞を、`errors.Is` の鎖を保ったまま直し、接頭辞のない 1 つの経路をテストで固定した

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しい行はない。

### Coverage gaps

- テストは走らせていない(/test の担当)。新しいテストが直す前のコードで落ちることは、コードを読んで確かめた
- `TestOrgSend_AnErrorWithoutTheSendPrefixGetsItOnce` は manifest の場所にディレクトリを置き、`errors.Is(err, syscall.EISDIR)` を見る。macOS と Linux では、ディレクトリを書き込みで開くと EISDIR になる。Linux では読んだだけで確かめていない
- 最初の結果の Coverage gaps(`CLAUDECODE` の受け継ぎ、確かめられないロックのときの stale の文、前の版の見張りとの共有、Linux)は、この 2 つのコミットで変わっていない

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。L1〜L4 は直り、L1 と L3 には直す前のコードで落ちるテストがある。新しい LOW は N1〜N3 の 3 件で、どれも挙動の誤りではない。L5・L6 は /sync-docs で直す)
- Follow-ups: N1 はコメントの 1 文字、N2 は文か doc の 1 行、N3 は任意の整理。直すと /self-review からやり直しになる。L5・L6 は /sync-docs で直す

## 付録: 最初の結果(HEAD 976a0ca9)

- 最初の結果の Scope: diff の品質だけ。`git diff dc41c4f5...HEAD`(HEAD 976a0ca9、33 ファイル、+3373/-319)。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない。計画の Design decisions F1〜F10 と Progress checklist に記録された決定(検査を `checkSpawnInput` の直後に置くこと、それより後の検査で断られた start が起動した見張りを残すこと、Dir を変えたときの `PWD` の書き換え、期待を変えた 6a のテスト)は、コードが記録と合っているかだけを見た

### Evidence reviewed(最初の結果)

- 全文を読んだ差分: `internal/org/watchgate.go`、`watchalive.go`、`watch.go`、`spawn.go`、`feature.go`、`verbs.go`、`internal/cli/org.go`、`org_watch.go`、`status.go`、`main_test.go`、`internal/org/prompts/leader.md`。テストは `watchgate_test.go`、`watchgate_positions_test.go`、`watchalive_test.go`、`watch_all_test.go`、`prompts_test.go`、`org_watch_gate_test.go`、`org_start_watch_test.go`、`org_watch_test.go` の差分を読み、`status_test.go` は足したテストの名前とコメントを読んだ
- 文書: `/org` skill の 4 面(`cmp` で 4 つとも同じ)、agent-messaging の 2 面(同じ)、quality-gates の 2 面(同じ行を足した)、README、AGENTS.md、仕様の注、tech-debt の差分
- 周りで読んだ今のコード: `Parse`・`splitHeaderLine`・`Validate`・`ValidateText`(`internal/org/protocol/protocol.go`)、`ResolveOrgStateDir`・`MainWorktreeRoot`・`LedgerMainWorktreeRoot`(`internal/org/statedir.go`)、`fileLockHeld`(`internal/org/lockfile.go`)、`runOrgWatchStop`・`waitOrgWatchReleased`・`orgWatchKillHint`(`internal/cli/org_watch.go`)、`StartFeature` の前半、`writeMark`・`finish`・`stopAndRelease`(`internal/org/watch.go`)、start の RunE と `runOrgStartPlan`
- 機械的な確認: 追加行にデバッグ出力・TODO・secret らしい文字列はない。tech-debt で足した行と書き換えた行の 5 本は、どれもエスケープしていない `|` が 6 個で、列は崩れていない。RESOLVED の注が挙げるテスト 13 本は、どれも HEAD に 1 つずつある。`orgWatchGateOverride` を使うのは TestMain と `useRealOrgWatchGate` だけで、`internal/cli` のテストに `t.Parallel` はない(package 変数の差し替えと並行しない)。消した `newOrgSpawnRuntime` を呼ぶコードは残っていない
- 確かめていないもの: テストもバイナリも走らせていない(「Coverage gaps」)

### 依頼された点の結論(最初の結果)

1. `NewWatchGate` の settle の繰り返し(`watchgate.go:107-127`): 判定し直すのは stale のときだけで、ほかの状態はすぐ返す。期限は実時計、印の判定は `now()` で、待ちは `min(watchGateProbeInterval, left)` なので期限を越えない。閉包が共有する可変の状態はない。`WatchGateError` の文は 5 通りとも `ralph org watch --all` と `no new work` を含み、leader の雛形のテストがそれを全状態で確かめている。running の枝は `--once` を前提に書かれているが、`ProbeWatch` が running を返すときは印が nil でないので、`WatchAcceptsWork` が false になる running は `Once` だけで、前提は成り立つ
2. `ProbeWatch` の stopping と印の deadman(`watchalive.go:206-226`): stopping は running と同じ窓で判定し、窓の外は stale になる。印の `deadman_minutes` は nil でなければ呼んだ側の値より先に使い、1 未満を 10 に読み替える規則は変わらない。前の版の印(欄なし)は呼んだ側の値で読む
3. `drainBeforeStop`(`watch.go:782-806`): stopping を書くのは書き直しの goroutine を始める前で、goroutine は `TS` だけを書く。元の goroutine は `close(done)` と `<-refresherDone` のあとで stopped を書くので、データ競合はない。BeforeStop がないときは stopping を書かない(doc のとおり。CLI はいつも BeforeStop を設定する、`internal/cli/org_watch.go:201`)。`OnLocked` は `defer w.finish(lock)` のあとに呼ぶので、`OnLocked` が panic してもロックは放される
4. 検査の位置: `Spawn` は `checkSpawnInput` の直後で、manifest の読み、dry-run の枝、`withManifestLock` の前(`spawn.go:464-472`)。`StartFeature` は `checkSpawnInput` の直後で、`Manifest.Read`・`spawnPrecheckErr`・worktree の前(`feature.go:194-200`)。`Send` は検証の直後で、dry-run の追記、`findSeat`、ctx を作る前(`verbs.go:277-291`)。どれも断ったときに何も書かない。ctx を作る前なので、検査の最大 2 秒は `--timeout-ms` を使わない。`Send` の `Parse` と `Validate` への分け方は `ValidateText`(`protocol.go:174-180`)と同じ順で、型付きのメッセージの扱いは変わらない。raw の TYPE は `Parse` が値を `TrimSpace` する(`protocol.go:144`)ので、空白と大文字小文字の扱いは計画 F5 のとおり
5. CLI: `ensureOrgWatch` と `launchAndAwaitOrgWatch` は計画の図 2 と tech-debt 206 の直しのとおりに動く。子が終わったら `exited` を nil にして select から外す書き方は正しく、そのあとは stale の間だけ待つ。`orgWatchChild` は環境から `HERDR_*` の 4 つを外し、Dir を変えたときだけ `PWD` を差し替える。`--config` は `filepath.Abs` で絶対パスにする。state dir は `ResolveOrgStateDir` が絶対パスで返す(`mustAbs`)ので、子の作業ディレクトリが変わっても `--state-dir` はずれない。banner は `OnLocked` から出すので、断られた 2 つ目の見張りは出さない。`wireOrgStartWatchGate` は ensure のあとで本物の検査を呼び、TestMain の差し替えがあるときは ensure を通らない。start に `--dry-run` はないので、dry-run が見張りを起動することはない
6. 子の起動の安全性: shell を通さず `os.Executable()` を exec し、引数は固定の語と絶対パスだけである。ロックの fd は Go の既定で close-on-exec なので子に渡らない。ログは 0644 の追記で 6a から変わらない
7. `status.go`: stopping と `(--once)` の行、`--json` の `once`、kill された見張りが残した stopping の印の行を足した。`statusDeadmanMinutes` の doc は、印に値がないときだけ使うと直してある
8. 雛形と文書: leader.md の項目は `TestRenderRolePrompt_Leader_WatchRefusalEscalatesWithoutRetry` が語句と `WatchGateError` の文の両方から固定している

### Findings(最初の結果)

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L1 | LOW | exception-handling | 起動した子が、先にロックを取った stopping か `--once` の見張り(または stale のままの見張り)に断られて終わると、start と `--detach` のエラーは「the watch (pid N) exited before it was running; see watch.log」になる。「その見張りが終わってから打ち直す」という次の手は出ない。最初の probe で同じ状態を見たときは `e.refuse(probe)` の文が出るので、相手の見張りにどの時点で出会ったかで答えが変わる。起こるのは、cron の `watch --all --once` が始まった直後に start を打ったとき(子はロックを 2 秒待ってから終わる)など。watch.log には相手の pid を含む拒否の 1 行が残るので、調べれば原因は分かる | `internal/cli/org_watch.go:472`(`childExited && probe.State != org.WatchLivenessStale` で抜ける)、`:482-483`(childExited の文)。最初の probe の扱いは `:439-441` | 子が終わったあとで、ほかの見張りがロックを持っている(`probe.State` が `not_running` でない)なら `e.refuse(probe)` を返す。テストは `TestOrgStart_AChildThatExitsWhileAnotherWatchRuns_StartGoesOn` と同じ作りで、launcher の中で `onceWatchMark` を書いてロックを取り、閉じた `exited` の子を返せば足せる |
| L2 | LOW | readability | 見張りがない(not_running)ときの文は、spawn にも TASK の send にも「or start the org with ralph org start, which starts one」と案内する。TASK を送る相手の座席は動いている org の中にあり、(A) 現セッション昇格の org には leader の座席がない。その org の人が案内どおり `ralph org start` を打つと、headless の leader の座席が新しく立ちうる。文の「start the org」は新しい org の話と読めるが、すでにある org への spawn と TASK には当てはまらない | `internal/org/watchgate.go:51-53`。`prompts_test.go` は not_running の文が `ralph org watch --all` と `no new work` を含むかだけを見るので、語を足しても雛形のテストは崩れない | 「or, for a new org, start it with ralph org start, which starts one」のように新しい org に限る。`WatchGateError` は呼び手を知らないので、文を絞るほうが手数は少ない |
| L3 | LOW | readability | send の拒否は `org: send: TASK not sent: org: no ralph org watch --all is running ...` と `org:` が 2 回出る。S3 は起動のエラーで内側の `org: watch --detach:` を外して二重にならないようにしたが、send は `WatchGateError` の頭の `org: ` にそのまま重ねている。また `verbs.go:287` の `strings.TrimSpace(msg.Type)` は、`Parse` が値をすでに `TrimSpace` している(`protocol.go:144`)ので何もしない。doc と計画の「空白を除いて」と並べて読むと、`Parse` は除かないと受け取りうる | `internal/org/verbs.go:287`、`:289`。`internal/cli/org_watch_gate_test.go:324` と `internal/org/watchgate_positions_test.go:204` が `org: send: TASK not sent: ` を部分一致で見る | 包みの文を `org: send: TASK not sent` から `org:` の重ならない形にするか、`WatchGateError` の文の頭の `org: ` を外す(spawn と start の `rejected: ` の行も短くなる)。`TrimSpace` は消すか、`Parse` が除くことを 1 行のコメントで書く |
| L4 | LOW | maintainability | 同じものが 2 か所にある。(a) `watchGatePIDText`(`internal/org/watchgate.go:83-88`)は `orgWatchPIDText`(`internal/cli/org_watch.go:592-597`)と本体が同じ。(b) stale の見張りを kill する案内が、検査の文(`watchgate.go:55-58`、「check that pid N is a ralph org watch and kill it」)と、`--detach`・`--stop` の `orgWatchKillHint`(`org_watch.go:603-609`、「kill N (the pid in watch-alive.json; check that it is still a ralph org watch first)」)の 2 通りで書かれている。(c) `orgWatchGateOverride` の引数(`stateDir`, `deadmanMinutes`)はどの実装も読まない(`internal/cli/org.go:268`、`:327`、`main_test.go:69`)。(d) `orgWatchDetachDeadline` は start の待ちにも使うようになり、doc は直したが名前は `--detach` のまま | 左の各行 | (a)(b) は org 層に exported の helper を 1 つずつ置き、cli から呼ぶ。(c) は引数のない型にするか、引数を読むテストを足す。(d) は改名するなら `orgWatchEnsureDeadline`。どれも挙動は変えないので、次にこのあたりを触るときでよい |
| L5 | LOW | maintainability | tech-debt の開いた行 2 本がこの PR で古くなり、閉じた行 1 本に「still open」が残る。(a) 183 行の (c) F-7 は「`newOrgSpawnRuntime` (`internal/cli/org.go`) copies the first two steps of `newOrgRuntime`」と書くが、この PR は `newOrgSpawnRuntime` を消した。写しは、spawn・`start <task>`・`start --plan` の 3 か所に並んだ `ResolveOrgStateDir` の呼び出し(`internal/cli/org.go:550`、`:726`、`:823`)と、`newOrgSpawnRuntimeAt` の guard に移っている。(b) 205 行の (b) は `newOrgSpawnRuntimeAt` の呼び手を「(spawn and `start --plan`)」と書くが、今は `start <task>` も直接呼ぶ。(c) 202 行(201 として閉じた行)の Impact の欄に「still open: a cycle that takes longer than the deadman of `./ralph.toml` but not the watch's own」が残る。印が見張り自身の deadman を持つようになったので、この残りもこの PR で閉じた | `docs/tech-debt/README.md:183`、`:205`、`:202` | (a) は F-7 の文を今の形(3 か所の解決と `newOrgSpawnRuntimeAt`)に書き直す。Trigger の「`newOrgRuntime` の次の変更」はこの PR で起きていないので、行は開いたままでよい。(b) は呼び手に `start <task>` を足す。(c) は「still open: ...」に取り消し線を引くか、RESOLVED の注に「この残りも閉じた」と足す。/sync-docs で直せる |
| L6 | LOW | readability | README の「Either way the watch runs detached in the ledger's main worktree root, so without `--config` it reads that root's `ralph.toml`.」は、台帳に main worktree がないとき(repo の外を指す `--state-dir`)には打った場所で動く、という条件を落としている。start と `watch --detach` の help は「when it has one」、skill は「求められなければ打った場所」と書く | `README.md:256`。help は `internal/cli/org.go:693-694`、`internal/cli/org_watch.go:63` | 「in the ledger's main worktree root (when the ledger has one)」のように条件を足す |

件数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6。

### Positive notes(最初の結果)

- `Org.WatchGate` を「nil なら検査しない」差し込み口にしたので、org 層の既存のテストは書き換えずに済んでいる。検査の位置は、stub の呼び出し回数と記録の snapshot で固定してある(`watchgate_positions_test.go`)。入力の検査より後で manifest の読みより前、という位置は、読めない manifest のときに検査の拒否が先に返ることでも確かめている
- 子の `PWD` を Dir に合わせる理由(Env を明示すると os/exec が `PWD` を直さない)をコメントに書き、`/bin/sh` の子で `pwd -P` と環境を確かめるテストを足している
- 起動のエラーから内側の `org: watch --detach:` を外し、呼び手の `errPrefix` で包む形にしたので、start と `--detach` が同じ起動の経路を通っても文が二重にならない
- leader の雛形のテストは、雛形の語句に加えて、`WatchGateError` の全状態の文がその語を含むことも見る。エラーの文を言い換えて雛形と食い違えば、テストが落ちる
- `drainBeforeStop` は、stopping の書き込み、goroutine の開始、終わりの `close` と待ち、stopped の書き込みの順で、mutex なしで競合しない

### Tech debt identified(最初の結果)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しい行はない。L5 は既存の行の書き直しで、新しい負債ではない。

### Coverage gaps(最初の結果)

- テスト、静的解析、バイナリでの実機確認は行っていない(/verify と /test の担当)。L1 の筋はコードを読んで確かめただけで、再現していない
- 子の環境から外すのは `HERDR_*` の 4 つだけである(F6)。Claude Code の Bash から start を打つと、子は `CLAUDECODE` などの変数も受け継ぐ。見張りの on-demand watcher が起動する `claude -p` がその環境でどう動くかは確かめていない。6a の `--detach` から同じで、この PR で始まったことではない
- `ProbeWatch` は watch.lock を確かめられない(open が ENOENT 以外で失敗する)とき、ロックありとみなす(6a からの規則、`internal/org/watchalive.go:191-194`)。その台帳では、検査と start は stale の文(「the ralph org watch --all that holds the lock ... find the process that holds ... and kill it」)を出し、確かめられなかったことは言わない。権限の壊れた台帳でだけ起こるので finding にはしていない
- 前の版の見張り(印に `once` も `deadman_minutes` もない)とこの版の検査が台帳を共有すると、前の版の `--once` の見張りは検査を通る。計画の Risks と Rollout のとおり release のノートで扱う
- Linux での動きは読んでいない(setsid の経路は 6a から変わっていない)

### Recommendation(最初の結果)

- 最初の結果の判定: yes(CRITICAL・HIGH・MEDIUM なし。LOW 6 件。検査の位置、止まりかけと `--once` の見張り、start による見張りの起動、印の `deadman_minutes` は、計画の F1〜F10 と Progress checklist の記録どおりに実装されている)
- Follow-ups: L1〜L4 は小さなコードの直しで、直すと /self-review からやり直しになる。L5 と L6 は文書だけなので /sync-docs で直せる
