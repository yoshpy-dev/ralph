# Verify report: org-role-models

- Date: 2026-11-05(cycle 1)、2026-11-06(cycle 2)
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(AC1〜AC13、D1〜D9、Progress の S1〜S8)。spec は `docs/specs/2026-11-05-org-role-models.md`(FR-1〜FR-9、受け入れ条件 15 件)
- Verifier: verifier サブエージェント(`/verify`)。cycle 2 は cross-review cycle 1 の ACTION_REQUIRED #1 を直した b4cfa641(S8)と、計画を S8 に合わせた 1f924de7 のあとの run
- Evidence: `docs/evidence/verify-2026-11-05-org-role-models.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。cycle 1 の Probe 1〜6 の後ろに、cycle 2 の run-static-verify の全出力と Probe 7〜11 を足した。run-verify 自身のログは cycle 1 が `docs/evidence/verify-2026-11-05-123137.log`、cycle 2 が `docs/evidence/verify-2026-11-05-152110.log`(どちらも対象外)
- この report は cycle ごとに書き足す。上が cycle 2 で、`## Verdict` は cycle 2 の判定。cycle 1 の記録は下の「Cycle 1」に、書いたときのまま残した(見出しの段だけを下げた)

## Cycle 2(HEAD 1f924de7)

### Scope(cycle 2)

- `git diff d631e9f8...HEAD` は 96 ファイル、+7539 / -2507。cycle 1 の HEAD eae6dd39 からの差は 32 ファイル(+1476 / -215)で、記録を除くと 23 ファイル(+899 / -159)。コードは b4cfa641 の 10 ファイル(テスト 10 ファイル)と、/test が足したテストの 19aa4144。文書は /sync-docs の 2cabff9a(AGENTS.md、recipe の 2 つの写し、tech-debt)と、計画・tech-debt の 0cbc6ebb・a6412dda・1f924de7
- 見たもの: AC3・AC4・D4・D9 と、冪等の打ち直し、director start の `driver_pool` の検査、start --plan の `DriverFromRoleEntry`、`ralph insights` の表示。ほかの AC は、S8 が触ったところだけを読み直した。文書は skill・rule・README・3 つの動詞の help が検査の場所を正しく書くか、tech-debt の Trigger(この branch 全体で)、計画と spec の本文
- Go のテストと `./scripts/run-test.sh` は打っていない(/test の担当)。HEAD から `go build ./cmd/ralph` した binary を一時ディレクトリに置き、一時の台帳で Probe 7〜9 を打った。herdr と agmsg は PATH になく(`PATH=/usr/bin:/bin`)、見張りは `--detach` で立てて trap の `watch --stop` で止め、終わったあとに `pgrep -fl verify-c2-bin` が何も返さないことと、見張りの pid がないことを確かめた
- worktree に書いたのは、この report、計画の AC3・AC4 の `[x]`、insight event の 1 行だけ。コードは変えていない
- 計画の digest: AC3・AC4 に印を付ける前もあとも `./scripts/plan-visual.sh digest` は `ba806f4c1584` で、`- Approved:` の行と同じ
- base: `origin/main` は d631e9f8 のままで、branch は遅れていない。`git merge-tree --write-tree origin/main HEAD` は rc 0

### 前の回の所見の扱い(cycle 2)

| ID | 状態 | 根拠 |
| --- | --- | --- |
| V1 | 解消 | 2cabff9a が tech-debt の 135・188・190・223・414 行に 2026-11-05 の carry-over の注を足した(cell 4 に `org-role-models` の注がある行を python で数えた) |
| V2 | 解消 | 152 行が「twelve」になった(「ten other」は残っていない) |
| I1〜I5 | 対応済みか、対応不要のまま | I4 の AC の印は cycle 1 のあとに付き、AC3・AC4 だけを cycle 2 のために外していた。I5 は 2cabff9a が AGENTS.md の `internal/org/` の行に `role_models.go` を足した |

### Spec compliance(cycle 2)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC3 値の検査(FR-5、D9、D4) | Met | 読み込みの検査(driver の値、モデルの文字、`""` は通る、`driver_pool` と突き合わせない)は cycle 1 から変わっていない(S8 は `internal/config/config.go:158-164` のコメントだけを直した)。フラグなしの起動の拒否は、`ResolveRoleModel` が役割のエントリから取った driver に `DriverFromRoleEntry` を立て(`internal/org/role_models.go:39`、`:69`)、spawn と `start <task>` と `start --plan` は `ValidateSpawnEnvelope`(`internal/org/envelope.go:66`)の `driverPoolErr`(`:143`)が、役割とエントリと `driver_pool` を名指して断る。director start は `checkDirectorStart` が同じ `driverPoolErr` を呼ぶ(`internal/org/director_start.go:1030`)。CLI は 4 か所とも `DriverFromRoleEntry` を渡す(`internal/cli/org.go:947`、`:1161`、`:1278`、`internal/cli/org_director_headless.go:348`)。start --plan は `StartFeatureParams.DriverFromRoleEntry` から leader の `SpawnParams` に渡る(`internal/org/feature.go:99`、`:410`)。冪等の打ち直しは、`Spawn` の本番の道でロックの下の `idempotentRespawn`(`internal/org/spawn.go:817`)が `ValidateSpawnEnvelope`(`:846`)より前に返り、start --plan の事前検査 `spawnPrecheckErr` も既に立っている座席では冪等の判断だけをする(`:1385` 以下)。テストは `TestOrgSpawn_RoleModels_RetryAfterDriverPoolNarrowed_ReturnsTheSeat`、`TestOrgStart_RoleModels_LeaderRetryAfterDriverPoolNarrowed_ReturnsTheLeader`、`TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool`(`internal/cli/org_role_models_retry_test.go`)、`TestOrgSpawn_RetryAfterDriverPoolNarrowed_ReturnsTheSeat`(`internal/org/spawn_role_models_retry_test.go:19`)、`TestValidateSpawnEnvelope_DriverPoolRefusalWording`(`internal/org/envelope_test.go:197`)、`TestStartDirector_RefusesBeforeWritingOrCallingHerdr` の 3 つの新しい case、`TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused`。Probe 7 と 8 と 9 が binary で同じ結果を出した(Observational checks)。 |
| AC4 解決(FR-3、D4) | Met | `ResolveRoleModel` は文字と `driver_pool` を見なくなり、決めるだけになった。残る拒否はエントリのない役割(`case !hasEntry`、`role_models.go:76`)と、エントリと違う `--driver` だけでエントリのモデルが空でないとき(`default:`、`:87`)の 2 つで、どちらも FR-3 の規則。`TestResolveRoleModel` の 19 case が、両方のフラグ、`--model` だけ、エントリと同じ `--driver` だけ、違う `--driver` だけ(`--model` を名指して断る)、エントリのモデルが `""` で違う `--driver`(CLI の既定)、エントリのない役割の 3 通り、文字の規則に合わない `--model` を検査せずに渡すこと、`driver_pool` の外の driver に印を付けて渡すことを打つ。文字の規則に合わない `--model` は、新しい座席では `ValidateSpawnEnvelope` が、director start では `checkDirectorStart`(`director_start.go:1033`)が断り、既に立っている座席では冪等の return が先に返る。テストは CLI の `refusalCases` と `retryCases`(spawn と start)、`TestOrgSpawn_RetryWithAModelNotOneWord_ReturnsTheSeat`(`spawn_role_models_retry_test.go:74`)、director start の「a model that is not one word」の 2 case、start --plan の新しい leader は org 層の `TestStartFeature_RefusedBeforeAnyRecord` の同名の case(`internal/org/feature_test.go:1038`)。 |
| D4・D9 の場所の移動 | Met(文の 2 点は I2-1) | 2 つの検査は `Spawn` の冪等の return のあとと `checkDirectorStart` にあり、計画の D4・D9・Scope の「設定」の項・Affected areas と合う。D4 の関数の形と「設定が変わっても」の言い方は I2-1。 |
| 新しい座席の拒否の記録 | Met(base と同じ) | driver_pool と文字の拒否は `reject()` を通るので、新しい座席では `rejected` と honored=false の receipt が残る。base の `model_pool` の拒否と同じ形で、cycle 1 の CLI 側の拒否(何も残さない)から戻った。spec は spec AC 4(`--driver codex` だけ)にだけ「何も記録しない」と書き、そちらは今も `ResolveRoleModel` の拒否なので何も残らない(`TestOrgSpawn_RoleModels_RefusedWithNothingRecorded`、`TestOrgStart_DriverOtherThanTheLeaderEntry_NonZeroExit` の `wantRejected=false` の行)。 |
| `ralph insights` の表示(self-review N1) | Met | `internal/cli/insights.go:231` が `org.ModelOrCLIDefault` を通す。`--json` は空の一覧のまま。テストは `TestInsightsCmd_ReceiptsNoCommandedModelShowsCLIDefault`(`internal/cli/insights_test.go:395`)で、text の 2 行、空の `commanded=  ` がないこと、`--json` の `commanded_models` が空であることを見る。 |
| AC1・AC2・AC5〜AC12 | Met(cycle 1 のまま) | S8 は `config.Load`、空のモデルの起動、読み先、残ったキー、doctor の検査の本体、init、ロックステップ、雛形を変えていない(差はコメント、`ResolveRoleModel` の戻り値の型、フィールドの受け渡し)。AC5 のフラグなしの解決(`driverFlag == ""` の case)はエントリの driver とモデルを返すままで、Probe 8 の s2 はエントリの codex / `gpt-6.1-sol` に解決したうえで `driver_pool` で断られ、エントリを名指す文が出た。s6 は、エントリと同じ `--driver codex` だけでエントリのモデルを取り、`--driver` 由来の文で断られた。2 つの `rejected` の event はどちらも `driver=codex model=gpt-6.1-sol` を記録した。AC12 の `check-skill-sync.sh` と `check-sync.sh` は cycle 2 でも通った。 |
| AC13 全体 | 静的な半分は Met。テストの半分は /test | `./scripts/run-static-verify.sh` は rc 0。 |

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、"All verifiers passed.") | `templates/base/ralph.toml` が未分類なので full fallback になり、golang pack を選んだ。範囲は cycle 1 と同じ。 |
| gofmt / go vet / golangci-lint / staticcheck(golang pack) | pass | `gofmt: ok`、golangci-lint は `0 issues.`。go vet と staticcheck は何も出さない。`Skipping` の行はない。手元の Go は go1.27.3。 |
| shellcheck、hook の `sh -n`、guard の awk、settings の `jq -e` | pass | |
| `scripts/check-sync.sh` | pass | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6。 |
| `check-pipeline-sync.sh` / `check-skill-sync.sh`(14 skill)/ `check-template-purity.sh` / tech-debt の plan 参照 | pass | |
| branch secret scan | pass | d631e9f8..1f924de7 は clean。 |

### Documentation drift(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org spawn --help`・`start --help`・`director start --help` | Yes | S8 は help の文字列を変えていない。`orgRoleModelsHelp`(`internal/cli/org.go:808` 以下)も director start の Long も、`driver_pool` と文字の検査がどこで走るかは書かず、書いてあることは S8 のあとも正しい。 |
| `/org` skill(4 つの写し) | Yes(I2-3) | 「役割ごとのモデル」節の「`--model` は herdr を呼ぶ前に」と「`driver_pool` にない driver」の項は、新しい座席について正しい。打ち直しが既存の座席を返すことは書いていないが、言い切ってもいない。`(CLI default)` の出る場所の一覧は I2-3。 |
| `/director` skill(4 つの写し) | Yes | 検査の場所を書いていない。 |
| `.claude/rules/ralph/model-routing.md`(2 つの写し) | Yes | 検査の場所を書いていない。 |
| `README.md` | Yes | 268 行の段落は役割のモデルと残ったキーだけを書き、検査の場所を書かない。 |
| `AGENTS.md` の repo map | Yes | `internal/org/` の行が `role_models.go` の `ResolveRoleModel` と `[org.roles]` の拒否を名指す(2cabff9a)。検査の場所は書かない。 |
| `docs/recipes/codex-seat-permissions.md`(2 つの写し) | Yes | 2cabff9a の 1 文(director の alias)が入り、写しは `check-sync.sh` で同じ。 |
| Go のコメント | No(V2-1) | `ResolveRoleModel`、`ValidateSpawnEnvelope`、`checkDirectorStart`、`DriverPool`、`applyMainWorktreeOrgLimits`、`checkOrgRoleModelDrivers` のコメントは S8 に合わせて直っている。テストのコメントが 1 つ残った(V2-1)。 |
| 計画 | Yes(I2-1、I2-4) | Scope の「設定」の項と Affected areas は 1f924de7 で S8 に合った。D4 の本文と S2 の行の 2 点は I2-1 と I2-4。 |
| spec | 注記(I2-2) | FR-5 と spec AC 12 は新しい座席の言い方で、打ち直しの例外を書かない。 |
| `docs/tech-debt/README.md` | No(V2-2) | 430 行の (e) に N3、(b) と (c) に S8 の解消が入った。cell 4 の Trigger を branch 全体で読み直すと、注のない発火が 5 行(と、`Stop` のコメント 1 行で発火したと読める 3 行)ある。 |

### Findings(cycle 2)

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V2-1 | LOW | `internal/cli/org_start_plan_config_test.go:375-379` のコメントが、S8 の前の検査の場所を書いたまま残った。「A model that is not one word ... is refused by the CLI before StartFeature since plan 2026-11-05-org-role-models」とあるが、b4cfa641 のあと `runOrgStartPlan` の `ResolveRoleModel` は文字を見ない。この形の `--model` は `StartFeature` の事前検査 `spawnPrecheckErr` の `ValidateSpawnEnvelope`(`internal/org/spawn.go:1402`)が worktree を作る前に断る。テストの選んだ拒否(権限の検査)は今も正しく、理由の文だけが違う。このコメントは S3 の 1bc70f0b が書き、S8 は同じ型のコメントを `org_test.go`(653 行と 2459 行のあたり)と `org_role_models_test.go` で直したが、このファイルは触れていない。 | `git log -S'refused by the CLI before StartFeature'` は 1bc70f0b だけ。`git grep -n 'refused by the CLI'` の当たりはこの 1 か所。 | 「is refused by StartFeature's pre-check (`spawnPrecheckErr`) before the worktree is made」の形に直す。コードのコメントなので、直すと全パイプラインを回し直すことになる。cycle の上限なら tech-debt の 430 行に (f) として足し、Trigger を「`org_start_plan_config_test.go` の次の変更」にする。 |
| V2-2 | LOW | tech-debt の Trigger のうち、この branch が発火させたのに carry-over の注がない行がある。cycle 2 の S8 による発火: (1) 381 行。Trigger は「`reject`、receipt の形、`ralph insights` の Receipts 節の次の変更」で、b4cfa641 が Receipts 節の text を変えた(N1)。S8 は新しい座席の `driver_pool` と文字の拒否を `reject()` に戻したので、この行の Debt(拒否が honored=false の receipt を残す)が当たる拒否も cycle 1 より増えた。(2) 395 行。Trigger は「`resolvedOrClean` の呼び出し元か `statedir.go` の次の変更」で、呼び出し元の `feature.go`(`:219`)を S8 が変えた(`StartFeatureParams.DriverFromRoleEntry`、`startFeatureLeaderParams`)。この行は前にも `feature.go` のコメントの変更で発火を記録している。cycle 1 の sweep が拾わなかった発火(V1 の 5 行のほか): (3) 191 行。「the `Spawn` doc の次の変更」で、S2 の bf1f4727 が `Spawn` の doc の手順 2 と 3 を書き換えた。(4) 303 行。「`watch_test.go` の次の変更」で、2 行が消えた。(5) 396 行。「`director_cmd_test.go` の次の変更」で、1 行が変わった。ほかに、179 行・183 行 (a)・361 行の「`Stop` の次の変更」には、bf1f4727 が `Stop` の中のコメント 1 行(`token = receipt.Honored` の行)を変えたことが当たると読める。 | Probe 10(cell 4 を `re.split(r"(?<!\\) \| ", ...)` で取り、cycle 1 の HEAD eae6dd39 からと base d631e9f8 からの変更ファイルと Go の関数名を突き合わせ、2026-11-05 の org-role-models の注がない行を出した)。191 行の発火は `git show bf1f4727 -- internal/org/spawn.go` の `Stateless envelope validation` の 4 行。303 と 396 は `git diff --stat d631e9f8 HEAD -- internal/org/watch_test.go internal/org/director_cmd_test.go`(-2 と ±1)。 | /sync-docs で (1)〜(5) に「2026-11-05: docs/spec-org-role-models hit this trigger with ..., but did not pay it ... Carried over」の注を足す。381 行には、S8 で拒否の receipt が base と同じ形に戻ったことも書く。179・183・361 行はコメントだけの変更なので、注を足すかは判断に任せる。319・365・414 行の行数は「at a3c8c972」と書いてあり、そのときの値としては合う(HEAD では `director_start.go` 1241、`spawn.go` 2740、`doctor.go` 1615、`config.go` 1121)。 |
| I2-1 | INFO | 計画の D4(80 行)の本文 2 点が S8 のあとのコードと合わない。(a) 関数の形が `(driver, model string, err error)` のままだが、b4cfa641 から `(ResolvedRoleModel, error)`(`internal/org/role_models.go:69`)。(b) 「既に立っている座席の打ち直しは、設定が変わっても今どおり既存の座席を返す」は、`driver_pool` と `--model` の文字については正しいが、self-review N3 の 2 つ(エントリのない役割でフラグが欠ける、`--driver` だけでエントリのモデルが空でない)は `Spawn` の前にあるので、`[org.role_models]` を最初の spawn のあとに変えた打ち直しは断られうる。AC3 と AC4 の文は狭く書かれていて、どちらも満たす。N3 は tech-debt 430 行の (e) にある。 | `role_models.go:76`、`:87`、`internal/cli/org.go:941-947`。 | digest の中の D の本文なので、cycle の中では直さなくてよい。直すなら (a) の形と、(b) を「`driver_pool` を狭めても、`--model` の文字の検査に合わなくても」にする。 |
| I2-2 | INFO | spec FR-5 の「`--model` の値も同じ文字の規則で、herdr を呼ぶ前に断る」(54 行)と spec AC 12(94 行)は、新しい座席の言い方のまま。S8 のあと、既に立っている座席の打ち直しは文字を見ずに座席を返す(Probe 7)。FR-5 の D9 の変更には日付の注があるが、S8 の例外には注がない。計画の D4 と AC4 は例外を書いている。 | `docs/specs/2026-11-05-org-role-models.md:54`、`:94`。 | 要約と端の場合の差で、動作は計画の AC4 どおり。spec を直すなら FR-5 に 1 句の注を足す。 |
| I2-3 | INFO | `/org` skill の「役割ごとのモデル」節(213〜215 行)は、`(CLI default)` と出すものを「spawn と start の出力行、`ralph org report`、`director status`」と挙げる。S7 の `ralph org status`・`ralph status` と、S8 の `ralph insights` も同じ表示になった。「だけ」とは言っていないので、例の一覧として読める。 | `.claude/skills/org/SKILL.md:213-215`、`internal/cli/org.go` の `printStatusTable`、`internal/cli/insights.go:231`。 | 任意。足すなら 4 つの写しをそろえる。 |
| I2-4 | INFO | 計画の Implementation outline の S2(106 行)は「`ResolveRoleModel` は解決した driver が `driver_pool` にあるかを確かめる(D9)」のまま。S8 の行(111 行)が後ろにあり、記録としては読める。self-review N4 (c) の点で、Progress は N4 を「計画の本文を直した」とだけ書き、(c) は直さず register にもない。 | 計画 106 行と 111 行、167 行。 | archive の前に S2 の文の末尾へ「(S8 で `ValidateSpawnEnvelope` と `checkDirectorStart` に移した)」と足すか、そのままにする。 |
| I2-5 | INFO | tech-debt 430 行の Related は self-review の「(Run 2)」だけを挙げ、(e) N3 の出どころの Run 3 を挙げない。Why deferred の「Each is cosmetic」は、(e) が表示や文言でない点と合わない(Impact の列は (e) を正しく書く)。 | `docs/tech-debt/README.md:430`。 | /sync-docs で Related に「(Run 2 and Run 3)」と書く。 |

### Observational checks(cycle 2)

すべて HEAD 1f924de7 から作った binary で、herdr と agmsg のない PATH、一時の HOME、`RALPH_LOCAL_CONFIG=off`。一時の台帳に、codex で立った座席 o1/s1(reviewer、`gpt-6.1-sol`、pane p1)の `spawn_started` と `spawned` を書いておき、`driver_pool = ["claude"]` と `reviewer = { driver = "codex", model = "gpt-6.1-sol" }` の設定(narrow.toml)で打った。

- Probe 7(既に立っている座席の打ち直し): 同じ `--driver codex --model gpt-6.1-sol`、フラグなし、`--driver codex --model "not a real model"` の 3 通りとも rc 0 で `seat "s1" already spawned (org_id=o1 driver=codex model=gpt-6.1-sol pane_id=p1)` を出し、manifest は 2 行のまま、receipt は 0 件のまま。herdr が PATH にないので、herdr を呼んでいれば失敗していた。
- Probe 8(同じ設定の新しい座席): フラグなしの s2 は rc 1 で `org: role "reviewer" runs on driver "codex" in [org.role_models].reviewer, which is not in [org].driver_pool [claude]: add codex to driver_pool, change the entry, or pass both --driver and --model`。`--driver codex --model gpt-6.1-sol` の s3 と `--driver codex` だけの s6 は `org: driver "codex" not in [org].driver_pool [claude]`。`--model "not a real model"` の s4 と `--model=-x` の s5 は `org: model ... must be "" ... that does not start with '-'`。5 つとも最後の event が `rejected` で、receipt が 1 件ずつ増えた(0 → 5)。s1 の event は 2 のまま。
- Probe 9(director start、どれも書く前に断る): 既定の設定で `--model "not a real model"` と `--driver claude --model=-x` は `org: director: start: model ... must be ""`。`driver_pool = ["codex"]` と `director = { driver = "claude", model = "" }` でフラグなしは `org: role "director" runs on driver "claude" in [org.role_models].director, ...`、`--driver claude --model sonnet` は `org: driver "claude" not in [org].driver_pool [codex]`。4 つとも rc 1、`director.jsonl` も `prompts/` もできなかった。CLI の Go のテストにない director start の文字の拒否は、この probe で CLI から確かめた。
- Probe 10: tech-debt の Trigger の sweep(V2-2)。
- Probe 11: 計画の digest と、`origin/main` との合わせ(Scope)。

### Coverage gaps(cycle 2)

- Go のテストの結果は /test の担当。この report の「Met」は、コードとテストの assert を読み、binary で Probe 7〜9 を打って確かめたもの。
- self-review N3 の 2 つの拒否(`[org.role_models]` を変えたあとの打ち直し)を打つテストはない。tech-debt 430 行の (e) にある。
- start --plan の文字の規則に合わない `--model` を CLI から打つテストはなく、binary でも打っていない(git の repo と承認した分割計画が要る)。`runOrgStartPlan` は `resolved.Model` をそのまま `StartFeatureParams` に渡すので、org 層の `TestStartFeature_RefusedBeforeAnyRecord` の case と同じ道を通ると読んだ。
- Probe 7 の打ち直しは台帳に書いた `spawned` の座席に対してで、herdr で実際に立てた座席ではない。冪等の return は manifest の `Roster` だけを見るので、結果は同じはず。
- cycle 1 の Coverage gaps(`[org.roles]` を残した設定の stop と disband の本番の道、spec AC 2 の implementer、独自の役割のフラグなしの spawn、leader の雛形の手順 2、herdr の pane での `--model` なしの座席)は変わっていない。cycle 1 で Go のテストがないと書いた stop と disband は、19aa4144 の `TestOrgCleanupVerbs_LegacyModelKeys_StillWork` が足した。

### Insight event(cycle 2)

`docs/insights/events/2026-11-05-org-role-models.jsonl` に 1 行足した(phase verify、`--cycle auto` は `cycle-count.json` の 2、verdict pass、low 2)。

## Verdict

pass

- Verified: AC3 と AC4(コード、テストの assert、binary の Probe 7〜9)。冪等の打ち直しが `driver_pool` を狭めても、文字の規則に合わない `--model` でも既存の座席を返し、何も書かないこと。新しい座席と director start は同じ要求を断り、エントリから取った driver では役割とエントリを名指すこと。start --plan に `DriverFromRoleEntry` が届くこと(CLI のテスト)。`ralph insights` の `(CLI default)`。AC13 の静的な半分(run-static-verify の全項目)。計画の digest は `ba806f4c1584` のまま。
- Partially verified: start --plan の文字の規則に合わない `--model` は org 層のテストだけ。N3 の 2 つの拒否は読んだだけ。
- Not verified: Go のテストの結果と `run-test.sh`(/test)。herdr で実際に立てた座席の打ち直し。
- 計画の AC3 と AC4 に `[x]` を付けた。AC1〜AC13 はすべて `[x]`。
- 所見: LOW 2 件(V2-1、V2-2)、INFO 5 件(I2-1〜I2-5)。どれも merge を止めない。
- /sync-docs に回すもの: V2-2(tech-debt の 381・395・191・303・396 行の carry-over の注。179・183・361 行は判断に任せる)、I2-5。V2-1 はテストのコメントなので、cycle の上限なら tech-debt の 430 行に足す。

## Cycle 1(HEAD eae6dd39)

- Date: 2026-11-05
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(AC1〜AC13、D1〜D9、Progress の S1〜S7)。spec は `docs/specs/2026-11-05-org-role-models.md`(FR-1〜FR-9、受け入れ条件 15 件)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff d631e9f8...HEAD`(base main d631e9f8、HEAD eae6dd39、85 ファイル、+6247 / -2476)。spec の準拠、静的解析、文書のずれを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない(`go test` は 1 度も打っていない)。HEAD から `go build` した binary を一時ディレクトリに置き、`ralph doctor`、3 つの動詞の `--help`、一時の台帳での `spawn --dry-run`・`status`・`stop --dry-run`・`disband --dry-run` を打った。herdr と agmsg は PATH になく、見張りは一時の台帳に `--detach` で立てて trap で止めた(終わったあとに残っていないことを `pgrep` で確かめた)。worktree には report と insight event のほかに何も書いていない。
- Evidence: `docs/evidence/verify-2026-11-05-org-role-models.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。中身は run-static-verify の全出力と、末尾の Probe 1〜6(doctor の 4 通りの設定、`--help` の flag の行、残った語の `git grep`、`events[len(events)-1]` の数、`spawn --dry-run` の 6 通り、`[org.roles]` を残した設定での status・stop・disband)。run-verify 自身のログは `docs/evidence/verify-2026-11-05-123137.log`(同じく対象外)。
- 計画の digest: `./scripts/plan-visual.sh digest` は `82b7b97155f4` で、`- Approved:` の行と同じ。
- Insight event: `docs/insights/events/2026-11-05-org-role-models.jsonl` に 1 行足した(phase verify、cycle 1 は `cycle-count.json` と同じ、verdict pass、low 2)。

### Spec compliance(cycle 1)

計画の AC は spec の FR と受け入れ条件を束ねているので、AC ごとに対応する FR と spec の受け入れ条件(spec AC の番号は spec の並び順)を書いた。テストは中身を読んで、何を assert しているかを確かめた。結果は /test が出す。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 設定と既定(FR-1、FR-2、spec AC 1・2) | Met | 既定の 2 組は `DefaultRoleModels`(`internal/config/role_models.go:83-94`)。`Default()` は claude の組(`internal/config/config.go:759`)。`Load` は表を空の map から読み(`config.go:864`)、2 つのファイルを当てたあとで `fillDefaultRoleModels`(`config.go:900`、`role_models.go:100-109`)が使われる `driver_pool` の組で足りない役割を埋める。テストは `TestDefaultRoleModels_PicksTheSetByDriverPool`、`TestDefault_RoleModelsIsTheClaudeSet`、`TestLoad_RoleModelsFollowTheDriverPoolInEffect`、`TestLoad_RoleModelsKeepWhatTheFilesSet`(`internal/config/role_models_test.go`)。Probe 1 の case a と b で、doctor の envelope の行が claude の組と codex の組をそのとおりに出した。 |
| AC2 ファイルごとの検査(FR-5、spec AC 7) | Met | `checkRoleModelEntries`(`role_models.go:134-159`)は pointer の probe でキーの有無を見て、`ralph.toml`(`config.go:872`)と `ralph.local.toml`(`config.go:891`)のそれぞれで呼ばれる。エラーはファイルの path と役割を名指す。`TestLoad_RoleModelsEntryMustSetBothKeys` がインラインと表見出しの両方を打ち、`TestLoad_RoleModelsKeepWhatTheFilesSet` が `ralph.local.toml` の 1 役割の置き換えを打つ。 |
| AC3 値の検査(FR-5、spec AC 8・12) | Met | driver の値は `role_models.go:151-153`、モデルの文字は `CheckOrgModelName`(`role_models.go:118-123`、`reviewerWordPattern` と先頭の `-`、`""` は通す)。`Load` は driver と `driver_pool` を突き合わせない(D9)。`driver_pool` そのものは `validateLoaded` が空と未知の値を断る(`config.go:940-947`、S5b)。テストは `TestLoad_RoleModelsValues`、`TestCheckOrgModelName`、`TestLoad_RoleModelsDriverOutsideDriverPoolLoads`、`TestLoad_DriverPoolMustNameKnownDrivers`。spawn 側の拒否は AC4 の行。 |
| AC4 解決(FR-3) | Met | `ResolveRoleModel`(`internal/org/role_models.go:52-87`)の switch を spec FR-3 の 4 つの規則と 1 つずつ突き合わせた。両方のフラグ、`--model` だけ(エントリの driver)、エントリと同じ `--driver` だけ、違う `--driver` だけ(エントリのモデルが `""` なら通り、そうでなければ `--model` を名指して断る)、エントリのない役割(役割と `[org.role_models].<role>` を名指して断る)、文字の規則に合わない `--model`、解決した driver が `driver_pool` にないとき(エントリ由来と `--driver` 由来で文が分かれる)。`TestResolveRoleModel` の 23 ケースがこれをすべて打つ。spec AC 3 は、エントリと `--model` の値を入れ替えた形(エントリ gpt-6.1-sol、`--model gpt-6-astra`)で打たれていて、通る道は同じ。 |
| AC5 動詞(FR-3、spec AC 1・4・5・6) | Met | spawn(`internal/cli/org.go:940-944`)、`start <task>`(`:1152-1156`)、`start --plan`(`:1270`)、director start(`internal/cli/org_director_headless.go:333`)が `ResolveRoleModel` を呼んでから動く。spawn と director start の `--driver` は必須でなく、start の `--driver` の既定は `""`(`org.go:957`、`:1172`)。`resolveModelOrWarn` は消えた。テストは `TestOrgSpawn_RoleModels_DefaultsWithoutDriverOrModel`(9 ケース、stderr が空であることも見る)、`TestOrgSpawn_RoleModels_RefusedWithNothingRecorded`(manifest、receipt、herdr、agmsg が空)、`TestOrgStart_RoleModels_LeaderEntryWithoutDriverOrModel`(`driver_pool = ["codex"]` で leader が codex になる)、`TestOrgDirectorStart_RoleModels_NoFlagsLaunchesTheCLIDefault`、`TestOrgSpawnStartAndDirectorStartHelp_RoleModels`(`--driver` の行に required も既定値もない)。HEAD の binary の spawn では Probe 5 の 6 通りが同じ結果になった(Observational checks)。 |
| AC6 モデルが `""`(FR-1、D5、spec AC 5) | Met | 座席の argv は `if p.Model != ""` のときだけ `--model` を足す(`internal/org/spawn.go:1077-1079`)。`spawn_started` の印は本番(`:1465`)と dry-run(`:2302`)の 2 か所。herdr の director と `launch_cmd` の起動スクリプトは同じ `directorAgentArgs`(`internal/org/director_start.go:953-959`、`internal/org/director_cmd.go:324`)。登録は `AttachHeadless`(`director_start.go:134`)が `CheckOrgModelName` で `""` を通す。receipt は `codexFoundReceipt`(`spawn.go:1992`)が指示なしのとき honored を `unknown` にし観測したモデルを入れ、stop は印のある event だけ観測する(`internal/org/verbs.go:1904`)。テストは `internal/org/spawn_role_models_test.go` の 7 本、`internal/org/director_role_models_test.go` の 6 本、`TestOrgDirectorStart_RoleModels_LaunchCmdWithoutModel`、表示の `internal/cli/org_model_display_test.go` の 3 本。印のない古い event は `TestOrgStop_Codex_EmptyModel_MarkDecidesTheObservation` と既存の `TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved` が固定する。 |
| AC7 読み先(FR-4、spec AC 9・13) | Met | `applyMainWorktreeOrgLimits`(`org.go:236-243`)が `RoleModels`・`DriverPool`・`LegacyModelKeys` を main worktree の値で丸ごと置き換え、spawn と `start <task>`(`newOrgSpawnRuntimeAt`)と `start --plan`(`resolveOrgStartConfig`)が共有する。director start は `withMainWorktreeDirectorStartConfig`(`org.go:301`)。`TestOrgSpawn_RoleModels_ReadFromMainWorktreeRalphToml` の 6 ケースが、`--config` なしで main の値、`--config` でそのファイルの値、写しの古い `driver_pool = ["claude"]` と main の `ralph.local.toml` の codex の reviewer の組み合わせで codex を打つ。`TestOrgStartPlan_RoleModels_ReadFromMainWorktreeRalphToml` が `start --plan` を打つ。 |
| AC8 残ったキー(FR-6、spec AC 10・11) | Met(`status`・`stop`・`disband` が通る半分は Probe 6 とコードで確認。Go のテストはない) | Probe 5: `model_pool` を残した `--config` で spawn は通り、stderr に 1 行だけ出す。`[org.roles]` を残した `--config` では rc 1 で、manifest の行数は変わらない。Probe 6: 同じ `[org.roles]` の設定で `status` は rc 0 で座席を出し、`stop --dry-run` と `disband --dry-run` は rc 0 で `stopped` と `disbanded` を書いた。`config.Load` は記録するだけで止まらない(`legacyModelKeys`、`role_models.go:167-182`、`TestLoad_LegacyModelKeysLoadInAnyShape`)。`[org.roles]` の拒否 `LegacyRolesErr` を呼ぶのは `retiredRoleConfigErr`(`spawn.go:1711`)と director start(`org_director_headless.go:330`)だけで、`retiredRoleConfigErr` は Spawn の本番(冪等の return のあと)、dry-run、`start --plan` の事前検査 `spawnPrecheckErr`(worktree を作る前)の 3 か所から呼ばれる。stop・disband・status の道にはない。テストは `TestOrgSpawn_LegacyOrgRolesRefused`(CLI、本番と dry-run で何も書かない)、`TestOrgSpawn_LegacyRolesTable_*` の 3 本、`TestSpawnPrecheckErr_MatchesSpawn` の「an [org.roles] table」、director start の `org_director_headless_test.go:257` のケース。`model_pool` の 1 行は `TestOrgSpawn_RoleModels_LegacyModelPoolNoticeOnce` と、start・director start の各テストの stderr の比較。`[org.role_models].lead` は `RetiredRoleConfigKeys` が拾う(`prompts_test.go` の 2211 行からのケース)。 |
| AC9 doctor(FR-7、spec AC 8・10・11・14) | Met | `checkCodexModelSlugs`(`internal/cli/doctor_codex_models.go:314`)は codex のエントリでモデルが空でないものだけを重複なしで見る。`checkOrgModelProbes`(`internal/cli/doctor.go:1120`)は組を重複なしで起動し、`ProbeModel`(`internal/org/driver/probe.go:22`)は `""` のとき `--model` を付けない。`codexSeatPossible`(`doctor_codex_writable_root.go:279-281`)は `driver_pool` に codex があるかだけを見る。envelope の行は `EnvelopeSummary` の役割ごとの組。残ったキーは `checkOrgLegacyModelKeys`(`doctor.go:1058`)、`driver_pool` にない driver は `checkOrgRoleModelDrivers`(`doctor.go:1088`)。Probe 1 の case c で残ったキーの warn 2 行、case d で driver の warn と cache にないスラッグの warn が出た。テストは `internal/cli/doctor_role_models_test.go` の 5 本と `TestDoctorCheckOrgEnvelope_ReportsRoleModelsAndMaxSeats`。 |
| AC10 init(FR-8) | Met | `initSettingsToTOML`(`internal/cli/init_settings.go:143`)は `driver_pool` だけを書き、`initTOMLCommentModelPool` とその 3 つのエラーは消えた。`TestInitSettings_EveryAnswerLoadsFromTheTemplateWithTheTableValues` は 3 通りの出力を `config.Load` で読み、`model_pool` と `[org.roles]` の行がないこと、役割の既定が `DefaultRoleModels` と一致することを見る。`TestRunDoctorOpts_InitSettingsForEachCLIPass` は leader の driver を見る。 |
| AC11 ロックステップ(spec AC 15) | Met | `TestDefaultsLockStep` に「role_models」の sub-test(`roleModelsLockStepDrift`)が入り、`templates/base/ralph.toml:176-186` のコメントの 2 組を `DefaultRoleModels` と比べる。`TestRoleModelsTemplateDrift_*` の 2 本が、値の変更・役割の削除・追加・表の生書き・壊れた組でずれを検出することを固定する。 |
| AC12 雛形と文書(FR-9) | Met | leader の雛形から「`--model` は必ず明示する」とプール先頭の文が消え、手順 2・3・5・14 が「前文に指定があるときだけ渡す」になった(`internal/org/prompts/leader.md`)。`/org` skill は「役割ごとのモデル(`[org.role_models]`)」節に置き換わった。残った語の `git grep`(Probe 3)は、履歴(spec の背景と改訂注記、計画)と、残ったキーを扱うコード・テスト・文書だけで、該当しない行は見つからなかった。`check-skill-sync.sh` と `check-sync.sh` は通った。 |
| AC13 全体 | 静的な半分は Met。テストの半分は /test | `./scripts/run-static-verify.sh` は rc 0。`run-test.sh` は打っていない。 |

### Static analysis(cycle 1)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、"All verifiers passed.") | `templates/base/ralph.toml` が未分類なので full fallback になり、golang pack を選んだ。範囲は `RALPH_VERIFY_SCOPE=full` と同じ。 |
| gofmt / go vet / golangci-lint / staticcheck(golang pack) | pass | `gofmt: ok`、golangci-lint は `0 issues.`。go vet と staticcheck は何も出さない(通ったときの形)。`Skipping` の行はない。手元の Go は go1.27.3 で、`go.mod` の go の行と同じ。 |
| shellcheck(hook と verify の script)、hook の `sh -n`、guard の awk、settings の `jq -e` | pass | |
| `scripts/check-sync.sh` | pass | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6(`model-routing.md` は KNOWN_DIFF)。 |
| `check-pipeline-sync.sh` / `check-skill-sync.sh`(14 skill)/ `check-template-purity.sh` / tech-debt の plan 参照 | pass | |
| branch secret scan | pass | d631e9f8..eae6dd39 は clean。 |

### Documentation drift(cycle 1)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org spawn --help`・`start --help`・`director start --help` | Yes | Probe 2。3 つとも `--driver` の既定は役割のエントリの driver、`--model` の既定はエントリのモデルで、`""` なら `--model` なしと書く。spawn と start の Long は FR-3 の規則、`[org.roles]` の拒否、`model_pool` の 1 行、読む場所を書く。director start の Long も `[org.roles]` と `model_pool` を書く。 |
| `/org` skill(4 つの写し) | Yes | `spawn`・`start`・`director` の行と「役割ごとのモデル」節がコードと合う。写しは `check-skill-sync.sh` と `check-sync.sh` でそろっている。 |
| `/director` skill(4 つの写し) | Yes(I2) | 前文に書くのは `ralph.toml` と違う座席だけ、という手順になった。`start --plan` の例は 2 つあり、2 つ目に `--driver claude --model opus` が残る(I2)。 |
| `internal/org/prompts/leader.md` | Yes | AC12 の行。 |
| `internal/org/prompts/director.md` | Yes(I3) | モデルや前文の文がなく、直すところがない。計画の Affected areas は名前を挙げていた。 |
| `.claude/rules/ralph/model-routing.md`(2 つの写し) | Yes | 「Org runtime model receipts」節が `[org.role_models]`、`""` の receipt(honored が `unknown`)、doctor のスラッグの検査を書く。「Codex has no model aliases」は、doctor が `[reviewer].codex_model` を見なくなったことを書く(tech-debt の 429 行と合う)。 |
| `README.md` | Yes | `ralph.local.toml` の段落と、役割のモデルの段落(既定の 2 組、`model_pool` の 1 行、`[org.roles]` の拒否)がコードと合う。 |
| `docs/recipes/codex-seat-permissions.md`・`director-launch-cmd.md` と `templates/base/` の写し | Yes | 写しは `cmp` で同じ。`--model` を付けない座席の alias の扱いと、writable root の条件(`driver_pool` に codex)がコードと合う。 |
| `docs/specs/2026-08-01-org-runtime.md` | Yes | 2026-11-05 改訂の節 (a)〜(c) があり、FR-1・FR-2・FR-7 の行、2026-09-16 改訂の (c)(d)(e)、運用ノートに改訂の印がある。 |
| `docs/specs/2026-10-07-org-multi-org-director.md` | Yes | 2026-11-02 の決定の末尾に、init が `model_pool` を扱わなくなったことを足した。 |
| `AGENTS.md` の repo map | Yes(I5) | `internal/config/` の行に `role_models.go` が入った。`internal/org/` の行は `role_models.go` を名指さない(I5)。 |
| `templates/base/ralph.toml` | Yes | `model_pool` の配列と `[org.roles]` のコメント例が消え、`[org.role_models]` の 2 組がコメントで入った。`driver_pool` のコメントが既定の組の選び方を書く。 |
| `docs/tech-debt/README.md` | No(V1、V2) | 閉じた 3 行と足した 3 行、148・156・201・255 行の carry-over の注は合っている。発火した Trigger のうち 5 行に注がない(V1)。152 行の数が違う(V2)。 |
| `ralph doctor` が読む設定(FR-7 の「実際に使われる」) | 注記(I1) | doctor は `./ralph.toml` を読み、main worktree の値を取らない。 |

### Findings(cycle 1)

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | tech-debt の Trigger のうち、この branch が発火させたのに carry-over の注がない行が 5 つある。この branch は 148・156・201・255 行には「2026-11-05: docs/spec-org-role-models hit this trigger ... Carried over」の注を足しているので、同じ書き方が抜けている。(1) 135 行: Trigger は「`doctor_codex_writable_root.go` の次の変更」で、S4 が `codexSeatPossible` を入れ outcome の番号を振り直した(102 行の変更)。(2) 188 行: (a) は「leader prompt の次の変更」、(b) は「`EnvelopeSummary` か `checkOrgEnvelope` の次の変更」で、leader.md と `EnvelopeSummary` がどちらも変わった。(3) 190 行 (d) と (4) 223 行 (b): 「`withMainWorktreeOrgLimits` の次の変更」で、148 行に書いた同じ変更が当たる。223 行 (b) の改名の理由は、関数が `RoleModels`・`DriverPool`・`LegacyModelKeys` も運ぶようになって強まった。(5) 414 行: 「5 つのファイルのどれかの次の変更」で、`internal/cli/doctor.go` は 1519 行から 1613 行に増え、`internal/config/config.go` は 1194 行から 1120 行に減った。 | 行番号は HEAD eae6dd39 の `docs/tech-debt/README.md`。行を `\|` で分けて 4 列目(Trigger)を python で読み、2026-11-05 の注があるのは 148・156・201・255 行だけだった。行数は `git show d631e9f8:<file> \| wc -l` と HEAD の `wc -l`。 | /sync-docs で 5 行に carry-over の注を足す(何を変えたか、払わなかった理由、414 行は新しい行数)。払うかは判断に任せる。 |
| V2 | LOW | 152 行(閉じた `TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry` の行)は「ten other tests in `internal/cli/org_test.go` index `events[len(events)-1]` ... (counted 2026-11-05)」と書くが、len の確かめのない添字は 12 テストに 1 か所ずつある。この行を書いた 0ee2bbaf でも 12 で、そのあとに `org_test.go` を変えたコミットはない。Trigger の列も「one of the ten others」と書く。 | Probe 4: `org_test.go` の 518・551・592・627・806・1198・1589・1691・2524・2571・2654・2675 行。どれも直前に `len(events)` の確かめがない(代入の行から添字の行までを python で調べた)。 | 152 行の 2 か所の「ten」を「twelve」にする(数え方を変えたのなら、その数え方を書く)。 |
| I1 | INFO | `ralph doctor` は `<targetDir>/ralph.toml`(`.`)だけを読む(`doctor.go:110`)。spawn と start は台帳が main worktree のものなら `[org.role_models]` と `driver_pool` を main の `ralph.toml` から読む(FR-4)ので、feature worktree で打った doctor の「Org envelope」「Org role model drivers」「Org codex model slugs」「Org legacy model keys」は写しの値を表す。FR-7 の「実際に使われる `[org.role_models]`」と食い違うのはこの場合だけで、main worktree の root で打てば同じになる。`max_orgs` などで前からある形で、今回の変更が持ち込んだものではない。 | `runDoctorStateDir` の `config.Load(filepath.Join(targetDir, "ralph.toml"))`。doctor の Go に `loadMainWorktree` を呼ぶ所はない。 | 直さないなら、`/org` skill の doctor の段落に「main worktree の root で打つ」と 1 句足すか、tech-debt の 148 行(設定と台帳の読み方の違い)に doctor を足す。 |
| I2 | INFO | FR-9 は「`ralph org start --plan` の例から `--driver` と `--model` を外す」と言うが、`/director` skill の 2 つ目の例は `--driver claude --model opus` を持つ。1 つ目の例はフラグなしで、2 つ目は「`ralph.toml` の leader が codex で、前文で leader を claude に決めたとき」の例として説明が付いている。計画の Progress(S6)は、`internal/org/skill_split_questions_test.go` が文を固定するので残したと書く。 | `.claude/skills/director/SKILL.md:597-602`。 | 記録済みの逸脱で、例の意味は新しい規則に合う。対応は要らない。 |
| I3 | INFO | 計画の Affected areas と S6 は `internal/org/prompts/director.md` を挙げるが、変わっていない。この雛形にはモデル、`--driver`・`--model`、分割計画の前文の文がない(`grep` で 3 行目の `driver: {{DRIVER}}` だけ)ので、直すところがない。 | `internal/org/prompts/director.md`。 | 対応は要らない。 |
| I4 | INFO | 計画の AC1〜AC13 と spec の FR・受け入れ条件のチェックボックスは `[ ]` のまま。Progress の「Review artifact created」も、self-review の report があるのに `[ ]`。 | 計画の 87〜99 行と 164 行。 | /pr の前に orchestrator が付ける。 |
| I5 | INFO | `AGENTS.md` の `internal/org/` の行は `role_models.go`(`ResolveRoleModel`、`LegacyRolesErr`、`LegacyModelPoolNotice`)を名指さない。計画の Verify plan が見るよう求めたのは `internal/config/` の行だけで、そちらは入っている。 | `AGENTS.md` の repo map。 | 任意。足すなら `internal/org/` の行の envelope の説明の近くに 1 句。 |

### Observational checks(cycle 1)

- Probe 1(HEAD の binary の `ralph doctor`、一時ディレクトリの 4 通りの設定。HOME、ZDOTDIR、CODEX_HOME を一時の場所にし、手元の `models_cache.json` の写しを置き、`RALPH_LOCAL_CONFIG=off`)。
  - 空の `ralph.toml`: `role_models: director=claude/(CLI default), implementer=claude/opus, leader=claude/opus, reviewer=claude/opus`。残ったキーと driver は pass、スラッグは「codex のエントリがない」の info。
  - `driver_pool = ["codex"]`: 3 役が `codex/gpt-6.1-sol`、director が `codex/(CLI default)`。スラッグの検査は pass で、`gpt-6.1-sol` が手元の cache にあることも確かめられた(spec の Dependencies)。
  - `model_pool` と `[org.roles]` を残した設定: 「Org legacy model keys」が 2 行の warn を出す。`[org.roles]` の行は、3 つの動詞が断ることと、消すか移すかを書く。
  - `driver_pool = ["claude"]` と codex の reviewer(cache にないスラッグ): 「Org role model drivers」の warn と「Org codex model slugs」の warn が出る(spec AC 8・14)。
- Probe 2: 3 つの動詞の `--help` の `--driver` と `--model` の行(Documentation drift の表)。
- Probe 5(計画の Verify plan の「フラグなしの `--dry-run` の spawn の行」)。spawn は `--dry-run` でも見張りを要る(`spawn.go:608` の `checkWatchGate`)ので、一時の台帳に `ralph org watch --all --detach` を立てた。座席がないので herdr は呼ばれない。reviewer を `--driver` と `--model` なしで打った結果は次のとおり。
  - 既定: `driver=claude model=opus`、stderr は空。
  - `driver_pool = ["codex"]`: `driver=codex model=gpt-6.1-sol`。
  - `model_pool` を残した設定: `driver=claude model=opus` のまま、stderr に `org: <path> sets [org].model_pool, which ralph no longer reads: ... remove model_pool to drop this note` の 1 行。
  - `[org.roles]` を残した設定: rc 1、`rejected: org: <path> sets [org.roles], ...`、manifest の行数は変わらない。
  - `--model sonnet` だけ: `driver=claude model=sonnet`。
  - `--driver codex` だけ: rc 1、`org: role "reviewer" runs on claude in [org.role_models]; pass --model for --driver codex`、manifest の行数は変わらない。
- Probe 6: `[org.roles]` を残した設定で、seed した座席の台帳に `status`(rc 0)、`stop --dry-run`(rc 0、`stopped`)、`disband --dry-run`(rc 0、`disbanded`)。

### Coverage gaps(cycle 1)

- Go のテストの結果は /test の担当。この report の「Met」は、コードを読み、テストの assert を読んで、規則と合うことを確かめたもの。
- AC8 と spec AC 11 の「`[org.roles]` が残っても `status`・`stop`・`disband` が通る」を打つ Go のテストはない。Probe 6 は `--dry-run` の stop と disband で、herdr を呼ぶ本番の道は打っていない。設定を読むところは両方の道で同じで、`LegacyRolesErr` の呼び出し元は spawn と director start の道だけ。足すなら、Probe 6 と同じ形の CLI のテスト(`[org.roles]` を書いた `--config`、seed した座席、stub の herdr)が一番小さい。
- spec AC 2 は implementer の spawn を名指すが、CLI のテストは `driver_pool = ["codex"]` を reviewer の spawn と leader の start で打ち、implementer は config の層(`TestLoad_RoleModelsFollowTheDriverPoolInEffect`)で打つ。通る道は役割によらず同じ(`ResolveRoleModel` の map の引き)。
- 独自の役割にエントリを書き、フラグなしで立てる場合は、config の読み込み(`qa_bot`、`role_models_test.go:259`)と doctor(`doctor_org_test.go:1172`)にあり、spawn のテストはない。解決は map の引きで、組み込みの役割と同じ道を通る。
- leader の雛形の手順 2 の新しい文を固定するテストはない(計画の Progress が記録済み。手順 3 と 5 は `prompts_test.go:1442-1446` が固定する)。
- herdr が `--` のあとの argv を pane にどう渡すかと、`--model` を渡さない座席で alias のモデルが効くかは、実機で確かめていない(self-review と同じ範囲)。
- 自己レビューの残り(L4、L6、N1、N2)は tech-debt の 430 行に入っていることを確かめた。中身はこの report では評価し直していない。

### Verdict(cycle 1)

pass

- Verified: AC1〜AC12 の実装と、それを固定するテストの存在と assert の中身。AC13 の静的な半分(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、secret scan)。HEAD の binary の doctor の 4 通り、3 つの `--help`、`spawn --dry-run` の 6 通り、`[org.roles]` を残した設定の status・stop・disband。計画の digest は承認のときと同じ。
- Partially verified: AC8 の stop と disband は `--dry-run` の道だけを打った(Go のテストはない)。spec AC 2 の implementer は config の層で確認。
- Not verified: Go のテストの結果と `run-test.sh`(/test)。herdr の pane で `--model` なしの座席が動くこと。
- 所見: LOW 2 件(V1、V2)、INFO 5 件(I1〜I5)。どれも merge を止めない。
- /sync-docs に回すもの: V1(tech-debt の 135・188・190・223・414 行の carry-over の注)、V2(152 行の数)。I1 と I5 は任意。
