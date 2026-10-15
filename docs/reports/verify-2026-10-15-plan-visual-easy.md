# Verify report: plan-visual-easy

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-plan-visual-easy.md(承認済み。`digest` を打ち直すと `819a5be1cf1f` で、`- Approved:` の行と一致する)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(cycle 1)
- Scope: `git diff origin/main...HEAD`(base 9cbf7080、HEAD 357299b1、9 コミット、34 ファイル)。self-review の直し 0067b837 を含む。仕様の照合、文書のずれ、静的解析を行った。`./scripts/run-test.sh` と `go test` は /test の受け持ちなので打っていない
- Evidence: `docs/evidence/verify-2026-10-15-plan-visual-easy.log`(git は無視する)。`run-static-verify.sh` 自身のログは `docs/evidence/verify-2026-10-15-040525.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `lint` が L1〜L6 を確かめ、出力と終了コードが契約どおり | Met | `lint_page`(`scripts/plan-visual.sh:293-559`)を計画の L1〜L6 と 1 つずつ照らした。L1 `OV_COUNT`・`OV_NAME`・`OV_EASY`、L2 `NEASY` と折りたたみの中の easy 図、L3 `scan_text`(パス・計画と同じ 13 の拡張子・snake_case)と `mono`・`<code>`、L4 箱 5 個まで・`<h2>`・空でない `aria-label`、L5 `NBEFORE` と折りたたみの中、L6 `NFOLD`・`DETAIL_IN_FOLD`・折りたたみの外の easy でない図。出力は `<label>: L<n>: <detail>`。手で打った結果: 引数なしは usage を stderr に出して rc 1、ないファイルは `plan-visual: html file not found or not readable: /nonexistent.html` で rc 1。`--help` の 4 行目が `  scripts/plan-visual.sh lint <html>` |
| AC2 テストに正しいページ、規則ごとに 1 つだけ破るページ、端の場合がある | Met(中身を読んだ。実行は /test) | `tests/test-plan-visual.sh` の lint 節に、正しいページ(`/director`・`--all`・`1/3` を含む)、`id` と `class` の逆順、シングルクォート、複数行のタグ、引用符の中の `>`、`a=b.html`、L1〜L6 それぞれの破り方と `assert_only_rule`、`S1/S2`、`&lt;slug&gt;.html`、前の形のページ(L1)、空のファイル、閉じていない `</figure>` と `</details>` がある |
| AC3 雛形 4 面が新しい形で `lint` を通り、`#overview` と `#detail-overview` で 1 枚だけ見える | Met | 4 面は byte 同一で、4 面とも `lint` rc 0(macOS の awk と ubuntu:24.04 の mawk 1.3.4 の両方)。root の雛形を撮った PNG を読んだ: `--fragment overview` は かんたん版の図 1 だけ(見出し・「承認の前に」・折りたたみは写らない)、`--fragment detail-overview` は閉じた折りたたみが開いて詳細の全体図だけ、全体の 1 枚は見出し・図 1・「承認の前に」・閉じた折りたたみの行 |
| AC4 `diagrams.md` 4 面にかんたん版の決まり、「承認の前に」、折りたたみの中の図の決まり、`lint` を含む自己点検、`lint` が確かめないこと | Met | `.claude/skills/plan/diagrams.md` の「Easy figures」(68-107 行)、「Before you approve」(109-117)、「Detail figures (in the fold)」(119-190)、「Self-check before showing」(192-246、step 1 が lint と古い helper の扱い)、「What `lint` does not check」(238-240)。4 面 byte 同一 |
| AC5 `/plan` step 10 が新しい形、`lint` が 0 まで直す、撮る図、`## Visual review` に lint の結果、古い helper の扱い(D7)。テストが `--help` の行を固定 | Met | `.claude/skills/plan/SKILL.md` の 10.b〜10.d と 12.b。10.c の Lint は `--help` に lint の行がなければ main のチェックアウトの helper、そちらにもなければ飛ばして理由を書く。テストは `tests/test-plan-visual.sh:209` が `  scripts/plan-visual.sh lint <html>` を固定 |
| AC6 `/pr` step 4.a が overview を撮って貼り、`detail-overview` の有無で照らす図を分ける(D8) | Met | `.claude/skills/pr/SKILL.md:37-41`。今の形(`id="detail-overview"` か `id='detail-overview'` がある)、前の形、どちらもの 3 つの子の箇条書きで、0067b837 で「Otherwise」がなくなった。今の形は直したあとに lint を打つ。撮るのは今のまま `--fragment overview`。前の形の実物(`org-director-start.html`)には `detail-overview` が 0 回、この計画のページには 1 回あり、見分けの条件はどちらでも正しく分かれる。origin/main の雛形と `diagrams.md` には `detail-overview` がなく、前の形のページが誤って今の形と読まれる余地はない |
| AC10 `leader.md` step 2 が lint を書き、`prompts_test.go` がその文を固定。`go test` が両方の Go で通る | Met(文と固定は読んだ。`go test` は /test) | `internal/org/prompts/leader.md:102-107` と `internal/org/prompts_test.go:638-640` の 3 行は同じ文。`go build`・`go vet` は手元の go1.26.0 と `GOTOOLCHAIN=go1.25.8` の両方で rc 0。`go test` は打っていない(S3 の進捗メモは両方で ok と書く) |
| AC7 AGENTS.md・repo-map・README・plan の雛形のコメントが lint と新しい形を説明し、sync の検査が通る | Met | AGENTS.md:111(`open` / `shot` / `lint` / `digest`)、`docs/architecture/repo-map.md:62`(open/screenshot/lint/approval digest)、README.md:218、`template.md` と `feature-plan.md`(root・template とも byte 同一)の `## Visual review` のコメント。`check-skill-sync.sh` rc 0、`check-sync.sh` rc 0(DRIFTED 0) |
| AC8 `open`・`shot`・`digest` の動きは変わらず、今のテストはそのまま | Met(静的に。実行は /test) | origin/main と HEAD で `parent_dir`・`abs_path`・`url_encode_path`・`resolve_opener`・`cmd_open`・`resolve_browser`・`cmd_shot`・`digest_body`・`cmd_digest` の本体を切り出して `cmp` し、9 つとも同一。helper の削除行はコメント 3 行だけ。テストの削除行は先頭のコメント 1 行だけ |
| AC9 この計画自身のページが新しい形で `lint` を通る | Met | `.harness/state/plan-visual/plan-visual-easy.html` が macOS の awk と mawk の両方で rc 0。ページに `id="detail-overview"` がある |

ユーザーの決定 3 点: (1) かんたん版の図 1 を `#overview` にし /pr が貼る図もそれにする、は D2・雛形 162 行・`/plan` 10.b・`/pr` の Shoot に入っている。(2) 人が打つコマンドは書いてよく、関数名・パス・イベント名は書かない、は D3・`diagrams.md` の決まり 2(例の `ralph org stop --all` は `internal/cli/org.go:1297` に実在する)・L3 が slash コマンドとフラグを咎めないことに入っている。(3) 機械で確かめられる範囲は `scripts/plan-visual.sh` で確かめる、は AC1 の `lint` に入っている。

### 依頼された判断

(a) L3 が `aria-label` を読まないこと(self-review L1、0067b837 で文書だけ直した)は、計画の L3 と食い違わないと判断した。計画の L3 は「かんたん版の図の中に … 語がない」と書き、属性で名指しするのは class の `mono` だけで、ほかは図の「語」を並べている。Objective は「承認する人が上半分だけを読んで決める」で、`aria-label` は画面に出ない。D3 の「機械で判定するのは … まで」も型の範囲を決める文で、属性まで読むことは求めていない。`diagrams.md` の L3 は 0067b837 で「`aria-label` は読まない。図と同じ平易な言葉で書く」と明記したので、読まない範囲は文書と一致する。確かめた例: `aria-label` にパスと snake_case を入れた雛形は rc 0、SVG の `<title>` に入れると L3 の 2 行で rc 1(`<title>` は文字として読まれる)。厳しくしたいなら、easy 図の `<svg>` の `A_ARIA` を `TEXT[f]` に足す 1 行で済む。今の計画の範囲では必要ない。

(b) S2 の逸脱(かんたん版の図 2・3 も 1 枚ずつ撮る)は、AC5 の「全体、overview、折りたたみの中の図を 1 枚ずつ」を含み、そのうえに撮る図を足すだけで、どの AC とも食い違わない。D5 は折りたたみの中の図を撮る理由を書く決定で、ほかの図を撮ることを禁じていない。S3 の 10.c を Lint と Self-check に分けたこと、`shot` が 2 で終わっても lint を飛ばさないことは、AC5 の lint を飛ばす条件(どちらの helper にも lint がない)と合っている。AC5 は Chrome がないときに lint を飛ばすとは書いていないので、分けたほうが AC に忠実になる。

L2(`leader.md` は「飛ばす」、`diagrams.md` は「main のチェックアウトの helper」)を残した判断にも同意する。leader が読む `diagrams.md` は leader の worktree の `.claude/skills/plan/` の下にあり(`leader.md:95-98`)、同じチェックアウトの `scripts/plan-visual.sh` と同じコミットから来る。worktree の helper に lint がなければ、その `diagrams.md` も lint より前の版で、main のチェックアウトの逃げ道を書いていない。食い違う 2 つの指示が同時に leader に届く組み合わせはない。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | 言語の範囲は full に戻った(`unclassified:.agents/skills/plan/visual-template.html`)。golang: `gofmt: ok`、golangci-lint `0 issues.`。shellcheck(hook と verify の script)、`sh -n`、guard の awk、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照、branch secret scan(9cbf7080..357299b1 clean)がすべて OK |
| `scripts/check-skill-sync.sh` | pass | 14 skill が一致 |
| `scripts/check-sync.sh` | pass(rc 0) | IDENTICAL 169、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `scripts/check-template.sh` | pass(rc 0) | `Template structure looks good.` 新しいファイルはなく、ファイルの一覧は変わらない(D4) |
| `gofmt -l internal/` | pass | 出力なし |
| `go vet ./...`(go1.26.0) | pass | rc 0 |
| `GOTOOLCHAIN=go1.25.8 go build ./... && go vet ./...` | pass | rc 0 |
| `shellcheck scripts/plan-visual.sh templates/base/scripts/plan-visual.sh tests/test-plan-visual.sh` | pass | ShellCheck 0.11.0、rc 0、出力なし |
| `cmp scripts/plan-visual.sh templates/base/scripts/plan-visual.sh` | pass | 同一 |
| 4 面の写しの `cmp`(`/plan` の SKILL.md・diagrams.md・template.md・visual-template.html、`/pr` の SKILL.md の 15 組と `feature-plan.md` の 2 組) | pass | すべて同一 |
| `git diff --check origin/main...HEAD` | pass | rc 0 |
| `scripts/plan-visual.sh digest`(この計画) | pass | `819a5be1cf1f` で `- Approved:` と一致。承認のあとに digest の範囲は変わっていない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| AGENTS.md の repo map(`plan-visual.sh` のサブコマンド) | Yes | `open` / `shot` / `lint` / `digest` |
| `docs/architecture/repo-map.md` | Yes | `open/screenshot/lint/approval digest`。`.harness/state/plan-visual/` の行(73)は今も正しい |
| README の `/plan` の段落(218)と per-plan の段落(260) | Yes | 218 はかんたん版・折りたたみ・lint を書く。260 は「図解ページを書く」の要約で、手順を並べていないので変える必要はない |
| `plan-visual.sh` のヘッダと usage | Yes(1 点だけ細部) | L4 のヘッダの注は「an `<svg>` with an aria-label」で、コードと `diagrams.md` は「空でない」。下の I-4 |
| `.claude/rules/ralph/ralph-workflow.md`、`.ralph/core/AGENTS.core.md`、`docs/recipes/codex-setup.md` | Yes | 「visual review page を見せる」「overview diagram を貼る」「画像が読めなければ自己点検を飛ばす」で、どれも今の形でも正しい。codex-setup の「自己点検を飛ばす」は 10.c の Self-check だけを飛ばす今の手順と合う |
| `docs/quality/`(quality-gates.md:78) | Yes | `digest` の規則だけを参照する |
| `/director` skill(80-84 行)、`/org` skill | Yes | 分割計画のページは「`/plan` skill と同じ手順で」作るので、lint を含む手順がそのまま届く(計画の Non-goals どおり) |
| `/pr` の `template.md` | Yes | `## 全体図` に overview の PNG を載せる。overview はかんたん版の図 1 になった |
| `docs/specs/2026-08-01-org-runtime.md` | Yes | 図解ページの手順を書いていない |
| `docs/specs/2026-10-07-org-multi-org-director.md`(69・72 行) | Yes | 「ralph はページを検査しない」は ralph の本体のことで、`lint` は leader が打つ script なので今も正しい。72 行は 7 段目の決定の記録 |
| `docs/tech-debt/README.md:215`(leader のページを実地で確かめていない行) | No(LOW) | 下の D-1 |
| `docs/tech-debt/README.md:251`(`digest` の `a=b.md`) | Yes | 中身は self-review が scratch で確かめた。参照先は active の計画で、/pr の `archive-plan.sh` が書き換える |
| 計画の `## Visual review`(68 行) | 情報 | 下の I-1 |
| 計画の Progress checklist | 情報 | 下の I-2 |

### Findings

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| D-1 | LOW(文書のずれ) | `docs/tech-debt/README.md:215` は per-plan の leader の手順を「ページを書き、`scripts/plan-visual.sh shot` で撮り、PNG を読んで直す。できなければ QUESTION に書く」と書き、Impact を「撮れない・読めない leader は、誰も点検していないページを送る」とする。この PR で leader は撮る前に `lint` を打つようになった(`leader.md:102-107`)。lint はブラウザなしで回るので、撮れない leader のページも決まりの形は機械で確かめられる。行の説明から lint が抜け、Impact が今より強く書かれている。Trigger の「ページを作ったか、撮ったか、読んだかを driver ごとに記録する」にも lint がない | /sync-docs で行を直す。手順に lint を足し、Impact を「撮れない・読めない leader のページは lint の形の点検だけで送られる(読みやすさと重なりは誰も見ない)」とし、Trigger の記録に「lint を打てたか」を足す。参照列に `docs/plans/active/2026-10-15-plan-visual-easy.md` を足す |
| I-1 | 情報 | 計画の `## Visual review` は「`lint` はまだないので、S1 のあとにこのページへかける(AC9)」のままで、lint の結果(rc 0)は Progress の S1・S3 のメモにある。この節を直すと digest が変わるので、Progress に書いたのは妥当 | 直さない。/pr はこの節からページのパスだけを読む |
| I-2 | 情報 | Progress checklist の「Review artifact created」が未チェックのまま。self-review の報告は 45efffb8 でコミット済み | /sync-docs か /pr でチェックする(digest の範囲外) |
| I-3 | 情報 | L3 が見るのはかんたん版の図だけで、見出し(`<h1>`・リード)と「承認の前に」の欄は見ない。承認する人はこの 2 つも読む。`__WHAT_TO_WATCH__` にパスと snake_case を入れた雛形、`<h1>` にパスを入れた雛形は、どちらも rc 0 だった。計画の L3 は「かんたん版の図の中」と決めているので、計画どおりの範囲 | 直さない。広げるなら次の計画で、L3 の対象に `.before` と `.chrome` の見出しを足すかを決める |
| I-4 | 情報 | `scripts/plan-visual.sh:32-34` のヘッダの L4 は「an `<svg>` with an aria-label」で、コード(`A_ARIA ~ /[^ \t\r\n]/`)と `diagrams.md` の L4 と出力の文は「空でない」。空白だけの `aria-label` を空とみなすのは S1 の記録どおり | 次にヘッダを直すときに「non-empty」を足す。動きは変わらない |

## Observational checks

- root の雛形を scratchpad に写し、`shot` で 3 枚撮って読んだ(PNG は scratchpad にあり、リポジトリには置いていない)
  - `--fragment overview --width 1150 --height 530`: 図 1 の枠だけ。今とこれからの 2 段、人の形、`n-old` の白と `n-new` の青の箱、矢印、下の `__TERM__` と「Built in S1 · S2」。見出し、「承認の前に」、折りたたみは写らない
  - `--fragment detail-overview --width 1150 --height 420`: 折りたたみの中の「Detail 1 · Overview」だけ。等幅のパス、S1・S2 の badge、group の帯。折りたたみの summary の行とほかの図は写らない。閉じた `<details>` は fragment で開いた(この機械の Chrome)
  - 全体(`--height 1500`): 見出し、図 1、「承認の前に」の 3 列、閉じた折りたたみの 1 行。詳細図は写らない(D5 の前提どおり)
- `lint` を ubuntu:24.04 の mawk 1.3.4(`--network none` のコンテナ)で打った。雛形・この計画のページ・前の形の `org-director-start.html`・`aria-label` の例・SVG の `<title>` の例の 5 つで、出力と終了コードは macOS の awk(20200816)と同じだった(R5)
- 前の形の `org-director-start.html` は L1・L2・L5・L6 の 8 行で rc 1(S1 の進捗メモと同じ)。読むだけで、そのページには触れていない

## Coverage gaps

- `./scripts/run-test.sh`、`tests/test-plan-visual.sh`、`go test ./internal/org/...` は打っていない(/test の受け持ち)。AC2・AC8・AC10 のテストの部分は中身を読んだだけ
- 撮ったのは root の雛形だけで、この計画のページは撮り直していない(計画のときに 10:31 に撮った PNG が worktree の `.harness/state/plan-visual/` にある)。4 面の雛形は byte 同一なので、root の結果がほかの 3 面にも当てはまる
- `<details>` が fragment で開くのは Chrome の動きで、ほかのブラウザでは確かめていない(R3)
- consult-advisor への相談はしていない。この subagent には Skill のツールがなく、`/consult` を打てない

## Verdict

- Verdict: pass(D-1 は tech-debt の 1 行の LOW の文書のずれで、/sync-docs で直せる。merge を止めるものではない)
- Verified: AC1、AC3、AC4、AC5、AC6、AC7、AC9。静的解析すべて(run-static-verify、skill と template の sync、check-template、gofmt、go vet と go build の 2 つの Go、shellcheck、写しの cmp、diff --check、digest)。ユーザーの決定 3 点が D2・D3・AC1 に入っていること。L1〜L6 が `diagrams.md` の文と食い違わないこと(L4 の「空でない」はヘッダの注だけが言っていない、I-4)
- Partially verified: AC2・AC8・AC10 は、文・固定・関数の本体を静的に確かめた。テストの実行は /test に残る
- Not verified: Chrome でないブラウザの `<details>` の展開。headless の leader が lint・shot を実地で打てるか(tech-debt 215 行の範囲)
- Insight event: `insights-append.sh --phase verify --verdict pass --cycle auto --low 1` で `docs/insights/events/2026-10-15-plan-visual-easy.jsonl` に 1 行足し(cycle 1)、この報告と同じコミットに入れた
