# Self-review report: org-cross-watch

- Date: 2026-10-11
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md(承認済み、digest d0a6f2735552)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)の再実行で、cross-review の前の直しなので cycle は増えない。最初の結果は b25e47da の時点のもので、末尾の「付録」に残してある
- Scope: diff の品質だけ。主に見たのは、最初の結果のあとに入った 2 コミット(`git diff 6a6976d2..HEAD`。1b64ae2b の直しと、188a640d の status の文言と計画の記録。13 ファイル、+470/-180、HEAD 188a640d)。そのうえでブランチ全体(`git diff 577ebeaf...HEAD`、41 ファイル、+6,018/-1,063)を、最初の結果が見落としたものに絞って軽く見直した。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない
- 番号の付け方: 最初の結果の M1、L1〜L7 はそのまま使う。今回足した finding は N1〜N5

## Latest run: 最初の結果の指摘の扱い(HEAD 188a640d)

| ID | 状態 | 根拠 |
| --- | --- | --- |
| M1 | 直った | (a) `ProbeWatch` は `watchAliveWindow` で窓を決め、1 未満の `deadman_minutes` を既定の 10 に置き換える(`internal/org/watchalive.go:163-200`)。`--detach`・`ralph status`・`--stop` はどれも `ProbeWatch` を通るので、呼び出し側を直す必要がなかった。(b) 窓は印の `interval_seconds` の 2 倍より短くならず、`interval_seconds` に読み手ができた。足したテストは、どれも直す前のコードでは落ちる: `TestOrgWatchDetach_DeadmanMinutesZero_StartsTheChild`(`internal/cli/org_watch_test.go:646`。前は stale のまま期限切れ)、`TestStatusCmd_OrgWatchWithDeadmanMinutesZeroJudgesByTheDefault`(`internal/cli/status_test.go:703`。前は新しい印も stale)、`TestProbeWatch_TellsRunningStaleAndNotRunningApart` の 7 行(「ちょうど 10 分」「ちょうど 2 間隔」は running で、境界も固定)。設定で 0 を拒まない判断と計画からのずれは、計画の Progress に書いてある。窓の規則を説明する文のうち 3 か所が古いまま(N1) |
| L1 | 直った | `runOrgWatchAll` は `--once` でないとき、`RunWatchAll` が返ったら `watcherCtx` を取り消す(`internal/cli/org_watch.go:165-166`、`:192-194`)。watcher の ALERT は `SendWatchdogAlertBounded` で `DriverCallTimeout` に打ち切られる(`internal/org/watch.go:956`)。`TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall` は 30 秒眠る claude の stub で、止める頼みから 5 秒以内に返ることを確かめる。直す前は stub の 30 秒を待つので落ちる。取り消した呼び出しが watcher の失敗として記録される点は N3 |
| L2 | 直った | `escalateAlert` は `context.Background()` から `inboxDesktopNotifyTimeout` の期限を作る(`internal/org/watch.go:1328-1335`)。テストは `TestWatch_Deadman_EscalateFuncGetsItsOwnDeadlineEvenWhenTheLoopIsCancelled`(`internal/org/watch_test.go:780`)に名前を変え、取り消し済みの ctx で `RunWatchAll` を回して、通知の ctx が終わっていないことと escalations.jsonl の 1 行を確かめる。直す前は `ctx.Err()` が Canceled なので落ちる。サイクルの関数(`cycle`・`pulse`・`pulseOrg`・`evaluateCycle`・`checkDeadman`・`escalateAlert`)から ctx の引数が消え、`evaluateCycle` と `RunWatchAll` の doc もそれに合わせて書き直された。ctx を取る関数が 1 つ残る(N4) |
| L3 | 一部 | 書く側は直った。`save` は `writeJSONFileAtomic`(同じディレクトリの一時ファイルと rename)で書く(`internal/org/watch.go:288-289`)。読めないファイルの扱いは、毎サイクルの 1 行が 1 回だけの 1 行に変わった。その org はプロセスが動いている間ずっと見張られないままで、`ralph status` の表示も変わらない(N2) |
| L4 | 直った | `sendInboxItemToHuman`・`inboxHumanTextFor`・`inboxDeadlineHumanText` の 3 つが同じ順番(`orgID, id, typ, reason`)になった。呼び出しは本番の 3 か所で、テストからは直接呼ばれていない(grep) |
| L5 | 直していない。tech-debt に記録 | `docs/tech-debt/README.md` の最後の行。書いてあること(`ensureWatchdogJoined` の `os.Getwd()` は `internal/org/watch.go:982`、`orgWatchChildArgs` は `--config` を渡されたときだけ付ける)は今のコードと合い、trigger は後半の PR で start が見張りを起動するとき。区切りの `|` は 6 個(5 列) |
| L6 | 直った | テストは「0644 を超える bit がない」と「所有者が読み書きできる」を確かめる(`internal/cli/org_watch_test.go:636`)。`startDetachedProcess` の doc も「0644 less the umask」になった |
| L7 | 直った | 計画の S3b の行が、`TestOrgWatch_RequiresOrgID` を消した理由と、拒否を引き継いだテストの名前を書くようになった |

## Evidence reviewed

- 全文を読んだ差分: 1b64ae2b と 188a640d の全ファイル(`internal/org/watchalive.go`、`internal/org/watch.go`、`internal/org/escalate.go`、`internal/org/deadline.go`、`internal/cli/org_watch.go`、`internal/cli/status.go`、4 つのテストファイル、`docs/tech-debt/README.md` の 2 行、計画の 2 行)
- 直しの周りで読んだ今のコード: `RunWatchAll`・`watchAllRun`・`pulseOrg`・`waitForNextCycle`・`finish`(`internal/org/watch.go`)、`sendAlert`・`raiseOrClear`・`SendWatchdogAlert`、`RunWatcher` と `realClaudeInvoke`(`internal/org/watcher.go`)、`callWithTimeout`(`internal/org/verbs.go`)、`writeJSONFileAtomic`・`ProbeWatch`・`AcquireWatchLock`(`internal/org/watchalive.go`)、`acquireHeldFileLock`・`fileLockHeld`(`internal/org/lockfile.go`)、`runOrgWatchAll`・`runOrgWatchDetach`・`runOrgWatchStop`・`newWatchdogHooks`(`internal/cli/org_watch.go`)、`statusDeadmanMinutes`・`orgWatchLine`・`readOrgWatchHeartbeat`(`internal/cli/status.go`)、`OrgConfig.DeadmanMinutes`(`internal/config/config.go`)
- 新しいテストが直す前のコードで落ちるかは、読んで確かめた(上の表)。テストは走らせていない
- 機械的な確認: ブランチ全体の追加コメントから識別子らしい語 238 個を取り出し、`internal/` と `cmd/` の定義と照合した。ないものは `writeFileAtomic` だけ(N5)。ctx を取る関数の残りは `grep -n 'ctx context.Context\|context.Background()' internal/org/watch.go` で数えた(N4)。旧いエラー文 `org: watch: write` / `org: watch: create state dir` に頼るテストや読み手は残っていない。直しの 2 行の tech-debt はどちらも区切りの `|` が 6 個。追加行にデバッグ出力、TODO、secret らしい文字列はない
- 直しの範囲: 1b64ae2b と 188a640d は指摘の範囲に収まり、関係のない変更はない(`git show --stat`)
- 確かめていないもの: 取り消された `claude -p` が `WaitDelay` の内に終わるか(実機)。`ralph insights` が `watcher_error` の receipt をどう見せるか(N3、`internal/insights/` の集計は読んでいない)

## 依頼された点の結論

1. サイクルの関数から ctx を外したこと: 正しい。サイクルの中で ctx を見るものはなくなり、ctx を見るのはサイクルの間の `waitForNextCycle` だけになった。`sendAlert` の使われない ctx の引数が 1 つ残る(N4)。tech-debt の「Stopping the cross-org watch waits for the current cycle」の行の Trigger(「org の間で止める頼みと loop の ctx を見る」)を実装するときは ctx を `pulse` に通し直すことになるが、行の記述は今も正しい
2. `SendWatchdogAlertBounded`: `callWithTimeout` と同じ打ち切り方で、親を呼び出し側の ctx にしたもの。止めたときに取り消せるよう、親を `context.Background()` にしていない。パルスの `sendAlert` が Background から作る `callWithTimeout` のままなのは、止める頼みでパルスの ALERT を切らないためで、2 つが違うのは正しい。`internal/cli` が呼ぶので export が要る。テストは期限切れと取り消しの両方を見る
3. `--once` 以外での hooks の ctx の取り消し: 置き場所は正しい。`RunWatchAll` が返った時点で `finish` が stopped の印を書いてロックを放しているので、`--stop` は「stopped」と言い、プロセスは watcher の呼び出しを畳んでから終わる。`--once` は今までどおり待つ(M-5 の意図を保つ)。副作用は N3
4. org ごとに 1 回の警告: N2
5. atomic な保存: `writeJSONFileAtomic` はディレクトリを作り(前の `save` と同じ)、一時ファイルを同じディレクトリに作るので rename は同じファイルシステムの中で済む。一時ファイルの名前(`.watch-status-<org>.json.<乱数>.tmp`)は、名前で状態ファイルを開く `readOrgWatchHeartbeat` に拾われない。ファイルの末尾に改行が付くようになったが、読み手は JSON として読むだけ。rename そのものを見分けるテストはない(Coverage gaps)
6. `ProbeWatch` の窓の規則: `max(deadman_minutes(1 未満なら 10), 2 × interval_seconds)`、比較は `>`(ちょうどの年齢は running)。印の `interval_seconds` が 0 か負(テストの印、手で書き換えた印)なら deadman だけで決まる。間隔の長い見張りが固まると最大 2 間隔のあいだ running に見えるが、これは窓の決め方から来るもので、計画のずれの記録に書いてある。次の PR の spawn と TASK の検査も同じ規則を受け継ぐ。規則を説明する文の残りは N1

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | readability | M1 の直しで生存の印の窓の規則が変わったが、規則を説明する文が 3 か所で古いまま残った。(a) `OrgConfig.DeadmanMinutes` の doc と、`ralph init` が配る `templates/base/ralph.toml` のコメントは、`deadman_minutes` を「生存の印が stale になる年齢」と書く。今の窓は `deadman_minutes`(1 未満なら 10)と `interval_seconds` の 2 倍の大きいほう。どちらの文も、0 が「デッドマンは時間切れで上げない」の意味だとは書いていない。この意味は直しの理由そのもので、`ProbeWatch` の新しい doc には書いてあるが、設定を書く人が読む 2 か所にはない。(b) `ralph status` の stale の行は「older than deadman_minutes (and two of its intervals)」と出す。`deadman_minutes = 0` の台帳では、実際の窓は 10 分なのに「0 分より古い」と読める | `internal/config/config.go:58-61`。`templates/base/ralph.toml:51-53`。`internal/cli/status.go:171`。新しい規則は `internal/org/watchalive.go:166-176`(doc)と `:195-200`(`watchAliveWindow`) | (a) 2 つの文を「…, and, with interval_seconds, how old the liveness mark may get (the larger of this, 10 when below 1, and two intervals); 0 turns off the timeout escalation」の形にする。(b) stale の行の末尾を「older than its window (deadman_minutes, 10 when below 1, and at least two of its intervals)」にし、`TestStatusCmd_OrgWatchLineAndJSONPerWatchState` の `staleTail` も合わせる |
| N2 | LOW | exception-handling | L3 の読めない状態ファイルは、知らせ方が弱くなった。直しのあと、読めないファイルの org は `watch.log` に 1 回だけ書かれ、あとはプロセスが動いている間、毎サイクル黙って飛ばされる。その org の stall・生存・デッドマンの見張りは止まったままで、`ralph status` の org の行は parse できないファイルを「まだ watch されていない」と同じに出す(変わっていない)。見張りは何日も動くので、1 行は何日も前のログに埋もれる。直す前は 30 秒ごとの 1 行で、うるさいが目に入った。最初の結果が勧めたのは「1 回だけ知らせて新しい状態から始める」か「`ralph status` に読めないと出す」で、直しは「1 回だけ」だけを採り、飛ばし続ける。計画の Progress は L3 を直ったと書き、tech-debt にも行がない。ほかに 2 点: 行の「(said once)」は操作者向けの言い方として分かりにくい。`unreadableWarned` は、走っていない org を `statuses` から消す `pulse` の掃除の対象外 | `internal/org/watch.go:620-640`(`pulseOrg`、警告は `:630-632`)、`:604`(`statuses` だけを消す)。`internal/cli/status.go:274-288`(`readOrgWatchHeartbeat` は parse の失敗も `ok=false`)。`docs/plans/active/2026-10-11-org-cross-watch.md:209`。`TestRunWatchAll_AStatusFileThatCannotBeReadIsSaidOncePerOrg`(`internal/org/watch_all_test.go:527`)は、飛ばし続けることを意図として固定する | どちらかにする。(1) `readOrgWatchHeartbeat` に「読めない」の 3 つ目の状態を足し、`ralph status` の org の行と `--json` に出す。見張りの側は今のままでよい。(2) 直さないなら tech-debt の「Small leftovers of org-cross-watch」の行に足し、計画の Progress の「L3 を直し」を「L3 の書く側を直し」に直す。どちらでも、行の言い方は「(not repeated)」程度にする |
| N3 | LOW | maintainability | L1 の直しで、`--stop` も実行中の watcher の呼び出しを取り消すようになった。取り消された呼び出しは、本物の失敗と同じに記録される。`RunWatcher` は `claude` の失敗(`signal: killed`)で `watcher_error` の receipt を `model-receipts.jsonl` に足し、hooks は stderr に「watchdog: watcher error for org …」を書く。receipt は見張りがロックを放したあとに書かれ、`ralph insights` の Receipts 節では watcher の失敗として数えられる。今まで同じことが起きたのは SIGINT と SIGTERM のときだけだった。scope_change は作業中の座席で何度も立つので、判定の最中に止める場面は珍しくない(最初の結果の L1 と同じ理由)。新しいテストは stderr の「watcher error」の行を期待どおりとして固定する | `internal/org/watcher.go:295-298`(`invokeErr` のとき `recordWatcherReceipt(..., watcherErrorReason)`)、`:252`(期限切れは区別するが、取り消しは区別しない)。`internal/cli/org_watch.go:490-494`。`internal/cli/org_watch_test.go:738` | `RunWatcher` で `ctx.Err() == context.Canceled` のときは receipt を書かない(または `watcher_cancelled` の理由で書く)。hooks では同じ条件で「watcher call cancelled because the watch stopped」と書き、テストの期待も合わせる。直さないなら N2 と同じ tech-debt の行に足す |
| N4 | LOW | maintainability | サイクルの関数から ctx を外す掃除が 1 つ残った。`sendAlert` はまだ最初の引数に `_ context.Context` を取り、doc は「Its context argument is not used」と書く。本番の呼び出しは `raiseOrClear` の 1 か所で、`context.Background()` を渡す。テストの呼び出しも 1 か所。1b64ae2b が 6 つの関数から ctx を外し、`evaluateCycle` の doc に「So the cycle takes no context」と書いたので、読み手は ctx を取る関数が 1 つだけ残っている理由を探すことになる | `internal/org/watch.go:996-999`(doc の最後の文と signature)、`:923`(呼び出し)。`internal/org/watch_test.go:643` | 引数と doc の最後の文を消し、2 か所の呼び出しから `context.Background()` を外す |
| N5 | LOW | naming | `writeWatchAlive` の doc が、存在しない関数 `writeFileAtomic` を指す。呼んでいるのは `writeJSONFileAtomic`。996e5c48 からあり、最初の結果は見落とした | `internal/org/watchalive.go:158` | `(writeJSONFileAtomic)` に直す |

## Positive notes

- 直しのテストは、どれも直す前のコードで落ちる形になっている(上の表)。L2 のテストは、前の版が「親の ctx から作る」を意図として固定していたことを、名前と doc の両方で書き換えた
- L2 は `escalateAlert` だけを特別扱いせず、「サイクルは ctx を受け取らない」という規則にまとめた。`evaluateCycle` と `RunWatchAll` の doc がその規則を言うので、次にサイクルへ呼び出しを足す人が迷わない
- M1 は呼び出し側 3 つではなく `ProbeWatch` の中で直したので、次の PR の spawn と TASK の検査も同じ規則を受け継ぐ
- L3 のテストは 3 サイクル回して、警告が 1 回だけであること、ファイルを消したあとのサイクルでその org の見張りが戻ること、もう一方の org が 3 サイクル進むことを確かめる
- tech-debt の M1 の行は、解けた部分だけを取り消し線にして、残る場合(`./ralph.toml` の deadman より長いサイクル)を書いた。L5 の行の記述はコードと合う

## Coverage gaps

- rename で書くことを見分けるテストがない。`TestRunWatchAll_AStatusFileThatCannotBeReadIsSaidOncePerOrg` の「`.tmp` が残らない」の確認は、一時ファイルを使わない直す前の `save` でも通る
- 取り消された `claude -p` が `WaitDelay` の内に終わるかは、実機で確かめていない(テストの stub は `exec sleep` なので、kill ですぐ終わる)
- 最初の結果の Coverage gaps(切り離した子の osascript の表示、herdr が止まっているときの 1 サイクルの長さ、Linux での setsid、`watch.log` が切り詰められないこと)は変わっていない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (このレビューでは足していない) | | | | |

_(N2・N3 を直さずに cross-review に進む場合は、tech-debt の「Small leftovers of org-cross-watch」の行に足す。N1・N4・N5 は数行の直しなので、行にしないで直す方が安い。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。今回の LOW は N1〜N5 の 5 件。最初の結果の M1・L1・L2・L4・L6・L7 は直り、L3 は一部で残りが N2、L5 は tech-debt に記録した)
- Follow-ups:
  - N1・N4・N5: コメント 3 か所と status の 1 行、引数 1 つで、合わせて 15 行ほど。直すなら /verify の前に。`.go` を触るので /self-review から回し直す(cross-review の前なので cycle は増えない)
  - N2・N3: 直すか、tech-debt の行に足す。N2 を直さない場合は、計画の Progress の L3 の記述も合わせる
  - /sync-docs に渡す: 窓の規則の変更に合わせる文書が 2 つある。仕様 FR-6 の注(`docs/specs/2026-10-07-org-multi-org-director.md:63`、「`deadman_minutes` 以内」)と、`/org` skill の 4 面の 775 行目(「`[org].deadman_minutes` より古い」)。N1 (a) をコードで直さない場合は `templates/base/ralph.toml:51-53` も
  - このスラグの insight event はまだない(`docs/insights/events/` に `org-cross-watch` のファイルがない)。最初の結果も追記していない。今回は依頼どおり report だけをコミットしたので、追記していない

## 付録: cycle 1 の最初の結果(b25e47da の時点)

最初の結果の header:

- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)
- Scope: diff の品質だけ。`git diff 577ebeaf...HEAD`(HEAD b25e47da、40 ファイル、+5,594/-986)。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない
- 番号の付け方: MEDIUM は M1、LOW は L1〜L7

### Evidence reviewed(最初の結果)

- 全文を読んだ差分: `internal/org/watchalive.go`、`internal/org/watch.go`(`RunWatchAll`、`watchAllRun`、`evaluateCycle`、呼び出しの打ち切り、`realGitStatus`、`escalateAlert`)、`internal/org/deadline.go`、`internal/org/escalate.go`、`internal/org/lockfile.go`、`internal/cli/org_watch.go`、`internal/cli/status.go`、`internal/config/config.go`、`scripts/ralph-config.sh`(root と `templates/base/` は一致)、`templates/base/ralph.toml`、`internal/org/spawn.go`・`verbs.go`・`watcher.go`・`inbox.go` の差分
- 移動したコード: `internal/cli/org.go` から消えた行と `org_inbox.go`・`org_watch.go` の行を突き合わせた。`org_inbox.go` で base の `org.go` にない行は、`InboxHintStateDir` への付け替え、`wireOrgInbox` の呼び出し、doc の書き直しだけ
- テストは中程度の深さで読んだ: `internal/cli/org_watch_test.go`(起動の差し替え、シグナル、`--detach`、`--stop`、`startDetachedProcess`)、`internal/org/watch_all_test.go` の補助と見出し、`internal/org/watchalive_test.go` のロックの競り合い、`internal/org/watch_test.go` の書き換えた 3 本、`internal/cli/main_test.go`
- テストの在庫: diff から消えた `func Test` 4 本のうち 3 本は `TestRunWatchAll_*` に名前を変えて残り、`TestOrgWatch_RequiresOrgID` だけが消えた(`comm` で集合の差を取った。L7)
- 機械的な確認: 追加行にデバッグ出力、TODO、secret らしい文字列はない。`chmod` で書き込みを失敗させるテストはない(root の CI でも落ちない)。新しい関数の上に別の関数の doc が残っていない。`RunWatch`・`WatchParams`・`watch --org-id` の残りは、置き換えの経緯を書いたコメントと計画・仕様・過去の report だけ。新しい tech-debt の 5 行は、どれも区切りの `|` が 6 個(5 列)
- 確かめていないもの: 切り離した子の osascript の通知が実機で出るか(計画の Open question)。agmsg の Join に空の作業ディレクトリを渡したときの動き(L5)。どのテストも走らせていない

### Findings(最初の結果)

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

### Positive notes(最初の結果)

- `--stop` はシグナルを送らず、台帳の頼みのファイルで見張りに止まってもらう。印の pid を使い回されたプロセスに送る危険を、設計の段階で消している。`TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal` は、印が生きている別のプロセス(`sleep 30`)を指すときにそのプロセスが残ることまで確かめる
- `startDetachedProcess` は子を回収するまで自分の子として持つので、`stop` の SIGTERM と SIGKILL が別のプロセスに当たらない。`--detach` の失敗の経路は、自分が起動した子を止めて回収し、ロックが放されたかを見てから返る
- `started_at` をロックを待つ前に決め、古い頼みだけを起動のときに消す。こうしたので、ロックを取ってから印を書くまでの間に書かれた `--stop` も失われない。時刻をナノ秒で持つ理由も doc に書いてある
- パルスの呼び出しを 1 回ごとに Background から作る期限で打ち切る。こうすると、SIGINT がサイクルの途中で来ても、生きている座席の liveness ALERT が偽って出ない。この利点と、止まるまでが延びる代償の両方を tech-debt に書いている
- manifest から読んだ org_id を `ValidateIdentifier` に通してから状態ファイルの名前にする。手で書き換えた行でパスを作れない
- テストは test binary を exec しない。`orgWatchLaunch` を TestMain で panic する偽物にし、`--detach` は同じプロセスの中で動く見張りで、`startDetachedProcess` はシェルスクリプトで確かめる。flock が開いたファイルに結びつくことを使い、同じプロセスの中でもロックの競り合いを再現している
- 移動したコード(`org_inbox.go`、`org_watch.go`)は、付け替えの行を除いて base と同じ行で、関係のない変更が混ざっていない

### Coverage gaps(最初の結果)

- 実機の確認はしていない: 切り離した子の osascript の表示、herdr が止まっているときの 1 サイクルの長さ、Linux での setsid
- L5 の「作業ディレクトリが消えたあとの Join と `claude -p`」の実際の動きは読んでいない(agmsg の script は repo の外にある)
- `watch.log` は追記だけで、切り詰めない。L3 の 1 行のような毎サイクルの失敗が続くと、1 つの条件で 1 日 2,880 行ほど増える。finding にはしていない。見張りを何日も動かす運用(ユーザーの決定)では、次の PR で start が起動するときに扱いを決めるとよい

### Tech debt identified(最初の結果)

- M1 と L1〜L6 を直さない場合は、tech-debt に 1 行にまとめて足す。M1 は既存の 201 行(`ralph status` と `--config` の食い違い)に足すのが自然
- L7 は計画の記録の直しだけで、tech-debt には要らない

### Recommendation(最初の結果)

- 最初の結果の判定: yes(CRITICAL・HIGH なし。MEDIUM 1 件(M1、`deadman_minutes` が 1 未満の設定で `--detach` が健全な子を止めて誤った理由で失敗する)、LOW 7 件)
- Follow-ups: (1) M1 は `/verify` に進む前に直すのが安い。config の検査を 1 つ足すか、呼び出し側で既定に置き換え、テストを 1 本足す。(2) L1・L2・L3 はどれも数行の直しで、見張りが人に届ける経路の取りこぼしを減らす。直すなら M1 と同じコミットでよい。(3) L4・L6 は機械的な直し。L5 は次の PR(start が見張りを起動する)の前に方針を決める。(4) L7 は計画の 1 行の直しで、記録だけ。`.go` を触れば diff が変わるので、`/self-review` から回し直す(cross-review の前なので cycle は増えない)
