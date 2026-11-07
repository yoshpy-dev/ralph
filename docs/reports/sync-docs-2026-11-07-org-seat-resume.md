# sync-docs report: org-seat-resume

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(承認の digest `54630558b591`)。spec は `docs/specs/2026-11-07-org-seat-resume.md`
- Branch: `docs/spec-org-seat-resume`。run 開始時の HEAD は `7f1e823e`(cycle 2 の test の report のコミット)、base は main `11239c8d`
- Pipeline cycle: 2。cycle 1 の report(commit `75a1cda9`)をこの版で上書きした。cycle 1 の版は git の履歴で読める
- 先行 report(cycle 2): self-review `bf8c6b71`(R4 のあと。LOW 5 件)、verify `8e91fa6f`(pass。LOW 7 件 V2-1〜V2-7、INFO 6 件)、test `7f1e823e`(pass。変異 37 個のうち 7 個が生き残った)。cross-review cycle 1 の triage は `docs/reports/cross-review-triage-org-seat-resume.md`(ACTION_REQUIRED 4 件)
- 制約: 計画の本文(`## Progress checklist` と `[ ]` の外)は変えていない。`./scripts/plan-visual.sh digest` は編集の前後とも `54630558b591`で、`- Approved:` の行と同じ
- Go を変えた(依頼に含まれる): `internal/org/resume_screens.go` の doc comment 3 か所だけ(コミット `64705fbc`)。コードも正規表現も変えていない。パイプラインを回し直すかは lead が決める

## Summary

verify が /sync-docs に回した V2-1〜V2-7 と、self-review の C2-1、C2-5、C2-6、R4-1 を処理した。R1(`Send` がロックのあとにもう一度 idle/done を待つ、push の wake が同じロックを取る)のあと古くなっていた `/director` skill の 2 か所、`/org` skill の `send` の行、AGENTS.md の pane_lock の項、org runtime の仕様の改訂 (b) を直した。tech-debt は、`Send` のロックの行を閉じ(V2-1)、R1〜R4 が Trigger を満たした行に引き継ぎの注を足し、解消にした行の注と `ACTION_REQUIRED #N` の cycle を直し、新しい 3 行(cycle 1 の L3〜L9、codex の footer、test の穴)を足した。仕様の Status に cross-review cycle 1 の改訂を書き、Open questions に C2-5 を足した。R4-1 は Go の doc comment の直しで済ませた。

依頼の外で直したものが 3 つある。(1) 仕様の FR-2 の 4 の下に、cross-review cycle 1 の改訂を指す 1 項(Status が「FR-2 の 4 を厳しくした」と書くので、FR-2 側にも印を置いた)。(2) tech-debt の `/director` skill の 2 行(262 と 379)と `/org` skill の表の行(180)の引き継ぎの注。verify は挙げていないが、skill の同じ段落を直したので Trigger に当たった。379 の cycle 1 の注は「2 本目は 1 本目が始めた leader に打ちうる」と書いていて R1 のあと古くなったので、履歴は残して、cycle 2 の注で訂正した。(3) tech-debt 451 行の Impact に C2-5 の取りこぼし(仕様の Open questions と同じ内容)。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | (V2-2 (b)) 見張りが送る 1 行の失敗の原因の括弧に「別の send か resume が director の pane のロックを持ち続けた」を足した。(V2-2 (a)、C2-1 (b)) `### 答える順` の 2 本の send の文を、「2 本目はロックを取ったあと leader が idle か done になるのをもう一度待つので、1 本目で始まった手番が続くあいだは打たない。その手番が `--timeout-ms` のうちに終わらなければ、何も打たずに終了コード 1 で終わる」に直した。`99653051` |
| `.claude/skills/org/SKILL.md` と 3 つの写し | (V2-3) `send` の行に、ロックを取ったあと座席が idle / done になるのをもう一度 `--timeout-ms` の残りまで待つこと、待ちが切れたときと herdr が失敗したときは何も入力せずに終了コード 1 で終わり、そのまま送り直せることを足した。`99653051` |
| `AGENTS.md` | (V2-3) `internal/org/` の行の pane_lock の項に、push の wake(`director_wake.go`)も同じロックを取ることと、`send` がロックのあとに idle / done をもう一度待つことを足した。`99653051` |
| `docs/specs/2026-08-01-org-runtime.md` | (V2-3) 2026-11-07 改訂 (b) に、send のロックの中の待ち直し(切れるか herdr が失敗すると何も打たずに終了コード 1)と、director の push の wake が同じロックを `driverCallTimeout` まで待って取り、取れなければ何も打たず wake が失敗して `woken` を記録しないことを足した。`99653051` |
| `internal/org/resume_screens.go` | (R4-1) doc comment 3 か所。画面の型の手順 2 は、`resumeCodexFooterLine` を「行の頭に錨があり、文脈の 2 つの形は行の終わりにも錨がある」と書き、3 つ目の形を「`<n>% context left` だけの行か ` · ` の最後の欄で、後ろに ` · ? for shortcuts` が付いてもよい」に直した。正規表現のコメントの先頭の文は「3 つの選択肢のすべてが行全体に合う」を同じ言い方に直した(1 つ目の選択肢には `$` がない)。コメントだけ。`64705fbc` |
| `docs/tech-debt/README.md` | 下の「tech-debt」。`e096c981` |
| `docs/specs/2026-11-07-org-seat-resume.md` | Status に cross-review cycle 1 の改訂(FR-2 の 4 と FR-5a)と、自動承認の方針に従って承認し直したことを足した。FR-2 の 4 の下に改訂の 1 項。Open questions に C2-5(prompt の下の最初の `⎿` の行が上限の文でないとき、その下の 2 つ目の `⎿` の行の上限の文は照合しない)。`ece09c81` |
| `docs/plans/active/2026-11-07-org-seat-resume.md` | `## Progress checklist` の「Implementation started」の下に 3 項目(R3 のあとの self-review `0720102d` と R4 `6ee7c1dd`、R4 のあとの self-review `bf8c6b71` と verify `8e91fa6f` と test `7f1e823e`、この sync-docs)。本文は変えていない。`ece09c81` |
| `docs/reports/sync-docs-2026-11-07-org-seat-resume.md` | この report |
| `docs/insights/events/2026-11-07-org-seat-resume.jsonl` | `sync_docs` の行(cycle 2、pass)を `insights-append.sh --slug org-seat-resume --cycle auto` で足した |

`.agents/skills/` の 2 つは `scripts/sync-skills.sh` で作り直し、`templates/base/.claude/skills/` と `templates/base/.agents/skills/` の 4 つは `cp` で合わせた。`director` の 4 つの `SKILL.md` は `md5` が同じ(`58a9ca0a2dc6e32a1a24d9e0ad17ae35`)、`org` の 4 つも同じ(`b56a7080e36d42750880296d150d73cd`)。

## 依頼の項目

| 項 | 結果 |
|----|------|
| V2-1(`Send` のロックの行 = 454 行) | 閉じた。直前に `<!-- RESOLVED ... -->` の行を足し、Debt と Trigger に `~~` を付けた。R1 `0aa8e274`(`AgentWait` のやり直し)、R3 `3714f6d5`(期限のヒント)、R4 `6ee7c1dd`(残る 2 つの場合を `pane_lock.go` の doc に書いた)を書き、残る 2 つの場合は doc を指した。Related に triage の cycle 1 #1 を足した |
| V2-2(`/director` skill の 2 か所、4 つの写し) | 直した。上の表 |
| V2-3(AGENTS.md、org runtime の仕様の改訂 (b)、`/org` skill の `send` の行) | 直した。上の表 |
| V2-4(226 の (b)、365、396、455、337 の判断) | 226 は (b) を払わず、2 回目の引き継ぎの注。365 は行数を base と `6ee7c1dd` で数え直して注(`verbs.go` 2442 → 2478、`verbs_test.go` 5883 → 5892、どちらも最初の列の +300 に届かない)。396 は `director_wake_test.go` 1635 → 1694 の注(固定の期限の 2 つのテストは触れていない)。455 は (d)(e)(g) が R1〜R3 で当たったことと、(g) の期限のない待ちが 2 つから 3 つになったことの注。337 は当たったと判断した(R1 が `Send` の idle の待ちを、ロックの中の 2 回目として足した)。払わず、引き継ぎの注にした |
| V2-5(446・451 の注、`ACTION_REQUIRED #N` の cycle) | 446 の RESOLVED の行と 447 の Trigger の RESOLVED の文に、「返事の先頭が一覧の文で始まると今も止まったと判定する」を足した。451 の Impact に C2-5 の取りこぼしを足した。445 の Related、447 の Related、448 の Why deferred と Related の `ACTION_REQUIRED #3`・`#4` を「cross-review cycle 1, ACTION_REQUIRED #N, commit dd008093」の形にした(triage の report は cross-review の cycle 2 で書き換わりうるので、コミットも書いた) |
| V2-6(新しい 3 行) | 足した。下の「新しい 3 行」。R4-1 は行にせず Go の doc comment を直した |
| V2-7(計画の Progress の注) | 足した。`0720102d`、`6ee7c1dd`、`bf8c6b71`、`8e91fa6f`、`7f1e823e` |
| C2-1(register と skill の古い文) | (a) は V2-1。(b) は V2-2。(c) は V2-5。(d) は V2-5 |
| C2-5(第 2 の `⎿` の下の上限の文) | 仕様の Open questions と tech-debt 451 行の Impact に書いた(両方) |
| C2-6(L3〜L9) | 新しい 1 行にまとめた |
| 仕様の Status | cross-review cycle 1 の改訂(FR-2 の 4、FR-5a)と、自動承認の方針に従って承認し直したことを書いた |

## tech-debt

`docs/tech-debt/README.md` の変更は `git show HEAD:` と行ごとに比べた(HEAD は `7f1e823e` の版)。変わった行は 14 行の編集、足した RESOLVED の行 1 行、足した新しい行 3 行(合計 18 行)。変わった表の行は 5 欄で、欄ごとの backtick と `~~` は偶数(`td_check.py` で確かめた)。足した文の `file:line` の引用は 0(関数名とテスト名で書いた)。`HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` は rc 0(「tech-debt README plan references」を含む)。行は先頭の言葉で示す。行番号は `7f1e823e` の版のもの。

### 閉じた行、状態を直した行

| 行 | 変更 |
|----|------|
| `Send` takes the pane lock after its idle/done wait(454) | RESOLVED。上の V2-1 |
| `typeDirectorWakeLine` does not take the pane lock(445。解消済み) | Related の `ACTION_REQUIRED #3` に cycle とコミットを足した |
| The stopped-screen match of `ClassifyResumeScreen`(447。解消済み。446 は RESOLVED の行) | 「返事の先頭が一覧の文で始まると今も止まる」を RESOLVED の行と Trigger の RESOLVED の文に足した。Related に cycle とコミット |
| The enter-only sentence of `ClassifyResumeScreen`(448) | Why deferred と Related の `ACTION_REQUIRED #4` に cycle とコミット |
| The usage-limit screens of claude and codex(451) | Impact に C2-5 を足した |

### 引き継ぎの注を足した行(R1〜R4 と、この pass の skill の直しが Trigger に当たったもの)

| 行 | 当たったもの | 払わなかった理由 |
|----|----|----|
| Small findings of org-plan-approval(226。(b)) | `Send` とその doc を R1・R3・R4 が変えた | 「`The watch gate is`」のあとの短い行は残り、doc の reflow は R1〜R4 の目的でない |
| A Codex leader that waits for its seats(337) | R1 が `Send` の idle の待ちを、ロックの中の 2 回目として足した。2 回目の待ちのエラーに R3 が同じヒントを付けた | 最初の待ちは変えていない。agmsg と watch に渡す案は計画の Scope の外 |
| `verbs.go` is 2277 lines ほか(365) | `verbs.go` 2442 → 2478、`verbs_test.go` 5883 → 5892(`6ee7c1dd` で `wc -l`) | move-only の分割は Scope の外。どのファイルも最初の列の +300 に届かない(+201、+38)。cycle 1 の注の「`verbs_test.go` は変わらず」は `a42b9845` の時点で正しいと注に書いた |
| Two tests with fixed deadlines(396) | `director_wake_test.go` 1635 → 1694(R1 の新しいテスト) | 固定の期限の 2 つのテストは触れていない |
| Seven mutants of `ralph org resume` survive(455。(d)(e)(g)) | R2・R3 が `ClassifyResumeScreen` の入力欄の読みを変えた。R1 の新しいテストが `<-typing` を期限なしで待つ | cap を使い切り、`/test` はテストを書かない |
| Two documentation findings of org-stop-all(180。(a)) | `/org` skill の表の `resume` の行(新規。表の行として 1,262 文字)と `send` の行(2,121 → 2,528) | `stop`(1,208)と `disband`(1,219)は変わらず、表の組み替えは Scope の外。cycle 1 の sync-docs が拾い損ねた分も、この注に入れた |
| Three test gaps of the org-director-attach PR(262) | `/director` skill の受信箱の節の 2 文(push の wake の失敗の原因、2 本の send) | `director_skill_sync_test.go` は触れていない |
| RC6(379) | `### 答える順` の 2 本の send の文を R1 に合わせて書き直した | 届いた印の候補は書いておらず、`ralph org read` も変えていない。cycle 1 の注が言う「2 本目は 1 本目が始めた leader に打ちうる」は R1 で古くなったので、cycle 2 の注で訂正した |

数え直したもの。`wc -l` を base(main `11239c8d`)、cycle 1 の sync-docs の `a42b9845`、HEAD で取り、注には最後に数えたファイルを変えたコミット `6ee7c1dd` を書いた。`spawn.go`(3093)、`spawn_test.go`(6408)、`prompts_test.go`(2700)、`director_skill_sync_test.go`(1223)は `a42b9845` から変わっていない。表の行の長さは `git show 11239c8d:.claude/skills/org/SKILL.md` と HEAD から、`| \`<動詞>\`` で始まる行を数えた。

### 新しい 3 行(末尾)

| 行 | 中身 |
|----|------|
| Seven LOW findings of the cycle 1 self-review of org-seat-resume(L3 から L9) | (L3) 見直しで herdr が失敗したときの理由が `the screen changed`、(L4) 文が残りうる `failed` に pane の読み方の案内がない、(L5) role prompt が続行の文で始まる行を指示とする、(L6) `heldFileLock` の doc の一覧に pane のロックと wake がない、(L7) 読む行数 60 が 2 か所、(L8) 起動スクリプトの `RetryEnv` の名前を検査しない、(L9) `PrintableInboxText` が Cf を通す。Trigger は項目ごとの関数の次の変更。直し方は cycle 1 の self-review の Recommendation |
| The recognition of codex's footer(`resumeCodexFooterLine`、`codexFooterStart`) | (R3-2 b) 証拠にない形が 3 つ(`Context <n>% left`、`<n>% context left`、後ろの ` · ? for shortcuts`)。(R3-2 c) codex が footer の形を変えると codex の全座席が「input box has text」で飛ばされ、理由から原因が分からない。R4-1 は直したと書いた |
| Five test gaps from the cycle 2 `/test`(Test gaps 1, 2, 3, 4, 6) | (a) wake のロックの待つ長さ(W2)、(b) R4 の `$` と「ほかの欄」の要求を 1 つずつ(F2、F3、F6)、(c) codex の `•`(C25)、(d) placeholder の例外が codex に限ること(C14)、(e) wake がロックのファイルを開けない枝 |

### 退けたもの

Trigger の欄を python で読み(取り消し線の外)、skill・仕様・AGENTS.md・`resume_screens.go` の言葉で走査した(`scan.py`)。名指された節か関数の hunk を読んで決めた。

| 行 | 退けた理由 |
|----|------------|
| 355、186、376、239、400(`/org` skill の受信箱の節、window のテストと文、split の例、終わり方の節、ledger の場所の項) | `send` の行だけを直し、これらの節や項には触れていない |
| 195、199、244〜252(`/director` skill が merge-order や承認を読む) | その節は直していない。直したのは受信箱の「見張りが送る 1 行」と `### 答える順` |
| 338(reviewer の stall ALERT を `/director` と `/org` skill が gate と同じに扱う) | その段落は直していない |
| 449、450(`Waiting` の文を `ralph status` で読む、codex の待ちの画面) | R4-1 の直しは footer の doc comment だけで、Trigger が名指す「再試行の表示」「codex の画面」ではない |
| 183・184・191・238 | verify が外した理由のまま(名指された関数を R1〜R4 が変えていない) |
| 55(AGENTS.md の repo map の更新) | Trigger は別の廃止予定を外すときの話で、今回の AGENTS.md の編集は pane_lock の項だけ |

## Go の変更の確かめ

| 確かめ | 結果 |
|----|------|
| `gofmt -l internal/org` | 空 |
| `go vet ./internal/org/` | 通った |
| `go test ./internal/org/ -run 'Skill\|ResumeScreen\|ClassifyResume\|Recipe' -count=1` | ok(1.271 秒)。skill の編集と `cp`、doc comment の編集のあとで打った |
| `./scripts/run-static-verify.sh` | exit 0、`All verifiers passed.`(gofmt ok、`0 issues.`、branch secret scan は `11239c8d..ece09c81` が clean)。証跡は `docs/evidence/verify-2026-11-07-032408.log`(`.gitignore` の `docs/evidence/*.log` に当たるので、コミットしていない) |

打っていないもの: `./scripts/run-test.sh`、`internal/org` と `internal/cli` の全体のテスト。変えた Go はコメント 3 か所だけで、正規表現も判定のコードも変えていない。

## 確かめたこと

- `./scripts/check-sync.sh`: PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。skill の編集のあとに打ち、最後にもう一度打った
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`: rc 0(tech-debt の plan の参照を含む)
- plan の digest: `./scripts/plan-visual.sh digest` は `54630558b591` で、`- Approved:` の行と同じ。plan の diff は `## Progress checklist` に 3 項目を足しただけ
- 仕様の箱: `[ ]` は AC-10 の 1 つのまま(変えていない)
- `git diff --check main...HEAD` は空。迷子のファイルは無い。scratch は `/tmp/docmaint-seat-resume-c2/` に置いた

## 残っていること(lead へ)

- パイプライン: Go の `64705fbc` は doc comment だけを変える。`post-implementation-pipeline.md` は「cycle の上限で `.go` の編集は全体の回し直し」と書く。cycle 1 の sync-docs の `.go` の編集と同じ扱いで、cycle の上限に達しているため回し直さず、`/cross-review` に進む前提で書いた。回し直すかは lead が決める。回し直さない場合も、PR の本文に「sync-docs が `resume_screens.go` の doc comment だけを直した」と書く
- tech-debt の Related が `docs/plans/active/2026-11-07-org-seat-resume.md` を引く行は 226、337、365、396 で、`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` は先に書いていない。新しい 3 行と 454 行は plan を slug(`2026-11-07-org-seat-resume`)で書いた
- AC-10 は未チェックのまま。tech-debt の AC-10 の行は変えていない
- PR の本文の既知の穴に載せるもの(cycle 1 から変わらないもの): AC-10 の最後の段階(座席が作業を最後まで進めること)と役割 prompt の効き、実機で見ていない道(Enter だけ、codex、headless の director、複数の org、応答が返らない通信断)、live の確認は S11 の前の binary。cycle 2 で足すもの: R1〜R4 のあとの binary は実機で走らせていない。push の wake とロックの取り合い、`Send` のロックの中の待ち直しは、偽物の herdr のテストだけで確かめた
- release のノートに載せるもの(計画の Rollout): 動いている org の座席には、spawn し直すまで再試行の変数が入らない。`api_wait_minutes` は 0 で止められる。cycle 2 で足すもの: `ralph org send` はロックを取ったあと座席の idle / done をもう一度待つので、同じ座席への resume や別の send と重なると、これまでより待たされるか、何も打たずに終了コード 1 で終わることがある
- diff の大きさ(/pr が walkthrough を決めるための数): `git diff main...HEAD` は 91 ファイル、+8,043/−202(HEAD `ece09c81` まで。この report とその jsonl は含まない)。Go のテストでないコードは +1,825/−80、Go のテストは +3,670/−42。walkthrough はここでは書いていない

## Insight event

`./scripts/insights-append.sh --slug org-seat-resume --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を打った。cycle は `cycle-count.json` と同じ 2。既存の `docs/insights/events/2026-11-07-org-seat-resume.jsonl` に足された。
