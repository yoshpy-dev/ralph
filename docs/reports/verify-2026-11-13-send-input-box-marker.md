# Verify report: send-input-box-marker

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-send-input-box-marker.md`(issue #322)。`- Approved:` の digest `sha256:6dc6132083c1` は、`./scripts/plan-visual.sh digest` で計算した今の計画の digest と一致した
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は 1)
- Scope: `git diff bbe2807d...HEAD`(HEAD `a199d598`)。32 ファイル、1628 行追加、26 行削除。Go の本体は `internal/org/send_screen.go` の 1 ファイルで、残りはテスト 6 ファイル、fixture 11、role prompt 2、skill の写し 8、証拠、計画、self-review の報告、insight event。テストは読んだだけで、回していない(`/test` の範囲)。実機の確かめはやり直していない(herdr、claude、codex は起動していない)
- Evidence: `docs/evidence/verify-2026-11-13-send-input-box-marker.log`(gitignore の対象。`run-static-verify.sh` が自分で書いた写しは `docs/evidence/verify-2026-11-13-080308.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: codex の実機の 4 画面(印の行が文の途中、最後、スクロールした入力欄の中、番号付き)で `sendInputHolds` が true | Met | `TestSendInputHolds_TypedMarkerFixtures`(`send_screen_marker_test.go:147`)が、`codex-input-typed-marker-line.txt`、`-line-last.txt`、`-line-scrolled.txt`、`codex-input-typed-marker-numbered.txt` の 4 つで、`sendTypedInputBox` が 0 桁目の頭の行(`› TYPE: QUESTION`、スクロールした画面は `› line 06: … lazy   ↑`、番号付きは `› TYPE: TASK`)を選ぶこと、`sendInputHolds` が true、`SendScreenSubmitted` が false を見る。本体は `sendTypedMarkerLineBox`(`send_screen.go:377`)で、`sendTypedInputBox` の先頭(`send_screen.go:322`)から呼ばれる。スクロールした画面は、頭の行に `↑` があるので codex の規則 2(終わりを見せる)で true になる |
| AC2: claude の改行 2 つと 1 つの文で true。改行 0(`❯ ❯ explain this`)と 4 行の文の placeholder も true のまま | Met | 同じテストの claude の 5 行。`claude-input-typed-marker-line.txt`(two-nl)、`-line-one-nl.txt`、`-line-wrapped.txt`(Enter の前の折り返した印の行)は新しい道で頭を選ぶ。`claude-input-typed-marker-no-nl.txt` と `claude-input-typed-paste-short.txt` は最後の印の行が 0 桁目の頭なので、新しい道は条件 1 で抜け、前と同じ道(`holdsHead` と規則 3)を通る。頭の行の `❯` のあとの U+00A0 は `strings.TrimSpace` が落とす(テストの期待の文は `TYPE: QUESTION`) |
| AC3: 今まで false の画面は false のまま。3 つの縁もテストで固定 | Met | `TestSendInputHolds_Refuses` と `TestSendInputHolds_NumberedMarkerLineRefuses`(選択の画面、`claude-dialog-below-old-input.txt` の入力欄の下の選択の画面、空の入力欄、ほかの文)は diff で 1 行も変わっていない(`send_screen_test.go` の numstat は追加 28、削除 0 で、追加は fixture の表と `TestSendScreenSubmitted` の行だけ)。新しい `TestSendInputHolds_MarkerLineRefuses`(`send_screen_marker_test.go:243`)の 16 行のうち、計画の 3 つの縁は「the indented marker line not of the message」(codex と claude)、「no marker line at column 0 above it」(codex と claude)、「claude, the indented marker line under the bottom rule」。各行は `sendTypedMarkerLineBox` が false で、`sendInputHolds` も false であることを見る。`claude-dialog-below-old-input.txt` は手で追った: 選択の行 ` ❯ 1. Yes` の上の ` Do you want to proceed?` が 1 字下げなので条件 3 で外れる |
| AC4: `SendScreenSubmitted` は文が残った新しい画面で false、Enter の 0.3 秒後の 2 画面で true。`findResumeInputBox`、`ClassifyResumeScreen`、`ResumeInputHolds`、`ClassifySendScreen` のコードは変えず、既存のテストの期待も変えない | Met(テストが通るかは `/test`) | false の側は `TestSendInputHolds_TypedMarkerFixtures` の 9 行、true の側は `TestSendScreenSubmitted` の 2 行(`send_screen_test.go:411-412`)。`resume_screens.go` は diff にない。`send_screen.go` の hunk は先頭の comment、`sendInputHolds` の doc、`sendTypedInputBox` と新しい関数、`sendCodexHasFooter` の 4 か所で、`ClassifySendScreen`(`send_screen.go:194`)とその下の関数には触れていない。`sendCodexHasFooter` を呼ぶのは `sendTypedInputBox` だけで、書き換え後の真偽(`sendCodexFooterInInput` か `footerStart < len(lines)`)は前と同じ。既存の 3 つの表(`resume_screens_test.go`、`send_screen_test.go`、`usage_limit_screen_test.go`)への変更は新しい fixture の行の追加だけで、削除は 0 行 |
| AC4b: 画面の道の `Send` の結合テストで、Enter 1 回、`SubmitConfirmed` が true、`sent` の Details に `via=screen` があり `submit_unconfirmed` がない | Met | `TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed`(`send_fallback_marker_test.go:21`)。herdr の呼びの並びを完全一致で比べ、`keys pane-1 Enter` は 1 回。`SubmitConfirmed`、`via=screen` の有無と `submit_unconfirmed` の不在、Details の全文、打った文が 1 回であること、pane のロックが空いたことを見る。注記(INFO): Enter の前の画面は証拠の「codex short」(alpha の文)で、Enter のあとの画面は「Enter のあとの画面」の codex(foxtrot の文)なので、2 つは別の文の画面になる。`SendScreenSubmitted` が見るのは空になった入力欄(`› Ask Codex to do anything`)で、履歴の文は見ないので、結果は変わらない |
| AC5: leader と director の role prompt、`/org` と `/director` の skill(4 写しずつ)に D3 の手順。prompt の文はテストで固定 | Met | leader: `leader.md:1128-1138`(typed protocol の節。打ち直さない、`ralph org read` で読む、「件を上げる」の節の手順で BLOCKED、キーを送らない、note の `clear or submit it there` より段落が優先)。director: `director.md:39-44`(ミッションの箇条)。`/org` skill の `send` の行(`.claude/skills/org/SKILL.md:360`)と `/director` skill の「答える順」の 2(`.claude/skills/director/SKILL.md:1126-1136`)。4 写しは `cmp` で一致(`.claude` と `templates/base/.claude`、`.agents` と `templates/base/.agents`。`.agents` の本文は `.claude` と同じ)。テストは `TestRenderRolePrompt_Leader_SendTextTypedNotSubmittedEscalates` と `TestRenderDirectorPrompt_SendTextTypedNotSubmittedReadsTheLeadersScreen`(herdr と launch_cmd の 2 形)。文面の裏取り: エラーの文は `verbs.go:639` と `645` でどちらも `(text typed but not submitted)` で終わり、`ExitCode` の既定は 1(`internal/cli/exitcode.go:45-46`)。note は `ralph org read` の呼びと `clear or submit it there` を持つ(`internal/cli/org.go:1474-1478`)。director の Enter は既存の許可(`/director` skill の 1151-1154 行と 2034-2036 行)の範囲に収まる |
| AC6: 確かめの記録があり、fixture はその画面の写し | Met | `docs/evidence/org-send-marker-probe-2026-11-13.md` がある。11 の fixture を probe の画面(`scratchpad/probe322/out/`)すべてと `cmp` で比べ、どれも 1 つの画面とバイト単位で一致した: codex の 4 つは `x1-short`、`x2-last`、`x3-medium`、`x4-numbered` の `.typed.txt`、claude の two-nl、one-nl、no-nl、paste-short は `c5`、`c6`、`c7`、`c1` の `.typed.txt`、wrapped は `cs-submit.typed.txt`、送れた 2 画面は `cs-submit.enter03.txt` と `xs-submit.enter03.txt`。証拠の「この確かめから決めたこと」が挙げる 11 画面と数が合う。打った文の定数は `TestSendMarkerProbeMessages_AsTyped` が証拠の表のバイト数と改行数に合わせている |

計画の Verify plan が求める「D2 の 5 つの条件がコードの分岐とテストの両方にあること」も確かめた。条件 1 は `send_screen.go:380` の `HasPrefix(lines[last.marker], marker)` と拒否の 1 行目、条件 2 は `sendTextInWant(last.first)` と拒否の 2〜5 行目(空の L を含む)、条件 3 は頭を探すループ(`send_screen.go:384-392`)と 6〜8 行目、条件 4 は `resumeInputBoxShape` と claude の `inputEnd <= last.marker`、codex の `sendCodexFooterStart`(`send_screen.go:403-415`)と 9〜13 行目、条件 5 は `send_screen.go:396` と 14、15 行目(受け入れる側は `TestSendInputHolds_MarkerLineUnderANumberedHead`)。知らない driver は `default` の分岐と 16 行目。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope。fixture の拡張子が分類されないので full fallback になり、Go の pack を選んだ) | PASS(rc=0) | hook の shellcheck、`sh -n`、awk の構文、`jq -e`: OK。`check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0)。`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照: OK。gofmt: ok。`go vet ./...`: 出力なし。golangci-lint: 0 issues。staticcheck: 出力なし。`secret-scan-branch` の `bbe2807d..a199d598`: clean |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-13-send-input-box-marker.md` | `6dc6132083c1` | 計画の `- Approved:` の行と一致 |
| fixture 11 と probe の画面の `cmp` | 11/11 一致 | AC6 |
| `git grep '^func Test'` の集合(`89a59858` と HEAD、4 ファイル) | 同じ 33 本 | self-review M1 の移動でテストは失われていない。`bbe2807d` にあったテストもすべて残る |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `sendTypedInputBox`、`sendInputHolds`、`sendTypedMarkerLineBox`、`sendTextInWant`、`sendCodexFooterStart` の doc と `send_screen.go` 先頭の comment | Yes | doc の条件 1〜5 はコードの分岐と同じ順で同じ意味。計画の D2 の条件 4 は codex の footer を `sendCodexHasFooter` で書くが、コードは S3 で足した `sendCodexFooterStart` を使う(真偽は同じ。INFO)。先頭の comment は V5 |
| `/org` skill の `send` の行(4 写し) | Yes | 括弧の中に D3 と note の扱いがある |
| `/director` skill の「答える順」の 2(4 写し) | Yes | INFO: 同じ節の終わりの「send が失敗したら resolve せず、上の 2 のとおりに送り直す」(1162-1163 行)は、2 の例外を「上の 2 のとおりに」で受けているので食い違いはない |
| `AGENTS.md` の `internal/org/` の行(screen fallback の項) | Yes | `sendInputHolds` を名指すだけで、入力欄の探し方は書いていない。変える必要はない |
| `.claude/rules/ralph/agent-messaging.md`(2 写し、`cmp` で同じ) | Yes | 「How send types a message」は打つあいだの失敗(届いたか分からない場合)だけを扱い、`(text typed but not submitted)` の手順は持たない。手順は role prompt と skill にあり、計画も rule を対象にしていない。足すかは `/sync-docs` が決めてよい(INFO) |
| org runtime の spec(`docs/specs/2026-08-01-org-runtime.md`) | No | V1 |
| tech-debt の、この PR の Known gaps の行 | No | V2 |
| tech-debt の、この PR が Trigger を満たした行 | No | V3 |
| tech-debt の 482 行目 (L11) と 483 行目 (9) の文 | No | V4 |

指摘(どれも文書だけ。`/sync-docs` で直す)。

- V1(LOW): spec には、`ralph org send` の動きを変えた PR ごとに日付の改訂の節がある(「2026-11-12 改訂(fix/send-screen-fallback)」、「2026-11-13 改訂(fix/send-claude-paste)」)。この PR の変更(Enter の前の確かめが、字下げされた印の行の上の 0 桁目の印の行を入力欄の頭とすること、打ったが送っていない send のあとの leader と director の手順)の節がない。計画の Verify plan も「spec の改訂の項」を確かめる文書に挙げている。2026-11-12 改訂 (c) の番号付きの行の説明は今も正しく、書き換えは要らない
- V2(MEDIUM): 計画の Affected areas は `docs/tech-debt/README.md`(この PR の Known gaps の行)を挙げ、Risks は「入力欄の頭が 0 桁目にあることに頼る … Known gaps に残す」と書くが、行がまだない(tech-debt は diff にない)。受け入れ条件ではないので verdict は変えないが、`/pr` の前に足す必要がある。入れるべき項目: (a) 頭が 0 桁目で続きが 2 字下げという形への依存(Claude Code 2.1.316、codex-cli 0.166.0、幅 53 桁の pane だけ。違う形では選び直しが働かず、今までどおり Enter を押さない)、(b) claude が上限に当たらずに返事をしたあとの画面を見ていない(計画の Open questions)、(c) 画面の道を ralph で通しで走らせていない(#308 の Known gaps (a) と同じ)、(d) 打つ前の確かめで前の send の残りがある場合(Non-goals)、(e) codex への bracketed paste(#326)
- V3(LOW): 行が base と同じで、持ち越しの注がないまま Trigger を満たした行。どれも計画の Scope の外なので、払うのではなく「carried over by fix/send-input-box-marker」の注を足すのが前例(#308、#319 の注)に合う。
  - 180 行目 (a)「`/org` skill の command table の次の編集」: `send` の行が 3,758 字から 4,086 字になった(4 写し)
  - 188 行目 (a)「the next change to `RolePromptVars` or the leader prompt」: `leader.md` の typed protocol の節に段落を足した(1332 行から 1344 行)。`RolePromptVars` と `{{SCOPE}}` は変わっていない
  - 262 行目「`/director` skill の受信箱の処理の節の次の変更」: `### 答える順`(1070 行)は `## 受信箱の処理`(781 行)の中にある。(1)〜(3) は手つかず
  - 380 行目「`/director` skill の send と確かめの手順の次の変更」: 「答える順」の 2 に段落を足した
  - 464 行目「`/director` skill か `director.md` の次の変更」: 両方を変えた。古い scaffold の扱いは決めていない
  - 482 行目「(L2, L7, L9, L10) `send_screen.go` の次の変更」と「(L11) `resumeMarkerLineBox` の 3 つ目の呼び手か `sendTypedInputBox` の次の変更」: 両方満たした。3 つ目の呼び手は `sendTypedMarkerLineBox`(`send_screen.go:379`)
  - 483 行目 (8)「`sendTypedInputBox` の次の変更: `TestSendInputHolds_NumberedMarkerLineRefuses` に行を足す」と (9)「`sendCodexHasFooter` の次の変更: codex の画面の行を `TestSendInputHolds_NumberedStepOnTheMarkerLine` に足し、変異を走らせる」: どちらも行は足していない
  - 484 行目「どちらかのファイルの次の変更: 先に番号付きの行のテストを移す」: `send_screen_test.go` は 831 行から 859 行(+28、既存の表の行だけ)。番号付きの行のテストは移していない。新しいテストは `send_screen_marker_test.go`(337 行)と `send_fallback_marker_test.go`(53 行)に置いた。`send_fallback_test.go` は 835 行のまま変わっていない
  - 460 行目「codex の画面の次の probe」: この PR の probe で codex 0.166.0 の画面を取り直した。footer は #308 と同じ形(`Context 0% used · main · No chan…` と `⚠ 11 · f2`)で、`% left` の形は出なかったので、払うものはない。注だけ
  - INFO: 365 行目「leader template tests の次の変更」は、新しいファイル(`prompts_send_unsubmitted_test.go`)の leader のテストを含むかが読み方による。行が数える 5 ファイルはどれも変わっていない。473 行目 (b) の Trigger は `sendNumberedStepsMsg` を実機に送ることを挙げる。この PR の probe は別の番号付きの TASK を codex に送ったが、番号付きの行は入力欄の頭に来なかった(`  › 2. explain this step`)。合成の 2 fixture はそのままで、条件 5 の受け入れる側のテストもその 2 つを編集した画面に頼る
- V4(LOW): 482 行目 (L11) の Debt の欄は「both callers, `findResumeInputBox` and `sendTypedInputBox`, pass the result straight to `resumeInputBoxShape`」と書くが、呼び手は 3 つになった。3 つ目の `sendTypedMarkerLineBox` は半分だけ埋まった箱から `marker` と `first` だけを読み、`resumeInputBoxShape` には新しく作った箱を渡すので、`text` に半端な箱が届かないという主張は今も成り立つ。数だけが古い。483 行目 (9) は「`if box.footerStart < len(lines) { return true }` を外す変異を走らせる」と書くが、その行は S3 の書き換えでなくなり、`sendCodexFooterStart` の `return box.footerStart, box.footerStart < len(lines)` になった(`sendCodexFooterInInput` を先に見る順に変わった)。変異の指し先を書き直す
- V5(INFO): `send_screen.go` 先頭の comment の「claude shows a long piece of it as a paste placeholder (`[Pasted text #1 +17 lines]`; evidence section 10)」は、#319 より前の素の打ち方の画面の説明。この PR の証拠では、bracketed paste で打った claude は 67 バイト・改行 3 つの短い文でも placeholder に畳んだ。この PR はこの段落に #322 の 1 文を足したので、ついでに「改行が 3 つ以上の文は畳まれる」と足すかを `/sync-docs` が決めてよい。規則の動きには関係しない

## Observational checks

- probe の画面(`/private/tmp/claude-502/…/scratchpad/probe322/out/` の 43 ファイル)を読み、fixture 11 との `cmp` を取った。probe はやり直していない
- `claude-dialog-below-old-input.txt` を `cat -vet` で読み、新しい道が条件 3 で外れることを手で追った
- self-review の M1 の移動(`0c41fbf8`)の前後で、4 つのテストファイルの `func Test` の集合を比べた(同じ 33 本)
- tech-debt の Trigger の欄を python で切り出し、この PR が触れたファイル名、関数名、skill の節の名前と照らした(V3 と V4)

## Coverage gaps

- テストは回していない(指示と skill の範囲)。AC1〜AC5 はコードとテストの assertion を読んで判定した。implementer は slice の確かめのコマンドが exit 0 だったと報告している(S1 の `internal/org` 62.994s、S2 160.821s、S3 278.151s)が、ここでは再現していない。通るかは `/test` で見る
- 計画の進捗にある、`sendTypedMarkerLineBox` の `reason != ""` を外す変異は生き残る。claude では `resumeInputBoxShape` が失敗すると空の箱が返り、`inputEnd(0) <= last.marker` が同じ結果にし、codex では `resumeInputBoxShape` が失敗しないので、等価な変異と読んだ(self-review と同じ結論。走らせてはいない)
- probe の生の画面は scratchpad にだけあり、repo には入っていない。あとから fixture が写しかを確かめ直すには、証拠に fixture 11 の sha256 を並べるのが最小の足し方(このあとの `/sync-docs` で足せる)
- Claude Code 2.1.316 と codex-cli 0.166.0 のほかの版、claude の返事のあとの画面は確かめていない(V2 の (a)(b))
- 確信を最も上げる次の確かめは、`/test` での `go test ./internal/org/ -run 'SendInputHolds|SendScreenSubmitted|SendMarkerProbe|ClassifyResumeScreen_Fixtures|ClassifySendScreen|UsageLimit|OrgSendScreen_MessageWithAMarkerLine|SendTextTypedNotSubmitted' -count=1` と、D2 の条件を 1 つずつ外す変異の再実行

## Verdict

- Verified: AC1〜AC6(AC4b を含む)の本体とテストの対応(file:line とテスト名)、D2 の 5 つの条件のコードとテストの対応、fixture 11 のバイト一致、AC4 が変えないとした 4 つの関数のコードと既存のテストの期待が変わっていないこと、skill の 4 写しの一致、prompt と skill の文面とエラーの文・note・既存の許可の一致、`run-static-verify.sh` の全項目、計画の承認の digest
- Partially verified: AC4 の「既存のテストは期待を変えずに通る」のうち、通ること(`/test`)
- Not verified: テストの実行結果。確かめた版のほかの CLI。claude の返事のあとの画面
- Verdict: pass。受け入れ条件はすべて満たし、静的検査はすべて通った。文書の指摘は V1(LOW、spec の改訂の節)、V2(MEDIUM、計画が挙げた Known gaps の行がない)、V3(LOW、Trigger を満たした tech-debt の 9 行)、V4(LOW、482 行目と 483 行目の古くなった文)、V5(INFO)。どれも `/sync-docs` で直し、V2 は `/pr` の前に済ませる
