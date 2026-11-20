# watch-usage-limit-retry

- Status: Approved
- Approved: 2026-11-20 sha256:70b063e49ad8
- Owner: Claude Code
- Date: 2026-11-20
- Related request: 下流の issue #353。機能の org の leader の座席(claude)が週の利用上限に当たり、Claude Code は手番を開いたまま再試行して上限が解けるのを待った(座席の環境に `CLAUDE_CODE_RETRY_WATCHDOG=1`)。横断の見張りはこれを `CONDITION: stall`・`REASON: leader_is_anomaly_subject` の件にし、`usage_limited` にしなかった。`ralph org resume --dry-run` も `waiting <org>/leader claude herdr reports working` と出し、上限とは読まなかった。ユーザーは担当とタイミングをレーン A に任せた(2026-11-20、系列の指示で計画は推す案で自動承認)
- Related issue: #353
- Type: fix
- Branch: fix/watch-usage-limit-retry

## Objective

claude の座席が利用上限に当たり、手番の中で再試行して解除を待っている画面を、見張りと `ralph org resume` が「上限の解除を待つ座席」と読めるようにする。見張りは stall ではなく `usage_limited` の件を上げ、`SEATS:` に座席を、本文に画面の上限の文とリセットの時刻を書く。`ralph org resume`(`--dry-run` も)は、その座席を上限の解除を待つ座席として出す。

## Scope

- `internal/org/usage_limit_screen.go`: `UsageLimitScreen` に 3 つ目の読み方を足す(claude だけ)。次の 3 つがそろったら上限と読む
  1. `ClassifyResumeScreen` が waiting と読む
  2. 今の手番のブロック(入力欄の上にある最後の prompt のブロックより後、prompt が画面にないときは入力欄の上のすべてのブロック)のどれかが、claude の StopGlyph(`⏺` か `⎿`)のあとに空白(U+00A0 も、`findResumeStop` と同じく `unicode.IsSpace`)、そのあとに上限の文(`UsageLimit` の表)を、ブロックの中のどこかに持ち、その文のあとに `resets <時刻>` がある。ブロックの行は 1 つの空白でつないでから探すので、折り返した文も見つかる。`<時刻>` は `resets ` のあとから、` (`(タイムゾーンの括弧)かブロックの終わりまで(issue の画面では `Nov 21 at 3pm`)
  3. 入力欄のすぐ上のブロックが claude の再試行の状態行で(`Retrying in` と `attempt` を両方含む。pane が狭いと折り返して語が入れ替わるので、語の並びでは照らさない)、2 の `<時刻>` を括弧に入れた `(<時刻>)` を含む(issue の画面の `Retrying in 29m (Nov 21 at 3pm)`)。この一致を、今の再試行が上限によるものだという証拠にする(Codex の計画助言の指摘 1)。通信断の再試行の行(`Retrying in 4s · attempt 13/300`)はリセットの時刻を持たないので当たらない
  2 と 3 の手番の中に上限の文が 2 つ以上あれば、3 の時刻と一致する最後のものを使う。返す行は、その glyph からブロックの終わりまで(上限の文とリセットの時刻。`usageLimitLine` で 1 行にし 300 字で切る)
- `internal/org/resume.go`: `resumeLook` が座席を waiting と出すとき(herdr が working と報告するときと、画面が waiting と読めたとき)、画面を読み(working のときは 1 回 `herdr pane read` を足す)、`UsageLimitScreen` が上限と読めば、理由を `waiting for the usage limit to reset: <行>` にする。読めなかったとき、上限と読めないときは今の理由のまま。何も打たない(outcome は waiting のまま)
- `internal/org/testdata/resume/` に、issue の画面を元にした fixture を足し、resume の判定の表と上限の判定の表の両方に期待を書く。テストは `internal/org/usage_limit_screen_test.go`、`internal/org/resume_test.go`(または resume の CLI のテスト)、見張りのテスト(`usage_limited` の件になること)
- 文書: `.claude/rules/ralph/agent-messaging.md`(templates の写しも)の `usage_limited` の段落、director skill の 4 写しの「解除を待っている座席」の文、`docs/tech-debt/README.md` の 514 行(#329 の既知の穴)に、この形は読めるようになったと追記

## Non-goals

- codex の座席の読み方
- 入力欄の上のブロックにだけ `Continuing automatically at` を出す claude の版(tech-debt の 514 行の残り)
- 上限の文がないまま再試行が長く続く画面を上限と推し量ること(エラーの要約の `Weekly li…` は pane の幅で切れるので手がかりにしない。括弧の時刻は、上限の文のリセットの時刻と一致するときだけ使う)
- 上限と確かめられなかった画面の stall の件に、上限の文を「確かめていない手がかり」として載せること(Codex の計画助言が挙げた案。stall の件の形を変えるので別の変更にする)
- 見張りの stall の条件そのもの(herdr の出力が変わり続けるときに stall にしない、など)
- 座席を別のアカウントに移す手順(#354)

## Assumptions

- issue の画面(Claude Code 2.1.316、`CLAUDE_CODE_RETRY_WATCHDOG=1`、herdr 0.7.5)では、上限の文が `⏺ Monitor "..." stream ended` のブロックの中に `⎿  You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)` として出て、入力欄の上は `✻` の状態行(`Weekly li… · Retrying in 29m (Nov 21 at 3pm) · attempt 1/55` が 2 列に折り返したもの)だけだった。fixture はこの 2 つの断片と、2026-11-07 の evidence の入力欄と footer の形から組み立てる。実際の画面の全体は手元にない(2026-11-20 にこのマシンの herdr のサーバーの pane を一覧したが、その org の pane はなかった)。再試行の行の括弧の時刻が上限のリセットの時刻と同じであることも、この 1 つの画面でしか見ていない。版が変わって形が違えば、今までどおり stall の件になる
- API の再試行の状態行は `✻ Can't reach the… · Retrying in 4s · attempt 13/300` の形(`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 3)。上限の文を伴わないので、3 つ目の読み方には入らない
- ralph は claude の座席に `CLAUDE_CODE_RETRY_WATCHDOG=1` と `CLAUDE_CODE_MAX_RETRIES` を入れる(#306)。429 の待ちは `CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS` で区切られ、上限が長いと再試行が何度も続く(`docs/evidence/org-seat-resume-retries-2026-11-07.md`)

## Affected areas

- `internal/org/usage_limit_screen.go`、`internal/org/resume.go`
- `internal/org/testdata/resume/`(新しい fixture)、`internal/org/usage_limit_screen_test.go`、`internal/org/resume_screens_test.go`(fixture の表)、同じ fixture を読む `internal/org/send_screen_test.go` と `internal/org/send_screen_marker_test.go`(表に期待を足す)、`internal/org/resume_test.go`、`internal/cli/org_resume_test.go`、見張りのテスト(`internal/org/watch_usage_limit_test.go`)
- `.claude/rules/ralph/agent-messaging.md`、`templates/base/.claude/rules/ralph/agent-messaging.md`
- `.claude/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`(`.agents` の 2 写しは `scripts/sync-skills.sh` で作る)
- `docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/watch-usage-limit-retry.html`。かんたんな図 1 枚(`overview`)と、詳しい図 2 枚(`detail-overview`、`detail-reading`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 3 枚の図を `shot` で撮って読んだ。直したのは 1 点で、ファイルの地図の agent-messaging.md のパスが箱からはみ出し S3 のバッジと重なっていたので、箱を広げバッジを群の見出しの横に移した。撮り直して、どれもきれいに出ることを確かめた

## Design decisions

Critical forks: None。次の 3 点は、系列の指示(計画は推す案で自動承認)に従って自律で選んだ。

1. 上限と読む証拠は、手番の中の上限の文と、今の再試行の行の時刻の一致に置く。上限の文だけでは、tool の出力が文を引用したあとの通信断の再試行を取り違える(Codex の計画助言の指摘 1)。再試行の行だけでも、エラーの要約(`Weekly li…`)は pane の幅で切れるので足りない。`usage_limited` の件は director を「解除を待つ」か「アカウントを替える」へ向けるので、通信断を上限と取り違えると、本当の原因が何時間も残りうる
2. 上限の文は今の手番(最後の prompt より後)からだけ探す。前の手番の上限の文が画面に残ったまま、今は通信断で再試行している画面を上限と読まないため。prompt が画面の外に出ているときは手番の境が分からないので、入力欄の上のすべてのブロックを探すが、そのときも 1 の時刻の一致を求める
3. resume は上限と読めた座席にも何も打たない。outcome は waiting のまま理由だけを変える。CLI は再試行を自分で続けるので、resume の続行の文を打つと再試行と重なる

## Acceptance criteria

- [x] AC1: issue の画面を元にした fixture で、`UsageLimitScreen("claude", …)` が上限と読み、返す行が `⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)` を含む(テスト)
- [x] AC2: 見張りが、stall の条件が立った座席でこの画面を読むと、stall の ALERT ではなく `usage_limited` の件(`REASON: usage_limit`、`SEATS:` に座席、本文に上限の文とリセットの時刻)を上げる(テスト)
- [x] AC3: `ralph org resume --dry-run` が、herdr が working と報告する座席でこの画面を読むと、`waiting for the usage limit to reset: <行>` を理由に出す。何も打たない(テスト)
- [x] AC4: 次の画面は上限と読まない(テスト): API の再試行の状態行だけの画面(2026-11-07 の evidence の 3 の形)、前の手番(最後の prompt より前)にだけ上限の文がある再試行の画面、上限の文が glyph のあとでなく返答の文の途中にある再試行の画面、再試行の状態行のない画面に上限の文が手番の中にあるだけの画面(今の stopped の読み方の対象でなければ上限と読まない)、tool の出力が glyph つきで上限の文を引用したあとに通信断で再試行している画面、同じ手番で上限の文のあとに通信断の再試行(時刻なし)に移った画面、prompt が画面の外で上限の文はあるが再試行の行の時刻と一致しない画面
- [x] AC5: codex の読み方と、fixture の表の今の行は変わらない。今ある見張りと usage limit のテストは変えずに通る。resume のテストのうち、herdr が working の座席の呼び出しの列(`get <n>` だけ)を固定しているものは、pane の読みを 1 つ足すのに合わせて直す(意図した変更として、報告に挙げる)
- [x] AC6: agent-messaging.md と director skill の 4 写しが、この待ちの形を書く。tech-debt の 514 行に追記がある。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る

## Implementation outline

1. S1 上限の 3 つ目の読み方: `UsageLimitScreen` に再試行の待ちの読み方を足し、fixture と、AC1・AC4 の単体テストを足す — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'UsageLimit|ResumeScreen|ClassifyResume'`
2. S2 resume と見張り: `resumeLook` の waiting の理由に上限を足し、AC2・AC3 のテストを足す — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Resume|UsageLimit|WatchUsage' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Resume'`
3. S3 文書: agent-messaging.md(root と templates)、director skill(root と templates、`scripts/sync-skills.sh`)、tech-debt — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC6。issue #353 の「期待すること」の 2 点(`usage_limited` の件、`resume --dry-run` の出し方)
- Documentation drift to check: agent-messaging.md の `usage_limited` の段落、director skill の「解除を待っている座席」、`usage_limit_screen.go` と `resume_screens.go` の doc comment の画面の決まり
- Evidence to capture: fixture の出どころ(issue の画面の断片と 2026-11-07 の evidence)を fixture の表の注に書く

## Test plan

- Unit tests: `UsageLimitScreen` の 3 つ目の読み方(AC1、AC4 の 4 つの否定)、折り返した上限の文、上限の文が `⏺` で始まる場合、行の長さの切り詰め
- Integration tests: 見張りの `usage_limited` の件(fake の herdr で pane の読みを返す)、resume の `--dry-run` の理由(working と waiting の両方)
- Regression tests: fixture の表(`TestUsageLimitScreen_EveryFixture` と resume の判定の表)、今の usage limit と resume のテスト
- Edge cases: 状態行が `✻` 1 字の行と本文に分かれて折り返す形、prompt が画面の外の画面、手番の中に上限の文が 2 つある画面(最後のものを返す)、working の座席の pane の読みが失敗したとき(理由は `herdr reports working` のまま)
- Evidence to capture: `go test -run 'UsageLimit|Resume'` の要約、`./scripts/run-test.sh` の結果

## Risks and mitigations

- API の再試行を上限と取り違える: 上限の文(glyph のあと)を今の手番に求め、状態行の要約や括弧の時刻では判断しない。取り違えても、上がるのは stall の件の代わりの `usage_limited` の件なので、人の目には届く
- 上限の文の折り返しで見落とす: ブロックの行を空白でつないでから探す
- resume が working の座席の pane を読むことで herdr の呼び出しが増える: 座席ごとに 1 回で、resume は人か director が打つときだけ走る
- 実際の画面の全体がない: fixture は issue の断片から組み立てる。版が変わって形が違えば、今までどおり stall の件になる(黙らない)

## Rollout or rollback notes

- 見張りと resume の画面の読みだけが変わる。設定と記録の形は変わらない。戻すときはこの PR を revert する
- resume の理由は、今の 2 つ目の読み方(5 時間の上限の footer の `Continuing automatically at`、`claude-5h-limit-continuing.txt`)の画面でも、`the CLI is retrying or waiting` から `waiting for the usage limit to reset: <行>` に変わる
- release ノートの候補: 「見張りが、手番の中で再試行して利用上限の解除を待つ claude の座席を `usage_limited` の件にする。`ralph org resume` はその座席を上限の解除を待つ座席として出す」。director skill と agent-messaging.md の文面は upgrade で届く

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 1 件: [MEDIUM] 手番の中の glyph つきの上限の文だけでは、tool の出力の引用のあとの通信断の再試行を上限と取り違え、director を誤った立て直しへ向ける。計画に入れた: 今の再試行の行の括弧の時刻と上限の文のリセットの時刻の一致を条件にし、AC4 に否定の 3 形を足した。stall の件に上限の文を確かめていない手がかりとして載せる案は Non-goals に回した。git の状態の差は、助言の最中に orchestrator が計画の Visual review の節を書いたことによるもので、reviewer の書き込みではない。consult(consult-plan-retry353、判定「直してから進める」)の 3 点も入れた: AC5 を resume のテストの呼び出し列を直す形に、実物の画面を取れるか確かめた(このマシンの herdr にその org の pane はなかった)、5 時間の上限の footer の画面でも resume の理由が変わることを Rollout に
- [x] Plan approved — 2026-11-20、系列の指示(計画は推す案で自動承認)で承認した。人の目での承認はしていない
- [x] Branch created
- [x] Implementation started
  - S1(10541be6、implementer): 確認コマンドが exit 0(orchestrator も `-count=1` で打ち直して ok)。fixture は `internal/org/testdata/resume/claude-weekly-limit-retrying.txt`(issue の 6 行はバイト単位で同じ、入力欄と footer は `claude-weekly-limit-under-prompt.txt` の末尾 5 行)。返す行は `⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)`。ずれ: `TestUsageLimitScreen_EveryFixture` の waiting の分岐を「footer の行か、再試行の待ちの行」に広げた(新しい fixture を通すため、今の fixture の判定は変わらない)。`send_screen_marker_test.go` は名前で絞るので変えていない。変異 12 種類がすべて落ちる。機械的セルフチェックは 5 項目とも clean か fixed。`resume_screens.go` の `Retrying in` のコメントに 3 つ目の読み方への参照を足すのは S2 で
  - S2(e57ddb65、implementer): 確認コマンドが exit 0(orchestrator も `-count=1` で打ち直して ok)。resume は working の座席も pane を 1 回読み、上限と読めたら理由を `waiting for the usage limit to reset: <行>` にする(idle で waiting の座席も同じ)。ずれ: CLI の表示(`resumeTargetText` が waiting の座席に画面の行を優先して出すので、この理由のときは理由を出す分岐を足した)、理由の前置きの定数を `ResumeReasonUsageLimitWait` として export、`--help` に 1 文。AC5 の意図した変更: `TestOrgResume_RetryingOrWorking_Waits` の working の行の呼び出しの列に `read pane-1 60` を足した。見張りのテスト(implementer と leader の 2 通り)で `usage_limited` の件の本文を確かめた。変異 5 種類がすべて落ちる。新しい fixture の working の座席の出力: `waiting  org-a/implementer  claude  waiting for the usage limit to reset: ⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)`。機械的セルフチェックは 5 項目とも clean か fixed
  - S3(5ddd4ef9、implementer、sonnet): agent-messaging.md(root と templates)の `usage_limited` の段落、director skill の 4 写しの解除を待つ座席の文と resume の出し方、tech-debt の 514 行の carry-over。`check-sync.sh`、`check-skill-sync.sh`、`go test ./internal/org/ -run 'Prompt|Skill|DocsMatch|OrgSkill'` が通った。機械的セルフチェックは 5 項目とも clean
- [x] Review artifact created — `docs/reports/self-review-2026-11-20-watch-usage-limit-retry.md`(94fa8031、Merge: yes、MEDIUM 1・LOW 6)。S4(cfe09a8f、implementer、sonnet)で M1(herdr が working の座席は、画面が waiting で上限と読めたときだけ理由を変える。上限で止まった画面は `herdr reports working` のまま)と L1(見張りのコメント)を直した。L2(working の座席の pane の読みの失敗を出さない)は計画で決めた挙動なので残した。L3〜L6 は直さずに記録(L5 は reviewer が tech-debt に 1 行足した)。確認コマンドが exit 0、変異で新しいテストが落ちることを確かめた。機械的セルフチェックは 5 項目とも clean
- [x] Verification artifact created — `docs/reports/verify-2026-11-20-watch-usage-limit-retry.md`(f8303063、pass)。LOW の文書のずれ 6 件のうち、Go の中の 2 件はインラインで直した(`ralph org watch --help` に 3 つ目の読み方、`ResumeReasonUsageLimitWait` の doc に M1 の条件。help の文とコメントだけ、機械的セルフチェックは 5 項目とも clean、`go test -run 'OrgWatch|Resume'` は ok)。残りの 4 件(仕様の改訂、tech-debt の持ち越し 6 行、800 行を超えた 2 つのテストファイル、AC の印)は /sync-docs に渡す
- [x] Test artifact created — `docs/reports/test-2026-11-20-watch-usage-limit-retry.md`(f246da8f、pass)。`./scripts/run-test.sh` は rc 0(shell 46 本、Go 11 パッケージ)。AC1〜AC5 のテストが通り、時刻の一致を外す変異は `TestUsageLimitScreen_RetryWaitShapes` の 5 subtest が落とした。base のテストを HEAD に当てて落ちたのは、AC5 が名指す呼び出しの列と新しい fixture の行の不足だけ
- [x] Sync-docs: pass。verify の V2(org runtime の仕様と resume の仕様に 2026-11-20 の改訂)、V4(tech-debt の持ち越しの注 6 行。sweep で見つけた 267・362・483 行にも)、V5(`usage_limit_screen_test.go` と `resume_test.go` が 800 行を超えた行)、V6(AC1〜AC6 に印)を直した。V1(`ralph org watch --help`)と V3(`ResumeReasonUsageLimitWait` の doc)は 29b5d5d6 で直し済み。README の 302 行に 1 文足した。`/org` skill は変えていない(理由は report)。report: `docs/reports/sync-docs-2026-11-20-watch-usage-limit-retry.md`
- [x] cross-review cycle 1 — codex の指摘 1 件(P2)を ACTION_REQUIRED 1 件にした(WORTH_CONSIDERING 0、DISMISSED 0)。3 つ目の読み方は、入力欄の上の最後のブロックが claude の状態行かどうかを見ない。手番の時間の行を出さない設定の座席が issue #353 の 2 行を引用して手番を終えると、見張りは stall を `usage_limited` に替え、resume は解除待ちの理由を出す。triage は `docs/reports/cross-review-triage-watch-usage-limit-retry.md`(bd55da61)
- [x] cycle 1 の直し — 51fe7ddf。最後のブロックが `*·✢✳✶✻✽` のどれかと、空白か文の終わりで始まることも求める(`claudeStatusLineStart`)。記号の組は `send_screen.go` の `claudeStatusSymbols` の 1 か所に置き、send の 2 つの正規表現(`sendClaudeBusyLine`、`sendClaudeTurnEnd`)もそこから組む。文字列は base とバイト単位で同じ。返答の引用の回帰テスト 3 行、`✶` の行、`TestUsageLimitScreen_ClaudeStatusLineStart` を足した。計画からのずれ(Scope の 3): Scope の 3 は状態行を「`Retrying in` と `attempt` を両方含む」とだけ書くが、ブロックの始まりも求めることにした。Scope は承認の digest に入るので書き換えていない(digest は `70b063e49ad8` のまま)
- [x] self-review cycle 2 — `docs/reports/self-review-2026-11-20-watch-usage-limit-retry.md` の Run 2(9d5ae3a4、Merge: yes、LOW 5: L7〜L11)。a5ec5464 が L7(残る取り違えの記述をコードが受ける範囲に合わせた)、L10(状態行の 2 つの範囲の違い)、L11(doc の重複と help の改行)を、コメントと help の改行だけで直した。L8(tech-debt の数)、L9(L3・L4・L6 を持つ行がない)、L7 の register の部分は sync-docs cycle 2 で扱った
- [x] verify cycle 2 — `docs/reports/verify-2026-11-20-watch-usage-limit-retry.md`(0ea15fd2、pass、LOW 4 件: V7〜V10)。AC1〜AC6 と、cross-review の指摘の 3 つの形が直ったことを確かめた。V7(仕様の始まりの条件)、V8(この Progress)、V9(tech-debt の持ち越し 2 行)、V10(cycle 1 の注の数)は sync-docs cycle 2 で扱った
- [x] test cycle 2 — `docs/reports/test-2026-11-20-watch-usage-limit-retry.md`(ddeb39f7、pass)。AC の 2 つの絞り込みが通り、変異 3 つ(始まりの確かめを消す、時刻の一致を常に真にする、記号の組から `✻` を外す)がどれもテストを落とした。`./scripts/run-test.sh` は rc 1 で、落ちたのはこの branch が触っていない `TestGate_TwoGatesRunTheirScriptsOneAtATime`(gate.lock の待ちの競合。単独の 20 回と `internal/org` 全体の打ち直しは通った)だけ
- [x] sync-docs cycle 2 — `docs/reports/sync-docs-2026-11-20-watch-usage-limit-retry.md` の cycle 2 の節。org runtime の仕様の改訂に始まりの条件と残る取り違え(23ff1bdb)、tech-debt に持ち越しの注 2 行と cycle 1 の注の数の直しと新しい 2 行(残る取り違え、L3・L4・L6)(b3ce2595)。README、`agent-messaging.md`、skill、resume の仕様は変えていない(理由は report)。コードは動かしていない
- [ ] PR created
