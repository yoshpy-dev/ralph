# Verify report: org-xreview-own-model

## Run 2 (cycle 2)

- Date: 2026-11-22(JST。pipeline の 2 周目、`cycle-count.json` は `cycle: 2`、上限 2)
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(issue #360)。`- Approved:` の digest `aa9ca6400c54` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する(Run 1 のあとの計画の変更は Progress だけ)
- Verifier: verifier サブエージェント(`/verify`)。今の判定は、この節のあとの `## Verdict`。Run 1 は下に残し、見出しだけ 1 段下げた
- Scope: branch 全体は `git diff origin/main...HEAD`(HEAD `7eb0179d`、origin/main `ccdf6293` は merge base と同じ、42 ファイル、+2580 −1840)。読み直したのは、Run 1(HEAD `b98b1b96`、report のコミット `7a1dd7ef`)のあとにこの branch が足したコミット。中身のあるものは `7c7cdec5`(`ralph init` の問いの説明。Run 1 の V1 の直し)、`0d44799f`(sync-docs)、`eff83438`(cross-review の ACTION_REQUIRED の直し)、`7eb0179d`(self-review Run 2 の LOW 3 件の直し)と、origin/main の取り込み `6b671276`(#363 と #370)。`a30b6097`、`82090f46`、`0cd1a32f`、`d43f5781` は report と insight event だけ。テストは走らせていない(`/test` の仕事)
- Evidence: `docs/evidence/verify-2026-11-22-org-xreview-own-model-run2.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない。Run 1 のログは書き換えていない)。wrapper 自身のログは `docs/evidence/verify-2026-11-21-232831.log`(UTC の日付)

CRITICAL、HIGH、MEDIUM はない。LOW 1(V9)、INFO 3(V10〜V12)。Run 1 の V1〜V6 は直った(V2 で注を足した行のうち 2 行に、この周の直しがもう一度当たった。V9)。V7 と V8 は INFO のまま残る。

### Spec compliance (Run 2)

新しい受け入れ条件はない。この周の直しは、手順 2 の `diff: failed` の道を AC1 の比べに通したもの。AC ごとに、Run 1 の判定がこの周の変更のあとも成り立つかを見た。

| Acceptance criterion | Status | Evidence (Run 2) |
| --- | --- | --- |
| AC1 | Met | 手順 5 の比べと `XREVIEW: git_changed` の BLOCKED は残り、5 への入り口が 3 つになった(4 から、3 の読み終えられない道から、2 の `diff: failed` から。`internal/org/prompts/reviewer.md:241-248`、`:268-271`、`:295-314`)。`git: changed` の括弧は `7eb0179d` で「2 で状態を書いたあとに HEAD か作業ツリーが動いたので、所見を出していたなら、それは今の branch を見ていない」(`:306-308`)になり、3 つの道のどれにも当てはまる。テストの `reviewerXreviewGone` の 20 語(`codex exec`、`claude -p`、`pick_reviewer`、`mcp list`、`sleep 1200` など)は、許した 2 文を除くと、この節にも「座席内 fan-out」の節にもない(Python のポート) |
| AC2 | Met(`eff83438` で条件が 1 つ付いた。V10) | `diff: failed` なら 3 と 4 を飛ばして 5 に進み、`git: same` のときだけ、6 で `Reviewer status: incomplete (git diff failed: <要点>)` の report を書き、7 で同じ値の RESULT を返す。`git: changed` なら AC1 の BLOCKED で、report は書かない(`:241-248`)。要点を読む出力は、`7eb0179d` で「2 のコマンドの出力」と名指した。5 の出力は `git: same` か `git: changed` の 1 行だけなので、どちらの出力かを取り違えない。3 行の確かめと、前の版の行を読まないことは変わらない |
| AC3 | Met | report の形と 4 つの返事は変わらない。6 の「incomplete のときも `Total reviewer findings: 0` と `After triage: ...=0` の行を残す」(`:322-324`)が、2 の道の report も覆う |
| AC4 | Met | この周の変更は `prompts.go`、`spawn.go`、雛形の `{{DRIVER}}` と `{{MODEL}}` に触れていない |
| AC5 | Met | leader の雛形はこの周に変わっていない。leader の `XREVIEW: git_changed` の項(`internal/org/prompts/leader.md:1302-1313`、「reviewer が読む前と report を書く前で git の状態が違い、report は書かれていない」)は、2 の道から来た BLOCKED にもそのまま当てはまる。`XREVIEW: incomplete` の項(`:1288-1291`。上限に数えず、頼み直さず、sync-docs に進む)は、`git: same` のときに返る `incomplete (git diff failed: ...)` を受ける。送り直しの上限(`:1222-1236`)は `git_changed` と QUESTION を合わせて数えるので、2 の道の `git_changed` もここに入る |
| AC6 | Met(振る舞い)。字面は満たさない(V7、Run 1 のまま) | `7a1dd7ef..HEAD` では、`scripts/xreview-helpers.sh` と写し、標準フローの `/cross-review` の skill は変わっていない。`origin/main...HEAD` の差分は Run 1 と同じコメントだけ |
| AC7 | Met | `check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh`(14 skill)は wrapper の中で通った。org と director の skill は 4 写しずつ `cmp` で同じ。`7eb0179d` で `t.Fatalf` の文字列 2 つ(`internal/cli/org_runtime_env_test.go:259`、`internal/org/driver/herdr_test.go:588`)が新しい動きを言うようになり、`git grep "takes the implementer's driver"` が出すのは report の行だけになった |
| AC8 | Met | recipe はこの周に変わっていない |

流れと取り込みの照合:

1. 手順 2・3・5・6・7 の流れ(`reviewer.md:224-369`)。2 の `diff: failed` → 5(`git: changed` なら BLOCKED、`git: same` なら 6 で incomplete の report、7 で RESULT)。3 の読み終えられない道 → 5 → 6 → 7。4 → 5 → 6 → 7。2 のコマンドは `git diff` より前の行で `-git-before.txt` を書く(`:236-237`)ので、`git diff` が失敗した道でも 5 には比べる相手がある。6 の `HEAD:` は `-git-before.txt` の 1 行目で、2 の道でも決まる。7 の括弧(`:367-368`)は `git_changed` を 5 に、QUESTION を 1 に送る。食い違いはない。3 の道の文は 5 の `git: changed` の分かれを書かずに「5 に進み、6 で report を…書いて」と言う(`:268-271`)。5 が「report を書かず」と言うので食い違いではないが、2 の道と書き方がそろっていない(V11)
2. leader の扱い: 上の AC5 の行
3. 仕様の 2026-11-22 の項(`docs/specs/2026-10-07-org-multi-org-director.md:145`)の「読む前と report を書く前の git の状態の比べと `XREVIEW: git_changed` の BLOCKED」は、2 の道を含むどの道にも当てはまる。仕様は `git diff` の失敗の道を書いていないので、この直しで古くなる文はない。項が名指すテスト 4 つは HEAD にある。`/org` skill の「org の終わり方」の 1(`.claude/skills/org/SKILL.md:1386-1389`)も「読む前と report を書く前の git の状態を比べ、違えば report を書かずに `XREVIEW: git_changed`」で、同じことを言う。`diff: failed` と `git diff failed` は、雛形とテストのほかには出てこない(evidence の 10)
4. #363 の取り込み(`6b671276`)。`/director` skill の「分割計画の前の質問」の段落(`.claude/skills/director/SKILL.md:566-580`)は、両側が同じ段落に文を足して衝突した所。解いた段落では、この branch の 2 文(reviewer 座席が自分のモデルでレビューする、違う系列が欲しいなら前文で driver を変える)のあとに main の 2 文(共有のファイルは共有のパスにもできる、lockfile のようなファイルだけを案にする)が続き、「機能の内側の分かれ目は聞かない」で閉じる。両側の挿入はどちらも 1 字も欠けずに入り、ほかに消えた文はない(evidence の 5)。段落の頭は 4 つの分かれ目(境目、予約、driver と model、順)を挙げ、説明は driver と model、共有のファイルの順に来る。この順は main の段落と同じで、この branch は driver と model の説明のあとに 2 文を足しただけ。「この問いでは」が指すのは直前の「座席の driver と model は…聞く」で、取り違えない。この段落を固定する 3 つのテスト(`internal/org/skill_xreview_test.go:707-709`、`skill_split_questions_test.go:282`、`skill_shared_paths_test.go:90`)の語句は、Python のポートで順に見つかった。取り込み全体では、main 側 45 ファイルと branch 側 39 ファイルの `+`/`-` 行が、director の 4 写しを除いて 1 行も違わない(evidence の 4)
5. `eff83438` と `7eb0179d` のテストの固定。書き直した語句(2 の 4 つ、3 の 3 つ、5 の 2 つ、fan-out の規則の 4 つ)が順に出ること、3 が `$TMPDIR` と「まぎれ」を持たないこと、消した古い文 3 つがないことを、`squashSpace` と同じく空白を落とす Python のポートで確かめた(evidence の 6)。Go のテストは走らせていない

### Run 1 の所見の行き先

| Run 1 | 内容 | 直したコミット | HEAD での状態 |
| --- | --- | --- | --- |
| V1(MEDIUM) | `ralph init` の問いの説明 | `7c7cdec5`(説明に 1 文、`ChangeReview` のコメント)と、`0d44799f` が足した台帳の行(556。CLI が 1 つのときに org の設定を聞くかは製品の判断) | 説明は成り立つ。台帳の行は開いたまま |
| V2(LOW) | Trigger に当たって注のない 10 行 | `0d44799f`(13 行に注) | 直った。ただし、この周の `eff83438` と `7eb0179d` が 2 行の Trigger にもう一度当たった(V9) |
| V3(LOW) | 行 343 の古い注 | `0d44799f` | 直った |
| V4(LOW) | `pick_reviewer` の行とテストのコメント | `0d44799f`(行 31 とコメント)、`7eb0179d`(`t.Fatalf` の文字列 2 つ。self-review の C2-1) | 直った |
| V5(INFO) | 行 432 の注の順 | `0d44799f` | 日付順になった(2026-11-21 の #366 の注のあとに 2026-11-22) |
| V6(INFO) | Progress に `f95a08f9` と取り込みの記録がない | Progress の R1 と「main の取り込み」の項 | 直った。この周の直しの記録はまだない(V10) |
| V7(INFO) | AC6 の字面 | なし(PR の本文で書く) | 変わらない |
| V8(INFO) | AC1 の字面 | なし | 変わらない |

### Static analysis (Run 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass(rc=0) | changed の範囲を求めたが、分類されない `scripts/xreview-helpers.sh` があるので full に落ちた(Run 1 と同じ)。shellcheck と hook の `sh -n`、settings の `jq -e`、Codex の hook の guard、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照。Run 1 は取り込みの前の `b98b1b96` で回したので、`6b671276` のあとの木の記録はこれが最初 |
| golang verifier(wrapper の中) | Pass | `gofmt: ok`、`go vet ./...`、`golangci-lint run ./...` は `0 issues.`、`staticcheck ./...` は出力なし。`7eb0179d` が足した `TestRenderRolePrompt_Reviewer_FanOutKeepsTheChildrensFilesOutOfTheWorktree` もここで型検査を通った |
| `scripts/secret-scan-branch.sh`(wrapper の中) | Pass | `ccdf6293..7eb0179d` を origin/main に対して読み、clean |
| `./scripts/plan-visual.sh digest` | `aa9ca6400c54` | 計画の `- Approved:` と一致 |
| 写しの `cmp` | 同じ | org と director の skill の 4 写しずつ |

### Documentation drift (Run 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| reviewer の雛形 | Yes | 上の流れの照合。fan-out の子の作業ファイルの決まりは「座席内 fan-out」の節(`reviewer.md:477-484`)に移り、`.harness/state/` を禁じる理由(座席が 2 と 5 で書くファイルとまぎれうる)が付いた。手順 3 はその節を指すだけ(`:265-267`) |
| leader の雛形、org と director の skill、仕様、`model-routing.md` | Yes | この周の直しで古くなった文はない。`model-routing.md` が名指すテストの名前は、HEAD にすべてある。新しい fan-out のテストはモデルの決め方を固定するものではないので、`model-routing.md` の一覧に入らなくてよい |
| `docs/tech-debt/README.md` | 一部(V9) | 下の表 |
| 計画の Progress | 一部(V10) | 下の表 |

### Findings (Run 2)

番号は Run 1 から続ける。

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V9 | LOW | 台帳の 2 行で、この周の直しが Trigger にもう一度当たったのに、この branch の注がそれを書いていない。(a) 「Seven test gaps of org-cross-review」(432)の (2) の Trigger は「the next change to steps 1 to 3 of the cross-review TASK section of `internal/org/prompts/reviewer.md`」。`eff83438` は手順 2 を、`7eb0179d` は手順 2 と 3(と 5)を変えた。この branch の 2026-11-22 の注は、最後に書いた直し `f95a08f9` について「it changed no step of the reviewer's cross-review TASK section, so (2) was not met again」で終わるので、読むと branch はもう (2) に当たっていないように見える。払っていないことは同じ(`REPORT:` のパスは今も文で確かめる)。(b) 「RC1 (org recheck 2026-10-24)」(377)の Trigger は「The next change to the implementer or reviewer role prompt」で、2 本とも `reviewer.md` を変えた。注の「599 → 507 lines, counted at origin/main ccdf6293 and at 7c7cdec5」は `7c7cdec5` の数としては正しいが、HEAD では 517。払っていないことは同じ(座席に agmsg のパスを渡していない)。Trigger の照合でほかに当たったのは「org runtime FR-7 quality gate is prompt-instructed」(161)だけで、その注の「the comparison of the git state … runs before the seat reads and before it writes its report」は 2 の道も覆うので、足すことはない | `docs/tech-debt/README.md:432` と `:377` の 4 列目。evidence の 8 と 9 | 2 周目の /sync-docs で、2 行の 2026-11-22 の注に 1 文ずつ足す。432 には、`eff83438`(cross-review の指摘: 失敗した `git diff` も 5 の比べを通る)と `7eb0179d`(self-review の C2-2 と C2-3)が手順 2、3、5 を変えて (2) にもう一度当たり、答えは同じ、と書く。377 には、2 本が `reviewer.md` をもう一度変えた(手順 2、3、5 と、fan-out の節の子の作業ファイルの規則。`7eb0179d` で 517 行)が、RESULT と BLOCKED の送り方の話ではない、と書く |
| V10 | INFO | 計画の Progress に、cross-review の直し `eff83438` と、self-review Run 2 の LOW の直し `7eb0179d` の記録がない。Implementation の項は V1 の直し `7c7cdec5` で終わり、Review artifact の行は Run 2 の C2-1〜C2-3 を挙げるが、直したことは書かない。`eff83438` は、計画の Scope の「読めなければ `incomplete (git diff failed: <要点>)`」と AC2 の字面に、「`git: same` のとき」という条件を足した。AC1 の比べを先に通すためで、AC の文は digest に入るので変えられない | `docs/plans/active/2026-11-22-org-xreview-own-model.md:22`、`:72`、`:123-131`。`docs/reports/cross-review-triage-org-xreview-own-model.md` の ACTION_REQUIRED 1 | Progress に 2 本の直しを 1 行ずつ足し、`eff83438` の行に AC2 の条件のことを書く。PR の本文でも、AC2 は AC1 の比べのあとで満たす、と書く |
| V11 | INFO | 手順 3 の「読み終えられなければ」の道は「出した所見も triage せずに 5 に進み、6 で report を `Reviewer status: incomplete (<理由>)` で書いて、7 のとおり…返す」と言い、5 の `git: changed` の分かれを書かない。`eff83438` で 2 の道は両方の分かれ(`git: changed` なら 5 のとおり BLOCKED、`git: same` なら 6)を書いたので、2 つの道の書き方がそろっていない。5 自身が「`git: changed` なら…report を書かず」と言うので、座席の動きは食い違わない | `reviewer.md:268-271`(3)、`:241-248`(2)、`:306-314`(5) | なし。次に手順 3 を直すときに、2 と同じく「`git: same` なら 6 で」と書けばそろう |
| V12 | INFO | self-review Run 1 の L1(model が `""` の座席で `Cross-review: ran (reviewer 座席 reviewer(codex、(CLI default)))` と括弧が重なる)は直っておらず、台帳に行がない。self-review Run 2 は、直さずに `/pr` に進むなら台帳に「Wording leftovers」の行を足す案を出した。案の (1)〜(3) にあたる C2-1〜C2-3 は `7eb0179d` で直ったので、残るのは (4) だけ | `docs/reports/self-review-2026-11-22-org-xreview-own-model.md`(Run 2 の Tech debt identified)、`internal/org/prompts.go:147` | `/pr` か 2 周目の /sync-docs で、(4) だけの行を足す。周の上限に届いているので、雛形を直すなら記録だけの修正の外になる |

### Coverage gaps (Run 2)

- テスト(`go test ./internal/org/ ./internal/cli/`、`./scripts/run-test.sh`)は走らせていない。`/test` の仕事。`eff83438` と `7eb0179d` の語句は Python のポートで確かめただけで、Go のテストの結果ではない。Run 1 の `/test`(HEAD `7c7cdec5`)は取り込みの前の木で回したので、取り込みのあとの木で `run-test.sh` を回すのは 2 周目の `/test` になる
- 座席の中で、`git diff` が失敗したあとに HEAD が動く場面は回していない。文の流れで確かめた
- 台帳の Trigger の照合は、この周のコミットが触ったファイルと節の名前で行った(evidence の 8)。「The role prompts grew with org-cross-review」(431)の Trigger は「adds a section」で、`7eb0179d` は既存の節に項目を足しただけなので当たらないと読んだ。「`internal/org/verbs.go` is 2277 lines」(370)の Trigger は leader の雛形のテストで、この周に変わったのは reviewer のテストだけなので当たらないと読んだ

## Verdict

- Verified: AC1〜AC5、AC7、AC8 は、この周の変更のあとも成り立つ。AC2 は `eff83438` で AC1 の比べのあとに満たす形になった(V10)。AC6 は振る舞いで満たす(Run 1 のまま)。手順 2・3・5・6・7 の流れ、leader の `git_changed` と incomplete の扱い、仕様の 2026-11-22 の項、`/org` skill は食い違わない。#363 の取り込みは両側の変更を保ち、`/director` skill の段落は読める。静的解析は、取り込みのあとの木で full の範囲で pass。計画の digest は一致
- Partially verified: AC6 と AC1 の字面(V7、V8)。台帳の 2 行の注(V9)
- Not verified: テストの実行と、座席の中での cross-review の実行
- Verdict: pass(LOW 1、INFO 3。止める指摘はない。V9 は 2 周目の /sync-docs に、V10 は計画の Progress と PR の本文に、V12 は `/pr` に渡す)

---

## Run 1 (cycle 1、履歴)

> 1 周目(HEAD `b98b1b96`)の記録をそのまま残し、見出しだけ 1 段下げた。今の判定は上の `## Verdict`。

- Date: 2026-11-22(JST。pipeline の 1 周目、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(issue #360)。`- Approved:` の digest `aa9ca6400c54` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff origin/main...HEAD`(HEAD `b98b1b96`、origin/main `bd079dbb`、36 ファイル、+2039 −1817)。HEAD は origin/main(#362 と #366 の merge まで)を取り込んだ merge commit で、取り込みの衝突は `docs/tech-debt/README.md` の 1 行だけ。self-review のあとの直し `f95a08f9`(M1、L1、L3、L4)を含む。テストは走らせていない(`/test` の仕事)
- Evidence: `docs/evidence/verify-2026-11-22-org-xreview-own-model.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。`run-static-verify.sh` の出力のあとに、AC6 の確かめ、写しの `cmp`、消した語の grep と Python のポート、台帳の Trigger の照合、digest を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-21-213815.log`(UTC の日付)

CRITICAL と HIGH はない。MEDIUM 1(V1)、LOW 3(V2〜V4)、INFO 4(V5〜V8)。

### Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | Met | `internal/org/prompts/reviewer.md` の「cross-review の TASK」(`:164-391`)は、座席が自分で差分と変わったファイルを読んで所見を出し(手順 3、`:247-270`)、triage する(手順 4)。`codex exec`、`claude -p` は、呼ばないと言う 1 文(`:171-172`)にだけ出る。この 1 文と、前の版の行を読まない 1 文(`:182-184`)を除くと、`pick_reviewer`、`mcp list`、`sleep 1200`、`xreview-helpers.sh`、`-out.md`、`相手`、`20 分`、`skipped` のどれも節にない(evidence の 7、Python のポートで確かめた)。手順 2 が読む前の git の状態を `-git-before.txt` に書き、手順 5 が report を書く前に `-git-after.txt` と `cmp` で比べ、違えば report を書かずに `XREVIEW: git_changed` の BLOCKED を返す(`:294-312`)。git の状態を取る式は main の版と 1 字も変わらない。固定するテストは `TestRenderRolePrompt_Reviewer_XreviewReviewsOnTheSeatsOwnModel` と `..._XreviewComparesTheGitStateBeforeTheReport` ほか。字面の「持たない」は、Scope が求める禁止の 1 文を除いて読む(V8) |
| AC2 | Met | 手順 1(`:188-197`)が確かめるのは `XREVIEW_BASE:`、`CYCLE:`、`REPORT:` の 3 行だけ。`IMPLEMENTER_DRIVER:`、`REVIEWER_CODEX:`、`REVIEWER_CLAUDE:` は読まないと書く(`:182-186`)。`diff: failed` なら `Reviewer status: incomplete (git diff failed: <要点>)` の report を書き、同じ値の RESULT を返す(`:241-245`)。`.harness/state/` は root と雛形の `.gitignore` の両方で無視される(`git check-ignore -v` で確かめた)ので、手順 2 が書く `.diff` は手順 5 の比べに入らない |
| AC3 | Met | report の見出し `# Cross-review triage report`、`Triager:`、`Reviewer status:`、`HEAD:`、`Base:`、`Cycle:`、`Total reviewer findings:`、`After triage:` の行、3 つの表(列も同じ)は main の版のまま(`:325-358`)。`- Driver: ...  Reviewer: ...` の行は `- Reviewer: reviewer 座席 {{SEAT_ID}}({{DRIVER}}、{{MODEL}})` に、`Reviewer output:` の行は `Diff:` の行に替わった。`Reviewer status:` から `skipped` が消えたのは Scope の「相手を呼ぶための skipped を消す」のとおりで、leader は前の版の reviewer のために `skipped` の扱いを残す(`leader.md:1280-1283`)。返事は `pass`、`action_required`、`incomplete`、`git_changed` の 4 つ(`:360-367`) |
| AC4 | Met | `RolePromptVars` に `Driver` と `Model`(`internal/org/prompts.go:76`、`:82`)、置き換えは `{{DRIVER}}` を `vars.Driver`、`{{MODEL}}` を `ModelOrCLIDefault(vars.Model)` に(`:146-147`、空なら `(CLI default)`)。`Spawn` と `dryRunSpawn` の 2 か所が `p.Driver` と `p.Model` を渡す(`spawn.go:1120`、`:2423`)。どちらも `AgentStart` と `--model` に渡す値と同じで、driver と model は CLI が `ResolveRoleModel` で決めてから `Spawn` に入る。雛形で置き場を持つのは `reviewer.md` の 2 行だけ(`:4`、`:329`)。`RenderRolePrompt` のほかの呼び出しは `reseat.go:920` で、空の vars で描いて文字列を捨てる。テストは `TestRenderRolePrompt_Reviewer_ShowsTheSeatsDriverAndModel`、`TestRenderRolePrompt_SeatTemplates_NoDriverOrModelPlaceholderLeft`(3 役 × model あり・なし)、`TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel` |
| AC5 | Met | `leader.md` の `### cross-review`(`:1043`〜)は、TASK に 3 行だけを書き、implementer の driver を台帳からも記憶からも読まないと言う(`:1232-1241`)。台帳を読むコマンドと `driver: error` の扱いは消えた。小節のコードブロックを main と比べると、状態のコマンドと上限の QUESTION は 1 字も変わらず、消えたのは driver のコマンド、変わったのは TASK の例だけ(evidence の 8)。`XREVIEW: git_changed` は先に進まず BLOCKED を上げ、戻したら送り直しの上限の中で同じ `<k>` を送り直す(`:1294-1305`)。`Cross-review:` の値は最後の report の `Reviewer status:` から決め、`ran (<report の Reviewer:>)` と書く(`:1315-1320`) |
| AC6 | Met(振る舞い)。字面は満たさない(V7) | 標準フローの `/cross-review` の skill は 4 写しとも差分に入らない。`scripts/xreview-helpers.sh` と写しは差分に入るが、変わった行はすべて `#` で始まり(`#` で始まらない変更行は 0)、コメントを除いた本体は main と同じ。root と雛形の写しは `cmp` で同じ。`tests/test-xreview-helpers.sh` は差分に入らず、その対象の関数(`pick_reviewer`、`count_triage_findings`、`detect_base_branch`)も変わらない。直したのは `pick_reviewer` の doc comment の呼び出し元の一覧で、reviewer 座席を外した。残すと「4 つ目の呼び出し元」という誤りが残る。この判断は Progress の S4 に記録がある |
| AC7 | Met | `model-routing.md`(2 写し。雛形との差は既存の KNOWN_DIFF)、org と director の skill(4 写しずつ `cmp` で同じ)、recipe 2 つ(2 写しずつ同じ)、README、仕様の 2026-11-22 の項目、`config.go` と `paneenv.go` と `spawn.go` のコメント、`templates/base/ralph.toml` のコメントが新しい動きを書く。`model-routing.md` が名指すテストの名前は 9 つともある。`check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh`(14 skill)は wrapper の中で通った。`ralph init` の問いの文は Scope の一覧の外で、古いまま(V1) |
| AC8 | Met | `docs/recipes/codex-seat-permissions.md:197-201` が、`workspace-write` には許すホストの一覧がなく座席はどのホストにも出られること、閉じるには `codex_network_access = false` にして座席を立て直すこと(「A change reaches only the seats started after it」を指す)を書く。雛形の写しと同じ |

ユーザーの決定(reviewer 座席が、系列にかかわらず自分のモデルでレビューする)との照合: reviewer の雛形、leader の雛形、2 つの skill、`model-routing.md`、recipe、README、`templates/base/ralph.toml` のどれも、系列で扱いを変える文を持たない。違う系列の目が欲しいときの手(`[org.role_models].reviewer` か分割計画の前文で reviewer の driver を変える)だけを書く。recipe と org skill の「既定の 2 組では implementer と reviewer が同じ系列」は `DefaultRoleModels`(`internal/config/role_models.go:83-94`)と合い、skill の例の `gpt-6.1-sol` は `defaultCodexRoleModel` と同じ。

#366 の取り込みとの照合: `leader.md` に入った人の決定の文(立て直しの規則の `:156-161`、手順 4 の `:492-496`、上限の答えの読み方の `:1188-1190`、運用規律の項の `:1615`〜)は、cross-review の上限の QUESTION を「人の決定はこの答えではない」と扱う。この branch は上限の QUESTION と、その答えの読み方を変えていないので食い違わない。cross-review の返事を待つあいだに人の決定が worktree を変えさせても、reviewer の手順 5 が `git_changed` で拾い、見ていない変更に pass は付かない。この待ちの決まりは main にもあり、この branch が作ったものではない。`Seven test gaps of org-cross-review` の行の衝突の解き方は、branch の行に main の 2 つの挿入を足した形で、どちらの変更も落ちていない(evidence の 10)。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass(rc=0) | 既定の changed の範囲を求めたが、分類されない `scripts/xreview-helpers.sh` があるので full に落ちた。shellcheck と `sh -n`(hook)、`jq -e`(settings)、Codex の hook の 3 つの guard、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照 |
| golang verifier(wrapper の中) | Pass | `gofmt: ok`、`go vet ./...`、`golangci-lint run ./...` は `0 issues.`、`staticcheck ./...` は出力なし |
| `scripts/secret-scan-branch.sh`(wrapper の中) | Pass | `bd079dbb..b98b1b96` を origin/main に対して読み、clean |
| `./scripts/plan-visual.sh digest` | `aa9ca6400c54` | 計画の `- Approved:` と一致 |
| 写しの `cmp` | 同じ | org と director の skill の 4 写しずつ、recipe 2 つの 2 写しずつ、`xreview-helpers.sh` の 2 写し |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| reviewer と leader の雛形 | Yes | 上の AC1〜AC5。小節の外の手順 9、12、14 と立て直しの規則(座席の確認の項)の「相手」「20 分」も直っている。`leader.md` に残る「相手」「20 分」「`pick_reviewer`」は `### 計画助言`(`:917-1042`)だけで、Non-goals のとおり |
| org と director の skill | Yes | cross-review の説明、`IMPLEMENTER_DRIVER:` の行、`pick_reviewer` の言及、待ちの例外を直した。cross-review の停滞は、画面が動いているあいだだけ待ち、10 分動かなければ上げる、で `leader.md:766-774` と合う(`f95a08f9`) |
| `model-routing.md`、README、recipe、仕様、`templates/base/ralph.toml`、Go のコメント | Yes | 上の AC7。仕様の 2026-11-01 の項と 2026-10-29 の項に残る「違う系列」「`pick_reviewer` に座席の driver を読ませる」は、日付つきの当時の記録で、2026-11-22 の項が変えたと書く |
| `ralph init` の「変更のレビュー」の問い | No(V1) | 下の表 |
| `docs/tech-debt/README.md` | 一部(V2〜V5) | この branch は 2 行に 2026-11-22 の注を足し、新しい行(計画助言の入れ子の CLI)を足した。Trigger に当たったまま注のない行が 10 ある。行番号は HEAD のもので、行の頭の語も書く |
| `internal/cli/org_runtime_env_test.go:182-185` のコメント | No(V4) | self-review の L2。/sync-docs に回してある |
| 計画の Progress | 一部(V6) | 下の表 |

### Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | MEDIUM | `ralph init` の「変更のレビュー」の問いが、機能の org の cross-review を「もう一方の AI のレビュー」と説明したまま。問いの答えは `[standard].cross_review` と `[org.review].cross_review` の両方に書かれる。この branch のあと、org の cross-review は reviewer 座席が自分のモデルで行い、既定の 2 組では implementer と同じ系列になるので、説明の「Claude Code reviews the changes Codex makes, Codex reviews Claude Code's」は org の側では当たらない。CLI が 1 つのとき `[org.review].cross_review` を false で書く決まり(仕様の 2026-11-02 の項)も、相手の CLI が要るという前提を失った。計画の Scope の文書の一覧に入っておらず、AC7 の字面には当たらない | `internal/cli/init_settings.go:331-336`(問い)、`:43-44`(「Both are false with one CLI」)、`:158-160`。仕様 `docs/specs/2026-10-07-org-multi-org-director.md:149` | 問いの説明に、機能の org では reviewer 座席が自分のモデルでもう一度レビューする、と 1 文足す(Go の文字列 1 つで、説明の文を固定するテストはない)。CLI が 1 つのときに org の cross-review を聞くかは製品の判断で、Non-goals(`[org.review].cross_review` の意味と既定)の外なので、直さないなら tech-debt に行を足す |
| V2 | LOW | 台帳の Trigger の照合: この差分が当たる Trigger で、この branch の注がない行が 10 ある。「org runtime FR-7 quality gate is prompt-instructed」(161、org の品質パイプラインの変更)、「What a leader is told about its reservation」(190 の (a)、`RolePromptVars` と leader の雛形)、「Three test gaps of the org-director-attach PR」(267、director skill の `## 受信箱の処理` の中の「件の処理」と「cross-review の上限」)、「`internal/org/verbs.go` is 2277 lines」(370、leader の雛形のテスト。`prompts_test.go` 2727→2787、`spawn.go` 3122→3127 で +300 には届かない)、「The recipes under `templates/base/docs/recipes/`」(376、recipe 2 つの編集。`codex-seat-permissions.md` の `docs/evidence/` の引用は 4→3 で、行の最後の注の「4 citations of 4 files」が古くなった)、「RC1 (org recheck 2026-10-24)」(377、reviewer の雛形)、「RC4」(382、leader の機能ごとの org の手順 9、12、14)、「After step 14 of the leader template」(389、手順 14 の PR の本文と RESULT の例)、「Five files went past the 800-line guide」(420、`internal/config/config.go` 1219→1221、コメントだけ)、「The section 「agmsg の受信箱の運用」」(528、立て直しの規則の座席の確認の項) | `docs/tech-debt/README.md` の各行の 4 列目。evidence の 9 | /sync-docs で、各行に「2026-11-22: fix/org-xreview-own-model (#360) hit this trigger with ...、but did not pay it」の注を足す(払うものはない。どれも Scope の外) |
| V3 | LOW | 「A Codex reviewer that runs the gate cannot be reached」(343)の 2026-11-01 の注は、Codex の reviewer が cross-review で相手の呼び出しを前面で最大 20 分走らせ、leader の雛形と 2 つの skill がその停滞を gate のときと同じに扱う、と行を広げた。この branch でその呼び出しはなくなり、`f95a08f9` で cross-review の停滞は gate と同じには扱わなくなったので、広げた分が今の動きと合わない。Trigger には当たらない | `docs/tech-debt/README.md:343` の 4 列目 | /sync-docs で、広げた分が 2026-11-22 になくなったことを 1 文足す |
| V4 | LOW | self-review の L2 が残っている。`pick_reviewer` の行(31)の 2 列目が「The fourth caller, a feature org reviewer seat's cross-review」を説明し、4 列目も呼び出し元に数える。`internal/cli/org_runtime_env_test.go:182-185` のコメントは「a reviewer seat's cross-review takes the implementer's driver from its TASK」と言う | `docs/tech-debt/README.md:31`、`internal/cli/org_runtime_env_test.go:182-185` | 計画の S4 のとおり /sync-docs で直す。行 31 は 4 つ目の呼び出し元の文を消すか、2026-11-22 に消えたと書く |
| V5 | INFO | 「Seven test gaps of org-cross-review」(432)の 4 列目で、この branch の 2026-11-22 の注が、取り込んだ #366 の 2026-11-21 の注より前に並ぶ。取り込みの衝突をこの順で解いたため。中身は両方そろっている | `docs/tech-debt/README.md:432` | 直すなら /sync-docs で 2 つの注を日付順に入れ替える |
| V6 | INFO | 計画の Progress に、self-review のあとの直し `f95a08f9`(M1、L1、L3、L4)と、origin/main の取り込み `b98b1b96`(#362 と #366、台帳の 1 行の衝突)の記録がない。Risks の「main を取り込む」の扱いは、「順番の入れ替え」の項が予定として書くだけ | `docs/plans/active/2026-11-22-org-xreview-own-model.md:125-128` | Progress に 1 行足す |
| V7 | INFO | AC6 の「(差分に入らない)」と Verify plan の「`git diff main...HEAD -- .claude/skills/cross-review scripts/xreview-helpers.sh` が空」は、字面では満たさない。差分はコメントだけで、振る舞いは変わらない(上の AC6 の行) | evidence の 2 | AC の文は変えない(変えると digest が変わる)。PR の本文で、AC6 はコメントだけの差分で振る舞いを変えずに満たしたと書く |
| V8 | INFO | AC1 の「`codex exec`、`claude -p` … を持たない」は、Scope の「`codex exec`、`claude -p`、そのほかの CLI で別のモデルを呼ばないと明記する」と字面でぶつかる。雛形はその 1 文だけに 2 語を持ち、テスト(`reviewerXreviewGone`)もその 1 文を除いてから探す | `reviewer.md:171-172`、`internal/org/prompts_xreview_reviewer_test.go:31-50` | なし(Scope が優先する読み方で満たす) |

### Observational checks

- 実機の座席での cross-review は走らせていない。Verify plan の「描いた雛形の該当の節(テストの出力)」は `/test` の出力に任せる。ここでは雛形の文を読み、Python のポート(空白をつぶし、許した 2 文を除いてから語を探す)で消した語の不在を確かめた
- `Seven test gaps` の行と `has not run inside a seat` の行の 2026-11-22 の注は、何が閉じて何が残るかを書いている。後者の Impact の列は古い文のまま残るが、注が「the impact's sentence ... no longer apply」と書く

### Coverage gaps

- テスト(`go test ./internal/org/ ./internal/cli/`、`tests/test-xreview-helpers.sh`、`tests/test-agent-models.sh`)は走らせていない。`/test` の仕事
- 座席の中で、新しい雛形の reviewer が差分を読んで report を書く流れ、新しい leader と前の版の reviewer が混ざったときの QUESTION、前の版の leader の 5 行の TASK を新しい reviewer が読まずに進むことは、文とテストの文字列でしか確かめていない(計画の Assumptions のとおり、release ノートで reseat を勧める)
- 台帳の Trigger の照合は、Trigger の列の語とこの差分のファイルと関数の名前を突き合わせたもので、文の意味から当たるかを決めた行が 3 つある(161 の「org の品質パイプライン」、370 の「leader の雛形のテスト」、528 の「座席の確認の項」)。「`placeSeatPane` (`internal/org/spawn.go`) is 89 lines」(448)の「the spawn saga」は、`Spawn` の中の雛形の vars の 1 行で当たるとは読まなかった。「Small findings of org-feature-worktree」(202)の (g) は閉じた項目で、数えなかった
- 足すとよい決定論の確かめ: `ralph init` の問いの説明と、それが書く設定の意味を突き合わせるものはない。V1 のような文のずれは、`initSettingsQuestions` の説明を README か仕様の 1 文と比べるテストで拾える

### Verdict

- Verified: AC1〜AC5、AC7、AC8(ソースと文書で)。AC6 は振る舞いで(コメントだけの差分、本体は main と同じ)。静的解析は full の範囲で pass。計画の digest は一致。skill と recipe と `xreview-helpers.sh` の写しは一致。#366 の人の決定の文と cross-review の小節は食い違わない。merge の衝突の解き方は両方の変更を保つ
- Partially verified: AC6 の字面(V7)。文書は、`ralph init` の問い(V1)と台帳の注(V2〜V4)を除いて合う
- Not verified: テストの実行と、座席の中での cross-review の実行
- Verdict: pass(MEDIUM 1、LOW 3。止める指摘はない。V1 は PR の前に直すか tech-debt に送るかを決め、V2〜V4 は /sync-docs に渡す)
