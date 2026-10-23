# Test report: trial-docs-wrapup

- Date: 2026-10-23(JST 15:18〜15:57、UTC 06:18〜06:57)
- Plan: `docs/plans/active/2026-10-23-trial-docs-wrapup.md`(試行の所見を直す系列の 8 本目で最後。AC3・AC4・AC6 と Test plan の Unit・Regression・Edge cases)
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの 1 回目(cycle 1 / 2)
- Scope: HEAD c300f1a9(base 3c1cd14b)。Go の変更はテストのファイル 2 つ(`internal/org/report_names_sync_test.go`、`internal/org/prompts_test.go`)と、go:embed の leader の雛形 `internal/org/prompts/leader.md` の手順 12 の 1 文だけで、ほかは Markdown(skill の 8 写し、recipe の 2 写し、記録)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64、12 CPU)、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8`)
- Evidence: run-test.sh が書いた `docs/evidence/verify-2026-10-23-062143.log` と、下の実行すべての出力をまとめた `docs/evidence/test-2026-10-23-trial-docs-wrapup.log`(どちらも `docs/evidence/*.log` で gitignore の対象なので、コミットしない)
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は依頼どおり Edit で書き換えて対象のテストだけを打ち、Edit で戻した。mutant のあとの `git status --porcelain` は空で、`git status --porcelain --ignored`(`.harness/cache/` を除く)は mutant の前と同じだった。本物の herdr・agmsg・osascript・claude・codex・tmux・gh・network を打つものは実行していない

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh` | shell 44 本、Go 8 パッケージ | 44 本、8 パッケージ | 0 | shell 1 件(下) | rc 0、942 秒(15:21:42〜15:37:24)。load average(1 分)は始まり 5.64、終わり 6.70 |
| 2 | `go test ./internal/org/... ./internal/cli/... -count=1 -timeout 30m`(go1.26.0) | 4 パッケージ | 4 | 0 | - | rc 0、414 秒(15:38:05〜15:44:59)。org 412.518 秒、driver 1.164 秒、protocol 1.511 秒、cli 321.173 秒。load average は 5.35 から 7.06 |
| 3 | 2 と同じコマンドを `GOTOOLCHAIN=go1.25.8` で | 4 パッケージ | 4 | 0 | - | rc 0、405 秒(15:45:16〜15:52:01)。org 399.912 秒、driver 1.671 秒、protocol 1.533 秒、cli 328.513 秒。load average は 6.49 から 5.60 |
| 4 | `go test ./internal/insights/ ./internal/scaffold/ ./internal/upgrade/ ./internal/config/ -count=1 -timeout 30m` を go1.26.0 と go1.25.8 で | 4 パッケージずつ | 8 | 0 | - | rc 0 と rc 0。go1.26.0 は 0.573 秒、1.477 秒、2.489 秒、1.081 秒。go1.25.8 は 1.099 秒、1.598 秒、2.979 秒、3.284 秒 |
| 5 | `go test ./internal/org/ -run 'TestOrgSkill\|TestDirectorSkill\|TestReportNames\|TestCodeBlockAfterLine\|TestRenderRolePrompt_Leader\|TestReadRepoFile\|TestMarkdownSection' -count=1 -v -timeout 30m` | `--- PASS` 77 行(subtest を含む。top-level 31 本) | 77 | 0 | 0 | rc 0、3.890 秒 |
| 6 | `./scripts/check-skill-sync.sh`(root)と、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/check-skill-sync.sh` | 14 skill と 13 skill | 一致 | 0 | - | rc 0 と rc 0 |
| 7 | 対照: mutant の対象の 3 本を、mutant の前とあとに `-count=1 -v` で | 3 本(subtest 5 件) | 3 | 0 | 0 | rc 0 と rc 0 |
| 8 | mutant 8 本(下の表) | 8 | - | 8 本が赤 | - | 1 本 0.45〜1.18 秒 |

- 1 の run-test.sh は `==> Language scope: changed (changed_languages)`、pack は golang。Go の ok の行は `internal/cli` 357.381s、`internal/config` 1.170s、`internal/org` 436.120s。`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` は `(cached)` だった(golang の pack は `-timeout=30m` を GOFLAGS の先頭に入れるが、`-count=1` は付けない)。`github.com/yoshpy-dev/ralph` と `cmd/ralph` はテストのファイルがない
- cached の 5 つは 2〜4 で cache なしに打ち直した。driver と protocol は 2 と 3 で、insights・scaffold・upgrade は 4 で打った。`go list -deps -test` で見ると、insights は `internal/org` を import し、`internal/org` の go:embed(`prompts/*.md`)は変えた leader.md を含む。scaffold と upgrade は `internal/org` にも、templates を embed する root のパッケージにも依存しない。2〜4 のあと、テストのある 8 パッケージすべてが、両方の Go で cache なしに通った
- shell の 44 本はどれも最後の行が `OK`。`tests/test-check-sync.sh` は 16 / 16(scratch の木で `check-sync.sh` の除外の規則を見るテスト。本物の木での `check-sync.sh` は静的解析の側で、/verify が eed46c2c で IDENTICAL 172、DRIFTED 0 を記録している)。`tests/test-check-skill-sync.sh` は 13 / 13。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばす)
- 5 は SKIP の数を見るために打った。skill を読むテストは `readRepoFileOrSkip` を使い、ファイルがないと黙って飛ぶので、パッケージ全体の実行の `ok` だけでは走ったかどうかが分からない。`--- SKIP` は 0 行だった。top-level の 31 本は、`TestOrgSkill_ResultExampleFollowsTheReviewerTemplate`、`TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping`、`TestDirectorSkill_*` 5 本と `TestDirectorSkillWaitBoundProblems`・`TestDirectorSkillNumberedStep`、`TestReportNames_DocsMatchConstants`、`TestRenderRolePrompt_Leader_*` 18 本、`TestCodeBlockAfterLine_TakesTheFirstBlockAfterTheLead`、`TestReadRepoFile_MissingFileIsNotAnError`、`TestMarkdownSection_AnchorsHeaderAndBoundsBody`。計画の Risks が挙げた director の skill の節の構成のテストは、箇条を足した写しで通った
- 6 はリードの依頼で打った(run-test.sh の test の mode は `check-skill-sync.sh` を打たない)。あわせて、`/org` と `/director` の `.claude` と `.agents` の写しを root と `templates/base/` で `cmp` し、recipe の 2 写しも `cmp` した。どれも同じだった
- 1〜8 のあと、`ralph-org` の名前を持つプロセスは残っていない

## AC とテストの対応

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC3 | `TestOrgSkill_ResultExampleFollowsTheReviewerTemplate`(`internal/org/report_names_sync_test.go:269`)、`TestCodeBlockAfterLine_TakesTheFirstBlockAfterTheLead`(`:220`、5 件) | root の `.claude/skills/org/SKILL.md` の `## typed protocol` の節で「RESULT の例」で始まる行のあとの最初の囲みを読み、`protocol.Parse` と `protocol.Validate` を通って TYPE が RESULT であること、`GATE: pass` がちょうど 1 つ、`HEAD:` と `BASE:` が空でない値で 1 つずつあること、本文に `EVIDENCE: docs/reports/review-<YYYY-MM-DD>-<org_id>-<seat_id>.md` の行があることを見る。囲みを探す関数は、導入の行がない、導入の行のあとに囲みがない、囲みが閉じない、の 3 つで found を false にし、導入の行より前の囲みを取らない。mutant M1・M2・M4〜M7 が赤 |
| AC4 | `TestRenderRolePrompt_Leader_FeatureOrgGate`(`internal/org/prompts_test.go:765`) | 描いた leader の雛形の手順 12 が「repo のテストの量とマシンの負荷によっては数十分かかるので、stdout と stderr を `.harness/state/` の下のファイルに書く」を含む。mutant M3・M3b が赤。skill の写しの「数十分」はテストで固定していない(穴 1) |
| AC6 | 1〜4 | `go test` の側は HEAD で満たした。計画の文の `go test ./internal/org/... -timeout 30m` を、より広い `./internal/org/... ./internal/cli/...` で、go1.26.0(2)と go1.25.8(3)の両方で `-count=1` で通した。run-verify.sh の静的解析の側は /verify の範囲で、verify の報告は f310e994 での run-verify(Mode all、scope full)が rc 0、eed46c2c での `run-static-verify.sh` が rc 0 と書く。eed46c2c のあとのコミットは c300f1a9(verify の報告だけ)である |
| AC1・AC2・AC5 | なし(文書と記録) | /verify の範囲。AC2 の「4 つの写しが揃う」は 6 でも見た |

## 計画の Test plan の項目

| 区分 | 項目 | テスト |
| --- | --- | --- |
| Unit | RESULT の例のテスト(新規) | 7 と 5 で PASS。mutant は M1・M2・M4〜M7 |
| Unit | leader の雛形の固定の文(既存のテストの文を変えた) | 7 と 5 で PASS。mutant は M3・M3b |
| Integration | なし(文書の系列) | - |
| Regression | `go test ./internal/org/... ./internal/cli/... -timeout 30m` を、手元の Go と `GOTOOLCHAIN=go1.25.8` で | 2 と 3。どちらも `-count=1` |
| Regression | `./scripts/run-test.sh` | 1 |
| Edge | 例の囲みを見つけられないときに落ちる | M5(導入の行の名前を変えた)が `t.Fatalf` で赤。`TestCodeBlockAfterLine_TakesTheFirstBlockAfterTheLead` の 3 件が、導入の行がない・囲みがない・囲みが閉じないで found が false になることを見る |
| Edge | 例を `self-review-foo.md` に戻すと落ちる | M1 |
| Edge | `GATE:` を消すと落ちる | M4 |

`internal/org/prompts_test.go` の差分は、固定の文 1 行を雛形の新しい文に合わせたもので、ほかの期待は変わっていない。

## Mutant

8 本。M1〜M3 はリードが指定したもの、M3b は M3 の別の形、M4〜M7 は計画の Edge cases と、テストの Parse と Validate の検査の分岐を見るために足したもの。skill の mutant は root の `.claude/skills/org/SKILL.md` だけを変えた(テストはこの写しだけを読む)。leader.md の mutant は go:embed のファイルを直接書き換えたので、`go test` がパッケージを作り直して読み込んだ。

| ID | 変更 | 打ったテスト | 結果 | 落ちた箇所 |
| --- | --- | --- | --- | --- |
| M1 | `/org` skill の例の EVIDENCE を `docs/reports/self-review-foo.md` に | `TestOrgSkill_ResultExampleFollowsTheReviewerTemplate` | 赤(rc 1) | `report_names_sync_test.go:298`(本文に `EVIDENCE: docs/reports/review-<YYYY-MM-DD>-<org_id>-<seat_id>.md` の行がない) |
| M2 | 例の `HEAD:` の行を消す | 同上 | 赤(rc 1) | `:293`(`HEAD:` のヘッダが 0 個) |
| M3 | leader.md の手順 12 の「数十分かかる」を「数分かかる」に(ほかの語と折り返しはそのまま) | `TestRenderRolePrompt_Leader_FeatureOrgGate` | 赤(rc 1) | `prompts_test.go:851`(手順 12 が固定の文を含まない) |
| M3b | 手順 12 を base の文「結果を台帳に記録する。数分かかるので、stdout と stderr を」に戻す(「repo のテストの量とマシンの負荷によっては」もない形。折り返しは 1 行) | 同上 | 赤(rc 1) | `:851` |
| M4 | 例の `GATE: pass` の行を消す | `TestOrgSkill_ResultExampleFollowsTheReviewerTemplate` | 赤(rc 1) | `:289`(`GATE: pass` のヘッダが 0 個) |
| M5 | 導入の行の「RESULT の例」を「RESULT のサンプル」に | 同上 | 赤(rc 1) | `:278`(`t.Fatalf`。節に「RESULT の例」で始まる行のあとの囲みがない) |
| M6 | `TYPE: RESULT` を `TYPE: REVIEW` に(Validate は通る形) | 同上 | 赤(rc 1) | `:286`(TYPE が `REVIEW`、err は nil) |
| M7 | `TASK_ID: t-1` の行を消す | 同上 | 赤(rc 1) | `:286`(Validate の `TASK_ID required for this TYPE`) |

8 本とも赤で、生き残りはない。各 mutant のあとに Edit で戻し、7 の対照(mutant のあと)は 3 本とも PASS だった。

## Coverage

- Statement: 今回は計っていない。この差分は本番の Go のコードを変えていない(変えたのはテストのファイル 2 つと、go:embed の Markdown の 1 文)ので、本番のコードの statement coverage は差分で動かない。最後の記録は `internal/org` 95.0%(`docs/reports/test-2026-10-23-busy-leader.md`)
- Branch: 計っていない(Go の標準の道具にない)
- Function: 計っていない
- Notes: 新しいテスト用の関数 `codeBlockAfterLine` の 3 つの「見つからない」の return と、見つかったときの return は、`TestCodeBlockAfterLine_TakesTheFirstBlockAfterTheLead` の 5 件ですべて通る(表の行と関数の分岐を突き合わせて確かめた。`-coverprofile` では計っていない)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `/org` skill の RESULT の例が `docs/reports/self-review-foo.md` を挙げ、reviewer の雛形の「`self-review-` で始まる名前を使わない」と食い違っていた(#244 の sync-docs が残したもの) | 直っていて、テストで固定された | 7 と 5 で PASS、M1 が赤 |
| leader の雛形が gate を「数分かかる」と書いていた(試行の gate と regate は 1375〜1990 秒) | 直っていて、テストで固定された | 7 と 5 で PASS、M3・M3b が赤 |
| skill の文を読む前からのテスト(`TestDirectorSkill_*`、`TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping`、`TestReportNames_DocsMatchConstants`) | 箇条と前提の項目を足した写しで通る | 5(SKIP 0) |

## Test gaps

1. skill の写しの「数十分」を固定するテストがない。`/org` skill の動詞の表の `gate` の行と「merge してよい順」の regate、`/director` skill の「merge のあとの取り直し」の 1 は、`grep -rn '数十分\|数分' --include='*_test.go' internal tests` で見ると、どのテストのファイルにも出てこない(出るのは `prompts_test.go:830` の leader の雛形の 1 行だけ)。そのため skill の写しを「数分」に戻してもテストは落ちないと推測する(その mutant は打っていない)。計画は AC4 を grep の 0 件で確かめると決めていて、/verify がそれを見る。重さは低い(文書の文)
2. 新しいテストは root の `.claude/skills/org/SKILL.md` だけを読む。ほかの 3 写しは `check-skill-sync.sh`(6、/verify の静的解析でも打つ)が同じであることを保つ。ほかの写しの 1 つだけを変えた変更は、Go のテストではなく `check-skill-sync.sh` が拾う
3. 新しいテストは `internal/org/prompts/reviewer.md` を読まず、`GATE`・`HEAD`・`BASE` のヘッダの名前をテストの中に書く。reviewer の雛形の RESULT の例がヘッダを足すか名前を変えても、このテストは気づかない。雛形とつながっているのは、`TestReportNames_DocsMatchConstants` が雛形のレポート契約で固定する `reviewerReportPrefix` だけである。テストの doc コメントがこの設計を書いている
4. recipe、folder trust の文、記録(AC1・AC2・AC5)には behavioral test がない。recipe を実機で通しで打つことは計画の Non-goals で、依頼の規則でも herdr と claude を打たない
5. coverage は今回計っていない(上の Coverage)

## Verdict

- Pass: pass。`./scripts/run-test.sh` が HEAD c300f1a9 で rc 0(942 秒、shell 44 本と Go 8 パッケージ)、`go test ./internal/org/... ./internal/cli/... -count=1 -timeout 30m` が go1.26.0 で rc 0(414 秒)、`GOTOOLCHAIN=go1.25.8` で rc 0(405 秒)。run-test.sh で cached だった 5 パッケージも、両方の Go で cache なしに通った。mutant は 8 本とも赤。skill の写しの「数十分」はテストで固定していない(穴 1、/verify の grep で見る)
- Fail: なし
- Blocked: なし
