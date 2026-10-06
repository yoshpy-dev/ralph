# Test report: plan-visual-review

- Date: 2026-10-06
- Plan: docs/plans/active/2026-10-05-plan-visual-review.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: branch feat/plan-visual-review の HEAD f557d5b9 と base origin/main d7877756 の差分(45 ファイル)。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の AC3・AC4・AC7・AC11 と、Test plan の unit・integration・regression・edge case を確かめた。手元の Mac で `scripts/plan-visual.sh` を本物の Chrome で動かす確認と、Linux の awk 3 種での実行も足した
- Evidence: `docs/evidence/test-2026-10-06-plan-visual-review.log`(`docs/evidence/*.log` は gitignore 対象なので commit しない)。`run-test.sh` 自身が書いたログは `docs/evidence/verify-2026-10-06-024353.log`、`run-verify.sh`(all)のログは `docs/evidence/verify-2026-10-06-024830.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `sh tests/test-plan-visual.sh` | 103 | 103 | 0 | 0 | 5 s、rc 0 |
| `sh tests/test-new-feature-plan.sh` | 21 | 21 | 0 | 0 | 1 s、rc 0 |
| `go test ./internal/scaffold/... -count=1 -v` | top-level 32 件 | 30 | 0 | 2(下の注) | 1 s、rc 0 |
| `./scripts/run-test.sh`(scope は既定の changed) | shell 35 ファイル(1,648 件)、Go 8 パッケージ | すべて | 0 | 1(下の注) | 239 s、rc 0 |
| `./scripts/run-verify.sh`(mode all、scope full) | 静的解析 + shell 35 ファイル(1,648 件)+ Go 8 パッケージ + branch secret scan | すべて | 0 | 1 | 240 s、rc 0 |
| `go test ./... -count=1 -cover` | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 48 s、rc 0 |
| `go build ./...` | 1 | 1 | 0 | 0 | rc 0 |
| 実地確認(手元の Mac、本物の Chrome) | 6 | 6 | 0 | 0 | 下の節 |
| Linux の awk での実行(Docker の ubuntu:24.04 の mawk と gawk、alpine:3.21 の busybox) | `test-plan-visual.sh` 102 件 × 4 回、`test-new-feature-plan.sh` 21 件 × 1 回、plan の digest × 3 | すべて | 0 | 各 1(`shasum` がない) | 下の節 |
| `scripts/plan-visual.sh` の mutation(私が作った 22 件) | 22 | 21 件が red、1 件は等価 | 0 | 0 | 下の節 |

- `run-test.sh` の scope は changed を指定したが、`visual-template.html` が分類できないため full に切り替わった(`Language scope: full fallback (unclassified:.agents/skills/plan/visual-template.html)`)。shell の 35 ファイルは `tests/test-*.sh` の全件で、ディスク上の一覧と実行した一覧を `comm` で比べて差はなかった。`run-verify.sh`(all)でも同じ 35 ファイルが走り、suite ごとの件数は `run-test.sh` と一致した。
- `run-verify.sh`(all)の静的解析の部分では `scripts/check-sync.sh` が IDENTICAL 164 / DRIFTED 0、`scripts/check-skill-sync.sh` が 13 skill で OK、branch secret scan が `scanned d7877756..f557d5b9 against origin/main: clean` だった。
- shell の skip 1 件は `tests/test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」のケース。手元の git が 2.49.0 なので飛ばされる。この変更とは関係がない。
- Go の skip 2 件は `internal/scaffold` の `TestBaseFS_WithMockFS` と `TestAvailablePacks_WithMockFS`。`go test` では `EmbeddedFS` が初期化されないので skip する(`internal/scaffold/embed_test.go:20,35`)。main でも同じ。
- `run-test.sh` と `run-verify.sh` の中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。`templates/base/` は `internal/scaffold` に go:embed されるので、`-count=1` で全パッケージを流し直した。
- shell の件数の数え方: 各 suite の `PASS` / `FAIL` / `SKIP` で始まる行を数え、`PASS: 29` や `PASS: 47 / 47` のような集計行は除いた。

### shell の suite ごとの件数

| Suite | Passed / Total |
| --- | --- |
| `test-agent-models.sh` | 47 / 47 |
| `test-agent-phase-boundaries.sh` | 44 / 44 |
| `test-branch-name.sh` | 26 / 26 |
| `test-check-mojibake.sh` | 15 / 15 |
| `test-check-skill-sync.sh` | 13 / 13 |
| `test-check-template.sh` | 55 / 55 |
| `test-codex-exec-invocation.sh` | 144 / 144 |
| `test-detect-changed-languages.sh` | 83 / 83 |
| `test-detect-languages-terraform.sh` | 8 / 8 |
| `test-ensure-pr-ready.sh` | 7 / 7 |
| `test-ensure-pr-title-prefix.sh` | 13 / 13 |
| `test-gc-artifacts.sh` | 11 / 11 |
| `test-hook-wiring.sh` | 68 / 68 |
| `test-insights-append.sh` | 39 / 39 |
| `test-language-pack-monorepo-roots.sh` | 29 / 29 |
| `test-new-feature-plan.sh`(新規) | 21 / 21 |
| `test-no-loop-references.sh` | 1 / 1 |
| `test-plan-visual.sh`(新規) | 103 / 103 |
| `test-post-edit-verify.sh` | 24 / 24 |
| `test-pre-bash-guard.sh` | 4 / 4 |
| `test-ralph-config.sh` | 19 / 19 |
| `test-ralph-dispatch.sh` | 33 / 33 |
| `test-ralph-worktree.sh` | 143 / 143 |
| `test-run-verify-branch-secret-scan.sh` | 32 / 32 |
| `test-run-verify-scope.sh` | 19 / 19 |
| `test-secret-scan-branch.sh` | 246 / 246(skip 1) |
| `test-secret-scan.sh` | 123 / 123 |
| `test-self-review-scope.sh` | 64 / 64 |
| `test-sync-skills.sh` | 22 / 22 |
| `test-template-purity.sh` | 10 / 10 |
| `test-terraform-gitignore.sh` | 47 / 47 |
| `test-terraform-pack-verify.sh` | 36 / 36 |
| `test-terraform-rule-frontmatter.sh` | 11 / 11 |
| `test-verify-mode-split.sh` | 59 / 59 |
| `test-xreview-helpers.sh` | 29 / 29 |

合計 1,648 件。2026-10-05 の rename-work-skill の 1,523 件から 125 件増えた。内訳は新しい 2 ファイル(103 件と 21 件)と、`test-check-template.sh` の GOLDEN_ENTRIES に `scripts/plan-visual.sh` を足した 1 件で、ほかの suite の件数は変わっていない。

## Plan の edge case と、それを見ているテスト

`tests/test-plan-visual.sh` を読み、plan と team-lead が挙げた edge case ごとにテスト名を対応させた。コードには手を入れていない。

| Edge case | 見ているテスト(`tests/test-plan-visual.sh`) | 結果 |
| --- | --- | --- |
| 空白を含むパス | fixture を `dir with space/` に置き、open の「stdout is only the absolute path」「opener gets the absolute path」、shot の「file URL encodes the space and carries #fragment」(`%20`)。`100% #1?.html` で `%` `#` `?` の符号化も見ている | PASS |
| 存在しない HTML | open の「missing file exits 1」「missing file does not run the opener」、shot の「missing html exits 1」 | PASS |
| ブラウザなし | 「RALPH_PLAN_VISUAL_BROWSER=none exits 2」と stderr の `no browser` | PASS |
| 実行できない `RALPH_PLAN_VISUAL_BROWSER` | 「non-executable browser path exits 2」(実行権のないファイル)、「directory as browser path exits 2」 | PASS |
| フラグメントあり / なし | 「file URL encodes the space and carries #fragment」、「no --fragment means no #」、「--fragment with a leading # does not double it」 | PASS |
| digest: 印を付けただけ / 本文の変更 | 一致の側は「ticked - [x] / - [X] / indented boxes」「one ticked box」「only Status/Approved/Branch and checklist body differ」「Status/Approved/Branch lines removed」「no final newline」。不一致の側は「text after a ticked box still counts」「indentation before a ticked box still counts」「an [x] in the middle of a line」「an [x] later in a list item」「one character in the body」「another header line」「a line in a section after Progress checklist」「an indented - Status: line is not excluded」 | PASS |
| オープナーあり / なし / 失敗(AC1) | stub のオープナー、`RALPH_PLAN_VISUAL_OPENER=none`、失敗するオープナー、PATH だけで選ぶ Darwin の `open` と Linux の `xdg-open`、PATH にオープナーがない場合 | PASS |
| macOS と Linux の SHA-256 ツール(AC2b) | `sha256sum` だけ、`shasum` だけを PATH に置いて既知の値 `1174b9b1faae` を確かめる。どちらもないと exit 1 | PASS(手元は両方ある) |

`tests/test-new-feature-plan.sh` は AC7 を見ている。`new-feature-plan.sh` で作った plan に `- Approved: TBD` と `## Visual review` が入り、`- Approved:` が `- Status:` の直後、`## Visual review` が `## Affected areas` の直後に来る。`feature-plan.md` と `.claude/skills/plan/template.md` が byte 同一であること、作った plan に承認の書き込みをしても digest が変わらず、Visual review 節を変えると digest が変わることも見ている。

テストの前提が外から崩れないことも確かめた。

- 呼び出し側で `RALPH_PLAN_VISUAL_OPENER=/nonexistent RALPH_PLAN_VISUAL_BROWSER=none` を export したまま流しても 103 / 103。suite の先頭の `unset` が効いている
- `sh`、`dash`、`bash`、`zsh --emulate sh` の 4 つで流して、どれも 103 / 103

## 実地確認(手元の Mac、CI ではない)

| 確認 | 結果 |
| --- | --- |
| `./scripts/plan-visual.sh shot .harness/state/plan-visual/plan-visual-review.html <scratchpad>/t-overview.png --fragment overview --width 1150 --height 470 --scale 2` | rc 0、6 s。stdout は PNG の絶対パスだけで、stderr は空。PNG は 2300 × 940(幅と高さの 2 倍)で 193 KB。画像を読み、図 1(全体図)だけが写っていて、見出し・凡例・図 2 は写っていないことを確かめた。`RALPH_PLAN_VISUAL_BROWSER` は指定していないので、`/Applications/Google Chrome.app` の自動検出も通った |
| フラグメントなしの `shot`(`--width 1150 --height 1600`) | rc 0。PNG は 1150 × 1600。ページの見出し、「変わること / 変わらないこと」、凡例、図 1、図 2 が写っている。フラグメントの有無で写る範囲が変わることを本物のブラウザで確かめた |
| `RALPH_PLAN_VISUAL_BROWSER=none` の `shot` | rc 2、stderr に `no browser: RALPH_PLAN_VISUAL_BROWSER=none` |
| `RALPH_PLAN_VISUAL_OPENER=none ./scripts/plan-visual.sh open .harness/state/plan-visual/plan-visual-review.html` | rc 0。stdout は `/Users/.../plan-visual-review/.harness/state/plan-visual/plan-visual-review.html` の絶対パス 1 行。stderr に `open the path above manually` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-05-plan-visual-review.md` | rc 0、`d4918bfcec38`。plan の `- Approved: 2026-10-06 sha256:d4918bfcec38` と一致した |
| 空白を含むパスに写した同じ plan(`sp dir/my plan.md`)の digest / 読めない plan(`chmod 000`)の digest | `d4918bfcec38`(rc 0)/ rc 1 と `plan file not found or not readable` |

team-lead の指示では PNG の出力先を `$TMPDIR` にしていたが、このセッションの決まりに合わせて scratchpad に書いた。出力先の違いだけで、確かめた内容は同じ。本物の `open`(ブラウザのウィンドウが開く)は動かしていない。

## Linux の awk での実行

`git archive HEAD` で `scripts`・`tests`・`docs/plans`・`.claude/skills/plan` などを固め、Docker のコンテナの中で流した。

| 環境 | awk | plan の digest | `test-plan-visual.sh` | `test-new-feature-plan.sh` |
| --- | --- | --- | --- | --- |
| ubuntu:24.04 | mawk(`/usr/bin/mawk`、既定) | `d4918bfcec38` | dash で 102 / 102、bash で 102 / 102 | 21 / 21 |
| ubuntu:24.04 + `apt-get install gawk` | GNU Awk 5.2.1 | `d4918bfcec38` | 102 / 102 | - |
| alpine:3.21 | busybox | `d4918bfcec38` | 102 / 102 | - |

どの環境にも `shasum` がないので、「shasum alone gives the known answer」の 1 件は SKIP になり、件数が 102 になる。`sha256sum` だけの経路は既知の値を出した。verify のレポートが未確認として残していた「Linux の awk で `[[:space:]]` を含む正規表現が同じ値を出すか」は、mawk・gawk・busybox の 3 つで同じ値になった。PR の CI(ubuntu-latest)でも同じ suite が走る。

Linux でのブラウザの自動検出も確かめた。suite は `RALPH_PLAN_VISUAL_BROWSER` を毎回指定するので、自動検出の経路を通らない(Test gaps)。

- alpine にブラウザがない状態で `shot` を実行すると rc 2、stderr に `no Chrome / Chromium browser found; set RALPH_PLAN_VISUAL_BROWSER to its executable`
- PATH に stub の `chromium-browser` だけを置くと、それが使われて rc 0
- `chromium-browser` と `google-chrome` を両方置くと `google-chrome` が使われる(候補の順序どおり)
- `xdg-open` のない Linux で `open` を実行すると rc 0、パスを出して `no opener found` と注記する

## Mutation

suite が `scripts/plan-visual.sh` の誤りを見分けられるかを確かめた。scratchpad に `scripts/plan-visual.sh` と `tests/test-plan-visual.sh` を写し、写しの方だけを perl で 1 か所ずつ書き換えて suite を流した。worktree の追跡ファイルには触れていない。

| Mutation | 結果 |
| --- | --- |
| M01: `- [x]` を `- [ ]` に直す処理を消す | red(印を付けた 2 件) |
| M02: 字下げした印を直さない | red(字下げ・タブの印) |
| M03: `[X]` を直さない | red |
| M04: Progress checklist を除かない | red(既知の値など 3 件) |
| M05: Progress checklist の後ろの節も除く | red(checklist の後ろの節の変更など 4 件) |
| M06: 字下げした `- Status:` も除く | red |
| M07: `- Branch:` を除かない | red(5 件) |
| M08: `%` を最後に符号化する | red(URL の 4 件) |
| M09: `#` を符号化しない | red |
| M10: オープナーが失敗したら exit 1 | red |
| M11: オープナーの stdout を stdout に流す | red |
| M12: ブラウザがないとき exit 1 | red(3 件) |
| M13: 前の PNG を消さない | red(古い PNG の 3 件) |
| M14: PNG と HTML が同じファイルかを確かめない | red(4 件) |
| M15: `RALPH_PLAN_VISUAL_BROWSER` の実行権を確かめない | red |
| M16: digest を 11 桁にする | red(スクリプト自身の桁数の検査が exit 1 にし、suite は最初の digest で `set -e` により止まる) |
| M17: `--width 0` を通す | red |
| M18: 相対パスを絶対パスにしない | red(6 件) |
| M19: `--fragment` の先頭の `#` を外さない | red |
| M20: 既定の高さを 2000 にする | red |
| M21: `open` でファイルがなければ exit 0 | red |
| M22: awk の `print` を `printf "%s\n"` 相当に替える | 等価(挙動が変わらない書き換えだった) |

意味のある 21 件はすべて red になった。M22 は末尾の改行の扱いを崩すつもりで作ったが、awk の `print` は入力の最後に改行がなくても改行を付けるので、書き換え前と同じ出力になる。等価な mutation として数から外した。M16 は FAIL の行を出さずに suite が止まる形で red になる。digest の計算が壊れたときは、その後の digest のケースが走らないまま rc 1 で終わる。

## Coverage

- Statement: Go は `go test ./... -count=1 -cover` で、`internal/cli` 84.7%、`internal/config` 92.3%、`internal/insights` 86.1%、`internal/org` 90.8%、`internal/org/driver` 92.0%、`internal/org/protocol` 97.9%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%。2026-10-05(rename-work-skill)の記録とすべて同じ値。この変更の Go の差分は `internal/scaffold/embed_test.go` の必須一覧に 1 行足しただけ
- Branch / Function: Go の標準ツールでは測っていない。shell は計測ツールがないので、ケースの範囲で見ている
- Notes: `scripts/plan-visual.sh` の分岐のうち、suite が通らないのは、ブラウザの自動検出(`/Applications` の候補、PATH の 4 つの名前、どれもない場合)、`<png>` がディレクトリの場合(`output path is a directory`)、`digest` の「読めないファイル」、`abs_path` のルート直下の分岐、SHA-256 の出力の形式の検査(M16 でだけ通った)。自動検出、ディレクトリ(`x.png` という名前のディレクトリで rc 1)、読めないファイルは手で確かめた。suite が通る分岐は、mutation 21 件がすべて red になったので、誤りを見分けている

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 雛形が scaffold-green のまま(`check-template.sh` の required_files) | 維持 | `tests/test-check-template.sh` 55 / 55。GOLDEN_ENTRIES に `scripts/plan-visual.sh` が加わった |
| 雛形の必須スクリプトが揃っている | 維持 | `go test ./internal/scaffold/...` rc 0(`requiredTemplateScripts` に `plan-visual.sh`) |
| skill ミラーの生成とずれの検査 | 維持 | `tests/test-sync-skills.sh` 22 / 22、`tests/test-check-skill-sync.sh` 13 / 13、`run-verify.sh` の中の `check-skill-sync.sh` が 13 skill で OK |
| 雛形に meta-repo 専用の内容が混ざらない | 維持 | `tests/test-template-purity.sh` 10 / 10 |
| `/plan` step 11.c を参照する codex advisory の呼び出し | 維持 | `tests/test-codex-exec-invocation.sh` 144 / 144(step 番号を変えずに図解と承認を足した) |
| self-review F-2 の修正(空の `<png>`、`.png` 以外、HTML と同じパスで HTML を壊さない) | 維持 | 「empty png path」「png path not ending in .png」「png path resolving to the html」の各 3 件と「rejected png paths do not start the browser」が PASS。M14 で red |
| verify V-1 の修正(印を付けても digest が変わらない) | 維持 | 印の 2 件が PASS、M01〜M03 で red。この plan の digest は `d4918bfcec38` のまま |

## Test gaps

- ブラウザの自動検出の経路(`/Applications` の 2 候補、PATH の `google-chrome` / `google-chrome-stable` / `chromium` / `chromium-browser`、どれもない場合の exit 2)は suite にない。suite は毎回 `RALPH_PLAN_VISUAL_BROWSER` を指定する。macOS の `/Applications/Google Chrome.app` は実地確認で、Linux の PATH の名前とブラウザなしは Docker で手で確かめた。PATH を絞って stub を置くテストを足せば閉じられる。マージを止める理由にはしない
- `digest` の「ファイルはあるが読めない」と、`shot` の `<png>` がディレクトリの場合は suite にない(存在しないファイルと、存在しないディレクトリだけを見ている)。どちらも手で rc 1 を確かめた
- digest の計算が壊れたとき、suite は最初の `_base_digest="$(...)"` で `set -e` により止まり、集計行を出さない(M16)。rc は 1 になるので見落とされはしないが、どのケースが落ちたかは出ない
- `shasum` だけの経路は手元の Mac でしか通らない。CI の ubuntu と今回の Docker では SKIP になる。`sha256sum` だけの経路は両方で通る
- 本物の `open` / `xdg-open` は動かしていない(手元でブラウザのウィンドウが開くため)。Linux の `xdg-open` がブラウザを閉じるまで戻らないかは、plan のメモのとおり未確認
- AC12 の後半(`gh pr create --attach` で PR 本文に PNG が載ること)は /pr の時点でしか確かめられない。手元の gh は 2.96.0
- Codex CLI が PNG を読めるか、Codex 側の番号つきの選択肢で承認が回るかは未確認(plan の Open questions)
- `/plan` の承認ゲートと `/implement` の digest の確認は SKILL.md の手順なので、テストでは確かめられない。確かめたのは、手順が使う `plan-visual.sh` の挙動と、テンプレートに行と節が入ることまで

## Verdict

- Pass: AC3(`tests/test-plan-visual.sh` 103 / 103 が plan の挙げたケースをすべて含み、`./scripts/run-test.sh` で rc 0)。AC4(`go test ./internal/scaffold/...` rc 0、`check-sync.sh` は DRIFTED 0)。AC7(`tests/test-new-feature-plan.sh` 21 / 21)。AC11(`run-verify.sh` の all モードが rc 0、その中の `check-skill-sync.sh` と `check-sync.sh` も OK)。`go test ./... -count=1` rc 0。実地確認 6 件、Linux の awk 3 種、mutation 21 / 21
- Fail: なし
- Blocked: なし

判定は pass。/pr に進んでよい。

## Cycle 2 (2026-10-06, HEAD 2b843ed9)

- 対象: `git diff d85e3ab1..HEAD`(6 コミット、22 ファイル、+298/-45)。4caaec72(sync-docs)、91a9c3f6(cross-review の triage)、de99dd6c と a09c057f(/pr step 5.c の文と、plan の直しと再承認)、459db019 と 2b843ed9(レポート)
- `git diff --stat d85e3ab1..HEAD -- scripts tests internal cmd templates/base/scripts` は空だった。スクリプト・テスト・Go のコードは変わっておらず、変わったのはスキルの本文・ドキュメント・plan・レポートだけ
- 上の節(Test execution 〜 Verdict)は d85e3ab1 の時点の記録として残し、書き換えていない
- Evidence: 同じログの `######## CYCLE 2` 以降に追記した。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-06-040853.log`

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | shell 35 ファイル(1,648 件)、Go 8 パッケージ | すべて | 0 | 1(cycle 1 と同じ git<2.41 のケース) | 255 s、rc 0 |
| `go test ./... -count=1 -cover` | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 50 s、rc 0 |
| plan の digest(macOS の BSD awk、Docker の mawk と busybox) | 3 | 3 | 0 | 0 | rc 0 |
| /pr step 5.c の PR の有無の確かめ方(読み取りだけ、gh 2.102.0) | 5 | 5 | 0 | 0 | 下の節 |

- shell の 35 ファイルはディスク上の `tests/test-*.sh` の全件で、`comm` で比べて差はなかった。suite ごとの件数は cycle 1 の表と 1 件も違わない(集計結果を `diff` で比べた)
- Go の coverage は `internal/cli` 84.7%、`internal/config` 92.3%、`internal/insights` 86.1%、`internal/org` 90.8%、`internal/org/driver` 92.0%、`internal/org/protocol` 97.9%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%。cycle 1 と同じ値
- cycle 1 の mutation、Linux の awk での suite の実行、Chrome での実地確認は流し直していない。`scripts/plan-visual.sh` と `tests/` が変わっていないので、結果は cycle 1 のまま使える

### plan の digest

`./scripts/plan-visual.sh digest docs/plans/active/2026-10-05-plan-visual-review.md` は `4590e050b18a`(rc 0)で、plan の `- Approved: 2026-10-06 sha256:4590e050b18a` と一致した。`git archive HEAD` で固めた同じ plan を Docker の ubuntu:24.04(mawk)と alpine:3.21(busybox)で計算しても `4590e050b18a` だった。

### /pr step 5.c の PR の有無の確かめ方

de99dd6c と a09c057f で、`gh pr create` が非ゼロで終わったときに PR の有無を確かめる方法が `gh pr view` から `gh pr list --head <branch> --base <base> --state open --json url --jq '.[0].url'` に変わった。この repo で読み取りだけの確認をした。gh は 2.102.0、使ったアカウントは gh の active account のまま。

| 確認 | 結果 |
| --- | --- |
| `gh pr list --head feat/plan-visual-review --base main --state open --json url --jq '.[0].url'` | rc 0、出力は空(このブランチの PR はまだない) |
| `gh pr list --head refactor/rename-work-skill --base main --state open ...`(マージ済みのブランチ) | rc 0、出力は空 |
| `gh pr view refactor/rename-work-skill --json state` | `MERGED`(#202)。`gh pr view` を使うとマージ済みの PR が見つかってしまう。5.c が `gh pr view` を使わないとした理由の再現 |
| 対照: 同じ問い合わせで `--state all` | `MERGED https://github.com/yoshpy-dev/ralph/pull/202`。PR があれば URL が出る。マージ済みの PR を外しているのは `--state open` |
| 対照: `--base no-such-base --state all` | rc 0、出力は空。base が違う PR は数えない |

PR がないとき、`--jq '.[0].url'` は `null` ではなく空の出力になる。5.c の「empty output means no such PR」と合う。repo に open の PR が 1 件もないので、`--state open` で URL が返る場合は確かめられなかった。代わりに `--state all` の対照で、同じ問い合わせが URL を返す形であることを見た。

### Test gaps(cycle 2)

- 5.c の確かめ方を自動で見るテストはない。SKILL.md の手順なので、上の読み取りの確認までしかできない。open の PR が URL で返る場合は、この PR を作る /pr の時点で分かる
- cycle 1 の Test gaps はそのまま残る(ブラウザの自動検出、`<png>` がディレクトリ、読めない plan、AC12 の後半、Codex での PNG の読み取り)

### Verdict(cycle 2)

- Pass: `./scripts/run-test.sh`(full)rc 0、shell 1,648 件と Go 8 パッケージ。`go test ./... -count=1` rc 0。plan の digest が承認の値 `4590e050b18a` と一致(BSD awk、mawk、busybox)。5.c の確かめ方が、このブランチとマージ済みのブランチの両方で空を返した
- Fail: なし
- Blocked: なし

判定は pass。/pr に進んでよい。
