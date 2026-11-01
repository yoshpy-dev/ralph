# Self-review report: org-ledger-gaps

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-ledger-gaps.md`(PR #290 の Known gaps、S1〜S9、D1〜D7、AC1〜AC10)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base 4fef6b8c、HEAD 07433b0a、計画の追加を含めて 12 コミット、38 ファイル(うち Go は 37)、+2771/−634)。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。依頼で個別に判断を求められた 5 点は、L1〜L5 に対応させて「依頼された 5 点の判断」に書いた。
- ID: 所見は L1〜L10 で、すべて LOW。CRITICAL、HIGH、MEDIUM はない。

## Evidence reviewed

- 本番コードの diff を全部読んだ。`internal/config/homepath.go`、`local.go`、`internal/org/statedir.go`、`cleanup.go`、`merge.go`、`merge_run.go`、`driver/agmsg.go`、`internal/cli/org_cleanup.go`、`org_merge.go`、`org_legacy_ledger.go`、`org_inbox.go`、`org_watch.go`、`doctor.go`、`doctor_codex_writable_root.go`、`config.go`。
- 新しいテストは、`internal/cli/org_retry_state_dir_test.go`、`internal/org/cleanup_retry_state_dir_test.go`、`statedir_branches_test.go`、`statedir_git_calls_test.go`、`internal/cli/org_retired_state_dir_condition_test.go`、`org_retired_state_dir_env_abs_test.go`、`org_retired_state_dir_env_git_test.go`、`doctor_load_error_test.go`、`doctor_help_test.go`、`internal/config/homepath_test.go`、`local_state_dir_test.go` の `TestLocalOrgStateDirAt_*`、`internal/org/driver/agmsg_home_test.go`、`internal/cli/org_watch_agmsg_home_test.go` を読んだ。既存テストの変更(`org_cleanup_test.go`、`org_merge_test.go`、`org_inbox_test.go`、`org_retired_state_dir_env_test.go`、`cleanup_test.go`、`merge_run_test.go`)は diff で、期待文を足しただけで弱めた所がないことを見た。
- S1 の移動: `93397d1f^` の `local_test.go` の `TestLoad_LocalOnlyKeyInRalphToml` 以降を `sort` し、新しい 2 ファイルと `comm -23` で突き合わせた。旧にだけある行は 0。新にだけある行は package 宣言、import、ファイル先頭のコメントだけ。
- 再実行の文の取りこぼし: 非テストの Go ソースを `ralph org (cleanup|merge)( |")` で grep し、コメント以外の全件を見た(cleanup.go、merge.go、merge_run.go、org_cleanup.go、org_merge.go、merge_wait.go、mergeenforce.go、regate.go、spawn.go)。
- `config.LocalOrgStateDir`、`absPath`、`errOrgLedgerUnresolved`、`errOrgLedgerRefusedByRetiredEnv`、`orgInboxHintStateDir`、`CleanupParams{`、`MergeParams{` の全呼び出し元を grep した。
- `wc -l` を base(4fef6b8c)と HEAD で、diff に出た Go ファイル 37 本について比べた。
- `import` の取りこぼし: `org_watch.go`、`driver/agmsg.go`、`local.go` で `os`、`strings`、`filepath` がまだ使われていることを grep で見た。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1: `config.LocalOrgStateDir(dir)` を本番で呼ぶ所がなくなった。呼ぶのは `homepath_test.go`(2 か所)と `local_state_dir_test.go`(7 か所)のテストだけ。 | `internal/config/local.go:203`。本番の呼び出しは `statedir.go` の `config.LocalOrgStateDirAt` に移った。grep の残りは `config.go:202,204` の doc と `homepath.go` の doc のコメント。 | 残す。理由は「依頼された 5 点の判断」の 1。`local.go:203` の doc に「`ResolveOrgStateDir` は `LocalOrgStateDirAt` を使う」を 1 文足す。 |
| LOW | readability | L2: `absPath` の seam のコメントが promptFilePath のためとしか書いておらず、本番の呼び出しが 8 か所あるのに合っていない。S8 で `mustAbs` が 1 つ増えた。 | `internal/org/spawn.go:2141-2146`。本番の呼び出しは `director_cmd.go:504,515`、`director_start.go:1039,1042,1046`、`gate_run.go:109`、`spawn.go:2134`、`statedir.go:263`。テストで差し替えるのは `spawn_test.go:3461,3496` と `statedir_branches_test.go:81`。 | コメントを「本番の絶対化はここを通る。テストが差し替えると、その間に呼ばれる全部に効く」に直す。seam の共有そのものは妥当。 |
| LOW | maintainability | L3: `config.Load` の doc の理由句「`ralph doctor` goes on with it after saying it uses the defaults」が S3 で偽になった。「Config は Default() になる」という文は今も正しい。 | `internal/config/config.go:870-873`。doctor は `doctor.go:107-111` で `cfg = config.Default()` に自分で差し替える。`config.go:196-205` の StateDir の doc は `LocalOrgStateDir` を名指すが、`LocalOrgStateDirAt` と同じ読み取りなので偽ではない。 | 今は直さなくてよい(#269 が同じファイルを変えている)。#269 の merge のあとに理由句を消すか、/sync-docs が tech-debt に 1 行書く。 |
| LOW | maintainability | L4: このブランチで 800 行を越えたファイルはない。すでに越えていて伸びたファイルがあり、最大は `cleanup.go` の +30。verb に依らない `orgRetryCommand` が、1119 行のそのファイルに置いてある。 | `wc -l`(base → HEAD): `cleanup.go` 1089 → 1119、`doctor.go` 1514 → 1519、`org_watch.go` 852 → 856、`org_cleanup_test.go` 1266 → 1280、`org_merge_test.go` 1160 → 1163、`merge_run_test.go` 1107 → 1111、`doctor_codex_writable_root.go` 1024 → 1025、`cleanup_test.go` 1816 → 1817。`internal/org/cleanup.go:298` に `orgRetryCommand`。 | 直さなくてよい。`cleanupShellWord`(同じファイル)が近いので置き場に理由はある。分割は計画の Non-goals と同じ扱いで、別の PR。 |
| LOW | consistency | L5: 同じコマンドの綴りが、org 層と CLI 層で変わりうる。org 層の `cleanupShellWord` は `: @ + ,` を quote しない。CLI 層の `shellQuoteIfNeeded` は quote する。`--state-dir` の値と branch に `:` などがあると、1 回の `ralph org merge` の出力に `/a:b` と `'/a:b'` の両方が出る。どちらも shell では同じ 1 語。 | `internal/org/cleanup.go:93`(`cleanupPlainWord`)、`internal/cli/org.go:615`(`shellSafeUnquoted`)。差は `org.go:612-615` のコメントが認めている既存のもので、この PR は `--state-dir` の値にそれを広げた。 | 直さなくてよい。直すなら、どちらかの正規表現を片方に合わせる別の変更。この PR のテストは平易な path だけを使うので影響しない。 |
| LOW | readability | L6: help の文「run ralph org cleanup --branch <branch> again, with the --state-dir cleanup was given, if any, as the reason names it」で、`as the reason names it` がどこにかかるか読み取れない。 | `internal/cli/org_cleanup.go:109-111`、`org_merge.go:147-149`。原文はテストが固定している(`org_merge_test.go:1099-1101`、`org_cleanup_test.go` の help のテスト)。 | 例: 「…again; the command in the reason repeats the --state-dir cleanup was given, if any.」。直すなら help と 2 つのテストの文を一緒に。 |
| LOW | maintainability | L7: `checkRetiredOrgStateDirEnv` の `!ok` の分岐は、ledger が `.harness/state/org` で終わらないときだけ通り、`config.LocalPath("ralph.toml")` で git を起動する。`ResolveOrgStateDir` の返す source ではこの分岐に来ず、AC7 の「拒否は git を打たない」と文字の上で食い違う。直接呼ぶ 3 つのテストはどれも条件に合う ledger を渡す。 | `internal/cli/org_legacy_ledger.go:152-155`。テストは `org_retired_state_dir_condition_test.go:55`、`org_retired_state_dir_env_abs_test.go:26`、`org_retired_state_dir_env_git_test.go`。coverage は取っていないので、他のテストが通す可能性は未確認。 | 防御として残してよい。残すなら、コメントに「通常は来ない。来たときだけ git を打つ」と書く。または `!ok` のときはファイルのパスを文から外す。 |
| LOW | maintainability | L8: テストの期待文の作り方にぶれがある。(a) `fx.stateDir` を quote せずに足す所(`org_merge_test.go:662,873,899`)と `shellQuoteIfNeeded(...)` を通す所(`:707`、`org_cleanup_test.go:192,254`)が混在する。前者は org 層の `cleanupShellWord` に合わせたもので、今は通るが、`t.TempDir()` の path に空白などが入ると落ちる(Go の TempDir は subtest 名の空白や `=` を残す。おそらく。未確認)。(b) `cleanupRetryAgain` は hint を `'` で手で囲む。 (c) `TestRetiredOrgStateDirRefusalCondition` は case に依らない 2 つの確認(doctor の warn と escalate の help)を 4 つの subtest のそれぞれで繰り返す。 | (b) `internal/org/cleanup_retry_state_dir_test.go:46-51`。呼ぶのは `""` と `/my ledgers/org` だけで、空白のない hint を渡すと期待が合わない。(c) `org_retired_state_dir_condition_test.go:118-123`。 | (a) どちらの層が書く文かをコメントで一言書くか、期待を `cleanupShellWord(fx.stateDir)` で作る。(b) `orgRetryCommand` の期待を使う。(c) ループの外へ出す。 |
| LOW | design | L9: `internal/org/driver` は ralph 内部の package を 1 つも import しない leaf だったが、`expandTilde` のために `internal/config` を import するようになった。`config` は go-toml と `gitroot` を連れてくる。循環はない(計画の Assumptions で確認済み)。 | `internal/org/driver/agmsg.go:11`。driver の非テストの import は、この 1 行だけが ralph 内部。`config` の内部 import は `gitroot` 1 つ。 | 許容。今後 `ExpandHomePath` と `IsAbsOrHomePath` を使う所が driver 以外にも増えるなら、leaf の小さな package に出す。 |
| LOW | readability | L10: 名前と置き場の小さな点。(a) `defaultOrgStateDirBase`(git を探す)と `defaultOrgStateDirBaseOf`(ledger のパスから逆算する)は名前が 1 語違いで、向きが逆。(b) `wrapHelpLines` は `cli/config.go` にあるが `org_inbox.go` も使う。幅の 74 が 2 か所に直書き。 | (a) `internal/org/statedir.go:96,143`。(b) `internal/cli/config.go:76,119`、`org_inbox.go:85`。 | (a) 後者を `ledgerBaseDir` のように向きの分かる名前に。(b) 直さなくてよい。使う所がさらに増えたら共有ファイルへ。 |

### 依頼された 5 点の判断

1. `config.LocalOrgStateDir(dir)`: 残す(L1)。消すか unexport すると、S1 で中身を変えずに移したテスト群(9 か所)を書き換えることになる。`config.go:202-204` の doc も名指しているので、#269 が触るファイルにも触れる。`LocalConfigEnv=off` のとき git を起動しない保証(`local_state_dir_test.go:182,212`)は、`dir` から personal file を探すこちら側にだけある。`LocalOrgStateDirAt` とは「git を打つか」で役割が分かれており、重複しているのは `off` の確認 1 行だけ。
2. `absPath` の共有: 妥当(L2)。S8 より前から本番の呼び出しが 7 か所あり、seam はすでに共有されていた。直すのはコメントだけ。テストの差し替えは `t.Cleanup` か `defer` で戻しており、新しいテストに `t.Parallel` はない。
3. `config.go` の 2 か所の doc: 今は直さなくてよい(L3)。StateDir の doc は偽ではない。Load の doc は理由句だけが偽で、文の主張は今も正しい。変更は 1 文なので、#269 の merge のあとに直せる。それまで /sync-docs が tech-debt に 1 行書けば、忘れない。
4. 800 行: このブランチで越えたファイルはない(L4)。すでに越えていて伸びたファイルと行数は L4 の Evidence に全部ある。800 行以下のまま伸びたのは `config_test.go` 568 → 618、`merge.go` 714 → 723、`org_cleanup.go` 461 → 493、`statedir.go` 225 → 268、`local.go` 234 → 248。縮んだのは `local_test.go` 1011 → 600(AC8 の目標 800 以下を満たす)。新しいテストファイルは 14 本で、最大は `org_retry_state_dir_test.go` 324 行、次が `local_state_dir_test.go` 314 行と `cleanup_retry_state_dir_test.go` 280 行。新しい本番ファイルは `homepath.go` 36 行。
5. `HintStateDir` と `orgRetryCommand` と sentinel:
   - 名前は `Org.InboxHintStateDir`、`orgInboxHintStateDir` と同じ語で揃っている。`orgRetryCommand` の verb は文字列で、呼び出し元は cleanup と merge の 2 つだけ。テストが両方を固定しているので、型にしなくてよい。
   - 取りこぼしはない。再実行を促す文は、cleanup.go、merge.go、merge_run.go、org_cleanup.go、org_merge.go のすべてが `orgRetryCommand` か `cleanupRetryCommand` を通る。`ledgerMoveStep` は意図どおり元の `--state-dir` を付けない。残るのは、help の説明文、`ralph org merge-order` への案内(計画が残す穴と決めたもの)、`regate.go:782`(計画の Non-goals)、`spawn.go:1496` の説明文。
   - 二重付与はない。`--state-dir` は 1 文に 1 回で、`assertCleanupRetry` と `org_retry_state_dir_test.go:101` が数えている。merge の branch が quote されるようになったのは良い変更(以前の `m.branch` は生のまま)。
   - `errOrgLedgerRefusedByRetiredEnv` の使い道は、escalate の `errors.Is` 1 か所と、文の頭を `%w` で出すことの 2 つ。名前は `errOrgLedgerUnresolved` と並んで読める。doc の「handles only this one as it handles errOrgLedgerUnresolved」(`org_legacy_ledger.go:55`)の `this one` が何を指すかは、1 度読み返さないと分からない(L10 に入れるほどではない)。

## Positive notes

- 再実行のコマンドの作り方が org 層の `orgRetryCommand` と CLI 層の `cleanupRetryCommand` の 1 つずつに集まり、`--state-dir` の値は shell の 1 語に quote される。出力のコマンドを `sh` で語に分け、2 つの台帳のうち最初の台帳だけが読まれることを確かめるテストがある(`org_retry_state_dir_test.go:200,264`)。
- S5 の `defaultOrgStateDirBase` は `config.LocalPath` と同じ探し方を書き写したものだが、`TestLedgerLocalConfigPath` が 4 つの経路(main checkout、linked worktree の subdirectory、bare の linked worktree、git の外)で、`config.LocalPath("ralph.toml")` と一致することを固定している。写しが将来ずれたときに落ちる。
- `ExpandHomePath` は `os.UserHomeDir` のエラーを包まずに返し、`LocalOrgStateDir` の既存の文を変えない。`expandTilde` はエラーで値をそのまま返す今の振る舞いを、専用のテストで保っている(`agmsg_home_test.go`)。
- doctor の `cfg = config.Default()` は 5 行で、コメントが理由を言い、`doctor_load_error_test.go` が Load の 2 つの失敗経路で「途中まで埋めた値を読まない」ことを確かめている。
- 秘密、デバッグの残り、範囲外のファイルは見つからなかった。

## Coverage gaps

- L7 の `!ok` の分岐を通すテストの有無は、coverage を取っていないので未確認(/test で見る)。
- `os.Getwd` の失敗は macOS で起こせず、`filepath.Abs` の失敗の 2 つの分岐は seam の差し替えで通している。Linux で実際に起きるかは未確認(計画の S8 にも同じ記録がある)。
- 計画の AC11(tech-debt 395〜400 行の書き換え)と、PR #290 の test gaps 3・4(実物の binary の確かめ)は、この diff にまだない。/sync-docs と /test の担当。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はない。L1〜L10 は LOW で、どれも merge を止めない)
- Follow-ups:
  - L2 は 1 つのコメントの直しで、`spawn.go` はほかの session が触っていない。次の slice か /sync-docs の前に直してよい。
  - L3 は #269 の merge のあとに直すか、/sync-docs が tech-debt に 1 行書く。
  - L1 は `local.go:203` の doc に 1 文。L6 は help とテストの文を一緒に直す。
  - L4、L5、L9、L10 は直さなくてよい。L7、L8 は任意。
