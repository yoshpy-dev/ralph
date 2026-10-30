# sync-docs report: org-state-dir-local

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-org-state-dir-local.md`(issue #276 の 2 本目 A2。承認の digest `1ea27c42968f` は run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。plan の編集は `## Progress checklist` の 1 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json`)。base は `origin/main` の `c03647f9`、branch は `feat/org-state-dir-local`。run 開始時の HEAD は `4045df27`。文書の変更を入れた commit は `3a6d1b11`(この report の commit は別)
- 先行 report: `docs/reports/self-review-2026-10-30-org-state-dir-local.md`(Merge: yes。L1〜L9 は LOW、L1・L4・L6 は S7 で直済み)、`docs/reports/verify-2026-10-30-org-state-dir-local.md`(pass。LOW の V1・V2、V3 は S8 で fixed、INFO の I1〜I6)、`docs/reports/test-2026-10-30-org-state-dir-local.md`(pass。Test gaps 4 つ)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、`3a6d1b11` の時点): 76 ファイル、+4352/-596。walkthrough を書くかどうかは `/pr` が決める
- 制約: コードは変えていない。`go test`・`go build`・`golangci-lint` は打っていない。打ったのは `check-sync.sh`、`check-skill-sync.sh`、`git diff --check`、`plan-visual.sh digest`、tech-debt の欄数・backtick・plan 参照の確認、`git grep` で、`secret-scan-branch.sh --strict` は push の前に打つ

## Summary

文書の本体(README、`/org` と `/director` skill の 4 つの写し、recipe、template、help)は S5 と S6 が済ませていて、`RALPH_ORG_STATE_DIR` は「もう読まない」の説明にしか出ない。残っていたのは `docs/tech-debt/README.md` と、README の 2 行、`AGENTS.md` の 1 行だった。この run で直したのは次のとおり。

- tech-debt: 148・170・192 行の env の記述を `[org].state_dir` に、188 行の閉じた (d) の注と 198 行の (h) を合わせた。Trigger に当たった 15 行(verify の V1 の 14 行と 192 行)に持ち越しの注。186 行の (c) F-7 を閉じた。新しい 4 行。
- README: `ralph config get` の行に、`ralph.local.toml` にだけ書けるキーを 1 文。`ralph status` の行の見出しに `[--state-dir <dir>]`。
- `AGENTS.md`: `internal/config/` の行に `LocalOnlyKeys` と `LocalOrgStateDir`。
- plan: Progress に sync-docs の行。AC の checkbox と `docs/specs/` は触っていない。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` 148 行(config と state dir を別の規則で探す行) | 項目 1 と V1。Debt の台帳の決め方を「flag、次に主 worktree の root の `ralph.local.toml` の `[org].state_dir`、次に git の段」に直し、Impact と Trigger の `RALPH_ORG_STATE_DIR` を `[org].state_dir` に(history の部分は「until feat/org-state-dir-local」と書いた)。Trigger の末尾に持ち越しの注: `ResolveOrgStateDir` の変更と `withMainWorktreeOrgLimits` の doc に当たったが、開いた部分(config の探し方)には触れていない。D1 で広がった点も書いた(台帳の個人用のファイルは cwd の repository の主 worktree、`config.Load` のは config のファイルのディレクトリで決まる)。Related に plan と self-review の L8 |
| 同 170 行(`LegacyWorktreeStateDir`) | 項目 1 と V1。V-2 の文から env を外し、`:169` と `:50-52` の行番号を名前に(この branch が `statedir.go` を 17 行動かし、番号が古くなったため)。Trigger に持ち越しの注: `guardLegacyOrgStateDir` の insights の分岐が消え、案内は常に `pass --state-dir`。V-1 の順序は決め方を変えるので払わず、V-2 の読めない台帳の注は `--state-dir` を言わないまま |
| 同 192 行(別版の `ralph` で台帳が割れる行) | 項目 1 と V1。run 1 の出来事の `RALPH_ORG_STATE_DIR` は history なので「当時は読んでいた」と添えた。`start <task>` の穴は env を外し、`[org].state_dir` が届く場合(leader が同じ repository で動くとき)を「推測、未確認」として書いた。pane の env を渡す案は、変数がなくなったと書いた。Trigger に持ち越しの注: `newOrgStartCmd` の `RunE` が `resolveOrgLedger` を呼ぶ変更に当たったが、`start <task>` の穴と版の比較は払っていない |
| 同 135 行(`checkCodexAgmsgWritableRoot`) | V1。持ち越しの注。`doctorCodexSandboxEnv` の doc の `runDoctorFull` が `runDoctorStateDir` になっただけで、検査と codex の config の読み方は同じ |
| 同 177 行(`/org` skill の command table、(a)) | V1。持ち越しの注。変わったのは `cleanup` の行(表の行全体が 3,217 から 3,234 文字)で、`stop` と `disband` の cell は同じ。(b) は触れていない |
| 同 186 行(org-limits-reserve の code-shape、(c) F-7) | 項目 3 と V2。(c) を `~~` で取り消し、`Closed in feat/org-state-dir-local (commit daf8e512)` と書いた。内容は、`resolveOrgLedger` が決めることと `guardLegacyOrgStateDir` を 1 つにし、`newOrgRuntime`、spawn、`start <task>`、`runOrgStartPlan` ほか 15 か所が通ること、`newOrgSpawnRuntimeAt` が guard を繰り返さないこと、doctor だけは直接 `org.ResolveOrgStateDir` を呼ぶこと。Impact、Why deferred、Trigger の 3 欄にも同じ印を付け、Related に plan の D6 と verify の V2。(a)(b) の letter は変えていない |
| 同 188 行(org-limits-reserve cycle 2 の wording、(d) ほか) | 項目 4(I1)と V1。閉じた (d) の注の末尾に、help の 2 文、`max_orgs` の template の注釈、`TestOrgSpawnAndStartHelp_OrgWideLimitsSource` が今は `[org].state_dir` で書かれていることを足した。Trigger の末尾に持ち越しの注: `orgWideLimitsHelp` の変更に当たったが、(b)(c)(e) は別の場所の話 |
| 同 198 行(org-feature-worktree の test gaps、(h)(j)) | 項目 4(I1)と V1。(h) は 3 欄とも `~~` で取り消し、`Moot since feat/org-state-dir-local (commit fe58f44b)` と書いた(env を読まないので、pane の環境が台帳を決めない。確かめたかった live check は要らない)。(j) には持ち越しの注: `LedgerMainWorktreeRoot` の `case` の変更に当たったが、`TestSamePath` の case は新しいテストで範囲外。`[org].state_dir` で名指す台帳にも及ぶと添えた(推測、未確認) |
| 同 208 行(org-cross-watch の小さい所見、(b)) | V1。持ち越しの注。`newOrgRuntimeAt` の doc は呼び出し元が台帳を決めて guard する形に変わったが、名指す呼び出し元は 2 つのまま |
| 同 246 行(org-merge-order の小さい所見) | V1。持ち越しの注。`org_merge_order.go` の呼び出しとそのテストを変えたが、letter のどれにも触れていない |
| 同 362 行(800 行を超えるファイル) | V1。持ち越しの注。`prompts_test.go` の leader の template のテスト 2 本と `director_skill_sync_test.go`。`prompts_test.go` は origin/main の 2675 から 2683、`director_skill_sync_test.go` は 1080 から 1085 で、+300 に届かない。行の欄の `spawn.go`・`spawn_test.go`・`prompts_test.go` の数は origin/main の時点ですでに古いので、それも書いた |
| 同 368 行(recipe が `docs/evidence/` を引く行) | V1。持ち越しの注。`codex-seat-permissions.md` の 2 つの写しを直したが、引用は増やしも言い換えもしていない |
| 同 386・390・391 行(#263 の merge cleanup の file size と文) | V1。持ち越しの注。386 は 3 ファイル(`org_merge_test.go` +7、`director_skill_sync_test.go` +5、`cleanup_test.go` +7)。390 は `ledgerMoveStep` と `org_cleanup_test.go`(1142 から 1262 行)、`cleanup.go` は 1086 から 1089 行。391 は断りの文の `ledgerMoveStep` で、(a) の `recordGone` と (b) の `registrationAtRisk` は同じ |
| 同 392 行(`resolvedOrClean` の wrapper) | V1 と、呼び出し元の数。行は `cleanup.go` の呼び出し(`cleanupFeatureWorktreeLocked` と `pathWithin` で 4 呼び出し)と #263 のテスト(`cleanup_test.go` に 10、`merge_run_test.go` に 3)を数えていなかったので足した(`grep` で数え直した)。Why deferred の `permissions.go:209`・`split.go:183`・`feature.go:197` は関数名にした。`feature.go:197` は origin/main の時点ですでに 207 行でずれていた。持ち越しの注も足した |
| 同 395〜398 行(新しい 4 行) | 項目 6。(a) cleanup と merge が案内する retry のコマンドに `--state-dir` が付かない(self-review の L9)。(b) 「絶対パスか `~/`」の判定 3 か所と `~/` の展開 2 か所、それに `config.Load` が local-only 以外のエラーで途中の cfg を返すこと(A1 の L2・L3、今回の L3・L4)。(c) env の拒否の条件が 5 か所に別々に書かれ、判定と文を突き合わせるテストがない(S8)。(d) `ralph doctor --help` の `--strict` の値の名前が `ralph upgrade` になる。(d) は register を引いて、同じ行がないことを確かめた(177 行の (b) は `--scope` の別の flag) |
| `README.md`(`ralph config get` の行) | 項目 5。「Exit 3 は ... 検証に失敗」の文に、`[org].state_dir` を持つ `ralph.toml` も失敗すること、`ralph config get org.state_dir` が `ralph.local.toml` の値を書いたまま(`~/` は展開しない)返すことを足した。verify の probe(exit 3、`~/ledger-home` をそのまま返す)で確かめた事実 |
| `README.md`(`ralph status` の行) | I5。見出しを `ralph status [--org-id <id>] [--state-dir <dir>]` にした。`status` は `--state-dir` を持ち(`internal/cli/status.go`)、doctor と insights の行は同じ flag を見出しに書くようになったため。この PR より前からあった欠けだが、1 語で済む |
| `AGENTS.md`(`internal/config/` の行) | I6。`local.go` が `LocalOnlyKeys` と `LocalOrgStateDir` を持つことを足した。個人用のファイルの仕組みを書く唯一の行で、`[org].state_dir` が加わったため。`internal/gitroot/` の行(config と org で共有)と `internal/org/` の行は変えていない |
| `docs/plans/active/2026-10-30-org-state-dir-local.md` | Progress に「Sync-docs report created」の行。本文の節は触れていない。digest は前後で同じ |
| `docs/reports/sync-docs-2026-10-30-org-state-dir-local.md` | この report |
| `docs/insights/events/2026-10-30-org-state-dir-local.jsonl` | `sync_docs` の行を足した(下の「Insight event」) |

## 依頼の項目ごとの結果

| 項目 | 結果 |
|------|------|
| 1 tech-debt の 148・170・192 行 | 直した(上の表)。順は flag、`[org].state_dir`(`ralph.local.toml`)、git-main-worktree、git-toplevel、cwd。env は history の部分にだけ「当時は読んでいた」と残した |
| 2 V1 の行 | 14 行すべてに持ち越しの注(依頼と V1 は「13 行」と書き、番号は 14 個挙げていた)。行番号は HEAD で読み直したが、1 つも動いていなかった(135、148、170、177、188、198、208、246、362、368、386、390、391、392)。192 行は、Trigger の「`newOrgStartCmd` の `RunE` の次の変更」に当たったので足し、合わせて 15 行。392 は呼び出し元の数も直した |
| 3 V2 | 186 行の (c) F-7 を閉じた(取り消し線と `Closed in`、4 欄とも) |
| 4 I1 | 198 行の (h) は moot にした。188 行の閉じた (d) の注は、help と template と test の今の文に合わせた |
| 5 README の `config get` の行 | 直した。**依頼の前提と違う点**: 依頼は local-only のキーを `[org].agmsg_home` と `[org].state_dir` と書いていたが、`config.LocalOnlyKeys()` は `org.state_dir` だけを返す(`internal/config/local.go`)。`[org].agmsg_home` は `ralph.toml` にも書ける(plan の Non-goals、A1 の決定)ので、README は `[org].state_dir` だけを名指した。`config get --help` の exit 3 の説明は Go(`internal/cli/config.go`)で、`config_test.go` が文を固定しているので直していない。PR の Known gaps に回す |
| 6 tech-debt の新しい行 | (a)〜(d) を 4 行で足した。(d) は既存の行がなかった |
| 7 ほかのずれ | 下の「Surfaces checked」 |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md` の `ralph.local.toml` の節(`:148`)、doctor・insights の行、cleanup の段落 | 変更なし | `[org].state_dir` の置き場所、値の形、エラー、`--state-dir` での回避、env の扱いが branch と合う(verify の AC10) |
| `README.md` の `config get` と `status` の行 | 直した | 上の表 |
| `AGENTS.md:89` | 直した | 上の表。`:90`(`internal/gitroot/`)は変更なし |
| `docs/architecture/repo-map.md:74`、`docs/quality/quality-gates.md` | 変更なし | 台帳の決め方は「`/org` skill の前提を見る」と書くだけで、env に触れない(`git grep`) |
| `.claude/rules/ralph/` | 変更なし | `agent-messaging.md` に state-dir の記述がない。`model-routing.md:118`(insights の receipts は「ralph org の動詞と同じ state-dir の順」)は今も正しい。`:151` の `defaults_sync_test.go` の列挙は、B の範囲なので触れていない(I2。B で `state_dir` と `TestTemplateDriftProbesHoldNoLocalOnlyKey` を足す) |
| `/org` と `/director` skill の 4 つの写し | 変更なし | S5・S6 が済ませている。`check-skill-sync.sh`: 14 本、`check-sync.sh`: DRIFTED 0 |
| recipe 2 本と写し、ほかの recipe | 変更なし | `RALPH_ORG_STATE_DIR` は `worktrees.md` と `codex-seat-permissions.md` の「もう読まない」の 1 文だけ。`director-launch-cmd.md` と `isolated-herdr-server.md` の `--state-dir` と state dir の記述は env に触れない |
| `templates/base/ralph.toml`、leader の雛形、`feature.go` の注釈、help | 変更なし | S5 が済ませている |
| `docs/tech-debt/README.md` の `RALPH_ORG_STATE_DIR` の出現 | 残りは history か「もう読まない」の説明 | 73 行は 2026-08-02 の RESOLVED の HTML コメント(残してよい履歴)。148・170・177・186・188・192・198・246・362・368・386・397 行は、当時の事実か、変わったことの説明 |
| `docs/specs/` | 直していない | 依頼の「触らないもの」と plan の Non-goals。下の「直さなかったもの」 |
| `docs/tech-debt/README.md` の欄数・backtick・plan 参照 | 通った | 5 欄の行が 295 から 299 に(新しい 4 行)、4 欄 1 行と 6 欄 1 行は前のまま。変えた行 20 行はすべて 5 欄で、各欄の backtick と `~~` が偶数。`file:line` の参照は変えた行に 0 件。`docs/plans/(active\|archive)/` の参照は missing 0 件(新しい参照は `docs/plans/active/2026-10-30-org-state-dir-local.md` で、`/pr` の `archive-plan.sh` が書き換える) |

## 直さなかったもの

| 項目 | 理由 |
|------|------|
| `ralph config get --help` の exit 3 の説明が local-only のキーを名指さない | Go の文字列(`internal/cli/config.go`)で、`internal/cli/config_test.go` の 230 行付近と 363 行付近が文を固定している。直すとパイプラインの最初からやり直し。README は直した。PR の Known gaps |
| `docs/specs/2026-10-07-org-multi-org-director.md` の `:88`・`:135`・`:144` と、`docs/specs/2026-08-01-org-runtime.md` の FR-2 の注 | 依頼の「触らないもの」。計画の Non-goals は過去の箇条の書き換えを禁じ、「新しい箇条を足すのは /sync-docs の判断」と書くが、依頼が docs/specs を外したので足していない。足すなら A1 の I3 と同じ形(FR-6 と FR-9 に 2026-10-30 の日付つきの箇条)。要るなら言ってほしい |
| `.claude/rules/ralph/model-routing.md:151` | B の範囲(依頼)。I2 |
| plan の AC の checkbox | 依頼になく、AC11 は `run-verify.sh` の既定モードを打っていない(verify は静的、test は `run-test.sh`)。lead が決める |
| self-review の L2(env の拒否の文の頭が「cannot resolve the org ledger」)、L5(git の呼び出しが 2 回から 4 回に増える)、L7(source 名 `config`) | コードとテストの変更で、`/sync-docs` は文書だけを直す。LOW で、self-review が残すと判断済み。register にも足していない(依頼の (a)〜(d) にない)。PR の Known gaps |
| test report の Test gaps 1〜4 | 同上。1(`FeatureRepoRoot` の bare repository のエラー分岐)、2(`filepath.Abs` の失敗)、3(実物の HOME での手動確認)、4(`watch --detach` の子の注意の 1 行)。PR の Known gaps。register に足すなら A1 の 2g と同じ形で 1 行にできる |

## Release notes

(PR 本文の「リリースノート」の材料。`ralph.toml` は seed なので、`ralph upgrade` は既存の project の `ralph.toml` を書き換えない。)

1. ralph は `RALPH_ORG_STATE_DIR` を読まなくなった。台帳(`.harness/state/org`)の場所は `--state-dir` か、`ralph.local.toml` の `[org].state_dir`(絶対パスか `~/` で始まる形)で決まる。env で変えていた人は、値を `[org].state_dir` に移すか `--state-dir` を渡し、env を消す。
2. env が空でない値で残っていると、台帳を決めるコマンドは stderr に注意を 1 行出して続ける。ただし `--state-dir` も `[org].state_dir` も台帳を名指さず、env が既定の台帳と違う場所を指すときは、台帳を書き換える動詞(`start`、`spawn`、`send` の TASK、`stop`、`disband`、`cleanup` など)は何も書かずに終了コード 1 で断られる。読むだけの動詞は注意を出して続ける。`ralph doctor` は env が残っている間 warn を出す。黙って別の台帳に移ると、元の org が見えなくなり、上限と予約が 2 つの台帳で別々に数えられるので、更新のあとに `ralph doctor` を打つ。
3. `ralph insights` と `ralph doctor` に `--state-dir` ができた。insights は `--receipts` があればそちらが勝つ。
4. `ralph.toml` に `[org].state_dir` を書くと読み込みがエラーになる(`ralph config get` は exit 3)。個人の機械の値なので、`ralph.local.toml` にだけ書ける。`templates/base/ralph.toml` は注釈でだけ触れるので、既存の project は落ちない。
5. 個人用のファイルが読めない、または `state_dir` が相対パスのとき、`--state-dir` のない org の動詞、`ralph status`、`ralph insights` は終了コード 1 でファイルのパスを名指す。`ralph doctor` は fail の行を出して残りの検査を続ける。相対の値は `--state-dir` を渡せば避けられる。
6. `--state-dir` 付きで起こした見張りの子が env を受け継ぐと、watch.log に注意が 1 行出る(害はない)。
7. 戻すときはこの PR を revert する。台帳の中身と形式は変えていない。ただし revert した ralph は `[org].state_dir` を読まない(推測、未確認)ので、移行した人は env を設定し直すか `--state-dir` を渡さないと既定の台帳に戻る。
8. #276 の残りは B(標準フローの値とスクリプト)。この PR は `Refs #276`。

自動の changelog(`.goreleaser.yml` は `^docs:`・`^test:`・`^chore:` を除く)に載る commit は次のとおり。

| commit | 題 |
|--------|----|
| `e2e7b7f1` | `feat: add the local-only [org].state_dir key to the personal config` |
| `daf8e512` | `feat: resolve the org ledger from the personal [org].state_dir` |
| `c78f6903` | `fix: let escalate reach the human when the org ledger cannot be resolved` |
| `4505223d` | `feat: add --state-dir to ralph insights and ralph doctor` |
| `fe58f44b` | `feat: stop reading RALPH_ORG_STATE_DIR and refuse a split ledger` |
| `3700470e` | `Merge remote-tracking branch 'origin/main' into feat/org-state-dir-local`(prefix がなく、filter に掛からない) |
| `c76c2ef2` | `fix: resolve the ledger of ralph org cleanup by --state-dir and [org].state_dir` |
| `187eaf8d` | `fix: return defaults when ralph.toml sets a local-only key and keep the config-tier ledger absolute` |
| `cb33a636` | `fix: say when RALPH_ORG_STATE_DIR still refuses the org verbs` |

`fe58f44b` は題が `feat:` でも、読んでいた変数を読まなくする破壊的な変更なので、changelog の行だけでは足りない。release のノートの 1 と 2 を先に読ませる。

## Notes for reviewers(PR 本文に入れる点)

- Known gaps(コードかテストで、この PR では直していない): self-review の L2・L5・L7、test report の Test gaps 1〜4、`ralph config get --help` の exit 3 の説明が local-only のキーを名指さないこと(README は直した)、verify の I2(`model-routing.md:151`、B)。tech-debt に足した 4 行(上の表)は、これとは別に register にある。
- 実行していない確かめ: `ralph org start`・`spawn` の拒否を実機で打つこと(見張りの子を残さないため、コードとテストで確かめた)、本物の herdr と agmsg での通し、実物の HOME の `~/` 展開。tech-debt の新しい行と持ち越しの注のうち、「推測、未確認」と書いた 3 点(192 行の leader が同じ repository で動くとき、198 行の (j) が `[org].state_dir` にも及ぶこと、新しい 395 行の retry が別の台帳を読むこと)は、コードを読んだ推論で、打っていない。
- run-verify の既定モード(`./scripts/run-verify.sh`)は、この run も打っていない。verify が静的、test が `run-test.sh` を打った。

## Left open

- tech-debt: 持ち越しの注を足した行(開いたまま)、新しい 4 行。
- 上の「直さなかったもの」。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | `PASS: all files in sync.`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step`。skill は変えていない |
| `git diff --check` | rc 0、出力なし |
| tech-debt の欄数と backtick | 変えた 20 行(新しい 4 行を含む)がすべて 5 欄、各欄の backtick と `~~` が偶数、`file:line` の参照 0 件(scratchpad の `tdcheck.py` が `git show HEAD~1:` と突き合わせた) |
| tech-debt の plan 参照 | `grep -o -E 'docs/plans/(active\|archive)/[A-Za-z0-9._-]+'` で missing 0 件 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-org-state-dir-local.md` | `1ea27c42968f`(`- Approved:` の行と同じ。Progress の編集の前後で同じ) |
| 変えたファイルの U+FFFD | 0 件(`grep -a` でバイト `EF BF BD` を数えた) |
| 打っていない | `go test`、`go build`、`golangci-lint`、`run-static-verify.sh`、`run-test.sh`、`run-verify.sh`。この run が変えたのは文書、plan の Progress の 1 行、この report、insight event で、コードも help も変わらない。README と `AGENTS.md` を固定するテストはない(`git grep` で確かめた) |

## Insight event

`docs/insights/events/2026-10-30-org-state-dir-local.jsonl` の末尾に `sync_docs` の 1 行を足した(`./scripts/insights-append.sh --slug org-state-dir-local --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`、rc 0、ts 2026-10-30T04:51:40Z、cycle 1、findings は 0)。file は 5 行になった。`git status --short` は、既存の jsonl の ` M` とこの report の `??` だけで、`<date>-<date>-<slug>.jsonl` のような余分な file はできていない。

## cycle 2

- Date: 2026-10-30
- Pipeline cycle: 2(`cycle-count.json`)。run 開始時の HEAD は `23af7887`。回し直しの原因は cross-review cycle 1 の ACTION_REQUIRED #1(`ralph.toml` の `[ORG]` / `STATE_DIR` が local-only の検査をすり抜ける)と、その直しの S9(`6415aabd`、`internal/config/local.go` と `local_test.go`、`checkLocalOnlyKeys` を go-toml の decoder と同じく大文字小文字を区別せずに引く)
- 先行 report: self-review cycle 2(Merge: yes。S9 の差分に所見なし、LOW は C2-L1 の 1 件)、verify cycle 2(pass。S9 で文書のずれなし)、test cycle 2(pass。S9 で新しい穴なし)
- 制約: 依頼で plan は編集していない(Progress への記録は orchestrator)。コードも触っていない。`go test` などは打っていない。AC の checkbox も触っていない
- origin/main は進んでいる: `1e266f96`(#289 `feat/regate-ci-checks`)が入り、`README.md`、`AGENTS.md`、`docs/tech-debt/README.md`(表の末尾に 3 行、287 行付近に 1 行の変更)、`internal/config/` に触れている。この branch の merge-base は `c03647f9` のまま。取り込むときは tech-debt の表の末尾で衝突する(両方の行を残す)

### Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` 399 行(新しい行) | 依頼 1。self-review の C2-L1 に従い、L2 と L5 を 1 行にまとめた。(a) L2: `checkRetiredOrgStateDirEnv` の拒否が `errOrgLedgerUnresolved` で包まれ、頭が「cannot resolve the org ledger:」になるのに、直後で「the ledger ralph finds by default is X」と言う。`guardLegacyOrgStateDir` は「org: refusing to change the org ledger:」で始まり、印も付かない。全文を固定するテスト(`TestOrgCleanup_RefusedWhileTheRetiredStateDirEnvNamesAnotherLedger` と `org_retired_state_dir_env_test.go` の start・spawn・send)と、印を見る escalate の not-recorded の振り分けが、直す範囲を決める。(b) L5: 既定の段で `gitroot.MainWorktree` を `config.LocalPath` の中と main worktree の段で 2 回呼び、git が 2 回から 4 回になる。`LocalOrgStateDir` が返す path はテストしか読まず、拒否の文は `config.LocalPath` を計算し直す(さらに 2 回)。Trigger は (a) が `resolveOrgLedger`・拒否・escalate の not-recorded の経路の次の変更、(b) が `ResolveOrgStateDir` か `LocalOrgStateDir` の次の変更。(a) は変数を `retiredEnvs` から外すと消える。L7(source 名 `config`)は計画の決定なので足していない |
| `docs/tech-debt/README.md` 400 行(新しい行、依頼にない) | `internal/config/local_test.go` が 800 行を超えた(`c03647f9` で 570、`23af7887` で 981)。この branch の Go ファイルで、800 行を越えたのはこの 1 つだけ(`git diff --name-only c03647f9...HEAD` の Go ファイルを、`c03647f9` と HEAD の `wc -l` で突き合わせた。ほかの 800 行超は前から超えていた)。cycle 1 の sync-docs は大きさを見ていなかった。#263 の 386・390 行と同じ形で Trigger は「次の変更で `LocalOrgStateDir` と `LocalOnlyKeys` のテストを先に別ファイルへ出す」。要らなければ lead が外してよい |
| `docs/reports/sync-docs-2026-10-30-org-state-dir-local.md` | この節 |
| `docs/insights/events/2026-10-30-org-state-dir-local.jsonl` | `sync_docs` の行(`--cycle auto`、cycle 2) |

### S9 の文書の確認

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md` の `:135`(`config get`)と `:148`(`[org].state_dir` の段落) | 変更なし | 「`ralph.toml` に `[org].state_dir` があると読み込みが失敗する」は、decoder が同じキーと読むどの綴りにも成り立つ。大文字小文字に触れる文はない |
| `AGENTS.md:89` | 変更なし | `LocalOnlyKeys` を「その file だけが書けるキー」と書くだけで、照合の細部は書いていない |
| `.claude/rules/ralph/`、`/org` と `/director` skill、recipe、`templates/base/ralph.toml`、`config get --help` | 変更なし | `git grep` で、local-only のキーの照合が大文字小文字を区別すると書いた文も、区別しないと書いた文もない。template の注釈(「only ralph.local.toml may set it, and this file fails to load with it」)も合う |
| tech-debt の Trigger(`local.go`、`local_test.go`、`checkLocalOnlyKeys`、`tomlDocHasKey`、`LocalOnlyKeys` と 2 本のテスト) | 当たった行なし | `local.go` を名指すのは 269 行(RESOLVED の HTML コメント)と 396 行。396 行の (a) が挙げる 4 つの関数は S9 で変わらず、(b) の「`config.Load` のエラーの道」は、S9 がどの文書が `checkLocalOnlyKeys` のエラーになるかを変えただけで、エラーのあとの扱い(`Default()` を返す)を変えていないので、verify の判断どおり当たっていない。294 行は `isolateLocalGitEnv` を名指すが、S9 は触れていない |
| 欄数・backtick・plan 参照 | 通った | 変えた行は新しい 2 行だけ。ともに 5 欄で、backtick と `~~` が偶数、`file:line` の参照 0 件。`docs/plans/(active\|archive)/` の参照は missing 0 件 |

**判断: 大文字小文字を文書に書かない。** 書かない理由は 3 つある。
- ralph.toml のどのキーも、go-toml の decoder は大文字小文字を区別せずに struct の field に当てる。`[org].state_dir` のときだけそう書くと、ほかのキーは区別するように読める。
- 契約は「`ralph.toml` に `[org].state_dir` を置くと断られる」で、ralph が `[org].state_dir` として読む綴りはすべてその「置く」に入る。綴りの数え上げは契約を狭めずに長くするだけ。
- 断られた人が見るエラーは正規の綴り `[org].state_dir` とファイルのパス、書く先の `ralph.local.toml` を名指す(C2-N1)。`STATE_DIR` と書いた人は、それでも直し方にたどり着く。

文書に足すと決めるとしたら、エラーの文に文書の綴りを併記する Go の変更になる。それは C2-N1 が「迷う時間は短い」として所見にしなかったもので、ここでは扱わない。

### PR の Known gaps(最新の一覧)

register に入っていないもの。

- source 名 `config`(self-review の L7)。計画の決定で、`--config` のファイルと取り違えうる。直すなら `statedir.go` と `org_legacy_ledger.go` の比較も合わせる。
- test report の Test gaps 1〜4(`FeatureRepoRoot` の bare repository のエラー分岐、`filepath.Abs` の失敗、実物の HOME での手動確認、`watch --detach` の子の注意の 1 行)。
- `ralph config get --help` の exit 3 の説明が local-only のキーを名指さないこと(Go、`config_test.go` が固定。README は直した)。
- verify の I2(`model-routing.md:151`、B の範囲)と I3(`docs/specs/` の日付つきの箇条、依頼で外した)。
- 注(所見にしないもの): self-review cycle 2 の C2-N1(エラーは正規の綴りを名指す)、C2-N2(「一致したキーをすべて辿る」を固定する行は map の順しだいで、決定的には落とせない)、C2-N3(`tomlDocHasKey` の doc は go-toml の版を書かない)。

register に入れたもの(Known gaps ではなく tech-debt から辿れる): 395〜398 行(cycle 1)、399 行(L2・L5)、400 行(`local_test.go` の大きさ)。

### Verification(cycle 2)

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | `PASS: all files in sync.`(DRIFTED 0) |
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `git diff --check` | rc 0、出力なし |
| tech-debt | `git show HEAD:` と突き合わせて、変わった既存の行は 0、増えた行は 2(399・400)。5 欄、backtick と `~~` が偶数、`file:line` の参照 0 件 |
| tech-debt の plan 参照 | missing 0 件 |
| `./scripts/plan-visual.sh digest ...` | `1ea27c42968f`(この run は plan を編集していない) |
| 変えたファイルの U+FFFD | `docs/tech-debt/README.md` に 0 件 |
| 打っていない | `go test`、`go build`、`golangci-lint`、`run-static-verify.sh`、`run-test.sh`、`run-verify.sh`。この run が変えたのは文書、この report、insight event だけ |

### Insight event(cycle 2)

`./scripts/insights-append.sh --slug org-state-dir-local --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`、rc 0、ts 2026-10-30T06:13:09Z、cycle 2、findings は 0。jsonl は 10 行になり、`git status --short` は既存の jsonl と tech-debt の ` M` だけで、余分な file はできていない。
