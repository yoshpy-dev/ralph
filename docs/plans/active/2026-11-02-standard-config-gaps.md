# standard-config-gaps

- Status: Approved
- Approved: 2026-11-02 sha256:11ccf2ed5a4c
- Owner: Claude Code
- Date: 2026-11-02
- Related request: 2026-11-01 のユーザーの依頼「前の 2 本の既知の穴を修正できますか？」のうち、PR #291(#276 の B1: `[standard]` と `[reviewer]` の設定)の Known gaps と、docs/tech-debt/README.md の 406・409・410 行。1 本目の PR #290 の穴は PR #296 で直した
- Related issue: N/A
- Type: fix
- Branch: fix/standard-config-gaps

## Objective

PR #291 が残した既知の穴を直す。`ralph org start --plan` は main worktree の `ralph.toml` を 2 回読み、承認方式と `[reviewer]` を別々の読み込みから取っている。テストが 800 行を超えたファイルがあり、テストが打たない分岐と、zsh で動かないテストの script もある。PR #291 が実物で確かめていない点(v6.0.0 の ralph での skill の扱い、fable の `--effort`、helper の値で claude と codex を起こすこと)も確かめて証跡に残す。

## Scope

行番号は main d217c575 の docs/tech-debt/README.md のもの。

- **406: `internal/cli/org_feature_test.go` の 819 行**
  - 最初のスライスで、`[reviewer]` のテスト 3 本(`TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings`、`TestOrgStartPlan_ReviewerFromMainWorktree`、`TestOrgStartHelp_ReviewerSource`)を `internal/cli/org_feature_reviewer_test.go` へ移す。中身は変えない。
- **409: `ralph org start --plan` の 2 回の読み込み**
  - `runOrgStartPlan`(`internal/cli/org.go`)は、設定を 1 回だけ読む。`--config` か `./ralph.toml` を 1 回、main worktree の `ralph.toml` を当てはまるときに 1 回読み、runtime の `[org]`(上限と承認方式)と leader のタスクの `[reviewer]` を、その読み込みから取る(D1)。
  - `ralph org spawn` などほかの動詞の読み方は変えない。上限と承認方式を main worktree の値で上書きする規則は 1 か所に置き、spawn と start で共有する(D1)。
  - main worktree の `ralph.toml` が読めないときのエラーの文は 1 つにし、読もうとした値をすべて名指す(D2)。
- **410(b): 読めない設定で start が止まる道のテスト**
  - main worktree の `ralph.toml` が読めないとき、`ralph org start --plan` はそのパスを名指すエラーで止まり、worktree も台帳の記録も作らない。これを打つテストを足す。`--config` で名指したファイルが読めないときも同じ。
- **410(c): `tests/test-ralph-config.sh` を `zsh -f` で動かす**
  - script の先頭近くで、zsh のときだけ単語の分割を sh と同じにする(D5)。`zsh -f tests/test-ralph-config.sh` が通り、sh・bash・dash での結果は変わらない。
- **PR #291 の Known gaps の実物の確かめ**(/test の evidence、D4)
  - 先に `ralph version` を打ち、PATH の ralph が 6.0.0 であることを確かめる(計画の時点では `/opt/homebrew/bin/ralph` が 6.0.0 で、`ralph config get` は exit 1)。違っていたら v6.0.0 の binary を別に用意する。
  - PATH の ralph 6.0.0(`config get` がない)で、`ralph_setting` が既定値を黙って返すことと、`ralph config get standard.cross_review` が exit 1 と `unknown command "config"` を返して skill が飛ばす形になることを確かめる。
  - `claude -p --model fable --effort <値>` を low・medium・high・xhigh で同じ質問に打ち、thinking のトークンと `modelUsage` のモデルを記録する。
  - 一時の repository の `ralph.toml` に既定と違う `[reviewer]` を書き、`config get` のある ralph で `ralph_setting` が読んだ値で、skill の形の claude と codex を起こす。claude は出力の `modelUsage`、codex はログの頭の model と reasoning effort で、渡した値が使われたことを確かめる。一時の repository は ralph の checkout の外に `git init` で作る。codex は skill と同じく `</dev/null` を付けて起こし、終わったら main checkout で `git status --short -- .codex/config.toml` が空であることを確かめる(codex の前後で main の `.codex/config.toml` が書き換わった前例があるため)。
- **tech-debt の表**: 406・409・410 行を直した形に書き換える。

## Non-goals

- feature org の leader が、タスクの 2 行の値で codex と claude を起こす通し。herdr・agmsg・見張りを使う org を 1 つ動かすことになり、重い。タスクに 2 行が入ることは単体のテストが見ている。PR の Known gaps に残す。
- leader が PATH の `ralph` で switch を読み、動詞を打つこと(PR #291 の Non-goals)。
- `ralph_setting` が exit 2・3 でも既定値で続け、review の switch は止まるという扱いの違い。README と `config get --help` に書いた設計で、穴ではない。
- `newOrgRuntimeAt` を使うほかの動詞の読み込みの回数。409 は start --plan の main worktree の 2 回を対象にする。
- 406 行が名指す、前から 800 行を超えていた `internal/org/feature_test.go` と `internal/config/config_test.go`。411 行(PR #293 の 5 ファイル)。PR #296 が残した穴。
- README、`/org` skill、`internal/config/config.go`。別のセッションの #269 が変えている最中なので、触らない。

## Assumptions

- `ralph org start --plan` の動きは変えない。決まる承認方式と `[reviewer]` の値、エラーで止まる場面は今と同じで、変わるのは読み込みの回数とエラーの文だけ。
- 読み込みの回数は、テストのための差し替えの変数で数えられる。この repository には前例がある(`absPath`、`retiredOrgStateDirEnvAbs`、`cleanupTestHooks`)。
- 実物の確かめは、main checkout の `ralph.local.toml` を書き換えずにできる。一時の repository の `ralph.toml` に値を書き、その root から打つ。
- fable と codex を打つ回数は数回で、費用は小さい。

## Affected areas

- `internal/cli/org.go`(`runOrgStartPlan`、`newOrgSpawnRuntimeAt`、`newOrgRuntimeAt`、`withMainWorktreeOrgLimits`、`resolveOrgStartReviewer`)
- `internal/cli/org_feature_test.go`、新しい `internal/cli/org_feature_reviewer_test.go`、start --plan の読み込みとエラーのテストを置く新しい `internal/cli/org_start_plan_config_test.go`
- `tests/test-ralph-config.sh`
- `docs/evidence/`(/test の実物の確かめ)
- `docs/tech-debt/README.md`(406・409・410 行)

## Visual review

- Page: `.harness/state/plan-visual/standard-config-gaps.html`(かんたんな図 1 つ、詳しい図 2 つ)
- Lint: `./scripts/plan-visual.sh lint` は exit 0。
- Self-check: 全体、overview、detail-overview、detail-reads を `shot` で撮って読んだ。detail-overview の `org.go` の箱で 2 行目が枠からはみ出していたので、文を短くして撮り直した。撮り直して、すべてきれい。plan advisory の所見 1 を受けて detail-reads の `newOrgRuntimeFromConfig` の箱の文を直し、lint と撮り直しで確かめた。

## Design decisions

- **D1: start --plan は先に設定を読み、その値から runtime を作る。** `newOrgRuntimeAt` を「設定を読む」部分と「読んだ設定から runtime を作る」部分(`newOrgRuntimeFromConfig`)に分け、start --plan は後者を使う。上限と承認方式を main worktree の値で上書きする部分は 1 つの関数(`applyMainWorktreeOrgLimits`)にし、spawn の `withMainWorktreeOrgLimits` と start の両方がそれを呼ぶ。流れは「`resolveConfig` と `loadMainWorktreeConfig` で 1 回ずつ読む → `applyMainWorktreeOrgLimits` で読んだ `[org]` に main の上限と承認方式を当てる → `newOrgRuntimeFromConfig` で runtime を作り、`[reviewer]` も同じ読み込みから取る」。`withMainWorktreeOrgLimits` は「読む」部分と `applyMainWorktreeOrgLimits` の呼び出しに分け、spawn からの呼ばれ方は変えない。`newOrgRuntimeAt` は引数の変わらない薄い包みとして残し、ほかの 11 か所の呼び出し元は変えない。`newOrgSpawnRuntimeAt` が runtime に設定している `InboxHintStateDir`(`orgInboxHintStateDir`。spawn の失敗の文が出す `ralph org stop` に `--state-dir` を付ける値)は、start --plan の新しい経路でも同じく設定する(plan advisory の所見 1)。tech-debt 409 行の Trigger の案と同じ形。別案の「runtime に `config.Config` 全体を持たせる」は、`org.Org` の型を変え、ほかの動詞にも波及するので採らない。
- **D2: main worktree の `ralph.toml` が読めないときの start の文は 1 つにする。** 今は先に読む runtime の側の文(上限と承認方式)が出て、`[reviewer]` の側の文は出ない。1 回の読み込みにすると、文は 1 つで足りる。文は `max_orgs`・`max_total_seats`・`[org.approval].mode`・`[reviewer]` をすべて名指す。今の文を固定したテストはない(grep で確かめた)。
- **D3: 回数は差し替えの変数で数える。** `internal/cli` に、設定のファイルを読む関数を指す変数を 1 つ置き、`resolveConfig` と `loadMainWorktreeConfig` の両方がそれを通る。本番は `config.Load` のまま、テストはパスごとに数える版に差し替える。今の start --plan は、main の経路で main worktree の `ralph.toml` を 2 回、cwd の `ralph.toml` を 1 回読む。git の起動の回数で数える案は、ほかの所も git を打つので、読み込みの回数と 1 対 1 にならない。テストは元のコードで 2 回を数えて落ちることを確かめる。
- **D4: 実物の確かめは /test の tester が行う。** PR #296 と同じ分け方。実装のスライスはコードとテストだけにする。
- **D5: zsh は先頭で `sh_word_split` を立てる。** 引用のない単語の並びを回すループが 9 か所あり、1 つずつ書き換えると差分が広がる。zsh のときだけ 1 行で sh と同じ分割にする。そのあとも zsh の別の違い(`$0`、glob など)で落ちたら、S3 の verify で見つけて 1 か所ずつ直し、Progress checklist に書く。
- Critical forks: None。D1〜D5 は、tech-debt の行が示す案か、この repository の前例のとおり。

## Acceptance criteria

- [x] AC1(406): `internal/cli/org_feature_test.go` は 800 行以下。3 本のテストは `internal/cli/org_feature_reviewer_test.go` にあり、中身は変わらない(移すだけのコミット。`go test -list` の名前は前後で同じ)。
- [x] AC2(409): `ralph org start --plan` は、main worktree の `ralph.toml` を main の上書きとして 1 回だけ読む。runtime の上限と承認方式と、leader のタスクの `[reviewer]` の 2 行は、その 1 回の読み込みから取る。`--config` か `./ralph.toml` も 1 回だけ読む(cwd が main worktree の root なら同じファイルを 2 回読むが、cwd の設定と main の上書きという別の役の読み込みなので許す)。`--config` を渡したときと台帳が main worktree のものでないときは、main worktree の `ralph.toml` を読まない。読み込みを数えるテストがあり、元のコードでは落ちる。決まる値は今と同じで、既存の start と spawn のテストが通る。`--state-dir` で既定でない台帳を選んで打った start --plan の leader が途中で残ったとき、出力の `ralph org stop` の案内がその `--state-dir` を含むことを、CLI のテストで確かめる(`InboxHintStateDir` の設定が抜けると落ちる)。
- [x] AC3(410b): main worktree の `ralph.toml` が読めないとき、`ralph org start --plan` はそのパスと読もうとした値を名指すエラーで止まり、feature の worktree、ブランチ、台帳の記録を作らない。`--config` で名指したファイルが読めないときも、エラーで止まり何も作らない。どちらもテストがある。
- [ ] AC4(410c): `zsh -f tests/test-ralph-config.sh` が rc 0 で終わる。sh・bash・dash(手元にあれば)での実行も rc 0。`shellcheck` が新しい警告を出さない。
- [ ] AC5: /test の evidence に、PR #291 の Known gaps の 3 つの実物の確かめ(v6.0.0 での扱い、fable の `--effort`、helper の値で起こす claude と codex)の結果がある。
- [ ] AC6: tech-debt の 406・409・410 行が、直した形に書き換わっている。`./scripts/run-verify.sh` と `./scripts/run-test.sh` が通る。

## Implementation outline

各スライスの `<go test>` は `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test` を指す。`internal/cli` のテストは全部で 10 分を超えるので、スライスでは `-run` で絞る。全体は /verify と /test で打つ。

1. S1(AC1): 3 本のテストを `internal/cli/org_feature_reviewer_test.go` へ移す。移すだけで、中身は変えない — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'OrgStartPlan_|OrgStartHelp_' ./internal/cli/ && test "$(wc -l < internal/cli/org_feature_test.go)" -le 800`
2. S2(AC2、AC3): `runOrgStartPlan` が設定を 1 回だけ読み、その値から runtime と `[reviewer]` を取る(D1・D2)。上書きの規則を 1 つの関数にして spawn と共有する。`InboxHintStateDir` も今と同じく設定する。回数を数えるテスト、読めない設定のテスト 2 つ、既定でない台帳で残った leader の止める案内のテストを足す(D3) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'OrgStart|OrgSpawn|OrgFeature' ./internal/cli/`
3. S3(AC4): `tests/test-ralph-config.sh` の先頭近くで、zsh のときだけ `sh_word_split` を立てる(D5) — verify: `shellcheck tests/test-ralph-config.sh && sh tests/test-ralph-config.sh && bash tests/test-ralph-config.sh && zsh -f tests/test-ralph-config.sh && { ! command -v dash >/dev/null || dash tests/test-ralph-config.sh; }`

実物の確かめ(AC5)は /test、tech-debt の表の書き換え(AC6)は /sync-docs で行う。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-skill-sync)。
- Spec compliance criteria to confirm: AC1〜AC6。AC2 は、start --plan の読み込みの経路を読み、main worktree の `ralph.toml` を読む呼び出しが 1 つであること、spawn の経路が変わらないことを見る。
- Documentation drift to check: `ralph org start --help` の `[reviewer]` と承認方式の出どころの文(`TestOrgStartHelp_ReviewerSource` と `TestOrgStartHelp_ApprovalModeSource` が固定)。`internal/cli/org.go` の関数の doc コメント(`withMainWorktreeOrgLimits` が start の読み方を説明している)。`.claude/rules/ralph/model-routing.md` の start --plan の段落。docs/tech-debt/README.md の 406・409・410 行。
- Evidence to capture: docs/reports/verify-2026-11-02-standard-config-gaps.md。

## Test plan

- Unit tests: S2 の回数のテストと、読めない設定のテスト 2 つ。
- Integration tests: `internal/cli` の `OrgStart`・`OrgSpawn`・`OrgFeature` のテストを通しで打つ。
- Regression tests: `./scripts/run-test.sh` 全体(shell のテストに `tests/test-ralph-config.sh` を含む)。既知の不安定なテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` と 30 分の timeout)で落ちたら、1 回だけ打ち直して記録する。
- Edge cases: main worktree の `ralph.toml` がない(既定値)、`--config` あり、台帳が main worktree のものでない、main worktree の `ralph.local.toml` だけが `[reviewer]` を変える、読めない `ralph.toml` と読めない `ralph.local.toml`。
- Evidence to capture: docs/reports/test-2026-11-02-standard-config-gaps.md と、実物の確かめの docs/evidence/test-2026-11-02-standard-config-gaps-live.md。token の値や session id は写さない。

## Risks and mitigations

- **start の決まる値が変わる**: 既存の `TestOrgStartPlan_ApprovalModeFromMainWorktree` と `TestOrgStartPlan_ReviewerFromMainWorktree` が通ることを S2 の verify に含める。
- **start の runtime の初期化が抜ける**: `newOrgSpawnRuntimeAt` を通らなくなるので、そこで設定していた値(`InboxHintStateDir`)が抜けうる。D1 に維持を書き、AC2 の CLI テストで確かめる(plan advisory の所見 1)。
- **spawn の上限の読み方が変わる**: 上書きの規則を共有の関数に移すだけで、spawn の呼び出しの順は変えない。`OrgSpawn` のテストを S2 の verify に含める。
- **差し替えの変数が本番の動きを変える**: 本番は `config.Load` を指したまま。差し替えはテストの中だけで、`t.Cleanup` で戻す。
- **実物の確かめが main checkout の設定を変える**: 一時の repository の `ralph.toml` だけに書き、main checkout の `ralph.local.toml` には触らない。作ったものは消す。
- **他のセッションとの衝突**: #269・#277 は `internal/config/config.go`・README・`/org` skill・ルートの `ralph.toml` を変える。この計画はどれも変えない。`internal/cli/org.go` は #269 も触りうるので、PR の前に main を取り込んで `OrgStart` のテストを打ち直す。

## Rollout or rollback notes

- 変わるのは、start --plan の設定の読み込みの回数と、読めないときのエラーの文。決まる値と出力の機械が読む行は変えない。
- 戻すときは PR を revert する。設定ファイルと台帳の形は変えない。

## Open questions

- なし。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh。`ralph_setting` は `config get` のある ralph で読んだ。PATH の ralph 6.0.0 には `config get` がない)。git の状態は前後で同じ。所見 1 件は、ユーザーの指示「HIGH と MEDIUM は聞かずに計画に反映する」に従い反映した。
  - [MEDIUM] 1: start --plan の runtime が `newOrgSpawnRuntimeAt` を通らなくなると、そこで設定していた `InboxHintStateDir` が抜け、既定でない台帳で残った leader の `ralph org stop` の案内から `--state-dir` が消える。→ D1 に設定の維持を書き、AC2 と S2 に CLI の回帰テストを足し、Risks に 1 行足した。図の detail-reads の箱の文も直した。
  - 計画の前に consult-plan-beta に D1〜D5 を聞いた(ユーザーの指示「計画の分かれ目は consult に聞いて推奨案を自分で選ぶ」)。判定は「進めてよい」。足した点: PATH の ralph の版を先に確かめる(計画の時点で 6.0.0 を確かめた)、codex の実物の確かめに `</dev/null` と main の `.codex/config.toml` の確認、一時の repository は checkout の外、D1 の流れの一文、D3 の数える所、AC2 の「main の上書きとして 1 回」、D5 の zsh の別の違いの扱い、leader の通しは PR の Known gaps に残す。
- [x] Plan approved — 自動承認(ユーザーの指示)。2026-11-02、digest 11ccf2ed5a4c。
- [x] Branch created — fix/standard-config-gaps(base d217c575)
- [x] Implementation started
  - S1(AC1): b71354d0。`org_feature_test.go` は 819 → 647 行、3 本を `org_feature_reviewer_test.go`(182 行)へ移した。本文は `cmp` で一致、`go test -list` の名前は前後で同じ 975 件。使わなくなった `internal/config` の import を 1 行消した。implementer は sonnet。
  - S2(AC2、AC3): 7b9dc257(`internal/cli/org.go` +82/−55、新しい `org_start_plan_config_test.go` 222 行)。`loadConfigFile`(本番は `config.Load`)、`newOrgRuntimeFromConfig`、`applyMainWorktreeOrgLimits`、`resolveOrgStartConfig` を足し、`resolveOrgStartReviewer` は呼ぶ所が 1 つだったので `resolveOrgStartConfig` に置き換えた。読む順は今と同じ(cwd か `--config` が先、main が後)。main が読めないときの文は `org: load max_orgs, max_total_seats, [org.approval].mode and [reviewer] from the main worktree's <path>: <err>`。数えるテストは元のロジックで 3 ケースとも落ちた(main 2 回、`custom.toml` 2 回、台帳が外のとき sub 2 回)。`InboxHintStateDir` の 1 行を外すと `TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir` が落ちた。AC3 は `TestOrgStartPlan_UnreadableConfigStartsNothing` の 2 ケース。mutant 2 つ(main の値を当てない、cwd の `[reviewer]` を使う)は既存のテストで落ちた。verify は rc 0(88 本、67.8 秒)、`golangci-lint` 0 issues。implementer は opus。implementer が自分で consult-plan-beta に聞き、返事はコミットの後に届いたが、指摘 2 つと注意 2 つは 7b9dc257 で満たしていて追加のコミットはない。tech-debt 148 行が消した `resolveOrgStartReviewer` を名指すので、/sync-docs で直す。
  - S3(AC4): 2d4baa2d。inline(1 ファイル 5 行、計画の inline の例外)。zsh のときだけ `setopt sh_word_split` を立てる。sh・bash・dash・`zsh -f` の 4 つとも rc 0、272/272。`shellcheck` は警告なし。zsh の別の違いでは落ちなかった。自己点検: 不要な変更 clean、誤字と写し間違い clean、デバッグの残り clean、秘密情報 clean、スライス外のファイル clean。
  - 実装の終わり: S1〜S3 の verify はすべて rc 0。
- [x] Review artifact created — docs/reports/self-review-2026-11-02-standard-config-gaps.md(1c49a107)。CRITICAL 0、HIGH 0、MEDIUM 0、LOW 7、Merge yes。insight event は reviewer が足した(`scripts/insights-append.sh` は UTC の日付を使うので、ファイルは `docs/insights/events/2026-11-01-standard-config-gaps.jsonl`)。
  - LOW 7 件はどれも S2 のファイルの中で安いので、すべて a12a08a5(F1、implementer は sonnet)で直した。L1 は包みの関数を足さず、2 か所に互いとテストの名前を書いた。L4 で数えるテストに cwd = main worktree の root のケースを足し、数えるテストは 4 ケースになった(同じパスを 2 回、それより多くない)。L5 の「常に真の `else if`」は実際には偽になる場合があり、`switch` にした。L6 はテストの名前を変えず、doc と、原因が `toml: ` で始まることの検査を足した(報告の案の `Contains("toml")` は、パスの `ralph.toml` に当たって何も固定しないため)。L7 は `[org].state_dir` の subtest を足した。新しいケースは `-overlay` の変異 3 つで落ちた。verify は rc 0(69.5 秒)。
- [x] Verification artifact created — docs/reports/verify-2026-11-02-standard-config-gaps.md(d619d051)。pass、LOW 2・INFO 3。V1(main worktree の root から打つと、壊れた同じ `ralph.toml` を cwd の読み込みが先に読み、文が `org: load config:` になって AC3 の「パスと値を名指す」が成り立たない。base でも同じ文で、この diff の持ち込みではない)は tech-debt に 1 行残し、PR の Known gaps に書く。V2(148・187(d)・198(e)・208(b)・220(b) の tech-debt の行が、この diff の Trigger に当たるか名前が古い)は /sync-docs に回す。I2: D1 の「ほかの 11 か所」は実際には 10 か所(計画の本文なので直さず、ここに記録する)。I3: AC1〜AC4 の箱をここで付けた。
- [ ] Test artifact created
- [ ] PR created
