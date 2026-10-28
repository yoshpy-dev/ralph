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

## Run 2 (fix 60602f73)

- Date: 2026-10-28
- Reviewer: reviewer サブエージェント(`/self-review`、cycle 1 の 2 回目の run。Run 1 は e9915a39)
- Scope: `git diff e9915a39 60602f73`(4 ファイル、+17/-15)。Run 1 の LOW 4 件のうち、1〜3 番目を直すコミットだけを見た。観点と、実行しないもの(テスト、ビルド、lint、run-verify)は Run 1 と同じ

### Run 1 の所見の扱い

行の最初の cell を重大度の語にすると、`ralph insights` の backfill が所見として数え直すため、この表は L1〜L4 の番号で始める。

| 番号 | 状態 | 根拠 |
| --- | --- | --- |
| L1(`TestLoad_RequireCodexCLI` の claude 側が何も確かめない) | 解決 | `internal/config/config_test.go:133-161`。関数名は `TestLoad_RequireAgentCLIs` になり、fixture は `require_claude_cli = true` と `require_codex_cli = true` を書き、claude 側の確かめは `!cfg.Doctor.RequireClaudeCLI` で落ちる。既定が false のままでは通らないので、`Load` がキーを読み飛ばせば落ちる。doc comment は「true で往復する」「その CLI が使えないときに fail にする」と今の意味に合う。2 つのキーが同じ値になったので、このテストだけでは 2 つのキーの取り違えを見分けられないが、`TestLoad_FullRoundTrip`(claude を true、go を false)が claude のキーを、このテストが codex のキーを固定するので、組で足りる |
| L2(雛形で `codex trust` の 2 行が規則の段落の末尾に付いた) | 解決 | `templates/base/ralph.toml:10-11`。2 行が `require_claude_cli = false` のすぐあと、`require_codex_cli = false` の直前に戻った。「Codex loads the project's `.codex/config.toml` only after `codex trust .` has been run ...」と主語を Codex にしたので、前の行の規則と切り離されて読める。`TestLoad_TemplateRalphToml` の `\nrequire_claude_cli = false\n` と `\nrequire_codex_cli = false\n` は、どちらもキーの行が単独の行のままなので、コメント行が間に入っても影響しない |
| L3(「at least one of them is required」の "required") | 解決 | `internal/cli/doctor.go:312` が「ralph needs at least one of them」になった。同じ句を持つのは `doctor_agent_cli_test.go:36` の `atLeastOne`、211-212 行の統合テストの 2 か所、doc comment と表の上のコメント 2 行で、すべて同じ commit で直っている。`git grep 'at least one of them is required'` は、計画と Run 1 の report の引用以外にヒットしない。表の `wantNotDetail` が `atLeastOne` を「入っていない」ことの確かめに使う行(required の側)は、新しい句でも `requiredClaude`/`requiredCodex` の文に含まれないので、空振りにならない |
| L4(`agentCLIResult` の位置引数 6 つ) | 受け入れ | 直さない。理由は「`"claude"` と `"codex"` を入れ替えると、16 行の表のテストが `Name` と detail の CLI 名で落ちるため。struct にすると保てる量より増えるコードが多い」。Run 1 でも「直さなくてよい」側に置いた項目で、`doctor.go:290-291` の 2 行の呼び出しに閉じており、同意する。台帳には足さない |

### Run 2 の新しい所見

なし(CRITICAL・HIGH・MEDIUM・LOW とも 0 件)。

確かめたこと:

- `TestLoad_RequireCodexCLI` の旧名: コードと文書に残りはない。ヒットは `docs/evidence/test-2026-10-13-*-go-test.txt` の過去のテスト出力 4 件(履歴)と Run 1 の本文の引用だけ
- 新しい雛形のコメントは `templates/base/ralph.toml` にだけある。この diff は root 側の `ralph.toml` を触っていない(計画の前提どおり、まだない)
- fix のコミットは、動作を変える行を `doctor.go:312` の文字列 1 つだけにしている。判定の `switch` と他の枝の文は変わっていない
- 確かめていない: テストと静的検査の実行(Run 1 と同じ理由で見送り)

### Recommendation (Run 2)

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 0。Run 1 の LOW 4 件は 3 件が解決、1 件が理由つきで受け入れ)
- Follow-ups: 台帳に足す行はない。`/verify` では、`TestLoad_RequireAgentCLIs` への改名と、雛形のコメントの移動を `git grep` で確かめる文書のずれの検査に入れる
