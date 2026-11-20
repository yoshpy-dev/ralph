# `ralph org reseat --restart-herdr`: 座席を別のアカウントへ移すときの herdr のサーバーの立て直し

- Date: 2026-11-20
- Issue: #354
- Status: Approved(2026-11-20、系列の指示で推す案を自律に承認した。consult の判定「直してから進める」の 3 点を入れたあと)
- 実装の担当: レーン A
- 実装: 2026-11-20、計画 `docs/plans/archive/2026-11-20-org-reseat-restart-herdr.md`(PR #359)。FR-1〜FR-8、FR-5b、NFR-1〜NFR-3、AC1〜AC8 の印は、verify の Met と test の pass(fake の herdr、lsof、ps によるテスト)に合わせて付けた。座席のある org での `ralph org reseat --restart-herdr` の通し、2 つのアカウントでの資格の切り替え(Open questions の最後)、Linux の `/proc/<pid>/environ` の経路は、実機では確かめていない。FR-2 の断りの順は、本当の reseat では 1 回目の pane の確かめが `PlanReseat` の 10 番(ゲート)のあとに来る(`PlanReseat` の doc)

## Summary

座席を別の Claude のアカウント(別の `CLAUDE_CONFIG_DIR`)へ移すには、herdr のサーバーを新しいアカウントの環境で起動し直してから `ralph org reseat` を打つ。今はサーバーを止める手順を人に頼むほかない。`ralph org reseat` に `--restart-herdr` を足し、herdr のサーバーにその org の座席のほかに pane が 1 つもないと確かめられたときだけ、座席を止めたあとにサーバーを止め、呼んだ側の環境で起動し直してから座席を立てる。`--claude-config-dir <dir>` を付ければ、起動し直すサーバーの `CLAUDE_CONFIG_DIR` をその値にし、`--claude-config-dir ''` なら外す(既定のアカウントへ戻す)。reseat は、動いているサーバーの `CLAUDE_CONFIG_DIR` を読めたら出し、`--restart-herdr` なしで呼んだ側の値と違えば知らせる。

## Background and problem

### Current state

- ralph が止まったサーバーを起動するとき(`ensureOrgHerdrServer`、`internal/cli/org_herdr_server.go:127-161`)、サーバーの環境は呼んだ側の `os.Environ()` から pane の変数と `CLAUDECODE`・`CLAUDE_CODE_*` を除いたもの(`orgHerdrServerEnv`、302-315)。`CLAUDE_CONFIG_DIR` は残り、pane はサーバーの環境を引き継ぐ
- `ralph org reseat --help`(`internal/cli/org_reseat.go:123-127`)と `/director` skill の「座席を立て直す」3(`.claude/skills/director/SKILL.md:1509-1520`)は、別のアカウントへ移すときは人が `herdr server stop` を打ち、新しい環境の shell から最初の reseat を打つ、と書く。director はサーバーを止めない(「しないこと」2276 行)
- ralph はサーバーの pid も環境も記録しない。herdr の `status server --json` も pid と環境を出さない
- herdr のサーバーはユーザーに 1 つで、台帳は repo ごと。1 つの台帳を見ても、ほかの repo の org や、ralph の外の pane があるかは分からない(2026-11-20、このマシンのサーバーには別の repo の org の 3 つの pane と、ただの shell の pane があった)
- `herdr server stop` はすべての pane を閉じ、プロセスを殺す(`docs/evidence/org-reseat-live-2026-11-12.md:240-268`)。起動し直すと、`session.json` から pane の記録だけが戻り、端末もプロセスもない pane が `pane list` に出る(同じ evidence の 263〜270 行)
- サーバーの環境は読める(2026-11-20、このマシン、macOS 26.5.2、herdr 0.7.5 で手で確かめた): `herdr status server --json` の `socket`(`~/.config/herdr/herdr.sock`)に `lsof -t` を当てると、サーバーの pid が 1 つだけ返り(コマンドは `/opt/homebrew/bin/herdr`)、`ps eww -o command= -p <pid>` で環境の 96 項目が読め、`CLAUDE_CONFIG_DIR` も見えた(値は記録していない)。Apple のプラットフォームのバイナリは `ps eww` で環境が出ないが、Homebrew の herdr はそうではない

### Desired state

- 人が「座席を X のアカウントへ移して」と頼んだとき、director か人が 1 つの動詞で済ませられる。ほかの pane を巻き込む場合は動詞が断る
- 動いているサーバーがどのアカウントの環境かを、人と director が確かめられる

## Requirements

### Functional requirements

- [x] FR-1 `ralph org reseat --org-id <id> --restart-herdr`: 今の reseat の確かめ(`PlanReseat` の 1〜9)のあと、`reseat.lock` の中で、座席を止めたあと、座席を立てる前に、herdr のサーバーを止めて起動し直す。起動は今の herdr のサーバーのゲート(`ensureOrgHerdrServer`)を使う(呼んだ側の環境)
  - 実装の順(2026-11-20、実装で決めた): 上の「`PlanReseat` の 1〜9」の 9 は、見張りと herdr のサーバーのゲートだった。実装では、`--restart-herdr` 自身の確かめ(呼んだ側が herdr の pane の中でない、`--seat` が active な座席の全部を含む、サーバーが動いていて状態を読める。`reseatRestartHerdrErr`)を 9 に、ゲートを 10 にした。ゲートは止まったサーバーを起動するので、断られる `--restart-herdr` がサーバーを起動しないよう、確かめを先にしたため。上の「1〜9」は、この順では 1〜8 と 10 に当たる。そのあとに `Reseat` が 1 回目の pane の確かめを読む(`--dry-run` でも)。座席を止めたあとは、2 回目の pane の確かめ、`herdr server stop`、止まるまでの待ち、ゲートでの起動、座席の起動の順
- [x] FR-2 止める前の確かめ。次のどれか 1 つでも当たれば、何もせずに終了コード 1 で断る。
  - 呼んだ側が herdr の pane の中にいる(`HERDR_PANE_ID` が空でない)。サーバーと一緒に自分も消えるため
  - `herdr pane list` の生きた pane のうち、その org の座席の pane(台帳の active な座席の pane id)でないものが 1 つでもある(ほかの repo の org、herdr の中の headless の director、ralph の外の shell を含む)。断る文に、その pane の id、label、cwd を並べる。前にサーバーを止めたあとの、記録だけが戻った pane(端末もプロセスもない)もここで当たるので、断る文で `herdr pane process-info <id>` での見分け方と、閉じ方を案内する(断る側に寄せる)
  - herdr の TUI は pane ではないので断らないが、サーバーを止めると TUI も閉じることを出力に 1 行書く
  - `herdr pane list` が読めない、またはサーバーが動いていない(動いていなければ止める必要がないので、`--restart-herdr` なしの reseat と同じに起動する、と案内する)
- [x] FR-3 確かめはサーバーを止める直前、座席を止めたあとにもう一度行う(座席を止めているあいだに、ほかの pane ができていないこと)。2 回目に当たれば、サーバーは止めず、止めた座席は今の reseat の失敗の扱い(座席の記録は stopped、`reseated` は記録しない)で終わり、打ち直しを案内する
  - 実装で決めたこと(2026-11-20、verify の V1 を受けて `/sync-docs` で足した): 2 回目の確かめで断ったときは座席を止めたあとなので、`reseated` は記録する。上の括弧の「`reseated` は記録しない」は、座席を 1 つでも止めるか立てようとした reseat は途中で終わっても `reseated` を 1 件書く、という前の仕様の FR-8(`docs/specs/2026-11-12-org-reseat.md`)に合わせて、この注のとおりに読む(`internal/org/reseat.go` の `finish`。AC4 のテスト `TestReseat_RestartHerdr_APaneMadeWhileTheSeatsStopKeepsTheServer` が固定する)。サーバーは止めず、座席は止めたまま、打ち直しを案内する点は変わらない
- [x] FR-4 止め方: `herdr server stop` を打ち、`herdr status server --json` が running でなくなるまで、決まった時間(案: 10 秒)待つ。過ぎたら断り、サーバーは止まったかもしれないと出す
- [x] FR-5 `--claude-config-dir <dir>`: reseat のあいだに ralph のゲートが herdr のサーバーを起動するとき、その環境の `CLAUDE_CONFIG_DIR` をこの値にする(呼んだ側の値を置き換える)。`--claude-config-dir ''` は `CLAUDE_CONFIG_DIR` を外す(既定のアカウントへ戻す。`docs/specs/2026-11-12-org-reseat.md` の D6 (1) の場面)。空でない `<dir>` は存在するディレクトリであること。`--restart-herdr` なしでも受け付ける(サーバーが止まっていてゲートが起動するときに効き、動いているサーバーには効かないことを出す)。人が新しいアカウントの shell を開かずに、director に頼めるようにするため
- [x] FR-5b 打ち直し: 止めたあとの起動に失敗した reseat を打ち直す道(`Retry`、`internal/org/reseat.go`)に、`--claude-config-dir` の値を写す。打ち直しが呼んだ側の環境でサーバーを起動して、古いアカウントに戻らないため
- [x] FR-6 見える化: reseat(`--dry-run` も)は、動いているサーバーの `CLAUDE_CONFIG_DIR` を読めたら 1 行出す(`herdr server: CLAUDE_CONFIG_DIR=<値>` または `unset`)。読めなければ「読めない」と理由を出す。`--restart-herdr` なしで、移したいアカウントの値(`--claude-config-dir` の値、なければ呼んだ側の値)とサーバーの値が違えば、座席はサーバーの環境で立つのでそのアカウントにはならない、と 1 行知らせる(断らない)
- [x] FR-7 サーバーの環境の読み方: サーバーの pid を `lsof -t <socket>`(`herdr status server --json` の `socket`)で引き、`ps eww -o command= -p <pid>` の出力から `CLAUDE_CONFIG_DIR=` を読む。pid が 2 つ以上返ったら、コマンドが herdr のものに絞り、それでも 1 つにならなければ「読めない」。どれかが失敗すれば「読めない」。値に空白があると `ps` の出力では区切れないので、次の `KEY=` までを値とする。Linux では `/proc/<pid>/environ` を読む(案)。この読み方が使えない環境では、ralph のゲートが起動したときの pid を台帳に残す案に替える
- [x] FR-8 文書: `ralph org reseat --help`、`/director` skill の「座席を立て直す」3 と「しないこと」(人に頼まれたときに `--restart-herdr` を打ってよい、と例外を書く)、`/org` skill の同じ手順、`docs/specs/2026-11-12-org-reseat.md` に追記

### Non-functional requirements

- [x] NFR-1 `--restart-herdr` なしの reseat の挙動は、FR-6 の 1〜2 行の知らせのほかは変わらない
- [x] NFR-2 黙ってほかの pane を消さない。確かめられないときは止めない
- [x] NFR-3 herdr と ps と lsof の呼び出しには、今の herdr の呼び出しと同じく期限を付ける

## Acceptance criteria

- [x] AC1 Given そのサーバーに対象の org の座席の pane だけがある(fake の herdr), when `reseat --restart-herdr`, then 座席を止め、`herdr server stop` を打ち、running でなくなったあとゲートで起動し、座席を立て、`reseated` を記録する
- [x] AC2 Given ほかの pane が 1 つある, then 何も止めずに断り、その pane の id・label・cwd を出す(終了コード 1)
- [x] AC3 Given 呼んだ側に `HERDR_PANE_ID` がある, then 断る
- [x] AC4 Given 座席を止めたあとにほかの pane ができた, then サーバーを止めずに終わり、打ち直しを案内する
- [x] AC5 Given `--claude-config-dir /x`, then 起動するサーバーの環境の `CLAUDE_CONFIG_DIR` が `/x`(fake の起動で環境を確かめる)。`--claude-config-dir ''` なら環境に `CLAUDE_CONFIG_DIR` がない。存在しないディレクトリは断る。止めたあとの起動に失敗して打ち直すと、打ち直しの起動にも同じ値が渡る
- [x] AC6 Given サーバーの環境が読める(fake の lsof と ps), then reseat が `CLAUDE_CONFIG_DIR` を出し、呼んだ側の値と違えば知らせる。読めなければ理由を出し、断らない
- [x] AC7 `--restart-herdr` なしの今の reseat のテストが変わらずに通る
- [x] AC8 文書(help、director と org の skill の 4 写し)が、新しい flag と確かめを書く

## Constraints

### In scope

- `internal/org/reseat.go`(止める位置と確かめ)、`internal/cli/org_reseat.go`(flag、知らせ)、`internal/cli/org_herdr_server.go`(止める、環境の上書き、環境の読み)、herdr のドライバ(`pane list`、`server stop`)、テスト、文書

### Out of scope

- ほかの org の座席をまとめて別のアカウントへ移すこと(1 回の reseat は 1 つの org)
- `ralph status` と `ralph doctor` での見える化(案 3 の残り。reseat の出力で足りなければ次の変更で)
- keychain の扱いの確かめ(`--claude-config-dir` は、人がその値を持つ shell から起動したときと同じ環境を作るだけ)
- herdr の `live_handoff` での引き継ぎ

## Decisions(推す案)

- D1 新しい動詞ではなく reseat の flag にする。サーバーを止めるのは座席をほかのアカウントへ移すときだけで、そのとき必ず reseat が続く。止める・起動する・座席を立てるを 1 回の lock の中で済ませると、止めてから立てるまでのあいだに別の動詞が割り込まない
- D2 安全の確かめは台帳ではなく herdr の生きた pane の一覧で行う。台帳は repo ごとで、サーバーはユーザーに 1 つなので、台帳ではほかの repo の org と ralph の外の pane が見えない
- D3 `--claude-config-dir` を足し、空の値で外せるようにする。director のセッションの環境は古いアカウントのままなので、呼んだ側の環境で起動するだけでは、director に頼んだときに移らない。D6(2026-11-12-org-reseat)が退けたのは pane ごとに herdr の `--env` で渡す案で、ここで変えるのはサーバーの環境そのもの(人がその値の shell から起動したときと同じ)。HOME は変えない(keychain の注意は HOME を動かしたときのもの)
- D5 consult の指摘で、打ち直しにも値を写し、`--restart-herdr` なしでも flag を受ける(起動に失敗したあとの打ち直しが古いアカウントに戻らないため)
- D4 見える化は reseat の出力に留め、環境は `lsof` と `ps` で読む。読めなくても断らない

## Open questions

- `lsof` がない環境(最小の Linux)。`/proc` を読めば足りるか
- herdr の `pane list` が、herdr の TUI の中の pane も含むか(含むはず、未確認)
- `herdr server stop` のあと、同じソケットで起動し直すまでの待ち(evidence では stop が返ったあとプロセスは残らなかった)
- `--claude-config-dir` で起動したサーバーで、claude の座席が新しいアカウントの資格で動くこと(実地の確かめは、2 つのアカウントがある人の手でしかできない)

## References

- issue #354
- `docs/specs/2026-11-12-org-reseat.md`(D6、185 行)
- `docs/evidence/org-reseat-live-2026-11-12.md`(240〜268 行)
- `docs/recipes/isolated-herdr-server.md`(46、53〜56、316〜318、493 行)
