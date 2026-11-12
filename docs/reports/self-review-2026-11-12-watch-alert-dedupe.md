# Self-review report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(issue #307 の前半)
- Branch: `fix/watch-alert-dedupe`、base は `origin/main` の `028a8f6d`。レビューした HEAD は `bfbf0a2a`。`e90c85da` が cycle 3 の 1 回目の `C3-1` の修正(S11)で、`bfbf0a2a` はその記録(計画の Progress)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 3/3 の 2 回目。cap は 3 に上げてあり(`cycle-count.json` の `cap`)、`cycle` は 2 のままなので、insight event は `--cycle 3` で足した
- Scope: `e90c85da` の diff(`watch_scope.go`、`watch_scope_test.go`)を全部読み、`C3-1` が直ったかと、その diff が新しい問題を作っていないかを確かめた。全体の diff は前の run で読んでいるので、今回は軽く見た(不可視 Unicode の走査、`--no-relative` の残り、git の fake の分岐の洗い出し)。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: 今回の指摘は `C3-2` の持ち越しだけで、新しい ID は無い。前の run の ID(cycle 1 の `M1`、`L1` から `L6`、`N1` から `N5`、cycle 2 の `C2-1` から `C2-4`、`C2b-1`、cycle 3 の 1 回目の `C3-1`、`C3-2`)は、tech-debt の行と計画が指すので「Earlier runs」にそのまま残した

## Evidence reviewed

`C3-1` の確認(`--no-relative` は git 2.28 以降にしか無く、古い git で commit の一覧が黙って失敗し続ける)

| # | 前の run の指摘 | 直し(`e90c85da`) | この run の確認 |
| --- | --- | --- | --- |
| C3-1 | `scopeCommittedPaths` の 2 つのコマンドの `--no-relative` を、この repo の慣例の `-c diff.relative=false` にする | 2 つの引数列の先頭を `-c`、`diff.relative=false` にした(`watch_scope.go:296`、`:298`)。doc の例と理由を直した。テストの fake が先頭の `-c` の組を飛ばして subcommand を見る `scopeSubcommand` を足し、4 つの固定文字列を新しい引数列に合わせた | 直っている。下の 4 点で確かめた |

1. 実際の git の挙動。本番の引数の順(`realGitRun` が `--no-optional-locks` を先に付けるので `git --no-optional-locks -c diff.relative=false log ...`)で、`diff.relative=true` を repo の local config に置き、cwd を `internal/org` にして、2 つの commit(`internal/org/x.go` と `cmd/y.go`)を触らせた。
   - git 2.49.0(この machine の git、`GIT_CONFIG_GLOBAL=/dev/null`)。`log --no-show-signature --cc --no-renames --name-only -z --format= HEAD~1..HEAD --` は `cmd/y.go` と `internal/org/x.go` の両方を返す。予備の `diff --no-renames --name-only -z HEAD~1 HEAD` も同じ。`-c` を外した対照は `x.go` だけを返す。`-c` が repo の local config に勝つことと、cwd の影響を消すことの両方が確かめられた
   - git 2.26.3(docker の `alpine:3.12`)。同じ引数の順で、`log` も `diff` も終了コード 0 で `cmd/y.go` と `internal/org/x.go` を返す。同じ git で `log --no-relative` は終了コード 128。前の run が 2.26.3 で出した結果と合う
   - 試していないもの: 2.26.3 と 2.49.0 の間の版、とくに境界の 2.28。doc の「git 2.28 から」は、`docs/plans/archive/2026-09-25-secret-scan-git-config.md:49` の記述と、`templates/base/scripts/secret-scan.sh:24-26`(「a -c key an older git does not know is ignored because that git has no such setting」)に頼っている
2. テストの fake の振り分け。`internal/org` の test で `gitRun` を差し替える所は `watch_scope_test.go` の 4 か所(`:854` の `failListing` の stub と、`scopeStubGit` を作る `:908`、`:1156`、`:1294`)だけで、どれも `scopeSubcommand` を通る。`scopeStubGit` の `answers` の中で argv を読むのは `args[len(args)-1]`(`:911`、`:1159`)で、先頭に 2 語足しても変わらない。`watch.go` と `watcher_test.go` に git の fake は無い。`git grep 'args\[0\]' -- internal/org` の残りは、merge、mergeorder、spawn、herdr、feature の別の fake で、watch の commit の一覧には届かない
3. `scopeSubcommand` の境界。`for i+2 < len(args) && args[i] == "-c"` は、`-c <kv>` のあとに 1 語以上あるときだけ 2 つ進む。`["-c", "k=v"]` だけの argv は `"-c"` を返して panic しない。空の argv は `args[0]` で panic するが、前の `args[0]` と同じで、呼ぶ側に空の argv は無い。`commands()` は `strings.Fields` した行を同じ関数に通す
4. 固定文字列の pin(`TestWatch_ReservedScope_RunsOnlyTheGitCallsItNeeds` の 4 か所、`:1215`、`:1250`、`:1265`、`:1277`)は、コードの引数の順と一致する。`--no-relative` は、コード、テスト、tech-debt に残っていない。残る所は、選んだ理由を書く `watch_scope.go:291` の comment と、計画の Progress(S10 の履歴と S11 の記録)

そのほか見たもの

- 不可視 Unicode(U+0080-009F、U+2028/2029、ゼロ幅、方向制御、BOM、U+FFFD)の走査を `e90c85da` の追加行にかけた。一致は無い
- 80 桁を超える comment 行は、`watch_scope.go` で前の run の 1 行(`:283`、81 桁)から 0 行になった。`scopeCommittedPaths` の doc を折り直したときに直った。`watch.go` の 7 行と `watch_recurrence.go:148` は前の run のまま
- doc の文の正しさ。「an older git fails both commands on it, listed is false, and the previous HEAD is kept for ever with nothing on stderr」は、前の run が読んだ失敗の道(`err != nil` から `listed=false`、`keepHead`)と合う。「A git that old has no diff.relative setting, so the override changes nothing there」は 1 の根拠のとおり
- `bfbf0a2a`(計画の Progress の 1 行)は、`C3-1` の内容、直した commit(`e90c85da`)、`C3-2` が `/sync-docs` の担当であることを書く。変異で 2 つのコマンドがそれぞれのテストで落ちたという記述は、この review では再現していない(テストを走らせないため)
- 見ていないもの: テストの実行結果、コンパイル、gofmt の整列。`git show --numstat e90c85da` の `watch_scope.go` +17 / -13、`watch_scope_test.go` +21 / -9 を、行ごとに読んだ

## Earlier runs

cycle 1 は 3 回、cycle 2 は 2 回、cycle 3 は 1 回(この run の前)。ID は run ごとに分けてある。

### cycle 3 の 1 回目(commit 65a82b01、HEAD `72d7d6ad`)

CRITICAL・HIGH・MEDIUM なし、LOW 2。cycle 2 の cross-review の ACTION_REQUIRED(`diff.relative=true` と下のディレクトリの cwd で、commit のパスが cwd からの相対になる)は `ec538952` で直っていることを確かめた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| C3-1 (LOW) | `ec538952` の `--no-relative` は git 2.28 以降の option で、同じ設定にこの repo が使ってきた `-c diff.relative=false` と形が違う。古い git では 2 つのコマンドが引数の誤りで失敗し、呼び出し元は「commit のパスを取れなかった」と読んで前の HEAD を据え置き、stderr に何も出さないので、`scope_change` が commit からは毎周期、黙って働かない。git 2.26.3 で `log --no-relative` が終了コード 128、`-c diff.relative=false` が 0 | `e90c85da` で `-c diff.relative=false` に替えた。この run で解消を確認(上の Evidence) |
| C3-2 (LOW) | tech-debt の「Eight gaps」の行が、(6) と (8) を直さない理由を「cap 2 of 2」と書くが、cap は 3 に上がった(行を書いた `74e6b1db` のあとで `7aee125e` が上げた) | 未処理。`/sync-docs` の担当。この run でも行は変わっていない(下の Findings) |

### cycle 2 の 2 回目(commit d3a34921、HEAD `7ac1e5c1`)

CRITICAL・HIGH・MEDIUM なし、LOW 1。`C2-1` から `C2-4` のうち 3 件が直り、`C2-3` は 4 件中 3 件が直った。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| C2b-1 (LOW) | `AnEscalationAlreadyOnARecurredEventRecordsNothingNew` の doc が、折り返された行で「(cross-review / finding 4)」と triage を番号で指す。`C2-3` の指摘は行ごとの grep で数えたので、2 行にまたがるこの 1 か所を拾わなかった | `3bb1828b` で「found by the cycle 1 cross-review of plan 2026-11-12-watch-alert-dedupe, the escalation replay」に直った(cycle 3 の run で grep し直して確認)。解消 |

注として、`5393a316` が書き換えた comment の 4 行が 80 桁を超えること、`watch_recurrence.go:145-148` で「test」が 2 回出て読みにくいことを書いた。重さは付けていない。`3bb1828b` は `watch_test.go` の comment だけを変えたので、この 2 点は今も同じ。

### cycle 2 の 1 回目(commit f853d973、HEAD `d3a63398`)

CRITICAL・HIGH・MEDIUM なし、LOW 4。cross-review の ACTION_REQUIRED 4 件(`907afde3`)は、次のとおり確かめて、4 件とも直っていた。

| 番号 | triage の指摘 | 確認 |
| --- | --- | --- |
| 1 | leader の活動が初めて見える周期に再発した firing を古い pending に畳むと、同じ周期の `checkDeadman` がその pending を消して firing が失われる | `leaderActedSinceAlert` を取り出して共有し、`foldIntoPendingAlert` が畳む前に同じ判定を使う。直っている |
| 2 | merge commit の `git log --name-only` が、merge の中での衝突解決のパスを出さない | `--cc` を足した。git 2.49.0 の scratch repo の 7 つの形(新しい内容で解決、解決と別の編集、upstream の内容、`--ours`、衝突なし、merge のあとの通常 commit、rebase の形)で、doc の文と合うことを確かめた。直っている |
| 3 | `origin/HEAD` が消えた ref を指すと、`git log ... --not <ref>` が毎周期失敗して前の HEAD が据え置かれる | `namesCommit` で指す先を確かめ、通らなければ main、master、最後に `git diff` に落とす。直っている。残りの穴(M48)は tech-debt の (6) |
| 4 | recurred を足したあと status を保存する前に止まると、立て直した watch が 2 回数える | escalate の打ち直しは `alertItemWithRecurredID` で直った。立ち上がり側に足した同じ確認は本番では当たらなかった(`C2-1`) |

| ID | 指摘 | 処理 |
| --- | --- | --- |
| C2-1 (LOW) | `recordRecurrence` の alert_id の確認は、周期ごとに変わる `now` から alert_id を作るので、止めた時計のテストだけで当たる。doc は「status を保存する前に止まった watch の再実行を見分ける」と書いていた | `5393a316` で確認とテストを消し、doc を escalate の打ち直しだけに合わせた。解消。残る窓は tech-debt の (7) |
| C2-2 (LOW) | 活動の判定を `leaderActedSinceAlert` に移したあとも、`checkDeadman` のものとして書いた comment が 4 か所に残った | `5393a316` で 4 か所を直した。解消 |
| C2-3 (LOW) | 4 つのテストの comment が、`/cross-review` がその場で書き直す triage を、番号(「cross-review finding N」)だけで指す | 4 件のうち 3 件を直し、1 件は C2-1 で消えた。折り返された 1 件が残った(`C2b-1`、`3bb1828b` で直った) |
| C2-4 (LOW) | `foldIntoPendingAlert` の引数に string が 5 つ並び、`orgID` と `org` が近い | `5393a316` で署名を 4 つに減らした。解消 |

### cycle 1 の 1 回目(commit 27a76047、HEAD `0b2c0bce`)

CRITICAL・HIGH なし、MEDIUM 1、LOW 6。`f4f1fb93` と `880a3fa0` で処理し、2 回目で現物を読んで確かめた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| M1 (MEDIUM) | `watchPendingAlert.LastRecurredTS` が書くだけで読まれず、件の「最後の時刻」が escalate の時刻になる | 欄を消し、コメントを実装に合わせた。解消。残り: `show` の `(last <ts>)` が pending のあいだの再発では escalate の時刻になることを、利用者に見える所に書いていない(`/sync-docs` に渡した) |
| L1 | commit の一覧の失敗で `ScopeHead` を進め、その周期の commit パスを失う | 失敗かつ git が前の HEAD を持つなら据え置く(`scopeHeadKnown`)。解消。残り: 失敗が続くと stderr に何も出ない。porcelain の失敗を黙って飛ばす既存の作りと揃っている |
| L2 | 署名の差で、パスが減ったときと `git add` でも立つ | 「前の周期に無かったパスか、予約の外を触る commit」に変更。解消 |
| L3 | 計画の Risks は「HEAD が変わった周期だけ」git を打つとしたが、実装は候補の記録がある周期ごとに打つ | 受け入れ。理由は計画の Progress(`880a3fa0`)。本文を変えると承認の digest が変わる |
| L4 | `watch.go` の再発の仕組み約 180 行が 1 つのまとまり | `watch_recurrence.go` に移した。解消 |
| L5 | `busySeats` は `len() > 0` でしか読まれない | `anotherSeatWorking bool` にした。解消 |
| L6 | `GitStatus` と `GitRun` の 2 つの注入口 | 受け入れ。doc が使い分けを書いている |

### cycle 1 の 2 回目(commit 6025f3cb、HEAD `880a3fa0`)

CRITICAL・HIGH・MEDIUM なし、LOW 4。`eb596f49` で 4 件とも直し、3 回目で確かめた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| N1 (LOW) | `printableScopePath` の doc コメントに U+2028 の文字そのものが入り、escape の例が欠けて見える | バックスラッシュと u2028 の 6 文字の escape にした。解消 |
| N2 (LOW) | `ScopeSeen` の key が `scope_outside_paths_seen` で欄名と食い違い、コメントが main に無い build を指す | 欄を `ScopeOutsidePathsSeen` に改名し、コメントに前の形と今の扱いを書いた。解消 |
| N3 (LOW) | `committedOutside` の項が何を足すか、doc にも計画にも書かれていない | `evaluateReservedScope` の doc に理由を足した。足した文の言い過ぎが N5 |
| N4 (LOW) | `recurred: 1 times (last ...)` | 回数 1 のとき `1 time`。解消 |

### cycle 1 の 3 回目(commit 690e7235、HEAD `eb596f49`)

CRITICAL・HIGH・MEDIUM なし、LOW 1。N1 から N4 はすべて直っていた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| N5 (LOW) | N3 の直しで足した doc が、commit の項の効き方を言い過ぎている。「commit は、報告済みのパスでも条件を立てる。ALERT の件が open なら、その件の再発になる」と読めるが、`raiseOrClear` は `rec.Active` のあいだ何もしないので、ALERT を出した直後の周期の commit は再発にも ALERT にもならない。再発になるのは、間に静かな周期が入って `Active` が落ちたあとの commit だけ | 挙動は変えず、tech-debt の「Eight gaps」の行の (5) に記録した。doc の文は今も変わっていない(`watch_scope.go` の `evaluateReservedScope`、`which the director should see` の行)。cycle 3 でも未修正のまま(受け入れ) |

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C3-2(持ち越し、前の run の指摘): tech-debt の「Eight gaps」の行が、(6) と (8) を直さない理由を「cap 2 of 2」と書くが、cap は 3 に上がった。cycle 3 の cap の最後の run は `diff.relative` の指摘(S10 と S11)に使われ、(6) と (8) はこの run でも直っていない。行は、直さない理由として今の事実を言っていない。この run で `e90c85da` と `bfbf0a2a` は `docs/tech-debt/README.md` を触っていない | `docs/tech-debt/README.md:462` の (8) の「Why deferred」の「Like (6), they are tests after the pipeline's last run (cap 2 of 2)」と、2026-11-12 の cycle 2 の段落の「the tests and the doc line are code after the pipeline's cap of 2 runs, which the cross-review findings spent」。`.harness/state/standard-pipeline/cycle-count.json` は `"cap":3`。`git grep -n 'cap 2 of' -- docs/tech-debt/README.md` は 462 行目だけを出す | この cycle の `/sync-docs` が、2 か所を「cap 3 の最後の run を `diff.relative` の指摘(S10 の `ec538952`、S11 の `e90c85da`)に使った」に直し、cycle 3 の段落を 1 つ足す(`ec538952` と `e90c85da` が `scopeCommittedPaths` と fixture と test の fake を変え、(1) から (8) は直していない)。この review は register を編集していない |

CRITICAL、HIGH、MEDIUM はない。`e90c85da` に新しい指摘は無い。`C3-1` は解消した。前の run の注(`watch_scope.go:283` の 81 桁)も、doc の折り直しで消えた。

## Positive notes

- 直しは、指摘のとおり 2 つのコマンドの先頭に `-c diff.relative=false` を置く形で、本番の変更は引数の 2 行と doc だけ。呼び出しの流れと失敗の扱いは変えていない。この repo が `templates/base/scripts/secret-scan.sh` などで使ってきた形に揃った
- doc が、`--no-relative` ではなく `-c` にした理由(2.28 未満での失敗と、前の HEAD が据え置かれたまま stderr に何も出ないこと)と、古い git では上書きが何も変えないことを書いたので、次に誰かが `--no-relative` に戻したくなっても理由が読める
- テストの fake は、`-c` の組を飛ばして subcommand を見る 1 つの関数(`scopeSubcommand`)に寄せた。`gitRun` を差し替える 4 か所すべてが通る。固定文字列の pin は `-c` の位置まで合わせてある。実際の git のテスト(`..._CommitPathsStayRootRelativeUnderDiffRelative`)は、引数の形に依らず `diff.relative=true` の下の結果で `-c` の効きを見る
- 前の run で出した 2 つの probe(2.49.0 と 2.26.3)を、本番の引数の順そのままで取り直して、両方で通った

## Tech debt identified

この run で足す行は無い。`C3-2` は `/sync-docs` が register の 2 か所と cycle 3 の段落で直す。この review は `docs/tech-debt/README.md` を編集していない(前の run と同じく、`/sync-docs` に渡す)。最後の run(cap 3/3)なので、`/sync-docs` が直さなければ `C3-2` は deferral になる。`C3-1` は直ったので、前の run が案に出した (9) の行は要らない。

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。`C3-1` は直っている(実 git 2.49.0 と 2.26.3 で確かめた)。LOW が 1 件(`C3-2`、`/sync-docs` の担当)。cycle 1 の `M1`、`L1` から `L6`、`N1` から `N4`、cycle 2 の `C2-1` から `C2-4`、`C2b-1` の扱いは変わらず、`N5` は受け入れのまま
- Follow-ups:
  - `C3-2` は、この cycle の `/sync-docs` が register の 2 か所と cycle 3 の段落で直せる
  - `e90c85da` は、この cycle の `/verify` と `/test` が回る前の木に入っている。新しいテスト(`..._CommitPathsStayRootRelativeUnderDiffRelative`)と、`scopeSubcommand` を通すようにした 4 つの fake のテストは、この cycle の `/test` で走る。`bfbf0a2a` が書く、変異で 2 つのコマンドがそれぞれのテストで落ちるという確認は、この review では再現していない
  - 2.28 未満の git が org の実行環境にあるかは、この review では確かめられなかった。直しは、2.28 未満でも 2.28 以上でも commit のパスを取れる形
  - 見ていないもの: テストの実行結果、コンパイル、gofmt の整列は `/verify` と `/test`
