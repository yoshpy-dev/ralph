# Verify report: config-path-and-ledger-docs

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-config-path-and-ledger-docs.md`(tech-debt 421 行の (a)〜(e)、396 行の (c)、397 行の残り、AC1〜AC7、D1〜D3、Progress の S1〜S4、S1 の補い a6be3f5e、S4 の補い b16f4bd0、self-review の直し F1 d3bb5b55)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base = origin/main 350f22ea、HEAD ff0bf0b9、15 ファイル、+634/−55)。ea8b4d16 で origin/main を取り込んでいるので、branch の変更は三点の diff で見た。spec の準拠、静的解析、文書のずれを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。打ったのは静的なものと観察だけで、run-static-verify、check-sync と check-skill-sync の単独の実行、AC5 のテストを python に写したもの(HEAD と base)、base と HEAD の binary で設定の読み込みのエラーを出す probe、`ralph doctor` の出力の比較の 5 つ。
- Evidence: `docs/evidence/verify-2026-11-02-config-path-and-ledger-docs.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。run-static-verify と check-sync の全出力、AC5 の写し、grep の結果、binary の probe、doctor の比較、tech-debt の Trigger の照合)。run-verify 自身のログは `docs/evidence/verify-2026-11-02-072714.log`(同じく対象外)。
- 計画の digest: `./scripts/plan-visual.sh digest` は `917011181315` で、`- Approved:` の行と同じ。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(421 (a)(b)): root で壊れた `ralph.toml` のまま `start --plan` を打つと `org: load config: ralph.toml: ` で始まり何も作らない。`--config <file>` では `org: load config: <file>: `。spawn、`start <task>`、`director start` も同じ形。root のケースのテストがあり、元のコードでは落ちる。`config.Load`、`ralph config get`、`ralph doctor` の文は変わらない | Met(コード、テストの存在、base と HEAD の binary の probe で確認。テストの結果は /test) | 変わったのは `resolveConfig`(`internal/cli/org.go:661`)だけで、`loadConfigFile` のエラーを `fmt.Errorf("%s: %w", path, err)` で包む。`path` は `--config` の値か相対の `ralph.toml`。`loadConfigFile` を呼ぶ本番コードは `resolveConfig` と `loadMainWorktreeConfig` の 2 つ。`resolveConfig` を通るエラーの文を grep で数えると 5 つ: `org: load config: ` の 4 つ(`org.go:380` の `newOrgRuntimeAt`、`org.go:696` の `resolveOrgStartConfig`、`org_director.go:396`、`org_director_headless.go:305`)と `status.go:313` の warning。4 つの文はどれも自分ではファイルを名指さない。warning は a6be3f5e で `load %v` に直り、出力は base と 1 字も変わらない(probe)。`newOrgRuntimeAt` の呼び出し元(merge、cleanup、merge-order、watch、wait、inbox)はエラーを包まずに返し、escalate は「the item could not be recorded」で包むだけ(コードを読んで確認。escalate は人に知らせるので打っていない)。HEAD の binary で壊れた TOML を読ませると、start --plan(root)、spawn、director status、wait --inbox --director は `org: load config: ralph.toml: toml: ...`、start --plan、start <task>、director start に `--config` を渡すと `org: load config: <file>: toml: ...` になった。base の binary はどれも `org: load config: toml: ...` で、root のケースの期待(`org: load config: ralph.toml: `)に合わないので、テストは元のコードで落ちる。root のケースで repository に `.harness` はできず、台帳も空のまま(rc 1)。パスが 2 回出るのは、Progress が受け入れた形(読み込みのエラー、個人用だけのキー、個人用のファイルの誤りで 2 つのファイルを名指す形)だけで、ほかに 1 つ、`ralph.toml` がディレクトリのときの `ralph.toml: read ralph.toml: is a directory` がある(I1)。`internal/config/config.go` の diff は 0 行がコメント以外(全行が `//`)、`internal/cli/config.go` と `doctor.go` は diff なし。`ralph config get` の出力は base と HEAD で同じ(rc 3、`config get: ralph.toml: cannot be read or is invalid: ...`)、`ralph doctor` の全出力の diff は空。テスト: `TestOrgStartPlan_UnreadableConfigStartsNothing`(`org_start_plan_config_test.go:149`、root のケースを含む 3 ケース)、`TestOrgSpawnStartAndDirectorStart_NameTheConfigFileTheyCouldNotLoad`(`:214`、3 つの動詞 × 2 つの渡し方)、`TestResolveConfig_PrefixesTheFileItWasAskedToRead`(`:271`、読めないファイルと壊れた個人用のファイル)。 |
| AC2(421 (c)): main worktree に `ralph.toml` を commit する手順は helper 1 つで、同じ 2 行が繰り返されない | Met | helper は `commitRalphTomls`(`org_start_plan_config_test.go:62`)。このファイルで `commit.gpgsign` が出るのは helper の中の 1 か所だけで、呼び出しは 169・468・598 行の 3 つ。ファイルは 363 → 612 行。AC2 の対象はこのファイルだけ(計画の Scope)で、ほかのテストファイルの同じ組は I4 に書いた。 |
| AC3(421 (d)): 3 つのエラーの道のテスト、上限のテストの `max_total_seats` の対照(`--config` と外の台帳)、外の台帳で main の `[org.approval].mode` を使わないことを記録された方式で見るテスト | Met(テストの存在と中身で確認。結果は /test) | `newOrgSpawnRuntimeAt` が `newOrgRuntimeAt` のエラーを返す道と `newOrgRuntimeAt` の `org: load config:` の道は、AC1 のテストの spawn と start <task> のケースが打つ(spawn は `org.go:890`、start <task> は `org.go:1096` で `newOrgSpawnRuntimeAt` を呼び、エラーをそのまま返す)。`runOrgStartPlan` が `resolveOrgLedger` のエラーを返す道は `TestOrgStartPlan_RefusedLedgerStartsNothing`(`:526`): git の外で `RALPH_ORG_STATE_DIR` を別の場所に向け、exit 1、`errOrgLedgerRefusedByRetiredEnv`、拒否の文の接頭辞(`org_legacy_ledger.go:161` の文と同じ形)、設定ファイルを読んでいないこと、台帳ができないこと、herdr を呼ばないことを見る。上限のテスト(`:441`)は `--config, max_total_seats` と `a ledger outside the main worktree, max_total_seats` の 2 行を足した。main の値は拒否の行と同じ `max_total_seats = 1`、読むファイルは 99 で、`Default()` の `MaxOrgs` は 10(`config.go:773`)なので、2 つ目の org が上限の別の理由で断られることはない。承認方式は `TestOrgStartPlan_OutsideLedger_RecordsTheApprovalModeOfTheFileItRead`(`:572`)で、4 ケースとも main と読むファイルの方式が逆になり、binding の `Approval` を直接読む。`resolveModelOrWarn` の道は計画の Non-goals のとおり入っていない。 |
| AC4(396 (c)): `Load` の doc に doctor の理由句がなく、`StateDir` の doc は `LocalOrgStateDirAt` を名指す | Met | `Load` の doc(`config.go:879-887`)は「the Config is Default(), none of ralph.toml's values applied.」で終わり、doctor の句はない。主張は `return Default(), err`(`config.go:910`)と合う。`StateDir` の doc(`config.go:199-211`)は「org.ResolveOrgStateDir reads it with LocalOrgStateDirAt」と書き、`internal/org/statedir.go:80` の呼び出しと合う。「LocalOrgStateDirAt (and LocalOrgStateDir, which calls it)」は `internal/config/local.go:205-212` と合う。 |
| AC5(397): README の段落と `/org` skill の 4 写しが定数の文を含み、それを確かめるテストがあり、文を変えると落ちる。check-skill-sync と check-sync が通る | Met(テストの結果は /test) | テスト `TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs`(`internal/cli/org_retired_state_dir_condition_docs_test.go:21`)を python に写し、定数を `org_legacy_ledger.go` から組み立てて HEAD と origin/main で打った。HEAD: README(backtick を外して)と 4 写し(code span)のどれも含む。origin/main: 5 つとも含まない。だから README か skill の文を戻すとテストは落ちる。4 写しは同じ blob(650dabf3)。check-skill-sync(14 skill)と check-sync(DRIFTED 0)は単独でも run-static-verify の中でも通った。skill の「`ralph doctor` の warn と `ralph org escalate --help` は、この条件を英語でこう書く」は、HEAD の binary で確かめた: doctor の warn は定数の文をそのまま含み、escalate の help は折り返しを 1 つの空白にすると含む。定数の doc(`org_legacy_ledger.go:66-79`)は README と skill の 4 写しとテストを名指す(F1 の L1)。 |
| AC6(421 (e)): codex の model の実物の確かめと、leader の通し 2 つ | 未着手(/test の担当) | 計画の Scope と D3 のとおり /test が証跡を取る。 |
| AC7: tech-debt の 396・397・421 行の書き換え、`run-verify.sh` と `run-test.sh` が通る | 未着手(書き換えは次の段の /sync-docs、テストは /test) | `docs/tech-debt/README.md` はこの branch で変わっていない(origin/main と同じ bytes)。静的な側(`run-static-verify.sh`)はこのステップで通った。 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、"All verifiers passed.") | Requested scope は changed、Language scope は changed(changed_languages)で、golang pack を選んだ。 |
| gofmt / go vet / golangci-lint / staticcheck(golang pack) | pass | `gofmt: ok`、golangci-lint は `0 issues.`。go vet と staticcheck は何も出さない(通ったときの形)。`Skipping` の行はない。 |
| shellcheck(hook と verify の script)、`sh -n` の hook、Codex の inline hook の検出、PR provenance の guard | pass | |
| `scripts/check-sync.sh`(run-static-verify の中と単独の 2 回) | pass | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6。 |
| `scripts/check-skill-sync.sh`(run-static-verify の中と単独の 2 回) | pass | 14 skill が lock-step。 |
| `check-pipeline-sync.sh` / `check-template-purity.sh` / tech-debt の plan 参照 | pass | |
| branch secret scan | pass | 350f22ea..ff0bf0b9 は clean。 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md:148`(`[org].state_dir` の段落) | Yes | 条件の文は定数と同じ(AC5)。「a `ralph.toml` with an `[org].state_dir` fails to load with an error that names its path」は、org の動詞では同じパスを 2 回出す形になったが、文は今も正しい(self-review L4、I5)。 |
| README の org の動詞の設定エラーの説明 | Yes | README は `org: load config` の文も、org の動詞のエラーがファイルを名指すかどうかも書いていない(grep)。直す文はない。 |
| `/org` skill の 4 写し | Yes | 足したのは「前提」節の台帳の場所の項(124 行)で、動詞の表ではない。4 写しは同じ blob。 |
| `AGENTS.md:89`(`internal/config/` の行) | Yes | `local.go` が `LocalOnlyKeys`(`[org].state_dir` と script の 5 つ、`local.go:123-131`)と `LocalOrgStateDirAt`(`ResolveOrgStateDir` が使う)を持つ、という記述は HEAD と合う。この branch は `internal/config/` のコードを変えていない。 |
| `docs/specs/`、`.claude/rules/`、`.claude/skills/`(`/org` 以外) | Yes | `org: load config` と `load ralph.toml` を名指す文はない(grep。出るのは過去の evidence と計画だけ)。 |
| コードのコメント | Yes | `resolveConfig`、`statusOrgConfig`、定数の doc、`Load` と `StateDir` の doc は、それぞれのコードと合う。`resolveConfig` の doc の 2 回出る形の並びは網羅ではない(I1)。 |
| `docs/tech-debt/README.md` 396・397・421 行 | 未着手 | AC7 で /sync-docs が書き換える。行ごとの中身は Verdict の申し送りに書いた。 |
| `docs/tech-debt/README.md` 148・411・88 行 | No(V1) | 下の V1。Trigger の列を python で読み、取り消し線の中を除いて照合した。 |
| 計画の Acceptance criteria の checkbox | 遅れ(INFO、I6) | AC1〜AC7 はすべて `[ ]` のまま。 |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | 計画が挙げる 396・397・421 のほかに、この diff が Trigger に当たる tech-debt の行が 3 つある。148: Trigger「any change to `resolveOrgConfig`」。`resolveOrgConfig` の本体は同じだが、返すエラーが `resolveConfig` の変更で変わった(パスが前に付く)。前例として、feat/standard-flow-config が `resolveOrgConfig` を `resolveConfig` 経由に替えたとき(エラーは同じ)を「hit this trigger」と記録している。開いている部分(設定の探し方)はこの PR も触っていない。411: Trigger「The next change to any of the five」で、`internal/config/config.go` が 1192 → 1194 行(doc のコメントだけ。feat/org-cross-review のコメントだけの変更も記録されている)。88: Trigger「Next time the corresponding file is touched for any other reason」で、a6be3f5e が `internal/cli/status.go` を変えた。(4) の `printStateDirLine`(`status.go:503`、使うのは `:749` の 1 か所)と `state_dir_source` の `omitempty`(`:597`)は残る。88 は古い行で、これまで `status.go` を変えた PR も記録していない。 | evidence の log の「tech-debt Trigger sweep」と行数。self-review は 396・397・421 だけを挙げた。 | /sync-docs で、148 と 411 に日付つきで「当たったが払わなかった」とその理由を書く(この表のほかの行と同じ形)。88 は記録するか、今の運用のまま触らないかを /sync-docs が決める。 |
| I1 | INFO | `resolveConfig` の doc(`org.go:645-660`)が挙げるパスの出方は網羅ではない。ほかに 2 つある。`ralph.toml` がディレクトリのときは `ralph.toml: read ralph.toml: is a directory` で同じパスが 2 回出る(doc は「opening」と書くが、`os.ReadFile` の not-exist 以外のエラーはどれも同じ形)。個人用のファイルを重ねたあとの検証エラーは `ralph.toml: <検証の文> (with /…/ralph.local.toml applied)` で、個人用のファイルは末尾に出る。計画の Assumptions は「ReadFile のエラー」と一般に書いているので、計画どおり。doc は網羅を主張していない。 | evidence の log の probe (d) と (e)。`config.go:891-894`、`:936-938`。 | 直さなくてよい。直すなら「opening」を「opening or reading」に。 |
| I2 | INFO | `resolveConfig` を通らない main worktree の `ralph.toml` の文(`withMainWorktreeOrgLimits`、`withMainWorktreeDirectorEnabled`、`withMainWorktreeDirectorStartConfig`、`withMainWorktreeDirectorWakeConfigFrom`、`resolveOrgStartConfig` の 2 つ目の文)は、前から絶対パスを自分で名指し、`Load` のエラーを包まずに付けるので、読み込みのエラーと個人用だけのキーでは同じパスが 2 回出る。この PR は変えていない。 | `org.go:208`、`:251`、`:283`、`:327`、`:701` と `loadMainWorktreeConfig`(`:354-368`)。コードを読んで確認し、binary では打っていない。 | 直さなくてよい。421 を閉じるとき、2 回出る形の Known gap にこの文も同じ理由で入ると書くとよい。 |
| I3 | INFO | `ralph status` の warning は末尾に「because ralph.toml could not be read」を持つので、ファイルの名前は文の中に 2 度出る。`path: path` の形ではなく、base と bytes が同じ(probe)。 | `status.go:313-315`。 | 直さなくてよい。 |
| I4 | INFO | 計画の Progress(S2)は「同じ add と commit の 2 行がほかのテストファイル 5 つにある」と書くが、数え方で数が変わる。メッセージが `ralph.toml` の組は 3 ファイル 4 か所(`org_approve_test.go:681`、`org_feature_reviewer_test.go:42` と `:117`、`org_feature_test.go:463`)、`commit.gpgsign=false` を使うほかのテストファイルは 6 つ。self-review も同じことを書いている。 | evidence の log の「AC2」。 | /sync-docs が 421 の (c) を書くときは「`ralph.toml` を commit する組は 3 ファイル 4 か所」と書く。 |
| I5 | INFO | `[org].state_dir` を `ralph.toml` に書いた人が org の動詞で見るエラーは `org: load config: ralph.toml: ralph.toml: [org].state_dir can be set only in ...` になった(self-review L4、D1 が受け入れた代価)。テストで固定していない。 | probe (a)。 | 計画どおり PR の Known gaps と 421 の書き換えに書く。 |
| I6 | INFO | 計画の Acceptance criteria の checkbox が全部 `[ ]` のまま。 | 計画の 94-100 行。 | /pr の前に lead が付ける。AC6 は /test、AC7 は /sync-docs と /test のあと。 |

## Observational checks

- base(origin/main 350f22ea)と HEAD の binary を `go build -o <S>/bin-<x>/ralph ./cmd/ralph` で作った(`<S>` は scratchpad)。base は一時の detached worktree で build し、すぐ消した。
- scratchpad に `git init` した repository の root に、壊れた `ralph.toml` を commit して打った。AC1 の表の 7 つの打ち方(6 つの動詞、start --plan は root と `--config` の 2 通り)、`ralph config get org.max_seats`、`ralph status` の stderr を base と HEAD で並べた。org の動詞はどれも rc 1 で、台帳のディレクトリも `.harness` もできなかった。
- `config.Load` が自分でファイルを名指す 6 つの形((a) 個人用だけのキー、(b) 読めないファイル、(c) 壊れた個人用のファイル、(d) ディレクトリ、(e) 個人用のファイルを重ねたあとの検証エラー、(f) 検証エラー)を spawn で base と HEAD に出させた。HEAD はどれも `org: load config: ralph.toml: ` で始まる。
- 壊れた `ralph.toml` の repository で `ralph doctor` を base と HEAD で打ち、全出力の diff は空だった(`ralph.toml: warn — parse error: ... — using defaults`)。
- `RALPH_ORG_STATE_DIR` を設定して HEAD の `ralph doctor` を打ち、warn が定数の文をそのまま含むことを見た。`ralph org escalate --help` は折り返しを 1 つの空白にすると含む。
- 計画の digest は承認時と同じ `917011181315`。worktree と main checkout の `git status --porcelain` は、作業の前後とも空。

## Coverage gaps

- Go のテストの結果は見ていない(/test の担当)。AC1・AC3・AC5 のテストが通ること、S2 の変異 5 つと S4 の変異 3 つが落ちることは、計画の Progress の記録による。
- `org: load config: ` の 4 つ目の文(`org_director.go:396`、`director attach` と `director status`)は、パスを名指すことを見るテストがない(self-review の Coverage gaps)。このステップでは HEAD の binary の `director status` で見た。
- escalate のエラーと banner は、人に知らせる(desktop の通知)ので binary で打っていない。コードを読んで、パスを 1 度だけ名指すことを確かめた。
- AC6(実物の確かめ)は /test、AC7 の書き換えは /sync-docs、テストは /test の担当。

## Verdict

pass

- Verified: AC1(エラーの文の数え直し、binary の probe で 7 つの打ち方の新しい形と base の形、root のケースが元のコードで落ちること、何も作らないこと、`config.go` の diff がコメントだけ、`config get` と `doctor` の出力が base と同じ)、AC2(helper 1 つ、このファイルに同じ組がない)、AC3(3 つの道、上限の対照 2 つ、承認方式の 4 ケースのテストの中身)、AC4(2 つの doc とコードの一致)、AC5(テストを写して HEAD で含み base で含まない、4 写しが同じ blob、2 つの sync の検査)。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、secret scan)はすべて通った。計画の digest は承認時と同じ。
- Not verified: Go のテストの結果(/test)、AC6(/test)、AC7(/sync-docs と /test)。
- 所見: LOW 1 件(V1)、INFO 6 件(I1〜I6)。どれも merge を止めない。
- /sync-docs に回すもの:
  - AC7 の 396: (c) を 3311cf16 で閉じる(`Load` の doc の理由句を外し、`StateDir` の doc は `ResolveOrgStateDir` が `LocalOrgStateDirAt` で読むと書いた)。(a)(b) は閉じ済みなので、行全体を閉じる。
  - AC7 の 397: 「Left open」を d257824e(README の 1 語と skill の 4 写しの code span、`TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs`)、b16f4bd0(skill は doctor の warn と escalate の help の言葉だと書く)、d3bb5b55(定数の doc が README と skill を名指す)で閉じる。「written in five places」と、Trigger の「When the variable is dropped from `retiredEnvs`, remove the constant and all five」は、今の場所(定数、doctor の warn、escalate の help、README、skill の 4 写し)と新しいテストに合わせて直す。
  - AC7 の 421: (a)(b) を 40cedc9e と a6be3f5e で閉じる。残る 2 回出る形(読み込みのエラー、個人用だけのキー、I1 のディレクトリ、I2 の main worktree の文)は D1 の代価として書く(I5、self-review L4)。(c) は f185f22d の `commitRalphTomls` で閉じ、同じ組がほかに 3 ファイル 4 か所あることを書く(I4)。(d) は 3 つの道、`max_total_seats` の対照、承認方式のテストで閉じ、`resolveModelOrWarn` の道は org-feature-worktree の行の (e) が追うと書く。(e) は /test の AC6 の結果で書く(打てなかった場合は「確かめていない」として残す)。
  - V1: 148(`resolveOrgConfig` のエラーが変わった、開いている部分は触っていない)と 411(`config.go` 1192 → 1194、コメントだけ)に繰り越しの注。88 の (4) は書くか触らないかを決める。
  - I6: 計画の AC の checkbox は /pr の前に。
