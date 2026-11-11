# Self-review report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(issue #307 の前半)
- Branch: `fix/watch-alert-dedupe`、base は `origin/main` の `028a8f6d`。レビューした HEAD は `eb596f49`
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 1 の 3 回目(2 回目の指摘 N1 から N4 を `eb596f49` で直したあとの再実行)
- Scope: `git diff 028a8f6d...HEAD` の 19 ファイル。重点は `eb596f49`(5 ファイル、+81 / -33)で、N1 から N4 が直ったかと、その diff が新しい問題を作っていないか。全体は、`watch.go` の非テスト diff、`watch_recurrence.go` の全文、`watch_scope.go` の 120-240 行と 355-450 行、`inbox.go`、`org_inbox.go`、`org_watch.go`、`deadline.go` の diff を読み直した。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない

## Earlier runs

指摘の ID は run ごとに分けてある(1 回目は `M1`、`L1` から `L6`、2 回目は `N1` から `N4`、この run は `N5`)。

### 1 回目(commit 27a76047、HEAD `0b2c0bce`)

CRITICAL・HIGH なし、MEDIUM 1、LOW 6。`f4f1fb93` と `880a3fa0` で次のように処理され、2 回目で現物を読んで確かめた。

| ID | 1 回目の指摘 | 処理 | 2 回目の確認 |
| --- | --- | --- | --- |
| M1 (MEDIUM) | `watchPendingAlert.LastRecurredTS` が書くだけで読まれず、件の「最後の時刻」が escalate の時刻になる | 欄を消し、コメントを実装に合わせた | 解消。`watch.go` と `watch_recurrence.go` の 2 か所が「回数だけを移し、時刻は escalate の時刻」で揃った。前の build が書いた `last_recurred_ts` を読んで捨てるテストがある。残り: `show` の `(last <ts>)` が pending のあいだの再発では escalate の時刻になることを、利用者に見える所に書いていない(下の「M1 と L1 の残り」) |
| L1 | commit の一覧の失敗で `ScopeHead` を進め、その周期の commit パスを失う | 失敗かつ git が前の HEAD を持つなら据え置く(`scopeHeadKnown`) | 解消。呼び出しの並びを固定するテストがある |
| L2 | 署名の差で、パスが減ったときと `git add` でも立つ | 「前の周期に無かったパスか、予約の外を触る commit」に変更 | 解消。ただし commit の項の説明が足りず N3 になり、N5 に続く |
| L3 | 計画の Risks は「HEAD が変わった周期だけ」打つとしたが、実装は候補の記録がある周期ごとに打つ | 受け入れ(`880a3fa0` で計画の Progress に理由を記録) | 記録を確認した。本文を変えると承認の digest が変わるので、Progress の記録で済ませる判断は筋が通る |
| L4 | `watch.go` の再発の仕組み約 180 行が 1 つのまとまり | `watch_recurrence.go` に移した | 解消。移した関数を元の行と突き合わせ、差が M1 の 2 行だけであることを確かめた |
| L5 | `busySeats` は `len() > 0` でしか読まれない | `anotherSeatWorking bool` にした | 解消 |
| L6 | `GitStatus` と `GitRun` の 2 つの注入口 | 受け入れ | 変更なし。doc が使い分けを書いている |

### 2 回目(commit 6025f3cb、HEAD `880a3fa0`)

CRITICAL・HIGH・MEDIUM なし、LOW 4。`eb596f49` で 4 件とも直された。この run で現物を読んで確かめた結果を右に書く。

| ID | 2 回目の指摘 | 処理 | この run の確認 |
| --- | --- | --- | --- |
| N1 (LOW) | `printableScopePath` の doc コメントに U+2028 の文字そのものが入り、escape の例が `(\n, \x1b,  )` と欠けて見える | 文字を `\u2028` の 6 文字の escape に置き換えた | 解消。`watch_scope.go:138` は `(\n, \x1b, \u2028)`。追加行を diff 全体で走査した(U+0080-009F、U+2028/2029、ゼロ幅、方向制御、BOM)。一致は、前回のこのレポート自身が N1 を引用した 1 行だけで、コードには無い。`strconv.QuoteRune` は U+2028 を `'\u2028'` に直すので、コメントの例は関数の出力と合う(読んで確かめた。実行はしていない) |
| N2 (LOW) | `ScopeSeen` の key が `scope_outside_paths_seen` で欄名と食い違い、コメントが main の誰にも見えない build を指す | 欄を `ScopeOutsidePathsSeen` に改名。コメントを「key が前の `scope_seen` と違い、porcelain の行を持つ前の形は読まずに基準を取り直す」に | 解消。素の `ScopeSeen` は `internal`、`.claude`、`templates`、`docs/specs` に残っていない(grep)。10 か所の参照が `ScopeOutsidePathsSeen` に揃った。`LastRecurredTS` が残るのは、前の build の key を読んで捨てるテストの名前 1 つだけ。注記: `scope_seen` は base(`028a8f6d`)の `internal` に無く、コメントの「earlier」は main の履歴に無い形を指す。ただしコメントとテストが、その形(`scope_outside` に porcelain の行)とコードの扱い(読まずに基準を取り直す)を書いているので、読み手は探さなくても済む。指摘にはしない |
| N3 (LOW) | `committedOutside` の項が何を足すか、doc にも計画にも書かれていない | `evaluateReservedScope` の doc に理由を足した | 理由は足りた(`watch_scope.go:385-390`)。ただし、足した文が「commit は報告済みのパスでも再発になる」と無条件に読め、実際は周期の並びに依ることが分かった(N5) |
| N4 (LOW) | `recurred: 1 times (last ...)` | 回数 1 のとき `1 time`、それ以外は `N times` | 解消。`org_inbox.go:482-487`。新しいテストは、回数 1 と 2 の 2 通りで head の行を完全一致で固定し、直す前の文では 1 件目が落ちる。4 回のときの既存テストは変わらない。`times (last` を名指しする文書は Go の外に無い(grep) |

### M1 と L1 の残り(指摘にはしない、判断のための記録)

- M1: `show` の `recurred: N times (last <ts>)` の `<ts>` は、pending のあいだの再発を件に移した場合は escalate の時刻になる(`InboxStore.update` が追記の時刻を入れる)。`InboxItem` の doc は「最新の recurred event の時刻」と正しく書いている。director が時刻を再発の発生時刻と読む場合だけ害になる。`printInboxItem` か `AppendRecurred` の doc に 1 文足せば閉じる。`/sync-docs` が `show` の説明を直すときに扱える
- L1: `scopeCommittedPaths` が失敗し続ける場合(`git log` が毎周期 `driverCallTimeout` で kill されるなど)、`ScopeHead` は据え置かれ、commit のパスは取れず、stderr にも何も出ない。porcelain の失敗も同じく黙って飛ばす既存の作りと揃っている

## Evidence reviewed

- `eb596f49` の diff を 5 ファイル全部読んだ(`org_inbox.go`、`org_inbox_test.go`、`watch.go`、`watch_scope.go`、`watch_scope_test.go`)。非テストの変更は、欄の改名、コメントの書き直し、`times` の分岐 4 行
- 改名の取り残し: `ScopeSeen`、`LastRecurredTS`、`busySeats`、`signature` を repo の Go、`.claude`、`templates`、`docs/specs` で grep した。コードと doc に古い名前は残っていない
- 新しいテスト `TestOrgInbox_ShowsOneRecurrenceAsOneTime` を、近くの `TestOrgInbox_ShowsTheRecurrences` の fixture(`e1`、`Escalate` の引数、`runInboxOK`)と突き合わせた。`AppendRecurred` の 4 引数と、回数の合計が `Recurrences` になる畳み込み(`inbox.go`)から、n 回目の recurred の直後に件が n 回を持つことを手で追って、期待の `1 time` / `2 times` と時刻 09:10、09:20 が合う。`for` の `tc` を `store.Now` の closure が取る形は、呼び出しがその場で行われるので問題ない。`tc.n == 1 && strings.Contains(stdout, "1 times")` は `want` の `1 time (last` と重なる確認で、害はない。`fmt` の import は新しいテストで使われる
- `evaluateReservedScope` の doc を本体と突き合わせた(`watch_scope.go:355-449`)。`raiseOrClear`(`watch.go:1306-1326`)を読んで、doc の「再発として記録される」が成り立つ条件を調べた結果が N5
- 全体の見直し: `watch.go` の非テスト diff(`watchOrgCycle`、`leaderIsWaiting`、`seatBusy`、`evaluateSeat` の葉の評価順、`raiseOrClear`、`escalateToInbox` の再発の分岐)、`watch_recurrence.go` の全文、`inbox.go` と `org_inbox.go` の diff、`org_watch.go` の help、`deadline.go` の doc、`watch_scope.go` の `printableScopePath` から `scopeEntry` まで。`evaluateSeat` の予約ありの分岐の `return` は関数の最後の文で、`snap` は関数の頭で `status.SeatSnapshots` に格納済みなので状態は失われない。将来 (e) の後ろに条件を足すと飛ばされるが、今は指摘にしない
- 見ていないもの: テストの実行結果と gofmt の整列(`watchSeatSnapshot` の新しい 3 欄は、コメント行のあとで別の整列の塊に見える)は `/verify` と `/test`。大きいテスト 3 ファイル(`watch_test.go`、`watch_scope_test.go`、`inbox_test.go`)は、L2 のテストと `eb596f49` の hunk のほかは読み直していない。計画、仕様、`agent-messaging.md` の diff は文書の drift なので見ていない

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | N5: N3 の直しで足した doc が、commit の項の効き方を言い過ぎている。「commit は、報告済みのパスでも条件を立てる。ALERT の件が open なら、それは新しい件ではなく、その件の再発(`recurred` の note 1 つ)になる」と読める。実際は、`active` が立っても `raiseOrClear` は `rec.Active` が true のあいだ何もしない。編集の周期で ALERT が出た直後の周期に commit が入ると、条件はまだ active のままなので、commit は再発にも ALERT にもならない。再発になるのは、間に 1 周期以上の「静かな」周期(新しいパスなし)が入って `Active` が落ちたあとの commit だけ。つまり `committedOutside` が director に見せるかどうかは、編集から commit までの周期数で決まる。テストは手順 6 の静かな周期を挟んで手順 7 の commit を立てており(テストの doc も、raise の手順は静かな周期のあとに置くと書いている)、「編集の次の周期に commit」の場合は固定していない | `internal/org/watch_scope.go:385-390`(足した doc)、`internal/org/watch.go:1306-1326`(`raiseOrClear`: `rec != nil && rec.Active` で `return`)。`internal/org/watch_scope_test.go:829-876`(手順 5 の新しいパス、6 の静かな周期、7 の commit)。直す前のコードでの差はない(N3 の直しは doc だけ) | doc に条件を 1 文足す(例: 「ただし commit が、ALERT を出した周期の次の周期に入ると、条件は active のままなので記録されない」)。挙動を変える必要はない。`committedOutside` の理由を「director に必ず見せる」と読まれないようにするだけでよい。足さずに受け入れるなら、N3 の理由の文の「which the director should see」を「which the director can see when the condition had cleared in between」に弱める |

CRITICAL、HIGH、MEDIUM はない。

## Positive notes

- `eb596f49` は 5 ファイルとも、指摘された箇所だけを触っている。欄の改名は、Go の欄、コメント、テストの 3 か所が同じ commit で揃い、key は変えていないので状態ファイルの互換に影響しない(`json:"scope_outside_paths_seen"` は diff で不変)
- N4 の直しは、表示の分岐を 4 行で足し、テストが「直す前に落ちる形」(回数 1 の完全一致)で書かれている。doc の 2 か所(`printInboxItem` とテストの doc)が新しい文を同じ commit で書いている
- N1 は、escape を ASCII の 6 文字にして、`inbox.go` の `PrintableInboxText` の doc と同じ書き方に揃った
- 1 回目と 2 回目で指摘した 11 件のうち、残りは理由つきの受け入れ(L3、L6)と、`/sync-docs` に渡す記録(M1 の残り)だけになった

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (今は足さない)N5 | doc の文が、commit の項の効き方を条件なしに読ませる。挙動は変わらず、director が時刻を読み違える害もない | orchestrator が直すか受け入れるかを決める前。doc の 1 文なので、直すなら次の `.go` の編集のついでが自然 | 直さないと決めたとき、または次に `evaluateReservedScope` を触るとき | 本報告 |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_ 今回は足していない。

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。2 回目の N1 から N4 はすべて直り、新しい指摘は LOW が 1 件(N5)。1 回目の M1 と L1 から L6 の扱いは変わっていない
- Follow-ups:
  - N5 は doc の 1 文で閉じる。`.go` の編集なので、直すなら cycle を使う。直さずに受け入れる場合は、`docs/tech-debt/README.md` の「Four gaps」に 1 項足すか、PR 本文の既知の穴に書く。どちらでも merge は止まらない
  - M1 の残り(`show` の時刻)は `/sync-docs` が `show` の説明を直すときに 1 文足せば閉じる
  - 見ていないもの: テストの実行結果と gofmt の整列は `/verify` と `/test`、help と文書の drift は `/verify` と `/sync-docs`
