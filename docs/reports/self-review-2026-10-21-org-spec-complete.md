# Self-review report: org-spec-complete

- Date: 2026-10-21
- Plan: `docs/plans/active/2026-10-21-org-spec-complete.md`(パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 765bcc46...HEAD`(HEAD 264814a4、3 ファイル、+315/-34)。docs だけの変更で、コードは含まない。内訳は仕様 `docs/specs/2026-10-07-org-multi-org-director.md`(+76/-34)、根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md`(新規 108 行)、計画の Progress の 1 行。diff の品質だけを見た。仕様適合、テストの実行、静的解析、文書のずれは `/verify`・`/test`・`/sync-docs` に任せた。

## Evidence reviewed

- 全文を読んだ: 根拠の表 108 行、計画 131 行、仕様の diff 全体(`-U0` と、変えた行ごとの語単位の差分)
- 仕様の変更が範囲に収まるか(計画の AC7): base と HEAD の仕様を行の集合で比べた。base の行のうち HEAD に同じ形で残らないのは 34 行で、内訳は印の `[ ]` から `[x]` への 1 文字の変更が 20、Rollout の PR 番号が 9、文を直した行が 5(7c-2 の注、8a-3b の注、8a-3a の注、8b-1 の注、FR-12 の注)。5 行の語単位の差分は、「checkbox は付けない」の類の文の書き換え、8b-1 の注の「121 行」「120 行」4 か所を受け入れ条件の文言に置き換えた分、のどちらかだけだった。要件・設計・受け入れ条件の文は変わっていない。足した行は Status の節 23 行と「印(2026-10-21)」の注 19 行
- 印と根拠の表の判定: 仕様の 32 個の checkbox を、表の 31 行(FR-12 は表の対象外)と 1 つずつ照らした。`[x]` は 21(表の 20 と、前から `[x]` の FR-12)、`[ ]` は 11 で、表の「判定の数」と一致した。`[ ]` の 11 項目の F 番号は、Status の節が挙げる F3、F4、F6、F7、F10、F13、D-F3 の全部と、Open questions の 2 つに重なる
- 注と表の突き合わせ: 19 の注を、表の「根拠」「注」の列と、試行の証拠(`docs/evidence/org-trial-2026-10-20.md` の F4・F6・F7・F10 の本文、Trial deviations、Not verified)と、仕様の該当の注(FR-6 の 6 段目の後半、FR-8 の 8a-3a、FR-9 の 8a-1 と 8a-2、FR-10 の 8b-1、4 段目の計画の「`rejected` は書かない」)に当たって照らした。数字と固有名(「4 回目の spawn」「4 回」「49 分後」「30 件」「`GOFLAGS=-timeout=30m`」「`guarded`」)は合った
- 根拠の表の記述の確かめ(読むだけ): 表の 65 個のテスト名を `git grep -n 'func <名前>(' -- '*.go'` で数え、101 か所の参照すべてで 1 件ずつ、表が書く `_test.go` のパスの中に見つかった。表が指す証拠のファイル 12 本は存在する。試行の節の見出し(Two orgs、Merges、Gates、Stop and restart の Part 1 と Part 2、Interventions、Trial deviations、Approval wait per feature plan、The director's handling of the inbox)はすべて実在する。tech-debt の行の名前 15 個は、`docs/tech-debt/README.md` の行の最初のセルの先頭に 1 件ずつある。Open questions の 1 つ目と 2 つ目は、FR-10 と NFR の注が引く文と同じ。`.claude/skills/org/SKILL.md` に「## 機能ごとの org」があり、「編成パターン」「Parallel」「Leaded」は 0 件。`.claude/skills/director/SKILL.md` に「### merge と auto_merge」がある。`scripts/verify.local.sh` が `check-sync.sh` と `check-skill-sync.sh` を走らせる
- 数の確かめ: PR 28 本(仕様 1、1 段目 1、2 段目 2、3 段目 1、4 段目 2、5 段目 1、6 段目 2、7 段目 4、8a 5、8b 4、9 段目 2、機能の PR 2、#244 の 1)は Status、Rollout、表の「段ごとの PR」で一致。表の「NFR 入れ方」の 25 本は、`docs/plans/archive/` の `2026-10-07` 以降の org の計画 24 本と `2026-10-20-secret-scan-test-tmp` の計 25 本に、self-review・verify・test・sync-docs の 4 つの report がすべてあることで裏づく(欠けは 0)。最新のタグは v5.1.0(2026-08-24)で、Status の「release はまだ切っていない」は今日は正しい
- F4 の推定の元: `internal/org/verbs_all.go` の `StopAll` は `Roster(...)` の `s.Active` だけを止め、`internal/org/reserve.go` の `TotalActiveSeats` も同じ `s.Active` だけを数え、`internal/org/seat.go` の `activeEvents` は `spawn_started`・`spawn_step`・`spawned` の 3 つで `spawn_failed` を含まない
- `git diff --check`(出力なし)、追加行への `/Users/`・`/home/`・メールの形・login name・`ghp_`・`sk-`・`AKIA`・`xox` の grep(0 件)、3 ファイルの U+FFFD(0 件)、`./scripts/plan-visual.sh digest`(`98781cfa6662`、承認時と同じ。S2 のあとの計画の差は Progress の 1 行だけ)
- 確かめていない: テストの本文(名前と置き場所だけを確かめた。表が「各テストの doc comment と本文の一部を読んだ」と書くとおりで、私もそれ以上は読んでいない)、`gh pr view` での PR の状態(表の記録を信じ、数の整合だけ見た)、F1〜F14 のうち F4・F6・F7・F10 以外の本文、natural-japanese の lint(hook が書き込み時に回す)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (M1) | maintainability(記録の主張) | F4 の同じ推定が、FR-2 では `[ ]` の理由になり、FR-3 では印を変えない扱いになっていて、FR-3 と受け入れ条件 3 の `[x]` に付く既知の未確認が仕様から読めない。FR-2、受け入れ条件 2、NFR 安全の不変条件は、「`spawn_failed` と記録されたまま動く座席を `stop --all` は止めないはず(コードからの推定、再現していない)」を理由に `[ ]` にした。一方、同じ座席を `max_total_seats` が数えないこと(`TotalActiveSeats` も同じ `s.Active` で数える)は、表の「確かめていないこと」の 2 つ目の項目に「FR-3 の印はこの推定では変えていない」と書かれただけで、理由は書かれていない。FR-3 の 3 つ目の項目「範囲の外への変更は、今の watchdog の検知(ALERT)で知らせる」についても、表の F6 の行は「検知は動いて ALERT を上げた」と読んで `[x]` を支える一方、「確かめていないこと」の 3 つ目は「FR-3 の範囲の外への変更の知らせは、この ALERT に頼る」と書き、同じ表の中で 2 つの記述が引き合う(試行の `watch.log` では leader への ALERT の失敗 10 回のうち 7 回が `scope_change`。`scope_change` の escalation は 13 件で、人への通知は 1 件を除いて shim に残る。その 1 件(03:48:26Z、doctor-escape の leader)が、原因を調べていない F13 そのもの)。仕様の FR-3 と受け入れ条件 3 の項目には、FR-2 や受け入れ条件 3 の `rejected` の注のような印の注が 1 つもない | 仕様 60 行(FR-2 の注)と 61 行(FR-3、注なしの `[x]`)、149〜151 行(受け入れ条件 2 の注と、受け入れ条件 3 の `rejected` の注。F4 の座席の数え方には触れない)。表 47 行(FR-2)、48 行(FR-3 の「注」の列)、103・104 行(確かめていないこと)。`internal/org/reserve.go` の `TotalActiveSeats`、`internal/org/verbs_all.go` の `StopAll`、`internal/org/seat.go` の `activeEvents`。`docs/evidence/org-trial-2026-10-20/watch.log` の `failed to ALERT leader` 10 行(うち `scope_change` が 7 行) | 次のどちらかを、`/verify` の前に決める。(a) FR-3 の項目の下に印の注を 1 つ足す。「`[x]`。次の 2 つは確かめていない: `spawn_failed` と記録されて動く座席が `max_total_seats` の数に入らないはず(FR-2 と同じ推定。再現していない)。試行の leader への ALERT の失敗 10 回のうち 7 回は `scope_change` で、原因は調べていない」。受け入れ条件 3 の注にも 1 句足す。(b) 推定で FR-2 を `[ ]` にしたのと同じ基準で FR-3 と受け入れ条件 3 も `[ ]` にし、表の判定の数(20 と 11)と Status の「11 項目」を直す。どちらでも、表の F4 の行の「FR-3 の印は…変えていない」に、変えない理由を 1 文足す |
| LOW (L1) | readability(根拠の言い方) | FR-6 の注は、F6 を FR-6 が `[ ]` である理由に並べるが、表の F6 の行の説明は、FR-6 のどの文とも食い違わないことを示している。F6 の「食い違う」は「はい(FR-6)」で始まるが、続けて「FR-6 は、見張りが人の経路へ直接送るものとして期限の切れた受信箱の件を挙げる。試行では期限の切れた件は 0 件で、人へ直接届いたのはパルスの escalation だった」と書く。FR-6 に、パルスの escalation を director か受信箱に通すと言う文はない(仕様 41 行は、人に届くのは watchdog のデッドマンだと今の姿を書き、FR-9 の director の仕事の列にも入っていない)。FR-6 を `[ ]` にする根拠は、F13(「通知を試したこと、人が受け取ったこと、座席を止めたことは、別々に記録する」という FR-6 の 6 つ目の項目に直接反する)だけで足りる。注と Status の「食い違う項目」を読む人は、F6 がどの文に反するのかを探すことになる | 仕様 91 行(FR-6 の注)、20 行(Status)。表 25 行(F6 の行)。仕様 84〜88 行(FR-6 の項目) | FR-6 の注を F13 から始め、F6 は「関連する所見(期限の切れた件ではなく、パルスの escalation が受信箱を通らなかった。設計の穴で、FR-6 の文とは食い違わない)」に分ける。表の F6 の「食い違う」は「いいえ。…」にして、関係する項目に FR-6 を残す。M1 で表の判定を直すなら、同じ編集で |
| LOW (L2) | maintainability(時間で古くなる文) | Status の「release(タグ)はまだ切っていない。人が `/release` で切る。」に日付がない。今日は正しい(最新のタグは v5.1.0、2026-08-24)が、タグが切られた日に誤りになり、仕様を直す理由が誰にも生じない。同じ節の 1 文目は「2026-10-21 に `gh pr view` で確かめた」と日付を持つので、この文だけが日付なしの現在形になっている | 仕様 24 行。`git tag --sort=-creatordate | head -1` は v5.1.0 | 「2026-10-21 時点で最新のタグは v5.1.0(2026-08-24)で、この仕様の段を含む release は切っていない。人が `/release` で切る」の形にする。「まだ」と現在形をやめる |
| LOW (L3) | readability(指す先がない言葉) | Status の「試行の所見の直しは、この系列の後の PR で入れる。それぞれの PR は…」の「この系列」は、仕様の中に指す先がない。定義は根拠の表の冒頭(「この PR に続く系列」)にだけある。仕様だけを読む人は、系列が何を指すかも、どこに系列の一覧があるかも分からない | 仕様 22 行。表 16 行 | 「試行の所見の直しは、この PR のあとの別々の PR で入れる(作業名と順は根拠の表の「試行の所見」の節)」の形にする |
| LOW (L4) | maintainability(確かめられない主張) | 表の NFR パリティの行の根拠に「このブランチの run-verify も通った」とある。いつの run かも、ログの場所も、exit code もなく、リポジトリのどこからも確かめられない(計画は S1 の implementer が run-verify のあとコミットの前で止まったと書くだけ)。同じ文の前半の「`scripts/verify.local.sh` が…`check-sync.sh` と `check-skill-sync.sh` を走らせ、CI も走らせる」が、`[x]` を支える仕組みの根拠として足りている | 表 60 行。計画 127 行の Progress | 後半の「このブランチの run-verify も通った」を削る(`/verify` の report が持つ事実なので、この表に重ねる必要がない)。残すなら report のパスと日付を添える |
| LOW (L5) | readability(日本語) | 次の 4 点は、読み返さないと意味が取れない。(a) 表の冒頭の「食い違うとは、項目の文が起きないと言うことを試行で見たか、…」は、「項目の文が(起きない、と)言っていることが、試行で起きた」の意味だが、そう読めない。F の行すべての判定がこの定義に依るので、誤読が一番高くつく。(b) 仕様 136 行(NFR 安全の不変条件の注)の「(状態の説明)」は、NFR の本文の「状態の説明・停止・期限切れの件の人への通知」の最初の語を断片で借りたもので、注の中では何の説明か分からない。(c) 仕様 147 行(受け入れ条件 1 の注)の「合成の根拠で」は、表の 59・62 行の言い方(「合成の根拠」)をそのまま写した造語で、注の読み手に説明がない。(d) 仕様 101 行(7c-2 の注)は、直したあとの 1 文目が「(8a-3b の注で決めた)」と決まった話で終わり、次の文が「それと FR-8 の `auto_merge` と director の報告は 8a 段目に残した」と始まる。元は「付けない。それと…残した」で、どちらも「未決のまま残したこと」を並べていたが、今は前の文が決着したので「それと」の並びが合わない | 表 5 行(定義)。仕様 136・147・101 行 | (a)「項目の文が起きないと言っていることが、試行で起きた。または、…回り道が要った」に。(b)「(NFR の「状態の説明」)」に。(c)「別々のテストで確かめた」と言い切り、「合成の根拠」の語は表の判定の注に置く。(d)「それと」を消すか「このほかに」にする |
| INFO (I1) | maintainability(参照の形) | 受け入れ条件 3 の注が、判断の根拠として `docs/plans/archive/2026-10-09-org-feature-worktree.md` と完全なパスで書く。仕様の中で計画を指す書き方は、ほかのすべての注が「詳細は 4 段目の計画(`2026-10-09-org-feature-worktree.md`)に書いた」の形(ファイル名だけ)。また「`start --plan` の事前の検査は何も書かずに断る」という決定は、4 段目の注にも仕様のどこにもなく、アーカイブされた計画の 31 行にだけある。受け入れ条件 3 の読みの根拠が仕様の外にある | 仕様 151 行。`docs/plans/archive/2026-10-09-org-feature-worktree.md` 31 行 | 急がない。パスをファイル名だけの形にそろえる。決定を 4 段目の注に 1 文足すのが筋だが、AC7(要件・設計の文を変えない)と引き合うので、系列の PR に回す |
| INFO (I2) | readability(数え方) | Status の 1 文目「段ごとの PR は次のとおりで、28 本すべてが merge 済み」の 28 本には、段の PR でない #241・#242(試行の機能の PR)と #244(F8 の直し)が含まれる(リストには別に書いてある)。表の NFR 入れ方の行の「9 段の PR のうち…25 本」も、9 段の外の #244 を含んで 25 にしている。どちらも数は正しく、リストを読めば分かる | 仕様 5 行。表 61 行 | 急がない。「次の 28 本」「関係する PR 28 本のうち」に言い換えると誤読がなくなる |
| INFO (I3) | maintainability(判断の記録) | 受け入れ条件 1 は `[x]` だが、表の「確かめていないこと」が「3 つの org は、テストでも実機でも 1 度に動かしていない」と書く。表の規則(部品ごとのテストで確かめた項目は `[x]`)に沿った判定で、注も「合成の根拠」「実機は 2 つの org まで」と開示している。ここは判定の好みの問題で、欠陥ではない。M1 で `[x]` の基準を締めるなら、同じ基準で読み直す余地がある | 仕様 146・147 行。表 62・105 行 | 対応は不要。M1 の方針(b)を選ぶ場合だけ見直す |

## Positive notes

- 仕様の変更は、計画の範囲に収まっている。base の 34 行(印 20、Rollout 9、文 5)と足した行(Status 23、注 19)を行単位で分類して、範囲外の行は 0 だった。8b-1 の注の「121 行」「120 行」4 か所は、条文の文言の引用に置き換わっていて、置き換えた引用は仕様の受け入れ条件の本文と一字一句合う。FR-12 の注に残る `checkbox` の語は 1 か所で、「FR-12 の checkbox は付けた」と今の印に合う
- 31 個の印は、表の判定と 1 対 1 で一致した。`[ ]` の 11 項目はどれも、注が所見の番号か Open questions の問いを指し、tech-debt の行の名前は、表のどの行も `docs/tech-debt/README.md` に 1 件ずつ実在する
- 注は、推定を推定と書く。FR-2、NFR 安全の不変条件、受け入れ条件 2 の注は「はずである(コードからの推定で、再現していない)」と言い切らない。FR-10 と NFR ツール非依存は「所見との食い違いはないが、…実機で動かしていない」と、`[ ]` の理由が所見でなく実機の不足であることを分けて書く。注の数字(4 回目の spawn、4 回待ち切れた、49 分後、30 件、`GOFLAGS=-timeout=30m`)は、試行の証拠の本文と合った
- 根拠の表の記述は、外から確かめられた。65 個のテスト名と 101 か所の参照(置き場所つき)がすべて実在し、証拠のファイル、試行の節の見出し、tech-debt の行も実在する。「確かめていないこと」の節が、推定、未調査の原因、テストの全文を読んでいないことを分けて書き、仕様の注が言い過ぎる方向の誤りを自分で減らしている
- Rollout の PR 番号と Status の PR の一覧は、memory の記録と、表の「段ごとの PR」に一致した。「進行中」は仕様から消えている。前例の `docs/specs/2026-08-17-overlay-scaffold-v2.md` と同じ位置(`#` の見出しの直後)に `## Status` がある
- 漏えいはなく(`/Users/`、メールの形、token の形が 0 件)、U+FFFD は 0 件、`git diff --check` は出力なし。計画の承認 digest は変わっていない

## Coverage gaps

- 表のテストが項目の文のどこまでを見るかは確かめていない。名前と置き場所の実在だけを見た。たとえば `TestMerge_MergesWithTheMethodAndRecordsPRMerged` が NFR 記録の置き場所の「merge を manifest に記録する」を実際に確かめるかは、本文を読んでいない
- `gh pr view` は打っていない。PR の状態は表の記録を信じた
- F1〜F14 のうち F4・F6・F7・F10 を除く本文は読んでいない。表の「食い違う」の判定のうち、F3・F13・D-F3 は表の文と仕様の注の一致だけを見た
- 日本語の lint は回していない(書き込み時の hook の担当)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | — | cycle 1 で、M1・L1〜L5 はすべて文の直しで、仕様と根拠の表の数行。cap の最後の回で残るときだけ、1 行にまとめて足す | — | — |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件は、F4 の同じ推定が FR-2 では `[ ]` の理由になり、FR-3 と受け入れ条件 3 では未確認のまま `[x]` で、その注が仕様にない点(M1)。LOW 5 件は、FR-6 の注が F6 を FR-6 の文との食い違いのように並べる点(L1)、Status の release の文に日付がない点(L2)、「この系列」の指す先が仕様にない点(L3)、「このブランチの run-verify も通った」を確かめられない点(L4)、日本語の読みにくい 4 か所(L5)。範囲の遵守、印と表の判定の 1 対 1 の一致、テスト名・証拠・tech-debt の行・PR の数の実在、漏えい、U+FFFD には欠陥が見つからなかった)
- Follow-ups:
  - M1・L1〜L5 の直しは、仕様の数行(FR-3 の注、FR-6 の注、Status の 2 文、101・136・147 行)と表の数行(5 行目、25 行目、60 行目、103 行目)で、コードは変わらない。これらは `post-implementation-pipeline.md` の「記録だけの修正」の外(仕様と表は、この task の report や insight event ではない)。いまは `/verify` の前なので、直すかどうかと、直したあとに self-review を回し直すかは lead の判断に任せる。M1 の(b)を選ぶと判定の数が変わるので、`/verify` の grep の数え上げの前に直すと手戻りがない
  - I1〜I3 は直さなくてよい
  - insight event(`self_review`、cycle 1、critical 0、high 0、medium 1、low 5)は、依頼が「このレポートだけを commit」だったので足していない。`./scripts/insights-append.sh --slug org-spec-complete --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 1 --low 5 --cycle auto --source skill` で足せる
