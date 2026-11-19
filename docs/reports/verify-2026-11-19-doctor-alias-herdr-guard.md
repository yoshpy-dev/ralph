# Verify report: doctor-alias-herdr-guard

- Date: 2026-11-19(cycle 1 の run 2。`cycle-count.json` は `cycle: 1` のまま。run 1 の fail はインラインで直し、cross-review の前なので cycle は増えない)
- Plan: `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md`。承認の digest `2c2d8e5932b3` は、計画の Progress を変えた `3be7cde6` のあとに `./scripts/plan-visual.sh digest` で計算し直しても一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff a8c5e289...HEAD`(base `a8c5e289`、HEAD `3be7cde6`、16 ファイル、+1818 −77)。run 1(HEAD `4fc29299`)からの差は、run 1 の report と insight event(`7f87aa56`)、register の 3 行(`b38e97ba`)、計画の Progress の 1 行(`3be7cde6`)だけ。コード、テスト、recipe、skill は変わっていない(`git diff --stat 4fc29299..HEAD` は docs の 4 ファイル)。issue #343 の受け入れ条件 3 つも照らした。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-19-doctor-alias-herdr-guard.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run 1 の内容は先頭にそのまま残した(`head -c` と `cmp` で確かめた)。末尾の `# ===== Run 2 (HEAD 3be7cde6, 2026-11-19) =====` の節が run 2 の出力。run 1 の節には、静的検査の出力のあとに doctor を打った観察の行がある
- Run 1 の記録: run 1 の判定は fail で、原因は静的検査の `tech-debt README plan references` だけだった。原因と直しは「Run 1 の FAIL と直し」に、判定の行は `## Verdict` の `### Run 1(HEAD 4fc29299)` にそのまま残した

## Spec compliance

run 2 では、下の表の根拠を `git diff 7f87aa56..HEAD` に照らし直した。その差分が触れるのは `docs/tech-debt/README.md` と計画だけで、表が引くコード、テスト、recipe、skill のファイルは `4fc29299` と同じ内容(行番号も同じ)なので、run 1 の確かめは HEAD にもそのまま当てはまる。AC7 の 2 つの検査は run 2 でも通った。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: issue の 2 行だけの rc で、herdr があっても warn でなく info。Detail は 2 つの alias が herdr の pane では定義されないと言う | Met | 印は `scanShellAliasFile`(`internal/cli/doctor_shell_alias.go:920`、`:964-971`)が tracker から付ける。`checkShellAliases` は印があり値を読み切った def を `paneGuardedItems` に回し(`:1371`)、warn の条件(`:1440`)には入れず info の条件(`:1442`)に入れる。テストは `TestScanShellAliasFile_Issue343LinesAreBothMarked`(`internal/cli/doctor_shell_alias_test.go:2130`)と `TestCheckShellAliases_Issue343Rc_InfoNotWarn`(`:2194`。`~/.config/zsh/.zshrc` に issue の 2 行、status info、2 つの file:line と「not defined in herdr's panes」が 1 文に入る)。base と HEAD を build して打った doctor でも、base は warn、HEAD は info になった(Observational checks) |
| AC2: 8 つの形が info になる | Met | `TestCheckShellAliases_PaneGuardedForms_Info`(`:2231`)が 8 つの形すべてと基本の形を、herdr ありの `checkShellAliases` で info と確かめる。scanner の段は `TestScanShellAliasFile_MarksAliasesAHerdrPaneIDTestKeepsOutOfThePane`(`:1887`、28 件)、語の段は `TestShellAliasGuardTest_ReadsEachExactHerdrPaneIDForm`(`:1787`、3 書き方 × 2 flag × 2 変数 × 引用符の有無の 24 件) |
| AC3: 挙げた形が warn のまま | Met | `TestCheckShellAliases_FormsAPaneCanDefine_StillWarn`(`:2271`、20 件)が AC3 の形を 1 つずつ持つ。条件のない alias、`command -v`、`if command -v`、向きの逆 2 つ、並びの頭にない判定、`false &&` と `false && \` の続き、パイプ、`-z` の if の else、`elif` の後ろ、条件が判定 1 文でない if の 4 つ、`{ … }` の中、`$HERDR_PANE` と `$OTHER_ID`、閉じていない if と余分な `fi`。scanner の段は `:1971`(36 件)、`:2074`、`:2091`、`:2108`。「`elif` の来た if の中」は elif より後ろの alias と読んでいる。`-z` の then の側で elif より前の alias は pane では走らないので印が付く(`TestScanShellAliasFile_ElifStopsMarkingFromThatPointOn`、`:2060`)。この読み方は計画の Progress の S1 の注記に書いてあり、recipe と skill の「an `elif` and what follows it」とも合う |
| AC4: 条件付きと条件なしの codex があれば warn。warn の文には条件なしの方だけが並ぶ | Met | 印のある def は warn の判定に使う `codexItems` と flag の種類の変数に入らない(`:1371-1376`)。`TestCheckShellAliases_PaneGuardedAndUnguardedCodex_WarnListsOnlyUnguarded`(`:2324`)は、warn の文が条件なしの `-s` だけで始まり、条件付きの `-m` は別の文にだけ出て model の節を足さないことを確かめる。doctor を打った観察でも同じ形になった |
| AC5: 条件付きの finding の Detail にも alias の値は出ない | Met | `shellAliasPaneGuardedSentence`(`:1214`)は item(`alias <name> in <file>:<line> adds <flags>`)と固定の文だけで組む。`TestCheckShellAliases_PaneGuardedAliasValueNeverAppearsInDetail`(`:2498`、1 行と 2 行に続く値)と AC1 のテスト(`gpt-6-astra`、`opus`、`xhigh`)が確かめる。観察の doctor の行にも 3 つの値は 0 件 |
| AC6: 今のテストがすべて通る。期待値を弱めない | 一部(ソースだけ) | `git diff a8c5e289...HEAD -- internal/cli/doctor_shell_alias_test.go` に削除行は 0 行(追加 858 行)。S4 `7cf73b5a` のテストの変更は、この branch で足した行の書き換えと assertion 1 つの追加だけ。古い案内の文(`remove the alias or start herdr …`)を固定していたテストはなかった。テストが通るかは `/test` で確かめる |
| AC7: recipe 2 つと org skill の 4 写しに書き方と doctor の扱いが載る。`check-skill-sync.sh` と `check-sync.sh` が通る | Met | `docs/recipes/codex-seat-permissions.md:28-34`(3 つ目の回避策)と `:53-66`(severity と読む形)、`docs/recipes/isolated-herdr-server.md:277-282`、`.claude/skills/org/SKILL.md:216-221` と `:229-241`。root と `templates/base` の 4 組は `cmp` で一致し、`.claude` と `.agents` の org skill の差分の行も同じ。2 つの検査は run 1 と run 2 のログで PASS |

### issue #343 の受け入れ条件

| 受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| `HERDR_PANE_ID` を条件にした alias だけの rc で warn にならない(fixture のテストあり) | Met | AC1 の行 |
| 条件のない alias と、ほかの条件の alias は今までどおり warn | Met | AC3 の行。`command -v codex >/dev/null && alias codex=…` は観察の doctor でも warn |
| recipe と `/org` skill の 4 面に書き方が載る | Met | AC7 の行 |

## Static analysis

run 2(HEAD `3be7cde6`)の結果。

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | rc=0 | `tech-debt README plan references` は OK(run 1 は FAIL)。ほかの検査は run 1 と同じく通った。gofmt は `ok`、`go vet ./...` と staticcheck は出力なし、golangci-lint は `0 issues.`、shellcheck、`sh -n`、guard の awk の解析は OK で、ログに `Skipping` の行はない。ラッパーは自分の記録として `docs/evidence/verify-2026-11-19-122804.log` も書いた(これも gitignore に当たる) |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-skill-sync.sh` | ok | 14 skill が lock-step |
| `scripts/check-pipeline-sync.sh`、`scripts/check-template-purity.sh` | OK | |
| `secret-scan-branch.sh` | clean | `a8c5e289..3be7cde6` |
| `./scripts/plan-visual.sh digest <plan>` | `2c2d8e5932b3` | 計画の `- Approved:` の行と一致。`3be7cde6` の Progress の 1 行は digest を変えない |
| `git diff --check a8c5e289...HEAD` | 出力なし | |
| 差分の U+FFFD | 0 件 | |
| `git merge-tree --write-tree --name-only origin/main HEAD` | 衝突なし | 手元の `origin/main` は `a8c5e289` で base と同じ。fetch はしていない |
| `/pr` の archive の模擬 | 3 件を書き換え、足りない参照 0 件 | scratch に `scripts/archive-plan.sh`、register、計画、register が引く計画を写して `sh scripts/archive-plan.sh 2026-11-19-doctor-alias-herdr-guard.md` を打つと「Updated 3 reference(s)」。そのあと register に active のこの計画のパスは 0 件、archive は 3 件で、`check_tech_debt_plan_refs` と同じ論理で足りない参照はない |

### Run 1 の FAIL と直し

- run 1 の原因: S4 `7cf73b5a` は self-review の L6(comment の計画のパスを archive に)を直すときに、register に書いた 3 つの行の計画のパスも archive にした。`check_tech_debt_plan_refs`(`scripts/verify.local.sh:141`)は、register が引く `docs/plans/active/` と `docs/plans/archive/` のパスがその場所にあることを確かめる。register の行は、PR のあいだは `docs/plans/active/` を引き、/pr の `scripts/archive-plan.sh` が archive に書き換える決まりで、前の PR もそうしていた(codex-hook-trust-check は `23a57869` で active を書き、`ed91722e` の archive で書き換わった)
- 直し(`b38e97ba`、`docs/tech-debt/README.md` の 1 ファイル、3 行): `git diff --word-diff=porcelain 7f87aa56..HEAD -- docs/tech-debt/README.md` の変わった語は次の 4 種類だけ。136、137、138 行の計画のパスを archive から active に戻し(3 か所)、136 行と 138 行の数え直しで `1436` を `1448` に(2 か所)、`be254791` を `4fc29299` に(2 か所)、`2572` を `2576` に(1 か所)した。HEAD の `wc -l` は `internal/cli/doctor_shell_alias.go` が 1448、その test が 2576 で、base `a8c5e289` の 968 と 1718 も行の記述と合う。register の中に archive のこの計画のパスは 0 件
- Go のソースの comment(`doctor_shell_alias.go:472`、`:570`、`:899`、`:1298`、test の `:1724`)は archive のパスのまま。これを読む検査はなく、/pr のあとは正しいパスになる

## Documentation drift

run 1 の行を HEAD で見直した。変わったのは register の計画のパスと行数の 2 行(どちらも `b38e97ba` で直った)。ほかの行は `git diff 7f87aa56..HEAD` が触れていないので、run 1 の判断のまま。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/recipes/codex-seat-permissions.md`(2 写し) | Yes | 読む書き方(`[[ … ]]`、`[ … ]`、`test …` と `-n`/`-z`、2 つの変数の書き方、引用符の有無)は `shellAliasGuardTest`(`:499`)と同じ。並びの頭の説明(前の行から続かない行の初め、`;`、`then`、`else`、`do`、`{` の後ろ)は `inlineGuarded`(`:787`)、`prevLineEndedClosed`(`:652`)、`shellAliasListHeadMarkers` と同じ。if の形(判定 1 文だけの条件、`-z` の then、`-n` の else)は `followIf`(`:709`)と `condClosedBeforeThen`(`:660`)と同じ。「ほかの形」の列挙(向きの逆、`{ …; }` の中、`elif` とその後ろ、fish)もコードが印を付けない形と合う。INFO: severity の括弧の「`HERDR_PANE_ID` test keeps the alias out … info」に対し、値を読み切れていない def(閉じない引用符)は印があっても warn に残る(`:1371`)。文の要約と端のケースの差で、Detail には not-fully-parsed の節が出る |
| `docs/recipes/isolated-herdr-server.md`(2 写し) | Yes | 1 文で書き方を示し、doctor が読む形は org skill と codex の recipe に任せている。同じ recipe の `env -i` の段落(169 行)が外す `HERDR_PANE_ID` は、サーバーを起動した側の pane の変数で、herdr が新しい pane に入れる値とは別なので、食い違いはない |
| org skill の 4 写し | Yes | 英語の recipe と同じ内容を日本語で書いている。4 写しは一致 |
| `internal/cli/doctor.go` の Check 8b の comment(163-170 行) | Yes | 「An alias a HERDR_PANE_ID test keeps out of herdr's panes is informational whatever its flags」。上と同じ INFO(読み切れていない def は warn、flag のない def は harmless で pass)。comment の主語は flag を足す alias なので、後者は外れていない |
| `checkShellAliases` の doc comment | Yes | 「on a fully read def」と書いてあり、`:1371` の `!d.Incomplete` と合う |
| `docs/tech-debt/README.md` の計画のパス(136、137、138 行) | Yes | run 1 は No。`b38e97ba` で active のパスに戻り、静的検査も通った |
| `docs/tech-debt/README.md` の 136 行と 138 行の行数 | Yes | run 1 は古い(INFO)。`b38e97ba` で「at 4fc29299」の 1448 行と 2576 行になり、HEAD の数と合う |
| `docs/tech-debt/README.md` の 373 行(templates の recipe が引く `docs/evidence/` の項) | No(記録の漏れ) | run 1 と同じで、HEAD でもこの branch の carry-over の文はない。Trigger の「The next edit of any of these five recipes」に当たる。この branch は `codex-seat-permissions.md` と `isolated-herdr-server.md` を 2 写しとも変えたが、日付つきの carry-over の文がない。これまでの PR は、変えるたびに文を足してきた。引用の数は base と HEAD で変わらない(`codex-seat-permissions.md` は 3 件 3 ファイル、`isolated-herdr-server.md` は 14 件 7 ファイルと `docs/tech-debt/README.md` の 1 件、`grep -o` で数えた)。`/sync-docs` に渡す |
| `docs/tech-debt/README.md` の 417 行(800 行の目安を超えた 5 ファイル) | No(記録の漏れ) | run 1 と同じで、HEAD でもこの branch の carry-over の文はない。Trigger の「The next change to any of the five」に `internal/cli/doctor.go` が入る。この branch は `1af2fcee` で Check 8b の comment を変え、1648 → 1650 行になった(base `a8c5e289` と HEAD で数えた)。`/sync-docs` に渡す |
| `b38e97ba` の register の編集が当てる Trigger | なし | 編集した 3 行はこの branch が足した行で、branch の差分のファイルの一覧も run 1 から増えていない。Trigger の列(各行の 4 つ目のセル)を python で読み、register、計画、verify report を名指す Trigger を探したが、この編集が当たるものはなかった |
| 計画の Acceptance criteria の印と Progress | 未更新(INFO) | AC1〜AC7 は `[ ]` のまま。Progress の「Verification artifact created」も `[ ]` で、run 1 の注記は「verify は打ち直す」で終わっている。これまでも /pr のころに付け直している |

## Observational checks

run 2 では打ち直していない。doctor を打つコードは `4fc29299` から変わっていないので、run 1 の観察が HEAD にも当てはまる。以下は run 1 の記録。

- `go build` で base `a8c5e289` と HEAD `4fc29299` の `ralph` を作り、scratch の git repo で `HOME=<scratch>`、`ZDOTDIR` なしで `ralph doctor` を打った。rc は issue #343 の 2 行だけの `~/.config/zsh/.zshrc`。この機械では `herdr: pass — available`。
  - base: `⚠ Shell aliases (codex/claude): warn — alias codex in …/.zshrc:2 adds --model (-m) — herdr expands the alias …`
  - HEAD: `ℹ Shell aliases (codex/claude): info — alias claude in …/.zshrc:1 adds --model; alias codex in …/.zshrc:2 adds --model (-m) — not defined in herdr's panes; …`。値(`gpt-6-astra`、`opus`、`xhigh`)は行に出ない
  - HEAD で rc を差し替えた 3 つ: `command -v codex >/dev/null && alias codex=…` は warn。条件付きの `-m` と条件なしの `-s` の 2 行は warn で、warn の文は `-s` の 2 行目だけを並べ、1 行目は条件付きの文に出る。`false &&` の次の行の判定は warn
  - パスが `~` でなく scratch の絶対パスで出たのは、`$TMPDIR` の末尾の `/` で `HOME` に `//` が入ったため。base も同じ出方で、この差分とは関係しない
- zsh 5.9 で issue の rc を読み込むと、`HERDR_PANE_ID=wD:p1` では codex と claude の alias が定義されず、変数がなければ 2 つとも定義された

## Coverage gaps

- テストは走らせていない。AC6 と、上の表でテストを根拠にした行が通るかは `/test` で確かめる
- herdr の pane に `HERDR_PANE_ID` が入ることは、この run でも確かめていない。根拠は issue の報告(herdr 0.7.5)と、ralph の approve の仕掛け(`internal/org/approve.go:218`)が同じ変数に頼っていること。ほかの herdr の版は分からない
- 印を誤って付ける 3 つの形(register の 137 行)は打ち直していない。self-review の 86 本の zsh の probe が確かめている
- `/pr` の archive は scratch の写しで模擬しただけで、本物の `/pr` は打っていない
- 決定的な検査で足りないものはない。register の Trigger の当たりは今も人が見ている(373 行と 417 行の漏れ)

## Verdict

- Verdict: **pass**(run 2、HEAD `3be7cde6`)。静的検査は rc=0 で、run 1 の唯一の FAIL(`tech-debt README plan references`)は `b38e97ba` で閉じた。run 1 から変わったのは register の 3 行と計画の Progress の 1 行だけで、spec の適合の判断は run 1 のまま
- Verified: AC1、AC2、AC3、AC4、AC5、AC7 と issue #343 の受け入れ条件 3 つ(コード、テストのソース、run 1 の doctor を打った観察。コードは run 1 から変わっていない)。`run-static-verify.sh` の全項目(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、plan references、secret scan)。承認の digest。register の 3 つのパスと行数。`/pr` の archive の模擬で、register の参照が 3 件とも書き換わり、足りない参照が出ないこと
- Partially verified: AC6(テストの削除と弱めがないことはソースで確かめた。通るかは `/test`)
- Not verified: herdr の pane の `HERDR_PANE_ID`(0.7.5 のほかの版も)。本物の `/pr` の archive
- `/sync-docs` に渡すもの: register の 373 行と 417 行の carry-over の文。計画の AC の印と Progress の「Verification artifact created」は orchestrator か `/sync-docs` が付け直す
- 次にいちばん効く確かめ: `/test` で `internal/cli` のテストを打ち、AC6 と、テストを根拠にした AC1〜AC5 の行を確かめる

### Run 1(HEAD 4fc29299)

- 判定: **fail**。静的検査が rc=1(`tech-debt README plan references`)。直しは register の 3 つのパスを active にすることだけで、spec の適合と文書の内容に問題はない
- Verified: AC1、AC2、AC3、AC4、AC5、AC7 と issue #343 の受け入れ条件 3 つ(コード、テストのソース、doctor を打った観察)。gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、secret scan。承認の digest
- Partially verified: AC6(テストの削除と弱めがないことはソースで確かめた。通るかは `/test`)
- Not verified: herdr の pane の `HERDR_PANE_ID`(0.7.5 のほかの版も)
- 次にいちばん効く確かめ: 3 つのパスを直したあと `./scripts/run-static-verify.sh` を打ち直し、rc=0 を見る。373 行と 417 行の carry-over の文と、136・138 行の行数の更新は `/sync-docs` に渡す
