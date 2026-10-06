# Verify report: plan-visual-review

- Date: 2026-10-06
- Plan: docs/plans/active/2026-10-05-plan-visual-review.md
- Verifier: verifier subagent (Claude)、cycle 1
- Scope: 仕様への適合(AC1〜AC12、AC2b を含む)、静的解析、文書のずれ。対象は `git diff d78777567f3d...HEAD`(base origin/main d78777567f3d、HEAD 5d7aa671、44 ファイル、+3051/-39)。self-review の修正コミット c4a66c0d を含む。テスト(`./scripts/run-test.sh`、`go test`、`tests/test-*.sh`)は /test の担当なので実行していない
- Evidence: `docs/evidence/verify-2026-10-06-plan-visual-review.log`(`docs/evidence/*.log` は gitignore 対象なので、手元にだけ残る)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `open` の 3 分岐と環境変数での差し替え | Met(コード上。実行は /test) | `scripts/plan-visual.sh:173-191`。ファイルがなければ `die 1`、あれば絶対パスを stdout に出してから開く。`RALPH_PLAN_VISUAL_OPENER`(`none` で無効)、macOS は `open`、それ以外は PATH 上の `xdg-open`(`:116-130`)。オープナーがない・失敗したときは stderr に注記して exit 0 |
| AC2: `shot` の PNG 出力、フラグメントとウィンドウの大きさ、ブラウザなしで exit 2、環境変数でブラウザ指定 | Met(コード上。実行は /test) | `:193-263`。`--fragment` / `--width` / `--height` / `--scale` を受け、`resolve_browser` が失敗すると `exit 2`(`:238`)。`RALPH_PLAN_VISUAL_BROWSER` は実行可能なファイルだけを受け、`none` はブラウザなしの扱い(`:134-164`)。self-review F-2 の修正(空の `<png>`、`.png` 以外、`<html>` と同じパスを exit 1 で拒む)も入っている(`:198`、`:226-236`) |
| AC2b: digest の除外範囲と、除外部分以外の差で値が変わること | Met(再計算) | `digest_body`(`:266-273`)は `- Status:` / `- Approved:` / `- Branch:` の行と `## Progress checklist` から次の `## ` までを落とす。この plan で `digest` を実行して `9c20a2da6606`。同じ規則を python で別に実装して求めた値も `9c20a2da6606`。macOS と Linux の一致は、テストが `sha256sum` だけ・`shasum` だけを PATH に置いて既知の値 `1174b9b1faae` を確かめる形になっている(実行は /test) |
| AC3: `tests/test-plan-visual.sh` が列挙されたケースを確かめる | Met(ケースの有無を確認。実行は /test) | ファイルなし、オープナーあり / なし / 失敗、`RALPH_PLAN_VISUAL_BROWSER=none` と実行できないパスで exit 2、スタブのブラウザに渡る `--window-size=1150,470`・`--force-device-scale-factor=2`・`file://…%20…#overview`、空白を含むパス、digest の一致(除外部分だけが違う、末尾改行なし)と不一致(本文 1 文字、別の header 行、checklist の後ろの節、字下げした `- Status:`)がある。orchestrator は c4a66c0d の時点で 96 件通過と記録している(未確認。/test で確かめる) |
| AC4: template 側への配置と必須一覧 | Met(`go test` は /test) | `templates/base/scripts/plan-visual.sh` は root と同じ blob `ab8f0517`、mode 100755。`scripts/check-template.sh`(root と template)、`internal/scaffold/embed_test.go:75`、`tests/test-check-template.sh` の GOLDEN_ENTRIES に載っている。`check-sync.sh` は DRIFTED 0 |
| AC5: `diagrams.md` の 8 項目と `visual-template.html` の凡例・図の枠・`#overview` | Met | `diagrams.md` に Purpose、Granularity、「What changes → figure type」の表、「When to draw, when to skip」、Visual vocabulary、Layout budgets、Grounding、Credit(nntto/skills へのリンク)がある。雛形は `.legend`、`<figure id="overview">` と `<figure id="detail-1">`、`body:has(figure:target)` で他の図と `.chrome` を隠す CSS(`visual-template.html:75-79`)を持つ。self-review F-1 / F-5 の修正(`&` `<` `>` のエスケープ、11px を許す範囲)も入っている |
| AC6: `/plan` の手順と順序、承認の記録、Needs changes、ブラウザなし・開けない環境 | Met | `.claude/skills/plan/SKILL.md` の step 8(critical forks)→ 10.a〜b(図解ページ)→ 10.c(自己チェック)→ 11(Codex advisory)→ 12(承認)。12.e で `Status: Approved` と `- Approved: <日付> sha256:<digest>` を書く。12.d で plan と図を直して 12.a に戻る。10.c で `shot` の exit 2 と画像を読めない場合は自己チェックを省く。12.a で開けないときも表示されたパスを渡す。step 11.c を参照する `cross-review/SKILL.md:55,60` と `tests/test-codex-exec-invocation.sh:96` はそのまま有効 |
| AC7: plan テンプレート 4 か所に `- Approved:` と `## Visual review` | Met(`new-feature-plan.sh` の実行は /test) | `docs/plans/templates/feature-plan.md` と `.claude/skills/plan/template.md` は同じ内容で、template 側も byte 同一。`scripts/new-feature-plan.sh:39-45` は `docs/plans/templates/feature-plan.md` を sed で写すので、作った plan に 2 つが入る。このブランチのバイナリで `ralph init` した場合も 2 つが入った。upgrade した下流では入らない(V-2) |
| AC8: `/implement` が Status と digest を比べ、合わなければ聞く | Met | `.claude/skills/implement/SKILL.md:32-38`。再開のたびに `- Status:` と `- Approved:` を読み、`plan-visual.sh digest` と比べる。`- Approved:` がない、`TBD`、別の Status、digest の不一致のときは、やり直す / 続ける / 止める の 3 択で聞く。`:71` は plan 本文への deviation note が digest を変えることも書いている |
| AC9: `/pr` の全体図の手順と PR テンプレートの欄 | Met(実地は AC12 の後半) | `.claude/skills/pr/SKILL.md:34-43`。5.a で図解ページなし・`--attach` なし・`shot` の失敗のときに省いて理由を 1 行書く、`git diff --name-only <ref>...HEAD` と図を照らす、全体図だけを撮る。5.b で `--body-file` と `--attach '<path>#<alt>'`。5.c で非ゼロ終了時に stdout の URL か `gh pr view` で PR の有無を確かめ、あれば `gh pr edit`、なければ添付なしで作り直す。`:63` の完了条件にも入っている。`.claude/skills/pr/template.md:5-9` に「全体図」節がある |
| AC10: Planning 節と `/plan` の説明(root と template) | Partially met(残りは /sync-docs) | `subagent-policy.md` の Planning 節は root と template の両方で「the visual review approval gate」を挙げている。`AGENTS.md:31`、`.ralph/core/AGENTS.core.md:20`、`README.md:218`、`ralph-workflow.md:16` と、template 側の `AGENTS.md` / `AGENTS.core.md` / `ralph-workflow.md` には図解と承認の記述がまだない。plan の Implementation outline 6 のとおり、/sync-docs で直す予定 |
| AC11: `check-skill-sync.sh`、`check-sync.sh`、`run-verify.sh` | Met(`run-verify.sh` は静的解析の部分だけ) | `./scripts/run-static-verify.sh`(= `HARNESS_VERIFY_MODE=static` の `run-verify.sh`)の中で `check-skill-sync.sh`(13 skill)と `check-sync.sh` が OK、全体で rc 0。既定の `all` モードのテスト部分は /test に回した |
| AC12: この plan で図解と承認を通し、PR 本文に全体図を載せる | 前半 Met、後半は /pr で確認 | 前半: 図解ページ `.harness/state/plan-visual/plan-visual-review.html` と PNG 3 枚が worktree にある(gitignore 済み)。全体図の PNG を読み、S1〜S5 と 3 つのスキルが図にあることを確かめた。plan は `Status: Approved`、`- Approved: 2026-10-05 sha256:9c20a2da6606` で、HEAD でも最初のコミット 7ebd885c でも digest は同じ値。承認の操作そのものは会話の中のことなので、成果物からは確かめられない。後半: 手元の gh は 2.96.0 で `--attach` がない(想定どおり)。/pr の時点で確かめる |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | 指定の範囲は changed だったが、`visual-template.html` が分類されないため full に切り替わった(`full fallback (unclassified:.agents/skills/plan/visual-template.html)`)。local verifier: shellcheck(対象一覧に `scripts/plan-visual.sh` と `tests/test-*.sh` を含む)OK、hook 20 本の `sh -n`、`jq -e`、`check-sync.sh`(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`。golang: `gofmt: ok`、`go vet` は出力なし(成功)、golangci-lint `0 issues.`、staticcheck は出力なし(成功)。branch secret scan: `scanned d7877756..5d7aa671 against origin/main: clean` |
| `shellcheck -s sh scripts/plan-visual.sh` | pass(rc 0) | POSIX sh として単独で確認 |
| `shellcheck tests/test-plan-visual.sh tests/test-new-feature-plan.sh` | pass(rc 0) | |
| `sh -n` / `dash -n scripts/plan-visual.sh` | pass(rc 0) | |
| `./scripts/check-template.sh`(root) | pass(rc 0) | `Template structure looks good.` だけで、以前出ていた hook の FAIL 行は出なかった |
| `git diff d78777567f3d...HEAD --check` | pass | 空白の誤りなし |
| `go build ./cmd/ralph`(このブランチと base d7877756) | pass | 下の Observational checks の init / upgrade 用 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `subagent-policy.md` の Planning 節(root / template) | Yes | 承認ゲートを挙げている。2 本は byte 同一 |
| `AGENTS.md` / `.ralph/core/AGENTS.core.md` の Primary loop、`README.md` の Plan、`ralph-workflow.md` の `/plan` の行(root / template) | No(予定どおり) | 図解と承認に触れていない。AC10 の残りとして /sync-docs が直す。実装の不備ではない |
| ミラー 4 か所(root / template × `.claude` / `.agents`) | Yes | 変更した skill 7 本は 4 か所とも同一(`cmp`)。`scripts/plan-visual.sh`、`check-template.sh`、`feature-plan.md`、`subagent-policy.md` も root と template で同一 |
| `/plan` の step 番号への参照 | Yes | `implement/SKILL.md:32,36,71` の step 10 / 12、`pr/SKILL.md:38` の step 10.c、`cross-review/SKILL.md:55,60` の step 11.c は、plan/SKILL.md の見出しと合っている |
| plan の Risks(140 行目)「`/implement` の定型の書き換えでは digest が外れない」 | No(V-1) | AC のチェックボックスに印を付けると digest が変わる |
| plan の Rollout notes「下流には `ralph upgrade` で core ファイルとして届く」 | No(V-2) | `docs/plans/templates/feature-plan.md` は seed なので、upgrade では advisory しか出ない |
| 任意: `AGENTS.md:111` の `scripts/` 一覧、`docs/recipes/codex-setup.md:41` の `$plan` の説明 | 任意 | `plan-visual.sh` と承認の手順が載っていない。どちらも AC10 の対象外で、一覧は主なスクリプトだけを挙げる形なので、/sync-docs で足すかどうかを決めればよい |

### Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V-1 | MEDIUM | digest は `## Acceptance criteria` 節を含むので、AC のチェックボックスに印を付けるだけで承認と合わなくなる。この repo では slice を終えるたびに plan の AC に印を付けるのが慣習になっている。そのため、`/implement` を再開するたびに(コンテキストの圧縮や別セッションのあと)「図解と承認をやり直すか」と聞かれるおそれがある。聞かれても止まりはしないが、毎回「続ける」を選ぶ習慣がつくと、承認ゲートの意味が薄れる。plan の Risks は「定型の書き換え(Branch、進捗)では外れない」と書いているが、AC の印には触れていない。AC2b の定義どおりの挙動なので、AC の違反ではない | probe: この plan の `- [ ] AC1:` を `- [x]` に変えると digest が `9c20a2da6606` から `b7b57e14b101` に変わる。Progress checklist の項目に印を付けた場合は `9c20a2da6606` のまま。アーカイブ済みの plan では、AC の印が slice の記録コミット(`059cbe01 docs: record Slice C results in the plan` で 7 件、`7bfe76bc` で 9 件)と /sync-docs(`7c984d60`)で付いている。`implement/SKILL.md:71` は「進捗だけの記録は Progress checklist に書く」と言うが、AC の印には触れていない | 次のどちらかを選ぶ。(a) `digest_body` で `- [x]` を `- [ ]` に正規化して、チェックボックスの状態を digest から外す(AC2b と Design decisions を直し、テストを足す。この plan の本文には Progress checklist の外に `[x]` がないので、今の値 `9c20a2da6606` は変わらない)。(b) `/implement` と `/sync-docs` に「承認後は AC の印を付けず、達成状況は Progress checklist に書く」と足す。どちらにしても、この PR の /sync-docs では、この plan の AC に印を付けないか、付けると digest が外れることを承知しておく |
| V-2 | LOW | `ralph upgrade` した下流では、plan を作るテンプレートが古いまま残る。`/plan` step 6 が呼ぶ `new-feature-plan.sh` は seed の `docs/plans/templates/feature-plan.md` を写すが、upgrade は seed を書き換えず advisory を出すだけ。core の `.claude/skills/plan/template.md` は新しくなるので、2 つのテンプレートの中身がずれる。新しく作った plan には `- Approved:` の行も `## Visual review` 節もなく、step 10.d の「`## Visual review` に書く」と step 12.e の「`- Approved:` を書く」は、その行や節があることを前提にした書き方になっている。digest と `/implement` の確認は行がなくても動く(`- Approved:` がない場合は未承認として聞く) | base d7877756 のビルドで `ralph init` し、このブランチのビルドで `ralph upgrade --yes`(rc 0)。Created は `diagrams.md` ×2、`visual-template.html` ×2、`scripts/plan-visual.sh`。`feature-plan.md` は Advisories に入り、upgrade 後も `- Approved:` / `## Visual review` は 0 行。`.claude/skills/plan/template.md` には 2 行とも入った | /plan の step 10.d と 12.e に「行や節がなければ足す」と 1 句足す。plan の Rollout notes と PR 本文に「upgrade した下流は `feature-plan.md` の advisory を取り込む」と書く |

## Observational checks

- このブランチのバイナリで、`git init` しただけの一時ディレクトリに `ralph init --yes`(rc 0)。`scripts/plan-visual.sh`(0755、root と byte 同一)、`.claude` と `.agents` の `diagrams.md` / `visual-template.html` ができ、manifest の owner はどれも core だった。`docs/plans/templates/feature-plan.md` は seed。scaffold の中で `./scripts/check-template.sh` は rc 0
- base d7877756 のビルドで init → このブランチのビルドで upgrade。結果は V-2 のとおり。一時領域はすべて検証のあとに消した。`git worktree add` は使っていない
- 全体図の PNG(`.harness/state/plan-visual/plan-visual-review-overview.png`)を読んだ。`/plan`・`/implement`・`/pr` の枠、S1〜S5 のバッジ、`scripts/plan-visual.sh` と `plan/diagrams.md・雛形` から各手順への点線があり、文字の重なりや切れは見当たらない。図の部品は差分のファイルと合っている
- 図解ページの HTML の更新時刻(10-05 19:32)は、plan の最初のコミット 7ebd885c(19:00)より後だった。承認のときに見せたページと今のページが同じかは分からない。ページはコミットしない設計なので、指摘にはしていない
- この plan の Progress checklist の項目名は「Visual review approved」で、テンプレートと `/plan` step 12.e の「Plan approved」と違う。テンプレートを決める前に作った plan なので実害はない。checklist は digest の対象外

## Coverage gaps

- テストの実行(AC3 の 96 件、`tests/test-new-feature-plan.sh`、`go test ./internal/scaffold/...`、`run-verify.sh` の `all` モードのテスト部分)は /test の担当で、この検証では実行していない
- AC12 の後半(`gh pr create --attach` で PR 本文に PNG が載ること)は /pr の時点でしか確かめられない。gh 2.99.0 以上での `--attach` の挙動、EMU アカウントと private repo での挙動は未確認
- 実際の Chrome での `shot` は self-review が probe で確かめた(exit 0、`#overview` で図だけが写る)。この検証では実行していない
- Linux の `xdg-open` が戻らない場合の扱い(step 12.a はバックグラウンドで実行するよう書いている)と、Linux での `sha256sum` の値は、実機で確かめていない
- Codex CLI が PNG を読めるか、Codex 側で番号つきの選択肢による承認が回るかは未確認(plan の Open questions のとおり)
- 承認の操作(AskUserQuestion での Approve)そのものは成果物に残らないので、`Status` と digest の一致までしか確かめられない

## Verdict

- Verified: AC1、AC2、AC5、AC6、AC8、AC9(コードと手順の記述)、AC2b(digest の再計算)、AC4 と AC7(配置、一覧、テンプレートの内容と、このブランチでの `ralph init`)、AC11(sync 系 2 本と `run-verify.sh` の静的解析の部分)、AC12 の前半。静的解析は全体の範囲で pass
- Partially verified: AC3(ケースの有無を確認。実行は /test)、AC10(`subagent-policy.md` は済み。残りは /sync-docs)
- Not verified: AC12 の後半(/pr で確認)
- 判定: **partial-pass**。AC の違反はない。partial にしたのは、予定どおり後の手順に回した AC10 の残りと AC12 の後半があるため。指摘は MEDIUM 1 件(V-1、digest が AC の印に反応する)と LOW 1 件(V-2、upgrade した下流の plan テンプレート)で、どちらもマージを止めるものではない
