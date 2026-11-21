# org-xreview-own-model

- Status: Approved
- Approved: 2026-11-22 sha256:aa9ca6400c54
- Owner: Claude Code
- Date: 2026-11-22
- Related request: issue #360(org の reviewer 座席が自分のモデルで cross-review し、相手の CLI をヘッドレスで呼ばない)
- Related issue: 360
- Type: fix
- Branch: fix/org-xreview-own-model

## Objective

機能の org の cross-review で、reviewer 座席が相手の CLI(`codex exec`、`claude -p`)を座席の中から入れ子で呼ぶのをやめ、自分のモデルでブランチの差分を読み、triage して report を書くようにする。codex の座席の sandbox の中で入れ子の codex が初期化できずに incomplete になること、相手の呼び出しで 20 分待つこと、通信が切れて incomplete になることがなくなる。

ユーザーの決定(2026-11-21、原文): 「というか、org runtimeでは、cross-review関係なく、reviewer座席がレビューを担う役割です。同じ系統でも異なる系統でも関係ありません。reviewer座席に指定されたモデルがレビューを担ってください。」

## Scope

- reviewer の雛形(`internal/org/prompts/reviewer.md`)の「cross-review の TASK」を書き直す
  - TASK の本文で読むのは `XREVIEW_BASE:`、`CYCLE:`、`REPORT:` の 3 行。今の確かめ(欠けた行、形の違う値は何も書かずに QUESTION)はそのまま。`IMPLEMENTER_DRIVER:`、`REVIEWER_CODEX:`、`REVIEWER_CLAUDE:` の行があっても読まない(前の版の leader が送る TASK でも動く)
  - 手順: base を確かめ、読む前の git の状態(HEAD、index、未追跡のファイルを含む作業ツリー)を今と同じく `.harness/state/xreview-{{ORG_ID}}-c<k>-git-before.txt` に書き、`git diff --end-of-options <XREVIEW_BASE>...HEAD` で差分を読み(読めなければ `incomplete (git diff failed: <要点>)`)、自分で差分と変わったファイルを読んで所見を出す。report を書く前に git の状態を `-git-after.txt` に書いて `cmp` で比べ、違えば今と同じく report を書かずに `XREVIEW: git_changed` の BLOCKED を返す(読んでいるあいだに HEAD や作業ツリーが動くと、見ていない変更を pass にしてしまうため。codex の計画助言の指摘)。report の `HEAD:` は `-git-before.txt` の 1 行目。所見を出すとき、読むのは差分と変わったファイル(正しさ、security、エラーの扱い、論理、テストの抜け、変更に特有の見落とし。重大度は HIGH/MEDIUM/LOW)。レビューの TASK のレポートとは別に、ブランチ全体をもう一度疑ってかかる目で読む
  - `codex exec`、`claude -p`、そのほかの CLI で別のモデルを呼ばないと明記する
  - triage の表と分け方、report の形(見出し、`Reviewer status:`、`HEAD:`、`Total reviewer findings:`、`After triage:` の行)、返事(`XREVIEW: pass`、`action_required`、`incomplete`)は今のまま。`Reviewer:` の行は「reviewer 座席 <seat_id>(<driver>、<model>)」にする。値の出所がなくなる `- Driver: <IMPLEMENTER_DRIVER の値>  Reviewer: <相手>` の行は `- Reviewer: ...` だけの行にする(leader が読むのは `Reviewer:` と `Reviewer status:` だけ)。`## Triage context` の `- Reviewer output: .harness/state/xreview-...-out.md`(相手の出力のファイル)は `- Diff: <XREVIEW_BASE>...<HEAD の SHA>` の行に替える(所見は report の表に直接書く)。相手を呼ぶための `skipped (<相手> not available)`、`helpers: missing`、`mcp: not off`、20 分の見張り、MCP を止める手順を消す。git の状態の比べと `XREVIEW: git_changed` は残す(比べる区間が、相手の呼び出しの前後から、自分が読む前と report を書く前に変わる)
  - 座席の雛形に driver と model を渡す(`RolePromptVars` に `Driver` と `Model` を足し、`{{DRIVER}}` と `{{MODEL}}` を置き換える。model が空なら CLI の既定と書く)。reviewer の雛形の先頭の行に出す
- leader の雛形(`internal/org/prompts/leader.md` の「cross-review」の小節)
  - TASK に書く行を `XREVIEW_BASE:`、`CYCLE:`、`REPORT:` にする。台帳から implementer の driver を読むコマンドと `driver: error` の扱い、`REVIEWER_CODEX:` と `REVIEWER_CLAUDE:` の行を消す(タスクの `- reviewer の codex:` と `- reviewer の claude:` の行は計画助言が使うので残す)
  - 「implementer と違う系列のモデル」「相手のモデル」の言い方を「reviewer 座席」に直す。`XREVIEW: git_changed` の扱い(BLOCKED を上げ、戻したら送り直す)と送り直しの上限は残し、「相手を呼ぶ前と後」の言い方を「reviewer が読む前と report を書く前」に直す。20 分の待ちの説明を消す。返事が来るまで worktree を変えない決まりは残す
  - `Cross-review: ran (<report の Reviewer:>)` の決め方は今のまま(値が座席の名前になる)
  - 上限、`done:` と `k:` の数え方、上限の QUESTION は今のまま
- 文書: `.claude/rules/ralph/model-routing.md`(と雛形の写し)の reviewer 座席の cross-review の説明とテストの一覧、org と director の skill(4 写しずつ)、README、AGENTS.md、`docs/architecture/repo-map.md`(あれば)、`docs/specs/2026-10-07-org-multi-org-director.md` に 2026-11-22 の項目、`docs/recipes/codex-seat-permissions.md`(と雛形の写し)の「The cross-review of a codex reviewer seat」の節を「入れ子の CLI を呼ばない」に書き直す、`docs/recipes/isolated-herdr-server.md`(と写し)の reviewer 座席の `pick_reviewer` の言及、`internal/config/config.go`(`[org.review]` と `max_cross_review_cycles` のコメントの「the other model」など)と `internal/org/driver/paneenv.go` のコメント、`templates/base/ralph.toml` のコメント
- #362 の consult の指摘: `docs/recipes/codex-seat-permissions.md` の「The network of a codex seat」の節に、座席の中のエージェントが外のどのホストにも出られる(宛先の許可リストはない)ことと、閉じ方の 1 文を足す
- tech-debt: leader の計画助言は今も座席の中から相手の CLI を入れ子で呼ぶ(codex の leader では同じ初期化の失敗が起きうる)ことを行にする

## Non-goals

- 標準フローの `/cross-review`(メインのセッションが相手の CLI を呼ぶ)と `scripts/xreview-helpers.sh`
- leader の計画助言の入れ子の CLI 呼び出し(tech-debt に残す)
- reviewer 座席の driver を implementer と違う系列にそろえる仕組み(ユーザーの決定で、系列によって扱いを変えない。違う系列の目が欲しい人は `[org.role_models].reviewer` を変える)
- `[org.review].cross_review` と `max_cross_review_cycles` の意味と既定
- codex の座席の sandbox の設定

## Assumptions

- 機能の org の reviewer 座席は、レビューの TASK と同じ worktree で `git diff` と読む道具を使える(今もレビューの TASK でそうしている)
- 動いている org の座席は立てたときの雛形のまま。新しい雛形は、この版の binary で立てた(または reseat した)座席から効く。前の版の leader が新しい reviewer に `IMPLEMENTER_DRIVER:` などを送っても、新しい reviewer は読まないので動く。新しい leader が前の版の reviewer に 3 行だけの TASK を送ると、前の reviewer は `IMPLEMENTER_DRIVER:` がないので QUESTION を返す(混ぜた版は reseat で揃える、と release ノートに書く)

## Affected areas

- `internal/org/prompts/reviewer.md`、`internal/org/prompts/leader.md`
- `internal/org/prompts.go`(`RolePromptVars` の `Driver` と `Model`)、`internal/org/spawn.go`(値を渡す所)
- テスト: `internal/org/prompts_xreview_reviewer_test.go`、`prompts_xreview_leader_test.go`、`skill_xreview_test.go`、`prompts_test.go`、`prompts_leader_plan_test.go`(reviewer の行の扱い)、そのほか雛形の文を固定するテスト
- 文書: 上の Scope の一覧

## Visual review

- ページ: `.harness/state/plan-visual/org-xreview-own-model.html`。かんたんな図 1 枚(`overview`)と、詳しい図 2 枚(`detail-overview`、`detail-flow`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 3 枚の図を `shot` で撮って読んだ。直したのは 1 点で、ファイルの地図の標準フローの箱の文字が右端に届きそうだったので 2 行に分けた。撮り直して、どれもきれいに出ることを確かめた

## Design decisions

Critical forks: None(reviewer 座席が自分のモデルでレビューすることと、系列で扱いを変えないことはユーザーが決めた)

- cross-review の TASK と `[org.review]` の設定は残し、reviewer 座席が自分で読む 2 度目のレビューにする。report の形と leader の扱い(上限、`Cross-review:` の行)を変えずに済み、issue の「report の形と leader への返し方は変えない」に合う
- report の `Reviewer:` は座席の名前と driver と model にする。PR の本文の `Cross-review: ran (...)` から、誰が見たかが読める
- 雛形に driver と model を渡す。座席が自分の CLI を推して書くより確か

## Acceptance criteria

- [ ] AC1: reviewer の雛形の「cross-review の TASK」が、自分で差分を読んで所見を出し triage することを言い、`codex exec`、`claude -p`、`pick_reviewer`、`mcp list`、`sleep 1200` を持たない。読む前と report を書く前の git の状態を比べ、違えば report を書かずに `XREVIEW: git_changed` の BLOCKED を返すことを言う(どちらもテストで固定)
- [ ] AC2: reviewer の TASK の確かめは `XREVIEW_BASE:`、`CYCLE:`、`REPORT:` の 3 行で、`IMPLEMENTER_DRIVER:` などの行があっても読まないことを言う。`git diff` が失敗したら `incomplete (git diff failed: <要点>)` で report を書いて返す
- [ ] AC3: report の形(見出し、`Reviewer status:`、`HEAD:`、`Total reviewer findings:`、`After triage:` の行、3 つの表)と、`XREVIEW: pass`、`action_required`、`incomplete`、`git_changed` の返し方が今のまま。`- Driver: ...  Reviewer: ...` の行は `- Reviewer: reviewer 座席 <seat_id>(<driver>、<model>)` の行に、`## Triage context` の `Reviewer output:` の行は `Diff:` の行に替わる(テストで固定)
- [ ] AC4: reviewer の雛形を描くと、先頭に座席の driver と model が出る(model が空なら CLI の既定と書く)。`RenderRolePrompt` が描く 3 役(leader、implementer、reviewer)の雛形に `{{DRIVER}}` と `{{MODEL}}` が残らない(director の雛形は別の置き換えで `{{DRIVER}}` を使うので、このテストの外)
- [ ] AC5: leader の雛形の「cross-review」の小節が、TASK に `XREVIEW_BASE:`、`CYCLE:`、`REPORT:` の 3 行を書くこと、implementer の driver を読まないこと、`XREVIEW: git_changed` を今のとおり扱う(先に進まず BLOCKED を上げ、戻したら送り直しの上限の中で送り直す)ことを言う。上限の数え方と上限の QUESTION、`Cross-review:` の行の決め方は今のまま(テストで固定)
- [ ] AC6: 標準フローの `/cross-review` の skill と `scripts/xreview-helpers.sh` は変わらない(差分に入らない)
- [ ] AC7: model-routing.md、org と director の skill、recipe、README、仕様が新しい動きを書き、`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る
- [ ] AC8: recipe の「The network of a codex seat」の節に、座席が外のどのホストにも出られることと閉じ方がある

## Implementation outline

1. S1 雛形に driver と model を渡す(AC4) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'RenderRolePrompt|Spawn'`
2. S2 reviewer と leader の雛形と、そのテスト(AC1〜AC3、AC5) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'RenderRolePrompt|Xreview|Prompt'`
3. S3 文書と skill(AC6〜AC8) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill|Xreview|DocsMatch' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Agents|Readme|Docs|Skill' && ./tests/test-agent-models.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC8。`git diff main...HEAD -- .claude/skills/cross-review scripts/xreview-helpers.sh` が空
- Documentation drift to check: reviewer 座席が相手の CLI を呼ぶと書いた所が残っていないか(`git grep -n "IMPLEMENTER_DRIVER\|REVIEWER_CODEX\|REVIEWER_CLAUDE"` が雛形と skill とルールに残らない。計画助言の `- reviewer の codex:` の行は残る)
- Evidence to capture: 描いた reviewer と leader の雛形の該当の節(テストの出力)

## Test plan

- Unit tests: reviewer の雛形の cross-review の節の文と、消した文がないこと。leader の雛形の TASK の例と行の一覧。`RenderRolePrompt` の `{{DRIVER}}` と `{{MODEL}}`
- Integration tests: spawn が reviewer 座席の雛形に driver と model を入れること(fake の herdr)
- Regression tests: 計画助言の reviewer の行(`prompts_leader_plan_test.go`)が今のまま。上限の数え方のコマンドが今のまま
- Edge cases: model が空の座席、前の版の TASK(`IMPLEMENTER_DRIVER:` などを持つ)を読まないこと
- Evidence to capture: `go test` の出力、`./scripts/run-test.sh` の rc

## Risks and mitigations

- 同じ系列の reviewer が implementer と同じ見落としをする: ユーザーの決定で受け入れる。違う系列の目が欲しい人は `[org.role_models].reviewer` を変えると skill と model-routing.md に書く
- 前の版と新しい版の座席が混ざる(reseat の前): 新しい reviewer は前の版の TASK でも動く。新しい leader と前の版の reviewer では QUESTION になるので、release ノートで reseat を勧める
- 雛形の文を固定するテストの書き換えが多い: S2 で雛形とテストを一緒に変え、消した文がないことも固定する
- #366(leader.md)と #363(leader.md、skill)と衝突する: S1 は今の main で始め、S2 の前と S3(skill の 4 写し)の前に、merge 済みの main を取り込む

## Rollout or rollback notes

- release ノート: 機能の org の cross-review は、reviewer 座席が自分のモデルで行い、相手の CLI を呼ばない。report の `Reviewer:` は座席の名前と driver と model。違う系列の目が欲しいときは `[org.role_models].reviewer` を implementer と違う driver にする。動いている org は reseat した座席から新しい雛形になる
- revert すると、座席は相手の CLI を呼ぶ前の雛形に戻る(revert も立て直した座席から効く)

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 1 件、計画に入れた。[MEDIUM] 1: git の状態の前後の比べを消すと、reviewer が読むあいだに HEAD が動いても、見ていない変更に pass が付く → 比べと `XREVIEW: git_changed` を残し、区間を「reviewer が読む前と report を書く前」にした(Scope、AC1、AC5)。git の状態は助言の前後で同じ
- [x] Plan approved — 2026-11-22、系列の決まり(計画は推す案で自動承認)で承認した。人の目での承認はしていない。consult(consult-plan-xreview)は「直してから進める」で、3 点(report の `Driver:` の行と `Reviewer output:` の行の扱い、AC4 を 3 役に限る)と config.go のコメントを計画に入れてから承認した。決定の読み方(cross-review の段を残し、reviewer 座席が自分で見る)はユーザーに聞き直さなくてよいという判定
- [x] Branch created
- [x] Implementation started
  - S1(21df9473、implementer): 確認コマンドが exit 0。`RolePromptVars` に `Driver` と `Model`、`{{DRIVER}}` と `{{MODEL}}`(空の model は `ModelOrCLIDefault`)。spawn の 2 か所(本物と dry run)に `p.Driver` と `p.Model`(AgentStart に渡す値と同じ)。reviewer の雛形の先頭に `- driver: {{DRIVER}} / model: {{MODEL}}`。3 役の雛形に置き場が残らないテスト、spawn が書いた雛形のファイルのテスト。直しを外すと落ちることを確かめた
  - 順番の入れ替え: Risks の「S2 の前に main を取り込む」は、#366 と #363 の merge を待つと遅れるので、S3 の前と PR の前に取り込む形にした。S2 が触る reviewer.md の cross-review の節と leader.md の cross-review の小節は、2 本が触る節と重ならない
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
