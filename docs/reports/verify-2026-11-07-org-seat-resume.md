# Verify report: org の座席を API エラーから続ける(`ralph org resume` と再試行の設定)

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(spec `docs/specs/2026-11-07-org-seat-resume.md`)
- Verifier: verifier サブエージェント(`/verify`)。パイプラインの cycle 3。上限はユーザーの方針で 3 に上がり、`.harness/state/standard-pipeline/cycle-count.json` は cycle 2・cap 3 のままなので、insight の event は lead の指示で `--cycle 3` を渡した。cycle 2 の報告(commit `8e91fa6f`)をこの版で上書きした。cycle 1 の版は `39db8f25`、cycle 2 の版は `8e91fa6f` で読める。tech-debt の register にこの報告を引く行はない(上書きで指す先が変わる行はない)
- Scope: `git diff 11239c8d...HEAD`(base main `11239c8d` = origin/main、HEAD `5edbb28b`、95 ファイル)。重点は cross-review cycle 2 の ACTION_REQUIRED 3 件の直し R5 `323e94d5`・R6 `6993faca` と、計画と仕様の改訂 `636ca9cf`。それより前の FR と AC は、R5・R6 がその道を変えていないことを diff で確かめ、cycle 2 の判断を引き継いだ。仕様適合、静的解析、文書の drift を見た。テストは走らせていない(/test の担当)
- Evidence: `docs/evidence/verify-2026-11-07-org-seat-resume.log`(git の対象外。末尾の「Cycle 3」の区切りの下に、`run-static-verify.sh` の出力、digest、試しの merge、tech-debt の Trigger の走査、行数、row 225・226・456 の確かめを足した。wrapper 自身のログは `docs/evidence/verify-2026-11-07-042055.log`)

## Spec compliance

計画の AC は仕様の AC をそのまま使う。テストの欄は、テストがあり assert が AC を言っていることを読んで確かめたもので、結果は /test が出す。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 2 #1: per-plan の implementer への TASK は、ロックの中の待ち直しのあと、打つ直前にも承認を確かめる | Met | `Send`(`internal/org/verbs.go:365`)の承認の確かめは 3 回になった。dry run の前(`:383`)、1 回目の idle/done の待ちのあと(`:454`、ロックの前で断る)、ペインロック(`:469`)の中の待ち直しのあと(`:504`)。3 回目は予算の確かめと `typeAndSubmit`(`:538`)の前にあり、断ると `SendResult{Err}` を返すだけで、ロックは `defer unlock()` が外す。`isTask` のときだけ走る。テスト: `TestOrgSend_ApprovalCheck_AgainUnderThePaneLock`(`internal/org/pane_lock_approval_test.go:54`。2 回目の `AgentWait` の中で計画を変えるので、確かめが待ち直しより前にあれば赤になる。計画の本文の変更、Progress の行の追加、改訂の承認の 3 通りで断り、変更なしと箱の印だけなら送る。typed と raw の両方。herdr の呼び出しは待ち 2 回だけ、manifest と receipt は変わらず、ロックは外れる)、`TestOrgSend_ApprovalCheck_PlanChangedWhileSendWaitsForThePaneLock`(`:156`。`/dev/fd` で開いたロックファイルを数えて `Send` がロックを待ち始めたことを見てから計画を変える) |
| cross-review cycle 2 #2: 印のないまとまりは止まった文に一致しない | Met | `findResumeStop`(`internal/org/resume_screens.go:564`)は、まとまりが `StopGlyphs` の 1 つ、空白、止まった文の順で始まるときだけ一致する。印のあとに空白がなければ `glyph` が false で外れる。codex の印から `•` を外した(`:227`、claude は `:207`)。fixture `claude-reply-error-paragraph.txt` と `claude-tool-output-error-paragraph.txt`(`resume_screens_test.go:155`、`:159`、どちらも unknown)、edge 4 本(`:242` 印のない文、`:247` 空白のない印、`:259` codex の `•`、`:264` codex の `■`) |
| cross-review cycle 2 #3: herdr の headless の director に設定された `wake_cmd` もペインロックの中で走る | Met | `directorWakeSender` の閉包(`internal/org/director_wake.go:517-518`)は `reg.HerdrLaunch() && reg.PaneID != ""` のとき `runDirectorWakeCmdUnderPaneLock`(`:589`)を呼ぶ。ロックの取り方は `lockDirectorWakePane`(`:603`)で、`typeDirectorWakeLine`(`:629`)と共有する。待つのは `driverCallTimeout` まで。取れなければ wake_cmd を走らせず wake の失敗。テスト: `TestWakeDirector_AWakeCmdOfADirectorInHerdrRunsUnderItsPaneLock`(`internal/org/director_wake_lock_test.go:30`。持たれている間は走らず、banner に 1 行、`woken` なし、5 分の見送り。外れたあとは走るあいだずっとロックを持ち、`{target}` は pane、終わると外れる、herdr は呼ばない)、`TestWakeDirector_AWakeCmdOfARegistrationWithNoHerdrPaneTakesNoPaneLock`(`:100`。昇格したセッション、launch_cmd の director、pane を記録していない herdr の director はロックファイルを作らない) |
| FR-5a とその改訂 3 つ(計画助言、cross-review cycle 1、cycle 2) | Met | 上の 3 行と cycle 2 の根拠。仕様の cycle 2 の改訂(`:65`)と計画の D8 の改訂(`:73`)の文は code と合う(言い過ぎは下の V3-6 (a)) |
| FR-1〜FR-4、FR-6〜FR-9、NFR 5 つ | Met | R5・R6 は `internal/cli/org_resume.go`、対象の選び方、順、`resumed`、`api_wait_minutes` と 3 つの変数を変えていない。cycle 2 の根拠のまま。R5 は LLM を呼ぶ道を足していない。打つ道が増える変更はない(R5 は断る側に 1 つ足し、R6 は打たない側にだけ効く) |
| FR-5(resume には見張りと承認の確かめをかけない) | Met | R5 の 3 回目の確かめは `Send` の中だけで、`Resume` は `typeAndSubmit` を直接呼ぶ(変更なし) |
| FR-10、FR-11(手順書、役割 prompt、文書) | Met(LOW V3-5) | 3 項目はそろっている。R5 で増えた 2 つ(最後の承認の確かめ、wake_cmd のロック)が AGENTS.md、org runtime の仕様、`/org` と `/director` の skill に書かれていない |
| AC-1〜AC-4 | Met | R6 のあとも実機の画面は止まったと判定される。`claude-enotfound.txt` は `⏺ API Error:`、`claude-weekly-limit-under-prompt.txt` は `⎿  You've hit your weekly limit`、`claude-session-limit.txt` は `⏺ You've hit your session limit`、`codex-usage-limit.txt` は `■ You've hit your usage limit` で、どれも印、空白、文の順。R6 は fixture の期待を 1 つも変えていない(変わったのは edge の 1 本、codex の印のない文が stopped から unknown に) |
| AC-5〜AC-9、AC-11〜AC-13、AC-15 | Met | 変更なし。cycle 2 の根拠のまま |
| AC-14 | Met | cycle 2 の根拠に、wake_cmd の道のロック(上の #3)が加わった |
| AC-10 | Partially met | 下の「AC-10 の判断」。cycle 1 からの判断を引き継ぐ |
| 計画の digest | Met | `./scripts/plan-visual.sh digest` は `104ee2d3ca75` を出し、`- Approved: 2026-11-07 sha256:104ee2d3ca75` と同じ |

### AC-10 の判断

cycle 1 と cycle 2 の判断を引き継ぐ。R5・R6 はこの判断の材料を変えていない。

- 実機で確かめたこと(`docs/evidence/org-seat-resume-live-2026-11-07.md` の 1): 通信断で座席が止まり herdr が `idle` を返す。`--dry-run` は `would_resume` を出して記録しない。resume は文を打ち、Enter の前の確かめを通って Enter を 1 回押し、`resumed` を 1 件記録した。座席は要求を出し直した。動いている座席は `waiting` で打たなかった。
- 確かめていないこと: 座席が作業を最後まで進めること(アカウントの週の利用上限で `rate_limit` が返った)。役割 prompt の 1 行の効き(実機の座席は役割 `worker`)。
- 実機の binary は `b5ea321f` で、R2・R6 より前である。接続の拒否の画面の止まった行は `⏺ API Error:` で始まり、R6 の規則(印、空白、文)でも同じ判定になる。実機ではやり直していない。
- ralph の決定的な部分は実機とテストで確かめた。AC-10 の文言は満たしていないので Partially met とし、merge は止めない。仕様の AC-10 の注と tech-debt の AC-10 の行がそう書いている。
- いちばん効く追加の確かめ: 週の上限のリセットのあと(日本時間 2026-11-07 23 時)、live の記録の 1 の手順を役割 `implementer` の座席で HEAD の binary で走らせ、`done.txt` ができることを見る。役割 prompt の 1 行の効きと R6 のあとの判定も同時に見られる。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass(exit 0、40 秒) | testdata の `.txt` が言語に分類されず、Go は全体で走った(`full fallback`) |
| shellcheck と `sh -n`(hooks、verify scripts、両方の写し) | OK | |
| `scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-pipeline-sync.sh` | OK | |
| `scripts/check-skill-sync.sh` | OK | 14 skill。R5・R6 は skill を変えていない |
| `scripts/check-template-purity.sh` | PASS | |
| tech-debt の plan の参照 | OK | |
| gofmt、`go vet ./...`、golangci-lint、staticcheck | OK | gofmt ok、golangci-lint `0 issues.`、vet と staticcheck は出力なし。`Skipping` の行はなく、4 つとも走った |
| `secret-scan-branch.sh` | clean | `11239c8d..5edbb28b` |
| `git diff --check main...HEAD` | 空 | |
| `git merge-tree --write-tree --name-only main HEAD` | 衝突なし | origin/main は `git ls-remote` でも `11239c8d` |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 仕様 FR-5a の cycle 2 の改訂、計画 D8 の cycle 2 の改訂 | Yes(LOW V3-6 (a)) | 規則は code と合う。claude の `⏺`・`⎿` を「エラーの印」と呼ぶのは言い過ぎ |
| 仕様の Status | 遅れ | V3-5 (e) |
| AGENTS.md の `internal/org/` の行 | 一部 | V3-5 (a) |
| org runtime の仕様の 2026-11-07 改訂 (b) | 一部 | V3-5 (b) |
| `/org` skill(4 つの写し) | 一部 | V3-5 (c) |
| `/director` skill(4 つの写し) | 一部 | V3-5 (d)。910〜912 行の wake の失敗の原因(ロックを持ち続けた)は wake_cmd の道にも当たり、そのままで正しい。1043〜1048 行の 2 本の send の段落も正しい |
| `ralph org send --help`、`ralph org watch --help` | Yes(INFO I3-4) | 間違った文はない |
| Go の doc comment(`StopGlyphs`、画面の型の手順 4 (c)、`pane_lock.go`) | 一部 | V3-6(self-review C3-2) |
| `docs/tech-debt/README.md` | No | V3-1〜V3-4 |
| 計画の `## Progress checklist` | 遅れ | V3-7 |
| `docs/recipes/director-wake.md`(2 つの写し) | Yes(INFO I3-5) | ロックに触れないが、間違った文はない |

### 所見

- V3-1(LOW、self-review C3-1 を確かめ、1 点を直した): R5・R6 が Trigger を満たしたか、事実を変えた register の行。
  - (a) 460 行。R5 は `typeDirectorWakeLine` を変え(ロックの取り方を `lockDirectorWakePane` に移した)、R6 は `StopGlyphs` を変えたので、Trigger に 2 回当たった。(c) C25 は意味を失った(R6 が `•` を外し、edge「a codex item glyph before a usage limit is not a stop」が外したことを固定する)。(a) W2 と (e) ロックファイルを開けない枝は払っていない。R5 の `TestWakeDirector_AWakeCmdOfADirectorInHerdrRunsUnderItsPaneLock` も待ちのあいだずっとロックを持たせる(`DriverCallTimeout` 50 ms)ので、待ちを 0 にしても通る。開けない枝は `lockDirectorWakePane` にあり、今は wake_cmd の道とも共有する。self-review の勧めのとおり、(c) を閉じ、(a) と (e) に引き継ぎの注を足し、Impact の「(c) and (d) are INFO」を (d) だけにする。
  - (b) 451 行の Impact。R6 のあと、codex の上限の文は `■` か `⚠` のあとでしか数えず、その前の印は実機で見ていない。codex が `•` か印なしで出すと、上限で止まった codex の座席は unknown で飛ばされる。仕様の FR-5a の改訂とコードのコメントは書くが、この行は書かない。
  - (c) 458 行の L6。`heldFileLock` の doc の持ち手の一覧に、R5 で wake_cmd の実行(`runDirectorWakeCmdUnderPaneLock`)も加わった。
  - (d) 365 行。self-review は「行は `verbs.go` を 2442 行とする」と書くが、行の最後の注はすでに 6ee7c1dd での 2478 行を書いている。足りないのは R5 の分の引き継ぎの注である。R5 は `Send` を変えたので Trigger にもう一度当たり、`verbs.go` は 2478 → 2501 行(1 列目の 2277 から +224、+300 には届かない)。`verbs_test.go` は 5892 行のまま(R5 のテストは新しいファイル `pane_lock_approval_test.go` 210 行と `director_wake_lock_test.go` 146 行に入った)。数え直しは HEAD で `wc -l`。
  - (e) 境界のもの: 456 行の (d)・(e) の Trigger「the next change to `ClassifyResumeScreen`」。R6 は `ClassifyResumeScreen` が止まった文を探すのに呼ぶ `findResumeStop` を変えた。cycle 2 では R2 の `empty` と R3 の `codexFooterStart` の変更をこの Trigger に当たったとして注を足した。同じ扱いなら R6 の分も注が要る。当たったとみなすかは /sync-docs が決める。
- V3-2(LOW、新規): 226 行の (b)。R5 は `Send` を変えたので Trigger「the next change to `Send`」に 3 回目に当たり、(b) の半分を払った。R5 が `Send` の doc を折り返し直したので、`The watch gate is` のあとの短い行はもうない(`internal/org/verbs.go:293-295`)。残りの半分、`internal/org/approval_test.go:945` の `// approval gets` だけの行は残っている。行の (b) を `Send` の分だけ閉じる注を足す。
- V3-3(LOW、新規): 225 行の Impact の数。R5 の `pane_lock_approval_test.go` は `enforceFixture` を使い、改訂の承認を手で書く 5 か所目(`approvedEvent` と `writePlan(approvedPlanText(...))`、`:69-72`)と、計画の本文の変更 `replaceOnce("Add the login form.", "Add the signup form.")` の写しを 3 行、Progress の行の変更(`skip the tests`)の写しを 1 行足した。行は「`approval_enforce_test.go` の 4 か所」「2 つのファイルの 6 つの写し」と数え、3 つ目のファイルを挙げていない。Trigger(helper の変更、計画の形の変更)には当たっていないので、Impact の数を直すか、R5 の分を注で足す。
- V3-4(LOW、新規): register が、上書きされる報告の項目を版なしで引く。
  - 456 行は test の報告の「Test gaps 1 to 6」と変異 R6・R21 などを引く。これは cycle 1 の版(`3148561b`)の番号で、cycle 2 の上書き(`7f1e823e`)で今のファイルの Test gaps 1〜6 は W2、F2・F3・F6、C25、C14、C21、`director_wake.go:608` になった(R21 は今の版に 1 回、`3148561b` の版に 4 回)。行の引く先はもう別の中身を指している。
  - 460 行は cycle 2 の版の「Test gaps 1 to 4 and 6」を引く。cycle 3 の /test が上書きすると同じことが起きる。
  - 447 行と 448 行の `(L10)`、455 行の `(L1)` は cycle 1 の self-review の ID で、self-review の報告は cycle 2 と cycle 3 で上書きされた。
  - 458 行(`the cycle 1 version, commit 65eea701`)と 459 行(`the cycle 2 version after R4`)の形にそろえ、456 行に `3148561b`、460 行に `7f1e823e`、447・448・455 行に `65eea701` を書く。
- V3-5(LOW、FR-10・FR-11): R5 の 2 つの変更が文書に入っていない。どれも今の文は間違いではないが、足りない。
  - (a) AGENTS.md の `internal/org/` の行(91 行)の「`ralph org send`, resume, and the push wake (`director_wake.go`) take one lock per herdr pane while they type」に、herdr の headless の director の wake_cmd もロックの中で走ることを足す。
  - (b) `docs/specs/2026-08-01-org-runtime.md` の 2026-11-07 改訂 (b)(63 行)の「director の push の wake(`ralph org watch --all`)も、director の pane に打つあいだ同じロックを取る」に、設定された wake_cmd も herdr の director(pane を記録したもの)ではロックの中で走り、取れなければ走らせないことを足す。per-plan の implementer への TASK は、ロックの中の待ち直しのあとにもう一度承認を確かめることも 1 文で足す。
  - (c) `/org` skill の per-plan の節(`.claude/skills/org/SKILL.md:1013-1015` と 3 つの写し)の「send は宛先の座席の手が空くのを待ち、待ったあと、打つ前にもう一度照合する」は、確かめが 1 回増えたことを書いていない。1 回目の待ちのあとと、pane のロックを取って座席をもう一度待ったあと、打つ直前の 2 回照合すること、「待つ間」にはロックの待ちも入ることを書く。
  - (d) `/director` skill の 285 行(と 3 つの写し)の「wake_cmd があれば見張りはそちらを使い、`{target}` は pane の id になる」に、そのあいだ director の pane のロックを持つことを足す(失敗の原因は 910〜912 行がすでに挙げている)。
  - (e) 仕様の Status(9 行)は cross-review cycle 1 の改訂だけを書く。cycle 2 の改訂(FR-5a の 65 行)と、上限を 3 に上げて承認し直したことを 1 文で足す。
- V3-6(LOW、self-review C3-2 を確かめた): doc comment の 2 か所がコードより広いことを言う。
  - (a) `StopGlyphs` の doc(`internal/org/resume_screens.go:163-166`)は「the glyphs the CLI starts an error line with」、画面の型の手順 4 (c)(`:80-81`)は「one of the driver's error glyphs」と書く。同じファイルの claude の表のコメント(`:203-206`)は、`⏺` は返事の始まり、`⎿` は prompt や tool 呼び出しの下の行の印だと書く。1 行目が一覧の文で始まる普通の返事(`⏺ API Error: ...`)は R6 のあとも止まったと読まれ、それを書くのは 446 行の解消の注だけである。計画の D8(`:73`)、仕様の FR-5a(`:65`)、R6 の commit の件名も「エラーの印」と書く。
  - (b) `internal/org/pane_lock.go:36-39` はロックを取らない wake_cmd を「a promoted session, a headless director in a terminal launch_cmd opened」と挙げるが、コードの条件は `reg.HerdrLaunch() && reg.PaneID != ""` で、pane を記録していない herdr の director もロックなしで走る(`director_wake_lock_test.go:100` の 3 つ目のケース)。同じファイルの 29 行の「The push wake of a headless director in herdr takes the same lock」も pane の条件を書かない。
  - コメントだけの直しで、/sync-docs は前にも Go のコメントを直している(`a42b9845`、`a4f60ff9`、`64705fbc`)。直さないなら self-review の Tech debt identified の行を足す。
- V3-7(LOW): 計画の `## Progress checklist`(digest の範囲の外)に、cycle 3 の self-review(`5edbb28b`、LOW 2 件)とこの verify の注がない。/sync-docs に回すもの(V3-1〜V3-6、self-review の C3-1・C3-2)を ID で並べる。

### INFO

- I3-1: `docs/specs/2026-10-07-org-multi-org-director.md:122`(日付つきの別の機能の記録)は「send は宛先の座席の手が空くのを待つので、待ったあと、打つ前に承認をもう一度照合する」と書く。R5 のあとも「待ったあと、打つ前」は合っており、ロックの中の確かめはこの PR の仕様の FR-5a の cycle 2 の改訂が書く。
- I3-2: 要約の「claude の `API Error:` で始まる行」(org runtime の仕様の改訂 (a) 62 行、`/org` skill の 362 行、`/director` skill の 980 行、それぞれ 4 つの写し、この PR の仕様の FR-3 の 56 行)。R6 のあと、印なしで `API Error:` から始まるまとまりは unknown で、実機の止まった行は `⏺ API Error:` である。どれも文の一覧の要約で、規則そのものは FR-5a と `resume_screens.go` にある。
- I3-3: この PR の仕様の FR-2 の 4 の cycle 1 の改訂(54 行)は「印を 1 つ落とした先頭」と cycle 1 の規則を書き、FR-5a を指す。cycle 2 の規則(印が必須)は FR-5a の 65 行にだけある。日付つきの記録なので間違いではない。
- I3-4: `ralph org send --help`(`internal/cli/org.go:1335-1361`)は承認の確かめの時点を書かず、`ralph org watch --help`(`internal/cli/org_watch.go:62-66`)は wake のロックを書かない。R5 の前から同じで、間違った文はない。
- I3-5: `docs/recipes/director-wake.md`(と `templates/base/` の写し)はロックに触れない。herdr の headless の director に wake_cmd を設定すると、見張りはロックを最大 10 秒待ち、そのあと wake_cmd が最大 10 秒走る。recipe の `wake-herdr.sh` の例は昇格したセッション向けで、ロックを取らない側である。

## Observational checks

- R5 と R6 の diff を読み、cross-review cycle 2 の 3 件のそれぞれについて、code の場所と、それを固定するテストの名前と assert を突き合わせた(上の表)。
- `Send` を HEAD で頭から読み、`checkSendApproval(p)` の 3 か所(`:383`、`:454`、`:504`)、ロック(`:469`)、待ち直し、予算の確かめ、`typeAndSubmit`(`:538`)の順を確かめた。3 回目の確かめの失敗は defer でロックを外す。
- 新しい fixture 2 つを読み、どちらも空行のあとのまとまりが印なしの `API Error:` で始まることを見た。実機の fixture 4 つ(上の AC-1〜AC-4 の行)の止まった行が、印、空白、文の順で始まることを見た。
- `acquirePaneLock(` の呼び出しを数えた(`verbs.go:469`、`resume.go:360`、`director_wake.go:604` の 3 つ。wake_cmd の道は `lockDirectorWakePane` 経由で 604 行を共有する)。
- tech-debt の Trigger の欄を python で切り出し(行の grep ではなく 4 つ目のセル)、R5・R6 が変えた関数名とファイル名で照合した。当たった行のうち、R5・R6 で新しく当たったか事実が変わったものを V3-1〜V3-3 に挙げた。137 行(`Send` の座席の探し方)、337 行(`Send` の 1 回目の待ち)、396 行(`director_wake_test.go`。R5 のテストは別のファイル)、225 行の Trigger(helper は変わっていない)は、名指されたものを R5・R6 が変えていないので外した。
- register が引く報告の版を、`git show 3148561b:` の版と今の版で `R21` を数えて確かめた(V3-4)。
- 計画の digest、AC の箱(仕様と計画とも `[ ]` は AC-10 だけ)、cycle の状態(`cycle-count.json` は cycle 2、cap 3)を見た。

## Coverage gaps

- Go のテストの結果は見ていない(/test)。表のテストの欄は、テストがあり assert が AC を言っていることまで。R5 の `paneLockOpens` が Linux の CI で期待どおりに数えるか(`/dev/fd`)は、読んで確かめただけである。
- 実機で確かめていない道は cycle 2 のまま(Enter だけ、codex の座席、headless の director、leader とほかの座席の順、応答が返らない通信断)。R5・R6 のあとの binary も実機では走らせていない。wake_cmd とロックの取り合いは偽物の herdr とスクリプトのテストだけ。
- codex が上限の文の前に出す印は実機で見ていない(R6 のコメントのとおり、違えば打たない側)。
- 決定的な検査がないもの: register の行が、上書きされる報告の項目を版なしで引くこと(V3-4)。最小の検査は、`docs/tech-debt/README.md` の行が `docs/reports/<kind>-<date>-<slug>.md` と同じ行で `(L<n>)`・`Test gaps` のような項目を引くとき、同じ括弧に `commit <sha>` か `version` があることを見る shell の 1 本である。

## Verdict

pass

- Verified: cross-review cycle 2 の ACTION_REQUIRED 3 件は、それぞれ code の場所と固定するテストがある(#1 `verbs.go:504` と `pane_lock_approval_test.go` の 2 本、#2 `resume_screens.go:564` と fixture 2 つと edge 4 本、#3 `director_wake.go:517-518`・`:589` と `director_wake_lock_test.go` の 2 本)。FR-1〜FR-11、FR-5a とその改訂 3 つ、非機能要件 5 つ、AC-1〜AC-9、AC-11〜AC-15 は、R5・R6 がその道を変えていないか、変えた道を上で確かめた。静的解析はすべて通った(gofmt、go vet、golangci-lint、staticcheck、shellcheck、sync の 4 つ、template purity、tech-debt の plan の参照、secret scan)。計画の digest は `104ee2d3ca75` で `- Approved:` の行と同じ。origin/main との衝突はない
- Partially verified: AC-10(座席が作業を最後まで進めることは週の上限で未確認。ralph の側は実機で確認済み。実機の binary は R2・R6 より前)
- Not verified: Go のテストの結果と `run-test.sh`(/test)。上の「Coverage gaps」の実機の道
- 所見: LOW 7 件(V3-1〜V3-7)、INFO 5 件(I3-1〜I3-5)。どれも merge を止めない。V3-1 と V3-6 は self-review の C3-1・C3-2 と同じ点(C3-1 (d) は「数を直す」ではなく「R5 の分の引き継ぎの注を足す」が正しい)。V3-1 (e)、V3-2〜V3-5、V3-7 はこの verify で新しく見つけた
- /sync-docs に回すもの: V3-1(460・451・458・365 の各行、456 の判断)、V3-2(226 の `Send` の分を閉じる)、V3-3(225 の数)、V3-4(447・448・455・456・460 の報告の版)、V3-5(AGENTS.md、org runtime の仕様の改訂 (b)、`/org` skill の per-plan の節と `/director` skill の 285 行の 4 つの写し、仕様の Status)、V3-6(Go のコメント 2 か所。直さないなら register の行)、V3-7(計画の Progress)
