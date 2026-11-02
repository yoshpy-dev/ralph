# Verify report: org-cross-review

- Date: 2026-11-01(cycle 1)、2026-11-02(cycle 2)
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(issue #269、AC1〜AC9 と AC5b、D1〜D10 と D6b・D7b)
- 構成: 先頭が cycle 2 の節と、cycle 2 の判定の `## Verdict`。そのあとが cycle 1 の節で、見出しを 1 段下げて「(cycle 1)」を足し、cycle 1 の `## Verdict` を `### Verdict(cycle 1)` にしたほかは cycle 1 のまま。この file の `## Verdict` は cycle 2 の 1 つだけ
- Evidence: `docs/evidence/verify-2026-11-01-org-cross-review.md`(cycle 1 の節と、最後の「Cycle 2」の節)

## Cycle 2(HEAD 5569729d)

### Scope(cycle 2)

- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 2。`cycle-count.json` は `"cycle": 2`)
- ブランチ全体: `git diff main...HEAD`(merge-base 4fef6b8c、HEAD 5569729d、47 ファイル、+6175/−243)。重点は cycle 1 の verify の HEAD d8e02c1d からの差分(24 ファイル、+1791/−171): F5 62d62be0(cycle 1 の V1・V2)、F6 17fc2b52、F7 4d289952(cross-review cycle 1 の ACTION_REQUIRED #1・#2)、F8 7724f6af(self-review cycle 2 の C2-1〜C2-4)、文書のコミット(sync-docs の de47b7a0、2e75c82a、計画の b3750ea6 と 5569729d)
- 見たもの: 計画の AC と D1〜D10・D6b・D7b を、雛形・skill・文書と突き合わせた。leader・reviewer・`/director`・`/org` の文が食い違わないか、仕様・README・`docs/reports/README.md` と写し・recipe・model-routing.md と写し・tech-debt の台帳がずれていないかも見た。F6 と F7 のコマンドは雛形から抜き出し、`/bin/sh`(bash 3.2.57)と `zsh -f`(5.9)で打った。`ralph` は stub と、worktree から `go build` したものを使い、codex と claude は呼んでいない
- 打っていないもの: `go test` と `./scripts/run-test.sh`(/test の範囲)
- handoff は base を「main 4fef6b8c(origin/main と同じ)」と書くが、origin/main はそのあと 31812a3b(PR #297)に進んだ。この回は merge-base 4fef6b8c の上で見た(V2-2)

### 前の回の所見と直しの扱い(cycle 2)

| 所見 | 扱い | 根拠 |
| --- | --- | --- |
| verify cycle 1 の V1(LOW、返事の例の TASK_ID) | 解決(F5) | `internal/org/prompts/reviewer.md:445`・`:457` は `TASK_ID: xreview-{{ORG_ID}}-c1` |
| verify cycle 1 の V2(LOW、report の `- HEAD:` の値を形を見ずに git に渡す) | 解決(F5) | `internal/org/prompts/leader.md:810` は、値が 40 字か 64 字の 16 進のときだけ `--end-of-options` のあとで `git diff` に渡す。probe で、短い SHA と `--output=<file>` はどちらも `since: changed` になり、ファイルはできなかった(evidence の「leader の `since:`」) |
| cross-review cycle 1 の ACTION_REQUIRED #1(`docs/reports/templates/` の変更が `since: same` になる) | 解決(F7) | pathspec は `':(exclude,glob)docs/reports/*.md'`(`leader.md:810`、説明は `:862-867`)。probe で、`docs/reports/templates/t.md`・`docs/reports/sub/y.md`・`docs/reports/notes.json` のコミットは changed、直下の report と `docs/reports/README.md` のコミットは same |
| cross-review cycle 1 の ACTION_REQUIRED #2(git が許すブランチ名を断る) | 解決(F7)。残る範囲は Info | `reviewer.md:185-208`。probe で、`release+stable`・`release=stable`・`release(v2)`・`dévelop`・`a@b` は `base: ok`、先頭の `-`・`..`・`@{-1}`・`main@{u}`・空白・タブ・引用符・`$`・`` ` ``・`\`・コマンド置換を含む値は `base: invalid` で、コマンド置換は実行されなかった |
| self-review cycle 2 の C2-1(tech-debt の行の (6)(7) が F7 で古くなった) | 解決(F8) | `docs/tech-debt/README.md:417` は 7 項目になり、(2) は `XREVIEW_BASE:` をコマンドが確かめると書く。行が挙げるテスト名 6 つと `leaderXreviewStatusJSON`・`advisoryShells`・`fx.checkBase` は `git grep` で見つかり、`reviewerXreviewBaseOK` は 0 件 |
| C2-2(heredoc が閉じない打ち方が QUESTION になる) | 解決(F8) | QUESTION は `base: invalid` のときだけ(`reviewer.md:165-168`)、どちらの行も出なければ打ち直す(`:197-200`)。probe で、5 行すべてに空白を付けた形と区切りの行だけに付けた形は何も出さず、値の行だけに付けた形は `base: invalid` を出した |
| C2-3(送り直しの上限が何を数えるか) | 解決(F8) | `leader.md:946-960`(QUESTION のあとと `git_changed` のあとを合わせて `-r2`・`-r3` まで。`-r3` から先は BLOCKED で止まり、人の DECISION のときだけ `-r4` から続ける)、`git_changed` の枝 `:1054-1065`、QUESTION の枝 `:1066-1072`。3 か所の言い方は食い違わない |
| C2-4(コメントと文書の折り返し、値を写す所の書き漏れ) | 解決(F8) | `internal/config/config.go:70-84`・`:827-832` と、`.claude/rules/ralph/model-routing.md:80-95` と写しの同じ範囲に、80 字を超える行はない。`config.go:78-79` は値を写す所に reviewer 座席を挙げる |

### Spec compliance(cycle 2)

| Acceptance criterion | Status | この回の差分と evidence |
| --- | --- | --- |
| AC1 off なら TASK を送らず `Cross-review: skipped (<理由>)` | met | この回の差分は触っていない(`leader.md:797-799`、`:815-828`) |
| AC2 on なら手順 9 の rebase のあと・sync-docs の前に `xreview-` の TASK | met | TASK の行は `leader.md:964-976`。`IMPLEMENTER_DRIVER:` は記憶でなく台帳から読む(F6、`:978-1001`)。worktree から build した `ralph` の `org status --json` に打つと、implementer が claude なら `driver: claude`、codex の座席を止めたあとも `driver: codex` を出した。driver が 2 つ、implementer がない、driver がない、`gemini`、`ralph` がない、jq がないときは、すべて `driver: error (<理由>)` で、TASK を送らず BLOCKED を上げる枝に入る(evidence の「leader の driver のコマンド」) |
| AC3 `pick_reviewer <implementer の driver>` で違う系列を選ぶ | met | `pick_reviewer "<IMPLEMENTER_DRIVER>"`(`reviewer.md:222`)は変わっていない。渡す driver の出どころが台帳になった |
| AC4 report、BLOCKED と RESULT、skipped と incomplete | met | report の形と返事は変わっていない。手順 1 の QUESTION は `base: invalid` のときだけになった(F8)。所見のパスは worktree の root からの相対で書く(F6、`reviewer.md:387-391`) |
| AC5 直しのループ、上限の QUESTION、cap のファイル | met | この回の差分は触っていない(`leader.md:880-944`) |
| AC5b TASK の前に上限を確かめ、見せていない変更があれば聞く | met | `since:` は `docs/reports/` 直下の `.md` のほかの変更で changed になる(F7)。probe の表のとおり |
| AC6 cross-review の TASK の間の停滞の ALERT | met | この回の差分は触っていない |
| AC7 雛形のテスト、写しの検査、文の固定のテスト | partly | 静的の 3 つ(check-skill-sync、check-sync、check-template-purity)は run-static-verify の中で通った。F7・F8 を打つテスト(`TestRenderRolePrompt_Reviewer_XreviewBaseCheckRunsGitCheckRefFormat`、`TestRenderRolePrompt_Leader_XreviewStateCommandSeesATemplateChange` など)と句の固定はあるが、走らせるのは /test |
| AC8 設定と skill の文書 | met(仕様の 1 文を除く) | `config.go` と `templates/base/ralph.toml` のコメント、model-routing.md と写しは reviewer 座席の cross-review を書く。仕様 `:133` の上限の条件だけが F7 の前の言い方(V2-1) |
| AC9 本物の claude と codex で両方向 | met | 相手を呼ぶ codex と claude のコマンド(`reviewer.md:274-284`、`:308-315`)は、2 回目の本物の CLI での確かめ(62d62be0、記録は afccca76)から変わっていない。`git diff 62d62be0 HEAD -- internal/org/prompts/reviewer.md` で変わったコマンドの行は、F7 が手順 1 に足した base の確かめの 3 行だけ。cycle 1 の Coverage gaps の「今の版の live の通しはない」は、呼び出しのコマンドについては解消した。座席の中での実行は known gap のまま |

#### Design decisions(cycle 2)

| 決定 | 状態 | この回の差分と evidence |
| --- | --- | --- |
| D5 TASK の形、欠けた行の QUESTION、`git_changed` | met | 送り直しの番号と上限(`leader.md:946-960`)、欠けた行と形の違う値の QUESTION(`reviewer.md:165-183`)。計画の D7b は `-r2`・`-r3` だけを書き、F8 の上限は Progress(`:147`)にある(Info) |
| D6・D6b 相手の呼び方 | met | 呼び出しのコマンドは変わらず、手順 1 に base の確かめ(`reviewer.md:185-208`)が入った。claude の `git diff` に `--end-of-options`(`:311`、F5) |
| D7・D7b 上限、`since:`、送り直し | met | `since:` の除外は D7b(計画 `:62`)の「`docs/reports/` 直下の `.md`(生成された記録)の外」と同じ。report の `HEAD:` の値は形を確かめてから使う(F5) |
| D1〜D4、D8〜D10 | met | この回の差分はこれらの文を触っていない。D8 の report のコミットの前に、`REPORT:` のパスの形と番号を確かめて `git add --` でそのファイルだけを足す文が入った(F5、`leader.md:1029-1038`) |

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc=0) | `All verifiers passed.`。`unclassified:scripts/xreview-helpers.sh` で full fallback になり golang が走った。check-sync は IDENTICAL 172・DRIFTED 0・ROOT_ONLY 0・KNOWN_DIFF 6、check-pipeline-sync ok、check-skill-sync 14 skill、check-template-purity PASS、tech-debt の plan の参照 OK、`gofmt: ok`、golangci-lint `0 issues.`、secret scan `4fef6b8c..5569729d against origin/main: clean`。ログは `docs/evidence/verify-2026-11-02-002630.log`(gitignore されるのでコミットしない) |
| `go vet`(`internal/org/`、`internal/config/`、`internal/cli/`、`internal/org/driver/`)/ `gofmt -l internal cmd` | pass | どちらも出力なし |
| `git diff --check main...HEAD` | pass(rc=0) | 出力なし |
| `./scripts/plan-visual.sh digest <plan>` | `ce18b629320e` | 計画の `- Approved:` と同じ。F8 のあとの 5569729d は Progress の 1 行だけを足した |
| 写しの `cmp` | 同じ | director と org の skill の写し 3 つずつ、`scripts/xreview-helpers.sh`、`docs/reports/README.md`、2 つの recipe |
| `git merge-tree --write-tree --name-only main HEAD` | conflict | `docs/tech-debt/README.md`(V2-2) |

### Documentation drift(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md:133` | 1 文ずれる | F6 の driver の読み方と、このブランチが足した report だけを数えることは書いてある(de47b7a0)。上限の QUESTION の条件「その回の report の `HEAD:` から `docs/reports/` の外が変わっていれば」は F7 の前の言い方(V2-1)。F7 の base の確かめと F8 の送り直しの上限は書いていないが、注は詳細を計画に送る要約なので所見にしない |
| `/org` skill(4 つ) | yes | 終わり方の 1 に driver の読み方と数える report(de47b7a0)。送り直しの上限や `since:` の細部は書かず、雛形と食い違う文はない |
| `/director` skill(4 つ) | yes | この回の差分は触っていない。送り直しの上限で leader が上げる BLOCKED は、director のふつうの BLOCKED の扱いで人に届く |
| `internal/config/config.go:70-84`・`:827-832`、`templates/base/ralph.toml:62-78` | yes | reviewer 座席の cross-review(leader が TASK で値を渡す)を書く |
| `.claude/rules/ralph/model-routing.md:85-93`・`:126-129`・`:202` と写し | yes | 写しの同じ段落は meta-repo の文のほかは同じ(check-sync の KNOWN_DIFF)。`:202` のテスト名 3 つは `git grep` で見つかる |
| README `:284`、`docs/reports/README.md` と写し、2 つの recipe と写し | yes | この回の差分は触っていない。README の説明は F6〜F8 のあとも雛形と合う |
| `docs/tech-debt/README.md:416` | info | 行数は 433df4ae の数(1,298・542・1,449・1,268)。HEAD では 1,311・565・1,515・1,309。行は数えたコミットを書くので誤りではない |
| `docs/tech-debt/README.md:417`・`:418` | yes | C2-1 の扱いのとおり |
| tech-debt の Trigger | 新しい発火なし | Trigger の欄を python で読み、leader.md・reviewer.md・config.go・model-routing・cross-review を含む 23 行を拾った。このブランチが注を足していない 7 行(156、188、246、251、303、414、415)の Trigger は、このブランチが触っていない段落・関数・ファイルを名指す(156 は model-routing.md の pin の段落 `:24` で、このブランチの変更は `:85-93`・`:126-129`・`:202`) |
| 計画 | info | D7b(`:62`)は `-r2`・`-r3` だけを書き、F8 の共有の上限と DECISION での続きは Progress(`:147`)にある |

### Findings(cycle 2)

#### V2-1(LOW): 仕様の注が `since:` の除外を F7 の前の言い方で書く

`docs/specs/2026-10-07-org-multi-org-director.md:133` は、上限の QUESTION の条件を「その回の report の `HEAD:` から `docs/reports/` の外が変わっていれば」と書く。F7 のあとの雛形は、`docs/reports/` 直下の `.md` のほかが変われば changed にし、`docs/reports/templates/` のような下のディレクトリの変更も数える(`internal/org/prompts/leader.md:810`、`:862-867`)。仕様のとおりに読むと、テンプレートの変更は相手に見せなくてよいことになる。handoff のとおり、sync-docs の cycle 2 で「`docs/reports/` 直下の `.md`(report)のほかが変わっていれば」に直すことを勧める。

#### V2-2(LOW): ブランチが origin/main から遅れ、main と合わせると tech-debt の台帳がぶつかる

origin/main は 31812a3b(PR #297)で、merge-base 4fef6b8c から 126 コミット進んだ。`git merge-tree --write-tree --name-only main HEAD` は `docs/tech-debt/README.md` で `CONFLICT (content)` を出す。両側が 259・362・368 行の行を書き換え、415 行のあとに行を足したため。両側が触ったほかの 12 ファイル(director と org の skill の写し 8 つ、README、仕様、`internal/org/spawn.go`、`internal/org/skill_doc_followup_test.go`)は自動でまとまる。この回の静的検査とこれまでの /test は merge-base の上の結果で、まとめたあとの tree では確かめていない。`/pr` の前に origin/main を取り込み、台帳は両側の書き換えと足した行をどちらも残し、`./scripts/run-static-verify.sh`(check-sync と secret scan を含む)と、自動でまとまった skill の文を固定するテスト(`internal/org/` の Skill と Director)を打ち直すことを勧める。

#### Info(cycle 2)

- `XREVIEW_BASE: origin/@` は `base: ok` になる。`git check-ref-format --branch @` が rc=0 で通すため。値は option にも shell が読む文字にもならず、git が解けなければ `diff: failed` か codex の incomplete に倒れる。leader は手順 9 の決め方(origin の HEAD の指す先、main、master)で値を作るので、ふつうは出ない
- git がブランチ名に許すのに手順 1 が断る文字が残る(`'`、`"`、`$`、`` ` ``)。`reviewer.md:193-194` は、コマンドが値を二重引用符で囲んで渡すための断りと書く。計画の Progress(`:147`)は、F7 の件名「git が許すブランチ名を受け付ける」が言い過ぎだと記録している
- cycle 1 の Info の 1 つ目(`switch: off` と `cap: error` が同時に出たときに読む順)は、この回も文に書いていない

### Coverage gaps(cycle 2)

- F7 と F8 を打つテストの結果は /test が確かめる(test report は run 2 の 17fc2b52 まで)
- base の確かめは本物の git と stub で打ち、本物の CLI では打っていない。相手を呼ぶコマンドは 62d62be0 の live probe のときと同じ
- origin/main と合わせた tree での検査はしていない(V2-2)
- driver のコマンドは、本物の `ralph org status --json` を手で書いた manifest に打った。本物の spawn と stop が書いた manifest ではない(`stopped` の event は stop と同じく role と driver を写して書いた)
- 座席の中での実行(`docs/tech-debt/README.md:418`)と、leader と director のあいだで上限の QUESTION と送り直しの上限の BLOCKED が往復することは確かめていない

### Insight event(cycle 2)

`./scripts/insights-append.sh --slug org-cross-review --flow standard --phase verify --verdict pass --cycle auto --source skill --low 2` は rc=0 で、UTC の日付が 2026-11-02 なので新しい `docs/insights/events/2026-11-02-org-cross-review.jsonl` に書いた(`"cycle":2`、`"low":2`)。スクリプトが書いたままコミットする。

## Verdict

pass

- Verified: AC1〜AC6、AC5b、AC8(仕様の 1 文を除く)、AC9 と D1〜D10(D6b・D7b を含む)は雛形・skill・文書の該当行にある。F6 の driver のコマンド、F7 の `since:` と base の確かめは、雛形から抜き出して 2 つの shell で打った結果が文のとおりだった。verify cycle 1 の V1・V2、cross-review cycle 1 の ACTION_REQUIRED 2 件、self-review cycle 2 の C2-1〜C2-4 は解決した。静的の検査はすべて通り、計画の digest は承認時の `ce18b629320e` のまま
- Partially verified: AC7(静的の部分だけ。テストは /test)
- Not verified: Go のテストの結果、origin/main と合わせた tree、座席の中での相手の CLI の呼び出し、leader と director の実機の往復
- Findings: LOW 2 件。V2-1(仕様の 1 文、sync-docs で直す)と V2-2(origin/main との衝突、`/pr` の前に取り込む)。どちらもブランチの中身の準拠は変えない

## Cycle 1(HEAD d8e02c1d)

- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1。`cycle-count.json` は `"cycle": 1`)
- Scope: `git diff origin/main...HEAD`(origin/main 4fef6b8c、HEAD d8e02c1d、38 ファイル、+4507/−195)。spec の準拠、静的解析、文書のずれを見た。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。`sh tests/test-xreview-helpers.sh` は振る舞いのテストだが、lead の handoff が静的の検査の一覧に名指したので打った
- Evidence: `docs/evidence/verify-2026-11-01-org-cross-review.md`(静的検査の要約、render の確かめ、状態のコマンドと reviewer のコマンドを stub で打った結果)。run-static-verify 自身のログは `docs/evidence/verify-2026-11-01-001525.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)

### Spec compliance(cycle 1)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 off(false、古い ralph、ralph なし)なら TASK を送らず、PR の本文の `Cross-review:` に `skipped (<理由>)` | met | 状態のコマンドの `switch:` の行 `internal/org/prompts/leader.md:798-799`、扱い `:815-828`(off は TASK を送らず `Cross-review: skipped (<理由>)` を控えて sync-docs へ)、PR の本文 `:603-610`、最後の値 `:1019-1025`。stub の probe で、rc 0 の `false` は `switch: off (off: [org.review].cross_review)`、127 は `switch: off (ralph not found; treated as off)`、rc 1 で `unknown command "config"` は `switch: off (ralph without config get; treated as off)` を出し、ほかの rc 1・rc 2・rc 0 の `maybe` は `switch: error` になった(evidence の A〜F) |
| AC2 on なら手順 9 の rebase のあと・sync-docs の前に `xreview-` の TASK。driver、base、回数、report のパス、reviewer の model と effort を持つ | met | 手順 9 の段落 `leader.md:351-360`(rebase のあと)、sync-docs の頭 `:362`。TASK_ID `:937-944`、本文の行 `:947-959`(`IMPLEMENTER_DRIVER:`、`XREVIEW_BASE:`、`CYCLE:`、`REPORT:`、`REVIEWER_CODEX:`、`REVIEWER_CLAUDE:`)、例 `:966-977`。手順の番号は振り直していない(見出しの比較で増えたのは `### cross-review` だけ) |
| AC3 reviewer は `pick_reviewer <implementer の driver>` で違う系列を選び、読むだけの形で呼ぶ(スタブのテストで両方向) | met | 手順 2 の 1 行 `internal/org/prompts/reviewer.md:189`(driver を一重引用符で確かめ、`pick_reviewer "<IMPLEMENTER_DRIVER>"`)、理由 `:177-186`。codex `:223-251`(`-c sandbox_mode=read-only exec review --ignore-rules`、`--disable plugins --disable apps`、MCP を切る)、claude `:253-280`(diff のファイル、`--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk`、`RALPH_WIP_CHECKPOINT=off`)。スタブのテスト `TestRenderRolePrompt_Reviewer_XreviewCallsTheOtherModel`(`internal/org/prompts_xreview_reviewer_test.go:837`)は pane の `RALPH_PRIMARY_CLI` を逆の値にして両方向を確かめる。手で打った probe でも、driver `codex` と pane の `RALPH_PRIMARY_CLI=claude` で `reviewer=claude`、driver `claude` で `reviewer=codex` になった(evidence の R1〜R2b)。テストが通るかは /test |
| AC4 report を `review-xreview-<date>-<org>-c<n>.md` に書き、ACTION_REQUIRED があれば BLOCKED、なければ RESULT。呼べなければ skipped、終わらなければ incomplete | met | report の形 `reviewer.md:349-391`、返事 `:393-400`、skipped `:195-206`・`:289-292`、incomplete `:172-176`・`:293-320`。probe で、CLI がなければ `available rc=1`、出力が空なら `out: empty`、見張りに止められると `codex rc=143` になった(evidence の R5〜R7、X2、X4、Y2) |
| AC5 直すものを implementer に頼みレビューから回し直す。complete の回が上限に届き ACTION_REQUIRED が残れば 3 択の QUESTION。人が上げた上限を `.harness/state/xreview-<org>-cap` に控える | met | 回し直し `leader.md:997-1005`、上限の QUESTION `:875-879`・`:889-915`、a の cap のファイル `:917-930`、b `:931-933`、c `:934-935`。cap の読み方 `:800-802`・`:829-835`(設定とファイルの 1 行目の大きいほう)。probe で、ファイル `3` は `cap: 3`、`03`・`1`・`5 extra` は設定の 2 のまま(evidence の I) |
| AC5b TASK の前に上限を確かめ、上限の回の pass のあとに変われば送らずに聞く | met | `leader.md:871-887`(`done:` が `cap:` 以上なら送らない。`since: changed` で 2 つ目の QUESTION)、`since:` の決め方 `:858-869`。probe で、complete の c3(AR=0)で done 2 = cap 2 のときは `since: same`、そのあと gate の直しのコミットを足すと `since: changed` になった(evidence の AC5b-1・AC5b-2)。HEAD の object が消えたときも `since: changed`(M) |
| AC9 本物の claude と codex で両方向に走らせ、complete の report の形の出力を `docs/evidence/` に残す。座席の中は known gap | met(586f7934 の時点) | `docs/evidence/xreview-live-probe-2026-11-01.md`。A(claude の implementer → codex)は `codex rc=0`・`out: not empty`・`git: same`、所見 1 件 `[P1]`。B(codex の implementer → claude)は `claude rc=0`・`out: not empty`・`git: same`、所見 4 件。座席の中で走らせていないことは `:9` と結論 `:143` に書いてある。利用者名や secret の形の文字列は grep で 0 件。記録は 586f7934 の版のコマンドで、そのあと reviewer.md の節は aa0a0fd5 などで変わった(Coverage gaps) |
| AC6 cross-review の TASK の間の reviewer の停滞の ALERT は gate と同じ扱い(雛形と director・org の skill) | met | `leader.md:504-512`(手順 12 の段落に cross-review の待ちを足した)、`:979-985`。`.claude/skills/director/SKILL.md:697-701`、`.claude/skills/org/SKILL.md:1236-1238`。写し 3 つずつは `cmp` で同じ |
| AC7 雛形のテスト、`check-skill-sync.sh`・`check-sync.sh`・`check-template-purity.sh`、文の固定のテスト | partly | 静的の 3 つは通った(下の表)。新しいテスト 3 本(`prompts_xreview_leader_test.go` 1,144 行、`prompts_xreview_reviewer_test.go` 1,127 行、`skill_xreview_test.go` 485 行)がある。既存のテスト 3 本の固定の文は新しい文に合わせて書き換えてあり(`prompts_test.go`、`prompts_leader_syncdocs_test.go`、`skill_doc_followup_test.go`)、意図は弱めていない。テストが通るかは /test が確かめる |
| AC8 設定と skill の文書が、leader が設定を読み reviewer が implementer の driver で相手を選ぶ形を書く | met | `internal/config/config.go:674-701`、`templates/base/ralph.toml:330-347`、`/org` skill `.claude/skills/org/SKILL.md:33-36`・`:50-58`・`:1074-1104`、仕様 `docs/specs/2026-10-07-org-multi-org-director.md:133`、README `README.md:278`・`:284`、`docs/reports/README.md:23`(写しと同じ) |

#### Design decisions(cycle 1)

| 決定 | 状態 | Evidence |
| --- | --- | --- |
| D1 設定は手順 9 で leader が config get で読む。終了コードは計画助言と同じ | met | `leader.md:787-788`、`:798-802`、`:815-828` |
| D2 手順 9 の rebase と sync-docs の間に 1 段落、詳しくは `### 計画助言` のあとの小節。番号は振り直さない | met | `leader.md:351-360`、`### cross-review` は `:780`(`### 計画助言` `:654` のあと)。origin/main との見出しの比較で増えたのは `### cross-review` だけ |
| D3 数えるのは complete の report だけ。skipped と incomplete も report を書き 0 の `After triage:` を残す | met | `leader.md:844-857`、`reviewer.md:353-355`。probe K で skipped の c2 は `done:` に入らず `k:` に入った |
| D4 report の名前を `review-` で始める | met | `scripts/check-sync.sh:80` の除外 `docs/reports/review-`、`internal/insights/backfill.go:69-80` は `review-` を読まない |
| D5 TASK の形、ミッション 1 の除外、返事の `XREVIEW:`、`git_changed`、欠けた行の QUESTION、`REVIEWER_*` がなければ incomplete | met | ミッション 1 `reviewer.md:17-21`、手順 1 `:165-176`、`git_changed` `:283-288` と leader の扱い `leader.md:1006-1012`、QUESTION の扱い `:1013-1017` |
| D6 相手の呼び方(codex は read-only の `exec review`、claude は diff のファイルを読むだけ、20 分の見張り、git の前後の比較、rc 0 かつ出力が空でない) | met | `reviewer.md:207-280`、`:298-301`。probe X1・Y1 |
| D6b `--disable apps`(leader の計画助言にも)、incomplete の理由の形、driver の確かめ、helper がないときの skipped、`sh -c 'command -v'`、claude の指示文、P0〜P3 の読み替え | met | `reviewer.md:229-247`、`:302-320`、`:189`、`:195-199`、`:263-268`、`:323-326`。leader の計画助言 `leader.md:720-738` |
| D7 上限での QUESTION、cap のファイル、送る前の確かめ、`HEAD:` の行 | met | `leader.md:871-935`、`reviewer.md:349-352`(`HEAD:` は相手に見せた HEAD) |
| D7b `k:` は番号の最大 + 1、`-r2`・`-r3`、`docs/reports/` の外で `since:` を決める、上限の回の ACTION_REQUIRED はそのまま QUESTION、`XREVIEW: cap_reached` と `FORK 1:`、c は何もしない | met | `leader.md:853-857`、`:937-944`、`:858-869`、`:999-1002`、`:889-935`。probe J で前の org の c9 があると `k: 10` |
| D8 report は leader がコミットしてから sync-docs | met | `leader.md:352-360`、`:987-990` |
| D9 停滞の ALERT を gate と同じく | met | AC6 と同じ |
| D10 PR の本文の `Cross-review:` の行と、off のときの `skipped (off: [org.review].cross_review)` | met | `leader.md:603-613`、`:799`、`:1019-1025`。RESULT の `CROSS_REVIEW:` `:627-628`、`/director` の報告 `.claude/skills/director/SKILL.md:663-668` |

### Static analysis(cycle 1)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc=0) | `All verifiers passed.`。`scripts/xreview-helpers.sh` が分類されないので full fallback になり golang が走った。shellcheck、`sh -n`、guard の awk、`jq -e`、Codex hook の検査 3 つ、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan の参照、gofmt、golangci-lint(`0 issues.`)、secret scan(`4fef6b8c..d8e02c1d`、clean) |
| `./scripts/check-template-purity.sh` | pass(rc=0) | `PASS: no meta-repo-specific references found in templates.` |
| `./scripts/check-skill-sync.sh` | pass(rc=0) | 14 skill が一致 |
| `./scripts/check-sync.sh` | pass(rc=0) | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0 |
| `sh tests/test-xreview-helpers.sh` | pass(rc=0) | PASS 53、FAIL 0。1f「explicit arg wins over env」がある |
| `git diff --check origin/main...HEAD` | pass(rc=0、出力なし) | |
| `go build ./...` / `go vet`(`internal/org/`、`internal/org/driver/`、`internal/cli/`、`internal/config/`)/ `gofmt -l internal cmd` | pass | どれも出力なし |
| `./scripts/plan-visual.sh digest <plan>` | `7789ae616bd0` | 計画の `- Approved:` の digest と同じ |
| 写しの `cmp` | 同じ | director と org の skill の写し 3 つずつ、`scripts/xreview-helpers.sh`、`docs/reports/README.md`、2 つの recipe |

### Documentation drift(cycle 1)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `internal/config/config.go:674-701` | yes | leader だけが使うときに読み、reviewer は TASK の driver で選ぶ。complete の回だけを数える |
| `templates/base/ralph.toml:330-347` | yes | 同じ中身。「the reviewer seat starts reading ... in a later release」の古い文は消えた |
| 仕様の注 `docs/specs/2026-10-07-org-multi-org-director.md:133` | yes | 雛形と合う。「数はコミット済みの report から読む」はこのブランチが足した report に限ることまでは書かないが、詳細は計画を指しているので info |
| README `:278`、`:284` | yes | `pick_reviewer` は driver がないときだけ変数を読む(`scripts/xreview-helpers.sh:109-116` と合う)。`xreview-<org_id>-c<k>`、`XREVIEW: cap_reached`、`Cross-review:` の 3 つの値 |
| AGENTS.md | yes | 機能の org の手順を並べない。触っていない |
| `docs/reports/README.md:23` と写し | yes | `review-xreview-YYYY-MM-DD-<org_id>-c<k>.md` |
| `docs/recipes/codex-seat-permissions.md:147-164` と写し | yes | network の注。座席の中は未確認と書く |
| `docs/recipes/isolated-herdr-server.md:250-255` と写し、`internal/org/driver/paneenv.go:41-47`・`:72-76`、`internal/org/spawn.go:996-1006` | yes | 警告文は leader の計画助言のことだけになった。「seat's cross-review may pick」「treats the seat's driver as claude」は過去の report のほかに残っていない(`git grep`) |
| `scripts/xreview-helpers.sh:87-105` と写し | yes | 4 つ目の呼び手(reviewer 座席)が引数を渡すこと |
| `/director` skill(4 つ)の上限の件 `:845-905` | yes | `FORK 1: a 上限を上げる(新しい上限 3)` の形、新しい上限の決め方(report の `- Cycle:` の `<n>` と leader の `done:` は同じ数になる)、report は機能の worktree `.claude/worktrees/org-<org_id>`(`internal/org/feature.go:163`)で読む |
| `/org` skill(4 つ) | yes | 前提 2 か所、終わり方の 1・4・6・7、締めの順 `:2046` |
| `docs/tech-debt/README.md:416` | yes | 行数は HEAD で数え直して同じ(leader.md 1,256、reviewer.md 526、テスト 1,144 と 1,127。origin/main は 975 と 224) |
| `#269` への前向きの参照 | なし | 仕様の `:131`(#265 の日付の付いた注)だけで、`:133` の注が受けている |
| 計画の Affected areas | info | `docs/recipes/codex-seat-permissions.md` と写し(S4 と Risks に名前がある)、`docs/tech-debt/README.md`(Risks の「tech-debt で扱う」)、`internal/org/skill_doc_followup_test.go`(固定の 1 行。「など」に入る)は Affected areas に名前がない。中身のずれではない |
| 計画の AC の checkbox | 未記入 | 指示どおり印は付けていない |

### Observational checks(cycle 1)

- render: `go run -overlay` で `org.RenderRolePrompt` を `OrgID: "org-a"` で呼び、leader と reviewer を render した。`{{` は 0 件。下の probe で使った 4 つのブロックは render した文にそのまま入っている。worktree の `git status --porcelain` は空のまま
- 状態のコマンド(`leader.md:797-810`)と reviewer のコマンド(`reviewer.md:189`、`:242-250`、`:274-279`)を、scratch の git repo で stub の `ralph`・`codex`・`claude` を使い、`/bin/sh` と `zsh -f` で打った。2 つの shell で出力は同じ(SHA とパスのほか)。結果は AC の表と evidence の表のとおり
- bash の `/bin/sh` では、見張りを止めたときのジョブの通知(`Terminated: 15`)が出力の行のあいだに入る。判定の行は別の行なので読み分けは変わらない(S1c で直したテストの話と同じもの)

### Findings(cycle 1)

#### V1(LOW): reviewer の雛形の返事の例の TASK_ID が leader の送る形と違う

`internal/org/prompts/reviewer.md:406` と `:418` の RESULT と BLOCKED の例は `TASK_ID: xreview-1` を使う。leader は `xreview-{{ORG_ID}}-c<k>` で送り(`leader.md:937`、例 `:968`)、reviewer は受けた TASK_ID で返す(`reviewer.md:393`)ので、動きは正しい。例の `REPORT:` は `-c1.md` なので、読み手には `xreview-{{ORG_ID}}-c1` の返事に見えるほうが自然。`prompts_xreview_reviewer_test.go:509` が `xreview-1` を固定しているので、直すならテストも合わせる。

#### V2(LOW): 状態のコマンドが report の `- HEAD:` の値を形を確かめずに git の引数にする

`leader.md:810` は report の `- HEAD:` の値 `h` を `git diff --quiet "$h" HEAD -- . ':(exclude)docs/reports/'` に渡す。値は reviewer が書く report から来る(雛形はほかの場所で座席の文をデータとして扱う)。`-` で始まる値は git の option になる。scratch で `h` を `--output=<file>` にすると rc=0 で 0 バイトのファイルができ、`since: same` が出た(evidence の「report の `- HEAD:` を option として読む場合」)。上限での「見せていない変更」の判定(AC5b)が、その場合だけ安全でない側に倒れる。reviewer は `-git-before.txt` の 1 行目(`git rev-parse HEAD`)を書くので、ふつうの運用では起きない。直すなら、`case "$h" in ''|*[!0-9a-f]*) echo 'since: changed' ;; ...` のように 16 進でなければ `changed` にする。reviewer の `git diff "<XREVIEW_BASE>...HEAD"`(`reviewer.md:276`)も、`XREVIEW_BASE:` の形(`origin/` で始まるか)は確かめていない(値は leader が作る)。

#### Info(cycle 1)

- 状態のコマンドで、`switch: off` と同時に `cap: error` か `base: error` が出たときにどちらを先に読むかは、文に書いていない(`leader.md:815-843`)。probe では rc 0 の `false` と cap の rc 2 の組で両方の行が出る。2 つの設定は同じ binary(#265)から読むので、ふつうは片方だけが落ちることはない
- report の `HEAD:` が存在しない object のときと、rebase で upstream が入ったときは `since: changed` になる。計画の D7b のとおり

### Coverage gaps(cycle 1)

- AC7 のテストの結果と、計画の Test plan の変異(相手を driver と同じにする、off で TASK を送る、cap のファイルを読まない)が落ちることは /test が確かめる
- AC9 の live の記録は 586f7934 の版のコマンドで、そのあと reviewer.md の節は 63 行増え 24 行減った(aa0a0fd5 の `--disable apps`、driver の `case`、`sh -c 'command -v claude'`、`log:` の行など)。今の版のコマンドを本物の CLI で通しで走らせた記録はない。stub の probe とテストは今の版の形を確かめる。下の「ずれ」の偶然の呼び出しで、codex 0.163.0 が今の版の引数(`--disable plugins --disable apps` の付いた `mcp list` と `exec review`)を受け付けて API まで届くことは見えたが、モデルを `gpt-x` にしていたので review は走っていない
- 座席の中での実行(codex の reviewer 座席の `--sandbox workspace-write` での network、pane の環境)は、計画のとおり known gap
- leader と director が実機で上限の QUESTION を往復することは確かめていない

### Deviations(cycle 1)

- 最初の zsh の probe を `zsh -f` でなく `/bin/zsh` で打ったため、zsh の起動ファイルが PATH の先頭を戻し、stub ではなく本物の `ralph`、`codex`(4 回)、`claude`(2 回)が scratch の repo で呼ばれた。codex の 3 回はモデル `gpt-x` を API に断られ、見張りを 2 秒にした 1 回は止められて rc=0(雛形の「見張りに止められても 0 を返すことがある」と同じ)。claude には stub 用の 1 語 `PROMPT` が渡り、読む道具だけで scratch の repo を見て「指示がない」と返した。送ったのは scratch の repo の数行の差分だけで、worktree の外にも中にも書いていない。上の結果は `zsh -f` で打ち直したもの。詳細は evidence の最後の節
- `sh tests/test-xreview-helpers.sh` は handoff の指示で打った

### Insight event(cycle 1)

`./scripts/insights-append.sh --slug org-cross-review --flow standard --phase verify --verdict pass --cycle auto --source skill --low 2` を、この report を書く前に打った。UTC の日付が 2026-11-01 に変わっていたので、self-review の event のある `docs/insights/events/2026-10-31-org-cross-review.jsonl` ではなく、新しい `docs/insights/events/2026-11-01-org-cross-review.jsonl` に書かれた(`"cycle":1`、`"low":2`)。スクリプトが作ったままコミットする。

### Verdict(cycle 1)

pass

- Verified: AC1・AC2・AC3・AC4・AC5・AC5b・AC6・AC8 と D1〜D10(D6b・D7b を含む)は、雛形と skill と文書の該当行にあり、状態のコマンドと reviewer のコマンドを stub で打った結果も文のとおりだった。静的の検査はすべて通り、計画の digest は承認時の `7789ae616bd0` のまま
- Partially verified: AC7(静的の部分だけ。テストは /test)、AC9(586f7934 の版で記録済み。今の版の live の通しはない)
- Not verified: Go のテストの結果、座席の中での相手の CLI の呼び出し、leader と director の実機の往復
- Findings: LOW 2 件(V1 例の TASK_ID、V2 `HEAD:` の値の形)。どちらも merge を止めない
