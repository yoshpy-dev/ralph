# Walkthrough: codex の hook が未承認なら doctor と codex の座席の起動が知らせる(codex-hook-trust-check)

- 計画: `docs/plans/archive/2026-11-12-codex-hook-trust-check.md`(PR の作成時に archive へ移す)
- issue: #310。PR #324
- 差分: `git diff 14531858...HEAD`(29 ファイル、約 +5,400 / −87。Go の本体は約 +1,100、Go のテストは約 +2,000、残りは計画・報告・evidence・文書・tech-debt)

## 1. codex の規則を写した部品(読む価値が一番高い)

`internal/cli/codex_hook_trust.go`。codex 0.166.0 が hook をどの hooks.json から読み、承認をどの鍵で引くかを、git を呼ばずに写す。

- `codexHookCheckoutOf`: dir から上にたどり、`.git` のファイル、または HEAD を持つ `.git` ディレクトリを持つ checkout を見つける。linked worktree なら `.git` のファイルの gitdir から main を求め、codex の `resolve_root` と同じ持ち主の確かめ(gitdir が `worktrees` の下、登録の `gitdir` と `commondir` が戻り先を指す、main の `.git` が実パスで common dir と同じ)をすべて通ったときだけ main を採る。落ちれば worktree 自身。
- `codexHookSpellings`: dir を字面のまま絶対パスにしたものと、symlink を解いた実パスの 2 つの綴り。座席の cwd がどちらかは分からないので、両方で承認を求める。
- 層: checkout の root から dir までの各ディレクトリのうち `.codex` ディレクトリを持つものが層で、その hook は `<repo root>/<相対 dir>/.codex/hooks.json` から読む(linked worktree では main 側)。
- `readCodexHookStates`: `[hooks.state."<key>"]` の `trusted_hash` と `enabled` だけを、Check 11b と同じ大きさの上限とファイルの種類の確かめで読む。decode の失敗は位置だけを出し、設定の中身は出さない。
- `codexHookTrustFor`: 層ごとの hooks.json の状態と slot、設定の状態、未承認と止めてある hook の一覧を返す。

`internal/cli/codex_hook_trust_parse.go`: hooks.json を codex の serde の型(`HooksFile`、`HookEventsToml`、`MatcherGroup`、`HookHandlerConfig`)に合わせて読み、group と handler の番号を並びのまま数える。空の command の handler は数えない。`{"hooks": null}` などは形の違うファイル。

## 2. doctor の確かめ

`internal/cli/doctor_codex_hook_trust.go` の `checkCodexHookTrust`。`doctor.go` の Check 3 の直後から呼ぶ。

- codex が `[org].driver_pool` に無い: pass「not needed」(設定を読まない)。
- codex が読む hooks.json が無い・読めない・形が違う: Check 3 と同じファイルなら info、linked worktree から見た main のものなら warn。ほかの層の承認を確かめていないことも書く。
- 設定ファイルが無い・読めない・decode できない: warn。
- 未承認・止めてある hook がある: warn と、開く directory と承認の手順(綴りが 2 つなら両方)。
- すべて承認済み: pass(鍵だけ、hash は確かめていない、変わった hook は座席の pane に「Hooks need review」が出る)。

## 3. 起動の 1 行

`internal/cli/codex_hook_trust_notice.go` の `printCodexHookTrustNotice`。spawn と start の 2 つの形(`org.go`)、director start(`org_director_headless.go`)が、driver を解決した直後に呼ぶ。codex のときだけ、未承認・止めてある hook があれば stderr に 1 行。設定ファイルが読めないとき、hooks.json が無いときは出さない(doctor が知らせる)。3 つの動詞の `--help` に段落を足した。

## 4. 文書と記録

- `.codex/README.md` の 2 写しの Trust UX: 鍵の形、`trusted_hash` が宣言全体の hash であること、worktree と入れ子の層の扱い、doctor の確かめ。
- `docs/recipes/codex-setup.md` の 2 写し、`.codex/config.toml` の 2 写しの注釈、README の doctor と org の行。
- `docs/evidence/codex-hook-trust-worktree-2026-11-12.md`: 一時の `CODEX_HOME` と一時 repo で、codex の app-server の `hooks/list` が返す鍵と状態を確かめ、作った binary の doctor の判定が一致した記録。
- `docs/tech-debt/README.md`: この PR が条件を満たした既存の行への持ち越しの注、codex との差、テストの穴、テストの構造の行。

## 見どころ

- 知らせが消える向きの差は、codex の `project_root_markers` を既定から変えた設定と、spawn の `--cwd` に別の checkout の中への symlink を渡したときだけ(tech-debt に向きを書いた)。
- cross-review cycle 1 の 3 件(入れ子の層、null の形、持ち主)は、どれも新しい確かめが誤って pass を返す形だったので直した。
- テストは `-run 'CodexHookTrust'` で 199 本。変異の確かめは slice ごとに行い、生き残った変異を捕まえるテストを足した(G1〜G4)。
