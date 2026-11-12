# Verify report: split-feature-digest

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-split-feature-digest.md`(issue #312 の受け入れ条件 3。`- Approved:` の digest `5a95d18e25ff` は `./scripts/plan-visual.sh digest` で出し直した値と一致した)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff 9ac72c50...HEAD`(HEAD `890ac5c2`、self-review の report のコミットまでの 8 コミット、18 ファイル)。静的解析は `./scripts/run-static-verify.sh` の既定(変わった言語だけ、今回は golang)。テストは走らせていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-11-12-split-feature-digest.log`(git が無視する場所。run-static-verify.sh の全出力と、下の観察の probe の出力)
- Insight event: `docs/insights/events/2026-11-12-split-feature-digest.jsonl` に verify の 1 行(cycle 1、pass、low 4)を足し、この report と同じコミットに入れた
- ID の付け方: `V1` から `V4`(LOW、/sync-docs に渡す drift)、`I1`(INFO)

## Spec compliance

受け入れ条件ごとに、コードと、それを固定するテストのソースを読んで判定した。テストを走らせて通ることは `/test` が確かめる。「満たす(静的)」は、コードがそう動き、そのとおりの期待値を持つテストがあるという意味である。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 機能ごとの digest の入力と、変わる編集・変わらない編集 | 満たす(静的) | `featureDigest`(`internal/org/split.go:654`)は前文、slug、Type、Reserve、Depends on、Body を `<名前> <長さ>\n<値>\n` で SHA-256 にかけて先頭 12 桁を取る。`finishSplitPlan` が Type の既定と `trimBlankLines` のあとで呼ぶ。`TestFeatureDigest_ChangesOnlyWithTheFeatureAndPreamble`(`split_test.go:816`)が、変わらない 13 行(ほかの機能の本文と Reserve、題、ヘッダの Owner、`## Features` の前置き、後ろの節、最後の機能の後ろへの追加、前後の空行、CRLF、既定の Type を書いたもの、同じ Reserve の別の綴り)と変わる 9 行(前文の文と見出し、本文、Type、Type の削除、Reserve、Depends on とその並び)を持つ。前文の無い計画と本文の無い機能は `TestFeatureDigest_NoPreambleAndEmptyBody`(`:930`)、符号化のバイト列は `TestFeatureDigest_Encoding`(`:957`) |
| AC2 `feature_digest=` を `approval=` の後ろに書き、読み戻す。2 つある token・空の値は欄を空にする | 満たす(静的)+ 観察 | `scopeReservedDetails`(`reserve.go:319`)と `featureBindingFromTokens`(`:398`)。`TestScopeReservedDetails_FeatureDigestTokenLast`(`reserve_test.go:465`)、`TestFeatureBinding_FeatureDigestRead`(`:533`、2 つ・空・メモの後ろ・branch の直後)。観察: HEAD の binary に `feature_digest=` を 2 回書いた記録を読ませると、`status --json` の `feature` に `feature_digest` が出ず、`incomplete` も付かなかった |
| AC3 `sameFeature` の比べ方と `Complete()` の 5 欄 | 満たす(静的) | `sameFeature`(`reserve.go:176`)は両方に `Complete` を求め、split・feature・branch・worktree・承認方式を比べたあと、両方が機能ごとの digest を持つときだけそれで比べ、それ以外は全体の digest で比べる。`Complete()` は 5 欄のまま。`TestSameFeature`(`reserve_test.go:699`)が両方向で 8 つの組と不完全な 5 種を持つ。D4 の回帰は `TestFeatureBinding_OlderRecordStillBound`(`:572`、`ActiveFeature`・`gateOrgLife`・`collectMergeTargets`) |
| AC4 ほかの機能の節だけを直して承認し直した計画で、止めた org を立て直せる。依存先の節でも同じ | 満たす(静的) | `TestStartFeature_OtherFeatureChangedRestarts`(`feature_test.go:1266`)の 2 つの場合(ほかの機能、依存先の機能)。leader が立ち、`scope_reserved` は 1 件のまま、binding は最初の start のまま、承認済みの implementer の spawn が通ることを見る |
| AC5 前文かその機能の節を直した start は断られ、何も記録しない。拒否の文 | 満たす(静的) | `featureDigestRefusal`(`reserve.go:652`)。`TestStartFeature_OwnSectionOrPreambleChangedRefused`(`feature_test.go:1329`)が、節と前文の 2 つの場合で、拒否の文全体(節か前文が変わったこと、前と今の機能ごとの digest、disband か別の org_id)と、記録も呼び出しも増えないことを見る |
| AC6 古い記録の org は、ほかの機能の節だけの変更でも断る。拒否の文 | 満たす(静的) | `TestStartFeature_OlderRecordRefusedAfterOtherFeatureChanged`(`feature_test.go:1369`)。文は「its record has no feature digest (an older ralph started it), every change to the split plan is refused; disbanding the org and starting it again records one」。計画を戻すと立ち、古い記録のまま残ることも見る |
| AC7 承認し直さない start は `CheckApproved` で断られる | 満たす(静的) | AC4 のテストの中で、直しただけの計画の start が "changed after its approval: approved digest" で断られ、何も記録しないことを見る。`CheckApproved` と `PlanDigest` は diff に無い |
| AC8 古い binary も 5 欄と承認方式を読める | 満たす(静的)+ 観察 | `TestScopeReservedDetails_OlderBinaryReadsFiveTokens`(`reserve_test.go:512`)の読み手 `oldBindingTokens` は、base(9ac72c50)の `featureBindingFromTokens` と同じく、知らない鍵の最初の token で読み終える(base のソースと突き合わせた)。観察: base から build した binary に新しい記録を読ませると、`status --json` の `feature` に 5 欄がそろい、`incomplete` は付かなかった |
| AC9 `status --json` の `feature_digest`(古い記録では出ない)、`start` と `status` の `--help` | 満たす + 観察 | `orgFeatureJSON`(`internal/cli/org.go:1995`、`omitempty`)。`TestOrgStatus_FeatureLineOnlyForBoundReservation`(`org_feature_test.go:609`)、`TestOrgStartAndStatusHelp_FeatureDigest`(`:569`)。観察: HEAD の binary で、新しい記録は `feature_digest` を出し、古い記録は出さなかった。`org start --help` と `org status --help` の文は下の drift の表のとおり |
| AC10 gate.go と regate.go の既存のテストが変更なしで通る | 一部(静的): 変更なしまで確かめた | diff に gate・regate・merge-order のコードもテストも無い(`git diff --name-only` で 0 件)。呼び出し側 `gate.go:688` と `regate.go:978` も変わっていない。通るかどうかは `/test` |
| AC11 org skill の 4 つの写しが新しい規則を言い、check-skill-sync が通る。機能をまたぐ約束は前文に書く | 満たす | 4 つの写しの `shasum` は同じ(`43d03de5…`)。`check-skill-sync.sh` は 14 skill で ok。`.claude/skills/org/SKILL.md` の 777 行目が機能をまたぐ約束を前文に書くよう言い、916〜921 行目が立て直し、931〜939 行目が拒否を言う |

計画の Assumptions のうち、判定に効く 2 つも確かめた。

- 「全体の digest が同じなら機能ごとの digest も同じ」: `PlanDigest`(`split.go:137`)は CR を残したまま行を hash し、飛ばすのは `- Status:`・`- Approved:`・`- Branch:`、`## Progress checklist` の節、`- [x]` の印だけである。分割計画はこれらを先頭の 2 行のほかは読み込みで拒否し、機能ごとの digest はヘッダを入れない。そのため主張は成り立つ。`sameFeature` の注釈の「never looser」も同じ理由で正しい。
- 「binding の全体の digest を今の分割計画と比べる読み手は無い」: 本番のコードで `.Digest` を読む箇所を grep した。binding の欄を読むのは `reserve.go`(比較・書き出し・文)、`feature.go`(書く側)、`internal/cli/org.go`(status の JSON)だけで、`approval.go` の `Digest` は機能の計画の `PlanRef` で別物だった。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0、53 秒) | 既定の changed scope。言語は golang。`uptime` の load average は 9〜13 で、時間切れは無かった |
| ・ shellcheck、hook の `sh -n` 20 本、guard の awk、settings.json 2 本の `jq -e` | ok | |
| ・ `scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0 |
| ・ `scripts/check-pipeline-sync.sh`、`scripts/check-skill-sync.sh`、`scripts/check-template-purity.sh` | ok | skill は 14 本がそろう |
| ・ tech-debt README の plan の参照 | ok | |
| ・ golang: `gofmt -l`、`go vet ./...`、`golangci-lint run ./...`、`staticcheck ./...` | ok | gofmt ok、golangci-lint 0 issues、staticcheck は出力なし |
| ・ `secret-scan-branch.sh` | clean | 9ac72c50..890ac5c2 |
| `git diff 9ac72c50...HEAD --check` | clean | |
| 変わった 18 ファイルの U+FFFD の検査 | 0 件 | |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| org skill の 4 つの写し | はい | 4 つとも同じバイト列。status の行(363 行目)、予約の記録の欄(465 行目、873 行目)、分割計画の節(777 行目、807 行目)、立て直し(916〜921 行目)、拒否(931〜939 行目)が新しい規則を言う |
| `ralph org start --help` | はい | HEAD から build した binary で読んだ。止めた org の立て直し、前文とその機能の節、承認方式、依存先を含むこと、台帳に何も足さないこと、古い記録は今までどおり断ることを言う |
| `ralph org status --help` | はい | `feature_digest` と、古い記録では出ないことを言う。重複か空の token の記録でも出ないことは help に無いが、self-review の L4 のとおり誤りではない |
| `docs/specs/2026-10-07-org-multi-org-director.md` の FR-4 | いいえ(V1) | FR-4 の注に、この変更の決定(2026-11-12)が無い。今の注は「4 段目で決めたこと」が結びつきを 5 欄と書き、7 段目の前半の注が `approval=` を足した形で止まっている。FR-4 本文の「digest が承認時の値と一致しなければ start を拒否する」と、受け入れ条件「本文が承認後に変わった start は拒否される」は今も正しい |
| `AGENTS.md` の `internal/org/` の行 | ほぼ(I1) | 「split plans and their approval digest (`split.go`)」は誤りではないが、機能ごとの digest を言わない。計画は「触るかは sync-docs で決める」としている |
| `docs/tech-debt/README.md` の Trigger | いいえ(V2) | この diff が満たした Trigger に、持ち越しの記録も支払いも無い(下の V2) |
| `docs/tech-debt/README.md` の 220 行目の項目の文 | いいえ(V3) | `--json` の `feature` が運ぶ欄を「split plan, slug, digest, branch, and worktree」と書く。今は `feature_digest` も運ぶ |
| `internal/org/seat.go:65-74` の `EventScopeReserved` の注釈 | いいえ(V4) | binding の token を `split= feature= digest= branch=` の 4 つと書く。`approval=` が無いのは前からで、今回 `feature_digest=` も抜けた。詳細は `reserve.go` の `scopeReservedPathsKey` を指している |
| `docs/quality/quality-gates.md` の Split plan approval の行、README、director skill、leader の雛形 | はい | どれも binding の比べ方を書いておらず、食い違う文は無い |

### V2 の内訳(この diff が満たした tech-debt の Trigger)

`/sync-docs` が、各行に持ち越しの記録を足すか、支払うかを決める。行番号は HEAD の `docs/tech-debt/README.md`。

1. 180 行目の (a)「the next edit to the `/org` skill's command table」: `status` の行を 623 字から 689 字にした(4 つの写し)。表の作り直しはしていない。
2. 200 行目の (k)「the next edit of the `parseSplitPlan` comment or of the header comment of `split.go`」: `split.go` の先頭の注釈に機能ごとの digest の段落(54〜60 行目)を足した。(k) が直すよう言う段落(62〜66 行目)は変わっていない。
3. 223 行目の (c)「the next change to `reserveAgain` or `featureBindingFromTokens`」: `featureBindingFromTokens` の鍵の表と注釈を変えた。(c) が求めるテスト(読めない承認方式の binding を補償で書き戻し、`RequiresApproval` が保たれることを見る)は足していない。
4. 396 行目「the next edit of one of the callers」: 呼び出し側の `feature.go`・`split.go`・`feature_test.go` を変えた。`resolvedOrClean` の呼び出しは触っていない(前の持ち越しの記録と同じ形)。
5. 192 行目の (c)「the next change to the `Long` of `ralph org status`」: status の Long を変え、新しい文だけを cobra の出力から確かめるテスト(`TestOrgStartAndStatusHelp_FeatureDigest`)を足した。既存の文(`reserved:` の行など)は確かめていないので、一部を払った形である。閉じるか持ち越すかを決める。
6. 201 行目の (e)・(f)「the next change to `runOrgStartPlan`, `StartFeature`, or `scriptFeatureWorktrees`」: `StartFeature` の doc の注釈だけを変えた(`startFeatureLeaderParams` も 1 行変えた)。2026-11-06 の記録は呼び出しと doc の両方の変更を数えている。注釈だけの変更を数えるかは判断が要る。

満たしていないと確かめたもの: 203 行目の (a)(diff は `TestStartFeature_WorktreeRecordMismatchRefused` の前の行を文脈として含むだけで、その 2 つのテストは変えていない)、365 行目(`verbs_test.go` は 5892 行から 5893 行。`Send` と leader の雛形のテストは変えていない)、200 行目の (c)(`splitFeatureDraft` 自体は変えておらず、埋め込んだ `SplitFeature` に欄を足しただけ)。

## Observational checks

HEAD(890ac5c2)と base(9ac72c50)から binary を build し、scratch の台帳に `scope_reserved` を 3 件書いて `ralph org status --org-id <id> --state-dir <scratch> --json` を打った(`env -i` で PATH を `/usr/bin:/bin` に絞り、HOME も scratch)。

- 新しい記録(`approval=per-plan feature_digest=0a1b2c3d4e5f`): HEAD は `feature_digest` を出した。base は 5 欄を出し、`incomplete` を付けなかった(AC8)。
- 古い記録(`feature_digest=` 無し): HEAD は `feature_digest` を出さなかった(AC9)。
- `feature_digest=` を 2 回書いた記録: HEAD は `feature_digest` を出さず、`incomplete` も付けなかった(AC2、D6)。

出力は evidence の log の「Observational probes」の節にある。start の立て直しを実機の herdr で回すことはしていない。

テストのファイルの行数は、`reserve_test.go` が 764 行から 1000 行に、`split_test.go` が 997 行から 1202 行になった。規則の 800 行の目安を超えたが、diff の質の判断なので所見にはしない(self-review も挙げていない)。

## Coverage gaps

- テストは走らせていない。AC1〜AC11 のテストが通ること、AC10 の gate と regate の既存のテストが通ることは `/test` が確かめる。
- 止めた leader の立て直しを、実機の herdr と本物の `ralph-worktree.sh` で回していない。テストは偽の herdr と worktree で回す。
- 古い binary の補償が binding を書き戻して `feature_digest=` を落とす道(計画の Risks の 5 つ目)は、全体の digest の比較に戻るだけで断る側に倒れる、とコードから読んだ。実際の古い binary で補償を起こしてはいない。
- `featureLeaderTask` に欄が増えたとき `featureDigest` にも足す約束は、両方の注釈にだけある。欄の増加を機械で捕まえる検査は無い(self-review の Coverage gaps と同じ)。足すなら、`featureLeaderTask` が読む `SplitFeature` の欄の集合と `featureDigest` が読む集合を reflect で比べるテストが、いちばん小さい。

## Verdict

- Verified: AC1〜AC9 と AC11 はコードとテストのソースから満たす。AC2・AC8・AC9 は build した binary でも観察した。静的解析はすべて通った。org skill の 4 つの写しと `start`・`status` の help は新しい規則と合っている
- Partially verified: AC10 は、gate・regate・merge-order のコードとテストが変わっていないことまで。通るかは `/test`
- Not verified: テストの実行結果。実機の herdr での立て直し。古い binary の補償の道
- Drift(/sync-docs へ): V1 FR-4 の決定の注記が無い。V2 tech-debt の Trigger 6 件。V3 tech-debt の 220 行目の項目の文。V4 `seat.go` の `EventScopeReserved` の注釈。I1 `AGENTS.md` の `split.go` の説明は任意
- Verdict: pass
