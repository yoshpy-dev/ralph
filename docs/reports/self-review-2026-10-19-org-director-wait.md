# Self-review report: org-director-wait

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-19-org-director-wait.md`(ステージ 8b-2b、パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff e38b89a9...HEAD`(HEAD 7368dee5、38 ファイル、+4895/-165)。diff の品質だけを見た。仕様適合、テストの網羅、文書のずれ、静的解析、テストの実行は `/verify` と `/test` に任せ、このレビューでは走らせていない

## Evidence reviewed

- 全文を読んだ: `internal/org/director_wait.go`(新規、736 行)、`internal/cli/org_wait_director.go`、`internal/cli/org_director_wait_line.go`、`internal/cli/exitcode.go` と `exitcode_test.go`、`internal/cli/org_wait_director_test.go`(712 行)
- diff を全部読んだ: `cmd/ralph/main.go`、`internal/cli/org_gate.go`(`signalCauseContext`)、`internal/cli/org.go`、`internal/cli/org_director.go`、`internal/cli/status.go`、`internal/cli/org_legacy_ledger.go`、`internal/org/escalate.go`、`internal/org/spawn.go`、`internal/org/prompts/director.md` と `prompts_test.go`、既存テスト 5 ファイルの変更(`org_director_test.go`、`org_director_headless_test.go`、`org_director_wake_test.go`、`org_director_check_line_test.go`、`status_test.go`)
- 周辺の本体を読んだ: `ProbeWatch`、`WatchAcceptsWork`、`WatchKillHint`、`CommandWithStateDir`、`DirectorRegistration.describe`、`PrintableInboxText`、`inboxField`、`newOrgRuntime` と `newOrgRuntimeAt`、`orgInboxHintStateDir`、`gateSignalContext` の 3 つの呼び出し元(gate、regate、merge)、Go の runtime の `sigInstallGoHandler`
- 一部だけ読んだ: `internal/org/director_wait_test.go`(1252 行のうち 1 から 135 行と 600 から 1000 行)。`internal/cli/org_director_wait_line_test.go`(648 行)は定数と関数名と signal・exec・sleep の grep だけ
- 文書: `/director` skill の diff(昇格、wait で待つ、してはいけないこと)、`/org` skill の diff、README と AGENTS.md と仕様の注は変更箇所の文面、tech-debt の変わった 5 行を cell ごと(row 306 の 800 行の row も)、evidence 2 ファイルは `$HOME`・token・pane・長い 16 進の grep と冒頭。skill の中身と実装の突き合わせは `/verify` に任せた
- 機械的な確認: `git diff --check` は evidence に 10 行(L2)。追加した Go の行に TODO・FIXME・debug の出力はない(テストの child 側の `fmt.Printf("returned: ...")` は意図したもの)。追加したコメント行で 90 桁を超えるものはない。skill 4 面は root、`.agents/`、`templates/base/` で `cmp` が一致し、templates 側の追加行に `internal/`・`cmd/`・plan・仕様への参照はない。tech-debt の追加・変更行は 5 cell のままで、cell 内に生の `|` はない。typo の grep(よくある綴り間違い)は空
- 依頼された重点(印の持ち主、見張りの確かめと 4 の遷移、`watch_accepting` の引き継ぎ、誤りの経路、`signalCauseContext` と `notIgnoredSignals` と再送、`ExitCode`、印の値の escape、`openInboxItems` の抽出で plain wait が変わらないか)は、下の「確認して問題がなかったもの」にまとめた

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

CRITICAL 0 件、HIGH 0 件、MEDIUM 1 件、LOW 7 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1: register の row 306 が trigger に書いた変更がこの PR で起きたのに、row は更新されておらず、800 行を超える新しいファイルも増えた。row 306(org-director-liveness の self-review L3 から)は `internal/cli/org_director.go` を「794 行、800 まで 6 行」と書き、Trigger に「次に `org_director.go` を 800 行より先に進める変更」を挙げる。この PR は 794 から 831 にしたが、row は変わっていない(変わった tech-debt の行は 209、247、新しい 1 行、`attach --replace` の行)。新しい `org_director_wait_line.go`(116 行)を作ったのに、report の struct の field、`readOrgDirectorReport`・`orgDirectorLine`・`printOrgDirectorReport` の変更、`director status` の help 16 行は `org_director.go` に入った。新しいテスト `internal/org/director_wait_test.go` は 1252 行で、1 ファイルに「印の読み書き」「持ち主」「引き継ぎと 4」「登録の変化」「書けない印」「読めない受信箱」が入っている。`director_wait.go` は 736 行で、次の変更で 800 を超える。`org.go`(1954)、`status.go`(890)、`spawn.go`(2356)は元から超えているので、この PR だけの問題ではない。row に書いた trigger を実行する機会を、先送りにする理由が plan にもない | 行数は `git show e38b89a9:<path>` を `wc -l` に渡した値と今の `wc -l` の比較(`org_director.go` 794 から 831、`org.go` 1917 から 1954、`status.go` 878 から 890)。`docs/tech-debt/README.md:306`(Debt の「794, 6 lines short of 800」と Trigger の「the next change that takes `internal/cli/org_director.go` past 800」)。`internal/cli/org_director.go:309-324`(help)、`:540-562`(struct と doc)、`:600`、`:626-637`、`:660-665`、`:826`。`internal/org/director_wait_test.go`(1252 行) | row 306 の Trigger どおりに払うなら、`printOrgDirectorReport` と `orgDirectorLine` の確かめの文を、動きを変えない移動で `org_director_status.go` のような別ファイルへ出す。テストは「持ち主と書き込み(AC2、AC3、書けない印)」と「見張りと引き継ぎ(AC5)」で 2 から 3 ファイルに分ける。この cycle で直さないなら、row 306 の Debt を 831 行と、`director_wait_test.go` 1252 行、`director_wait.go` 736 行に書き直し、Related に今の plan を足す。どちらでも row を今の数字に合わせる。移動はパイプライン全体の再走を起こす |
| LOW | naming | L1: 同じ 30 秒に名前が 2 つあり、同じ前置きの文字列が 3 か所に直書きされている。`directorWaitMarkInterval`(小文字)と `DirectorWaitMarkInterval`(大文字)は値が同じで、大文字の方は「status 側が読むために export しただけの別名」。どちらを変えても片方が残るので、直す人は両方に気づく必要がある。`directorWaitErrPrefix` は `org: wait --inbox --director: ` を 1 か所で持つのに、stderr に直接出す 3 つの文(`sayLost` の 2 つと `sayWriteFailed`)は `wait --inbox --director: ` を直書きしている。skill の「`wait --inbox --director: ` で始まる行」はこの 3 つに合わせて書かれているので、文面を変えると skill とずれる | `internal/org/director_wait.go:71-84`(2 つの定数)、`:351`(小文字側の使用)、`:88`(`directorWaitErrPrefix`)、`:643`、`:647`、`:657`(直書き)。`internal/cli/org_director_wait_line.go:68`、`:86`(大文字側の使用) | 定数は大文字の `DirectorWaitMarkInterval = 30 * time.Second` 1 つにして、`newDirectorWaitRun` の既定もそれを使う。stderr の 3 文は `directorWaitNoticePrefix = "wait --inbox --director: "` のような定数に寄せ、doc に「`/director` skill がこの前置きで行を見分ける」と 1 行書く |
| LOW | maintainability | L2: evidence に末尾の空白が 10 行ある。`git diff --check` が空でなくなり、前の PR の self-review は同じ確認が空だった。`wN stdout: ` の行で、stdout が空だったことを `echo "wN stdout: $(cat ...)"` が書いた結果。内容は空という事実なので、空白を落としても記録の意味は変わらない | `docs/evidence/director-wait-livefire-2026-10-19.md:117`、`:168`、`:190`、`:210`、`:233`、`:251`、`:260`、`:281`、`:323`、`:341`(`git diff --check e38b89a9...HEAD` の出力) | 10 行の末尾の空白を削るか、`wN stdout: (empty)` に直す。evidence は「記録だけの修正の例外」の対象(`docs/reports/` と insight event)に入らないので、直すなら M1 などほかの修正と同じコミット群で、パイプラインを回す |
| LOW | typo | L3: 出荷するコメントに、実装の段階の呼び名が残っている。`DirectorWaitEnd*` の定数の doc が「The CLI (S2 of the plan) maps them」と書く。S2 は計画の中の作業の区切りで、読む人には何のことか分からず、「これから入る」とも読める。マッピングの実体は 4 と 5 が `ExitCode`(`internal/cli/exitcode.go`)、signal が `runOrgWaitInboxDirector` の再送で、すでに入っている | `internal/org/director_wait.go:49-51` | 「The CLI maps them to exit codes (ExitCode for 4 and 5, a re-raised signal for signal): ...」のように、関数名で指す。コメントに S1 から S5 の呼び名を書かない |
| LOW | readability | L4: コメントの 2 か所が読みにくい。(a) `orgDirectorLine` の doc は、挿入のときに段落を組み直さず、元は 1 行だった「Attached and not attached」を `Attached` と `and not attached` の 2 行に割った。次の行が「end with what ...」と続くので、`and not attached` だけの短い行が段落の中に残る。(b) `gateSignalContext` の doc 末尾「It is signalCauseContext without the signal that arrived, which `ralph org wait --inbox --director` needs too.」の `which ... needs too` は、gate はその signal を要らないので「too」が何を指すのか分からない | `internal/cli/org_director.go:630-632`(組み直しの跡)。`internal/cli/org_gate.go:128-129` | (a) 段落を組み直す。(b) 「It is signalCauseContext without its second result, the signal that arrived, which only `ralph org wait --inbox --director` reads.」のように主語を言う |
| LOW | readability | L5: `director_wait.go` の冒頭の説明が、コードの 1 つの読み手を数えていない。「The mark is read by status and never to decide what to stop or start」と書くが、印は次の wait の `begin` も読み、前の印の `watch_accepting` で終了コード 4 を決める(D3 の引き継ぎ)。「止めるか起動するか」は見張りやプロセスのことなので文は成り立つが、読み手を status だけに見せる。この PR の核の 1 つが引き継ぎなので、冒頭の説明から落ちると、次に印を変える人が status しか見ない | `internal/org/director_wait.go:36-38`(説明)、`:389-403`(`begin` が前の印を読んで `wasAccepting` を決める) | 「The mark is read by status, and by the next wait of the same registration for the watch_accepting it carries over (D3); nothing reads it to start or stop a watch or a session.」のように読み手を 2 つ書く |
| LOW | maintainability | L6: plan の Rollout にある「release のノートに書く(新しい skill には新しい binary が要る)」の届け先がない。`.goreleaser.yml:36-39` は `^docs:`・`^test:`・`^chore:` を落とす。skill の 4 面、雛形 `director.md`、README、仕様の注は `docs:` の aba16ec2 と 61114ca4 に入り、`feat:` の 3 コミットには caveat がない。ノートは手書きで、PR と release を結ぶ場所がない(reviewer memory の pattern-chore-commit-release-note-handoff)。skill は template として binary と一緒に配られ、版が揃うのが普通なので、影響は手で skill だけを更新した project に限る。もう 1 点、aba16ec2 は `docs:` だが、出荷される headless director の雛形(`internal/org/prompts/director.md`)と Go のテスト 2 ファイルも含む | `.goreleaser.yml:36-39`。`git show --stat aba16ec2`(`internal/org/prompts/director.md`、`internal/org/prompts_test.go`、`internal/cli/org_director_check_line_test.go`)。`docs/plans/active/2026-10-19-org-director-wait.md` の Rollout or rollback notes | `/pr` の本文に caveat を書くのに加えて、release の issue に「`--director` を付ける新しい skill は新しい binary が要る。古い binary は `--director` を知らないフラグとして終了コード 1 で断る」の 1 行コメントを足す。push 済みのコミットの題は変えない |
| LOW | maintainability | L7: tech-debt の row 231 の (j) が、この PR で古くなったのに開いたまま残っている。(j) は「go.mod が 1.26 以上になったら、`gateSignalContext` とそのテスト 2 つを消して、呼び出しを `signal.NotifyContext` に戻せる」と払い方を書く。この PR で実体は `signalCauseContext` に移り、`gateSignalContext` はその薄い包みになり、`ralph org wait --inbox --director` が `received`(どの signal が来たか)のために `signalCauseContext` を直接使う。Go 1.26 の `NotifyContext` は cause に文字列の `signalError` を入れるだけで、signal そのものを返さない(手元の go1.26.0 の `os/signal/signal.go:281-300` で確認)。だから戻す手順は、gate・regate・merge の 3 つの呼び出しを戻したうえで、director の wait は `signalCauseContext` を残すか、cause の文字列から signal を読み直すことになり、row の「呼び出しを戻す」だけでは足りない。`signalCauseContext` 自体のテストも新しく入った(`TestOrgWaitDirector_SignalCauseContextWithNoSignalsRegistersNone`) | `docs/tech-debt/README.md:231`(Debt の (j))。`internal/cli/org_gate.go:110-180`(`gateSignalContext` と `signalCauseContext`)、`internal/cli/org_wait_director.go:64`(直接の使用)、`internal/cli/org_regate.go:144`、`internal/cli/org_merge.go:196`、`internal/cli/org_gate.go:98`(包み経由の 3 つ) | row 231 の (j) に 1 節足す: 「stage 8b-2b から `gateSignalContext` は `signalCauseContext` の包みで、director の wait は来た signal を `received` で読むので、`signal.NotifyContext` に戻しても `signalCauseContext` は残る」。row だけの変更で、コードは変えない |

## 確認して問題がなかったもの

- **印の持ち主。** 書き込みは `save()` の 1 か所と `begin()` の 1 か所で、どちらも `director-wait.lock` の flock(`withFileLock`)の中で印を読んでから `writeJSONFileAtomic` で書く。`owns()` は、ディスクの `wait_id` が自分のとき、または自分の書き込みがまだ 1 回も着いておらず(`written` が false)ディスクが `begin` の読んだ前の印(`prevWaitID`)のままのときだけ書く。`A が始める、B が始める、A の書き直し、B の終わり` をたどると、A の書き直しは `written` が true でディスクが B なので `lost` になり、B の終わりが残る。`begin` の書き込みが失敗した wait は、ディスクが前の印のままなら書き直せ、別の wait の印に替わっていれば `lost`。ロックが取れなかった `begin` は、ロックなしで前の印を読み(`decided` が false のとき)、書かない。壊れた印は `save()` の読みで誤りになり、1 回だけ言って待ち続ける。
- **遷移だけの 4。** `w.accepting` は「最後に settle した観測、または前の印から引き継いだ値」で、`probeAgain` が「受け付けていた」から「受け付けない」への変化を見つけると `noWatch` を立てて true を返し、`loop` が受信箱を読み直してから `endNoWatch` に進む。open の件が先に来れば `end=items` で `accepting` は true のまま残り、次の wait の `begin` が前の印から 4 にする。`endNoWatch` だけが `accepting = false` にする。始めたときから受け付けていない wait(前の印がない、別の登録、false)は `note:` を 1 行出し、あとで受け付けるようになれば `note:` と即時の書き直しを 1 回する。settle は受け付けていた wait だけが使い、`probe` が ctx で止まったときは `ok=false` で何も決めず、`begin` は前の値を保つ。
- **終了と時間切れ。** `loop` は `expired` を立ててからもう 1 回受信箱を読んでから時間切れにするので、締め切りの直前に届いた件を返す(`WaitInbox` と同じ)。`probeAgain` の settle が最大 2 秒 `--timeout-ms` を超えて遅らせることはあるが、ctx は見ているので signal では遅れない。`NewTicker` に渡す 3 つの間隔は、`newDirectorWaitRun` と `loop` の既定で 0 以下にならない。
- **`signalCauseContext` と `notIgnoredSignals`。** `stop` は `cancel(nil)` のあと goroutine の終わり(`ended`)を待つが、goroutine は `ctx.Done()` で必ず抜けるので詰まらない。`ctx` が最初から終わっているときは `ended` をその場で閉じる。signal を取った goroutine が `got` を mutex の下で書き、`received` も mutex の下で読む。`stop` の後は `got` が変わらない。gate、regate、merge の 3 つの呼び出しは `gateSignalContext` 経由で `received` を使わず、動きが変わらない。`signal.Ignored` が true を返すのは、始めたときに無視されていた SIGHUP と SIGINT だけで(`sigInstallGoHandler` が継承した無視を尊重するのは `_SIGHUP` と `_SIGINT` だけ)、SIGTERM は常に登録される。それより前に `signal.Notify` を呼ぶ非テストのコードはない(呼び出し元は gate、regate、merge、この wait だけで、別の verb)。signal が 0 個のときは `Notify` を呼ばない(呼ぶと SIGCHLD と SIGURG まで流れる)ことをテストが固定している。
- **再送。** `stop()` の後で `received()` を読み、signal があれば `signal.Reset` して自分に同じ signal を送る。`Notify` はそれ以前に止まっている。送れない、または終わらないとき(2 秒)は wait の誤りを返す。件を出力してから送るので、件は失われない。テストは、プロセス内では再送を記録する関数に差し替え、本物は `os.Args[0]` の子で確かめ、ロックを人が持つ間の 2 回目の SIGTERM でも終わらないこと、SIGHUP と SIGINT を `trap ''` で無視した子が終わらないことまで見ている。
- **`ExitCode` と `main.go`。** `errors.Is` の順は nil、drift(3)、4、5、既定 1 で、3 を 1 に落とす変更ではない。`main.go` は `errors` の import を落とし、エラーを先に stderr に出してから `os.Exit(cli.ExitCode(err))`。4 と 5 の sentinel を返すのは `WaitInboxForDirector` だけで、`--director` のない wait は返さない。
- **印の値の escape。** `Detail` は `end()` で `PrintableInboxText` を通してから書く。読み側の `orgDirectorWaitFacts`、`orgDirectorWaitLine`、`orgDirectorWaitClause` は `EndedAt`・`End`・`StartedAt`・`SeenAt`・読めなかった理由・`r.Registration.ID` を `inboxField` に通し、`PID` は `%d`。`directorWaitWatchState` は見張りの印から読んだ `State`・`TS`・`StoppedAt` を `PrintableInboxText` に通す。`directorWaitCheckCurrent` の `describe()` は attach 時刻を escape し、`cur.ID` は fold が `d<N>` を検査している。`HintStateDir` は `CommandWithStateDir` が `shellQuote` する。`--json` の `director.wait` は登録の値と同じく escape しない(doc に書いてある)。読めない印は誤りの文を `inboxField` に通して 1 行足し、status は失敗しない。
- **`openInboxItems` の抽出。** `WaitInbox` の読み(`Inbox.Read` と `InboxStateOpen` の絞り込み)と誤りの包み方(`org: wait --inbox: %w`)は変わらず、`open` が nil のときの扱いも同じ。`printInboxWaitItems` の抽出も、出力の書式と `return err` の位置が前と同じ。`--director` のない wait は 4 と 5 を返さず、印も書かず、見張りも読まない。`--director` を `--inbox` なしで付けると、設定も台帳も読む前に `org: wait: --director needs --inbox`。id が `d<N>` でなければ、`ResolveOrgStateDir`(git を呼びうる)より前に断る。
- **既存テストの変更。** 5 ファイルの変更は、期待する文字列に新しい `wait:` の行と `; no wait --inbox --director mark for it` の節を足しただけで、assert を弱めた箇所はない。
- **テストの安全。** 台帳はすべて `t.TempDir()` と `--state-dir`。herdr、ps、gh、network を打つ経路はない。signal は、`signalSelf`(先にテスト自身の channel で `Notify` する)と、テストが起こした `os.Args[0]` の子にだけ送る。`os.Chmod(f.dir, 0o555)` を使うテストは `os.Geteuid() == 0` で skip し、もう 1 つは「印の場所に directory を置く」形で root でも通る。`t.Parallel` はなく、パッケージ変数の差し替えは `t.Cleanup` で戻す。
- **evidence と register。** evidence 2 つに `$HOME` の値、token、pane の id、16 進の長い文字列はない(`wait_id` は pid と 19 桁の数字で、secret の scan の形を避けた設計)。`/tmp/cc-daemon-502/<…>` の 502 は uid で、socket のパスの残りは伏せてある。tech-debt の変更 5 行は cell が 5 つのままで、部分的に閉じた 247 行の説明は Debt の cell に書かれ、Trigger の cell も直してある。新しい行は 5 cell すべてを持つ。この PR が古くした open の行は、row 306(M1)と row 231 の (j)(L7)の 2 つ。register は `wait --inbox`・`gateSignalContext`・`signalCauseContext`・`WaitInbox`・`cmd/ralph/main.go`・「exit code 3」で grep して該当行を読んだが、全行は読んでいない。
- **コメントの plan 参照。** `docs/plans/active/2026-10-19-org-director-wait.md` が Go のコメント 4 か所と evidence 2 つに書かれ、`/pr` の archive は register だけを書き換える。前の PR のコメント(`director_check.go:17` など)は archive 後も `active/` のままで、先例があるので上げない。
- **関数の長さ。** `loop`(65 行)と `begin`(48 行)は 50 行の目安に近いかそれを超えるが、`WaitInbox` と同じ型で、`begin` の `decide` と `decided` は「ロックが取れなかったときも前の印を読む」ためのものとして読める。上げない。
- **ほか。** どの process でも `--director d1` を付けて wait を打てば印を奪えるが、plan の Assumptions と D5 が「印は wait が書いたことだけを表し、pid を出す」と決めている。`orgDirectorReportWait` は `DirectorWaitMark` の 10 field から 8 key への写しだが、`ralph status --json` と `director status --json` が本当に marshal し、key の一覧をテストが固定している(前の PR の `DirectorCheck` のような「テストしか読まない JSON の説明」ではない)。

## Positive notes

- 印の書き込みを 2 か所(`begin` と `save`)に絞り、「ディスクの `wait_id` が自分のとき、または自分の書き込みがまだ着いていないときの前の印」だけを持ち主の条件にした。実装と doc とテスト(`A が始める、B が始める、A の書き直し、B の終わり`、書けなかった最初の書き込み、書き込みの前に印が消えた場合)が同じ言葉で書かれ、読む人が 1 つずつ対応を追える。
- 「受け付けていた見張りが受け付けなくなったときだけ 4」を `accepting` の 1 つの変数と、`endNoWatch` だけが false を書く規則で表し、open の件や signal が先に来たときに変化を失わない作りにした。plan の D2 を実装に合わせて書き直した記録(a1f815fa)も残っている。
- 4 と `note:` の文を「lock と見張りの印が示すこと」だけにし、原因を言う語(crash、died、killed、panic など)をテストで禁じた。`directorWaitWatchState` は印の値をすべて `PrintableInboxText` に通す。
- signal の再送を「印を書く、件を出す、`Notify` を止める、同じ signal を自分に送る」の順に固定し、本物の再送を子のテストバイナリで確かめた。始めたときに無視されていた signal を登録しないことも、`trap ''` の子で確かめている。
- `ExitCode` を `cmd/ralph/main.go` の外の関数に出し、3 と 4 と 5 と 1 の表をテストで固定した。`main.go` は短くなった。

## Coverage gaps

- テストは走らせていない。`director_wait_test.go` の 135 から 600 行と 1000 行以降、`org_director_wait_line_test.go` の本体は読んでいない(関数名と grep のみ)。テストが各 AC の変異を捕まえるかは `/test` に任せた。
- skill、help、README、AGENTS.md、仕様の注の中身が実装と合うかは見ていない(`/verify` と `/sync-docs`)。1 点だけ `/verify` への引き継ぎ: help(`internal/cli/org.go`)と skill は「待つあいだに `--replace`・detach・stop で登録が変わると 5 で返る」と書くが、検知は `MarkInterval`(30 秒)ごとの読み直しの時点で、evidence の (e) は 30 秒後に 5 で返っている。help と skill にその間隔の言及はない。
- 実機の確認(`docs/evidence/director-wait-livefire-2026-10-19.md`)は文面を読んだだけで、再現していない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| row 306 の更新(M1): `internal/cli/org_director.go` は 831 行、新しい `internal/org/director_wait_test.go` は 1252 行、`internal/org/director_wait.go` は 736 行で、row の「794、800 まで 6 行」は古い | 次に director の表示や wait を変える変更が、800 行を超えるファイルの中で探すことになる | 動きを変えない移動でも全パイプラインの再走を起こす。row 306 は「次に `org_director.go` を 800 行より先に進める変更」を trigger にしており、この PR がそれ | 次に `org_director.go` か `director_wait.go` を変える計画。または今の cross-review の修正 cycle でほかの修正と一緒に | `docs/tech-debt/README.md:306`、`docs/plans/active/2026-10-19-org-director-wait.md`、このレポートの M1 |
| row 231 の (j) の更新(L7): `gateSignalContext` は `signalCauseContext` の包みになり、director の wait が `received` のために後者を直接使うので、「go.mod が 1.26 以上になったら呼び出しを `signal.NotifyContext` に戻せる」は gate・regate・merge の分にしか当たらない | go 1.26 に上げる変更が row の手順どおりに進めると、director の wait が来た signal を失う | row の文を書き足すだけで、コードは変わらない | go.mod の `go` 行を 1.26 以上にする変更 | `docs/tech-debt/README.md:231`、このレポートの L7 |

_(上の項目は、依頼により `docs/tech-debt/README.md` には反映していない。このレポート以外のファイルは変えないという指示のため。反映する場合は lead が決める。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件は、register の row 306 の trigger がこの PR で満たされたのに row が古いまま、かつ 1252 行の新しいテストファイルが増えた点(M1)。LOW 7 件は、同じ値の定数の 2 つの名前と直書きの前置き(L1)、evidence の末尾の空白(L2)、コメントに残った「S2 of the plan」(L3)、組み直していないコメント(L4)、冒頭の説明が引き継ぎの読み手を数えていない点(L5)、release のノートの届け先(L6)、row 231 の (j) の払い方が古くなった点(L7)で、いまの動きを壊すものはない。印の持ち主、4 の遷移と引き継ぎ、signal の再送、終了コード、escape、`openInboxItems` の抽出、テストの安全に欠陥は見つからなかった)
- Follow-ups:
  - M1 は、分割するか row 306 を今の数字に書き直す。分割はコードに触れるので、パイプライン全体を最初から回す。row だけの書き直しは、コード・skill・plan に触れず、`docs/tech-debt/README.md` だけの変更になる(記録だけの修正の例外の対象外なので、`/sync-docs` と cross-review の扱いは lead が決める)。L7 の row 231 の書き足しも同じ `docs/tech-debt/README.md` の変更で、M1 の row 306 とまとめて 1 コミットにできる
  - L1、L3、L4、L5 はコメントと定数の名前だけで、動きは変わらない。L2 は evidence の空白を削る。いずれもコードや evidence に触れるので、直すなら post-implementation pipeline を最初から回す(記録だけの修正の例外には入らない)
  - L6 は release の issue への 1 行コメントで、コードは変えない。`/pr` で本文に書くときに一緒に決める

## Fixes review (cycle 1)

- Date: 2026-10-19
- Range: `git diff 73a6310b..9f7a883e`(f0a4d04c、c1d31e3f、33da4185、9f7a883e の 4 コミット、17 ファイル)。この節より上の行は変えていない
- 見たもの: 直しの diff を全部、移動した本体の突き合わせ、直した場所の今の行、`git diff --check`。テストと静的解析は走らせていない(run-verify が別に走っている)

### 指摘ごとの結果

| 指摘 | 結果 | 根拠 |
| --- | --- | --- |
| M1 | fixed | `wc -l` で `internal/cli/org_director.go` 831 から 505、新しい `internal/cli/org_director_report.go` 342、`internal/org/director_wait_test.go` 1252 から 683、新しい `internal/org/director_wait_watch_test.go` 590。`director_wait.go` は 736 から 743 で、800 まで 57 行。row 306 は `docs/tech-debt/README.md:306` で数字と Trigger(`director_wait.go` が 800 を超える変更)と Related(plan とこのレポート)を直し、5 cell のまま。残る N1 は下 |
| L1 | fixed | `internal/org/director_wait.go:84` の `DirectorWaitMarkInterval` 1 つになり、既定は `:358`、読む側は `org_director_wait_line.go:68`、`:86` のまま。stderr の 3 文は `:95` の `directorWaitNoticePrefix` を `:650`、`:654`、`:664` で使い、doc(`:93-94`)に skill が前置きで行を見分けることが書いてある。`org: ` 付きの `directorWaitErrPrefix`(`:91`)は別の文面なので残してよい |
| L2 | fixed | `git diff --check e38b89a9...HEAD` が空。`docs/evidence/director-wait-livefire-2026-10-19.md` の 10 行は `wN stdout:` の末尾の空白だけが消え、冒頭の説明に「各行は colon の後に空白 1 つで終わっていたが、ここでは落とした」という注を足した。記録の意味は変わらない |
| L3 | fixed | `director_wait.go:53-56` が「The CLI maps them to exit codes」と書き、`ExitCode`(`internal/cli/exitcode.go`)と `runOrgWaitInboxDirector` の再送を名前で指す。`S2 of the plan` はない |
| L4 | fixed | (a) `internal/cli/org_director_report.go:128-152` の `orgDirectorLine` の doc は段落を組み直し、`and not attached` だけの短い行はない(移動した本文との差はこの 8 行から 7 行への組み直しだけ)。(b) `internal/cli/org_gate.go:128-129` が「It is signalCauseContext without its second result, the signal that arrived, which only `ralph org wait --inbox --director` reads.」になった |
| L5 | fixed | `director_wait.go:38-41` が読み手を 2 つ書く(status と、次の wait の `watch_accepting` の引き継ぎ(D3))。ついでに `:22-25` の doc も「30 秒ごとの読み直しで、次の読みで終わる」に合わせた |
| L6 | not fixed(先送りを妥当と見る) | リポジトリ内に届け先はない。PR がまだなく、release のノートの控えもまだ書けないので、いま直せる場所がない。道は妥当: `/pr` の本文に caveat を 1 行書き、merge の後に release の issue へ 1 行コメントを足す(`.goreleaser.yml:36-39` が `docs:`・`test:`・`chore:` を落とすので、コミットの題には載らない)。lead の記録の「release のノートに足すこと」にも 8b-2b の項目を足す。`/pr` で実行する |
| L7 | fixed(内容) | `docs/tech-debt/README.md:231` の (j) に、「stage 8b-2b から `gateSignalContext` は `signalCauseContext` の包みで、director の wait が来た signal を読むので、`signal.NotifyContext` に戻すのは gate・regate・merge の呼び出しだけ。`signalCauseContext` とその新しいテストは残る」が入った。残りは N2 |

### 動きを変えない移動か

- `internal/cli/org_director.go` から `org_director_report.go`: 73a6310b の `org_director.go` の移した範囲(324 行)と、新しいファイルの同じ範囲(322 行)を `diff` した。違いは `orgDirectorLine` の doc の組み直し(L4a)だけで、関数の本体、struct、tag、定数、escape の呼び出しは同じ。新しいファイルは冒頭に説明と import を足しただけ。`org_director.go` は `strconv` と `strings` の import を落とし、ほかの import は残り、新しいファイルの import と使用数が合う(使用数を数えた。コンパイルは `/verify` に任せた)
- `internal/org/director_wait_test.go` から `director_wait_watch_test.go`: 73a6310b のファイルの全行を sort して、新しい 2 ファイルの全行を sort した集合と `comm` で比べた。消えた行は 0、増えた行は `package`、`import` の括弧と 9 つの import、新しいファイルの冒頭のコメント 7 行だけ。テストの数は 25 から 12 と 13 で合う。fixture とヘルパは `director_wait_test.go` に残り、冒頭のコメントがそう言う
- `33da4185` は `org_director_check.go:17`、`org_director_wait_line.go:15`、`org_director_wake.go:19` のコメントだけ。`org_director.go` と書いたままの `org_director_headless.go:27`、`org_director_check.go:16`、`org_director_wake.go:17`、`org_director_report.go:19` は、今も `org_director.go` にあるもの(`newOrgDirectorAttachCmd` など)を指していて正しい
- `director_wait.go` の変更は定数 1 つにまとめたことと、コメント、前置きの定数化だけ。`DirectorWaitMarkInterval` の値は 30 秒のままで、`p.MarkInterval` の既定(`:358`)と読む側の `2*org.DirectorWaitMarkInterval` は変わらない。stderr の 3 文は、前置きの定数を足しても文字列が前と同じ(`wait --inbox --director: `)で、skill の「その前置きで始まる行」は保たれる
- help と skill: `internal/cli/org.go:1267` と `.claude/skills/director/SKILL.md:503-505` が「30 秒ごとの読み直しで、変わった後の最初の読みで 5」と書く。前の節の Coverage gaps に書いた `/verify` への引き継ぎ(間隔の言及がない)は解けた。skill の 4 面は `cmp` が一致

### 直しで増えた指摘

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 2 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | typo | N1: row 306 の Debt の cell の最後の文が途中で終わっている。「the new `internal/org/director_wait.go` is 743」で、単位の「lines」も句点もなく、すぐ次が `\|` の cell の区切り。row の前の部分は「(1252)」「(683)」のように数字だけを括弧に入れて書くが、この文だけは主語と数字が直接つながり、何の数字か読めない | `docs/tech-debt/README.md:306` の Debt の cell の末尾 | 「the new `internal/org/director_wait.go` is 743 lines.」にする。row の 1 文字の直しで、`docs/tech-debt/README.md` だけの変更 |
| LOW | maintainability | N2: row 231 の (j) が、直したあとも 2 つのテストを消せるものとして挙げている。(j) の前半は「`gateSignalContext` と `TestOrgGate_SignalContextParentCauseAndStop` と `TestOrgGate_SignalContextKeepsLaterSignalsCaught` は消せる」と言い、後半(今回足した分)は「`signalCauseContext` とその新しいテストは残る」と言う。しかし残る `signalCauseContext` の cause、親の cancel、`stop`、後から来た signal の扱いを直接確かめるのはこの 2 つのテストで(`gateSignalContext` 経由)、新しいテスト(`org_wait_director_test.go:547`)は「signal 0 個なら登録しない」だけを見る。director の wait の end-to-end のテスト(`:457`、`:485`、`:600`)は signal ごとの cause と 2 回目の SIGTERM を見るが、親が自前の cause を持つ場合(`:729` は `context.WithCancelCause` の親で `errParent` を確かめる。`:457` の親は `context.WithCancel` で cause は `context.Canceled` のまま)と、`stop` で cause が `context.Canceled` になること(`:729`)は見ない。(j) の手順どおりに消すと、この 2 点を押さえるテストが `signalCauseContext` から失われる | `docs/tech-debt/README.md:231`。`internal/cli/org_gate_test.go:729`、`:763`(2 つのテスト、`gateSignalContext` を呼ぶ)。`internal/cli/org_wait_director_test.go:457`、`:485`、`:547`、`:600` | (j) の前半を「2 つのテストは `gateSignalContext` ではなく `signalCauseContext` を呼ぶ形に書き換えて残す」に直す。row だけの変更で、コードは変えない。次に go.mod を 1.26 以上にする変更の中で払ってもよいが、その場合も row の文は今直す方が誤読が少ない |

どちらも、動きとテストに影響せず、`docs/tech-debt/README.md` の row だけに閉じる。N1 と N2 は同じファイルなので、直すなら 1 コミットにできる。

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。最初のレポートの MEDIUM 1 件(M1)と LOW 6 件(L1、L2、L3、L4、L5、L7)は直った。L6 は PR 本文と release の issue への 1 行コメントとして先送りし、その道は妥当。直しで増えた LOW 2 件(N1 の途中で終わった文、N2 の消せるテストの挙げ方)は row だけの文面の直しで、動きを壊すものはない。移動した 2 組のコードは動きを変えていない)
- Follow-ups:
  - N1 と N2 は `docs/tech-debt/README.md` の 2 行(306 と 231)の直しだけ。コード・skill・plan に触れないので、パイプラインの再走は要らないが、記録だけの修正の例外(`docs/reports/` と insight event だけ)の対象でもないので、どう扱うかは lead が決める。直すなら、`/pr` の前にその 1 コミットにする
  - L6 は `/pr` の本文に caveat を書き、merge のあとに release の issue へ 1 行を足す

## Cycle 2 review (8a33364b..7866785b)

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-19-org-director-wait.md`(パイプライン cycle 2。cross-review の cycle 1 の ACTION_REQUIRED 1 件の直しと、test の穴の払いの確認)
- Range: `git diff 8a33364b..HEAD`(38cac6f1、b3b692e1、7866785b の 3 コミット、8 ファイル、+375/-27)。HEAD 7866785b。この節より上の行は変えていない
- 見たもの: 直しの diff を全部。`internal/org/director_wait.go` の全文(789 行)。新しいテスト `director_wait_owner_test.go`(235 行)の全文と、`director_wait_test.go` のフィクスチャ(1 から 200 行)。`DirectorStore.Read`(`director.go:450-459`)と `withFileLock`(`lockfile.go:78-120`)。`internal/cli/org_wait_director.go` と `org.go:1255-1290` の help。plan の D1、D2、D3 とチェックリスト、tech-debt の 231、247、248、306、308 の行。cross-review の triage。cycle 1 の節(上)
- 走らせたもの(依頼で許された範囲だけ):
  - `go test ./internal/org/ -run 'DirectorWait|WaitInbox' -count=1 -timeout 30m -coverprofile=<scratchpad>` は ok(6.3 s)
  - 新しい 6 テストを `-race -count=8` で走らせて ok(4.6 s)
  - `go vet ./internal/org/ ./internal/cli/` は出力なし
  - `git diff --check 8a33364b..HEAD` は空。追加行に TODO、FIXME、debug の出力はなく、追加したコメント行で 90 桁を超えるものもない。`gofmt -l` は変更した Go 6 ファイルで空(読むだけ)
  - 全体のテストと静的解析は `/verify` と `/test` に任せた

### cycle 1 の指摘の扱い

| 指摘 | 結果 | 根拠 |
| --- | --- | --- |
| cross-review の ACTION_REQUIRED(始める前の確かめとロックのあいだの `attach --replace`) | fixed | 38cac6f1。`begin` がロックの中で `director.jsonl` を読み直し、`refresh` と同じ `directorWaitCheckCurrent(w.p.ID, &w.reg, ...)` で確かめる(`director_wait.go:437-450`)。今の登録でなければ `ErrDirectorWaitNotCurrent`、読めなければ prefix 付きの plain error を返し、どちらも `decide()` と書き込みの前に出る。triage が求めた「d2 の印がそのまま残り、d1 が 5 で返り、d2 がそのあとも書ける」を `TestWaitInboxForDirector_ARegistrationReplacedBeforeTheStartTakesTheLock` が固定する |
| test の穴 G1 から G5 と G7(tech-debt の 308 行) | fixed(G5 は下の I1) | b3b692e1。G1 は `..._AMarkRemovedWhileTheWaitRunsIsNotWrittenAgain`、G2 は `..._AStartThatCannotTakeTheLockStillCarriesOverTheWatch`、G3 は `..._ACancelDuringTheStartSettleWritesNoNote`、G4 は `..._TheDetailIsEscaped`、G5 は `TestDirectorWaitLine_TheWaitHelpSaysTheIntervalsOfTheOrgLayer`、G7 は `reraiseSignal` のコメント(`org_wait_director.go:119-127`)。名前はすべて存在する。G6 は 308 行に残した |
| self-review の N1、N2(row 306 の途中で終わった文、row 231 の (j)) | fixed(cycle 1 のうちに e7940546) | row 306 は「is 743 lines.」で終わり、row 231 の (j) は「2 つのテストは `signalCauseContext` に移す」と書く |
| self-review の L6(release のノートの届け先) | open のまま。`/pr` で実行 | リポジトリ内に届け先がない点は変わらない。cycle 1 の節のとおり、PR 本文の 1 行と merge 後の release の issue へのコメント |

### begin の直しの確認

- ロックの順: ロックの中で呼ぶ `DirectorStore.Read()` は lock を取らない(`director.go:450-452`「Read takes no lock」)ので、`director-wait.lock` の中に別の lock は入れ子にならない。`directorWaitLockFile` を取るのは `begin` と `save` だけ(grep)。`director.lock` を持ったまま `director-wait.lock` を待つ経路はない。
- 直列の議論: d2 の attach は d2 の wait の lock の前に着く。d1 が lock の中で「d1 が今の登録」と読めば、d2 の wait は d1 が手を放してから書くので d2 の印が後に残る。d1 が d2 の wait より後に lock を取れば、読み直しで d2 が見える。`attach` の 2 行(detached と attached)は 1 回の write なので、lock なしの `Read` が間の状態を見ることはない(`director.go:467-472`)。
- `begin` の出口: (1) 今の登録で書く、(2) 今の登録でなく、印も end も `note:` も書かずに 5、(3) 読めず、同じく何も書かずに plain error、(4) ロックが取れず、ロックなしで前の印を読んで引き継ぎ、失敗を 1 回言って続ける、(5) ctx が settle を切り、前の値を保つ。(2) と (3) は `decide()` を呼ぶ前に closure を出て、`w.end`、`sayWriteFailed`、`note:` の行も通らない。coverprofile で `begin` は 100%、`directorWaitCheckCurrent` も 100%。
- seam: `directorWaitBeforeBeginLock`(`:479`)を呼ぶのは `begin`(`:430-432`)だけで、非テストのコードに代入はない(grep)。nil のときの本番の動きは 2 行の nil 判定だけ。テストは `setDirectorWaitBeforeBeginLock` で入れて `t.Cleanup` で戻し、`t.Parallel` を使うテストは director の wait の 3 ファイルにない。seam を読む goroutine(d1 と d2)が走るのは、テストの冒頭の代入と `t.Cleanup` の復元のあいだだけで、`-race -count=8` で警告はなかった。
- エラーの扱い: `begin` が返す error を握りつぶす呼び出しはない。`begin` の呼び出しは `WaitInboxForDirector`(`:299`)と `director_wait_test.go` の 2 か所で、2 か所とも戻り値を確かめる。`readErr` の prefix は最初の読み(`:293`)と `refresh`(`:602`)と同じ `directorWaitErrPrefix` で、テストが `org: wait --inbox --director: org: director: read ` で始まることを確かめる。

### 新しいテストの確認

- 時間に頼る正の assert はない。待つ側は `waitForMark`、`waitForStderr`、`awaitDirectorWait` の 10 秒の上限つきの polling で、固定 sleep は 2 つ。G1 の 150 ms は「書き直されない」を見る負の窓で、遅い環境では見逃す側にしか倒れず、誤って落ちる側には倒れない。G3 の 100 ms は、cancel が `begin` の前に着いても `probe` の `select` が `ctx.Done()` を先に拾うので、同じ道を通る(sleep がなくても成り立つ)。
- goroutine の後始末: 競合のテストの d2 は `defer d2Cancel(nil)` で止まり、正常系は `awaitDirectorWait` で戻りを待ってから終わる。`t.Fatal` の失敗の道では d2 の goroutine が `t.TempDir` の掃除と重なりうるが、失敗の道だけ。
- 判別力: 競合のテストは `begin` の確かめを外すと d1 の印が残って落ちる。読めない `director.jsonl` のテストは、`readErr` の道を外すと `res.Current` が nil のまま `directorWaitCheckCurrent` に入り `ErrDirectorWaitNotCurrent` になるので、`errors.Is(..., ErrDirectorWaitNotCurrent)` の否定で落ちる。2 つの道が別々のテストで固定される。変異は走らせていない。読んだだけ。
- G1 から G4 は `docs/evidence/test-2026-10-19-org-director-wait-mutation.txt` の probe と同じ形で、引き継ぎ(`wasAccepting`)の判定、`!settled` の `note:` 抑止、`PrintableInboxText` の escape を 1 つずつ見る。

### 指摘

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 2 件、INFO 3 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1: `begin` が 70 行になり、リポジトリの 50 行の目安を超えた。cycle 1 は 48 行で「`WaitInbox` と同じ型」として上げなかった。この直しで 22 行増え、ロックの中の closure が外の 2 変数(`readErr`、`notCurrent`)に結果を置き、閉じたあとの `switch` で戻す形になった。違いは prefix を付けるか付けないかだけで、2 変数と `switch` は 1 つの「断られた」error で足りる。2 つの名前の型もそろっていない(`readErr` は error の語尾、`notCurrent` は名詞)。`director_wait.go` は 789 行で、row 306 の Trigger(800 行を超える変更)まで 11 行しかないので、次にここへ足す人が分ける羽目になる | `internal/org/director_wait.go:405-474`(`begin`)、`:433-456`(2 変数と `switch`)、`wc -l` 789。`docs/tech-debt/README.md:306` | closure の中で `refused = fmt.Errorf("%s%w", directorWaitErrPrefix, err)` と `refused = err` を入れ、閉じたあとは `if refused != nil { return refused }` にする(2 変数と `switch` が消え、4 行ほど減る)。もう 1 つの道は、ロックの中の読み、確かめ、書きを `w.writeStart(observed)` のようなメソッドに出すこと。どちらもコードの変更なので、cap の 2 回目の今は直さず、下の tech-debt の 1 行にする |
| LOW | readability | L2: この直しで、コメントが読む点を 2 つのまま数えている所が残った。(a) パッケージ冒頭(`:21-25`)は登録を「始める前と `DirectorWaitMarkInterval` ごとに読む」と書くが、今は始める前、印を書く直前のロックの中、30 秒ごとの 3 回。`WaitInboxForDirector` と `begin` の doc は直っていて、冒頭だけが残った。(b) `directorWaitCheckCurrent` の doc は「started is nil before the start」と書くが、`begin`(wait の start)は `&w.reg` を渡す。nil を渡すのは最初の読み(`:295`)だけで、「start」が wait 全体を指すのか `begin` を指すのか読み取れない。(c) 参考: `directorWaitLockFile` の doc の「A wait refused before it starts creates neither file」は最初の確かめには正しい。`begin` で断られた wait は空の lock ファイルを作るが、その旨は `WaitInboxForDirector` の doc(「the lock file may exist then」)にだけある | `internal/org/director_wait.go:21-25`、`:305-311`(doc)と `:443`(`&w.reg`)、`:295`(nil)、`:46-49`、`:241-243` | (a) 「read before it starts, again under the lock before it writes its mark, and again every ...」。(b) 「started is nil for the first read, before the wait has taken the lock」。(c) 「refused by the first read」と書くか、そのままでよい。コメントだけの変更だが、コードのファイルなので記録だけの修正の例外の対象外。cap の今は L1 と同じ 1 行にまとめる |
| INFO | test(`/test` に引き継ぐ) | I1: G5 を払うテストは定数と help を突き合わせるが、ゼロ値の既定が定数を使うこと(`newDirectorWaitRun` の既定 5 つ、`p.WatchProbeInterval = DirectorWaitWatchProbeInterval` の `:372` を含む)は coverprofile で 1 度も走らない。`:372` の右辺を別の値にしても全テストが通る。同じく、`begin` の `&w.reg`(`:443`)を nil にしても通る。attached_at が変わる場合は `directorWaitCheckCurrent` と `refresh` のテストにはあるが、`begin` のテストにはない。row 308 の「G1 から G5 がテストになった」は、定数と help の突き合わせとしては正しい | coverprofile の未走行ブロック `director_wait.go:368-382`、`:485-487`。`internal/org/director_wait_watch_test.go:375-379`(refresh)、`director_wait_test.go:330`(関数) | `/test` の判断に任せる。row 308 の G6 の一覧に「既定値の代入」を足してもよいが、`:372` は名前付きの定数を代入する 1 行なので、足さなくても危険は小さい |
| INFO | exception-handling | I2: ロックが取れなかった `begin` は読み直さずに続ける(`:457-459`)。このとき `decide()` がロックなしで読んだ前の印が `prevWaitID` になるので、`attach --replace` の競合と重なった場合に限り、30 秒後の `refresh` が書く `not_current` の end が `owns` を通って d2 の印を置き換えうる(ディスクが `prevWaitID` のままなら持ち主扱い)。cycle 1 の ACTION_REQUIRED と同じ結果が、2 つの失敗の重なりで残る。ただしロックが開けない(directory、権限)ときは `save` も同じロックを取れず書かない。残るのは 5 秒の timeout(別のプロセスが 5 秒ロックを持つ)と競合が重なる場合だけで、揃いにくい。`begin` の doc の「the first refresh reads the registration again」は書かない理由の説明としては足りるが、その refresh の end の書き込みには触れていない | `internal/org/director_wait.go:457-459`、`:677-688`(`owns`)、`:596-610`(`refresh`) | 上げない。直すなら、ロックが取れなかった `begin` は `prevWaitID` を空にして、前の印を置き換えられない状態で続ける |
| INFO | maintainability | I3: 計画の D2(`:66`)は `directorWaitMarkInterval`(小文字)のままで、cycle 2 は D3(`:70`)の定数名だけを `DirectorWaitWatchProbeInterval` に直した。D2 の名前は cycle 1 の L1 で 1 つの `DirectorWaitMarkInterval` になったもので、チェックリスト(`:169`)に記録がある。計画を直すと承認の digest が変わるので、記録で足りる | `docs/plans/active/2026-10-19-org-director-wait.md:66`、`:70`、`:169` | 上げない。archive される計画は、D2 の名前が実装と違うことをチェックリストで示している |

### 確認して問題がなかったもの

- 小文字の旧名: `directorWaitWatchProbeInterval` は `docs/reports/test-...md:145` と `sync-docs-...md:29`(cycle 1 の記録)にだけ残り、コード、help、計画の D3、tech-debt にはない。新しい名前の使用は定数の宣言、`DirectorWaitParams` の doc、既定の代入、CLI のテストの 4 か所で、どれも大文字側。
- tech-debt: row 306 と 308 は `|` が 6 本(5 cell)で、cell 内に生の `|` はない。row 306 の数字は `wc -l` と一致する(`director_wait.go` 789、`director_wait_test.go` 687、`director_wait_watch_test.go` 630、`director_wait_owner_test.go` 235、`org_director.go` 505、`org_director_report.go` 342、`director_start.go` 1203、`director_check_test.go` 857、`director_start_cleanup_test.go` 804、`herdr_test.go` 852、`director_stop.go` 782)。row 308 が挙げるテスト名 5 つと関数名 4 つ(`directorWaitItemsDetail`、`probeAgain`、`directorWaitWatchFacts`、`directorWaitWatchState`)は存在する。G6 の一覧は coverprofile の未走行ブロック(`:718`、`:577-579`、`:754-755`、`:771`)と合う。この直しで古くなった open の行はない(247、248 は `begin` を述べない)。
- 呼び出しの網羅: 直しが古くした「`begin` は登録を読み直さず書く」と述べる文は、`director_wait.go` の冒頭の 1 か所(L2)以外にない。`/director` skill、README、AGENTS.md、仕様の 8b-2b の注、help は「始める前に今の登録でなければ待たずに 5」までしか言わず、直しと食い違わない(書き足りるかは `/sync-docs`)。
- help のテスト: 3 つの期待文は `org.go:1262-1272` の help を空白で畳んだ文に含まれ、`strconv.FormatFloat` で「30 s」「5 s」を作る。既存の `TestDirectorWaitLine_HelpDescribesTheWait` は status 側の 30s だけを見るので、重複でなく別の面。テストの名前の前置きが `DirectorWaitLine_` なのは、30 秒を見る兄弟の隣に置いたため。
- `reraiseSignal` のコメント: 「stop は前でも後でもよい」は、`signal.Reset(sig)` が Notify を外してから送るので成り立つ。呼び出し側は `stop()` を先に呼ぶ(`:76`)ままで、コメントが言う条件は実装より弱いだけ。
- コミットの題: b3b692e1 は `test:` だが、定数の export とコメントを含む。export はテストが読むためで、差は小さい。上げない。

### Tech debt identified

最後の cycle(cap 2 の 2 回目)で直さない LOW は先送りになる。次の 1 行を `docs/tech-debt/README.md` に足すと、先送りが記録に残る(依頼により、レポート以外は変えていない)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small findings of the org-director-wait PR's cycle-2 self-review (LOW; none changes behavior). (a) L1: `begin` in `internal/org/director_wait.go` is 70 lines, over the 50-line target; the closure under `director-wait.lock` hands two errors out through `readErr` and `notCurrent` and a `switch` returns them, which a single `refused` error would do in fewer lines. (b) L2: the header of the file (`:21-25`) lists two reads of `director.jsonl` (before the start, every 30 s), not the three (the third is under the lock before the first write of the mark), and the doc of `directorWaitCheckCurrent` says "started is nil before the start" while `begin` passes `&w.reg` | A reader counts two reads and misses the one that guards the mark's first write; `begin` is the longest function of the file after `loop`, in a file 11 lines under 800 | Both are code-side changes found on the last pipeline cycle (cap 2); a change to `begin` or to a comment of the file restarts the post-implementation pipeline | The next change to `begin` or to the header of `internal/org/director_wait.go`, or the change that takes the file past 800 lines (row 306) | `docs/plans/active/2026-10-19-org-director-wait.md`; `docs/reports/self-review-2026-10-19-org-director-wait.md` (Cycle 2 review, L1, L2) |

### Recommendation

- Merge: yes(ブランチ全体、`main...HEAD`。CRITICAL・HIGH・MEDIUM なし。cycle 1 の MEDIUM 1 件(M1)と LOW 6 件(L1 から L5、L7)と N1・N2 は直ったままで、L6 は `/pr` で実行する。cycle 2 の直し(`begin` がロックの中で登録を読み直す)は、ロックの順、書かない 2 つの出口、seam の本番での nil、テストの時間の使い方のどれにも欠陥が見つからず、LOW 2 件(`begin` の長さ、冒頭のコメントの読む点の数)と INFO 3 件は動きを壊すものではない)
- Follow-ups:
  - L1 と L2 は、cap のまま進むなら上の 1 行を `docs/tech-debt/README.md` に足す(コミット前に lead が決める)。cap を上げて直すなら、L1 と L2 は 1 コミットで足り、パイプライン全体を最初から回す
  - I1 は `/test` に引き継ぐ。I2 と I3 は上げない
  - この節でレポート以外は変えていない。insight event(`self_review`、cycle 2、low 2)は `docs/insights/events/2026-10-19-org-director-wait.jsonl` に足したが、コミットはしていない
