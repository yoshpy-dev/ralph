# Walkthrough: 見張りの ALERT を 1 件にまとめ、leader の stall と予約の中の scope_change を上げない(watch-alert-dedupe)

- 計画: `docs/plans/archive/2026-11-12-watch-alert-dedupe.md`(PR の作成時に archive へ移す)
- issue: #307 の前半。issue の受け入れ条件 1〜3 をこの PR で満たし、4(`ralph org wait --inbox` で ALERT を除いて待つ)と desktop notification のまとめは後半の PR(#307b)に回す
- 差分: `git diff 028a8f6d...HEAD`(34 ファイル、約 +5,600 / −170。うちテストを除く Go は約 +1,350、Go のテストは約 +3,300)

差分の大半はテストと記録なので、次の順に読むと仕組みが追いやすい。

## 1. 続きの扱い(読む価値が一番高い)

- `internal/org/watch_recurrence.go`: (org, 座席, 条件) ごとの「続き」の判定。冒頭の関数ほど上位にある。
  - `recordRecurrence`: 条件が立ち上がったとき、同じ鍵の pending の ALERT があればその回数を足し(`foldIntoPendingAlert`)、受信箱に片付いていない ALERT の件があれば `recurred` の event を足す。どちらも無いときだけ、呼び手が今までどおり新しい ALERT を送る。失敗はすべて新しい ALERT の側に倒れる。
  - `foldIntoPendingAlert`: 畳む前に、leader がその ALERT のあとに動いたかを `leaderActedSinceAlert`(`checkDeadman` と共有)で確かめる。動いていれば、その続きは終わったものとして新しい ALERT にする(cycle 1 の cross-review の指摘 S8)。
  - `reconcilePendingAlertsWithInbox`: 周期の初め、座席の評価の前に、受信箱がすでに持つ pending(件の alert_id か recurred の event の alert_id)を `escalateAlert` に通して片付ける。件を作った直後、状態の保存前に止まった watch の残りが、新しい発火を飲み込まないようにする(run 3 の cross-review の指摘 S12)。
- `internal/org/watch.go`: `raiseOrClear` が `recordRecurrence` を呼ぶ。`escalateToInbox` は、同じ鍵の片付いていない件があれば新しい件の代わりに `recurred` を足し(D2)、pending のあいだに貯めた回数を作った件へ移す(AC4)。`checkDeadman` は活動の判定を `leaderActedSinceAlert` に移した。
- `internal/org/inbox.go`: event `recurred`(alert_id、count、note)、件の `recurrences` と `last_recurred_at`、`AppendRecurred`(`inbox.lock` の下で読んで決めて足す。resolved の件は `ErrInboxResolved`)。期限(`deadline.go`)は回数を読まない。
- テスト: `internal/org/watch_test.go` の `TestWatch_Recurrence_*` と、既存の 2 本(`TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`、`TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess`)。後の 2 本は D1 で期待が変わった(計画の Progress に理由)。

## 2. 待っている leader の stall

- `internal/org/watch.go` の `evaluateCycle` と `evaluateSeat`: leader 以外の座席を先に評価し、`herdr agent get` が `working` で文字列が前の周期から変わった座席がいれば、その周期の leader の stall を上げない。受信箱にその org の片付いていない QUESTION・BLOCKED・RESULT があるときも上げない(D3、D4)。受信箱が読めないときは上げる側。
- テスト: `TestWatch_LeaderStall_*`(5 本)。herdr が `working` のまま文字列の固まった座席(#308)は数えないことも見ている。

## 3. 予約での scope_change

- `internal/org/watch_scope.go`: 予約を持つ org だけの道(`evaluateReservedScope`)。
  - パスは `git status --porcelain=v1 -z --untracked-files=all`(rename の両側)と、前の周期の HEAD から今の HEAD までの commit から集める。commit は `git -c diff.relative=false log --no-show-signature --cc --no-renames --name-only -z --format= <前>..<今> --not <origin の既定のブランチ> --` で、upstream から rebase や merge で入った変更は数えず、merge 自身の編集(衝突の解決)は数える。既定のブランチが分からないときは `git diff` に戻る。
  - 予約に含まれるか(既存の `reservePathsOverlap`)と、その org が新しく作った計画と報告(`docs/plans/` と `docs/reports/` の下、`templates/` の外、分岐点の tree に無いもの)を除く。分岐点が分からないときは除かない。
  - 立てるのは、前の周期に無かった予約の外のパスが出たときか、予約の外を触る commit があったとき。本文には予約の外のパスだけを載せる。
  - 予約を持たない org は今のまま(`GitStatus` の差し替え口と生の porcelain の比較)。
- テスト: `internal/org/watch_scope_test.go`。実際の git の一時 repo で、rename の commit、rebase と merge、指す先の無い `origin/HEAD`、`diff.relative=true` と下のディレクトリの cwd、役割 prompt と除く規則の照合(AC11)を見ている。

## 4. CLI と文書

- `internal/cli/org_inbox.go`: `ralph org inbox show` の `recurred: N times (last ...)`(1 回なら `1 time`)と、event の行の `alert_id=`・`count=`。
- `internal/cli/org_watch.go`: `ralph org watch --all` の help に 3 つの規則(テストで文を固定)。
- 文書: `.claude/rules/ralph/agent-messaging.md`(と templates/base の写し)、`docs/specs/2026-10-07-org-multi-org-director.md` の FR-3 と FR-6 の注、`/org` skill(4 つの写し)、README、AGENTS.md、`docs/quality/quality-gates.md` と `docs/recipes/worktrees.md`(写しつき)。
- `docs/tech-debt/README.md`: RC3 を RESOLVED にし、この PR の既知の穴を「Eleven gaps」の行にまとめた(落ちうる窓 W1〜W9、テストの穴、コメントのずれ)。

## 5. 記録

- 計画の Progress に S1〜S12 と、4 回のパイプラインの経過(cross-review の指摘と直し、上限を 2 から 3 に上げた理由)がある。
- 報告: `docs/reports/self-review-…`、`verify-…`、`test-…`、`sync-docs-…`(どれも最後の run で上書きし、前の run の ID を残している)、`cross-review-triage-watch-alert-dedupe.md`。

## 見どころ

- 落ちうる窓のうち、黙る向きのものは W9(1 周期の中で受信箱が一時的に読めないとき)だけで、ほかは二重か回数のずれ(知らせる側)。一覧は tech-debt の行の (9)。
- 機能の org の 3 席は 1 つの worktree を共有するので、予約の外の変更は最大 3 件になる(既知の穴)。
- 下流の 95 件の再発の仕組み(herdr の応答の揺れで stall が解除され、約 2 周期後にまた立つ)は推測で、下流の watch-status は見ていない。この PR はどちらの原因でも件が増えない形にした。実機の herdr での件の数え直しはしていない。
