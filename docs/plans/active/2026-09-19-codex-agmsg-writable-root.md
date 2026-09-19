# codex-agmsg-writable-root

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-19
- Related request: #155 の実機検証(`docs/evidence/codex-seat-permissions-2026-09-18.md` P3、Run A → A2)で、`--sandbox workspace-write` で動く codex 座席は agmsg の SQLite DB(`~/.agents/skills/agmsg/db/messages.db`)に書けず(`attempt to write a readonly database (8)`)、RESULT を lead に送れなかった。codex config の `[sandbox_workspace_write] writable_roots` に agmsg の `db` ディレクトリを足すと送れる。現状は recipe で手動設定を求めているだけで、設定漏れに気づく手段がない(issue #164)
- Related issue: 164
- Type: feat
- Branch: feat/codex-agmsg-writable-root

## Objective

`ralph doctor` に「Codex sandbox (agmsg writable root)」Check を追加する。codex 座席が `workspace-write` の sandbox で動く構成なのに、ユーザーの codex config(`$CODEX_HOME/config.toml`、既定 `~/.codex/config.toml`)の `[sandbox_workspace_write].writable_roots` が agmsg の `db` ディレクトリを覆っていなければ warn し、`docs/recipes/codex-seat-permissions.md` の該当節へ案内する。ralph が座席の sandbox を自動で広げることはしない。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | Check 本体 | `internal/cli/doctor_codex_writable_root.go`(新規) | `checkCodexAgmsgWritableRoot(...) checkResult`。入力: `codex_verified`(`cfg.Org.Permissions.CodexVerified`)、解決済みの agmsg home(`driver.ResolveAgmsgHome(cfg.Org.AgmsgHome)`)、agmsg Check の結果が pass か、codex config のパスを返す resolver。config は `github.com/pelletier/go-toml/v2`(既存依存)で読む。プロセスは起動しない |
| 2 | 判定 | 同上 | 「writable root が必要な理由」を集める: (a) `codex_verified = true`(edits / autonomous の codex 座席に ralph が `--sandbox workspace-write` を付ける)、(b) ユーザー config の `sandbox_mode = "workspace-write"`(ralph がフラグを付けない guarded の codex 座席がそのまま継承する)。理由がなければ `pass`(不要である旨と根拠)。理由があれば `writable_roots` の各要素を `filepath.Clean` し、agmsg の `db` ディレクトリと同一、またはその祖先であれば「覆っている」とみなす(両者を `filepath.EvalSymlinks` で解決した形でも比較。解決できなければ解決前の形で比較)。覆っていれば `pass`(該当 root を名指し)、覆っていなければ `warn` |
| 3 | 重大度の例外 | 同上 | agmsg が未導入(agmsg Check が pass でない)なら org 座席自体が使えないので `info`。config が存在しない場合は `writable_roots` なしとして扱う(理由があれば warn)。config が TOML として読めない、または `writable_roots` が文字列配列でない場合は `info`(パスと理由だけを出し、config の内容は出さない)。codex home を解決できない場合は `info`。トップレベルの `profile` が設定されている場合は、Detail に「profile は評価していない」と添える。warn / info は exit code に影響しない |
| 4 | 登録と seam | `internal/cli/doctor.go`、`internal/cli/main_test.go` | Check 11(codex スラッグ)の直後に Check 11b として登録。config パスの resolver は package 変数 `doctorCodexConfigPath`(既定は `$CODEX_HOME` を `codexModelsCachePath` と同じ規則でリテラルに使う)にし、`TestMain` で存在しないパスに固定して、既存の `runDoctor*` テストが開発者の実 config を読まないようにする(#162 の `doctorShellAliasEnv` と同じ形) |
| 5 | テスト | `internal/cli/doctor_codex_writable_root_test.go`(新規) | temp dir の config fixture で: 不要(理由なし)→ pass、`codex_verified = true` で root なし → warn(`db` ディレクトリ、config パス、recipe パスを含む)、root が `db` と同一 → pass、root が祖先(agmsg home)→ pass、接頭辞が同じだけの別ディレクトリ(`…/db` に対する `…/dbx`、`/a/bc` に対する `/a/b`)→ warn、symlink 経由で同一 → pass、`codex_verified = false` でも `sandbox_mode = "workspace-write"` なら warn(guarded 座席に言及)、config なし + `codex_verified = true` → warn、壊れた TOML → info(内容を出さない)、`writable_roots` が配列でない → info、agmsg 未導入 → info、`CODEX_HOME` がリテラルに使われる、`profile` 設定時の注記、`runDoctorOpts` の出力に Check 行が出る統合テストと `TestMain` 既定での pass。fixture に `api_key=` のような secret 風の文字列を書かない |
| 6 | 文書 | `docs/recipes/codex-seat-permissions.md` + template、`.claude/skills/org/SKILL.md` + 3 ミラー | recipe の「Add the agmsg database to the sandbox's writable roots」節と、skill の permission 作法の codex 行に「`ralph doctor` の Check が設定漏れを warn する」を一文追加 |

## Non-goals

- ralph が codex 座席の引数に `-c sandbox_workspace_write.writable_roots=…` を自動付与すること(ユーザー判断で不採用。`-c` がユーザー設定の配列を置き換えるか併合するかが未確認で、置き換えなら他の writable root を黙って無効にする。座席の sandbox を ralph が黙って広げない方針とも合わない)
- プロジェクト階層の `.codex/config.toml`、`profiles.<name>`、`-c` の上書きなど、codex の設定レイヤーの完全な再現。読むのはユーザー階層の config 1 ファイルだけで、Detail に読んだパスを出す
- `sandbox_mode = "read-only"` や未設定時の codex 既定の評価(agmsg に限らず何も書けない構成で、この Check の対象外)
- codex 座席を実際に起動しての再検証。「root を足せば RESULT が届く」ことは #155 の Run A2 で実証済み。本 Check は設定漏れの検出だけを行う

## Assumptions

- agmsg の DB は `<agmsg home>/db/` の下にある(#155 evidence P3、`~/.agents/skills/agmsg/db/messages.db`)。SQLite は同じディレクトリに journal / WAL を作るので、必要なのはファイルではなくディレクトリへの書き込み権
- codex は `writable_roots` の要素を絶対パスとして扱う。`~` や相対パスの要素を codex が展開するかは未確認なので、Check は展開せず、一致しないものとして扱う(見逃しではなく warn 側に倒れる)
- codex home の解決は `codexModelsCachePath` と同じ規則(`CODEX_HOME` が空でなければリテラルに使用、なければ `~/.codex`)
- guarded の codex 座席がユーザー config の `sandbox_mode` を継承することは、ralph が guarded にフラグを付けない(`permissionArgsForDriver` が nil を返す)ことからの推論で、座席での実測はしていない。Detail はこの条件を断定ではなく理由として書く

## Affected areas

- `internal/cli/doctor_codex_writable_root.go`、`internal/cli/doctor_codex_writable_root_test.go`(新規)
- `internal/cli/doctor.go`(登録)、`internal/cli/main_test.go`(seam の固定を 1 つ追加)
- `docs/recipes/codex-seat-permissions.md` + template、`.claude/skills/org/SKILL.md` + 3 ミラー

## Design decisions

- **方式(critical fork、ユーザー決定 2026-09-19)**: 「doctor で検査」を採用。自動付与と両方は不採用(Non-goals 参照)。理由: 座席の sandbox を ralph が黙って広げない。#162 と同じ「検出 + recipe 案内」の形に揃う。静的な TOML 読み取りなので fixture でテストでき、実機の座席起動が要らない

既定として採った選択:

- warn の条件に `sandbox_mode = "workspace-write"`(guarded 座席)を含める。同じ 1 ファイルの読み取りで分かり、失敗の形(RESULT が届かない)も同じため
- agmsg 未導入なら info に落とす(#162 の herdr 未導入と同じ考え方)
- config の内容は Detail に出さない。出すのは読んだパス、agmsg の `db` ディレクトリ、覆っている root だけ
- Check 名は「Codex sandbox (agmsg writable root)」

## Acceptance criteria

- [ ] AC-1: `ralph doctor` の出力に「Codex sandbox (agmsg writable root)」Check が codex スラッグの Check の直後に出る
- [ ] AC-2: Scope 5 のテストがすべて pass。warn の Detail が agmsg の `db` ディレクトリ、読んだ config のパス、`docs/recipes/codex-seat-permissions.md` を含む。祖先判定がパス要素単位で、接頭辞が同じだけのディレクトリを覆っているとみなさない
- [ ] AC-3: 理由なし → `pass`、覆っている → `pass`、覆っていない → `warn`、agmsg 未導入 / 読めない config / 型違い / codex home 解決不能 → `info`。`countFailed` の対象にならない(既存の exit code テストが pass のまま)
- [ ] AC-4: 既存の `runDoctor*` テストが開発者の実 `~/.codex/config.toml` を読まない(`TestMain` の固定と、それを確かめるテスト)
- [ ] AC-5: recipe(root / template 一致)と `/org` skill(4 面一致)に Check への言及がある。`check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` pass
- [ ] AC-6: `gofmt` / `go vet` / golangci-lint clean、`go test ./internal/cli/... -count=1` green、`./scripts/run-verify.sh` green、`./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/main)..HEAD"` が exit 0
- [ ] AC-7: evidence として、このマシンの実 config での `ralph doctor` の該当行と、スクラッチの `CODEX_HOME` + `codex_verified = true` の `ralph.toml` で root なし(warn)/ root あり(pass)の該当行を plan に記録する。issue の受け入れ条件「実機で確認」は、採用方式が検出のみであるため、この doctor 出力と #155 の Run A / A2(root なしで失敗、ありで成功)への参照で満たす
- [ ] AC-8: PR 本文は `Closes #164`

## Implementation outline

1. Slice A(Go、implementer 委譲): Scope 1〜5 → `gofmt -l internal/`、`go vet ./internal/cli/...`、対象テスト、`go test ./internal/cli/... -count=1`、`./scripts/run-verify.sh`
2. Slice B(docs、inline): Scope 6 → `sync-skills.sh` + cp → 同期ゲート 3 本
3. evidence(AC-7)を plan の Deviation notes に記録 → post-implementation pipeline → `/cross-review` → push 前に range secret scan → `/pr`(`Closes #164`)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、同期ゲート 3 本、range secret scan
- Spec compliance criteria to confirm: AC-1〜AC-8。Detail の文言が doc comment・recipe・skill と一致すること
- Documentation drift to check: recipe の該当節、skill の permission 作法、`docs/evidence/codex-seat-permissions-2026-09-18.md` P3 と矛盾しないこと。README / AGENTS.md に doctor の Check 一覧がないこと(#162 で確認済み)
- Evidence to capture: grep / cmp の出力、AC-7 の doctor 出力

## Test plan

- Unit tests: Scope 5 の各ケース
- Integration tests: `runDoctorOpts` 経由で Check 行が出ること、`TestMain` 既定での結果、exit code 不変
- Regression tests: `go test ./internal/... -count=1`
- Edge cases: (1) `/a/b` と `/a/bc` の接頭辞一致、(2) 末尾スラッシュ付きの root、(3) symlink(macOS の `/tmp` → `/private/tmp`)、(4) 空の `writable_roots`、(5) `CODEX_HOME` が空白を含む / 存在しない、(6) config がディレクトリ、(7) `~` で始まる root は一致しない
- Evidence to capture: `go test -count=1 -v` の出力、`docs/reports/test-*.md`

## Risks and mitigations

- codex の設定レイヤー(project 階層、profile、`-c`)を再現しないため、実効値とずれる可能性 → Detail に読んだパスを出し、`profile` 設定時は注記する。Non-goals に明記
- guarded 座席の `sandbox_mode` 継承は推論 → Assumptions に明記し、文言を断定にしない
- 既存テストが実 config を読む → `TestMain` の seam で固定(AC-4)
- fixture の文字列が CI の secret scan(履歴を読む)に掛かる → fixture に secret 風の文字列を書かない。push 前に range scan を実行(AC-6)

## Rollout or rollback notes

doctor の Check 追加と文書のみ。下流へは次回 release でバイナリ経由、skill / recipe の文言は `ralph upgrade` / `ralph init` で配布。revert は PR 単位で安全。

## Open questions

なし。

## Deviation notes

- 2026-09-19 plan: 方式選択の前に、`-c sandbox_workspace_write.writable_roots=…` がユーザー設定の配列を置き換えるか併合するかを、API を呼ばない `codex sandbox` サブコマンド(スクラッチの `CODEX_HOME`)で確かめようとしたが、同サブコマンドは config の `sandbox_mode` を見ず既定が読み取り専用で、permission profile の指定方法も合わなかったため確認できなかった。ユーザーには未確認である旨を示したうえで方式を選んでもらい、「doctor で検査」に決定

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
