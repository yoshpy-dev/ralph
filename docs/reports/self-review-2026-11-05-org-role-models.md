# Self-review report: org-role-models

- Date: 2026-11-05
- Plan: `docs/plans/active/2026-11-05-org-role-models.md`(spec `docs/specs/2026-11-05-org-role-models.md`、S1〜S6 と S5b、D1〜D9、AC1〜AC13)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1、Run 1 と Run 2)
- Scope: Run 1 は `git diff d631e9f8...HEAD`(base d631e9f8、HEAD d7f02479、78 ファイル、+5741 / -2438)。コードは S1 ffd614f8、S2 bf1f4727、S3 1bc70f0b、S4 39b50712、S5 14c77c43、S5b b23e6af5。文書は S6 0ee2bbaf と d7f02479。Run 2 は同じ base の HEAD e401d1a6(85 ファイル、+6207 / -2476。増えた分は Run 1 の report と insight event を入れた f32fa2ed と、修正の e401d1a6)。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。
- ID: Run 1 の所見は M1 と L1〜L6。コードのコメントが `self-review L2` のようにこれらの ID を引いているので、Run 2 でも番号は振り直さない。Run 2 の新しい所見は N1 と N2。
- 現在の状態(Run 2): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4(L4、L6、N1、N2)。M1・L1・L2・L3・L5 は e401d1a6 で解消した。

## Evidence reviewed

### Run 1

- `internal/config/role_models.go` と `internal/config/config.go` の `Load`・`validateLoaded`・`Default` の diff を全部読んだ。`checkRoleModelEntries` は pointer の probe でファイルごとにキーの有無を見る。役割は `slices.Sorted(maps.Keys(...))` の順で、エラーはファイルの path と役割を名指す。`reviewerWordPattern` の `^...$` は Go の既定(複数行でない)で文字列の終わりにだけ合うので、`"opus\n"` は通らない。`CheckOrgModelName` は `""` を通し、先頭の `-` を断る。
- `legacyModelKeys` は文書を `map[string]any` に読み、`checkLocalOnlyKeys` と同じ `tomlDocHasKey`(`internal/config/local.go:165`)で探す。大文字小文字と空の表も拾う。`Load` の 2 つのブロック(ralph.toml と ralph.local.toml)で呼ぶ順は「上書き → 個別の検査 → legacy の記録」で、どちらも `fillDefaultRoleModels` の前。
- `internal/org/role_models.go` の `ResolveRoleModel` を、doc の 4 つの規則と switch の 6 つの case を 1 対 1 で突き合わせた。`!hasEntry` は両フラグがある case のあとに置かれ、役割のない独自の役割でも両フラグがあれば通る。`fromEntry` は「driver を entry から取ったか」だけを見ていて、エラー文の出し分けに合う。`--driver` だけで別の driver、かつ entry の model が `""` のときは `""` になる。
- `config.Load` の呼び出し元 4 か所(`internal/cli/config.go:101`、`org.go:385`、`org.go:688`、`doctor.go:110`)を読み、エラーのときの `cfg`(`RoleModels` が空の map)を使う所がないことを確かめた。doctor は `doctor.go:115` で `config.Default()` に差し替える。
- `internal/org/spawn.go` の `--model` の分岐(`:1075`)、`spawn_started` の印(`:1463`、dry-run の `:2300`)、`codexFoundReceipt`(`:1990`)を読んだ。`EventSpawnStarted` を書く所は本番で 2 か所(`:1462` と `:2299`)で、どちらも印を付ける。`verbs.go` の `codexSpawnCorrelation` は dry-run を飛ばす(`:1749`、`:1772`)ので、dry-run の印が本番の spawn_started を押しのけない。`observeStopModelReceipt`(`:1899`)の条件 `corr.Model == "" && !corr.ModelCLIDefault` は、印のない古い event を今までどおり飛ばし、印のある event を観測する。観測済みの判定 `hasObservedCodexReceiptAfter` は `ReportedEffectiveModel != ""` を見るので、honored が `unknown` でも二重に足されない。
- `director_start.go`(`directorAgentArgs`、`checkDirectorStart`、`AttachHeadless`)、`director_cmd.go`(`directorLaunchScript.text()`)、`driver/probe.go` を読んだ。空のモデルで消えた検査は 3 か所(`AttachHeadless`、`checkDirectorStart`、`text()`)で、`text()` の NUL の検査(`:308`)は残る。起動スクリプトの全語は `shellQuote` を通る。herdr の agent start は `--` のあとに argv で渡す(`internal/org/driver/herdr.go:365-368`)。
- `internal/cli/org.go` の diff を全部読んだ(`withMainWorktreeOrgLimits`、`applyMainWorktreeOrgLimits`、`resolveOrgStartConfig`、`withMainWorktreeDirectorStartConfig`、spawn・start・start --plan の呼び出し、`printLegacyModelPoolNotice`、help の定数)。`org_director_headless.go` の `startDirector` の順(notice → enabled → agmsg_home → `LegacyRolesErr` → `ResolveRoleModel`)と、`--driver` が空のときだけ `checkOrgDirectorDriver` を飛ばす分岐を読んだ。
- 表示の箇所は、`.Model` を印字する非テストの Go を `grep -rn '\.Model\b' internal/cli internal/insights` で全部出して、`ModelOrCLIDefault` を通る所と通らない所を分けた(L1)。
- 残りの参照は `git grep` で追った。`model_pool`・`ModelPool`・`[org.roles]`・`resolveModelOrWarn`・`ModelAllowedForRole`・`DefaultModelForDriver`・`modelInPool`・`OrgModelPoolEntry` は、本番の Go で残るのが「残ったキーを扱うコード」と、それを説明するコメントだけ。「ralph は --model を常に渡す」の言い回しは `internal/cli/doctor_shell_alias.go` に 4 か所残っていた(M1)。
- 削除・改名されたテストは、`git diff | grep '^-func Test'` と `'^+func Test'` を `comm` で突き合わせ、名前が消えた 34 本を 1 本ずつ置き換え先に当てた(下の「削除されたテストの扱い」)。
- `docs/tech-debt/README.md` の diff は変わった行だけ読んだ(閉じた 3 行と、足した 2 行)。閉じ方は `RESOLVED-by-removal` の注記で、直した commit と理由が書いてある。
- diff 全体の U+FFFD、TODO、debug の出力、秘密の値は見つからなかった。

### Run 2

- e401d1a6 の diff を全部読んだ(13 ファイル、+409 / -64)。コードは 7 ファイル(`doctor_shell_alias.go`、`org.go`、`org_director_report.go`、`status.go`、`director_start.go`、`role_models.go`、`spawn.go`)、テストは 6 ファイル。消えたテストはなく(`git show e401d1a6 | grep -c '^-func Test'` が 0)、足したテストは 5 本。
- M1: `grep -rnE 'always (passes?|supplies|sends?) .{0,20}--model|every spawn|spawn\.go:[0-9]+'` を `internal/`、`docs/recipes`、skill、rule、README、AGENTS.md に打った。非テストの Go に残る `always` は `doctor_shell_alias.go` の「a claude --model finding is always informational」(重大度の話で、`--model` を渡す回数の話ではない)だけ。Run 1 が挙げた 4 か所(warn の文 2、分類のコメント 2)は条件付きの文に変わり、古い行番号 `spawn.go:643` は関数名(`Spawn` の `agentArgs`、`directorAgentArgs`)に変わった。新しい文の根拠は読んで確かめた。seat は `Spawn` の `if p.Model != ""`、director は `directorAgentArgs` の `if model != ""`、launch_cmd の director は `directorLaunchScript.text()` が agent を絶対パスで `exec` する(pane の shell を通らない)。herdr の pane が alias を展開するかは、Run 1 と同じく herdr 側を見ていない。
- M1 のテスト: 新しい warn の文字列と、`TestCheckShellAliases_CodexModelOnly_...` と `..._ClaudeModelOnly_...` が `strings.Contains` で探す部分文字列を、raw string の連結を含めて目で突き合わせた。一致する。古い文言(`always passes --model`、`every spawn fails`)が出ないことも assert している。
- L1: `.Model` を `Fprint`/`Sprintf` で印字する非テストの Go を `internal/cli`・`internal/insights`・`internal/org` で再び出した。`ralph org status` の表(`org.go` の `printStatusTable`)、`ralph status` の座席の表(`status.go`)、`director status` の `model:`(`headless` のときだけ `ModelOrCLIDefault`)は通るようになった。`CommandedModel` を印字する所も辿った。`report.go` は通り、`org.go` の codex mismatch の warn は honored が `false` のときだけで(モデルなしの座席は `unknown`)問題ない。`internal/cli/insights.go` は通らない(N1)。
- L1 のテスト: `org_model_display_test.go` は text の表 2 つと `director status` の 4 通り(herdr と cmd の空のモデル、herdr のモデル付き、昇格)、`--json` が `""` のままで `(CLI default)` を含まないこと、モデルなしの印がある event と印のない古い event が同じ表示になることまで固定している。`status_followup_test.go` の期待値の変更(`leader` の行が `(CLI default)` になる)は、この表示の変更に合う。
- L2: `AttachHeadless` と `checkDirectorStart` の検査の順は、driver → モデルの文字 →(wake・permission)で、doc の列挙の順と合う。エラー文は `org: director: start: model "..." must be ""...` で、`CheckOrgModelName` の文に接頭辞だけを足す(`org:` が二重にならない)。launch_cmd のときもモデルの検査が agent と ralph の探索より前であることを、テスト(`wakeTestLaunchCmd` が探索を失敗にする)が固定している。`SpawnParams.Model`・`DirectorStartParams.Model`・`DirectorHeadlessAttach.Model` の 3 つのフィールドに `""` の意味が付いた。
- L3: `TestDefaultRoleModels_KeysAreTheOrgRoles` は `config.Default()` と `DefaultRoleModels` の 3 通りの入力のキーを、4 つの定数(`LeaderIdentity`、`implementerRole`、`reviewerRole`、`DirectorRole`)から作った一覧と突き合わせる。右辺が定数なので、定数の値を変えると落ちる(自分の写しとの比較ではない)。`TestResolveRoleModel_DefaultsOfConfigDefault` の `"reviewer"` も `reviewerRole` になった。
- L5: `legacyKeyFilesSet` は 1 件で `<path> sets`、2 件以上で `<a> and <b> set`。`legacyKeyPaths` が `slices.Compact` するので、同じ path が続いても 1 件扱い。呼び出し元の `LegacyRolesErr`・`LegacyModelPoolNotice` は path が 0 件のとき先に返るので、0 件の分岐は実行されない。テストが 1 件と 2 件の接頭辞を固定する。doctor の `checkOrgLegacyModelKeys` は 1 ファイル 1 行なので影響を受けない。
- e401d1a6 が持ち込む問題の確認: 触った非テストの関数の doc comment を body と突き合わせた(`printStatusTable`、`printOrgDirectorReport`、`StartDirector`、`AttachHeadless`、`ModelOrCLIDefault`、`checkShellAliases`)。矛盾はなかった。折り返しが不揃いな所が 3 か所あるが、読みには影響しない(`checkShellAliases` の `directorAgentArgs),` だけの行、`printOrgDirectorReport` の `and self-review L1). wake_target follows` の行、`StartDirector` の doc の途中の短い行)。
- Run 1 が読まなかったか薄かった所: `doctor.go`、`doctor_codex_models.go`、`doctor_codex_writable_root.go`、`init_settings.go` の diff 全部、`config.go` の `Load`・`validateLoaded`、`driver/probe.go`、`envelope_summary.go`、`report.go`、`manifest.go`、`prompts.go`、`internal/org/prompts/leader.md`、`templates/base/ralph.toml`、`/org` skill の追加行を読んだ。N1 と N2 はここから出た。`checkOrgModelProbes` の重複排除と sort、`probeOrgModel` の `(CLI default)` の名前、`ProbeModel` の `slices.Concat(nil, ...)`、`codexSeatPossible` の導入で振り直した outcome の番号(コメント内の参照 `(8)` も合う)に問題はなかった。
- `docs/tech-debt/README.md`: e401d1a6 は触っていない。M1 の修正で嘘になる行(`checkShellAliases` が `ralph.toml` を読まない件の行)は、「flag class だけで格付けする」と言うだけで、`--model` の回数には触れていないので、そのまま有効。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

### Run 1 の所見の状態

| ID | Run 1 の重大度 | 内容 | Run 2 | 根拠 |
| --- | --- | --- | --- | --- |
| M1 | MEDIUM | doctor の「Shell aliases」が「`--model` を常に渡す」と言う | 解消 | e401d1a6。codex の文と claude の文が「モデルが `""` でない座席と herdr の director にだけ渡す。`""` なら渡さないので alias のモデルが効く」に変わり、重大度のコメントと分類のコメントも同じ前提に直った。行番号の pointer は関数名になった。テストが新しい文と、古い文言が出ないことを固定する。recipe の文言(`docs/recipes/codex-seat-permissions.md`)と doctor の出力は同じことを言う。codex の `--model` の finding は warn のまま(組み込みの leader・implementer・reviewer が `""` でないので妥当)。 |
| L1 | LOW | `(CLI default)` の表示が箇所で揃わない | 解消(同じ型の見落としが N1) | e401d1a6。`org status` の表、`ralph status` の座席の表、`director status` の `model:`(headless のとき)が `ModelOrCLIDefault` を通る。`--json` は `""` のまま。古い event の座席が同じ表示になる件は `ModelOrCLIDefault` の doc に書いた(推奨どおり)。 |
| L2 | LOW | director の model の文字の検査が CLI にしかない | 解消 | e401d1a6。`checkDirectorStart` と `AttachHeadless` が `config.CheckOrgModelName` を呼ぶ。何も書く前、herdr を打つ前、launch_cmd の探索の前。3 つの `Model` フィールドの doc に `""` の意味が付いた。 |
| L3 | LOW | 役割名の綴りを結ぶテストが 2 本分しかない | 解消 | e401d1a6。`TestDefaultRoleModels_KeysAreTheOrgRoles`(`internal/org/role_models_test.go`)。 |
| L5 | LOW | 2 ファイルのとき `sets` の文法が合わない | 解消 | e401d1a6。`legacyKeyFilesSet` と、1 件・2 件の接頭辞を固定するテスト 2 本。 |
| L4 | LOW | 空の `[org.roles]` の表も断られる | 残る(orchestrator の判断で直さない) | 下の表。spec FR-6 の字面(「`[org.roles]` が残っていれば断る」)に合う挙動で、バグではない。文言だけが残る。 |
| L6 | LOW | 同じ文の複製 | 残る(orchestrator の判断で直さない) | 下の表。 |

### 現在の所見(Run 2 で残るもの)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L4: 空の `[org.roles]` の表(と `reviewer = []` のような空の一覧)も「残ったキー」として spawn・start・director start を断る。`legacyModelKeys` の doc と `TestLoad_LegacyModelKeysRecordsEachFile` の「an empty roles table」の行が意図して拾っているが、この PR が消した `TestLoad_OrgRolesEmpty` は「空の `[org.roles]` は有効で、制限なしの意味」だった。空の表は何も制限していなかったので、断る根拠の「制限が黙って外れる」(spec FR-6、`LegacyRolesErr` の文)が当たらない。エラー文は「`no longer restricts a role's models`」と言い、空の表を持つ人には何を失うのか分からない。旧テンプレートは `[org.roles]` をコメントでしか出さなかった(diff の `templates/base/ralph.toml`)ので、空の表を生で持つ下流は多くないはず。確かめてはいない。**Run 2**: orchestrator は FR-6 の「消す」を理由に直さない。FR-6 の字面は空の表を区別していないので、挙動は spec に合う。残るのはエラー文が空の表の人に当たらない点だけで、害は小さい。 | `internal/config/role_models.go:161-182`(`tomlDocHasKey(doc, "org.roles")` はキーがあれば真)。`internal/config/role_models_test.go:402-470` の「an empty roles table, no ralph.toml」。`internal/org/role_models.go` の `LegacyRolesErr`。削除された `TestLoad_OrgRolesEmpty`(`git show d631e9f8:internal/config/config_test.go`)。 | 変えないなら、release ノートの R2 の文に「空の `[org.roles]` も断られる」と足す。エラー文に「空の表でも断る」理由を 1 句入れるのは任意。変えるなら、`legacyModelKeys` が「`[org.roles]` に 1 つ以上の非空の一覧がある」ときだけ記録する。 |
| LOW | maintainability | L6: 同じ文が 2 か所に写されている。(a) 「driver が `driver_pool` にない」の文は `ResolveRoleModel` の `--driver` 側(`org: driver %q not in [org].driver_pool %v`)と `ValidateSpawnEnvelope` が同じ文字列で持ち、`ResolveRoleModel` の doc が「`ValidateSpawnEnvelope` が使う言い回し」と言うだけでつながる。片方を直すともう片方が違う文を出す。(b) 残ったキーの文は `LegacyRolesErr`・`LegacyModelPoolNotice`・`checkOrgLegacyModelKeys` の 3 か所と、`orgRoleModelsHelp` の 1 段落に別々に書かれている(関数は 3 つ、見る人は同じ)。**Run 2**: 直さない判断(cosmetic)。e401d1a6 は (b) の `LegacyRolesErr`/`LegacyModelPoolNotice` の「sets/set」だけを `legacyKeyFilesSet` に寄せたので、(b) の複製は 3 か所から変わっていない。 | (a) `internal/org/envelope.go:59`、`internal/org/role_models.go` の `ResolveRoleModel` 末尾。(b) `internal/org/role_models.go` の `LegacyRolesErr`・`LegacyModelPoolNotice`、`internal/cli/doctor.go` の `checkOrgLegacyModelKeys`、`internal/cli/org.go` の `orgRoleModelsHelp`。 | (a) `errDriverNotInPool(cfg, driver)` のような 1 つの関数にして両方が呼ぶ。(b) doctor は per-file の文なので別でよいが、`LegacyRolesErr`/`LegacyModelPoolNotice` の「消す・移す」の部分は、doctor の detail が呼べる定数か関数にする。 |
| LOW | readability | N1: `ralph insights` の Receipts 節が、`--model` なしで立てた座席を `commanded=` と空欄で出す。L1 と同じ型の、表示の揃え直しが届かなかった箇所。`internal/insights/aggregate.go` は `CommandedModel != ""` のものだけを集合に入れる(空は捨てる)ので、モデルなしの座席の `CommandedModels` は空になり、`internal/cli/insights.go` が `strings.Join(s.CommandedModels, ",")` をそのまま `commanded=%s` に入れる。spawn の行、`ralph org report`、`org status` は同じ座席を `(CLI default)` と出す。Run 1 の L1 は `.Model` を印字する所を探した grep で、receipt の `CommandedModel(s)` を印字する所は見ていなかった。再 spawn で同じ座席 ID に空の receipt と `opus` の receipt が混ざることがあれば(leader の手順 14 は reviewer を同じ ID で spawn し直す)、`aggregate.go` が空を捨てるので `commanded=opus` だけが出て、CLI の既定で立てた回が見えなくなる。集計と receipt の項目は plan の Non-goals なので、直すのは text の 1 行の表示だけでよい。 | `internal/insights/aggregate.go:262`(`if r.CommandedModel != ""`)。`internal/cli/insights.go:226`(`commanded := strings.Join(s.CommandedModels, ",")`)と、続く `commanded=%s` の Fprintf。比較: `internal/org/report.go:238`(`ModelOrCLIDefault(r.CommandedModel)`)。receipt の書き手は `CommandedModel: p.Model`(`internal/org/spawn.go` の 3 か所、`verbs.go` の stop の観測)で、モデルなしなら `""`。読んだだけで、`ralph insights` は動かしていない。 | `insights.go` の `commanded` が `""` のときに `org.CLIDefaultModelText` を入れる(同ファイルは `internal/org` を import 済み)。`--json` の `commanded_models` と `aggregate.go` は変えない。混ざる件まで直すには集計の変更が要り Non-goals に当たるので、LOW のまま触らないでよい。 |
| LOW | naming | N2: e401d1a6 が足した ID の引用に、plan を名指さないものが 2 つある。`self-review L<n>` は、このリポジトリで別の plan の report の ID も指す(`doctor_shell_alias.go` の `self-review L7`、`doctor_shell_alias_test.go` の `self-review L1 fix` と `L2(c) fix` は、この PR の touched ファイルにある別の plan の ID)。plan を名指さない `(self-review L5)` は `git grep` で他の plan の L5 と区別できない。同じファイルの 148 行目は `self-review L3 of plan 2026-11-05-org-role-models` と書いていて、そろっていない。この report は Run ごとに in place で書き換えるので、ID が指す先は番号を振り直さない限り変わらない(Run 2 では振り直していない)が、plan がなければ検索しても行き着かない。 | `internal/org/role_models_test.go:184` と `:224`(`(self-review L5)`)。同ファイル `:148`(plan 付き)。`internal/cli/org_director_report.go` の `D5 and self-review L1` は直前に plan 名があり、ここでは足りている。 | 2 か所に `of plan 2026-11-05-org-role-models` を足す。 |

### 依頼された点の判断

1. `internal/config/role_models.go`: ファイルごとの検査は D3 のとおりで、表見出しと inline の両方に効く(`probe` が pointer なので `model` だけの表見出しは `Model == nil` で落ちる)。`driver` の値と `model` の文字の検査は `driver_pool` に触れず(D9)、`Load` の後の層に任せている。legacy キーの検出は大文字小文字と形(配列の表、文字列、空の表)を問わず拾い、エラーにしない。拾いすぎる側が 1 つあり、空の表(L4)。既定の埋めは両方のファイルを重ねたあとで、`driver_pool` の最終値に従う。`Default()` は claude の組、`Load` が表を空の map から始めるので、`driver_pool = ["codex"]` の文書で claude の entry が残らない。`DefaultRoleModels` は呼ぶたびに新しい map を返し、`Default()` と `Load` でその map が共有されることはない。役割の名前の綴りは、Run 1 では org の定数と結びついていなかったが(L3)、e401d1a6 のテストで結びついた。
2. `internal/org/role_models.go` の `ResolveRoleModel`: 規則は doc の 4 つと実装が合う。エラー文は 3 通りで、(a) entry なしは役割と `[org.role_models].<role>` の書き方まで案内、(b) 別の driver で model が必要なときは「`--model` を渡す」と名指し、(c) driver が `driver_pool` にないときは entry 由来と `--driver` 由来で文を分ける。(c) の `--driver` 由来の文は `ValidateSpawnEnvelope` と同じ字面の複製(L6)。`--model ""` の明示は省略と同じ扱いで、plan の edge case に書いてある。
3. `spawn.go`・`verbs.go`・`director_start.go`・`director_cmd.go` の空のモデル: argv は 3 か所すべてで `--model` を条件付きにしている(seat の `agentArgs`、herdr の director、起動スクリプトは `directorAgentArgs` を共有)。`spawn_started` の印は本番と dry-run の 2 か所の writer の両方が付ける。stop のときの観測は印のある event だけ広げ、印のない古い event は今までどおり飛ばす。`director` の登録は空のモデルを受け付け、`json:"model,omitempty"` で event に出ない。director の登録を読む側は、モデルが空であることで headless かどうかを決めていない(`Headless()` は `Launch` で決まる)。
4. `internal/cli/org.go`: spawn・start・start --plan・director start の 4 つとも `ResolveRoleModel` を `rt.Spawn`/`StartFeature`/`StartDirector` の前に呼び、解決の失敗は何も書く前に返る。start の `--driver` の既定を `""` にしたので、help の既定値の文言も変えてある。main worktree の上書きは `RoleModels`・`DriverPool`・`LegacyModelKeys` を丸ごと置き換える(map の共有はあるが、`Load` のあとに書き換える所はない)。`model_pool` の notice は spawn・start `<task>`・start --plan・director start の 4 か所で、config を決めたあとに 1 回だけ出る。
5. 語として herdr の pane と起動スクリプトに届く値: `--model` は `ResolveRoleModel` と `ValidateSpawnEnvelope`(seat)で `^[A-Za-z0-9._:@/+-]+$` かつ先頭が `-` でないことを確かめてから届く。`driver` は `claude`/`codex` の値に限られる。director のモデルは、Run 1 では CLI でだけ確かめていたが(L2)、e401d1a6 で `StartDirector` と `AttachHeadless` も確かめる。起動スクリプトは全語を単引用符で包み、NUL を断る。

#### Run 2 で依頼された点

6. e401d1a6 は M1・L1・L2・L3・L5 を解消するか: する。上の「Run 1 の所見の状態」の表に、1 件ずつ根拠を書いた。推奨した範囲を超える変更はなく、推奨から外れたのは L1 の `report.go` の古い manifest の件を直さず doc に書いたことだけで、これは推奨どおり。
7. e401d1a6 が新しい問題を持ち込むか: 見つからなかった。コメントの折り返しの不揃いがあるだけ(Evidence の Run 2)。
8. Run 1 の見落とし: N1(insights の `commanded=`)と N2(ID の引用の plan)。どちらも LOW。ほかに、`doctor.go` から `leader.md`、`templates/base/ralph.toml` までの diff に指摘はなかった。

### 削除されたテストの扱い

plan の Test plan が「消すものは self-review の report に理由を書く」と求めた分。`git diff` で名前が消えた 34 本を置き換え先に当てた。e401d1a6 はテストを消していない(`TestCheckShellAliases_*` と `TestShellAliasCodexSentence` は期待する文言を新しい文に差し替えただけで、`..._CodexModelOnly_...` と `..._ClaudeModelOnly_...` には古い文言が出ないことの assert を足した。弱めた assert はない)。

- 改名で残るもの(意図は同じ): `TestOrgSpawn_Rejection_ModelOutOfPoolAndMaxSeatsWithOrgIsolation` → `..._ScopeGateAndMaxSeatsWithOrgIsolation`、`TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel` → `..._ConfigFlagRecoversFromARalphTomlThatFailsToLoad`(plan が名指した model_pool の復旧テストの置き換え)、`TestOrgStart_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry` → `..._TakesTheLeaderRoleModel`、`TestOrgStart_NoMatchingModelPoolEntryForDriver_NonZeroExit` → `..._DriverOtherThanTheLeaderEntry_NonZeroExit`、`TestOrgSpawn_RetiredRoleOrID_RefusedBeforeModelFallback` → `..._RefusedBeforeResolvingTheRoleModel`、`TestOrgSpawn_MainWorktreeRalphToml_OnlyOrgWideLimitsTaken` → `..._OrgWideValuesTakenMaxSeatsNot`、`TestEnvelopeSummary_*` の 2 本 → `..._ListsRoleModelsMaxSeatsAndPermissionDefault`・`..._EmptyRoleModels_NoneConfiguredMarker`、`TestCheckOrgEnvelope_ReportsPoolSizeAndMaxSeats` → `TestDoctorCheckOrgEnvelope_ReportsRoleModelsAndMaxSeats`、`TestOrgSpawn_Rejected_OutOfPoolModel` → `TestOrgSpawn_Rejected_ModelNameNotOneWord`、`TestOrgSpawn_ModelFlagOmitted_RespectsRoleAllowlist` → `TestOrgSpawn_LegacyOrgRolesRefused`(意図が「許可リストを守る」から「`[org.roles]` が残れば断る」に変わった)。
- 機能ごと消えたもの(置き換えなし、理由は対象の関数・キーがなくなったこと): `TestDefaultModelForDriver_*` 4 本と `TestDefaultModelForDriverAndRole_*` 3 本(`DefaultModelForDriver`/`DefaultModelForDriverAndRole` を削除)、`TestCodexModelPermittedForRole` と `TestCheckCodexAgmsgWritableRoot_{NoCodexModelInPool_Pass,RoleModelRestrictionExcludesCodex_ReasonAAbsent,RoleModelRestrictionIncludesCodex_ReasonAPresent}`(条件が `driver_pool` に codex があるかだけになった。`TestCodexSeatPossible` と `TestCheckCodexAgmsgWritableRoot_CodexInDriverPoolWithoutCodexRoleEntry_StillChecked` が代わり)、`TestInitSettings_ModelPoolProblemsAreErrorsForOneCLI`(`model_pool` の書き換えを消した)、`TestLoad_{DriverPoolOnlyOverride_*,ExplicitModelPoolStillStrict,LocalPoolKeys,OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit,OrgRejects,OrgRolesEmpty}`、`TestOrgSpawn_ModelFlagOmitted_{DefaultsToFirstMatchingPoolEntry,NoPermittedModel_NonZeroExit}`(`model_pool` を読まなくなった)。`TestLoad_OrgRejects` が持っていた「driver が `driver_pool` にない」の観点は、`TestLoad_RoleModelsDriverOutsideDriverPoolLoads` と `TestOrgSpawn_RoleModels_RefusedWithNothingRecorded`(読み込みは通り、spawn が断る)に移った。`TestLoad_OrgRolesEmpty` は L4 の対象。

## Positive notes

- 空のモデルの扱いが、座席の argv、headless の director の argv、起動スクリプトの 3 つで 1 つの条件に寄っている(`directorAgentArgs`)。起動スクリプトは director の herdr の argv と同じ関数から語を作るので、2 つの入口の語がずれにくい。
- `spawn_started` の印は、「モデルを指示しなかった」と「`Model` の項目がなかった古い event」を分けるための足し方になっていて、additive(`omitempty`)。stop のときの観測は、印のない古い event を今までどおり飛ばすので、既存の manifest の読み方が変わらない。
- 読み込みは止めず、spawn・start・director start だけが断る形(D1・D7)で、`status`・`stop`・`disband` は動く。`[org.roles]` の拒否は idempotent の early return のあと(本番)と先頭(dry-run)に置かれ、`retiredRoleConfigErr` の既存の順と合う。
- `driver_pool` の検査を `validateLoaded` に足し直した S5b は、`model_pool` を消すと間接的な検査(空、`claude`・`codex` 以外)が一緒に消える点を拾っている。マージ後の値を見るので、`ralph.local.toml` が `ralph.toml` の誤りを直せる。
- 文字の検査の共有(D8)は、`reviewerWordPattern` を再利用し、CLI の `ResolveRoleModel` と `ValidateSpawnEnvelope` が同じ `CheckOrgModelName` を呼ぶ形。`model = ""` の通し方(「CLI の既定」)が 1 か所で決まる。e401d1a6 で director の 2 つの入口も同じ関数を呼ぶ。
- 新しい doc comment の根拠が具体的(go-toml の「表見出しは key ごとにマージ、inline は丸ごと置き換え」を `checkRoleModelEntries` の理由として書いている)。
- e401d1a6 のテストは、新しい文言を探すだけでなく、古い文言が出ないこと(`always passes --model`、`every spawn fails`)、`--json` が `(CLI default)` を含まないこと、L3 の右辺が定数であることを assert している。変えた挙動を戻すと落ちる形になっている。
- 秘密、debug の残り、範囲外のファイル、U+FFFD は見つからなかった(Run 1 と e401d1a6 の両方)。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(今の時点では L4・L6・N1・N2 はこの PR の中で直せる範囲) | - | - | - | - |

_(cycle 1 の Run 2 なので、推奨はまだ cycle 内の直し。L4 と L6 は orchestrator が直さないと決めたが、cycle の上限(既定 2)に届くまでに他の所見と一緒に残ったものは、最後の cycle で defer になる。そのとき `docs/tech-debt/README.md` に 1 行にまとめる案: 「org-role-models の LOW。(a) 空の `[org.roles]` の表も spawn・start・director start を断り、エラー文は何を失うか言わない(spec FR-6 の字面どおり、`internal/org/role_models.go` の `LegacyRolesErr`)。(b) 『driver が `driver_pool` にない』の文が `ResolveRoleModel` と `ValidateSpawnEnvelope` に、残ったキーの文が `LegacyRolesErr`・`LegacyModelPoolNotice`・`checkOrgLegacyModelKeys`・`orgRoleModelsHelp` に別々にある。(c) N1 と N2 を直さずに残した場合はここに足す。Trigger: 次にそれらの文を変えるとき」。この PR が足した `docs/tech-debt/README.md` の 2 行(`/org` skill の表と `[reviewer].codex_model` の slug)は、この report の所見とは別の既知の穴で、そのまま有効。)_

## Coverage gaps

- Go のテスト、`go vet`、静的解析は打っていない(/verify と /test の担当)。import が残っているかは、各スライスの検証が通ったという plan の記録を信頼した。ただし `errors`・`strings` は diff で触った 5 つの Go ファイルでまだ使われていることを `grep` で確かめた。e401d1a6 のテストが足した import(`maps`、`slices`、`errors`、`io/fs`、`os`、`strconv`)は、使われていることを目で確かめたが、ビルドはしていない。
- go-toml の「表見出しはマージ、inline は置き換え」は、plan の Assumptions にある 2026-11-05 の一時モジュールでの確認を信頼した。この review では動かしていない。
- `skills`・`README.md`・recipes・`docs/specs/` の改訂は、diff の質(M1 との矛盾)の範囲でしか読んでいない。`/org` skill は追加行を読んだが、文書の drift は /verify の担当。
- herdr が `--` のあとの argv をどう pane に渡すかは、`driver/herdr.go` の組み立て(`:365-368`)までしか見ていない。herdr 側が shell の文字列に組み直すかは確かめていない。`CheckOrgModelName` が通す文字(`A-Za-z0-9._:@/+-`)はどちらでも安全。同じ理由で、M1 の新しい文が言う「herdr の director は pane の shell の alias を通る」は、seat の既存の文言(`herdr expands the alias in the seat's pane`)と同じ前提に立つ。
- N1 は `ralph insights` を動かさず、集計と表示のコードを読んで判断した。
- plan の Progress checklist に e401d1a6 の行がない。記録の話で、この review の範囲ではない(/verify か orchestrator)。

## Recommendation

- Merge: yes。CRITICAL と HIGH と MEDIUM はない。
- Follow-ups: M1・L1・L2・L3・L5 は e401d1a6 で解消し、新しい問題は持ち込まれていない。残る LOW は L4・L6(orchestrator の判断で直さない)と N1・N2(Run 2 の新しい所見)。N1 は 1 行、N2 は 2 か所のコメントで、直すなら cycle 内で足りる。コードを変えるので、直したら /self-review から全パイプラインを回す(post-implementation-pipeline.md)。直さずに進めるなら、最後の cycle の `/pr` の前に、上の案の tech-debt の行を足す。直すのが記録だけ(この report の文言など)なら、同ファイルの例外の 3 条件を確かめる。
