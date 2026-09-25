# doctor-shell-alias-rc-types

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-25
- Related request: #162(`ralph doctor` の「Shell aliases (codex/claude)」Check、PR #168)の cross-review cycle 2 で WORTH_CONSIDERING が 2 件残り、cap 到達のためユーザー判断でこの issue に送った(triage: `docs/reports/cross-review-triage-doctor-shell-alias-check.md` の WC-3 / WC-4)。(1) rc の候補が FIFO だと `os.Open` が書き手を待って固まり、`ralph doctor` 全体が何も出さずに止まる。(2) 相対パスの `$ZDOTDIR` を無視しており、doc comment の「zsh itself would refuse it」は誤り(zsh はシェルの作業ディレクトリ基準で解決して読む)
- Related issue: 167
- Type: fix
- Branch: fix/doctor-shell-alias-rc-types

## Objective

`ralph doctor` の「Shell aliases (codex/claude)」Check が、通常ファイルでない rc(FIFO、デバイスなど)を開かずに「読めなかった」と報告し、相対パスの `$ZDOTDIR` も走査の対象にする。

## Scope

1. **通常ファイルでない rc を開かない(WC-3)**: 候補の rc が存在し(symlink は辿る)、ディレクトリでも通常ファイルでもないとき、`os.Open` を呼ばない。Detail の既存の「could not read」の節に `<file> (not a regular file)` として出し、Check は info(他の rc の所見による warn はそのまま優先)。ディレクトリは今までどおり黙って飛ばす。判定と文言は #164 の `readCodexUserConfig`(`internal/cli/doctor_codex_writable_root.go`、stat して通常ファイルでなければ開かずに `not a regular file`)に揃える
2. **相対パスの `$ZDOTDIR`(WC-4)**: `shellAliasEnv` に doctor の作業ディレクトリ(`Cwd`)を足し、`shellAliasEnvFromOS` は `os.Getwd()` で埋める。相対の `$ZDOTDIR` は `Cwd` を基準に絶対パスにして、絶対パスの `$ZDOTDIR` と同じ位置(候補の先頭)で走査する。`Cwd` が空(`os.Getwd` の失敗)のときは相対の `$ZDOTDIR` を今までどおり飛ばす。環境の値は seam から注入し、テストは実際の cwd や環境変数に依存しない
3. **コメント**: 「zsh itself would refuse it」を消し、zsh は相対の `ZDOTDIR` をシェルの作業ディレクトリ基準で解決すること、herdr の pane の cwd(座席の `--cwd`)は doctor の cwd と一致するとは限らず doctor は自分の cwd で近似すること、候補の一覧は over-approximate の方針のままであることを書く

## Non-goals

- `source` で読み込まれるファイルを辿ること(既存の方針のまま)
- 相対の `$ZDOTDIR` を座席の実際の cwd(`ralph.toml` や org の状態)から解決すること
- stat と open の間に rc が FIFO に差し替えられる競合の対策(#164 の前例と同じ扱い。差し替えられる者はすでにアカウントを握っている)
- `/dev/null` への symlink などの特別扱い(通常ファイルでないので「読めなかった」と正直に出す)
- 重大度の規則(flag の分類、herdr の有無)の変更
- Windows(リリース対象外。FIFO のテストは `//go:build !windows` の別ファイル)

## Assumptions

- `internal/cli` のテストは unix でだけ走る(`internal/org/lockfile.go` が `syscall.Flock` を無条件に使う)。FIFO のテストは #164 と同じく `doctor_*_unix_test.go` に置く
- Go 1.25 なので `t.Chdir` が使える(`shellAliasEnvFromOS` の `Cwd` のテスト用)
- `internal/cli/main_test.go` の TestMain は `doctorShellAliasEnv` を存在しないホームに固定している。`Cwd` を足しても、その固定を壊さない(空の `Cwd` は「相対の `$ZDOTDIR` を飛ばす」になる)

## Affected areas

- `internal/cli/doctor_shell_alias.go`(`shellAliasEnv`、`shellAliasEnvFromOS`、`shellAliasRcCandidates`、`scanShellAliasFile` かその呼び出し側、doc comment)
- `internal/cli/doctor_shell_alias_test.go`(相対 `$ZDOTDIR`、`shellAliasEnvFromOS` の `Cwd`、ディレクトリの回帰)
- `internal/cli/doctor_shell_alias_unix_test.go`(新規、FIFO)
- 必要なら `internal/cli/main_test.go`(TestMain の固定に `Cwd` を明示する場合だけ)
- 文書: 変更なしの見込み(recipe と `/org` skill は Check の存在に触れるだけで、候補の一覧や `$ZDOTDIR` の扱いは書いていない。sync-docs で再確認)

## Design decisions

- 通常ファイルの判定は「stat して通常ファイルでなければ開かない」(#164 の前例)。open に `O_NONBLOCK` を付けて開いた後に fstat する方式もあるが、前例と揃える方を採る(競合は Non-goals)
- 相対の `$ZDOTDIR` の基準は doctor の cwd を seam(`shellAliasEnv.Cwd`)で渡す。`filepath.Abs` を直接呼ぶとテストが実際の cwd に依存するため
- Critical forks: None(issue が方式を指定しており、前例がある)

## Acceptance criteria

- [ ] AC-1: 候補の rc(`~/.zshrc`)が FIFO のとき、`checkShellAliases` は 5 秒以内に返り、Check は info、Detail の「could not read」の節に `~/.zshrc (not a regular file)` が入る。FIFO を開かない
- [ ] AC-2: 候補の rc が FIFO を指す symlink のときも AC-1 と同じ(symlink の候補名で出る)
- [ ] AC-3: FIFO の rc があっても他の候補は走査され、別の rc にある codex の alias(herdr あり)は warn になり、Detail に両方が出る
- [ ] AC-4: 候補の位置にあるディレクトリは今までどおり黙って飛ばす(Detail に「could not read」が出ない)
- [ ] AC-5: `$ZDOTDIR` が相対パス(例 `relative-rc`)で `Cwd` が与えられたとき、`<Cwd>/relative-rc/.zshrc` の codex の alias が検出される(herdr ありで warn)。Detail は解決後のパスで file:line を示す
- [ ] AC-6: `$ZDOTDIR` が相対パスで `Cwd` が空のとき、候補は `$ZDOTDIR` 未設定と同じになり、panic しない
- [ ] AC-7: `shellAliasEnvFromOS` は `Cwd` を `os.Getwd()` の値で埋める。絶対パスの `$ZDOTDIR` と未設定の挙動は変わらない(既存のテストが pass)
- [ ] AC-8: `internal/cli/doctor_shell_alias.go` に「would refuse」の記述が残っていない。相対の `$ZDOTDIR` の解決と、doctor の cwd による近似がコメントに書かれている
- [ ] AC-9: `./scripts/run-verify.sh` green、`TMPDIR=/tmp go test ./internal/cli/... -count=1` pass

## Implementation outline

1. Slice A(WC-3): 通常ファイルの判定と「could not read」への合流。FIFO のテスト(直接、symlink 経由、他の rc の所見との併存)とディレクトリの回帰テスト
2. Slice B(WC-4): `shellAliasEnv.Cwd`、`shellAliasEnvFromOS` の `os.Getwd()`、候補での相対 `$ZDOTDIR` の解決、コメントの訂正。テスト(相対で検出、`Cwd` 空、`shellAliasEnvFromOS` の `Cwd`)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt / go vet / 既存の lint)
- Spec compliance criteria to confirm: AC-1〜AC-9 をコードとテストに対応付ける。Non-goals(`source` を辿らない、重大度の規則は不変)を維持している
- Documentation drift to check: `docs/recipes/codex-seat-permissions.md`(2 コピー)と `/org` skill の Check の記述が変更後も正しいこと。`docs/tech-debt/README.md` の shell alias Check の行
- Evidence to capture: 実機(ビルドした `ralph`)で、FIFO の `.zshrc` を置いた偽 HOME に対し `ralph doctor` が固まらず終わり、Check の行に `not a regular file` が出ること(修正前のバイナリは固まることも確認し、時間の上限を付けて打ち切る)

## Test plan

- Unit tests: `checkShellAliases` の FIFO(直接、symlink 経由)、他の rc の所見との併存、ディレクトリの回帰、相対 `$ZDOTDIR`(検出、`Cwd` 空)、`shellAliasEnvFromOS` の `Cwd`(`t.Chdir`)
- Integration tests: `runDoctor*` 経由の既存テストが TestMain の固定のまま pass すること
- Regression tests: 既存の shell alias のテスト一式(絶対 `$ZDOTDIR`、symlink の dedup、読めない rc、長い行 ほか)
- Edge cases: FIFO を指す symlink、`$ZDOTDIR` が `.`(cwd そのもの)、相対 `$ZDOTDIR` の解決先が `$HOME` と同じ(dedup で 1 回)
- Evidence to capture: red/green(判定を外すと FIFO のテストが時間切れで落ちる、相対の解決を外すと AC-5 のテストが落ちる)、`TMPDIR=/tmp` での実行、race

## Risks and mitigations

- rc が `/dev/null` への symlink の環境では、Check が pass から info(`not a regular file`)に変わる。info なので doctor は失敗しない。事実(読んでいない)を言っているので受け入れる
- stat と open の間の差し替えで固まる可能性は残る(Non-goals)。前例と同じ扱い
- 相対の `$ZDOTDIR` を doctor の cwd で解決するので、座席の cwd と違うディレクトリを読むことがある。候補の一覧は元々 over-approximate で、余分に読んだ結果の warn は「その alias が pane で効く可能性がある」の範囲に収まる
- FIFO のテストが失敗したとき、テストの goroutine が open で止まったまま残る。時間切れで Fatal にし、後始末で FIFO を書き込み側で開いて解放する

## Rollout or rollback notes

- 変更は doctor の 1 つの Check に閉じる。revert すれば従来の挙動に戻る。設定やファイルの形式は変えない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 現状のコード(候補の解決、走査、Detail の組み立て、TestMain の固定)と #164 の前例を確認した
- [x] critical fork なし
- [x] AC は hermetic な fixture(`t.TempDir()`、seam の `shellAliasEnv`)で決定的に確認できる
