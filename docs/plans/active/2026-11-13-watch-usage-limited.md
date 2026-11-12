# 利用上限で止まった座席を、見張りが stall ではなく usage_limited として org ごとに 1 件で知らせる(#312a)

- Status: Approved
- Approved: 2026-11-13 sha256:f2971e9c4a70
- Owner: Claude Code
- Date: 2026-11-13
- Related request: https://github.com/yoshpy-dev/ralph/issues/312(下流で、2 つの feature org の claude の座席 6 席がアカウントの週の上限で止まり、watch は stall の ALERT を上げ続けた。上限で止まったことは director が画面を読んで約 8 時間後に分かった)
- Related issue: 312
- Type: feat
- Branch: feat/watch-usage-limited

## Objective

issue #312 の受け入れ条件 1「利用上限で止まった座席が、watch で `usage_limited` として 1 件にまとまって知らされる」を満たす。横断の見張り(`ralph org watch --all`)は、stall になりそうな座席の画面を読み、`ralph org resume` が読む利用上限の文で止まっていれば、stall ではなく org の条件 `usage_limited` として、その org の 1 件にまとめて受信箱に上げる。件には止まった座席と、画面の上限の文(解除の時刻の `resets …` や `try again at …` を含む)をそのまま添える。受け入れ条件 2(reseat)は #312c(レーン D)、3 は #312b(PR #320、merge 済み)が受け持つ。

## Scope

- 利用上限の文の見分け: `ralph org resume` の画面の判定(`ClassifyResumeScreen`)が「止まった」と読む文のうち、利用上限の文(claude の `You've hit your session limit`・`weekly limit`・`Opus limit`・`Sonnet limit`、codex の `You've hit your usage limit`・`Usage limit reached`)で止まった画面を見分ける。通信断の文(`API Error:` など)は含めない
- 見張り: 座席ごとの stall の判定のうち、leader の待ちの除外(`leaderIsWaiting`: ほかの座席が動いている、leader の上げた件が未解決)を当てる前の判定(herdr の agent get と台帳が変わらない)が立つ座席にだけ、その座席の pane を 1 回読む(1 回の呼び出しは driver の呼び出しの上限で区切る)。上限の文で止まっていれば、その座席の stall は立てず、org の `usage_limited` の対象にする。上限でなければ、leader の待ちの除外を当てて今までどおり stall を決める(plan advisory の 2: 除外を先に当てると、上限で止まった leader の画面が読まれない)。1 周期の中で org の上限の座席を集め、1 つ以上あれば org の条件 `usage_limited` を立てる
- 件: `usage_limited` の ALERT は、leader の deadman を待たずに、その周期のうちに受信箱の ALERT の件にする(届け方は今の見張りの ALERT の件と同じ規則: director か人)。理由(REASON)は `usage_limit`。本文に、止まった座席ごとに driver と画面の上限の文を書く
- 1 件にまとめる: 見張りの状態ファイル(org ごとの watch-status)に、org の今の上限の座席の集合を持つ。条件が立っているあいだに集合に新しい座席が入ったら、それを条件の立ち上がりとして扱う(条件の記録をいったん下ろしてから立てる)。そのとき、org の `usage_limited` の件が open(director も人もまだ受け付けていない)なら、件に recurred の出来事を 1 つ足し、note に増えた座席を書く(開いたときに読まれる)。件が acked か resolve 済みなら、新しい件にする(受け付けた人に増えた座席が届かないまま resolve されるのを避ける。plan advisory の 1)。recurred を足すあいだに件が resolve された(`ErrInboxResolved`)ときも新しい件にする。org に上限の座席がなくなったら条件は下り、集合も空にする。そのあと再び立ったときは #307a と同じ規則(件が resolve されていなければ recurrence、されていれば新しい件)
- 検知の時機: 画面を読むのは stall の判定のあとなので、上限で止まってから `[org.watchdog].stall_minutes` ほどで知らせる(今の stall と同じ遅れ)
- 画面が読めない、上限の文でない(通信断、判断できない画面)座席は、今までどおり stall
- 文書: director の skill(4 つの写し)の ALERT の件の扱いに `usage_limited` を足す(リセットの時刻まで待つか、別のアカウントで立て直す。リセット後は「止まった座席を続ける」)。org の skill(4 つの写し)の見張りの条件の一覧、`.claude/rules/ralph/agent-messaging.md` の見張りの ALERT の件の段落、org runtime の仕様の改訂の項、AGENTS.md の internal/org の行

## Non-goals

- 解除の時刻を読んで時刻として扱うこと(画面の文をそのまま添えるだけ)
- 上限で止まった座席を自動で続けること(`ralph org resume` は人が頼んだときだけ)
- 座席を別のアカウントや driver で立て直す動詞(#312c)
- 台帳全体(すべての org)を 1 件にまとめること(件は org のもの。org ごとに 1 件)
- headless の director の画面を見張ること(座席だけ)

## Assumptions

- 利用上限は座席の画面に、`ralph org resume` の文の表(`resumeScreenSentences` の Stopped)にある文で出る(実機: claude 2.1.312 の weekly limit、docs/evidence/org-seat-resume-live-2026-11-07.md。session limit と codex の文は文書と binary の文字列から)
- 上限で止まった座席は手番を終えていて、herdr は idle か done と言い、manifest にも何も起きないので、今の stall の判定が立つ(issue #312 で stall が上がり続けた)
- 同じアカウントの座席はほぼ同時に止まり、leader も止まっていることが多い。leader に ALERT を送っても動けないので、件はすぐに受信箱に上げる
- ALERT の件の見出しの `SEAT` は座席 id の形でなければならない(`checkWatchEscalationValues`)。org の条件の件は leader の座席 id を `SEAT` に使い、本文に止まった座席を並べる(D2)

## Affected areas

- `internal/org/resume_screens.go` か新しいファイル: 上限の文の見分け
- 新しい `internal/org/watch_usage_limit.go`: 見張りの `usage_limited` の条件
- `internal/org/watch.go`: `evaluateSeat` の stall の前の画面の確かめ、`evaluateCycle` で org の条件を立てる、`checkDeadman` ですぐに上げる、`escalateToInbox` の本文
- `internal/org/watch_recurrence.go`: 座席が増えたときの立ち上がりと recurred の note
- 見張りの状態ファイルの形(watch-status の org ごとの上限の座席の集合、pending の ALERT の座席と画面の行)。古い状態ファイルは空として読む
- 文書: `.claude/skills/director/SKILL.md` と 3 つの写し、`.claude/skills/org/SKILL.md` と 3 つの写し、`.claude/rules/ralph/agent-messaging.md`(と templates/base の写しがあればそれも)、`docs/specs/2026-08-01-org-runtime.md`、AGENTS.md

## Visual review

- Page: `.harness/state/plan-visual/watch-usage-limited.html`
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: 全体、overview、detail-overview、detail-cycle を撮って読んだ。重なりとはみ出しはない。plan advisory の直しのあと、detail-cycle(画面を読む座席を leader の待ちの除外より前に、増えた座席の扱い)を描き直し、lint と撮影をやり直して問題はない

## Design decisions

Critical forks: None。次の点は issue と #307 の仕組みから決められるので、自律で選んだ(2026-11-12 のユーザーの指示「推奨案を自動承認します」)。consult(判定「直してから進める」)の指摘で、座席が増えたときの立ち上がりの作り(Scope)、理由と説明を条件で決めること(D3)、画面の読み方(D4)、AC4 と AC10 を足した。

- D1 まとめ方は org ごとに 1 件。issue の「1 件にまとまって」を、件が org のものである受信箱の作りに合わせて読んだ。台帳全体で 1 件にすると、件の org が決まらず、director が org ごとに扱う今の手順に合わない
- D2 org の条件の鍵は (org、leader、`usage_limited`)。どの org にも leader がいるので鍵が変わらず、#307a の recurrence と dedupe がそのまま効く。件の `SEAT` は `leader` になるので、本文の先頭に「org の条件で、止まった座席は次のとおり」と書き、`SEATS:` の行に座席を並べる。director の skill には、`CONDITION: usage_limited` のときは `SEAT` ではなく `SEATS:` の座席の画面を読む、と書く(今の skill は `SEAT` の座席の画面を読むよう書いている)
- D3 すぐに上げる。ALERT の Subject を leader にするので、`checkDeadman` の今の分岐(leader 自身の ALERT は leader の動きを待たずに上げる)に乗る。ただし件の理由(REASON)と本文の説明は Subject ではなく条件で決める: `usage_limited` なら理由は `usage_limit`、説明は「org の座席が利用上限で止まった」(今の `escalateAlert` の理由と `watchEscalationBody` の説明は Subject で決まり、そのままだと「leader 自身の ALERT」になる)。止まった座席と画面の行は pending の ALERT に持たせ、escalation のときに本文を作る。leader への ALERT(agmsg)も今までどおり送る(leader が動けるなら読める)
- D4 画面を読むのは、leader の待ちの除外を当てる前の stall の判定が立つ座席だけ。健康な座席の画面は読まない。座席が上限で止まっているあいだは周期ごとに読む(stall の判定が毎周期立つので、上限か stall かを毎回決める)。画面は `w.org.Herdr.PaneRead` で読む(`HerdrClient` が `PaneRead` を持つので、型アサートはしない。Herdr が無ければ読まず、stall のまま。当初は見張り側の interface への型アサートと書いていたが、要らないので外した)。座席の pane は台帳の `PaneID`。読もうとして読めなかった(失敗、期限切れ、pane id が無い)ときは、その座席の stall の ALERT(leader への agmsg)に、上限かどうかを確かめられなかった理由を 1 行足す。inbox に載せられない道では、stderr の banner に止まった座席を出す
- D5 上限の文は、resume の表の Stopped の文のうち上限の文として印を付けたもので見分ける。表と別の一覧にすると食い違うので、表の側に印を持たせるか、一覧の文がすべて表にあることをテストで固める
- D6 解除の時刻は読まない。画面の上限の文の塊をそのまま本文に入れる(本文の上限 2000 字に収まるよう、座席ごとに切る)

## Acceptance criteria

- [ ] AC1 stall が立つはずの座席の画面が上限の文で止まっていれば、その座席の stall は立たず、org の `usage_limited` が立つ。件の本文に、座席、driver、画面の上限の文(`resets …` などを含む)が出る
- [ ] AC2 同じ周期に同じ org の座席が複数上限で止まっていれば、件は 1 件で、本文に全員が並ぶ
- [ ] AC3 `usage_limited` の ALERT は、deadman を待たずに同じ周期のうちに受信箱の件になり、理由は `usage_limit`。届け方は今の見張りの ALERT の件と同じ(director が有効で見張りが仕事を受けていれば director、そうでなければ人)
- [ ] AC4 条件が立っているあいだに上限の座席が増えたら、件が open なら新しい件はできず、件に recurred の出来事が 1 つ足され、note に増えた座席が出る。件が acked か resolve 済みなら新しい件になり、増えた座席が届く。recurred を足すあいだに件が resolve されたときも新しい件になる
- [ ] AC10 件の理由は `usage_limit` で、本文の説明は org の座席が利用上限で止まったことを言う(leader 自身の ALERT の文にならない)
- [ ] AC5 org に上限の座席がなくなると条件は下りる。そのあと再び立ったとき、件が resolve されていなければ recurrence、されていれば新しい件
- [ ] AC6 画面が読めない座席、通信断の文や判断できない画面の座席は、今までどおり stall
- [ ] AC7 leader の待ちの除外を当てる前の stall の判定が立たない座席の画面は読まない。画面の読みは driver の呼び出しの上限で区切られる
- [ ] AC11 上限で止まった leader は、ほかの座席が動いていても、leader の上げた件が未解決でも、`usage_limited` に入る。上限でない leader の stall は今までどおり除外される
- [ ] AC8 上限の文の見分けが、既存の fixture(claude-session-limit、claude-weekly-limit-under-prompt、codex-usage-limit は上限、claude-enotfound などの通信断は上限でない)で期待どおり
- [ ] AC9 director と org の skill、agent-messaging.md、仕様、AGENTS.md がこの挙動に合う

## Implementation outline

1. 上限の文の見分け(D5)と、fixture すべてに対する表のテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'UsageLimit|Resume|Screen'`
2. 見張りの `usage_limited`(D1〜D4、D6): 画面の確かめ、org の条件、すぐの escalation、座席が増えたときの recurred、件の本文 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Watch|UsageLimit|Deadman|Escalat|Recur|Inbox'`
3. 文書(director と org の skill の 4 つの写し、agent-messaging.md、仕様、AGENTS.md) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/ -run 'Skill|Messaging|Director'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC11。issue #312 の受け入れ条件 1
- Documentation drift to check: director と org の skill の 4 つの写し、agent-messaging.md、`docs/specs/2026-08-01-org-runtime.md`、AGENTS.md、`ralph org watch --help`(条件の一覧があれば)
- Evidence to capture: テストの出力

## Test plan

- Unit tests:
  - 上限の文の見分け: fixture すべて(上限の 3 つ、通信断、入力待ち、動作中、選択の画面)
  - 見張り: 偽の herdr(agent get が変わらない)と偽の pane(上限の画面)で、stall の代わりに `usage_limited` が立ち、同じ周期で受信箱の件になる(理由 `usage_limit`、本文に座席と上限の文)。2 座席が上限 → 1 件。次の周期で 3 座席目が上限 → 新しい件はなく recurred が 1 つ、note に 3 座席目。件を ack したあと、resolve したあとに 3 座席目が上限 → どちらも新しい件。recurred を足すあいだに resolve された → 新しい件。件の理由が `usage_limit` で、本文が leader 自身の ALERT の文でないこと。上限で止まった leader と動いている座席が同じ org にいる、leader の上げた件が open か acked → leader は `usage_limited` に入る。上限でない leader は今までどおり除外。上限が解けた(画面が入力待ちになり manifest が動いた)→ 条件が下りる。再び上限 → 件が open なら recurrence、resolve 済みなら新しい件。画面が読めない、通信断の画面 → stall。健康な座席 → pane を読まない
  - 本文の上限: 座席が多い、上限の文が長いときも 2000 字に収まる
- Integration tests: `RunWatchAll` の 1 周期を偽の herdr で通す(既存の見張りのテストの形)
- Regression tests: 既存の見張り(stall、liveness、scope_change、recurrence、deadman、alert_notify)のテストがそのまま通ること
- Edge cases: leader だけが上限、leader 以外だけが上限、claude と codex が混ざる org、件の本文に入れる画面の行に制御文字
- Evidence to capture: `go test` の結果

## Risks and mitigations

- 上限でない画面を上限と読み、stall を隠す。対策: 上限の文は resume の表の上限の文だけ。画面が読めない・判断できないときは stall のまま
- pane を読む回数が増え、見張りの周期が遅くなる。対策: stall が立つ座席だけを読み、1 回の読みを driver の呼び出しの上限で区切る
- 件の `SEAT: leader` を、leader 自身の問題と読み違える。対策: 本文の先頭と `SEATS:` の行で org の条件だと書き、director の skill にも書く
- director の skill と org の skill は、レーン D(#312c)も触る。対策: merge の前に origin/main を取り込み、ほかのセッションに知らせる

## Rollout or rollback notes

- 新しい binary の `ralph org watch --all` から効く。見張りを止めてから binary を入れ替える(#307 と同じ)
- skill の文面は `ralph upgrade` で新しくなる
- 戻すときは revert する

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 2 件をどちらも計画に入れた(常設の指示により聞かずに直した)。[HIGH] 増えた座席を recurred にするだけでは、acked の件では誰にも届かずに resolve されうる → 件が open なら recurred、acked か resolve 済み、または recurred のあいだに resolve されたら新しい件(Scope、AC4)。[HIGH] leader の待ちの除外を先に当てると、上限で止まった leader の画面が読まれない → 除外の前の stall の判定で画面を読み、除外は上限でないときの stall にだけ当てる(Scope、D4、AC7、AC11)。それより前に consult(判定「直してから進める」)も入れた
- [x] Plan approved — 2026-11-13、自動承認(ユーザーの指示「私の承認は不要です。あなたたちの推奨案を自動承認します」)
- [x] S1 上限の文の見分け — 8720ed80(implementer。`UsageLimitScreen`、表に `UsageLimit` の印。verify の命令が通った。自己点検: コメント 2 か所を fixed、ほか clean)
- [x] S2 見張りの `usage_limited` — 4d695e5b(implementer。verify の命令とフィルタなしの org のテストが通った。変異 3 つすべて死亡。自己点検: doc の一文を fixed、ほか clean。pending の座席は比べられる形を保つため構造体へのポインタにした)。origin/main(#324、4220ea7c)を 78611dbc で取り込み、build と vet が通った
- [x] S3 文書 — 439b393e(implementer。check-skill-sync、check-sync、対象のテストが通った。自己点検: help の 1 文を fixed、ほか clean。範囲外の所見: quality-gates.md の見張りの行と multi-org の仕様の理由の一覧 → sync-docs で直す)
- [x] self-review(LOW 7、Merge: yes)— 8010c4ca。L1・L2・L4・L6・L7 を直し、L3 はコメントだけ(既存のテストが pending を == で比べるのでポインタは要る)、L5(読めない周期のあとに座席が「増えた」と数えられる)は計画どおりなので Known gaps に書く
- [x] S4 self-review の直し — e25def99(implementer。verify の命令が通った。変異 4 つで固めた。自己点検すべて clean)。D4 を実装に合わせて直し(型アサートをやめた、読めなかった理由を stall の ALERT に、inbox に載せられない道の banner に座席)、承認をやり直した。図は変えていない
- [x] self-review の再確認 — b7faa2bd(LOW 3: L5、N1、N2。Merge: yes。L3 は reviewer が取り下げた)。N2(計画の D4 が報告の指摘 ID を指す)は D4 の文から ID を外して直し、承認をやり直した。N1(読めなかった理由は leader への ALERT にだけ出て、受信箱の件の本文には届かない)と L5 は Known gaps に書く
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
