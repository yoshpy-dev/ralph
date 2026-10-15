# Self-review report: org-director-start

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest d59311f57d8e。8 段目の 8a-2)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)
- Scope: diff の品質だけ。`git diff 9cbf7080...HEAD`(HEAD 731cba34、11 コミット、35 ファイル、+6848/-301)。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs の担当なので行っていない。計画の D1〜D11(先に登録する D3、token の名前 D9、`stop --force` の D11 など)と、計画に書いた既知の穴(tech-debt の新しい 5 行)は、欠陥として数えない。

## Evidence reviewed

- 読んだ差分(コード、全体): `internal/org/director_start.go`(640 行)、`director_stop.go`(468 行)、`director.go` の diff すべて、`internal/cli/org_director_headless.go`(355 行)、`org_director.go` の diff すべて、`org.go`・`status.go`・`org_legacy_ledger.go` の diff すべて、`spawn.go`・`permissions.go`・`prompts.go`・`envelope.go`・`verbs_all.go` の diff すべて、`prompts/director.md`
- 読んだ差分(文書): `/director` skill(363 行すべて)、`templates/base/ralph.toml` の 2 か所、`AGENTS.md` の 1 行(word diff)、README の 3 か所、仕様 FR-9 の追記 1 行、`docs/tech-debt/README.md` の追加 5 行(`|` で分けた列は全行 5)
- 読んだテスト: `internal/cli/org_director_test.go` の diff すべて(削除行は、attach の `--driver` の test を attach と start の 2 つの動詞に広げた書き換えで、弱めていない)、`org_director_headless_test.go` の helper(1〜138 行)と `RefusedWithNothingWritten`・`WatchGate`、`director_stop_test.go` の `ForceWhenAGetTimesOut`。新しい test 名の一覧は 3 ファイルとも確認。残りの本文は読んでいない
- 読んだ周辺のコード: `confirmSeatPane`・`confirmOrgWorkspace`・`isNotFound`(`verbs.go`)、`herdrAgentName`・`writePromptFile`・`PromptFilePointer`・Spawn の `agentArgs` の組み立て(`spawn.go`)、`identifierPattern`(`identifier.go`)、`CommandWithStateDir`・`shellQuote`、`driver.Herdr.WorkspaceCreate`(エラー時に出力を返す癖)、`newOrgRuntimeAt` の runner の作り方、`orgInboxHintStateDir`、secret scan のパターン(`scripts/secret-scan.sh`)
- 確かめたこと
  - `git diff --check` は空。追加行に TODO・FIXME・debug 出力(`fmt.Print*`、`log.*`、`t.Log`)・panic・秘密らしい文字列はない(test の `"token":"0123456789ab"` は secret scan の generic パターン `access_token` などに当たらない)。生成物の混入もない
  - skill は 4 面が `md5` で同一(6509f6c7…)
  - **herdr に打つ破壊的な呼び出し**: `grep -rn 'PaneClose(\|PaneSendKeys(\|WorkspaceClose('` で、この diff が足したのは `director_stop.go` の 370・375・400 行の 3 か所だけ。3 か所とも `closeDirectorPane`・`closeDirectorWorkspace` の中で、呼び出し元は `closeConfirmedDirectorResources` だけ。そこへ入る道は 2 つで、`StopDirector`(137 行、`if closed.err == nil` の中)と `closeDirectorResources`(289 行、`if c.err != nil { return c }` のあと)。`--force` の道は `closed.err != nil` のまま `recordDirectorStop` へ進み、close を打つ分岐を持たない。`confirmDirectorResources` は pane と workspace の両方の label を確かめ、どちらかが確認できなければ、どちらにも何も送らない。確認できない id には C-c も close も届かない
  - label の文字列: `confirmSeatPane(wsLabel, tabLabel, paneID)` は pane の tab の label と workspace の label を文字列で比べるだけで、manifest を読まない。`confirmOrgWorkspace` も同じ。org の workspace・座席の tab の label に `_` は入らない(`identifierPattern`)ので、`ralph_director_<token>`・`director_<token>` と重ならない
  - 打つコマンドの quoting: `--state-dir` は `CommandWithStateDir` → `shellQuote` を通る。herdr の id は `directorShellWord`(英数字と `_.:-` 以外は `shellQuote`)。`{{STATE_DIR}}` は雛形の描画で `shellQuote`。`id` は `directorIDNumber` で `d<N>` に限り、token は `^[0-9a-f]{12}$` でしか畳み込みに入らない。端末への出力は `inboxField` を通る
  - 雛形: `strings.NewReplacer` は置換した文字を読み直さず、描画後の `{{` の残りを最後に断る。`{{CWD}}` は引用なしだが、雛形のどのコマンドにも入っていない
  - 並行性: `AttachHeadless`・`RecordWorkspace`・`RecordPane`・`MarkLaunched`・`markStopped` は `director.lock` の下で読み直してから決め、herdr を打つ間はロックを持たない。`markStopped` は現在の登録なら `detached` と `stopped` を 1 回の書き込みにし、`detached` を先にする理由がコメントにある。start の `abort` は、herdr が返した id を記録の成否にかかわらず閉じる(`r.reg.WorkspaceID` は返った時点で入る)。`WorkspaceCreate`・`TabCreate` がエラー時に出力を返す癖は、`= ""` に戻して扱っている
  - Spawn との引数の順: `permArgs`、`codexWritableRootArgs`、`--model`、雛形の 1 文、の順で Spawn(`spawn.go:898-923`)と同じ
  - 畳み込み: 新しい field のない 8a-1 の行はそのまま読める。headless の `attached` の `pane_id` は読まない(`registrationOf`)。`workspace_created`・`pane_created` は最初の 1 つを残す。`result()` は登録のコピーを返す
  - 8a-1 の test の書き換え 2 つ(`TestOrgDirector_AttachRefusesABadDriverBeforeReadingAnything`、`TestOrgDirectorHelp_DescribesTheRegistration`)は、元の確認を残したまま足している。`root` で動く環境の skip(`Geteuid`)も残っている。20 ms の `DriverCallTimeout` を使う test は、時間切れが起きる側を見ていて、勝ち負けの競合ではない
  - register の 5 行は `|` で分けて全行 5 列。Trigger 列に 8a・director・headless を含む行を全部抜き出して照らした(M1)
- 確かめていないもの: テスト、`gofmt`、`go vet`、go1.25.8 でのビルド。本物の herdr での動き(`_` を含む 27 字の label が返ること、実機の agent start。27 字は 15 + 12 で、初版の「22 字」は /sync-docs で訂正した)。`ralph org director start|stop` を binary で動かした出力。`director_start_test.go`・`director_test.go`・`director_stop_test.go` の本文の大半

## Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 1、LOW 6。

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1: 登録簿の行のうち、この PR の段(8a-2、headless の director)を Trigger や本文で名指ししたものが、そのまま残っている。192 行は「trigger は headless の director(`ralph org director start`、stage 8a-2)とその自動化に移る」と書いたまま、8a-2 が入ったのに判断の 1 文がない。245 行は「`pane_id` field は、stage 8a-2 の headless director が書くまで空」と書いているが、headless の `attached` は `pane_id` を持たず、pane は `pane_created` から来る(この PR が `DirectorEvent` の doc を直したのに、行は直っていない)ので、この文は偽になった。189・212・234〜241 行の「stage 8a の director」は、8a-1 の director への判断だけで、headless の director への 1 文がない | `docs/tech-debt/README.md:192`・`:245`・`:189`・`:212`・`:234-241`。`internal/org/director.go` の `registrationOf`(`reg.State, reg.PaneID = DirectorStateStarting, ""`)と `DirectorEvent` の doc。8a-1 の self-review の M2 も同じ型 | 245 行の文を「pane は `pane_created` が書く。attached の `pane_id` は promoted session の分だけ」の趣旨に直す。192・189・212・234〜241 には、8a-1 が兄弟の行に書いたのと同じ形の判断を 1 文ずつ足す(headless の director も同じ `/director` skill に従う session なので 8a-1 の判断が変わらないのか、変わるのか、行ごとに言う)。登録簿は records-only の例外の外なので、cycle 1 の fix か /sync-docs の回で入れる |
| LOW | exception-handling | L1: `director start` の help・skill・コメントは「start は、director が attached の間は何も書かず herdr も打たずに断る」と言うが、見張りの確かめ(`checkWatchGate`)が attached の確かめ(`AttachHeadless`)より前にある。見張りが動いていないときに attached の登録があると(attach は見張りなしでも通り、警告するだけ)、start は見張りを起動して `watch.log`・`watch.lock`・印を書き、stderr に `started the cross-org watch` と出してから、attached で断る。test は attached の場合に見張りの gate を差し替えて動いている状態にしてあり、この道を通らない | `internal/org/director_start.go:445-452`(`checkDirectorStart` の 514 行で gate、449 行で `AttachHeadless`)。`org_director_headless.go:61-66`(help)。`.claude/skills/director/SKILL.md:126`。`org.go:378-392`(「input checks のあと、書く前に gate」「入力で断られた start は見張りを起動しない」)。`org_director_headless_test.go:235-282`(`setEscalateWatch` で attached の case だけ見張りを受け付け状態に) | 動きは `ralph org start`(Spawn の idempotent・上限の確かめが gate のあと)と同じで害は小さい。help・skill・`org.go` のコメントを「attached の確かめの前に、見張りがなければ起動することがある」に直すか、`checkDirectorStart` で `store.Read()` を 1 回見て、current があれば gate の前に断る(AttachHeadless は引き続きロックの下で決める) |
| LOW | maintainability | L2: headless の登録を `detach --id`・`detach --force`・`attach --replace` で外すと、成功の出力は `detached d1` と `attached d2 ..., replacing d1 ...` だけで、d1 の pane と workspace が開いたままであることも、`stop` で閉じることも言わない。それを言うのは `attach` が断られたときの文(`attachedError`)と `status` の `director:` の行だけ。人の手で外す道(8a-2 の D4 で「detach は記録だけ」を保った道)の出力に 1 行がない | `internal/cli/org_director.go:193-197`(`detached %s` の 2 つの Fprintf)、`:130-137`(attach の成功の出力)。`ended`・`replaced` は `Launch` を持つので `Headless()` で分けられる。`detach --help` も headless に触れていない | `ended.Headless()`・`replaced.Headless()` のとき `its herdr pane and workspace stay open: ralph org director stop --id d1` を 1 行足す。`detach --help` に 1 文 |
| LOW | maintainability | L3: `orgDirectorLine` と `orgDirectorOpenHeadlessNote` が出す `ralph org director stop --id dN` に、この呼び出しの `--state-dir` が付かない。同じ PR の他の打ち方(`attachedError`・`startAttachedError`・`stop --all` の note・`DirectorCleanupError.StopCommand`)は `CommandWithStateDir` で付ける。`ralph status --state-dir X` と `director status --state-dir X` で、既定の台帳に効く打ち方が出る(8a-1 の「hint が既定を仮定する」型) | `internal/cli/org_director.go:492`・`:529`・`:531`。`orgDirectorLine(r orgDirectorReport)` は report だけを受け取る。`internal/cli/org.go` の `printStopAllDirectorNote`(同じ打ち方に `org.CommandWithStateDir`) | `orgDirectorLine` と note に hint を引数で渡し、`CommandWithStateDir` を通す。status の test の文字列は `--state-dir` 付きに変わる |
| LOW | readability | L4: director の stop が seat 用の確認関数(`confirmSeatPane(orgID, seatID, ...)`、`confirmOrgWorkspace(orgID, ...)`)に workspace の label と tab の label を渡して再利用していて、label の不一致のとき利用者に出る文に `seat` と `org_id` の語が混ざる(`pane "p1" is not seat "director_<token>"'s: ...`、`workspace "ws-1" is not org_id "ralph_director_<token>"'s: herdr labels it "org-a"`)。関数の doc(「seatID's pane in orgID's workspace、spawn が付けた label」)と引数名は、呼び出しが 1 種類だった前提のまま。test は外側の文と `labels it "org-a"` だけを固定していて、内側の語は固定していない | `internal/org/verbs.go:798-808`・`:809`・`:837-845`・`:850-855`・`:867-868`。`internal/org/director_stop.go:312`・`:320`(`fmt.Errorf("pane %q is not confirmed as %s's ...: %w", ..., err)` が内側の文を包む)。`director_stop_test.go:113` | 引数名を `workspaceLabel`・`tabLabel` にして doc を「label」に直し、内側の文から seat・org_id の語を外す。直さないなら、doc に「`confirmDirectorResources` も director の label で呼ぶ」と 1 行足す |
| LOW | naming | L5: package をまたぐためだけの pass-through が 2 つできた。`ModelAllowedForRole` は未公開の `modelAllowedForRole` を返すだけ。`DirectorRoleInputErr` は `directorRoleSpawnErr` を返すだけで、名前も `...InputErr` と `...SpawnErr` で揃っていない(同じ検査が 2 つの名前を持ち、test は未公開の側を叩く)。既存の `RetiredRoleInputErr` は 1 つの公開関数で、この形が前例 | `internal/org/envelope.go:149-153`、`internal/org/spawn.go:1428-1441`、`internal/cli/org.go:690`、`internal/cli/org_director_headless.go:178`。呼び出しは `modelAllowedForRole` が 2 か所(`envelope.go:59`、`envelope_summary.go:77`) | 元の関数を公開名に変える(`directorRoleSpawnErr` → `DirectorRoleInputErr`、`modelAllowedForRole` → `ModelAllowedForRole`)。呼び出し側の置換だけで済み、wrapper が要らない |
| LOW | readability | L6: コメントの手入れが残っている。(a) 折り返しが片方だけ直った長い行(他の行は 72〜80 桁)が 4 か所: `director.go:311`(115 桁)・`director.go:521`(88 桁)・`permissions.go:158`(88 桁)・`org.go:357`(86 桁)。(b) `directorRoleSpawnErr` の doc が `Like` で行を切る(`spawn.go:1425`)。(c) `notRestoredError` の doc が「attach or detach」のまま、start・stop も通る(`org_director.go:345`) | 各行は左の path:行。(a) は編集箇所の途中だけ詰め直した形 | 4 行を詰め直し、(b)(c) を直す。1 コミットで足りる |

## Positive notes

- 破壊的な 3 つの呼び出し(C-c、pane close、workspace close)が、label の確認を通った道にしかない。`--force` は確認できないものに何も打たず、記録だけを書く。確認の失敗、時間切れ、label の違い、id の振り直しを、test が別々に固定している
- 先に登録して herdr を打つ間はロックを持たない形が、コードと doc コメントで一致している。start の後始末が、herdr の返した id を記録の成否に関係なく閉じる(`TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade`)
- 畳み込みが、新しいイベントを知らない古い binary でも壊れない形で書かれ、`detached` を `stopped` より先に書く理由(読み手が間の状態を見ても止められる側に倒れる)がコメントにある
- 新しい出力はすべて `inboxField`、JSON は encoder、描画は `shellQuote` を通る。hand edit した `director.jsonl` から端末と shell に値が漏れる道は見つからなかった
- 8a-1 の test の書き換えが「文の更新」と test のコメントに明記され、確認は残っている。main worktree の `ralph.toml` で読む規則は、`withMainWorktreeDirectorStartConfig` に 4 つの key だけを移す形で、他の key(leader が書ける設定)は動かさない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(登録簿には追記していない。M1 は新しい debt ではなく既存の行の書き換えで、L1〜L6 は小さな差分で直せる。cycle 1 の fix で直さず最後の cycle に届いた LOW だけを、1 つの行にまとめて追記する。)_

## Coverage gaps

- 実機の動き: herdr 0.7.5 が `ralph_director_<token>`(27 字、`_` を含む。初版の「22 字」は /sync-docs で訂正した)の label をそのまま返すこと、claude と codex の agent start が通ること、`ralph org director start|stop` の出力を見ていない。計画の Evidence to capture と AC13 のとおり /test の担当
- L1 の道(見張りなし、director が attached、start)を通る test がない
- `internal/org/director_test.go`(+958 行)ほか 3 つの新しい test ファイルの本文の大半は読んでいない。test 名の一覧と、拒否・時間切れ・label の違いの test だけを読んだ

## Recommendation

- Merge: yes(CRITICAL・HIGH はなし。MEDIUM 1 件は登録簿の文の直しで、コードの欠陥ではない。LOW 6 件は help の文言、hint、命名、コメントで、herdr に打つ処理と登録の畳み込みに欠陥は見つからなかった)
- Follow-ups:
  1. M1: 登録簿の 245 行の文を直し、192・189・212・234〜241 に 8a-2 の 1 文を足す。cycle 1 の fix か /sync-docs の回で入れる(登録簿は records-only の例外の外)
  2. L1: help・skill・`org.go` のコメントの文を直すか、`checkDirectorStart` で gate の前に current を見る。どちらでも skill の 4 面を同時に直す
  3. L2・L3: 出力と hint に `--state-dir` と pane が開いたままの 1 行を足す。status の test の期待文字列が変わる
  4. L4〜L6: 小さな差分なので、fix の回にまとめて入れるとよい。直さずに最後の cycle に届いたら、1 つの行にまとめて登録簿に足す
  5. Go のコメントの `docs/plans/active/2026-10-15-org-director-start.md` は、`/pr` の archive で古くなる。8a-1 のときと同じ形で、指摘としては数えない
- Known gaps: テスト、`gofmt`、`go vet`、go1.25.8 でのビルドは確かめていない。herdr の実機の動きと CLI の実際の出力は見ていない。新しい test の本文の大半は読んでいない
