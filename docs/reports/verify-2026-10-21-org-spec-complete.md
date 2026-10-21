# Verify report: org-spec-complete

- Date: 2026-10-21
- Plan: `docs/plans/active/2026-10-21-org-spec-complete.md`(Status: Approved。`plan-visual.sh digest` は `98781cfa6662` で `- Approved:` 行と一致)
- Verifier: verifier subagent(Claude Code、pipeline cycle 1)
- Scope: `git diff 765bcc46...HEAD`(HEAD 82ba75b5、5 ファイル、+397/-33、docs だけ)。AC1〜AC8 と AC6b、計画の Verify plan、Non-goals、Assumptions、Progress に記録された 2 つのずれ。self-review の報告 `docs/reports/self-review-2026-10-21-org-spec-complete.md` は c56b70a4 の版を見た再レビューで Merge: yes。そのあとの 43bf5c6e(N1・N4・N5 の直し)と 82ba75b5(計画の Progress)もこの範囲に含めた
- Evidence: `docs/evidence/verify-2026-10-21-org-spec-complete.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-21-032425.log`(どちらも `docs/evidence/*.log` の gitignore でコミットしない生の出力)。照合に使った python(Status と印の注を除いて base と比べる正規化、テスト名の照合、印と判定の照合、tech-debt の行の照合)は scratchpad に置き、出力を前者の log に写した

## Spec compliance

テストの実行は `/test` の範囲なので回していない。印の照合と語単位の差は、scratchpad の python を `python3 -I` で HEAD と base に当てて確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 仕様に `## Status` があり、9 段の完了、段ごとの PR 番号、`[ ]` が残ることとその理由の置き場所、release は人が切ることを書く。PR 番号は MERGED を確かめた番号だけ | 満たす | 仕様 3〜24 行。5 行が 9 段の完了、7〜18 行が 28 本を段ごとに並べ、並びと番号は計画の Assumptions の一覧と同じ。20 行が「2026-10-21 の時点で 12 項目は `[ ]`」と、理由が根拠の表と各項目の「印(2026-10-21)」にあることを書き、24 行が「人が `/release` で切る」。28 本を `gh pr view <n> --json state,mergedAt` で読み、すべて `MERGED`(読んだだけ)。24 行の「最新のタグは v5.1.0(2026-08-24)」は `git tag` と `gh release list` で合う |
| AC2: Rollout に「進行中」がなく、9 段の行それぞれに PR 番号がある | 満たす | `grep -c 進行中` は HEAD で 0(base は 1)。Rollout(262 行の節、264〜272 行)の 9 行すべてに PR 番号があり、Status の番号と同じ |
| AC3: FR-1〜FR-11、NFR 5 つ、受け入れ条件 15 個の印が根拠の表の判定と一致し、`[ ]` の項目に注がある | 満たす(V1、V3) | 印と判定の照合: FR-12 を除く 31 項目(FR 11、NFR 5、受け入れ条件 15)を表の判定の列と 1 つずつ比べ、31 件一致。表にない項目も、表にだけある項目もない。`grep -c '^- \[x\]'` は 20(表の 19 と前から `[x]` の FR-12)、`'^- \[ \]'` は 12(表の 12、Status の「12 項目」と同じ)。`[ ]` の 12 項目はどれも直下に「印(2026-10-21)」の注を持つ。所見と食い違う 10 項目の注は所見と tech-debt の行を指し、実機が足りない 2 項目(FR-10、NFR ツール非依存)の注は Open questions の 1 つ目と 2 つ目を文言で指す。2 つの文言は 284 行の節の 1・2 番目の問いとそのまま合う |
| AC4: 「checkbox は付けない」の類の文が残らない | 満たす | `checkbox` を含む行は HEAD で 2 行(base は 6 行)。100 行は FR-7 の承認の digest が計画の checkbox を飛ばす話で、仕様の印ではない。132 行は「FR-12 の checkbox は付けた」で、FR-12 の `[x]` と合う。「付けない」が残る 5 行(65・91・107・116・118 行)は予約・flag・片付けの話で、印とは関係がない |
| AC5: FR-10 の 8b-1 の注に「121 行」「120 行」が残らず、受け入れ条件を文言で指す | 満たす | 「121 行」「120 行」は HEAD で 0 件(base は各 2 件)。数字に「行」が続く残りは「1 行」「2 行」「1 行目」で、行番号の参照ではない。127 行の 4 か所は受け入れ条件を文言で指し、その文言は 165 行(`wake = ["pull", "push"]` で…)と 167 行(`wake` に push を含むが…)にそのままある。base の「121 行の「when 設定を読む…」」は 167 行の条件、「120 行の「when escalate が記録される」」は 165 行の条件の when 節で、指す先は入れ替わっていない |
| AC6: 根拠の表が 31 項目を持ち、各行に根拠と判定があり、表のテスト名が `git grep -n 'func <名前>('` で 1 件ずつ見つかる | 満たす | 表の「項目」(45〜77 行)は 31 行で、FR 11、NFR 5、受け入れ条件 15。どの行も根拠と判定の列が埋まっている。テスト名の照合: 表に出るテスト名 103 か所(66 個)を、同じ行で直前に書かれた `_test.go` のパスと組にし、`git grep -n -F 'func <名前>('` が repo 全体で 1 件、そのパスでも 1 件になることを確かめた。不一致は 0。表の 7 行は 85ec1339 で確かめたと書くが、HEAD 82ba75b5 でも同じ結果になる。表が挙げるパス 75 個のうち 72 個は HEAD にあり、残る 3 個は仕様の例の `internal/auth/` と `internal/auth/token.go`、それにブランチ名 `docs/org-spec-complete`。根拠が計画と報告の有無だけの「NFR 入れ方」は別に確かめた: 25 本の PR の files を `gh pr view --json files` で読むと、どれも `docs/plans/archive/` の計画と self-review・verify・test・sync-docs の報告を 1 つずつ足していて、その 125 ファイルは HEAD にある |
| AC6b: 所見の表があり、各所見に関係する項目、食い違うか、直す PR を書く。食い違う所見に結びつく項目は `[ ]` で、所見と tech-debt の行を指す | 満たす(V2) | 所見の表(18〜37 行)は 18 行(F1〜F14、D-F3、folder trust、G1、「leader への ALERT」)で、どの行も 6 列そろっている。「食い違う」が「はい」の 7 行(F3、F4、F7、F10、F13、D-F3、ALERT)は、Status 20 行の所見の一覧と同じ。結びつく項目(FR-2、FR-3、FR-4、FR-5、FR-6、FR-7、FR-9、NFR 安全の不変条件、受け入れ条件 2・5)はすべて `[ ]` で、注が所見と tech-debt の行を指す。F4 の行が「食い違わない」とする受け入れ条件 3 は `[x]` で、注(152 行)が `RunningOrgs` を理由に挙げる。tech-debt の行の照合: 表と仕様の注が引く行の名前 31 か所のうち 30 か所は、`docs/tech-debt/README.md` の行の最初のセルの先頭と 1 行だけ一致する。残る 1 か所(表 27 行の F8)は閉じの `~~` の位置が違うだけで、`~~` を除けば 320 行に 1 行だけ一致する |
| AC7: 印・注・Status・Rollout の PR 番号の外の行は変わらない | 満たす | HEAD から Status の節と「印(2026-10-21)」の注 20 行を除き、`[x]` を `[ ]` に戻してから、base と `git diff --no-index --word-diff=porcelain` で比べた。残る差は 5 hunk で、FR-7 の 7c-2 と 8a-3b の注の checkbox の文、FR-8 の 8a-3a の注の checkbox の文の削除、FR-10 の 8b-1 の注の行番号 4 か所と checkbox の文、FR-12 の注の checkbox の文、Rollout の PR 番号 9 行だけ(FR-12 の `[x]` が `[ ]` に見える差は正規化で生じたもの)。どれも計画の Scope が名指した文で、要件と設計の文の変更は 0。7c-2 の注は「まだ決めていないので、FR-7 の checkbox は付けない」を「この時点ではまだ決めていなかった(8a-3b の注で決めた)」に変え、歴史の記述は残している |
| AC8: `./scripts/run-verify.sh` が exit 0(check-sync と check-skill-sync を含む) | 静的な半分を満たす(テストの半分は `/test`) | `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh` が rc 0、`All verifiers passed.`。check-sync は IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、check-skill-sync は 14 skill がそろう。`run-verify.sh` の既定(all)のテスト部分は回していない |

計画のほかの項目:

- Non-goals: diff に `.go` も `.sh` もない。ほかの仕様(`docs/specs/2026-08-01-org-runtime.md` など)は diff に入っていない。タグは v5.1.0 のままで、release は切られていない
- Assumptions: 28 本の PR はすべて MERGED。仕様の行番号を指す文書は reports・evidence・archive の外で 0 件(`git grep 'org-multi-org-director.md:[0-9]'`。reports と evidence の中には 45 件あり、過去の記録なので直さない)。`#L`・`line <n>` の形も 0 件。仕様を読むテストやスクリプトはない(`git grep -l org-multi-org-director -- '*_test.go' tests/ scripts/ .github/` は 0 件。`tests/test-no-loop-references.sh` は `docs/specs/` を除外している)
- F4 のコードからの推定(FR-2、FR-3、受け入れ条件 2・3、NFR 安全の不変条件の注): `StopAll`(`internal/org/verbs_all.go:133`)は 148 行で active でない座席を飛ばす。`activeEvents`(`internal/org/seat.go:134`)は `spawn_started`・`spawn_step`・`spawned` だけで `spawn_failed` を含まない。`TotalActiveSeats`(`internal/org/reserve.go:511`)は active の座席だけを数え、`RunningOrgs`(同 464 行)は閉じていない workspace と予約でも org を数える。注の「はず」はこの 4 つと合う。再現はしていない(表も仕様もそう書く)
- 新しい tech-debt の行(327 行): `docs/evidence/org-trial-2026-10-20/watch.log` の `failed to ALERT leader` は 10 行で、`scope_change` 7、`liveness` 2、`stall` 1、すべて `bash: timed out: context deadline exceeded`、org 別では doctor-escape 7、report-escape 3。`sendAlert`(`internal/org/watch.go:1197`)は `callWithTimeout` の下で `SendWatchdogAlert` を打ち、失敗を stderr に書いて進み、送れたかどうかにかかわらず deadman の記録を足す。`defaultDriverCallTimeout` は `internal/org/verbs.go:744` で 10 秒。行の「FR-3 stays `[ ]`」は今の印と合う
- 43bf5c6e の語単位の差は、Status 20 行の日付、22 行の更新の対象(「この節の数と所見の一覧」)と PR の指し方、根拠の表の F4 行への受け入れ条件 3 の追加、ALERT 行と「確かめていないこと」から F7 を候補から外す文、tech-debt 327 行の Trigger だけだった。self-review の N1・N4・N5 の勧めと合う
- 表と仕様の注が挙げる証拠の節(試行の Two orgs、Merges、Gates、Stop and restart の Part 1・Part 2、The director's handling of the inbox、Approval wait per feature plan、Interventions、`org-watch-gate-live` の「3. `kill -9` のあと」、`org-feature-worktree-live` の Run 3、`org-plan-approval-live` の (a) と (e)、`director-wake-livefire` の Run 1 と What this run does not cover、`director-launch-cmd-livefire` の Run 3)は、どれもその見出しで見つかる。4 段目の計画(`docs/plans/archive/2026-10-09-org-feature-worktree.md` 31 行)は「`rejected` は書かない」と書き、受け入れ条件 3 の注と合う。`/org` skill の 4 つの写しに「編成パターン」「Parallel」「Leaded」は 0 件(FR-11 の行)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(HEAD 82ba75b5) | rc 0、`All verifiers passed.` | `Language scope: changed (docs_only)`、言語パックは選ばれない。shellcheck、hook の `sh -n`、Codex hook の 3 つの guard、check-sync(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 5)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt README の計画の参照、branch secret scan(765bcc46..82ba75b5 clean) |
| `./scripts/secret-scan-branch.sh --strict` | rc 0、clean | |
| `git diff --check 765bcc46...HEAD` | rc 0、出力なし | |
| 追加行の U+FFFD と漏えいの型 | U+FFFD 0 件、16 桁以上の 16 進 0 件 | `/Users/` などの型の名前に当たる 4 件は、どれも self-review の報告が「0 件だった」と書く型の名前で、値ではない |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-21-org-spec-complete.md` | `98781cfa6662` | `- Approved:` 行と同じ。Progress の追記(c56b70a4、82ba75b5)は digest に入らない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 仕様の行番号を指すほかの文書 | 合う | reports・evidence・archive の外で 0 件。この PR は仕様の先頭に 23 行足したが、壊れる参照はない |
| `docs/tech-debt/README.md` の行と仕様の印 | 合う | 仕様を名指す行は 12 行(172・200・203・207・209・210・213・226・234・255・299・327 行)。327 行のほかは FR の番号か節を指すだけで、印には頼らない。327 行の「FR-3 stays `[ ]`」は今の印と同じ |
| `docs/specs/2026-08-01-org-runtime.md` の「2026-10-07 後継の仕様」 | 合う | 「その仕様の各段が入った時点でそちらの記述が優先する」で、印にも Status にも頼らない。印の変更は Non-goals |
| 計画の AC の checkbox | 遅れている(V3) | AC1〜AC8 と AC6b の 9 つが `[ ]` のまま。下の Findings |
| 計画の Progress のずれの記録 | 一部遅れている(V1) | FR-3 の追加は 129 行にあるが、FR-4 と受け入れ条件 5 の追加は記録がない。下の Findings |

## Observational checks

- 根拠の表の判定のうち、所見と食い違うかどうか(F3、F4、F7、F10、F13、D-F3、ALERT)は人の読みで決まる。ここでは、表の文と仕様の注が互いに合うこと、Status の一覧が「はい」の行の和集合であること、試行の記録に表が引く事実の語があること(`GOFLAGS=-timeout=30m` 5 件、`herdr agent rename` 2 件、`agent_not_found` 2 件、`spawn_failed` 5 件、`AGMSG_SELF_NAME=off` 7 件)を確かめた。試行の本文の Findings を全部読んで、判定そのものを確かめ直してはいない
- 「印の規則」は、回り道が要った項目を「食い違う」に入れる。F4 で leader が残った pane を手で閉じた件を、表は FR-4 と受け入れ条件 5 の `[ ]` の理由にしており、この規則の当て方と合う

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | INFO | 計画の見込みの `[ ]`(36・37 行)は 9 項目で、Progress 129 行が足したのは FR-3 だけと書く。実際の `[ ]` は 12 項目で、FR-4 と受け入れ条件 5 も最初のスライス(1a5bc2ee の表、bc9fe9e0 の仕様)から `[ ]` だった(F4 で implementer を立てるのに回り道が要ったため)。この 2 つのずれは Progress に記録がない。AC3 は表の判定との一致を言い、見込みは AC を縛らないので、可否は変わらない。あわせて、AC3 の文は `[ ]` の注が「Open questions の問いを指す」と言うが、印の規則(28・30 行)は、所見と食い違う項目には所見と tech-debt の行を、それ以外の項目には Open questions を指させる。仕様は規則に従っており、10 項目は tech-debt を、2 項目は Open questions を指す | 計画 36・37・129 行、81 行(AC3)、28・30 行。`git show 1a5bc2ee:docs/evidence/org-spec-evidence-2026-10-21.md` の 78 行(`[ ]` が 11、FR-4 と受け入れ条件 5 を含む) | `/sync-docs` か `/pr` で、Progress に「見込みの `[ ]` に FR-4 と受け入れ条件 5 も足した(F4 の pane を手で閉じた回り道)。`[ ]` の注は印の規則に従い、所見と食い違う項目は tech-debt を指す」の 1 行を足す。Progress の行は digest に入らない。AC の文は直さない(digest が変わる) |
| V2 | INFO | 根拠の表 27 行(F8)の tech-debt の行の引用 `` "~~`scripts/check-sync.sh` fails the gate of every feature org~~" `` は、閉じの `~~` を「org」の直後に置く。register の 320 行は打ち消しが文の先まで続くので、引用は行の先頭と字面では一致しない。`~~` を除けば 320 行の 1 行だけに一致し、行は特定できる | 表 27 行、`docs/tech-debt/README.md:320`。tech-debt の行の照合の結果(31 か所中 30 か所が字面で一致) | 対応不要。系列の PR で表を更新するときに、引用の閉じを外すか「(RESOLVED の行)」だけで足りる |
| V3 | INFO | 計画の AC の checkbox(75〜83 行)が 9 つとも `[ ]` のまま。この報告が AC1〜AC7 と AC6b を満たすと確かめ、AC8 は静的な半分を確かめた | 計画 75〜83 行 | `/sync-docs` か `/pr` で、この報告と test の報告をもとに印を付ける。checkbox の印は digest に入らない |

## Coverage gaps

- テストは回していない(`/test` の担当)。AC8 のテスト部分(`run-verify.sh` の既定の all)と、計画の Test plan の `go test ./...` の比較も `/test` に任せる
- 表のテストが項目の文のどこまでを確かめるかは見ていない。見たのは名前と置き場所の実在、それに F4 の推定、ALERT の行、受け入れ条件 3 の注に関わるコードだけ(表の「確かめていないこと」の最後の項目と同じ限界)
- F4 の座席を `stop --all` が止めないこと、`max_total_seats` が数えないことは、コードの読みで合うと確かめただけで、再現していない
- self-review の報告は c56b70a4 の版の再レビューで、43bf5c6e と 82ba75b5 はその後のコミット。43bf5c6e は語単位の差を読み、N1・N4・N5 の勧めの範囲に収まることを確かめた。self-review を回し直してはいない
- insight event はこの報告と同じコミットで `docs/insights/events/2026-10-21-org-spec-complete.jsonl` に足す(self-review は足していないので、このファイルは verify の行から始まる)
- 決定的な verifier は足りている。印と根拠の表の照合(この報告で使った印と判定の照合、テスト名の照合)は、系列の PR が印を更新するたびに要る。小さな検査として、仕様の `^- \[[ x]\]` の行と根拠の表の判定の列を突き合わせる script を `tests/` に足せるが、表の形がまだ系列の PR で変わりうるので、今回は提案に留める

## Verdict

- Verdict: pass(CRITICAL・HIGH・MEDIUM・LOW はない。INFO 3 件(V1〜V3)は計画の Progress と AC の checkbox の遅れ、根拠の表の引用の字面で、どれも `/sync-docs` か系列の PR で直せる)
- Verified: AC1(Status の中身、28 本の PR が MERGED、タグ)、AC2、AC3(31 項目の印と判定の一致、数、`[ ]` の注)、AC4、AC5(指す先の入れ替わりなし)、AC6(31 行、テスト名 103 か所の照合、パスの実在、NFR 入れ方の 25 本)、AC6b(18 行、「はい」の和集合、結びつく項目の印、tech-debt の行の照合)、AC7(正規化した語単位の差)、静的解析(run-static-verify rc 0、strict secret scan、`git diff --check`、U+FFFD)、文書のずれ(行番号の参照、tech-debt、後継の仕様の注)、計画の digest
- Partially verified: AC8(静的な半分だけ)、所見と食い違うかどうかの判定(表と注の整合と試行の記録の語の有無だけ)
- Not verified: テストの実行、表のテストが項目の文のどこまでを確かめるか、F4 の推定の再現
