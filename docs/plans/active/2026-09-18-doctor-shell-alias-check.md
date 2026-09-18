# doctor-shell-alias-check

- Status: In progress
- Owner: Claude Code
- Date: 2026-09-18
- Related request: #155 の実機検証で、ユーザーの `~/.zshrc` の `alias codex="codex -m gpt-6-astra …"` が herdr pane の対話シェルで展開され、`ralph org spawn --model` が生成する `codex … --model` と衝突して `error: the argument '--model <MODEL>' cannot be used multiple times` で座席が即終了、`spawn_failed` になった(evidence P1)。`alias claude="claude --model fable …"` も同じ形で衝突するはず(未検証)。herdr は実行ファイル名をそのまま pane に送るため ralph 側で alias を迂回できない。`ralph doctor` で検出して warn する(issue #162)
- Related issue: 162
- Type: feat
- Branch: feat/doctor-shell-alias-check

## Objective

`ralph doctor` に「Shell aliases (codex/claude)」Check を追加する。ユーザーのシェル rc ファイルを静的に走査し、`codex` / `claude` に対する alias の値が ralph が座席起動時に付けるフラグ(`--model` / `-m`、`--permission-mode`、`--sandbox`、`--ask-for-approval`)を含む場合に warn し、evidence P1 を要約した対処(alias を外す、または alias を読まない HOME / rc で herdr を起動する、`docs/recipes/codex-seat-permissions.md` の前提)を Detail に出す。herdr が未導入なら同じ検出を info に落とす(org 座席を使わない限り害がないため)。テストは temp HOME の rc fixture で warn / pass / info / symlink 重複排除 / `ZDOTDIR` を固定する。`/org` skill の前提節と recipe の前提に「`ralph doctor` が検出する」旨を一文足す。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | Check 本体 | `internal/cli/doctor_shell_alias.go`(新規) | `checkShellAliases(herdrPresent bool) checkResult`。rc 候補: `$ZDOTDIR/.zshrc`(環境変数があれば)、`~/.zshrc`、`~/.config/zsh/.zshrc`(macOS の `/etc/zshenv` が `ZDOTDIR=$HOME/.config/zsh` を設定する慣例)、`~/.zshenv`、`~/.zprofile`、`~/.zsh_aliases`、`~/.bashrc`、`~/.bash_profile`、`~/.bash_aliases`、`~/.profile`、`~/.config/fish/config.fish`。`os.UserHomeDir()` 基準。symlink は `filepath.EvalSymlinks` で実体に解決して重複排除(この機種では `~/.zshrc` → `~/.config/zsh/.zshrc`)。各ファイルを行単位で読み、`^\s*alias\s+(codex\|claude)=(.*)$`(sh 系)と `^\s*alias\s+(codex\|claude)\s+(.*)$`(fish)に一致する行の値を調べ、`--model`(前方一致。`--model=<値>` を含む)、`-m`(空白区切りの単独トークン、または `-m<値>` のように値を連結した形。`-m` の直後が `-` 以外の文字)、`--permission-mode`、`--sandbox`、`--ask-for-approval` のいずれかを含めば衝突とみなす。読めないファイルは無視(best-effort)。home が解決できなければ info |
| 2 | 判定と Detail | 同上 | 衝突あり + herdr あり → `warn`、Detail 例: `alias codex in ~/.config/zsh/.zshrc:35 adds --model (-m); herdr expands it in the seat's pane, so ralph org spawn --model fails with "cannot be used multiple times" — remove the alias or start herdr from an alias-free rc (docs/recipes/codex-seat-permissions.md)`。複数あれば `; ` で連結。衝突あり + herdr なし → `info`(同文言 + `herdr not installed, so no seat is affected today`)。alias はあるが衝突フラグなし → `pass`(`alias codex in <file> has no conflicting flags`)。alias なし → `pass`(`no codex/claude alias in N shell rc file(s)`)。home 解決不能 → `info`。パスは home を `~` に置換して表示 |
| 3 | 登録 | `internal/cli/doctor.go` | Check 8 の herdr の直後に `results = append(results, checkShellAliases(herdrOK))` を追加(Check 番号コメントを 8b として挿入、以降の番号は変えない)。`herdrOK` は Check 8 の結果 Status が pass かどうか |
| 4 | テスト | `internal/cli/doctor_shell_alias_test.go`(新規) | temp HOME(`t.Setenv("HOME", dir)`、`t.Setenv("ZDOTDIR", "")` で既定に戻す)+ PATH に herdr stub の有無で: (a) rc なし → pass、(b) `~/.zshrc` に `alias codex="codex -m gpt-6-astra -c '…'"` → warn、Detail に `~/.zshrc:N`・`--model (-m)`・recipe パス、(b2) `-mgpt-5.5`(連結形式)と `--model=gpt-5.5`(`=` 形式)もそれぞれ warn、(c) `alias claude="claude --model fable --effort xhigh"` → warn、(d) `alias codex="codex"`(フラグなし)→ pass で alias を名指し、(e) `~/.config/zsh/.zshrc` に定義し `~/.zshrc` をその symlink にする → warn が 1 件だけ(重複なし)、(f) `ZDOTDIR=<dir>/zdot` の `.zshrc` に定義 → 検出、(g) fish `alias codex "codex -m x"` → warn、(h) (b) と同じ rc で PATH に herdr がない → info、(i) `HOME=""` → info。`runDoctorOpts` 経由の既存テストは実 HOME を読むが、warn は exit code に影響しない(`countFailed` は fail のみ) |
| 5 | 文書 | `.claude/skills/org/SKILL.md` + 3 ミラー、`docs/recipes/codex-seat-permissions.md` + template | 前提節の alias 注意に「`ralph doctor` の Shell aliases Check が warn する」を一文追加。`README.md` の doctor 記述は generic なので触らない(sync-docs が判断) |

**改訂(2026-09-18、self-review cycle 1 と CLI 実測を反映)**: 上表の項目 1〜5 は次の点で上書きする。

1. 行の解析は正規表現ではなく小さな shell word reader で行う(Assumptions 参照)
2. 検出フラグと重大度は Design decisions のドライバ別規則に従う: herdr があり、codex の所見または claude の `--permission-mode` の所見があれば `warn`。herdr がなければ所見は `info`。claude の `--model` だけの所見は常に `info`
3. すべての Detail に走査したファイル名を列挙し、それらが `source` するファイルは追わないことを書く。rc が 1 つもなければ `no shell rc file found to scan`。衝突フラグのない alias は `alias codex in <file>:<line> has no conflicting flags` と名指しする
4. 開けない / 読み切れない rc は Detail に `could not read: <file> (<理由>)` と出し、所見がなくても `pass` ではなく `info` にする。読めた分の所見は捨てない。1 行の上限は 1 MiB
5. シグネチャは `checkShellAliases(resolveEnv func() (shellAliasEnv, error), herdrPresent bool)`。`runDoctorFull` は package 変数 `doctorShellAliasEnv`(既定 `shellAliasEnvFromOS`: `os.UserHomeDir` + `ZDOTDIR`)を渡す。`internal/cli/main_test.go` の `TestMain` がこの変数を rc のないディレクトリに固定し、既存の `runDoctor*` テスト 13 箇所が開発者の実 rc を読まないようにする(self-review L7)
6. `doctor.go` は Check 8 の結果を `herdrResult` の名前付き変数で受けて渡す(L5)
7. 文書: skill 4 面と recipe の「claude の alias も衝突するはず(未検証)」を CLI 実測の結果に置き換え、evidence P1 に追記を 1 行足す

## Non-goals

- 対話シェルを起動して `alias` を評価する方式(`$SHELL -ic 'alias codex'`)。rc によっては tmux 起動やプロンプトで止まり、doctor の決定性を損なう
- `source` / `.` で読み込まれる別ファイルの再帰的追跡(候補リストにない場所の alias は検出しない。Detail に「候補 rc のみ」と明記)
- ralph 側で alias を迂回する起動方法(`command codex` 等)の実装 — herdr の integration が実行ファイル名を送るため ralph からは制御できない。必要なら herdr 側への提案
- claude 座席を実際に起動しての確認(#155 の evidence で未検証と明記済み)。CLI 単体での重複フラグの挙動は本 plan 内で実測した(Design decisions)。座席での確認は未実施

## Assumptions

- herdr の pane シェルは login zsh(`-zsh`)で、`/etc/zshenv` → `$ZDOTDIR/.zshrc` の順に読む(#155 evidence P1 で確認)。したがって `~/.config/zsh/.zshrc` を候補に含める
- `-m` の判定はトークン単位で行う: トークンがちょうど `-m`、または `-m` で始まり 3 文字目が `-` 以外(`-mgpt-5.5` の連結形式)。`--model=` は `--model` の前方一致で拾う(Codex plan advisory MEDIUM-1: 連結形式 `-mgpt-5.5` も `codex … --model gpt-5.5` と重複して exit 2 になることを advisory 側が実測)
- alias 文は小さな shell word reader で読む: 行頭(空白可)の `alias` に続く語を、単引用符 / 二重引用符 / バックスラッシュを解釈して分割する。引用符の外で語の先頭に来た `#` 以降はコメントとして捨てる。行は引用符の外の `;` `|` `&` で文に分割し、各文の先頭語(`then` / `else` / `do` / `{` / `builtin` は読み飛ばす)が `alias` なら alias 文として読む(`command -v codex >/dev/null && alias codex=…` や `alias ll=…; alias codex=…` を拾うため。self-review 再確認 N2)。alias 文が閉じていない引用符または行末のバックスラッシュで終わる場合は後続行を最大 32 行まで連結して読み、それでも閉じなければ `not fully parsed: <file>:<line>` を Detail に出す。1 つの alias 文に `NAME=VALUE` の語が複数あってもよく、オプション語(`-g` など)は読み飛ばす。fish の `alias NAME VALUE` 形式も同じ reader で読む。値は空白でトークン化して判定する。当初は「引用符を剥がさず部分文字列で判定」としていたが、行末コメント中の `--model` を誤検知する(self-review M4)ため変更した
- `checkResult` の Status 語彙(pass / info / warn / fail)と、warn が exit code を変えない契約(`countFailed`)は既存どおり
- Windows では候補 rc が存在しないため pass(`no shell rc file found to scan`)

## Affected areas

- `internal/cli/doctor_shell_alias.go`、`internal/cli/doctor_shell_alias_test.go`(新規)
- `internal/cli/doctor.go`(登録 1 行 + コメント)
- `.claude/skills/org/SKILL.md` + 3 ミラー、`docs/recipes/codex-seat-permissions.md` + template

## Design decisions

Critical forks: None。

既定として採った選択:

- 静的な rc 走査を採る(Non-goals 参照)。決定的で fixture テストが書け、プロセスを起動しない。見逃し(候補外の rc、動的定義)は Detail と doc comment で明示する
- herdr 未導入時は info に落とす。org 座席を使わないユーザーにとって alias は害がなく、doctor の warn を増やすと本当に効く warn が埋もれる
- 検出対象フラグは ralph がそのドライバに実際に付けるものに限定する。codex: `--model` / `-m`、`--sandbox` / `-s`、`--ask-for-approval` / `-a`。短縮形は clap 上同じ引数で、codex-cli 0.154.0 で `-s read-only --sandbox workspace-write` と `-a never --ask-for-approval never` がどちらも `cannot be used multiple times` になることを実測した(self-review H1)。claude: `--model`、`--permission-mode`(claude 2.1.274 の `--help` に該当する短縮形はない)。`--effort` や `-c key=value` は評価対象外
- ドライバで重大度を分ける。codex は重複フラグを拒否して座席が起動しないので、herdr があれば `warn`。claude は重複を受け付けて後ろの値が勝つ(claude 2.1.274 で `--model haiku --model sonnet -p …` の `modelUsage` に sonnet、逆順では haiku のみ。`--permission-mode plan --permission-mode acceptEdits` もエラーにならない)。ralph のフラグは alias 展開の後ろに付くので ralph の値が効き、座席は起動する。したがって claude の `--model` の所見は `info` とし、alias の他のフラグが全座席に効くことを伝える。ただし ralph が permission フラグを付けるのは edits / autonomous 座席だけで、guarded 座席には何も付けない(`internal/org/permissions.go` の `permissionArgsForDriver`)。そのため claude の `--permission-mode` の alias は guarded 座席の permission mode を黙って変える(herdr があれば `warn`)。codex の `--sandbox` / `--ask-for-approval` の alias も、edits / autonomous では spawn 失敗、guarded では alias の値が黙って効く。Detail の文はフラグの種類ごとにこの条件を書き分ける(self-review 再確認 N1)。issue #162 の「claude も同じ形で衝突するはず(未検証)」はこの実測と合わなかった
- Check 名は「Shell aliases (codex/claude)」。既存の Check 名の英語表記に合わせる

## Acceptance criteria

- [ ] AC-1: `ralph doctor` の出力に「Shell aliases (codex/claude)」Check が herdr Check の直後に出る。`grep -c 'checkShellAliases' internal/cli/doctor.go` が 1 以上
- [ ] AC-2: テスト (a)〜(k) と改訂で追加したケースが pass し、codex の warn Detail が `file:line`(home は `~`)、衝突フラグ名、`docs/recipes/codex-seat-permissions.md` を含む。`-m value` / `-mvalue` / `--model=value`、`-s value` / `-svalue`、`-a value` の各形式が検出される fixture がある。行末コメント中の `--model` は検出されない。1 つの alias 文に `codex=` と `claude=` が並ぶ場合にそれぞれ正しい名前で報告される。symlink ケースで所見が 1 件だけ。`ZDOTDIR` ケースで検出される
- [ ] AC-3: herdr あり + codex の所見 は `warn`、herdr あり + claude の `--permission-mode` の所見 は `warn`、herdr 未導入の所見 は `info`、claude の `--model` だけの所見 は `info`、解析しきれなかった alias 文(`not fully parsed`)があり他に所見なし は `info`、alias なし / 衝突なし は `pass`(衝突なしの alias は file:line で名指し)、home 解決不能は `info`、読めない rc があり所見なし は `info`。どの Detail も走査したファイルを列挙する。`countFailed` の対象にならない(既存の `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` が引き続き pass)
- [ ] AC-4: `/org` skill の前提節(4 面 `cmp` 一致)と recipe(root / template 一致)に `ralph doctor` の Check への言及がある。`check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` pass
- [ ] AC-5: `gofmt` / `go vet` / golangci-lint clean、`go test ./internal/cli/... -count=1` green、`./scripts/run-verify.sh` green。このマシンで `go run ./cmd/ralph doctor` を実行すると本 Check が warn し、`~/.config/zsh/.zshrc:35` の codex alias を spawn 失敗の原因として、`:34` の claude alias を「座席は起動するが alias のフラグが全座席に効く」ものとして名指しする(evidence としてレポートに記録)
- [ ] AC-6: PR 本文は `Closes #162`

## Implementation outline

1. Slice A(Go、implementer 委譲): 項目 1〜4。`gofmt -l internal/`、`go vet ./internal/cli/...`、`go test ./internal/cli/ -run 'ShellAlias|TestRunDoctorOpts' -count=1 -v`、`go test ./internal/cli/... -count=1`、`./scripts/run-verify.sh` → commit `feat: warn in ralph doctor when a codex/claude shell alias adds seat-launch flags`
2. Slice B(docs、inline): 項目 5 → `sync-skills.sh` + cp → `check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` → commit `docs: mention the doctor shell-alias check in the /org skill and the codex seat recipe`
3. 実機確認: `go run ./cmd/ralph doctor | grep -A1 'Shell aliases'` の出力を plan の Deviation notes に記録 → post-implementation pipeline → `/cross-review` → `/pr`(`Closes #162`)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`check-skill-sync.sh`、`check-sync.sh`、`check-template-purity.sh`
- Spec compliance criteria to confirm: AC-1〜AC-6 の grep / cmp。Detail 文言が doc comment と一致すること
- Documentation drift to check: skill 4 面と recipe の文言が Check 名・挙動(warn / info)と一致すること。`docs/evidence/codex-seat-permissions-2026-09-18.md` P1 と矛盾しないこと
- Evidence to capture: grep / cmp の出力、このマシンでの doctor 出力

## Test plan

- Unit tests: 項目 4 の (a)〜(i)
- Integration tests: `runDoctorOpts` 経由の既存テスト(exit code 不変)
- Regression tests: `go test ./internal/... -count=1`
- Edge cases: (1) コメント行 `# alias codex=…` は無視、(2) `alias codex='codex'`(フラグなし)は pass、(3) 同じ alias が複数ファイルにある場合は各 file:line を列挙、(4) rc が読めない(権限)場合は Detail に出して info、(6) 64 KiB を超える長い行を含む rc でも以降の alias を検出する、(5) `ZDOTDIR` が相対パスや空
- Evidence to capture: `go test -count=1 -v` の出力、`docs/reports/test-*.md`

## Risks and mitigations

- 実 HOME を読む既存の doctor テストが環境依存になる → `TestMain` が `doctorShellAliasEnv` を rc のないディレクトリに固定する。既存テストは warn / pass どちらでも exit code が変わらない
- 誤検知(alias の値に `-m` が別の意味で含まれる)→ warn は助言で、Detail に該当行を示すので運用者が判断できる
- 見逃し(候補外の rc)→ Non-goals と Detail に明記

## Rollout or rollback notes

doctor の Check 追加のみ。下流へは次回 release でバイナリ経由、skill / recipe の文言は `ralph upgrade` / `ralph init` で配布。revert は PR 単位で安全。

## Open questions

なし。

## Deviation notes

- 2026-09-18 plan: Codex plan advisory(codex-cli 0.154.0、`</dev/null` 付き)が MEDIUM 1 件(`-mgpt-5.5` の連結形式を見逃す)を報告。採用し Scope 1・Assumptions・AC-2 を改訂。Slice B(文書)は advisory 待ちの間に先行して 9344c57 で実施
- 2026-09-18 work: Slice A は implementer に委譲(f72e1d4、3 ファイル、逸脱なし)。orchestrator が HEAD 一致・porcelain 空・差分を確認し、対象テスト 14 件+既存 `TestRunDoctorOpts_*` 2 件を再実行して pass。テストは (a)〜(i) に加えて (j) コメント行、(k) `conflictingFlag` の表(`--models-dir` や `--m` を誤検知しないこと)、`runDoctorOpts` 出力に Check 行が出る統合テストを持つ
- 2026-09-18 work: AC-5 の実機 evidence。このマシンで `go run ./cmd/ralph doctor` を実行した結果(1 行): `⚠ Shell aliases (codex/claude): warn — alias claude in ~/.config/zsh/.zshrc:34 adds --model; alias codex in ~/.config/zsh/.zshrc:35 adds --model (-m) — herdr expands the alias in the seat's pane, ... (docs/recipes/codex-seat-permissions.md)`。`~/.zshrc` は同ファイルへの symlink なので所見は 1 ファイル分だけ
- 2026-09-18 self-review cycle 1(`docs/reports/self-review-2026-09-18-doctor-shell-alias-check.md`): HIGH 1 / MEDIUM 3 / LOW 5。全件を同 cycle 内で修正する。H1 codex の `-s` / `-a` 未検出、M2 Detail が走査ファイルを名指ししない、M3 開けない rc と scanner エラーを黙って捨て部分所見も失う、M4 行末コメントと同一行の 2 つ目の alias を誤帰属、L5 `results[len(results)-1]` の位置結合、L6 `conflictingFlag` の命名、L7 既存 `runDoctor*` テスト 13 箇所が実 rc を読む、L8 `ZDOTDIR` まわりの未記載の挙動、L9 skill の行長
- 2026-09-18 CLI 実測(orchestrator): claude 2.1.274 は `--model` / `--permission-mode` の重複をエラーにせず後ろの値が勝つ。codex-cli 0.154.0 は `-s` / `-a` も長い形式と重複エラーになる。これを受けて claude の所見を `warn` から `info` に変更し(Design decisions)、Scope の改訂段落・Assumptions・AC-2 / AC-3 / AC-5 を書き換えた。skill / recipe の「claude も衝突するはず(未検証)」も実測結果に置き換える
- 2026-09-18 work: Slice C(self-review の修正)は implementer に委譲(c12169f、4 ファイル)。逸脱 1 件: gofmt が doc comment 内の連続した引用符を曲線引用符に書き換えるため、該当例を散文に言い換えた。orchestrator が HEAD 一致・porcelain 空・差分を確認し、`shellAliasConflictingFlags` に残っていた重複の引用符剥離(word reader が処理済み)を 8202d58 で除去。対象テスト再実行と `./scripts/run-verify.sh` は green
- 2026-09-18 work: AC-5 の実機 evidence(改訂後)。`go run ./cmd/ralph doctor` の該当行: `⚠ Shell aliases (codex/claude): warn — alias codex in ~/.config/zsh/.zshrc:35 adds --model (-m) — herdr expands the alias in the seat's pane and codex rejects the repeated flag ("cannot be used multiple times"), so ralph org spawn fails; remove the alias or start herdr from an alias-free rc (docs/recipes/codex-seat-permissions.md). alias claude in ~/.config/zsh/.zshrc:34 adds --model — claude accepts the repeated flag and the value ralph org spawn passes last wins, so the seat still starts; the alias's other flags reach every seat. scanned 4 shell rc file(s): ~/.config/zsh/.zshrc, ~/.config/zsh/.zshenv, ~/.zprofile, ~/.profile (files they source are not followed)`
- 2026-09-18 self-review 再確認(同 report の Revalidation 節、1cf548d): 当初 9 件はすべて解消。修正で生じた新所見 6 件(MEDIUM 2 / LOW 4)も同 cycle 内で直す。N1 Detail と文書が「ralph も同じフラグを付ける」前提で書かれているが permission フラグは guarded では付かない → 文をフラグ種別ごとに書き分け、claude の `--permission-mode` は warn に上げる。N2 継続行と `;` 以降の alias を見逃したまま pass と言い切る → 行を文に分割して読み、継続行を連結し、読み切れなければ `not fully parsed` を出す。N3 部分読み取りの Detail が自己矛盾 → `partially read:` に分け、走査済みに数える。N4 「claude に短縮形はない」は誤り → 該当 2 フラグに短縮形がない、に修正。N5 型名 `shellAliasWord` → `shellAliasAssignment`。N6 Assumptions に残った旧文言を更新

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
