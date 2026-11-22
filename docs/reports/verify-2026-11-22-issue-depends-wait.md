# Verify report: issue-depends-wait

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364。承認の digest `7b44cea2d6bc` は HEAD の本文から計算した値と一致)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 2 回目(`cycle-count.json` は cycle 2、既定の上限 2 の最後の周)。今の判定は下の `## Verdict`。Run 1(cycle 1)は「Run 1」の見出しの下にそのまま残し、見出しを 1 段下げた(Run 1 の判定は `### Verdict`)。所見の ID は Run 1 の V1〜V6 に続けて V7 から振る(台帳 561 行が Run 1 の V3 を指す)
- Scope: cycle 1 の verify(`95d755b2`)から HEAD `792ffd97` までの 6 コミット(`git diff --stat 95d755b2..HEAD` は 18 ファイル、+847/-112)。中身の変更は `ff59a0e1`(cross-review の 2 件の直し)と `792ffd97`(cycle 2 の self-review の C2-L1〜C2-L4 の直し)で、残りの 4 つは report だけのコミット。AC は全体を見直したが、読み直したのはこの 2 つの直しが触れた文と、それが頼る文とコードに限った。テストは実行していない(`/test` の担当。別のタスクの run がマシンを使っている)
- Evidence: `docs/evidence/verify-2026-11-22-issue-depends-wait.log`(git の管理外)の末尾の「Run 2 (cycle 2)」の段。Run 1 の内容は先頭のまま残し、追記の前後で `head -c` の比較が一致することを確かめた。静的検証の全出力、8 写しの md5、digest、JSON のタグ、`binding.ts` の出どころ、規則の語の出現の数、仕様の該当文、台帳 561・562 行の Trigger、skill の hunk の位置を入れた

## Run 2 (cycle 2)

行番号は `.claude/skills/director/SKILL.md` と `.claude/skills/org/SKILL.md` の HEAD `792ffd97` のもの。

### cross-review の 2 件が skill の文で閉じたか

| # | 指摘 | 閉じたか | 確かめたこと |
| --- | --- | --- | --- |
| 1 | 前の試みの merge 済みの PR が、今の分割計画の結びつき((a) に当たる)と組んで `merged` と出る | 閉じた | 頼る先の merge の確かめに (c) が入った(director 752-770): `pr.number` の PR の merge の時刻が `binding.ts` より前でないときだけ数える。コードでは、`binding` はそのブランチの最新の完全な `scope_reserved` で、`ts` はその event の時刻(`mergeorder.go:582`)。`pr` は fork を除いた番号が最大の PR で(`pickMergePR`、`:1108-1119`)、class `merged` はその PR の state から決まる(`:1186-1187`)ので、`merged` の件の `pr.number` は `merged` の根拠の PR を指す。指摘の道(結びつきの org が PR を開く前に disband)では、その PR の merge は結びつきより前なので (c) に落ち、起票と引き継ぎでは聞かずに待つ(794-796)。2 では「記録の前」も「記録の直前」も確かめに当たるときだけ数える(2606-2609) |
| 2 | 別の分割計画の同じ名前のブランチの PR が記録より後に merge されると、follow-up が `ready` になり、頼る先のないまま着手できると出る | 閉じた | 「merge を待つ issue」の 2 は、頼る先の機能のブランチの `ready` の follow-up も確かめに当たるときだけ待ち終えたと数える(2612-2617)。当たらなければ着手を聞かず、固定の note `<ブランチ> の merge は頼る先の merge でない` で done にし、今の印の行で記録し直し、PR の番号とともに人に伝える(2617-2622)。引き継ぎは、この note の done の件を人の答えと読まない(2492-2497)。done と記録し直しのあいだで止まったブランチは follow-up のないブランチとして記録し、頼る先の問いは聞かない(2510-2512、2623-2624)。重なる org のブランチの `ready` は確かめない(2626-2627) |

`ready` の比べ方(`pickFollowupPRs` の `at.Before(added)`、`mergeorder_followup.go:355`)は同じ秒の merge を「後」と数える。(c) の「より前でない」も同じ秒を数えるので、2 か所の向きはそろっている。`binding.ts` は `Org.now`(`spawn.go:411-417`、`.UTC().Format(time.RFC3339)`)で書かれ、skill の例 `2026-11-22T03:04:05Z` の形と合う。

### 依頼の確かめ

| 項目 | 結果 | 根拠 |
| --- | --- | --- |
| 規則の定義が 1 か所 | 成り立つ | `mergedAt` と `binding.ts` は skill の中で (c)(director 752-757)だけに出る。(a)・(b)・(c) の条件は director の 1 にだけ書かれ、ほかの箇所は「頼る先の merge の確かめ」の名前で呼ぶ(728、731、848、2501、2513、2552、2608、2615、org 2438)。`TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling` が (c) の文、gh のコマンド、読めないときの文、伝える回数の文を含む 8 つの断片を director の 4 写しで 1 回ずつと数える。`TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge` は org skill の節に `(a)`・`(b)`・`(c)` がないことを確かめる。repo 全体を `git grep` しても、規則を言い直す文は仕様の日付つきの項目(下の V7)のほかにない |
| org skill が director の規則を名前で呼ぶ | 成り立つ | org 2436-2443: 「`/director` skill の頼る先の merge の確かめに当たるときだけ」「規則は同じ skill の「起票と記録」の 1 にだけあり」。`merged` のブランチにも `ready` の follow-up にも掛かると言う(C2-L1 の直し) |
| JSON のキーと gh の引数がコードと合う | 合う | `MergeOrderEntry.PR` は `json:"pr,omitempty"`(`mergeorder.go:263`)、`MergeOrderPR.Number` は `json:"number"`(`:284`)、`MergeOrderBinding.TS` は `json:"ts"`(`:328`)、gh の PR のフィールドは `ghPullRequest.MergedAt` の `json:"mergedAt,omitempty"`(`:750`)で、merge-order 自身も `--json ...,mergedAt` で gh に頼む(`mergeorder_followup.go:62`)。テストの `checkMergeOrderJSONNames` がこの 4 つを構造体のタグと突き合わせる。`gh pr view <番号> --repo github.com/<owner>/<repo>` の並びは skill の「予約外の文書の follow-up」の 2(2801)と同じで、origin が GitHub でなければ gh の pr のコマンドを打たない規則(961-962)にも入る |
| C2-L2 の回復の道(読めない時刻で記録した follow-up) | ある | (c) は、読めない時刻を「数えないが、(c) に当たらないとは決まっていない」とし、2 の done と記録し直しを打たない。起票と引き継ぎではほかの数えないブランチと同じく待ち、4 で記録する(762-766)。その follow-up は記録の前の merge なので `waiting` のままで、ブランチは `merged`。2 は「open の follow-up のうち、ブランチが `merged` なのに STATE が `waiting` の件がある issue も、`ready` の件がなくても、merge と進捗の報告のたびに」確かめる(2594-2597)。頼る先のブランチは記録の直前の merge でも確かめに当たるときだけ数え(2606-2609)、そこで時刻を読み直す(766-768)。読めたら記録の直前に merge 済みと数えて着手を聞く。読めなかったことは、ブランチごとにまだ伝えていないときだけ人に伝える(768-770)。2 の `ready` の道でも、読めなければ done にも記録し直しもせず、着手も聞かない(2624-2626)ので、次の報告でまた読む |
| 仕様の 2026-11-22 の項目が `792ffd97` のあとの skill と合う | 一部ずれる(V7) | 仕様 173 行の末尾の (c) の文は `ff59a0e1` で足したまま。読めない時刻を「数えない」とだけ言い、2 で done と記録し直しを打たないこと、次の報告で読み直すこと、伝えるのは 1 回だけということを書いていない |
| README と AGENTS.md | ずれなし | どちらも起票の待ち、`after:`、`Depends on` を書いていない(`grep` で 0 件)。この PR は Go の本体のコードを変えていない |

### Spec compliance (Run 2)

| Acceptance criterion | Status | cycle 2 で変わったこと |
| --- | --- | --- |
| AC1 | Met | 頼る先の merge の確かめに (c) が加わり、「merge 済みは待たない」の条件が狭くなった(数える場合が減るので、待ちすぎる向き)。待つブランチの決め方(走っている org の `feature:` の行、走っていなければ `<Type>/<slug>`)は変わらない |
| AC2 | Met | 4 と引き継ぎの「まだないブランチを記録して `waiting` で待つ」文は変わらない。引き継ぎの follow-up のないブランチの定義が「open の件も done の件もなく(固定の note の done の件は数えない)」になった(2504-2506) |
| AC3 | Met | 3 は cycle 2 で変わっていない(hunk は 1 と引き継ぎと 2 だけ。evidence の hunk の一覧) |
| AC4 | Met | 例の流れ(`locks` を `--issue` で立て、その PR の merge で `api` の follow-up が `ready` になり、2 で着手を聞く)は、(b) と (c)(結びつきのあとに開いた PR の merge)に当たるので、2 の確かめ直しを通る。例の文は 2 の確かめを名指さないが、「「merge を待つ issue」の 2」を指す |
| AC5 | Met | 静的検証の中で check-sync は PASS(DRIFTED 0)、check-skill-sync は「14 skill(s) in lock-step」。md5 は director の 4 写しが `2e46821f3b4f724cf6b78899916cf4e8`、org の 4 写しが `bd1c9f28a2efd53e074fef32ef57826b` で、それぞれ同じ |
| AC6 | Met | AC6 の「(a) か (b) に当たるときだけ」は、(c) を足しても成り立つ((c) は条件を足すだけ)。(b) の正しい流れ(`--issue` で立てた org の PR の merge)は結びつきより後の merge なので数える。テストの固定は上の表のとおり |
| AC7 | Met | 2 の `- Type:` と 3 の Type の読み方は cycle 2 で変わっていない |

計画の Scope の 1 つ目と 4 つ目の項目は、merge 済みと数える条件を (a) と (b) だけで書いており、`ff59a0e1` の (c) と、2 の `ready` の確かめ直し(done と記録し直し)を含まない。どちらも cross-review の ACTION_REQUIRED の直しで、条件を狭める向きの追加。計画の本文は digest で守られているので直さず、計画の Progress の verify の行に一文で書いた(Run 1 の V4 と同じ扱い)。

### self-review Run 2 の所見の直し(`792ffd97`)

| ID | 直ったか | 確かめたこと |
| --- | --- | --- |
| C2-L1 | 直った | org skill は (a)・(b) を並べず、director の確かめを名前で呼ぶ。`ready` の follow-up にも掛かると言う(org 2436-2443)。テストは org skill の節に `(a)`・`(b)`・`(c)` がないことを確かめる |
| C2-L2 | 直った(推奨と違う道を含む) | (c) の時刻は (a) か (b) に当たる件でだけ読む(753-755)。読めなかったことを伝えるのはブランチごとに 1 回(768-770)。推奨の「1 回読み直す」と「人が「数える」と言えることを伝える」は取らず、2 が報告のたびに読み直す道で回復する(上の表)。gh が失敗し続けるあいだは、すでに merge された頼る先を待ち続け、人には 1 回だけ伝わる |
| C2-L3 | 直った | 引き継ぎの例外が規則の条件の中に入った(2510-2512「2 で done にしたあと記録し直す前に止まったブランチ…でなく」)。「(done の件も含めて)」は「open の件も done の件もなく(…固定の note の done の件は数えない)」になった(2504-2506)。`skill_split_questions_test.go` の固定 1 か所もこの文に合わせた |
| C2-L4 | 直った | `time.Parse` の重ねた確かめを外した。テストのファイルは 790 行で、800 行の目安の手前(V11) |

### Static analysis (Run 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0) | changed scope で golang を選んだ(full scope でも golang だけなので同じ)。hook の shellcheck と `sh -n`、settings.json の jq、Codex の hook の 3 つの guard、check-sync(DRIFTED 0)、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、gofmt(ok)、golangci-lint(0 issues)、branch の secret scan(`2a069b50..792ffd97` clean)がすべて通った |
| `./scripts/plan-visual.sh digest <plan>` | `7b44cea2d6bc` | 計画の `- Approved:` の行と一致。Progress の追記のあとも同じ値 |
| `md5`(skill の 8 写し) | 一致 | AC5 の行 |

### Documentation drift (Run 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| director skill(4 写し) | Yes(INFO V10) | (c) のコマンド、JSON のキー、時刻の形はコードと合う(上の表) |
| org skill(4 写し) | Yes | C2-L1 の直しで director の確かめを名前で呼ぶ。「起票のときに 1 回だけ人に聞く」は Run 1 の V3 のまま(V9) |
| `docs/specs/2026-10-07-org-multi-org-director.md` 173 行 | 一部 No(V7) | `ff59a0e1` の (c) と 2 の確かめ直しは書かれている。`792ffd97` の決めごと(読めない時刻の扱い、done と記録し直しのあいだで止まったブランチは聞かない)がない。Run 1 の V1・V2 は `f6d1cd35` で直った(下の表) |
| `docs/tech-debt/README.md` | No(V8・V9) | 同じ PR が足した 561・562 行の Trigger が cycle 2 の直しで当たり、払っても注記してもいない。cycle 2 の self-review の Tech debt identified の 2 行は台帳にまだ入っていない(`/sync-docs` への引き継ぎ) |
| `ralph org followup add --help`、org skill のコマンド表 | Yes(Run 1 の V5 のまま) | cycle 2 で変わっていない |
| `README.md`、`AGENTS.md`、`internal/org/prompts/director.md` | Yes | 起票の待ちを書いていない。director.md の gh への言及は「ralph と gh の操作だけ」の 1 文で、新しい `gh pr view` もその中に入る |
| 計画の Progress checklist | 補った | verify の行に Run 2 の一文を足した(本文は直していない) |

台帳の Trigger は、cycle 2 で変わったファイル(director と org の skill、仕様、`skill_issue_depends_test.go`、`skill_split_questions_test.go`)と、新しく頼るようになったもの(`gh pr view --json mergedAt`、`binding.ts`、`scope_reserved` の書き手)の語で、開いた行の Trigger の列を python で読んで調べた(取り消し線の文は除いた)。当たったのは 561 行の (d) と 562 行の (1)・(2) だけ。426 行(`skill_split_questions_test.go` に次のテストを足すか、節の読み手を変える)は、`792ffd97` が断片の文を 1 つ直しただけなので当たらない。427 行も同じ。561 行の本体の Trigger(「merge を待つ issue」の 3 の次の変更)は、cycle 2 の hunk が 3 に入っていないので当たらない。`reserveAgain` などを名指す行(186、188、195、196、225)は、Go のコードを変えていないので当たらない。

### Findings (Run 2)

| ID | Severity | Finding | Evidence | Hand-off |
| --- | --- | --- | --- | --- |
| V7 | LOW | 仕様の 2026-11-22 の項目の (c) の文は「`gh pr view <番号> --json mergedAt` で読み、読めなければ数えない」とだけ言い、続く 2 の文は「当たらなければ…done にして同じブランチで記録し直し」と言う。`792ffd97` のあとの skill では、読めない時刻は (c) に当たらないとは決まっていないので、2 は done も記録し直しも打たず、着手も聞かず、報告のたびに読み直す。起票と引き継ぎでは記録して待ち、2 が読み直して数える。読めなかったことはブランチごとに 1 回だけ伝える。時刻は (a) か (b) に当たる件でだけ読む。引き継ぎは、done と記録し直しのあいだで止まったブランチでは頼る先の問いを聞かない。仕様だけを読む人は、読めない時刻で done と記録し直しを打つと読みうる | 仕様 173 行の末尾(evidence に該当の文)、director 752-770・2510-2512・2623-2626 | `/sync-docs`: 173 行の (c) の文に、読めないときの扱い(2 は done も記録し直しもしない、読み直して回復、伝えるのは 1 回)と、引き継ぎの例外を足す |
| V8 | LOW | 台帳 562 行(この PR の `f6d1cd35` が足した)の Trigger が、そのあとの cycle 2 の直しで当たった。(1)「`skill_issue_depends_test.go` に次のテストを足す」: `ff59a0e1` が `TestDirectorSkill_StartCheckCountsAReadyDependencyBranchOnlyByTheDependencyCheck` を足した。(2)「`directorDependsAskingSentences` か、それが読む箇所の次の変更」: `ff59a0e1` と `792ffd97` が、それが読む引き継ぎと「merge を待つ issue」の 2 を変えた。どちらも払っておらず(`TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` の後半と `.git` の stat はそのまま、照合は「`feature`だけが合」と「聞く」のまま)、行に注記もない | `skill_issue_depends_test.go:418-440`・`:121-132`、`git diff 95d755b2..HEAD` の `+func Test` 1 つ、evidence の 562 行の Trigger | `/sync-docs`: 562 行に日付つきの注記を足す(cycle 2 の直しで (1)・(2) が当たったが、上限の最後の周なので払っていない、と次の Trigger)。払うならテストの変更になり、記録だけの修正の例外に入らない |
| V9 | LOW | 台帳 561 行の (d) の Trigger「`/org` skill の `followup` の節の次の変更: 「引き継ぎで follow-up のないブランチに 1 回」を足す」が、`792ffd97`(C2-L1 が同じ節の同じ文を書き直した)で当たった。org skill は今も「director が起票のときに 1 回だけ人に聞く」で、払っておらず、行に注記もない。テストもこの文をそのまま固定する | org 2442、`skill_issue_depends_test.go:771`、evidence の 561 行の (d) | `/sync-docs`: 561 行の (d) に日付つきの注記を足す(当たったが払っていない、理由は V8 と同じ)。払うなら org skill の 4 写しとテストの固定を 1 つずつ変える |
| V10 | INFO | 「merge を待つ issue」の 2 の「follow-up のあるそのブランチは、`feature` だけが合っても聞かない(…follow-up は記録より後の merge だけを数えるので、前の別の分割計画の同じ名前のブランチの merge では `waiting` のまま)」の理由は、記録より前の merge(`waiting` の件)にだけ当てはまる。頼る先のブランチでは、別の分割計画の PR が記録より後に merge されると `ready` になる。その場合はすぐ次の文(2612-2622)が確かめ直しと done と記録し直しを書き、聞かないという結論は同じなので、動きに食い違いはない | director 2609-2617 | `/sync-docs`: 台帳の行の候補(次にこの文を触るときに、理由を「記録より前の merge では `waiting` のまま、後の merge で `ready` になった件は次の文のとおり確かめ直す」にする) |
| V11 | INFO | `internal/org/skill_issue_depends_test.go` は `95d755b2` の 607 行から 790 行になり、800 行の目安の手前にある。兄弟の `skill_split_questions_test.go`(1205 行)は台帳 426 行の形に入った。562 行の (1) の払い方(後半を削る)は少し縮める | `wc -l`(evidence) | `/sync-docs`: 562 行の注記に行数を書くか、新しい行(次にテストを足すとき、確かめ別にファイルを分ける) |
| V12 | INFO | `binding.ts` は、そのブランチの最新の完全な `scope_reserved` の時刻で、書き手は 3 つある: 予約つきの最初の spawn(`spawn.go:1540`)、`idempotentRespawn` が予約を書き足すとき(`spawn.go:1691`)、`reserveAgain` の補償(`verbs.go:2010`)。self-review Run 2 の範囲外の危険の 2 は `reserveAgain` だけを挙げる。`idempotentRespawn` が merge のあとに書くことがあるかは確かめていない | `mergeorder.go:576-582`、evidence の `scopeReservedEvent(o.now()` の grep | `/sync-docs`: 台帳の行の文を「同じ結びつきの `scope_reserved` が merge のあとに書かれると(`reserveAgain` の補償、`idempotentRespawn` の書き足し)」にする |
| V13 | INFO | 計画の Scope の 1 つ目と 4 つ目の項目は (c) と 2 の確かめ直しを含まない(上の Spec compliance の段落) | 計画 20 行・24 行 | 計画の Progress の verify の行に書いた。`/pr`: 本文に、cross-review の直しで数える条件に (c) を足したことと、Known gaps(V8・V9 の未払いの Trigger、gh が失敗し続けると merge 済みの頼る先を待ち続けること、self-review Run 2 の範囲外の危険)を書く |

### Run 1 の所見の行き先

| ID | 行き先 | HEAD での状態 |
| --- | --- | --- |
| V1 | `f6d1cd35`(sync-docs)で直った | 仕様 173 行は、問いを起票の 1 で 1 回聞くこと、選択肢、答えが `after:` に残ること、引き継ぎで聞くのは follow-up のないブランチだけということを書いている。cycle 2 の分は V7 |
| V2 | `f6d1cd35` で直った | 「次の 2 つの場合だけ」になった。cycle 2 の 2 の done と記録し直しは issue の本文を書き換えないので、この 2 つに足すものはない |
| V3 | 台帳 561 行の (d) に入った | cycle 2 で (d) の Trigger が当たった(V9) |
| V4 | Run 1 で計画の Progress に書いた | 計画 118 行 |
| V5 | 任意のまま | help と表は cycle 2 で変わっていない |
| V6 | `f6d1cd35` で印が付いた | AC1〜AC7 は `[x]`、digest は変わらない |

### `/sync-docs` への引き継ぎ

1. 仕様 173 行の (c) の文を `792ffd97` に合わせる(V7)
2. 台帳 562 行に、Trigger (1)・(2) が cycle 2 で当たったが払っていないことの日付つきの注記(V8)。テストのファイルの行数も書く(V11)
3. 台帳 561 行の (d) に同じ形の注記(V9)
4. self-review Run 2 の Tech debt identified の 2 行を台帳に入れる。1 行目(範囲外の危険): 後から同じ名前のブランチで立つ別の分割計画の org、`binding.ts` を merge より後にする `scope_reserved`(V12 のとおり `reserveAgain` と `idempotentRespawn`)、報告ごとに増える gh の呼び出し(`merged` で (a) か (b) に当たり (c) に落ちるブランチは、新しい PR が開くまで報告のたびに `gh pr view` を打つ)。2 行目(C2-L1〜C2-L4): `792ffd97` で 4 つとも直ったので、行にするなら残りだけ(C2-L2 の推奨のうち取らなかった 2 点)
5. V10 の理由の文を台帳の行の候補に入れる(2 の行か 561 行の追記)

### Coverage gaps (Run 2)

- テストは走らせていない。`ff59a0e1` の新しいテストと `792ffd97` の固定の変更が通るかは `/test` で確かめる
- skill の手順のとおりに director が (c) の時刻を読み、2 で done と記録し直しを打つかは、実機の director で確かめていない。gh の `--jq .mergedAt` が merge されていない PR で何を出すかも確かめていない(class が `merged` の件だけで読むので、ふつうは当たらない)
- gh が失敗し続けると、すでに merge された頼る先を待ち続け、人には 1 回だけ伝わる。人が「数える」と言える道は skill に書かれていない(C2-L2 の推奨のうち取らなかった点)。PR の Known gaps に書く
- 決定的な検査で足りないもの: 台帳の行が同じ PR のあとのコミットで Trigger に当たったことを見つける仕組みがなく、V8・V9 は人の照合で見つけた(台帳 558 行の Trigger の候補を出す script の提案と同じ穴)

## Verdict

- Result: pass(LOW 3、INFO 4。どれも merge を止めない)
- Verified: cross-review の 2 件が skill の文で閉じたこと、規則の定義が director の 1 にだけあり org skill は名前で呼ぶこと、skill が名指す JSON のキー(`pr`、`number`、`ts`)と gh の `mergedAt` がコードと合うこと、`binding.ts` の形と同じ秒の向き、読めない時刻で記録した follow-up を 2 が報告のたびに読み直す道、self-review Run 2 の C2-L1〜C2-L4 の直し、AC1〜AC7 が cycle 2 のあとも Met、静的検証 rc=0、skill の 8 写しがそれぞれ同じ、計画の digest の一致
- Partially verified: 仕様の 2026-11-22 の項目(`ff59a0e1` の分は合い、`792ffd97` の分がない。V7)
- Not verified: テストの実行(`/test` の担当)、実機の director の動き、`idempotentRespawn` が merge のあとに `scope_reserved` を書くことがあるか(V12)

---

## Run 1 (cycle 1、履歴)

> 以下は cycle 1(HEAD `03ced266`)の report をそのまま残したもの。見出しだけ 1 段下げた。

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364。承認の digest `7b44cea2d6bc` は今の本文から計算した値と一致)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `2a069b50` から HEAD `03ced266` までの `git diff origin/main...HEAD`(14 ファイル)。self-review の対象(`2407bb35` まで)に、そのあとの直し `03ced266`(M1・L1・L2・L3・L5)を加えた範囲。テストは実行していない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-22-issue-depends-wait.log`(git の管理外。静的検証の全出力、4 写しの md5、digest、コードの名前の grep、既存テストの行、仕様の該当文)

### Spec compliance

行番号は `.claude/skills/director/SKILL.md` と `.claude/skills/org/SKILL.md` の HEAD のもの。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 「起票と記録」の 1 が、重なる org のブランチに加えて頼る先ごとに、走っている org のブランチか分割計画の `<Type>/<slug>` を待つと言う(merge 済みは待たない)。テストで固定 | Met | director 704-781。走っている org は `feature:` の行と `issue:` の行が (a) か (b) に当たる org だけで、slug だけ合う org は走っていない扱い。走っていなければ `<Type>/<slug>`(`- Type:` がなければ `feat/<slug>`)。待たないのは確かめで数えたときと、人が「この merge と数える」と答えたときだけ。ブランチ名はコードの `feature.go:449`(`Type + "/" + Slug`)、既定の Type は `split.go:99`・`:718` の `feat` と合う。固定は `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn`(1 の fragment、問いの行のバイト一致、`defaultSplitFeatureType`、`MergeClassMerged`) |
| AC2: 同じ節の 4 と引き継ぎが、まだないブランチの follow-up を記録し `waiting` のまま待つと言う | Met | director 809-838(4)、2516-2528(引き継ぎ)。コードでは `validateFollowupBranch`(`followup.go:239`)が形だけを見て、CLI の `followup add`(`internal/cli/org_followup.go:84-105`)は検査のあと `store.Add` を呼ぶだけで git も gh も読まない。PR のないブランチは `mergeorder_followup.go:315` で `waiting`、DETAIL は `<branch> has no PR yet: ...`。既存テスト `mergeorder_followup_test.go:197` と `internal/cli/org_merge_order_followup_test.go:156` が固定済み |
| AC3: 「merge を待つ issue」の 3 が、Type と slug を変えないこと、変えたら待つ issue を移すことを言う | Met | director 2658-2674。固定は `TestDirectorSkill_RewriteKeepsTheTypeAndSlugOrMovesTheWaitingIssues`。移す issue の選び方が広い点は self-review の L4 で、台帳へ回す |
| AC4: #364 の例で、待つ側の `after:` に頼る先の `<Type>/<slug>` が入り、頼る先の merge で follow-up が `ready` になる流れを追える | Met | director 843-888。印の行は issue ごとの見出しの下に枠 1 つずつ。`locks` を同じ slug と Type と `--issue` で立て、その PR の merge で `api` の follow-up が `ready` になる流れと、前の別の分割計画の同名ブランチがあるときの問いと答えごとの結果を書いている。`ready` への移り方はコードの「記録より後に merge された PR があれば ready」(`mergeorder_followup.go:306-307`)と合う。固定は `TestDirectorSkill_IssueDependsExampleFollowsAFiledDependency`(印の行の正規表現、`ValidateFollowupAdd`、`#` のあとの番号が #364 だけ) |
| AC5: `check-sync.sh` と `check-skill-sync.sh` が通り、director skill の 4 写しが同じ | Met | 静的検証の中で check-sync は PASS(DRIFTED 0)、check-skill-sync は「14 skill(s) in lock-step」。md5 は director の 4 写しが `7bd283f9ccb33ec533e7746482b5a2d1`、org の 4 写しが `f6fdd1b5bed6a4c24f42687a293dc2fa` で、それぞれ同じ |
| AC6: 前の別の分割計画の同名ブランチが merge 済みでも、`binding` の (a) か (b) に当たるときだけ merge 済みと数え、`feature` だけ合うときは人に聞く。(b) も数える。テストで固定し、JSON の `binding` のキーは既存のテストで確かめる | Met | director 731-775 に確かめの規則を 1 回だけ書き、4・引き継ぎ・2・org skill は名前で呼ぶ(規則の文が skill 全体で 1 回だけ出ることを `TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling` が数える)。`feature` だけ合うときは起票の 1 で 1 回聞く(選択肢 2 つ、推奨が先頭、「人への聞き方」の列挙に入った)。引き継ぎと 2 は follow-up のあるブランチでは聞かない(V3・V4 参照)。JSON のキーは既存の `mergeorder_test.go:3233`(`feature`・`org_id`・`split`・`ts`)、`internal/cli/org_start_issue_test.go:391-404` と `mergeorder_issue_test.go:61`(`issue`)が固定し、新しい `checkMergeOrderJSONNames` も構造体のタグと突き合わせる。コード側のタグは `mergeorder.go:238`・`:253`・`:256`・`:279`・`:326`・`:327`・`:340` |
| AC7: 「起票と記録」の 2 が本文に `- Type:` を書くと言い、「merge を待つ issue」の 3 が本文の `- Type:` から読む | Met | director 782-796(2、分割計画になければ `- Type: feat`)、2658-2661(3、本文の行、なければ分割計画の機能の値、それもなければ `feat`)。どちらも fragment で固定 |

### self-review の直し(`03ced266`)の確かめ

| ID | 直ったか | 確かめたこと |
| --- | --- | --- |
| M1 | 直った(推奨から 1 点ずれる) | 問いは起票の 1 の「待つブランチを決める前」に 1 回(director 752-775)。選択肢は「新しい merge を待つ」(推奨・先頭)と「この merge と数える」で、問いの文は 1 行の枠。「人への聞き方」の列挙に入った(521-527)。2 は follow-up のあるブランチでは聞かない(2579-2582)。例の最後は答えごとの結果を書く(879-888)。推奨との差は、follow-up のないブランチ(起票の記録が途中で止まった場合)だけ引き継ぎで 1 回聞く例外を足したこと(2482-2496)。答えは `after:` に残るので同じブランチで 2 回は聞かない |
| L1 | 直った | 「頼る先の merge の確かめ」の名前で規則を 1 か所に置き、ほかは名前で呼ぶ。(b) の番号の読み方(待つ側の `頼られる issue:` の行、起票の途中は 3 で出た番号)、起票していない機能は (a) だけ、slug だけ合う走っている org の扱い、頼る先でない同じ分割計画の機能に (a) を使うことを書いた |
| L2 | 直った | 番号を `<locks の番号>` の形にし、印の行を見出しつきの枠 2 つに分けた。テストが `#` のあとの番号を #364 だけに限る |
| L3 | 直った | org skill 2436-2442 が (a)・(b) と「起票のときに 1 回だけ人に聞く」を言う(言い回しの差は V3) |
| L5 | 直った | 引き継ぎの知らせは「まだ人に伝えていないものだけを 1 回」で、頼る先の issue の番号と、頼る先が `after: none` なら人が着手を決めるまで進まないことを言う(2519-2528)。起票の 3 も番号と同じ注意を言う(802-808) |
| L4・L6 | 予定どおり未修正 | 3 の移す手順(2664-2674)とテストの後半(`skill_issue_depends_test.go:330-352`)は self-review の時点のまま。`/sync-docs` で台帳に入れる |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0) | changed scope で golang を選んだ。`./scripts/detect-languages.sh`(full scope の選び方)も golang だけを返すので、full scope でも同じ pack になる。hook の shellcheck と `sh -n`、settings.json の jq、Codex の hook の 3 つの guard、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、gofmt(ok)、golangci-lint(0 issues)、branch の secret scan(`2a069b50..03ced266` clean)がすべて通った |
| `./scripts/plan-visual.sh digest <plan>` | `7b44cea2d6bc` | 計画の `- Approved:` の行と一致。計画の本文は承認のあと変わっていない |
| `md5`(skill の 4 写し) | 一致 | director、org とも 4 写しが同じ値(AC5 の行) |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| director skill(4 写し) | Yes | skill が引くコマンド名、JSON のキー(`entries`・`branch`・`class`・`binding.split`・`binding.feature`・`binding.issue`)、class `merged`、STATE `waiting`、DETAIL `has no PR yet`、`issue: #<番号>` と `feature:` の行(`internal/cli/org.go:2322`・`:2324`)、`followup list` の列名 `AFTER_BRANCH`(`org_followup.go:267`)、既定の Type `feat` は、すべてコードと合う |
| org skill(4 写し) | Yes(INFO V3) | 「merge を待つ issue」の例外の文は director と同じ (a)・(b) を言う |
| `docs/specs/2026-10-07-org-multi-org-director.md` 173 行(2026-11-22 の項目) | No(V1・V2) | `03ced266` より前に書いた文のまま |
| `ralph org followup add --help`、org skill のコマンド表の `followup add` の行 | Yes(INFO V5) | どちらも「merge 済みのブランチは follow-up にしない」という一般の規則で、例外は org skill の節に書いてある。この PR の前から、答えの記録の例外も表と help にはなかった |
| `README.md`、`AGENTS.md` | Yes | どちらも起票の手順と `after:` を書いていない。この PR は Go のコードを変えず、AGENTS.md はテストのファイルを挙げない |
| `internal/org/prompts/director.md` | Yes | 起票と follow-up の文はない |
| `docs/tech-debt/README.md` | Yes | Trigger の列を python で読んで調べ、この diff で満たす行はなかった。行 426(`skill_split_questions_test.go` に次のテストを足すか、節の読み手を変える)と行 427(同じファイルに次のテストを足す)は、この PR が fragment とコメントだけを直し、新しいテストを新しいファイルに置いたので満たさない。行 395 のファイルは触っていない。行 480(leader のタスクの `Depends on`)は leader のタスクを変えていないので当たらない。行 470 は解決済み。self-review の L4・L6 は `/sync-docs` で足す |
| 計画の Progress checklist | 補った(V4) | `03ced266` の直しが計画の Scope の文からずれたことの記録がなかったので、verify の行に短く書いた |

### Findings

| ID | Severity | Finding | Evidence | Hand-off |
| --- | --- | --- | --- | --- |
| V1 | LOW | 仕様の 2026-11-22 の項目は「`feature` だけが合うときは人に聞く。引き継ぎと着手を聞く前の判定(「merge を待つ issue」)も同じ数え方にした」と書き、`03ced266` の決めごとを持たない。今の skill は、起票の 1 で 1 回だけ 2 つの選択肢(「新しい merge を待つ」が推奨)で聞き、答えは `after:` に残り、follow-up のあるブランチは引き継ぎと 2 で聞かない。聞くのは follow-up のないブランチ(起票の記録が途中で止まった場合)だけで、引き継ぎで 1 回。今の文は、報告のたびに聞くとも読める | 仕様 173 行(evidence に該当の 2 文)、director 752-775・2482-2496・2579-2582 | `/sync-docs`: 173 行の項目に、問いを聞く場所と回数、選択肢、答えの残り方、引き継ぎの例外を足す |
| V2 | LOW | 同じ項目の「この場合だけは、今までどおり director が印の行を書き換える」(slug か Type を変えたとき)と、計画の Assumptions が挙げる例外 1 つが、`03ced266` で増えた 2 つ目の書き換えを含まない。引き継ぎで「この merge と数える」と答えると、そのブランチを `gh issue edit` で `after:` から除く | 仕様 173 行、director 2489-2496、計画 41 行 | `/sync-docs`: 仕様の「この場合だけ」を 2 つの場合に直す。`/pr`: Known gaps に 2 つ目の書き換えの経路を書く |
| V3 | INFO | 問いを聞く場所の書き方が 3 か所で少し違う。org skill は「director が起票のときに 1 回だけ人に聞く」、director の 1 は「例外は 1 つで、引き継ぎが follow-up のないブランチを記録するときに聞くことがある」、2 は「引き継ぎのとおり、記録の前に 1 回聞く」と自分の中でも聞き、「人への聞き方」は引き継ぎだけを挙げる。動作は同じ条件(follow-up のないブランチ、ブランチごとに 1 回)で、食い違いはない | org 2441-2442、director 523-524・770-773・2584-2587 | 任意。直すなら org skill の文に「記録が途中で止まったときは引き継ぎでも 1 回」を足す |
| V4 | INFO | 計画の Progress に `03ced266` の行がなく、M1 の直しが Scope の 4 つ目の項目(引き継ぎと 2 でも `feature` だけ合えば人に聞く)からずれたことが記録されていなかった。計画の本文は digest で守られているので本文は直さない | 計画 24 行・112-115 行 | verify の Progress の行に一文で書いた。`/pr` の本文にも書く |
| V5 | INFO | `ralph org followup add --help` の「a branch whose PR is already merged needs no follow-up」と、org skill のコマンド表の「PR がもう merge されたブランチは follow-up にせず」は一般の規則で、頼る先の同名ブランチの例外を書かない。例外は org skill の節にあり、計画の Progress も help は直さなくてよいと記録している | `internal/cli/org_followup.go:74-75`、org 399 | 任意 |
| V6 | INFO | 計画の受け入れ条件の印(AC1〜AC7)がすべて `[ ]` のまま。この report では 7 つとも Met | 計画 63-69 行 | `/pr` の前に付ける(印を付けても digest は変わらない) |

### Observational checks

- 振る舞いの観察はしていない。この PR は skill の文とテストだけを変え、Go の本体のコードは変えていない(`git diff origin/main...HEAD` の Go のファイルは `internal/org/skill_issue_depends_test.go` と `internal/org/skill_split_questions_test.go` だけ)
- skill の該当の節の文は evidence の代わりに上の行番号で示した

### Coverage gaps

- テストは走らせていない。新しい 7 本と直した fragment が通るかは `/test` で確かめる
- director が skill の手順のとおりに 1 回だけ聞くか、`after:` が実際の起票で頼る先のブランチを持つかは、実機の director で確かめていない(skill は LLM が読む手順で、決定的な検査はテストの fragment だけ)
- (b) の `binding.issue` が実際の `ralph org start --plan --issue` で記録されることは #365 の単体テストが固定している。実機の確かめはない
- 決定的な検査で足りないもの: 仕様の日付つきの項目を固定するテストはなく、V1・V2 はそのため `03ced266` のあとも残った。skill の側は、規則の文が 1 回だけ出ること、step 4 が聞かないこと、引き継ぎと 2 の問いが follow-up のないブランチだけに付くことをテストが数えているので、足すなら org skill の文が「起票のとき」以外の聞く場所を言わないことの扱い(V3)を決めてから、その fragment を 1 つ足すのが最小

### Verdict

- Result: pass(LOW 2、INFO 4。どれも merge を止めない)
- Verified: AC1〜AC7 を skill の文、コードの名前と挙動、既存のテストと照らした。静的検証は rc=0、skill の 4 写しは director と org の両方で同じ、計画の digest は一致、self-review の M1・L1・L2・L3・L5 の直しが入ったこと
- Partially verified: AC4 の流れ(skill の文と、`mergeorder_followup.go` の ready の判定までは確かめた。実機の起票と merge は通していない)
- Not verified: テストの実行(`/test` の担当)、実機の director が問いを 1 回だけ聞くこと
