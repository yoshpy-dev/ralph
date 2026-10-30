# Walkthrough: leader-sync-docs(機能の org の leader が gate の前に sync-docs を doc-maintainer に任せ、予約の外の文書のずれを follow-up として残し、director がそれを集めて文書だけの org を人と立てられるようにする)

- Date: 2026-10-30
- Plan: docs/plans/active/2026-10-30-leader-sync-docs.md(/pr で archive に移す)
- Issue: #268(org: leader が sync-docs を行い、予約外の文書のずれを文書の org にまとめる)
- PR: 未作成(/pr で作る)
- Branch: feat/leader-sync-docs(この walkthrough の前の HEAD は 1015e3c0。merge を除く 46 commits と、origin/main を取り込んだ merge commit 0dfc49f0 の 1 つ)
- Base: 0351800d(origin/main、「Merge pull request #290」= org-state-dir-local)。0dfc49f0 で取り込んだので、`git diff origin/main...HEAD` の分岐元も 0351800d。branch を切った元は 1e266f96
- 読む範囲: `git diff origin/main...HEAD` は 21 files、+3809/-94(HEAD 1015e3c0。この walkthrough は数えていない)。Go の本番コードは 0 files で、`.go` は新しいテスト 2 files だけ。内訳は、binary に埋め込まれる雛形 `internal/org/prompts/leader.md` が 1 file(+131/-5)、skill が 8 files(+984/-88。`/org` と `/director` の同じ文を `.claude`・`.agents`・`templates/base/.claude`・`templates/base/.agents` の 4 面に写したもの)、Go のテストが 2 files(+1607)、記録(計画・report 5 つ・insight event)が 7 files(+1083)、残りの文書が 3 files(+4/-1。README、仕様、tech-debt)。動きを決めるのは `leader.md`(手順 9 の末尾の 102 行)、`.claude/skills/org/SKILL.md`(+156/-20)、`.claude/skills/director/SKILL.md`(+90/-2)。テストは新規の `internal/org/prompts_leader_syncdocs_test.go`(717 行、12 本)と `internal/org/skill_doc_followup_test.go`(890 行、18 本)
- 後工程: パイプラインは 3 回。cross-review が 1 回目と 2 回目に ACTION_REQUIRED を 1 件ずつ出し、2 回目のあとに上限を 2 から 3 に上げた(`cycle-count.json` は 2 のまま)。self-review は 5 run(cycle 1 の run 1〜3、cycle 2、cycle 3。CRITICAL と HIGH は 0、MEDIUM は run 1 の 2 件、LOW は 6・3・1・2・1 件。どれも Merge 可)、verify は 4 run(初回は HIGH 1 件で fail、残り 3 回は pass)、test は 3 run(どれも pass)、sync-docs は 3 回、cross-review(codex)は 3 回(所見は 1・1・0 件)。記録だけの修正の例外を使った記録は triage report にない

```
何を変えるか → 読む順 → 手順 9 の末尾 → 手順 12 と 14 → /org skill → /director skill → 仕様と README → テスト → 直しの経過 → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで:
  - 機能の org の leader は、reviewer のレビューが通ったあと、手順 9 でレポートをコミットして fetch と rebase をし、手順 10 で org の report と計画の archive をコミットし、手順 12 で reviewer に gate を打たせる。標準フローの `/sync-docs` に当たる段はない(base の `leader.md` と `/org` skill に `doc-maintainer` は 0 件、`/director` skill に `DOC_FOLLOWUP` は 0 件。`git show origin/main:<path>` を `grep -c` で数えた)。
  - 文書が変わるのは、implementer が予約したパスの中でした分だけ。README・AGENTS.md・`.claude/rules/` のような共有の文書は予約の外にあることが多く、並行する org が触ると merge で衝突して `ralph org regate` が止まる(issue の背景)。予約の外のずれはどこにも記録されない。
- この PR から:
  - 手順 9 の末尾(fetch と rebase のあと、手順 10 の前)で、leader が sync-docs を doc-maintainer の subagent に渡す。leader 自身は文書を直さない。Claude Code は `Task(subagent_type="doc-maintainer")`、Codex は `.codex/agents/doc-maintainer.toml`。
  - doc-maintainer が直してよいのは機能の予約の中だけ。予約の外で見つけたずれは、sync-docs の report の `## 予約外の文書の follow-up` に 1 件 1 行で書かせる。leader はそれを PR の本文の同名の節と、RESULT の `DOC_FOLLOWUP: <パス>: <ずれ>` の行に写す。
  - doc-maintainer が終わったら、leader は変わったパスを読む。予約の外のパス、凍らせたファイル、機能の計画のどれかが 1 つでも変わっていれば、report のほかの変更をすべて戻して follow-up に足す。残りと report を `docs: sync docs for <機能の slug>` の 1 コミットにする。
  - 呼べないとき(定義が worktree にない、呼び出しの失敗、report を書かずに終わった)は、変更を戻して飛ばし、RESULT に `SYNC_DOCS: skipped (<理由>)` を書く。
  - gate が文書の変更で落ちたら、今の `GATE: fail` のやり直しに乗る。やり直しの手順 9 で sync-docs がもう一度走るが、これまでの sync-docs のコミットが変えたファイルは凍らせて渡す。
  - `/director` は、RESULT の件の follow-up を人に報告し、resolve の `--note` には PR の URL と件数の 1 行だけを書く。分割計画のどの機能も `merged` か `closed` になったら、follow-up を集めて、文書だけの機能(`- Type: docs`)1 つの分割計画の案を人に見せる。人が進めると言ったら、今の手順で org を立てる。
  - Go の本番コードは変えていない。変わるのは、binary に埋め込まれる雛形の文、skill、テスト。
- issue との違い(issue #268 の本文と比べた):
  - 置き場所: issue は「手順 9 のレポートのコミットのあと、report と計画の archive(手順 10)の前」と書く。この PR は手順 10 の先頭ではなく手順 9 の末尾の段落に置き、番号を振り直していない(D1)。`prompts_test.go` が手順 10 の 1 行目(`10. ` に続く `ralph org report`)を固定していて、振り直すと雛形の 34 か所とテストの 23 か所の参照が動き、#269 も同じ雛形を触るため。順は issue のとおり(reviewer、rebase、sync-docs、report と archive、gate)。
  - 予約の外を直されたとき(all-or-nothing の戻し): issue は「文書の変更は予約の中だけ」とだけ書く。この PR は、予約の外の変更を見つけたら一部でなく report 以外のすべてを戻す。写しの組(skill の写しなど)の片方だけが残ると、gate の写しの検査(`check-skill-sync.sh`)が落ち、予約の外を直せない implementer も直せずに進めなくなるため(plan advisory の HIGH)。doc-maintainer には、組のすべてが予約の中にあるときだけ組を直させ、1 つでも外なら組の写しを 1 つ 1 行ずつ follow-up に書かせる(AC8)。
  - gate が落ちたとき(D2c): issue は「gate の TASK のやり直しの手順に乗せる」だけ。この PR は、やり直しの sync-docs が同じ変更を入れ直して同じ理由で落ちる往復を止めるため、これまでの sync-docs のコミットが変えたファイルの和集合を git から引いて凍らせる。最初の計画になく、self-review の M1 で足した。
  - 検査の名前: issue は `scripts/check-sync.sh` を例に挙げる。雛形と `/org` skill は、scaffold した project に入る `scripts/check-skill-sync.sh` だけを名指す。`check-sync.sh` はこの repo だけにあり、`check-template-purity.sh` が `templates/base/` の文に出ることを断る(verify V1)。
  - 変わったパスの読み方(`--untracked-files=all`): `git status --porcelain --untracked-files=all` を使う。既定の形は新しい未追跡のディレクトリを `?? docs/new/` の 1 行にまとめ、予約とファイルごとに照らせない(cross-review cycle 1 の P2)。
  - 呼ぶ前の確かめ: doc-maintainer を呼ぶ前に、同じコマンドが何も出さないことを確かめる。手順 9 の最初の確かめは短い形のまま(`prompts_test.go` が固定)で、`status.showUntrackedFiles=no` では他人の未追跡のファイルが見えず、あとの戻しがそれを消しうるため(F5b)。
  - director が集める時期と印: issue は「全機能の merge のあと」。この PR は `merged` か `closed` で始め、`closed` の機能の follow-up は集めずに機能の名前を人に言う。集めた印として、文書の分割計画の本文の先頭に `元の分割計画: <id>` を書き、印のある分割計画は集め直さない。文書の分割計画から出た follow-up は人に報告するだけにする(self-review L4)。
  - director が予約に入れるもの: 写しの組は組のすべてを `- Reserve:` に入れる。案を見せる前に、follow-up の文書を読むテストと、その文を固定するテスト・fixture を探して `- Reserve:` に入れ、足したものと理由を人に言う。予約の外のテストは implementer が直せず、文書の org が gate を通れないため(cross-review cycle 2 の P2)。
  - `--note`: `ValidateInboxNote` が 1 行・500 字までなので、follow-up そのものは入れず、PR の URL と件数だけにする(plan advisory の MEDIUM)。URL は件の本文でなく `ralph org merge-order --all` の PR の列から取る(self-review L5)。
  - 受け入れ条件は、issue が 5 つ、計画が AC1〜AC8(AC8 が写しの組)。
- 計画の本文との違い(承認の digest は最後に `94dc29701d29`。本文は 4 回書き換えて承認し直した):
  - 書き換えは self-review と cross-review の直しに合わせたもので、承認はそのたびに自動(ユーザーの「最後まで止まらずに」の決め)。digest は計画の Progress の記録で、`8655b404a42a`(最初)→ `c8485493b418`(c2370794。self-review run 1 のあと、D2・D2c の新設・D3・D4・AC6)→ `ab78e5f0645b`(1286ab79。D2c の `git log --grep` を件名の完全一致に)→ `220fa1f3994f`(ef4a15a7。D2 と Risks を `--untracked-files=all` と呼ぶ前の確かめに)→ `94dc29701d29`(ebc2b64a。D4 にテストと fixture の予約)。本文はこの PR の中身と合っている。
  - D2b は写しの検査を「`scripts/check-skill-sync.sh`・`scripts/check-sync.sh` があれば」と書いたまま。雛形と `/org` skill は F4 で `scripts/check-skill-sync.sh など` にした。本文を直すと承認のし直しが要るので、計画は直していない(verify の INFO)。
  - Progress の S1 の行は「戻した変更は report に足さず、手順 14 のために控える」と書く。F1 で、控えを leader の記憶に置かず、report の `## 戻した変更` の節と follow-up の節に書く形に変えた(self-review L3)。D2 の本文は書き換え済みで、S1 の行だけが前の形のまま。
  - Progress の Test artifact の行は「変異 23 個が落ちる」。test の report は cycle 3 の変異 27 個のうち 26 個が落ち、1 個(D6f)が残ったと書く。23 は依頼の分(当て直し 21 と手順 6 の 2)で、依頼の外で足した 4 個のうち 3 個が落ち、1 個が残った。
  - AC1〜AC8 の checkbox は 1015e3c0 で付けた。Progress の「PR created」は未チェック。
  - 計画の図解ページ(`.harness/state/plan-visual/leader-sync-docs.html`)は gitignore の下にあり、この walkthrough では読んでいない。
- release のノート: 計画の Rollout の 1 文。「機能の org の leader が gate の前に sync-docs を doc-maintainer に任せ、予約外の文書のずれを RESULT の `DOC_FOLLOWUP:` と PR の本文に書く。director はそれを集めて文書の org を人と立てる」。雛形は binary に埋め込まれるので、新しい binary で立てた leader から効き、走っている org の leader には効かない。skill は `ralph upgrade` で配られる。
- 戻し方: この PR を revert する。台帳の形と ralph の動きは変えていないので、走っている org への影響はない(計画の Rollout)。

## 2. 読む順

1. 計画の D1〜D5(D2b・D2c を含む)と AC1〜AC8。後ろの節の D 番号と F 番号はここを引く。
2. `internal/org/prompts/leader.md` の手順 9 の末尾(`:348-448`)。前文の 1 文(`:119-120`)も(3 節)。手順 12 の `GATE: fail` の枝(`:534-543`)と手順 14(`:550-610`)は 4 節。
3. `.claude/skills/org/SKILL.md`: 「org の終わり方」の 1(`:1050-1150`)、4(`:1213-1219`)、6(`:1233-1272`)、「座席内 fan-out」(`:1598-1604`)、受信箱の RESULT(`:470-478`)、2 つの要約(`:1770-1776`、`:1922-1927`)、「完了条件」(`:1944-1956`)(5 節)。
4. `.claude/skills/director/SKILL.md` の「件の処理」の RESULT(`:655-675`)と「### 予約外の文書の follow-up」(`:1086-1148`)(6 節)。
5. 仕様 FR-4 の注(`docs/specs/2026-10-07-org-multi-org-director.md:93`)、README(`:280`)、tech-debt の 2 行(`docs/tech-debt/README.md:404-405`)(7 節)。
6. テスト(8 節)。skill の写し 3 面は、1 面を読めば足りる(`cmp` で 6 組が一致、10 節)。

## 3. 雛形の手順 9 の末尾(`internal/org/prompts/leader.md:348-448`)

- 前文(`:119-120`): 機能の org の前文は、手順のうちの書き込み(手順 6 の計画、9 の reviewer のレポート、10 の report と計画の移動、12 の計画の戻し)を予約の外に書いてよいと言う。ここに「手順 9 の終わりに sync-docs が書く report(`docs/reports/` の下)」を 1 文足した。`prompts_test.go:549` がこの列挙を 1 つの語句で固定しているので、列挙は書き換えず、直後に足した。
- 手順 9 の最初の段落(reviewer のレポートのコミット、fetch、rebase、gate の TASK への `GATE: pass` は手順 12 で扱うという文)は変えていない。その直後、`:348` の「rebase が済んだら(要らなかったときも)」から 2 つの段落が続く。
- 流れ(1 つ目の段落):

| 順 | 行 | 内容 |
|---|---|---|
| 1 | `:348-356` | doc-maintainer を呼ぶ。Claude Code は `Task(subagent_type="doc-maintainer")`、Codex は `.codex/agents/doc-maintainer.toml`。定義が worktree にない(`.claude/agents/doc-maintainer.md`、Codex は `.codex/agents/doc-maintainer.toml`)ときは呼ばずに飛ばす。戻す変更はない |
| 2 | `:356-368` | 呼び出しの失敗(Codex で custom agent を呼べない、途中で止まった、時間切れ)と、report を書かずに終わったときは、同じ 1 つの枝。status に出るパスを report も含めてすべて戻し、`## 戻した変更` は書かず、status が空なのを確かめてから飛ばす |
| 3 | `:369-377` | 呼ぶ前に `git status --porcelain --untracked-files=all` が何も出さないことを確かめる。出たら呼ばず、手順 9 の最初の確かめと同じに振り分ける(reviewer のレポートはコミットして確かめ直し、implementer の変更は TASK でコミットを頼んで(per-plan では `PLAN:` のヘッダ)手順 8 の往復へ、それ以外は BLOCKED) |
| 4 | `:378-390` | これまでの sync-docs のコミットを引く。`git log --format='%H %s' origin/<既定のブランチ>..HEAD \| awk 'substr($0, index($0, " ") + 1) == "docs: sync docs for <機能の slug>" { print $1 }'` で件名が完全一致するものだけを新しい順に出し、それぞれの `git show --name-only --format= <コミット>` のファイルの和集合(`docs/reports/sync-docs-` で始まる report は除く)を、凍らせたファイルとして渡す。`git log --grep` は `grep.patternType` で `^`・`$` の読みが変わり、本文の行にも当たるので使わない。gate の `LOG:` から原因を推し量ることもしない |
| 5 | `:391-409` | 渡す指示(D2b): 計画のパスと base(`origin/<既定のブランチ>`)、予約したパスとその読み方(ファイルはそのパス、`/` で終わるディレクトリはその下、`.` は repo 全体)、予約の中だけを直す、凍らせたファイルは予約の中でも直さず follow-up に書く、写しの組は組のすべてが予約の中のときだけ直し、1 つでも外なら直さずに組の写しを 1 行ずつ follow-up に書く、直したあと `scripts/check-skill-sync.sh など` があれば打って落ちる自分の変更を戻す、report の置き場所と follow-up の節、insight event・ステージ・コミット・計画の変更・`ralph org` を禁じる |

- 流れ(2 つ目の段落、doc-maintainer が終わったあと):

| 順 | 行 | 内容 |
|---|---|---|
| 6 | `:411-417` | 同じ status で変わったパスを読む。`--untracked-files=all` を付ける理由(新しいディレクトリが 1 行にまとまる、`status.showUntrackedFiles=no` では何も出ない)を文に書いてある |
| 7 | `:418-435` | report のほかに、予約の外のパス・凍らせたファイル・機能の計画のどれかが 1 つでも変わっていれば、report のほかをすべて戻す。戻す前に `git diff HEAD -- <パス>`(未追跡は中身)を読んで控える。追跡しているファイルとステージした新しいファイルは `git restore --source=HEAD --staged --worktree -- <パス>`、未追跡は一覧のファイルを 1 つずつ消す。戻したら report に `## 戻した変更` を足し、同じ項を `## 予約外の文書の follow-up` にも足す(`なし` の行は消す)。一部だけ戻さない理由(写しの組の片方が残る)、計画を戻す理由(per-plan の承認の検査)、凍らせたファイルを戻す理由(同じ理由で gate がまた落ちうる)も文にある |
| 8 | `:435-443` | 残った変更と report を `docs: sync docs for <機能の slug>` の 1 コミットにし、status が空なのを確かめる。report だけが変わったときは report だけで同じメッセージのコミット。report も前の回と同じで何も出なければコミットせず手順 10 へ。メッセージは前のコミットを引く側が件名の完全一致で探すので、言い方を変えない |
| 9 | `:444-448` | gate をやり直すとき(手順 12 の `GATE: fail`、implementer の手が要る断り)は、手順 8 の往復を経て手順 9 を通り直すので、sync-docs も新しい HEAD でもう一度走る。手順 14 が使うのは最後の回のもの(最後の回の report か、最後の回で控えた `SYNC_DOCS:` の行) |

- 見てほしいところ:
  1. 順の固定。呼ぶ前の確かめ、前のコミットの引き、渡す指示、終わったあとの確かめ、戻し、コミットがこの順で 1 つの段落に入っている。手順 10 の 1 行目は変えていない。
  2. コミットのメッセージ `docs: sync docs for <機能の slug>` が契約の一部であること。言い方を変えると凍らせる一覧が空になる。
  3. 凍らせる一覧は回ごとに増えるだけで、`GATE: fail` の原因が文書かどうかを見ない。そのため、やり直しで implementer のコードが前の回に直した文書をまた古くしても、その文書は直さず follow-up になる。計画が決めた引き換えで、self-review run 2 は所見にしなかった。
  4. 一覧から `docs/reports/sync-docs-` の report を除くこと。除かないと、同じ日のやり直しで doc-maintainer が report を書けない(F1)。
  5. doc-maintainer の定義の `memory: project` が書く `.claude/agent-memory/` は、root と `templates/base/` の `.gitignore` が無視するので status に出ず、「すべて戻す」を起こさない(self-review run 1)。
  6. 戻しの `git restore --source=HEAD --staged --worktree -- <パス>`。`git restore <パス>` だけでは、ステージ済みの変更に rc 0 で何もせず、そのあとのコミットに入る(L1。使い捨ての repo で確かめた)。未追跡をディレクトリごと消さないのは、ignore されたファイルを消さないため。
- 固定しているテスト(`internal/org/prompts_leader_syncdocs_test.go`、各テストの名前は `TestRenderRolePrompt_Leader_FeatureOrg` で始まる): `SyncDocsEndsStep9`(`:138`。順、手順 10 の 1 行目)、`SyncDocsHandsTheDocMaintainerItsLimits`(`:186`)、`SyncDocsUndoesEveryChangeOrNone`(`:253`)、`SyncDocsStatusListsEachNewFile`(`:294`)、`SyncDocsRecordsTheUndoneChangesInTheReport`(`:381`)、`SyncDocsFreezesEarlierSyncDocsFiles`(`:415`)、`SyncDocsFreezeFindsTheFixedMessage`(`:451`)、`SyncDocsSkipsWithoutTheDocMaintainer`(`:527`)、`SyncDocsCountsANoReportEndAsASkip`(`:578`)、`SyncDocsChecksForLeftoversBeforeTheCall`(`:608`)、`SyncDocsRunsAgainOnARedo`(`:639`)、`ResultAndPRBodyCarryTheDocFollowups`(`:675`)。

## 4. 手順 12 の `GATE: fail` の文と手順 14(`leader.md:534-543`、`:576-610`)

- 手順 12 の `GATE: fail` の枝(`:538-543`): 前の文(archive を戻し、`REASON:` と `LOG:` を TASK に書いて手順 8 へ、reviewer が通したら手順 9 からやり直す)のあとに 1 文足した。手順 9 の sync-docs のコミットが変えたファイルで落ちたとき(写しの検査など)も同じで、implementer に予約の中で直しを頼む。手順 9 からやり直すとその終わりで sync-docs がもう一度走り、これまでの sync-docs のコミットが変えたファイルは凍らせて渡すので、同じ変更を入れ直させない。
- 手順 14 の PR の本文(`:575-584`): 本文に `## 予約外の文書の follow-up` の節を置く。中身は、手順 9 の `git log` と `awk` のコマンドが出す 1 行目(最も新しい sync-docs のコミット)が入れた `docs/reports/sync-docs-` のファイルを読み直し、同じ名前の節の項を 1 件 1 行(パスと何が古いか)で書く。手順 9 で戻した変更もその節に足してあるので、記憶からは足さない。なければ `なし`。最後の回で sync-docs を飛ばしたときは report を読まず(最も新しい sync-docs のコミットは前の回のもの)、控えた `SYNC_DOCS: skipped (<理由>)` の理由を書く。
- 手順 14 の RESULT(`:587-610`): ヘッダの `HEAD:` と `BASE:` は今までどおり gate の出力のファイルから写す。本文には PR の本文と同じ項を 1 件 1 行の `DOC_FOLLOWUP: <パス>: <ずれ>` で書く(なければ `DOC_FOLLOWUP: なし`)。本文が 2,000 字に収まらなければ、その行の代わりに件数と PR の本文の節を指す 1 行(`DOC_FOLLOWUP: 14 件(PR の本文の「予約外の文書の follow-up」の節)` など)。最後の回で飛ばしたときは `DOC_FOLLOWUP:` の代わりに `SYNC_DOCS: skipped (<理由>)`。例のメッセージ(`:600-610`)に `DOC_FOLLOWUP:` が 2 行入った。
- 見てほしいところ: 飛ばしたとき `DOC_FOLLOWUP: なし` を書かないこと。確かめていないのに「なし」と書かないため(S1 で計画から決めた)。
- 固定しているテスト: `ResultAndPRBodyCarryTheDocFollowups`(`:675`。例の RESULT を `validateEscalation` に通し、`protocol.DefaultMaxBodyChars` が 2000 であることを見る)、`SyncDocsRunsAgainOnARedo`(`:639`)、`SyncDocsSkipsWithoutTheDocMaintainer`(`:527`)。

## 5. `/org` skill(`.claude/skills/org/SKILL.md`)

- 「org の終わり方」の 1(`:1050-1150`): 雛形の手順 9 の末尾と同じ文を、skill の番号(手順 10 は 2、手順 14 は 6)で書いた。呼ぶ前の確かめの振り分けの終わりだけは、雛形が「手順 8 の往復に戻り」と書くところを「reviewer にレビューを頼み直し」、BLOCKED の上げ方を縮める、という書き分けがある(手順 9 の最初の段落の雛形と skill の書き分けと同じ)。テストはこの部分より前の共通の文までを突き合わせる。
- 4 の `GATE: fail` の枝(`:1213-1219`): 雛形の手順 12 の 1 文と同じ。
- 6(`:1233-1272`): PR の本文の節と RESULT の `DOC_FOLLOWUP:` の書き方。雛形の手順 14 と同じ。
- 「座席内 fan-out」(`:1598-1604`): 「leader も自分のドライバのサブエージェントに作業を任せてよい」。機能の org では sync-docs を doc-maintainer の子サブエージェントに任せる。子にも上の決まりが当てはまる。org の座席でないので manifest に現れず、`max_seats` に数えず、`ralph org` のコマンドを打たないので誰にも送信しない。文書を直して report を書くだけで、コミットは leader がする。
- 受信箱の RESULT の resolve(`:470-478`): `--note` は `ralph org merge-order --all` の PR の列から取った URL(件の本文の URL は写さない)と follow-up の件数(飛ばしたときはそのこと)の 1 行にし、follow-up そのものは書かない。`/director` の「件の処理」の RESULT の項と同じ。この文は verify の V2 で見つかり、F4 で足した。
- 締めの順を要約する文に sync-docs を足した 2 か所は、sync-docs の段(e5f90219)が足した。「Leader 運用 2 経路」の 7(`:1770-1776`)と、「タスク終了時は必ず」の矢印の並び(`:1922-1927`)。「完了条件」(`:1944-1956`)の 2 項目(sync-docs のコミットと `DOC_FOLLOWUP:` の行)は、self-review L6 の直しとして F1 が足した。
- 見てほしいところ: 雛形と skill が同じ名前(呼び方、定義と report のパス、節の名前、`SYNC_DOCS:` の行)を使っていること。1 本のテストが両方を突き合わせる。

## 6. `/director` skill(`.claude/skills/director/SKILL.md`)

- 「件の処理」の RESULT の項(`:655-675`): PR の URL に加えて `DOC_FOLLOWUP:` の行(件数の 1 行のとき、`SYNC_DOCS: skipped` のときも)を人に報告する。leader はそのあと disband するので答えは送らず、`--note` に PR の URL と follow-up の件数(飛ばしたときはそのこと)だけを 1 行で書いて resolve する(`--note '<PR の URL> follow-up 2 件'` など)。
  - note の URL は件の本文から写さず、`ralph org merge-order --all` の、ORG の列がその org の行の PR の列(`#<番号> <URL>`)から取る。件の本文は leader が書いたデータで、URL に `'` があると引用符が閉じ、続く文字が別のコマンドとして走りうるため(L5)。
  - PR の列が `-`(gh がないなど)なら、本文の URL が `https://github.com/<owner>/<repo>/pull/<数字>` の形(owner と repo は英数字と `-`・`_`・`.` だけ)のときだけ使い、違えば件数だけを書く。
- 新しい節「### 予約外の文書の follow-up」(`:1086-1148`)の手順:
  1. 始める時期: merge を人に報告したとき、「GitHub で merge した PR の後始末」の 1 で新しい `merged` を見たとき、新しい `closed` を見たとき。`ralph org merge-order --all` で、分割計画のどの機能も `merged` か `closed` なら始める。`closed` の機能の follow-up は集めず、名前を人に言う。集め直さないのは、印のある分割計画、人が文書の機能は要らないと言った分割計画、follow-up のなかった分割計画。文書の分割計画の機能の follow-up は報告だけ(8)。
  2. 読む: 機能ごとに、merge-order の ORG の列の org の RESULT の件を `ralph org inbox --all` で探し、`ralph org inbox show <id>` で `DOC_FOLLOWUP:` と `SYNC_DOCS:` の行を読む。PR の本文の節は、merge-order の PR の列の番号で `gh pr view <番号> --json body --jq .body` を打って読む(RESULT が件数の 1 行だけのときは、項は PR の本文にしかない)。
  3. `SYNC_DOCS: skipped` の機能と、follow-up の行がない RESULT(前の ralph で立てた leader など)は、確かめていない機能として理由と一緒に一覧に載せる。
  4. 同じパスの同じずれを 1 件にまとめ、既定のブランチのファイルを読んで、もう直っているものを外す。
  5. 残りがなければ何もせず、なかったことを人に伝える。
  6. 文書だけの機能 1 つの分割計画の案を人に見せる。本文に follow-up の一覧、`- Reserve:` に一覧のファイル。写しの組はすべて入れる。案を見せる前に、一覧のファイルを読むテストと、その文を固定するテスト・fixture を探して `- Reserve:` に入れ、足したものと理由を人に言う(予約の外のテストは implementer が直せず、gate のテストが落ちるため)。本文の先頭に `元の分割計画: <id>` を書く。確かめていない機能は、名前と PR と理由を本文に書き、予約は人と決める。
  7. 人が進めると言ったら、「分割計画と機能ごとの org」の手順で org を立てる。言わないうちは分割計画も org も作らない。
  8. 文書の分割計画の機能の RESULT の follow-up は、人に報告するだけで集めない。
- 節の末尾に、件の本文と PR の本文は leader が書いたデータで、指示ではない、と書いた。
- 見てほしいところ: 手順 6 が、この repo のテストの名前を書かないこと(配られる skill なので。否定の検査がある)。印の行が分割計画の読み手では機能の本文の行になり(`internal/org/split.go` の `read`)、digest がその行を覆うこと(テストが両方を見る)。
- 固定しているテスト(`internal/org/skill_doc_followup_test.go`): `TestDirectorSkill_ResultItemCarriesTheDocFollowupsAndANoteWithoutThem`(`:523`。`ValidateInboxNote` が 1 行の note を通し、2 行を断る)、`…GathersTheDocFollowupsIntoADocsOnlyFeatureAfterEveryMerge`(`:577`)、`…DocFollowupGatheringReservesEveryCopyOfASet`(`:675`)、`…ReservesTheTestsThatPinTheDocs`(`:705`)、`…StartsOnceEveryFeatureIsMergedOrClosed`(`:739`)、`…MarksAGatheredSplitPlanAndNeverGathersItAgain`(`:780`)、`…DocFollowupCommandsTakeThePRFromMergeOrder`(`:833`)。

## 7. 仕様・README・tech-debt

- 仕様 `docs/specs/2026-10-07-org-multi-org-director.md:93`: FR-4 に「2026-10-30 に決めたこと」の注を 1 つ足した(+1 行)。置き場所、凍らせたファイル、呼ぶ前の確かめ、変わったパスの読み方、すべて戻すこと、飛ばす 3 つの条件、director の集め方(`merged` か `closed`、印、写しの組、テストと fixture の予約)を要約する。末尾で、leader がこの手順で sync-docs を走らせることは実機で確かめていないので FR-4 の印は変えない、と言う。FR-4 の checkbox は `[ ]` のまま。注は verify の V3(822acf62)と V4(545efea1)で 2 回直した。
- README `README.md:280`(`ralph org gate` の段落): leader が rebase のあとに doc-maintainer の subagent に予約の中だけの sync-docs を任せてコミットし、予約の外のずれは RESULT と PR の本文の follow-up に書くこと、director が全機能の `merged` か `closed` のあとに文書だけの機能を人と書くこと、やり直しでは前の sync-docs のコミットが変えたファイルを直さないことを足した。
- tech-debt(`docs/tech-debt/README.md:404-405`): 2 行。404 は、集めた印が文書の分割計画の本文にしか残らず、「人が要らないと言った」と「follow-up がなかった」と leader の `SYNC_DOCS: skipped` の控えが記憶にしか残らないこと(self-review R2-2・R2-3)。405 は、凍らせるファイルの引き方が macOS の awk でしか確かめられておらず、`git show --name-only` と `git status --porcelain` の引用が空白のあるパスで分かれうること、`log.showSignature` は未確認であること(R3-1)。
- 変えなかったもの(sync-docs の確かめ、3 回): AGENTS.md の地図、`docs/quality/quality-gates.md` とその template の写し、`definition-of-done.md`、recipes、`agent-messaging.md`、`subagent-policy.md`、`model-routing.md`、仕様 FR-7 の日付つきの注。

## 8. テスト

- 新しい 2 files が 30 本(12 + 18)。`prompts_leader_syncdocs_test.go` は雛形を `RenderRolePrompt` で描いて文を見る。`skill_doc_followup_test.go` は 4 つの写しの SKILL.md を実行時に読む(embed でないので、`go test -overlay` では差し替わらない)。比べ方は、空白と改行を除いて、順を位置で見る。折り返しを変えても通る。定数(コミットのメッセージ、2 つのコマンド、report の接頭辞、戻し方、節の名前)は `prompts_leader_syncdocs_test.go` が持ち、同じ package の skill のテストも使うので、雛形と skill の両方に同じ文を固定する。
- 文だけでなく、コマンドを実際に走らせるテストが 3 つ:
  - `…SyncDocsFreezeFindsTheFixedMessage`(`prompts_leader_syncdocs_test.go:451`): 使い捨ての repo に sync-docs のコミットと、本文の行にメッセージがあるコミット、件名にタブのあるコミット、前方一致の slug(`a-b`)のコミット、origin の上のコミットを作り、雛形の `git log --format='%H %s' … | awk …` と `git show --name-only --format=` を `sh -c` で走らせる。`grep.patternType` を basic と fixed にして回す。
  - `…SyncDocsStatusListsEachNewFile`(`:294`): 使い捨ての repo に、新しいディレクトリの下の予約したファイル、予約しないファイル、ignore されるファイルを置き、`status.showUntrackedFiles` を `normal` と `no` にして、短い形が `?? docs/new/` の 1 行か何も出さないこと、`--untracked-files=all` が 1 行ずつ出して ignore を出さないこと、`reservePathsOverlap` で予約の中と外に分かれること、1 つずつ消すと何も出ないことを見る。消すのは Go の `os.Remove` で、leader が打つ shell のコマンドではない。
  - `…SyncDocsHandsTheDocMaintainerItsLimits`(`:186`): `featureLeaderTask` が `- 機能:` と `- 予約したパス:` の行を今も書くことと、`reservePathsOverlap` の読み方が雛形の説明と合うこと。雛形全体に `check-sync.sh` が出ない否定の検査もここにある。
- ほかのテストの要点: `TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate`(`:307`)は雛形と skill の名前が同じことを見る。`TestDirectorSkill_…MarksAGatheredSplitPlan…`(`:780`)は分割計画の読み手と `PlanDigest` が印の行を残し、覆うことを見る。`TestDirectorSkill_ResultItemCarries…`(`:523`)は `ValidateInboxNote` を実際に呼ぶ。
- AC7: `prompts_test.go`・`prompts_verification_test.go`・`prompts_leader_plan_test.go`・`director_skill_sync_test.go`・`org_skill_send_sync_test.go` の 5 files は `git diff origin/main...HEAD` が 0 行で、92 本が通った(test の report)。この walkthrough の作成時にも、5 files の diff が空であることを確かめた。
- 変異(test の report。雛形は `go test -overlay`、skill は `git archive HEAD` の写しで test binary を 1 回 build して SKILL.md を 1 つずつ変えた。追跡しているファイルは変えていない)。12 個(cycle 1)→ 21 個(cycle 2)→ 27 個(cycle 3)と増え、cycle 3 は 26 個が落ち、1 個が残った。代表:

| 変異 | 落ちたテスト |
|---|---|
| 手順 9 の sync-docs の段落を手順 10 の終わりに移す | 雛形の新しいテスト 12 本すべてと `TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate`(cycle 2 の数) |
| 「すべて戻す」の文を消す、または「予約の外の分だけ戻す」に変える | `…SyncDocsUndoesEveryChangeOrNone` |
| `--grep` の形に戻す(テストの定数と否定の固定も止めた版では、走らせる行だけが落とせるかを見た) | 4 本。定数も止めた版は `…FreezeFindsTheFixedMessage` の走らせる行が、`grep.patternType=basic` で本文の行のコミットを含む 3 件を拾って落ちた |
| 手順 14 の例から `DOC_FOLLOWUP:` の 2 行を消す | `…ResultAndPRBodyCarryTheDocFollowups` |
| `/director` の始める条件から `closed` を落とす、印の文を消す、「もう集めた分割計画」の箇条を消す | 1〜2 本ずつ |
| sync-docs の段落の `--untracked-files=all` の 5 か所のどれかを短い形に戻す | 2〜9 本(`…StatusListsEachNewFile` の否定の検査を含む) |
| 呼ぶ前の確かめの 2 文、または `PLAN:` の句を消す | `…ChecksForLeftoversBeforeTheCall` |
| 手順 6 のテストと fixture を予約する文を消す、F8 より前の言い方に戻す、テストのファイルの名前を足す | `…ReservesTheTestsThatPinTheDocs` |
| 手順 6 の終わりに `tests/test-check-sync.sh` と `internal/org/testdata/` の名前を足す(D6f) | 落ちない(残った) |

  新しいテストは変更の前の版でも落ちた: cycle 1 は 26 本、cycle 2 は 29 本が `origin/main` の雛形と skill で落ちた。cycle 3 の手順 6 のテスト 1 本は、`origin/main`・147e84bc・44694f5b の director の skill で落ちた。
- coverage は測っていない(本番の Go のコードに変更がないため)。

## 9. 直しの経過

- 計画: 1e4a4bca、承認 b8ef4d51(自動承認)。consult の判定は「直してから進める」。直したのは、置き場所を手順 10 の先頭から手順 9 の末尾にしたこと(D1)と、doc-maintainer に insight event を付けさせず report のパスと follow-up の節を名指すこと(D2b)。plan advisory(codex)は HIGH 1 件(写しの組の片方だけが予約の中にあると、予約の外だけを戻したとき写しの検査が gate で落ちる)と MEDIUM 1 件(follow-up を `resolve --note` に入れると 1 行・500 字で断られる)を出し、D2・D2b・D4・AC8 に反映した。
- スライス: S1 2d480eb4(雛形、テスト 6 本)、S2 4a437059(`/org` skill、テスト 6 本)、S3 44275a04(`/director` skill、仕様、README、テスト 4 本)。計画への記録は 8ebb90b1、f9234b39。
- パイプライン 1 回目:
  - self-review run 1(117ab006、HEAD f9234b39): MEDIUM 2、LOW 6、Merge 可。M1 は、sync-docs の変更で落ちた gate が、やり直しで同じ変更が入り直して繰り返されうること(手順 12 の「2 回続けたら BLOCKED」は `GATE: fail` を数えない)。M2 は、文書の機能の予約が写しの組の一部しか覆わないことがあること。L1 は `git restore <パス>` がステージ済みの変更を戻さないこと、L2 は呼び出しの失敗の枝が戻さずに進むこと、L3 は戻した変更の控えが leader の記憶にしかないこと、L4 は director の集め始めと止め方、L5 は leader が書いた URL を director の shell に入れること、L6 は `/org` skill の「完了条件」に sync-docs がないこと。
    - 計画の本文を c2370794 で書き換え(D2・D2c・D3・D4・AC6)。F1 dadd4daa(M1、M2 の leader 側、L1、L2、L3、L6)、F2 738e7bf8(M2 の director 側、L4、L5、凍らせたファイルの見張り)、8556265e(inline。仕様の注と README)。
  - self-review run 2(c2417dc9、HEAD b95f8d8c): LOW 3。run 1 の 8 件のうち 7 件が直り、L4 は一部。R2-1 は `git log --grep` が `grep.patternType=fixed` で 0 件になり本文の行にも当たること、R2-2 は人が「要らない」と言った分割計画と follow-up のなかった分割計画の記録がないこと、R2-3 は失敗の枝の戻し方の指し先が合わず、飛ばした記録が記憶にしかないこと。F3 7f33188b が R2-1・R2-3 を直した(件名の完全一致)。1286ab79 は計画の D2c の古いコマンドを直し、台帳に 2 行足した。
  - 0dfc49f0: origin/main(0351800d、#290)を merge。重なった 11 files に衝突はなく、雛形・2 つの skill・2 つのテストに手の解消はない(`git show --cc` の出力 0 行、self-review run 3)。
  - verify 初回(ad820f13、HEAD 0dfc49f0)は fail。V1(HIGH)は、`/org` skill の `templates/base/` の写しが meta-repo だけの `scripts/check-sync.sh` を名指しし、`check-template-purity.sh` が落ちること(CI の `verify.yml` も同じ検査で落ちる)。V2(LOW)は、受信箱の文が「PR の URL を `--note` に書いて resolve」のままなこと。見落とした理由は、S2・S3・F1〜F3 の検証コマンドに `run-static-verify.sh` も purity も入っていなかったこと。F4 d8e73710 が、雛形と skill を `scripts/check-skill-sync.sh など` にし、雛形全体に `check-sync.sh` が出ない否定のテストを足し、受信箱の文を `/director` に合わせた。
  - self-review run 3(d4bb1a78、HEAD 3247c0af): LOW 1。R3-1 は台帳 2 行の言い方で、`core.quotePath` で分かれるのは日本語でなく空白のあるパス。feeb1e80 が直した。
  - verify 再実行(91e82463、pass)、test cycle 1(a5e28500、pass、変異 12 個が落ちた)、sync-docs(e5f90219。`/org` skill の要約 2 か所に sync-docs を足した)。
  - cross-review cycle 1(da707a0d): codex の P2 1 件。既定の `git status --porcelain` が新しい未追跡のディレクトリを 1 行にまとめる。ACTION_REQUIRED にし、`cycle-count.json` を 2 に上げた(上限 2)。
- パイプライン 2 回目:
  - F5 4d0f70e4(cross-review #1)が、sync-docs の段落の `git status` 4 か所を `--untracked-files=all` にし、未追跡は一覧のファイルを 1 つずつ消す形にした。F5b aec1381f(consult の勧め)が呼ぶ前の確かめを足し、`--untracked-files=all` の `git status` は 5 か所になった。計画は ef4a15a7 で書き換えて承認し直した。
  - self-review cycle 2(8bd985fb、HEAD ef4a15a7): LOW 2。C2-1 は F5 の前の書き方がテストのコメントと台帳に残ること、C2-2 は呼ぶ前の確かめの振り分けが per-plan の `PLAN:` のヘッダを落とすこと。94c10923(inline。台帳の行)と F6 41468881 が直した。
  - verify cycle 2(b0845cd3、pass。V3 LOW は仕様 FR-4 の注のコマンド)、test cycle 2(147e84bc、pass、変異 21 個が落ちた)、sync-docs(822acf62。V3 を直した)。
  - cross-review cycle 2(308409a7): codex の P2 1 件。director が書く文書だけの機能の予約に、follow-up の文書の文を固定するテストが入らず、文書の org が gate を通れない。ACTION_REQUIRED。上限 2 に届いたので、ユーザーの決めどおり上限を `RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` に上げて回し直した。
- パイプライン 3 回目:
  - F7 44694f5b(cross-review #1)が、`/director` skill の手順 6 にテストと fixture を探して予約に入れ人に言う 2 文を足した。計画は ebc2b64a で書き換えて承認し直した。
  - self-review cycle 3(eea3eb84、HEAD ebc2b64a): LOW 1。C3-1 は F7 の文が読点で 2 通りに読めること。F8 b6a8060a(inline。4 つの写しとテストの 5 files、各 1 行)が直した。
  - verify cycle 3(d751a6e3、pass。V4 LOW は仕様 FR-4 の注にテストの予約がないこと)、test cycle 3(bc00d1d1、pass)、sync-docs(545efea1。V4 を直した)、cross-review の 3 回目(21d6f4f2。codex の所見は 0 件)、AC の checkbox(1015e3c0)。
- consult の助言で入ったもの(計画の Progress が書くもの。回数は数えていない): S3 への申し送り(director は `SYNC_DOCS: skipped` の行も拾う)、F1 の前(凍らせたファイルを doc-maintainer が直したときの見張り)、F5b の勧め。F4 は consult が進めてよいと答えた。

## 10. 確かめたことと確かめていないこと

- 確かめたこと(各 report の記録。この walkthrough では再実行していない):
  - verify cycle 3(HEAD 7f14d8bc): `./scripts/run-static-verify.sh` rc 0(`All verifiers passed.`、golangci-lint `0 issues.`、secret scan clean)、`check-template-purity.sh` rc 0、`check-skill-sync.sh`(14 skill が一致)、`check-sync.sh`(DRIFTED 0、ROOT_ONLY 0)、`git diff --check` rc 0、`go build ./...`・`go vet ./internal/org/`・`gofmt -l internal/org` は出力なし。`/director` skill に `_test.go` と `check-sync.sh` がない。計画の digest が承認の行と同じ。AC1〜AC6・AC8 は met、AC7 は静的な部分だけ。
  - test cycle 3(HEAD d751a6e3): `GOFLAGS="-timeout=30m -count=1" ./scripts/run-test.sh` rc 0(shell 45 本、Go 10 パッケージ、1,762 秒)。絞った回は 122 本(AC7 の 5 files の 92 本と新しい 30 本)がすべて通った。変異 27 個のうち 26 個が落ちた。
  - sync-docs(545efea1): `check-template-purity.sh`、`check-skill-sync.sh`、`check-sync.sh`(IDENTICAL 172、DRIFTED 0)、`git diff --check`、`go test -count=1 ./internal/org/ -run 'Skill|RenderRolePrompt'`(ok、23.8 秒)。
  - cross-review の 3 回目: `codex exec review --base origin/main`(`-c sandbox_mode=read-only` と `--ignore-rules`)が exit 0 で所見 0。read-only の sandbox で Go のテストは走らせていない。
  - この walkthrough の作成時に確かめたこと: skill 2 つの写し 3 面ずつ(6 組)が `.claude/skills/` の版と `cmp` で一致、`git diff --check origin/main...HEAD` rc 0、`.go` の変更が新しいテスト 2 files だけ、AC7 の 5 files の diff が空。
  - 7f14d8bc(verify cycle 3 の HEAD)以後に変わったのは、記録、計画、insight event と、仕様 FR-4 の注の 1 行だけ(`git diff --name-only 7f14d8bc HEAD`)。雛形、skill、テストは変わっていない。
- 確かめていない・残っていること:
  - 実機の org で、leader がこの手順を走らせること(doc-maintainer を呼ぶ、戻す、コミットする、follow-up を書く)と、director が follow-up を集めて分割計画を人と書くこと。どちらも LLM の動きで、テストが固定するのは文だけ。コマンドを走らせるのは 3 か所(凍らせるファイルの引き方、status の形、予約の読み方)。仕様 FR-4 の印を `[ ]` のままにした理由。
  - Codex の leader が herdr の pane の中で custom agent を呼べるか。計画のリストにあり、呼べなければ飛ばして記録する、と雛形に書いてある。未確認。
  - 戻しのコマンド `git restore --source=HEAD --staged --worktree -- <パス>` と未追跡の削除を走らせる自動のテスト。implementer と self-review が使い捨ての repo で確かめた記録だけがある。
  - 凍らせるファイルの引き方を gawk と mawk で走らせること(手元は macOS の `/usr/bin/awk` だけ。CI は ubuntu で同じテストが走るはずだが、どの awk かは確かめていない)、`log.showSignature` の署名行、空白のあるパスの引用の食い違い(tech-debt 405)。
  - test の report の穴(LOW): 件数の 1 行の RESULT を `validateEscalation` に通す行がない。「2 回目の sync-docs が何も変えない」は文の固定だけ。呼ぶ前の確かめで何か出たときの振り分けは文の固定だけ。手順 6 がこの repo のテストの名前を書かないことの否定の検査は `_test.go` しか見ず、`tests/test-*.sh` や `testdata/` の名前を足しても通る(D6f)。
  - 仕様 FR-4 の注の変更(545efea1)のあとに、`run-static-verify.sh` と `run-test.sh` の全体は打ち直していない。打ったのは sync-docs の絞った確かめ(上)だけ。
  - `internal/org/skill_doc_followup_test.go` は 890 行で、コードレビューの 800 行の目安を越える。tech-debt に、このファイルの行はない(`docs/tech-debt/README.md` に `skill_doc_followup` は出ない)。払い方は未確認。`prompts_leader_syncdocs_test.go` は 717 行。
  - 計画の図解ページ(gitignore の下)と、insight event の中身(18 行)は読んでいない。
  - Known gaps(PR の本文に書く。tech-debt の行):
    - 404 行: 集めた印は、人が進めると言ってから書く文書の分割計画の中にしかない。人が要らないと言った分割計画と follow-up のなかった分割計画は、director の会話の外に記録がなく、compaction や再起動のあとに同じ案を見せうる。leader の `SYNC_DOCS: skipped` の控えも、手順 14 まで leader の記憶にしかない。承認なしには何も merge も start もされない。
    - 405 行: 凍らせるファイルの引き方(`awk`、`git show --name-only`)の確かめの範囲。凍らせたファイルを見落とすと、M1 の往復が戻りうる。払い方は、両側を `-z` で読む形。
