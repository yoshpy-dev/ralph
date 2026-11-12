# codex-hook-trust-check

- Status: Approved
- Approved: 2026-11-12 sha256:147d7ec0e7ae
- Owner: Claude Code
- Date: 2026-11-12
- Related request: 下流の運用 issue #307〜#313 の系列、レーン B の 2 本目。codex の hook が未承認でも `ralph doctor` が pass にし、codex の座席で本番を守る hook が黙って効かない
- Related issue: https://github.com/yoshpy-dev/ralph/issues/310
- Type: feat
- Branch: feat/codex-hook-trust-check

## Objective

codex の座席がありうる repo で、`.codex/hooks.json` の hook が codex に承認されているかを `ralph doctor` が確かめ、未承認なら warn にする。`ralph org spawn`、`ralph org start`(2 つの形)、`ralph org director start` は、解決した driver が codex のとき同じ確かめをし、未承認なら stderr に 1 行出す。

STOIC では、本番の資源を変えうるコマンドを `.codex/hooks.json` の PreToolUse で止めていたが、repo を移したあと承認の記録が古いパスにしか無く、codex の座席ではその検査が効かない状態だった。そのときの `ralph doctor` は `Codex effective config: pass — hooks.json wired; confirm codex trust . ran for this project` だった。

## Scope

- 新しいファイル `internal/cli/codex_hook_trust.go`: hooks.json の handler ごとの鍵を数え、codex のユーザー設定(`$CODEX_HOME/config.toml`、無ければ `~/.codex/config.toml`)の `[hooks.state]` と突き合わせる部品
- 鍵は `<main worktree の root>/.codex/hooks.json:<event の snake_case>:<group の番号>:<handler の番号>`。hooks.json は main worktree の root のものを読む(linked worktree の codex は main の hooks.json を読み、鍵も main のパスで引く。`docs/evidence/codex-hook-trust-worktree-2026-11-12.md`)
- 鍵があり `trusted_hash` を持てば承認済み。鍵が無いか `trusted_hash` が無ければ未承認、`enabled = false` なら止めてある。hash の中身は確かめない(codex 独自の TOML 正規化 `version_for_toml` で作るため、Go で作り直さない)
- doctor の新しい Check「Codex hook trust」(Check 3 の「Codex effective config」の直後)。`codexSeatPossible` のときだけ確かめ、そうでなければ「not needed」の pass。未承認・止めてあるものがあれば warn、承認の手順を示す。pass の文には「hash は確かめていない」と書き、承認のあとで変わった hook は座席の pane に codex の「Hooks need review」が出ることを併記する
- 「Codex effective config」の pass の文 `confirm codex trust . ran for this project` を、新しい Check を指す文に直す
- spawn、start(task の形と `--plan` の形)、director start: 解決した driver が codex のとき、座席の cwd(`--plan` は main worktree、director start は director が動く directory)の repo で同じ確かめをし、未承認・止めてあるものがあれば stderr に 1 行(`printLegacyModelPoolNotice` の近く、#313 の知らせの次)。動きと終了コードは変えない
- 文書: README(doctor の行と「doctor は hook の承認を確かめられない」の文)、`.codex/README.md` と `templates/base/.codex/README.md` の Trust UX、`docs/recipes/codex-setup.md`
- hash の食い違いを、座席の pane に出る codex の「Hooks need review」(`internal/org/resume_screens.go` がすでに選択の画面として見分ける)で知らせる案を、doctor の文と文書と tech-debt に併記する(実装は後続)

## Non-goals

- 未承認のとき起動を断る設定(ユーザーの決定で入れない)
- hash を Go で計算して食い違いを確かめること
- codex の app-server(`hooks/list`)を doctor から呼ぶこと
- `.codex/config.toml` の `[hooks]`(TOML で書いた hook)の鍵を確かめること。ralph の雛形は hooks.json だけを使う
- 「Hooks need review」の画面を見張りが見つけて ALERT にすること(案として記録する)
- 古い codex(linked worktree で worktree のパスの鍵を引く版がもしあれば)への対応

## Assumptions

- codex-cli 0.166.0 の実機と、tag `rust-v0.166.0` のソースで確かめた鍵の形と worktree の扱いが、ralph が対象にする codex に当てはまる(`docs/evidence/codex-hook-trust-worktree-2026-11-12.md`)
- `[hooks.state."<key>"]` は `trusted_hash = "sha256:<64 hex>"` と、任意の `enabled` を持つ。ほかの鍵は無視する
- 承認の記録はユーザー設定(`$CODEX_HOME/config.toml`)の `[hooks.state]` だけを読む。codex は設定の層すべてから `[hooks.state]` を集めるが、project の `.codex/config.toml` に state が書かれた例は観測していない
- group と handler の番号は hooks.json の並びのまま。codex は空の command の handler を読み飛ばすが番号は使う。読み飛ばされる handler は承認が要らないので数えない
- event の名前は codex の 14 種(`PreToolUse` → `pre_tool_use` など)。知らない名前の event は数えない
- `internal/cli` のテストは 1 回で 5〜14 分かかる。slice の確かめは `-run` で絞る

## Affected areas

- `internal/cli/codex_hook_trust.go`(新規)と `internal/cli/codex_hook_trust_test.go`(新規)
- `internal/cli/doctor.go`(Check の追加、Check 3 の pass の文)、doctor のテスト
- `internal/cli/doctor_codex_writable_root.go`(codex の設定ファイルを読む部分を共有する場合だけ)
- `internal/cli/org.go`(spawn と start の 2 つの形)、`internal/cli/org_director_headless.go`(director start)、CLI のテスト
- `README.md`、`.codex/README.md`、`templates/base/.codex/README.md`、`docs/recipes/codex-setup.md`
- `docs/evidence/codex-hook-trust-worktree-2026-11-12.md`(計画の前の実機の確かめ)

## Visual review

- Page: `.harness/state/plan-visual/codex-hook-trust-check.html`(やさしい図 2 枚、詳しい図 2 枚)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: full、overview、easy-states、detail-overview、detail-decide の 5 枚を撮って読んだ。detail-decide で「すべて済み」の矢印が warn の箱にかかっていたので、箱の外を回して撮り直した

## Design decisions

Critical forks: None。ユーザーが決めた点(doctor の Check、鍵の有無だけ、`codexSeatPossible` のときだけ、spawn・start・director start で 1 行、断る設定は入れない、hash は「Review hooks」で知らせる案を併記)の残りを、次のとおり自律で選んだ。

- D1(自律で選んだ、実機で確かめた、計画助言の指摘 1 で直した): 読む hooks.json と鍵のパスは、codex が cwd から求める root のもの。codex(tag `rust-v0.166.0` の `codex-rs/git-utils/src/trust.rs` の `resolve_root`)と同じく、cwd を字面のまま上にたどって `.git` を探し、`.git` がディレクトリならその場所、ファイル(linked worktree)なら中の `gitdir:` が `<main>/.git/worktrees/<name>` を指すときその `<main>` を root にする(gitdir が `worktrees` の下でない、`<main>/.git` が無いなど codex が確かめに落ちる形では、worktree 自身を root にする)。git のコマンドは呼ばない。実機(codex 0.166.0)で、linked worktree の codex は main の hooks.json を読み、鍵も main のパスで引いた。main のチェックアウトを symlink のパスで開くと鍵もそのパスになった。codex の座席の cwd が字面か実パスかは確かめられないので、cwd を字面のまま使った root と、cwd の実パス(`filepath.EvalSymlinks`)から求めた root の両方を候補にし、綴りが違うときは両方の鍵に承認があるときだけ承認済みとする(知らせる側に倒す)
- D2(自律で選んだ): 状態は 3 つに分ける。承認済み(鍵があり `trusted_hash` が空でない、`enabled` が false でない)、未承認(鍵が無いか `trusted_hash` が無い)、止めてある(`enabled = false`)。止めてある hook も座席では動かないので warn に数える
- D3(自律で選んだ): doctor の結果。`codexSeatPossible` でなければ pass「not needed」。codex が読む hooks.json が無い・読めない・JSON でないとき、それが Check 3 の読むファイル(`<targetDir>/.codex/hooks.json`)と同じファイルなら Check 3 が知らせるので info で「確かめていない」とし、違うファイル(linked worktree から流したときの main の hooks.json)なら warn にして、codex の座席では hook が 1 つも動かないことを言う(計画助言の指摘 2: Check 3 は worktree の hooks.json を読むので、main の hooks.json が無いことを誰も知らせなくなる)。codex の設定ファイルが無ければ warn(1 つも承認されていない)。読めない・decode できなければ warn(理由だけ、中身は出さない。Check 11b の `readCodexUserConfig` と同じ扱い)。未承認・止めてあるものがあれば warn で、数と、先頭のいくつかの鍵の末尾(`pre_tool_use:0:0` など)、承認の手順(「`<root>` で codex を対話で開き、Hooks need review で Trust all and continue を選ぶ」)を出す。すべて承認済みなら pass で、「hash は確かめていない。承認のあとで変わった hook は、座席の pane に codex の Hooks need review が出る」と書く
- D4(自律で選んだ): codex の設定ファイルは Check 11b と同じく `$CODEX_HOME/config.toml`、無ければ `~/.codex/config.toml` を、同じ大きさの上限とファイルの種類の確かめで読む。decode は `[hooks.state]` だけの別の構造体で行い、Check 11b の decode と互いに影響しないようにする
- D5(自律で選んだ): spawn・start・director start の 1 行は、未承認・止めてあるものがあるときだけ出す。設定ファイルが無いときも「1 つも承認されていない」として出す。設定ファイルが読めないとき、hooks.json が無いときは出さない(doctor が知らせる)。出す位置は `printLegacyModelPoolNotice` と #313 の知らせのあと、driver を解決した直後。`--dry-run` の spawn でも出す。codex の設定ファイルの場所は doctor と同じ変数 `doctorCodexSandboxEnv` から読み、新しい変数は作らない(テストの TestMain がこの変数を存在しないパスに固定しているので、テストがユーザーの `~/.codex` を読まない)
- D6(自律で選んだ): 1 行の文は `note: <verb>: codex seat <id>: <n> of <m> hooks in <hooks.json> are not approved in <config>, so they do not run in the seat; open codex in <root> and trust them (ralph doctor lists them)` の形。stdout には出さない

## Acceptance criteria

- [ ] AC1: `[org].driver_pool` に codex があり、hooks.json の handler の鍵が codex の設定に無い repo で、`ralph doctor` の「Codex hook trust」が warn になり、未承認の数と承認の手順を示す(issue の受け入れ条件 1)
- [ ] AC2: すべての handler の鍵が `trusted_hash` つきである repo では、同じ Check が pass になり、文に「hash は確かめていない」ことと「Hooks need review」が出ることを書く(issue の受け入れ条件 2)
- [ ] AC3: `enabled = false` の handler、`trusted_hash` の無い鍵、codex の設定ファイルが無いときは warn。読めない・decode できない設定ファイルは warn で、中身を出さない
- [ ] AC4: codex が driver_pool に無いときは pass「not needed」で、設定ファイルを読まない。codex が読む hooks.json が無い・読めないとき、Check 3 と同じファイルなら info、違うファイル(linked worktree から流して main の hooks.json が無い・壊れている)なら warn
- [ ] AC5: linked worktree から doctor を流しても、main worktree の root の hooks.json と、そのパスの鍵で確かめる。root は codex と同じ規則(字面のまま上にたどる、`.git` のファイルの gitdir から main を求める)で、git のコマンドを使わない
- [ ] AC9: cwd の字面と実パスで root の綴りが違うとき(symlink 越しの main のチェックアウト)は、両方の綴りの鍵に承認があるときだけ承認済みにする。片方の綴りにだけ承認があれば、どちら向きでも warn(と起動の 1 行)になる
- [ ] AC6: `ralph org spawn`(codex に解決したとき)、`ralph org start` の 2 つの形、`ralph org director start` で、未承認・止めてあるものがあれば stderr に 1 行が 1 回出る。claude に解決したとき、すべて承認済みのとき、hooks.json が無いときは出ない。動きと終了コードは変わらない(issue の受け入れ条件 3)
- [ ] AC7: 鍵の event 名は codex の 14 種の snake_case、番号は hooks.json の並び。空の command の handler は数えない
- [ ] AC8: 「Codex effective config」の pass の文が新しい Check を指し、README、`.codex/README.md`(2 つの写し)、`docs/recipes/codex-setup.md` が、doctor の確かめと worktree の扱いと「Hooks need review」の案を説明している

## Implementation outline

1. 承認の確かめの部品: `internal/cli/codex_hook_trust.go` に、hooks.json の handler の鍵を数える関数、`[hooks.state]` を読む関数、両者を突き合わせて結果(承認済み・未承認・止めてあるの鍵の一覧)を返す関数、repo の root を決める関数を作り、単体テストを書く(AC3 の判定、AC5 の root、AC7) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'CodexHookTrust'`
2. doctor の Check: `checkCodexHookTrust` を Check 3 の直後に足し、Check 3 の pass の文を直す。doctor のテスト(AC1〜AC5)。init した repo で doctor を流す既存テストすべてに効くので、確かめは package の全体 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/`
3. spawn・start・director start の 1 行と、その CLI テスト(AC6) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'CodexHookTrust|TestOrgSpawn_RoleModels|TestOrgStart_RoleModels|TestOrgStartPlan_RoleModels|TestOrgDirectorStart_RoleModels'`
4. 文書と実機の確かめ: README、`.codex/README.md` と `templates/base/.codex/README.md`、`docs/recipes/codex-setup.md`。作った binary で、一時的な `CODEX_HOME` と一時的な repo に対して doctor が warn → 承認(app-server の `currentHash` を写す)→ pass になることを evidence に足す(AC8、AC1・AC2 の実機) — verify: `./scripts/check-sync.sh && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC8 をテスト名と doctor の出力の文で突き合わせる。issue #310 の受け入れ条件 3 つが AC1、AC2、AC6 に入っていること
- Documentation drift to check: README の doctor の行と codex の節、`.codex/README.md` の 2 つの写しが同じこと(`scripts/check-sync.sh`)、`docs/recipes/codex-setup.md`、`docs/recipes/codex-seat-permissions.md` に関係する記述が無いか
- Evidence to capture: `run-static-verify.sh` の結果、実機の doctor の warn と pass の出力(`docs/evidence/codex-hook-trust-worktree-2026-11-12.md` に足す)

## Test plan

- Unit tests: 鍵の数え方(複数の event・group・handler、空の command、知らない event、hooks.json が JSON でない)、`[hooks.state]` の読み方(鍵が無い、`trusted_hash` が無い、`enabled = false`、余計な鍵、`[hooks]` が無い、大きすぎるファイル、ディレクトリ)、root の決め方(main、linked worktree、git の外)
- Integration tests: doctor を一時的な repo と一時的な `CODEX_HOME`(`doctorCodexSandboxEnv` を差し替える既存のやり方)で流し、warn と pass と not needed と info を確かめる。linked worktree から流したとき main のパスの鍵で確かめること。spawn(`--dry-run` と stub の herdr)、start の 2 つの形、director start を codex と claude で流し、stderr の 1 行の有無を確かめる
- Regression tests: 既存の doctor のテストで「Codex effective config」の pass の文を固定しているものを直す。spawn・start・director start の既存のテストは stderr を完全一致で見ているので、テストの `CODEX_HOME` を一時ディレクトリにして、承認の確かめが既存のテストの stderr を変えないこと(codex の driver の既存のテストで、hooks.json が無い repo なら出ない)を確かめる
- Edge cases: hooks.json の handler が 0 個、`enabled = false` と `trusted_hash` が両方ある、`CODEX_HOME` が空、main worktree を持たない git(bare の linked worktree)、symlink 越しの main のチェックアウトで承認が実パスにだけある・symlink のパスにだけある(AC9、両方向)、linked worktree の hooks.json は正しく main の hooks.json が無い・壊れている(AC4 の warn)、`.git` のファイルの gitdir が `worktrees` の下でない(worktree 自身を root にする)
- Evidence to capture: `go test` の結果と所要時間、実機の doctor の出力

## Risks and mitigations

- 既存の CLI テストが codex に解決する spawn・start・director start を、`.codex/hooks.json` のある repo を座席の cwd にして流すと、1 行が出て stderr の完全一致が落ちる。TestMain(`internal/cli/main_test.go`)は `doctorCodexSandboxEnv` を存在しないパスに固定しているので、設定ファイルが無い扱いになり、D5 のとおり 1 行が出る(`CODEX_HOME` を一時ディレクトリにしても抑えられない) → 抑えられるのは座席の cwd に hooks.json が無いときだけ。既存の codex のテストの多くは `t.Chdir(t.TempDir())` と一時の `--cwd` を使い、`--cwd "."` のテスト(`org_feature_test.go`、`org_watch_gate_test.go`)は引数の誤りか claude なので今は出ない。/test で package 全体を流して確かめ、落ちたテストは cwd を一時ディレクトリにするか期待に 1 行を足す
- doctor の既存テストも同じ固定を受ける。`ralph init` の雛形は `.codex/hooks.json` を持ち、既定の driver_pool は codex を含むので、init した repo で doctor を流す既存テストでは新しい Check が warn になる。warn の数や出力の全体を固定するテストがあれば落ちる → slice 2 の確かめは `internal/cli` の全体を流す
- ユーザーの `~/.codex/config.toml` をテストが読んでしまう → doctor も spawn 側も `doctorCodexSandboxEnv` から読む。TestMain の固定が両方に効く
- 鍵の形が codex の版で変わる → 0.166.0 で実機とソースを確かめた。違う版で鍵が見つからないと warn になり、知らせる側に倒れる
- main のチェックアウトを symlink のパスで使うと codex の鍵が変わる → 字面と実パスの両方の綴りで承認を求めるので、片方にしか承認が無いと warn になる(知らせる側。計画助言の指摘 1)。両方に承認があるのに codex が別の綴り(第三のパス)を使う形は残るが、ralph が座席に渡す cwd は字面か実パスのどちらかなので起きない見込み。文書に書く
- codex の root の求め方を写すので、codex がその規則を変えると鍵のパスがずれる → ずれれば鍵が見つからず warn になる(知らせる側)。evidence に codex の版を残す
- ほかのレーンが `internal/cli/org.go`、`internal/cli/doctor.go`、README、tech-debt を変える → PR を作る前と merge の前に origin/main を取り込む

## Rollout or rollback notes

- doctor の Check が 1 つ増え、spawn・start・director start に stderr の 1 行が増えるだけで、断る動きは無い。戻すときは PR を revert する
- 次の release ノートに「doctor が codex の hook の承認(鍵の有無)を確かめる」「codex の座席の spawn で未承認なら 1 行」を入れる

## Open questions

- なし(ユーザーの決定と実機の確かめで埋まっている)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。2 件。(1) [HIGH] main のチェックアウトを symlink のパスで開くと codex の鍵もそのパスになり、実パスで探すと承認が実パスにだけあるとき誤って pass になる → D1 を「codex と同じ規則で root を求め、字面と実パスの両方の綴りで承認を求める」に直し、AC9 と両方向のテストを足した。(2) [MEDIUM] Check 3 は worktree の hooks.json を読み、新しい Check は main のものを読むので、main に hooks.json が無いと誰も知らせない → D3 と AC4 を「Check 3 と別のファイルなら warn」に直した。consult(consult-plan-hooktrust)の指摘(TestMain が `doctorCodexSandboxEnv` を固定している、init の雛形が hooks.json を持つ、ユーザー設定だけを読む)は Risks、D5、Assumptions、slice 2 の確かめに入れた
- [x] Plan approved — 自動承認(ユーザーの指示「計画は自動承認」、2026-11-12。ralph-d1 経由の中継「推奨案を自動承認」)
- [x] Branch created
- [x] Implementation started
  - S1 148f9b19(implementer): 部品(root、鍵、[hooks.state] の読み、分け方)。`readCodexFileBounded` を doctor_codex_writable_root.go から切り出した。範囲外の所見: codex は prompt・agent 型の handler、SessionEnd の MCP tool、server か tool が空の MCP tool、matcher がコンパイルできない group の handler には鍵を作らないが、ralph は数える(ralph の雛形は command 型だけなので今は影響なし。知らせる側に倒れる差として tech-debt と Known gaps)
  - S2 51256efd(implementer): doctor の Check、Check 3 の pass の文。`internal/cli` 全体 722.7 秒で通過。テストの雛形(`setupTestEmbedFS`)は hooks.json を持たないので、既存の doctor テストでは新しい Check は info になり、直したテストは無い
  - S3 56c558dd(implementer): spawn・start の 2 つの形・director start の 1 行と、slice 2 の文言の直し 2 つ(info の「Check 3 が知らせる」、cli_test.go の古いコメント)。`internal/cli` 全体 577.4 秒で通過。D6 の文の形からのずれ: 後半は「run 'ralph doctor' for the keys and how to trust them」で、件数で trust・re-enable を出し分ける
  - S4 d135aacb(implementer): README、`.codex/README.md` の 2 写し、`docs/recipes/codex-setup.md` と雛形の写し `templates/base/docs/recipes/codex-setup.md`(check-sync が一致を求めるので足した)、3 つの動詞の `--help`、director の 1 行を「codex director」にした
  - S5 0d813954(implementer): 実機で見つけた不具合の直し。相対の dir(doctor の `.`)だと実パスの候補が落ち、承認が symlink のパスにだけあると誤って pass になった。絶対パスにしてから symlink を解くようにし、テストを足した(直す前は落ちることを確かめた)
  - 実機の確かめ: 0d813954 の binary で doctor の warn → pass(main と linked worktree)、symlink 越しの warn、`enabled = false` の warn を確かめ、app-server の `hooks/list` と食い違わないことを evidence に足した
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
