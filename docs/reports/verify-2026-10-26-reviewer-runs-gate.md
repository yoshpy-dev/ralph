# Verify report: reviewer-runs-gate

- Date: 2026-10-26
- Plan: `docs/plans/active/2026-10-26-reviewer-runs-gate.md`(AC1〜AC8、D1〜D8、Progress checklist の逸れ 2 件。パイプライン cycle 1)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff origin/main...HEAD`(HEAD c9caf03a、base 633cd7c8、37 ファイル、+2103/-582、16 コミット)。self-review の 3 回のあとの直し 286f3d28、64f36f3f、67f3609d、d3c8e337 を含む。Go のテストと `run-test.sh` は打っていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-26-reviewer-runs-gate.log`(git が無視する場所)。`run-static-verify.sh` の出力、同期の検査 3 本、`gofmt`、`go vet` の 2 版、下の probe の出力を入れた。`run-static-verify.sh` 自身も `docs/evidence/verify-2026-10-26-143306.log` を書いた

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(S1: codex の機能の org の reviewer に `--add-dir`) | 満たす | `internal/org/permissions.go:199` の `writesLedger := role == LeaderIdentity \|\| (featureOrg && role == reviewerRole)`。ほかの条件(codex、autonomous か edits、台帳が cwd の外)は変わっていない。`internal/org/spawn.go:1050` は `events` から `ActiveFeature` を読む。この `events` は manifest のロックの下の読み直し(`:777`、stale のときは `:958`)で、`spawnFeatureSeatErr` と同じもの。新しいロックはない。`reviewerRole` は `internal/org/approval.go:70`。表のテスト `TestCodexWritableRootArgs_FeatureOrgReviewerGetsAddDir` は、機能の org の reviewer(autonomous と edits)と leader に `--add-dir` が付き、台帳が cwd の下の場合、guarded、claude、大文字の `Reviewer`、implementer、未知と空の役割、director に付かない行を持つ。結びつきのない org の reviewer の行は既存の表に `featureOrg=false` で残る。argv のテスト `TestOrgSpawn_Codex_WorkspaceWrite_FeatureOrgReviewerGetsAddDir` は、`--add-dir <台帳>` が sandbox の flag のすぐあと、`--model` の前にあることと、機能の org の implementer と結びつきのない org の reviewer の argv に入らないことを、autonomous と edits の両方で見る(読んだだけで、走らせていない) |
| AC2(S2: 見張りの数え方とコメント、断りの文、help) | 満たす | `internal/org/watch.go` の差分は 1 つの塊で、コメント (c)(`:1345` から)だけ。`leaderActivityEventCount`(`:1386`)の `case EventGatePassed, EventGateFailed`(`:1401`)は変わっていない。コメントは、gate が leader の `sent` の TASK を起点にすること、基準値 `watchPendingAlert.ManifestLen`(`:239`、`:1263`)が更新と戻しをまたいでずれること、`checkDeadman` が `activity && !subjectIsLeader`(`:1587`)のときだけ消すことを書く。3 つともソースと合う。regate の断りの文(`internal/org/regate.go:481-483`)と、そのテスト(`regate_test.go:594-596`)は reviewer が leader の TASK で打つと言う。help(`internal/cli/org_gate.go:30-35`、`:70-76`)とテスト `TestOrgGateHelp` のピン、`internal/org/gate.go:16-27` のコメント(「Gate does not check who runs it」)も同じ向きで、計画にない `internal/cli/org_regate.go:37-39` の help と `internal/org/regate_merge.go` の 2 つのコメントもそろえてある |
| AC3(S3: leader の雛形) | 満たす | `internal/org/prompts/leader.md:249` の手順 12 が、TASK_ID `gate-<n>`、本文の 2 つのコマンド(`:261` の gate のコマンドは `reviewer.md:51` と同じ文字列)、結果が来るまで worktree を変えず implementer に TASK を送らないこと、`SEAT: reviewer`・`CONDITION: stall` の ALERT で stop も spawn もしないこと、3 つの結果と QUESTION の扱いを書く。`GATE: pass` の項(`:278`)は `.out` の `GATE: pass` と `HEAD:` を RESULT と `git rev-parse HEAD` に照らし、座席を止める前に `wip:` の `git log` を打つ。手順 9(`:203`)は「TASK_ID が `gate-` で始まらない」レビューの TASK で始まり、gate の TASK の RESULT は手順 12 で扱うと書く。手順 14(`:324`)は控えた `HEAD:` と照らし、違えば手順 5 と同じ flag(`--role reviewer` を含む)で reviewer を spawn し直す。ミッションの 5(`:32`)・6(`:42`)・7(`:46`)、運用規律(`:528`)、受信箱の節(`:512`、`:517`)も直っている。`ralph org gate` が出る 4 文はすべて許された形(下の probe)で、leader 自身が打つ文はない |
| AC4(S3: reviewer の雛形) | 満たす | `internal/org/prompts/reviewer.md:44` の「gate の TASK」の節が、TASK_ID の見分け、差分レビューと素の実行をしないこと、2 つのコマンドと同じときだけそのまま打ち、違えば QUESTION で聞くこと(逸れ 1)、Claude Code は `run_in_background`、Codex は前面と timeout、worktree を書き換えずコミットしないこと、3 つの返し方、打ち直しは 1 回を書く。前文(`:12-15`)に台帳の gate の記録とログと `.out` が入り、`ralph org gate` のほかの動詞を打たず台帳を直接書かないと書く。ミッションの 1(`:17`)とレポート契約(`:198`)も gate の TASK を分ける。例の RESULT と BLOCKED は `TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand` が `protocol.Parse`・`Validate`・`HeaderValues` で検査する(読んだだけ) |
| AC5(S4: テストで固定し、文を戻すと落ちる) | 満たす(静的に一部を確かめた) | 新しいテスト 5 本と、既存のピンの書き換えを読んだ。消えたピン(leader の手順 12 の `run_in_background` の順)は reviewer のテストの同じ順の検査に移り、意図は残る。`go test` は打たず、純粋なテキストの検査 2 つ(`leaderRunsGateItself` と、`TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate` の順の検査)を `python3 -I` に写して走らせた。HEAD は 0 件と 0 件、origin/main の `leader.md` は 8 件と 3 件で落ちる。HEAD から 1 文ずつ消すか戻した 4 つの mutant(`gate-<n>` の文、leader が gate を打たない文、停滞の ALERT の文、手順 14 の spawn し直しの文)は、どれも 1 件以上で落ちる。写しであってテストそのものではない。残りの文の mutant と本物の実行は `/test` に残す |
| AC6(S5: skill と文書) | 満たす | `check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)が通る。`/org` と `/director` の 4 写しは `cmp` で同一で、`.claude/` と `.agents/` は先頭 5 行を除いて差がない。recipe と `git-commit-strategy.md` の root と template は `cmp` で同一。quality-gates(KNOWN_DIFF)の Quality pipeline gate の行は、root にだけある「(see docs/tech-debt)」を除いて同じ文。`/org` skill の「org の終わり方」1〜7(`.claude/skills/org/SKILL.md:943` から)は leader の手順 9〜14 と 16 と同じ順で、項 4 の止めない座席、`gate-<n>`、待つあいだ worktree を変えないこと、停滞の ALERT、頼み直しの上限、結果の 4 つの枝、項 6 の spawn し直しの flag が雛形と同じことを言う。動詞の表の `gate` の行(`:221`)、gate の記録の数え方(`:1097` からの節)、codex の `--add-dir` の段落(`:1491-1496`)、README(`:252`、`:264`)、recipe(`docs/recipes/codex-seat-permissions.md:89-99`)、`/director` skill(`:666-670`、`:713`、`:936-938`)も合う |
| AC7(S6: 仕様、根拠の表、tech-debt) | 満たす | 仕様の FR-7(`docs/specs/2026-10-07-org-multi-org-director.md:113`)が `[ ]` で、「2026-10-26 に決めたこと」(`:124`)と「印(2026-10-26)」(`:126`)がある。Status(`:39`)は `[ ]` を 6 項目と書き、仕様の `[ ]` は FR-3、FR-4、FR-7、FR-10、ツール非依存、受け入れ条件 5 の 6 行で一致する。根拠の表は base の `[x]` 26・`[ ]` 5 から HEAD の 25・6 になり、`:105` の文も 25 と 6。仕様の `- [x]` は 26 で、表に入らない FR-12 の分だけ多い(仕様 32 項目、表 31 行)。研究の所見の 1 段目の行(`:258`)に今回の変更が足されている。印と注が挙げるテスト名 5 つは実在する。tech-debt の `:225` は codex の reviewer が gate のスクリプトを走らせると書く |
| AC8(`run-verify.sh` と Go のテストの 2 版) | この報告では確かめていない(`/test`) | 静的な半分は通った(`run-static-verify.sh` rc 0、`go vet` は go1.27.3 と go1.25.8 の両方で rc 0)。オーケストレーターの `GOTOOLCHAIN=go1.25.8 ./scripts/run-verify.sh`(rc 0)と `GOTOOLCHAIN=local go test -count=1 -timeout 30m ./...`(rc 0)は HEAD 6182cb2e のもので、そのあとの直し 4 コミットは `leader.md`、`reviewer.md`、`prompts_test.go`、`permissions.go`、`approval.go` を変えた。HEAD c9caf03a での実行は `/test` が打つ |

leader と reviewer の雛形の食い違い(計画の Verify plan): leader の手順 12 が本文に書く 2 つのコマンドと、reviewer の「gate の TASK」の節が照らす 2 つは、展開の前の文字列が同じ(`leader.md:261`、`reviewer.md:51`)。TASK_ID の形 `gate-<n>` も両方にある(`leader.md:258`、`reviewer.md:46`)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOTOOLCHAIN=go1.25.8 ./scripts/run-static-verify.sh` | rc 0 | changed の範囲で golang の pack を選んだ。`gofmt: ok`、`go vet` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし。shellcheck、hook の `sh -n`、guard の awk、settings.json の `jq`、Codex の hook の検査、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照も通った。branch の secret scan は `633cd7c8..c9caf03a` で clean |
| `./scripts/check-skill-sync.sh` | rc 0 | `14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-pipeline-sync.sh` | rc 0 | 6 ファイルすべてが pipeline の段を参照 |
| `gofmt -l internal` | rc 0、出力なし | |
| `GOTOOLCHAIN=local go vet ./...`(go1.27.3) | rc 0、出力なし | |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | rc 0、出力なし | |

`GOTOOLCHAIN=go1.25.8` を付けたのは環境の都合で、変更の問題ではない。go.mod は `go 1.25.8` を指し、CI もその版で動く。手元の go は 1.27.3 で、手元の golangci-lint 2.9.0 は go1.26 で build されているので、1.27.3 の下では typecheck で落ちる(オーケストレーターの報告による。私は 1.27.3 の下で golangci-lint を打っていない)。

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| leader が gate を打つと言う文(雛形、skill、help、README、quality-gates、仕様、recipe、コメント) | 合っている | 2 つの掃引をした(evidence の 8・9)。1 つ目は `git grep -i -E 'leader.*org gate\|org gate.*leader'` を、archive の計画、`docs/reports/`、`docs/evidence/`、この計画の外に当てた。2 つ目は段落をつないで文に切り、leader と gate を打つ動きが同じ文にあるものを出す `python3 -I` の掃引。残りは、reviewer が leader の TASK で打つという文、leader が頼むという文、leader が打たないという文と、仕様の 7b と 8a-3b の注(`:121`、`:123`。`:124` が当時の決定として残すと書く)だけ。`internal/cli/org.go:41-42` の help の「the head a feature org's leader pushes」は push する人の話で、食い違わない |
| codex の `--add-dir` を leader だけと言う文 | 合っている | `add-dir` を含む行のうち「leader だけ」「only the leader」などを含むのは、仕様の `:124`(変えたことの説明)と `:258`(1 段目の決定に今回の変更を足した行)だけ。recipe、`/org` skill、`permissions.go` と `spawn.go` のコメントは、機能の org の reviewer にも付くと書く |
| 仕様の日付つきの注と印 | 合っている(履歴) | `:162` の「印(2026-10-23)」は、Codex の leader が gate で待つあいだを書いた当時の記録。印の行は履歴として残すという計画とリードの指示のとおり |
| `.claude/rules/ralph/git-commit-strategy.md` と template の写し(逸れ 2) | 合っている | 「after its reviewer's gate passes and again before it pushes ... (steps 12 and 14)」は `leader.md` の手順 12 の pass の項と手順 14 に合う。2 つの写しは `cmp` で同一 |
| `docs/tech-debt/README.md` の Trigger の列(V1) | 一部が遅れている(LOW) | この diff が当たる Trigger を持ち、このブランチの持ち越しの注がない行が 10 ある(evidence の 10)。`:157`「The next change to the org quality pipeline」。`:171`「The next change to ... `codexWritableRootArgs`」(引数と述語が変わった)。`:177` の (a)「The next edit to the `/org` skill's command table」(`gate` の行を変えた)。`:197` の (g)「the next time the feature-org section of `leader.md` is edited or re-wrapped」。`:234`「The next change to `internal/org/gate.go`」(冒頭のコメントを変えた)。`:257`「The next change to the inbox section (`受信箱の処理`) of the `/director` skill」(`:666-670` と `:713` はその節の中)。`:357`「the leader template tests, or any of these files growing past its count here plus 300」(`prompts_test.go` は行の数の 2165 から 2670 になり、この PR だけで 2189 から 2670、+481)。`:359`「The next change to `internal/org/gate.go`」(この行の Debt の列は直したが、Trigger の列の注はない)。`:363`「The next change to the implementer or reviewer role prompt」。`:367`「the feature-org steps of `internal/org/prompts/leader.md`」。どれも振る舞いの食い違いではなく、`:357` と `:367` は前の PR(fix/leader-spawn-pr-rules)が「Carried over to the next change that meets this trigger」と書いた形がある。`/sync-docs` で、払うか、このブランチの名前と新しい数で持ち越しの注を足す。計画が `/sync-docs` に回した Codex の reviewer の send の行(Progress checklist)とは別 |
| 計画の Progress checklist と AC の印(I1) | 遅れている(情報) | AC1〜AC8 は `[ ]` のまま、「Review artifact created」も `[ ]` だが、self-review の報告は a183673e から入っている。影響範囲(Affected areas)に、直しで触れた `internal/org/approval.go`(F6 の `reviewerRole`)、`internal/org/regate_merge.go`、`internal/cli/org_regate.go`、`internal/cli/org_gate_test.go` がない。どれも S2 と F6 の続きで、計画の意図から外れていない。`/pr` の前に印を付けるなら同じ時に書く |
| AGENTS.md の repo map | 合っている | `internal/org/` の gate の説明は打つ人を書かないので、直す所はない |

## Observational checks

- この変更は雛形と spawn の引数の変更で、画面や外の道具の観察は計画の範囲外(Non-goals の「実機での確かめ」)。herdr、agmsg、codex の sandbox は打っていない。
- 写しの probe(evidence の 8): `render` は `{{ORG_ID}}` を `org-a` に、`{{TASK}}` を空にするだけで、`markdownSection`・`markdownItem`・`squashSpace`・`templateSentences`・`leaderRunsGateItself` を行ごとに写した。結果は次のとおり。

  | 入力 | `leaderRunsGateItself` の件数 | 順の検査の件数 |
  | --- | --- | --- |
  | HEAD の `leader.md` | 0 | 0 |
  | origin/main の `leader.md` | 8 | 3 |
  | m1(`gate-<n>` の文を消す) | 0 | 1 |
  | m2(leader が打たない文を、打ってよい文に替える) | 1 | 1 |
  | m3(停滞の ALERT の文を消す) | 0 | 1 |
  | m4(手順 14 の spawn し直しの文を消す) | 0 | 1 |

## Coverage gaps

- Go のテストと `run-test.sh` は打っていない(AC8 は `/test`)。オーケストレーターの 2 回の実行は直し 4 コミットの前の HEAD 6182cb2e のもの。
- AC5 の「1 文ずつ戻すと落ちる」は、テキストの検査 2 つを写した probe と 4 つの mutant で確かめただけ。ほかのピン(reviewer の節、手順 9、ミッションの 5 と 7、受信箱の節)の mutant は打っていない。
- 実機(herdr の上で reviewer が gate を打つこと、codex の reviewer の `--add-dir` で台帳に書けること)は確かめていない。計画の R6 と D8 のとおり FR-7 を `[ ]` にしてある。
- self-review の開いた指摘 LOW F11(手順 14 から手順 12 に戻る道で、止めた implementer を立て直す手がない)と INFO N2(`watch.go` のコメント (c) の長さ)は、この報告では評価を変えない。F11 を tech-debt に足すかは `/sync-docs` の判断に残る。
- insight event は `docs/insights/events/` に足していない。リードの指示が報告だけを書いてコミットしないことで、self-review も足していないため、`--events-dir <scratchpad>` で作った行を evidence の 13 に残した。`{"schema":1,"ts":"2026-10-26T14:44:50Z","slug":"reviewer-runs-gate","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`。足すときは `./scripts/insights-append.sh --slug reviewer-runs-gate --flow standard --phase verify --verdict pass --cycle auto --source skill`
- 足すとよい決定論の検査: tech-debt の Trigger の列のうち、ファイル名か関数名を名指すものを diff のファイル一覧と照らし、持ち越しの注がない行を出す小さなスクリプト(`scripts/` に置く)。今回の V1 のような遅れを `/sync-docs` の前に機械で出せる。

## Verdict

- Verified: AC1、AC2、AC3、AC4、AC6、AC7。AC5 はテストの中身を読み、テキストの検査 2 つを写した probe で確かめた。静的解析はすべて通った(`run-static-verify.sh` rc 0、`check-skill-sync.sh`・`check-sync.sh`・`check-pipeline-sync.sh` rc 0、`gofmt -l` は空、`go vet` は 2 版で rc 0)
- Partially verified: AC5(写しの probe と mutant 4 つ。本物の実行は `/test`)
- Not verified: AC8(`/test`)、実機での leader と reviewer の動き
- Findings: LOW 1 件(V1、tech-debt の 10 行の持ち越しの注。`/sync-docs` で直せる)と情報 1 件(I1、計画の印と影響範囲)。CRITICAL・HIGH・MEDIUM はない
- 判定: pass
