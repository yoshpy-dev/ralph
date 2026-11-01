# org-ledger-gaps

- Status: Approved
- Approved: 2026-11-01 sha256:338876ea047b
- Owner: Claude Code
- Date: 2026-11-01
- Related request: 2026-11-01 のユーザーの依頼「前の 2 本の既知の穴を修正できますか？」のうち、PR #290(#276 の A2: `RALPH_ORG_STATE_DIR` の撤去と `[org].state_dir`)の Known gaps と、docs/tech-debt/README.md の 395〜400 行
- Related issue: N/A
- Type: fix
- Branch: fix/org-ledger-gaps

## Objective

PR #290 が残した既知の穴を直す。台帳(org の state dir)の決め方と個人用のファイル `ralph.local.toml` の読み方の周りに、次の穴が残っている。

- 再実行の案内のコマンドが、`--state-dir` で名指した台帳を落とす。
- 同じ判定が何か所にも書かれている。
- doctor の文がずれている。
- テストが打たない分岐がある。

もう 1 本の PR #291(B1)の穴は、この PR の merge のあとに別の計画で直す(D1)。

## Scope

穴の出どころごとに、直す中身を書く。行番号は main 4fef6b8c の docs/tech-debt/README.md のもの。

- **395: cleanup と merge の再実行のコマンド**
  - `ralph org cleanup` と `ralph org merge` が出す再実行のコマンドすべてに、`--state-dir` が渡されたときだけ `--state-dir <dir>` を付ける(D5)。`--branch` のある形(`ralph org cleanup --branch …`、`ralph org merge --branch …`)も、`--branch` のない形(`ralph org cleanup` で merge 済みのものをまとめて片付けたときの「run ralph org cleanup again」など)も含む。
  - `--branch` のない形は、元の打ち方の branch の選び方も保つ。`--branch` を付けて打ったときの文は `--branch` を付け、付けずに打ったときの文は付けない。今は、`--branch` を付けて打っても、中断の文(`runCleanupSteps`)は `--branch` のない形を出すので、写して打つと片付けの範囲が広がる(plan advisory 1)。
  - 対象は、cleanup の拒否の文(`cleanupAgain` と `internal/org/cleanup.go` の同じ形の文)、merge の拒否の文(`mergeCheck.again`、`internal/org/merge_run.go` のロックと中断の文)、CLI の文(`cleanupOriginRefusal`、`internal/cli/org_cleanup.go` の gh の文と main worktree がないときの文と `runCleanupSteps` の中断の文、`mergeCleanupRoot`)。
  - 例外は、台帳を worktree の外へ動かすよう案内する文(`ledgerMoveStep`)。元の `--state-dir` を付けると、動かした先でなく元の場所を選んでしまう。この文は元の値を付けず、「`--state-dir <新しい場所>` を付けるか、`ralph.local.toml` の `[org].state_dir` に新しい場所を書いて打ち直す」の形にする(plan advisory 2)。
  - `CleanupParams` と `MergeParams` に、その値を受け取るフィールドを足す。merge が中で呼ぶ cleanup にも渡す。
  - `ralph org cleanup --help` と `ralph org merge --help` の「again」の文に、`--state-dir` を付けて打ち直すことを足す。
- **396(a): 「絶対か `~/`」の判定と `~/` の展開**
  - `internal/config` に、判定の関数 1 つと展開の関数 1 つを置く(D2)。展開は `(string, error)` を返し、`os.UserHomeDir` の失敗はエラーにする。
  - 判定を呼ぶのは `checkLocalAgmsgHome`、`LocalOrgStateDir`、`agmsgHomeIsRelative`(否定の形)。
  - 展開を呼ぶのは `LocalOrgStateDir` と `expandTilde`(`internal/org/driver/agmsg.go`)。`expandTilde` はエラーのときに値をそのまま返す今の振る舞いを保つ。
- **396(b): doctor の「using defaults」**
  - `config.Load` がエラーを返したら、doctor は `config.Default()` で後の検査を続ける(D3)。こうすると、warn の文「using defaults」のとおりになる。`config.Load` の契約は変えない。
- **397: `RALPH_ORG_STATE_DIR` で断る条件の文**
  - 条件の文を Go の定数 1 つにする。doctor の warn(`retiredEnvs` の `effect`)と、`ralph org escalate --help` の最後の文は、この定数と同じ文にそろえる(D4)。
  - 判定 `checkRetiredOrgStateDirEnv` に 4 つの場合(`--state-dir`、個人用のファイル、既定の台帳と同じ場所、違う場所)を通す表のテストを足す。テストは、結果(拒否か注意か)と、2 つの文が定数を含むことを確かめる。
- **398: `ralph doctor --help` の `--strict`**
  - usage の文から backtick を外し、`--strict` の行に値の名前が出ないことをテストで確かめる。
- **PR #290 の Known gaps: `ralph config get --help` の exit 3**
  - exit 3 の行に、local-only のキーを書いた `ralph.toml` も含むことを書く。キーの名前は `config.LocalOnlyKeys()` から作る。
- **399(b): 既定の層での git の 2 回**
  - `ResolveOrgStateDir` は、main worktree の root を 1 回だけ求め、その root の `ralph.local.toml` を読む。
  - `checkRetiredOrgStateDirEnv` の拒否の文は、`ralph.local.toml` のパスを台帳のパスから作り、git を打たない。
- **400: `internal/config/local_test.go` の 1011 行**
  - 最初のスライスで、`TestLoad_LocalOnlyKeyInRalphToml` から後ろのテスト(local-only のキーと `LocalOrgStateDir`)を別のファイルへ移す。中身は変えない。
- **PR #290 の test gaps 1・2**
  - `FeatureRepoRoot` の「main worktree のない bare repository の linked worktree」のエラーの分岐を打つテストを足す。
  - `checkRetiredOrgStateDirEnv` と `mustAbs` の `filepath.Abs` の失敗の分岐を打つテストを足す(D7)。
- **399(a): `RALPH_ORG_STATE_DIR` での拒否の文の頭**(最後のスライス。D6)
  - 拒否の文を「cannot resolve the org ledger:」でなく、`guardLegacyOrgStateDir` と同じ「refusing to change the org ledger:」の形で始める。
  - 解決できない台帳と見分ける sentinel を足し、`ralph org escalate` の記録しない道はどちらも同じに扱う。全文を固定したテストを直す。
- **PR #290 の test gaps 3・4**(/test の evidence)
  - build した binary を一時の repository で打つ。実物の `ralph.local.toml` に、実際の HOME の `~/` を使う値を書いて確かめる。
  - `--state-dir` 付きの `ralph org watch --detach` の子が watch.log に注意の 1 行を出すことも確かめる。
- **tech-debt の表**: 395〜400 行を直した形に書き換える(直さずに残る部分があれば、その部分だけ残す)。

## Non-goals

- PR #291(B1)の穴。tech-debt 406・409・410、`ralph org start --plan` の 2 回の読み込み、`tests/test-ralph-config.sh` の zsh、実物の claude と codex の確かめが当たる。この PR の merge のあとに別の計画で直す(D1)。
- README の `[org].state_dir` の段落と、`/org` skill の 4 つの写しの文(397)。言い方が Go の文と違い(skill は日本語)、`/org` skill と README はセッション A の #269 が変えている最中なので、触らない(D4)。
- `ralph org regate` が出す `ralph org cleanup --branch` と自分の再実行の文。regate の文は tech-debt 403 行の領域で、#264(セッション A)の側。
- PR #290 の self-review L7(source の名前 `config` が `--config` と紛らわしい)。計画で決めた名前で、`case "flag", "config":` などコードが比べる値。変えると出力の契約が変わる。
- PR #290 の Known gaps の regate の不安定なテスト(#264 の側、tech-debt 393 行)と、cross-review の codex が read-only の sandbox で Go のテストを打てないこと(環境の制約)。
- `config.Load` がエラーのときに `Default()` を返すように変えること(396(b) の別案。呼び出し元が多い)。
- `internal/org/feature_test.go`・`internal/config/config_test.go` など、この 2 本の前から 800 行を超えていたファイルの分割。

## Assumptions

- `internal/org/driver` から `internal/config` を import しても循環しない(consult で両方向とも確認済み)。
- `--state-dir` を付けるかどうかの規則は `orgInboxHintStateDir`(`internal/cli/org_inbox.go`)と同じでよい。`ralph.local.toml` の `[org].state_dir` で決めた台帳は、素のコマンドでも同じファイルを読むので付けない。
- `os.Getwd` の失敗(cwd を消す)は、macOS と Linux で同じようには起こせないかもしれない。起こせなければ、`filepath.Abs` をパッケージの変数にしてテストで差し替える(D7)。
- セッション A(#269)とセッション C(#271・#277)の branch は、この計画のファイルのうち `internal/config/config.go`・`README.md`・`/org` skill に触る。この計画はこのうち `internal/config/config.go` だけを読む(変えない)。

## Affected areas

- `internal/config/`: `local.go`(判定と展開の関数、`LocalOrgStateDir` の root の受け取り)、新しいテストのファイル(`local_test.go` から移すもの、判定と展開の表のテスト)
- `internal/org/`: `statedir.go`(`ResolveOrgStateDir`、`mustAbs`)、`cleanup.go`、`merge.go`、`merge_run.go`、`driver/agmsg.go`、各テスト
- `internal/cli/`: `org_cleanup.go`、`org_merge.go`、`org_legacy_ledger.go`、`org_inbox.go`(escalate の help)、`org_watch.go`(`agmsgHomeIsRelative`)、`doctor.go`、`config.go`(`config get --help`)、各テスト
- `docs/tech-debt/README.md`(395〜400 行)
- `docs/evidence/`(/test の実物の確かめ)

## Visual review

- Page: `.harness/state/plan-visual/org-ledger-gaps.html`(かんたんな図 2 つ、詳しい図 4 つ)
- Lint: `./scripts/plan-visual.sh lint` は exit 0。
- Self-check: 全体、overview、easy-rest、detail-overview、detail-statedir-flag、detail-homepath、detail-git-calls を `shot` で撮って読んだ。detail-homepath の破線が箱を横切っていたので並べ方を直した。detail-overview の確かめることの文で `config.go` がどちらのものか曖昧だったので直した。撮り直して、すべてきれい。

## Design decisions

- **D1: 2 本に分け、この計画は PR #290 の穴だけを持つ。** 1 本にまとめると、スライスが 13 前後、diff が数千行になり、self-review と cross-review が重くなる。B1 の穴(`internal/cli/org_feature.go` と `tests/test-ralph-config.sh`)とこの計画の穴は、触るファイルがほとんど重ならない。ユーザーの並べ方(A2 → B1)に合わせて、こちらを先にする。consult-plan-gaps の判定どおり。
- **D2: 判定と展開を `internal/config` に置き、`expandTilde` の振る舞いは変えない。** tech-debt 396 行の案のうち、1 か所にまとめるほうを選ぶ。別案の「同じ値を 3 つの判定に通すテストだけ」では、4 つ目の読み手がまた自分で書ける。`expandTilde` は seat の agmsg の場所を決める driver の関数で、`os.UserHomeDir` の失敗で値をそのまま返す。これをエラーにすると、agmsg を呼ぶ道の振る舞いが変わるので、関数の中でエラーを捨てて今の値を返す。
- **D3: 396(b) は doctor の側で直す。** `config.Load` が途中まで埋めた cfg を返す道は 5 つあり、呼び出し元は doctor のほかにも多い。Load を変えると、その全部の振る舞いを確かめ直すことになる。doctor は文で「using defaults」と言っているので、その文に合わせて `Default()` を使う。
- **D4: 397 は Go の 2 か所と判定の表のテストまで。** README と `/org` skill の文は、同じ条件を別の言い方(skill は日本語)で書いていて、定数を含むかどうかのテストは成り立たない(consult-plan-gaps の指摘)。どちらも #269 が変えている最中で、衝突しやすいファイルでもある。
- **D5: `--state-dir` は、渡されたときだけ付ける。** `orgInboxHintStateDir` と同じ規則。個人用のファイルで決めた台帳は、素のコマンドでも同じファイルを読む(tech-debt 395 行の Impact)。
- **D6: 399(a) は最後のスライスに置き、膨らんだら落とす。** 全文を固定したテストが 6 本ある(`internal/cli/org_cleanup_test.go` の 2 本、`org_retired_state_dir_env_test.go` の 3 本、`org_ledger_config_test.go` の 1 本)。変数は撤去済みで、残したまま更新した人だけが読む文なので、価値は低い。この 6 本と escalate の `errors.Is` のほかに直す所が出たら、スライスを落として tech-debt の行に残す。
- **D7: `filepath.Abs` の失敗は、まず cwd を消して起こし、起こせなければパッケージの変数で差し替える。** テストのための差し替えはこの repository にもある(`cleanupTestHooks`)。ただ、本番のコードに足す変数はなるべく減らしたい。
- Critical forks: 上の D1〜D7 は consult-plan-gaps に聞いたうえで選んだ(ユーザーの指示「計画の分かれ目は consult に聞いて推奨案を自分で選ぶ」)。

## Acceptance criteria

- [ ] AC1(395): `--state-dir <dir>` を付けて打った `ralph org cleanup` と `ralph org merge` が出す再実行のコマンドは、`--branch` のある形もない形も、どれも `--state-dir <dir>`(shell の 1 語に quote)を含む。`--branch` を付けて打った `ralph org cleanup` の中断の文は `--branch` を含む。`--state-dir` を付けずに打つと、個人用のファイルで台帳を決めたときも含めて、台帳の指定のないコマンドを出す。台帳を動かすよう案内する文は元の `--state-dir` を含まず、新しい場所の 2 つの指定のしかた(`--state-dir <新しい場所>` と `[org].state_dir`)を名指す。builder ごとにテストがある。少なくとも 1 つのテストは、2 つの台帳を用意し、出力から取り出した再実行のコマンドをそのまま打ち、最初に選んだ台帳だけが読まれることを確かめる。
- [ ] AC2(396a): 「絶対か `~/`」の判定と `~/` の展開が、`internal/config` に 1 つずつある。`checkLocalAgmsgHome`・`LocalOrgStateDir`・`agmsgHomeIsRelative`・`expandTilde` はそれを呼ぶ。`~/x`、`~`、`~user/x`、相対、絶対、空の 6 つの値を、3 つの判定と 2 つの展開に通す表のテストがあり、判定どうしが食い違わない。`expandTilde` は `os.UserHomeDir` が失敗したとき値をそのまま返す。
- [ ] AC3(396b): 読めない `ralph.toml` で `ralph doctor` を打つと、warn の行「parse error: … — using defaults」が出て、後の検査は `config.Default()` の値を読む。途中まで埋めた値を読まないことを、テストで確かめる。
- [ ] AC4(397): 断る条件の文が Go の定数 1 つにあり、doctor の warn と `ralph org escalate --help` はその文を含む(help は改行を空白に読み替えて比べる)。4 つの場合を `checkRetiredOrgStateDirEnv` に通す表のテストが、拒否か注意かと、2 つの文の両方を確かめる。
- [ ] AC5(398): `ralph doctor --help` の `--strict` の行に値の名前が出ない。テストがある。
- [ ] AC6: `ralph config get --help` の exit 3 の説明に、local-only のキーを書いた `ralph.toml` も exit 3 になることが書いてあり、キーの名前は `config.LocalOnlyKeys()` から作る。テストがある。
- [ ] AC7(399b): 既定の層で `ResolveOrgStateDir` が main worktree を探す git の呼び出しは 1 回。`checkRetiredOrgStateDirEnv` の拒否は `ralph.local.toml` のパスを求めるのに git を打たない。回数を数えるテストがあり、元のコードでは落ちる。
- [ ] AC8(400): `internal/config/local_test.go` は 800 行以下になり、移したテストの中身は変わらない(移すだけのコミット)。
- [ ] AC9: `FeatureRepoRoot` の bare repository の分岐と、`filepath.Abs` の失敗の 2 つの分岐を打つテストがある。
- [ ] AC10(399a、D6 で落とせる): `RALPH_ORG_STATE_DIR` での拒否の文が「refusing to change the org ledger:」の形で始まる。`ralph org escalate` はそれでも記録しない道で人に届ける。落としたら、tech-debt 399 行の (a) を残し、理由を書く。
- [ ] AC11: tech-debt の 395〜400 行が、直した形に書き換わっている。`./scripts/run-verify.sh` と `./scripts/run-test.sh` が通る。

## Implementation outline

各スライスの verify の `<go test>` は `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test` を指す。`internal/cli` のテストは全部で 10 分を超えるので、スライスでは `-run` で関係するテストに絞る。全体は /verify と /test で打つ。

1. S1(AC8): `internal/config/local_test.go` の `TestLoad_LocalOnlyKeyInRalphToml` から後ろを、`internal/config/local_only_keys_test.go` と `internal/config/local_state_dir_test.go` に移す。移すだけで、中身は変えない — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && <go test> ./internal/config/ && test "$(wc -l < internal/config/local_test.go)" -le 800`
2. S2(AC2): `internal/config` に判定と展開の関数を足し、4 か所から呼ぶ。表のテストを足す — verify: `test -z "$(gofmt -l internal/config internal/org/driver internal/cli)" && go vet ./internal/config/ ./internal/org/driver/ ./internal/cli/ && <go test> ./internal/config/ ./internal/org/driver/ && <go test> -run 'AgmsgHome|OrgWatch|DirectorStart' ./internal/cli/`
3. S3(AC3): doctor が Load のエラーのとき `config.Default()` を使う。テストを足す — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'Doctor' ./internal/cli/`
4. S4(AC5、AC6): `--strict` の usage と `config get --help` の exit 3 の行を直し、help のテストを足す — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'Doctor|ConfigGet' ./internal/cli/`
5. S5(AC7): `ResolveOrgStateDir` が main worktree の root を 1 回だけ求める。拒否の文のパスは台帳から作る。回数のテストを足す — verify: `test -z "$(gofmt -l internal/config internal/org internal/cli)" && go vet ./internal/config/ ./internal/org/ ./internal/cli/ && <go test> ./internal/config/ && <go test> -run 'StateDir|FeatureRepoRoot|LedgerMainWorktreeRoot' ./internal/org/ && <go test> -run 'RetiredStateDirEnv|OrgLedger|Legacy' ./internal/cli/`
6. S6(AC4): 断る条件の定数と、判定の表のテスト — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'RetiredStateDirEnv|RetiredEnv|OrgEscalate' ./internal/cli/`
7. S7(AC1): cleanup と merge の再実行のコマンドすべてに `--state-dir` を付け、`--branch` の選び方を保つ。台帳を動かす案内は例外にする。help とテスト(出力のコマンドを打ち直すテストを含む)を足す — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && <go test> -run 'Cleanup|Merge' ./internal/org/ && <go test> -run 'OrgCleanup|OrgMerge' ./internal/cli/`
8. S8(AC9): `FeatureRepoRoot` の bare repository と `filepath.Abs` の失敗のテスト(D7) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && <go test> -run 'FeatureRepoRoot|ResolveOrgStateDir|MustAbs' ./internal/org/ && <go test> -run 'RetiredStateDirEnv' ./internal/cli/`
9. S9(AC10、D6 で落とせる): 拒否の文の頭と sentinel、escalate の扱い、全文を固定したテスト — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && <go test> -run 'RetiredStateDirEnv|OrgCleanup|OrgLedger|OrgEscalate' ./internal/cli/`

tech-debt の表の書き換え(AC11)は /sync-docs で行う。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck)。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh`(skill は変えないが、drift がないことを見る)。
- Spec compliance criteria to confirm: AC1〜AC11。特に AC1 は、`--state-dir` を付けて出るコマンドを 1 つずつ数え、対象の builder がすべて当たっていることを見る。
- Documentation drift to check: `ralph org cleanup --help`・`ralph org merge --help`・`ralph doctor --help`・`ralph config get --help`・`ralph org escalate --help` の文と README の対応する段落。docs/tech-debt/README.md の 395〜400 行。AGENTS.md の地図(`internal/config/` の行が新しい関数を名指す必要があるか)。
- Evidence to capture: docs/reports/verify-2026-11-01-org-ledger-gaps.md。

## Test plan

- Unit tests: 各スライスで足すテスト(上の AC)。
- Integration tests: `internal/cli` の org のテスト(`OrgCleanup`、`OrgMerge`、`RetiredStateDirEnv`、`OrgLedger`、`Doctor`)を通しで打つ。
- Regression tests: `./scripts/run-test.sh` 全体。既知の不安定なテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` と 30 分の timeout)で落ちたら、1 回だけ打ち直して記録する。
- Edge cases: branch の名前に空白や `'` を含むとき、`--state-dir` に空白を含むときの quote。`--branch` ありとなしの `ralph org cleanup` の中断。台帳を動かす案内(`--state-dir` ありとなし)。`~` だけ、`~user/x`。`RALPH_LOCAL_CONFIG=off`。bare repository の linked worktree。
- Evidence to capture: docs/reports/test-2026-11-01-org-ledger-gaps.md。PR #290 の test gaps 3・4 の実物の確かめは docs/evidence/ に残す。HOME の下に作るディレクトリは一意の名前にし、終わったら消す。

## Risks and mitigations

- **他のセッションとの衝突**: #269 が `internal/config/config.go`・README・`/org` skill を変えている。この計画はそのどれも変えない。tech-debt の表は中ほどの行を書き換えるだけで、末尾には足さない。
- **全文を固定したテストの書き換えが広がる**: S9 は D6 の基準で落とす。
- **driver の振る舞いが変わる**: `expandTilde` は今の振る舞いを保ち、そのことを AC2 のテストで確かめる。
- **cwd を消すテストが不安定**: D7 のとおり差し替えに切り替える。テストは `t.Chdir` を使い、並列にしない。
- **merge の中の cleanup に値が渡らない**: AC1 のテストに、merge が中で呼ぶ cleanup の拒否(`CLEANUP: skipped`)の行を入れる。
- **再実行の文の取りこぼし**: `--branch` のない形も対象にした(plan advisory 1)。S7 の前に `ralph org cleanup` と `ralph org merge` を出す文を grep で数え直し、数と対象の一覧を Progress checklist に書く。
- **台帳を動かす案内が食い違う**: `ledgerMoveStep` は元の値を付けない例外にした(plan advisory 2)。

## Rollout or rollback notes

- 出力の文と help が変わる。機械が読む行(`MERGED:`、`CLEANUP:` などの見出し)は変えない。
- 戻すときは PR を revert する。設定ファイルや台帳の形は変えない。

## Open questions

- なし(分かれ目は consult-plan-gaps で決めた)。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh。`ralph_setting` は main から build した ralph で読んだ。PATH の ralph 6.0.0 には `config get` がない)。git の状態は前後で同じ。所見 2 件は、ユーザーの指示「HIGH と MEDIUM は聞かずに計画に反映する」に従い、どちらも反映した。
  - [HIGH] 1: AC1 が `--branch` のある形だけを見ていて、`ralph org cleanup` の `--branch` のない再実行の文(`runCleanupSteps` の中断など)が既定の台帳を選ぶ。→ Scope の 395、AC1、S7、Risks、Test plan を、`--branch` のない形も含めた「再実行のコマンドすべて」に広げた。元の `--branch` の有無を保つことと、出力から取り出したコマンドを 2 つの台帳で打ち直すテストを AC1 に足した。
  - [MEDIUM] 2: 台帳を動かすよう案内する `ledgerMoveStep` に元の `--state-dir` を付けると、動かした先でなく元の場所を選ぶ。→ この文を例外にし、新しい場所の 2 つの指定のしかたを名指す形にした(Scope の 395、AC1)。図の detail-statedir-flag も直し、lint と撮り直しで確かめた。
- [x] Plan approved — 自動承認(ユーザーの指示)。2026-11-01、digest 338876ea047b。
- [x] Branch created — fix/org-ledger-gaps(base 4fef6b8c)
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
