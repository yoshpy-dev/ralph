# Walkthrough: doctor-agmsg-store-in-project

- Date: 2026-09-27
- Plan: docs/plans/archive/2026-09-26-doctor-agmsg-store-in-project.md(PR 作成時に active から移動)
- Issue: #170(Closes)。#164(PR #171)の cross-review cycle 2 で残った WORTH_CONSIDERING 1 件(WC-3)
- Branch: fix/doctor-agmsg-store-in-project(base: main @ 88e7429)
- 差分規模: 17 files / +1,083 −68。コードは `internal/cli/` の 4 ファイル(+436 −61): `doctor_codex_writable_root.go`、`doctor.go`(呼び出しの 1 か所)、テスト 2 ファイル。文書は recipe 2 コピー、`/org` skill 4 面、tech-debt(+34 −7)。残りは plan と pipeline のレポート

## 何が変わったか

`ralph doctor` の「Codex sandbox (agmsg writable root)」Check の warn の文言です。

1. **プロジェクト内の保存先の warn を条件付きにする**: agmsg の保存先(`<agmsg home>/db` または `AGMSG_STORAGE_PATH`)が doctor の対象ディレクトリの中にあり、codex が保護する `.git` / `.agents` / `.codex` をまたがないとき、Detail は「作業ディレクトリがこの保存先を含む codex の座席はすでに書ける。作業ディレクトリが保存先を含まない座席(task worktree など)は writable root を足さないと RESULT を送れない」と述べる。以前は無条件に「送れない、root を足せ」と書いていた(codex の workspace-write は作業ディレクトリの中には書けるので、誤った案内)。config がある場合とない場合の両方の文に適用。
2. **status は warn のまま**: doctor は座席の `--cwd` を知らない(座席は task worktree のような、保存先を含まない作業ディレクトリで動くことがある)。プロジェクトディレクトリを writable root とみなして pass にはしない。
3. **判定**: `runDoctorFull` が `targetDir` を絶対パスにして `checkCodexAgmsgWritableRoot` に渡す。warn のときだけ、明示の root と同じ `codexRootCoverage`(symlink を解決した組み合わせも含めて保護ディレクトリをまたがないことを確かめる)でプロジェクトが保存先を含むかを判定する。含まない場合、保護ディレクトリをまたぐ場合、`targetDir` が解決できない場合の文は main と byte 一致。

## 読む順番

1. `internal/cli/doctor_codex_writable_root.go` — `codexWritableRootDetail`(`containingProject` が空でないときの 2 つの文)→ `checkCodexAgmsgWritableRoot` の末尾(warn の直前の判定)→ 関数の doc comment の outcome 9
2. `internal/cli/doctor.go` — `projectDir` の受け渡し
3. `internal/cli/doctor_codex_writable_root_test.go` — `// --- issue #170 ---` 以降(プロジェクト内、保護ディレクトリの下、プロジェクト外、symlink、`AGMSG_STORAGE_PATH`、保存先 = プロジェクト)と、`runDoctorOpts` 経由で配線を固定するテスト
4. `docs/recipes/codex-seat-permissions.md`、`.claude/skills/org/SKILL.md`、`docs/tech-debt/README.md` の該当行

## コミット単位

| SHA | 内容 |
|---|---|
| 8ee7762, 6fcab50 | plan。Codex plan advisory の MEDIUM 1 件(末尾に補足を足すだけでは既存の断定と矛盾する)を反映し、文そのものを条件付きにする方針に |
| 24e455f | `projectDir` の引数、条件付きの 2 つの文、テスト |
| c71a93b | recipe、skill、tech-debt の 1 文ずつ |
| a2faa60 | self-review の LOW 6 件(対比の軸を「作業ディレクトリが含むか」に、引数を `containingProject` の 1 つに、コメント、保護ディレクトリの例外を文書に)と、配線を固定するテスト、否定の確認 |
| その他 | plan の記録(中断と再開の手順を含む)、各レポート、insight events、tech-debt の行 |

## 設計判断

- 補足を末尾に足すのではなく、文そのものを条件付きにする(同じ診断の中で「送れない」と「すでに書ける」が並ぶと、不要な設定を誘発する)。
- 判定は既存の coverage の規則を再利用する。新しい規則を作らない。
- `projectDir` は明示の引数で渡す(環境の struct や `os.Getwd` に隠すと hermetic でなくなる)。

## 注意して見てほしい点

- プロジェクト外の文が main と byte 一致であること(tester が main のバイナリと比較)。
- `containingProject` が空でない値になるのは `covers && !blocked` のときだけであること。
- status が warn のままであること。

## Known limitations

- doctor は座席の `--cwd` を知らないので、プロジェクト内の保存先でも warn のまま(pass にしない)。tech-debt に記録済み。
- `ralph doctor` をプロジェクトのサブディレクトリから実行すると、そこに `ralph.toml` がないため既定値で動き、この Check の warn には到達しない(main からある挙動。tech-debt に記録)。
- 保存先 = プロジェクトディレクトリのテストは「inside this project」の部分文字列だけを確かめており、2 つの文の入れ替えはほかのテストで検出する(tech-debt に記録)。
