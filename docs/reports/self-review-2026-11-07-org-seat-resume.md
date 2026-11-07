# Self-review report: org-seat-resume

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(spec `docs/specs/2026-11-07-org-seat-resume.md`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 2(cross-review cycle 1 の直しのあとの回し直し。上限 2 の最後の回)
- Scope: `git diff 11239c8d...HEAD` の全体(base main 11239c8d、HEAD 95f45240、90 ファイル、+7853 / -182)。重点は cycle 1 の直しの 3 コミット: R1 `0aa8e274`(`Send` がロックの中で idle/done を待ち直す、director の push の wake がロックを取る)、R2 `306ee58f`(入力欄の全行が空、codex の入力欄と footer、止まった文の先頭一致)、`95f45240`(D8 と FR-5a の改訂)。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。

## Evidence reviewed

- R1 のコード: `internal/org/verbs.go`(`Send` のロックの中の `AgentWait`、`SendProgressNothingSent` と `Send` の doc、`capMSToContext`、`sendWaitDeadlineHint`、`typeAndSubmit`、`confirmSubmitted`)、`internal/org/director_wake.go`(`typeDirectorWakeLine`、`sendDirectorWake` の失敗の道、`directorWakeSender`)、`internal/org/pane_lock.go`、`internal/org/resume.go`(`defaultResumePaneLockWait`、`Resume` の doc の手順 4、`resumeOne` のロックの中の見直し、`resumeSubmitBudget`)。
- R1 のテスト: `TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing`、`TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing`(`turnWaitHerdr`)、`TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping`、`verbs_test.go` の `AgentWait` の回数と順番の更新、まだ `Send` の成功を期待する `TestOrgResume_SendWaitsForAResumeHoldingThePaneLock`。
- R2 のコード: `internal/org/resume_screens.go`(画面の型の doc の手順 2〜4、`resumeSentence` と `StopGlyphs` の doc、`findResumeInputBox` の codex の枝、`codexFooterStart`、`resumeInputBox.empty`、`isCodexComposerPlaceholder`、`findResumeStop`、`findResumeSentence`、`ResumeInputHolds`)。
- R2 のテスト: `resume_screens_test.go` の新しい 3 つの fixture と 9 つの edge、期待値を Stopped から Unknown に変えた 2 つの edge。`internal/org/testdata/resume/codex-*.txt` の 5 つを `cat -n` で読み、どれも `›` の行の下に空行を挟んだ footer があることを確かめた。
- 記録: `docs/tech-debt/README.md` の diff(R1 と R2 が解消にした 2 行、新しい enter-only の行、cycle 1 で足した `Send` の行)、`docs/reports/cross-review-triage-org-seat-resume.md`、計画の D8 の改訂 2 つ、仕様の FR-5a の改訂。
- 突き合わせに読んだもの: `internal/cli/org.go` の `ralph org send --help`(1324〜1349 行)、`.claude/skills/director/SKILL.md` の `### 答える順`(1028〜1052 行、ほかの 3 つの写しも同じ行)、`internal/org/lockfile.go` の `heldFileLock` の doc、`internal/cli/org_resume.go` の `printResumeTargetNotes` と `resumeReadCommand`、`internal/org/prompts/implementer.md` の続行の文の段落、`internal/org/director_cmd.go` の `text()`。
- 机上で追った場合: codex の画面で、空の `›` の行、空行、下書きの段落の順に並び、footer がない場合の `codexFooterStart` と `empty`(下の C2-4)。

## Cycle 1 の指摘の今の状態

| ID | 今の状態 | 根拠 |
| --- | --- | --- |
| L1 | 直った(R1)。register の行は開いたまま(C2-1) | `internal/org/verbs.go:465-474`;`docs/tech-debt/README.md:454` |
| L2 | 直った(`a4f60ff9`) | `internal/org/resume.go:54-58` |
| L3 | 開いたまま、register にない。R1 が同じ形を `Send` に足した(C2-2) | `internal/org/resume.go:376-381` |
| L4 | 開いたまま、register にない | `internal/cli/org_resume.go:210-212` |
| L5 | 開いたまま、register にない | `internal/org/prompts/implementer.md:101-105` |
| L6 | 開いたまま、register にない | `internal/org/lockfile.go:154-180` |
| L7 | 開いたまま、register にない | `internal/cli/org_resume.go:34`、`:230`;`internal/org/resume.go:83` |
| L8 | 開いたまま、register にない | `internal/org/director_cmd.go:327` |
| L9 | 開いたまま(この PR より前の helper)、register にない | `internal/org/inbox.go:180-194` |
| L10 | register の新しい行に移った(R2 は止まった文だけを直した) | `docs/tech-debt/README.md` の enter-only の行 |

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-1. R1 と R2 のあとの register と skill が、直した挙動を古いまま書いている。(a) cycle 1 で足した `Send` の行(L1)は、R1 がその「Trigger to pay down」のとおりに直したのに開いたままで、「`Send` は手番を始めた座席に打つ」「`Send` と pane のロックの doc は書いていない」と言い続ける。R1 の fix commit が解消にしたのは wake の行だけだった。(b) `/director` skill の 4 つの写しは「2 本目の idle の待ちが 1 本目の Enter より前に終わっていると、2 本目は 1 本目で手番を始めた leader に打つ」と書く。R1 のあとは、2 本目はロックの中で待ち直す。この文は cycle 1 でこの PR が書いた。(c) 止まった文の行は全体に取り消し線を引いたが、返事の先頭が一覧の文で始まるとき(`⏺ API Error: 401 ...` と始まる返事)はまだ止まったと判定する。解消の注は「after other text」とだけ書き、残る場合を書いていない。(d) 解消にした 2 行の Related の欄は `cross-review-triage-org-seat-resume.md` の `(ACTION_REQUIRED #3)`、`(ACTION_REQUIRED #4)` を cycle なしで指す。cycle 2 の `/cross-review` はこの file を書き直すので、番号は別の指摘を指しうる | `docs/tech-debt/README.md:454`(L1 の行);`:444-447`(解消にした 2 行);`.claude/skills/director/SKILL.md:1041-1044`(`.agents/skills/director/SKILL.md`、`templates/base/` の 2 つも 1044 行);`internal/org/resume_screens.go:520-541`(`findResumeStop`、関数は 524 行から) | (a) 454 行を `<!-- RESOLVED ... -->` と取り消し線で閉じ、R1 の `0aa8e274` と `TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing` を書く。前の持ち主の submit が確かめられずに(herdr がまだ idle/done と報告して)ロックが外れたときは、待ち直しがすぐ通ることを残りとして書く。(b) skill の文を「2 本目はロックを取ったあと idle/done を待ち直すので、1 本目で始まった手番には打たない」に直す(`/sync-docs` の回でもよい)。(c) 解消の注に「返事が一覧の文で始まるときは止まったと判定する」を足す。(d) Related の欄を「cycle 1 の ACTION_REQUIRED #3(commit dd008093)」の形にする |
| LOW | exception-handling | C2-2. `Send` のロックの中の待ちの失敗の文が、何が起きたかと合わない。(a) herdr の呼び出しが失敗した(時間切れでない)ときも「seat was not idle/done after the lock of its pane was taken (another send or resume may have started a turn in it)」と出る。cycle 1 の L3(resume の「the screen changed」)と同じ形で、`TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing` の「the wait fails」がこの文を固定している。(b) この待ちが期限切れで終わっても `sendWaitDeadlineHint` が付かない。`ralph org send --help` は「When the wait runs out, nothing is typed and the error says the seat's turn did not end and how to send again」と書き、`/director` skill は `the seat's turn did not end within the --timeout-ms deadline` を含むときに「resolve せず、あとで打ち直す」とする(`director_skill_sync_test.go` がこの文を固定)。ロックの中の期限切れは、この規則の外の「ほかの失敗」になる。(c) `capMSToContext` の doc は、使う側を確認の待ちだけとして書く(「a confirmation wait」「a confirmation call」「Wait, verbs.go's other AgentWait caller」)。R1 で確認でない呼び出し元が増えた | `internal/org/verbs.go:472-474`(`:472` が `capMSToContext` の新しい呼び出し);`:426-436`(1 回目の待ちのヒント);`:758-769`(`capMSToContext` の doc);`internal/cli/org.go:1335-1340`;`.claude/skills/director/SKILL.md:1047-1050`;`internal/org/pane_lock_test.go:206-221` | 2 回目の待ちの失敗を 1 回目と同じ形にする: `errors.Is(ctx.Err(), context.DeadlineExceeded)` のときは `sendWaitDeadlineHint(timeoutMS)` を足し、「(another send or resume may have started a turn in it)」はそのときだけ付ける。文の頭は「wait for seat %q idle/done again under the lock of its pane (%s)」のように、何をしていたかを言う形にする。`capMSToContext` の doc は「確認の待ちと、ロックの中の idle/done の待ち」に直す |
| LOW | readability | C2-3. `pane_lock.go` の doc が R1 の効き目を強く言いすぎている。「a Send that comes after a Resume or another Send types nothing into a turn that one started or an approval dialog it brought up, and refuses instead」は 2 点で正しくない。1 つ目に、`Send` は前の持ち主が始めた手番が終わるのを残りの期限まで待ち、終われば打つ。断るのは終わらないときだけである。2 つ目に、前の持ち主は `confirmSubmitted` のあとにロックを外す。その確認が working/blocked を見ずに終わると(短い手番、herdr の報告の遅れ。`confirmSubmitted` の doc が書いている 2 つ)、herdr はまだ idle/done と報告し、ロックの中の待ちはすぐ通る。`Send` の doc(verbs.go)は 1 つ目を正しく書き、2 つ目は書いていない | `internal/org/pane_lock.go:17-21`;`internal/org/verbs.go:298-313`(`Send` の doc);`:723-751`(`confirmSubmitted` の doc);`:720`(確認の結果を返すだけで、ロックは呼び出し側が外す) | `pane_lock.go` の文を「ロックを取ったあと残りの期限まで idle/done を待ち直し、前の持ち主が始めた手番が終わらなければ何も打たずに断る。前の持ち主の submit が確かめられないまま外れたロックでは、herdr がまだ idle/done と報告し、待ち直しはすぐ通りうる」に直す。`Send` の doc にも後半の 1 文を足す |
| LOW | null-safety | C2-4. `codexFooterStart` は、空行を挟んだ最後の行の並びを、footer らしいかどうかを見ずに footer とする。footer が画面にないと、空の `›` の行、空行、下書きの段落の順の画面で、下書きが footer になり、入力欄は空と判定される。上に止まった文があれば resume は続行の文を打って Enter を押し、下書きも一緒に送る。cross-review の ACTION_REQUIRED #2 の害が、より狭い形で残っている。doc の「so a draft line below the › line is input and never footer」は、この形では正しくない。codex の 5 つの fixture と `codexResumeScreen` は、どれも footer を持つ。記録の画面ではいつも footer が出ているので、起こる見込みは低い | `internal/org/resume_screens.go:43-51`(doc);`:370-388`(`codexFooterStart`);`:390-403`(`empty`);`internal/org/resume_screens_test.go:36-38`;`internal/org/testdata/resume/codex-*.txt` | footer とする並びに footer の形を求める(行に `Context` と `% used`、または `for shortcuts` がある)。合わない並びは入力とし、判定を unknown に倒す。そうしないなら、doc の「never」を外して「codex がいつも footer を出す」という前提を書き、footer のない画面の edge を 1 つ足して今の挙動を固定する |
| LOW | maintainability | C2-5. R2 は edge「only the first ⎿ line under a prompt starts a block, the next ones join it」の期待値を Stopped から Unknown に変えた。prompt の下の最初の `⎿` の行がほかの行(@ で付けたファイルの `⎿  Read notes.md (3 lines)` など)のとき、その下の `⎿  You've hit your session limit` は、もう続ける対象にならない。打たない側に倒れるので害は小さく、テストの注が仕組みを説明している。ただ、計画の D8 の R2 の改訂と、register の利用上限の行は、この取りこぼしを書いていない。`.claude/rules/ralph/testing.md` は、テストの意図を弱めたときは計画と review の報告に書くよう求める | `internal/org/resume_screens_test.go:290-296`;`docs/plans/active/2026-11-07-org-seat-resume.md` の D8 の 2 つ目の改訂;`docs/tech-debt/README.md` の利用上限の行 | D8 の R2 の改訂に「prompt の下の最初の `⎿` の行でない上限の行は照合しない(打たない側)」の 1 文を足し、利用上限の行の Impact にも同じことを書く。この報告にはここで書いた |
| LOW | maintainability | C2-6. cycle 1 の L3〜L9 の 7 件は、HEAD でも直っておらず、register のどの行にもない。cycle 1 の報告は、直さないときに行を足すのを L1 と L10 にだけ求めた。今回は上限 2 の最後の回で、`/pr` が計画を archive すると、ほかにこれらを持つものがなくなる。直していない L3 と同じ形(herdr の失敗を状態の文で包む)を、R1 が `Send` に足した(C2-2) | 上の「Cycle 1 の指摘の今の状態」の表の L3〜L9 の行 | 7 件を 1 行にまとめて register に足す(下の Tech debt identified の 1 行目)。L3 は C2-2 と一緒に直すなら、その行から外す |

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6(C2-1〜C2-6)。

## Positive notes

- `Send` のロックの中の待ちは、`unlock` を `sync.OnceFunc` と `defer` で包んだあとに置かれ、失敗の return でもロックが外れる。この道は pane を一度も呼ばず、`Progress` は `SendProgressNothingSent` のまま、`sent` も記録しない。期限の確かめ(budget check)はこの待ちのあとなので、2 つの待ちが使った時間を引いた残りで測る。`AgentWait` の時間は `capMSToContext` で残りの期限に切り詰め、1 ms より短くならないので、herdr の `--timeout` が省かれて無期限になることはない。
- wake のロックは、pane の確かめ(`confirmSeatPane`)のあと、`send-text` の前に取り、`defer` で外す。取れなかったときは何も打たず、`sendDirectorWake` の既存の失敗の道(`mem.recordFailure`、banner の 1 行、`woken` は書かない)にそのまま乗る。`TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping` は、取れないとき、取り直しを待つとき(held back)、外れたあとの 2 回の wake を確かめる。resume の director の対象は登録の `PaneID` を使うので(`internal/org/resume.go:309`)、wake と同じロックの file になる。
- `resumeInputBox.empty` は、2 つの driver を 1 つの道にまとめた。placeholder は codex の印の行でだけ許し、2 行目の placeholder は文字として扱う(edge「codex placeholder on the line under an empty › line is text」)。
- `findResumeStop` は、空のまとまりと prompt のまとまりを先に外し、driver ごとの印を 1 つだけ落とす。ほかの driver の印は落とさない(edge「a glyph of the other driver is not dropped」)。待ち中と Enter だけの文は部分一致のままで、どちらも文字を打たない側に倒れる。
- R1 と R2 のコメントは、指摘を「cross-review cycle 1 #N of org-seat-resume」と cycle 付きで書いており、repo の既存の書き方(`approval_enforce_test.go:312` など)と合う。

## Tech debt identified

この review では register を直接編集していない。コミットを報告と insight の event だけにする指示のためである。次の 2 行を `docs/tech-debt/README.md` に足すことを勧める。C2-1 は既存の行の直しなので、ここには書かない。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| org-seat-resume の cycle 1 の self-review の LOW のうち、直さずに merge するもの(L3〜L9)。(L3) resume のロックの中の見直しで herdr の呼び出しが失敗すると、理由が `the screen changed (...)` になる(`resumeOne`、`internal/org/resume.go`)。(L4) 文が pane に残りうる `failed`(send-text の失敗、Enter の前の確かめの失敗、Enter の失敗)に、pane の読み方の案内が出ない(`printResumeTargetNotes`、`internal/cli/org_resume.go`)。(L5) role prompt 3 つと `agent-messaging.md` は、続行の文で「始まる」行を続行の指示とし、後ろに続く文字の扱いを書いていない。(L6) `heldFileLock` の doc(`internal/org/lockfile.go`)の保持する側の一覧に pane のロックがない。(L7) 読む行数の 60 を CLI の help と案内が文字列で持ち、`org` の側は非公開の `resumeReadLines`。(L8) 起動スクリプトの `text()`(`internal/org/director_cmd.go`)は `RetryEnv` の名前を検査せず、引用符なしで書く。(L9) `PrintableInboxText`(`internal/org/inbox.go`)は Cc だけを escape し、Cf(双方向の上書き)を通す | LOW。どれも打つ道の安全には関わらない。L5 は `ralph org send --raw` を打てる者が続行の文に指示をつなげうる点、L4 は残った文を人が見落としうる点が大きい | 上限 2 の最後の回で、cross-review cycle 1 の ACTION_REQUIRED 4 件を優先した | `resume.go` の見直し、`org_resume.go` の結果の行、role prompt の続行の段落、`lockfile.go` の doc、`director_cmd.go` の `text()`、`PrintableInboxText` のどれかを次に変えるとき。直し方は cycle 1 の self-review の L3〜L9 の Recommendation のとおり | `docs/reports/self-review-2026-11-07-org-seat-resume.md` の git 履歴の cycle 1 版(commit 65eea701、L3〜L9);plan `2026-11-07-org-seat-resume` |
| `Send` のロックの中の idle/done の待ちと、`codexFooterStart` の footer の前提(org-seat-resume の cycle 2 の self-review の C2-2〜C2-4)。(C2-2) その待ちの herdr の失敗が「was not idle/done ... another send or resume may have started a turn」と出る。期限切れには `sendWaitDeadlineHint` が付かず、`ralph org send --help` と `/director` skill の打ち直しの規則から外れる。(C2-3) `pane_lock.go` の doc は「打たずに断る」と強く言い、待ってから打つ場合と、前の持ち主の submit が確かめられないまま外れた場合を書いていない。(C2-4) footer のない codex の画面では、空行のあとの下書きを footer と読み、入力欄を空と判定する | LOW。C2-4 だけが打つ道に関わる。記録の画面はどれも footer を持つので見込みは低い | 上限 2 の最後の回で、直すと `/self-review` からの回し直しが要る | `Send` の待ちか `codexFooterStart` を次に変えるとき、または codex の footer のない画面の記録。直し方は C2-2〜C2-4 の Recommendation のとおり | `docs/reports/self-review-2026-11-07-org-seat-resume.md`(cycle 2、C2-2〜C2-4) |

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。LOW は 6 件(C2-1〜C2-6)。cross-review cycle 1 の 4 件の直しは、どれも打つ道を狭める方向に効いている。ロックの解放、何も打たない失敗の道、wake の失敗の記録に欠陥は見つからなかった。
- Follow-ups: C2-1 と C2-5 は記録と skill の文の直しで、`/sync-docs` の回か record-only の直しでも済む(C2-1 (b) の skill は `/sync-docs`)。C2-2〜C2-4 と cycle 1 の L3〜L9 は Go のコードの直しで、この回では直さないなら上の 2 行を register に足す(C2-6)。
