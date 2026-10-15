# Self-review report: plan-visual-easy

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-plan-visual-easy.md(承認済み、digest 819a5be1cf1f)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)と 2 回目(cycle 2)
- Latest cycle: 2/2(末尾の「Cycle 2」節)。cycle 2 の重大度は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2。Merge: yes。cycle 1 の節(以下「Cycle 2」より前)は cycle 1 の時点の記録で、数と Merge の行はそのまま残している
- Scope: diff の品質だけ。`git diff origin/main...HEAD`(HEAD 69a9dc1b、6 コミット、32 ファイル、+2665/-604)。S1 `821a3c69`、S2 `5b056944`、S3 `138481e0`。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない

## Evidence reviewed

- 読んだ差分(全体): `scripts/plan-visual.sh`(`lint_page` の awk(292〜556 行)すべて、`cmd_lint`、usage とヘッダコメント、ディスパッチ)、`tests/test-plan-visual.sh` の lint 節(fixture と 全 case)、`.claude/skills/plan/visual-template.html`(新しい版を全行)、`.claude/skills/plan/diagrams.md`(全行)、`/plan` の step 10・12.b、`/pr` の step 4.a、`leader.md` の step 2 と `prompts_test.go` の 3 行、`template.md` / `feature-plan.md` のコメント、AGENTS.md・README・repo-map の行、tech-debt の追加行
- `open`・`shot`・`digest` の本体: `git diff` の hunk は、ヘッダコメント、usage、`cmd_shot` と `digest_body` の間に入った `lint_page` と `cmd_lint`、ディスパッチの `lint)` の分岐だけで、3 つの関数の中身には触れていない
- 4 面の写し: `.claude/skills/{plan/SKILL.md, plan/diagrams.md, plan/template.md, plan/visual-template.html, pr/SKILL.md}` を `.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/` と `cmp` した 15 組、`scripts/plan-visual.sh` と `templates/base/scripts/plan-visual.sh`、`docs/plans/templates/feature-plan.md` と `.claude/skills/plan/template.md`、同じく template 側の `feature-plan.md`。すべて byte 同一
- 動かして見たもの(この macOS の awk 20200816 だけ): 雛形で `lint` が rc 0 で 0.17 秒、雛形に詳細図を 40 回足した 38 KB のページで 0.23 秒。次の入力を作って打った: `aria-label` にパスと snake_case を入れた easy 図(rc 0、何も出ない)、`<FIGURE>`(L1 と L2 で咎める)、`and/or`・`Yes/No`・`Node.js`(L3 で咎める)。tech-debt の `a=b.md` の行は、scratch で `digest a=b.md </dev/null` が `e3b0c44298fc`・rc 0、`2026-a=b.md` が実ファイルの digest になることで確かめた。diagrams.md の Credit の eli5 の URL は `gh api` で `eli5/` が実在するのを確かめた
- 確かめていないもの: mawk 1.3.4 での実行(この機械に mawk がない。S1 の進捗メモの Docker での結果と、POSIX 外の機能がないことをコードで読んだことに頼る)、テスト全体の結果(/test)、`:has()` と `:target` の描画(S2 の PNG は計画が読んだと書くが、ここでは撮っていない)、図の読みやすさ

## Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 1、LOW 5。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | readability | `/pr` step 4.a の「Match the figure to the diff」は 1 つの長い箇条書きで、「`id="detail-overview"` があれば今の形」の分岐の中に「Then, if what was built no longer matches what the easy figures ... told the approver, edit the easy figures ...」という入れ子の if がある。その直後の「Otherwise the page was made before the current form: ...」は、いちばん近い if(簡単版の図が合っているか)の else と読める。そう読むと、今の形のページで簡単版の図が diff と合っているときに、前の形の手順(「overview が示す部品を diff に合わせて直す」)へ入り、overview(かんたん版の図 1)にパスを足す直しをしうる。PR に貼る図と `lint` の決まり(L3)に反する | `.claude/skills/pr/SKILL.md:37`(4 面とも同じ)。1 行 約 1200 字。前の版は「overview と file-map を diff と照らし、変わっていれば直す」の 2 文だけだった(`git diff origin/main...HEAD -- .claude/skills/pr/SKILL.md`) | 分岐を 3 つの子の箇条書きに分ける。(a)「ページに `id="detail-overview"`(またはシングルクォート)がある: 今の形」(`detail-overview` とファイルの地図を diff と照らす。そのあと簡単版の図が承認時の説明と変わっていれば、平易な言葉で直す)、(b)「ない: 前の形」(overview を今のとおり照らす)、(c)「どちらも: diff が示すものだけを足す・直す」。「Otherwise」をやめて(b)に条件を書く |
| L1 | LOW | maintainability | L3(パス・拡張子・snake_case・`<code>`・mono)は図の「文字」だけを見て、`aria-label` などの属性値は見ない。`diagrams.md` と `--help` の L3 は「in an easy figure」と書くので、属性まで掛かるように読める。`aria-label="internal/org/watch.go と workspace_created"` の easy 図が rc 0 で通る(probe)。aria-label は画面に出ないので、承認する人には見えない。ただ、`__OVERVIEW_ALT__` に計画のパスを写す書き手は出うる | `scripts/plan-visual.sh:430`(aria-label は L4 の空判定にだけ使う)、`scripts/plan-visual.sh:459-480`(`scan_text` は `TEXT[f]` だけ)、`.claude/skills/plan/diagrams.md:222-225` | 直すなら、easy 図の `<svg>` の `A_ARIA` を `TEXT[f]` に足す(1 行)。直さないなら、`diagrams.md` と usage の L3 を「文字(`aria-label` などの属性は見ない)」と書く。どちらにするかは /verify が計画の L3(「図の中」)と照らして決める |
| L2 | LOW | consistency | `leader.md` step 2 は leader に「`diagrams.md` に沿って書く」と言い、古い helper(`--help` に lint の行がない)のときは lint を飛ばして QUESTION に書くよう言う。一方、leader が読む `diagrams.md` の「Self-check」step 1 は、同じ条件のときに main のチェックアウトの helper を打つよう言う。同じ条件に対する逃げ道が 2 つあり、leader には書いていない側が見える。計画は「leader には main のチェックアウトの逃げ道を書かない」と決めている(S3 の進捗メモ)が、`diagrams.md` を経由してそれが届く | `internal/org/prompts/leader.md:102-107`、`.claude/skills/plan/diagrams.md:190-201` | `leader.md` に「`diagrams.md` の main のチェックアウトの helper の手順は使わない」の 1 句を足す。計画の決定どおり逃げ道を leader に認めるなら、leader.md を `diagrams.md` に合わせる。いずれにしても prompts_test.go の固定文を合わせる |
| L3 | LOW | maintainability | `/pr` で簡単版の図を直したあとに `lint` を打つ指示がない。S3 の進捗メモに「範囲外で気づいたこと」として 1 行あるが、計画の進捗メモは archive へ移り、tech-debt の登録簿には行がない。`/pr` の「`/plan` step 10.c と同じ自己点検」は、10.c の中で Lint と Self-check が別の段になったので、lint を含むかが読み取れない。前の形のページに lint をかけると L1 で咎めるので、「含む」とだけ書くのも足りない | `docs/plans/active/2026-10-15-plan-visual-easy.md:151`(S3 のメモの最後)、`.claude/skills/pr/SKILL.md:38`、`docs/tech-debt/README.md`(該当の行なし) | 4.a の(a)の最後に「簡単版の図を直したら、撮る前に `./scripts/plan-visual.sh lint <ページ>` を打つ(今の形のページだけ)」と足す。足さないなら、登録簿に 1 行足す(`/pr` の 4.a、Trigger は /pr の次の変更) |
| L4 | LOW | unnecessary-change | 雛形の CSS と placeholder に、使われていない・文書にない・紛らわしいものがある。(1) `.easy svg .t-head` はどこでも使われない(雛形の SVG にも `diagrams.md` にもない)。(2) 図の見出しの丸い番号 `.easy h2 .no`、行の見出し `t-row` / `t-row now`、仕切りの `rule` は雛形の例で使うが、`diagrams.md` の「Easy figures」の語彙にも雛形の先頭のコメントにも出ない。書き手が 2 枚目・3 枚目で写し忘れる。(3) `__NON_GOAL__`(承認の前に)と `__NON_GOALS__`(折りたたみの中)は 1 字違いで、`grep __NON_GOAL` が両方に当たる。リポジトリの方針は grep しやすい名前 | `.claude/skills/plan/visual-template.html:57`(`.t-head`)、`:43`(`.no`)、`:226`・`:316`(placeholder)、`.claude/skills/plan/diagrams.md:68-103` | `.t-head` の行を消す。`no`・`t-row`・`rule` を `diagrams.md` の Easy figures に 1 行で足す(または雛形の easy 図のコメントに)。placeholder は `__NOT_IN_PLAN__` と `__NOT_IN_PAGE__` のように、見出しの文言と対にして名を変える |
| L5 | LOW | readability | awk の読みにくさが 2 点ある。(1) `scan_text` の拡張子の正規表現の先頭の文字の集まり `[A-Za-z0-9_;-]` が `;` を含むのは、`&gt;.html`(`plan-visual/&lt;slug&gt;.html` を拡張子で咎めるテスト)のためだが、コメントがない。読む人は `;` を消してよいか判断できない。(2) `widen` は結果の終わり位置を大域変数 `W_END` で返し、呼び出し側は `widen` のあとに `RSTART` が変わらないことに頼る(`scan_text` の 2 つ目のループは `RSTART + RLENGTH - 1`)。いまは正しいが、`widen` の中で `match` を使うと壊れ、テストは 7 件の出力数で気づく程度 | `scripts/plan-visual.sh:468`、`:446-455`・`:463`・`:476` | `:468` の上に「`&lt;slug&gt;.html` の `;` を前の文字として許す」と 1 行。`widen` の注に「`match` を呼ばない(呼び出し側が `RSTART` を使う)」と足す |

## Positive notes

- `lint_page` は POSIX の awk だけでできている。`gensub`・`match` の配列・`RS` の正規表現・`{n}` の区間表現・`[[:space:]]`・`length(array)` は使われておらず、`substr` / `index` / `match` / `split` / `sub` と、局所変数を引数の後ろに並べた関数だけ。引用符は `SQ = "\047"` と `DQ = "\""` で書いてあり、シェルの単一引用符の中にアポストロフィが出ない。この機械の awk 20200816 で動く
- `lint_page` を `< "$1"` で読み、`a=b.html` が awk の変数代入にならないようにしたこと(コメントとテストがある)。同じ穴が `digest` にあることを見つけて、挙動を変えずに tech-debt に書いた(行の内容は scratch で確かめて合っている)
- `cmd_lint` の終了コード: awk の 0 と 1 をそのまま返し、それ以外(awk の構文エラーなど)は「lint failed: awk exited N」で 1。ファイルなし、ディレクトリ、引数の数の誤りは `usage_error` か `die 1` で、`open`・`shot`・`digest` と同じ形
- 咎めた語を出力に含める(`path-like text: internal/org`)ので、R1 の言い換えができる。同じ行は 1 回だけ出る(`SEEN`)
- タグの読み方(`tag_end`、`parse_attrs`、コメント・style・script の飛ばし)は、引用符の中の `>`、複数行のタグ、シングルクォート、閉じていない引用符、途中で切れたファイル、空のファイルのそれぞれにテストがある。読み切れない入力は、落ちずに L1 か L6 の行になる
- テストの fixture は 1 つの正しいページを部品にして、規則ごとに 1 つだけ壊す形(`assert_only_rule` でほかの規則が出ないことまで固定)で、読みやすい
- 4 面と root・template の写しがすべて byte 同一。leader.md の追加と `prompts_test.go` の 3 行は同じ文字列で、`--help` の行の固定(`  scripts/plan-visual.sh lint <html>`)は skill の見分け方と同じ文字列
- 登録簿の行は他の行と同じく区切りの `|` が 6 個で、列がずれていない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L3: `/pr` で簡単版の図を直したあとの `lint` | 承認後に直した簡単版の図が、L3・L4 に反しても気づかれない | S3 で範囲外と判断した(進捗メモ) | `/pr` の 4.a を次に変えるとき | `docs/plans/active/2026-10-15-plan-visual-easy.md`(進捗メモの S3) |

_(登録簿には追記していない。L3 は M1 の直しと同じ箇所で 1 文足せば済む。cycle 1 の fix で入れないなら、その回に行として足す。)_

## Coverage gaps

- mawk 1.3.4 での `lint` の実行は、この機械ではしていない。CI(Ubuntu、mawk)が R5 の最後の確認
- `:has()` / `:target` / `<details>` の自動展開は Chrome の機能で、雛形を `#overview` と `#detail-overview` で開いた PNG は S2 が読んだと書くが、ここでは見ていない
- `lint` は大文字のタグを読まない(計画の前提どおり)。`<FIGURE id="overview">` は L1 と L2 で咎められ、出力の文言は「`<FIGURE>` にある」になる。書き手が大文字で書くと原因に気づきにくいが、計画が決めた範囲なので指摘にしない

## Recommendation

- Merge: yes(CRITICAL・HIGH はなし。MEDIUM 1 件は `/pr` の 1 文の読み違いで、コードの欠陥ではない。直してから進めるのがよい)
- Follow-ups:
  1. M1: `/pr` の 4.a の「Match the figure to the diff」を、今の形・前の形・どちらもの 3 つの箇条書きに分け、「Otherwise」をやめる(4 面)。L3 の 1 文もここで足す
  2. L1: `aria-label` を L3 の対象に入れるか、`diagrams.md` と usage の L3 の文言を「文字」に限る。/verify の仕様の照合と合わせて決める
  3. L2: `leader.md` と `diagrams.md` の、古い helper の逃げ道を 1 つにそろえる
  4. L4・L5 は小さな差分(CSS の 1 行、コメント 2 行、`diagrams.md` の語彙 1 行、placeholder の改名)で直せる。cycle 1 の fix に入れるとよい。直さずに最後の cycle に届いたものは、1 つの行にまとめて登録簿へ

## Cycle 2

- Date: 2026-10-15
- Reviewed HEAD: `52c9db31`(cross-review の修正)
- Scope: `git diff fb7648b0..HEAD`(10 ファイル、+240/-47)。cross-review の指摘 2 件の修正 `52c9db31`(`lint_page` の awk、テスト 37 件、`diagrams.md` の L3・L6 の 2 行、`templates/base` の写し)と、記録(triage、計画の Test plan と digest、insight event)。範囲は cycle 1 と同じ diff の品質だけで、テストの実行、静的解析、仕様適合、文書のずれの検査はしていない
- Cycle: 2/2(上限の cycle)
- 指摘の ID: 以下の `C2-n` は、この節の中だけで使う。cycle 1 の `M1`・`L1`〜`L5` とは別の番号

### Cycle 1 の指摘の扱い

| ID | HEAD での状態 | 確認したこと |
| --- | --- | --- |
| M1 | 直した(`0067b837`) | `.claude/skills/pr/SKILL.md:37-40` は、見分け方の 1 行と 3 つの子の箇条書き(今の形、前の形、どちらも)で、「Otherwise」がない。4 面で同じ |
| L1 | 直した(文書だけ、`0067b837`) | `diagrams.md` と usage が、L3 は属性を読まないと書く。verify が計画の L3 と照らして同意している |
| L2 | 残した(verify が同意) | `leader.md:102-106` と `diagrams.md:195-200` は今も別の逃げ道を書く。verify の説明(leader が読む `diagrams.md` と helper は同じ worktree の同じコミットから来るので、2 つの指示が同時に届く組み合わせはない)を、`leader.md:95-98` の読む場所の指定と照らして受け入れる。登録簿の行は要らない |
| L3 | 直した(`0067b837`) | `/pr` 4.a の今の形の末尾に、直したあとは `/plan` step 10.c と同じに lint を打ち、lint がどちらの helper にもなければ飛ばして節に書く、とある |
| L4 | 直した(`0067b837`) | 雛形に `.t-head` と `__NON_GOAL` がない |
| L5 | 直した(`0067b837`) | `scripts/plan-visual.sh:476` と `:499` にコメントがある |

### Evidence reviewed

- 読んだ差分: `scripts/plan-visual.sh` の cycle 2 の hunk すべて(`BEGIN` の `INLINE`、`parse_attrs` の `HAS_XY`、`tag_splits`、`close_figures`、`start_tag` の入れ子の分岐、`end_tag`)と、その周りの `lint_page` の全行、`tests/test-plan-visual.sh` の新しい 7 ケース、`diagrams.md` の L3・L6、plan-visual.sh のヘッダ、計画の差分、triage、insight event
- `cmp`: `diagrams.md` の 4 面と、`scripts/plan-visual.sh` と `templates/base/scripts/plan-visual.sh` は byte 同一。`git diff fb7648b0..HEAD --check` は何も出さない
- この修正の 3 つの awk での確認(手で作った 12 ページに `lint` を掛けただけで、テストスイートは動かしていない): macOS の awk 20200816、ubuntu:24.04 の mawk と gawk 5.2.1(docker、apt 版)。3 つとも出力が同じ。結果は次の表
- `digest` を打つと `3b11a1def777` で、計画の `- Approved:` と一致する。この計画のページ(`.harness/state/plan-visual/plan-visual-easy.html`)と雛形は `lint` が rc 0
- テストの数: コミットの「37 件」は、新しい 7 ケースの assertion を数えて 37(8、12、2、5、3、4、3)で合う

| probe(easy 図の中の text) | 結果 |
| --- | --- |
| `<text>run</text><text>/director</text>`(`x` `y` つき) | 出力なし、rc 0 |
| `internal/<tspan>org</tspan>` | `path-like text: internal/org` |
| `internal/<tspan x="5" y="6">org</tspan>` | 出力なし(行を分ける) |
| `internal/<tspan dx="5">org</tspan>` | `internal/org` を咎める(`dx` は分けない) |
| `<tspan data-x="1">`、`<tspan title="a x=3">` | `internal/org` を咎める(`x` を属性名として読む) |
| `<tspan x>`(値なし) | 出力なし(分ける) |
| `internal/` の後ろの改行と空白のあとに `<tspan>org` | 出力なし(ソースの空白は語を切る) |
| `in<tspan/>ternal/org` | 咎める(自己終了も分けない) |
| 図の中で `<figure id="inner">` が始まる | L6 `figure starts inside figure easy-2` と、`inner` が折りたたみの外の L6 の 2 行 |
| `internal/<mark>org</mark>`、`<u>`、`<abbr>`、`<kbd>`、`<q>` | 出力なし(下の「観察」) |
| 閉じ忘れの easy 図のあとに section と折りたたみ | L3 の 5 行のあとに L6 の 1 行(下の「観察」) |
| `</figure>` が 1 つ余る | 出力なし(これまでと同じ) |

### Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | maintainability(テストが分岐を固定しない) | 入れ子の分岐の `close_figures()` は、開いている図を閉じる前に `check_easy` で、その図の文字と箱を検査する。今のテストは、図が閉じて次の図に文字と箱が流れ込まないことは固定するが、閉じる図の検査が走ることは固定しない。入れ子の開始を試す 4 ケースは、閉じられる図が欠陥のない図(`l6-nested`・`l6-nested-noid`・雛形は non-easy の詳細図、`l6-nested-easy` は欠陥のない overview)ばかりで、出力は L6 の 1 行だけを期待する。そのため行 434 を `NOPEN = 0; NPEND = 0`(閉じるが検査しない)に替えても、4 ケースの期待は変わらないはず(読んだうえの判断で、複製にスイートは掛けていない) | `scripts/plan-visual.sh:434`。probe: overview に `internal/org` を書いて `</figure>` を抜き、次に easy-2 を置いたページは、HEAD で `overview: L3: path-like text: internal/org` と L6 の 2 行、行 434 を替えた複製では L6 の 1 行だけで rc 1(複製に `lint` を掛けただけで、スイートは動かしていない)。`</figure>` の側は同じ `close_figures()` を呼ぶので、検査は既存のテストが固定している。固定されていないのは入れ子の呼び出しの 1 か所だけ | fixture を 1 つ足す: `internal/org` を持つ easy 図に `</figure>` がなく、次に別の図が始まるページで、その図の L3 の行と L6 の行の 2 行が出ること |
| C2-2 | LOW | readability | `diagrams.md` の L3 の箇条書きで、新しい 2 文(「inline tags … do not split a word」と「to break a line, use …」)が、「not other attribute values such as `aria-label`.」とそれに続く「Write the `aria-label` in the same plain words as the figure」の間に入った。`aria-label` を読まないという文と、`aria-label` を図と同じ言葉で書くという助言は、cycle 1 の L1 の直しで 1 組にしたものだった。今は助言が「…, or `<br>`.」の直後に来て、改行の話の続きに読める。同じ箇条書きの「(the words between tags)」も、タグ(インライン)をまたぐ語を 1 語と数えるようになったので、そのままでは正確でない | `.claude/skills/plan/diagrams.md:226-235`(4 面で同じ) | 「Write the `aria-label` …」を「not other attribute values such as `aria-label`.」の直後に戻し、インラインのタグの 2 文を最後に置く。「(the words between tags)」は「(the words between tags that start a new line)」のように、次の文と合う形にしてもよい。skill の文面なので 4 面を同じに直す |

この cycle は上限(2/2)なので、どちらも直さない場合は登録簿に残る場所がほかにない。C2-1 と C2-2 は、すでにある plan-visual-easy の行(`docs/tech-debt/README.md:252`、Test gaps)に (e) として足すのが 1 番簡単で、Trigger は「次に `lint_page` の入れ子の分岐か `diagrams.md` の L3 の文を変えるとき」がよい。直す場合、C2-2 は skill の文面なので記録だけの修正の例外には入らず、上限を上げて全体を回し直すことになる。どちらを選ぶかは lead が決める。

### Positive notes

- `tag_splits` の戻り値は、開始タグ(`parse_attrs` のあと)と終了タグで同じ関数を通る。`start &&` が終了タグで古い `HAS_XY` を読まないようにしていて、`<tspan x y>lib</tspan>/util` のテストがそれを固定している(この条件を外すと落ちる)
- `parse_attrs` の `HAS_XY = 0` の初期化は、`l3-inline` の 1 行目(`<text x y>` の直後の `<tspan>`)が固定している。初期化を外すと `text` の `x` が次の `tspan` に残り、`internal/org` を見逃す
- `class="x"`(属性名ではなく値)、`dx`・`dy`、シングルクォートの `y='140'` を別々の行で固定している。属性名の比較が `==` であることの確認として足りている
- 10 個のインラインのタグそれぞれが、開始と終了の両方で語をつなぐことを `in<t>/t</t>/end` の 1 本で固定する。1 つでも `INLINE` から外すと、その名前の行と件数の assertion の両方が落ちる
- 入れ子の分岐の 4 ケースは、ラベルが新しい図のものであること、id のない図が `figure#5` と順番で呼ばれること、easy 図の箱が 2 つの図にまたがらないこと(overview の 5 個と easy-2 の 2 個で `7 boxes` にならない)、雛形から `detail-overview` の `</figure>` を抜いた現実の形、を 1 つずつ見る
- awk は POSIX の範囲(`split`、`in`、`substr`、`index`、`match` と局所変数)で、3 つの awk で同じ結果になった。`start` は awk の予約語ではない
- コメントは、`tag_space`、`tag_splits`、`close_figures`、入れ子の分岐の理由、`parse_attrs` の `HAS_XY` のどれも、コードと合っている。`end_tag` の「figures do not nest: see start_tag」は、入れ子の分岐への参照になっている
- コミットメッセージの記述(F1・F2・テストの件数・docs の面)は、差分と合っている

### 観察(指摘にしない)

- `INLINE` は 10 個の閉じた一覧で、`diagrams.md` と同じ。HTML の phrasing 要素のうち `mark`・`u`・`abbr`・`kbd`・`q` などは入っておらず、`internal/<mark>org</mark>` は通る(上の probe)。cross-review の指摘は `tspan` の場合で、`/plan` のページは SVG の `<text>` を使うので、書き手がこれらを語の途中に置く見込みは小さい。足すなら `BEGIN` の 1 語と `diagrams.md` の 4 面
- `OPEN` と `PEND` は、入れ子の分岐のあと、1 つしか持てない。`for (k = 1; k <= NOPEN; k++)` の繰り返しと「every figure still open」というコメントは、スタックだった頃の形が残っている。1 つの変数にまとめれば読みやすいが、直すと上限を超えてパイプラインを回し直すので、今は直さなくてよい
- 最後の easy 図の `</figure>` を抜くと、その後ろの `<section class="before">` や折りたたみの chrome の `<code>` とパスが、その図の文字として L3 に出る(probe の 5 行)。ブラウザでも同じ入れ子になるので咎めは正しい。規則の順に出すので、原因の L6 の行は 6 番目に来る。以前は L6 の行が 1 行も出なかったので、修正で改善している。行の文言の「(missing </figure>?)」が手がかりになる
- 入れ子の図が、折りたたみの外にある easy でない図のときは、L6 が同じ図について 2 行出る(「starts inside」と「outside the fold」)。どちらも事実で、重複を嫌うほどの量ではない
- コメントの中の語の切れ目(`internal/<!-- c -->org`)と大文字のタグ(`<TSPAN>`)は、どちらも語の切れ目として扱われる(コメントは空白、`TSPAN` は `INLINE` の名前に合わない)。見逃す方向で、計画が「大文字のタグは読まない」と決めた範囲

### Coverage gaps

- テストスイートは動かしていない(/test の担当)。新しい assertion が通ることは確かめていない。上の probe は、`lint` を手で作った 12 ページに掛けたもの
- mawk のバージョン番号は確かめていない(ubuntu:24.04 の apt 版)。cycle 1 の S1 の記録は mawk 1.3.4
- C2-1 の複製は 1 行だけ替えたもので、テストスイートに掛けて落ちないことまでは確かめていない。読んだ範囲では、閉じられる図に欠陥がある入れ子のテストはない

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C2-1: 入れ子の開始で閉じる図の検査が固定されていない | 入れ子の分岐の呼び出しから検査が落ちても、テストが通る | 最後の cycle で、lead が直すか残すかを決める | 次に `lint_page` の入れ子の分岐を変えるとき | `scripts/plan-visual.sh:434`、`tests/test-plan-visual.sh`(`l6-nested` 系) |
| C2-2: `diagrams.md` の L3 の文の並び | 助言の文が前の文から離れて読みにくい(動作は変わらない) | skill の文面で、直すと 4 面と全体の回し直しになる | 次に `diagrams.md` の L3 を変えるとき | `.claude/skills/plan/diagrams.md:226-235` |

_(登録簿には追記していない。直さないと決めたときは、上の 2 行を `docs/tech-debt/README.md:252` の (e) としてまとめる。)_

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。LOW 2 件)
- Follow-ups:
  1. C2-1: 閉じられる図に `internal/org` がある入れ子のページで、L3 と L6 の 2 行が出ることを固定する fixture を 1 つ足す。直すなら `lint_page` は変わらず、テストだけの差分
  2. C2-2: 「Write the `aria-label` …」を元の位置に戻す、4 面の同じ編集
  3. どちらも直さない場合は、`docs/tech-debt/README.md:252` に (e) として 1 行足す。足さないままこの cycle で `/pr` に進むと、2 件とも記録が残らない
