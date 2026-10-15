# Walkthrough: plan-visual-easy(/plan の図解ページをかんたん版の図から始める)

- Date: 2026-10-15
- PR: https://github.com/yoshpy-dev/ralph/pull/230
- Plan: docs/plans/archive/2026-10-15-plan-visual-easy.md
- Branch: feat/plan-visual-easy(base 9cbf7080。この walkthrough を書いた時点で 23 commits、39 files、+4040/-609)
- 読む範囲: 報告・evidence・insight・計画を除くと 30 files、+2718/-608。そのうち 4 面の写し(`.agents/skills/`、`templates/base/` の 2 つ)と helper の写し(`templates/base/scripts/plan-visual.sh`)は元とバイト一致なので、読むのは `.claude/skills/` と `scripts/` の側だけでよい。中身のあるファイルは `scripts/plan-visual.sh`(+341/-3)、`tests/test-plan-visual.sh`(+376/-1)、`.claude/skills/plan/visual-template.html`(+243/-96)、`.claude/skills/plan/diagrams.md`(+158/-46)と、数行ずつの手順と文書
- 後工程は 2 回まわした(上限 2)。cross-review の指摘は cycle 1 が 2 件(下の 6)、cycle 2 は 0 件

次の順に読むと、ページの形 → それを確かめる lint → lint を打つ手順、の順に追える。

## 1. ページの形(`.claude/skills/plan/visual-template.html`、5b056944、のちに 0067b837)

- 並びは、見出しとリード文 → かんたん版の図(`<figure class="easy">`、1 枚目が `id="overview"`)→ 「承認の前に」の欄(`<section class="before chrome">`)→ 閉じた折りたたみ(`<details class="deep">`)。折りたたみの中に、今までの「変わるもの・変わらないもの」、凡例、`id="detail-overview"` の全体図、詳細図、「この図にないもの」を置く
- かんたん版の語彙は CSS の `n-new`・`n-old`・`n-fail`、`area`、`line`、人の形、行の見出し `t-row`、丸い番号 `no`、仕切り `rule`
- `#<id>` で 1 枚だけを見せる規則(`body:has(figure:target) …`)は、かんたん版の図にも折りたたみの中の図にも効く。閉じた `<details>` の中の図へ飛ぶと Chrome が開くので、`--fragment detail-overview` でその図だけが撮れる
- 0067b837 で、使われていない `.easy svg .t-head` を消し、1 字違いだった置き場の名前を `__NOT_IN_PLAN__` と `__NOT_IN_PAGE__` に変えた(self-review L4)

## 2. 書き方の決まり(`.claude/skills/plan/diagrams.md`、5b056944、のちに 0067b837・52c9db31・55cce2b7)

- かんたん版の決まり 7 つ: 見出しは結論の文、登場するのは人と役割、パス・関数名・イベント名・等幅の書体は使わない(人が打つコマンドは書いてよい。例は `/director` と `ralph org stop --all`)、避けられない用語は図の下に 1 行、箱は 5 個まで、スライスは図の下に添える、計画にないことは書かない
- 「承認の前に」の 3 列(承認すると始まること、気をつけてほしいこと、今回は作らないもの)
- 今までの決まり(図の種類、語彙、配置の予算、Grounding)は、折りたたみの中の図の決まりとして残した
- 自己点検は lint → 全体・overview・残りの図を 1 枚ずつ撮る → PNG を読む。古い worktree の helper に lint がなければ main のチェックアウトの helper で打ち、そちらにもなければ飛ばして理由を書く(計画の D7)
- 「What `lint` checks」と「does not check」を書き分けた。L3 が読むのは図の文字、`class`、`<code>` で、`aria-label` は読まない(self-review L1)

## 3. lint(`scripts/plan-visual.sh` の `lint_page` と `cmd_lint`、821a3c69、のちに 0067b837・52c9db31)

- ページを `< "$1"` で読み(`a=b.html` が awk の変数代入にならない)、awk の 1 本でタグを読む。`<!-- -->`、`<style>`、`<script>` の中身は飛ばし、引用符の中の `>` でタグを終えない。複数行のタグ、シングルクォートも読む
- 規則 L1〜L6 は計画の Scope のとおり。違反は 1 件ごとに `<図の id か page か figure#N>: L<n>: <中身>` を stdout に出し、exit 1。L3 は咎めた語を出す(`path-like text: internal/org`)
- 52c9db31(cross-review cycle 1 の 2 件): 図が開いている間に次の `<figure` が来たら L6 を 1 行出し、開いていた図を調べてから閉じる。L3 は `tspan`・`a`・`b`・`i`・`em`・`strong`・`span`・`small`・`sub`・`sup` をまたいで文字をつなぐ(`tag_splits`)。`x` か `y` を持つ `tspan`(`HAS_XY`)、`<br>`、`<code>`、コメント、別の `<text>` では区切るので、`<text>run</text><text>/director</text>` は `run/director` にならない
- POSIX の awk だけを使う(`gensub`、`match` の配列、正規表現の `RS`、区間表現を使わない)。macOS の awk 20200816、mawk 1.3.4、gawk 5.2.1 で同じ出力になることを verify と test が Docker で確かめた
- `open`・`shot`・`digest` の本体は変えていない(AC8。origin/main の suite 103 件が新しい helper で通る)

## 4. テスト(`tests/test-plan-visual.sh`、821a3c69・5b056944・52c9db31・55cce2b7)

- 1 つの正しいページを部品に分け、規則ごとに 1 か所だけ壊して `assert_only_rule` でほかの規則が出ないことまで固定する形。suite は 243 件
- 雛形の 4 面がそのまま lint を通ること、`--help` に `  scripts/plan-visual.sh lint <html>` の行があること(古い helper の見分けに使う文字列)
- 55cce2b7 で、入れ子の開始で閉じた図も L3 で調べられることを固定した(self-review C2-1。`close_figures()` を調べずに閉じる mutant が落ちる)
- 生き残った mutant 10 個とテストのないもの(template 側の helper を suite が打たない、skill の文だけの手順、Chrome でしか見ていない `<details>`)は `docs/tech-debt/README.md` の行にした

## 5. lint を打つ手順(`/plan` と `/pr` の SKILL.md、leader.md、138481e0、のちに 0067b837)

- `/plan` の step 10.b〜10.d: ページの並び、Lint(0 で終わるまで直す。古い helper の逃げ道)と Self-check(撮る図、`shot` が 2 で終わっても lint は回す)、`## Visual review` に lint の結果を書く。12.b の要約にも lint を飛ばしたかを入れる
- `/pr` の step 4.a: ページの形を `id="detail-overview"` の有無で見分ける。新しい形は折りたたみの全体図とファイルの地図を diff と照らし、かんたん版がずれていれば平易な言葉で直し、直したら lint。前の形は今までどおり overview を照らし、lint は打たない。`<fit>` は viewBox + 200(0067b837 で 3 つの子の箇条に分けた。self-review M1)
- `internal/org/prompts/leader.md` の step 2: 機能の計画のページにも lint をかけ、helper に lint がなければ飛ばして QUESTION に書く。`internal/org/prompts_test.go` に 3 行の固定を足した

## 6. cross-review cycle 1 の直し(52c9db31)

- Codex の P2 その 1: `</figure>` を 1 つ読むと開いている図がすべて閉じるので、`detail-overview` の `</figure>` が抜けても exit 0 だった。ブラウザでは次の図がその中に入り、`#detail-1` の PNG が白くなる
- Codex の P2 その 2: L3 はタグの境目に空白を入れていたので、`internal/<tspan>org</tspan>` が通っていた
- 直す前に計画の Test plan の Edge cases に 2 行を足し、承認の digest を 3b11a1def777 に記録し直した。直し方は 3 の 3 つ目の項

## 7. 文書(138481e0、fb7648b0、648bdd27)

- AGENTS.md の repo map と `docs/architecture/repo-map.md` に `lint`、README の /plan の段落にページの形と lint、plan の雛形 3 つのコメントに lint の結果
- `docs/tech-debt/README.md`: leader の図解ページの行に lint を足した。`digest` が `a=b.md` のような名前で stdin を読む件(昔からの挙動)と、lint のテストの穴の行を足した

## 残っていること

- `/plan` の 10.c の古い helper の扱いと、`/pr` の 4.a の形の見分けは、skill の文だけで動かして確かめていない。merge 後の最初の実地は、前の形のページを持つ org-director-start の /pr(4.a の前の形の分岐)
- self-review L2(leader.md と diagrams.md の、古い helper のときの書き方の違い)は残した。leader が読む diagrams.md と helper は同じ worktree から来る
- 計画の `## Visual review` は「lint はまだない」の記述のまま archive される。結果は Progress checklist にある
