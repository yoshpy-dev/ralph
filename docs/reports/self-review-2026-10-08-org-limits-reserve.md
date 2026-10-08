# Self-review report: org-limits-reserve

- Date: 2026-10-09 JST(ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Branch: feat/org-limits-reserve(HEAD 710de10d、base 51855166)
- Reviewer: reviewer subagent (Claude)、パイプライン 2 回目(cycle 2、既定の上限の最後の回)
- Scope: diff の品質だけ。cycle 1 の self-review(d8ec84da)以降の差分 `git diff d8ec84da..HEAD -- ':!docs'` の 13 ファイル(+348/-81)が対象。コードは 8dd19634(文言)、975df92b と afcbc6c2(cross-review の ACTION_REQUIRED の修正)、c3a95c48(テスト)、文書とヘルプ文は 76d1cf1c(`/sync-docs` の入力)。重点は、`validateMaxOrgs` が `ValidateOrgWideCapacity` の挙動と文言を保つか、動いていない leader の予約が上限の判定なしに記録される経路がほかにあるか、そこを plain rejection にした選択、76d1cf1c のヘルプ文とコードの一致。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査、formatter、mutation は実行していない
- 番号の付け方: cycle 1 の finding は F-1〜F-11(付録 A に原文を残す。tech-debt の行 172〜176 と verify・plan が F 番号で指している)。この回の finding は `C2-` を付ける。同じ報告の上書きで番号が付け替わらないようにするため

## Evidence reviewed

- `git diff d8ec84da..HEAD -- ':!docs'` の全行。`internal/org/envelope.go`、`internal/org/spawn.go`、`internal/config/config.go`、`internal/cli/org.go`、`AGENTS.md`、`templates/base/ralph.toml`、`templates/base/docs/quality/quality-gates.md`、skill の 4 面、`spawn_test.go` と `verbs_test.go` の追加分。コミットの親子は `git log --format='%h parent=%p'` で確かめ、一直線(8dd19634 → fbb04f83 → c3a95c48 → fff1ec23 → 76d1cf1c → 196205f8 → 6600b3a9 → 975df92b → afcbc6c2 → 710de10d)で、この回のコード変更はすべて上の範囲に入っている
- `validateMaxOrgs` の同値性: `git show d8ec84da:internal/org/envelope.go` の `ValidateOrgWideCapacity` と HEAD を並べて読んだ。条件(`len(runningOrgs) >= cfg.MaxOrgs && !slices.Contains(runningOrgs, orgID)`)、`running == ""` のときの `none`、`fmt.Errorf` の書式と引数の順、max_orgs → max_total_seats の順序、上限が 0 以下の hand-built config の扱いは同じ。変わったのは仮引数が `req.OrgID` から `orgID` になった点だけ
- 予約を記録する経路: `grep -rn 'scopeReservedEvent\|EventScopeReserved' internal cmd` で非テストの呼び出しを数えた。`checkCapacityAndStart`(`spawn.go:1040`)、`idempotentRespawn`(`:1110`)、`dryRunSpawn`(`:1704`、dry-run のイベント)、`reserveAgain`(`verbs.go:1233`、補償)の 4 つ。`idempotentRespawn` の呼び出し元は Phase 1(`:606`)と Phase 2(`:762`)の 2 つで、どちらも `Roster` の座席をそのまま渡す
- `Roster`(`manifest.go:171`)、`currentOrgLives`(`reserve.go:187`)、`RunningOrgs`(`:233`)、`reservationDecision`(`:283`)、`stateEvents` と `activeEvents`(`seat.go:59`、`:77`)、`reject`(`spawn.go:1671`)を読み、plain rejection の理由と、座席が `spawned` なのに Active でない状態を作る経路を確かめた
- 76d1cf1c のヘルプ文: `orgWideLimitsHelp` を `withMainWorktreeOrgLimits`(`cli/org.go:148`)と `ResolveOrgStateDir`(`statedir.go:59-72`)、`MainWorktreeRoot`(`:81`)に突き合わせた。`status` の `Long` は `printStatusTable` と `orgStatusJSON`(`omitempty`)に突き合わせた
- skill の差分(補償の例外、設定の読み元、末尾の `/` のないパス、`{{SCOPE}}`)をコードと突き合わせた。4 面は `cmp` で同一、`docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md` の envelope validation の行は同じ文
- 機械的な確認: `git diff --check d8ec84da..HEAD` は空。追加行に U+FFFD、`fmt.Print`、`println`、`TODO`、`FIXME` はない。出荷物(`internal`、`templates/base`、`.claude`、`.agents`、`scripts`、`AGENTS.md`)で `director` の語は 0 件
- tech-debt の行 170、172〜176 を HEAD の内容と突き合わせた。cycle 1 の推奨で足した行で、この回のコミットが事実を変えたものがないかを見た

## 依頼された 4 点の結論

1. `validateMaxOrgs` は `ValidateOrgWideCapacity` と同じ挙動、同じ文言を返す(上の Evidence)。`ValidateOrgWideCapacity` の doc も、まだ合っている。
2. 動いていない leader の予約は、上限を見ずに記録される経路が残っていない。`idempotentRespawn` は座席が Active でないとき `validateMaxOrgs` を通り、`checkCapacityAndStart` は `spawnCapacityErr` の全判定を通る。残る 2 つは、`reserveAgain`(補償。判定しないことは行 172 に記録済み)と dry-run のイベント(数えられない)。座席が `spawned` のまま Active でなくなるのは、`disbanded` が `stopped` なしで続いた古い台帳だけ。現行の `Disband` は先に `stopped` を書き、`stop_failed` は状態のイベントではないので座席は Active のまま残る。
3. plain rejection(`rejected` を書かない)の選択は適切。`rejected` は状態のイベント(`stateEvents`)なので、書くと座席の最新の状態が `spawned` から `rejected` に変わり、次の予約なしの再試行が idempotent の返しを失って、新しい spawn に進む。ただし doc の理由はこの枝に当てはまらない(C2-2)。
4. ヘルプ文は内容がコードと合う。`status` の `Long` は `printStatusTable` と JSON の `reservation`(予約がなければ出ない)に合う。`orgWideLimitsHelp` の読み元の条件は `withMainWorktreeOrgLimits` と合う。直す点は文の組み立てだけ(C2-4)。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| C2-1 LOW | readability | `idempotentRespawn` の doc で「decided first」が 2 回出て、何が先か読み取れない。1 段落目は「the reservation is decided first against the same locked events」、2 段落目は「max_orgs is decided first with validateMaxOrgs」。コードの順序は max_orgs、予約、座席を返す、の順。`Spawn` の手順 1 の「the reservation is decided first, after max_orgs when the seat is not active」も、1 文の中で「first」と「after」が食い違う。3 か所とも同じ編集で書かれ、互いに矛盾して読める | `spawn.go:345-346`、`:1081`、`:1088`、`:1100-1112` | 「座席を返す前に」の意味で書き直す。`Spawn` の手順 1: 「With Reserve, max_orgs (when the seat is not active) and then the reservation are decided before the seat is returned (idempotentRespawn).」。`idempotentRespawn` は 1 段落目の「first」を「before the seat is returned」に、2 段落目を「max_orgs is decided before the reservation」にする |
| C2-2 LOW | maintainability | 新しい枝の拒否は plain rejection で、その選択は正しい。ただし理由が書かれていない。doc の理由は「a `rejected` for the seat would replace `spawned` as its latest state event (for a running leader, showing it inactive)」だが、この枝の座席はすでに inactive なので「inactive に見える」は当てはまらない。実際の理由は、`rejected` が状態のイベントなので次の予約なしの再試行が idempotent の返しをせず、新しい spawn を始めてしまうこと。計画側も、AC1 は「台帳に `rejected` が残る」、進捗(156 行)は「AC1 にコードを合わせる直し」と書き、この枝が `rejected` を書かないことには触れていない。AC1 の文字どおりには満たさない経路が、満たしたように読める | `spawn.go:1094-1097`(doc)、`:1100-1103`、`seat.go:59-67`(`stateEvents` に `EventRejected`)、`spawn.go:1671-1676`(`reject` は `SeatID: p.SeatID` で書く)、計画 89 行(AC1)と 156 行 | doc の括弧を「for an inactive seat it would also stop the next retry from being returned as idempotent」の趣旨に差し替える。計画には「この枝は AC14 の拒否と同じく `rejected` を書かない」の 1 行を足す。どちらもコメントと計画の文なので、cap のこの回では tech-debt に送る(Tech debt identified) |
| C2-3 LOW | maintainability | `if !seat.Active` は、外しても結果が変わらない。Active な座席は `RunningOrgs` の最初のループでその org を「走っている」にするので、`validateMaxOrgs` は `slices.Contains(runningOrgs, orgID)` で nil を返す。分岐が省くのは、通ると決まっている呼び出しだけで、doc の「An Active seat's org is running, so it skips this check」もそれを書いている。そのため、(1) 読み手には Active の有無で規則が分かれるように見える、(2) この条件を外す・間違える変更を落とすテストは書けない(新しいテストは、どちらの条件でも `validateMaxOrgs` が拒否か通すかを決める)、(3) 正しさが `Active` の導出と `RunningOrgs` の定義の両方に寄りかかる。コードを読んだ結論で、条件を外した mutation は回していない(未確認) | `spawn.go:1100-1104`、`reserve.go:233-239`(`RunningOrgs`)、`envelope.go:107-108` | 条件を外し、予約つきの再試行では常に `validateMaxOrgs` を通す(費用は再試行ごとの `RunningOrgs` 1 回)。残すなら、doc の「skips this check」を「skips a call that cannot fail」と言い換え、近道であることを書く |
| C2-4 LOW | readability | `orgWideLimitsHelp` の「Only when --config is not given and the ledger is the main worktree's, those two are read from ...」は、only が文頭にあるのに語順が平叙文のままで、英文として崩れている。意味の面でも、cycle 1 の版は最初の文の中で「(no --state-dir or RALPH_ORG_STATE_DIR)」と条件を絞っていたが、今の版は後ろの「In every other case (...)」で初めて、`--state-dir <main>/.harness/state/org` のように flag で main の台帳の path を指した場合を外す。コードは source が `git-main-worktree` のときだけ読む(`MainWorktreeRoot`)ので、最初の文だけ読むと、その場合も main の `ralph.toml` を読むように取れる。同じ言い回し(「when the state dir is the main worktree's .harness/state/org」)が、この差分では触れていない `templates/base/ralph.toml` にもあり、そちらは利用者に配られる | `internal/cli/org.go:363-372`、`internal/org/statedir.go:59-72`、`:81-84`、`templates/base/ralph.toml:41-44`、cycle 1 の版は `git show d8ec84da:internal/cli/org.go` の 365-367 行 | 「Those two are read from the main worktree's ralph.toml (built-in defaults when it has none) only when --config is not given and the ledger was resolved by default to the main worktree's (neither --state-dir nor RALPH_ORG_STATE_DIR is set).」の形にして、後ろの「In every other case」は残す。`ralph.toml` のコメントも「resolved by default」を足して同じ条件にする |
| C2-5 LOW | maintainability | `/verify` と `/test` はこの回、報告を同じパスに上書きする。tech-debt の行と Go のコメントは、その中の番号を指している。行 170 は verify の V-7、V-8、行 172 は V-2、行 174 は V-4、V-5、行 176 は test の Test gaps 3〜5、7、8 と mutation X18、`verbs_test.go` のコメントは「verify V-2」。この報告は付録 A で F 番号を残すが、verify と test に同じ手当てを頼まないと、番号が別の finding に付け替わる。行 176 の(b)の「`idempotentRespawn` (87.5%)」は 975df92b が枝を足す前の値で、今の関数には合わない | `docs/tech-debt/README.md:170`、`:172`、`:174`、`:176`、`internal/org/verbs_test.go:3083-3085`、`docs/reports/verify-2026-10-08-org-limits-reserve.md` の 69、82、86、99、103 行、`docs/reports/test-2026-10-08-org-limits-reserve.md` の 96、132 行 | `/verify` と `/test` の cycle 2 に、V-1〜V-9、Test gaps 1〜8、X18 の番号を残すよう頼む(旧版を付録に残し、新しい指摘は別の接頭辞で採番する)。行 176 のカバレッジの値は関数名だけにする。Go のコメントの「verify V-2」は、挙動(「the pane compensation does not check max_total_seats」)で書き直す |
| C2-6 LOW | readability | 行 173 の(a)は F-9 を「`--reserve internal/auth` is accepted and protects only a file ... and nothing says so」と書くが、同じ行の見直しの欄は「the help text and the `/org` skill state them」と書き、skill には 76d1cf1c が足した「末尾に `/` がないパスはファイルなので、`--reserve internal/auth` が守るのは `internal/auth` という名前のファイルだけ」がある。行の中で食い違う。言いたいことは「入力したときに警告が出ない」で、その文言なら食い違わない | `docs/tech-debt/README.md:173`(説明の欄と見直しの欄)、`.claude/skills/org/SKILL.md:223-225` | 「and nothing says so」を「and nothing warns when the path is typed」に変える |
| C2-7 LOW | readability | `AGENTS.md` の repo map の「envelope validation (per-org `max_seats`; org-wide `max_orgs` / `max_total_seats` and scope reservations in `reserve.go`)」は、`in reserve.go` が 3 つ全部にかかるように読める。org-wide の判定そのもの(`ValidateOrgWideCapacity`、`validateMaxOrgs`)は `envelope.go` にあり、`reserve.go` には数え方(`RunningOrgs`、`TotalActiveSeats`)と予約がある。この文書は「map」で、grep で場所に着くことが役割 | `AGENTS.md:90`、`internal/org/envelope.go:92`、`:107`、`internal/org/reserve.go:233`、`:265` | 「org-wide `max_orgs` / `max_total_seats` in `envelope.go`, their counts and scope reservations in `reserve.go`」の形にする |

その他に数えない指摘が 1 つある。`autonomousScopeGateErr` の文(`spawn.go:1207`)は「(--reserve on the leader seat only; or --allow-unscoped to explicitly bypass)」で、セミコロンのあとの `or` が `--reserve` の注記の続きに読める。F-3 の推奨(leader 限定を文に入れる)は満たしているので、直さなくてよい。

## cycle 1 の finding の現況

| F | 重さ | この回の判定 | 根拠 |
| --- | --- | --- | --- |
| F-1 | MEDIUM | 直った | 8dd19634 が `config.go` と `templates/base/ralph.toml` から director の文を消した。出荷物で `director` は 0 件 |
| F-2 | LOW | 繰り越し(行 175 の(a))。悪化なし | 新しいコードの bool は `record`(`idempotentRespawn`)で、`reserve` の二重の意味は増えていない |
| F-3 | LOW | 直った | 8dd19634。`spawn.go:1207` が「--reserve on the leader seat only」を持つ。上の「その他」を参照 |
| F-4 | LOW | 繰り越し(行 174 の(a))。skill に断り書きが入った | 76d1cf1c が skill に `{{SCOPE}}` の説明を足し、行 174 の(a)もそれを書く |
| F-5 | LOW | 繰り越し(行 172 の(a))。悪化なし | `verbs.go` はこの回に触れていない |
| F-6 | LOW | 繰り越し(行 175 の(b))。悪化なし | `reserve.go` はこの回に触れていない |
| F-7 | LOW | 繰り越し(行 175 の(c))。悪化なし | `newOrgSpawnRuntime` はこの回に触れていない |
| F-8 | LOW | 繰り越し(行 173 の(c))。悪化なし | 同上 |
| F-9 | LOW | 繰り越し(行 173 の(a))。skill が規則を明記した | 76d1cf1c。行の文言との食い違いは C2-6 |
| F-10 | LOW | 繰り越し(行 172 の(b))。悪化なし | `CloseDeferredSelfPane` はこの回に触れていない |
| F-11 | LOW | 直った | 76d1cf1c。(1)`--config` の説明から spawn と start の文を外し、(2)`orgWideLimitsHelp` の頭を「`ralph org spawn` and `ralph org start`」にした。新しい文の組み立ては C2-4 |

## Positive notes

- 判定を `validateMaxOrgs` に切り出して、新しい枝と既存の経路が同じ関数、同じ文言を使う。テスト(`TestOrgSpawn_Reserve_InactiveLeaderChecksMaxOrgs`)の期待する文字列は `ValidateOrgWideCapacity` の書式を組み立て直したもので、2 つが食い違えば落ちる
- 新しいテストは、上限に達した場合に記録なし、receipt なし、agmsg の呼び出しなし、座席が `spawned` のまま inactive、`RunningOrgs` が org-b だけ、を確かめる。Phase 1 と Phase 2 の両方、上限 1 と 2、別の座席で org が走っている場合を含む。修正前のコードなら org-a の `scope_reserved` が増えて `lastEvent` の検査で落ちる(読みによる。mutation は回していない)
- 判定の順序が、新しい spawn(max_seats、max_orgs、max_total_seats、予約)と idempotent の枝(max_orgs、予約)で同じ向きに並ぶ。走っている org の再試行は max_orgs を通るので、上限が no-op の再試行を拒否することはない
- 補償の例外を skill に書き、pane 側の窓(`max_total_seats` を 1 つ超える)をテストで固定し、行 172 の見直しの欄に「テストと skill の文を一緒に変える」と書いてある
- skill の 4 面、`quality-gates.md` の該当行は、ミラーのどちらも同一

## Coverage gaps

- テスト、mutation、`check-sync.sh`、`check-skill-sync.sh` は実行していない(指示による)。C2-3 は読みによる結論
- ヘルプ文は `ralph org spawn --help` を出して確かめていない。Go の文字列と周辺のコードを読んだ
- dry-run は idempotent の枝を持たないので、動いていない leader(Active な leader も同じ)への `--reserve --dry-run` は max_seats と max_total_seats まで判定し、本番の枝より厳しい予測になる。cycle 1 から同じ形で、この回の差分で広がってはいない
- verify の cycle 1 の報告は、V-1(bare リポジトリの列挙)、V-2(補償の断り)、V-3(quality-gates)、V-5(`{{SCOPE}}`)、V-6(F-11)、V-9(チェックボックス)を未対応と書くが、いずれも後続のコミット(76d1cf1c、196205f8、計画の差分)で解けたように見える。この回の `/verify` が HEAD で読み直す前提で、ここでは判定しない

## Tech debt identified

この回は既定の上限の最後なので、C2-1〜C2-7 のうち直さないものは繰り延べにあたる。どれもコメント、ヘルプ文、計画、台帳の文言で、挙動は変わらない。コードのコメントを直すと全パイプラインの再実行になり、上限を超える。1 行にまとめて `docs/tech-debt/README.md` に足す案を次に書く。足すのは `/sync-docs` の担当か、オペレーターの判断。

> org-limits-reserve の文言の LOW(self-review cycle 2 の C2-1〜C2-7)。(a) `idempotentRespawn` と `Spawn` 手順 1 の doc で「decided first」が食い違い、plain rejection の理由が既に inactive な座席に当てはまらない。(b) `if !seat.Active` は `validateMaxOrgs` が org の走っている座席に通す結果と同じなので外せる。(c) `orgWideLimitsHelp` と scaffold の `ralph.toml` の読み元の条件の書き方。(d) 行 170〜176 と `verbs_test.go` が verify の V 番号、test の Test gaps、カバレッジの値を指す。(e) `AGENTS.md` の repo map の `reserve.go` の係り方。トリガー: `idempotentRespawn`、`orgWideLimitsHelp` の次の変更、または verify・test の報告が番号を付け替えたとき

C2-6(行 173 の 1 語)は、台帳の別の編集のついでに直せる。

## Recommendation

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。LOW は 7 件(C2-1〜C2-7)で、いずれもコメント、ヘルプ文、台帳、計画の文言。cross-review の修正(975df92b、afcbc6c2)は挙動、エラー文、順序が正しく、`ValidateOrgWideCapacity` の既存の挙動を変えていない
- Follow-ups: C2-1〜C2-7 は cap のため直さず、上の 1 行にまとめて tech-debt へ送るのが現実的。C2-5 の番号の保持は、直す・直さないにかかわらず、`/verify` と `/test` の cycle 2 に先に頼む。`/sync-docs` に渡すもの: 行 173 の C2-6、C2-4 の `ralph.toml` のコメント、C2-7 の `AGENTS.md`
- cycle 1 の F-1(MEDIUM)、F-3、F-11 は直り、F-2、F-4〜F-10 は行 172〜176 に載っていて悪化していない

## 付録 A: cycle 1 の finding、Coverage gaps、Tech debt(d8ec84da 時点の原文)

cycle 1 の self-review の本文のうち、tech-debt の行 172〜176 と verify、計画が F 番号と Coverage gaps の名前で指している部分を、そのまま残す。`file:line` は d8ec84da 時点のもので、`spawn.go` はこの回の差分で行がずれている(関数名で探す)。cycle 1 の `Recommendation` の節は、この回の判定と食い違わないように載せない。

### Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | maintainability | scaffold が配る `templates/base/ralph.toml` の説明と、`config.go` の doc comment が「the director is not a seat」と書く。director は `docs/specs/` と計画にしかなく、scaffold した project にも、このリポジトリの出荷物(README、skill、rules、コード)にも存在しない語で、読み手は何のことか分からない。計画の Non-goals も「director はまだなく」と書いていて、まだない機能について「数えない」と断る文になっている。director が入る 8 段目で数え方が変わると、この一文が先に古くなる | `templates/base/ralph.toml:46`、`internal/config/config.go:53`、`:58`。出荷物での `director` の語(単語境界)の grep は、この 3 か所だけ | 3 か所とも「The director is not a seat and is not counted」の文を消す(`max_orgs` は走っている org、`max_total_seats` は動いている座席、で完結している)。director を数える・数えないは、director を入れる段の計画で書く。scaffold の `ralph.toml` なので、`/verify` の前に直せば cycle は増えない |
| F-2 LOW | readability | `spawn.go` の中で `reserve` が 2 つの型を指す。入力検査(`:437`)では `NormalizeReservePaths` の結果の `[]string`、dry-run の分岐(`:500`)、`checkCapacityAndStart`(`:1032`)、`spawnCapacityErr` の戻り値(`:1063`)、`dryRunSpawn` の引数(`:1680`)では「予約を記録するか」の `bool`。隣に `p.Reserve`(`[]string`)もある。`scopeReservedEvent("", p.OrgID, p.Reserve, "", true)`(`:1686`)と `scopeReservedEvent(o.now(), orgID, paths, "restored: "+why, false)`(`verbs.go:1233`)は、文字列 2 つと `bool` を位置で渡すので、呼び出し側だけでは ts と note、dry-run の向きが読めない | `spawn.go:437`、`:500`、`:1032`、`:1063`、`:1680`、`:1686`、`verbs.go:1233`、`reserve.go:151` | `bool` 側を `recordReservation` にする。dry-run の呼び出し(`ts` が空)は、`dryRunSpawn` が `steps[i].TS` を上書きするので、`scopeReservedEvent` の引数から `ts` を外して呼び出し側で足す形でもよい |
| F-3 LOW | readability | autonomous の scope ゲートのエラー文が「requires --scope or --reserve」で、`--reserve` を使えない座席(leader 以外)にも勧める。その座席が従うと、次は「only the leader seat reserves paths ... (spawn it without reserve paths)」で拒否され、2 回目の拒否が 1 回目の案内と食い違う。`autonomousScopeGateErr` の条件は `p.SeatID` を見ない(`Reserve` を渡せるのが leader だけなので、条件としては合っている) | `spawn.go:1189`、`:430-441`、`:1177-1193` | 文を seat によって変えるか、「requires --scope (the leader seat may pass --reserve instead)」のように leader 限定を文に入れる。`autonomousScopeGateErr` は `p` を持っているので、`p.SeatID == LeaderIdentity` で分けられる |
| F-4 LOW | maintainability | 予約だけを渡した autonomous の leader は、scope ゲートを通るのに、役割プロンプトの `{{SCOPE}}` は `p.Scope` だけで描かれ、既定の「未指定(読み取り中心で、リポジトリ規約に従うこと)」になる。ゲートの目的(autonomous の座席が担当範囲を持つこと)を `--reserve` で満たしたことにしながら、座席に範囲が渡らない。計画の進捗の(c)として実装者が見つけて送る予定の項目と同じ。直す場合は `Scope` が空のとき `strings.Join(p.Reserve, ", ")` を渡す 1 行で、予約と `--scope` が一致する | `spawn.go:809`、`:1715`(`Scope: p.Scope`)、`prompts.go:21-26`、`:82`、`internal/org/prompts/leader.md:4`、計画の進捗 156 行 | 今直すか tech-debt に送るかを決める。送るなら、`/org` skill と README の「`--reserve` は `--scope` のゲートを満たす」の隣に、座席のプロンプトには範囲が出ないことを 1 文足す |
| F-5 LOW | exception-handling | `reserveAgain` に渡す `rr.Events` は、`CloseDeferredSelfWorkspace` の冒頭(`:1087` 付近)で読んだ snapshot で、最大 3 回 10 秒の herdr の呼び出しの前のもの。窓の間に同じ org_id を `ralph org start --reserve` で立て直した場合、snapshot は「予約なし」を示し、`reserveAgain` は古い予約を新しい予約の後ろに追記する。`currentOrgLives` は最後の `scope_reserved` を採るので、新しい予約が古いものに入れ替わる。重なりの検査もしない。計画のリスクに書いた窓(ほかの org が枠を取る)とは別の、同じ org_id の組み合わせ。`reactivateSeat` も同じ snapshot を使う既存の形なので、新しく持ち込んだというより同じ形を踏襲している | `verbs.go:1111`(呼び出し)、`:1215-1237`(`reserveAgain`)、`reserve.go:218`(最後が勝つ) | 追記の直前に `o.Manifest.Read()` で読み直し、そのとき `ActiveReservation` が nil の場合だけ書く。直さないなら `reserveAgain` の doc の窓の説明に、同じ org_id の立て直しも含めて書く |
| F-6 LOW | maintainability | 同じ導出が 2 つある。`currentOrgLives` の workspace の畳み込み(`DeleteFunc` してから `Created` なら追記)は `openOrgWorkspaces` と同じ処理を、最後の `disbanded` より後に限って書き直したもの。`orgLife.workspaces` は `len(...) > 0` の判定にしか使われず、「oldest first」の順序も誰も読まない。2 つが食い違うと、`disband --all` の対象(`orgsToDisband` が使う `openOrgWorkspaces`)と max_orgs の「走っている」が別々に動く。いまは `PaneID == ""`、dry-run、org 単位の 3 条件が同じで合っている | `reserve.go:159-164`、`:206-214`、`:241`、`spawn.go:1833-1850` | `currentOrgLives` が org ごとに `events[d+1:]` を `openOrgWorkspaces` へ渡す形にするか、少なくとも `openOrgWorkspaces` の doc に「`currentOrgLives` と同じ規則」と足す。`workspaces` は `[]string` のまま持つ理由がなければ `open bool` でよい |
| F-7 LOW | maintainability | `newOrgSpawnRuntime` は `newOrgRuntime` の 2 手順(`ResolveOrgStateDir`、`guardLegacyOrgStateDir`)をそのまま写している。`newOrgRuntime` に手順が増える(別の guard や警告)と、spawn と start だけが取り残される。写した側の doc は「`newOrgRuntime` と同じ」としか言わず、写しであることが読み手に見えにくい | `cli/org.go:108-114`、`:121-134` | 解決と guard を 1 つの関数(`resolveOrgLedger(cmd, stateDir, access) (dir, source string, err error)` など)に切り出して両方が呼ぶか、`newOrgRuntime` に限度の読み元を差し替える引数を足す |
| F-8 LOW | readability | 読めない `scope_reserved` を「repo 全体」として扱う選択は fail-closed で、doc comment(`reserve.go:183-186`)にも書いてある。ただし読み戻した結果は本物の `.` と区別できない。`ralph org status` は `reserved: .` と出し、`reservationDecision` の拒否は「already reserves ., so a different reservation ... is refused」と言い、`reserveAgain` はそれを `paths=.` の明示の記録として書き直す。利用者から見ると、自分が予約していない repo 全体の予約が現れ、原因の台帳の記録の破損は出ない | `reserve.go:216-219`、`:256-262`、`:289-292`、`verbs.go:1226-1233`、`cli/org.go:1109` | 影響は壊れた記録がある org の中だけなので、直さなくてもよい。直すなら、`ActiveReservation` が「読めなかった」を返し、`status` と拒否の文に「(unreadable scope_reserved record, treated as the whole repo)」を足す |
| F-9 LOW | readability | ディレクトリを末尾の `/` なしで書く(`--reserve internal/auth`)と、規則どおりファイルの予約になり、`internal/auth/token.go` や `internal/auth/` の予約と重ならない。拒否もされず、予約が付いたように見えて何も守らない。規則は計画の AC6 のとおりで、help と skill にも書いてあるが、書き間違えたときに何も知らせない。CLI は `--cwd` を持っているので、パスが存在するディレクトリなら気づける | `reserve.go:90-102`(`default: return a == b`)、`cli/org.go:372-378`(`orgReserveFlagUsage`) | CLI で、`filepath.Join(cwd, path)` が既存のディレクトリで末尾の `/` がないとき、stderr に「`internal/auth` is a directory; write `internal/auth/` to reserve its contents」を 1 行出す(予約の判定は変えない)。出さないなら tech-debt に残す |
| F-10 LOW | maintainability | 補償は workspace の経路(`CloseDeferredSelfWorkspace`)にしか予約を戻さない。`CloseDeferredSelfPane` は `reactivateSeat` だけで、`disbanded` で解けた予約を戻さない(計画の進捗の(a)と同じ)。その経路の org は、座席が動いている扱いに戻って走っている一方、範囲は誰にも守られない。AC15 は workspace の経路に限っているので計画どおりだが、同じ種類の欠けが片方に残る | `verbs.go:1010-1034`(`CloseDeferredSelfPane`)、`:1111`(`CloseDeferredSelfWorkspace` の `reserveAgain`)、計画の進捗 156 行 | 計画どおり tech-debt に送る。`reserveAgain` の doc に、pane の経路は戻さないことを 1 文足すと、次に読む人が経路の非対称を見落とさない |
| F-11 LOW | readability | help 文の 2 点。(1) `--config` の persistent flag の説明に、spawn と start だけに関わる文が入り(「without --config, spawn and start read ...」)、`--config` 自身の説明の中で「without --config」と言うので読みにくい。`stop` や `status` の help にも出る。(2) `orgWideLimitsHelp` は「Before starting a seat, spawn checks ...」で始まり、`ralph org start --help` にもそのまま出るので、start の読み手には動詞の `spawn` が何を指すか曖昧 | `cli/org.go:46`、`:361-370`、`:395`、`:531` | (1)は `--config` の説明を元に戻し、限度の読み元は `spawn` と `start` の Long にだけ書く(`orgWideLimitsHelp` に既にある)。(2)は「Before starting a seat, `ralph org spawn` and `ralph org start` check ...」にする |


### Coverage gaps

- テスト、mutation、`check-sync.sh`、`check-skill-sync.sh` は実行していない(指示による)。`/org` skill の 4 面が同じ内容であることは `diff -q` だけで見た
- 本物の herdr で、補償の窓(`disbanded` から補償までの数十秒)を作った実測はしていない。窓の挙動は `verbs_test.go` の fake で固定されたものを読んだだけ
- 同時実行のテスト(`TestOrgSpawn_ConcurrentSpawns_*`、`ConcurrentReservations_*`)は名前と構成を見ただけで、flock の下で本当に直列化されているかは再現していない
- macOS の標準の大文字小文字を区別しないファイルシステムでは、`Internal/` と `internal/` は別のパスとして比べられ、重ならない。パスの正規化は大文字小文字を畳まない。予約は助言であり(範囲外への書き込みは止めない)、計画に書かれていない扱いなので、finding にはせず、`/verify` と `/sync-docs` が判断する材料として残す

### Tech debt identified

新しい行は `/sync-docs` で足す。計画の進捗が挙げた 4 件のうち、(a)は F-10、(c)は F-4 と同じ。(b)(予約のパスの `*` はそのままファイル名として扱う)は skill に書いてあり、コードの挙動は `normalizeReservePath` が `*` を弾かないことと合う。(d)(start の `--scope` の help の表示崩れ)は既存。このほか、F-5、F-8、F-9 は直さない場合に tech-debt の候補。

`/sync-docs` に渡すもの(文書のずれは見ていないので、位置だけ):

- `docs/quality/quality-gates.md:77` は envelope validation を `max_seats` だけで書いている
- `internal/org/envelope_summary.go:36`(leader のプロンプトの `{{ENVELOPE}}`)と `internal/cli/doctor.go:788`(doctor の `[org]` の要約)は `max_seats` だけを出し、`max_orgs` と `max_total_seats` は出さない。出す・出さないは判断が要る
- `.claude/skills/org/SKILL.md:201` の「`--config` を渡したときと、台帳を flag か env で決めたとき、git の外では」は、`ResolveOrgStateDir` の 4 段目(`git-toplevel`、bare repository の linked worktree)も、main の `ralph.toml` を読まない場合に入ることを書いていない(`internal/org/statedir.go:38-44` と `withMainWorktreeOrgLimits` は `source != "git-main-worktree"` のすべてで読まない)
- 同じ SKILL の「どちらも台帳のロックの下で判定するので、同時に打った `spawn` でも超えない」は、自分の workspace の close が失敗した補償の窓では超える(計画のリスクとテスト `..._RiskWindowClearedByRetry` が固定している)ことを書いていない

