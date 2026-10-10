# Self-review report: org-inbox-escalate

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-10-org-inbox-escalate.md(承認済み、digest b09718e048cd)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)
- Scope: diff の品質だけ。`git diff 382c18c8...HEAD`(6 コミット、21 ファイル、+4348/-102)。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない。ただし依頼された「雛形と /org skill の文言が、コードのしていることと合っているか」は、文言の根拠をコードで確かめた
- 番号の付け方: MEDIUM は M1〜M2、LOW は L1〜L9

## Evidence reviewed

- 全文を読んだ非テストのコード: `internal/org/inbox.go`、`internal/org/escalate.go`、`internal/org/lockfile.go`、`internal/org/protocol/protocol.go` の `Parse` / `Validate`(escalate が頼る検査の中身)
- 差分を読んだ非テストのコード: `internal/cli/org.go`(escalate、inbox の 4 動詞、`wait --inbox`、表示の補助関数、`newOrgRuntimeAt` の配線)、`internal/org/watch.go`(`escalationRecord`、`osascriptNotify` の切り出し)、`internal/org/spawn.go`(`Org` の 4 欄)、`internal/cli/org_legacy_ledger.go`、`internal/cli/main_test.go`
- 文言の根拠をコードで確かめた文書: `internal/org/prompts/leader.md`、`.claude/skills/org/SKILL.md`(差分全体)、`README.md`、`AGENTS.md`、仕様 FR-5 の注記。skill の 4 面は、追加行を取り出して 4 面で同じであることを確かめた
- テスト: `internal/org/prompts_test.go` の差分、`internal/cli/org_inbox_test.go` の先頭の補助関数と `wait --inbox` のテスト、`internal/org/escalate_test.go` の fixture と `WaitInbox` のテスト。タイミングに頼る箇所(200ms の遅延と 150ms の下限、300ms の期限と 50ms の遅延)は余裕がある
- `withFileLock` の 3 つのエラー文は、`what="manifest"` のとき改称前の文と一字ずつ同じ
- 機械的な確認: `fmt.Print` 系のデバッグ出力、TODO、FIXME、secret らしい文字列は追加行にない。`inbox.jsonl` に書く経路は `appendLocked` の 1 つで、呼び出しは `Escalate` と `update` のロックの中の 2 か所だけ
- `docs/tech-debt/README.md` は差分にない。差分が触れた箇所を指す行を探して照合した(M2)

## 依頼された点の結論

1. 次の ID の割り当て(`inbox.go` の `foldInbox`、`seeN`、`nextID`): 生の行の正規表現と、JSON として読めた行の `id` の両方から最大を取る。本文に入る `"id":"e9"` は JSON のエスケープで `\"id\"` になり正規表現に当たらないので、本文で次の ID が押し上がることはない。`e0`・`e01`・`E1`・`e+5` は `inboxIDNumber` が往復で弾く。int64 の上限は `nextID` がエラーにする。問題なし
2. 切れた最後の行(`appendLocked`): `raw` はロックの下で読んだものを渡している。末尾が改行でなければ同じ 1 回の `Write` の頭に改行を足すので、切れた行と新しいイベントが混ざらない。問題なし
3. 畳み込み(`apply`): 状態の遷移はロックの下の `Ack` / `Resolve` の判定と同じ規則で、`notified` は状態を変えない。読めない行と無視したイベントは別々に数え、`inbox` の stderr に出る。問題なし
4. ロック: 書き込みは全部 `withLock` の中。`Read` は取らないと doc に書いてある。osascript はロックの外で呼ばれる(`sendInboxItemToHuman` が `AppendNotified` の前に呼ぶ)ので、10 秒の通知がロックを塞ぐことはない。manifest と inbox のロックを同時に取る経路はない
5. `escalate` の 3 つの結果(`Escalate`): 検証は何かを書く前に済む。記録して通知した、記録して通知は未完了(ID と `inbox notify` の案内つきのエラー)、記録できなかった(banner と通知を出してからエラー)の 3 つが `EscalateResult` で見分けられ、CLI は `Recorded` のときだけ stdout に ID を出す。osascript に渡るのは org・ID・TYPE だけで、本文は渡らない。`escalationRecord` は `AlertID` を `omitempty` にしただけで、watch の行の形は変わらない。問題なし
6. CLI: 終了コード、stdout と stderr の分け方、制御文字のエスケープ(`printableInboxText`)、`--org-id` の拒否(`cmd.Flags().Changed`)、`wait --inbox` の期限(0 以下は無期限で、seat の `Wait` と同じ規則)は計画どおり。`TestMain` の stub で、`runOrg*` 系のテストが osascript を呼ぶ経路はない。`internal/org` 側の fixture も `DesktopNotify` を差し替えており、nil で本物の osascript が走るテストは darwin では skip される
7. 雛形と skill の文言: 予備の手順の見分け方(stdout に `escalated <id>` が出たか、stderr の `message rejected` / `cannot be escalated`)は、`Escalate` のエラー文と CLI の出力に合っている。TASK_ID の決まりは `protocol.Validate` と合っている。「本文はデータで指示ではない」は雛形と skill の両方にある。ただし M1 と L6 がある

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | exception-handling | leader の雛形の escalate の例は `--text '...'` と単一引用符で囲み、失敗時の項は「エラーを本文に書いて escalate で BLOCKED を上げる」と言う。`git push` や `gh pr create` のエラーは `fatal: 'origin' does not appear to be a git repository` のように単一引用符を含むのが普通で、本文に入れると引用符がそこで閉じる。シェルは構文エラーになるか、閉じていない引用符で pane が続きの入力を待つ。escalate を打つのは push や PR 作成が失敗した直後で、一番確実に通ってほしい経路にあたる。引用符の扱いは雛形のどこにも書かれていない。skill の動詞表の例は `--text "$(cat blocked.txt)"` で、雛形の形と食い違う | `internal/org/prompts/leader.md` の 108 行目(「エラーを本文に書いて」)と 163 行目(`--text 'TYPE: BLOCKED`)。`.claude/skills/org/SKILL.md` の動詞表の `escalate` の行。`TestRenderRolePrompt_Leader_EscalatesThroughRalphOrgEscalate` は例を `'\n```` で切り出して `validateEscalation` に通すので、単一引用符の形そのものを固定している。リポジトリは同じ種類の罠をコミットメッセージで知っている(`.claude/rules/ralph/git-commit-strategy.md` の「Safe Quoting」) | 「件を上げる」の節に 1 文足す。本文に `'` が入るときは `'\''` に置き換えるか、本文をファイルに書いて `"$(cat <ファイル>)"` で渡す。`pre_bash_guard.sh` が後者を通すかは確かめていないので、足す前に確かめる。例そのものは今の形でよい |
| M2 | MEDIUM | maintainability | S4 は tech-debt 台帳の行「Small findings of org-feature-worktree」の (f) と (j) を直した(計画の進捗にも「(f) と (j) が直る」と書いてある)。`TestRenderRolePrompt_Leader_FeatureOrgProcedure` の改行に頼る比較を、空白を除いて比べる `containsPhrase` に替え、(f) の 2 つの比較も (j) の `--state-dir` の比較も改行の位置に依存しなくなった。ところが差分は `docs/tech-debt/README.md` を触れておらず、(f) と (j) は台帳で開いたまま。行の Debt・Impact・Trigger の各欄が (f) と (j) を、Why deferred の欄が (j) を名指ししているので、1 か所では済まない | `git diff --stat 382c18c8...HEAD` に `docs/tech-debt/` がない。台帳 193 行目の (f)「pins two phrases that contain a line break」、(j)「N1 came back in a third form」、Impact の「(f) a template re-wrap breaks a test for no behavioral reason」、Trigger の「(f), (j): the next edit of the feature-org section of `leader.md`」。直した側は `internal/org/prompts_test.go` の `squashSpace` / `containsPhrase` と、`FeatureOrgProcedure` の 8 か所の置き換え | (f) と (j) を `~~...~~ (RESOLVED ...)` で閉じ、Impact・Why deferred・Trigger の (f)・(j) への言及も同じ形で閉じる。閉じる文には、直し方が trigger の案(単一の空白で結ぶ)と違い、空白をすべて除く形であることを書く。この PR の `/sync-docs` で足してもよい。その場合は /sync-docs への依頼にこの 1 件を入れる |
| L1 | LOW | exception-handling | `ralph org escalate` は、検証に通る前に `newOrgRuntime` で実行環境を丸ごと組む。`inbox.go` と `escalate.go` が使うのは `Inbox`・`EscalationsPath`・`DesktopNotify` だけだが、`newOrgRuntimeAt` は先に `resolveOrgConfig` を呼ぶので、`ralph.toml` が壊れていると `org: load config: ...` で終わり、banner も desktop 通知も出ない。計画の AC2b は「台帳が壊れた場面でも leader の件が人に届くように」と言う。旧い台帳の拒否はほかの動詞と同じにする計画の決定なので問題ないが、config の失敗は計画に書かれていない。機能ごとの org の leader は機能の worktree で動き、その機能が `ralph.toml` を触ることはありうる。雛形の予備の手順(pane に書いて止まる)があるので、悪くても今の動きと同じ | `internal/cli/org.go` の `newOrgEscalateCmd`(`newOrgRuntime` を呼んでから `rt.Escalate`)、`newOrgRuntimeAt`(`resolveOrgConfig` が先頭)、`resolveOrgConfig`(`config.Load` のエラーをそのまま返す)。`internal/cli/org_inbox_test.go` に `--config` や壊れた `ralph.toml` のケースはなく、この経路はどのテストも通らない | `escalate` と `inbox notify` は `Org{Inbox, EscalationsPath, DesktopNotify}` だけを組む、または config の失敗をこの 2 つの動詞では banner と通知つきの失敗にする。直さないなら skill の「受信箱」節に「`ralph.toml` が読めないときは何も届かない」と 1 文書く |
| L2 | LOW | exception-handling | 復旧の案内が `--state-dir` を落とす。`Escalate` のエラー(escalate.go:117)と `NotifyInboxItem` のエラー(同 146)は `ralph org inbox notify <id>` と言うが、escalate は `--state-dir` つきで打たれる(機能ごとの org の leader は必ず付ける)。案内の通りに打つと既定の台帳を読む。既定の台帳が同じなら通るが、別版の `ralph` で台帳が分かれたとき(台帳 189 行目の行)や `RALPH_ORG_STATE_DIR` が違うときは、ID が連番なので別の台帳の同じ ID(`e5`)に当たりうる。リポジトリには、この落とし穴を避ける前例がある | 前例: `internal/cli/org.go` の `orgReadCommandHint`(`--state-dir` が明示されたときだけ付ける、その理由の doc)。落とすコード: `internal/org/escalate.go:117`、`:146`。雛形は 187 行目で正しい形(`--state-dir <台帳> <id>`)を書いているので、エラー文だけが食い違う | org 層はフラグを知らないので、CLI 側で `--state-dir` が明示されたときにエラーへ `(--state-dir <path> を付けて)` を添える。直さないなら、雛形が正しい形を言っていることを理由に tech-debt に 1 行 |
| L3 | LOW | readability | `--org-id` の説明が読みにくくなった。前は「(required, except for stop --all and disband --all; ...)」。今は「(required, also by escalate; stop --all, disband --all, the inbox verbs, and wait --inbox refuse it; ...)」で、「also by escalate」は、ほかの動詞もすべて required なので、何と比べているのか分からない。拒否する側の一覧は正しい(`stop --all` と `disband --all` は `rejectFlagsWithAll` で拒否する) | `internal/cli/org.go:48`。拒否の確認: `rejectFlagsWithAll(cmd, "stop", "org-id", "seat")`(1042 行目付近)、`rejectFlagsWithAll(cmd, "disband", "org-id")`(1403 行目付近)、`newOrgInboxRuntime`、`runOrgWaitInbox` | 「required, except stop --all, disband --all, the inbox verbs, and wait --inbox, which refuse it; start --plan defaults ...」のように、前の「except」の形に戻す |
| L4 | LOW | naming | 同じ語が 2 つの意味で使われている。(a) `InboxEvent.Body` / `InboxItem.Body`(JSON の `body`)は typed message の全文(`TYPE:` の行を含む)だが、`protocol.Message.Body` は header のあとの部分。`inboxSummary` は `protocol.Parse(item.Body)` の `m.Body` を使うので、1 つの関数の中で `body` が 2 つの意味になる。(b) `InboxStore.Escalate` は記録だけ、`(*Org).Escalate` は検証・記録・人への通知までで、同じ名前で契約が違う。`grep Escalate` は watch の `escalate` 系も含めて 3 種類に当たる(`architecture.md` の grep できる名前) | `internal/org/inbox.go:79`(`Body string`)と `InboxEscalation` の doc(「the whole message text as Body」)、`internal/cli/org.go` の `inboxSummary`、`internal/org/inbox.go:166` と `internal/org/escalate.go:101` | JSON のキーは出荷後に変えにくいので、いまのうちに決める。`body` を全文のまま残すなら doc に「header を含む」と書く。`InboxStore.Escalate` は兄弟の `AppendNotified` に合わせて `AppendEscalated` にする |
| L5 | LOW | maintainability | `internal/cli/org.go` が 1646 行から 2109 行になった。足した約 460 行(escalate、inbox の 4 動詞、補助の関数と型 10 個)は、ほかの部分と依存しないひとまとまりで、テストは最初から `org_inbox_test.go` に分けてある。`org_legacy_ledger.go` が同じ分け方の前例。グローバルの code-review の規則(ファイルは 800 行未満)には元から届いていないが、分けやすい塊を一番大きいファイルに足している | `wc -l internal/cli/org.go`(2109)、`git show 382c18c8:internal/cli/org.go` の行数(1646)。足した塊は `newOrgEscalateCmd`(1714 行目)から `yesNo`(2104 行目)まで | `internal/cli/org_inbox.go` に移す(コードは変わらない)。fix のパスが再実行になるなら一緒にやる。やらないなら tech-debt に 1 行 |
| L6 | LOW | readability | leader の雛形で「受信箱」が 2 つのものを指す。新しい節は「org の受信箱は台帳の下の `inbox.jsonl` ... agmsg の受信箱とは別のもの」と書くが、同じ雛形の「スター型トポロジのルール」(128 行目「受信箱(agmsg)経由で確認し」)と見出し「## 受信箱の運用」(196 行目、中身は agmsg)は元のまま。skill では同じ曖昧さを、確認の文を「agmsg の受信箱の確認は」に直して避けている。雛形だけが直っていないので、leader が「受信箱の運用」を読んで org の受信箱のことと取りうる | `internal/org/prompts/leader.md:128`、`:196`、`:151` 以降の新しい節。skill 側の直し: `.claude/skills/org/SKILL.md` の「agmsg の受信箱の確認は `/agmsg` skill の手順に従う」 | 見出しを「## agmsg の受信箱の運用」に、128 行目は「agmsg の受信箱」に直す。見出しをテストが固定していないかを先に確かめる |
| L7 | LOW | readability | skill の「機能ごとの org」の段落を、`escalate・inbox notify` を足すために折り直したが、最後の 1 行が折り直されていない。折り直した 3 行は表示幅 71〜75 で、最後の行だけ 104。雛形側の同じ折り直しは最後の行が 78 で、近隣の 72〜77 に収まっている。skill は 4 面に同じ行があるので、4 面とも同じ見た目になる | `.claude/skills/org/SKILL.md:453`(`pane に渡さないので、leader の側では当てにできない。付けないと leader の座席が別の台帳に入り、予約と上限`)。ほか 3 面も同じ行 | その行を 2 行に折る。`scripts/sync-skills.sh` で写しを作り直す |
| L8 | LOW | maintainability | 2 つ目の使い手が増えたのに doc や名前が 1 つ目のまま。(a) `manifestLockTimeout` と `manifestLockPollInterval` の名前と doc は「`withManifestLock` が待つ」と言うが、`withFileLock` を通る inbox のロックも同じ値を使う。(b) `Org.Inbox` の doc は「Escalate が記録し NotifyInboxItem が読む」と 2 つの使い手を挙げるが、`WaitInbox` も読み、nil のときエラーを返す。(c) `EscalationsPathIn` を足したが、同じパスを作る `watch.go` の 452 行目は `filepath.Join(p.StatusDir, EscalationsRelName)` のまま | `internal/org/lockfile.go:20-31`(定数の doc)と `:60-80`(`withFileLock` が 2 定数を使う)、`internal/org/spawn.go` の `Org.Inbox` の doc、`internal/org/watch.go:452` | (a) 定数の doc に「inbox のロックも使う」と 1 文足す(名前は変えなくてよい)。(b) 「`WaitInbox` も」と足す。(c) `EscalationsPathIn(p.StatusDir)` に替える。どれも挙動は変わらない |
| L9 | LOW | security | 人への経路が、ファイルから読んだ値を検証も escape もせずに外へ出す。`NotifyInboxItem` は `item.OrgID` と `item.Type` を `inbox.jsonl` から読み、`sendInboxItemToHuman` が banner(stderr)と osascript の文に入れる。`Escalate` の経路は書く前に `ValidateIdentifier` と TYPE の列挙で検査するので、手で編集した `inbox.jsonl` だけが届く。CLI の `inbox` / `show` / `wait` は同じ値を `printableInboxText` で escape しており、banner だけが漏れている。`InboxStore.Escalate` も `OrgID` と `Type` が空でないことしか見ない | `internal/org/escalate.go:236`(banner)、`:238`(`inboxDesktopNotify(fmt.Sprintf("org %s raised %s (%s)", orgID, id, typ))`)、`:145`(`item.OrgID`、`item.Type` を渡す)。`internal/org/inbox.go:167`(空の検査だけ)。CLI 側の escape: `internal/cli/org.go` の `printableInboxText` | `NotifyInboxItem` で `ValidateIdentifier("org_id", item.OrgID)` と TYPE の列挙を確かめてから送る(通らなければ「手で編集された件」としてエラー)。または `InboxStore.Escalate` に同じ検査を入れて、書く側で守る |

## Positive notes

- ID の割り当てが生の行を読むこと、切れた行の改行、`withFileLock` の共有化が、それぞれ理由つきの doc と、外すと落ちるテストで守られている。`withManifestLock` のエラー文は改称前と一字ずつ同じ
- escalate の 3 つの結果が `EscalateResult` の 3 つの欄で見分けられ、CLI が `Recorded` だけで stdout を決めるので、leader の予備の手順(stdout の `escalated <id>` の有無で分ける)がコードと 1 対 1 で合っている
- osascript に本文を渡さないことが、コードの組み立て(org・ID・TYPE のみ)とテスト(`the body must not reach osascript`)の両方で固定されている
- 端末に出す受信箱の文字列は全部 `printableInboxText` を通り、`--json` の警告は stderr に出るので、機械向けの出力が汚れない
- `--org-id` を黙って無視せず拒否する決定(計画からのずれ)が、計画の進捗に理由つきで記録され、help・雛形・skill・仕様の 4 か所に書かれている
- 雛形のテストを空白を除く比較に替えた変更(`squashSpace` / `containsPhrase`)は、それ自体にテストがあり、否定側(「人に上げる」が残っていないこと)の比較も同じ形になっている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (なし) | | | | |

_(このレビューでは行を足していない。M1・M2 と LOW 9 件は、直してから cross-review に進むことを勧める。M2 は台帳の既存の行を閉じる作業で、新しい行ではない。cycle 2 の時点で直されずに残る LOW は、そのとき 1 行にまとめた tech-debt の行が要る。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 2 件は PR の前に直すことを勧める。どちらも挙動を変えない文言と記録の直しで、M1 は雛形 1 文と skill の例の合わせ、M2 は台帳の行の閉じ方)
- Follow-ups:
  - M1: leader.md の「件を上げる」の節に、本文に `'` を含むときの書き方を足す(skill の動詞表の例と合わせる)
  - M2: 台帳 193 行目の (f) と (j) を閉じる(`/sync-docs` に回してもよい)
  - L1・L2・L9: escalate が人に届く条件に関わるので、直すなら M1 と同じ fix のパスで。L9 は書く側(`InboxStore.Escalate`)で守るのが一番小さい
  - L3〜L8: 文言・名前・ファイルの置き場所で、挙動は変わらない。L5(`org_inbox.go` への移動)は、fix のパスが再実行になるなら一緒に
  - 直したあとは /self-review から打ち直す。cross-review の前の直しなので、pipeline の cycle は増えない
