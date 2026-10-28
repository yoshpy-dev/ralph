# Verify report: doctor-either-agent-cli

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-doctor-either-agent-cli.md`(AC1〜AC7、D1〜D6、Verify plan)。issue #272 の受け入れ条件 5 つ
- Verifier: verifier サブエージェント(`/verify`、cycle 1)
- Scope: `git diff 15e301c8...HEAD`(HEAD d1a09d01、11 ファイル、+648/-84)。テストは打っていない(`/test` の範囲)。観察のため、worktree から `go build` した binary で scratchpad の project に `ralph doctor` を 7 通り打った。`go test` は打っていない
- Evidence: `docs/evidence/verify-2026-10-28-doctor-either-agent-cli.log`(gitignore。`run-static-verify.sh` の全出力、`git diff --check`・`gofmt`・`go vet`、doctor の probe の出力、文書の grep)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 既定(false)で claude だけ・codex だけの環境なら、claude と codex の行に fail がない。使えない側は warn で、相手があるので動くことと、その CLI を呼ぶ計画助言と cross-review が動かないことを書く | Met | 判定は `internal/cli/doctor.go:297-315` の `agentCLIResult`。使えない側の require が false で相手が使えるときは warn で、detail は `<cli> unusable (not required): <err> — <other> is usable, so ralph runs, but the plan advisory and cross-review that call <cli> will not run`。scaffold した project(HEAD の雛形の `ralph.toml`)で打つと、claude だけは `⚠ Codex: warn`、codex だけは `⚠ Claude Code CLI: warn` で、どちらも rc 0、`✗` は 0 行 |
| AC2 両方使えないとき、2 行とも fail で、どちらか 1 つが要ると書き、`runDoctorOpts` がエラーを返す(終了コード 1) | Met | `doctor.go:311-313` の default の枝。probe では 2 行とも `✗ … fail — <cli> unusable: … — neither claude nor codex is usable, and ralph needs at least one of them` で、rc 1、stderr の末尾は `doctor: 2 check(s) failed`。D1 のとおり |
| AC3 `require_claude_cli = true` で claude がなければ codex があっても fail。codex 側も同じ | Met | `doctor.go:308-310`(required の枝は相手より先に評価される)。probe で `require_codex_cli = true` と claude だけでは `✗ Codex: fail — codex unusable: …`、rc 1。`require_claude_cli = true` と codex だけでは `✗ Claude Code CLI: fail`、rc 1。どちらも fail は 1 行 |
| AC4 herdr がない・agmsg の home に `scripts/send.sh` がないとき warn で、標準フローに影響しないと書き、エラーにならない | Met | `doctor.go:780-790` と `:812-818`。probe の 7 通りすべてで `⚠ herdr: warn — herdr not installed — org runtime seats unavailable (the standard flow is unaffected)` と同じ形の agmsg の行が出た。両方あるときは rc 0 で `All checks passed.`。`countFailed`(`:222-230`)は fail だけを数える |
| AC5 `Default()` の 2 つが false、雛形が両方 false と明示し、`TestLoad_TemplateRalphToml` が読み込んだ値と `Default()` の一致を確かめる | Met | `internal/config/config.go:538-539`。`templates/base/ralph.toml:9` と `:12`。`internal/config/config_test.go:117-131` が `\nrequire_claude_cli = false\n` と `\nrequire_codex_cli = false\n` の文字列と、読み込んだ値と `Default()` の一致を見る。`ralph init --yes` が書いた `ralph.toml` の `[doctor]` も同じ(probe の冒頭) |
| AC6 doctor のテストが AC1〜AC4 の組み合わせを固定し、claude だけ・codex だけの統合のケースは雛形を使う | Met(テストの有無と中身を読んだ。実行は `/test`) | `internal/cli/doctor_agent_cli_test.go:20` の表は 16 件(有無 4 通り × require 4 通り)で、status と detail の句、含まない句を見る。`:189` の統合は 7 件で、`ralphToml` が nil のケース(両方あり・claude だけ・codex だけ・両方なし・壊れた shim)は `templateRalphToml`(`:146-158`、リポジトリの雛形)を写す。fail の行数(`✗ `)も見る。herdr と agmsg は `internal/cli/doctor_org_test.go:70`(warn と新しい文、`solo execution` がないこと)と `:198`(エラーなし、`herdr: warn`・`agmsg: warn`)。`go vet ./internal/cli/ ./internal/config/` はテストのファイルを含めて rc 0 |
| AC7 README の doctor の説明が AC1〜AC4 と合い、tech-debt の (h) が閉じている | Met | `README.md:60` と `:127` は下の Documentation drift のとおり実装と合う。`docs/tech-debt/README.md:197` の (h) は、Debt・Impact・Trigger の 3 列で取り消し線と closed の注記が付いた。Why deferred と Related の列は (h) を名指さない |

issue #272 の受け入れ条件との対応:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| 1. claude だけ・codex だけで fail を出さない(片方がないことは warn) | Met | AC1 |
| 2. 両方ないときは fail で、終了コードが 1 | Met | AC2(probe の rc 1) |
| 3. `require_*` を true にした側がないときは fail | Met | AC3 |
| 4. herdr と agmsg がないときに warn | Met | AC4 |
| 5. `internal/cli` の doctor のテストで組み合わせを固定 | Met(有無と中身。実行は `/test`) | AC6 |

計画の Non-goals に照らした確かめ:

- Check 8b(shell の alias)は `doctor.go:134` で `herdrResult.Status == "pass"` を渡す。herdr がないときの状態が warn になっても false のままで、判定は変わらない
- Check 11b の「agmsg がない」は `internal/cli/doctor_codex_writable_root.go:936` で info のまま。`TestCheckCodexAgmsgWritableRoot_AgmsgVersionMismatch_StillGraded`(`doctor_codex_writable_root_test.go:1038`)が前提にする「版の違いは info」も `doctor.go:826-830` で変わっていない。S3 の逸脱(このテストを直さなかったこと)は Progress に記録済みで、D6 と合う
- `--strict` は scaffold の検査(`scaffoldViolationStatus`)だけを fail にする。`--strict` の help(`doctor.go:39-52`)の「every other doctor check (e.g. missing claude/codex CLI) keeps its existing pass/warn/fail semantics」は今も正しい

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | 1 回で通った(52.6 秒、時間切れなし)。`templates/base/ralph.toml` が言語に分類できないので範囲は full に倒れ、pack は golang。`gofmt: ok`、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし(どちらも Skipping の行がなく、実行された)。local verifier の shellcheck、hook の `sh -n`(root と template の 10 本ずつ)、guard の awk、settings の `jq`、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画参照はすべて通った。secret scan は 15e301c8..d1a09d01 で clean。wrapper 自身のログは `docs/evidence/verify-2026-10-28-095926.log` |
| `git diff --check 15e301c8...HEAD` | rc 0 | 出力なし |
| `test -z "$(gofmt -l internal/cli internal/config)"` | rc 0 | |
| `go vet ./internal/cli/ ./internal/config/` | rc 0 | テストのファイルを含む。go1.27.3(go.mod と同じ) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md:60`(Quick start の `ralph doctor` の注記) | Yes | `environment check (Claude Code or Codex)`。どちらか 1 つで通る動きと合う |
| `README.md:127`(コマンド表の `ralph doctor`) | Yes | どちらかが使えれば通り、使えない側は warn、両方ないときと require を true にした側がないときは fail、既定は両方 false、herdr と agmsg がないときは warn で終了コードを変えない。`doctor.go:277-315`・`:780-818`・`:222-230`、`config.go:538-539` と合う。「the one that is not usable is `warn`」は require が false の側の話で、次の文が require が true の側を fail と書くので矛盾しない |
| `templates/base/ralph.toml:3-13`(`[doctor]` のコメント) | Yes | 規則の段落(`:4-8`)は実装と合う。`codex trust` の 2 行(`:10-11`)は `require_codex_cli` の直前に戻った。前の位置や前の文(「Codex is optional by default」「The accompanying `.codex/config.toml`」)を参照する文書・テストはない(`git grep` の rc 1)。テスト用の embed FS の `ralph.toml`(`internal/cli/cli_test.go:42` ほか 5 か所)は `require_codex_cli = false` だけを書く小さな fixture で、雛形のコメントを写していない |
| `git grep -n require_claude_cli` / `require_codex_cli` | Yes | 今の状態を書く文書で、claude を既定で必須と書くもの、片方がないと fail と書くものはない。残るヒットは計画・self-review の報告・テスト・雛形と、下の Info 1 の古い仕様 2 つ |
| `git grep -n 'solo execution'` と herdr・agmsg の `info` の文 | Yes | `internal/` に残るのは、それを使わないことを確かめるテストの文字列(`doctor_org_test.go:69`、`:93-94`)だけ。ほかのヒットは `docs/evidence/` の 2026-10-13 の出力、過去の報告、計画の本文で、すべて履歴。`.claude/skills/org/SKILL.md:142` と `docs/recipes/codex-seat-permissions.md:38` の「herdr 未導入なら info」は Check 8b(alias)の話で、Non-goals のとおり変わっていない |
| テストの改名(`TestLoad_RequireCodexCLI` → `TestLoad_RequireAgentCLIs`、`AbsentIsInfo` → `AbsentIsWarn`)と消した関数(`checkClaudeCLI`・`checkCodexCLI`) | Yes | 旧名のヒットは `docs/evidence/` の過去のテスト出力、self-review の報告(Run 1 の指摘の引用)、計画(置き換える対象として名指す)だけ。今の状態として旧名を書く文書・計画はない |
| `docs/tech-debt/README.md:197` の (h) | Yes | Debt の列は `~~(h) V-3: …~~ Closed in feat/doctor-either-agent-cli (2026-10-28): the not-installed detail now says the standard flow is unaffected, and its status is warn (issue #272).`。(g) と同じ形で、中身は `doctor.go:784` と `:816` の文と status に合う。Impact は `~~(h) …~~ (closed)`、Trigger は `~~the next change to … ((h): "standard flow unaffected")~~ done in feat/doctor-either-agent-cli` |
| `docs/recipes/codex-setup.md:20`、`.codex/README.md:29`、`.codex/AGENTS.override.md:59` | Yes | 「Claude Code/Codex presence を確かめる」「codex が `$PATH` にあるかを確かめる」は検査の対象を書く文で、fail かどうかは書かない。今も正しい |
| `.claude/skills/org/SKILL.md:24-25`(前提)と写し | Yes | 「`ralph doctor` で herdr / agmsg のチェックを確認する(org を使わない標準フローは両ツールなしで動く)」。status を名指さず、新しい detail の文と同じことを言う |
| `docs/recipes/codex-seat-permissions.md:16`、`docs/recipes/isolated-herdr-server.md:447` | Yes | herdr と agmsg がある前提で「pass する」と書く。変わらない |
| ルートの `ralph.toml` | 該当なし | `git ls-files '*ralph.toml'` は `templates/base/ralph.toml` だけ。`check-sync.sh` は `TEMPLATE_ONLY ralph.toml` と出し、PASS(計画の Assumptions のとおり、#264・#265 が作る) |

Info(直さなくてよい):

1. `docs/specs/2026-04-16-ralph-cli-tool.md:249` と `docs/specs/2026-05-07-codex-cli-parity.md:13` は `require_claude_cli = true` を書く。前者は退役した `[pipeline]` の鍵を含む設定の例、後者は 2026-05-07 の「現状」の節で、どちらもその日の記録。計画の Verify plan も `docs/specs` を履歴として除く
2. `docs/specs/2026-08-01-org-runtime.md:89` の非機能要件(`[ ]`)は「座席 0 のソロ実行のみ両ツールなしで動作」と退役した Solo の言い方を残す。status は書かず、この PR の前からある文で、計画の範囲の外
3. `docs/tech-debt/README.md:197` の Trigger の列の日付つきの注記(feat/reviewer-runs-gate の 2026-10-26 の注記など)は「The open items (a) to (e), (h), (i), (k), and (l) …」と (h) を開いた項目に数える。その日の時点では正しい履歴の文
4. warn の detail と README の「the plan advisory and cross-review that call claude will not run」: 今の計画助言は codex だけを呼ぶ(`.claude/skills/plan/SKILL.md:73-79`)ので、claude がないときに止まるのは reviewer が claude の cross-review だけ。「that call claude」で絞っているので文は偽ではない。D4 は #265・#266 で呼び先が変わることを見越して CLI を主語にすると決めている
5. warn の行があっても要約の行は `All checks passed.`(`doctor.go:215`)。前からの動きだが、herdr と agmsg を入れていない project では warn 2 行のあとにこの行が出るようになる。計画の Risks に「warn が増えて騒がしくなる」として載っている
6. 計画の AC1〜AC7 のチェックボックスはすべて `[ ]` で、Progress の「Review artifact created」も未チェック(self-review の報告はある)。チェックボックスの遅れで、`/sync-docs` か `/pr` で付ければよい

## Observational checks

`go build -o <scratch>/ralph ./cmd/ralph`(worktree の HEAD)、`ralph init --yes` で scaffold した project、stub の `claude`(`2.1.0 (Claude Code)`)と `codex`(`codex-cli 0.154.0`)、PATH は `<stub のディレクトリ>:/usr/bin:/bin`(herdr・gh・go はない)、`RALPH_ORG_AGMSG_HOME` はないディレクトリ。各ケースで `ralph.toml` の `ralph.toml:` 警告の行は出ていない(fixture の壊れはない)。

| ケース | rc | Claude Code CLI | Codex | fail の行 |
| --- | --- | --- | --- | --- |
| 両方あり | 0 | pass | pass | 0 |
| claude だけ | 0 | pass | warn(`codex unusable (not required): … claude is usable, so ralph runs, but the plan advisory and cross-review that call codex will not run`) | 0 |
| codex だけ | 0 | warn(同じ形で claude 側) | pass | 0 |
| 両方なし | 1 | fail(`… neither claude nor codex is usable, and ralph needs at least one of them`) | fail(同じ) | 2 |
| claude だけ、`require_codex_cli = true` | 1 | pass | fail(`codex unusable: codex not found in PATH: …`) | 1 |
| codex だけ、`require_claude_cli = true` | 1 | fail(`claude unusable: claude not found in PATH: …`) | pass | 1 |
| codex と壊れた claude の shim(`--version` が exit 1) | 0 | warn(`claude unusable (not required): claude --version failed: exit status 1 — …`) | pass | 0 |

herdr と agmsg の行は 7 ケースすべてで warn と `(the standard flow is unaffected)`。fail の行は、どのケースでも claude と codex の行だけだった。

## Coverage gaps

- テストの実行(`internal/cli` の doctor のテスト、`internal/config`、`internal/scaffold`、`./scripts/run-test.sh`)は `/test` の範囲で、ここでは打っていない。AC6 はテストの有無と assert の中身を読んだだけ
- 既存の `runDoctor*` を呼ぶテスト(テストのファイル 7 つで 22 か所)がすべて claude・codex の stub を置いているという計画の Assumptions は確かめていない。claude だけを外して required の fail を期待するテストがあれば落ちる。`/test` のパッケージ全体の実行で分かる
- 本物の claude と codex の CLI での doctor は打っていない(stub だけ)。`probeBinary` は今回変わっていない
- Codex の driver で cross-review が `claude -p` を呼ばずに止まることは、skill の文(`.claude/skills/cross-review/SKILL.md:50`)を読んだだけ

## Verdict

- Verified: AC1〜AC5 と AC7(コードを読み、ビルドした binary の doctor で 7 通りを観察)。静的解析(`run-static-verify.sh` rc 0、`git diff --check`・`gofmt`・`go vet` rc 0)。文書の同期(README の 2 か所、雛形のコメント、tech-debt の (h)、recipe と skill の doctor の記述)
- Partially verified: AC6(テストの有無と中身は確かめた。実行は `/test`)
- Not verified: テストの実行結果、既存の `runDoctor*` のテストへの影響(`/test`)
- Verdict: pass
