# Walkthrough: config-path-and-ledger-docs(前の 2 本が残した穴の、最後の分を直す)

- Date: 2026-11-02
- Plan: docs/plans/archive/2026-11-02-config-path-and-ledger-docs.md
- Issue: なし(PR #297 と PR #296 の後続。`Refs #276`)
- Branch: fix/config-path-and-ledger-docs(分岐元 50f2ffc0。途中でセッション C の PR #300 が merge されたので、ea8b4d16 で origin/main 350f22ea を取り込んだ。衝突はない)
- 読む範囲: `git diff origin/main...HEAD` は 21 files、+1187/−61。動きを変えるのは `internal/cli/org.go` の 1 行(`resolveConfig`)と `internal/cli/status.go` の 1 行だけで、残りはテスト(`org_start_plan_config_test.go` +260 ほか)、doc とコメント、README と skill の 1 文、tech-debt、記録
- 後工程: パイプラインは cycle 1 の 1 回。self-review は LOW 5(4 件を F1 で直した)、verify と test は pass、cross-review(codex)は所見 0

```
エラーにパス(S1) → テストの穴(S2) → doc(S3) → 条件の文(S4) → self-review の直し(F1) → 実物の確かめ → 文書 → 残したこと
```

## 1. エラーにパス(S1 40cedc9e、a6be3f5e)

org の動詞が設定ファイルを読めないとき、今までは `org: load config: toml: …` と誤りの中身だけを言い、どのファイルかを言いませんでした。main worktree の root から打った `ralph org start --plan` と、`--config` で名指したファイルのときです(tech-debt 421 行の (a)(b))。

今は `resolveConfig`(`internal/cli/org.go:661`)が `loadConfigFile` のエラーを `<path>: <err>` で包みます。`<path>` は読んだパスそのもので、`--config` の値か、相対のままの `ralph.toml` です。`org: load config: ` で始まる 4 つの文(`newOrgRuntimeAt`、`resolveOrgStartConfig`、`org_director.go`、`org_director_headless.go`)はどれもこの関数を通るので、1 か所の変更で `org: load config: ralph.toml: toml: …` の形になります。

`config.Load`、`ralph config get`、`ralph doctor` は変えていません(計画 D1)。`Load` の doc の規則では、呼び出し元が渡したパスは繰り返さず、`config get` はその規則どおり自分でパスを付けています。`Load` で付けると、この 2 つと、main worktree の `ralph.toml` を読む文の 3 か所でパスが二重になります。計画の段階で consult は一度 Load を推しましたが、この根拠で撤回しました。

`%w` で包むので、`errors.Is(err, fs.ErrPermission)` などはそのまま通ります。

一方で、S1 のあと `ralph status` の warning が `load ralph.toml: ralph.toml: toml: …` とパスを 2 回出すようになりました。warning が自分で `ralph.toml` を名指していたためで、計画の Assumptions が見落としていた呼び出し元です。a6be3f5e で warning は名前をエラーに任せる形にし、出力は前と同じに戻りました。

テスト(`internal/cli/org_start_plan_config_test.go`):

- `TestOrgStartPlan_UnreadableConfigStartsNothing`(`:149`): main worktree の root から打つケースを足した。`org: load config: ralph.toml: ` で始まり、何も作らない。`--config` のケースは `org: load config: <path>: ` を見る。古いコードでは root のケースが落ちた。
- `TestOrgSpawnStartAndDirectorStart_NameTheConfigFileTheyCouldNotLoad`(`:214`): spawn、`start <task>`、`director start` の 3 つを、`./ralph.toml` と `--config` の 2 通りで打つ。
- `status_test.go` の既存の条件に、`ralph.toml: ralph.toml` を含まないことを足した。

## 2. テストの穴(S2、f185f22d)

テストだけの変更です(tech-debt 421 行の (c)(d))。

- 同じファイルで 2 回書かれていた「main worktree に `ralph.toml` を書いて commit する」2 行を、helper `commitRalphTomls` にまとめた。
- 3 つのエラーの道のうち、`newOrgSpawnRuntimeAt` と `newOrgRuntimeAt` の道は S1 のテストがすでに通るので、重ねていない。`runOrgStartPlan` が `resolveOrgLedger` のエラーを返す道は `TestOrgStartPlan_RefusedLedgerStartsNothing`(`:526`)が打つ。git の外で `RALPH_ORG_STATE_DIR` を別の場所に向け、設定を 1 つも読まず、台帳も作らないことを見る。
- 上限のテスト(`:441`)に `max_total_seats` の対照を 2 つ足した。`--config` のときと、台帳が main worktree の外のときは、main の `max_total_seats = 1` を使わない。
- `TestOrgStartPlan_OutsideLedger_RecordsTheApprovalModeOfTheFileItRead`(`:572`): 台帳が外のとき、main の `[org.approval].mode` ではなく読んだファイルの mode が記録されることを、binding を直接読んで確かめる(両向きの 4 ケース)。

どのテストも、守るコードを壊した写し(`go test -overlay`)で落ちることを確かめています。

## 3. doc(S3、3311cf16)

`internal/config/config.go` のコメントだけを直しました(tech-debt 396 行の (c))。

- `Load` の doc から、「`ralph doctor` がそのまま続けるから `Default()` を返す」の理由句を外した。doctor は PR #296 から、Load のエラーのとき `config.Default()` で続けている。
- `StateDir` の doc は、台帳の場所を `org.ResolveOrgStateDir` が `LocalOrgStateDirAt` で読むと書く。

## 4. 条件の文(S4 d257824e、b16f4bd0)

古い変数 `RALPH_ORG_STATE_DIR` で台帳を書き換える動詞を断る条件は、Go の定数 `retiredOrgStateDirRefusalCondition`(`internal/cli/org_legacy_ledger.go`)にあり、doctor の warn と `ralph org escalate --help` がそれを使います(PR #296)。README と `/org` skill は別の言葉で書いていて、テストが結んでいませんでした(tech-debt 397 行)。

- README の `[org].state_dir` の段落は、「the variable names」を「`RALPH_ORG_STATE_DIR` names」にした。backtick を外すと定数の文と同じになる。
- `/org` skill の 4 つの写しは、日本語の文を残し、「`ralph doctor` の warn と `ralph org escalate --help` は、この条件を英語でこう書く」に続けて、定数の文を code span で添えた。最初は「条件は Go の定数と同じ」と書いたが、scaffold した project には Go の定数がないので b16f4bd0 で言い直した(implementer の指摘)。
- `TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs`(`internal/cli/org_retired_state_dir_condition_docs_test.go:21`)が、README(backtick を外す)と `.claude`・`templates/base/.claude` の skill(code span)を読む。`.agents` の写しは `check-skill-sync.sh`、templates の写しどうしは `check-sync.sh` が一致を見る。

## 5. self-review の直し(F1、d3bb5b55)

LOW 5 件のうち 4 件を、出力を変えずに直しました。

- 定数の doc に、README と skill の 4 写しも同じ言葉を持つと書いた。
- `resolveConfig` の doc を、渡されたパスを前に付ける、と正確にした。パスが 2 回並ぶ形のうち 2 つ(開けないファイルの `ralph.toml: open ralph.toml: …` と、壊れた `ralph.local.toml` の `ralph.toml: <abs>/ralph.local.toml: …`)を `TestResolveConfig_PrefixesTheFileItWasAskedToRead`(`:271`)で固定した(計画の Risks のとおり)。
- 新しいテストの doc と表を読みやすくした。

残る L4(`ralph.toml` に `[org].state_dir` を書いた人が見る移行用の error が `ralph.toml: ralph.toml: …` になる)は、D1 で受け入れた代価として tech-debt 421 行の (f) に書きました。

## 6. 実物の確かめ(/test、docs/evidence/test-2026-11-02-config-path-and-ledger-docs-live.md)

- codex の model: 一時の repository の `ralph.toml` に既定と違う `gpt-6-luna`・`low` を書き、skill と同じ形で codex を起こした。session の記録の `turn_context` は `model=gpt-6-luna`、`effort=low`。PR #297 の確かめは既定の slug だったので、設定から来た model だと区別できなかった。
- leader の通し(隔離した herdr の server、`[org.review].plan_advisory = true`、`[reviewer]` は 4 つとも既定以外):
  - claude の leader → codex: leader は codex を `-m gpt-6-luna -c model_reasoning_effort=low exec --sandbox read-only …` で起こし、タスクの `reviewer の codex: gpt-6-luna low` の行と一致した。承認の QUESTION で待つところまで 2 分 20 秒。
  - codex の leader → claude: 確かめていない。codex は leader のコマンドを zsh で打ち、ユーザーの `~/.config/zsh/.zshenv` が PATH を組み直すので、leader の `ralph` が `/opt/homebrew/bin/ralph`(6.0.0、`config get` がない)になった。役割指示の規則で advisory は off 扱いになり、claude を起こさなかった。

## 7. 文書(/sync-docs、1982be83、18d336a0)

- tech-debt: 396 行は全部を払った。397 行は「Left open」を払い、場所の数と Trigger を今の場所に合わせた。421 行は (a)(b)(d) を払い、(c) はこのファイルの分を払った。新しい (f) に、パスが 2 回並ぶ形を D1 の代価として書いた。(e) の case 2 は Trigger つきで残した。148・411・88 の行に、当たった Trigger の注を書いた。
- AGENTS.md と README のほかの段落は、食い違う文がなく変えていない。

## 8. 残したこと

- codex の leader が claude の reviewer を起こす通し(421 行の (e))。PATH の古い ralph の問題で、tech-debt の PATH の行と同じ根。
- パスが 2 回並ぶ形(421 行の (f)): 開けないファイル、`ralph.toml` という名前のディレクトリ、`ralph.toml` の local-only のキー、main worktree の文。2 つはテストで固定し、2 つは固定していない。
- 同じ add と commit の 2 行が、ほかのテストファイル 3 つ(4 か所)に残る(421 行の (c))。
- /test の leader の通しの副作用: ユーザーの `~/.codex/config.toml` に、消した一時ディレクトリの `trust_level = "trusted"` の 2 行が足された。case 2 の leader が起こした codex の app-server daemon 2 つが残っている。どちらもユーザーの環境なので、触らずに報告した。
