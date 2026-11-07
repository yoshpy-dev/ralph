# Self-review report: org-seat-resume

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(spec `docs/specs/2026-11-07-org-seat-resume.md`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 3。cross-review cycle 2 の ACTION_REQUIRED 3 件を直した R5 `323e94d5` と R6 `6993faca` のあとの回し直しで、上限はユーザーの方針で 3 に上がった(`.harness/state/standard-pipeline/cycle-count.json` は cycle 2、cap 3)
- Scope: R5(5 ファイル、+478 / -54)、R6(5 ファイル、+108 / -36)、計画と仕様の改訂 `636ca9cf`。前回の報告(cycle 2 after R4、`bf8c6b71`)のあとの記録の commit(`8e91fa6f` から `ed918303` まで)は、前回までの指摘の今の状態を確かめる範囲だけ読んだ。diff 全体(`git diff 11239c8d...HEAD`)は見直していない。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。

## Evidence reviewed

- R5 の `Send`(`internal/org/verbs.go`)を HEAD で頭から読んだ。承認の確かめは 3 回ある。dry run の前、1 回目の idle/done の待ちのあと(`:448-457`、ロックの前で断る)、ロックの中の待ち直しのあと(`:497-507`、予算の確かめと `typeAndSubmit` の前)。最後の確かめで断ると `SendResult{Err: err}` を返すだけで、ロックは `:476-477` の `sync.OnceFunc` と `defer unlock()` が外す。`checkSendApproval` が使うのは `ManifestStore.Read`(`internal/org/manifest.go:111`、ロックを取らない)と `checkPlanApproved` の `os.ReadFile`(`internal/org/approval.go`)だけで、外部コマンドも別のロックも使わない。ペインロックを持ったまま別のロックを待つ順序の問題は起きない。
- `Send` の doc の 2 つの段落(承認の確かめ、ペインロック)と本文のコメント 3 か所(`enterDelay`、2 回目の確かめ、ロック)が、3 回の確かめの位置について同じことを言うのを確かめた。`checkSendApproval` の doc も 3 回を挙げる。
- R5 の `internal/org/director_wake.go`。`directorWakeSender` の閉包(`:516-521`)は `reg.HerdrLaunch() && reg.PaneID != ""` のときだけ `runDirectorWakeCmdUnderPaneLock`(`:589-596`)を呼ぶ。`lockDirectorWakePane`(`:603-612`)は `typeDirectorWakeLine` と共有され、`notDone` に「nothing was typed into it」を渡す `typeDirectorWakeLine` のエラーの文は R5 の前と 1 字も変わらない。`sendDirectorWake` は `sender.send` を inbox や director のロックの外で呼ぶ。wake_cmd の道は `ValidateDirectorWakeTarget(reg.PaneID)` を通さずにロックを取るが、`paneLockName`(`internal/org/pane_lock.go`)が `[A-Za-z0-9._-]` の外の byte を `_` にし `pane-` を前に付けるので、手で直した `director.jsonl` の pane id でもロックファイルは台帳の dir の外に出ない。
- 待ちの長さ。wake はロックを `driverCallTimeout`(既定 10 s、`verbs.go:929`)まで待ち、wake_cmd も `callWithTimeout` で同じ上限で走る。resume のロックの待ちは既定 30 s(`defaultResumePaneLockWait`、`internal/org/resume.go:92`)なので、director の pane への resume は wake_cmd 1 回が持つロックを待ち切れる。
- R5 のテスト。`TestOrgSend_ApprovalCheck_AgainUnderThePaneLock` は 2 回目の `AgentWait` の中で計画を変えるので、最後の確かめが待ち直しの前にあれば TASK は送られて落ちる。確かめの有無だけでなく位置も固定している。`TestOrgSend_ApprovalCheck_PlanChangedWhileSendWaitsForThePaneLock` は `/dev/fd` で開いているロックファイルを数えて `Send` がロックを待ち始めたことを見る。`Send` がロックファイルを開くのは 2 回目までの確かめを通ったあとなので、この時点で計画を変えれば、ロックの中の確かめだけが拒否の理由になる。`internal/org` はすでに `syscall.Flock` を条件なしに使っており、`syscall.Stat_t` を使う helper で対応する OS が減ることはない。
- R6 の `findResumeStop`(`internal/org/resume_screens.go:560-586`)。印が合っても後ろに空白がなければ `glyph = head != rest` が false になり、照合しない。まとまりの分け方(`resumeScreenBlocks`)は `afterPrompt` を空行で下ろすので、`⏺ Bash(...)` の下の `⎿` の行は `⏺` のまとまりに入る。tool の出力の 1 行目が一覧の文でも、`⎿` で始まるまとまりにはならず、R6 で増えた止まったと読む道はない。
- R6 のテストと fixture 2 つ(`claude-reply-error-paragraph.txt`、`claude-tool-output-error-paragraph.txt`)。どちらも空行のあとの段落が `API Error:` で始まり、印はない。
- 記録。`docs/tech-debt/README.md` の 365、446〜460 行。`.claude/skills/director/SKILL.md:1044-1046`。計画の D8 の改訂(`:73`)と Progress(`:172`)。仕様の FR-5a の cycle 2 の改訂(`:65`)。triage の cycle 2 の 3 件。

## 前回までの指摘の今の状態

| ID | 今の状態 | 根拠 |
| --- | --- | --- |
| C2-1 | 直った(cycle 2 の sync-docs)。(a) `Send` の行は R1 `0aa8e274` と R3 `3714f6d5` を書いて閉じた。(b) skill は「2 本目はロックを取ったあと、leader が idle か done になるのをもう一度待つ」になった。(c) 止まった文の行の解消の注に残る場合がある。(d) Related は cycle と commit で指す | `docs/tech-debt/README.md:454-455`、`:446-447`;`.claude/skills/director/SKILL.md:1044-1046` |
| C2-5 | 記録で閉じた。利用上限の行の Impact と仕様の Open questions が、2 本目の `⎿` の行の取りこぼしを書く。計画の D8 には書いていないことを、その行自身が書く | `docs/tech-debt/README.md:451` |
| C2-6 | register の行に移った(L3〜L9) | `docs/tech-debt/README.md:458` |
| R3-2 (b)(c) | register の行に移った | `docs/tech-debt/README.md:459` |
| R4-1 | 直った(`64705fbc`)。画面の型の手順 2 と正規表現のコメントは「行頭に錨、context の 2 つは行の終わりにも」と書き、1 つ目の選択肢の後ろを見ないことも書く | `internal/org/resume_screens.go:46-53`、`:258-274` |
| L1、L2 | 直った(前回のまま) | `internal/org/verbs.go:479-495`;`internal/org/resume.go:54-58` |
| L3〜L9 | register の行に移った。R5 と R6 はこれらのファイルを変えていない(`git log 64705fbc..HEAD -- internal/` は R5 と R6 だけ)。L6 だけは、R5 がペインロックを持つ側を 1 つ足したので、行の文が 1 つ足りなくなった(下の C3-1 (c)) | `docs/tech-debt/README.md:458` |
| L10 | register の行のまま | `docs/tech-debt/README.md:448` |

R5 と R6 で悪くなった前回の指摘は、L6 の数え上げに持ち手が 1 つ増えたことだけである。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C3-1(新規)。R5 と R6 が直したことで、register の 4 行が今のコードと合わなくなった。(a) 460 行(cycle 2 の `/test` の穴)の (c) C25 は「codex の `•` が `StopGlyphs` にあることを固定するテストがない」と書くが、R6 は `•` を外し、edge「a codex item glyph before a usage limit is not a stop」で外したことを固定した。(c) は意味を失った。同じ行の Trigger は `typeDirectorWakeLine` と `StopGlyphs` の次の変更を挙げ、R5 は前者を変えた(ロックを `lockDirectorWakePane` に移した)が、(a) の待ちの長さ(W2)も (e) のロックファイルを開けない枝も払っていない。R5 の新しいテストもロックを待ちのあいだずっと持たせるので、待ちを 0 にしても通る。開けない枝(`take the lock of pane %q, so ...`)を走らせるテストはない(テストの中の `take the lock of pane` は resume の `pane_lock_test.go:283` の 1 件だけ)。その枝は今は wake_cmd の道とも共有されている。(b) 451 行(利用上限の画面)の Impact は、実際の上限の画面を取りこぼす既知の場所を 2 本目の `⎿` の行の 1 つだけ挙げる。R6 のあと、codex の上限の文は `■` か `⚠` のあとでしか数えず、その前の印は実機で見ていない。codex が `•` か印なしで出すと、上限で止まった codex の座席はどれも unknown で飛ばされる。打たない側で、コードのコメントと仕様の FR-5a は書いているが、この行の Trigger で画面を取る人が読む Impact にはない。(c) 458 行の L6 は「`heldFileLock` の doc の持ち手の一覧にペインロックがなく、R1 からは push の wake もない」と書く。R5 で wake_cmd の実行(`runDirectorWakeCmdUnderPaneLock`)も持ち手になった。(d) 365 行は `verbs.go` を 2442 行(a42b9845 で数えた)とするが、HEAD では 2501 行ある(0aa8e274 で 2464、3714f6d5 で 2479、6ee7c1dd で 2478、323e94d5 で 2501) | `docs/tech-debt/README.md:460`、`:451`、`:458`、`:365`;`internal/org/resume_screens.go:219-227`;`internal/org/resume_screens_test.go:259`;`internal/org/director_wake.go:603-612`;`internal/org/director_wake_lock_test.go:32-41` | `/sync-docs` で直す。(a) 460 行の (c) を取り消し線で閉じ、R6 `6993faca` と edge の名前を書く。(a) と (e) には「R5 が `typeDirectorWakeLine` を変えたが払っていない。`lockDirectorWakePane` は wake_cmd の道と共有になったので、どちらのケースも両方の道に効く」と引き継ぎの注を足し、Impact の「(c) and (d) are INFO」を (d) だけにする。(b) 451 行の Impact に「R6 のあと、codex の上限の文は `■` か `⚠` のあとでしか数えない。その印は見ていない(edge『a codex item glyph before a usage limit is not a stop』と『a stop sentence with no glyph before it is not a stop』が固定する)」を足す。(c) 458 行の L6 に wake_cmd の実行を足す。(d) 365 行の数を 323e94d5 の 2501 に直す |
| LOW | readability | C3-2(新規)。R6 と R5 が足した doc comment の 2 か所が、コードより広いことを言う。(a) R6 は `StopGlyphs` を「the glyphs the CLI starts an error line with」、画面の型の手順 4 (c) で「one of the driver's error glyphs」と呼ぶ(テストのコメント、計画の D8 と仕様の FR-5a の改訂の「エラーの印」、commit の件名も同じ)。ところが同じファイルの claude の表のコメントは、`⏺` は返事のすべての始まり、`⎿` は prompt や tool 呼び出しの下の行の印だと書く。claude の 2 つは返事の印でもあるので、1 行目が一覧の文で始まる普通の返事(`⏺ API Error: Connection refused is what the dial test expects`)は R6 のあとも止まったと読まれる。この残りは register の 446 行の解消の注にだけあり、手順 4 (c) にも `StopGlyphs` の doc にもない。「エラーの印」と読んだ人は、この返事も外れると考えうる。(b) R5 の `pane_lock.go` は「A wake_cmd of any other registration (a promoted session, a headless director in a terminal launch_cmd opened) takes no lock」と書くが、コードの条件は `reg.HerdrLaunch() && reg.PaneID != ""` で、pane を記録していない herdr の headless の director もロックなしで走る(`TestWakeDirector_AWakeCmdOfARegistrationWithNoHerdrPaneTakesNoPaneLock` の 3 つ目のケース)。`directorWakeSender` の doc は「with a pane recorded」と書くが、`pane_lock.go` は書かない | `internal/org/resume_screens.go:81`、`:163-166`、`:203-207`;`internal/org/resume_screens_test.go:240`、`:258`;計画 `:73`;仕様 `:65`;`docs/tech-debt/README.md:446`;`internal/org/pane_lock.go:36-39`;`internal/org/director_wake.go:517` | コメントだけの直しで済む。(a) `StopGlyphs` の doc を「the glyphs a stopped sentence was seen right after on a real screen. claude's ⏺ and ⎿ also start every reply and every line under a prompt or a tool call, so a reply whose first words are a stopped sentence still counts as stopped」にし、手順 4 (c) の「error glyphs」を「StopGlyphs」に替えて同じ 1 文を足す。計画と仕様の「エラーの印」は、次に書き直すときに「止まった文の前に見た印」にする。(b) 括弧に「a headless director in herdr whose pane was not recorded」を足す。`/sync-docs` は前にも Go のコメントを直している(`a42b9845`、`a4f60ff9`、`64705fbc`)ので、そこで直せる。直さないなら下の Tech debt identified の行を足す |

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2(C3-1、C3-2。どちらも新規)。前回の LOW 5 件は、2 件が直り(C2-1、R4-1)、3 件が記録で閉じた(C2-5 は 451 行、C2-6 は 458 行、R3-2 (b)(c) は 459 行)。

## Positive notes

- R5 の最後の承認の確かめは、ロックの中の待ち直しのあと、予算の確かめと打つ処理の前にある。断るときは何も打たずに返り、ロックの解放は既存の defer に任せている。新しい失敗の道でロックが残ることはない。
- R5 のテストは確かめの位置まで固定している。計画を変えるのが待ち直しの最中なので、確かめを待ち直しの前に動かすと TASK が送られて赤になる。もう 1 つのテストは、`Send` がロックを待ち始めたことをロックファイルを開いた数で見るので、ロックの前の 2 回の確かめを通ったあとにだけ計画を変えられる。
- wake のロックの取り方は `lockDirectorWakePane` の 1 か所になり、エラーの文は `notDone` だけが変わる。herdr に打つ道の文は R5 の前と同じで、それを見るテストを書き換える必要がなかった。
- R6 は印のあとに空白を求め、`⏺API Error` を edge で固定した。codex の `•` を外した理由(普通の項目の印)と、上限の文の前の印を見ていないので打たない側に倒れることを、コードのコメント、計画の D8、仕様の FR-5a に書いた。止まったから unknown に変わった R2 の edge は、テストのコメントだけでなく計画と仕様にも理由がある。cycle 2 の C2-5 と同じ漏れは起きていない。
- R6 の fixture 2 つは、triage の指摘の形(返事の段落、tool の出力の段落)を 1 つずつ写しており、どちらも空行で分かれたまとまりが印なしで始まる。

## Known gaps

- テストは走らせていない(`/test` の範囲)。R5 の `/dev/fd` を数える helper が Linux の CI で期待どおりに数えるかは、机上で `/dev/fd` が `/proc/self/fd` を指すことを確かめただけである。
- codex が上限の文の前に出す印は実機で見ていない(R6 のコメントのとおり)。
- claude の普通の返事が一覧の文で始まることがどれほどあるかは測っていない。C3-2 (a) の返事が実際に出るかは未確認です。
- `TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing` が stub の都合で「failed; nothing was typed」の文を期待する点は、前回のとおり `/test` の範囲とした。

## Tech debt identified

この review では register を直接編集していない。コミットを報告と insight の event だけにする指示のためである。`/sync-docs` に次を頼む。

閉じる、または直す行:

| 行 | すること | 理由 |
| --- | --- | --- |
| 460 | (c) C25 を取り消し線で閉じ、R6 `6993faca` と edge「a codex item glyph before a usage limit is not a stop」を書く。(a) と (e) に R5 の引き継ぎの注を足し、Impact の INFO を (d) だけにする | C3-1 (a) |
| 451 | Impact に、R6 のあと codex の上限の文は `■` か `⚠` のあとでしか数えず、その印は見ていないことを足す | C3-1 (b) |
| 458 | L6 の持ち手に wake_cmd の実行(`runDirectorWakeCmdUnderPaneLock`)を足す | C3-1 (c) |
| 365 | `verbs.go` の数を 323e94d5 で数えた 2501 にする | C3-1 (d) |

C3-2 をコメントで直さないときに足す行:

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| The doc comments of `StopGlyphs` and of step 4 (c) of the screen model (`internal/org/resume_screens.go`) call claude's ⏺ and ⎿ the CLI's error glyphs, while the claude table's own comment says ⏺ starts every reply and ⎿ every line under a prompt or a tool call; a claude reply whose first words are a stopped sentence still counts as stopped, which only the RESOLVED note of the stopped-screen row says. The doc of `internal/org/pane_lock.go` lists the registrations whose wake_cmd takes no lock without a headless director in herdr whose pane was not recorded (org-seat-resume, cycle 3 self-review C3-2) | LOW. A reader can take the R6 rule to exclude such a reply. No behavior is wrong | Comment-only; the pipeline ran at its raised cap of 3 | The next change to `StopGlyphs`, `findResumeStop`, the screen model doc, or `pane_lock.go`: reword as C3-2 recommends | `docs/reports/self-review-2026-11-07-org-seat-resume.md`(cycle 3、C3-2);plan `2026-11-07-org-seat-resume`(D8) |

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。LOW は 2 件(C3-1、C3-2)。R5 は承認の確かめを打つ直前に移し、herdr の director の wake_cmd をペインロックに入れた。どちらも失敗の道でロックを外し、何も打たない。R6 は止まった文を印と空白のあとに限り、変化は打たない側にだけ効く。
- Follow-ups: `/sync-docs` に C3-1 の 4 行(460、451、458、365)と C3-2 のコメントの直しを渡す。C3-2 を直さないなら上の行を足す。計画の Progress に cycle 3 の注を書くときは、`/sync-docs` に回すもの(C3-1 (a)〜(d)、C3-2)を ID で並べる。
