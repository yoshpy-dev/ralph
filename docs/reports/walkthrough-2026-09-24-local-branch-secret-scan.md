# Walkthrough: local-branch-secret-scan

- Date: 2026-09-24
- Plan: docs/plans/archive/2026-09-20-local-branch-secret-scan.md(PR 作成時に active から移動)
- Issue: #169(Closes)。PR #168 で CI の secret scan が 2 回失敗したことの恒久対応
- Branch: feat/local-branch-secret-scan(base: main @ a0d5bf5)
- 差分規模: 約 32 files / +3,194 −67(cycle 2 まで。`git diff main...HEAD --stat`)。実装は shell 4 ファイル(`scripts/secret-scan-branch.sh` 新規 298 行、`run-verify.sh` +32、`secret-scan.sh` +3、`check-template.sh` +2)と `xreview-helpers.sh` のヘッダー、`templates/base/scripts/` の同一コピー、テスト 2 ファイル(`tests/test-secret-scan-branch.sh` 1,206 行・96 件、`tests/test-run-verify-branch-secret-scan.sh` 299 行・32 件)、`/pr` skill 4 面、`quality-gates.md` 2 コピー、AGENTS.md の 1 行、`internal/scaffold/embed_test.go` の 1 行。残りは plan と pipeline レポート

## 何が変わったか

CI の `verify` ジョブは branch の履歴全体(`git log -p` の追加行)を secret scan するが、ローカルには同じ scan がなかった。PR #168 では、テストの偽の値と、それを許可するために足した `.gitallowed` の行自体が、push 後に初めて掛かった。履歴を読む scan なので、後のコミットで消しても検出は残る。

1. **`scripts/secret-scan-branch.sh [--strict]`**: CI と同じ range(base branch との merge-base から HEAD)を、同じ scanner で scan する。base は `GITHUB_BASE_REF`、なければ `detect_base_branch`(`RALPH_XREVIEW_BASE` を尊重)。`origin/<base>` があればそれを、なければローカルの `<base>` を使う。merge-base には、存在を確認した完全な ref 名(`refs/remotes/origin/<base>` / `refs/heads/<base>`)をそのまま渡す(短い名前は表示にだけ使う。base と同名の tag や、`origin/<base>` という名前のローカル branch があっても git の短縮名解決の影響を受けない)。allowlist は HEAD にコミット済みの `.gitallowed` だけ(作業ツリーの未コミットの変更と `RALPH_SECRET_ALLOWLIST` は無視して、その旨を 1 行出す。CI が読むのはコミット済みの内容だけなので)。`.gitallowed` は tree の mode で読み分ける: 通常ファイルはそのまま読む。symlink はリンク先を、絶対パスと `..` を拒否したうえでコミット済みの tree の中で 1 段だけ解決する。それ以外(エントリなし、ディレクトリ、submodule、解決できない symlink)は空の allowlist で scan し、その旨を通知する。
2. **exit code**: 0 = scan して問題なし。1 = 検出。2 = 使い方の誤り。3 = `--strict` で「scan できない」(base の ref がない、merge-base が取れない、scanner がない、git の外)または「scan するものがない」(HEAD が base branch、range が空)。既定の mode では後者 2 種類は理由を出して exit 0(早期検出用で、失敗にはしない)。scanner が 1 以外の非 0 で終わったときは `scanner failed with exit <rc>` と出してその code で終わる。
3. **`run-verify.sh`**: `static` / `all` のとき branch scan を実行する(`test` では実行しない)。言語の verifier の有無(`ran_any`)には数えない。失敗したら末尾に `==> Branch secret scan failed` を出して非 0 で終わる。`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1`(`1` のときだけ)で skip。
4. **`/pr` skill**: コミットの後・push の直前に `--strict` で scan し、exit 0 以外では push しない(Step 3)。plan の archive のコミットを push する前にも再 scan する(Step 8)。
5. **案内文**: scanner が検出したとき、`.gitallowed` の行は自分自身がパターンに一致しない形(キー名を `api_ke[y]` のような括弧式)で書くよう案内する。

## 読む順番

1. `scripts/secret-scan-branch.sh` — 上から: usage と `--strict` → trap(signal では exit する)→ `report_and_exit` / `cannot_scan` / `nothing_to_scan` → repo root → base の解決 → 「HEAD が base branch」→ base ref(origin 優先、merge-base には完全な ref 名を渡す)→ merge-base → range が空 → scanner の有無 → HEAD の `.gitallowed` を一時ファイルへ(`allowlist_ls_tree_mode` / `allowlist_target_is_safe` / `report_allowlist_unreadable` によるモードの振り分け)→ 無視の通知 → scan と結果行
2. `scripts/run-verify.sh` — `branch_scan_failed` の導入、`static|all` の case、末尾の失敗行(`ran_any` の分岐の外)
3. `.claude/skills/pr/SKILL.md` — Step 3、Step 8、Completion gate
4. `tests/test-secret-scan-branch.sh`(96 件)— 一時 repo の組み立て、fixture の実行時生成(`printf` で分割した断片から)、exit 0 の理由行の固定(`against <ref>: clean`)、strict のすり抜け 2 通り、stub scanner、`test_ac2d_*`(symlink allowlist の解決・拒否・dangling)と `test_ac3b_*`(tag と `origin/<base>` という名前の branch による ref shadow)の 2 グループ
5. `tests/test-run-verify-branch-secret-scan.sh`(32 件)— 3 つの mode、skip の値、docs だけの変更と検出の組み合わせ、`RALPH_VERIFY_SCOPE` の固定
6. `scripts/secret-scan.sh` の案内文、`scripts/check-template.sh` と `internal/scaffold/embed_test.go` の必須一覧、`docs/quality/quality-gates.md`

## コミット単位

| SHA | 内容 |
|---|---|
| d4fd72f, 3d572b0 | plan。Codex plan advisory の 2 件(skip が exit 0 だと未 scan の履歴が push できる、scan が push する HEAD に紐づいていない)を反映して `--strict` と「push の直前」に |
| 2ca73aa | `secret-scan-branch.sh`、テスト、template のコピー、必須一覧 |
| 335dddf | `run-verify.sh` への組み込み、案内文、テスト |
| d4b9336, eda0248 | `/pr` skill(4 面)、`quality-gates.md`、AGENTS.md |
| 3afac6e | self-review の 12 件(strict の exit 0 を「scan して問題なし」だけに、`run-verify.sh` の失敗行、skip は `1` だけ、scanner の exit の区別、signal の trap、必須一覧に `xreview-helpers.sh`、テストの理由行の固定 ほか) |
| 9554a58 | test の追加 5 件(gpgsign を含む環境からの隔離、base 名の `/`、`test` mode で検出があっても走らない) |
| dbba825 | cross-review cycle 1 の AR-1(HEAD の `.gitallowed` がコミット済みの symlink だと、リンク先のパス文字列自体が allowlist の正規表現として効く)/ AR-2(base ref の存在確認は完全な ref 名で行うのに merge-base には短い名前を渡すため、base と同名の tag があると range がずれる)を修正。orchestrator の審査で見つかった cwd 相対の `ls-tree` とディレクトリを指す symlink の 2 件も同じコミットに amend で取り込んだ |
| fc7d2e4 | self-review cycle 2 の MEDIUM 1 / LOW 5(コメントの訂正、通常ファイルの `git show` のガード、`./` の除去順の入れ替え、複数行 guard の一言、通知の一本化、テスト名を `test_ac2d_*` / `test_ac3b_*` に)を修正し、テスト 4 件を追加 |
| その他 | plan の進捗・逸脱記録、各レポート(cycle 2 の self-review / verify / test を含む)、insight events、sync-docs |

## 設計判断(plan Design decisions より)

- 1 つのスクリプトに寄せる。`/pr` と `run-verify.sh` と人が同じコマンドで同じ結果を得る。
- `run-verify.sh` からの呼び出しは skip を失敗にしない(早期検出。最終の関門は `/pr` の strict と CI)。`/pr` は strict で呼び、exit 0 は「scan して問題なし」だけを意味する。strict の「scan するものがない」を exit 0 にする当初の案は、base の指定を現在の branch にする、`origin/<base>` がない状態でローカルの base を先端まで進める、のどちらでも未 push のコミットを抱えたまま通ることが self-review で実証され、exit 3 に変えた。
- allowlist は CI と同じくコミット済みのものだけ。
- `run-verify.sh` では既定で有効(無効が既定だと PR #168 と同じ見落としが起きる)。scaffold 先でも CI と同じ基準。
- symlink は 1 段だけ解決し、解決できなければ空の allowlist で scan する(fail-closed: 例外が減っても検出が隠れることはない)。

## 注意して見てほしい点

- strict で exit 0 になる経路が `scanned … : clean` の 1 つだけであること。
- `report_and_exit` を通らずに終わる経路がないこと(temp ファイルは EXIT の trap で消える)。
- `run-verify.sh` の締めの行が、`ran_any` と scan の失敗のどの組み合わせでも事実を言っていること。
- テストのソースに secret 風のリテラルがないこと(この branch 自身への range scan は clean)。
- `allowlist_ls_tree_mode` の 1 行・パス一致の検査と mode の case が、別々の入力を弾いていること(末尾 `/` を付けたディレクトリ symlink の拒否は前者、bare なディレクトリと submodule の拒否は mode の case)。

## Known limitations

- 最終の cross-review(cycle 2、cap 到達)の 2 件は未修正で #176 に送った: git の設定に `color.ui=always` があると scanner の `git log -p` の追加行が色のエスケープで始まり読み飛ばされる(scanner 既存の弱点)。symlink の `.gitallowed` のリンク先が改行で終わると、コマンド置換で改行が落ちて改行なしの名前のファイルに解決される。どちらも「strict は clean、CI は検出」の向き(`docs/reports/cross-review-triage-local-branch-secret-scan.md` の AR-3 / AR-4)。
- 中断は協調的。`scanner failed with exit <rc>` は scanner 側の異常終了(130 など)で出る。
- `scripts/check-template.sh` の必須一覧には、項目を落としても落ちるテストがない(既存の欠落。test report に記録)。
- SIGINT の trap は実行環境の制約で実機確認できていない(TERM で確認。INT / HUP は同じ形)。
- CI では、専用の step と `run-verify.sh` の中で同じ scan が 2 回走る(最初の失敗で止まるので重複だけ)。
- `.gitallowed` が submodule(mode `160000`)や symlink-to-symlink(`120000` → `120000`)のときは空の allowlist で fail-closed に scan することを手作業で確認したが、`tests/test-secret-scan-branch.sh` に固定するテストはない(cycle 2 の test report「Test gaps」)。
- `.//x` のような `./` 付きの symlink target を検査の前に正規化する順序の入れ替え、および通常ファイルの `git show` に付けたガードは、self-review cycle 2 が予測したとおり suite の red/green に観測可能な差を出さない(git 自身の絶対パス pathspec 拒否と、mode 検査による blob 存在の事前証明がそれぞれ実質的な防波堤になっている)。破損 object での分岐は手作業でのみ確認した。
