# Walkthrough: secret-scan-attr-parity (#181)

- Date: 2026-09-28
- Plan: docs/plans/active/2026-09-27-secret-scan-attr-parity.md
- Branch: fix/secret-scan-attr-parity(base main 03f3e8a)
- Diff: 21 files、+2806 / -82(うち docs/reports と plan が約 1,130 行、テストが約 1,000 行)

## 何を直したか

ローカルの secret scan が CI と違う属性で `git log -p` を読む 3 つの経路を閉じた。

1. ローカルの `diff.<driver>.binary` / `diff.<driver>.algorithm`。コミット済みの `.gitattributes` が指す driver に対して、ローカル config の値を `auto` に固定し、`--diff-algorithm=default` で algorithm を既定に戻す。
2. `.git/info/attributes` の規則。git はどの設定でもこのファイルを読み、CI の clone には存在しない。`secret-scan-branch.sh` は規則の行があると「cannot scan」で止まる(`--strict` は exit 3、既定 mode は exit 0)。scanner 自体は拒否しないので、merge 中の hook は無関係な規則で止まらない。
3. base が分岐後に `.gitattributes` を変えた branch。CI は PR の merge commit を checkout するので、ローカルは `git merge-tree --write-tree <base> HEAD` の tree を `RALPH_SECRET_SCAN_ATTR_SOURCE` として scanner に渡し、merge 結果の属性で scan する。

## 読む順

### scripts/secret-scan.sh(template と byte 一致)

- `scan_range`: `diff.*.binary` の列挙と `auto` への固定(`--config-env` は key に `=` を含む driver 名のため)、`GIT_ATTR_NOSYSTEM=1` と `core.attributesFile=/dev/null`、`RALPH_SECRET_SCAN_ATTR_SOURCE` の受け取り(tree-ish に解決できなければ exit 3、範囲が HEAD で終わらないときは無視)。
- 既定の設定での結果は変えていない(AC-8)。

### scripts/secret-scan-branch.sh(template と byte 一致)

- `check_info_attributes`: 規則の行の判定(空行とコメントは除外、読めないファイルも停止)。
- `use_merge_attributes`: base が `.gitattributes` を変えたかを `git diff --quiet ... ':(top,glob)**/.gitattributes'` で判定してから merge-tree を呼ぶ。merge-tree はローカル設定から隔離して実行する(`GIT_ATTR_SOURCE` を HEAD の commit に固定、`GIT_ATTR_NOSYSTEM=1`、`core.attributesFile=/dev/null`、`attr.tree=`、`merge.renormalize=false`、`merge.renames=true`、`merge.directoryRenames=conflict`、`merge.renameLimit=7000`)。どの pin が効いていて、どれが既定の再掲かはコメントに書いた。
- `attributes_unguaranteed <reason> [<remedy>]`: merge 結果の属性を CI と同じに計算できないとき、`--strict` は exit 3、既定 mode は通知して HEAD の属性で scan する。「base を merge か rebase する」という remedy は、それで解決する原因(ローカルの merge driver、2.41 未満の git、merge-tree の失敗)にだけ付く。
- 衝突(rc 1)は HEAD の属性で scan し、通知を 1 行出す。
- allowlist の一時ファイルが作れないときも「cannot scan」で止まる(以前は無言の exit 1)。

### tests/

- `tests/test-secret-scan.sh`(123 件): binary / algorithm の固定、NUL バイトの fixture、`GIT_ATTR_NOSYSTEM` の wrapper、属性ソースの受け取り。
- `tests/test-secret-scan-branch.sh`(246 件): info/attributes の判定、merge 結果の属性(base が `-diff` を足す / 外す)、衝突、merge driver、merge-tree の失敗 4 分岐、pin ごとの隔離ケース(renormalize、directoryRenames、merge-tree が受け取る環境)、mktemp の 2 箇所。本物の git が 2.41 未満のときは merge 経路のケースを SKIP し、文書どおりの exit 3 と代替の挙動を確認する。
- token は実行時に分割した文字列から組み立てている(CI は branch の履歴を scan する)。

### docs/

- `docs/quality/quality-gates.md`(2 コピー)と `docs/tech-debt/README.md`: 一致の範囲(git 2.41 以降)、残る差(replace ref、2.41 未満、merge guard が読む info/attributes、既定 mode の不 scan、GitHub のサーバー側 merge が読む属性は未文書)。
- `.claude/skills/pr/SKILL.md`(4 コピー): exit 3 の原因と対処に merge-tree の経路を追加。

## 検証

| 項目 | 結果 |
|---|---|
| `sh tests/test-secret-scan-branch.sh`(git 2.49、sh と dash) | 246 PASS / 0 FAIL |
| `sh tests/test-secret-scan.sh` | 123 PASS / 0 FAIL |
| `sh tests/test-run-verify-branch-secret-scan.sh` | 32 PASS / 0 FAIL |
| docker alpine:3.18(git 2.40.4) | 194 / 0(SKIP 10)、94 / 0(SKIP 2) |
| docker alpine:3.19(git 2.43.7) | 241 / 0(SKIP 3)、122 / 0(SKIP 1) |
| pin の mutation 6 種(scratch コピー) | すべて該当テストが落ちる |
| `./scripts/run-verify.sh` | All verifiers passed |
| `./scripts/secret-scan-branch.sh --strict` | 03f3e8a..8cbb918 clean |

## pipeline の履歴

- cycle 1: self-review(MEDIUM 3 / LOW 3、Slice D で修正)→ verify pass → test pass → sync-docs → cross-review で ACTION_REQUIRED 2 件(merge-tree の隔離、テストの git version gate)。ユーザーの決定で修正して再実行。
- cycle 2: Slice E(fe4f383)→ self-review(MEDIUM 1 / LOW 5、Slice F 8443a03 で修正、C2-L1 は tech-debt に記録)→ verify pass → test pass → sync-docs drift なし → cross-review 指摘 0 件(cap 2/2 到達、Case C)。

## 残る gap

- `use_merge_attributes` の「HEAD does not resolve to a commit」分岐は専用テストがない(merge-base と rev-list が通った後は到達しないと判定)。
- `.git/info/attributes` に規則があると既定 mode は scan しない(main は scan していた)。AC-3 の決定どおりで、tech-debt に記録。
- GitHub のサーバー側 merge がどの属性を読むかは文書がない。committed の `.gitattributes` が `.gitattributes` 自身に `merge=` を与える場合だけ差が出る。
