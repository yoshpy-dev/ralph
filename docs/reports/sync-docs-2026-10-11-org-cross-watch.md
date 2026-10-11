# sync-docs report: org-cross-watch

## Latest run: cycle 2(HEAD aae0ee5c)

- Date: 2026-10-12(JST)
- Plan: `docs/plans/active/2026-10-11-org-cross-watch.md`(承認済み、digest `d0a6f2735552`。計画は触っていないので値は変わらない)
- Pipeline cycle: 2(上限 2)。差分は `577ebeaf` から branch HEAD `aae0ee5c`(feat/org-cross-watch)まで。cycle 1 の sync-docs(`0d161a80`)のあとに本番のコードを変えたのは、`49d8d605`(cross-review cycle 1 の #1・#2 の直し)と `d2a750ec`(self-review cycle 2 の C2-1〜C2-4 の直し)の 2 つ。`2ccdd11b` はテストだけ、`011f43b7` と `aae0ee5c` は記録だけで、`git diff d2a750ec..HEAD --stat -- . ':!docs' ':!*_test.go'` は空
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-11-org-cross-watch.md`(`e0ac493f`。HEAD `49d8d605` で実行。Merge 可、LOW 5 件の C2-1〜C2-5)、
  `docs/reports/verify-2026-10-11-org-cross-watch.md`(`b0a226a9`。HEAD `d2a750ec` で実行。pass、文書のずれ D1〜D7)、
  `docs/reports/test-2026-10-11-org-cross-watch.md`(`011f43b7`。pass。生き残った mutant 2 件は、`2ccdd11b` のテスト 2 本で塞いだ)
- 内容の commit: この report と insight event を含む同じ commit(`docs: sync docs for org-cross-watch (cycle 2)`)。cycle 1 のように内容と report を分けていない
- 項目の番号: 依頼の D1〜D6 は、verify の「Documentation drift」の D1〜D6 と同じ項目。verify の D7(evidence の `last cycle` の引用)は書き換えない(「Found but left」)

### Summary

verify と self-review が挙げた 6 か所(D1〜D6)をすべて直した。古くなっていたのは 2 点で、`49d8d605` の終わりの順(最後のサイクルのあと、watcher の呼び出しを待つ間も `watch.lock` と running の印を持ち、待ち終えてから stopped の印を書いてロックを放す)と、`d2a750ec` の印の時刻の意味(`ts` は見張りが running の印を最後に書いた時刻で、`ralph status` では `mark updated`、`--json` では `updated_at`)を書いた文である。直したのは `/org` skill の 4 面(`--stop` の待ち、`stale` の定義、`--stop` が時間切れになったあとの説明)、仕様 FR-6 の注、tech-debt の 2 行(202 行目と 204 行目)。README、AGENTS.md、`docs/quality/quality-gates.md` と配る側の写し、`agent-messaging.md`、`docs/recipes/` は、終わりの順も `last cycle` も書いていないので変更していない。コード、計画、ほかの report、evidence は触っていない。

差分の大きさ(`git diff 577ebeaf...HEAD --shortstat`、HEAD `aae0ee5c` 時点、この commit を含まない): 47 files changed, 8,979 insertions(+), 1,066 deletions(-)。`docs/` を除くと 36 files changed, 6,594 insertions(+), 1,064 deletions(-)、さらにテストファイルも除くと 24 files changed, 2,678 insertions(+), 924 deletions(-)。この commit の大きさは返信の `git show --stat HEAD` に書く。walkthrough は書いていない(`/pr` が決める)。

### Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md:176`(動詞表の `watch` の行)と 3 つの写し(D1) | `--stop` の説明を、「見張りがそのサイクルを終えて止まるまで最大 30 秒待つ」から、「見張りが止まるまで最大 30 秒待つ。見張りが止まるのは、そのサイクルを終え、実行中のオンデマンド watcher の呼び出しの待ちも終えてから」に直した。長時間動かしている見張りはその呼び出しを取り消すので待ちはすぐ終わり、`--once` の見張りは取り消さずに待つので最大でおよそ 70 秒(呼び出し 60 秒 + ALERT の送信 10 秒)かかる、と足した。根拠は `runOrgWatchAll`(`internal/cli/org_watch.go`)の `hooks.BeforeStop`(`--once` でなければ `cancelWatchers()`、そのあと `watcherWG.Wait()`)、`watcherInvokeTimeout`(60 秒、`internal/org/watcher.go`)、ALERT の送信の `DriverCallTimeout`(10 秒)。編集は `.claude/skills/org/SKILL.md` に 1 回だけ行い、`./scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り、`.claude` の方を `templates/base/.claude/skills/org/SKILL.md` に `cp` して、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で `templates/base/.agents/skills/org/SKILL.md` を作った。4 面は `cmp` で同一 |
| `.claude/skills/org/SKILL.md:775-786`(長時間運用の項)と 3 つの写し(D2・D3) | D2: 「古い」の定義を、「最後のサイクルの時刻が」から、「印の時刻(見張りが最後に running の印を書いた時刻。`ralph status` では `mark updated`)が」に直した。印の時刻(`--json` では `org_watch.updated_at`)は、最初のサイクルの前と各サイクルの終わりに書き直され、止まる前に watcher の呼び出しを待つ間も間隔ごとに書き直される、と足した。窓の長さの規則は cycle 1 のまま。D3: 「30 秒で止まらなくても頼みは残り、見張りはそのサイクルの終わりに止まる」を、「そのサイクルが終わり、実行中のオンデマンド watcher の呼び出しの待ちも終わると止まる」に直した。CLI のエラー文(`runOrgWatchStop`、`internal/cli/org_watch.go`)と同じ事実。新しい 12 行の表示幅は 57〜76(前後の行は 47〜77) |
| `docs/specs/2026-10-07-org-multi-org-director.md:63`(FR-6 の「6 段目の前半で決めたこと」。D4) | 2 か所。(1) `running` の説明の「最後のサイクルの時刻 `ts`」を、「見張りが印を最後に書いた時刻 `ts`」に直し、「最後のサイクルの時刻とは限らない」と足した。(2) `--stop` の文の直後に 2 文足した。見張りは最後のサイクルのあと、実行中のオンデマンド watcher の呼び出しを待つ間(`--once` でなければ先に取り消す)も `watch.lock` と running の印を持ち続け、印は間隔ごとに書き直す。待ち終えてから stopped の印を書いてロックを放すので、その間も 2 つ目の `watch --all` は断られ、`--stop` はロックが放されるまで待つ。窓の規則の説明と、要件の 59 行目は変えていない |
| `docs/tech-debt/README.md:204`(「Small leftovers of org-cross-watch」。D5) | (d) の括弧を、「once `RunWatchAll` returns」から「in `WatchHooks.BeforeStop`, before it writes the stopped mark and releases the lock」に直した。(e) の Trigger の「the next change to `runOrgWatchAll`」を、「the second half of stage 6 (the next PR, where `ralph org start` launches the watch and changes this path)」に替えた。`AcquireWatchLock` の断りの変更という 2 つ目の Trigger は残し、前の Trigger が `49d8d605` で発火したのに banner の位置は動かなかったことを、括弧の 1 句で書いた。コードは直していない。Related に `49d8d605` と self-review の `C2-5` を足した |
| `docs/tech-debt/README.md:202`(「Stopping the cross-org watch waits for the current cycle while herdr hangs」。D6) | 「a SIGINT, a SIGTERM, or a `--stop` request takes effect only when the cycle ends, which can take the number of hanging calls × 10 s」のあとに、「and then until the watch has waited for its in-flight on-demand watcher calls (`WatchHooks.BeforeStop`: they are cancelled at once, except that a `--once` watch stopped by `--stop` lets them finish, up to about 60 s + 10 s)」を足した。ほかの文は変えていない。SIGINT と SIGTERM は、watcher の呼び出しの context が `ctx` の子なので、`--once` でも呼び出しがすぐ取り消される。待ちが長くなるのは `--once` の見張りに `--stop` を打つ場合だけ、という区別を括弧に入れた |
| `docs/insights/events/2026-10-11-org-cross-watch.jsonl` | `sync_docs` の event を 1 行足した(`--slug org-cross-watch`、cycle 2、verdict pass、findings 0)。既存の event ファイルへの追記で、新しいファイルはできていない(`git status --short` で `M` を確認) |
| この report | 先頭に cycle 2 の節を置き、cycle 1 の内容を末尾の付録に移した(見出しを 1 段下げた以外は変えていない) |

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `/org` skill の動詞表の `watch` の行のうち `--stop` 以外 | `--all` 必須、`--org-id` の拒否、`--detach` の 10 秒、herdr・agmsg・git の呼び出しの 10 秒の打ち切り、2 つ目の `watch --all` の拒否、SIGINT と SIGTERM が「そのサイクルを終えてから止まる」ことが、コードと合っている。SIGINT と SIGTERM では watcher の呼び出しの context が `ctx` の子として先に取り消されるので、サイクルのあとの待ちはすぐ終わる。変更なし |
| `/org` skill の「受信箱」節、長時間運用の項の起動と同時起動の説明 | cycle 1 で確かめた内容から、このあと変わったコードは drain、止める頼みの取り出し、印の時刻の名前だけ。受信箱の期限、`watch.log`、`--detach` の記述は動かない。変更なし |
| 起動のときの古い止める頼みの取り出し(`takeWatchStopRequest` が `watch-stop.json` を `.watch-stop.json.taken` に rename する) | skill、仕様、README のどれも、台帳の下のファイルの一覧としては書いていない。利用者が打つ `--stop` の頼みは今までどおり `watch-stop.json`。一時ファイルの細部なので書き足していない |
| `README.md:124-125`、`:253`、`AGENTS.md:90`、`docs/quality/quality-gates.md:79` と `templates/base/` の写し(78 行目)、`.claude/rules/ralph/agent-messaging.md` と写し、`docs/recipes/`、`docs/specs/2026-08-01-org-runtime.md` | `last cycle`、`last_cycle_ts`、`--stop` の待ちの長さ、stopped の印は書いていないか、書いていても合っている。README の 253 行目の「rewrites its liveness mark」は毎サイクルの書き直しを言うだけで、drain の間の書き直しと矛盾しない。`ralph status` については `org watch:` の行と `org_watch` の名前だけを書いている。quality-gates の watch の行は 2 面で同じ文。変更なし |
| `ralph org watch --help`(`internal/cli/org_watch.go` の `Long`) | 「rewrites its liveness mark every cycle」「SIGINT or SIGTERM ends it after the current cycle」「wait up to 30s for it to end」は、drain の間の書き直しと watcher の待ちに触れないが、誤りではない。コードなので触っていない |
| `docs/evidence/org-cross-watch-live-2026-10-11.md` | dd63a414 のビルドの出力に `last cycle …` と `last_cycle_ts` が残る(verify の D7、14 か所)。tester が 612 行目に、d2a750ec で `mark updated …` と `updated_at` に名前が変わった注を足してある。そのビルドの記録としては正しいので書き換えない |
| 計画の Scope(45 行と 62 行)と AC | 「`ts`(最後のサイクルの時刻)」「最後のサイクルの時刻」が残る。承認 digest を保つため本文を変えない。Progress に d2a750ec のずれが書いてある(verify の「計画」の行)。計画は触っていない |
| tech-debt の 202・204 行目が名指しする関数とテスト | HEAD にある: `evaluateCycle`、`DriverCallTimeout`、`RunWatcher`、`TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall`(`internal/cli/org_watch_test.go`)、`TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly`(`internal/cli/org_test.go`)、`TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock`(`internal/cli/org_watch_test.go`)。`WatchHooks.BeforeStop` は `internal/org/watch.go` にある。`file:line` の参照は増やしていない |
| tech-debt のほかの watch の行(201 行目の status と deadman、205・206 行目の `--detach`) | `RunWatchAll returns`、`last cycle`、stopped の印、`watch-stop.json` を drain や印の時刻の意味と食い違う形で書いた箇所を `grep` で探して、なかった。変更なし |

### Found but left

- 計画の本文(Scope の 45 行と 62 行)の「最後のサイクルの時刻」と、Progress checklist の `/sync-docs` cycle 2 の記録の行: 承認 digest を保つため本文は触らない。Progress の記録は lead が書く約束なので足していない
- evidence の `last cycle` と `last_cycle_ts`(verify の D7): そのビルドの記録として正しい。注はすでに tester が足した
- test report cycle 2 の Test gaps のうち、台帳に入れていない 3 件。cycle 1 のときと同じく、台帳に足すかは lead の判断:
  - 取り出しと削除の間に見張りが殺されると、`.watch-stop.json.taken` が台帳に残る。次に止める頼みのある起動まで消えず、読まれないファイルなので動きには関わらない(実機 2)
  - `BeforeStop` が返らない場合は、doc の前提(上限のある時間で返す)だけで、仕組みでは打ち切らない(self-review の C2-3。doc の一文で直し、verify も妥当と判定した)
  - watcher の呼び出しの途中で `--stop` を打つ流れは CLI のテスト(stub)でだけ確かめてあり、後始末が 30 秒を超えて `--stop` が時間切れになる流れは文を見るだけ
- self-review cycle 2 の「直さない LOW は /sync-docs で tech-debt に 1 項目にまとめる」: C2-1〜C2-4 は `d2a750ec` で直り(verify が確認)、C2-5 は 204 行目の直しそのものなので、足す項目はない
- この cycle の `/cross-review` への注意(verify の依頼どおり、sync-docs では扱えない): コードのコメントが指す `docs/reports/cross-review-triage-org-cross-watch.md` の cycle 1 の節を、triage を書き直すときに残すこと

### Verification

実行した結果(`./scripts/run-verify.sh` と 2 つの check は、skill・仕様・tech-debt の編集のあと、report と insight event を書く前の作業ツリーで実行した):

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | rc 0(`[ok] check-skill-sync: 13 skill(s) in lock-step`) |
| `./scripts/check-sync.sh` | rc 0(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5、`PASS: all files in sync.`) |
| `cmp` で `/org` skill の 4 面 | 同一 |
| `go test ./internal/... -count=1 -run 'Skill\|Sync\|Template\|Defaults'` | rc 0(8 パッケージが ok。一致するテストがないパッケージは `[no tests to run]`) |
| `./scripts/run-verify.sh`(full) | rc 0(`==> All verifiers passed.`。shellcheck と hook の `sh -n`、settings の `jq -e`、Codex の 3 つの guard、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の参照、`tests/*.sh`、Go は `gofmt: ok`・`0 issues.`・8 パッケージ ok、branch の secret scan は `577ebeaf..aae0ee5c` で clean)。ログは `docs/evidence/verify-2026-10-11-233430.log`(gitignore の対象) |
| 計画した `old_string` の数(scratchpad の `anchors.py`) | 14 組(skill の 2 組 × 4 面、仕様 2、tech-debt 4)がどれも 1 回だけ |
| tech-debt の表の検査(scratchpad の `tdcheck.py`。`git show HEAD:` と比べる) | 変えた行は 202 と 204 の 2 行だけ。どちらも 7 つに分かれる(5 列)、列ごとのバッククォートは偶数、`file:line` の参照は 0 のまま |
| `git diff d2a750ec..HEAD --stat -- . ':!docs' ':!*_test.go'` | 空(verify が見たあとに、本番のコードは変わっていない) |

### Not verified

- `/org` skill と仕様の日本語は、natural-japanese の lint を手では回していない。PostToolUse の hook からも指摘は返らなかった。直した箇所は skill の 13 行(動詞表の 1 行と、長時間運用の項の 12 行。4 面とも同じ)と、仕様の 2 か所(1 句と 2 文)
- `--stop` の待ちの長さ(`--once` でなければ取り消してすぐ終わる、`--once` は最大およそ 70 秒)は、コードの読みと、test report が挙げる CLI のテスト(`TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall`、`TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds`)に頼っている。実機で 70 秒を測ってはいない(test report の Test gaps も同じ)
- `/cross-review` と `/pr` は、この cycle ではまだ走っていない

## 付録: cycle 1 の結果

- Date: 2026-10-11(JST)
- Plan: `docs/plans/active/2026-10-11-org-cross-watch.md`(承認済み、digest `d0a6f2735552`。計画は触っていないので値は変わらない)
- Pipeline cycle: 1(上限 2)。差分は `577ebeaf` から branch HEAD `ed1636cb`(feat/org-cross-watch)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-11-org-cross-watch.md`(`6a6976d2`、`7afda04c` で再実行の結果に更新。Merge 可、再実行は LOW 5 件の N1〜N5)、
  `docs/reports/verify-2026-10-11-org-cross-watch.md`(`a411bb57`。pass、文書のずれ D1 と D2)、
  `docs/reports/test-2026-10-11-org-cross-watch.md`(`fd35c1eb`、生き残った mutant を塞ぐテストの追記は `4aaef4ed`。pass、「参考」に 2 件)
- 内容の commit: `04bcb342`(skill 4 面・仕様・tech-debt)。この report と insight event は、その次の commit に入れる

### Summary

古くなっていたのは、`/org` skill の生存の印の窓(verify の D1)と、仕様 FR-6 の 6 段目の注(D2)だった。tech-debt には、test report の「参考」にあった 2 件を足した。README、AGENTS.md、`docs/quality/quality-gates.md` と配る側の写し、`agent-messaging.md` はコードと合っていて変更していない。コード、計画、ほかの report は触っていない。

`94d5248c` 以降に入ったコミットは、テストと記録だけで、本番のコードは変わっていない(`git diff 94d5248c..HEAD --stat` で確認)。verify が見た時点のコードと今の HEAD の文書の突き合わせは、そのまま有効。

差分の大きさ(`git diff 577ebeaf...HEAD --shortstat`、`04bcb342` 時点、この report の commit を含まない): 45 files changed, 7338 insertions(+), 1064 deletions(-)。`04bcb342` 単体は 6 files changed, 24 insertions(+), 15 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

### Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md` と 3 つの写し(D1) | 長時間運用の項の `stale` の説明を直した。「`[org].deadman_minutes` より古い」を、「最後のサイクルの時刻が、`[org].deadman_minutes`(1 未満なら 10 分)と見張りの間隔の 2 倍のうち長いほうより前」にした。コードの `watchAliveWindow`(`internal/org/watchalive.go`)と同じ規則。4 行が 6 行になり、表示幅は 36〜75(前後の行は 63〜76)。編集は `.claude/skills/org/SKILL.md` に 1 回だけ行い、`./scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り、`.claude` の方を `templates/base/.claude/skills/org/SKILL.md` に `cp` して、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で `templates/base/.agents/skills/org/SKILL.md` を作った。4 面は `cmp` で同一 |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-6 の 6 段目の注。D2) | `running` を「ロックあり、印が running で `ts` が窓の内」、`stale` を「窓より古い」に直し、「出す。」の直後に窓の規則を足した。窓は `deadman_minutes` と印の `interval_seconds` の 2 倍の大きいほう。`deadman_minutes` が 1 未満のときは既定の 10 に読み替える。`deadman_minutes = 0` は「デッドマンを時間切れで上げない」の意味で、そのまま窓にすると 0 を使う台帳の健全な見張りがすべて `stale` になり、`--detach` が起動した子を止めてしまうことも書いた。要件(59 行)の「印が古いとき(`deadman_minutes` より前)」の本文は変えず、注で「古い」はこの窓で読むこと、後半の PR の拒否も同じ判定関数 `ProbeWatch` を使うことを書いた |
| `docs/tech-debt/README.md`(新しい行。test report の「参考」の 2 つ目) | 同時の `ralph org watch --all --detach` が、見張りが動いているのに終了コード 1 になる件。ロックを取れなかった子が 2 秒待って `another ralph org watch --all is running` を `watch.log` に書いて終わり、その `--detach` は自分の子が先に終わったので `the watch (pid N) exited before it was running` で終了コード 1 になる(`runOrgWatchDetach` の `child.exited` の場合)。実機で 8 本を同時に 4 回打ち、どの回も動く見張りは 1 つで、`started` 1 本、`already running` 5〜6 本、終了コード 1 が 1〜2 本。Trigger は、後半の PR で `ralph org start` が見張りを起動するとき「起動が終了コード 1 でも `ProbeWatch` が running なら成功」と扱うこと。`runOrgWatchDetach` 側で、自分の子が終わって別の見張りが running なら `already running` で終了コード 0 にする案も書いた(`TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild` が今の終了コード 1 を固定しているので、そのテストも変わる)と、`--detach` が失敗したのに `ralph status` が running の報告 |
| `docs/tech-debt/README.md`(「Small leftovers of org-cross-watch」の行に (e)。「参考」の 1 つ目) | 2 つ目の `watch --all` が、断られる前に起動の banner(`watching every org (...)`)を stdout に出す件。`runOrgWatchAll` が banner を出してから `RunWatchAll` を呼び、`RunWatchAll` がロックを取って断るため。ロックの競争に負けた `--detach` の子は、banner と断りの文を `watch.log` に書く。4 列(Debt、Impact、Why deferred、Trigger)と Related を (e) で伸ばし、行の最初の文に「(e) is the order of two lines of output」を足した。Trigger は、`runOrgWatchAll` か `AcquireWatchLock` の断りの次の変更で、ロックを取ったあとに banner を出す(`WatchAllParams` の callback を `RunWatchAll` がロックのあとに呼ぶ)ことと、`TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock` に stdout に banner がないことを足すこと |
| `docs/tech-debt/README.md`(status と見張りの deadman が食い違う行) | 取り消し線の中の「RESOLVED 2026-10-11 in feat/org-cross-watch by the self-review M1 fix」に commit `1b64ae2b` を足した |
| `docs/insights/events/2026-10-11-org-cross-watch.jsonl` | `sync_docs` の event を 1 行足す(`--slug org-cross-watch`、cycle 1、verdict pass)。既存の event ファイルへの追記で、新しいファイルはできていない(`git status --short` で確認) |

### Surfaces checked for drift

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

### Found but left

- 計画の Progress checklist には、`/sync-docs` を終えた記録の行がない。ほかの計画では、post-impl の行(re-review、verify、test、sync-docs)を `Implementation started` の下位の行に書いている。依頼が計画を触らないことだったので足していない。足すかは lead の判断。`PR created` と AC の `[ ]` は `/pr` の分
- test report の Test gaps のうち、台帳に入れたのは「参考」の 2 件と、すでにある status の deadman の件だけ。残り(本物の osascript の通知を切り離した子から出すこと、Linux の CI での SID、本物の herdr が返らない場合、`ralph status` の共有ロックと見張りの起動の取り合いを別プロセスで何度も試すこと)は、test report の Test gaps にだけある。台帳に足すかは lead の判断
- 同時の `--detach` の終了コード 1 は、計画の AC12(子が先に終われば終了コード 1)のとおりの動きで、テスト(`TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild`)も固定している。台帳の行は、直す場所を後半の PR の start 側と `--detach` 側の 2 つ挙げたが、どちらにするかは決めていない

### Verification

実行した結果(`./scripts/run-verify.sh` と 2 つの check は、commit 前の作業ツリーで実行した。内容は `04bcb342` と同じ。`secret-scan-branch.sh --strict` は commit 後に実行した):

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | rc 0(`[ok] check-skill-sync: 13 skill(s) in lock-step`) |
| `./scripts/check-sync.sh` | rc 0(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5、`PASS: all files in sync.`) |
| `cmp` で `/org` skill の 4 面 | 同一 |
| `./scripts/run-verify.sh` | rc 0(`==> All verifiers passed.`。shellcheck と hook の `sh -n`、settings の `jq -e`、Codex の 3 つの guard、check-sync、tech-debt の参照、Go は `gofmt: ok`・`0 issues.`・8 パッケージ ok、branch の secret scan は clean) |
| `./scripts/secret-scan-branch.sh --strict` | rc 0(`scanned 577ebeaf..04bcb342 against origin/main: clean`) |
| tech-debt の表の検査(scratchpad の script。`git show HEAD:` と比べる) | 変えた行は 3 つ(status と deadman の行、Small leftovers の行、新しい行)で、どれも 7 つに分かれる(5 列)、列ごとのバッククォートは偶数、`file:line` の参照は 0 |

### Not verified

- `/org` skill の日本語の文は、表示幅を数えただけで、読みやすさの lint(natural-japanese)は通していない。直した文は 3 行(窓の規則)だけ
- 後半の PR の拒否が `ProbeWatch` の窓を使うことは、仕様の注と計画の Open questions に書いてあるが、まだコードにない
