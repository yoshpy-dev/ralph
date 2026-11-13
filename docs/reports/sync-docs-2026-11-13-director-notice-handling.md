# sync-docs report: director-notice-handling

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(承認の digest `debc4ca9fe00`)。issue #318
- Branch: `docs/director-notice-handling`。base は origin/main `e902705c`(#327)で、`f65575ad` で取り込み済み。run 開始時の HEAD は `f65575ad`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)。2 回目は末尾の `## Cycle 2` に書いた。それより前は cycle 1 の文のままで、cycle 2 で古くなった点は `## Cycle 2` の表に挙げた
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

## Cycle 2

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(承認の digest `debc4ca9fe00`)。issue #318
- Pipeline run: 2 回目(`cycle-count.json` は `cycle` 2)。run 開始時の HEAD は `aeab4b80`。これは origin/main `bbe2807d`(#328、fix/send-claude-paste)を、cycle 2 の `/test`(`7a70e773`)のあとに取り込んだ merge
- 先行 report(cycle 2): self-review の Cycle 2 addendum(`f9fef551`。C2-1 から C2-6 は解決。残る LOW は C2-7、C2-8 と、受け入れ済みの L3、N6)、verify の Cycle 2(`1d7eda3d`。pass。LOW 4 件、V2-1 から V2-4)、test の Cycle 2(`7a70e773`。pass。G5 が LOW、G6 と G7 が INFO)
- 見た範囲: `git diff bbe2807d aeab4b80` は 18 ファイル、+1,860/-16(この run の commit の前の数)。計画、report 5 つ、insights の 7 ファイルを除くと 11 ファイル、+939/-16(skill の 4 写しが 4 × +79/-1、テスト 2 ファイルが +565、`leader.md` が +45/-8、`director.md` が +6、tech-debt が +5/-3、spec と README が各 1 行)。この branch が変えた Go はテストの 2 ファイルだけで、本番の Go は変えていない。この run も Go のコード、テスト、skill、prompt は変えていない。足したのは文書だけ。この run の変更を入れた working tree の数(この行を足す前)は `git diff bbe2807d` で 18 ファイル、+1,971/-20、記録を除く 11 ファイルは +945/-20。walkthrough は書いていない
- 制約: 計画の本文は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `debc4ca9fe00`、`git diff --numstat` は 2 行追加、0 行削除(子の箇条を 2 つ足した)。AC の箱は `[ ]` のまま

### Summary

verify の LOW 4 件と self-review の C2-7、C2-8 を文書で扱った。V2-2 と C2-8 は spec の括弧に項を足して直した。V2-3 は tech-debt 188 行 (a) の注を、HEAD の行数と cycle 2 の中身に書き直した。V2-4 は 262、377、427 行に持ち越しの注を足し、verify が INFO にした 200 行 (g) にも足した。V2-1 と C2-7 は直さず、新しい行(497 行目)にした。test の G5 から G7 は 1 行にまとめた(498 行目)。origin/main の取り込みで 365 行の数が動いたので、取り込んだ木で数え直し、main の #319 の注と食い違わないようにした。

### Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-9 の #318 の項の「変えたのは」の括弧に、「件の処理」の手順 3 の項(hook の知らせを載せた QUESTION を小節へ送る項)を足した(V2-2、C2-8)。同じ項の「`/org` skill には足していない」という文の次に 1 文足した。director のない設定では、人が `/org` skill の受信箱の手順で答えるので `HOOK_SEAT:` が付かず、承認待ちや cap の待ちの leader が別の答えと読みうること、直さずに tech-debt の行に残したこと。足した理由は、元の理由(人が座席を spawn するセッションでは行が人の目に入る)が、leader が spawn する director なしの org には当たらないため(V2-1) |
| `docs/tech-debt/README.md` | 188 行 (a) の 2026-11-13 の注を書き直した(V2-3)。262、377、427、200 行の Trigger の末尾に 2026-11-13 の cycle 2 の持ち越しの注、Related の末尾に計画(262 と 427 はテストのファイルも)(V2-4)。365 行の #318 の注の数を取り込んだ木で数え直した。495 行の Trigger の括弧を言い直した。末尾に 2 行(497、498 行目)。下に行ごとの中身を書く |
| `docs/plans/active/2026-11-13-director-notice-handling.md` | Progress の節だけ。上の「制約」のとおり。足した 2 つは、origin/main の取り込みと確かめの記録(handoff のとおり)と、この run の要約 |
| `docs/reports/sync-docs-2026-11-13-director-notice-handling.md` | この report。cycle 1 の文は変えず、先頭の `Pipeline run` の行に cycle 2 への一言を足し、この節を足した |
| `docs/insights/events/2026-11-13-director-notice-handling.jsonl` | `sync_docs` の行(`--cycle auto` は 2、pass)。`git status --short` は既存の jsonl の `M` だけで、日付つきの slug の余計なファイルはできていない |

commit は 1 つ(`docs: sync docs for cycle 2 of director-notice-handling`)。report が自分の commit の SHA を書けないので、内容と report は分けていない。

### 入力ごとの扱い

| 入力 | 扱い |
|------|------|
| V2-1 / C2-7(director のない設定で、人の答えに `HOOK_SEAT:` が付かず、hook の QUESTION もすぐ resolve されない) | 直さない。tech-debt の新しい行(497 行目)。Debt は (a) 承認待ちと cap の待ちの leader が答えを別の答えと読むこと、(b) 件が開いたままの間、`orgHasOpenLeaderItem` が leader の待ちとみなして見張りが stall を知らせないこと、既定の設定は影響を受けないこと。(b) は verify の読みで、`watch.go` の 1228 から 1242 行を自分でも読んで同じ結論になった(QUESTION、BLOCKED、RESULT の resolved でない件を数える)。Trigger は `/org` skill の受信箱の手順の次の変更か、no-director の org の報告。#322(fix/send-input-box-marker)が `/org` skill の send の手順を変えるので、同じ項に触れうることを Trigger に書いた |
| V2-2 / C2-8(spec の変えた場所の並び) | spec の括弧に足した |
| V2-3(188 行の注の数、commit、中身) | 書き直した。下の表 |
| V2-4(262、377、427 行に持ち越しの注がない) | 3 行とも足した。200 行 (g) も足した(前例の注があるため)。下の表 |
| verify の INFO: 365 行の注 | 数え直した。下の表 |
| verify の INFO: 495 行の Trigger の括弧 | 言い直した。Trigger は「hook の行そのものの末尾に足された句」で、行を説明する文は数えない。cycle 2 は後者を足しただけで、`####` の見出しも足していないので、発火していない |
| verify の INFO: README の director の段落 | 変えていない。cycle 1 の文は cycle 2 のあとも正しく、すぐ resolve することや `HOOK_SEAT:` は段落の粒度では要らない |
| verify の INFO: 小節が 65 行 | 直していない。増えた分は cross-review の 2 件を閉じる文と C2-3 から C2-5 の直し |
| test の G5(`HOOK_SEAT:` の名前が skill 側と leader 側で別の文字列) | tech-debt 498 行目の (a) |
| test の G6(DECISION の形を `Parse` と `Validate` に通すテストがない)、G7(除外を書いた読み手が 2 つだけ) | 同じ行の (b)、(c)。どちらも INFO だが、払い方と Trigger が書けるので行に入れた |
| test の G1 から G4(cycle 1 のまま) | G1、G2 は 495 行目のまま。G3、G4 は INFO で、払い方がないので行にしていない(cycle 1 と同じ) |
| self-review の L3、N6 | L3 は 495 行目。N6 は cycle 1 のとおり行にしていない |

### tech-debt の中身

| 行 | 変えた所 |
|----|----------|
| 188 (a) | 2026-11-13 の注の括弧を書き直した。cycle 2 で足した `HOOK_SEAT:` の決まり(人の答えは `HOOK_SEAT: <座席 id>` の DECISION で届く。この DECISION は hook の QUESTION だけへの答えで、計画の承認、cap の QUESTION、ほかの QUESTION や BLOCKED の答えではない。director は何も送らずに resolve する)と、hook の項の外の 2 文(手順 4 の頭と、`### cross-review` の cap の読み手)を入れた。行数は `1,332 to 1,369`(`wc -l` を origin/main の `e902705c` と、この branch が `leader.md` を最後に変えた `8e8d26b5` で。`bbe2807d` も 1,332 行で、main は `leader.md` を変えていない)。cycle 1 の版が書いた 1,359 は `e95aa15f` の数で、cycle 2 の前だったことも書いた。払っていない持ち越しの形は変えていない |
| 262 | 注を足した。Trigger は「`/director` skill の受信箱の節(`受信箱の処理`)の次の変更」。`### 件の処理` はその節の中で、手順 3 に項が入ったので当たる。(1) から (3) は `director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` の話で、どちらも触れておらず、`WaitInbox` も締め切りの経路も変わっていないので払っていない。項は新しい `prompts_notice_test.go` の `TestDirectorSkill_SaysWhatToDoWithTheHumanNotes` が固定する。cycle 1 の表は 262 を「当たらない」とした(cycle 1 の skill の hunk は受信箱の節の外だけだった)。cycle 2 で当たった |
| 377 | 注を足した。Trigger は「`leader.md` の機能ごとの org の手順の次の変更」。手順 4 の頭に `HOOK_SEAT:` の除外の文が入り、`### cross-review` の cap の読み手にも 1 文が入ったので当たる。行が求める「2 行をファイルの編集ツールで書き換えると手順 4 に書く」は入っておらず、`/org` skill の split-plan の例も変わっていないので、RC4 は払っていない |
| 427 (1)(4)(7) | 注を足した。Trigger は「`### cross-review` 小節の次の変更」。cap の読み手に 1 文が入ったので当たる。文は断片の順で `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer` が固定し、返事の `REPORT:` の path やレポートの path を commit の前に確かめるコマンドは足していないので払っていない。(2)(3)(6) は当たっていない |
| 200 (g) | 注を足した。(g) は f1b0352e で、(f)(j) は b0fc4c59 で閉じている。前の branch は `leader.md` を変えるたびに同じ形の注を残している(2026-10-26 の 2 本、10-30、11-01、11-07)ので、そろえた。cycle 1 の表は 200 (g) を「当たらない」と読んだ(その時の hunk は `件を上げる` の 1 文と `運用規律` の 1 項)。cycle 2 の手順 4 と `### cross-review` の hunk は機能ごとの org の部分そのもので、前例の注は `運用規律` と agmsg の受信箱の節の編集も数えているため、注は cycle 1 の hunk も合わせて挙げた。開いたままの項は (a)〜(e)、(i)、(k)、(l) で、どれも計画の Scope にない |
| 365 | #318 の注の数字を書き直した。`prompts_notice_test.go` は 436 行(`wc -l`、最後に変えた `8e8d26b5`。cycle 1 の版にあった 339 は `f78e2b1b` の時点の数で、`8ae5ae1c` のあとは 360 だった。前の直しは件名で commit を指していたので、SHA にした)で、cycle 2 で `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer` が入った。6 ファイルは `aeab4b80` で `verbs.go` 2,615、`verbs_test.go` 5,893、`prompts_test.go` 2,722、`spawn.go` 3,101、`spawn_test.go` 6,408、`director_skill_sync_test.go` 1,307。この branch が変えたものはなく(`git diff --stat bbe2807d HEAD` の 6 ファイルは空)、`verbs.go` が `e902705c` の 2,597 から動いたのは main の #319(2,615)による。main の #319 の注と Related が先、この branch の注と Related が次、の順は取り込みの解決のまま変えていない |
| 495 | Trigger の括弧だけ言い直した(上の表)。中身と Related は変えていない |
| 497(新) | V2-1 と C2-7。上の表 |
| 498(新) | G5 から G7。(a) G5: 変異 g(`leader.md` と leader 側の文字列だけ `HOOK_SEAT_ID` にする)が、`TestDirectorSkill_` と `TestRenderRolePrompt_` の 163 本をすべて通り、対照 g0(`leader.md` だけ)は 2 本落ちる。(b) G6: 小節の DECISION の形を `Parse` と `Validate` に通すテストがない。(c) G7: 除外を書いた読み手は 2 つで、ほかの待ちは hook の項の一般の文に頼る。Trigger はそれぞれ、ヘッダの名前か `directorNoticeRules` か hook の項の次の変更、DECISION の形か `protocol.Parse` の次の変更、`leader.md` に DECISION の読み手を足すとき |

構造の確かめ(`git show HEAD:` との比較、scratchpad の `tdcheck.py`): 変わった既存の行は 188、200、262、365、377、427、495 だけ。行は 5 列(区切りの縦線で 7 つに分かれる)、バッククォートは列ごとに偶数、`file:line` の形は 0、U+FFFD は 0。plan の参照は 125 件で、無い参照は 0 件。#313 の行(464、465)と main の #319 の行(496)は触れていない。

#### 既存の行の Trigger(cycle 2 の hunk)

`docs/tech-debt/README.md` の全行の Trigger の列(取り消し線と日付つきの注を除いて)を、cycle 2 が触れた名前で引き直した(scratchpad の `sweep.py`。語は `leader.md`、`leader prompt`、`leader template`、`role prompt`、`director skill`、`受信箱の処理`、`件の処理`、`feature-org`、`### cross-review`、`DECISION`、`HOOK_SEAT`、`RolePromptVars`、`prompts_notice_test`、`prompts_test`、`director_skill_sync`、`運用規律`、`件を上げる`、`orgHasOpenLeaderItem` など 34 語)。編集の前に 46 行が当たり、Trigger を読んだ。

| 行 | 判断 |
|----|------|
| 188 (a)、200 (g)、262、365、377、427 (1)(4)(7) | 当たった。注を足した、または書き直した |
| 495 | 当たらない。括弧の言い方を決めた |
| 384、426 | 当たらない。384 の条件(座席を止める規則)は変わっていない。cycle 2 が書き直した hook の項は別の項。426 の条件は節を足すことで、足していない |
| 239 (o)(p)、380 | 当たらない。239 は手順 9 と 10、380 は send と確かめの手順で、どれも変えていない。小節は「答える順」の 1 と 2 を指すだけ |
| 195、197、224、355、367、372、471、472、280 ほか | 当たらない。名前は出るが、条件は別のコード、別の節 |

### origin/main の取り込み(`aeab4b80`)のあとの確かめ

- 取り込みは `/test` のあとで、`/test` の report は取り込みの前の木(`1d7eda3d`)のもの。#328 が変えたのは `verbs.go`、`send_paste.go` とそのテスト、`internal/cli/org.go`、`/org` skill の 4 写し(send の手順)、`agent-messaging.md` の rule(root と雛形)、`AGENTS.md`、org-runtime の spec、tech-debt などで、`leader.md` と `/director` skill は変えていない(`git diff --name-only e902705c bbe2807d` で確かめた)。衝突は tech-debt の 2 か所(365 行と、表の末尾)で、両側を残した
- handoff の記録(orchestrator が取り込んだ木で通したもの): check-sync、check-skill-sync、gofmt、vet。`internal/org` の `RenderRolePrompt|DirectorSkill|RenderDirectorPrompt|Send|Paste` と `internal/cli` の `NoticeSkill|ScaffoldVersion|StatusHelp|Send` の絞ったテスト。`internal/cli` の最初の回は load 28 ほどで落ち、出力は残していない。続く 2 回は通り、1 回は `-count=1 -v` で 135 PASS。この run では回していない
- この run で回したもの: `./scripts/check-sync.sh` は `PASS: all files in sync.`、`./scripts/check-skill-sync.sh` は `14 skill(s) in lock-step`、`git diff --check` は rc 0。この run は skill の写しに触れていない
- 回していないもの: Go のテスト全体と `./scripts/run-test.sh`(この run は Go を変えていない)。`/test` の cycle 2 は `1d7eda3d` の木で回していて、その木はこの branch の最後の Go の commit `8e8d26b5`(`leader.md` と `prompts_notice_test.go` の各 1 行の置き換え)を含む。取り込みで変わったのは main の #328 の分だけで、その分の確かめは上の handoff の記録のとおり

### 実装と合っていると確かめたもの

| Doc / contract | 結果 |
|----|----|
| `AGENTS.md`、`CLAUDE.md`、`.claude/rules/ralph/*`、`docs/quality/*`、`docs/architecture/*` | 小節にも知らせの扱いにも触れていない。`git grep -i` で `ralph upgrade`、`older than this CLI`、`scaffold version`、`manifest.toml` を引くと当たるのは `AGENTS.md` の 87、88、106 行目で、どれも scaffold と upgrade の仕組みの説明。`返事が来るまで`、`返事まで止め` は rules、quality、specs、README、AGENTS.md に無い |
| `HOOK_SEAT` を含む追跡ファイル | skill の 4 写し、`leader.md`、`prompts_notice_test.go`、multi-org director の spec、tech-debt、計画、report だけ。`agent-messaging.md` は任意のヘッダを認める書き方(`<any other KEY: value header lines>`)なので、`HOOK_SEAT:` の名を挙げなくても合う。同じ種類のヘッダ(`XREVIEW:`、`VISUAL:`、`FORK 1:`)も spec にだけ書かれている |
| `docs/specs/2026-08-01-org-runtime.md` | #318 の項も、DECISION のヘッダの決まりも無い(verify と同じ) |
| `README.md` の director の段落 | cycle 1 の 1 文は cycle 2 のあとも正しい |
| `docs/architecture/repo-map.md`、`docs/quality/*.md` | script も hook も足していないので、変える所が無い |

### 実行したもの

- `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`git diff --check`: すべて rc 0(commit の前、変更を含めて)
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-13-director-notice-handling.md`: 編集の前後とも `debc4ca9fe00`
- tech-debt の構造の確かめ(`tdcheck.py`)、Trigger の引き直し(`sweep.py`、`showtrig.py`)。どれも scratchpad で、repo には置いていない
- `./scripts/insights-append.sh --slug director-notice-handling --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`: `--slug` は日付なしの slug。cycle は 2

### 残り

- AC1 から AC8 のチェックボックスは `[ ]` のまま(計画の本文は変えない約束)。`/pr` の前に付ける
- PR 本文の Known gaps には、cycle 1 の分(L3 = 495 行目、N6)に加えて、V2-1 と C2-7(497 行目。director のない設定)、G5 から G7(498 行目)を書くとよい。G3(director と leader が実際の会話で文に従うか)は行にしていない。実会話では確かめていない
- PR 本文の release の項は、cycle 1 の分(director が知らせの行を人に伝える動き)に、hook の QUESTION を director が DECISION を送らずにすぐ resolve すること、人の答えは `HOOK_SEAT:` の DECISION で届くことを足すとよい。これは binary に埋め込む prompt と、`ralph upgrade` で配る skill の両方に及ぶ
- パイプラインの cycle は 2 回目(上限に達している)。この run は文書だけを変え、record だけの修正ではない(skill、prompt、tech-debt、spec を含む)ので、cross-review の扱いは lead が決める
- V2-1 と C2-7 を直す場合は `/org` skill の受信箱の手順(4 写し)を変える。#322 が同じ skill の send の手順を変えるので、重ならない順で
