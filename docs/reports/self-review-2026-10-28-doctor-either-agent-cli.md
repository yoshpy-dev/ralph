# Self-review report: doctor-either-agent-cli

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-doctor-either-agent-cli.md`(D1〜D6、AC1〜AC7)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプライン cycle 1 の 1 回目)
- Scope: `git diff origin/main...HEAD`(base 15e301c8、9 ファイル、+550/-77)。コミットは a6ad522e(S1 既定値)、c9148ae9(S2 `checkAgentCLIs`)、17ba114b(S3 herdr と agmsg の warn、tech-debt (h))、d171e3e6(S4 README)、計画のコミット 2 本。見た観点は naming、readability、null safety、exception handling、security、maintainability。spec の合致、テストの網羅、文書のずれは `/verify` と `/test` の範囲なので見ていない。不要な変更・誤字・デバッグ残り・秘密情報・範囲外のファイルは実装側の確認の項目で、diff に出てきたものだけ見る方針にした(該当なし)

## Evidence reviewed

- `internal/cli/doctor.go` の差分を全文: `checkAgentCLIs`(277 行)、`agentCLIResults`(289 行)、`agentCLIResult`(297 行)の 3 関数。probe は各 CLI 1 回、判定は probe のエラーの有無と `config.DoctorConfig` だけを読む純関数で、`switch` の順は pass → required の fail → 相手が使えるなら warn → 両方使えない fail。4 つの枝が互いに重ならず、`required` を `otherUsable` より先に見るので、AC3 の「相手が使えても required は fail」が成り立つ
- `checkHerdrAvailable` と `checkAgmsgAvailable` の全文と doc comment: ないときだけ info を warn にし、agmsg の版違いの info は残してある。`agmsgTestedVersion` の doc comment の「版違いは info のまま」と矛盾しない。Check 8b は `herdrResult.Status == "pass"`(134 行)で決めるため、herdr の状態が warn になっても分岐は変わらない
- Check 11b の `checkCodexAgmsgWritableRoot` の doc comment(`doctor_codex_writable_root.go:872-876`)は、AgmsgAvailable の結果で判定し、別の "agmsg" 行の Status では判定しないと書く。この diff の影響を受けない
- 旧関数名の取りこぼし: `git grep -n 'checkClaudeCLI\|checkCodexCLI'` は計画の本文以外に残りなし。`git grep 'solo execution'` は `internal/` で、それを「使わない」ことを確かめるテストの文字列だけ
- `RequireClaudeCLI` / `RequireCodexCLI` の読み手は `doctor.go:290-291` だけ。スクリプトと他の Go コードに `require_claude_cli` を読むものはない
- `internal/cli/doctor_agent_cli_test.go`(新規 286 行)を全文: 判定の表 16 行(使える側 4 通り × `require_*` 4 通り)、`doctorLine`、統合 7 件。統合テストの環境の閉じ方(`PATH` を stub だけにする、`RALPH_ORG_AGMSG_HOME` を空き地へ)は、既存の `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` と同じ
- `internal/cli/doctor_org_test.go` の差分: 名前を `AbsentIsWarn` にし、`info` を `warn` に、新しい文と古い文の不在を確かめる。コメントの直しは S1 で古くなった「required-by-default」と消えた関数名の 2 か所だけ
- `internal/config/config.go`、`config_test.go`、`templates/base/ralph.toml` の差分を全文。`TestLoad_FullRoundTrip` は `require_claude_cli = true` を書くので、既定が false になったあとも true の読み込みを区別して確かめられる
- `docs/tech-debt/README.md` の行 197 を 5 つの cell に分けて、(h) の扱いを main と HEAD で比べた: Cell 0(本文)、Cell 1(Impact)、Cell 3(Trigger)の 3 か所が取り消し線と閉じた旨になった。Cell 3 に残る「The open items (a) to (e), (h), (i), (k), and (l) ...」の 2 文は 2026-10-14 と 2026-10-26 の日付つきの記録で、その時点では正しい
- 確かめていない: テスト・静的検査・ビルドの実行(他のセッションが重い処理を走らせているため、頼まれて見送った)、ビルドした `ralph doctor` を PATH を変えて打つ probe(`/test` と `/verify` の Evidence に回る)

## Findings

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。動作を変えるものはない。

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | 既定値の変更で、`TestLoad_RequireCodexCLI` の claude 側の確かめが何も確かめなくなった。この fixture は `require_claude_cli = false` を書き、`require_claude_cli = true after loading explicit false` で見る。既定が true だったときは「明示した false が既定の true を上書きする」ことを確かめていたが、既定が false になったので、`Load` が `require_claude_cli` を読み飛ばしても落ちない。doc comment の「warn (default) and fail」も、今は相手の CLI が使えるかで warn か fail かが分かれるので古い | `internal/config/config_test.go:134-158`(fixture の 2 行目と 156-158 行の確かめ)、`internal/config/config.go:538` | claude 側の確かめを、既定と違う値を使う形にするか(`require_claude_cli = true` を書いて true を確かめる。ただしこれは `TestLoad_FullRoundTrip` と重なる)、消して codex 側だけにする。doc comment は「`require_codex_cli = true` が読めて、`ralph doctor` が使えない codex を fail にする」程度に直す |
| LOW | readability | 雛形の `[doctor]` のコメントの最後の 2 行(`.codex/config.toml` は `codex trust .` のあとだけ読まれる)は、前は `require_codex_cli` の直前のコメントの一部で、codex の話の続きだった。今は doctor の pass と fail の規則を説明する段落の末尾に付き、`require_claude_cli` の直前に来る。規則の話から `codex trust` の話へ急に移り、`require_*` のどちらにも関係しない。この ralph.toml は seed として全 project に写る | `templates/base/ralph.toml:9-10` | 規則の段落のあとに空の `#` 行を挟んで別の段落にするか、`require_codex_cli` の直前に戻す。前者のほうが 2 つの `require_*` の並びを崩さない |
| LOW | readability | 両方使えないときの fail の文「neither claude nor codex is usable, and at least one of them is required」の "required" は、同じ関数の warn の枝の "(not required)" と、設定名 `require_*_cli = false` の "require" と同じ語で、別のことを言っている。既定のまま両方ない人は、`require_*_cli = false` なのに "required" と読む。1 行に " — " が 2 回出る点(行の区切りと detail の中)は既存の `agmsg` の版違いの文にもあり、問題にしない | `internal/cli/doctor.go:312`。同じ句が `doctor_agent_cli_test.go:35` の `atLeastOne` と 210-211 行にある | "ralph needs at least one of them" のように、設定の語と離れた動詞にする。直すなら 3 か所の句を同時に |
| LOW | readability | `agentCLIResult` が位置引数 6 つを取り、`name` と `bin`、`bin` と `otherBin` が文字列で並ぶ。呼び出し側(290-291 行)で `"claude"` と `"codex"` を入れ替えても型で気づけず、文の中の CLI 名だけが逆になる。表のテストが detail の CLI 名を固定しているので落ちるが、`name` は `bin` から決まるのに別の引数になっている | `internal/cli/doctor.go:297`、呼び出し 290-291 行 | 呼び出し側が組む小さな struct(`agentCLI{Line, Bin string}` と `usable bool`)にして `agentCLIResult(self, other agentCLI, probe, required)` のような形にするか、今のままでよいと判断する。動作は変わらない |

## Positive notes

- 判定を probe の結果と `config.DoctorConfig` だけを読む純関数に分け、stub の binary なしに 16 通りの表で試せる。`checkAgentCLIs` は probe 2 回と委譲だけで、`--version` を打つ回数は変わらない
- 使えない側の `required` を `otherUsable` より先に見る順は、AC3 の「相手が使えても required は fail」を `switch` の順だけで表し、コメントなしで読める
- 統合テストが `templates/base/ralph.toml` をそのまま project に写す。テスト用の embed FS の `ralph.toml` を読む既存のテストでは、雛形の `require_claude_cli` を true に戻しても気づかない(計画の Codex 助言 1)。このテストと `TestLoad_TemplateRalphToml` の 2 本で防げる
- S3 の既存テストの直しが、`info` を `warn` に置き換えるだけでなく、`solo execution` の文字列が残っていないことと新しい文を確かめる。tech-debt (h) を閉じる根拠がテストに残った
- tech-debt の (h) は Impact・Trigger・本文の 3 つの cell を同じ形で閉じ、閉じた旨が branch 名と日付と変更の中身(warn、標準フロー)を持つ。半分だけ閉じる形(`pattern_techdebt_half_closure`)ではない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(上の LOW 4 件は、直さずに進めるなら 1 行にまとめて台帳へ足す) | — | — | — | — |

_(足した行はない。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4)
- Follow-ups: LOW 4 件はどれも動作を変えない。直すなら 1 番目(`TestLoad_RequireCodexCLI` の確かめ。テストの意図が静かに弱くなっている)を優先し、3 番目は文言と 3 か所の句を同時に変える。2 番目と 4 番目は好みの範囲で、直さなくてよい。直す場合は、`/self-review` から回し直すかどうかを cycle の上限(既定 2 回)に照らして決める。直さない場合は、台帳に 1 行で足す
