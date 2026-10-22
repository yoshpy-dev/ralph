# watch-escalation-inbox

- Status: Approved
- Approved: 2026-10-23 sha256:0bf5660a7229
- Owner: Claude Code
- Date: 2026-10-23
- Related request: 試行の所見を直す系列の 6 本目「見張りの escalation」(ユーザーの依頼「2. 修正可能なものは修正してください」、2026-10-21)
- Related issue: N/A
- Type: fix
- Branch: fix/watch-escalation-inbox

## Objective

横断の見張り(`ralph org watch --all`)の escalation を、org の受信箱の件(TYPE `ALERT`)として記録する。director が有効で、見張りが仕事を受け付けているときは、leader の escalate と同じく director に任せ、ack の期限で人に届ける。そうでないときは、記録と同時に人へ送る。どちらの道でも、通知を試した結果を記録に残す。試行(stage 9)の所見 F6(見張りの escalation 30 件が director を通らずに人へ直接届いた)と F13(見張りの escalation の道が通知の結果を捨てる)を閉じる。あわせて、記録に残る「fix/split-preamble」を「#251」に直す。

## Scope

- 見張りの escalation(`internal/org/watch.go` の `escalateAlert`。理由は `leader_is_anomaly_subject` と `deadman_timeout`)を、受信箱の件として記録する
  - 件の TYPE は `ALERT`。本文は ralph が作る typed message で、org、対象の座席、条件(stall・liveness・scope_change など)、理由、alert_id、元の ALERT の時刻を持つ
  - 受信箱の `escalated` の event に `alert_id` の欄を足す。見張りは書く前に同じ alert_id の件を探し、あれば書かない(重複を防ぐ)
  - 届け方は、leader の escalate の判定(`escalateDelivery`)と共有する関数で決める。`[org.director].enabled` で、この見張りが `--once` でなく、止める頼みも出ていなければ delivery `deadline`。そうでなければ delivery `now` で、記録と同時に人への道(`sendInboxItemToHuman`)で送る
- 人への道を ALERT の件に合わせる
  - `notifiableType` が ALERT を `<invalid>` にしない。escalate が受け付ける TYPE の一覧(`escalateTypes`)とは別に、受信箱の件の TYPE の一覧を持つ。`ralph org escalate` が ALERT を断る動きは変えない
  - ALERT の件を人へ送るとき、escalations.jsonl の行に alert_id と対象の座席を残し、デスクトップ通知の題は `ralph org watch` にする
- 受信箱に書けないとき(受信箱がない、読めない、書けない)と、件の見出しに入れる見張りの状態ファイルの値が検査を通らないときは、今の道に落とす。stderr の banner、通知、escalations.jsonl の 1 行の順にし、行には新しい欄 `osascript` で通知の結果を書く。値は受信箱の `notified` と同じ(`ok`、`failed: <err>`、`skipped`)にする(F13)
- director の扱い: `/director` skill(4 写し)の「受信箱の処理」と、雛形 `internal/org/prompts/director.md` に ALERT の件の扱いを足す。ALERT は ralph が書いた件である。director は `ralph status`、`ralph org read --seat leader`、`ralph org report` を見て、leader に知らせるか、人に notify するところまでを行う。座席や org を止めるかは人が決める
- 文書
  - `/org` skill(4 写し)の動詞の表の `watch` の行(「デッドマン時の人間エスカレーション」)と「受信箱」節
  - `.claude/rules/ralph/agent-messaging.md` の「Escalation to the org inbox」(と `templates/base/` の写し)
  - 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-6 の注。director の resolve が見張りの escalation を人に届けなくしうること、遅れの上限、org-runtime 仕様の「人間エスカレーションはパルス層からのみ発火し、LLM が抑止できない」との関係を書く
  - `ralph org watch` の help の escalation の文
- 記録
  - tech-debt の F6 と F13 の行を RESOLVED にする
  - 根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` の F6・F13 の行と、FR-6・NFR 安全の不変条件の行を直す
  - 仕様・根拠の表・tech-debt に残る「fix/split-preamble」を「#251」に直し、PR 一覧を 34 本にする

## Non-goals

- leader への ALERT の送信が時間切れになる件(tech-debt の「The watch's ALERTs to the leaders failed 10 times」の行)。別の所見で、この PR では扱わない
- ALERT の誤検知を減らすこと(stall や scope_change の判定を変えること)
- leader の escalate の届け方を変えること(判定の関数を共有するだけで、動きは変えない)
- director が座席や org を自分で止めること
- 受信箱の件をまとめて resolve する動詞
- 実機の確かめ(本物の herdr と director のセッション)。Go のテストで見る

## Assumptions

- 書き手は `watch.lock` を持つ見張り 1 つなので、同じ alert_id の件を探してから書くあいだに、ほかの見張りが同じ件を書くことはない
- 受信箱の fold(`internal/org/inbox.go`)は TYPE が空でないことだけを見て、JSON の知らない欄は無視する。そのため、古い binary も ALERT の件と `alert_id` の欄を読める
- 受信箱の期限(`InboxDeadlineDue`)は TYPE を見ない。delivery `deadline` の ALERT の件は、open で `notified` がなければ ack の期限で、resolve されなければ resolve の期限で人に届く
- escalations.jsonl を読むコードはテストのほかにない(consult が rg で確かめた)ので、欄を足しても読み手は壊れない
- 本番の通知は、見張りの `Escalate` と受信箱の `DesktopNotify` のどちらも、CLI では同じ `orgDesktopNotifyOverride`(本番は nil で osascript)を使う(`internal/cli/org_watch.go`、`internal/cli/org.go`)

## Affected areas

- `internal/org/watch.go`、`internal/org/watch_test.go`、`internal/org/watch_all_test.go`
- `internal/org/escalate.go`、`internal/org/escalate_test.go`
- `internal/org/inbox.go`、`internal/org/inbox_test.go`
- `internal/org/deadline.go`(ALERT の件の deadline の文が要れば)、`internal/org/deadline_test.go`
- `internal/org/prompts/director.md`、`internal/org/prompts_test.go`
- `internal/cli/org_watch.go`(help)と、その文を固めるテスト
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と写し(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)
- `.claude/rules/ralph/agent-messaging.md` と `templates/base/.claude/rules/ralph/agent-messaging.md`
- `docs/specs/2026-10-07-org-multi-org-director.md`
- `docs/tech-debt/README.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`

## Visual review

- ページ: `.harness/state/plan-visual/watch-escalation-inbox.html`(かんたん版の図 2 つ: `overview`、`easy-direct`。折りたたみの中に `detail-overview`、`detail-flow`、`detail-records`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- 自己点検: 全体、かんたん版の 2 つ、詳細の 3 つを `shot` で撮って読んだ。詳細 1 の記録の箱から文字がはみ出していたので幅を広げ、撮り直して直ったことを確かめた。Codex の計画助言を入れたあと、詳細 2 の重複の箱と期限の箱、承認の前の箱を直し、lint(exit 0)と撮り直しで確かめた
- Codex の計画助言: 2 件(HIGH 1: 重複の判定が送り残しを捨てる、MEDIUM 1: ack のあとの遅れの上限)。どちらも D4・D6・AC2・AC4・Test plan・Risks・Rollout に入れた

## Design decisions

- D1 方式(consult-plan-watchesc の案 A): 見張りの escalation を受信箱の件にし、leader の escalate と同じ判定と人への道を使う。F13 の通知の結果は、受信箱の `notified` がそのまま持つ。案 B(F13 だけ直し、director には写しだけ送る)では F6 の「人に 5 分ごとに通知が来る」が直らない。案 C(director の生存の確かめが新しいときだけ任せる)は、仕様 FR-9 の「登録の有無で escalate の動きは変わらない」とずれる
- D2 TYPE: 件の TYPE は `ALERT` にする。protocol の TYPE の表で、見張りが異常を知らせるときの TYPE が ALERT なので、それに合わせる。受信箱の件に別の印(source の欄など)は足さない。`ralph org escalate` は今までどおり ALERT を断るので、ALERT の件を書けるのは見張りだけになる
- D3 届け方の判定: `escalateDelivery` の中身を、Enabled と「仕事を受け付けているか」を受け取る関数に切り出し、leader の escalate と見張りの両方が使う。見張りの側の「受け付けている」は、そのサイクルの見張り自身について、`--once` でなく(`WatchAllParams.Cycles == 0`)、止める頼みが出ていないこと(`stopRequested`)とする。止まる直前の見張りが delivery `deadline` で書いた件は、次の見張りが起動するまで誰も期限を見ないため、そのときは今すぐ送る
- D4 重複: escalated の event に `alert_id` を持たせ、見張りは書く前に受信箱を読んで同じ alert_id の件を探す。重複の判定(件を 2 つ書かない)と、届け終えたか(通知を終えたか)は分けて扱う。同じ alert_id の件があれば新しい件は書かない。その件が open で `notified` がなく、今の見張りの判定(D3)が delivery `now` のときは、その判定の理由を付けて同じ ID で人への道に送る。それ以外(届け済み、ack 済み、resolve 済み、今の見張りが任せられる)は、受信箱にもうあることを stderr に 1 行書くだけにし、director と受信箱の期限に任せる(open で `notified` のない件なので、ack の期限で人に届く)。件を書いた見張りの判定ではなく今の見張りの判定で決めるのは、新しい件と同じ規則にそろえ、送る理由を今の見張りの事実と合わせるため(実装中の consult。2026-10-23 の逸脱の注)。status ファイルの保存は org のサイクルの最後なので、件を書いたあと、通知の前やその保存の前に落ちた見張りが、次の起動で同じ alert を escalate しても、件は 1 つのままで、送り残しは送られる(Codex の計画助言 1)
- D5 受信箱に書けないとき: 今の道(banner、通知、escalations.jsonl の 1 行)に落とす。件の見出しに入れる見張りの状態ファイルの値(対象の座席が座席の名前でない、alert_id が見張りの書く形でない、alert_id か元の時刻が改行などの制御文字を含む)が検査を通らないときも、件を書かずに同じ道に落とす。director が読む本文の見出しが増えないようにするためで、banner の値は逃がしてから出す(self-review F4、d9b8cd9a)。行は通知のあとに 1 行で書き、`osascript` の欄に結果を入れる。通知の前に書いて結果を 2 行目にする案は、行の数が増えるだけで読み手が得るものがないので採らない
- D6 安全の性質: director が resolve した ALERT の件は、人に届かない。org-runtime 仕様の「人間エスカレーションはパルス層からのみ発火し、LLM が抑止できない」は、機能ごとの org の仕様が優先する(org-runtime 仕様の 2026-10-07 の注)と読み、FR-6 の注に書く。全 LLM が止まれば、ack と resolve の期限で人に届くので、NFR 安全の不変条件(期限切れの件の人への通知)は保たれる。人に届くまでの遅れの上限は、件の状態で分かれる。open のまま残った件は、escalation から最大 `ack_minutes` + 見張りの間隔で届く。director が ack してから止まった件は、ack の期限から外れるので(`InboxDeadlineDue`)、escalation から最大 `resolve_minutes` + 見張りの間隔で届く(既定で 60 分と間隔)。deadman_timeout の件は、どちらにも、ALERT から escalation までの `deadman_minutes` が先に加わる。ALERT の件だけ ack で ack の期限を外さない案は、leader の件と規則が分かれるので採らない(Codex の計画助言 2)
- D7 director の役目: ALERT の件に対して、director は状態を読み、leader に知らせるか人に notify するまでにする。座席や org を止めるのは人の判断にする。director が止めると、LLM が人への道を閉じる形が増えるため
- D8 director が無効のとき: ALERT の件も、leader の件と同じく記録と同時に人へ送り、resolve の期限も効く。人が resolve しない ALERT の件は、`resolve_minutes`(既定 60 分)のあとにもう 1 回人に届く。leader の件と同じ規則にそろえ、`/org` skill の「受信箱」節に、人が ALERT の件を resolve することを書く
- D9 通知の手段: 受信箱の道の通知は、leader の件と同じく `Org.DesktopNotify` を使う(`sendInboxItemToHuman`)。受信箱に書けないときの今の道は、今までどおり見張りの `WatchAllParams.Escalate` を使い、nil で darwin でなければ通知せずに `skipped` と書く。CLI ではどちらも同じ `orgDesktopNotifyOverride`(本番は nil で osascript)なので、本番で使う手段は変わらない。見張りの「受け付けている」は、org ごとの `watchRun` が `watchAllRun` から受け取る(`--once` は起動時に決まり、止める頼みはサイクルごとに読む)
- Critical forks: None(D1〜D8 は consult と相談して決めた。どれも 1 スライスの中で戻せる)

## Acceptance criteria

- [x] AC1: 見張りの escalation は、受信箱に TYPE `ALERT` の件を 1 つ書く。本文は `protocol.Parse` と `protocol.Validate` を通る typed message で、見張りの ALERT の代わりの文(`sendAlert`)と同じ見出し `ORG_ID`・`SEAT`・`CONDITION` に、`REASON`(`leader_is_anomaly_subject` か `deadman_timeout`)、`ALERT_ID`、`ALERT_TS`(元の ALERT の時刻)を足した見出しと、1 行の説明を持つ。escalated の event と、それを畳んだ件(`InboxItem`)は `alert_id` の欄を持つ。欄のない古い行も今までどおり読める
- [x] AC2: `[org.director].enabled` で、見張りが `--once` でなく止める頼みも出ていないとき、件は delivery `deadline` で記録だけされ、escalations.jsonl の行も通知もない。stderr には `WATCHDOG ESCALATION` で始まり、件の ID と director に任せたことを言う 1 行が出る。その件は、ack の期限を過ぎて open なら、次のサイクルの `SweepInboxDeadlines` で人に届く(理由 `inbox_ack_deadline`)。ack されてから resolve されない件は、resolve の期限を過ぎたサイクルで人に届く(理由 `inbox_resolve_deadline`、D6 の ack のあとの上限)
- [x] AC3: director が無効のとき、`--once` の見張りのとき、止める頼みが出ているときは、件は delivery `now` で記録され、同じサイクルで人への道に送られる。送る理由は leader の件と同じ `inbox_director_disabled` か `inbox_no_watch` にする。escalations.jsonl の行は `inbox_id`・`alert_id`・対象の座席(`subject`)を持ち、`subject` が `leader` なら元の理由は `leader_is_anomaly_subject`、それ以外なら `deadman_timeout` と読める。`notified` の event は通知の結果を持ち、通知が失敗すると `failed: <err>` になる(F13)。人への道を終えられなかったとき(escalations.jsonl に書けないなど)は、stderr に `ralph org inbox notify <id>` を示す 1 行を出す。その件は open で `notified` がないので、ack の期限でも人に届く
- [x] AC4: 同じ alert_id の件がもう受信箱にあれば、見張りは新しい件を書かない(落ちた見張りの再起動の形)。その件が届け済み、ack 済み、resolve 済みなら通知もしない。その件が open で `notified` がなく、今の見張りが任せられない(判定が delivery `now`)なら、同じ ID で人への道に送る(件を書いたあと通知の前に落ち、ack の期限の前に `--once` で起動し直した形を含む)。今の見張りが任せられるなら送らず、stderr に 1 行書き、件は ack の期限で届く。どの場合も、escalate した alert は pending から消える
- [x] AC5: 受信箱に書けないとき(Inbox が nil、読めない、書けない)と、見出しに入れる値が検査を通らないとき(D5)は、stderr の banner、通知、escalations.jsonl の 1 行の順で今の道に落ちる。行は `alert_id`、`subject`、`reason` と、通知の結果の `osascript`(`ok`、`failed: <err>`、通知の手段がなく darwin でもないときは `skipped`)を持つ。通知が失敗すると `osascript` は `failed: <err>` になる(F13)
- [x] AC6: 人への道の ALERT の件では、記録と同時に送るときも期限で送るときも、デスクトップ通知が org、件の ID、TYPE `ALERT` を出し(`<invalid>` にならない)、題は `ralph org watch` になる。期限で送るときの escalations.jsonl の行も `alert_id` と `subject` を持つ。本文は通知に出さない。`ralph org escalate` は TYPE `ALERT` を今までどおり断り、何も書かない
- [x] AC7: leader の escalate の届け方は変わらない(既存の escalate のテストが、期待を変えずに通る)。届け方の判定は、leader の escalate と見張りが同じ関数で決める
- [x] AC8: 文書を合わせる。`/director` skill(4 写し)の「件の処理」に ALERT の件の扱い(D7)、雛形 `director.md` にその 1 行、`/org` skill(4 写し)の `watch` の行と「受信箱」節(D8)、`agent-messaging.md`(2 写し)の escalation の節、`ralph org watch` の help、仕様の FR-6 の注(D6)。写しはバイト一致。prompts_test.go が director の雛形の 1 行を固める
- [x] AC9: 記録を合わせる。tech-debt の F6 と F13 の行を取り消し線と RESOLVED の注で閉じる。根拠の表の F6・F13 の行の直す PR、FR-6 と NFR 安全の不変条件の行を `[x]` にする(ほかの所見が残っていないことを確かめる)。仕様の FR-6 と安全の不変条件を `[x]` にし、印の数を直す(`[x]` 28 / `[ ]` 4、根拠の表は 27 / 4 の見込み)。「fix/split-preamble」を「#251」に直し、PR 一覧を 34 本にする。この PR は merge の前のブランチ名 `fix/watch-escalation-inbox` で書き、番号は次の PR が書く
- [x] AC10: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0。変えたパッケージの Go のテストが `GOTOOLCHAIN=go1.25.8` でも通る

## Implementation outline

1. S1(implementer): D2・D3・D4・D6 の受信箱の側。受信箱の `alert_id` の欄、受信箱の件の TYPE の一覧と `notifiableType`、届け方の判定の関数の切り出し、ALERT の件の人への道(escalations.jsonl の行の alert_id と座席、通知の題)。inbox_test.go と escalate_test.go(AC6、AC7 と、AC3 の人への道の部分)
2. S2(implementer): D1・D3・D4・D5 の見張りの側。`escalateAlert` が件を書き、届け方を決め、重複を探し、書けなければ今の道に落ちて `osascript` を残す。`watchAllRun.cycle` がそのサイクルの「受け付けている」を渡す。watch_test.go と watch_all_test.go(AC1〜AC5)。今のテストの期待(escalations.jsonl の行の数、banner)は、受信箱のない Org では今の道のまま通るように見て、変える期待は報告に書く
3. S3(implementer): 文書(AC8)。skill の写しは `scripts/sync-skills.sh` で作り、`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` で確かめる
4. S4(implementer、sonnet): 記録(AC9)

## Verify plan

- Static analysis checks: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh`(gofmt、go vet、shellcheck、check-sync、check-skill-sync ほか)
- Spec compliance criteria to confirm: AC1〜AC10。仕様の印の数が FR-6 と安全の不変条件の 2 つだけ動くこと
- Documentation drift to check: 見張りの escalation がどこへ行くかを書いた文(`/org` と `/director` の skill、雛形、agent-messaging.md、仕様、`ralph org watch` の help、`escalateAlert`・`realEscalate`・`WatchAllParams.Escalate` の doc)が食い違わないか。「デッドマン時の人間エスカレーション」「直接人へ」の古い文が残っていないか
- Evidence to capture: run-verify の rc と時間

## Test plan

- Unit tests: inbox_test.go に `alert_id` の欄の書き込みと読み戻し、古い形の行(欄なし)の読み込み。escalate_test.go に、受信箱の件の TYPE の一覧と `notifiableType`(ALERT が通る、escalate は ALERT を断る)、届け方の判定の関数(Enabled と受け付けの 4 通り)、ALERT の件の人への道(escalations.jsonl の行、通知の題と文、`failed:` の結果)
- Integration tests: watch_test.go の `evaluateCycle` で、受信箱のある Org の deadman と leader の異常が件になること(delivery の 2 通り)、重複(届け済み・ack 済み・resolve 済みの件は送らない、未通知で open の件は今の見張りが任せられないなら同じ ID で送り、任せられるなら件の delivery が `now` でも送らずに 1 行書く)、受信箱に書けないときの今の道と `osascript`(`ok`、`failed:`、`skipped`)。watch_all_test.go の `RunWatchAll` で、`--once`(Cycles 1)なら delivery `now`、長く動く見張りなら delivery `deadline` で、ack の期限を過ぎたサイクルで人に届くこと、ack したまま resolve されない件は resolve の期限を過ぎたサイクルで届くこと、止める頼みが出たサイクルでは `now` になること、件を書いたあと通知の前に落ちた形(受信箱に未通知の件があり、status に同じ alert が pending のまま)から `--once` で起動すると、同じ ID で届き件は 1 つのままであること
- Regression tests: 受信箱のない Org の今の deadman のテスト(escalations.jsonl の行の数、banner、1 回だけ)が通る。leader の escalate のテストが期待を変えずに通る。`TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem` が通る
- Edge cases: alert_id の座席の部分が `leader` のとき(理由 `leader_is_anomaly_subject`)、受信箱が読めるが書けないとき、DesktopNotify が nil で darwin でないとき(`skipped`)、同じサイクルで 2 つの org が escalate するとき
- Evidence to capture: `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` を go1.26 と go1.25.8 で

## Risks and mitigations

- director が resolve すると、見張りの escalation は人に届かない(LLM が人への道を閉じうる): D6 のとおり FR-6 の注に書く。全 LLM が止まれば期限で届く。director の skill は、ALERT の件で座席を止めず、人に知らせるかを判断する手順にする
- 人に届くのが遅れる。open の件は最大 `ack_minutes` + 間隔、director が ack してから止まった件は最大 `resolve_minutes` + 間隔(deadman はどちらもさらに `deadman_minutes`): D6 のとおり FR-6 の注と `/director` skill に分けて書く。急ぐ人は `[org.director].ack_minutes` と `resolve_minutes` を短くするか、director を無効にする
- 件を書いたあと、通知の前に見張りが落ちると、送り残しが出る: D4 のとおり、次の見張りが同じ alert を escalate するとき、任せられない見張りなら同じ ID で送り、任せられる見張りなら director と ack の期限に任せる。status ファイルが保存されたあとに落ちた場合は pending が残らないが、そのときは件の通知も済んでいる(通知は保存より前)
- 見張りが SIGTERM や kill で止まる直前に delivery `deadline` で書いた件は、次の見張りが起動するまで期限を誰も見ない(leader の escalate と同じ形): 止める頼みが出ているサイクルは `now` にする(D3)。kill は防げないので、FR-6 の注に書く。次の見張りの最初のサイクルで期限の切れた件は送られる
- director が無効なとき、人が resolve しない ALERT の件は resolve の期限でもう 1 回届く(通知が増える): D8 のとおり leader の件と同じ規則にそろえ、`/org` skill に書く。release のノートにも書く
- 受信箱を読むのが escalation のたびに 1 回増える: 書き手は見張り 1 つで、escalation は多くても数分に 1 回なので、読む量は受信箱の大きさだけ
- 既存の deadman のテストが、受信箱のある Org を使っていれば期待が変わる: S2 で受信箱のない Org のテストを今の道として残し、受信箱のある Org のテストを足す。変える期待は報告に書く
- 古い binary の見張りと新しい binary の director が混ざる: 古い見張りは今までどおり人へ直接送る。新しい skill は ALERT の件を扱うだけなので、件がなければ何もしない。release のノートに、新しい binary が要ると書く

## Rollout or rollback notes

- 効くのは次の release の binary から。scaffold の skill と規則の文は `ralph upgrade` で入る
- release のノートに書く文: 「横断の見張りの escalation は受信箱の ALERT の件になり、director が有効なら director に任され、open のままなら ack の期限で、ack されて resolve されないなら resolve の期限で人に届く。director が無効なら今までどおりすぐ人に届き、resolve しない件は resolve の期限でもう 1 回届く。新しい skill には新しい binary が要る」
- 戻すときはこの PR を revert する。受信箱に残った ALERT の件は、古い binary でも読めて resolve できる

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved(ユーザーの常設の指示により自動承認、2026-10-23)
- [x] Implementation started
  - S1 は bb15d508(受信箱の alert_id、受信箱の件の TYPE の一覧、届け方の判定 `inboxDelivery`、ALERT の件の人への道)
  - 逸脱の注(2026-10-23、S2 の実装中): S2 の consult の勧めで、重複した件を送るかを、件を書いた見張りの判定でなく今の見張りの判定だけで決めることにした(D4・AC4・Test plan・Risks を直し、図解の詳細 2 と承認の前の箱を直して lint と撮り直しで確かめ、digest を 95d7155105d7 から c6039d0bef22 に取り直して自動承認した)。Codex の計画助言 1 の形(落ちたあと `--once` で起動し直す)は、今の見張りが任せられないので同じ ID で送られる
  - S2 は 100b6932(見張りの escalation を受信箱の ALERT の件にする、今の道の `osascript`)、S2b は 6bc7864e(重複した件を今の見張りの判定だけで送る)
  - S3 は 587f5b6a(文書)と b278070c(consult の指摘で、2 つの仕様の注の範囲を「見張りが見るすべての org」に直し、resolve で止まるのは delivery `deadline` の件だけと書いた)
  - S4 は b1fb161b と a560bec6(記録)。self-review の LOW 9 件は d9b8cd9a で直した(F7 の引数の束ね直しだけ見送り、F9 と F1 の受信箱の道は tech-debt の行にした)
  - 逸脱の注(2026-10-23、verify の所見): self-review F4 の直し(d9b8cd9a)で、件の見出しに入れる見張りの状態ファイルの値が検査を通らないときも今の道に落ちるようになった。Scope・D5・AC5 に足し、図解の詳細 2 の分かれ道の札と注を直して lint(exit 0)と撮り直しで確かめ、digest を c6039d0bef22 から 0bf5660a7229 に取り直して自動承認した
  - release のノートに足す文(S3 の consult の勧め): 「resolve の期限は人に送ったかを見ないので、resolve されない ALERT の件は、最初の通知(director が無効か登録がないとき、`--once` か止める頼みの出た見張りのとき、director が `inbox notify` で送ったとき)のあと resolve の期限でもう 1 回届き、1 つの alert の通知が最大 2 回になる」。tech-debt にも 1 行残した(S4 の b1fb161b と a560bec6)
- [x] Branch created
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
  - cross-review は 86620383(cycle 1/2、指摘 0 件)。最後の `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は 95374404 から打ち、rc 0(1014 秒、internal/cli と internal/org は cache なし、秘密の走査は `e8e8d652..1cf9fc8b` で clean)。go1.25.8 は 08083792 で rc 0(376 秒)で、そのあとに Go のファイルの変更はない。AC1〜AC10 に印を付けた
- [ ] PR created
