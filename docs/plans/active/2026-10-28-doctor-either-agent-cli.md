# doctor-either-agent-cli

- Status: Approved
- Approved: 2026-10-28 sha256:0a45c27ac33e
- Owner: Claude Code
- Date: 2026-10-28
- Related request: doctor は claude と codex のどちらかがあれば通し、herdr と agmsg がないときは warn にする(issue #272。2026-10-27 にユーザーと決めた方針: 両方ないときだけ fail、片方だけないときは warn で相手のモデルの計画助言と cross-review が動かないことを出す、`require_*` は残して既定を両方 false、herdr と agmsg がないときは info から warn)
- Related issue: #272
- Type: feat
- Branch: feat/doctor-either-agent-cli

## Objective

`ralph doctor` を、claude と codex のどちらか 1 つがあれば通るようにする。今は claude と codex を別々に検査し、`[doctor].require_claude_cli` の既定が true なので、codex だけを入れた project では doctor が失敗する。herdr と agmsg がないときは info で 1 行出すだけで、気づきにくい。

この計画のあとは、claude と codex の両方を probe してから 2 つの行の状態を決める。両方とも使えないときだけ 2 行とも fail にし、「claude か codex のどちらかが要る」と書く。片方だけ使えないときは、その行を warn にし、その CLI を呼ぶ計画助言と cross-review が動かないことを書く。`require_*` を true にした側は、今までどおり使えなければ fail にする。`require_claude_cli` の既定を false にする。herdr と agmsg がないときは warn にする。warn は doctor の終了コードを変えない(`countFailed` は fail だけを数える)。

## Scope

- S1(既定値): `internal/config/config.go` の `Default()` の `RequireClaudeCLI` を false にする。`templates/base/ralph.toml` の `require_claude_cli` を false にし、`[doctor]` のコメントを、どちらか 1 つがあれば通ること、true にした側はなければ fail になることに書き直す。`internal/config/config_test.go` の既定値のテスト(`TestDefault`、`TestLoad_MissingFile`)を直す。`TestLoad_TemplateRalphToml` に、雛形が `require_claude_cli = false` と `require_codex_cli = false` を明示して書いていることと、読み込んだ値が `Default()` と同じことの確かめを足す(Codex の計画助言 1。既存の auto_merge と wake の確かめと同じ形)。
- S2(claude と codex の判定): `internal/cli/doctor.go`
  - `checkClaudeCLI` と `checkCodexCLI` を、両方を `probeBinary` で 1 回ずつ probe してから 2 つの `checkResult` を返す `checkAgentCLIs(cfg)` に置き換える。判定は probe の結果(エラーの有無)と `cfg.Doctor` だけを読む関数に分け、表で試せるようにする(D2)。
  - 判定の規則(D1・D3・D4):
    - 使える側は今までどおり pass で、detail は版の行。
    - 使えない側の `require_*` が true なら fail。detail は今の文(`claude unusable: <err>`)のまま。
    - 使えない側の `require_*` が false で、もう片方が使えるなら warn。detail は今の文(`claude unusable (not required): <err>`)に、相手の CLI があるので ralph は動くこと、その CLI を呼ぶ計画助言と cross-review が動かないことを足す。
    - 使えない側の `require_*` が false で、もう片方も使えないなら fail。detail に、claude と codex のどちらも使えず、どちらか 1 つが要ることを書く。
  - 行の名前(`Claude Code CLI` と `Codex`)と並び(Check 1 と 2)は変えない。
  - テストは新しいファイル `internal/cli/doctor_agent_cli_test.go` に書く。判定の表(claude と codex の有無 4 通り × `require_*` の組み合わせ)と、`runDoctorOpts` を stub の binary で通して終了コード(エラーの有無)を見るテスト。claude だけ・codex だけの統合のケースは、`executeInit` のあとに、リポジトリの `templates/base/ralph.toml` をそのまま project の `ralph.toml` に写して通す(テスト用の embed FS の `ralph.toml` は `require_codex_cli` しか書かないため。Codex の計画助言 1)。`require_*` を true にするケースは、その値を明示した `ralph.toml` を書く。
- S3(herdr と agmsg): `internal/cli/doctor.go` の `checkHerdrAvailable` と `checkAgmsgAvailable` で、ないときの状態を info から warn にする。detail の末尾の `(solo execution unaffected)` を、標準フローに影響しないという文に直す(D6)。関数の doc comment の「info なので終了コードを変えない」を、warn も終了コードを変えないことに書き直す。agmsg の版が試した版と違うときの info は変えない。既存のテスト(`internal/cli/doctor_org_test.go` の AbsentIsInfo、PathBootstrapperNotHome、HerdrAgmsgAbsent_ExitCodeUnaffected、`internal/cli/doctor_codex_writable_root_test.go` の `checkAgmsgAvailable` の前提の確かめ)を warn と新しい文に直す。`docs/tech-debt/README.md` の行の (h)(V-3: この detail の文が退役した Solo の語のまま)を、(g) と同じ形(取り消し線と `Closed in feat/doctor-either-agent-cli (2026-10-28)`)で閉じる。Impact と Trigger の列の (h) も同じ形で閉じる。
- S4(文書): `README.md` の Quick start の `ralph doctor` の注記(`environment check (Claude + Codex)`)と、コマンド表の `ralph doctor` の行に、claude か codex のどちらかで通ること、片方だけないときと herdr・agmsg がないときは warn になることを足す。

## Non-goals

- 既存の project の `ralph.toml` を書き換えること。`ralph.toml` は seed なので `ralph upgrade` は上書きしない。既存の project に残る `require_claude_cli = true` は、今までどおり claude がなければ fail にする(release ノートに書く)
- `ralph init` で使う CLI を聞いて `require_*` に書くこと(#277)
- 計画助言を相手のモデルにすること(#266)、計画助言と cross-review の実施有無の設定(#265)
- Check 11b(codex の sandbox と agmsg)の「agmsg がない」ときの info。同じことを 2 行で warn しないため変えない
- Check 8b(shell の alias)の判定。herdr の行が pass かどうかで決めており(`internal/cli/doctor.go` の `herdrResult.Status == "pass"`)、herdr がないときの状態が warn になっても変わらない
- `--strict` の扱い(scaffold の検査だけを fail にする今のまま)
- tech-debt の同じ行の (h) 以外の項目

## Assumptions

- `ralph` の終了コードは、doctor がエラーを返せば 1(`internal/cli/exitcode.go` の `ExitCode` の default)。doctor は fail の行が 1 つ以上あればエラーを返す
- 既存の `runDoctor*` を呼ぶテスト(20 か所)は、どれも claude・codex・go の stub を PATH に置いているので、判定の変更で結果は変わらない(consult で確認)
- このリポジトリにはルートの `ralph.toml` がまだない(#264 と #265 が作る)。この計画では作らない

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`
- `templates/base/ralph.toml`
- `internal/cli/doctor.go`(Check 1・2・8・9)
- `internal/cli/doctor_agent_cli_test.go`(新規)
- `internal/cli/doctor_org_test.go`、`internal/cli/doctor_codex_writable_root_test.go`(info を warn に)
- `docs/tech-debt/README.md`(行の (h) を閉じる)
- `README.md`

## Visual review

- Page: `.harness/state/plan-visual/doctor-either-agent-cli.html`(かんたんな図 2 つ: codex だけでも通ること、herdr と agmsg の注意。折りたたみの詳細 2 つ: 触るファイルとスライス、4 通りの組み合わせの判定)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: 全体・overview・easy-herdr・detail-overview・detail-judge の PNG を読んだ。detail-judge の 2 列目の文字が欄の端に触れていたので列の幅を広げ、撮り直して確かめた

## Design decisions

- D1(両方ないときの出し方): `Claude Code CLI` と `Codex` の 2 行をどちらも fail にし、どちらの detail にも、claude と codex のどちらも使えず、どちらか 1 つが要ることを書く。3 つ目の行は足さない。理由: 行の名前と数が変わらない。片方を入れれば、その行が pass に、もう片方が warn に変わるので、次に何をすればよいかが行から読める。代案の「2 行を warn のままにして 3 つ目の行を fail にする」は、両方ないときにだけ現れる行が増え、warn の行の「相手の CLI があるので動く」が成り立たない。fail は 2 件と数えられ、エラーの文は `doctor: 2 check(s) failed` になる(この文を固定したテストはない)。
- D2(判定を 1 か所にまとめる): 片方の判定にもう片方の probe の結果が要るので、2 つの関数を、両方を probe してから決める 1 つの関数にする。判定は probe の結果と `cfg.Doctor` だけを読む関数に分け、stub の binary なしに表で試せるようにする。probe は今と同じく各 1 回で、`--version` を打つ回数は変わらない。
- D3(`require_*` が true の側): 相手の有無によらず、使えなければ fail。detail は今の文のまま。両方ないときに required でない側は D1 の文で fail になる。
- D4(warn の文): 片方だけ使えないときの detail は、今の `<cli> unusable (not required): <err>` に「`<相手>` があるので ralph は動く。`<cli>` を呼ぶ計画助言と cross-review は動かない」を足す。issue の「相手のモデルを使う計画助言と cross-review が動かないことを出す」に沿う。主語は CLI にし、skill の名前と driver の向きは書かない。どの skill が相手の CLI を呼ぶかは #265 と #266 で変わるため。
- D5(既定値): `require_claude_cli` と `require_codex_cli` の既定を両方 false にする(`Default()` と `templates/base/ralph.toml`)。既存の project の `ralph.toml` は seed で upgrade が触らないため、`require_claude_cli = true` を書いた project は今の動きのまま。これは release ノートに書く。既定値を S1 で先に変えるのは、S2 の統合テストが雛形の値で claude だけ・codex だけを通すため(consult の所見 1)。
- D6(herdr と agmsg): ないときを warn にし、detail の末尾を標準フローに影響しないという文に直す。tech-debt の (h) は「次に `checkHerdrAvailable` か `checkAgmsgAvailable` を変えるとき」を trigger にしており、S3 がそれに当たるので閉じる(consult の所見 2)。agmsg の版の違いの info は、版の違いで動かなくなるとは限らないので info のままにする。Check 11b の「agmsg がない」は Check 9 と同じことなので info のまま。
- Critical forks: None(D1 は S2 の中で直せる表示の選び方で、どちらを選んでも 1 スライスを超える手戻りにならない)

## Acceptance criteria

- [ ] AC1: `require_*` が既定(false)のとき、claude だけが使える環境と codex だけが使える環境で、doctor の claude と codex の行に fail がない。使えない側の行は warn で、相手の CLI があるので動くことと、その CLI を呼ぶ計画助言と cross-review が動かないことを書く
- [ ] AC2: claude と codex の両方が使えないとき、2 行とも fail で、どちらか 1 つが要ることを書き、`runDoctorOpts` がエラーを返す(`ralph doctor` の終了コードは 1)
- [ ] AC3: `require_claude_cli = true` で claude が使えないとき、codex が使えても claude の行は fail。`require_codex_cli = true` で codex が使えないとき、claude が使えても codex の行は fail
- [ ] AC4: herdr がないとき herdr の行が warn、agmsg の home に `scripts/send.sh` がないとき agmsg の行が warn で、detail は標準フローに影響しないことを書き、どちらも `runDoctorOpts` のエラーにならない
- [ ] AC5: `config.Default()` の `require_claude_cli` と `require_codex_cli` が両方 false。`templates/base/ralph.toml` が両方を false と明示して書き、読み込んだ値が `Default()` と同じことを `TestLoad_TemplateRalphToml` が確かめる
- [ ] AC6: `internal/cli` の doctor のテストが、AC1〜AC4 の組み合わせ(claude と codex の有無 4 通り、`require_*` の各側、herdr と agmsg がないとき)を固定している。claude だけ・codex だけの統合のケースは、リポジトリの `templates/base/ralph.toml` を使う
- [ ] AC7: `README.md` の doctor の説明が、AC1〜AC4 の動きと合っている。tech-debt の (h) が閉じている

## Implementation outline

1. S1: `require_claude_cli` の既定を false にし、雛形のコメントと config のテストを直す(AC5) — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ ./internal/scaffold/`
2. S2: claude と codex の判定を `checkAgentCLIs` にまとめ、判定の表と `runDoctorOpts` のテストを足す(AC1・AC2・AC3・AC6) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Doctor|AgentCLI|ProbeBinary'`
3. S3: herdr と agmsg がないときを warn にして detail の文を直し、既存のテストを直し、tech-debt の (h) を閉じる(AC4・AC6・AC7) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Doctor|Herdr|Agmsg|WritableRoot'`
4. S4: `README.md` の doctor の説明を直す(AC7) — verify: `grep -n "ralph doctor" README.md`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、check-sync、check-skill-sync を含む)
- Spec compliance criteria to confirm: AC1〜AC7。issue #272 の受け入れ条件 5 つとの対応(AC1↔1、AC2↔2、AC3↔3、AC4↔4、AC6↔5)
- Documentation drift to check: `README.md` の Quick start とコマンド表、`templates/base/ralph.toml` の `[doctor]` のコメント、`docs/recipes/codex-setup.md` の doctor の説明、`.claude/skills/org/SKILL.md` の「前提」(herdr と agmsg を doctor で確かめる)。`git grep -n "require_claude_cli"` と `git grep -n "solo execution unaffected"` で、古い既定と古い文が残っていないか(`docs/plans/archive`・`docs/specs`・`docs/evidence`・`docs/reports` の履歴は除く)
- Evidence to capture: スライスごとの verify の出力、`go run ./cmd/ralph doctor` を PATH を変えて打った出力(claude だけ、codex だけ、両方なし)

## Test plan

- Unit tests: 判定の関数の表(claude と codex の有無 4 通り × `require_claude_cli`・`require_codex_cli` の true/false)。各行の status と detail の要点(「unusable (not required)」「どちらか 1 つが要る」「計画助言と cross-review」)を見る
- Integration tests: `runDoctorOpts` を、`executeInit` で作った project と stub の binary の PATH で通す。claude だけ・codex だけ(リポジトリの雛形の `ralph.toml` で、エラーなし、ない側が warn)、両方なし(エラーあり、2 行とも fail)、`require_codex_cli = true` を書いた `ralph.toml` で codex なし(エラーあり)
- Regression tests: 既存の doctor のテスト(`./internal/cli/` 全体)と `./internal/config/`、`./internal/scaffold/`。/test で `./scripts/run-test.sh` を通す
- Edge cases: claude が PATH にあるが `--version` が失敗する壊れた shim(使えない側として扱う。今の `probeBinary` の振る舞い)。両方とも required で両方ない(2 行とも fail、required の文)。herdr がないときも Check 8b(alias)は herdr を pass と見なさない
- Evidence to capture: `go test` の出力、`run-test.sh` の結果

## Risks and mitigations

- 既存のテストで、claude と codex のどちらも PATH に置かずに `runDoctor` のエラーなしを期待しているものが落ちる → consult が 20 か所すべて stub を置いていると確かめた。S2 の verify で doctor のテストを走らせ、/test でパッケージ全体を通す
- 既存の project で `require_claude_cli = true` のままの人には動きが変わらない → 意図どおり(D5)。release ノートに、どちらか 1 つで通したいときは false にすることを書く
- 雛形の値を戻しても、テスト用の embed FS がそれを読まないため気づかない → S1 の `TestLoad_TemplateRalphToml` と S2 の統合テストがリポジトリの雛形を読む(Codex の計画助言 1)
- warn が増えて doctor の出力が騒がしくなる(herdr と agmsg を使わない project で warn が 2 行出る) → issue の方針どおり。detail に「標準フローには影響しない」を書く
- 並行するセッションが同じファイル(README、`templates/base/ralph.toml`、`docs/tech-debt/README.md`)を触る → merge のときに両方の変更を残して直す

## Rollout or rollback notes

- 新しい binary から効く。既存の project の `ralph.toml` は upgrade で変わらない
- release ノートに 2 行: 「doctor は claude か codex のどちらかがあれば通る。`require_claude_cli` の既定は false になった(既存の `ralph.toml` に true があればそのまま fail する)」「herdr と agmsg がないときは warn になった(終了コードは変わらない)」
- 戻すときは PR を revert する。設定の形は変えないので、データの移行はない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(consult: 判定「直してから進める」。S1 の統合テストが既定値の変更より前に走る点と、tech-debt の (h) の trigger に当たる点を直した。Codex の計画助言: MEDIUM 1 件(雛形の値をテストが読まない)を反映し、既定値のスライスを先に回した)
- [x] Plan approved(自動承認(ユーザーの指示)。2026-10-28、digest sha256:0a45c27ac33e)
- [x] Branch created
- [x] Implementation started
  - S1(a6ad522e、implementer): 既定値を両方 false、雛形のコメント、`TestLoad_TemplateRalphToml` の確かめ。verify rc=0(`internal/config` ok、`internal/scaffold` ok)。雛形と `Default()` を戻すとテストが落ちることを写しで確かめた。self-check 5 項目 clean
  - S2(c9148ae9、implementer): `checkAgentCLIs` と純関数 `agentCLIResults`、`internal/cli/doctor_agent_cli_test.go`(判定の表と統合 7 件。claude だけ・codex だけ・両方あり・壊れた shim は雛形の `ralph.toml` を写す)。verify rc=0(100s)。mutation 4 種がすべて落ちる。self-check: 誤字 1 件 fixed(テストの doc comment)、ほか clean
  - S3(17ba114b、implementer): herdr と agmsg がないときを warn、detail を「(the standard flow is unaffected)」に、tech-debt の (h) を閉じた。verify rc=0(66s)、`git grep "solo execution unaffected" -- internal/` はヒットなし。self-check 5 項目 clean
    - 計画からの逸脱: Scope S3 は `internal/cli/doctor_codex_writable_root_test.go` の `checkAgmsgAvailable` の確かめを warn に直すと書いたが、その確かめ(`TestCheckCodexAgmsgWritableRoot_AgmsgVersionMismatch_StillGraded`)は agmsg の版が違うときの前提で、D6 どおり info のままが正しいので変えなかった
    - 範囲内の追加: `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` の「required-by-default」のコメント(S1 で古くなった)と、消した `checkClaudeCLI` を名指ししていたコメントを直した
  - S4(d171e3e6、inline: README の 2 か所だけの変更): verify `grep -n "ralph doctor" README.md` で 60 行と 127 行の新しい文を確かめた。self-check: 不要な変更 clean、誤字とコピペ clean、デバッグの残り clean、秘密情報 clean、範囲外のファイル clean(stage は README.md だけ)
  - 別のセッション(stoic-app)から、この Mac で 90 分の local CI を流すので重い処理を始めないよう頼まれた(2026-10-28 17:2x)。/verify と /test は「再開してよい」の連絡のあとに回す
  - S5(追加のスライス、implementer): /test(1b9798b1)の mutant P9 で、Test plan の Edge cases の「herdr がないときも Check 8b(alias)は herdr を pass と見なさない」を固めるテストがないと分かった。`doctor.go:134` の `== "pass"` を `!= "info"` に変えても `internal/cli` 全体が通る(herdr がないときが warn になったため、この比べ方が初めて結果を左右する)。`internal/cli/doctor_shell_alias_test.go` に herdr の stub を置かない `runDoctorOpts` のテストを 1 本足す。consult(consult-plan-p9)の判定は A「cross-review の前に足し、/self-review → /verify → /test を cycle 1 のまま全部回し直す」。cycle を増やすのは cross-review の ACTION_REQUIRED のあとだけなので、cycle-count.json は 1 のまま — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Doctor|Herdr|Agmsg|ShellAlias'`
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
