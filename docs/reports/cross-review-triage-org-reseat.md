# Cross-review triage report: org-reseat

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-12-org-reseat.md
- Base branch: main (merge base bbe2807d)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/3(3 回目のパイプライン。2 回目の cross-review で cap を上げたとき、skill の手順どおり `cycle` は増やさず `cap` を 3 にした)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-org-reseat.md(承認 digest 95a056c00a9a)
- Self-review report: docs/reports/self-review-2026-11-12-org-reseat.md(3 回目のパイプライン)
- Verify report: docs/reports/verify-2026-11-12-org-reseat.md(pass)
- Test report: docs/reports/test-2026-11-12-org-reseat.md(pass)
- Implementation context summary: 2 回目の cross-review の ACTION_REQUIRED 1 件(立てる時点の idempotent)を def8d2a3、df8f674e、b142944d で直し、origin/main を 8460bf64 と 46dfc5a8 で取り込んだあとの 3 回目。仕分けの線引き(人の方針): 今回の変更が持ち込んだ、知らせや案内が働かなくなる向きの穴は直す。前からある問題は tech-debt と Known gaps に回す。
- 判断の前に consult(consult-plan-reseatxr3)に聞いた。判定は「2 件とも ACTION_REQUIRED で直し、パイプラインを回し直す」。足した点: disband で止まったときは打ち直しを出さない代わりに、理由に `ralph org start --plan` から立て直すことを書く(disband のあとの reseat は結びつきが無いので断られる)。leader を立てたあとの disband では、立てた座席も disband で止まったことを添える。指摘 2 は、仕様の FR-7 と org skill の「DECISION の send と read には --config を付けない」も逆向きに直す。`ApprovalDecisionCommand`(`internal/org/approve.go`)も --config を付けないが、前からの形なので今回は広げず tech-debt に回す。Disband に `reseat.lock` を取らせる案は窓を閉じるが Disband の意味が変わるので、tech-debt の候補として書く。

codex の要約(原文): "The restart phase can revive a concurrently disbanded org, and generated follow-up commands can lose the configuration needed to run. Diff whitespace checks passed; targeted Go tests could not run because the read-only sandbox denied creation of the build directory."

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `stopAll` が終わってから座席を立てるまでのあいだに、ほかのコマンドが org を disband すると、`startAll` は命を確かめ直さずに計画どおり立てる。非 leader だけの reseat では、`Spawn` から見て結びつきが無いので `spawnFeatureSeatErr` が active な leader と承認の検査を飛ばし、結びつきの無い座席が立つ。leader を立てる reseat は、元の承認を持たない新しい命を作る。`reseat.lock` は `Disband` を排他しない(`internal/org/reseat.go:1181-1185`) | 実際に起きる。`Disband`(`internal/org/verbs.go`)は `reseat.lock` も座席のロックも取らず、leader が自分の pane で打つこともある。`--seat implementer` の reseat のあいだに leader が disband すると、implementer は承認の検査を通らずに結びつきの無い org で動き、reseat は started と報告して DECISION を出す。止める側は止める直前ごとに台帳を読み直し(FR-5a)、disband されていれば座席が見つからずに止まるが、立てる側にはこの読み直しが無い。止めてから立てるまでの窓は今回の変更が持ち込んだもので、人が disband で止めた org を reseat が黙って動かし直す。直し方: 計画の時点の org の最後の `disbanded` の位置を計画に持たせ、各座席を立てる直前に台帳を読み直して比べる。違えば、その座席から先は立てず、disband されたことを言う理由で止める。読み直しから `Spawn` の書き込みまでの窓は残る(止める側の FR-5a と同じ型)ので、Known gaps に書く | internal/org/reseat.go、internal/org/reseat_start_test.go(または reseat_test.go)、internal/cli/org_reseat.go |
| 2 | [P2] `--config /path/valid.toml` で reseat が成功しても、作業ディレクトリの `ralph.toml` が壊れているか読めないと、出力の DECISION の `ralph org send` は leader に知らせずに失敗する。新しいコメントの説明と違い、`org send` は `newOrgRuntimeAt` で既定の設定を読み込んで検証してから送る。出力の `ralph org read` も同じで、案内どおりに打つと立て直した座席が TASK を受け取らないままになる。打ち直しのコマンドと同じく、絶対パスにした `HintConfigPath` を両方に渡す(`internal/org/reseat.go:1359-1360`) | 実際に起きる。`ralph org send` と `ralph org read` はどちらも `newOrgRuntime` → `resolveOrgConfig`(`internal/cli/org.go`)で `--config`、無ければ `./ralph.toml` を読み、読めない・壊れていれば失敗する。help(`internal/cli/org_reseat.go` の「read no ralph.toml and do not repeat it」)と `HintConfigPath` のコメントは偽で、今回の変更が持ち込んだ。案内どおりに打った DECISION が失敗すると、leader は座席が立て直されたことを知らず、TASK を送り直さない。直し方は 1 か所ずつで小さい: `ReseatDecisionCommand` と、CLI が出す `ralph org read` に `--config` を付け、help とコメントを直す | internal/org/reseat.go、internal/cli/org_reseat.go、internal/cli/org_reseat_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
