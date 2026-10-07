# Verify report: org-state-dir-common

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-org-state-dir-common.md
- Verifier: verifier subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: 仕様への適合(AC1〜AC9)、静的解析、文書のずれ。対象は `git diff origin/main...HEAD`(merge base 1c4cea5a、HEAD 3832595b、22 ファイル、+1796/-103)。self-review の 2 回目(4961d0f5)のあとに入った折り返しの修正 3832595b を含む。テスト(`./scripts/run-test.sh`、`go test`)は /test の担当なので実行していない。HEAD からビルドしたバイナリ(`go build ./cmd/ralph`)を scratchpad の一時リポジトリで動かす観察用の probe は行った(テストスイートではない)
- Evidence: `docs/evidence/verify-2026-10-07-org-state-dir-common.log`(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-07-070540.log`

## Spec compliance

plan の承認: `./scripts/plan-visual.sh digest` は `79224e28032a` を返し、plan の `- Approved: 2026-10-07 sha256:79224e28032a` と一致した。plan の進捗の行にある承認済みの逸脱(main worktree の中では show-toplevel を使う、`--add-dir` は leader だけ、古い台帳が読めないときは書き換えの動詞を止める、`ralph insights` の注意は `RALPH_ORG_STATE_DIR` を案内する)は、どれもその内容で実装されている。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: main のチェックアウトのルートとサブディレクトリで `<main>/.harness/state/org` と `git-main-worktree` | Met(テストの実行は /test) | `internal/org/statedir.go:59-72` の順は flag → env → `gitMainWorktree` → `gitToplevel` → cwd。`gitMainWorktree`(`:137`)は `--git-dir` と `--git-common-dir` が同じとき show-toplevel を返す。テストは `TestResolveOrgStateDir_GitRoot`(`statedir_test.go:107`)、`TestResolveOrgStateDir_GitSubdirResolvesToToplevel`(`:83`)、`TestResolveOrgStateDir_LinkedWorktreeResolvesToMainWorktree`(`:203`)の main root と main subdirectory の 2 例 |
| AC2: linked worktree のルートとサブディレクトリで AC1 と同じパスと `git-main-worktree` | Met(テストの実行は /test) | linked worktree では `git worktree list --porcelain` の先頭の記録を `parseMainWorktreeRecord`(`statedir.go:173`)で読み、bare・common dir と同じパス・存在しないディレクトリなら返さない(`:158-164`)。`statedir_test.go:203` が main の外の worktree(sibling)と main の中の worktree(`.claude/worktrees/nested`、ralph の task worktree の配置)を、それぞれルートとサブディレクトリで確かめる。probe P2 では、linked worktree から打った `ralph status` が `state-dir=<main>/.harness/state/org (source: git-main-worktree)` を出した |
| AC3: bare(名前が `.git` のものを含む)の linked worktree は show-toplevel と `git-toplevel`。`--separate-git-dir <別>/.git` の main は作業ツリーの下と `git-main-worktree` | Met(テストの実行は /test) | 名前が `.git` の bare は `TestResolveOrgStateDir_BareDotGitWorktreeFallsBackToToplevel`(`statedir_test.go:238`)。名前が `.git` でない bare(`proj.git`)はテストにないので probe P1 で確かめ、`state_dir=<proj-wt>/.harness/state/org`、`state_dir_source=git-toplevel` だった。`--separate-git-dir` は `TestResolveOrgStateDir_SeparateGitDir`(`:270`)の main の例が作業ツリーの下と `git-main-worktree` を確かめ、AC にない linked worktree の例(git dir の名前が `.git` でない)も持つ。git dir の名前が `.git` の linked worktree は Non-goals に入っていて、`statedir.go:134-136` と `/org` skill の「前提」節に制約として書いてある |
| AC4: flag と env の優先順は変わらない、git の外は `cwd` | Met(テストの実行は /test) | `ResolveOrgStateDir` の flag と env の 2 段(`statedir.go:60-65`)は差分で変わっていない。`TestResolveOrgStateDir_ExplicitFlagWins` と `TestResolveOrgStateDir_EnvWinsOverGitAndCwd` は差分なし。既存のテストで変わったのは `GitSubdir…` と `GitRoot` の source の期待値(`git-toplevel` → `git-main-worktree`)だけで、パスの期待値は同じ。plan の Risks がこの書き換えを予定している。git の外は `TestResolveOrgStateDir_NonGitCwdFallsBackToCwd`(`:122`) |
| AC5: 古い台帳に動いている座席があるとき、書き換えの 6 動詞は終了コード 1、メッセージに古い台帳・新しい台帳・`--state-dir`、台帳と herdr に触らない、flag か env なら止まらない | Met(テストの実行は /test) | `ralph org` の 10 動詞を全部数えた。`guardLegacyOrgStateDir(..., orgLedgerMutating)` を通るのは spawn(`org.go:348`)、start(`:461`)、send(`:534`)、stop(`:729`)、disband(`:854`)、watch(`:949`)で、どれも `newOrgRuntimeAt` より前で返る。spawn の guard より前には `RetiredRoleInputErr` の判定しかなく、台帳と herdr には触らない。拒否の文言(`org_legacy_ledger.go:59-62`)は古い台帳のパス、座席の数、共通の台帳のパス、`--state-dir` の 2 つの選び方を含む。`TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive`(`org_legacy_ledger_test.go:108`、`--dry-run` を含む 7 例)が文言、stdout が空、古い manifest が変わらない、共通の台帳のディレクトリができない、herdr と agmsg の呼び出しが増えないことを確かめる。flag と env は `TestOrgLegacyLedger_ExplicitStateDirSkipsGuard`(`:173`) |
| AC6: 動いている座席がないとき、または読むだけの動詞では、古い台帳のパスと `--state-dir` を含む注意を stderr に 1 回出して続ける。stdout は変えない。古い台帳がない・main のチェックアウト・flag か env では出さない | Met(main のチェックアウトで台帳のディレクトリが読めない場合に例外 1 件。V-1) | 読むだけの動詞は wait(`org.go:654`)、read(`:692`)、status(`:759`)、report(`:895`)、`ralph status`(`status.go:49`)、`ralph insights`(`insights.go:49`)が `orgLedgerReadOnly` を渡す。注意は `cmd.ErrOrStderr()` にだけ書く(`org_legacy_ledger.go:70-72`)。テストは `TestOrgLegacyLedger_ReadOnlyVerbsNoteAndContinue`(`org_legacy_ledger_test.go:265`、7 例、`--json` の 3 例は stdout が JSON として読めるかも見る)、`TestOrgLegacyLedger_MutatingVerbWithoutActiveLegacySeats`(`:208`)、`TestStatusJSON_FromLinkedWorktreeReportsSharedLedger` の main の例(`:320`)。例外は V-1 を参照 |
| AC7: linked worktree からの `ralph status --json` の `state_dir` が main 側 | Met(テストの実行は /test) | `TestStatusJSON_FromLinkedWorktreeReportsSharedLedger`(`org_legacy_ledger_test.go:320`)が `state_dir` と `state_dir_source` を、古い台帳あり・なし、main から打つ場合の 3 例で比べる。probe P2 でも同じ結果 |
| AC8: leader で edits か autonomous の codex 座席で、台帳が cwd の下にないときだけ `--add-dir <台帳>`。implementer・reviewer・cwd の下・guarded・claude では付けない | Met(テストの実行は /test) | `codexWritableRootArgs`(`internal/org/permissions.go:169`)は `role != LeaderIdentity`・codex 以外・edits と autonomous 以外で nil を返す(`:170`)。包含の判定は symlink を解いて `filepath.Rel` で比べる(`pathWithinDir`、`:183`)。呼び出しは `spawn.go:762` の 1 か所で、permArgs の直後、`--model` の前に入る。`TestCodexWritableRootArgs`(`permissions_test.go:177`、18 例。名前の前方一致だけの別ディレクトリ、symlink 経由の 3 通り、implementer・reviewer・未知・空の役割を含む)と、起動の引数全体を比べる `TestOrgSpawn_Codex_WorkspaceWrite_AddDirWhenStateDirOutsideCwd`(`spawn_test.go:2195`、2 モード × 3 例) |
| AC9: `--state-dir` のヘルプ文 2 か所、`/org` skill の「前提」節 4 面、recipe 2 面、`check-skill-sync.sh` と `check-sync.sh` | Met | ヘルプ文は `org.go:42` と `status.go:56`(plan の `:53` から 3 行ずれた)。ビルドしたバイナリの `ralph org --help` に新しい順が出る(probe P3)。`ralph insights` の Long と `--receipts` のヘルプも同じ順に直してある(AC の外)。`/org` skill は「前提」節(`.claude/skills/org/SKILL.md:31-57`)、「既定の model_pool」節、permission 作法の codex の箇条(`:302-309`)を直した。4 面は `cmp` と `check-skill-sync.sh` で一致。recipe は「The org ledger and `--add-dir`」節(`docs/recipes/codex-seat-permissions.md:78`)で、root と `templates/base/` は `cmp` で一致。どちらの check も OK |

Verify plan の残りの 2 点も確かめた。台帳を移す処理は入っていない(テスト以外の Go の追加行に `os.Rename`・`os.Remove`・`io.Copy`・`os.WriteFile`・`OpenFile`・`MkdirAll`・`.Append(` がない。`LegacyWorktreeStateDir` は stat と読み取りだけ)。止める動詞と注意だけの動詞の分け方は Scope のとおり(上の AC5 と AC6 の行)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | PASS(rc 0) | Mode static、scope changed。`verify.local.sh`(shellcheck、hook の `sh -n` 20 本、settings の `jq -e` 2 本、Codex の hook guard 3 本、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt README の plan の参照)、golang verifier(gofmt ok、`go vet ./...` は出力なしで成功、golangci-lint 0 issues、staticcheck は出力なしで成功)、branch secret scan(`1c4cea5a..3832595b` clean) |
| `./scripts/check-sync.sh`(上の中) | PASS | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh`(上の中) | PASS | 13 skill が一致 |
| `git diff --check origin/main...HEAD` | PASS | 出力なし |
| 追加行の U+FFFD 検索 | PASS | 0 件 |
| ミラーの `cmp` | PASS | `/org` skill の 4 面(`.claude/` と `.agents/` の root と template。今回は frontmatter も含めて同一)、recipe の 2 面 |
| `./scripts/plan-visual.sh digest` | PASS | `79224e28032a`、plan の Approved 行と一致 |

## Documentation drift

plan の Verify plan の grep(`git grep -n 'git-toplevel\|toplevel .harness/state/org\|リポジトリルートの'`、archive・reports を除く)で残るのは、新しい順を正しく書いたコードのコメント、plan 自身、2026-08-02 の smoke の記録(`docs/evidence/org-watchdog-smoke-2026-08-02.txt:34`、履歴)、tech-debt の RESOLVED 行(`docs/tech-debt/README.md:81`、履歴)だけだった。範囲を広げて `toplevel` を大文字小文字を問わず grep し、次の 2 件を見つけた。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `--state-dir` のヘルプ文(`org.go:42`、`status.go:56`)、`ralph insights` のヘルプ | Yes | |
| `/org` skill の 4 面 | Yes | 解決順、古い台帳の扱い、`--add-dir` が leader だけであること、`--separate-git-dir` の制約 |
| `docs/recipes/codex-seat-permissions.md` の 2 面 | Yes | 折り返しの残り(V-3)は文意に関係しない |
| `docs/tech-debt/README.md:144` | No(D-1、/sync-docs に回してある) | state dir の順を「`--state-dir`、`RALPH_ORG_STATE_DIR`、git の toplevel、cwd」と書いたまま。self-review の L-5 と同じで、plan が /sync-docs に回している。この PR は行が挙げる trigger(`ResolveOrgStateDir` の変更)を満たす |
| `docs/specs/2026-10-07-org-multi-org-director.md:35`(FR-1) | 一部ずれ(D-2) | FR-1 は main worktree を「`git worktree list --porcelain` の先頭の、bare でない記録」から決めると書く。S1 の逸脱のあとの実装は、main worktree の中では show-toplevel を使い、先頭の記録は linked worktree の中だけで読む。FR-1 は詳細を 1 段目の plan に委ねているので、読む人が誤る程度は小さい。同じ spec の `:16` は直す前の状態を現在形で書き、「1 段目の PR で直している」と添えている。どちらもマージ後に /sync-docs で直すかを決めればよい |
| `docs/recipes/worktrees.md` の「Org runtime」節 | Yes(書き足す候補) | 誤った記述はない。台帳が worktree をまたいで 1 つになったことを書く場所の候補 |
| `.claude/rules/ralph/model-routing.md:100` | Yes | 「`ralph org` の動詞と同じ順」とだけ書いていて、具体的な順を持たない |

## Observational checks

HEAD 3832595b からビルドしたバイナリを、scratchpad の一時リポジトリ(`GIT_CONFIG_GLOBAL=/dev/null`、`GIT_CONFIG_NOSYSTEM=1`、`GIT_DIR` 系と `RALPH_ORG_STATE_DIR` を外した環境)で動かした。git 2.49.0、go 1.26.0、macOS。

- P1(名前が `.git` でない bare `proj.git` の linked worktree): `ralph status --json` は `state_dir=<proj-wt>/.harness/state/org`、`state_dir_source=git-toplevel`、注意なし、rc 0
- P2(通常のリポジトリの linked worktree): `ralph status` は `no org runtime state found (state-dir=<main>/.harness/state/org (source: git-main-worktree))`
- P3: `ralph org --help` の `--state-dir` は新しい順を出す
- P4(main のチェックアウトで `.harness/state/org` を `chmod 000`): V-1 の再現。`ralph status --json` は rc 1 で、stderr の 1 行目が `note: cannot read this linked worktree's older org ledger (org: stat <main>/.harness/state/org/manifest.jsonl: ... permission denied)`、2 行目が status 自身の manifest の読み取りエラー。`ralph org stop` は `org: refusing to change the org ledger: cannot tell whether this linked worktree's older ledger still has active seats (...)` で rc 1
- P5(P4 のあと権限を戻す): `ralph status --json` は rc 0、stderr は空

## Findings

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| V-1 | LOW | `LegacyWorktreeStateDir` は候補の manifest を stat し(`statedir.go:100-105`)、そのあとで候補が解決結果と同じか(main のチェックアウトか)を見る(`:106`)。このため main のチェックアウトで stat が ENOENT 以外で失敗すると(台帳のディレクトリの権限がない、`.harness/state` がファイルになっているなど)、注意と拒否が「この linked worktree の古い台帳」として出る(P4)。AC6 は main のチェックアウトでは注意を出さないと決めている。どちらの動詞も manifest を読めずに失敗するので、影響はメッセージが原因を取り違えることだけ。テストはこの場合を持たない | `samePath(candidate, resolvedDir)` の判定を `os.Stat` より前に動かす。直すなら、main のチェックアウトで stat が失敗する場合のテストを 1 つ足す。直さない場合は known gap として PR に書く |
| V-2 | LOW | 読むだけの動詞で古い台帳が読めないときの注意(`org_legacy_ledger.go:50-52`)は、古い manifest のパス(エラーの文言の中)と共通の台帳のパスを出すが、`--state-dir` で台帳を選ぶ方法は書かない。書き換えの動詞の拒否のほうは書く。AC6 の文言の要件は読める台帳の場合を想定していて、読めない場合の扱いは plan の進捗の行で決めた(中身は決めていない) | 任意。揃えるなら `--state-dir`(insights は `RALPH_ORG_STATE_DIR`)の案内を足す |
| V-3 | LOW | self-review の L-6(b) が残っている。3832595b は recipe の折り返しの位置を 1 語ずらしただけで、`docs/recipes/codex-seat-permissions.md:93` は 91 桁(前後の行は 72〜75 桁)。`templates/base/` 側も同じ。L-6(a)(`permissions.go:155-157`)は直っている | 段落を 75 桁前後で折り直す。2 面を同時に直す |

## Coverage gaps

- テストは実行していない(/test の担当)。上の表の「Met」は、コードとテストの中身を読み、テストが AC の各場合を持つことを確かめた結果で、テストが通ることは確かめていない
- codex の `--add-dir` が sandbox の中で効くこと: 実装者が codex-cli 0.160.0(macOS)で確かめた記録(`docs/evidence/codex-add-dir-2026-10-07.md`)を読んだだけで、verifier は codex を動かしていない。座席として起動した leader が実際に共通の台帳に書けるかは、herdr を使う実機の確認が要る
- 名前が `.git` でない bare リポジトリは probe P1 だけで、テストにない
- git 2.31 より古い git(`--path-format=absolute` がない)で `git-toplevel` に落ちる経路は、コードを読んだだけ(`statedir.go:142-145` の `len(dirs) != 2`)
- Windows のパスの扱い(`pathWithinDir` の区切り文字)は見ていない

## Verdict

- Verdict: pass
- Verified: AC1〜AC9 をコードとテストの中身で確かめた(AC3 の名前が `.git` でない bare と AC7 は probe でも確認)。静的解析は `run-static-verify.sh` が rc 0。plan の承認の digest が一致。台帳を移す処理がないこと、止める動詞と注意だけの動詞の分け方が Scope のとおりであること
- Partially verified: AC6 は main のチェックアウトで台帳のディレクトリが読めない場合だけ外れる(V-1、LOW)。文書は D-1(tech-debt の行、/sync-docs に回してある)と D-2(director spec の FR-1 の書き方)が残る
- Not verified: テストの実行、codex の座席としての `--add-dir` の実機の効き目、古い git での動作
