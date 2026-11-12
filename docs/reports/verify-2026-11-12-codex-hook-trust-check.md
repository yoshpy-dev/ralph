# Verify report: codex-hook-trust-check

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(issue #310。`- Approved:` の digest `147d7ec0e7ae` は `./scripts/plan-visual.sh digest` で出し直した値と一致した。HEAD `75f3757d` でも同じ値)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)。この版は、HEAD `dddb946d` で書いて `9492dc5a` でコミットした最初の版に、そのあとのコミットを足した更新
- Scope: `git diff origin/main...HEAD`(HEAD `75f3757d`。origin/main `d2e5c6e9` は f36cfc52 で取り込み済み。21 ファイル、+3169 / -77)。最初の版の範囲は HEAD `dddb946d`(20 ファイル、+2796 / -77)。そのあと `9492dc5a` がこの report を足し、`1f7ccadb` がテスト 3 ファイルだけを足し(+231 / -0)、`75f3757d` が self-review の report を更新した。本番のコードは `dddb946d` から変わっていない(`git diff --stat 9492dc5a..HEAD -- '*.go' ':!*_test.go'` が空)。静的解析は `./scripts/run-static-verify.sh` の既定(変わった言語だけで、今回は golang)を HEAD `75f3757d` で取り直した。テストは走らせていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-11-12-codex-hook-trust-check.log`(git が無視する場所)。最初の版の出力(run-static-verify.sh の全出力と下の観察の probe の出力)を先頭にそのまま残し、この更新の出力を末尾の「Update at HEAD 75f3757d」の節に足した(前の 12417 byte が新しい log の先頭と一致することを `head -c | cmp` で確かめた)
- Insight event: 最初の版で `docs/insights/events/2026-11-12-codex-hook-trust-check.jsonl` に verify の 1 行(cycle 1、pass、low 3)を足し、その版の report と同じコミットに入れた。この更新では足していない。cycle は同じ 1 で、判定も LOW の数(V1〜V3)も変わらず、2 行目を足すと同じ cycle の verify が 2 回に数えられる。self-review の更新(`75f3757d`)も同じ理由で足していない
- ID の付け方: `V1` から `V3`(LOW、/sync-docs に渡す drift)、`I1` から `I4`(INFO。I4 はこの更新で足した)

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

## Verdict

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
