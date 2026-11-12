# Test report: codex-hook-trust-check

- Date: 2026-11-12(run-test.sh は 22:30:54 から 22:50:52 JST、追加の回は 22:30 と 22:51〜22:55 JST)
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

## Verdict

- Pass: run-test.sh の全体(shell 45 本、Go 10 パッケージ。`internal/cli` 499.5 秒、`internal/org` 655.1 秒、ほか 8 パッケージは 0.7〜14.0 秒)、`-run 'CodexHookTrust' -v` の 126 行(`1f7ccadb` の 6 本はすべて PASS、SKIP 0)、coverage の絞った回、probe の 8 行
- Fail: 0
- Blocked: 0。1 回目の試行は ENOSPC で無効になり report を書かなかった。今回は空き 35.8 GiB 以上を保って最後まで走った
- Verdict: pass
