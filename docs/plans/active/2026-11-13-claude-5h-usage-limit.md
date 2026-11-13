# claude の 5 時間の上限と、上限の解除を待つ画面を、見張りが usage_limited と見分ける(#329)

- Status: Approved
- Approved: 2026-11-13 sha256:be05f6884990
- Owner: Claude Code
- Date: 2026-11-13
- Related request: https://github.com/yoshpy-dev/ralph/issues/329(claude 2.1.316 の座席が 5 時間の上限で止まった画面を、見張りが usage_limited と見分けず stall にする。2026-11-13 にレーン B が #322 の確かめで実機で当たった)
- Related issue: 329
- Type: fix
- Branch: fix/claude-5h-usage-limit

## Objective

claude の座席がアカウントの利用上限に当たり、上限の解除を待っている画面を、横断の見張り(`ralph org watch --all`)が stall ではなく org の条件 `usage_limited` として知らせる。#312a(PR #327)の見分けは、`ralph org resume` の画面の判定が「止まった」(stopped)と読む画面だけを見ていた。claude 2.1.316 は 5 時間の上限で、解除の時刻にひとりでに続きを始める画面(`Continuing automatically at 3:50pm`)を出し、resume の判定はそれを「待ち中」(waiting)と読むので、見張りはその座席を stall のままにしていた。

## Scope

- claude の文の表(`internal/org/resume_screens.go` の `resumeScreenSentences["claude"]`)に、実機で見た `You've hit your 5-hour limit` を、止まった文(`Stopped`)と上限の文(`UsageLimit`)として足す。出典は `seen: Claude Code 2.1.316` と、この branch で足す evidence
- 待ち中の文のうち、上限の解除を待つ文に印を付ける。claude の `Continuing automatically at` と `Continuing shortly`(どちらも docs の wait-for-a-usage-limit-to-reset が出典)。新しい欄 `UsageLimitWaiting` に持ち、どれも `Waiting` の文であることをテストで固める(`UsageLimit` と `Stopped` の関係と同じ形)
- `UsageLimitScreen`(`internal/org/usage_limit_screen.go`)は、resume の判定が waiting で、`UsageLimitWaiting` の文が入力欄の下の欄(footer。CLI が描く欄で、返事やツールの出力は出ない)にあるときも、上限の画面と答える。入力欄と footer は `ClassifyResumeScreen` と同じ道(`findResumeInputBox`)で見つける。待ちの文が入力欄より上のブロック(返事、ツールの出力、引用)にあるだけの画面は、上限の画面と答えない(D2)。返す行はその footer のブロックを 1 行にしたもの。stopped の画面の扱いは変えない
- 実機の画面を fixture(`internal/org/testdata/resume/claude-5h-limit-continuing.txt`)と evidence(`docs/evidence/claude-5h-limit-screen-2026-11-13.md`)に入れる
- 引用の画面を fixture かテストの中の画面として足す: 返事の最後の段落が `Usage limit reached · limit resets 3:50pm` と `Continuing automatically at 3:50pm · Esc to cancel` を引用した画面、ツールの出力が同じ文を出した画面。どちらも resume は今までどおり waiting と読み、見張りは上限の画面と答えない
- 既存の fixture `claude-continuing-automatically.txt`(作った画面で、待ちの文は入力欄の上のブロック)は、今までどおり上限の画面ではない。テストの期待は変えない
- `usage_limit_screen_test.go` の、上限の画面は必ず stopped と断定しているところ(今の 176〜178 行)と、文の一覧の期待(316 行あたり)を、stopped か footer の待ちのどちらか、に直す
- 文書: org runtime の仕様の改訂の項(#312a の項の文の一覧のあとに、2026-11-13 の #329 の項)、resume の仕様(`docs/specs/2026-11-07-org-seat-resume.md`)の止まった文の一覧、`.claude/rules/ralph/agent-messaging.md` と雛形の写しの `usage_limited` の段落、`docs/tech-debt/README.md` の 452 行(上限の画面は実機で見ていない)

## Non-goals

- `ralph org resume` の振る舞いを変えること。上限の解除を待つ画面は今までどおり waiting で、resume は何も打たない(CLI がひとりでに続きを始める)
- director の skill の `usage_limited` の扱いを変えること。件の行に `Continuing automatically at …` が出るので、解除のあとに座席がひとりでに続くことは行から読める。director の skill はレーン A の #318 とレーン B の #322 も直している
- 解除の時刻を時刻として読むこと(#312a と同じく、画面の文をそのまま添える)
- codex の表を変えること(codex の待ち中の文はどれも通信断の待ちで、上限の解除の待ちではない)
- 上限の文を `You've hit your … limit` の形でまとめて見分けること(D3)

## Assumptions

- 実機の画面(claude 2.1.316、herdr 0.7.5、pane の幅 53 桁)は、レーン B が `herdr pane read --source recent --lines 60 --format text` で Enter の 1.5 秒後に読んだもので、手番の終わりの読みとバイト単位で同じだった(写しはこのセッションの scratchpad の `probe-claude-5h/cs-submit.enter15.txt`)
- 上限の解除を待つあいだも、見張りの stall の判定が立つ。stall の判定は herdr の状態の語ではなく、`herdr agent get` の出力の文字列と台帳が `stall_minutes` のあいだ変わらないことで決まる(`internal/org/watch.go` の stall の判定)。待ちのあいだに agent get の出力が変わり続けるかは未確認(画面には手番の終わりの行 `✻ Brewed for 0s · done 3:36 PM` がある)
- `Continuing automatically at`(大文字の C)は、待ちのあいだの画面にだけ出る。待ちが終わると、会話の記録には小文字の `continuing automatically at` が残る(`resume_screens.go` の表のコメント)。そのため解除のあとは waiting と読まれず、条件は下りる

## Affected areas

- `internal/org/resume_screens.go`(claude の表、`resumeSentenceTable` の新しい欄)
- `internal/org/usage_limit_screen.go`(`UsageLimitScreen` と、待ち中の文を見つける小さな関数)
- `internal/org/testdata/resume/claude-5h-limit-continuing.txt`(新規)
- `internal/org/usage_limit_screen_test.go`、`internal/org/resume_screens_test.go`、`internal/org/send_screen_test.go`(fixture の表に 1 行。`TestClassifySendScreen_EveryFixture` がすべての fixture を求める)
- `docs/evidence/claude-5h-limit-screen-2026-11-13.md`(新規)
- `docs/specs/2026-08-01-org-runtime.md`、`docs/specs/2026-11-07-org-seat-resume.md`、`.claude/rules/ralph/agent-messaging.md`、`templates/base/.claude/rules/ralph/agent-messaging.md`、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/claude-5h-usage-limit.html`(かんたん版の全体図 1 枚、承認の前の確かめ、たたみの中に詳細図 3 枚: ファイルの地図、上限の画面かどうかの決め方、実機の画面のどの行を読むか)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- 自己点検: ページ全体と 4 枚の図を `shot` で撮って読んだ。崩れなし(詳細図 3 の画面の行頭の空白は SVG で詰まるが、読むのに差し支えない)

## Design decisions

Critical forks: None。次の点は、コードと docs と実機の画面から決められるので、自律で選んだ(ユーザーの「推奨案を自動承認します」、2026-11-13)。

- D1 上限の解除を待つ画面も `usage_limited` にする。代わりの案(`You've hit your 5-hour limit` を足すだけ)では、実機の画面は直らない。resume の判定は入力欄のすぐ上のブロック(`⏺ Usage limit reached · continuing automatically at …`)を候補にし、その上の `⎿ You've hit your 5-hour limit` の行は読まない。しかも入力欄の下の `Continuing automatically at` が先に waiting を決める。待っている座席は解除までは動かず、人が知りたいのは上限で止まったことなので、stall より `usage_limited` が合う
- D2 上限の解除の待ちは、待ち中の文の印(`UsageLimitWaiting`)と、その文のある場所で見分ける。`Retrying in` と `Waiting for API response` は API の再試行の待ちなので印を付けず、今までどおり stall にする。印の付いた文が `Waiting` にあることはテストで固める。場所は入力欄の下の footer だけにする(codex の計画助言 MEDIUM)。resume の waiting の判定は待ちの文をどのブロックでも数えるので、返事やツールの出力が文を引用すると waiting と読む。resume ではそれで打たないだけだが、見張りでは stall の知らせが止まり、件を resolve しても画面が変わらないあいだは stall に戻らない。footer は CLI が描く欄で返事は出ないので、引用で当たらない。実機の画面(2.1.316)では待ちの文は footer にあった。作った fixture のように入力欄の上のブロックにだけ待ちの文がある画面は、上限の画面と答えない(stall のまま。黙る向きにはならない)
- D3 上限の文は 1 つずつ持つ。`You've hit your … limit` の形でまとめると、版が変わって語が増えても当たるが、引用やほかの文と取り違える幅が広がる。#312a と同じく表の 1 行ずつにして、出典を書く
- D4 待ちの道で `UsageLimitScreen` が返す行は、待ちの文のある footer のブロックを `usageLimitLine` で 1 行にしたもの。実機の画面では `🤖 Haiku 5.5 | ⚡ low` などの状態の行も入る。状態の行は切らずにそのまま使う(解除の時刻 `limit resets 3:50pm` と `Continuing automatically at 3:50pm` が入り、300 文字で切られる。claude の footer の行の形を見分ける規則を足すより単純)
- D5 `You've hit your 5-hour limit` も `Stopped` と `UsageLimit` に足す。この文が候補のブロックの頭に来る画面(2.1.312 の weekly limit のように、待ちの行が出ない画面)では、今の stopped の道で上限と読まれる。resume はその画面で続行の文を打つ(ほかの上限の文と同じ)

## Acceptance criteria

- [ ] AC1 実機の画面(`claude-5h-limit-continuing.txt`)を、`UsageLimitScreen("claude", …)` が上限の画面と答え、行に `limit resets 3:50pm` と `Continuing automatically at 3:50pm` が入る
- [ ] AC2 `ClassifyResumeScreen` は、実機の画面も `claude-continuing-automatically.txt` も今までどおり waiting と読む(resume は打たない)
- [ ] AC3 待ちの文が入力欄より上のブロックにだけある画面は上限の画面ではない: `claude-continuing-automatically.txt`、返事の最後の段落が待ちの文を引用した画面、ツールの出力が待ちの文を出した画面。`Retrying in` と `Waiting for API response` で waiting と読む画面も上限の画面ではない
- [ ] AC4 `⎿  You've hit your 5-hour limit · resets …` が候補のブロックの頭にある画面は stopped で、`UsageLimitScreen` も上限の画面と答える
- [ ] AC5 `UsageLimitWaiting` のどの文も、同じ driver の `Waiting` の文である(テスト)。`UsageLimit` と `Stopped` の関係のテストは変えずに通る
- [ ] AC6 見張りが実機の画面の座席を stall ではなく `usage_limited` の対象にする。返事が待ちの文を引用した画面の座席は、続けて 2 周期以上でも stall のまま(`watch_usage_limit_test.go` の既存の形)
- [ ] AC7 `send_screen_test.go` の fixture の表に新しい fixture の行があり、`TestClassifySendScreen_EveryFixture` が通る
- [ ] AC8 仕様 2 つ、`agent-messaging.md` の 2 つの写し、tech-debt の 452 行がこの振る舞いに合い、`./scripts/check-sync.sh` が通る

## Implementation outline

1. S1 見分けと fixture とテスト: `resume_screens.go` の claude の表(5 時間の上限の文、`UsageLimitWaiting`)、`usage_limit_screen.go`(waiting の画面の道、doc comment)、新しい fixture、`usage_limit_screen_test.go`・`resume_screens_test.go`・`send_screen_test.go`・`watch_usage_limit_test.go` の表と期待、evidence の文書 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'UsageLimit|ClassifyResumeScreen|ResumeScreenSentences|ClassifySendScreen|SendInputHolds|WatchUsageLimit'`
2. S2 文書: 仕様 2 つ、`agent-messaging.md` の 2 つの写し、tech-debt の 452 行 — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、secret scan)
- Spec compliance criteria to confirm: AC1〜AC8。issue #329 の「直す方向の候補」のうち、D1〜D5 で採ったものと採らなかったもの
- Documentation drift to check: 仕様 2 つと `agent-messaging.md` の上限の文の一覧、`UsageLimitScreen` の doc comment、`resume_screens.go` の表のコメント、tech-debt の 452 行と #312a の行(487 など)
- Evidence to capture: verify の報告、実機の画面の evidence の文書

## Test plan

- Unit tests: `UsageLimitScreen` の fixture の表と、文の表を直に書いたケース(5 時間の上限の stopped、上限の解除の待ち、API の再試行の待ち)。`ClassifyResumeScreen` の fixture の表に新しい fixture。`UsageLimitWaiting` が `Waiting` の文であるテスト
- Integration tests: 見張りの 1 周期で、実機の画面の座席が `usage_limited` に入り stall にならない(`watch_usage_limit_test.go` の既存の形)
- Regression tests: `TestClassifyResumeScreen_Fixtures`、`TestUsageLimitScreen_EveryFixture`、`TestClassifySendScreen_EveryFixture`、`TestUsageLimitSentences_AreStoppedSentences` が通る。resume の判定はどの fixture でも変わらない
- Edge cases: 待ちが終わったあとの画面(小文字の `continuing automatically at` だけが記録に残る)は上限の画面ではない。`Retrying in` の待ちは上限の画面ではない。返事の最後の段落やツールの出力が待ちの文を引用した画面は上限の画面ではない。プロンプトが待ちの文を引用した画面は、今の判定どおり(プロンプトのブロックは読まない)
- Evidence to capture: test の報告、変異の確かめ(`UsageLimitWaiting` の照合を外すと AC1 のテストが落ちる、footer の限定を外すと AC3 の引用のテストと AC6 の 2 周期のテストが落ちる、`Retrying in` に印を付けると AC3 の否定のテストが落ちる)

## Risks and mitigations

- 上限でない待ちを上限と読み、stall を隠す。対策: 印は docs が上限の解除の待ちとして書く 2 つの文だけ。API の再試行の待ちは印を付けず、テストで固める
- 返事やツールの出力が待ちの文を引用すると、見張りが stall を止める。対策: 待ちの道は footer の文だけを見る(D2)。引用の画面のテストと、見張りの 2 周期のテストで固める
- 版が変わり、claude が待ちの文を入力欄の上にだけ出すようになると、見張りは上限と見分けず stall にする。対策: 黙る向きにはならない(今と同じ stall)。fixture と evidence があるので、実機の画面が変わったら表と規則を直す。Known gaps に書く
- レーン B の #322(branch fix/send-input-box-marker)が、`send_screen_test.go` の fixture の表と `internal/org/testdata/resume/` に行と fixture を足す。対策: 行単位の衝突なので、後から merge する側が両方の行を残す。計画の承認と merge の前後に B に知らせる
- 実機の画面は 1 つだけ(Haiku、幅 53 桁)。`Continuing shortly` の画面と、ほかの幅や版の画面は見ていない。対策: Known gaps に書き、tech-debt の 452 行を直す

## Rollout or rollback notes

- 新しい binary の `ralph org watch --all` から効く。見張りを止めてから binary を入れ替える(#312a と同じ)
- 戻すときは revert する

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。[MEDIUM] 1 件: 返事やツールの出力が待ちの文を引用すると、resume の waiting の判定はどのブロックでも数えるので、見張りが stall を止め、件を resolve しても画面が変わらないあいだ戻らない → 待ちの道は入力欄の下の footer の文だけを見る(Scope、D2、D4、AC3、AC6、Risks、Test plan)。作った fixture `claude-continuing-automatically.txt` は上限の画面にしない、に変えた。聞かずに入れた(ユーザーの常の指示)。並べて consult(判定「進めてよい」)も入れた: stall の判定は herdr の状態の語ではなく agent get の出力の文字列で決まる(Assumptions を直した)、`usage_limit_screen_test.go` の stopped の断定と文の一覧の期待を直すことを Scope に書いた、D4 の行は状態の行を切らないと決めた
- [x] Plan approved — 自動承認(ユーザーの「推奨案を自動承認します」、2026-11-13)
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
