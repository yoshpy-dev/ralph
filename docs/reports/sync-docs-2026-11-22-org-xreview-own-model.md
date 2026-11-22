# sync-docs report: org-xreview-own-model

## Run 2 (cycle 2)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(承認の digest `aa9ca6400c54`。この run の前後とも同じ)。issue #360
- Branch: `fix/org-xreview-own-model`。run 開始時の HEAD は `45e8eadc`。origin/main は `ccdf6293` のままで、`git merge-base origin/main HEAD` も `ccdf6293`(origin/main は HEAD の祖先)
- Pipeline run: 2 回目(`cycle-count.json` は `cycle` 2。上限 2 の最後の周)。Run 1 は下にそのまま残す。今の状態はこの節
- 先行 report(引いた SHA は各 report を含む commit。report の中の `HEAD:` の行の値ではない): cross-review の triage `0cd1a32f`(ACTION_REQUIRED 1。直しは `eff83438` と `7eb0179d`)、self-review Run 2 `d43f5781`(Merge: yes、LOW 3。直しは `7eb0179d`)と Run 3 `65d44585`(Merge: yes、LOW 2。直しは `a607b72d`)、verify Run 2 `5c5ead66`(pass、LOW 1、INFO 3)と Run 3 `2f1b2f15`(pass、INFO 2)、test Run 2 `0f116f03`(pass)と Run 3 `45e8eadc`(pass、`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` rc 0。回した木は `5af1a67b`)
- 見た範囲: Run 1 の commit `82090f46` のあと(`82090f46..45e8eadc`)の変更。中身のあるのは `eff83438`、`7eb0179d`、`18360371`、`a607b72d`、`5af1a67b` の 5 つで、ほかは report と insight event だけ。この 5 つが触れた雛形とテストは `internal/org/prompts/reviewer.md`、`internal/org/prompts_xreview_reviewer_test.go`、`internal/cli/org_runtime_env_test.go`(`t.Fatalf` の文字列 1 つ)、`internal/org/driver/herdr_test.go`(同じく 1 つ)。skill、leader の雛形、仕様、README、`model-routing.md` は触れていない。台帳の Trigger の照合はこの 4 ファイルと、この run が編集した recipe を対象にした
- 制約: `./scripts/run-test.sh` と `./scripts/run-verify.sh` は回していない(依頼の指示)。skill の本文は変えていないので `sync-skills.sh` は要らなかった。一時ファイルは scratchpad に置いた。commit は 1 つで、push はしていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff ccdf6293 HEAD` は 43 ファイル、+3,091/-1,835。Go は全体で +1,325/-1,231、テスト以外の Go が +54/-28(Run 1 と同じ)。雛形は `leader.md` が +58/-80(Run 1 と同じ)、`reviewer.md` が +127/-209(Run 1 は +116/-208)。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

### Summary

依頼の項目をすべて片づけた。台帳は 5 行に手を入れた。432 と 377 には、`eff83438` と `7eb0179d` が Trigger に当たったことを注に足した(verify Run 2 の V9)。370 と 190 は、`wc -l` の数を最後の Go の commit `5af1a67b` に取り直した。新しい 557 行目には、まだ開いている 4 件(`Cross-review: ran (...)` の括弧の重なり、手順 3 の書き方、test の Test gaps 1 と 2)を 1 行にまとめた。

文書の食い違いは、recipe の `codex-seat-permissions.md`(2 写し)に 1 か所あった。「`git diff` が失敗すると incomplete の report を書く」と言い切っていて、`eff83438` のあとは、git の状態が動いていないときだけそうなる。動いていれば report を書かずに `XREVIEW: git_changed` の BLOCKED を返す。1 文ずつ足して直した。

Run 1 の自分の注に誤りが 2 つあった。どちらも直した(下の「Run 1 の注の誤り」)。

### Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/tech-debt/README.md` | 432 行目の 2026-11-22 の注(Run 1 の誤りの直しと cycle 2 の分)、377 行目の注(517 行になったこと)、376 行目の注(recipe を編集したが引用は足さない)、370 行目と 190 行目の `wc -l` の数と commit、新しい 557 行目。5 列、backtick と `~~` の数が偶数、`docs/plans/` の参照が存在することは `git show HEAD:` と比べて確かめた | この report の commit |
| `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md` | 「The cross-review of a codex reviewer seat」の節の incomplete の文に条件(git の状態が動いていない)を足し、動いたときの BLOCKED(`XREVIEW: git_changed`、report なし、上限に数えない)を 1 文足した。2 写しは同じ文で、`cmp` で同じ。`docs/evidence/` の引用は 3 件・3 ファイルのまま | この report の commit |
| `docs/plans/active/2026-11-22-org-xreview-own-model.md` | Progress に、`eff83438`、`7eb0179d`、テストの 3 commit の sub-bullet と、Sync-docs の行の周回 2。本文は変えていない。digest は前後とも `aa9ca6400c54` | この report の commit |
| `docs/reports/sync-docs-2026-11-22-org-xreview-own-model.md` | この節を足した。Run 1 は下にそのまま残した | この report の commit |
| `docs/insights/events/2026-11-22-org-xreview-own-model.jsonl` | `sync_docs` の行(cycle 2、pass)。5 行目。UTC の日付が 11-22 になっているので、Run 1 の行がある 11-21 のファイルではなくこちら | この report の commit |

### 依頼の項目

#### V9: 台帳の 432 と 377

- 432: Run 1 の注は、`f95a08f9` が「手順 1〜3 を変えていないので (2) に当たらなかった」と書いていた。実際は、`f95a08f9` が reviewer の手順 3 に 1 つ箇条を足していて(`git show f95a08f9 -- internal/org/prompts/reviewer.md` で +5 行)、(2) は Run 1 でも当たっていた。注を直し、cycle 2 の分(`eff83438` が手順 2、`7eb0179d` が手順 2、3、5、`18360371` が新しい 2 行)を足した。(2)(`REPORT:` の値を文で確かめている)は払っていない
- 377: Trigger は「the implementer or reviewer role prompt の次の変更」で、2 本とも `reviewer.md` を変えた。注の「599 → 507」を「599 → 517」にし、`7eb0179d` で数えたと書いた。RESULT と BLOCKED の送り方の話ではないので払っていない
- 376: 今回の recipe の編集が Trigger(「any of these five recipes の次の編集」)に当たった。引用は足さず、言い換えもしていないので払っていない。注を足した

#### V10: Progress

計画の Progress の「Implementation started」に sub-bullet を 3 つ足した。`eff83438`(cross-review の ACTION_REQUIRED 1 の直し。AC2 に「`git: same` のとき」という条件が付いたことを含む)、`7eb0179d`(self-review Run 2 の C2-1〜C2-3)、`18360371`・`a607b72d`・`5af1a67b`(テストの行と、そのコメントの直し)。AC の文は変えていない(digest に入る)。

#### V12 と self-review の台帳の案

| 所見 | 行き先 |
|------|--------|
| self-review Run 2 の C2-1、C2-2、C2-3 | `7eb0179d` で直った。行にしていない |
| self-review Run 1 の L1(Run 2 の案の (4))、verify Run 2 の V12 | 括弧の重なり。557 行目の (a) |
| self-review Run 3 の C2a-1、C2a-2(案の (5)、(6)) | `a607b72d` で直った。行にしていない |
| verify Run 2 の V11 | 557 行目の (b) |
| verify Run 3 の V13 | `5af1a67b` で直った(コメントの「before git diff」を、行が固定するものに合わせた)。verify Run 3 のあとに入った commit で、test Run 3 はこの木で回した。行にしていない |
| verify Run 3 の V14 | 直さない。テストのコメント 2 か所(`prompts_xreview_reviewer_test.go` の 288 行目と 829 行目)は、周の番号なしで triage report の所見を指す。2 周目の `/cross-review` が周 1 の節を `## Cycle 1` として残すあいだは読める。残さずに書き直すなら、2 つのコメントが指す先が消える |
| test Run 1 の Test gaps 1 と 2 | 557 行目の (c)、(d) |
| test Run 1 の Test gaps 3(座席の中での実行) | 433 行目が持っている |
| test Run 1 の Test gaps 4(前の版の reviewer 座席に新しい leader が 3 行の TASK を送ると QUESTION になる) | 計画の Assumptions にある扱い(release ノートで reseat を勧める)。台帳にはせず、PR の本文の Known gaps に書く |
| test Run 1 の Test gaps 5 | 穴ではない(dry run では見えない)。記録しない |
| test Run 3 の Test gaps | Run 2 の 1 は閉じた。残りは Run 1 のもの |

#### Run 1 の注の誤り

1. 432 の注の「`f95a08f9` は手順 3 を変えていない」(上)。直した。Run 1 の本文(下の `## Run 1`)の「(2) は `f95a08f9` が reviewer の cross-review の手順 1〜3 を変えていないので、もう一度は当たらなかった」は、書いたときのまま残してある
2. 190 と 370 の注は、`leader.md` の「1,668 → 1,646」を「7c7cdec5 で数えた」と書いていた。7c7cdec5 では 1,638 で、1,646 は main の取り込み(`6b671276`、#363 の 8 行)のあとの数。ほかの数(`prompts_test.go`、`spawn.go` など)は 7c7cdec5 と HEAD で同じ。2 行とも、数えた commit を最後の Go の commit `5af1a67b` に直し、370 は `prompts_xreview_reviewer_test.go` を 1,099 から 1,206、`reviewer.md` を 507 から 517 にした。どちらも +300 には届かない

### Trigger の照合(周回 2)

台帳の Trigger のセルを、struck の文と日付つきの注を除いて、`reviewer.md`、`prompts_xreview_reviewer_test.go`、`org_runtime_env_test.go`、`herdr_test.go`、recipe の名前と、手順の番号(`steps 1 to 3`、`step 5`)、`fan-out`、`git diff`、`git_changed` で探した。

| 行(最初の語) | 当たったか | 理由 |
|------|-----------|------|
| 432(Seven test gaps of org-cross-review) | 当たった((2)) | `eff83438` と `7eb0179d` が手順 2、3、5 を変えた。払っていない(`REPORT:` は文で確かめている)。注を足した |
| 377(RC1) | 当たった | `reviewer.md` の変更。払っていない。注を足した |
| 376(The recipes under `templates/base/docs/recipes/`) | 当たった | recipe の編集。引用は足していない。注を足した |
| 370(`internal/org/verbs.go` is 2277 lines) | 当たらない(数を取り直した) | Trigger は `Send` かリーダーの雛形のテストの変更、または +300。reviewer のテストは 1,206 行で、base の 1,378 より小さい |
| 190(What a leader is told about its reservation) | 当たらない(数を取り直した) | `RolePromptVars` にも leader の雛形にも触れていない |
| 431(The role prompts grew with org-cross-review) | 当たらない | Trigger は節を足す変更。`7eb0179d` が足したのは「座席内 fan-out」の節の箇条で、見出しは足していない(`git diff 82090f46 HEAD` の `+#` と `-#` が 0 行)。`reviewer.md` は 599 → 517 に縮んだまま |
| 161(FR-7 quality gate is prompt-instructed) | 当たらない | Run 1 の注の「git の状態の比べは、座席が読む前と report を書く前に走る」は、2 の道にも当てはまる。verify Run 2 も同じ判断 |
| 432 の (3)(`advisoryShells`) | 当たらない | 新しい 2 行は `advisoryShells(t, false)` を使うだけで、変えていない |
| 441(`driver/herdr.go` grew) | 当たらない | Trigger は `herdr.go` の `--env` の確かめとペイン操作の変更。`herdr_test.go` は `t.Fatalf` の文字列 1 つ |
| 433(The cross-review of a feature org has not run inside a seat) | 当たらない | Trigger は、実機の最初の run か、`Cross-review:` が sandbox や通信の理由で `skipped` か `incomplete` になった PR。どちらでもない。recipe を編集したのは、この行の直し方が名指す場所だというだけ |
| 507 の (b)(A `/cross-review` that writes the triage report again from 1) | 当たらない(この先の話) | 2 周目の `/cross-review` がこの Trigger に当たるかどうかは、周 1 の節を残すか(`## Cycle 1`)で決まる。V14 と同じ |
| 88、202、242、267、382、389、528 ほか Run 1 で注を足した行 | 当たらない | `82090f46` のあとの変更は、`docs/reports/` を除くと 7 ファイル(`reviewer.md`、テスト 3 つ、計画、insight event 2 つ)だけで、leader の雛形、skill、`xreview-helpers.sh`、`config.go`、`spawn.go` には触れていない |

### 確かめたもの

| コマンド | 結果 |
|----------|------|
| `./scripts/check-sync.sh` | PASS: all files in sync(recipe の 2 写しを編集したあと。TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step(skill は編集していない) |
| `cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md` | 同じ |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill\|DocsMatch\|Xreview'` | ok、39.647 s |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Agents\|Readme\|Docs\|Skill'` | ok、1.065 s |
| `docs/tech-debt/README.md` を `git show HEAD:` と比べる | 変わったのは 190、370、376、377、432 と、足した 557 行目。どれも 5 列(`\|` で割って 7 片)、backtick と `~~` は列ごとに偶数、U+FFFD なし |
| tech-debt の plan 参照(`scripts/verify.local.sh` の `check_tech_debt_plan_refs` と同じ式) | 存在しない参照は 0 件 |
| 衝突マーカーの `git grep`(tech-debt と計画) | 0 件 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-22-org-xreview-own-model.md` | `aa9ca6400c54`(計画の `- Approved:` と一致。計画の差分は Progress だけ) |

### 直していないもの、PR の本文に書くもの

- `./scripts/run-test.sh`: 依頼の指示で回していない。Go のファイルはこの run で変えていない。`/test` Run 3 が `5af1a67b` の木で rc 0
- verify の V7(AC6 の字面「差分に入らない」)と V8(AC1 の「持たない」と Scope の禁止の 1 文): Run 1 のまま。AC の文は変えない(digest が変わる)。PR の本文で、AC6 はコメントだけの差分で振る舞いを変えずに満たしたと書く
- AC2: `eff83438` で「`git: same` のとき」という条件が付いた(AC1 の比べを先に通す)。PR の本文で書く
- 計画の AC の checkbox(`- [ ] AC1` から `AC8`)と `- [ ] PR created`: `/pr` の仕事
- PR の本文の Known gaps に載せるもの: 557 行目の (a)〜(d)、556 行目(`ralph init` の問い)、553 行目(leader の計画助言)、test Run 1 の Test gaps 3(座席の中での cross-review の実行はしていない)と 4(前の版の reviewer 座席は新しい leader の 3 行の TASK に QUESTION を返す。release ノートで reseat を勧める)、verify の V7 と V8

### 追記した insight event

```
./scripts/insights-append.sh --slug org-xreview-own-model --flow standard --phase sync_docs --verdict pass --cycle auto --source skill
```

終了コード 0。`git status --short` は、既存の `docs/insights/events/2026-11-22-org-xreview-own-model.jsonl` が `M` になっただけで、新しいファイルはできていない(行は `cycle` 2、pass)。

---

## Run 1 (cycle 1、履歴)

下は Run 1(run 開始時の HEAD は `6b671276`)の report を、書いたときのまま 1 バイトも変えずに残したもの。見出しの段は Run 1 の `#` と `##` のままで、この節より下の `# sync-docs report` が Run 1 の題である。Run 1 の注の誤り 2 つは、上の「Run 1 の注の誤り」で直した。

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
