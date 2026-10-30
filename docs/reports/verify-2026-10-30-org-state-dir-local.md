# Verify report: org-state-dir-local

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-org-state-dir-local.md`(issue #276 の 2 本目 A2、AC1〜AC11、D1〜D7、Progress の S1〜S7)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base = origin/main c03647f9、HEAD e0ffa2aa、70 ファイル、+3311/-578)。途中に main の取り込み 3700470e があるので、計画の base e3a3fbf7 からではなく origin/main から見た。spec の準拠と静的解析と文書のずれだけを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。
- Evidence: `docs/evidence/verify-2026-10-30-org-state-dir-local.md`(コミットする。静的解析の要約、grep、probe の出力、help の grep、tech-debt の掃き出し)。全出力は `docs/evidence/verify-2026-10-30-org-state-dir-local.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-verify 自身のログは `docs/evidence/verify-2026-10-30-030124.log`(同じく対象外)。
- load average(`sysctl -n vm.loadavg`): static の直前 26.72、直後 53.04。go build の直前 16.74。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `--state-dir` がないとき個人用のファイルの `[org].state_dir` が台帳になる(org の動詞、status、insights、doctor、source `config`)。`--state-dir` が勝つ | 満たす | `internal/org/statedir.go:72-82`(flag → `config.LocalOrgStateDir("")` → git の段)。本番の呼び出しは 16 か所で、15 か所が `resolveOrgLedger`(`internal/cli/org_legacy_ledger.go:74-86`)を通り、doctor は `internal/cli/doctor_merge_guarantee.go:41` で同じ関数を呼ぶ。probe(HEAD から build した binary、一時 repo): `ralph status --json` は `state_dir_source: config` を返した。`--state-dir` を付けると `flag` になった。個人用のファイルが台帳を指していると、doctor は merge guarantee をその台帳に `cached in <W>/ledger/merge-guarantee.json` と書いた。テストは `TestOrgLedger_PersonalFileNamesTheLedgerFromAnyWorktree`、`TestInsights_PersonalFileNamesTheReceiptsLedger`、`TestMergeGuaranteeEnvFromOS_RealCommandsAndTheResolvedStateDir` |
| AC2 cwd の repository の主 worktree の root のファイルを読む。linked worktree・サブディレクトリから同じ値で、`--config` で変わらない | 満たす(`--config` 側はコードとテストの存在で確認) | `internal/config/local.go:180` は `LocalPath(<cwd>/ralph.toml)` を使い、`ResolveOrgStateDir` は `--config` を受け取らない。probe: main、`main/pkg/sub`、linked、`linked/pkg/sub` の 4 か所で同じ `<W>/ledger`、source `config`。linked worktree に置いた別の値の `ralph.local.toml` は読まれない。`--config` はテスト `TestOrgLedger_ConfigFlagDoesNotChooseThePersonalFile` |
| AC3 相対の値は台帳を決めるときにパスを名指すエラー。`~/` は home に展開。Load は値を検査せず、`--state-dir` を渡した `ralph org stop` は動く | 満たす | `internal/config/local.go:194-207`。`Load` に値の検査はない(`internal/config/config.go:731` は `checkLocalOnlyKeys` だけ)。probe: `rel/ledger` で `ralph status`、`ralph org status`、`ralph insights`、`ralph org stop --all` がどれも exit 1 になり、`<W>/main/ralph.local.toml: [org].state_dir "rel/ledger" must be an absolute path or start with ~/` を名指した。`ralph org stop --all --state-dir <W>/flag-ledger` は exit 0 で `no active seats` を出した。`~/ledger-home` は `<W>/home/ledger-home` に展開され、`ralph config get org.state_dir` は `~/ledger-home` を書いたとおりに返した |
| AC4 `ralph.toml` の `[org].state_dir` で Load がエラー、パスと `ralph.local.toml` を言う。テンプレートは注釈だけ | 満たす | `internal/config/local.go:118-141`、`config.go:731-733`(エラーのときは `Default()` を返す)。probe: `ralph config get org.max_seats` は exit 3 で `ralph.toml: [org].state_dir can be set only in ralph.local.toml, ... set it in ralph.local.toml instead` を出した。`RALPH_LOCAL_CONFIG=off` のときと、値が空のときも同じだった。`templates/base/ralph.toml:69-75` は注釈で、有効な行はない(`git grep -n '^[^#]*state_dir' -- templates/base/ralph.toml ralph.toml` は 0 件)。テンプレートの Load は `TestLoad_TemplateRalphToml`(/test の範囲) |
| AC5 読めない・正しくない個人用のファイルで、org の動詞・status・insights が exit 1 でパスを名指す。`ralph.toml` が壊れていても台帳は決まる。doctor は fail の 1 行で続けてキャッシュを書かない。TOML でないファイルは `--state-dir` があっても Load が失敗する | 満たす | probe: TOML でないファイルで `ralph org status` が exit 1(`cannot resolve the org ledger: <path>: toml: expected character ]`)。`--state-dir` を付けると `org: load config: <path>: ...` で exit 1。`ralph.local.toml` がディレクトリのときは `read <path>: is a directory` で exit 1。壊れた `ralph.toml` と正しい個人用のファイルの組では、`ralph status --json` が exit 0 で source `config` を返した。doctor は `✗ Org ledger: fail — cannot resolve the org ledger: <path>: ...` の次に `Merge guarantee: ... not cached: the org ledger could not be resolved` を出して続けた(`doctor_merge_guarantee.go:70-81`)。insights は Load を通らないので、`--state-dir` を付ければ動く(probe で exit 0) |
| AC6 本番は `RALPH_ORG_STATE_DIR` を値として読まない。注意・拒否・doctor の warn | 満たす(start と spawn はコードとテストで確認) | env を読むのは `org_legacy_ledger.go:107`(`checkRetiredOrgStateDirEnv`)と `doctor.go:901`(`checkRetiredEnv`)だけ。計画の S4 の grep は 0 件、`EnvOrgStateDir` は計画の本文にしか残っていない。`scripts/`、`tests/`、`.codex/`、`cmd/`、`.claude/rules/` にも出現はない。probe: env が別の場所を指すと、`ralph status`、`org inbox`、`org merge-order`、`org director status` は注意を出して既定の台帳を読んだ。`org stop --all`、`org disband --all`、`org send`(TASK)は exit 1 で断られ、env の場所も既定の台帳も作られなかった。`--state-dir` を渡したとき、env が既定の台帳と同じとき、個人用のファイルが台帳を指すときは注意だけで動いた。doctor は `⚠ RALPH_ORG_STATE_DIR: warn` を出した。start と spawn の拒否は、`resolveOrgLedger` が `newOrgSpawnRuntimeAt` と見張りの配線より前にあること(`org.go:814-818`、`:1013-1021`、`:1114-1122`)と、`TestOrgRetiredStateDirEnv_MutatingVerbsRefuseAnotherLedger`(`refuseOrgWatchLaunch`、herdr と agmsg の呼び出しの記録)で確かめた。detach した見張りを残さないよう、probe では打っていない。doctor の warn の文には V3 がある |
| AC7 insights と doctor の `--state-dir`。insights は `--receipts` が勝つ。古い台帳の案内は常に `--state-dir` | 満たす | `internal/cli/insights.go:50`・`:61-62`、`doctor.go:54`。probe: `ralph insights --state-dir <W>/ins-ledger` はその台帳の receipts を読んだ(`ORG probe  SEAT lead`)。`--receipts` も渡すとそちらを読んだ(`ORG other  SEAT rev`)。`ralph doctor --state-dir <W>/doc-ledger` は `cached in <W>/doc-ledger/merge-guarantee.json` を出した。案内は `org_legacy_ledger.go:172-174` で `pass --state-dir` だけになり、origin/main にあった insights 向けの `set RALPH_ORG_STATE_DIR=` の分岐は消えた |
| AC8 source `config` を `env` と同じに扱う。子には `--state-dir` を明示で渡す | 満たす | `statedir.go:127` は `case "flag", "config"`(origin/main の `:110` は `"flag", "env"`)。`FeatureRepoRoot`(`:146-159`)は `git-main-worktree` 以外の source を同じ道で扱う。`loadMainWorktreeOrgConfig` は `LedgerMainWorktreeRoot` を通る(`org.go:332`)。hint は flag のときだけ付く(`org_inbox.go:350-355`)。本番の Go で source `"env"` を比べる箇所は 0 件。見張りの子は `orgWatchChildArgs` が常に `--state-dir` を付ける(`org_watch.go:358`)。director は `director_cmd.go:323`、leader はタスクの `- 台帳:` の行(`feature.go:427`) |
| AC9 `RALPH_LOCAL_CONFIG=off` で個人用のファイルの段を飛ばす | 満たす | `local.go:177`(git も打たない)。gate と regate は `gate_run.go:53`・`:75-83` で `RALPH_LOCAL_CONFIG=off` を付けて scripts を走らせる。cli・org・config の TestMain も `off` に固定している。probe: 個人用のファイルがあっても `RALPH_LOCAL_CONFIG=off` では source `git-main-worktree` になった |
| AC10 README、`/org` skill の 4 つの写し、recipe 2 本の 2 つの写し、テンプレートの `ralph.toml`、leader の雛形、help が `--state-dir` と `[org].state_dir` で説明する。env は「もう読まない」だけ | 満たす(tech-debt と付随の文は /sync-docs) | `git grep RALPH_ORG_STATE_DIR` の出現(`internal/`、`docs/reports`、`docs/plans`、`docs/specs`、`docs/insights`、`docs/evidence` を除く)は `README.md:148`、skill の 4 つの写しの `:115`、recipe の 4 ファイルの 1 行ずつで、どれも「no longer reads / もう読まない」の文。残りは `docs/tech-debt/README.md` の 6 行(下の文書のずれ)。skill の 4 つの写しは md5 が一致し、recipe はルートとテンプレートが `cmp` で一致。help は 35 のコマンドを打ち、env に触れるのは escalate の 1 文(`org_inbox.go:83`、「ralph no longer reads」)だけ。`--state-dir` の usage は 4 種類で、どれも `[org].state_dir in ... ralph.local.toml` を既定の最初の段に挙げる。leader の雛形は `:122-130` と `:735` |
| AC11 build、gofmt、vet、golangci-lint、check-sync、check-skill-sync、run-verify | 一部(静的の部分は満たす) | `./scripts/run-static-verify.sh` は rc=0(175 秒)。`go build -o <scratch>/ralph ./cmd/ralph` も rc=0。run-verify の既定モード(テストを含む)は /test の範囲で、ここでは打っていない |

計画の Verify plan の「16 か所の呼び出しがすべて helper を通ること」は、15 か所が helper を通り、doctor は `ResolveOrgStateDir` を直接呼ぶ。doctor は fail の行で続け(D4)、env には warn を出す(D5)ので、cobra の `cmd` と legacy の guard を前提にする helper を通らない。この扱いは Progress の S2・S3 の記録にある(I4)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0、175 秒、12:01:23〜12:04:18 JST) | 範囲は `Language scope: full fallback (unclassified:templates/base/ralph.toml)` で、changed の既定でも Go は全体で走った。gofmt `ok`、go vet と staticcheck は無出力(Skipping の行なし)、golangci-lint `0 issues.`。local verifier(shellcheck、`sh -n`、awk の構文、`jq -e`、Codex の hook の検査)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照もすべて OK。secret scan は `c03647f9..e0ffa2aa` で clean。timeout には当たらず、打ち直しもしていない |
| `go build -o <scratch>/ralph ./cmd/ralph` | pass(rc=0) | probe 用。`go version` は go1.27.3、`go.mod` も 1.27.3。build の前後で `git status --porcelain` は空 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-org-state-dir-local.md` | `1ea27c42968f` | `- Approved:` の digest と同じ |
| `git diff --check origin/main...HEAD`、変えたファイルの U+FFFD の grep | どちらも空 | |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README(`ralph.local.toml` の節 `:148`、doctor `:134`、insights `:136`、cleanup の段落) | はい(1 点は /sync-docs) | `:148` が置き場所、書けるファイル、値の形、エラーと doctor の扱い、`--state-dir` での回避、env の扱いを書く。`ralph config get` の行(`:135`)の exit 3 は local-only のキーを名指さない(/sync-docs、lead の指示どおり) |
| `/org` skill の 4 つの写し、`/director` skill の 4 つの写し | はい | 「前提」の段の並び(`:89-93`)と新しい項(`:107-120`)。director の 2 か所(`:899-901`、`:1012-1014`)。check-skill-sync と check-sync が通る |
| recipe 2 本(`worktrees.md:54-61`、`codex-seat-permissions.md:84-89`)と写し | はい | |
| `templates/base/ralph.toml`、leader の雛形、`feature.go` の注釈 | はい | |
| help(org の persistent flag、status、insights、doctor、escalate、cleanup、merge) | はい(V3 あり) | `ralph config get --help` の exit 3 は local-only のキーを名指さない(/sync-docs)。`ralph doctor --help` の `--strict ralph upgrade` は前からの問題(/sync-docs) |
| `docs/tech-debt/README.md` 148・170・192 行 | いいえ(/sync-docs に回す、lead の指示どおり) | env を台帳を選ぶ手として書く |
| `docs/tech-debt/README.md` 198 行の (h) と 188 行の閉じた (d) | いいえ(I1、/sync-docs) | (h) は「`--state-dir` か `RALPH_ORG_STATE_DIR` を pane に渡すコードはない」と書き、env を設定した live check を提案している。ralph は env を読まなくなったので、その確かめは要らなくなった。188 行の (d) の閉じた注は、`orgWideLimitsHelp` が今も「named with --state-dir or RALPH_ORG_STATE_DIR」と言うと書くが、この PR で文が変わった(`org.go:712-721`) |
| `docs/tech-debt/README.md` の Trigger に当たった行 | いいえ(V1・V2、/sync-docs) | 下の Findings |
| `docs/specs/2026-10-07-org-multi-org-director.md:135` | 書き換えない(計画の non-goal) | #263 の記録。同じ spec の `:88`・`:144` と `docs/specs/2026-08-01-org-runtime.md` の FR-2 の注も、日付つきの決定として env を書いている。env をやめたことを言う後の箇条はない(I3) |
| `.claude/rules/ralph/model-routing.md:151` | 判断が要る(I2) | `defaults_sync_test.go` の説明が、照合しないキーを `[org.roles]`、`[org.permissions].roles`、`launch_cmd`、`wake_cmd` と挙げ、`state_dir` と新しい `TestTemplateDriftProbesHoldNoLocalOnlyKey` に触れない。計画の Non-goals は model-routing.md の書き直しを B に回している |
| `AGENTS.md:89`(`internal/config/` の行) | 任意(I6) | `local.go` は個人用のファイルを重ねるとだけ書き、`LocalOnlyKeys` と `LocalOrgStateDir` に触れない |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | tech-debt の Trigger にこの branch が当たったのに、持ち越しの注がない行が 14 ある(cycle 1 の版は「13」と書いていた。下の行番号は 14 個で、cycle 2 で直した)。どの行も origin/main と同じバイト列のまま(この branch は `docs/tech-debt/README.md` を変えていない)。148・170 行は lead の一覧にある env の記述とは別に、Trigger にも当たっている。148 は `ResolveOrgStateDir` の変更、170 は `guardLegacyOrgStateDir` の変更(insights の分岐を外した)。ほかに 135(`doctor_codex_writable_root.go` の注釈 1 行)、177 の (a)(`/org` skill の命令の表の `cleanup` の行)、188(`orgWideLimitsHelp` の文。(b)・(c)・(e) が開いたまま)、198 の (j)(`LedgerMainWorktreeRoot` の `case`)、208 の (b)(`newOrgRuntimeAt` の doc を直したが、まだ呼び出し元を 2 つ名指す)、246(`org_merge_order.go` とテスト)、362(leader の雛形のテスト 2 本。`prompts_test.go` は 2683 行で +300 の閾値には届かない)、368(`codex-seat-permissions.md` の 2 つの写し。`docs/evidence/` の引用は 2 のまま)、386(4 ファイルのうち `org_merge_test.go`、`cleanup_test.go`、`director_skill_sync_test.go` の 3 つ)、390(`cleanup.go` の `ledgerMoveStep`。1086 → 1089 行、`org_cleanup_test.go` は 1142 → 1262 行)、391(断りの文の `ledgerMoveStep`)、392(`statedir.go` と呼び出し元の `feature.go`・`cleanup.go`。行は呼び出し元に `cleanup.go` を数えていない) | evidence の「tech-debt Trigger sweep」と、各行の Trigger の欄 | /sync-docs で各行に「carried over by feat/org-state-dir-local」の注を足す。392 は呼び出し元の数も直す |
| V2 | LOW | tech-debt 186 行の (c) F-7 は、この PR で解消したのに開いたまま。F-7 は「`newOrgRuntime` の解決と guard の 2 段が spawn・start・start --plan に写され、`newOrgSpawnRuntimeAt` が guard を繰り返す」という指摘。いまは 15 か所が `resolveOrgLedger` を通り、`newOrgSpawnRuntimeAt` は guard を呼ばない(`org.go:154-160`)。Trigger「the next change to `newOrgRuntime` (F-7: one shared function for the resolve and guard steps)」が求める形そのもの | `docs/tech-debt/README.md:186`、`internal/cli/org_legacy_ledger.go:74-86`、`internal/cli/org.go:134`・`:814`・`:1013`・`:1114` | /sync-docs で (c) を閉じる。Debt・Impact・Why deferred・Trigger の 4 つの欄の F-7 の記述をそろえて直す |
| V3 | LOW | doctor の warn の文が、台帳を変える動詞が断られる条件を言い切れていない。文は「while it names another directory than the ledger ralph finds by default, the org verbs that change the ledger are refused」だが、拒否は `--state-dir` も `[org].state_dir` も台帳を名指さないときだけ起きる(`org_legacy_ledger.go:116`)。probe で、個人用のファイルが `<W>/ledger` を、env が `<W>/env-ledger` を指すとき、doctor は同じ warn を出し、`ralph org stop --all` は注意だけで exit 0 になった。escalate の help の最後の文(`org_inbox.go:83-84`)も同じ条件を省いている。README `:148` と `/org` skill `:116-118` は条件を書いている | `internal/cli/doctor.go:884-887`、evidence の「Probes 3」 | 2 つの文に「and neither --state-dir nor [org].state_dir names the ledger」に当たる条件を足す。文を固定するテストも合わせる。Go の文字列なので /sync-docs では直せない。直すならパイプラインの次の実行で、直さないなら PR の Known gaps に書く |
| I1 | INFO | 上の文書のずれの表の、198 行の (h) と 188 行の閉じた (d) | 同上 | /sync-docs で、148・170・192 行と一緒に直す |
| I2 | INFO | 上の表の model-routing.md | 同上 | B に回すか、/sync-docs で照合しないキーに `state_dir` を足す |
| I3 | INFO | 上の表の spec の日付つきの箇条 | 同上 | 任意。A1 の I3 と同じく、env をやめた日付つきの箇条を /sync-docs が足せる |
| I4 | INFO | 計画の Verify plan の「16 か所がすべて helper を通る」に対し、doctor は直接呼ぶ | Spec compliance の表の下の段落 | 記録済みの逸脱。対応は要らない |
| I5 | INFO | README の `ralph status [--org-id <id>]`(`:132`)は `[--state-dir <dir>]` を書かない。origin/main でも同じで、この PR の前からある | `git show origin/main:README.md` の 132 行 | 任意 |
| I6 | INFO | 上の表の AGENTS.md | 同上 | 任意 |

Trigger の掃き出しで当たったが、当たっていないと判断した行: 136(残る Trigger は `checkShellAliases` の severity の `switch`)、144(`doctor_codex_writable_root_test.go` は変わらず、doctor の対象ディレクトリの決め方も同じ)、150(`TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry` に触れていない)、203、215、236(`signalCauseContext` の channel は変わらない)、258(`org_director.go` の変更は `newOrgDirectorRuntime` で、attach・detach の `RunE` ではない)、259(director の skill の変更は「進捗の報告」の節で、「受信箱の処理」の節ではない)、266 の (b)(`director_skill_sync_test.go` に足したのは既存のテストの中の検査で、新しいテストではない)、353(`sendInboxItemToHuman` と呼び出し元のエラーの行は変わらない)、373 と 380(leader の雛形の変更は機能ごとの org の前文と escalate の節で、手順 3〜14 ではない。373 は前文を「手順」に含めるかで判断が分かれる)。

self-review の L2・L3・L5・L7・L8・L9 は判断し直していない。L8 は上の文書のずれの表の 148・170・192 行と同じもの。

## Observational checks

HEAD から build した binary を、scratchpad の一時ディレクトリの repo(`git init` → 空コミット → `git worktree add`)で打った。どれも `env -i PATH=/usr/bin:/bin HOME=<W>/home` で打ったので、`RALPH_LOCAL_CONFIG`・`RALPH_ORG_STATE_DIR`・`GIT_DIR` は外れている。全出力は evidence にある。

- AC1・AC2: 個人用のファイルを main の root に置き、linked worktree に別の値の同名ファイルを置いた。`ralph status --json` は 4 か所とも `<W>/ledger`、source `config` を返した。
- AC3〜AC5: 上の表の値のとおり。doctor は狭めた PATH に claude と codex がないので fail 2 件が加わり、どの場合も exit 1 になる。この変更とは関係ない。
- AC6: `org stop --all`、`org disband --all`、`org send`(TASK)の拒否のあと、`<W>/env-ledger` も `<W>/main/.harness` も作られていない。
- help: 35 のコマンドの `--help` を打ち、`RALPH_ORG_STATE_DIR` に触れる行は escalate の 1 行だけだった。
- escalate の NOT RECORDED の経路は、macOS の通知を出すので probe では打っていない。テスト(`TestOrgEscalate_LedgerThatCannotBeResolvedStillReachesTheHuman`、`TestOrgRetiredStateDirEnv_EscalateStillReachesTheHuman`、`TestEscalateNotRecorded_WithNoLedgerWritesNothing`)の存在とコード(`org_inbox.go:98-101`、`escalate.go:233-239`・`:547-553`)で確かめた。

## Coverage gaps

- Go のテスト(`go test`)、`./scripts/run-test.sh`、run-verify の既定モードは打っていない(/test の範囲)。AC11 のテストの部分と、上に挙げたテストが通ることはそちらに残る。
- `ralph org start` と `ralph org spawn` の拒否は、コードの順とテストの存在で確かめた。見張りの子を残さないため、実際には打っていない。
- 本物の herdr と agmsg、機能の org の leader の pane での台帳の受け渡しは打っていない(計画の範囲外)。
- Trigger の掃き出しは、ファイルの basename と hunk の関数名、touched な skill や雛形の名前の部分一致で引いた。語で書かれた Trigger(節の名前、出来事)は、引けた行だけを読んだ。
- 足すと効く決定的な検査: V3 の種類(拒否の条件を言う利用者向けの文)は、文全体を固定するテストがあっても条件の漏れを捕まえない。`checkRetiredOrgStateDirEnv` の判定と doctor・help の文を同じ定数や関数から作れば、ずれは起きにくくなる。

## Verdict

pass

- Verified: AC1〜AC10 は、コードの該当箇所と probe か grep で確かめた。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-skill-sync、check-template-purity、secret scan)はすべて通り、計画の digest は承認時と同じ。
- Partially verified: AC2 の `--config` の側、AC6 の start と spawn の拒否、escalate の NOT RECORDED は、コードとテストの存在まで。AC11 は静的の部分だけ。
- Not verified: Go のテストの結果(/test)、本物の herdr と agmsg での通し。
- 所見: LOW 3 件(V1 tech-debt の Trigger 14 行、V2 186 行の F-7 の解消、V3 doctor の warn と escalate の help の条件)、INFO 6 件。どれも merge を止めない。
- /sync-docs に回すもの: tech-debt 148・170・192 行の env の記述(lead の指示)、198 行の (h) と 188 行の (d) の注(I1)、V1 の 14 行の持ち越しの注、V2 の 186 行の (c) を閉じること、README `:135` と `ralph config get --help` の exit 3 が local-only のキーを名指さないこと(lead の指示。help は Go なので /sync-docs では README だけ)、`ralph doctor --help` の `--strict` の値の名前(前からの問題、lead の指示)。任意: I2・I3・I5・I6。
- Insight event: `docs/insights/events/2026-10-30-org-state-dir-local.jsonl` に verify の行を 1 行足した(`--verdict pass --cycle auto --low 3`)。
- S8(cb33a636)まで見ても pass。V3 は直った(下の「S8 の確認」)。S8 で新しい所見はない。開いた所見は LOW 2 件(V1、V2)と INFO 6 件で、どれも merge を止めない。V1・V2 と I1 は /sync-docs が扱う。insight event は同じ jsonl に 1 行(`--verdict pass --cycle auto --low 2`)。
- cycle 2(S9 6415aabd、HEAD a689bac7)まで見ても pass。run-static-verify は rc=0(41 秒)。cross-review の ACTION_REQUIRED #1 が破っていた AC4 は、大文字小文字を混ぜた 5 つの綴りで、`RALPH_LOCAL_CONFIG` を外しても `off` にしても Load が断る。S9 の前の binary は同じ `ralph.toml` で `/x` を返した。ほかの AC は、cycle 1 の probe を HEAD の binary で打ち直して出力が S8 の 1 行のほか変わらないことで確かめた。S9 で新しい所見はない。V1・V2・I1・I5・I6 は /sync-docs(3a6d1b11)で扱われた。開いているのは INFO の I2(B の範囲)、I3(依頼で specs を外した)、I4(記録済みの逸脱)だけ。insight event は同じ jsonl に 1 行(`--verdict pass --cycle auto --low 0`、cycle 2)。下の「cycle 2」の節を見る。

## S8 の確認

- Date: 2026-10-30 12:30〜12:45 JST
- Verifier: verifier サブエージェント(`/verify` の追補。cross-review の前なので cycle は 1 のまま)
- Scope: `git diff 5e65830a..cb33a636`(4 ファイル、+10/−5)。HEAD は b55e877b。d5881800 は計画の Progress checklist に 3 行足しただけ(`@@ -189,0 +190,3 @@`)、b55e877b は self-review report だけを変えた。テストは打っていない(/test の範囲)。run-static-verify の全体は打ち直していない(lead の指示)。
- load average(`sysctl -n vm.loadavg`): go のコマンドの直前 12.95、直後 12.95。
- Evidence: `docs/evidence/verify-2026-10-30-org-state-dir-local.md` の「S8 check」の節。全出力は同じ名前の `.log` の末尾。

### 打ったもの

| Command | Result | Notes |
| --- | --- | --- |
| `go build ./...` | pass(rc=0) | |
| `go build -o <scratch>/ralph-s8 ./cmd/ralph` | pass(rc=0) | probe 用。HEAD b55e877b から build した |
| `go vet ./internal/cli/` | pass(rc=0、無出力) | |
| `golangci-lint run ./internal/cli/` | pass(`0 issues.`) | |
| `gofmt -l`(S8 の 4 ファイル) | 空 | |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-org-state-dir-local.md` | `1ea27c42968f` | 承認時と同じ。d5881800 の変更は Progress checklist の中だけ |
| `git diff --name-only <c>^ <c>`(3 コミット) | lead の説明どおり | cb33a636 は上の 4 ファイル、d5881800 は計画、b55e877b は self-review report |

どれも timeout に当たらず、打ち直しもしていない。

### V3 が直ったか

直った。

- コード: doctor の warn の `effect`(`internal/cli/doctor.go:885-886`)と escalate の help の最後の文(`internal/cli/org_inbox.go:82-85`)に「neither --state-dir nor [org].state_dir names the ledger」が入った。拒否の判定(`internal/cli/org_legacy_ledger.go:116`、source が flag か config なら先に注意で返る)と、文の条件がそろった。README `:148` と同じ言い方になった。
- probe(前回と同じ設定。個人用のファイルが `<W>/ledger` を、`RALPH_ORG_STATE_DIR` が `<W>/env-ledger` を指す): S8 の binary の doctor は、条件付きの warn を出した。同じ設定で `ralph org stop --all` は注意だけで exit 0 になり、文と動きが合う。個人用のファイルを外すと、同じ env で `ralph org stop --all` は exit 1 で断られる。`ralph org escalate --help` の最後の文も条件付きになった。
- テスト: `TestCheckRetiredEnv` の期待値は Detail の全文一致のまま新しい文に変わった。`TestRunDoctorOpts_RetiredOrgStateDirEnv` と `TestOrgEscalateAndInboxHelp_DescribeTheDirectorSwitch` の断片も新しい条件を含む。検査は弱まっていない。テストが通ることは /test で確かめる。

### 文書のずれ

新しいずれはない。

- 古い文(`while it names another directory`、`and when RALPH_ORG_STATE_DIR, which ralph`)は、`docs/reports/` と `docs/evidence/` の記録を除いて 0 件(`git grep`)。
- 拒否の条件を書くほかの面のうち、README `:148` と `/org` skill `:116-118` はもとから条件を書いている。README の doctor の行(`:134`)と recipe は env の warn に触れない。読むだけの動詞の注意(`org_legacy_ledger.go:128-130`)は既定の段で決まったときにしか出ないので、条件を足さなくても正しい。
- tech-debt の Trigger を S8 の 4 ファイルと関数(`retiredEnvs`、`newOrgEscalateCmd`、3 本のテストの名前)で引いた。当たった 127・144・203 行は、どれも別の関数か閉じた項目を名指しており、当たっていない。`docs/tech-debt/README.md` は S8 でも変わっていない。

### 前の所見の扱い

- V3: fixed(cb33a636)。
- V1、V2、I1〜I6: S8 は触らないので、そのまま残す。V1・V2・I1 は /sync-docs が扱う。

## cycle 2

- Date: 2026-10-30 14:20〜14:50 JST
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 2。`cycle-count.json` は 2)
- Scope: e5d5a5a4..a689bac7 の 8 コミット。コードの変更は S9 6415aabd だけで、`internal/config/local.go`(`checkLocalOnlyKeys` の doc と `tomlDocHasKey`)と `local_test.go`。残りの 7 コミットは記録と文書(test report、/sync-docs の 3a6d1b11 と 5602c49d、計画の 84930e3e と 845b4eff、cross-review の triage の ca7c651f、self-review の a689bac7)。static はブランチ全体で打った。テストは打っていない(/test の範囲)。
- 直す元の所見: `docs/reports/cross-review-triage-org-state-dir-local.md` の ACTION_REQUIRED #1。`ralph.toml` の `[ORG]` / `STATE_DIR` が local-only の検査をすり抜け、AC4 が破れていた。
- load average(`sysctl -n vm.loadavg`): static の直前 9.50、直後 17.58。build の直前 16.38。
- Evidence: `docs/evidence/verify-2026-10-30-org-state-dir-local.md` の「cycle 2」の節。全出力は同じ名前の `.log` の末尾。run-verify 自身のログは `docs/evidence/verify-2026-10-30-052232.log`。

### 打ったもの

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0、41 秒、14:22:31〜14:23:13 JST) | 範囲は `Language scope: full fallback (unclassified:templates/base/ralph.toml)` で、Go は全体で走った。gofmt `ok`、go vet と staticcheck は無出力(Skipping の行なし)、golangci-lint `0 issues.`。check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照、local verifier もすべて OK。secret scan は `c03647f9..a689bac7` で clean。timeout には当たらず、打ち直しもしていない |
| `go build -o <scratch>/ralph-c2 ./cmd/ralph` | pass(rc=0) | probe 用。HEAD a689bac7 から build した。build の前後で `git status --porcelain` は空 |
| `git archive ca7c651f` を scratch に展開して `go build` | pass(rc=0) | S9 の直前の binary。比べるためだけに使い、展開した木は消した |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-org-state-dir-local.md` | `1ea27c42968f` | 承認時と同じ。この間の計画の変更は AC の checkbox(84930e3e)と Progress の記録(3a6d1b11、845b4eff)だけ |

### AC4(大文字小文字を混ぜた綴り)

満たす。

- コード: `tomlDocHasKey`(`internal/config/local.go:153`)は、区切りごとに `strings.ToLower` が同じキーをすべて辿る。go-toml v2.3.0(`go.mod:8`)の `structFieldPath`(module cache の `unmarshaler.go:1386-1414`)は、field の名前とその `strings.ToLower` を表に入れ、キーをそのまま、なければ `strings.ToLower(キー)` で引く。local-only のキーの各区切りは小文字の tag と同じなので、2 つの規則は同じキーを拾う。`checkLocalOnlyKeys`(`:129`)が失敗すると、Load は前と同じく `Default()` を返す。
- probe(HEAD の binary、一時 repo の main の root に `ralph.toml` だけを置いた): `[ORG]` / `STATE_DIR = "/x"`、`[Org]` / `State_Dir`、`ORG.STATE_DIR`、`ORG = { STATE_DIR = ... }`、`[org]` のあとの 2 つ目の table `[ORG]` の 5 つの綴りで、結果は次のとおり。
  - `ralph config get org.state_dir` は、`RALPH_LOCAL_CONFIG` を外しても `off` にしても exit 3 になった。文は `config get: ralph.toml: cannot be read or is invalid: ralph.toml: [org].state_dir can be set only in ralph.local.toml, ... set it in ralph.local.toml instead` で、ファイルのパスと書く先の `ralph.local.toml` を名指す。
  - `ralph org status --org-id a` は `org: load config: ralph.toml: ...` で exit 1 になった。doctor は `⚠ ralph.toml: warn — parse error: <同じ文> — using defaults` を出した。
  - `ralph status` は、どちらの設定でも exit 0 のまま、stderr に `warning: status: load ralph.toml: ralph.toml: [org].state_dir can be set only in ralph.local.toml, ...` を出し、既定の値で続けた。読めない `ralph.toml` で status を止めない #274 の振る舞いで、文はパスと `ralph.local.toml` を名指している。依頼の「`ralph status` がエラーになる」とは形が違うが、AC4 が求めるのは Load のエラーなので、AC4 の外れではない。
- 対照: `[ORGANIZATION]` / `STATE_DIR` と `[org]` / `state_dirs` は断られない。`config get org.state_dir` は空で exit 0、`org status` も exit 0 になった。
- S9 の前(ca7c651f の binary): 同じ `[ORG]` / `STATE_DIR = "/x"` で、`ralph config get org.state_dir` は `RALPH_LOCAL_CONFIG` のどちらの設定でも `/x` を返して exit 0 になった。cross-review の所見のとおりの穴で、HEAD では exit 3 になる。
- 個人用のファイルの側: `ralph.local.toml` に `[ORG]` / `STATE_DIR = "<W>/ledger"` と書くと、`ralph status --json` は `state_dir` が `<W>/ledger`、source `config` を返した。`LocalOrgStateDir` は struct に読むので、もとから decoder の規則で引いている。
- テスト: `TestLoad_LocalOnlyKeyInRalphToml`(`local_test.go:585`)の 7 行と、`TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder`(`:729`)。後者は各行を本物の `toml.Unmarshal` で `Config` に読み、その結果を期待値にする。通ることは /test の範囲。

### ほかの AC が崩れていないか

崩れていない。cycle 1 の probe(`Probes 1`)を HEAD の binary で同じ手順で打ち直し、出力を cycle 1 のものと `diff` した。違うのは doctor の `RALPH_ORG_STATE_DIR` の warn の 1 行だけで、これは S8 で変えた文。AC1〜AC7 の probe の出力(AC4 の小文字の `[org].state_dir` の 3 通りを含む)は同じだった。S9 が変えたのは `checkLocalOnlyKeys` と `tomlDocHasKey` だけで、`LocalOrgStateDir`、`ResolveOrgStateDir`、cli には触れていない。

### 文書のずれ

S9 で新しいずれは出ていない。

- README `:135`(`config get` の行、/sync-docs が 3a6d1b11 で直した)と `:148` は「`ralph.toml` に `[org].state_dir` があると読み込みが失敗する」と書く。大文字小文字には触れないが、どの綴りでも断る今の動きと食い違う文はない。`git grep` で、README、AGENTS.md、skill、recipe、テンプレートの `ralph.toml`、tech-debt に、local-only のキーの照合を大文字小文字を区別すると書いた文はない。
- エラーは、文書の綴り(`STATE_DIR`)ではなく `[org].state_dir` と名指す。doc comment(`local.go:122-128`)とテストがそう固定している(self-review の C2-N1)。
- 計画の Progress(845b4eff)の S9 の記録は、コードの `strings.ToLower` と、`EqualFold` を使わない理由(U+0130、U+017F)に合う。
- tech-debt の Trigger を S9 の名前(`local.go`、`local_test`、`checkLocalOnlyKeys`、`tomlDocHasKey`、`LocalOnlyKeys`、2 本のテストの名前)で引いた。当たった 78・134 行は閉じた行。199・259・294 行は別の関数を名指している。396 行の (b)「`config.Load` のエラーの道の次の変更」は当たっていないと判断した。S9 は、どの文書が `checkLocalOnlyKeys` のエラーになるかを変えたが、エラーのあとの扱い(`Default()` を返す)は変えていない。396 行の (a) が名指す 4 つの関数も変わっていない。
- self-review cycle 2 の C2-L1(L2 と L5 が register にない)は、lead の指示どおり /sync-docs に回す。判断し直していない。

### cycle 1 の所見の扱い

| ID | いまの状態 | 根拠 |
| --- | --- | --- |
| V1 | /sync-docs で対応済み(3a6d1b11) | 14 行すべてに `org-state-dir-local` の注がある(`sed -n <行>p` で数えた: 135、148、170、177、188、198、208、246、362、368、386、390、391、392)。cycle 1 の V1 は「13 行」と書いていたが、挙げた番号は 14 個で、14 が正しい。sync-docs の報告がこの数え違いを指摘したので、この版で上の Findings と Verdict の数を 14 に直した |
| V2 | /sync-docs で対応済み(3a6d1b11) | 186 行の (c) を `~~` で取り消し、4 つの欄に `Closed in feat/org-state-dir-local` がある |
| V3 | fixed(cb33a636) | 上の「S8 の確認」 |
| I1 | /sync-docs で対応済み | 198 行の (h) は `Moot since feat/org-state-dir-local`、188 行の閉じた (d) に今の文の注。148・170・192 行の env の記述は「もう読まない」の注になった |
| I2 | 開いたまま(B の範囲) | `model-routing.md:151`。sync-docs の報告は PR の Known gaps に入れる |
| I3 | 開いたまま | 依頼で `docs/specs/` を外した(sync-docs の報告の「直さなかったもの」) |
| I4 | 記録済みの逸脱 | 対応は要らない |
| I5 | /sync-docs で対応済み | README `:132` の見出しが `ralph status [--org-id <id>] [--state-dir <dir>]` になった |
| I6 | /sync-docs で対応済み | `AGENTS.md:89` に `LocalOnlyKeys` と `LocalOrgStateDir` |

### Coverage gaps(cycle 2)

- Go のテスト(S9 の 2 本を含む)と run-verify の既定モードは打っていない(/test の範囲)。
- go-toml の規則は v2.3.0 の source を読んで確かめた。版が上がったときの追従は、`TestCheckLocalOnlyKeys_LetterCaseFollowsTheDecoder` が decoder と突き合わせることに頼る。
- 「一致したキーをすべて辿る」規則は、2 つ目の table に置いた probe 1 件で見た。map の順は決まらないので、1 回の probe では「最初の一致だけ辿る」変異を必ず落とせるとは言えない(self-review の C2-N2)。
