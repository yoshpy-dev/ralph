# sync-docs report: review-switches-config

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-review-switches-config.md`(issue #265。承認の digest `cc42f13b59d0` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` に足した 1 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `da9d1b3a`、branch は `feat/review-switches-config`。run 開始時の HEAD は `5f458a87`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-29-review-switches-config.md`(0a5aa556 と 234f15c1 の 2 回。run 2 は Merge: OK、MEDIUM 0)、`docs/reports/verify-2026-10-29-review-switches-config.md`(5f38f0c9、pass、LOW の V1 と情報の V2〜V5)、`docs/reports/test-2026-10-29-review-switches-config.md`(b6bb3bdd、pass。mutant 19 個のうち 18 個が落ち、k だけ生き残った)
- 差分の大きさ(`git diff da9d1b3a...HEAD --shortstat`、この run の前): 36 ファイル、+1719/-203。walkthrough を書くかどうかは `/pr` が決める

## Summary

この branch は issue #265 の対応で、`ralph.toml` に 2 つの表を足した。`[standard]` は `plan_advisory` と `cross_review`、`[org.review]` は `plan_advisory`、`cross_review`、`max_cross_review_cycles` を持つ。既定は false、false、false、false、2 で、2 は最初の review を含む回数の上限である。値を読む動詞 `ralph config get <key>` も足した。`ralph org spawn` は座席の pane(leader を含む)に `RALPH_PRIMARY_CLI=<driver>` を渡す。この repo のルートに `ralph.toml` を新しく置き、4 つの実施有無を true にした。どの skill もまだ値を読まないので、標準フローの動きは変わらない(読み始めるのは #266)。

現在形の文書を洗った結果、この branch が足した記述(README、AGENTS.md、`/org` skill の 4 つの写し、isolated-herdr-server recipe とその写し、spec の FR-7 の注)は、実装と食い違っていなかった。直したのは 3 か所である。

1. verify の V2。`templates/base/ralph.toml` の `[standard]` のコメントが、`ralph config get --help` より狭い約束を書いていた。help に合わせた。
2. `.claude/rules/ralph/model-routing.md` の「Where the values live」にある `defaults_sync_test.go` の行。このテストが検査するものを列挙した行で、この branch が `[standard]` の照合と、上限と `RALPH_STANDARD_MAX_PIPELINE_CYCLES` の照合を足したのに、書き足されていなかった。
3. 計画の `## Progress checklist` に、この report の行を足した。

## Changes made

| File | Change |
|------|--------|
| `templates/base/ralph.toml:19-29` | `[standard]` のコメントの終了コードの段落を、`ralph config get --help` の約束に合わせた。旧: exit 2 は「an unknown key」だけ、「記録する」と「ほかの exit 1 は止まる」がなかった。新: 古い ralph の exit 1 と `unknown command "config"` だけを off として扱い、そのことを記録する。ほかの exit 1、exit 2(unknown、table、array のどれでも)、exit 3 では止まって報告する。コメントだけで、キーと値は変えていない。`[org.review]` のコメントは「[standard] above と同じに扱う」と参照しているので、変えていない(V2) |
| `.claude/rules/ralph/model-routing.md:148` | `defaults_sync_test.go` の行に 3 点を足した。`[org.review]` も `orgTemplateDrift` の対象であること、`standardTemplateDrift` が `[standard]` の 2 キーをテンプレートと `Default()` で照合すること、`[org.review].max_cross_review_cycles` の既定が ralph-config.sh の `RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定と同じであること。ralph-config.sh の既定を変える人が、このテストで落ちる理由を先に読めるようにした。root だけの写し(`KNOWN_DIFFS` の対象。雛形の model-routing.md にこの行はない) |
| `docs/plans/active/2026-10-29-review-switches-config.md` | `## Progress checklist` に「Sync-docs report created」の行を足した(本文の節は触れていない。digest は同じ) |
| `docs/reports/sync-docs-2026-10-29-review-switches-config.md` | この report |
| `docs/insights/events/2026-10-28-review-switches-config.jsonl` | `sync_docs` の行を `insights-append.sh` で足した(file 名は UTC の日付で、self-review・verify・test の行と同じ file) |

ルートの `ralph.toml` は変えていない(4 つの実施有無だけを持つ)。Go のコードと help の文言も変えていない。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md:128`(`ralph config get <key>` の行) | 変更なし(同期済み) | 読む file(`--config`、なければ `./ralph.toml`)、キーなし・file なしは既定、exit 2 は unknown・table・array、exit 3 は読めない・parse できない・検証で落ちる、が `internal/cli/config.go:46-66` の help と合う。「古い ralph だけを off、2 と 3 は止まる」も同じ向き。「ほかの exit 1」には触れないが、「only an older ralph」の書き方で誤りにはならない |
| `README.md:259`(director の段落) | 変更なし(同期済み) | 座席の pane は `RALPH_PRIMARY_CLI=<driver>` も持ち、director の pane は持たない、と書く。director の launch script が出す 2 つを「the same two values ralph gives every herdr pane it creates」と呼ぶ文は、座席の pane が 2 つに加えて 1 つ持つだけなので、今も正しい |
| `AGENTS.md:86`(`internal/cli/` の列挙) | 変更なし(同期済み) | `config` が入っている。`.ralph/core/AGENTS.core.md`、`templates/base/AGENTS.md`、`CLAUDE.md` に subcommand の列挙はない(`git grep` で確かめた) |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 つの写し | 変更なし(同期済み) | 前提の節が pane の 3 つ目の env、`--env` のない herdr の警告、`[org.review]` と `ralph config get` を書く。`check-skill-sync.sh` は 14 skill で lock-step。skill の本文は触れていないので `sync-skills.sh` は走らせていない |
| `.claude/skills/director/SKILL.md`(と写し)、`docs/recipes/director-launch-cmd.md`(と写し) | 変更なし(矛盾なし) | どちらも headless の director の pane と launch script の話で、env は 2 つ。director は `RALPH_PRIMARY_CLI` を持たないので正しい |
| `docs/recipes/isolated-herdr-server.md`、`templates/base/docs/recipes/isolated-herdr-server.md` | 変更なし(同期済み) | 3 つ目の env と、`--env` のない herdr で 3 つとも付かないことを書く。2 つの写しは `check-sync.sh` で byte 一致 |
| `docs/recipes/codex-setup.md:66-76`、`.codex/AGENTS.override.md:26`、`.claude/rules/ralph/subagent-policy.md:88`、`.claude/skills/cross-review/SKILL.md:38`・`:179` | 変更なし(矛盾なし) | 人の対話セッションで `RALPH_PRIMARY_CLI` を自分で設定する話。org の座席に自動で付く話ではないので、書き足さない |
| `.claude/rules/ralph/git-commit-strategy.md:34-38`・`:53-56` | 変更なし(矛盾なし) | `RALPH_WIP_CHECKPOINT` だけの話で、「every herdr pane」は今も正しい |
| `.claude/rules/ralph/ralph-workflow.md`、`templates/base/` の写し | 変更なし(依頼) | 「Codex advisory is optional ... silently skipped」と「`/cross-review` (optional)」は、#266 が入るまで正しい。依頼で文言を変えない |
| `.claude/rules/ralph/model-routing.md:148` | 直した | 上の Changes made |
| `docs/specs/2026-10-07-org-multi-org-director.md:129`(FR-7 の「2026-10-29 に決めたこと」) | 変更なし(同期済み) | `[org.review]` の 3 キー、既定、数え方、`ralph config get`、pane の 3 つ目の env、director の pane には渡さないこと、この版では読む座席がまだないこと |
| `docs/specs/2026-08-01-org-runtime.md:157`(「3 面ロックステップ」) | 変更なし(この branch の外) | `ralph-config.sh` に残る `[org]` の既定は #273 から `agmsg_home` だけで、この行は #273 より前の言い方のまま。この branch は `[org.review]` の上限を shell の既定に照合するが、新しい `[org]` の既定を shell に足してはいない |
| `docs/quality/definition-of-done.md:32`、`docs/quality/quality-gates.md` | 変更なし | pipeline の cap と順序の話で、標準フローの `RALPH_STANDARD_MAX_PIPELINE_CYCLES` は変わっていない。新しい検査は足していない |
| `docs/architecture/repo-map.md` | 変更なし | subcommand と `ralph.toml` の節を持たない。新しい script はない。`KNOWN_DIFF` の件数(5 から 6)を書いた箇所は docs にない(`docs/evidence/` の過去の log だけ) |
| `docs/tech-debt/README.md` の `SeatPaneEnv()` と `TabCreate` を名指しする行(:193、:269、:323、:336) | 変更なし(矛盾なし) | `SeatPaneEnv()` は今も同じ 2 つの値を返す。:269 の「ralph passes none to herdr」は `RALPH_ORG_AGMSG_HOME` の話で、今も正しい |
| `scripts/ralph-config.sh:45-54` の `[org]` の既定の注 | 変更なし(計画の Non-goals) | 「This file sets no other [org] default」は今も正しい。上限は標準フローの変数との照合で、`[org]` の既定を shell に足したものではない |
| `scripts/check-sync.sh:112-116`(`KNOWN_DIFFS` の注) | 変更なし(同期済み) | 「root holds only the four review switches ... turned on」。ルートの実物は 4 キーだけ(`ralph.toml`) |
| `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-template-purity.sh` | 通る | 下の Verification |

## Info(直さなかったもの)

- `README.md:128` の「The skills and seats that read a setting with it treat only an older ralph ... as off」は、この版では読む skill がないので、実際には誰にも当たらない。条件付きの書き方なので誤りではなく、#266 が入ると当たり始める。共有の file なので触れなかった。
- verify の V1(PATH に `ralph` がないときの exit 127 の扱いが、計画の Assumptions と help で食い違う)は、読む skill がまだないので害がない。#266 の計画で決める。
- verify の V3(`/org` skill は終了コードの約束を書かない)は、計画の D4 のとおり。#266、#267、#269 が skill を読む形に変えるときに書く。
- verify の V4(計画の AC1〜AC6 のチェックと Scope の「テンプレートの写し」)は、plan の本文が digest の対象なので触れていない。AC1〜AC6 は `[x]` になっている。Scope の古い言い回しは Progress の Deviation(M1)が訂正している。
- test の mutant k(壊れた `ralph.toml` に知らないキーを渡すと 3 になる、という順序を押さえるテストがない)と、test の Test gaps の `ralph config` を引数なしで打つ場合のテストの欠けは、LOW で、PR の Known gaps に載せる。Go は触れない依頼なので、ここでは足していない。

## Plan drift notes

- 計画の Scope・Design decisions・Acceptance criteria の本文は digest の対象で、この run は触れていない。残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける。
- 実装中に計画の外へ出た点は、Progress の S1〜S4、M1 の Deviation、L3 の決定、F1 に記録済み(ルートの `ralph.toml` をテンプレートの写しから 4 つだけの形に変えたこと、`cross_review` の上限の数え方、`TabCreate` が受ける `extraEnv`、`internal/cli` の 2 つのテストの argv の書き換え)。この run で新しく見つかった差はない。
- verify の V2 の直しは、計画の D6(終了コードの約束)を変えず、テンプレートのコメントを D6 と help に合わせただけである。

## Follow-ups(この PR では足していない)

- ルートの `ralph.toml` は `check-sync.sh` の `KNOWN_DIFFS` に入っている。`KNOWN_DIFFS` は file 全体を飛ばすので、ルートの中身がテンプレートからずれても `check-sync.sh` は報告しない(`docs/tech-debt/README.md:117` の行と同じ形)。さらに、ルートの `ralph.toml` を読む常設のテストも検査もない(`git grep` で `tests/`、`scripts/`、`.github/` と Go のテストを確かめた。今の 4 つの true は S4 の検証コマンドと verify の tomllib の照合で確かめただけ)。テンプレートが 4 つのうちのキーを改名すると、ルートに古い名前が残り、`config.Load` は知らないキーを無視するので、この repo の切り替えが黙って off になる。#266 が最初にこの値を読む PR なので、そこでルートの 4 つを `Default()` に重ねた値と比べるテストを足すのが自然である。tech-debt の行は、共有の file の最小の変更という依頼に従って足していない。足すかどうかは `/pr` の前に lead が決める。Trigger は「#266 の計画」か「`[standard]` / `[org.review]` のキーの改名」になる。
- `docs/specs/2026-08-01-org-runtime.md:157` の「3 面ロックステップ」は #273 から不正確なまま残っている(この branch の外)。

## Release notes(`/pr` の本文に置く)

verify の V5 と、計画の Rollout or rollback notes に基づく。

1. 設定の表を 2 つ足した。`[standard]` は `plan_advisory`(既定 false。標準フローの `/plan` の計画助言)と `cross_review`(既定 false。`/cross-review`)。`[org.review]` は `plan_advisory`(既定 false。org の leader の計画助言)、`cross_review`(既定 false。org の reviewer 座席の cross-review)、`max_cross_review_cycles`(既定 2、1 以上)。上限は cross-review を回す回数の全部で、最初の 1 回を含む(`RALPH_STANDARD_MAX_PIPELINE_CYCLES` と同じ数え方。2 は最初の review と、直したあとの review 1 回)。0 以下は `ralph.toml` の読み込みでエラーになる。
2. 既存の project は表を自分で書き足す。`ralph.toml` は seed なので、`ralph upgrade` は既存の `ralph.toml` に表を足さず、template の変更を advisory diff で示す。表がなくても既定の off が効くので、動きは変わらない。on にしたい project は表ごと書く(`[org.approval]` の release のときと同じ書き方)。
3. この版では、どの skill も設定を読まない。標準フローの動きは変わらない。
4. #266 が入ると、標準フローの計画助言と cross-review は既定の off で飛ばされる。今の動き(codex が使えるなら計画助言と cross-review を回す)を続けたい project は、`[standard]` の 2 つを true にする(この repo は root の `ralph.toml` で 4 つを true にした)。
5. #266 の skill は新しい binary の `ralph config get` を使う。PATH の `ralph` が古いと(release 済みの v6.0.0 を使っている場合も)、設定を on にしても off として扱われる。新しい binary(`go install ./cmd/ralph` か次の release)が要る。
6. 新しい動詞 `ralph config get <key>`。`--config` の file、なければ `./ralph.toml` を読み、キーは `standard.cross_review`、`org.review.max_cross_review_cycles` のような点つきの toml の名前。値を 1 行で stdout に出す(true/false、10 進の数、文字列)。キーのない file と、file がない場合は既定を出す。終了コード: 0 は値、2 はキーが 1 つの値のキーでない(知らない、table、array)、3 は `ralph.toml` を読めない・parse できない・検証で落ちる。動詞のない古い ralph は exit 1 と `unknown command "config"` で終わり、skill はそれだけを off とみなして記録する。ほかの exit 1、2、3 では止まって報告する。注意: cwd の `./ralph.toml` を読み、ないときは黙って既定(off)を出すので、skill は repo の root から打つ前提になる(#266 の計画に入れる)。
7. `ralph org spawn` は座席の pane(leader を含む)に `RALPH_PRIMARY_CLI=<claude|codex>` を `herdr tab create --env` で渡す(`AGMSG_SELF_NAME=off`、`RALPH_WIP_CHECKPOINT=off` のあと)。座席の cross-review が、もう一方の CLI を選ぶ `pick_reviewer` に座席の driver を読ませるためで、値がないと codex の座席の driver を claude とみなして取り違える。headless の director の pane には渡さない。`--env` のない herdr では、今までの 2 つと同じく env なしで pane を作り、stderr に警告を 1 行出す。
8. この repo のルートに `ralph.toml` を足した(4 つの実施有無が true。この repo だけの file で、`ralph init` が配るのは `templates/base/ralph.toml`)。

自動の changelog(`.goreleaser.yml` は `^docs:`・`^test:`・`^chore:` を除く)に載る commit は、この branch の `feat:` 3 本と `fix:` 2 本である。

| commit | 題 |
|--------|----|
| `eff0340d` | `feat: add the [standard] and [org.review] review switches to ralph.toml` |
| `8e51e1c9` | `feat: add ralph config get to read ralph.toml values` |
| `33d859bb` | `feat: pass the seat driver to its pane as RALPH_PRIMARY_CLI` |
| `10a05dfe` | `fix: keep the root ralph.toml to the four review switches and correct the review-switch wording` |
| `2a6cb022` | `fix: correct the review-switch wording in spawn's TabCreate comment` |

`feat:` の 3 本は題が内容を言っている。`fix:` の 2 本は、同じ branch の未 release の機能の文言とこの repo だけのルートの `ralph.toml` を直したもので、利用者に見える不具合の修正ではない。changelog に「fix」として出るので、release のノートの 1〜8 を先に読ませるか、手で整理する。`docs:` の commit(self-review、verify、test、sync-docs の report と Progress の記録、文書の直し)は載らない。この run の commit も `docs:` である。

PR の Notes for reviewers に入れる点: D4 の外れ(issue の「古い binary では off として扱うことを skill に書く」は、この PR では skill に書かず、動詞の help とテンプレートのコメントに書いた。skill への組み込みは #266)、verify の V1、test の mutant k、S3 の遅れた指摘(警告は Herdr の値ごとに 1 回で、最初の `TabCreate` が director の pane だと座席の節が出ない。director は別のプロセスの Herdr なので実害はない見込み)、実物の herdr で pane の env を確かめていないこと。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。テンプレートの `ralph.toml` と model-routing.md は `KNOWN_DIFF` の対象なので、この run の 2 つの編集は byte 比較の対象外 |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-template-purity.sh` | rc 0。`PASS: no meta-repo-specific references found in templates.` |
| `git diff --check` | rc 0、出力なし(commit の前、追跡ファイルの変更に対して) |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/config/ -run 'LockStep\|TemplateDrift'` | rc 0。`ok ... 0.692s`。`TestDefaultsLockStep`(`org`、`standard`、`shell_and_skills` の 3 つのサブテスト)、`TestOrgTemplateDrift_DetectsChangedAndMissingKeys`、`TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`、`TestStandardTemplateDrift_EachKeyIsComparedOnBothSides` が PASS。テンプレートのコメントの編集後に打った |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-29-review-switches-config.md` | `cc42f13b59d0`(`- Approved:` 行の値と同じ。Progress の編集の前後で同じ) |
| 現在形の文書の `git grep` | `RALPH_PRIMARY_CLI`、`RALPH_WIP_CHECKPOINT`、`AGMSG_SELF_NAME`、`defaults_sync_test`、`RALPH_STANDARD_MAX_PIPELINE_CYCLES`、`ralph.toml`、subcommand の列挙を、README、AGENTS、CLAUDE、`.ralph/core`、`docs/recipes`、`docs/quality`、`docs/architecture`、`docs/specs`、`docs/tech-debt`、`.claude/rules`、`.claude/skills`、`.codex` で打った。`docs/evidence/`、`docs/reports/`、`docs/plans/archive/` の過去の記録は、当時の状態の記録なので洗っていない |
| `go build`、golangci-lint、`run-*.sh`、全体の `go test` | 打っていない。依頼で止められた。この run が変えたのはテンプレートのコメント、rules の 1 行、計画の Progress、この report、insight event で、コードの動きは変わらない |
| insight event | `./scripts/insights-append.sh --slug review-switches-config --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で追記する(結果は commit のメッセージの前に確かめる) |
