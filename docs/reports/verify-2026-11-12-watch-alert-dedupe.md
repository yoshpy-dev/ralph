# Verify report: 見張りの ALERT を 1 件にまとめ、leader の stall と予約の中の scope_change を上げない(#307 の前半)

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(digest `e5a18ef9477a`、`./scripts/plan-visual.sh digest` の値と `- Approved:` の行が一致)
- Verifier: verifier subagent(Claude Code)、pipeline cycle 3/3。cap は cycle 2 の cross-review(`7aee125e`)で 3 に上がり、`cycle-count.json` の `cycle` は 2 のままなので、lead の指示どおり insight event は `--cycle 3` で足した。cycle 2 の報告(`0a957745`)をこの報告で上書きした。前の cycle の所見は末尾の「Cycle 2 の記録」と「Cycle 1 の記録」に ID ごとに残した。tech-debt の行が指す V6、V11、V12 もそこにある
- Scope: branch 全体は `git diff 028a8f6d...HEAD`(HEAD `521edd5b`、34 ファイル、+5135/-171)。cycle 3 で新しく読んだのは、コードの 2 commit(S10 `ec538952` は cycle 2 の cross-review の 1 件、S11 `e90c85da` は cycle 3 の self-review の C3-1)と、記録の commit(`7aee125e`、`72d7d6ad`、`65a82b01`、`bfbf0a2a`、`521edd5b`)。`0a957745` のあとの `internal/` の差分は `watch_scope.go` と `watch_scope_test.go` の 2 ファイルだけで、+102/-27(`git diff --numstat`)。cycle 2 の /test(`fe4eff60`)と /sync-docs(`0cc32e6d`、`74e6b1db`、`cd4a9268`)は、それぞれの報告が扱うので、ここでは前の cycle の所見の処理を確かめるのにだけ使った。origin/main は `028a8f6d` のまま(`git ls-remote`)なので、base との衝突はない
- Evidence: `docs/evidence/verify-2026-11-12-watch-alert-dedupe.log`(git が無視する)。cycle 1 と cycle 2 の内容のあとに、cycle 3 の静的解析と probe を足した。足す前の 23623 バイトが先頭にそのまま残ることを `head -c | cmp` で確かめた。スクリプト自身の写しは `docs/evidence/verify-2026-11-12-020648.log`

## Spec compliance

計画の改め方(D6 の rebase と AC15、L1〜L3、N5、S8 の細部)は cycle 2 までと同じく Progress checklist に記録されている。S10 と S11 も Progress の 160 行と 161 行にある。下の判定は、これらで改めた計画に照らした。テストの欄は、テストがあり assert が AC を言っていることまでを読んで確かめた。テストは走らせていない(/test の役目)。

S10 と S11 が変えたのは `scopeCommittedPaths` の 2 つの引数列と doc、テストの fixture と git の fake だけなので、AC9 と AC15 を中心に、同じ道を通る AC8 と AC14 を読み直した。ほかの AC の道(`watch.go`、`watch_recurrence.go`、`inbox.go`、`internal/cli/`)は `3bb1828b` のあと変わっていないので、行番号も cycle 2 の値のまま使える。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 open か acked のあいだは件が 1 つで、立つたびに `recurred` が 1 つ増え、show と `--json` に回数と最後の時刻が出る | Met | cycle 3 で変わっていない。`internal/org/inbox.go:398-420`(`AppendRecurred`)、`watch.go:1307-1322`、`watch_recurrence.go:112-140`、`internal/cli/org_inbox.go:482-502`。テスト `watch_test.go:3126`(5 回立って件 1、recurred 4)、`TestInboxRead_RecurredCountsFiringsWithoutChangingTheState`、`TestOrgInbox_ShowsTheRecurrences` |
| AC2 resolved のあとは新しい件 | Met | cycle 3 で変わっていない。`watch_recurrence.go:75-83`、`:130-136`。テスト `watch_test.go:3197-3208` |
| AC3 続きのあいだは leader に agmsg の ALERT を送らない | Met | cycle 3 で変わっていない。`watch.go:1315-1317`、`foldIntoPendingAlert` と `leaderActedSinceAlert`(`watch.go:1760-1775`)。テスト `watch_test.go:3551`、`:3650` |
| AC4 deadman の前の再発は pending を増やさず、escalate した件に回数が残る | Met | cycle 3 で変わっていない。`watch_recurrence.go:187`、`watch.go:1887-1895`。テスト `watch_test.go:3225`、`:3551` |
| AC5 watch の立て直しで記録が無くても、同じ鍵の片付いていない件に `recurred` を足す | Met | cycle 3 で変わっていない。`watch_recurrence.go:118-139`、`watch.go:1875-1880`、`escalateAsRecurrence`。テスト `watch_test.go:3299`、`:3392`、`:3699`、`:3832`。残る窓は tech-debt の 462 行の (7) |
| AC6 片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たず受信箱にも入らない | Met | cycle 3 で変わっていない。`watch.go:1155-1195`、`:1257`。テスト `watch_test.go:3975`、`:4027` |
| AC7 ほかの座席が `working` で文字列が変わるあいだは立たず、変わらなくなれば今までどおり立つ | Met | cycle 3 で変わっていない。`watch.go:1036-1052`、`seatBusy`(`:1197`)。テスト `watch_test.go:4076`、`:4119`、`:4151` |
| AC8 予約の中だけと、自分の新しい計画と報告だけの変更では立たない | Met | 道は `watch.go:1274-1277`、`watch_scope.go:349-370`(`dropOwnRecords`)、`:124`(`ownRecordCandidate`)。S10 で、座席の cwd が下のディレクトリで利用者の `diff.relative=true` のときも、予約の中の commit のパスは root からの相対で返り、予約の中と読まれる(`watch_scope.go:296`、`:298` の `-c diff.relative=false`)。前はこの形で `internal/org/x.go` が `x.go` になり、予約の外と誤って立った。新しいテスト `TestWatch_ReservedScope_CommitPathsStayRootRelativeUnderDiffRelative`(`watch_scope_test.go:748`)は、`log` と予備の `diff` の 2 つの場合で、予約の中の commit で ALERT が 0 件なことと、`log` の場合は下のディレクトリの cwd でも自分の新しい計画で 0 件なことを確かめる。既存のテスト `:397`、`:435` は変わっていない |
| AC9 予約の外の変更で立ち、本文に外のパスが出る。外から中への移動を 1 周期で commit しても移動元が出る | Met | `watch_scope.go:295-305`。S10 と S11 で、2 つの引数列の先頭に `-c diff.relative=false` が付いた(本番の argv は `realGitRun` が前に `--no-optional-locks` を足した形)。同じテスト `:748` の後半は、予約の外の commit(`cmd/y.go`)について ALERT が 1 件で、本文の外のパスがちょうど `cmd/y.go` であることを `slices.Equal` で確かめる(`wantOneScopeAlert`、`:383-392`)。前の形では cwd の外の兄弟のパスが一覧から落ち、この ALERT が出なかった。座席の `Worktree` は spawn の `Cwd` そのもの(`spawn.go:1304`)で、テストも `p.Cwd` を下のディレクトリにしているので、本番の呼び方と同じ形を見ている。既存の `:483`(7 つの場合)と `:662`(merge の衝突の解決)はそのまま |
| AC10 予約のない org は今までどおり | Met | cycle 3 で変わっていない。`watch.go:1279-1287`(`gitStatus`)。テスト `watch_scope_test.go:1285` |
| AC11 除く規則が役割 prompt の書かせる `docs/` のパスを覆う | Met | cycle 3 で変わっていない。`TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite`(`watch_scope_test.go:1363`) |
| AC15 rebase で入った upstream の commit では立たない | Met | `--not <ref>` の形は変わらない(`watch_scope.go:298`)。テスト `:602`(rebase と merge)、`:642`、`:694`。`RunsOnlyTheGitCallsItNeeds`(`:1141`)の 4 つの固定文字列は、`-c diff.relative=false` を含む今の argv に合わせてある(`:1215`、`:1250`、`:1265`、`:1277`)。fake は `scopeSubcommand`(`:1095`)で先頭の `-c <key=value>` を飛ばして subcommand を読む。`gitRun` を差し替える所は `watch_scope_test.go` の `:854` と、`scopeStubGit` を作る `:908`、`:1156`、`:1294` だけで、`git grep` でほかのテストに git の fake は無い |
| AC14 分岐点にあった記録の変更と、分岐点が分からないときは立つ | Met | cycle 3 で変わっていない。`watch_scope.go:359-365`(ref が無いか merge-base が読めなければ何も除かない)。`ls-tree` は `--full-tree`(`:331`)なので、下のディレクトリの cwd でも root からのパスで照らす(probe で確かめた)。テスト `:536`、`:694` |
| AC12 古い watch-status と `recurred` の無い受信箱を読む | Met | S10 と S11 は watch-status の形を変えていない。テスト `watch_test.go:3745`、`:3801`、`watch_scope_test.go:1007`、`inbox_test.go:859` |
| AC13 help、agent-messaging(2 つの写し)、仕様の注記、RC3 の行 | Met | cycle 2 の V9 と V10 は `0cc32e6d` と `74e6b1db` で直り、仕様の FR-3 の注(82 行)と RC3 の注(375 行)は `--cc` と origin の既定のブランチの決め方を書く。S10 と S11 は、書かれている規則(予約の外のパスで立つ)を変えず、利用者の設定でそれが崩れる穴を閉じた。両方の注は `-c diff.relative=false` に触れないが、パスが cwd からの相対になるとも書いていないので、間違いではない(INFO I11)。help(`internal/cli/org_watch.go:83-92`)と agent-messaging の 2 つの写しは cycle 3 で変わっておらず、今も合う |

計画の決定との照合: S10 と S11 は D6 の「commit の変更ファイル」を、利用者の `diff.relative` と座席の cwd によらず repo の root からのパスで取るようにした。計画の Assumptions の「予約のパスも repo の root からの相対」と、porcelain の「作業ディレクトリによらず root からの相対」に、commit の一覧をそろえた形で、D6 の意図の中にある。新しい AC は要らない。

cross-review と self-review の直しの照合:

- cycle 2 の cross-review の 1 件(`diff.relative=true` と下のディレクトリの cwd で、commit のパスが cwd からの相対になり、両向きに誤る): S10 で直り、S11 で形を変えた。テスト `:748` が両向き(予約の中で立たない、外で立つ)と 2 つのコマンド(`log` と `diff`)を見る
- C3-1(S11、`e90c85da`): `--no-relative` は 2 つの引数列から消え、`-c diff.relative=false` になった。`git grep -n -- '--no-relative'` のコードの残りは、選んだ理由を書く `watch_scope.go:291` の comment だけ
- C3-2: 未処理(V15)。/sync-docs の担当

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、golang) | PASS(exit 0) | gofmt ok、`go vet ./...` は出力なし、golangci-lint `0 issues.`、staticcheck は出力なし(go 1.27.3。どちらにも「Skipping」の行は無い) |
| 同じ実行の中の local verifier | PASS | shellcheck と `sh -n`(hooks の 2 つの写し)、guard の awk、`settings.json` の jq、Codex の hook 検査 3 つ |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0 |
| `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh` | PASS | skill は 14 本とも同じ |
| tech-debt README の plan 参照の検査 | PASS | |
| branch の secret scan | PASS | `028a8f6d..521edd5b` を origin/main に対して clean |
| `./scripts/plan-visual.sh digest` | `e5a18ef9477a` | `- Approved:` の行と一致 |
| `git diff --check 0a957745 HEAD -- ':!docs/evidence'` | rc 0 | |
| `0a957745` のあとに変わった 14 ファイルの U+2028、U+2029、U+FFFD、U+200B、BOM | 0 件 | バイト列で数えた |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org watch --help` | Yes | cycle 3 で変わっていない。commit の一覧の形には触れない |
| `.claude/rules/ralph/agent-messaging.md` と templates/base の写し | Yes | cycle 3 で変わっていない |
| `/org` skill(4 つの写し) | Yes | `0cc32e6d` で escalation の打ち直しの 1 文が入った(cycle 2 の I6)。4 つの写しは同じ(`check-skill-sync.sh`、`check-sync.sh`)。commit の一覧のコマンドには触れない |
| README.md、AGENTS.md、`docs/quality/quality-gates.md`、`docs/recipes/worktrees.md`(と写し) | Yes | 要約で、S10 と S11 で間違いにはならない |
| 仕様の FR-3 の注(82 行) | Yes(INFO I11) | V9 は直った |
| `docs/tech-debt/README.md` | 一部 | V15、V16 |
| 計画の Progress checklist | 遅れ | V17 |
| 計画の D6 の本文、Scope、Implementation outline の 4 | 記録されたずれ | INFO I13 |
| `watch_scope.go` の `evaluateReservedScope` の doc | 要約 | INFO I12 |

### 所見(cycle 3)

ID は cycle 2 の V9〜V14、I6〜I10 の続きにした。どれも /sync-docs に回す。コードの直しは含まない。

- V15(LOW): tech-debt の既知の穴の行(462 行、「Eight gaps that fix/watch-alert-dedupe leaves」)
  - (a) C3-2(lead が挙げた既知のずれ)。(8) の Why deferred の「Like (6), they are tests after the pipeline's last run (cap 2 of 2)」と、Trigger の欄の cycle 2 の段落の「the tests and the doc line are code after the pipeline's cap of 2 runs, which the cross-review findings spent」は、cap が 3 に上がったこと(`7aee125e`、`cycle-count.json` の `"cap":3`)と合わない。cycle 3 の run は S10 と S11 に使われ、(6) と (8) のテストも (5) の doc も入っていない。`git grep -n 'cap 2 of\|cap of 2 runs' -- docs/tech-debt/README.md` のうち、この branch の行は 462 行だけ(191 行と 443 行は別の PR の行)
  - (b) 行全体の Trigger「The next change to the `scope_change` ... code of ... `internal/org/watch_scope.go`」は、S10 と S11 でもう一度当たった。どちらも `scopeCommittedPaths` の引数と doc を変えた。項目ごとの Trigger は当たっていない。(5) は `evaluateReservedScope` の doc の編集で、(6) は `scopeRecordsAtBase`、`dropOwnRecords`、`evaluateReservedScope` の HEAD の据え置き、`scopeDefaultBranchRef` の変更だが、`git diff -U0 0a957745 HEAD -- internal/org/watch_scope.go` の hunk は `scopeCommittedPaths`(271〜305 行)の中だけにある。hunk の見出しが `scopeDefaultBranchRef` を名指すのは、hunk がその関数の閉じ括弧の次の空行から始まるためで、その関数の行は変わっていない。(1)〜(8) は払われていない。cycle 2 の段落のあとに、cycle 3 で当たり、引き継いだことを 1 文で足す
  - (c) (3) の「`scopeCommittedPaths` gives none」は S10 と S11 のあとも正しい(git が前の HEAD を持たないときの道は変わっていない)
- V16(LOW): tech-debt の 357 行(`watch.go` と `watch_test.go` の大きさ)の引き継ぎの注は「At 3bb1828b, the last Go commit of the branch, ... `watch_scope.go` 461」と書く。今の最後の Go の commit は `e90c85da` で、`watch_scope.go` は 467 行(`ec538952` で 463 行)、`watch_scope_test.go` は 1354 行から 1423 行になった(`git show <commit>:<file> | wc -l`)。行の Trigger(「the watch's escalation or its tests」)には S10 と S11 は当たっていない。変えたのは予約の判定とそのテストで、escalation ではない。直すのは「the last Go commit」の 1 句と数だけでよい。`watch.go` 2081 行、`watch_test.go` 4188 行、`watch_recurrence.go` 214 行は今も同じ
- V17(LOW): 計画の Progress checklist。cycle 3 の self-review の 2 回目(`521edd5b`: Merge: yes、C3-1 は直った、C3-2 は持ち越し)と、この verify の行が無い。`521edd5b` で計画を grep すると 0 件だった。もう 1 点、cycle 2 の cross-review、S10、cycle 3 の self-review の 1 回目と S11 の 3 行(159〜161 行)が、「PR created」(158 行)と cycle 1 の cross-review の行(162〜171 行)のあいだに入っていて、並びが時の順になっていない。Progress checklist は digest の外なので(`scripts/plan-visual.sh` の `digest_body` は `## Progress checklist` の節を飛ばす)、並べ替えても承認は変わらない

### INFO(cycle 3)

- I11: 仕様の FR-3 の注(82 行)と tech-debt の RC3 の注(375 行)は、commit の一覧を `git log --cc --no-renames` と書き、`-c diff.relative=false` に触れない。どちらも要約で、`--no-show-signature`、`-z`、`--not <ref>` も書いていない。パスが cwd からの相対になるとは書いておらず、書かれた規則(予約の外のパスで立つ。予約は root からの相対)は、HEAD ではどの cwd でもどの `diff.relative` でも成り立つ(probe)。足すなら「利用者の `diff.relative` や座席の cwd によらず repo の root からのパスで取る」の 1 句でよく、足さなくても間違いではない。lead が挙げた既知の項目
- I12: `evaluateReservedScope` の doc(`watch_scope.go:380-393`)は、2 つのコマンドを `-c diff.relative=false` なしで要約し、`scopeCommittedPaths` を指す。指す先の doc(`:272-294`)は完全な argv と理由を書く。この doc を直すとコードの変更になり、tech-debt 462 行の (5) の Trigger(N5 の文を同じ編集で直す)にも当たる。今は直さなくてよい
- I13: 計画の D6(64 行)、Scope(21 行)、Implementation outline の 4(89 行)は `--no-renames` だけを書く(cycle 2 の I8 の続き)。Progress の 160 行は S10 が `--no-relative` を足したと書き、161 行が S11 でそれを `-c diff.relative=false` に替えたと書く。記録されたずれとして扱う。計画の本文は digest に入るので、変えない
- I14: S11 の doc の「`--no-relative` は git 2.28 から」と「それより古い git には `diff.relative` の設定が無い」は、この verify では git の release notes と照らしていない(手元に写しが無い)。self-review の probe(git 2.26.3 で `log --no-relative` は終了コード 128、`-c diff.relative=false` は 0 で root からのパスを返す)と、この repo の `docs/plans/archive/2026-09-25-secret-scan-git-config.md:49` の記述とは合う。2.26.3 と 2.49.0 のあいだの版は、self-review もこの verify も試していない

## Observational checks

- S10 と S11 の diff を全部読み、triage の 1 件(`docs/reports/cross-review-triage-watch-alert-dedupe.md`)と self-review の C3-1 について、直しの道とテストの名前と assert を突き合わせた(上の表と照合)
- 実際の git の probe(git 2.49.0、`GIT_CONFIG_GLOBAL=/dev/null`、作業用の clone に `diff.relative=true` と `status.relativePaths=true`、cwd は `internal/org`)。本番の argv(`git --no-optional-locks -c diff.relative=false log --no-show-signature --cc --no-renames --name-only -z --format= <prev>..<head> --not refs/remotes/origin/main --`)は `cmd/y.go` と `internal/org/x.go` を返し、`-c` を外した対照は `x.go` だけを返した。予備の `diff` も同じ結果だった。同じ cwd で `status --porcelain=v1 -z --untracked-files=all` は ` M cmd/y.go` と `?? docs/plans/active/new.md` を root からのパスで返し、`ls-tree --full-tree --name-only -z <base> -- docs/plans/active/old.md docs/plans/active/new.md` は分岐点にあった `old.md` だけを返した。scope の判定に入るパスを出す 4 つの git が、どれも cwd と利用者の設定によらず root からのパスを返すことになる(triage の Implementation context の主張と合う)。出力は evidence の log にある
- 座席の `Worktree` は spawn の `Cwd` から来る(`spawn.go:1277`、`:1304`)。`evaluateReservedScope` はそれを git の cwd に渡す(`watch.go:1276`)。下のディレクトリを cwd にした座席が本番でもありうることと、テストの fixture(`newReservedScopeFixtureAt`)が同じ形を作ることを確かめた
- tech-debt の Trigger の欄を python で切り出し(`re.split(r"(?<!\\) \| ", row)` の 4 つ目のセル)、S10 と S11 が触った名前(`watch_scope`、`scopeCommittedPaths`、`newReservedScopeFixture`、`scopeStubGit`、`RunsOnlyTheGitCallsItNeeds`、`scopeSubcommand`、`diff.relative`、`scope_change`)で照合した。当たったのは 462 行(V15)だけ。210 行は Trigger(「`evaluateCycle` の呼び出しの打ち切り方の変更」)に当たらず、S10 と S11 は git の呼び出しの数を変えていないので、注の「at most ten」と「about 100 s」は今も正しい。357 行は Trigger に当たらず、注の数だけが古い(V16)。180、353、423 行は名前を一般の語で拾っただけ
- cycle 2 の所見の処理を確かめた(下の「Cycle 2 の記録」)
- origin/main は `028a8f6d` から動いていない(`git ls-remote origin refs/heads/main`)

## Coverage gaps

- Go のテストは走らせていない(/test)。表のテストの欄は、テストがあり assert が AC を言っていることまでである。とくに S10 の新しいテスト(`watch_scope_test.go:748`)と、`scopeSubcommand` を通すようにした 4 つの fake のテストは、読んだだけ。Progress の 161 行の「変異で 2 つのコマンドがそれぞれのテストで落ちる」も、この verify では再現していない
- probe は git 2.49.0 だけで走らせた。2.28 の境界と、それより古い git での `-c diff.relative=false` の振る舞いは、self-review の 2.26.3 の probe に頼っている(I14)
- 実機の走行は無い。機能の org を新しい binary で走らせ、受信箱の件を数え直すことは RC3 の注のとおり残っている
- cycle 1 からの「決定的な検査が無いもの」(skill の `--json` の欄の一覧が `org.InboxItem` の JSON の欄に追いつかないこと)は今も無い。最小の検査は、`InboxItem` の `json:"..."` の名前のうち基本の欄を除いたものが `.claude/skills/org/SKILL.md` の `inbox` の行にすべて出ることを見る Go のテスト 1 本である

## Verdict

pass

- Verified: AC1〜AC15 は、コードの道とそれを assert するテストがあることを確かめた。S10 と S11 は cycle 2 の cross-review の 1 件と self-review の C3-1 を直し、利用者の `diff.relative=true` と下のディレクトリの cwd でも commit のパスが root からの相対になることを、テストの形と git 2.49.0 の probe で確かめた。AC13 は、cycle 2 の V9 と V10 が直ったので Met にした。静的解析はすべて通った(gofmt、go vet、golangci-lint、staticcheck、shellcheck、同期の検査 4 つ、template purity、tech-debt の plan 参照、secret scan)。計画の digest は承認の行と同じ。base との衝突はない
- Not verified: Go のテストの結果と `run-test.sh`(/test)、S11 の変異、2.28 の境界の git、実機での件の数え直し
- 所見(cycle 3): LOW 3 件(V15〜V17)、INFO 4 件(I11〜I14)。どれも文書と記録のずれで、merge を止めない。lead が挙げた既知のずれ(C3-2 と、仕様と tech-debt の commit の一覧の説明)は V15 (a) と I11 で確かめた。V15 (b)、V16、V17 はこの verify で新しく見つけた
- /sync-docs に回すもの: V15(462 行の 2 か所の cap の文と、cycle 3 の引き継ぎの 1 文)、V16(357 行の「the last Go commit」と数)、V17(計画の Progress に `521edd5b` とこの verify、並びの順)。I11 は任意

## Cycle 2 の記録

cycle 2(HEAD `3bb1828b`、commit `0a957745`)の判定は pass で、所見は LOW 6 件(V9〜V14)と INFO 5 件(I6〜I10)だった。AC13 は「一部」(V9、V10)。V9〜V14 は cycle 2 の /sync-docs(`0cc32e6d`、`74e6b1db`、`cd4a9268`)がすべて処理した(`docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md`)。この cycle で、処理された文が今も HEAD にあることを確かめた。

| ID | 中身(cycle 2) | 処理 |
| --- | --- | --- |
| V9 | 仕様の FR-3 の注(82 行)に S8 の `--cc`(merge commit 自身の編集を数える)が無い。origin の既定のブランチの決め方は任意 | `0cc32e6d`。両方が入った |
| V10 | tech-debt の RC3 の注(375 行)に同じく `--cc` が無い | `74e6b1db`。両方が入った |
| V11 | 462 行の既知の穴の行。(a) C2-1 の残る窓、(b) M48 の Trigger が S8 で当たった、(c) (5) の Trigger が S8 で当たった、(d) 行全体の Trigger の引き継ぎ、(e) 見出しの数 | `74e6b1db`。(a) は (7)、C24 と C04 は (8)、(b)〜(d) は Trigger の欄の cycle 2 の段落、見出しは「Eight gaps」。cap の文は cycle 3 で古くなった(V15) |
| V12 | 210 行の git の呼び出しの数(S8 で最大 10 回、およそ 100 秒)と、`foldIntoPendingAlert` の呼び出し | `74e6b1db`。S10 と S11 のあとも正しい |
| V13 | 357 行の数と cycle 2 での当たり、457 行の (f) の判断 | `74e6b1db`。357 行の注の「the last Go commit」は cycle 3 で古くなった(V16)。457 行の (f) は注を足した |
| V14 | 計画の Progress に `d3a34921`、`3bb1828b`、cycle 2 の verify が無い | `cd4a9268`(Progress の 167〜171 行) |

INFO(cycle 2): I6 は `/org` skill の 4 つの写しが S8 のあとも正しく、escalation の打ち直しの 1 文は任意(`0cc32e6d` で足された)。I7 は `docs/recipes/worktrees.md` の要約。I8 は計画の D6 の本文などが `--no-renames` だけを書くこと(I13 に続く)。I9 は self-review の報告(`d3a34921`)の C2b-1 が `3bb1828b` で直ったこと。I10 は tech-debt 303 行の `newTestWatchRun` の呼び出しの数が正しいこと(S10 と S11 は `watch_test.go` を触っていないので今も同じ)。

## Cycle 1 の記録

cycle 1(HEAD `f45133f0`、commit `b8058948`)の判定は pass で、所見は LOW 8 件(V1〜V8)と INFO 5 件(I1〜I5)だった。V1〜V8 は cycle 1 の /sync-docs(`45e84b3e`、`984761f8`、`0c727338`、`2bebc1ac`)がすべて処理した(`docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md`)。

| ID | 中身(cycle 1) | 処理 |
| --- | --- | --- |
| V1 | `/org` skill の 4 つの写しの `watch` の行、`inbox` の行の `--json` の欄、「見張りの ALERT の件」の節に今回の挙動が無い。計画の Rollout の「skill は変えない」 | `45e84b3e`。Rollout は計画の Progress の注(`2bebc1ac`) |
| V2 | README の 131 行と 278 行 | `45e84b3e` |
| V3 | `docs/quality/quality-gates.md` の 2 つの写しの Watchdog pulse layer の行 | `45e84b3e` |
| V4 | `docs/recipes/worktrees.md` の 2 つの写しの「recorded worktree と比べる」 | `45e84b3e` |
| V5 | AGENTS.md の `internal/org/` の行 | `45e84b3e` |
| V6 | tech-debt の register。(a) 462 行の Trigger に `watch_recurrence.go` が無い。(b) N5 が入っていない。(c) Trigger に当たって注の無い 357・303・211 (a)・353 行。(d) コメントだけで当たった 355・258 行。(e) 210 行の Item に、予約のある org の座席が 1 周期に git を最大 9 回打つことを足す | `984761f8`。(e) の数は S8 のあと 10 回になった(V12) |
| V7 | self-review の報告の U+2028 | `0c727338` |
| V8 | 計画の AC の箱と Progress | `2bebc1ac` |

INFO: I1 は help の `seatBusy` の要約、I2 は仕様の FR-3 の注が L2 のあとの規則の要約であること、I3 は `/director` skill の 967 行が件の回数を書かないこと(計画の Non-goals で #307b、今も残る)、I4 は `internal/org/reserve.go:26` と `/org` skill の 460〜462 行が今の挙動と合うこと、I5 は計画の Affected areas の `watch_all_test.go` と `reserve.go` が変わっていないこと。
