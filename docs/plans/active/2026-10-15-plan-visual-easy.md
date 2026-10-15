# plan-visual-easy

- Status: Approved
- Approved: 2026-10-15 sha256:3b11a1def777
- Owner: Claude Code
- Date: 2026-10-15
- Related request: /plan の図解ページが難しいので、eli5 のように簡単にしたい(2026-10-15)。案 A(かんたん版を先頭に置き、今の図は折りたたむ)を選び、org-director-start の計画で作った試作を見て「かなり良い」と判断した。決定: (1) かんたん版の図 1 を `#overview` にし、/pr が PR に貼る図もそれにする (2) 人が自分で打つコマンドはかんたん版に書いてよい。関数名・パス・イベント名は書かない (3) 機械で確かめられる範囲は `scripts/plan-visual.sh` で確かめる
- Related issue: N/A
- Type: feat
- Branch: feat/plan-visual-easy

## Objective

承認する人が、図解ページの上半分(かんたん版の図 1〜3 枚と「承認の前に」の欄)だけを読んで、計画を承認するか決められるようにする。今の図(パスと関数名で描いた全体図と詳細図)は消さず、ページの下の折りたたみに移して、仕組みを確かめたい人とエージェントの点検に使う。かんたん版の決まりのうち形で判定できるものは、新しい `scripts/plan-visual.sh lint` が確かめる。

## Scope

- `scripts/plan-visual.sh lint <html>`(と `templates/base/scripts/plan-visual.sh` の同じ写し)。ブラウザを使わず、ページの HTML の文字だけを読む
  - L1: `id="overview"` を持つ要素がちょうど 1 つあり、それが class に `easy` を含む `<figure>` である
  - L2: かんたん版の図(class に `easy` を含む `<figure>`)が 1〜3 枚あり、どれも折りたたみ(`<details class="deep">`)の外にある
  - L3: かんたん版の図の中に、等幅の class `mono`、`<code>`、パスに見える語(英字で始まる語の間の `/`、例 `internal/org`)、ファイルの拡張子で終わる語(`.go`・`.sh`・`.md`・`.json`・`.jsonl`・`.toml`・`.html`・`.yaml`・`.yml`・`.py`・`.ts`・`.js`・`.awk`)、snake_case の識別子(例 `workspace_created`)がない。先頭が `/` の slash コマンド(`/director`)と、`--all` のようなフラグは咎めない(決定 2)
  - L4: かんたん版の図ごとに、箱(class が `n-` で始まる要素)が 5 個以下で、`<h2>` の見出しがあり、`<svg>` に `aria-label` がある
  - L5: 折りたたみの外に「承認の前に」の欄(class に `before` を含む `<section>`)が 1 つある
  - L6: 折りたたみが 1 つあり、その中に `id="detail-overview"` の図がある。かんたん版でない `<figure>` はすべて折りたたみの中にある
  - 属性の値はダブルクォートとシングルクォートのどちらでも読む。タグが複数行にまたがっても読む
  - 出力: 決まりを満たせば何も出さず終了コード 0。満たさなければ、1 件ごとに `<図の id か page>: <L1〜L6>: <中身>` を stdout に 1 行で出し、終了コード 1。引数の誤りやファイルがないときは stderr に理由を出して終了コード 1(今の open・shot・digest と同じ)
  - `--help` と usage に `scripts/plan-visual.sh lint <html>` の行を載せる。古い helper かどうかは、この行があるかで見分ける(下の `/plan` の step 10.c)
- `visual-template.html`(4 面: `.claude/skills/plan/`、`.agents/skills/plan/`、`templates/base/` の 2 つ)を試作の形にする
  - 見出し・計画のパス・リード文(2 文)、かんたん版の図(`<figure class="easy">`、最初が `id="overview"`)、「承認の前に」の欄(承認すると・気をつけてほしいこと・今回は作らないもの)、折りたたみ(中に今の「変わるもの・変わらないもの」、凡例、`id="detail-overview"` の全体図、詳細図、「この図にないもの」)
  - CSS はかんたん版の語彙(`n-new`・`n-old`・`n-fail`・`area`・`line`・人の形)を足す。`#<id>` で 1 枚だけを見せる規則は、かんたん版の図と折りたたみの中の図の両方に効くようにする
  - 雛形そのものが `lint` を通る
- `diagrams.md`(4 面)
  - かんたん版の節: 見出しは結論の文、登場するのは人と役割、パス・関数名・イベント名・等幅の書体は使わない(人が打つコマンドは書いてよい)、避けられない用語は図の下に 1 行の説明、1 枚に箱 5 個まで、色は「新しく増える」と「今のまま」の 2 つと失敗の破線、矢印は 1 種類、基本の形は「今 → これから」、スライスは図の下に「S1・S2 で作る」と添える
  - 「承認の前に」の欄の書き方(スライス、Design decisions と Risks から承認する人が知るべきこと、Non-goals)
  - 今の決まり(Granularity、図の種類、語彙、配置の予算、Grounding)は折りたたみの中の図の決まりとして残す。折りたたみの最初の図は `id="detail-overview"` の全体図
  - 自己点検: `lint` が 0 で終わるまで直す → ページ全体と `--fragment overview` を撮る → 折りたたみの中の図を 1 枚ずつ `--fragment <id>` で撮る(閉じた `<details>` の中の要素へ飛ぶと Chrome が開く)
  - `lint` が確かめることと確かめないこと(見出しが結論の文か、関数名の CamelCase、読みやすさ、重なり)を書き分ける
- `/plan` の SKILL.md(4 面)の step 10.b〜10.d: 新しいページの形、`lint`、撮る図の数、`## Visual review` に書く内容(`lint` の結果を含む)
  - 古い helper の扱い: この変更より前に作った worktree の `scripts/plan-visual.sh` には `lint` がない。skill は main のチェックアウトから読まれるので、step 10.c は、worktree の helper の `--help` に `lint` の行がなければ main のチェックアウトの helper(`$(git rev-parse --path-format=absolute --git-common-dir)/../scripts/plan-visual.sh`)でページの絶対パスを確かめると書く。そちらにもなければ `lint` を飛ばし、`## Visual review` にその理由を書く(承認には進める)
- `/pr` の SKILL.md(4 面)の step 4.a: 撮る図は今のまま `--fragment overview`(かんたん版の図 1)。ページの形は `id="detail-overview"`(またはシングルクォート)があるかで見分ける。あれば新しい形で、diff と照らすのは `detail-overview` とファイルの地図。なければこの変更より前の形で、今のとおり overview を照らす。かんたん版の図は、作ったものが承認時の説明と変わったときに直す
- `internal/org/prompts/leader.md` の機能ごとの org の step 2(per-plan のページ)に、`./scripts/plan-visual.sh lint <ページ>` で決まりを確かめて直すことを足し、`internal/org/prompts_test.go` の文の固定に 1 行足す。leader は手順に書かれたことを打つので、`diagrams.md` を経由するだけでは `lint` を飛ばしうるため
- 文書: AGENTS.md の repo map(`plan-visual.sh` の `open` / `shot` / `lint` / `digest`)、`docs/architecture/repo-map.md`、README の `/plan` の段落、plan の雛形 2 つ(`.claude/skills/plan/template.md` の 4 面、`docs/plans/templates/feature-plan.md` と template 側)の `## Visual review` のコメントに `lint` の結果を足す

## Non-goals

- この変更より前に作った図解ページの描き直し(作業中の org-director-start の計画のページを含む)。`lint` は新しく作るページにだけ求める
- `/spec` の承認に図解ページを足すこと(2026-10-15 の会話で案として出しただけで、頼まれていない)
- 見出しが結論の文になっているか、関数名(CamelCase)を使っていないか、読みやすいか、図の重なりの機械的な判定(PNG の自己点検と人の目に任せる)
- `/director` skill の文言の変更(分割計画のページは「`/plan` skill と同じ手順で」作ると書いてあるので、`lint` を含む新しい手順がそのまま届く)
- ダークモード、ページの公開(ローカルの HTML のまま)

## Assumptions

- Chrome は、閉じた `<details>` の中の要素へ fragment で飛ぶと `<details>` を開く。2026-10-15 に試作で `--fragment detail-overview` を撮って確かめた(図が 1 枚だけきれいに写った)。折りたたみの先頭に印を置いて中の図をまとめて撮る方法は、上半分が白く抜けて使えなかった
- `body:has(figure:target)` の規則は、かんたん版の図を `<figure>` にすれば今のまま効く。試作では `<section>` にしたので規則を足したが、雛形では `<figure class="easy">` にする
- `lint` は POSIX の sh と awk だけで書ける。手元は macOS の awk(20200816)、CI の Ubuntu は mawk なので、gawk だけの機能(`gensub`、正規表現の `RS`、`match` の配列)を使わない
- エージェントが書くページの要素名と class は小文字で、雛形の書き方に従う(`lint` は大文字のタグを読まない)
- skill の本文と雛形は main のチェックアウトから読まれ、helper は worktree の `scripts/` から打たれる(R2 と Codex の計画助言)。main のチェックアウトの skill が新しければ、同じチェックアウトの `scripts/plan-visual.sh` も新しい

## Affected areas

- 変えるファイル: `scripts/plan-visual.sh`、`templates/base/scripts/plan-visual.sh`、`tests/test-plan-visual.sh`、`diagrams.md`・`visual-template.html`・`SKILL.md`・`template.md`(`/plan` の 4 面)、`SKILL.md`(`/pr` の 4 面)、`internal/org/prompts/leader.md`、`internal/org/prompts_test.go`、`AGENTS.md`、`docs/architecture/repo-map.md`、`README.md`、`docs/plans/templates/feature-plan.md`、`templates/base/docs/plans/templates/feature-plan.md`
- 新しいファイル: なし(`lint` は `plan-visual.sh` の中に書くので、`scripts/check-template.sh` と `internal/scaffold/embed_test.go` のファイル一覧は変わらない)

## Visual review

- 図解ページ: `.harness/state/plan-visual/plan-visual-easy.html`。この計画が決める新しい形で描いた(かんたん版 3 枚: `#overview`・`#easy-check`・`#easy-pr`、「承認の前に」、折りたたみの中に詳細 3 枚: `#detail-overview` 変わるファイル・`#detail-page` ページの並び・`#detail-helper` 古い helper のとき)
- 自己点検: 全体、`--fragment overview`(幅 1150、高さ 500、scale 2)、残りの 5 枚を 1 枚ずつ `--fragment` で撮って読んだ。重なりと、線がラベルを横切る所はない。詳細 3 の等幅の注は文字幅の見積もりで箱を超えたので、撮る前に普通の書体の短い文に直した。全体図の PNG は下の余白が少ないので、/pr で撮るときは高さを 540 にする。`lint` はまだないので、S1 のあとにこのページへかける(AC9)

## Design decisions

- D1 かんたん版の図は `<figure class="easy">` にする。試作の `<section>` をやめるのは、`#<id>` で 1 枚だけを見せる今の CSS と、/pr の `--fragment overview` をそのまま使うため
- D2 `#overview` はかんたん版の図 1 で、/pr が PR に貼る図もそれになる(ユーザーの決定 1)。/pr が diff と照らす図は、パスを描いた図に移す。前の形のページ(作業中の計画)は overview にパスがあるので、/pr の文は両方の形に効くように書く
- D3 かんたん版には人が自分で打つコマンドを書いてよく、関数名・パス・イベント名は書かない(ユーザーの決定 2)。機械で判定するのは、パス・拡張子・snake_case・等幅・`<code>` まで。CamelCase の関数名は `GitHub` のような普通の語と見分けられないので、`diagrams.md` の文で求めるだけにする
- D4 `lint` は `plan-visual.sh` の中の awk で書き、別のファイルにしない。雛形に入れるファイルの一覧(`check-template.sh`、`embed_test.go`)を変えずに済み、`digest` と同じ置き方になる
- D5 折りたたみの中の図は、自己点検で 1 枚ずつ `--fragment <id>` で撮る(上の Assumptions)。今は全体を 1 枚撮れば詳細図も写っていたが、閉じた折りたたみは全体の 1 枚に写らないため
- D6 「承認の前に」の欄を必ず置く。今のページには、Design decisions にある「承認する人が知るべき危うさ」(例: 8a-2 の director は既定で確認なしにコマンドを打てる)を見せる場所がない。試作でこの欄が役に立った

- D7 古い helper の worktree では、main のチェックアウトの helper で `lint` を打つ。Codex の計画助言 1(MEDIUM)。この変更より前に作った worktree で計画を直すと、新しい skill が `lint` を求めるのに worktree の helper にはない。見分けは `--help` の `lint` の行で、`lint` を打って終了コードを見る方法は、古い helper の「知らないサブコマンド」(1)と決まり違反(1)を区別できないので退けた。どちらの helper にもなければ `lint` を飛ばして理由を書き、承認には進める(ページの自己点検の PNG は残る)
- D8 /pr はページの形を `detail-overview` の有無で見分ける。consult(consult-plan-visualeasy)の指摘 1。作業中の org-director-start の /pr は、この変更の merge 後に新しい /pr の文を読む(R2)ので、形の判定を文に任せず、機械的な条件で書く

Critical forks: ユーザーが 2026-10-15 に決めた 3 点(Related request の (1)〜(3))のほかにない。D1、D4〜D8 は試作と今の仕組みから決まる既定の選び方と、consult と Codex の計画助言の指摘を受けた直し。

## Acceptance criteria

- [x] AC1 `scripts/plan-visual.sh lint <html>` が L1〜L6 を確かめる。決まりを満たすページでは何も出さずに 0 で終わり、満たさないページでは 1 件ごとに `<id>: <規則>: <中身>` を 1 行出して 1 で終わる。引数がない、ファイルがないときは stderr に理由を出して 1 で終わる。`--help` と usage に `lint` が載る
- [x] AC2 `tests/test-plan-visual.sh` に、決まりを満たすページ(雛形の形)が通る場合と、L1〜L6 をそれぞれ 1 つだけ破るページが、その規則の 1 行を出して 1 で終わる場合がある。端の場合: `id` と `class` の並びが逆、属性がシングルクォート、タグが複数行にまたがる、`&lt;slug&gt;.html` は拡張子で咎める、`S1/S2` はパスの形で咎める、`/director` と `ralph org stop --all` と `1/3` は咎めない、前の形のページ(overview が easy でない)は L1 で咎める
- [x] AC3 `visual-template.html` の 4 面が新しい形で、雛形そのものが `lint` を通る(テストで確かめる)。`#overview` で開くとかんたん版の図 1 だけが見え、`#detail-overview` で開くと折りたたみの中の全体図だけが見える(撮って確かめる)
- [x] AC4 `diagrams.md` の 4 面に、かんたん版の決まり、「承認の前に」の欄、折りたたみの中の図の決まり、`lint` を含む自己点検、`lint` が確かめないことが書いてある
- [x] AC5 `/plan` の SKILL.md の 4 面の step 10 が、新しい形、`lint` が 0 で終わるまで直すこと、撮る図(全体、overview、折りたたみの中の図を 1 枚ずつ)、`## Visual review` に `lint` の結果を書くことを求める。worktree の helper の `--help` に `lint` の行がなければ main のチェックアウトの helper で打ち、どちらにもなければ飛ばして理由を書く(D7)。テストは `--help` に `scripts/plan-visual.sh lint <html>` の行があることを固定する
- [x] AC6 `/pr` の SKILL.md の 4 面の step 4.a が、overview(かんたん版の図 1)を撮って貼り、ページに `id="detail-overview"`(またはシングルクォート)があれば `detail-overview` とファイルの地図を、なければ overview を diff と照らすとする(D8)
- [x] AC10 `internal/org/prompts/leader.md` の step 2 が `./scripts/plan-visual.sh lint <ページ>` で確かめて直すことを書き、`internal/org/prompts_test.go` がその文を固定する。`go test ./internal/org/...` が手元の Go と `GOTOOLCHAIN=go1.25.8` の両方で通る
- [x] AC7 AGENTS.md、`docs/architecture/repo-map.md`、README、plan の雛形のコメントが `lint` と新しい形を説明し、`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る
- [x] AC8 `open`・`shot`・`digest` の動きは変わらず、今あるテストはそのまま通る
- [x] AC9 この計画自身の図解ページが新しい形で書かれ、S1 のあとで `lint` を通る(dogfood)

## Implementation outline

1. S1 `plan-visual.sh lint`(root と template の写し)と `tests/test-plan-visual.sh` の場合。この計画のページに `lint` をかける
2. S2 `visual-template.html` と `diagrams.md` の 4 面。雛形が `lint` を通るテスト
3. S3 `/plan` と `/pr` の SKILL.md の 4 面、`leader.md` とその文の固定、plan の雛形のコメント、AGENTS.md、repo-map、README

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(shellcheck は `scripts/*.sh` のグロブで `plan-visual.sh` を含む)、`scripts/check-skill-sync.sh`、`scripts/check-sync.sh`、`scripts/check-template.sh`、`gofmt -l`、`go vet ./...`、`GOTOOLCHAIN=go1.25.8 go build ./... && go vet ./...`
- Spec compliance criteria to confirm: ユーザーの決定 3 点が D2・D3・AC1 に入っているか。L1〜L6 が `diagrams.md` の文と食い違っていないか
- Documentation drift to check: `plan-visual.sh` のサブコマンドを並べた箇所(AGENTS.md、repo-map、usage)、`/pr` の「Match the figure to the diff」、`diagrams.md` の Page structure と Self-check
- Evidence to capture: 雛形とこの計画のページの `lint` の出力、雛形を `#overview` と `#detail-overview` で撮った PNG を読んだ結果

## Test plan

- Unit tests: `tests/test-plan-visual.sh` に `lint` の場合(AC2)。fixture はテストの中の heredoc で書く(今の `write_plan` と同じ)
- Integration tests: 雛形の 4 面を `lint` にかける。`./scripts/run-test.sh`
- Regression tests: `open`・`shot`・`digest` の今のテストがそのまま通る
- Edge cases: AC2 の端の場合。空のファイル、`<figure>` が閉じていないページは L1 か L6 で咎める(落ちずに終わる)
  - 途中の図の閉じ忘れ: 図が開いている間に次の `<figure` が来たら、L6 で咎める(cross-review cycle 1 の指摘 1)
  - インラインのタグをまたぐ語: `internal/<tspan>org</tspan>` と、`tspan` で分けた snake_case は L3 で咎める。`x` か `y` を持つ `tspan` と、別の `<text>` どうし(`<text>run</text><text>/director</text>`)は語を区切るので咎めない(cross-review cycle 1 の指摘 2)
- Evidence to capture: `docs/evidence/` に `lint` の場合の出力

## Risks and mitigations

- R1 `lint` が普通の言葉を咎める(例 `and/or`)。咎める形をパス・拡張子・snake_case に絞り、出力に咎めた語を出すので、書く側は言い換えられる。咎めない語をテストに入れる(AC2)
- R2 作業中の計画(org-director-start)の /pr が、merge 後の新しい /pr の文を読む(skill は main のチェックアウトから読まれる)。前の形のページでも overview を撮れ、/pr の文は前の形のページでは overview を diff と照らすと書く(D2、AC6)
- R3 Chrome でないブラウザ、または将来の Chrome が閉じた `<details>` を開かないと、詳細図の PNG が空になる。自己点検で PNG を読むので空の図は見つかる。そのときは `diagrams.md` の手順どおり `<details>` に一時的に `open` を付けて撮る
- R4 4 面の写しのずれ。`check-skill-sync.sh` と `check-sync.sh` が止める
- R5 awk の方言で `lint` の結果が変わる。POSIX の機能だけを使い、CI(Ubuntu、mawk)でもテストが走る(`scripts/verify.local.sh` が `tests/test-*.sh` をすべて走らせ、`.github/workflows/verify.yml` が ubuntu-latest で `run-verify.sh` を打つ)
- R6 古い helper の worktree で計画を直すと、新しい skill が求める `lint` がない(Codex の計画助言 1)。D7 の手順で main のチェックアウトの helper を使い、それもなければ飛ばして理由を書く

## Rollout or rollback notes

- `ralph upgrade` で下流の雛形(core の skill と script)が新しい形になる。seed の `docs/plans/templates/feature-plan.md` は upgrade で上書きされないが、コメントだけの差で動きは変わらない
- 戻すときはこの PR を revert する。新しい形で作ったページは、前の /pr でも `#overview` を撮れる

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created

進捗メモ:
- 2026-10-15 計画の承認: 2026-10-11 のユーザーの指示(作った計画は自動承認、承認待ちは要らない)に従って自動で承認した。consult(consult-plan-visualeasy)も同じ判定。Codex の計画助言(MEDIUM 1 件、古い helper の worktree)は D7・AC5・R6 に入れた。consult の 3 点は D8・AC2・AC6・AC10 に入れた
- 2026-10-15 S1(821a3c69、implementer): `plan-visual.sh lint` と 89 件のテスト(全体 193 件 pass)。契約の外で決めたこと: `<!-- -->` は 1 つのタグとして読み飛ばす(雛形のコメントの例で L1 が 2 件にならないように)、`<style>` と `<script>` の中身は図の文字に入れない(`font: 13px/1.4` が L3 に掛かるため)、引用符の中の `>` でタグを終えない、空白だけの aria-label は空とみなす、L3 は一致した語を前後に広げて出す(`internal/org`)。Docker の ubuntu 24.04 で mawk 1.3.4 と gawk 5.2.1 でも同じ結果。この計画のページは lint rc=0(AC9)、前の形の org-director-start のページは L1・L2・L5・L6 で咎められた。範囲外で見つけたこと: `digest` は `awk ... "$1"` で読むので、`a=b.md` のような `=` を含む相対名だと stdin を読みにいく(昔からの挙動、tech-debt に書く)
- 2026-10-15 S2(5b056944、implementer): 雛形と `diagrams.md` の 4 面、雛形の 4 面に `lint` をかけるテスト(全体 201 件 pass)。雛形は 4 面とも `lint` rc=0 で、全体・`#overview`・`#detail-overview` の PNG を読んで崩れがないことを確かめた(AC3)。契約の外で決めたこと: `<h1>` の置き場 `__HEADLINE__` を `<title>` の `__TITLE__` と分けた、折りたたみの詳細図の例の id を `detail-1` にした、Credit に eli5 を足した。`diagrams.md` の自己点検は、D5 の折りたたみの中の図に加えて、かんたん版の図 2・3 も 1 枚ずつ撮ると書いた(全体の 1 枚にも写るが、1 枚ずつのほうが崩れを見つけやすい。D5 より多く撮るだけで、D5 と食い違わない)。consult の指摘 2 点は S3 で扱う: /pr の `<fit>` を `diagrams.md` と同じ +200 にそろえる、かんたん版の決まり 2 の例 `director start` はまだないコマンドなので、今あるコマンドに替える
- 2026-10-15 S3(138481e0、implementer): `/plan` の step 10.b〜10.d と 12.b、`/pr` の step 4.a(ページの形の見分けと `<fit>` の +200)、leader.md の step 2 の `lint` と固定の 3 行、plan の雛形のコメント、`diagrams.md` の決まり 2 の例(`/director`、`ralph org stop --all`)、AGENTS.md・repo-map・README、tech-debt の `digest` の行。`.ralph/core/AGENTS.core.md` と `templates/base/AGENTS.md` には helper の一覧がないので変えていない。決めたこと: 10.c は Lint と Self-check の 2 段にし、`shot` が 2 で終わったときに飛ばすのは撮るほうだけ(lint はブラウザなしで回る)。leader には main のチェックアウトの helper の逃げ道を書かない(leader の worktree と main は同じ scaffold)。`go test ./internal/org/...` は手元の Go と `GOTOOLCHAIN=go1.25.8` で ok、`go test -overlay` で 変更前の leader.md に差し替えると新しい 3 行が FAIL することも確かめた。S3 のあとでこの計画のページに `lint` をかけ直して rc=0。範囲外で気づいたこと: `/pr` の 4.a で図を直したあとに `lint` を打つ指示はない(4.a は `/plan` の 10.c と同じ自己点検と書くので、撮る前の点検は読める)
- 2026-10-15 cross-review cycle 1(fb7648b0 の差分、codex): ACTION_REQUIRED 2 件(途中の図の閉じ忘れを通す、`tspan` で分けたパスを通す)。consult(consult-plan-xreviewfix)も同じ判定。直す前に Test plan の Edge cases に 2 行を足したので digest が 819a5be1cf1f から 3b11a1def777 に変わり、2026-10-11 の指示(作った計画は自動承認)に従って `- Approved:` を書き直した。/plan の step 10 は、図解ページの図がこの 2 つの場合を描いていないので描き直さず、`lint` をかけ直して rc=0 を確かめた。直したあとは cycle 2 として /self-review から回し直す
