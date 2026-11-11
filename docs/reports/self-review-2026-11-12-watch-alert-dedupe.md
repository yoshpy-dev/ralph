# Self-review report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(issue #307 の前半)
- Branch: `fix/watch-alert-dedupe`、base は `origin/main` の `028a8f6d`。レビューした HEAD は `0b2c0bce`
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 1
- Scope: `git diff 028a8f6d...HEAD` の 16 ファイル(+3509 / -109)のうち、実装の `internal/org/inbox.go`、`internal/org/deadline.go`(コメントのみ)、`internal/org/watch.go`、`internal/org/watch_scope.go`、`internal/cli/org_inbox.go`、`internal/cli/org_watch.go`。テストは、実装の主張を確かめるために `watch_scope_test.go` の git 呼び出しの数を固定するテストと、新しいテストの名前の一覧だけを読んだ。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない

## Evidence reviewed

- `internal/org/watch.go` の diff を頭から読んだ。`raiseOrClear` が条件の立ち上がりで `recordRecurrence` を呼ぶ位置(`status.Conditions[key]` を Active にしたあと、`sendAlert` の前)、`recordRecurrence` の 3 つの出口(pending に足す、受信箱の件に `recurred` を足す、新しい ALERT に倒す)、`escalateToInbox` の D2 の枝(同じ alert_id の件の確認のあと、新しい件を作る前)と `escalateAsRecurrence`、`evaluateCycle` の座席の並べ替え、`evaluateSeat` の stall の式(`stalled && !leaderIsWaiting`)を読んだ。`escalateAlert`(`:1879-1891`)が `escalateToInbox` の nil のあとに pending を消すことも確かめた
- `internal/org/watch_scope.go` を全文読んだ(425 行)。git に渡す引数をすべて洗った。read-back した `snap.ScopeHead` と `head` は `isGitObjectName`(40 または 64 桁の小文字 16 進)を通してから `prev..head` に使う。`merge-base` の出力も `isGitObjectName` を通す。origin の既定のブランチは `refs/remotes/origin/` で始まる完全な ref 名で、`refs/` で始まるので option に読まれない。`ls-tree` と `log` のパスは `--` のあとに置く。`ls-tree` の候補は `docs/plans/` か `docs/reports/` で始まるので、pathspec の magic(`:(...)`)にならない
- `internal/org/inbox.go` の `AppendRecurred`。検査の順(alertID、count、note の順に、読む前に断る)、`update` の `withLock` の中で読み直してから足すこと(`:502-523`)、`ErrInboxResolved` と TYPE の確かめがロックの中の `decide` にあることを読んだ。`validateInboxNoteLine` への切り出しで、`ValidateInboxNote` の検査の順と文言(`resolve` の語)が変わらないことを、前後の diff で確かめた
- 実際の git 2.49.0 で、scratchpad の使い捨て repo を使って次を確かめた。(1) `git status --porcelain=v1 -z --untracked-files=all` の rename は `R  src/b.go\0src/a.go\0` の順で、`parsePorcelainZ` の想定と同じ。(2) `-c status.branch=true` を付けても porcelain v1 の出力は変わらない。(3) `log.diffMerges` を `first-parent` / `separate` / `combined` / `dense-combined` / `on` にしても、merge commit に対する `git log --name-only` の出力は空のまま。`scopeCommittedPaths` の「merge commit はパスを出さない」の主張はこの設定で崩れない。(4) `diff.relative=true` は cwd が repo の root でないときだけ効く(watch の cwd は座席の worktree の root なので、実害は見つからなかった)
- `scopeOutsideNote` の予算の計算を、3 つの 200 文字のパスで手計算した。前のパスを残した時点で `, … (+N more)` の分の余白が確保されていて、最後の `room < 0` の枝で書く文は余白に収まる。合計は 436 文字で 500 以下
- `printableScopePath` と `validateInboxNoteLine` の拒否条件(改行、不正な UTF-8、制御文字、500 文字)を突き合わせた。`scopeOutsideNote` の出力は拒否されない。パスの文字が原因で `AppendRecurred` が失敗し、再発が新しい ALERT に倒れ続けることはない
- `reservePathsOverlap` と `scopePathInside` の `len(r) <= len(p)` の条件を、`.`、ディレクトリ、ファイル、nested repo のディレクトリ(`sub/`)で読み合わせた
- 呼び出し側の洗い出し: `evaluateSeat` の呼び出しは `evaluateCycle` の 1 か所(`watch.go:1103`)だけで、`cycle` が nil になる道はない。`watchRun{` を組む本番のコードは `newWatchAllRun` だけで、`evaluateReservedScope` は `w.gitRun == nil` でも `realGitRun` に落ちる
- `LastRecurredTS` と `busySeats` の読み手を grep した(下の M1、L5)
- `wc -l`: `watch.go` は base で 1865 行、今は 2171 行
- `docs/tech-debt/README.md` の diff。RC3 の行を取り消し線にして RESOLVED のコメントを足し、新しい 4 つの穴の行を足している。コメントの行を表の前に置く形は、この register の 87 件の RESOLVED の行と同じ

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1: `watchPendingAlert.LastRecurredTS` は書くだけで、どこも読まない。`recordRecurrence` のコメントは「`Recurrences` と `LastRecurredTS` を更新し、`escalateToInbox` が件に移す」と言うが、`escalateToInbox` が件に移すのは回数だけ。件の `recurred` event の時刻は `InboxStore.update` が追記の時刻(`ev.TS = s.now()`)で埋めるので、pending のあいだの再発を移した件の `recurred: N times (last <ts>)` の `<ts>` は、最後の再発の時刻ではなく deadman で escalate した時刻になる。実際の最後の再発より最大 `[org].deadman_minutes` 遅い。AC1 の「最後の時刻が出る」は、この経路では実際の時刻を指さない | 書き込み: `internal/org/watch.go:1401`。読み手は `watch_test.go`(`:486`、`:3257`、`:3260`、`:3569`、`:3582`)だけ。コメント: `watch.go:240-246`(回数だけを移すと正しく書いてある)と `:1384`(`LastRecurredTS` も移すと書いてある)が食い違う。移す側: `watch.go:1958` の `AppendRecurred(id, alertID, "", pending.Recurrences)` に時刻を渡す引数がない。`inbox.go:520` | どちらかに寄せる。(a) 欄を消して `:1384` のコメントを直す。件の「最後」が escalate の時刻になることを `AppendRecurred` か `show` の説明に 1 文書く。(b) `AppendRecurred` に時刻を指定する道(`InboxEvent.TS` を呼び出し側から渡せる内部の関数)を足し、`LastRecurredTS` を移す。件の時刻が追記の時刻から外れるので、`InboxStore.Now` の注入の前提を使うテストは見直しが要る。小さい方は (a) |
| LOW | exception-handling | L1: `scopeCommittedPaths` は git の失敗をすべて `nil` にし、呼び出し側は commit のパスが取れなかった周期でも `snap.ScopeHead` を今の HEAD に進める。doc が挙げる原因は「git が持っていない前の HEAD」だけで、この場合は進めてよい。しかし `driverCallTimeout` で kill された `git log`、一時的な失敗も同じ道を通り、その周期に commit された予約の外のパスは、あとの周期で拾われない。同じ関数が既定のブランチを解決できない、merge-base が失敗する、の場合は「知らせる側に倒す」と決めているので、この 1 か所だけが倒れる向きが逆で、stderr の行もない。porcelain の失敗は「周期を飛ばして基準を動かさない」(`:373-375`)で、ここも向きが違う | `internal/org/watch_scope.go:282-285`(`err != nil` で `return nil`)、`:392-394`(呼び出し)、`:422-424`(`head != ""` で `ScopeHead` を進める)。doc: `:272-276`、`:373-376` | 前の HEAD を git が持っているかを `git cat-file -e <prev>^{commit}` で区別し、持っているのに log が失敗したときは `ScopeHead` を進めずに stderr に 1 行書く。持っていないときだけ今のように進める。直さないなら、doc の「git が持っていない前の HEAD」を「git が失敗した場合」に広げ、既知の穴の行(tech-debt の 4 つの穴の 3 つ目の近く)に足す |
| LOW | maintainability | L2: scope_change の条件は「署名が前の周期と違い、空でない」。署名は porcelain の `XY <path>` の集まりなので、予約の外のパスが増えたときだけでなく、減ったとき、同じパスの `XY` が変わったときも立つ。`[" M x", " M y"]` から `[" M y"]` になるとき、`" M x"` から `"M  x"`(`git add`)になるとき、`"M  x"` から `"commit x"`(commit)になるときがそうで、1 つのファイルの編集から commit までで再発の記録が 2〜3 件つく。新しいパスが無い周期の `recurred` の note は、残っているパスだけを並べる。再発は ALERT にならず件に畳まれるので害は小さいが、#307 の目的(ノイズを減らす)と、help の「予約の外のパスだけ」の読み方に対して、最初の 1 件のあとも `recurred` と stderr の行が増える | `internal/org/watch_scope.go:407`(`active := snap.ScopeSeen && len(signature) > 0 && !slices.Equal(signature, snap.ScopeOutside)`)。署名の組み立て: `:219-232`、`:398-406` | 立てる条件を「前の署名に無い、予約の外のパスが今の署名にある」にする(署名の `XY` を外した path の集合で比べる)。直さないなら、plan の D5 か help に「署名が変わるたびに再発になる」と書く |
| LOW | maintainability | L3: 計画の Implementation outline 4 と Risks は、merge-base と ls-tree を「HEAD が変わった周期だけ」打つとしているが、実装は候補の記録(`docs/plans/` と `docs/reports/` の下のパス)が porcelain にあるあいだ、毎周期 `symbolic-ref`、`rev-parse`(最大 2 回)、`merge-base`、`ls-tree` を打つ。HEAD が変わらなくても打ち、結果を持ち越さない。3 席が worktree を共有する機能の org では、報告を書いてから commit するまでの数周期に、同じ worktree へ座席の数だけ同じ問い合わせが走る。テストはこの毎周期の呼び出しを固定していて、計画との差は進捗の記録にない | `internal/org/watch_scope.go:322-343`(キャッシュなし)、`internal/org/watch_scope_test.go:896-906`(`candidate record and origin/main runs` の期待が `symbolic-ref, rev-parse, merge-base, ls-tree`。HEAD が変わらない周期でも同じ呼び出しが走る)。計画 `:89` と `:120` | 計画の Risks の文を実装に合わせるか、座席の snapshot に `(HEAD, 候補の一覧)` のキャッシュを持たせる。`/verify` に計画との差として渡す |
| LOW | maintainability | L4: `watch.go` は base で 1865 行、今は 2171 行(+306)。global のルール(ファイルは 800 行未満)は base から超えていたが、今回の追加のうち再発の仕組み(`watchAlertID`、`alertIDKey`、`pendingAlertForKey`、`openAlertItemForKey`、`recordRecurrence`、`escalateAsRecurrence`、`beforeAppendRecurred`、約 180 行)は 1 つのまとまりで、scope の側は `watch_scope.go` に出した方針と揃っていない | `internal/org/watch.go:328-392`(部品)、`:1375-1430`(`recordRecurrence`)、`:2023-2050`(`escalateAsRecurrence`)。`wc -l` | `watch_recurrence.go` に移す(挙動は変わらない)。今回直さないなら、tech-debt の既存の行に「watch.go が 2000 行を超えた」と足す |
| LOW | readability | L5: `watchOrgCycle.busySeats` は seat の名前を順に貯めるが、読むのは `len(cycle.busySeats) > 0` だけ(`:1207`)。doc の「評価した順」は誰も使わない順序を約束している | `internal/org/watch.go:1189-1191`(欄と doc)、`:1207`(読み)、`:1310`(追加)。ほかの読み手はない(grep) | `anyBusy bool` にするか、stall を抑えた stderr の 1 行に名前を載せて読み手を作る(S3 の報告は、その行を watch-status の欄が要るとして見送った。名前は cycle の中だけで足りる) |
| LOW | maintainability | L6: `WatchAllParams` に `GitStatus` と `GitRun` の 2 つの注入口があり、`GitStatus` は予約のある org では黙って無視される。doc は書いてあるが、予約のある org のテストに `GitStatus` だけを渡すと、`evaluateReservedScope` は実際の git を存在しない worktree で打ち、`status` が失敗して周期を黙って飛ばす。結果は「何も起きない」なので、テストが落ちずに空振りする | `internal/org/watch.go:576-594`(2 つの欄の doc)、`watch_scope.go:378-385`(`git == nil` の落ち先)と `:382-385`(失敗は黙って return) | 2 つの欄の doc が使い分けをすでに書いているので、受け入れてよい。残る穴はテストを書く人が `GitStatus` を渡したときの空振りだけで、直すなら予約のある org の周期で `git status` が失敗したときの 1 行を stderr に出す(今は失敗を黙って飛ばす。非予約の org の `realGitStatus` の失敗も同じく黙っているので、揃えるなら両方) |

CRITICAL と HIGH はない。

## Positive notes

- 失敗の向きが揃っている。`recordRecurrence` は pending も件も見つからない、受信箱を読めない、追記が失敗した、のどれでも「新しい ALERT を送る」側に倒れ、黙って件が増えない側には倒れない。`ErrInboxResolved` は director が resolve した直後の競合で、行を出さずに新しい ALERT になる。競合の窓に `beforeAppendRecurred` の seam があり、既存の `afterStopRequestTaken` と同じ形で、テストの後始末も `t.Cleanup` で戻している(並列のテストはない)
- git に渡す値の検査がある。状態ファイルから読み戻した HEAD は 40 または 64 桁の 16 進だけを通し、option に読まれる値(`--output=/tmp/x` の形)を、テストで固定している(`watch_scope_test.go` の `ScopeHead = "--output=/tmp/x"`)
- `parsePorcelainZ` が、実際の git の出力の rename の順(新しいパス、元のパス)と、同じ XY を元のパスにも付ける設計で合っている。X と Y のどちらかが R か C のときだけ次のフィールドを読む
- `seatBusy` は `snap.AgentGet = agentGetOut` で上書きする前に呼ばれる(`:1309` と `:1315`)ので、前の周期の返答と比べる。leader の stall が立ちうるときだけ受信箱を読み(`leaderIsWaiting` は `stalled &&` の右側)、1 org の 1 周期に 1 回で済む。受信箱を読めなければ stall を上げる側に倒す
- `AppendRecurred` は事前の検査(alertID、count、note)をロックの前に済ませ、状態と TYPE の確かめをロックの中の `decide` に置く。拒否は何も書かない。`ErrInboxResolved` と `ErrInboxUnknownID` を `%w` で包むので、呼び出し側は `errors.Is` で分ける
- `InboxItem.Recurrences` を `json` で `omitempty` にしない選択は、同じ struct の `notified` と `woken` と揃っている。`InboxDeadlineDue` が `Recurrences` を読まない、ということを関数の doc に書いた(`deadline.go`)
- `realGitStatus` を `runGitBounded` に寄せたが、挙動は変えていない(`WaitDelay` と `ErrWaitDelay` の読み方が同じ)。`escalateTypes` は `escalate.go` のものを共有していて、種類の一覧が 2 つに増えていない
- `git log` に `--no-show-signature` を足した a650ae2e は、利用者の `log.showSignature` が名前の一覧に gpg の行を混ぜる経路を塞ぐ。同じ種類の設定(`log.diffMerges`)は、git 2.49 で出力に効かないことを確かめた

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (今は足さない)M1、L1 から L6 | 上の表のとおり | orchestrator が直すか先に送るかを決める前なので、register には書いていない。直さない指摘が残るなら、その ID の行を足す | 直さないと決めたとき | 本報告 |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_ 今回は足していない。

## Recommendation

- Merge: yes。CRITICAL と HIGH はない。MEDIUM が 1 件(M1)ある。M1 は `LastRecurredTS` を消してコメントを直す小さな変更で済み、AC1 の「最後の時刻」に近い表示の食い違いなので、直してから進めることを勧める
- Follow-ups:
  - 直すなら M1(欄とコメント。選択肢 (a))と L1(3 行と stderr の行)を優先する。L1 は検出の抜けを作る向きに関わる。cycle 1 なので、直したら `/self-review` から全パイプラインを回し直す(上限 2 の 2 回目)
  - L3 は計画との差なので `/verify` に渡す。L2 は plan の D5 の読み方の確認も要る
  - L4 から L6 は直さずに進めてよい。進めるなら tech-debt の既存の行に L4 を足し、L5 と L6 はコメントの直しで済む
  - 見ていないもの: 新しいテストの中身(振る舞いの証拠は `/test`)、help と文書の drift(`/verify`、`/sync-docs`)
