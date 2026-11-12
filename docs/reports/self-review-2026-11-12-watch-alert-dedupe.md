# Self-review report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(issue #307 の前半)
- Branch: `fix/watch-alert-dedupe`、base は `origin/main` の `028a8f6d`。レビューした HEAD は `7ac1e5c1`。直前の `5393a316` が cycle 2 の 1 回目の LOW 4 件(`C2-1` から `C2-4`)の修正で、`7ac1e5c1` はその記録(計画の Progress の S9)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 2 の 2 回目。`cycle-count.json` は cycle 2、`[standard].max_pipeline_cycles` は 2 なので、この cycle が最後の run
- Scope: `5393a316` の diff(4 ファイル、`watch.go`、`watch_recurrence.go`、`watch_test.go`、`watch_scope_test.go`)を全部読み、`C2-1` から `C2-4` が直ったかと、その diff が新しい問題を作っていないかを確かめた。cycle 2 の 1 回目に全体を読んでいるので、今回は直した周りだけを見た(`watch_recurrence.go` の全文、`watch.go` の `checkDeadman` と `leaderActedSinceAlert` と `escalateToInbox`)。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: 今回の指摘は `C2b-1`。cycle 2 の 1 回目の `C2-1` から `C2-4` は、計画の Progress(S9)が指すので下の「Earlier runs」にそのまま残した。cycle 1 の ID(`M1`、`L1` から `L6`、`N1` から `N5`)も、tech-debt の行と計画が指すので残した

## Evidence reviewed

`C2-1` から `C2-4` の確認

| ID | 直し(`5393a316`) | この run の確認 |
| --- | --- | --- |
| C2-1 | `recordRecurrence` の「firing 自身の alert_id が recurred event に載っているか」の確認と、そのテスト(`TestWatch_Recurrence_AFiringAlreadyOnARecurredEventIsNotAppendedAgain`)を消した。`alertItemWithRecurredID` の doc を escalate の打ち直しだけに合わせ、`recordRecurrence` の doc から「すでに記録済み」の項を消した | 直っている。本番の挙動は変わらない(消した確認は、周期ごとに変わる `now` から作る alert_id には当たらなかった)。`alertItemWithRecurredID` の呼び出しは `escalateToInbox`(`watch.go:1875`)に 1 つ残り、これは status に残る pending の alert_id なので立て直しても当たる。そのテスト(`watch_test.go:3698`)と、関数自体のテスト(`:3873` 付近)も残っている。`recordRecurrence` の doc は recurred event の alert_id を `watchAlertID` の新しい値と書き、「記録済み」を言っていない。`watchAlertID` の doc(`watch.go:336`)の「recordRecurrence the recurred event of a firing it folds」も合う。残る窓(recurred を足してから status を保存するまでに止まる)は、計画の Progress の S9 に書いてある。tech-debt の行には、まだ入っていない |
| C2-2 | `leaderActivityEventCount` の doc の 2 か所と `historyLeaderLineCount` の doc の 1 か所の `checkDeadman` を `leaderActedSinceAlert` に直した。`checkDeadman` の doc から番兵の理由(PR④ known gap #5)の段落を `leaderActedSinceAlert` の doc に移し、`checkDeadman` の doc には「判定は `leaderActedSinceAlert`、`foldIntoPendingAlert` と共有」と書いた。helper の doc の `recordRecurrence` を `foldIntoPendingAlert` にした | 直っている。移した段落は、前の `checkDeadman` の doc と 14 行すべて同じ文面で、新しい場所に入った(コメントの `//` を外して `diff` した)。`#2`、`#3` の番号は、helper の doc の冒頭が並べる順(manifest、probe、history)と合う。`watch.go` に残る `checkDeadman` は 10 か所で、どれも `checkDeadman` 自身の挙動を言っている(`:1030`、`:1059`、`:1082`、`:1523` の「clears an alert only when there is activity and the subject is not the leader」、`:1690`、`:1701`、`:1740`、`:1780`、`:1783`、`:2037`)。取り残しは無い |
| C2-3 | 4 つのテストの comment の「(cross-review finding N)」を、「the case the cycle 1 cross-review of plan 2026-11-12-watch-alert-dedupe found」に直した(`watch_test.go:3540`、`watch_scope_test.go:649`、`:679`)。`:3738` のテストは C2-1 で消えた | 4 件中 3 件は直った。残り 1 件は、同じ形の指摘が折り返された行にあり、最初の指摘も直しも拾っていない(`C2b-1`) |
| C2-4 | `foldIntoPendingAlert` の署名を `(status, orgID, seatID, condType)` にし、`key` は中で `conditionKey` から作り、stderr 用の escape は manifest が読めないときの分岐の中だけにした | 直っている。呼び出しは `recordRecurrence`(`watch_recurrence.go:115`)の 1 か所で、ほかに無い。`recordRecurrence` の後ろの 3 つの stderr 行は、先頭で作った `org, seat` をそのまま使う。`key` の導出が `recordRecurrence` と `foldIntoPendingAlert` で 2 回あるが、同じ `conditionKey` 呼び出しで、ずれる余地は無い |

そのほか見たもの

- 直しの diff は `git show --stat 5393a316` で 4 ファイル、`+59 / -91`。`watch_recurrence.go` の import(`errors`、`fmt`、`slices`、`strings`、`time`、`protocol`)は、残った本文がどれも使う。`recordRecurrence` の `key`、`now`、`org`、`seat` は、消した確認のあとも使う箇所が残っている
- 消したテストが使っていた helper(`escalateAlertItem`、`mustReadInbox`、`watchdogAlerts`、`newWatchInboxFixture`)は、ほかのテストが呼ぶ(`grep -c` で 10、90、44、19 件)。`reflect.DeepEqual` も 8 件残る。消したテスト名は、リポジトリの `.go` と `.md` のどこにも残らない(この report の前の版だけが書いていた)
- S9 の記録によると、テストの削除は python で行われた(Edit ではない)。差分は 32 行の削除だけで、削除の前後の空行(`}` / 空行 / `// AC12`)が保たれる(diff の文脈行で確認)。`5393a316` の追加行に、不可視 Unicode(U+0080-009F、U+2028/2029、ゼロ幅、方向制御、BOM、U+FFFD)の一致は無い
- `foldIntoPendingAlert` の `read` / `events` の持ち方、ループの止まり方は変わっていない。`5393a316` が触ったのは引数と stderr 行の escape の位置だけ
- `5393a316` の commit message は subject のみで body が無い。`refactor:` の中で本番の分岐を 1 つ消しているが、消した分岐が本番で当たらなかったことは S9 の記録と `C2-1` の指摘に残っている
- 全体の diff(`git diff 028a8f6d...HEAD`)の追加行に、不可視 Unicode の走査をかけた。一致は 1 行で、`f853d973` が書いた前の版の report の N1 の行(`docs/reports/self-review-2026-11-12-watch-alert-dedupe.md:45`)に、U+2028 の文字そのものが入っていた。N1 が指摘した形と同じで、`0c727338` が直したあとに、cycle 2 の 1 回目の書き直しで戻っていた。この版では、バックスラッシュと u2028 の 6 文字の escape として書いた(書いたあとにもう一度走査した)。Go のコードと comment には無い
- 見ていないもの: テストの実行結果、コンパイル、gofmt の整列(python で編集したテストファイルを含む)。comment の折り返しだけは目で数えた(`C2b-1` の下の注)

## Earlier runs

cycle 1 は 3 回、cycle 2 の 1 回目が 1 回。ID は run ごとに分けてある。

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
| C2-1 (LOW) | `recordRecurrence` の alert_id の確認は、周期ごとに変わる `now` から alert_id を作るので、止めた時計のテストだけで当たる。doc は「status を保存する前に止まった watch の再実行を見分ける」と書いていた | `5393a316` で確認とテストを消し、doc を escalate の打ち直しだけに合わせた。解消。残る窓は S9 に書かれ、tech-debt の行には未記載 |
| C2-2 (LOW) | 活動の判定を `leaderActedSinceAlert` に移したあとも、`checkDeadman` のものとして書いた comment が 4 か所に残った | `5393a316` で 4 か所を直した。解消 |
| C2-3 (LOW) | 4 つのテストの comment が、`/cross-review` がその場で書き直す triage を、番号(「cross-review finding N」)だけで指す | 4 件のうち 3 件を直し、1 件は C2-1 で消えた。折り返された 1 件が残る(`C2b-1`) |
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
| N5 (LOW) | N3 の直しで足した doc が、commit の項の効き方を言い過ぎている。「commit は、報告済みのパスでも条件を立てる。ALERT の件が open なら、その件の再発になる」と読めるが、`raiseOrClear` は `rec.Active` のあいだ何もしないので、ALERT を出した直後の周期の commit は再発にも ALERT にもならない。再発になるのは、間に静かな周期が入って `Active` が落ちたあとの commit だけ | 挙動は変えず、tech-debt の「Six gaps」の行の (5) に記録した(`984761f8`)。doc の文は今も変わっていない(`watch_scope.go` の `evaluateReservedScope`、`which the director should see` の行)。cycle 2 でも未修正のまま(受け入れ) |

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2b-1: `C2-3` の直しが 1 か所残っている。`AnEscalationAlreadyOnARecurredEventRecordsNothingNew` の doc が、折り返された行で「(cross-review / finding 4)」と triage を番号で指す。`C2-3` の指摘(4 か所)は、行ごとの grep で数えたため、2 行にまたがるこの 1 か所を拾わなかった(この report の取りこぼしで、直しは指摘の一覧どおりに行われた)。`/cross-review` が triage をその場で書き直したあとは、「finding 4」が別の行を指すか、無くなる。直した 3 件は「the case the cycle 1 cross-review of plan ... found」と、番号なしで case を言葉で書くので、この 1 件だけ形が違う | `internal/org/watch_test.go:3691-3692`(`// D2, with D4 of plan 2026-10-23-watch-escalation-inbox (cross-review` の次の行が `// finding 4): a pending alert ...`)。直した 3 件は `watch_test.go:3540`、`watch_scope_test.go:649`、`:679`。`grep -n 'finding [0-9]' internal/org/watch*.go` が、このほかに `watch.go` の 2 件(`:329`、`:997` は別の PR の指摘の番号で、この PR の triage とは関係が無い)を出すだけ | 「(cross-review / finding 4)」を「the case the cycle 1 cross-review of plan 2026-11-12-watch-alert-dedupe found」に直す(3 件と同じ書き方)。ついでに、次の注の 4 行の折り返しも直せる |

CRITICAL、HIGH、MEDIUM はない。

注(重さを付けない): `5393a316` が書き換えた comment の 4 行が、80 桁を超える。80 桁を超える comment 行は、直す前に `watch.go` が 4 行、`watch_recurrence.go` が 0 行で、直したあとは 7 行と 1 行になった。名前を置き換えたとき、折り返しを直さなかった。`watch.go:1476`(81 桁)、`:1518`(83 桁)、`:1607`(83 桁)、`watch_recurrence.go:148`(87 桁)。gofmt の対象ではなく、挙動に影響しない。`watch_recurrence.go:145-148` は、「the test it shares with checkDeadman, which clears it by that test (leaderActedSinceAlert)」と、「test」が 2 回出て読みにくい。`C2b-1` を直すときに一緒に直す程度で、単独では指摘にしない。

## Positive notes

- 4 件はどれも、指摘された箇所だけを触って直っている。直しは 4 ファイル、`+59 / -91` で、挙動を変えた所は消した確認 1 つだけ。それも本番では当たらなかった確認なので、本番の挙動は変わらない
- `C2-1` は、確認を残して doc を弱める道ではなく、確認と、止めた時計でだけ通るテストを一緒に消した。doc の約束と、実際に当たる範囲が合った。escalate の打ち直しの確認と、その実 alert_id を使うテストは残っている
- `C2-2` で、番兵の理由を `checkDeadman` から、その guard を持つ `leaderActedSinceAlert` に移した。理由が条件式の隣に来た。`checkDeadman` の doc は 1 行の参照になった
- `C2-4` の署名は、5 つの string から 4 つに減り、`key` と escape の導出が呼び出し元から消えた。stderr 行を書かない周期では escape を作らない
- 追加行の不可視 Unicode は、`5393a316` には無い。古い名前(`LastRecurredTS`、`busySeats`、`ScopeOutside`、`signature`)と、消した関数・テストの名前が、コードと comment に取り残されていない

## Tech debt identified

cycle 2 の最後の run(cap 2/2)なので、`C2b-1` と、受け入れた残りの窓は deferral になる。直す場合は `.go` の編集になり、パイプラインを回し直す(cap を上げる判断が要る)。`docs/tech-debt/README.md` は、この run では変えていない(前の run と同じく `/sync-docs` に渡す)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 「Six gaps that fix/watch-alert-dedupe leaves in the watch's alerts」の行に (7): 立ち上がりの recurred を足してから status を保存するまでに watch が止まると、立て直した watch は別の `now` の alert_id でもう 1 つ recurred を足す(`C2-1`) | 件の回数が 1 つ多くなる。状態と期限には響かない | 窓を閉じるには、recurred に載せる alert_id を周期をまたいで同じにする必要があり、立ち上がり側には status に残る id が無い。`5393a316` は、当たらない確認を消して doc を実際の範囲に合わせた | 次の `recordRecurrence` の変更 | 計画の Progress の S9。この report の `C2-1` |
| 同じ行に (8): comment・折り返しの取り残し(`C2b-1`、4 行の折り返し、`watch_recurrence.go:145-148` の言い回し) | 挙動は変わらない。triage を番号で指す comment が、`/cross-review` のあとで別の行を指す | 直すと `.go` の編集になり、pipeline を回し直す | 次の `watch_test.go` か `watch.go`、`watch_recurrence.go` の comment の編集 | この report の `C2b-1` |
| 同じ行の (6)(M48): `scopeDefaultBranchRef` の symbolic-ref の答えが `refs/remotes/origin/` の下か確かめる check の test が無い。「Trigger」の文は「次の `scopeDefaultBranchRef` の変更」 | `907afde3` がこの関数を変えた(`namesCommit` を足した)ので Trigger は満たされたが、M48 の test は足されていない。`origin/HEAD` を `refs/heads/...` に向ける test は無く、`HasPrefix` の check を外しても落ちない形は変わらない。(6) の本文に 1 文足す | cycle の cap | 次の `scopeDefaultBranchRef` の変更 | `docs/tech-debt/README.md` の (6)、`internal/org/watch_scope_test.go` |

_(行の見出しは「Six gaps」から「Eight gaps」に変わる。上の表は、`/sync-docs` が足す内容の案で、この review は register を編集していない。)_

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。`C2-1` から `C2-4` のうち 3 件は直り、`C2-3` は 4 件中 3 件が直って 1 件が残った(`C2b-1`、LOW)。`5393a316` の diff に新しい不具合は無い。cycle 1 の `M1`、`L1` から `L6`、`N1` から `N4` の扱いは変わらず、`N5` は受け入れのまま
- Follow-ups:
  - `C2b-1` は 1 か所の comment の書き換えで直る。最後の cycle なので、cap を上げて回し直すほどの重さは無い。直さない場合は上の (8) を register に足す
  - `C2-1` の残る窓は、計画の Progress の S9 が「tech-debt の既知の穴の行に足す」と書いているが、register にはまだ入っていない。`/sync-docs` が (7)、(8) と (6) の 1 文を足す
  - `5393a316` は verify、test、sync-docs の報告より後なので、この cycle の `/verify` と `/test` は直したあとの木に対して回し直す。消したテスト(`..._AFiringAlreadyOnARecurredEventIsNotAppendedAgain`)の名前は、前の版のこの report のほか、どの報告にも出ない
  - 見ていないもの: テストの実行結果、コンパイル、gofmt の整列は `/verify` と `/test`
