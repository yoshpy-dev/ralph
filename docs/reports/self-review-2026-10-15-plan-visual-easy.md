# Self-review report: plan-visual-easy

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-plan-visual-easy.md(承認済み、digest 819a5be1cf1f)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)
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
