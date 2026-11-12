# Test report: codex-hook-trust-check

- Date: 2026-11-12(run-test.sh は 22:30:54 から 22:50:52 JST、追加の回は 22:30 と 22:51〜22:55 JST)
- Cycle 2: 2026-11-13、HEAD `0b5b1e74`。末尾の「Cycle 2」の節に結果を書き、`## Verdict` を cycle 2 のものにした。ここから「Test gaps」までは cycle 1(HEAD `6441d229`)の記録で、書き換えていない
- Plan: `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(`## Test plan`、`## Risks and mitigations`、AC1〜AC9。issue #310)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff origin/main...HEAD`(HEAD `6441d229`、origin/main `d2e5c6e9` は `f36cfc52` で取り込み済み、21 ファイル)。本番のコードは `dddb946d` から変わっておらず、そのあとのコードの変更は `1f7ccadb` のテスト 3 ファイルだけである。run-test.sh は changed scope のままで golang を選んだ(`Language scope: changed (changed_languages)`)。worktree の production code とテストには手を入れていない
- Evidence: `docs/evidence/test-2026-11-12-codex-hook-trust-check.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。先頭に run-test.sh の全出力(wrapper 自身の log は `docs/evidence/verify-2026-11-12-133054.log`)、その後ろに起動の meta、30 秒ごとの空き容量と load、`-run 'CodexHookTrust' -v` の全出力、coverage、probe のソースと出力
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った。shell の SKIP の行は前の PR の回と同じ 2 行だった

## 1 回目の試行(無効)

この /test の 1 回目は、run-test.sh の shell の段で Data volume が埋まり(ENOSPC)、Go の段に入る前に rc 1 で終わった。落ちたのは一時ファイルを書く shell のテストだけで、原因は他のセッションの変異テストで膨らんだ Go の build cache である。結果は無効として扱い、report は書いていない。そのあと build cache が整理され(今回の開始時の `~/Library/Caches/go-build` は 21 GiB)、`1f7ccadb` が前の tester の挙げた穴のテスト(G1〜G4)を足した。

今回は開始前に `df -h /System/Volumes/Data` で空きが 37 GiB であることを確かめ、run-test.sh のあいだ 30 秒ごとに空きを記録した。最小は 35.8 GiB で、5 GiB を下回ることは無かった。log の `no space left` は 0 件である。指示に従い、変異テストは回していない。

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,198 秒(shell の段は約 9 分、Go の段は約 11 分。Go の終わりと org の所要時間から逆算した概数) |
| └ Go: `internal/cli` | 上位 1,156 本 | ok | 0 | — | 499.5 秒 |
| └ Go: `internal/org` | 上位 1,668 本 | ok | 0 | — | 655.1 秒 |
| └ Go: `cmd/ralph`、`internal/config`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | ok。`(cached)` は 0 | 0 | — | 0.7、14.0、3.2、4.3、3.3、1.3、1.6、2.9 秒 |
| `go test ./internal/cli/ -run 'CodexHookTrust' -v -count=1` | 上位 50 本(subtest 込みの RUN 126) | 126 | 0 | 0 | 9 秒(テスト自体は 8.1 秒) |
| coverage: `go test ./internal/cli/ -run 'CodexHookTrust\|CodexEffectiveConfig\|CheckCodexAgmsgWritableRoot\|CodexSandboxEnvFromOS\|RoleModels' -count=1 -coverprofile` | 絞った回 | ok | 0 | — | 22 秒 |
| probe: `CODEX_HOME` が空の経路と decode エラーの位置(overlay で足した 1 本、コミットしない) | 2 本(subtest 8) | 8 | 0 | 0 | 1.1 秒(build を除く) |

上位の本数は `go test -list .` で数えた。Skipped の 2 行は前の PR の回と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらも手元の環境によるもので、この diff と関係がない。

run-test.sh のあいだ、1 分の load は 4.9 から 15.9 だった。落ちたテストは無く、打ち直しはしていない。

### `1f7ccadb` のテストは SKIP でなく PASS した

`-run 'CodexHookTrust' -v` の出力に `--- SKIP` は 0 行で、`=== RUN` の 126 行すべてに `--- PASS` が対応した。`1f7ccadb` が足した 6 本は次のとおりである。

| テスト | 結果 |
| --- | --- |
| `TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings`(AC9 の起動の 1 行) | PASS。6 行(`real_only`、`link_only`、`both` と、その `relative,` の 3 行)すべて PASS |
| `TestCodexHookTrustNotice_StartWithClaudePrintsNothing` | PASS |
| `TestCodexHookTrustNotice_StartPlanWithClaudePrintsNothing` | PASS |
| `TestCodexHookTrustNotice_DirectorStartWithClaudePrintsNothing` | PASS |
| `TestCodexHookTrust_RootOfABareRepositorysLinkedWorktree` | PASS |
| `TestCheckCodexHookTrust_WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksAreUnreadable` | PASS。`a_directory` と `mode_000` の 2 行とも PASS |

`mode_000` の行は、root で走るときと mode 000 でも読めてしまう FS では Skip するが、この機械(root でない利用者、APFS)では PASS した。symlink を作れないときに Skip する `codex_hook_trust_test.go:62` の helper も Skip せず、symlink の行(`RootCandidatesOfASymlinkedMainCheckout` の 3 行、doctor と起動の 1 行の symlink のテスト)はすべて PASS した。CI(`ubuntu-latest`)で同じく走るかは確かめていない。

## 受け入れ条件とテスト

どのテストも run-test.sh のパッケージ全体の回と、上の `-run 'CodexHookTrust' -v` の回で PASS した。テストと AC の対応は verify report の表に従い、ここでは実行の結果だけを書く。

| AC | 走ったテスト(すべて PASS) |
| --- | --- |
| AC1 未承認で warn、数と手順 | `TestCheckCodexHookTrust_WarnsNamingTheHookNotApproved`、`…NamesAtMostThreeHooks` |
| AC2 すべて承認済みで pass、hash を見ない旨と Hooks need review | `TestCheckCodexHookTrust_PassWhenEveryHookIsApproved` |
| AC3 `enabled = false`、`trusted_hash` なし、設定なし、読めない・decode できない設定 | `TestCodexHookTrust_ClassifyUnderOneSpelling`、`TestCheckCodexHookTrust_WarnsWhenAHookIsTurnedOff`、`…WarnsWhenTheCodexConfigIsMissing`、`…WarnsWhenTheCodexConfigIsUnreadable`、`…WarnsWithOnlyThePositionOfAnUndecodableConfig` |
| AC4 not needed、info と warn の分け方 | `TestCheckCodexHookTrust_NotNeededWithoutCodexReadsNothing`、`…InfoWhenTheConfigPathCannotBeResolved`、`…InfoWhenItsOwnHooksJSONCannotBeRead`、`…WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksCannotBeRead`、`…AreUnreadable`(`1f7ccadb`) |
| AC5 linked worktree から main の hooks.json と鍵 | `TestCodexHookTrust_RootFollowsCodexsWalk`、`…RootOfALinkedWorktreeMadeByGit`、`…RootOfABareRepositorysLinkedWorktree`(`1f7ccadb`)、`TestCheckCodexHookTrust_LinkedWorktreeUsesTheMainCheckoutsKeys` |
| AC9 symlink の綴りの両方向 | `TestCodexHookTrust_ClassifyNeedsEverySpelling`、`TestCodexHookTrustFor_SymlinkedMainCheckoutNeedsBothSpellings`、`TestCheckCodexHookTrust_SymlinkedMainCheckoutNeedsBothSpellings`、`…FromDotNeedsBothSpellings`、`TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings`(`1f7ccadb`) |
| AC6 spawn、start の 2 つの形、director start の 1 行 | `TestCodexHookTrustNotice_SpawnPrintsOneLineForACodexSeat`(11 行)と start、`--plan`、director start の codex の対、claude の 3 本(`1f7ccadb`) |
| AC7 event 名と番号 | `TestCodexHookTrust_EventLabelsAreCodexs14`、`…SlotsCountPositionsLikeCodex` |
| AC8 Check 3 の pass の文、`--help` | `TestCheckCodexEffectiveConfig_PassPointsAtCodexHookTrust`、`TestRunDoctor_CodexHookTrustFollowsCodexEffectiveConfig`、`TestCodexHookTrustHelp_EachVerbDescribesTheNote`。README と `.codex/README.md` の文はテストで固定していない(/verify の範囲) |

## Coverage

- Statement: 新しい 3 ファイルは、上の絞った回で 244/258 文(94.6%)。`codex_hook_trust_notice.go` 32/32、`codex_hook_trust.go` 134/142、`doctor_codex_hook_trust.go` 78/84。この diff が足した行のうち cover profile の block に入る行は、`codex_hook_trust_notice.go` 40/40、`doctor.go` 3/3、`org.go` 9/9、`org_director_headless.go` 4/4、`codex_hook_trust.go` 166/180、`doctor_codex_hook_trust.go` 100/107、`doctor_codex_writable_root.go` 10/12(`git diff -U0 origin/main...HEAD` の足した行と profile を突き合わせた)。名前で絞った回なので、パッケージ全体の数字(30.3%)は意味を持たない
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- Function: `checkCodexHookTrust`、`codexHookTrustHooksProblem`、`codexHookTrustGapDetail`、`codexHookTrustApprovalStep`、`printCodexHookTrustNotice`、`codexHookTrustNotice`、`parseCodexHookSlots`、`codexHookRootAt`、`classifyCodexHookSlot` などは 100%。100% でないのは `codexHookTrustSameDir`(44.4%)、`codexHookRootCandidates`(84.6%)、`codexHookTrustFor`(84.6%)、`codexHookTrustConfigProblemDetail`(85.7%)、`codexHooksJSONError`(88.9%)、`readCodexFileBounded`(88.9%)、`codexLinkedWorktreeMain`(89.5%)、`codexHookRoot`(90.0%)
- Notes: 通らなかった足した行は、どれもエラーの枝である(下の Test gaps の 1)。パッケージ全体の回の coverage は取っていないので、ほかのテストがそれらの行を通るかは確かめていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

1 回目の試行の失敗は ENOSPC によるもので、テストの失敗ではない(上の「1 回目の試行」)。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| spawn・start・director start の既存のテストが stderr を完全一致で見ている(計画の Risks の 1 つ目) | 変わらない | `internal/cli` の全体(上位 1,156 本)が 499.5 秒で ok。1 行が既存のテストの stderr に混ざって落ちたものは無い |
| init した repo で doctor を流す既存のテストの warn の数や出力(Risks の 2 つ目) | 変わらない | 同じ回で doctor の既存テストがすべて ok |
| テストが利用者の `~/.codex/config.toml` を読む(Risks の 3 つ目) | 読まない | TestMain の固定(`doctorCodexSandboxEnv`)のまま全体が通った。新しいテストは自分の codex home を渡す。probe も一時ディレクトリの `HOME` と `CODEX_HOME` だけを使った |
| S5 で直した、相対の dir(doctor の `.`)で実パスの候補が落ちる不具合 | 直ったまま | `TestCheckCodexHookTrust_SymlinkedMainCheckoutFromDotNeedsBothSpellings` と、`1f7ccadb` の `relative,` の 3 行が PASS |
| Check 3 の pass の文の変更(`cli_test.go` の `TestCheckCodexEffectiveConfig_FullyWired_Pass` はコメントだけを直した) | 通る | 同じ回で ok |

## Probe(コミットしない)

計画の Test plan の Edge cases の「`CODEX_HOME` が空」は、verify report が「テストが無い」と挙げた。`internal/cli/zz_probe_codex_home_test.go` を `go test -overlay` で足し、本番の resolver `codexSandboxEnvFromOS` を通して doctor の Check と起動の 1 行を確かめた。全文は evidence にある。

- `CODEX_HOME=""` で `$HOME/.codex/config.toml` に承認がある: doctor は pass(「approved in ~/.codex/config.toml」)、起動の 1 行は出ない
- `CODEX_HOME=""` で設定が無い: doctor は warn(「no codex user config at ~/.codex/config.toml」)、起動の 1 行が出る
- 対照として、`CODEX_HOME` を別の空のディレクトリにして承認を `$HOME/.codex` にだけ置くと、doctor は warn で、そのディレクトリの `config.toml` を名指しする。1 行目の pass が `CODEX_HOME` を無視した結果でないことを示す

同じ probe で、`[hooks.state]` の型の違う設定 4 通り(`enabled` が文字列、`hooks` が整数、`state` が文字列、`trusted_hash` が整数)と構文エラー 1 通りを読ませた。どれも warn で「line N, column M」の位置つきになり、仕込んだ値(`SECRETMARK`、`12345`)は文に出なかった。位置の無い decode エラーの文(`doctor_codex_hook_trust.go:140`)に届く入力は見つからなかった。おそらく go-toml v2 の `Unmarshal` は decode の失敗をすべて `*toml.DecodeError` で返すので、その枝には届かない。未確認です。

## Test gaps

1. 絞った回で通らなかった足した行。どれもエラーの枝で、テストは無い。
   - `filepath.Abs` の失敗(cwd が読めない): `codex_hook_trust.go:166-167`、`:260-261`、`:264-265`、`codexHookTrustFor` のその枝 `:447-449`
   - `.git` のファイルが symlink・通常のファイルでない・64 KiB を超える(`:219-220`)、読めない(`:223-224`)。fixture で作れるので、足すならこの 2 つが小さい
   - hooks.json の decode が型でも構文でもないエラーを返したとき(`:146`)
   - `codexHookTrustSameDir` の、どちらかの dir が無いときの比べ方(`doctor_codex_hook_trust.go:119-124`)。doctor の対象の dir が無いときに通る
   - 位置の無い decode エラーの文(`doctor_codex_hook_trust.go:140`)。上の probe では届かなかった
   - `readCodexFileBounded` の、ENOENT でない stat の失敗(`doctor_codex_writable_root.go:198`)と読み込みの失敗(`:212`)
2. `CODEX_HOME` が空のときのコミットされたテストは無い。上の probe で振る舞いは確かめた。足す最小のテストは、probe の `TestZZProbe_EmptyCodexHome` の 3 行(空で承認あり、空で設定なし、別の `CODEX_HOME`)である。
3. 実機の herdr での spawn・start・director start の 1 行は流していない。テストは stub の herdr で回す。
4. codex の版が変わったときの検査は無い(root の求め方、鍵の形、14 種の event)。ずれれば warn に倒れる(計画の Risks)。
5. bare の repo の linked worktree で、codex が ralph と同じ root を出すかは確かめていない(ralph の側だけをテストで固定した)。
6. codex が鍵を作らない handler(prompt・agent 型ほか)を ralph が数える差は、tech-debt と Known gaps に記録されているもので、テストは無い。今の雛形は command 型だけを使う。
7. 今回は変異テストを回していない(空き容量を守るための指示)。テストが guard を固定しているかは、行の coverage でしか見ていない。

## Cycle 2(HEAD 0b5b1e74)

- Date: 2026-11-13(run-test.sh は 00:53:39 から 01:17:47 JST。追加の回は 00:53〜00:58 と 01:18 JST)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 2。`cycle-count.json` は `{"cycle": 2}`)。cross-review cycle 1 の ACTION_REQUIRED 3 件を直したあと
- Scope: `git diff origin/main...HEAD`(origin/main `0936aaea` は `d9103dfc` で取り込み済み)。cycle 1 の test の HEAD `6441d229` のあとでコードを変えたのは `9e72d49a`(入れ子の層、null の形、worktree の持ち主の確かめ)、`f71a2a33`(パーサーを `codex_hook_trust_parse.go` に移すだけ)、`7ace1857`(self-review の指摘の直し)、`0b5b1e74`(`org.go` の注釈だけ)と、origin/main の取り込み `d9103dfc`(#321 の gate lock と accept)である。worktree の production code とテストには手を入れていない
- Language scope: run-test.sh は `full fallback (unclassified:.codex/config.toml)` を出した。cycle 1 の sync-docs(`23a57869`)が言語に分類されない `.codex/config.toml` を変えたためで、選んだ pack は cycle 1 と同じ golang だけだった
- Evidence: `docs/evidence/test-2026-11-12-codex-hook-trust-check-cycle2.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない。cycle 1 の log は上の本文が引くので上書きしていない)。run-test.sh の全出力(wrapper 自身の log は `docs/evidence/verify-2026-11-12-155339.log`)、起動の meta、30 秒ごとの空き容量と load、`(cached)` だった 3 パッケージの打ち直し、`-run 'CodexHookTrust' -v` の全出力、coverage、probe のソースと出力を入れた
- 起動の仕方: cycle 1 と同じく、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った
- 空き容量: 開始前の `df -h /System/Volumes/Data` は 66 GiB、起動の直前は 71 GiB だった。run-test.sh のあいだ 30 秒ごとに記録した 49 回の最小は 71 GiB(ほかのセッションが消したらしく、終わりには 91 GiB)で、5 GiB を下回ることは無かった。log の `no space left` は 0 件。指示に従い、変異テストは回していない

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,448 秒(shell の段は約 11.5 分、Go の段は約 12.5 分。Go の段の始まりは 30 秒ごとの観察と org の所要時間から見積もった) |
| └ Go: `internal/cli` | 上位 1,180 本 | ok | 0 | — | 610.8 秒 |
| └ Go: `internal/org` | 上位 1,724 本 | ok | 0 | — | 756.4 秒 |
| └ Go: `cmd/ralph`、`internal/config`、`internal/insights`、`internal/org/driver`、`internal/scaffold` | — | ok | 0 | — | 0.8、14.6、1.6、3.4、2.8 秒 |
| └ Go: `internal/gitroot`、`internal/org/protocol`、`internal/upgrade` | — | run-test.sh では `(cached)`。`go test -count=1` で打ち直して ok | 0 | — | 打ち直しで 2.6、1.0、1.9 秒 |
| `go test ./internal/cli/ -run 'CodexHookTrust' -v -count=1` | 上位 64 本(subtest 込みの RUN 199) | 199 | 0 | 0 | 16 秒(テスト自体は 12.5 秒) |
| coverage: `go test ./internal/cli/ -run 'CodexHookTrust\|CodexEffectiveConfig\|CheckCodexAgmsgWritableRoot\|CodexSandboxEnvFromOS\|RoleModels' -count=1 -coverprofile` | 絞った回 | ok | 0 | — | 30 秒 |
| probe: `CODEX_HOME` が空の経路と、3 つの層の hooks.json の並べ方(overlay で足した 1 本、コミットしない) | 2 本(subtest 3) | 4 | 0 | 0 | 1.0 秒(build を除く) |

上位の本数は `go test -list .` で数えた。`internal/cli` の 1,180 本は cycle 1 の 1,156 本より 24 本多い。内訳は cycle 2 の codex hook trust のテスト 14 本と、origin/main から入った #321 のテスト 10 本(`TestOrgGate…`、`TestOrgGateAccept…`、`TestOrgMerge_MergesAnAcceptedBranch`、`TestOrgMergeOrder_HelpNamesTheAcceptedClass`)で、消えたテストは無い。Skipped の 2 行は cycle 1 と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。

run-test.sh のあいだ、1 分の load は 5.8 から 11.3 だった。落ちたテストは無く、打ち直しは `(cached)` の 3 パッケージだけである。

### cycle 2 のテストは SKIP でなく PASS した

`-run 'CodexHookTrust' -v` の出力に `--- SKIP` は 0 行で、`=== RUN` の 199 行すべてに `--- PASS` が対応した。cycle 1 の test の HEAD `6441d229` と HEAD の `^func Test` を比べて見つけた、cycle 2 の 14 本は次のとおりである。

| テスト | AC | 結果 |
| --- | --- | --- |
| `TestCodexHookTrustFor_ReadsEveryLayerDownToTheDirectory` | AC10 | PASS |
| `TestCodexHookTrustFor_LinkedWorktreeReadsNestedLayersFromTheMainCheckout` | AC10 | PASS |
| `TestCodexHookTrustFor_NestedLayerProblems` | AC10、AC11 | PASS(subtest 2) |
| `TestCheckCodexHookTrust_CountsNestedLayersDownToTheDirectory` | AC10 | PASS |
| `TestCheckCodexHookTrust_LinkedWorktreeChecksNestedLayersUnderTheMainCheckout` | AC10 | PASS |
| `TestCheckCodexHookTrust_ReportsANestedHooksJSONItCannotRead` | AC10、AC11 | PASS |
| `TestCheckCodexHookTrust_SaysTheOtherLayersApprovalsWereNotChecked` | AC10 | PASS |
| `TestCodexHookTrustNotice_SpawnInASubdirectoryCountsItsNestedHooks` | AC10(起動の 1 行) | PASS |
| `TestCodexHookTrust_SlotsRejectNullAndMissingFieldsLikeCodex` | AC11 | PASS(subtest 30) |
| `TestCodexHookTrust_SlotsAcceptWhatCodexAccepts` | AC11 | PASS |
| `TestCodexHookTrust_SlotErrorsQuoteAtMost64Runes` | AC11 | PASS(subtest 6) |
| `TestCheckCodexHookTrust_NullHookStructuresAreNeverAPass` | AC11 | PASS(subtest 7) |
| `TestCodexHookTrustNotice_PrintsNothingForAHooksJSONCodexRefuses` | AC11(起動の 1 行は出さない) | PASS |
| `TestCodexHookTrust_RootChecksTheLinkedWorktreesOwnership` | AC12 | PASS(subtest 13) |

verify report が「symlink を作れない環境で Skip する」と挙げた `RootChecksTheLinkedWorktreesOwnership/the_gitdir_a_symlink` の行は、この機械(root でない利用者、APFS)で PASS した。cycle 1 の `mode_000` の行と symlink の行も、引き続きすべて PASS した。CI(`ubuntu-latest`)で同じく走るかは確かめていない。

AC1〜AC9 のテスト(上の「受け入れ条件とテスト」の表)は、同じ回ですべて PASS した。`f71a2a33` で `codex_hook_trust_parse_test.go` に移っただけの 4 本(`EventLabelsAreCodexs14`、`SlotsCountPositionsLikeCodex`、`SlotsOfAFileWithNoHandlers`、`SlotsRejectWhatIsNotCodexsShape`)も PASS した。cycle 1 のテストで期待を変えたのは `TestCheckCodexHookTrust_InfoWhenItsOwnHooksJSONCannotBeRead` の「not JSON」の行だけで、`7ace1857` が文言を「not valid JSON (at byte 1)」から「syntax error at byte 1」にした。byte の位置は今も固定している。

### Regression checks(cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| spawn・start・director start の既存のテストが stderr を完全一致で見ている(計画の Risks の 1 つ目)。cycle 2 で入れ子の層を数えるようになった | 変わらない | `internal/cli` の全体(上位 1,180 本)が 610.8 秒で ok |
| init した repo で doctor を流す既存のテストの warn の数や出力(Risks の 2 つ目) | 変わらない | 同じ回で doctor の既存テストがすべて ok |
| テストが利用者の `~/.codex/config.toml` を読む(Risks の 3 つ目) | 読まない | TestMain の固定のまま全体が通った。probe も一時ディレクトリの `HOME` と `CODEX_HOME` だけを使った |
| origin/main の取り込み(`d9103dfc`、#321)と、この branch の `org.go` の変更が混ざること | 通る | `internal/cli` と `internal/org` の全体が ok。#321 のテスト 10 本も同じ回で通った |
| S5 で直した、相対の dir(doctor の `.`)で実パスの候補が落ちる不具合 | 直ったまま | `TestCheckCodexHookTrust_SymlinkedMainCheckoutFromDotNeedsBothSpellings`、`TestCodexHookTrust_RootCandidatesOfARelativeDirInASymlinkedMainCheckout`、`TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings` の `relative,` の 3 行が PASS |
| cycle 1 の probe で見た `CODEX_HOME` が空のときの振る舞い | 変わらない | 下の probe で、層の一覧になった報告でも同じ 3 つの結果になった |

### Coverage(cycle 2)

- Statement: 上の絞った回で、本体 4 ファイルは 502/531 文(94.5%)。`codex_hook_trust_notice.go` 30/30、`codex_hook_trust.go` 206/216、`codex_hook_trust_parse.go` 169/180、`doctor_codex_hook_trust.go` 97/105。名前で絞った回なので、パッケージ全体の数字(32.9%)は意味を持たない
- この diff が足した行(`git diff -U0 origin/main...HEAD -- internal/` の非テストの `.go`)のうち cover profile の block に入る行は 642/694。`codex_hook_trust_notice.go` 40/40、`doctor.go` 3/3、`org.go` 9/9、`org_director_headless.go` 4/4、`codex_hook_trust.go` 252/272、`codex_hook_trust_parse.go` 201/222、`doctor_codex_hook_trust.go` 123/132、`doctor_codex_writable_root.go` 10/12
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- Notes: 通らなかった足した行の分け方は下の Test gaps の 1 と 2。パッケージ全体の回の coverage は取っていないので、ほかのテストがそれらの行を通るかは確かめていない

### Probe(cycle 2、コミットしない)

`internal/cli/zz_probe_c2_test.go` を `go test -overlay` で足した。worktree は変えていない(前後の `git status --porcelain` は空)。全文は evidence にある。

- `CODEX_HOME` が空: cycle 1 の probe の 3 行を HEAD で流し直した。`$HOME/.codex/config.toml` に承認があれば doctor は pass で起動の 1 行は出ない。設定が無ければ doctor は warn で 1 行が出る。対照として `CODEX_HOME` を別の空のディレクトリにすると、承認が `$HOME/.codex` にあっても warn で、そのディレクトリの `config.toml` を名指す。cycle 1 と同じ結果だった
- 3 つの層(root、`sub`、`sub/deeper` のそれぞれに hooks.json): `codexHookTrustPathList` の 3 つ以上を並べる枝(`doctor_codex_hook_trust.go:262`)はコミットされたテストが通らない。`sub/deeper` から流すと、root だけ承認したときの doctor は warn「2 of 4 hooks in <root>, <sub>, and <deeper> are not approved …」で、承認の手順はいちばん深い `sub/deeper` を名指した。すべて承認すると pass「4 hooks in <root>, <sub>, and <deeper> approved …」になった。起動の 1 行も同じ並べ方の 1 行だった

### Test gaps(cycle 2)

cycle 1 の Test gaps 1〜7(tech-debt の 478 行目が「Test gaps 1 to 5」として引く)は上にそのまま残した。cycle 2 で分かったことと、変わったことを書く。

1. 絞った回で通らなかった、cycle 2 で足された行。fixture で作れて、テストが無いもの:
   - `codexWorktreeRegisteredAt` の、登録の `commondir` のファイルが空か空白だけのとき(`codex_hook_trust.go:188-189`)。`RootChecksTheLinkedWorktreesOwnership` には「no commondir file」の行はあるが、空のファイルの行は無い
   - `codexMainOwnsCommonDir` の、`<main>/.git` がファイルで gitdir の行を持たないとき(`:211-212`)
   - `codexReadGitdirFile` が `codexReadGitMetadata` の拒否を受けるとき(`:226-227`。`.git` のファイルが symlink、通常のファイルでない、64 KiB を超える)と、`codexReadGitMetadata` の読み込みの失敗(`:246-247`)。cycle 1 の Test gaps 1 の 2 つ目と同じ穴で、tech-debt 478 行目の (b) に当たる。cycle 2 で `.git` のファイルのほかに登録の `gitdir` と `commondir` もこの関数を通るようになった
   - `codexHookTrustPathList` の 3 つ以上を並べる枝(`doctor_codex_hook_trust.go:262`)。上の probe で振る舞いは確かめた。足すなら probe の `TestZZProbe_ThreeLayersNameEveryHooksJSON` がそのまま使える
2. 絞った回で通らなかった行で、おそらく今の呼び出し方では届かないもの。どれも未確認です。
   - `filepath.Abs` の失敗(cwd が読めない): `codex_hook_trust.go:83-84`、`:281-282`、`:285-286`、`:621-622`
   - `Lstat` がディレクトリと言ったあとの `EvalSymlinks` の失敗(`:145-146`)。競合のときだけ
   - `ApprovalDirs` の空の `NotApproved`(`:600-601`)と `codexHookTrustPathList` の 0 個(`doctor_codex_hook_trust.go:256`)。呼び出し側は、未承認の hook があるときと、読めた hooks.json があるときだけ呼ぶように見える
   - パーサーの、すでに `json.Unmarshal` を通った JSON の decode の失敗(`codex_hook_trust_parse.go:148-149`、`:306-319`、`:339-340`、`:394-395`)、位置の無い「syntax error」(`:290`)、`codexJSONKind` の空の入力(`:429-430`)、`codexJSONPath` の空の `at`(`:482-483`)
   - cycle 1 から持ち越した `codexHookTrustSameDir` の dir が無いときの比べ方(`doctor_codex_hook_trust.go:159-164`)、位置の無い decode エラーの文(`:180`)、`readCodexFileBounded` の 2 つ(`doctor_codex_writable_root.go:198`、`:212`)
3. `CODEX_HOME` が空のときのコミットされたテストは、cycle 2 でも無い(tech-debt 478 行目の (a))。verify report の V5 のとおり、この PR の cycle 2 は 478 行目の Trigger を引いた
4. verify report の I6(codex の `project_root_markers` を変えたとき、別の checkout への symlink の cwd)と I7(linked worktree の入れ子の hooks.json が main に無いとき)の形を見るテストは無い
5. AC10〜AC12 は codex の実機とは比べていない(verify はソースと比べた)。実機の herdr での起動の 1 行も流していない
6. 今回も変異テストを回していない(空き容量を守るための指示)。テストが guard を固定しているかは、行の coverage でしか見ていない。implementer の報告(S7 の変異 5 つを検出)は、この report では確かめていない

## Verdict

- Pass: cycle 2(HEAD `0b5b1e74`)の run-test.sh の全体(rc 0、1,448 秒。shell 45 本、Go 10 パッケージ。`internal/cli` 610.8 秒、`internal/org` 756.4 秒、ほか 5 パッケージは 0.8〜14.6 秒、`(cached)` だった 3 パッケージは打ち直して 1.0〜2.6 秒)、`-run 'CodexHookTrust' -v` の 199 行(cycle 2 の 14 本はすべて PASS、SKIP 0)、coverage の絞った回、probe の 4 行
- Fail: 0
- Blocked: 0。空き容量は 71 GiB を下回らず、ENOSPC は無かった
- Verdict: pass

### Cycle 1(HEAD 6441d229)

- Pass: run-test.sh の全体(shell 45 本、Go 10 パッケージ。`internal/cli` 499.5 秒、`internal/org` 655.1 秒、ほか 8 パッケージは 0.7〜14.0 秒)、`-run 'CodexHookTrust' -v` の 126 行(`1f7ccadb` の 6 本はすべて PASS、SKIP 0)、coverage の絞った回、probe の 8 行
- Fail: 0
- Blocked: 0。1 回目の試行は ENOSPC で無効になり report を書かなかった。今回は空き 35.8 GiB 以上を保って最後まで走った
- Verdict: pass
