# org-narrow-verification

- Status: Approved
- Approved: 2026-10-27 sha256:97abe345a188
- Owner: Claude Code
- Date: 2026-10-27
- Related request: 機能の org のスライスの検証を絞り、reviewer のフルの test を gate の 1 回にする(issue #262。2026-10-27 にユーザーと決めた方針: leader が計画のスライスに検証コマンドを書いて implementer の TASK に写す、implementer は機械的な確かめと絞った検証の結果を RESULT に書く、reviewer のレビューの TASK は `run-static-verify.sh` と差分のレビューだけにする)
- Related issue: #262
- Type: perf
- Branch: perf/org-narrow-verification

## Objective

機能の org で、同じフルの test が何度も走るのをやめる。今は implementer がスライスごとに `./scripts/run-verify.sh`(静的検査とテストを全言語で)を打つ。leader の TASK が検証コマンドを書かないため、雛形の既定がそのまま使われる。reviewer はレビューの TASK で `run-static-verify.sh` と `run-test.sh` を打ち、PR の前の gate の TASK でも `ralph org gate` が同じ 2 つを走らせる。そのため、1 つの org でフルの test が「スライス数 + 2」回走る。このリポジトリの `run-test.sh` は多くが 640〜1,090 秒かかる。レビューの時点の test は、そのあとのコミットと rebase で push する HEAD の検査ではなくなる。push する HEAD を確かめるのは gate だけになる。

この計画では、#261 で標準フローに入れた形を org の雛形に移す。leader は機能の計画のスライスごとに検証コマンドを書き、implementer への TASK に写す。implementer はそのコマンドと、コミットの前の機械的な確かめを行い、結果を RESULT に書く。機能ごとの org では、leader が reviewer へのレビューの TASK に `run-static-verify.sh` だけを指示し、レビューは静的検査と差分のレビューだけにする。フルの test は gate の TASK の 1 回にする。指示のないレビューでは、reviewer は今のまま両方を打つ(D6)。

## Scope

- S1(implementer の雛形): `internal/org/prompts/implementer.md`
  - スライスの検証コマンドは leader の TASK のもの(本文の `VERIFY:` の行、D4)を打つ。
  - TASK にコマンドがなければ、そのスライスが触ったファイルを覆う、いちばん狭い検査を選ぶ。Go なら触ったパッケージの gofmt・vet・test。`./scripts/run-verify.sh` は、それより狭いものがないときだけ使う(D1)。選んだコマンドは RESULT に書く。
  - `RALPH_VERIFY_SCOPE=changed` では絞れないことも書く。
  - コミットの前に、stage した差分(`git diff --cached`)で機械的な確かめを行う(D2)。項目は 5 つ: 不要な変更、誤字とコピペの誤り、デバッグの残り、秘密情報、scope の外のファイル。
  - 範囲内で消せない秘密情報があれば、コミットせずに unstage し、作業ツリーからも出して BLOCKED を返す。BLOCKED にはファイルと行を書き、値は書かない。
  - RESULT の契約と例に、検証コマンドとその結果、5 項目それぞれの結果(clean、fixed、found and not fixed)を足す。
  - 雛形のテスト(`internal/org/prompts_test.go`)を足す。
- S2(reviewer の雛形): `internal/org/prompts/reviewer.md`
  - レビューの TASK のゲートの既定は今のまま変えない。leader の TASK にコマンドの指示があればそれだけを打ち、なければ `./scripts/run-static-verify.sh` と `./scripts/run-test.sh` を打つ(D6)。機能ごとの org の leader は `./scripts/run-static-verify.sh` だけを指示するので(S3)、その org のレビューでは `run-test.sh` を打たない。この節に、指示があるときは足しも削りもしないことと、機能ごとの org のフルの test は PR の前の gate の TASK で `ralph org gate` が 1 回走らせることを書く。
  - レビューの RESULT と BLOCKED の `GATE:` は、打ったコマンドの結果を指すと書く(D5)。
  - 差分のレビューの観点を 10 項目と受け入れ条件にする(D3)。10 項目は、命名、読みやすさ、不要な変更、誤字、null の扱い、デバッグの残り、秘密情報、例外処理、security、保守性。
  - レビューの RESULT と BLOCKED の本文に、打ったコマンドを `CHECKS:` の行で書く。leader は、どのコマンドの結果の `GATE:` かをこの行で確かめられる(D5)。
  - 座席内 fan-out の例と BLOCKED の例を、静的検査の失敗に合わせる。RESULT の例のヘッダ(`GATE:`・`HEAD:`・`BASE:`)と `EVIDENCE:` の行の形は変えない。`/org` skill の RESULT の例と突き合わせるテスト(`internal/org/report_names_sync_test.go` の `TestOrgSkill_ResultExampleFollowsTheReviewerTemplate`)があるため。
  - gate の TASK の節は変えない。
  - 雛形のテストを直し、足す。
- S3(leader の雛形): `internal/org/prompts/leader.md`
  - 機能ごとの org の手順 1 で、計画の `## Implementation outline` のスライスごとに検証コマンドを書く。雛形に欄があればそれに沿う。中身は、触ったパッケージの gofmt・vet・test 程度にする。`RALPH_VERIFY_SCOPE=changed` は Go では `go test ./...` を全部走らせるので、絞り込みに使わない。
  - implementer への TASK に、そのスライスのコマンドを `VERIFY:` の行で写す。ミッションの 3 にも同じことを書く(機能ごとでない org にも当てはまる)。
  - 機能ごとの org では、reviewer へのレビューの TASK に、コマンドとして `./scripts/run-static-verify.sh` だけを書く(D6)。フルの test は手順 12 の gate の TASK で走るので、レビューでは打たせない。機能ごとの org の手順 8(TASK・RESULT・レビューの往復)に書く。機能ごとでない org はコマンドを書かず、reviewer の既定(静的検査とテストの両方)のままにする。
  - typed protocol の節に、implementer への TASK の例(`VERIFY:` の行つき)を足す。
  - 雛形のテストを足す。
- S4(/org skill): `.claude/skills/org/SKILL.md` の「役割」の implementer と reviewer、「座席内 fan-out」の reviewer の例を S1〜S3 に合わせる。ほかに、reviewer がレビューで必ず `run-test.sh` を打つと読める箇所があれば、機能ごとの org では打たないことを足す。写しは `scripts/sync-skills.sh` で作る。
- S5(文書): 次の文書の、reviewer がレビューで `run-test.sh` を打つという記述に、機能ごとの org では leader が静的検査だけを指示し、フルの test は gate の 1 回になることを足す。
  - `docs/quality/quality-gates.md` の Quality pipeline gate の行(root と `templates/base/`)
  - `docs/recipes/codex-seat-permissions.md` の reviewer の段落(root と `templates/base/`)
  - `docs/specs/2026-08-01-org-runtime.md` に「2026-10-27 改訂」の節を足す。本文の履歴は書き換えない。これまでの改訂と同じ扱い。

## Non-goals

- `ralph org gate` と `ralph org regate` の振る舞い(どちらも今のまま `run-static-verify.sh` と `run-test.sh` を走らせる)
- regate の test を CI に任せること(#264 で扱う)
- 標準フローの `/implement`・`/self-review`(#261 で済み)
- leader の sync-docs(#268)、reviewer の cross-review(#269)
- hook の「Run ./scripts/run-verify.sh before claiming done」の文(#261 の tech-debt の行のまま)
- director の雛形(検証を打たない)

## Assumptions

- 雛形は go:embed で binary に入る。新しい binary で spawn した座席から効き、古い binary で立てた org の座席は今の雛形のまま動く
- typed protocol の本文の 2,000 字の上限の中に、`VERIFY:` の 1 行は収まる
- 機能の計画の雛形は、プロジェクトの `docs/plans/templates/feature-plan.md`(#261 で検証コマンドの欄を足した)を使う。雛形に欄がないプロジェクトでも、leader の雛形の文だけで書ける
- `go test ./internal/org/` の全体は 300 秒前後かかるので、スライスの検証は `-run` で雛形のテストに絞る

## Affected areas

- `internal/org/prompts/implementer.md`、`internal/org/prompts/reviewer.md`、`internal/org/prompts/leader.md`(go:embed の雛形)
- `internal/org/prompts_test.go`
- `.claude/skills/org/SKILL.md` と写し(`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`)
- `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md`
- `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md`
- `docs/specs/2026-08-01-org-runtime.md`

## Visual review

- Page: `.harness/state/plan-visual/org-narrow-verification.html`(やさしい図 2 つ: `overview`、`easy-handoff`。詳しい図 2 つ: `detail-overview`、`detail-sequence`)
- Lint: `./scripts/plan-visual.sh lint` が終了コード 0
- Self-check: 全体のページと 4 つの図を `shot` で撮って読んだ。2 つ目のやさしい図の説明の行と、詳しい図 2 の gate の箱の文字がはみ出していたので直し、撮り直して重なり・切れ・線の通り抜けがないことを確かめた。Codex の計画助言を受けて D6 を改めたあと、文を合わせた 3 つの図(全体のページ、詳しい図 1 と 2)を撮り直して読んだ

## Design decisions

Critical forks: None。分かれ目はどれも、issue #262 の方針か #261 で決めた形で決まる。理由を次に残す。

- D1: TASK に検証コマンドがないときの implementer の既定は、触ったファイルを覆ういちばん狭い検査にし、`run-verify.sh` はほかにないときだけにする。#261 の `/implement` 手順 6 と同じ扱い。既定を `run-verify.sh` のままにすると、TASK に書き忘れたときにスライスごとのフルの test が戻る
- D2: 機械的な確かめは、スライスごとのコミットの前に、stage した差分で行う。issue の「最後の RESULT の前に」は、これで満たせる。#261 の implementer と同じ時点なので、2 つの雛形の読み方がそろう。項目の 5 つ目は、org では予約に当たる「scope の外のファイル」にする
- D3: reviewer の差分のレビューは、issue の文のとおり 10 項目と受け入れ条件にする。標準フローの `/self-review` は 6 観点に絞ったが、その代わりに独立した `/verify` と CI の secret scan がある。org では reviewer が implementer 以外で差分を読む唯一の座席なので、implementer の機械的な確かめと重なっても、10 項目を残す
- D4: 検証コマンドは TASK の本文の `VERIFY:` の行で渡し、ヘッダにはしない。ralph は TASK の本文を読まず、ヘッダの `PLAN:` だけを承認の検査で読む。ヘッダにすると、ralph が検査する行と見分けにくくなる
- D5: レビューの RESULT と BLOCKED の `GATE:` は、打ったコマンドの結果を指すと雛形に書き、打ったコマンドを `CHECKS:` の行で返させる。機能ごとの org では、`GATE: pass` は静的検査が通ったことだけを意味する。leader の手順は `GATE:` の値で分かれるので、ヘッダの名前は変えない。フルの test の結果は gate の TASK の `GATE:` で見る
- D6: reviewer のレビューの既定(コマンドの指示がないとき)は、今のまま静的検査とテストの両方にする。test を外すのは、機能ごとの org の leader がレビューの TASK に `./scripts/run-static-verify.sh` だけを指示したときに限る。初稿では逆に、reviewer の既定を静的検査だけにして、機能ごとでない org の leader に 2 つのコマンドを書かせていた。Codex の計画助言(HIGH)が次の 2 つを指摘したので改めた。
  - leader が書き忘れると、テストを走らせないまま `GATE: pass` になる。
  - 雛形は spawn のたびに展開される(`internal/org/spawn.go`)。新しい binary の reviewer を古い leader が立てると、2 つのコマンドを書かない leader と、テストを省く reviewer が組み合わさる。
  - 今の形では、書き忘れても、版が混ざっても、テストは打つ側に倒れる。
    - 古い leader と新しい reviewer: leader が指示しないので、reviewer は両方を打つ。
    - 新しい leader と古い reviewer: 古い雛形も「指示があればそれ」なので、静的検査だけを打つ。
  - 機能ごとの org の結果は、初稿と同じになる。issue の受け入れ条件 3 の「reviewer のレビューの TASK が `run-test.sh` を打たない」は、機能ごとの org のレビューの TASK について満たす。

## Acceptance criteria

- [x] AC1: leader の雛形の TASK の例に検証コマンド(`VERIFY:` の行)があり、機能ごとの org の手順 1 に、計画のスライスごとの検証コマンドと `RALPH_VERIFY_SCOPE=changed` の注意がある(issue の受け入れ条件 1)
- [x] AC2: implementer の雛形の RESULT の契約に、機械的な確かめ(5 項目それぞれの結果)と、打った検証コマンドとその結果がある。TASK にコマンドがないときの既定が `run-verify.sh` ではなく、いちばん狭い検査になっている(issue の受け入れ条件 2)
- [x] AC3: 機能ごとの org では、reviewer のレビューの TASK が `run-test.sh` を打たない。leader の雛形の手順 8 が `./scripts/run-static-verify.sh` だけを指示し、reviewer の雛形は指示があればそれだけを打つ(足しも削りもしない)。指示がないときの reviewer の既定は、今のまま両方。gate の TASK は今のまま `ralph org gate` を打つ(issue の受け入れ条件 3、D6)
- [x] AC4: `/org` skill の「役割」と「座席内 fan-out」が S1〜S3 と食い違わず、4 つの写しが一致する
- [x] AC5: quality-gates・recipe・仕様の reviewer の記述が S2 と食い違わない
- [x] AC6: 雛形のテスト(`internal/org/prompts_test.go`)と skill の同期の検査(`scripts/check-skill-sync.sh`、`scripts/check-sync.sh`、`internal/org` の `*_sync_test.go`)が通る(issue の受け入れ条件 4)

## Implementation outline

1. S1: implementer の雛形とそのテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && go test ./internal/org/ -run 'TestRenderRolePrompt|TestRolePrompts|TestRetiredRole'`
2. S2: reviewer の雛形とそのテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && go test ./internal/org/ -run 'TestRenderRolePrompt|TestRolePrompts|TestRetiredRole'`
3. S3: leader の雛形とそのテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && go test ./internal/org/ -run 'TestRenderRolePrompt|TestRolePrompts|TestRetiredRole'`
4. S4: `/org` skill と写し — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && go test ./internal/org/ -run 'OrgSkill|ReportNames|SendDefaults|DirectorSkill'`
5. S5: quality-gates・recipe・仕様 — verify: `./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、shellcheck、secret scan)、`./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`
- Spec compliance criteria to confirm:
  - AC1〜AC6
  - 雛形の 3 つが互いに食い違わないこと。leader が写す `VERIFY:` の行を implementer が読む。reviewer の `GATE:` の意味を leader が読む
  - gate の TASK の節が変わっていないこと
- Documentation drift to check:
  - `run-test.sh` を grep し、reviewer のレビューの TASK が打つと書いた残りがないこと。root、`templates/base/`、`docs/`、`.claude/` を見る
  - README の `ralph org gate` の段落と、`/org` skill の gate の節が今のままで正しいこと
- Evidence to capture: `docs/reports/verify-2026-10-27-org-narrow-verification.md`

## Test plan

- Unit tests(`internal/org/prompts_test.go`):
  - implementer: `VERIFY:` のコマンドを打つこと、既定がいちばん狭い検査であること、5 項目の確かめが stage した差分に対してあること、秘密情報の BLOCKED に値を書かないこと、RESULT の例と契約にこの 2 つの結果があること。`run-verify.sh` を既定として書いていないこと
  - reviewer: `## ミッション` の節を区切って、レビューの TASK の項目に次があること。指示があればそれだけを打つ(足しも削りもしない)。指示がなければ `run-static-verify.sh` と `run-test.sh` の両方を打つ(既存のテストを残す)。機能ごとの org のフルの test は gate の 1 回。ほかに、`GATE:` が打ったコマンドの結果を指すこと、`CHECKS:` の行、10 項目と受け入れ条件。gate の TASK の節(`reviewer.md` の「gate の TASK」)と、leader の手順 12 の既存のテスト(`TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`)は、変えずに通り続けること
  - leader: 手順 1 の検証コマンドと `RALPH_VERIFY_SCOPE=changed` の注意、TASK の例の `VERIFY:` の行、ミッションの 3 の文。機能ごとの org の手順 8 で、レビューの TASK のコマンドを `./scripts/run-static-verify.sh` だけにすること(D6)。機能ごとでない org にはその指示がないこと(ミッションに `run-static-verify.sh` だけを指示する文がない)
- Integration tests: なし(座席の実機の起動は確かめない。既存の spawn のテストは雛形の中身を見ない)
- Regression tests:
  - `go test ./internal/org/` の全体(gate、regate、spawn の既存のテスト)
  - `internal/org` の `*_sync_test.go`(`/org` skill の写し)
  - `run-test.sh` のフルの実行は /test で行う
- Edge cases:
  - leader が指示を書き忘れたとき(機能ごとでない org、古い leader): reviewer の既定が両方のままであること(D6)。reviewer のミッションの既定の文を固めるテストで確かめる
  - 機能ごとの org の手順 8 の指示が、手順 12 の gate の TASK の本文と混ざらないこと(手順を区切って確かめる)
- Evidence to capture: `docs/reports/test-2026-10-27-org-narrow-verification.md`。テストの変異を 1 つ以上試す(reviewer のミッションに `run-test.sh` を戻すと落ちる、など)

## Risks and mitigations

- R1: レビューで test を打たなくなるので、テストの失敗が見つかるのが gate(PR の前)まで遅れる。
  - 手当て: implementer がスライスごとに、触ったパッケージのテストを打つ。gate で落ちたら、leader の手順 12 の `GATE: fail` の枝で implementer に差し戻す。この道は今もある
- R2: 機能ごとの org の leader が手順 8 の指示を書き忘れる。
  - 手当て: D6。reviewer は既定の両方を打つので、テストは抜けない(遅くなるだけ)。reviewer の `CHECKS:` の行で、leader は何が走ったかを確かめられる
- R3: leader が `VERIFY:` を書き忘れる。
  - 手当て: implementer の既定をいちばん狭い検査にし、選んだコマンドを RESULT に書かせる。leader はそれを読んで裁定できる
- R4: 雛形の文を変えると、既存のテストの照合文字列が崩れる。
  - 手当て: スライスごとに `-run` で雛形のテストを回す。/test で `internal/org` の全体を回す

## Rollout or rollback notes

- 雛形は go:embed で binary に入る。新しい binary で spawn した座席から効く。release のノートに次を書く。
  - 機能の org の reviewer は、レビューで `run-test.sh` を打たない。フルの test は PR の前の gate の 1 回になる
  - implementer の既定の検証は、`run-verify.sh` ではなく、触ったところの検査になる
- コミットの型は `perf:` にする(`.goreleaser.yml` の changelog に出る)
- 戻し方: この PR を revert する。古い雛形に戻るだけで、台帳の形は変わらない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(consult の判定は「進めてよい」。S4 の検証コマンドの `-run` を実在のテスト名に直し、`CHECKS:` の行を足した。Codex の計画助言は HIGH 1 件で、D6 の向きを逆にして反映した。reviewer の既定は今のまま両方にし、機能ごとの org の leader だけが静的検査を指示する)
- [x] Plan approved(自動承認。2026-10-11 のユーザー指示「作成したプランは自動承認します」と、2026-10-27 の「最後まで止まらずに」による)
- [x] Branch created
- [x] Implementation started(スライスは implementer に委譲)
  - S1 8ee986bd: implementer の雛形。`VERIFY:` の行、いちばん狭い検査、5 項目の確かめ、秘密情報の BLOCKED、RESULT の例の `VERIFY:` と `SELF-CHECK:` の行。テスト 3 本を足し、変異 6 つがそれぞれ落ちることを確かめた。検証: 雛形のテスト 40 本が通過、`gofmt`・`go vet` OK。機械的な確かめは 5 項目とも clean
  - 逸れ(S3 に足す): RESULT の例と TASK の `VERIFY:` の行には `$(gofmt -l …)` が入る。座席が本文をコマンドの引数に二重引用符で直接書くと、shell が展開してしまう(consult の指摘)。leader の雛形は受信箱の節でだけ、ファイルに書いて `--text "$(cat <ファイル>)"` で渡すよう書いている。S3 で、leader の TASK と implementer の RESULT の本文もファイルに書いて渡す 1 文を足す。S3 の範囲に `internal/org/prompts/implementer.md` を加える
  - S2 3478cb68: reviewer の雛形。指示があればそれだけを打つ(既定は両方のまま)、機能ごとの org の leader は静的検査だけを指示する、ミッションの 5 に `GATE:` と `CHECKS:`、10 項目、例に `CHECKS:` の行、BLOCKED の例を静的検査の失敗に。typed protocol の節に「`CHECKS:` は本文の行」、fan-out の例に「子にも同じコマンドだけ」を足した(handoff の外の小さな追加)。テスト 4 本を足し、変異 14 個がそれぞれ落ちることを確かめた。検証: 雛形のテスト 44 本と `OrgSkill|ReportNames` の 3 本が通過。機械的な確かめは 5 項目とも clean
  - 逸れ(S5 に足す): `docs/tech-debt/README.md` の「A feature org's PR runs the tests twice」の行は、この計画で解消する。S5 で解決済みにし、「The reviewer seat runs the same scripts」と書くほかの行も確かめる
  - S3 3989e87f: leader の雛形。ミッションの 3 に `VERIFY:` の行(どの org でも)と、機能ごとでない org はレビューの TASK にコマンドを書かないこと。手順 1 にスライスごとの `— verify:` と `RALPH_VERIFY_SCOPE=changed` の注意、手順 5 に `VERIFY:` への写し、手順 8 にレビューの TASK は静的検査だけと `CHECKS:` の確かめ。typed protocol に本文をファイルで渡す決まりと implementer の TASK の例。implementer の雛形にも送り方の 1 文(上の逸れ)。テスト 6 本を足し、変異 13 個がそれぞれ落ちることを確かめた。検証: 雛形のテスト 50 本が通過。機械的な確かめは 5 項目とも clean
  - S4 a8253fa9: `/org` skill の 4 つの写し。fan-out の例、「役割」の implementer と reviewer、「org の終わり方」の冒頭に検証の分け方の段落、typed protocol の RESULT の例に `CHECKS:`。handoff の外で「Leader 運用 2 経路」の手順 3 にも `VERIFY:` の 1 文を足した(昇格したセッションの leader は雛形を読まず、この skill だけを読むため)。検証: check-skill-sync ok、check-sync PASS、`OrgSkill|ReportNames|SendDefaults|DirectorSkill` の 11 本が通過。機械的な確かめは 5 項目とも clean
  - S5 3801a512: quality-gates(root と template を別々に)、codex-seat-permissions の recipe(2 つの写し)、仕様に「2026-10-27 改訂」の節と FR-7 の印、tech-debt の「tests twice」の行を解決済みにし、ほかの 2 行の reviewer の文を直した。検証: check-sync PASS、recipe の写しは一致、tech-debt の列の数は変わらない。機械的な確かめ: 誤字とコピペの誤りを 1 件 fixed(仕様の (d) の言い方を D6 にそろえた)、ほかの 4 項目は clean
- [x] Review artifact created(self-review 06f60e10: LOW 8、Merge yes)
  - 直し F1 0ea899f5(implementer): レビューの TASK のコマンドを本文の `CHECKS:` の行にし、返った `CHECKS:` の行が欠けたら送り直す。implementer の scope の例外、1 つの TASK で複数スライスのときは `VERIFY: S1: …` の行をスライスごとに、コミットの前の確かめの順。テスト 3 本を足し、変異 11 個がそれぞれ落ちることを確かめた。機械的な確かめは 5 項目とも clean
  - 直し F2 f0870d6e(implementer): 送り直しに上限(2 回続けて通らなければ BLOCKED)、`/org` skill の 4 つの写しを F1 にそろえた(`CHECKS:` の形、送り直し、複数スライス、`send` の例を `.harness/state/` に)。このブランチで足したテストを `internal/org/prompts_verification_test.go` に移し(`prompts_test.go` は main の版と同じに戻った)、補助を読みやすくした。テスト 53 本は移す前後で同じ。機械的な確かめは 5 項目とも clean
  - self-review の 2 回目 f2d075c8: 1 回目の 8 件はすべて解消、新しい LOW 2、Merge yes
  - 直し F3 8cda482e(implementer): 複数スライスの RESULT はスライスごとに名前付きの `COMMIT:`・`VERIFY:`・`SELF-CHECK:`、reviewer が指示とみなすのは TASK の本文の `CHECKS:` の行だけ(行がなければ既定の両方。散文の指示を受け入れる文を外した。打つ検査が増える側にしか動かない)。skill の 4 つの写しも同じに。テスト 2 本を `internal/org/prompts_verification_multi_test.go` に足し、変異 6 つがそれぞれ落ちることを確かめた。機械的な確かめ: 不要な変更 fixed(テストのコメントの折り直しを戻した)、誤字とコピペの誤り fixed(重言)、ほかの 3 項目は clean
  - 残り(sync-docs に渡す): tech-debt の「org runtime FR-7 quality gate is prompt-instructed, not hook-enforced.」の行にも「The reviewer seat re-runs `run-static-verify.sh` / `run-test.sh`」という古い文がある。trigger は「The next change to the org quality pipeline」で、この計画も当たる。払わずに繰り越す注記を、2026-10-26 の reviewer-runs-gate のときと同じ形で足す
- [x] Verification artifact created(verify 584f8103: pass、AC1〜AC5 を満たし、AC6 は静的な半分。self-review の 2 回目の LOW 2 件は F3 で解消を確認。所見は LOW 1(director の仕様 2026-10-07 の FR-7 の「テストは review と gate の 2 回」が古い)と既知 1(tech-debt の 157 行目)で、どちらも sync-docs に渡す)
  - 逸れ: テストは AC6 と S1 が名指す `internal/org/prompts_test.go` ではなく、F2 と F3 で `internal/org/prompts_verification_test.go` と `internal/org/prompts_verification_multi_test.go` に置いた(`prompts_test.go` は main の版のまま)
- [x] Test artifact created(test 3afdd03a: pass。`run-test.sh` が rc 0、1,572 秒で、shell 45 本と Go のテストのある 9 パッケージがすべて通過。名指しの雛形のテスト 66 本と `TestStartFeature` 27 本が通過。変異 a〜d(c は 3 つの形)はそれぞれ落ち、対照は通った。座席の実機は確かめていない)
- [x] Cross-review 1 回目(triage 8438af88: ACTION_REQUIRED 1)。implementer の秘密情報の片付けが、座席が始める前から stage されていた scope の外の変更まで消しうる
  - 直し F4 ffafc9d2(implementer): 直すのはこのスライスの変更だけ。前からあった変更(stage 済みを含む)は書き換えも削除も unstage もしない。前からあった変更の秘密情報と、前から stage されていた scope の外のパスは、コミットせずに BLOCKED。skill の 4 つの写しも同じに。テスト 2 本を足し、変異 7 つがそれぞれ落ちることを確かめた。機械的な確かめは 5 項目とも clean
  - 隣の件(sync-docs に渡す): 標準フローの `/implement` 手順 7 も、止まった slice の秘密情報を「新しい未追跡のファイルなら消す」としている。止まった slice が作ったファイルに限る文がないので、同じ形の穴がある。#261 で merge 済みの文で、この計画の範囲の外なので、tech-debt に記録する
  - cycle を 2 にして、/self-review から回し直す
- [x] Cycle 2 の self-review 1 回目(dd2ffa05: LOW 5、Merge yes。cross-review の指摘は解消)
  - 直し F5 a7d879d9(implementer): 「前からあった変更」を TASK ごとの記録で 1 度だけ定義し、呼び方をそろえた。コミットしない 3 つの場合を 1 文で並べた。unstage はパスを指定した `git restore --staged -- <パス>` にし、パスなしの `git reset` を禁じた。F4 のテストを `internal/org/prompts_verification_predate_test.go` に移し、テストを 1 本足した。変異 3 つがそれぞれ落ちることを確かめた。機械的な確かめは 5 項目とも clean
  - LOW 4(標準フローの implementer の定義と `/implement` の手順 7 の unstage と削除)は、#261 の文でこの計画の範囲の外なので、sync-docs で tech-debt に記録する。Claude と Codex の agent 定義、skill、それぞれの `templates/base/` の写しを名指す
  - 直し F6 854be9db(implementer): 止まったスライスの再開。leader は頼み直す TASK の本文の `RESUME:` の行に、止まった TASK が残したパスを並べ、秘密情報で止まったあとは値が消えたことを確かめる(消えていない、または前からあった変更にあるなら escalate)。implementer は `RESUME:` のパスの変更を引き継ぎ、scope の外なら BLOCKED。skill の 4 つの写しも同じに。テスト 3 本を足し、変異 3 つがそれぞれ落ちることを確かめた。機械的な確かめは 5 項目とも clean
  - 直し 51cf59e2(inline): F6 の implementer が挙げた穴。スライスを狭めるときも残したパスはすべて `RESUME:` に並べ、外すパスの変更は元に戻すよう TASK に書く(leader はコードに触れないため)。検証: `gofmt`・`go vet` と雛形のテストが通過。この文を消すとテストが落ちることを `-overlay` で確かめた。機械的な確かめ: unnecessary changes clean / typos and copy-paste errors clean / debug leftovers clean / secrets clean / files outside the slice clean
- [x] Cycle 2 の self-review 2 回目(bd21c5fd: LOW 5、Merge yes。前回の 1〜3 は解消、5 は一部)
  - 直し F8 28d83b96(implementer): R2-1 複数スライスの `RESUME:` の行にもスライス名を付け、その名前のスライスのコミットに入れる。R2-2 skill の leader の手順 3 に、値が消えていなければ escalate と、狭めるときの文。R2-3 leader の「前からあった変更」の説明を implementer の定義にそろえた。R2-4 「止まった TASK がコミットせずに残したパス」と明記し、折り返しを直した。変異 5 つがそれぞれ落ちることを確かめた。機械的な確かめは 5 項目とも clean
  - R2-5(テストの整理: 同じ文を 2 か所で固定、ファイル名が `predate` のまま再開のテストも入る、順序の照合が 2 通り、テスト名の語)は、前回の所見 4(標準フローの面)と一緒に、sync-docs で tech-debt に記録する。直すたびに新しい LOW が出ているので、self-review はここで止め、最終の diff は verify と cross-review が見る
- [x] Cycle 2 の verify(verify 7458318d: pass。AC1〜AC5 を満たし、AC6 は静的な半分。cross-review の指摘と self-review の R2-1〜R2-4 の解消を確認。所見は LOW 1(仕様 2 つに `RESUME:` の行と片付けの境界がない)と既知 1(tech-debt に R2-5 と標準フローの片付けの 2 行)で、どちらもこの cycle の sync-docs で扱った)
- [x] Cycle 2 の test(test 1637025d: pass。`run-test.sh` は rc 1 で、落ちたのは前からある不安定な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` 1 本だけ。`internal/org` を `-count=1` で打ち直すと通った。雛形の組のトップレベル 72 本と `TestStartFeature` 27 本が通過。変異 a、b1、b2、c1、c2、d はそれぞれ 1 本のテストが落とし、対照は通った。座席の実機は確かめていない)
- [x] Cycle 2 の sync-docs(この commit: 仕様 2 つの 2026-10-27 の項に `RESUME:` の行と片付けの境界を足した。tech-debt に R2-5 の行と標準フローの片付けの行を足し、不安定なテストの行に 3 回目を足した)
- [x] Cross-review 2 回目(triage 45f8b554: ACTION_REQUIRED 2、cap に届いた)。1. 狭めた `go test` に timeout がなく、Go の既定の 10 分で打ち切られる(言語パックの `run_tests` は `GOFLAGS` の先頭に `-timeout=30m` を置く。`internal/org` は負荷の下で 943 秒かかった)。2. 入れ子の `go.mod` の module を repo の root から打つと落ちる(言語パックは module ごとに cd する)
  - consult の判定は「進めてよい」(案 A)。コマンドは `test -z "$(gofmt -l <dir>)" && go vet ./<dir>/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./<dir>/` にする。入れ子の module では、その `go.mod` のディレクトリへ `(cd <module> && …)` で入り、`<dir>` をそこからの相対のパスにする。手元の確かめ(scratchpad の使い捨ての repo、go1.27.3): root からの `go vet ./service/pkg/` は「cannot find main module」で落ち、`(cd service && …)` は通る。テストの残り時間は 30m、呼び手の `GOFLAGS=-timeout=45m` と `go env -w GOFLAGS=-timeout=50m` はあとに来て勝つ。root に `go.work` があると root からも通る
  - 逸れ: 同じ裸のコマンドが plan の雛形の 6 つの写しにもある(`docs/plans/templates/feature-plan.md:39` と、`.claude/skills/plan/template.md`・`.agents/skills/plan/template.md`、それぞれの `templates/base/` の写し。#261 で入った)。leader の手順 1 は「雛形に欄があればそれに沿う」ので、org の雛形と形をそろえる。D1 の「触ったパッケージの gofmt・vet・test」は変わらないので、計画の本文は変えない
  - skill のとおり cap を上げて回し直す(`RALPH_STANDARD_MAX_PIPELINE_CYCLES=3`、`cycle-count.json` は 2 のまま)。直しは F9(org の雛形、skill の 4 つの写し、Go のテスト)と F10(plan の雛形の 6 つの写しと `tests/test-slice-verification.sh`)の 2 スライス
  - 直し F9 3c64d65d(implementer): implementer と leader の雛形のコマンドと例を新しい形にし、timeout の理由と入れ子の module の `(cd <module> && …)` の文を足した。skill の 4 つの写しの implementer の箇条も同じに。テスト 2 本を足し(1 本は言語パックの `run_tests` が今も `-timeout=30m` で始まることも確かめる)、変異 11 個がそれぞれ狙ったテストだけで落ちることを確かめた。検証: 雛形と skill のテストが通過、check-skill-sync と check-sync が通過。機械的な確かめは 5 項目とも clean
  - 直し F10 662d3e47(implementer): plan の雛形の 6 つの写しの例を同じ形にし、timeout の理由と入れ子の module の例を足した。`tests/test-slice-verification.sh` に写しごとの pin を 2 本ずつ足した(162 本が通過)。雛形の括弧書きを「root に `go.work` がなければ、Go は root から入れ子の module に入らない」に直した(consult の任意の指摘)。変異 3 つがそれぞれ落ちることを確かめた。機械的な確かめは 5 項目とも clean
  - cycle 2 の self-review 3 回目(ea7a6a5b: LOW 3、Merge yes。triage の 2 件は解消、F8 も読んで R2-1〜R2-4 の解消を確認)
  - 直し 246c7432(inline、3 ファイルの数行): R3-3。雛形の「呼び手が」を「環境の `GOFLAGS` か `go env -w` で timeout が決めてあれば」にし、テストの定数と崩れていたコメントを直した。検証: `gofmt`・`go vet` と雛形のテストが通過。HEAD の古い文を `-overlay` で差し込むとテストが落ちることを確かめた。機械的な確かめ: unnecessary changes clean / typos and copy-paste errors clean / debug leftovers clean / secrets clean / files outside the slice clean
  - R3-1(`prompts_verification_test.go` が 890 行になった)と R3-2(plan の雛形の理由の文を shell の pin が固定しない)は、sync-docs で tech-debt の R2-5 の行に足す。self-review はここで止め、最終の diff は verify と cross-review が見る
  - verify の回し直し(bbb02949: pass、LOW 1)。現セッションから昇格した leader は skill だけを読むが、skill の「Leader 運用 2 経路」の手順 3 には Go の形も「役割」の箇条への参照もない。triage の 1 と 2 の「leader が書くコマンド」に当たるので直す
  - 直し F12 d721ed7d(implementer): 手順 3 の括弧に「Go なら「役割」の節の implementer の項の形で、`GOFLAGS` の timeout と、入れ子の module の `(cd <module> && …)` の包みもそのまま使う」を足した(skill の 4 つの写し)。`TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck` に手順 3 の照合を足し、句を消すと落ちることを確かめた。機械的な確かめは 5 項目とも clean
- [ ] PR created
