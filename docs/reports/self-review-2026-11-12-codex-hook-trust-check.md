# Self-review report: codex-hook-trust-check

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(issue #310)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `f36cfc52`、origin/main `d2e5c6e9` を取り込んだ merge の前の変更だけ、18 ファイル、+2732 / -77)。Go の本体 3 本(`codex_hook_trust.go`、`doctor_codex_hook_trust.go`、`codex_hook_trust_notice.go`)、`doctor.go`・`doctor_codex_writable_root.go`・`org.go`・`org_director_headless.go` の diff と呼び出しの周辺、テスト 3 本(`codex_hook_trust_test.go`、`doctor_codex_hook_trust_test.go`、`codex_hook_trust_notice_test.go`)を全部読んだ。README と `.codex/README.md` と recipe は diff を読んだ(`templates/base` の写しは `check-sync` の範囲として読んでいない)。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- 追加の範囲: `1f7ccadb`(テスト 3 ファイルだけ、+231 / -0、本番コードの変更なし)。上の Scope のあとに足されたコミット。読んだ内容と指摘は「Update: 1f7ccadb」の節にまとめた。`9492dc5a`(verify の report)は diff の質の範囲外
- ID の付け方: `L1` から `L5`(LOW)は最初の範囲、`L6` から `L9`(LOW)は `1f7ccadb`

## Evidence reviewed

lead の依頼どおり、この check が利用者の codex の設定を読むので、中身が出力に出る道を先に調べた。コードを読んで追ったもので、実行はしていない。

1. 出力に入る文字列の出どころ。(a) 設定の decode の失敗は `codexConfigDecodeErrorOf`(`codex_hook_trust.go:349`)が行と桁だけを持つ `*codexConfigDecodeError` にする。go-toml の `Error()` は読まない。(b) 設定の読み込みの失敗は `codexConfigReadReason` が `*fs.PathError` の `Err` だけを出し、ほかは `readCodexFileBounded` の固定の文(`not a regular file`、`larger than 1 MiB`)。(c) 鍵の末尾は `codexHookEventLabels` の固定の 14 語と整数。(d) hooks.json の decode の失敗は `codexHooksJSONError` が syntax error の位置と型の不一致の JSON の種類と field 名だけにする。field 名は repo の hooks.json のもので、利用者の codex の設定ではない。テストは 4 つの形(不正な TOML、同じ表の二重定義、`trusted_hash` の型違い、`hooks` が表でない)で、設定に仕込んだ印が `Error()` に出ないことを確かめている(`TestCodexHookTrust_ReadStatesOfAnUndecodableConfigCarriesNoText`)。
2. 開く前の確かめ。`readCodexFileBounded`(`doctor_codex_writable_root.go`)は、通常のファイルでないものを開かず、1 MiB で切る。`.git` のファイルは `codexLinkedWorktreeMain`(`codex_hook_trust.go:216`)が `Lstat` で通常のファイルと確かめ、64 KiB で切ってから読む。FIFO で止まる道は見つからなかった。
3. `readCodexUserConfig` を `readCodexFileBounded` に分けた変更。分ける前後で、設定が無い・stat の失敗・通常のファイルでない・1 MiB 超の 4 つの戻り値の組が同じ(`exists` と `err` の組を 1 つずつ突き合わせた)。
4. 添字と nil。`codexHookTrustApprovalStep` の `roots[0]` は、呼び出しが `HooksState == codexHookSourceRead` のあとだけで、そのとき `codexHookTrustFor` が `Roots` を入れている(関数の注釈にも書いてある)。`tails[:codexHookTrustMaxTails]` への `append` は、`tails` がその場で作った slice なので他へ影響しない。
5. 呼び出しの位置。spawn、start の task の形、`--plan` の形、director start のどれも、driver を解決した直後に 1 回だけ呼ぶ。spawn と start の task の形は `--cwd` が空なら先に断る(`org: --cwd is required`)ので、空の dir は届かない。
6. codex 側の判定との突き合わせ。`rust-v0.166.0` の `codex-rs/hooks/src/engine/discovery.rs` を `gh api` で読んだ(読み取りだけ)。`hook_trust_status`(`Some("")` は trusted と違うので Modified)、`hook_enabled`(`enabled == Some(false)` だけが止める)、`hook_hash`(event 名、matcher、正規化した handler で `timeout_sec` を含む)は、`codexHookTrustUnder` の分け方と `.codex/README.md` の「hash は宣言全体」の文に合う。

調べて、指摘にしなかったこと。

- spawn の 1 行は `rt.Spawn` の前に出るので、すでに起こした座席への再実行と、watch gate に断られる spawn でも出る。plan の D5 が `--dry-run` でも出すと決めていて、`printLegacyModelPoolNotice` と同じ位置。設計の範囲で、diff の質の指摘ではない。
- `info` の文「(the same file Codex effective config checks)」は、`.codex/config.toml` が無いとき Check 3 が hooks.json を読まずに戻る(`doctor.go:394-397`)ので、その場合は Check 3 は hooks.json の欠けを言わない。ただしそのとき Check 3 自身が `.codex/config.toml not found` を warn で言うので、利用者に何も知らせない道は残らない。`checkCodexHookTrust` の注釈が同じ事情を書いている。
- Go の `encoding/json` は鍵の大文字小文字を区別せずに struct の欄へ当てる(`"Hooks"` も読む)が、codex の serde は区別する。hooks.json にそう書く人はほぼいないこと、結果が警告に倒れることから、指摘にしない。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L1 | LOW | maintainability | `codexHookTrustItem.Keys` と `codexHookTrustReport.Approved` は、本番のコードに読み手がいない。`classifyCodexHookSlots` が各 slot について root ごとの鍵を組み立てて `Keys` に入れ、承認済みの一覧も作るが、doctor は pass の文に `len(report.Slots)`、gap の文に `NotApproved` と `Off` だけを使い、通知は `NotApproved` と `Off` の長さだけを使う。この 2 つを読むのはテストだけ。読み手のないデータに、テストが固定する形ができている | 生産側: `codex_hook_trust.go:408`(`Keys`)、`:466-469`、`:472`(`approved = append`)。本番の読み手: `.Keys` は 0 件(`doctor.go` の `maps.Keys` は別物)、`report.Approved` は 0 件。テスト側の読み手: `codex_hook_trust_test.go:557`、`:566-567`、`:585`、`:618`、`:681` | `Keys` と、そのための root の内側のループを消す。承認の手順の文に鍵そのものを出したい意図があるなら、doctor の文がそれを使う形にする。`Approved` は pass の文の件数に使うか、使わないなら消す |
| L2 | LOW | maintainability | 「`<n>` of `<m>` hooks in `<hooks.json>` are not approved in `<config>`, `<k>` turned off」の 3 通りの組み立て(未承認と止めてあるの両方、未承認だけ、止めてあるだけ)の書式が、doctor の `codexHookTrustGapDetail` と通知の `codexHookTrustNotice` に同じ文字列で 2 つある。通知にだけ「設定が無い」の 1 通りがある。片方の文言を直すと、もう片方は黙って古いまま残る。plan は通知が doctor と同じ件数を言うと決めている | `doctor_codex_hook_trust.go:154-161` と `codex_hook_trust_notice.go:59-65`(書式 3 つが一字違わない) | 件数の句を返す小さい関数(例 `codexHookTrustCountsClause(report, cfgDisplay)`)を 1 つ作り、両方から呼ぶ。通知の「設定が無い」の分岐だけが外に残る |
| L3 | LOW | naming | 定数 `codexHookTrustDirectorVerb` を呼び出し側が使っていない。注釈は「`ralph org director start` が `printCodexHookTrustNotice` に渡す verb」と言うが、呼び出しは文字列 `"org director start"` を直接渡している(隣の `printScaffoldVersionNotice` の呼び出しと同じ書き方)。`codexHookTrustAgent` は定数と比べるので、どちらかの綴りを直すと director の行が黙って「codex seat director」になる。`TestCodexHookTrustNotice_DirectorStartPrintsOneLineForACodexDirector` が出力を固定しているので、ずれは CI で気づける | 定義と比較だけ: `codex_hook_trust_notice.go:86`、`:94`。呼び出し: `org_director_headless.go:366`。定数の参照は `internal/` 全体でこの 2 か所だけ | 呼び出しを `codexHookTrustDirectorVerb` にする。または、定数をやめて `codexHookTrustAgent` が `seat == org.DirectorRole` かつ別の印で見分ける形にする |
| L4 | LOW | readability | hooks.json が JSON でないとき、doctor の文が「is not valid (not valid JSON (at byte 1))」と、同じ語を括弧の中で繰り返す。warn の文は「codex seats read hooks from X, which is not valid (not valid JSON (at byte 1)), so no hook runs in a codex seat」になり、括弧が 2 重になる。型の不一致(`is not valid (unexpected JSON array at "hooks")`)は読める。テストがこの重なった文をそのまま固定している | `doctor_codex_hook_trust.go:98`(`is not valid (%v)`)と `codex_hook_trust.go:137`(`not valid JSON (at byte %d)`)。固定: `doctor_codex_hook_trust_test.go:129` | `codexHooksJSONError` の syntax error の文を `syntax error at byte %d` のように「not valid」を含まない形にし、テストの期待を合わせる |
| L5 | LOW | readability | `parseCodexHookSlots` の注釈(と `codexHookTrustReport.Slots` の注釈の「every handler codex asks approval for」)は、承認を求められる handler を数えると言うが、コードは「command 型で command が空」以外のすべての handler を数える。plan の S1 の記録によると、codex は prompt 型と agent 型の handler、SessionEnd の MCP tool、server か tool が空の MCP tool、matcher がコンパイルできない group の handler には鍵を作らない。そういう handler を持つ hooks.json では、codex で承認しても鍵ができないので、doctor の warn と起動時の 1 行が消えない。ralph の雛形は command 型だけで、結果は知らせる側に倒れるので重くはない。この差は plan の進捗の欄にしか無く、コードの注釈にも `docs/tech-debt/README.md` にも無い(この diff は register を変えていない) | `codex_hook_trust.go:96-101`、`:118`、`:422`。plan の Progress checklist の S1 の項 | 注釈に「数えすぎることがある(上の種類の handler は数えるが、codex は鍵を作らない)」と書く。register の行は plan が tech-debt と Known gaps に書くと言っているので、`/sync-docs` で足す(下の表) |

## Positive notes

- 失敗の作り方が、利用者の権限に依らない。読めない設定や hooks.json を、`chmod` ではなくディレクトリを置いて作っている(`TestCodexHookTrust_ReadStatesOfAnUnreadableConfig`、`TestCodexHookTrustFor_ReportsEachInputApart`)。root で動く CI でも同じに落ちる。
- 利用者の `~/.codex` を読まない。コアのテストは設定の path を引数で渡し、通知のテストは `doctorCodexSandboxEnv` を差し替えて `t.Cleanup` で戻す。`TestMain` の固定とも合う。
- 相対の dir と `$PWD` が symlink の綴りのときに、実パスの候補が落ちる不具合(S5)を、`chdirCodexHookTrustAs` で `os.Getwd` が綴りを返すことを確かめてから再現するテストにしている。helper が前提を `t.Fatalf` で守るので、環境が違うときに通るだけの空のテストにならない。
- 両方の綴りの承認が要る規則(AC9)を、`classifyCodexHookSlot` の単体(5 行)、`codexHookTrustFor`(3 行)、doctor(実パス・symlink の 2 つの入口 × 3 行)の 3 段で、片方だけの承認が両方向で warn になることまで固定している。
- 失敗が黙らない。hooks.json が読めない・壊れているとき、Check 3 と同じファイルなら info、別のファイル(linked worktree から見た main の hooks.json)なら warn と分けている(`codexHookTrustHooksProblem`)。起動時の 1 行は、doctor が言う場合(設定が読めない・hooks.json が無い)に出さず、同じことを二重に言わない。
- codex の規則を写した箇所すべてに、版(`rust-v0.166.0`)とファイルが注釈に書いてある。root の求め方、鍵の形、番号の数え方、hash を確かめない理由がたどれる。

## Coverage gaps

- テストは走らせていない(`/test` の範囲)。`go vet` も `gofmt` も見ていない。
- `templates/base/.codex/README.md` と `templates/base/docs/recipes/codex-setup.md` は、root 側との一致を `check-sync` に任せ、読んでいない。
- 既存の org の CLI テスト(`TestOrgSpawn_RoleModels` など)が、codex に解決して `.codex/hooks.json` のある cwd を使うとき 1 行が増えて stderr の完全一致が落ちるという plan の Risks は、読むだけでは確かめられない。package 全体の実行(`/test`)の範囲。

## Update: 1f7ccadb(テストだけの追加)

`git show 1f7ccadb` の diff を読んだ。`codex_hook_trust_notice_test.go`(+139)、`codex_hook_trust_test.go`(+47)、`doctor_codex_hook_trust_test.go`(+45)の 3 ファイルで、本番コードには触れていない。追加は 6 本。spawn を symlink 経由の `--cwd` で打つ 1 本(6 行の表)、claude の leader・`start --plan`・director では 1 行も出ないことの 3 本、bare repository の linked worktree の root の 1 本、main の hooks.json が読めないときの doctor の 1 本(2 行の表)。周りとして、使われる helper(`chdirCodexHookTrustAs`、`symlinkCodexHookTrust`、`isolateFeatureGitEnv`、`newLegacyLedgerRepo`、`directorStartLedger`)、本番側の `codexHookRootAt`、`codexLinkedWorktreeMain`、`readCodexFileBounded`、`codexConfigReadReason`、同じ種類のテスト(`internal/config/local_test.go`、`internal/upgrade/replaceplan_test.go` の mode 000)を読んだ。テストは走らせていない。

### Evidence reviewed

1. flaky になりうる点を先に追った。
   - symlink: 6 つの subtest がそれぞれ temp tree、stub PATH、codex home を作り、共有する状態がない。相対の形は `chdirCodexHookTrustAs` が `os.Getwd` の綴りを `t.Fatalf` で確かめるので、`$PWD` が効かない環境で空の通過にならない。symlink が作れない環境は `symlinkCodexHookTrust` が Skip する。`t.Chdir` は subtest の Cleanup で戻る。
   - mode 000: root で Skip、`chmod`、Cleanup で戻す、という形は repo の既存の書き方(`internal/config/local_test.go:350-357`、`internal/upgrade/replaceplan_test.go:700-709`)と同じで、そこに「`os.Open` が通ってしまう環境(権限を見ない FS)は Skip」が足してある。Cleanup は subtest の `t` に登録され、`newLegacyLedgerRepo` の `t.TempDir()` より後なので、LIFO で chmod の戻しが先に動いてから temp dir が消える。root で Skip したとき、読めない場合の本体は「a directory」の行が通すので、検査が丸ごと抜けることはない。
   - bare repository: `isolateFeatureGitEnv` が `GIT_CONFIG_GLOBAL=/dev/null` と GIT_DIR 系の unset を行う。`git clone --bare` と `worktree add` は local の path だけで、network も sleep も時刻もない。`<tmp>/.git` が無いという前提は、`tmp` が新しい temp dir で `src`・`store.git`・`wt-a`・`proj`・`wt-b` だけを持つことで決まる。
2. 本番の規則との突き合わせ。`codexLinkedWorktreeMain` は「`<main>/.git` が存在するときだけ main を返す」(`codex_hook_trust.go:237-241`)。bare repository の 2 通り(`store.git` は親に `.git` が無いので worktree 自身、`proj/.git` は親 `proj` が root)は、その分岐の両側を突いている。テストの注釈の文は本番の注釈と合う。
3. 弁別力。本番の分岐とテストの表を突き合わせて読んだ(変異は走らせていない)。symlink の表は、cwd を先に実パスへ解決する変更なら「real only」が通って行が消えるので落ち、symlink の綴りだけを見る変更なら「link only」が通るので落ちる。「both」は 2 つの綴りの鍵がそろうと行が出ないことを固定する。claude の 3 本は、`start <task>` が同じ repository の codex の起動で行が出ることを同じテストの中で確かめる control を持つ(fixture が検査に届く)。director の 1 本は started 行まで見て、起動が止まって stderr が空になった偽の通過を避けている。

### Findings (1f7ccadb)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L6 | LOW | readability | `TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings` は、承認の 3 通り(real のみ、link のみ、両方)と cwd の 2 通り(絶対、相対の `.`)の積 6 行を、行ごとに鍵の closure を書いて並べている。同じ closure が 2 回ずつあり、`func(r, l string)` の引数は行によって片方が使われず、`l` は `1` と見分けにくい。`const pre, start` は同じファイルの 66 行目の写し。期待する 1 行も、同じファイルの表の「codex, a config with no approvals」(80-81 行目)と一字違わない。文言を直すとき 2 か所を直す必要があり、片方を忘れるとテストが黙って別の文を固定する(L2 で言った本番側の 2 通りの組み立ての、テスト側の写し) | `codex_hook_trust_notice_test.go:174`(const)、`:181-186`(表)、`:216-219`(期待する文)、`:66` と `:80-81`(元の写し) | 承認の 3 通りと cwd の 2 通りを別の小さい表にして入れ子で回し、鍵は `[]string{"real", "link"}` のような集合から組み立てる。const は package に 1 つ。期待する文は `codexHookTrustNoticeUnapproved` の隣に件数つきの helper を置いて 2 か所から呼ぶ |
| L7 | LOW | readability | `TestCodexHookTrustNotice_StartPlanWithClaudePrintsNothing` は `strings.Contains(stderr, "codex")` で見ているが、同じ commit の他の 2 本(`stderr != ""`)と、本物の対(`StartPlanChecksTheMainCheckout`、stderr 全体の一致)は完全一致で見ている。この対は 1 行だけを出すので、claude の `start --plan` の stderr も空のはずで、`stderr != ""` で足りる。部分文字列だと、将来この経路に別の note が足されたとき、関係のない行でこのテストが落ちる。また fixture の 7 行は対と一字違わず、「codex の対では行が出る」ことを確かめる control もこのテストの中にない(plain の start の claude 版は control を持つ)。plain の start の control の `t.Fatalf("start with codex: %v", err)` は stdout と stderr を捨てている(`_`)ので、落ちたとき理由が出ない。1 つ目の呼び出しは両方を出している | `codex_hook_trust_notice_test.go:323-341`(7 行の fixture は `:293-303` と同じ)、`:338`、`:278-282` | `stderr != ""` にそろえ、fixture は helper(例 `newCodexHookTrustFeatureStart(t)`)に出して対と共有する。control の呼び出しは stdout・stderr を受けて失敗の文に入れる |
| L8 | LOW | readability | `...WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksAreUnreadable` の 2 行は、どちらも共通の前置き `could not be read (` しか固定しない。repo の同じ種類のテスト(`internal/config/local_test.go:344-358`)は行ごとに理由(`is a directory`、`permission denied`)を持ち、同じファイルの兄弟(`InfoWhenItsOwnHooksJSONCannotBeRead`)も `(not a regular file)` まで固定する。mode 000 の行は「`open` が失敗する枝」を通すために足した行のはずで、理由を固定しないと、その枝の文言が壊れても通る(directory の行の `not a regular file` と見分けが付かない)。同じ兄弟(`...CannotBeRead`)が warn の行に渡す `notWant`(`Codex effective config`)もここでは `nil`。名前も紛らわしい: 兄弟は `CannotBeRead`(中身は「無い」と「形が違う」)、こちらは `AreUnreadable` で、`go test -run` の出力では区別が付かず、名前に main 自身から見ると info という半分も入っていない。前置き(`newLegacyLedgerRepo`、worktree の hooks.json、`t.TempDir()` を呼び出しのたびに作る `resolveEnv`、`{worktree, "."}` の回し)も兄弟の写し | `doctor_codex_hook_trust_test.go:191-234`(固定する文は `:226-232`)、兄弟 `:160-189`(`:177` の `resolveEnv`、`:185` の `notWant`)、`:128` | 行に `reason` の列を足す(directory は `not a regular file`、mode 000 は `permission denied`)。warn の行に兄弟と同じ `notWant` を渡す。名前を、兄弟と見分けが付くもの(読めない理由が開けないことだと言える形)に変える。`resolveEnv` の 3 行は兄弟と 1 つにまとめる。「a directory」の行は、既に `"<dir>"` の合図を使う兄弟の表の行にもできる |
| L9 | LOW | maintainability | `TestCodexHookTrust_RootOfABareRepositorysLinkedWorktree` は、2 つの場面(親に `.git` が無い `store.git`、親が `.git` を持つ `proj/.git`)を 1 つの関数に `t.Run` なしで並べている。場面 A の `codexHookRootCandidates` は見ず(B だけ見る)、A の途中の `t.Fatalf` は B を隠す。名前の `Repositorys` は複数形の綴りが違う(同じファイルの兄弟は `Checkouts` のように所有の `'s` を落としているだけ)。空の commit の 2 行(`-c user.name=t -c user.email=t@example.com -c commit.gpgsign=false ... --allow-empty`)は、同じ package にある `gateCLICommit(t, dir, msg)`(`org_gate_test.go:139-143`)と同じ呼び出しを手で書いている。`newLegacyLedgerRepo` と `config_test.go:342` の写しに続く 3 つ目の写し | `codex_hook_trust_test.go:292-332`(`:296-298` が写し、`:305-311` と `:318-331` が非対称)。写し元: `org_gate_test.go:139-143`。ほかの写し: `org_legacy_ledger_test.go:69-71`、`config_test.go:342` | 場面ごとに `t.Run` に分け、両方で `codexHookRoot` と `codexHookRootCandidates` を見る。名前は `...RootOfALinkedWorktreeOfABareRepository`。空の commit は `gateCLICommit(t, src, "init")` を呼ぶ |

### Positive notes (1f7ccadb)

- 結果が環境に依らない。mode 000 は repo の既存の書き方に沿い、root と、権限を見ない FS の両方で Skip し、どちらでも directory の行が同じ関数を通す。bare repository のテストは、developer の git の設定を `isolateFeatureGitEnv` で切っている。
- 「出ない」ことを言うテストが、出る側を同じテストの中(`StartWithClaudePrintsNothing`)か、同じ fixture の対のテスト(`StartPlanChecksTheMainCheckout`、`DirectorStartPrintsOneLineForACodexDirector`)に持つ。fixture が検査まで届かなくて空のまま通る形になっていない。
- symlink の表は、読んだ限り、cwd を先に実パスへ解決する本番の変更と、symlink の綴りだけを見る変更の両方で落ちる形をしている。S5 の不具合(相対の `.` と `$PWD`)を、spawn の起動の行まで通して固定した。

### Coverage gaps (1f7ccadb)

- テストは走らせていない(`/test` の範囲)。mode 000 の行は root で動く CI では Skip になるので、`/test` の結果で skip と pass を見分けたい。
- bare repository の 2 つの場面で期待する root は、本番の注釈と `codexLinkedWorktreeMain` を読んで確かめた。codex 本体の `resolve_root` は今回読み直していない(最初の範囲で `rust-v0.166.0` の `discovery.rs` だけ読んだ)。このテストの注釈は版もファイルも書かず、本番の注釈(`codex_hook_trust.go:155-156`)に任せている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| prompt 型・agent 型の handler、SessionEnd の MCP tool、server か tool が空の MCP tool、matcher がコンパイルできない group の handler を、codex は承認の対象にしないが、`parseCodexHookSlots` は数える(L5) | そういう handler を持つ hooks.json で、承認しても消えない warn と起動時の 1 行が出る | ralph の雛形は command 型だけで、知らせる側に倒れる差なので | 雛形が command 型以外の handler を使うとき、または下流の利用者がこの warn を消せないと報告したとき | `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(S1 の記録、Scope) |
| hash の食い違い(承認のあとで変わった hook)を、座席の pane に出る codex の「Hooks need review」(`internal/org/resume_screens.go` がすでに選択の画面として見分ける)から見張りが見つけて知らせる案。今は doctor の pass の文と `.codex/README.md` に書くだけ | 承認のあとに hook が変わると、doctor は pass のまま、座席は 1 つも動かない | plan の Non-goals(ユーザーの決定で後続) | 下流で、承認の食い違いが起きて hook が黙って効かない報告があるとき | `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(Scope の最後の項、Non-goals) |

_この 2 行は plan の Scope が tech-debt に書くと決めているもの。lead の頼みはレポートだけを 1 コミットにすることなので、`docs/tech-debt/README.md` にはまだ足していない。`/sync-docs` で足す。_

`1f7ccadb` の L6 から L9 はテストの書き方の指摘で、どれも直し方が小さく、先送りの理由もない。register には足さない。直さずに進める場合は、PR 本文の Known gaps に 1 行で書く。

## Recommendation

- Merge: OK(CRITICAL・HIGH・MEDIUM なし、LOW 9: `f36cfc52` までで 5、`1f7ccadb` で 4)。利用者の codex の設定の中身が doctor の文にも起動時の 1 行にも出る道は見つからなかった。`1f7ccadb` はテストだけの追加で、flaky になりうる点(mode 000、symlink、bare repository の git)は repo の既存の書き方か helper の守りで抑えられており、Merge の判定を変える指摘はない。
- Follow-ups: L1 から L9 は、このブランチに足す次のコミットで直せる。どれも判定や出力の意味を変えないので、パイプラインを回し直す理由にならない。直すなら cross-review の前に 1 コミットにまとめる(L4 は期待する文が変わるのでテスト 1 行も直す。L6 と L7 と L8 は同じテストファイルの整理なので、L2 の件数の句の helper と一緒に直すと期待する文の重複が 1 回で片付く)。L5 の register の行は `/sync-docs` で足す。
- 記録の食い違い: `docs/insights/events/2026-11-12-codex-hook-trust-check.jsonl` の self_review の行は `low: 5` のまま。この更新では足していない(`internal/insights` に、同じ phase と cycle の行を後のもので置き換える処理は見当たらなかった。行ごとに数えるなら 2 行目を足すと self_review が 2 回に見える。集計までは読んでいない)。9 に合わせるなら、記録だけの修正(pipeline の例外の条件 1 から 3 を満たす)で直すか、cycle の終わりにまとめて足す。
