# org-ssh-alias-host

- Status: Approved
- Approved: 2026-11-19 sha256:f4a9803ca80e
- Owner: Claude Code
- Date: 2026-11-19
- Related request: 下流の repo(ralph 8.2.1)で、origin が ssh の別名(`~/.ssh/config` の `Host github.com.emu`、`Hostname github.com`)だと `ralph doctor` の Merge guarantee が unknown になり、`ralph org merge` も同じ host で gh を打つので merge できない。gh 自体は別名の origin でも動く。issue 系列 #340〜#344 のレーン B の 1 本目
- Related issue: #340
- Type: fix
- Branch: fix/org-ssh-alias-host

## Objective

origin が ssh の URL(`ssh://` か scp 形)のとき、gh と同じく `ssh -G <host>` が返す `hostname` を gh の `--hostname` と `-R` の host に使う。doctor の Merge guarantee と、`ralph org merge`・`ralph org cleanup`・`ralph org regate`(`regate_checks = "ci"`)が origin を読む関数を 1 つにそろえ、同じ host で gh を打つ。解けないときは今の host のまま進み、doctor と merge の案内に repo ごとに鍵を選ぶ手(`git config core.sshCommand`)を足す。

## Scope

- `internal/org/mergeorigin.go`: origin の URL と host から gh の host を決める関数を 1 つ足す(仮名 `resolveMergeAPIHost`)
  - ssh の URL(`ssh://` と scp 形)のときだけ、`MergeOrderExec` で `ssh -G <host>` を打ち、出力の `hostname ` で始まる行の最後の値を使う(go-gh の `pkg/ssh` の `Translator` と同じ読み方)。https・http・`git://` では打たない
  - 解いた値を小文字にしてから、今の `ssh.github.com` → `github.com` の読み替えを重ねる(別名が `ssh.github.com` に解ける場合がある。go-gh も解いた後に読み替える。go-gh は大文字小文字を無視して比べるが、`mergeGuaranteeAPIHost` は完全一致で比べるので先に小文字にする)
  - `ssh -G` が PATH にない、失敗した、`hostname` の行を出さない、専用の期限(5 秒、`mergeSSHResolveTimeout`)を過ぎた、解いた値が空・`-` で始まる・空白や制御文字や `/` を含むときは、origin の host のまま進み、解けなかった理由を返す
  - `-` で始まる host には `ssh` を打たない(今の `ReadMergeOrigin` は既にそのような origin を断る。doctor の読みでも打たない)
  - 本物の `ssh -G` は `realMergeDetachedExec`(自分のプロセスグループで走らせ、ctx が終わればグループごと SIGKILL する)で打つ。OpenSSH は `Match exec` の命令を子プロセスで走らせるので、`realMergeOrderExec` で ssh だけを止めると子が残る。呼び手は git と gh を打つ実行関数とは別に、ssh を打つ実行関数を渡す。nil なら `realMergeDetachedExec`。テストは両方に fake を渡す
- `ReadMergeOrigin`(`mergeorigin.go`): 引数に ssh を打つ実行関数を足す(呼び手は merge の `m.runDetached`、regate(ci)は同じ形で足す欄、cleanup は nil)。get-url のあとに上の関数で host を決める。`MergeOrigin` に、解けなかったときの理由と origin に書かれた host を持たせ、gh が失敗したときの案内(後述)に使う。親の ctx が `ssh -G` の間に終われば `MergeOriginInterrupted` を返す
- doctor の `mergeGuaranteeRun.readOrigin`(`mergeguarantee.go`): 同じ関数で `apiHost` を決める。`ssh -G` は検査全体の 20 秒の期限と専用の 5 秒の短いほうで打つ。ssh を打つ実行関数は `MergeGuaranteeParams.Exec` が nil なら `realMergeDetachedExec`、fake なら同じ fake
- キャッシュ `merge-guarantee.json`(`MergeGuaranteeCheck`)と `ralph status`・`ralph org merge-order` の `MergeGuarantee`: `host` は origin に書かれた host のまま保ち、gh に渡した host を新しい欄 `api_host`(`omitempty`)に持つ。`ReadMergeGuarantee` の照合は今のまま `host` で行う。版(`version: 1`)は上げない(欄を足すだけで、古いキャッシュも読める)
- doctor の Detail: 解いた host が origin の host と違うとき、両方を 1 か所で言う(例: `origin's host github.com.emu is an ssh alias of github.com (ssh -G), which gh read`)。gh が失敗し、別名を解けなかったとき(理由がある)は、今の「point origin at the real host」に代えて、理由と `core.sshCommand` の手を言う。解けて gh が失敗したときは、今の gh の到達とログインの案内だけを言う
- `ralph org merge` の案内(`merge.go` の `ghAccess` と `ghMissing`): 別名を解けなかったとき(理由がある)だけ、doctor と同じ `core.sshCommand` の手を足す。`mergeReady.origin()` は `apiHost` と `repo` から `MergeOrigin` を作り直すので、理由は `mergeReady` に欄を 1 つ足して `readOrigin`(`merge.go:416`)で入れる
- `CheckMergeEnforcement`(`mergeenforce.go`): `Host` は `ReadMergeOrigin` が決めた gh の host を受け取る(今も merge.go から `r.apiHost` が渡っている)。`ssh -G` は打たない。コメントを直す
- 文書: README の doctor の行(136 行)、org skill の 4 写し(1986 行付近の「origin が ssh の別名なら」)、`docs/specs/2026-10-07-org-multi-org-director.md` の 7c-2 の決定(129 行)に #340 で改めたことを足す、`mergeguarantee.go` の C7 のコメント、`mergeorigin.go`・`merge.go`・`mergeenforce.go` のコメント、`docs/tech-debt/README.md` の 250 行の項を RESOLVED にし、director skill の確かめ(Non-goals)の行を足す
- slice 1 の実装中に見つけた 2 点(計画からのずれ): `RegateParams.Exec`(`regate.go:243`)のコメントに ssh -G も通ることを足す。merge・regate(ci)・cleanup の取り消しの文(`while git remote get-url origin ran`)を、ssh -G の最中の取り消しにも合う文に直す
- テスト(fake の `MergeOrderExec`): 解く関数の表のテスト、doctor の別名の origin、キャッシュの `host` と `api_host`、`ReadMergeGuarantee` が別名の origin で `origin_changed` にならないこと、`ReadMergeOrigin` の別名、merge と doctor が同じ host を使うこと、PR の url の照合(D10)が解いた host で通ること、解けないときの案内

## Non-goals

- merge-order の PR の照合(`mergeOriginRepo.check`)。host を比べず owner/repo だけで照合するので、別名の影響を受けない
- `git+ssh://` など、今 `mergeOriginFromURL` が読まない scheme を読めるようにすること
- `ralph status` が `ssh` や gh を打つこと。status はキャッシュだけを読む(今のまま)
- origin の host の書き方を `ssh.github.com` と `github.com` の間で変えたときの `origin_changed`(tech-debt の別の項。今のまま)
- merge queue(#342、次の PR)
- 実機での `ralph org merge` の実走(org を立てる必要がある。fake のテストで確かめる)
- `/director` skill の gh の issue のコマンドの前の確かめ(origin の URL を `github.com` の形の grep で確かめ、当たらなければ打たない。`.claude/skills/director/SKILL.md` 767〜772 行)。別名の origin ではこの確かめに当たらず、director は issue を起票しない。skill の shell の手順の変更になるので、この計画では直さず、tech-debt に行を足す(実装中に見つけた。計画からのずれ)

## Assumptions

- gh(go-gh v2 の `pkg/ssh`、2026-11-19 に GitHub の main で読んだ)は、scheme が ssh の remote(scp 形は `git.ParseURL` が `ssh://` に直す)の host を `ssh -G <host>` で解き、出力の `hostname` の行の最後の値を使う。失敗したとき(`ssh` がない、終了コードが 0 でない、`hostname` の行がない)は元の host を使い、解いたあとに `ssh.github.com` を `github.com` に読み替える。ralph はこれに合わせる
- OpenSSH の `ssh -G <host>` は設定を読んで出すだけで、接続しない。ただし `Match exec` の命令は走る(gh も同じ)。`~/.ssh/config` は passwd の home から読まれ、`$HOME` は使われないはず(未確認)
- `ralph status` の照合はキャッシュの `host` と origin の host を比べる(`mergeguarantee.go:791〜796`)。`host` を origin に書かれたままにすれば、別名の origin でも `origin_changed` にならない
- このマシンの `~/.ssh/config` に `Host github.com.emu`(`hostname github.com`)がある(`ssh -G github.com.emu` で確かめた)。実機の確かめに使える

## Affected areas

- `internal/org/mergeorigin.go`(解く関数、`ReadMergeOrigin`、`MergeOrigin`)
- `internal/org/mergeguarantee.go`(`readOrigin`、`readRepository` の案内、`MergeGuaranteeCheck` と `MergeGuarantee` の `api_host`、Detail、C7 のコメント)
- `internal/org/merge.go`(`ghAccess`・`ghMissing` の案内、`mergeReady` のコメント)
- `internal/org/mergeenforce.go`(`Host` のコメント)
- テスト: `internal/org/mergeorigin_test.go`(新規、または `merge_test.go` の `ReadMergeOrigin` の表)、`internal/org/mergeguarantee_test.go`、`internal/org/merge_test.go`、必要なら fake が `ssh` を受けるように直す既存のテスト
- 文書: `README.md`、`.claude/skills/org/SKILL.md` ほか 3 写し、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/tech-debt/README.md`
- 共有ファイル(系列の連携): README.md、org skill の 4 写し、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-ssh-alias-host.html`。かんたんな図 2 枚(`overview`、`easy-record`)と、詳しい図 3 枚(`detail-overview`、`detail-sequence`、`detail-record`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 5 枚の図を `shot` で撮って読んだ。直したのは 4 点。図 1 の人の絵の位置(下にずれていた、「これから」の文字と重なっていた)、図 2 の「選ばない」が線に重なっていたこと、ファイルの地図の Host の線が regate_ci.go と org_cleanup.go の箱を横切っていたこと、順序の図の 2 つの文字が線に重なっていたこと。撮り直して、どれもきれいに出ることを確かめた
- 計画助言の指摘(プロセスグループ)を入れたあと、ファイルの地図(regate_ci.go と org_cleanup.go を「変える」に)、順序の図(失敗の箱に 1 行)、「承認の前に」を直し、lint(exit 0)と撮り直しをした。順序の図の失敗の箱で文字が箱からはみ出したので 3 行に分けた

## Design decisions

Critical forks: 6 件。どれも consult(consult-plan-sshalias、判定「進めてよい」)に聞き、自律で選んだ。

- D1 `ssh -G` を打つ範囲: ssh の URL なら host を問わず毎回打つ(gh と同じ)。github.com と ssh.github.com を省く案は外部コマンドを 1 回減らせるが、`Host github.com` に `Hostname` を書いた人の解き方が gh とずれる。AC3 は「別名でない origin で gh に渡す host が変わらない」と読み、別名でない ssh の origin でも `ssh -G` が 1 回増えることは許す(OpenSSH は同じ host を返す)
- D2 `ssh -G` の期限: 専用の 5 秒(`mergeSSHResolveTimeout`)。doctor では検査全体の 20 秒との短いほう、`ReadMergeOrigin` では get-url と別に区切る。我々の gh は `--hostname` と `-R` を渡すので gh 自身は `ssh -G` を打たず、`Match exec` の命令が返らないときに止まるのは ralph の `ssh -G` だけになる。呼び手の期限を共有すると doctor の 20 秒を食い尽くす。期限で止めるときは、計画助言(codex)の指摘を受けて、ssh のプロセスグループごと止める(`realMergeDetachedExec`)。ssh だけを止めると `Match exec` の子が残り、doctor や merge を打つたびに溜まる
- D3 `core.sshCommand` の案内を足す範囲: doctor と `ralph org merge`(`ghAccess`・`ghMissing`)だけ。cleanup と regate(ci)は host が解ければ直り、解けないのは `ssh` がない・設定が壊れているといったまれな場合なので、案内は今のまま。issue が求めたのは doctor の案内
- D4 キャッシュ: `host` は origin に書かれた host のまま、`api_host` を `omitempty` で足し、版は 1 のまま。ユーザーの指示どおり別の欄に持つ。古い binary は知らない欄を読み飛ばすので版を上げる理由がない
- D5 置き場所: 解く関数は `mergeorigin.go` に置く。origin を読む関数がすでにある所で、AGENTS.md の repo map を変えずに済む
- D6 解いた値の検査: 空、`-` で始まる、空白・制御文字・`/` を含む値は使わず、origin の host のまま進む。gh はこの検査をしないが、ralph は `-R <host>/<o>/<r>` で gh の引数に入れるので、gh がフラグと読む値を防ぐ。self-review の L1・L5 を受けて(計画からのずれ)、ssh に渡す origin の host と ssh が返す hostname の両方を 1 つの関数 `mergeSSHHostName` で確かめる形にした。通すのは、空でなく、`-` で始まらず、どのバイトも `A–Z a–z 0–9 . _ : -` の名前だけ。origin の host が `$(id)` や `a;touch` のような名前なら ssh を打たない(ssh の設定の `Match exec "... %h"` が host を shell に渡しうるため)。ASCII でない hostname と `%`(IPv6 の zone)を返す設定は、origin の host のまま進む。どちらも今の版と同じ扱いになるだけで、悪くはならない
- `CheckMergeEnforcement` は origin を読まない(`merge.go:343` で `r.apiHost` を受け取る)ので `ssh -G` を打たず、受け取った Host をそのまま使う。memory の「3 か所」は `mergeGuaranteeAPIHost` の呼び所の数で、解く場所は `ReadMergeOrigin` と doctor の `readOrigin` の 2 つになる

## Acceptance criteria

- [x] AC1: origin が ssh の別名(`ssh -G` が `hostname github.com` を返す)のとき、doctor の Merge guarantee は gh を `--hostname github.com` で打ち、保護を読んで種類を決める(unknown にならない)。fake のテストで確かめる
- [x] AC2: `ralph org merge` が使う host(`ReadMergeOrigin` の `-R` と `CheckMergeEnforcement` の `--hostname`)と doctor が使う host が、同じ origin と同じ `ssh -G` の答えで一致する。テストで確かめる
- [x] AC3: 別名でない origin(https、`git://`、`ssh -G` が同じ host を返す ssh)の gh の host は今と変わらない。https と `git://` では `ssh` を打たない。テストで確かめる
- [x] AC4: キャッシュの `host` は origin に書かれた host のまま、`api_host` に gh に渡した host が入り、`ralph status`(`ReadMergeGuarantee`)は別名の origin で `origin_changed` にならない。テストで確かめる
- [x] AC5: `ssh -G` が失敗・期限切れ・`hostname` なし・不正な値のときは origin の host のまま進み、gh が失敗したら doctor と merge の案内が理由と `git config core.sshCommand` の手を言う。テストで確かめる
- [x] AC5b: 本物の ssh を打つ経路は `realMergeDetachedExec` を使う。PATH に置いた偽の `ssh`(子プロセスを立てて止まる)で、専用の期限と親の ctx の取り消しのどちらでも呼び出しが戻り、子プロセスも終わることを実プロセスのテストで確かめる。doctor・merge・regate(ci)・cleanup の本番の経路が ssh に `realMergeDetachedExec` を渡すこともテストで確かめる
- [x] AC6: 別名の origin で、gh が返した PR の url(`https://github.com/<o>/<r>/pull/<n>`)が origin の PR と認められる(D10 の照合が解いた host で通る)。テストで確かめる
- [x] AC7: 実機: このマシンで一時 clone の origin を `ssh://git@github.com.emu/yoshpy-dev/ralph.git` にし、新しい binary の `ralph doctor` の Merge guarantee が `--hostname github.com` で読み、ssh の別名を理由に unknown にならない(結果を evidence に残す)
- [x] AC8: 文書(README の doctor の行、org skill の 4 写し、spec の 7c-2、tech-debt の 250 行)が新しい挙動と合う。`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る

## Implementation outline

1. 解く関数と `ReadMergeOrigin`: `mergeorigin.go` に `resolveMergeAPIHost` と専用の期限を足し、`ReadMergeOrigin`(ssh を打つ実行関数の引数を足す)と `MergeOrigin` を直す。呼び手の merge(`m.runDetached`)、regate(ci)(`regateCI` に同じ形の欄を足す)、cleanup(nil)を直す。merge.go の `ghAccess`・`ghMissing` の案内に解けなかった理由を足す。表のテスト(ssh の別名、`ssh://`、https と `git://` で打たない、失敗・期限切れ・`hostname` なし・不正な値・複数行の最後、`ssh.github.com` への解決、`-` の host)と、`ReadMergeOrigin` の別名、PR の url の照合、merge の案内のテスト。AC5b の実プロセスのテスト(PATH の偽の `ssh` が子を立てて止まり、専用の期限と親の ctx の取り消しで戻り、子も終わる)と、本番の経路が ssh に `realMergeDetachedExec` を渡すテスト(`merge_run_test.go` の reflect で関数を比べる形)。既存の fake が `ssh` を受けないために落ちるテストを直す(呼び出しの番号で動く `after` を持つ fake は、ssh を別に記録して番号をずらさない) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/`
2. doctor とキャッシュ: 最初に `internal/cli/main_test.go` の TestMain と doctor・merge・cleanup の CLI のテストが PATH を fake の bin に限る範囲を確かめ、本物の `ssh` が走る経路がないことを見る。`readOrigin` が同じ関数を使い、`MergeGuaranteeCheck` と `MergeGuarantee` に `api_host` を足し、Detail と `readRepository` の案内を直す。C7 と `mergeenforce.go` のコメントを直す。doctor の別名の origin(AC1)、キャッシュの `host`・`api_host` と `ReadMergeGuarantee`(AC4)、doctor と merge が同じ host(AC2)、案内(AC5)のテスト。CLI のテスト(`internal/cli`)の fake が `ssh` を受けるか確かめる — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/`
3. 文書: README の doctor の行、org skill の 2 つの本体(`.claude/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`)を直して `scripts/sync-skills.sh` で `.agents` の 2 写しを作る、spec の 7c-2 の決定、tech-debt の 250 行を RESOLVED — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh`
4. 実機の確かめ(AC7): binary を作り、一時 clone で doctor を打ち、結果を `docs/evidence/ssh-alias-host-livefire-2026-11-19.md` に残す(token や鍵のパスは写さない) — verify: `test -s docs/evidence/ssh-alias-host-livefire-2026-11-19.md`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck)
- Spec compliance criteria to confirm: AC1〜AC8。gh の解き方(go-gh の `Translator`)と同じ順序(解く → `ssh.github.com` の読み替え)であること
- Documentation drift to check: README の doctor の行、org skill の 4 写し、spec の 7c-2、tech-debt、`mergeguarantee.go` の C7 と `CheckMergeGuarantee` の doc comment、`mergeorigin.go` の `MergeOrigin` の doc comment、`MergeEnforcementParams.Host`
- Evidence to capture: verify の report、実機の evidence

## Test plan

- Unit tests: `resolveMergeAPIHost` の表(上の slice 1)、`ReadMergeOrigin` の別名と失敗、`MergeOrigin.hasPR` の解いた host での照合、`CheckMergeGuarantee` の別名の origin、`api_host`、Detail、`ReadMergeGuarantee` の照合
- Integration tests: doctor の CLI のテスト(`internal/cli/doctor_merge_guarantee_test.go`)が fake のまま通ること。CLI のテストは PATH を fake の bin だけにしているので、本物の `ssh` は走らない(走るテストがあれば fake を入れる)
- Regression tests: 既存の origin の表(https、`ssh.github.com`、GHES、大文字小文字、`-` の host)の期待値が変わらないこと
- Edge cases: `ssh -G` が複数の `hostname` 行を出す、`hostname` の値が空、`-` で始まる、空白を含む、期限切れ、親の ctx が `ssh -G` の間に終わる、別名が `ssh.github.com` に解ける、IPv6 の host
- Evidence to capture: test の report、`go test` の出力

## Risks and mitigations

- `ssh -G` は `~/.ssh/config` の `Match exec` の命令を走らせる。gh も remote を解くときに同じことをするので、ralph が新しく増やす危険は小さい。それでも doctor・merge・cleanup・regate のたびに走るので、Risk として README と doc comment に書く
- `ssh -G` が止まる(`Match exec` の命令が返らない)と、doctor の 20 秒の期限を食う。専用の 5 秒で区切り、過ぎたら origin の host のまま進む。止めるときはプロセスグループごと SIGKILL する。命令が `setsid` などで自分のグループを作った場合は残る(gh も同じで、ralph からは止められない)
- `ssh -G` の出力をそのまま gh の引数に渡す。`-` で始まる値、空白・制御文字・`/` を含む値は使わない(`-R <host>/<o>/<r>` で gh がフラグと読むのを防ぐ)。self-review のあとは、入力の host と出力の hostname の両方を `A–Z a–z 0–9 . _ : -` だけの名前に限った(D6)
- 別名でない ssh の origin でも `ssh -G` が 1 回増える。OpenSSH は同じ host を返すので gh の host は変わらない(AC3)。手元で数十ミリ秒(未計測)
- テストが本物の `ssh` を打つと、走らせる人の `~/.ssh/config` で結果が変わる。org のテストは fake の `MergeOrderExec`、CLI のテストは PATH を fake の bin だけにしているので打たない。slice 2 で確かめる
- 既存の fake の多くは知らないコマンドに error を返す。解けないときは origin の host のまま進むので結果は変わらないが、呼び出しの数や順序を確かめるテストは直す
- 解けなかったときの doctor の案内が「point origin at the real host」から変わるので、この文を確かめる既存のテスト(`mergeguarantee_test.go` の「gh fails on an ssh alias host」)を新しい文に直す。slice 2 の工数に入れる
- `ssh -G` の stdin: `runMergeOrderCommand` は `cmd.Stdin` を設定しないので、Go の `os/exec` は null device をつなぐ。`Match exec` の命令が入力を待って止まることはない

## Rollout or rollback notes

- 次の release に入る。release ノートの候補: 「origin が ssh の別名(`~/.ssh/config` の Host)でも、doctor の Merge guarantee と `ralph org merge`・`cleanup`・`regate`(ci)が `ssh -G` で実ホストを解いて gh を打つ。キャッシュに `api_host` が増える」
- 戻すときは PR を revert する。キャッシュの `api_host` は古い binary が読み飛ばすので、戻しても壊れない

## Open questions

- なし(分かれ目は Design decisions に書く)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 1 件: [MEDIUM] `ssh -G` を期限で止めても、`realMergeOrderExec` は ssh だけを止めるので `Match exec` の子プロセスが残る。→ 取り入れた(ssh は `realMergeDetachedExec` で打ち、`ReadMergeOrigin` に ssh の実行関数を足す。AC5b に実プロセスのテストと本番の経路のテストを足した。D2、Risks、slice 1、図を直した)
- [x] Plan approved — ユーザーの指示(「計画は推す案で自動承認する」、2026-11-19)により、推す案で自動承認した
- [x] Branch created — `fix/org-ssh-alias-host`(worktree `.claude/worktrees/org-ssh-alias-host`)
- [x] Implementation started
- [x] Slice 1(554d04aa): 解く関数、`ReadMergeOrigin`、merge・regate(ci)・cleanup、merge の案内、AC5b の実プロセスのテスト。implementer の報告を確かめた(HEAD 一致、作業木は空、自己点検 5 項目 clean、検証コマンド EXIT=0、internal/org 795 秒)。検証コマンドは計画の slice 1 の行に `go vet ./internal/cli/` と `go build ./...` を足して渡した(`org_cleanup.go` が internal/cli にあるため)。ずれ: `TestCheckMerge_PassesWithEverythingTheMergeNeeds` の期待値に `originHost` を足した(欄が増えたため、条件は緩めていない)、検査に不正な UTF-8 を足した、表テストの書き換え 1 回を python で行った(ASCII のみ)
- [x] Slice 2(40afa91e): doctor の `readOrigin` が同じ関数を使い、キャッシュと `MergeGuarantee` に `api_host`、別名の注記(`aliasNote`)、`mergeSSHAliasHint` の案内、`newMergeGuaranteeRun`(本番の ssh は `realMergeDetachedExec`)、`RegateParams.Exec` と `MergeEnforcementParams.Host` のコメント、取り消しの文(`while origin was read (git remote get-url origin, and ssh -G for an ssh origin)`)。報告を確かめた(HEAD 一致、作業木は空、自己点検 5 項目は clean か fixed、検証コマンド 2 回とも EXIT=0、internal/org 931〜1040 秒、internal/cli 740 秒)。ずれ: `APIHost` は ssh -G の最中に止まったときは空(gh の host が決まっていないため)。意図した挙動の変化: 別名でない ssh の GHES や https の github.com 以外の origin で gh が失敗したとき、「doctor cannot resolve it」の文は出ない(AC5 の設計どおり)。CLI のテストで本物の ssh が走る経路はない(doctor は TestMain の Exec、cleanup・merge・regate は PATH を stub の bin に絞る)。implementer が main のチェックアウトの `.claude/agent-memory/implementer/` に memory を書いたが、gitignore 済みで main は汚れていない
- [x] Slice 3(44a56862): README の doctor の行、org skill の 4 写し(`sync-skills.sh` で `.agents` の 2 写しを作った。template 側は `CLAUDE_ROOT`・`CODEX_ROOT` を渡して作った)、spec の 7c-2、tech-debt の 250 行を RESOLVED、director skill の行を追加。`check-skill-sync.sh` と `check-sync.sh` が通った。報告を確かめた(HEAD 一致、作業木は空、自己点検 5 項目は clean か fixed)
- [x] Slice 3b(4ec556e4、inline の 1 ファイルの直し): implementer の申し送りから、README の `ralph org merge` の段落に ssh -G で host を解くことを足した。`check-sync.sh` が通った。自己点検 5 項目(不要な変更、誤字・貼り間違い、デバッグの残り、秘密情報、範囲外のファイル)はどれも clean
- [x] Slice 4(d19a2fa6、inline の新しい 1 ファイル): 実機の確かめを `docs/evidence/ssh-alias-host-livefire-2026-11-19.md` に書いた。別名の origin で 8.2.1 は unknown、新しい binary は `github_protection`(`host` は別名、`api_host` は github.com)、`ralph status` は `origin_changed` にならない、別名でない 2 つの origin は 8.2.1 と同じ行、ssh を PATH から外すと `core.sshCommand` の案内。`test -s` が通った。自己点検 5 項目は clean(token などの値は写していない)
- [x] Self-review(15066e7c): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 7、Merge: yes。L1・L2・L3・L5 を直し、L4・L6・L7(引数の数、テストの重複と書き方)は残した
- [x] Slice 5(55cdb7c3、self-review の直し): `mergeSSHHostName` と `mergeSSHResolveRuns`、merge と cleanup の `--help`。報告を確かめた(作業木は空、自己点検 5 項目 clean)。検証コマンドは 1 回目が負荷による既知の flake 1 件(`TestRegateCI_AnInterruptOnTheMergePath`、load average 46、単独では通る、fixture の origin は今回の判定の外)で EXIT=1、2 回目が EXIT=0。ずれ: `-` で始まる host の理由の文を `%q` にした。D6 と Risks を実装に合わせて直し、digest を取り直して自動承認した
- [x] Self-review の 2 回目(a77538ab): LOW 6(残した L4・L6・L7 と新しい L8〜L10)、Merge: yes。L1・L2・L3・L5 は解決と判定
- [x] Slice 6(3c369ff4、2 回目の self-review の直し): L10(ssh の出力を弾いたときの理由の文を `mergeSSHHostNameWhy` に一本化し、「gh が受け取れない」と言わないようにした)、L8(regate の `--help` の ci の段落と doctor の subprocess のコメントに ssh -G を足した)。L9(理由の文の `%q` と案内の `%s` の混在)は、表示の前に `PrintableInboxText` が制御文字を escape するので残した。検証コマンド(関係するテストに絞った `-run`)は EXIT=0。自己点検 5 項目 clean。申し送り: README 136 行の「no hostname that `gh` can take」は /sync-docs で直す
- [x] main(e6d5d0ed、#346・#347)を取り込んだ(0a71a930)。README の衝突は、main が足した `ralph pack remove` の行と、この branch の doctor の行を残して解いた(main は doctor の行を変えていなかった)
- [x] Verify の 1 回目(bc221a8d): fail。tech-debt に足した director の行が計画を `docs/plans/archive/` で引いていた(計画はまだ active)。9783b9a7 で active のパスに直した(/pr の `archive-plan.sh` が archive に移すときに書き換える)
- [x] da80502f(verify の D3): このブランチが `merge_test.go`(940 行)と `regate_ci_test.go`(833 行)に足したテストを、`merge_ssh_alias_test.go` と `regate_ci_ssh_alias_test.go` に移した(移動だけ。テスト関数の集合は同じ)
- [x] 14597e36(テストの意図を条件つきで緩めた。`testing.md` に従ってここに書く): `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` の期限の場合を 1 秒から 3 秒にし、偽の ssh が子の pid を書かないまま期限が来たときは `t.Skipf` で飛ばす。load average がおよそ 100 のときに、偽の ssh が子を立てる前に期限が来て 1 回落ちたため。呼び出しが戻ることと理由の文は skip の前に確かめるので、緩むのは「期限切れのときも子が終わる」の確かめだけで、それも skip した回にだけ緩む。ctx の取り消しの場合は、pid が書かれてから取り消すので skip しない
- [x] Verify の 2 回目(b01d04c1): pass。static は rc=0。/sync-docs に渡すもの: README の doctor の行の「no hostname that `gh` can take」(D1)、Trigger に当たる tech-debt の行への carried-over の注(D2)、evidence のテストの置き場所(D5)
- [x] Cross-review の 1 回目(79eb08dd、cycle 1/2): codex の指摘 1 件を ACTION_REQUIRED に分けた。doctor だけが `checkMergeGuarantee(context.Background(), ...)` で呼ぶので、`ssh -G` が `Match exec` で待つ間に Ctrl-C で doctor を止めると、自分のプロセスグループで動く ssh とその子が残る(#340 より前は doctor と同じグループで動いていた)。ユーザーの指示どおり聞かずに直し、cycle を 2 にした
- [x] 240324fe(cross-review の直し、計画からのずれ: `internal/cli/doctor.go` と `doctor_merge_guarantee.go` の ctx の扱いを足した): Merge guarantee の検査を Node の検査と同じ形の `doctorMergeGuaranteeSignalContext`(`signal.NotifyContext`、SIGINT・SIGTERM・SIGHUP)で走らせる。割り込まれたら後の検査を打たず、そこまでの行と `Interrupted: the Merge guarantee check did not finish. Run ralph doctor again.` を出してエラーで終わる。キャッシュの規則は変えていない。新しいテスト `doctor_merge_guarantee_signal_test.go`(本物の SIGINT の場合を含む)。検証コマンド EXIT=0、自己点検 5 項目 clean。申し送り: doctor の検査が signal で止まるときの動きは README に書かれていない(Node の検査も)。/sync-docs で決める
- [x] 申し送りのうち直さなかったもの: `.claude/skills/pr/SKILL.md` 46 行の「SSH host aliases that `gh` cannot resolve」(pr skill の owner の sed の説明で、gh 一般の話。#340 の範囲外として PR の follow-up に書く)
- [x] 計画の更新(slice 1 のあと): Non-goals に director skill の確かめを足し、Scope に tech-debt の行と slice 1 で見つけた 2 点を足した。digest を取り直して自動承認した
- [x] Test(63e1bc72): pass。`./scripts/run-test.sh` が rc 0(1,270 s、shell の suite 45 本と Go の 11 パッケージ)。名前を挙げられたテスト 21 件は `-v` の絞った run で PASS し、本物の ssh の期限切れの場合(14597e36 で skip を足した)は 11 回とも走って skip 0。変異 23 個はどれも落ちた。HEAD の binary で AC7 の場面を打ち直し、evidence と同じ内容の行を見た。Test gaps は LOW 2 件(`resolveMergeAPIHost` の `sshRun == nil` の既定を通すテストがない、cleanup が `ReadMergeOrigin` に nil を渡すことを固定する CLI のテストがない)と、main(#348・#349)を取り込んだ tree では走らせていないこと
- [x] Sync-docs: D1(README の doctor の行の括弧を `mergeSSHHostName` の規則に合わせた)、D2(Trigger に当たった tech-debt の行 250・251・258・310・394・409・419・501 に carry-over の注を足し(250 は verify の一覧になく、sync-docs の走査で見つけた)、`merge_test.go` の大きさの行を足した)、D5(evidence のテストのファイル名と、binary を打ち直した verify・test の report への指し先)。D4 は 6a17dfad で直済み。`docs/reports/sync-docs-2026-11-19-org-ssh-alias-host.md`
- [x] Self-review の 3 回目(e07bdb0b、cycle 2): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6(残した L4・L6・L7・L9 と、`240324fe` の新しい L11・L12。L8・L10 は解決)、Merge: yes
- [x] Verify(fd76b7c0、cycle 2): pass。AC はすべて Met。HEAD の binary に SIGINT と SIGTERM を送り、`ssh -G` とその子が消えて doctor が exit 1 で終わることを見た。`/sync-docs` に D6(README の doctor の行に signal の動きがない)、D7(tech-debt の `doctor.go` の行の行数と commit)、D8(register に足す行)を渡した
- [x] Test(530bf97c、cycle 2): pass。`./scripts/run-test.sh` が rc 0(1,464 s、shell の suite 45 本と Go の 11 パッケージ)。`240324fe` の signal のテストは 23 回ずつ PASS、変異 11 個はどれも落ちた。M12(検査の signal の登録を release する時点を doctor の終わりまで遅らせる)は生き残った(Test gaps 2、LOW)。HEAD の binary に SIGINT・SIGTERM・SIGHUP を送り、どれでも ssh とその子が消えて exit 1 になった
- [x] main(32d6dc51、#348・#349・#350)を取り込んだ(e6a98fad)。衝突は `docs/tech-debt/README.md` の「Five files went past the 800-line guide」の行の 1 か所で、#341 の注とこの branch の注を両方残して解いた
- [x] Sync-docs の 2 回目(cycle 2): D6(README の doctor の行に、Merge guarantee と Node toolchain の検査が signal で止まるときの動きを 1 文足した)、D7(tech-debt の `doctor.go` の行のこの branch の注を、1651 → 1683 行と、コメントの `3c369ff4` とコードの `240324fe` の 2 commit に直した)、D8(残した LOW の L4・L6・L7・L9・L11・L12 と test の M12 を 1 行にして register に足した)。`docs/reports/sync-docs-2026-11-19-org-ssh-alias-host.md`
- [x] Review artifact created — `docs/reports/self-review-2026-11-19-org-ssh-alias-host.md`
- [x] Verification artifact created — `docs/reports/verify-2026-11-19-org-ssh-alias-host.md`
- [x] Test artifact created — `docs/reports/test-2026-11-19-org-ssh-alias-host.md`
- [ ] PR created
