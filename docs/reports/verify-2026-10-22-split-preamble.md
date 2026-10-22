# Verify report: split-preamble

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-split-preamble.md`(パイプライン cycle 1)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff a7bdc884...HEAD`(HEAD c0ff8398、22 ファイル、+844/-86)。self-review の LOW を直した b09b7c12 と c0ff8398 を含む。テストは実行していない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-22-split-preamble.log`(git が無視する場所。`run-static-verify.sh` 自身のログは `docs/evidence/verify-2026-10-22-140439.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 試行の分割計画の前文 | 満たす(コードとファイルで確認。テストの実行は `/test`) | `docs/evidence/org-trial-2026-10-20/splits/trial.md` は 1〜4 行がヘッダ(題、`- Status:`、`- Approved:`)、6 行が `## 目的`、11 行が「座席の決まり(両方の機能に共通):」、15 行が「- 予約の外のファイルは編集しない」、17 行が `## Features`。`parseSplitPlan` は `featuresLine == 0` の間だけ見出しの行と本文の行を前文に足し(`internal/org/split.go:339`、`:355`)、`trimBlankLines` で前後の空行を詰める(`:610`、`:634`)ので、前文は 6〜15 行になる。機能の読み込みの行は diff で変わっていない。`TestLoadSplitPlan_TrialPreamble`(`internal/org/split_test.go:520`)が同じ境界と 2 機能の slug・Type・Reserve・本文の先頭と末尾を固める |
| AC2 境界 | 満たす(コードで確認) | フェンスの判定(`split.go:329`)が見出しの判定より前にあるので、フェンスの中の `## Features` は見出しにならず前文に残る。`## Features` より後の `## ` の見出しは `cur` を nil にし(`:331`)、`featuresLine != 0` なので前文にも本文にも入らない。`\r` は前文に足す前に落ちる(`:325`)。ヘッダのあとすぐ `## Features` なら前文の slice は空で、`trimBlankLines` は "" を返す。`TestLoadSplitPlan_Preamble`(`split_test.go:563`、11 件)と拒否の 3 件(`:310` 前文の `- Status:`、`:319` 前文の `- [x]`、`:334` 前文の閉じていないフェンス)が打つ |
| AC3 `featureLeaderTask` の順とフェンス | 満たす(コードで確認) | `internal/org/feature.go:409-412` が、今の行のあとに `"\n\n" + featurePreambleLead + "\n" + fence + "\n" + plan.Preamble + "\n" + fence` を足し、そのあとに `"\n\n" + f.Body` を足す。前文が空白だけなら何も足さない。`fenceFor`(`:423`)は最長のバッククォートの並び + 1、最低 3。`TestFeatureLeaderTask_Preamble`(`internal/org/feature_test.go:580`)が 4 連 → 5 連、3 連のフェンス → 4 連、チルダと inline code → 3 連、空白だけの前文を、`TestFenceFor`(`:631`)が並びの位置を見る |
| AC4 role prompt に前文が入る | 満たす(コードで確認。テストの実行は `/test`) | テストは `testOrg` の `fakeHerdr` と `fakeAgmsg`(`internal/org/spawn_test.go:506`)で `StartFeature` を回すので、本物の herdr は使わない。leader の role prompt は改行を含むのでファイルに書かれ(`internal/org/spawn.go:1002-1030`)、`Task` に長さの上限はない(`DefaultMaxBodyChars` を使うのは send・escalate・watch だけ)。`strings.NewReplacer`(`internal/org/prompts.go:87`)は 1 回の走査なので、前文の `{{ORG_ID}}` は置き換わらない。`TestStartFeature_LeaderPromptHoldsPreamble`(`feature_test.go:657`)が、2,000 字を超える前文(`:685`)、節の見出しが 1 回ずつ見つかること、`{{ORG_ID}}` が残ることを固める |
| AC5 leader.md の手順 1・3・5 | 満たす | 手順 1 は `internal/org/prompts/leader.md:90-94`(囲みの行も含めて原文のまま `## 全機能の座席に適用する決まり` の節に写す)、手順 3 は `:133-136`(承認の前の reviewer も同じ driver と model)、手順 5 は `:151-155`(driver と model は spawn の引数、ほかの決まりは TASK の本文に写さず計画の節に従わせる)。3 文とも「全機能の座席に適用する決まり」を使い、`featurePreambleLead`(`feature.go:375`)の先頭と一致する。`TestRenderRolePrompt_Leader_FeatureOrgSeatRules`(`internal/org/prompts_test.go:1049`)が固める 7 つの句を、空白を詰めて leader.md の節で探すと、すべて見つかり、順も合う(log の手作業の確かめ)。`TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`:475`)は diff で変わっていない |
| AC6 start の help とエラー | 満たす | help は `internal/cli/org.go:920-922`、エラーは `:1044`。scratch に build した binary の `ralph org start --help` が「and its task from the feature's body and the plan's preamble (its ## sections before ## Features)」と出す(log)。テストの期待は `TestOrgStartHelp_TaskSource`(`internal/cli/org_feature_test.go:524`)と `TestOrgStartPlan_FlagCombinationsRefused` の行(`:97`)で直した |
| AC7 文書 | 満たす | `/org` skill: 例に `## 座席の決まり`(`.claude/skills/org/SKILL.md:448`)、分割計画の節に前文が渡ること、渡らない 3 つ、前文が履歴に入ること(`:487-506`)、start の手順 6 のタスクの中身(`:576-580`)、Leader のサイクル(`:1214-1225`)。`/director` skill は `.claude/skills/director/SKILL.md:465-469`。仕様 FR-4 の注(`docs/specs/2026-10-07-org-multi-org-director.md:80`)と README の段落(`README.md:252`)。org と director の 4 写しは `cmp` でバイト一致し、`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)も通る |
| AC8 記録 | 満たす | tech-debt の F5 の行は取り消し線と RESOLVED の注(`docs/tech-debt/README.md:322-323`)、新しい open の行は `:347`(Related 欄の「the F5 row above」は 323 行を指す)。変えた表の行はすべて 5 欄。RESOLVED のコメントが挙げるテスト 7 本は `func <name>(` で 1 か所ずつ見つかる。根拠の表の F5 の行(`docs/evidence/org-spec-evidence-2026-10-21.md:24`)は直す PR を `fix/split-preamble` と書き、tech-debt の行を `"~~...~~"(RESOLVED の行)` の形で引く。`fix/seat-wip-checkpoint` は、この PR の計画の 3 行のほかには(archive と過去の reports を除く)残っていない。PR の一覧は仕様の Status(`spec:5`、`:7-23`)と根拠の表(`:85-102`)のどちらも 33 本で、集合が一致する。印は仕様が `[x]` 26 / `[ ]` 6、根拠の表が 25 / 6 で、FR-12 だけが表にない。項目ごとの印の食い違いは 0 件で、base(a7bdc884)と同じ数 |
| AC9 run-verify と go1.25.8 | 一部(静的な部分だけ確認) | `./scripts/run-static-verify.sh` は rc 0(26.9 秒)。`go vet ./internal/org/ ./internal/cli/` は go1.26.0 と `GOTOOLCHAIN=go1.25.8` のどちらも rc 0、go1.25.8 の `go build ./...` も rc 0。`GOFLAGS=-timeout=30m ./scripts/run-verify.sh` はテストを含むので打っていない。テストの実行と go1.25.8 でのテストは `/test` が見る |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | changed-language scope で golang の pack を選んだ。gofmt ok、golangci-lint 0 issues、shellcheck、hook の `sh -n`、`check-sync.sh`(IDENTICAL 171、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、branch secret scan(a7bdc884..c0ff8398 clean) |
| `go vet ./internal/org/ ./internal/cli/`(go1.26.0) | rc 0 | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/org/ ./internal/cli/` | rc 0 | go.mod は `go 1.25.8`。`fenceFor` の組み込みの `max` は 1.21 から使える |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | rc 0 | - |
| `gofmt -l internal/ cmd/` | 出力なし | - |
| `git diff --check a7bdc884...HEAD` | 出力なし | diff 全体に U+FFFD は 0 件 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `featureLeaderTask`・`StartFeature`・`SplitPlan.Preamble`・`parseSplitPlan`・split.go 先頭の書式の doc | はい | 範囲(最初の `## ` から `## Features` の前まで)、見出しとフェンスを残すこと、前後の空行を詰めること、空の前文で何も足さないことが、コードと一致する |
| `featurePreambleLead` の doc(L1) | はい | 「Steps 1, 3, and 5」に直っている(`feature.go:372`) |
| leader.md の手順 1・3・5 | はい | 前置きの行と同じ語を使う。手順 1 の「囲みの行も含めて」は lead の「下の囲みに入れた」と合う |
| `/org` skill(4 写し) | はい | 渡る範囲、渡らない 3 つ、前文が履歴に入ること、手順 6 のタスクの順(`進め方` の行も今のコードどおりに足した)がコードと合う。L2 の折り返しは 43 列の行がなくなり、1217・1218 行は 60・61 列(log) |
| `/director` skill(4 写し) | はい | 全機能に共通の決まりを `## Features` より前の `## ` の節に書く文と、leader が計画に写す文がある |
| README・仕様 FR-4・start の help とエラー | はい | どれも、タスクは機能の本文と前文から作ると書く。「`## Features` の外の節は自由に書ける」の古い文は、この PR の計画の verify plan の 1 行のほかに残っていない |
| tech-debt・根拠の表・仕様の Status | はい | AC8 の行のとおり。L3 は `S3 1b65bf49 and 546ad029` で、2 つの id の区切りがわかる形に直っている。L4 の前置きの行は「原文のまま下の囲みに入れた」になり、定数とテスト 2 か所が同じ文字(`feature.go:375`、`feature_test.go:581`、`:672`)。旧文「下に囲む」は self-review の報告にだけ残る |
| `internal/org/prompts.go:31`、`:48-52`、`internal/org/spawn.go:418-423` | いいえ(この PR より前から) | I1 を参照 |
| 計画の受け入れ条件の印と Progress checklist | 遅れている | I2 を参照 |

## Findings

| Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- |
| INFO (I1) | この PR より前からの doc の古さ。`RolePromptVars` の doc は置き換えを「plain strings.ReplaceAll」と書くが、コードは `strings.NewReplacer` の 1 回の走査で、AC4 の「前文の `{{ORG_ID}}` は残る」はこの性質に頼る。同じ doc の `Task` と `SpawnParams.Task` の doc は、`ralph org start` の位置引数だけが `Task` を作ると書き、`start --plan` の `featureLeaderTask` に触れない。3 か所とも base(a7bdc884)で同じ文で、この diff は触っていない | `internal/org/prompts.go:31`、`:48-52`、`:87`、`internal/org/spawn.go:418-423` | 次にこの辺りを触る PR で直す。順に置き換える形へ書き換えられても、`TestStartFeature_LeaderPromptHoldsPreamble` が `{{ORG_ID}}` で落とすので、挙動は守られている |
| INFO (I2) | 計画の AC1〜AC9 の印と Progress checklist の「Review artifact created」が `[ ]` のまま。Progress の Implementation の行は LOW の直し(b09b7c12、c0ff8398)を挙げない。前の計画(`docs/plans/archive/2026-10-22-seat-wip-checkpoint.md`)は archive の前に印を付けている | `docs/plans/active/2026-10-22-split-preamble.md:71-79`、`:130-132` | `/pr` の前に印と Progress を更新する。判定には影響しない |
| INFO (I3) | 根拠の表の「確かめていないこと」に F5 の直しの項目がない。F4(#249)と F11(#250)は実機の範囲を書いている。F5 の直しは実機を回していない(計画の Non-goals)ことと、leader が前文に従うかを ralph が確かめないことは、tech-debt の RESOLVED のコメントと open の行(`:347`)に書いてある。AC8 はこの項目を求めていない | `docs/evidence/org-spec-evidence-2026-10-21.md:107-117`、`docs/tech-debt/README.md:322`、`:347` | 足すなら「F5 の直しは Go のテストで role prompt までを見た。実機の leader が前文を写して座席を立てるかは見ていない」の 1 項目 |

CRITICAL・HIGH・MEDIUM・LOW はない。self-review の L1〜L4 は 4 件とも直っている(Documentation drift の表)。

## Observational checks

- `go build -o <scratch>/ralph ./cmd/ralph` で作った binary の `ralph org start --help` の文を読んだ(herdr・agmsg・gh・ネットワークは使っていない)
- 試行の分割計画の行番号と、前文になる範囲を目で突き合わせた
- leader.md の機能ごとの org の節で、AC5 のテストが固める 7 つの句を空白を詰めて探した(順も合う)
- 仕様と根拠の表の印と PR の一覧を、python の小さなスクリプトで HEAD と base の両方について数えた

## Coverage gaps

- テストは実行していない。AC1〜AC6 のテストの合否と、`GOFLAGS=-timeout=30m ./scripts/run-verify.sh` の rc、go1.25.8 でのテストは `/test` が確かめる。self-review の時点の run-verify の rc 0 は 7cc48f33 のもので、そのあと b09b7c12 が `feature.go` と `feature_test.go` を変えている
- PR の状態は gh で読み直していない(依頼の制約)。#250 が merge 済みであることは、ローカルの main の merge commit a7bdc884 で見た
- 実機の leader が前文を計画に写し、指定の model で座席を立てるかは見ていない(計画の Non-goals)
- insight event は追記していない(依頼が報告のほかのファイルを変えないとしたため)。`--events-dir` を scratch にして作った行は次のとおりで、追記は orchestrator に任せる。`./scripts/insights-append.sh --slug split-preamble --flow standard --phase verify --verdict pass --cycle auto --source skill`
  `{"schema":1,"ts":"2026-10-22T14:11:54Z","slug":"split-preamble","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`

## Verdict

- Verdict: pass
- Verified: AC1〜AC8 をコード・文書・記録で確かめた。静的解析(`run-static-verify.sh`、go vet を go1.26.0 と go1.25.8 で、go1.25.8 の build、gofmt)はすべて rc 0。skill の 4 写しはバイト一致し、仕様と根拠の表の印(26 / 6、25 / 6)と PR の一覧(33 本)は数が合う。self-review の L1〜L4 は直っている
- Partially verified: AC9 は静的な部分だけ。テストを含む run-verify は `/test` の範囲
- Not verified: テストの実行、gh での PR の状態の読み直し、実機の leader の振る舞い
