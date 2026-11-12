# Verify report: codex-hook-trust-check

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(issue #310。`- Approved:` の digest `147d7ec0e7ae` は `./scripts/plan-visual.sh digest` で出し直した値と一致した)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `dddb946d`。origin/main `d2e5c6e9` は f36cfc52 で取り込み済み。20 ファイル、+2796 / -77)。静的解析は `./scripts/run-static-verify.sh` の既定(変わった言語だけで、今回は golang)。テストは走らせていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-11-12-codex-hook-trust-check.log`(git が無視する場所。run-static-verify.sh の全出力と、下の観察の probe の出力)
- Insight event: `docs/insights/events/2026-11-12-codex-hook-trust-check.jsonl` に verify の 1 行(cycle 1、pass、low 3)を足し、この report と同じコミットに入れた
- ID の付け方: `V1` から `V3`(LOW、/sync-docs に渡す drift)、`I1` から `I3`(INFO)

## Spec compliance

受け入れ条件ごとに、コードと、それを固定するテストのソースを読んで判定した。テストが通ることは `/test` が確かめる。「満たす(静的)」は、コードがそう動き、その期待値を持つテストがあるという意味で使う。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 driver_pool に codex があり鍵が無い repo で warn、未承認の数と承認の手順(issue の受け入れ条件 1) | 満たす(静的)+ 観察 | `checkCodexHookTrust`(`internal/cli/doctor_codex_hook_trust.go:46`)の outcome 6 と 7。文は `codexHookTrustGapDetail`(`:150`)と `codexHookTrustApprovalStep`(`:184`)が作る。`TestCheckCodexHookTrust_WarnsNamingTheHookNotApproved`(`doctor_codex_hook_trust_test.go:238`)が「1 of 2 hooks … is not approved … (pre_tool_use:0:0)」と「open codex in <repo> and trust the hooks (Hooks need review → Trust all and continue)」を持つ。`TestCheckCodexHookTrust_NamesAtMostThreeHooks`(`:296`)が鍵の末尾を 3 つで切ることを見る。観察: HEAD の binary の (a) |
| AC2 すべて `trusted_hash` つきなら pass、hash を確かめていないことと Hooks need review を書く(issue の受け入れ条件 2) | 満たす(静的)+ 観察 | outcome 8(`doctor_codex_hook_trust.go:80-82`)。`TestCheckCodexHookTrust_PassWhenEveryHookIsApproved`(`:312`)が「keys only; the hash is not checked」と「makes codex show Hooks need review in the seat's pane」を持つ。観察: HEAD の binary の (b)。実機の codex 0.166.0 の `hooks/list` と食い違わないことは `docs/evidence/codex-hook-trust-worktree-2026-11-12.md` の 4 節(binary は 0d813954。check のコード 6 ファイルは 0d813954 から HEAD まで変わっていない) |
| AC3 `enabled = false`、`trusted_hash` の無い鍵、設定ファイルが無いときは warn。読めない・decode できない設定は warn で中身を出さない | 満たす(静的)+ 観察 | 分け方は `codexHookTrustUnder`(`codex_hook_trust.go:374`)。`TestCodexHookTrust_ClassifyUnderOneSpelling`(`codex_hook_trust_test.go:465`)が「no trusted_hash」を NotApproved、「turned off」の 2 行を Off にする。doctor では `TestCheckCodexHookTrust_WarnsWhenAHookIsTurnedOff`(`doctor_codex_hook_trust_test.go:250`)、`…WarnsWhenTheCodexConfigIsMissing`(`:201`)、`…WarnsWhenTheCodexConfigIsUnreadable`(`:213`)、`…WarnsWithOnlyThePositionOfAnUndecodableConfig`(`:225`、設定に仕込んだ印と `trusted_hash` と hash の値が文に出ない)。decode の失敗は `codexConfigDecodeErrorOf`(`codex_hook_trust.go:349`)が行と桁だけを持つ。観察: (c) |
| AC4 codex が無ければ pass「not needed」で設定を読まない。hooks.json が無い・読めないとき、Check 3 と同じファイルなら info、違うファイルなら warn | 満たす(静的)+ 観察 | outcome 1 は `resolveEnv` を呼ぶ前に返る(`doctor_codex_hook_trust.go:48-52`)。`TestCheckCodexHookTrust_NotNeededWithoutCodexReadsNothing`(`:91`)が、壊れた入力でも pass で、`resolveEnv` が呼ばれないことを見る。info と warn の分け方は `codexHookTrustHooksProblem`(`:90`)と `codexHookTrustSameDir`(`:113`)。`…InfoWhenItsOwnHooksJSONCannotBeRead`(`:121`、無い・ディレクトリ・JSON でない、絶対パスと `.`)、`…WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksCannotBeRead`(`:160`、無い・形が違う)。観察: (d) |
| AC5 linked worktree からでも main の hooks.json とそのパスの鍵で確かめる。root は codex と同じ規則で、git のコマンドを使わない | 満たす(静的) | `codexHookRoot`(`codex_hook_trust.go:163`)、`codexHookRootAt`(`:188`)、`codexLinkedWorktreeMain`(`:216`)。3 ファイルとも `os/exec` を import していない(grep で 0 件)。`TestCodexHookTrust_RootFollowsCodexsWalk`(`codex_hook_trust_test.go:194`、HEAD の無い `.git`、`worktrees` の下でない gitdir、`.git` の無い main、相対の gitdir、gitdir の行が無い、git の外の 10 行)、`…RootOfALinkedWorktreeMadeByGit`(`:260`)、`TestCheckCodexHookTrust_LinkedWorktreeUsesTheMainCheckoutsKeys`(`doctor_codex_hook_trust_test.go:327`、worktree のパスの鍵だけでは warn) |
| AC9 root の綴りが 2 つなら、両方の綴りの承認があるときだけ承認済み。片方だけなら両方向で warn(と起動の 1 行) | doctor は満たす(静的)。起動の 1 行は同じ道を通るとコードで確かめた | `codexHookRootCandidates`(`codex_hook_trust.go:257`)と `classifyCodexHookSlot`(`:392`)。`TestCodexHookTrust_ClassifyNeedsEverySpelling`(`codex_hook_trust_test.go:491`)、`TestCodexHookTrustFor_SymlinkedMainCheckoutNeedsBothSpellings`(`:590`)、doctor の `…SymlinkedMainCheckoutNeedsBothSpellings`(`doctor_codex_hook_trust_test.go:352`)と `…FromDotNeedsBothSpellings`(`:391`、S5 の不具合の再現)。起動の 1 行は `codexHookTrustFor` の同じ結果から作る(`codex_hook_trust_notice.go:36`)が、symlink の綴りで 1 行を見るテストは無い(Coverage gaps) |
| AC6 spawn、start の 2 つの形、director start で、未承認・止めてあれば stderr に 1 行が 1 回。claude、すべて承認済み、hooks.json が無いときは出ない。動きと終了コードは変わらない(issue の受け入れ条件 3) | 満たす(静的) | 呼び出しは `internal/cli/org.go:964`(spawn)、`:1199`(start の task の形)、`:1329`(`--plan` の形、main の root)、`org_director_headless.go:366`(director start、director が動く directory)。どれも driver を解決した直後に 1 回。`TestCodexHookTrustNotice_SpawnPrintsOneLineForACodexSeat`(`codex_hook_trust_notice_test.go:65`)は stderr 全体を完全一致で見る 11 行(設定なし、承認なし、1 つ承認、相対の `--cwd`、止めてある、両方、すべて承認、claude、hooks.json なし、読めない設定、decode できない設定)で、どれも exit 0 と dry-run の spawned を見る。start は `:169`、`--plan` は `:199`、director start は `:232`。claude での早い return は `codex_hook_trust_notice.go:29` |
| AC7 鍵の event 名は codex の 14 種の snake_case、番号は並びのまま、空の command は数えない | 満たす(静的) | `codexHookEventLabels`(`codex_hook_trust.go:31`)と `parseCodexHookSlots`(`:102`)。`TestCodexHookTrust_EventLabelsAreCodexs14`(`codex_hook_trust_test.go:94`)、`…SlotsCountPositionsLikeCodex`(`:111`、空白だけの command の次の handler が位置 1 を保つ、空の command、知らない event、空の group) |
| AC8 Check 3 の pass の文が新しい Check を指す。README、`.codex/README.md` の 2 写し、`docs/recipes/codex-setup.md` が説明する | 満たす | `doctor.go:469`。`TestCheckCodexEffectiveConfig_PassPointsAtCodexHookTrust`(`doctor_codex_hook_trust_test.go:420`、完全一致)と、doctor の出力で 2 行が並ぶことを見る `TestRunDoctor_CodexHookTrustFollowsCodexEffectiveConfig`(`:439`)。文書は下の drift の表。`--help` の段落は `TestCodexHookTrustHelp_EachVerbDescribesTheNote`(`codex_hook_trust_notice_test.go:329`)と、HEAD の binary の 3 つの `--help` |

issue #310 の受け入れ条件 3 つは、1 が AC1、2 が AC2、3 が AC6 に入っている。issue の「やること」の hash の突き合わせと「断る設定」は、計画の Non-goals(ユーザーの決定)が外しており、AC2 の pass の文と README、`.codex/README.md`、recipe がそのことを書いている。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | 既定の changed scope。言語は golang |
| ・ shellcheck、hook の `sh -n` 20 本、guard の awk、settings.json の `jq -e` | ok | |
| ・ `scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0 |
| ・ `scripts/check-pipeline-sync.sh`、`scripts/check-skill-sync.sh`、`scripts/check-template-purity.sh` | ok | skill は 14 本がそろう |
| ・ tech-debt README の plan の参照 | ok | |
| ・ golang: `gofmt -l`、`go vet ./...`、`golangci-lint run ./...`、`staticcheck ./...` | ok | gofmt ok、golangci-lint 0 issues、go vet と staticcheck は出力なし |
| ・ `secret-scan-branch.sh` | clean | d2e5c6e9..dddb946d |
| `git diff origin/main...HEAD --check` | clean | |
| 変わった 20 ファイルの U+FFFD の検査 | 0 件 | |
| `./scripts/plan-visual.sh digest`(計画) | `147d7ec0e7ae` | `- Approved:` の値と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の Quick start(65〜70 行目) | はい | doctor が、codex が `[org].driver_pool` にあるとき未承認と `enabled = false` を知らせ、鍵だけを見ることを言う |
| README の `ralph org …` の行と `ralph doctor` の行 | はい | org の行は spawn・start(2 つの形)・director start の 1 行と、続けて動くことを言う。doctor の行は、読む hooks.json(linked worktree からは main のもの)、鍵が無い・止めてある・設定が無いときの warn、hash を見ないことを言う |
| `.codex/README.md`(root と雛形。`cmp` で同じ) | はい | One-time setup の 3 番(29〜36 行目)、Trust UX の 2 番(鍵の形)、4 番(宣言全体の hash)、5 番(linked worktree と symlink)、doctor の段落。鍵の形は `codexHookKey` と合う。ダイアログの文言は `docs/evidence/org-seat-resume-probe-2026-11-07.md:119-126` にある。`ralph org resume` がそのダイアログを選択の画面として扱い何も打たないことは `internal/org/resume_screens.go:64` と `:99-100` が言う。hash が宣言全体にかかることは self-review が `rust-v0.166.0` のソースで確かめたもので、ここでは読み直していない |
| `docs/recipes/codex-setup.md`(root と雛形。`cmp` で同じ) | はい | doctor の段落が新しい Check を説明する。古い「per-command-hash」の文は消えた |
| `ralph org spawn`・`start`・`director start` の `--help` | はい | HEAD から build した binary で 3 つとも段落を 1 回持つ |
| Check 3 の pass の文と `checkCodexEffectiveConfig` の注釈 | はい | `doctor.go:379-385`、`:469` |
| `.codex/AGENTS.override.md:56`、`.codex/hooks/README.md:35-36`、README の 197 行目と 304 行目 | はい | 対話での承認が要ることを言うだけで、doctor が確かめられないとは言っていない |
| `docs/recipes/codex-seat-permissions.md` | はい | 計画の Verify plan が挙げた。hook の承認に触れる記述は無い |
| `.codex/config.toml` の 109〜114 行目の注釈(root と雛形、`cmp` で同じ) | いいえ(V1) | 「`ralph doctor` verifies the wiring but cannot probe trust state」と言う。文脈は hook の対話での承認で、今の doctor は「Codex hook trust」でその鍵を見る |
| `docs/tech-debt/README.md` の Trigger | いいえ(V2) | この diff が満たした Trigger 4 件に、持ち越しの記録も支払いも無い(下の内訳) |
| `docs/tech-debt/README.md` の新しい行 | いいえ(V3) | 計画の Scope の最後の項は「Hooks need review」で知らせる案を tech-debt に書くと決め、S1 の記録は数えすぎ(self-review L5)を tech-debt と Known gaps に書くと言う。どちらの行もまだ無い。self-review の Tech debt identified の 2 行がそのまま使える |
| `--help` の段落の「There is no note …」(`codex_hook_trust_notice.go:112-114`) | ほぼ(I1) | 1 行が出ない場合として「hooks.json が無い・不正」「設定が読めない・decode できない」を挙げる。コードは hooks.json が読めないとき(ディレクトリ、1 MiB 超)と、設定のパスが決まらないときも出さない。列挙を「これだけ」とは言っていないので誤りではない |
| `/org` skill の 121〜125 行目(4 つの写し) | はい(I2) | `codex trust .`(project の信頼)の話で、今も正しい。hook の承認と新しい 1 行には触れない。#313 の版の知らせも skill には書いていないので、それと同じ扱いになる |
| 計画の D6 の文と AC のチェックボックス | INFO(I3) | 実装の 1 行の後半は「run 'ralph doctor' for the keys and how to trust them」で、D6 の形と違う。S3 の進捗の記録がこの違いを書いている。D6 は digest の範囲なので直さないのが今の扱い。AC1〜AC9 は `[ ]` のまま |

### V2 の内訳(この diff が満たした tech-debt の Trigger)

`/sync-docs` が、各行に持ち越しの記録を足すか、支払うかを決める。行番号は HEAD の `docs/tech-debt/README.md`。行数は origin/main d2e5c6e9 と HEAD dddb946d で `wc -l` を取った。

1. 135 行目「Next touch to `internal/cli/doctor_codex_writable_root.go` for another reason」: `readCodexUserConfig` から `readCodexFileBounded` を切り出し、`codexConfigMaxBytes` の注釈に 1 文を足した。991 行から 1009 行。`pathCovers`、`codexWritableRootSuffix`、読む設定は変わっていない。2026-10-30、11-01、11-05 の持ち越しの記録と同じ形になる。
2. 201 行目の (e)・(f)「the next change to `runOrgStartPlan`, `StartFeature`, or `scriptFeatureWorktrees`」: `runOrgStartPlan`(`internal/cli/org.go:1326-1329`)に `printCodexHookTrustNotice` の呼び出しと 3 行の注釈、その doc に 3 行を足した。新しい呼び出しは `TestCodexHookTrustNotice_StartPlanChecksTheMainCheckout` が走らせる。#313 の 2026-11-12 の記録と同じ形になる。
3. 371 行目「The next edit of any of these five recipes」: `codex-setup.md` の 2 つの写しの doctor の段落を書き直した。`docs/evidence/` の引用は origin/main と HEAD で同じ 1 件(`codex-config-rewrite-2026-09-30.md`)、`docs/tech-debt/README.md` の引用は 0 件で、増えも書き換えもしていない。
4. 415 行目「The next change to any of the five」: `internal/cli/doctor.go` が 1615 行から 1624 行になった(Check 3b の呼び出しと 6 行の注釈、Check 3 の doc と pass の文)。

満たしていないと確かめたもの: 144 行目(`doctor_codex_writable_root_test.go` は変えておらず、target directory の決め方も同じ)、275 行目(`startDirector` に足したのは `ResolveRoleModel` と cwd の決定のあとの 1 行の表示で、拒否の順は変わっていない)、466 行目と 467 行目(`printScaffoldVersionNotice` の引数も `orgScaffoldVersionNoticeHelp` も変えていない)、427 行目の (6)(status の JSON の変更は origin/main の取り込みで入ったもので、この branch のコミットではない)、88 行目、110 行目、223 行目、425 行目(Trigger が名指すファイルや関数を変えていない)。

行数の観察: 新しいファイルはどれも 800 行より短い(最長は `codex_hook_trust_test.go` の 686 行)。すでに 800 行を超えていた `doctor_codex_writable_root.go`(991 → 1009)と `internal/cli/org.go`(2261 → 2273)が少し伸びた。どちらも大きさを追う行は register に無い。diff の質の判断なので所見にはしない。

## Observational checks

HEAD(dddb946d)から binary を build し、scratch に git の repo を作って `templates/base/.codex/hooks.json` を置き、`ralph doctor` を打った。`env -i` で `HOME` と `CODEX_HOME` を scratch に、`PATH` を `<scratch>/stub:/usr/bin:/bin` に絞った。`codex` と `claude` は版を表示するだけの stub で、本物の codex も利用者の `~/.codex` も使っていない。

- (a) 設定なし、既定の driver_pool: warn「no codex user config at ~/.codex/config.toml, so none of the 4 hooks … is approved」と承認の手順
- (b) 4 つの鍵に `trusted_hash`: pass「4 hooks … approved … (keys only; the hash is not checked …)」
- (c) `pre_tool_use:0:0` に `enabled = false`: warn「1 of 4 hooks … is turned off (enabled = false) … (pre_tool_use:0:0)」と、`enabled = false` を消す手順
- (d) `driver_pool = ["claude"]`: pass「not needed: codex is not in [org].driver_pool …」
- `org spawn`・`org start`・`org director start` の `--help`: 段落の文「The Codex hook trust check of」が 1 回ずつ

出力は evidence の log の「Observational probe」の節にある。spawn・start・director start の 1 行を実機の herdr で出すことはしていない(テストは stub の herdr で回す)。

## Coverage gaps

- テストは走らせていない。上の表のテストが通ることは `/test` が確かめる。
- AC9 の起動の 1 行の側: 1 行は doctor と同じ `codexHookTrustFor` から作るので、片方の綴りだけの承認で 1 行が出るとコードから読んだ。symlink の綴りの `--cwd` で 1 行を見るテストは無い。
- AC4 の「読めない」main の hooks.json(linked worktree から): テストは無い場合と形が違う場合だけ。読めない場合も `codexHookTrustHooksProblem` の同じ分岐を通るとコードから読んだ。
- AC6 の claude: 新しいテストで claude を見るのは spawn だけで、start と director start は `codex_hook_trust_notice.go:29` の早い return をコードで確かめた。
- 計画の Test plan の Edge cases のうち、main worktree を持たない git(bare の repo の linked worktree)と `CODEX_HOME` が空のときを見るテストは無い。bare の repo の名前が `.git` でなければ `codexLinkedWorktreeMain` は worktree 自身を root にし、名前が `.git` なら親を root にする。codex が後者を同じに扱うかは、注釈(`codex_hook_trust.go:212-215`)が「git が作った worktree なら通る」として写していない確かめ次第で、確かめていない。
- codex の規則(root の求め方、鍵の形、14 種の event、hash の範囲)は codex-cli 0.166.0 の実機と tag `rust-v0.166.0` のソースに合わせてある。版が変わったときの検査は無く、ずれれば warn に倒れる(計画の Risks)。
- 確度をいちばん上げる追加の確かめ: `TestCodexHookTrustNotice_PrintsNothingWithoutAConfigPathOrAHookToApprove` の隣に、symlink の綴りの checkout を `printCodexHookTrustNotice` に渡し、実パスだけに承認がある設定で 1 行が出ることを見るテストを 1 つ足す。AC9 の後半(起動の 1 行)を直接固定する。

## Verdict

- Verified: AC1〜AC8 はコードとテストのソースから満たす。AC1〜AC4 は HEAD の binary でも観察した。AC9 は doctor の側を 3 段のテストで満たし、起動の 1 行の側は同じ関数を通ることをコードで確かめた。issue #310 の受け入れ条件 3 つは AC1、AC2、AC6 に入っている。静的解析はすべて通った。README、`.codex/README.md` と `docs/recipes/codex-setup.md` の 2 つずつの写し、3 つの `--help` は振る舞いと合っている
- Partially verified: AC9 の起動の 1 行(直接のテストが無い)。codex の hash の範囲についての `.codex/README.md` の文は、self-review のソースの読みに依っている
- Not verified: テストの実行結果。実機の herdr での spawn・start・director start の 1 行。bare の repo の linked worktree での root
- Drift(/sync-docs へ): V1 `.codex/config.toml` の注釈(2 写し)。V2 tech-debt の Trigger 4 件(135、201、371、415 行目)。V3 計画の Scope が決めた tech-debt の 2 行。I1〜I3 は任意
- Verdict: pass
