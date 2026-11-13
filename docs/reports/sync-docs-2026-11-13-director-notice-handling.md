# sync-docs report: director-notice-handling

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(承認の digest `debc4ca9fe00`)。issue #318
- Branch: `docs/director-notice-handling`。base は origin/main `e902705c`(#327)で、`f65575ad` で取り込み済み。run 開始時の HEAD は `f65575ad`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `450535e2`(再確認は `40b68cc9`、`7b5569ac`、`121e6f34`。Merge: yes。未解決の LOW は L3 と N6)、verify `fcd368b0`(pass。LOW 3 件)、test `4618677e`(pass。Test gaps G1〜G4)
- 見た範囲: `git diff e902705c HEAD`(14 ファイル。取り込み済みの main との差なので、この branch の変更だけ)。Go のコード、テスト、skill、prompt は変えていない。この run が足したのは文書だけ
- 制約: 計画の本文は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `debc4ca9fe00`、`git diff --numstat` は 7 行追加、3 行削除(3 つの箱を `[x]` にして path を付け、子の箇条を 4 つ足した)。AC の箱は `[ ]` のまま
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): `git diff e902705c HEAD` は 14 ファイル、+1,165/-6(計画、report 3 つ、insights を含む)。それらの記録 5 ファイルを除くと 9 ファイルで +736/-6(skill の 4 写しが 4 × +58/-1、テスト 2 ファイルが +468、`leader.md` が +28/-1、`director.md` が +6、tech-debt が +2/-1)。walkthrough は書いていない
- report の型: sync-docs の report に template は無い(`docs/reports/templates/` は review、test、verify、cross-review、walkthrough だけ)ので、直近の `sync-docs-2026-11-13-upgrade-behavior-notes.md` の節立てに合わせた

## Summary

verify の LOW 3 件をすべて直した。tech-debt の 188 行 (a) と 365 行に 2026-11-13 の持ち越しの注を足し、multi-org director の spec の FR-9 に #318 の日付つきの項を足した。test の G1 と G2(= self-review の L3)は tech-debt の新しい 1 行(495 行目)にした。N6 は行にしていない(理由は下)。verify が INFO にした README の director の段落に 1 文を足した。Progress に verify、test、main の取り込み、sync-docs の記録を足した。main の取り込み(`f65575ad`)は `/test` のあとなので、取り込んだ木で check-sync、check-skill-sync、新しいテストを回し直した。ほかの文書は実装と合っていた。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 188 行 (a) の Trigger セルの末尾に 2026-11-13 の持ち越しの注、Related の末尾に計画。365 行も同じ形で、Related には `internal/org/prompts_notice_test.go` も足した。末尾に 1 行(495 行目)。`difflib` で、既存の 2 行は挿入だけ(置き換えも削除も 0)、ほかの既存の行は変わっていないことを確かめた。#313 の行(464、465)は触れていない |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-9 の項、2026-11-01(#271)の次(FR-10 の前)に、`2026-11-13 に決めたこと(issue #318)` を 1 項足した。1 行の挿入 |
| `README.md` | 285 行目の director の段落の、merge の文の次に 1 文。verb が出す `note:` の行(雛形が古い、codex の hook が未承認)を director が人に伝えること、`ralph upgrade` を打たず、codex の設定も書き換えず、hook も承認しないこと、`ralph org spawn` の行は leader が QUESTION で上げること |
| `docs/plans/active/2026-11-13-director-notice-handling.md` | Progress の節だけ。上の「制約」のとおり |
| `docs/reports/sync-docs-2026-11-13-director-notice-handling.md` | この report |
| `docs/insights/events/2026-11-13-director-notice-handling.jsonl` | `sync_docs` の行(cycle 1、pass) |

commit は 1 つ(`docs: sync docs for director-notice-handling`)。report が自分の commit の SHA を書けないので、内容と report は分けていない。

## verify が挙げた drift

### tech-debt 188 行 (a)

Trigger は「`RolePromptVars` か leader の prompt の次の変更」。`leader.md` が増えたので当たっている。足した注の中身は次のとおり。

- 変わった所: `運用規律` に、`ralph org spawn` が出す codex の hook の `note:` の行の項(25 行)。`件を上げる` の「QUESTION を上げたら返事まで作業を止める」の文に、この QUESTION は例外だと言う一句(2 行)
- 行数: 1,332 から 1,359(`wc -l` を origin/main の `e902705c` と、この branch が `leader.md` を最後に変えた `e95aa15f` で。`git log -1` で最後の commit を引いた)
- 払っていない理由: `RolePromptVars` と `{{SCOPE}}` の文は前のままで、`--reserve` だけの leader は今も「指定なし」の文を読む。計画の Scope は leader と director の `note:` の行の扱い(D1〜D5)。(b) は当たっていない(`EnvelopeSummary` と `checkOrgEnvelope` は変わっていない)

### tech-debt 365 行

Trigger は「`Send` か leader の template のテストの次の変更」。leader の template のテスト `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` を足したので当たっている。テストは新しい `internal/org/prompts_notice_test.go`(339 行。director の skill と prompt のテストも入っている)にあり、数えている 6 ファイルは増えていない。`wc -l` は `f65575ad` で、`verbs.go` 2,597、`verbs_test.go` 5,893、`prompts_test.go` 2,722、`spawn.go` 3,101、`spawn_test.go` 6,408、`director_skill_sync_test.go` 1,307。`git diff --stat e902705c HEAD` にこの 6 つは出ない。注には、数が `e902705c` と同じであること、払っていない理由(move-only の split は Scope の外)を書いた。

### spec の #318 の項

verify は 151 行目と書いたが、main の取り込みで 1 行ずれて、`director は実装しない。許すのは…` は今 152 行目にある。足した項は、この spec の前例(#270、#271 の項と同じ `YYYY-MM-DD に決めたこと(issue #N):` の形、末尾は「詳細は計画(`<slug>.md`)に書いた」)に合わせた。書いた中身は次のとおり。

- director は `ralph upgrade` を打たない。雛形の版の知らせを見たら人に伝える(行の 3 つの値を入れ、流すかどうかと流し方は人が決めると添える)。FR-9 の「ralph と gh の操作に限る」からこの動詞を除いた
- codex の設定を書き換えず、hook を承認しない。hook の知らせを見たら人に伝え、承認するかは人が決める。`ralph org spawn` の行は leader が QUESTION で上げ、その QUESTION は座席を止めない
- 変えた面(skill の 4 写し、`director.md`、`leader.md`)と、変えなかった面(Go のコード、行の文面、`--help`、`/org` skill)

## tech-debt

### 既存の行の Trigger

自分で `docs/tech-debt/README.md` の全行の Trigger の列(取り消し線の中を除いて)を、この branch が触れた名前で引き直した(`scratchpad` の `trigger_sweep.py`。語は `director skill`、`director/SKILL`、`director.md`、`leader.md`、`leader prompt`、`leader template`、`director_skill_sync`、`prompts_test`、`RolePromptVars`、`role prompt`、`README`、`multi-org-director`、`org-runtime.md` など)。当たった行の Trigger を読んだ。

| 行 | 判断 |
|----|------|
| 188 (a) | 当たった。注を足した |
| 365 | 当たった。注を足した |
| 200 (g)、239 (o)(p)、426 | 当たらない。条件は機能ごとの org の節、手順 9 と 10、「節を足す」で、この branch が `leader.md` で触れたのは `件を上げる` の 1 文と `運用規律` の末尾の 1 項 |
| 384 | 当たらない。条件の「座席を止める規則」(ミッションの 6 と運用規律の最初の項)は変わっていない。足した項は、止めるかを人の DECISION に任せる別の項 |
| 262、269 (b)、380、381、390、421、422 | 当たらない。director の skill の受信箱の節、始める 3、送ってから確かめる手順、wait の終了コード、`director_skill_sync_test.go` のどれにも、この branch は触れていない(skill の hunk は「昇格」の 2、出力先の 3、「進捗の報告」の小節、「してはいけないこと」の 4 つだけ。新しいテストは新しい 2 ファイル) |
| 372、427、459、471、472 | 当たらない。implementer と reviewer の prompt、`### cross-review`、resume の項、gate accept の項は変わっていない |
| 177、195、241〜252、256〜259、263、270、312、332〜338、349、350、367、377、411、416、420、432、435、451 | 当たらない。`/director` skill や `leader.md` の名は出るが、条件は別のコード、別の節 |
| 88、466、467、468、480、482 | 当たらない。`scaffoldVersionNotice`、`codexHookTrustNotice`、`orgScaffoldRoot` を含む本番の Go は変わっていない |
| README の名を条件に持つ 245、249、250、254、371、401、468 | 当たらない。足した README の文は director の段落にあり、`[org].state_dir` の段落、merge-order の follow-up の文、`ralph org start` の help の行、recipe のどれでもない |
| 800 行の行 | 次の節 |

### 新しい 1 行(495 行目)

「Test gaps of docs/director-notice-handling (#318), the test report's G1 and G2」。(a) G1: `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines` は 2 つの断片だけを本番の行に結び付ける。skill が引くその周りの英語(`older than this CLI (<CLI の版>)`、` and how to <直し方>`)は結び付かず、変異 p4、p3 が AC7 のテストと skill のテストを通った。既存の `ScaffoldVersion`(10 本)と `CodexHookTrust`(8 本)のテストは落ちるので気づかれずには入らないが、その失敗文は skill に触れない。(b) G2 と L3: hook の行は `strings.Contains` で、末尾までは見ない。小節を切る補助は `internal/org` と終わりの規則が違い、`####` の見出しを入れると 2 か所で切れ方が変わる。今は差が出ない。

Related は `docs/plans/active/2026-11-13-director-notice-handling.md`(`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` とは書いていない)、test と self-review の report、テストのファイル。構造の確かめ(`git show HEAD:` との比較、scratchpad の `tdcheck.py`):行は `| ` で始まり ` |` で終わる、列は 5、バッククォートは偶数、`file:line` の形は 0、U+FFFD は 0。plan の参照は 124 件で、無い参照は 0 件。

### 行にしなかったもの

- N6(`prompts_notice_test.go` の comment の折り返しが 1 行だけ短い): self-review の `Tech debt identified` の表は L3 だけを挙げていて、N6 は挙げていない。Recommendation も「N6 は折り返しの話なので、Known gaps に書かなくてよい」と書く。払う中身が comment の折り返しだけで、Trigger を付けられないので行にしていない。計画の Progress が「L3 と N6 は PR の Known gaps に書く」と言うので、書くかは `/pr` が決める
- G3(director と leader が実際の会話で文に従うか): INFO。単体テストでは確かめられず、行にしても払い方がない
- G4(変異 (c) で AC7 のテストが通る): INFO。役割の分け方どおり
- skill と prompt の長さ: director の skill は 4 写しとも 2,042 から 2,099 行、`leader.md` は 1,359 行で 800 行を超える。skill と prompt は code ではなく、計画の Risks が長さを認めていて(「2,020 行」)、直近の sync-docs(watch-usage-limited)も skill の写しの長さを行にしていない。365 行の注が `leader.md` を「a prompt, not code」と呼ぶ前例に合わせた
- 新しいテストの 2 ファイル(339 行、129 行)は 800 行を超えない。`wc -l` を、この branch の diff の全ファイルで origin/main と比べた(上の「差分の大きさ」の 14 ファイル)。800 行を超えるのは skill の 4 写しと `leader.md` だけで、どれも元から超えていた

### #313 の行(464、465)

RESOLVED の comment と取り消し線の行、main の持ち越しの注(#312a)と Related は、取り込みの解決のまま触れていない。comment が「Pinned by」に挙げるのは director 側の 2 つのテストで、leader の `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` と `internal/cli` の `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines` は挙げていない。書いたことは偽ではなく、行の主題(director が知らせをどう扱うか)には足りるので直していない(verify の INFO と同じ判断)。

## main の取り込み(`f65575ad`)のあとの確かめ

取り込みは `/test` のあとで、`/test` の report は取り込みの前の木(`fcd368b0`)のもの。main の #327 は director の skill(4 写しとも)、`/org` skill、agent-messaging の rule、quality-gates、tech-debt などを変えている。衝突は tech-debt の #313 の行だけで、orchestrator が解決した(この branch の取り消し線と RESOLVED の comment、main の持ち越しの注と Related を残した)。

- orchestrator が取り込んだ木で通したもの(handoff の記録): check-skill-sync、check-sync、`go vet`、`internal/org` と `internal/cli` の絞ったテスト
- この run で回し直したもの:
  - `./scripts/check-sync.sh`: `PASS: all files in sync.`。`./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`。README の編集のあとにも回した
  - skill の 4 写し: root と雛形の `cmp` が一致(`.claude`、`.agents` とも)、`.claude` と `.agents` の本文(4 行目まで除く)も一致
  - `go test -count=1 -v -run 'TestDirectorSkill_SaysWhatToDoWithTheHumanNotes|TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection|TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping' ./internal/org/`: 3 本 PASS。`go test -count=1 -v -run 'TestDirectorNoticeSkill_' ./internal/cli/`: 1 本と subtest 4(4 写し)PASS。`gofmt -l internal/org internal/cli` は 0 件
- 回していないもの: `internal/org` と `internal/cli` の全体のテスト、`./scripts/run-test.sh`、`run-static-verify.sh`(Go のコードを変えていない)。取り込みで入った main の変更(#327)のテストは、main の merge の時点で通っているものとして扱った
- verify の行番号(skill の 1443〜1489 行など)は取り込みの前の木のもの。取り込みで director の skill が増えたので、今の行番号はずれている。この report と足した文書は skill の行番号を書いていない

## 実装と合っていると確かめたもの

| Doc / contract | 結果 |
|----|----|
| `--help` の段落(`orgScaffoldVersionNoticeHelp`、`statusScaffoldVersionNoticeHelp`、`orgCodexHookTrustNoticeHelp`) | 変えない決定のまま。本番の Go を変えていない |
| `README.md` の 133、134 行(`ralph org` と `ralph status` の行の知らせの説明) | 合っている。行の形(`run 'ralph upgrade' in <dir>`)は変わらない |
| `AGENTS.md`、`CLAUDE.md`、`.claude/rules/ralph/*`、`docs/quality/*`、`docs/architecture/*` | director の skill の小節にも知らせの扱いにも触れていない。`git grep -i` で `ralph upgrade`、`older than this CLI`、`scaffold version`、`manifest.toml` を引くと、当たるのは `AGENTS.md` の 88 行目(`internal/upgrade/`)と 106 行目(`.ralph/core/`)だけで、どちらも upgrade の仕組みの説明。agent-messaging の rule は QUESTION の TYPE を決めるだけで、「上げたら返事まで作業を止める」一般の規則は rule にも spec にも無く、`leader.md` だけにある(`git grep` で `返事が来るまで`、`返事まで止め` を引いた。`/org` skill の 1288 行は別の待ちの話) |
| `docs/specs/2026-08-01-org-runtime.md` | director の知らせの扱いを書いた所は無い(verify と同じ) |
| `.claude/skills/org/SKILL.md` ほか | 計画の Non-goals どおり、足していない |
| `docs/architecture/repo-map.md`、`docs/quality/*.md` | この branch は script も hook もテストの仕組みも足していない。変える所が無い |
| `templates/`、`.codex/`、`scripts/` | skill の雛形の 2 写しは `sync-skills.sh` の出力として root と一致(`check-skill-sync.sh`) |

## 実行したもの

- `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`git diff --check`: すべて rc 0(commit の前、4 ファイルの変更を含めて)
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-13-director-notice-handling.md`: 編集の前後とも `debc4ca9fe00`
- tech-debt の構造の確かめ(`tdcheck.py`、`tdcheck2.py`)と Trigger の引き直し(`trigger_sweep.py`)。どれも scratchpad で、repo には置いていない
- 上の「main の取り込みのあとの確かめ」のテスト
- `./scripts/insights-append.sh --slug director-notice-handling --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`: `--slug` は日付なしの slug

## 残り

- AC1〜AC8 のチェックボックスは `[ ]` のまま(計画の本文は変えない約束。verify の INFO)。`/pr` の前に付ける
- PR 本文には、(1) director が知らせの行を人に伝える動きが次の release の振る舞いのノートの対象になること(`internal/upgrade/notes/<X.Y.Z>.md` は `/release` で書く)、(2) Known gaps として test の G1 と G2(= L3。tech-debt の 495 行目)と N6、(3) 動きは実際の会話で確かめていないこと(G3)を書くとよい
- skill の小節は 47 行で、計画のリスクの欄の「20 行ほど」を超える(verify の INFO)。中身は AC1 と AC2 が求める項目だけなので、直していない
