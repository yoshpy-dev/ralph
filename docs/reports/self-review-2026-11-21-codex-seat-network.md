# Self-review report: codex-seat-network

- Date: 2026-11-21
- Plan: `docs/plans/active/2026-11-21-codex-seat-network.md`(issue #362、Scope、AC1〜AC7、Progress checklist の S1〜S3 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 の 1 回目の run)
- Scope: `git diff origin/main...HEAD`(merge-base 09d27450、HEAD 88fde5c4、23 ファイル、+1243/-55)。コード 3 コミット(S1 574f8a6e、S2 f9113e91、S3 cbb831bc)の diff 品質だけを見た。見た 6 項目は naming、readability、null safety、exception handling、security、maintainability。ユーザーが決めた既定(codex の座席は既定で通信を許す)は所見にしない。テスト・静的解析・spec 適合・文書の同期は走らせていない

## Evidence reviewed

- `git diff origin/main...HEAD` のうち `internal/org/permissions.go`、`internal/config/config.go`、`templates/base/ralph.toml`、`internal/cli/doctor_codex_network.go`(全文)、`internal/cli/doctor.go`、`doctor_shell_alias.go`、`doctor_codex_writable_root.go`、`internal/cli/org.go` の hunk、テスト 5 ファイル(`permissions_test.go`、`spawn_test.go`、`doctor_codex_network_test.go`、`config_test.go`、`defaults_sync_test.go`)、recipe、org skill、README、evidence、tech-debt の追加行
- 引数の並びと別名: `internal/org/spawn.go:1143-1157`(`permArgs` → `codexWritableRootArgs` → `--model`)、`permissionArgsForDriver` の呼び元(`spawn.go:781`、`:949`、`:1478` と `director_start.go:1083`)。`codexAutonomousArgs` と `codexEditsArgs` を直接読むコードは `permissions.go` だけ。`slices.Concat` は新しい slice を返し、`TestPermissionArgsForDriver_CodexReturnsANewSlice` が最初の結果を書き換えて 2 回目を比べている
- 手組みの `config.OrgConfig{}`: 非テストの `OrgConfig{` は `internal/cli/org.go` のエラー返し(6 か所)と `org_director_headless.go:518`(`StopDirector` だけ。権限の引数を作らない)。`config.Load` は `Default()` から始め、鍵がなければ true のまま(`config.go:927`、`TestLoad_OrgPermissionsCodexNetworkAccess`)。`ralph.local.toml` は同じ Config への 2 回目の Unmarshal で重なる(`config.go:967`)ので、recipe の「local を上に重ねる」は合っている。`withMainWorktreeDirectorStartConfig` は `Permissions` を構造体ごと写す(`org.go` の hunk のコメントも新しい鍵を挙げている)
- doctor の判定: `codexSeatPossible`(`driver_pool` に codex)と `codexSeatModesPossible`(director 以外の役が edits か autonomous)を 11b と共有。`codex_verified` false と「立てえない」の文言は `codexNotNeededWhyA` と 11b の `not needed:` を再利用
- 文書: recipe の新しい節の主張を `internal/cli/org_reseat.go:25-29`(`orgReseatHelp`)、`internal/org/spawn.go:519,1691`(`SpawnOutcomeIdempotent`)、`internal/cli/org_reseat.go:614-659`(`start failed:`、`left stopped:`、`left running:`)と突き合わせた。recipe とその雛形の写しは `cmp` で同一。org skill の 2 写しの diff は同一
- tech-debt: 追加 1 行(spawn が呼んだ側の `ralph.toml` から `[org.permissions]` を読む)の `resolveConfig`(`internal/cli/org.go:691`)、`applyMainWorktreeOrgLimits`、`permission_mode=`(`spawn.go:1849`)は実在する。800 行の行(`docs/tech-debt/README.md:420`)は diff に触れられていない
- 前回の所見はない(cycle 1 の 1 回目)

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | 800 行のガイドの行(5 ファイルの行)の Trigger「5 つのどれかの次の変更」をこの PR が 2 ファイルで満たしたが、持ち越しの記録がない | `internal/cli/doctor.go` 1683 → 1692、`internal/config/config.go` 1195 → 1215(どちらも merge-base 09d27450 と HEAD の `wc -l`)。`docs/tech-debt/README.md:420` の行は両ファイルを挙げ、Trigger は "The next change to any of the five"。この PR の tech-debt の変更は 1 行の追加(spawn の権限の読み)だけで、この行は触れていない。S2 の進捗記録は「800 行の行に持ち越しが要る」と書くが、S3 が sync-docs に渡した 4 項目に入っていない。前例(2026-11-20 の org-merge-queue-order)は sync-docs の pass が日付つきの持ち越しを足している | sync-docs で、この行の末尾に日付つきの持ち越し(2 ファイルの行数、「コメントと鍵の追加で、move-only の分割は Scope の外」)と、Related plan の列にこの plan を足す。この PR で分割はしない |
| LOW | maintainability | `ralph org reseat` が受けるのは `ralph org start --plan` の org だけなのに、鍵のコメント 2 か所は「org の座席を立て直す」と書く。doctor の文と recipe は「feature org」と限定しており、書き分けがずれている | `templates/base/ralph.toml:233`「`ralph org reseat` starts an org's running seats again」、`internal/config/config.go:310`「starts an org's seats again」。`internal/cli/org_reseat.go:25-29`(`orgReseatHelp`)は "Only an org started with `ralph org start --plan` is accepted; any other org is refused"。`doctor_codex_network.go:19-21` は "the seats of a feature org"、recipe も同じ。雛形のコメントは下流の人が最初に読む面で、`ralph org spawn` で立てた org の人が reseat を打つと断られる | 2 か所を「a feature org's seats」にして、recipe の節を指す。どちらも既存の 1 文の言い換えで足りる |
| LOW | maintainability | 「`-c` の上書きが `~/.codex/config.toml` に勝つ」が、確かめた範囲より広く、限定なしで書かれた面がある。限定は recipe の別の段落にしかない | 確かめたのは `codex sandbox -c sandbox_mode=workspace-write -c …network_access=<bool>` を codex-cli 0.166.0 の macOS で 3 通り走らせた 1 回(`docs/evidence/codex-seat-network-2026-11-21.md`)。座席の `codex --sandbox workspace-write … -c …`(対話の TUI)は走らせておらず、evidence も「確かめていないこと」に書いている。限定あり: `permissions.go` の `codexNetworkAccessArgs` のコメント(「on codex-cli 0.166.0」)、recipe の 260 行目以降。限定なし: recipe の 206 行目「whatever `~/.codex/config.toml` says: a `-c` override wins over that file」、`templates/base/ralph.toml:229`、`internal/config/config.go:304`「either way」、org skill の 2609-2610 行目「人の codex config によらず通信を閉じる」。別名の `-c` との順序の主張は、recipe の 228-232 行目と `doctor_shell_alias.go` の新しい段落の両方に「codex のソースを読んだ。座席では動かしていない」とあり、頼まれた確認の対象はそのとおり明記されている | recipe の 206 行目の文に「(codex-cli 0.166.0 の `codex sandbox` で確かめた。座席の中では未確認)」を足し、`config.go:304` も同じ限定にする。雛形と skill は recipe の節を指すだけでもよい |
| LOW | naming | doctor の info と warn の detail が、ralph.toml だけを読む検査の結果として書かれておらず、座席の中で何が起きるかの観測のように読める | `doctor_codex_network.go:74-76`「so a check that needs the network … can reach it inside them」、`:80-83`「fails inside them」。検査は `orgCfg` だけを読み、codex を動かさず `~/.codex/config.toml` も読まない(同ファイル 28-29 行目、recipe 241-242 行目に書いてある)が、doctor の行自体にはそれがない。述語は 11b と同じで正しく、`pass`/`not needed` の文言も 11b と hook trust と揃っている(`doctor_codex_writable_root.go:936`、`doctor_codex_hook_trust.go:54`)。言い過ぎるのは結果の部分だけ | 結果の句を ralph 側の事実に寄せる。たとえば「…so codex's sandbox does not block the network for a check such as tflint's plugin init (read from ralph.toml, not probed)」。テストの文字列定数 `wantCodexSeatNetworkOpen`/`Closed` も同時に変える |
| LOW | maintainability | `testOrgConfig()` が組む `Permissions` は、新しい鍵の零値(false)を黙って持ち、`config.Default()` の true と逆になる。そのコメントは `codex_verified` の差しか言わない | `internal/org/envelope_test.go:23-26`:`// codex_verified false, not config.Default's true (issue #274)` と `OrgPermissionsConfig{CodexVerified: false}`。この設定の `CodexVerified = true` だけを足すテストは `reseat_test.go:558,1203,1489`、`reseat_start_test.go:68,137`、`spawn_layout_test.go:753`、`spawn_retry_env_test.go:95` で、その座席は今 `-c …=false` で起動する(引数を assert していないので落ちない)。本番の経路には出ない: spawn と start と reseat は `config.Load` か `Default()` から来て、`org_director_headless.go:518` の `OrgConfig{}` は stop だけ。のちにこれらのテストが引数全体を assert すると、既定と逆の `=false` を意図せず固定する | `testOrgConfig()` に `CodexNetworkAccess: true`(`Default()` と同じ)を足すか、コメントに「codex_network_access も false」と書く。足す場合、新しいテストは `Permissions` を丸ごと置き換えているので影響しない |

## Positive notes

- 引数の並び「権限の引数 → `-c …=<bool>` → `--add-dir` → `--model` → プロンプト」は、`spawn.go:1143-1157` の組み立てと `TestOrgSpawn_Codex_NetworkAccessInStartArgs`(autonomous・edits・guarded × codex_verified × 鍵の 8 通りの argv 全体)と、`--add-dir` を含む 2 つの既存テストが固定している。期待値の `codexNetworkOnArgs` は `codexNetworkAccessArgs` を呼ばずに書き下してあり、関数側の変更がテストで見える
- 別名の心配はない。`slices.Concat` は毎回新しい slice を返し、claude の分岐も毎回リテラルを作る。`permissionArgsForDriver` の doc に「non-nil の返りは新しい slice」と書いてあり、`TestPermissionArgsForDriver_CodexReturnsANewSlice` が最初の結果を書き換えて 2 回目と比べる。`codexAutonomousArgs` と `codexEditsArgs` を直接読むコードは `permissions.go` 以外にない
- guarded、claude、`codex_verified=false` は `-c` を受けない。`permissionArgsForDriver` で guarded は `nil, nil`、claude の分岐は無変更、`codex_verified=false` の edits と autonomous は `-c` を作る前にエラーを返す。`TestPermissionArgsForDriver_CodexNetworkAccess` が driver 2 × mode 3 × 2 つの鍵の全組み合わせでこれを固定している(拒否のとき `got` が nil であることも)
- false を「渡さない」ではなく `=false` で明示するので、人の設定ファイルが開けていても鍵 false で閉じる。ユーザーの決定(既定で通信を許す)は他のガードを弱めていない: `--scope` の必須、`codex_verified` の fail-closed、director の codex の断り、`--add-dir` の条件は触れていない。`-c` が触るのは `sandbox_workspace_write.network_access` だけ
- 手組みの `config.OrgConfig{}` は本番の経路に出ない(上の Evidence)。`withMainWorktreeDirectorStartConfig` が `Permissions` を構造体ごと写すので、新しい鍵は director の start でも落ちない。`internal/cli/org.go` のコメントも新しい鍵を挙げている
- doctor の述語は 11b の `codexSeatPossible` と `codexSeatModesPossible` を再利用しており、新しく書き写していない。役がすべて claude でも codex が `driver_pool` にあれば出す(`TestDoctorCheckCodexSeatNetwork_Matrix` の全 claude の `RoleModels`)、director の edits と autonomous は数えない、`--strict` でも warn は warn のまま(`TestRunDoctorFull_CodexSeatNetworkLine`)。fail を返さない
- `defaults_sync_test.go` の 3 つの検査(値の一致、鍵の削除、両側から比べる行)に新しい鍵が入っている。`config_test.go` は鍵なし、型の誤り(`"false"` も)、雛形の鍵を false に書き換えた読みまで見ている
- 実機で確かめた範囲と確かめていない範囲が evidence と recipe に分けて書かれている。座席の TUI の中の確認、設定ファイルに鍵がないマシンの既定は「未確認」とある。leader が worktree の `ralph.toml` を書き換えると座席の通信を変えられる、という範囲外の穴は、嘘のない文面で tech-debt に 1 行として残した
- `codexNetworkAccessArgs` は 1 つの bool のために `config.OrgConfig` 全体を受けるが、呼び元の `permissionArgsForDriver` が `cfg` を持ち、同じ関数の `CodexVerified` の読み方と揃うので、所見にしない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `docs/tech-debt/README.md:420`(5 ファイルが 800 行を超えている行)の持ち越し: この PR が `internal/cli/doctor.go`(1683 → 1692)と `internal/config/config.go`(1195 → 1215)に触れた | 既存の行の記録の欠け。行を足すのではなく、その行に日付つきの持ち越しを書く | move-only の分割はこの plan の Scope の外。持ち越しの記録は前例どおり sync-docs の pass | 5 ファイルのどれかの次の変更(行の Trigger のとおり) | `docs/plans/active/2026-11-21-codex-seat-network.md`(Progress checklist の S2)、この報告の L1 |

_(新しい債務の行はない。上の 1 件は既存の行への追記で、`docs/tech-debt/README.md` はこの commit では触れない: 前例どおり sync-docs が書く。)_

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。LOW は 5 件(L1 から L5)で、どれも merge を止めない
- Follow-ups: L1 は sync-docs で記録する。L2 と L3 は recipe、雛形の `ralph.toml`、`config.go` のコメントの語の直しで、直すなら 1 つの commit にまとまる。L4 は文言とテストの定数の変更で、直すなら pipeline の cycle 2 に入る(コードなので全 run のやり直しになる)。L5 は 1 行の追加で、直すなら同じ cycle にまとめる。cap が 2 なので、cycle 2 を回す場合は L2 から L5 を 1 回の修正で足りるようにまとめる。残す場合は PR 本文の Known gaps に書く
- Known gaps(この review では見ていない): 実機の座席の TUI の中で `-c` が効くこと(evidence の「確かめていないこと」)、設定ファイルに鍵がない machine の既定の値、別名の `-c` との順序(ソースを読んだだけ)
