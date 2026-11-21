# sync-docs report: codex-seat-network

- Date: 2026-11-21(実行は 2026-11-22 の未明、JST。insight event の時刻は 2026-11-21T17:35Z)
- Plan: `docs/plans/active/2026-11-21-codex-seat-network.md`(承認の digest `2267f124015d`。この run の前後とも同じ)。issue #362
- Branch: `fix/codex-seat-network`。run 開始時の HEAD は `8fdd98de`(origin/main `9cd63d60` を取り込んだ merge commit)。origin/main は 9cd63d60 のまま
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `4dfa57bb`(Merge: yes、LOW 5 件。L2〜L5 の直しは `b6c28c1a`)、verify `f3214c5b`(pass、LOW 5 件 V1〜V5、INFO 6 件 I1〜I6)、test `3a31078d`(pass、`run-test.sh` rc 0、test gaps 4 件)。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない。evidence の `50a47846`(`codex exec` のエージェントの確かめ)は verify のあとに入った
- 見た範囲: この task の変更 `git diff 9cd63d60 HEAD`
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。Go の変更は `internal/cli/config_test.go` の表に 1 行だけ(下の「テストの穴 1」)。`## Progress checklist` と AC の箱のほかは計画の本文を変えておらず、`./scripts/plan-visual.sh digest docs/plans/active/2026-11-21-codex-seat-network.md` は編集の前後とも `2267f124015d`
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff 9cd63d60 HEAD` は 31 ファイル、+1570/-71(計画、report 3 本、insights、tech-debt、evidence、recipe、skill の写しを含む)。Go は 14 ファイルで +711/-48。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary

verify の V1〜V5 と I1 を直し、self-review の L1(800 行の行への持ち越し)を tech-debt に書いた。コードは変えていない。テストの穴 1 だけは、`ralph config get` の既定値の行を 1 行足して塞いだ。tech-debt の行は 8 行に日付つきの注記を足した。うち 5 行は Trigger に当たった行の持ち越し、2 行(codex の reviewer が gate の scripts を打てるかの行と、機能の org の cross-review の行)は直した分と残る分、1 行(`codexNotNeededWhyA`)は 2 つ目のテストの名前。どの行も閉じていない。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `.codex/README.md`、`templates/base/.codex/README.md` | 「Org seat permission policy」の行に、`README.md` と同じ句(`edits` と `autonomous` に `-c sandbox_workspace_write.network_access=true`、`codex_network_access = false` で `=false`、recipe を指す)を足した | `6cf1bad8` |
| `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md` | cross-review の節の「may not give it」に「when `codex_network_access = false`」と、上の節への指し先を足した。その節の 1 文だけ | `6cf1bad8` |
| `docs/tech-debt/README.md` | 8 行の Trigger 列と Related 列に日付つきの注記と計画のパスを足した(`git diff --numstat` は 8/8)。新しい行は足していない | `60c2d193` |
| `internal/cli/config_test.go` | `TestConfigGet_NoRalphTomlPrintsTheDefaults` の表に `{"org.permissions.codex_network_access", "true\n"}` を 1 行 | `eacf591e` |
| `docs/plans/active/2026-11-21-codex-seat-network.md` | AC1〜AC7 を `[x]`、Review・Verification・Test の artifact を `[x]` にして report の path を添えた。「Implementation started」に `b6c28c1a` と `50a47846` の 2 行、`Sync-docs run 1` の 1 行を足した | この report の commit |
| `docs/reports/sync-docs-2026-11-21-codex-seat-network.md` | この report | この report の commit |
| `docs/insights/events/2026-11-21-codex-seat-network.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行 | この report の commit |

## 依頼の項目

### V1: `.codex/README.md` の 2 写し

`README.md` の Known differences の行と同じ句を足した。`.codex/README.md` と `templates/base/.codex/README.md` は `cmp` で同一のまま。`./scripts/check-sync.sh` は `PASS: all files in sync.`

### V2 と self-review L1: 持ち越しの注記

注記は Trigger 列の末尾に日付つきで足し(前例の 2026-11-19、2026-11-20 の注記と同じ形)、Related 列に `docs/plans/active/2026-11-21-codex-seat-network.md` を足した。行は内容で探した(merge で行番号がずれる)。`wc -l` は merge-base 09d27450 と、各ファイルを最後に変えた commit で数えた。

| 行(頭の語) | 当たったもの | 行数 | 払わなかった理由 |
|------------|--------------|------|------------------|
| `checkCodexAgmsgWritableRoot` | `doctor_codex_writable_root.go` の `codexSeatModesPossible` の doc に 5 行(S3、`cbb831bc`) | 1009 → 1014 | `pathCovers`、`codexWritableRootSuffix`、読む codex config は変わっていない。新しい検査は `ralph.toml` だけを読み、codex config は読まない |
| `checkShellAliases` | `doctor_shell_alias.go` の comment 12 行(S2、`f9113e91`)と、`permissionArgsForDriver` の `-c`(S1、`574f8a6e`)。この行の Trigger「`[org.permissions]` のフラグを driver に渡す仕方の次の変更」も後者で引いた | 1448 → 1460 | severity の `switch` は変えておらず、`shellAliasFlagClass` は `-c` に class を与えない。config を読む grading は払っていない |
| `internal/cli/doctor_shell_alias.go` is 1448 lines | 同じ comment 12 行 | 1448 → 1460、test は 2577 のまま | comment で、約 335 行の move-only の分割は Scope の外 |
| `The recipes under templates/base/docs/recipes/` | `codex-seat-permissions.md`(2 写し)の新しい節と cross-review の 1 句 | `codex-seat-permissions.md` は 3 件・3 ファイルから 4 件・4 ファイル。`isolated-herdr-server.md` は 14 件・7 ファイルのまま | 新しい引用 1 件(`docs/evidence/codex-seat-network-2026-11-21.md`)は行が求める「ralph repository の記録」の書き方で、古い 3 件は書き換えていない |
| `Five files went past the 800-line guide`(L1) | `doctor.go` に Check 11b-2(comment 7 行と results への 1 行、`f9113e91`)、`config.go` に `CodexNetworkAccess` の field・comment・既定値(`574f8a6e`、`b6c28c1a`) | `doctor.go` 1683 → 1692、`config.go` 1195 → 1219 | 新しい検査は `doctor_codex_network.go` に置いて `doctor.go` は配線だけ、`config.go` は bool の鍵と comment、move-only の分割は Scope の外。残りの 3 ファイルは触っていない |

`config.go` の 1219 は `b6c28c1a`(self-review の L2・L3 の comment の直し)のあとの数で、self-review が書いた 1215 は `574f8a6e` までの数。

### V3: 一部が解消した 2 行(閉じていない)

- 「Whether a codex reviewer can run the gate's scripts」: Trigger「`gate_failed` whose log shows a sandbox denial」を #362 の報告が引いた。直したのは通信の部分(既定で `-c ...=true`、鍵 false で閉じる、doctor が知らせる)。残るのは、座席の sandbox の外にある Go の build と module の cache と、座席の中での実地の確かめ。確かめたのは `codex sandbox` と `codex exec` のエージェントで、herdr の pane の TUI の中は確かめていない(`50a47846` の evidence に合わせて書いた)
- 「The cross-review of a feature org has not run inside a seat」: Trigger の問い(reviewer 座席に通信を与えるか)のうち通信の部分を #362 が決めた。debt の文「may not reach the network」は、鍵が false のとき(と、guarded の座席で人の codex config が閉じているとき)だけの話になった。残るのは、座席の中で cross-review が一度も走っていないこと、TUI の中の確かめ、入れ子の CLI の sandbox と permission mode(issue #360、別の branch)

### V4: recipe の cross-review の節

「`codex_network_access = false` のとき」と、「The network of a codex seat」への指し先を足した。1 文だけで、レーン B(#360)が後でこの節を書き直しても衝突しにくい範囲に収めた。2 写しは `cmp` で同一。

### V5: 計画の Progress と AC

- AC1〜AC7 を `[x]` にした。AC1・AC2・AC3・AC4・AC6 は test report の表で PASS。AC5 と AC7 は文書の AC で、test report では local verifier の中の `check-sync.sh` と `check-skill-sync.sh` が通ったことまで、中身は verify の report が met としている
- Progress: Review、Verification、Test の artifact に `[x]` と report の path、verdict を添えた。`b6c28c1a`(L2〜L5 の直しで挙動の変更なし)と `50a47846`(evidence)を「Implementation started」の子の項に足した。`Sync-docs run 1` の行を足した
- digest は編集の前後とも `2267f124015d`

### I1: `codexNotNeededWhyA` の行

(a) の Trigger に、新しい検査 `checkCodexSeatNetwork` も同じ文を出すこと、`TestDoctorCheckCodexSeatNetwork_DirectorOnlyWorkspaceWriteIsNotCounted`(`doctor_codex_network_test.go`、定数 `wantCodexSeatNetworkAllGuarded`)がその文を固定していること、(a) を払う commit はその期待も変えることを足した。Related に計画のパスを足した。コードは確かめた(`doctor_codex_network.go` の `codexNotNeededWhyA(orgCfg.Permissions.CodexVerified)`)。

### テストの穴 1: `ralph config get` の既定値

足した。表に 1 行で、`internal/cli/config_test.go` は 625 → 626 行。

- `test -z "$(gofmt -l internal/cli)"` と `go vet ./internal/cli/` は通った
- `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 -v ./internal/cli/ -run 'ConfigGet'` は `ok`(21 関数がすべて PASS、2.489 s)
- 期待値を一時的に `false\n` にすると `config get org.permissions.codex_network_access = (stdout "true\n", ...); want ("false\n", "", 0)` で落ちた。戻して再び通った。行が実際に実行されることの確かめ
- test report の穴 1 はこれで塞がる。ただし test report と verify report は書き換えていない。この Go の変更は test report のあとの 1 行で、この 1 行のために `/pr` の前に全部を回し直すかは、パイプラインの cycle の判断になる(Go の挙動は変わらず、テストが 1 行増えただけ)

残りの穴: 2(TUI の中)は V3 の 2 行に未確認として書いた。3(鍵のない machine での codex の既定)は evidence の「確かめていないこと」にある。4(spawn は呼んだ側の `ralph.toml` から `[org.permissions]` を読む)は S3 が足した tech-debt の 1 行にある。

## 既存の行の Trigger の掃引

verify の列の掃引とは別に、`9cd63d60` の `docs/tech-debt/README.md`(branch の編集の前)の開いている行を 5 列に分け、4 つ目の列(Trigger)から取り消し線の部分を除いて、この branch が触れた名前(`permissions.go`、`permissionArgsForDriver`、`config.go`、`OrgPermissionsConfig`、`doctor.go`、`doctor_codex`、`doctor_shell_alias`、`codex-seat-permissions`、`defaults_sync`、`spawn_test.go`、`config_test.go`、`codex_verified`、`[org.permissions]`、`network`、`sandbox`、`--add-dir`、`codexWritableRootArgs`、`ralph.toml`、`config get`、`wc -l`、`800 lines` など 40 語)で探した。当たった 50 行ほどのうち、語の周りを読んだのは verify が判断していない行(118、150、167、181、192、203、267、280、387、430、434、436、485)と 146 で、どれも過去の PR の持ち越しの note の中の語か、別のファイルの Trigger だった。

- 引いた行は verify の 5 行(135、136、138、376、420)と、一部が解消した 2 行(231、433)で、掃引で足した行はない
- 引いていない行の理由は verify の「Tech-debt の Trigger の確かめ」と同じ。146 は `doctor_codex_writable_root_test.go` の次の変更か `runDoctorFull` の対象ディレクトリの決め方の変更が Trigger で、どちらもない(doctor.go の変更は results への 1 行)
- 新しい行 546(spawn が呼んだ側の `ralph.toml` から `[org.permissions]` を読む)は S3 が足した。この run では触れていない

## 通常の確認

| 場所 | 確かめたこと | 結果 |
|------|-------------|------|
| `README.md` の Known differences | `-c ...=true`、鍵 false で `=false`、recipe を指す | 変更なし。`.codex/README.md` の 2 写しをこの句に合わせた(V1) |
| `AGENTS.md`、`CLAUDE.md`、`.claude/rules/`、`docs/quality/` | 座席の引数、`[org.permissions]` の鍵の列挙を探した | 変更なし。verify が確かめたとおり、`--sandbox workspace-write --ask-for-approval never` を書く残りの箇所は今の動きと合う |
| org skill の 4 写し、director skill の 4 写し | verify が確かめた(org は新しい節の記述、director は headless の codex の director が guarded だけ) | 変更なし。skill は変えていないので `sync-skills.sh` は回していない |
| recipe の 2 写し | V4 のあと `cmp` | 同一 |
| `docs/evidence/codex-seat-network-2026-11-21.md` | `50a47846` が `codex exec` のエージェントの確かめを足した | recipe の節は `codex sandbox` の 1 回の確かめと「座席の中は未確認」を書いており、今も誤りではない。エージェントの確かめを recipe に写す直しは頼まれておらず、レーン B が同じ recipe を触るので、していない |

verify の INFO のうち I2(`pass` の `not needed:` の行)、I3(`permissionArgsForDriver` の doc)、I5(「network がないと」の文)は直していない。I4(`docs/tech-debt/README.md` の merge 衝突)は `8fdd98de` で解消済み(2 行とも残っている)。I6 は次の release のノートと PR 本文の項目で、`internal/upgrade/notes/` は `/release` で書くので何もしていない。

## 確かめたこと

- `./scripts/check-sync.sh`: `PASS: all files in sync.`
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`: rc 0(`tech-debt README plan references` は OK、`check-template-purity` は PASS)
- tech-debt の編集した 8 行は 5 列のまま。Related の計画のパスは `docs/plans/active/` で書いた(`/pr` の `archive-plan.sh` が書き換える。先に `archive/` とは書いていない)

## 変えなかったもの

- Go のコードと doc comment。テストは `config_test.go` の 1 行のほかは変えていない
- `README.md`、`AGENTS.md`、`CLAUDE.md`、`.claude/rules/`、`docs/quality/`、spec、skill、`templates/` の skill
- self-review、verify、test の report、evidence(`50a47846` のまま)
- tech-debt の行の閉じ(8 行とも開いたまま。取り消し線を付けた行はない)

## Not verified

- `./scripts/run-test.sh` と `./scripts/run-static-verify.sh` は回していない。後者の代わりに `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` を回した(上の「確かめたこと」)
- 座席の TUI の中で `-c` が効くこと、tflint の初期化が通ること、設定ファイルに鍵がない machine での codex の既定は、誰も確かめていない(V3 の 2 行と evidence に書いた)
- tech-debt の注記の行数は `git show <commit>:<path> | wc -l` で数えた。注記を書いた 8 行のうち、前の記録がある行(`doctor_codex_writable_root.go` の 1009 など)は前の記録と merge-base の数が合った
