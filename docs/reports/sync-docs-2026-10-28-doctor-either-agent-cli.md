# sync-docs report: doctor-either-agent-cli

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-doctor-either-agent-cli.md`(承認の digest `0a45c27ac33e` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは、受け入れ条件 7 行と Progress 3 行のチェックボックスだけ)
- Issue: #272
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` は 1)。base は `15e301c8`、branch は `feat/doctor-either-agent-cli`。run 開始時の HEAD は `4fca2f4d`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-28-doctor-either-agent-cli.md`(Run 4、`bf29f451`、Merge yes、CRITICAL・HIGH・MEDIUM・LOW とも 0)、`docs/reports/verify-2026-10-28-doctor-either-agent-cli.md`(回し直し、`80fdce33`、pass)、`docs/reports/test-2026-10-28-doctor-either-agent-cli.md`(回し直し、`4fca2f4d`、pass。`run-test.sh` rc 0)
- 差分の大きさ(`git diff 15e301c8...HEAD --shortstat`、この run の前): 14 ファイル、+1106/-84。walkthrough を書くかどうかは `/pr` が決める

## Summary

`ralph doctor` が claude か codex のどちらか 1 つで通るようになった変更(issue #272)の文書の同期である。branch の時点で、`README.md` の 2 か所(60 行目の Quick start、127 行目のコマンド表)、`templates/base/ralph.toml` の `[doctor]` のコメント、`docs/tech-debt/README.md` の (h) が実装に合わせてある。これらを `internal/cli/doctor.go` と `internal/config/config.go` に照らし直し、ずれはなかった。

この run で見つけたずれは 1 件で、`docs/recipes/codex-setup.md` とその `templates/base/` の写しである。Codex だけを入れた machine では `Claude Code CLI` の行が `warn` になるのに、recipe は「doctor の warn はすべて直してから hook の安全を頼る」と書いていた。この `warn` は直す対象ではないので、例外として 1 文を足した。ほかに、claude を既定で必須と書く文書、片方がないと fail と書く文書、herdr・agmsg がないことを info と書く文書は、現在の状態を書くものの中にはなかった。

## Changes made

| File | Change |
|------|--------|
| `docs/recipes/codex-setup.md`(20〜22 行目、33〜36 行目) | コマンドの注釈 `confirms Claude Code/Codex presence` を `confirms Claude Code or Codex is usable (either one passes)` に直した。段落の最後の文「Resolve every doctor warning before relying on hook-driven safety」に、Codex だけの machine では `Claude Code CLI` の行が `warn` で doctor は通ること、意味は claude を呼ぶ計画助言と cross-review が動かないだけであることを例外として足した。文言は doctor が出す warn の detail(「the plan advisory and cross-review that call claude will not run」)に合わせた |
| `templates/base/docs/recipes/codex-setup.md`(同じ行) | 同じ 2 か所。root と `cmp` で同一(`check-sync.sh` の IDENTICAL) |
| `docs/plans/active/2026-10-28-doctor-either-agent-cli.md` | 受け入れ条件 AC1〜AC7 を `[x]` に、Progress の「Review artifact created」「Verification artifact created」「Test artifact created」を `[x]` にした。verify と test の report が AC1〜AC7 を Met とし、issue の受け入れ条件 5 つも Met としている。「PR created」は `/pr` が付けるので `[ ]` のまま。本文は触っていない |
| `docs/insights/events/2026-10-28-doctor-either-agent-cli.jsonl` | `sync_docs` の event を 1 行追記した(`--verdict complete --source pipeline --cycle auto`)。既存のファイルへの追記 |
| `docs/reports/sync-docs-2026-10-28-doctor-either-agent-cli.md` | この report |

## Surfaces checked for drift

変更の範囲は `git diff --name-only 15e301c8...HEAD`: `README.md`、`templates/base/ralph.toml`、`docs/tech-debt/README.md`、`internal/cli/doctor.go`、`internal/config/config.go` と、そのテスト 4 つ、計画と report と insight event である。skill、hook、script、verb、agent の定義は追加も削除もしていない。

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md` 60 行目(Quick start の注釈 `environment check (Claude Code or Codex)`) | 変更なし(正確) | どちらか 1 つで通る動きと合う |
| `README.md` 127 行目(コマンド表の `ralph doctor`) | 変更なし(正確) | 「どちらかが使えれば通り、使えない側は `warn`。両方とも使えないとき、または `require_*` を true にした側が使えないときは fail。既定は両方 false。herdr・agmsg がないのは `warn`で、`warn` は終了コードを変えない」は `agentCLIResult`(`internal/cli/doctor.go`)、`checkHerdrAvailable`、`checkAgmsgAvailable`、`config.Default()` と一致する。agmsg の版が違うときの info は書いていないので、変わっていない動きと矛盾しない |
| `templates/base/ralph.toml` の `[doctor]` のコメント | 変更なし(正確) | 規則の段落が実装と合う。`codex trust` の 2 行は `require_codex_cli` の直前にある。`TestLoad_TemplateRalphToml` が両方の値を `false` と明示して書くことと、`Default()` との一致を見る |
| `docs/tech-debt/README.md` の (h) | 変更なし(正確) | Debt・Impact・Trigger の 3 列で取り消し線と `Closed in feat/doctor-either-agent-cli (2026-10-28)` が付き、(g) と同じ形。その行の計画参照は `docs/plans/active/` を指す |
| `docs/recipes/codex-setup.md` と `templates/base/` の写し | 直した | 上の Changes made |
| `.codex/README.md` 29 行目、`.codex/AGENTS.override.md` 59 行目と、`templates/base/.codex/` の写し | 変更なし | 「`codex` が `$PATH` にあるかを確かめる」「doctor が missing と報告したら」は検査の対象を書く文で、fail か warn かは書かない。今も正しい |
| `.claude/skills/org/SKILL.md` 24 行目(前提)と `.agents/skills/org/SKILL.md`、`templates/base/` の写し | 変更なし | 「`ralph doctor` で `herdr` / `agmsg` チェックを確認する(org を使わない標準フローは両ツールなしで動く)」は status を名指さず、新しい detail の文と同じことを言う。skill は編集していないので `sync-skills.sh` は打っていない |
| `.claude/skills/org/SKILL.md` 142 行目、`docs/recipes/codex-seat-permissions.md` 38 行目の「herdr 未導入なら info」(各写しも同じ) | 変更なし(正確) | Check 8b(shell の alias)の話で、herdr の行の状態ではなく herdr が pass かどうかで決める。計画の Non-goals のとおり変わらず、`TestRunDoctorOpts_ShellAliasesCheck_CodexAliasStaysInfoWithoutHerdr` が固めている |
| `docs/recipes/codex-seat-permissions.md` 16 行目、`docs/recipes/isolated-herdr-server.md` 447 行目 | 変更なし | herdr と agmsg が入っている前提で「pass する」と書く。変わらない |
| `AGENTS.md`、`CLAUDE.md`、`.ralph/core/AGENTS.core.md`、`templates/base/AGENTS.md` | 変更なし | doctor の検査の中身を書いていない(`AGENTS.md` の repo map は `internal/cli/` の subcommand 名に `doctor` を挙げるだけ) |
| `docs/quality/`(definition-of-done、quality-gates) | 変更なし | `doctor` の語を含む行がない |
| `.claude/rules/ralph/` と `templates/base/` の写し | 変更なし | `model-routing.md` の `ralph doctor` の言及はモデルのスラッグの検査の話で、この変更と関係しない |
| `git grep` の残り(`require_claude_cli`、`require_codex_cli`、`checkClaudeCLI`、`checkCodexCLI`、`solo execution`) | 履歴だけ | 現在の状態を書く文書のヒットは `README.md` 127 行目と `templates/base/ralph.toml` だけ。旧関数名と旧文言は `docs/evidence/`、過去の report、計画の本文にだけ残り、Go のコードには残っていない |
| `scripts/check-sync.sh` | rc 0 | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `scripts/check-skill-sync.sh` | rc 0 | `[ok] check-skill-sync: 14 skill(s) in lock-step` |

## Info(記録として残したもの)

日付つきの仕様と過去の report は、その日の記録なので書き換えない。

- `docs/specs/2026-04-16-ralph-cli-tool.md` 249 行目と `docs/specs/2026-05-07-codex-cli-parity.md` 13 行目は `require_claude_cli = true` を書く。前者は退役した `[pipeline]` の鍵を含む設定の例、後者は 2026-05-07 の「現状」の節である。同じ仕様の AC-6 は「片方のみインストールされていてもエラーにならず warning に留める」で、今回の動きと同じ向きである
- `docs/specs/2026-08-01-org-runtime.md` 89 行目の非機能要件(`[ ]`)は「herdr / agmsg は ralph の必須依存とし `ralph doctor` で検査。座席 0 のソロ実行のみ両ツールなしで動作」と書く。status は書かず、退役した Solo の言い方もこの PR の前からある。今回の warn と矛盾しない
- `docs/tech-debt/README.md` 197 行目(org-feature-worktree の小さな所見の行)の Trigger に残る日付つきの注記(2026-10-26 の `feat/reviewer-runs-gate` の注記など)は、「open items (a) to (e), (h), (i), (k), and (l)」と (h) を開いた項目に数えている。その日の時点では正しい記録で、(h) を閉じた注記は同じ行の Debt・Impact・Trigger にある
- `docs/tech-debt/README.md` 37 行目は `internal/cli/doctor.go:225-294`(`checkHooks`)を、39 行目(解決済みの行)は `doctor.go:111-138`(`probeBinary`)を引く。base の `15e301c8` で `checkHooks` は 570 行目、`probeBinary` は 248 行目で、この PR より前からずれている。この PR が `checkHooks` を 12 行ずらしたが、元のずれの方が大きい。直していない(この PR の範囲の外。直すなら行番号をやめて関数名だけにする)
- 警告の detail と `README.md` の「the plan advisory and cross-review that call claude will not run」は、今の計画助言が codex だけを呼ぶので、claude がないときに実際に止まるのは Codex が primary のときの cross-review が呼ぶ `claude -p` だけである。「that call claude」で呼ぶ側に限っているので文は偽ではなく、計画の D4 が #265・#266 で呼び先が変わることを見越して CLI を主語にしている
- warn の行があっても要約の行は `All checks passed.` のまま(fail の行がないとき)。前からの動きで、herdr と agmsg を入れていない project では warn 2 行のあとにこの行が出る。計画の Risks に載っている

## Decisions on the reports' remaining items

| 出どころ | 項目 | 扱い |
|----------|------|------|
| self-review | L4(`agentCLIResult` の位置引数 6 つ) | 受け入れ済み(Run 2)。台帳には足さない。呼び出しが `doctor.go` の 2 行に閉じ、入れ替えは 16 行の表のテストが落とす |
| self-review | N2 の後半(兄弟のテストと共通の準備、手書きの `os.Pipe()` 2 か所) | 受け入れ済み(Run 4)。台帳に足すのは cross-review が ACTION_REQUIRED を返して最後の cycle に入ったときだけ、と Run 4 が書いている。今は足さない |
| verify | Info 1〜5(古い仕様、退役した Solo の語、日付つきの注記、warn の文の対象、要約の行) | 上の Info の節 |
| verify | Info 6・8(AC のチェックボックスと Progress の遅れ) | この run で付けた |
| verify | Info 7(test の report が `322bfe18` の記録で古い) | test の回し直し(`4fca2f4d`)が更新済み。Test gaps の 1 は閉じている |
| test | Test gaps 2(本物の claude・codex では打っていない) | tech-debt の行は足さない。stub の `--version` の振る舞いは `probeBinary` で、この PR は変えていない |
| test | Test gaps 3(`TestRunDoctor_Passes` が戻り値を見ない) | 前からの形で、この PR と関係しない。足さない |
| test | Test gaps 4(`codex-setup.md` と `/org` skill の文は behavioral test の対象外) | この run で確かめた(上の表)。`codex-setup.md` を直した |

## Plan drift notes

- 計画本文の Scope と Affected areas は、S1〜S5 のとおりに実行した。逸脱は 1 件で、S3 が `internal/cli/doctor_codex_writable_root_test.go` の `checkAgmsgAvailable` の前提の確かめを直さなかったこと。agmsg の版が違うときの info は D6 どおり info のままなので、Progress の記録のとおりで正しい
- 計画の Affected areas にない `docs/recipes/codex-setup.md` と `templates/base/` の写しをこの run で変えた。計画の Verify plan が drift の確認先として名指ししていたファイルで、変更は 2 つの写しに同じ 2 か所である
- 残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける

## `/pr` と release に渡すこと

- 実機の org や本物の claude・codex での確かめはない。doctor の出力は、組み立てた binary と stub の `claude`・`codex` で 7 通り(verify)と 5 通り(test)を打って確かめた
- 差分は 14 ファイル、+1106/-84(この run の前)。この run が足したのは recipe 2 つと計画のチェックボックスと report
- コミットの型は `docs:` が 14 件、`feat:` が 3 件、`test:` が 2 件、`fix:` が 1 件(この report の commit を除く)。`.goreleaser.yml` の changelog は `^docs:`、`^test:`、`^chore:` を除くので、自動の changelog に載るのは `feat:` の 3 件(`a6ad522e`、`c9148ae9`、`17ba114b`)と `fix:` の 1 件(`60602f73`)。`60602f73` は同じ PR の `feat:` の文言の直しで、手書きの release ノートでは新しい動きの一部として扱い、既存の不具合の修正とは書かない
- tech-debt の行の Related 列は `docs/plans/active/` を指す。`/pr` の `archive-plan.sh` が `archive/` に書き換える

### Release notes

PR の本文と release ノートに使える文。

- `ralph doctor` は、claude と codex のどちらか 1 つが使えれば通る。使えない側は `warn` になり、その CLI を呼ぶ計画助言と cross-review が動かないことを伝える。どちらも使えないときだけ、2 行とも fail になる(終了コード 1)
- `[doctor].require_claude_cli` の既定が `false` になった(`require_codex_cli` と同じ)。`templates/base/ralph.toml` も両方を `false` と明示して書く。`ralph.toml` は seed なので `ralph upgrade` は上書きしない。既存の project の `ralph.toml` が `require_claude_cli = true` のままなら、今までどおり claude がないときに fail する。どちらか 1 つで通したいときは `false` にする。`require_*` を `true` にした側は、もう片方が使えても、使えなければ fail する
- herdr と agmsg がないときは `info` ではなく `warn` になる(detail は「the standard flow is unaffected」)。`warn` は doctor の終了コードを変えない。agmsg の版が試した版と違うときの `info` は変わらない

## Verification

| Check | Result |
|-------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-28-doctor-either-agent-cli.md` | `0a45c27ac33e`(`- Approved:` 行の `sha256:0a45c27ac33e` と同じ。チェックボックスの編集の前後で同じ) |
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh` | rc 0。14 skill が一致 |
| `cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md` | rc 0(同一) |
| `git diff --check` | rc 0、出力なし |
| テスト | 打っていない(`/test` の範囲)。この run が変えたのは文書、計画のチェックボックス、report、insight event だけで、コードとスクリプトには触れていない |
| insight event | `./scripts/insights-append.sh --slug doctor-either-agent-cli --flow standard --phase sync_docs --verdict complete --source pipeline --cycle auto` を実行した。既存のファイルへの追記 |
