# Verify report: org の座席を API エラーから続ける(`ralph org resume` と再試行の設定)

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(spec `docs/specs/2026-11-07-org-seat-resume.md`)
- Verifier: verifier サブエージェント(`/verify`)。パイプラインの cycle 2。cycle 1 の報告(commit `39db8f25`)をこの版で上書きした。cycle 1 の版は git の履歴で読める
- Scope: `git diff 11239c8d...HEAD`(base main `11239c8d` = origin/main、HEAD `bf8c6b71`、91 ファイル)。重点は cycle 1 の cross-review の直し R1 `0aa8e274`・R2 `306ee58f`、cycle 2 の self-review の直し R3 `3714f6d5`・R4 `6ee7c1dd`、計画と仕様の改訂 `95f45240`・`fc7efaa8`。仕様適合、静的解析、文書の drift を見た。テストは走らせていない(/test の担当)
- Evidence: `docs/evidence/verify-2026-11-07-org-seat-resume.log`(git の対象外。`run-static-verify.sh` の出力、digest、試しの merge、tech-debt の Trigger の走査、行数を入れた)

## Spec compliance

計画の AC は仕様の AC をそのまま使う。テストの欄は、テストがあり assert が AC を言っていることを読んで確かめたもので、結果は /test が出す。cycle 1 から変わっていない FR と AC は、R1〜R4 がその道を変えていないことを diff で確かめ、cycle 1 の判断を引き継いだ。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| FR-1 動詞、対象、`--org-id`、`--dry-run`、`--state-dir` | Met | R1〜R4 は `internal/cli/org_resume.go` と対象の選び方を変えていない。cycle 1 の根拠のまま(`TestOrgResume_OrgIDFilter_TwoOrgsAndTheDirector`) |
| FR-2 判定の順と 2026-11-07 の改訂(prompt の行での区切り) | Met | 順(自分の pane → `agent get` → `pane read --lines 60` → `ClassifyResumeScreen`)は変わらない。R2 で 4 の「入力欄が空」が厳しくなった(下の FR-5a) |
| FR-3 文の一覧を 1 ファイルに、3 種類、出どころ | Met | `resume_screens.go` の `resumeScreenSentences`。R2 で driver ごとに `StopGlyphs` を足した(`:200`、`:215`)。一覧はこのファイルの 1 か所のまま |
| FR-4 leader 以外 → leader → director | Met | 変更なし |
| FR-5 続行の文、Enter 1 回、Enter だけ、watch gate と承認をかけない | Met | 変更なし。`typeAndSubmit` を `Send` と共有 |
| FR-5a ロック、ロックの中の読み直し、Enter の前の確かめ | Met | cycle 1 の根拠に加え、下の 4 行(改訂の各項) |
| FR-5a 改訂: `Send` はロックのあと idle/done をもう一度待つ | Met | `verbs.go:459`(ロック)、`:480`(ロックの中の `AgentWait`、残りの期限、`capMSToContext`)。失敗は何も打たず `SendProgressNothingSent`。期限切れには `sendWaitDeadlineHint` が付く(R3)。`TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing`(herdr の失敗と期限切れの 2 通り、2 回目の待ちの期限が残りを超えないこと、`/director` skill の文が期限切れのときだけ出ること)、`TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing` |
| FR-5a 改訂: director の push の wake も同じロックを取る | Met | `director_wake.go:603`(`acquirePaneLock(reg.PaneID, driverCallTimeout)`、取れなければ何も打たず wake の失敗)。`TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping`(ロックを持たれたままなら herdr に打たず、banner に 1 行、`woken` なし、5 分の見送り。外れたあとは send-text と send-keys の両方でロックを持つ) |
| FR-5a 改訂: 入力欄のすべての行が空のときだけ空(codex は placeholder だけ) | Met | `empty`(`resume_screens.go:420`)は印の行と、入力欄の残りのすべての行を見る。codex の入力欄の終わりは `codexFooterStart`(`:396`)で、既知の footer の形の並びのときだけ footer にする(R3、R4 で行頭と行末に錨)。fixture `claude-input-draft-after-empty-line.txt`、`codex-input-draft-after-empty-line.txt`、`codex-input-draft-no-footer.txt` と `TestClassifyResumeScreen_Edges` の footer の 7 通り |
| FR-5a 改訂: 止まった文は先頭の印を 1 つ落とした先頭に来るときだけ | Met | `findResumeStop`(`:551`)。fixture `claude-reply-quotes-error.txt`(`⏺ Added regression coverage for API Error: ...` が unknown)と、引用・印なし・印 2 つ・他の driver の印の edge |
| FR-6 `resumed`、状態を変えない、dry run は記録しない | Met | 変更なし |
| FR-7 座席ごとの 1 行、リセット時刻、終了コード | Met | 変更なし。help は cycle 1 の sync-docs で記録の失敗の 1 を書いた |
| FR-8 `api_wait_minutes` と 3 つの変数 | Met | 変更なし |
| FR-9 codex の座席と director には足さない | Met | 変更なし |
| FR-10 director の手順書と役割 prompt | Met | cycle 1 の V1(`waiting` に `ralph org read`)は `8ab92612` で直った。手順書の別の 2 か所が R1 のあと古い(下の V2-2)。FR-10 の 3 項目そのものはそろっている |
| FR-11 文書 | Met(LOW V2-3) | 動詞の一覧と改訂はある。ロックの記述が R1 の前のまま(AGENTS.md、org runtime の仕様の改訂 (b)、`/org` skill の send の行) |
| NFR 5 つ | Met | wake のロックの待ちは `driverCallTimeout`、`Send` の 2 回目の待ちは ctx の残りで区切る。どちらも LLM を呼ばない。打つ道が増える変更はない(R2・R3・R4 は打たない側に倒す向きだけ) |
| AC-1〜AC-9、AC-11〜AC-13 | Met | cycle 1 の根拠のまま。R2 のあとも記録の 2 の画面(`⏺ API Error: ...`)と週の上限の画面(`⎿  You've hit your weekly limit`)は先頭の印を落とすと文で始まるので、止まったと判定される(fixture は変わらない) |
| AC-10 実機で通信断 → 戻す → resume で作業が最後まで進む | Partially met | 下の「AC-10 の判断」。cycle 1 の判断を引き継ぐ |
| AC-14 同じ pane への同時の打ち込み、承認画面への変化、入力欄の確かめ | Met | cycle 1 の 4 本に加え、R1 の 2 本で「`Send` が 2 つ目のとき」も打たないことを固定した(cycle 1 の INFO I4 が言った穴が埋まった) |
| AC-15 小さい `api_wait_minutes` で測る | Met | 変更なし(live の記録の 2) |
| cross-review cycle 1 の ACTION_REQUIRED #1 | Met | 上の「`Send` はロックのあと idle/done をもう一度待つ」 |
| cross-review cycle 1 の ACTION_REQUIRED #2 | Met | 上の「入力欄のすべての行が空」。footer のない codex の画面の穴(self-review C2-4)は R3 で閉じた |
| cross-review cycle 1 の ACTION_REQUIRED #3 | Met | 上の「push の wake も同じロック」 |
| cross-review cycle 1 の ACTION_REQUIRED #4 | Met | 上の「先頭の印を 1 つ落とした先頭」 |
| 計画 D1〜D8 と D8 の改訂 2 つ | Met | 改訂の文と code が合う。digest は `./scripts/plan-visual.sh digest` が `54630558b591` を出し、`- Approved:` の行と同じ |

### AC-10 の判断

cycle 1 と同じ判断を引き継ぐ。R1〜R4 はこの判断の材料を変えていない。

- 実機で確かめたこと(`docs/evidence/org-seat-resume-live-2026-11-07.md` の 1): 通信断で座席が止まり herdr が `idle` を返す。`--dry-run` は `would_resume` を出して記録しない。resume は文を打ち、Enter の前の確かめを通って Enter を 1 回押し、`resumed` を 1 件記録した。座席は要求を出し直した。動いている座席は `waiting` で打たなかった。
- 確かめていないこと: 座席が作業を最後まで進めること(アカウントの週の利用上限で `rate_limit` が返った)。役割 prompt の 1 行の効き(実機の座席は役割 `worker`)。
- ralph の決定的な部分(見つける、1 回打つ、Enter 1 回、記録、止まっていない座席に打たない)は実機とテストで確かめた。AC-10 の文言は満たしていないので Partially met とし、merge は止めない。PR の既知の穴に載せる。仕様の AC-10 の注と tech-debt の AC-10 の行(`docs/tech-debt/README.md:456`)がそう書いている。
- 実機の確認は `b5ea321f` の binary で、S11、R2、R3 より前である。R2 で止まった文の照合は「先頭の印を落とした先頭」になったが、接続の拒否の画面の止まった行は `⏺ API Error:` で始まるので、R2 の前後で同じ判定になる。実機ではやり直していない。
- いちばん効く追加の確かめ: 週の上限のリセットのあと、live の記録の 1 の手順を役割 `implementer` の座席で HEAD の binary で走らせ、`done.txt` ができることを見る。役割 prompt の 1 行の効きと R2 のあとの判定も同時に見られる。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass(exit 0、35 秒) | testdata の `.txt` が言語に分類されず、Go は全体で走った(`full fallback`) |
| shellcheck と `sh -n`(hooks、verify scripts、両方の写し) | OK | |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-pipeline-sync.sh` | OK | |
| `scripts/check-skill-sync.sh` | OK | 14 skill。`director` の 4 つの `SKILL.md` は md5 が同じ |
| `scripts/check-template-purity.sh` | PASS | |
| tech-debt の plan の参照 | OK | |
| gofmt、`go vet ./...`、golangci-lint、staticcheck | OK | gofmt ok、golangci-lint `0 issues.`、vet と staticcheck は出力なし(どちらも skip されていない) |
| `secret-scan-branch.sh` | clean | `11239c8d..bf8c6b71` |
| `git diff --check main...HEAD` | 空 | |
| `git merge-tree --write-tree --name-only origin/main HEAD` | 衝突なし | origin/main は `git ls-remote` でも `11239c8d` |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 仕様 FR-5a の改訂、計画 D8 の改訂 2 つ | Yes | code と合う |
| `/director` skill(4 つの写し) | No | V2-2 |
| `/org` skill(4 つの写し)の `send` の行 | 一部 | V2-3 |
| AGENTS.md の `internal/org/` の行 | 一部 | V2-3 |
| org runtime の仕様の 2026-11-07 改訂 (b) | 一部 | V2-3 |
| `ralph org send --help`、`ralph org resume --help` | Yes(INFO I2-1) | 間違った文はない |
| Go の doc comment(`resume_screens.go` の画面の型の手順 2、正規表現のコメント) | 一部 | V2-6 の (c)(self-review R4-1) |
| `docs/tech-debt/README.md` | No | V2-1、V2-4、V2-5、V2-6 |
| 計画の `## Progress checklist` | 遅れ | V2-7 |
| README.md | Yes | resume の段落はロックを「`ralph org send` と同じもの」と書くだけで、間違いはない |

### 所見

- V2-1(LOW、self-review C2-1 (a)): tech-debt の `Send` のロックの行(`docs/tech-debt/README.md:454`)は R1 で払われたのに開いたまま。Trigger の直し方(ロックのあとに座席の状態を見直し、idle でも done でもなければ何も打たずに断る)を、R1 は `AgentWait` のやり直しで行った。`<!-- RESOLVED ... -->` と取り消し線で閉じ、R1 `0aa8e274` と R3 `3714f6d5`(期限のヒント)を書く。残る 2 つの場合(手番が終わったあとは打ちうること、前の持ち主の submit が確かめられないままロックが外れると待ち直しが通ること)は `internal/org/pane_lock.go:17-24` の doc が書いているので、注はそこを指せばよい。
- V2-2(LOW、FR-10): `/director` skill の 2 か所が R1 の前の挙動を書く。4 つの写しすべて。
  - (a) self-review C2-1 (b): `.claude/skills/director/SKILL.md:1043-1045` の「2 本目の idle の待ちが 1 本目の Enter より前に終わっていると、2 本目は 1 本目で手番を始めた leader に打つことになる」。R1 のあと、2 本目はロックを取ってから idle/done を待ち直すので、1 本目で始まった手番が続くあいだは打たない。手番が `--timeout-ms` のうちに終わらなければ、何も打たずに終了コード 1 で終わり、stderr は直後の段落が引く文を含む。
  - (b) 新規: `.claude/skills/director/SKILL.md:910` の「送るのに失敗すると(wake_cmd が 0 以外で終わった、herdr が失敗した)」は、失敗の原因を括弧で並べている。R1 で 3 つ目の原因(director の pane のロックを別の send か resume が `driverCallTimeout` のあいだ持ち続けた)が増えた。括弧に足すか、「など」にする。
- V2-3(LOW、FR-11): ロックの記述が R1 の前のまま。どれも間違いではないが、R1 で足した 2 つ(ロックのあとの待ち直しとその失敗、wake がロックを取ること)が抜けている。
  - AGENTS.md の `internal/org/` の行の「the pane lock (`pane_lock.go`: `ralph org send` and resume take one lock per herdr pane while they type ...)」に、push の wake(`director_wake.go`)も同じロックを取ることを足す。`acquirePaneLock` の呼び出しは `verbs.go:459`、`resume.go:360`、`director_wake.go:603` の 3 つ。
  - `docs/specs/2026-08-01-org-runtime.md` の 2026-11-07 改訂 (b) は「send は `idle`/`done` の待ちのあとに取り、本文を打って Enter を押し、送信を確かめるまで持つ」と書く。ロックの中の待ち直しと、その失敗で何も打たずに終了コード 1 になることを足す。利用者に見える stderr が 1 つ増えている(`wait for seat %q idle/done again under the lock of its pane (%s)`)。wake のロックも 1 文で足す。
  - `/org` skill の `send` の行(`.claude/skills/org/SKILL.md:359` と 3 つの写し)の「idle / done の待ちのあとに、その pane のロック(`resume` と共有)を取り、本文を入力して Enter を押し、送信を確かめるまで持つ」に、ロックを取ったあと idle / done をもう一度待ち、そうならなければ何も入力せずに終了コード 1 で終わることを足す。
- V2-4(LOW): R1〜R4 が Trigger を満たしたのに、引き継ぎの注がない行。cycle 1 の sync-docs のあとのコミットで当たったもの。走査は Trigger の欄を python で切り出し、R1〜R4 の関数名とファイル名で照合した(証跡の「tech-debt Trigger sweep」)。
  - 226 (b): 「the next change to `Send`」。R1・R3・R4 が `Send` とその doc を変えた。(b) の短い行(`internal/org/verbs.go:290` の `The watch gate is`)は残っている。
  - 365: 「The next change to `Send`」。`verbs.go` は 2442 → 2478 行、`verbs_test.go` は 5883 → 5892 行(cycle 1 の注は「`verbs_test.go` did not change」と書くが、R1 が変えた)。数え直しは base `11239c8d`、cycle 1 の sync-docs の終わり `75a1cda9`、HEAD `bf8c6b71` で `wc -l`。+300 には届かない。
  - 396: 「the next change to `internal/org/director_wake_test.go`」。R1 が `TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping` を足した(1635 → 1694 行)。固定の期限の 2 つのテストは触っていない。
  - 455: (d)・(e) の「the next change to `ClassifyResumeScreen`, `ResumeInputHolds`」は R2 と R3 が当てた(`empty` の呼び出し、`codexFooterStart`)。両方の文を見せる fixture も、文の先頭だけを見せる入力欄も足していない。(g) の「the next test added to `pane_lock_test.go`」は R1 が当てた。新しい `TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing` も `<-typing` を期限なしで待つ(`internal/org/pane_lock_test.go:575`)ので、行の「two tests」は 3 つになった(`:464`、`:499`、`:575`)。
  - 境界のもの: 337 の「the next change to `Send`'s idle wait」。R1 は 1 回目の待ちを変えず、ロックの中に 2 回目を足した。当たったとみなすかは /sync-docs が決める。
- V2-5(LOW、self-review C2-1 (c)(d)、C2-5): 解消にした行の注の抜け。
  - `docs/tech-debt/README.md:446` の解消の注は、まとまりの先頭が一覧の文で始まる普通の返事(先頭の印を落とすと `API Error:` などで始まるもの)を今も止まったと判定することを書いていない。
  - R2 で、prompt の下の最初の `⎿` の行がほかの行のとき、その下の上限の行は照合しなくなった(`TestClassifyResumeScreen_Edges` の「only the first ⎿ line under a prompt starts a block, the next ones join it」が Unknown を期待する)。打たない側に倒れるが、計画の D8 の R2 の改訂と、利用上限の行(`:451`)の Impact に書いていない。計画の D8 は digest の範囲なので、行の Impact だけに書くのが手軽である。
  - 444-447 の Related の欄の `(ACTION_REQUIRED #3)`、`(ACTION_REQUIRED #4)` は cycle を書いていない。cross-review の cycle 2 が triage の報告を書き換えると別の指摘を指す。「cycle 1 の ACTION_REQUIRED #3(commit `dd008093`)」の形にする。
- V2-6(LOW、self-review C2-6、R3-2、R4-1): register にまだない LOW。上限 2 の最後の回で、`/pr` が計画を archive すると持つものがなくなる。
  - (a) cycle 1 の self-review の L3〜L9 の 7 件(L6 の `heldFileLock` の doc の保持する側の一覧には、R1 で wake も加わった)。
  - (b) codex の footer の判定(`resumeCodexFooterLine`、`resume_screens.go:261`)。証拠にない形が 3 つ入っていること(R3-2 (b))と、codex が footer の形を変えると codex の全座席が「input box has text」で飛ばされ、理由の文から原因が分からないこと(R3-2 (c))。
  - (c) R4-1: 画面の型の手順 2(`resume_screens.go:48`、`:50-51`)と正規表現のコメントの先頭の文(`:253`)は「行全体に合う」「`<n>% context left` で終わる行」と書くが、1 つ目の選択肢は行頭だけに錨があり、3 つ目は数字で終わるだけの行には合わない。正しい説明は同じコメントの項目 1〜3 にある。doc の 3 行の直しで済む。Go の変更になるので、直すか register に入れるかは lead が決める。
  - self-review の Tech debt identified の 2 行がそのまま使える。
- V2-7(LOW): 計画の `## Progress checklist`(digest の範囲の外)に、R3 のあとの self-review(`0720102d`、LOW 6)、R4 `6ee7c1dd`、R4 のあとの self-review(`bf8c6b71`、LOW 5)、この verify の注がない。self-review が勧めたとおり、/sync-docs に回すもの(C2-1、C2-5、C2-6、R4-1)と register に入れるもの(L3〜L9、R3-2 (b)(c)、直さないときの R4-1)を ID で並べる。

### INFO

- I2-1: `ralph org send --help` はロックのあとの待ち直しを書いていない。「Before typing, send waits up to --timeout-ms for the seat to be idle or done」がそれも含むと読め、期限切れの文(`the seat's turn did not end`)は 2 回目の待ちのエラーにも付くので、間違った文はない。
- I2-2: `Send` がロックの中で手番の終わりを待つあいだ、ロックを持ち続ける。director の長い send(`--timeout-ms` が 25 分に近いもの)が 2 回目の待ちに入ると、そのあいだ同じ pane への resume は 30 秒でその座席を `skipped` にし、ほかの send は自分の期限まで待つ。`resume.go` の `defaultResumePaneLockWait` の doc と `pane_lock.go` の doc が書いている設計で、2 回目の待ちは 1 回目のあとに誰かが手番を始めたときだけ長くなる。
- I2-3: 仕様の Status の段落は、計画助言のあとの改訂だけを書き、cross-review cycle 1 の FR-5a の改訂に触れていない。FR-5a の本文には日付つきの改訂がある。
- I2-4: 計画の Open questions の 1 つ目(「S10 でも当てられない」)は cycle 1 の I5 のまま古い。digest の範囲なので、計画の承認をし直すときに直す。
- I2-5: `docs/specs/2026-10-07-org-multi-org-director.md` の wake の記述(send-text、750 ms、Enter)はロックに触れない。日付の付いた別の機能の記録で、ロックはこの PR の仕様の FR-5a の改訂が書いている。
- I2-6: 計画の Progress の S4 と S5 の注(「普通の返事の … stopped と判定しうる(tech-debt)」「director の push の wake はロックを取らない(tech-debt)」)は R1・R2 のあと古いが、その時点の記録として読め、後ろの注(151 行)が直したことを書いている。

## Observational checks

- R1〜R4 の diff を読み、FR-5a の改訂の 4 項と cross-review の 4 件のそれぞれについて、code の場所と、それを固定するテストの名前と assert を突き合わせた(上の表)。
- 新しい fixture 4 つを読み、テストの名前の期待(unknown、input box has text)と画面の形が合うことを見た。`codex-usage-limit.txt` は R3・R4 のあとも footer の 2 行(`GPT-6-Astra · xhigh · Context 12% used`、`? for shortcuts`)を持ち、正規表現の 2 つ目と 1 つ目の選択肢に当たる形である。
- `acquirePaneLock` の呼び出しを数えた(3 つ)。AGENTS.md と org runtime の仕様が挙げるのは 2 つ。
- tech-debt の Trigger の欄を python で切り出し、R1〜R4 が変えた関数名とファイル名で照合した。当たった 15 行のうち、R1〜R4 で新しく当たったものを V2-4 に、解消すべきものを V2-1 に挙げた。183・184・191・238・338・449・450 は、名指されたものを R1〜R4 が変えていないので外した。
- 計画の digest、AC の箱(仕様と計画とも `[ ]` は AC-10 だけ)、cycle の状態(`cycle-count.json` は 2)を見た。

## Coverage gaps

- Go のテストの結果は見ていない(/test)。表のテストの欄は、テストがあり assert が AC を言っていることまで。
- `resumeCodexFooterLine` の正規表現を Go で走らせていない。self-review が Python の `re` で机上の確かめをしている。
- 実機で確かめていない道は cycle 1 のまま(Enter だけ、codex の座席、headless の director、leader とほかの座席の順、応答が返らない通信断)。R1〜R4 のあとの binary も実機では走らせていない。push の wake とロックの取り合いは偽物の herdr のテストだけ。
- 決定的な検査がないもの: 文書がロックを取る呼び出しの数(AGENTS.md と仕様が 2 つ、code が 3 つ)を追う検査。最小の検査は、`internal/org` の `acquirePaneLock(` の呼び出しのファイル名が AGENTS.md の `pane_lock.go` の括弧の中にすべて出ることを見る Go のテスト 1 本。

## Verdict

pass

- Verified: FR-1〜FR-11、FR-5a とその cross-review cycle 1 の改訂の 4 項、非機能要件 5 つ、AC-1〜AC-9、AC-11〜AC-15 を code とテストの assert で確かめた。cross-review cycle 1 の ACTION_REQUIRED 4 件は、それぞれ code の場所と固定するテストがある。静的解析はすべて通った(gofmt、go vet、golangci-lint、staticcheck、shellcheck、sync の 4 つ、template purity、tech-debt の plan の参照、secret scan)。計画の digest は `54630558b591` で `- Approved:` の行と同じ。origin/main との衝突はない
- Partially verified: AC-10(座席が作業を最後まで進めることは週の上限で未確認。ralph の側は実機で確認済み。実機の binary は R1〜R4 の前)
- Not verified: Go のテストの結果と `run-test.sh`(/test)。上の「Coverage gaps」の実機の道
- 所見: LOW 7 件(V2-1〜V2-7)、INFO 6 件(I2-1〜I2-6)。どれも merge を止めない。V2-1、V2-2 (a)、V2-5、V2-6、V2-7 は self-review の C2-1、C2-5、C2-6、R3-2、R4-1 と同じ点で、V2-2 (b)、V2-3、V2-4 はこの verify で新しく見つけた
- /sync-docs に回すもの: V2-1(454 行を閉じる)、V2-2(director の skill の 2 か所、4 つの写し)、V2-3(AGENTS.md、org runtime の仕様の改訂 (b)、`/org` skill の send の行の 4 つの写し)、V2-4(226・365・396・455 の引き継ぎの注、337 の判断)、V2-5(446 と 451 の注、444〜447 の Related)、V2-6(register の 2 行。(c) の Go の doc を直すかは lead)、V2-7(計画の Progress の注)
