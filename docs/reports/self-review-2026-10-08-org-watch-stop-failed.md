# Self-review report: org-watch-stop-failed

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-08-org-watch-stop-failed.md
- Branch: fix/org-watch-stop-failed(HEAD 3bfa64b9、base da4dccb0)
- Reviewer: reviewer subagent (Claude)、パイプライン 1 回目(cycle 1)
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、typo、null 安全、デバッグ用コード、秘密情報、例外処理、安全性、保守性、コメントの正確さ)。対象は `git diff da4dccb0...HEAD` の 4 ファイル(+259/-10)で、コードは 2765f002、残りは計画。重点は、`leaderActivityEventCount` の doc comment が `stop_failed` の書き手、`reason=watchdog_` の除外、更新をまたぐ窓について正確か、`fakeWatchHerdr.PaneGetErr` が未設定のとき既存のテストの挙動を変えないか、tech-debt の行の RESOLVED の書き方が既存の行と合うか。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査、mutation は実行していない

## Evidence reviewed

- `git diff da4dccb0...HEAD -- internal/org/watch.go internal/org/watch_test.go docs/tech-debt/README.md` を全行読んだ。計画は `docs/plans/active/2026-10-08-org-watch-stop-failed.md` を全行読んだ
- `stop_failed` の書き手: `EventStopFailed` の非テストの参照は `verbs.go:920`(`Stop`)だけ。`Stop` の呼び出し元は `verbs.go:1716`(`Disband`)、`:1754`(`disbandOwnLast`)、`verbs_all.go:138`、`:145`(`StopAll`)、`cli/org.go:768`(`stop`)。`DisbandAll` は `Disband` を経由する。`watch.go` に `.Stop(` も `PaneGet` もない。doc comment の「`stop` と `disband`(と `--all`)が書き、watchdog は書かない」は合っている
- v5.1.0 が `stop_failed` を書かないこと: `git grep -n 'stop_failed\|StopFailed' v5.1.0 -- internal` は 0 件。`git tag --contains da4dccb0` は空。doc comment の「v5.1.0 は書かない」「#208 は未配布」は合っている
- `fakeWatchHerdr` の使われ方: `internal/org/watch_test.go` の外に参照はない。`f.PaneGet(` を呼ぶ箇所もない。ほかの fake のメソッド(`AgentGet` など)が `f.mu` を持ったまま `PaneGet` に入る経路もない
- tech-debt の RESOLVED の書き方: 24〜26 行、148〜149 行と比べた
- `/pr` の archive が tech-debt の plan 経路を書き換えること: ee4dc3e7(#208)が `docs/tech-debt/README.md` の `plans/active/2026-10-07-org-stop-all` 8 か所を `plans/archive/` に直している

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | maintainability | 追加した 1 文「`stop_failed` gets no guard of this kind」が、コードと同じ comment の前の文に反する。コードは `stop_failed` を同じ `case` に入れたので、`reason=watchdog_` の除外は `stop_failed` にも掛かる。テストもそれを固定している(`stop_failed carrying reason=watchdog_ does not count`)。comment の前半も「The exclusion applies to any event in this lifecycle set」と言い、その一覧に `stop_failed` を足している。そのあとの段落で「guard なし」と読める文が来るので、読み手は `stop_failed` に除外が掛からないと受け取る。言いたいのは「除外を `stop_failed` に付ける互換上の理由はない(古い watchdog は書かない)。掛かるのは `case` を共有しているから」のはず。計画の Scope 18 行も「ほかの種類と同じく、`reason=watchdog_` を持つものは数えない」と書いている。誤読して `stop_failed` を別の `case` に分けて除外を外すと、`TestLeaderActivityEventCount_StopFailed` の 2 つ目が落ちるので壊れはしないが、comment が間違った方向に誘う | `internal/org/watch.go:816-822`(追加した文)、`:796-798`(「The exclusion applies to any event in this lifecycle set」)、`:839-840`(`case` と `reason=watchdog_` の判定)、`internal/org/watch_test.go:1481` | 文を「`stop_failed` は `case` を共有するので同じ除外が掛かるが、除外を付ける互換上の理由はない。」のように、除外は掛かること・互換の理由がないことの 2 点に分けて書く。AC4 が求める「互換のための除外が要らない理由」はそのまま残せる |
| F-2 LOW | readability | 追加した段落は、リリースの順序を前提にした書き方をしていて、次のリリースで古くなる。「it is first counted here in the same release」「v5.1.0, the latest release before that」の「that」は直前の節(`stop_failed` が数えられる release)を指すが、コードからは release が読めない。v5.2.0 が出ると「the latest release before that」は v5.2.0 のことにも読める。最後の文は 5 行に 3 つの条件(older watch が動いている間に新しい stop が書く、watch を新バイナリで立て直す、警告が pending)を入れ子にしていて、1 回では読み切れない | `internal/org/watch.go:816-822` | 「v5.1.0 までは `stop_failed` を書かない」と事実で書き、窓の説明は 2 文に割る(1: 基準が `stop_failed` を取りこぼす条件、2: そのあと数え直しが 1 大きくなる条件)。「同じ release に入れる」という運用は PR 本文と計画に任せ、コードの comment からは外す。tech-debt の HTML コメントの「it ships in the same release as PR #208」も同じ種類 |
| F-3 LOW | maintainability | `fakeWatchHerdr.PaneGet` が `f.mu` を取るようになったが、テストは `h.PaneGetErr = ...` を `mu` なしで代入する(`:1400`、`:1449`)。ロックは `PaneGetErr` を守っていない。同じ構造体の `AgentGetErrSeq` も、テストは `mu` なしで代入し、`AgentGet` の中だけで `mu` を取るので、既存のやり方とは合っている。PaneGet は並行に呼ばれないので害はない | `internal/org/watch_test.go:90-95`、`:1400`、`:1449` | 直さなくてよい。直すなら、ロックを外すか、`PaneGetErr` の doc に「テストは cycle の間にだけ代入する」と足す |

未設定のときの挙動: `PaneGetErr == nil` なら、返す値は変更前と同じ `("", "", NewHerdrError(PaneNotFound, ...))`。ロックの追加でデッドロックになる経路はない(上の Evidence)。既存のテストの挙動は変わらない。

tech-debt の行: 既存の RESOLVED の行(25〜26、148〜149 行)と同じ形になっている。行の上に `<!-- RESOLVED <date> in <branch>: ... Row preserved for traceability. -->`、1 列目と「次に直すとき」の列を `~~` で消して `(RESOLVED <date> in <branch>)` を付け、Impact と Why deferred は残し、Related に解決した計画を足す。コメントが言う「`EventStopFailed` を case から外すと 1 つ目と 3 つ目が落ちる」も、テストを読んだ範囲では合う(2 つ目は別 org のテストで、外しても緑のまま)。mutation は実行していない。Related 列が `docs/plans/active/...` を指すのは、`/pr` の archive の commit が書き換える(ee4dc3e7 の前例)ので、今は直さない。

## Positive notes

- テストの `stop_failed` は、手書きの記録ではなく本物の `Stop` が書いたものを使う(`PaneGetErr` で確認の get を失敗させる)。書き手の変更(details の形、`OrgID`、`SeatID`)にテストが追従する。setup の assert が `stop_failed` がちょうど 1 件で、org と seat が合うことも見ている
- 別 org のテストは、`org-b` の `Stop` が実際に `stop_failed` を書いたことを assert してから、`org-a` の警告が escalate されることを見ている。`stop_failed` が書かれなかったために緑になる、という形を避けている
- 単体テストの `Details` の文字列は、`Stop` が組み立てる形(`ctrl_c=skipped: pane not confirmed`、`pane=not closed: ...`、`leave=skipped: ...`、`verbs.go:957`)と合う
- 変更は `case` の 1 語、comment、fake の 1 フィールド、テスト、tech-debt の 1 行に収まる。不要な整形や無関係な変更はない
- デバッグ用コード、秘密情報、`t.Skip` の追加はない

## Coverage gaps

- テストと mutation は実行していない(指示による)。tech-debt の行と計画の進捗の「この直しを戻すと新しいテストが落ちる」は、実装者の記録を読んだだけで、再現していない
- 更新をまたぐ窓(古い `ralph org watch` が動いている間に新しい `ralph org stop` が `stop_failed` を書き、watch を新バイナリで立て直す)は、doc comment と計画の進捗の記録に書いてあるだけで、テストはない。計画の「前提」の 3 つ目(33 行)と「リスク」(85 行)はこの窓を「起きない」「残るのは手元ビルドだけ」と書いたままで、進捗の記録(106 行)と食い違う。計画の本文は承認時の digest に入っているので、直さず進捗の記録を残している。文書のずれは `/sync-docs` の担当で、PR 本文の既知の穴に窓を書くことだけは忘れないようにしたい

## Tech debt identified

新しい行は足さない。F-1 と F-2 は doc comment の直しで、`/verify` の前に直せば再実行は要らない。F-3 は受け入れる。

## Recommendation

- Merge: 可。CRITICAL、HIGH はない。MEDIUM 1 件(F-1)は doc comment の 1 文が、コードと同じ comment の前の文に反する点。`/verify` の前に直すことを勧める(コードは変わらないので、直しても cycle は増えない)
- Follow-ups: F-1 を直す。F-2 を同じ編集で直す(段落の書き直しなので、同じ 7 行を触る)。F-3 は不要。PR 本文に、更新をまたぐ窓と「#208 と同じ release に入れる」を書く(計画の進捗にも同じ指示がある)

## Re-check after 3b219fc7

3b219fc7 は `internal/org/watch.go` と `docs/tech-debt/README.md` の 2 ファイルだけを変える(+11/-8)。`git show 3b219fc7 -- internal/org/watch.go docs/tech-debt/README.md` を読んだ。テストは実行していない。

| 指摘 | 結果 | 根拠 |
| --- | --- | --- |
| F-1 MEDIUM | 解消 | `watch.go:816-818` が「除外は `case` を共有するので `stop_failed` にも掛かる。`reason=watchdog_` を持つ `stop_failed` は存在しない」と書き、`:818-825` が「互換のための guard は別に要らない。例外の窓は 1 つ」と書く。前の文(`:796-798`)と `:839-840` の `case`、`watch_test.go:1481` の subtest と食い違わない。「no guard」の語は消えた |
| F-2 LOW | 解消 | 「same release」「the latest release before that」は、`git diff da4dccb0...HEAD -- internal docs/tech-debt` の追加行から消えた(`grep -i 'same release\|latest release\|before that'` は 0 件)。事実は「ralph up to v5.1.0 never writes it」になった。窓は 2 文(古い v5.1.0 の watch が動いている間に新しい stop が `stop_failed` を書くと、警告の基準がそれを含まない。新しいバイナリで watch を立て直すと、数え直しが 1 大きくなって警告が消える)で、前の版と同じ内容。tech-debt の HTML コメント(`README.md:167`)も同じ言い方に揃った |
| F-3 LOW | 変更なし | 受け入れ済みのまま |

新しい指摘はない。`watch.go` の追加文を計画の進捗(106 行)の窓の説明と突き合わせたが、条件は同じ(v5.1.0 の watch が動いている、新しい stop が書く、新バイナリで立て直す、警告が pending)。

- Merge: 変わらない(可)。F-1 と F-2 は解消し、残る指摘は受け入れ済みの LOW の F-3 だけ。上の「Recommendation」の Merge 行は元のまま残す
