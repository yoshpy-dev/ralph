# Walkthrough: codex-agmsg-writable-root

- Date: 2026-09-19
- Plan: docs/plans/archive/2026-09-19-codex-agmsg-writable-root.md(PR 作成時に active から移動)
- Issue: #164(Closes)、後続 #170
- Branch: feat/codex-agmsg-writable-root(base: main @ 2c511a4)
- 差分規模: 21 files / +4370 −6。実体は Go 4 ファイル(Check 本体 約 960 行、テスト 約 1,990 行、`doctor.go` の登録 7 行、`main_test.go` の seam の固定)、skill 4 面、recipe 2 コピー、evidence への追記 1 行、tech-debt 1 行、`.gitallowed` の規則 1 行。残りは plan と pipeline レポート 2 cycle 分

## 読む順番

1. `internal/cli/doctor_codex_writable_root.go` — 上から順に読める。環境の seam(`codexSandboxEnv` / `codexSandboxEnvFromOS` / `doctorCodexSandboxEnv`)→ config の読み取り(`readCodexUserConfig`: 通常ファイル以外は開かない、1 MiB 上限、decode 失敗はメッセージ本文を持たない `codexConfigDecodeError` に包む)→ 理由の判定(`codexModelPoolModels` / `codexModelPermittedForRole` / `codexSeatModesPossible` / `codexSandboxReasons`)→ 保存先(`agmsgStoreDir`)→ 覆っているかの判定(`pathCovers` / `pathCrossesCodexProtectedDir` / `resolveNearestExisting` / `codexRootCoverage` / `coveringWritableRoot` / `codexImplicitWritableRoots` / `codexCoveringImplicitRoot`)→ Detail の組み立て → `checkCodexAgmsgWritableRoot`(doc comment の番号付きの結果一覧がコードの順序と一致)
2. `internal/cli/doctor.go` — Check 11(codex スラッグ)の直後に Check 11b として登録
3. `internal/cli/main_test.go` — `TestMain` が seam を「存在しない config、空の override、暗黙の一時 root なし」に固定
4. `internal/cli/doctor_codex_writable_root_test.go` / `_unix_test.go` — 結果ごとのテストが名前で引ける(verify レポートの対応表を参照)。FIFO と権限のテストは `!windows` のファイル
5. `docs/recipes/codex-seat-permissions.md`(+ template)と `.claude/skills/org/SKILL.md`(+ 3 ミラー)— writable root の節 / permission 作法の codex の行
6. `docs/tech-debt/README.md` — この Check が確認していない 4 点。`.gitallowed` — レポートの衛生確認の言い回しだけに一致する規則

## コミット単位

| SHA | 内容 |
|---|---|
| e78dd15, a49763a | plan(方式はユーザー決定「doctor で検査」。Codex plan advisory の MEDIUM 2 件: `AGMSG_STORAGE_PATH`、導入判定は `driver.AgmsgAvailable`) |
| e2b558a, 445cea7, f14737f | Check 本体・テスト・登録(初版)、symlink 解決の共通化、skill / recipe |
| 636da6a, 0811070, 65a6f86, 5bacc46 | self-review cycle 1 の 9 件(config のキー名の漏れ、`TMPDIR` 依存のテスト、解決しない比較、理由を role / `driver_pool` から導く ほか) |
| 4b7a2b0, 33ac5a8 | 再確認の LOW 6 件(実効 default の解決、理由の番号付け ほか) |
| 78c4ed9, 04b6df5, 90dc4be, ee70868, e2c64c3 | テスト 2 件追加、sync-docs、secret scan の許可リスト(広い 2 行を狭い 1 行に置換) |
| 2fecee7, ab09dbc, 91a5542, c03c21e, 4c891ec | cross-review cycle 1 の 3 件(保護ディレクトリ、暗黙の一時 root、model の条件)と、固定の一時 root を seam から注入する修正(CI の ubuntu でテスト 16 件が落ちる問題を push 前に発見) |
| 235abb0, 735fc45, 5fcf070, a16e3f6, c83ce78, d18fc36 | self-review cycle 2 の 8 件(`.agents` が symlink の場合、綴り 4 通りの組み合わせ ほか)、verify cycle 2 の指摘、テスト 2 件、tech-debt |
| その他 | plan の進捗・逸脱記録、各レポート(cycle 1・2)、insight events |

## 設計判断(plan Design decisions より)

- ralph は座席の sandbox を自動で広げない。設定漏れを検出して recipe に案内する(ユーザー決定)。
- 判定は warn 側に倒す。誤 pass は「RESULT が届かない」という、この Check が見つけるべき失敗を隠すため。`~` / 相対の root、保護ディレクトリの深さ、綴りの組み合わせは、いずれも不確かな場合に「覆っていない」とする。
- 理由は設定から導く。`driver_pool` / `model_pool` に codex がなければ不要。`codex_verified = true` で codex の model を使える role が edits / autonomous に解決される場合と、codex config が `sandbox_mode = "workspace-write"` で同様の role が guarded に解決される場合だけ root が必要。
- config の内容は Detail に出さない。出すのは読んだパス、保存先、覆っている root、`profile` の名前だけ。decode 失敗は行と列だけ。
- Check は exit code に影響しない(warn / info / pass のみ)。

## Known gaps

- cross-review cycle 2 の WORTH_CONSIDERING 1 件は未修正(ユーザー判断、上限 2/2 到達)。agmsg の保存先が座席の作業ディレクトリの中にあると、追加の root なしで書けるのに warn が出る。doctor は座席の `--cwd` を知り得ないため、warn の文を補足する形で #170 で対応する。
- codex 座席を実際に起動しての再検証はしていない。「root を足せば RESULT が届く」ことは #155 の Run A2 が実証済みで、本 Check は設定漏れの検出だけを行う。
- 読むのはユーザー階層の `config.toml` だけ(profile、project 階層の config、`-c` の上書きは評価しない)。tech-debt に記録。

## レビューで特に見てほしい箇所

- `codexRootCoverage`: 覆っているかは解決後の綴りで、保護ディレクトリは綴り 4 通りの組み合わせで判定していること。
- `codexSeatModesPossible` が `internal/org/envelope.go`(`modelAllowedForRole`)と `internal/org/permissions.go`(`ResolvePermissionMode`)に一致していること。
- 環境の値(home、`TMPDIR`、固定の一時 root、`AGMSG_STORAGE_PATH`、config のパス)を `codexSandboxEnvFromOS` だけが読むこと。
- `.gitallowed` の新しい規則が、衛生確認の言い回しだけに一致すること(scanner の免除は行単位)。
