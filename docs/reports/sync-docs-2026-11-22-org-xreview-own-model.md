# sync-docs report: org-xreview-own-model

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(承認の digest `aa9ca6400c54`。この run の前後とも同じ)。issue #360
- Branch: `fix/org-xreview-own-model`。run 開始時の HEAD は `6b671276`(origin/main の `ccdf6293`、#362・#366・#363 の merge を取り込んだ merge)。`git merge-base origin/main HEAD` は `ccdf6293` で、origin/main は HEAD の祖先
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report(引いた SHA は各 report を含む commit。report の中の `HEAD:` の行の値ではない): self-review `66c7820e`(Merge: yes。MEDIUM 1、LOW 4。直しは `f95a08f9`)、verify `7a1dd7ef`(pass。MEDIUM 1、LOW 3、INFO 4。見た HEAD は `b98b1b96`)、test `a30b6097`(pass。`./scripts/run-test.sh` rc 0。回した木は `7c7cdec5`)
- 見た範囲: `ccdf6293..HEAD`(run 開始時の HEAD で、13 commit と merge 2 つの `b98b1b96`、`6b671276`)。台帳の Trigger の照合は、この差分の 40 ファイルを対象にした
- 制約: `./scripts/run-test.sh` と `./scripts/run-verify.sh` は回していない(依頼の指示)。skill の本文は変えていないので `sync-skills.sh` は要らなかった。一時ファイルは scratchpad に置いた
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff ccdf6293 HEAD` は 40 ファイル、+2,273/-1,837。Go は全体で +1,221/-1,234、テスト以外の Go が +54/-28。雛形は `leader.md` が +58/-80、`reviewer.md` が +116/-208。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary

依頼の項目をすべて片づけた。self-review の L2 と verify の V1〜V5 は、コメント 1 か所、台帳の行の直しと注、仕様の 1 句で閉じた。V1 の残り(CLI が 1 つのときに org の cross-review を聞くか)は製品の判断なので、tech-debt の新しい行にした(556 行目)。

台帳の注は 13 行に足した。verify が挙げた 10 行と、R1 の implementer が挙げた 202 の (g) に加え、sweep で 88 の (7) と 242 の (o)(p) が見つかった。この 3 行は handoff にも verify にもなかった。

`6b671276` の merge は `/verify` と `/test` のあとに入ったので、両方が見ていない木になる。衝突を直した director の skill 4 写しと tech-debt は、`cmp`、`check-sync.sh`、`check-skill-sync.sh`、衝突マーカーの検索、org のテストの一部(下の「確かめたもの」)で見て、食い違いはなかった。取り込んだ木での `./scripts/run-test.sh` は回していないので、回すかどうかは `/pr` の前に lead が決める。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `internal/cli/org_runtime_env_test.go` | `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` のコメントが、reviewer 座席の cross-review は TASK から implementer の driver を受ける、と書いたままだった。`paneenv.go`、`herdr_test.go`、`spawn.go` の直し済みのコメントと同じ言い回しにした(座席が自分の driver と model でレビューし、ほかの CLI を呼ばない)。コメントだけで、1 行増えた | `0d44799f` |
| `docs/tech-debt/README.md` | 31 行目(`pick_reviewer`)の Impact、Trigger、Related。13 行に 2026-11-22 の注。343 行目に 2026-11-22 の注。432 行目の注の順と R1 の分。新しい行 1 つ(556 行目)。plan の参照はすべて `docs/plans/active/` で書いた。5 列と、backtick と `~~` の数が偶数なのは `HEAD` と比べて確かめた。変えたのは言及の行だけで、新しい `file:line` はない | `0d44799f` |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 2026-11-22 の項目(145 行目)に、R1 の待ちの決まりと、それを固定するテストの名前を足した | `0d44799f` |
| `docs/plans/active/2026-11-22-org-xreview-own-model.md` | Progress に、R1(`f95a08f9`)、main の取り込み 2 つ、V1(`7c7cdec5`)の sub-bullet と、Sync-docs の行を足した。本文は変えていない。digest は前後とも `aa9ca6400c54` | この report の commit |
| `docs/reports/sync-docs-2026-11-22-org-xreview-own-model.md` | この report | この report の commit |
| `docs/insights/events/2026-11-21-org-xreview-own-model.jsonl` | `sync_docs` の行(cycle 1、pass)。2 行から 3 行(UTC の日付が 11-21 のままなので、同じファイル) | この report の commit |

## 依頼の項目

### L2: コメントと 31 行目

コメントは上の表のとおり。31 行目は、Impact の「The fourth caller」の文を、4 つ目の呼び出し元(reviewer 座席の cross-review)は消えたこと、reviewer の雛形は `pick_reviewer` を呼ばず、leader の TASK も `IMPLEMENTER_DRIVER:` を運ばないので、呼び出し元は上の 3 つに戻ったこと、に書き換え、「the first three callers」を「the three callers」にした。Trigger の括弧は、`/plan` の 11.b と leader の計画助言の 2 つが narrow している、と書き、reviewer 座席の cross-review を呼び出し元から外した。Related の archive の計画の引用の行に、4 つ目の呼び出し元を消した計画(`docs/plans/active/2026-11-22-org-xreview-own-model.md`)を足した。

### V2: Trigger に当たった行の注

注は `2026-11-22: fix/org-xreview-own-model (#360) hit this trigger with ..., but did not pay it: ... Carried over to the next change that meets this trigger` の形で、Trigger のセルの終わりに足した。Related には `docs/plans/active/2026-11-22-org-xreview-own-model.md` を足した。数は、この差分の base `ccdf6293` と、`7c7cdec5`(この branch で最後に Go を変えた commit。sync-docs の前)で `wc -l` を取った。

| 行(最初の語) | 当たった所 | 払わなかった理由 |
|------|-----------|-----------------|
| 88(6 batchable LOW findings)の (7) | `scripts/xreview-helpers.sh` と写しの `pick_reviewer` の doc comment | 判断。Trigger は「そのファイルを別の用で触ったとき」で、コメント 12 行を変えただけ。(7) が言う先頭の行(9〜16 行付近)は変えていない。lead が要らないと思えばこの注は消してよい |
| 161(FR-7 quality gate is prompt-instructed) | cross-review の TASK と leader の小節、`XREVIEW: git_changed` の比べの区間 | hook も verb も足さない。Non-goals が `[org.review].cross_review` と `max_cross_review_cycles` をそのままにする |
| 190(What a leader is told about its reservation)の (a) | `RolePromptVars` に `Driver` と `Model`、leader の雛形の手順 9・12・14 と `### cross-review`(`leader.md` 1,668 → 1,646 行) | `{{SCOPE}}` の文はそのまま。(b) は当たらない |
| 202(Small findings of org-feature-worktree)の (g)、閉じた (f)・(j) | 機能ごとの org の節の手順 9・12・14、座席を確かめる項、`### cross-review` | (g) は `f1b0352e`、(f)(j) は `b0fc4c59` で直し済み。書き直したテストは `assertPhrasesInOrder`、`squashSpace`、`containsPhrase` で読むので、折り返しで落ちない。(k) は `split.go` を触っていないので当たらない |
| 242(Small findings of org-gate cycles 3 and 4)の (o)、(p) | 手順 9 の 1 節(上限に届いたときの HEAD)と、org skill の「org の終わり方」の 1、4、6 | (o) の report の commit の文と、(p) の 2 つの折り返しの行は触っていない。Scope は reviewer の cross-review |
| 267(Three test gaps of the org-director-attach PR) | `/director` skill の `受信箱の処理` の RESULT の `Cross-review:`、ALERT の stall、`### cross-review の上限`(4 写し) | `director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は変えていない。(1)〜(3) は開いたまま |
| 370(`internal/org/verbs.go` is 2277 lines) | `prompts_test.go` 2,727 → 2,787、`spawn.go` 3,122 → 3,127、`prompts_leader_reseat_test.go` 782 → 794、ほか。`verbs.go`、`verbs_test.go`、`spawn_test.go`、`director_skill_sync_test.go` は触っていない | どれも +300 に届かない。分割は Scope の外 |
| 376(The recipes under `templates/base/docs/recipes/`) | recipe 2 つ(2 写しずつ)。`codex-seat-permissions.md` の `docs/evidence/` の引用が 4 から 3(`xreview-live-probe-2026-11-01.md` を、古い cross-review の節と一緒に消した) | 引用は足さず、ほかは言い換えていない。2026-11-21 の注の「4 citations of 4 files」は、この branch の前の数だと注に書いた。`grep -o` で測り直した(2026-11-22)。`isolated-herdr-server.md` は 14 件・7 ファイル・tech-debt の引用 1 件のまま |
| 377(RC1) | `reviewer.md` 599 → 507 行 | agmsg の path は足していない(`reviewer.md` に `{{AGMSG_SCRIPTS}}` は 0 件) |
| 382(RC4) | 機能ごとの org の手順 9・12・14 と `### cross-review` | 手順 4 は触っていない。`/org` skill の分割計画の例も触っていない(コマンド表の 33 行も変わらない) |
| 389(After step 14 of the leader template) | 手順 14 の PR の本文の `Cross-review:` の値と RESULT の例 | 手順 14 の reviewer の spawn と、座席を止める決まりは変えていない |
| 420(Five files went past the 800-line guide) | `internal/config/config.go` 1,219 → 1,221(コメントだけ) | コードは変えていない。ほかの 4 つは触っていない |
| 528(The section 「agmsg の受信箱の運用」) | 立て直しの規則の座席を確かめる項(`f95a08f9` が待ちの決まり、`09367878` が reviewer の QUESTION の枝) | 該当の 1 文は `leader.md` と `prompts_leader_inbox_test.go` に 1 つずつ残る(`grep -c -F` で確かめた)。7 回目 |

R1 の implementer が挙げた行は、432 の (2)、528、202 の (g)、370、377。528・202・370・377 は上のとおり。432 は下の V5。(2) は `f95a08f9` が reviewer の cross-review の手順 1〜3 を変えていないので、もう一度は当たらなかった。

### V3: 343 行目

2026-11-01 の注は、Codex の reviewer が cross-review で相手の呼び出しを前面で最大 20 分走らせるので、`Send` はその間も届かず、leader の雛形と 2 つの skill はその stall を gate のときと同じに扱う、と行を広げていた。2026-11-22 の注に、その広がりを取り戻したこと(相手の呼び出しがなくなり、`Send` が届かないのは gate のあいだだけ)、leader の雛形と skill の待ちの決まり(画面が動いているあいだは待ち、10 分動かず RESULT も BLOCKED も来なければ BLOCKED。画面が動かなくても待つのは gate のときだけ)、2026-11-01 の注は前の状態の記録であること、を書いた。大きな diff を読むあいだに `Send` が時間切れになるかは測っていないとも書いた。Trigger には当たらないので、「Carried over」の句は付けていない。

### V5 と Progress

432 行目の注は、branch の 2026-11-22 の注が #366 の 2026-11-21 の注より前にあった。#366 の注を前に出し、branch の注を最後に置いた(日付順)。branch の注には、`f95a08f9` が `### cross-review` の小節と reviewer の fan-out の節をもう一度変えたので (1)(4)(7) に当たり、(2) は当たらなかったこと、を足した。計画の Progress は、R1、merge 2 つ、V1 の sub-bullet と、Sync-docs の行を足した。

### V1 の残り

`ralph init` の「変更のレビュー」の問いは、CLI が 2 つあるときだけ聞き(`bothOnly`)、CLI が 1 つなら `[standard].cross_review` も `[org.review].cross_review` も false で書く。org の cross-review はもう相手の CLI を要らないので、CLI が 1 つでも聞くかどうかは製品の判断で、この PR では変えない(`7c7cdec5` は説明の文だけ)。tech-debt の 556 行目に 1 行足した。Trigger は `internal/cli/init_settings.go` の次の変更か、CLI が 1 つの repo で org の cross-review を使いたいという報告。直し方は、1 つのときに org の設定だけ聞くか、2 つの設定を分けて聞くかを決め、仕様の 2026-11-02 の項(「1 つのときは false で書く」)と、フォームの 3 つのテストを一緒に変える、と書いた。

### 仕様の 2026-11-22 の項目

R1 の決まりは書かれていなかったので足した。cross-review の返事を待つあいだ、leader は reviewer の画面が動いていれば待ち、10 分動かず RESULT も BLOCKED も来なければ、座席の名前を書いた BLOCKED を上げる。画面が動かなくても待つのは gate のときだけ。前の版は、相手の呼び出しが 20 分まで静かに続くので、cross-review のあいだも gate と同じに待っていた。固定するテストは 3 つ(`TestRenderRolePrompt_Leader_XreviewBlockedFollowsTheGatesAndStallIsBounded`、`TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves`、`TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallOnlyWhileTheScreenMoves`)。

## Trigger の照合で当たらなかった行

台帳の Trigger のセルを、struck の文と日付つきの注を除いて、この差分のファイル名、関数名、節の名前(`reviewer.md`、`leader.md`、`RolePromptVars`、`spawn.go`、`config.go`、`org の終わり方`、`stall`、`fan-out`、`[org.review]`、`XREVIEW` など)で探した。当たりそうに見えて、Trigger の字面に当たらなかった行は次のとおり。

| 行(最初の語) | Trigger | 当たらない理由 |
|------|---------|----------------|
| 160(`tests/test-agent-models.sh` self-test gaps) | model-routing.md の pin の段落 | 変えたのは cross-review の同期の注、effort の段落、テストの一覧。pin の段落は触っていない |
| 182(Two documentation findings of org-stop-all)の (a) | `/org` skill のコマンド表 | 表の 33 行は base と HEAD で同じ。`spawn` 3,026、`send` 4,683、`stop` 1,208、`disband` 1,219、`watch` 3,410 字も同じ |
| 191(Code-shape findings of org-limits-reserve) | `spawnCapacityErr` などの次の変更 | `dryRunSpawn` は (a) の `reserve` の bool の話で出るだけ。この branch が足したのは、雛形の vars の `Driver` と `Model` |
| 369、387(spawn の失敗経路、`reject`) | spawn の saga の失敗経路、`reject` | `spawn.go` の変更は雛形の vars とコメントだけ |
| 405、458(`[org]` の検証) | `config.go` の `[org]` の検証 | コメントだけ。`Load` と検証には触っていない |
| 417(The leader's lookup of earlier sync-docs commits) | 手順 9 の lookup の次の変更 | 手順 9 で変えたのは、上限に届いたときの HEAD の 1 節。lookup ではない |
| 431(The role prompts grew with org-cross-review) | `leader.md` か `reviewer.md` に節を足す変更 | 見出しは足していない(`git diff` の `+#` と `-#` が 0 行)。`leader.md` は 1,668 → 1,646、`reviewer.md` は 599 → 507 に縮んだ |
| 433(The cross-review of a feature org has not run inside a seat) | 最初の実機の run | この branch の S3 が行を直してある(古い sandbox の推測を strike)。注は足していない |
| 470(The `/director` skill ... scaffold) | `/director` skill の次の変更 | 行は RESOLVED(2026-11-13、1 つ目のセルを strike)。Trigger のセルは strike されておらず、前の branch もこの形のまま注を足していない |
| 519(Fourteen small findings of docs/spec-org-reseat)の (b)(d) | `[reviewer]` の検査、`PlanReseat` | 触っていない |
| 521(Test gaps of docs/spec-org-reseat)の (h) | `/org` skill の reseat の節 | 変えたのは「org の終わり方」などで、reseat の節ではない |
| 533(The seat-check item of the re-seat rule) | 項目の中の implementer の slice を送り直す文 | `f95a08f9` と `09367878` が変えたのは、reviewer の待ちと reviewer の QUESTION の枝。slice を送り直す文は触っていない |
| 553 | この branch が足した行 | 自分の編集 |

## 確かめたもの

| コマンド | 結果 |
|----------|------|
| `gofmt -l internal/cli/org_runtime_env_test.go` | 出力なし |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgRuntimeEnv\|Env'` | ok、17.562 s(コメントの直しのあと) |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill\|Xreview\|DocsMatch'` | ok、42.149 s(merge `6b671276` を取り込んだ木で) |
| `./scripts/check-sync.sh` | PASS: all files in sync(TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step |
| org と director の skill の `cmp`(`.claude/skills` と、`.agents/skills`、`templates/base/.claude/skills`、`templates/base/.agents/skills`) | 4 写しとも同じ |
| 衝突マーカー(`<<<<<<<`、`>>>>>>>`、行頭の `=======`)の `git grep` | 0 件 |
| tech-debt の plan 参照(`scripts/verify.local.sh` の `check_tech_debt_plan_refs` と同じ式) | 存在しない参照は 0 件 |
| `docs/tech-debt/README.md` を `git show HEAD:` と比べる | 変わったのは 31、88、161、190、202、242、267、343、370、376、377、382、389、420、432、528 と、足した 556 行目。どれも 5 列(`\|` で割って 7 片)、backtick と `~~` は列ごとに偶数、新しい `file:line` なし |
| `./scripts/plan-visual.sh digest` | `aa9ca6400c54`(計画の `- Approved:` と一致。計画の差分は Progress の 4 行足しだけ) |

## 直していないもの

- verify の V7(AC6 の字面「差分に入らない」)と V8(AC1 の「持たない」と Scope の禁止の 1 文): AC の文は変えない(変えると digest が変わる)。PR の本文で、AC6 はコメントだけの差分で振る舞いを変えずに満たしたと書く
- `./scripts/run-test.sh`: 依頼の指示で回していない。`/test` は `7c7cdec5` の木で回した。`6b671276` の取り込みで入った #363 と #370 の差分(director の skill の 4 写し、tech-debt)は、上の cheap な確かめだけを通した
- 計画の AC の checkbox(`- [ ] AC1` から `AC8`)と `- [ ] PR created`: `/pr` の仕事

## 追記した insight event

```
./scripts/insights-append.sh --slug org-xreview-own-model --flow standard --phase sync_docs --verdict pass --cycle auto --source skill
```

終了コード 0。`git status --short` は、既存の `docs/insights/events/2026-11-21-org-xreview-own-model.jsonl` が `M` になっただけで、新しいファイルはできていない。
