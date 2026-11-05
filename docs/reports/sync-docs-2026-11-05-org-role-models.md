# sync-docs report: org-role-models

- Date: 2026-11-05
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(AC1〜AC13、D1〜D9、S1〜S7)。spec は `docs/specs/2026-11-05-org-role-models.md`
- Branch: `docs/spec-org-role-models`。run 開始時の HEAD は `a3c8c972`、base は main `d631e9f8`
- Pipeline cycle: 1(この節)と 2(末尾の「Cycle 2」)。cycle 1 の記録は書いたときのまま残した。ただし下の「sweep で退けたもの」の 183・191・381 と、表にない 179・361 は、cycle 2 で当たりに改めた(「Cycle 2」の「cycle 1 の判断の訂正」)
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

## Cycle 2

- Date: 2026-11-06
- 開始時の HEAD: `ccd88ec5`(cycle 2 の self-review・verify・test の report と、test が足したテストの `fd573a50` のあと)。base は main `d631e9f8`
- Pipeline cycle: 2(`cycle-count.json` の `cycle` は 2。既定の上限 2 の最後の run)
- 先行 report: self-review Run 3(Merge: yes、LOW 5 件)、verify cycle 2(pass、LOW 2 件の V2-1・V2-2、INFO 5 件の I2-1〜I2-5)、test cycle 2(`run-test.sh` rc 0。足した 2 本は `fd573a50` でコミット済み)
- 制約: Go のファイルと help の文字列は読んだだけ。plan は `## Progress checklist` に 1 行足しただけで、digest は前後とも `ba806f4c1584`

### Summary

verify cycle 2 が渡した 6 件を扱った。コードも help も変えていないので、pipeline の再実行は要らない。

- V2-1: tech-debt 430 行に (f) として足した。Trigger は `org_start_plan_config_test.go` の次の編集
- V2-2: 381・395・191・303・396 行に carry-over の注を足した。判断を任された 179・183・361 行にも足した(理由は下)。430 行の Related に Run 3 と verify cycle 2 を足した(I2-5)
- I2-3: `/org` skill の `(CLI default)` の一覧を直し、4 つの写しをそろえた
- I2-2: spec の FR-5 に、冪等な打ち直しは 2 つの検査の外にある、という段落を足した
- I2-1・I2-4: plan の Progress checklist に 1 行足した。本文は直していない
- 6 番: resolver が `driver_pool` の外の driver を断ると書く文書は、見つからなかった(Go の help も同じ)

### Changes made(cycle 2)

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 9 行を変えた(179、183、191、303、361、381、395、396、430)。8 行は Trigger の末尾に 2026-11-05 の注、Related の末尾に plan を足しただけ。430 行は見出し・Debt・Impact・Why deferred・Trigger・Related の 6 か所 |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 「役割ごとのモデル」節の 1 段落(3 行を 4 行に。display 幅は 59〜72 で既存の行と同じ) |
| `docs/specs/2026-11-05-org-role-models.md` | FR-5 の末尾(FR-6 の前)に 1 段落 |
| `docs/plans/active/2026-11-05-org-role-models.md` | `## Progress checklist` の「Implementation started」の下に 1 行(numstat は +1/−0) |
| `docs/reports/sync-docs-2026-11-05-org-role-models.md` | この節。cycle 1 の節は、`Pipeline cycle` の行に cycle 2 への案内を足した 1 行だけ変えた |
| `docs/insights/events/2026-11-05-org-role-models.jsonl` | `sync_docs` の行(cycle 2)を足す |

### V2-1(430 行の (f))

テストの名前は `TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir`(`internal/cli/org_start_plan_config_test.go`)。コメントは「refused by the CLI before StartFeature since plan 2026-11-05-org-role-models」で、`git grep -F` の当たりはこの 1 か所だけ(行番号は引かず、テスト名で書いた)。S8 のあと、文字の規則に合わない `--model` を断るのは `StartFeature` の事前検査(`spawnPrecheckErr` から `ValidateSpawnEnvelope`)。(f) の Trigger は verify の提案どおり「そのファイルの次の編集」にして、直し方の文言も書いた。

430 行の他の列も合わせた。

- 見出しは「self-review and verify」、「(L4, L6, N2, N3, V2-1; N1 resolved)」にした
- Impact は「(f) は テストのコメントに書いた理由が古いだけで、テストは正しい」と足した
- Why deferred の「Each is cosmetic」は、verify I2-5 が言うとおり (e) と合わなかった。「Each but (e) is cosmetic」にして、(e) は Impact を見るよう書き、(f) は上限の最後の run で見つかり、Go のコメントを直すと 3 回目になる、と足した
- Related は「(Run 2 and Run 3)」にし、verify の cycle 2(V2-1、I2-5)を足した

### V2-2(carry-over の注)

注の形は cycle 1 と同じ。日付は cycle 1 の注と同じ「2026-11-05」にした。ブランチのコミットはすべて 2026-11-05 の作業で(S8 の `b4cfa641` は 23:58)、同じ文字列で `grep` するとこのブランチの注が全部出る。行番号は `ccd88ec5` の README のもの。

| 行 | 当たったもの | 払わなかった理由 |
|----|--------------|------------------|
| 381(RC10、`reject` と Receipts 節) | `ralph insights` の Receipts 節(S8 の N1: `printInsightsHuman` が指示したモデルが空の座席を `(CLI default)` と出す) | receipt の契約の変更は Scope の外。`reject` と `receipts.go` は同じ。新しい座席の `driver_pool` と文字の拒否は `reject` を通って `honored=false` の receipt を残すが、base も `ValidateSpawnEnvelope` で同じ形だった(driver の突き合わせは同じで、`model_pool` の照合が文字の規則に替わった)。`ResolveRoleModel` の拒否と読み込みの失敗は何も残さない。注にこの比較も書いた |
| 395(`resolvedOrClean`) | 呼び出し元の `feature.go`(S8: `StartFeatureParams` の field `DriverFromRoleEntry` と `startFeatureLeaderParams`)と `feature_test.go`(S2: 「プールの外のモデル」の case が「1 語でないモデル」と `[org.roles]` の case に替わった) | 呼び出しの改名は Scope の外。`resolvedOrClean` と `statedir.go` は変えていない |
| 191(`Spawn` の doc) | S2(`bf1f4727`)が doc の手順 1 の廃止キーの句、手順 2、手順 3、dry-run の段落の例を書き換えた | (b)(c)(e) は `idempotentRespawn` の doc、quality-gates の行、report の id で、Scope の外。`idempotentRespawn` と `orgWideLimitsHelp` は同じ(足したのは別の定数 `orgRoleModelsHelp`) |
| 303(`watch_test.go`) | S5(`14c77c43`)が `watchTestConfig` の `ModelPool` と `Roles` の 2 行を消した | `newTestWatchRun` は今も `os.MkdirTemp` で、呼び出しは 28 のまま(`grep -c 'newTestWatchRun('` は定義を入れて 29)。引数に `testing.T` を足すのは Scope の外 |
| 396(`director_cmd_test.go`) | S2 が `TestDirectorLaunchScript_RefusesValuesItCannotWrite` の「no model」の case を「a NUL byte in the model」に替えた(空のモデルが起動スクリプトの正しい値になったため) | 時間条件の 2 つのテストは触っていない。`director_wake_test.go` は変わっていない |
| 179(code-shape)、183(a)、361(spawn lock) | `Stop` の中のコメント 1 行(`token = receipt.Honored`)と `StopResult` の doc の列挙(S2)。コードは変わっていない | 179 は `result.Err = err` の包み直しと own-pane の読み手、183 (a) は読めない台帳などの case、361 は spawn lock を `Stop` が取らないこと。どれも Scope の外 |

179・183・361 は、verify が判断に任せた 3 行。Trigger の文面が「`Stop` の次の変更」なので、コメントだけの変更も文面どおりには当たる。cycle 1 は 88 行を 1 行の変更でも当たりとして足したので、同じ数え方にした。コードを変えない編集は当たりに数えない、と lead が決めるなら、この 3 行の注(Trigger の末尾の 1 文と、Related の plan)は外してよい。

### cycle 1 の判断の訂正

| 行 | cycle 1 の記録 | 訂正 |
|----|----------------|------|
| 183 | 「`Stop` の本体は変えていない」 | `Stop` の中のコメント 1 行と `StopResult` の doc は変わっていた(コードは変わっていない) |
| 191 | 「`Spawn` の doc … はどれも変えていない」 | `Spawn` の doc の手順 1〜3 は S2 で書き換わっていた。cycle 1 は `RetiredRoleInputErr` と `DirectorRoleInputErr` の doc だけを数えていた |
| 381 | 「`reject` と `receipts.go` の形は変えていない」 | その時点では合っていた。S8 が Receipts 節を変えたので、cycle 2 で当たりになった |
| 179、361 | 表になし | 上の `Stop` のコメント |

### 第 2 の sweep(S8 が触ったファイルと関数)

b4cfa641 と fd573a50 が触った名前(`envelope.go`、`ValidateSpawnEnvelope`、`feature.go`、`insights.go`、`StartFeature`、`checkDirectorStart`、`spawnPrecheckErr`、`idempotentRespawn`、`Stop`、`verbs.go` など 29 語)を、全 open 行の Trigger(取り消し線の外)に python で当てた。当たりは 39 行。12 行は cycle 1 の注があり、8 行は上の表で、残る 19 行を 1 行ずつ読んで退けた。

| 行 | 退けた理由 |
|----|------------|
| 49、138、164 | Trigger は外部の条件(codex CLI の更新など)か、`ralph insights` の読み違いの報告。この diff では起きない |
| 133、290、370 | 取り消し線の中(閉じた行) |
| 173 | Trigger は `LegacyWorktreeStateDir` と `guardLegacyOrgStateDir` |
| 184 | Trigger の `CloseDeferredSelfPane`、`reactivateSeat`、`reserveAgain` などは変わっていない |
| 192、288 | `idempotentRespawnDecision` と `Spawn` の冪等の枝は変わっていない。S8 が変えたのは `SpawnRequest` を組む行(`spawnPrecheckErr` の 1 行を入れて 3 か所)だけ |
| 199、203、204、390 | `StartFeature` を名指すが、Trigger は二重 start の報告、特定のテストの編集、`featureCanonicalRef`、`runWorktreeScript` と `StartFeature` の検査。S8 は `StartFeatureParams` に field を 1 つ足しただけ |
| 200、262 | Trigger は leader.md の feature-org 節((g) は直済み)と `/director` の受信箱の節。S8 は触っていない |
| 269、274 | `checkDirectorStart` を名指すが、Trigger は path の解決と権限のコード。S8 が足した `driverPoolErr` と `CheckOrgModelName` の呼び出しは、その手前にある |
| 316 | Trigger は `wake_cmd` の `{target}` |

12 行の注があった行のうち 152 は、S8 が `org_test.go` を 36 行変えたので数え直した。`events[len(events)-1]` は今も 12 か所で、S8 の diff に当たる行はない。

### I2-3(`/org` skill)

「役割ごとのモデル」節の段落は、`(CLI default)` と出すものを「`spawn` と `start` の出力行、`ralph org report`、`director status`」と挙げていた。コードで `org.ModelOrCLIDefault` を通る所を `grep` で出し直して(`org.go` の `printStatusTable`、`status.go`、`insights.go`、`doctor.go` と `envelope_summary.go`)、次の一覧にした。

> `spawn` と `start` の出力行、`ralph org status` と `ralph status` の座席の表、`ralph org report`、`director status`、`ralph insights` の Receipts 節、`ralph doctor` の envelope の行

依頼は 3 つだったが、`ralph doctor` の envelope の行(`role_models: director=claude/(CLI default)`)も同じ表示なので足した。足さないと、また一覧が足りなくなる。編集は `.claude/skills/org/SKILL.md` に Edit で行い、`./scripts/sync-skills.sh` で `.agents/` を作り直し、`templates/base/` の 2 つには `cp` した(sync-skills はテンプレートの写しを作らない)。4 つの写しで、旧文は 0 件、新文は 1 件ずつ。この文を読むテストは見つからず、skill を読む `internal/org` と `internal/cli` のテストを `-run 'Skill|SendSync|ReportNames|SendDefaults|Sync'` と `-run 'RetiredStateDir|Docs|Skill'` で打って、どちらも ok。

### I2-2(spec の FR-5)

FR-5 の末尾に 1 段落足した。書いたこと: 冪等な打ち直しは `--model` の文字と `driver_pool` の突き合わせの外にあり、既に立っている座席には、`driver_pool` を狭めても、1 語でない `--model` でも、既存の座席を返し、台帳にも receipt にも足さず、herdr も呼ばない。`start --plan` の leader の事前検査も、立っている座席には冪等の判断だけをする。検査が効くのは新しい座席と director start。受け入れ条件の `--model` の拒否も同じ(2026-11-05 の cross-review cycle 1 の指摘で変えた、という日付の注)。FR-3 の 2 つの断りは設定だけで決まり、冪等の判断より先にあるので、`[org.role_models]` を最初の spawn のあとに書き換えた打ち直しは断られうる、と self-review N3 を指して書いた。根拠は verify cycle 2 の Probe 7(binary で 3 通りとも rc 0、manifest は 2 行のまま、receipt は 0 件)と、CLI のテスト 3 本。自分では binary を打っていない。natural-japanese の lint は、足した行に指摘なし。

### I2-1・I2-4(plan)

`## Progress checklist` の「Implementation started」の下に 1 行足した。D4 の関数の形が `(ResolvedRoleModel, error)` になったこと、D4 の「設定が変わっても今どおり既存の座席を返す」は `driver_pool` と文字の検査についてだけ正しく、N3 の 2 つの断りが例外であること(tech-debt 430 行の (e))、Implementation outline の 2 番(S2)の一文が S8 より前の記録で、7 番(S8)が移したことを書いている。本文は直していない。`./scripts/plan-visual.sh digest` は `ba806f4c1584` のまま、`git diff --numstat` は +1/−0。

### 6 番(resolver が `driver_pool` を断るという古い説明)

| 対象 | 結果 |
|------|------|
| `/org` skill(「`driver_pool` にない driver」の項、「上書きの規則」) | 合っている。エントリ由来のとき、役割とエントリと `driver_pool` を名指して断ると書く。新しい座席の話で、打ち直しには触れていない(言い切ってもいない) |
| `/director` skill、`internal/org/prompts/leader.md`・`director.md` | `driver_pool` の検査の場所を書いていない |
| `model-routing.md`(2 つの写し)、README | 検査の場所を書いていない |
| `templates/base/ralph.toml` のコメント | 「An entry whose driver is not in driver_pool still loads, but ralph refuses to launch that role with it」。S8 のあとも合っている |
| spawn・start の help(`orgRoleModelsHelp`、`orgDriverFlagUsage`、`orgModelFlagUsage`)と director start の Long(`internal/cli/org.go`、`org_director_headless.go`) | `driver_pool` の拒否がどこで走るかを書かない。書いてあることは合っている |
| `git grep` で `ResolveRoleModel` を名指す、internal の外の文書 | `AGENTS.md` の `internal/org/` の行(`role_models.go` の説明)と tech-debt の行だけ。どれも `driver_pool` を resolver の検査とは書かない |

Go の help の文言のずれは見つからなかった。ずれたコメントは V2-1 の 1 か所だけで、tech-debt に回した。

## 確かめた結果(cycle 2)

- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- `./scripts/check-sync.sh`: PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `/org` skill の 4 つの写し: `.claude` の 2 つと `.agents` の 2 つが、それぞれ `cmp` で同一
- tech-debt の構造: `git show HEAD:docs/tech-debt/README.md` と行ごとに比べた。変わった行は 179、183、191、303、361、381、395、396、430 の 9 行だけ。どれも 5 欄のまま、欄ごとの backtick と `~~` が偶数で、`file:line` の参照は増えていない。8 行は Trigger と Related への追記、430 行は 6 か所の編集。Edit の anchor は打つ前に python で数え、1 個(395 行の Related)が 5 か所に当たったので、左へ広げて 1 か所にした
- tech-debt の plan 参照: `verify.local.sh` の `check_tech_debt_plan_refs` と同じ grep と sed で、存在しない path は 0 件
- carry-over の注の数: 「2026-11-05: docs/spec-org-role-models」を含む行は 26 行(`ccd88ec5` の 18 行と、今回の 8 行。18 行のうち 148、156、255 は sync-docs の前の slice が書いた注)。Trigger 欄に `spec-org-role-models` を含む行は 31 行(`ccd88ec5` では 23 行。150、161、201、223、371 は書き方が違う)
- plan の digest: `./scripts/plan-visual.sh digest` は `ba806f4c1584`(承認時と同じ)
- 打っていないもの: `run-static-verify.sh`、`run-test.sh`(Go、help、skill の検査対象のコードは変えていない。skill の文は `check-skill-sync.sh`、`check-sync.sh`、上の 2 つの `go test -run` で見た)
- `git status --short`: 変更は skill の 4 つの写し、plan、spec、tech-debt、この report、insight event の jsonl。迷子のファイルはない(下の Insight event のあとで確かめた)

## lead へ(cycle 2)

- 全部 docs と skill の変更で、コードも help も変えていない。pipeline の再実行は要らない。cycle 2 が上限の最後の run なので、V2-1 のようなコメントの直しは tech-debt に回した
- 179・183・361 の注は、コメントだけの変更を当たりに数えた判断。外してよい(上の表の下に書いた)
- PR 本文に書く数は、sync-docs が足した carry-over の注が cycle 1 の 17 行と今回の 8 行で 25 行(cycle 1 の report は 17 行と書いた)。sync-docs の前の slice が書いた 6 行(148、150、156、161、201、255)は入れていない
- `/org` skill の一覧に、依頼にない `ralph doctor` の envelope の行を足した。要らなければ 1 句を削れば足りる(4 つの写しは `cp` と `sync-skills.sh` で作り直せる)
- 未決の点: `ralph doctor` は `./ralph.toml` だけを読む(cycle 1 の verify I1)。今回も手を付けていない
- diff の大きさ(/pr が walkthrough を決めるための数): `git diff d631e9f8...HEAD` は 97 ファイル、+7,962/−2,507(コミット済みの分。この cycle の未コミットの編集は、この report を除いて +28/−21)。Go は 68 ファイルで +5,785/−2,049(本番コード 25 ファイルで +1,170/−729)。docs を除くと 84 ファイルで +6,650/−2,460。walkthrough はここでは書いていない

## Insight event(cycle 2)

`./scripts/insights-append.sh --slug org-role-models --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を打った。cycle は `cycle-count.json` と同じ 2。
