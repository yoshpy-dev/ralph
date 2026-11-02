# Walkthrough: org-ledger-gaps(PR #290 の既知の穴を直す)

- Date: 2026-11-02
- Plan: docs/plans/archive/2026-11-01-org-ledger-gaps.md
- Issue: なし(PR #290、#276 の A2 の後続)
- Branch: fix/org-ledger-gaps(分岐元 4fef6b8c。テストのあとに origin/main の f6c844d1 を ea6adfdb で取り込んだ。衝突も、重なるファイルもない)
- 読む範囲: `git diff main...HEAD` は 49 files、+3486/-660。内訳は Go の本番コード 16 files(+423/-169)、Go のテスト 22 files(+2218/-472)、記録(計画・report・evidence・insight event)8 files(+825)、文書 3 files(+20/-19)。動きを決めるのは `internal/cli/org_cleanup.go`(+70/-38)、`internal/org/cleanup.go`(+59/-29)、`internal/org/statedir.go`(+56/-12)、`internal/cli/org_legacy_ledger.go`(+41/-7)、`internal/config/local.go`(+38/-22)、新しい `internal/config/homepath.go`(+36)
- 後工程: パイプラインは 1 回。self-review は LOW 10(5 件を df194fa1 で直した)、verify は pass(LOW 3)、test は pass、cross-review(codex)は所見 0

```
打ち直しのコマンド(S7) → 判定と展開(S2) → doctor と help(S3・S4・S6) → git の探索(S5) → 拒否の文の頭(S9) → テスト(S1・S8) → 文書 → 確かめたことと残したこと
```

## 1. 打ち直しのコマンド(S7、e9b94433)

一番大きい変更です。`ralph org cleanup` と `ralph org merge` は、断ったときや途中で止まったときに「このコマンドを打ち直す」と出します。今まではそのコマンドに `--state-dir` が付かず、`--state-dir` で台帳を選んで打った人が写して打つと、既定の台帳を読みました。

- org の層: `orgRetryCommand(verb, branch, hintStateDir)`(`internal/org/cleanup.go`)が cleanup の `cleanupAgain` と merge の `mergeCheck.again` の両方の文を作ります。`CleanupParams` と `MergeParams` に `HintStateDir` を足し、merge が中で呼ぶ cleanup にも渡します。
- CLI の層: `cleanupRetryCommand`(`internal/cli/org_cleanup.go`)が gh がないとき、origin が読めないとき、中断のときの文を作ります。`--branch` を付けて打ったときは `--branch` を保ちます。今までは中断の文が `--branch` なしの形を出していて、写して打つと片付けの範囲が広がりました(plan advisory の所見 1)。
- `--state-dir` は、渡したときだけ付けます。`ralph.local.toml` の `[org].state_dir` で決めた台帳は、素のコマンドでも同じファイルを読むためです(`orgInboxHintStateDir` と同じ規則)。
- 例外は `ledgerMoveStep` です。台帳を worktree の外へ動かすよう言う文に元の `--state-dir` を付けると、動かした先でなく元の場所を選びます。この文は `--state-dir <新しい場所>` か `[org].state_dir` を名指します(plan advisory の所見 2)。
- テスト: `internal/org/cleanup_retry_state_dir_test.go` と `internal/cli/org_retry_state_dir_test.go`。2 つの台帳を用意し、出力から取り出したコマンドをそのまま打って、最初に選んだ台帳だけが読まれることを確かめるテストがあります。古いコードでは既定の台帳を読んで落ちました。

## 2. 判定と展開(S2、d8e70e0f)

「絶対パスか `~/` で始まるか」の判定と `~/` の展開を、`internal/config/homepath.go` の `IsAbsOrHomePath` と `ExpandHomePath` に 1 つずつ置きました。呼ぶのは `checkLocalAgmsgHome`・`LocalOrgStateDir`(`local.go`)、`agmsgHomeIsRelative`(`internal/cli/org_watch.go`)、`expandTilde`(`internal/org/driver/agmsg.go`)です。

`expandTilde` は seat の agmsg の場所を決める関数で、HOME が読めないときは値をそのまま返します。この振る舞いは変えていません(計画 D2)。`~/x`、`~`、`~user/x`、相対、絶対、空の 6 つの値を通す表のテストが、package ごとにあります。

## 3. doctor と help(S3・S4・S6)

- `runDoctorStateDir`(`internal/cli/doctor.go`)は、`config.Load` がエラーを返したら `config.Default()` で後の検査を続けます。warn の文「using defaults」のとおりになりました。`config.Load` の契約は変えていません(計画 D3)。
- `--strict` の usage から backtick を外しました。pflag は usage の最初の backtick の中を値の名前として出すためです。
- `ralph config get --help` の exit 3 の説明に、local-only のキーを書いた `ralph.toml` も exit 3 になることを書きました。キーの名前は `config.LocalOnlyKeys()` から作り、折り返しは `wrapHelpLines` が行います。
- `RALPH_ORG_STATE_DIR` で断る条件の文を、定数 `retiredOrgStateDirRefusalCondition`(`internal/cli/org_legacy_ledger.go`)にまとめました。doctor の warn と `ralph org escalate --help` の最後の文がこれを使います。

## 4. git の探索(S5、bc2e191e)

`ResolveOrgStateDir`(`internal/org/statedir.go`)は、既定の層で main worktree の root を 1 回だけ探し(`ledgerBaseDir`)、その root の `ralph.local.toml` を `config.LocalOrgStateDirAt` で読みます。main checkout では git の起動が 4 回から 2 回になりました。`gitroot.MainWorktree` は 1 回の呼び出しで git を 2 回起動するので、計画の「1 回」は探索の回数と読みました。

`checkRetiredOrgStateDirEnv` の拒否の文に出す `ralph.local.toml` のパスは、`org.LedgerLocalConfigPath` で台帳のパスから作り、git を打ちません。回数を数えるテスト(`statedir_git_calls_test.go`、`org_retired_state_dir_env_git_test.go`)は、元のコードでは落ちます。

## 5. 拒否の文の頭(S9、9756e03e)

`RALPH_ORG_STATE_DIR` での拒否は、今まで「cannot resolve the org ledger:」で始まり、台帳が解決できないときと見分けられませんでした。今は「<command>: refusing to change the org ledger: …」で始まります。sentinel `errOrgLedgerRefusedByRetiredEnv` を足し、`ralph org escalate` は 2 つの sentinel をどちらも記録しない道へ送ります。頭はコマンドのパスで、`resolveOrgLedger` の既存の形に合わせました。

## 6. テスト(S1・S8)

- S1(93397d1f): `internal/config/local_test.go` を 1011 行から 600 行にしました。移した先は `local_only_keys_test.go` と `local_state_dir_test.go` です。`go test -list` の名前は前後で同じ 104 件です。
- S8(3c3fe4ea): `FeatureRepoRoot` の bare repository の linked worktree の分岐と、`filepath.Abs` の失敗の分岐にテストを足しました。darwin では cwd を消しても `os.Getwd` が失敗しないので、`absPath`(`internal/org/spawn.go`)と `retiredOrgStateDirEnvAbs` を差し替えて起こします(計画 D7)。

## 7. 文書

- docs/tech-debt/README.md: 398・399・400 行は全部、395・396・397 行は一部を払った形に書き換えました。このブランチが Trigger に当たった行に繰り越しの注を書き、末尾に「Small leftovers of org-ledger-gaps」を足しました。
- AGENTS.md の `internal/config/` の行は、`ResolveOrgStateDir` が読む関数を `LocalOrgStateDirAt` に直しました。
- docs/specs/2026-10-07-org-multi-org-director.md: 台帳を動かす案内(137 行)を、この PR の `ledgerMoveStep` と同じ形にしました。88・146 行は決定の記録として残し、`RALPH_ORG_STATE_DIR` に撤去の注だけを足しました。
- README、`/org` skill、`internal/config/config.go` は、別の PR(#269)が変えている最中なので触っていません。

## 8. 確かめたことと残したこと

確かめたこと:

- `./scripts/run-static-verify.sh` は verify と sync-docs で rc 0。`./scripts/run-test.sh` は rc 0(1,481 秒、打ち直しなし)
- 変異: スライスごとに入れた変異と、/test の 13 個は、すべてテストで落ちた
- 実物の binary: 実物の `ralph.local.toml` の `~/` が実際の HOME で解決されること、`--state-dir` 付きの `ralph org watch --detach` の子が注意を 1 行だけ出すこと(docs/evidence/test-2026-11-01-org-ledger-gaps-real-binary.md)

残したこと(どれも tech-debt に書いた):

- merge と cleanup が出す、ほかの verb を名指す案内(`ralph org merge-order` が約 12 か所、`ralph org inbox`、`ralph org status --org-id`、`ralph org disband --org-id`)は `--state-dir` を持たない(395 行)
- `ralph org cleanup --help` の「the reason names that command」は、`--branch` なしで打ったときの SKIPPED の行の理由と食い違う(verify の V1)
- org の層と CLI の層で quote の規則が違う(self-review の L5)
- README と `/org` skill の断る条件の文(397 行)、`config.Load` の doc の理由句(396 行)
- `cleanup.go` の 2 つの分岐(ディレクトリでない物、`git worktree remove` の失敗)はテストが打たない。Linux での本物の `os.Getwd` の失敗は確かめていない
