# Self-review report: org-trial

- Date: 2026-10-20
- Plan: `docs/plans/active/2026-10-20-org-trial.md`(9 段目の試行、パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff origin/main...HEAD`(HEAD ce8dc0a4、25 ファイル、+2465/-10)。記録だけの変更で、コードは含まない。内訳は証拠 `docs/evidence/org-trial-2026-10-20.md`(916 行)、台帳の sanitize 済みコピー `docs/evidence/org-trial-2026-10-20/`(21 ファイル)、仕様 `docs/specs/2026-10-07-org-multi-org-director.md`(FR-12 の注と Open questions)、`docs/tech-debt/README.md`(既存 7 行の更新と新しい 12 行)、計画。diff の品質だけを見た。仕様適合、テストの網羅、文書のずれ、静的解析、テストの実行は `/verify` と `/test` に任せ、このレビューでは走らせていない(下の jq・awk・grep は証拠の再計算で、リポジトリのテストではない)

## Evidence reviewed

- 全文を読んだ: 証拠 916 行、計画 506 行、spec の 3 か所の差分(word-diff)、tech-debt の変えた行と足した行(各行を unescaped の `|` で分け、欄ごとに)、`gate-excerpts.md`、`interventions.log`、`seat-usage.tsv`、`watcher-usage.txt`、`splits/trial.md`、`inbox.jsonl` の e1〜e6、2 つの org report の Event timeline
- 再計算した: 証拠の `sh` ブロック 17 本すべてを worktree の root で実行し、直前の表(12 本)か直後の `text` ブロック(5 本)と比べた。17 本とも一致した(jq 1.8.2、macOS の awk・sort・grep)
- 数字を突き合わせた: spec の FR-12 の注、証拠の表、tech-debt の新しい行の三者(1302 s、109 s と 423 s、213 s と 997 s、$12.81、43 s、9370 s = 2 h 36 min 10 s、30 件の escalation = 23 + 7、osascript の 49 件 = 3 + 11 + 4 + 11 + 20、`sent` 11 件 = 6 + 5 と report の行 4 + 5、U+FFFD の 4 行が「1 つか 2 つ」、所見 F1〜F14 が行 12 本と F9・F10 の既存行 2 本に対応)。コスト表は seat-usage と価格から掛け直した(watcher 1.72、seat の cache write 合計 opus 491,330 と sonnet 382,876)
- 漏えいの grep(追加ファイル全体): `/Users/`、`/home/`、`/var/folders`、herdr の pane id の形(`w<N>:p<M>`、`p_`、`w_`)、UUID、メールの形、`hiroki`・`yoshioka`・`dena`、`sk-`・`ghp_`・`AKIA`・`xox`・`eyJ`・`BEGIN … KEY`、IP とホスト名。どれもなし。残る `/private/tmp/ot/` は sanitize の節が「そのまま残した」と書く 1 種類で、login name は `user-1`、pane は `pane-N`、session は `session-N` に置き換わっている。JSON 側の `pane_id`・`tab_id`・`terminal_id`・`workspace_id`・`agent_session` も全部ラベルで、生の値はない
- コードと照らした(読むだけ): `internal/org/watch.go` の 404 行(`osascript`)と 1557 行(`_ = w.escalateFn`)、`internal/org/report.go` の 180 行、`internal/org/verbs.go` の `truncateForDetails`、`internal/org/feature.go` の `featureLeaderTask`、`internal/org/spawn.go` の `failStepWithNote`・`compensatePane`・`compensateLeave`、`internal/cli/doctor_merge_guarantee.go` の 90 行、閉じた行が名指す 2 つのテスト名、`leader.md` の「agmsg の受信箱の運用」、`session_end_summary.sh` の WIP commit、`scripts/check-sync.sh` の除外一覧、`scripts/archive-plan.sh` の tech-debt の書き換え
- `git diff --check origin/main...HEAD`(下の I2)
- 確かめていない: 価格表の値(pricing ページは取っていない。証拠は「2026-10-20 に読んだ」と書く)、director の notes にだけある数字(各 verb の exit code、pane の画面、load average、trust probe、Orca の tab)、セッション記録の token 合計(コミットされていないので、掛け算と足し算だけ確かめた)、2 本の PR 本体のコード

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (M1) | maintainability(記録の主張) | tech-debt の「leader が図解ページを作って確かめられるか」の行(216 行)の Debt 欄が、同じ欄の中で食い違う。前半は元の文のまま「no run of a claude or codex leader in a herdr pane has done so (the `lint` step included)」で、後半に足した文は「The stage 9 trial met the trigger for claude: both claude leaders said … that the page passed `lint` and was shot」。後半が正しい(e1・e2 の SUMMARY は「図解ページは lint が通り、shot で撮って確かめた」)ので、前半が試行の前の状態を述べる文のまま残っている。もう 1 点、この行の Trigger は「Stage 9 … record whether the page was made, whether `lint` ran …, whether the page was shot, and whether the PNG was read, for each driver」と、記録を求める場所を証拠の側に置くが、証拠 `org-trial-2026-10-20.md` には lint・shot・図解ページの記述がなく(`grep -n -i 'lint\|visual\|shot\|図解'` の 9 行は、`snapshot` や `golangci-lint` の部分一致だけ)、結果はこの行の Debt 欄と `inbox.jsonl` の本文にしかない | `docs/tech-debt/README.md` 216 行 Debt 欄の 2 つの文。`docs/evidence/org-trial-2026-10-20/inbox.jsonl` の e1・e2 の `body`。コミット d02e7a24(題は「record F14 and the row 216 trial result」)の証拠側の変更は F14 の 11 行だけ | 前半を「before the stage 9 trial, no run … had」に直す。証拠に 3〜4 行(どの座席が `VISUAL:` を付けたか、lint と shot は座席の自己申告で PNG を読んだかは未確認、codex の leader は試していない)を足すか、行の Trigger を「証拠に記録した」で閉じる。register と証拠の編集は `post-implementation-pipeline.md` の「記録だけの修正」の外だが、`/verify` の前なので追加の cycle は要らない |
| LOW (L1) | maintainability(計画の記録) | 計画の Progress の「Deviations from the body」に、director が ruleset を変えた件がない。計画の冒頭の決定の 3 つ目(人が ruleset を直す)と Non-goals(63 行)の「Changing the repository's ruleset (the human does it), or GitHub settings of any kind」に対し、実際は I6 で人が director に「直して merge しろ」と言い、director が `gh api -X PUT` で ruleset を変えた(証拠の Trial deviations の I6 と、tech-debt の RESOLVED コメントの「on the human's word」は書いている)。計画の注の逸脱は 5 項目(D6、F8、interventions.log、R1 と D-F3 と F10 の項、AC8 の行番号)で、この件がない。Non-goals に直接反する逸脱で、GitHub の設定を 1 件変えた | `docs/plans/active/2026-10-20-org-trial.md` 63 行と 481〜502 行。証拠 853〜854 行(I6)。tech-debt 298 行のコメント | 注の逸脱の一覧に「I6: ruleset の変更は人の指示で director が行った(Non-goals は人が行うとしていた)。証拠の Trial deviations と tech-debt 298 行に記録」を 1 項目足す |
| LOW (L2) | maintainability(記録の主張) | 246 行(org-merge-guarantee の小さな残り)の Debt 欄の冒頭の文が、「(g) is a gap in the tests, and (h) is a gap in the escaping of doctor's Detail」と (h) を現在形の欠陥として挙げたまま。(h) の本文、Impact、Why、Trigger は打ち消して RESOLVED を付けたが、この総覧の 1 文だけが残る。同じ行で (h) が開いているとも閉じているとも読める | `docs/tech-debt/README.md` 246 行 Debt 欄の 1 文目と、同じ欄の `~~(h) ...~~ (RESOLVED 2026-10-20 in PR #242 ...)` | 1 文目から「and (h) is a gap in the escaping of doctor's Detail」を外すか、(h) の部分を `~~` で囲む。217 行の 1 文目(「none changes what is refused or allowed」)は (a) の閉じ方と矛盾しないので触らない |
| LOW (L3) | maintainability(参照の壊れやすさ) | 323 行(F14)の Why 欄が「Row 217's (a), which PR #241 closed, named only `\|`」と行番号で別の行を指す。register が行番号で行を指すのはこれが初めてで(base で `grep -noE '[Rr]ows? [0-9]{2,3}'` は 0 件、HEAD で 1 件)、計画自身が「PR #240 が 170 行の前に 1 行足したので 170 より後の行は全部 1 つずれた」と書いている。次に上のどこかへ行が入ると 217 が別の行になる | `docs/tech-debt/README.md` 323 行 Why 欄。計画 465〜469 行(行番号のずれの注) | 「the org-plan-approval leftovers row's (a)」のように行の頭の文で指す。この PR の他の相互参照(「the `check-sync.sh` row above」「the WIP commit row below」「the `truncateForDetails` row above (F12)」)は位置と F 番号の併記で、新しい 12 行が並んで入る間は合っている |
| LOW (L4) | maintainability(記録の主張) | 309 行(`run_tests` に `-timeout` がない行)は、Debt 欄に F10(director の `ralph org regate` が `GOFLAGS` なしで走り、`internal/cli` が 601.4 s、`internal/org` が 602.7 s で timeout)の長い段落が入ったが、Impact は「A local run-verify can fail on time alone」、Trigger は「The next local run-verify that times out, or a CI verify job that takes more than 8 minutes」のまま。F10 の失敗は run-verify ではなく gate と regate の経路で、Trigger に当たらず、Impact の説明にも入っていない。Debt 欄の最後の文(「Neither `ralph org gate` nor `ralph org regate` sets a test timeout, so a gate's result depends on the environment of whoever runs it」)が行の中でいちばん新しい情報で、読み手が Impact と Trigger だけを見ると外れる | `docs/tech-debt/README.md` 309 行の Debt・Impact・Trigger の 3 欄 | Impact に「a gate or regate run from a shell without `GOFLAGS` fails the same way」を 1 文、Trigger に「a gate or regate that times out」を足す |
| LOW (L5) | typo(ラベル) | 証拠の Not verified の項「Why one watchdog escalation has no shim notification (F6) was not investigated, and the records cannot tell: … (F13)」が、同じ文で F6 と F13 を両方挙げる。F6 の本文(732〜739 行)に shim の通知がない escalation の話はなく、該当は F13(814〜824 行)。コミット 9c9a4f5e の題は「point the not-verified escalation item at F13」で、文末に F13 を足したが、文頭の F6 を直していない | `docs/evidence/org-trial-2026-10-20.md` 907〜909 行。コミット 9c9a4f5e の diff(F6 は context 行) | 文頭の `(F6)` を `(F13)` にして文末の `(F13)` を外す |
| INFO (I1) | maintainability | sanitize 済みの 2 つの org report(`reports/org-manifest-doctor-escape-2026-10-20.md` に 2 行、`...report-escape-2026-10-20.md` に 1 行)に生の U+FFFD(EF BF BD)が入っている。証拠の Snapshot の節が「kept as ralph wrote them」と明記する意図どおりで、mojibake hook は Edit と Write でしか走らず、CI にも検査はないので、今は何も落ちない。ただ `.claude/hooks/mojibake-allowlist` にこの 2 ファイルは入っておらず、あとで誰かが Edit すると exit 2 で「壊れた箇所を書き直せ」と促され、逐語の証拠を直してしまう | `grep -c $'\xef\xbf\xbd'` で 2 ファイルだけに 2 行と 1 行。`.claude/hooks/mojibake-allowlist` に `docs/evidence/` の項目はない。証拠 101〜106 行 | 直さなくてよい。直すなら証拠の Snapshot の節に「この 2 ファイルは逐語。Edit せず、直すなら Bash で」を 1 文足す。allowlist の追加は `.claude/hooks/` と mirror の話になるのでこの PR の範囲の外 |
| INFO (I2) | readability | 証拠の散文で、折り返しの幅を外れた行が 3 つある。700 行(109 桁、F3 の文の途中を書き直したあとの再折り返し漏れ)、284 行(87 桁で、次の行は短い)、302 行(223 桁の inline code 1 つ。出力を逐語で写したもので、折れない)。`git diff --check` は `gate-excerpts.md` の 23・25・32・36 行(原本の空行の写し「392: 」の末尾の空白)に trailing whitespace を出す。逐語の写しで意図どおり、リポジトリに検査もない | `docs/evidence/org-trial-2026-10-20.md` 284・302・700 行。`docs/evidence/org-trial-2026-10-20/gate-excerpts.md` 23・25・32・36 行 | 700 行と 284 行は折り直す。残りはそのままでよい |
| INFO (I3) | maintainability | 304 行(`TestRegate_FetchFailures` の flake)の Trigger「The test fails again, in CI or by hand: run it alone first, then find the cause」は、試行で 2 回満たされた(F9: gate 2 と director の clone での実行)。行は「cause was not looked for (the trial plan's Non-goals)」と書いて開いたままにしており、扱い自体は正しい。Trigger の文言が、次の 1 件を新しい情報なしに同じ形で再び満たす | `docs/tech-debt/README.md` 304 行の Trigger と Debt 欄の末尾 | 急がない。次に触るとき、Trigger を「a failure where the git output before exit 128 is recorded」のように、原因の調べ方に結びつける |

## Positive notes

- 証拠の再計算コマンド 17 本がすべて、表または `text` ブロックと一致した。sh の `|` を含む jq の式も、worktree の root から相対パスで動く
- spec の FR-12 の注の数字(7 回、1302 秒、109 秒と 423 秒、213 秒と 997 秒、$12.81、43 秒、10・30・10・60 の 4 つの既定値と「変えない」)は、証拠の表と一致し、注は「試行は値が正しいことも示していない」と、結論が示す範囲を超えない書き方をしている。Open questions の 3 項目の更新は FR-12 の注を指す形で、数字を重ねて持たない。介入時間の 1302 と 1301 の差は、証拠の冒頭が理由(秒未満の丸め)まで書いている
- 変えた・足した tech-debt の 19 行は、すべて 5 欄(unescaped の `|` で数えた)。217 行と 246 行の部分的な閉じ方は、register の既存の形(複数項目の行で、項目を `~~` で打ち消して `(RESOLVED <日付> in <PR>, <branch>, merge commit <SHA>: ...)` を付ける。178 行ほか)に沿い、Debt・Impact・Trigger の 3 欄に Related の「resolved in」を添えている。299 行(base の 298 行、ruleset の行)は完全に閉じた行で最も多い形(Debt と Trigger を打ち消し、Impact と Why は現在形、行の上の 298 行に `<!-- RESOLVED ... -->`)。Trigger 欄の「— RESOLVED: ...」で結末が読める
- 閉じた 2 行の主張は、コードとテストに合う。`mergeGuaranteeResult` は 90 行で `org.PrintableInboxText(..., false)` を 1 回通し、`TestMergeGuaranteeResult_EscapesControlCharacters` と `TestBuildOrgReport_EscapesPipesInEventDetails` は実在する。F14 が言う「#241 のあとも `\|` だけ」は `report.go` の 180 行と一致する。新しい行が名指すシンボル(`failStepWithNote`、`compensatePane`、`compensateLeave`、`featureLeaderTask`、`truncateForDetails`、`escalateAlert`、`realEscalate`)も、行番号の引用(`watch.go:404`、`watch.go:1557`、`report.go:180`)も、いまのコードに合う
- 新しい 12 行は、所見ごとに「どこで起きたか(F の番号、時刻、件数)」「なぜ今直さないか(計画の Non-goals)」「何が起きたら直すか」の 3 つが分かれて書かれ、Trigger は動作で書かれている。F13 の主張(doctor-escape の leader の `leader_is_anomaly_subject` は escalation 12 件で shim の通知 11 件、03:48:26Z の 1 件だけ通知がない)は `escalations.jsonl` と `osascript.log` から直接数えて合った
- Related 欄の計画への参照は `docs/plans/active/2026-10-20-org-trial.md` で、`/pr` の step 8 の `scripts/archive-plan.sh` が register のその参照を `archive/` に書き換える(`Updated N reference(s) in docs/tech-debt/README.md`)。証拠の 3 行目の同じ参照は書き換わらないが、`director-liveness-livefire` の証拠が先例で、問題にしていない
- 計画の Progress の注は、S0 のあとで D11 の文面と実際の PR #240 が違う点、tech-debt の行番号のずれ(行は文面で探す)、D6 の変形、F8 の移動、interventions.log の書き直し、`AC8` の行番号を、ここに書いてある

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | — | cycle 1 で、M1・L1〜L5 はすべて文の直しで、register・証拠・計画の 3 ファイルの数行。cap の最後の回で残るときだけ、1 行にまとめて足す | — | — |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件は、216 行の Debt 欄が試行の前の文と試行の結果を同じ欄に並べて食い違う点と、その結果が証拠に書かれていない点(M1)。LOW 5 件は、計画の逸脱の一覧に ruleset の変更がない点(L1)、246 行の総覧に (h) が残る点(L2)、323 行の行番号参照(L3)、309 行の Impact と Trigger が F10 を含まない点(L4)、証拠の F6 と F13 のラベル(L5)。漏えい、再計算、数字の一致、行の欄の数、閉じた行の主張には欠陥が見つからなかった)
- Follow-ups:
  - M1・L1〜L5 の直しは、tech-debt の 4 行(216・246・309・323)、証拠の 1 項(Not verified)と 3〜4 行(M1)、計画の注 1 項目で、コードは変わらない。register・証拠・計画の編集は `post-implementation-pipeline.md` の「記録だけの修正」の外にあたる。いまは `/verify` の前なので、直すかどうかと、直したあとに self-review を回し直すかは lead の判断に任せる。直さずに進めるなら、cycle 2 の self-review で残りを 1 行の tech-debt にまとめる(今回は足していない)
  - I1〜I3 は直さなくてよい。I2 の 700 行と 284 行の折り直しは、M1 の証拠の追記と同じ commit に入れると手間が少ない
  - insight event(`self_review`、cycle 1、critical 0、high 0、medium 1、low 5)は、依頼が「このレポートだけを commit」だったので足していない。`./scripts/insights-append.sh --slug org-trial --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 1 --low 5 --cycle auto --source skill` で足せる

## Re-review of the fixes (a288d23b)

- Date: 2026-10-20
- Scope: `git show a288d23b` だけ(`docs/tech-debt/README.md`、`docs/evidence/org-trial-2026-10-20.md`、計画の Progress の注。+28/-16)。diff の品質だけを見た。テスト、静的解析、仕様適合は走らせていない
- 確かめたこと: 変えた 4 つの register 行を欄ごとに旧新で比べた(未エスケープの `|` で分けて、どれも 7 片 = 5 欄で変わらない)。証拠の 5 か所は `--word-diff` で語の増減を見た。M1 の証拠の文は `inbox.jsonl` の e1・e2 の本文と、F6・F13 の参照は証拠の全箇所と、計画の注は計画の決定・Non-goals と証拠の Trial deviations と register 298 行のコメントと突き合わせた。`./scripts/plan-visual.sh digest docs/plans/active/2026-10-20-org-trial.md` は `5b82ee1e30a1`(承認時と同じ)。足した行に `/Users/`・`/home/`・pane id の形・UUID・メールの形・login name・秘密の形はなく、U+FFFD も 0 件。`git diff --check a288d23b^ a288d23b` は出力なし。証拠の変更 5 か所はどれもコードフェンスの外で、表・`sh` ブロック・`text` ブロックに触れていないので、前回の再計算 17 本の一致はそのまま有効(再実行はしていない)

| ID | 結果 | 確かめたこと |
| --- | --- | --- |
| M1 | fixed | 216 行の Debt 欄の前半が「before the stage 9 trial, no run of a claude or codex leader in a herdr pane had done so」になり、後半の試行の結果と食い違わない。`was unknown` への時制の変更は妥当で、「before the stage 9 trial」が 1 文全体にかかり、次の文が「PNG を読んだかは leader の自己申告だけ、codex の leader は試していない」と残りを述べる。証拠には 383〜386 行(「Approval wait per feature plan」の末尾)に 4 行が入った。e1・e2 の本文は両方に `VISUAL: .harness/state/plan-visual/<slug>.html` の行があり、SUMMARY が「図解ページは lint が通り、shot で撮って確かめた」なので、証拠の文(「passed `lint` and was shot」「leaders' own claim」「PNG を読んだかは未確認」)は本文より控えめで、超えていない。座席は計画の Non-goals どおり claude だけで、「no codex leader was tried」と合う |
| L1 | fixed | 計画の注に I6 の項が入った(AC8 の項と同じ深さ)。「third user decision」は冒頭の決定の 3 つ目(人が ruleset を直す)、Non-goals は 63 行、`gh api -X PUT` の 06:09:55Z は証拠 288〜289 行、Trial deviations の I6 は証拠 860 行、「on the human's word」は register 298 行のコメントにあり、5 つとも合う。digest は変わらない |
| L2 | fixed | 246 行の 1 文目は「(a), (b), (d), (e), (f) and (i) change no behavior, and (g) is a gap in the tests.」になり、(h) の語が総覧から消えた。`~~` で囲まず文から外した判断は妥当(総覧には (c) も元から入っていない)。(h) の本文・Impact・Why・Trigger・Related の閉じ方は変えていない |
| L3 | fixed | 323 行の Why 欄は「The org-plan-approval leftovers row's (a)」。217 行の先頭は「Small leftovers of org-plan-approval」、(a) は PR #241 で RESOLVED なので指す先が合う。register に `[Rr]ows? [0-9]{2,3}` の形はなくなった(grep 0 件) |
| L4 | fixed | 309 行の Impact に「A gate or regate run from a shell without `GOFLAGS` fails the same way (F10)」、Trigger に「or a gate or regate that times out」。Debt 欄の F10(regate は `GOFLAGS` なしで timeout、座席の gate は herdr の環境の `-timeout=30m` で通った)と合う。Trigger の形は元の「a CI verify job that ...」と同じ並びで、「next」が付かない点は元の文の形を引いただけなので指摘にしない |
| L5 | fixed | 証拠 914〜916 行は「(F13) was not investigated, and the records cannot tell: the watchdog's path discards the notification's error」。F13 の本文(821 行)がその内容で、ほかの `F6`(198・427・739 行)と `F13`(470 行)の参照も合う |
| I1 | fixed | Snapshot の節(106〜107 行)に「The two org reports are verbatim: do not edit them with the Edit or Write tools (the mojibake hook would ask to rewrite the U+FFFD)」。`.claude/hooks/check_mojibake.sh` は編集された file 全体を走査して exit 2 を返すので、文は正しい |
| I2 | fixed | 284 行と 700 行の再折り返し。`--word-diff` で語の増減はなく、空白の位置だけが変わった。2 つとも 80 桁以内に戻り、残る長い行(inline code を含む行と 302 行の 223 桁)は前回の報告が許した形のまま |
| I3 | not fixed | 依頼どおり直していない。304 行の Trigger は前回の報告のまま(INFO) |

### New findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (N1) | maintainability(記録の主張) | 216 行の Trigger 欄が元の文のまま「Stage 9 (FR-12, two orgs run at once), or the first feature org run in per-plan mode: record whether the page was made, whether `lint` ran ..., whether the page was shot, and whether the PNG was read, for each driver」。この PR が stage 9 の試行で、Debt 欄は「The stage 9 trial (2026-10-20) met the trigger for claude ... no codex leader was tried, so the row stays open」と書くので、Trigger の 2 つの条件はどちらも満たされ、まだ満たされていない条件(codex の leader の run、claude の leader が PNG を読んだことの確認)は Debt 欄にしか書かれていない。Debt 欄だけ直して Trigger 欄が古い文を残す形で、register の他の行で繰り返し出ている(PR 自身の段階を Trigger が名指す形)。もう 1 点、証拠に結果の 4 行が入ったのに、Related 欄は証拠の該当の節を指さない(Debt 欄が `inbox.jsonl` を引くだけ)。前回の報告は Trigger の「record」の置き場の不在を指摘しただけで、この 2 点には触れていなかった | `docs/tech-debt/README.md` 216 行の Trigger 欄(4 つ目)と Related 欄(5 つ目)、Debt 欄の最後の 2 文。証拠 383〜386 行 | Trigger 欄を残る条件で書き直す(例: 「The first feature org run with a codex leader, or a claude leader run whose session record shows the PNG read: record ...」)。Related 欄に `docs/evidence/org-trial-2026-10-20.md`(Approval wait per feature plan)を足す。register の 1 行の編集で、コードも digest も変わらない |

- Merge: yes(CRITICAL・HIGH なし。前回の MEDIUM 1 件(M1)と LOW 5 件(L1〜L5)と INFO 2 件(I1・I2)は直り、I3 は依頼どおり据え置き。新しい LOW は 1 件(N1)で、216 行の Trigger 欄と Related 欄の 2 か所の直し。`/verify` の前なので、直すかどうかは lead の判断に任せる。直さずに最後の self-review の回まで残るなら、その回で tech-debt の 1 行にまとめる)
- insight event: 前回の cycle 1 の event(medium 1、low 5)はそのまま。この再レビューの N1 を数えた event は足していない(依頼が「このレポートだけを commit」)
