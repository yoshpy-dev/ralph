# sync-docs report: codex-hook-trust-check

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(承認の digest `147d7ec0e7ae`)。issue #310
- Branch: `feat/codex-hook-trust-check`。run 開始時の HEAD は `d9cd00c0`。origin/main `d2e5c6e9`(#312 の merge)は `f36cfc52` で取り込み済み
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `dddb946d`(更新 `75f3757d`。Merge: yes、LOW 9 件 L1〜L9)、verify `9492dc5a`(更新 `6441d229`。pass。LOW 3 件 V1〜V3、INFO 4 件 I1〜I4)、test `d9cd00c0`(pass)
- 見た範囲: `git diff origin/main...HEAD`(この commit の前で 22 ファイル、+3,307/-77)。Go の本体は +888/-34、テストは +1,750/-2、README・`.codex/README.md`・recipe は +105/-41。walkthrough は書いていない(`/pr` が決める)
- 制約: 計画の本文と AC の箱は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `147d7ec0e7ae`、`git diff --numstat` は 6/3(足した 3 つの項と、`[x]` にした 3 つの箱だけ)。Go のコードとテストとスキルは変えていない

## Summary

verify の V1〜V3 と I4 を扱った。V1 は `.codex/config.toml` の注釈(root と雛形の 2 写し)を直した。V2 は tech-debt の Trigger を引いた 4 行に持ち越しの note を足し、掃引でさらに 2 行(153、195)が当たったので同じ形で足した。V3 は新しい tech-debt の行を 3 つ足した。I4 は計画の Progress に S6 とパイプラインの結果を足した。README、`.codex/README.md` と recipe の 2 つずつの写し、3 つの動詞の `--help` は実装と合っていて、直した所は無い。

## Changes made

| File | Change |
|------|--------|
| `.codex/config.toml` | 113〜114 行目の注釈を、`ralph doctor` が配線を確かめ、codex が `[org].driver_pool` にあるとき Codex hook trust check が承認の鍵を読む(hash は見ないので承認のあとで編集した hook は通る)と言う形に直した。注釈だけで、4 行になった。下の「V1」 |
| `templates/base/.codex/config.toml` | 同じ直し(`check-sync.sh` が一致を求める) |
| `docs/tech-debt/README.md` | 既存の 6 行(135、153、195、201、371、415)の Trigger の末尾に持ち越しの note と、Related の末尾に計画の path を足した(どれも挿入だけ)。新しい 3 行を末尾に足した(471〜473 行)。下の「V2」「V3」 |
| `docs/plans/active/2026-11-12-codex-hook-trust-check.md` | `## Progress checklist` の「Implementation started」の下に S6、パイプライン 1 回目の結果、sync-docs の 3 項を足し、artifact の 3 つの箱を `[x]` にした。「PR created」と AC の箱は `[ ]` のまま |
| `docs/reports/sync-docs-2026-11-12-codex-hook-trust-check.md` | この report |
| `docs/insights/events/2026-11-12-codex-hook-trust-check.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行 |

commit は 1 つ(`docs: sync docs for codex-hook-trust-check`)。

## V1: `.codex/config.toml` の注釈

元の文は「`ralph doctor` verifies the wiring but cannot probe trust state.」で、hook の対話での承認を話題にしていた。今は doctor の Codex hook trust check が `[hooks.state]` の鍵を見る。新しい文は次のとおり。

```
`ralph doctor` verifies the wiring and, when codex is in
[org].driver_pool, its Codex hook trust check reads the hooks' approval
keys from codex's user config (not the hash, so a hook edited after its
approval still passes).
```

`.codex/README.md` の Trust UX と doctor の段落(185 行目から)と同じ内容で、同じ言い方を使った。`grep` で `cannot probe` を探すと、残るのは `internal/cli/doctor.go` の注釈(`checkCodexEffectiveConfig` の doc。「Codex's project trust state」のこと。S2 で直済み)だけで、`.codex/config.toml` の 2 写しには無くなった。`.codex/config.toml` は codex が書き換えることがある追跡ファイルなので(tech-debt の 145 行目)、2 つの写しを `cmp` で比べて同じことを確かめた。

## V2: tech-debt の Trigger

verify が挙げた 4 行は、行を開いて Trigger を読んだ。note は Trigger セルの末尾に足し(`. 2026-11-12: feat/codex-hook-trust-check (#310) hit ...`)、Related の末尾に `docs/plans/active/2026-11-12-codex-hook-trust-check.md` を足した。`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` とは書いていない。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 135「`checkCodexAgmsgWritableRoot`」 | `doctor_codex_writable_root.go`(S1 148f9b19) | `readCodexUserConfig` の bytes を読む部分を `readCodexFileBounded` に切り出し、`codexConfigMaxBytes` の doc に 1 節を足した。991 → 1009 行(origin/main d2e5c6e9 と 148f9b19 で `wc -l`)。`pathCovers`、`codexWritableRootSuffix`、読む設定は変わっていない(self-review が切り出しの前後で 4 つの戻り値の組を突き合わせた)。持ち越し。新しい check も user config だけを読むので、その穴は 472 行目の行が持つと note に書いた |
| 201「Test gaps left by org-feature-worktree」(e)(f) | `runOrgStartPlan`(S3) | `printCodexHookTrustNotice` の呼び出しと 3 行の注釈、その doc の 3 行。`TestCodexHookTrustNotice_StartPlanChecksTheMainCheckout` が新しい呼び出しを走らせる。(e)(f) の走らない枝は今も走らない。持ち越し |
| 371「The recipes under `templates/base/docs/recipes/`」 | `codex-setup.md`(S4、2 写し) | doctor の段落を書き直した。`docs/evidence/` の引用は `grep -o` で 1 件 1 ファイル(`codex-config-rewrite-2026-09-30.md`)のまま、`docs/tech-debt/README.md` の引用は 0 件。直前までの note と同じ形(「The row stays open」) |
| 415「Five files went past the 800-line guide」 | `internal/cli/doctor.go`(S2 51256efd) | 1615 → 1624 行(origin/main d2e5c6e9 と 51256efd で `wc -l`)。Check 3b の呼び出しと 6 行の注釈、`checkCodexEffectiveConfig` の doc と pass の文。move-only の分割は Scope の外。ほかの 4 ファイルは触っていない。持ち越し |

掃引で見つけた 2 行(verify も handoff も挙げていない):

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 153「codex ignores `[profiles.*]` in a project-local `.codex/config.toml`」 | **この sync-docs の編集自身**(V1) | Trigger は「The next change to `.codex/config.toml`」。V1 で root と雛形の 2 写しを編集したので当たる。`[profiles.*]` の表と Profiles の注釈は変えていない。持ち越し。Related の計画の参照は「Progress checklist; the carry-over note」とした |
| 195「A `ralph` of another version on the pane's `PATH`」 | `newOrgStartCmd` の `RunE`(S3 の呼び出し 1 つと S4 の `--help` の段落) | Trigger に「the next change to the `RunE` of `newOrgStartCmd`, for the `start <task>` gap」がある。#313 の note と同じ形で、note が operator の stderr に出るだけで leader の task には入らないので、`start <task>` の leader に ledger の行が渡らない穴と版のずれは変わらない、と書いた。持ち越し |

当たらないと判断した行(Trigger を読んだ。行番号は HEAD の `docs/tech-debt/README.md`):

| 行 | 判断 |
|----|----|
| 144 | Trigger は `doctor_codex_writable_root_test.go` の変更と、`runDoctorFull`/`projectDir` の対象 directory の決め方の変更。どちらも変えていない |
| 148 | Trigger は `resolveOrgConfig`、`withMainWorktreeOrgLimits`、`MainWorktreeRoot`、`ResolveOrgStateDir` の変更。どれも変えていない。新しい `codexHookRoot` は codex の root の求め方の写しで、これらの関数を呼ばない(「project-root discovery helper が 1 つで足りる」という行の案から見ると 3 つ目の root の求め方が増えたが、Trigger の文言には当たらない) |
| 110 | `cli_test.go` の Trigger は `TestAddPack_LegacyManifest_StaysOwnerless` の近くで、RESOLVED の印がある。S3 は `TestCheckCodexEffectiveConfig_FullyWired_Pass` の注釈だけを直した |
| 118 | `templates/base/docs/quality/quality-gates.md` と template の verify chain への次の変更。この diff は触っていない |
| 127(取り消し線)、128、123、106 | 127 は解消済み。128 は `.codex/hooks.json` への SessionEnd/PreCompact の配線。123 と 106 は解消済み。この diff は hooks.json を変えていない |
| 145 | 「次の発生」(`.codex/config.toml` が書き換わること)。この commit の編集は Edit ツールで、発生ではない |
| 156、138、255 | codex の呼び出し、`ObserveCodexEffectiveModel`、`runDoctor*` のテストが codex の cache で落ちること。どれも変えていない |
| 189、200、203 | Trigger が挙げる関数(`scopeReservedEvent`、`StartFeatureResult`、`maxFeatureOrgIDLen` など)を変えていない。200 の (i) は `maxFeatureOrgIDLen` の変更で、`org start --help` の段落は足したが、その上限の文には触っていない |
| 274、275、432 | `checkDirectorStart` の拒否の順、`codexNotNeededWhyA`、`HerdrServerGate` の呼び出しの場所を変えていない。`startDirector` に足したのは `ResolveRoleModel` と cwd の決定のあとの 1 行の表示 |
| 464、465、466、467 | `/director` skill、`checkManifestVersion`、`printScaffoldVersionNotice` の root の引数と `orgScaffoldRoot`、`orgScaffoldVersionNoticeHelp` を変えていない(467 の (b) は新しい段落がその後ろに続くだけで、文は変えていない) |
| 177、199、318、368、401、425、431、444 | Trigger が挙げる `stop --all`、`start --plan` の二重起動、director の確認、split plan の preamble、`checkRetiredOrgStateDirEnv`、4 つのテスト、`ResolveRoleModel`、`confirmSeatPane` を変えていない |

掃引は、Trigger セルを python で切り出し(`" | "` で分け、バッククォートの中は区切りにしない)、`~~...~~` と最初の日付つきの note より後ろを外し、変えたファイルの名前、関数名、`--help`、`.codex/config.toml`、`README.md` などの語で探した。当たった 78 行のほとんどは `doctor`、`docs/evidence`、`--help` のような一般語の当たりで、特定の識別子で当たった行と、Debt の列だけが当たった行(「any of」「the five」の形の Trigger)を読んだ。

## V3: 新しい tech-debt の行(471〜473 行)

| 行(先頭の語) | 内容 |
|----|----|
| 「A hook that was changed after codex approved it still passes...」 | 計画の Scope の最後の項。hash の食い違いを座席の pane に出る codex の「Hooks need review」で知らせる案(見張りが pane を読んで `ALERT` にする)。実装していない。`ralph org resume` がそのダイアログを選択の画面として扱い何も打たないことは `resume_screens.go` の注釈にある。見張りの stall の ALERT がその画面で既に鳴るかは確かめていない、と Impact に書いた。Trigger は報告か、`resume_screens.go` の画面の規則と見張りの ALERT の条件の次の変更 |
| 「Where the Codex hook trust check ... reads differently from codex」 | (a) codex が鍵を作らない handler(prompt・agent 型、SessionEnd の MCP tool、server か tool が空の MCP tool、matcher がコンパイルできない group)を `parseCodexHookSlots` は数え、承認しても警告が消えない(self-review L5)。雛形は command 型だけ。(b) `[hooks.state]` は user config だけを読み、codex が集める他の層は読まない。`.codex/config.toml` の `[hooks]`(TOML の hook)は見ない |
| 「Test gaps of feat/codex-hook-trust-check」 | (a) `CODEX_HOME` が空のときを check の水準で見るテストが無い(tester の probe は overlay で、コミットしていない)。(b) 絞った coverage が通らなかった error の枝。(c) 実機の herdr での起動の 1 行。(d) bare repository の linked worktree の root を codex と比べていない。(e) codex の版が変わったときの検査が無い |

名前はすべて関数、型、テストの名前で、`file:line` は 0 件(`.go:数字` の形などを正規表現で調べた)。参照した名前は `git grep` で実在を確かめた(`checkCodexHookTrust`、`parseCodexHookSlots`、`readCodexHookStates`、`codexLinkedWorktreeMain`、`codexHooksJSONError`、`codexHookTrustSameDir`、`readCodexFileBounded`、`codexSandboxEnvFromOS`、`TestCodexHookTrust_RootOfABareRepositorysLinkedWorktree` ほか)。行の中の evidence と report の path は存在を確かめた。

self-review の L1〜L4(コードの形、重複、定数、文の重なり)と L6〜L9(テストの書き方)は register に足していない。self-review 自身が、直し方が小さく先送りの理由が無いので register に足さず、直さないなら PR 本文の Known gaps に 1 行で書く、としている。下の「残っていること」に書いた。

## I1〜I4

| 項 | 扱い |
|----|----|
| I1 `--help` の「There is no note …」の列挙 | 触らなかった。列挙は「これだけ」とは言っておらず、誤りではない。直すならコード(`codex_hook_trust_notice.go` の定数)と、その文を固定するテストの変更で、この pass の範囲外 |
| I2 `/org` skill の 121〜125 行目(4 つの写し) | 触らなかった。`codex trust .`(project の信頼)の話で今も正しい。skill には触らない指示 |
| I3 計画の D6 の文と実装の 1 行の違い | 触らなかった。D6 は digest の範囲で、S3 の Progress がこの違いを書いている |
| I4 計画の Progress | S6 の項、パイプラインの結果、sync-docs の項を足し、artifact の 3 つの箱を `[x]` にした。Test plan の Edge cases の「`CODEX_HOME` が空」は 473 行目の (a) に入れた。変異 M1〜M4 の記録はこの branch の test report に無い(今回の run は変異テストを回していない)ので、Progress には「handoff の報告に従った」と書いた |

## 確かめたこと(文書が実装と合っていた所)

- `README.md` の Quick start(65〜70 行目)、`ralph org …` の行、`ralph doctor` の行: 動詞 3 つの 1 行、`Codex hook trust` check が読む hooks.json(linked worktree からは main のもの)、鍵が無い・止めてある・設定が無いときの warn、hash を見ないこと、が `doctor_codex_hook_trust.go` と `codex_hook_trust_notice.go` と合う
- `.codex/README.md`(root と雛形): One-time setup の 3 番、Trust UX の 2・4・5 番、doctor の段落。鍵の形 `<hooks.json の絶対パス>:<event の snake_case>:<group>:<handler>` は `codexHookKey` と合う。`cmp` で 2 つは同じ
- `docs/recipes/codex-setup.md`(root と雛形): doctor の段落が新しい check を説明し、古い「per-command-hash」の文は無い。`cmp` で同じ
- `ralph org spawn`・`start`・`director start` の `--help`: 3 つとも `orgCodexHookTrustNoticeHelp` を `Long` の最後に持つ(`org.go` の 2 か所と `org_director_headless.go` の 1 か所)。HEAD から build した binary で `org spawn --help` と `org start --help` を読み、段落が出ることを見た。文は `codex_hook_trust_notice.go` の定数と同じ。本番のコードは `dddb946d` から変わっておらず、verify も 3 つを確かめた
- Check 3 の pass の文「hooks.json wired; confirm `codex trust .` ran for this project (hook approvals: see Codex hook trust)」と `checkCodexEffectiveConfig` の doc: `doctor.go` の 379〜385 行目と 469 行目が合う
- `.codex/AGENTS.override.md`、`.codex/hooks/README.md`、`docs/recipes/codex-setup.md` の 212 行目の hook trust への言及、README の 197 行目: 対話での承認が要ることを言うだけで、doctor が見られないとは言っていない
- `AGENTS.md`、`docs/architecture/repo-map.md`、`docs/quality/quality-gates.md`: doctor の check を 1 つずつ並べておらず、`doctor_codex_hook_trust.go` などのファイル名も挙げていない。足す所は無い(`git grep` で `doctor_codex`、`codex_hook` を探すと tech-debt と spec の言及だけ)
- 計画の D1〜D6 と AC1〜AC9 は verify が照合済み。この pass で本文は変えていない

## 残っていること(lead へ)

- コードの動きとテストは変えていない(変えたのは `.codex/config.toml` の注釈だけ。2 写し)。パイプラインの再走は要らない
- 払っていないもの: 持ち越しの note を足した 6 行の Trigger と、新しい 3 行の中身
- self-review の L1〜L9 は直していない。L1〜L4 と L6〜L9 を直すならコードとテストの変更なので、cross-review の前に 1 commit にまとめるなら、そのあとの `/self-review` からの再走が要る(`.claude/rules/ralph/post-implementation-pipeline.md` の例外は記録だけの修正に限る)。直さずに進めるなら、PR 本文の Known gaps に 1 行ずつ書く。L5 は 472 行目に入れた
- self-review の insights 行は `low: 5` のまま(L6〜L9 を足した更新では足していない)。`/pr` の前に 9 に合わせるなら、記録だけの修正の条件に当たる
- 同じファイル(`docs/tech-debt/README.md`)を別のセッションも編集している。この commit は 6 行への挿入と末尾の 3 行の追加で、既存の行は消していない。merge の衝突は末尾の追加行と、135・153・195・201・371・415 行目の Trigger と Related の末尾で起きうる
- 下流への注意: `.codex/config.toml` は core が所有するファイルなので、この注釈は `ralph upgrade` で下流に配られる(コメントだけの変更で、動きは変わらない)

## Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(`tdcheck.py`)で、変わった行は 135、153、195、201、371、415(どれも `difflib` の操作が `insert` のみ)と、末尾の新しい 3 行(471〜473)。変わった行も新しい行も列は 5、列ごとのバッククォートと `~~` は偶数。`file:line` は 0 件
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` が全部存在する(無い参照 0 件)
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-12-codex-hook-trust-check.md` は編集の前後とも `147d7ec0e7ae`
- `./scripts/check-sync.sh` は `PASS: all files in sync.`、`git diff --check` は rc 0。変えた 4 ファイル(`.codex/config.toml` の 2 つ、tech-debt、計画)の U+FFFD は 0 件
- 実行していないもの: テスト、`run-verify.sh` の全体(コードの動きを変えていない)

## Insight event

```
./scripts/insights-append.sh --slug codex-hook-trust-check --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-codex-hook-trust-check.jsonl` が 3 行から 4 行になり、足した行は cycle 1、pass、ts は 2026-11-12T14:04:26Z。この時点の `git status --short` は `.codex/config.toml`、`templates/base/.codex/config.toml`、tech-debt、計画、既存の jsonl の `M` だけで、迷子のファイルは無かった(この report は未追跡)。
