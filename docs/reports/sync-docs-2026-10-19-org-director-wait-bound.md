# sync-docs report: org-director-wait-bound

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-19-org-director-wait-bound.md`(8b-2c。承認の digest `505a695e5548` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `3a803a87`、branch は `fix/org-director-wait-bound`。run 開始時の HEAD は `8787be18`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-19-org-director-wait-bound.md`、`docs/reports/verify-2026-10-19-org-director-wait-bound.md`(pass。INFO 2)、`docs/reports/test-2026-10-19-org-director-wait-bound.md`(pass。LOW 1 は 9eb98897 で直した、INFO 1)
- この run が変えたのは、tech-debt の 1 行(248 行)の 1 文、計画の Progress checklist の 1 項目への注、この report、insight event。計画の本文、先行 3 report、skill、コードには触っていない

## Summary

`3a803a87...8787be18` の差分(17 file)のうち、文書にあたる面を skill の最終の文(de2b05d1 のあと)に突き合わせた。変わったのは `/director` skill の Claude Code の wait(`--timeout-ms 1500000` を Monitor で見張り、Monitor の `timeout_ms` に `1800000`)と、終了コードの扱い(時間切れの打ち直し、Monitor の期限切れの例外)で、Go のコードと CLI の動きは変わっていない。

ずれは 1 点だった。tech-debt 248 行の「Not checked」が、実機の確認のあとに変えた skill の文(dc98d5ec が箇条の文を狭めて signal の例外をその wait 自身の Monitor に結び、de2b05d1 が終わりを先に見たときも決められる形にした)を、実機では打っていないことを書いていなかった。livefire の evidence の「Not checked here」(de2b05d1 で足した)と仕様の 8b-2c の注(同じ commit で足した)と verify report は書いている。self-review の M1 が求めた「記録の確かめていないことを evidence 以上に揃える」が、de2b05d1 では tech-debt の行にだけ届いていなかった。1 文足して直した。

そのほかの面は、skill の変更に対して正しいままだった(下の表)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(248 行の Debt 欄の「Not checked」) | 「and a headless director started with `ralph org director start`」を、「a headless director started with …, and the skill's text after the live check」に替え、括弧で 5aea657f の文を読んだセッションと、そのあとの dc98d5ec、de2b05d1 の文を実機で打っていないことを足した。ほかの列、区切りの `|` の数(6)、計画の道は変わっていない |
| `docs/plans/active/2026-10-19-org-director-wait-bound.md`(Progress checklist) | 「Test artifact created」の下に、この run の注を 1 項目足した。本文(Progress checklist より上)は変えていない。digest は前後で `505a695e5548` |
| `docs/reports/sync-docs-2026-10-19-org-director-wait-bound.md` | 新規 |
| `docs/insights/events/2026-10-19-org-director-wait-bound.jsonl` | sync_docs の event を 1 行足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md` の director の段落(258 行)と `[org.director].wake` の段落(260 行) | 合っている | director が `ralph org wait --inbox --director <id>` で待つと書き、時間切れも `--timeout-ms` の値も書かない。skill の数字を写していないので、今回の変更では古くならない。`--timeout-ms 0` はない |
| `README.md` の動詞の表(124 行)、`ralph status` の行(125 行) | 合っている | wait の区切りの話を持たない |
| `docs/recipes/director-wake.md`(冒頭、「When to stay on pull」)、`docs/recipes/director-launch-cmd.md`(188 行) | 合っている | pull を `ralph org wait --inbox --director <id>` と書くだけで、無期限とも `--timeout-ms 0` とも書かない。`forever`、`unbounded`、`no timeout` の grep に当たりなし。root と `templates/base/docs/recipes/` は `check-sync.sh` が同じと確認した |
| `AGENTS.md` の Repo map | 変更なし | 載せている面(`internal/org/` のファイルの列、`.claude/skills/`、`.agents/skills/`)は、ファイルの追加も改名もない。`director_wait.go` の行は wait の印を書くが、wait の区切りと runner は書かない |
| `CLAUDE.md`、`.claude/rules/ralph/*.md`(root と `templates/base/`) | 合っている | `director` の語を持つのは `ralph-workflow.md` の 10 行と 28〜32 行だけで、昇格と headless の起こし方を書き、wait の形は持たない。`agent-messaging.md` に `director` の語はない。`post-implementation-pipeline.md`、`subagent-policy.md`、`model-routing.md` に関係する変更はない |
| `docs/architecture/repo-map.md`(48 行) | 合っている | `.claude/skills/director/` の説明は昇格、受信箱、進捗の報告で、wait の形を持たない |
| `docs/quality/`(`definition-of-done.md`、`quality-gates.md`) | 合っている | `quality-gates.md` の 79・80 行は見張りと watch gate で、skill の wait の形を書かない。`definition-of-done.md` に director の行はない |
| `.claude/skills/org/SKILL.md`(175 行、318 行) | 変更なし | 175 行の `--timeout-ms 0`(無期限)は `ralph org wait` の動詞の説明と人の wait の例で、director の wait ではない(依頼のとおり範囲外)。318 行は `wait --inbox` で待てることだけを書く |
| `internal/cli/org.go` の `wait` の help(1243〜1282 行) | 合っている | 「0 waits unbounded」はコマンドの動きの説明で、skill が `--timeout-ms 0` を打たなくなっても CLI は無期限の wait を受け付けるので、正しいまま。tech-debt 248 行も「the command still accepts an unbounded wait」と書く。Go のコードは変えていない |
| `internal/org/prompts/director.md`(42〜44 行) | 変更なし | 「skill のとおり `--timeout-ms` を足して打つ」と書き、数字を持たない。`3a803a87` の版と `cmp` で同じバイト |
| `docs/specs/2026-10-07-org-multi-org-director.md` FR-10(91 行)と 8b-2c の注(98 行) | 合っている | 98 行は 25 分と 30 分の数字、Monitor の期限、signal の例外、実機の確認が 5aea657f の文であること、そのあとに書き方を変えたことを書く。de2b05d1 が足した「そのあとに書き方を変えた」の文は 98 行に入っている。93 行の「Claude Code は Monitor で待ち、codex は timeout 付きの wait を繰り返す」は FR-10 の元の要件の文で、98 行の注がそれを区切り付きの形に読み替えたことを書くので、書き換えない。214 行(codex の pull)は変更なし |
| `docs/tech-debt/README.md` 247 行 | 合っている | Trigger の欄は 8324476b で、終わった 8b-2c を指さない形に直してある(self-review が確かめた) |
| `docs/tech-debt/README.md` 248 行 | 直した | 上の Changes made |
| `docs/tech-debt/README.md` のファイル行数の行(306 行) | 変更なし | この PR の `internal/org/director_skill_sync_test.go` は 629 行で、800 行の線の下。行が載せるのは 800 行を超えたファイルだけ |
| `docs/tech-debt/README.md` の skill の cell の長さの行(174 行) | 変更なし | その行が数えるのは `/org` skill の `stop` の cell などで、今回の差分は `/director` skill だけ。director の skill の長さを載せる行はない(`director/SKILL.md` を名指す tech-debt の行も grep に当たらない) |
| skill の 4 つの写し(`.claude/`、`templates/base/.claude/`、`.agents/`、`templates/base/.agents/`) | 合っている | `check-skill-sync.sh` が 14 skill で lock-step。`cmp` で root と template が同じバイト。`.agents` の写しは S1 が `sync-skills.sh` で作っていて、この run は skill に触れていない |
| `scripts/check-sync.sh` | pass | `PASS: all files in sync`(ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| 計画 | 合っている | Progress checklist に S1〜S6、self-review、再レビュー、verify、AC9、test の記録がある。計画の本文は digest を保つため、D3 と R1 の「Monitor の期限切れの知らせのあと」の前提を直していない(Progress の 128・129 行が、skill が終わりを見た時点で決める形に変わったことを書いている) |

## Plan drift notes

- D3 と R1 は、Monitor の知らせが wait の終わりの「あと」に来る前提で書かれている。skill の最終の文は、終わりを見た時点で知らせが届いていれば打ち直し、届いていなければ失敗とし、あとから届けばそこで打ち直す(de2b05d1)。計画の本文は承認の digest を保つために変えず、ずれは Progress の 128・129 行と仕様の注に書いてある。この run は足していない

## Left open

- `internal/org/director_skill_sync_test.go` が固定するのは数、`--timeout-ms 0` がないこと、Claude Code の箇条と小節の文の断片(9eb98897)。仕様の 98 行は「数字は …のテストが 4 つの写しで固定する」と書き、断片の固定には触れない。書いてあることは正しいので書き換えていない
- verify の V1(D6 のモデルの書き分け)と V2(Monitor の既定と最大の出どころ)、self-review の I2(248 行の長さ)と I3(表のテストの名前)は、先行 report が merge に要らないと判断したもので、この run でも直していない
- 本物の 25 分と 30 分、Bash `run_in_background` の実機、codex、headless の director、何時間も動く director、de2b05d1 までの skill の文を実機で打つこと(`/director` skill の文に従うセッションの確認)。いずれも tech-debt 248 行、livefire の evidence、仕様の注が「確かめていない」と書く

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `./scripts/run-static-verify.sh`(doc の編集のあと) | rc 0。`check-skill-sync.sh`(14 skill)、`check-sync.sh`、`check-pipeline-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照、gofmt、golangci-lint(`0 issues.`)、branch の secret scan(`3a803a87..8787be18`、clean)が通った。ログは `docs/evidence/verify-2026-10-19-142448.log`(`.gitignore` の対象で commit しない) |
| `./scripts/plan-visual.sh digest <plan>` | `505a695e5548`(承認の digest と同じ) |
| tech-debt 248 行の区切りの `|` | 6 個(見出しの行と同じ) |
| `cmp`(`internal/org/prompts/director.md` と `3a803a87` の版) | 同じバイト |

`go test` と `run-verify.sh` は走らせていない。この run はコードも skill も変えておらず、変えたのは markdown 2 file と insight event と、この report である。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 上の Left open の実機の面、Linux、`go test`(コードの変更なし)

## Insight event

`./scripts/insights-append.sh --slug org-director-wait-bound --flow standard --phase sync_docs --verdict pass --cycle 1 --source skill` で `docs/insights/events/2026-10-19-org-director-wait-bound.jsonl` に 1 行足した。
