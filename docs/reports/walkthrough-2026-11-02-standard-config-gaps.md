# Walkthrough: standard-config-gaps(PR #291 の既知の穴を直す)

- Date: 2026-11-02
- Plan: docs/plans/archive/2026-11-02-standard-config-gaps.md
- Issue: なし(PR #291、#276 の B1 の後続。`Refs #276`)
- Branch: fix/standard-config-gaps(分岐元 d217c575、origin/main と同じで、途中で main は動いていない)
- 読む範囲: `git diff main...HEAD` は 14 files、+1498/−239。動きを決めるのは `internal/cli/org.go`(+95/−59)だけで、あとはテスト(`org_start_plan_config_test.go` +363、`org_feature_reviewer_test.go` +182 と `org_feature_test.go` −172 の移動)、`tests/test-ralph-config.sh`(+5)、tech-debt(+9/−8)、記録(計画・報告・evidence・insight event)
- 後工程: パイプラインは cycle 1 の 1 回。/test の Run 1 でテストの穴が見つかったので S4 を足し、/self-review から /test までを cycle 1 のまま回し直した(Run 2)。self-review は Run 1 と Run 2 で LOW 11(すべて F1・F2 で直した)、verify と test は両方の Run で pass、cross-review(codex)は所見 0

```
設定の読み込みを 1 回に(S2) → 上限のテスト(S4) → テストの移動(S1) → zsh(S3) → 実物の確かめ → 文書 → 残したこと
```

## 1. 設定の読み込みを 1 回に(S2、7b9dc257、直し a12a08a5)

`ralph org start --plan` は、今まで main worktree の `ralph.toml` を 2 回読んでいました。runtime を作る `newOrgSpawnRuntimeAt` の中の `withMainWorktreeOrgLimits` が上限と承認方式のために 1 回、`resolveOrgStartReviewer` が `[reviewer]` のために 1 回です。間にファイルが書き換わると 2 つの版の値が混ざり、`config.Load` のたびに git も起動します。

今は `runOrgStartPlan`(`internal/cli/org.go:1204`)が `resolveOrgStartConfig`(`:702`)を 1 回呼びます。

1. `resolveConfig`(`:674`)で `--config` か `./ralph.toml` を 1 回読む。
2. `loadMainWorktreeConfig`(`:355`)で main worktree の `ralph.toml` を 1 回読む。`--config` があるときと、台帳が main worktree のものでないときは読まない。
3. main が当てはまれば `applyMainWorktreeOrgLimits`(`:221`)で上限(`max_orgs`、`max_total_seats`)と承認方式を main の値にし、`[reviewer]` も main から取る。当てはまらなければ 1 の値を使う。
4. `newOrgRuntimeFromConfig`(`:389`)で runtime を作り、`InboxHintStateDir` を `orgInboxHintStateDir` の規則で設定する。

読む順は今までと同じ(cwd か `--config` が先、main が後)で、決まる値も同じです。cwd が main worktree の root なら同じファイルを 2 回読みますが、「cwd の設定」と「main の上書き」という別の役の読み込みなので許しました(計画 AC2)。

まわりの形:

- `applyMainWorktreeOrgLimits` は spawn の `withMainWorktreeOrgLimits`(`:206`)も呼ぶので、上書きの規則は 1 か所です。spawn と `--plan` なしの start は、今までどおり `newOrgSpawnRuntimeAt` を通ります。
- `newOrgRuntimeAt`(`:378`)は `newOrgRuntimeFromConfig` を呼ぶ薄い包みとして残し、ほかの 10 か所の呼び出し元は変えていません。
- `InboxHintStateDir` を設定する行は `newOrgSpawnRuntimeAt` と `runOrgStartPlan` の 2 か所にあり、どちらのコメントももう一方とそれを守るテストを名指します。これが抜けると、`--state-dir` で選んだ台帳で leader が途中で残ったとき、`ralph org stop` の案内から `--state-dir` が消えます(計画の plan advisory の所見 1)。
- main の `ralph.toml` が読めないときの文は `org: load max_orgs, max_total_seats, [org.approval].mode and [reviewer] from the main worktree's <path>: <err>` の 1 つです。
- 読み込みを数えるために、パッケージの変数 `loadConfigFile`(`:685`、本番は `config.Load`)を置き、`resolveConfig` と `loadMainWorktreeConfig` がそれを通ります。

テスト(`internal/cli/org_start_plan_config_test.go`):

- `TestOrgStartPlan_ReadsEachConfigFileOnce`(`:64`): 既定の台帳、`--config`、台帳が main の外、cwd が main の root の 4 ケースで、パスごとの読み込みの回数を数える。元のコードでは 2 回を数えて落ちた。
- `TestOrgStartPlan_UnreadableConfigStartsNothing`(`:124`): 壊れた TOML の main の `ralph.toml` と `--config` のファイルで、エラーで止まり、worktree・ブランチ・記録・manifest・herdr の呼び出しを何も残さない。
- `TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir`(`:224`): `--state-dir` のときは止める案内に `--state-dir` が付き、個人用のファイルの `[org].state_dir` のときは付かない。`InboxHintStateDir` の 1 行を外すと落ちた。

## 2. 上限のテスト(S4、bd41041a、直し a52cbe35)

/test の Run 1 で、start --plan が上限を main から取ることを固定するテストがないと分かりました。`applyMainWorktreeOrgLimits` を「承認方式だけを main の値にする」形にした変異が、`internal/cli` 全体で生き残っていました。S2 で start --plan が spawn と別の経路になり、`--plan` なしの start のテストでは覆えなくなったためです。

`TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml`(`:287`)は、main の `max_orgs = 1`(または `max_total_seats = 1`)と sub の 99 で org-a を動かしておき、sub から打った start --plan が断られて何も作らないこと、`--config` と外の台帳の対照では通ることを見ます。Run 2 では同じ変異がこのテストの 2 ケースで落ちました。

## 3. テストの移動(S1、b71354d0)

`internal/cli/org_feature_test.go` の `[reviewer]` のテスト 3 本を `internal/cli/org_feature_reviewer_test.go` へ移しました(819 → 647 行)。本文は `cmp` で一致し、`go test -list` の名前は前後で同じです。

## 4. zsh(S3、2d4baa2d)

`tests/test-ralph-config.sh` の runner は、引用のない単語の並び(`$KEYS`、`$shells` など)をループで回します。zsh は `sh_word_split` がないと分割しないので、zsh のときだけ `setopt sh_word_split` を立てます。sh・bash・dash・`zsh -f` の 4 つで 272/272 です。

## 5. 実物の確かめ(/test、docs/evidence/test-2026-11-02-standard-config-gaps-live.md)

- PATH の ralph 6.0.0 では、`ralph_setting` と `ralph_setting_or_fail` が既定値を黙って返し、`ralph config get` は exit 1 と `unknown command "config"` を返す。skill はこれを off として review を飛ばす。
- fable は low・medium・high・xhigh の 4 つとも `claude-fable-5-1` で動いた。空に近い質問では thinking が 4 つとも 0 だったので計算の質問を足すと、low 0、xhigh 78 に分かれた。
- 一時の repository の `ralph.toml` に haiku・low・gpt-6-astra・low を書くと、helper がその値を読み、claude は haiku、codex は reasoning effort low で起動した。main の `.codex/config.toml` は前後で変わらなかった。

## 6. 文書

- docs/tech-debt/README.md: 406・409・410 を RESOLVED にした。名前が古くなった 148・187(d)・198(e)・208(b)・220(b) を直し、Trigger に当たったが払っていない分は注を書いた。末尾に「Small leftovers of standard-config-gaps」を足した。
- README、`/org` skill、`internal/config/config.go` は別の PR(#269)が変えている最中なので触っていない。食い違う文書はなかった。

## 7. 確かめたことと残したこと

確かめたこと:

- `./scripts/run-static-verify.sh` は verify の Run 1・Run 2 と sync-docs で rc 0
- `./scripts/run-test.sh` は Run 1(1,049 秒)と Run 2(1,096 秒)で rc 0、打ち直しなし
- 変異: S2 の数えるテストと止める案内のテスト、F1 の 3 つ、S4 の m1 と m2 は、どれもテストで落ちた

残したこと(tech-debt の「Small leftovers of standard-config-gaps」):

- main worktree の root から打つと、壊れた `ralph.toml` の文が `org: load config: …` になり、パスと値を名指さない(前からの動き)
- `--config` の壊れたファイルの文にパスがない(前からの動き)
- テストの `ralph.toml` を commit する 2 行の重複
- 前からある error の return 4 つ、`max_total_seats` の対照、外の台帳の承認方式を直接見ていないこと
- leader の通しと、codex の model の出どころ
