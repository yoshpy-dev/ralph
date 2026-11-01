# Verify report: standard-config-gaps

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-standard-config-gaps.md`(PR #291 の Known gaps と tech-debt 406・409・410 行、AC1〜AC6、D1〜D5、Progress の S1〜S3 と self-review の直し a12a08a5)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base = origin/main d217c575、HEAD cfe92c27、8 ファイル、+758/−231)。spec の準拠、静的解析、文書のずれを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。打った shell のテストは、lead が許した `zsh -f tests/test-ralph-config.sh` の 1 本だけ。probe は HEAD と origin/main(`git archive` から build)の binary で、scratchpad の一時の git repo に対して打った。
- Evidence: `docs/evidence/verify-2026-11-02-standard-config-gaps.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。run-static-verify の全出力、`zsh -f` の実行、shellcheck、binary の probe、S1 の突き合わせ、呼び出し元の数)。run-verify 自身のログは `docs/evidence/verify-2026-11-01-165649.log`(同じく対象外)。
- 計画の digest: `./scripts/plan-visual.sh digest` は `11ccf2ed5a4c` で、`- Approved:` の行と同じ。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(406): `org_feature_test.go` は 800 行以下、3 本は `org_feature_reviewer_test.go` に中身を変えずに移る | Met | `wc -l` で `org_feature_test.go` は 647 行、`org_feature_reviewer_test.go` は 182 行。b71354d0 で消えた行(import と空行を除く)と新しいファイルの本文(空行と閉じ括弧を除く)を `diff` で比べ、差はない。`internal/cli` の `func Test` の名前は base 976、HEAD 979 で、増えたのは S2 の 3 本だけ(`comm -3`)。`go test -list` は /test の担当なので打っていない。 |
| AC2(409): main worktree の `ralph.toml` を main の上書きとして 1 回、`--config` か `./ralph.toml` を 1 回読む。`--config` や外の台帳では main を読まない。値は今と同じ。`InboxHintStateDir` を両方の経路で設定する | Met(コードとテストの存在で確認。テストの結果は /test) | `runOrgStartPlan`(`internal/cli/org.go:1204-1239`)は `resolveOrgStartConfig`(`:702-716`)を 1 回呼ぶ。中で `resolveConfig`(`:703`、cwd か `--config`)を 1 回、`loadMainWorktreeConfig`(`:707`)を 1 回呼び、後者は `configPath != ""` か台帳が main のものでないとき読まずに `ok=false` を返す(`:356-362`)。main の `[org]` は `applyMainWorktreeOrgLimits`(`:221-225`)で当て、`[reviewer]` は同じ `mainCfg` から取る。この経路に設定を読むほかの呼び出しはない(`wireOrgStartWatchGate` の `orgWatchChild` は `rt.Config.AgmsgHome` を受け取るだけで、`resolveConfig`・`resolveOrgConfig`・`loadMainWorktree*` を呼ばない)。旧コードと値を比べた: cwd の `[org]` に main の `MaxOrgs`・`MaxTotalSeats`・`Approval` を当て、`[reviewer]` は main が当てはまれば main、なければ cwd から取る点は同じ。spawn と `start <task>` は `newOrgSpawnRuntimeAt` のまま(`:899`、`:1105`)で、読む順と文は前と同じ。`withMainWorktreeOrgLimits` は `applyMainWorktreeOrgLimits` を呼ぶだけになり、当てる 3 つの値は同じ。`newOrgRuntimeAt` のテスト以外の呼び出し元は base も HEAD も 10 か所で、変わっていない。`InboxHintStateDir` は `:172`(spawn)と `:1217`(start --plan)で設定され、`StartFeature` の事前検査(`internal/org/feature.go:238`)がそれを使う。数えるテストは `TestOrgStartPlan_ReadsEachConfigFileOnce`(`org_start_plan_config_test.go:63`、4 ケース、`maps.Equal` なので「読まない」側も固定)、案内のテストは `TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir`(`:205`)。元のコードで落ちることと、1 行を外して落ちることは Progress の S2 と self-review の直しの記録による(このステップでは打っていない)。 |
| AC3(410b): main の `ralph.toml` が読めないとき、そのパスと値を名指して止まり、何も作らない。`--config` が読めないときも止まり何も作らない | Met(subdirectory から打つ場合。main worktree の root から打つ場合は V1) | `TestOrgStartPlan_UnreadableConfigStartsNothing`(`org_start_plan_config_test.go:127`)の 2 ケースが、worktree、ブランチ、`ralph-worktree.sh` の記録、manifest、herdr の 5 つを見る。HEAD の binary で一時の repo の `pkg/sub` から打つと、`org: load max_orgs, max_total_seats, [org.approval].mode and [reviewer] from the main worktree's <repo>/ralph.toml: toml: expected character ]`、rc 1 で、ブランチ、worktree、`.harness`、`.git/ralph` は増えない。`--config` の壊れたファイルでは `org: load config: toml: expected character ]`、rc 1。main worktree の root から打つと、先に cwd の読み込みが落ちて、パスも値も名指さない(V1)。 |
| AC4(410c): `zsh -f tests/test-ralph-config.sh` が rc 0、sh・bash・dash も rc 0、shellcheck の新しい警告なし | Met(zsh と shellcheck はこのステップで確認。sh・bash・dash は S3 の記録) | `zsh -f tests/test-ralph-config.sh`(zsh 5.9)は 272/272、rc 0。`shellcheck` 0.11.0 は HEAD と origin/main の両方で警告なし。`sh -n` と `bash -n` は rc 0。追加は `tests/test-ralph-config.sh:20-23` の 1 行で、`ZSH_VERSION` があるときだけ `setopt sh_word_split` を立てる。sh・bash・dash での実行はこのステップでは打っていない(Progress の S3 は 4 つとも rc 0 と記録。/test の `run-test.sh` が sh で打つ)。 |
| AC5: PR #291 の Known gaps 3 つの実物の確かめ | 未着手(担当は次の段の /test) | 計画の D4 と Implementation outline のとおり /test が行う。 |
| AC6: tech-debt 406・409・410 行の書き換え、`run-verify.sh` と `run-test.sh` が通る | 未着手(担当は次の段の /sync-docs、テストは /test) | `docs/tech-debt/README.md` はこの diff で変わっていない。書き換えのほかに V2 の 5 行も /sync-docs に回す。静的な側(`run-static-verify.sh`)はこのステップで通った。 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、"All verifiers passed.") | `tests/test-ralph-config.sh` が未分類なので full fallback で、golang pack を選んだ。 |
| gofmt / go vet / golangci-lint / staticcheck(golang pack) | pass | `gofmt: ok`、golangci-lint `0 issues.`。go vet と staticcheck は何も出さない(通ったときの形)。`Skipping` の行はない。 |
| shellcheck(hook と verify の script)、`sh -n` の hook、guard の awk、`jq -e` の settings | pass | |
| `scripts/check-sync.sh` | pass | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6。 |
| `check-pipeline-sync.sh` / `check-skill-sync.sh` / `check-template-purity.sh` / tech-debt の plan 参照 | pass | |
| branch secret scan | pass | d217c575..cfe92c27 は clean。 |
| `shellcheck tests/test-ralph-config.sh`(個別) | pass | HEAD と origin/main の両方で警告なし。 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org start --help` の承認方式と `[reviewer]` の出どころ(`internal/cli/org.go:1037-1048`)と `--config` の flag の文(`:63`) | Yes | 文は変わっていない。「max_orgs と max_total_seats と同じ ralph.toml から読む」は、1 回の読み込みになって文字どおりの意味になった。HEAD の binary の `org start --help` で同じ文を確かめた。 |
| `internal/cli/org.go` の doc コメント | Yes | `newOrgSpawnRuntimeAt`(`:145-159`)、`withMainWorktreeOrgLimits`(`:202-205`)、`applyMainWorktreeOrgLimits`、`loadMainWorktreeConfig`(`:347-354`)、`newOrgRuntimeFromConfig`、`wireOrgInbox`、2 つの override の変数、`resolveConfig`(`:670-673`)、`loadConfigFile`(`:682-685`)、`resolveOrgStartConfig`(`:687-701`)、`runOrgStartPlan`(`:1190-1203`)が、読む順(cwd か `--config` が先、main が後)、root で同じファイルを 2 回読むこと、`InboxHintStateDir` を両方で設定することを同じように書く。消えた `resolveOrgStartReviewer` を名指す doc はコードに残っていない。 |
| エラーの文 | Yes | 旧文 `org: load [reviewer] from the main worktree's` と新しい文を名指す文書は、計画とテスト(`org_start_plan_config_test.go:149`)のほかにない。spec と rule は start --plan のエラーの文を書いていない。 |
| `.claude/rules/ralph/model-routing.md:79-85`(start --plan の段落)と `templates/base/` の写し | Yes | 「start --plan が読んだ値を leader のタスクの 2 行に書く」は今も正しい。読む回数は書いていない。 |
| `README.md:150` | Yes | 「start --plan が `[reviewer]` の 4 つの値を leader のタスクに写す」は今も正しい。 |
| `docs/specs/` | Yes | 変えた関数もエラーの文も名指していない。 |
| `docs/tech-debt/README.md` 406・409・410 行 | 未着手 | AC6 で /sync-docs が書き換える。406 は S1、409 は S2、410(b) は `TestOrgStartPlan_UnreadableConfigStartsNothing`、410(c) は S3 で直った。 |
| `docs/tech-debt/README.md` 148・187・198・208・220 行 | No(V2) | 下の V2。Trigger の列を python で読み、変えた関数の名前で引いた。 |
| 計画の Acceptance criteria の checkbox | 遅れ(INFO) | AC1〜AC6 はすべて `[ ]` のまま。 |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | AC3 の「main の `ralph.toml` が読めないとき、そのパスと読もうとした値を名指して止まる」は、cwd が main worktree の root でないときだけ成り立つ。root(self-review L4 が「普通に打つ場所」と書く所)から打つと、先に cwd の読み込み(`resolveConfig`、`internal/cli/org.go:703`)が同じファイルで落ち、`org: load config: toml: expected character ]` で止まる。パスも値も名指さない。何も作らない点は同じ。origin/main の binary も同じ文を出すので、この diff が持ち込んだものではない。テストは `pkg/sub` からだけ打つ(`org_start_plan_config_test.go:156`)。 | HEAD と base の binary の probe(evidence の log の "probe" の節)。base は sub から `org: load max_orgs, max_total_seats and [org.approval].mode from the main worktree's <repo>/ralph.toml: …`、root から `org: load config: toml: expected character ]`。 | merge は止めない。PR の Known gaps に書くか、tech-debt に 1 行足す(/sync-docs)。直すなら `resolveConfig` の失敗の文に読んだパスを付ける(spawn と `start <task>` の `org: load config:` も同じ形なので、start --plan だけの話ではない)。 |
| V2 | LOW | 406・409・410 のほかに、この diff が Trigger に当たる、または名前が古くなった tech-debt の行が 5 つある。148: Trigger「`resolveOrgConfig`、`withMainWorktreeOrgLimits` … への変更」に当たる(`withMainWorktreeOrgLimits` の本体と doc、`resolveConfig` が変わった)。本文は消えた `resolveOrgStartReviewer` を名指す。187(d): Trigger「`withMainWorktreeOrgLimits` の次の変更」に当たる。198(e) T-5: Trigger「`runOrgStartPlan` の次の変更」に当たる。「`newOrgSpawnRuntimeAt` の失敗の枝が打たれない」は、その枝が `resolveOrgStartConfig` の失敗になり `TestOrgStartPlan_UnreadableConfigStartsNothing` が打つので古い。`resolveModelOrWarn` の枝は今も打たれない。208(b): Trigger「`newOrgRuntimeAt` の次の編集」に当たる(本体が `newOrgRuntimeFromConfig` を呼ぶ形に変わった)。本文の「`newOrgSpawnRuntimeAt`(spawn、`start <task>`、`start --plan`)」は、start --plan がもう通らないので古い。220(b): Trigger「`withMainWorktreeOrgLimits` の次の変更: 名前を変える」に当たる。 | `docs/tech-debt/README.md:148,187,198,208,220`。self-review は 148 だけを挙げた。 | /sync-docs で、行ごとに「払う」か、日付つきで「当たったが払わなかった」と理由を書く(この repo のほかの行と同じ形)。148 と 198(e) と 208(b) は古い名前と記述も直す。220(b) の改名と 208(b) の doc の言い換えはコードの変更なので、払うなら別のスライス。 |
| I1 | INFO | `--config` で名指した壊れたファイルの文 `org: load config: toml: expected character ]` はパスを含まない。`config.Load` が TOML のエラーにパスを付けないため(`internal/config/config.go:892-893`)。AC3 は `--config` の場合に「止まり何も作らない」までを求め、それは満たす。前からある形。 | probe(HEAD と base で同じ文)。 | V1 と同じ直し方で両方が直る。 |
| I2 | INFO | 計画の D1 は「ほかの 11 か所の呼び出し元は変えない」と書くが、`newOrgRuntimeAt` のテスト以外の呼び出し元は base も HEAD も 10 か所。変えないという中身は成り立つ。 | `git grep 'newOrgRuntimeAt('`(evidence の log)。self-review も 10 と数えた。 | 直さなくてよい。直すなら計画の 1 語。 |
| I3 | INFO | 計画の Acceptance criteria の checkbox が全部 `[ ]` のまま。 | 計画の 78-83 行。 | /pr の前に lead が付ける。 |

## Observational checks

- HEAD の binary を `go build -o <S>/ralph ./cmd/ralph` で作り、origin/main は `git archive` で取り出して `<S>/ralph-base` を作った(`<S>` は scratchpad のディレクトリ)。scratchpad に `git init` した repo に壊れた `ralph.toml`(`[org` で閉じない)をコミットし、`ralph org start --plan plan.md --feature b` を `pkg/sub` と root から、`--config <壊れたファイル>` つきで `pkg/sub` から打った。エラーはどれも設定の読み込みで出るので、herdr、agmsg、見張りには届かない。すべて rc 1 で、repo のブランチ、worktree、`.harness`、`.git/ralph` は増えなかった。
- HEAD と base の違いは、`pkg/sub` から打ったときの文だけ(`[reviewer]` が加わった)。root から打ったときと `--config` のときの文は同じ。
- 計画の digest は承認時と同じ `11ccf2ed5a4c`。
- load average は終わりの時点で 2.63(`uptime`)。

## Coverage gaps

- Go のテストの結果(数えるテスト 4 ケース、読めない設定のテスト 2 ケース、案内のテスト 2 subtest、既存の start と spawn のテスト)は見ていない。/test の担当。元のコードでテストが落ちることと、1 行を外して落ちることは、Progress の記録を信頼した。
- `tests/test-ralph-config.sh` の sh・bash・dash での実行は打っていない(lead が許したのは `zsh -f` の 1 本)。S3 の記録と /test の `run-test.sh` で見る。
- 本番の見張りの起動(`orgWatchChild`)が設定を読まないことは grep で見ただけで、打っていない。
- AC5(実物の確かめ)は /test、AC6(tech-debt の書き換え)は /sync-docs の担当で、このステップでは見ていない。

## Verdict

pass

- Verified: AC1(行数、移した本文の一致、テストの名前の差)、AC2(読み込みの経路と回数、spawn と `start <task>` の経路、値の同等、`InboxHintStateDir` の 2 か所、テストの存在)、AC3(テストの存在、HEAD の binary の probe)、AC4(`zsh -f` 272/272 rc 0、shellcheck の警告なし)。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、secret scan)はすべて通った。計画の digest は承認時と同じ。
- Partially verified: AC3 は subdirectory から打つ場合だけ文面どおり(V1。前からある動き)。AC4 の sh・bash・dash は S3 の記録による。
- Not verified: Go のテストの結果(/test)、AC5(/test)、AC6(/sync-docs)。
- 所見: LOW 2 件(V1、V2)、INFO 3 件。どれも merge を止めない。
- /sync-docs に回すもの: tech-debt 406・409・410 行の書き換え(AC6)、V2 の 148・187・198・208・220 行、V1 を tech-debt に残すならその 1 行。
