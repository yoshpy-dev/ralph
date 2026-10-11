# Self-review report: org-cross-watch

- Date: 2026-10-11
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md(承認済み、digest d0a6f2735552)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)
- Scope: diff の品質だけ。`git diff 577ebeaf...HEAD`(HEAD b25e47da、40 ファイル、+5,594/-986)。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない

## Evidence reviewed

- 全文を読んだ差分: `internal/org/watchalive.go`、`internal/org/watch.go`(`RunWatchAll`、`watchAllRun`、`evaluateCycle`、呼び出しの打ち切り、`realGitStatus`、`escalateAlert`)、`internal/org/deadline.go`、`internal/org/escalate.go`、`internal/org/lockfile.go`、`internal/cli/org_watch.go`、`internal/cli/status.go`、`internal/config/config.go`、`scripts/ralph-config.sh`(root と `templates/base/` は一致)、`templates/base/ralph.toml`、`internal/org/spawn.go`・`verbs.go`・`watcher.go`・`inbox.go` の差分
- 移動したコード: `internal/cli/org.go` から消えた行と `org_inbox.go`・`org_watch.go` の行を突き合わせた。`org_inbox.go` で base の `org.go` にない行は、`InboxHintStateDir` への付け替え、`wireOrgInbox` の呼び出し、doc の書き直しだけ
- テストは中程度の深さで読んだ: `internal/cli/org_watch_test.go`(起動の差し替え、シグナル、`--detach`、`--stop`、`startDetachedProcess`)、`internal/org/watch_all_test.go` の補助と見出し、`internal/org/watchalive_test.go` のロックの競り合い、`internal/org/watch_test.go` の書き換えた 3 本、`internal/cli/main_test.go`
- テストの在庫: diff から消えた `func Test` 4 本のうち 3 本は `TestRunWatchAll_*` に名前を変えて残り、`TestOrgWatch_RequiresOrgID` だけが消えた(`comm` で集合の差を取った。L7)
- 機械的な確認: 追加行にデバッグ出力、TODO、secret らしい文字列はない。`chmod` で書き込みを失敗させるテストはない(root の CI でも落ちない)。新しい関数の上に別の関数の doc が残っていない。`RunWatch`・`WatchParams`・`watch --org-id` の残りは、置き換えの経緯を書いたコメントと計画・仕様・過去の report だけ。新しい tech-debt の 5 行は、どれも区切りの `|` が 6 個(5 列)
- 確かめていないもの: 切り離した子の osascript の通知が実機で出るか(計画の Open question)。agmsg の Join に空の作業ディレクトリを渡したときの動き(L5)。どのテストも走らせていない

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | exception-handling | 生存の印が新しいかの判定は `deadman_minutes` だけで決まり、その値の下限がない。(a) `config.Load` は `[org].deadman_minutes` を検査せず、0 も負の値も通る。デッドマンの側では 0 が「時間切れでは escalate しない」として働く(`> 0` の判定)。一方 `ProbeWatch` は 1 未満を「どの印も stale」と扱う。そのため `deadman_minutes = 0` の台帳では `--detach` が健全な子を起動し、10 秒待ち、子に SIGTERM を送り、「not running within 10s, so it was stopped」で終了コード 1 になる。子は正常に動いていたので、この診断は誤り。`ralph status` もずっと `stale ... older than deadman_minutes` と出す。次の PR の spawn と TASK の検査は同じ関数を使うので、この設定ではすべて断られる。(b) 印には `interval_seconds` が入るが、どこからも読まれない。`interval_seconds` が `deadman_minutes` の分より長い設定では、設定ファイルが 1 つでも、健全な見張りがサイクルの間で `stale` になる。tech-debt に足した行(`ralph status` と `--config` の食い違い)は、2 つの設定がずれる場合しか書いていない | `internal/org/watchalive.go:163-165`(doc)、`:172`(`deadmanMinutes < 1` で stale)。`internal/cli/org_watch.go:310`(`deadman := rt.Config.DeadmanMinutes`)、`:327`(running を待つ)、`:341-346`(子を止めて誤った理由を返す)。`internal/cli/status.go:156`。`internal/config/config.go:333-393`(隣の値はすべて `>= 1` を検査するが、`deadman_minutes` は検査しない)。`internal/org/watch.go:1271`(`if w.cfg.DeadmanMinutes > 0`)。`internal/org/watchalive.go:70` と `internal/org/watch.go:578`(`IntervalSeconds` を書くだけで、grep で読み手がない)。`docs/tech-debt/README.md:201` | 次のどちらかにする。`config.Load` で `deadman_minutes >= 1` を検査し、`checkDeadman` の `> 0` の分岐を外す。または、生存を判定する呼び出し側(`runOrgWatchDetach`、`statusDeadmanMinutes`、次の PR の検査)で 1 未満を既定の 10 に置き換える。(b) は `ProbeWatch` の窓を `max(deadman, 2 × mark.IntervalSeconds)` にするか、`interval_seconds` と `deadman_minutes` の組を検査する。どちらでも `deadman_minutes = 0` での `--detach` のテストを 1 本足す。直さないなら tech-debt の 201 行に (a) と、1 つの設定でも起きる (b) を足す |
| L1 | LOW | maintainability | `--stop` の頼みで止まった見張りは、`RunWatchAll` が ctx を取り消さずに返るので、`watcherWG.Wait()` が on-demand watcher の goroutine の終わりを待つ。印は `stopped`、ロックは放してあり、`--stop` は「stopped」と言って終了コード 0 で返る。それでもプロセスは `watcherInvokeTimeout`(60 秒)まで残る。watcher の判定のあとの `SendWatchdogAlert(ctx, ...)` には `callWithTimeout` がないので、agmsg が返らなければ残る時間に上限がない。scope_change は作業中の座席で何度も立つので、判定が走っている最中に止める場面は珍しくない。`runOrgWatchAll` のコメントは「long-running mode では ctx で区切られる」と書くが、頼みのファイルで止まる経路はこれに当たらない。計画の Rollout(`--stop` のあと `not_running` を確かめてからバイナリを入れ替える)のときも、古いプロセスが残る | `internal/cli/org_watch.go:178-188`(コメントと `watcherWG.Wait()`)、`:492`(`SendWatchdogAlert(ctx, ...)`)。`internal/org/watch.go:526` と `:672`(頼みのファイルで `true` を返すが、ctx は生きたまま)。`internal/org/watcher.go:74`(60 秒)、`:278` | hooks に渡す ctx を `context.WithCancel` で包み、`RunWatchAll` が `--once` 以外で返ったら取り消す(`--once` は今どおり待つ)。watcher の ALERT の送信も `callWithTimeout` 相当の上限で包む。直さないならコメントを「`--stop` で止まった場合は最大 60 秒(と agmsg の送信)待つ」に直す |
| L2 | LOW | exception-handling | ループの ctx に残る呼び出しは、デッドマンのデスクトップ通知だけになった。サイクルの途中で SIGINT か SIGTERM が届くと(操作者の `kill <pid>`。エラーの案内が勧める手順)、`escalateAlert` は `escalations.jsonl` の行を書き、alert を escalate 済みにして `PendingAlerts` から消す。そのうえで、取り消し済みの ctx から作った ctx で osascript を呼ぶので、通知は出ないか即座に止まる。その alert の通知は二度と出ない。切り離した見張りでは banner が `watch.log` に入るので、人に届くのは osascript だけになる。受信箱の期限の通知は `context.Background()` から作るので、同じシグナルでも届く。2 つの経路で扱いが違う | `internal/org/watch.go:1312`(`context.WithTimeout(ctx, ...)`)、`:717-719`(「ctx reaches only the deadman's EscalateFunc」)。`internal/org/escalate.go:379`(受信箱は Background から)。`TestWatch_Deadman_EscalateFuncGetsItsOwnDeadline`(`internal/org/watch_test.go:776`)が、cycle の ctx から作ることを意図として固定している | `evaluateCycle` のほかの呼び出しと同じく、`context.Background()` から `inboxDesktopNotifyTimeout` で作る。テストは「親の値を引き継ぐ」の確認を「ループの ctx が取り消されていても期限つきで呼ばれる」に替える |
| L3 | LOW | exception-handling | org の状態ファイルの扱いが変わった。読めない状態ファイルの org は、毎サイクル飛ばされ続ける(印は書かれ、`ralph status` は `org watch: running`)。1 org の watch のときは、起動の時点でエラーで止まって人の目に入った。今は `watch.log` に 30 秒ごとに 1 行出るだけで、その org の stall・生存・デッドマンの見張りは黙って止まる。`ralph status` の org ごとの行は、壊れたファイルを「まだ watch されていない」と同じに見せる。一方、書く側の `save` は `os.WriteFile`(切り詰めてから書く)のままで、rename を使わない。kill やディスクの満杯で途中まで書かれたファイルは、次の起動で読めない。同じ PR がこの package に `writeJSONFileAtomic` を足しているのに、使うのは印と止める頼みだけ | `internal/org/watch.go:287-299`(`save`)、`:627-630`(読めなければ `return err`)、`:611`(毎サイクルの 1 行)。`internal/cli/status.go:279-287`(parse の失敗を「ファイルなし」と同じに扱う)。`internal/org/watchalive.go` の `writeJSONFileAtomic` | `save` を `writeJSONFileAtomic` で書く。読めないファイルは、注意を 1 回だけ出して新しい状態から始める(重複の抑止を失うだけで済む)か、`ralph status` の org の行に「読めない」と出す |
| L4 | LOW | maintainability | 文字列が 4 つ並ぶ引数がまた増えた。tech-debt の (g) は「文字列が並ぶと、取り違えても通る」を理由に `hintStateDir` を外して閉じた。一方、同じ PR が `reason` を足し、`sendInboxItemToHuman(orgID, id, typ, reason)` が `inboxHumanTextFor(reason, orgID, id, typ)` を呼ぶ。隣り合う 2 つの関数で、同じ 4 つの文字列の順番が違う。banner のテストが多くの取り違えを拾うので、今の害は小さい | `internal/org/escalate.go:291`、`:298`、`:319`。`internal/org/deadline.go:221`。`docs/tech-debt/README.md:199` の (g) の閉じた記述 | 3 つの関数を同じ順番(`orgID, id, typ, reason`)に揃える。小さな struct(`inboxSend{orgID, id, typ, reason}`)にしてもよい |
| L5 | LOW | security | 切り離した子は、`--detach` を打ったプロセスの作業ディレクトリと環境をそのまま、何日も持ち続ける。(a) `/org` skill は leader に自分で起動させるので、機能の worktree から打つことがある。その worktree が `/pr` や片付けで消えると、見張りの作業ディレクトリが消えたまま動く。`ensureWatchdogJoined` の `os.Getwd()` は失敗し、エラーを捨てて空の作業ディレクトリで Join する。on-demand watcher の `claude -p` も消えたディレクトリで動く(どちらも実際の動きは未確認)。(b) leader の pane から打てば `HERDR_PANE_ID` などもそのまま入る。今の watch はこれらを読まないので、害は今のところない | `internal/cli/org_watch.go:229-231`(「The child inherits the working directory」)、`:275-277`(`exec.Command` に `Dir` と `Env` の指定がない)。`internal/org/watch.go:964`(`cwd, _ := os.Getwd()`) | 子の `Dir` を台帳の main worktree の root(`org.MainWorktreeRoot` か `LedgerMainWorktreeRoot`)に固定し、`--config` を絶対パスにしてから渡す。`HERDR_*` は子の環境から外す。直さないなら tech-debt に 1 行足し、次の PR で start が起動するときの trigger にする |
| L6 | LOW | maintainability | `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` は `watch.log` の mode が厳密に `0644` であることを確かめる。`os.OpenFile(..., 0o644)` は umask を受けるので、umask が 027 や 077 の環境ではこのテストが落ちる(flaky ではなく、毎回落ちる)。印のほうは `writeJSONFileAtomic` が `Chmod(0o644)` で mode を決めるので、umask に左右されない | `internal/cli/org_watch_test.go:634`。`internal/cli/org_watch.go:264` | テストは「`0o644` を超える bit がない(`perm&^0o644 == 0`)」と「所有者が読み書きできる」の確認に替える。mode を固定したいなら、`startDetachedProcess` で開いたあとに `Chmod` する |
| L7 | LOW | readability | 計画の Progress の S3b の行は「消したテストはない」と書くが、`TestOrgWatch_RequiresOrgID`(`internal/cli/org_test.go`)は 33f6383b で消えた。`--org-id` を必須とする挙動が廃止されたので、テストを消したことは正しい。逆の挙動(`--org-id` を断る)は `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` が固定している。記録だけが実態と合わない(testing.md は、意図を変えたテストを計画と report に書くことを求める) | `docs/plans/active/2026-10-11-org-cross-watch.md:205`。`git log -S'func TestOrgWatch_RequiresOrgID' 577ebeaf..HEAD` は 33f6383b を返す | S3b の行を「1 org の `RunWatch` のテスト 3 本は `RunWatchAll` に書き換え、`TestOrgWatch_RequiresOrgID` は挙動の廃止に合わせて消し、拒否は `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` が持つ」に直す |

## Positive notes

- `--stop` はシグナルを送らず、台帳の頼みのファイルで見張りに止まってもらう。印の pid を使い回されたプロセスに送る危険を、設計の段階で消している。`TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal` は、印が生きている別のプロセス(`sleep 30`)を指すときにそのプロセスが残ることまで確かめる
- `startDetachedProcess` は子を回収するまで自分の子として持つので、`stop` の SIGTERM と SIGKILL が別のプロセスに当たらない。`--detach` の失敗の経路は、自分が起動した子を止めて回収し、ロックが放されたかを見てから返る
- `started_at` をロックを待つ前に決め、古い頼みだけを起動のときに消す。こうしたので、ロックを取ってから印を書くまでの間に書かれた `--stop` も失われない。時刻をナノ秒で持つ理由も doc に書いてある
- パルスの呼び出しを 1 回ごとに Background から作る期限で打ち切る。こうすると、SIGINT がサイクルの途中で来ても、生きている座席の liveness ALERT が偽って出ない。この利点と、止まるまでが延びる代償の両方を tech-debt に書いている
- manifest から読んだ org_id を `ValidateIdentifier` に通してから状態ファイルの名前にする。手で書き換えた行でパスを作れない
- テストは test binary を exec しない。`orgWatchLaunch` を TestMain で panic する偽物にし、`--detach` は同じプロセスの中で動く見張りで、`startDetachedProcess` はシェルスクリプトで確かめる。flock が開いたファイルに結びつくことを使い、同じプロセスの中でもロックの競り合いを再現している
- 移動したコード(`org_inbox.go`、`org_watch.go`)は、付け替えの行を除いて base と同じ行で、関係のない変更が混ざっていない

## Coverage gaps

- 実機の確認はしていない: 切り離した子の osascript の表示、herdr が止まっているときの 1 サイクルの長さ、Linux での setsid
- L5 の「作業ディレクトリが消えたあとの Join と `claude -p`」の実際の動きは読んでいない(agmsg の script は repo の外にある)
- `watch.log` は追記だけで、切り詰めない。L3 の 1 行のような毎サイクルの失敗が続くと、1 つの条件で 1 日 2,880 行ほど増える。finding にはしていない。見張りを何日も動かす運用(ユーザーの決定)では、次の PR で start が起動するときに扱いを決めるとよい

## Tech debt identified

- M1 と L1〜L6 を直さない場合は、tech-debt に 1 行にまとめて足す。M1 は既存の 201 行(`ralph status` と `--config` の食い違い)に足すのが自然
- L7 は計画の記録の直しだけで、tech-debt には要らない

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件(M1、`deadman_minutes` が 1 未満の設定で `--detach` が健全な子を止めて誤った理由で失敗する)、LOW 7 件)
- Follow-ups: (1) M1 は `/verify` に進む前に直すのが安い。config の検査を 1 つ足すか、呼び出し側で既定に置き換え、テストを 1 本足す。(2) L1・L2・L3 はどれも数行の直しで、見張りが人に届ける経路の取りこぼしを減らす。直すなら M1 と同じコミットでよい。(3) L4・L6 は機械的な直し。L5 は次の PR(start が見張りを起動する)の前に方針を決める。(4) L7 は計画の 1 行の直しで、記録だけ。`.go` を触れば diff が変わるので、`/self-review` から回し直す(cross-review の前なので cycle は増えない)
