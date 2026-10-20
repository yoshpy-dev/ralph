# sync-docs report: org-trial

- Date: 2026-10-20
- Plan: `docs/plans/active/2026-10-20-org-trial.md`(承認の digest `5b82ee1e30a1` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `a1ce47d8`(origin/main)、branch は `chore/org-trial`。run 開始時の HEAD は `a54f7999`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-20-org-trial.md`(Merge yes。M1・L1〜L5 は a288d23b で直し済み、N1 は 8e1d5c40 で直した)、`docs/reports/verify-2026-10-20-org-trial.md`(pass)、`docs/reports/test-2026-10-20-org-trial.md`(pass。T1 は a54f7999 で直した)
- この run が変えたのは、tech-debt の 3 行(文の直しだけ)、計画の Progress checklist、この report、insight event。コード、skill、rule、template、先行 3 report、証拠(`docs/evidence/org-trial-2026-10-20.md` と `docs/evidence/org-trial-2026-10-20/`)、計画の本文には触っていない

## Summary

`a1ce47d8...a54f7999` の差分は 29 ファイルで、すべて `docs/` の下にある(`git diff --name-only origin/main...HEAD` で `docs/` 以外は 0 件)。内訳は証拠 1 本とその snapshot(21 ファイル)、仕様の FR-12 の tick と注と Open questions、tech-debt の行、計画、self-review・verify・test の 3 report、insight event。公開の面、CLI、skill、hook、rule、既定値は変わっていない。

振る舞いや手順を書く文書(`README.md`、`AGENTS.md`、`CLAUDE.md`、`.claude/rules/ralph/`、`docs/recipes/`、`docs/quality/`、`docs/architecture/`、`/org` と `/director` の skill、leader と director の雛形)を、試行の記録と突き合わせた。試行と食い違う記述は見つからなかった。試行の所見(F1〜F14)のうち、文書が今の動きを書いているものは、動きが合っている。文書が書かず、記録だけが持つものは、tech-debt の行が持つ。

直したのは tech-debt の 3 点。新しい行 2 本の code の行番号を名前にし、F11 の行から同じ hook と同じ report の置き場の古い行 2 本へ道を付けた。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(行 322、`escalateAlert` が通知の error を捨てる行) | Debt 欄の「(line 1557)」を外した。同じ文が `escalateAlert` と `_ = w.escalateFn(...)` を名指すので、文は変わらず読める。行番号は次にこの file を触る PR で動く |
| `docs/tech-debt/README.md`(行 323、`BuildOrgReport` の event timeline の行) | Debt 欄の `internal/org/report.go:180` を「the one `strings.ReplaceAll` of `details` there」に替えた。行の頭が `BuildOrgReport` と `internal/org/report.go` を名指す。書き換えで未エスケープの `|` を足さないよう、`"|"` を含む形は避けた(最初の案は欄を 1 つ増やしたので、欄の数を数え直して直した) |
| `docs/tech-debt/README.md`(行 320、F11 の session-end WIP hook の行) | Related 欄に「the older rows on the same hook (`session_end_summary.sh` committing a reviewer's leftovers, org-feature-worktree Run 3) and on `ralph org report` writing under the directory it runs in」を足した。行 191 と 192 が同じ hook と同じ置き場の話を持ち、行 320 の Trigger の選択肢(`ralph org report` を worktree の外に書かせる)は行 192 の選択肢と同じなので、片方だけ直すともう片方が残る |
| `docs/plans/active/2026-10-20-org-trial.md`(Progress checklist) | 「Test artifact created」の下に sync-docs cycle 1 の注を足した。本文(Progress checklist より上)は変えていない。digest は前後で `5b82ee1e30a1` |
| `docs/reports/sync-docs-2026-10-20-org-trial.md` | 新規 |
| `docs/insights/events/2026-10-20-org-trial.jsonl` | sync_docs の event を 1 行足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| 差分の範囲 | docs だけ | `git diff --name-only origin/main...HEAD` が 29 ファイル、`grep -vc '^docs/'` が 0。code・script・skill・rule・template は変わらず、既定値も変わっていない |
| 9 段目、FR-12、試行を名指す文書(`README.md`、`AGENTS.md`、`CLAUDE.md`、`.claude/rules/ralph/`、`docs/recipes/`、`docs/quality/`、`docs/architecture/`、`/org` と `/director` の skill、`docs/specs/2026-08-01-org-runtime.md`) | 合っている | `stage 9`、`9 段目`、`trial`、`試行` の grep が 0 件。`FR-12` が当たる `internal/cli/status.go` と `docs/specs/2026-08-17-overlay-scaffold-v2.md` は別の仕様の FR-12(scaffold ownership)で、この試行と関係がない |
| 4 つの既定値(`max_orgs` 10、`max_total_seats` 30、`ack_minutes` 10、`resolve_minutes` 60) | 合っている | `internal/config/config.go` の `Default()`(551・552・569・570 行)、`templates/base/ralph.toml`(47・50・144・145 行)、`README.md` の watch の段落(`ack_minutes` と `resolve_minutes` の既定)が同じ値。仕様の FR-12 の注は「4 つとも変えない」と書く |
| リポジトリの ruleset を書く文書 | 合っている | `ruleset` の grep が当たる live の文書(`README.md` の merge の段落、`/org` と `/director` の skill)は、`ralph org merge` が求める条件を一般に書くもので、このリポジトリの ruleset の状態は書かない。「このリポジトリの ruleset には必須の検査がない」と書く行は tech-debt の ruleset の行(行 299 とその上の RESOLVED コメント)だけで、試行の中で閉じてある |
| 所見 F3(agmsg が herdr の agent を改名)、F1・F2(隔離した herdr の設定) | 合っている | `AGMSG_SELF_NAME`、`agent_name_taken`、`CLAUDE_ENV_FILE`、`GIT_CONFIG_GLOBAL`、`GH_CONFIG_DIR` を live の文書と skill と雛形で grep して 0 件。「隔離した herdr で試す手順」を書く recipe は `docs/recipes/` になく、行 313 が「recipe がない」と書くのと合う |
| 所見 F6(watchdog の escalation は人へ直接)、F13 | 合っている | `README.md` 256 行、`docs/quality/quality-gates.md` の Watchdog の行、`/org` skill の watch の行は、deadman を人への escalation と書き、watchdog の通知が inbox を通るとは書かない。行 317 と行 322 が今の動きを書く |
| 所見 F8・F11(feature org の gate と WIP hook) | 文書は今の動きを書く。食い違いは register が持つ | `/org` skill の feature org の手順と `check-sync.sh` の除外一覧の食い違いは、行 319 が「別の小さな PR で直す。設計の選択が開いている」と書く。skill を今ここで直すと、その選択を先取りするので触らない。WIP hook は `git-commit-strategy.md` の「End-of-Session」の節どおりに動き、節の記述と食い違わない |
| 停止と再開(Part 1、Part 2) | 合っている | `/org` skill の `start` の行、`leader.md`、`director.md`、`README.md` に、止めた org を start し直したときの leader の動きを約束する文がない(`再開`、`start し直`、`restart`、`resume` の grep は一般の文だけ)。仕様の Open questions の新しい項(計画の承認を待つ間に止めた org)が、試していない形を書く |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 合っている | FR-12 だけが `[x]` で、FR-1〜FR-11 は `[ ]` のまま。注が「ほかの FR の checkbox は今までどおり付けない」と書くのと合う。Open questions の 3 項は FR-12 の注を指し、数字を重ねて持たない |
| `docs/tech-debt/README.md` | 直した(上の 3 点) | 足した 12 行と書き換えた行の欄の数は、未エスケープの `|` が 6 個(5 欄)。形の崩れた行は行 22(7)、行 32(5)、行 278(10)の 3 行だけで、origin/main と同じ。code の `file:line`(`\.(go\|sh\|md):\d+` と `(line N)`)は、足した 20 行のうち 2 行だけで、直した。`run-static-verify.sh` の「tech-debt README plan references」は OK |
| 証拠の中の `watch.go:1557` と `report.go:180` | 触らない | 証拠は commit 時点の記録で、行番号をその時点のものとして持つ。register の行は動き続ける code を指し、次に `watch.go` か `report.go` を触る PR で行番号が動くので、register だけを名前にした |
| `docs/architecture/repo-map.md`、`AGENTS.md` の Repo map | 変更なし | `docs/evidence/` は repo-map に「what counts as evidence」で載る。新しい種類の file も directory もない |
| `templates/base/`、skill の mirror | 変更なし | `check-sync.sh` は `PASS: all files in sync.`(この branch は root 側も template 側も触っていない)。`check-skill-sync.sh` は 14 skill が lock-step |
| 計画 | 合っている | Progress checklist に self-review、verify、test の印と、deviation(D6 の変形、F8、interventions.log、R1・D-F3・F10、AC8 の行番号、I6)がある。AC1〜AC10 はすべて印が付き、`PR created` だけが開いている |

## Plan drift notes

- 計画の D6 は「計画の承認を待つ org を止める」としていたが、試行は PR のあとに止めた形で走った。逸脱は計画の Progress の注と証拠の Stop and restart が持ち、仕様の Open questions の新しい項が試さなかった形を書く。計画の本文は digest を保つために変えていない
- 計画の Non-goals は ruleset の変更を人の作業とするが、director が人の言葉で変えた(I6)。計画の注、証拠の Trial deviations、tech-debt の RESOLVED コメントの 3 か所が書いている

## Left open

- F8 の直し(`check-sync.sh` の除外か、reviewer の report の名前か、`ralph org report` の書き出し先)は別の小さな PR に残る。それまで `/org` skill の feature org の手順は、この repo の gate を通らない(行 319)
- 行 313(隔離した herdr の recipe がない)は row のまま。書くのは次の live check か試行のとき
- 証拠の `docs/evidence/org-trial-2026-10-20/reports/` の 2 file は U+FFFD を逐語で持つ。mojibake hook は Edit と Write でしか走らないので、読むだけなら何も落ちない(self-review の I1。証拠の Snapshot の節が「Edit しない」と書く)

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(tech-debt と計画の編集のあと) | rc 0。`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、branch の secret scan(`a1ce47d8..a54f7999`、clean)が通った。scope は `changed`(`docs_only`)で、言語の pack は選ばれず、gofmt と golangci-lint は走っていない。ログは `docs/evidence/verify-2026-10-20-100810.log`(`.gitignore` の対象で commit しない) |
| `./scripts/check-sync.sh` | rc 0、`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | rc 0、`14 skill(s) in lock-step` |
| `./scripts/plan-visual.sh digest <plan>` | `5b82ee1e30a1`(承認の digest と同じ) |
| tech-debt の行の形(未エスケープの `|` の数) | 行 319・320・322・323 は 6 個。形の崩れた 3 行は origin/main と同じ |

`go test` と `run-verify.sh` は走らせていない。この run はコードも skill も変えておらず、変えたのは markdown 3 file と insight event と、この report である。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 振る舞いの文書と所見の突き合わせは grep と該当箇所の読みで、全文書の全文を読み直してはいない。F8 の `/org` skill の手順そのものは、行 319 の記述と self-review の確認(`check-sync.sh` の除外一覧)に頼り、この run では読み直していない。GitHub の ruleset と PR #241・#242 の状態は読んでいない(証拠の記録のまま)

## Insight event

`./scripts/insights-append.sh --slug org-trial --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-20-org-trial.jsonl` に 1 行足す。
