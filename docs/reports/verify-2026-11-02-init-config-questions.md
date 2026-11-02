# Verify report: init-config-questions

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-init-config-questions.md`(issue #277、AC1〜AC10、D1〜D10)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1。`cycle-count.json` は `"cycle": 1`)
- Scope: `git diff origin/main...HEAD`(origin/main c8c8c0f1、merge-base も c8c8c0f1、HEAD 42d69da3、9 ファイル、+2235/−21)。コードは `internal/cli/init.go`、新しい `internal/cli/init_settings.go`、新しいテスト 3 本、`README.md` の `ralph init` の行。spec の準拠、静的解析、文書のずれを見た。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。「テストで固定」とある AC は、テストの中身を読んで当てている値を確かめた
- Evidence: `docs/evidence/verify-2026-11-02-init-config-questions.log`(run-static-verify の出力、base の確かめ、問いの文字列の禁止語の走査、implementer の pty の記録の抜き書き)。run-static-verify 自身のログは `docs/evidence/verify-2026-11-02-051513.log`。どちらも `.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 Both なら 5 問が聞かれ、答えが表のとおり書かれる | met | 1 つ目のフォームに CLI の問いの group を足す `internal/cli/init.go:165-177`、2 つ目のフォーム `:184-190`、`newSettingsFields` は Both なら 4 問 `internal/cli/init_settings.go:423-435`。書く値 `init_settings.go:156-188`(`[doctor]` の 2 つ、`[standard]` と `[org.review]` の `plan_advisory`・`cross_review`、`[org.gate].regate_checks`、`[org.director].auto_merge`)。テスト `init_settings_form_test.go:59`(Both で 4 問が順に出て答えが入る)、`init_settings_test.go:106`(48 通りを `config.Load` で読み、表の値を当てる)。通しの配線は implementer の pty の記録で見た(読んだだけ。下の Observational checks) |
| AC2 CLI が 1 つなら 2 問が出ず false、codex だけなら `watcher_enabled = false` | met | `bothOnly` の 2 問を飛ばす `init_settings.go:426`、値は `both && s.PlanReview` などで false `:174-177`、codex なら `watcher_enabled = false` `:184-186`。テスト `init_settings_form_test.go:98-100`(1 つで 4 つ答えを流しても 2 問しか読まない)、`init_settings_test.go:150-173`。pty の記録で codex の `ralph.toml` は `watcher_enabled = false`(238 行目)、claude の `model_pool` の行が `#` 付き |
| AC3 問いの title、description、選択肢に D3 の禁止語が出ない | met | 問いの文字列は `init_settings.go:297-407` の 27 個。evidence の 3 で D3 の正規表現を当て、当たりは 0。テスト `init_settings_form_test.go:309`(正規表現)、`:311`(各語を拾い、`gateway` などは拾わない)、`:344`(表の文字列と、描いた画面と accessible モードの出力の両方) |
| AC4 `--yes` は雛形と同じバイト列 | met | `runInitNonInteractive` は `cfg` をそのまま渡し(`init.go:320-322`)、`RunE` は `Settings` を設定しない(`init.go:81-89`)。`Settings` が nil なら `renderFS = baseFS`(`init.go:363-372`)。テスト `init_settings_write_test.go:161`(本物の雛形でバイト列が同じ、manifest は雛形のハッシュで `DiskHash` なし) |
| AC5 48 通りが `config.Load` で読め、CLI ごとに doctor が通る | met | `init_settings_test.go:106`(48 通り、答えない値は雛形のまま `reflect.DeepEqual`)、`init_settings_write_test.go:371`(CLI ごとに init し、その CLI と go の stub だけの PATH で `runDoctorOpts` がエラーなし、`  ⚠ ralph.toml: warn` の行なし、`model_pool` の件数の行)。doctor の warn の行の形は `internal/cli/doctor.go:115-120`・`:237` と合う |
| AC6 `ralph.toml` があり `--force` なしなら聞かず 1 行、変えない。`--force` なら聞いて書く | met | `askInitSettings` `init_settings.go:283-292`(Lstat で、ファイル・symlink・dangling・directory のどれでも聞かない)、`runInitInteractive` は false なら CLI の group も 2 つ目のフォームも作らず `Settings` は nil(`init.go:174-190`)。テスト `init_settings_form_test.go:408`(5 つの形 × `--force` の有無)、`init_settings_write_test.go:224`(なしなら変えず「not written」の行、ありなら答えで上書き)。pty の記録 `exist.log` は 1 行を出し、CLI の問いも yes/no も出ていない(S4 の文言の直しの前の binary) |
| AC7 init 直後の `upgrade --dry-run` に `ralph.toml` の advisory が出ない | met | `hashes[initSettingsFile] = settingsTemplateHash`(`init.go:398`)、`SetFileOwned(path, OwnerSeed, templateHash, settingsDiskHash)`(`:456-461`)。upgrade の seed は `TemplateHash` だけを比べる(`internal/upgrade/replaceplan.go:307-317`)。テスト `init_settings_write_test.go:332`(`advisories: 0 files`、`TemplateHash` を答えのハッシュにすると 1 件になる対照つき) |
| AC8 key か `model_pool` がないと何も書かずにエラー | met | 答えの計算は `MkdirAll` より前(`init.go:363-377`)。`initTOMLSet` は 0 行も 2 行以上もエラー(`init_settings.go:217-224`)、`model_pool` の欠け・`]` なし・選んだ CLI の行なしもエラー(`:252-275`)。テスト `init_settings_write_test.go:278`(対象のディレクトリが作られず、空の既存ディレクトリにも何も書かない)、`init_settings_test.go:271`・`:329`・`:348`・`:366`・`:396`・`:412` |
| AC9 help と README の `ralph init` の行 | met | help `init.go:41-54`、`README.md:127`。新しい問い、PATH による「(Recommended)」、Both のときだけの 2 問、`ralph.toml` があれば聞かないこと、`--force` は init の書くすべてのファイルを上書きすること、`--yes` は雛形の `ralph.toml` を書くことを、どちらもコードと同じに書いている |
| AC10 描き出しが失敗しても答えが残り、答えなしのやり直しのあとも残る | met | `initAnsweredFS.ReadFile` が `ralph.toml` だけ答えを返し(`init_settings.go:79-91`)、`scaffold.RenderFS` は `fs.ReadFile` で読む(`internal/scaffold/render.go:66`)。テスト `init_settings_write_test.go:428`(`scripts` を普通のファイルにして 1 回目を失敗させ、消してからやり直す。`TemplateHash` は雛形のハッシュ)、`:98`(包みの FS の読み方) |

Design decisions は D1〜D10 のどれもコードと合う。D2 の前提(huh v2.0.3 の accessible モードは hide を見ずに全部の field を聞く)は `charm.land/huh/v2@v2.0.3/form.go` の `runAccessible` で確かめた。計画の digest は `./scripts/plan-visual.sh digest` で `d1e86274a18e`、`- Approved:` の行と同じ。

issue #277 の受け入れ条件 5 つは、計画のとおり AC1〜AC5 に 1 対 1 で対応する(`gh issue view 277` で本文を読んだ)。issue の 4 つ目「`--yes` では既定値の `ralph.toml`」は AC4 の雛形のバイト列と同じ意味。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc=0、2 分 0 秒) | `All verifiers passed.`。shellcheck、`sh -n`、guard の awk、`jq -e`、Codex hook の検査 3 つ、check-sync(IDENTICAL 172、DRIFTED 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の plan の参照、golang(`gofmt: ok`、golangci-lint `0 issues.`。go vet と staticcheck は通れば何も出さない)、branch の secret scan(`c8c8c0f1..42d69da3`、clean)。時間切れはなく、1 回で済んだ |
| `go build ./...` | pass | 出力なし |
| `git diff --check origin/main...HEAD` | pass(rc=0) | 出力なし |
| `./scripts/plan-visual.sh digest <plan>` | `d1e86274a18e` | 承認時の digest と同じ |
| 問いの文字列の禁止語の走査(`python3 -I`) | 当たり 0 | evidence の 3。`init_settings.go:297-407` の 27 個の文字列 |
| base の確かめ | 同じ | `origin/main` と merge-base はどちらも c8c8c0f1。取り込みの試しは要らない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md:127`(`ralph init` の行) | yes | AC9 のとおり |
| `init` の help(`init.go:32-54`) | yes | AC9 のとおり |
| `AGENTS.md` の地図 | yes | `internal/cli/` の行(`:86`)はサブコマンドを並べ、ファイルは並べない。`init_settings.go` は init の一部なので足す必要はない |
| `README.md:134`(`ralph doctor` の行) | 一部ずれ(V1) | 「either Claude Code or Codex is usable なら通る」「`require_*` は既定で `false`」は config の既定としては正しいが、対話の init が選んだ CLI の `require_*` を `true` で書くことに触れていない |
| `docs/recipes/codex-setup.md:20-21`・`:34-37` と `templates/base/` の写し(`cmp` で同じ) | 一部ずれ(V1) | 「either one passes」「codex だけの machine では Claude Code の行は warn で doctor は通る」は、`--yes` か、init で Codex を選んだ project では正しい。Claude Code か Both を選んだ project では `require_claude_cli = true` なので fail する |
| `templates/base/docs/recipes/adding-a-language-pack.md:30-38` | yes | 言語パックの問いと `--yes` の説明で、今も正しい。後ろの問いには触れないが、pack の recipe なので要らない |
| `docs/specs/2026-10-07-org-multi-org-director.md:134`・`:139`(#264・#274 の日付の付いた注) | yes(Info) | 「`ralph init` のテンプレートは `local`・`auto_merge = true` を書く」は雛形と `--yes` について今も正しい。この系列(#264、#265、#274、#276)は spec に日付の付いた注を足してきたので、#277 で対話の init が `auto_merge` と `regate_checks` を聞くことを注に足すかは sync-docs で決める |
| `docs/specs/2026-04-16-ralph-cli-tool.md:28` | 対象外 | 初期の spec の init の問い(Ralph Loop、TUI)。歴史の記録なので触らない |
| `docs/tech-debt/README.md` | yes | init に触れる行(20、160、368)の Trigger はどれもこの変更で起きない(4 つ目の列を python で読んだ) |
| 計画の Verify plan | Info | 「Spec compliance criteria to confirm: AC1〜AC9」のままで、計画助言で足した AC10 が入っていない。この report では AC10 も見た |
| 計画の AC の checkbox | 済み | 実装のときに `[x]` が付いている。digest は変わっていない |

## Observational checks

- implementer の S3 の pty の記録(`impl-initq-s3/` の `both.log`、`codex.log`、`exist.log`、`t-both/ralph.toml`、`t-codex/ralph.toml`)を読んだ。打ち直してはいない。どれも S4 1af26382 の前の binary で、S4 は文言(`--force` の説明、`(Recommended)` の注)と関数名だけを変えたので、流れの確かめとしては今も使える
  - Both(PATH に claude と codex):CLI の問いは `3. Both (Recommended)`、そのあと yes/no が 4 問、`✓ ralph.toml (wrote your answers)`。書かれた値は `plan_advisory = true`(2 か所)、`cross_review = false`(2 か所)、`regate_checks = "ci"`、`auto_merge = false`、`require_*` は両方 true
  - Codex(PATH に codex だけ):`2. Codex (Recommended)` を Enter で選び、yes/no は 2 問。`driver_pool = ["codex"]`、claude の `model_pool` の 4 行が `#` 付き、`watcher_enabled = false`
  - `ralph.toml` がある場合:1 行を出し、CLI の問いも yes/no も出ていない
- 1 行の知らせ(`init_settings.go:290`)と「wrote your answers」の行(`init.go:402`)は `ralph.toml` の名前を出す。D3 の禁止語の `toml` に当たるが、AC3 の範囲は問いの文なので、ずれではない

## Findings

### V1(LOW): doctor の説明が、対話の init が `require_*` を書くことを言わない

対話の `ralph init` は、選んだ CLI の `[doctor].require_claude_cli`・`require_codex_cli` を `true` で書く(`init_settings.go:172-173`)。Both を選んだ project では、どちらかが使えなければ doctor は fail する(計画の D4 とリスクの表に書かれた、意図した動き)。`README.md:134` の doctor の行と `docs/recipes/codex-setup.md:20-21`・`:34-37`(と写し)は、どちらか 1 つで通ることと既定の `false` だけを書く。PATH で初期値を決めるので、Enter だけで進めた人がこの差に当たるのは、claude も codex もない machine で init したあとに片方だけ入れた場合と、選び直した場合に限られる。直す先は sync-docs で、たとえば README の doctor の行に「the interactive `ralph init` sets the one the project uses to `true`(both with Both)」を足し、recipe の 2 か所に「`--yes`、または init で Codex を選んだ project では」と条件を足す。

## Coverage gaps

- `runInitInteractive` の 2 つのフォームのつなぎ(`init.go:165-191`)を通しで流す自動のテストはない。huh のフォームが `os.Stdin` から読むためで、2 つのフォームはそれぞれ accessible モードのテストがある。通しは implementer の pty の記録(S4 の前の binary)だけ。いちばん小さい足し方は、`runInitInteractive` の 2 つの `Run()` を、テストから入力と出力を差し替えられる形(たとえば `func(*huh.Form) error` を `initConfig` か引数で渡す)にして、accessible モードで「CLI に 3、yes/no に 4 つ」を流し、`executeInit` に渡る `Settings` を当てるテストを 1 本足すこと
- テストが通るかは /test が確かめる。implementer が記録した変異(S1 14 個、S2 6 個、S3 8 個)が落ちることも /test の範囲
- 本物の `ralph doctor` を init 直後の project で打つことは、stub の PATH の `runDoctorOpts` のテストまで。実機の claude と codex での確かめはしていない
- 全画面(TERM が dumb でない)のフォームの表示は、implementer の `tui.log` があることを見ただけで、中身は読んでいない

## Insight event

`./scripts/insights-append.sh --slug init-config-questions --flow standard --phase verify --verdict pass --cycle auto --source skill --low 1` を打ち、`docs/insights/events/2026-11-02-init-config-questions.jsonl` に 1 行足した。この report と一緒にコミットする。

## Verdict

pass

- Verified: AC1〜AC10 と D1〜D10 は、コードの該当行とテストの中身で確かめた。AC1・AC2・AC6 の通しの流れは implementer の pty の記録でも見た。静的の検査はすべて通り、計画の digest は承認時の `d1e86274a18e` のまま
- Partially verified: AC5・AC7・AC10 など「テストで固定」とある部分は、テストが当てる値を読んで確かめた。結果は /test が見る
- Not verified: Go のテストの結果、`runInitInteractive` を通しで流す自動の確かめ、実機の CLI での doctor
- Findings: LOW 1 件(V1、doctor の説明。sync-docs で直す)。merge を止めない
