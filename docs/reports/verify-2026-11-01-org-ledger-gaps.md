# Verify report: org-ledger-gaps

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-ledger-gaps.md`(PR #290 の Known gaps と tech-debt 395〜400 行、AC1〜AC11、D1〜D7、Progress の S1〜S9 と self-review の直し)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base = origin/main 4fef6b8c、HEAD df194fa1、40 ファイル、+2899/−641)。spec の準拠、静的解析、文書のずれを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。probe は HEAD から build した binary(`<S>/ralph`、`<S>` は scratchpad のディレクトリ)と、origin のない一時の git repo で打った。作業ツリーには lead の計画の未コミットの変更(Progress の self-review の行)があり、このコミットには含めていない。
- Evidence: `docs/evidence/verify-2026-11-01-org-ledger-gaps.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。run-static-verify の全出力、check-sync、binary の probe、5 つの help、S1 の突き合わせ)。run-verify 自身のログは `docs/evidence/verify-2026-11-01-042248.log`(同じく対象外)。
- load average: probe の後で 6.11(`uptime`)。timeout には当たっていない。
- 計画の digest: `./scripts/plan-visual.sh digest` は作業ツリーの計画でも 27bf9af7 の計画でも `338876ea047b` で、`- Approved:` の行と同じ。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `--state-dir` を付けて打った cleanup と merge の再実行のコマンドはどれも `--state-dir <dir>` を含み、`--branch` の有無を保つ。付けずに打てば台帳の指定なし。台帳を動かす案内は元の値を含まない。builder ごとのテストと、2 つの台帳での打ち直しのテスト | 満たす | 下の「AC1 の数え直し」。binary の probe(evidence の probe A〜F)で、`--branch 'feat/a b' --state-dir '<S>/led ger/org'`、`--state-dir` だけ、`--branch` だけ、merge の 4 通りがそれぞれの形の再実行のコマンドを出し、個人用のファイルで台帳を決めたとき(probe E・F)は `--state-dir` がない。`ledgerMoveStep`(`internal/org/cleanup.go:529-533`)は `orgRetryCommand("cleanup", branch, "")` を使い、`--state-dir <new place>` と `[org].state_dir` の 2 つを名指す。テストは `TestCleanupRetryCommand_QuotesEachWordAndAddsTheStateDirOnlyWhenGiven`、`TestCleanupFeatureWorktree_RefusalsBeforeGitRepeatTheStateDir`・`_WorktreeRefusalsRepeatTheStateDir`・`_LedgerMoveStepNamesNoOldStateDir`、`TestMerge_RetryCommandsRepeatTheStateDir`、`TestMerge_CleanupRepeatsTheStateDir`、`TestMergeCleanupInterrupted_QuotesTheBranchInItsCommand`、`TestOrgCleanup_RetryCommandKeepsTheBranchAndTheStateDir`、`TestOrgCleanup_RefusalsAndInterruptionsNameTheStateDir`、`TestOrgCleanup_NoMainWorktreeNamesTheBranchAndTheStateDir`、`TestOrgMerge_CleanupRootNamesTheStateDir`、打ち直しは `TestOrgCleanup_RetryCommandReplaysOnTheLedgerNamedFirst` と `TestOrgMerge_RetryCommandsReplayOnTheLedgerNamedFirst`(存在と中身を読んだ。実行は /test) |
| AC2 判定と展開が `internal/config` に 1 つずつ。4 か所がそれを呼ぶ。6 つの値の表のテスト。`expandTilde` は失敗で値をそのまま返す | 満たす | `IsAbsOrHomePath`(`internal/config/homepath.go:16`)、`ExpandHomePath`(`:27`)。呼び出しは `checkLocalAgmsgHome`(`internal/config/local.go:106`)、`LocalOrgStateDirAt`(`:241`・`:245`、`LocalOrgStateDir` はこれを呼ぶ)、`agmsgHomeIsRelative`(`internal/cli/org_watch.go:446`)、`expandTilde`(`internal/org/driver/agmsg.go:178`)。非テストの Go に `HasPrefix(…, "~/")` の判定は `homepath.go` の 2 か所のほかに残っていない(grep)。6 つの値は `TestHomePath_ChecksAndExpansionsAgree`(config)、`TestAgmsgHomeIsRelative_ComplementsConfigIsAbsOrHomePath`(cli)、`TestExpandTilde_AgreesWithConfigExpandHomePath`(driver)がそれぞれ同じ 6 つを並べる。package が 3 つに分かれるので表も 3 つ。失敗の側は `TestExpandHomePath_UserHomeDirFails` と `TestExpandTilde_KeepsTheValueWhenUserHomeDirFails` |
| AC3 読めない `ralph.toml` で doctor は「parse error: … — using defaults」を出し、後の検査は `Default()` を読む | 満たす(コードとテストの存在まで) | `internal/cli/doctor.go:107-112` で `cfgErr != nil` なら `cfg = config.Default()`、warn は `:119`。doctor の中で `config.Load` を呼ぶのはこの 1 か所だけ(grep)。ファイルがないとき `config.Load` はエラーを返さない(`internal/config/config.go:886-888`)ので、`ralph.local.toml` だけがある場合の値は捨てない。テストは `TestRunDoctorOpts_LoadErrorUsesDefaults`。doctor 自体は claude と codex を探すので打っていない |
| AC4 断る条件の定数 1 つ。doctor の warn と escalate の help が含む。4 つの場合の表のテスト | 満たす | 定数 `retiredOrgStateDirRefusalCondition`(`internal/cli/org_legacy_ledger.go:75`)。doctor の `effect` は `:905` で定数から組み立てる。escalate の help は `internal/cli/org_inbox.go:85-88` で定数を `wrapHelpLines` に通す。binary の `ralph org escalate --help` の空白を詰めた文に定数が 1 回ある(evidence)。表のテスト `TestRetiredOrgStateDirRefusalCondition` は 4 つの場合を両方の access で通し、拒否か注意かと、2 つの文が定数を含むことを見る |
| AC5 `ralph doctor --help` の `--strict` の行に値の名前が出ない | 満たす | binary の help の行は `--strict             fail (exit 1) …` で、型の語がない。usage の backtick は外れた(`internal/cli/doctor.go:50`)。テストは `TestDoctorHelp_StrictFlagHasNoValueName` |
| AC6 `ralph config get --help` の exit 3 に local-only のキーの `ralph.toml` も書き、キーは `config.LocalOnlyKeys()` から作る | 満たす | `internal/cli/config.go:76-77`。binary の help の exit 3 の下に「A ralph.toml that sets a key only ralph.local.toml may set exits 3 too: org.state_dir, plan_visual.browser, plan_visual.opener, standard.base_branch, verify.base, verify.skip_branch_secret_scan.」が 74 桁以内で折り返して出る。テストは `TestConfigGet_HelpNamesTheLocalOnlyKeysOfExit3` と `TestConfigGet_WrapHelpLines` |
| AC7 既定の層で main worktree を探すのは 1 回。拒否は `ralph.local.toml` のパスに git を打たない。回数のテスト | 満たす | `ResolveOrgStateDir`(`internal/org/statedir.go:75-87`)は `defaultOrgStateDirBase`(`:96`)を 1 回呼び、その root の `ralph.local.toml` を `config.LocalOrgStateDirAt`(`:80`)で読む。`defaultOrgStateDirBase` の探し方は `config.LocalPath`(`internal/config/local.go:42-54`)と同じ順(MainWorktree → Toplevel → cwd)。拒否のパスは `org.LedgerLocalConfigPath`(`internal/cli/org_legacy_ledger.go:152`)。`!ok` のときだけ `config.LocalPath` で git を打つ(`:154`)が、`checkRetiredOrgStateDirEnv` は source が flag と config のとき先に戻り、既定の層の台帳はいつも `.harness/state/org` で終わる(`mustAbs` が失敗した相対の場合も `ledgerBaseDir` は ok)ので、`resolveOrgLedger` からは来ない(self-review L7 と同じ読み)。「1 回」は探索の回数で、`gitroot.MainWorktree` 自体は git を 2 回起動する(計画の Progress S5 に記録あり)。テストは `TestResolveOrgStateDir_LooksForItsDirectoryOnce`、`TestLedgerLocalConfigPath`、`TestOrgRetiredStateDirEnv_RefusalNamesThePersonalFileWithoutGit` |
| AC8 `local_test.go` は 800 行以下。移したテストの中身は変わらない | 満たす | `local_test.go` は 1011 → 600 行。93397d1f の前の `local_test.go` の 607 行目以降と、93397d1f の 2 つの新しいファイルを行ごとに `sort` して `comm` で突き合わせた。旧にだけある行は 0。新にだけある行は package、import、先頭のコメントだけ |
| AC9 `FeatureRepoRoot` の bare の分岐と `filepath.Abs` の失敗の 2 つの分岐のテスト | 満たす(テストの存在まで) | `TestFeatureRepoRoot_LinkedWorktreeOfBareRepositoryHasNoMainWorktree`、`TestMustAbs_KeepsThePathAsGivenWhenAbsFails`(`mustAbs` は `absPath` を通る、`internal/org/statedir.go:263-264`)、`TestOrgRetiredStateDirEnv_KeepsARelativeValueAsGivenWhenAbsFails`(`retiredOrgStateDirEnvAbs`)。差し替えにした理由(darwin では cwd を消しても `os.Getwd` が失敗しない)は Progress S8 にある |
| AC10 拒否の文が「refusing to change the org ledger:」の形で始まり、escalate は記録しない道で人に届ける | 満たす | `resolveOrgLedger` は拒否を `errOrgLedgerRefusedByRetiredEnv` で包む(`internal/cli/org_legacy_ledger.go:107`)ので、文は「<command>: refusing to change the org ledger: RALPH_ORG_STATE_DIR names …」になる。頭がコマンドのパスで guard の「org:」でないことは Progress S9 に理由がある。escalate は 2 つの sentinel のどちらでも `newOrgEscalateFallbackRuntime("")` に進む(`internal/cli/org_inbox.go:102`)。`errOrgLedgerUnresolved` を `errors.Is` で見る所はほかにない(grep)。全文を固定したテストは `org_cleanup_test.go:611`、`org_retired_state_dir_env_test.go:165`・`:289`・`:348` |
| AC11 tech-debt 395〜400 行の書き換え、run-verify と run-test が通る | 未着手(/sync-docs と /test の担当) | 計画の Implementation outline のとおり、表の書き換えは /sync-docs が行う。このブランチは `docs/tech-debt/README.md` に触れていない。run-verify の静的の部分は下の表のとおり通った。テストは /test |

### AC1 の数え直し

`origin/main` の非テストの Go で再実行のコマンドを作る所を grep で数え、HEAD の対応を確かめた。計画の Progress(S7 の前の数え直し)の一覧と一致する。

- org の層、cleanup: `cleanupAgain(p)` の呼び出し 12 か所(`internal/org/cleanup.go:205`・`:275`・`:342`・`:346`・`:437`・`:440`・`:459`・`:716`・`:760`・`:839`・`:884`・`:1006`)と、`cleanupCommand(c.p)` 2 か所(`:719` の「from outside it」、`:1014` の「goes on from what is left」)。どれも `orgRetryCommand("cleanup", p.Branch, p.HintStateDir)`(`:298`)を通る。例外の `ledgerMoveStep`(`:529`)は空の hint を渡す。
- org の層、merge: `mergeCheck.again`(`internal/org/merge.go:348`)の使用 14 か所(merge.go 13、`merge_run.go:462` 1)。`m.hintStateDir` は `newMergeCheck` で `MergeParams.HintStateDir` から入る(`merge.go:172`)。ロックの文(`merge_run.go:222`)、中断の文 `mergeCleanupInterrupted`(`:279`)、merge の中の cleanup(`:292`、`HintStateDir: p.HintStateDir`)。
- CLI の層: `cleanupRetryCommand`(`internal/cli/org_cleanup.go:211`)を、main worktree がないとき(`:166`)、origin の拒否 3 通り(`:231`)、gh がないとき(`:285`)、`--branch` なしの PR が読めない SKIPPED(`:303`)、`--branch` ありの PR が読めない拒否(`:329-330`)、中断 2 通り(`:381` の command を `:389`・`:399`)が使う。`runCleanupSteps` は `HintStateDir: hintStateDir` を渡す(`:393`)。merge は `mergeCleanupRoot`(`internal/cli/org_merge.go:286`)と `MergeParams.HintStateDir`(`:255`)。
- `CleanupParams{` と `MergeParams{` を作る非テストの所は 3 つ(`merge_run.go:291`、`org_cleanup.go:392`、`org_merge.go:254`)で、どれも hint を渡す。hint は `orgInboxHintStateDir`(`internal/cli/org_inbox.go:353-358`)で、`--state-dir` が渡されたときだけ値を持つ。
- 範囲の外として残るもの(計画の Non-goals と Progress S7): `ralph org merge-order` への案内、`merge.go` の `ralph org inbox`、`merge_wait.go` の `ralph org status --org-id`・`ralph org disband --org-id`、`regate.go:782` の `ralph org cleanup --branch`。

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、49 s) | scope は changed、言語は golang。gofmt `ok`、go vet と staticcheck は無出力(Skipping の行なし)、golangci-lint `0 issues.`。shellcheck、`sh -n`、`jq -e`、Codex の hook の検査、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `4fef6b8c..df194fa1` で clean |
| `./scripts/check-sync.sh` | pass(rc 0) | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 14 本が一致。skill はこのブランチで変わっていない |
| `go build -o <S>/ralph ./cmd/ralph` | pass(rc 0) | HEAD df194fa1。前後で `git status --short` は計画の未コミットの変更 1 件だけ |
| `./scripts/plan-visual.sh digest` | `338876ea047b` | 承認の digest と同じ |
| S1 の突き合わせ(`sort` と `comm`) | 旧にだけある行 0 | AC8 の行を見る |

## Documentation drift

5 つの help を HEAD の binary から取り、README と docs と突き合わせた。

| Surface | Result | Notes |
| --- | --- | --- |
| `ralph org cleanup --help` と README.md:290 | 合う(V1 は help 自身の言い過ぎ) | 再実行の文に「the reason names that command with the --state-dir cleanup was given, if any」が入った。台帳を動かす案内は help・README とも `--state-dir` か `[org].state_dir` で新しい場所を指す形で、`ledgerMoveStep` と合う。README は再実行が `--state-dir` を含むことに触れていないが、`--help` に委ねる要約なので偽ではない(I1) |
| `ralph org merge --help` と README.md:288 | 合う | cleanup の skip の文に同じ句が入った。merge 自身の拒否は help で「says why and what to do next」としか書かず、コマンドの形を言わないので古くならない |
| `ralph doctor --help` と README.md:134 | 合う | `--strict` の行は値の名前なし。「using defaults」の振る舞いは README に書かれておらず、矛盾もない |
| `ralph config get --help` と README.md:135 | 合う | README の行はもとから local-only のキーの `ralph.toml` が exit 3 になることを名指している。Progress S4 の「README の exit 3 の段落が local-only のキーに触れていない件」は直す所がない(I1) |
| `ralph org escalate --help` と README.md:148 | 合う | README の `[org].state_dir` の段落の条件の言い方は定数と同じ語順で、拒否は exit 1、escalate は記録せずに人に届けるという説明も今の振る舞いと合う(README は計画の D4 で範囲の外) |
| `/org` と `/director` skill の「cleanup を打ち直す」文 | 合う | `.claude/skills/org/SKILL.md:1334-1351`、`.claude/skills/director/SKILL.md:920-923`・`:1071-1076`。director は台帳をいつも `--state-dir` で名指す(`:311-320`)ので、理由のコマンドにも付く |
| AGENTS.md:89 の `internal/config/` | ずれ(V2) | `LocalOrgStateDir` を「the org ledger's place for `ResolveOrgStateDir`」と書くが、`ResolveOrgStateDir` は `LocalOrgStateDirAt` を呼ぶ(`internal/org/statedir.go:80`)。`LocalOrgStateDir` を呼ぶ本番のコードはもうない |
| `internal/config/config.go` の doc コメント | ずれ(既知、self-review L3) | StateDir の doc(`:200-205`)は「read … with LocalOrgStateDir」、Load の doc の理由句(`:872-873`)は doctor の古い振る舞いを言う。#269 が同じファイルを変えているので計画どおり触らず、/sync-docs が tech-debt に 1 行書く |
| docs/tech-debt/README.md の 395〜400 行 | 未更新(AC11) | 395(再実行の `--state-dir`)、396(a)(b)、398、399(a)(b)、400 はこのブランチで直った。397 は Go の 2 か所と表のテストまでで、README と `/org` skill の 2 か所は D4 で残る |
| tech-debt の Trigger(このブランチが当てたもの) | 繰り越しの記録が要る(V3) | 下の V3 |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | `ralph org cleanup --help` は、MANUAL: のない SKIPPED: の行のあとは `ralph org cleanup --branch <branch>` を打ち直し、「the reason names that command」と言う。ただ `--branch` なしの実行で PR が読めなかった branch の SKIPPED: の理由は、`ralph org cleanup [--state-dir <dir>] again` を名指し、`--branch` を含まない | help は `internal/cli/org_cleanup.go:109-111`。理由は `:277` と `:303`(`cleanupSteps` のループは `branch == ""` のときだけ回る)。`--branch` と理由のずれは base にもあったが、「that command」の句で言い切る形になった | help の句を「the reason names the command to run, with …」のように理由に合わせるか、この行を例外として書く。どちらのコマンドを打っても片付けは進むので、merge は止めない。直すなら help とその文を固定するテストを一緒に |
| V2 | LOW | AGENTS.md の地図が `ResolveOrgStateDir` の読み手を古い名前で書く | AGENTS.md:89、`internal/org/statedir.go:80` | /sync-docs で `LocalOrgStateDirAt` に直すか、両方を名指す。`homepath.go` の 2 つの関数を足すかは任意 |
| V3 | LOW | このブランチが tech-debt の Trigger を当てたが、払っていない行がある | 135(`doctor_codex_writable_root.go` のコメント、1024 → 1025 行)、386(`org_merge_test.go` 1160 → 1207、`merge_run_test.go` 1107 → 1111、`cleanup_test.go` 1816 → 1817)、390(`cleanup.go` 1089 → 1119、`orgRetryCommand` を足した。`org_cleanup_test.go` 1266 → 1280)、391(`cleanup.go` の拒否の文の `ledgerMoveStep`・`checkWorktree`・`remove` を変えた。`recordGone` の文と `registrationAtRisk` のコメントはそのまま)、392(`statedir.go` を変えた。`resolvedOrClean` の包みは `:253` に残る) | /sync-docs が各行に「fix/org-ledger-gaps も当てたが払っていない」と新しい行数を書く。800 行を新しく越えたファイルはない |
| I1 | INFO | README.md:288・:290 は再実行のコマンドが `--state-dir` を含むことに触れない。Progress S4 が /sync-docs に回した README の exit 3 の件は、README.md:135 がもとから書いているので直す所がない | 上の Documentation drift | /sync-docs で README に 1 句足すかを判断する(任意) |
| I2 | INFO | AC7 の「1 回」は探索の回数で、git の起動は main checkout で 2 回(旧 4 回)。AC2 の表は package ごとに 3 つ | Progress S5、テストのコメント | 記録済みの解釈なので扱いは要らない |

## Observational checks

- binary の probe(origin のない一時の repo、`--state-dir` は空白を含む `<S>/led ger/org`): origin の拒否で、cleanup は `--branch 'feat/a b' --state-dir '<S>/led ger/org'` と `--state-dir '<S>/led ger/org'`(`--branch` なしのまま)、`--branch feat/a`(`--state-dir` なし)を、merge は `--branch feat/a --state-dir '<S>/led ger/org'` を名指した。`ralph.local.toml` の `[org].state_dir` で同じ台帳を決めたときは、cleanup も merge も `--state-dir` のない形を出した。どれも exit 1。台帳には merge が取った `regate.lock` だけが残り、repo の `.harness/state/org` はできていない。
- 5 つの help は evidence にそのまま残した。escalate の help の 74 桁を越える行は 1 行目だけで、このブランチは変えていない。

## Coverage gaps

- Go のテストの結果(/test)。AC1〜AC10 のテストは存在と中身を読んだだけで、打っていない。変異のテストが落ちたという Progress の記述も確かめていない。
- AC3 の doctor の実物の出力。doctor は claude と codex の CLI を探すので打っていない。
- PR #290 の test gaps 3・4(実物の `ralph.local.toml` と実際の HOME の `~/`、`ralph org watch --detach` の子の watch.log の注意の行)。計画どおり /test の evidence で確かめる。
- Linux で `os.Getwd` が実際に失敗するかは未確認(Progress S8 と同じ)。
- AC11(/sync-docs と /test)。
- Insight event: lead の指示(ほかの文書を変えない)に従い、`docs/insights/events/` には足していない。`./scripts/insights-append.sh --slug org-ledger-gaps --flow standard --phase verify --verdict pass --cycle auto --source skill --low 3 --events-dir <scratch>` で作った行は次のとおり。`{"schema":1,"ts":"2026-11-01T04:33:02Z","slug":"org-ledger-gaps","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":3},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`

## Verdict

pass

- Verified: AC1、AC2、AC4、AC5、AC6、AC7、AC8、AC10 は、コードの行、HEAD の binary の help と probe、grep と `comm` で確かめた。AC3 と AC9 はコードとテストの存在で確かめた。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、secret scan)はすべて通り、計画の digest は承認時と同じ。
- Partially verified: AC3(doctor の実物の出力は見ていない)。
- Not verified: Go のテストの結果、PR #290 の test gaps 3・4(どちらも /test)。AC11 は /sync-docs の担当で、このあとに行う。
- 所見: LOW 3 件(V1〜V3)、INFO 2 件。どれも merge を止めない。
- /sync-docs に回すもの: tech-debt 395〜400 行の書き換え(397 は README と `/org` skill の 2 か所を残す)、V2(AGENTS.md:89)、V3(135・386・390・391・392 の繰り越しの記録)、self-review L3 の `config.go` の 1 行、Progress S7 が挙げた範囲の外の案内(`merge-order`・`inbox`・`status`・`disband`)の tech-debt の行。V1 は help のコードの直しなので、直すなら別のスライス。
