# sync-docs report: org-cross-watch

## Cycle 1

- Date: 2026-10-11(JST)
- Plan: `docs/plans/active/2026-10-11-org-cross-watch.md`(承認済み、digest `d0a6f2735552`。計画は触っていないので値は変わらない)
- Pipeline cycle: 1(上限 2)。差分は `577ebeaf` から branch HEAD `ed1636cb`(feat/org-cross-watch)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-11-org-cross-watch.md`(`6a6976d2`、`7afda04c` で再実行の結果に更新。Merge 可、再実行は LOW 5 件の N1〜N5)、
  `docs/reports/verify-2026-10-11-org-cross-watch.md`(`a411bb57`。pass、文書のずれ D1 と D2)、
  `docs/reports/test-2026-10-11-org-cross-watch.md`(`fd35c1eb`、生き残った mutant を塞ぐテストの追記は `4aaef4ed`。pass、「参考」に 2 件)
- 内容の commit: `04bcb342`(skill 4 面・仕様・tech-debt)。この report と insight event は、その次の commit に入れる

## Summary

古くなっていたのは、`/org` skill の生存の印の窓(verify の D1)と、仕様 FR-6 の 6 段目の注(D2)だった。tech-debt には、test report の「参考」にあった 2 件を足した。README、AGENTS.md、`docs/quality/quality-gates.md` と配る側の写し、`agent-messaging.md` はコードと合っていて変更していない。コード、計画、ほかの report は触っていない。

`94d5248c` 以降に入ったコミットは、テストと記録だけで、本番のコードは変わっていない(`git diff 94d5248c..HEAD --stat` で確認)。verify が見た時点のコードと今の HEAD の文書の突き合わせは、そのまま有効。

差分の大きさ(`git diff 577ebeaf...HEAD --shortstat`、`04bcb342` 時点、この report の commit を含まない): 45 files changed, 7338 insertions(+), 1064 deletions(-)。`04bcb342` 単体は 6 files changed, 24 insertions(+), 15 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md` と 3 つの写し(D1) | 長時間運用の項の `stale` の説明を直した。「`[org].deadman_minutes` より古い」を、「最後のサイクルの時刻が、`[org].deadman_minutes`(1 未満なら 10 分)と見張りの間隔の 2 倍のうち長いほうより前」にした。コードの `watchAliveWindow`(`internal/org/watchalive.go`)と同じ規則。4 行が 6 行になり、表示幅は 36〜75(前後の行は 63〜76)。編集は `.claude/skills/org/SKILL.md` に 1 回だけ行い、`./scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り、`.claude` の方を `templates/base/.claude/skills/org/SKILL.md` に `cp` して、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で `templates/base/.agents/skills/org/SKILL.md` を作った。4 面は `cmp` で同一 |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-6 の 6 段目の注。D2) | `running` を「ロックあり、印が running で `ts` が窓の内」、`stale` を「窓より古い」に直し、「出す。」の直後に窓の規則を足した。窓は `deadman_minutes` と印の `interval_seconds` の 2 倍の大きいほう。`deadman_minutes` が 1 未満のときは既定の 10 に読み替える。`deadman_minutes = 0` は「デッドマンを時間切れで上げない」の意味で、そのまま窓にすると 0 を使う台帳の健全な見張りがすべて `stale` になり、`--detach` が起動した子を止めてしまうことも書いた。要件(59 行)の「印が古いとき(`deadman_minutes` より前)」の本文は変えず、注で「古い」はこの窓で読むこと、後半の PR の拒否も同じ判定関数 `ProbeWatch` を使うことを書いた |
| `docs/tech-debt/README.md`(新しい行。test report の「参考」の 2 つ目) | 同時の `ralph org watch --all --detach` が、見張りが動いているのに終了コード 1 になる件。ロックを取れなかった子が 2 秒待って `another ralph org watch --all is running` を `watch.log` に書いて終わり、その `--detach` は自分の子が先に終わったので `the watch (pid N) exited before it was running` で終了コード 1 になる(`runOrgWatchDetach` の `child.exited` の場合)。実機で 8 本を同時に 4 回打ち、どの回も動く見張りは 1 つで、`started` 1 本、`already running` 5〜6 本、終了コード 1 が 1〜2 本。Trigger は、後半の PR で `ralph org start` が見張りを起動するとき「起動が終了コード 1 でも `ProbeWatch` が running なら成功」と扱うこと。`runOrgWatchDetach` 側で、自分の子が終わって別の見張りが running なら `already running` で終了コード 0 にする案も書いた(`TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild` が今の終了コード 1 を固定しているので、そのテストも変わる)と、`--detach` が失敗したのに `ralph status` が running の報告 |
| `docs/tech-debt/README.md`(「Small leftovers of org-cross-watch」の行に (e)。「参考」の 1 つ目) | 2 つ目の `watch --all` が、断られる前に起動の banner(`watching every org (...)`)を stdout に出す件。`runOrgWatchAll` が banner を出してから `RunWatchAll` を呼び、`RunWatchAll` がロックを取って断るため。ロックの競争に負けた `--detach` の子は、banner と断りの文を `watch.log` に書く。4 列(Debt、Impact、Why deferred、Trigger)と Related を (e) で伸ばし、行の最初の文に「(e) is the order of two lines of output」を足した。Trigger は、`runOrgWatchAll` か `AcquireWatchLock` の断りの次の変更で、ロックを取ったあとに banner を出す(`WatchAllParams` の callback を `RunWatchAll` がロックのあとに呼ぶ)ことと、`TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock` に stdout に banner がないことを足すこと |
| `docs/tech-debt/README.md`(status と見張りの deadman が食い違う行) | 取り消し線の中の「RESOLVED 2026-10-11 in feat/org-cross-watch by the self-review M1 fix」に commit `1b64ae2b` を足した |
| `docs/insights/events/2026-10-11-org-cross-watch.jsonl` | `sync_docs` の event を 1 行足す(`--slug org-cross-watch`、cycle 1、verdict pass)。既存の event ファイルへの追記で、新しいファイルはできていない(`git status --short` で確認) |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `/org` skill の動詞表の `watch` の行(176 行目) | `--all` 必須、`--org-id` の拒否、`--detach` の 10 秒(`orgWatchDetachDeadline`)、`--stop` の 30 秒(`orgWatchStopDeadline`)、`watch-alive.json`・`watch-stop.json`・`watch.log`、herdr・agmsg・git の呼び出しの 10 秒の打ち切りが、コードと合っている。変更なし |
| `/org` skill の「受信箱」節 | ack の期限は open で `notified` がない件、resolve の期限は resolved でない件、理由 `inbox_ack_deadline` と `inbox_resolve_deadline`(`internal/org/deadline.go`)、banner の `ORG INBOX DEADLINE:`、デスクトップ通知の題 `ralph org watch`(`inboxNotifyTitleWatch`)、`escalated_at` を読めない件を 1 回だけ警告して飛ばすこと、書けなかった通知を `ack_minutes` に 1 回までしか送り直さないことが、コードと合っている。変更なし |
| `/org` skill の長時間運用の項(`stale` の窓以外) | `--stop` が 30 秒で終わらなくても頼みは残り、そのサイクルの終わりに止まること、2 つ目の `watch --all` が断られること、見張りを起動しないのは今は start も spawn も同じことが、コードと計画の Non-goals に合っている |
| `README.md` の org の動詞の表(124 行)、`ralph status` の行(125 行)、「Org runtime」節の watch の段落(253 行) | `--all`、`--detach`、`--stop`、`--org-id` の廃止、`ack_minutes` の 10 と `resolve_minutes` の 60、`org watch:` の行と `org_watch` が合っている。窓の規則は書いていない。変更なし |
| `AGENTS.md` の repo map の `internal/org/` の行(90 行目) | `RunWatchAll`(`watch.go`)、`watch.lock`、`watch-alive.json`(`watchalive.go`)、`deadline.go` が入っている。変更なし |
| `docs/quality/quality-gates.md` と `templates/base/` の写し | watch の行は 2 面で同じ文(`diff` で確認)。`check-sync.sh` の KNOWN_DIFF の対象ファイルだが、この行の差はない。変更なし |
| `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し | `watchdog` の identity と `TO: leader` の記述は `--all` でも正しい。`--org-id` の形を書いていない。変更なし |
| `ralph org watch --org-id` の残り | `git grep` で、計画・report・evidence・テストを除くと、廃止を説明する文(skill の 4 面、README、仕様の注)、断りの文(`internal/cli/org_watch.go`)、コメント(`internal/org/watch.go`)だけ。古い使い方を書いた箇所はない |
| 仕様 FR-6 の要件(59 行)、背景の 172 行(`SendWatchdogAlert`) | 要件の本文は変えない。172 行の `SendWatchdogAlert` は今もあり、`SendWatchdogAlertBounded` が隣に足されただけなので、書いてあることは正しい |
| tech-debt の S0 で閉じた行(`inboxSummary`、`wireOrgInbox`、`PrintableInboxText`、`EscalationsPathIn`、`org.go` の 1,544 行) | HEAD と照合した。`internal/cli/org.go` は 1,544 行、`inboxSummary` は `internal/cli/org_inbox.go`、`wireOrgInbox` は `internal/cli/org.go`、`PrintableInboxText` は `internal/org/inbox.go`、`TestPrintableInboxText` は `internal/org/escalate_test.go` に残っている、`newWatchAllRun` は `EscalationsPathIn(p.StatusDir)` を使う。`RunWatch` を名指しした箇所は、取り消し線の中の元の文と、閉じた文の「もうない」だけ。`WatchParams` を名指しした行はない。`sendAlert` を挙げた古い行(81 行目)は名前だけの参照で、ctx の引数が外れても正しい。変更なし |
| tech-debt の新しい行(status と deadman、停止が cycle の終わりまで待つ、古い comment、Small leftovers の (a)〜(d)、切り離した子の cwd) | 挙げた関数とテストが HEAD にある: `statusDeadmanMinutes`、`evaluateCycle`、`pulseOrg`、`unreadableWarned`、`readOrgWatchHeartbeat`、`RunWatcher` の `watcherErrorReason`、`TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall`、`TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly`、`ensureWatchdogJoined`(`os.Getwd()`)、`orgWatchChildArgs`、`runOrgWatchAll` の「最初のシグナルのあとにシグナルの既定の動きを戻す」処理。`newOrgRuntimeAt` の doc は今も send と watch のコマンドを名指ししている。`file:line` の参照は、増えた行にも、ほかの行にもない。変更なし |
| `ralph.toml` の雛形、`OrgConfig.DeadmanMinutes` の doc、`ralph status` の stale の文言 | `94d5248c` が窓の規則と 0 の意味に合わせてある(verify が確認)。skill と仕様が、これでコードの doc と同じ規則になった |
| 計画の Scope(47 行)と AC11(133 行) | 古い窓の規則(`deadman_minutes` 以内)のまま。承認 digest を保つため本文は変えない。ずれは Progress の self-review の行(M1)に書いてあり、verify も妥当と判定した。計画は触っていない |

## Found but left

- 計画の Progress checklist には、`/sync-docs` を終えた記録の行がない。ほかの計画では、post-impl の行(re-review、verify、test、sync-docs)を `Implementation started` の下位の行に書いている。依頼が計画を触らないことだったので足していない。足すかは lead の判断。`PR created` と AC の `[ ]` は `/pr` の分
- test report の Test gaps のうち、台帳に入れたのは「参考」の 2 件と、すでにある status の deadman の件だけ。残り(本物の osascript の通知を切り離した子から出すこと、Linux の CI での SID、本物の herdr が返らない場合、`ralph status` の共有ロックと見張りの起動の取り合いを別プロセスで何度も試すこと)は、test report の Test gaps にだけある。台帳に足すかは lead の判断
- 同時の `--detach` の終了コード 1 は、計画の AC12(子が先に終われば終了コード 1)のとおりの動きで、テスト(`TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild`)も固定している。台帳の行は、直す場所を後半の PR の start 側と `--detach` 側の 2 つ挙げたが、どちらにするかは決めていない

## Verification

実行した結果(`./scripts/run-verify.sh` と 2 つの check は、commit 前の作業ツリーで実行した。内容は `04bcb342` と同じ。`secret-scan-branch.sh --strict` は commit 後に実行した):

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | rc 0(`[ok] check-skill-sync: 13 skill(s) in lock-step`) |
| `./scripts/check-sync.sh` | rc 0(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5、`PASS: all files in sync.`) |
| `cmp` で `/org` skill の 4 面 | 同一 |
| `./scripts/run-verify.sh` | rc 0(`==> All verifiers passed.`。shellcheck と hook の `sh -n`、settings の `jq -e`、Codex の 3 つの guard、check-sync、tech-debt の参照、Go は `gofmt: ok`・`0 issues.`・8 パッケージ ok、branch の secret scan は clean) |
| `./scripts/secret-scan-branch.sh --strict` | rc 0(`scanned 577ebeaf..04bcb342 against origin/main: clean`) |
| tech-debt の表の検査(scratchpad の script。`git show HEAD:` と比べる) | 変えた行は 3 つ(status と deadman の行、Small leftovers の行、新しい行)で、どれも 7 つに分かれる(5 列)、列ごとのバッククォートは偶数、`file:line` の参照は 0 |

## Not verified

- `/org` skill の日本語の文は、表示幅を数えただけで、読みやすさの lint(natural-japanese)は通していない。直した文は 3 行(窓の規則)だけ
- 後半の PR の拒否が `ProbeWatch` の窓を使うことは、仕様の注と計画の Open questions に書いてあるが、まだコードにない
