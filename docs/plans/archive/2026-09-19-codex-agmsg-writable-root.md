# codex-agmsg-writable-root

- Status: Done (PR #171)
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
| 1 | Check 本体 | `internal/cli/doctor_codex_writable_root.go`(新規) | `checkCodexAgmsgWritableRoot(orgCfg config.OrgConfig, agmsgHome string, resolveEnv …) checkResult`。入力: `cfg.Org`(`driver_pool`、`[org.permissions]` の `default` / `roles` / `codex_verified` を使う)、解決済みの agmsg home(`driver.ResolveAgmsgHome(cfg.Org.AgmsgHome)`)、agmsg が導入済みか(`driver.AgmsgAvailable(home) == nil`。agmsg Check の Status は使わない。バージョン違いの導入済み環境は info を返すため)、環境の resolver(codex config のパスと `AGMSG_STORAGE_PATH` の値を返す)。config は `github.com/pelletier/go-toml/v2`(既存依存)で読む。プロセスは起動しない |
| 2 | 判定 | 同上 | まず `[org].driver_pool` に codex がなければ codex 座席は spawn できないので `pass`(不要)。次に「writable root が必要な理由」を集める: (a) `codex_verified = true` かつ edits / autonomous に解決される role があり得る(`[org.permissions].default` の実効値か `roles` の値のどれかが edits / autonomous。ralph がその codex 座席に `--sandbox workspace-write` を付ける)、(b) ユーザー config の `sandbox_mode = "workspace-write"` かつ guarded に解決される role があり得る(ralph がフラグを付けない guarded の codex 座席がそのまま継承する)。permission mode は role の設定だけで決まり、spawn 時の上書きフラグはない。理由がなければ `pass`(不要である旨と、どの条件が外れているか)。理由があれば、まず agmsg の保存ディレクトリを agmsg 自身と同じ優先順で決める: `AGMSG_STORAGE_PATH` が空でなければその値(末尾のスラッシュを 1 つ落とす)、なければ `<agmsg home>/db`。次に `writable_roots` の各要素を `filepath.Clean` し、その保存ディレクトリと同一、またはその祖先であれば「覆っている」とみなす(両者を `filepath.EvalSymlinks` で解決した形でも比較。解決できなければ解決前の形で比較)。覆っていれば `pass`(該当 root を名指し)、覆っていなければ `warn`。**cross-review cycle 1 による改訂**: (1) root から保存ディレクトリまでのパス要素に `.git` / `.agents` / `.codex` があれば覆っているとみなさない(codex は writable root の配下のこれらを再帰的に読み取り専用で保護する。公式ドキュメントで確認)。該当する祖先 root があれば warn の Detail で名指しし、保存ディレクトリそのものを足すよう案内する。(2) workspace-write が既定で書ける一時ディレクトリ(`/tmp`、`$TMPDIR`)も暗黙の root として扱う。`[sandbox_workspace_write]` の `exclude_slash_tmp` / `exclude_tmpdir_env_var` が true なら除く。(3) 理由の判定は `[org].model_pool` と `[org.roles]` も見る: pool に codex の model がなければ codex 座席は spawn できないので不要。`[org.permissions.roles]` の各 role の mode は、その role に許可された codex の model があるときだけ数える |
| 3 | 重大度の例外 | 同上 | agmsg が未導入(`driver.AgmsgAvailable` がエラー)なら org 座席自体が使えないので `info`。導入済みでバージョンだけ違う場合は通常どおり判定する。`AGMSG_STORAGE_PATH` を使った場合は Detail にその旨と、herdr の pane の環境変数が doctor のものと違い得ることを添える。config が存在しない場合は `writable_roots` なしとして扱う(理由があれば warn)。config が TOML として読めない、または `writable_roots` が文字列配列でない場合は `info`(パスと理由だけを出し、config の内容は出さない)。codex home を解決できない場合は `info`。トップレベルの `profile` が設定されている場合は、Detail に「profile は評価していない」と添える。warn / info は exit code に影響しない |
| 4 | 登録と seam | `internal/cli/doctor.go`、`internal/cli/main_test.go` | Check 11(codex スラッグ)の直後に Check 11b として登録。環境の resolver は package 変数 `doctorCodexSandboxEnv`(既定は `$CODEX_HOME` を `codexModelsCachePath` と同じ規則でリテラルに使った config パスと、`AGMSG_STORAGE_PATH`)にし、`TestMain` で存在しない config パス・空の override に固定して、既存の `runDoctor*` テストが開発者の実 config を読まないようにする(#162 の `doctorShellAliasEnv` と同じ形) |
| 5 | テスト | `internal/cli/doctor_codex_writable_root_test.go`(新規) | temp dir の config fixture で: 不要(理由なし)→ pass、`codex_verified = true` で root なし → warn(`db` ディレクトリ、config パス、recipe パスを含む)、root が `db` と同一 → pass、root が祖先(agmsg home)→ pass、接頭辞が同じだけの別ディレクトリ(`…/db` に対する `…/dbx`、`/a/bc` に対する `/a/b`)→ warn、symlink 経由で同一 → pass、`codex_verified = false` でも `sandbox_mode = "workspace-write"` なら warn(guarded 座席に言及)、config なし + `codex_verified = true` → warn、壊れた TOML → info(内容を出さない)、`writable_roots` が配列でない → info、agmsg 未導入 → info、導入済みでバージョン違い(agmsg Check が info)でも root なしなら warn、`AGMSG_STORAGE_PATH` 設定時に既定の `db` だけを覆う root → warn / override 先を覆う root → pass、`CODEX_HOME` がリテラルに使われる、`profile` 設定時の注記、`runDoctorOpts` の出力に Check 行が出る統合テストと `TestMain` 既定での pass。fixture に `api_key=` のような secret 風の文字列を書かない |
| 6 | 文書 | `docs/recipes/codex-seat-permissions.md` + template、`.claude/skills/org/SKILL.md` + 3 ミラー | recipe の「Add the agmsg database to the sandbox's writable roots」節と、skill の permission 作法の codex 行に「`ralph doctor` の Check が設定漏れを warn する」を一文追加 |

## Non-goals

- ralph が codex 座席の引数に `-c sandbox_workspace_write.writable_roots=…` を自動付与すること(ユーザー判断で不採用。`-c` がユーザー設定の配列を置き換えるか併合するかが未確認で、置き換えなら他の writable root を黙って無効にする。座席の sandbox を ralph が黙って広げない方針とも合わない)
- プロジェクト階層の `.codex/config.toml`、`profiles.<name>`、`-c` の上書きなど、codex の設定レイヤーの完全な再現。読むのはユーザー階層の config 1 ファイルだけで、Detail に読んだパスを出す
- `sandbox_mode = "read-only"` や未設定時の codex 既定の評価(agmsg に限らず何も書けない構成で、この Check の対象外)
- codex 座席を実際に起動しての再検証。「root を足せば RESULT が届く」ことは #155 の Run A2 で実証済み。本 Check は設定漏れの検出だけを行う

## Assumptions

- agmsg の DB は既定で `<agmsg home>/db/` の下にある(#155 evidence P3、`~/.agents/skills/agmsg/db/messages.db`)。agmsg 1.1.13 の `scripts/lib/storage.sh` は `AGMSG_STORAGE_PATH`(messages.db を置くディレクトリ)を最優先で使うので、Check も同じ優先順に従う。座席が見る環境変数は herdr の pane のもので、doctor のプロセスと一致するとは限らない。SQLite は同じディレクトリに journal / WAL を作るので、必要なのはファイルではなくディレクトリへの書き込み権
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

- [x] AC-1: `ralph doctor` の出力に「Codex sandbox (agmsg writable root)」Check が codex スラッグの Check の直後に出る
- [x] AC-2: Scope 5 のテストがすべて pass。warn の Detail が agmsg の `db` ディレクトリ、読んだ config のパス、`docs/recipes/codex-seat-permissions.md` を含む。祖先判定がパス要素単位で、接頭辞が同じだけのディレクトリを覆っているとみなさない
- [x] AC-3: `driver_pool` に codex なし → `pass`、`model_pool` に codex の model なし → `pass`、理由なし(`codex_verified` が false / edits・autonomous に解決される role がない / guarded に解決される role がない / `sandbox_mode` が workspace-write でない)→ `pass`、覆っている → `pass`、保存ディレクトリが既定で書ける一時 root(固定の一時ディレクトリ / `$TMPDIR`。`exclude_slash_tmp` / `exclude_tmpdir_env_var` で除外されていないもの)の下 → `pass`、覆っていない → `warn`(agmsg が導入済みならバージョン違いでも warn)、祖先の root が `.git` / `.agents` / `.codex` をまたぐだけ → `warn`(その root を名指し)、`AGMSG_STORAGE_PATH` があればその値で判定、agmsg 未導入 / 読めない config / 型違い / codex home 解決不能 → `info`。`countFailed` の対象にならない(既存の exit code テストが pass のまま)
- [x] AC-4: 既存の `runDoctor*` テストが開発者の実 `~/.codex/config.toml` を読まない(`TestMain` の固定と、それを確かめるテスト)
- [x] AC-5: recipe(root / template 一致)と `/org` skill(4 面一致)に Check への言及がある。`check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` pass
- [x] AC-6: `gofmt` / `go vet` / golangci-lint clean、`go test ./internal/cli/... -count=1` green、`./scripts/run-verify.sh` green、`./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/main)..HEAD"` が exit 0
- [x] AC-7: evidence として、このマシンの実 config での `ralph doctor` の該当行と、スクラッチの `CODEX_HOME` + `codex_verified = true` の `ralph.toml` で root なし(warn)/ root あり(pass)の該当行を plan に記録する。issue の受け入れ条件「実機で確認」は、採用方式が検出のみであるため、この doctor 出力と #155 の Run A / A2(root なしで失敗、ありで成功)への参照で満たす
- [x] AC-8: PR 本文は `Closes #164`

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
- 2026-09-19 plan: Codex plan advisory(codex-cli 0.154.0、`-m gpt-6-astra -c model_reasoning_effort=xhigh`、stdin を閉じて実行)が MEDIUM 2 件を報告。どちらも事実を確認して採用した。(1) agmsg 1.1.13 は `AGMSG_STORAGE_PATH` を最優先で使うので、`<agmsg home>/db` 固定では別の場所を検査して pass と言い得る → agmsg と同じ優先順で保存ディレクトリを決める。(2) 既存の agmsg Check はバージョン違いの導入済み環境に info を返すので、「pass でなければ未導入」とすると agmsg の更新で warn が消える → 導入判定は `driver.AgmsgAvailable` で行う
- 2026-09-19 work: Slice A は implementer に委譲(e2b558a、4 ファイル、逸脱なし)。orchestrator が HEAD 一致・porcelain 空・差分を確認。root 側の symlink 解決が「存在しなければ諦める」、store 側が「最も近い既存の祖先まで遡る」と非対称だったので、両側を `resolveNearestExisting` に揃え、未作成の store を symlink 経由の root が覆うケースのテストを追加した(445cea7)。対象テストと `./scripts/run-verify.sh` は green、implementer が実行した履歴込みの range secret scan は exit 0。Slice B(recipe 2 コピー + skill 4 面)は f14737f、同期ゲート 3 本 pass
- 2026-09-19 work: AC-7 の evidence(ビルドしたバイナリ、agmsg は実機の `~/.agents/skills/agmsg`)。(1) このリポジトリ + 実 config: `✓ Codex sandbox (agmsg writable root): pass — not needed: [org.permissions].codex_verified is false and ~/.codex/config.toml does not set sandbox_mode = "workspace-write"`。(2) スクラッチのプロジェクト(`ralph.toml` に `codex_verified = true`)+ root のないスクラッチ `CODEX_HOME`: `⚠ … warn — no writable root in <scratch>/ch-none/config.toml covers the agmsg store <home>/.agents/skills/agmsg/db, so a codex seat under workspace-write cannot send RESULT to lead ("attempt to write a readonly database"); needed because [org.permissions].codex_verified = true (ralph passes --sandbox workspace-write to edits and autonomous codex seats); add <home>/.agents/skills/agmsg/db to [sandbox_workspace_write].writable_roots (docs/recipes/codex-seat-permissions.md)`。(3) 同じプロジェクト + root を入れたスクラッチ `CODEX_HOME`: `✓ … pass — writable root <home>/.agents/skills/agmsg/db in <scratch>/ch-root/config.toml covers the agmsg store …; needed because …`。(2) と (3) の doctor の exit code はどちらも 0。root なしで RESULT が届かず、root ありで届くこと自体は #155 の Run A / A2 が実証済み
- 2026-09-19 self-review cycle 1(`docs/reports/self-review-2026-09-19-codex-agmsg-writable-root.md`、03d9e93): MEDIUM 3 / LOW 6、CRITICAL / HIGH なし。全件を同 cycle 内で修正する。M1 `*toml.DecodeError` でない解析エラー(重複キーなど)の生メッセージが Detail に出て config のキー名が漏れる → 解析失敗を専用のエラー型に包み、メッセージ本文は出さない。M2 `~` 表示が実 `os.UserHomeDir()` を読むため `TMPDIR` が `$HOME` 配下だとテスト 6 件が落ちる → env に `Home` を足して注入する。M3 symlink を解決しない 1 回目の比較が、store が root の外への symlink でも pass にする → 解決する比較だけにする。L4 config が存在しないのに「does not set sandbox_mode」と言う → 存在しない旨を言う。L5 型違いの有効な TOML を「not valid TOML」と言う → 「decode できない」に統一。L6 理由が `codex_verified` だけで立ち、role の設定と `driver_pool` を見ない → mode は role 設定だけで決まる(spawn 時の上書きなし)ので、tech-debt に回さずコードで直す(Scope 2 / AC-3 を改訂)。L7 Windows では `syscall.Mkfifo` がコンパイルできず skip に到達しない → FIFO のテストを `!windows` のビルドタグ付きファイルへ。L8 名前が主張する経路を通っていないテスト → M3 の修正で解決する比較だけになるので名前と内容を合わせる。L9 `AGMSG_STORAGE_PATH` の根拠を evidence に帰している誤り → agmsg の `scripts/lib/storage.sh` に直す
- 2026-09-19 work: Slice C(self-review の 9 件)は implementer に委譲(0811070、5 ファイル)。逸脱は 1 件だけで、テスト用 helper の引数を増やさず、`Home` が要る 1 テストだけ inline の resolver を使った。orchestrator が HEAD 一致・porcelain 空・差分を確認し、`codexNotNeededWhyA` の未使用引数を 65a6f86 で除去。`TMPDIR` を `$HOME` 配下にした再現コマンドが pass、`GOOS=windows go vet` の出力に本 Check のファイルなし、`./scripts/run-verify.sh` green、履歴込みの range secret scan は exit 0。ビルドしたバイナリで確認: 重複キーの config → info(キー名を出さない)、`driver_pool` に codex なし → pass(不要)、default guarded + `sandbox_mode = "workspace-write"` → 理由 (b) だけで warn、config なし + `codex_verified = true` → warn(`does not exist` を表示)。文書(recipe 2 コピー + skill 4 面)の一文を role / `driver_pool` の条件に合わせて更新(5bacc46)
- 2026-09-19 work: AC-7 の evidence の更新。このリポジトリ + 実 config の該当行は `✓ Codex sandbox (agmsg writable root): pass — not needed: [org.permissions].codex_verified is false and no role resolves to guarded`(既定の permission mode が autonomous で guarded の role がないため、理由 (b) の条件が外れる)。スクラッチ構成の warn / pass は理由の文が `… codex_verified = true and a role resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats)` に変わった以外は前回の記録と同じ
- 2026-09-19 self-review 再確認(同 report の Revalidation 節、3e43a3c): 当初 9 件はすべて解消、verdict は merge。L6 をコードで直したため、reviewer が提案していた tech-debt 行は取り下げ。修正で生じた新所見 6 件(すべて LOW)も同 cycle 内で直す。NEW-1 理由が 2 つ立つと各理由の中の「and」と連結の「and」が混ざって 4 節の羅列になる → 2 つのときは番号を振る。NEW-2 実効 default を `ResolvePermissionMode(orgCfg, "")` で求めているため、`roles` に空文字キーがあると default が無視され、warn すべき構成が pass になる → `Roles` を外したコピーで解決する。NEW-3 config がない warn で「does not exist」の括弧が store に掛かって読める → config を主語にした文にする。NEW-4 Scope 1 の入力の記述が古い → 本コミットで修正。NEW-5 `config.Load` が失敗しても既定値の `cfg.Org` で判定する点を doc comment に明記。NEW-6 コメントの不正確な 2 箇所
- 2026-09-19 work: Slice D(再確認の NEW-1〜3・5・6)は implementer に委譲(33ac5a8、2 ファイル、逸脱なし)。NEW-5 は `runDoctorFull` が `config.Load` の失敗後も同じ `cfg` で各 Check を続けることを implementer が `doctor.go` で確認し、doc comment に明記した。再々確認は行わない(1 cycle 1 回)ため、orchestrator が非コメントの差分(実効 default の解決、理由の番号付け、config がない warn の文)を通読し、対象テストと履歴込みの range secret scan(exit 0)を再実行した。implementer の `./scripts/run-verify.sh` は green
- 2026-09-19 verify(`docs/reports/verify-2026-09-19-codex-agmsg-writable-root.md`、8fefb77): PASS、drift なし。AC-1〜AC-7 を満たし、未レビューだった 33ac5a8 も所見と突き合わせて確認済み。verifier が HEAD からビルドしたバイナリで AC-7 の 3 行を再現し、plan の記録と一致
- 2026-09-19 test(`docs/reports/test-2026-09-19-codex-agmsg-writable-root.md`、4656651): PASS。対象 94 件 PASS / SKIP 0、8 package ok、race なし、`internal/cli` 83.5%、バイナリの 13 構成と exit code 不変の確認が期待どおり、`CODEX_HOME` を差し替えても `TestRunDoctorOpts*` の結果は同一。tester が挙げた未到達の分岐 2 つ(`codexConfigDecodeError.Error`、`codexConfigReadReason` の `*fs.PathError` 経路)は、報告後に orchestrator がテスト 2 件を追加して埋めた(テストのみの変更。static verify と対象テストは green)
- 2026-09-19 sync-docs(`docs/reports/sync-docs-2026-09-19-codex-agmsg-writable-root.md`、04b6df5 / b6e85d0): tech-debt に 1 行追加(ユーザー階層の config だけを読むこと、`~` / 相対の root を覆っていないものとして扱うこと、guarded 座席の `sandbox_mode` 継承が推論であること)、evidence P3 に追記 1 行。README / AGENTS.md / spec / 他の recipe は変更不要。同期ゲート 3 本 pass
- 2026-09-19 sync-docs の副作用: 履歴込みの range secret scan が、test / verify レポートの衛生確認の文(バッククォート付きのキー名 + 「-shaped」)に反応して失敗した。doc-maintainer が `.gitallowed` に 2 行足して通した(90dc4be、ee70868)が、その 2 行は「バッククォート付きのキー名を含む行すべて」を免除する広い規則だったので、orchestrator が「キー名が `=` で終わり `/` で連なって `-shaped` が続く」言い回しだけに一致する 1 行に絞った(e2c64c3)。確認: 履歴込みの scan は commit の前後とも exit 0、バッククォート付きのキー名と本物らしい値が同じ行にある場合とバッククォート内に値がある場合は引き続き BLOCK、scanner のテスト 6 件 pass。レポート本文は commit 済みで scan が履歴を読むため、文言の書き換えでは解消できない(#169 の対象)
- 2026-09-19 cross-review cycle 1(`docs/reports/cross-review-triage-codex-agmsg-writable-root.md`、3530a72): Codex の所見 3 件(すべて P2)を ACTION_REQUIRED 1(AR-1 祖先 root が `.agents` などの保護ディレクトリをまたぐのに pass と言う)/ WORTH_CONSIDERING 2(WC-1 既定で書ける一時ディレクトリを見ず不要な warn、WC-2 codex の model を使えない role の mode まで数えて不要な warn)に分類。AR-1 の前提は codex の公式ドキュメントで確認した。ユーザー判断(AskUserQuestion): 3 件とも修正して全 pipeline を再実行(cycle 2/2)
- 2026-09-19 work(cycle 2): Slice E(cross-review の AR-1 / WC-1 / WC-2)は implementer に委譲(ab09dbc、2 ファイル)。AR-1 は修正を外すと対象テストが FAIL し、戻すと PASS することを implementer が確認(red / green)。implementer が `pathCrossesCodexProtectedDir` 単体の誤り(root の外へ出る相対パスでも true)も見つけて修正。さらに orchestrator の WC-1 の設計の穴を報告: `/tmp` を Check 内に直書きすると、実 `TMPDIR` が `/tmp` の環境(CI の ubuntu)では `t.TempDir()` の fixture がすべて覆われ、warn を期待するテスト 16 件が落ちる(`TMPDIR=/tmp` で再現)。Slice E2(91a5542)で `/tmp` も環境の seam(`codexSandboxEnv.SlashTmpDir`)から注入する形に変更。本番の resolver だけが `/tmp` を設定し、テストの既定は空。`TMPDIR=/tmp go test ./internal/cli/... -count=1` は ok
- 2026-09-19 work(cycle 2): orchestrator の確認。`TMPDIR=/tmp` でのパッケージ全体のテストと対象テストを再実行して ok、履歴込みの range secret scan は exit 0、`./scripts/run-verify.sh` green。ビルドしたバイナリで確認: ホームを root にした config → warn(`<home> contains it, but codex keeps .git, .agents, and .codex directories under a writable root read-only`)、保存先が `/tmp` の下で root なし → pass(`exclude_slash_tmp is not set`)、同じ構成で `exclude_slash_tmp = true` → warn、autonomous の role が claude の model に限定されている構成 → pass(不要)。WC-2 に合わせて理由と「不要」の文を「codex の model を使える role」に限定した表現に直し(c03c21e)、recipe 2 コピーと skill 4 面の一文も更新(4c891ec)。同期ゲート 3 本 pass
- 2026-09-19 self-review cycle 2(同 report の Cycle 2 節、4c48ae3): verdict は merge、MEDIUM 1 / LOW 7。cross-review の 3 件は判定そのものが直っていると確認され、role / model の判定は 7 構成を `ValidateSpawnEnvelope` と突き合わせて全件一致。所見は全件この cycle で対応: C2-1(MEDIUM、誤 pass)保護ディレクトリの判定が symlink 解決後の綴りだけ → 設定どおりの綴りでも判定。C2-2 一致した暗黙 root と別の exclude キーを名指しし得る → root と一緒にキーを返す。C2-3 `.gitallowed` の免除が行単位である旨をコメントに明記、C2-4 tech-debt の「未検証の制限」を 3 つに修正(どちらも 235abb0)。C2-5 config がないときのコメントと挙動の不一致 → 暗黙 root が保護ディレクトリで block された場合は warn に出す。C2-6 常に真になる assertion 3 件を修正。C2-7 重複したコメント文。C2-8(関数長 86 行)は reviewer の判断どおり再構成しない
- 2026-09-19 work(cycle 2): Slice F は implementer に委譲(735fc45、2 ファイル)。C2-1 は red / green を確認。比較を `codexRootCoverage` に共通化し、暗黙 root は一致した構造体をそのまま返す形にした(文字列での照合をなくす)。追加の再確認ラウンドがないため orchestrator が通読し、バイナリで `.agents` が symlink の構成を試したところ、スクラッチが固定の一時 root の下にあるために pass になる経路が残っていた: root が symlink 経由の綴り(darwin の固定一時 root)、保存先が解決後の接頭辞 + 深い位置の `.agents` symlink だと、同じ綴り同士のどちらの組でも保護要素が見えない。root と保存先の綴りの 4 通りの組み合わせすべてで判定するよう修正し、専用テストを追加(5fcf070、修正を外すと FAIL、戻すと PASS)。`TMPDIR=/tmp go test ./internal/cli/... -count=1` ok、`./scripts/run-verify.sh` green、履歴込みの range secret scan は exit 0
- 2026-09-19 verify cycle 2(同 report の Cycle 2 節、d2495e4): PASS、blocking なし。未レビューだった 735fc45 / 5fcf070 を C2-1 / C2-2 と突き合わせて確認済み。static verify と履歴込みの range secret scan(21 commit)は exit 0。非ブロッキングの指摘 3 件を反映: AC-3 に新しい結果(`model_pool` に codex の model なし、暗黙の一時 root、保護ディレクトリをまたぐ祖先 root)を明記、AC-7 の evidence を現行の文言で取り直し(下)、skill の一文に一時ディレクトリの扱いを追記
- 2026-09-19 work: AC-7 の evidence(最終、HEAD からビルドしたバイナリ)。(1) このリポジトリ + 実 config: `✓ Codex sandbox (agmsg writable root): pass — not needed: [org.permissions].codex_verified is false and no role that can use a codex model resolves to guarded`。(2) スクラッチのプロジェクト(`codex_verified = true`)+ root のないスクラッチ `CODEX_HOME`: `⚠ … warn — no writable root in <scratch>/ev/ch-none/config.toml covers the agmsg store <home>/.agents/skills/agmsg/db, so a codex seat under workspace-write cannot send RESULT to lead ("attempt to write a readonly database"); needed because [org.permissions].codex_verified = true and a role that can use a codex model resolves to edits or autonomous (ralph passes --sandbox workspace-write to those codex seats); add <home>/.agents/skills/agmsg/db to [sandbox_workspace_write].writable_roots (docs/recipes/codex-seat-permissions.md)`。(3) 同じプロジェクト + root を入れたスクラッチ `CODEX_HOME`: `✓ … pass — writable root <home>/.agents/skills/agmsg/db in <scratch>/ev/ch-root/config.toml covers the agmsg store <home>/.agents/skills/agmsg/db; needed because …(同じ理由の文)`。(2) と (3) の doctor の exit code はどちらも 0。これより前の Deviation notes にある evidence の引用は、理由の文言が変わる前のもの
- 2026-09-19 test cycle 2(同 report の Cycle 2 節、5e939d1): PASS、所見なし。対象 202 件 PASS / SKIP 0、8 package ok、CI を模擬する `TMPDIR=/tmp go test ./internal/cli/... -count=1` ok、race なし、`internal/cli` 84.0%(`doctor_codex_writable_root.go` の全 27 関数が 80% 以上)。`$HOME` 配下に作った fixture でビルド済みバイナリの 17 構成(ホームを root にした場合の名指し付き warn、`.agents` が symlink の場合の warn、`driver_pool` / `model_pool` に codex なしの pass、role の model 制限、`/tmp` の保存先と `exclude_slash_tmp`、重複キー、FIFO、agmsg 未導入 など)と exit code 不変を確認。`CODEX_HOME` / `TMPDIR` を差し替えても結果は同一。履歴込みの range secret scan は exit 0。tester が挙げた未到達の分岐 2 つ(覆っていない暗黙 root の読み飛ばし、config がないときの暗黙 root の pass の文言)は、報告後に orchestrator がテスト 2 件を追加して埋めた(テストのみの変更。static verify と対象テストは green)
- 2026-09-19 sync-docs cycle 2(d18fc36 / aaf876b): tech-debt の行に 4 つ目の制限(`$TMPDIR` / `AGMSG_STORAGE_PATH` は doctor 自身のプロセス環境から読むため、herdr の pane の値と違い得る)を追記。recipe / skill はコードと一致、`doctor.go` の Check 11b のコメントは短いだけで不正確ではない、との確認。同期ゲート 3 本 pass、履歴込みの range secret scan は exit 0
- 2026-09-19 cross-review cycle 2(bc36054): Codex の所見 1 件(P2)を WORTH_CONSIDERING に分類。WC-3 agmsg の保存先が座席の作業ディレクトリの中にあると、追加の root なしで書けるのに warn が出る(誤 warn。既定の保存先では起きない)。doctor は座席の `--cwd` を知り得ないので、プロジェクトディレクトリを無条件に root とみなす対応は誤 pass の危険がある。cycle 1 の 3 件は再指摘なし。上限(2/2)到達のためユーザーに確認し、判断は「PR を作成し、後続 issue で直す」。issue #170 を起票し、PR 本文の Known gaps に記録する
- 2026-09-19 pr: PR #171 を作成(`Closes #164`、Known gaps に WC-3、後続 #170)。push 前に `./scripts/run-verify.sh` green、`TMPDIR=/tmp go test ./internal/cli/... -count=1` ok、`origin/main` 基準の履歴込みの range secret scan exit 0 を確認。title prefix / ready チェック pass。walkthrough: `docs/reports/walkthrough-2026-09-19-codex-agmsg-writable-root.md`

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created (#171)
