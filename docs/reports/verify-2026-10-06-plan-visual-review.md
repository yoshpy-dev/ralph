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

## Re-run (2026-10-06, after b0ea4a23 / 804c5d84)

- HEAD: 804c5d84(pipeline cycle 1 の中の修正。cycle-count.json は 1 のまま)
- 対象: `git diff ba4bf3ad..HEAD`(12 ファイル、+95/-36)。b0ea4a23 は `scripts/plan-visual.sh` の digest、`tests/test-plan-visual.sh`、`/plan` と `/implement` の SKILL.md(4 か所のミラー)を直した。804c5d84 は plan 本文を直し、承認ゲートを通し直した
- 上の節(Spec compliance 〜 Verdict)は ba4bf3ad の時点の記録として残し、書き換えていない。digest の値 `9c20a2da6606` など古い値は、当時のものとして読む
- Evidence: 同じログの `######## RE-RUN` 以降に追記した

### 指摘の状態

| ID | Status | Evidence |
| --- | --- | --- |
| V-1 | Resolved | `digest_body`(`scripts/plan-visual.sh:272-283`)は、行頭(字下げ可)の `- [x]` / `- [X]` を `- [ ]` に直してから hash する。probe: この plan の AC1 に `[x]` を付けても、AC 12 個に `[X]` を付けても、digest は `d4918bfcec38` のまま。印に加えて本文を 1 文字変えると `5f6f5fc04615` に変わる。Progress checklist の印でも変わらない。ba4bf3ad の plan 本文(印なし)は新しい計算でも `9c20a2da6606` で、印のない plan の値は前と同じ。`implement/SKILL.md:35,71` と `plan/SKILL.md:99-100` が「印は digest を変えない」と書き、plan の Scope、Design decisions、AC2b、Risks も同じ内容に直った。Design decisions の「アーカイブ済み plan に `- [x]` が 1183 か所」は `docs/plans/archive/*.md` を数えて 1183 で一致した。テストは 7 件増えた(`--help` の説明が 1 件。digest が 6 件で、`- [x]` / `- [X]` / 字下げ / タブをまとめた印、印 1 個、印と本文の同時変更、字下げの変更、行の途中と項目の後ろの `[x]`)。実行は /test |
| V-2 | Resolved(PR 本文の一言は /pr で確認) | `plan/SKILL.md:71`(10.d)は節がなければ `## Affected areas` の後に足す、`:99`(12.e)は `- Approved:` の行がなければ `- Status:` の直後に足す、と書いている。どちらもテンプレートの並びと合う。`:68`(10.a)も 10.d を参照する。plan の Rollout notes(148 行目)に seed の扱いと、PR 本文に advisory の取り込みを書くことが入った。PR 本文に実際に入るかは /pr の時点で確かめる |

### AC の再確認

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC2b(改訂後: 印を `- [ ]` とみなす) | Met(再計算) | `./scripts/plan-visual.sh digest` が `d4918bfcec38`。同じ規則(印の正規化を含む)を python で別に実装して求めた値も `d4918bfcec38`。上の probe で、除外部分と印だけが違う plan は同じ値、本文が 1 文字違えば別の値になった。macOS と Linux の一致はテストが見る。Linux の awk(CI の ubuntu-latest)で `[[:space:]]` を含む新しい正規表現が同じ値を出すかは、手元に gawk / mawk がないので未確認で、PR の CI で分かる |
| AC6 | Met | step の順序(8 → 10 → 11 → 12)は変わっていない。12.e の書き込み内容に、古いテンプレートの plan への追記と、印の扱いの説明が加わった。12.d の Needs changes と、10.c / 12.a のブラウザなし・開けない環境の扱いはそのまま |
| AC8 | Met | `implement/SKILL.md:32-38` の比較と 3 択は変わっていない。不一致の説明に「印は digest を変えない」が加わった。`:71` は印付けを許し、それ以外の本文の変更で承認が外れることを書いている |
| AC12 の前半 | Met | plan は `Status: Approved`、`- Approved: 2026-10-06 sha256:d4918bfcec38` で、`./scripts/plan-visual.sh digest` の出力と一致する。Visual review 節に再承認の記録があり、図は変えていないと書いている(どの図も digest の計算方法を描いていないので妥当)。再承認の操作そのものは会話の中のことなので、成果物からは確かめられない |

### 静的解析(再実行)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | 前回と同じく full に切り替わった。`check-sync.sh` は IDENTICAL 164 / DRIFTED 0 / ROOT_ONLY 0、`check-skill-sync.sh` は 13 skill、`check-template-purity.sh` PASS、`gofmt: ok`、golangci-lint `0 issues.`、secret scan `scanned d7877756..804c5d84 against origin/main: clean` |
| `shellcheck -s sh scripts/plan-visual.sh`、`shellcheck tests/test-plan-visual.sh`、`dash -n scripts/plan-visual.sh` | pass(rc 0) | |
| `git diff ba4bf3ad..HEAD --check` | pass | |
| ミラー | Yes | `plan/SKILL.md` と `implement/SKILL.md` は 4 か所とも同一。`scripts/plan-visual.sh` は root と template が同じ blob `8420ba7d`(100755) |

### 文書のずれ(再実行)

- digest の説明は、`plan-visual.sh` の header と `--help`、`plan/SKILL.md` の 12.e / 12.f、`implement/SKILL.md` の step 4 と Plan drift detection、plan の 4 か所で同じ内容になっている。ほかに digest の範囲を説明している文書はない(`.claude/skills`、`.claude/rules`、テンプレート、`README.md`、`AGENTS.md` を grep した)
- AC10 の残り(`AGENTS.md`、`.ralph/core/AGENTS.core.md`、`README.md`、`ralph-workflow.md`)は前回と同じく /sync-docs の担当
- 指摘にはしない残り: 正規化するのは `-` で始まる項目だけで、`*` や `1.` で始まるチェックボックスは印が digest に入る。テンプレートと既存の plan は `-` を使っているので、実害はない

### 判定(再実行)

- 判定: **partial-pass**。V-1 と V-2 は解消し、新しい指摘はない。partial のままにしたのは、前回と同じく AC10 の残り(/sync-docs)と AC12 の後半(/pr での `--attach`)が残っているため
- 未確認: Linux の awk での新しい正規表現の挙動(PR の CI で分かる)、テストの実行(/test)、PR 本文への seed の注記(/pr)

## Cycle 2 (2026-10-06, HEAD a09c057f)

- 対象: `git diff f557d5b9..HEAD`(22 ファイル、+455/-45)。d85e3ab1(test レポート)、4caaec72(sync-docs、plan の AC に印)、91a9c3f6(cross-review triage、ACTION_REQUIRED 1 件)、de99dd6c(`/pr` 5.c の PR の探し方)、459db019(cycle 2 の self-review、LOW の F-6 と F-7)、a09c057f(F-6 と F-7 の修正、plan の再承認)
- d85e3ab1 より後に `scripts/`、`tests/`、`internal/` の変更はない。/test の結果(f557d5b9 で実行)は今のスクリプトとテストにそのまま当てはまる
- 上の節(初回と Re-run)は当時の記録として残し、書き換えていない
- Evidence: 同じログの `######## CYCLE 2` 以降に追記した

### AC の確認

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC9(5.c の改訂後) | Met | `.claude/skills/pr/SKILL.md:40` は、stdout に URL がなければ `gh pr list --head "$(git branch --show-current)" --base <base> --state open --json url --jq '.[0].url'` で探す。出力が空なら PR なしとし、merged / closed の PR や別の base への PR も「PR なし」として扱う。`gh pr view` を使わない理由も書いてある。`<base>` は「Step 3 の `<ref>` から `origin/` を除いたもの」(F-6 の修正)。`scripts/secret-scan-branch.sh:164-169` が出す `<ref>` は `origin/<base>` か `<base>` の 2 形なので、この説明でどちらの場合も合う。plan の AC9 と Design decisions(89 行目)も同じ探し方になった(F-7 の修正)。4 か所のミラーは同一。読み取りだけの gh probe: 存在しないブランチでは `gh pr list … --jq '.[0].url'` が空文字(rc 0)。merged 済みの `refactor/rename-work-skill` は `gh pr list --state open` が空で、`gh pr view refactor/rename-work-skill` は `https://github.com/yoshpy-dev/ralph/pull/202 MERGED` を返した。cross-review の指摘どおりの挙動で、新しい手順はそれを避けられる |
| AC10 | Met | 4caaec72 で残りが入った。root: `AGENTS.md:31-32`、`.ralph/core/AGENTS.core.md:20-21`、`README.md:218-219`(と 225 行目の `/pr`)、`.claude/rules/ralph/ralph-workflow.md:16-24`(と 37 行目の `/pr`)。template: `templates/base/AGENTS.md:27-28`、`templates/base/.ralph/core/AGENTS.core.md`、`templates/base/.claude/rules/ralph/ralph-workflow.md`。`AGENTS.core.md` と `ralph-workflow.md` は root と template が同一。`templates/base/AGENTS.md` の managed block は `templates/base/.ralph/core/AGENTS.core.md` と一致し、root の `AGENTS.md` の harness 一覧も `AGENTS.core.md` と一致する。`README.md` は template 側にない(前回と同じ)。`subagent-policy.md:74` は root と template の両方で承認ゲートを挙げている |
| AC12 の前半 | Met | `./scripts/plan-visual.sh digest` が `4590e050b18a` で、plan の `- Approved: 2026-10-06 sha256:4590e050b18a` と一致する。python で別に計算した値も `4590e050b18a`。承認の操作そのものは会話の中のことなので、成果物から確かめられるのは値の一致まで |
| AC12 の後半 | 未確認(/pr) | 前回と同じ |
| ほかの AC | 変化なし | AC1〜AC8 と AC11 の対象(`scripts/`、`tests/`、`plan` と `implement` の SKILL.md、plan テンプレート、`diagrams.md`、雛形)は、この範囲で変わっていない。plan の AC の印(4caaec72)は AC12 を除いて `[x]` |

### 文書と digest の確認

- V-1 の修正を実際の作業で確かめられた。4caaec72 で plan の AC 12 個に印が付いたが、f557d5b9・4caaec72・de99dd6c・459db019 のどの時点でも digest は `d4918bfcec38` のままで、`- Approved:` の行と合っていた。a09c057f で AC9 と Design decisions の本文が変わり、`4590e050b18a` になって再承認された
- 4caaec72 のほかの文書も手順と合っている。`docs/quality/definition-of-done.md` の承認の項目は `/implement` の 3 択の 2 番目(続けたことを plan に書く)と合う。`docs/recipes/codex-setup.md` の Codex 向けの説明(番号で答える、画像を読めないと自己チェックを省く)は `/plan` の 10.c と 12.c に合う。`docs/architecture/repo-map.md` に `plan-visual.sh` と `.harness/state/plan-visual/` が入った。DoD と codex-setup は root と template が同一
- `gh pr view` の残り: 現役の文書で残っているのは `/pr` 5.c の「使わない」と、plan の Design decisions の同じ趣旨の記述だけ
- Re-run の節で未確認としていた Linux の awk の挙動は、/test が ubuntu:24.04(mawk、gawk)と alpine:3.21(busybox)のコンテナで確かめ、どれも `d4918bfcec38` を出した(`docs/reports/test-2026-10-06-plan-visual-review.md` の「Linux の awk での実行」)。この検証では再実行していない

### 静的解析(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | full に切り替わったのは前回と同じ。`check-sync.sh` IDENTICAL 164 / DRIFTED 0 / ROOT_ONLY 0、`check-pipeline-sync.sh` 全項目 ok、`check-skill-sync.sh` 13 skill、`check-template-purity.sh` PASS、`gofmt: ok`、golangci-lint `0 issues.`、secret scan `scanned d7877756..a09c057f against origin/main: clean` |
| `./scripts/check-pipeline-sync.sh`(単独) | pass(rc 0) | `implement/SKILL.md`、`cross-review/SKILL.md`、`subagent-policy.md`、`definition-of-done.md`、`README.md`、`AGENTS.md` の 6 本すべて `all pipeline steps referenced`。パイプラインの順序は変わっていない |
| `./scripts/check-template.sh` | pass(rc 0) | |
| `git diff f557d5b9..HEAD --check` | pass | |

### 新しく気づいたこと(指摘にはしない)

- `/pr` 5.c は「別の base への open の PR も『PR なし』とみなし、添付なしで作り直す」としつつ、最後に「同じブランチの PR を 2 つ作らない」とも書いている。同じブランチから別の base への open の PR が先にあると、この 2 文は両立しない。起きるのは同じブランチから 2 つの base へ PR を出している場合だけで、この repo の運用では起きないと考えられる。未確認です。直すなら、その場合は止めて報告する、と 1 句足せば済む

### 判定(cycle 2)

- 判定: **partial-pass**。新しい指摘はない。AC10 は root と template の両側で満たし、AC9 は新しい 5.c で満たす。AC12 の前半は digest `4590e050b18a` で一致した。partial のままにしたのは、AC12 の後半(この PR の本文に `--attach` で全体図が載ること)が /pr でしか確かめられないため
- 未確認: AC12 の後半、gh 2.99.0 以上での `--attach` と EMU / private repo での挙動、PR 本文への seed の注記(いずれも /pr)
