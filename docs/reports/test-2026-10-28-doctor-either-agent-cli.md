# Test report: doctor-either-agent-cli

- Date: 2026-10-28(JST 19:09〜19:55)
- Plan: `docs/plans/active/2026-10-28-doctor-either-agent-cli.md`(Test plan、AC1〜AC7)
- Tester: tester サブエージェント(Claude Opus 5.5)
- Scope: HEAD 322bfe18(base 15e301c8)。behavioral test だけを実行した(静的解析は /verify の担当で、verify の報告は pass)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、`go env GOFLAGS` は空。ほかのセッションと CPU を分けており、1 分の load average は始まりが 7.4、`run-test.sh` の終わりが 111.0、最後の実行の終わりが 74.4
- Evidence: `docs/evidence/test-2026-10-28-doctor-either-agent-cli.log`(`run-test.sh` の出力すべて、名指しの実行、coverage、mutant のスクリプトと落ちたテストの行、組み立てた binary の doctor の出力)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告のほかは変えていない。mutant は `git archive HEAD` の scratchpad の写しで試し、試すたびに元へ戻して `cmp` で HEAD と同じことを確かめた。`run-test.sh` のあとの `git status --porcelain` は空だった

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh`(既定の changed。`templates/base/ralph.toml` が言語に分類されないので full に広がり、pack は golang だけ) | 下の 1a と 1b | すべて | 0 | 1 件(下) | rc 0、1,491 秒(19:09:37〜19:34:28) |
| 1a | 1 の shell の部分(`verify.local.sh` の test モード、`tests/test-*.sh`) | 45 本 | 45 | 0 | 1 件 | 1 に含む |
| 1b | 1 の Go の部分(`go test ./...`) | テストのある 9 パッケージ | 9 | 0 | - | `internal/cli` 407.3 秒、`internal/org` 575.7 秒、ほかの 7 つは 0.97〜5.14 秒。`(cached)` はなし |
| 2 | `go test ./internal/config/ -count=1` | 1 パッケージ | 1 | 0 | - | rc 0、0.49 秒 |
| 2a | 同じパッケージで `-run 'TestDefault$\|TestLoad_TemplateRalphToml\|TestLoad_RequireAgentCLIs\|TestLoad_MissingFile' -v` | 4 | 4 | 0 | 0 | rc 0 |
| 3 | `go test ./internal/cli/ -run 'AgentCLI\|Doctor\|HerdrAgmsg' -count=1`(`-v` を足して数えた) | トップレベル 19、サブテストを含めて 45 | 45 | 0 | 0 | rc 0、27 秒(パッケージ 25.7 秒) |
| 4 | 雛形と本体の mutant(対照 1、雛形 3、本体 9。下の「雛形の写し」と「本体の mutant」) | 実行ごとに名指しの組 | 対照は通過 | mutant 12 のうち 11 がどれかのテストで落ちた | 0 | 1 回 1〜28 秒。P9 のパッケージ全体は 1,233.6 秒 |
| 5 | 組み立てた `ralph` で `ralph init --yes` のあと `env -i` で PATH を狭めて `ralph doctor`(5 通り。下の「binary での確かめ」) | 5 の実行 | 期待どおり 5 | - | - | 1 回 1 秒未満 |

- 1a の 45 本は `tests/test-*.sh` の数と同じで、どの本も `OK` で終わった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.50.1 なので「2.41 より古い本物の git」の事例を飛ばし、stub の古い git がその分岐を見る)
- 3 の 45 本の内訳は、`TestAgentCLIResults_EveryCombination` の 16 通り、`TestRunDoctorOpts_AgentCLIs` の 7 件、`TestRunDoctorFull_PrintsTheMergeGuarantee` の 3 件、トップレベル 19 本。3 の `-run` に当たらない `TestCheckAgmsgAvailable_PathBootstrapperNotHome` と `TestCheckScaffoldIntegrity_*` の `runDoctorFull` の呼び出しは、1b のパッケージ全体で通った
- 打ち直しはない。30 分の timeout による失敗も、`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の失敗も起きなかった

## stub の前提(計画の Assumptions)

計画は「既存の `runDoctor*` の呼び出しは、どれも claude と codex の stub を PATH に置いている」としている。テストのファイルの `runDoctor`、`runDoctorOpts`、`runDoctorFull` の呼び出しは 7 ファイルに 22 か所あり、新しい `doctor_agent_cli_test.go` の 1 か所を除く 21 か所(6 ファイル、16 のテスト)をすべて読んだ。

| ファイル | 呼び出し | PATH に置く stub |
| --- | --- | --- |
| `doctor_codex_writable_root_test.go` | 3(1796、1859、1929) | claude、codex、go |
| `doctor_shell_alias_test.go` | 2(1494、1545) | claude、codex、go(1545 は herdr も) |
| `doctor_merge_guarantee_test.go` | 2(414、455) | `doctorMGProject` が claude、codex、go(455 は gh と git も) |
| `doctor_scaffold_test.go` | 10(5 つのテストが strict なしと strict ありで 2 回ずつ) | claude、codex、go |
| `doctor_org_test.go` | 3(226、268、1237) | claude、codex、go(268 は 1 つずつ書き、claude に呼び出しの記録を付ける) |
| `cli_test.go` | 1(345) | 自前の `echo` の script で claude、codex、go |

- 21 か所のどれも claude と codex の両方を置いている。片方だけを置く呼び出しはないので、新しい規則で結果の変わるテストはなく、意図の弱まったテストもない
- 変わったテストのファイルは `doctor_agent_cli_test.go`(新規)、`doctor_org_test.go`、`config_test.go` の 3 つで、消えたテストはない。`TestLoad_RequireCodexCLI` は `TestLoad_RequireAgentCLIs` に変わり、`require_claude_cli = false` の読み込みの代わりに `true` の読み込みを見る。新しい既定が false なので、明示した false の読み込みは既定と区別できない。雛形が明示する false は `TestLoad_TemplateRalphToml` が見る
- `cli_test.go:345` の `TestRunDoctor_Passes` は `_ = runDoctor(dir)` で結果を捨てる。前からの形で、このブランチでは変わっていない(panic しないことだけを見る)

## 雛形の写し(AC5・AC6)

`TestRunDoctorOpts_AgentCLIs` の 7 件のうち、both usable、claude only、codex only、neither、broken claude shim の 5 件は `templateRalphToml(t)`(`internal/cli/doctor_agent_cli_test.go:150`)でリポジトリの `templates/base/ralph.toml` をディスクから読み、`executeInit` のあとに project の `ralph.toml` へ写す。`require_*` を true にする 2 件は `[doctor]` の 1 行を書いた `ralph.toml` を使う。計画(S2 と進捗の S2)とテストの doc comment のとおり。写しの `templates/base/ralph.toml` を変えて確かめた。

| mutant | 変更 | `TestRunDoctorOpts_AgentCLIs` | `TestLoad_TemplateRalphToml` |
| --- | --- | --- | --- |
| 対照 | 変えない | 7 件すべて通過 | - |
| T1 | `require_claude_cli = false` を `true` に | 落ちた: codex only、neither、broken claude shim | 落ちた |
| T2 | `require_codex_cli = false` を `true` に | 落ちた: claude only、neither | 落ちた |
| T3 | `require_claude_cli = false` の行を消す | 通過(値が `Default()` の false と同じため) | 落ちた(明示の行を見る) |

- T1 と T2 で、統合のテストが embed の FS ではなくリポジトリの雛形を読んでいることを確かめた。T3 は統合のテストでは生き残るが、`internal/config` の確かめが落とす
- 5 の `ralph init --yes` が書いた `ralph.toml` は `templates/base/ralph.toml` とバイト単位で同じだった(`cmp`)。init が配る値と、テストが読む値は同じファイルから来る

## 本体の mutant

`internal/cli/doctor.go` と `internal/config/config.go` の写しの 1 か所を変え(1 回だけ現れる字面を置き換える perl。スクリプトは evidence にある)、3 と同じ `-run` で走らせた(P8 は `internal/config` の全体、P9 は alias のテストも足した)。build の失敗はない。

| mutant | 変更 | 落ちたテスト |
| --- | --- | --- |
| P1 | `agentCLIResults` で claude の行の `otherUsable` に `claude.Err == nil` を渡す | 表の codex only 2 通り、統合の codex only と broken claude shim |
| P2 | `case required:` を `case required && !otherUsable:` に | 表の 4 通り(片方だけで required の側が使えない)、統合の required 2 件 |
| P3 | 両方ないときの `default` の status を warn に | 表の neither 3 通り、統合の neither |
| P4 | herdr がないときを info に戻す | `TestCheckHerdrAgmsgAvailable_AbsentIsWarn`、`TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` |
| P5 | agmsg がないときを info に戻す | 同じ 2 本 |
| P6 | `checkAgentCLIs` が codex の代わりに claude を probe する | 統合の 6 件(both usable 以外) |
| P7 | herdr がないときを fail に | トップレベル 12 本: herdr の stub を置かずに `runDoctorOpts` か `runDoctorFull` の戻り値を見る 11 本(統合 7 件、merge guarantee 3 件ほか)と `TestCheckHerdrAgmsgAvailable_AbsentIsWarn` |
| P8 | `Default()` の `RequireClaudeCLI` を true に戻す | `TestDefault`、`TestLoad_MissingFile`、`TestLoad_TemplateRalphToml` |
| P9 | Check 8b の配線 `herdrResult.Status == "pass"` を `!= "info"` に | なし。`-run 'AgentCLI\|Doctor\|HerdrAgmsg\|ShellAlias\|Alias'` のトップレベル 93 本も、`internal/cli` のパッケージ全体(`-count=1`、1,233.6 秒)も通った |

- P9 は生き残った。base では herdr がないとき info なので `!= "info"` は `== "pass"` と同じに振る舞ったが、このブランチで herdr がないときが warn になったため、`!= "info"` は herdr がないのに「ある」と見なし、alias の行を info ではなく warn にする。本体の `== "pass"` は正しい(下の binary の確かめ)。計画の Test plan の Edge cases の「herdr がないときも Check 8b(alias)は herdr を pass と見なさない」を固めるテストがない。下の Test gaps の 1

## binary での確かめ

`go build ./cmd/ralph` で組み立てた binary で、scratchpad に `ralph init --yes` で project を作り、`env -i HOME=<scratch> PATH=<stub の dir> RALPH_ORG_AGMSG_HOME=<ない dir>` で `ralph doctor` を打った。

| 条件 | rc | Claude Code CLI | Codex | herdr / agmsg | Shell aliases |
| --- | --- | --- | --- | --- | --- |
| claude だけ | 0 | pass | warn(claude があるので動く、codex を呼ぶ計画助言と cross-review は動かない) | warn / warn(standard flow is unaffected) | pass |
| codex だけ | 0 | warn(同じ文の claude 版) | pass | warn / warn | pass |
| どちらもない | 1(`doctor: 2 check(s) failed`) | fail(neither … needs at least one) | fail(同じ) | warn / warn | pass |
| codex だけ、`~/.zshrc` に `alias codex="codex -m x"`、herdr なし | 0 | warn | pass | warn / warn | info(末尾に `herdr not installed, so no seat is affected today`) |
| 同じ、herdr の stub あり | 0 | warn | pass | pass / warn | warn |

- 4 行目で、herdr がないとき alias の行が warn にならず info のままであることを、本物の配線で確かめた。P9 の振る舞いはテストが固めていないが、HEAD の binary は計画どおりに動く
- warn がある実行でも、最後の行は前からの `All checks passed.` のまま(fail の行がないとき)。herdr と agmsg を入れていない project では、warn が 2 行出たうえでこの行が出る

## Coverage

- Statement: 3 の `-run` に `Agmsg\|Herdr` を足して `-coverprofile` を取った(rc 0、51 秒)。`go tool cover -func` で、`checkAgentCLIs`、`agentCLIResults`、`agentCLIResult`、`checkHerdrAvailable`、`checkAgmsgAvailable` が 100.0%、`runDoctorFull` が 93.5%。`internal/config` の `Default` は 100.0%(パッケージ全体 96.9%)。パッケージ全体の `internal/cli` は、この `-run` では 36.5% で、比べる意味はない
- Branch / Function: Go の標準の道具にない。代わりに上の mutant で、`agentCLIResult` の 4 つの分岐(pass、required、otherUsable、default)と herdr・agmsg の状態をそれぞれ変えると落ちることを確かめた
- Notes: P9 の配線(Check 8b に渡す herdr の判定)は行としては通るが、比べ方を変えても落ちない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| codex だけを入れた project で doctor が失敗する(issue #272) | 統合の codex only と binary の 2 行目で rc 0、claude の行は warn | 3、5 |
| herdr と agmsg がないことが info の 1 行で気づきにくい(issue #272) | warn になり、終了コードは変わらない。P4・P5 を落とす | 3、4、5 |
| detail の退役した Solo の語(tech-debt の (h)) | `solo execution` を含まないことを `TestCheckHerdrAgmsgAvailable_AbsentIsWarn` が見る | 3 |
| 既存の doctor のテスト(scaffold の strict、merge guarantee、codex sandbox、shell alias、retired role key) | 1b の `internal/cli` 全体で通過 | 1b |
| 既存の project の `require_claude_cli = true` が今までどおり fail する(計画の D5) | 統合の claude required and absent、表の required の 8 通りで fail | 3 |

## Test gaps

1. P9: 計画の Test plan の Edge cases にある「herdr がないときも Check 8b(alias)は herdr を pass と見なさない」を固めるテストがない。`internal/cli/doctor.go:134` の `herdrResult.Status == "pass"` を `!= "info"` に変えても、`internal/cli` の全体が通る。このブランチで herdr がないときの状態が info から warn に変わったため、この比べ方が初めて結果を左右するようになった。足すなら、`TestRunDoctorOpts_ShellAliasesCheck_WarnsThroughTheSeam` と同じ組み立てで herdr の stub を置かない 1 本(alias の rc あり、`Shell aliases (codex/claude): info` と `herdr not installed` を見る)で P9 を落とせる見込み。足して P9 が落ちるかは試していない
2. 本物の claude と codex の CLI では試していない(どれも stub の `--version`)
3. `TestRunDoctor_Passes`(`cli_test.go:345`)は doctor の戻り値を見ない。前からの形で、このブランチの変更とは関係しない
4. `docs/recipes/codex-setup.md` と `.claude/skills/org/SKILL.md` の文は、behavioral test の対象ではない(/verify と /sync-docs の担当)

## Verdict

- Pass: pass。HEAD 322bfe18 で `./scripts/run-test.sh` は rc 0(1,491 秒。shell 45 本、Go 9 パッケージ、`(cached)` なし)。名指しの `internal/config` と `internal/cli` の実行も rc 0 で、`TestDefault`、`TestLoad_TemplateRalphToml`、`TestLoad_RequireAgentCLIs` は通った。既存の 21 か所の doctor の呼び出しは、どれも claude と codex の両方を stub で置いており、意図の弱まったテストはない。統合の 5 件はリポジトリの雛形を読み、雛形を戻すと落ちる。本体の mutant は 9 のうち 8 が落ちた。生き残った P9 は計画の edge case 1 つで、HEAD の binary の振る舞いは正しい。/pr の前に 1 本足すかはリードの判断に任せる(Test gaps の 1)
- Fail: なし
- Blocked: なし
