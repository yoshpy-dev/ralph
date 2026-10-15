# sync-docs report: plan-visual-easy

## Cycle 1

- Date: 2026-10-15
- Plan: `docs/plans/active/2026-10-15-plan-visual-easy.md`(承認済み。digest は `819a5be1cf1f` で、編集の前後とも `- Approved:` の行と一致)
- Pipeline cycle: 1。差分は base `9cbf7080`(origin/main)から branch HEAD `2dfa4910`(feat/plan-visual-easy)まで。実装の commit は S1 `821a3c69`、S2 `5b056944`、S3 `138481e0`、self-review の直し `0067b837`
- 先行 report(それぞれを追加した commit。各 report 内の `HEAD:` の欄は、その report を commit する前の値なので使っていない):
  `docs/reports/self-review-2026-10-15-plan-visual-easy.md`(`45efffb8`。Merge yes、MEDIUM 1・LOW 5。直しは `0067b837`)、
  `docs/reports/verify-2026-10-15-plan-visual-easy.md`(`c84f2d2b`。pass、D-1 と情報 I-1〜I-4)、
  `docs/reports/test-2026-10-15-plan-visual-easy.md`(`f1cb06b2`。pass、生き残った mutation 9 個)

## Summary

古くなっていたのは `docs/tech-debt/README.md` の 2 か所だった。

1. leader がページを確かめる手順の行(215 行目)。leader は撮る前に `scripts/plan-visual.sh lint` を打つようになったが(`internal/org/prompts/leader.md` の step 2)、行の説明・Impact・Trigger のどれにも lint がなかった(verify D-1)。lint はブラウザなしで動くので、撮れない leader のページも決まりの形は機械で確かめられる。Impact の「誰も点検していないページ」は、今は強すぎる書き方だった。
2. /test が出した lint のテストの穴。登録簿に行がなかったので、1 行足した(252 行目)。

そのほかの文書(AGENTS.md、README.md、`docs/architecture/repo-map.md`、`docs/quality/`、`docs/recipes/`、`.claude/rules/ralph/`、`.ralph/core/`、`docs/specs/`、`/director` と `/org` の skill)に、この PR で間違いになる記述はなかった。AGENTS.md の helper の一覧、repo-map の script の一覧、README の `/plan` の段落は S3 で直してあり、いまの差分と合っている。plan は触っていない(後述)。

差分の大きさ(`git diff origin/main...HEAD --shortstat`、この report と tech-debt の編集を足す前): 37 files changed, 3163 insertions(+), 608 deletions(-)。skill と雛形が root・`.agents`・`templates/base` の 4 面にあるので、数は見た目より大きい。写しを除いた実体のあるコードの変更は、`scripts/plan-visual.sh` が +312/-3(`templates/base/` の写しも同じ)、`tests/test-plan-visual.sh` が +293/-1、`internal/org/prompts/leader.md` が +6、`internal/org/prompts_test.go` が +3。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(leader のページの行、215 行目) | verify D-1 の直し。列は 5 のまま。(1) 説明: leader の手順に「`scripts/plan-visual.sh lint` を打ち、出た指摘を終了コード 0 になるまで直す(ブラウザは要らない。`--help` に `lint` がなければ飛ばし、QUESTION に書く)」を撮る前の段として足し、「まだ live で走らせていない」の対象に lint の段を含めた。(2) Impact: 撮れない・読めない leader のページも、`lint` があれば決まりの形(L1〜L6)は確かめられるが、配置・重なり・読みやすさは誰も見ない、と書き分けた。lint も打てず撮れも読めもしない leader だけが「誰も点検していないページを送る」ことになる。(3) Why deferred: plan-visual-easy が template に lint の段を足し、`prompts_test.go` で 3 文を固定したが、live の実行はない、と足した。(4) Trigger: 記録する項目を「ページを作ったか、lint を打てたか(または lint がなくて飛ばしたか)と終了コード 0 に届いたか、撮ったか、PNG を読んだか」にした。(5) Related: `leader.md` の注に step 2 の lint の文、`scripts/plan-visual.sh`(`lint`)、この plan(`docs/plans/active/` のパス)、verify report(D-1)を足した |
| `docs/tech-debt/README.md`(新しい行、252 行目) | /test の Test gaps と mutation の表を 1 行にした。(a) 生き残った 9 個(L1e、L2b、L2c、L4f、L5b、O6、O7、O9、O11)を、それぞれ何を壊したか・なぜテストの行が見分けないか・どの fixture か assertion で落ちるか、の順で。(b) `templates/base/scripts/plan-visual.sh` を suite が打たないこと(`check-sync.sh` が byte 同一を見る。/test が手で `lint` を打って同じ終了コードを確かめた)。(c) `/plan` step 10.c の古い helper の扱いと `/pr` step 4.a のページの形の見分けが skill の文だけで、テストが固定するのは見分けに使う `--help` の 1 行だけ。(d) `<details>` が fragment で開くのを Chrome でしか見ていないこと(plan の R3)。Trigger は `lint_page`・`cmd_lint`・`diagrams.md` の L1〜L6 を次に変えるとき、と `/plan` 10.c・`/pr` 4.a・`--help` の文を次に変えるとき。Related は plan(`docs/plans/active/`)、test report(Test gaps、Coverage、mutant の名前)、`tests/test-plan-visual.sh`、`scripts/plan-visual.sh`(`lint_page`、`cmd_lint`) |
| `docs/insights/events/2026-10-15-plan-visual-easy.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`) |
| `docs/reports/sync-docs-2026-10-15-plan-visual-easy.md` | この report |

2 つの行とも、コードは行番号ではなく関数・変数・節の名前(`lint_page`、`cmd_lint`、`SEEN`、`parse_attrs`、`/plan` step 10.c)で書き、新しい `file:line` の参照はない。plan のパスは `docs/plans/active/` で、`/pr` の `archive-plan.sh` が書き換える(早く `archive/` と書いていない)。

plan は変えていない。`## Progress checklist` に「Docs synced」にあたる項目がなく、Review・Verification・Test の 3 項目は `c93c16aa` と `2dfa4910` でチェック済みだった。「PR created」は `/pr` が付ける。verify I-2(「Review artifact created」が未チェック)は、そのチェックがすでに入っていて解消している。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `git grep -n plan-visual`(`docs/reports`、`docs/plans`、`docs/evidence`、`docs/insights`、`tests`、script 本体、`.agents`、`templates/base/.agents`、`templates/base/.claude` を除く) | 当たった先は、AGENTS.md:111(`open` / `shot` / `lint` / `digest`)、`docs/architecture/repo-map.md:62`(open/screenshot/lint/approval digest)と 73(`.harness/state/plan-visual/` の行)、README.md の 218・252・260 行、`docs/quality/quality-gates.md` の分割計画の承認の行、`docs/specs/2026-10-07-org-multi-org-director.md`、`/director`・`/org`・`/implement` の skill、`internal/org` の Go のコメントとテスト、`docs/tech-debt/README.md`。AGENTS.md・repo-map・README 218 は S3 で直してあり合っている。README の 252 行(分割計画の承認は `digest` の値)と 260 行(per-plan の leader が「図解ページを描く」)は、手順を並べておらず、新しい形でも正しい。`digest` の動きは変わっていない(AC8) |
| `.ralph/core/AGENTS.core.md` と `templates/base/AGENTS.md` | helper の一覧がなく、「shows a visual review page」の 1 句だけ。変更なし(S3 と同じ判断) |
| `/director` skill(`## 分割計画と機能ごとの org` の 2・`## 機能の計画の承認` の 1)と `/org` skill(承認の 2) | 分割計画のページは「`/plan` skill と同じ手順で」作るので、lint を含む手順がそのまま届く。leader が作ったページは `open` で開くだけで、lint の話は出ない。変更なし |
| `docs/recipes/codex-setup.md`(41・47 行)、`.claude/rules/ralph/ralph-workflow.md`(40 行)、`.claude/rules/ralph/subagent-policy.md`(74 行) | 「visual review page を見せる」「overview diagram を貼る」で、今の形でも正しい。変更なし |
| `docs/quality/`(definition-of-done、quality-gates) | `git grep -i -E 'overview\|図解\|visual review'` に当たりなし。quality-gates の分割計画の承認の行は `digest` の規則だけを指す。変更なし |
| `.claude/skills/pr/template.md` | `## 全体図` に overview の PNG を載せる。overview はかんたん版の図 1 になったが、貼る手順は `/pr` 4.a にあり、テンプレートの文面は変わらない |
| tech-debt の 251 行目(PR が S3 で足した `digest` と `a=b.md` の行) | 内容は self-review が scratch で確かめている。plan のパスは `active/`。変更なし |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。この sync で変えたのは tech-debt、report、insight だけで、template に写しのあるファイルは変えていない |
| harness 内部の整合(skill、hook の配線、rule、language pack、script の一覧) | この sync の差分は tech-debt、report、insight だけ。skill の追加・削除・改名、hook、rule、language pack、script の追加・削除はなく、AGENTS.md の repo map と README の loop は合っている |

## Found but left

- verify I-4(情報): `scripts/plan-visual.sh` のヘッダの L4 は「an `<svg>` with an aria-label」で、コードと `diagrams.md` は「空でない」。動きは変わらない。直すのは script のコメント(root と template の 2 か所)で、この時点でコードのファイルを変えると pipeline を最初から回し直すことになる。次にヘッダを直すときに「non-empty」を足す
- verify I-3(情報): L3 が見るのはかんたん版の図だけで、見出し(`<h1>` とリード)と「承認の前に」の欄は見ない。承認する人はこの 2 つも読む。plan が L3 を「かんたん版の図の中」と決めたとおりの範囲なので、登録簿には書いていない。広げるかは次の plan で決める。必要なら 1 行足せる
- verify I-1(情報): plan の `## Visual review` は「lint はまだない」のまま。直すと digest が変わるので触らない(`/pr` はこの節からページのパスだけを読む)。lint の結果は Progress の S1・S3 のメモにある
- self-review L2: `leader.md` は古い helper のとき lint を飛ばすと書き、`diagrams.md` は main のチェックアウトの helper を使うと書く。verify が、leader が読む `diagrams.md` は leader の worktree の同じコミットから来るので、2 つの指示が同時に届く組み合わせはない、と判断して残した。この sync も変えない
- verify の Not verified: Chrome でないブラウザでの `<details>` の展開と、headless の leader が lint・shot を実地で打てるか。後者は 215 行目の行の範囲で、今回その行に lint を足した

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-15-plan-visual-easy.md` | `819a5be1cf1f`(plan の `- Approved:` の行と一致。この sync は plan を触っていない) |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0) |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(tech-debt README plan references を含めてすべて OK)。編集のあとに実行 |
| tech-debt の行の形(scratch の `tdcheck.py`、`git show origin/main:docs/tech-debt/README.md` との行ごとの比較) | 今回の編集の 215 行と 252 行は、どちらも 5 列、列ごとのバッククォートと `~~` は偶数、`file:line` の参照なし、U+FFFD なし。比較で変わった行は 215(編集)、251(この PR が S3 で足した `digest` の行)、252(足した行) |

この commit のあとに `./scripts/secret-scan-branch.sh --strict` を流し、exit 0 なら push する(結果は返信に書く)。

## Not verified

- `./scripts/run-test.sh` と `tests/test-plan-visual.sh` は流していない(`/test` が確認済み)。252 行目の mutant の名前・fixture・「落ちることを scratch で確かめた」は test report の Test gaps からの転記で、この sync では再現していない
- (b)の「/test が template 側の写しに `lint` を打って同じ終了コードだった」も test report の記述による
- 215 行目に書いた leader の手順は `leader.md` の step 2 の文から書いた。実際の leader の実行では確かめていない(それがこの行の主題である)
- 「ほかの文書に間違いになる記述はない」は、上の grep と読んだ範囲の確認。ページの手順を別の言い方で書いた文を全文書で網羅したわけではない。未確認です
