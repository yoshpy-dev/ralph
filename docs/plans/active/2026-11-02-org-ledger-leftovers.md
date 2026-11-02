# org-ledger-leftovers

- Status: Approved
- Approved: 2026-11-02 sha256:78557bfbc265
- Owner: Claude Code
- Date: 2026-11-02
- Related request: 2026-11-02 のユーザーの依頼「残りの穴も直せますか？」のうち、PR #296(#290 の穴)が残した穴と、そのときに当てた tech-debt の trigger(docs/tech-debt/README.md の 388・390・391・395・420 行)
- Related issue: N/A
- Type: fix
- Branch: fix/org-ledger-leftovers

## Objective

PR #296 が残した、`ralph org merge` と `ralph org cleanup` の周りの穴を直す。

- ほかの動詞(merge-order、inbox、status、disband)を案内する文が、`--state-dir` で名指した台帳を落とす。
- 打つコマンドの語を quote する関数が 2 つあり、同じ値が 1 つの出力に 2 通りの形で出る。
- cleanup の help の 1 文が、`--branch` なしで打ったときの理由の形と食い違う。
- テストが打たない分岐と、作業ディレクトリに空白があると落ちるテストがある。

あわせて、`internal/org/cleanup.go` を変えるときに払う約束になっている tech-debt(390・391)を払う。

PR #297(#291 の穴)の残りは、この PR の merge のあとに別の計画(δ)で直す(D1)。

## Scope

行番号は main 31812a3b の docs/tech-debt/README.md のもの。

- **390: 長すぎるファイルの分割**(最初のスライス、移すだけ)
  - `internal/org/cleanup.go`(1119 行)の文を組み立てる関数(`branchMovedSteps`、`keptBranch`、`registrationAtRisk`、`ledgerMoveStep`、`fetchHeadCmd`、`cleanupShellWord`、`cleanupGitCmd`、`orgRetryCommand`、`cleanupCommand` など)を `internal/org/cleanup_text.go` に移し、`cleanup.go` を 800 行以下にする。
  - `internal/cli/org_cleanup_test.go`(1280 行)を 2 つのファイルに分け、どちらも 800 行以下にする。新しいファイルの名前は、S1 で移すテストの中身に合わせて決める。
  - 中身は変えない。
- **388・420(b): quote の関数を 1 つにする**
  - org の層の `cleanupShellWord`(`[A-Za-z0-9_./:@+,-]` を裸のまま)を export し(`org.ShellWord`)、CLI の層の `shellQuoteIfNeeded`(`[A-Za-z0-9_./-]` だけ裸)と `shellSafeUnquoted` を消して、その 6 か所の呼び出し(`org.go` の `orgReadCommandHint`、`org_legacy_ledger.go` の 4 か所、`org_cleanup.go` の `cleanupRetryCommand`)を `org.ShellWord` に替える(D3)。
  - 変わるのは、`: @ + ,` を含む値の CLI の出力の形だけ(quote なしになる)。shell が特別に読む文字はどちらの関数も quote するので、打ったときの動きは変わらない。
- **395 の残り: ほかの動詞の案内に `--state-dir`**
  - `ralph org merge` と `ralph org cleanup` が出す文のうち、`ralph org merge-order`(`--all` 付きを含む)、`ralph org inbox`、`ralph org status --org-id`、`ralph org disband --org-id` を案内するものすべてに、`--state-dir` が渡されたときだけ `--state-dir <dir>` を付ける(D5)。
  - 対象は main 31812a3b で 15 か所ほど: `internal/org/cleanup.go:271`、`merge.go` の 274・286・517・553・721 行、`mergeenforce.go:179`、`merge_run.go` の 418・441・517・567 行、`merge_wait.go:139-140`、`internal/cli/org_merge.go` の 224・340 行、`internal/cli/org_cleanup.go:342`。実装の前に、行をまたぐ文も含めて数え直す。
  - 値は `CleanupParams` と `MergeParams` の `HintStateDir`(PR #296 で足したもの)から取る。
  - help の説明文と flag の説明(`org_merge.go:47・49・263`、`org_cleanup.go:185`)は、この打ち方の案内ではないので対象にしない。
- **420(a): cleanup の help の 1 文**
  - `ralph org cleanup --help` の「run ralph org cleanup --branch <branch> again; the reason names that command」を、理由が名指すコマンドを打つ、と書き直す(D2)。理由のコマンドは 2 通りある。branch の側の理由(未追跡のファイル、regate.lock、org が動いているなど、org の層の `cleanupAgain`)は `ralph org cleanup --branch <branch>`。PR や origin を読めなかった理由(CLI の層の `cleanupRetryCommand`)は打ったとおりの形で、`--branch` は付けて打ったときだけ付く。どちらも `--state-dir` は渡したときだけ付く。help はこの 2 通りを分けて書く。help を固定したテストを一緒に直す。出力の振る舞いは変えない。
- **391: cleanup の文とコメント**
  - (a) 記録がないときの拒否の文(`recordGone`)の「remove it by hand」を「remove what is left by hand」にする。
  - (b) `registrationAtRisk` のコメントを、detached HEAD だけを名指す形にする。
- **420(c): 空白で落ちるテスト**
  - `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead` の期待の `git -C` の語を、コードと同じ quote で作る。空白を含む作業ディレクトリで通ることをテストで確かめる。
- **420(d): テストの穴**
  - `cleanupRun.run` で、feature worktree のパスにディレクトリでないものがあるときの拒否(`st.notDir`)を打つテスト。
  - `cleanupRun.remove` で、登録されていて実体のない worktree の `git worktree remove` が失敗する分岐を打つテスト。
  - `checkRetiredOrgStateDirEnv` で、`org.LedgerLocalConfigPath` が `ok == false` を返すときの分岐を打つテスト(関数を直接呼ぶ)。
  - Linux だけで走るテスト(ほかの OS では `t.Skip`)。作業ディレクトリを消したあとに、本物の `absPath`(org)と `retiredOrgStateDirEnvAbs`(cli)が失敗することを確かめ、PR #296 の D7 の前提を CI(ubuntu-latest)で確かめる(D7)。
- **tech-debt の表**: 388・390・391・395・420 行を、直した形に書き換える(/sync-docs)。

## Non-goals

- 396(c)(`internal/config/config.go` の `Load` と `StateDir` の doc)と 397 の残り(README の `[org].state_dir` の段落と `/org` skill の 4 つの写し)。どちらの tech-debt の行も trigger が「#269 の merge のあと」で、#269 はまだ OPEN(D1)。δ の /plan のときに #269 が CLOSED なら δ に入れ、そうでなければ #269 を待ってから別の計画で直す。
- PR #297(#291)の穴(tech-debt 421 行)。δ で直す。
- tech-debt 386 行の 4 つのテストファイル(`internal/cli/org_merge_test.go`、`internal/org/merge_run_test.go`、`internal/org/director_skill_sync_test.go`、`internal/org/cleanup_test.go`)の分割(D6)。この PR は新しいテストを、PR #296 で作った `internal/org/cleanup_retry_state_dir_test.go` と `internal/cli/org_retry_state_dir_test.go`、または新しいファイルに置き、4 つのファイルには足さない。
- `ralph org regate` の打ち直しの文(tech-debt の `gateWording.again` の行、#264 の側)。
- ほかに 800 行を超えているファイル(`internal/cli/org.go` など)の分割。

## Assumptions

- `--state-dir` は `ralph org` の persistent flag なので、merge-order・inbox・status・disband のどれも受け付ける(`internal/cli/org.go:62`)。
- `cleanupShellWord` と `shellQuoteIfNeeded` は、quote するときの形(POSIX の single quote と `'\''`)が同じ。違うのは裸のまま残す文字の集まりだけ。
- `internal/org` から `internal/cli` は import できないので、共有の関数は org の層に置く。
- CI の `verify.yml` は ubuntu-latest で `go test` を打つ。
- #269(セッション A)の branch は、この計画が変えるファイルのうち `docs/tech-debt/README.md` だけを変える(`git diff --stat origin/main...origin/feat/org-cross-review` で確かめた)。Go のファイルは重ならない。

## Affected areas

- `internal/org/`: `cleanup.go`、新しい `cleanup_text.go`、`merge.go`、`merge_run.go`、`mergeenforce.go`、`merge_wait.go`、テスト(`cleanup_retry_state_dir_test.go` と新しいファイル)
- `internal/cli/`: `org.go`(`shellQuoteIfNeeded` を消す)、`org_legacy_ledger.go`、`org_cleanup.go`(help と文)、`org_merge.go`、`org_cleanup_test.go`(分ける、420(c))、`org_retry_state_dir_test.go` と新しいテストファイル
- `docs/tech-debt/README.md`(388・390・391・395・420 行)

## Visual review

- Page: `.harness/state/plan-visual/org-ledger-leftovers.html`(かんたんな図 2 つ、詳しい図 4 つ)
- Lint: `./scripts/plan-visual.sh lint` は exit 0。
- Self-check: 全体、overview、easy-quote、detail-overview、detail-quote、detail-hint、detail-split を `shot` で撮って読んだ。easy-quote の補足の文が「どの行が引用符つきか」を取り違えていたので直した。lead の `--state-dir` が行の途中で折れていたので折れないようにした。撮り直して、すべてきれい。

## Design decisions

- **D1: #290 と #291 の系列で 2 本に分け、#269 待ちの文書は外す。** この計画(γ)は #290 の系列、δ は #291 の系列。396(c) と 397 の残りは tech-debt の trigger が「#269 の merge のあと」で、PR #296 の D4 は「#269 が変えている最中」を理由に外した。hunk が重ならないのは衝突の話で、その理由は消えない(consult-plan-leftovers)。#269 の CLOSED はバックグラウンドで待ち、δ の /plan のときに判定する。
- **D2: 420(a) は help の文を直す。** 別案は「`--branch` なしで打ったときの理由を `--branch <branch>` の形にする」。PR #296 の AC1 は「`--branch` の有無を保つ」と決めていて、それを裏返すことになる。help の 1 文と、それを固定したテストだけで閉じる。help は、理由のコマンドが 2 通りあること(branch の側の理由は `--branch <branch>`、PR や origin を読めなかった理由は打ったとおりの形)を分けて書く。「どの理由も打ったとおりの形」とまとめると、`--branch` 付きを出す大半の理由と食い違う(plan advisory 1)。
- **D3: quote は org の層の関数に寄せる。** org の層は cli を import できない。CLI の側に寄せると、cleanup の文で `: @ + ,` が quote され、org の層の固定文のテストが多く変わる。org の側に寄せると、変わるのは CLI の 6 か所と、その出力を固定したテストのうち `: @ + ,` を含む値のものだけ。
- **D4: 390 の分割は最初の移すだけのスライスにする。** 390 の trigger は「次に cleanup.go を変えるとき」で、3 回繰り越している。この PR は cleanup.go を 4 か所で変える。移すだけのコミットにし、関数の本体が変わらないことと、`go test -list` の名前が前後で同じことを証跡にする。`internal/cli/org_cleanup_test.go` も 420(c) で変えるので、同じスライスで分ける。
- **D5: `--state-dir` は、渡されたときだけ付ける。** PR #296 の `orgRetryCommand` と同じ規則。`ralph.local.toml` の `[org].state_dir` で決めた台帳は、素のコマンドでも同じファイルを読む。
- **D6: tech-debt 386 行の 4 つのファイルは分けない。** この PR はその 4 つにテストを足さない。D3 で 4 つのどれかの固定文が変わるときは、期待の 1 行を直すだけにし、386 行に繰り越しの注を書く。4 つのうち `director_skill_sync_test.go` はこの PR の範囲と関係がなく、全部を分けると移すだけの diff が 5,000 行を超える。
- **D7: Linux の os.Getwd は、Linux だけで走るテストで CI に確かめさせる。** 手元は darwin で、cwd を消しても `os.Getwd` は失敗しない(PR #296 の S8 で実測)。テストは `runtime.GOOS != "linux"` で skip し、CI の ubuntu-latest で走る。
- Critical forks: D1〜D7 は consult-plan-leftovers に聞いたうえで選んだ(ユーザーの指示「計画の分かれ目は consult に聞いて推奨案を自分で選ぶ」)。

## Acceptance criteria

- [ ] AC1(390): `internal/org/cleanup.go` は 800 行以下で、文を組み立てる関数は `internal/org/cleanup_text.go` にある。`internal/cli/org_cleanup_test.go` は 2 つのファイルに分かれ、どちらも 800 行以下。このスライスのコミットは移すだけで、移した関数とテストの本体は 1 文字も変わらない(移す前後の関数ごとの本体を比べた結果を証跡に残す)。`go test -list` のテストの名前は前後で同じ。
- [ ] AC2(388、420(b)): 打つコマンドの語を quote する関数は `org.ShellWord` の 1 つで、`shellQuoteIfNeeded` と `shellSafeUnquoted` はない。`feat/a+b`、`a:b`、`a@b`、`a,b`、空白を含む値、`'` を含む値について、CLI の層の出力(`cleanupRetryCommand` と `orgReadCommandHint`)と org の層の出力(`orgRetryCommand`)が同じ形で書くことを、表のテストで確かめる。
- [ ] AC3(395): `--state-dir <dir>` を付けて打った `ralph org merge` と `ralph org cleanup` が出す、merge-order・inbox・status・disband を案内する文は、どれも `--state-dir <dir>`(`org.ShellWord` で quote)を含む。`--state-dir` なしで打つと、個人用のファイルで台帳を決めたときも含めて、どれも含まない。数え直した一覧を Progress checklist に書き、文の組ごとにテストがある。少なくとも 1 つのテストは、2 つの台帳を用意し、出力から取り出した `ralph org merge-order` のコマンドをそのまま打って、最初に選んだ台帳だけが読まれることを確かめる。
- [ ] AC4(420(a)): `ralph org cleanup --help` は、SKIPPED の理由のコマンドが 2 通りあることを言う。branch の側の理由は `ralph org cleanup --branch <branch>`、PR や origin を読めなかった理由は打ったとおりの形(`--branch` は付けて打ったときだけ)で、どちらも `--state-dir` は渡したときだけ付く。help を固定したテストがその文を見る。`--branch` なしで打った 2 つの場合(未追跡のファイルの拒否と、PR を読めなかった拒否)のテストがあり、前者は `--branch <branch>` を含み、後者は含まないことを確かめる(どちらも今の振る舞いのまま)。
- [ ] AC5(391): 記録がないときの拒否の文は「remove what is left by hand」を含む。`registrationAtRisk` のコメントは detached HEAD だけを名指す。
- [ ] AC6(420(c)): `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead` の期待はコードと同じ quote で作られ、空白を含む作業ディレクトリでも通る(そのケースがある)。
- [ ] AC7(420(d)): `st.notDir` の拒否、`git worktree remove` の失敗、`LedgerLocalConfigPath` の `ok == false` の分岐を打つテストがある。Linux だけで走るテストが、作業ディレクトリを消したあと本物の `absPath` と `retiredOrgStateDirEnvAbs` が失敗することを確かめ、CI(ubuntu-latest)で通る。
- [ ] AC8: tech-debt の 388・390・391・395・420 行が、直した形に書き換わっている。`./scripts/run-verify.sh` と `./scripts/run-test.sh` が通る。

## Implementation outline

各スライスの verify の `<go test>` は `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test` を指す。`internal/cli` のテストは全部で 10 分を超えるので、スライスでは `-run` で関係するテストに絞る。全体は /verify と /test で打つ。

1. S1(AC1): `cleanup.go` の文を組み立てる関数を `cleanup_text.go` へ、`org_cleanup_test.go` の半分を新しいテストファイルへ移す。移すだけ — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && <go test> -run 'Cleanup' ./internal/org/ && <go test> -run 'OrgCleanup' ./internal/cli/ && test "$(wc -l < internal/org/cleanup.go)" -le 800 && for f in internal/cli/org_cleanup*_test.go; do test "$(wc -l < "$f")" -le 800 || exit 1; done`
2. S2(AC2): `cleanupShellWord` を `org.ShellWord` として export し、CLI の 6 か所をそれに替え、`shellQuoteIfNeeded` と `shellSafeUnquoted` を消す。表のテストを足し、変わる固定文を直す — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && <go test> -run 'Cleanup|Merge|ShellWord' ./internal/org/ && <go test> -run 'OrgCleanup|OrgMerge|OrgRead|Legacy|RetiredStateDirEnv|RetryStateDir|ShellWord' ./internal/cli/`
3. S3(AC3): ほかの動詞を案内する文に `--state-dir` を付ける。文の組ごとのテストと、出力の merge-order を 2 つの台帳で打ち直すテストを足す — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && <go test> -run 'Cleanup|Merge|RetryStateDir|HintStateDir' ./internal/org/ && <go test> -run 'OrgCleanup|OrgMerge|RetryStateDir|HintStateDir' ./internal/cli/`
4. S4(AC4、AC5): cleanup の help の文、`recordGone` の文、`registrationAtRisk` のコメント。固定文のテストを直す — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && <go test> -run 'Cleanup' ./internal/org/ && <go test> -run 'OrgCleanup' ./internal/cli/`
5. S5(AC6、AC7): HeadNotHere の期待の quote と空白のケース、`st.notDir`・`git worktree remove` の失敗・`ok == false` のテスト、Linux だけのテスト — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && <go test> -run 'Cleanup|AbsPath|Getwd' ./internal/org/ && <go test> -run 'OrgCleanup|RetiredStateDirEnv|Getwd' ./internal/cli/`

tech-debt の表の書き換え(AC8)は /sync-docs で行う。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck)。`./scripts/check-sync.sh`。
- Spec compliance criteria to confirm: AC1〜AC8。AC1 は移す前後の関数の本体を比べる。AC3 は `merge-order`・`ralph org inbox`・`ralph org status`・`ralph org disband` を grep で数え、出力の文がすべて当たっていることを見る。
- Documentation drift to check: `ralph org cleanup --help` と `ralph org merge --help` の文、README の cleanup と merge の段落(読むだけ。README は #269 が変えているので、直す所があれば tech-debt に書く)、docs/tech-debt/README.md の 386・388・390・391・395・420 行、AGENTS.md の地図の `internal/org/` の行(`cleanup_text.go` を名指す必要があるか)。
- Evidence to capture: docs/reports/verify-2026-11-02-org-ledger-leftovers.md。

## Test plan

- Unit tests: 各スライスで足すテスト(上の AC)。
- Integration tests: `internal/cli` の org のテスト(`OrgCleanup`、`OrgMerge`、`OrgRead`、`RetiredStateDirEnv`、`Legacy`)を通しで打つ。
- Regression tests: `./scripts/run-test.sh` 全体。既知の不安定なテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` と 30 分の timeout)で落ちたら、1 回だけ打ち直して記録する。
- Edge cases: `: @ + ,`、空白、`'` を含む branch と `--state-dir`。`--state-dir` ありとなし、`--branch` ありとなし。個人用のファイルで決めた台帳。空白を含む作業ディレクトリ。
- Evidence to capture: docs/reports/test-2026-11-02-org-ledger-leftovers.md。build した binary で、`--state-dir` を付けた `ralph org cleanup` が出す merge-order の案内を写して打ち、選んだ台帳を読むことを docs/evidence/ に残す。Linux のテストは CI の結果を証跡にする。

## Risks and mitigations

- **移すだけのスライスに変更が混じる**: 関数ごとに本体を比べる(AC1)。`go test -list` の名前も比べる。
- **quote を寄せて固定文のテストが広く変わる**: D3 で org の側に寄せた。変わるテストを S2 で数えて Progress checklist に書く。386 行の 4 つのファイルの固定文が変わるときは期待の 1 行だけを直す(D6)。
- **案内の文の取りこぼし**: S3 の前に、行をまたぐ文も含めて数え直す(`merge-order` を単独で grep する)。
- **merge の中の cleanup の文に値が渡らない**: AC3 のテストに、merge が中で呼ぶ cleanup の文を入れる。
- **他のセッションとの衝突**: #269 と重なるのは `docs/tech-debt/README.md` だけ。この PR は表の中ほどの行を書き換えるだけにし、末尾には足さない。衝突したら両方の変更を残して直す。
- **Linux だけのテストが CI で落ちる**: 落ちたら PR #296 の D7 の前提が崩れたことになる。原因を見て、`absPath` の差し替えのテストを残したまま、Linux のテストの期待を事実に合わせ、tech-debt に書く。

## Rollout or rollback notes

- 出力の文と help が変わる。機械が読む見出し(`MERGED:`、`CLEANUP:`、`CLEANED:`、`SKIPPED:`、`MANUAL:`)は変えない。
- 戻すときは PR を revert する。設定ファイルや台帳の形は変えない。

## Open questions

- なし(分かれ目は consult-plan-leftovers で決めた)。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh。`ralph_setting` は main から build した ralph で読んだ。PATH の ralph 6.0.0 には `config get` がない)。git の状態は前後で同じ。所見 1 件は、ユーザーの指示「HIGH と MEDIUM は聞かずに計画に反映する」に従って反映した。
  - [MEDIUM] 1: AC4 の help の文「理由のコマンドは打ったとおりの形」は、`--branch` なしの一括の cleanup でも未追跡のファイルなど branch の側の理由が `--branch <branch>` を出す今の振る舞い(`TestOrgCleanup_RefusalsAreSkippedLines`)と食い違う。→ Scope の 420(a)、D2、AC4 を、理由のコマンドが 2 通り(branch の側は `--branch <branch>`、PR や origin を読めなかったときは打ったとおり)あることを書く形に直し、`--branch` なしで打った 2 つの場合のテストを AC4 に足した。図は help の中身を描いていないので描き直しは要らない。
- [x] Plan approved — 自動承認(ユーザーの指示)。2026-11-02、digest 78557bfbc265。分かれ目(D1〜D7)は consult-plan-leftovers の判定(「直してから進める」: #269 待ちの文書を外す、386・391 の trigger の扱いを書く)を反映して選んだ。
- [x] Branch created — fix/org-ledger-leftovers(base 31812a3b)
- [x] Implementation started
  - S1(AC1): 3b54f1b1。`cleanup.go` 1119 → 790 行(18 の宣言を `cleanup_text.go` 341 行へ)。候補 15 個だけでは 903 行で 800 に届かず、git を打たずに読んだデータから拒否の文を作る `checkRecord`・`checkRegistration` と、葉の関数 `pathWithin` も移した。`org_cleanup_test.go` 1280 → 701 行(10 の宣言を `org_cleanup_steps_test.go` 595 行へ)。宣言ごとの byte 比較で欠落・重複・書き換えなし、`go test -list` の名前は org 1424 件・cli 979 件で前後同じ。コミット本文が `pathWithin` も「文を組み立てる」としているのは不正確(amend しない)。cleanup.go の余裕は 10 行なので、S3 以降は文の組み立てを `cleanup_text.go` に置く。implementer は sonnet。
  - S2(AC2): 47a4fda1。`cleanupShellWord` → `ShellWord`、`cleanupPlainWord` → `shellPlainWord`。CLI の `shellQuoteIfNeeded`・`shellSafeUnquoted` を消し、本番 9 か所・テスト 25 か所を `org.ShellWord` に替えた。`org_merge_test.go` の `orgLayerShellWord`・`orgPlainWord` も消した(2 つの関数の差のためだけにあった)。固定文の期待で変わったものは 0 本(`: @ + ,` を含む値の固定文がなかった)。表のテストは `internal/org/shell_word_test.go` と `internal/cli/org_shell_word_test.go`。tech-debt 386 行の 4 つのうち `org_merge_test.go`・`internal/org/cleanup_test.go`・`merge_run_test.go` は呼び出しの名前と helper の削除だけで、テストは足していない(D6、/sync-docs で 386 行に注を書く)。implementer は sonnet。
  - S2 の補い(AC2、手で直した 1 ファイル): f972c80b。implementer が範囲の外で見つけた 3 つ目の「要るときだけ quote する」関数 `directorShellWord`(`internal/org/director_stop.go`、herdr の id 用、裸の文字は `[A-Za-z0-9_.:-]`)を消し、`ShellWord` を使う。AC2 の「1 つ」は、要るときだけ quote する関数の意味で読む。いつも quote する `shellQuote` は `ShellWord` の部品で、escalate・approve・watchgate などの文が直接使うが、388 の範囲の外。`go test -run Director` は org・cli とも ok。機械的な確かめ: 不要な変更・typo・debug の残り・secret・範囲外のファイル、どれも clean。
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
