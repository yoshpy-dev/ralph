# Verify report: org-plan-approval

- Date: 2026-10-13(cycle 2 の 2 回目)
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(`Status: Approved`。`./scripts/plan-visual.sh digest` の出力 951cf09bd7bc は `- Approved:` 行の digest と一致する。ffd6b3ec から計画は変わっていない)
- Verifier: verifier subagent (Claude)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2、上限は既定の 2)の、verify の 2 回目。/test cycle 2(1b4ac934、pass)の mutation の生き残り Q07・Q08 を閉じる 6a7d35b0 と、self-review cycle 2 の 3 回目(9ccc830c、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 6、Merge: yes)のあとに走らせた。cycle 2 の 1 回目(HEAD 8fe7ba73)と cycle 1 の結果は、末尾の 2 つの付録に残した
- Scope: 前の verify(HEAD 8fe7ba73)からの差分 `git diff 8fe7ba73..HEAD`(HEAD 9ccc830c)。コードの変更は 6a7d35b0 の `internal/org/approval_enforce_test.go`(+86/-0、テスト 2 本)だけで、ほかのコミット(ffd6b3ec、1b4ac934、9ccc830c)は報告、evidence、insight の行だけである。非テストのコード、計画、ほかの文書は変わっていない(Static analysis の差分の範囲の行)。テストは走らせていない(/test の担当)
- Evidence:
  - `docs/evidence/verify-2026-10-13-org-plan-approval-c2r2-static.txt`: `./scripts/run-static-verify.sh` の出力と、`gofmt -l internal/`、`go vet ./internal/...`、`check-sync.sh`、`check-skill-sync.sh`、計画の digest、差分の範囲の確認
  - wrapper が自分で書いた `docs/evidence/verify-2026-10-13-011931.log` は gitignore の対象なので、コミットしない
  - insight event: `docs/insights/events/2026-10-13-org-plan-approval.jsonl` に verify の 1 行(`--cycle auto` で cycle 2、verdict pass、low 7 は D6〜D12)を足し、この報告と同じコミットに入れた

## Latest run: cycle 2 の 2 回目(HEAD 9ccc830c)

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | ブランチの差分に `scripts/ralph-config.sh` があるので、前の回と同じく full fallback で golang の pack を選んだ。`gofmt: ok`、`go vet` と staticcheck は出力なし、golangci-lint は `0 issues.`。shellcheck、hook の `sh -n`、guard の awk の構文、settings の `jq -e`、Codex の guard 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照の検査、branch の secret scan(225dff27..9ccc830c、clean)も通った |
| `gofmt -l internal/` | 出力なし(rc 0) | |
| `go vet ./internal/...` | 出力なし(rc 0) | |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5(前の回と同じ) |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `./scripts/plan-visual.sh digest <plan>` | 951cf09bd7bc | `- Approved:` 行(4 行目)と一致 |
| `git diff --stat 8fe7ba73..HEAD`(テストのファイル 1 つと `docs/reports`・`docs/evidence`・`docs/insights` を除く) | 空 | 6a7d35b0 のほかに、コードと文書の変更はない。6a7d35b0 は 1 ファイル +86/-0 |
| `./scripts/secret-scan.sh --file` で evidence のファイル | rc 0 | コミットの前に確かめた |

### Spec compliance の差分

非テストのコードが 8fe7ba73 から変わっていないので、AC ごとの判定は cycle 2 の 1 回目(付録)のまま変わらない。増えたのは、AC5 の 2 つの枝を固定するテスト 2 本である。行番号は HEAD 9ccc830c のもの。2 本が通ること、Q07・Q08 の mutant で落ちることは、読んで組み立てただけで、確かめるのは /test である。

| テスト | 固定するもの | 計画との対応 |
| --- | --- | --- |
| `TestOrgSend_ApprovalCheck_AfterTheWaitReadsTheManifestAgain`(`internal/org/approval_enforce_test.go:524`) | 待ったあとの 2 回目の検査(`internal/org/verbs.go:394`)が manifest を読み直すこと(`checkSendApproval` の `o.Manifest.Read()`、`:502`)。待つ間に計画が改訂されて改訂版が承認され(`plan_approved` の e2)、2 行が書かれると、最初の承認を名指す TASK は改訂版の digest に案内する TaskHeader の拒否になり、何も打たない。typed と raw の 2 周。待つ前に読んだ manifest を使う版(Q08)では最新の承認が最初の digest のままなので、`approval.go:502` の条件が偽になり、PlanChanged の拒否(次の手は改訂を戻す)が残る | AC5 の「承認のあとに計画が変わった」と「承認し直したあとは新しい digest の `PLAN:` で送れる」、`--raw` の TASK。Test plan の Edge cases の「`--raw` の TASK」「承認のあとの計画の改訂と承認し直し」と、Evidence の mutation(検査の条件)。cross-review cycle 1 の #1(triage の報告の 27 行、待ったあと新しい状態で検査し直す)の直しを固定する |
| `TestOrgSend_ApprovalCheck_LatestApprovalIsPerPlanPath`(`:563`) | `latestApprovedDigest`(`approval.go:575`)が、TASK の計画のパスの最新の承認を返すこと。org にもう 1 つの計画(audit)があり、その承認が最後に記録された台帳で、(a)(`:582`)承認し直した計画の古い digest の TASK は改訂版の digest への案内になる(`approval.go:502` の呼び出し)。(b)(`:587`)承認のない計画を名指す TASK の文に「(its latest approval is of ...)」が付かない(`approval.go:563-564` の呼び出し)。2 つの subtest が 2 つの呼び出しを 1 つずつ固定する。パスを見ない版(Q07)では、どちらの subtest も落ちる | AC5 の「承認の記録がない」と「承認し直したあと」。Test plan の Unit tests の「判定の関数(記録なし、digest 違い)」と、Edge cases の「承認し直し」。cross-review cycle 1 の #2 と self-review の N7 の規則(最新の承認にだけ案内する)を、計画が 2 つある org でも固定する |

- AC14 の固定は、この 2 本では増えない。2 本の改訂は本文の行(`Add the login form.` を `Add the signup form.` に替える)で、`PlanDigest` が読む場所である。AC14 の 3 つの場所は、cycle 2 の 1 回目の表のとおり既存のテストが見る
- AC5 の「何も記録せず」: 1 本目は打たないこと(`pane_send_text`、`pane_send_keys` がない)だけを見て、manifest と receipt のバイト列は比べない(待つ間に自分で e2 を足すため)。2 回目の検査で断ったときは、拒否の種類によらず `verbs.go:394-395` の同じ return を通るので、記録しないことは、バイト列を比べる既存の `TestOrgSend_ApprovalCheck_AgainAfterTheWait`(`:319`)と同じ道で成り立つ。これは読んで判断した
- AC11: `approval_enforce_test.go` は base(225dff27)にないファイルで(`git cat-file -e` が失敗する)、6a7d35b0 は追加だけ(-0)なので、base からあるテストは書き換えていない
- 2 本の doc が挙げる参照は合う。Q07・Q08 は /test の報告の cycle 2 の mutation の表(55 行、61 行)と生き残りの表(75 行、76 行)にあり、説明が一致する。「cross-review cycle 1 #1」は triage の報告の #1 である
- FR-7 と、仕様の受け入れ条件の 109・110 行との対応は変わらない

### Documentation drift

D6〜D10 は、付録の cycle 2 の 1 回目の表のまま /sync-docs に渡す。6a7d35b0 は、どれの指す場所にも触れていない。今も `templates/base/docs/quality/quality-gates.md:80` は「whose current version is approved」のままで(D6)、`internal/org/verbs.go:375` に「budget check right after that wait」が残り(D10 の C2-6)、`docs/tech-debt/README.md` の 213〜219 行に C2-6・C2-7 はない(D10)。新しいずれは次の 2 つで、どちらも記録の追加である。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D11 `docs/tech-debt/README.md`(self-review cycle 2 の 3 回目の C2-9・C2-10) | いいえ | 9ccc830c の報告の「Tech debt identified」の行が README にない(213〜219 行に C2-9・C2-10 は 0 件)。(a) は `approval_enforce_test.go` の再承認の手書きが 4 か所になったこと、(b) は `LatestApprovalIsPerPlanPath` の (b) の 2 つ目の assertion が 1 つ目より先に落ちないこと。報告は「N4 の行(219 行)への追記でもよく、C2-8 の編集と 1 回にまとめられる」と書くので、D10 と同じ編集で入れる |
| D12 計画の Progress checklist(digest の外) | 遅れている | D9 に加えて、/test cycle 2(1b4ac934、mutation の生き残り Q07・Q08)、それを閉じた 6a7d35b0(テスト 2 本)、self-review cycle 2 の 3 回目(9ccc830c)の記録がない。最後の記録は 194 行で、計画に `6a7d35b0`・`Q07`・`Q08` は 0 件。D9 と 1 行にまとめてよい |

情報: /test の報告(1b4ac934)は Q07・Q08 を「生き残り」と書く。6a7d35b0 より前の回の記録なので直さず、次の /test が 2 本で閉じたことを記録する。

### Coverage gaps

- テストは走らせていない(/test の担当)。2 本が HEAD で通り、Q08・Q07 で落ちるという判断は、この報告で辿った道筋と、self-review の 3 回目が probe の evidence(`docs/evidence/test-2026-10-13-org-plan-approval-c2-mutation.txt`)とテストを比べた結果だけに基づく
- 付録の cycle 2 の 1 回目の Coverage gaps は、どれも残る。承認が 1 回で先頭の 2 行が未記入のまま、打ち間違えた digest の TASK を送る入力は、2 本のどちらも扱わない

## Verdict

- Verdict: pass
- Verified: 6a7d35b0 はテストの追加だけで、非テストのコード、計画、ほかの文書は 8fe7ba73 から変わっていない。AC ごとの判定は cycle 2 の 1 回目のまま変わらず、AC5 の 2 つの枝(待ったあとの検査が新しい台帳を読むこと、最新の承認を計画のパスごとに選ぶこと)にテストの固定が増えた。静的解析(changed scope、full fallback)、`gofmt -l internal/`、`go vet ./internal/...`、`check-sync.sh`、`check-skill-sync.sh` は rc 0 で、計画の digest は 951cf09bd7bc のまま一致する
- Partially verified: AC11(base のテストは書き換えていない。通るかは /test)、AC12(D6 を /sync-docs で直す。D7・D8 は任意、D9〜D12 は記録の追加)、AC5 の「herdr も呼ばずに」(D9 のまま)
- Not verified: テストの実行(2 本と Q07・Q08 の mutant を含む)、2 回目の検査と案内の規則のバイナリでの確認、leader の図解ページと承認の手順の実機、Linux、release のノートと PR の本文

## 付録: cycle 2 の 1 回目の結果(HEAD 8fe7ba73 の時点)

cycle 2 の 1 回目の header(Date から Evidence まで)と本文を、そのとき書いたまま残す。`## Latest run` の見出しを外し、本文の見出しに「(cycle 2 の 1 回目)」を付け、`## Verdict` は「cycle 2 の 1 回目の判定」に、`- Verdict:` の行は `- cycle 2 の 1 回目の判定:` に変えた。本文の行番号は HEAD 8fe7ba73 の時点のもの。本文の「末尾の「付録: cycle 1」」は、この下の付録を指す。

- Date: 2026-10-13(cycle 2)
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(`Status: Approved`。`./scripts/plan-visual.sh digest` の出力 951cf09bd7bc は `- Approved:` 行の digest と一致する。b1f934c9 からの計画の変更は Progress checklist の 1 行だけ)
- Verifier: verifier subagent (Claude)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2、上限は既定の 2)。cross-review cycle 1 の ACTION_REQUIRED 2 件の直しと、self-review cycle 2 の 2 回目(8fe7ba73、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 4、Merge: yes)のあとに走らせた。cycle 1 の結果は末尾の「付録: cycle 1」に残した
- Scope: 仕様適合、静的解析、文書の整合。重点は cycle 2 の差分 `git diff b1f934c9..HEAD`(HEAD 8fe7ba73、7 コミット、16 ファイル、+640/-86)のうち、3563129d(send の待ったあとの確かめ直しと、古い `PLAN:` の案内の規則)、2d238702(テスト 3 本)、5e809ac6(spawn と send の `--help`)、edee787f(コメントと tech-debt の N4 の行)。非テストのコードで変わったのは `internal/org/verbs.go`、`internal/org/approval.go`、`internal/cli/org.go` の 3 つだけである。テストは走らせていない(/test の担当)。HEAD のビルド(`go build -o bin/ralph ./cmd/ralph`、`bin/` は gitignore の対象)で spawn と send の `--help` を読んだ
- Evidence:
  - `docs/evidence/verify-2026-10-13-org-plan-approval-c2-static.txt`: `./scripts/run-static-verify.sh` の出力
  - `docs/evidence/verify-2026-10-13-org-plan-approval-c2-sync.txt`: `gofmt -l`、`go vet ./...`、`check-sync.sh`、`check-skill-sync.sh`、計画の digest、skill の 4 面の sha256、写しの `cmp`、quality-gates の 2 面の該当行
  - `docs/evidence/verify-2026-10-13-org-plan-approval-c2-help.txt`: HEAD のビルドの `ralph org spawn --help` と `ralph org send --help` の説明の部分
  - wrapper が自分で書いた `docs/evidence/verify-2026-10-13-002026.log` は gitignore の対象なので、コミットしない
  - insight event: `docs/insights/events/2026-10-13-org-plan-approval.jsonl` に verify の 1 行(`--cycle auto` で cycle 2、verdict pass、low 5 は D6〜D10)を足し、この報告と同じコミットに入れた

### Spec compliance(cycle 2 の 1 回目)

行番号は HEAD 8fe7ba73 のもの。テストは名前と位置だけを示し、通るかどうかは /test で確かめる。

cycle 2 が非テストのコードで変えたのは次の 3 つである。

1. `Send` は、宛先の座席の手が空くのを待ったあと(`AgentWait`、`internal/org/verbs.go:386`)、`--timeout-ms` の残りの検査(`:399-426`)と `PaneSendText`(`:428`)の前に、承認の検査をもう一度回す(`:390-397`)。manifest を読み直す `checkSendApproval`(`:501`)を、dry-run の枝の前の 1 回目(`:334`)と同じく呼ぶ
2. `checkTaskPlanApproved`(`internal/org/approval.go:487`)が古い `PLAN:` の TASK を今の digest に案内するのは、ファイルの今の digest がそのパスの最新の承認(`latestApprovedDigest`、`:575`)のときだけになった(`:502`)。今の digest が「承認のあとに計画が変わった」で断られるときは、その拒否を返す(`:511-513`)
3. spawn と send の `--help` に承認の拒否を書いた(`internal/cli/org.go:531-537`、`:898-904`)

`internal/org/spawn.go`、`approve.go`、`reserve.go`、`feature.go`、`internal/config/`、雛形の `internal/org/prompts/`、`scripts/`、`.claude/rules/`、仕様、README、AGENTS.md は cycle 2 で変わっていない(`git diff --stat b1f934c9..HEAD` が空)。

| Acceptance criterion | Status | Evidence(cycle 2 で見たこと) |
| --- | --- | --- |
| AC1 `[org.approval].mode` の既定・検証・3 面 | 満たす(変わらない) | 設定の 3 面と `Load()` は cycle 2 で変わっていない。`scripts/ralph-config.sh` の 2 面は `cmp` で一致する |
| AC2 approve の記録・resolve・出力 | 満たす(変わらない) | `internal/org/approve.go` と `internal/cli/org_approve.go` は cycle 2 で変わっていない |
| AC3 approve の拒否と all-or-nothing | 満たす | approve のコードは変わらない。2d238702 の `TestOrgApprove_RefusesAnItemItCannotResolve`(`internal/org/approve_test.go:299`)は、resolve の note(`approved <パス> sha256:<digest>`)が `InboxNoteMaxRunes`(500)を超える件が記録の前に断られ(理由の文は `approveCandidates` の `approve.go:264`)、manifest と受信箱のバイト列が変わらないことを見る。全部か無しかのうち、resolve できない件の端に当たる。複数の ID は cycle 1 からの `TestOrgApprove_AllOrNothing`(`:259`)が見る |
| AC4 座席の pane からの approve | 満たす(変わらない) | `approveTripwire` は変わっていない |
| AC5 per-plan の implementer への TASK | 満たす(D9 の読みで) | 1 回目の検査の位置は変わらない。見張りの検査(`verbs.go:329`)、承認の検査(`:334`)、dry-run の枝(`:344`)の順で、計画の Scope 40 行のとおりである。この順は 2d238702 の `TestOrgSend_WatchGateBeforeApprovalCheck`(`internal/org/approval_enforce_test.go:519`)が、typed・raw・dry-run・raw の dry-run の 4 通りで固定する。2 回目の検査は Scope にない追加で、cross-review cycle 1 の #1 の直しである。`PaneSendText` より前にあるので、待つ間に変わった計画の TASK は、何も打たず何も記録せずに断られる(`TestOrgSend_ApprovalCheck_AgainAfterTheWait`、`:319`。digest が変わる編集と Progress のメモを断り、編集なしと checkbox の印は送る。typed と raw の両方)。案内の規則を変えても、`checkTaskPlanApproved` が nil を返すのは TASK 自身の digest が照合を通るとき(`approval.go:492-495`)だけで、ほかの枝はどれも拒否を返す。承認し直したあとの新しい digest の TASK は、cycle 1 からの `TestOrgSend_ApprovalCheck_PassesAndApprovedAgain`(`:243`)のとおり通る。AC5 の「herdr も呼ばずに」は、呼んだ時点で拒否の理由に当たる TASK について成り立つ(1 回目で断られ、`AgentWait` に届かない)。2 回目で断られる TASK は、herdr の `agent wait` を 1 回呼んだあとになる(打たない、記録しない)。Scope 40 行の「herdr も agmsg も呼ばない」とのこの差は Progress に書かれていない(D9) |
| AC6 承認の要らない TASK | 満たす | 2 回目も同じ `sendApprovalErr`(`verbs.go:519`)を通るので、結びつきのない org、split、トークンのない記録、reviewer への TASK は、どちらの位置でも通る。TASK でない送信は `isTask`(`:328`)が偽なので、2 回目を回さない |
| AC7 implementer の spawn | 満たす | `spawn.go` は変わっていない。`spawnFeatureSeatErr` は 3 か所とも `spawnCapacityErr` のあとにある(dry-run は `spawn.go:542` のあと `:546`、`spawnPrecheckErr` は `:1189` のあと `:1192`、ロックの下は `:1222` のあと `:1227`)。Scope 41 行のとおりである。2d238702 の `TestOrgSpawn_CapacityCheckBeforeFeatureOrgChecks`(`approval_enforce_test.go:543`)が、dry-run と本番で、容量の拒否が承認の拒否より先に返ることを固定する |
| AC8 leader のいない org の spawn | 満たす(変わらない) | leader の検査は承認の検査と同じ `spawnFeatureSeatErr`(`spawn.go:1282`)の中にあり、容量の検査との順は AC7 の呼び出しの位置で決まる。上のテストの fixture は leader が動いているので、観測できるのは承認の側だけである(self-review の C2-4) |
| AC9 前の命の承認 | 満たす | `latestApprovedDigest` に渡すのは `currentPlanApprovals`(`approval.go:365`)の結果なので(`:502`、`:554`、`:563`)、新しい案内の規則も前の命の承認を最新と読まない |
| AC10 leader のタスクと雛形 | 満たす(逸脱 2 の読みで。変わらない) | 雛形は変わっていない。手順 5 の「最後に承認された digest」(`internal/org/prompts/leader.md`)と、最新の承認にだけ案内する新しい規則が、同じ digest を言うようになった |
| AC11 既存のテスト | 一部(通るかは /test) | cycle 2 が触ったテストのファイル 3 つ(`approval_enforce_test.go`、`approval_test.go`、`approve_test.go`)は、どれもこのブランチで足したもので、base(225dff27)にはない。base からあるテストは書き換えていない。`approval_test.go` で消えた 1 行は doc のコメントの行である |
| AC12 文書と同期の検査 | 一部(D6 を /sync-docs で直す) | `check-sync.sh` は rc 0(DRIFTED 0)、`check-skill-sync.sh` は rc 0。skill の 4 面は同じ sha256(e9e4b1d5…)、agent-messaging の 2 面は `cmp` で一致する。template の quality-gates の行が直す前の文のまま残る(D6) |
| AC13 start が記録する方式 | 満たす(変わらない) | `reserve.go`、`feature.go`、`internal/cli/org.go` の start と `withMainWorktreeOrgLimits` は cycle 2 で変わっていない |
| AC14 digest が読まない場所の変更 | 満たす | 全文のハッシュの照合は変わらない。2 回目の検査は、待つ間に Progress checklist に足した行も断り、checkbox の印だけなら送る(`TestOrgSend_ApprovalCheck_AgainAfterTheWait` の 2 ケース)。古い `PLAN:` の TASK で、今のファイルが最新の承認の版に Progress のメモを足したものなら、今の digest の「承認のあとに計画が変わった」で断る(N6、`approval.go:511-513`。`TestCheckTaskPlanApproved_StaleOrMistypedDigest` の (h)、`internal/org/approval_test.go:1153`)。どの道でも断られることは変わらず、次の手が「その変更を戻す」になった |

#### 古い `PLAN:` の案内の規則(3563129d)の確認(cycle 2 の 1 回目)

`checkTaskPlanApproved`(`approval.go:487-517`)の枝をコードで追った。

- TASK の digest が照合を通る: nil(`:492-495`)
- 1 回目の拒否が NoApproval でも PlanChanged でもない(先頭の 2 行、そのほか): その拒否(`:496-498`)
- ファイルが読めない、今の digest が TASK と同じ、今の digest がそのパスの最新の承認でない: TASK 自身の拒否(`:502`)。TASK が最新の承認を名指すとき、今の digest は TASK と違うので最新ではなく、ここに入る(cross-review cycle 1 の #2 と N7。(f)、`approval_test.go:1123`)。ファイルが前に承認した版に戻っていて TASK の digest が打ち間違いのときも同じ((g)、`:1128`)
- 今の digest が照合を通る: TaskHeader の拒否で、今の digest を名指す(`:506-509`)
- 今の digest が先頭の 2 行か PlanChanged で断られる: その拒否(`:511-513`)。今の digest は最新の承認で記録があり、ファイルの digest そのものなので、PlanChanged になるのは全文のハッシュが違うとき、つまり digest が読まない場所が変わったときだけである(N6)
- そのほか: TASK 自身の拒否

`checkPlanApproved`(`:549-570`)の書き換えは、`latest` の求め方を `latestApprovedDigest` に移しただけで、`matching` の条件と拒否の文は前と同じである。spawn が使う `checkAnyPlanApproved`(`:650`)の結果は変わらない。skill の 4 面(案内の項目は 669〜696 行)、root の quality-gates(81 行)、雛形の手順 5 は、この規則と合う。template の quality-gates(80 行)は合わない(D6)。skill の 683 行の項目は入力を「承認し直す前の digest」とだけ書くが、承認が 1 回で先頭の 2 行が未記入のまま打ち間違えた digest の TASK を送った場合も、同じ枝に入る(D7、任意)。

#### 仕様 FR-7 との対応(cycle 2 の変更)(cycle 2 の 1 回目)

- 3 項目の「TASK の再投入や計画の改訂のときも同じ検査をする」: 2 回目の検査で、send が宛先を待つ間の改訂にも検査が届くようになった。検査と送信の間の窓は、待ちの長さ(`--timeout-ms` まで)から、2 回目の検査と `PaneSendText` の間に縮んだ。窓が残ることは、検査をその時点の判定とした計画の A2 のとおりである
- 1・2・4 項目、受け入れ条件の 109・110 行、Security considerations の読みは cycle 1 から変わらない。5・6 項目と FR-8 の「merge してよい順」は 7b に送ったままである
- 仕様の注(`docs/specs/2026-10-07-org-multi-org-director.md:72`)は 2 回目の検査に触れない(D8、任意)

#### 計画からのずれと Progress の記録(cycle 2 の 1 回目)

- Progress の 194 行は 3563129d、2d238702、5e809ac6 の中身を書き、コードと合う(#1 の確かめ直し、#2 と N7 の最新の承認だけへの案内、N6 の PlanChanged、tech-debt の 2 行の削除、R03〜R05 と R11 のテスト 3 本、D2 の help、計画の本文を変えていないこと)
- 書かれていないことが 2 つある(D9)。edee787f(コメントの指し先に cycle を付けたこと、2 つの doc の直し、N4 の行の 1 文)と、2 回目の検査で断られる TASK が herdr の `agent wait` のあとになること(Scope 40 行と AC5 の「herdr も呼ばない」の読み)である。`Send` の doc(`verbs.go:266-279`)は、2 回目の拒否を「nothing typed, nothing recorded」と書き分けていて、コードと合う
- 逸脱 2 の続き(189 行)の「今の承認済みの計画と違う digest なら、その digest に直す」は 3563129d より前の規則で、194 行が「最新の承認のときだけ」に絞ったと書く。その時点の記録としては合っている

#### cycle 1 の D1〜D5 の扱い(cycle 2 の 1 回目)

- D1(skill の `approve` の行の括弧): 7e025b45 で直った。4 面とも(180 行)「leader への承認の DECISION はまだ送らず、先に `ralph org inbox show <ID>` で resolve の note を読むよう言う」を含み、sha256 が同じ
- D2(spawn と send の `--help`): 5e809ac6 で直った。ビルドで読んだ文はコードと合う。spawn の文は、leader 以外の座席を leader が動いていなければ断ること(`spawnFeatureSeatErr` は seat id と role がどちらも leader の spawn だけを通す)、per-plan の implementer を最後の disband より後の承認が計画のファイルに合うまで断ること(`checkAnyPlanApproved`)、どちらも dry-run を含めて `rejected` を書くことを言う。send の文は、per-plan の implementer への TASK を raw と dry-run を含めて何も記録せずに断ること、文が見つけたこと・目印・次の手の順であること(`planNotApprovedError`)を言う。help を固定するテストはない
- D3(仕様の注の「本文」の読み): 7e025b45 で直った(`:72` の「上の要件の「TASK の本文の `PLAN:`」は、ヘッダの塊の 1 行 `PLAN: <path> sha256:<digest>` と読んだ」)
- D4(計画が挙げる受け入れ条件の行番号): 記録だけ。計画の本文は変えていない
- D5(Progress と AC のチェックボックス): まだ付いていない(下の表の D5)

### Static analysis(cycle 2 の 1 回目)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | ブランチの差分に `scripts/ralph-config.sh` があるので full fallback になり、golang の pack を選んだ。`gofmt: ok`、`go vet` と staticcheck は出力なし、golangci-lint は `0 issues.`。shellcheck、hook の `sh -n`、guard の awk の構文、settings の `jq -e`、Codex の guard 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照の検査、branch の secret scan(225dff27..8fe7ba73、clean)も通った |
| `gofmt -l internal cmd` | 出力なし(rc 0) | |
| `go vet ./...` | 出力なし(rc 0) | |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`docs/quality/quality-gates.md` は KNOWN_DIFF なので、D6 はこの検査では出ない |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `shasum -a 256` で org の skill 4 面 | 4 面とも e9e4b1d5469c… | |
| `cmp` で agent-messaging と `ralph-config.sh` の 2 面 | どちらも一致 | |
| `./scripts/plan-visual.sh digest <plan>` | 951cf09bd7bc | `- Approved:` 行と一致 |
| `go build -o bin/ralph ./cmd/ralph` | rc 0 | help を読むためのビルド |
| `./scripts/secret-scan.sh --file` で evidence の 3 ファイル | rc 0 | コミットの前に確かめた |

### Documentation drift(cycle 2 の 1 回目)

D6 から番号を振る(D1〜D5 は付録の cycle 1 と、上の「cycle 1 の D1〜D5 の扱い」で使った)。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D6 `templates/base/docs/quality/quality-gates.md:80`(「Feature plan approval」の行) | いいえ | root の `docs/quality/quality-gates.md:81` は 3563129d で「… of a plan whose current version is its latest approval and passes the check, never to an earlier approved version」に変わった。template の行は「… of a plan whose current version is approved」のままである。`ralph init` が配る面で、cycle 1 の時点では 2 つの行は同じ文だった。`scripts/check-sync.sh` はこのファイルを `KNOWN_DIFFS`(`:101`)に入れているので、rc 0 でも気づかない(self-review の C2-2)。template の 80 行目の句を root の 81 行目と同じ句にする |
| D7 `/org` skill の「機能の計画の承認」(4 面)(任意) | ほぼ合う | (a) 683 行の項目は「TASK の `PLAN:` が承認し直す前の digest で」と書くが、コードは、承認が 1 回で先頭の 2 行が未記入のまま打ち間違えた digest の TASK も同じ枝(今の digest の先頭の 2 行の拒否)に入れる(`approval.go:502-513`)。`checkTaskPlanApproved` の doc は edee787f でこの入力を挙げた。「承認し直す前の digest か打ち間違い」にする。(b) 655 行の send の項目は、send が宛先の手が空くのを待ったあと、打つ前にもう一度照合することを書かない(待つ間に計画が変われば、そこで同じ拒否になる)。拒否の文と次の手は 1 回目と同じなので、従って誤ることはない。直すなら 4 面とも同じにする |
| D8 仕様 FR-7 の注(`docs/specs/2026-10-07-org-multi-org-director.md:72`)(任意) | ほぼ合う | 「上の要件の「TASK の再投入や計画の改訂」にも同じ照合をかける」のあとに、send は宛先の座席を待ったあと打つ前にもう一度照合する、と足せる。注のほかの文(TASK の拒否は何も記録しない、など)は 2 回目の検査とも合う |
| D9 計画の Progress checklist(digest の外) | 遅れている | 194 行は 3563129d・2d238702・5e809ac6 を書くが、edee787f(C2-1、C2-3〜C2-5 のコメントと N4 の行の直し)と self-review cycle 2 の 2 回の実行を書かない。また、2 回目の検査で断られる TASK は herdr の `agent wait` を 1 回呼んだあとになる(打たない、記録しない)。Scope 40 行の「断ったときは manifest に何も書かず、herdr も agmsg も呼ばない」と AC5 の「herdr も呼ばずに」は、呼んだ時点で拒否の理由に当たる TASK について成り立つ、と読んだことを 1 行で残す |
| D10 `docs/tech-debt/README.md` | 一部 | self-review cycle 2 の 2 回目の「Tech debt identified」の行(C2-6 は `verbs.go:375` の「budget check right after that wait」、C2-7 は 2 つの doc の折り返し、(c) は C2-2 が直らなかったとき)が README にない。C2-8 の N4 の行(219 行)の Impact・Trigger・Related の句も足されていない。C2-6 と C2-7 は Go のコメントなので、上限の 2 では直さず行にする(self-review の Follow-ups)。D6 を直すなら (c) は行に入れない |
| D5 計画の Progress checklist と AC のチェックボックス(情報) | 遅れている | `- [ ] Review artifact created` 以下と AC1〜AC14 はまだ付いていない。193 行は「AC と checklist の印は cycle 2 で付ける」と書く。digest の外なので、/sync-docs か /pr で付ける |
| `/org` skill のほかの箇所(4 面) | 合う | 4 面の sha256 は同じ。「機能の計画の承認」の案内の項目(669〜696 行)は、TaskHeader・先頭の 2 行・PlanChanged・TASK 自身の拒否の 4 つの枝に 1 つずつ当たる(D7 の (a) を除く)。verb 表の `spawn`・`send`・`approve` の行(168、169、180)はコードと合う |
| `.claude/rules/ralph/agent-messaging.md:102-108` と template の写し | 合う | cycle 2 で変わっておらず、`cmp` で一致する。「otherwise `ralph org send` refuses it, with nothing recorded」は 2 回目の検査とも合う |
| `docs/quality/quality-gates.md:81` | 合う | 案内の句が、最新の承認にだけ案内する規則と合う |
| README の org runtime(`:124`、`:258`)と AGENTS.md の repo map | 合う | cycle 2 で変わっていない。cycle 2 の変更はどちらの記述も崩さない |
| `internal/org/prompts/leader.md` | 合う | 変わっていない。手順 5(最後に承認された digest)、手順 6(承認のあとにメモを書かない)、手順 7(目印の後ろの次の手に従う)は新しい規則と合う |
| `ralph org spawn --help`、`ralph org send --help` | 合う | 上の D2 のとおり |
| `Send` の doc(`verbs.go:266-279`、`:311-315`) | 合う | 2 回目の検査の位置と、そこで断ったときに打たず記録しないことを書く。本体のコメント(`:374-376`)の「budget check right after that wait」は古い(D10 の C2-6) |
| tech-debt の 213〜219 行 | 合う(D10 を除く) | N6 と N7 の行は消え、残る 7 行に N6・N7 の参照はない。218 行(N8)の「承認し直す前の digest の TASK は、新しい digest の先頭の 2 行の拒否になる」は、今の digest が最新の承認である枝として 3563129d のあとも成り立つ。219 行(N4)に足した 1 文の「6 か所」は `skip the tests` の数(2 つのファイルに 3 つずつ)と合う |

D6 は /sync-docs で直す。D9 と D10 は記録の追加で、/sync-docs か orchestrator が入れる。D7 と D8 は任意。D5 は /sync-docs か /pr で印を付ける。

### Observational checks(cycle 2 の 1 回目)

HEAD のビルドで spawn と send の `--help` を読んだ(D2)。cycle 2 は approve のコードを変えていないので、cycle 1 の approve の probe は取り直していない。2 回目の検査をバイナリで確かめるには、herdr が座席を待つ間に計画を書き換える必要があり、使い捨ての台帳では組まなかった。

### Coverage gaps(cycle 2 の 1 回目)

- テストは走らせていない(/test の担当)。2 回目の検査を消す mutant と `PaneSendText` の後ろへ動かす mutant が落ちること、(f)(g)(h) が直す前のコードで落ちること、2d238702 の 3 本が R03〜R05 と R11 で落ちることは、読んで組み立てただけである
- /test cycle 1 の実機確認(`docs/evidence/org-plan-approval-live-2026-10-13.md`)は 9dcdbfab のビルドで行った。そのあと非テストのコードは、`verbs.go`、`approval.go`、`internal/cli/org.go` が +116/-44 変わった(`git diff --stat 9dcdbfab..HEAD`)。雛形(go:embed)は変わっていない。2 回目の検査と新しい案内の規則は、ビルドしたバイナリでは確かめていない
- 承認が 1 回で先頭の 2 行が未記入のまま、打ち間違えた digest の TASK を送る入力は、読んだ範囲のテストにない(self-review cycle 2 の Coverage gaps と同じ)。通る枝は (e) が固定するものと同じである
- 2 回目の検査から `PaneSendText` までの窓は残る(計画の A2 のとおり、検査はその時点の判定)。spawn には確かめ直しがない(計画を名指さず、ロックの下で 1 回だけ検査する)
- help の文を読むテストはない
- Linux、別の herdr サーバーの pane の id が重なる場合、release のノートと PR の本文は、cycle 1 と同じく確かめていない

### cycle 2 の 1 回目の判定

- cycle 2 の 1 回目の判定: pass
- Verified: cycle 2 の差分のあとも、AC1〜AC10、AC13、AC14 はコードとテストの対応を読んで満たすと判断した。Send の検査の順(見張り、承認、dry-run)と spawn の順(容量、機能の org の検査)は計画の Scope 40・41 行のとおりで、2d238702 のテストがそれぞれを固定する。待ったあとの 2 回目の検査は打つ前にあり、断ったときは打たず記録しない。古い `PLAN:` の案内は最新の承認にだけ移り、どの枝でも拒否は拒否のまま残る。AC3 の全部か無しかは、resolve できない件の端までテストで固定された。cycle 1 の D1・D3 は 7e025b45、D2 は 5e809ac6 で直った。静的解析(changed scope、full fallback)、`gofmt -l`、`go vet`、`check-sync.sh`、`check-skill-sync.sh` は rc 0 で、計画の digest は 951cf09bd7bc のまま一致する
- Partially verified: AC11(cycle 2 は base のテストを書き換えていない。通るかは /test)、AC12(D6 を /sync-docs で直す。D7・D8 は任意、D9・D10 は記録の追加)、AC5 の「herdr も呼ばずに」(2 回目で断られる TASK は `agent wait` のあと。D9)
- Not verified: テストの実行、2 回目の検査と新しい案内の規則のバイナリでの確認、leader の図解ページと承認の手順の実機、Linux、release のノートと PR の本文

## 付録: cycle 1 の結果(HEAD 78b3e873 の時点)

cycle 1 の header(Date から Evidence まで)と本文を、そのとき書いたまま残す。本文の見出しは 1 段下げて「(cycle 1)」を付け、`## Verdict` は「cycle 1 の判定」に変えた。本文の行番号は HEAD 78b3e873 の時点のものである。D1〜D5 は、上の cycle 2 の「cycle 1 の D1〜D5 の扱い」で扱った。

- Date: 2026-10-13(cycle 1)
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(`Status: Approved`。`./scripts/plan-visual.sh digest` の出力 951cf09bd7bc は `- Approved:` 行の digest と一致する)
- Verifier: verifier subagent (Claude)。パイプラインの 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は 1)。self-review の 3 回目(48d7282f、CRITICAL 0・HIGH 0、Merge: yes)と、その LOW を tech-debt の行にした 78b3e873 のあとに走らせた
- Scope: 仕様適合、静的解析、文書の整合。対象はブランチ全体の `git diff 225dff27...HEAD`(HEAD 78b3e873、43 ファイル、+6832/-265)。テストは走らせていない(/test の担当)。HEAD のビルド(`go build -o bin/ralph ./cmd/ralph`、`bin/` は gitignore の対象)で、`ralph org approve` を使い捨ての台帳に対して動かした
- Evidence:
  - `docs/evidence/verify-2026-10-13-org-plan-approval-static.txt`: `./scripts/run-static-verify.sh` の出力
  - `docs/evidence/verify-2026-10-13-org-plan-approval-sync.txt`: `check-sync.sh`、`check-skill-sync.sh`、skill の 4 面の sha256、写しの `cmp`、digest
  - `docs/evidence/verify-2026-10-13-org-plan-approval-approve-probe.txt`: approve の実機確認のスクリプトと出力
  - `docs/evidence/*.log` は gitignore の対象なので、コミットする evidence は `.txt` にした。wrapper が自分で書いた `docs/evidence/verify-2026-10-12-210514.log` はコミットしない
  - insight event: `docs/insights/events/2026-10-12-org-plan-approval.jsonl` に verify の 1 行(`--cycle auto` で cycle 1、verdict pass、low 3 は D1〜D3)を足し、この報告と同じコミットに入れた

### Spec compliance(cycle 1)

行番号は HEAD 78b3e873 のもの。テストの関数は名前と位置だけを示し、通るかどうかは /test で確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `[org.approval].mode` の既定・検証・3 面 | 満たす | 定数と既定は `internal/config/config.go:166` と `:228`、`Load()` の検証は `:432`(空文字も拒否し、補わない)。3 面は `templates/base/ralph.toml:130-131`、`scripts/ralph-config.sh:103`(template の写しと `cmp` で一致し、export しない)で、`internal/config/defaults_sync_test.go:177` が固定する。テストは `TestDefault_OrgApproval`(`config_test.go:904`)と `TestLoad_OrgApproval`(`:917`。節なし、キーなし、per-plan、split、空、未知、大文字、前後の空白)。設定の値を読むのは `featureApprovalMode`(`internal/org/feature.go:345`)だけで、`withMainWorktreeOrgLimits`(`internal/cli/org.go:176`)が main worktree の値を写す。ほかに `Approval.Mode` を読む非テストのコードはない(grep) |
| AC2 approve の記録・resolve・出力 | 満たす | `Org.Approve`(`internal/org/approve.go:96`)。照合と追記は同じロックの下(`recordUnderLock`、`:146`、`recordApprovals`、`:306`)、resolve は `resolveApproved`(`:386`)、DECISION の文は `ApprovalDecisionText`(`:353`)、CLI の出力は `printApprovedPlans`(`internal/cli/org_approve.go:102`)、Details は `planApproval.details`(`internal/org/approval.go:311`)。テストは `TestOrgApprove_RecordsResolvesAndLeavesThePlan`(`approve_test.go:43`)、CLI の `TestOrgApprove_RecordsResolvesAndPrintsTheDecision`(`org_approve_test.go:155`)と `TestOrgApprove_TheDecisionCommandRunsInAPOSIXShell`(`:215`)。実機でも、stdout の 2 行と DECISION のコマンド、manifest の 1 行、件の resolve(note `approved <path> sha256:<digest>`)、計画のファイルの sha256 が変わらないことを確かめた |
| AC3 approve の拒否と all-or-nothing | 満たす | 受信箱だけで決まる検査は `approveCandidates`(`approve.go:237`)、台帳とファイルの検査は `checkApproveCandidate`(`:276`)、パスは `normalizePlanPath` と `resolvePlanFile`(`approval.go:144`、`:168`)、まとめての拒否は `approveRefusals`(`approve.go:169`)。`TestOrgApprove_Refusals`(`approve_test.go:161`)は 17 ケース(知らない ID、resolved、BLOCKED、PLAN なし・2 行・形違い、絶対パス、`..`、結びつきなし、plain な予約、disband 済み、不完全な結びつき、外を指す symlink、ないファイル、ディレクトリ、QUESTION のあとの変更)で、manifest と受信箱のバイト列が変わらないことまで見る。ほかに `TestOrgApprove_AllOrNothing`(`:259`)、`TestOrgApprove_IDList`(`:293`)、CLI の `TestOrgApprove_OneBadItemRecordsNothing`(`org_approve_test.go:293`)。実機で `e1 e2 e9` は 2 件の理由を並べて rc 1 になり、台帳の hash は変わらなかった |
| AC4 座席の pane からの approve | 満たす | `approveTripwire`(`approve.go:203`)は受信箱を読む前に走る(呼び出しは `:108`、受信箱の読み取りは `:111`)。`TestOrgApprove_Tripwire`(`approve_test.go:320`)は、受信箱を読めなくしても先に断ること、未設定・別の pane・止まった座席・dry-run の座席・disband した org の座席の pane は通ることを見る。CLI は `TestOrgApprove_RefusedFromAnActiveSeatsPane`(`org_approve_test.go:452`)。実機で leader の pane の値では rc 1 で台帳が変わらず、別の値では通って `herdr_pane=set` を記録した |
| AC5 per-plan の implementer への TASK | 満たす | 検査は見張りの検査のあと、dry-run の枝の前(`internal/org/verbs.go:326`)。`sendApprovalErr`(`:491`)、`checkTaskPlanApproved`(`approval.go:470`)、`checkPlanApproved`(`:529`)、`checkApprovedPlanFile`(`:558`)、`checkApprovedPlanHeader`(`:588`)。`TestOrgSend_ApprovalCheck_Refusals`(`approval_enforce_test.go:168`)は 9 つの理由を typed・raw・dry-run・raw の dry-run の 4 通りで送り、manifest・receipt・driver の呼び出し回数が変わらないことを見る。承認し直したあとの新しい digest は `TestOrgSend_ApprovalCheck_PassesAndApprovedAgain`(`:242`)、古い digest と打ち間違いは `TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest`(`:305`) |
| AC6 承認の要らない TASK | 満たす | `sendApprovalErr`(`verbs.go:491`)は、結びつきのない org、`RequiresApproval` が false の org、role が implementer でない座席を通す。`TestOrgSend_ApprovalCheck_NotApplied`(`approval_enforce_test.go:434`)が、reviewer への TASK、implementer への QUESTION と DECISION、split の org、トークンのない記録の org、結びつきのない org を、typed と dry-run で送る。仕様の受け入れ条件(今の 110 行)に当たる |
| AC7 implementer の spawn | 満たす | `spawnFeatureSeatErr`(`internal/org/spawn.go:1282`)を、dry-run(`:546`)、ロックの下の `checkCapacityAndStart`(`:1227`)、`spawnPrecheckErr`(`:1192`)の 3 か所で `spawnCapacityErr` のあとに呼び、拒否は `reject()` を通る。`TestOrgSpawn_ApprovalCheck_Implementer`(`approval_enforce_test.go:629`)は、dry-run の `rejected` と receipt が本番と同じ文であること、承認があれば spawned、reviewer は承認の前でも spawned、split は承認なしで spawned を見る。並びは `TestSpawnPrecheckErr_MatchesSpawn`(`feature_test.go:1246`)にも足された |
| AC8 leader のいない org の spawn | 満たす | `spawnFeatureSeatErr`(`spawn.go:1282`)は、seat id と role がどちらも leader の spawn だけを通し、ほかは active な leader がいなければ断る。`TestOrgSpawn_FeatureOrgNeedsItsLeader`(`approval_enforce_test.go:720`)は両方の方式で、seat id か role の片方だけが leader の座席も断ること、leader を立て直せば通ること、結びつきのない org は leader なしでも通ることを見る |
| AC9 前の命の承認 | 満たす | `currentPlanApprovals`(`approval.go:365`)は、最後の本物の `disbanded` より後の記録だけを読む。テストは `TestCheckPlanApproved` の `a record from the previous life (AC9)`(`approval_test.go:582`)、send(`approval_enforce_test.go:201`)、spawn(`:676`、本物の `Disband` のあと) |
| AC10 leader のタスクと雛形 | 満たす(逸脱 2 の読みで) | タスクの行は `featureLeaderTask`(`feature.go:390`)。雛形は `internal/org/prompts/leader.md:76-82`(方式ごとに飛ばす手順)と手順 1〜7(`:86`〜`:159`)。`TestRenderRolePrompt_Leader_FeatureOrgPlanApproval`(`prompts_test.go:578`)が、語句、手順の順、例の QUESTION を escalate と `ParsePlanHeader` が受け付けること、`ApprovalDecisionText` を手順 4 が承認と読めること、拒否の文に目印と `ralph org approve` があり `--raw` は TASK の拒否にだけあることを固定する。AC10 の「BLOCKED を escalate する」は、逸脱 2 で理由ごとの次の手に替わった(下の「計画からのずれ」)。既存の固定の語句のうち 2 つ(手順の番号)は書き換えた(AC11) |
| AC11 既存のテスト | 一部(書き換えの記録は合う。通るかは /test) | 各スライスのコミットで、消えた行(コメントを除く)を持つ既存のテスト関数を数え直した。S1 f779cc56 は 11 本(feature_test.go 4、reserve_test.go 3、spawn_feature_test.go 2、verbs_test.go 1、prompts_test.go 1)で、Progress の「11 本と共有の helper」と合う。S3 e3d34062 で行を消したのは `TestOrgSpawn_Feature_PlainReserveIntoBoundOrgRefused` だけで、feature_test.go の 2 本(`OrgIDAtTheLengthLimit`、`SlugAtTheLengthLimit`)は承認の記録を足しただけだった。S4 5596561a は既存のテストのファイルに触れていない。S5 36b9d3a3 は `TestRenderRolePrompt_Leader_FeatureOrgProcedure` の語句 2 つ。9fb3fafa(L11)が同じ intro の語句をもう一度書き換えたが、base から見て変わった語句は 2 つのままである。`TestSpawnPrecheckErr_MatchesSpawn` には新しいケースが足され、既存のケースは変わっていない |
| AC12 文書と同期の検査 | 一部(D1 を /sync-docs で直す) | `check-sync.sh` は rc 0(DRIFTED 0)、`check-skill-sync.sh` は rc 0。skill の 4 面は同じ sha256(34e90d19…)、agent-messaging の 2 面は `cmp` で一致する。ずれは下の「Documentation drift」の D1〜D3 |
| AC13 start が記録する方式 | 満たす | 記録は `startFeatureLeaderParams`(`feature.go:361`)、トークンは branch の後ろ(`scopeReservedDetails`、`internal/org/reserve.go:281`)。読み戻しでは、2 つあるか空のトークンを `approvalModeUnreadable` にし(`:372`)、`RequiresApproval`(`:119`)はトークンなしだけを split と読む。`validateFeatureBinding`(`:157`)は 2 つの値のほかを断り、`sameFeature`(`:144`)は方式の違う同じ機能を別の結びつきとする。テストは `TestOrgStartPlan_ApprovalModeFromMainWorktree`(`internal/cli/org_feature_test.go:436`、本物の start で main の値を記録し、サブディレクトリの値に負けず、節がなければ per-plan、`--config` ならその値)、`TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart`(`org_approve_test.go:583`、feature worktree・サブディレクトリ・`--config` の 3 か所から send と spawn を dry-run)、`TestOrgSend_ApprovalCheck_ModeFromTheRecord`(`approval_enforce_test.go:573`)、`TestFeatureBinding_ApprovalModeRead`(`reserve_test.go:453`、11 ケース)、`TestStartFeature_RecordsApprovalMode`(`feature_test.go:315`、方式を変えた打ち直しの拒否) |
| AC14 digest が読まない場所の変更 | 満たす | `PlanContentHash`(`approval.go:280`)と、`checkApprovedPlanFile` のその照合(`:574-579`)。テストは `TestPlanContentHash`(`approval_test.go:368`)、`TestCheckPlanApproved` の AC14 の 3 ケース(`:596` Progress checklist、`:600` Branch、`:604` 見出しの後ろの Status)、send(`approval_enforce_test.go:196`、`:199`)、spawn(`:686`、checkbox の印だけなら spawned)。実機で、Progress checklist に足した 1 行は digest を変えず、approve が記録した `content=` は、その行を含む全文から先頭の 2 行を除いた内容の SHA-256 と一致した |

#### 仕様 FR-7 との対応(cycle 1)

`docs/specs/2026-10-07-org-multi-org-director.md` の FR-7(`:65-71`)、注「7 段目の前半で決めたこと」(`:72`)、受け入れ条件、Security considerations と照らした。

- 1 項目(`[org.approval].mode` で選び、既定は per-plan): AC1 と AC13 で満たす。
- 2 項目(split では分割計画だけを人が承認する): split の org では spawn も TASK も承認を読まない(AC6、AC7)。分割計画の承認は 4 段目のままである。満たす。
- 3 項目(per-plan の機能の計画の承認、まとめての承認、implementer の spawn と実装の TASK の拒否、`PLAN:` の照合、再投入と改訂): approve(AC2、AC3)、受信箱の件の ID でのまとめての承認(A4)、拒否(AC5、AC7)、照合(AC5)は満たす。再投入は TASK ごとに照合され、改訂は digest か全文のハッシュで断られる(AC5、AC14)。要件は「TASK の本文の `PLAN: <path> <sha256>`」と書くが、実装はヘッダの塊の `PLAN: <path> sha256:<12 桁>` だけを読む(D3)。
- 4 項目(`/plan` と同じ図解ページと digest、director は代行しない): 人に見せる digest は `PlanDigest` で、`/plan` の digest と同じ値になる(`TestPlanDigest_MatchesScript`、`internal/org/split_test.go:777`)。図解ページは leader が雛形で書き、ralph は検査しない(計画の Non-goals、tech-debt 216 行)。director の扱いは 8a 段目に送ったと注に書いてある。7a の範囲では満たす。
- 5・6 項目と FR-8 の「merge してよい順」: 7b(次の PR)に送った。注の末尾にそう書いてあり、計画の Non-goals(`:50`)と合う。FR-8 の `auto_merge` は 8a 段目(仕様の段の表 `:215`)である。
- 注(`:72`): 書かれた決定をコードと照らし、すべて合った。方式の読み方と記録、トークンのない記録は split、読めない記録は承認を求める、検査は台帳とファイルの両方を見る、approve が書くもの、全文のハッシュ、検査が効く範囲、`PLAN:` が 2 行なら断る、今の命、TASK の拒否は記録しない、spawn の拒否は `rejected` と receipt、leader の検査、Security の読み、の順に確かめた。注が挙げるテスト 3 本も実在する。
- 受け入れ条件: `:109`(per-plan で未承認の TASK は拒否)は `TestOrgSend_ApprovalCheck_Refusals` と AC13 の CLI のテストで、`:110`(split は承認なしで送れる)は `TestOrgSend_ApprovalCheck_NotApplied` と同じ CLI のテストの split の場合で満たす。計画はこの 2 行を「107・108 行」と書くが、base でも 108・109 行で、注が 1 行増えた今は 109・110 行である(D4)。
- Security considerations(`:222`「director も leader もこの検証を迂回できない」): 承認については満たさない。計画の A2 と注がそう読み、tech-debt 213 行が記録している。`approveTripwire` は `HERDR_PANE_ID` の値と active な座席の pane を比べるだけなので、変数を外せば通る。leader は manifest に `plan_approved` を直接書け、`currentPlanApprovals` は Details だけを読んで書き手を区別しない。記録の `by=`・`herdr_pane=`・`cwd=` は `ralph org report` の event timeline に出る(`BuildOrgReport` はすべての event を表にする、`internal/org/report.go:164-180`)。「境界ではなく tripwire」という書き方は、skill、README、quality-gates、approve の help、コードのコメントで揃っている。Security の節そのものには注への参照がない(任意)。

#### 計画からのずれ(Progress checklist の記録)の判断(cycle 1)

- 逸脱 1(`approve.go` と `approve_test.go` に分けた): AGENTS.md の repo map が両方のファイルを挙げ、`seat.go` のコメントも `approve.go` を指す(8ad54167)。妥当。
- 逸脱 2(拒否の次の手を理由ごとに 1 つにした): `approvalNextStep`(`approval.go:663`)の 5 つの kind(TASK のヘッダは、ないときと digest が古いときで 2 通り)は、Progress、雛形の手順 7、skill、quality-gates の書き方と一致する。AC10・Scope 45 行・Risks 145 行の「BLOCKED」は、記録がない場合の後半と「それ以外」の場合に残る。妥当。
- 逸脱 3(`- Approved:` の行の形を厳しくした): `approvedLineDigest`(`:249`)。Scope 38 行の「`/implement` の 4 手目と同じ読み方」より厳しいが、コードのコメント(`:191-200`)と skill の照合の項に書いてある。Progress の「docs/plans の 22 本のうち 21 本がこの形」は数え直して合った(`- Approved:` の行を見出しの前に持つ計画が 22 本で、残る 1 本はテンプレートの `TBD`)。
- 逸脱 4(拒否の文から手順の番号を外した): `approvalNextStep` の文に手順の番号はない。妥当。
- S1 の `gofmt -w`: `internal/cli/org_feature_test.go` の base からの差分に消えた行はなく、中身は変わっていない。

### Static analysis(cycle 1)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | `scripts/ralph-config.sh` の変更で full fallback になり、golang の pack を選んだ。`gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、staticcheck は出力なし。shellcheck、hook の `sh -n`、guard の awk の構文、settings の `jq -e`、Codex の guard 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照の検査、branch の secret scan(225dff27..78b3e873、clean)も通った |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `shasum -a 256` で org の skill 4 面 | 4 面とも 34e90d19d554… | |
| `cmp` で agent-messaging と `ralph-config.sh` の 2 面 | どちらも一致 | quality-gates の 2 面は base と同じ KNOWN_DIFF だけが違い、新しい行(`:81`)は同じ |
| `./scripts/plan-visual.sh digest <plan>` | 951cf09bd7bc | `- Approved:` 行と一致 |
| `./scripts/secret-scan.sh --file` で evidence の 3 ファイル | rc 0 | コミットの前に確かめた |

### Documentation drift(cycle 1)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1 `/org` skill の `approve` の行(`.claude/skills/org/SKILL.md:180`、4 面) | いいえ | 括弧の「approve が受信箱を読んだあとにほかの誰かが件を resolve していたときは、打ち方は示さず、承認は記録してあると言う」は 9fb3fafa(L3)の時点の文である。4b6d0d55(N3)のあと、エラーは「その人が直しを求めたかもしれないので、まだ leader に承認の DECISION を送らず、先に `ralph org inbox show <ID>` で note を読む」とも言い(`internal/org/approve.go:405-408`)、stdout には DECISION のコマンドがそのまま出る。括弧に「leader への承認の DECISION はまだ送らず、先に `ralph org inbox show <ID>` で resolve の note を読むよう言う」を足す。4 面とも同じに直す |
| D2 `ralph org spawn --help` と `ralph org send --help`(`internal/cli/org.go`)(任意) | 一部 | どちらも見張りの拒否は書くが、per-plan の implementer の spawn と TASK の拒否、leader のいない org の spawn の拒否は書かない。計画の文書の一覧(Scope 46 行、AC12)に help は入っていないので、AC の不足ではない。help は `internal/` の文字列なので、/sync-docs で直さないなら tech-debt の行にするか 7b に回す |
| D3 仕様 FR-7 の注(`docs/specs/2026-10-07-org-multi-org-director.md:72`)(任意) | ほぼ合う | 要件(`:68`)は「TASK の本文の `PLAN: <path> <sha256>`」と書く。実装はヘッダの塊の `PLAN: <path> sha256:<12 桁>` だけを読み、空行のあとの本文の行は数えない(`protocol.HeaderValues`、雛形の手順 5)。注は「TASK の `PLAN:` のヘッダ」と書くが、要件の「本文」をヘッダと読んだことは書いていない。たとえば「上の要件の「TASK の本文の `PLAN:`」は、ヘッダの塊の 1 行 `PLAN: <path> sha256:<digest>` と読んだ」を足す |
| D4 計画が挙げる仕様の受け入れ条件の行番号(情報) | 直さない | 計画の A5・AC6・Verify plan は「107・108 行」「108 行」と書くが、base(225dff27)でも 108・109 行で、今は 109・110 行。計画の本文は承認の digest の対象なので直さない。この報告は今の行番号を使った |
| D5 計画の Progress checklist と AC のチェックボックス(情報) | 遅れている | `- [ ] Review artifact created` は self-review の報告があるのに付いていない。AC1〜AC14 のチェックボックスも付いていない。`## Progress checklist` は digest の外なので、/sync-docs か /pr で付けてよい。AC は検証のあとに付ける段取りと読んだ(これまでの計画と同じ) |
| `/org` skill のほかの箇所(4 面) | 合う | verb 表の `spawn`・`send`・`start`(`:168`、`:169`、`:177`)、「機能の計画の承認」節(`:611`〜)、「start が拒否するもの」の方式の違う同じ機能(`:526`)、上限の節の方式の読み方(`:231`)、受信箱の節(`:307`)、leader のサイクル(`:852`〜)をコードと照らした。補償で予約を戻したときに前の承認が数えられないこと(tech-debt 215 行)も書いてある |
| `.claude/rules/ralph/agent-messaging.md:102-108` と template の写し | 合う | `cmp` で一致。`--raw` の TASK も同じヘッダの規則で読むこと、ほかの座席への TASK とほかの TYPE は承認が要らないことがコードと合う |
| README の org runtime(`:124` の verb 表、`:258` の段落) | 合う | `approve` が verb 表にあり、段落はまとめての all-or-nothing、herdr の pane の外で打つこと、境界ではないこと、start が記録する方式、古い版の org、leader のいない org の拒否を書く |
| AGENTS.md の repo map(`:90` の `internal/org/`) | 合う | `approve.go` と `approval.go`、`start --plan` が記録する方式を挙げる |
| `docs/quality/quality-gates.md`(2 面) | 合う | `:81` の行は 2 面で同じ。理由ごとの次の手の列挙はコードと合う |
| `templates/base/ralph.toml:120-131` | 合う | 既定 per-plan、ほかの値と `""` はエラー、start が org とともに記録すること、節がなければ per-plan であることが `Load()` と合う |
| `internal/org/prompts/leader.md` | 合う | `:76-82` と手順 1〜7。目印 `refused by the approval check`(`approval.go:645`)と、spawn の拒否が stdout の `rejected:` の行に出ること(CLI のテストが stdout の先頭を見る)が合う |
| `ralph org approve --help`、`ralph org --help`、`ralph org start --help` | 合う | ビルドで読んだ。approve の help は確かめることと all-or-nothing、記録の中身、tripwire、`--org-id` の拒否、split の org の記録を書く。`--org-id` の説明に approve が入った |
| tech-debt 213〜221 行 | 合う | 9 行とも 78b3e873 までに入った。挙げた関数とテスト(`approveTripwire`、`checkPlanApproved`、`sendApprovalErr`、`seatRoleFromEvents`、`spawnFeatureSeatErr`、`orgFeatureJSON`、`printStatusTable`、`reserveAgain`、`CloseDeferredSelfPane`、`CloseDeferredSelfWorkspace`、`currentPlanApprovals`、`approvalModeUnreadable`、`checkTaskPlanApproved`、`currentPlanRef`、`resolveApproved`、`approvalNextStep`、`TestCheckTaskPlanApproved_StaleOrMistypedDigest`、`checkFixture`、`enforceFixture`)は実在し、書かれた挙動がコードと合う。213 行の tripwire の記述は `approveTripwire`(`approve.go:203`)の比較と合い、214 行のとおり `orgFeatureJSON`(`internal/cli/org.go:1400`)に方式の欄はない。217 行の (d) は 9fb3fafa で閉じて行から外した。同じブランチで足して外したので、取り消し線は要らない |

D1 は /sync-docs で直す。D2 と D3 は任意で、D4 と D5 は記録だけ。

### Observational checks(cycle 1)

HEAD 78b3e873 のビルドで、タスクの worktree の gitignore された `.harness/state/verify-probe/` に台帳と機能の worktree を作り、`ralph org approve` を打った。approve は herdr も agmsg も呼ばない。受信箱の件は手で書いたので、escalate と通知は走っていない。スクリプトと出力は evidence の probe のファイルにある。

| 場合 | 結果 | 台帳 |
| --- | --- | --- |
| `--org-id a` を付ける | rc 1、`--org-id does not apply` | 変わらない |
| `HERDR_PANE_ID` が leader の pane | rc 1、tripwire の文(座席と org を名指し、pane の外の端末で打つよう言う) | 変わらない |
| `e1 e2 e9`(e2 は BLOCKED、e9 はない) | rc 1、「2 of the 3」と 2 件の理由 | 変わらない |
| Objective を 1 行変えたあとに `e1` | rc 1、digest が今は 7b9b4ffdf28d で QUESTION の 03e15b9b55e1 と違う | 変わらない |
| 計画を戻して、別の pane の値で `e1` | rc 0、approved の行、DECISION のコマンド、記録の注 | `plan_approved` 1 行(`by=verifier herdr_pane=set cwd=…`)、e1 は resolved |
| もう一度 `e1` | rc 1、already resolved | 変わらない |
| Progress checklist に 1 行足して、同じ QUESTION の `e3` | rc 0。digest は 03e15b9b55e1 のまま | 記録の `content=` は、足した行を含む全文から先頭の 2 行を除いた内容の SHA-256 と一致 |

- 計画のファイルの sha256 は approve の前後で同じだった(AC2)
- 最初の試行では、計画を変える場所を Progress checklist にしてしまい、approve は断らずに記録した。digest がその節を読まないためで、skill の「QUESTION が上がってから approve を打つまでの間に、digest が読まない場所が書き換えられていれば、approve はそれも含めて記録する」のとおりの動きである。変える場所を Objective にして取り直したのが上の表で、Progress checklist の場合は最後の行に分けて残した

### Coverage gaps(cycle 1)

- テストは走らせていない(/test の担当)。表の関数が通ること、検査の条件を外すと落ちることは読んだだけである
- spawn と TASK の拒否は、ビルドしたバイナリでは確かめていない。見張りの検査(長く動く `ralph org watch --all`)が先に断るので、使い捨ての台帳で見張りを動かす必要があり、herdr と agmsg に触れずに済む形が組めなかった。CLI の層では `TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart` が stub の PATH で本物の start から通す。計画の Verify plan が求める「一時の台帳での実機確認」のうち、approve の側(成功、計画の変更の拒否、座席の pane からの拒否)はこの報告の probe で取った
- leader が図解ページを作って撮れるか、headless の leader が承認の手順を最後まで進めるかは、実機で走らせていない(tech-debt 216 行、9 段目の試走)
- `ralph org approve` の、追記が途中で失敗した場合と resolve が失敗した場合は、テストを読んだだけである(`TestOrgApprove_AppendFailureRecordsNothing`、`TestOrgApprove_ResolveFailureKeepsTheRecord`)
- Linux と、別の herdr サーバーの pane の id が重なる場合は試していない
- release のノート(既定の per-plan で `start --plan` の org の implementer が承認を待つこと、`ralph.toml` は seed のファイル(`ownerForScaffoldPath`、`internal/cli/init.go:294`)で、おそらく `ralph upgrade` は既存のものに `[org.approval]` を足さないので、節のない利用者も既定の per-plan になること。upgrade の動きは確かめていない)と PR の本文の A2 の読みは、/pr と release の段で確かめる

### cycle 1 の判定

pass

- Verified: AC1〜AC10、AC13、AC14 はコードとテストの対応を読んで満たすと判断した。AC2〜AC4 と AC14 の一部はビルドした approve でも確かめた。FR-7 の 1〜4 項目は 7a の範囲で満たし、5・6 項目と FR-8 の「merge してよい順」が 7b に送られたことは仕様の注に書いてある。受け入れ条件の 109・110 行はテストに対応がある。Security considerations の「迂回できない」は承認については満たさないと読み、仕様の注、計画の A2、tech-debt 213 行がそう書いていることを確かめた。逸脱 1〜4 は妥当と判断した。静的解析(changed scope、full fallback)、`check-sync.sh`、`check-skill-sync.sh` は rc 0
- Partially verified: AC11(書き換えた本数と理由は記録と合う。通るかは /test)、AC12(D1 を /sync-docs で直す。D2 と D3 は任意)
- Not verified: テストの実行、spawn と TASK の拒否のバイナリでの確認、leader の図解ページと承認の手順の実機、Linux、release のノートと PR の本文
