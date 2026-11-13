# Verify report: upgrade-behavior-notes

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-upgrade-behavior-notes.md`(issue #317、承認 digest `0de679dfe2a1`)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン cycle 1(`.harness/state/standard-pipeline/cycle-count.json` が cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `0cce026b`。origin/main `4220ea7c` は `f55823a2` で取り込み済み)。15 ファイル、1546 行追加、20 行削除。見たのは受け入れ条件への適合、静的解析、文書の食い違い。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-13-upgrade-behavior-notes.log`(gitignore の対象。`run-static-verify.sh` の全出力、/release の段 4b の dry run、`git diff --check`)。スクリプト自身が書いた log は `docs/evidence/verify-2026-11-12-173345.log`(UTC の日付)
- Insight event: `docs/insights/events/2026-11-12-upgrade-behavior-notes.jsonl`(ファイル名は UTC の日付)に `phase=verify`、`cycle=1`、`verdict=pass`、`low=2` の 1 行を `insights-append.sh --cycle auto --low 2` で足した

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 8.2.0 と 8.3.0 の注記があるとき、8.1.0 → 8.3.0 の報告に `## Behavior changes` があり `### 8.2.0`、`### 8.3.0` の順。8.2.0 → 8.3.0 なら 8.3.0 だけ | Met(実物の注記での証拠は次の release のあと) | 選び方は `internal/upgrade/behavior_notes.go:66-105`(範囲 `:68` と `:87`、版の順の並べ替え `:92-94`)。描画は `internal/upgrade/report.go:79` と `:100-111`、呼び手は `internal/cli/upgrade_v2.go:188` と `:195`。テスト: `TestBehaviorNotesBetween_SelectsTheReleasesAfterFromUpToTo` の `two releases`(8.1.0 → 8.3.0 で 8.2.0、8.3.0)と `only the upper end`(8.2.0 → 8.3.0 で 8.3.0)、`TestBehaviorNotesBetween_OrdersByVersionNotByName`、`TestRenderUpgradeReport_BehaviorNotes_SectionAfterVersionLine`、通しの `TestRunUpgradeV2_BehaviorNotes_InRangeNotesInReportAndStdout`(1.0.0 → 1.1.0 で 1.0.5、1.1.0)。HEAD の `internal/upgrade/notes/` は `README.md` だけなので、実物の注記をまたぐ証拠は計画の Rollout のとおり次の release のあとに取る |
| AC2: from か to が semver でない、from ≥ to、範囲に注記が無いときは節を出さない | Met | `behavior_notes.go:67-70`(読めない版と from ≥ to で早く返る)。テスト: 上の表のテストの `same version`、`downgrade`、`from is dev`、`from is empty`、`to is dev`、`to is empty`、`no note in the range`、通しの `TestRunUpgradeV2_BehaviorNotes_NoneInRange_NoSectionNoLine` |
| AC3: 節は `Version:` の行の次、`## Summary` の前。注記が無ければ変更前とバイト単位で同じ | Met | `report.go:75-79` で `Version:` の行のすぐ後、`renderSummaryTable` の前に呼ぶ。`:101-103` は注記が 0 個なら何も書かない。`report.go` の描画の経路への差分はこの呼び出し 1 行だけで、ほかは doc comment と名前の関数の追加。テスト: `TestRenderUpgradeReport_BehaviorNotes_SectionAfterVersionLine`、`TestRenderUpgradeReport_BehaviorNotes_OnlyAddTheSection`(節を除くと注記なしの報告と等しい)、`TestRenderUpgradeReport_NoBehaviorNotes_ByteIdentical`(nil と空の slice) |
| AC4: 注記があるとき stdout の `Upgrade complete:` の後に `behavior changes` の 1 行。無いときは出ない | Met | `upgrade_v2.go:246-249`、`:266-273`。文言は `  behavior changes: 1 release, see the report` と `<n> releases`。計画の Scope の `release(s)` からの変更は Progress の S2 の行に記録があり、README も実装と同じ文言。テスト: InRange(2 releases)、`TestRunUpgradeV2_BehaviorNotes_UnusableNoteSkippedWithANote`(1 release)、NoneInRange(出ない)、`TestRunUpgradeV2_BehaviorNotes_DryRunAndNoOpPrintNone`(dry run と no-op で出ない) |
| AC5: 旧レイアウトの移行のあとも移行前の版から範囲を求める | Met | `internal/cli/migrate.go:993` が `runUpgradeV2` に `oldManifest.Meta.Version`(移行前の版)を渡し、`upgrade_v2.go:188` はその `fromVersion` で選ぶ。通常の経路は `internal/cli/upgrade.go:157`。テスト: `TestRunMigrateLegacy_HappyPath_Yes`(`internal/cli/migrate_test.go:1146-1155` で 0.8.0、1.5.0、2.0.0 の注記を置き、`:1308-1321` で 0.9.0-test → 2.0.0-test の報告に 1.5.0 だけがあることと stdout の 1 行を見る) |
| AC6: 埋め込みの注記の形のテストがあり、semver でない名前(README を除く)、空の本文、見出しで落ちる | Met | `ValidateBehaviorNotes`(`behavior_notes.go:115-135`)、`behaviorNoteVersion`(`:142-151`)、`readBehaviorNote`(`:156-174`)。テスト: `TestValidateBehaviorNotes_EmbeddedNotesPass`(本物の埋め込みのディレクトリを読み、README があることも確かめる)、`TestValidateBehaviorNotes_RefusesBadNotes`(`changelog.md`、`v8.2.0.md`、`8.2.md`、`8.2.0+abc.md`、`08.2.0.md`、空、空白だけ、見出し 3 種、ディレクトリ)、`TestValidateBehaviorNotes_AcceptsWellFormedNotes`(見出しを持つ README は通る)、`TestValidateBehaviorNotes_ReportsEveryBadNote` |
| AC7: /release の 2 写しに、版を決めた後で tag の前の注記の段があり、範囲、注記の有無と別の版の注記が無いことの確かめ、要るときは終えて PR で入れてからやり直すこと、挙動の変わる点が無いときは報告に書くことがある。disable-model-invocation と allowed-tools は変えない | Met | `.claude/skills/release/SKILL.md:39-46`(段 4b。step 4 の後、step 5 の tag の前)、`:62`(step 9 の報告)、`:68`(Completion gate)。`.agents` の写しは `:37`、`:60`、`:66` で本文が同じ(`check-skill-sync.sh` が ok)。frontmatter の `:4-5` は差分に出ない(`name`、`description`、`disable-model-invocation`、`allowed-tools` の変更行を grep して 0 件)。allowed-tools の範囲で進められるかは下の Observational checks |
| AC8: README の `ralph upgrade` の節が注記の節を説明している | Met | `README.md:165-167`(範囲と順、出ない条件、stdout の 1 行、同じ日の 2 回目の名前) |
| AC9: 同じ日にもう一度報告を書くとき最初の報告は上書きされず、2 回目は `-2.md`。stdout の `report:` の行も実際の名前 | Met | `report.go:292-308`(`NextUpgradeReportRelPath`)、`upgrade_v2.go:218-224`、`:248`、`:253`(drift のエラーも書いた名前を出す)。テスト: `TestRunUpgradeV2_SameDayRerun_KeepsEarlierReports`(3 回の実行。1 回目は注記を持ち、2 回目は `-2.md`、3 回目は `-3.md`、前の報告は変わらない)と `TestNextUpgradeReportRelPath_*` の 5 つ(ディレクトリと symlink も使用中、上限の両側、symlink の `docs/reports` を拒む、名前の削り)。テストの版は 1.0.0 → 1.1.0 で、AC の 8.1.0 → 8.3.0 と同じ形 |
| AC10: 段 4b が見る範囲は `templates/`(`templates/base/docs/` を除く)と `internal/org/prompts/`。`templates/base/.codex/agents/implementer.toml` だけの release でも判断を求める | Met | `SKILL.md:40` に pathspec と implementer.toml の例がある。`git ls-files` での確かめ(下の Observational checks)で、implementer.toml、`AGENTS.override.md`、managed block の元、hook、script、agent、rule、`internal/org/prompts/leader.md` が pathspec に入り、`templates/base/docs/quality/definition-of-done.md` は入らない |

issue #317 の受け入れ条件 2 つ(`gh issue view 317` で読んだ。OPEN)との対応:

- 「注記のある版をまたいで `ralph upgrade` を流すと、報告の先頭にその注記が並ぶ」は AC1(並び)と AC3(`Version:` の行の次という位置)に入っている
- 「注記の置き場所と書く手順が /release の手順書に書かれている」は AC7 に入っている。置き場所は `SKILL.md:39` の `internal/upgrade/notes/<X.Y.Z>.md` と書き方の `internal/upgrade/notes/README.md`、書く手順は `:43-45`

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(既定の changed scope) | exit 0 | shellcheck、hook の `sh -n` 20 本、guard の awk、`jq -e`、`check-sync.sh`(DRIFTED 0、IDENTICAL 172)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照、golang(gofmt ok、`go vet ./...` は出力なし、golangci-lint `0 issues`、staticcheck は出力なし)、branch の secret scan(`4220ea7c..0cce026b` clean)。すべて通った |
| `git diff --check origin/main...HEAD` | exit 0 | 空白の誤りなし |
| `/release` の frontmatter の変更行の grep | 0 件 | 2 写しとも |

手元の Go は 1.27.3 で go.mod と同じ版。

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の `ralph upgrade` の節 | Yes | `README.md:165-167` が実装と一致する。範囲(`<from>` < version ≤ `<to>`、古い順)、出ない条件(範囲に注記なし、`dev`、不明な `<from>`)、stdout の文言(`1 release`)、`-2.md` と `-3.md`、`report:` の行が書いた名前を出すこと |
| /release の 2 写し | Yes | `check-skill-sync.sh` が ok。本文は同じで、`.agents` の写しに allowed-tools が無いのは origin/main と同じ |
| `internal/upgrade/notes/README.md` | Yes | 名前の規則、見出しの禁止、`ValidateBehaviorNotes` とテストの説明がコードと一致する |
| `docs/specs/2026-08-17-overlay-scaffold-v2.md` の FR-1 (7) | Partial(V2、LOW) | 報告の名前を `docs/reports/upgrade-<version>-<date>.md` とだけ書いている(`:59`)。同じ日の 2 回目以降が `-2.md`、`-3.md` になることは書かれていない。報告の中身は spec に書かれていない(#313 の `Version:` の行も無い)ので、`## Behavior changes` の節が無いのは前例どおりで食い違いにしない。/sync-docs で FR-1 (7) の括弧に「同じ日の 2 回目以降は `-2.md` …(#317)」を足すかを決める |
| AGENTS.md の `internal/upgrade/` の行 | Partial(V1、LOW) | `upgrade report writer (docs/reports/upgrade-*.md)` の glob は `-2.md` も含むので名前は合っている。ただ、release のたびに人が書く `internal/upgrade/notes/<X.Y.Z>.md`(埋め込みの注記で、報告の `## Behavior changes` に出る)が map に無い。/sync-docs で 1 句足すかを決める |
| CLAUDE.md の `/release` の行 | Yes | 手動起動だけで、変更なし |
| `ralph upgrade --help` | Yes | Long の文は報告の名前も節も書いていない(`internal/cli/upgrade.go:31-45`) |
| `/org` skill の 4 写しの `docs/reports/upgrade-*.md` | Yes | glob が `-2.md` を含む |
| `docs/specs/2026-08-01-org-runtime.md:87` | Yes | 「binary を更新してから `ralph upgrade`」の 2 段の配布は変わらない |
| `docs/architecture/repo-map.md` | 対象外 | `internal/upgrade/` の項目が無い |
| 計画の受け入れ条件のチェックボックス | 遅れ(INFO) | すべて `[ ]`。この repo では PR の前に付ける慣習 |
| `docs/tech-debt/README.md` | 変更不要 | upgrade、報告、release、scaffold の版の知らせに触れる行を python で読んで、4 列目の Trigger を見た。`WriteUpgradeReport` の権限の行(Trigger は「列挙した呼び手への次の変更」)は、関数の本体も呼び出しも変わらず、増えたのは comment での言及だけ。`checkManifestVersion` の行と `orgScaffoldRoot`、`printScaffoldVersionNotice` の行は、どれも対象の関数が変わっていない(`scaffold_version_notice.go` の変更は `semverWithV` を `upgrade.SemverWithV` に替えただけ)。`runUpgradeIO` の引数の行は、引数が増えていない |

## Observational checks

/release の段 4b の git の命令を、最新の tag `v8.1.0`(origin にもある)に対して読み取りだけで流した。tag は作っていないし、/release も走らせていない。

- `git diff --stat v8.1.0..HEAD -- templates/ internal/org/prompts/ ':(exclude)templates/base/docs/'` は exit 0 で 10 ファイル。exclude なしは 14 ファイル、`templates/base/docs/` だけは 4 ファイルで、exclude ありの一覧に `templates/base/docs/` は 0 件。exclude の pathspec は効いている
- `git ls-tree --name-only HEAD internal/upgrade/notes/` は exit 0 で `internal/upgrade/notes/README.md` だけ。出力は repo からの path なので、`X.Y.Z.md` の有無は末尾の名前で読む
- いま /release を走らせると、差分が空でなく 8.x の注記も無いので、段 4b は `write a note` と `no behavior change` を聞くことになる。計画どおりの止まり方
- pre-check 4 で HEAD が origin/main と同じなので、「HEAD に注記がある」は「main に注記がある」と同じ
- 段 4b が使う道具は Bash(`git diff --stat`、`git ls-tree`)、Read(notes の README)、AskUserQuestion だけで、Edit と Write は要らない。注記が要るときは /release を終えて PR で入れ、やり直す流れが `:44` にある

pathspec に入るかの確かめ(`git ls-files -- templates/ internal/org/prompts/ ':(exclude)templates/base/docs/'` に完全一致で grep):

- 入る: `templates/base/.codex/agents/implementer.toml`、`templates/base/.codex/AGENTS.override.md`、`templates/base/.ralph/core/AGENTS.core.md`、`templates/base/.claude/hooks/pre_bash_guard.sh`、`templates/base/scripts/run-verify.sh`、`templates/base/.claude/agents/verifier.md`、`templates/base/.claude/rules/ralph/testing.md`、`internal/org/prompts/leader.md`
- 入らない: `templates/base/docs/quality/definition-of-done.md`
- 言語 pack は `templates/packs/<lang>/`(例 `templates/packs/_template/verify.sh`)にあり、`templates/` に入る

## Coverage gaps

- テストは走らせていない(`/test` の担当)。上の表のテスト名は読んで AC に当てたもので、通るかは未確認
- AC1 の実物の証拠(本物の注記がある版をまたいだ upgrade の報告)は、最初の注記が入る次の release のあとに取る(計画の Rollout)
- 段 4b の分かれ目(AskUserQuestion の 2 択、注記の PR のあとのやり直し、別の版の注記が残っているときの停止)は流していない。確かめたのは 2 つの git の命令だけ。「最新の tag より上の版の注記」の比べ方は人が読むか Bash の `sort -V` で行うことになり、スクリプトは無い
- self-review の報告は `ae1724a8` の時点のままで、N1 と N2 を未解決と書いている。`0cce026b` で N1 は計画の Progress の行(`:131`)で閉じ、N2 の `"-100"` の literal は `strconv.Itoa(upgradeReportNameLimit)`(`internal/upgrade/report_test.go:465`)になった。同じテストの doc comment(`:446`)は「-99 with the limit of 100」「-100」と例の数を書いたままで、N2 の comment の半分は残っている。動作には関係しない
- 800 行の目安: `internal/cli/upgrade_v2.go` は 959 → 993 行、`internal/cli/upgrade_v2_test.go` は 2026 → 2309 行。どちらも前から 800 を超えていて、register の行は無い。self-review は指摘していない
- 足すと確かさが最も上がる確認: 段 4b の 2 つの確かめ(差分の有無、`notes/` にある版の一覧と最新の tag の比較)を `/release` から呼べる小さなスクリプトにすれば、版の比べ方を人の読みに頼らなくなる。ただし `/release` は repo の保守者だけが使い、計画は注記の中身の機械的な確かめを Non-goal にしているので、急ぐものではない

## Verdict

- Verified: AC1 から AC10 の実装とテストの対応(file:line とテスト名)、`run-static-verify.sh` の全項目、`git diff --check`、/release の frontmatter が変わっていないこと、段 4b の git の命令が動くことと pathspec の範囲、issue #317 の受け入れ条件 2 つが AC1、AC3、AC7 に入っていること
- Partially verified: AC1 の実物の注記での証拠(次の release のあと)、段 4b の対話の分かれ目
- Not verified: テストの実行結果(`/test`)、release の binary での `ralph upgrade` の実走
- Verdict: pass。文書の食い違いは LOW 2 件(V1: AGENTS.md の `internal/upgrade/` の行に注記のディレクトリが無い。V2: overlay-scaffold-v2 の spec の FR-1 (7) に `-2.md` の名前が無い)で、どちらも /sync-docs で直すかを決める
