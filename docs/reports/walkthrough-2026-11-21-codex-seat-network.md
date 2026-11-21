# Walkthrough: codex-seat-network

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-21-codex-seat-network.md
- Issue: #362
- Base: main 9cd63d60

差分は 33 ファイル、1729 行。そのうち約 800 行は報告、計画、証拠で、コードとテストは 14 ファイル(約 760 行)、残りは文書と skill の写し。下の順に読むと早い。

## 1. 何が起きていたか

codex の座席は、autonomous と edits の権限で `--sandbox workspace-write` を付けて立つ。codex の既定では、この sandbox は外への通信を止める。そのため座席の中で通信の要る検査(tflint のプラグインの初期化、依存の取得)が落ち、gate も落ちていた。下流では、人が `~/.codex/config.toml` に `network_access = true` を書いて回避していた。

## 2. 直し方

- `codexNetworkAccessArgs`(`internal/org/permissions.go:97`)が `-c sandbox_workspace_write.network_access=<bool>` を返し、`permissionArgsForDriver` が autonomous と edits の sandbox の引数のすぐ後ろに足す(`permissions.go:181`、`:186`)。`slices.Concat` で新しい slice を返すので、パッケージの `codexAutonomousArgs` と `codexEditsArgs` は書き換わらない
- 値は新しい鍵 `[org.permissions].codex_network_access` で決まる(`internal/config/config.go:316`、既定 true、`Default()` は `:852`)。ユーザーの決定(2026-11-21)は「codex の座席は既定で全部許す」。鍵がない `ralph.toml` も true と読む
- false のときも `=false` を渡す。渡さないと人の設定ファイルの値が効き、鍵が false でも通信が開いたままになりうる
- guarded の codex の座席には何も足さない(sandbox の引数を渡さず、人の設定に従う)。claude の座席も変わらない
- `ralph doctor` に「Codex sandbox (seat network)」の項目を足した(`internal/cli/doctor_codex_network.go:59`、登録は `internal/cli/doctor.go:234`)。workspace-write の codex の座席を立てうるかは、agmsg の writable root の検査と同じ `codexSeatPossible` と `codexSeatModesPossible` で決め、`codex_verified` が false なら数えない。鍵が true なら info、false なら warn を出す。どちらの文も、鍵の変更はこれから立てる座席にだけ効くと言う
- `doctor_shell_alias.go` のコメントは、alias の `-c` と ralph の `-c` がぶつからないこと(codex は `-c` を何度でも受け、後のものが勝つ)を codex のソースで確かめた結果を書く

## 3. 先に読むテスト

- `TestPermissionArgsForDriver_CodexNetworkAccess`(`internal/org/permissions_test.go:186`): 2 つの driver、3 つの権限、鍵の true と false の組み合わせ。guarded と claude に何も足さないことも見る
- `TestPermissionArgsForDriver_CodexReturnsANewSlice`(`permissions_test.go:240`): 返した slice を書き換えても、次の呼び出しの値が変わらないこと
- `TestOrgSpawn_Codex_NetworkAccessInStartArgs`(`internal/org/spawn_test.go:4220`): spawn が herdr に渡す起動の引数で、`-c` が sandbox の引数の直後、`--model` の前に入ること。鍵の true と false、guarded に何も足さないことも見る。`--add-dir` の並びを見る既存のテストは、sandbox の引数、`-c`、`--add-dir`、`--model` の順に直して通る
- `TestLoad_OrgPermissionsCodexNetworkAccess` ほか 2 つ(`internal/config/config_test.go:501`、`:539`、`:559`): 鍵がないとき true、型の違う値で読み込みが失敗すること、雛形の値を false にしたときの読み込み
- `internal/config/defaults_sync_test.go`: 雛形の `ralph.toml` と `Default()` が鍵の値で一致すること。雛形から鍵を消したときと値を反転したときに落ちる場合を足した
- `TestDoctorCheckCodexSeatNetwork_Matrix`(`internal/cli/doctor_codex_network_test.go:53`)と、すべての役が claude でも codex が `driver_pool` にあれば出す場合(`:119`)、director だけが workspace-write の場合を数えないこと(`:140`)、`runDoctor` の通しの行(`:197`)

## 4. 確かめたこと

- `./scripts/run-test.sh` rc 0(test report)、`./scripts/run-static-verify.sh` pass(verify report)
- `codex sandbox` と `codex exec --sandbox workspace-write` で、`-c sandbox_workspace_write.network_access=false` が設定ファイルの true を上書きして curl が名前を引けずに落ち、`=true` で 200 が返ることを確かめた(codex-cli 0.166.0、`docs/evidence/codex-seat-network-2026-11-21.md`)
- cross-review(codex)は指摘なし

## 5. 残したもの

- herdr の pane に立てた座席の TUI の中で tflint の初期化が通ることは確かめていない(証拠の「確かめていないこと」)。設定ファイルに鍵がないマシンでの codex の既定も、このマシンでは `false` の上書きで代わりに確かめた
- tech-debt の行(`docs/tech-debt/README.md:547`): `ralph org spawn` は `[org.permissions]` を呼んだ側の `ralph.toml` から読み、main の worktree の `ralph.toml` からは読まない。機能の org の worktree で鍵を変えると、その worktree の版が効く
- 動いている座席は、鍵を変えても古い引数のまま。立て直すまで効かない(recipe の「The network of a codex seat」の節、`docs/recipes/codex-seat-permissions.md:178`)
