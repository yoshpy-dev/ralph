# Walkthrough: doctor-shell-alias-check

- Date: 2026-09-19
- Plan: docs/plans/archive/2026-09-18-doctor-shell-alias-check.md(PR 作成時に active から移動)
- Issue: #162(Closes)、後続 #167
- Branch: feat/doctor-shell-alias-check(base: main @ 537faa8)
- 差分規模: 20 files / +3840 −17。実体は Go 3 ファイル(Check 本体 約 850 行、テスト 約 1,400 行、`doctor.go` の登録 10 行)と `main_test.go`、skill 4 面、recipe 2 コピー、evidence への追記 1 行、tech-debt 1 行。残りは plan と pipeline レポート 2 cycle 分

## 読む順番

1. `internal/cli/doctor_shell_alias.go` — 上から順に読める。`shellAliasEnv` と seam(`doctorShellAliasEnv`)→ 候補リスト `shellAliasRcCandidates`(login zsh が読む 4 ファイル × `$ZDOTDIR` / `~` / `~/.config/zsh`、bash、fish。stat 失敗はディレクトリ単位で報告)→ 行の reader `shellAliasStatements`(引用符・バックスラッシュ・`;|&` での文分割・`#`)→ `parseAliasStatements` / `parseAliasWords`(`alias` 文の認識、`NAME=VALUE` 複数、fish 形式)→ `scanShellAliasFile`(1 MiB 行バッファ、継続行の連結 32 行、未完の文の記録)→ フラグ判定 `shellAliasConflictingFlags` / `shellAliasFlagClass` → 文の組み立て `shellAliasCodexSentence` / `shellAliasClaudeSentence` → `checkShellAliases`(集計と重大度)
2. `internal/cli/doctor.go` — Check 8(herdr)の直後に Check 8b として登録。herdr の結果を名前付き変数で渡す
3. `internal/cli/main_test.go` — `TestMain` が seam を存在しない home に固定し、既存の `runDoctor*` テストが開発者の実 rc を読まないようにする
4. `internal/cli/doctor_shell_alias_test.go` — 各所見に対応するテストが名前で引ける(verify レポートの対応表を参照)
5. `.claude/skills/org/SKILL.md`(+ 3 ミラー)と `docs/recipes/codex-seat-permissions.md`(+ template)— alias の注意書き。どの座席にどのフラグを ralph が付けるかを Go の文と同じ内容で説明
6. `docs/evidence/codex-seat-permissions-2026-09-18.md` — P1 に追記 1 行(claude の CLI 実測)。`docs/tech-debt/README.md` — 設定を読まないことと解析の制限の 1 行

## コミット単位

| SHA | 内容 |
|---|---|
| d1c0d4c, b18e937, babde30 | plan(Codex plan advisory の MEDIUM 1 件 `-mgpt-5.5` 形式を反映) |
| 9344c57 | skill 4 面と recipe に Check への言及を追加 |
| f72e1d4 | Check 本体・テスト・登録(初版) |
| 2cfbcb9, fa7f1eb, c12169f, 8202d58 | self-review cycle 1 の 9 件と、claude の CLI 実測に基づく設計変更(claude の `--model` は info) |
| 90654ee, 33e5bad, 4352419 | 再確認の N1〜N6(permission フラグは edits / autonomous だけに付く、文の分割と継続行、`partially read`) |
| 36bda6d, eadf0ac, e4e98ab | verify の指摘 2 件、テスト 2 件追加、sync-docs(tech-debt 1 行) |
| f1898d8, c5d646f, c8e20e5 | cross-review cycle 1 の 4 件(login zsh のファイル、sandbox と approval の分離、値の再解析、stat 失敗の報告) |
| 4d26aed, 3783610, 812d5c4 | self-review cycle 2 の 6 件(改行の退行、未完の alias 文、値の中の `#`、`codex_verified` の関門、関数抽出)と verify cycle 2 の文言指摘 |
| その他 | plan の進捗・逸脱記録、self-review / verify / test / sync-docs / cross-review の各レポート(cycle 1・2)、insight events |

## 設計判断(plan Design decisions より)

- 対話シェルを起動せず rc ファイルを静的に走査する。決定的で fixture テストが書け、プロセスを起動しない。`source` されるファイルや動的に定義される alias は見えないので、Detail に走査したファイル名を必ず出す。
- 検出対象は ralph がそのドライバに実際に付けるフラグだけ。codex は `--model` / `--sandbox` / `--ask-for-approval` と短縮形 `-m` / `-s` / `-a`、claude は `--model` / `--permission-mode`。
- 重大度はフラグの種類と herdr の有無で決める。codex は重複フラグを拒否する(codex-cli 0.154.0 で実測)ので herdr があれば warn。claude は重複を受け付けて後ろの値が勝つ(claude 2.1.274 で実測)ので `--model` だけなら info、guarded 座席の permission mode を黙って変える `--permission-mode` は warn。issue の「claude も同じ形で衝突するはず(未検証)」はこの実測と合わなかった。
- ralph が permission フラグを付けるのは edits / autonomous 座席だけ(`--ask-for-approval` は autonomous だけ)で、codex のそれらのモードは `codex_verified = true` が前提。Detail の文はこの条件をそのまま書く。設定を読んで断定する版は tech-debt に記録。
- Check は exit code に影響しない(warn / info / pass のみ)。

## Known gaps

- cross-review cycle 2 の WORTH_CONSIDERING 2 件は未修正(ユーザー判断、上限 2/2 到達)。rc の候補が FIFO だと doctor が固まる。相対パスの `$ZDOTDIR` を無視し、コメントの「zsh itself would refuse it」が誤り。#167 で修正する。
- claude 座席を実際に起動しての確認はしていない(CLI 単体の実測のみ)。

## レビューで特に見てほしい箇所

- `shellAliasCodexSentence` / `shellAliasClaudeSentence` の各文が `internal/org/permissions.go` の `permissionArgsForDriver` と一致していること。
- `scanShellAliasFile` の継続行の連結が alias 文に限定されていること(無関係な行の引用符が後続の alias 行を飲み込まない)。
- alias の値が Detail に一切出ないこと(値に秘密情報が入り得るため。専用テストあり)。
