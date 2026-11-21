# codex-seat-network

- Status: Approved
- Approved: 2026-11-21 sha256:2267f124015d
- Owner: Claude Code
- Date: 2026-11-21
- Related request: issue #362(codex の座席の sandbox が外への通信を止め、tflint の初期化と gate が落ちる)
- Related issue: 362
- Type: fix
- Branch: fix/codex-seat-network

## Objective

codex の座席を workspace-write の sandbox で立てるとき、ralph が既定で外への通信を許す。gate や検査がプラグインや依存を取りに行けず(tflint の初期化など)落ちることをなくす。通信を閉じたい人のために設定の鍵を足し、閉じているときは `ralph doctor` が前もって知らせ、recipe に書く。

ユーザーの決定(2026-11-21): codex の座席は既定ですべて通信を許す。

## Scope

- `internal/org/permissions.go`: codex の autonomous と edits(どちらも `--sandbox workspace-write`)の座席の引数に `-c sandbox_workspace_write.network_access=true` を足す。leader、implementer、reviewer のすべての座席が対象。guarded は今のとおり何も足さない(人の設定に従う)
- 設定: `[org.permissions].codex_network_access`(bool、既定 true)。false なら `-c sandbox_workspace_write.network_access=false` を渡して通信を閉じる(人の設定ファイルの値によらず閉じる)。`internal/config/config.go` の `OrgPermissionsConfig`、既定値、`templates/base/ralph.toml` のコメントと値、`ralph config get` で読めること、`internal/config/defaults_sync_test.go` の一致
- `ralph doctor`: codex の座席の通信を知らせる項目。codex の座席を立てうるかは、既存の writable root の検査と同じ判定で決める(`internal/cli/doctor_codex_writable_root.go` の `codexSeatPossible` と `codexSeatModesPossible`: codex が `[org].driver_pool` にあれば、`[org.role_models]` が claude の役でも `--driver codex` で codex の座席を立てられる)。`codex_verified` が false なら codex の座席は guarded に限られるので数えない。workspace-write の codex の座席を立てうるとき、`codex_network_access` が true なら info(ralph が座席に通信を許す `-c` を渡す)、false なら warn(通信の要る検査、たとえば tflint の初期化や依存の取得が座席の中で落ちる、と言う)。どちらの文も、鍵の変更はこれから立てる座席にだけ効き、動いている座席には `ralph org reseat` で立て直すまで効かないと言う。立てえないときはこの項目を出さない
- `docs/recipes/codex-seat-permissions.md`: 座席の通信の節(既定、鍵、閉じたときに落ちるもの、確かめ方 `codex sandbox -c ... -- curl ...`)。鍵の変更と ralph の版の上げ下げは、これから立てる座席にだけ効く(座席の引数は起動のときに決まり、`ralph org spawn` は動いている座席を立て直さず、`ralph org resume` は文字を打つだけ)。動いている座席に効かせるには、その org の座席を `ralph org reseat` で立て直し、`ralph org status --live` か herdr の一覧で古い座席の pane が残っていないことを確かめる。立て直しが途中で失敗したときは、reseat が出す打ち直しのコマンドを打ち、同じく確かめる
- 座席の引数を書く文書(README、director と org の skill の 4 写し、`docs/specs/2026-10-07-org-multi-org-director.md`、`internal/config/config.go` のコメントなど、`--sandbox workspace-write --ask-for-approval never` を書くところ)を、新しい引数に合わせる

## Non-goals

- codex の headless の director(今も autonomous と edits の codex の director は断る)
- guarded の codex の座席の通信(人の設定ファイルに従う)
- claude の座席の通信(claude は今も sandbox で通信を止めない)
- 通信先を絞ること(codex の workspace-write の sandbox には宛先の許可リストがない)
- 標準フローの cross-review と計画助言の codex(`--sandbox read-only` で走り、通信を止めるのは意図どおり)

## Assumptions

- codex-cli 0.166.0 で `-c sandbox_workspace_write.network_access=<bool>` は設定ファイルの値を上書きする。このマシンで、設定ファイルが true のまま `-c ...=false` を渡すと `curl` が名前を引けずに落ち、`true` で通った(`docs/evidence/codex-seat-network-2026-11-21.md` に移す)
- codex の既定は `network_access = false`(issue の本文)
- 座席の codex は対話の `codex` で、`-c` はどの位置でも効く(標準フローで問題になった「`exec` のあとの `-c`」は `codex exec` の話で、座席は `exec` を使わない)。実装で座席の引数の並びを確かめる

## Affected areas

- `internal/org/permissions.go` とテスト(`permissionArgsForDriver`、`codexAutonomousArgs`、`codexEditsArgs`)
- `internal/config/config.go`、`internal/config/*_test.go`(既定値と defaults_sync)、`templates/base/ralph.toml`、この repo の `ralph.toml`(必要なら)
- `internal/cli/doctor_codex_network.go`(新しい項目、新しいファイル)と、doctor の項目を並べる箇所(`internal/cli/doctor.go`)、テスト
- `docs/recipes/codex-seat-permissions.md`、README.md、director と org の skill(4 写しずつ)、`docs/specs/2026-10-07-org-multi-org-director.md`、AGENTS.md(必要なら)
- 共有ファイルの注意: `docs/recipes/codex-seat-permissions.md` と skill は、レーン B の #360 も触る

## Visual review

- ページ: `.harness/state/plan-visual/codex-seat-network.html`。かんたんな図 2 枚(`overview`、`easy-close`)と、詳しい図 2 枚(`detail-overview`、`detail-args`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 4 枚の図を `shot` で撮って読んだ。直したのは 1 点で、ファイルの地図の recipe のパスが箱からはみ出していたので、文書の群と箱を広げた。撮り直して、どれもきれいに出ることを確かめた

## Design decisions

Critical forks: None(通信を許す既定はユーザーが決めた)

- 鍵 `codex_network_access` を足す。既定 true でユーザーの決定どおり。false は、通信を許したくない人が設定ファイルを触らずに閉じる道で、閉じたときは doctor が warn で知らせる
- false のとき `-c ...=false` を明示して渡す。渡さないと人の設定ファイルの値が効き、鍵が false でも通信が開いたままになりうる
- 鍵は `[org.permissions]` に置く。座席の sandbox の扱いで、`codex_verified` と同じ表

## Acceptance criteria

- [x] AC1: 既定の設定で、codex の autonomous と edits の座席(leader、implementer、reviewer)の引数に `-c sandbox_workspace_write.network_access=true` が入る。guarded の codex の座席と claude の座席の引数は変わらない
- [x] AC2: `[org.permissions].codex_network_access = false` なら、同じ座席の引数に `-c sandbox_workspace_write.network_access=false` が入る
- [x] AC3: `ralph config get org.permissions.codex_network_access` が既定で `true` を返し、雛形の `ralph.toml` と `config.Default()` が一致する(defaults_sync のテストが通る)
- [x] AC4: `ralph doctor` は、workspace-write の codex の座席を立てうるとき(`codexSeatPossible` と `codexSeatModesPossible`、`codex_verified` を見る)、鍵が true なら info、false なら warn(通信の要る検査が座席で落ちる、と言う)を出し、どちらも鍵の変更が動いている座席に効かないことを言う。`[org.role_models]` の役がすべて claude でも codex が `driver_pool` にあれば出す(テストで確かめる)。codex が `driver_pool` にない、または `codex_verified` が false なら出さない
- [x] AC7: recipe が、鍵の変更と ralph の版の上げ下げはこれから立てる座席にだけ効くこと、動いている座席の立て直しと古い pane が残っていないことの確かめ方、立て直しが途中で失敗したときの手順を書く
- [x] AC5: recipe に座席の通信の節があり、座席の引数を書く文書が新しい引数を書く。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る
- [x] AC6: 既存の座席の引数のテスト(`--add-dir` の並び、`--model` の前に置くことなど)が、新しい引数を足して通る

## Implementation outline

1. S1 座席の引数と設定の鍵(AC1〜AC3、AC6) — verify: `test -z "$(gofmt -l internal/org internal/config)" && go vet ./internal/org/ ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Permission|Spawn|CodexWritable'`
2. S2 doctor の項目(AC4) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Doctor'`
3. S3 文書(AC5、AC7)と実機の記録 — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill|Director|DocsMatch' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Agents|Readme|Docs|Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC6 と issue #362 の受け入れ条件 2 つ
- Documentation drift to check: 座席の引数を書く文書すべて(`git grep 'ask-for-approval never'`)、recipe、`ralph doctor` の項目の説明、`[org.permissions]` のコメント
- Evidence to capture: verify の report、`docs/evidence/codex-seat-network-2026-11-21.md`

## Test plan

- Unit tests: 役と mode と鍵の組み合わせごとの座席の引数(autonomous・edits・guarded × true・false × claude・codex)、設定の読み(既定、false、型の誤り)、doctor の項目(codex の役あり・なし、true・false、guarded だけの codex)
- Integration tests: `ralph org spawn --dry-run` か spawn のテストで、codex の座席の起動の引数に `-c` が入る
- Regression tests: 既存の permissions と spawn と doctor のテスト
- Edge cases: `codex_verified = false`(codex の座席は guarded に限られるので `-c` は入らない)、`--add-dir` と一緒の並び
- Evidence to capture: `./scripts/run-test.sh` の log、`-c` を外す変異でテストが落ちること

## Risks and mitigations

- 座席が外へ通信できるようになる(既定): ユーザーの決定。claude の座席は今も通信できるので、codex の座席だけを閉じる理由は薄い。閉じたい人は鍵を false にし、doctor が warn で知らせる
- 他のレーン(B の #360)が recipe と skill を触る: テストが固定する文を変える前に知らせ、merge の前に main を取り込む
- 鍵の追加で、雛形の `ralph.toml` を持つ下流が upgrade で鍵を得ない(雛形は seed): 鍵がなければ既定 true で読む。release ノートに書く
- 鍵を false にしても、動いている座席は通信できたまま(codex の計画助言の 1): 引数は起動のときに決まる。doctor と recipe で、立て直すまで効かないことと立て直しの手順を書く
- `--driver codex` で立てた座席を doctor が見落とす(codex の計画助言の 2): 既存の writable root の検査と同じ判定(driver_pool と permission mode)を使う

## Rollout or rollback notes

この PR を revert すれば、これから立てる codex の座席は人の設定ファイルの `network_access` に従う今の動きに戻る。動いている座席は起動のときの引数のまま通信できるので、閉じるには `ralph org reseat` で立て直す(recipe に書く)。鍵を書いた `ralph.toml` は、revert のあと知らない鍵として読まれるかを確かめる(読みが知らない鍵を断るなら release ノートに書く)。

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 2 件、どちらも計画に入れた。[MEDIUM] 1: 鍵を変えても revert しても動いている座席には効かない(引数は起動のときに決まり、spawn は動いている座席を立て直さず、resume は打つだけ) → doctor の文と recipe に、立て直すまで効かないこと、`ralph org reseat` での立て直しと古い pane が残っていないことの確かめ、途中で失敗したときの手順を書く(Scope、AC4、AC7、Rollout)。[MEDIUM] 2: `--driver codex` で立てた座席を doctor が見落とす → 既存の `codexSeatPossible` と `codexSeatModesPossible` を使い、`codex_verified` を見る。役がすべて claude でも codex が driver_pool にあれば出すことをテストする(AC4)。図解の「気をつけること」にも 1 行足した。git の状態は助言の前後で同じ
- [x] Plan approved — 2026-11-21、系列の決まり(計画は推す案で自動承認)で承認した。人の目での承認はしていない
- [x] Branch created
- [x] Implementation started
  - S1(574f8a6e、implementer): 確認コマンドと `./scripts/check-sync.sh` が exit 0。`codexNetworkAccessArgs(cfg)` を足し、並びは「権限の引数 → `-c sandbox_workspace_write.network_access=<bool>` → `--add-dir` → `--model` → prompt」。`permissionArgsForDriver` は毎回新しい slice を返す。`[org.permissions].codex_network_access`(`Default()` は true、雛形の `ralph.toml` に値とコメント)。`ralph config get org.permissions.codex_network_access` はビルドした binary で true / false / 型の誤りで exit 3 を確かめた。変異 3 種でテストが落ちる。範囲外の気づき: (1) spawn は `[org.permissions]` を呼んだ側の `ralph.toml` から読むので、機能の worktree の leader が写しを書き換えれば自分が立てる座席の通信を変えられる(`codex_verified` と permission mode も今は同じ扱い。この PR では変えず tech-debt に記録する)、(2) `internal/cli/org.go:292` のコメントが写す Permissions の鍵を古い列挙で書く(S3 で直す)、(3) `internal/cli/doctor_shell_alias.go` 999〜1010 行付近の座席の codex の引数の列挙に `-c` がない(S2 で見る)
  - S2(f9113e91、implementer): 確認コマンドが exit 0。`ralph doctor` に「Codex sandbox (seat network)」の行(`internal/cli/doctor_codex_network.go`、`doctor.go` の Check 11b の直後)。既存の `codexSeatPossible` と `codexSeatModesPossible` を使う。鍵が true なら info、false なら warn で、どちらも鍵の変更はこれから立てる座席にだけ効き、機能の org なら `ralph org reseat` が立て直すと言う。立てえないときは pass の `not needed: …`(既存の writable root の検査の慣習)。warn は `--strict` でも warn のまま。役がすべて claude でも codex が driver_pool にあれば出すことをテストした。変異 5 種でテストが落ちる。`doctor_shell_alias.go` はコメントだけ(codex は `-c` を何度でも受け、同じ鍵は後が勝つので、alias のあとに来る ralph の `-c` が効く。codex 0.166.0 のソースで確かめ、座席では動かしていない)。範囲外の気づき: README の doctor の行、`CodexModeRunsUnderWorkspaceWrite` のコメントが使う検査に新しい項目を挙げていない、`internal/cli/doctor.go` が 1692 行(tech-debt の 800 行の行に持ち越しが要る)
  - S3(cbb831bc、implementer): 確認コマンドと `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` が exit 0。recipe(と雛形の写し)に「The network of a codex seat」の節(既定、鍵、落ちるもの、確かめ方、これから立てる座席にだけ効くこと、立て直しと古い pane の確かめ、途中で止まったとき、`[org.permissions]` は打った場所の `ralph.toml` から読むこと、alias の `-c` は効かないこと)。org skill の 4 写し、README の Known differences の表、Go のコメント 3 つ、`docs/evidence/codex-seat-network-2026-11-21.md`、tech-debt の 1 行(spawn が呼んだ側の `ralph.toml` から `[org.permissions]` を読む)。director skill と仕様と director の断りの文は今も正しいので変えていない。範囲外の気づき(sync-docs に渡す): (1) `.codex/README.md` 65 行目と雛形の写しの Known differences の行が通信の `-c` に触れていない、(2) tech-debt の 231 行目付近(codex の reviewer が gate のスクリプトを打てるか)と 433 行目付近(機能の org の cross-review、trigger が「reviewer 座席に通信を与えるかを決める」)は一部が解消した、(3) director skill の 981 行目付近と recipe の cross-review の節の「network がないと」(条件つきで誤りではない、レーン B の #360 の範囲)、(4) leader.md の 595・864 行目の「ネットワークのない sandbox など」(条件つきで誤りではない)
  - b6c28c1a(self-review の LOW 2〜5 の直し、挙動は変えない): L2 鍵のコメント(雛形の `ralph.toml` と `config.go`)が、`ralph org reseat` は機能の org の座席を立て直し、ほかの org の座席は stop と spawn で立て直すと言う。L3 `-c` が設定ファイルに勝つことは `codex sandbox` で 1 回確かめただけで座席の中では未確認、という限定を recipe(2 写し)、org skill(4 写し)、雛形、`config.go` に足した。L4 doctor の info と warn の文を「ralph が座席をどの引数で立てるか」に寄せ、テストの定数を合わせた。L5 `testOrgConfig` が `CodexNetworkAccess: true` を持つ
  - 50a47846(evidence): 座席と同じ `--sandbox workspace-write` の `codex exec` のエージェントの中でも、`-c sandbox_workspace_write.network_access=<bool>` が設定ファイルより優先して効くことを `docs/evidence/codex-seat-network-2026-11-21.md` に足した(`false` で名前が引けず、`true` で 200)。座席の TUI の中は未確認のまま
- [x] Review artifact created — `docs/reports/self-review-2026-11-21-codex-seat-network.md`(4dfa57bb。Merge: yes、LOW 5 件。L2〜L5 は b6c28c1a で直し、L1 は sync-docs が tech-debt の 800 行の行(`internal/cli/doctor.go` と `internal/config/config.go` を挙げる行)に持ち越しを書いた)
- [x] Verification artifact created — `docs/reports/verify-2026-11-21-codex-seat-network.md`(f3214c5b。pass、LOW 5 件 V1〜V5 と INFO 6 件 I1〜I6。V1〜V5 と I1 は sync-docs で直し、I2〜I6 は直さない)
- [x] Test artifact created — `docs/reports/test-2026-11-21-codex-seat-network.md`(3a31078d。pass、`run-test.sh` rc 0、変異 6 つがどれも落ちる、test gaps 4 件。gap 1 は sync-docs が `TestConfigGet_NoRalphTomlPrintsTheDefaults` に行を足して塞いだ。gap 2 の座席の TUI の中は tech-debt の 2 行(codex の reviewer の gate と、機能の org の cross-review)と evidence に未確認として残した。gap 3 は evidence の「確かめていないこと」、gap 4 は tech-debt の既存の 1 行)
- [x] Sync-docs run 1(`docs/reports/sync-docs-2026-11-21-codex-seat-network.md` を含む commit): verify の V1〜V5 と I1 を直した。V1 は `.codex/README.md` の 2 写しに通信の `-c` の句、V2 は tech-debt の 5 行(Trigger に当たった行)に日付つきの持ち越し、V3 は 2 行に直した分と残る分、V4 は recipe の cross-review の節(2 写し)の 1 句、V5 はこの Progress と AC、I1 は tech-debt の 1 行。テストの穴 1 は `internal/cli/config_test.go` の表に 1 行足して塞いだ(`test:` の commit、`ConfigGet` の絞り込みが通り、期待値を `false` にすると落ちる)。test report のあとに足した Go の変更はこの 1 行だけ。AGENTS.md、README、quality-gates は変更なし
- [ ] PR created
