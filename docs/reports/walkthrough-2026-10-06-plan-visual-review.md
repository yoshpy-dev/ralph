# Walkthrough: plan-visual-review

- Date: 2026-10-06
- Plan: docs/plans/archive/2026-10-05-plan-visual-review.md(この PR の最後のコミットで archive に移す)
- Branch: feat/plan-visual-review(base main d7877756)
- Diff: 60 files、+4,113 / -63。plan・報告・insight・ミラー(`.agents/`、`templates/`)を除くと 22 files、+1,379 / -26。うち新しいスクリプトとテストが 3 files で +953

## 何を変えたか

`/plan` が承認を求める前に、実装する予定の構成と流れを図にした HTML ページ(図解ページ)を作り、ブラウザで見せるようにした。そのうえで Approve / Needs changes を聞く承認ゲートを新しく置いた。図解ページはエージェントが inline の SVG で書く。外部のライブラリや CDN は使わないので、オフラインでも開ける。目的は記録ではなく、人が実装の中身をつかむことなので、ページはコミットしない(`.harness/state/plan-visual/` に置く)。

承認したときには plan の digest を記録する。`/implement` は起動のたびに digest を計算し直し、記録と食い違えば、図解と承認をやり直すかどうかを聞く。`/pr` は図解ページの全体図を PNG にし、`gh pr create --attach` で PR 本文に載せる。

## 読む順

1. `bd6b6333` `scripts/plan-visual.sh`(open・shot・digest)
   - `open` は絶対パスを必ず出力し、開けない環境でも exit 0 で終わる。`shot` はヘッドレス Chrome / Chromium で PNG を撮る。ブラウザがなければ exit 2 で終わり、呼び出し側はそれを見て自己チェックを省く。`digest` は承認の対象になる部分の sha256 を計算する。
   - 配布物の必須一覧(`check-template.sh` の root と template、`test-check-template.sh` の GOLDEN_ENTRIES、`embed_test.go`)と、shellcheck の対象一覧に登録した。
2. `273a30ea` 図解の手引き `.claude/skills/plan/diagrams.md` とページの雛形 `visual-template.html`
   - 粒度は「全体図 1 枚 + 承認者が答えを知りたい問いごとの詳細図」。詳細図の種類は変更の種類で決め、各図に「What to check:」を 1 行添える。詳細図が 4 枚を超えたら、plan の分割を勧める。
   - 図解を省けるのは、ドキュメントだけの変更、1 ファイル(とそのミラー)だけの変更、機械的な改名や値の変更で、しかもやり取り・状態・データ形式・モジュール境界のどれも変えないとき。
   - 参照先の skill(nntto/skills の explanatory-diagrams)にはライセンスファイルがないので、考え方だけを借り、文章は書き起こした。リンクは残している。
3. `cbfbdceb` `/plan` の手順
   - step 9 と 10 をまとめ、空いた step 10 に図解ページと自己チェックを、新しい step 12 に承認ゲートを置いた。Codex advisory は step 11 のまま残した。`/cross-review` の SKILL.md とテスト 2 本が「`/plan` step 11.c」を参照しているため。
   - plan のテンプレート 6 か所に `- Approved:` の行と `## Visual review` 節を足し、`tests/test-new-feature-plan.sh` を新しく作った。
4. `64458ca2` `/implement` の step 4 に、承認と digest の確認を足した。step 番号は変えていない。
5. `84ad2d46` `/pr` の step 5.a〜5.c と、PR テンプレートの「全体図」節。step 番号は変えていない。
6. 実装後のパイプラインで直したもの
   - `c4a66c0d` self-review の 5 件。SVG の文字で `<slug>` をエスケープすること(エスケープしないとラベルが黙って消える)、`shot` の出力名の検査、文字サイズの目安の整合など。
   - `b0ea4a23` verify の 2 件。V-1: AC に印を付けるだけで digest が変わっていたので、行頭の `- [x]` は `- [ ]` とみなすようにした。アーカイブ済みの plan には `- [x]` が 1,183 か所あり、印を付ける慣習を前提にした。V-2: `docs/plans/templates/feature-plan.md` は upgrade で書き換わらない seed なので、行や節がない plan には `/plan` が足す。
   - `de99dd6c` cross-review の 1 件。`/pr` 5.c で既存の PR を探すとき、`gh pr view` は open の PR がないと同じブランチ名の merged / closed の PR を返す。そこで `gh pr list --head <branch> --base <base> --state open` に変えた。
   - `a09c057f` cycle 2 の self-review の 2 件(`<base>` の形の説明、plan の AC9 の記述の追随)。
   - `7231c44f`・`3caa56e4` cycle 2 の cross-review の 1 件と、追加パスの self-review の F-8。`gh pr list --head` は owner を区別しないので、`headRepositoryOwner` を origin の owner と照らす。owner は `gh repo view` ではなく origin の URL から `sed` で取る(SSH の別名ホスト `github.com.emu` を `gh` が解決できないため)。メンテナの判断で、パイプラインの上限を 2 から 3 に上げて直した。
   - `594354a9` 追加パスの cross-review の 1 件。cycle 2 の cross-review の insight event が `cycle: 1` で記録され、`ralph insights` で cycle 1 の `action_required` を上書きしていた。パイプラインが書いた記録の値だけなので、メンテナの判断でパイプラインは回さずに直した。
7. `804c5d84`・`a09c057f` plan の再承認
   - plan の本文を直すと digest が変わり、`/implement` が再承認を求める。この PR の中でその仕組みを 2 回実際に使った。最初の承認は `9c20a2da6606`、いまの記録は `4590e050b18a`。

## 変えていないもの

- `/spec` の流れ(導入先は `/plan` だけ、とユーザーが決めた)
- `/implement` 以降のパイプラインの順序(`check-pipeline-sync.sh` は対象が変わらず pass)
- 実装の途中で図を更新すること、Codex advisory に図を見せること(どちらも plan の Non-goals)
- org runtime。`/plan` を呼ぶ経路がない

## 下流への届き方

| 所有権 | ファイル | upgrade での扱い |
|---|---|---|
| core | `scripts/plan-visual.sh`、`.claude/skills/plan/{diagrams.md,visual-template.html,template.md,SKILL.md}`、`.agents/skills/` の同じもの、`implement`・`pr` の SKILL.md、rules | 作られる、または置き換わる |
| block | `AGENTS.md` | 管理ブロックの Primary loop が更新される |
| seed | `docs/plans/templates/feature-plan.md`、`docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md` | 本文は変わらず、advisory が出る。古い `feature-plan.md` のままでも、`/plan` が足りない行と節を足す |

verifier が、base のビルドで `ralph init` したプロジェクトにこのブランチのビルドで `ralph upgrade --yes` をかけて確かめた。

## 確かめたこと

- `tests/test-plan-visual.sh` 103 件、`tests/test-new-feature-plan.sh` 21 件。`./scripts/run-test.sh` では shell 35 ファイルの 1,648 件が PASS、FAIL 0。`go test ./...` は 8 パッケージ ok(cycle 1 と cycle 2 の両方)
- digest は macOS の awk に加え、Docker の mawk・gawk・busybox でも同じ値になった
- この Mac の Chrome で `shot --fragment overview` を実行し、全体図だけが写ることを確かめた
- `check-skill-sync.sh`、`check-sync.sh`、`check-template-purity.sh`、`check-pipeline-sync.sh`、`run-static-verify.sh` がすべて exit 0

## 確かめていないこと

- Codex CLI が PNG を読めるか(読めなければ、自己チェックを省いたと書く手順になっている)
- Linux の `xdg-open` がブラウザを閉じるまで戻らないことがあるか(手順では、Linux ではバックグラウンドで開く)
- `gh pr create --attach` の EMU アカウント・private repo での挙動。この repo では、この PR の作成で確かめる
