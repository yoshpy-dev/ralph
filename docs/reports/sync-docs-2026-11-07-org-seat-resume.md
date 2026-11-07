# sync-docs report: org-seat-resume

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(承認の digest `104ee2d3ca75`)。spec は `docs/specs/2026-11-07-org-seat-resume.md`
- Branch: `docs/spec-org-seat-resume`。run 開始時の HEAD は `96b7f34f`(cycle 3 の test の report のコミット)、base は main `11239c8d`
- Pipeline cycle: 3(上限はユーザーの方針で 3 に上がっている。`.harness/state/standard-pipeline/cycle-count.json` は cycle 2・cap 3 のままなので、insight の event は lead の指示で `--cycle 3` を渡した)。cycle 2 の report(commit `a5ecb26a`)をこの版で上書きした。cycle 1 の版は `75a1cda9`、cycle 2 の版は `a5ecb26a` で読める。tech-debt の register にこの report を引く行はない
- 先行 report(cycle 3): self-review `5edbb28b`(LOW 2 件 C3-1・C3-2)、verify `0cbce5fb`(pass。LOW 7 件 V3-1〜V3-7、INFO 5 件)、test `96b7f34f`(pass。変異 21 個のうち 4 個が生き残った)。cross-review cycle 2 の triage は `docs/reports/cross-review-triage-org-seat-resume.md`(ACTION_REQUIRED 3 件、commit `ed918303`)
- 制約: 計画の本文(`## Progress checklist` と `[ ]` の外)は変えていない。`./scripts/plan-visual.sh digest docs/plans/active/2026-11-07-org-seat-resume.md` は編集の前後とも `104ee2d3ca75` で、`- Approved:` の行と同じ。計画の diff は Progress の 2 行の追加だけ(`git diff --numstat` で 2/0)
- Go を変えた(依頼に含まれる): `internal/org/resume_screens.go` と `internal/org/pane_lock.go` の comment だけ(コミット `70eb32a6`)。コードも正規表現も変えていない。パイプラインを回し直すかは lead が決める

## Summary

verify の V3-1〜V3-7 と self-review の C3-1・C3-2 を処理した。cross-review cycle 2 の R5(`Send` の最後の承認の確かめ、herdr の director の `wake_cmd` のロック)と R6(止まった文は印、空白、文の順)のあと古くなっていた文書を直した。skill の 2 か所(4 つの写し)、AGENTS.md の pane_lock の項、org runtime の仕様の改訂 (b)、この PR の仕様の Status、Go の comment 3 か所(`StopGlyphs` の doc、画面の型の手順 4 (c)、`pane_lock.go` の doc の 2 文)。tech-debt は 10 行を直した。460 行の (c) を閉じて (a)・(e) に R5 の引き継ぎの注を足し、cycle 3 の test の穴を (f)〜(h) と (a)・(e) の追記で入れた。

依頼の外で直したものが 2 つある。(1) 460 行に G9(印のあとの空白の判定、INFO)を (h) として足した。依頼の項目は A2・W3・W6・`director_wake.go` の開けない枝だけだが、test report の Test gaps 4 で、生き残った変異 4 個の 1 つなので、記録から落とさない。(2) 460 行の Trigger から `StopGlyphs` を取り消し線で外し、`lockDirectorWakePane`・`directorWakeSender`・`Send` の承認の確かめを足した。(c) が閉じたことと、R5 で道が移ったことによる。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | (V3-5 (c)) per-plan の節の `send` の項を、確かめが 3 回あることに合わせた。待ったあとにもう一度照合し、そのあと pane のロックを取って座席の手が空くのをもう一度待ち、打つ直前にも照合する。`--dry-run` は 1 回だけ。「待つ間」にロックの待ちも入る。`49f0924c` |
| `.claude/skills/director/SKILL.md` と 3 つの写し | (V3-5 (d)) `herdr で立てた director` の項に、wake_cmd が走るあいだ見張りが director の pane のロックを持つこと、別の send か resume がロックを持ち続けて待ちが切れたときは wake_cmd を走らせず wake の失敗になることを足した。`49f0924c` |
| `AGENTS.md` | (V3-5 (a)) `internal/org/` の行の pane_lock の項に、push の wake が打つ 1 行と、pane を記録した herdr の director に設定された `wake_cmd` の両方がロックを取ること、`send` が per-plan の org の implementer への TASK で、ロックの中の待ち直しのあとに計画の承認を最後にもう一度確かめることを足した。`49f0924c` |
| `docs/specs/2026-08-01-org-runtime.md` | (V3-5 (b)) 2026-11-07 改訂 (b) に、per-plan の org の TASK の最後の承認の確かめ(外れていれば何も打たず、記録せず、終了コード 1、ロックは外れる)と、pane を記録した herdr の director の `wake_cmd` がロックの中で走り、取れなければ走らせないこと、ロックを取らない登録(pane を記録していない herdr の director、昇格したセッション、launch_cmd の director)を足した。`49f0924c` |
| `internal/org/resume_screens.go` | (V3-6 (a)、C3-2 (a)) `StopGlyphs` の doc を「止まった文を実機で見た(claude)か見ると予想する(codex)直前の印」にし、claude の `⏺` は返事の始まり、`⎿` は prompt の下の行の印でもあるので、返事の最初の言葉が止まった文なら今も止まったと読むことを書いた。画面の型の手順 4 (c) の「error glyphs」を `StopGlyphs` に替えて同じ 1 文を足した。コメントだけ。`70eb32a6` |
| `internal/org/pane_lock.go` | (V3-6 (b)、C3-2 (b)) 「push の wake は同じロックを取る」を「pane id を記録していれば」とし、wake_cmd がロックを取らない登録の括弧に「pane id を記録していない herdr の headless の director」を足した。コメントだけ。`70eb32a6` |
| `docs/tech-debt/README.md` | 下の「tech-debt」。`d757f748`、456 行の Trigger の注の判断を直した `1fdf719e` |
| `docs/specs/2026-11-07-org-seat-resume.md` | (V3-5 (e)) Status に、cross-review cycle 2 の指摘 3 件を受けた FR-5a の 2 度目の改訂と、パイプラインの上限を 2 から 3 に上げたこと、承認し直し(digest `104ee2d3ca75`)を足した。FR-5a の改訂の文は変えていない。`ec1ef19e` |
| `docs/plans/active/2026-11-07-org-seat-resume.md` | (V3-7) `## Progress checklist` の「Implementation started」の下に 2 項目(cycle 3 の self-review・verify・test、この sync-docs)。本文は変えていない。`ec1ef19e` |
| `docs/reports/sync-docs-2026-11-07-org-seat-resume.md` | この report |
| `docs/insights/events/2026-11-07-org-seat-resume.jsonl` | `sync_docs` の行(cycle 3、pass)を `insights-append.sh --slug org-seat-resume --cycle 3` で足した。13 行から 14 行で、`git status --short` は既存の jsonl の `M` だけ(迷子のファイルなし) |

`.agents/skills/` の 2 つは `scripts/sync-skills.sh` で作り直し、`templates/base/.claude/skills/` と `templates/base/.agents/skills/` の 4 つは `cp` で合わせた。`org` の 4 つの `SKILL.md` は `md5` が同じ(`f67ccf75f900ab58c4ba70312bc64e54`)、`director` の 4 つも同じ(`f36b19316e18234ebe52258845363181`)。編集した skill の行の表示幅(East Asian wide を 2 で数えた)は最大 76 で、前後の行の最大 77 を超えない。

## 依頼の項目

| 項 | 結果 |
|----|------|
| V3-1 (a)(460 行) | 直した。(c) C25 を `~~` で閉じ、R6 `6993faca` と edge「a codex item glyph before a usage limit is not a stop」を書いた(`•` を戻す G3 が赤になる)。(a) と (e) には Trigger の末尾に R5 の引き継ぎの注。Impact の「(c) and (d) are INFO」は (d) だけにした |
| V3-1 (b)(451 行) | Impact に、R6 のあと limit の文は印と空白のあとでしか数えず、codex の印は見ていないので、`•` か印なしなら unknown で飛ばす(打たない側)ことを足した。Trigger に、codex の画面を取ったら印を `StopGlyphs` に入れるか規則を直す、を足した |
| V3-1 (c)(458 行 L6) | `heldFileLock` の doc の持ち手に、R5 の wake_cmd の実行(`runDirectorWakeCmdUnderPaneLock`)を足した。直し方の欄にも |
| V3-1 (d)(365 行) | 数を直すのではなく引き継ぎの注を足した(verify の訂正のとおり)。`verbs.go` 2478 → 2501(+224、+300 に届かない)、`verbs_test.go` 5892 のまま。数えたのは R5 `323e94d5`、数えたファイルを最後に変えたコミット。self-review の「2442 → 2501」は、行の最後の注がすでに `6ee7c1dd` で 2478 と書いているので採らなかった |
| V3-1 (e)(456 行 (d)(e)) | 当たったと判断し、払わない引き継ぎの注を Trigger の末尾に足した(`1fdf719e`)。R6 が変えたのは `ClassifyResumeScreen` が呼ぶ `findResumeStop` と `StopGlyphs` で、(d) の enter-only と waiting の探す順も、(e) の入力欄の読みも変えていない。それでも当たったとしたのは、cycle 2 が R2 の `empty` と R3 の `codexFooterStart` を、(d) と (e) の両方に当たったとして注を足したのと揃えるため(Trigger の文は `ClassifyResumeScreen` の次の変更)。最初は「当たっていない」と書いて `d757f748` に入れ、cycle 2 の扱いと食い違うので `1fdf719e` で直した |
| V3-2(226 行 (b)) | `Send` の分を閉じた。R5 が doc を折り返し直したので `The watch gate is` のあとの短い行はもうない。`approval_test.go` の `approval gets` だけの行は残り、R5 は触れていない。Debt の (b) の `Send` の部分を `~~` で閉じ、Trigger に注 |
| V3-3(225 行 Impact) | R5 の `pane_lock_approval_test.go`(`enforceFixture` を使う)が持つ写しを足した。5 か所目の手書きの再承認(`a revision approved`)、計画の本文の変更 3 つ、Progress の行の変更 1 つ。数は `grep -c` で確かめた(`replaceOnce("Add the login form.", "Add the signup form.")` が 3、`skip the tests` が 1)。Related に `323e94d5` |
| V3-4(447・448・455・456・460) | 版を足した。447・448 は Impact の「(self-review L10 …)」と Related を「the cycle 1 version, commit 65eea701」に、455 の Related の `(L1)` も同じ。456 は第 1 文と Related を「the cycle 1 version, commit 3148561b」に(R21 は `git show 3148561b:` の版に 4 回、`7f1e823e` の版に 1 回で、今の版の Test gaps 1〜6 は別の中身)。460 は第 1 文と Related を cycle 2 の版 `7f1e823e` と cycle 3 の版 `96b7f34f` に |
| V3-5 (a)〜(e) | 上の表。(c) の「待つ間」にロックの待ちも入ることを書いた。(e) は Status だけ。FR-5a の改訂の文は変えていない |
| V3-6 (a)(b)、C3-2 | Go の comment を直した。下の「Go の変更の確かめ」。直したので self-review の Tech debt identified の行(C3-2 を直さないときの行)は足していない |
| V3-7 | 計画の Progress に 2 項目 |
| C3-1 (a)〜(d) | V3-1 (a)〜(d)。(d) は上のとおり注 |
| test の穴 A2、W3、W6、`director_wake.go` の開けない枝、G9 | 460 行に入れた。A2 = (f)、W3 = (g)、G9 = (h)。W6 は (a) の W2 と同じ穴(R5 で `lockDirectorWakePane` を typed の wake と wake_cmd が共有した)なので (a) に追記。開けない枝は (e) に追記(R5 で `lockDirectorWakePane` に移り、wake_cmd の道からも通る)。新しい行は作らず 460 行を伸ばした。460 の第 1 文の「Five test gaps from the cycle 2 `/test`」は、cycle 3 の test report が「460 行、『Five test gaps from the cycle 2 /test』の (a)」と引いているので残した |
| test の穴 6(460 行の (c) などの食い違い) | V3-1 (a) で直した。(a) の W2 は wake_cmd の道にも広がった、と (a) に、(e) の行は `lockDirectorWakePane` に移った、と (e) に書いた |

## tech-debt

`docs/tech-debt/README.md` の変更は `git show HEAD:` と行ごとに比べた(`/tmp/docmaint-seat-resume-c3/tdcheck.py`、HEAD は `70eb32a6` の版。`1fdf719e` の 456 行の直しも同じ検査を通した)。変わった行は 10 行(225、226、365、447、448、451、455、456、458、460)で、どれも 5 欄(raw で 7 つに分かれる)、欄ごとの backtick と `~~` は偶数。足した文に `file:line` の引用は 0(関数名とテスト名で書いた)。planned anchor 26 個はすべて、編集の前に `anchors.py` で件数 1 を確かめた。行番号は編集の前後で変わらない(行を足していない)。

### 閉じた行、状態を直した行

| 行 | 変更 |
|----|------|
| Five test gaps from the cycle 2 `/test`(460) | (c) を閉じた。(a)・(e) に R5 の追記、(f)〜(h) を足した。Why deferred に cycle 3 の probe と取り消した (c)。Trigger に R5・R6 の引き継ぎの注。Related に版 |
| The usage-limit screens of claude and codex(451) | Impact と Trigger に、codex の印(R6 の規則)を足した |
| Seven LOW findings of the cycle 1 self-review(458) | L6 の持ち手に wake_cmd の実行を足した |
| Small findings of org-plan-approval(226。(b)) | `Send` の分を閉じた(`~~` と注) |

### 引き継ぎの注を足した行、数を足した行

| 行 | 当たったもの | 払わなかった理由 |
|----|----|----|
| `verbs.go` is 2277 lines ほか(365) | R5 が `Send` を変えた(2478 → 2501) | move-only の分割は Scope の外。+300 に届かない |
| Seven mutants of `ralph org resume` survive(456。(d)(e)) | R6 が `findResumeStop` と `StopGlyphs` を変えた(`ClassifyResumeScreen` の次の変更) | R6 のテストは止まった文の照合のためのもので、`/test` はテストを書かない(上の V3-1 (e)) |
| The org-plan-approval tests keep their approval helpers twice(225) | Trigger には当たっていない(helper は変わっていない)。Impact の数が古い | 数を足しただけ |

### 退けたもの

Trigger の欄を python で読み(取り消し線の外)、skill・仕様・AGENTS.md・Go の symbol の言葉で走査した(`scan.py`、キーワードは `StopGlyphs`、`findResumeStop`、`resume_screens`、`pane_lock`、`acquirePaneLock`、`heldFileLock`、`/org` skill、`/director` skill、`per-plan`、`checkSendApproval`、`wake_cmd`、`AGENTS.md`、`org runtime`、`runDirectorWakeCmd`、`lockDirectorWakePane`、`typeDirectorWakeLine`、`director_wake`)。名指された節や関数を読んで決めた。

| 行 | 退けた理由 |
|----|------------|
| 180(`/org` skill の表)、376(split の例)、400(`[org].state_dir` の項)、355、186、338 | 直したのは `/org` skill の per-plan の節の 3 行だけで、表の行、split の例、ledger の場所の項には触れていない |
| 177、195、199、244〜252、262、379、380(`/director` skill) | 直したのは `herdr で立てた director` の項の wake_cmd の 3 行で、受信箱の節、送って確かめる手順、wait の終了コードの節ではない。244〜252 は merge-order と承認を読む節 |
| 159(org の quality pipeline) | 計画の外。直した文は pane のロックの話 |
| 316(`{target}` の shell への安全) | 足した文は `{target}` を扱わない |
| 449、450(`Waiting` の文、codex の待ちの画面) | `resume_screens.go` の comment を直したが、Trigger が名指す `Waiting` の文の表も codex の待ちの画面も変えていない |
| 55(AGENTS.md の repo map) | 別の廃止予定を外すときの話 |
| 79、381、411(org runtime の仕様) | 名指された節ではない。381 は receipts の形 |
| 396(`director_wake_test.go` の固定の期限) | R5 のテストは別のファイル(`director_wake_lock_test.go`) |

## Go の変更の確かめ

| 確かめ | 結果 |
|----|------|
| `gofmt -l internal/org` | 空 |
| `go vet ./internal/org/` | 通った |
| `go test ./internal/org/ -run 'Skill\|ResumeScreen\|ClassifyResume\|PaneLock\|Recipe' -count=1` | ok(3.991 秒)。依頼の `'Skill'` と `'ResumeScreen\|PaneLock'` を 1 回にまとめた。skill の編集と `cp`、Go の comment の編集のあとで打った |
| `./scripts/run-static-verify.sh` | rc 0、`All verifiers passed.`(`gofmt: ok`、`0 issues.`、tech-debt README plan references OK、branch secret scan は `11239c8d..1fdf719e` が clean)。最後の回の証跡は `docs/evidence/verify-2026-11-07-051820.log`(`.gitignore` の `docs/evidence/*.log` に当たるので、コミットしていない)。その前の `ec1ef19e` での回(`verify-2026-11-07-051512.log`)も rc 0 |

打っていないもの: `./scripts/run-test.sh`、`internal/org` と `internal/cli` の全体のテスト。変えた Go は comment だけで、判定のコードも正規表現も変えていない。

## 確かめたこと

- `./scripts/check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。skill の編集のあとと、最後にもう一度
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- plan の digest: `104ee2d3ca75`(編集の前後)。`- Approved:` の行は `2026-11-07 sha256:104ee2d3ca75`
- 仕様の箱: `[ ]` は AC-10 の 1 つのまま(spec の `grep -c '^- \[ \]'` は 1)。計画の `[ ]` は 2(AC-10 と PR created)で、変えていない
- `git diff --check main...HEAD` は空。迷子のファイルは無い。scratch は `/tmp/docmaint-seat-resume-c3/` に置いた

## 残っていること(lead へ)

- パイプライン: Go の `70eb32a6` は comment だけを変える。`post-implementation-pipeline.md` は「cap に達した run の `.go` の編集は全体の回し直し」と読める。cycle 1・2 の sync-docs の `.go` の編集と同じ扱いで、cycle 3 は上限 3 の最後の run なので回し直さず、`/cross-review` に進む前提で書いた。回し直すかは lead が決める。回し直さない場合も、PR の本文に「sync-docs が `resume_screens.go` と `pane_lock.go` の comment だけを直した」と書く
- 直さずに残したもの: 計画の D8(`:73`)と仕様の FR-5a の cycle 2 の改訂は、claude の `⏺`・`⎿` と codex の `■`・`⚠` を「エラーの印」と呼ぶ(self-review のとおり、次に書き直すときに「止まった文の前に見た印」にする。計画は digest の範囲)。`resume_screens_test.go` の comment 3 か所(`:240`、`:258`、`:264`)も「error glyphs」と書く。コードの doc は直したので、読み違える入口は `StopGlyphs` の doc と手順 4 (c) では塞がっている
- tech-debt の Related が `docs/plans/active/2026-11-07-org-seat-resume.md` を引く行は 226、337、365、396 で、`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` は先に書いていない。460 行の追記は plan を slug で書いた
- AC-10 は未チェックのまま。tech-debt の AC-10 の行は変えていない
- PR の本文の既知の穴に載せるもの(cycle 2 から変わらないもの): AC-10 の最後の段階(座席が作業を最後まで進めること)と役割 prompt の効き、実機で見ていない道(Enter だけ、codex、headless の director、複数の org、応答が返らない通信断)、live の確認は S11 の前の binary。cycle 3 で足すもの: R5・R6 のあとの binary は実機で走らせていない。codex が usage limit の文の前に出す印は見ていない(違えば打たない側)。test の穴は register の 460 行に入っている(A2、W3、W6、G9、開けない枝)
- release のノートに載せるもの(計画の Rollout): 動いている org の座席には、spawn し直すまで再試行の変数が入らない。`api_wait_minutes` は 0 で止められる。`ralph org send` はロックを取ったあと座席の idle / done をもう一度待つので、同じ座席への resume や別の send と重なると、これまでより待たされるか、何も打たずに終了コード 1 で終わることがある。cycle 3 で足すもの: per-plan の org の implementer への TASK は、ロックの中でもう一度承認を確かめるので、待つあいだに計画が変わると断られる。herdr の headless の director に `wake_cmd` を設定していると、見張りは director の pane のロックを最大 10 秒待ち、そのあと wake_cmd が最大 10 秒走る(取れなければ wake_cmd は走らない)。claude の座席の止まった文は、印と空白のあとでしか読まないので、印のない画面は unknown になる
- diff の大きさ(/pr が walkthrough を決めるための数): `git diff main...HEAD` は 95 ファイル、+8,572/−230(HEAD `ec1ef19e` まで。この report とその jsonl は含まない)。Go のテストでないコードは +1,932/−95、Go のテストは +4,056/−42。walkthrough はここでは書いていない

## Insight event

`./scripts/insights-append.sh --slug org-seat-resume --flow standard --phase sync_docs --verdict pass --cycle 3 --source skill` を打った(rc 0)。`--cycle auto` なら `cycle-count.json` の 2 になるので、lead の指示で 3 を渡した。既存の `docs/insights/events/2026-11-07-org-seat-resume.jsonl` に足された(13 行から 14 行)。
