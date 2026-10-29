# Verify report: local-config-org

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-local-config-org.md`(issue #276 の 1 本目 A1、AC1〜AC13、D1〜D10)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base = origin/main 5eba4013、HEAD 27f43d67、72 ファイル、+2579/-968)。spec の準拠と静的解析だけを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。S5 の verify は Progress checklist で改めた形(README を grep の対象から外して読む)を使った。Rollout の「`/ralph.local.toml`」は既知のずれとして扱った(本文は digest を保つため直していない)。
- Evidence: `docs/evidence/verify-2026-10-29-local-config-org.log`(gitignore の対象でコミットしない。run-static-verify の全出力、probe の出力、doctor の差分)。run-verify 自身のログは `docs/evidence/verify-2026-10-29-123935.log`。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 個人用のファイルがキーごとに上書きし、表はキーごと、配列は置き換え、`ralph.toml` がなくても読み、`config get` が上書きした値を返す | 満たす | `internal/config/config.go:695`(`Load`)は `ralph.toml` のあと `:726` で同じ `Config` に個人用の文書を `Unmarshal` し、検査は `:754` の `validateLoaded` で 1 回。probe(HEAD から build した binary): `ralph.toml` の `max_seats = 4` に `driver_pool` だけの個人用ファイルを重ねると 4 のまま(表の別キーが残る)、`ralph.toml` なしの dir で `max_seats = 7` を返す、個人用の `[[org.model_pool]]` が `ralph.toml` の 2 件を置き換え `[org.roles].implementer = ["opus"]` が pool から外れて exit 3。テストは `internal/config/local_test.go` の `TestLoad_LocalFileOverridesKeyByKey`(:198)、`TestLoad_LocalFileReplacesArrays`(:235)、`TestLoad_LocalFileWithoutRalphToml`(:264) |
| AC2 git の中では主 worktree の root のファイルを読み、linked worktree とサブディレクトリの同名ファイルは読まない。3 か所から打った `config get org.agmsg_home` が同じ | 満たす | `internal/config/local.go:42`(`LocalPath`)は dir を `filepath.Abs` してから `gitroot.MainWorktree`、次に `gitroot.Toplevel`、最後に dir。probe: 一時 repo の main root、`main/pkg/sub`(`ralph.toml` なし)、linked worktree の root、`linked/pkg/sub` の 4 か所すべてで `/abs/of-main-root-local`、exit 0。`linked/ralph.local.toml` と `main/pkg/sub/ralph.local.toml` の値は出ない。テストは `internal/cli/config_test.go` の `TestConfigGet_ReadsTheMainWorktreesLocalFileFromAnyWorktree` |
| AC3 読めない・正しくない個人用のファイルは Load のエラーで、文にパスが入る。`config get` は exit 3 | 満たす(所見 V2 あり) | `internal/config/local.go:67-80`(`fs.ErrNotExist` だけを「なし」とする)、`config.go:726`・`:742` がパスを付ける。probe: ディレクトリ、壊れた TOML、`max_seats = 0` の 3 つとも exit 3 で、文に `<...>/ralph.local.toml` が入る。ただし文の頭は `config get: ralph.toml: cannot be read or is invalid:` になる(V2) |
| AC4 `RALPH_LOCAL_CONFIG=off` で読まない。gate のスクリプトは呼んだ側の値にかかわらず `=off` で走る | 満たす(gate 側はコードとテストの存在で確認) | `internal/config/local.go:68` はちょうど `off` のときだけ git も呼ばずに戻る。probe: 未設定 9、`off` 4、`OFF` 9、空 9(ralph.toml の値は 4、個人用は 9)。gate は `internal/org/gate_run.go:75` で受け継いだ `RALPH_LOCAL_CONFIG` を名前の一致で落とし、`:83` で `RALPH_VERIFY_SCOPE=full` と `RALPH_LOCAL_CONFIG=off` を足す(`:53`)。`Gate` と `Regate` は同じ関数を通る。テストは `internal/org/gate_test.go:1303` と `TestGateScriptEnv`(:1342)。実際の gate は打っていない |
| AC5 ルートとテンプレートの管理ブロックに先頭 `/` なしの `ralph.local.toml` があり同じ。`git check-ignore` が 2 つのパスで通る | 満たす | `.gitignore:25` と `templates/base/.gitignore:25`。2 つのファイルは `cmp` で一致し、`# BEGIN RALPH MANAGED`〜`# END RALPH MANAGED`(:3〜:66)を切り出した 64 行も一致。`git check-ignore -v ralph.local.toml config/ralph.local.toml sub/dir/ralph.local.toml` は 3 つとも `.gitignore:25` に当たり、`-q` は両方 exit 0。`ralph.toml` は無視されない(exit 1) |
| AC6 `RALPH_ORG_AGMSG_HOME` を設定しても org の verb と doctor の agmsg の場所は変わらない。production で読むのは doctor の warn だけ | 満たす | `git grep -n 'RALPH_ORG_AGMSG_HOME\|AgmsgHomeEnv' -- internal cmd ':!*_test.go'` は `internal/cli/doctor.go` の 139〜142・839〜862 行(`legacyAgmsgHomeEnv` と warn)と `internal/org/driver/agmsg.go:154` の注釈 1 行だけ。`ResolveAgmsgHome`(`agmsg.go:157`)は上書き口 → cfgValue → 既定で、env を読まない。`scripts/`、`templates/`、`tests/`、`.claude/`、`.agents/`、`.codex/` の生きた参照は 0 件。probe: `RALPH_ORG_AGMSG_HOME=/elsewhere/agmsg` でも `config get org.agmsg_home` は個人用の `/abs/doc-agmsg`、doctor の agmsg の 2 つの check もどちらも `/abs/doc-agmsg` を見る |
| AC7 個人用のファイルの相対の `agmsg_home` は Load が断る。`--config` の相対は watch の起動でも director start と同じ文で断る | 満たす(watch 側はコードとテストの存在で確認) | Load: `internal/config/local.go:91-106`。probe: 個人用の `rel/agmsg` は exit 3、`~/x/agmsg` と `/abs/agmsg` は通り、`ralph.toml` の相対 `rel/agmsg` は今までどおり通る。watch: `internal/cli/org_watch.go:405-407` の `orgWatchChild` の先頭で、director start の `checkStartAgmsgHome`(`internal/cli/org_director_headless.go:375-378`)と同じ条件・同じ `relativeAgmsgHomeError`(`org_watch.go:450`)で断る。違うのは前置きと「何がそこで動くか」(`the watch runs` / `the director and the watch run`)だけ。テストは文全体を比べる(`org_director_headless_test.go:623` の雛形を `org_start_watch_test.go:707` と `org_watch_test.go:1135` が使う)。watch の実起動は打っていない |
| AC8 launch_cmd の起動スクリプトは `RALPH_ORG_AGMSG_HOME` を export しない。director start は env の警告を出さない | 満たす | `internal/org/director_cmd.go:259` の `directorLaunchScript` から `AgmsgHome` が消え、`text()` が export するのは `:317` の `RALPH_DIRECTOR_TOKEN` と `:318` の `driver.SeatPaneEnv()` だけ。`DirectorStartParams` からも消えた(`director_start.go`)。`checkStartAgmsgHome` は env を読まない。テストは `TestOrgDirectorStart_TheAgmsgHomeVariableGetsNoWarning`(`org_director_headless_test.go:723`)と `TestOrgDirectorStart_LaunchCmdPassesTheDirectorNoAgmsgHomeVariable`(:2196) |
| AC9 doctor は env があれば warn を出す | 満たす | `internal/cli/doctor.go:142` と `:853`(`checkLegacyAgmsgHomeEnv`、空なら行なし)。probe: env ありとなしの doctor の出力の差は 1 行だけで、`⚠ RALPH_ORG_AGMSG_HOME: warn — set to "/elsewhere/agmsg", but ralph no longer reads it; write the value in [org].agmsg_home of ralph.local.toml ...` |
| AC10 テストは本物の agmsg と開発機の個人用のファイルに触れない。cli と org の TestMain が上書き口を存在しないパスに、cli・org・config の TestMain が `RALPH_LOCAL_CONFIG=off` に固定する | 満たす(静的に確認) | `internal/cli/main_test.go:127`(off)・`:131`(旧 env を unset)・`:135`(上書き口)、`internal/org/main_test.go:29`・`:33`、`internal/config/main_test.go:18`。`config.Load` を呼ぶ package は `internal/cli` と `internal/org` だけで、`internal/cli`・`internal/org`・`internal/config` を import するテストもこの 3 つ以外にない。driver のテスト(`agmsg_test.go:313`)は文字列を比べるだけで agmsg を動かさない。実行時に触れないことはテストを走らせないと言えないので、/test の範囲 |
| AC11 ledger の決まり方は変わらない。config と org が同じ helper を使う | 満たす(静的に確認) | `internal/gitroot/gitroot.go` の `MainWorktree`・`parseMainWorktreeRecord`・`Toplevel`・`SamePath` は、移す前の `statedir.go` の関数と `-C dir` 以外は同じで、dir が空なら引数列も同じ(`command()`、:125)。`internal/org/statedir.go:67`・`:70` ほかが `gitroot.*("")` を呼び、`internal/config/local.go:47`・`:50` も同じ関数を呼ぶ。旧名(`gitMainWorktree`・`gitToplevel`・`samePath`)の残りは 0 件。`TestParseMainWorktreeRecord` は `gitroot_test.go:218` に移った。`statedir_test.go` が通ることは /test の範囲 |
| AC12 README、テンプレートの `ralph.toml`、`/org` と `/director` skill、recipe 2 本、AGENTS.md が新しい決まりを書き、env を設定の手段として書かない | 満たす | 改めた S5 の grep(`templates/base/ralph.toml`、skill の 4 つの写し、recipe の 2 つの写し、AGENTS.md)は 0 件。README の言及は `README.md:146` の「ralph no longer reads」1 件だけで、読んで確かめた。README は `### ralph.local.toml`(:142)の節を持つ。`templates/base/ralph.toml:2`・`:89`、`AGENTS.md:89`・`:90`。skill と recipe はルートとテンプレートの写しが `cmp` で一致 |
| AC13 check-skill-sync、check-sync、Go のテスト、run-verify が通る | 一部(静的の部分は満たす) | run-static-verify の中で `check-skill-sync: 14 skill(s) in lock-step`、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、全体 `All verifiers passed.`(rc=0)。Go のテストと run-verify の既定モード(テストを含む)は /test の範囲で、ここでは打っていない |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0、45 秒) | `Language scope: full fallback (unclassified:.gitignore)` なので、changed の既定でも Go は全体で走った。gofmt `ok`、go vet は無出力、golangci-lint `0 issues.`、staticcheck は無出力(`~/go/bin/staticcheck` が PATH にある)。local verifier の shellcheck(hook と verify のスクリプト)、`sh -n`、awk の構文、settings の `jq -e`、Codex の hook の検査、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画参照もすべて OK。branch の secret scan は `5eba4013..27f43d67` で clean |
| `golangci-lint run ./...` / `staticcheck ./...` の単独実行 | 打っていない | どちらも run-static-verify の中で走ったため(lead の指示どおり) |
| `shellcheck scripts/ralph-config.sh templates/base/scripts/ralph-config.sh tests/test-ralph-config.sh`(既定の severity) | pass(rc=0) | run-static-verify の shellcheck は `--severity=warning` なので、変えた 3 本を info まで見た。2 つの `ralph-config.sh` は `cmp` で一致、`sh -n` と `bash -n` も rc=0 |
| `go version` と `go.mod:3` | 一致 | どちらも 1.27.3 |
| `go build -o <scratch>/ralph ./cmd/ralph` | pass(rc=0) | probe 用。build の前後で `git status --porcelain` は空 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README(`ralph.local.toml` の節、`config get`、watch、director、gate、merge-order) | はい | :142〜:146 の節が置き場所、上書きの決まり、git の無視、エラー、`agmsg_home` の形、env を読まなくなったことを書く。gate の段落に `RALPH_LOCAL_CONFIG=off` |
| `templates/base/ralph.toml`、AGENTS.md、`/org` と `/director` skill、recipe 2 本 | はい | 上の AC12。`config get` の help(`internal/cli/config.go`)、gate の help と `GateParams.Env` の doc、org start と watch `--detach` の help にも新しい決まりが入っている |
| `scripts/ralph-config.sh` と `defaults_sync_test.go`、`tests/test-ralph-config.sh` | はい | `RALPH_ORG_AGMSG_HOME` の行と説明、それを突き合わせる検査が消えた |
| `docs/tech-debt/README.md:272`(D13 の変異 2 つ) | いいえ(V1、LOW) | Trigger「The next edit of `checkStartAgmsgHome`」がこの PR で引かれたが、求める 2 本のテストも持ち越しの注もない |
| `docs/tech-debt/README.md:198`(j) | 判断が要る(I1、INFO) | Trigger「the next change to `gitroot.SamePath` or `LedgerMainWorktreeRoot`」。この PR は `samePath` を中身を変えずに移し、Trigger の名前も書き換えた。求める case(シンボリックリンクの親の下の、まだない ledger)は `TestSamePath`(`internal/gitroot/gitroot_test.go:190`)にない |
| `docs/tech-debt/README.md:209`(RESOLVED の HTML コメント) | 履歴として可(I2、INFO) | この PR が消した `orgWatchChildAgmsgHome` と 2 本のテスト(`TestOrgWatchDetach_ARelativeAgmsgHomeIsPassedAbsoluteToTheRelocatedChild`、`TestOrgStart_ARelativeAgmsgHomeIsPassedAbsoluteToTheWatch`)を今の形で名指している |
| `docs/specs/2026-10-07-org-multi-org-director.md:110`(FR-6)、`:144`(FR-9) | 履歴として可(I3、INFO) | 見張りの子が絶対パスにした `RALPH_ORG_AGMSG_HOME` を受け取ること、起動スクリプトがそれを export することを書いた日付つきの箇条。どちらもその FR の最新の箇条ではなく、計画は過去の仕様を書き換えないと決めている。env がなくなったことを言う後の箇条はない |
| 計画の Assumptions「`internal/org` の TestMain は `statedir_test.go` にある」 | ずれ(I4、INFO) | origin/main の `internal/org` に TestMain はなく(`git grep -n 'func TestMain(' origin/main -- internal/org` は 0 件)、この PR が `internal/org/main_test.go` を新しく作った。Progress には S4b の行で `main_test.go` の名前が出るだけで、前提が外れたことは書かれていない |
| 計画の Rollout の「`/ralph.local.toml`」 | 既知のずれ | lead の指示どおり。実装と README は先頭の `/` なし |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | tech-debt の行の Trigger を引いたのに、行が古いまま。行は「`checkStartAgmsgHome` を次に編集するとき、`--config ""` で start が `launched` まで進むテストと、`enabled = false` と相対の `agmsg_home` の両方を持つ `--config` で、どちらの拒否が勝つかを固定するテストを足す」と書く。この PR は `checkStartAgmsgHome` の引数と本体と拒否の文を変えたが、director start の側に 2 本はない。`--config ""` の行は watch 側(`org_watch_test.go:1116`)にだけある | `docs/tech-debt/README.md:272`、`internal/cli/org_director_headless.go:375-378`、`git grep -- '"--config", ""' internal/cli/*_test.go` は `org_watch_test.go:1116` の 1 件だけ | 2 本を足すか、行に「carried over by feat/local-config-org」の注を足す(/sync-docs か後の slice) |
| V2 | LOW | `ralph config get` は Load のエラーをすべて `config get: <config のパス>: cannot be read or is invalid: ...` で包むので、個人用のファイルのエラーも `ralph.toml:` で始まる。後ろに個人用のファイルのパスが入るので AC3 は満たすが、最初のラベルは別のファイルを指す。self-review の L3(doctor の `Name: "ralph.toml"`)と同じ種類 | `internal/cli/config.go:84`、probe の出力 `config get: ralph.toml: cannot be read or is invalid: read <...>/ralph.local.toml: is a directory`、`internal/cli/doctor.go:98` | L3 と一緒に直すか、残すと決める。直すなら、個人用のファイルのエラーのときラベルを外すかパスを差し替える(help と README の文も確かめる) |
| I1 | INFO | 上の表の tech-debt 198 行(j) | 同上 | 純粋な移動を「変更」と見るかを決め、見るなら持ち越しの注 |
| I2 | INFO | 上の表の tech-debt 209 行の HTML コメント | 同上 | 残してよい。必要なら「S4b で消えた」の 1 文 |
| I3 | INFO | 上の表の multi-org spec の 2 つの箇条 | 同上 | 任意。#265 と #267 にならい、日付つきの箇条で env をやめたことを書ける |
| I4 | INFO | 計画の Assumptions の 1 つが外れ、Progress に記録がない | 同上 | 任意。Progress に 1 行(再実行: e13b217a が Progress の 204 行に記録した。対応済み) |
| V3 | LOW | (再実行で発見)tech-debt 366 行(`TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` の race)の Trigger「the next change to `org_watch_test.go`」に、S4b の `internal/cli/org_watch_test.go` の変更(+54/-43)が当たった。行は origin/main と同じバイト列のままで、race も残る(`:1340-1341` は `stdin: eof` を書いてから `trap '' TERM` を置く)。S6 が直した 287 行と同じ形の race | 下の「Re-run after S6」の Trigger の表、`docs/tech-debt/README.md:366`、`internal/cli/org_watch_test.go:1340-1341` | 287 行と同じ判断をする。行の Trigger が書く 1 行の直し(trap を `stdin:` の行の前に移す)を入れるか、/sync-docs で持ち越しの注を足す |
| V4 | LOW | (再実行で発見)ほかに 5 行の Trigger にこの branch が当たり、どの行にも持ち越しの注がない: 144(`doctor_codex_writable_root_test.go`)、235 と 364(`gate.go`・`gate_run.go`)、316(`director_start.go`)、367(recipe 2 本の 2 つの写し)、177 の (b)(`newOrgStartCmd`。(b) 自体は main の 216b2cdd ですでに直っている) | 同上 | /sync-docs で各行に「carried over by feat/local-config-org」の注を足す。177 の (b) は取り消す |
| I5 | INFO | (再実行で発見)287 行の上の HTML コメントの「up to 10 s like the other marker waits of the package」は、ほかの 4 つの `waitForFileWithin` のうち 3 つにしか合わない(`director_cmd_test.go:1155` は 30 s)。描画では見えない | `internal/org/director_cmd_test.go:1073`・`:1155`、`internal/org/director_stop_test.go:1128`・`:1223` | 任意。「like most of the other marker waits」 |
| I6 | INFO | (再実行で発見)Trigger に当たったかどうかの判断が要る 3 件。170・171 行(`LegacyWorktreeStateDir` の中の呼び出し 2 行の付け替えだけ。I1 と同じ問い)、362 行(leader の template を描画する `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir` を変えたが、行が挙げる 5 ファイルの外)、294 行(org の `isolateGitEnv` は変わっていない。S1 が `internal/gitroot/gitroot_test.go` に同名の写しを `XDG_CONFIG_HOME` なしで足した) | 同上 | I1 と一緒に決める。当たったとみるなら持ち越しの注 |

self-review の L1〜L4 は判断し直していない(L3 は V2 と同じ種類)。再実行で L5・L6 も判断し直していない。

## Observational checks

HEAD から build した binary を scratchpad の一時ディレクトリで打った。`HOME` は一時のディレクトリにし、`RALPH_LOCAL_CONFIG`・`RALPH_ORG_AGMSG_HOME`・`GIT_DIR`・`GIT_WORK_TREE` は外した。全出力は evidence の log にある。

- AC2: 一時 repo(`git init` → 空コミット → `git worktree add`)で、main root の `ralph.local.toml` に `/abs/of-main-root-local`、main root の `ralph.toml` に `/abs/of-ralph-toml`、linked worktree の root と `main/pkg/sub` に別の値の `ralph.local.toml` を置いた。`ralph config get org.agmsg_home` は main、`main/pkg/sub`、linked、`linked/pkg/sub` の 4 か所で `/abs/of-main-root-local`、どれも exit 0。
- AC1・AC3・AC4・AC7 の Load 側: 上の表の値のとおり。
- AC6・AC9: 個人用のファイルに `/abs/doc-agmsg` を置いた git の外の dir で、`env PATH=/usr/bin:/bin ralph doctor` を env ありとなしで打った。差は warn の 1 行だけで、agmsg の check はどちらも `/abs/doc-agmsg` を見た。doctor はどちらも exit 1 だが、理由は狭めた PATH に claude と codex がないこと(2 つの fail)で、この変更とは関係ない。shell の alias の check は `ZDOTDIR` を通して実際の zsh の rc を読んだ(読むだけ)。
- `git check-ignore`(AC5)は worktree で打った。probe の一時 repo には管理ブロックがないので、そこでは無視されない(exit 1、想定どおり)。

## Coverage gaps

- Go のテスト(`go test`)と `./scripts/run-test.sh`、run-verify の既定モードは打っていない(/test の範囲)。AC10 と AC11 の実行時の確認、AC13 の Go のテストはそちらに残る。
- gate の環境(AC4 の gate 側)は、コードとテスト(`TestGateScriptEnv`)の存在で確かめた。実際の `ralph org gate` は打っていない。
- watch の起動での拒否(AC7 の後半)と起動スクリプト(AC8)は、コードと、文全体を比べるテストの存在で確かめた。detach したプロセスを残さないため、watch の実起動と director start は打っていない。
- `ralph upgrade` が管理ブロックに行を足すことは、`.gitignore` が管理ブロックの面であること(`internal/cli/init.go:498`)を読んだだけで、打っていない。
- 本物の herdr と agmsg での通しは打っていない(計画の範囲外)。
- 決定的な検査で足りないもの: AC6 の後半(「この名前を読むのは doctor だけ」)と、上書き口が production に漏れないことを固定するテストはない(self-review L4 と同じ)。足すなら、`internal/` と `cmd/` の `_test.go` 以外を歩き、`"RALPH_ORG_AGMSG_HOME"` が `doctor.go` 以外にないこと、`AgmsgHomeOverride` への代入がテスト以外にないことを見る 1 本が最小。

## Re-run after S6 (6e1f6563)

- Date: 2026-10-30 00:25〜00:37 JST(UTC では 2026-10-29)
- Verifier: verifier サブエージェント(`/verify` の再実行。cross-review の前なので cycle は 1 のまま)
- Scope: 64a20f69..6e1f6563 の 7 コミット。コードの変更は S6 の 2 コミット(321203dc、140ed1d2)だけで、`internal/org/director_cmd_test.go` の 1 hunk(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の中、+7/-3)と `docs/tech-debt/README.md` の 286〜287 行。残りの 5 コミットは記録(計画の Progress checklist、test report、self-review report、jsonl)。static はブランチ全体で打ち、tech-debt の Trigger はブランチ全体で掃き出した。テストは打っていない(/test の範囲)。
- load average(`sysctl -n vm.loadavg`): 開始時 28.75 30.94 102.16、static の直前 13.29 25.73 94.17、直後 14.23 24.05 89.64、終了時 10.24 14.66 49.64。
- Evidence: `docs/evidence/verify-2026-10-29-local-config-org.log` の「Re-run after S6」の節に、static の全出力と下の probe の出力を足した。

### 打ったもの

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0、46 秒、00:25:43〜00:26:29) | 範囲は `Language scope: full fallback (unclassified:.gitignore)` で、changed の既定のまま Go は全体で走った。gofmt `ok`、go vet と staticcheck は無出力、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、awk の構文、`jq -e`)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `5eba4013..6e1f6563` で clean。timeout には当たらず、打ち直しもしていない。run-verify 自身のログは `docs/evidence/verify-2026-10-29-152543.log` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-29-local-config-org.md` | `1609f175543b` | `- Approved:` の digest と同じ。e13b217a と 8381769c の変更は Progress checklist の中の 1 hunk(203 行から)だけ |
| `git diff --name-only <c>^ <c>`(7 コミット) | lead の説明どおり | e13b217a と 8381769c は計画だけ、148d542e は test report と jsonl、d7f09f81 は self-review report と jsonl、6e1f6563 は self-review report だけ、321203dc と 140ed1d2 は上の 2 ファイルだけ |
| `gofmt -l internal/org/director_cmd_test.go`、`git diff --check 64a20f69..HEAD`、U+FFFD の grep(S6 の 2 ファイル) | すべて空 | |

### S6 が計画の範囲に収まるか

収まる。S6 は計画の本文(Scope、Implementation outline)にないが、本文は digest を保つため直さず、Progress checklist の 206 行(consult の案 C を選んだ理由: 行の Trigger の 2 つ「次の失敗」と「`director_cmd_test.go` の次の変更」の両方にこの PR が当たり、過去 4 回の持ち越しの理由が使えない)、208 行(321203dc)、209 行(140ed1d2)に記録がある。記録どおり、触ったのはテストのファイル 1 つと tech-debt の行だけで、テストの変更は 1 本の関数の中に収まる。production のコードは変わらず、Non-goals にも触れない。

### RESOLVED の行(286〜287 行)

- 欄の数: 未エスケープの `|` で数えて、見出し(14 行)・287 行・289 行とも 5 欄。表全体では 5 欄の行が 286 行あり、ほかの 3 行(22・32・284 行)は origin/main と 64a20f69 でも同じ数で、S6 の前からある。
- 形: 287 行は Debt と Trigger の欄を取り消す `S--S-`。`<!-- RESOLVED` のすぐ下の 78 行のうち 62 行がこの形で、すぐ下の 289 行も同じ(ほかは `S----` 13、`S-----` 1、取り消しなし 2)。Debt の欄は `~~...~~ (RESOLVED 2026-10-29 in feat/local-config-org)`、Trigger の欄は 289 行と同じ `~~...~~ — RESOLVED: ...` で終わる。286 行の HTML コメントは `<!-- RESOLVED 2026-10-29 in feat/local-config-org:` で始まり `Row preserved for traceability. -->` で終わる。
- 事実の突き合わせ:

| 行の記述 | 突き合わせた先 | 結果 |
| --- | --- | --- |
| 固定の 100 ms の待ちをやめ、`trap '' TERM; : > <marker>; exec sleep 30` | `director_cmd_test.go:305-307` | 合う |
| `waitForFileWithin` で最長 10 s 待ち、過ぎたら失敗 | `:307`、helper の `:1110-1120`(`t.Fatalf`) | 合う |
| 「like the other marker waits of the package」 | ほかの 4 つの呼び出し: `director_cmd_test.go:1073`、`director_stop_test.go:1128`・`:1223` は 10 s、`director_cmd_test.go:1155` は 30 s | 3 つに合う(I5) |
| 5 回目の失敗。`run-test.sh` が rc 1 で落ちたのはこのテストだけ、load average 16.48 39.03 47.15、1 回の打ち直しで通った | test report の表の 1 と 3(15・17 行)、load の表(53 行)、Failure analysis(120 行) | 合う |
| この PR は `director_cmd_test.go` も変える | S6 の前に S4b が変えた(`git diff --stat origin/main...64a20f69` で +33/-74) | 合う |
| Related の欄に足した `docs/reports/test-2026-10-29-local-config-org.md (Failure analysis)` | 見出しは 116 行 | 合う |
| 日付 2026-10-29 | 計画の日付、321203dc の UTC の日付(2026-10-29 15:09Z。JST では 10-30 00:09) | 合う |

- `director_cmd_test.go` と S6 が使う関数(`waitForFileWithin`、`startTestProcess`、`cmdRegistrationOf` ほか)を Trigger に持つ開いた行は、287 行のほかにない。

### ブランチ全体の Trigger の掃き出し

cycle 1 の verify は、差分が触った行と、機能の語で引いた行(272・198)しか見ていなかった。再実行では、開いた行の Trigger の欄を、ブランチが変えた記録以外のファイル 54 個の basename と、hunk から取った Go の関数名 153 個で引き、当たった行の Trigger と差分を読んだ。Trigger に当たり、origin/main と同じバイト列のまま残る行は次のとおり。

| 行 | Trigger | ブランチがしたこと | 所見 |
| --- | --- | --- | --- |
| 366 | the next failure of that test, or the next change to `org_watch_test.go` | S4b が `internal/cli/org_watch_test.go` を変えた(+54/-43)。race は `:1340-1341` に残る | V3 |
| 144 | Next touch to `doctor_codex_writable_root_test.go` for another reason | `:1791`・`:1848`・`:1924` の 3 行を変えた | V4 |
| 235 | the next change to `internal/org/gate.go`, `gate_run.go`, or `lockfile.go` | `gate.go`(`GateParams.Env` の doc)と `gate_run.go`(`gateLocalConfigEnv`)を変えた | V4 |
| 364 | the next change to `internal/org/gate.go` or `gate_run.go` | 同上 | V4 |
| 316 | the next change to `director_start.go` | `DirectorStartParams.AgmsgHome` とその doc を消した(1205 行 → 1201 行。行の「1203 lines」は origin/main の時点ですでにずれていた) | V4 |
| 367 | the next edit of any of these five recipes | `isolated-herdr-server.md` と `director-launch-cmd.md` を 2 つの写しとも直した。`docs/evidence/` の引用の数は 6 と 1 のまま | V4 |
| 177 の (b) | the next edit to `newOrgStartCmd` | help に D7 の文を足した(`internal/cli/org.go:956-959`)。(b) の直し(usage の backtick を外す)は main の 216b2cdd で済んでいて、`org.go:1021` に backtick はない | V4 |
| 170・171 | the next change to `LegacyWorktreeStateDir` | 呼び出しの付け替え 2 行だけ(`gitToplevel()` → `gitroot.Toplevel("")`、`samePath` → `gitroot.SamePath`)。170 行は S5 が名前を直した | I6 |
| 362 | the next change to `Send` or to the leader template tests | leader の template を描画する `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir`(`internal/org/prompts_leader_inbox_test.go:193`)を変えた。行が挙げる 5 ファイルには入らない | I6 |
| 294 | the next change to `isolateGitEnv` | org の `isolateGitEnv` は変わっていない。S1 が同名の写しを `internal/gitroot/gitroot_test.go` に `XDG_CONFIG_HOME` なしで足した | I6 |

掃き出しに当たったが、Trigger が別の関数や出来事を名指していて当たっていない行: 149、150、186、188、192、196、197、203、245、259(`/director` skill の hunk は 229〜293 行で、`受信箱の処理` の節は 492 行から)、271、290、313、361、377。148 行と 385 行はこのブランチが書き換えた。

### 前の所見の扱い

- V1: 開いたまま。Progress の 206 行が、/sync-docs で持ち越しの注を足し PR の Known gaps に書くと決めている。
- V2、I1〜I3: S6 は触らないので、そのまま残る。
- I4: e13b217a が Progress の 204 行に記録したので対応済み。

### Notes(所見にしないもの)

- test report(148d542e)は HEAD 64a20f69 の時点を書く。Failure analysis の「この PR は ... このテストも変えていない」「テストのコードは直していない」と `docs/tech-debt/README.md:286` の参照(いまは HTML コメントで、行は 287 行)は、S6 のあとは古い。/test の再実行で直る見込み。
- コメントの「macOS prints the full path, Linux only `sleep`」は self-review の N6 が扱ったので、判断し直していない。

### Coverage gaps(再実行)

- Go のテスト(負荷の下での S6 のテストを含む)は打っていない。/test の範囲。
- Trigger の掃き出しは basename と関数名の部分一致で引いた。語で書かれた Trigger(節の名前、出来事)は、引けた行だけを読んだので、引けなかった行に当たったものが残っているかもしれない。

## Verdict

pass

- Verified: AC1〜AC9、AC12 は、コードの該当箇所と probe か grep で確かめた。AC5 は `cmp` と `git check-ignore`。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-skill-sync、check-template-purity、secret scan)はすべて通った。
- Partially verified: AC4 の gate 側と AC7 の watch 側はコードとテストの存在まで。AC10 と AC11 は静的に確かめた。AC13 は静的の部分だけ。
- Not verified: Go のテストの結果(/test)、実際の gate、watch の実起動、`ralph upgrade` の実行、本物の herdr と agmsg。
- 所見: LOW 2 件(V1 tech-debt 272 行の Trigger、V2 `config get` のエラーのラベル)、INFO 4 件。どれも merge を止めない。
- Insight event: `docs/insights/events/2026-10-29-local-config-org.jsonl` に verify の行を 1 行足した(`--verdict pass --cycle auto --low 2`)。
- 再実行(6e1f6563)でも pass。static は全体で通り(rc=0)、S6 は Progress に記録した範囲(テストのファイル 1 つの 1 関数と tech-debt の行)に収まり、計画の digest は承認時と同じ。287 行は 5 欄で、register の主な形 `S--S-` と同じ。事実は test report とコミットに合う(I5 の言い回しだけが 4 つのうち 3 つに合う)。ブランチ全体の Trigger の掃き出しで、cycle 1 で見落とした LOW 2 件(V3 366 行、V4 5 行)と INFO 2 件(I5、I6)を足した。I4 は対応済み。開いた所見は LOW 4 件(V1〜V4)と INFO 5 件(I1〜I3、I5、I6)で、どれも merge を止めない。
- 再実行の insight event: 同じ jsonl に verify の行をもう 1 行足した(`--verdict pass --cycle auto --low 4`)。
