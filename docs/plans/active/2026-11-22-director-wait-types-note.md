# director-wait-types-note

- Status: Approved
- Approved: 2026-11-22 sha256:574fe7322c7d
- Owner: Claude Code
- Date: 2026-11-22
- Related request: issue #374(director の `ralph org wait --inbox --director` が 2 つ重なると、どちらも新しい件を返さずに時間切れになることがある)
- Related issue: 374
- Type: fix
- Branch: fix/director-wait-types-note

## Objective

director の wait が受信箱の件を返さずに時間切れになったとき、なぜ返さなかったかが分かるようにし、重なった wait でも新しい件は返ることをテストで固定する。調べ(scratchpad の probe-374)では、issue の見立て(重なり)は原因ではない。乗っ取られた wait も乗っ取った wait も毎回 open の件を読んで返す(`internal/org/director_wait.go` 528-542 行、乗っ取りで止まるのは `director-wait.json` の書き込みだけ、667-670 行)。別プロセス 2 つの実測でも、どちらも件を返して終了コード 0 で終わった。症状(2 つとも issue と同じ文で時間切れ、件は 10 分後に ack の期限で人へ送られて `NOTIFIED: yes`)がすべて合うのは、wait が `--types QUESTION,BLOCKED,RESULT` 付きで、件が leader の stall の ALERT だった場合で、そのとき時間切れの文は `--types` なしのときと 1 文字も違わず、何も残らない。

## Scope

- S1 時間切れの文に、数えなかった種類の open の件を足す: `WaitInbox`(`internal/org/escalate.go` 324 行付近)と `WaitInboxForDirector`(`internal/org/director_wait.go` 541 行付近)が `--types` 付きで時間切れになるとき、すべての種類の open の件をもう 1 回読み、`--types` の外の件があれば、時間切れの文の後ろに `(counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e144 ALERT, e145 ALERT)` の形で足す(件が多いときは数を決めて切り、残りの数を書く)。`inboxWaitTimeoutPhrase`(`no open inbox item arrived within`)は先頭のまま変えない(skill とテストが照合している)。`--types` なしのとき、外の件がないときは今の文のまま。`director-wait.json` の `detail` にも同じことを書く(乗っ取られた wait は書かない、今のまま)
- S1 重なった wait のテスト: 2 つの director wait が重なり、後の wait が `director-wait.json` を取ったあとに見張りの escalate の形で件が入ると、乗っ取られた wait も持ち主の wait も件を返すことを固定する(issue の期待の 2 つ目「重なっていても新しい件は少なくとも 1 つの wait が返す」が今も成り立つことの証拠)。`--types` 付きの重なりで、時間切れの文に外の件が出ることも固定する
- S2 `/director` skill(4 写し)の「wait で待つ」: 新しい wait を張る前に、前の wait が終わった(Monitor の終わりの知らせが届いた、または `run_in_background` の終わりの知らせが届いた)ことを確かめる、と書く。wait の出力に「another ralph org wait --inbox --director … took over」の行が出たら、自分が wait を重ねているので、それ以上 wait を張らず、古いほうはそのまま自分の時間切れ(25 分)で終わるのに任せる、と書く(古いほうの Monitor を止めると、その終わりは「Monitor の期限切れ」以外の signal の終わりになり、skill は失敗として受信箱の待ちをやめる。重なっていても両方が件を返すことは S1 のテストで固める。codex の計画助言の指摘)。古いほうが件を返したら、件を処理し、新しいほうが動いているあいだは次の wait を張らない。時間切れの文に `open items of other TYPEs` が出たら、その件を読む(`--types` を付けたのは人の指示なので、ALERT を処理するかは指示に従う)と書く
- S2 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の 8b-2b の項に 2026-11-22 の項目、`ralph org wait --help` の `--types` の説明に時間切れの文の追記を 1 文

## Non-goals

- 後の wait が前の wait を終わらせる(issue の期待の 1 つ目)。原因に当たらず、新しい終了コードと skill の扱いが要る。重なりは skill の手順で防ぐ
- 見張りの配送の判断(`inboxDelivery`)や ack の期限の変更
- issue の台帳(別の repo)の実データの確かめ。ユーザーに `ralph org inbox show e144` の events と director の Monitor のコマンド行の `--types` を見てもらう(PR の本文と報告で頼む)

## Assumptions

- `openInboxItems`(`escalate.go` 341-353 行)は state が open かと `--types` だけを見て、`NOTIFIED` や作られた時刻は見ない
- 見張りは `director-wait.json` を読まない(読むのは状態の表示 `internal/cli/org_director_wait_line.go` だけ)
- 時間切れのあとにもう 1 回読む今の作りはそのまま(時間切れの直前に来た件も返る)

## Affected areas

- `internal/org/escalate.go`、`internal/org/director_wait.go` とテスト
- `internal/cli` の `ralph org wait` の help
- `/director` skill(4 写し)、仕様

## Visual review

None (時間切れの文に 1 つの句を足し、skill に手順を 2 つ足すだけで、部品の間のやり取りの順や、記録や状態の移り変わりは変えない)

## Design decisions

Critical forks: None

- issue の期待の 2 つ目は今も成り立つので、テストで固定するだけにする。1 つ目(乗っ取りで前の wait を終わらせる)は原因に当たらず、終了コードを増やすので採らない
- 外の件は時間切れの文に足し、終了コードは変えない(skill は終了コード 1 と時間切れの文で「もう一度打つ」を決めている。足した句があっても同じ扱いで、件を読む手順を足す)

## Acceptance criteria

- [ ] AC1: `--types QUESTION,BLOCKED,RESULT` 付きの wait(`WaitInbox` と `WaitInboxForDirector` の両方)が、ALERT の open の件だけがある受信箱で時間切れになると、時間切れの文が `no open inbox item arrived within <ms> ms` で始まり、そのあとに数えた種類と外の件の ID と TYPE が出る。`--types` なし、外の件がないときは今の文のまま。件が多いときは切って残りの数を書く(テストで固定)
- [ ] AC2: 2 つの director wait が重なり、後の wait が `director-wait.json` を取ったあとに件が入ると、どちらの wait も件を返す(fake の時計か短い間隔のテストで固定)
- [ ] AC3: `/director` skill の「wait で待つ」が、前の wait の終わりを確かめてから張ること、乗っ取りの行が出たら古いほうを止めずに自分の時間切れに任せ、動いている wait があるあいだは次を張らないこと、時間切れの文に外の件が出たときの扱いを言う。skill が古い Monitor を止めるとは言わないこと(止めた終わりが失敗の扱いに入るため)もテストで確かめる。4 写しがそろう(`./scripts/check-skill-sync.sh`)。テストで固定
- [ ] AC4: 仕様と `ralph org wait --help` が足した句を言う。`./scripts/check-sync.sh` が通る

## Implementation outline

1. S1 時間切れの文と重なりのテスト(AC1、AC2) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'WaitInbox|DirectorWait|Inbox' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgWait|Wait'`
2. S2 skill、仕様、help(AC3、AC4) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'DirectorSkill|Skill|DocsMatch' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgWait|Help|Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC4。時間切れの終了コードと `inboxWaitTimeoutPhrase` の位置が変わらないこと
- Documentation drift to check: `/director` skill の wait の終了コードの表と「wait で待つ」、仕様の 8b-2b、`ralph org wait --help`、`ralph status` と `ralph org director status` の wait の表示(`detail`)
- Evidence to capture: テストの出力、時間切れの文の例

## Test plan

- Unit tests: 時間切れの文を組む関数(外の件なし、1 件、上限を超える件数、`--types` なし)
- Integration tests: `WaitInbox` と `WaitInboxForDirector` の時間切れ、重なった 2 つの director wait、`ralph org wait` の CLI の終了コードと stderr
- Regression tests: 既存の director wait のテスト(乗っ取り、終了コード 4 と 5、Monitor の期限)、skill の時間切れの文の照合(`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase`)
- Edge cases: 時間切れの直前に外の件が resolve される、外の件の本文が長い(ID と TYPE だけを出す)、受信箱が読めない(今の失敗のまま)
- Evidence to capture: `go test` の出力、変異(外の件を足す処理を外す)で落ちること

## Risks and mitigations

- 本当の原因を確かめていない(issue の台帳は別の repo): 症状が合う道は `--types`、件が届いたときに読む wait がなかった、届いてすぐ ack か resolve された、の 3 つ。足す句は 1 つ目を見えるようにし、重なりのテストは 2 つの wait が件を返すことを示す。ユーザーに e144・e145 の events を見てもらい、違う道なら追いの issue にする
- 時間切れの文が長くなる: 件は ID と TYPE だけにし、数を決めて切る

## Rollout or rollback notes

- 新しい binary が要る(時間切れの文)。skill の文は binary なしでも効く
- revert すると今の文と skill に戻る
- PR の本文は `Refs #374` にする(原因は実データで確かめていないため)。ユーザーに `ralph org inbox show e144` の notified の reason(`inbox_ack_deadline` なら `--types` の道、すぐ人へなら見張りが受け付けていなかった道)と、director の Monitor のコマンド行の `--types` を見てもらい、合えば issue を閉じる

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 1 件、計画に入れた。[MEDIUM] 1: 重なりを解くために古い Monitor を止めると、その終わりは「Monitor の期限切れ」以外の signal の終わりで、skill(1112 行付近)は失敗として受信箱の待ちをやめ人の指示を待つ → 停止の指示を外し、古い wait は自分の時間切れに任せ、動いている wait があるあいだは次を張らない、とした(Scope S2、AC3)。git の状態は助言の前後で同じ。consult(consult-plan-373-374): #374 は「進めてよい」。原因は未確認なので PR は `Refs #374` にし、実データの確かめをユーザーに頼む(Rollout に書いた)
- [x] Plan approved — 系列の決まり(推奨案で自動承認、2026-10-11 のユーザーの指示)で承認
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
