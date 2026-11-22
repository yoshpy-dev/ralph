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
  - S2(09367878、implementer。途中で利用の上限で止まり、同じ implementer が続けた): reviewer の「cross-review の TASK」を、3 行の確かめ、読む前の git の状態と差分のファイル、自分で読んで所見、triage、report を書く前の git の比べ(`git_changed`)、report、返事の順に書き直した。相手の CLI、MCP、`pick_reviewer`、20 分の見張り、`IMPLEMENTER_DRIVER:` と `REVIEWER_*:` を消した。report の `Reviewer:` は座席と driver と model、`Diff:` の行。`Reviewer status:` から skipped を外した(leader 側は前の版の reviewer のために扱いを残した)。leader の「cross-review」の小節は TASK を 3 行にし、driver を読むコマンドを消した。小節の外の手順 9・12・14 と立て直しの規則の「相手」「20 分」の文も直した。config.go のコメント。確認コマンドは、範囲外の `internal/org/skill_xreview_test.go` の 3 件(消した文を固定)が落ちるので exit 1 で、それを外すと ok。この 3 件は S3 で直す。変異 8 つで書き直したテストが落ちることを確かめた。範囲外の気づき: reviewer の「座席内 fan-out」の節は、レビューの観点やファイルごとの子のサブエージェントのモデルを決めていない(ユーザーの決定「reviewer 座席に指定されたモデルがレビューを担う」に合わせ、S3 で、判断を担う子は座席と同じモデル、安いモデルは gate の実行だけと書く)
  - 順番の入れ替え: Risks の「S2 の前に main を取り込む」は、#366 と #363 の merge を待つと遅れるので、S3 の前と PR の前に取り込む形にした。S2 が触る reviewer.md の cross-review の節と leader.md の cross-review の小節は、2 本が触る節と重ならない。S3 も 2 本の merge の前に進めたので、取り込みは PR の前(cross-review の前)にまとめる
  - S3(7e4b33b9、implementer): org と director の skill(4 写しずつ)、model-routing.md(2 写し)、recipe 2 つ(2 写しずつ。codex の座席の通信の節に、座席は外のどのホストにも出られて許可リストがないことと閉じ方を足した。#362 の consult の指摘)、仕様の 2026-11-22 の項目、README、tech-debt(leader の計画助言の入れ子の CLI の行を足した)、reviewer の「座席内 fan-out」(判断を担う子は座席と同じモデル、安いモデルはゲートのコマンドの実行だけ)、`paneenv.go` と `templates/base/ralph.toml` のコメント、`skill_xreview_test.go` の書き直し。確認コマンドが exit 0。受け入れ条件 2(同じ系列のとき)は、director の「cross-review の上限」と「分割計画の前の質問」、org の「役割ごとのモデル」と「org の終わり方」の 1 に、どの系列でも reviewer 座席が自分のモデルで見る、違う系列の目が欲しいときは `[org.role_models].reviewer` か分割計画の前文で reviewer の driver を変える、と書いた
  - S4(7d124a16、implementer): S3 の範囲外の気づきのコメントと例。`scripts/xreview-helpers.sh` と写しの `pick_reviewer` の doc comment(呼び出し元から reviewer 座席を外した)、`spawn.go`、`herdr_test.go`、`config.go` の `reviewerWordPattern` のコメント、leader の手順 14 の RESULT の例(`CROSS_REVIEW: ran (reviewer 座席 reviewer(codex、gpt-6.1-sol))`)。AC6 は「標準フローの `/cross-review` と `xreview-helpers.sh` は変わらない(差分に入らない)」だが、`xreview-helpers.sh` の古いコメントを残すと誤りになるので、コメントの 1 か所だけを直した(`#` で始まらない変更行は 0、関数は 1 バイトも変わらない、`tests/test-xreview-helpers.sh` が通る)。AC6 は振る舞いが変わらないことで満たすと読む。残った古い記述は 2 か所(`internal/cli/org_runtime_env_test.go` 182〜185 行のコメント、tech-debt の `pick_reviewer` の行)で、/sync-docs に回す
  - R1(f95a08f9、implementer): self-review の M1(cross-review の返事を待つあいだの「画面が動かなくても待つ」の例外に上限がなかった。画面が動いているあいだだけ待ち、10 分動かなければ leader が BLOCKED を上げる形にし、画面が動かなくても待つのは gate のときだけにした)、L1(`Cross-review: ran (<report の Reviewer:>)` に名前をそろえた)、L3(fan-out の子にも worktree に書かせず、作業ファイルは worktree の外に置かせる)、L4(文の組み立てと折り返し)。leader の雛形と reviewer の雛形、org と director の skill(4 写しずつ)、テスト 4 ファイル。cross-review の reviewer も、10 分たっても画面が動かなければ BLOCKED になる
  - main の取り込み: b98b1b96(#362 と #366。衝突は tech-debt の 1 行)と 6b671276(#363 と #370。衝突は director の skill の 4 写しと tech-debt で、両側の追記を残して直した)。`check-sync.sh` と `check-skill-sync.sh` は通る。verify は b98b1b96、test は 7c7cdec5 の木で回したので、6b671276 の取り込みはどちらの前でもない(sync-docs で `go test ./internal/org/ -run 'Skill|Xreview|DocsMatch'` が ok。`./scripts/run-test.sh` を取り込んだ木で回すかは /pr の前に決める)
  - V1(7c7cdec5): `ralph init` の「変更のレビュー」の問いの説明と `ChangeReview` のコメントに、AI がチームで動くときは、レビューする側のセッションが変更全体をもう一度見る、と足した(Go の文字列とコメントだけ)。CLI が 1 つのときに org の cross-review を聞くかは製品の判断で、この PR では変えず、tech-debt に行を足した
  - cross-review 1 回目の ACTION_REQUIRED 1 の直し(eff83438): reviewer の「cross-review の TASK」の手順 2 で `diff: failed` が出たとき、3 と 4 を飛ばして 5 に進み、git の状態を比べる形にした。`git: changed` なら 5 のとおり BLOCKED(`XREVIEW: git_changed`、report は書かない)、`git: same` のときだけ 6 で `Reviewer status: incomplete (git diff failed: <要点>)` の report を書き、7 で同じ値の RESULT を返す。前の形は、`git diff` が失敗したあとに HEAD が動いても比べを通らず、`git_changed` を上げずに leader が sync-docs に進めた(codex の P2 の指摘。triage は `docs/reports/cross-review-triage-org-xreview-own-model.md`)。AC2 の字面「`git diff` が失敗したら `incomplete (git diff failed: <要点>)` で report を書いて返す」は、AC1 の比べを先に通すので「`git: same` のとき」という条件が付く。AC の文は digest に入るので変えていない(verify 周回 2 の V10)。PR の本文には、AC2 は AC1 の比べのあとで満たす、と書く
  - self-review 周回 2 の LOW 3 件の直し(7eb0179d、文言だけ): C2-1 は `org_runtime_env_test.go` と `herdr_test.go` の `t.Fatalf` の文字列 2 つを、座席が自分の driver と model でレビューする言い方にした。C2-2 は fan-out の子の作業ファイルの決まりを手順 3 から「座席内 fan-out」の節へ移し、区間を「cross-review の TASK の 2 から 5 のあいだ」とし、`.harness/state/` を禁じる理由(座席が 2 と 5 で書く git の状態のファイルと差分のファイルにまぎれうる)を書いた。手順 3 はその節を指す 1 文にした。C2-3 は手順 5 の `git: changed` の括弧を 2 から来た道にも合う言い方にし、手順 2 の `git: same` の項の「出力」を「2 のコマンドの出力」と名指しした
  - テストの行と直し(18360371、a607b72d、5af1a67b): 18360371 は test 周回 2 の Test gaps (cycle 2) の 1 を閉じる。`TestRenderRolePrompt_Reviewer_XreviewCommandOutcomes` に、手順 2 の `diff: failed` のあと手順 5 を走らせる 2 行(何も変えない行は `git: same`、あいだにコミットを足す行は `git: changed`)を bash と zsh で足した。a607b72d は self-review 周回 3 の C2a-1(コメントの「so」)と C2a-2(`goneDiffCmd` の使い回し)、5af1a67b は verify 周回 3 の V13(コメントの「before git diff」が行の固定より強い。順は行の一致のテストが固定する、と書いた)の直しで、どちらもコメントと変数だけ。5af1a67b は verify 周回 3 のあとに入り、test 周回 3 はこの木で回した
- [x] Review artifact created — `docs/reports/self-review-2026-11-22-org-xreview-own-model.md`(周回 1、Merge: yes。CRITICAL 0、HIGH 0、MEDIUM 1、LOW 4。MEDIUM は reviewer が読んでいるあいだの待ちの例外に上限がないこと、LOW は `<相手>` の名前、古いコメントと register の行、fan-out の子の読み取り専用、文の組み立て)。周回 2(HEAD eff83438、Merge: yes。CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3。周回 1 の 5 件のうち M1、L1、L4 は直り、L2 は 2 つの `t.Fatalf` の文字列が残り(C2-1)、L3 は禁止の理由と置き場所が狭い(C2-2)。eff83438 の手順の流れは食い違わず、手順 5 の `git: changed` の括弧が 2 から飛んだ道に合わない(C2-3)。3 件とも文言だけ)。周回 2 の追加(18360371 のテストの行、Merge: yes。CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2。`reviewerXreviewAfterAFailedDiff` のコメントの「so」が因果を逆に読ませること(C2a-1)と、`origin/gone` の手順 2 のコマンドを同じ関数で 2 回組み立てていること(C2a-2)。2 件ともコメントと変数だけ)。C2a-1 と C2a-2 は orchestrator が inline で直した(1 ファイルのコメントと変数だけで、handoff のほうが高くつくため)。確かめ: `gofmt -l` が空、`go vet ./internal/org/`、`go test -count=1 ./internal/org/ -run 'RenderRolePrompt_Reviewer_Xreview'` が ok。機械の自己チェック: 不要な変更、誤字とコピペの誤り、デバッグの残り、秘密情報、スライス外のファイルの 5 つとも clean(最初の Edit で `;` のあとの空白が落ち、`gofmt -w` で直した)
- [x] Verification artifact created — `docs/reports/verify-2026-11-22-org-xreview-own-model.md`(周回 1、HEAD b98b1b96、pass。静的検査は full の範囲で rc=0。AC1〜AC8 を満たす。AC6 はコメントだけの差分で振る舞いを変えずに満たし、字面の「差分に入らない」は満たさない(V7、S4 の記録のとおり)。MEDIUM 1: `ralph init` の「変更のレビュー」の問いが org の cross-review を「もう一方の AI」と説明したまま(V1)。LOW 3: 台帳の Trigger に当たって注のない 10 行、行 343 の古い注、self-review の L2 の残り(V2〜V4、/sync-docs へ))。周回 2(HEAD 7eb0179d、pass。静的検査は取り込みのあとの木で full の範囲で rc=0。AC1〜AC8 はこの周の変更のあとも成り立ち、AC2 は eff83438 で AC1 の比べのあとに満たす形になった。手順 2・3・5・6・7 の流れと leader の `git_changed` と incomplete の扱いは食い違わない。LOW 1: 台帳の 432 と 377 の注がこの周の直しを書いていない(V9、/sync-docs へ)。INFO 3: Progress にこの周の直しの記録がない(V10)、手順 3 の書き方が 2 とそろっていない(V11)、self-review L1 の台帳の行(V12、/pr へ))。周回 2 の追加(Run 3、HEAD a607b72d、pass。静的検査は full の範囲で rc=0。18360371 の 2 行は `diff: failed` のあとの手順 5 を雛形のコマンドで走らせ、test の Test gaps (cycle 2) の 1 の B01・B02・B04 と B05 は、bash と zsh の probe でどれかの行の期待から外れる。INFO 2: コメントの「before git diff」が行の固定より強い(V13)、テストのコメント 2 か所が指す triage の所見は、2 周目の /cross-review が周 1 の節を残すあいだだけ読める(V14)。V9・V10・V12 は開いたまま)
- [x] Test artifact created — `docs/reports/test-2026-11-22-org-xreview-own-model.md`(周回 1、HEAD 7c7cdec5、pass。`./scripts/run-test.sh` は rc 0、1,836 s。shell 46 本と Go 11 パッケージで落ちたテストは 0、`(cached)` なし。依頼の新しいテスト(bash と zsh でのコマンドの run、git の状態が変わる 4 種類、前の版の TASK の行、R1 の待ちの決まり)は `-count=1` で通った。変異 25 個のうち 22 個が落ちた。残った 3 つは、git の状態の部品(`git diff`、`git diff --cached`、`git status`)を雛形と共有の定数から一緒に外したもので、雛形だけならコマンドの行の一致で落ちる。Test gaps の 1 に LOW で記録した)。周回 2(HEAD 5c5ead66、pass。`./scripts/run-test.sh` は rc 0、1,350 s。shell 46 本と Go 11 パッケージで落ちたテストは 0、`(cached)` の 5 パッケージは `-count=1` でも通った。eff83438 と 7eb0179d の文を外す変異 F01〜F11 と B01t は落ちた。eff83438 の道(`diff: failed` のあと手順 5 の比べ)は HEAD で動くが、雛形のコマンドで走らせるテストはなく、コマンドとテストの固定を一緒に変える変異 3 つが残る。Test gaps (cycle 2) の 1 に LOW で記録した)。周回 2 の追加の周(Run 3、HEAD 5af1a67b、pass。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` は rc 0、1,288 s。shell 46 本と Go 11 パッケージで落ちたテストは 0、`(cached)` の 8 パッケージは `-count=1` でも通った。18360371 の 4 行(bash と zsh の `step 5 after diff: failed` と `…and a new commit`)は `-v` の run で PASS、SKIP なし。Run 2 で残った変異 B01・B02・B04・B12 と、逆向きの B05 は、この 4 行で落ちる。Test gaps (cycle 2) の 1 は閉じた)
- [x] Sync-docs artifact created — `docs/reports/sync-docs-2026-11-22-org-xreview-own-model.md`(周回 1。文書の直しは 0d44799f)。self-review の L2(`org_runtime_env_test.go` のコメントと tech-debt の `pick_reviewer` の行)、verify の V1(`ralph init` の問いは tech-debt の新しい行に)、V2(Trigger に当たった行の注)、V3(行 343)、V5(行 432 の注の順)を直し、仕様の 2026-11-22 の項に R1 の待ちの決まりを足した。注は 13 行(verify が挙げた 10 行と、R1 の implementer が挙げた 202 の (g)、sweep で見つけた 88 の (7) と 242 の (o)(p))で、432 は R1 の分を足し、343 は 2026-11-22 の注を足した。周回 2(HEAD 45e8eadc からの run): 台帳の 432 と 377 に eff83438 と 7eb0179d の分を足し(verify 周回 2 の V9)、432 の周回 1 の注の誤り(f95a08f9 の fan-out の 1 文は手順 3 にあり、(2) にも当たっていた)を直し、370 と 190 の `wc -l` を 5af1a67b の数にし(leader.md の 1,646 は 7c7cdec5 でなく main の取り込みのあとの数)、新しい行 557(`Cross-review: ran (...)` の括弧の重なり、V11、test 周回 1 の Test gaps 1 と 2)を足した。V10 はこの Progress の sub-bullet で閉じた。V13 は 5af1a67b、self-review 周回 2 の (1)〜(3) は 7eb0179d、周回 3 の (5)(6) は a607b72d で直っているので、台帳の行にはしていない
- [ ] PR created
