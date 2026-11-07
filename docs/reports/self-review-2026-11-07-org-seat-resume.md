# Self-review report: org-seat-resume

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(spec `docs/specs/2026-11-07-org-seat-resume.md`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 2 の再実行で、cycle 2 の self-review(`9ade1af6`)のあとの直し R3 `3714f6d5` を受けたもの。上限 2 の最後の回で、cycle は増やしていない
- Scope: `git diff 11239c8d...HEAD` の全体(base main 11239c8d、HEAD fc7efaa8、91 ファイル、+7963 / -191)。重点は R3 `3714f6d5`(`codexFooterStart` と `resumeCodexFooterLine`、`Send` のロックの中の待ちのエラーの文と期限のヒント、`capMSToContext` の doc、テスト)と、計画の D8 を直した `fc7efaa8`。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。

## Evidence reviewed

- R3 のコード: `internal/org/resume_screens.go`(画面の型の doc の手順 2、`resumeCodexFooterLine`、`codexFooterStart`、`empty`、`text`、`ClassifyResumeScreen` の footer のまとまりの読み方)は、ファイル全体を読み直した。`internal/org/verbs.go` は `Send` の doc と本体(290〜530 行)、`sendWaitDeadlineHint`、`confirmSubmitted`、`capMSToContext` を読んだ。
- R3 のテスト: `TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing` の 2 つの場合(`directorSkillSendWaitTimeoutPhrase` を含むかどうかの確かめが増えた)、`TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing` の期待する文、`resume_screens_test.go` の新しい fixture 1 つ、edge 5 つ、`TestResumeInputHolds` の 1 行。`internal/org/testdata/resume/codex-*.txt` の 7 つを読み、footer の行がどれも新しい正規表現に合うこと(`Context 0% used`、`Context 12% used`、`? for shortcuts`、`? for shortcuts ... ⚠ 11 · f2`)と、`codex-hooks-review.txt` の `enter confirm · esc skip` は選択画面の枝で先に unknown になり footer の判定まで来ないことを確かめた。
- 正規表現は机上で追った: `^` は 1 つ目の選択肢 `\? for shortcuts` だけに掛かり、`Context \d+% (?:used|left)` と `\d+% context left` は行のどこにあっても合う。edge の 5 つの画面の結果は、どれもテストの期待どおりになる。
- `capMSToContext` の呼び出し元を grep した(`verbs.go:481` と `:765` の 2 つ)。doc の「two callers」と合う。
- 古い文の残りを grep した: `was not idle/done`、`after the lock of its pane` は `internal/org/verbs.go:312` の 1 か所だけに残る(下の R3-1)。
- 証拠: `docs/evidence/org-seat-resume-probe-2026-11-07.md` の 4 節(codex の画面)と 5 節(codex のバイナリの文字列)。
- 記録: 計画の D8 の改訂と進み具合の注(`fc7efaa8`)、`docs/tech-debt/README.md`(R3 は変えていない)、`/director` skill の 4 つの写しの 1044 行。

## Cycle 2 の指摘の今の状態

| ID | 今の状態 | 根拠 |
| --- | --- | --- |
| C2-1 | 開いたまま。`/sync-docs` で扱う(計画の進み具合の注) | `docs/tech-debt/README.md:454`、`:444-447`;`.claude/skills/director/SKILL.md:1044`(ほかの 3 つの写しも 1044 行) |
| C2-2 | 直った(R3)。(a) herdr の呼び出しの失敗は「... failed; nothing was typed: 」と出る。(b) 期限切れには `sendWaitDeadlineHint` が付き、`/director` skill の打ち直しの規則に乗る。(c) `capMSToContext` の doc は 2 つの呼び出し元を書く。ただし `Send` の doc の前のほうの段落が古い文のまま残った(R3-1) | `internal/org/verbs.go:481-486`、`:349-358`、`:770-784`;`internal/org/pane_lock_test.go:203-243` |
| C2-3 | 開いたまま。R3 は触れておらず、計画の進み具合の注も `/sync-docs` に回す件に挙げていない(C2-1、C2-5、C2-6 だけ)。Go の doc comment なので、`/sync-docs` が拾う保証もない | `internal/org/pane_lock.go:17-21`;`docs/plans/active/2026-11-07-org-seat-resume.md:152` |
| C2-4 | 直った(R3)。footer とする並びは、すべての行が `resumeCodexFooterLine` に合うときだけになり、footer のない画面の下書きは入力になった。残りは R3-2 | `internal/org/resume_screens.go:251`、`:381-404`;`internal/org/testdata/resume/codex-input-draft-no-footer.txt` |
| C2-5 | 開いたまま。`/sync-docs` で扱う | 計画の D8 の R2 の改訂(`:72`)に `⎿` の取りこぼしの 1 文がない |
| C2-6 | 開いたまま。`/sync-docs` で扱う | 下の「Cycle 1 の指摘の今の状態」の L3〜L9 |

## Cycle 1 の指摘の今の状態

| ID | 今の状態 | 根拠 |
| --- | --- | --- |
| L1 | 直った(R1)。register の行は開いたまま(C2-1) | `internal/org/verbs.go:470-486`;`docs/tech-debt/README.md:454` |
| L2 | 直った(`a4f60ff9`) | `internal/org/resume.go:54-58` |
| L3 | 開いたまま、register にない。R1 が `Send` に足した同じ形は R3 で直ったが、resume の側はそのまま | `internal/org/resume.go:376-381` |
| L4 | 開いたまま、register にない | `internal/cli/org_resume.go:210-212` |
| L5 | 開いたまま、register にない | `internal/org/prompts/implementer.md:101-105` |
| L6 | 開いたまま、register にない | `internal/org/lockfile.go:154-180` |
| L7 | 開いたまま、register にない | `internal/cli/org_resume.go:34`、`:230`;`internal/org/resume.go:83` |
| L8 | 開いたまま、register にない | `internal/org/director_cmd.go:327` |
| L9 | 開いたまま(この PR より前の helper)、register にない | `internal/org/inbox.go:180-194` |
| L10 | register の新しい行に移った(R2 は止まった文だけを直した) | `docs/tech-debt/README.md:448` |

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-1(前回のまま)。R1 と R2 のあとの register と skill が、直した挙動を古いまま書いている。(a) cycle 1 で足した `Send` の行(L1)は、R1 がその「Trigger to pay down」のとおりに直したのに開いたままで、「`Send` は手番を始めた座席に打つ」と言い続ける。(b) `/director` skill の 4 つの写しは「2 本目は 1 本目で手番を始めた leader に打つ」と書く。R1 のあとは、2 本目はロックの中で待ち直す。(c) 止まった文の行の解消の注は、返事の先頭が一覧の文で始まるときにまだ止まったと判定することを書いていない。(d) 解消にした 2 行の Related の欄は triage の `(ACTION_REQUIRED #3)`、`(ACTION_REQUIRED #4)` を cycle なしで指す | `docs/tech-debt/README.md:454`、`:444-447`;`.claude/skills/director/SKILL.md:1041-1044`(ほかの 3 つの写しも同じ行);`internal/org/resume_screens.go:537-558`(`findResumeStop`) | 前回の Recommendation のとおり。(a) 454 行を `<!-- RESOLVED ... -->` と取り消し線で閉じ、R1 `0aa8e274` と R3 `3714f6d5` を書く。(b) skill の文を「2 本目はロックを取ったあと idle/done を待ち直すので、1 本目で始まった手番には打たない」に直す。(c) 解消の注に残る場合を足す。(d) Related を「cycle 1 の ACTION_REQUIRED #3(commit dd008093)」の形にする。どれも `/sync-docs` の回で済む |
| LOW | readability | C2-3(前回のまま、扱いが決まっていない)。`pane_lock.go` の doc が R1 の効き目を強く言いすぎている。「a Send that comes after a Resume or another Send types nothing into a turn that one started or an approval dialog it brought up, and refuses instead」は 2 点で正しくない。1 つ目に、`Send` は前の持ち主が始めた手番が終わるのを残りの期限まで待ち、終われば打つ。2 つ目に、前の持ち主は `confirmSubmitted` のあとにロックを外す。その確認が working/blocked を見ずに終わると(短い手番、herdr の報告の遅れ)、herdr はまだ idle/done と報告し、ロックの中の待ちはすぐ通る。R3 はこの doc に触れず、計画の注は C2-3 をどこにも回していない。上限 2 の最後の回なので、直さなければ merge で消える | `internal/org/pane_lock.go:17-21`;`internal/org/verbs.go:298-313`(`Send` の doc);`:735-763`(`confirmSubmitted` の doc);`docs/plans/active/2026-11-07-org-seat-resume.md:152` | `pane_lock.go` の文を「ロックを取ったあと残りの期限まで idle/done を待ち直し、前の持ち主が始めた手番が終わらなければ何も打たずに断る。前の持ち主の submit が確かめられないまま外れたロックでは、herdr がまだ idle/done と報告し、待ち直しはすぐ通りうる」に直す。`Send` の doc にも後半の 1 文を足す。直さないなら下の Tech debt identified の 2 行目に入れる。計画の注にも C2-3 の扱いを書く |
| LOW | maintainability | C2-5(前回のまま)。R2 は edge「only the first ⎿ line under a prompt starts a block, the next ones join it」の期待値を Stopped から Unknown に変えた。prompt の下の最初の `⎿` の行がほかの行のとき、その下の上限の行は続ける対象にならない。打たない側に倒れるが、計画の D8 の R2 の改訂と、register の利用上限の行は、この取りこぼしを書いていない(`.claude/rules/ralph/testing.md` は計画と review の報告に書くよう求める) | `internal/org/resume_screens_test.go:296-303`;計画の D8 の 2 つ目の改訂(`:72`);`docs/tech-debt/README.md:451` | D8 の R2 の改訂に「prompt の下の最初の `⎿` の行でない上限の行は照合しない(打たない側)」の 1 文を足し、利用上限の行の Impact にも同じことを書く。この報告にはここで書いた |
| LOW | maintainability | C2-6(前回のまま)。cycle 1 の L3〜L9 の 7 件は、HEAD でも直っておらず、register のどの行にもない。cycle 1 の報告は、直さないときに行を足すのを L1 と L10 にだけ求めた。上限 2 の最後の回で、`/pr` が計画を archive すると、ほかにこれらを持つものがなくなる | 上の「Cycle 1 の指摘の今の状態」の L3〜L9 | 7 件を 1 行にまとめて register に足す(下の Tech debt identified の 1 行目) |
| LOW | readability | R3-1(新規)。R3 は `Send` の doc の終わりに、ロックの中の待ちのエラーの新しい文を書き足したが、同じ doc の前のほうの段落は古い文のまま残った。310〜313 行は「That wait failing or running out of time is refused the same way, with an error that says the seat was not idle/done after the lock of its pane was taken」と書く。R3 のあと、どちらの失敗の文にも「was not idle/done after the lock of its pane was taken」はない(grep でこの 1 か所だけに残る)。1 つの doc comment が、同じエラーを 2 通りに書いている | `internal/org/verbs.go:310-313`(古い段落);`:349-358`(R3 が足した段落);`:481-486`(コード) | 310〜313 行の「with an error that says the seat was not idle/done after the lock of its pane was taken」を消し、「with nothing typed (its error is described at the end of this comment)」のように後ろの段落を指す形にする。または後ろの段落の中身をここへ移し、エラーの文を 1 か所だけで書く |
| LOW | null-safety | R3-2(新規、C2-4 の残り)。`resumeCodexFooterLine` は `^\? for shortcuts\|Context \d+% (?:used\|left)\|\d+% context left` で、`^` は 1 つ目の選択肢だけに掛かる。(a) 状態行の 2 つの選択肢は行のどこにあっても合うので、footer のない画面で、空の `›` の行、空行、`Context 5% used` や `90% context left` を含む下書きの行の順に並ぶと、下書きは footer になり、入力欄は空と判定される。C2-4 の害がさらに狭い形で残る。doc の「a draft below the › line with no footer under it is input」は、この形では正しくない。前回の C2-4 の Recommendation(「行に `Context` と `% used` がある」)もこの形を許していた。(b) コメントは `Context <n>% left` と `<n>% context left` を「codex-cli 0.165.0 binary strings」にあるとするが、証拠の 5 節(codex のバイナリの文字列)はこの 2 つを載せていない。edge「codex footer with the context left forms is a footer」の画面は合成である。(c) codex の release が footer の形を変えると、codex の座席の画面はどれも「input box has text」で unknown になる。打たない側に倒れ、doc の手順 2 はこれを書いているが、理由の文は入力欄を指し、footer の形が原因であることは画面の結果からは分からない | `internal/org/resume_screens.go:245-251`(`resumeCodexFooterLine` とその doc);`:44-55`(画面の型の手順 2);`:129`(`resumeReasonInputHasText`);`internal/org/resume_screens_test.go:412-415`;`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 5 節 | (a) 状態行の選択肢を `(?:^\|· )Context \d+% (?:used\|left)` のように区切りの後ろに限るか、doc に「状態行の文字を含む下書きの行は footer と読む」を書いて edge を 1 つ足し、今の挙動を固定する。(b) 2 つの形を次の probe で証拠の 5 節に足すか、コメントで「証拠には載せていない」と書く。(c) `resumeReasonInputHasText` の doc か register の行に「codex の footer の形が変わると、この理由で全座席が飛ばされる。そのときは `resumeCodexFooterLine` を直す」を書く |

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6(前回から残る C2-1、C2-3、C2-5、C2-6 と、R3 で生じた R3-1、R3-2)。C2-2 と C2-4 は R3 で直った。

## Positive notes

- `Send` のロックの中の待ちの失敗は、1 回目の待ちと同じく `ctx.Err()` で期限切れを見分けるようになった。期限切れの文は `sendWaitDeadlineHint` で終わり、`/director` skill の打ち直しの規則が鍵にする文を含む。テストは `directorSkillSendWaitTimeoutPhrase` を含むかどうかを 2 つの場合の両方で確かめ、herdr の失敗の側に含まれないことも固定している。herdr の失敗の文は「failed; nothing was typed」で、待ち中の手番を推測しなくなった。cycle 1 の L3 と同じ形の問題は `Send` の側では解消した。
- 失敗の 2 つの道は、どちらも `defer unlock()` のあとの return で、pane を一度も呼ばず、`Progress` は `SendProgressNothingSent` のまま、`sent` を記録しない。この点は R1 のときから変わっていない。
- `codexFooterStart` の変更は、footer を受け入れる範囲を狭める向きだけに効く。合わない並びは入力になり、入力欄に文字があるとして unknown になるので、打つ側に倒れる新しい道はない(R3-2 (a) の、状態行の文字を含む下書きの行を除く)。hint の行は行頭に限り、edge「press ? for shortcuts」がそれを固定している。
- 新しい fixture `codex-input-draft-no-footer.txt` は前回の C2-4 の画面をそのまま写しており、`TestResumeInputHolds` の 1 行が、同じ画面で下書きが入力欄の文字として読まれることを確かめている。
- `capMSToContext` の doc は呼び出し元の数を書き、grep の結果(2 か所)と合う。

## Known gaps

- `TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing` は、resume が手番を始めた場面で「failed; nothing was typed」の文を期待する。stub の `--timeout` が ctx より先に待ちを終えるためで、テストの注はそう書いている。コードの注(`verbs.go:476-480`)は、実際の herdr では手番が続く座席の待ちは ctx の期限のあとに終わり、期限のヒントが付くとする。この場面の実際の文(期限切れの文)は、`TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing` の 2 つ目の場合が別に固定している。テストの忠実さは `/test` の範囲なので、指摘にはしていない。
- 正規表現が実際の codex の footer に合うかは、証拠 4 節の 1 画面でしか確かめていない。

## Tech debt identified

この review では register を直接編集していない。コミットを報告と insight の event だけにする指示のためである。次の 2 行を `docs/tech-debt/README.md` に足すことを勧める。C2-1 は既存の行の直しなので、ここには書かない。前回の 2 行目は C2-2〜C2-4 を持っていたが、C2-2 と C2-4 は R3 で直ったので、C2-3 と R3 の 2 件に入れ替えた。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| org-seat-resume の cycle 1 の self-review の LOW のうち、直さずに merge するもの(L3〜L9)。(L3) resume のロックの中の見直しで herdr の呼び出しが失敗すると、理由が `the screen changed (...)` になる(`resumeOne`、`internal/org/resume.go`)。(L4) 文が pane に残りうる `failed`(send-text の失敗、Enter の前の確かめの失敗、Enter の失敗)に、pane の読み方の案内が出ない(`printResumeTargetNotes`、`internal/cli/org_resume.go`)。(L5) role prompt 3 つと `agent-messaging.md` は、続行の文で「始まる」行を続行の指示とし、後ろに続く文字の扱いを書いていない。(L6) `heldFileLock` の doc(`internal/org/lockfile.go`)の保持する側の一覧に pane のロックがない。(L7) 読む行数の 60 を CLI の help と案内が文字列で持ち、`org` の側は非公開の `resumeReadLines`。(L8) 起動スクリプトの `text()`(`internal/org/director_cmd.go`)は `RetryEnv` の名前を検査せず、引用符なしで書く。(L9) `PrintableInboxText`(`internal/org/inbox.go`)は Cc だけを escape し、Cf(双方向の上書き)を通す | LOW。どれも打つ道の安全には関わらない。L5 は `ralph org send --raw` を打てる者が続行の文に指示をつなげうる点、L4 は残った文を人が見落としうる点が大きい | 上限 2 の最後の回で、cross-review cycle 1 の ACTION_REQUIRED 4 件と cycle 2 の C2-2、C2-4 を優先した | `resume.go` の見直し、`org_resume.go` の結果の行、role prompt の続行の段落、`lockfile.go` の doc、`director_cmd.go` の `text()`、`PrintableInboxText` のどれかを次に変えるとき。直し方は cycle 1 の self-review の L3〜L9 の Recommendation のとおり | `docs/reports/self-review-2026-11-07-org-seat-resume.md` の git 履歴の cycle 1 版(commit 65eea701、L3〜L9);plan `2026-11-07-org-seat-resume` |
| `Send` と pane のロックの doc、codex の footer の判定(org-seat-resume の cycle 2 の self-review の C2-3、R3-1、R3-2)。(C2-3) `pane_lock.go` の doc は「打たずに断る」と強く言い、待ってから打つ場合と、前の持ち主の submit が確かめられないまま外れた場合を書いていない。(R3-1) `Send` の doc の 310〜313 行が、ロックの中の待ちのエラーを R3 の前の文で書く。(R3-2) `resumeCodexFooterLine` の状態行の選択肢は行のどこにあっても合い、footer のない画面で状態行の文字を含む下書きの行を footer と読む。`% left` の 2 つの形は証拠に載っていない。codex が footer の形を変えると、全座席が「input box has text」で飛ばされる | LOW。R3-2 の下書きの場合だけが打つ道に関わり、footer がなく、かつ下書きが状態行の文字を含むときに限る | 上限 2 の最後の回で、直すと `/self-review` からの回し直しが要る | `pane_lock.go` か `Send` の doc、`resumeCodexFooterLine` を次に変えるとき、または codex の release が footer を変えたとき。直し方は C2-3、R3-1、R3-2 の Recommendation のとおり | `docs/reports/self-review-2026-11-07-org-seat-resume.md`(cycle 2 の R3 のあとの版、C2-3、R3-1、R3-2) |

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。LOW は 6 件(C2-1、C2-3、C2-5、C2-6、R3-1、R3-2)。R3 は C2-2 と C2-4 を直し、どちらの直しも打つ道を狭める向きに効く。ロックの解放、何も打たない失敗の道、期限切れの見分け方に欠陥は見つからなかった。
- Follow-ups: C2-1、C2-5、C2-6 は記録と skill の文の直しで、`/sync-docs` の回で扱う予定(計画の進み具合の注)。C2-3 は計画の注のどこにも回されていないので、`/sync-docs` で `pane_lock.go` の doc を直すか、上の 2 行目に入れる。R3-1 は doc comment の 4 行の直しで、R3-2 と一緒に直さないなら同じ 2 行目に入れる。
