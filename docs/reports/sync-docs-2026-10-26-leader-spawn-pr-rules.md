# sync-docs report: leader-spawn-pr-rules

- Date: 2026-10-26
- Plan: `docs/plans/active/2026-10-26-leader-spawn-pr-rules.md`(承認の digest `7fb9a3d2e3cf` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run は計画を触れていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `619cf5f8`(origin/main)、branch は `fix/leader-spawn-pr-rules`。run 開始時の HEAD は `ba3c877c`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-26-leader-spawn-pr-rules.md`(Merge OK。LOW 2 件は 0309acf8 で直った)、`docs/reports/verify-2026-10-26-leader-spawn-pr-rules.md`(pass。LOW 3 件 V1〜V3 と情報 2 件)、`docs/reports/test-2026-10-26-leader-spawn-pr-rules.md`(pass。mutant 12 個がすべて落ちた。Test gaps 4 件)
- 差分の大きさ(`git diff 619cf5f8...HEAD --shortstat`、この run の前): 16 ファイル、+604/-59。walkthrough を書くかどうかは `/pr` が決める

## Summary

この PR が変えた振る舞いは 3 つある。`ralph org start --plan` が leader に渡すタスクの `- 予約したパス:` の行に、座席の spawn に付ける `--scope` の引数(`shellQuote` で引用したもの)が入った。leader の雛形(`internal/org/prompts/leader.md`)は、ミッションの 2 と手順 3・5 でその引数を implementer と reviewer の spawn に書き換えずに付けさせ、手順 14 では PR のタイトルと本文を `/pr` skill の手順(言語、タイトルの prefix)と `template.md` に合わせさせる。`/org` skill はその 2 か所(leader のサイクルの手順 2、「org の終わり方」の手順 6)を同じ向きにした。

この振る舞いについて、古い形のまま残った記述を AGENTS.md、README、`docs/architecture/`、`docs/recipes/`、`docs/quality/`、`.claude/rules/ralph/`、`/director` skill、`/org` skill の残りの節、仕様と根拠の表で探したが、見つからなかった(下の「Surfaces checked for drift」)。

この run が直したのは `docs/tech-debt/README.md` の 3 行で、どれも Trigger がこの PR で当たったのに、引き継ぎの注がなかった(verify の V1〜V3)。どの行も RESOLVED にしていない。足したものは insight event とこの report である。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(RC4 の行。V1) | Trigger の末尾に「2026-10-26: fix/leader-spawn-pr-rules hit this trigger ... did not pay it: RC4 is in that plan's Non-goals ... Carried over to the next change that meets this trigger」を足した。当たった理由は雛形の手順 3・5・14 とミッションの 2 の変更で、払わなかった理由は、ユーザーが RC2 と RC5 だけを release の前に直すと決めたこと(計画の Non-goals)と、この PR が手順 4 と `/org` skill の分割計画の例のどちらにも触れていないこと。Related に計画のパス(`docs/plans/active/2026-10-26-leader-spawn-pr-rules.md`、Non-goals)を足した |
| `docs/tech-debt/README.md`(ファイルの大きさの行。V2) | Trigger の末尾に同じ形の注を足した。当たった理由は `TestRenderRolePrompt_Leader_FeatureOrgProcedure` の変更で `prompts_test.go` が 2165 行から 2189 行になったこと(+24、閾値の +300 より下。ほかの 4 ファイルは触れていない)。払わなかった理由は、移すだけの分割がこの計画の Scope(S4 がテストの変更を名指す)の外で、行の「A move-only split, separate from this PR's behavior」が今も当たること。Related に計画のパス(Scope S4)を足した |
| `docs/tech-debt/README.md`(org-feature-worktree の小さな所見の行。V3) | (g) を、この行の (f)・(j) と同じ書き方で直した。最初のセルの (g) を `~~...~~` で消して「f1b0352e(2026-10-14)で直った。導入が手順 9 の reviewer のレポートを挙げる」と書き、Impact の (g) にも `~~...~~ (closed)` を付けた。Trigger の末尾には、この PR が同じ節(ミッションの 2、手順 3・5・14)を編集して (g) と (f)・(j) の Trigger に当たったが、(g) は f1b0352e で、(f) と (j) は b0fc4c59 で直っていて払うものがなかったこと、開いている (a)〜(e)・(h)・(i)・(k)・(l) はこの計画の Scope の外で、それぞれの Trigger に残ることを書いた。行は RESOLVED にしていない。Related に計画のパス(Scope S1)を足した |
| `docs/insights/events/2026-10-26-leader-spawn-pr-rules.jsonl` | `sync_docs` の event を 1 行足した(`./scripts/insights-append.sh --slug leader-spawn-pr-rules --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`) |
| `docs/reports/sync-docs-2026-10-26-leader-spawn-pr-rules.md` | 新規(この report) |

3 行とも 5 欄のまま(`(?<!\\) \| ` で分けて数えた)。V3 の行の `~~` は各欄で偶数個。新しい注は 3 行とも英語(このファイルの言葉)で、`|` を含まない。計画のパスは `/pr` の `archive-plan.sh` が `docs/plans/archive/` に書き換える形(`docs/plans/active/<name>` のあとが空白か `(`)で書いた。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md`(repo map の `internal/org/`、`.claude/skills/` と `scripts/` の行) | 変更なし | `feature.go` を「worktree + bound leader with the recorded `[org.approval].mode`」と書き、タスクの行の中身には触れない |
| `README.md:252`(機能の org の段落) | 変更なし | 「leader が `gh pr create` で PR を開く」とだけ書き、spawn の flag も PR の言語も書かない(`Japanese` と `日本語` の当たりは README 全体で 0 件) |
| `docs/architecture/repo-map.md` | 変更なし | `feature.go`、`leader.md`、`featureLeaderTask` の名前を出さない(`grep` で 0 件) |
| `docs/recipes/`(`isolated-herdr-server.md`、`director-wake.md`、`director-launch-cmd.md`、`codex-seat-permissions.md`、`codex-setup.md`)と `templates/base/docs/recipes/` | 変更なし | leader の spawn と PR の言語を書く recipe はない。`--scope` が出るのは `codex-seat-permissions.md`(2 写し)の scratch の座席の例 `--scope "<scratch-cwd>/**"` だけで、leader の雛形とは別の手順。`isolated-herdr-server.md` の `autonomous` の当たりは claude の bypass mode の説明で、spawn の検査ではない |
| `docs/quality/quality-gates.md`(Envelope validation の行、77 行)と template の写し | 変更なし | 「`--reserve` path overlap ... Spawn rejected, recorded in manifest」は、spawn の検査を変えていないので今も当たる(計画の Non-goals) |
| `.claude/rules/ralph/`(`agent-messaging.md`、`model-routing.md`、`subagent-policy.md`、`ralph-workflow.md`) | 変更なし | `予約したパス` と `--scope` の当たりは 0 件。leader のタスクの行の形を書く規則はない |
| `.claude/skills/director/SKILL.md`、`internal/org/prompts/director.md` | 変更なし | `--scope` の当たりは 0 件(`scope` を含む行は 660 行の `scope_change` の ALERT の説明だけ)。PR の言語を書く文もない |
| `.claude/skills/org/SKILL.md` の `spawn` の行(208 行)と「`--reserve`」の節(300〜303 行) | 変更なし | 「`--scope` は担当範囲の説明。autonomous では必須」「`--reserve` を渡した spawn は `--scope` 必須のゲートを満たす」は、雛形のミッションの 2 と合う。`start --plan` が leader を spawn するときの scope は `split <id> feature <slug> (reserve: <paths>)` で、座席の `--scope` とは別の値 |
| `.claude/skills/org/SKILL.md`(1002〜1010 行、1306〜1313 行)と 3 つの写し | 同期 | この PR が直した 2 か所。`check-skill-sync.sh` が「14 skill(s) in lock-step」、`check-sync.sh` が IDENTICAL 172、DRIFTED 0 |
| `.claude/skills/org/SKILL.md` の `start --plan` の説明(651〜670 行) | 変更なし(足さないと決めた) | タスクの行の並びは今も正しい。`- 台帳:` の行には段落 1 つ分の説明があるが、`- 予約したパス:` の行の `--scope` の引数の説明は 1306〜1313 行にだけある。古い形を書いた文ではない。足すと 1002 行と 1306 行より前に行が入り、self-review と verify の report が引く行番号がずれる(下の「見送り」) |
| `.claude/skills/pr/SKILL.md`、`template.md` と 3 つの写し | 変更なし | この PR の Non-goals。雛形の手順 14 が指す手順 5 の「PR title and body must be written in Japanese」は 4 つの `SKILL.md` にあり、`template.md` の 2 写しに言語の決まりはない(verify が確かめた) |
| `docs/specs/2026-10-07-org-multi-org-director.md`、`docs/evidence/org-spec-evidence-2026-10-21.md` | 同期 | 仕様の `- [ ]` は base も HEAD も 5、`- [x]` は 27(下の「数の突き合わせ」)。FR-4 と受け入れ条件 5 は `[ ]` のままで、印(2026-10-26)が「テストで固定したこと」と「実機で確かめていないこと」を分けて書く(self-review の F1 の直しのあと) |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし | `--scope` は「spawn `--scope` の manifest 記録+役割プロンプト指示」と書く(176 行)。この PR は記録の形を変えていない |
| `docs/tech-debt/README.md` の RC2・RC5 の行 | 同期 | RC2 と RC5 は取り消し線で、ブランチ名の注を持つ(364、368 行)。開いた RC は RC1、RC3、RC4、RC6、RC9、RC10、RC11 の 7 行で、仕様の Status と根拠の表の「未定」の 7 行と合う |
| `tests/`、`scripts/`(`予約したパス`) | 変更なし | 当たりは 0 件。タスクの行を読むのは雛形とテスト(`feature_test.go`、`prompts_test.go`)だけ |

## 見送り

1. `/org` skill の `start --plan` の説明(651〜670 行)に、`- 予約したパス:` の行の `--scope` の引数を足す案(verify の情報)。直さなかった。誤った文はなく、leader が読む手順(leader のサイクルの手順 2)と雛形に書いてある。足すと 4 つの写しで 1002 行より前に行が入り、self-review と verify の report が引く `:1002-1010` と `:1306-1313` がずれる。この節を次に編集するときに足せる。
2. leader が実際に `--scope` を書き換えずに付けるか、PR の題と本文を日本語で書くか、spawn が検査を通るか。実機の確かめで、計画の Non-goals と R3 のとおり次の実行で行う。FR-4 と受け入れ条件 5 の `[ ]` はそのまま。
3. codex の leader が `.agents/skills/pr/SKILL.md` を読むこと。ファイルがあることだけを verify が確かめた。

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| verify の V1(RC4 の行) | 上の Changes made のとおり、Trigger に引き継ぎの注を足した。RC4 は直していないので RESOLVED にしない |
| verify の V2(ファイルの大きさの行) | 同じく注を足した。計画の Non-goals に分割は書かれていないので、「Scope(S4)の外」と書いた |
| verify の V3((g) の古い記述) | (g) が f1b0352e で直っていることを、行の (f)・(j) と同じ書き方(取り消し線と「Closed in」)で書いた。ユーザーの指示は「直っていることを書く」で、行は RESOLVED にしていない |
| verify の情報(`/org` skill の `start --plan` の説明) | 見送り(上の 1) |
| verify の情報(計画の Progress の「Review artifact created」) | 直し済み(ba3c877c で `[x]`)。残る `[ ]` は「PR created」だけで、`/pr` が付ける |
| test の LOW の穴(`$(...)`、`~`、`!`、先頭か末尾の `'`、`/bin/sh` だけで読ませること、sh の構文の誤りで止まること) | tech-debt の行にしない。`shellQuote` はすべての文字を単一引用符で包むので `$` とバッククォートの事例が通れば同じ形の入力も通り(probe の 9 事例 × `/bin/sh`・zsh・bash の 27 通りが一致した)、構文の誤りで止まるのは落ちる側の動きで、判定が変わらない |
| test の「既知の不安定」(`TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing/SIGTERM`) | この run では行を足さない。slice 2 で 1 回(load average 約 12)落ち、`/test` の 3 回の全体の実行では落ちていない。tech-debt に行はなく、もう一度落ちたときに足すかを決める(test の report の判断のまま) |
| test の Test gaps 1・4(実機、パッケージ全体の coverage) | 文書には書かない。1 は計画の R3 と Non-goals、4 は差分が `featureLeaderTask` の 1 行と雛形の文で、coverage の数に効かない |

## 数の突き合わせ

| 数 | 値 | 同じ数を言う場所 |
|----|----|------------------|
| 仕様の `- [ ]` と `- [x]` | base 5 と 27、HEAD 5 と 27 | 仕様の Status、根拠の表の 105 行(判定の列は 26 と 5)。この PR は印の数を増やしていない |
| tech-debt の RC の行 | 開いた 7 行(RC1、RC3、RC4、RC6、RC9、RC10、RC11)、取り消し線の 2 行(RC2、RC5) | 根拠の表(52〜63 行の RC1〜RC12 の 12 行)の「直す PR」が「未定」の 7 行、仕様の Status の「RC1、RC3、RC4、RC6、RC9、RC10、RC11 の 7 つは直しておらず、開いたまま残る」 |
| `internal/org/prompts_test.go` の行数 | base 2165、HEAD 2189(`wc -l`) | V2 の注の「2165 から 2189」、tech-debt の行の「2165」(#253 の時点) |
| leader の雛形の差分の範囲 | ミッションの 2、手順 3・5・14 の 4 か所(`git diff -U0` の hunk が 4 つ。手順 4 は触れていない) | V1 の注の「steps 3, 5, and 14 ... and item 2 of its mission」と「neither step 4」 |
| (g) の直し | f1b0352e は base の祖先(`git merge-base --is-ancestor`)。現在の導入が「9 の reviewer のレポート」を挙げる(`leader.md:68`) | V3 の注 |

## Plan drift notes

- 計画の Progress は「PR created」を除いて `[x]`。この run が計画を触れていないので digest は承認のまま(`7fb9a3d2e3cf`)
- この run が足した tech-debt の 3 行の注は、計画の Affected areas には挙がっていない。計画の Non-goals(RC4 を直さない)と Scope(S4 のテスト、S1 の雛形)を引いて引き継ぐだけで、スコープは広げていない

## Left open

- 実機の確かめ(FR-4 と受け入れ条件 5 の `[ ]`)。release のノートに「leader の雛形が座席の spawn に `--scope` を渡し、PR の題と本文を `/pr` skill の手順に合わせる」と書く(計画の Rollout)
- 開いたままの tech-debt の行: RC1、RC3、RC4(この run で引き継ぎの注を足した)、RC6、RC9、RC10、RC11。仕様と根拠の表の「leader への ALERT」、Codex の leader、watcher の時間切れ、FR-10、N4

## Verification

| Check | Result |
|-------|--------|
| `./scripts/run-verify.sh`(出力は scratchpad のファイルに書いた。tech-debt の 3 行を直したあとで、この report と insight event を足す前に始めた。07:27〜07:48 UTC) | rc 0、終わりの行は `==> All verifiers passed.`。`tech-debt README plan references` は `OK`、`check-skill-sync.sh` は「14 skill(s) in lock-step」、Go は `gofmt: ok`、golangci-lint は `0 issues.`、`go test` は 8 パッケージが `ok`(`internal/cli` 488.688s、`internal/org` 580.721s。ほかの 5 つは `(cached)`)、`secret-scan-branch: scanned 619cf5f8..ba3c877c against origin/main: clean`。`FAIL` で始まる Go の行は 0 件。run-verify は証拠として `docs/evidence/verify-2026-10-26-072743.log`(git が無視する場所)も書いた |
| `./scripts/check-sync.sh`(この report と event を足したあと) | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-26-leader-spawn-pr-rules.md` | `7fb9a3d2e3cf`(承認の digest と同じ) |
| `git diff --check`(作業ツリー) | 出力なし、rc 0 |
| tech-debt の 3 行 | 5 欄のまま、V3 の行の `~~` は各欄で偶数個。この run が変えたファイル 3 つの U+FFFD は 0 個、`/Users/` は 0 件 |
| natural-japanese の lint(この report) | warn 1 件(`yomiyasu_halfwidth_space`、13 行)。`digest`、`run`、`Scope` のような英語の語と助詞のあいだの空白で、前の sync-docs の report と同じ書き方なので直していない |
| テスト | この run は Go のコードもスクリプトも触れていない(変えたのは tech-debt の文、report、event)。run-verify が上のとおり全テストを通した |

## Insight event

追記した。`./scripts/insights-append.sh --slug leader-spawn-pr-rules --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` が rc 0 で、`docs/insights/events/2026-10-26-leader-spawn-pr-rules.jsonl` の末尾の行(cycle 1、`ts` は 2026-10-26T07:48:10Z)になった。UTC の日付が 2026-10-26 だったので、self-review、verify、test の event と同じファイルに入り、新しいファイルは増えなかった(`git status --short` で確かめた)。
