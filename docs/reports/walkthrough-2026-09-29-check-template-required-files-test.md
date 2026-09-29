# Walkthrough: check-template-required-files-test (#183)

- Date: 2026-09-29
- Plan: docs/plans/active/2026-09-29-check-template-required-files-test.md
- Branch: test/check-template-required-files-test(base main 4794808)
- Diff: 12 files、+838 / -29(コードは `tests/test-check-template.sh` +197、`internal/scaffold/embed_test.go` +151 / -29 相当、`check-template.sh` 2 コピー +12 / -1 ずつ、tech-debt 1 行。残りは plan と報告)

## 何を直したか

`scripts/check-template.sh` の `required_files` から項目を落としても、どのテストも落ちなかった(#169 の mutation で判明)。一覧を守る回帰テストを shell と Go の両側に足し、Go 側の一覧(`templates/base/scripts/` に必ず入るスクリプト)と shell 側の一覧(project に必須のファイル)の `scripts/` 部分を同じ集合にした。

## 読む順

1. `scripts/check-template.sh`(template と byte 一致): `required_files` が 18 → 28 項目。追加したのは Go 側だけにあった 10 本(`run-static-verify.sh`、`run-test.sh`、`detect-changed-languages.sh`、`detect-languages.sh`、`new-feature-plan.sh`、`codex-check.sh`、`ralph-config.sh`、`ralph-worktree.sh`、`check-template.sh`、`check-skill-sync.sh`)。`scripts/` の項目は Go 側と同じ順に並べた(読みやすさのためで、テストは集合で比べる)。
2. `tests/test-check-template.sh`(mode 100755、32 assertion): A = script の一覧を golden と順序込みで比較(空なら書式変更として FAIL、差は diff で表示)、B = 必須ファイルをすべて持つ最小 fixture で pass を確認してから、golden の各項目を 1 つずつ外して `Missing required file` の検出を確認(golden が駆動するので script 側の欠落も検出する。`mktemp -d` の失敗は FAIL、EXIT trap で後始末)、C = repo root に全項目が存在。
3. `internal/scaffold/embed_test.go`: `required` を package 変数 `requiredTemplateScripts` にし、`TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles` が template 側 `check-template.sh` の `required_files` から `scripts/` 項目を読んで集合の一致を確認する。不一致は「only in check-template.sh / only in the Go list」で列挙し、0 件なら書式変更として Fatal。コメントに、一覧を変えるときに更新する 4 箇所(check-template.sh 2 コピー、`requiredTemplateScripts`、`GOLDEN_ENTRIES`)を書いた。
4. `docs/tech-debt/README.md`: 「回帰テストがない」行を削除し、#189 の行(hook 参照検査の誤検出と fail-open、meta-repo 専用の必須 3 項目で scaffold の PR CI が red)を追加。

## 検証

| 項目 | 結果 |
|---|---|
| `bash tests/test-check-template.sh` | 32 passed / 0 failed(3 回連続) |
| `go test ./internal/scaffold/...`(`TMPDIR=/tmp` も) | ok |
| `./scripts/run-test.sh`(既定と `RALPH_VERIFY_SCOPE=full`) | green |
| `./scripts/run-static-verify.sh`(`RALPH_VERIFY_BASE=main`)、check-sync、shellcheck | green |
| mutation: shell 側 28 項目を 1 つずつ削除 | 28 / 28 で A と B が落ちる |
| mutation: Go 側・template 側から 1 項目削除 | 等価テストが名前を挙げて落ちる |
| mutation: `required_files="` を `='` に | shell は空ブロックで FAIL、Go は Fatal |
| golden を壊したコピーで `HARNESS_VERIFY_MODE=test ./scripts/verify.local.sh` | exit 1(`chmod -x` にすると黙って exit 0 = 100755 の根拠) |
| fresh scaffold(`ralph init --yes`) | 22 本すべて存在。`check-template.sh` の `Missing required file` は #189 の 3 項目だけ |

## pipeline の履歴

- plan: fresh scaffold の probe で `check-template.sh` が main の時点で通らないことを発見(#189 に起票、AC-3 を修正)。Codex advisory MEDIUM 1(実行権限がないと runner が黙って飛ばす)→ AC-7 を追加。
- self-review: MEDIUM 2(tech-debt 行のパイプ未エスケープ、scaffold 側の呼び出し元を bootstrap.sh と誤記 → 実際は `verify.yml` の PR CI)/ LOW 4 → Slice B で修正。#189 の本文も訂正。
- verify pass(push 済み branch では `run-static-verify.sh` の既定 scope が Go を飛ばすと判明 → #190 に起票)→ test pass → sync-docs drift なし → cross-review 指摘 0 件。

## 残る gap

- どちらの一覧にもない 6 本(`check-coverage.sh`、`check-pipeline-sync.sh`、`gc-artifacts.sh`、`insights-append.sh`、`ralph-common.sh`、`sync-skills.sh`)を必須にするかは未判断(plan の Open questions)。
- #189(hook 参照検査、meta-repo 専用の必須項目)と #190(changed scope の base)は後続。
