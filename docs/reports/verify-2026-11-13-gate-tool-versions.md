# Verify report: gate-tool-versions

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-gate-tool-versions.md`(issue #311、承認の digest `df776faef33e`。HEAD で計算し直しても同じ)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン cycle 1
- Scope: `git diff origin/main...HEAD`(HEAD `a6657d48`、base は origin/main の `4220ea7c`。merge-base も同じなので、#323 と #324 の取り込みは差分に入らない)。self-review の 2 回目(`5b84df9a`、HEAD `3d89cfe7` を見た)のあとの修正 `9c19e5ba` と記録 `a6657d48` も含む。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-13-gate-tool-versions.log`(gitignore 済み。run-static-verify の全出力、tech-debt の Trigger の照合、doctor の観察、pnpm の書き込みの跡)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 log の `# tool` の行、shim の印、`# mise current:`、見出しの 1 行(issue の受け入れ条件 1) | 満たす | `gateToolsLogLines`(`internal/org/gate_tools.go:141-194`)が `gateToolsExplainLine`(「a child a script starts through a shim ... can see other versions」)、場所とツールごとの `# tool <tool> (<dir>): <版> (<path>)`、`missing`、`unknown (<理由>)`、`(mise shim)`・`(asdf shim)`、`# mise current:` を書く。`runGateScripts` が `# load at start:` の直後、最初のスクリプトの前に呼ぶ(`gate_run.go:214`)。local の regate も同じ `runGateScripts` を通る。テスト `TestGate_RecordsTheToolVersionsWithTheRealProbe` が見出しの並びを全文で固定している。issue の「走った版」との差は I6 |
| AC2 記録の `tool_*` と `tool_dirs`、読み戻し、`unknown`・`mixed`、前の形 | 満たす | `gateRecord.toolDetails` は `log=` の前に書き、1 トークンでない値を `unknown` にする(`gate_record.go`)。`gateRecordFromDetails` は知らない鍵を読み飛ばし、`tool_dirs` が数でなければ 0。`TestGateRecordDetails_Tools` が往復、空白入りの値、`tool_*` の無い前の形、`tool_dirs=two` を確かめる。`mixed` は `gateToolsSummary` |
| AC3 測る場所の一覧(4 つの構成、上限、python3 と go、ls-files の失敗) | 満たす | `LayoutFromFiles`・`placeManager`(`internal/toolchain/projects.go`)。`TestLayoutFromFiles_FindsPlacesAndTheirPackageManager` に「root only」「no root markers, apps/a on pnpm and apps/b on yarn」「version files only in subdirectories」「pnpm workspace」の行がある。上限は `TestLayoutFromFiles_CapsPlacesAndCountsTheRest` と log の `# tools:` の行。ls-files の失敗は `RootLayout` と log の行(`TestMeasureGateTools_WithoutTrackedFiles`)。計画助言の MEDIUM は D2・D8 と AC3 に入っている |
| AC4 1 回の sh、15 秒、時間切れで verdict が変わらない、stdout を握る子、ctx の終わり | 満たす | `gateToolsTimeout = 15s`、`toolchain.Run` が Setpgid、group への SIGTERM と `KillGrace` のあとの SIGKILL、`WaitDelay`。`TestGate_ProbePastItsBudgetLeavesTheToolsUnknown`、`TestProbe_ShimChildHoldingTheOutputDoesNotStopTheRest`、`TestRun_ChildHoldingStdoutAfterTheShellExitsDoesNotHang`、`TestGate_InterruptDuringTheProbeRecordsNothing` |
| AC5 `TOOLS:` の行、reviewer の prompt、`/org` と director の skill(4 写し)、固定のテスト | 満たす | `printGateTools` を gate の pass と fail、regate の `printRegateLines` が `LOAD:` のあとに呼ぶ。reviewer の prompt は BLOCKED の本文と例に `TOOLS:` を足した。4 写しは `cmp` で同じ。固定は `prompts_test.go`、`org_skill_send_sync_test.go`、`director_skill_sync_test.go`、CLI の `TestOrgGate_PrintsTheToolsLine` |
| AC6 doctor の「Node toolchain」(issue の受け入れ条件 2) | 満たす | `checkNodeToolchain`(`internal/cli/doctor_node.go`)。JS の場所が無ければ shell を打たずに info、合わなければ warn(直し方に `MISE_NODE_VERSION` と corepack)、合えば pass、別名と読めない範囲は info、`$SHELL` が空か sh の仲間でなければ `/bin/sh` と書く。HEAD から作った binary を、scratch の JS の repo で実際の `/bin/zsh -lc` に対して打ち、warn 3 行と info 1 行、終了コード 0 を観察した(Observational checks)。中断で終了コード 1 になる点は I1 |
| AC7 npm の範囲の表のテスト | 満たす | `TestRangeAllows_NpmRangeForms` に完全な版、部分、`^`、`~`、比較演算子、AND、`\|\|`、ハイフン、`x` の行があり、読めない形は `TestParseRange_RefusesFormsOutsideTheSubset`。見本の 20 行ほどを npm の semver の決まりと照らし、食い違いは無かった |
| AC8 手順書の mise・asdf の節 | 満たす | `docs/recipes/isolated-herdr-server.md` の `### Version managers (mise, asdf)` が `### Shell aliases` の直後にある。shim が子の PATH を書き換えること、`MISE_<TOOL>_VERSION`、`mise current` と `# tool` の行での確かめ方、#308 の見込みの原因(未確認と明記)を書く。templates/base の写しと `cmp` で同じ |
| AC9 help、AGENTS.md の Repo map、tech-debt、2 つの同期の検査 | 満たす | gate と regate の help は 15 秒、ls-files の 10 秒、10 か所、`TOOLS:`、`tool_*`、stderr の行の書き出しまでコードと合う。`ralph doctor --help` には Check の一覧が無く、食い違う文も無い。AGENTS.md に `internal/toolchain/` の行と internal/org の行の追記がある。tech-debt に 4 行。`check-skill-sync.sh` と `check-sync.sh` は run-static-verify の中で通った。help に無い文書のずれは V2〜V4 |
| AC10 測ることで何も変えない | 書かれた条件は満たす。見出しの言い切りは pnpm で崩れる(V1) | 7 つの変数は `probeEnvOverrides` にあり、呼ぶ側の値を消してから最後に置く(`TestProbeEnv_SetsTheSevenVariablesLastOverCallerValues`、`TestDoctorNodeToolchain_ProbeEnvForcesTheSevenVariables`)。偽の shim の試験は `TestGate_RecordsTheToolVersionsWithTheRealProbe`(変数が効いて何も書かない、スクリプトは呼ぶ側の値のまま)と `TestGate_ProbeThatWritesInTheWorktreeIsReported`(効かない shim で log と stderr に出る)。計画助言の HIGH は D7・AC10・R7 に入っている。ただし pnpm 自身の版の切り替えは 7 つの変数で止まらず、doctor の観察で worktree の外に pnpm@1.0.0 が入った |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | 通過(rc=0) | shellcheck、hook の `sh -n` と awk、`jq -e`、Codex の 3 つの guard、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、golang(gofmt ok、`go vet`、golangci-lint 0 issues、staticcheck)、branch の secret scan(`4220ea7c..a6657d48`、clean) |
| `git diff --check origin/main...HEAD` | rc=0 | |
| `gofmt -l internal/toolchain internal/org internal/cli` | 空 | |
| `./scripts/plan-visual.sh digest <plan>` | `df776faef33e` | 計画の `- Approved:` の行と同じ |
| `cmp` で skill の 4 写し(org、director)と手順書の 2 写し | 同じ | |
| origin の先頭(`git ls-remote`) | main `4220ea7c`、ブランチ `a6657d48` | 取り込み後に main は動いていない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org gate --help`、`ralph org regate --help` | 合う。ただし V1 | 「a version that is not installed shows as unknown」(`internal/cli/org_gate.go:113-114`)は pnpm で崩れる |
| `ralph doctor --help` | 合う | Check の一覧が無い。中断で 1 を返すことは書いていない(I1) |
| reviewer の役割 prompt | 合う | `TOOLS:` を写す、無ければ書かない |
| `/org` skill と director skill(4 写し) | 合う。ただし V1 | org skill の gate の行の「入っていない版は `unknown` になる」は pnpm で崩れる |
| 手順書(2 写し) | 合う。ただし V1 | 349-351 行の「would have to be downloaded, therefore shows as `unknown`」は pnpm で崩れる。I3 も |
| `internal/toolchain/doc.go` | V1 | 24-26 行の「So the tools ralph knows install and write nothing」 |
| AGENTS.md の Repo map | 合う | |
| `docs/tech-debt/README.md` | V1、V5、V6 | 新しい 4 行の中身はコードと合う(487 行の Impact の「nothing seen so far does」は V1 で古くなった) |
| README.md | V2 | |
| `docs/specs/2026-10-07-org-multi-org-director.md` | V3 | |
| doctor の Check の並び | 合う | `doctor.go` の注釈で Check 3、3b(#324)、4 … 13、14 の順。Check 14 は最後で、`runDoctorStateDir` の結果の表示の前 |
| `internal/cli/doctor.go` の注釈 | V4 | |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | MEDIUM | pnpm 自身の版の管理は、probe の 7 つの変数で止まらない。`package.json` の `packageManager` が手元の pnpm と違う版を指す場所で `pnpm -v` を打つと、pnpm がその版を取ってきて worktree の外に入れる。HEAD から作った `ralph doctor` を、`packageManager: pnpm@1.0.0` の scratch の repo で打ったところ(手元の pnpm は mise の shim の 12.9.1)、`~/.local/share/pnpm/package-manager-store/v11/` に pnpm@1.0.0 が入り、`global/v11/pnpm-lock.yaml` が書かれた。#311 の構成(`packageManager: pnpm@10.10.0`、mise の `pnpm = "latest"`)でも同じことが起きると見込む(未確認)。gate では scripts の `pnpm install` も同じ取得をするので影響は小さい。読むだけの検査の doctor がネットワークから取って書くのは、計画の Objective の「測ることで環境を変えない」に反する。言い切りの文も崩れる: `internal/toolchain/doc.go:24-26`、手順書 349-351 行(2 写し)、gate の help(`org_gate.go:113-114`)、org skill の gate の行(4 写し)、tech-debt 487 行の Impact。root の pnpm は切り替わった 1.0.0 と測られて pass になったと見込む(warn があると pass の行は出ないので、未確認) | evidence の「side effect after that doctor run」(`projects/057f…` が `../tmp/pnpm-engine-1.0.0-<pid>-…` を指し、`links/@/pnpm/1.0.0` と lockfile の `packageManagerDependencies: pnpm 1.0.0` の時刻が doctor を打った 17:51Z)。7 つの変数は mise・corepack・npm・go のもので、pnpm の設定は無い(`internal/toolchain/probe.go:22-30`) | pnpm の切り替えを止める設定を `probeEnvOverrides` に足してテストで固定し、文書の言い切りを合わせる。設定の名前は手元の pnpm 12.9.1 で確かめること: `pnpm config get manage-package-manager-versions` は既定でも `npm_config_…=false` でも `undefined` を返したので、名前が変わっている見込み(未確認)。足せない場合は、言い切りの文を弱め、487 行に pnpm の場合を書いて Impact を上げる |
| V2 | LOW | README.md の gate の段落(288 行)は `LOAD:` の行と記録の `load_*` の鍵を挙げるが、`TOOLS:` と `tool_*` を挙げない。`LOAD:` か `load_start` に触れる文書のうち、ツールの版に触れないのは README.md だけ | `git grep -l -e 'load_start' -e 'LOAD:'` と `git grep -l -e 'TOOLS:' -e 'tool_dirs'` の差 | /sync-docs で、同じ段落にツールの版の 1 文と、doctor の「Node toolchain」の 1 文を足す |
| V3 | LOW | multi-org の仕様の FR-7 は、gate の振る舞いを変えるたびに日付の注を足してきた(2026-11-12 の #309 の注は log と記録に load average を残すと書く)。#311 のツールの版の注が無い | `docs/specs/2026-10-07-org-multi-org-director.md:139` | /sync-docs で 2026-11-13 の注(issue #311、計画へのパス)を足す |
| V4 | LOW | `runDoctorOpts` の注釈は、subprocess を起こす Check を Check 1・2 の `--version` と Check 11c(と opt-in の Check 12)だけと書く。Check 14 は `git ls-files`(10 秒まで)と `$SHELL -lc`(20 秒まで)を起こす | `internal/cli/doctor.go:76-84` | 注釈に Check 14 を足す(コードの注釈なので、扱いは lead が決める) |
| V5 | LOW | tech-debt の Trigger の欄(4 列目)を、変えたファイルと関数の名前で照らすと、このブランチが引いて、持ち越しの記録が無い行がある(行番号は HEAD `a6657d48`)。238(`gate.go`・`gate_run.go`。その (b) は、`gateInterrupted` がスクリプトのあとの割り込みでも「the scripts were stopped」と言う件で、このブランチは `gateInterrupted` を `gateRunInterrupted` に書き直したが、`gateCheckAfter` と記録の割り込みは今も `true` を渡してその文を出す。`gate.go:233`、`:242`)。293(`regate_test.go`、`regate_merge_test.go`)。294(記録の鍵が増えた。`merged=` は足していない)。367(`gate.go`・`gate_run.go`)。372(reviewer の役割 prompt)。398 (a)(`Regate`、`runGateScripts`)。405(`regate.go`、`org_regate_test.go`)。415(`doctor.go` 1624 → 1645 行)。180 (a)(`/org` skill のコマンドの表の gate と regate の行)。390(`director_skill_sync_test.go`)。371(`isolated-herdr-server.md` の編集)。464(`/director` skill の変更)。159(org の品質の流れの変更。gate が測る段を足したので当たると読んだが、判断が要る) | evidence の「tech-debt Trigger sweep」。どの行も origin/main と同じで、このブランチの日付の注が無い | /sync-docs で各行に「2026-11-13: feat/gate-tool-versions hit this trigger ... Carried over」を足す。238 (b) は今回の書き直しで直せたので、そう書くか、直す |
| V6 | LOW | 新しい `internal/cli/doctor_node_test.go` は 994 行で、800 行の目安を超える。サイズの行が無い | `wc -l`。`grep -c doctor_node docs/tech-debt/README.md` は新しい 486 行だけ | /sync-docs でサイズの行を足すか、テストを分ける |

INFO(merge を止めない)

- I1: doctor の中断を終了コード 1 にした(S6、`9c19e5ba`)。D6 と AC6 の「exit code は変わらない」とは文の上で食い違う。計画の Progress の S6 の行が「warn の話で、中断は別」と理由を書いているので、記録されたずれとして扱う。D6・AC6 の本文は直っていない。新しい stdout の行(「Interrupted: the Node toolchain check did not finish. Run ralph doctor again.」)はどの文書にも無い
- I2: git が追跡されたファイルを列挙できないとき、doctor は root の直下のファイルから場所を作る(`doctorNodeLayout`)。gate は python3 だけを測る(`RootLayout`)。D8 の「同じ一覧」から外れる場合で、計画には書いていない。テストはある(`TestDoctorNodeToolchain_NotAGitRepositoryReadsTheTopLevelFiles`、`TestDoctorNodeToolchain_ListingFailureSaysWhy`)
- I3: 手順書の「ralph doctor and ralph status」の箇条と `/org` skill の前提は、読む doctor の Check に「Node toolchain」を挙げない。手順書の #308 の節から新しい節への参照も無い(S4 の申し送りどおり /sync-docs の担当)
- I4: self-review の 2 回目の LOW 2(文字を濾す処理が 3 か所)は、まだ tech-debt に無い。計画は /sync-docs で書くとしている
- I5: Go の注釈 5 か所が `docs/plans/active/2026-11-12-gate-tool-versions.md` を指す。`archive-plan.sh` は Go の注釈を書き換えない。repo 全体の前からの形
- I6: issue #311 の受け入れ条件 1 は「走った Node と pnpm の版」。gate が残すのは、スクリプトの前に `/bin/sh` が解決した版で、shim を通した子の版ではない。計画は D1・R2 でそう決め、tech-debt 485 行(MEDIUM)に残した。issue の場面では、python3 の行の `(mise shim)` と `# mise current:` の行が手がかりになる

## Observational checks

- HEAD から `go build ./cmd/ralph` で作った binary を、scratch の git repo(root に `.node-version` の 99、`engines.node` の `>=99`、`packageManager: pnpm@1.0.0`、`pnpm-lock.yaml`。`apps/a` に `.nvmrc` の `lts/*` と `engines.node` の `lts/*`)で `ralph doctor --state-dir <存在しないパス>` として打った。`$SHELL` は `/bin/zsh`。6 秒で終わり、終了コード 0。「Node toolchain」は warn 3 行(root の `.node-version`、root の `engines.node`、`apps/a` の `packageManager`。解決先に「a mise shim」が付く)と info 1 行(別名と読めない範囲)だった。AC6 の warn・info と、warn が終了コードを変えないことを、実際の login shell で確かめた
- 同じ実行の副作用が V1。この verify が `~/.local/share/pnpm/` に pnpm@1.0.0 を入れ、`global/v11/pnpm-lock.yaml` を書いた。書く前の中身は分からないので、戻していない
- このリポジトリには JS の印が無い(`git ls-files` に該当なし)ので、この repo での doctor は shell を打たない

## Coverage gaps

- テストの実行結果(`/test`)。上の AC の根拠はテストのソースとコードを読んだもの
- 実機の herdr と座席での gate。codex の座席の shell と doctor の `$SHELL -lc` の違い(tech-debt 486 行)
- mise と corepack の変数が実際に取得と pin を止めること(計画の Assumptions は文書による。偽の shim の試験だけ)
- V1 の pnpm の設定の名前と、それで止まるか
- doctor の shell の失敗と時間切れの warn は、テストのソースだけ

足すと確かさがいちばん上がる検査: `TestProbe_ChangesNothingWithShimsThatWouldPinAndInstall` の形で、`packageManager` を見て別の版を取りに行く偽の pnpm(V1 の設定の変数で止まる)を足す。

## Verdict

- Verified: AC1〜AC9 はコードとテストのソースから満たす。AC6 は実際の login shell でも観察した。AC10 は書かれた条件(7 つの変数、呼ぶ側の値の上書き、偽の shim の 2 つの試験、スクリプトの環境が変わらないこと)を満たす。issue #311 の受け入れ条件 1・2 は AC1・AC6 に、計画助言の 2 件は D2・D7 と AC3・AC10 に入っている。HEAD `a6657d48` の静的解析はすべて通った。doctor の Check の並び(3、3b … 13、14)と、取り込んだ手順書の節は食い違わない
- Partially verified: AC10 の見出しの「測ることで何も変えない」。pnpm 自身の版の切り替えは止まらず、worktree の外に書く(V1)
- Not verified: テストの実行結果。実機の座席での gate。mise と corepack の変数の実際の効き目
- Drift(/sync-docs へ): V2 README、V3 仕様の FR-7 の注、V5 tech-debt の Trigger 13 行、V6 サイズの行。V4 はコードの注釈。I1〜I6 は任意
- Verdict: pass(MEDIUM の V1 は /pr の前に直すか記録するかを lead が決める)
