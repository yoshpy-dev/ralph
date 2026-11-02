# Walkthrough: org-ledger-leftovers(PR #296 が残した穴を直す)

- Date: 2026-11-02
- Plan: docs/plans/archive/2026-11-02-org-ledger-leftovers.md
- Issue: なし(PR #296、#276 の A2 の後続。`Refs #276`)
- Branch: fix/org-ledger-leftovers(分岐元 31812a3b。途中で #269 が PR #298 として merge されたので、e208a2c6 で origin/main c8c8c0f1 を取り込んだ。衝突はなく、この branch のコードのファイルとは重ならない)
- 読む範囲: `git diff origin/main...HEAD` は 52 files、+3018/−1171。動きを決めるのは S2・S3・S4 の本番コードで、残りは S1 の移動(約 900 行の移動)、テスト、tech-debt、記録(計画・報告・evidence・insight event)
- 後工程: パイプラインは cycle 1 の 1 回。self-review は LOW 11(10 件を F1 で直し、1 件を tech-debt に書いた)、verify と test は pass、cross-review(codex)は所見 0

```
ファイルの分割(S1) → quote を 1 つに(S2) → 案内に --state-dir(S3) → help と文(S4) → テスト(S5) → self-review の直し(F1) → 実物の確かめ → 文書 → 残したこと
```

## 1. ファイルの分割(S1、3b54f1b1)

移すだけのコミットです。tech-debt 390 行は「次に `internal/org/cleanup.go` を変えるとき、文の組み立てを `cleanup_text.go` に移す」を 3 回繰り越していました。この PR は `cleanup.go` を変えるので、最初に払いました。

- `internal/org/cleanup.go`: 1119 → 790 行。18 の宣言を新しい `internal/org/cleanup_text.go` へ移した。拒否の文と、それが名指すコマンドを組み立てる関数(`cleanupAgain`、`orgRetryCommand`、`recordGone`、`registrationAtRisk`、`ledgerMoveStep` など)に加え、git を打たずに読んだデータから文を作る `checkRecord`・`checkRegistration` と、葉の関数 `pathWithin` も移した。候補だけでは 903 行で 800 に届かなかったため。
- `internal/cli/org_cleanup_test.go`: 1280 → 701 行。中断・`cleanupSteps`・`cleanupPrinter`・help のテスト 10 の宣言を `internal/cli/org_cleanup_steps_test.go` へ移した。

宣言を doc コメントごと byte で比べ、欠落・重複・書き換えがないことを確かめました(verify の AC1 も同じ比べ方で確かめた)。`go test -list` の名前は org 1424 件、cli 979 件で前後同じです。

HEAD では `cleanup.go` が 792 行で、余裕は 8 行です。新しい文は `cleanup_text.go` に置く約束を tech-debt 390 行に書きました。

## 2. quote を 1 つに(S2 47a4fda1、f972c80b、F1 bb0afab1)

打つコマンドの語を「要るときだけ quote する」関数が 3 つあり、裸のまま残す文字が違っていました。

| 関数 | 層 | 裸のまま残す文字 |
|---|---|---|
| `cleanupShellWord` | org | `[A-Za-z0-9_./:@+,-]` |
| `shellQuoteIfNeeded` | cli | `[A-Za-z0-9_./-]` |
| `directorShellWord` | org(director stop の herdr の id) | `[A-Za-z0-9_.:-]` |

このため `feat/a+b` のような値が、1 つの出力の中で裸と `'feat/a+b'` の 2 通りに出ていました(tech-debt 388 行)。今は `org.ShellWord`(`internal/org/shellword.go:31`)の 1 つです。

- org の層は `cleanupShellWord` を `ShellWord` に改名し、呼び出しをすべて替えた。org の層は cli を import できないので、共有の関数は org の側に置いた(計画 D3)。
- cli の層は `shellQuoteIfNeeded` と `shellSafeUnquoted` を消し、本番 9 か所(`orgReadCommandHint`、旧台帳の文、`cleanupRetryCommand`)とテスト 25 か所を `org.ShellWord` に替えた。テストの `orgLayerShellWord` と `orgPlainWord` は、2 つの関数の差のためだけにあったので消した。
- `directorShellWord` は implementer が範囲の外で見つけた 3 つ目で、f972c80b で `ShellWord` に寄せた。
- F1 で、`ShellWord`・`shellPlainWord`・`VerbCommand` と、いつも quote する `shellQuote`(`feature.go` から)を `shellword.go` にまとめた。`ShellWord` の doc は、いつも quote する `shellQuote` を escalate・approve・watchgate などがいまも直接使うことを書く。

出力が変わるのは、`: @ + ,` を含む値の cli の層の行と director stop の herdr の id だけで、どれも quote なしになります。shell が特別に読む文字はどの関数も quote するので、打ったときの動きは変わりません。固定文の期待で変わったものは 0 本でした。

テスト:

- `TestShellWord_QuotesOnlyWhatAShellReads`(`internal/org/shell_word_test.go:11`): 裸の 4 値(`feat/a+b`、`a:b`、`a@b`、`a,b`)と quote する値(空白、`'`、`$x`、backtick、空文字列)の表。
- `TestShellWord_PrintedCommandsSpellAValueLikeTheOrgLayer`(`internal/cli/org_shell_word_test.go:18`): 同じ 6 値で、cli の `cleanupRetryCommand`・`orgReadCommandHint` と org の `orgRetryCommand` が同じ文字列を書くことを、期待の文字列で確かめる。

## 3. 案内に --state-dir(S3、a326e050)

PR #296 は、`ralph org merge` と `ralph org cleanup` が出す「同じ動詞の打ち直し」のコマンドに `--state-dir` を付けました。この PR は、ほかの動詞を案内する 18 文に付けます(tech-debt 395 行の残り)。

- org の層 15 文: `cleanup.go:265` の checkCleanupParams(`merge-order --all`)、`merge.go` の 7 文、`mergeenforce.go` の 1 文、`merge_run.go` の 5 文、`merge_wait.go:145-146` の `status --org-id` と `disband --org-id`。
- cli の層 3 文: `org_merge.go` の `--branch` がない拒否と結果のない error、`org_cleanup.go` の `cleanupNotInLedger`(`:354`)。

組み立ては 1 か所です。`org.VerbCommand`(`internal/org/shellword.go:52`)は `ralph org <verb and args>` に、`--state-dir` が渡されたとき(`HintStateDir` が空でないとき)だけ ` --state-dir <ShellWord(dir)>` を付けます。`orgRetryCommand` もこの上に組み直しました。cli の層は `orgVerbHint`(`internal/cli/org_cleanup.go:231`)が dir を `PrintableInboxText` で escape してから `VerbCommand` に渡し、F1 で `cleanupRetryCommand` もその上に乗せました。

規則は PR #296 と同じです(計画 D5)。`ralph.local.toml` の `[org].state_dir` で決めた台帳は、素のコマンドでも同じファイルを読むので付けません。

値を届けるために、関数とフィールドを足しました。

- 引数: `recordMerge`、`printMergeResult`、`cleanupNotInLedger`、`enforce`、`readEnforcementRepository`。
- フィールド: `MergeEnforcementParams.HintStateDir`、`mergeWait.hintStateDir`。

`enforce` が引数なのは、`mergeGuaranteeRun` を doctor と共有しているためです。`--branch` がない拒否は台帳を解決する前に断るので、`--state-dir` が渡されたときだけ `org.ResolveOrgStateDir(*stateDir, true)` で絶対パスにします。この呼び出しは git を打たず、error も返しません(F1 でコメントに書いた)。

変えなかった言及: help と flag の説明、`org_cleanup.go:174-176` の読み方の説明、merge-order 自身のエラー文の書き換え(`mergeLsRemoteRefusal`)、merge-order の Detail が名指す gate と regate のコマンド(verify V4、tech-debt 395 行に残した)。

テスト:

- `internal/org/merge_hint_state_dir_test.go`: 文の組ごとに 6 本(`TestVerbCommand_…`、`TestCleanupFeatureWorktree_HintStateDirReachesMergeOrderWithNoBranch`、`TestCheckMerge_…`、`TestCheckMergeEnforcement_…`、`TestMerge_…MergeOrderOnceGHPRMergeRan`、`TestMerge_…StatusAndDisbandOnceTheWaitIsUsedUp`)。空白入りの dir あり・なしの両方で打ち、`--state-dir` の出現数まで数える。
- `internal/cli/org_hint_state_dir_test.go`: cli の 3 文と、`TestOrgCleanup_HintStateDirMergeOrderReplaysOnTheLedgerNamedFirst`(`:120`)。2 つの台帳を用意し、出力から取り出した `ralph org merge-order --all --state-dir …` をそのまま打って、名指した台帳の一覧を読むことを確かめる。hint を無視する写しでは落ちた。

## 4. help と文(S4、94c07d76)

- `ralph org cleanup --help` は、SKIPPED の理由が名指すコマンドが 2 通りあることを書きます。branch の側の理由(未追跡のファイル、regate.lock、org が動いているなど)は、`--branch` なしで打っても `ralph org cleanup --branch <branch>` を名指します。PR や origin を読めなかった理由は、打ったとおりの形を名指します。どちらも `--state-dir` は渡したときだけ付きます(tech-debt 420 行の (a))。前の文は「打ち直すのは `--branch <branch>`」とだけ言っていました。計画の段階で「どの理由も打ったとおり」とまとめかけ、codex の計画助言に指摘されて 2 通りに分けました。どちらの場合も既存のテストが `--branch` なしで打って確かめています(`TestOrgCleanup_RefusalsAreSkippedLines`、`TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing`、`TestOrgCleanup_WithoutGHRemovesNothing`)。
- 記録がないときの拒否(`recordGone`、`cleanup_text.go:130`)は「remove what is left by hand」と言います。前の「remove it by hand」は、直前に残すと言った物を指すようにも読めました(tech-debt 391 行の (a))。
- `registrationAtRisk`(`cleanup_text.go:287`)のコメントは、新しい branch に退避するのを detached HEAD だけにしました。どの branch にもない HEAD は git がゼロで出すので、その道には来ません(391 行の (b))。

## 5. テスト(S5、cf69cab5)

本番のコードは変えていません。

- `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead` は、期待の `git -C` の語を `org.ShellWord` で作り、空白を含む一時ディレクトリの subtest を足しました(tech-debt 420 行の (c))。Go 1.27 の `t.TempDir` は GOTMPDIR を先に読むので、GOTMPDIR と TMPDIR の両方を空白入りのディレクトリに向けます。
- `TestCleanupFeatureWorktree_NotADirectoryAtTheWorktreesPathRemovesNothing`(`internal/org/cleanup_run_test.go:27`): worktree の場所に通常のファイルがあるときの拒否(`st.notDir`)。
- `TestCleanupFeatureWorktree_FailedRemovalOfAMissingWorktreesRegistrationRemovesNothing`(`:83`): `git worktree lock` した登録の `git worktree remove` が失敗するときの Failed と MANUAL の行。unlock のあとは次の cleanup が進む。
- `TestOrgRetiredStateDirEnv_RefusalOfALedgerElsewhereNamesTheWorkingDirectorysLocalFile`: `org.LedgerLocalConfigPath` が `ok == false` を返す守りの分岐。
- Linux だけで走る 2 本(`TestAbsPath_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved` と `TestOrgRetiredStateDirEnvAbs_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`): 作業ディレクトリを消したあと、本物の `absPath` と `retiredOrgStateDirEnvAbs` が失敗する。PR #296 の D7 の前提(Linux では `os.Getwd` が失敗する)を確かめる。darwin では skip し、CI の ubuntu で走る。implementer は linux/arm64 のテストバイナリを docker の ubuntu:24.04 でも打った。

どれも分岐を 1 行壊した写し(`go test -overlay`)で落ちることを確かめています。

## 6. self-review の直し(F1、bb0afab1)

LOW 11 件のうち 10 件を、出力の文を変えずに直しました。`cleanupRetryCommand` を `orgVerbHint` の上に乗せる、`shellword.go` にまとめる、`VerbCommand` の doc に「escape しない、呼び出し側が文全体を `PrintableInboxText` に通す」を書く、などです。tech-debt 386 行の 4 つのファイル(計画 D6 でテストを足さないとした)では、この branch が足した行と 1 つの改名だけを変えました。残る 1 件(L6、`cleanup.go` の余裕 8 行)は tech-debt 390 行に書きました。

F1 は self-review の後、#269 の取り込みと verify・test・cross-review の前に入りました。

## 7. 実物の確かめ(/test、docs/evidence/test-2026-11-02-org-ledger-leftovers-real-binary.md)

この branch から build した binary を、一時の repository で打ちました。gh は引数を記録するだけの偽物、origin は手元の bare に答える偽の ssh を使ったので、本物の gh にもネットワークにも出ていません。

- `ralph org cleanup --branch feat/none --state-dir '<空白入りの台帳>'` の拒否は、`ralph org merge-order --all --state-dir '<その台帳>'` を名指した。写して打つと、名指した台帳の一覧を返した。既定の台帳の一覧とは違う。
- `--state-dir` なしでは、文に `--state-dir` が付かない。
- `ralph org merge` を `--branch` なしで `--state-dir <dir>` 付きで打つと、`ralph org merge-order --state-dir <dir>` を名指した。

org の層の 15 文は GitHub の PR を読める repository が要るので、binary では出さず、テストだけが見ています。

## 8. 文書(/sync-docs、b3d3d06b、c21d3aa6)

- tech-debt: 391 行は全部を払った。388・390・395・420 行は、払った部分を取り消し線にし、残りに Trigger を付けた。386 行に繰り越しの注、170・197・290・389・392・401・403 行に当たった Trigger の注、313 行に `shellQuote` の新しい場所を書いた。表の末尾には足していない。
- AGENTS.md: `internal/org/` の行に `cleanup_text.go` と `shellword.go` を足した。
- README と `/org`・`/director` の skill は、どの印字したコマンドが `--state-dir` を持つか、cleanup の理由がどのコマンドを名指すかを書いていないので、変えていない。

## 9. 残したこと

- いつも quote する `shellQuote` は、escalate・approve・watchgate などがいまも直接使う。同じ台帳が、`ralph org inbox` の案内では `'/x'`、merge の案内では `/x` と、動詞によって 2 通りに出る(tech-debt 388 行)。
- `cleanup.go` は 792 行で、余裕は 8 行(390 行)。
- merge の「ready でない」拒否が付ける merge-order の Detail は、`ralph org regate --branch` と `ralph org gate --org-id` を `--state-dir` なしで名指す(verify V4、395 行)。
- 一時ディレクトリのパスに空白があると、前からある 10 本のテストが落ちる(`internal/org/cleanup_test.go` の 9 本、`internal/cli/org_cleanup_test.go` の 1 本)。期待の文を quote なしのパスから作っているためで、base c8c8c0f1 でも同じ。本番の出力は正しく quote されている(420 行)。
- 実物の binary は main checkout の root からだけ打った。linked worktree とサブディレクトリはテストだけが見ている(420 行)。
- tech-debt 386 行の 4 つのテストファイルは分けていない(計画 D6)。
- #269 待ちだった README・`/org` skill・`internal/config/config.go` の文(396(c)、397 の残り)と、PR #297 の穴(421 行)は、次の PR で直す。
