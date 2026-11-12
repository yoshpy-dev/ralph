# Self-review report: codex-hook-trust-check

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(issue #310)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `f36cfc52`、origin/main `d2e5c6e9` を取り込んだ merge の前の変更だけ、18 ファイル、+2732 / -77)。Go の本体 3 本(`codex_hook_trust.go`、`doctor_codex_hook_trust.go`、`codex_hook_trust_notice.go`)、`doctor.go`・`doctor_codex_writable_root.go`・`org.go`・`org_director_headless.go` の diff と呼び出しの周辺、テスト 3 本(`codex_hook_trust_test.go`、`doctor_codex_hook_trust_test.go`、`codex_hook_trust_notice_test.go`)を全部読んだ。README と `.codex/README.md` と recipe は diff を読んだ(`templates/base` の写しは `check-sync` の範囲として読んでいない)。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: `L1` から `L5`(LOW)

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

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| prompt 型・agent 型の handler、SessionEnd の MCP tool、server か tool が空の MCP tool、matcher がコンパイルできない group の handler を、codex は承認の対象にしないが、`parseCodexHookSlots` は数える(L5) | そういう handler を持つ hooks.json で、承認しても消えない warn と起動時の 1 行が出る | ralph の雛形は command 型だけで、知らせる側に倒れる差なので | 雛形が command 型以外の handler を使うとき、または下流の利用者がこの warn を消せないと報告したとき | `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(S1 の記録、Scope) |
| hash の食い違い(承認のあとで変わった hook)を、座席の pane に出る codex の「Hooks need review」(`internal/org/resume_screens.go` がすでに選択の画面として見分ける)から見張りが見つけて知らせる案。今は doctor の pass の文と `.codex/README.md` に書くだけ | 承認のあとに hook が変わると、doctor は pass のまま、座席は 1 つも動かない | plan の Non-goals(ユーザーの決定で後続) | 下流で、承認の食い違いが起きて hook が黙って効かない報告があるとき | `docs/plans/active/2026-11-12-codex-hook-trust-check.md`(Scope の最後の項、Non-goals) |

_この 2 行は plan の Scope が tech-debt に書くと決めているもの。lead の頼みはレポートだけを 1 コミットにすることなので、`docs/tech-debt/README.md` にはまだ足していない。`/sync-docs` で足す。_

## Recommendation

- Merge: OK(CRITICAL・HIGH・MEDIUM なし、LOW 5)。利用者の codex の設定の中身が doctor の文にも起動時の 1 行にも出る道は見つからなかった。
- Follow-ups: L1 から L5 は、このブランチに足す次のコミットで直せる。どれも判定や出力の意味を変えないので、パイプラインを回し直す理由にならない。直すなら cross-review の前に 1 コミットにまとめる(L4 は期待する文が変わるのでテスト 1 行も直す)。L5 の register の行は `/sync-docs` で足す。
