# Self-review report: alert-notify-batch

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-alert-notify-batch.md`(issue #307 の後半)
- Branch: `fix/alert-notify-batch`、base は `origin/main` の `94e61c38`。レビューした HEAD は `199de480`(S5 の記録。直しは `1f2197dc`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)の 2 回目の run。初回の指摘 L1 から L4 を直したあとの再レビュー
- Scope: 初回の L1 から L4 が直ったかの確認、`1f2197dc` の diff の全部、`git diff 94e61c38...HEAD` の軽い通読(`alert_notify.go` を全部、`escalate.go` の差分を全部、`deadline.go` の掃き出し、`watch_recurrence.go` の reconcile、help・テンプレート・org skill の該当節)。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)
- ID の付け方: 初回の指摘は `L1` から `L4` のまま。この run の新しい指摘はなし

## Earlier run (first run, 87015807): findings and dispositions

初回は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4、Merge: yes。4 件とも `1f2197dc` で直した。この run で 1 件ずつ確かめた結果は次のとおり。

| ID | Severity | Area | 初回の指摘 | 直した内容(`1f2197dc`) | この run の確認 |
| --- | --- | --- | --- | --- | --- |
| L1 | LOW | maintainability | `alert_notify` を読まない送りの一覧が 5 か所で別の中身 | テンプレート、`config.go` の doc、`watch --help`、director skill を、4 つの送り(leader の件、`inbox notify`、サイクルの始めの再送、受信箱に書けないときの予備の道)にそろえた。org skill と `alert_notify.go` の先頭の comment は元から 4 つ | 解消。6 か所すべてが同じ 4 つを挙げる(`templates/base/ralph.toml:257-261`、`config.go:333-338`、`org_watch.go:88-91`、`.claude/skills/org/SKILL.md:663-665`、director skill、`alert_notify.go:21-25`)。director skill の 4 つの写しは差分が同じ(index の hash が 4 つとも `8e15278a..8651397d`)。コードとの突き合わせ: `AlertNotify` を読むのは `alert_notify.go:85` の 1 か所で、4 つの道はどれも batch に `nil` を渡すか、batch を持たない |
| L2 | LOW | readability | `--types` の待ちが ALERT の open のまま時間切れになると、印の `detail` が「no open inbox item」になる | `directorWaitTimeoutDetail(types, timeout)` を足した。`--types` があれば「no open inbox item of type QUESTION,BLOCKED,RESULT within N ms」。stderr の文(`inboxWaitTimeoutPhrase`)は変えていない | 解消。`director_wait.go:540` が新しい関数を呼び、ほかに `no open inbox item within` を作る本番コードは無い。印の `detail` は `internal/cli/org_director_wait_line.go:65` が JSON の `detail` にそのまま写すだけで、文字列を一致で見る本番コードは無い(text の行は `End` の語だけを出す: 同 82 行)。director skill が一致を見るのは `no open inbox item arrived within`(`escalate.go:102`)で、そのまま |
| L3 | LOW | maintainability | 失敗の記憶のキー `inboxDeadlineKey` を 2 か所で別々に作っていた | 関数の中に `triedSend{send, key}` を置き、`holdsBack` に渡した `key` をそのまま `recordFailure(t.key, now)` に渡す | 解消。`deadline.go` の `inboxDeadlineKey{` のリテラルは `key := ...` の 1 か所だけ |
| L4 | LOW | naming | `flushAlertNotifyBatch` と `flushAlertNotifications` が近い名前。`Escalate` と `NotifyInboxItem` の裸の `nil` | `flushDeadmanAlertNotifications` に改名し、doc に「flush して、失敗した append を報告する」と書いた。`nil` の 2 か所に「No batch: …」の comment を添えた | 解消。`flushAlertNotifications` を参照する本番コード・テストは残っていない(grep は旧レポート以外に当たらない)。`watch_recurrence.go` の `nil` にも同じ comment が先に付いていた |

## Evidence reviewed

読んだもの。

- `git show 1f2197dc`(13 ファイル、99 行追加、33 行削除)の全部。コード 6 ファイル、テスト 2 ファイル、director skill の 4 つの写し、テンプレート
- `internal/org/alert_notify.go`(121 行、全部)。初回から変わっていない
- `internal/org/escalate.go` の `git diff 94e61c38 HEAD` の全部(`sendInboxItemToHuman` の batch 引数、`WaitInbox` の可変長の `types`、`openInboxItems`、`ParseInboxWaitTypes`)
- `internal/org/deadline.go` の `SweepInboxDeadlines` の全体(直したあとの形)
- `internal/org/director_wait.go` の `directorWaitTimeoutDetail` と呼び出し
- `internal/cli/org_watch.go` の help と `TestOrgWatch_HelpDescribesTheAlertNotifyBatch`(`strings.Fields` で空白をそろえてから `Contains` で見るので、help の改行位置が変わっても通る形)
- 差分の追加行に不可視 Unicode(U+2028、U+200B 系、U+202A から U+202E、U+FEFF)と U+FFFD は無い(`git diff 94e61c38 HEAD` の追加行 すべてで 0 行)。`1f2197dc` の追加行で 90 桁を超えるのは help の文字列とテストの行だけで、コメントの行は 80 桁以内

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- 初回の「指摘なしで確かめたこと」の項目は、`1f2197dc` が触っていない部分(見張りを黙らせる道が `off` の 1 つだけ、失敗の勘定、batch に入らない道、文の値の守り、`ParseInboxWaitTypes`)について変わらない。`escalate.go` と `alert_notify.go` を読み直して同じ結果だった
- `1f2197dc` は `sendInboxItemToHuman` の流れも batch の扱いも変えていない。変えたのは `SweepInboxDeadlines` の記録の持ち方(キー)、watch の関数名と comment、`--types` の印の `detail`、文書の文言だけ
- `triedSend` の `key` は `holdsBack` を通った(held back でなかった)件のものだけが入る。held back の件は `tried` に入らず、`recordFailure` も呼ばれないので、前の版と同じ

読んでいない・実行していないもの。テスト、`go build`、`go vet`、linter は走らせていない(このレビューの対象外)。`TestDirectorWaitTimeoutDetail` の `leaderInboxTypes` は `escalate_test.go:1596` に定義があることだけ確かめた。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| - | - | 新しい指摘はない。初回の L1 から L4 は解消した | 上の「Earlier run」の表 | - |

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 0(初回の LOW 4 件はすべて解消)。

## Positive notes

- L1 の直しが、4 つの送りの一覧を 6 か所で同じ並びにそろえた。help のテストは新しい文を `Contains` で固定したので、help だけが次に食い違うことはない
- L3 の直しは、キーを持つ入れ物にして 2 か所の作りを 1 か所にした。欄が増えても `key := ...` だけを直せば足りる
- L2 の直しは、stderr の文(skill が一致を見る)に触らず、印の `detail` だけを変えた。直した理由をテストの comment に書いている(`director_wait_test.go:783-785`)

観察として残すもの(指摘にはしない)。

- `directorWaitTimeoutDetail` の doc の最後の文「The error's text, with inboxWaitTimeoutPhrase, is not this detail.」は、エラーの文が変わらない理由(director skill がその一致で再実行を決める)を言っていない。理由は `inboxWaitTimeoutPhrase`(`escalate.go:102`)の側の comment にあるので、直さなくても読み手は辿れる。次にこの関数を触るときに 1 句足すとよい
- org skill の表の 2 つの行(`watch` の行と `inbox` の行)は、ALERT の件の通知が「`off` なら出ない」とだけ書く。`inbox` の行には、サイクルの始めの再送と受信箱に書けないときの予備の道の例外が無い。`watch` の行は予備の道を「今までどおり」と別の文で書いている。4 つの送りの全部は同じ skill の 661-666 行にあるので、表は要約として許せる範囲と見て指摘にはしない
- 初回に書いた観察(通知の遅れの上限が計画の Risks の文より大きいこと、複数 org の ALERT が 1 回の掃き出しに入ると文が最も古い件の org だけを出すこと)は、そのまま。計画の本文は digest を保つため変えず、Progress の S5 の行に記録してある

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | - | - | - |

_(このレビューは `docs/tech-debt/README.md` を編集していない。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 0)
- Follow-ups: なし。観察の 2 つは任意で、直すならコメントの 1 句と表の 1 行だけ。ただし、どちらもコメントか文書の変更なので、パイプラインの 2 回目の run に入ることになる(cap 2 の最後の run)。ほかに直す理由ができたときにまとめるのが無駄が少ない
