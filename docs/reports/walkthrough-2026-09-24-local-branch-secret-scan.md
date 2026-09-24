# Walkthrough: local-branch-secret-scan

- Date: 2026-09-24
- Plan: docs/plans/archive/2026-09-20-local-branch-secret-scan.md(PR 作成時に active から移動)
- Issue: #169(Closes)。PR #168 で CI の secret scan が 2 回失敗したことの恒久対応
- Branch: feat/local-branch-secret-scan(base: main @ a0d5bf5)
- 差分規模: 約 27 files / +2,000 −70。実装は shell 4 ファイル(`scripts/secret-scan-branch.sh` 新規 183 行、`run-verify.sh` +32、`secret-scan.sh` +3、`check-template.sh` +2、`xreview-helpers.sh` のヘッダー)と `templates/base/scripts/` の同一コピー、テスト 2 ファイル(約 1,040 行)、`/pr` skill 4 面、`quality-gates.md` 2 コピー、AGENTS.md の 1 行、`internal/scaffold/embed_test.go` の 1 行。残りは plan と pipeline レポート

## 何が変わったか

CI の `verify` ジョブは branch の履歴全体(`git log -p` の追加行)を secret scan するが、ローカルには同じ scan がなかった。PR #168 では、テストの偽の値と、それを許可するために足した `.gitallowed` の行自体が、push 後に初めて掛かった。履歴を読む scan なので、後のコミットで消しても検出は残る。

1. **`scripts/secret-scan-branch.sh [--strict]`**: CI と同じ range(base branch との merge-base から HEAD)を、同じ scanner で scan する。base は `GITHUB_BASE_REF`、なければ `detect_base_branch`(`RALPH_XREVIEW_BASE` を尊重)。`origin/<base>` があればそれを、なければローカルの `<base>` を使う。allowlist は HEAD にコミット済みの `.gitallowed` だけ(作業ツリーの未コミットの変更と `RALPH_SECRET_ALLOWLIST` は無視して、その旨を 1 行出す。CI が読むのはコミット済みの内容だけなので)。
2. **exit code**: 0 = scan して問題なし。1 = 検出。2 = 使い方の誤り。3 = `--strict` で「scan できない」(base の ref がない、merge-base が取れない、scanner がない、git の外)または「scan するものがない」(HEAD が base branch、range が空)。既定の mode では後者 2 種類は理由を出して exit 0(早期検出用で、失敗にはしない)。scanner が 1 以外の非 0 で終わったときは `scanner failed with exit <rc>` と出してその code で終わる。
3. **`run-verify.sh`**: `static` / `all` のとき branch scan を実行する(`test` では実行しない)。言語の verifier の有無(`ran_any`)には数えない。失敗したら末尾に `==> Branch secret scan failed` を出して非 0 で終わる。`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1`(`1` のときだけ)で skip。
4. **`/pr` skill**: コミットの後・push の直前に `--strict` で scan し、exit 0 以外では push しない(Step 3)。plan の archive のコミットを push する前にも再 scan する(Step 8)。
5. **案内文**: scanner が検出したとき、`.gitallowed` の行は自分自身がパターンに一致しない形(キー名を `api_ke[y]` のような括弧式)で書くよう案内する。

## 読む順番

1. `scripts/secret-scan-branch.sh` — 上から: usage と `--strict` → trap(signal では exit する)→ `report_and_exit` / `cannot_scan` / `nothing_to_scan` → repo root → base の解決 → 「HEAD が base branch」→ base ref(origin 優先)→ merge-base → range が空 → scanner の有無 → HEAD の `.gitallowed` を一時ファイルへ → 無視の通知 → scan と結果行
2. `scripts/run-verify.sh` — `branch_scan_failed` の導入、`static|all` の case、末尾の失敗行(`ran_any` の分岐の外)
3. `.claude/skills/pr/SKILL.md` — Step 3、Step 8、Completion gate
4. `tests/test-secret-scan-branch.sh`(64 件)— 一時 repo の組み立て、fixture の実行時生成(`printf` で分割した断片から)、exit 0 の理由行の固定(`against <ref>: clean`)、strict のすり抜け 2 通り、stub scanner
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
| その他 | plan の進捗・逸脱記録、各レポート、insight events、sync-docs |

## 設計判断(plan Design decisions より)

- 1 つのスクリプトに寄せる。`/pr` と `run-verify.sh` と人が同じコマンドで同じ結果を得る。
- `run-verify.sh` からの呼び出しは skip を失敗にしない(早期検出。最終の関門は `/pr` の strict と CI)。`/pr` は strict で呼び、exit 0 は「scan して問題なし」だけを意味する。strict の「scan するものがない」を exit 0 にする当初の案は、base の指定を現在の branch にする、`origin/<base>` がない状態でローカルの base を先端まで進める、のどちらでも未 push のコミットを抱えたまま通ることが self-review で実証され、exit 3 に変えた。
- allowlist は CI と同じくコミット済みのものだけ。
- `run-verify.sh` では既定で有効(無効が既定だと PR #168 と同じ見落としが起きる)。scaffold 先でも CI と同じ基準。

## 注意して見てほしい点

- strict で exit 0 になる経路が `scanned … : clean` の 1 つだけであること。
- `report_and_exit` を通らずに終わる経路がないこと(temp ファイルは EXIT の trap で消える)。
- `run-verify.sh` の締めの行が、`ran_any` と scan の失敗のどの組み合わせでも事実を言っていること。
- テストのソースに secret 風のリテラルがないこと(この branch 自身への range scan は clean)。

## Known limitations

- 中断は協調的。`scanner failed with exit <rc>` は scanner 側の異常終了(130 など)で出る。
- `.gitallowed` が HEAD で symlink のとき、「未コミットの変更を無視した」の通知が誤って出ることがある(scan 自体は正しい。cosmetic)。
- `scripts/check-template.sh` の必須一覧には、項目を落としても落ちるテストがない(既存の欠落。test report に記録)。
- SIGINT の trap は実行環境の制約で実機確認できていない(TERM で確認。INT / HUP は同じ形)。
- CI では、専用の step と `run-verify.sh` の中で同じ scan が 2 回走る(最初の失敗で止まるので重複だけ)。
