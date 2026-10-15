# Test report: plan-visual-easy

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-plan-visual-easy.md(パイプライン 1 回目、cycle 1)
- Tester: tester subagent (Claude)
- Scope: `git diff origin/main...HEAD`(base 9cbf7080、HEAD c93c16aa)。動きを確かめたのは、新しい `scripts/plan-visual.sh lint`、変わらないはずの `open`・`shot`・`digest`、雛形 4 面の `lint`、`internal/org/prompts_test.go` が固定する `leader.md` の 3 文。静的解析は /verify の受け持ちなので打っていない
- Evidence: `docs/evidence/plan-visual-lint-2026-10-15.txt`(代表的なページの `lint` の出力と mutation の一覧。コミットする)、`docs/evidence/test-2026-10-15-plan-visual-easy.log`(`run-test.sh` と `go test` の生の出力。git は無視する)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`: shell 41 本(範囲は full に戻った。`unclassified:.agents/skills/plan/visual-template.html`) | 4,217 | 4,216 | 0 | 1 | 30 分 40 秒(Go と合わせて。load average 7〜12) |
| `./scripts/run-test.sh`: Go `go test ./...`(go1.26.0) | 8 packages | 8 | 0 | 0 | cli 362.6 s、org 151.6 s、scaffold 0.8 s。ほかの 5 つは `(cached)` |
| `go test -count=1 ./internal/config/ ./internal/insights/ ./internal/upgrade/`(上で `(cached)` だった 3 つ) | 3 packages | 3 | 0 | 0 | 3.1 s |
| `bash tests/test-plan-visual.sh` | 201 | 201 | 0 | 0 | 15.3 s |
| `sh tests/test-plan-visual.sh`(macOS の `/bin/sh` は bash 3.2) | 201 | 201 | 0 | 0 | 約 15 s |
| `go test -count=1 ./internal/org/...`(go1.26.0) | 3 packages | 3 | 0 | 0 | org 145.9 s |
| `GOTOOLCHAIN=go1.25.8 go test -count=1 ./internal/org/...` | 3 packages | 3 | 0 | 0 | org 204.6 s |
| ubuntu:24.04 の mawk 1.3.4 で `sh tests/test-plan-visual.sh`(`--network none`) | 201 | 200 | 0 | 1 | — |
| ralph-guard-awk:test の GNU Awk 5.2.1 で同じ suite | 201 | 200 | 0 | 1 | — |
| origin/main の `tests/test-plan-visual.sh` を HEAD の helper で打つ(AC8) | 103 | 103 | 0 | 0 | — |
| `lint` の mutation(scratch の写しで 1 か所ずつ変えて suite を打つ) | 57 | 48 が落ちた | — | 9 が生き残った | 約 13 分(4 並列) |
| `go test -overlay` で `leader.md` を差し替える(変えない写し 1 + 変える 4) | 5 | 5 が予想どおり | — | — | 各 1 s 未満 |

- shell の数は、`run-test.sh` のログで suite ごとに `PASS`・`FAIL`・`SKIP` で始まる行を数えた(`PASS: 29` や `PASS: 47 / 47` のような集計の行は除いた)。`SKIP` の 1 件は `tests/test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合で、手元の git が 2.49.0 だから飛ばされる。この変更とは関係がない
- Docker の 2 回の実行の `SKIP` 1 件ずつは、`digest` の節の `shasum is not installed here` で、この変更より前からある。`lint` の節の 97 件はどちらの awk でもすべて通った
- `run-test.sh` で `(cached)` だった 5 つのうち、`internal/org/driver` と `internal/org/protocol` は上の `go test -count=1 ./internal/org/...` で、`config`・`insights`・`upgrade` は表の 3 行目で、キャッシュなしでも通った。8 packages すべてを go1.26.0 のキャッシュなしで 1 回ずつ打ったことになる
- `tests/test-plan-visual.sh` の 201 件のうち、この変更で増えたのは 98 件(`lint` の節 97 件と、`--help` に `  scripts/plan-visual.sh lint <html>` の行があることを固定する 1 件)。残りの 103 件は origin/main の suite と同じ

## Coverage

- Statement: shell を計る道具はない。Go は `leader.md` の文を足しただけで、`go test -cover` は打っていない(計測の対象のコードは変わっていない)
- Branch: `lint` の分岐は mutation で測った。57 個のうち 48 個で suite が落ちた。L1〜L6 の規則をそれぞれ丸ごと止める 6 個はすべて落ち、落ちた assertion の数は L1 11、L2 6、L3 10、L4 10、L5 6、L6 19 だった
- Function: `lint_page` の中の関数(`parse_attrs`・`tag_end`・`scan_text`・`widen`・`check_easy`・`finish`)は、それぞれを壊す mutation のどれかで suite が落ちた。生き残った 9 個は下の Test gaps に書いた
- Notes: 生き残った 9 個はどれも HEAD の動きが正しい場合で、テストの行がその分岐を見分けていないだけだった。9 個とも、場合か assertion を 1 つ足せば落ちることを scratch で確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `open`・`shot`・`digest` の動きが変わらない(AC8) | 通る | origin/main の suite(103 件)を HEAD の `scripts/plan-visual.sh` で打って 103/103。HEAD の suite の差分で消えた行は先頭のコメント 1 行だけ |
| 雛形 4 面が `lint` を通る(AC3) | 通る | suite の 4 面のループ 8 件が通る。4 面を origin/main の雛形に替えた scratch では、この 8 件だけが FAIL(193/201) |
| 前の形のページを `lint` が咎める(AC2) | 通る | origin/main の雛形は L1・L2・L5・L6 の 7 行で rc 1、作業中の org-director-start のページは 8 行で rc 1(evidence のファイル) |
| `a=b.html` を awk の代入と取り違えない | 通る | suite の場合と mutation O2(stdin で読むのをやめる)が 2 件で落ちる |
| `leader.md` の 3 文の固定(AC10) | 通る | origin/main の `leader.md` を overlay で差し替えると、新しい 3 行の固定だけが FAIL。1 文ずつ消した 3 つの overlay では、消した文の固定 1 行だけが FAIL。変えない写しの overlay は ok。go1.26.0 と go1.25.8 で同じ |
| この計画のページが `lint` を通る(AC9) | 通る | `.harness/state/plan-visual/plan-visual-easy.html` は root と `templates/base/` の helper のどちらでも rc 0、出力なし |

### 実地で見たこと

- 雛形を本物の headless Chrome で撮って PNG を読んだ(PNG は scratchpad に置き、リポジトリには入れていない)
  - `--fragment detail-overview --width 1150 --height 460`: 閉じた折りたたみが開き、「Detail 1 · Overview」の図だけが写った(等幅のパス 3 つ、S1・S2 の印、group の帯)
  - `--fragment detail-1`: 「Detail 2 · __DETAIL_TITLE__」の図だけが写った
  - `--fragment overview --width 1150 --height 530`: かんたん版の図 1 だけ(Today と After の 2 段、人の形、4 つの箱)
  - 全体(`--height 1500`): 見出し、図 1、「Before you approve」、閉じた折りたたみの 1 行。詳細図は写らない(D5 の前提どおり)
- 代表的な 12 のページ(雛形、この計画のページ、L1〜L6 をそれぞれ 1 か所だけ破った雛形、前の形の 2 ページ、空のファイル、図 1 の途中で切れたページ)の `lint` の出力は、macOS の awk、mawk 1.3.4、GNU Awk 5.2.1 で byte 単位で同じだった(R5)
- L1〜L6 を 1 つずつ破った雛形は、破った規則の行だけを出して rc 1 だった。L6 の折りたたみをなくした雛形は、L6 の 4 行(折りたたみがない、`detail-overview` が折りたたみの中にない、詳細図 2 枚が外にある)を出し、ほかの規則の行は出さなかった

## Test gaps

生き残った mutation(scratch で、生き残りを落とす fixture を確かめた。どれも LOW で、merge を止めるものではない):

| Mutation | 生き残った理由 | 落とす fixture(確かめた) |
| --- | --- | --- |
| L1e コメントの分岐を消す(`<!--` を `<!` の分岐に任せ、最初の `>` で切る) | suite のコメント `<!-- <figure id="overview"> ... -->` は、最初の `>` が `"overview"` の直後にあり、`<figure>` を食べてしまう | `<!-- a > b <figure class="easy" id="overview"> -->` を図 1 の前に足した雛形。HEAD は rc 0、mutant は L1 の「2 elements」で rc 1 |
| L2b かんたん版の図の上限 3 を 2 にする | 3 枚で通るページが suite にない(雛形は 1 枚、この計画のページは 3 枚だがコミットされない) | 図 1 を 2 つ写して 3 枚にした雛形。HEAD は rc 0、mutant は L2 で rc 1 |
| L2c 下限 1 を 0 にする | かんたん版の図が 0 枚なら L1 も必ず咎めるので終了コードは 1 のまま。前の形のページと空のファイルの場合は L1 の行しか確かめていない | 前の形のページで `page: L2: 0 easy figures ...` の行を確かめる |
| L4f `aria-label` を `<svg>` 以外に付けても認める | suite の L4 の fixture は `aria-label` を `<svg>` にしか書かない | `<figure ... aria-label="x">` で `<svg>` に `aria-label` がない雛形。HEAD は L4 の行、mutant は何も出さない |
| L5b 「承認の前に」が 2 つでも咎めない | 2 つある場合が suite にない(計画の L5 は「1 つある」) | `<section class="before ...">` を 2 つにした雛形。HEAD は L5 の「2 ... elements」、mutant は何も出さない |
| O6 同じ行を 1 回だけ出す仕組みを消す | 同じ語や同じ `mono` が 2 回出る fixture がない | `internal/org` を 2 回、`class="mono"` を 2 回書いた図。HEAD は 2 行、mutant は 4 行 |
| O7 同じ属性が 2 つあるとき後ろを読む | 計画の外の決まり(S1 で決めた「最初が勝つ」)で、場合がない | `id="overview" id="easy-x"` の図。HEAD は rc 0、mutant は L1 |
| O9 ディレクトリを渡しても読みにいく | mutant でも awk が読めずに `lint failed: awk exited 2` で rc 1 になり、suite は終了コードしか見ない | ディレクトリの場合に stdout が空であること、stderr が `not found or not readable` であることを確かめる(mutant は L1〜L6 の 5 行を stdout に出す) |
| O11 awk が 0・1 以外で終わったときに 0 を返す | awk が落ちる場合が suite にない | PATH の先頭に `exit 2` だけの `awk` を置く。HEAD は rc 1、mutant は rc 0 |

そのほか:

- `templates/base/scripts/plan-visual.sh` は suite が打たない。root の写しと byte 同一で(`cmp`)、ずれは `check-sync.sh` が止める(verify で確認済み)。この計画のページと前の形の雛形には、template 側の写しでも `lint` をかけて同じ終了コードだった
- `/plan` の step 10.c(worktree の helper に `lint` の行がなければ main のチェックアウトの helper を使う)と、`/pr` の step 4.a(`detail-overview` の有無でページの形を見分ける)は skill の文の指示で、動くテストはない。テストが固定するのは、見分けに使う `--help` の 1 行だけ
- `<details>` が fragment で開くことは、この機械の Chrome でしか確かめていない(R3)
- 見出しが結論の文か、CamelCase の関数名、読みやすさ、重なりは、計画の Non-goals どおり `lint` の外で、テストもない
- consult-advisor への相談はしていない。この subagent には Skill のツールがなく、`/consult` を打てない

## Verdict

- Pass: `run-test.sh`(shell 41 本 4,216 件と Go 8 packages)、`tests/test-plan-visual.sh` 201/201(bash、sh、mawk、gawk)、`go test ./internal/org/...`(go1.26.0 と go1.25.8)、AC2・AC3・AC8・AC9・AC10 の動き
- Fail: なし
- Blocked: なし
- Verdict: pass。生き残った mutation 9 個は、テストの行を足せば閉じられる LOW の穴で、HEAD の動きの誤りではない
