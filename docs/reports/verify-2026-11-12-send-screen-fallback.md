# Verify report: send-screen-fallback

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-send-screen-fallback.md`(issue #308。`- Approved:` の digest `123428594289` は `./scripts/plan-visual.sh digest` で出し直した値と一致した)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(origin/main `d2e5c6e9`、HEAD `71e8bfe3`、69 ファイル)。静的解析は `./scripts/run-static-verify.sh` の既定の範囲(変わった言語だけ。testdata の `.txt` がどの言語にも分けられないので full に落ち、選ばれた pack は golang だけ)。テストは走らせていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-11-12-send-screen-fallback.log`(git が無視する場所)。静的解析 2 回の全出力、build した binary の help、偽の herdr で binary を動かした probe の出力、origin/main から build した binary との比べを入れた
- Insight event: `docs/insights/events/2026-11-12-send-screen-fallback.jsonl` に verify の 1 行(cycle 1、pass、low 3)を足した。指示どおり、この report のコミットには入れていない
- ID の付け方: `V1` から `V3` は LOW で、/sync-docs に渡す drift。`I1` から `I3` は INFO

## Spec compliance

「満たす」は、コードとテストのソースを読んで AC を満たすと判断したという意味。テストが通るかは `/test` が確かめる。「観察」は、build した binary を、`PATH` の先頭に置いた偽の herdr(シェルの台本)で動かした結果で、evidence の節の名前を添えた。実機の herdr では、どの AC も動かしていない。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 herdr が working のまま、画面が入力待ちの座席へ `--timeout-ms` のうちに打ち、`sent` に `via=screen` | 満たす。実機で herdr が working を言い続ける座席への送信は未確認 | コード: `sendWaitForSeat`(`internal/org/send_fallback.go:126`)が herdr の最初の待ちに `--timeout-ms` から reserve を引いた時間を渡し、時間切れのときだけ `checkSendScreen`(`:186`)で 2 回読む。`verbs.go:642` が `screenPath.details()`(`send_fallback.go:60`)を Details の頭に付ける。テスト: `TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen`(claude と codex。herdr の呼び出しの列、`via=screen herdr_state=working`、`--timeout-ms` のうちに終わること、ロックの解放)。観察(probe send A): `agent wait` が idle にならず `agent get` が working を返す偽の herdr と `claude-idle-after-reply.txt` の画面で、`--timeout-ms 21000` の send が 18 秒で終了コード 0。最初の待ちは `--timeout 14000`、`sent` の Details は `via=screen herdr_state=working TYPE: QUESTION…` で、`submit_unconfirmed` は付かなかった。実機の症状は probe の evidence(`docs/evidence/org-send-screen-probe-2026-11-12.md` の 9 節)で再現しなかった |
| AC2 動作中の行、選択の画面、終わりの行も起動直後の形も無い、判断できない、2 回で違う、のどれかでは打たず、エラーに herdr の状態・画面の判定と理由・次の手 | 満たす | コード: `checkSendScreen` と `sendReadIdleScreen`(`send_fallback.go:219`)が idle 以外の画面と読みの失敗を verdict にし、`sendScreenWaitErr`(`:350`)と `sendScreenNextSteps`(`:359`)がエラーの文を作る。テスト: `TestOrgSendScreen_ScreenNotWaiting_TypesNothingAndSaysWhy` の 7 行(2 回の読みが違う、busy、2 回目が busy、choice、終わりの行が無い(unknown)、読みの失敗、`agent get` の失敗で state が unknown)。どの行もエラー全文、`directorSkillSendWaitTimeoutPhrase` が残ること、何も記録しないことを比べる。画面の分類は `TestClassifySendScreen_EveryFixture` と `TestClassifySendScreen_Edges`(unknown の driver、空の画面、入力欄なしなど)。観察(probe send B): busy の画面で終了コード 1、21 秒。stderr は待ちの切れの文のあとに `screen check: herdr state working, the screen was busy (a busy line is on the screen: "✶ Deliberating… (6s · ↓ 100 tokens)")` と 3 つのコマンドが続き、`ralph org read` と `ralph org status --live` には渡した `--state-dir` が付いた。herdr の 2 回目の待ちは残りの `--timeout 6376` だった |
| AC3 ロックの下の読み直しで入力待ちでなければ打たない | 満たす | コード: `sendScreenUnderLock`(`send_fallback.go:274`)が herdr の待ちの代わりに画面を読み、`verbs.go:546` が画面の道でだけ呼ぶ。テスト: `TestOrgSendScreen_ScreenChangedUnderThePaneLock_TypesNothing`(busy になった画面と読みの失敗。何も打たず、何も記録せず、ロックを放す)。binary では観察していない |
| AC4 Enter の前に文が入力欄に無ければ Enter を押さず、打ったが送っていないと返す | 満たす | コード: `sendScreenSubmitChecks` の `beforeEnter`(`send_fallback.go:304`)が `sendInputHolds`(`send_screen.go:259`)で確かめ、`verbs.go:619` が `SendProgressTextTyped` と `(text typed but not submitted)` を返す。CLI の note に「入力欄が無ければ Enter を押さない」が足された。テスト: `TestOrgSendScreen_TextNotInTheInputBox_EnterNotPressed`(入力欄が空、読みの失敗)、通る側の `TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed`(実機の複数行の画面 3 つ)、`TestSendInputHolds_TypedMultiLineFixtures`、`_Refuses`、`_ClaudePasteBoxes`。claude が長い文の頭を落とす件は画面から見分けられず、#319 に分けた(tech-debt の 468 行目の (c)) |
| AC10 画面の道で Enter のあと herdr が working のまま文が入力欄に残れば `submit_unconfirmed=true`。入力欄が空になったか動作中の行が出たときだけ確かめ済み | 満たす | コード: `confirm`(`send_fallback.go:315`)は herdr を見ず、`SendScreenSubmitted`(`send_screen.go:213`)が真のときだけ真を返す。`typeAndSubmit` は `confirm` があればそちらを使う(`verbs.go:846`)。テスト: `TestOrgSendScreen_AfterEnter_ConfirmedOnTheScreenOnly`(残ったまま、`raw=true` の並び、2 回目の読みで空になる)、`TestSendScreenSubmitted`。Enter のあとの呼び出しが pane の読みだけで、`working,blocked` の待ちが無いことも比べている。claude の `Press up to edit queued messages` は、積まれた文として確かめ済みに数える(計画の Scope の「積まれている=動作中」と同じ扱い) |
| AC11 確かめの手前で時間切れになり、画面が入力待ちでない座席が `--timeout-ms` のうちに idle か done になれば、今までの道で届く | 満たす。計画の例(30 秒、20 秒、25 秒)の尺では動かしていない | コード: `sendWaitForSeat` が、確かめのあと残りの時間で herdr の待ちに戻る(`send_fallback.go:158`)。確かめは reserve の半分まで(`:149`)。テスト: `TestOrgSendScreen_ScreenBusyThenHerdrDone_SendsByHerdrsPath`(縮めた尺で、確かめの 150 ms 後に done)、`TestOrgSendScreen_PaneReadHangs_HerdrsWaitGetsTheRest`(pane read が詰まっても残りが herdr の待ちに渡る。done が来る場合と来ない場合)、`TestOrgSendScreen_HerdrDoneAtTheCheck_ScreenNotRead` |
| AC5 `--no-screen-fallback` では herdr の待ちだけで、時間切れは今の文。idle か done の道は今と同じ | 満たす | コード: `sendScreenReserve`(`send_fallback.go:101`)が `NoScreenFallback`、claude と codex 以外の driver、reserve の不足で 0 を返し、そのとき待ちは今までの 1 回だけ(`:129-137`)。フラグは `internal/cli/org.go:1517`。テスト: `TestOrgSendScreen_NoScreenFallback_HerdrOnly`、`TestOrgSendScreen_HerdrIdleAtOnce_ScreenNotRead`、`TestOrgSendScreen_ReserveTooShortForTheCheck_HerdrOnly`、`TestSendScreenReserve_Table`(20250 で 6.75 秒、20249 で 0)、CLI の `TestOrgSend_NoScreenFallbackFlag_ReachesSend`。観察(probe send C): `--no-screen-fallback` では herdr の呼び出しが `agent wait … --timeout 21000` の 1 回だけで、エラーは今の文だけ(`screen check:` なし)。観察(send base vs head): herdr がすぐ done を返すとき、origin/main の binary と HEAD の binary で herdr の呼び出しの列と `sent` の Details が同じだった。違いは最初の待ちの `--timeout` が 30000 から 20000 になったことだけで、計画の Test plan と `TestOrgSendScreen_HerdrIdleAtOnce_ScreenNotRead` の説明どおり |
| AC6 `status --live` が herdr の状態と画面の判定を出し、食い違いに印と `herdr agent explain` の案内。`--live` なしの出力は今と同じ | 満たす | コード: `LiveSeatStates`(`internal/org/status_live.go:68`)、`Mismatch` の条件(`:84`)、`printLiveMismatches`(`internal/cli/org.go:2184`)。テスト: `TestLiveSeatStates_*` の 7 本、CLI の `TestOrgStatus_WithoutLiveTheOutputIsUnchangedAndHerdrIsNotCalled`、`TestOrgStatusLive_TableShowsHerdrAndTheScreenAndTheMismatch`、`_JSONAddsALiveObjectPerActiveSeat`、`_NoHerdrReachableStillExitsZero`、`TestOrgStatus_HelpDescribesLive`。観察(probe status --live): herdr が working で画面が idle の座席に `HERDR`・`SCREEN` の列と `mismatch:` の行が出て、`--json` に `live` が付いた。観察(status base vs head): `--live` なしでは、origin/main の binary と HEAD の binary の表と `--json` がバイト単位で同じで、偽の herdr は 1 回も呼ばれなかった。origin/main の binary は `--live` を知らない(`unknown flag`) |
| AC7 codex 0.166 の footer で入力欄が見つかり、入力待ちの fixture で空と読める。既存の fixture の `ClassifyResumeScreen` は変わらない | 満たす | コード: `resumeCodexFooterLine`(`internal/org/resume_screens.go:290`)に `(?: · .+)?` を足しただけ。テスト: `TestFindResumeInputBox_Codex0166Footer`(0.166 の画面 8 つ)、`TestResumeCodexFooterLine`、`TestClassifyResumeScreen_Fixtures` と `TestResumeInputHolds` に足した行。diff は既存の行を変えず、足しているだけ |
| AC8 入力待ちの判定が、採った画面と既存の testdata のすべてで期待どおりに分かれる | 満たす | `sendScreenFixtureWants`(`internal/org/send_screen_test.go:24`)のキーと `internal/org/testdata/resume/` のファイルは、どちらも 61 で、同じ集合だった(evidence の「fixtures vs sendScreenFixtureWants keys」)。`TestClassifySendScreen_EveryFixture` は両方向の抜けを落とし、名前に `-idle-` か `-working-` などを含む fixture の kind も確かめる。起動直後、手番の途中、入力待ち、積まれた文、送った直後の画面は claude と codex の両方にある |
| AC9 help、org skill、recipe、AGENTS.md がこの挙動に合う | 満たす | 下の「Documentation drift」の表。AC9 が名指しする面はどれも挙動と合っている。名指しされていない面に V1 から V3 がある |

issue #308 の受け入れ条件 1 は AC1 と AC2、受け入れ条件 2 は AC6 が受け持つ。条件 2 は「status か doctor」で、計画の D7 のとおり status で満たしている。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(1 回目) | fail(exit 1) | golangci-lint だけが `Error: parallel golangci-lint is running` で止まった。ほかの session の lint と重なったため。直後の `pgrep -fl golangci-lint` には何も無かった。ほかの検査は通った |
| `./scripts/run-static-verify.sh`(2 回目) | pass(exit 0) | gofmt ok、`go vet ./...` は何も出さず、golangci-lint は `0 issues.`、staticcheck は何も出さなかった。shellcheck と `sh -n`、`jq -e`、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill が lock-step)、`check-template-purity.sh`、tech-debt の plan の参照、branch の secret scan(`d2e5c6e9..71e8bfe3` clean)も通った |
| `./scripts/plan-visual.sh digest <plan>` | `123428594289` | plan の `- Approved:` の値と一致 |
| `go build ./cmd/ralph`(HEAD と `d2e5c6e9`、scratch に出力) | pass | help の確かめと probe に使った。repo には何も残していない |
| `shasum -a 256` と `cmp`(4 つの写しと recipe の双子) | 一致 | org skill の 4 つは `0921a99d…`、director skill の 4 つは `ca8fb5f7…`。`docs/recipes/isolated-herdr-server.md` と `templates/base/` の写しは同じ |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org send --help`(built binary) | yes | 段落の数字は既定値と合う。10 秒は `defaultSendScreenReserve`、3 秒は `defaultSendScreenReadGap`。「3 分の 1 が 6 秒と Enter の前の待ちに満たない」は `reserve < 2*gap + enterDelay` で、「Enter の前の待ちが 4 秒を超える」は `10s < 6s + enterDelay` にあたる。4 秒ちょうどなら reserve は 10 秒で残るので、「over」で合っている。`--no-screen-fallback` と `--timeout-ms` の説明も出る |
| `ralph org status --help`(built binary) | yes | `--live` の段落とフラグ。「herdr のサーバーを起動しない」は、`LiveSeatStates` が herdr の gate を通らず、`agent get` と `pane read` だけを呼ぶこと(probe の呼び出しの記録)と合う |
| `ralph org resume --help` | 変更不要 | help は入力欄の見つけ方に触れていない |
| org skill の 4 つの写し(send と status の行、folder trust の段落) | yes | 「20250 未満」は 3 × (6 + 0.75) 秒。画面の道で送れたかを画面で確かめること、note、`via=screen herdr_state=<状態>` もコードと合う |
| director skill の 4 つの写し | yes | エラーの文の `screen check:` と `changed under the lock of its pane` は `sendScreenWaitErr` と `sendScreenUnderLock` の文に含まれる。入力待ちの説明は起動直後の画面を省いた要約(I2) |
| `internal/org/prompts/leader.md` | yes | 前面で待つ leader の画面は動作中なので届かない、という説明はフォールバックの規則と合う |
| `docs/recipes/isolated-herdr-server.md` と templates の双子 | yes | 新しい節の手順(`status --live`、`herdr agent explain`、入力待ちの条件、20250、`--no-screen-fallback`、実機で通していないこと)はコードと probe の evidence に合う |
| `AGENTS.md`(internal/org の行) | yes | `send_fallback.go`、`send_screen.go`、`status_live.go` と pane lock の「画面の道では読み直す」が入った |
| `README.md` | yes | director から leader への send の説明に、画面の確かめと `status --live` が入った。2 回の読みが同じ画面であることは省いた要約(I2) |
| `.claude/rules/ralph/agent-messaging.md` | 変更不要 | send の待ちに触れていない |
| `docs/specs/2026-08-01-org-runtime.md` | no(V1) | 下の V1 |
| `docs/tech-debt/README.md` | no(V2、V3) | この PR の行(468 行目)は足された。発火した Trigger の注記が無い(V2)。self-review の LOW 7 件が register に無い(V3) |
| plan の AC の印と Progress checklist | lag(I1) | 下の I1 |

### V1(LOW): org runtime の仕様に、この変更の改訂の項が無い

`docs/specs/2026-08-01-org-runtime.md` は挙動の変更を日付つきの改訂の項で足していく。send の pane ロックの変更も、そうして記録した(99653051、49f0924c)。この PR はその仕様に触れていない。いまの文は 2 か所で、画面の道と合わない。

- 「2026-11-07 改訂」の (b)(63 行目)は、ロックを取ったあと座席が `idle`/`done` になるのをもう一度待つ、と言う。画面の道では待つ代わりに画面を読み直す(`sendScreenUnderLock`)
- FR-3(104 行目)は、追加の指示は `herdr agent wait --until idle` を挟んだ `send` を通す、と言う。いまは herdr が working と言っていても、画面が入力待ちなら打つ

計画の Verify plan の drift の一覧にこの仕様は無いので、AC9 は満たしている。/sync-docs で「### 2026-11-12 改訂(fix/send-screen-fallback)」を 1 項足すとよい。中身は、画面のフォールバック(既定で有効、`--no-screen-fallback`)、ロックの下の読み直し、`sent` の `via=screen`、`ralph org status --live`(既定の status は台帳だけを読むまま)。本文は書き換えない、という仕様の決まりに沿う。

### V2(LOW): tech-debt の Trigger が 6 行で発火し、注記が無い

`docs/tech-debt/README.md` の Trigger の列を python で読み、この diff が満たす行を探した。どの行にも 2026-11-12 の注記は無い。この register は、発火したら払ったか持ち越したかを日付つきで書く決まり。

- 226 行目(b)「the next change to `Send`」: `Send` を変えた。残りの半分(`approval_test.go` の `TestCheckTaskPlanApproved_StaleOrMistypedDigest` の doc の `approval gets` だけの行)は払っていない
- 337 行目「the next change to `Send`'s idle wait」: 最初の待ちを `--timeout-ms` から reserve を引いた長さにし、そのあとに画面の確かめを入れた。払っていない。前面で待つ Codex の leader の画面には動作中の行が出るので、フォールバックはこの行の症状を直さない。注記にはそこまで書くとよい
- 365 行目「the next change to `Send` … or any of these files growing past its count here plus 300」: `Send` を変えた。さらに `internal/org/verbs.go` が 2501 行から 2597 行になり、1 列目の 2277 行に 300 を足した 2577 行を超えた(`wc -l`、origin/main と HEAD)。払っていない
- 457 行目「the next change to … `defaultResumePaneLockWait`, `ClassifyResumeScreen`, `ResumeInputHolds`」: `defaultResumePaneLockWait` の doc comment を書き換えた。`resumeCodexFooterLine` も変えたので、`findResumeInputBox` を通して `ClassifyResumeScreen` と `ResumeInputHolds` の結果が変わる。R6 が `findResumeStop` を変えたときも、この行は発火と数えている。(d)(e) の fixture は足していない。今回の複数行の fixture は文の頭ではなく終わりを見せる画面なので、(e) にはあたらない
- 460 行目「the next probe of codex's screens, or the next change to `resumeCodexFooterLine`」: 発火し、大部分を払った。画面は `docs/evidence/org-send-screen-probe-2026-11-12.md` の 6 節に残り、形は正規表現と fixture に入り、comment は見た形(0.165 と 0.166)と binary の文字列から取った形を分けて書いている。(c) の起きる場面(codex の release が footer を変え、codex の座席がすべて `input box has text` になる)は、codex 0.166.0 で実際に起きた(`resume_screens_test.go` に足した行の comment)。残りは「reason に footer の形を出すか」の検討だけ。行を一部解決にするか、注記を足す
- 461 行目「the next change … to `resumeCodexFooterLine`」: 発火した。(b) は一部払ったように見える。`TestResumeCodexFooterLine` には、状態の行の終わりの錨(F2)を落とすと赤になる `GPT-6-Luna · low · Context 7% used ·` と、前の項目を要る条件(F3)を落とすと赤になる `Context 7% used · main · No chan…` と `please keep Context 5% used in mind` がある。`% context left` の終わりの錨(F6)を確かめる下書き(`50% context left is plenty, go on` など)は無い。どれが赤になるかは読んで出した見込みで、`/test` の mutant で確かめる

境目の 1 件は I3 に回した。

### V3(LOW): self-review の残りの LOW 7 件が register に無い

`docs/reports/self-review-2026-11-12-send-screen-fallback.md` は、L2、L4、L5、L6、L7、L9、L10 を直さずに残し、Tech debt の節で、オーケストレーターが 1 行にまとめて register に足すよう書いている。いまの register には無い。この PR の行(468 行目)の (a) から (c) とは中身が重ならない。plan を archive すると、ほかに残る場所は report だけになる。/sync-docs か `/pr` の前に 1 行足す。

### INFO

- I1: plan の AC1〜AC11 はすべて `[ ]` のままで、Progress checklist の `Branch created` から `PR created` も印が無い。Affected areas は `pane_lock.go` を「変えない見込み」と書くが、doc comment を変えた(コードは変えていない)。どれも記録の遅れで、判定には使わない
- I2: director skill は入力待ちを「手番の終わりの行の下に空の入力欄」とだけ言い、起動直後の画面を省く。README は「2 回とも入力待ち」とだけ言い、2 回が同じ画面であることを省く。どちらも要約で、正しい規則は help、org skill、recipe にある
- I3: 137 行目の Trigger「the next `fakeHerdr` delay-injection hook added for any reason」。この PR は `sendFallbackHerdr`(`fakeResumeHerdr` を包む偽物)に `doneAfter`、`blockRead`、`blockGet` を足した。`fakeHerdr` そのものではないので、発火とみなすかは /sync-docs の判断に任せる

## Observational checks

偽の herdr は scratch の `probe/bin/herdr` に置いた。呼び出しをすべて記録し、`agent wait` は `--timeout` の分だけ眠ってから失敗し、`agent get` は working を返す。`pane read` は、send-text の前は画面の fixture を、send-text のあとは入力欄に文がある画面を、send-keys のあとは `claude-submitted-just-now.txt` を返す。台帳は `spawned` の 1 行だけを手で書いた(seat-1、claude、pane-1)。実機の herdr、claude、codex は 1 度も呼んでいない。結果は evidence の各節にある。

- probe send A(AC1): 上の表のとおり。呼び出しの列は `agent wait`(14000)、`agent get`、`pane read` 2 回、ロックの下の `pane read`、`pane send-text`、Enter の前の `pane read`、`pane send-keys`、Enter のあとの `pane read`。単体テストの期待の列と同じ
- probe send B(AC2): 上の表のとおり。21 秒で終わり、`--timeout-ms` を超えなかった
- probe send C(AC5): 上の表のとおり
- send base vs head(AC5): 上の表のとおり
- probe status --live と status base vs head(AC6): 上の表のとおり

## Coverage gaps

- 実機の herdr が手番の終わった座席を working と言い続ける場面で、送信は 1 度も通していない(AC1、AC2、AC11)。probe の evidence の 9 節で症状が再現せず、この PR の tech-debt の行(468 行目の (a))に記録がある。単体テスト、CLI のテスト、偽の herdr を使った binary の観察が覆うのは、ralph の側の判断と呼び出しの順まで。herdr の本物の `agent wait` が ctx で殺されたときの振る舞いと、本物の pane の読みにかかる時間は覆っていない
- 既定の `--timeout-ms` 30000 での動きは観察していない。probe は時間を縮めるため 21000 で動かした。reserve が最小に近い値(20250 から数秒)で、確かめの herdr 呼び出しが Enter の前の待ちの半分(375 ms)に収まるかは測っていない(self-review の L10)
- `TestOrgSendScreen_AgentGetHangs_EndsAtTheDriverCallTimeout` が示す「`agent get` が詰まっても画面で届く」は、`DriverCallTimeout` を確かめの時間より短くしたテストの中でだけ起きる。本番の既定では先に確かめの ctx が切れる(self-review の L9)。AC には関わらない
- 計画の Test plan の Integration tests は「CLI の send と status --live を偽物の herdr で通す」と言う。CLI のテストで画面の道(`via=screen`)を通すものは無い。CLI の send のテストは、フラグが `Send` に届くことを、herdr がすぐ idle を返す場面で見るだけ。今回の probe send A から C はその代わりの観察で、テストとしては残っていない。足すなら、`stubResumeHerdr` の台本に「`agent wait` が idle にならない」場合を足し、`ralph org send` が `via=screen` を記録することと、busy の画面でのエラーの文を比べる CLI のテストが最も小さい
- テストの実行結果(`/test` の範囲)。`TestClassifySendScreen_EveryFixture` の 61 件、時間に頼るテスト(`PaneReadHangs_HerdrsWaitGetsTheRest` の余裕は約 100 ms)が CI で通るかは、ここでは確かめていない

## Verdict

- Verified: AC1〜AC11 はコードとテストのソースから満たす。AC1、AC2、AC5、AC6 は、build した binary を偽の herdr で動かして観察した。AC5 の「idle か done の道は今と同じ」と AC6 の「`--live` なしは今と同じ」は、origin/main の binary とも比べた。静的解析は 2 回目ですべて通った(1 回目は golangci-lint が並行の実行と重なって止まっただけ)。help、org skill と director skill の 4 つずつの写し、recipe の双子、leader の雛形、AGENTS.md、README は挙動と合う
- Partially verified: AC11 は縮めた尺のテストだけ。AC3、AC4、AC10 は単体テストのソースだけで、binary では観察していない
- Not verified: 実機の herdr での画面の道。テストの実行結果。既定の `--timeout-ms` での時間の配分
- Drift(/sync-docs へ): V1 org runtime の仕様の改訂の項。V2 tech-debt の Trigger 6 行(226、337、365、457、460、461)の注記。V3 self-review の LOW 7 件の register の行。INFO は I1〜I3
- Verdict: pass
