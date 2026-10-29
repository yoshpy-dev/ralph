# Walkthrough: doctor-either-agent-cli(doctor は claude か codex のどちらかで通す)

- Date: 2026-10-28
- Plan: docs/plans/active/2026-10-28-doctor-either-agent-cli.md(/pr で archive に移す)
- Issue: #272
- Branch: feat/doctor-either-agent-cli(この walkthrough の前の HEAD 1925919d。22 commits、18 files、+1261/-90)
- Base: 15e301c8(main、「Merge pull request #279」)
- 読む範囲: 実行コードは `internal/cli/doctor.go`(+100 行前後の入れ替え)と `internal/config/config.go`(既定値 1 行)。残りのコードはテストで、`internal/cli/doctor_agent_cli_test.go`(新規 +287)、`doctor_org_test.go`(+52 行前後)、`doctor_shell_alias_test.go`(+53)、`internal/config/config_test.go`。文書は `templates/base/ralph.toml` のコメント、`README.md` の 2 か所、`docs/recipes/codex-setup.md` とその雛形の写し、`docs/tech-debt/README.md` の (h)。記録は計画、後工程の report 6 本、insight event。
- 後工程は cycle 1 の中で 2 回まわした。1 回目の /test の mutant で、計画の Test plan の edge case を固めるテストがないと分かり、スライス S5 を足して /self-review・/verify・/test を回し直した。cross-review は所見 0 件。記録だけの修正の例外は使っていない

```
何を変えるか → 既定値(S1) → claude と codex の判定(S2) → herdr と agmsg(S3) → README(S4) → alias の配線のテスト(S5) → 文書の同期 → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで: `ralph doctor` は claude を既定で必須として検査していた(`require_claude_cli` の既定が true)。codex だけを入れた machine では doctor が fail した。herdr と agmsg がないときは info だった。
- この PR から:
  - claude と codex のどちらか 1 つが使えれば doctor は通る。使えない側の行は warn で、その CLI を呼ぶ計画助言と cross-review が動かないことを伝える。
  - 両方とも使えないときは 2 行とも fail(終了コード 1)。
  - `require_claude_cli` の既定は false(`require_codex_cli` と同じ)。`require_*_cli = true` にした側は、もう片方が使えても、使えなければ fail。
  - herdr と agmsg がないときは warn で、「the standard flow is unaffected」と書く。warn は終了コードを変えない。agmsg の版が違うときの info は変えない。
- release のノート: changelog に出るのは `feat:` 3 件(a6ad522e、c9148ae9、17ba114b)と `fix:` 1 件(60602f73)。60602f73 は同じ PR の文言の直しなので、ノートでは新しい動きの一部として書く。ノートの文は `docs/reports/sync-docs-2026-10-28-doctor-either-agent-cli.md` の「Release notes」にある。`ralph.toml` は seed のファイルで、`ralph upgrade` は上書きしない。既存の project で `require_claude_cli = true` が残っていれば、今までどおり claude がないと fail する。
- 戻し方: この PR を revert する。設定の形と項目名は変えていないので、既存の `ralph.toml` はそのまま読める。

## 2. 既定値(S1、a6ad522e)

- `internal/config/config.go:538-539`: `Default()` の `RequireClaudeCLI` と `RequireCodexCLI` を両方 false に。
- `templates/base/ralph.toml:3-13`: `[doctor]` の説明を新しい判定に合わせ、2 つの項目を false と明示して書く。`codex trust .` の注意のコメントは `require_codex_cli` の直前に移した。
- テスト: `TestDefault`(`config_test.go:14`)、`TestLoad_TemplateRalphToml`(`:76`、雛形の値が `Default()` と同じこと)、`TestLoad_RequireAgentCLIs`(`:138`、旧名 `TestLoad_RequireCodexCLI`、true を書いた toml が読めること)。
- 見ること: Codex の計画助言(MEDIUM)の指摘で、既定値のスライスを最初に回した。テストの embed FS の `ralph.toml` は雛形と別物なので、雛形が後退しても統合テストが気づかない。そのため S2 の統合テストはリポジトリの雛形をディスクから読む。

## 3. claude と codex の判定(S2、c9148ae9)

- `internal/cli/doctor.go:107`: Checks 1 と 2 を `checkAgentCLIs(cfg)` の 1 回の呼び出しにまとめた。行の名前は「Claude Code CLI」と「Codex」のまま。
- `:277-282` `checkAgentCLIs`: claude と codex を `probeBinary` で 1 回ずつ確かめてから、両方の行を決める。片方の判定にもう片方の結果が要るため。
- `:289-293` `agentCLIResults` と `:297-316` `agentCLIResult`: 判定の順は、使える → pass、必須 → fail、相手が使える → warn、どちらも使えない → fail。warn の文は「相手が使えるので ralph は動くが、この CLI を呼ぶ計画助言と cross-review は動かない」。
- テスト: `TestAgentCLIResults_EveryCombination`(`doctor_agent_cli_test.go:20`、16 行の表)、`TestRunDoctorOpts_AgentCLIs`(`:189`、統合 7 件)。統合のうち 5 件(両方あり、claude だけ、codex だけ、どちらもなし、`--version` が失敗する claude の shim)はリポジトリの `templates/base/ralph.toml` を写し、残る 2 件は `require_*_cli = true` を書いた toml を使う。
- 見ること: `--version` が失敗する shim は「使えない」として扱う。これは今の `probeBinary` の振る舞いで、この PR は変えていない。

## 4. herdr と agmsg(S3、17ba114b)

- `internal/cli/doctor.go:780-790` `checkHerdrAvailable` と `:812` からの `checkAgmsgAvailable`: ないときの状態を info から warn に、detail の結びを「(the standard flow is unaffected)」に変えた。
- Check 8b(shell の alias、`doctor.go:134`)は `herdrResult.Status == "pass"` で herdr の有無を決めており、この PR では変えていない。
- `docs/tech-debt/README.md` の (h) を閉じた(3 つの列)。
- テスト: `TestCheckHerdrAgmsgAvailable_AbsentIsWarn`(`doctor_org_test.go:70`、旧名 `..._AbsentIsInfo`)、`TestCheckAgmsgAvailable_PathBootstrapperNotHome`(`:133`)、`TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected`(`:198`)。
- 見ること(計画からの逸れ): 計画の S3 は `doctor_codex_writable_root_test.go` の確かめも warn に直すと書いていた。その確かめ(`TestCheckCodexAgmsgWritableRoot_AgmsgVersionMismatch_StillGraded`)は agmsg の版が違うときが前提で、info のままが正しいので変えていない。計画の Progress に記録してある。

## 5. README(S4、d171e3e6)

- `README.md:60`: `ralph doctor` のコメントを「environment check (Claude Code or Codex)」に。
- `README.md:127`: コマンドの表の doctor の行に、どちらか一方で通ること、使えない側は warn、両方ないときと必須にした側がないときは fail、herdr と agmsg がないときは warn、warn は終了コードを変えないことを書いた。

## 6. alias の配線のテスト(S5、8408b30c と ee5b3bda)

- 1 回目の /test で、`doctor.go:134` の `== "pass"` を `!= "info"` に変える mutant(P9)が生き残った。herdr がないときの状態が info から warn に変わったので、この比べ方が結果を左右するようになったが、それを固めるテストがなかった。計画の Test plan の edge case「herdr がないときも Check 8b は herdr を pass と見なさない」にあたる。
- `internal/cli/doctor_shell_alias_test.go:1563` `TestRunDoctorOpts_ShellAliasesCheck_CodexAliasStaysInfoWithoutHerdr`: herdr を PATH に置かず、codex の alias がある rc で doctor を通す。herdr の行が warn、alias の行が info、detail に「(herdr not installed, so no seat is affected today)」が入ることを `doctorLine` で同じ行について確かめる。
- self-review の Run 3 の LOW 3 件(名前、stdout の取り込み、確かめ方)は ee5b3bda で直した。兄弟のテストと共通の準備部分は、兄弟のテストに触るので残した。
- 再実行の /test で、P9 はこのテストだけで落ちることを確かめた。

## 7. 文書の同期(40b4e79c)

- `docs/recipes/codex-setup.md:20-22` と `:33-36`、`templates/base/docs/recipes/codex-setup.md` の同じ 2 か所: doctor は claude か codex のどちらかが使えれば通ること、codex だけの machine では「Claude Code CLI」の行が warn になることを書いた。計画の Affected areas にないファイルで、sync-docs の report の Plan drift notes に記録してある。
- `docs/specs/` の古い spec が `require_claude_cli = true` と書く箇所は、日付つきの記録なので残した。

## 8. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(HEAD bf29f451)
  - `./scripts/run-test.sh` rc 0(HEAD 80fdce33、Go 9 パッケージ、shell 45/45)
  - 本体の mutant 9 つ(1 回目)のうち 8 つが落ち、残りの P9 も S5 のテストで落ちる(2 回目)
  - verify で、worktree から組み立てた binary を scratchpad の project で 7 通り打ち、claude だけ・codex だけ・両方なし・必須の組み合わせ・herdr と agmsg なしの出力と終了コードを見た
  - cross-review(codex、cycle 1/2)の所見は 0 件
- 確かめていないこと:
  - 本物の claude と codex を入れた machine での doctor。出力は stub の `claude`・`codex` で verify 7 通り、test 5 通りを打って確かめた
  - 40b4e79c(文書の同期)と 1925919d(cross-review の report)のあとの `run-static-verify.sh`。文書だけの変更で、CI の verify ジョブで確かめる
