# Walkthrough: plan-visual-followups

- Date: 2026-10-06
- Plan: docs/plans/archive/2026-10-06-plan-visual-followups.md(この PR の最後のコミットで archive に移す)
- Branch: fix/plan-visual-followups(base main f9baf6a7)
- Diff(HEAD 4767c324、この walkthrough を除く): 44 files、+1,514 / -96。plan・報告・insight・ミラー(`.agents/`、`templates/`)を除くと 15 files、+558 / -31

## 何を変えたか

PR #203 で残った 4 件を直した。許可リストに `sed` と `git remote get-url` を足した。パイプラインの記録だけを直す修正は全工程を回し直さない、という例外を規則に書いた。insight event の cycle を 1 か所で決めるようにした。`/pr` 5.c の PR の探し方の端のケースを塞いだ。2 件目は、#203 でメンテナの判断で行った扱い(insight event の値だけを直して回し直さなかった)を、規則として認めるものである。

## 読む順

1. `5160f23b` 許可リスト。3 つの settings ファイルに同じ 2 行を足した。`templates/base/.ralph/core/settings.ralph.json` は template の settings と byte 単位で同じでなければならない(`internal/upgrade/snapshot_test.go`)
2. `ee0cb3f7` 記録だけの修正の例外。`post-implementation-pipeline.md` の節と、`/cross-review` の step 8(4 つの選択肢リストすべての 2 番目に "Fix records only")と step 9。cap に届いていても使える。`ralph-workflow.md` には、追跡ファイルを Edit / Write で書き換えるという一文を足した
3. `00dfbea3` `insights-append.sh --cycle auto`。解決の規則は `/cross-review` の step 1 と同じで、状態ファイルが欠けている・壊れている・`plan_path` が合わないときは 1 になり、append は落ちない。4 つの skill は `--cycle auto` を渡すだけ
4. `d7d31480` `/pr` 5.c。owner を取る `sed` が末尾の `/` を落とす。jq は大文字小文字を区別せず、null の owner でも落ちない。作り直す前に base を限らない 2 回目の検索をする
5. 実装後のパイプラインで直したもの
   - `c24eab98` self-review の F-1(修正のコミットが triage レポートを書き換えられる穴。triage レポートを条件 2 の対象から外し、確認結果は別の追記だけのコミットで書く)、F-3・F-5・F-6
   - `3a426bb8`・`38f6931a` R-1〜R-3。「追加した行だけ」の確認は途中への挿入を見分けないので、前の内容が新しい内容の先頭とバイト単位で一致することで確かめる。前のファイルが改行で終わることも確かめ、一時ファイルは `mktemp -d` の下に書いて消す
   - `25c45213` verify の V-1(triage レポートを記録のコミットで作る場合を `git cat-file -e` で弾く)、V-4(cap に届いたときの選択肢の番号)、`cycle-count.json` がないときのテスト
   - `e23c8047` test のレポートが挙げたテストの穴 G1・G2
   - `93ac7252` `/sync-docs`(V-3・V-5・V-6)

## 変えていないもの

- パイプラインの順序(`check-pipeline-sync.sh` は対象が変わらず pass)
- 例外の対象の外にある修正の扱い(コード・スクリプト・skill・rules・docs・plan を変えたら、これまでどおり全工程を回し直す)
- `ralph insights` の集計の仕組み。cap を上げた追加の 1 周は cycle を上げないので、同じ cycle の記録が 2 つになり、後の方が残る

## 下流への届き方

| 所有権 | ファイル | upgrade での扱い |
|---|---|---|
| core | skill(`/self-review`・`/verify`・`/test`・`/cross-review`・`/pr`)、rules、`scripts/insights-append.sh` | 置き換わる |
| settings | `.claude/settings.json` | 3-way merge で 2 行が足される |
| seed | `docs/quality/definition-of-done.md`、`docs/insights/README.md` | 本文は変わらず、advisory が出る |

## 確かめたこと

- `./scripts/run-test.sh` で shell 37 ファイルの 1,694 件が PASS、`go test ./...` は 8 パッケージ ok
- 新しいテストは ubuntu:24.04(dash、mawk、GNU sed と coreutils)でも通った
- 規則の確認のコードブロックを rules から取り出し、macOS(sh・bash・zsh)と ubuntu(dash・bash)で 5 ケースを実行した。追記 0、挿入 1、書き換え 1、改行のない行の延長 1、記録のコミットでの新規作成 128
- mutation で、`--cycle auto` を外すとテストが落ちること、`ascii_downcase` を外すとテストが落ちることを確かめた

## 確かめていないこと

- `/pr` 5.c の非ゼロ終了の経路を、実際の GitHub の応答で通すこと
- 例外の経路を実際の cross-review で使うこと(この PR では使っていない)
