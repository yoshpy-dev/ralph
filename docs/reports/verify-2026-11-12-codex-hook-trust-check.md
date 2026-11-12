# Verify report: codex-hook-trust-check

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(issue #310。`- Approved:` の digest `147d7ec0e7ae` は `./scripts/plan-visual.sh digest` で出し直した値と一致した。HEAD `75f3757d` でも同じ値)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)。この版は、HEAD `dddb946d` で書いて `9492dc5a` でコミットした最初の版に、そのあとのコミットを足した更新
- Scope: `git diff origin/main...HEAD`(HEAD `75f3757d`。origin/main `d2e5c6e9` は f36cfc52 で取り込み済み。21 ファイル、+3169 / -77)。最初の版の範囲は HEAD `dddb946d`(20 ファイル、+2796 / -77)。そのあと `9492dc5a` がこの report を足し、`1f7ccadb` がテスト 3 ファイルだけを足し(+231 / -0)、`75f3757d` が self-review の report を更新した。本番のコードは `dddb946d` から変わっていない(`git diff --stat 9492dc5a..HEAD -- '*.go' ':!*_test.go'` が空)。静的解析は `./scripts/run-static-verify.sh` の既定(変わった言語だけで、今回は golang)を HEAD `75f3757d` で取り直した。テストは走らせていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-11-12-codex-hook-trust-check.log`(git が無視する場所)。最初の版の出力(run-static-verify.sh の全出力と下の観察の probe の出力)を先頭にそのまま残し、この更新の出力を末尾の「Update at HEAD 75f3757d」の節に足した(前の 12417 byte が新しい log の先頭と一致することを `head -c | cmp` で確かめた)
- Insight event: 最初の版で `docs/insights/events/2026-11-12-codex-hook-trust-check.jsonl` に verify の 1 行(cycle 1、pass、low 3)を足し、その版の report と同じコミットに入れた。この更新では足していない。cycle は同じ 1 で、判定も LOW の数(V1〜V3)も変わらず、2 行目を足すと同じ cycle の verify が 2 回に数えられる。self-review の更新(`75f3757d`)も同じ理由で足していない
- ID の付け方: `V1` から `V3`(LOW、/sync-docs に渡す drift)、`I1` から `I4`(INFO。I4 はこの更新で足した)
- Cycle 2: パイプライン 2 回目(`cycle-count.json` は cycle 2)の結果は、`## Verdict` の直前の「Cycle 2(HEAD c0337338)」の節と、`## Verdict` の先頭の行にまとめた。それより上の本文は cycle 1 のまま残した(tech-debt の 478 行目がこの report の「Coverage gaps」を引いている)。cycle 2 の ID は `V4` から `V6`、`I5` から `I8`

## Spec compliance

受け入れ条件ごとに、コードと、それを固定するテストのソースを読んで判定した。テストが通ることは `/test` が確かめる。「満たす(静的)」は、コードがそう動き、その期待値を持つテストがあるという意味で使う。テストの行番号は HEAD `75f3757d` のもので、`1f7ccadb` がテストを挿したため最初の版から動いた行は `grep -n '^func Test'` で引き直した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 driver_pool に codex があり鍵が無い repo で warn、未承認の数と承認の手順(issue の受け入れ条件 1) | 満たす(静的)+ 観察 | `checkCodexHookTrust`(`internal/cli/doctor_codex_hook_trust.go:46`)の outcome 6 と 7。文は `codexHookTrustGapDetail`(`:150`)と `codexHookTrustApprovalStep`(`:184`)が作る。`TestCheckCodexHookTrust_WarnsNamingTheHookNotApproved`(`doctor_codex_hook_trust_test.go:283`)が「1 of 2 hooks … is not approved … (pre_tool_use:0:0)」と「open codex in <repo> and trust the hooks (Hooks need review → Trust all and continue)」を持つ。`TestCheckCodexHookTrust_NamesAtMostThreeHooks`(`:341`)が鍵の末尾を 3 つで切ることを見る。観察: dddb946d の binary の (a) |
| AC2 すべて `trusted_hash` つきなら pass、hash を確かめていないことと Hooks need review を書く(issue の受け入れ条件 2) | 満たす(静的)+ 観察 | outcome 8(`doctor_codex_hook_trust.go:80-82`)。`TestCheckCodexHookTrust_PassWhenEveryHookIsApproved`(`doctor_codex_hook_trust_test.go:357`)が「keys only; the hash is not checked」と「makes codex show Hooks need review in the seat's pane」を持つ。観察: dddb946d の binary の (b)。実機の codex 0.166.0 の `hooks/list` と食い違わないことは `docs/evidence/codex-hook-trust-worktree-2026-11-12.md` の 4 節(binary は 0d813954。check のコード 6 ファイルは 0d813954 から HEAD まで変わっていない) |
| AC3 `enabled = false`、`trusted_hash` の無い鍵、設定ファイルが無いときは warn。読めない・decode できない設定は warn で中身を出さない | 満たす(静的)+ 観察 | 分け方は `codexHookTrustUnder`(`codex_hook_trust.go:374`)。`TestCodexHookTrust_ClassifyUnderOneSpelling`(`codex_hook_trust_test.go:512`)が「no trusted_hash」を NotApproved、「turned off」の 2 行を Off にする。doctor では `TestCheckCodexHookTrust_WarnsWhenAHookIsTurnedOff`(`doctor_codex_hook_trust_test.go:295`)、`…WarnsWhenTheCodexConfigIsMissing`(`:246`)、`…WarnsWhenTheCodexConfigIsUnreadable`(`:258`)、`…WarnsWithOnlyThePositionOfAnUndecodableConfig`(`:270`、設定に仕込んだ印と `trusted_hash` と hash の値が文に出ない)。decode の失敗は `codexConfigDecodeErrorOf`(`codex_hook_trust.go:349`)が行と桁だけを持つ。観察: (c) |
| AC4 codex が無ければ pass「not needed」で設定を読まない。hooks.json が無い・読めないとき、Check 3 と同じファイルなら info、違うファイルなら warn | 満たす(静的)+ 観察 | outcome 1 は `resolveEnv` を呼ぶ前に返る(`doctor_codex_hook_trust.go:48-52`)。`TestCheckCodexHookTrust_NotNeededWithoutCodexReadsNothing`(`doctor_codex_hook_trust_test.go:91`)が、壊れた入力でも pass で、`resolveEnv` が呼ばれないことを見る。info と warn の分け方は `codexHookTrustHooksProblem`(`:90`)と `codexHookTrustSameDir`(`:113`)。`…InfoWhenItsOwnHooksJSONCannotBeRead`(`doctor_codex_hook_trust_test.go:121`、無い・ディレクトリ・JSON でない、絶対パスと `.`)、`…WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksCannotBeRead`(`:160`、無い・形が違う)。`1f7ccadb` で `…WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksAreUnreadable`(`:194`)が足された。main の hooks.json がディレクトリの行と mode 000 の行の 2 つで、linked worktree の絶対パスと `.` からは warn「codex seats read hooks from <main の hooks.json>, which could not be read (」と「so no hook runs in a codex seat」、main 自身からは info「could not be read (」と「the same file Codex effective config checks」を見る。どちらも `codexHookTrustHooksProblem` の `codexHookSourceUnreadable` の枝(`doctor_codex_hook_trust.go:95-96`)を通る。mode 000 の行は root で走るときと、mode 000 でも開けてしまう FS では Skip する。観察: (d) |
| AC5 linked worktree からでも main の hooks.json とそのパスの鍵で確かめる。root は codex と同じ規則で、git のコマンドを使わない | 満たす(静的) | `codexHookRoot`(`codex_hook_trust.go:163`)、`codexHookRootAt`(`:188`)、`codexLinkedWorktreeMain`(`:216`)。3 ファイルとも `os/exec` を import していない(grep で 0 件)。`TestCodexHookTrust_RootFollowsCodexsWalk`(`codex_hook_trust_test.go:194`、HEAD の無い `.git`、`worktrees` の下でない gitdir、`.git` の無い main、相対の gitdir、gitdir の行が無い、git の外の 10 行)、`…RootOfALinkedWorktreeMadeByGit`(`:260`)、`TestCheckCodexHookTrust_LinkedWorktreeUsesTheMainCheckoutsKeys`(`doctor_codex_hook_trust_test.go:372`、worktree のパスの鍵だけでは warn)。`1f7ccadb` で `TestCodexHookTrust_RootOfABareRepositorysLinkedWorktree`(`codex_hook_trust_test.go:292`)が足された。git で作った bare repository の linked worktree で、親に `.git` の無い `store.git` のものは worktree 自身が root、`proj/.git` のものは `proj` が root で、`codexHookRootCandidates` も `[proj]` だけを返す。2 つの期待値は `codexLinkedWorktreeMain` の `<main>/.git` の確かめ(`:237-241`)の両側に当たる |
| AC9 root の綴りが 2 つなら、両方の綴りの承認があるときだけ承認済み。片方だけなら両方向で warn(と起動の 1 行) | 満たす(静的)。doctor と起動の 1 行の両方 | `codexHookRootCandidates`(`codex_hook_trust.go:257`)と `classifyCodexHookSlot`(`:392`)。`TestCodexHookTrust_ClassifyNeedsEverySpelling`(`codex_hook_trust_test.go:538`)、`TestCodexHookTrustFor_SymlinkedMainCheckoutNeedsBothSpellings`(`:637`)、doctor の `…SymlinkedMainCheckoutNeedsBothSpellings`(`doctor_codex_hook_trust_test.go:397`)と `…FromDotNeedsBothSpellings`(`:436`、S5 の不具合の再現)。起動の 1 行は `codexHookTrustFor` の同じ結果から作る(`codex_hook_trust_notice.go:36`)。`1f7ccadb` で `TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings`(`codex_hook_trust_notice_test.go:173`)が足された。spawn の `--cwd` を symlink の綴りの絶対パスと、`$PWD` を symlink の綴りにした `.` の 2 通りで打ち、承認が実パスだけ・symlink の綴りだけ・両方の 3 通りと掛けた 6 行。片方だけの 4 行は stderr 全体が「note: org spawn: codex seat implementer: 2 of 2 hooks in <symlink の綴りの hooks.json> are not approved in ~/.codex/config.toml, so they do not run in the seat; run 'ralph doctor' for the keys and how to trust them」の 1 行と完全一致し、両方の 2 行は stderr が空。期待する文は `codexHookTrustNotice` の `n > 0` の枝(`codex_hook_trust_notice.go:62-63`)と `:81` の組み立てに合う |
| AC6 spawn、start の 2 つの形、director start で、未承認・止めてあれば stderr に 1 行が 1 回。claude、すべて承認済み、hooks.json が無いときは出ない。動きと終了コードは変わらない(issue の受け入れ条件 3) | 満たす(静的) | 呼び出しは `internal/cli/org.go:964`(spawn)、`:1199`(start の task の形)、`:1329`(`--plan` の形、main の root)、`org_director_headless.go:366`(director start、director が動く directory)。どれも driver を解決した直後に 1 回。`TestCodexHookTrustNotice_SpawnPrintsOneLineForACodexSeat`(`codex_hook_trust_notice_test.go:65`)は stderr 全体を完全一致で見る 11 行(設定なし、承認なし、1 つ承認、相対の `--cwd`、止めてある、両方、すべて承認、claude、hooks.json なし、読めない設定、decode できない設定)で、どれも exit 0 と dry-run の spawned を見る。start は `:231`、`--plan` は `:293`、director start は `:349`。claude での早い return は `codex_hook_trust_notice.go:29`。`1f7ccadb` で claude の 3 本が足された。`…StartWithClaudePrintsNothing`(`codex_hook_trust_notice_test.go:260`、stderr が空。同じ repository を codex の leader で start すると 1 行が出ることを同じテストの中で見る)、`…StartPlanWithClaudePrintsNothing`(`:323`、stderr に `codex` を含まない。codex の対は `:293`)、`…DirectorStartWithClaudePrintsNothing`(`:374`、stderr が空で、stdout が `started d1 (driver=claude, ` で始まる。この形は `org_director_headless.go:385` の出力) |
| AC7 鍵の event 名は codex の 14 種の snake_case、番号は並びのまま、空の command は数えない | 満たす(静的) | `codexHookEventLabels`(`codex_hook_trust.go:31`)と `parseCodexHookSlots`(`:102`)。`TestCodexHookTrust_EventLabelsAreCodexs14`(`codex_hook_trust_test.go:94`)、`…SlotsCountPositionsLikeCodex`(`:111`、空白だけの command の次の handler が位置 1 を保つ、空の command、知らない event、空の group) |
| AC8 Check 3 の pass の文が新しい Check を指す。README、`.codex/README.md` の 2 写し、`docs/recipes/codex-setup.md` が説明する | 満たす | `doctor.go:469`。`TestCheckCodexEffectiveConfig_PassPointsAtCodexHookTrust`(`doctor_codex_hook_trust_test.go:465`、完全一致)と、doctor の出力で 2 行が並ぶことを見る `TestRunDoctor_CodexHookTrustFollowsCodexEffectiveConfig`(`:484`)。文書は下の drift の表。`--help` の段落は `TestCodexHookTrustHelp_EachVerbDescribesTheNote`(`codex_hook_trust_notice_test.go:468`)と、dddb946d の binary の 3 つの `--help` |

issue #310 の受け入れ条件 3 つは、1 が AC1、2 が AC2、3 が AC6 に入っている。issue の「やること」の hash の突き合わせと「断る設定」は、計画の Non-goals(ユーザーの決定)が外しており、AC2 の pass の文と README、`.codex/README.md`、recipe がそのことを書いている。

`1f7ccadb` の 6 本について、self-review の L6〜L9 はテストの書き方の指摘で、判定には響かない。そのうち 2 つは固定する範囲が兄弟のテストより狭い。L7 の `…StartPlanWithClaudePrintsNothing` は stderr に `codex` を含まないことだけを見る。L8 の読めない hooks.json の 2 行は、読めない理由(ディレクトリなら `not a regular file`)まで見ずに `could not be read (` で止まる。どちらも AC4 と AC6 が求める「出ない」「warn か info か」は固定しているので、上の Status は変えていない。

## Static analysis

HEAD `75f3757d` で取り直した。

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | 既定の changed scope。言語は golang。script 自身の log は `docs/evidence/verify-2026-11-12-132218.log` |
| ・ shellcheck、hook の `sh -n` 20 本、guard の awk、settings.json の `jq -e` | ok | |
| ・ Codex hook single-source guard、inline hook detector の smoke test、PR provenance policy guard | ok | |
| ・ `scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0 |
| ・ `scripts/check-pipeline-sync.sh`、`scripts/check-skill-sync.sh`、`scripts/check-template-purity.sh` | ok | skill は 14 本がそろう |
| ・ tech-debt README の plan の参照 | ok | |
| ・ golang: `gofmt -l`、`go vet ./...`、`golangci-lint run ./...`、`staticcheck ./...` | ok | gofmt ok、golangci-lint 0 issues、go vet と staticcheck は出力なし。golangci-lint と staticcheck が無いときに出る「Skipping」の行は無い |
| ・ `secret-scan-branch.sh` | clean | d2e5c6e9..75f3757d |
| `git diff origin/main...HEAD --check` | clean | |
| 変わった 21 ファイルの U+FFFD の検査 | 0 件 | |
| `./scripts/plan-visual.sh digest`(計画) | `147d7ec0e7ae` | `- Approved:` の値と一致 |

## Documentation drift

`1f7ccadb` はテストだけ、`75f3757d` は self-review の report だけを変えたので、最初の版の V1〜V3、I1〜I3 はそのまま残る。I4 はこの更新で足した。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の Quick start(65〜70 行目) | はい | doctor が、codex が `[org].driver_pool` にあるとき未承認と `enabled = false` を知らせ、鍵だけを見ることを言う |
| README の `ralph org …` の行と `ralph doctor` の行 | はい | org の行は spawn・start(2 つの形)・director start の 1 行と、続けて動くことを言う。doctor の行は、読む hooks.json(linked worktree からは main のもの)、鍵が無い・止めてある・設定が無いときの warn、hash を見ないことを言う |
| `.codex/README.md`(root と雛形。`cmp` で同じ) | はい | One-time setup の 3 番(29〜36 行目)、Trust UX の 2 番(鍵の形)、4 番(宣言全体の hash)、5 番(linked worktree と symlink)、doctor の段落。鍵の形は `codexHookKey` と合う。ダイアログの文言は `docs/evidence/org-seat-resume-probe-2026-11-07.md:119-126` にある。`ralph org resume` がそのダイアログを選択の画面として扱い何も打たないことは `internal/org/resume_screens.go:64` と `:99-100` が言う。hash が宣言全体にかかることは self-review が `rust-v0.166.0` のソースで確かめたもので、ここでは読み直していない |
| `docs/recipes/codex-setup.md`(root と雛形。`cmp` で同じ) | はい | doctor の段落が新しい Check を説明する。古い「per-command-hash」の文は消えた |
| `ralph org spawn`・`start`・`director start` の `--help` | はい | dddb946d から build した binary で 3 つとも段落を 1 回持つ。`--help` の文を持つ `codex_hook_trust_notice.go` は dddb946d から変わっていない |
| Check 3 の pass の文と `checkCodexEffectiveConfig` の注釈 | はい | `doctor.go:379-385`、`:469` |
| `.codex/AGENTS.override.md:56`、`.codex/hooks/README.md:35-36`、README の 197 行目と 304 行目 | はい | 対話での承認が要ることを言うだけで、doctor が確かめられないとは言っていない |
| `docs/recipes/codex-seat-permissions.md` | はい | 計画の Verify plan が挙げた。hook の承認に触れる記述は無い |
| `.codex/config.toml` の 109〜114 行目の注釈(root と雛形、`cmp` で同じ) | いいえ(V1) | 「`ralph doctor` verifies the wiring but cannot probe trust state」と言う。文脈は hook の対話での承認で、今の doctor は「Codex hook trust」でその鍵を見る |
| `docs/tech-debt/README.md` の Trigger | いいえ(V2) | この diff が満たした Trigger 4 件に、持ち越しの記録も支払いも無い(下の内訳) |
| `docs/tech-debt/README.md` の新しい行 | いいえ(V3) | 計画の Scope の最後の項は「Hooks need review」で知らせる案を tech-debt に書くと決め、S1 の記録は数えすぎ(self-review L5)を tech-debt と Known gaps に書くと言う。どちらの行もまだ無い。self-review の Tech debt identified の 2 行がそのまま使える |
| `--help` の段落の「There is no note …」(`codex_hook_trust_notice.go:112-114`) | ほぼ(I1) | 1 行が出ない場合として「hooks.json が無い・不正」「設定が読めない・decode できない」を挙げる。コードは hooks.json が読めないとき(ディレクトリ、1 MiB 超)と、設定のパスが決まらないときも出さない。列挙を「これだけ」とは言っていないので誤りではない |
| `/org` skill の 121〜125 行目(4 つの写し) | はい(I2) | `codex trust .`(project の信頼)の話で、今も正しい。hook の承認と新しい 1 行には触れない。#313 の版の知らせも skill には書いていないので、それと同じ扱いになる |
| 計画の D6 の文と AC のチェックボックス | INFO(I3) | 実装の 1 行の後半は「run 'ralph doctor' for the keys and how to trust them」で、D6 の形と違う。S3 の進捗の記録がこの違いを書いている。D6 は digest の範囲なので直さないのが今の扱い。AC1〜AC9 は `[ ]` のまま |
| 計画の Progress checklist と Test plan | INFO(I4) | Progress には S5 と実機の確かめまでの記録があり、`1f7ccadb` のテストの追加は書かれていない。Test plan の Edge cases のうち、bare の linked worktree は `1f7ccadb` で埋まった。`CODEX_HOME` が空のときのテストは今も無い。Progress に 1 行足すかは任意 |

### V2 の内訳(この diff が満たした tech-debt の Trigger)

`/sync-docs` が、各行に持ち越しの記録を足すか、支払うかを決める。行番号は HEAD の `docs/tech-debt/README.md`(`9492dc5a` から変わっていない)。行数は origin/main d2e5c6e9 と HEAD dddb946d で `wc -l` を取った。`dddb946d` 以降は本番のコードも文書も変わっていないので、下の行数は HEAD `75f3757d` でも同じ。

1. 135 行目「Next touch to `internal/cli/doctor_codex_writable_root.go` for another reason」: `readCodexUserConfig` から `readCodexFileBounded` を切り出し、`codexConfigMaxBytes` の注釈に 1 文を足した。991 行から 1009 行。`pathCovers`、`codexWritableRootSuffix`、読む設定は変わっていない。2026-10-30、11-01、11-05 の持ち越しの記録と同じ形になる。
2. 201 行目の (e)・(f)「the next change to `runOrgStartPlan`, `StartFeature`, or `scriptFeatureWorktrees`」: `runOrgStartPlan`(`internal/cli/org.go:1326-1329`)に `printCodexHookTrustNotice` の呼び出しと 3 行の注釈、その doc に 3 行を足した。新しい呼び出しは `TestCodexHookTrustNotice_StartPlanChecksTheMainCheckout` が走らせる。#313 の 2026-11-12 の記録と同じ形になる。
3. 371 行目「The next edit of any of these five recipes」: `codex-setup.md` の 2 つの写しの doctor の段落を書き直した。`docs/evidence/` の引用は origin/main と HEAD で同じ 1 件(`codex-config-rewrite-2026-09-30.md`)、`docs/tech-debt/README.md` の引用は 0 件で、増えも書き換えもしていない。
4. 415 行目「The next change to any of the five」: `internal/cli/doctor.go` が 1615 行から 1624 行になった(Check 3b の呼び出しと 6 行の注釈、Check 3 の doc と pass の文)。

満たしていないと確かめたもの: 144 行目(`doctor_codex_writable_root_test.go` は変えておらず、target directory の決め方も同じ)、275 行目(`startDirector` に足したのは `ResolveRoleModel` と cwd の決定のあとの 1 行の表示で、拒否の順は変わっていない)、466 行目と 467 行目(`printScaffoldVersionNotice` の引数も `orgScaffoldVersionNoticeHelp` も変えていない)、427 行目の (6)(status の JSON の変更は origin/main の取り込みで入ったもので、この branch のコミットではない)、88 行目、110 行目、223 行目、425 行目(Trigger が名指すファイルや関数を変えていない)。

`1f7ccadb` で増えた Trigger は無い。変えたのはこの branch で作った 3 つのテストファイルだけで、register が名指す helper(275 行目の `directorStartLedger`、398 行目の `setupOrgStubPATH`、425 行目の `commitRalphTomls`、466 行目の `newLegacyLedgerRepo`)は呼ぶだけで変えていない。190 行目が言う bare repository の CLI テストは `MainWorktreeRoot` を通るもので、新しい bare のテストは `codexHookRoot` を通るので当たらない。

行数の観察: 新しいファイルはどれも 800 行より短い(最長は `codex_hook_trust_test.go` で、HEAD で 733 行)。すでに 800 行を超えていた `doctor_codex_writable_root.go`(991 → 1009)と `internal/cli/org.go`(2261 → 2273)が少し伸びた。どちらも大きさを追う行は register に無い。diff の質の判断なので所見にはしない。

## Observational checks

HEAD(dddb946d)から binary を build し、scratch に git の repo を作って `templates/base/.codex/hooks.json` を置き、`ralph doctor` を打った。`env -i` で `HOME` と `CODEX_HOME` を scratch に、`PATH` を `<scratch>/stub:/usr/bin:/bin` に絞った。`codex` と `claude` は版を表示するだけの stub で、本物の codex も利用者の `~/.codex` も使っていない。この更新では build し直していない。`dddb946d` から HEAD `75f3757d` まで本番のコードは変わっていないので、下の観察は HEAD にもそのまま当てはまる。

- (a) 設定なし、既定の driver_pool: warn「no codex user config at ~/.codex/config.toml, so none of the 4 hooks … is approved」と承認の手順
- (b) 4 つの鍵に `trusted_hash`: pass「4 hooks … approved … (keys only; the hash is not checked …)」
- (c) `pre_tool_use:0:0` に `enabled = false`: warn「1 of 4 hooks … is turned off (enabled = false) … (pre_tool_use:0:0)」と、`enabled = false` を消す手順
- (d) `driver_pool = ["claude"]`: pass「not needed: codex is not in [org].driver_pool …」
- `org spawn`・`org start`・`org director start` の `--help`: 段落の文「The Codex hook trust check of」が 1 回ずつ

出力は evidence の log の「Observational probe」の節にある。spawn・start・director start の 1 行を実機の herdr で出すことはしていない(テストは stub の herdr で回す)。

## Coverage gaps

最初の版で挙げた穴のうち、次の 4 つは `1f7ccadb` のテストが埋めた。AC9 の起動の 1 行(symlink の綴りの `--cwd`)、AC4 の読めない main の hooks.json(linked worktree から)、AC6 の claude(start、start --plan、director start)、bare の repo の linked worktree の root(ralph の側)。残りは次のとおり。

- テストは走らせていない。上の表のテストが通ることは `/test` が確かめる。`1f7ccadb` の mode 000 の行は root で走ると Skip し、symlink の行は symlink を作れない環境で Skip する。go test の既定の出力では Skip と pass の見分けが付かない。CI(`.github/workflows/verify.yml`)は `ubuntu-latest` で container を使っていないので、おそらく root ではなく mode 000 の行も走る。未確認です。
- 計画の Test plan の Edge cases のうち、`CODEX_HOME` が空のときを見るテストは無い。
- bare の repo の linked worktree で、ralph が出す root(`store.git` なら worktree 自身、`proj/.git` なら `proj`)はテストで固定された。codex が同じ root を出すかは確かめていない。codex は worktree の登録が checkout を指し返すことと `<main>/.git` が共通の directory であることも確かめるが、ralph は写していない(`codex_hook_trust.go:212-215`)。self-review も codex の `resolve_root` を読み直していない。
- codex の規則(root の求め方、鍵の形、14 種の event、hash の範囲)は codex-cli 0.166.0 の実機と tag `rust-v0.166.0` のソースに合わせてある。版が変わったときの検査は無く、ずれれば warn に倒れる(計画の Risks)。
- 確度をいちばん上げる追加の確かめ: `/test` で `go test ./internal/cli -run 'CodexHookTrust' -v` を流し、`1f7ccadb` の 6 本(`SpawnThroughASymlinkNeedsBothSpellings` の 6 行、claude の 3 本、bare の 1 本、読めない hooks.json の 2 行)が SKIP でなく PASS になったことを出力で確かめる。最初の版が挙げた symlink の起動の 1 行のテストはこの 6 本に入ったので、次に効くのは、それらが手元で本当に走ったことの確認になる。

## Cycle 2(HEAD c0337338)

- Date: 2026-11-13(JST。UTC では 2026-11-12)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は cycle 2)。cross-review cycle 1 の ACTION_REQUIRED 3 件(入れ子の層、null の形、bare repo の持ち主)を直したあと
- Plan: `- Approved:` の digest `e3164744a981` は、`./scripts/plan-visual.sh digest` で出し直した値と一致した
- Scope: `git diff origin/main...HEAD`(origin/main `0936aaea` は `d9103dfc` で取り込み済み。29 ファイル、+5026 / -87)。cycle 1 の verify(`6441d229`)のあとの branch のコミットのうち、コードを変えたのは `9e72d49a`(入れ子の層、null の形、持ち主の確かめ)、`f71a2a33`(パーサーを `codex_hook_trust_parse.go` に移すだけ)、`7ace1857`(self-review の指摘の直し)の 3 つ。`d9103dfc` は origin/main の取り込みで、README の表の 2 行と tech-debt の衝突を両側を残して解いたもの。`83ae4e55`、`798504e2`、`c0337338` は記録だけ。本体 4 ファイル(`codex_hook_trust.go`、`codex_hook_trust_parse.go`、`doctor_codex_hook_trust.go`、`codex_hook_trust_notice.go`)は全部読み、テスト 4 ファイルは AC に当たるテストを読んだ。テストは走らせていない(`/test` の範囲)
- Evidence: 同じ log の末尾に「Cycle 2 at HEAD c0337338」の節を足した(前の 29967 byte が新しい log の先頭と一致することを `head -c | cmp` で確かめた)。`run-static-verify.sh` 自身の log は `docs/evidence/verify-2026-11-12-154038.log`
- codex のソース: AC5、AC10、AC12 の「codex と同じ規則」を確かめるため、`gh api` で tag `rust-v0.166.0` の `codex-rs/git-utils/src/trust.rs`、`codex-rs/config/src/loader/mod.rs`、`codex-rs/config/src/project_root_markers.rs` を読んだ(読み取りだけ。置き場は repo の外の scratch)
- Binary: build していない(lead の頼み)。`--help` は呼び出し側のコードと `TestCodexHookTrustHelp_EachVerbDescribesTheNote` で確かめた
- Insight event: `docs/insights/events/2026-11-12-codex-hook-trust-check.jsonl` に verify の 1 行(`--cycle auto` で cycle 2、pass、`--low 3` で V4〜V6 の 3 件)を足し、この report と同じコミットに入れた

### Spec compliance(cycle 2)

行番号は HEAD `c0337338` のもの。cycle 2 で足されたテストは 14 本(`75f3757d` と HEAD の `^func Test` を比べた。消えたテストは無い)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 driver_pool に codex があり鍵が無い repo で warn、未承認の数と承認の手順 | 満たす(静的) | outcome 7 は `checkCodexHookTrust` の `doctor_codex_hook_trust.go:82-83` から `codexHookTrustGapDetail`(`:191`)。件数の句は `codexHookTrustCounts`(`:222`。`7ace1857` で起動の 1 行と共有になった)、手順は `codexHookTrustApprovalStep`(`:242`。未承認の hook を持ついちばん深い層の dir を名指す)。`TestCheckCodexHookTrust_WarnsNamingTheHookNotApproved`(`doctor_codex_hook_trust_test.go:283`)、`…NamesAtMostThreeHooks`(`:341`)。入れ子の層があると鍵は `sub/.codex/hooks.json:stop:0:0` の形で名指す(`ItemName`、`codex_hook_trust.go:587`)。binary での観察は cycle 1 のもので、cycle 2 では流し直していない |
| AC2 すべて `trusted_hash` つきなら pass、hash を見ないことと Hooks need review を書く | 満たす(静的) | `doctor_codex_hook_trust.go:84-87`。`…PassWhenEveryHookIsApproved`(`:357`)が「keys only; the hash is not checked」と「Hooks need review」を見る。層が 2 つあると、pass の文は両方の hooks.json を名指す(`…CountsNestedLayersDownToTheDirectory`、`:425`、3 つ目の確かめ) |
| AC3 `enabled = false`、`trusted_hash` の無い鍵、設定ファイルが無いときは warn。読めない・decode できない設定は warn で中身を出さない | 満たす(静的) | 分け方は `codexHookTrustUnder`(`codex_hook_trust.go:475`)で、cycle 1 と同じ。`readCodexHookStates`(`:432`)の本文は `75f3757d` と HEAD で同じ(本文の md5 が一致)。`…ClassifyUnderOneSpelling`(`codex_hook_trust_test.go:563`)、doctor の `…WarnsWhenAHookIsTurnedOff`(`:295`)、`…WarnsWhenTheCodexConfigIsMissing`(`:246`)、`…IsUnreadable`(`:258`)、`…WarnsWithOnlyThePositionOfAnUndecodableConfig`(`:270`)。decode の失敗は `codexConfigDecodeErrorOf`(`codex_hook_trust.go:450`)が行と桁だけを持つ |
| AC4 codex が無ければ pass「not needed」で設定を読まない。hooks.json が無い・読めないとき、Check 3 と同じファイルなら info、違えば warn(層の形で読み直した) | 満たす(静的) | outcome 1 は `resolveEnv` の前に返る(`doctor_codex_hook_trust.go:52-56`、`…NotNeededWithoutCodexReadsNothing`、`:91`)。報告が層の一覧になり、問題のある層は `HooksProblems`(`codex_hook_trust.go:532`)が選ぶ。root の層は「無い・読めない・不正」のどれでも、入れ子の層は「読めない・不正」のときだけ。Check 3 と同じファイルかは、その層の `Dirs[0]` と targetDir を `codexHookTrustSameDir`(`doctor_codex_hook_trust.go:153`)で比べ、`codexHookTrustHooksProblem`(`:100`)が info か warn にする。root の層は cycle 1 と同じ期待の `…InfoWhenItsOwnHooksJSONCannotBeRead`(`:121`)、`…WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksCannotBeRead`(`:160`)、`…AreUnreadable`(`:194`)が固定する。入れ子の層は `…ReportsANestedHooksJSONItCannotRead`(`:457`、深い dir からは warn、その層の dir からは info)と `…SaysTheOtherLayersApprovalsWereNotChecked`(`:487`)。動きの変化が 1 つある。自分の `.codex` を持たない linked worktree では、cycle 1 は main の hooks.json を読んだが、今は codex と同じく root の層を worktree 自身にして「hooks.json が無い」と報告する(`layers`、`codex_hook_trust.go:340-352`。codex の `discover_project_layers` は `.codex` の無い dir を層にしない)。それは Check 3 が読むファイルなので info になり、AC4 の文面に合う |
| AC5 linked worktree からでも main の hooks.json とそのパスの鍵。root は codex と同じ規則で、git のコマンドを使わない | 満たす(静的)、codex のソースと突き合わせ済み | `codexHookCheckoutOf`(`codex_hook_trust.go:80`)、`codexHookRootAt`(`:105`)、`codexLinkedWorktreeMain`(`:135`)、`codexWorktreeRegisteredAt`(`:169`)、`codexMainOwnsCommonDir`(`:201`)。本体 4 ファイルとも `os/exec` を import していない。`trust.rs` の `resolve_root` と 1 段ずつ比べた。上にたどる規則(HEAD の無い `.git` ディレクトリを飛ばす)、`.git` のファイルの読み方(通常のファイルで symlink でなく 64 KiB 以下)、gitdir がディレクトリで symlink でないこと、gitdir の実パスの親が `worktrees` であること、登録の `gitdir` と `commondir` の確かめ、main を字面の gitdir の 3 つ上とすること、持ち主の確かめが、同じ順と同じ比べ方だった。`loader/mod.rs` の `find_git_checkout_root` と `discover_project_root`(既定の印は `.git`)も HEAD の無い `.git` を飛ばす。テスト: `TestCodexHookTrust_RootFollowsCodexsWalk`(`codex_hook_trust_test.go:154`)、`…RootOfALinkedWorktreeMadeByGit`(`:314`)、doctor の `…LinkedWorktreeUsesTheMainCheckoutsKeys`(`doctor_codex_hook_trust_test.go:372`) |
| AC9 root の綴りが 2 つなら両方の綴りの承認が要る。片方だけなら両方向で warn と起動の 1 行(層の形で読み直した) | 満たす(静的)。doctor と起動の 1 行の両方 | 綴りは `codexHookSpellings`(`codex_hook_trust.go:278`)。`codexHookLayersOf`(`:358`)が層ごとに綴りの dir を `Dirs` に集め、`classifyCodexHookSlots`(`:636`)が層ごとの `Dirs` で `classifyCodexHookSlot`(`:493`)を呼ぶので、入れ子の層も綴りごとに承認を求める。`…ClassifyNeedsEverySpelling`(`codex_hook_trust_test.go:589`)、`…RootCandidatesOfASymlinkedMainCheckout`(`:393`)、`…OfARelativeDirInASymlinkedMainCheckout`(`:422`)、`TestCodexHookTrustFor_SymlinkedMainCheckoutNeedsBothSpellings`(`:808`)、doctor の `:550` と `:589`(両方向、承認の手順に 2 つの dir)、起動の `TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings`(`codex_hook_trust_notice_test.go:173`) |
| AC6 spawn、start の 2 つの形、director start で未承認・止めてあれば stderr に 1 行が 1 回。claude、すべて承認済み、hooks.json が無いときは出ない。動きと終了コードは変わらない | 満たす(静的) | 呼び出しは `org.go:966`(spawn)、`:1201`(start の task の形)、`:1331`(`--plan`、main の root)、`org_director_headless.go:366`(director start。`7ace1857` で verb を定数 `codexHookTrustDirectorVerb` にした)。どれも driver を解決した直後に 1 回。出すかどうかは `codexHookTrustNotice`(`codex_hook_trust_notice.go:51`)が `Classified()` と件数で決め、claude は `:31` で返る。テストは cycle 1 と同じ `codex_hook_trust_notice_test.go:65`、`:296`、`:325`、`:358`、`:388`、`:414`、`:439` |
| AC7 event 名は codex の 14 種の snake_case、番号は並びのまま、空の command は数えない | 満たす(静的) | `codexHookEventLabels`(`codex_hook_trust_parse.go:29`)と `parseCodexHookSlots`(`:72`)。`f71a2a33` で移しただけ(下の注)。`TestCodexHookTrust_EventLabelsAreCodexs14`(`codex_hook_trust_parse_test.go:15`)、`…SlotsCountPositionsLikeCodex`(`:32`)、`…SlotsOfAFileWithNoHandlers`(`:77`) |
| AC10 root と dir のあいだ(dir を含む)の `.codex/hooks.json` も数え、そのパス(linked worktree では main 側)の鍵で確かめる。root がすべて承認済みでも入れ子が未承認なら warn と起動の 1 行 | 満たす(静的)、codex のソースと突き合わせ済み | 層は `codexHookCheckout.layerRels`(`codex_hook_trust.go:321`)と `layers`(`:340`)。checkout の root から dir までの各 dir のうち、checkout の中に `.codex` ディレクトリを持つものが層になり、hooks.json は `<Repo>/<相対 dir>/.codex/hooks.json` から読んで、そのパスで鍵を引く。codex の `discover_project_layers`(cwd から project root までの各 dir で、`.codex` がディレクトリのものだけ)と `root_checkout_hooks_folder_for_dir`(checkout root と repo root が違えば `<repo root>/<相対 dir>/.codex`)に合う。テスト: `TestCodexHookTrustFor_ReadsEveryLayerDownToTheDirectory`(`codex_hook_trust_test.go:699`。`.codex` の無い dir は層にならず、hooks.json の無い `.codex` は黙り、脇の `.codex` は読まない)、`…LinkedWorktreeReadsNestedLayersFromTheMainCheckout`(`:749`)、`…NestedLayerProblems`(`:784`)。doctor は `TestCheckCodexHookTrust_CountsNestedLayersDownToTheDirectory`(`doctor_codex_hook_trust_test.go:425`。root の 2 つが承認済みで入れ子が未承認のとき warn「1 of 3 hooks in <root> and <sub> is not approved … (sub/.codex/hooks.json:stop:0:0)」)と `…LinkedWorktreeChecksNestedLayersUnderTheMainCheckout`(`:514`)。起動の 1 行は `TestCodexHookTrustNotice_SpawnInASubdirectoryCountsItsNestedHooks`(`codex_hook_trust_notice_test.go:233`。stderr 全体が 1 行と完全一致し、すべて承認すると空) |
| AC11 `{"hooks": null}`、null の group や handler、command の無い command 型の handler は形が違うものとして扱い、Check 3 と違うファイルなら warn | 満たす(静的) | `parseCodexHookSlots` から `parseCodexHookHandler` まで(`codex_hook_trust_parse.go:72-256`)が、`hooks`・event の配列・group・group の `hooks`・handler の null、型の違い、`type` と型ごとの必須の欄の欠け、未知の top-level の鍵、重複を拒む。共有の表 `codexHookTrustInvalidHooks`(`codex_hook_trust_parse_test.go:126`。`{"hooks": null}`、null の event・group・group の hooks・handler、`type` の無い handler、`command` の無い command 型の 7 行)を、全文一致で見る `…SlotsRejectNullAndMissingFieldsLikeCodex`(`:138`、ほかに 23 行)と、doctor の `TestCheckCodexHookTrust_NullHookStructuresAreNeverAPass`(`doctor_codex_hook_trust_test.go:398`。linked worktree からは warn「which is not valid (…), so no hook runs in a codex seat」、main からは info)が使う。起動の側は D5 のとおり何も出さない(`TestCodexHookTrustNotice_PrintsNothingForAHooksJSONCodexRefuses`、`codex_hook_trust_notice_test.go:276`)。serde の型との一致は self-review が `hook_config.rs` と `protocol/src/hooks.rs` で確かめたもので、この verify では読み直していない |
| AC12 bare repo の linked worktree で、gitdir から求めた `<main>` の `.git` が common dir と別物なら worktree 自身を root にする | 満たす(静的)、codex のソースと突き合わせ済み | `codexMainOwnsCommonDir`(`codex_hook_trust.go:201`)は、`<main>/.git`(ディレクトリ、または gitdir のファイルが指す先)の実パスが common dir と同じときだけ `<main>` を採る。`trust.rs` の 199〜223 行目と同じ。`TestCodexHookTrust_RootChecksTheLinkedWorktreesOwnership`(`codex_hook_trust_test.go:242`、13 行。「a bare repository whose parent's .git is another repository」は worktree 自身が root、`--separate-git-dir` の形の `.git` のファイルが common dir を指すときだけ main が root)と、git で作る `…RootOfABareRepositorysLinkedWorktree`(`:340`。`outer/cache.git` の worktree で、層は worktree 自身の hooks.json 1 つ、鍵もそのパス)。計画の D7 (c) は持ち主の確かめだけを書くが、実装は登録の確かめ(登録の gitdir が checkout を指し返す、commondir が common dir)もする。codex と同じで、AC12 と食い違わない(I8) |
| AC8 Check 3 の pass の文が新しい Check を指し、文書が説明する | 満たす | `doctor.go:469`。`TestCheckCodexEffectiveConfig_PassPointsAtCodexHookTrust`(`doctor_codex_hook_trust_test.go:618`)、`TestRunDoctor_CodexHookTrustFollowsCodexEffectiveConfig`(`:637`)。`.codex/README.md` の 2 写しは Trust UX の 5 番に入れ子の層の文を足した(`9e72d49a`、`cmp` で同じ)。`--help` の段落(`codex_hook_trust_notice.go:103-115`)は「in the checkout's root and in each directory below it down to that one」で層を言い、3 つの動詞の `Long` の末尾にあることを `TestCodexHookTrustHelp_EachVerbDescribesTheNote`(`codex_hook_trust_notice_test.go:533`)が見る。文書は下の drift の表 |

issue #310 の受け入れ条件 3 つは、引き続き AC1、AC2、AC6 に入っている。

`f71a2a33` が移すだけであることも確かめた。`9e72d49a` の 2 ファイル(`codex_hook_trust.go` と `_test.go`)と `f71a2a33` の 4 ファイルのトップレベルの宣言の見出し 108 行は、並べ替えると一致した。コメントを除いて行の多重集合を比べると、差は import の 3 行(`"errors"`、`"fmt"`、`"testing"`)だけだった。宣言の本文まで doc コメントつきで比べたのは self-review(107 件、差なし)。

### Static analysis(cycle 2)

HEAD `c0337338` で取り直した。

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | 既定の changed scope。`.codex/config.toml` が言語に分類されないので、全体の fallback で golang の pack を選んだ。2026-11-12T15:40:38Z。script 自身の log は `docs/evidence/verify-2026-11-12-154038.log` |
| ・ shellcheck、hook の `sh -n` 20 本、guard の awk 2 つ、settings.json の `jq -e` 2 つ | ok | |
| ・ Codex hook single-source guard、inline hook detector の smoke test、PR provenance policy guard | ok | |
| ・ `scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| ・ check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt README の plan の参照 | ok | |
| ・ golang: `gofmt -l`、`go vet ./...`、`golangci-lint run ./...`、`staticcheck ./...` | ok | 「gofmt: ok」と「0 issues.」。go vet と staticcheck は出力なし。「Skipping」の行は無い |
| ・ `secret-scan-branch.sh` | clean | 0936aaea..c0337338 |
| `git diff origin/main...HEAD --check` | clean | |
| 変わった 29 ファイルの U+FFFD の検査 | 0 件 | |
| `./scripts/plan-visual.sh digest`(計画) | `e3164744a981` | `- Approved:` の値と一致 |

### Documentation drift(cycle 2)

cycle 1 の所見の今の状態:

| ID | 状態 | HEAD での根拠 |
| --- | --- | --- |
| V1 | 解決(`23a57869`) | `.codex/config.toml` の 2 写しの注釈が、Codex hook trust の check が承認の鍵を読み hash は見ないと言う。HEAD でも正しい(2 写しは `cmp` で同じ) |
| V2 | 解決(`23a57869`) | 135、153、195、201、371、415 行目に持ち越しの記録が入った。`git log 75f3757d..HEAD --no-merges` で見ると、cycle 2 の branch のコミットが触れたのは `org_director_headless.go` の 1 行だけで、`org.go`、`doctor.go`、`doctor_codex_writable_root.go`、recipe は変えていない。記録の数字は今も正しい |
| V3 | 解決(`23a57869`) | 新しい 3 行。origin/main の取り込みで 476〜478 行目に動いた(I8) |
| I1 | 残る(任意) | `--help` の「There is no note …」は、設定のパスが決まらないとき(`codex_hook_trust_notice.go:34-37`)を挙げない。挙げている場合は層の形に合わせて書き直されている |
| I2 | 残る(問題なし) | `/org` skill は `codex trust .` の話だけで、今も正しい |
| I3 | 残る | 起動の 1 行の後半が D6 の文と違う。AC のチェックボックスは AC1〜AC12 すべて `[ ]` |
| I4 | 一部解決 | Progress に S6 と cycle 2 の S7、S8 が入った。`CODEX_HOME` が空のときのテストは今も無い(V5) |

cycle 2 の確かめ:

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.codex/README.md`(root と雛形、`cmp` で同じ)の Trust UX の 5 番 | はい | root から codex が動く dir までの各 subdirectory の `.codex/hooks.json` を読むこと、それぞれのパスで鍵を引くこと、linked worktree では main 側の同じ相対 dir になること、ralph もそれを確かめることを言う。codex の `discover_project_layers` と `root_checkout_hooks_folder_for_dir` に合う |
| `spawn`・`start`・`director start` の `--help` の段落 | はい | 層、linked worktree では main 側、1 行が出ない場合を言う(I1 を除く) |
| `docs/recipes/codex-setup.md`(root と雛形、`cmp` で同じ) | はい | 層には触れないが、鍵を見て hash を見ないことだけを言うので、層の形と矛盾しない |
| README の Quick start、`ralph org …` の行、`ralph doctor` の行。`.codex/README.md` の doctor の段落 | ほぼ(I5) | 「the `.codex/hooks.json` codex reads」と 1 つのファイルの形で書く |
| `internal/cli/org.go:1328-1330` の注釈 | いいえ(V4) | 無くなった関数を名指す |
| `docs/tech-debt/README.md` 478 行目「Test gaps of feat/codex-hook-trust-check」 | いいえ(V5) | この PR の cycle 2 が自分の行の Trigger を引いた |
| `docs/tech-debt/README.md` 477 行目「Where the Codex hook trust check … reads differently from codex」 | いいえ(V6) | cycle 2 で増えた差が無い |
| codex と読み方が違う点で、register にも注釈にも無いもの | INFO(I6) | 既定の設定と ralph の座席の cwd では起きない 2 つ |
| linked worktree の入れ子の層で、main に hooks.json が無いとき | INFO(I7) | 計画どおりだが、root の層と扱いが違う |
| 計画の Progress と D7 (c) | INFO(I8) | 行番号の古さ、`7ace1857` の記録が無い、D7 (c) より実装が広い |

- **V4(LOW)**: `internal/cli/org.go:1328-1330` の注釈「a codex leader in it reads the main checkout's hooks (codexHookRoot)」が、`9e72d49a` で無くなった関数 `codexHookRoot` を名指す。`git grep -w codexHookRoot -- 'internal/**/*.go'` の hit はこの 1 件だけ。今の名前は `codexHookCheckoutOf`(root を決めるのは `codexHookRootAt`)。注釈の言う理由(feature worktree の root で動く座席は main の hooks.json を読む)は今も正しい。注釈は S3(`56c558dd`)で書かれ、関数を消した `9e72d49a` は org.go を変えていない。Go の注釈なので、/sync-docs で直すか、次にこの関数を触るときに直すかは lead が決める
- **V5(LOW)**: tech-debt 478 行目の Trigger は「The next change to `codex_hook_trust.go`, `doctor_codex_hook_trust.go`, or `codex_hook_trust_notice.go`: add (a) first (the three cases of the probe) and the two `.git` cases of (b)」。cycle 2 の `9e72d49a` と `7ace1857` が 3 ファイルすべてを変えたが、(a) の `CODEX_HOME` が空のテストも、(b) の `.git` の 2 つ(通常のファイルでない、64 KiB 超)のテストも足されていない(テストのファイルを grep して 0 件)。行には持ち越しの記録も無い。あわせて (b) の文が古い。「neither a syntax nor a type error」の type error の分岐は `9e72d49a` で消えた(self-review C2-1 (1))。今の `codexHooksJSONError`(`codex_hook_trust_parse.go:285`)は、syntax error でない失敗を位置なしの `syntax error` にする。`9e72d49a` で増えた登録の `gitdir` と `commondir` の読み(`codexWorktreeRegisteredAt`)も同じ `codexReadGitMetadata` を通るが、(b) に入っていない。(d) は、この verify で codex のソースの `resolve_root` と 1 段ずつ比べたので、「実機の codex とは比べていない」に書き直せる。/sync-docs で、持ち越しの記録を足すかテストを足すかを決める
- **V6(LOW)**: 477 行目は (a) と (b) だけを挙げ、`9e72d49a` で入った差を持たない。serde は struct を JSON の配列からも読むので、codex は `{"hooks": []}` のような形を読み込むが、`parseCodexHookSlots` はそれを拒む。doctor は invalid(info か warn)と言い、起動の 1 行も出ない。この差はパーサーの注釈(`codex_hook_trust_parse.go:69-71`)と計画の Progress(S7)にしかない(self-review C2-1 (2))。向きは知らせる側。/sync-docs で (c) として足す
- **I5**: README の Quick start(65〜70 行目)と doctor の行、`.codex/README.md` の doctor の段落は、doctor が読むものを「the `.codex/hooks.json` codex reads」と 1 つのファイルの形で書く。入れ子の層は `.codex/README.md` の Trust UX の 5 番と `--help` にある。誤りではなく、要約が層を言わないだけ
- **I6**: codex のソースを読んで見つけた、読み方の差 2 つ。register にも注釈にも無い。(1) codex の層の上端は `project_root_markers`(既定は `[".git"]`、`project_root_markers.rs:5`)で決まり、`discover_project_layers` は cwd からその root までの dir を層にする。利用者が codex の設定でこの印を変え、印が checkout root より上にあると、codex はその上の `.codex` も層として読むが、ralph は checkout root から下だけを見るので、その hook の承認を確かめない。(2) 座席の cwd が別の checkout の中への symlink のとき、`codexHookLayersOf`(`codex_hook_trust.go:358`)は 2 つの綴りの層を `Rel` でまとめ、`Dirs[0]`(字面の綴り)の hooks.json だけを読む。実パスの側の checkout の hooks.json は読まず、その鍵だけを字面の側の slot で引く。どちらも既定の設定と ralph の座席の cwd(worktree の root、または main の root)では起きない。477 行目に足すかは任意
- **I7**: linked worktree の入れ子の層で、worktree に `sub/.codex/hooks.json` があって main に無いとき、codex はその hook を動かさず(main 側を読む)、ralph も黙る。`HooksProblems`(`codex_hook_trust.go:532-541`)が入れ子の「無い」を問題にしないため。root の層の同じ形は warn にする(AC4、計画助言の指摘 2)。D7 (a) は入れ子の扱いを「読めない・形が違う」に限っているので計画どおりで、AC10 にも当たる。branch で入れ子の hooks.json を足した worktree の座席に知らせが出ない、という非対称が残る
- **I8**: 計画の Progress の sync-docs の記録は新しい tech-debt の行を「471〜473」と書くが、origin/main の取り込みで 476〜478 に動いた。Progress の「Cycle 2 の実装」に `7ace1857` の記録が無い。D7 (c) は持ち主の確かめだけを書くが、実装は codex と同じく登録の確かめもする。AC のチェックボックスはすべて `[ ]`。計画の本文は digest の範囲なので、/pr か archive のときに扱えばよい

#### Trigger の掃き出し(cycle 2)

cycle 1 の sync-docs(`23a57869`)から HEAD までに branch のコミットが触れたファイルと関数の名前(`codex_hook_trust*`、`doctor_codex_hook_trust*`、`org_director_headless.go`、`startDirector`、`.codex/README.md`、`resume_screens`、テストの helper の `newLegacyLedgerRepo`、`setupOrgStubPATH`、`runLedgerTestGit`、`isolateFeatureGitEnv`、`directorStartLedger`)で、tech-debt の各行の Trigger の欄(python で 4 列目を切り出した)を引いた。この PR が新しく引いたのは 478 行目だけだった(V5)。ほかの hit は次の理由で当たらない。

- 275 行目(`startDirector` の拒否の順): `7ace1857` は director start の呼び出しの verb を定数にした 1 行で、順は変わっていない
- 466 行目(`printScaffoldVersionNotice` の root 引数と `orgScaffoldRoot`): 変えていない
- 477 行目 (b) の「the next change to `readCodexHookStates`」: 関数の本文は `75f3757d` と HEAD で同じ
- 476 行目(`resume_screens.go` と watch の ALERT): 変えていない
- 398 行目 (b) の `setupOrgStubPATH`: 呼ぶだけで変えていない
- 135、201、371、415 行目: cycle 2 の branch のコミットは対象のファイルを変えていない(V2 の行)

行数: `codex_hook_trust_test.go` は 908 行で 800 行の目安を超える(self-review C2-8)。大きさを追う register の行は無い。diff の質の判断なので所見にはしない。

### Observational checks(cycle 2)

binary は作らず、流していない。代わりに codex `rust-v0.166.0` のソース 3 つを読み、AC5、AC10、AC12 の規則と突き合わせた(上の表)。実機の codex 0.166.0 の `hooks/list` と比べた記録(`docs/evidence/codex-hook-trust-worktree-2026-11-12.md` の 4 節)は 0d813954 の binary のもので、入れ子の層、null の形、持ち主の確かめは実機で流していない。evidence の結論の「main worktree の root の `.codex/hooks.json` を読み」は cycle 1 の時点の記録で、層の形は書いていない。

### Coverage gaps(cycle 2)

- テストは走らせていない(`/test` の範囲)。cycle 2 の 14 本のうち、`…RootChecksTheLinkedWorktreesOwnership` の「the gitdir a symlink」の行は symlink を作れない環境で Skip する
- AC10〜AC12 は codex のソースと突き合わせたが、実機の codex とは比べていない。AC11 の serde の型は、この verify では読み直していない
- I6 と I7 の形を見るテストは無い
- `CODEX_HOME` が空のときと、`.git` の metadata の 2 つの分岐のテストは無い(V5)
- 確度をいちばん上げる追加の確かめ: `/test` で `go test ./internal/cli -run 'CodexHookTrust' -v` を流し、cycle 2 の 14 本が SKIP でなく PASS になったことを出力で見る。その次は、codex 0.166.0 の app-server の `hooks/list` を、入れ子の `.codex/hooks.json` を持つ dir と `outer/cache.git` の worktree で引き、doctor の判定と比べること(evidence の 4 節と同じやり方)

## Verdict

- Verified: AC1〜AC12 はコードとテストのソースから満たす。cycle 2 で足された AC10(入れ子の層)、AC11(null と欠けた欄)、AC12(bare repo の持ち主)は、doctor と起動の 1 行の両方にテストがある(AC11 の起動の側は、D5 のとおり出さないことを固定)。AC4、AC5、AC9 は層の一覧になった報告で読み直し、cycle 1 と同じ期待のテストが残っている。AC5、AC10、AC12 の root と層の規則は、codex `rust-v0.166.0` の `trust.rs` と `loader/mod.rs` と 1 段ずつ合った。`f71a2a33` は移すだけだった。issue #310 の受け入れ条件 3 つは AC1、AC2、AC6 に入っている。HEAD `c0337338` の静的解析はすべて通った。`.codex/README.md` と `docs/recipes/codex-setup.md` の 2 つずつの写しと、3 つの `--help` は層の形の動きと合っている
- Partially verified: AC11 の serde との一致は self-review の読みに依る。AC10〜AC12 は実機の codex とは比べておらず、比べたのはソースだけ
- Not verified: テストの実行結果。実機の herdr での起動の 1 行。`CODEX_HOME` が空のとき。I6 と I7 の形
- Drift(/sync-docs へ): V4 `org.go:1328-1330` の注釈の古い関数名(Go の注釈なので扱いは lead が決める)。V5 tech-debt 478 行目の Trigger をこの PR が引いたのに、テストも持ち越しの記録も無く、(b) の文も古い。V6 477 行目に配列の形の差が無い。I5〜I8 は任意。cycle 1 の I1 と I3 は残る
- Verdict: pass

### Cycle 1(HEAD 75f3757d)

- Verified: AC1〜AC9 はコードとテストのソースから満たす。AC9 は doctor の側を 3 段のテストで、起動の 1 行の側を `TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings` の 6 行で固定した。AC6 の claude は spawn、start、start --plan、director start の 4 つすべてにテストがある。AC4 の読めない main の hooks.json と、AC5 の bare の repo の linked worktree の root にもテストが入った。AC1〜AC4 は dddb946d の binary でも観察し、その後本番のコードは変わっていない。issue #310 の受け入れ条件 3 つは AC1、AC2、AC6 に入っている。HEAD `75f3757d` の静的解析はすべて通った。README、`.codex/README.md` と `docs/recipes/codex-setup.md` の 2 つずつの写し、3 つの `--help` は振る舞いと合っている
- Partially verified: codex の hash の範囲についての `.codex/README.md` の文は、self-review のソースの読みに依っている。bare の repo の root は ralph の側だけを固定し、codex が同じ root を出すことは確かめていない
- Not verified: テストの実行結果(Skip になる行を含む)。実機の herdr での spawn・start・director start の 1 行。`CODEX_HOME` が空のとき
- Drift(/sync-docs へ): V1 `.codex/config.toml` の注釈(2 写し)。V2 tech-debt の Trigger 4 件(135、201、371、415 行目)。V3 計画の Scope が決めた tech-debt の 2 行。I1〜I4 は任意
- Verdict: pass

### 最初の版(HEAD dddb946d、9492dc5a でコミット)

- Verified: AC1〜AC8 はコードとテストのソースから満たす。AC1〜AC4 は HEAD の binary でも観察した。AC9 は doctor の側を 3 段のテストで満たし、起動の 1 行の側は同じ関数を通ることをコードで確かめた。issue #310 の受け入れ条件 3 つは AC1、AC2、AC6 に入っている。静的解析はすべて通った。README、`.codex/README.md` と `docs/recipes/codex-setup.md` の 2 つずつの写し、3 つの `--help` は振る舞いと合っている
- Partially verified: AC9 の起動の 1 行(直接のテストが無い)。codex の hash の範囲についての `.codex/README.md` の文は、self-review のソースの読みに依っている
- Not verified: テストの実行結果。実機の herdr での spawn・start・director start の 1 行。bare の repo の linked worktree での root
- Drift(/sync-docs へ): V1 `.codex/config.toml` の注釈(2 写し)。V2 tech-debt の Trigger 4 件(135、201、371、415 行目)。V3 計画の Scope が決めた tech-debt の 2 行。I1〜I3 は任意
- Verdict: pass
