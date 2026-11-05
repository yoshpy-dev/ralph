# sync-docs report: org-role-models

- Date: 2026-11-05
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(AC1〜AC13、D1〜D9、S1〜S7)。spec は `docs/specs/2026-11-05-org-role-models.md`
- Branch: `docs/spec-org-role-models`。run 開始時の HEAD は `a3c8c972`、base は main `d631e9f8`
- Pipeline cycle: 1
- 先行 report: self-review(Run 2、Merge: yes)、verify(pass、LOW 2 件の V1・V2 と INFO 5 件の I1〜I5)、test(pass、`run-test.sh` rc 0、足したテスト 2 本は `19aa4144` でコミット済み)
- 制約: Go のファイル、skill、rule、README は変えていない。plan は AC1〜AC13 の箱と Progress checklist だけを変えた。digest は前後とも `82b7b97155f4`

## Summary

verify が渡した 4 件のうち V1、V2、I5 と、recipe の S7 の注を直し、AC の箱を付けた。

- V1: 依頼の 5 行(135、188、190、223、414)に carry-over の注を足した。自分で全 open 行の Trigger を引き直したところ、依頼にない 12 行(88、136、144、180、275、276、319、362、365、371、376、383)も当たっていたので、同じ形で足した。
- V2: 152 行の「ten」を「twelve」にした(2 か所)。
- I5: `AGENTS.md` の `internal/org/` の行に `role_models.go` を足した。
- S7: `docs/recipes/codex-seat-permissions.md` と `templates/base/` の写しに、director の alias の 1 文を足した。
- AC1〜AC13 を `[x]` にした。

表の末尾に新しい行は足していない。コード、help、skill、README の文は変えていないので、pipeline の再実行は要らない。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 18 行を変えた(88、135、136、144、152、180、188、190、223、275、276、319、362、365、371、376、383、414)。17 行は Trigger の末尾に 2026-11-05 の注、Related の末尾に plan を足しただけで、152 行は数の直し。ほかの行は同一 |
| `AGENTS.md` | `internal/org/` の行の、envelope の説明のあとに「the driver and model of a role from `[org.role_models]` (`role_models.go`: `ResolveRoleModel`, and the refusal of a config that still sets `[org.roles]`)」を足した。1 行の中の 1 句で、行は増やしていない |
| `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md` | 「Shell aliases break every codex spawn」の項に 1 文(5 行)を足した。`cmp` で 2 つは同一 |
| `docs/plans/active/2026-11-05-org-role-models.md` | AC1〜AC13 の箱を `[x]` にした。Progress checklist では、Review・Verification・Test の 3 つの artifact の箱を `[x]` にして report の path を付け、「Implementation started」の下に実装後の確認の 1 行を足した。「PR created」は付けていない。numstat は 17 行追加・16 行削除 |
| `docs/reports/sync-docs-2026-11-05-org-role-models.md` | この report |
| `docs/insights/events/2026-11-05-org-role-models.jsonl` | `sync_docs` の行(cycle 1、pass)を `insights-append.sh --slug org-role-models --cycle auto` で足した。既存のファイルに入り、迷子のファイルはできなかった(`git status --short` で確認) |

## tech-debt の carry-over の注

注の形は 2026-11-02 の config-path-and-ledger-docs と同じ(何を変えたか、払わなかった理由、「Carried over to the next change that meets this trigger」)。行数は base `d631e9f8` と `a3c8c972` で `wc -l` を取った。Related には `docs/plans/active/2026-11-05-org-role-models.md` を足した(`/pr` の `archive-plan.sh` が `archive/` に書き換える)。

### 依頼の 5 行

| 行 | 当たったもの | 払わなかった理由 |
|----|--------------|------------------|
| 135(`checkCodexAgmsgWritableRoot`) | `doctor_codex_writable_root.go`(1025 → 991 行)。S4 で `codexSeatPossible` が `codexModelPoolModels` と `codexModelPermittedForRole` に替わり、doc の outcome の番号が 1 つ減った | `pathCovers`、`codexWritableRootSuffix`、読む codex の config は変えていない。plan の Scope は「どの座席が codex で動けるか」(spec FR-7) |
| 188((a) leader の prompt、(b) `EnvelopeSummary`) | (a) `leader.md` の mission 2 と手順 3・5・14。(b) `EnvelopeSummary` は役割ごとの driver と model を先に出し、`checkOrgEnvelope` の doc もそれに合わせた | `RolePromptVars` と `{{SCOPE}}` は変えていないので `--reserve` だけの leader は今も「未指定」を読む。行に `max_orgs` と `max_total_seats` はまだない |
| 190((d) `withMainWorktreeOrgLimits`) | `withMainWorktreeOrgLimits` と `applyMainWorktreeOrgLimits`(`RoleModels`・`DriverPool`・残ったキーの記録も取る、D6) | bare リポジトリの linked worktree を打つ CLI のテストは足していない。読み先のテストは普通のリポジトリの linked worktree(`TestOrgSpawn_RoleModels_ReadFromMainWorktreeRalphToml`)とその下位ディレクトリ(`TestOrgStartPlan_RoleModels_ReadFromMainWorktreeRalphToml`)で動く |
| 223((b) 改名) | 同じ関数。運ぶ値が 6 つになり、上限は 2 つだけ | 2 つの関数の改名はこの機能に要らず、D6 が名前を残した |
| 414(5 つの大きなファイル) | `doctor.go` 1519 → 1613 行、`config.go` 1194 → 1120 行(`role_models.go` 182 行へ新しいコードを置いた) | `retiredEnv` などを動かすのは move-only の変更で、Scope(S4・S5)は doctor の org の検査と 2 つの field の除去。残りの 3 ファイルは変えていない |

### 自分の sweep で足した 12 行

全 open 行の Trigger(取り消し線の外)を python で引き、この diff が変えたファイル名・関数名・test 名と突き合わせた。当たりの Trigger は 1 行ずつ読んで、文面どおりに当たるものだけを残した。

| 行 | 当たったもの | 払わなかった理由 |
|----|--------------|------------------|
| 88((4) の `status.go`) | `status.go` の席の表が `""` を `(CLI default)` で出す(964 → 966 行) | `printStateDirLine` と `omitempty` は変えていない。前の注(11-02)が 1 行の変更でも当たりとして足したので、同じ数え方にした |
| 136(`doctor_shell_alias.go`) | S7 で `--model` の文とコメントが「ralph は常に `--model` を渡す」でなくなった(953 → 968 行) | severity の `switch` と `permissionArgsForDriver` は変えていない |
| 144(`doctor_codex_writable_root_test.go`) | model_pool の gate のテストが `codexSeatPossible` のテストに替わった(+92/−155 行) | (a)が名指す `StoreEqualsProjectDir_Warn` は触っていない。`runDoctorFull` の対象 directory も同じ |
| 180((a) `/org` の command table) | `spawn`・`start`・`director` の行(表の行全体で 2,308 → 2,514、2,182 → 2,267、5,981 → 6,155 文字) | `stop` と `disband` のセル(行の diff に出ない)は変わらず、4 つの写しの表の組み直しは Scope の外 |
| 275(`startDirector` の拒否の順) | `checkStartAgmsgHome` の次に `org.LegacyRolesErr`、続けて `org.ResolveRoleModel` が入った。base では `resolveModelOrWarn` と `[org.roles]` の model 検査だった | (a)(b) のテストは足していない。`checkStartAgmsgHome` と拒否の条件は同じ |
| 276(`StartDirector` の herdr の呼び出し) | `directorAgentArgs` が `""` のとき `--model` を落とす | codex の director の実機の run はしていないので 2 つの前提は同じ |
| 319(`director_start.go` 1201 → 1228 行) | `directorAgentArgs`、`AttachHeadless` の `CheckOrgModelName`、field の doc | 分割は Scope の外。同じ行の他のファイルで変わったのは `org_director.go`(コメント 1 つ、505 行)と `org_director_report.go`(342 → 351 行)だけ |
| 362(席の表と roster) | 席の表の model 列と `manifest.go` の `ModelCLIDefault` の field | 席の行、state 列、`Roster` は変えていない。`spawn_close_failed` の席は今も `spawn_step (active)` の空の列で読める |
| 365(`Send` と leader の雛形のテスト) | `prompts_test.go`(2683 → 2700)の leader のテスト。あわせて `verbs.go`(2277 → 2290)、`spawn.go`(2708 → 2733)、`spawn_test.go`(6030 → 6043)、`verbs_test.go`(12 行、5854 のまま) | 行の最初の欄の数に +300 を足した値には、どのファイルも届かない。move-only の分割は Scope の外 |
| 371(5 つの recipe が引く evidence) | `codex-seat-permissions.md` と `director-launch-cmd.md`(写しも)。この sync-docs が足した 1 文もここに入る | `docs/evidence/` の引用は 1 つも書き換えていない(recipe の diff の `docs/evidence` は hunk の見出しだけ)。数は同じ |
| 376(RC4) | `leader.md` の mission 2 と手順 3・5・14、`/org` skill の split-plan の例の 1 行 | 手順 4 と、例の「予約の外は編集しない」の規則は触っていない |
| 383(手順 14) | 手順 14 の reviewer の spawn し直しが、手順 5 で渡したときだけ `--driver` と `--model` を付ける | 「implementer をもう一度立てる」手順は Scope の外 |

### V2(152 行)

`internal/cli/org_test.go` の `events[len(events)-1]` を python で数え直した。12 か所、12 テストに 1 か所ずつ、どれも直前に `len(events)` の確かめがない。Debt の「ten other tests」と Trigger の「one of the ten others」を「twelve」にした。

### sweep で退けたもの

| 行 | 退けた理由 |
|----|------------|
| 49、138、271、427 | 外部の条件(codex CLI の更新、codex の director の mapping、live run)の Trigger。この diff では起きない |
| 55 | `AGENTS.md` を名指すが、Trigger は旧 shell CLI の `cleanup` コマンドの条件 |
| 98、134、197 | Trigger が取り消し線の中(閉じた行) |
| 158 | `model-routing.md` の「pin 段落」の編集が Trigger。この diff が変えたのは receipts の節と「Codex has no model aliases」の項で、`model: opus` の pin の文は同じ |
| 183、184(`verbs.go`) | 変えたのは `observeStopModelReceipt` の周辺、`codexSpawnInfo`、doc だけ。`Stop` の本体、`Disband`、`stopSeatPane`、補償のコードは変えていない |
| 191 | Trigger の `idempotentRespawn`、`Spawn` の doc、`orgWideLimitsHelp` はどれも変えていない。変わったのは `RetiredRoleInputErr` と `DirectorRoleInputErr` の doc と、`orgWideLimitsHelp` の前に足した別の定数 `orgRoleModelsHelp` |
| 195 | Trigger は「ledger の場所か event の形を変える次の release」。`spawn_started` に印が付いたのは event の形の変更だが、release を切るのはユーザーで、この PR では当たらない。plan の R3 が release ノートに回している |
| 200 | `leader.md` の feature-org 節の編集は (g) の Trigger だが、(g) は f1b0352e で直った |
| 248、249、261 | `director_cmd.go`・`org_director.go` は変えたが、Trigger は merge-order の JSON、`mergeorder.go`、`attach`/`detach` の `RunE`。この diff は触っていない |
| 364 | Trigger は spawn saga の失敗の道(`failStepWithNote` など)。変えたのは argv、receipt、拒否の文で、失敗の道ではない |
| 367、372 | `gate.go` と reviewer・implementer の雛形は変えていない |
| 381 | `reject` と `receipts.go` の形は変えていない。receipt の値(honored が `unknown`)は既存の形 |
| 389 | 名指す 4 ファイルのテストは変えていない |
| 400 | README の `[org].state_dir` の段落は変えていない(変えたのは role-models の段落) |
| 416 | `status.go` の Trigger は「節を足す」。足したのは表の 1 列の表示だけ |
| 425、426 | `leader.md` に節を足していない。`### cross-review` の小節と `reviewer.md` も変えていない |
| 429 | この branch が足した行そのもの |

## ほかの文書の食い違い

| 対象 | 結果 |
|------|------|
| `git grep 'model_pool\|org\.roles\|ModelPool\|プール先頭\|--model は必ず'`(履歴と tech-debt、spec、テストを除く) | 残るのは、残ったキーを扱う skill・recipe・README・Go のコードのコメントだけ。古い説明は見つからなかった |
| `AGENTS.md` | `internal/config/` の行は `role_models.go` を持つ。`internal/org/` の行に I5 を足した。1 行の中の 1 句なので、短さの規則は守った |
| `README.md`、`docs/architecture/repo-map.md`、`docs/quality/` | README の role-models の段落は verify が合うと確認した。repo-map と quality docs に `role_models` や `model_pool` を名指す所はない |
| `.claude/rules/ralph/model-routing.md`(2 つの写し) | 変えていない。receipts の節と「Codex has no model aliases」は verify が合うと確認した |
| `/org`・`/director` skill、`templates/base/` | 変えていない。`check-skill-sync.sh` は 14 skill が揃っている |
| `docs/recipes/codex-seat-permissions.md` | 追加の 1 文は `doctor_shell_alias.go` のコメント(herdr の director は pane の shell を通り、`launch_cmd` の director は絶対パスで exec する)と `director-launch-cmd.md`(起動スクリプトの 4 番目の手順)に合わせた |

## 残った項目の扱い(verify の INFO)

| 項 | 扱い |
|----|------|
| I1(`ralph doctor` は `./ralph.toml` だけを読む) | 直していない。`/org` skill の 4 つの写しに 1 句足すか、tech-debt に行を足すかの二択で、どちらも依頼の範囲を出る。148 行の Debt も doctor を名指さない。lead が選ぶ |
| I2、I3 | 記録済みの逸脱と、直すところがない件。対応なし |
| I4(AC の箱) | 付けた(下の「lead へ」) |

## lead へ

- AC1〜AC13 は `[x]` にした。AC8 の `status`・`stop`・`disband` の半分は、verify の時点では Go のテストがなく、`19aa4144` の `TestOrgCleanupVerbs_LegacyModelKeys_StillWork` が埋めた(test report が `-count=3` で通った)。AC13 は verify が静的な半分、test が残りの半分を満たした。
- tech-debt の注は依頼の 5 行より 12 行多い。当たりが不確かなら、88(`status.go` の 2 行の変更)と 144(テストの機械的な差し替え)は外してよい。
- 12 行は Trigger の文面どおりの当たりを選んだので、PR の本文に「tech-debt の 17 行に carry-over の注を足した」と書ける。
- recipe に足した 1 文は、tech-debt の 371 行(recipe の evidence 引用の行)の注にも書いた。
- `scripts/archive-plan.sh` が /pr で `docs/plans/active/` を `archive/` に書き換える。tech-debt の plan 参照の検査(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ grep)は、いまの path で通る。
- diff の大きさ(/pr が walkthrough を決めるための数): `git diff d631e9f8...HEAD` は 89 ファイル、+6,685/−2,476(この report と plan の編集を入れる前)。Go は 62 ファイルで +5,027/−2,038(本番コード 23 ファイルで +1,098/−718)。docs を除くと 78 ファイルで +5,886/−2,447。walkthrough はここでは書いていない。

## 確かめた結果

- `./scripts/check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- recipe の 2 つの写し: `cmp` で同一
- tech-debt の構造: `git show HEAD:docs/tech-debt/README.md` と行ごとに比べた。変わった行は上の 18 行だけ。18 行とも 5 欄に割れ、欄ごとの backtick と `~~` が偶数で、`file:line` の参照は足していない。152 行以外は、変わった欄が元の文字列の末尾への追記になっている
- tech-debt の plan 参照: `grep -o -E 'docs/plans/(active|archive)/[A-Za-z0-9._-]+'` で取った 111 件がすべて存在する
- Edit の anchor: 打つ前に python で数えた。1 個(376 行の Trigger の短い形)は 2 行に当たったので、左へ広げて 1 行だけに当たる形にした
- plan の digest: `./scripts/plan-visual.sh digest` は `82b7b97155f4`(承認時と同じ)
- `git status --short`: 変更は `AGENTS.md`、plan、recipe の 2 つの写し、tech-debt、insight event の jsonl の 6 つ。迷子のファイルはない
- 打っていないもの: `run-static-verify.sh`、`run-test.sh`(Go、skill、rule は変えていない。直前の verify と test が通っている)

## Insight event

`./scripts/insights-append.sh --slug org-role-models --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を打った。cycle は `cycle-count.json` と同じ 1。
