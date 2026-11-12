# upgrade-behavior-notes

- Status: Approved
- Approved: 2026-11-13 sha256:0de679dfe2a1
- Owner: Claude Code
- Date: 2026-11-13
- Related request: 下流の運用 issue の系列、レーン B の 3 本目。#313(PR #316)で upgrade の報告に `Version: <from> → <to>` だけを足し、「版ごとに挙動が変わること」の注記は /release の運用と合わせて別にすることにした(ユーザーの決定)。その別 issue
- Related issue: https://github.com/yoshpy-dev/ralph/issues/317
- Type: feat
- Branch: feat/upgrade-behavior-notes

## Objective

`ralph upgrade` が書く報告(`docs/reports/upgrade-<version>-<date>.md`)の先頭、`Version: <from> → <to>` の行の次に、上げる前と後の版のあいだの release が持つ「挙動が変わること」の注記を並べる。注記は release ごとに 1 つのファイルとして repo に置き、binary に埋め込む。/release の手順に、その注記を書いてから tag を打つ段を足す。

STOIC では 7.0.0 から 8.1.0 に上げると director の skill の「座席の driver と model は `[org.role_models]` と違う座席の分だけを書く」が変わったが、報告からは版しか読めず、どの手順が変わったかが分からなかった。

## Scope

- 注記の置き場所: `internal/upgrade/notes/<X.Y.Z>.md`(その release で、雛形の skill・rule・role prompt などの挙動が変わる点を短い箇条書きで書く)。`internal/upgrade` の `//go:embed notes/*.md` で binary に入れる
- 選び方: 上げる前の版 < 注記の版 ≤ 上げた後の版(semver、`golang.org/x/mod/semver`)の注記を、版の古い順に並べる。上げる前か後の版が semver として読めない(`dev`、空)ときは並べない
- 報告: `UpgradeReportData` に注記の一覧を足し、`Version:` の行の次に `## Behavior changes` の節(版ごとに `### <X.Y.Z>` と本文)を出す。注記が無ければ節を出さない
- stdout: `Upgrade complete:` の行のあと、注記があるときだけ `  behavior changes: <n> release(s), see the report` の 1 行
- 報告を上書きしない: 同じ版・同じ日の報告(`docs/reports/upgrade-<version>-<date>.md`)が既にあるとき(衝突を直して同じ日にやり直したときなど)は、`upgrade-<version>-<date>-2.md`、`-3.md` と次の空いた名前に書く。やり直しの報告は範囲が「同じ版 → 同じ版」で注記を持たないので、上書きすると最初の報告の注記が消えるため(計画助言の指摘 1)
- /release の手順(`.claude/skills/release/SKILL.md` と `.agents/skills/release/SKILL.md`): 版を決めた(step 4)あと、tag を打つ(step 5)前に「注記の段」を足す。前の tag からの `templates/`(`templates/base/docs/` を除く。両 CLI の skill・rule・agent の定義、`.codex/agents/`、`AGENTS.override.md`、managed block の元、hook、script、言語 pack を含む)と `internal/org/prompts/` の変更を見せ、`internal/upgrade/notes/<計算した版>.md` があるかと、まだ release されていない別の版の注記のファイルが無いかを確かめる。変更があるのに注記が無ければ、人に「書く」か「挙動の変わる点は無い」を選んでもらう。「書く」なら /release をそこで終え、注記を PR で main に入れてから /release をやり直す(/release は Edit・Write を持たず、main が clean で origin と一致していることを前提にするため、注記をその場で書かない)。やり直しの step 3 の変更の一覧に注記の PR が見えることも書く
- 注記のファイルの形を確かめるテスト(名前が semver、本文が空でない、見出しを含まない など)
- README の `ralph upgrade` の節

## Non-goals

- /release そのものを実行すること(release は人が切る)
- 過去の release(8.1.0 以前)の注記を遡って書くこと。一部だけ書くと「それしか変わっていない」と読まれるおそれがあるため。最初の注記は次の release の /release で書く
- `ralph upgrade --dry-run` に注記を出すこと
- PR ごとに注記を書かせる仕組み(未リリースの注記を貯める形)。release の時に、前の tag からの変更を見てまとめて書く
- 注記の中身を CI で機械的に確かめること(形だけを確かめる)

## Assumptions

- `golang.org/x/mod` は #313 で go.mod に入っている
- 上げる前の版は `runUpgradeV2` の `fromVersion`(#313 で足した)から取れる。旧レイアウトの移行のあとも移行前の版
- 上げた後の版は CLI の `Version`。release の binary は goreleaser の `{{.Version}}`(先頭に `v` なし)
- /release は tag を打つ前に main が clean で origin と一致していることを確かめる。注記のファイルは PR で main に入ってから tag を打つ

## Affected areas

- `internal/upgrade/notes/`(新規のディレクトリ。最初は `README.md` か `.gitkeep` だけ。go:embed が空のパターンで失敗しないようにする)
- `internal/upgrade/behavior_notes.go`(新規)と `internal/upgrade/behavior_notes_test.go`(新規)
- `internal/upgrade/report.go`、`internal/upgrade/report_test.go`
- `internal/cli/upgrade_v2.go`、`internal/cli/upgrade_v2_test.go`
- `.claude/skills/release/SKILL.md`、`.agents/skills/release/SKILL.md`
- `README.md`

## Visual review

- Page: `.harness/state/plan-visual/upgrade-behavior-notes.html`(やさしい図 2 枚、詳しい図 2 枚)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: full、overview、easy-release、detail-overview、detail-range の 5 枚を撮って読み、重なりや切れが無いことを確かめた

## Design decisions

- D1(critical fork、consult を通して自律で選んだ。consult-plan-upgradenotes も案 A に同意): 注記を書く時と置き方。案 A: release の時に、/release の手順の中で、前の tag からの変更を見て `internal/upgrade/notes/<X.Y.Z>.md` を書き、PR で入れてから tag を打つ。案 B: 挙動を変える PR がそのつど `notes/unreleased.md` に足し、/release が版の名前に変える PR を出す。A を選ぶ。B は PR の作り手が版を知らないので未リリースの置き場が要り、/release も結局名前を変える PR が要る。さらに、雛形に配られる /pr の skill には repo 専用の注記の手順を書けない(下流には `internal/upgrade/notes/` が無い)。A は /release の既存の「変更の一覧を見せる」段(step 3)の続きで書け、release を切る人がまとめて確かめられる
- D2(自律で選んだ): 注記は 1 release 1 ファイル、名前は `<X.Y.Z>.md`(先頭に `v` なし、prerelease も可)。本文は箇条書きの Markdown で、見出し(`#` で始まる行)を含めない(報告の側が `### <X.Y.Z>` を付ける)
- D3(自律で選んだ): 選ぶ範囲は from < v ≤ to。from か to が semver として読めないときは並べない(dev の binary や、版の記録の無い manifest)。from ≥ to(下げる、同じ)のときも並べない
- D4(自律で選んだ): 埋め込みは `internal/upgrade` の `embed.FS`。`notes/` に注記が 1 つも無い間も build が通るよう、`README.md`(注記の書き方)を置き、`*.md` のうち名前が semver のものだけを注記として読む
- D6(自律で選んだ、計画助言の指摘 1 で足した): 報告のファイルは上書きしない。既にある名前なら `-2`、`-3` … と次の空いた名前に書き、stdout の `report:` の行もその名前を出す。上書きをやめる代わりに前の報告を移す・中身を引き継ぐ案は、どちらの報告が正かを読み手が判断しにくいので採らない
- D7(自律で選んだ、計画助言の指摘 2 で足した): /release の注記の段が見る範囲は、`templates/` から `templates/base/docs/` を除いたものと `internal/org/prompts/`。docs だけの変更では止めず、挙動を決めるファイル(skill・rule・agent の定義、hook、script、managed block の元、言語 pack)の変更では止める
- D5(自律で選んだ): 注記の形が違う(名前が semver でない `.md` が README 以外にある、本文が空、見出しを含む)と、テストで落とす。実行時は読めない注記を飛ばし、報告の Notes に 1 行残す

## Acceptance criteria

- [ ] AC1: `internal/upgrade/notes/` に `8.2.0.md` と `8.3.0.md` があるとき、8.1.0 から 8.3.0 に上げた報告に `## Behavior changes` の節があり、`### 8.2.0`、`### 8.3.0` の順に本文が並ぶ。8.2.0 から 8.3.0 なら 8.3.0 だけ
- [ ] AC2: 上げる前か後の版が semver として読めない(`dev`、空)とき、from ≥ to のとき、範囲に注記が無いときは、節を出さない
- [ ] AC3: 節は `Version:` の行の次、`## Summary` の前に出る。注記が無いときの報告はこの変更の前とバイト単位で同じ
- [ ] AC4: 注記があるとき、`ralph upgrade` の stdout の `Upgrade complete:` の行のあとに `behavior changes` の 1 行が出る。無いときは出ない
- [ ] AC5: 旧レイアウトの移行のあとに上げたときも、移行前の版から範囲を求める
- [ ] AC6: 埋め込みの注記の形を確かめるテストがあり、名前が semver でない `.md`(README を除く)、空の本文、見出しを含む本文で落ちる
- [ ] AC7: /release の skill(2 写し)に、版を決めたあと tag の前の「注記の段」があり、次が書かれている: 見る変更の範囲(雛形の skill・rule・agent と role prompt)、`notes/<計算した版>.md` の有無と、release されていない別の版の注記が無いことの確かめ、注記が要るときは /release を終えて注記を PR で入れてからやり直すこと、挙動の変わる点が無いと判断した release では注記を作らずその判断を報告に書くこと。/release は disable-model-invocation のままで、allowed-tools も変えない
- [ ] AC8: README の `ralph upgrade` の節が注記の節を説明している
- [ ] AC9: 8.1.0 から 8.3.0 に上げて報告が書かれたあと、同じ日に(衝突を直したなどで)もう一度 `ralph upgrade` が報告を書くとき、最初の報告は上書きされず注記が残り、2 回目の報告は `upgrade-8.3.0-<date>-2.md` に書かれる。stdout の `report:` の行も実際に書いた名前を出す
- [ ] AC10: /release の注記の段が見る範囲は `templates/`(`templates/base/docs/` を除く)と `internal/org/prompts/` で、たとえば `templates/base/.codex/agents/implementer.toml` だけを変えた release でも、注記を書くか「挙動の変わる点は無い」の判断を求めることが skill に書かれている

## Implementation outline

1. 注記の読み込みと選び方: `internal/upgrade/notes/README.md`、`internal/upgrade/behavior_notes.go`(埋め込み、名前の semver の確かめ、範囲の選び方)と単体テスト(AC1〜AC3 の選び方、AC6)。テストでは埋め込みの FS を差し替えられるようにする — verify: `test -z "$(gofmt -l internal/upgrade)" && go vet ./internal/upgrade/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/upgrade/`
2. 報告と stdout: `UpgradeReportData` に注記を足し、`RenderUpgradeReport` が節を出す。`runUpgradeV2` が範囲の注記を渡し、stdout に 1 行。既にある報告を上書きせず次の空いた名前に書く(`UpgradeReportRelPath` と `WriteUpgradeReport` の周り)。テスト(AC1〜AC5、AC9) — verify: `test -z "$(gofmt -l internal/upgrade internal/cli)" && go vet ./internal/upgrade/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/upgrade/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Upgrade|Migrat'`
3. /release の skill と README(AC7、AC8、AC10) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC8。issue #317 の受け入れ条件 2 つ(注記のある版をまたぐと報告の先頭に並ぶ、注記の置き場と書く手順が /release の手順書にある)が AC1・AC7 に入っていること
- Documentation drift to check: README、/release の 2 写し、`docs/specs/2026-08-17-overlay-scaffold-v2.md` の upgrade の報告の記述、AGENTS.md の `internal/upgrade/` の行
- Evidence to capture: `run-static-verify.sh` と `go test` の結果

## Test plan

- Unit tests: 範囲の選び方(古い順、境界の from と to、prerelease、`v` の有無、dev、空、from ≥ to)、名前の確かめ、本文の確かめ、`RenderUpgradeReport` の節の位置と、注記が無いときのバイト一致
- Integration tests: `runUpgradeV2` の既存のテストに、差し替えた注記の FS で 1.0.0-test → 1.1.0-test などに上げたときの節と stdout の 1 行を足す。旧レイアウトの移行(`TestRunMigrateLegacy_HappyPath_Yes` の 0.9.0-test → 2.0.0-test)でも範囲が移行前の版から決まること
- Regression tests: 埋め込みの本物の `notes/` が形の確かめを通ること。`go test ./internal/upgrade/ ./internal/cli/` の全体
- Edge cases: 注記が 0 個、同じ版の注記の重複(名前が同じなので起きない)、範囲の上端だけ、prerelease の to(`8.2.0-rc.1` には `8.2.0` の注記を含めない)、同じ日の 2 回目・3 回目の報告(`-2`、`-3`)、既にある名前が通常のファイルでない(symlink、ディレクトリ)とき
- Evidence to capture: テストの結果

## Risks and mitigations

- /release で注記を書き忘れる → /release の段で、前の tag からの `templates/`(`templates/base/docs/` を除く)と role prompt の変更があるのに注記のファイルが無ければ止まり、「書く」か「挙動の変わる点は無い」と判断して書き残すかを求める。`templates/base/docs/` を除くのは、docs の同期で毎回変わり、毎回止まるため(consult の指摘)。それ以外を広く見るのは、codex の agent 定義や hook だけの変更でも手順が変わるため(計画助言の指摘 2)。docs 以外の小さな変更でも止まるが、そのときは「挙動の変わる点は無い」と書けば進める(知らせる側)
- やり直しで版の選択が変わる(patch から minor など)と、注記のファイル名と計算した版がずれる → 注記の段で `notes/<計算した版>.md` の有無と、release されていない別の版の注記が無いことを確かめる
- /release はその場で注記を書けない(allowed-tools に Edit・Write が無い、main が clean で origin と一致が前提) → 注記が要るときは /release を終えて PR を通し、やり直す流れを skill に書く
- 注記の中身が誤っている・足りない → release を切る人が PR で確かめる。報告には「この release の挙動の注記」として出すだけで、網羅を約束しない文にする
- 下流の repo が古い binary で上げると注記は出ない → 注記は上げた後の CLI の binary に入っているので、上げに使う binary が新しければ出る。古い版から新しい版に上げるときは、新しい binary を使うのが前提(今の upgrade と同じ)
- テストが本物の埋め込みの `notes/` に頼ると、注記を足すたびに期待が変わる → 選び方と報告のテストは差し替えた FS で行い、本物の `notes/` は形の確かめだけ

## Rollout or rollback notes

- 注記が 1 つも無い間は、報告と stdout は今と同じ。最初の注記は次の release の /release で入る。AC1 の実地の証拠(本物の注記のある版をまたいだ upgrade の報告)は、その release のあとで取る
- 戻すときは PR を revert する

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — consult(consult-plan-upgradenotes): 進めてよい、案 A に同意。指摘 3 点(/release は Edit を持たないので「終えて PR を通し、やり直す」、版とファイル名の一致の確かめ、見る範囲の絞り方)を Scope、AC7、Risks に入れた。plan advisory: ran with codex。2 件 [MEDIUM]。(1) 同じ日のやり直しで報告が「同じ版 → 同じ版」で上書きされ、最初の注記が消える → D6、AC9、slice 2 を足し、既にある報告は上書きせず `-2` などに書く。(2) /release の見る範囲から codex の agent 定義・`AGENTS.override.md`・managed block・hook・script が漏れる → D7、AC10 を足し、`templates/`(`templates/base/docs/` を除く)と `internal/org/prompts/` を見る
- [x] Plan approved — 自動承認(ユーザーの指示「推奨案を自動承認」、ralph-d1 経由の中継、2026-11-13)
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
