# Self-review report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(issue #307 の前半)
- Branch: `fix/watch-alert-dedupe`、base は `origin/main` の `028a8f6d`。レビューした HEAD は `d3a63398`。直前の `907afde3` が cycle 1 の cross-review の ACTION_REQUIRED 4 件の修正で、`d3a63398` はその記録(計画の Progress に 3 行)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 2 の 1 回目。`ralph config get standard.max_pipeline_cycles` は 2、`cycle-count.json` は cycle 2 なので、この cycle が最後の run
- Scope: `git diff 028a8f6d...HEAD` の 33 ファイルのうち Go の非テスト diff が中心。重点は `907afde3`(`watch.go`、`watch_recurrence.go`、`watch_scope.go`、テスト 2 ファイル)で、triage の 4 件が直ったかと、その diff が新しい問題を作っていないか。そのあと全体を比例した深さで見直した(`watch_recurrence.go` の全文、`watch.go` の `evaluateCycle` / `raiseOrClear` / `sendAlert` / `checkDeadman` / `escalateToInbox`、`watch_scope.go` の `scopeDefaultBranchRef` から `scopeCommittedPaths`)。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: 今回の指摘は `C2-1` から。cycle 1 の ID(`M1`、`L1` から `L6`、`N1` から `N5`)は下の「Earlier runs」にそのまま残した。tech-debt の行と計画の Progress が `N5`、`L2`、`L3` を指すため、書き直しで指す先を失わないようにした

## Cross-review(cycle 1)の 4 件の確認

`docs/reports/cross-review-triage-watch-alert-dedupe.md` の ACTION_REQUIRED 4 件。`907afde3` の diff と、直したあとの現物を読んで確かめた。

| # | triage の指摘 | 直し | この run の確認 |
| --- | --- | --- | --- |
| 1 | leader の活動が初めて見える周期に再発した firing を、古い pending に畳むと、同じ周期の `checkDeadman` がその pending を消し、firing が失われる | `leaderActedSinceAlert`(`watch.go:1744-1771`)を `checkDeadman` から取り出して共有し、`foldIntoPendingAlert`(`watch_recurrence.go:170`)が畳む前に同じ判定を使う。leader が動いていたら pending を消して次の pending を試し、無ければ false を返して新しい ALERT になる | 直っている。畳む判定と消す判定が同じ関数になった。`foldIntoPendingAlert` のループは、毎回 pending を消すか return するので止まる。manifest が読めないときは stderr に 1 行出して false(新しい ALERT 側)に倒れる。新しいテスト(`watch_test.go:3550`)は manifest の `sent`、agmsg の history、何もしない、の 3 通りで、前の挙動では同じ周期の `checkDeadman` が `first` を消して `status.PendingAlerts[want]` が nil になり、最初の 2 通りが落ちる(読んで追った) |
| 2 | merge commit の `git log --name-only` はパスを出さず、merge の中での衝突解決が漏れる | `git log` に `--cc` を足した(`watch_scope.go:292`)。doc は 274-288 行と 374-381 行を引数に合わせて書き直した | 直っている。git 2.49.0 の scratch repo で確かめた(下の Evidence)。衝突を新しい内容で解決した merge はそのパス 1 つ、merge の中の余計な編集はそのパス、upstream の内容で解決した merge と `--ours` で解決した merge と衝突の無い merge は空。`-z` の出力は NUL 区切りの名前だけで、merge でも形が変わらない。`--cc` を外すと最初の例が空になり、元の穴が再現する |
| 3 | `origin/HEAD` が指す先の ref が消えていると、`git log ... --not <ref>` が毎周期失敗し、L1 の再試行の規則で前の HEAD が永久に据え置かれる | `namesCommit`(`watch_scope.go:255-258`)で symbolic-ref の指す先も `rev-parse --verify --quiet <ref>^{commit}` で確かめ、通らなければ main、master、最後に `git diff` へ落とす(`:260`) | 直っている。実 git のテスト(`watch_scope_test.go:685`)が main あり・main も master も無しの 2 通り、stub のテストが呼び出しの並び(`symbolic-ref` のあとに `rev-parse` が 1 つ増える)を固定している。残りの穴(M48)は「Tech debt identified」の項目 (6) に書いた |
| 4 | `escalateAsRecurrence` が recurred を足したあと status を保存する前に止まると、立て直した watch が同じ pending を escalate し直し、件自身の alert_id しか見ない検索を通って 2 回数える(resolved なら新しい件ができる) | `alertItemWithRecurredID`(`watch_recurrence.go:60`)を足し、`escalateToInbox`(`watch.go:1871`)が自分の alert_id の件の検索のあと、`openAlertItemForKey` の前で呼ぶ。見つけたら何も足さず stderr に 1 行 | escalate の打ち直しについては直っている。alert_id は status に残る pending の id で、立て直しても同じなので、件が open でも resolved でも当たる。テスト(`watch_test.go:3697`)は両方の状態を持つ。立ち上がり側に足した同じ確認(`watch_recurrence.go:128-133`)は本番では当たらない(C2-1) |

## Earlier runs(cycle 1)

cycle 1 は 3 回回した。ID は run ごとに分けてある(1 回目は `M1`、`L1` から `L6`、2 回目は `N1` から `N4`、3 回目は `N5`)。

### 1 回目(commit 27a76047、HEAD `0b2c0bce`)

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

### 2 回目(commit 6025f3cb、HEAD `880a3fa0`)

CRITICAL・HIGH・MEDIUM なし、LOW 4。`eb596f49` で 4 件とも直し、3 回目で確かめた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| N1 (LOW) | `printableScopePath` の doc コメントに U+2028 の文字そのものが入り、escape の例が欠けて見える | ` ` の 6 文字の escape にした。解消 |
| N2 (LOW) | `ScopeSeen` の key が `scope_outside_paths_seen` で欄名と食い違い、コメントが main に無い build を指す | 欄を `ScopeOutsidePathsSeen` に改名し、コメントに前の形と今の扱いを書いた。解消 |
| N3 (LOW) | `committedOutside` の項が何を足すか、doc にも計画にも書かれていない | `evaluateReservedScope` の doc に理由を足した。足した文の言い過ぎが N5 |
| N4 (LOW) | `recurred: 1 times (last ...)` | 回数 1 のとき `1 time`。解消 |

### 3 回目(commit 690e7235、HEAD `eb596f49`)

CRITICAL・HIGH・MEDIUM なし、LOW 1。N1 から N4 はすべて直っていた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| N5 (LOW) | N3 の直しで足した doc が、commit の項の効き方を言い過ぎている。「commit は、報告済みのパスでも条件を立てる。ALERT の件が open なら、その件の再発になる」と読めるが、`raiseOrClear` は `rec.Active` のあいだ何もしないので、ALERT を出した直後の周期の commit は再発にも ALERT にもならない。再発になるのは、間に静かな周期が入って `Active` が落ちたあとの commit だけ | 挙動は変えず、tech-debt の「Six gaps」の行の (5) に記録した(`984761f8`)。doc の文は今も変わっていない(`watch_scope.go:394-402`、`which the director should see` は 400 行)。cycle 2 でも未修正のまま(受け入れ) |

## Evidence reviewed

- `907afde3` の diff を 5 ファイル全部読んだ(`watch.go`、`watch_recurrence.go`、`watch_scope.go`、`watch_test.go`、`watch_scope_test.go`)。非テストの変更は +170 / -53 行(`git show 907afde3 --numstat`)で、`leaderActedSinceAlert` の取り出し、`foldIntoPendingAlert`、`alertItemWithRecurredID`、`namesCommit`、`--cc`
- `--cc` の probe: `GIT_CONFIG_GLOBAL=/dev/null` の scratch repo(git 2.49.0)で、`git log --no-show-signature --cc --no-renames --name-only -z --format= <prev>..HEAD --not refs/remotes/origin/main --` を 7 つの形に当てた。A 衝突を新しい内容で解決: `other/x.go` のみ。B 衝突の解決と merge 内の別ファイルの編集: その 2 パス。C upstream の内容で解決: 空。D 自分の内容で解決(`--ours`): 空。E 衝突の無い merge: 空。F 衝突の無い merge のあとの通常 commit: その commit のパスだけ。G rebase の形(prev が upstream の先頭): 置き直した commit のパスだけ。`--cc` を付けない同じ範囲は A、B、C、D、E で空。出力は NUL 区切りの名前で、merge でも余計な区切りが入らない
- 追加行の不可視 Unicode の走査(U+0080-009F、U+2028/2029、ゼロ幅、方向制御、BOM、U+FFFD)を `git diff 028a8f6d...HEAD` 全体にかけた。一致なし。`fmt.Print`、`TODO`、`FIXME` の追加も無し
- 古い名前の取り残し: `pendingAlertForKey` の戻り値が 2 つになった変更は、呼び出し元 `foldIntoPendingAlert` とテスト 1 か所だけで、取り残しなし
- `evaluateCycle` の順序を読んだ(座席の評価が `rr.Events`、そのあと `rr2` の読み直しで `checkDeadman`)。`foldIntoPendingAlert` が manifest を読み直すのは、`evaluateSeat` が受け取る `rr.Events` が周期の頭の読みで、`checkDeadman` の読みとずれるため。この読み方は doc が理由を書いている
- 計画の Progress(`d3a63398`)に S8 と D6 の本文との差(`--cc` と `origin/HEAD` の確かめ)が書いてある。本文は digest が変わるので変えない、という判断は過去の run と同じ扱いで、筋が通る
- DAG の順: `18bb7424`(triage)の子が `907afde3`(修正)、その子が `d3a63398`。verify、test、sync-docs の報告と AC の `[x]` は `907afde3` より前で、直す前の木のことを書いている。この cycle の `/verify`、`/test`、`/sync-docs` が回し直す
- 見ていないもの: テストの実行結果と gofmt の整列。偽の agmsg の `HistorySeq` が呼び出しごとにどう進むか(`watch_test.go:3550` の 2 つ目のケースが、`leaderActedSinceAlert` の履歴の読みで 1 行増えたと見えるかは `/test`)。manifest のパスにディレクトリを渡すと `Read()` が失敗する前提(`watch_test.go:3649`)。後者 2 つは読んで筋は通ったが、実行していない。文書の drift は `/sync-docs`

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-1: `recordRecurrence` に足した「firing 自身の alert_id が recurred event に載っているか」の確認は、本番では当たらない。alert_id は `watchAlertID(key, now)` で、`now` は周期ごとの `w.org.nowTime()`(ナノ秒)。watch が recurred を足したあと status を保存する前に止まると、立て直した watch の次の立ち上がりは別の `now` で別の alert_id を作るので、確認は外れ、2 つ目の recurred が付く。同じ `now` を 2 回使えるのは、時計を止めたテストだけ。それでも `alertItemWithRecurredID` の doc は「status を保存する前に止まった watch の再実行を見分ける」と書き、`recordRecurrence` の doc も「すでに記録済み」と書く。triage の 4 件目が閉じたのは escalate の打ち直し(alert_id が status に残る pending の id)だけで、立ち上がり側の同じ窓は閉じていない。害は件の回数が 1 つ多くなるだけで、状態と期限には響かない | `internal/org/watch_recurrence.go:128-133`(確認)、`:54-59` と `:104-107`(doc)。`internal/org/watch.go:340-342`(`watchAlertID`)、`:1005`(`evaluateCycle` が周期ごとに `now` を 1 回取る)。`internal/org/watch_test.go:3742`(`TestWatch_Recurrence_AFiringAlreadyOnARecurredEventIsNotAppendedAgain` は `watchAlertID(key, f.clk.Now())` で recurred を先に作り、止めた時計と同じ id にしている) | 確認とそのテストを消し、2 つの doc を escalate の打ち直しだけに合わせる。立ち上がり側の窓(recurred を足してから status を保存するまでに止まると、立て直した watch がもう 1 つ足す)は、既知の穴の行に 1 項足す。確認を残すなら、doc を「同じ `now` の再実行だけ見分ける」に弱める |
| LOW | readability | C2-2: 活動の判定を `leaderActedSinceAlert` に移し、呼び手が `checkDeadman` と `foldIntoPendingAlert` の 2 つになったが、判定を `checkDeadman` のものとして書いた文が 4 か所に残る。(a) `leaderActivityEventCount` の doc「この関数の 2 つの呼び出し元は baseline と recount の組で、sendAlert が snapshot を取り checkDeadman が数え直す」。呼び出し元の数は今も 2 つ(`sendAlert` と `leaderActedSinceAlert`)で合うが、数え直すのは `leaderActedSinceAlert`。(b) 同じ doc の「checkDeadman が count を baseline と比べる」。(c) `historyLeaderLineCount` の doc「cur > baseline, see checkDeadman」で、比べているのは `leaderActedSinceAlert`。(d) `checkDeadman` の doc が、baseline が番兵のとき probe を活動に数えない理由(PR④ の known gap #5)を長く書いたまま、その guard の条件式は helper に移り、doc から helper への参照が無い。また `leaderActedSinceAlert` の doc は「recordRecurrence は畳まない」と書くが、呼ぶのは `foldIntoPendingAlert` | `internal/org/watch.go:1474-1476`(a)、`:1518`(b)、`:1607`(c)、`:1698-1711`(d)。`:1752`(helper の doc の `recordRecurrence`)。`:1724` と `internal/org/watch_recurrence.go:189`(2 つの呼び出し) | (a) から (c) の `checkDeadman` を `leaderActedSinceAlert` に直す。(d) は `checkDeadman` の doc に「判定は `leaderActedSinceAlert`」と 1 行足し、番兵の理由は helper の doc に移す。`recordRecurrence` は `foldIntoPendingAlert` にする |
| LOW | maintainability | C2-3: `907afde3` が足した 4 つのテストの doc が、bare の「cross-review finding N」で triage を指す。`/cross-review` は `docs/reports/cross-review-triage-<slug>.md` を cycle ごとにその場で書き直し、次の工程がそれなので、cycle 2 の triage が書かれると「finding 1」から「finding 4」の番号は別の行を指すか、無くなる。同じ comment は先頭に計画の ID(`D1`、`D2`、`D6`、`AC9`)を置いていて、そちらは計画の本文が変わらないかぎり安定している | `internal/org/watch_test.go:3540`(`D1 (cross-review finding 1)`)、`:3738`(`D2 (cross-review finding 4)`)。`internal/org/watch_scope_test.go:649`(`AC9 and D6 (cross-review finding 2)`)、`:678`(`Cross-review finding 3`) | 括弧の「(cross-review finding N)」を消し、先頭の `D1` などだけにする。triage の指摘を残したいなら、`triage cycle 1 #N of watch-alert-dedupe` のように cycle と slug を付ける。`/cross-review` が走る前に直すのが筋 |
| LOW | readability | C2-4: `foldIntoPendingAlert` の引数が `(status, orgID, key, condType, org, seat string)` で、5 つの string が並び、`orgID` と `org` は名前が近いのに意味が違う(後者は stderr 用に escape した写し)。`key` は呼び出し元が `orgID`、`seatID`、`condType` から作ったもので、seat の素の値は渡されない。呼び出し元は escape した 2 値を、stderr 行を書かない周期でも毎回作る。入れ替えてもコンパイルが通り、doc の 1 文(`org and seat are orgID and the seat, escaped for stderr`)だけが歯止め | `internal/org/watch_recurrence.go:117-118`(呼び出し元)、`:170`(署名)、`:151-169`(doc) | `seatID` を渡し、escape は manifest が読めないときの 1 つの分岐の中でやる(`recordRecurrence` の後ろの 3 つの stderr 行は、今の `org, seat` をそのまま使う)。引数が `(status, orgID, seatID, condType)` に減り、`key` は中で作れる |

CRITICAL、HIGH、MEDIUM はない。

## Positive notes

- 4 件はどれも、指摘された箇所だけを触って直っている。`907afde3` は 5 ファイルで、非テストは +170 / -53、doc の書き直しが同じ commit に入っている(`raiseOrClear` の「leader が動いていない pending」、`escalateToInbox` の recurred event の項、`scopeCommittedPaths` の `--cc` の項、`evaluateReservedScope` の `--cc` の項)
- 1 件目は判定の取り出しで直した。`checkDeadman` と `foldIntoPendingAlert` が別々の条件式を持つ形ではなく、同じ関数を呼ぶ形なので、2 つの判定が将来ずれにくい。`foldIntoPendingAlert` が読めない manifest の場合に新しい ALERT 側へ倒れるのは、この watch の「迷ったら出す」の規則と揃っていて、そのテストが stderr の行まで固定している
- 2 件目は引数 1 つの変更で、probe の 7 つの形に対して doc の文(「パスのうち、merge 結果がどの親とも違うもの」)が合った。実 git のテスト(`watch_scope_test.go:654`)は、upstream だけが持ち込むパス(`other/new.go`)が出ないことまで見ている
- 3 件目の stub のテストは、`rev-parse` の呼び出し 2 つ(trunk の確認、main の確認)と、trunk が commit を指す場合に main と master を聞かないことを順番ごと固定している
- 追加行の不可視 Unicode は無く、改名の取り残しも無い

## Tech debt identified

この run は cycle 2 の最後の run(cap 2/2)なので、LOW 4 件はそのままだと deferral になる。直す場合は `.go` の編集になり、パイプラインを回し直すことになる(cap を上げる判断が要る)。直さない場合の row の文面を下に置く。`docs/tech-debt/README.md` は、この run では変えていない(前の run と同じく、`/sync-docs` に渡す)。

「Six gaps that fix/watch-alert-dedupe leaves in the watch's alerts」の行(`docs/tech-debt/README.md`)に足す内容を、場所・今の状態・直し方で書く。

| 場所 | 今の状態 | 直し方 |
| --- | --- | --- |
| 行の頭 | 「Six gaps」 | 「Eight gaps」。Related に `docs/reports/self-review-2026-11-12-watch-alert-dedupe.md`(C2-1 から C2-4)は既にあるファイルなので ID だけ足す |
| 項目に (7) | 無い | C2-1: 立ち上がりの recurred を足してから status を保存するまでに watch が止まると、立て直した watch は別の `now` の alert_id でもう 1 つ recurred を足す(件の回数が 1 つ多い)。`recordRecurrence` の alert_id の確認(`watch_recurrence.go:128-133`)はこの再実行には当たらず、止めた時計のテストだけが通る。Impact は件の回数のずれだけ。Why deferred は cycle cap。Trigger は次の `recordRecurrence` の変更 |
| 項目に (8) | 無い | C2-2 から C2-4 の comment・署名の取り残し(`watch.go:1474-1476`、`:1518`、`:1607`、`:1698-1711`、`:1752`、`watch_test.go:3540`、`:3738`、`watch_scope_test.go:649`、`:678`、`foldIntoPendingAlert` の署名)。挙動は変わらない。Trigger は次の `watch.go` か `watch_recurrence.go` の編集 |
| 項目 (6) | 「`scopeDefaultBranchRef` の symbolic-ref の答えが `refs/remotes/origin/` の下か確かめる check(M48、INFO)」の test が無い。Trigger は「次の `scopeDefaultBranchRef` の変更」 | `907afde3` がこの関数を変えた(`namesCommit` を足した)ので Trigger は満たされたが、M48 の test は足されていない。`origin/HEAD` を `refs/heads/...` に向ける test は `watch_scope_test.go` に無い(`refs/heads` の grep は 0 件)。`HasPrefix` の check を外しても落ちない形は変わらず、むしろ `namesCommit` が通るので、`refs/heads/main` を default と取る mutant は今も生き残る。(6) の本文に「`907afde3` が変えたが M48 は足していない」と 1 文足す |

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。cross-review の 4 件はすべて直っていて、新しい指摘は LOW 4 件(C2-1 から C2-4)。cycle 1 の M1 と L1 から L6、N1 から N4 の扱いは変わらず、N5 は受け入れのまま
- Follow-ups:
  - C2-1 が 4 件のうちいちばん中身がある。本番で当たらない確認と、それを前提にした doc とテストが残る。コードの編集(確認の削除)なので、cap が 2/2 の今は、直すなら cap を上げて回し直す判断になる。直さない場合は上の (7) を row として足す
  - C2-3 は `/cross-review` が triage をその場で書き直す前に直すのが筋で、直せるのは今だけ。test の comment だけの変更でも pipeline の対象になる点は同じ
  - 上の行(6)の Trigger が満たされた件は、`/sync-docs` が `Six gaps` の行を直すときに 1 文足す
  - `907afde3` は verify、test、sync-docs の報告と AC の `[x]` より後なので、この cycle の `/verify` と `/test` は直したあとの木に対して回し直す。この run で grep した範囲(`.claude/skills`、README、`docs/recipes`、`docs/specs`、AGENTS.md、`templates/base/.claude`)には、`git log --no-renames` や「merge は 0 件」の記述が無く、`--cc` の直しで文書の本文を直す所は見つからなかった。`/sync-docs` が触るのは tech-debt の行が中心になる見込み(drift の判定そのものは `/sync-docs` の仕事)
  - 見ていないもの: テストの実行結果と gofmt の整列は `/verify` と `/test`
