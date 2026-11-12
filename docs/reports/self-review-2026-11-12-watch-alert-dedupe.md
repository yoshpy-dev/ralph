# Self-review report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(issue #307 の前半)
- Branch: `fix/watch-alert-dedupe`、base は `origin/main` の `028a8f6d`。レビューした HEAD は `72d7d6ad`。直前の `ec538952` が cycle 2 の cross-review の ACTION_REQUIRED 1 件(S10)の修正で、`72d7d6ad` はその記録(計画の Progress)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 3/3。cap は 3 に上げてあり(`cycle-count.json` の `cap`)、この cycle が最後の run。`cycle-count.json` の `cycle` は 2 のままなので、insight event は `--cycle 3` で足した
- Scope: `ec538952` の diff(`watch_scope.go`、`watch_scope_test.go`)を全部読み、triage の ACTION_REQUIRED が直ったかと、その diff が新しい問題を作っていないかを確かめた。全体の diff は前の 2 cycle で読んでいるので、今回は軽く見た(不可視 Unicode の走査、番号で triage を指す comment の grep、tech-debt の行と cap の突き合わせ)。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: 今回の指摘は `C3-1`、`C3-2`。前の run の ID(cycle 1 の `M1`、`L1` から `L6`、`N1` から `N5`、cycle 2 の `C2-1` から `C2-4`、`C2b-1`)は、tech-debt の行と計画が指すので「Earlier runs」にそのまま残した

## Evidence reviewed

triage の ACTION_REQUIRED の確認(`docs/reports/cross-review-triage-watch-alert-dedupe.md`、cycle 2、1 件)

| # | triage の指摘 | 直し(`ec538952`) | この run の確認 |
| --- | --- | --- | --- |
| 1 | `diff.relative=true` と下のディレクトリの cwd で、`git log` と予備の `git diff` が cwd からの相対名を返し、予約の判定を両向きに誤る | 2 つのコマンドに `--no-relative` を足した(`watch_scope.go:292`、`:294`)。`scopeCommittedPaths` の doc の段落を折り直して理由を 2 行足した。実際の git のテスト `TestWatch_ReservedScope_CommitPathsStayRootRelativeUnderDiffRelative` と、cwd を選べる fixture(`newReservedScopeFixtureAt`)を足した | 直っている。下の 3 点で確かめた |

1. git の挙動(git 2.49.0、`GIT_CONFIG_GLOBAL=/dev/null` の scratch repo、`diff.relative=true`、cwd は `internal/org`、2 つの commit が `internal/org/x.go` と `cmd/y.go` を触る)。コードと同じ引数の `git log --no-show-signature --cc --no-renames --name-only -z --format= HEAD~1..HEAD --` は `x.go` だけを返す(`cmd/y.go` が消え、`x.go` は相対名)。`--no-relative` を足したものも、`-c diff.relative=false` を前に付けたものも、`cmd/y.go` と `internal/org/x.go` の両方を返す。予備の `git diff --no-renames --name-only -z <a> <b>` でも同じ。`git status --porcelain=v1 -z --untracked-files=all` と `git ls-tree --full-tree` は、この cwd から root 相対で返る(`merge-base`、`rev-parse`、`symbolic-ref` はパスを出さない)。`internal/org` の本番コードでパスの一覧を取るほかの `git diff`(`regate_inway.go`、`regate_merge.go`)は `-C <root>` で動くので、cwd の影響を受けない
2. テスト。サブテストは 2 つ(`git log` の経路と、origin の ref が無い `git diff` の経路)。どちらも、cwd が `internal/org` の leader で、予約 `internal/org/` の中の commit では alert が立たず、予約の外の `cmd/y.go` の commit では 1 件だけ、`cmd/y.go` だけを名指しして立つ。`gitRun: o.realGitRun` なので本物の git を使い、`scopeGitRepo` が `isolateGitEnv` を呼ぶので利用者の gitconfig は結果に響かない。修正前のコードに 1 の挙動を当てはめると、最初の commit の確認で `x.go` が予約の外と読まれて失敗し(偽の alert)、最後の確認でも `cmd/y.go` が落ちて失敗する(見逃し)。両向きを別々の assertion で落とす形になっている。ここまでは読んで追っただけで、実行は `/test` に任せる
3. 固定文字列を pin する `TestWatch_ReservedScope_RunsOnlyTheGitCallsItNeeds` の 4 か所は、コード(`watch_scope.go:292`、`:294`)の引数の順と一致する。fixture の分割は、`newReservedScopeFixture`(16 か所から呼ばれる)が `newReservedScopeFixtureAt` に引数をそのまま渡す形で、変わったのは `p.Cwd` の元だけ。At 側に doc がある

そのほか見たもの

- 不可視 Unicode(U+0080-009F、U+2028/2029、ゼロ幅、方向制御、BOM、U+FFFD)の走査を、全体の diff(`git diff 028a8f6d...HEAD`)の追加行と `ec538952` の追加行にかけた。一致は無い。S10 の記録は、実装者が最初の 1 か所を python で書き換えたと書く(Edit を使う規則から外れる)。diff を読んだ範囲では、行頭の tab と空行は保たれている
- `C2b-1`(折り返された「(cross-review / finding 4)」)は `3bb1828b` で直っている。この PR が足した行に、番号で triage を指す comment(`cross-review finding N`、`finding [0-9]`、行末の `cross-review`)は残っていない。grep に出る 4 件(`watch.go:329`、`:997`、`watcher_test.go:78`、`:252`)は、この PR の diff に無い別の PR の指摘の番号
- tech-debt の「Eight gaps」の行(`docs/tech-debt/README.md:462`)を HEAD で読み直した。(7) と (8) は入っていて、`C2b-1` は open として載っていない(`3bb1828b` と合う)。cap の書き方に食い違いがある(`C3-2`)
- 80 桁を超える comment 行は、`ec538952` の前が `watch.go` 7、`watch_recurrence.go` 1、`watch_scope.go` 0 で、あとが 7、1、1。増えた 1 行は `watch_scope.go:283`(81 桁、`scopeCommittedPaths` の doc の折り直しで 1 文字あふれた)。ほかの 8 行は前の run の注のまま
- 見ていないもの: テストの実行結果、コンパイル、gofmt の整列。`git show --numstat ec538952` の `watch_scope.go` +16 / -14、`watch_scope_test.go` +62 / -5 を、行ごとに読んだ

## Earlier runs

cycle 1 は 3 回、cycle 2 は 2 回。ID は run ごとに分けてある。

### cycle 2 の 2 回目(commit d3a34921、HEAD `7ac1e5c1`)

CRITICAL・HIGH・MEDIUM なし、LOW 1。`C2-1` から `C2-4` のうち 3 件が直り、`C2-3` は 4 件中 3 件が直った。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| C2b-1 (LOW) | `AnEscalationAlreadyOnARecurredEventRecordsNothingNew` の doc が、折り返された行で「(cross-review / finding 4)」と triage を番号で指す。`C2-3` の指摘は行ごとの grep で数えたので、2 行にまたがるこの 1 か所を拾わなかった | `3bb1828b` で「found by the cycle 1 cross-review of plan 2026-11-12-watch-alert-dedupe, the escalation replay」に直った(cycle 3 の run で grep し直して確認)。解消 |

注として、`5393a316` が書き換えた comment の 4 行が 80 桁を超えること、`watch_recurrence.go:145-148` で「test」が 2 回出て読みにくいことを書いた。重さは付けていない。`3bb1828b` は `watch_test.go` の comment だけを変えたので、この 2 点は今も同じ(上の「そのほか見たもの」)。

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
| LOW | maintainability | C3-1: `ec538952` の `--no-relative` は git 2.28 以降の option で、同じ設定(`diff.relative`)にこの repo が使ってきた `-c diff.relative=false` と形が違う。それより古い git では `git log --no-relative` が引数の誤りで失敗する。呼び出し元は失敗を「commit のパスを取れなかった」と読み、前の HEAD を据え置いて stderr に何も出さないので、その git では commit からの `scope_change` が毎周期、黙って働かない。修正前は、`diff.relative` の設定が無い古い git でも commit のパスを取れていた | `docs/plans/archive/2026-09-25-secret-scan-git-config.md:49`(「scaffold 先の git は `--no-relative`(2.28 以降)より古いことがある。`-c` での上書きならどの版でも効く」)。`templates/base/scripts/secret-scan.sh:158` と `:330` と `templates/base/scripts/secret-scan-branch.sh:527` が `-c diff.relative=false` を使い、`secret-scan.sh:24-26` が「a -c key an older git does not know is ignored because that git has no such setting」と理由を書く。`internal/org` は、版で挙動が変わる所に版を書く(`regate_inway.go:92` の git 2.38)。ほかに `git branch --show-current`(2.22)を使うので、2.28 は `internal/org` に新しい、書かれていない下限になる。失敗の道は `watch_scope.go:296-298` の `err != nil` から `listed=false`、`:431` の `keepHead` へ。git 2.26.3(docker の `alpine:3.12`)で確かめた: `git log --no-relative --name-only -z --format= HEAD` は `fatal: unrecognized argument: --no-relative`(終了コード 128)、`git diff --no-relative ...` は usage を出して終了コード 129、`git -c diff.relative=false log ...` は終了コード 0。境界が 2.28 であることは上の計画の記述に頼った(2.26.3 と 2.49.0 の間の版は試していない) | 2 つのコマンドの先頭に、`--no-relative` の代わりに `-c diff.relative=false` を置く(`realGitRun` が `--no-optional-locks` を先に付けるので、`git --no-optional-locks -c diff.relative=false log ...` になり、global option が subcommand の前に来る)。`TestWatch_ReservedScope_RunsOnlyTheGitCallsItNeeds` の 4 つの固定文字列と、`scopeCommittedPaths` の doc の 2 か所を合わせて直す。実際の git のテストは、そのまま通る形。この cycle で直さないなら、「Eight gaps」の行に (9) として足す(案は下の表) |
| LOW | maintainability | C3-2: tech-debt の「Eight gaps」の行が、(6) と (8) を直さない理由を「cap 2 of 2」と書くが、cap は 3 に上がった。行を書いた `74e6b1db` のあとで、`7aee125e` の triage が cap を 3 に上げた。cycle 3 の run は別の指摘(`C3-1` の元になった `diff.relative`)に使われ、(6) と (8) はこの run でも直っていない。行は、直さない理由として今の事実(cap 3 の最後の run を `diff.relative` の指摘に使った)を言っていない | `docs/tech-debt/README.md:462` の (8) の「Why deferred」の「Like (6), they are tests after the pipeline's last run (cap 2 of 2)」と、2026-11-12 の cycle 2 の段落の「the tests and the doc line are code after the pipeline's cap of 2 runs, which the cross-review findings spent」。`.harness/state/standard-pipeline/cycle-count.json` は `"cap":3`。`git log -S'cap 2 of' -- docs/tech-debt/README.md` は `74e6b1db` だけを出す | 次の `/sync-docs`(この cycle で回し直される)が、2 か所を「cap 3 の最後の run を `diff.relative` の指摘(S10、`ec538952`)に使った」に直し、cycle 3 の段落を 1 つ足す(`ec538952` が `scopeCommittedPaths` と fixture を変え、(1) から (8) は直していない)。この review は register を編集していない |

CRITICAL、HIGH、MEDIUM はない。

注(重さを付けない): 80 桁を超える comment 行が、`watch_scope.go:283`(81 桁)に 1 行増えた。`C3-1` を直して `scopeCommittedPaths` の doc を触るときに、一緒に折り直せる。`watch.go` の 7 行と `watch_recurrence.go:148` は、前の run の注のまま。

## Positive notes

- 直しは、指摘の 2 つのコマンドに 1 つの flag を足す形で、本番の変更は引数の 2 行と doc だけ。呼び出しの流れと失敗の扱いは変えていない
- テストが実際の git で、両向きの誤り(偽の alert と見逃し)を別々の assertion で捕まえ、`git log` の経路と `git diff` の経路を別のサブテストにしている。org 自身の新しい計画の判定(`ls-tree --full-tree` を下のディレクトリから読む経路)も同じ cwd で通る。git の設定は `isolateGitEnv` で切り離してある
- `scopeCommittedPaths` の doc は、1 行足すだけでなく段落全体を折り直し、`--no-relative` の理由(cwd からの相対名になると、cwd の兄弟のパスが落ちる)を 2 行で書いた
- fixture の分割は、16 か所の呼び出しに触れずに、cwd を選べる入口を足した。S10 の記録に、python で書き換えた逸脱と、U+FFFD が無いことの確認が残っている
- `C2b-1` は、行ごとの grep で拾えなかった折り返しの形まで、`3bb1828b` が直している

## Tech debt identified

最後の run(cap 3/3)なので、`C3-1` を直さない場合、`C3-2` と合わせて deferral になる。この review は `docs/tech-debt/README.md` を編集していない(前の run と同じく、`/sync-docs` に渡す)。「Eight gaps」の行に足す内容の案:

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (9) `scopeCommittedPaths` の `--no-relative` は git 2.28 以降の option で、古い git では commit のパスを取る 2 つのコマンドが失敗し、`scope_change` が commit からは立たない(`C3-1`) | 2.28 未満の git で、予約を持つ org の commit 経由の `scope_change` が、stderr に何も出さずに働かない。作業中の変更からの検出は残る | cycle の cap(3 の最後の run を `diff.relative` の指摘に使った)。直すのは `.go` の編集で、pipeline を回し直す | 次の `scopeCommittedPaths` の変更。置き換えは 2 つのコマンドに `-c diff.relative=false` | この report の `C3-1`、`docs/plans/archive/2026-09-25-secret-scan-git-config.md`、`internal/org/watch_scope.go` |
| (6) と (8) の理由の書き直し(`C3-2`) | 行が「cap 2 of 2」と書くが、cap は 3 | 文書の編集(`/sync-docs`) | この cycle の `/sync-docs` | この report の `C3-2` |

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。cycle 2 の cross-review の ACTION_REQUIRED(`diff.relative`)は直っている。LOW が 2 件(`C3-1`、`C3-2`)。cycle 1 の `M1`、`L1` から `L6`、`N1` から `N4`、cycle 2 の `C2-1` から `C2-4`、`C2b-1` の扱いは変わらず、`N5` は受け入れのまま
- Follow-ups:
  - `C3-1` は、この cycle で直すか、(9) として残すかを選ぶ。直す場合は 2 つの引数と 4 つの固定文字列と doc で、`.go` の編集なので pipeline を回し直す(cap を上げる判断が要る)。(9) として残して進めると、2.28 未満の git では修正前より悪くなる(修正前は commit のパスを取れた)。2.28 以上の git では今の直しが正しく働く。2.28 未満の git が org の実行環境にあるかは、この review では確かめられなかった
  - `C3-2` は、この cycle の `/sync-docs` が register の 2 か所と cycle 3 の段落で直せる
  - `ec538952` は、この cycle の `/verify` と `/test` が回る前の木に入っている。cycle 2 の test の報告は `ec538952` より前なので、新しいテスト(`..._CommitPathsStayRootRelativeUnderDiffRelative`)は、この cycle の `/test` で初めて走る
  - 見ていないもの: テストの実行結果、コンパイル、gofmt の整列は `/verify` と `/test`
