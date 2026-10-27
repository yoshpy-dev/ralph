# verify-slices-lighter-self-review

- Status: Approved
- Approved: 2026-10-27 sha256:bd5c0d0ce1bb
- Owner: Claude Code
- Date: 2026-10-27
- Related request: 標準フローのスライスの検証を計画に書いて絞り、/self-review を軽くする(issue #261。2026-10-27 にユーザーと決めた方針: 「計画に検証コマンド」、「/self-review は軽くして残す」、implementer が渡す前に機械的な項目を確かめる)
- Related issue: #261
- Type: refactor
- Branch: refactor/verify-slices-lighter-self-review

## Objective

標準フローの検証の重複を減らす。今は `/implement` の手順 12 がフルの `./scripts/run-verify.sh` を打ち、その直後にパイプラインの `/verify` と `/test` が同じ検査を走らせる。スライスの検証に何を渡すかも決まっておらず、フルの検査を渡すとスライスごとに十数分かかる(このリポジトリの `run-test.sh` は多くが 640〜1,090 秒)。

この計画では、計画の雛形のスライスごとに、触ったパッケージだけの検証コマンドを書くようにし、`/implement` はそれを implementer に渡す。フルの検査はパイプラインの `/verify` と `/test` に任せる。implementer は報告の前に機械的な項目(不要な変更、誤字、デバッグコード、秘密情報、handoff の外のファイル)を確かめ、`/self-review` はそれ以外の 6 項目に絞る。

## Scope

- S1(計画の雛形と /plan): `docs/plans/templates/feature-plan.md` と `.claude/skills/plan/template.md` の `## Implementation outline` に、スライスごとの検証コマンドの書き方と、`RALPH_VERIFY_SCOPE=changed` で絞れない注意を足す。`/plan` の手順 7 に、スライスごとの検証コマンドを書くことを足す
- S2(/implement と規則): `/implement` の手順 6 で handoff が計画のスライスの検証コマンドを写す。検証コマンドのない計画(この変更より前に承認された計画)では、orchestrator がスライスの触るファイルを覆う最も狭いコマンドを選び、`## Progress checklist` に書いてから渡す(D4)。手順 7 の inline のスライスも同じコマンドを打ち、コミットの前に implementer と同じ機械的な確かめをして、結果を `## Progress checklist` に書く。手順 12 のフルの `run-verify.sh` を外し、Completion gate の 1 行目を「各スライスの検証が通った。フルの検査は直後の `/verify` と `/test`」にする。`.claude/rules/ralph/git-commit-strategy.md` の Validation Gate の手順 2、`.claude/rules/ralph/model-routing.md` の handoff の表と報告の契約、`.claude/rules/ralph/subagent-policy.md` の implementer の報告を合わせる
- S3(implementer): `.claude/agents/implementer.md` と `.codex/agents/implementer.toml` に、コミットの前の機械的な確かめ(不要な変更、誤字、デバッグコード、秘密情報、handoff の外のファイル)と、その結果を報告に書くことを足す
- S4(/self-review): `.claude/skills/self-review/SKILL.md` の観点を 6 つ(命名、読みやすさ、null safety と防御、例外処理、security、保守性)にする。外した 4 つは実装の側(implementer か inline の orchestrator)が見ること、秘密情報は `/verify` の branch secret scan と CI の secret scan でも見ることを書く。スクリプト名は書かない(D5)。frontmatter の description、`template.md` の Area の値、`.claude/agents/reviewer.md` と `.codex/agents/reviewer.toml` の観点を合わせる。`tests/test-self-review-scope.sh` に 6 項目の検査を足すのもこのスライスで行う
- 写し: `.agents/skills/`(`scripts/sync-skills.sh` で作る)、`templates/base/` の同名のファイル(`templates/base/.agents/skills/` も同じスクリプトで作る)
- テスト: `tests/test-self-review-scope.sh` の追加は S4 で行う。`tests/test-slice-verification.sh` を足し、雛形の注意、`/plan` 手順 7、`/implement` の手順 6(検証コマンドのない計画の扱い)・手順 7(inline の機械的な確かめ)・手順 12・Completion gate、Validation Gate、implementer の定義の機械的な確かめの文を固める

## Non-goals

- org の座席の検証と self-check(#262 で扱う。この計画の雛形の欄を #262 が使う)
- `scripts/run-verify.sh` などのスクリプトの振る舞いの変更
- パイプラインの順番(`/self-review → /verify → /test → /sync-docs → /cross-review → /pr`)の変更
- `.codex/AGENTS.override.md` の「`run-verify.sh` を迂回しない」の文(`/verify` と `/test` は包みのスクリプト経由で `run-verify.sh` を打つので、迂回にならない)
- 過去の report と archive の計画の書き換え

## Assumptions

- 雛形は言語に依存しないので、検証コマンドの例は Go を 1 つ挙げ、ほかの言語は言語パックのパッケージ単位の検査を使うと書く
- 計画の digest は雛形を読まないので、雛形を変えても既存の計画の承認は変わらない
- `templates/base/.claude/rules/ralph/model-routing.md` は `scripts/check-sync.sh` の KNOWN_DIFFS にあるので、root と別々に直す

## Affected areas

- `docs/plans/templates/feature-plan.md`、`.claude/skills/plan/template.md`、`.claude/skills/plan/SKILL.md`
- `.claude/skills/implement/SKILL.md`
- `.claude/rules/ralph/git-commit-strategy.md`、`.claude/rules/ralph/model-routing.md`、`.claude/rules/ralph/subagent-policy.md`
- `.claude/agents/implementer.md`、`.codex/agents/implementer.toml`
- `.claude/skills/self-review/SKILL.md`、`.claude/skills/self-review/template.md`、`.claude/agents/reviewer.md`、`.codex/agents/reviewer.toml`
- 写し: `.agents/skills/{plan,implement,self-review}/`、`templates/base/` の上の各ファイル
- テスト: `tests/test-self-review-scope.sh`、`tests/test-slice-verification.sh`(新規)

## Visual review

- Page: `.harness/state/plan-visual/verify-slices-lighter-self-review.html`(やさしい図 2 つ: `overview`、`easy-review`。詳しい図 2 つ: `detail-overview`、`detail-order`)
- Lint: `./scripts/plan-visual.sh lint` が終了コード 0
- Self-check: 全体のページと 4 つの図を `shot` で撮って読み、重なり・切れ・線の通り抜けがないことを確かめた

## Design decisions

- D1: スライスの検証コマンドは、新しい節を作らずに `## Implementation outline` の各項目に書く。項目がスライスに当たるので、handoff が写す元が 1 か所になる
- D2: `/implement` の手順 12 は消さずに、「各スライスの検証の結果を確かめ、フルの検査はパイプラインに任せる」に書き換える。手順の番号を変えると、ほかの文書の「手順 13」の参照が崩れるため
- D3: `/self-review` の null safety は、issue のとおり 6 項目の 1 つとして残す(例外処理とは別の行)
- D4(Codex の計画助言の 2 件目): 検証コマンドのない承認済みの計画は、再承認を求めずに進める。orchestrator がスライスの触るファイルを覆う最も狭いコマンドを選び、`## Progress checklist`(digest が読まない節)に書いてから handoff に入れる。implementer の「handoff の欄が欠けたら止まる」は handoff の契約で、計画の欄ではないので、止まらない
- D5(Codex の計画助言の 3 件目): `/self-review` の本文に `run-static-verify.sh` などのスクリプト名を書かない。`tests/test-self-review-scope.sh` は self-review の本文にスクリプト名があることを禁じていて、このガードを緩めずに済む。秘密情報の確かめの場所は「`/verify` の branch secret scan と CI の secret scan」と書く
- D6(Codex の計画助言の 1 件目): 機械的な確かめは implementer だけでなく inline のスライスにも課す。どちらの経路でも、`/self-review` から外した 4 項目を誰かが見る

Critical forks: None(方針はユーザーと issue #261 で決めた。D1〜D3 は戻しやすい書き方の選択、D4〜D6 は Codex の計画助言を受けた直し)

## Acceptance criteria

- [ ] AC1: 計画の雛形(2 つとその写し)の `## Implementation outline` に、スライスごとの検証コマンドの書き方と `RALPH_VERIFY_SCOPE=changed` の注意がある
- [ ] AC2: `/plan` の手順 7 が、スライスごとの検証コマンドを書くことを求めている
- [ ] AC3: `/implement` の handoff と inline のスライスが計画の検証コマンドを使い、手順 12 と Completion gate にフルの `run-verify.sh` がない。検証コマンドのない計画の扱い(D4)と、inline のスライスの機械的な確かめ(D6)が書かれている
- [ ] AC4: Validation Gate、model-routing の handoff と報告の契約、subagent-policy が新しい形に合っている
- [ ] AC5: implementer(Claude Code と Codex)の定義に、機械的な確かめと、その結果を報告に書くことがある
- [ ] AC6: `/self-review` の観点が 6 項目で、外した 4 項目は implementer の確かめにある。reviewer の定義と report の Area も合っている
- [ ] AC7: `scripts/check-skill-sync.sh`、`scripts/check-sync.sh`、`tests/test-agent-models.sh`、`tests/test-self-review-scope.sh`、`tests/test-new-feature-plan.sh`、`tests/test-slice-verification.sh` が通る

## Implementation outline

1. S1: 計画の雛形と `/plan` の手順 7(root と `templates/base/`、`.agents/` の写し) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && sh tests/test-new-feature-plan.sh`
2. S2: `/implement`、Validation Gate、model-routing、subagent-policy(root と `templates/base/`、写し) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && sh tests/test-agent-models.sh`
3. S3: implementer の定義(Claude Code と Codex、`templates/base/`) — verify: `./scripts/check-sync.sh && sh tests/test-agent-models.sh`
4. S4: `/self-review`、report の雛形、reviewer の定義(root と `templates/base/`、写し)、`tests/test-self-review-scope.sh` の検査の追加 — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && sh tests/test-self-review-scope.sh`
5. S5: `tests/test-slice-verification.sh` の新規 — verify: `sh tests/test-slice-verification.sh && shellcheck tests/test-slice-verification.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(shellcheck が新しいテストを見る)
- Spec compliance criteria to confirm: AC1〜AC7。issue #261 の受け入れ条件 5 つ
- Documentation drift to check: 「`run-verify.sh` (or equivalent)」をスライスの検証として書いている文書が残っていないか(`grep -rn "run-verify.sh\` (or equivalent)"`)。`/self-review` の観点を 10 個のまま書いている文書が残っていないか(`grep -rn "Debug code\|debug code"` を skills・agents・rules で)
- Evidence to capture: `scripts/check-skill-sync.sh` と `scripts/check-sync.sh` の出力

## Test plan

- Unit tests: なし(Go のコードは変えない)
- Integration tests: `tests/test-slice-verification.sh`(新規)、`tests/test-self-review-scope.sh`(検査を足す)
- Regression tests: `tests/test-new-feature-plan.sh`、`tests/test-agent-models.sh`、`tests/test-check-skill-sync.sh`、`tests/test-sync-skills.sh`、`./scripts/run-test.sh` の全体
- Edge cases: `templates/base/` 側だけに残った古い文(check-sync が拾う)。`.agents/` の写しの作り直し忘れ(check-skill-sync が拾う)
- Evidence to capture: 各テストの PASS の数

## Risks and mitigations

- R1: 雛形の例のコマンドが Go に寄り、ほかの言語の project で誤解される → 例は Go の 1 つにとどめ、「言語パックのパッケージ単位の検査を使う」と書く
- R2: 手順 12 を外すと、`/implement` の直後にフルの検査がない時間ができる → 直後に `/verify` と `/test` が必ず走り、PR の前の pre-check がその verdict を読むので、PR までにフルの検査は必ず通る
- R3: `/self-review` から秘密情報を外すと見落としが増える → implementer の確かめ、`run-static-verify.sh` の branch secret scan、CI の secret scan の 3 か所が残る

## Rollout or rollback notes

文書と定義だけの変更。戻すときはこの PR を revert する。scaffold 先には次の release の `ralph upgrade` で届く(skills と agents は core。雛形の `feature-plan.md` は seed なので、upgrade は advisory を出すだけ)。

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(Codex の計画助言 3 件(MEDIUM)を D4〜D6 として反映)
- [x] Plan approved(自動承認。2026-10-11 のユーザー指示「作成したプランは自動承認します」と、2026-10-27 の「最後まで止まらずに」による)
- [x] Branch created
- [x] Implementation started(5 スライスをすべて implementer に委譲)
  - S1 168c3950: 雛形の例のコマンドを `test -z "$(gofmt -l …)"` にした(`gofmt -l` は未整形のファイルを一覧しても終了コード 0 のため)
  - S2 dd7d4070: 委譲したスライスの確認項目にも「self-check の結果がある」を足した
  - S3 f1c68ebd: self-review から外れたのは 4 項目で、5 つ目(handoff の外のファイル)はもともと self-review の観点でないと書き分けた
  - S4 d501efad: scope テストの照合を太字の見出しにした(大文字小文字を区別しない grep が description にも当たるため)
  - S5 c77b12c9: 84 件。subagent-policy の検査も足し、変異 5 種で落ちることを確かめた
  - `ralph-workflow.md` の「成功を言う前に run-verify.sh か同等の検査」は、`/verify` と `/test` がそれに当たるので残した
- [x] Review artifact created(self-review 90f4ecca: MEDIUM 1・LOW 6、Merge yes)
  - 所見の直し cb094ae8(implementer): 5 項目の正本を model-routing に置き各所を同じ語にした、found and not fixed は次のスライスの前に直させる、inline の順序(stage のあとに確かめる)、Completion gate に /verify と /test、テストの穴をふさいだ
  - inline の追加(このコミット): 範囲内で消せない秘密情報はコミットせずに止める、を implementer の定義に足し、scope テストで固めた。検証: check-sync PASS、scope 198/198、agent-models 47/47、shellcheck と toml の読み込み OK。機械的な確かめ: unnecessary changes clean / typos and copy-paste errors clean / debug leftovers clean / secrets clean / files outside the slice clean
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
