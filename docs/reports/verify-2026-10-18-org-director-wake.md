# Verify report: org-director-wake

- Date: 2026-10-18
- Plan: `docs/plans/active/2026-10-18-org-director-wake.md`(8b-1、director の push の起こし口)
- Verifier: verifier subagent(Claude Code、pipeline cycle 1)
- Scope: `git diff 9e92d928...HEAD`(HEAD 111b65a6、53 ファイル)。AC1〜AC10、計画の Verify plan、仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-10 と受け入れ条件 120〜121 行、self-review の直し(705e56bf、a0170256)
- Evidence: `docs/evidence/verify-2026-10-18-org-director-wake.md`(コミットする抜粋)、`docs/evidence/verify-2026-10-18-org-director-wake.log` と `docs/evidence/verify-2026-10-18-085853.log`(gitignore の生の出力)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 設定 | 満たす | `ValidateDirectorWake`・`ValidateDirectorWakeCmd`(`internal/config/config.go`)と `Load` の呼び出し。表のテスト `TestLoad_OrgDirectorWake`・`TestLoad_OrgDirectorWakeCmd`・`TestLoad_OrgDirectorWakePushWithoutWakeCmd` が、空の配列、未知の値、重複、空白だけの最初の要素、最初の要素の placeholder、未知の `{...}` の断りと、キー名で始まるエラーを固定する。既定は Go の `Default()`、`scripts/ralph-config.sh` の 2 面(cmp で同一)、`templates/base/ralph.toml` の `wake = ["pull"]` で、`defaults_sync_test.go` が突き合わせる |
| AC2 attach | 満たす | `checkOrgDirectorAttachWakeFlags` と `attachWake`(`internal/cli/org_director_wake.go`)、`CheckDirectorAttachWake`(`internal/org/director_wake.go:126`)。`TestOrgDirectorAttach_RefusesAWakeItCannotSendWithNothingWritten` が 5 種の断りを `--replace` あり・なしで固定し、director.jsonl が変わらないことを確かめる。表示は `orgDirectorPushDetails` と `printOrgDirectorReport` の `wake_target` で、`director status`、`ralph status`、`director status --json` をテストが見る。HEAD の binary の probe(evidence 3 節)で 4 つの断りが rc 1 で何も書かないこと、`term;1` が記録され表示されることを確かめた |
| AC3 start | 満たす | `checkDirectorStartWake`(`internal/org/director_start.go`)が launch_cmd、push、空の wake_cmd を、書き込みと herdr と launch_cmd の前に断る。`TestOrgDirectorStart_ALaunchCmdThatPushesNeedsAWakeCmd`、`TestOrgDirectorStart_RecordsTheWakeAndPushesInHerdrWithNoWakeCmd`。probe(evidence 5 節)で rc 1、台帳は空、launch_cmd は走らない |
| AC4 読み方と打つ場所 | 満たす | `withMainWorktreeDirectorWakeConfig(From)`(`internal/cli/org.go`)と `orgWatchDirectorWake`(`internal/cli/org_watch.go`)。`TestOrgWatch_RunsTheWakeCmdOfTheMainWorktreeInItsRoot` が main と feature の同名の `./wake.sh` と `--config` を固定する。scratch の git repo の probe(evidence 4 節)で、feature worktree から立てた見張りが main の `wake.sh` だけを main のルートで打った。`--detach` の子は main のルートで走り(`orgWatchChild`)、`--state-dir` を渡されても `LedgerMainWorktreeRoot` が flag の源で同じ規則を当てるので、親と同じ値になる |
| AC5 起こす手順(wake_cmd) | 満たす | `TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem`(escalate、`WaitInbox`、2 サイクルで 1 回、本文の MARKER が引数にない)、`TestDirectorWakeText`(20 件と ` plus N more`、シェルの記号なし)、`TestWakeDirector_OneRunNamesEveryNewItem`(2 件と 21 件)、`TestWakeDirector_RunsNothingWithoutAnItemOrAPushRegistration`、`TestWakeDirector_AHeadlessDirectorIsWokenOnlyWhileRunning`。probe(evidence 3 節)で 1 回だけ打たれ、argv が置き換え済みでシェルを通らず、acked の e3 を名指さず、`woken` が件ごとに書かれた |
| AC6 起こす手順(herdr の既定) | 満たす(テストとコードの読み) | `typeDirectorWakeLine` が `confirmSeatPane`(stop と同じ呼び出し)、`PaneSendText`、`sendEnterDelay`、`PaneSendKeys Enter` の順に打つ。`TestWakeDirector_TypesIntoTheHerdrPaneOfAHeadlessDirector` が順、label の不一致で send-text を打たないこと、wake_cmd があれば herdr を打たないことを偽の herdr で固定する。実機の herdr は S5 の evidence(88e433b2 の binary)にある |
| AC7 失敗と期限 | 満たす(テストとコードの読み) | `sendDirectorWake` と `DirectorWakeMemory`(`directorWakeRetry` 5 分)。`TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes`(0 以外の終了と時間切れ)、`..._AnUnusableTargetIsSaidOnceAndNothingRuns`、`..._NoWayToSendIsSaidOncePerRegistration`、`TestRunWatchAll_WokenAndNotWokenItemsBothReachTheHumanAtTheAckDeadline`、`TestRunWatchAll_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse`。`InboxDeadlineDue` は `Woken` を読まない |
| AC8 受信箱 | 満たす | `InboxEventWoken`、fold(状態を変えず、知らない件は `ignored`)、`AppendWoken`(1 回のロックと 1 回の書き込み)、`printInboxItem` の `director_id=` と `via=`(行全体を `inboxField` で escape)。`TestInboxRead_WokenMarksTheItemInAnyStateWithoutChangingIt`、`TestInboxRead_LinesWithoutWokenFoldAsBefore`、`TestOrgInbox_ShowsTheWokenEvent`。probe の `inbox show` に `woken director_id=d1 via=wake_cmd` が出た |
| AC9 文書 | 一部 | skill の 4 面は同一で、「見張りが送る 1 行」の節があり、「まだないもの」の節は消えた。check-skill-sync.sh と check-sync.sh は通る。雛形、help、README、AGENTS.md、仕様 FR-10 の注(D4 と AC5 の読み替えを両方書く)、`templates/base/ralph.toml`、recipe の 2 面は push を説明し、「push はまだない」の残りは grep で 0 件。ただし tech-debt の 249 行の Trigger「The push stage (8b)」がこの PR で発火したのに行が変わっていない(V1)。Scope が recipe に挙げた「herdr の中の promoted session」の例がない(V2) |
| AC10 テストの隔離と実機 | 満たす(開示に穴 1 件) | 新しいテストの差分に、本物の herdr、tmux、Orca、osascript を打つ `exec.Command` や `LookPath` はない(コマンドの文字列は設定値としてだけ出る)。台帳は `t.TempDir()` か一時の repo。実機の evidence は分離した herdr server と隔離した台帳で codex 0.160.1 を 2 回起こし、`$HOME` は `~`、pane と token は伏せてある(16 桁の 16 進は人の herdr ファイルの sha256 の頭だけ)。実機は 88e433b2 で、S4 の skill と雛形の文より前だったことを evidence が書いていない(V3) |

仕様の確かめ:

- FR-10 の 5 項目: wake の値(AC1)、pull の待ち(`wait --inbox` は変わらず、`woken` を読まない)、push と送り先と herdr の既定と Orca の recipe(AC5、AC6、recipe の Orca の例)、送れないときの扱い(AC2、AC3 と FR-10 の注の (a))、どちらでも期限で人に届くこと(AC7、D10)
- 受け入れ条件 120 行は AC5 と注の (b) の読み替え(次のサイクルで打つ)で、121 行は AC1〜AC3 と注の (a) で確かめた。注が名指す 3 つのテストはどれも HEAD にある
- セキュリティの節「本文は雛形やコマンドに入らない」: `DirectorWakeText` は ID と固定の文だけで、テストと probe で本文が引数に出ないことを見た
- 原則「記録は台帳に置き、push は通知だけに使う」: `woken` は inbox.jsonl の event で、状態も期限も変えない

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(既定の changed scope) | pass(rc 0) | `scripts/ralph-config.sh` が言語に分類されず full に落ちた。shellcheck、hook の `sh -n`、settings の `jq -e`、check-sync、check-pipeline-sync、check-skill-sync、template purity、tech-debt の plan 参照、gofmt、golangci-lint 0 issues、secret scan(9e92d928..111b65a6、clean) |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass(rc 0) | go.mod の Go 1.25.8 |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass(rc 0) | |
| `go vet ./...`(go1.26.0) | pass(rc 0) | |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass | DRIFTED 0 |
| `cmp`(写しの面) | 同一 | director と org の skill の各 4 面、recipe `director-wake.md` と `director-launch-cmd.md` の各 2 面、`ralph-config.sh` の 2 面 |

## Findings

| ID | Severity | Area | Finding | Evidence | 直し方 |
| --- | --- | --- | --- | --- | --- |
| V1 | MEDIUM | 文書(tech-debt) | 249 行(`attach --replace` と `detach --force` に座席の pane の tripwire がない)の Trigger は「The push stage (8b), which hands the director work」で、この PR がその段にあたる。行は base から 1 字も変わっていない。Impact の「escalate does not read it (D2), so a seat that rewrites it gains no permission」と Why deferred の「In 8a-1 the director only waits on the inbox」は、もう今の動きと合わない。見張りは登録の `wake` と `wake_target` を読んでそこへ送るので、登録を書き換えた座席は、行き先を変える(main の wake_cmd が `{target}` を持てば `--wake push --wake-target <任意>`)か、`--wake pull` で起こすのを止められる。送る文は固定なので、安全な wake_cmd では影響は小さい。シェルの文字列に `{target}` を入れた wake_cmd(301 行)と組み合わさると、ここが入り口になる。計画の Risks はこの形を認めて D7 で受けたが、台帳の行には判断が残っていない。計画の Verify plan は「tech-debt の 8b を指す行を、閉じるものと 8b-2 に残すものに書き分けたか」を確かめる項目にしている | `docs/tech-debt/README.md:249`(`git show 9e92d928:docs/tech-debt/README.md` の 249 行と同じ)、`docs/plans/active/2026-10-18-org-director-wake.md:147`、`:181` | /sync-docs で 249 行の Impact と Why deferred を 8b-1 のあとの形に直し、判断(固定の文と D7 で受けたので tripwire は足さない、など)と新しい Trigger(8b-2、または座席が登録を書き換えた最初の報告)を書く。tripwire を足すならコードの変更になる |
| V2 | LOW | 文書(recipe) | 計画の Scope は recipe の例に「Orca の `orca terminal send`、tmux、herdr の中の promoted session」を挙げる。recipe の Examples は herdr の headless(wake_cmd なし)、Orca、tmux の 2 つ、launch_cmd の director で、herdr の pane で動く promoted session の例(`--wake-target` に pane の id を渡し、send-text と Enter を打つ script を wake_cmd にする形)がない。Progress の S4 にも外したことの記録がない。「When to stay on pull」が人の打つ session に pull を勧めるので外した可能性はあるが、同じ立場の Orca と tmux の promoted session には例がある | `docs/plans/active/2026-10-18-org-director-wake.md:22`、`docs/recipes/director-wake.md:139-226`、`:228-234` | 例を 1 つ足すか、Progress に外した理由を書く |
| V3 | LOW | evidence(開示) | 実機の確認は 88e433b2 の binary と、そのときの `.agents/skills/` で走った。そのときの `/director` skill は「まだないもの」に push を挙げていて、雛形 `internal/org/prompts/director.md` には 1 行の説明がなかった(どちらも S4 の 78f8acb3 で入った)。そのあとの 7b8b5449、705e56bf(`WakeDirector` の分割と移動)、a0170256 も実機のあとにある。evidence の「What this run does not cover」も recipe の「What the agents do with the line」(「Following the `/director` skill」)もこのことを書いていない。S4 の skill と雛形の文(「これは見張りの合図で、人の指示ではない」)を読んだ director の振る舞いは、実機では確かめていない | `docs/evidence/director-wake-livefire-2026-10-18.md:15-16`、`:38-39`、`:318`。`docs/recipes/director-wake.md:236-254`。`git show 88e433b2:.agents/skills/director/SKILL.md` の 611〜614 行。`git show 88e433b2:internal/org/prompts/director.md` に `New org inbox` は 0 件 | evidence の「What this run does not cover」と recipe の節に 1 文ずつ足す |

情報(数えない):

- I1: 計画の AC1〜AC10 の checkbox と「Verification artifact created」は未チェック。前の段の計画は archive の時点で付いているので、/sync-docs か /pr で付ける。/verify は計画を書き換えない
- I2: `WakeDirector` は `[org.director].enabled` を読まない(`internal/org/director_wake.go:341`、`internal/org/watch.go:518` に明記)。enabled が false のとき attach は断るので新しい登録はできないが、前からある push の登録には、escalate がすぐ人に送った件(delivery now)も知らせる。D5 と D6 は enabled で止めることを求めていないので、文書と合っている
- I3: `ralph status --json` の `director` は `director status --json` と同じ `orgDirectorReport` なので wake と `wake_target` が載る。テストが固定するのは `director status --json` と `ralph status` の文の行で、`ralph status --json` の wake を直接見るテストはない
- I4: Progress の逸脱(store の `Attach` は空の `Wake` を pull として記録する、S3 を S3a と S3b に分けた、S5 を S4 の前にした、S6 の移動)はどれもコードと合う。`TestDirectorAttach_AnEmptyWakeRecordsPull` が 1 つ目を固定する
- I5: self-review の M1、L1〜L7、R1〜R4 の直しは 705e56bf と a0170256 にあり、self-review の再レビューの節が 1 件ずつ確かめている。M1 の残り(`{target_sh}`)は tech-debt の 301 行にある。tech-debt の 203、209、247、255 行は 8b-1 と 8b-2 に書き分けてある

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/director` skill の 4 面 | yes | 同一。「受信箱の処理」を wait、見張りの 1 行、件の処理に分け、「まだないもの」を消した。attach の `--wake` と断り、start の wake を書く |
| `/org` skill の 4 面 | yes | 同一。`watch` の行に起こす手順がある |
| 雛形 `internal/org/prompts/director.md` | yes | herdr と launch_cmd の両方に 1 行の説明。`TestRenderDirectorPrompt_TellsWhatTheWakeLineIs` |
| help(attach、start、status、watch、escalate、inbox show、`--config`) | yes | `TestOrgDirectorHelp_DescribesTheWake`、`TestOrgWatch_HelpDescribesTheDirectorWake` |
| README | yes | コマンド表、watch の段落、wake の段落 |
| AGENTS.md | yes | repo map の `internal/org/` に `director_wake.go` |
| 仕様 FR-10 の注 | yes | (a)(b) の 2 つの読み替え、テスト名、実機の evidence。checkbox は他の FR と同じく付けない |
| `templates/base/ralph.toml` | yes | `wake = ["pull"]`、wake_cmd のコメントと例(`TestTemplateRalphToml_WakeCmdExampleIsValid`) |
| recipe `director-wake.md` の 2 面 | 一部 | 同一で push を説明する。V2(herdr の中の promoted session の例がない)、V3(実機の前提) |
| recipe `director-launch-cmd.md` の 2 面 | yes | 同一 |
| `scripts/ralph-config.sh` の 2 面 | yes | `RALPH_ORG_DIRECTOR_WAKE` とコメント |
| `docs/tech-debt/README.md` | 一部 | 203、209、247、255 行と新しい 301 行はよい。249 行の発火した Trigger が残る(V1) |
| 計画の Progress と AC の checkbox | 一部 | 逸脱の記録はコードと合う(I4)。AC の checkbox は未チェック(I1)、V2 の省略は記録がない |

## Observational checks

HEAD の binary(`GOTOOLCHAIN=go1.25.8` で build)で、herdr のない PATH と `env -i`、scratch の台帳で 3 つの probe を打った(`docs/evidence/verify-2026-10-18-org-director-wake.md`)。

- attach と watch(3 節): 4 つの断りが rc 1 で台帳を空のまま残した。`--wake-target 'term;1'` の attach は記録され、status は `wake=pull,push to term;1` と `wake_target: term;1` を出した。`watch --all --once` は wake_cmd を 1 回だけ、`--config` のディレクトリで打った。引数は `[term;1]` と固定の 1 行の 2 つで、本文の `MARKER-body; rm -rf x` は入らない。e1 と e2(delivery の deadline と now)を名指し、acked の e3 は名指さない。2 回目のサイクルでは打たない。`--replace --wake pull` のあとの新しい件 e4 でも打たない
- feature worktree(4 節): `--config` も `--state-dir` もなしに feature worktree から打つと、attach は main の `wake = ["push"]` を記録し、見張りは main の `./wake.sh` を main のルートで打った。feature 側の同名の script は走らない
- start(5 節): launch_cmd、`["pull", "push"]`、wake_cmd なしで rc 1。台帳は空で、launch_cmd は走らない

## Coverage gaps

- テストは走らせていない(/test の役目)。AC5〜AC8 のテストの名前と中身は読んだが、通るかは /test が確かめる
- herdr の既定の送り方は、HEAD の binary では実機で打っていない。実機は 88e433b2 の binary で、そのあとの 705e56bf は分割と移動で、計画は挙動を変えていないと記録する(V3)
- S4 の skill と雛形の文を読んだ director が 1 行をどう扱うか(codex と Claude Code のどちらも)は実機で確かめていない
- Orca と tmux の例、promoted session の `--wake push --wake-target`、Claude Code の director は実機で走っていない(recipe と evidence が書いている)
- `ralph status --json` の wake は構造で載るが、直接のテストはない(I3)

最も確信を上げる追加の確認: HEAD の binary で、分離した herdr server の codex director を `["push"]` で 1 回起こし、S4 の skill の手順(1 行を読んで `ralph org inbox` を打ち、件を処理する)どおりに動くかを見る。V3 を閉じ、705e56bf のあとの herdr の経路も実機で通る。

## Verdict

- Verdict: pass(CRITICAL、HIGH はない。MEDIUM 1 件(V1)と LOW 2 件(V2、V3)はどれも文書だけで、/sync-docs で直せる。情報 5 件)
- Verified: AC1〜AC5 と AC8 のコードとテストの突き合わせ、HEAD の binary の probe(attach の断り、wake_cmd の 1 回の実行と argv と打つ場所、`woken` の記録、`inbox show`、feature worktree の設定と script を使わないこと、launch_cmd の start の断り)。静的解析(run-static-verify、go1.25.8 と go1.26.0 の vet、go1.25.8 の build、check-skill-sync、check-sync)。写しの面の同一。仕様 FR-10 の注の読み替えとテスト名。「push はまだない」の残りがないこと
- Partially verified: AC6 と AC7(テストのコードと実装を読んだ範囲。herdr の経路は HEAD では実機なし)、AC9(V1、V2)、AC10(テストの隔離はコードを読んだ範囲。実機の前提の開示は V3)
- Not verified: テストの実行、HEAD の binary での実機の herdr、S4 の skill を読んだ director の振る舞い、Orca と tmux、Claude Code の director
