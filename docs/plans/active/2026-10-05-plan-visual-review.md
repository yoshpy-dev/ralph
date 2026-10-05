# plan-visual-review

- Status: Approved
- Approved: 2026-10-05 sha256:9c20a2da6606
- Owner: Claude Code
- Date: 2026-10-05
- Related request: ralph の spec か plan に、承認の前に計画を図示してから承認を求める仕組みを入れたい(2026-10-05 ユーザー依頼。参考: https://github.com/nntto/skills/tree/main/skills/explanatory-diagrams)
- Related issue: N/A
- Type: feat
- Branch: feat/plan-visual-review

## Objective

`/plan` が承認を求める前に、実装予定の構成や流れを図にした HTML ページ(以下、図解ページ)を作ってブラウザで見せ、Approve / Needs changes の承認ゲートを通すようにする。PR を作るときは、図解ページの全体図を画像にして PR 本文に載せる。

図解ページの目的は記録ではなく、エージェントが何を実装するのかを人がつかめるようにすること。見やすさとわかりやすさを優先し、plan の md にテキストで図を残すことは求めない(ユーザー、2026-10-05)。

## Scope

- `/plan` に 3 つの手順を足す。図解ページを作る、見た目を自己チェックする、承認ゲートを通す。承認されたら plan の `Status: Draft` を `Approved` に書き換え、`- Approved:` の行に日付と plan の digest を書く
- 図解の手引き `.claude/skills/plan/diagrams.md`(新規)。粒度の規則、表す対象から図の種類を選ぶ表、図解を省ける plan の条件、色と線の意味、レイアウトの数値の目安、図のノードとパスの対応
- 図解ページの雛形 `.claude/skills/plan/visual-template.html`(新規)。色のトークン、凡例、図の枠、全体図だけを表示する切り替え
- `scripts/plan-visual.sh`(新規)。`open`(ブラウザで開く。開けない環境ではパスを出す)、`shot`(ヘッドレス Chrome / Chromium で PNG を撮る。ブラウザがなければ exit 2)、`digest`(plan の承認対象の部分から sha256 を出す)の 3 つのサブコマンド
- plan テンプレートに `- Approved:` の行と `## Visual review` 節を足す。Visual review には図解ページのパスか、`なし(理由)` を書く
- `/implement` が plan の `Status` と digest を確かめる。`Approved` でないとき、または承認後に plan が変わって digest が合わないときは、図解と承認をやり直すか、そのまま続けるかをユーザーに聞く
- `/pr` が全体図の PNG を `gh pr create --attach` で PR 本文に載せる。載せる前に図と変更ファイルの一覧を照らし合わせる。gh に `--attach` がないときは図を省き、理由を本文に 1 行書く。添付つきの作成が非ゼロで終わったときは、PR ができたかを確かめてから直す
- 配布物の登録: `.agents/skills/`(`scripts/sync-skills.sh` で再生成)、`templates/base/` 側のスキル・スクリプト・rules・plan テンプレート、`scripts/check-template.sh`(root と template)と `internal/scaffold/embed_test.go` の必須ファイル一覧、`scripts/verify.local.sh` の shellcheck 対象一覧
- ドキュメント: `subagent-policy.md` の「Planning — always inline」節、`AGENTS.md` と `.ralph/core/AGENTS.core.md` の Primary loop、`README.md` の Plan の説明、`ralph-workflow.md` の `/plan` の行(root と `templates/base/` の両側)

## Non-goals

- `/spec` への導入(ユーザー確定。要件の承認は今の step 5 のまま)
- 実装中に図を更新すること、`/sync-docs` で図とコードのずれを調べること
- Codex advisory に図を見せること、図と本文の食い違いを Codex に確かめさせること
- 図を plan の md にテキスト(Mermaid など)で残すこと。図解ページと PNG はコミットしない
- draw.io・Mermaid・D2 などの描画ツールへの依存
- 参照先 skill の文章やテンプレートの複製。参照先のリポジトリにはライセンスファイルがない(GitHub API の license が 404)ので、考え方だけを借りてリンクを残す
- org runtime(headless leader)向けの非対話モード。`internal/org/` と `.claude/skills/org/SKILL.md` には `/plan` を呼ぶ経路がない(grep で確認、`watch.go:646` の一致はコメント)
- gh の自動更新。古い gh では添付を省くだけにする
- リリース(`/release` は手動)

## Assumptions

- `.harness/state/` は root と `templates/base/` の両方で gitignore 済み(`.gitignore:47`)。図解ページと PNG を `.harness/state/plan-visual/` に置けばコミットされない。task worktree ごとに別の場所になる
- Claude Code は Read で PNG を見られる(このセッションで試作のスクリーンショットを読めた)。Codex CLI が PNG を読めるかは未確認。読めない場合は自己チェックを省き、省いたことを `## Visual review` に書く
- gh v2.99.0(2026-09-01)以上は `gh pr create` / `gh pr edit` に `--attach` を持つ。本文中のローカルパスへの参照はアップロード先の URL に置き換わる(リリースノートで確認)。手元は 2.96.0 なので、S5 の実地確認の前に `brew upgrade gh` が要る。private repo と EMU アカウントでの挙動は未確認
- ヘッドレス Chrome の CLI(`--screenshot`)はウィンドウ全体を撮り、要素単位では撮れない。全体図だけを撮るときは、ページ側で他の図を隠し、ウィンドウの高さを図に合わせる
- `ralph upgrade` は、テンプレートに新しく現れた core ファイルを下流に作る(`internal/upgrade/replaceplan.go` の `classifyCore`)。新規の `diagrams.md`・`visual-template.html`・`plan-visual.sh` は upgrade で下流に届く
- `tests/test-*.sh` は `scripts/verify.local.sh:210` が列挙してすべて走らせる。新しいテストを一覧に足す必要はない

## Affected areas

root 側:
- `.claude/skills/plan/SKILL.md`、`template.md`、`diagrams.md`(新規)、`visual-template.html`(新規)と、`.agents/skills/plan/` の同名ミラー
- `.claude/skills/implement/SKILL.md` と `.agents/skills/implement/SKILL.md`
- `.claude/skills/pr/SKILL.md`、`template.md` と `.agents/skills/pr/` の同名ミラー
- `docs/plans/templates/feature-plan.md`(`scripts/new-feature-plan.sh` が使うテンプレート。`.claude/skills/plan/template.md` と同じ内容)
- `scripts/plan-visual.sh`(新規)、`scripts/check-template.sh`(必須一覧)、`scripts/verify.local.sh`(shellcheck 対象一覧)
- `internal/scaffold/embed_test.go`(必須一覧)
- `tests/test-plan-visual.sh`(新規)
- `.claude/rules/ralph/subagent-policy.md`、`.claude/rules/ralph/ralph-workflow.md`
- `AGENTS.md`、`.ralph/core/AGENTS.core.md`、`README.md`

`templates/base/` 側:
- `.claude/skills/{plan,implement,pr}/` と `.agents/skills/{plan,implement,pr}/` の上記と同じファイル
- `scripts/plan-visual.sh`、`scripts/check-template.sh`
- `docs/plans/templates/feature-plan.md`
- `.claude/rules/ralph/subagent-policy.md`、`.claude/rules/ralph/ralph-workflow.md`
- `AGENTS.md`、`.ralph/core/AGENTS.core.md`(`README.md` と `scripts/verify.local.sh` は template 側にない)

## Visual review

- 図解ページ: `.harness/state/plan-visual/plan-visual-review.html`(この plan 自体で新しい手順を試す)
- 自己チェック: ヘッドレス Chrome で全体と `#overview` を撮って確認(2 回。矢印とバッジの重なり 1 か所を直した)
- digest の計算方法(S1 の `plan-visual.sh digest` はこれと同じ値を出す): `- Status:`・`- Approved:`・`- Branch:` で始まる行と、`## Progress checklist` の行から次の `## ` 見出しの手前(なければ末尾)までを除き、残りの行を改行つきでつないだ UTF-8 のバイト列の sha256 の先頭 12 桁

## Design decisions

- **導入先: `/plan` だけ(ユーザー確定)**。見たいのは実装予定の構成(How)で、`/spec` の役割表では `/plan` の担当になる。`/spec` は要求が曖昧なときしか呼ばれないので、spec に入れると図を見ずに実装へ進むタスクが多く残る。`/spec` から `/plan` へ引き継ぐ場合は承認が 2 回になるが、要件の承認と設計の承認で見る対象が違うので許容する(consult)
- **形式: エージェントが書く単体の HTML + inline SVG(ユーザー確定)**。外部ライブラリと CDN は使わず、オフラインで開ける。ユーザーが「記録より視認性」と明言し、試作を見て「この方向でよい」と答えた。採らなかった案: plan の md に Mermaid(記録には向くが見た目の自由度が低い)、draw.io の PNG(アプリへの依存が配布先に入る。手元にも入っていない)、ASCII(表現力が低い)、Claude Code の Artifact(Claude 専用で Codex とそろわず、claude.ai に送ることになる)
- **粒度: 全体図 1 枚 + 問いごとの詳細図(ユーザー確定)**。全体図には全機能を載せ、slice 番号か AC 番号を振る。詳細図は変更の種類で決める(やり取り → シーケンス図、状態 → 状態遷移図、データ構造 → 構造図、モジュール境界 → 変更前後の比較)。各図に「確かめてほしいこと」を 1 行添える。詳細図が 4 枚を超えたら、図を足さずに plan の分割を提案する
- **図解を省ける条件(consult の指摘で列挙式にした)**: 次のどれかに当たり、かつ、やり取り・状態・データ形式・モジュール境界のどれも変えない plan だけが省ける。(1) ドキュメントだけの変更、(2) 1 ファイル(とその自動生成のミラー)だけの変更、(3) 名前や設定値だけを変える機械的な変更。省くときは `## Visual review` に `なし(理由)` を書く
- **手順の順序**: 草案 → critical forks → 図解ページ → 自己チェック → Codex advisory → 承認。Codex の指摘で plan を直したら、図も作り直してから承認に進む。Codex の指摘への質問と承認の質問は別のまま並べる(consult の既定)
- **承認の記録と失効(Codex advisory の指摘 1、ユーザー確定)**: 承認されたら `Status: Approved` に書き換え、`- Approved: <日付> sha256:<先頭 12 桁>` を書く。digest は plan 全体から `- Status:`・`- Approved:`・`- Branch:` の行と `## Progress checklist` 節を除いた部分で計算する(`plan-visual.sh digest`)。`/implement` は `Approved` でないとき、または digest が合わないときに、図解と承認をやり直すか続けるかを聞く。止めはしない(この変更より前の Draft の plan でも進められるように)。採らなかった案: SKILL.md の規則だけで済ませる(軽いが、守られたかを機械的に確かめられない)
- **レイアウトの数値の目安(consult の指摘)**: 試作でも文字詰まりが 1 か所出たので、手引きに数値を書く。viewBox の幅は 1080、文字は 12px 以上、ノードの幅は「全角の文字数 × 文字サイズ + 24px」以上、1 枚の図のノードは 10 個まで
- **全体図だけの PNG**: 雛形に `#overview` のフラグメントで全体図以外を隠す CSS を入れ、`plan-visual.sh shot` にフラグメントとウィンドウの大きさを渡して撮る(consult の既定)
- **PR への添付は同じ plan の最後の slice(ユーザー確定)**。`gh pr create --help` に `--attach` がなければ省く。添付つきの `gh pr create` が非ゼロで終わっても、PR が作られている場合がある(gh v2.99.0 の help「If some attachments upload and others fail, the pull request is still created ... the new pull request's URL is still printed to stdout」と `create.go` の `submitPR`。Codex advisory の指摘 2)。そこで終了コードと stdout を残し、stdout の URL か `gh pr view <head branch>` で PR があるかを確かめる。あれば `gh pr edit` で本文を直し、なければ添付なしで作り直す。どちらの場合も理由を本文に 1 行書く
- Critical forks: None。残った分岐(PNG の撮り方、質問の分け方、`/implement` の確認の強さ)は、どれも 1 slice 以内でやり直せるので既定で決めた

## Acceptance criteria

- [ ] AC1: `scripts/plan-visual.sh open <html>` は、ファイルがあれば OS のオープナー(macOS は `open`、それ以外は `xdg-open`、環境変数で差し替え可)で開いて exit 0。オープナーがなければ、開くべきパスを stdout に出して exit 0。ファイルがなければ exit 1
- [ ] AC2: `scripts/plan-visual.sh shot <html> <png>` は、Chrome / Chromium を見つけたら PNG を書いて exit 0。フラグメントとウィンドウの幅・高さを指定できる。ブラウザが見つからなければ、その旨を stderr に出して exit 2。環境変数でブラウザの実行ファイルを指定できる
- [ ] AC2b: `scripts/plan-visual.sh digest <plan>` は、`- Status:`・`- Approved:`・`- Branch:` の行と `## Progress checklist` 節を除いた内容の sha256 の先頭 12 桁を stdout に出す。除いた部分だけが違う 2 つの plan は同じ値になり、それ以外の行が 1 文字でも違えば別の値になる。macOS(`shasum`)と Linux(`sha256sum`)で同じ値を出す
- [ ] AC3: `tests/test-plan-visual.sh` が、ファイルなし、オープナーあり / なし、ブラウザなし(exit 2)、スタブのブラウザに渡る引数(ファイル URL・フラグメント・ウィンドウの大きさ)、空白を含むパス、digest の一致と不一致(AC2b の両方向)を確かめ、`./scripts/run-test.sh` で通る
- [ ] AC4: `scripts/plan-visual.sh` が `templates/base/scripts/` にもあり、`scripts/check-template.sh`(root と template)と `internal/scaffold/embed_test.go` の必須一覧に載っている。`./scripts/check-sync.sh` と `go test ./internal/scaffold/...` が通る
- [ ] AC5: `.claude/skills/plan/diagrams.md` が次をすべて含む。図解の目的(見やすさ優先)、粒度の規則、表す対象から図の種類を選ぶ表、図解を省ける条件、色と線の意味、レイアウトの数値の目安、ノードを実在のパスか新規作成予定のパスに対応させる規則、参照先 skill へのリンク。`visual-template.html` が凡例・図の枠・`#overview` での全体図だけの表示を持つ
- [ ] AC6: `.claude/skills/plan/SKILL.md` に図解ページ・自己チェック・承認の手順があり、順序が「critical forks → 図解ページ → 自己チェック → Codex advisory → 承認」になっている。承認で `Status: Approved` と `- Approved:`(日付と digest)を書く。Needs changes なら plan と図を直して出し直す。ブラウザがない(`shot` が exit 2)ときと、ページを開けない環境のときの扱いが書いてある
- [ ] AC7: plan テンプレート(`docs/plans/templates/feature-plan.md`、`.claude/skills/plan/template.md`、それぞれの template 側)に `- Approved:` の行と `## Visual review` 節がある。`./scripts/new-feature-plan.sh` で作った plan にこの 2 つが入る
- [ ] AC8: `.claude/skills/implement/SKILL.md` が plan の `Status` と `- Approved:` の digest を `plan-visual.sh digest` の結果と比べる。`Approved` でないとき、または digest が合わないとき(承認 → 設計を変更 → 再開の場合)は、図解と承認をやり直すか続けるかをユーザーに聞く
- [ ] AC9: `.claude/skills/pr/SKILL.md` に次の手順がある。図解ページがあれば全体図だけの PNG を撮る。図のノードと `git diff --name-only <base>...HEAD` を照らし合わせ、ずれていれば図を直す。`gh pr create --attach` で本文に載せる。gh に `--attach` がない・図解ページがない・ブラウザがないときは図を省き、理由を本文に 1 行書く。添付つきの作成が非ゼロで終わったときは、stdout の URL か `gh pr view` で PR の有無を確かめ、あれば `gh pr edit` で直し、なければ添付なしで作り直す。`.claude/skills/pr/template.md` に全体図の欄がある
- [ ] AC10: `subagent-policy.md` の Planning 節が承認ゲートを挙げている。`AGENTS.md`・`.ralph/core/AGENTS.core.md`・`README.md`・`ralph-workflow.md` の `/plan` の説明が図解と承認に触れている(root と template の両側)
- [ ] AC11: `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`./scripts/run-verify.sh` が通る
- [ ] AC12(実地): この plan で図解ページを作って承認ゲートを通す。この PR の本文に全体図の PNG が `--attach` で載る(gh を 2.99.0 以上に上げたうえで)

## Implementation outline

1. S1: `scripts/plan-visual.sh`(open・shot・digest)とテスト、配布物の登録(AC1〜AC4、AC2b)
2. S2: 図解の手引き `diagrams.md` と雛形 `visual-template.html`、ミラーの再生成(AC5)
3. S3: `/plan` の手順と plan テンプレート、`subagent-policy.md`(AC6、AC7、AC10 の一部)
4. S4: `/implement` の `Status` と digest の確認(AC8)
5. S5: `/pr` の全体図の添付(AC9)。gh を上げて、この PR で実地に確かめる(AC12 の後半)
6. 残りのドキュメント(AGENTS.md ほか)は `/sync-docs` で直す(AC10 の残り)

## Verify plan

- Static analysis checks: shellcheck(`scripts/verify.local.sh` の対象一覧に `scripts/plan-visual.sh` を足す)、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`./scripts/check-template.sh`、`scripts/check-template-purity.sh`(手引きと雛形に meta-repo 固有の名前を入れない)、`go vet ./...`
- Spec compliance criteria to confirm: AC1〜AC12(AC2b を含む)。とくに AC6 の手順の順序、AC8 の digest の比較、AC9 の省略と部分成功の扱い
- Documentation drift to check: `AGENTS.md` と `.ralph/core/AGENTS.core.md` の Primary loop、`README.md` の Plan、`ralph-workflow.md`、`subagent-policy.md`。パイプラインの順序は変えないので `scripts/check-pipeline-sync.sh` の対象は変わらない
- Evidence to capture: verify レポート、各チェックの出力

## Test plan

- Unit tests: `tests/test-plan-visual.sh`。オープナーとブラウザをスタブにして分岐ごとの exit code と引数を確かめる
- Integration tests: `./scripts/new-feature-plan.sh` で作った plan に `## Visual review` が入ること。`go test ./internal/scaffold/...`(必須ファイルの存在)
- Regression tests: `./scripts/run-test.sh` の全体。とくに `test-check-template.sh`、`test-sync-skills.sh`、`test-check-skill-sync.sh`、`test-template-purity.sh`
- Edge cases: 空白を含むパス、存在しない HTML、ブラウザなし、`RALPH_*` の環境変数で実行できないファイルを指定したとき、フラグメントの有無
- Evidence to capture: test レポート、実地確認(この plan の図解ページのスクリーンショット、この PR 本文の画像)

## Risks and mitigations

- 手描きの SVG が崩れる → 手引きに数値の目安を書き、自己チェックで撮った画像を見て直す。ブラウザがない環境では自己チェックを省いたことを明記する
- `/plan` が承認待ちで止まる → `/plan` は人と対話する前提の inline スキルで(`subagent-policy.md`)、org runtime からは呼ばれない。Codex 側は番号つきの選択肢で同じ承認を取る
- gh の `--attach` が使えない、または失敗する(古い gh、EMU、private repo)→ 添付を省いて PR 作成を続け、理由を本文に書く。添付つきで失敗したら添付なしで作り直す
- PR に載せる図が実装とずれている → `/pr` で図のノードと変更ファイルの一覧を照らし、ずれていれば直してから撮る
- Codex が画像を読めない → 自己チェックを省き、`## Visual review` に書く。未確認のまま残る点として PR に書く
- `/plan` ごとのトークンが増える → 図解を省ける条件で小さな plan を外す。詳細図は 4 枚までにする
- ミラーが 4 か所(root / template × Claude / Codex)あり、写し漏れが起きやすい → `scripts/sync-skills.sh`、`check-skill-sync.sh`、`check-sync.sh` で機械的にそろえる
- この変更より前に作った Draft の plan で `/implement` が 1 回多く質問する → 止めずに確認だけにする
- 承認後の plan の書き換えで digest が合わなくなる。`/implement` の途中で設計の節を直すと、再開時に聞かれる → 意図した挙動。Branch の行と Progress checklist は digest から除くので、`/implement` の定型の書き換え(step 2g の Branch、step 8 の進捗)では外れない
- PR が作られたのに添付つきのコマンドが非ゼロで終わる → PR の有無を確かめてから直す(Design decisions)

## Rollout or rollback notes

- スキル本文の変更と新規スクリプトだけで、データの移行はない。戻すときは PR を revert する
- 下流には次のリリースの `ralph upgrade` で core ファイルとして届く。戻すときも次のリリースで消す

## Open questions

- Codex CLI が PNG を読めるか(未確認)
- `gh pr create --attach` の EMU アカウント・private repo での挙動(未確認。S5 の実地確認で分かる範囲を記録する)

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Visual review approved
- [x] Implementation started
  - S1 完了(bd6b6333): plan-visual.sh(open・shot・digest)、テスト 82 件、必須一覧の登録。digest は承認時の値 9c20a2da6606 と一致
  - S2 完了(273a30ea): diagrams.md と visual-template.html、4 か所のミラー。SVG の色はクラスで付ける形にした(CSS の `svg text { fill }` が属性に勝つため)
  - S3 の進め方: cross-review の SKILL.md・test-codex-exec-invocation.sh・defaults_sync_test.go が「/plan step 11.c」を参照しているので、Codex advisory は step 11 のまま残す。step 9 と 10 をまとめ、空いた step 10 に図解ページと自己チェック、step 12 に承認ゲートを置く
  - メモ: Linux の `xdg-open` はバックエンドによってブラウザが閉じるまで戻らないかもしれない(未確認)。S3 で `open` の呼び方を書くときに考える
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
