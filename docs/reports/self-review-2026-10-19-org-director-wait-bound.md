# Self-review report: org-director-wait-bound

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-19-org-director-wait-bound.md`(ステージ 8b-2c、パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 3a803a87...HEAD`(HEAD 82df77df、10 ファイル、+763/-66)。diff の品質だけを見た。仕様適合、テストの網羅、文書のずれ、静的解析、テストの実行は `/verify` と `/test` に任せ、このレビューでは走らせていない

## Evidence reviewed

- 全文を読んだ: `internal/org/director_skill_sync_test.go`(408 行。足した 150 行目以降と、足す前の既存のテスト)、`docs/evidence/director-wait-bound-livefire-2026-10-19.md`、`docs/evidence/director-wait-monitor-2026-10-19.md`、計画 `docs/plans/active/2026-10-19-org-director-wait-bound.md`
- diff を読んだ: `.claude/skills/director/SKILL.md`(周りの「受信箱の処理」の節 460〜580 行も読んだ)、`docs/tech-debt/README.md`(247・248 行)、`docs/specs/2026-10-07-org-multi-org-director.md`(追加した 1 行)
- 照らし合わせた: `docs/evidence/director-wait-session-end-2026-10-19.md`(8b-2b の実験。248 行の Bash の `run_in_background` の主張)、`docs/tech-debt/README.md` の `8b-2c`・`Monitor`・`leaked wait`・`part B` を含む行(247・248 のほかになし)、`docs/plans/archive/` と `docs/plans/active/`(テストのコメントが挙げる計画の場所)、前回の `self-review-2026-10-19-org-director-wait.md` の書式
- 手元で確かめた(読むだけのコマンド): `cmp` で skill の 4 つの写しが同じこと、`git diff --check`(rc 0)、追加行の `/Users/`・`hiroki`・12 桁以上の 16 進・UUID・pane id・token の `grep`(evidence の `5522-1792…` 形の `wait_id` と `bg-spare` の語のほかになし)、tech-debt の 245〜249 行の区切りの `|` の数(どれも 6 で見出しと同じ)、mark log の `wait_id` の後ろの数を `date -r` で戻した時刻と `started_at` が一致すること(12:12:14Z、12:13:41Z、12:18:05Z の 3 件)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (M1) | maintainability(記録の主張) | tech-debt 248 行の「Not checked」と仕様の 8b-2c の注が、evidence 自身の「確かめていないこと」より短い。evidence は、本物の `/director` skill で director が何時間も待ち続けるか、headless の director、の 2 つを挙げる。さらに、最初の試験セッションは 5 回目の時間切れのあとに自分で止まり、2 本目には「この役に終わりはない」の 1 行を足していた。248 行の並びは 25 分と 30 分の本物の数字、Bash の `run_in_background`、daemon、codex の 4 つで、この 2 つがない。仕様の注は「skill に従うセッションが…人の入力なしで打ち直し」とだけ書き、試験用の指示だったことを書かない。同じ行の「the live check used bounds of 60 s or less」は、(c) が `--timeout-ms 600000` と Monitor の `timeout_ms` 900000 で打った事実と合わない(`/exit` が先に止めたので、その長さまでは走っていない) | `docs/evidence/director-wait-bound-livefire-2026-10-19.md` の 43〜51 行(Notes)、36 行((c))、122〜129 行(Not checked here)。`docs/tech-debt/README.md` 248 行の末尾「Not checked: …」。仕様の追加行(FR-10 の 8b-2c の注)。計画 R5(「待ち直さなければ skill の文を直してから先に進む」)を閉じた判断は、計画の進捗(126 行)と evidence にしかない | 248 行の Not checked に 2 つを足し、「60 s or less」を「(c) の 10 分と 15 分はその長さまで走っていない」に合わせる。仕様の注は「試験用の director の指示で」と書く。1 文ずつで足りる |
| LOW (L1) | maintainability(記録の主張) | skill の「Monitor の wait は、claude の SIGKILL のほかは、セッションがどう終わっても SIGTERM で止まる」が、確かめた範囲より広い。確かめたのは Claude Code 2.1.291 の 4 つの終わり方(`/exit` の既定、Move to background and exit、pane を閉じる、Monitor の期限)。claude 自身への SIGTERM や、途中で落ちる場合は見ていない。同じ主張を tech-debt 248 行は「every session end claude handles」と、4 つを挙げて狭く書いている。director はこの文で動きを変えないので、読んで得るものがない | `.claude/skills/director/SKILL.md` 501〜502 行(4 つの写しとも同じ)。evidence の `director-wait-monitor-2026-10-19.md` の m1・m2・m4・m5 | 消すか、「`/exit` の既定、Move to background and exit、pane を閉じる、Monitor の期限のいずれでも」に絞る。直すなら `.agents` の 2 つを `scripts/sync-skills.sh` で作り直す |
| LOW (L2) | readability | signal での終わりの例外の段落が、「Monitor が期限切れを知らせた…すぐあとに」と書く。知らせが wait の終わりより先に来ることを前提にしているが、evidence の (f) は「3 秒後に打ち直した」としか書かず、どちらが先に届いたかを記録していない。(g) は知らせがないほうだけ。また、どの wait の Monitor の知らせか、「すぐ」がどれだけかを書かない。前の wait の知らせのあとに、人が別の wait へ `kill -TERM` を送ると、例外に読める。害は打ち直しが 1 回増えるだけで小さい | `.claude/skills/director/SKILL.md` 564〜571 行。evidence の 38 行((f))と 39 行((g)) | 「その wait を見張っていた Monitor の期限切れの知らせが来ていれば(知らせと wait の終わりの前後は問わない)」のように、同じ wait の Monitor に結び、順を外す |
| LOW (L3) | maintainability | 同じ数が 2 か所以上に書かれ、テストが読むのは 1 か所だけ。`1800000` は箇条(488 行)と例外の段落(568 行)にあり、テストは箇条の分しか読まない。「25 分」は 490・499・541 行、「30 分」は 489 行にあり、N と M には結びついていない。N を 1200000 にしても、テストは通り、文は「25 分」のまま残る | `.claude/skills/director/SKILL.md` 487〜490・499・541・568 行。`internal/org/director_skill_sync_test.go` の `directorSkillMonitorTimeoutRe`(163 行)は `directorSkillClaudeCodeWaitBullet` が返す箇条にだけ当てる(232〜234 行) | テストで、全文の「`timeout_ms` に `X`」が全部 M と同じ、を足す(`FindAllStringSubmatch` 1 回)。または段落を「上の `timeout_ms`」にして数を 1 か所にする |
| LOW (L4) | maintainability | テストのコメントが `docs/plans/active/2026-10-19-org-director-wait-bound.md` を挙げる。`/pr` が計画を archive へ動かすと、この道は古くなる。同じファイルの 108 行は `docs/plans/active/2026-10-17-org-auto-merge.md` を挙げ、その計画はいま `docs/plans/archive/` にある。`/pr` の archive の commit は tech-debt の行の道だけ直し、Go のコメントは直さない | `internal/org/director_skill_sync_test.go` 258 行と、先例の 108 行。`ls docs/plans/archive` に `2026-10-17-org-auto-merge.md` がある | 新しいほうは計画を道ではなくスラッグ(`org-director-wait-bound`)で挙げる。108 行は別の PR で直せる |
| LOW (L5) | typo(evidence の文面) | livefire evidence の (e) の行が「waited again at 12:15:18Z; two more timeouts were run again」と書く。mark log では、12:15:18 の wait の時間切れ(12:15:38)は 12:15:40 に打ち直され、12:15:40 の wait の時間切れ(12:16:00)は打ち直されていない(Notes がそのあとで止まったと書く)。打ち直しは 1 回。また、計画の D6 は Monitor の task id を記録すると書くが、evidence に task id はなく、冒頭の「Values left out」にも挙がっていない。意図して外したのか、記録し忘れたのか読めない | `docs/evidence/director-wait-bound-livefire-2026-10-19.md` の 38 行、43〜47 行、77〜79 行、8 行。計画の D6 | 「one more timeout was run again, and the session stopped after the next」にする。「Values left out」に Monitor の task id を足す(外したのなら) |
| LOW (L6) | maintainability(テストの縁) | 表の 14 件は、箇条の範囲(ほかの小節、後ろの Codex の箇条、箇条の後の段落、文末、折り返し)と N・M の大小を押さえている。足りないのは、箇条に `timeout_ms` が 2 つある場合(M の「ちょうど 1 つ」の枝は 0 件のときしか走らない)と、`--timeout-ms 0` が全文の末尾にある場合(正規表現の `$` の枝)。`directorSkillOneNumber` の `strconv.Atoi` の失敗は、桁あふれのときしか起きず、そのときの文言「is not a number」は読み手を迷わせる | `internal/org/director_skill_sync_test.go` 205〜218 行、159〜166 行、322〜393 行の表 | 2 件を表に足し、文言を「does not fit in int」にする |
| INFO (I1) | readability | Monitor の実験の evidence の「Reading」が、現在形で「The `/director` skill tells a Claude Code director to run … `--timeout-ms 0 2>&1` under Monitor」と書く。この PR のあとは偽になる。計画と同じ commit に入るので、書いた時点(3a803a87 の skill)を添えるほうがよい。98 行は 113 文字で、前後の折り返し(72 字ほど)から外れている | `docs/evidence/director-wait-monitor-2026-10-19.md` の 84〜88 行、98 行 | 「At 3a803a87 the skill told …」にする。98 行は折り返す |
| INFO (I2) | readability | tech-debt 248 行は約 4870 字で、Debt の欄だけで 3190 字ほど。隣の 247 行(2800 字ほど)や前の 248 行(2540 字ほど)より大きい。runner ごとの残り方、区切り、印の書き直し、残る穴、親の pid で見ない理由、Not checked の 6 つが 1 つの欄にある。この表の書き方に合っており、列の数も合う。直すなら「親の pid で見ない理由」と「Not checked」を evidence の側に置き、行は結論と道を持つ形にできる | `docs/tech-debt/README.md` 248 行 | 直さなくてよい |
| INFO (I3) | naming | 表のテスト `TestDirectorSkillWaitBoundProblems` は、同じファイルの `TestDirectorSkill_<動詞…>` の形と違い、下線がない。`go test -run TestDirectorSkill` には当たる。ファイルの読み込みと skip の繰り返しが 3 つ目の写しになった | `internal/org/director_skill_sync_test.go` 54〜63・129〜148・279〜287 行 | 3 つのテストが同じ読み込みを使うなら、小さな関数にまとめられる。急がない |

## Positive notes

- skill の 4 つの写しは `cmp` で同じ。変えたのは Claude Code の箇条、終了コードの箇条の 1 と、その前後の段落、例外の段落、`note:` の重複の注意の括弧の 1 つで、`note:` の括弧は 25 分の繰り返しが増えたことへの追従なので不要な変更ではない。Codex の箇条とそのほかの節は変わっていない
- 例外の書き方は狭く守られている。「失敗にしない signal は 1 つだけ」と言い切り、panic、時間切れの文のない終了コード 1、それ以外の signal を「上のとおり失敗」に戻している。director は merge の待ちに Monitor を使わない(731・779・802・815 行は background の実行で、Claude Code では Bash の `run_in_background`)ので、「Monitor expired」の知らせが merge の待ちと混ざる経路もない
- テストの補助関数の doc コメントは、コードの動きと合っている(箇条の終わりは「空でなく、空白で始まらない次の行」。`withSingleSpaces` で折り返しに耐える。N と M はそれぞれちょうど 1 つ)。失敗の文は数を出す(`N=%d must be less than … M=%d`)。直し方は、同じファイルのほかのテストと同じく doc コメントに回し、そこにはあるべき形が書いてある。`directorSkillMonitorMaxTimeoutMs` の更新の手順も書いてある
- tech-debt の 247 行は Trigger の 1 欄だけを直し(8b-2c が終わった段を指していた語を落とし、「at most the wait's bound」を足した)、248 行は閉じずに書き換えて、runner ごとの残り方、25 分と 300 秒の上限、返した件の行き先、親の pid で見ない理由を書いている。主張は evidence と合った: 8b-2b の b2・b4(Bash が Move to background と SIGKILL のあとも残る)、Monitor の m1〜m5、livefire の (b)(c)(d)(40 秒の区切りで終わり、`zsh -c` も終わる)。両行とも区切りの `|` は 6 で、見出しと同じ
- evidence に `$HOME` の絶対パス(`/Users/…`)はなく、`~` と `/private/tmp/xbw` だけ。pane id、claude の session id、12 桁以上の 16 進、token の値はない。`wait_id` は pid と 19 桁の数で、同種の先例が 8b-2b の evidence にある。`git diff --check` は rc 0。mark log の時刻は、`wait_id` の後ろの数を戻した開始時刻と一致する
- 時間切れ、Monitor の期限、外からの SIGTERM の 3 つが、(a)〜(g) の表と mark log で 1 本の筋になっている。(g) の対照(外から TERM を送ると打ち直さない)と、失敗した最初の (g) の記録も残っている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | — | cycle 1 で、上の指摘は直せる。cap の最後の回(cycle 2)で LOW が残るときだけ、1 行にまとめて足す | — | — |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件は、tech-debt 248 行と仕様の注が、evidence 自身の「確かめていないこと」を落としている点(M1)。LOW 6 件は、skill の「セッションがどう終わっても」が確かめた範囲より広い点(L1)、例外の段落が知らせの順と相手の Monitor を決めていない点(L2)、同じ数を 2 か所以上に書いてテストが 1 か所しか読まない点(L3)、テストのコメントが active の計画の道を挙げる点(L4)、evidence の (e) の「two more」と task id の記載(L5)、表の縁の 2 件と Atoi の文言(L6)で、いまの動きを壊すものはない。skill の 4 つの写しの同一、例外の狭さ、evidence の個人情報、tech-debt の区切り、mark log の時刻には欠陥が見つからなかった)
- Follow-ups:
  - M1 は記録の文だけの直し(248 行と仕様の 1 行)で、コードと skill は変わらない。tech-debt と仕様の編集は `post-implementation-pipeline.md` の「記録だけの修正」の例外(`docs/reports/` のレポートと insight event だけ)の外なので、直すとパイプラインの再実行になる
  - L1・L2・L3 は skill の 2 つの元の写しを変える。直すなら `scripts/sync-skills.sh` を 2 回(root と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`)回し、テストの doc コメントと表も合わせる。L3 のテストの足しは L1・L2 の文面に依らない
  - I1〜I3 は上げるだけで、直さなくてよい(I1 は M1 と同じ commit に入れると手間が少ない)
  - 直さずに cap(2)のまま進むなら、L1〜L6 は cycle 2 の self-review で LOW の残りとして 1 行の tech-debt にまとめる(今回は足していない)
  - この節でレポート以外は変えていない。insight event(`self_review`、cycle 1、medium 1、low 6)を `docs/insights/events/2026-10-19-org-director-wait-bound.jsonl` に足し、レポートと同じ commit に入れる

## Re-review after fixes (cycle 1)

- Date: 2026-10-19
- Scope: `git show dc98d5ec`(skill の 4 つの写しとテスト)と `git show ffc56a2e`(tech-debt の 248 行、仕様の 98 行、evidence の 2 つ)。HEAD の 546b46ab は計画の進捗の注だけなので対象外。変えた範囲は全文を読んだ。diff の品質だけを見て、テストと静的解析は走らせていない
- 手元で確かめた(読むだけ): skill の 4 つの写しは `cmp` で同じ。`git diff --check 3a803a87...HEAD` は rc 0。dc98d5ec^ から HEAD の追加行に、`/Users/`、`hiroki`、12 桁以上の 16 進、UUID、token の語、pane id、session id はない。本物の skill の `### wait で待つ`(481〜575 行、中に `#` で始まる行なし)を、テストと同じ切り方と空白の畳み方で Python に通すと、`([0-9]+) ?分` は 5・30・25・25・25・5 の 6 件で、どれも N/60000=25、既定 5、最大 30 のどれか。`` `timeout_ms` に `X` `` は 2 件で、どちらも 1800000(箇条に 1 件、段落に 1 件)。箇条の N は 1500000 の 1 件、M は 1800000 の 1 件。全文に `--timeout-ms 0` はない。表の 21 件は、コードを読んで 1 件ずつ追い、期待した問題の数と順に一致した

### 指摘ごとの結果

- M1: 解消。248 行の Not checked に、本物の `/director` skill で何時間も動く director と headless の director を足し、試験用の指示の枠、1 つ目のセッションが 5 回目の時間切れで止まったこと、2 つ目に足した 1 行も書いた。「60 s or less」は「時間切れまで走らせた区間はどれも 60 s 以下(15 s、40 s、20 s、10 s)で、(c) の 10 分の wait は `/exit` が先に止めた」に合わせた。仕様の 98 行も、試験用の director の指示だったことと、この 2 つを確かめていないことを書いた。どちらも livefire の Notes と表の (b)(c)(d)(g) に合い、248 行の区切りの `|` は 6 のまま
- L1: 解消。skill の文は「`/exit` の既定、Move to background and exit、pane を閉じたとき、Monitor の期限のどれでも」の 4 つに絞られ、Monitor の実験の m1・m2・m5・m4 と 1 対 1 で合う。4 つの写しは同じ
- L2: 一部。「その wait を見張っていた Monitor が期限切れを知らせた」ときに限るので、別の wait の知らせや、(g) のように知らせのない外からの TERM には読めない。1 つの Monitor の期限切れより広くは読めず、この点は解消。知らせと wait の終わりの順を決めない点は、下の N1 に残る
- L3: 解消。小節の `` `timeout_ms` に `X` `` を全部 M と、「<K> 分」を N/60000・既定 5・最大 30 と突き合わせる。小節の外の「10 分」「60 分」(merge の節)は読まず、表の「小節の外」の 1 件がそれを押さえる。N と M の読めない場合の skip と、N が分の倍数でない場合の K の skip は、doc コメントとコードが合う
- L4: 解消。テストのコメントは計画をファイル名だけで挙げ、`active/` や `archive/` の道を持たない。dc98d5ec は同じファイルの 109 行の先例(`docs/plans/active/2026-10-17-org-auto-merge.md`)も `archive/` に直した。この PR の対象外の行だが、指摘どおり古くなっていた道で、害はない
- L5: 解消。(e) の行は mark log と合う(12:15:18 の wait は 12:15:38 に時間切れ、12:15:40 に打ち直し、12:15:40 の wait の時間切れ 12:16:00 のあとで止まった。時間切れは 12:14:14・12:14:36・12:14:58・12:15:38・12:16:00 の 5 回で、Notes の「5 回目」と合う)。Monitor の task id は「Values left out」の次に、記録しなかったことと、`wait_id` で結べることを書いた。計画の D6 が (e)〜(g) を挙げているのと合う
- L6: 解消。表に「箇条に `timeout_ms` が 2 つ」と「`--timeout-ms 0` が文末」が入り、どちらも狙った枝を通る。Atoi の文は「does not fit in an int」になった
- I1: 解消。「At 3a803a87, before stage 8b-2c, the `/director` skill told …」の過去形になった(`git show 3a803a87:.claude/skills/director/SKILL.md` の 487〜488 行が `--timeout-ms 0` と Monitor か `run_in_background` で、合う)。足した行は 75 字以内で、前後の折り返しに合う
- I2: 変えていない(INFO、直さなくてよい)。248 行は 4871 字から 5321 字になった
- I3: 変えていない(INFO、直さなくてよい)

### 新しい指摘

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (N1) | maintainability(skill の規則と記録) | 例外の段落が「知らせと wait の終わりのどちらが先に届いてもよい」と書くので、director が signal での終わりを見た時点で知らせがまだ来ていないとき、失敗にするのか知らせを待つのかが決まらない。待つ側に読むと、(g) のように知らせの来ない外からの TERM で、失敗として人に伝えず待ち続ける。cycle 1 の文(「すぐあとに」)は、その時点で決められた。この形は L2 で勧めたもので、順を書かないことだけを見て、見た時点で決められるかを見ていなかった。また、実機の確認のセッションが読んだのは 5aea657f の「すぐあとに」の文(evidence の 100 行に版がある)で、(g) の記録は「知らせは終わりの前に来なかった」と、順を決めた文への結果になっている。今の「どちらが先でも」の文と、L1 で絞った文は、実機の確認が読んだ文ではない。仕様の 98 行と tech-debt の 248 行は、確認がどの版の skill に対するものかを書かない | `.claude/skills/director/SKILL.md` 565〜571 行。`git show 5aea657f:.claude/skills/director/SKILL.md` の 565 行。`docs/evidence/director-wait-bound-livefire-2026-10-19.md` の 40・41 行、100 行 | 見た時点で決まる文にする。例は「その wait を見張っていた Monitor の期限切れの知らせが、wait の終わりを見た時点で来ている(または同じ通知に載っている)ときに限る。まだ来ていなければ失敗として扱う」(順を決めるが、(f)(g) の記録と合う)。または今の文のまま、livefire の「Not checked here」に「L1 と L2 の文は実機の確認のあとに変えた。signal での終わりが知らせより先に見えた場合は確かめていない」と 1 文足す。前者は skill の 2 つの元の写しを変えて `scripts/sync-skills.sh` を 2 回回し、表のテストに順を決めた文の 1 件を足す |

- Merge after fixes: yes(CRITICAL・HIGH なし。M1、L1、L3〜L6、I1 は解消し、L2 は相手の Monitor に結ぶ点が解消して、順の点が N1(LOW)として残る。N1 は 1 文の直しで、いまの動きを壊すものではなく、害は、外からの TERM のあとに director が人に伝えず待つ読み方をした場合に限る。skill の 4 つの写しの同一、追加行の個人情報、tech-debt の区切り、evidence と mark log の時刻には欠陥がなかった)
