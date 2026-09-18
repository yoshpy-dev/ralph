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

## Non-goals

- 対話シェルを起動して `alias` を評価する方式(`$SHELL -ic 'alias codex'`)。rc によっては tmux 起動やプロンプトで止まり、doctor の決定性を損なう
- `source` / `.` で読み込まれる別ファイルの再帰的追跡(候補リストにない場所の alias は検出しない。Detail に「候補 rc のみ」と明記)
- ralph 側で alias を迂回する起動方法(`command codex` 等)の実装 — herdr の integration が実行ファイル名を送るため ralph からは制御できない。必要なら herdr 側への提案
- claude 座席での衝突の実機検証(#155 の evidence で未検証と明記済み。本 Check は claude alias も同じ規則で warn するが、evidence は codex のみ)

## Assumptions

- herdr の pane シェルは login zsh(`-zsh`)で、`/etc/zshenv` → `$ZDOTDIR/.zshrc` の順に読む(#155 evidence P1 で確認)。したがって `~/.config/zsh/.zshrc` を候補に含める
- `-m` の判定はトークン単位で行う: トークンがちょうど `-m`、または `-m` で始まり 3 文字目が `-` 以外(`-mgpt-5.5` の連結形式)。`--model=` は `--model` の前方一致で拾う(Codex plan advisory MEDIUM-1: 連結形式 `-mgpt-5.5` も `codex … --model gpt-5.5` と重複して exit 2 になることを advisory 側が実測)
- alias の値の引用符(`"…"` / `'…'`)は剥がさず、部分文字列として判定する(誤検知より見逃しが少ない側に倒す。`# alias codex=` のようなコメント行は行頭の `#` で除外)
- `checkResult` の Status 語彙(pass / info / warn / fail)と、warn が exit code を変えない契約(`countFailed`)は既存どおり
- Windows では候補 rc が存在しないため pass(`no codex/claude alias in 0 shell rc file(s)`)

## Affected areas

- `internal/cli/doctor_shell_alias.go`、`internal/cli/doctor_shell_alias_test.go`(新規)
- `internal/cli/doctor.go`(登録 1 行 + コメント)
- `.claude/skills/org/SKILL.md` + 3 ミラー、`docs/recipes/codex-seat-permissions.md` + template

## Design decisions

Critical forks: None。

既定として採った選択:

- 静的な rc 走査を採る(Non-goals 参照)。決定的で fixture テストが書け、プロセスを起動しない。見逃し(候補外の rc、動的定義)は Detail と doc comment で明示する
- herdr 未導入時は info に落とす。org 座席を使わないユーザーにとって alias は害がなく、doctor の warn を増やすと本当に効く warn が埋もれる
- 検出対象フラグは ralph が実際に付けるもの(`--model`、claude の `--permission-mode`、codex の `--sandbox` / `--ask-for-approval`)と codex の短縮形 `-m` に限定する。`--effort` や `-c key=value` は重複しても codex / claude がエラーにしない(評価対象外)
- Check 名は「Shell aliases (codex/claude)」。既存の Check 名の英語表記に合わせる

## Acceptance criteria

- [ ] AC-1: `ralph doctor` の出力に「Shell aliases (codex/claude)」Check が herdr Check の直後に出る。`grep -c 'checkShellAliases' internal/cli/doctor.go` が 1 以上
- [ ] AC-2: テスト (a)〜(i) が pass し、warn の Detail が `file:line`(home は `~`)、衝突フラグ名、`docs/recipes/codex-seat-permissions.md` を含む。`-m value` / `-mvalue` / `--model=value` の 3 形式すべてが warn になる fixture がある。symlink ケースで warn が 1 件だけ。`ZDOTDIR` ケースで検出される
- [ ] AC-3: herdr 未導入 + 衝突あり は `info`、alias なし / 衝突なし は `pass`、home 解決不能は `info`。`countFailed` の対象にならない(既存の `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` が引き続き pass)
- [ ] AC-4: `/org` skill の前提節(4 面 `cmp` 一致)と recipe(root / template 一致)に `ralph doctor` の Check への言及がある。`check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` pass
- [ ] AC-5: `gofmt` / `go vet` / golangci-lint clean、`go test ./internal/cli/... -count=1` green、`./scripts/run-verify.sh` green。このマシンで `go run ./cmd/ralph doctor` を実行すると本 Check が warn し、`~/.config/zsh/.zshrc:34` と `:35` を名指しする(evidence としてレポートに記録)
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
- Edge cases: (1) コメント行 `# alias codex=…` は無視、(2) `alias codex='codex'`(フラグなし)は pass、(3) 同じ alias が複数ファイルにある場合は各 file:line を列挙、(4) rc が読めない(権限)場合は黙って飛ばす、(5) `ZDOTDIR` が相対パスや空
- Evidence to capture: `go test -count=1 -v` の出力、`docs/reports/test-*.md`

## Risks and mitigations

- 実 HOME を読む既存の doctor テストが環境依存になる → Check 自体は `HOME` / `ZDOTDIR` 経由でのみパスを決めるので、環境依存を避けたいテストは `t.Setenv("HOME", tmp)` で固定できる。既存テストは warn / pass どちらでも exit code が変わらない
- 誤検知(alias の値に `-m` が別の意味で含まれる)→ warn は助言で、Detail に該当行を示すので運用者が判断できる
- 見逃し(候補外の rc)→ Non-goals と Detail に明記

## Rollout or rollback notes

doctor の Check 追加のみ。下流へは次回 release でバイナリ経由、skill / recipe の文言は `ralph upgrade` / `ralph init` で配布。revert は PR 単位で安全。

## Open questions

なし。

## Deviation notes

- 2026-09-18 plan: Codex plan advisory(codex-cli 0.154.0、`</dev/null` 付き)が MEDIUM 1 件(`-mgpt-5.5` の連結形式を見逃す)を報告。採用し Scope 1・Assumptions・AC-2 を改訂。Slice B(文書)は advisory 待ちの間に先行して 9344c57 で実施

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
