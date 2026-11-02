# sync-docs report: config-path-and-ledger-docs

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-config-path-and-ledger-docs.md`(PR #297 が残した穴と PR #296 が残した穴のうち、tech-debt 396 行の (c)、397 行の残り、421 行の (a)〜(e)。AC1〜AC7、D1〜D3、S1〜S4、S1 の補い a6be3f5e、S4 の補い b16f4bd0、self-review の直し F1 d3bb5b55)
- Branch: `fix/config-path-and-ledger-docs`。run 開始時の HEAD は `12de7ff8`、base は origin/main `350f22ea`(`ea8b4d16` で取り込み済み)
- Pipeline cycle: 1
- 先行 report: self-review(Merge: yes、LOW 5。4 件は `d3bb5b55` で直し、L4 だけ残る)、verify(pass、LOW 1 と INFO 6)、test(pass、`run-test.sh` rc 0。AC6 は一部満たした)
- 制約: Go のファイル、README、AGENTS.md、`/org` の skill は変えていない。plan は AC1〜AC5 と AC7 の箱、/sync-docs の行 2 つだけを変えた。digest は前後とも `917011181315`

## Summary

tech-debt の 396・397・421 行を書き換え、148・411・88 行に、当たったが払わなかった Trigger の注を足した。

- 396 は (c) を払って行全体を閉じた。
- 397 は残っていた README と skill の言葉を払った。行は、変数を `retiredEnvs` から外すときに消すものの一覧として Trigger を残した。
- 421 は (a)〜(d) を閉じ、(e) は codex の model と claude の leader を閉じた。(e) の case 2、(c) の他のファイルの 4 テスト、D1 の代価(パスが 2 回出る形。新しい (f))は Trigger つきで残した。

表の末尾に新しい行は足していない。コード、help、skill、README の文は変えていないので、pipeline の再実行は要らない。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 6 行を書き換えた(88、148、396、397、411、421)。ほかの行は同一。内容のコミットは `1982be83` |
| `docs/plans/active/2026-11-02-config-path-and-ledger-docs.md` | AC1〜AC5 と AC7 の箱を `[x]` にし、Progress checklist に「Docs synced」の行と、AC6 の箱を付けなかった理由の行を足した。AC6 の箱は付けていない |
| `docs/reports/sync-docs-2026-11-02-config-path-and-ledger-docs.md` | この report |
| `docs/insights/events/2026-11-02-config-path-and-ledger-docs.jsonl` | `sync_docs` の行(cycle 1、pass、findings 0)を `insights-append.sh --slug config-path-and-ledger-docs --cycle auto` で足した。既存のファイルに入り、迷子のファイルはできなかった |

## 3 行の扱い

| 行 | 扱い |
|----|------|
| 396(personal file の残り) | (c) を払って全部閉じた。`Load` の doc は理由句が外れて「none of ralph.toml's values applied」で終わり、`StateDir` の doc は `org.ResolveOrgStateDir` が `LocalOrgStateDirAt` で読むと書く(S3、`3311cf16`、コメントだけ。AC4 を `return Default(), err` と `internal/org/statedir.go` の呼び出しに見比べた verify の Met による)。(a)(b) は前の PR が項ごとに閉じた形なので、行全体を取り消し線にせず、同じ形で (c) を足した。Debt は (c) を取り消し線にして「Closed in ...」を付け、Impact は `(closed)`、Trigger は `(done in ..., commit 3311cf16)`。Why deferred には「#269 は PR #298 で merge した」と足した |
| 397(断る条件の言葉) | 「Left open」を払った。S4(`d257824e`)、b16f4bd0、d3bb5b55。「written in five places」を「定数 1 つと 7 つの文(doctor の warn、escalate の help、README の段落、skill の 4 写し)」に直した。Impact の「README と skill の写しはまだ落とせる」を閉じ、Why deferred に「#269 の merge のあとで払った」を足した。Debt には、テストが読むのは README と `.claude` の 2 写しだけで、`.agents` の 2 写しは `check-skill-sync.sh`、`templates/base` の写しどうしは `check-sync.sh` が見ること、skill の日本語の文はテストが読まないことを書いた(定数の doc の L1 と同じ内容)。Trigger の「remove the constant and all five」は「定数、その言葉を持つ文(warn、help、README の段落、skill の 4 写し)、新しいテスト」に直し、「条件を変えるときは定数と一緒に変え、テストが落ちるまで README と code span を直す。日本語の文は手で直す」を足した。「once #269 has merged」は満たしたので外した |
| 421(standard-config-gaps の小さな残り) | 下の表 |

### 421 の項ごと

| 項 | 扱い |
|----|------|
| (a)(b) | `40cedc9e`(`resolveConfig` が読んだパスを前に付ける)と `a6be3f5e`(`ralph status` の warning が名前を 2 回出さない)で閉じた。Debt は (a)(b) を取り消し線にして、閉じ方を 1 つの文で書いた(`org: load config: ` の 4 つの文、変えなかった `config.Load`・`ralph config get`・`ralph doctor`、root のケースと 3 つの動詞のテスト)。Trigger は取り消し線と `(done in ...)`。Why deferred には、`Load` でなく `resolveConfig` で払った理由(D1)を足した |
| (c) | `f185f22d` で `org_start_plan_config_test.go` について閉じた(helper `commitRalphTomls`、呼び出し 3 つ)。同じ組が残るテストは verify I4 の数え方で、3 ファイル 4 か所: `TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart`、`TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings`、`TestOrgStartPlan_ReviewerFromMainWorktree`、`TestOrgStartPlan_ApprovalModeFromMainWorktree`。plan の Progress は「5 つ」と書くが、メッセージが `ralph.toml` の組を数えた verify I4 の数に合わせた。Debt・Impact・Trigger のどれも、閉じた部分に取り消し線を付け、残る 4 テストを名指した |
| (d) | `f185f22d` で閉じた。ただし 3 つのエラーの道のうち、`newOrgSpawnRuntimeAt` の道と `newOrgRuntimeAt` の `org: load config:` の道を打つのは、S1 の `40cedc9e` が足したテストの spawn と `start <task>` のケース。`f185f22d` が足したのは `resolveOrgLedger` の道(`TestOrgStartPlan_RefusedLedgerStartsNothing`)、上限の `max_total_seats` の対照 2 つ、承認方式のテスト(4 ケース)。依頼は「f185f22d で閉じた」と書いたが、verify と Progress の分担のとおりに書いた。`resolveModelOrWarn` の道は、198 行(org-feature-worktree の test gaps)の (e) に残る |
| (e) | codex の model(check A、`gpt-6-luna`)と claude の leader → codex(case 1)を閉じた。case 2(codex の leader → claude)は残した。理由は test report と証跡のとおり: codex の shell tool は zsh で、`~/.config/zsh/.zshenv` が PATH を組み直すため、leader の `ralph` が `/opt/homebrew/bin/ralph` 6.0.0 になり、`config get` が `unknown command "config"` で終わって計画助言が off 扱いになり、claude が起きなかった。補いの手動の実行(`--model sonnet --effort low`、rc 0)は「leader の run ではない」と書いた。Trigger は、次に codex の leader で feature org を実物で打つとき、または `featureLeaderTask` の `[reviewer]` の 2 行の次の変更にして、打つときの手順(leader の `ralph version` と `plan_advisory` を先に読む)を書いた。pane の `PATH` が古い `ralph` を指す問題の行(192 行)と同じ根なので、Why deferred と Trigger の本文で、行番号でなく「the row on a `ralph` of another version on the pane's `PATH`」と名指した(行が動くため) |
| (f) 新規 | D1 の代価。`resolveConfig` の前置きのせいで、`config.Load` が自分でファイルを名指す形はパスが 2 回出る(self-review L4、verify I1・I2・I5)。`TestResolveConfig_PrefixesTheFileItWasAskedToRead` が固定する 2 つ(開けない `ralph.toml`、壊れた `ralph.local.toml`)と、固定していない 2 つ(ディレクトリの `ralph.toml`、`ralph.toml` に書いた個人用だけのキー)、`resolveConfig` を通らない main worktree の文(5 つの関数名)が前からパスを 2 回出していることを書いた。既存の項の続きではなく、閉じた (a)(b) の代価なので、新しい項 (f) にした。Trigger は、`resolveConfig` か `config.Load` のパスを名指す文の次の変更、または二重のパスで迷ったという報告 |

## 当たった Trigger の繰り越しの注

| 行 | 注 |
|----|----|
| 148(`ralph org` の動詞の config と state dir の規則) | Trigger は `resolveOrgConfig` の変更。`resolveOrgConfig` の本体は同じだが、呼ぶ `resolveConfig` が返すエラーが変わった(パスが前に付く)。Debt の末尾の文(`ralph.toml` の検証エラーが teardown の動詞を止める)に「since fix/config-path-and-ledger-docs the error names the file」と足し、Trigger に 2026-11-02 の注(当たったが、開いている部分は config の探し方で触っておらず、`config.Load` は Non-goals で変えない)と Related に plan を足した。前の同じ形の注(feat/standard-flow-config、fix/standard-config-gaps)の後ろに置いた |
| 411(5 つの大きなファイル) | Trigger は「any of the five」。`internal/config/config.go` が 1192 → 1194 行(origin/main `350f22ea` と HEAD で `wc -l`)。コメントだけで、S3 が名指す 2 つの doc。ほかの 4 ファイルは変えていない。前の注(feat/org-cross-review)の後ろに置いた |
| 88(org-debt-batch の LOW 6 件) | Trigger は「対応するファイルを触るとき」。a6be3f5e が `internal/cli/status.go` を触り、(4) の `printStateDirLine` と `state_dir_source` の `omitempty` が残る。これまで `status.go` を触った PR はこの行に注を残していない。今回は注を足した(依頼どおり)。`status.go:263` などの `file:line` の参照は古い(verify は `printStateDirLine` を :503、`omitempty` を :597 に見た)が、書き換えは範囲の外 |

### sweep で退けたもの

全 open 行の Trigger(取り消し線の外)と Debt を、この diff が変えたファイル名・関数名・テスト名・help の語で引いた(python で、取り消し線の外の Trigger と Debt を語の一致で引いた)。当たりに見えて退けたものは次のとおり。

| 行 | 退けた理由 |
|----|------------|
| 413、359、257、267 | `ralph status` の Trigger(節を足す、座席の表、header の行、`--config`)。この branch は `statusOrgConfig` の warning の 1 行だけ(`status.go` 963 → 964 行)で、どれも変えていない |
| 170、186、192、208(`resolveOrgLedger`、`newOrgRuntimeAt`、`newOrgSpawnRuntimeAt`) | 関数は変えていない。この branch のテストが戻りを打つだけ |
| 187、220(`withMainWorktreeOrgLimits`、`applyMainWorktreeOrgLimits`) | 変えていない。上限のテストに対照を足しただけ |
| 198 | (e) の `runOrgStartPlan` は変えていない。`resolveModelOrWarn` の道は plan の Non-goals のとおり残る |
| 177 | Trigger は `/org` skill の command table の次の編集。この branch の変更は「前提」節の台帳の場所の項で、表の外。stop/disband のセルの長さは動かない |
| 149、236、266、369、373、376、385 | Debt だけが SKILL.md を名指す行。Trigger は別の箇所 |
| 386、420 | `org_retired_state_dir_*` の別のテストファイルの行。この branch は新しい 1 ファイル(60 行)を足しただけ |
| 368 | recipes の evidence 引用。recipes は変えていない |
| 271 | `config.Load` を名指すが、Trigger は `checkDirectorStart` の permission の code |
| 423 | `internal/cli/org.go` の `ralph org status` の JSON。変えていない |

## ほかの文書の食い違い

| 対象 | 結果 |
|------|------|
| `AGENTS.md` の `internal/config/` の行 | `local.go` が `LocalOnlyKeys` と `LocalOrgStateDirAt`(`ResolveOrgStateDir` が使う)を持つ、という記述は `StateDir` の新しい doc と合う。変えていない |
| `README.md` | 148 行の `[org].state_dir` の段落は、条件の文が定数の言葉になった(S4)。「fails to load with an error that names its path」は、org の動詞で同じパスが 2 回出る形でも正しい。org の動詞の `org: load config` の文を書いた所はない(grep)。124 行あたりは別のセッションの変更(#300)で、触っていない。変えていない |
| `/org` skill の 4 つの写し | 足したのは「前提」節の台帳の場所の項の 1 文(S4、b16f4bd0)だけ。変えていない。`check-skill-sync.sh` は 14 skill が揃っている |
| `docs/architecture/`、`docs/quality/`、`docs/specs/`、`docs/recipes/`、`.claude/rules/` | 変えた名前(`resolveConfig`、`statusOrgConfig`、`org_legacy_ledger.go`、`org_start_plan_config_test.go`)を名指す所はない(`git grep`) |
| 他の tech-debt の行 | `org: load config` を含むのは 148 と 421 だけ。「names no file」「naming neither the path」は 421 の取り消し線の中だけ |

## lead へ

- AC6 の箱は付けていない(依頼どおり)。A と case 1 は満たしたが、case 2 は確かめていない。plan の D3 と AC7 は、打てなかった場合を 421 行に「確かめていない」として残せば AC7 を満たすとしているので、AC7 の箱は付けた。理由は plan の Progress checklist の「Docs synced」の下の行と、421 行の (e) に書いた。
- AC7 の箱は付けた。396・397・421 行の書き換えは済み、case 2 は 421 行の (e) に「確かめていない」として残り、verify は `run-static-verify.sh`(中で run-verify の log `docs/evidence/verify-2026-11-02-072714.log` を作る)、test は `run-test.sh` が通っている。PR の CI はまだ見ていない。
- Progress checklist の「Verification artifact created」と「Test artifact created」の箱は、依頼が「plan のほかを変えない」としたので付けていない。report はどちらも commit 済み(`81e469b4`、`12de7ff8`)。
- 192 行は編集していない。依頼は 421 行から指すところまでで、192 行の Debt と Trigger は claude の Bash tool と tmux だけを挙げており、codex の shell tool が zsh で PATH を組み直す形は書かれていない。足すなら、192 行の Trigger の末尾に「2026-11-02: fix/config-path-and-ledger-docs の実物の確かめで、codex の leader の shell tool が同じ skew に当たった」と 1 文。
- PR の Known gaps に回すもの: 421 の (f)(self-review L4)、AC6 の case 2、test report の「ほかに見えたこと」(`~/.codex/config.toml` に足された trust の 2 行、`~/.claude.json` の trust の項目、codex の app-server daemon が消したディレクトリを cwd に残していること)。
- 内容のコミットは 1 回 amend した。最初の `b4fd74ac` の 397 行に、原文にない引用に見える言い回しと「one word」の数え方があり、`1982be83` で直した。未 push で、`run-static-verify.sh` は `1982be83` の上で打ち直した。
- tech-debt の行は plan を `docs/plans/active/2026-11-02-config-path-and-ledger-docs.md` で書いた。`scripts/archive-plan.sh` が /pr で `archive/` に書き換える。`verify.local.sh` の plan 参照の検査は、いまの path で通る。
- diff の大きさ(/pr が walkthrough を決めるための数): `git diff origin/main...HEAD` は 19 ファイル、+1,043/−61(この report と plan の編集を入れる前)。Go のコードは 7 ファイルで +384/−50(本番コード 4 ファイルで +37/−13)。docs を除くと 12 ファイルで +397/−55。walkthrough はここでは書いていない。

## 確かめた結果

- `./scripts/check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。plan と report を書いたあとにもう一度打って、同じ
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`。skill は変えていないので、4 つの写しは前後で同一
- `./scripts/run-static-verify.sh`(内容のコミット `1982be83` の上、plan と report を書く前の tree): rc 0、`All verifiers passed.`。gofmt `ok`、`0 issues.`、check-pipeline-sync、check-template-purity、tech-debt の plan 参照の検査はすべて OK。branch secret scan は `350f22ea..1982be83` で clean。ログは `docs/evidence/verify-2026-11-02-085715.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。`run-test.sh` は打っていない(Go のファイルを変えていない。直前の test report は rc 0)
- tech-debt の構造: `git show HEAD~1:docs/tech-debt/README.md`(内容のコミットの前)と行ごとに比べた。変わった行は 88、148、396、397、411、421 の 6 行だけ。6 行ともに 5 欄に割れ、欄ごとの backtick と `~~` が偶数で、`file:line` の参照は足していない。引用した `docs/plans`・`docs/reports`・`docs/evidence` の path は全部ある
- Edit の anchor: 打つ前に 38 個の `old_string` を python で数え、count 1 を確かめた。1 個(148 行の plan の path)は 4 行に当たったので、左へ広げて 148 行だけに当たる形にした
- plan の digest: `./scripts/plan-visual.sh digest` は `917011181315`(承認時と同じ)。plan の diff は +8/−6 行で、箱 6 つと Progress の 2 行
- `git status --porcelain`(report を書く前): `docs/insights/events/2026-11-02-config-path-and-ledger-docs.jsonl` と plan が `M`。迷子のファイルはない
