# Verify report: doctor-alias-herdr-guard

- Date: 2026-11-19(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md`。承認の digest `2c2d8e5932b3` は `./scripts/plan-visual.sh digest` で計算し直した値と一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff a8c5e289...HEAD`(base `a8c5e289`、HEAD `4fc29299`、15 ファイル、+1730 −77)。issue #343 の受け入れ条件 3 つも照らした。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-19-doctor-alias-herdr-guard.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。静的検査の出力のあとに、doctor を打った観察の行を足してある

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: issue の 2 行だけの rc で、herdr があっても warn でなく info。Detail は 2 つの alias が herdr の pane では定義されないと言う | Met | 印は `scanShellAliasFile`(`internal/cli/doctor_shell_alias.go:920`、`:964-971`)が tracker から付ける。`checkShellAliases` は印があり値を読み切った def を `paneGuardedItems` に回し(`:1371`)、warn の条件(`:1440`)には入れず info の条件(`:1442`)に入れる。テストは `TestScanShellAliasFile_Issue343LinesAreBothMarked`(`internal/cli/doctor_shell_alias_test.go:2130`)と `TestCheckShellAliases_Issue343Rc_InfoNotWarn`(`:2194`。`~/.config/zsh/.zshrc` に issue の 2 行、status info、2 つの file:line と「not defined in herdr's panes」が 1 文に入る)。base と HEAD を build して打った doctor でも、base は warn、HEAD は info になった(Observational checks) |
| AC2: 8 つの形が info になる | Met | `TestCheckShellAliases_PaneGuardedForms_Info`(`:2231`)が 8 つの形すべてと基本の形を、herdr ありの `checkShellAliases` で info と確かめる。scanner の段は `TestScanShellAliasFile_MarksAliasesAHerdrPaneIDTestKeepsOutOfThePane`(`:1887`、28 件)、語の段は `TestShellAliasGuardTest_ReadsEachExactHerdrPaneIDForm`(`:1787`、3 書き方 × 2 flag × 2 変数 × 引用符の有無の 24 件) |
| AC3: 挙げた形が warn のまま | Met | `TestCheckShellAliases_FormsAPaneCanDefine_StillWarn`(`:2271`、20 件)が AC3 の形を 1 つずつ持つ。条件のない alias、`command -v`、`if command -v`、向きの逆 2 つ、並びの頭にない判定、`false &&` と `false && \` の続き、パイプ、`-z` の if の else、`elif` の後ろ、条件が判定 1 文でない if の 4 つ、`{ … }` の中、`$HERDR_PANE` と `$OTHER_ID`、閉じていない if と余分な `fi`。scanner の段は `:1971`(36 件)、`:2074`、`:2091`、`:2108`。「`elif` の来た if の中」は elif より後ろの alias と読んでいる。`-z` の then の側で elif より前の alias は pane では走らないので印が付く(`TestScanShellAliasFile_ElifStopsMarkingFromThatPointOn`、`:2060`)。この読み方は計画の Progress の S1 の注記に書いてあり、recipe と skill の「an `elif` and what follows it」とも合う |
| AC4: 条件付きと条件なしの codex があれば warn。warn の文には条件なしの方だけが並ぶ | Met | 印のある def は warn の判定に使う `codexItems` と flag の種類の変数に入らない(`:1371-1376`)。`TestCheckShellAliases_PaneGuardedAndUnguardedCodex_WarnListsOnlyUnguarded`(`:2324`)は、warn の文が条件なしの `-s` だけで始まり、条件付きの `-m` は別の文にだけ出て model の節を足さないことを確かめる。doctor を打った観察でも同じ形になった |
| AC5: 条件付きの finding の Detail にも alias の値は出ない | Met | `shellAliasPaneGuardedSentence`(`:1214`)は item(`alias <name> in <file>:<line> adds <flags>`)と固定の文だけで組む。`TestCheckShellAliases_PaneGuardedAliasValueNeverAppearsInDetail`(`:2498`、1 行と 2 行に続く値)と AC1 のテスト(`gpt-6-astra`、`opus`、`xhigh`)が確かめる。観察の doctor の行にも 3 つの値は 0 件 |
| AC6: 今のテストがすべて通る。期待値を弱めない | 一部(ソースだけ) | `git diff a8c5e289...HEAD -- internal/cli/doctor_shell_alias_test.go` に削除行は 0 行(追加 858 行)。S4 `7cf73b5a` のテストの変更は、この branch で足した行の書き換えと assertion 1 つの追加だけ。古い案内の文(`remove the alias or start herdr …`)を固定していたテストはなかった。テストが通るかは `/test` で確かめる |
| AC7: recipe 2 つと org skill の 4 写しに書き方と doctor の扱いが載る。`check-skill-sync.sh` と `check-sync.sh` が通る | Met | `docs/recipes/codex-seat-permissions.md:28-34`(3 つ目の回避策)と `:53-66`(severity と読む形)、`docs/recipes/isolated-herdr-server.md:277-282`、`.claude/skills/org/SKILL.md:216-221` と `:229-241`。root と `templates/base` の 4 組は `cmp` で一致し、`.claude` と `.agents` の org skill の差分の行も同じ。2 つの検査はログで PASS |

### issue #343 の受け入れ条件

| 受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| `HERDR_PANE_ID` を条件にした alias だけの rc で warn にならない(fixture のテストあり) | Met | AC1 の行 |
| 条件のない alias と、ほかの条件の alias は今までどおり warn | Met | AC3 の行。`command -v codex >/dev/null && alias codex=…` は観察の doctor でも warn |
| recipe と `/org` skill の 4 面に書き方が載る | Met | AC7 の行 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | **rc=1** | FAIL は 1 つだけで、`tech-debt README plan references`。`docs/tech-debt/README.md` が `docs/plans/archive/2026-11-19-doctor-alias-herdr-guard.md` を 3 か所(136、137、138 行の Related の欄)で引くが、計画はまだ `docs/plans/active/` にある。ほかは通った。gofmt は `ok`、`go vet ./...` と staticcheck は出力なし、golangci-lint は `0 issues.`、shellcheck と `sh -n` は OK、ログに `Skipping` の行はない |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-skill-sync.sh` | ok | 14 skill が lock-step |
| `scripts/check-pipeline-sync.sh`、`scripts/check-template-purity.sh` | OK | |
| `secret-scan-branch.sh` | clean | `a8c5e289..4fc29299` |
| `./scripts/plan-visual.sh digest <plan>` | `2c2d8e5932b3` | 計画の `- Approved:` の行と一致 |
| `git diff --check a8c5e289...HEAD` | 出力なし | |
| 差分の U+FFFD | 0 件 | |
| `git merge-tree --write-tree --name-only origin/main HEAD` | 衝突なし | 手元の `origin/main` は `a8c5e289` で base と同じ。fetch はしていない |

### FAIL の原因と直し方

- 原因: S4 `7cf73b5a` は self-review の L6(comment の計画のパスを archive に)を直すときに、register に書いた 3 つの行の計画のパスも archive にした。`check_tech_debt_plan_refs`(`scripts/verify.local.sh:141`)は、register が引く `docs/plans/active/` と `docs/plans/archive/` のパスがその場所にあることを確かめる。register の行は、PR のあいだは `docs/plans/active/` を引き、/pr の `scripts/archive-plan.sh` が archive に書き換える決まりで、前の PR もそうしていた(codex-hook-trust-check は `23a57869` で active を書き、`ed91722e` の archive で書き換わった)
- 直し方: `docs/tech-debt/README.md` の 136、137、138 行の `docs/plans/archive/2026-11-19-doctor-alias-herdr-guard.md` を `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md` にする。register の写しで 3 つを書き換えて同じ検査の論理を当てると、足りない参照は 0 件になった(ログの末尾)。Go のソースの comment(`doctor_shell_alias.go:472`、`:570`、`:899`、`:1298`、test の `:1724`)は検査の外で、/pr のあとは正しいパスになるので、そのままでよい
- CI(`.github/workflows/verify.yml`)は `pull_request` でしか走らず、PR を作る前に /pr が計画を archive に動かすので、直さなくても CI は通る見込み。ただ、ここから /pr までの `run-static-verify.sh` は毎回 rc=1 になり、/pr が途中で止まれば register は無いパスを引いたまま残る

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/recipes/codex-seat-permissions.md`(2 写し) | Yes | 読む書き方(`[[ … ]]`、`[ … ]`、`test …` と `-n`/`-z`、2 つの変数の書き方、引用符の有無)は `shellAliasGuardTest`(`:499`)と同じ。並びの頭の説明(前の行から続かない行の初め、`;`、`then`、`else`、`do`、`{` の後ろ)は `inlineGuarded`(`:787`)、`prevLineEndedClosed`(`:652`)、`shellAliasListHeadMarkers` と同じ。if の形(判定 1 文だけの条件、`-z` の then、`-n` の else)は `followIf`(`:709`)と `condClosedBeforeThen`(`:660`)と同じ。「ほかの形」の列挙(向きの逆、`{ …; }` の中、`elif` とその後ろ、fish)もコードが印を付けない形と合う。INFO: severity の括弧の「`HERDR_PANE_ID` test keeps the alias out … info」に対し、値を読み切れていない def(閉じない引用符)は印があっても warn に残る(`:1371`)。文の要約と端のケースの差で、Detail には not-fully-parsed の節が出る |
| `docs/recipes/isolated-herdr-server.md`(2 写し) | Yes | 1 文で書き方を示し、doctor が読む形は org skill と codex の recipe に任せている。同じ recipe の `env -i` の段落(169 行)が外す `HERDR_PANE_ID` は、サーバーを起動した側の pane の変数で、herdr が新しい pane に入れる値とは別なので、食い違いはない |
| org skill の 4 写し | Yes | 英語の recipe と同じ内容を日本語で書いている。4 写しは一致 |
| `internal/cli/doctor.go` の Check 8b の comment(163-170 行) | Yes | 「An alias a HERDR_PANE_ID test keeps out of herdr's panes is informational whatever its flags」。上と同じ INFO(読み切れていない def は warn、flag のない def は harmless で pass)。comment の主語は flag を足す alias なので、後者は外れていない |
| `checkShellAliases` の doc comment | Yes | 「on a fully read def」と書いてあり、`:1371` の `!d.Incomplete` と合う |
| `docs/tech-debt/README.md` の計画のパス(136、137、138 行) | **No** | 上の FAIL。active のパスにする |
| `docs/tech-debt/README.md` の 373 行(templates の recipe が引く `docs/evidence/` の項) | No(記録の漏れ) | Trigger の「The next edit of any of these five recipes」に当たる。この branch は `codex-seat-permissions.md` と `isolated-herdr-server.md` を 2 写しとも変えたが、日付つきの carry-over の文がない。これまでの PR は、変えるたびに文を足してきた。引用の数は base と HEAD で変わらない(`codex-seat-permissions.md` は 3 件 3 ファイル、`isolated-herdr-server.md` は 14 件 7 ファイルと `docs/tech-debt/README.md` の 1 件、`grep -o` で数えた) |
| `docs/tech-debt/README.md` の 417 行(800 行の目安を超えた 5 ファイル) | No(記録の漏れ) | Trigger の「The next change to any of the five」に `internal/cli/doctor.go` が入る。この branch は `1af2fcee` で Check 8b の comment を変え、1648 → 1650 行になった(base `a8c5e289` と HEAD で数えた)が、carry-over の文がない |
| `docs/tech-debt/README.md` の 136 行と 138 行の行数 | 古い(INFO) | 「968 to 1436 lines … at be254791」と「2572 lines」は、書いたとおり `be254791` での数としては正しい。ただ、その行を書いた `7cf73b5a` 自身が `doctor_shell_alias.go` に 12 行、test に 4 行を足したので、HEAD では 1448 行と 2576 行 |
| 計画の Acceptance criteria の印 | 未更新(INFO) | AC1〜AC7 は `[ ]` のまま。これまでも /pr のころに付け直している |

## Observational checks

- `go build` で base `a8c5e289` と HEAD `4fc29299` の `ralph` を作り、scratch の git repo で `HOME=<scratch>`、`ZDOTDIR` なしで `ralph doctor` を打った。rc は issue #343 の 2 行だけの `~/.config/zsh/.zshrc`。この機械では `herdr: pass — available`。
  - base: `⚠ Shell aliases (codex/claude): warn — alias codex in …/.zshrc:2 adds --model (-m) — herdr expands the alias …`
  - HEAD: `ℹ Shell aliases (codex/claude): info — alias claude in …/.zshrc:1 adds --model; alias codex in …/.zshrc:2 adds --model (-m) — not defined in herdr's panes; …`。値(`gpt-6-astra`、`opus`、`xhigh`)は行に出ない
  - HEAD で rc を差し替えた 3 つ: `command -v codex >/dev/null && alias codex=…` は warn。条件付きの `-m` と条件なしの `-s` の 2 行は warn で、warn の文は `-s` の 2 行目だけを並べ、1 行目は条件付きの文に出る。`false &&` の次の行の判定は warn
  - パスが `~` でなく scratch の絶対パスで出たのは、`$TMPDIR` の末尾の `/` で `HOME` に `//` が入ったため。base も同じ出方で、この差分とは関係しない
- zsh 5.9 で issue の rc を読み込むと、`HERDR_PANE_ID=wD:p1` では codex と claude の alias が定義されず、変数がなければ 2 つとも定義された

## Coverage gaps

- テストは走らせていない。AC6 と、上の表でテストを根拠にした行が通るかは `/test` で確かめる
- herdr の pane に `HERDR_PANE_ID` が入ることは、この run では確かめていない。根拠は issue の報告(herdr 0.7.5)と、ralph の approve の仕掛け(`internal/org/approve.go:218`)が同じ変数に頼っていること。ほかの herdr の版は分からない
- 印を誤って付ける 3 つの形(register の 137 行)は打ち直していない。self-review の 86 本の zsh の probe が確かめている
- 決定的な検査で足りないものはない。register の Trigger の当たりは今も人が見ている(373 行と 417 行の漏れ)

## Verdict

- 判定: **fail**。静的検査が rc=1(`tech-debt README plan references`)。直しは register の 3 つのパスを active にすることだけで、spec の適合と文書の内容に問題はない
- Verified: AC1、AC2、AC3、AC4、AC5、AC7 と issue #343 の受け入れ条件 3 つ(コード、テストのソース、doctor を打った観察)。gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、secret scan。承認の digest
- Partially verified: AC6(テストの削除と弱めがないことはソースで確かめた。通るかは `/test`)
- Not verified: herdr の pane の `HERDR_PANE_ID`(0.7.5 のほかの版も)
- 次にいちばん効く確かめ: 3 つのパスを直したあと `./scripts/run-static-verify.sh` を打ち直し、rc=0 を見る。373 行と 417 行の carry-over の文と、136・138 行の行数の更新は `/sync-docs` に渡す
