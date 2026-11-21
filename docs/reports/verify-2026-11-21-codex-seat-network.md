# Verify report: codex-seat-network

- Date: 2026-11-21
- Plan: `docs/plans/active/2026-11-21-codex-seat-network.md`(issue #362)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(base `origin/main` 09d27450 = merge-base、HEAD b6c28c1a、26 ファイル)。コードは S1 574f8a6e、S2 f9113e91、S3 cbb831bc と、self-review の LOW 2〜5 を直した b6c28c1a(文書とコメントとテストの定数、`testOrgConfig` の 1 行)。静的検査と仕様の確かめは b6c28c1a を含む HEAD で行った。ユーザーの決定(codex の座席は既定で通信を許す)は所見にしない。テストは打っていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-21-codex-seat-network.log`(gitignore の対象。wrapper 自身の log は `docs/evidence/verify-2026-11-21-154723.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 既定の設定で、codex の autonomous と edits の座席(leader、implementer、reviewer)の引数に `-c sandbox_workspace_write.network_access=true` が入る。guarded の codex の座席と claude の座席の引数は変わらない | met(静的) | `internal/org/permissions.go:181,186` が `codexAutonomousArgs` と `codexEditsArgs` に `codexNetworkAccessArgs`(`:97`)を足す。`permissionArgsForDriver` は役を見ないので、leader・implementer・reviewer の別はない(呼び元は `spawn.go:781,949,1478` と `director_start.go:1083`)。並びは `spawn.go:1153-1157` で「権限の引数 → `--add-dir` → `--model`」。guarded は `nil, nil`、claude の分岐は変わっていない。既定値は `config.go:852` の `CodexNetworkAccess: true`。テストは `TestPermissionArgsForDriver_CodexNetworkAccess`(`permissions_test.go:186`、driver 2 × mode 3 × 鍵 2 × codex_verified 2)、`TestOrgSpawn_Codex_NetworkAccessInStartArgs`(`spawn_test.go:4220`、implementer の argv 全体)、leader と機能の org の reviewer は既存の `--add-dir` の 2 つのテストが `codexNetworkOnArgs` 込みの argv を見る |
| AC2: `codex_network_access = false` なら同じ座席の引数に `-c sandbox_workspace_write.network_access=false` が入る | met(静的) | `codexNetworkAccessArgs` は false で `=false` を返す(渡さないのではなく明示する)。上の 2 つのテストの false の行が確かめる |
| AC3: `ralph config get org.permissions.codex_network_access` が既定で `true`、雛形と `config.Default()` が一致する | met | HEAD から build した binary を git の外の scratch で打った(`RALPH_LOCAL_CONFIG=off`): ralph.toml なしで `true`、雛形の写しで `true`、`= false` で `false`、`= "false"` で exit 3(bool に decode できない)。雛形は `templates/base/ralph.toml:239` に `codex_network_access = true`。`orgTemplateDrift` に鍵の行が入り(`defaults_sync_test.go:95,146`)、削除と両側の比較の検査にも入っている。テストを通すことは `/test` の担当 |
| AC4: doctor は workspace-write の codex の座席を立てうるとき、true なら info、false なら warn、どちらも動いている座席に効かないことを言う。役がすべて claude でも driver_pool に codex があれば出す。codex がない、または codex_verified が false なら出さない | met(info と warn を出さない、の読みで。I2 を参照) | `checkCodexSeatNetwork`(`doctor_codex_network.go:59`)は Check 11b の `codexSeatPossible` と `codexSeatModesPossible` を使う。`ralph init --yes` の scratch で HEAD の binary を打った: 既定で `ℹ … info`、鍵 false で `⚠ … warn`(exit 0、`--strict` でも exit 0 で warn のまま)、`codex_verified = false` と `driver_pool = ["claude"]` は `✓ … pass — not needed: …`。info と warn の文はどちらも `codexSeatNetworkReseatClause`(動いている座席は起動し直すまで引数を保つ)で終わる。雛形の `[org.role_models]` はすべて claude で、info が出た。テストは `TestDoctorCheckCodexSeatNetwork_Matrix`(`:53`、全 claude の `RoleModels`)ほか 3 本 |
| AC5: recipe に座席の通信の節があり、座席の引数を書く文書が新しい引数を書く。check-sync と check-skill-sync が通る | met | recipe の「The network of a codex seat」(`:178`)、README の Known differences の行(`README.md:356`)、org skill の 4 写し(`:2603-2623`)、`config.go` と雛形のコメント、`permissions.go` の doc。`git grep 'ask-for-approval never'` の残り(tech-debt の 276 行、`config.go:286`、`permissions.go:59,70`、雛形 `:216`)はどれも今の動きと合う。`check-sync.sh` は `DRIFTED: 0`、`check-skill-sync.sh` は 14 skill が lock-step(wrapper の中)。recipe の 2 写しと skill の 4 写しは `cmp` で同一。`.codex/README.md:65` は新しい引数に触れていない(V1、誤りではない) |
| AC6: 既存の座席の引数のテストが新しい引数を足して通る | partially met(静的) | `--add-dir` の 2 つのテストは期待の argv に `codexNetworkOnArgs` を足し(`spawn_test.go` の diff)、`VerifiedUnlocksMapping` は期待に `-c` を足した。期待を弱めた変更はない。`ask-for-approval` か `workspace-write` を期待に書く他のテストを grep し、ネットワークの `-c` を抜いたままの argv の比較はなかった。通ることは `/test` が確かめる |
| AC7: recipe が、鍵の変更と版の上げ下げはこれから立てる座席にだけ効くこと、立て直しと古い pane の確かめ方、途中で失敗したときの手順を書く | met | recipe の「A change reaches only the seats started after it」(`:271`)。主張をコードと突き合わせた: reseat が `[org.permissions]` を `--config` か打った場所の ralph.toml から読む(`resolveOrgStartConfig`、`applyMainWorktreeOrgLimits` は `Permissions` を写さない)、`--dry-run`、機能の org 以外は stop と spawn の打ち方を出して断る(`orgReseatHelp`)、`started:` の行の末尾の pane(`orgReseatStartSeat`)、`stop failed:`・`start failed:`・`left stopped:`・`left running:`・`to start the seats left, run:`(`org_reseat.go:602-659`)、`ralph org status --live` の `spawned (active)`(`org.go:2159`)、座席の pane の label は seat id(`spawn.go:2643`)、`spawn` の idempotent(`spawn.go:1691`) |
| issue #362 の受け入れ条件 1: 通信の要る検査が gate の中で通る、または通らないことを doctor が前もって知らせる | met(座席の中の通りは未確認) | 既定では ralph が `=true` を渡す。`-c` が設定ファイルに勝つことは `codex sandbox` で 1 回確かめた(`docs/evidence/codex-seat-network-2026-11-21.md`)。座席の TUI の中で tflint が通ることは確かめていない(同 evidence の「確かめていないこと」)。鍵が false のときは doctor が warn を出す(上の AC4 の実行)。guarded の codex の座席は人の codex config に従い、doctor は何も言わない(計画の Non-goals) |
| issue #362 の受け入れ条件 2: codex の座席の通信の扱いが recipe に書いてある | met | recipe の `:178-325` の 3 節(既定と鍵、確かめ方、効く範囲と立て直し) |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(既定の changed scope) | pass(`rc=0`、`All verifiers passed.`) | `Language scope: full fallback (unclassified:templates/base/ralph.toml)`、`Language packs selected: golang`。local verifier(shellcheck、`sh -n`、jq、check-sync `DRIFTED: 0`、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt README plan references)はすべて OK |
| gofmt(golang verifier の中) | pass | `gofmt: ok` |
| `go vet ./...`(同) | pass | 出力なし |
| `golangci-lint run ./...`(同) | pass | `0 issues.` |
| `staticcheck ./...`(同) | pass | 出力なし |
| branch secret scan(同) | pass | `scanned 09d27450..b6c28c1a against origin/main: clean` |
| `git diff --check origin/main...HEAD` | pass | 出力なし |
| コミットの帰属の行 | なし | `git log origin/main..HEAD` の本文に `Co-Authored-By` と `Generated with` はない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| recipe `docs/recipes/codex-seat-permissions.md`(と雛形の写し、`cmp` で同一) | yes(1 か所 LOW、V4) | 新しい 3 節の主張は上の AC7 のとおりコードと合う。b6c28c1a の限定(`-c` が設定ファイルに勝つのは `codex sandbox` で 1 回確かめただけ、座席の中では未確認)が `:206-210` に入った。cross-review の節の `:333-334` は既定の変化を反映していない(V4) |
| org skill の 4 写し(`:2603-2623`) | yes | 既定、鍵、b6c28c1a の限定、doctor の項目、立て直し(機能の org は reseat、ほかは stop と spawn)が recipe と合う |
| director skill の 4 写し | yes | headless の codex の director は今も guarded だけ(`:309-316`)。`:981` の「network がないと」は条件つきで誤りではない(I5) |
| README の Known differences(`README.md:356`) | yes | `-c …=true` と、鍵の false で `=false` になることを書く |
| `.codex/README.md:65` と `templates/base/.codex/README.md:65` | no(LOW、V1) | 同じ表の行が「mapped to Codex's own `sandbox_mode` + `approval_policy` flags」のままで、通信の `-c` に触れない |
| `templates/base/ralph.toml:224-239` のコメント | yes | 既定、false の動き、確かめた範囲、guarded と claude に効かないこと、reseat は機能の org の座席(b6c28c1a の L2) |
| `internal/config/config.go:275-316` のコメント | yes | 雛形と同じ内容。`OrgPermissionsConfig` の doc と `CodexVerified` の doc にも 1 文ずつ足した |
| `internal/org/permissions.go` の doc | yes(INFO、I3) | `:158-161` の「not that file, decides」は限定がない。限定は同じファイルの `codexNetworkAccessArgs` の doc(`:92-93`、「on codex-cli 0.166.0」)にある |
| doctor の項目の文(`doctor_codex_network.go:72-84`、`doctor.go:227-234` のコメント) | yes | b6c28c1a で「ralph が座席をどの引数で立てるか」の文に寄せた(L4)。scratch の実行の出力がテストの定数と一字一句同じ |
| `docs/evidence/codex-seat-network-2026-11-21.md` | yes | 確かめた 3 通りと確かめていない 2 点を分けて書く。計画のパスを `docs/plans/archive/` で書いており、/pr の archive のあとに合う(今は先の参照) |
| `docs/specs/2026-10-07-org-multi-org-director.md:284`、`docs/specs/2026-08-01-org-runtime.md:290` | yes | どちらも通信に触れず、今も正しい |
| README の doctor の行(`README.md:137`) | yes(INFO、I5) | 個々の codex の検査を並べない要約で、agmsg writable root の検査も載っていない。新しい項目を足さなくても誤りではない |
| 計画の Progress checklist | no(LOW、V5) | 下の Findings を参照 |
| `docs/tech-debt/README.md` | no(LOW、V2、V3) | 下の「Tech-debt の Trigger の確かめ」を参照 |

### Findings

| ID | Severity | File:line | Finding | For /sync-docs |
| --- | --- | --- | --- | --- |
| V1 | LOW | `.codex/README.md:65`、`templates/base/.codex/README.md:65` | 「How Codex differs」の表の「Org seat permission policy」の行が、`README.md:356` の同じ行と違って通信の `-c` に触れていない。誤りではないが、2 つの表の Codex の列がずれた(S3 の範囲外の気づき (1)) | 2 写しの行に README.md:356 と同じ句(`edits` と `autonomous` に `-c sandbox_workspace_write.network_access=true`、`codex_network_access = false` で `=false`、recipe を指す)を足す。`check-sync.sh` が 2 写しを同一に保つ |
| V2 | LOW | `docs/tech-debt/README.md:135,136,138,376,420` | この branch が Trigger を引いたのに、持ち越しの記録がない行が 5 つある。135: `doctor_codex_writable_root.go` に触れた(`codexSeatModesPossible` の doc に 5 行、1009 → 1014 行)。136: `doctor_shell_alias.go` に触れ、残りの Trigger「ralph が `[org.permissions]` のフラグを driver に渡す仕方の次の変更」も `permissionArgsForDriver` の `-c` で引いた。設定を読んだ grading は払っていない。138: `doctor_shell_alias.go` の次の変更(1448 → 1460 行、分割なし)。376: recipe の 2 写しを編集し、`docs/evidence/codex-seat-network-2026-11-21.md` の引用を「the ralph repository's」として足した(引用は 3 ファイル 3 件から 4 ファイル 4 件)。新しい引用は行の求める書き方だが、`:14` などの古い 3 件は書き換えていない。420: self-review の L1。`doctor.go` 1683 → 1692、`config.go` 1195 → 1219(self-review の 1215 は b6c28c1a の前の数) | 5 行の Trigger の末尾に、前例と同じ形の日付つきの持ち越し(引いた理由、行数、払わなかった理由、Scope)を足し、Related に計画のパスを足す。行数は merge-base 09d27450 と b6c28c1a の `wc -l` |
| V3 | LOW | `docs/tech-debt/README.md:231`、`:433` | 一部が解消した行。231 の Trigger「a `gate_failed` whose log shows a sandbox denial」は、issue #362 の報告そのもの(codex の reviewer 座席の `run-static-verify.sh` で tflint が sandbox の通信止めで落ちた)が引いた。通信の部分はこの PR の既定 `=true` で手当てされたが、Go の build と module の cache、座席の中での実地の確かめは残る。433 の Trigger は「reviewer 座席に通信を与えるかを決める」と言い、この PR が既定で与えると決めたので、debt の文「may not reach the network」は既定では古い(S3 の範囲外の気づき (2)) | 231 に、#362 が引いたこと、通信は既定で開いたこと(鍵 false で閉じる)、残るもの(cache、座席の中の実地)を書く。433 に、codex の reviewer 座席は既定で通信を受けること、座席の中で `claude -p` や入れ子の `codex exec review` が通るかは未確認のままであることを書く。どちらも行は閉じない |
| V4 | LOW | `docs/recipes/codex-seat-permissions.md:333-334`(と雛形の写し) | cross-review の節の「which a codex reviewer seat under `--sandbox workspace-write` (edits and autonomous) may not give it」は、既定の変化(ralph が既定で通信を許す)を言っていない。今は鍵が false のとき(と、guarded で人の設定が閉じているとき)だけの話になった。条件つきの文なので誤りではない | 「`codex_network_access = false` のとき」を足し、上の節を指す。レーン B の #360 が同じ recipe と skill を触るので、その PR と順を合わせる(S3 の気づき (3)) |
| V5 | LOW | `docs/plans/active/2026-11-21-codex-seat-network.md:118-119` | 「Review artifact created」が `[ ]` のまま(self-review の report は 4dfa57bb でコミット済み)。b6c28c1a(self-review の LOW 2〜5 の直し)の行が Progress にない | 「Review artifact created」と「Verification artifact created」に印を付け、b6c28c1a の 1 行(L2〜L5 を直した、挙動の変更なし)を足す |
| I1 | INFO | `docs/tech-debt/README.md:279`、`internal/cli/doctor_codex_network_test.go:140` | 279 (a) の「`codexNotNeededWhyA` の文は role director だけが edits のとき字義どおりには正しくない」が、新しい検査にもそのまま出る(同じ関数を使う)。新しいテスト `TestDoctorCheckCodexSeatNetwork_DirectorOnlyWorkspaceWriteIsNotCounted` がその文を固定したので、279 (a) を払う人は 2 つ目のテストの期待も変える必要がある。Trigger(`codexNotNeededWhyA` かその既存のテストの次の編集)は引いていない | 279 (a) に新しいテストの名前と、新しい検査も同じ文を出すことを足す |
| I2 | INFO | 計画 `:22`(Scope の doctor の項目)と `:67`(AC4) | Scope は「立てえないときはこの項目を出さない」、実装は `pass` の `not needed:` の行を出す(Check 11b の慣習)。S2 の Progress にこの選択が書いてある。AC4 の「出さない」は info と warn を出さない、と読んで met にした | 要らない。PR の本文で触れるなら 1 文 |
| I3 | INFO | `internal/org/permissions.go:158-161` | `permissionArgsForDriver` の doc の「so `[org.permissions].codex_network_access`, not that file, decides whether the seat reaches the network」は、self-review の L3 の限定を受けていない。同じファイルの `:92-93` に限定がある | 要らない(コードのコメント。直すなら次にこのファイルに触れるとき) |
| I4 | INFO | `docs/tech-debt/README.md`(末尾) | origin/main が 9cd63d60(#367)に進み、`git merge-tree --write-tree --name-only origin/main HEAD` が `docs/tech-debt/README.md` の衝突を出す。どちらも表の末尾に行を足したため | /pr の前に origin/main を取り込み、2 行とも残す |
| I5 | INFO | `.claude/skills/director/SKILL.md:981`、`internal/org/prompts/leader.md:595,864`、`README.md:137` | 「network がないと」「ネットワークのない sandbox など」は条件つきで、今も誤りではない(S3 の気づき (3)(4))。README の doctor の行は検査を並べない要約 | 要らない |
| I6 | INFO | (次の release) | 計画の Risks が言う release ノートの項目: codex の座席が既定で通信を受ける、鍵のない下流の `ralph.toml`(seed)は true で読む、動いている座席は立て直すまで前の引数のまま。`internal/upgrade/notes/` は `/release` で書くので今は何もしない | PR の本文の release ノートの項目に入れる |

### Tech-debt の Trigger の確かめ

`docs/tech-debt/README.md` の開いている行を ` | ` で分け、4 つめの列(Trigger)を、この branch が触ったファイル名、関数名、語(`permissions.go`、`config.go`、`doctor.go`、`doctor_shell_alias`、`doctor_codex`、`codex-seat-permissions`、`network`、`org.permissions`、`codex_verified`、`permissionArgsForDriver`、`org/SKILL`、`/org`、`Known differences`、`templates/base/ralph.toml`、テストの名前、`any of the five`、`tflint`、`workspace-write`、`codexWritableRootArgs`、`codex seat`、`codex reviewer`、`recipe`、`Default()`、`mapping`、`.codex/README` など)で探した。当たった行を 1 つずつ読んだ。

- 引いた(V2): 135、136、138、376、420
- 一部が解消(V3): 231(issue #362 の報告が Trigger)、433(Trigger の「決める」をこの PR が決めた)
- 引いていない: 88(項目のファイルに `spawn_test.go` の codex の argv テストは入らない)、146(`doctor_codex_writable_root_test.go` は変えていない)、153(`DefaultRoleModels` は変えていない)、158(`/cross-review` と `/plan` の codex の呼び出しは変えていない)、161(org の品質パイプラインの手順・判定・記録は変えていない)、176(`codexWritableRootArgs` の本体は変えていない)、182 (a)(`/org` の command table は変えていない)、276 と 281(director の起動と mapping は変えていない)、369(spawn の失敗の道のテストではない)、370(`spawn_test.go` は 6448 → 6508。行が最後に記録した 6408 + 300 = 6708 の内)、458(`[org]` の検証は変えていない。bool の鍵に検証はない)、519(reseat のコードは変えていない)
- 新しい行 546(spawn が呼んだ側の `ralph.toml` から `[org.permissions]` を読む)は、`resolveConfig`(`org.go:691`)、`applyMainWorktreeOrgLimits`(`:242-249`、`Permissions` を写さない)、reseat の `resolveOrgStartConfig` を読んで、書いてあるとおりと確かめた。Related は `docs/plans/active/` のパスで、plan-refs の gate を通る

## Observational checks

- HEAD から build した binary の `ralph doctor` を `ralph init --yes` の scratch で 4 通り打った(既定、鍵 false、`codex_verified = false`、`driver_pool = ["claude"]`)。出力の行は evidence の log にある。鍵 false は exit 0、`--strict` でも exit 0 で warn の行が 1 本
- `ralph config get org.permissions.codex_network_access` を git の外の scratch で 4 通り打った(上の AC3)
- 座席の引数の並びは、codex が `-c` を何度でも受け、同じ鍵は後の値が勝つことに頼る(別名の `-c` との順)。recipe と `doctor_shell_alias.go` の新しい段落は、これを codex 0.166.0 のソースで読んだだけで座席では動かしていないと書いており、主張の範囲は記録と合う
- 鍵を false にしても動いている座席は通信を保つ。recipe、org skill、doctor の文、雛形と `config.go` のコメントのすべてがこれを言う

## Coverage gaps

- テストは 1 本も打っていない(`/test` の担当)。AC1、AC2、AC6 の「met」は読みと静的検査による
- 座席の TUI の中で `-c sandbox_workspace_write.network_access=true` が効き、tflint の初期化が通ることは誰も確かめていない(evidence の「確かめていないこと」)。issue #362 の受け入れ条件 1 の「通る」の側は、`codex sandbox` での 1 回の確かめと引数の固定による
- 設定ファイルに `network_access` がないマシンでの codex の既定の値は確かめていない(evidence)
- guarded の codex の座席の通信は人の codex config に従い、doctor は何も言わない(計画の Non-goals)
- 次に足すと効く確かめ: 分離した herdr で codex の reviewer 座席を 1 つ autonomous で立て、その pane で `curl -sS -o /dev/null -w '%{http_code}' https://example.com` を打たせ、鍵 true で 200、false で 000 になることを記録する(recipe の「Check it on your machine」を座席の中に移したもの)。決定的な静的検査の不足はない

## Verdict

- Verified: AC3(binary で 4 通り)、AC4(binary で 4 通り、`--strict` を含む)、AC5、AC7、issue の受け入れ条件 2、静的検査のすべて(`run-static-verify.sh` rc=0)
- Partially verified: AC1、AC2、AC6(コードとテストを読んで met、テストの実行は `/test`)、issue の受け入れ条件 1(座席の中の通りは未確認)
- Not verified: 座席の中での `-c` の効き、tflint の初期化、設定ファイルに鍵がないマシンの既定
- Verdict: **pass**。fail にする所見はない。LOW 5 件(V1〜V5)はどれも文書と記録の直しで、`/sync-docs` が拾う
