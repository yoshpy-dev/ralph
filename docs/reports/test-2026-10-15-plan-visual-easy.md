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
- Verdict: pass(cycle 2 の HEAD 0e6dbf53 で打ち直した。cycle 1 も pass。上の Pass の行と下の文は cycle 1 の時点のもので、cycle 2 の分は `## Cycle 2` の「Verdict(cycle 2)」にある)。cycle 1 の時点で生き残った mutation 9 個は、テストの行を足せば閉じられる LOW の穴で、HEAD の動きの誤りではない

## Cycle 2

- Date: 2026-10-15
- Tester: tester subagent (Claude)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 2 で、既定の上限 2 の最後)。cross-review cycle 1 の ACTION_REQUIRED 2 件の直し 52c9db31、self-review cycle 2(85a46052、Merge: yes)の LOW の直し 55cce2b7、verify cycle 2(0e6dbf53、pass)のあとに走らせた
- Scope: `git diff fb7648b0..HEAD`(HEAD 0e6dbf53)。動きが変わったのは `scripts/plan-visual.sh` の `lint_page` だけで(L6: 図が開いている間に次の `<figure` が来たら咎めて、開いている図を調べてから閉じる。L3: 10 個のインラインのタグは語を区切らず、`x` か `y` を持つ `tspan` の開始タグは区切る)、`templates/base/scripts/plan-visual.sh` も同じ写し。`tests/test-plan-visual.sh` は 42 件増えて 243 件になった(差分は +83 −0 で、cycle 1 の 201 件はそのまま)。ほかは `diagrams.md` 4 面の L3・L6 の文、計画の Test plan の 2 行、記録。静的解析は /verify の受け持ちなので打っていない
- Evidence: `docs/evidence/plan-visual-lint-2026-10-15.txt` の「cycle 2」の節(cycle 2 の fixture の `lint` の出力、awk 3 種の比較、mutation の一覧、生き残りを落とす fixture。コミットする)、`docs/evidence/test-2026-10-15-plan-visual-easy.log` の「Cycle 2」の節(`run-test.sh` と `go test` の生の出力。git は無視する)

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`: shell 41 本(範囲は cycle 1 と同じく full に戻った。`unclassified:.agents/skills/plan/visual-template.html`) | 4,259 | 4,258 | 0 | 1 | 16 分 42 秒(Go と合わせて。始めたときの load average は 1 分 8.5、5 分 13.1) |
| `./scripts/run-test.sh`: Go `go test ./...`(go1.26.0) | 8 packages | 8 | 0 | 0 | org 88.2 s、scaffold 0.6 s。ほかの 6 つは `(cached)` |
| `go test -count=1 ./internal/cli/ ./internal/config/ ./internal/insights/ ./internal/upgrade/`(上で `(cached)` だった 4 つ) | 4 packages | 4 | 0 | 0 | cli 309.8 s、config 0.9 s、insights 1.5 s、upgrade 2.0 s |
| `bash tests/test-plan-visual.sh` | 243 | 243 | 0 | 0 | 18 s |
| `sh tests/test-plan-visual.sh`(macOS の `/bin/sh` は bash 3.2) | 243 | 243 | 0 | 0 | 15 s |
| `go test -count=1 ./internal/org/...`(go1.26.0) | 3 packages | 3 | 0 | 0 | org 126.7 s |
| `GOTOOLCHAIN=go1.25.8 go test -count=1 ./internal/org/...` | 3 packages | 3 | 0 | 0 | org 114.6 s |
| ubuntu:24.04 の mawk 1.3.4 20240123 で `sh tests/test-plan-visual.sh`(`--network none`) | 243 | 242 | 0 | 1 | — |
| ralph-guard-awk:test の GNU Awk 5.2.1 で同じ suite | 243 | 242 | 0 | 1 | — |
| fb7648b0(直す前)の helper で HEAD の suite を打つ(直しを見分けるか) | 243 | 209 | 34(予想どおり) | 0 | — |
| `lint` の mutation: 52c9db31 で増えた分岐 | 40 | 37 が落ちた | — | 3 が生き残った | 約 6 分(4 並列) |
| `lint` の mutation: cycle 1 の 57 個を HEAD の suite で打ち直す | 57 | 49 が落ちた | — | 8 が生き残った | 約 7 分(4 並列) |

- shell の数え方は cycle 1 と同じ(`PASS`・`FAIL`・`SKIP` で始まる行で、集計の行は除く)。PASS は cycle 1 の 4,216 から 42 増えて 4,258 で、増えたのは `tests/test-plan-visual.sh` の 42 行だけだった。ほかの suite の行は、guard の AC8 の行の秒数を除いて cycle 1 と同じ。`SKIP` の 1 件は cycle 1 と同じ `tests/test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」で、この変更とは関係がない
- Docker の 2 回の `SKIP` 1 件ずつは cycle 1 と同じ `digest` の節の `shasum is not installed here`。`lint` の節の 139 件(cycle 1 の 97 件と cycle 2 の 42 件)は、どちらの awk でもすべて通った
- `run-test.sh` で `(cached)` だった 6 つのうち、`internal/org/driver` と `internal/org/protocol` は上の `go test -count=1 ./internal/org/...` で、残りの 4 つは表の 3 行目で打ち直した。cycle 2 は Go のファイルを変えていない
- 直す前の helper で通った 8 件は、fixture が `</figure>` を本当に 1 つ消したかを確かめる 2 件、咎めない場合の `l3-split` の 2 件(直す前の helper はどのタグでも語を区切るので通る)、`l6-nested-easy` と `l6-nested-checked` の終了コードなど 4 件。同じケースの L6 の行の assertion は、直す前の helper で落ちる。一覧は evidence のファイル

### cycle 2 で足したテストと Test plan の対応

| 計画の Edge cases の行 | ケース(assertion の数) | 結果 |
| --- | --- | --- |
| 途中の図の閉じ忘れ: 図が開いている間に次の `<figure` が来たら L6 で咎める | `l6-nested`(5)、`l6-nested-noid`(3)、`l6-nested-easy`(4)、`l6-nested-checked`(5)、雛形から `detail-overview` の `</figure>` を抜いたページ(3) | 20 件すべて通る |
| インラインのタグをまたぐ語: `internal/<tspan>org</tspan>` と `tspan` で分けた snake_case は咎め、`x` か `y` を持つ `tspan` と別の `<text>` は咎めない | `l3-inline`(8)、`l3-inline-each`(12)、`l3-split`(2) | 22 件すべて通る |

- evidence の fixture でも同じ動きを確かめた: 雛形から overview か `detail-overview` の `</figure>` を消すと L6 の 1 行で rc 1、閉じ忘れた overview に `internal/org` を足すと L3 と L6 の 2 行、svg の `tspan` 4 行と 10 個のタグの `<p>` はそれぞれ L3 の行で rc 1、別の `<text>`・位置つきの `tspan`・`<br>`・`<tspan>--all</tspan>`・`<b>/director</b>` は rc 0
- 一覧の外の `<mark>`、大文字の `<TSPAN>`、コメント(`internal/<!-- c -->org`)は、今の決まりどおり語を区切って rc 0 だった(verify cycle 2 の C2-I2 と同じ。ブラウザでは `internal/org` と描かれる)

### Coverage(cycle 2)

- Statement: shell を計る道具はない。Go は cycle 2 で変わっていない
- Branch: mutation で測った。52c9db31 で増えた分岐の 40 個のうち 37 個で suite が落ちた。入れ子の開始の分岐(L6 の行、閉じる、閉じる前に調べる、ラベル)、`close_figures` の 3 つの働き、`tag_splits` の 5 通りの書き換え、`HAS_XY` の 5 通り、`parse_attrs` と `tag_splits` の順、`<br>` と `<text>` を一覧に足すこと、10 個のタグを 1 つずつ一覧から外すことは、どれも落ちた。生き残りは下の Test gaps の 3 個(1 個は等価)
- cycle 1 の 57 個を HEAD の suite で打ち直すと 49 個が落ちた。cycle 1 で生き残った 9 個のうち L2b が今は落ち、残りの 8 個は今も生き残る。cycle 1 で落ちた 48 個は今もすべて落ちる(O10 は `close_figures` に移ったので O10c2 として打った)
- L2b(かんたん版の図の上限 3 を 2 にする)を落としたのは、55cce2b7 の `l6-nested-checked` の「an easy figure closed by a nested start gives exactly 2 findings」の 1 件だけだった。図 3 枚のページで mutant が `page: L2: 3 easy figures …` の行を足し、3 行になる。verify cycle 2 の見立てどおり。ただし L2 の行を確かめる assertion ではなく、行の数で落ちている

### Regression checks(cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の 201 件 | 通る | HEAD の suite で 201 件は変わらず通る。直す前の helper でも 201 件は通る(落ちた 34 件はどれも cycle 2 の 42 件の中) |
| cycle 1 の 12 個の fixture の `lint` の出力 | 変わらない | 同じ手順で作り直した fixture は cycle 1 のものと byte 単位で同じで、HEAD の helper の出力は cycle 1 の evidence の節と diff 0 行 |
| 雛形とこの計画のページが `lint` を通る(AC3、AC9) | 通る | 雛形と `.harness/state/plan-visual/plan-visual-easy.html` は、root と `templates/base/` の helper のどちらでも rc 0 |
| `open`・`shot`・`digest` の動き(AC8) | 通る | HEAD の suite のそれらの節は cycle 1 と同じ行で通る |
| `leader.md` の 3 文の固定(AC10) | 通る | `go test -count=1 ./internal/org/...` が go1.26.0 と go1.25.8 で ok |
| awk の方言(R5) | 同じ | 22 個の fixture(cycle 1 の 12 個、cycle 2 の 7 個、生き残りを落とす 3 個)の出力は、macOS の awk、mawk 1.3.4、GNU Awk 5.2.1 で byte 単位で同じ(md5 ded83d8b…)。`templates/base/` の helper でも同じ md5 |

### Test gaps(cycle 2)

生き残った mutation(どれも LOW で、merge を止めるものではない。生き残りを落とす fixture は scratch で HEAD と mutant にかけて確かめた):

| Mutation | 生き残った理由 | 落とす fixture(確かめた) |
| --- | --- | --- |
| B3 コメントの分岐の `tag_space()` を消す(コメントが語を区切らなくなる) | コメントで語を区切る場合が suite にない。HEAD の動き(区切る)は 52c9db31 のコミットの文と `tag_space` の注にあるが、固定する行がない。なお mutant の動き(`internal/<!-- c -->org` を咎める)はブラウザの描き方と合う | 図 1 の foot に `<p>internal/<!-- c -->org を直す</p>`。HEAD は rc 0、mutant は `overview: L3: path-like text: internal/org` で rc 1 |
| T4 `tag_splits` の `name == "tspan"` を外す(`x` か `y` を持つインラインのタグはどれも区切る) | `x` か `y` を持つ `tspan` 以外のインラインのタグが suite にない。実物のページで `<span x>` のような書き方はまずない | `<p>in<span x="1">/x</span> と</p>`。HEAD は `overview: L3: path-like text: in/x` で rc 1、mutant は rc 0 |
| N9 入れ子の L6 の行で、開いている図を `PEND[NPEND]` でなく `PEND[1]` で名指す | 等価。入れ子の開始は開いている図をすべて閉じてから自分を積み、`</figure>` もすべてを閉じるので、この分岐に入るとき NPEND は必ず 1 | 落とせない。overview と `detail-overview` の `</figure>` を両方消したページでも、HEAD と mutant の出力は同じ 2 行 |

cycle 1 から持ち越す生き残り 8 個(L1e、L2c、L4f、L5b、O6、O7、O9、O11)。cycle 1 の Test gaps の表の fixture を HEAD の helper と mutant の写しにかけ直し、8 個とも今も出力が分かれることを確かめた。L1e の mutant は、cycle 2 の入れ子の検査で L6 の行が 1 行増えた(コメントの中の `<figure>` を本物の開始と読み、その中で本物の overview が始まるため)。

そのほか:

- tech-debt の 252 行(lint のテストの穴)に載せる生き残りは、cycle 1 の一覧の 8 個(L1e、L2c、L4f、L5b、O6、O7、O9、O11)と cycle 2 の 2 個(B3、T4)。L2b は外せる(上の Coverage のとおり、行の数でしか落ちていない)。N9 は等価なので載せない。suite のケースの数は 243
- `templates/base/scripts/plan-visual.sh` は今も suite が打たない。root の写しと `cmp` で同一で、22 個の fixture をかけた出力も root と同じ md5 だった
- `/plan` の step 10.c と `/pr` の step 4.a は、cycle 1 と同じく動くテストがない(cycle 2 で変わっていない)
- `<details>` が fragment で開くことは、cycle 1 の Chrome での確認のまま。雛形は cycle 2 で変わっていないので撮り直していない
- consult-advisor への相談はしていない。この subagent には Skill のツールがなく、`/consult` を打てない

### Verdict(cycle 2)

- Verdict: pass(cycle 2 の HEAD 0e6dbf53 の時点)
- Pass: `run-test.sh`(shell 41 本 4,258 件と Go 8 packages)、`(cached)` だった 4 つの打ち直し、`tests/test-plan-visual.sh` 243/243(bash、sh、mawk、gawk。mawk と gawk は `digest` の SKIP 1 件を除く 242 件)、`go test -count=1 ./internal/org/...`(go1.26.0 と go1.25.8)。計画の Test plan の Edge cases の新しい 2 行は、cycle 2 で足した 42 件がすべて通り、直す前の helper ではそのうち 34 件が落ちる
- Fail: なし
- Blocked: なし
- Mutation: 52c9db31 で増えた分岐は 40 個のうち 37 個が落ちた(生き残りは B3 と T4、N9 は等価)。cycle 1 の 57 個は 49 個が落ち、L2b が新しく落ちた
