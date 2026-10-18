# Self-review report: org-director-liveness

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-18-org-director-liveness.md`(ステージ 8b-2a、パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 88e37b1d...HEAD`(HEAD 27a7ec98、46 ファイル、+6534/-288)。diff の品質だけを見た。仕様適合、テストの網羅、文書のずれ、静的解析、テストの実行は `/verify` と `/test` に任せ、このレビューでは走らせていない

## Evidence reviewed

- 全文を読んだ: `internal/org/director_check.go`(新規、498 行)、`internal/org/director_start_cleanup_test.go` の 1 から 805 行
- diff を全部読んだ: `internal/org/director.go`、`director_start.go`、`director_stop.go`、`director_wake.go`、`watch.go`、`spawn.go`、`verbs.go`、`driver/herdr.go`、`internal/cli/org_director.go`、`org_director_headless.go`、`org_watch.go`、`status.go`、`internal/org/director_cmd_test.go`
- 周辺の本体を読んだ: `withFileLock` と `manifestLockTimeout`、`stopDirectorProcess`、`readDirectorProcess`、`directorProcessEnded`、`confirmSeatPane`、`callWithTimeout`、`PrintableInboxText`、`inboxField`、`DirectorRegistration.describe`、`DirectorCleanupError.Error`、`checkDirectorStart` 全体、`newOrgDirectorHerdrRuntime`、`DirectorWakeMemory.warnOnce`
- 文書: `/director` skill の diff、`docs/tech-debt/README.md` の変わった行(セルの数と取り消し線の形)、`docs/evidence/director-liveness-livefire-2026-10-18.md`(`$HOME`、token、pane の値、secret の形の grep)。README、AGENTS.md、仕様、recipe は本文の diff を拾い読みしただけで、コードとの突き合わせは `/verify` に任せた
- テスト: `director_check_test.go` の定数・fixture・JSON と印のテスト、`watch_all_test.go` の diff の前半(サイクルの順のテスト)、`internal/cli/org_director_cleanup_test.go` の 1 から 140 行と 370 から 497 行。ほかの新規テストは名前と、signal・exec・環境変数・実 ps・sleep の grep だけ見た。全ケースは読んでいない
- 機械的な確認: `git diff --check` は空。追加した Go の行に TODO・FIXME・debug の出力はない(`fmt.Fprintln(p.Notice, ...)` は意図した出力)。追加したコメント行で 84 桁を超えるものはない。skill の 4 面(director、org)と recipe の 2 面は `cmp` で一致。tech-debt の変わった行は 5 セルのまま(`|` の混入なし)
- 依頼された重点(版の結びつけ、start の片付けが版 A だけに打つこと、見張りのサイクルの順、signal と close の送り先、mark と登録から読んだ値の escape、テストが自分で起こしていないプロセスや実物に触れないか、コメントと名前の正確さ)は、下の「確認して問題がなかったもの」にまとめた

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 4 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | exception-handling | L1: `MarkCmdLaunched` は director.lock を持ったまま、starter の ps を最大 10 秒まで読む。ほかの director.lock の待ち手は 5 秒で諦める。doc は「ps にかかる時間だけ lock を持つ、その上限の範囲で」と書くが、その上限(10 秒)が待ち手の上限(5 秒)より長いことは書いていない。起動スクリプトの check-in で ps が詰まる(負荷の高い機械、ps のハング)と、同じ登録を見ている見張りの `CheckDirector`(各サイクル `readDirectorVersion` で lock を取る)は 5 秒で `watchdog: director check skipped this cycle: org: acquire director lock ...: timed out` を出してそのサイクルの確かめを落とし、同時に走った `director stop`・`detach`・`attach` は待たずにロックのタイムアウトで失敗する。通常の ps は数十 ms なので、起きるのは詰まったときだけで、見張りの確かめは次のサイクルで戻る | `internal/org/director_start.go:343-347`(`directorCheckInStarterTimeout = defaultDriverCallTimeout`、10 秒)、`:371-373`(doc)、`:425`(`withLock` の中での呼び出し)。`internal/org/lockfile.go:30`(`manifestLockTimeout = 5 * time.Second`、`withFileLock` の待ちの上限)。同じ形の過去の指摘は reviewer memory の pattern-lock-scope-contradicts-comment(ロックの取得上限が、中で起こる外部呼び出しの最悪値より短い) | starter が終わったことは戻らない(gone、zombie、起動時刻の違いは元に戻らない)ので、ps は lock を取る前に読み、lock の中では読んだ結果で決めても、受け付けるか断るかは変わらない。計画の D2 が「ロックの下で読む」と書いているなら、lock の中の読みの上限を `manifestLockTimeout` より短く(例: 2 秒)するか、doc に「待ち手の 5 秒より長い」と書く |
| LOW | naming | L2: 同じ 4 つの field(state、how、checked_at、detail)の struct が 3 つあり、そのうち `DirectorCheck` の JSON の説明が実物と合わない。`DirectorCheck` の doc は「その JSON は計画が `director-alive.json` に与えた形」と言うが、印の実体は `attached_at` を足した `DirectorAlive`(6 キー)で、status の `director.check` は id を持たない `orgDirectorReportCheck`(4 キー)。`DirectorCheck` は production のどこでも marshal されず、JSON tag が効くのは `TestDirectorCheck_JSON` だけで、そのテストのコメントは「`director-alive.json` と status の `director.check` が持つ 5 キー」と言う(ファイルは 6、status は 4)。`DirectorCheck` の tag を直しても印にも status にも効かず、テストは緑のままなので、直した人は直ったと思う。`DirectorAlive` の field を増やしても `orgDirectorReportCheck` は増えず、status の JSON だけが置いていかれる | `internal/org/director_check.go:120-132`(doc と tag)、`:434-441`(`DirectorAlive`)。`internal/org/director_check_test.go:691-700`。`internal/cli/org_director.go:626-631`(`orgDirectorReportCheck`)、`:727`(それを `org.DirectorCheck` に戻して `orgDirectorCheckText` に渡している) | `DirectorCheck` の JSON tag とそのテストを外し、doc を「JSON にしない。印は `DirectorAlive`」に直す。もう 1 歩進めるなら、`orgDirectorReportCheck` を `DirectorAlive.Check()` から作って 3 つ目の写しをなくす |
| LOW | maintainability | L3: 新しい機能が 800 行を超えるファイルに入った。`internal/cli/org_director.go` は 780 から 941 行になり、この PR で 800 を超えた。`internal/org/director_stop.go` も 740 から 871 行で同じ。`internal/org/director_start.go` は元から 977 行で、1214 行になった(+24%)。新しいファイル `director_check.go`(498 行)を作ったのに、片付け(`cleanUpEndedDirector` と通知の 2 関数、約 130 行)は `director_stop.go` に、start の確かめと starter(`checkCurrentDirector`、`directorStarter`、`MarkStoppedAtVersion`、`directorVersionText`、約 240 行)は `director_start.go` に、CLI の確かめの文(`attachHeadlessChecks`、`orgDirectorCheckText`、`orgDirectorReportCheck`、`readOrgDirectorCheck`、`orgDirectorWatchCheckClause`、約 160 行)は `org_director.go` に入っている。新しいテストファイルも `director_check_test.go` 854 行、`director_start_cleanup_test.go` 805 行と、わずかに 800 を超える。`watch.go`(1613 行)、`spawn.go`(2354 行)など、すでに 800 を超えるファイルは repo にあるので、この PR だけの問題ではない | 行数は `git show 88e37b1d:<path> \| wc -l` と今の `wc -l` の比較(director_start.go 977→1214、director_stop.go 740→871、org_director.go 780→941、director.go 906→947)。`director-wake` の self-review の L 項目(`org_director.go` 727→867)と同じ形 | 次の動きの変更に入る前に、片付けを `director_cleanup.go` に、CLI の確かめの文を `org_director_check.go` に、動きを変えない移動だけのコミットで分ける。移動は全パイプラインの再走を起こすので、この cycle で直さないなら tech-debt に 1 行にまとめる |
| LOW | security | L4: `stubHerdrAgentNotFound` は、PATH 上の `herdr` を `exec.LookPath` で探して、`agent get` 以外の呼び出しをそのまま `exec` で渡す。先に `directorStartLedger`(`setupOrgStubPATH`)が stub を PATH の先頭に置いていれば、渡す先は stub になる。唯一の呼び出し元(`:80`)はそうしている。しかし helper はそれを前提にしていることを書かず、確かめもしない。別のテストが `directorStartLedger` なしで呼ぶと、開発者の本物の `herdr` が見つかり、`pane close`・`workspace close` が本物の server に届く(AC9 の「本物の herdr を打たない」に反する) | `internal/cli/org_director_cleanup_test.go:30-47`(`exec.LookPath("herdr")` の結果をそのまま `exec` する)、`:78-80`(呼び出し元) | `LookPath` の結果が `t.TempDir()` か `os.TempDir()` の下であることを確かめて、そうでなければ `t.Fatalf` する(helper の doc に「`setupOrgStubPATH` の後」と書くだけでも、次の呼び出し元は気づける) |

## 確認して問題がなかったもの

- **版の結びつけ。** `directorVersionOf` は `id`、今の登録の id、その id の event の数(parse できる行のうち `ev.ID == id`)、fold が読んだ登録の `%#v` を比べる。`DirectorRegistration` の `Wake` は `[]string` で `%#v` に pointer が出ず、2 回の読みで安定する。`CheckDirector` は版 A を lock なしで読み、ps と herdr を読み、`director.lock` の下で読み直す。差があれば捨てて 1 回だけ確かめ直し、2 回続けて変われば `unknown`(`changed`)で、そのときの登録とその版を返す。lock の下では herdr も ps も打たない。2 回目に確かめるものがなければ `Checked=false` を返す。読んでいる途中の半端な行は、読み直しで event の数が変わるので `changed` に落ちる。
- **start の片付けが版 A だけに打つこと。** `cleanUpEndedDirector` は `chk.Registration` を受け取って、登録を読み直さない。herdr は `closeHerdrDirector(reg)`(label の照合のあとに C-c と close、閉じる id は reg のもの)、launch_cmd は `stopDirectorProcess(reg)`(`reg.PID <= 0` なら何も送らず、alive で起動時刻が記録と同じときだけ SIGTERM、SIGKILL は送らない)。最後の記録は `MarkStoppedAtVersion` で、lock の下で、ほかの何より先に版を比べ、版が違えば何も書かず `ErrDirectorChangedSinceCheck` で断る。ゼロの版は一致しない(`id` が空なので)。`StopDirector` から `closeHerdrDirector` を取り出した変更は、移す前の本体と一致する(文面の `org: director: stop <id>: ` だけは呼び出し元が付ける形になり、`%w` で `ErrDirectorStopSelfPane`・`ErrDirectorStopSelfWorkspace` は残る)。
- **見張りのサイクルの順と書き込み。** `cycle()` は「期限 → 確かめ → 起こす → パルス → 印」で、`RunWatchAll` の doc の番号(1〜5、失敗の節は 1〜4)とコメント、`WakeDirector` の手順の番号(3 を足して 4〜6 に繰り下げ)、`sendDirectorWake` の doc の「steps 5 and 6」、テストのコメント(`step 3`、`step 2`)がそろっている。確かめが失敗しても `wake(nil)` で進む。印は `Checked` のときだけ書き、`director.jsonl` には何も書かない。`directorCheckFoundEnded` は id と `attached_at` の両方を比べる。
- **escape。** `attachHeadlessChecks`、`orgDirectorCheckText`、`orgDirectorWatchCheckClause`、`startAttachedError` の確かめの文は、`inboxField`(= `PrintableInboxText`)を印の `ID`・`CheckedAt`・`State`・`How`・`Detail`、`markErr` に通している。`cur.StarterPID` は int、`cur.ID` は fold が `d<N>` を検査している。`PrintableInboxText` は制御文字だけを書き換えるので、書き込み側(`checkDirectorOnce`)と読み出し側の二重の escape は冪等。片付けの通知 2 行は `PrintableInboxText` を C-c・pane・workspace の注と `stop.note` に通し、`reg.Token` から作る名前は 12 桁の 16 進で fold が検査している。`describe()` の `Driver` は fold が `claude` か `codex` に限っている。`--json` の `check` は印の値をそのまま JSON の文字列にする(登録の値と同じ扱い)。
- **テストの安全。** 新しいテストの ps は、fake の `DirectorProcessReader`(`procs` の表、pid 4242 と 5151 は表になければ gone)か、テスト自身の `os.Getpid()` と、自分で起こして待った子の pid だけを読む。SIGTERM が届く道(`stopDirectorProcess` で alive かつ起動時刻が同じ)は、自分で起こした `sleep` の pid だけで通る。確かめが ended でも片付けが alive を読む組は、fake の pid には作られていない。herdr は fake か、`setupOrgStubPATH` の stub。`deadChildPID` は pid の再利用がありうることをコメントが認めており、起きても落ちるだけで signal は誰にも届かない。
- **ほか。** `AttachHeadless` の starter の検査は、読み書きより前に pair の片方だけ・負の pid を断る。`directorStarter` は `checkDirectorStart` のあとに ps を打つので、断られた start は自分の起動時刻を読まない。`directorCheckInStarterEnded` は ps の失敗と知らない state を「受け付ける」に倒す(doc と一致)。`checkHerdrAgentGetResult` の文面は `checkHerdrEnvelopeError` と同じで(`%w (%s)` と `%w (%w)` は同じ文字列)、座席のパルスの `AgentGet` の挙動は変わらない。`checkDirectorStart` の順(入力の検査 → 最初の確かめ(片付け)→ 見張りの検査)は doc と help が書いたとおりで、「片付けたあとで見張りの検査に断られても、片付けた 1 行は残る」も help に書いてある。コード内の plan の引用 3 か所が `docs/plans/active/...md` の形(`director_check.go:17` など)だが、同じ形の先例が `merge_wait.go`、`skill_frontmatter_test.go`、`approval_enforce_test.go` にあるので上げない。`Org` に足した 2 つのテスト用の関数 field は、`DirectorStore.beforeMarkStopped` の先例と同じ形。

## Positive notes

- 確かめの結果を版に結びつけ、片付けも記録も版 A だけに向けた作りが、コードと doc と test の 3 つで一貫している。「読み直す変異」を捕まえる seam(`directorStartAfterCheck`)と、版が変わる 3 つの道(event が増える、別の登録が current になる、記録の前に差し込まれる)のテストが、順を決めて書かれている。
- 見張りは書くものを印 1 つに絞り、片付けと close を start の側に寄せた。ended と読んだ登録に起こす 1 行を送らない理由(pane は shell に戻っていて、行がコマンドとして読まれる)がコードの doc に書いてある。
- 文書の更新が広い範囲に及ぶのに、skill 4 面と recipe 2 面は byte 一致で、tech-debt の 255 行は「解決した分」と「残る分(unknown の各 how)」を同じ Debt セルで書き分け、トリガーの欄も直してある。
- 実機の evidence は `$HOME` を `~` に、token・pane・pid を `<...>` に置き換えてあり、secret の形(`--token`、`api_key` など)も長い 16 進もない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 片付け(`cleanUpEndedDirector` と通知)、start の確かめと starter、CLI の確かめの文が、800 行を超えるファイル(`director_stop.go`、`director_start.go`、`org_director.go`)に入っている(L3) | 次に director を触る変更が、1000 行を超えるファイルの中で探すことになる | 移動だけのコミットでも全パイプラインの再走を起こす。動きは変わらない | 次に `director_start.go` か `org_director.go` を変える計画。または今の cross-review の修正 cycle でほかの修正と一緒に | `docs/plans/active/2026-10-18-org-director-liveness.md`、このレポートの L3 |

_(上の行は、依頼により `docs/tech-debt/README.md` には足していない。このレポート以外のファイルは変えないという指示のため。足す場合は lead が決める。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 4 件は、check-in の lock の中の ps の上限が待ち手より長い点(L1)、同じ field の struct の写しと、効かない JSON の説明(L2)、800 行を超えたファイル(L3)、stub herdr の helper が前提を確かめない点(L4)で、いまの動きを壊すものはない。版の結びつけ、片付けの送り先、見張りの順、escape、テストの安全に欠陥は見つからなかった)
- Follow-ups:
  - L1 と L4 は数行の直し。L2 は tag とテストを外す直し。どれも production の動きは変わらないが、コードに触るので、直すなら post-implementation pipeline を最初から回す(記録だけの修正の例外には入らない)
  - 修正 cycle を作るなら、L1・L2・L4 をまとめて 1 回で直す。作らないなら、L1 から L4 を known gaps として PR に書き、L3 は tech-debt の 1 行にする
  - insight event(`docs/insights/events/`)は、「このレポート以外のファイルは変えない」という指示のため、追記していない
