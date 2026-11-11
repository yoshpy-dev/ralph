# Self-review report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(issue #307 の前半)
- Branch: `fix/watch-alert-dedupe`、base は `origin/main` の `028a8f6d`。レビューした HEAD は `880a3fa0`
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 1 の 2 回目(1 回目の指摘を `f4f1fb93` で直したあとの再実行)
- Scope: `git diff 028a8f6d...HEAD` の 19 ファイル(+3833 / -109)。重点は `f4f1fb93` の `internal/org/watch.go`、`internal/org/watch_recurrence.go`(新規)、`internal/org/watch_scope.go` とそのテスト。`inbox.go`、`org_inbox.go`、`deadline.go`、tech-debt と仕様の diff も読み直した。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない

## Earlier run (1 回目、commit 27a76047、HEAD `0b2c0bce`)

CRITICAL・HIGH なし、MEDIUM 1、LOW 6。このあと `f4f1fb93` と `880a3fa0` で次のように処理された。この run で現物を読んで確かめた結果を右に書く。

| ID | 1 回目の指摘 | 処理 | この run の確認 |
| --- | --- | --- | --- |
| M1 (MEDIUM) | `watchPendingAlert.LastRecurredTS` が書くだけで読まれず、件の「最後の時刻」が escalate の時刻になる | 欄を消し、コメントを実装に合わせた(選択肢 a) | 解消。`LastRecurredTS` は非テストのコードとコメントに残っていない(計画の Progress の記録は除く)。`watch.go:240-247` と `watch_recurrence.go:79-86` の 2 か所が「回数だけを移し、時刻は escalate の時刻」で揃った。テストが `e1.LastRecurredAt == escalatedAt` で固定している。前の build が書いた `last_recurred_ts` を読んで捨てるテストもある。残り: `show` の `(last <ts>)` が、pending のあいだの再発では escalate の時刻になることは、利用者に見える所に書いていない(下の「M1 と L1 の残り」) |
| L1 | commit の一覧の失敗で `ScopeHead` を進め、その周期の commit パスを失う | 失敗かつ git が前の HEAD を持つなら据え置く。持たないときだけ進める(`scopeHeadKnown`) | 解消。`watch_scope.go:410-414` と `:442-444`。呼び出しの並びを固定するテスト(`AFailedCommitListingAsksWhetherGitHasThePreviousHead`)が `rev-parse --verify --quiet <prev>^{commit}` の最後の呼び出しまで確かめている。stderr の 1 行は足していない(1 回目の推奨は任意の扱いだった) |
| L2 | 署名の差で、パスが減ったときと `git add` でも立つ | 「前の周期に無かったパスか、予約の外を触る commit」に変更 | 大半が解消。減る、`git add` では立たない。テスト `OnlyANewPathOrACommitOutsideRaisesScopeChange` の手順 3 と 4 は、前の署名の比較なら立つので、直す前のコードでは落ちる。残り: すでに報告したパスを commit したときの再発(N3) |
| L3 | 計画の Risks は「HEAD が変わった周期だけ」打つとしたが、実装は候補の記録がある周期ごとに打つ | 受け入れ(`880a3fa0` で計画の Progress に理由を記録) | 記録を確認した。計画の本文(Implementation outline 4 の「HEAD が変わったときだけ取り直す」と「署名」、Risks の 1 行目)は書き換えていない。本文を変えると承認の digest が変わるので、Progress の記録で済ませる判断は筋が通る |
| L4 | `watch.go` の再発の仕組み約 180 行が 1 つのまとまり | `watch_recurrence.go` に移した | 解消。`watch.go` は 2171 行から 2045 行、新ファイルは 147 行。移した関数を元の行と空行を除いて突き合わせたところ、差は `LastRecurredTS` の 2 行(コメントと代入)だけで、挙動は変わっていない |
| L5 | `busySeats` は `len() > 0` でしか読まれない | `anotherSeatWorking bool` にした | 解消。`busySeats` の参照は残っていない(grep) |
| L6 | `GitStatus` と `GitRun` の 2 つの注入口 | 受け入れ | 変更なし。doc が使い分けを書いている |

## Evidence reviewed

- `f4f1fb93` の diff を全部読んだ。`watch.go`(欄の削除と改名、`watchOrgCycle` の bool 化、移したコメント)、`watch_recurrence.go`(147 行を全文)、`watch_scope.go`(445 行を全文。`scopeCommittedPaths` の戻り値が `(paths, listed)` になったこと、`scopeHeadKnown`、`evaluateReservedScope` の `keepHead` と `newOutside` / `committedOutside`)、2 つのテストファイルの変更
- 改名と削除の取り残し: `LastRecurredTS`、`busySeats`、`ScopeOutside`(`Paths` なし)、`signature` を repo 全体で grep した。非テストのコードと doc に古い名前は残っていない。残っているのは、前の build の key(`scope_outside`、`scope_seen`、`last_recurred_ts`)を読んで捨てることを確かめるテストの文字列と、その説明のコメントだけ
- import の確認(コンパイルはしていない): `watch_scope.go` の 11 個の import はすべて本文で使われている。`watch_recurrence.go` の 5 個も同じ。`watch.go` は移動のあとも `errors`(4 回)、`protocol`(8 回)、`strings`(17 回)、`slices`(3 回)を使っている
- 移動の忠実さ: `git show f4f1fb93 -- internal/org/watch.go` の削除行から移した 5 つの関数を取り出し、`watch_recurrence.go` の 18 行目以降と、空行を除いて diff した。差は M1 の 2 行だけ
- `evaluateReservedScope` の分岐を表にして、L2 のテストの 8 手順(2 つ編集、何も新しくない、片方を戻す、片方を stage、新しいパス、何も新しくない、commit、clean)を手で追った。期待は active が `true / false / false / false / true / false / true / false`、ALERT の本文が `[x y]`、`[y z]`、`[y z]` で、コードと合う。手順 3 と 4 は直す前の署名の比較なら立つので、このテストは直す前のコードを区別できる
- L1 のテストの git 呼び出しの並び `status, rev-parse, symbolic-ref, rev-parse, rev-parse, diff, rev-parse` を、`evaluateReservedScope` から手で組み立てて合うことを確かめた(HEAD、既定の ref の 3 段、diff、`scopeHeadKnown`)
- 状態ファイルの互換: 前の build の `scope_head` と `scope_outside` と `scope_seen` を持つ座席の記録は、新しい key(`scope_outside_paths_seen`)が無いので `ScopeSeen == false` で読まれ、最初の周期は基準だけを記録する。テストがこれを固定している
- git に渡す引数の洗い直し(1 回目と同じ結果): HEAD と前の HEAD は 40 または 64 桁の 16 進を通してから使う。merge-base の出力も同様。既定の ref は `refs/remotes/origin/` で始まる完全な名前。パスは `--` のあと
- 追加行の不可視文字の走査(U+0080-009F、U+2028/2029、ゼロ幅、方向制御、BOM): `watch_scope.go:138` の 1 件(N1)
- `inbox.go`、`org_inbox.go`、`deadline.go`、`docs/tech-debt/README.md`、`docs/specs/2026-10-07-org-multi-org-director.md` の diff を読み直した。tech-debt の「Four gaps」の 3 つ目(git が持たない前の HEAD)は、L1 の直しのあとも「持たないときは進める」で正しい
- 見ていないもの: テストの実行結果。L2 のテストが期待する 3 件の ALERT(pending が次の再発の前に消えること)は、手で追えるのは `evaluateReservedScope` の中までで、fixture の周期ごとの pending の扱いは `/test` が確かめる

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

ID は 1 回目の `M1`、`L1` から `L6` と混ざらないよう、この run の新しい指摘に `N1` から振る。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | N1: `printableScopePath` の doc コメントに、U+2028(LINE SEPARATOR)の文字そのものが入っている。「Go の escape で書く」例として `\n`、`\x1b` に並べて書いたつもりの位置で、` ` と読めない。Go のコンパイラは通すが、多くのエディタと diff の表示は U+2028 を行の区切りか空白として扱い、コメントが `(\n, \x1b,  )` と欠けて見える。1 回目では見落とした。`61fc63f1`(S4)で入った | `internal/org/watch_scope.go:138`(`xxd` で `e2 80 a8`)。同じ文字は追加行のほかには無い(diff 全体を走査)。対比: `internal/org/inbox.go` の `PrintableInboxText` の doc は `\u0085` と escape で書いている | ` ` と書き直す。`.go` の編集なので、直すなら次にコードを触るときにまとめる |
| LOW | naming | N2: `watchSeatSnapshot.ScopeSeen` の JSON key が `scope_outside_paths_seen` で、欄の名前と食い違う。隣の `ScopeOutsidePaths` は `scope_outside_paths` で対応している。この repo の同じ種類の食い違い(`LeaderAgentGet` の `lead_agent_get`)は「古い状態を読むために旧 key を残した」ものだが、ここは key のほうを新しくして欄を残していて向きが逆。理由のコメント(「earlier build of that plan」)が指すのは、この PR が merge されるまで main の誰にも見えない build で、merge 後は何を指すか読み手に分からない | `internal/org/watch.go:278-281`(コメント)、`:292`(`ScopeSeen bool json:"scope_outside_paths_seen,omitempty"`)。テスト `watch_scope_test.go:881-893` | 欄を `ScopeOutsidePathsSeen` に改名して key と揃える。コメントは「基準はパスの集まりで、`scope_outside` と `scope_seen`(porcelain の行を持つ形)は読まない」と、形の違いだけを書く |
| LOW | maintainability | N3: `active := snap.ScopeSeen && (newOutside \|\| committedOutside)` の `committedOutside` は、`newOutside` で拾えない場合にだけ効く。`paths` はすでに commit のパスを含むので、commit だけで入った新しいパスは `newOutside` が立てる。つまりこの項が足すのは「報告済みの予約外のパスを commit したとき」の再発だけで、1 つのファイルの編集から commit までで ALERT 1 件と `recurred` 1 件がつく。1 回目の L2 の目的(編集から commit までの再発を減らす)に対して、2 回目が残る。テストの手順 7(`the paths outside committed` が active)がこれを固定していて、doc(`watch_scope.go:382-388`)にも計画にも「なぜ立てるか」が無いので、次の人が `newOutside` だけに「直す」可能性がある | `internal/org/watch_scope.go:426-427`、`:382-388`(doc)。`internal/org/watch_scope_test.go` の `TestWatch_ReservedScope_OnlyANewPathOrACommitOutsideRaisesScopeChange` の手順 7。計画 `:149`(S6)は「予約の外を触る commit」とだけ書く | 立てる理由を doc に 1 文足す(例: commit で変更が branch の履歴に入るので、報告済みのパスでも知らせる)。理由が無いなら `committedOutside` と `scopeEntry.committed` を外し、手順 7 の期待を `false` にする(commit だけで入ったパスは `newOutside` が拾う) |
| LOW | readability | N4: `ralph org inbox show` の行が `recurred: %d times (last %s)` で、回数が 1 のとき `recurred: 1 times (last ...)` と出る。director が読む画面で、再発が最初に記録された直後は必ず回数 1 になる | `internal/cli/org_inbox.go:482`。テストは 4 回のときの文だけ(`org_inbox_test.go:1098`) | `recurred: 1 (last ...)` のように語を外すか、単数と複数を分ける。文を変えるなら `org_inbox.go:452` の doc とテストも合わせる |

CRITICAL、HIGH、MEDIUM はない。

### M1 と L1 の残り(指摘にはしない、判断のための記録)

- M1: `show` の `recurred: N times (last <ts>)` と、仕様の「最後の時刻を出す」の `<ts>` は、pending のあいだの再発を件に移した場合は escalate の時刻になる(`InboxStore.update` が追記の時刻を入れる)。1 回目の推奨(a)にあった「利用者に見える説明に 1 文」は、`watch.go` と `watch_recurrence.go` のコメントにだけ書かれた。差は最大 `[org].deadman_minutes` で、`InboxItem` の doc は「最新の recurred event の時刻」と正しく書いている。director が時刻を再発の発生時刻と読む場合だけ害になる。`AppendRecurred` か `printInboxItem` の doc に 1 文足せば閉じる
- L1: `scopeCommittedPaths` が失敗し続ける場合(`git log` が毎周期 `driverCallTimeout` で kill されるなど)、`ScopeHead` は据え置かれ、commit のパスは取れず、stderr にも何も出ない。`evaluateReservedScope` の doc の「次の周期で同じ commit を列挙するので知らせる側に倒れる」は、一時的な失敗には当てはまり、失敗が続く場合は当てはまらない。porcelain の失敗も同じく黙って飛ばす既存の作りと揃っている

## Positive notes

- M1 の直しは、選択肢のうち小さい (a) を取り、欄の削除、コメントの 2 か所の統一、テストでの「escalate の時刻」の固定、前の build の key を読んで捨てるテストまでそろっている。欄を消した結果、`watchPendingAlert` の JSON は計画前の形に戻る
- L1 の直しは、「git が失敗した」と「git が前の HEAD を持たない」を `scopeHeadKnown` で分けている。持たない場合は、あとの周期でも列挙できないので進めるのが正しい。テストは本物の git(commit した repo)と stub(呼び出しの並び)の両方を使い、log と diff の 2 つの列挙の経路を表で回している
- L2 の直しは、判定の単位を porcelain の行(`XY path`)からパスの集まりに替えた。署名を作る手順が消え、`scopeEntry` に `committed` を持たせて「commit が触った」を保つ作りで、コメントが前の作りの語(署名)を残していない(grep)
- L4 の移動は挙動を変えずに済んでいる。`watch_recurrence.go` の頭に、何が何を呼ぶかが 4 行で書いてある
- 状態ファイルの互換の扱いが、前の build と計画前の両方についてテストで固定されている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (今は足さない)N1 から N4 | 上の表のとおり。どれも読み手の誤読か軽い見た目の問題で、挙動は変わらない | orchestrator が直すか先に送るかを決める前。`.go` の編集は `/self-review` からの再実行になるので、直さないなら 1 行の register にまとめるのが自然 | 直さないと決めたとき、または次に `watch_scope.go` / `org_inbox.go` を触るとき | 本報告 |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_ 今回は足していない。

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。1 回目の M1 と L1 は直り、L2 と L4 と L5 も直った。L3 と L6 は理由つきの受け入れで、記録も確認した。新しい指摘は LOW が 4 件(N1 から N4)
- Follow-ups:
  - N1 は不可視の文字なので直す価値が高い。ただし `.go` の編集は cycle を使うので、N2 から N4 と一緒に 1 回の comment と改名の commit にまとめるか、直さずに `docs/tech-debt/README.md` の「Four gaps」の近くに 1 行足す。どちらでも merge は止まらない
  - N3 は理由を決めてから(doc に書くか、項を外すか)直す。外すと手順 7 のテストの期待が変わるので、`/test` で見る
  - M1 の残り(`show` の時刻)は、`/sync-docs` が `show` の説明を直すときに 1 文足せば閉じる
  - 見ていないもの: テストの実行結果(L2 のテストが期待する ALERT の件数を含む)は `/test`、help と文書の drift は `/verify` と `/sync-docs`
