# local-branch-secret-scan

- Status: Done (PR #177)
- Owner: Claude Code
- Date: 2026-09-20
- Related request: CI の `verify` ジョブは `./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/<base>)..HEAD"` で branch の履歴全体(`git log -p` の追加行)を scan するが、ローカルには同じ scan を走らせる手順がない。PR #168 では、テストの偽の値と、それを許可するために足した `.gitallowed` の行自体が CI で初めて掛かり、push 後に 2 回失敗した。履歴を読む scan なので、push 後に気づくと fixture を直しても過去の commit で掛かり続ける(issue #169)
- Related issue: 169
- Type: feat
- Branch: feat/local-branch-secret-scan

## Objective

CI と同じ「branch の履歴込みの secret scan」を、push の前にローカルで必ず通るようにする。`/pr`(push の直前)と `./scripts/run-verify.sh` の両方から、1 つのスクリプトで実行する。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | branch scan | `scripts/secret-scan-branch.sh`(新規)+ `templates/base/scripts/` のコピー | **現在の契約(要約)**: 既定 mode はどの状態(scan できない・nothing to scan・検出なし)でも理由を 1 行出し、検出以外は exit 0(失敗にしない)。`--strict` は「scanned, clean」のときだけ exit 0 で、それ以外(scan できない、HEAD が base branch、range が空を含む nothing to scan)はすべて exit 3。allowlist は常に HEAD にコミット済みの `.gitallowed` を使う。以下は改訂の経緯。base branch を解決し(`GITHUB_BASE_REF` があればそれ、なければ `scripts/xreview-helpers.sh` の `detect_base_branch`)、`origin/<base>`(なければローカルの `<base>`)との merge-base から `HEAD` までを `./scripts/secret-scan.sh --range` で scan する。exit 0 = 問題なし、または理由を 1 行出して skip。exit 1 = 検出。exit 2 = 使い方の誤り。skip するのは: git の作業ツリーでない、HEAD が base branch そのもの、base の ref がどちらもない、merge-base が取れない、range に commit がない、scanner がない。skip は既定では失敗にしない(`run-verify.sh` 用)。**Codex advisory による改訂**: `--strict` を付けると、「scan できなかった」(scanner がない、git の作業ツリーでない、base の ref がどちらもない、merge-base が取れない)は exit 3 で終わる。問題なし(exit 0)と区別するため。**self-review cycle 1 による改訂**: strict の exit 0 は「scan して問題なし」だけを意味する。「HEAD が base branch」と「range が空」も strict では exit 3 にする(当初は exit 0 としていたが、base の指定を現在の branch と同じにする、`origin/<base>` がない状態でローカルの base を branch の先端まで進める、のどちらでも、未 push のコミットを抱えたまま exit 0 になることを reviewer が一時 repo で実証した。`/pr` の流れでは range が空であること自体が異常なので、止めて調べる)。既定の mode では従来どおり理由を出して exit 0。allowlist は常に HEAD にコミット済みの `.gitallowed` を使う(`git show HEAD:.gitallowed` を一時ファイルに書き出して scanner に渡す。HEAD にファイルがなければ空)。作業ツリーの未コミットの変更や `RALPH_SECRET_ALLOWLIST` の上書きは、CI が読まないので無視し、無視したことを 1 行出す。**cross-review cycle 1 による改訂**: (a) merge-base と range には、存在を確認した完全な ref 名(`refs/remotes/origin/<base>` / `refs/heads/<base>`)をそのまま使う。短い名前は表示だけ(git は短い名前を tag や `refs/heads/origin/<base>` に先に解決し得るため)。(b) HEAD の `.gitallowed` が symlink(tree の mode 120000)なら、コミット済みの tree の中で 1 段だけ解決して読む。解決できない、または blob でも symlink でもない場合は空の allowlist で scan し、その旨を 1 行出す(例外が減る方向なので安全側) |
| 2 | run-verify | `scripts/run-verify.sh` + `templates/base/scripts/` のコピー | `HARNESS_VERIFY_MODE` が `static` か `all` のとき、branch scan を 1 つの step として実行する。検出したら全体を失敗にする(`status=1`)。「言語の verifier が 1 つも走らなかった」の判定(`ran_any`)には数えない。緊急時の回避として `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` で飛ばせる(飛ばしたことを 1 行出す) |
| 3 | /pr | `.claude/skills/pr/SKILL.md` + `.agents/skills/pr/SKILL.md` + `templates/base/` の 2 面 | **Codex advisory による改訂**: Pre-checks ではなく Steps に入れる。未コミットの変更をコミットする手順の後、`git push` の直前に「`./scripts/secret-scan-branch.sh --strict` を実行し、exit 0 以外なら push しない」を置く(事前チェックの後にコミットが増えるので、scan は実際に push する HEAD に対して行う)。exit 1(検出)と exit 3(scan できない)それぞれの対処を 1〜2 行で書く |
| 4 | 案内 | `scripts/secret-scan.sh` + template のコピー、`docs/quality/quality-gates.md` | 検出時の案内文に 1 行: `.gitallowed` に足す行は、その行自体が scanner のパターンに一致しない形で書く(例: キー名を `api_ke[y]` のように括弧式にする)。range の scan は `.gitallowed` を足した commit の追加行も読むため。quality-gates にローカルの branch scan を追記 |
| 5 | 必須ファイル | `scripts/check-template.sh` + template のコピー | 必須ファイルの一覧に `scripts/secret-scan-branch.sh` を追加 |
| 6 | テスト | `tests/test-secret-scan.sh`(または新規 `tests/test-secret-scan-branch.sh`)、必要なら scaffold のテスト | 下の Test plan |
| 7 | 文書 | `README.md` / `AGENTS.md` の scripts の説明(該当箇所があれば)、`docs/recipes/`(該当があれば) | 新しいスクリプトの 1 行 |

## Non-goals

- scanner のパターンや `.gitallowed` の仕組みそのものの変更
- pre-push の git hook の追加(`/pr` と `run-verify.sh` で足りる。hook はユーザーの `.git/hooks` に入れる別の導入手順が要る)
- CI の workflow の変更(CI は今の step のまま)
- 既存の履歴にある検出済みの文字列の整理

## Assumptions

- `scripts/xreview-helpers.sh` は template にも入っており、`detect_base_branch` をそのまま使える
- `scripts/` と `templates/base/scripts/` は `check-sync.sh` が同一であることを確認する(新規ファイルも両方に置く)
- `git log -p` の range scan は branch の大きさに比例するが、通常の task branch では 1 秒未満

## Affected areas

- `scripts/`(新規 1、変更 3)と `templates/base/scripts/` のコピー
- `/pr` skill の 4 面
- `tests/`、`docs/quality/quality-gates.md`
- scaffold されたプロジェクト: `ralph upgrade` 後、`run-verify.sh` が branch scan を実行するようになる(CI と同じ基準。回避は環境変数)

## Design decisions

- Critical forks: None。
- **Codex plan advisory(2026-09-20、HIGH 1 / MEDIUM 1、ユーザー決定: 対応案で plan を更新)**: (1) skip が exit 0 だと、scan できていない状態でも `/pr` の関門を通り、未 scan の履歴が remote に出る → `--strict` を足し、`/pr` は strict で呼ぶ。(2) scan が実際に push する内容に紐づいていない(`/pr` は事前チェックの後にコミットする。scanner は作業ツリーや環境変数の allowlist を読む)→ `/pr` の scan を push の直前に移し、allowlist は HEAD にコミット済みのものだけを使う。
- 1 つのスクリプトに寄せる。`/pr` と `run-verify.sh` と人が、同じコマンドで同じ結果を得る。base の解決や skip の条件を 2 箇所に書かない。
- `run-verify.sh` から呼ぶときの skip は失敗にしない。ここでの scan は「push 前に気づく」ための早期検出で、最終の関門は `/pr` の strict な scan と CI。fetch していない、base の ref がない、といった環境の事情で verify 全体を落とさない。代わりに skip の理由を必ず 1 行出す。
- `run-verify.sh` では既定で有効にする。無効が既定だと #168 と同じ見落としが起きる。scaffold 先でも CI と同じ基準なので、新しい種類の失敗は増えない。

## Acceptance criteria

- [ ] AC-1: feature branch の途中の commit に secret 風の文字列があり、後の commit で消してあっても、`./scripts/secret-scan-branch.sh` は exit 1 になる(履歴を読むことの確認)
- [ ] AC-2: 問題のない feature branch では exit 0。base branch 上、base の ref がない、merge-base が取れない、range が空、git の外、では理由を 1 行出して exit 0
- [ ] AC-2b: `--strict` では、scanner がない、git の作業ツリーでない、base の ref がどちらもない、merge-base が取れない、HEAD が base branch、range が空、のいずれも exit 3 で理由を出す。strict の exit 0 は「scan して問題なし」のときだけ(self-review cycle 1 による改訂)
- [ ] AC-2c: allowlist は HEAD にコミット済みの `.gitallowed` だけが効く。作業ツリーで未コミットの行を足しても、`RALPH_SECRET_ALLOWLIST` で別のファイルを指しても検出は消えず、無視した旨が 1 行出る。コミットすれば効く
- [ ] AC-3: `origin/<base>` があればそれを、なければローカルの `<base>` を使う。`GITHUB_BASE_REF` があればそれを base にする
- [x] AC-3b: base と同じ名前の tag や、`origin/<base>` という名前のローカル branch があっても、merge-base は存在を確認した branch の ref から取る(cross-review cycle 1 の AR-2)
- [x] AC-2d: HEAD の `.gitallowed` が symlink のとき、コミット済みの tree の中のリンク先を allowlist として読む。リンク先が読めない場合は空の allowlist で scan し通知を出す(cross-review cycle 1 の AR-1)
- [ ] AC-4: `./scripts/run-verify.sh` は、`static` / `all` のとき branch scan を実行し、検出したら非 0 で終わる。`test` のときは実行しない。`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` で飛ばせ、その旨を出力する。docs だけの変更の判定や既存の出力は変わらない
- [ ] AC-5: `.gitallowed` に、scanner のパターンに自分自身が一致する行を足した commit を含む branch は exit 1 になり、案内文が書き方の注意を示す。括弧式で書いた行なら exit 0
- [ ] AC-6: `/pr` の Steps の「コミットの後、push の直前」に strict な branch scan が入り、exit 0 以外では push しないと明記され、skill の 4 面が一致する(`check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` が通る)
- [ ] AC-7: `scripts/` と `templates/base/scripts/` の該当ファイルが同一で、`check-template.sh` の必須一覧に新しいスクリプトが入っている。scaffold / upgrade の Go テストが通る
- [ ] AC-8: `./scripts/run-verify.sh` と `./scripts/run-test.sh` が green。PR 本文に `Closes #169`

## Implementation outline

1. Slice A: `secret-scan-branch.sh` とテスト(AC-1〜AC-3、AC-5)。template へのコピーと必須一覧(AC-7)
2. Slice B: `run-verify.sh` への組み込みとテスト(AC-4)、scanner の案内文(AC-5 の後半)
3. Slice C: `/pr` の Steps(push の直前、4 面)と `quality-gates.md` などの文書(AC-6)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(shellcheck を含む)、`sh -n` / `bash -n`、`check-skill-sync.sh`、`check-sync.sh`、`check-template-purity.sh`、`check-template.sh`
- Spec compliance criteria to confirm: AC-1〜AC-8。Non-goals(scanner のパターン、hook、CI は不変)
- Documentation drift to check: `quality-gates.md`、`/pr` skill、README / AGENTS.md の scripts の説明
- Evidence to capture: verify / test のログ、AC-1 の実演(一時 repo)

## Test plan

- Unit tests: 一時 repo を作る shell テスト。検出あり(途中の commit、後で削除)、問題なし、base branch 上、origin なし(ローカルの base に fallback)、base がどちらもない、range が空、`GITHUB_BASE_REF`、git の外、`.gitallowed` の自己一致と括弧式
- Integration tests: `run-verify.sh` の 3 つの mode と環境変数での skip。検出時の exit code
- Regression tests: 既存の `tests/test-secret-scan.sh`、`run-verify.sh` を使う既存のテスト、scaffold / upgrade の Go テスト
- Edge cases: strict での各「scan できない」条件、未コミットの allowlist と環境変数の上書き、detached HEAD、base と同じ commit にいる feature branch(range が空)、`RALPH_XREVIEW_BASE` の上書き、空白を含むパス
- Evidence to capture: `./scripts/run-test.sh` の出力

## Risks and mitigations

| リスク | 影響 | 対策 |
|---|---|---|
| テストの fixture 自体が、この repo の CI の secret scan に掛かる | #168 の再発 | fixture の文字列はテストの実行時に組み立てる(ソースに secret 風の代入を書かない)。既存の `tests/test-secret-scan.sh` の書き方に合わせる。push 前に range scan を実行する |
| scaffold 先で `run-verify.sh` が新しく失敗する | 利用者の verify が赤くなる | CI と同じ基準なので新しい種類の失敗ではない。回避の環境変数と、skip の理由の出力を用意する |
| base の解決を誤り、巨大な range を scan して遅くなる | verify が遅い | merge-base が取れなければ skip。range は merge-base から HEAD まで |
| `run-verify.sh` の `ran_any` / docs-only の判定を壊す | 既存の出力が変わる | branch scan は `ran_any` に数えない。既存のテストで確認 |

## Rollout or rollback notes

- 追加だけの変更。revert で元に戻る。緊急時は `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1`

## Open questions

- なし

## Deviation notes

- 2026-09-21 work: Slice A(2ca73aa)、B(335dddf)、C(d4b9336)は implementer に委譲。A: `scripts/secret-scan-branch.sh`(base の解決、merge-base..HEAD の scan、既定は skip で exit 0、`--strict` は「scan できない」で exit 3、allowlist は HEAD にコミット済みの `.gitallowed` だけ)とテスト 44 件。「base branch 上」と「range が空」の判定は scanner の有無の確認より前に置いた(strict でも exit 0 にするため)。template へのコピー、`check-template.sh` の必須一覧、`internal/scaffold/embed_test.go` の必須スクリプトの一覧に追加。B: `run-verify.sh` が `static` / `all` で branch scan を実行(`ran_any` には数えない、環境変数で skip、スクリプトがなければ 1 行出して続行)、scanner の案内文に `.gitallowed` の書き方を 3 行。テスト 21 件。テストは CI の `GITHUB_BASE_REF` と `run-test.sh` が export する `RALPH_VERIFY_SCOPE` を継承しないよう、入れ子の実行の前に固定する。C: `/pr` の Steps に strict な scan(コミットの後・push の直前)を追加して以降の番号を振り直し、既存の誤り(Step 1 が archival を Step 6 と書いていた)も直した。`quality-gates.md`(template 側は issue 番号を書かない)、AGENTS.md の scripts の説明。逸脱: 自己一致する allowlist の行のテストは、2 コミットの形にした(HEAD の `.gitallowed` がその行を含む間は、その行自体が許可されるため。後のコミットで行が変わると過去の追加行が検出される: PR #168 と同じ形)
- 2026-09-21 work: orchestrator が HEAD 一致・porcelain 空・スクリプト全文と差分を確認。`/pr` は PR 作成後にも push する(plan の確定とアーカイブ)ので、「以降の push の前にも毎回実行する」の 1 文を skill の 4 面に追記(eda0248)。新しいテスト 2 本と、この branch 自身への `--strict` の実行が pass
- 2026-09-21 self-review cycle 1(`docs/reports/self-review-2026-09-21-local-branch-secret-scan.md`、0d7b4d0): マージ可(MEDIUM を先に修正)、MEDIUM 2 / LOW 10。4 面の skill と scripts のコピーが byte 一致、fixture が実行時組み立て、linked worktree・detached HEAD・空白入りパスで動くことは確認された。全件を同 cycle 内で修正する。M1 branch scan だけが失敗し言語の verifier が 1 つも走らなかった場合、exit は非 0 なのに締めの行が「docs だけの変更」で終わる(テストもその挙動を固定し、コメントは逆のことを書いていた)。M2 strict の exit 0 が「scan して問題なし」と「scan するものがない」を兼ねていた → strict では後者も exit 3 にする(AC-2b と Scope 1 を改訂。当初の案は orchestrator の提案で、すり抜けの実証を受けて安全側に変更)。LOW: skip の環境変数は `1` のときだけ効かせる、scanner の exit 1 以外を `findings` と表示しない、signal の trap で exit する、exit 3 の説明、`xreview-helpers.sh` を必須一覧に追加、archive の手順に再 scan を明記、テストの理由行の固定と helper の修正、`quality-gates.md` の issue 番号
- 2026-09-22 work: Slice D は implementer に委譲(3afac6e、15 ファイル)。self-review cycle 1 の 12 件を修正。`run-verify.sh` は scan の失敗を `branch_scan_failed` で持ち、`ran_any` に関係なく末尾に失敗行を出す。skip の環境変数は `1` のときだけ効く。`secret-scan-branch.sh` は strict で「scan するものがない」も exit 3(reviewer のすり抜け 2 通りをテストに追加)、scanner の exit 1 以外は「scanner failed with exit <rc>」と表示、signal の trap は exit する。`xreview-helpers.sh` を必須一覧に追加し、そのヘッダーの「/cross-review だけが使う」という記述を直した。`/pr` の skill は exit 0 の意味(scan して問題なし)と exit 3 の 2 つの場合を書き、archive の手順に「コミット、再 scan、push」を明記。テストは exit 0 の理由行(`against <ref>: clean`)を固定。修正前のスクリプトに戻すと、新旧のテストがそれぞれ 8 件落ちることを implementer が確認。逸脱: SIGINT はこの実行環境では再現できず(background の shell は INT を無視する)、TERM だけを実機で確認した(exit 143、一時ファイルは削除)。INT / HUP は同じ形の trap。orchestrator が HEAD 一致・porcelain 空・差分を確認
- 2026-09-22 メモ: この Slice は handoff から完了まで約 30 時間かかったが、ファイルの更新時刻から見て実働は断続的な数十分で、間の空白(20 時間と 5 時間)はセッションが動いていなかった時間とみられる(原因は未確認)。固まったプロセスはなかった
- 2026-09-22 `/verify`(`docs/reports/verify-2026-09-22-local-branch-secret-scan.md`、186183c): pass。AC-1〜6 は満たす、AC-7 の static 半分と AC-8 は `/test` へ委譲。Scope 1 の改訂を重ねた書式について「最終的な契約は正しいが読みにくい」という指摘のみ(不合格ではない)
- 2026-09-24 `/test`(`docs/reports/test-2026-09-24-local-branch-secret-scan.md`、ca3d824): pass。shell + Go 一式 green、環境行列・repeat・concurrency・red/green mutation 6 件・実演すべて通過。テスト 2 本を追加(`test_base_name_with_slash`、`test_mode_test_never_runs_even_with_a_finding`)。gap 2 件を記録: (a) `check-template.sh` の `required_files` 一覧に回帰テストがない(mutation 6f で確認)、(b) `.gitallowed` が symlink のときの「未コミット無視」通知が誤発火しうる(cosmetic)
- 2026-09-24 `/sync-docs`(cycle 1): 対象範囲外の主要な文書(README.md、definition-of-done.md、git-commit-strategy.md の Enforcement 文、ralph-workflow.md、work/SKILL.md、docs/recipes/、`.ralph/core/AGENTS.core.md`)を確認したが、いずれも既存の記述のままで正しい(stale なし)。`docs/tech-debt/README.md` に (a) を新規行として追加((b) は cosmetic・非 AC のため report のみに記録し register には入れない、tester の判断を踏襲)。Scope 1 の冒頭に「現在の契約(要約)」の 1 文を追加し、改訂の経緯はそのまま残した。この branch の insight event が日付ごとに 3 ファイル(2026-09-20/22/24)に分かれている点は `insights-append.sh` の仕様どおり(呼び出し時の UTC 日付でファイル名を決める)で、`docs/insights/README.md` の「per-task files」の趣旨(ブランチ間の merge 安全性)はファイルが複数でも成立するため、統合しない
- 2026-09-24 cross-review cycle 1(`docs/reports/cross-review-triage-local-branch-secret-scan.md`、0c4f84b): Codex の指摘 2 件(どちらも P2)を ACTION_REQUIRED と判定。どちらも「strict で clean なのに CI では検出」という、この plan が排除する向きの誤り。AR-1 `.gitallowed` がコミット済みの symlink だと `git show` がリンク先のパスを返し、それが正規表現として効く。AR-2 base ref の存在確認は完全な ref 名で行うのに merge-base には短い名前を渡すため、base と同名の tag があると range がずれる(Codex が再現)。ユーザー決定(AskUserQuestion): 2 件とも修正して cycle 2/2 としてパイプラインを再実行
- 2026-09-24 work(cycle 2): Slice E は implementer に委譲(dbba825、3 ファイル)。AR-2: merge-base には存在を確認した完全な ref 名(`refs/remotes/origin/<base>` / `refs/heads/<base>`)を渡し、短い名前は表示だけに使う。AR-1: HEAD の `.gitallowed` は `git ls-tree --full-tree` の mode で読み分ける(通常ファイルはそのまま。symlink はリンク先を、絶対パスと `..` を拒否したうえでコミット済みの tree の中で 1 段だけ解決する。それ以外は空の allowlist で scan して通知を出す)。orchestrator の審査(fixture repo での probe)で追加の穴が 2 件見つかり、同じコミットに amend で取り込んだ: (1) `git ls-tree` は cwd 相対なので、サブディレクトリから実行すると allowlist が黙って空になり誤検出と誤った「未コミット無視」通知が出る(Slice E の前は cwd に依存しなかった)→ `--full-tree` にした。(2) リンク先がディレクトリで末尾に `/` が付いていると `ls-tree` が子を列挙して mode の検査を通り、`git show HEAD:<dir>/` が tree の一覧を exit 0 で出して、その行が regex として検出を隠す(fail-open)→ `ls-tree` の結果が 1 行で、そのパスが問い合わせたパスと一致するときだけ mode を採る。テストは 77 件から 87 件へ(tag の影、`refs/heads/origin/<base>` の影、symlink の解決・不一致・dangling・リンク先名が fixture と一致、サブディレクトリ、`rules/`、`.`)。修正前のスクリプトに戻すと新しいテストが落ちることを implementer が確認(1 回目 10 件、2 回目 5 件)。orchestrator は HEAD 一致・porcelain 空・差分・template の byte 一致、probe 4 通り(サブディレクトリ、`rules/`、`.`、`rules`)、shell テスト 3 本、`run-verify.sh`、strict scan を確認
- 2026-09-24 self-review cycle 2(同じ report の `## Cycle 2`、973bd15): マージ可(修正後)、MEDIUM 1 / LOW 5。C2-M1 `allowlist_ls_tree_mode` のコメントが「末尾 `/` なしのディレクトリも 1 行・パス一致の検査で弾く」と書いていたが、それを弾くのは mode の case(040000 は 100644|100755 にない)で、その行を pin するテストがなかった(mode の case に 040000 を足す mutation でテストが緑のまま strict が clean になることを reviewer が実証)。LOW: 通常ファイルの `git show` が `set -e` で無防備、`./` の除去より前に安全性を検査していた(`.//x` が `/x` になる)、複数行 guard の役割の説明、テスト名が `ar1`/`ar2`(cross-review の triage の id は次の cycle で上書きされる)、ref の説明の重複と通知文字列の重複。cycle 1 の test report の gap (b)(symlink のときの「未コミット無視」通知の誤発火)は HEAD で解消していることを reviewer が確認(cycle 2 の `/test` で再確認する)
- 2026-09-24 work(cycle 2): Slice F は implementer に委譲(fc7d2e4、3 ファイル)。self-review cycle 2 の 6 件を修正: コメントの訂正、通常ファイルの `git show` も失敗時は空の allowlist と通知、`./` の除去を検査の前に、複数行 guard の一言、テスト名を `test_ac2d_` / `test_ac3b_` に、ヘッダーの説明を本文への参照に縮めて通知は `report_allowlist_unreadable` に集約。テストは 87 件から 96 件へ(ディレクトリを指す bare な symlink、絶対パス・`..`・`.//x` のリンク先)。mode の case に 040000 を足す mutation で bare ディレクトリのテストが落ちることを implementer が確認。通常ファイルの `git show` の失敗は blob を消した repo で再現し、修正前は rc 128 で scan せずに落ち、修正後は通知を出して scan を続けて検出する。`.//x` は修正前も git 側の拒否で同じ結果になるため、red/green の差は出ない(スクリプト自身の検査に変えただけ)。orchestrator は HEAD 一致・porcelain 空・差分・template の byte 一致、probe 4 通り、shell テスト 3 本、`run-verify.sh`、strict scan を確認
- 2026-09-24 `/verify`(cycle 2、同じ report の `## Cycle 2`、7d46144): pass。AC-2d / AC-3b と Scope 1 の現在の契約を、dbba825 と fc7d2e4 の delta に対して再確認。self-review cycle 2 の 6 件がすべて修正されていることをコードに突き合わせて確認。AC-1〜8 のうち delta が触れないものは byte 不変で再確認。文書 drift は shipped docs になし。report artifact(walkthrough の Known limitations)の 1 行が stale であることを指摘し、`/sync-docs` に送った
- 2026-09-24 `/test`(cycle 2、同じ report の `## Cycle 2`、0e7f164): pass。`run-test.sh`(840 shell PASS + 8 Go package)、`test-secret-scan-branch.sh` 96/96、`test-run-verify-branch-secret-scan.sh` 32/32。環境行列 7/7、repeat 10/10、concurrency 2 形状、red/green mutation 6 件(4 件が判別、2 件は self-review の予測どおり無差)、実演 5 シナリオ、cycle-1 test report の gap (b)(symlink の誤通知)の解消を再確認。残る gap: submodule(`160000`)と symlink-to-symlink(`120000`→`120000`)は手作業で fail-closed を確認したが suite にテストなし(cycle-1 からの `check-template.sh` required_files の gap は継続、この cycle の delta 外)
- 2026-09-25 `/sync-docs`(cycle 2): shipped docs(`/pr` skill 4 面、`quality-gates.md` 2 コピー、AGENTS.md、README.md、definition-of-done.md、docs/recipes/、`.ralph/core/AGENTS.core.md`、`secret-scan.sh` の案内文、`secret-scan-branch.sh` 自身のヘッダー)は verify cycle 2 の結論どおり drift なしを再確認。`docs/tech-debt/README.md` に submodule / symlink-to-symlink の未テストを 1 行で追加。walkthrough を cycle 2 用に更新(差分規模、`.gitallowed` の mode 振り分けの説明、コミット単位に dbba825/fc7d2e4、symlink notice 誤発火の記述を削除して未テスト 2 モードの記述に置き換え)
- 2026-09-25 cross-review cycle 2(`docs/reports/cross-review-triage-local-branch-secret-scan.md`、reviewed HEAD f720cc2): Codex の指摘 2 件(P1 / P2)を ACTION_REQUIRED と判定(AR-3 / AR-4)。どちらも「strict で clean なのに CI では検出」の向きで、triager も fixture repo で再現した。AR-3 `color.ui=always` だと scanner の `git log -p` の追加行が色のエスケープで始まり、`+` で始まる行だけを読む parser が読み飛ばす(scanner 既存の弱点で、CI の runner は既定設定なので CI 自体には影響しない)。AR-4 symlink のリンク先の末尾改行がコマンド置換で落ち、改行なしの名前のファイルに解決される(改行で終わるファイル名とリンク先が両方コミットされている場合に限る)。cap(2 サイクル)に到達しているため修正して再実行の選択肢はなく、ユーザーに「cap を上げて再実行」「PR を作成し known gaps として記録」「中止」を提示した。ユーザー決定(AskUserQuestion): PR を作成し、AR-3 / AR-4 は known gaps として記録して後続 issue で直す
- 2026-09-25 `/pr`: 後続 issue #176(AR-3 / AR-4)を起票し、PR #177 を作成(Closes #169、Known gaps に 2 件を記載)。push 前の `run-verify.sh` pass、`secret-scan-branch.sh --strict` は e7cdcb8 で clean。`ensure-pr-title-prefix.sh` / `ensure-pr-ready.sh` pass。逸脱: この plan ファイルは b78289f〜e7cdcb8 で本文が二重化していた(orchestrator の編集スクリプトの誤りで、`str.replace` の置換文字列に全文を渡していた)。先頭の 1 部と cross-review cycle 2 の bullet、checklist 1 組に復元した(内容の欠落はない)

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created (#177)

## Readiness checklist

- [x] 現状の配線を確認した(CI の step、hook 群、`run-verify.sh`、template のコピー、既存の range のテスト)
- [x] critical fork なし
- [x] AC は一時 repo を使う shell テストで決定的に確認できる
- [x] Codex plan advisory(2 件、対応案で plan を更新)
