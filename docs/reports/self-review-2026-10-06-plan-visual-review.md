# Self-review report: plan-visual-review

- Date: 2026-10-06
- Plan: docs/plans/active/2026-10-05-plan-visual-review.md
- Branch: feat/plan-visual-review(base d78777567f3d、HEAD 7df53e9f)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、typo、シェルのクォートとパス、URL エンコード、一時ファイル、コメントの正確さ、保守性)。対象は `git diff d78777567f3d...HEAD`(42 ファイル、+2905/-39)。`.agents/skills/` と `templates/base/` は生成物または byte 同一のミラーなので、`.claude/` 側を 1 回読み、ミラーの乖離だけを見た。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。

## Evidence reviewed

- `scripts/plan-visual.sh`(310 行)を全行読んだ。`open` / `shot` / `digest` の各経路、`trap`、`mktemp`、`abs_path` / `parent_dir` / `url_encode_path`、`require_positive_int` を対象にした。
- 実機の probe(scratchpad 内、リポジトリは触っていない):
  - 実 Chrome 154 が起動中の状態で `shot page.html out.png --fragment overview --width 1150 --height 400 --scale 2` を実行した。exit 0、約 7 秒、PNG を出力した。PNG を読むと、`#overview` で header / summary / legend が隠れ、図だけが写っている。
  - 偽ブラウザで `shot e.html ""`(`<png>` が空文字)を実行した。`rm: .../pv/: is a directory` が出て exit 1(F-2)。
  - 偽ブラウザで `shot swapped.png precious.html`(引数の取り違え)を実行した。`precious.html` が書き換わった(F-2)。実 Chrome でも同じパスに PNG を書くので、`rm -f` だけが原因ではない。
  - `shot same.html same.html`(入力と出力が同じパス)は exit 0 で、HTML が上書きされた(F-2)。
  - Chrome の `--dump-dom` で `<svg><text>.harness/state/plan-visual/<slug>.html</text></svg>` を読ませた。結果は `<text ...>.harness/state/plan-visual/<slug>.html</slug></text>` で、`<slug>` が未知の要素として解釈され、表示文字が `.harness/state/plan-visual/` までに切れる(F-1)。
  - `digest` を、末尾改行のある plan とない plan で比べた。どちらも `d2968bb5016b`(F-3)。`## Progress checklist ` のように見出しの末尾に空白があると、チェックリストが除外されず、チェック状態でダイジェストが変わる。これは header コメントに「exact heading」と明記されているので指摘にしていない。
  - この plan 自身のダイジェストを `digest` で再計算した。`9c20a2da6606` で、plan の `- Approved:` 行の値と一致する。
- ミラー: `.claude/` 側の変更ファイル 17 本が `templates/base/` 側と `cmp` で一致。`.agents/skills/` 側の 7 本は `.claude/` 側と一致。`docs/plans/templates/feature-plan.md` は `.claude/skills/plan/template.md` と一致。`scripts/plan-visual.sh` は root と `templates/base/` が同じ blob(`b6940958`)、4 つの新規・変更スクリプトは mode 100755。
- 外部への言及の確認: `gh api repos/nntto/skills` の `license` が `null`(diagrams.md の「no license file」と整合)。`gh api repos/cli/cli/releases/tags/v2.99.0` の本文に `--attach` の節がある(pr/SKILL.md の「gh 2.99.0 and later」と整合)。手元の gh は 2.96.0 で `gh pr create --help` に `--attach` がなく、これは pr/SKILL.md が自分で例示している状況と一致する。
- step 番号の参照: `/plan` step 11.c を参照する `cross-review/SKILL.md:55,60` はそのまま有効。`implement/SKILL.md:32,36,71` と `pr/SKILL.md:38` の「step 10」「step 12」は plan/SKILL.md の見出しと一致する。
- `.harness/state/` は root と `templates/base/` の両方の `.gitignore` に入っている(`.gitignore:47`)。
- 秘密情報、デバッグ痕跡、ローカルパス: 追加行を `/Users/`、`TODO`、`FIXME`、`console.log`、`set -x`、`password`、`token`、`api_key` で grep した。ヒットは `mktemp` の `TMPDIR` 1 行(`scripts/plan-visual.sh:238`)だけで問題なし。`git diff --check` も空。
- 兄弟テストとの慣用の照合: `tests/test-new-feature-plan.sh:21` の `trap cleanup EXIT HUP INT TERM` と `cleanup() { [ -n "$_tmp" ] && rm -rf "$_tmp"; }` は `tests/test-branch-name.sh:13-16`、`tests/test-gc-artifacts.sh:13-16` と同じ形なので、指摘にしていない。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | maintainability | 図解ページの手順に、SVG のテキストに入れる `<` `>` `&` のエスケープがない。`diagrams.md` は node のラベルを「path か plan が使う名前」にするよう指示するが、この repo の path 表記には `<slug>` `<date>` が頻出する。生のまま `<text>` に書くと、`<slug>` が未知の要素になって文字が黙って消える。承認者が見る図のラベルが欠ける。自己チェックの PNG 確認で気づける場合もあるが、画像を読めない環境ではチェック自体が省かれる(diagrams.md:135-139) | `.claude/skills/plan/diagrams.md:97-101`(Grounding)と `:102-120`(Page structure)にエスケープの記述がない。`visual-template.html:163` のプレースホルダ文にも注意書きがない。probe: Chrome `--dump-dom` で `<text>.harness/state/plan-visual/<slug>.html</text>` が `<text>...plan-visual/<slug>.html</slug></text>` に変わった | `diagrams.md` の Grounding か Page structure に 1 行足す。例:「テキスト中の `<` `>` `&` は `&lt;` `&gt;` `&amp;` と書く(例: `plan-visual/&lt;slug&gt;.html`)」。`.claude/` を直せば他のミラーは `sync-skills.sh` と `templates/base/` の複製で揃う |
| F-2 LOW | exception-handling | `shot` の `<png>` 引数を検証していない。(a) 空文字だと `abs_path` が `<cwd>/` を返し、`rm -f` が `is a directory` を出して exit 1 になる。`plan-visual:` の前置きもメッセージもなく、利用者は原因が分からない。(b) `<html>` と `<png>` を取り違える、または同じパスを渡すと、HTML を PNG で上書きする。実 Chrome でも同じパスに書くので `rm -f` だけの問題ではないが、`rm -f` が既存ファイルの削除を確実にする | `scripts/plan-visual.sh:225-229`(`<png>` の検査は親ディレクトリの存在と `-d` だけ)、`:240`(`rm -f "$_png_abs"`)。probe: `shot e.html ""` は `rm: .../pv/: is a directory`、`shot swapped.png precious.html` は `precious.html` を書き換え、`shot same.html same.html` は exit 0 で上書き | `abs_path` の後に `[ "$_html_abs" != "$_png_abs" ] \|\| die 1 "..."` と、`<png>` が空文字または `*.png` でないときの `die 1` を足す。skill 側の呼び出しは固定名なので、実害が出るのは手で呼んだときだけ |
| F-3 LOW | readability | `digest` の header コメントが理由を取り違えている。「Each kept line is hashed followed by a newline, so the value matches across sha256sum and shasum」とあるが、`sha256sum` と `shasum -a 256` は同じ入力バイトなら同じ値を返すので、改行の有無とは関係がない。awk の `print` が末尾に改行を足すことで得られる性質は「plan の末尾改行の有無でダイジェストが変わらない」こと | `scripts/plan-visual.sh:26-29`(コメント)、`:259-266`(`digest_body`)。probe: 末尾改行なしと末尾改行ありの plan がどちらも `d2968bb5016b` | コメントを「Each kept line is printed with a trailing newline, so a plan with or without a final newline gives the same digest.」の趣旨に直す |
| F-4 LOW | maintainability | `RALPH_PLAN_VISUAL_OPENER=none` の扱いが 2 か所にある。`cmd_open` が先に `none` を判定して返すので、`resolve_opener` 内の `none` 分岐は `cmd_open` から到達しない。`resolve_browser` は `none` を関数内で 1 回だけ扱っており、2 つの resolver で作りが違う | `scripts/plan-visual.sh:115-117`(到達しない分岐)、`:178-181`(`cmd_open` 側の判定) | どちらかを消す。メッセージを分けたいなら `cmd_open` 側を残し、`resolve_opener` の `none` 分岐を削る |
| F-5 LOW | readability | `diagrams.md` の予算「Text size ≥ 12px everywhere」と、同じ組のテンプレートの CSS が食い違う。slice の badge は 11px、legend の pill は 11px、legend の「Removed」見本の ✕ は 9px | `.claude/skills/plan/diagrams.md:82`、`.claude/skills/plan/visual-template.html:43,46,72` | 予算を「node と edge のラベルは ≥ 12px(badge と legend の見本は除く)」に言い換えるか、`.badge-text` と `.pill` を 12px にする。badge の rect は高さ 17 なので 12px は収まる |

## Positive notes

- `scripts/plan-visual.sh` は `trap cleanup EXIT` と `trap 'exit 1' HUP INT TERM` を分けており、シグナル後に処理が再開しない形になっている。`_log` は trap の前に空文字で初期化され、`rm -f` は `-n` で守られている。
- `abs_path` は `CDPATH='' cd --` と、`/` 直下の特別扱いまで入れている。`url_encode_path` は `%` を最初に変換している(順序が正しい)。テストが `100% #1?.html` で確認している。
- `shot` は、前回の PNG が残っていても今回の出力と見なさないよう `rm -f` してから撮り、ブラウザが exit 0 でも PNG が書かれていなければ失敗にする。ブラウザの出力は失敗時だけ末尾 20 行を出す。
- stdout に出すのは絶対パスだけに限り、opener の stdout を stderr に回している。呼び出し側(skill)がそのまま利用者に渡せる。
- テストは opener とブラウザを常に stub にし、PATH を絞って `uname` / `open` / `xdg-open` / `sha256sum` / `shasum` 単独の経路を別々に確認している。ダイジェストは外部実装で求めた既知の値で固定し、承認時の書き換え(Status / Approved / Branch / チェックリスト)でダイジェストが変わらないことと、本文 1 文字の変更で変わることを両方見ている。
- plan 自身の `- Approved:` のダイジェストが、再計算した値と一致している。
- ミラーの乖離はなく、`.harness/state/` は gitignore 済みなので、図解ページと PNG は commit されない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

なし。F-1 から F-5 はこの PR の中で直せる大きさなので、`docs/tech-debt/` には足さない。

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: 可。CRITICAL と HIGH はない。MEDIUM 1 件(F-1)は、図解ページの手順に 1 行足せば閉じる。直してから `/verify` に進むことを勧める。直さずに進める場合は、既知の欠けとして PR 本文に残す。
- Follow-ups: F-2 から F-5 は任意。F-2 と F-3 は `scripts/plan-visual.sh` を触る 1 コミットで済み、変更後は root と `templates/base/scripts/plan-visual.sh` を同じ内容に揃える(byte 同一を保つ)。F-1 と F-5 は `.claude/skills/plan/` を直した後に `scripts/sync-skills.sh` で `.agents/` を再生成し、`templates/base/` に複製する。
- Insight event: 依頼で「report 以外のファイルを編集しない」とされたので、`./scripts/insights-append.sh --slug plan-visual-review --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 1 --low 4 --source skill` は実行していない。必要なら呼び出し側で実行する。

## レビュー後の対応(orchestrator 追記)

- F-1〜F-5 はすべて c4a66c0d で直した。F-1 は diagrams.md の Page structure 節と visual-template.html の先頭コメントにエスケープの規則を足した。F-2 は `shot` が空の `<png>`、`.png` 以外の名前、`<html>` と同じパスを exit 1 で拒むようにし、テストを 4 件足した。F-3 はコメントの理由を末尾改行の扱いに直した。F-4 は重複していた `none` の分岐を消した。F-5 は文字サイズの目安を、バッジと記号に限って 11px を認める形にし、テンプレートの ✕ を 11px にした。`tests/test-plan-visual.sh` は 96 件すべて通過、`./scripts/run-verify.sh` は rc 0。
- Insight event は c340945f で記録した(`docs/insights/events/2026-10-06-plan-visual-review.jsonl`)。

## Cycle 2 (2026-10-06, HEAD de99dd6c)

- Reviewer: reviewer subagent (Claude)、cycle 2
- Scope: diff の品質だけ。全体の `git diff d78777567f3d...HEAD`(60 ファイル、+3621/-63)を見たうえで、cycle 1 以降の `git diff 10f76e06..HEAD`(42 ファイル、+725/-96)に絞って読んだ。対象は c4a66c0d(F-1〜F-5 の修正)、b0ea4a23(digest のチェックボックス正規化と古いテンプレートへの対応)、804c5d84(再承認)、4caaec72(sync-docs)、de99dd6c(`/pr` 5.c の PR 探索)。`.agents/skills/` と `templates/base/` はミラーなので、`.claude/` 側を 1 回読み、乖離だけを見た。仕様への適合、テストの網羅、文書のずれは見ていない。

### Evidence reviewed

- cycle 1 の F-1〜F-5 を、修正後の現物で 1 件ずつ確認した。すべて閉じている。
  - F-1: `.claude/skills/plan/diagrams.md:109-111` に `&` `<` `>` のエスケープ規則と例(`plan-visual/&lt;slug&gt;.html`)、`.claude/skills/plan/visual-template.html:6-7` に同じ注意書きがある。
  - F-2: `scripts/plan-visual.sh:204`(空の `<png>` は usage で exit 1)、`:232-235`(`.png` 以外を拒否)、`:242`(`<html>` と同じパスを拒否)。header コメント `:22` にも規則がある。テスト 4 件と `.PNG` の受理は `tests/test-plan-visual.sh:314-341`。
  - F-3: header コメント `:27-34` の理由が「末尾改行の有無でダイジェストが変わらない」に直っている。
  - F-4: `resolve_opener` の到達しない `none` 分岐がなくなり、`:120-121` のコメントが `cmd_open` 側で扱うと書いている。
  - F-5: `diagrams.md:82` の予算がバッジと小さな記号に限って 11px を認める形になり、`visual-template.html:44` の ✕ が 11px になった。テンプレートの CSS と SVG の `font-size` を grep すると、12px 未満は badge / pill / 見本の 11px だけ。
- digest の正規化(`scripts/plan-visual.sh:272-284`)を probe した(リポジトリのファイルには触れず、標準入力に流した)。macOS の awk 20200816 に `- [x] a`、`  - [X] b`、タブ字下げの `- [x] c`、`- [ ] d`、`text [x] e` を渡すと、先頭 3 行が `- [ ]` になり、`- [ ] d` と行中の `[x]` はそのまま残った。`match` の `RLENGTH` で `[` までを残し、`x` だけを空白に替えているので、`]` 以降の本文は動かない。`## Progress checklist` の除外は正規化より前の規則なので、順序も合っている。
- この plan 自身の `- Approved:`(`d4918bfcec38`)と `./scripts/plan-visual.sh digest docs/plans/active/2026-10-05-plan-visual-review.md` の出力が一致する。804c5d84 の後に 4caaec72 で plan が 15 行変わっているが、変更はチェックボックスの印だけで、digest は動いていない。
- `gh pr list --head no-such-branch-zz --base main --state open --json url --jq '.[0].url'` を gh 2.102.0 で実行した。exit 0、stdout は改行 1 つで、`null` ではない。de99dd6c の「empty output means no such PR」と整合する。`gh pr view` が出てくるのは `.claude/skills/pr/SKILL.md:40` の「ここでは使わない」の 1 文だけ。
- 新しいテスト(`assert_same_bytes`、`write_ac_plan`、空の `<png>` 以降の 4 件、`.PNG`、末尾改行なしの plan、チェックボックスの 8 件)を読んだ。helper の命名とコメントの形は同じファイルの既存 helper(`assert_has_line`、`write_plan`)と揃っている。
- ミラー: cycle 1 以降に変わった `.claude/` 側・docs 側の 14 ファイルが `templates/base/` 側と `cmp` で一致。`.agents/skills/` 側の 7 ファイルも一致。`scripts/plan-visual.sh` は root と `templates/base/` が同じ blob(`8420ba7d`)で mode 100755。
- digest の規則を述べる箇所(`implement/SKILL.md:35,71`、`plan/SKILL.md:99-100`、`plan-visual.sh` の header と usage)はどれもチェックボックスの扱いを書いていて、食い違いはない。
- 秘密情報とデバッグ痕跡: cycle 1 以降の追加行を `/Users/`、`TODO`、`FIXME`、`console.log`、`set -x`、`password`、`api_key` で grep した。実害のあるヒットなし(`docs/reports/` の `secret-scan` の記述とパス表記だけ)。`git diff --check` は空。

### Findings (cycle 2)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-6 LOW | readability | `/pr` の同じ Step 内で、同じ「base」を指す記号が 2 つある。5.a は `<ref>`(「Step 3 が `against <ref>` と出した base ref」)、5.c は `--base <base>`。`<ref>` は `origin/main` の形で出る(remote の ref があるとき)が、`gh pr list --base` が取るのは `main` の形で、`<base>` が `<ref>` から `origin/` を除いたものだとはどこにも書いていない。`<ref>` をそのまま渡すと、open の PR があっても見つからず「No PR」になり、`gh pr create` が「PR が既にある」と断って、5.c が「画像以外が原因なので止めて報告する」と案内する。安全側に倒れるが、原因が分かりにくい | `.claude/skills/pr/SKILL.md:37`(`<ref>`)、`:40`(`--base <base>`)、`:29`(`git fetch origin <base>` は `<base>` をブランチ名として使っている)。`scripts/secret-scan-branch.sh:164-169,528`(remote の ref があると `base_ref="origin/$base_name"`、出力は `against origin/main: clean`) | 5.c に 1 句足す。例:「`<base>` はブランチ名で、`<ref>` から remote の接頭辞を除いたもの(`origin/main` なら `main`)」。`.claude/` を直したら `sync-skills.sh` と `templates/base/` の複製で揃える |
| F-7 LOW | maintainability | plan の AC9 と Design decisions が、`/pr` の復旧手順を今も「`gh pr view` で PR の有無を確かめる」と書いていて、最終の実装(de99dd6c の `gh pr list --head ... --state open`)と食い違う。AC9 は `[x]` で閉じている。`/verify` の担当(仕様適合)に近いが、plan が diff に含まれ、契約として読まれるので書いておく。本文を直すと digest(`d4918bfcec38`)が変わり、再承認が要る。最後の cycle で再承認を足すのは重い | `docs/plans/active/2026-10-05-plan-visual-review.md:88`(Design decisions「`gh pr view <head branch>` で PR があるかを確かめる」)、`:102`(AC9「`gh pr view` で PR の有無を確かめ」)。実装は `.claude/skills/pr/SKILL.md:40` | digest を変えない方法で足す。plan の「Progress checklist」(`:155`、digest の対象外)に 1 行、「cross-review の指摘で、5.c の PR 探索を `gh pr view` から open の PR だけを見る `gh pr list --head <branch> --base <base> --state open` に変えた(de99dd6c)」と書く。AC9 と Design decisions の本文は、このあとの `/pr` でアーカイブされるまま残して構わない |

### Positive notes (cycle 2)

- F-2 の修正は、検査を usage、拡張子、同一パスの 3 段に分け、どれもブラウザを起動する前(`:204`、`:232-235`、`:242`)に置いている。テストが「拒否されたパスではブラウザが起動しない」ことと、HTML のバイト列が変わらないことを両方確かめている。
- digest の正規化は `match` と `substr` だけで、外部コマンドを足していない。範囲を行頭の `- [x]` / `- [X]` に絞り(行中の `[x]` は数える)、その境界を 3 件のテスト(本文の変更、字下げ、行中)が固定している。header コメントと usage が同じ範囲を同じ言葉で書いている。
- 古いテンプレートの plan への対応(`plan/SKILL.md` の 10.a、10.d、12.e)は、seed ファイルが `ralph upgrade` で書き換わらないという事実に基づいて、どこに何を足すかまで書いている。
- de99dd6c は、`gh pr view` を使わない理由(open がないとき、同名ブランチの merged / closed の PR を返す)と、closed / merged / 別 base を「PR なし」に数える規則を同じ文に入れている。コミットメッセージが原因と対処を書いていて、diff の範囲もミラー 4 か所だけで済んでいる。

### Tech debt identified (cycle 2)

なし。F-6 は 1 句、F-7 は plan に 1 行で閉じる大きさなので、`docs/tech-debt/` には足さない。

### Recommendation (cycle 2)

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。cycle 1 の 5 件は閉じている。
- Follow-ups: F-6 と F-7 は LOW で任意。pipeline の上限(2 回)に達しているので、直す場合は doc だけの小さな commit にとどめる。直さない場合は、PR 本文の既知の欠けに 1 行ずつ残す(F-7 は plan の Progress checklist への 1 行が digest を動かさない最小の対応)。
- Insight event: `./scripts/insights-append.sh --slug plan-visual-review --flow standard --phase self_review --cycle 2 --verdict pass --critical 0 --high 0 --medium 0 --low 2 --source skill` を実行して、この report と同じ commit に入れた。
