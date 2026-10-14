# Self-review report: org-director-attach

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-14-org-director-attach.md(承認済み、digest 27414f27525b。8 段目の 8a-1)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)
- Scope: diff の品質だけ。`git diff ef575748...HEAD`(HEAD 8093d47b、10 コミット、49 ファイル、+6254/-654)。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない。計画の Design decisions D1〜D9 と Progress checklist にある決定(approve を記録だけにする D7、登録に `id` を持たせる D8、`enabled` を main worktree の規則で読む D6、計画からのずれ 3 件)は、欠陥として数えていない。ずれ 3 件のうち、テストの意図を変えた 2 件(1)(3)は「テストの意図の変更」の節で判断を書いた

## Evidence reviewed

- 読んだ差分(コード、全体): `internal/org/director.go`(453 行すべて)、`internal/org/escalate.go`・`inbox.go`・`deadline.go`・`approval.go`・`spawn.go` の hunk、`internal/org/approve.go` の hunk(全部)、`internal/cli/org_director.go`(459 行すべて)、`internal/cli/org.go`・`org_inbox.go`・`org_approve.go`・`org_legacy_ledger.go`・`status.go` の hunk、`internal/config/config.go` の hunk、`scripts/ralph-config.sh` と `templates/base/ralph.toml` の hunk
- 読んだ差分(文書): `/director` skill(233 行すべて)、`/org` skill の変更行すべて、`internal/org/prompts/leader.md` の 2 か所、README の 5 か所、`docs/quality/quality-gates.md` の 2 行(root と template の hunk は同一)、仕様の 5 か所(word diff)、AGENTS.md、`docs/tech-debt/README.md`(変更 8 行、追加 2 行、削除 1 行。`|` で分けた列は全行 5 で、見出しと合う)
- 読んだテスト: `internal/org/approve_test.go` の diff の全行、`internal/cli/org_approve_test.go` の削除行すべてと `PassesWithAnInboxItCannotWrite`、`status_merge_guarantee_test.go` の diff、`status_test.go` の fallback の test、`internal/cli/org_director_test.go` の helper(1〜102 行)、`director_test.go`・`escalate_test.go`・`org_inbox_test.go`・`deadline_test.go` の test 名の一覧。残りは読んでいない
- 読んだ周辺のコード(定義と呼び出しを確かめるため): `withFileLock`・`openLockFile`(`lockfile.go`)、`LedgerMainWorktreeRoot`・`gitMainWorktree`(`statedir.go`)、`WatchAcceptsWork`・`WatchLiveness*`(`watchgate.go`、`watchalive.go`)、`SweepInboxDeadlines`、`config.Load`(`Default()` に重ねて decode する)、`ParsePlanHeader`・`parsePlanRef`・`normalizePlanPath`(plan のパスに制御文字は入らない)、`orgWatchLine`、`wait --inbox` の help(`--timeout-ms 0` は無期限)
- 確かめたこと
  - `git diff --check` は空。追加行に TODO・FIXME・debug 出力・秘密らしい文字列・`/Users/` はない(`secret` は test の定数名と本文だけ)
  - skill は `/org` と `/director` のどちらも 4 面が `cmp` で同一。`ralph-workflow.md`、`scripts/ralph-config.sh` は root と template が同一。`quality-gates.md` は root と template が元から違うので、この PR の hunk 同士を比べて同一だった。`/director` skill に `internal/`・`cmd/`・段の名前はない
  - 並行性: `Attach`・`Detach` は `director.lock` の下で読み直してから決める。`--replace` の `detached` と `attached` は 1 回のロックの中。`id` の割り当ては、raw テキストの regex(`"id":"d<N>`、閉じ引用符なし)と decode した `id` の両方の最大値に 1 を足し、JSON のエスケープされた引用符の中の `"id"` には当たらない。ロックの取り方は `inbox.go` と同じ形で、manifest や inbox のロックを同時に持たない
  - escalate の 3 つの場合: 検査で断られた message は probe を打たず(`TestEscalate_RefusalProbesNoWatch`)、`enabled = false` は probe を打たず、`Org{}` のゼロ値(`Enabled` false、`EscalateWatch` nil)は「すぐ送る」側に倒れる。どの場合の送信失敗も同じ文言と `inbox notify` の案内
  - approve: 記録は manifest のロックの中で `checkNotApprovedYet` → `checkApproveCandidate` → 追記。ロックのあとに受信箱を 1 回読み直し、読めなければ件ごとの `inbox show` を案内するエラー。同じ件を 2 回渡すのは `checkApproveIDs` が先に断る
  - 端末への出力: `director.jsonl` 由来の値(`id`・`driver`・`attached_at`・`user`・`pane_id`・`wake`・`cwd`・読み込みエラー)は text で `inboxField` を通る。JSON は `encoding/json`。store のエラー文の attach 時刻は `describe()` が escape。`delivery` はどの text にも出ない
  - `withMainWorktreeOrgLimits` の書き換え: 返す値と、エラー文の文言(`org: load max_orgs, max_total_seats and [org.approval].mode from the main worktree's <path>: <err>`)は変わらない。`--config` あり・台帳が main worktree のものでないときは `cfg` をそのまま返す点も同じ。その test はこの diff に出ない
  - help と実際の動き: `director` の 4 つの help、`escalate`・`inbox`・`inbox notify`・`approve` の help、`--config` と `--org-id` の flag の help を、コードの分岐と 1 つずつ照らして合っている(例外は M1 の status の help 1 文)
  - skill と文書の順序: `/director`・`/org`・README・leader の雛形・仕様のどこにも、resolve を send より先に書いた手順がない。RESULT(PR の URL を note にして resolve)は返す DECISION がないので順序の対象外で、そう書いてある
  - `/director` skill が使う動詞と flag を実物で照らした: `wait --inbox --timeout-ms 0`(無期限)、`org watch:` の `running (--once)`、`inbox ack` の後は ack の期限が効かず resolve の期限だけ残る(`InboxDeadlineDue`)、起動し直した見張りの最初のサイクルが期限切れの件を送る(`SweepInboxDeadlines`)
  - 「director を登録できない」「Until a director can be registered」「今後入る director」の残りを、Go・md・sh・toml の全体で grep した。残りは、それが消えたことを確かめる test の文字列だけ
- 確かめていないもの: テスト、`gofmt`、`go vet`、`go build`(それは /verify と /test)。`ralph org director` を実機で動かして出力を見ること。`git worktree list` を打つ分岐(連結した worktree から `--state-dir` を渡す status)を実際に動かすこと

## Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 2、LOW 10。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | doc accuracy | `ralph status` は、`--state-dir` か `RALPH_ORG_STATE_DIR` で台帳を指すとき、cache が新しいかどうかにかかわらず `git rev-parse`(連結した worktree からは `git worktree list --porcelain` も)を打つ。経路は `statusDirectorEnabled` → `withMainWorktreeDirectorEnabled` → `loadMainWorktreeOrgConfig` → `org.LedgerMainWorktreeRoot` → `gitMainWorktree`。PR はこれに合わせて `TestStatusCmd_MergeGuaranteeRunsNoGH` を緩め、test のコメントも直したが、「git は origin の get-url だけ、cache が新しいときだけ」と書いた 3 か所が残った。help は利用者向けの約束で、test は緩めたのに約束だけ元のまま | `internal/cli/status.go:71-72`(help「git runs only for the local `git remote get-url origin` in the current directory, and only when the cache is fresh」)、`status.go:168-169`(コメント)、`README.md:125`(`ralph status` の行「runs git only for the local `git remote get-url origin`, and only when the cache is fresh」)。実際に打つ git は `internal/org/statedir.go:105-118`(`LedgerMainWorktreeRoot`、source が `flag` か `env`)と `:205-226`(`gitMainWorktree`)。緩めた test は `status_merge_guarantee_test.go:383-398` | 3 か所を「git は origin の get-url(cache が新しいときだけ)と、`--state-dir` か `RALPH_ORG_STATE_DIR` のときの main worktree を調べる `rev-parse` と `worktree list`。gh もネットワークも使わない」の趣旨に書き直す。help の直しは cycle 1 のうちに入れられる |
| M2 | MEDIUM | tech-debt register | 登録簿の Trigger 列に「stage 8a's director」と書いた行が、この PR(8a-1)で入った director に触れるのに、書き換えもメモもされていない。PR が手を入れたのは 189・192・209・212・232 の 5 行で、そこには「8a-1 の director は人が対話で動かす interactive な session で、この PR は行を変えない」という判断が書いてある。同じ判断が要る行が 8 行ある。`/director` skill の「進捗の報告」は、`ralph status`(`merge guarantee:` の行を含む)と `ralph org merge-order` を人に報告するたびに打たせるので、234(merge-order を報告のたびに読む)と 239・240(保証を報告する)の Trigger は文字どおり満たされる。235・236・237・238 は merge-order の読み方(頻度、PR、JSON)で、8a-1 は text を人の問いのたびに読むだけなので満たされないはずだが、それを書いた行がない。241(`auto_merge`)と 171(無人の `--all`)は 8a-3 以降で、満たされない | `docs/tech-debt/README.md:234`(「stage 8a's director running merge-order for each report」)、`:235`(「reading merge-order often enough to meet appends」)、`:236`(「reading merge-order's PRs」)、`:237`(「reading merge-order's JSON」)、`:238`(「... stage 8a's director reading merg...」)、`:239`・`:240`(「stage 8a's director reporting the guarantee」)、`:241`(「deciding `auto_merge`」)。この PR の hunk は `-189 +189`、`-192 +192`、`-204`、`-210 +209`、`-213 +212`、`-218 +217`、`-233 +232`、`+245,2` だけ。`.claude/skills/director/SKILL.md` の「進捗の報告」 | 234・239・240 に「8a-1 の director(interactive、`/director` skill の進捗の報告)がこれを読む。件数は人の問いのたびで、頻度の問題にならない。保証の表示は status と merge-order の text のまま」のような 1 文を足し、Trigger を 8a-2 以降の無人の director に絞り直す。235〜238 と 241 は、8a-1 では満たされないことを 189・192 と同じ言い方で 1 文書く。登録簿の編集は記録だけの例外に入らないので、cycle 1 の fix か、次の full pipeline に乗せる |
| L1 | LOW | duplication | `ralph status` は `ralph.toml` を 2 回読む。`statusDeadmanMinutes` と、この PR の `statusDirectorEnabled` が、それぞれ `resolveOrgConfig("")` を呼び、読めないときはどちらも stderr に warning を出す。壊れた `ralph.toml` では warning が 2 行になる。test は director の方の文言を `Contains` で見るだけなので、2 行になっても通る | `internal/cli/status.go:199-209`(`statusDeadmanMinutes`)、`:211-231`(`statusDirectorEnabled`)、`:158`・`:163`(呼び出し)。`status_test.go` の `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad` は stderr を `Contains` で見る | `runStatus` で 1 回読み、`DeadmanMinutes` と `Director.Enabled` を同じ `cfg` から取る。warning は 1 行にする |
| L2 | LOW | dead field | `EscalateResult.Deferred` は escalate.go:188 で立てるが、production のコードに読む所がない(CLI は `Recorded` と `Err` だけを見る)。`Notified` の doc も「It is false when Deferred」と、読む人のいない field 同士を結んでいる。test だけが読む | `internal/org/escalate.go:133-140`、`:188`。`grep -rn Deferred internal/cli` は 0 件 | CLI が `Deferred` を使う(例えば stdout の行を変える)か、field をやめて banner だけにする。残すなら doc に「test と将来の呼び出し元のため」と書く |
| L3 | LOW | doc accuracy | 記録だけにした件の banner が書く ack の期限(`(%dm)`)は、escalate を打った process の `Config.Director.AckMinutes` で、機能の org の leader なら feature worktree の `ralph.toml` の値。期限を実際に使う `ralph org watch --all` は main worktree のルートで動いて、そちらの `ralph.toml` を読む。D6 は、leader が書き換えられる `ralph.toml` に `enabled` を決めさせないために main worktree の規則を入れたが、banner の数字には同じ規則がない。値を変えた feature worktree では、banner が watch の使わない期限を言う。表示だけの差 | `internal/org/escalate.go:322-326`(`o.Config.Director.AckMinutes`)、`internal/org/deadline.go:173`(watch は自分の `o.Config.Director` を読む)。watch が main worktree のルートで動くことは README の `watch --all` の段落 | banner から分の数字を外して「ack の期限(`ack_minutes`)」とだけ書く。数字を残すなら `withMainWorktreeDirectorEnabled` の隣で `AckMinutes` も同じ規則で読む |
| L4 | LOW | robustness | `detach`(`--force` を含む)も、`ralph.toml` と main worktree の `ralph.toml` が読めないと失敗する。`newOrgDirectorRuntime` が動詞を問わず設定を読むが、detach は `enabled` を使わない。`--force` は「動かなくなった session の後始末」のためにあるので、設定が壊れていると片付けの手段が塞がる | `internal/cli/org_director.go:260-267`(config の読み込みと `withMainWorktreeDirectorEnabled`)、`:173`(detach が同じ関数を通る) | detach は設定を読まない経路にする(`newOrgDirectorRuntime` に「設定が要る動詞か」を渡す、または attach と status だけが読む) |
| L5 | LOW | readability / atomicity | `--replace` は `detached` と `attached` を 2 回の `write` で書く。ロックの中なので書き手同士は混ざらないが、ロックを取らない読み手(`status`、`attachedError` の `Read`)は 2 回の間に「登録なし」を見る。途中で落ちると「detached のまま、新しい登録がない」状態が残り、そのための文言は用意してある。2 行を 1 回の `write` にすれば、どちらも起きにくくなる | `internal/org/director.go:210-229`(`appendLocked(raw, ended)` のあとに `appendLocked(raw, ev)`)、`:174`(`Read` はロックなし)、`internal/cli/org_director.go:288-294`(`attachedError` が `Read`) | `appendLocked` に複数の event を渡せる形にして 1 回で書く。直さないなら doc の「same lock hold」に「読み手には 2 回に見える」と書く |
| L6 | LOW | stale comment | `withFileLock` の doc が「the flock mechanism behind withManifestLock and the inbox lock (inbox.go's inboxLockFile)」と 2 つだけ挙げる。director のロックが 3 つ目で、`manifestLockTimeout` は名前も doc も manifest のままで 3 つのロックの待ち時間を決める | `internal/org/lockfile.go:69-70`(doc)、`internal/org/director.go:298-300`(`withLock` が `withFileLock` を呼ぶ) | doc の 1 句に `director.lock` を足す |
| L7 | LOW | readability | attach が断られたときの文言で、同じ 2 つの選択肢が 2 回出る。store のエラーが「…; detach it first, or attach with replace」で終わり、CLI の `attachedError` がそのあとに「: end it with `ralph org director detach --id d1` from the session that attached it (…), or replace it with `ralph org director attach --driver claude --replace`」を続ける。1 文の中で「detach するか replace する」を 2 度言う | `internal/org/director.go:202`(store の tail)、`internal/cli/org_director.go:288-297`(CLI の続き)、`internal/cli/org_director_test.go:156`(test が両方を固定) | store の tail を「; it must end first」のように選択肢を言わない形にし、コマンドは CLI の側だけに置く |
| L8 | LOW | readability | status の 3 つの出力関数の引数が 10〜12 個になった。`printStatusEmpty`(10)、`printStatusJSONAllOrgs`(12)、`printStatusTableAllOrgs`(11)に、この PR が `director orgDirectorReport` を 1 つずつ足した。`watch`・`director`・`guarantee` は型が違うので取り違えは起きないが、7c-2 の `guarantee` に続いて 2 つ目の追加で、次の追加も同じ形になる | `internal/cli/status.go:384`・`:569`・`:605` | 次にもう 1 つ足すとき、`watch`・`director`・`guarantee` を 1 つの struct(例えば `statusLedgerLines`)にまとめる。今は直さなくてよい |
| L9 | LOW | robustness | `InboxItem.Delivery` は、手で編集された行の `delivery` をそのまま通す。`InboxStore.Escalate` は `now` と `deadline` 以外を断るので、書く側は 2 値に閉じているのに、読む側は何でも通し、`--json` の利用者に doc にない値(`"deadline "`、`"x"`)が出る。text には出ないので端末の escape の問題はない。`inboxItemDeferred` は `== deadline` だけを見るので、不明な値は「deferred でない」に倒れる | `internal/org/inbox.go:462-468`(fold)、`:214-223`(`Escalate` の検査) | fold で `now` と `deadline` 以外を `now` に寄せる(delivery のない古い行と同じ扱い)。直さないなら doc の「one of the InboxDelivery* constants」を「written as one of ...」にする |
| L10 | LOW | design note | 座席が `ralph org director attach --replace` や `detach --force` を打っても止まらない。approve には `approveTripwire`(HERDR_PANE_ID が active な座席の pane なら断る)があるが、登録の動詞にはない。登録は escalate の動きを変えない(D2)ので、何かの権限を開くものではない。ただし `/director` skill の昇格 5 は、`id` が違っていたら director の仕事(ack、resolve、send、start)をやめる、と書くので、登録を書き換えられる process は director を止められる。座席の prompt には director の話が出ないので、起きにくい | `internal/org/director.go`(`Attach` と `Detach` に pane の検査なし)、`.claude/skills/director/SKILL.md` の「昇格」5、`internal/org/approve.go:218-235`(approve の tripwire) | 直さなくてよい。8b で director に仕事を渡す(push)ときに、tripwire を入れるか、`--replace` と `--force` を人の端末に限るかを決める。登録簿に 1 行足すなら、その Trigger にする |

## Positive notes

- `DirectorStore` の「`id` を壊れた行も含めた最大値の次にする」が、raw テキストの regex(閉じ引用符なし。途中で切れた行の `"id":"d12` も数える)と、decode した `id` の両方で最大値を取る。JSON のエスケープされた `\"id\":\"d9` には当たらず、`int64` を超える値は飛ばし、最大値のときは明示のエラーにする。`TestDirectorAttach_NeverReusesTheIDOfADamagedLine` と `FailsWhenNoIDIsLeft` が分けて固定している
- detach は `id` の形(`d<N>`)を、ファイルを読む前に検査し、`id` と `--force` の両方、どちらもなし、を別々の文言で断る。ファイルがなければロックもディレクトリも作らない。`ErrDirectorIDMismatch` のエラーは、今の登録の `id` と、入れ替えられたので外さなくてよいことを言い、A → B(`--replace`)→ A の detach の順を test が通している
- escalate の 3 つの場合が `escalateDelivery` の 1 か所に集まり、`Org{}` のゼロ値と `EscalateWatch` が nil の場合が「すぐ送る」に倒れる。`enabled = false` では probe を打たず、検査で断られた message でも probe を打たない(`RefusalProbesNoWatch`)。どの場合の送信失敗も同じ文言と `inbox notify` の案内に揃う
- `enabled` を読む関数が `loadMainWorktreeOrgConfig` に分かれ、`withMainWorktreeOrgLimits` は返す値もエラー文の文言も変えない。2 つの関数が同じ規則(`--config` なし、台帳が main worktree のもの)を 1 か所で持つ。escalate だけは、main worktree の `ralph.toml` が読めないときに人へ届く経路(`EscalateNotRecorded`)へ倒し、test がある
- approve の再読は、受信箱が読めないときも、件ごとの `inbox show` を案内するエラーを返し、`ResolvedElsewhere` を立てない。test は `Org.Now` に差し込んだ処理で「1 回目と 2 回目の読みの間」に受信箱を壊す。7a で resolve の失敗を固定していた test が、「受信箱が書けなくても approve は通り、件は open のまま」(`PassesWithAnInboxItCannotWrite`、root は skip)に反転していて、反転の理由が test のコメントに書いてある
- `/director` skill の語彙(動詞、flag、`org watch:` の行の文言、`--timeout-ms 0`)が実物と合い、「答える順」を 1 か所に置いて、計画の承認の節と `/org` skill がそこを指す。RESULT の resolve(返す DECISION がない件)を順序の例外として明記している
- 設定の `enabled` の既定が、`Default()`、`templates/base/ralph.toml`、`scripts/ralph-config.sh` の 2 面、`defaults_sync_test.go` の 4 か所に同じ値で入り、`[org.director]` の一部の key だけを書いた `ralph.toml` が `enabled = true` のまま読まれることを test が固定している(`Load` は `Default()` に重ねて decode する)

## Test intent changes recorded by the plan

- (1) `TestStatusCmd_MergeGuaranteeRunsNoGH`(`status_merge_guarantee_test.go:383-398`)。元の意図は「status は gh を打たず、git は origin の get-url だけを 1 回の実行につき 1 回」。変更後も、gh を打たないこと(`ghLog` が作られない)と、`git remote get-url origin` が実行 1 回につき 1 回であること(`origin != runs` で比べる。2 回目の get-url が入る退行はまだ赤くなる)は保たれている。落ちたのは「ほかの git を打たない」の半分で、`rev-parse ` の接頭辞なら何回でも通る。意図は保たれているが、緩め方が広い。回数か引数を縛らないので、`rev-parse` のループが入っても気づかない。連結した worktree からの `git worktree list --porcelain`(`gitMainWorktree` の分岐)は `default` の腕に落ちて赤くなるが、test は通常の repo で動くので届かない。この変更に付いてくるはずの help・README の直しが入っていない(M1)。判断: 意図は保つが、文書の 3 か所を直すこと(M1)と、可能なら `rev-parse` を回数で縛ること(1 回の実行で `rev-parse` が 2〜3 回なら 3 回までなど)
- (3) 7a の approve の test。`TestOrgApprove_RecordsResolvesAndLeavesThePlan` → `RecordsAndLeavesTheItemAndThePlan`(件が open のまま、note なし、inbox ファイルが byte 単位で同じ)、`..._ResolveFailureKeepsTheRecord` → `TakesNoInboxLock`(inbox のロックを塞いでも approve が通る。塞いだロックで resolve が実際に失敗することを前提として確かめる)、CLI の `AResolveThatFailsKeepsTheRecordAndPrintsIt` → `PassesWithAnInboxItCannotWrite`、`RecordsResolvesAndPrintsTheDecision` → `RecordsAndPrintsTheSendThenTheResolve`、`TheDecisionCommandRunsInAPOSIXShell` → 送る打ち方と resolve の打ち方を両方 `sh -c` で通す。`ResolvedBySomeoneElse` は、`Contains` と否定の確認だったものが、エラー文全体の等号に変わって強くなった。元の意図(all or nothing、記録は残る、他者が resolve した件は `show` を案内して resolve の打ち方を出さない、`--state-dir` の写し)はどれも同じ強さで残っている。`RefusesAnItemItCannotResolve` → `RefusesAnItemWhoseResolveNoteIsTooLong` は、記録の前に断る点が同じで、理由が「approve が resolve に失敗する」から「approve が出す resolve の打ち方が失敗する」に変わった。判断: 契約の変更(D7)に沿った書き換えで、意図は弱まっていない。対応する新しい test(`RefusesAnItemAlreadyApproved`、`AnApproveThatStopsBeforeTheSendReachesAHumanAtADeadline`、`AnInboxItCannotReadAgainKeepsTheRecords`)が、元の test が守っていた「止まった件が誰かに届く」を、期限の側から守り直している

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L10: `director attach --replace` と `detach --force` に座席の pane の tripwire がない | 座席の process は director の登録を書き換えられる。登録は escalate の動きを変えないが、`/director` skill の昇格 5 を読む director は、`id` が変わったら仕事をやめる | 登録は権限を開かない(D2、D4)。座席の prompt に director は出ない | 8b で director に push の仕事を渡すとき、または座席が登録を書き換えた報告 | `docs/reports/self-review-2026-10-14-org-director-attach.md`(L10)、`internal/org/approve.go`(`approveTripwire`) |

_(登録簿には追記していない。M2 は新しい debt ではなく既存の行の書き換え。LOW のうち直さずに最後の cycle に届いたものだけを、1 つの行にまとめて追記する。L10 は、直さないと決めたなら cycle 1 の fix の回に行として足してよい。)_

## Coverage gaps

- 実機の出力: `ralph org director status` の text と `--json`、`ralph status` の `director:` の行、escalate の banner を、binary で動かして見ていない(test が文言を固定している)。計画の Evidence to capture のとおり /test の担当
- `git worktree list --porcelain` を打つ status(連結した worktree から `--state-dir` を渡す)は、どの test も通っていない(M1 と同じ原因)
- escalate の probe と record の間に見張りが落ちる競合は、計画が既知の穴として 210 行目に 1 文足している。ここでは追加の指摘をしない
- 同じ件に対する approve の同時実行(2 つの process が同時に打つ)は、manifest のロックの中の `checkNotApprovedYet` で片方が断られる設計だが、2 process の test は読んでいない(`RefusesAnItemAlreadyApproved` は直列)

## Recommendation

- Merge: yes(CRITICAL・HIGH はなし。MEDIUM 2 件は文書と登録簿の直しで、コードの欠陥ではない。直してから進めるのがよい)
- Follow-ups:
  1. M1: `status.go` の help(71-72 行)とコメント(168-169 行)、README 125 行の 3 か所を、実際に打つ git に合わせて直す。cycle 1 の fix の回で入れる
  2. M2: 登録簿の 234・239・240 に「8a-1 の director が読む」旨の 1 文、235〜238・241 に「8a-1 では満たされない」旨の 1 文を足す。189・192・209・212・232 と同じ言い方にする
  3. LOW のうち、L1(`ralph.toml` の 2 回読み)、L3(banner の数字)、L4(detach が設定を読む)は、小さな差分で直せるので cycle 1 の fix に入れるとよい。L2・L5〜L9 は、直さずに最後の cycle に届いたら 1 つの行にまとめる。L10 は上の表のとおり
  4. test の緩和(1)は、`rev-parse` の回数を縛る(任意)
  5. Go のコメントの `docs/plans/active/2026-10-14-org-director-attach.md`(`director.go:20`、`org_director.go:19` など)は、`/pr` の archive で古くなる。`mergeguarantee.go`・`mergeorder.go` が同じ形で、登録簿もこの形を許してきたので、指摘としては数えない
- Known gaps: テスト、`gofmt`、`go vet`、go1.25.8 でのビルドは確かめていない。`ralph org director` の実機の出力は見ていない。`git worktree list` の分岐は動かしていない。approve の同時実行の test は 2 process のものを読んでいない

## Fixes after this review (cycle 1, before /verify)

この節はオーケストレーターが足した。直しはコミット 57c551ff(`fix: address the org-director-attach self-review findings`、15 ファイル)。上の重大度と `Merge:` の行は変えていない。

| ID | 対応 |
| --- | --- |
| M1 | `ralph status` の help に、gh を打たずネットワークも使わないこと、origin の get-url は cache が新しいときだけ打つこと、main worktree を探す `rev-parse`(連結した worktree では `worktree list --porcelain` も)は cache によらず打つことを書いた。`runStatus` のコメントと `README.md` の `ralph status` の行も同じ内容に直した。help の文は test で固定した |
| test (1) | `TestStatusCmd_MergeGuaranteeRunsNoGH` に、1 回の実行ごとの `rev-parse` の上限 2(`statusMGMaxRevParsePerRun`、今の 1 回分)を足した。`withMainWorktreeDirectorEnabled` を 2 回呼ぶ変異で落ちることを確かめた |
| M2 | 登録簿の 234・239・240 に、8a-1 の director は人が動かす対話の速さで status と merge-order を読み、merge は人が決める、と 1 文ずつ足し、Trigger を 8a-2 と 8a-3 に移した。235〜238・241・171 には、8a-1 では満たされないと書いた |
| L1 | `statusOrgConfig` で `./ralph.toml` を 1 回だけ読む。読めないときの warning は 1 行になった |
| L2 | `Deferred` の doc に、2 つの記録の場合を見分けたい呼び出し元のための field で、CLI は banner で区別し、test が読むと書いた |
| L3 | 記録だけの件の banner から分の値を外し、`ralph org watch --all` の ack の期限(`[org.director].ack_minutes`)とだけ書く |
| L4 | `detach`(`--id` と `--force`)は設定を読まない。壊れた設定でも detach が通る test を足した |
| L5 | replace の `detached` と `attached` を 1 回の write で書く。失敗したときは何も記録されず、前の登録が残る |
| L6 | `withFileLock`、`manifestLockTimeout`、`pollInterval` の doc に `director.lock` を足した |
| L7 | store のエラーは登録の説明で終わり、detach と `--replace` の打ち方は CLI だけが出す |
| L8 | 直していない(次に引数を足すときにまとめる) |
| L9 | 受信箱の畳み込みは `deadline` 以外の `delivery` を `now` に寄せる |
| L10 | 直していない。登録簿の末尾に行を足した(Trigger は 8b の push と、座席が登録を書き換えた報告) |

確かめたこと(implementer の報告): `gofmt -l internal` は出力なし、`go vet ./...` は通過、`go test ./internal/org/... ./internal/cli/...` は go1.26 と `GOTOOLCHAIN=go1.25.8 -count=1` の両方で ok、`go build ./...`、`check-skill-sync.sh`、`check-sync.sh` も通過。オーケストレーターは `go test ./internal/org/ ./internal/cli/ -run 'Director|StatusCmd|Escalate|InboxFold|Delivery' -count=1` を打ち直して ok を確かめた

## Cycle 2

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-14-org-director-attach.md(承認済み、digest 27414f27525b。変わっていない)
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2)。`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定 2 のうち最後の回
- Scope: diff の品質だけ。cross-review cycle 1 の ACTION_REQUIRED 1 件の直し 634b019a(`git diff 90fe4e06..HEAD`、`internal/org/director.go` +74/-15、`director_test.go` +216)を、全体の `git diff ef575748...HEAD` の中で読んだ。cycle 1 の指摘(M1、L1 など)と取り違えないよう、この節の ID には `C2-` を付ける。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない

### Evidence reviewed

- 読んだ差分: 634b019a の全 hunk(`director.go` の 6 か所、`director_test.go` の追加 216 行すべて)
- 読んだ周辺: `director.go` の store 部分(150〜420 行)と畳み込み(437〜517 行)、`internal/cli/org_director.go` の attach・detach・status の RunE と `rt.command`・`attachedError`・`readOrgDirectorReport`・`warnDirectorDamage`、`InboxStore.appendLocked`(`inbox.go:389-410`)、`taskNotSentError`(`verbs.go:219-235`)、cross-review の triage report 全体、test report の D5 の 2 か所(78 行、102 行)、plan の Progress checklist、登録簿の 243〜247 行
- 確かめたこと
  - restore の位置と大きさ: `appendLocked` の呼び出しは `Attach`(`director.go:244`)と `Detach`(`:304`)だけで、どちらも `withLock` の中。`raw` は同じロックの中で `readRaw` したものなので、`len(raw)` は追記前の大きさそのもの。ファイルがなかったときは `O_CREATE` が空のファイルを作り、`Truncate(0)` で空に戻る。test は `readDirectorFile` が読めないと Fatal になるので、「消さずに空」を固定している。open の失敗と marshal の失敗は何も書かないので戻さない
  - close の失敗: write の seam が `f` を閉じてから `appendLocked` の `Close` を失敗させる作りで、HEAD の `Close` の分岐をそのまま通る
  - エラーの包み方: `restoreLocked` の `%w, and %w (%w)` が 3 つを包み、`directorAppendFailed` がさらに包む。test が sentinel、ENOSPC、truncate のエラーに `errors.Is` を打ち、restored の側では sentinel でないことも確かめる。文言は全文を等号で固定している
  - seam: nil なら `(*os.File).Write` と `os.Truncate`(型が同じ)。非 test のコードで代入する所はなく、`DirectorStore` を作るのは `NewDirectorStore` だけ。production の動きは変わらない
  - 修正前のコードで新しい test が落ちるか(読んで判断した。実行はしていない): 修正前は部分書き込みがそのまま残るので、restored の 3 本は `got != before`(missing file の 1 本は `got != ""`)で、close の 1 本は 2 行が残って赤くなる。`restoreLocked(len(raw), …)` の `len(raw)` を `0` にする変異は、replace と detach の test が赤くする
  - D5 の test: ファイルは `attached d1` と `detached d2` の 2 行だけで、畳み込みの `ev.ID != f.current.ID`(`director.go:485`)に当たって `IgnoredEvents == 1`。この分岐を外す変異では current が nil になり赤くなる。コメントの「既存の test では 2 行が打ち消し合う」は test report の 78 行と合う
  - `git diff --check` は空。追加行に TODO・debug 出力・`/Users/`・秘密らしい文字列はない
  - cycle 1 の直し(57c551ff)のうち、この変更に触れるのは L5 と L7。L5: replace は 1 回の write のままで、`Attach` の doc の「two lines of one write」は今も正しい。既存の `AReplaceThatCannotWriteRecordsNothing` は open 失敗の経路で、コメントの「neither the detached nor the new attached is in the file」も今の動きに合う。L7: store の `ErrDirectorAttached` の文言は変わらず、`director_test.go:207` が固定している。ただし新しい store のエラーがコマンドを名指す(C2-L1)。L6 と L9 は触れていない
- 確かめていないもの: テスト、`gofmt`、`go vet`、go1.25.8 でのビルド。実際のディスクフルでの出方。CLI を通したときのエラーの出方(この変更に CLI の test がない)

### Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-L1 | LOW | hint / convention | `ErrDirectorWriteNotRestored` を包むエラー文が `ralph org director status` を名指すが、`--state-dir` がない。CLI は `rt.command(...)` で、`--state-dir` を渡した呼び出しにだけ `--state-dir` を足す(`attachedError` も同じ作り)。cycle 1 の L7 は「store の文言は登録の説明で終わり、コマンドは CLI だけが出す」と決めたのに、新しい store の文言がコマンドを持った。attach と detach の RunE は `ErrDirectorAttached` 以外のエラーを `return err` で素通しするので、`--state-dir` を付けて打った人にも既定の台帳を読ませる文言が出る。重さは LOW と判断した。起きるのは write と truncate の両方が失敗したときだけで(縮める truncate は空きを要らないので、ENOSPC なら普通は戻せる)、`status` は読むだけの動詞で、エラー文の `append to <path>` がファイルを名指す。`/director` skill は `--state-dir` を一度も使わない(grep で 0 件)ので、既定の台帳を読むのが普通の使い方になる | `internal/org/director.go:407`(`directorAppendFailed` の文言)、`director_test.go:369-370`(その全文を固定)、`internal/cli/org_director.go:121`・`:183`(attach と detach の `return err`)、`:288`(`command`)、`:298`(`attachedError`)、`director_test.go:207`(L7 の約束を固定する test) | store の文言は「…so it may hold part of the write」で止める。CLI の 2 か所で `errors.Is(err, org.ErrDirectorWriteNotRestored)` のとき `rt.command("ralph org director status")` を足す(`attachedError` と同じ形)。`--state-dir` つきの CLI の test を 1 本。直さないなら C2-L5 の行に入れる |
| C2-L2 | LOW | readability | not-restored の文言は `org: director: attach: org: director: append to <path>: no space left on device, and director.jsonl could not be restored …` と、`org: director:` が続けて 2 回出る。restored の文言(`…is still the current registration: org: director: append to …`)にも 2 回目はあるが、間に説明の句があるので目立たない。repo は `org:` を 1 回にする書き方を取っている(`taskNotSentError` の doc、org-watch-gate の L3)。`appendLocked` の open・marshal・append・close のエラーは、`directorAppendFailed` を通る 2 つの呼び出しからしか返らない | `director_test.go:369-370`(want の全文)、`director.go:369-380`、`:407`、`internal/org/verbs.go:219-235` | `appendLocked` の内側のエラーから `org: director: ` を外し、`directorAppendFailed` が 1 回だけ足す。3 つの文言と test の want が変わる。直さないなら C2-L5 の行に入れる |
| C2-L3 | LOW | stale pointer | 「cross-review finding 1 of org-director-attach」が 4 か所(Go のコメント 3、test のコメント 1)にある。指す先の triage report は `/cross-review` が cycle ごとに書き換え(ヘッダは `Cycle: 1/2`、表の番号は 1 から振り直される)、cycle 2 の triage が終わると「finding 1」は別の指摘を指しうる。同じ package の書き方は、追記だけの self-review の ID(`self-review L5 of org-director-attach`)か、段階を書いた「Codex plan advisory finding 1」。`director_test.go:850` の「mutant D5 of the org-director-attach test report」も、mutation の番号が test report の run ごとのもので、cycle 2 の `/test` が表を作り直したときに同じ ID が残る保証はない。この cycle の cross-review がまだ走っていない今は、指す先がまだ 1 つしかない | `director.go:69`・`:209`・`:353`、`director_test.go:305`・`:850`、`docs/reports/cross-review-triage-org-director-attach.md`(`Cycle: 1/2`) | 欠陥を言葉で書く(「a write that stops partway, after the detached line」)か、「cycle 1 cross-review」と段階を足す。コメントの変更で pipeline をもう 1 回回すことになるなら、C2-L5 の行に回す |
| C2-L4 | LOW | doc accuracy | `ErrDirectorWriteNotRestored` の doc の「Every other append failure is restored and means nothing was recorded」は、open の失敗と marshal の失敗が何も書かないので、戻すものがない。戻すのは write と close の失敗だけで、`appendLocked` の doc はそう書いている。`Attach` の doc の「A plain attach fails the same two ways」は、「two ways」の指す先が前の文にない(戻せた、戻せなかった、の 2 つ)。`Detach` の doc は同じ内容を「cut back as Attach's is」と書き、言い方が揃っていない | `director.go:70-71`、`:209-210`、`:265-268` | 「Every other append failure left nothing in the file」、「A plain attach ends the same two ways」程度に直し、`Detach` も同じ言い方にする |
| C2-L5 | LOW | tech-debt register | cycle 1 の L8(status の出力関数 3 つの引数が 10〜12 個)は、直さず、登録簿の行も作らないまま、最後の cycle(cap 2 のうち 2)に届いた。cycle 1 の報告は「L2・L5〜L9 は、直さずに最後の cycle に届いたら 1 つの行にまとめる」と書いていて、L2・L5〜L7・L9 は 57c551ff で直った。L8 だけが残り、`grep -n 'orgDirectorReport\|printStatusTableAllOrgs' docs/tech-debt/README.md` に当たる行がない(`:18` と `:213` にある `printStatus…` は別の話の行)。L10 の行は `:247` にある。この cycle の C2-L1〜C2-L4 も、直さなければ同じ扱いになる | cycle 1 の表の L8 の行と「Fixes after this review」の L8 の行、`internal/cli/status.go:401`・`:586`・`:622`(今も引数 10〜12 個) | 1 つの行にまとめて、L8 と、残った C2 の指摘を入れる(Trigger は「次に status の出力関数へ引数を足すとき」と「director の store のエラー文に触るとき」)。登録簿の編集は記録だけの例外に入らないので、通常の経路(次の `/sync-docs` か、cap を上げた re-run)で入れる |

### Positive notes

- restore がロックの中にあり、大きさの根拠が `readRaw` と同じロックの保持で取った `raw` なので、「追記前の大きさ」を別に数え直さない。作ったファイルは消さずに空へ戻すので、`Detach` の「ファイルがなければロックも作らない」の分岐と、空のファイルの読みが食い違わない(空は登録なし、`ErrDirectorNotAttached`。test が force の detach で確かめている)
- 直しが triage の推奨の字面ではなく意図に届いている。plain な attach と detach も同じ経路に入れ、restored と not-restored で文言を分け、D5 の test も同じ commit に入れた。not-restored の test は truncate の呼び出しの引数(path と大きさ)まで固定している
- close の失敗を、実際の `Close` を失敗させて通す(seam が先に閉じる)ので、`Close` の分岐の restore が test で赤くなる。mock の error を返すだけの作りより強い
- 受信箱の `appendLocked`(`inbox.go:389-410`)には同じ restore がないが、1 回の write は 1 行で、途中で切れた行は壊れた 1 行になるだけで状態を変えない。director で restore が要ったのは、replace の 2 行 write で `detached` が完全な行として残るからで、受信箱に広げる必要はない(指摘にしない)
- D5 の test は、既存の test と打ち消し合わないよう 2 行のファイルだけにして、コメントで理由を書いている。test report の 78 行の説明と合う

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| cycle 1 の L8(status の出力関数 3 つが引数 10〜12 個)と、直さずに残った C2-L1〜C2-L4(store のエラー文が `--state-dir` のないコマンドを名指す、`org: director:` の重複、「cross-review finding 1」の指す先、2 つの doc の言い回し) | どれも挙動を変えない。C2-L1 は、write と truncate が両方失敗し、`--state-dir` を付けて打ったときだけ、既定の台帳を読ませる | cap 2 の最後の cycle で、文言とコメントの変更に pipeline をもう 1 回回す価値がない | 次に status の出力関数へ引数を足すとき(L8)。director の store のエラー文か、attach・detach の RunE に触るとき(C2-L1〜C2-L4) | `docs/reports/self-review-2026-10-14-org-director-attach.md`(cycle 1 の L8、cycle 2 の C2-L1〜C2-L4) |

_(登録簿には追記していない。上の行は、C2-L1〜C2-L4 を直さずに進めるときに 1 行で足す形。)_

### Coverage gaps

- not-restored の test は replace の 1 本だけ。plain な attach と detach の not-restored は同じ `directorAppendFailed` の分岐を通るので、分岐は 1 つで、文言は同じ
- 末尾に改行がない `raw` への追記が失敗した場合(先頭に足した `\n` ごと戻す)の test はない。式は `len(raw)` 1 つで、追記前の大きさそのものなので、穴は小さい
- `os.Truncate` は、ファイルが `len(raw)` より短くなっていると NUL で伸ばす。ロックを持つ ralph の書き手だけなら起きない。人が読んだ直後に手で縮めたときだけの話で、指摘にはしない
- CLI を通したエラーの出方に test がない(C2-L1 の直しを入れるなら足す)

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。LOW 5 件は文言、コメント、登録簿で、restore の動きに欠陥はない)
- Follow-ups:
  1. C2-L5: L8 と、残った C2 の指摘を 1 つの行にまとめて登録簿に足す。cap 2 の最後の cycle なので、直さない LOW はこの行が唯一の受け皿になる
  2. C2-L1: 直すなら CLI の 2 か所(`org_director.go:121`・`:183`)で `rt.command` を使う形が、コードで一番小さい。直さないなら C2-L5 の行
  3. C2-L3: コメントを直す機会があるなら、cycle 2 の `/cross-review` が triage を書き換える前に直す。そのあとでは「finding 1」が 2 つの cycle の指摘を指す
  4. plan の Progress checklist に、cross-review cycle 1 と 634b019a の行がない。`/pr` が plan を archive するので、`/sync-docs` で足すとよい(diff の品質の外なので、指摘には数えない)
- Known gaps: テスト、`gofmt`、`go vet`、go1.25.8 でのビルドは確かめていない。実際のディスクフルでの出方と、CLI を通したエラーの出方は見ていない。restore の「新しい test が修正前で赤くなる」は読んで判断したもので、実行はしていない

### Fixes after the cycle-2 review (before /verify)

この節はオーケストレーターが足した。直しはコミット f4c1a394(`fix: address the cycle-2 self-review findings for org-director-attach`、5 ファイル)。上の重大度と `Merge:` の行は変えていない。

| ID | 対応 |
| --- | --- |
| C2-L1 | store の not-restored のエラー文はコマンドを持たず「so it may hold part of the write」で終わる。CLI の `notRestoredError` が `errors.Is(err, org.ErrDirectorWriteNotRestored)` のときだけ、`rt.command` で `--state-dir` 付きの `ralph org director status` を足す。helper を直接テストした。attach と detach の RunE がこの helper を通ることにはテストがない(store の seam が unexported のため。known gap) |
| C2-L2 | `appendLocked` のエラーから `org: director:` を外し、`directorAppendFailed` だけが付ける。3 つの文言で 1 回になった |
| C2-L3 | 「cross-review finding 1」と「mutant D5」のコメントを、起きることを言葉で書く形にした |
| C2-L4 | `ErrDirectorWriteNotRestored`、`Attach`、`Detach` の doc を、open と marshal の失敗は何も書かず、write と close の失敗は切り戻す、という書き方に直した |
| C2-L5 | 登録簿の末尾に、cycle 1 の L8(status の出力関数の引数が 10〜11 個)の行を足した。Trigger は `ralph status` のヘッダに次の行を足すとき。C2-L1〜L4 はすべて直したので、行には L8 だけを入れた |

確かめたこと(implementer の報告): `gofmt -l internal` は出力なし、`go vet ./...` は通過、`go test ./internal/org/... ./internal/cli/...` は go1.26 と `GOTOOLCHAIN=go1.25.8 -count=1` の両方で ok、`go build ./...` も通過。オーケストレーターは `go test -count=1 ./internal/org/ ./internal/cli/ -run 'Director'` を打ち直して ok を確かめた

## Cycle 3

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-14-org-director-attach.md(承認済み、digest 27414f27525b。変わっていない)
- Reviewer: reviewer subagent (Claude)。パイプライン 3 回目(cycle 3)。cross-review cycle 2 の ACTION_REQUIRED を直すために `RALPH_STANDARD_MAX_PIPELINE_CYCLES` を 2 から 3 に上げた回で、`cycle-count.json` は 2 のまま(意図どおり)
- Scope: diff の品質だけ。cross-review cycle 2 の直し c8a9227b(`/director` skill の受信箱の節、4 面で +36/-16)を、全体の `git diff ef575748...HEAD` の中で読んだ。cycle 1・2 の ID と取り違えないよう、この節の ID には `C3-` を付ける。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない

### Evidence reviewed

- 読んだ差分: c8a9227b の全 hunk(4 面とも同じ 13 行の変更)
- 読んだ周辺: `internal/org/escalate.go` の `WaitInbox`(253〜302 行)、`internal/cli/org.go` の `newOrgWaitCmd` と `runOrgWaitInbox`(1108〜1190 行)、`internal/cli/root.go` の `SilenceErrors`、`cmd/ralph/main.go` の終了コード、`/director` skill の全文、`/org` skill の `wait` の行(173 行)、cross-review の triage の cycle 2 の指摘、plan の Progress checklist、登録簿の director の行
- 確かめたこと
  - 文言とコード: 時間切れの文は `org: wait --inbox: no open inbox item arrived within %d ms`(`escalate.go:295`)で、skill が引く `no open inbox item arrived within` はその部分文字列。`root.go` が `SilenceErrors` なので、エラーは `main.go` の `fmt.Fprintln(os.Stderr, err)` が 1 回だけ stderr に出す
  - 終了コード 3 には届かない: `main.go` が 3 で終わるのは `errors.Is(err, cli.ErrUpgradeDriftRemaining)` のときだけで、その sentinel を作るのは `runUpgradeV2`(`upgrade_v2.go:226`・`:302`)だけで、`migrate.go:989` はそれを `ralph upgrade` の移行の経路でそのまま返す。`wait --inbox` は 0 か 1 で終わる(外から kill されたときを除く)ので、skill が「終了コード 1」と書いて足りる
  - 時間切れでない失敗: `--inbox` と `--org-id`・`--seat`・`--until` の併用(`org.go:1176`)、`newOrgRuntime` の失敗(旧台帳の guard、`ralph.toml` の load)、受信箱の read の失敗(`escalate.go:283`、`TestWaitInbox_ReadFailureReturnsAtOnce`)は、どれも時間切れの文を含まず、すぐ返る。skill の「stderr が時間切れの文でなければ」はこれらをすべて含む
  - Claude Code の `--timeout-ms 0`: `WaitInbox` は `timeout > 0` のときだけ `deadline` を作る(`escalate.go:268-273`)ので、この run は時間切れの文を出さない。終了コード 0 で返るのは件の行が 1 行以上あるときだけ(`runOrgWaitInbox` は `err == nil` なら `items` が空でない)
  - 4 面: `md5` が 4 面で同じ(`ac5b10a3…`)。`.claude` と `.agents` の `diff` は空
  - 新しい文に ID の引用、行番号、`AR#` はない。commit message は「cross-review cycle 2」と回で書いている
  - `git diff --check ef575748...HEAD` は空。追加行に TODO、debug 出力、`/Users/`、秘密らしい文字列はない
  - 登録簿: この PR が足した director の行 5 件に、cap の数字や「最後の run」を理由にした書き方は残っていない
- 確かめていないもの: テスト、`gofmt`、`go vet`。skill の文を実際の Codex と Claude Code の session で動かしたときの挙動

### Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C3-L1 | LOW | cross-file coupling | skill が「打ち直すか止まるか」の分かれ目を、エラー文の部分文字列 `no open inbox item arrived within` に置いた。この文を作る `WaitInbox` のエラーにも `WaitInbox` の doc にも、skill が照合していることを示す記述がない。文を固定する 2 本の test は、ふつうのエラー文の確認として書かれている。文言を直した人は test の期待値を書き換えれば通り、skill に気づく手がかりがない。そのとき Codex の director は、毎回の時間切れを失敗として扱い、5 分待った最初の wait で人に伝えて止まる(打ち直しの loop にはならず、人に伝えるので黙った停止でもないが、動きは壊れる) | `internal/org/escalate.go:295`(コメントなし)と `:253-260`(doc)、`escalate_test.go:1216`、`internal/cli/org_inbox_test.go:1222`、`.claude/skills/director/SKILL.md:105-107`。skill の文を読む test は見つからなかった(`grep -rn 'arrived within'` は上の 4 か所と report だけ) | `escalate.go:295` の上に 1 行(`/director` skill が打ち直す条件としてこの文を照合する)。2 本の test の期待値の横にも同じ 1 行。機械で守るなら、skill の引用が実際の時間切れのエラーに含まれることを見る test を 1 本 |
| C3-L2 | LOW | skill instruction | cross-review の指摘は「2 つの driver の両方に書く」だった。直しは Codex の項に条件を書き、Claude Code の項は「件の行を出して返ったら件を処理し」に変えただけで、件の行を出さずに返ったとき(エラー)の扱いは共通の段落に任せた。その段落は「終了コード 1」と「時間切れの文」で書かれているが、`--timeout-ms 0` の run は時間切れの文を出さない。Claude Code の director には「時間切れでなければ」が常に真で、段落が分ける相手がない。また Monitor で見張る形では、director に届くのが stdout の行だけだと(未確認)、エラーの文が見えず、「エラーの文を人に伝える」に届かない | `.claude/skills/director/SKILL.md:101-103`(Claude Code の項)と `:109-111`(共通の段落)、`escalate.go:268-273`、triage の cycle 2 の指摘「2 つの driver の両方に書く」 | Claude Code の項に「件の行が出ずに返ったら、`--timeout-ms 0` に時間切れはないのでエラー。stderr の文を人に伝えて止まる」を足す。Monitor で見張るなら stderr を読む手段(`2>&1`、または出力ファイル)も 1 句足す |
| C3-L3 | LOW | skill instruction | 新しい段落の「エラーの文を人に伝えて止まる」は、止まったあとの登録と受信箱をどうするかを言わない。同じ skill の別の止まり方(昇格の 5)は「受信箱の ack と resolve、leader への send、start をしない」「detach も打たない」と書き、終わりの節は件の残りを人に伝えてから detach を打つ。新しい止まり方がどちらに当たるのか読み取れない。detach しないまま止まると、登録が `attached` のまま残り、次の `/director` の attach が断られて人の `detach --force` が要る。件は登録の有無によらず期限で人に届くので、実害は小さい | `.claude/skills/director/SKILL.md:109-111`(新しい段落)、`:68-74`(昇格 5)、`:210-218`(終わり) | 「止まる」を「director として動くのをやめ、人に伝える。detach するかは人に聞く」か「終わりの手順に進む」のどちらかに決めて、1 句足す |

### Positive notes

- 直し方が triage の推奨と合う。CLI の終了コードを分けず(「座席の wait も含めた契約の変更になる」)、skill の側で打ち直す条件を決めた。照合する文は CLI が 1 回だけ出す文で、`SilenceErrors` と `main.go` の出し方に合う
- 打ち直す条件が閉じた形(時間切れの文があるときだけ打ち直す)なので、想定していない失敗は止まる側に倒れる。新しい loop は増えず、止まるときは人に伝える
- 終了コード 3 の経路を skill に持ち込んでいない。`wait --inbox` から届かないので、「終了コード 1」の一言で足りる
- 4 面が同一で、新しい文に ID、行番号、`AR#` の引用がない。cycle 2 の triage が回ごとに書き換わっても、skill の文は指す先を失わない
- Codex の `--timeout-ms 300000` と Claude Code の `--timeout-ms 0` の run の形は、前の文のまま残した

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C3-L1〜C3-L3(skill が時間切れの文を部分一致で照合していることがコードに書かれていない、Claude Code の項に件の行がないときの扱いがない、「止まる」の先が決まっていない) | どれも CLI の動きを変えない。C3-L1 は文言が変わると Codex の director が最初の時間切れで止まる。C3-L2 は Monitor でエラーの文が見えないと、Claude Code の director が失敗に気づくのが遅れる | cap 3 の最後の pipeline で、skill の数行と Go のコメント 1 行のために run をもう 1 回回す価値がない | `WaitInbox` の時間切れの文を変えるとき、`/director` skill の受信箱の節を直すとき、8b で director の起こし口を足すとき | `docs/reports/self-review-2026-10-14-org-director-attach.md`(cycle 3 の C3-L1〜C3-L3) |

_(登録簿には追記していない。登録簿の編集は記録だけの例外に入らないので、直さずに進めるときは、この cycle の `/sync-docs` で 1 行にまとめて足す。)_

### Coverage gaps

- skill の文を読んで実際のエラーと突き合わせる test はない(C3-L1)。skill の本文を固定している test が repo にあるかは調べていない
- 終了コード 1 以外の終わり方(外からの kill、Codex の shell tool 自体の timeout)には、skill のどの文も指示を持たない。この直しの前からの穴で、300000 ms の wait が Codex の tool の制限に収まるかも書かれていない。Codex の既定は未確認
- Monitor が stderr を director に渡すかは未確認(C3-L2)
- `/org` skill の `wait` の行(173 行)と `wait` の help は「届かなければ終了コード 1」と書く。時間切れの説明としては正しく、ほかの失敗も 1 で終わることは書かない。director が読むのは `/director` skill の節なので、指摘にしない

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。LOW 3 件は skill の文と Go のコメントで、c8a9227b の動きに欠陥はない)
- Follow-ups:
  1. C3-L1〜C3-L3 を直さず進めるなら、この cycle の `/sync-docs` で登録簿に 1 行にまとめて足す。cap 3 の最後の pipeline なので、この行が唯一の受け皿になる
  2. 直すなら、C3-L2 と C3-L3 は skill の同じ節の数行で、C3-L1 は Go のコメント 1 行。skill は 4 面を揃える(`.agents` 側は `scripts/sync-skills.sh`)。pipeline をもう 1 回回すことになるので、cap をもう一度上げるかどうかは人が決める
  3. plan の Progress checklist に、cross-review cycle 2 の ACTION_REQUIRED と c8a9227b の行がない(cycle 2 の節の follow-up 4 と同じ種類)。`/pr` が plan を archive するので、`/sync-docs` で足すとよい(diff の品質の外なので、指摘には数えない)
- Known gaps: テスト、`gofmt`、`go vet` は実行していない。skill の文を実際の Codex と Claude Code の session で動かしていない。Monitor の stderr の扱いと Codex の shell tool の timeout は未確認

### Fixes after the cycle-3 review (before /verify)

この節はオーケストレーターが足した。直しはコミット 8ac59595(`fix: address the cycle-3 self-review findings for org-director-attach`、7 ファイル)。上の重大度と `Merge:` の行は変えていない。

| ID | 対応 |
| --- | --- |
| C3-L1 | `WaitInbox` の時間切れの文の直前に、`/director` skill がこの文を照合して時間切れと失敗を見分けること、文を変えるなら skill の受信箱の節も直すことをコメントで書いた。文を固定する 2 つのテスト(`internal/org/escalate_test.go`、`internal/cli/org_inbox_test.go`)にも skill を指す 1 行のコメントを足した。文そのものは変えていない |
| C3-L2 | Claude Code の項を `ralph org wait --inbox --timeout-ms 0 2>&1` にし、`--timeout-ms 0` には時間切れがないので、件の行を出さずに返ったとき(終了コード 1)はいつも失敗で打ち直さない、と書いた。Monitor が stderr をどう扱うかは書いていない(未確認) |
| C3-L3 | 共通の段落の「止まる」を、受信箱を待つのをやめ、エラーの文を人に伝えて人の指示を待つ、detach は打たないので登録は残り、見張りが動いていれば件は期限で人に届く、と書き直した。昇格の 5(detach を打たない)と同じ扱い。「ack の期限」でなく「期限」と書いたのは、ack 済みの件が届くのは resolve の期限のため |

確かめたこと(implementer の報告): `gofmt -l internal` は出力なし、`go vet ./internal/org/ ./internal/cli/` は通過、`go test -count=1 ./internal/org/ ./internal/cli/ -run 'WaitInbox|Inbox'` は ok、`check-skill-sync.sh` と `check-sync.sh` は通過、skill の 4 面は md5 が同じ
