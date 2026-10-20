# sync-docs report: secret-scan-test-tmp

- Date: 2026-10-20
- Plan: `docs/plans/active/2026-10-20-secret-scan-test-tmp.md`(承認の digest `3ee6f70482f6` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `e333200c`、branch は `fix/secret-scan-test-tmp`。run 開始時の HEAD は `823066af`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-20-secret-scan-test-tmp.md`(Merge yes。M1・L1・L2・I1 は直し済み)、`docs/reports/verify-2026-10-20-secret-scan-test-tmp.md`(pass。V1 は 74598215 で直した、V2〜V4 は INFO)、`docs/reports/test-2026-10-20-secret-scan-test-tmp.md`(pass。G1 は LOW、G2〜G5 は INFO)
- この run が変えたのは、tech-debt の 1 行の追加、計画の Progress checklist、この report、insight event。コード、skill、rule、先行 3 report、計画の本文には触っていない

## Summary

`e333200c...823066af` の差分でコードを変えたのは `tests/test-secret-scan.sh` だけで、`expect_exit` の capture を固定の `/tmp/ralph-secret-scan-test.out` と `.err` から、実行ごとの `mktemp -d` の `capdir` に移した。公開の面、CLI、skill、hook、rule は変わっていない。

旧い固定パス、2 つ同時の実行で起きる overlap の flake、重ねて走らせないよう勧める文を、live の文書から探した。いまの動きとして旧い動きを書く文書はなかった。旧い動きを書く 2 つの register の項目(tech-debt の 170・171 行の固定パスの行と、178 行の org-stop-all の test gaps の項目 (e))は、`ece77f30` と `90fc10d1` で閉じてあり、取り消し線と RESOLVED の文で今の動きを書いている。

直した点は、test report の G1(LOW)が register に載っていなかったこと。固定のパスに戻す変更を止める check が suite になく、AC4 を確かめたのは suite の外の harness だけである。tech-debt に 1 行足した。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(311 行、末尾に追加) | test report の G1 を 1 行足した。区切りの `|` は 6 個で、隣の行と同じ。計画の道は `docs/plans/active/` の形(`/pr` の `archive-plan.sh` が書き換える)。G2〜G5 は INFO で 1 文に収まらないので Related の欄に「INFO」と書くだけにして、中身は test report に残した |
| `docs/plans/active/2026-10-20-secret-scan-test-tmp.md`(Progress checklist) | 「Verification artifact created」と「Test artifact created」にチェックを入れ、verify と test の結果の注、この run の注を足した。本文(Progress checklist より上)は変えていない。digest は前後で `3ee6f70482f6` |
| `docs/reports/sync-docs-2026-10-20-secret-scan-test-tmp.md` | 新規 |
| `docs/insights/events/2026-10-20-secret-scan-test-tmp.jsonl` | sync_docs の event を 1 行足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| 固定パス `ralph-secret-scan-test` を書く文書(`.git` と `docs/evidence/` を除く全 file の grep) | 合っている | 当たるのは `tests/test-secret-scan.sh` の新しい prefix(44 行)、tech-debt の 3 行(170・171・178 行。取り消しと RESOLVED の文)、この計画、`docs/reports/` と `docs/plans/archive/` の過去の記録。履歴の report と archive は直さない |
| `test-secret-scan.sh` を名指す文書 | 合っている | live の文書で当たるのは tech-debt の上の行と `tests/test-secret-scan-branch.sh` の 248 行の comment だけ。後者は git の版の読み方の話で、capture のパスには触れない |
| overlap、clobber、2 つ同時、固定の `/tmp` を書く文書(`docs/`、`.claude/rules/`、`.claude/skills/`、`AGENTS.md`、`README.md`、`CLAUDE.md`、`templates/`) | 合っている | `docs/quality/quality-gates.md` 31 行の「non-overlapping」は `run-static-verify.sh` と `run-test.sh` の責任の分け方で、実行の重なりではない。tech-debt の 225 行(`run-verify.sh` の evidence log)、228 行(gate の再実行)は別の欠陥の行。重ねて走らせないよう勧める文はない |
| `docs/tech-debt/README.md` 170・171 行 | 合っている | 解消の文は evidence と合う(単独 123/0、2 つ同時に両方 `FAIL: 0`、TERM のあと dir なし)。verify の V1 が指した「どう旧版を動かしたか」の文は 74598215 で直してある |
| `docs/tech-debt/README.md` 178 行の項目 (e) | 合っている | 3 欄で閉じ、(a)〜(d) は開いたまま。計画の Scope は「row 170」だけを挙げるが、(e) を閉じた理由は計画の Progress の注に残っている(self-review の M1) |
| `docs/architecture/repo-map.md`(67 行) | 変更なし | `tests/` の説明は「shell test suites (`tests/test-*.sh`)」で、個々の test の中身を持たない |
| `docs/quality/quality-gates.md`、`docs/quality/definition-of-done.md` | 変更なし | test の実行手順は `run-test.sh` で、capture の置き場を書かない。`mktemp` の約束を書く rule もない(`.claude/rules/ralph/` で当たるのは `post-implementation-pipeline.md` の例の 1 行だけ) |
| `AGENTS.md`、`CLAUDE.md`、`README.md`、`.claude/rules/ralph/` | 変更なし | 公開の面が変わらない |
| `templates/base/` | 変更なし | `templates/base/tests/` はなく、scaffold は `tests/test-secret-scan.sh` を持たない。`check-sync.sh` は `PASS: all files in sync.` |
| 計画 | 合っている | Progress checklist に S1、self-review、verify、test の記録がある。AC1〜AC5 はすべて印が付いている |

## Plan drift notes

- 計画の Scope は tech-debt の「row 170」だけを挙げる。同じ欠陥の 178 行の項目 (e) も閉じたので範囲は少し広がったが、理由は Progress の注(self-review の M1)に書いてあり、計画の本文は digest を保つために変えていない

## Left open

- G1 は register に載せた。止めるには `scripts/verify.local.sh` に、`tests/*.sh` の固定の `/tmp/<name>` への redirect を断る grep を足す。`tests/test-pre-bash-guard.sh` は guard に食わせるコマンドの文字列として `/tmp/` を持つ行が 12 あるので、grep は狭く書く
- G2(相対の `TMPDIR` は HEAD も base も支えていない)、G3(HUP と INT の trap は走らせていない)、G4(Linux の GNU `mktemp` と dash の `/bin/sh` では走らせていない)、G5(3 本以上の同時の実行は打っていない)は INFO で、test report に残した

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(register の行と計画の編集のあと) | rc 0。`check-sync.sh`(`PASS: all files in sync.`)、`check-template-purity.sh`、tech-debt の計画の参照、gofmt、golangci-lint(`0 issues.`)、branch の secret scan(`e333200c..823066af`、clean)が通った。ログは `docs/evidence/verify-2026-10-20-024006.log`(`.gitignore` の対象で commit しない) |
| `./scripts/plan-visual.sh digest <plan>` | `3ee6f70482f6`(承認の digest と同じ) |
| tech-debt の新しい行の区切りの `|` | 6 個(隣の 309・310 行と同じ) |

`go test` と `run-verify.sh` は走らせていない。この run はコードも skill も変えておらず、変えたのは markdown 2 file と insight event と、この report である。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 上の Left open の G2〜G5(test report の記録のまま)、Linux

## Insight event

`./scripts/insights-append.sh --slug secret-scan-test-tmp --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-20-secret-scan-test-tmp.jsonl` に 1 行足した。
