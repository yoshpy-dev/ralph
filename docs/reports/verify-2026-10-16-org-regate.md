# Verify report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 3c417c48763c で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 のうち、merge のあとの取り直し。8 段目の 8a-3b)
- Verifier: verifier subagent (Claude)。パイプライン 3 回目の 2 度目の verify。1 度目(40629f28)の V3-1 を受けて D4 と AC7 に件名の条件を足し、承認し直したあとに回した(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 2 のまま、上限は 3)
- Scope: `git diff 670ec219...HEAD`(HEAD fadf6b4c)。1 度目の verify からの差分は、ca7a02eb(D4 と AC7 の書き直し、承認し直し)、0844ad2b(V3-1 の直し)、62f8e31b(`internal/org/seat.go` の注だけ)、fadf6b4c(計画の Progress)。書き直した D4 と AC7 を重く見て、コードを読み、3 つの binary で打った。HEAD の binary、直す前(40629f28)の binary、HEAD から `--no-show-signature` だけを外した変異の binary。ほかの AC は、差分がそのコードを変えていないことを確かめ、前回の結果を引き継ぐ。テストは走らせていない(/test の受け持ち)。行番号は HEAD fadf6b4c のもの
- Evidence: `docs/evidence/verify-2026-10-16-org-regate.log` の「cycle 3, second pass」の節(前回までの節のあとに足した。run-static-verify の出力、Go の検査、git の RelNotes の該当行、probe 8 のスクリプトと 3 つの binary の出力、正規化した差分。`$PROBE8` は scratchpad の probe のディレクトリ)。wrapper の全ログは `docs/evidence/verify-2026-10-16-231648.log`。どちらも gitignore の対象

## 前回からの変化

- 0844ad2b: `regateIsOwnMerge`(`internal/org/regate_merge.go:148-158`)は、記録の base の代わりに記録そのもの(`gateRecord`)を受け取る。親がちょうど 2 つで `[originHead, 記録の base]` のときだけ、`git log -1 --no-show-signature --format=%s <HEAD>` で件名を読み(`:153`)、末尾の改行を落として `regateMergeSubject(記録の base_ref, 記録の base, ブランチ)` と比べる(`:157`)。親が違えば件名を読まずに false を返す(`:150`)。`regateMergeSubject`(`:160-166`)は `Merge <base_ref から refs/remotes/ を除いた名前> <base の先頭 12 文字> into <ブランチ>` を返し、base が 12 文字より短ければ全体を使う。`regateMergeTip` も件名をこの関数で作る(`:315`)。呼び出し元の `regateHeadNotOnOrigin`(`:80`)と `regateResetHint`(`:120`)は渡すものが記録になっただけで、分岐は変わっていない
- 0844ad2b の新しいテスト(読んだだけ): `TestRegate_APersonsMergeOfTheDefaultBranchIsNotTakenForRegates`(`internal/org/regate_merge_test.go:1297`。`git merge origin/main` と GitHub の「Update branch」の件名の 2 つと、通った記録と落ちた記録の 2 つで 4 case)と `TestRegate_AMergeCommitWhoseSubjectAHookChangedIsNotPushedOnly`(`:1364`。prepare-commit-msg の hook が件名の頭に `[checked] ` を足す)
- 62f8e31b: `internal/org/seat.go:72-84` の注に、gate の event を `ralph org regate` も書くことと、`via=regate` が `log=` の前に来ることを足した。下の「Documentation drift」で確かめた
- 計画: ca7a02eb で D4(`:70`)と AC7(`:86`)に件名の条件を足し、承認し直した(sha256:3c417c48763c)。fadf6b4c で Progress(`:167-168`)に V3-1 の受け方と直しを書いた

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 断る 12 の場合 | 満たす(前回の確認のまま) | 差分は D3 の確かめにも断りの文にも触らない |
| AC2 fetch の失敗、origin にブランチがない、HEAD が `origin/<b>` と違う | 満たす(cycle 1 の binary の確認のまま) | 差分が触らない。probe 8 の B2、C1、E2、G1 も「HEAD … has commits that origin's tip … does not」で断った |
| AC3 何もしない道と、壊れた記録のあとの取り直し | 満たす(binary でも確認) | probe 8 の E3 で、断りに従って人が push したあと打ち直すと `UP_TO_DATE:` で終了コード 0、manifest は 3 行のまま |
| AC4 取り込みの道 | 満たす(binary でも確認) | probe 8 の A1、D1、E1、F1 で merge した。件名は `Merge origin/main <base の 12 桁> into feat/auth` で、前と同じ(F1 は `--no-show-signature` の出力、E1 は hook が頭に `[checked] ` を足したもの)。D1 は `GATE: pass`・`MERGED:`・`HEAD:`・`BASE:`・`LOG:`・`PUSHED:` を出して終了コード 0 |
| AC5 merge commit を作れない | 満たす(前回の確認のまま) | 差分が触らない |
| AC6 スクリプトの失敗、記録の前の中断、`head_moved` と `tree_changed` | 満たす(cycle 1 の確認のまま) | 差分が触らない |
| AC7 push の失敗と記録のあとの中断、打ち直し(書き直したもの) | 満たす(binary でも確認) | コード: `regateIsOwnMerge` は D4 の 4 つの条件(親が 2 つ、第 1 親が `origin/<b>` の先頭、第 2 親が記録の base、件名が regate の書くもの)をそのまま見る。push だけの道(`:79-83`)と手(`:115-127`)の両方がこれを使う。probe 8(HEAD の binary): A1 は push が断られて終了コード 1、A2 の打ち直しは `PUSH_ONLY:`。A3 は main が進んだあと `reset --keep` の手付きで断った。B は前回と同じく、push せず手も出さなかった。C1 は人の merge commit を gate だけの regate が通したあと origin を第 1 親に戻すと、断り、push しなかった(origin は第 1 親のまま)。C2-1 は main も進めると、手を出さずに断り、何も変えなかった。G1 は衝突の断りのコマンドどおりに人が merge した commit で C1 と同じ結果。D は push の直後に fetch された tracking ref を残した。E2 は hook が件名を変えた regate の merge commit の打ち直しを、手を出さずに断った(D4 の「断る側」)。F2 は署名と `log.showSignature=true` のもとで `PUSH_ONLY:`。直す前の binary は C1・G1・E2 で `PUSH_ONLY:` で push し、C2-1 で手を出した。変異の binary は F2 で断った。テスト(読んだだけ): 上の 2 つと `TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt`(`:1253`) |
| AC8 スクリプトの間の org か結びつきの変化 | 満たす(テストのコードで確認) | 差分が触らない |
| AC9 gate だけの道 | 満たす(binary でも確認) | probe 8 の B1、C0、G0 は、merge も push もせずに gate を走らせ、`via=regate` の `gate_passed` を書いた |
| AC10 merge-order の理由の欄 | 満たす(前回の確認のまま) | 差分は `internal/org/mergeorder.go` に触らない |
| AC13 Spawn との排他(D9) | 満たす(テストのコードで確認) | 差分が触らない |
| AC11 文書 | 一部(LOW のずれ 5 件) | 下の V3-2〜V3-6 |
| AC12 テストの隔離、go1.26 と go1.25.8 | 一部(静的な部分を確認) | 足した 2 つのテストは `newRegateFixture` の bare origin、`setHooks`、`rejectPushes`、origin の中の `commit-tree` だけを使い、URL、`osascript`、本物の `herdr` を打たない。`go build` と `go vet` は go1.26.0 と `GOTOOLCHAIN=go1.25.8` の両方で通った(`min` の組み込み関数は Go 1.21 から)。`--no-show-signature` を押さえるテストはない(V3-7)。テストの実行は /test |

D1〜D9:

- D4: 書き直した文とコードは一致する。D4 の件名「`Merge <記録の base_ref の origin/ 以下> <記録の base の 12 桁> into <b>`」は、base_ref から `refs/remotes/` を除いた `origin/main` を使うコードと同じで、D3 の件名(「base_ref の短い名前」)とも同じ。親が合うときだけ件名を読み、違う件名なら push だけの道にも手にも進まない。記録の base_ref と base は、regate が merge に使った `t.base.ref` と `t.base.sha` から書かれる(`internal/org/regate.go:333`、`:666`)。そのため、hook が件名を変えない限り、regate 自身の merge commit の件名は記録から作った件名と一致する
- D4 の `--no-show-signature`: git 2.10.0 で `log.showSignature` と一緒に入った(Homebrew の git 2.49.0 が持つ RelNotes の `2.10.0.adoc:70-71`)。`gateGit` はどの git にも `--no-optional-locks`(2.15.0、`2.15.0.adoc:97-100`)を渡すので、regate が動く git はもともと 2.15 以上で、この flag は下限を上げない。署名のない commit では、`log.showSignature=true` でも flag の有無にかかわらず件名だけが出る(Apple の git 2.50.1 で確認)。署名した commit では、flag なしの `git log -1 --format=%s` は件名の前に `Good "git" signature for …` の行を stdout に出し、flag ありでは件名だけになる(probe 8 の F、git 2.49.0)。flag を外した変異の binary は F2 で regate 自身の merge commit を形に当たらないと読んで断ったので、flag は効いている
- D1〜D3、D5〜D9: 差分が触らない。前回までの結果のまま

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | scope は changed、言語は golang。hook の shellcheck と `sh -n`、`check-sync.sh`(IDENTICAL 170、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan の参照、gofmt ok、golangci-lint `0 issues.`、`secret-scan-branch: scanned 670ec219..fadf6b4c against origin/main: clean` |
| `go vet ./...`(go1.26.0) | pass | 出力なし、rc 0 |
| `gofmt -l internal/ cmd/` | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod の Go |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/org/...` | pass | 出力なし、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし、rc 0 |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass(rc 0) | IDENTICAL 170、DRIFTED 0、KNOWN_DIFF 5 |
| `git diff --check 670ec219...HEAD`、`git diff --check 40629f28 HEAD` | pass | 出力なし |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | 3c417c48763c |

## Findings

| ID | Severity | 種類 | Finding | Evidence | 勧め |
| --- | --- | --- | --- | --- | --- |
| V3-1 | LOW(0844ad2b で直った。数えない) | 振る舞い(D4 の形の判定) | 1 度目の verify の指摘: 衝突の断りに従って人が作る merge commit は、親だけなら regate 自身の merge commit と同じ形なので、gate だけの regate のあとに origin を第 1 親に戻すと、push し直すか `reset --keep` を勧めた。0844ad2b の `regateIsOwnMerge` は件名も比べ、人の merge commit の件名(git の既定の `Merge remote-tracking branch '…' into feat/auth`)は regate の件名と違うので、形に当たらない | probe 8 の C と G。HEAD の binary: C1 は終了コード 1 で断り、push せず(origin は第 1 親のまま)、C2-1 は手を出さずに断り、HEAD・origin の 2 つの ref・作業ツリー・manifest の行数を変えなかった(`unchanged: yes`)。G1(`git -C <worktree> merge refs/remotes/origin/main` で作った件名 `Merge remote-tracking branch 'refs/remotes/origin/main' into feat/auth`)も C1 と同じ。直す前の binary(40629f28)は C1 と G1 で `PUSH_ONLY:` を出して origin を人の merge commit に戻し、C2-1 で `reset --keep` を勧め、従うと C2-2 が `conf.txt` の衝突で断った。コードは `internal/org/regate_merge.go:148-158` | 済み。tech-debt に V3-1 は書かない(V3-3 の勧めを参照) |
| V3-2 | LOW | 文書(skill) | `/org` skill の 4 面の `regate` の行は、`reset --keep` の手の条件を「HEAD の第 1 親が origin の先頭」と書く。556f73fb と 0844ad2b のあとの条件は、親がちょうど 2 つで、第 1 親が origin の先頭、第 2 親がその記録の base、件名が regate の書く `Merge origin/<既定のブランチ> <base の 12 桁> into <ブランチ>`(`regateIsOwnMerge`)。同じ行の push の打ち直しの文(「打ち直すと、push が届いていなければ HEAD の push だけをし」)は、`via=regate` の記録ではこの形のときだけ push することも、hook が件名を変えた merge commit では打ち直しが断り、自分で push するよう言うことも書かない | `.claude/skills/org/SKILL.md:188`(`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` も同じ。4 面は cmp で同一)。`internal/org/regate_merge.go:79-83`、`:115-127`、`:148-158`。probe 8 の E2 と E3 | 4 面の手の条件を上の 4 つに直し、push だけの道にも同じ条件を 1 文足す。hook が件名を変えたときは打ち直しが断り、断りに従って push すれば次の regate は `UP_TO_DATE:` になることも書く(/sync-docs) |
| V3-3 | LOW | 文書(tech-debt) | 0e08999f のあと、4 つの行が D3 の関数を古いファイルに置く。`regateMergeWrites`(281 行の本文と最後の欄、282 行の最後の欄)、`regatePathInTheWay` と `regateSpellingOnDisk`(285 行の本文と最後の欄、286 行の本文と最後の欄)はどれも `internal/org/regate_merge.go` と書くが、今は `internal/org/regate_inway.go` にある(282 行と 287 行の `regateMergeTip` は `regate_merge.go` のまま)。286 行は、regate の間に断られるものを「the org's `spawn` and `start --plan` are refused (D9)」と書き、C3-1 で直した help(spawn は `--dry-run` でないもの、start はどちらの形も)と合わない。1 度目の verify から変わっていない | `docs/tech-debt/README.md:281`、`:282`、`:285`、`:286` | 4 行のファイル名を `internal/org/regate_inway.go` に、286 行の文を「the org's `spawn` (not a `--dry-run`) and `start` (either form) are refused (D9)」に直す。self-review の報告が足すよう勧める 1 行は、(a)〜(d) が 82cf0d39・0e08999f・14252f06 で直ったので (e) を書く。(e) の「HEAD's parents are then origin's tip and the record's base」には、0844ad2b のあとの条件に合わせて「and its subject is the one regate writes」を足す。V3-1 は直ったので書かない。V3-5〜V3-7 を直さないなら、この行に足す(/sync-docs) |
| V3-4 | LOW | 文書(repo-map) | `AGENTS.md` の repo-map は regate のファイルを `regate.go`、`regate_merge.go`、`regate_git.go`、gate のファイルを `gate.go`、`gate_run.go` と名指すが、0e08999f と 14252f06 が足した `regate_inway.go`(merge の前の D3 の確かめ)と `gate_record.go`(gate と regate が書き、merge-order が読む gate の記録)を名指さない。1 度目の verify から変わっていない | `AGENTS.md:90` | 2 つのファイルを repo-map の regate と gate の句に足す(/sync-docs) |
| V3-5 | LOW | 文書(Go の注) | `leaderActivityEventCount` の注の (c) は「Only `ralph org gate` writes them, and the leader template has a feature org's leader run it, so a gate's record counts as the leader acting」と書く。この PR から `ralph org regate` も `gate_passed` と `gate_failed` を書く。62f8e31b は同じ食い違いを `seat.go` の注で直したが、この注には触れていない。数え方はそれで誤らない(コードを読んだ範囲): regate は org が走っていれば断り(D2)、記録の直前にも manifest のロックの下で org が走っていないことを確かめる(D6)ので、走っている org の窓に regate の記録は入らない | `internal/org/watch.go:1208-1214`、`:1243-1244`。`internal/org/seat.go:72-84`。`internal/org/regate.go:241`、`:689` | (c) に「`ralph org regate` writes them too, only while the org does not run (D2, D6), so a regate record never counts for a running org」の 1 文を足す。Go の注なので、直すなら code の commit になる。直さないなら V3-3 の tech-debt の行に書く |
| V3-6 | LOW | 文書(help) | `ralph org regate --help` の打ち直しの段落は「Run regate again then: when the push did not land, it pushes HEAD and does nothing else」と条件なしに言う。0844ad2b のあと、prepare-commit-msg のような hook が regate の merge commit の件名を変えると、push が落ちたあとの打ち直しは push だけの道に進まず、「push them (git -C <worktree> push origin <b>) and run … again」で断る。D4 が「断る側に倒れる」と書くとおりの振る舞いで、断りに従えば次の regate は `UP_TO_DATE:` で終わる。help はこの場合を書かない | `internal/cli/org_regate.go:99-105`。probe 8 の E(HEAD の binary): E1 は件名 `[checked] Merge origin/main <12 桁> into feat/auth` の merge commit の push が断られ、E2 は終了コード 1 で断り、`reset --keep` の手を出さず、何も変えなかった(`unchanged: yes`)。E3 は人の push のあと `UP_TO_DATE:`。直す前の binary は E2 で `PUSH_ONLY:` を出した | help の段落に「unless a hook changed the merge commit's subject: regate then refuses and asks you to push HEAD yourself」の 1 文を足す。Go の文字列なので、直すなら code の commit になり、help を押さえる `internal/cli/org_regate_test.go` も見る。直さないなら V3-3 の tech-debt の行に書く |
| V3-7 | LOW | テストの穴 | `--no-show-signature` を押さえるテストがない。fixture は `isolateGitEnv`(`internal/org/statedir_test.go:139`)で `GIT_CONFIG_GLOBAL=/dev/null` にし、どのテストも commit に署名せず、`log.showSignature` も立てない(`internal/` を grep すると、`commit.gpgsign=false` を渡す箇所だけがある)。読んだ範囲では、flag を外してもどのテストも落ちない(テストは走らせていない)。flag は効いている: 外した変異の binary は、署名した regate 自身の merge commit の打ち直しを断った | probe 8 の F。HEAD の binary の F2 は `PUSH_ONLY:` で終了コード 0、変異の binary の F2 は終了コード 1 で断った。`internal/org/regate_merge.go:153` | `ssh-keygen` で鍵を作り、fixture の repo に `gpg.format=ssh`・`user.signingkey`・`commit.gpgsign=true`・`log.showSignature=true` を置いて、push が落ちたあとの打ち直しが `PUSH_ONLY` になることを確かめるテストを足す(`ssh-keygen` がなければ skip)。足さないなら V3-3 の tech-debt の行に書く |

情報(ずれに数えない):

- I1 件名が regate のものとちょうど同じ人の merge commit(`-m` で同じ文を書いたもの)は、今も regate 自身の merge commit と読まれる。件名で見分ける以上避けられず、D4 はそれを前提にしている。probe では打っていない
- I2 regate の help の「When origin's default branch or origin's branch moved in between, regate refuses and says what changed」(`internal/cli/org_regate.go:104-105`)は、regate 自身の merge commit が届いたあとに origin を第 1 親に戻した場合(self-review の C3-5 の (2))には当たらない。その場合は件名も合うので、0844ad2b のあとも push し直す(読んで導いた。binary では打っていない)。C3-5 の登録簿の行で扱う
- I3 GitHub の「Update branch」の件名 `Merge branch 'main' into <b>` は、`regateIsOwnMerge` の doc(`internal/org/regate_merge.go:140-141`)とテストが書く値で、本物の GitHub では確かめていない(ネットワークを打たない)。件名が regate のものと違えば断る側に倒れるので、GitHub の実際の件名が別の文でも結果は同じ。regate の衝突の断りが名指す `git merge refs/remotes/origin/main`(`:367`)の既定の件名は `Merge remote-tracking branch 'refs/remotes/origin/main' into feat/auth`(probe 8 の G)。`/org` skill の衝突の手順の `git merge origin/main`(`.claude/skills/org/SKILL.md:1036`)の既定の件名は、テストの期待値では `Merge remote-tracking branch 'origin/main' into feat/auth`(読んだだけ)
- I4 `internal/org/regate_merge_test.go` の冒頭の注は「the merge commit, its undo, and the push」と書くが、D3 の確かめのテストもこのファイルにある。移動の前からの書き方で、テストのファイルを分けるときに直せばよい(tech-debt の 279 行)
- I5 計画の Progress(`:167-168`)は V3-1 の受け方と直しを書く。「直す前に 5 件が赤だった」は、直す前のコードを読むと、人の merge の 4 case(通った記録では push か手、落ちた記録では手)と hook の case(push だけ)が期待と違うので、数は合う(テストは走らせていない)。AC1〜AC13 は `[x]`。digest は 3c417c48763c

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の 4 面 | 一部(V3-2) | 4 面は同一。`regate` の行の手と push だけの道の条件が 556f73fb と 0844ad2b の前のまま |
| `/director` skill の 4 面 | yes | 手と push だけの道の条件を書かないので、0844ad2b とぶつからない |
| CLI の help(`ralph org regate`) | 一部(V3-6、情報 I2) | 打ち直しの段落が、hook が件名を変えた場合を書かない |
| `AGENTS.md` の repo-map | 一部(V3-4) | `regate_inway.go` と `gate_record.go` がない |
| `README.md` | yes | 手と打ち直しの条件を書かない |
| 仕様(75 行の 8a-3b の注) | yes | 「打ち直すと push だけをする」は設計の要約で、形と件名の条件は計画の D4 が持つ |
| `docs/tech-debt/README.md` | 一部(V3-3) | 4 行のファイル名と 286 行の文。C3-5 の行はまだない(/pr の前に足す予定) |
| `docs/quality/quality-gates.md` の 2 面 | yes | 差分が触らない |
| `internal/org/seat.go:72-84` の注(62f8e31b) | yes | 書き手は `Gate`(`internal/org/gate.go:135`)と `Regate`(`internal/org/regate.go:276`)で、その 2 つだけが gate の記録を作る。Details の並び(`gate_failed` では `reason=` が先頭、regate の記録では ` via=regate` が `log=` の前)は `gateRecord.details`(`internal/org/gate_record.go:70-89`)と一致する。「A refusal before the scripts run records neither」は regate でも成り立つ(断り、`PUSH_ONLY`、`UP_TO_DATE` は記録しない) |
| `internal/org/watch.go:1208-1214` の注 | 一部(V3-5) | 「Only `ralph org gate` writes them」が残る |
| Go の注(0844ad2b) | yes | `regateHeadNotOnOrigin`、`regateResetHint`、`regateIsOwnMerge`、`regateMergeSubject`、`regateMergeTip` の doc はコードと合う |
| 計画の AC と Progress | yes | 情報 I5 |

## Observational checks

probe 8 は、HEAD fadf6b4c の binary(`go build ./cmd/ralph` を scratchpad に出したもの)、直す前の 40629f28 と、HEAD から `--no-show-signature` だけを外した変異を、それぞれ `git archive` で scratchpad に出して作った binary で打った。シナリオ A〜D は probe 7 のまま、E〜G を足した。`env -i` に `PATH=/usr/bin:/bin` を渡して走らせ、場合ごとに Homebrew の git 2.49.0 の symlink と `/usr/bin`・`/bin` だけの PATH にした。gh は PATH にない。HOME と `GIT_CONFIG_GLOBAL` は probe の中の偽の場所、origin は probe の中の bare repository、台帳は scratch の main checkout の `.harness/state/org`(結びつきと `disbanded` の 2 行)。gate のスクリプトは `echo` と exit 0 だけの stub。F の鍵は偽の HOME の中で `ssh-keygen` が作り、`SSH_AUTH_SOCK` は外した(agent を使わない)。場合ごとに作り直した。ネットワーク、herdr、osascript、端末の app、tmux は打っていない。signal は送っていない。

- A(regate 自身の merge commit): 3 つの binary で同じ。A1 は push が断られて終了コード 1、A2 は `PUSH_ONLY:`、A3 は main が進んだあと `reset --keep` の手付きで断った
- B(gate だけ、1 つ親の commit): 3 つの binary で同じ。push せず、手も出さなかった
- C(gate だけ、衝突を解いた人の merge commit): HEAD の binary は C1 と C2-1 で断り、push も手もなかった。直す前の binary は C1 で push し直し、C2-1 で手を出し、従うと C2-2 で衝突した。V3-1 が直った
- D(push の直後の fetch): 3 つの binary で同じ。tracking ref を fetch の値のまま残した
- E(hook が件名を変えた regate の merge commit): HEAD の binary は E2 で断り(手なし)、E3 で人の push のあと `UP_TO_DATE:`。直す前の binary は E2 で `PUSH_ONLY:`
- F(ssh の署名と `log.showSignature=true`): HEAD の binary は F2 で `PUSH_ONLY:`。変異の binary は F2 で断った。flag なしの `git log -1 --format=%s` は件名の前に署名の確かめの行を出した(鍵の指紋は evidence で伏せた)
- G(衝突の断りのコマンドどおりの人の merge commit): HEAD の binary は G1 で断り、push しなかった。直す前の binary は `PUSH_ONLY:` で push し直した

## Coverage gaps

- テストは走らせていない。go1.26 と `GOTOOLCHAIN=go1.25.8` の `go test ./internal/org/ ./internal/cli/` は /test で確かめる
- `--no-show-signature` を押さえるテストはない(V3-7)
- commit-msg の hook が件名を変えた場合と、`i18n.logOutputEncoding` が UTF-8 でなくブランチの名前に ASCII 以外の文字がある場合は打っていない。どちらも件名が合わず、断る側に倒れると読んだ(未確認)
- 本物の GitHub の「Update branch」の件名(情報 I3)
- leader の記録(`via` なし)で origin を戻したあとの push し直し(C3-5 の (1))と、regate 自身の merge が届いたあと第 1 親に戻された場合(C3-5 の (2))は、binary で打っていない
- 前回までの Coverage gaps はそのまま残る: 中断の道、`head_moved` と `tree_changed`、hook が git を殺す場合、AC13 の spawn と `start --plan` の binary の確認、端末の Ctrl-C の実機、ssh の origin、本物の GitHub への push、Linux、本物の 2.38 より前の git、case を区別するファイルシステムでの D3

## Verdict

- Verdict: pass(MEDIUM 以上はない。V3-1 は 0844ad2b で直った。残る LOW は 6 件。V3-2〜V3-4 は 1 度目からの文書のずれで、/sync-docs が直す。V3-5 と V3-6 は Go の注と help の文のずれ、V3-7 はテストの穴で、直さないなら tech-debt の行に書く)
- Verified: 承認の digest。書き直した D4 と AC7 のコード(`regateIsOwnMerge`、`regateMergeSubject`、`regateMergeTip` の件名、`regateHeadNotOnOrigin` と `regateResetHint` の呼び出し)と計画の文の突き合わせ。HEAD の binary で、V3-1 の場合(C1、C2-1)と衝突の断りのコマンドどおりの人の merge commit(G1)で push も手も出ないこと、regate 自身の merge commit の打ち直しと手(A、A3)、署名と `log.showSignature=true` のもとでの打ち直し(F2)、hook が件名を変えたときに断る側に倒れること(E2)、B と D が前回と変わらないこと。直す前の binary との対照 4 件(C1、C2-1、E2、G1)と、変異との対照 1 件(F2)。`--no-show-signature` が入った版と、regate の git の下限(RelNotes)。`seat.go` の注。静的解析(run-static-verify、go vet、gofmt、go1.25.8 の build と vet、check-skill-sync、check-sync、`git diff --check`)
- Partially verified: AC6、AC8、AC13 のうちテストだけが固定する場合、AC11(LOW のずれ 5 件)、AC12 のテストの隔離(コードを読んだ範囲)
- Not verified: テストの実行、`--no-show-signature` を押さえるテスト、本物の GitHub の「Update branch」の件名、C3-5 の 2 つの場合の binary の確認、端末の Ctrl-C の実機、ssh と GitHub の origin、Linux
- Insight event: `insights-append.sh` で `--verdict pass --cycle 3 --medium 0 --low 6` を記録した
