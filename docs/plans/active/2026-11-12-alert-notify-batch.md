# 見張りの ALERT の desktop notification を周期ごとにまとめ、`wait --inbox` を種類で絞る(#307 の後半)

- Status: Approved
- Approved: 2026-11-12 sha256:a805c3b2b4b4
- Owner: Claude Code
- Date: 2026-11-12
- Related request: https://github.com/yoshpy-dev/ralph/issues/307(前半は PR #314 で merge 済み。この PR は受け入れ条件 4 と、通知のまとめ・切る設定を扱う)
- Related issue: 307
- Type: fix
- Branch: fix/alert-notify-batch

## Objective

横断の見張りが人へ送る ALERT の件の desktop notification を、送りのまとまり(期限の掃き出し、org ごとの deadman の段)ごとに 1 回へまとめ、設定で切れるようにする。`ralph org wait --inbox` に種類の絞り込みを足し、ALERT を除いて待てるようにする。director skill の件の処理の順を、答えが要る件を先にする形に直す。issue #307 の受け入れ条件 4 を満たし、PR #314 と合わせて #307 を閉じる。

## Scope

- `[org.watchdog].alert_notify`(`"batch"` が既定、`"off"`)。ほかの値は設定の検証で断る。テンプレートの `ralph.toml` と既定値の一致のテスト
- 人へ送る TYPE ALERT の件の desktop notification を、送りのまとまりごとに 1 回にまとめる。まとまりは 2 種類で、期限の掃き出し(`SweepInboxDeadlines`)1 回と、org 1 つの見回りの deadman の段(`checkDeadman`)1 回。それぞれの終わりに、その場で 1 回送り、各件に `notified` の event を残す。`escalations.jsonl` の行と banner は今までどおり 1 件ずつすぐ出す。文は件数と最も古い件(最小の ID)と、その org
- `alert_notify = "off"` では ALERT の件の desktop notification を出さず、`notified` の event の `osascript` に `off` を残す
- `ralph org wait --inbox [--director <id>] --types <QUESTION,BLOCKED,RESULT,ALERT のうち、コンマ区切り>`。指定した種類の open の件があるときだけ返り、その件だけを出す。既定はすべての種類(今と同じ)。知らない種類は待つ前に断る。`--director` の終了コード 4 の「open の件が無い」も、指定した種類で数える
- director skill(4 つの写し): 件を処理する順を QUESTION・BLOCKED → RESULT → ALERT にする。ALERT の件の `recurred` の回数の読み方、`--types` の使いどころ(既定の待ちを続ける理由)。org skill(4 つの写し)の `wait` と `inbox` の行
- `ralph org watch --all` と `ralph org wait` の help、`docs/specs/2026-10-07-org-multi-org-director.md` の注、tech-debt の「Eleven gaps」の行のうち #307b に回した項目

## Non-goals

- push の wake(見張りが director を起こす 1 行)の種類の絞り込み
- leader の件(QUESTION・BLOCKED・RESULT)、人が手で打つ `ralph org inbox notify`、受信箱に書けないときの予備の道の通知(どれも今までどおり 1 件ずつ)
- 通知の文の多言語化、通知先の追加(Slack など)
- #307a の既知の穴(落ちうる窓 W1〜W9 など)の払い

## Assumptions

- 見張りの 1 周期は `watchAllRun.cycle`(`internal/org/watch.go`)で、期限の掃き出し(`SweepInboxDeadlines`)、director の確かめと wake、座席の見回り(`pulse`)、印の書き込みの順に進む。人への送りはすべて `sendInboxItemToHuman`(`internal/org/escalate.go`)を通る
- `notified` の event の `osascript` の欄は `ok`・`failed: <err>`・`skipped` を持つ。読み手は値を表示するだけで、`inbox show` と期限の判定(`InboxDeadlineDue`)は event の有無しか見ない
- 2 つの待ち(`WaitInbox` と `WaitInboxForDirector`)は、どちらも `openInboxItems` で open の件を数える

## Affected areas

- `internal/config/config.go`(`OrgWatchdogConfig.AlertNotify`、既定値、検証)、`internal/config/defaults_sync_test.go` ほか、`templates/base/ralph.toml`
- `internal/org/escalate.go`(`sendInboxItemToHuman` のまとめ・切る道、まとめの記録と送り)、`internal/org/watch.go`(周期の始まりと終わり)、`internal/org/deadline.go`(コメント)
- `internal/org/escalate.go` と `internal/org/director_wait.go`(`openInboxItems` の種類の絞り込み、`WaitInbox` と `DirectorWaitParams` の引数)
- `internal/cli/org.go`(`wait` の `--types` と help)、`internal/cli/org_watch.go`(help)、それぞれのテスト
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と、その `.agents/` と `templates/base/` の写し
- `docs/specs/2026-10-07-org-multi-org-director.md`、`docs/tech-debt/README.md`

## Visual review

- Page: `.harness/state/plan-visual/alert-notify-batch.html`
- やさしい図 2 枚(overview、easy-wait)、折りたたみの中に detail-overview と detail-cycle
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: 全体と 4 枚をそれぞれ `shot` で撮って読み、はみ出しや線の重なりが無いことを確かめた。計画の助言を反映したあと、overview と detail-cycle を送る段ごとの形に描き直し、lint と撮り直しで確かめた

## Design decisions

Critical forks: None。次の 6 点は、コードと既存の記録から決められるので自律で選んだ(ユーザーの 2026-11-12 の指示「今後も自律的に進めて」と 2026-10-11 の指示「計画は自動承認」に従う)。

- D1 設定は `[org.watchdog].alert_notify` に置く。ALERT の件の通知を送るのは見張りで、受信箱の期限の設定(`[org.director]`)とは別に切りたい人がいるため。値は `"batch"` と `"off"` の 2 つに絞る(1 件ずつの送りは、まとめが 1 件のときと同じになる)
- D2 まとめの単位は、期限の掃き出し 1 回と、org 1 つの見回りの deadman の段 1 回。どちらも、その段の終わりにその場で 1 回送る。見張りの 1 周期に出る ALERT の通知は、多くても 1 + (escalate した org の数)回になる。最も古い件は ID の数が最小の件。周期の終わりまで待つ形は取らない(計画の助言の指摘 1)。今の見張りは、期限の送りを director の確かめや座席の見回り(herdr や git を待ちうる)より先に済ませる約束(`RunWatchAll` の doc の 1)なので、それを崩さないため。deadman の段の escalate は、今もその org の座席の見回りのあとに走るので、まとめても遅れは増えない。周期の初めの突き合わせ(#307a の S12)の再送は件ごとのまま
- D2b 期限の掃き出しでは、まとめた送りの結果を件ごとに確かめ、`notified` の追記に失敗した件は今までどおり送りの失敗として掃き出しの記憶(`InboxDeadlineMemory`)に残し、ack の期限の長さだけ間をあける。追記できた件は `Sent` に数える(計画の助言の指摘 2)。desktop notification そのものの失敗は、今と同じく `notified` の `osascript` に `failed: <err>` として残る
- D3 `"off"` でも件は人への経路に送ったものとして扱い(banner と `escalations.jsonl` は出す)、`notified` の event の `osascript` に `off` を残す。期限で送り直さないため
- D4 まとまりの送りの前に見張りが止まると、そのまとまりの ALERT の件には `notified` の event が残らない。次の見張りの ack の期限で送り直される(知らせる側に倒れる)。banner は 2 回出うる
- D5 director skill の待ちは既定(すべての種類)のまま。ALERT を外して待つと、director が見ないあいだに ack の期限で ALERT が人へ送られるため。代わりに件を処理する順を QUESTION・BLOCKED → RESULT → ALERT にし、`--types` は「答えが要る件だけを待ちたいとき」の手段として、期限のことと一緒に書く
- D6 `--types` の値は受信箱の 4 つの種類(QUESTION、BLOCKED、RESULT、ALERT)だけを大文字で受け付ける。空の項目や知らない種類は待つ前に終了コード 1 で断る。重複は許す

## Acceptance criteria

- [ ] AC1 期限の掃き出し 1 回で人へ送る ALERT の件が 2 件以上あると、desktop notification は 1 回だけ送られ、その文に件数と最も古い件と、その org が出る。org 1 つの見回りの deadman の段で escalate する ALERT の件も同じ。各件の `notified` の event に同じ結果が残り、`escalations.jsonl` の行と banner は件ごとに出る
- [ ] AC1b 期限の掃き出しのまとめた送りは、director の確かめや座席の見回りより前に終わる(後の段を止めても、期限の ALERT の通知は送られている)
- [ ] AC1c 期限の掃き出しで `notified` の追記に失敗した件は送りの失敗として記憶され、ack の期限の長さのあいだ送り直されない。追記できた件は送り直されない(複数の周期で確かめる)
- [ ] AC2 ALERT の件が 1 件だけのときは、今までと同じ文で 1 回送られる
- [ ] AC3 `alert_notify = "off"` では、ALERT の件の desktop notification は出ず、`notified` の event の `osascript` が `off` になる。banner と `escalations.jsonl` は出る。期限で送り直されない
- [ ] AC4 leader の件(QUESTION・BLOCKED・RESULT)、`ralph org inbox notify`、受信箱に書けないときの予備の道は、今までどおり 1 件ずつ送られ、`alert_notify` を読まない
- [ ] AC5 まとまりの送りの前に止まった見張りの件は `notified` の event を持たず、次の見張りの ack の期限で送られる
- [ ] AC6 `alert_notify` の既定は `"batch"` で、ほかの値は設定の検証で断る。テンプレートの `ralph.toml` に既定値とコメントがあり、既定値の一致のテストが通る
- [ ] AC7 `ralph org wait --inbox --types QUESTION,BLOCKED,RESULT` は、open の ALERT の件だけがあるあいだは返らず、指定した種類の件が届けば、その件だけを出して返る。`--types` を付けないときは今と同じ。知らない種類と空の項目は待つ前に断る(issue の受け入れ条件 4)
- [ ] AC8 `--director` と `--types` を一緒に使うと、終了コード 4 の判定と印の件数は指定した種類の件だけで数える
- [ ] AC9 director skill と org skill(どちらも 4 つの写し)が、処理の順、ALERT の回数の読み方、`--types` と `alert_notify` を書き、`scripts/check-skill-sync.sh` が通る
- [ ] AC10 `ralph org watch --help` と `ralph org wait --help`、仕様の注、tech-debt の行がこの挙動に合う

## Implementation outline

1. 設定 `[org.watchdog].alert_notify` と検証、テンプレート、既定値の一致のテスト — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/`
2. ALERT の件の desktop notification のまとめと `off`。`sendInboxItemToHuman` が TYPE ALERT の件で設定とまとめの有無を見る。期限の掃き出しと、org ごとの deadman の段で、それぞれまとめを始めて段の終わりに 1 回送り、各件に `notified` を足す。掃き出しは件ごとの結果で記憶を更新する — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Inbox|Escalat|Watch|Deadline|Notify'`
3. `wait --inbox --types`(`openInboxItems` の絞り込み、`WaitInbox` と `WaitInboxForDirector` の引数、CLI のフラグと help と検証) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/ -run 'Wait|Director|Inbox'`
4. director skill と org skill(4 つの写しずつ)、`ralph org watch --help`、仕様の注、tech-debt — verify: `./scripts/check-skill-sync.sh && test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Watch|Skill' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC10。issue #307 の受け入れ条件 4 がこの PR で満たされ、1〜3 は PR #314 で満たされたことを PR 本文に書く(`Closes #307`)
- Documentation drift to check: help の 2 つ、director と org の skill の 4 つの写し、仕様 FR-5・FR-6 の注、AGENTS.md の internal/org の項(受信箱の通知の書き方)、テンプレートの `ralph.toml`
- Evidence to capture: まとめのテストで数えた desktop notification の回数(まとめる前は件の数、まとめたあとは 1)

## Test plan

- Unit tests:
  - 設定: 既定値、`"batch"` と `"off"` を読む、ほかの値と空文字を断る
  - まとめ: 期限の掃き出しで 3 件、deadman の段で 2 件の ALERT の件を送る周期で、送りが 2 回(掃き出しと org の段に 1 回ずつ)、文に件数・最も古い件・org、各件に同じ結果の `notified`。1 件のときの文は今までと同じ。ALERT が無い段では送らない
  - 送る順: 座席の見回り(herdr の偽物)を止めても、掃き出しのまとめた送りは先に済んでいる(AC1b)
  - 掃き出しの記憶: `notified` の追記を失敗させた件が、次の周期に送り直されず、ack の期限の長さのあとに送り直される。追記できた件は送り直されない(AC1c)
  - `off`: 送りが 0 回、`notified` の `osascript` が `off`、期限で送り直さない
  - 範囲の外: leader の件、`inbox notify`、予備の道は周期の中でも 1 件ずつ送られる
  - 止まった見張り: 送りの前に止めた件に `notified` が無く、次の見張りの ack の期限で送られる
  - 送りの失敗: 各件の `notified` に `failed: <err>` が残る(今の 1 件ずつの送りと同じ扱い)
  - `--types`: ALERT だけのあいだ返らない、QUESTION が届くと返りその件だけを出す、既定は今と同じ、知らない種類・空の項目・小文字の断り、`--director` との組み合わせで終了コード 4 の判定が種類で数える
- Integration tests: `RunWatchAll` を 1 周期回し、偽の通知の回数と受信箱の event を数える
- Regression tests: 既存の受信箱・escalate・期限・見張り・director の待ちのテストがそのまま通ること
- Edge cases: まとめの中の件が周期の中で resolve された(それでも送りは記録)、`notified` の追記の失敗、ALERT の件の org の値が識別子でない(今と同じ `inboxNotifyInvalid`)
- Evidence to capture: `go test` の結果、テストで数えた送りの回数

## Risks and mitigations

- まとめで desktop notification が遅れる。対策: まとまりは期限の掃き出しと org ごとの deadman の段で、どちらもその段の終わりにすぐ送るので、今の送りより遅れるのは同じ段のほかの件の書き込みの分だけ。banner と `escalations.jsonl` は今までどおりすぐ出る
- まとめた送りの結果の扱いを誤ると、書けない受信箱で毎周期通知が出る。対策: 掃き出しは件ごとの結果で失敗の記憶を更新する(D2b、AC1c)
- 送りの前に見張りが止まると banner が 2 回出うる。対策: 知らせる側に倒れる向きなので受け入れ、D4 として記録する
- director skill の写しの同期を忘れる。対策: `scripts/check-skill-sync.sh` と `internal/org/director_skill_sync_test.go` が落とす
- `--types` の値の解釈の違い(小文字、空白)。対策: 受け付ける形を 1 つに決め、断るときの文で受け付ける値を出す

## Rollout or rollback notes

- 動いている見張りには、止めて新しい binary で立て直すまで効かない(release ノートの既存の項目と同じ)
- 新しい設定のキーは、古い binary では知らないキーとして扱われる。release ノートに「`alert_notify` は新しい binary で読む」を足す
- skill の変更は `ralph upgrade` で下流に届く
- 戻すときは revert する。受信箱に残った `osascript: off` の `notified` は、古い binary も読める(値を表示するだけ)

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 2 件を計画に反映した。(1) [HIGH] 周期の終わりに 1 回送る形は、director の確かめや座席の見回りが詰まると期限の通知を遅らせ、期限の送りを先に済ませる見張りの約束を崩す → まとまりを期限の掃き出しと org ごとの deadman の段に分け、段の終わりにすぐ送る(D2、AC1b)。(2) [MEDIUM] 後回しの記録は掃き出しの失敗の記憶を迂回する → 掃き出しは件ごとの結果で記憶を更新する(D2b、AC1c)
- [x] Plan approved — ユーザーの 2026-11-12 の指示(今後も自律的に進める)と 2026-10-11 の指示(計画は自動承認)に従って承認を記録した
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
