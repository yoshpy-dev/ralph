# plan-visual-followups

- Status: Approved
- Approved: 2026-10-06 sha256:eb13fda57381
- Owner: Claude Code
- Date: 2026-10-06
- Related request: PR #203 の後続 4 件(2026-10-06 ユーザー依頼。「1. .claude/settings.json の許可リストに sed と git remote get-urlを入れてください。2. 次はこの件について、取り組んでください。3. 着手してください。」)
- Related issue: N/A
- Type: fix
- Branch: fix/plan-visual-followups

## Objective

PR #203(/plan の図解ページと承認ゲート)で残った 4 件を片付ける。

1. `/pr` 5.c の復旧の経路で許可を求められないように、許可リストに `sed` と `git remote get-url` を足す
2. パイプラインの記録(レポートと insight event)だけを直す修正は、全工程を回し直さずに記録して進める、という例外を規則に書く。#203 ではユーザーの判断でこの扱いをしたが、規則には書かれていなかった
3. `/self-review`・`/verify`・`/test`・`/cross-review` の insight event のコマンドに `--cycle` を足す。いまは cycle 2 以降の記録も既定の cycle 1 になり、`ralph insights` で前の cycle の verdict を上書きする
4. `/pr` 5.c の PR の探し方で、origin の URL の末尾が `/`、owner の大文字小文字の違い、別の base への open な PR、のときに取り違えたり矛盾したりしないようにする

## Scope

- 許可リストに `Bash(git remote get-url:*)`(`Bash(git config:*)` の後)と `Bash(sed:*)`(`Bash(sort:*)` の後)を足す。`.claude/settings.json`、`templates/base/.claude/settings.json`、`templates/base/.ralph/core/settings.ralph.json` の 3 か所を同じ内容にする
- `.claude/rules/ralph/post-implementation-pipeline.md`(root と template)に例外の節を足す。対象は次の 3 つをすべて満たす修正。(a) 指摘が記録そのもの(レポートの記述、insight event の値)を対象にしている、(b) 修正のコミットが、その task の slug を名前に含む `docs/reports/` 直下のレポートと `docs/insights/events/` のイベントだけを変える(`git diff --name-only` で確かめる。`docs/reports/templates/` は `/cross-review` などが読む入力なので対象外)、(c) self-review・verify・test レポートの verdict の行を変えない(`/pr` の前提条件がその verdict を読むため)。確かめた結果(修正コミットの `git show --stat` と verdict の行が変わっていないこと)を triage レポートに、例外を使ったことを PR 本文に書き、`cycle-count.json` は上げずに進む
- `.claude/skills/cross-review/SKILL.md` の step 8(Case A・Case B)に、上の例外にあたる指摘だけのときの選択肢を足し、step 9 にその扱いを書く
- `scripts/insights-append.sh`(root と template)に `--cycle auto` を足す。`/cross-review` の step 1 と同じ規則で cycle を決める: `.harness/state/standard-pipeline/active-plan.json` と `cycle-count.json` がそろっていて、2 つの `plan_path` が一致すれば `cycle-count.json` の `cycle`、それ以外(どちらかがない、`plan_path` が違う、読めない)は 1。状態の置き場所は `--state-dir DIR` で差し替えられる(テスト用、既定は repo の `.harness/state/standard-pipeline`)
- 4 つの skill の insight event のコマンドに `--cycle auto` を足す(規則を skill に書き写さない)
- `.claude/rules/ralph/ralph-workflow.md`(root と template)に一文: 追跡ファイルは Edit / Write の道具で書き換え、`sed -i` やその場しのぎのスクリプトでは書き換えない。PostToolUse の hook(mojibake の検査など)は Edit / Write にしか掛からないため(`Bash(sed:*)` を許可したことへの手当て、consult の指摘)
- `.claude/skills/pr/SKILL.md` 5.c: owner を取る `sed` の先頭で末尾の `/` を落とす。jq は owner を小文字にそろえ、null の owner でも落ちない形(`.headRepositoryOwner.login // ""`)で比べる。「PR なし」と判断して作り直す前に、base を限らずに同じ head・同じ owner の open な PR を探し(`--json url,baseRefName,headRepositoryOwner`)、別の base への PR が見つかったら作り直さずに止めて報告する。gh が「PR がすでにある」と断ったときも止めて報告する
- テスト: `tests/test-insights-append.sh` に `--cycle auto` のケース(正常な cycle 2、`active-plan.json` がない、`plan_path` が違う、`cycle-count.json` が壊れている)を足す。`tests/test-skill-insight-cycle.sh`(新規、4 skill の insight event のコマンドに `--cycle auto` があること、4 面で)。`tests/test-pr-owner-lookup.sh`(新規、`/pr` の SKILL.md から `sed` と 2 つの jq の式を取り出して実行する。URL の 7 形で owner を、fixture で大文字の login・null の owner・別の owner・別の base の PR の扱いを確かめる。jq がなければ SKIP)
- ミラー: `.agents/skills/`(`scripts/sync-skills.sh`)と `templates/base/`

## Non-goals

- 例外の対象を広げること。plan、コード、スクリプト、skill、rules、docs を変える修正は、これまでどおり全工程を回し直す
- #203 の逸脱(594354a9)をさかのぼって回し直すこと。マージ済みで、今回の例外の条件にも当てはまる
- `sed` の許可を細かく絞ること(たとえば `sed -E` だけ)。ユーザーの指示どおり `Bash(sed:*)` にする。前置きの一致では `-i` を除けないので、絞っても効果が薄い
- `ralph insights` の集計の仕組みを変えること
- `ralph upgrade` の 3-way merge の仕組みを変えること(snapshot と template を同じ内容にするだけ)

## Assumptions

- `templates/base/.ralph/core/settings.ralph.json` は `templates/base/.claude/settings.json` と byte 単位で同じでなければならない(`internal/upgrade/snapshot_test.go:51`)。root と template の `.claude/settings.json` は `scripts/check-sync.sh` が同じであることを確かめる
- 下流では、`ralph upgrade` の 3-way merge が新しい許可の 2 行を足す(snapshot と template の差分として届く)。利用者が許可リストを変えていても、追加の行はぶつからない
- `scripts/insights-append.sh` は `--cycle N` を受け付け(既定は 1)、JSON の組み立てに jq を使う。`auto` の解決にも jq を使えるので、依存は増えない
- `gh pr list --jq` は `--arg` を受け付けないので、owner は jq の式の中に文字列として埋め込む
- 手元の確認(2026-10-06): 末尾の `/` を落とす `sed` は、`https://github.com/Yoshpy-Dev/ralph/` を含む 5 形で owner を返した。jq の `(.headRepositoryOwner.login // "" | ascii_downcase) == ("yoshpy-dev" | ascii_downcase)` は、大文字の login に一致し、null の owner でも落ちず、別の owner を外した
- jq は手元(`/opt/homebrew/bin/jq`)と CI(ubuntu-latest)にある

## Affected areas

- `.claude/settings.json`、`templates/base/.claude/settings.json`、`templates/base/.ralph/core/settings.ralph.json`
- `.claude/rules/ralph/post-implementation-pipeline.md`、`templates/base/.claude/rules/ralph/post-implementation-pipeline.md`
- `.claude/rules/ralph/ralph-workflow.md`、`templates/base/.claude/rules/ralph/ralph-workflow.md`
- `.claude/skills/{cross-review,self-review,verify,test,pr}/SKILL.md` と、`.agents/skills/`・`templates/base/.claude/skills/`・`templates/base/.agents/skills/` の同じファイル
- `scripts/insights-append.sh`、`templates/base/scripts/insights-append.sh`
- `tests/test-insights-append.sh`、`tests/test-skill-insight-cycle.sh`(新規)、`tests/test-pr-owner-lookup.sh`(新規)
- ドキュメント(`/sync-docs` で確かめる): `docs/quality/definition-of-done.md`、`README.md` の Operating loop、`AGENTS.md` の Primary loop は、パイプラインの順序を書いているが例外には触れていない。触れる必要があるかは `/sync-docs` が判断する

## Visual review

- 図解ページ: `.harness/state/plan-visual/plan-visual-followups.html`
- 自己チェック: ヘッドレス Chrome で全体・`#overview`・`#flow` を撮って確認。図 2 のラベルの重なり 1 か所を直した。consult と Codex plan advisory の指摘を反映したあと、図 1(S2 に `ralph-workflow.md`、S3 に `insights-append.sh`)と図 2 の注記を直して撮り直した

## Design decisions

- **2 番目の件の取り組み方: 例外として規則に書く(ユーザー確定、2026-10-06)**。採らなかった案: 例外を作らず、記録の値だけでも全工程を回すと明記する
- **例外の範囲は、その task のレポートと insight event に限る**。どちらもパイプラインが書く記録で、self-review・verify・test・sync-docs のどれも中身を検査しない。plan は承認の digest があり、docs は `/sync-docs` の対象なので入れない。`docs/reports/templates/` は入力なので除く(Codex plan advisory の指摘 1)。名前に slug を含むファイルに限ると、別の task のレポートも除ける
- **verdict の行は例外でも変えない(consult の指摘)**。`/pr` の前提条件 1〜3 は self-review・verify・test レポートの verdict を読むので、パスだけの条件では fail を pass に書き換える修正も通ってしまう。指摘が記録そのものを対象にしていることと、verdict の行が変わっていないことを条件に足し、確かめた結果を triage レポートに残す
- **例外は `/cross-review` の step 8 だけに置く**。self-review・verify・test はその回のレポートを自分で書く段階なので、例外は要らない
- **`Bash(sed:*)` の手当ては規則の一文にする**。許可そのものはユーザーの指示どおり広く入れ、`sed -i` で追跡ファイルを書くと hook を素通りする点は、Edit / Write を使うという規則で抑える
- **例外に当たるかは機械的に確かめる**: 修正のコミットの `git diff --name-only` が 2 つのパスの下だけであること。1 つでも外れたら全工程を回し直す
- **`--cycle` の値は `insights-append.sh --cycle auto` が 1 か所で決める(Codex plan advisory の指摘 3)**。`cycle-count.json` をそのまま読むと、`/cross-review` の fallback 規則(`active-plan.json` がない、`plan_path` が違うときは 1)とずれ、中断した別の task の状態が残った環境で cycle を誤る。規則を 4 つの skill に書き写すとずれやすいので、スクリプトに置いてテストで縛る。`/cross-review` の cap を上げた追加の 1 周は cycle を上げないので、同じ cycle の記録が 2 つになり、集計では後の方が残る。これは #203 で確かめた挙動で、今回は変えない
- **別の base への PR は、作り直す前の 2 回目の検索で見つける(Codex plan advisory の指摘 2)**。1 回目の検索は `--base <base>` で絞っているので、別の base への PR が見えない。作り直す前に base を限らずに探し、`baseRefName` で分岐する
- **`/pr` 5.c のテストは SKILL.md から式を取り出して実行する**。文書と挙動を同じテストで縛り、式を書き換えたら壊れるようにする
- Critical forks: None。残った分岐(テストの置き場所、許可の並び順、例外の文言)は 1 slice 以内でやり直せる

## Acceptance criteria

- [ ] AC1: 3 つの settings ファイルの `permissions.allow` に `Bash(git remote get-url:*)` と `Bash(sed:*)` が 1 回ずつあり、3 ファイルが同じ内容。`go test ./internal/upgrade/...` と `./scripts/check-sync.sh` が通る。JSON として読める
- [ ] AC2: `post-implementation-pipeline.md`(root と template、同じ内容)に例外の節がある。3 つの条件(指摘が記録そのものを対象にしている、その task の slug を名前に含む `docs/reports/` 直下のレポートと `docs/insights/events/` のイベントだけを変える(`docs/reports/templates/` は対象外)、self-review・verify・test レポートの verdict の行を変えない)、確かめ方(`git diff --name-only`、修正コミットの `git show --stat`、verdict の行の差分)、triage レポートと PR 本文への記録、`cycle-count.json` を上げないこと、1 つでも外れたら全工程を回し直すこと、が書いてある。`ralph-workflow.md`(root と template)に Edit / Write を使う一文がある
- [ ] AC3: `/cross-review` の step 8 の Case A と Case B(cap に届いていても届いていなくても)に、例外にあたる指摘だけのときの選択肢があり、step 9 にその扱い(修正、3 つの条件の確認、triage レポートへの記録、`cycle-count.json` は上げない、`/pr` へ)が書いてある。条件の文言は `post-implementation-pipeline.md` を指し、同じ内容を二重に書かない
- [ ] AC4: `insights-append.sh --cycle auto` が、`active-plan.json` と `cycle-count.json` がそろい `plan_path` が一致するときはその `cycle` を、それ以外(片方がない、`plan_path` が違う、JSON が壊れている)は 1 を書く。`tests/test-insights-append.sh` がこの 4 通りを `--state-dir` で確かめる。4 つの skill の insight event のコマンドに `--cycle auto` があり、`tests/test-skill-insight-cycle.sh` が 4 skill × 4 面で確かめ、`--cycle auto` を外すと落ちる。root と template の `insights-append.sh` は同じ内容
- [ ] AC5: `/pr` 5.c の `sed` が末尾の `/` を落とし、jq が owner を小文字にそろえて null でも落ちずに比べる。作り直す前に base を限らない 2 回目の検索をし、別の base への open な PR があれば作り直さずに止めて報告する、gh が「PR がすでにある」と断ったときも止めて報告する、と書いてある。`tests/test-pr-owner-lookup.sh` が SKILL.md から取り出した式で、URL 7 形(https の `.git` あり・なし・末尾 `/`、`ssh://`、ポートつき `ssh://`、`git@host:`、SSH の別名ホスト)と jq の fixture(大文字の login、null の owner、別の owner、別の base の PR)を確かめる
- [ ] AC6: `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`./scripts/check-pipeline-sync.sh`、`bash scripts/check-template-purity.sh`、`./scripts/run-verify.sh` が通る

## Implementation outline

1. S1: 許可リスト(AC1)
2. S2: パイプラインの例外(AC2、AC3)
3. S3: `insights-append.sh --cycle auto`、4 skill の `--cycle auto`、テスト(AC4)
4. S4: `/pr` 5.c の端のケースとテスト(AC5)
5. ドキュメントは `/sync-docs` で確かめる

## Verify plan

- Static analysis checks: shellcheck(新しいテスト 2 本)、`check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-template-purity.sh`、settings の JSON の読み込み
- Spec compliance criteria to confirm: AC1〜AC6。とくに AC2 と AC3 の例外の範囲が同じ書き方になっていること
- Documentation drift to check: パイプラインの順序を書いている文書(`check-pipeline-sync.sh` の対象)が例外と矛盾しないか
- Evidence to capture: verify レポート

## Test plan

- Unit tests: 新しいテスト 2 本
- Integration tests: `go test ./internal/upgrade/...`(snapshot と template の一致)、`go test ./internal/cli/...`(init と upgrade の fixture)
- Regression tests: `./scripts/run-test.sh` の全体
- Edge cases: URL 末尾の `/`、大文字の owner、null の owner、jq がない環境(SKIP になること)、`cycle-count.json` がないとき
- Evidence to capture: test レポート

## Risks and mitigations

- `Bash(sed:*)` は `sed -i` での書き換えも許してしまう → ユーザーの指示で入れる。Edit / Write の道具は別に許可を求めるので、ファイルの書き換えが黙って増えるのは `sed -i` を使う場合だけ。PR 本文に書く
- 例外が広く読まれ、本来は全工程を回すべき修正にも使われる → パスを 2 つに限り、`git diff --name-only` で機械的に確かめる手順にする
- 例外の修正でレポートの verdict を書き換えれば、`/pr` の前提条件をすり抜けられる → verdict の行を変えないことを条件にし、確かめた結果を triage レポートに残す
- `sed -i` で追跡ファイルを書くと、Edit / Write にしか掛からない PostToolUse の hook を素通りする → `ralph-workflow.md` に Edit / Write を使う一文を足す。機械的には止めない(PreToolUse の guard は今回の範囲外)
- テストが SKILL.md の書式に依存して壊れやすい → 式を取り出す目印(` sed -E '` と `--jq '`)を 1 か所に決め、見つからなければテストを FAIL にして気づけるようにする
- 下流の許可リストに 2 行が増える → upgrade の 3-way merge の差分として届く。利用者が消したければ消せる

## Rollout or rollback notes

- 設定と文書とテストだけの変更。戻すときは PR を revert する
- 下流には次のリリースの `ralph upgrade` で届く。settings は 3-way merge、skill と rules は core として置き換わる

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
  - S1 完了(5160f23b、inline): 3 つの settings ファイルに 2 行。3 ファイルは byte 同一、`go test ./internal/upgrade/... ./internal/cli/...` と check-sync が通過
  - S2 完了(ee0cb3f7): 例外の節、`/cross-review` の step 8・9、ralph-workflow.md の一文。対象のファイルは「slug を名前に含む」より厳しい「plan の slug で名付けたファイル」にした(似た slug の別 task を除くため。AC2 の範囲内)。cap に届いていても例外の選択肢は使える
  - S3 完了(00dfbea3): `insights-append.sh --cycle auto` と `--state-dir`、4 skill × 4 面、テスト(49 件・16 件)。既存の jq の式の行に SC2016 の disable を 1 行足した(jq の変数で誤検知)
  - メモ(範囲外): `scripts/verify.local.sh` の shellcheck の対象一覧に `scripts/insights-append.sh` が入っていない
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
