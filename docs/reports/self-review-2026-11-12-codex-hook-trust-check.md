# Self-review report: codex-hook-trust-check

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(issue #310)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `f36cfc52`、origin/main `d2e5c6e9` を取り込んだ merge の前の変更だけ、18 ファイル、+2732 / -77)。Go の本体 3 本(`codex_hook_trust.go`、`doctor_codex_hook_trust.go`、`codex_hook_trust_notice.go`)、`doctor.go`・`doctor_codex_writable_root.go`・`org.go`・`org_director_headless.go` の diff と呼び出しの周辺、テスト 3 本(`codex_hook_trust_test.go`、`doctor_codex_hook_trust_test.go`、`codex_hook_trust_notice_test.go`)を全部読んだ。README と `.codex/README.md` と recipe は diff を読んだ(`templates/base` の写しは `check-sync` の範囲として読んでいない)。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- 追加の範囲: `1f7ccadb`(テスト 3 ファイルだけ、+231 / -0、本番コードの変更なし)。上の Scope のあとに足されたコミット。読んだ内容と指摘は「Update: 1f7ccadb」の節にまとめた。`9492dc5a`(verify の report)は diff の質の範囲外
- ID の付け方: `L1` から `L5`(LOW)は最初の範囲、`L6` から `L9`(LOW)は `1f7ccadb`。cycle 2 の指摘は `C2-1` から(末尾の「Cycle 2」の節)
- Cycle 2: 末尾の「Cycle 2」の節に、`9e72d49a` と `f71a2a33` の指摘と、L1 から L9 の現在の状態・場所を書いた。この上の節(cycle 1)の `file:line` は、`9e72d49a` で `codex_hook_trust.go` を書き直したあとの行とずれているので、読むときは cycle 2 の節の「L1 から L9 の現在地」の表を使う。cycle 1 の L1 から L9 のうち、`9e72d49a` と `f71a2a33` で解決したものはない

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

## Cycle 2

- Date: 2026-11-13
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(`cycle-count.json` は cycle 2、既定の上限 2 回)。cross-review cycle 1 の ACTION_REQUIRED 3 件の直しのあと
- Scope: `git diff origin/main...HEAD`(HEAD `83ae4e55`、origin/main `0936aaea` を `d9103dfc` で取り込み済み)のうち、cycle 1 のあとに足された `9e72d49a`(入れ子の層、null の形、持ち主の確かめ。8 ファイル、+1537 / -314)と `f71a2a33`(パーサーの移動、4 ファイル)。`83ae4e55` は plan の記録 1 行で、diff の質の対象にしていない
- 全部読んだもの: `codex_hook_trust.go`(651 行)、`codex_hook_trust_parse.go`(455 行)、`doctor_codex_hook_trust.go`(263 行)、`codex_hook_trust_notice.go`(120 行)、`codex_hook_trust_parse_test.go`(216 行)、`codex_hook_trust_test.go` の 1 から 439 行と 540 から 892 行。`codex_hook_trust_notice_test.go` と `doctor_codex_hook_trust_test.go` は `9e72d49a` の diff(+65、+122)だけ読み、残りは cycle 1 で読んだまま。`codex_hook_trust_test.go` の 440 から 539 行(設定を読むテスト)は cycle 1 から変わっておらず、読み直していない。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: `C2-1` から `C2-8`(すべて LOW)。cycle 1 の `L1` から `L9` は番号を変えない(register の行が `L5` を指している)

### Evidence reviewed

1. cycle 1 の指摘を 1 件ずつ HEAD で確かめた。`grep` と該当ファイルの読みで、`9e72d49a` と `f71a2a33` が触れた指摘は無かった(下の「L1 から L9 の現在地」)。cycle 1 の報告が「直すなら cross-review の前に 1 コミットで」と書いたとおりにはなっていない
2. `f71a2a33` が移すだけであることを確かめた。`go/parser` の小さい program で、`9e72d49a` の `codex_hook_trust.go` と `codex_hook_trust_test.go`、HEAD の 4 ファイル(本体、パーサー、テスト 2 本)について、各ファイルのトップレベルの宣言を doc コメントつきで切り出して並べ替え、`cmp` した。107 件の宣言が 1 件ずつ一致した(2141 行、差なし)。宣言の外のコメント(ファイルの頭の説明)だけが新しい。import の過不足は `/verify` の範囲
3. 利用者の設定の中身が出る道。cycle 1 の道(decode は位置だけ、読みの失敗は `*fs.PathError.Err` か固定の文)は `9e72d49a` で変わっていない。足されたコードのうち、`.git` と worktree の登録のファイル(`gitdir`、`commondir`)は `codexReadGitMetadata` が bool だけを返し、中身も読み込みの失敗の理由も外に出さない。hooks.json の decode の失敗は、新しいパーサーが自分で文を作る。その文に hooks.json の文字列が入る(C2-2)。hooks.json は repo のファイルで、codex の設定ではない
4. 開く前の確かめ。`codexHookRootAt` は `os.Stat` で `.git` を見て、ディレクトリでなければ `codexLinkedWorktreeMain` に渡す。その先の読みはすべて `codexReadGitMetadata` の `Lstat`(通常のファイルでなければ開かない、64 KiB で切る)を通るので、FIFO で止まる道は見つからなかった。`codexMainOwnsCommonDir` も同じ
5. 注釈と動きの突き合わせ。`codexLinkedWorktreeMain`(「main は gitdir の 3 つ上」)、`codexWorktreeRegisteredAt`、`codexMainOwnsCommonDir`、`codexHookRootAt`、`HooksProblems`、`ItemName`、`ApprovalDirs`、`layers` の注釈を 1 文ずつコードと読み合わせ、合っていた。持ち主の確かめのテスト表(12 行)を、`codexLinkedWorktreeMain` の分岐ごとに突き合わせた。symlink の gitdir、gitdir の欠け・空・別の checkout・`.git` で終わらない名前、commondir の欠け・別のディレクトリ、bare の親に `.git` が無い・別の repo の `.git`、`--separate-git-dir` の main(名指しが合う・合わない)。どの分岐も落ちる行を持つ。`gitdir` が `worktrees` の下でない形は `RootFollowsCodexsWalk` の submodule の行にある
6. codex 側の型との突き合わせ。`gh api`(読み取りだけ)で `rust-v0.166.0` の `codex-rs/config/src/hook_config.rs`(`HooksFile`: `deny_unknown_fields`、`hooks` は `serde(default)`)と `codex-rs/protocol/src/hooks.rs`(`HookHandlerConfig` の 4 つの形、`commandWindows` の別名、`async: bool`、`deserialize_mcp_tool_input` の `representable`)を読み、`codexHookHandlerTypes` の欄と型、重複の扱い、MCP の `input` の確かめ(null なし、2^63-1 超なし)が合うことを確かめた

調べて、指摘にしなかったこと。

- `codexHookLayersOf` は 2 つの綴りの層を `Rel` で突き合わせる。cwd の下に別の場所を指す symlink のディレクトリがあると、2 つの綴りで `Rel` が違い、層が別々に数えられる。ralph の雛形にも、座席に渡す cwd の作り方にも、この形は見つからない。起きても片方の綴りだけで承認を見るだけで、警告が消える向きではない
- `ItemName` は項目ごとに `SlotPaths()` を作り直す。件数は小さいので指摘にしない
- `commandWindows` の別名や `async` の型の書き方は、codex の型と合う(上の 6)。指摘にする差は見つからなかった

### Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | maintainability | cycle 1 の L1 から L4 と L6 から L9 の 8 件が、上限(2 回)の最後の回で未解決のまま残る。cycle 1 の報告は「register には足さない。直さずに進める場合は PR 本文の Known gaps に 1 行で書く」と言っているので、PR 本文が唯一の置き場になる。register は L5 の行だけ(`23a57869`)。さらに `9e72d49a` のあとで register の 2 行に食い違いが出た。(1)「Test gaps of feat/codex-hook-trust-check」の (b) は「`codexHooksJSONError` の decode error で syntax error でも type error でもないもの」を挙げるが、type error の分岐は `9e72d49a` で消えた(`git show 8502c846:internal/cli/codex_hook_trust.go` の 139 行に `UnmarshalTypeError`、HEAD の `codex_hook_trust*.go` と doctor には 0 件)。(2)「Where the Codex hook trust check … reads differently from codex」は (a) と (b) だけで、`9e72d49a` で増えた差(serde は JSON の配列からも同じ形の object を読むが、ralph は拒み、codex が読める hooks.json を invalid と言う)は `parseCodexHookSlots` の注釈と plan の進捗にしか無い。invalid になると起動時の 1 行も出ない(`Classified()` が偽)。L5 と同じ形(差が register の外にある)で、向きは知らせる側。なお cycle 1 の「Tech debt identified」の末尾の注記(register にまだ足していない)は `23a57869` で解消している(2 行とも足された) | 8 件が HEAD で未解決であることは、本体と test の `grep` で確かめた(下の「L1 から L9 の現在地」)。register: `docs/tech-debt/README.md` の「Test gaps of feat/codex-hook-trust-check」の行の (b)、「Where the Codex hook trust check」の行。注釈: `codex_hook_trust_parse.go:59-62` | 直せるものは verify と test の前に直す(下の Recommendation)。直さないものは、L1 から L4 と L6 から L9 を 1 行の register(または PR 本文の Known gaps)にまとめ、trigger を書く。register の 2 行は `/sync-docs` で直す。(1) は「syntax error でないもの」にし、(2) は (c) として配列の形を足す。この run は report だけのコミットなので register には触れていない |
| C2-2 | LOW | security | hooks.json の decode の失敗の文に、hooks.json の中の文字列が長さの制限なしで入る。`unknown key %q at the top level`(`codex_hook_trust_parse.go:89`)、`duplicate key %q`(`:228`、`:310`)、`unknown handler type %q`(`:264`)、MCP の `input` の鍵(`:404` の `codexJSONPath(at, key)` が path に足す)。文は `codexHookTrustLayerProblem`(`doctor_codex_hook_trust.go:134` の `is not valid (%v)`)から doctor の Detail の 1 行に入る。hooks.json は 1 MiB まで読むので、1 つの鍵が長ければ、その長さの 1 行が doctor に出る。hooks.json は repo のファイルで codex の設定ではなく、秘密を出す道は見つからなかった。ただ「報告の文に入力の中身を入れない」という cycle 1 の作りと違い、hooks.json に限っては中身が入る | 上の行。`codexHookTrustLayerProblem` の呼び出し元は `codexHookTrustHooksProblem`(`doctor_codex_hook_trust.go:102`、`:105`) | `%q` の 4 か所を、長さを切る小さい関数(例 `codexJSONQuoted(s)` が 40 文字で切る)に通す。または鍵の位置(`hooks.PreToolUse.0.hooks.1`)だけを出し、利用者が付けた名前は出さない |
| C2-3 | LOW | readability | `parseCodexHookSlots` の先頭の `switch kind := codexJSONKind(raw); kind` は、`codexJSONUnexpected(raw, "")` を手で展開したもので、null だけ別の文(`not a JSON object`)にしている。同じファイルの null の文は他のすべてが `unexpected JSON null at "<場所>"` で、先頭だけが違う。2 つの文を、2 本のテストが別々に固定している(`codex_hook_trust_parse_test.go:100` と `:101`) | `codex_hook_trust_parse.go:70-76`(手で作る文)、`:436-438`(`codexJSONUnexpected`。`at == ""` なら `at the top level` を出すので、同じ文になる) | `if codexJSONKind(raw) != "object" { return nil, codexJSONUnexpected(raw, "") }` の 3 行にし、null の期待を `unexpected JSON null at the top level` に合わせる |
| C2-4 | LOW | readability | `9e72d49a` の書き直しのあとで、前の形を言ったままの名前と注釈が 3 つ残る。(a) `codexHookTrustMaxTails` とその注釈「how many key tails」は、`9e72d49a` で名前を出す側が `ItemName`(入れ子の層では `sub/.codex/hooks.json:stop:0:0`)になったのに、tail の数と言う。同じ commit で使う側の変数は `tails` から `names` に直っている。(b) `codexHooksJSONError` の注釈「without Go's type names: the byte offset of a syntax error」は、type error の分岐を `9e72d49a` で消したあとの残りの `return err` は、`json.Unmarshal` を `json.RawMessage` に向けた呼び出しでは届かない(読みでは syntax error だけが返る。実行はしていない)。注釈の目的(Go の型名を出さない)と、素のエラーを返す分岐が食い違う。(c) L5 の過大な言い方が、新しい `parseCodexHookHandler` の注釈「every handler of a known type does, except …」に平らな文で入り、`codexHookHandlerTypes` の `"prompt": {}`、`"agent": {}` にも、register の行(prompt と agent の handler には codex が鍵を作らない)への指しがない。テストの注釈(`codex_hook_trust_parse_test.go:208-210`)は「plan の進捗に記録した」と指すが、plan は `/pr` で archive へ移る。cross-review の cycle 1 の直しで L5 が広がった | (a) `doctor_codex_hook_trust.go:172-174`、`:197-198`。(b) `codex_hook_trust_parse.go:271-279`。(c) `codex_hook_trust_parse.go:44-62`、`:179-202`、`:204-206`、`codex_hook_trust_parse_test.go:208-210` | (a) `codexHookTrustMaxNames` にして注釈を「hooks」にする。(b) 分岐を syntax error だけにして、他は呼び出し側で扱う形にするか、注釈を「syntax error の位置。他は素のまま返す(json.RawMessage には届かない)」にする。(c) `codexHookHandlerTypes` の注釈に 1 文「prompt と agent の handler には codex は鍵を作らないが、ここでは数える(register の「reads differently from codex」の (a))」を足し、テストの注釈の指し先を register にする |
| C2-5 | LOW | readability | hooks.json の層に問題があり、別の層に hook がある場合の doctor の warn が、その別の層の承認を見ていないことを言わない。`Classified()` が偽になると承認の仕分けは全部飛ぶ。同じ事情の「設定が読めない」の文は「the approvals of the … were not checked」と言い、info の文は「hook approvals not checked」と言うが、warn の文は「so no hook from it runs in a codex seat」だけで、root の hook が未承認でも何も言わない。直したあとで初めて root の未承認が出る。同じ関数で、層の dir を、自分で `Dirs[0]` から作った path から `filepath.Dir(filepath.Dir(path))` で戻している | `doctor_codex_hook_trust.go:96-122`(戻しは `:101`、文の組み立ては `:112-120`)。テスト: `doctor_codex_hook_trust_test.go:453-455` の「the root's hooks still run」は注釈で、warn の文は何も言わない | `layer.Dirs[0]` を直接使う。`report.SlotCount() > 0` のとき、warn の文の終わりに「approvals of the other hooks.json were not checked」を足す |
| C2-6 | LOW | readability | `TestCodexHookTrustNotice_PrintsNothingForAHooksJSONCodexRefuses` は「出ない」ことだけを見て、出る側の対を持たない。2 つ目の場面は、root が未承認で、入れ子の hooks.json が null の handler を持つ。入れ子のファイルが無ければ「no codex user config … to approve the 2 hooks」の 1 行が出る形だが、それを同じテストの中で確かめていないので、fixture が検査に届かなくなっても通る(cycle 1 の L7 と同じ形)。2 つの場面は 1 つの `bytes.Buffer` を共有し、2 つ目の失敗の文に 1 つ目の出力が混ざりうる | `codex_hook_trust_notice_test.go:276` から。同じ fixture の出る側は `TestCodexHookTrustNotice_SpawnInASubdirectoryCountsItsNestedHooks`(`:233`) | 入れ子のファイルを置く前に同じ関数を呼んで 1 行が出ることを確かめ、置いたあとで空になることを見る。場面ごとに `bytes.Buffer` を作る |
| C2-7 | LOW | readability | パーサーの拒みの確かめが 2 本の表に分かれ、強さが違う。`SlotsRejectWhatIsNotCodexsShape`(9 行、文の先頭だけ `HasPrefix`)と `SlotsRejectNullAndMissingFieldsLikeCodex`(共有表 7 行と自前の 23 行、全文一致)は、同じ関数の同じ種類の失敗を見ていて、event の値が object の行(`:105` と `:145`)のように、同じ種類の行が両方にある。前者だけが「Go の型名が出ない」を見ていて、後者の 30 行は見ない。共有表 `codexHookTrustInvalidHooks` は `codex_hook_trust_parse_test.go:126` にあり、doctor のテスト(`TestCheckCodexHookTrust_NullHookStructuresAreNeverAPass`、`doctor_codex_hook_trust_test.go:398`)が使うが、このファイルの頭の説明は「共有する helper は `codex_hook_trust_test.go` にある」と言う。その doctor のテストは名前が「Null…」で、表には「a handler without type」「a command handler without command」のように null でない行も入る。cycle 1 の L4 の重なった文は、`doctor_codex_hook_trust_test.go:477` で 2 か所目の固定になった(1 か所目は `:129`) | `codex_hook_trust_parse_test.go:93-121`、`:123-175`。`doctor_codex_hook_trust_test.go:398`、`:129`、`:477` | 前者の表を後者に畳み、「Go の型名が出ない」の確かめを後者の `t.Run` の中に入れる。共有表の置き場を、使う 2 ファイルから見える名前にそろえる。doctor のテストの名前を `…InvalidHooksJSONIsNeverAPass` にする。L4 を直すなら期待の文は 2 か所 |
| C2-8 | LOW | maintainability | `codex_hook_trust_test.go` が 892 行で、800 行の目安(register の「Five files went past the 800-line guide」の行が使う基準)を超える。`f71a2a33` でパーサーのテスト 204 行を出したあとでも超えている。まとまりの切れ目は、root・持ち主・綴り(`codexHookCheckoutOf`、`codexHookSpellings`、`codexLinkedWorktreeMain`)のテスト群(`:93-430`、約 340 行)で、残りの設定の読みと仕分けと層のテストは 550 行ほどになる。`TestCodexHookTrust_RootOfABareRepositorysLinkedWorktree`(L9)は、同じ表の持ち主のテスト(`RootChecksTheLinkedWorktreesOwnership`)が手書きの登録で持つ場面(親に `.git` なし、親が `.git`、親の `.git` が別の repo)を実際の git で作るもので、重なりは意図(手書きの登録と git の本物の 2 通りで確かめる)に見えるが、注釈にその理由が無い | `codex_hook_trust_test.go` の全体(892 行)。重なり: `:223-293` と `:321-372` | `codex_hook_trust_checkout_test.go` に root と持ち主と綴りのテストを移す(宣言は動かすだけ)。bare の本物の git のテストの注釈に、表の行との役割の違いを 1 文書く |

### L1 から L9 の現在地

cycle 1 の指摘の番号は変えない。`9e72d49a` で `codex_hook_trust.go` が書き直され、cycle 1 の `file:line` が合わなくなったので、HEAD の場所を書く。

| ID | HEAD での状態 | HEAD の場所 |
| --- | --- | --- |
| L1 | 未解決。`Keys` と `Approved` を読むのはテストだけ。`9e72d49a` でテストの読み手が増えた(`codex_hook_trust_test.go:369`、`:647`、`:719`、`:758`、`:823` ほか) | `codex_hook_trust.go:509`(`Keys`)、`:520`(`Approved`)、`:626`、`:637-639` |
| L2 | 未解決。`paths` が両方の書式に入り、3 通りの書式が 2 か所に同じ文字列で残る | `doctor_codex_hook_trust.go:185-192`、`codex_hook_trust_notice.go:63-70` |
| L3 | 未解決。定義と比較だけが定数を使い、呼び出しは文字列 | `codex_hook_trust_notice.go:90`、`:98`、`org_director_headless.go:366` |
| L4 | 未解決。固定する場所が 1 か所から 2 か所に増えた | `doctor_codex_hook_trust.go:134`、`codex_hook_trust_parse.go:276`、`doctor_codex_hook_trust_test.go:129`、`:477` |
| L5 | 注釈は未解決(C2-4 の (c) で広がった)。register の行は足された(`23a57869`、「Where the Codex hook trust check … reads differently from codex」の (a)) | `codex_hook_trust_parse.go:44-62`、`:204-206` |
| L6 | 未解決(変更なし) | `codex_hook_trust_notice_test.go:173-226`(`const` は `:174`) |
| L7 | 未解決。行が `9e72d49a` で +65 ずれた | `codex_hook_trust_notice_test.go:388`(関数)、`:403`(`strings.Contains(stderr, "codex")`) |
| L8 | 未解決(変更なし) | `doctor_codex_hook_trust_test.go:194` |
| L9 | 未解決。場面が 2 つから 3 つに増えた(`outer/cache.git`)が `t.Run` は無く、名前、手書きの空 commit(`:326-327`)もそのまま | `codex_hook_trust_test.go:321-372` |

### Positive notes (cycle 2)

- 持ち主の確かめが、codex の `resolve_root` の順に 1 関数 1 つの確かめで分かれている(`codexReadGitdirFile`、`codexReadGitMetadata`、`codexWorktreeRegisteredAt`、`codexMainOwnsCommonDir`)。どれも通常のファイルの `Lstat` を通してから読み、失敗は bool で返して理由を外に出さない。
- 失敗の作り方が、利用者の権限に依らない。入れ子の層の「読めない」は mode 000 でなくディレクトリを置いて作っている(`NestedLayerProblems` の `<dir>`)。
- `parseCodexHookHandler` が必須の欄の不足を `slices.Sorted(maps.Keys(fields))` の順に見て、エラーの文を決まった順にしている。重複の検出が、JSON の鍵の別表記(`o`)を `dec.Token()` の復号後の文字で比べるので、表記違いで素通りしない。
- 入れ子の層のテストが、worktree 側と main 側の綴りを分けて、承認が worktree 側のパスにだけあるとき warn になることまで固定している(`LinkedWorktreeChecksNestedLayersUnderTheMainCheckout` の最後の行)。
- パーサーの移動は、宣言の単位で 107 件が一致する純粋な移動だった。

### Coverage gaps (cycle 2)

- テストは走らせていない(`/test` の範囲)。mode 000 を使う cycle 1 のテストは、root で動く環境では Skip になるので、`/test` の結果で skip と pass を見分けたい。
- `templates/base/.codex/README.md` と `templates/base/docs/recipes/codex-setup.md` は `check-sync` に任せ、読んでいない。`.codex/README.md` は `9e72d49a` の 6 行の diff だけ読んだ。
- `d9103dfc`(origin/main の取り込み)の衝突の解き方は、この branch が足した README と tech-debt の差分(origin/main との差)を読んだだけで、merge commit 自体の diff は読んでいない。README の表の 2 行と tech-debt の 6 行が origin/main と違う形で残るが、どれも対になる `-` と `+` で、origin/main の行が消えた形は無かった。
- `codexReadGitMetadata` の通常のファイルでない・64 KiB 超・読めない、の行は、register の「Test gaps」の (b) に挙がっていて、テストは見当たらなかった(`codex_hook_trust_test.go` で 64 KiB を作るのは設定の 1 MiB の確かめだけ)。`9e72d49a` で同じ関数を通る読みが増えた(登録の `gitdir`・`commondir`)が、その分は (b) に入っていない。
- codex 本体の `resolve_root` は、今回も読み直していない(cycle 1 では `discovery.rs` を、今回は `hook_config.rs` と `protocol/src/hooks.rs` を読んだ)。持ち主の確かめが codex と同じ形であることは、`.codex` の注釈と evidence の記述に任せている。

### Tech debt identified (cycle 2)

新しい行は、この run では `docs/tech-debt/README.md` に足していない(lead の頼みが report だけの 1 コミット)。足す候補は C2-1 に書いた。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| cycle 1 の L1 から L4 と L6 から L9(使われない `Keys` と `Approved`、2 つの書式の重複、使われない定数、重なった error 文、テストの整理 4 件)が、上限の最後の回で残る | 直しは小さく、判定や出力の意味を変えない。放っておくと、`Keys` を読むテストが増え、文言を直すときに 2 か所が別々に古くなる | 上限の回で、verify と test の前に直さなかった場合の繰延 | doctor か launch の文言を次に直すとき、または `codexHookTrustItem` を使う新しい読み手が出るとき | `docs/reports/self-review-2026-11-12-codex-hook-trust-check.md`(L1 から L4、L6 から L9) |
| register の 2 行の食い違い(Test gaps の (b) の type error、「reads differently from codex」に配列の形が無い) | 次に読む人が、消えた分岐を探す・差を知らずに warn を見る | `/sync-docs` の範囲 | `/sync-docs` | 同 report の C2-1 |

### Recommendation (Cycle 2)

- Merge: OK(CRITICAL・HIGH・MEDIUM なし、cycle 2 の新しい指摘は LOW 8)。`9e72d49a` の 3 つの直し(入れ子の層、null の形、持ち主の確かめ)は、diff の質の面で、判定を弱める道や利用者の設定の中身が出る道を新しく開くものは見つからなかった(仕様との適合は `/verify` の範囲)。`f71a2a33` は宣言の単位で純粋な移動だった。cycle 1 の L1 から L9 は、9 件とも HEAD で未解決(L5 だけ register に行がある)。
- Follow-ups: 今回の self-review は cycle 2 の verify と test の前なので、ここで直す分は追加の run を要しない(上限 2 回のうちの 2 回目)。直すなら 1 コミットで次を勧める。L4(文を `syntax error at byte %d` に、期待 2 か所と parse の 2 行)、L3(定数を使う)、L1(`Keys` と `Approved` を消す、またはテスト以外の読み手を作る)、L2(件数の句の関数)、C2-3、C2-4 の (a) と (b) と (c)、C2-5、C2-2。残る L6 から L9 と C2-6 から C2-8 はテストの整理で、直さないなら PR 本文の Known gaps に 1 行ずつ書き、register の 2 行は `/sync-docs` で直す。
- 記録の食い違い: `docs/insights/events/2026-11-12-codex-hook-trust-check.jsonl` の self_review は cycle 1 の行が `low: 5` のままで、report の cycle 1 は LOW 9 件になっている。cycle 2 の行は、この cycle の新しい指摘 8 件を `low: 8` で足した(cycle 1 の未解決 9 件は数えない)。
