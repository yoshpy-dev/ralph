# Walkthrough: org-state-dir-local(org の台帳を `--state-dir` と `ralph.local.toml` の `[org].state_dir` で決め、`RALPH_ORG_STATE_DIR` を読まなくする)

- Date: 2026-10-30
- Plan: docs/plans/archive/2026-10-30-org-state-dir-local.md
- Issue: #276(`Refs`。#276 を 3 本に分けた 2 本目 A2。1 本目 A1 は #287 で merge 済み、3 本目 B が標準フローの値とスクリプトを受け持ち `Closes #276`)
- PR: #290(https://github.com/yoshpy-dev/ralph/pull/290)
- Branch: feat/org-state-dir-local(この walkthrough の前の HEAD は 27bc1166。merge を除く 38 commits と、origin/main を取り込んだ merge commit 3700470e と 668e9349)
- Base: c03647f9(origin/main、「Merge pull request #288」= org-merge-cleanup)。分岐元も同じ。2 回目の取り込みで 1e266f96(#289 regate-ci-checks)まで入れた
- 読む範囲: `git diff origin/main...HEAD` は 78 files、+5082/-596。内訳は Go の本番コード 21 files(+624/-253)、Go のテスト 31 files(+2280/-200)、skill の写し 8 files(+156/-92)、記録(計画・report・evidence・insight event)9 files(+1956)、leader の雛形 1 file(+9/-9)、残りの文書と設定 8 files(+57/-42)。動きを決めるのは `internal/config/local.go`(+120)、`internal/cli/org_legacy_ledger.go`(+111)、`internal/cli/doctor.go`(+83)、`internal/cli/org.go`(+76/-65、ほとんどが呼び出しの付け替え)、`internal/cli/doctor_merge_guarantee.go`(+57)、`internal/org/statedir.go`(+49/-32)。テストの差分の大半は、新しいテスト(`org_ledger_config_test.go` 472 行、`org_retired_state_dir_env_test.go` 466 行、`internal/config/local_test.go` +411 行、`statedir_test.go` +264 行)と、`RALPH_ORG_STATE_DIR` を `t.Setenv` していたテストの書き換え
- 後工程: パイプラインは 2 回(`cycle-count.json` は 2、既定の上限)。cycle 1 の cross-review が ACTION_REQUIRED を 1 件出し、S9 で直して /self-review から回し直した。self-review は cycle 1 が LOW 9(S7 と S8 の見直しを含む)、cycle 2 が LOW 1(C2-L1)でどちらも Merge 可。verify は cycle 1・cycle 2 とも pass(S8 の追補を含む)。test は cycle 1・cycle 2 とも pass。sync-docs は 2 回。cross-review(codex)は cycle 1 が所見 1、cycle 2 が 0。記録だけの修正の例外を使った記録は、triage report にも計画の Progress にもない

```
何を変えるか → 個人用のファイルのキー(S1) → 台帳の決め方と cli の helper(S2) → escalate が人に届く道(S2b) → insights と doctor の --state-dir(S3) → env の撤去・拒否・warn(S4) → 文書(S5) → #288 の cleanup の合わせ(S6) → レビューで足した直し(S7、S8、S9) → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで:
  - 台帳(`.harness/state/org`)の場所は `--state-dir` > 環境変数 `RALPH_ORG_STATE_DIR` > git の主 worktree > git の toplevel > cwd の順で決まった。
  - `ralph insights` と `ralph doctor` には `--state-dir` がなく、台帳を変えられるのは env だけだった。渡し忘れると黙って別の台帳を見る。
- この PR から:
  - 台帳は `--state-dir` > `ralph.local.toml` の `[org].state_dir`(source 名 `config`)> git の主 worktree > git の toplevel > cwd の順で決まる。ralph は `RALPH_ORG_STATE_DIR` を値として読まない。
  - `[org].state_dir` は個人用のファイルだけに書けるキー。`ralph.toml` に書くと `config.Load` がエラーにする(キーの大文字小文字は問わない)。値は絶対パスか `~/` だけ。
  - `ralph insights` と `ralph doctor` に `--state-dir` が付く。`insights` は `--receipts` があればそちらが勝つ。`doctor` は merge guarantee のキャッシュの置き場所に使う。
  - env が空でない値で残っているとき、台帳を決めるコマンドは stderr に注意を 1 行出す。台帳が `--state-dir` か `[org].state_dir` で決まらず、env が既定の台帳と違う場所を指すときは、台帳を変えるコマンドが何も書かずに終了コード 1 で断られる。`ralph doctor` は env が残る間 warn を出す。
- issue の文との違い(issue は #276 の本文。計画の Design decisions D1〜D7 から書く):
  - D1 個人用のファイルは、`--config` ではなく cwd の repository の主 worktree の root のものを読む。台帳が決まってから主 worktree の `ralph.toml` を読む処理があり、`--config` に依存させると順が循環しうるため。
  - D2 個人用のファイルの相対の `state_dir` は、台帳を決める側(`LocalOrgStateDir`)だけが断る。`config.Load` は値を検査しない。Codex の計画助言の MEDIUM で、`--state-dir` を渡した `ralph org stop` などが `Load` に妨げられない逃げ道を残すため。
  - D3 `ralph.toml` の `[org].state_dir` は無視や warn でなくエラー。commit された値は全員の台帳を 1 か所に向ける。`templates/base/ralph.toml` は注釈でだけ触れる。
  - D4 個人用のファイルが読めない、または正しくないときは、次の段に黙って落ちずにエラーにする。doctor だけは fail の行を 1 つ出して続ける。
  - D5 env が残っているとき、初めは「注意だけ」だった。Codex の計画助言の HIGH を受けて、台帳を変えるコマンドを断る形に変えた。start と spawn は見張りを自動で起こすので watch gate では止まらず、更新した人の新しい作業が既定の台帳で始まり、上限と予約が 2 つの台帳で別々に数えられてしまう。
  - D6 台帳を決める呼び出しを `resolveOrgLedger` に寄せる。D7 スライスは、返り値の変更(S2)と env の撤去(S4)に分ける。
- 計画の本文の読み替え(承認の digest `1ea27c42968f` を変えないため本文は直さず、Progress checklist に書いた):
  - 「16 か所の呼び出し」のうち helper を通るのは 15 か所で、doctor は `org.ResolveOrgStateDir` を直接呼ぶ(fail の行で続けるため、cobra の `cmd` と legacy の guard を前提にする helper を通らない)。16 は doc コメント 1 行を数えていた。
  - S2b(escalate)、S6(#288 の cleanup)、S7〜S9 は本文の Implementation outline にない。判断と理由は Progress に書いて足した。
- release のノート: PR 本文の「release のノートに書くこと」と、sync-docs の report の「Release notes」の節。要点は、台帳を env で変えていた人は値を `[org].state_dir`(`ralph.local.toml`)に移すか `--state-dir` を渡し、env を消すこと。更新のあとに `ralph doctor` を打つ。
- 戻し方: この PR を revert する。台帳の中身と形式は変えていない。ただし revert した ralph は `[org].state_dir` を読まない(推測、未確認)ので、移行した人は env を設定し直すか `--state-dir` を渡さないと既定の台帳に戻る。

## 2. 個人用のファイルのキー(S1、e2e7b7f1)

- `OrgConfig.StateDir`(`internal/config/config.go:105`、`toml:"state_dir"`)。`Load` は値を検査しない(D2)。`ralph config get org.state_dir` は reflect で引けるので配線を足さず、個人用のファイルの値を書いたとおり(`~/` を展開せず)に返す。
- `LocalOnlyKeys()`(`internal/config/local.go:118`)は `["org.state_dir"]` を返す。`checkLocalOnlyKeys`(`:129`)は `ralph.toml` の文書だけを map に読み、キーがあれば値が空でもエラーにする。エラーはファイルのパスを名指し、書く先の `ralph.local.toml` を言う。`config.Load`(`config.go:760`)は、このエラーのとき `Default()` を返す(S7、下の 9 節)。`RALPH_LOCAL_CONFIG=off` でもエラーにする(commit された `ralph.toml` の誤りは、個人用のファイルを読むかどうかと関係ない)。
- `LocalOrgStateDir(dir)`(`:190`): `RALPH_LOCAL_CONFIG=off` なら git も打たずに空を返す。`LocalPath`(A1)で `dir` の repository の個人用のファイルを引き、`[org].state_dir` だけを struct に読む。空なら空、`~/` は `os.UserHomeDir()` で展開(失敗はエラー)、絶対は `filepath.Clean`、そのほか(相対、`~` だけ、`~user/…`)はファイルのパスと値を名指すエラーにする。`ralph.toml` は読まない。
- テスト(`internal/config/local_test.go`、`defaults_sync_test.go`、`internal/cli/config_test.go`): `TestLocalOrgStateDir_Values`、`_RefusesValuesThatAreNotAbsolute`(`~` と `~user/ledger` と先頭の空白を含む)、`_FileErrorsNameItsPath`、`_LocalConfigEnv`、`_ReadsTheMainWorktreesFile`、`TestLoad_LocalStateDir`、`TestLoad_LocalOnlyKeyInRalphToml`、`TestLocalOnlyKeys`、`TestTemplateDriftProbesHoldNoLocalOnlyKey`(drift の probe が `LocalOnlyKeys` のキーを持たないこと)、`TestConfigGet_OrgStateDirPrintsThePersonalFilesValueAsWritten`、`TestConfigGet_RalphTomlThatSetsALocalOnlyKeyExits3`。
- 検証: rc 0。変異 7 つをテストが落とした。

## 3. 台帳の決め方と cli の helper(S2、daf8e512)

- `ResolveOrgStateDir`(`internal/org/statedir.go:72`)は `(dir, source, err)` を返す。順は flag(source `flag`)→ `config.LocalOrgStateDir("")`(source `config`)→ `gitroot.MainWorktree`(`git-main-worktree`)→ `gitroot.Toplevel`(`git-toplevel`)→ cwd。個人用のファイルの段がエラーを返したら下の段に進まない(D4)。値は `mustAbs` を通す(`$HOME` が相対でも台帳が絶対になる。S7 の L6)。
- source `config` は、これまでの `env` と同じに扱う。`LedgerMainWorktreeRoot`(`:123`)は `case "flag", "config"`、`FeatureRepoRoot`(`:146`)は `git-main-worktree` 以外を同じ道で扱い、エラーの中で次に打つコマンドの `--state-dir` の hint は flag のときだけ付く(`orgInboxHintStateDir`、`internal/cli/org_inbox.go:351`)。`LegacyWorktreeStateDir`(`:176`)は flag か個人用のファイルの選択に古い台帳はないとして何もしない。
- cli の helper `resolveOrgLedger(cmd, explicit, access)`(`internal/cli/org_legacy_ledger.go:74`)は、台帳を決め、`checkRetiredOrgStateDirEnv` で撤去した env を確かめ、`guardLegacyOrgStateDir`(`:150`、linked worktree が残した古い台帳の検査。この PR より前からある)を同じ `access` で呼ぶ。`orgLedgerAccess` は `orgLedgerReadOnly` と `orgLedgerMutating`(`:21-35` に、どの動詞がどちらかを列挙)。
- 台帳を決められないとき(個人用のファイルが読めない、相対の値)のエラーは `<コマンド>: cannot resolve the org ledger: <ファイルのパスつきの原因>` で、`errOrgLedgerUnresolved`(`:49`)で包む。終了コード 1。
- 呼び出し: helper を通るのが 15 か所(`status.go:133`、`insights.go:50`、`org.go:134` の `newOrgRuntime`、`:816` の spawn、`:1017` の start、`:1118` の `runOrgStartPlan`、`:1209` の send、`org_inbox.go:99` の escalate と `:331`、`org_watch.go:112`、`org_director.go:389`、`org_wait_director.go:54`、`org_merge_order.go:144`、`org_merge.go:228`、`org_cleanup.go:141`)。doctor は `doctor_merge_guarantee.go:41` で `org.ResolveOrgStateDir` を直接呼ぶ。spawn・start・start `--plan` の guard は `newOrgSpawnRuntimeAt` から helper に移った(1 回だけ走る)。
- doctor は、台帳が決まらないとき `Org ledger` の fail の行を出して残りを続け、Merge guarantee は走らせてキャッシュを書かない(`; not cached: …`)。
- テスト: `internal/org/statedir_test.go` に `TestResolveOrgStateDir_PersonalFileTier`、`_PersonalFileOrder`、`_PersonalFileErrors`、`TestFeatureRepoRoot`。`internal/cli/org_ledger_config_test.go`(新規、472 行)に、AC1・AC2 の `TestOrgLedger_PersonalFileNamesTheLedgerFromAnyWorktree`(主 worktree の root、サブディレクトリ、linked worktree、その下のサブディレクトリで同じ台帳)と `_ConfigFlagDoesNotChooseThePersonalFile`、AC3 の逃げ道 `_RelativePersonalStateDirFailsUntilStateDirIsPassed`(`--state-dir` 付きの `org stop --all` は動き、付けない `org status` はパスを名指すエラー)、AC5 の `_UnparsablePersonalFileNamesItsPath` と `TestRunDoctorFull_PersonalFileThatCannotResolveTheLedgerIsAFailLine`。
- 検証: gofmt・vet・build rc 0、org(1245 本)と cli(870 本)を 4 分割して前景で rc 0、変異 8 つをテストが落とした。

## 4. escalate が人に届く道(S2b、c78f6903)

- S2 の直後に、implementer が後退を見つけた。S2 より前は、個人用のファイルが読めないと `newOrgRuntimeAt` の `Load` で失敗して `EscalateNotRecorded` に流れ、banner・通知・`escalations.jsonl` が人に届いた。S2 のあとは `resolveOrgLedger` が先に失敗して終了コード 1 だけになる。leader の雛形は普通の org の leader に `--state-dir` なしで escalate を打たせるので、台帳が決まらないとき人に届かない。consult(案 B)で直すと決めた。
- 直し: `internal/cli/org_inbox.go:99-101` で、`resolveOrgLedger` のエラーが `errOrgLedgerUnresolved` を持つなら、台帳のない fallback の Org(`newOrgEscalateFallbackRuntime("")`、`:365`。`DesktopNotify` だけ)で `EscalateNotRecorded`(`internal/org/escalate.go:233`)に流す。Inbox も `EscalationsPath` もないので、どの台帳にも書かない。banner は `ORG ESCALATION (NOT RECORDED): org=<id> type=<T> -- the item is not in any inbox: <原因>`。`validateEscalation` が先なので、断られた message は通知しない。古い台帳の拒否(`guardLegacyOrgStateDir`)には印を付けず、今までどおり素のエラーにする。
- D4(黙って別の台帳を使わない)とは矛盾しない。どの台帳にも書かず、人に知らせるだけ。escalate の help に 1 文を足した。
- テスト: `TestEscalateNotRecorded_WithNoLedgerWritesNothing`(`internal/org/escalate_test.go`)、`TestOrgEscalate_LedgerThatCannotBeResolvedStillReachesTheHuman`。変異 5 つをテストが落とした。

## 5. insights と doctor の `--state-dir`(S3、4505223d)

- `ralph insights --state-dir`(`internal/cli/insights.go:62`): `resolveOrgLedger` を ReadOnly で呼び、`--receipts` があればそちらが勝つ。insights は `config.Load` を通らないので、壊れた個人用のファイルでも flag があれば動く。
- `ralph doctor --state-dir`(`internal/cli/doctor.go:54` の flag): 本体を `runDoctorStateDir`(`:105`)に移し、`runDoctorFull`(`:96`)は委譲する。`mergeGuaranteeEnvFromOS`(`doctor_merge_guarantee.go:41`)は `(targetDir, explicitStateDir, explicitSet)` を取り、flag があれば git も個人用のファイルも読まない。
- 古い台帳の案内(`org_legacy_ledger.go` の `guardLegacyOrgStateDir`)は、insights にも `--state-dir` ができたので、どのコマンドにも `pass --state-dir` を言う。origin/main にあった insights 向けの `set RALPH_ORG_STATE_DIR=` の分岐は消えた。
- help は env を名指さない文にした。`--receipts` の help も合わせた。
- テスト: `TestInsightsCmd_StateDirFlagNamesTheReceiptsLedger`、`_ReceiptsFlagOverridesStateDir`、`TestInsights_PersonalFileNamesTheReceiptsLedger`、`TestInsights_BrokenPersonalFileFailsUntilStateDirOrReceiptsIsPassed`、`TestMergeGuaranteeEnvFromOS_StateDirFlagNamesTheLedgerWithoutGit`、`TestDoctor_StateDirFlagChoosesTheLedgerOverThePersonalFile`。変異 6 つをテストが落とした。

## 6. env の撤去、拒否、doctor の warn(S4、fe58f44b)

- `ResolveOrgStateDir` から env の段と `EnvOrgStateDir` を外した。env を読むのは `checkRetiredOrgStateDirEnv`(`org_legacy_ledger.go:106`、定数 `retiredOrgStateDirEnv` は `:57`)と、doctor の `checkRetiredEnv`(`doctor.go:899`)だけ。
- `checkRetiredOrgStateDirEnv` は、台帳を決めた直後、古い台帳の guard の前に呼ばれる。env が空か空白だけなら何もしない。値は `TrimSpace` して cwd 基準で絶対にする。
  - source が `flag` か `config`、または既定の段で決まった台帳が env の場所と `gitroot.SamePath` で同じ: stderr に注意を 1 行出して続ける(`--state-dir` 付きで起こした見張りの子が env を受け継ぐと、watch.log に出る 1 行もこれ。害はない)。
  - 既定の段で決まった台帳が env の場所と違い、`orgLedgerMutating`: エラーで断る(`errOrgLedgerUnresolved` で包む)。エラーは env の場所と既定の台帳の両方を名指し、`[org].state_dir` に書くか `--state-dir` を渡す道、または env を消す道を言う。呼び出し側は manifest の書き込み、見張りの起動、herdr、agmsg のどれより前にこのエラーを返す。
  - 同じで `orgLedgerReadOnly`: 「台帳を変えるコマンドは断られる」を足した注意を出して続ける。
- doctor は撤去した変数の表 `retiredEnvs`(`doctor.go:880`、`RALPH_ORG_AGMSG_HOME` と `RALPH_ORG_STATE_DIR`)の warn に広げた。値、読まなくなったこと、移し先を言う。`RALPH_ORG_STATE_DIR` の行には拒否の条件を足した(S8、下の 9 節)。
- cli と org の TestMain が `RALPH_ORG_STATE_DIR` を unset する。
- テストの書き換え: env を `t.Setenv` していた箇所 28 か所と、一覧外の 2 か所(`org_test.go` の `TestOrgSend_StateDirHint_OmitsFlagWhenResolvedFromEnv`、`status_test.go` の `TestStatusCmd_DirectorStopCommandRepeatsTheStateDir`)。台帳を指すだけのものは `--state-dir` か個人用のファイルに、env の段の振る舞いを確かめていたものは個人用のファイルの段のテストに替え、名前が変わった 2 本は `TestResolveOrgStateDir_EnvWinsOverGitAndCwd` が `_RetiredEnvIsIgnored` に、`TestOrgSend_StateDirHint_OmitsFlagWhenResolvedFromEnv` が `_OmitsFlagWhenResolvedFromThePersonalFile` になった。対応表は計画の Progress。
- 新しいテスト(`org_retired_state_dir_env_test.go`、466 行): 読むだけの動詞が注意つきで既定の台帳を読む、start・spawn・send の拒否(manifest がバイト単位で同じ、既定の台帳が作られない、herdr・agmsg の呼び出しが 0、見張りの起動の差し替え口が呼ばれない)、注意だけで動く 5 通り(`--state-dir`、`[org].state_dir`、既定と同じ、相対で同じ、空白だけ)、escalate が not recorded になる、env が古い linked worktree の台帳を指すとき stop が env の拒否になる、`TestCheckRetiredEnv`、`TestRunDoctorOpts_RetiredOrgStateDirEnv`。
- 検証: org(1246 本)と cli(884 本)を 4 分割して前景で rc 0、計画の grep の否定(`os.Getenv(EnvOrgStateDir)` が本番に 0 件)rc 0、変異 11 個をテストが落とした。

## 7. 文書と help(S5、93ad5b41)

- README: `### ralph.local.toml` の節に `[org].state_dir` の段落(`README.md:148`)。置き場所、値の形、エラー、`--state-dir` での回避、env の扱いと拒否の条件。doctor と insights の行に `[--state-dir <dir>]`。
- `/org` skill(`.claude/skills/org/SKILL.md`)の「前提」に「台帳の場所を個人用のファイルで決める」の項と、順の説明。`.claude` と `templates/base/.claude` を直し、`.agents` の 2 つは `./scripts/sync-skills.sh` で作り直した(4 つの写しは一致、`check-skill-sync` 14 本)。
- recipe 2 本(`docs/recipes/worktrees.md`、`codex-seat-permissions.md`)と `templates/base/` の写し。`RALPH_ORG_STATE_DIR` は「もう読まない」の 1 文だけ。
- `templates/base/ralph.toml` の `max_orgs` の注釈(有効な行を書くと init したすべての project が落ちるので、注釈だけ)。leader の雛形(`internal/org/prompts/leader.md`)と `internal/org/feature.go` の注釈。help(`org.go`、`status.go`、`org_inbox.go`、`doctor.go`、`insights.go`)。help と雛形を固定するテストに、env を名指さないことの否定の検査を足した(既存の検査は弱めていない)。
- sync-docs(3a6d1b11): README の `ralph config get` の行(`ralph.local.toml` にだけ書けるキーは `[org].state_dir` で、`ralph.toml` に書くと読み込みが失敗する)と `ralph status` の見出しに `[--state-dir <dir>]`、`AGENTS.md:89` の `internal/config/` の行。tech-debt の表(下の 10 節)。

## 8. #288 の cleanup の合わせ(S6、c76c2ef2)と main の取り込み

- 3700470e(`git merge origin/main`、#288 `ralph org cleanup` まで): 衝突は README と `internal/cli/org.go`。#288 は S2〜S4 より前の決め方で書かれていて、merge のコミットではビルドが通らなかった(`internal/org/cleanup.go` の `EnvOrgStateDir`、`internal/cli/org_cleanup.go` の 2 値の `ResolveOrgStateDir` と helper を通らない guard)。remote にビルドが通らない状態を置かないよう、S6 と一緒に push した。
- S6: `ralph org cleanup` の台帳を `resolveOrgLedger`(Mutating)で決め(`org_cleanup.go:141`)、`org.FeatureRepoRoot` には helper の source を渡す。台帳を動かす手順の文 `ledgerMoveStep`(`internal/org/cleanup.go:499`)は「`--state-dir` or `[org].state_dir` in ralph.local.toml」を名指す。cleanup と merge の help、README の cleanup の段落、org の skill(2 か所)と director の skill(2 か所)の「新しい場所を指す」を同じ 2 つの置き場所に。それを固定するテスト(`org_skill_send_sync_test.go`、`director_skill_sync_test.go`)に、skill が env を名指さないことの検査を足した。
- テスト: `TestOrgCleanup_RefusedWhileTheRetiredStateDirEnvNamesAnotherLedger`(exit 1、`errOrgLedgerUnresolved`、エラー文の全体、stdout 空、gh 0 回、worktree・登録・branch・記録・台帳は変わらず、`--state-dir` を付ければ注意 1 行で片付く)、`TestOrgCleanup_PersonalFileNamesTheLedger`(相対の値は exit 1 で何も消さない、絶対の値は動かした台帳にだけ `worktree_cleaned` を記録する)。変異 2 つをテストが落とした。
- 668e9349(2 回目、#289 regate-ci-checks まで): 衝突は `docs/tech-debt/README.md` の表の末尾だけで、両方の行を残した。#289 のコードは `ResolveOrgStateDir` を呼ばず、env を読まない。

## 9. レビューで足した直し(S7、S8、S9)

- self-review(2f801336): LOW 9 件、Merge: yes。
  - 直した(S7、187eaf8d): L1(env の段が書き残された注釈とドキュメント)、L4(`checkLocalOnlyKeys` のエラーのとき `Load` が ralph.toml の値入りの cfg を返し、doctor が「using defaults」と言いながらそれを使っていた。`Default()` を返す)、L6(個人用のファイルの段が `mustAbs` を通らず、`$HOME` が相対だと台帳が相対になった)。
  - L8(tech-debt が変わった事実を追っていない)は sync-docs で直した。
  - 残した: L2(env の拒否が「cannot resolve the org ledger」で始まる)、L5(git の起動が増える)は tech-debt 399 行、L3(判定と展開の重複)は 396 行、L9(cleanup と merge の案内に `--state-dir` が付かない)は 395 行、L7(source 名 `config`)は PR の Known gaps。
- verify(5e65830a): pass。V3(拒否の条件を省いた doctor の warn と escalate の help が、個人用のファイルで台帳を決めた人にも「断られる」と読める)は S8(cb33a636)で、文に「`--state-dir` も `[org].state_dir` も台帳を名指さないとき」を足し、期待値を全文一致のまま合わせた。V1(tech-debt の 14 行に持ち越しの注がない)・V2(186 行の F-7)は sync-docs で直した。
- cross-review cycle 1(ca7c651f): codex(gpt-6-astra、xhigh、read-only)の所見 1 件、ACTION_REQUIRED。`ralph.toml` の `[ORG]` / `STATE_DIR = "/ledger"` が local-only の検査をすり抜け、`ralph config get org.state_dir` は `/ledger` を答えるのに、実際のコマンドは黙って既定の台帳を使う。triager が overlay のテストで再現した。
- S9(6415aabd): `tomlDocHasKey`(`internal/config/local.go:153`)を go-toml v2.3.0 の decoder(`structFieldPath`: 完全一致、なければ `strings.ToLower` で一致)と同じ規則にし、一致したキーをすべて辿る(`[org]` と `[ORG]` は 1 つの文書の別の table で、decoder はどちらも `Org` に当てる)。`strings.EqualFold` は U+0130 と U+017F で decoder と結果が逆になるので使わない。エラー文は正規の `[org].state_dir` で名指す。
  - テスト: `TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder` は、各行を本物の `toml.Unmarshal` で `Config` に読み、その結果を期待値にして比べる(go-toml が規則を変えると、検査より先にテストの前提が落ちる)。`TestLoad_LocalOnlyKeyInRalphToml` に 7 行。
  - cycle 2 の self-review は S9 の差分に所見なし。verify は HEAD の binary で、大文字小文字を混ぜた 5 つの綴りが `config get`(exit 3)、`org status`(exit 1)で断られ、S9 の前の binary は通していたことを確かめた。
- cross-review cycle 2: codex の所見 0 件。
- sync-docs(3a6d1b11 と 5c01a2c9): tech-debt の表に、Trigger に当たった 15 行(verify の 14 行と 192 行)に持ち越しの注、186 行の (c) F-7 を取り消し線で閉じる(`resolveOrgLedger` が解消)、148・170・192 行の env の記述と 188・198 行の注の更新、新しい 6 行(395〜400 行: cleanup と merge の案内、判定と展開の重複と `Load` の途中の cfg、env の拒否の条件が 5 か所、`ralph doctor --help` の `--strict`、L2 と L5、`local_test.go` が 800 行を超えた)。

## 10. スライスと受け入れ条件(AC)

| スライス | commit | 当たる AC | 内容 |
|---|---|---|---|
| S1 | e2e7b7f1 | AC3 AC4 AC9 | `OrgConfig.StateDir`、`LocalOnlyKeys`、`LocalOrgStateDir` |
| S2 | daf8e512 | AC1 AC2 AC3 AC5 AC8 | `ResolveOrgStateDir` が `(dir, source, err)`、source `config`、`resolveOrgLedger`、doctor の fail の行 |
| S2b | c78f6903 | AC5 | escalate の NOT RECORDED(台帳を決められないとき) |
| S3 | 4505223d | AC1 AC7 | `insights --state-dir`、`doctor --state-dir`、古い台帳の案内 |
| S4 | fe58f44b | AC6 AC8 | env の撤去、`checkRetiredOrgStateDirEnv`、`retiredEnvs`、テストの書き換え |
| S5 | 93ad5b41 | AC10 | README、skill、recipe、template、leader の雛形、help |
| S6 | c76c2ef2 | AC6 AC8 AC10 | #288 の `ralph org cleanup` を helper に寄せる |
| S7 | 187eaf8d | AC4 AC8 | self-review の L1・L4・L6 |
| S8 | cb33a636 | AC6 AC10 | doctor の warn と escalate の help に拒否の条件 |
| S9 | 6415aabd | AC4 | `ralph.toml` の local-only のキーを大文字小文字を問わず断る |
| sync-docs | 3a6d1b11 5c01a2c9 | AC10 | tech-debt、README、AGENTS.md |

AC11(build、gofmt、vet、golangci-lint、`check-sync`、`check-skill-sync`、run-verify)は個別の commit に当たらず、verify の `run-static-verify.sh`(cycle 1 は 175 秒、cycle 2 は 41 秒、どちらも rc 0)と test の `run-test.sh`(cycle 1 は 1,375 秒、cycle 2 は 1,766 秒、どちらも rc 0)が見ている。

## 11. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(verify の cycle 1・cycle 2。gofmt、go vet、golangci-lint 0 件、staticcheck、shellcheck、`check-sync`(DRIFTED 0)、`check-skill-sync`(14 本)、tech-debt の plan 参照、branch の secret scan)
  - `./scripts/run-test.sh` rc 0(cycle 1 と cycle 2。shell 45 本、Go 10 package。`internal/org` 1,285 本、`internal/cli` 910 本、`internal/config` 90 本)。`func Test` はこの diff で 41 本増え、2 本が消えた
  - カバレッジ(test の cycle 1): `ResolveOrgStateDir` 100%、`LedgerMainWorktreeRoot` 100%、`resolveOrgLedger` 100%、`LocalOrgStateDir` 100%、`checkRetiredOrgStateDirEnv` 92.9%、`checkLocalOnlyKeys` 88.9%。打たれていないのは `os.Getwd` が失敗したときだけ通る守りの道など
  - 変異: implementer が S1〜S7 で 41 個、S9 で 3 個を確かめ、test が独立に 5 個(個人用のファイルの段を飛ばす、env の拒否をやめる、source `config` を注意だけの側から外す、など)。S9 の「最初の一致だけ辿る」は map の順しだいで、20 回中 19 回落ちた。すべてテストが落とした
  - HEAD の binary の probe(verify): 主 worktree の root、サブディレクトリ、linked worktree、その下から打った `ralph status --json` がどれも `state_dir_source: config`。`--state-dir` が勝つ。相対の値は `status`・`org status`・`insights`・`org stop --all` が exit 1 でパスを名指し、`--state-dir` 付きの `org stop --all` は動く。env が別の場所を指すと `org stop --all`・`org disband --all`・`org send`(TASK)は exit 1 で断られ、env の場所も既定の台帳も作られない。`RALPH_LOCAL_CONFIG=off` では個人用のファイルの段を飛ばす。help 35 コマンドの `--help` で `RALPH_ORG_STATE_DIR` に触れるのは escalate の 1 文だけ
  - 2 回目の取り込み(668e9349)のあと: `go build`、`go vet`、gofmt、`check-skill-sync`、`check-sync`、`go test ./internal/config/ ./cmd/...`、`go test ./internal/cli/ -run '<関係する分>'`
- 確かめていないこと・残っていること:
  - 2 回目の取り込みのあと、`run-test.sh` 全体と `run-verify.sh` の既定のモードは打っていない。全体は CI に任せる(この walkthrough を書いた時点で、CI の `verify` は pending)
  - `go test ./internal/org/ -run '<関係する分>'` で、#289 の `TestRegateCI_AnInterruptOnTheMergePath/while_the_push_runs` が load 約 30〜34 の中で「out/started did not appear within 10s」で 1 回落ちた。単独の `-count=3` は 3 回とも通った(4〜8 秒)。regate の経路で、この branch の台帳の決め方と cleanup を通らない。負荷で 10 秒の待ちを越えたと見ているが、未確認(tech-debt 393 行と同じ形で、#264 の側に渡す)
  - `ralph org start` と `spawn` の拒否を実機で打つこと(見張りの子を残さないため、コードの順とテストで確かめた)、本物の herdr と agmsg、実物の HOME での `~/` の展開
  - cross-review の reviewer(codex)は、read-only の sandbox で Go のビルドのディレクトリを作れず、どちらの cycle でもテストを打てていない。Go のテストの結果は `/test` による
  - Known gaps(この PR では直していない): source 名 `config` が `--config` と取り違えやすいこと(L7、計画で決めた名前)、test の Test gaps 4 つ(`FeatureRepoRoot` の bare repository のエラー分岐、`filepath.Abs` の失敗、実物の `ralph.local.toml` での手動の確かめ、`watch --detach` の子の注意)、`ralph config get --help` の exit 3 の説明が local-only のキーを名指さないこと(Go の文)、`.claude/rules/ralph/model-routing.md:151`(B の範囲)、`docs/specs/` への日付つきの箇条(non-goal)
- 注意して読んでほしい所:
  - `checkRetiredOrgStateDirEnv`(`org_legacy_ledger.go:106`)の判定の順。source が flag か config、または `SamePath` で同じ場所のときは、先に注意だけで返る。拒否は、既定の段で決まった台帳が env の場所と違い、Mutating のときだけ
  - `resolveOrgLedger` が返す 2 種類のエラー。決められない(個人用のファイルの誤り、env の拒否)は `errOrgLedgerUnresolved` を持ち、古い台帳の拒否(`guardLegacyOrgStateDir`)は持たない。escalate だけがこの印で道を分ける
  - `tomlDocHasKey` が `strings.ToLower` を使い `strings.EqualFold` を使わない理由(U+0130 と U+017F)と、一致したキーをすべて辿ること
  - 台帳を決めるのは個人用のファイルだけで、`ralph.toml` は読まない。`ralph.toml` が壊れていても台帳は決まる。個人用のファイルが TOML として読めないと、`--state-dir` を渡しても `config.Load` がエラーにする(A1 からの振る舞い)
  - doctor だけが `resolveOrgLedger` を通らず、`org.ResolveOrgStateDir` を直接呼ぶこと(fail の行で続けるため)
