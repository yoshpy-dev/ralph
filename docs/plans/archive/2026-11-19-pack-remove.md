# pack-remove

- Status: Approved
- Approved: 2026-11-19 sha256:88ea9bb6774f
- Owner: Claude Code
- Date: 2026-11-19
- Related request: 下流の repo(ralph 8.2.1)で、`ralph init` の言語 pack の問いで使わない pack(golang、terraform、typescript)を選んでしまい、外す手段がなかった。`ralph pack` のサブコマンドは add と list だけ。issue #340〜#344 を 4 レーンで直す系列のレーン D として直す(2026-11-19)
- Related issue: #344
- Type: feat
- Branch: feat/pack-remove

## Objective

`ralph pack remove <language>` を足し、入れた言語 pack を外せるようにする。外したあとの repo では `ralph doctor --strict` が通り、`ralph upgrade` は外した pack を戻さない。

## Scope

- `internal/scaffold/manifest.go`: `Manifest.Write` を、同じディレクトリの一時ファイル(`os.CreateTemp`)に全部書き、mode を 0644 にし、閉じてから `os.Rename` で置き換える形にする。どこかで失敗したら一時ファイルを消し、元の manifest は変えない。`internal/org/watchalive.go` の `writeJSONFileAtomic` と同じ形。呼び手(init、pack add、upgrade、eject、adopt、migrate、pack remove)はそのまま。テスト用に rename を unexported の変数にする
- `internal/cli/pack_remove.go`(新規): `ralph pack remove <language> [--dry-run]`
  - `.ralph/manifest.toml` がない、または v2 でない(legacy)なら、`ralph pack add` と同じく書き込みなしで断る(legacy は `errLegacyLayoutFailClosed` をそのまま返す)
  - `Meta.Packs` にない pack はエラーにする。文に入っている pack の一覧を添える
  - 名前が空、`/` や `\` を含む、`.` か `..` のときは、manifest を読む前に断る。`packPrefixFor` は文字列の連結なので、改ざんされた manifest の `Meta.Packs` から repo の外を指すパスを作らせない
  - manifest の項は `delete(m.Files, path)` で外す(`Manifest` に項を消すメソッドはない)
  - 対象: manifest の項のうち、`packPrefixFor(lang)`(`packs/languages/<lang>/`)の下にあるもの、`packRuleRelPath(lang)`(`.claude/rules/ralph/<lang>.md`)、`legacyPackRuleRelPath(lang)`(`.claude/rules/<lang>.md`。migration が残した場合)
  - 項ごとの扱い
    - disk にない: 項だけ外す(手で消したあとの repo を直せる)
    - 通常のファイルで、hash が manifest の記録(`DiskHash`、空なら `Hash`。upgrade の `classifyPath` と同じ読み方)と同じ、かつ owner が core: ファイルを消し、項を外す
    - hash が違う(変更された)、owner が fork(`ralph eject` 済み)、または通常のファイルでない(symlink、ディレクトリ): ファイルは残し、項を外し、理由を付けて名前を出す
  - 消すのは `upgrade.ApplyOps` の `OpDelete` に任せる。ApplyOps は全部の op のパスと親の連なりを先に確かめ、だめなら 1 つも消さずにエラーを返す
  - 消したあと、`packs/languages/<lang>/` の下の空になったディレクトリを下から消す。続けて `packs/languages/` と `packs/` も空なら消す。中にファイルが残れば消さず、残ったことを 1 行出す
  - 最後に manifest から項と `Meta.Packs` の pack を外して書く。書けなければエラーにし、同じコマンドを打ち直せば終わると出す。書き込みは package の変数(仮名 `writePackRemoveManifest`)を通し、テストで失敗させられるようにする
  - `--dry-run`: 消すもの、残すもの、外す項を出し、何も書かない
- `internal/cli/pack.go`: `newPackCmd` に remove を足す。`addPack` の doc コメントは変えない
- `internal/cli/init.go` の `Long`: 「A pack can be added later with ralph pack add <language>.」に remove を足す
- 文書
  - `README.md` の Commands の表(135 行)に `ralph pack remove <lang> [--dry-run]` の行を足す。Language packs の節(318 行付近)に外し方を 1 つ足す
  - `templates/base/docs/recipes/adding-a-language-pack.md` に「Removing a pack」の節を足す(何を消し、何を残し、残ったファイルをどうするか、upgrade が戻さないこと)
  - `docs/recipes/adding-a-language-pack.md`(ralph の repo の貢献者向け)は pack を作る手順なので変えない。`scripts/check-sync.sh` は 2 つの差を既知の差として扱っている
  - ほかの文書は足さない。`/sync-docs` の skill は既に「Language packs added/removed」を見る対象にしている(`templates/base/.claude/skills/sync-docs/SKILL.md:28`)
- テスト(`internal/cli/pack_remove_test.go`、新規)

## Non-goals

- `ralph pack add` の挙動(既にあるファイルを上書きする)の変更。外したあと残ったファイルは、同じ pack を add し直すと上書きされる。recipe にそう書くだけにする
- 言語の marker ファイル(go.mod など)、`scripts/detect-languages.sh`、`run-verify.sh` の pack の選び方
- legacy の manifest の pack を外すこと(先に `ralph upgrade` で v2 にする)
- `ralph init` の対話の問いの変更
- 次の release の注記(`internal/upgrade/notes/<版>.md`)。`/release` の段で書く。候補は A に送る
- `ralph pack add` が manifest の書き込みの失敗を警告で流す(`internal/cli/pack.go:114`)こと。別の穴として A に送る

## Assumptions

- pack が書くファイルは `renderPackInto`(`internal/cli/language_pack.go`)が書く 2 か所だけ: `packs/languages/<lang>/` の下と `.claude/rules/ralph/<lang>.md`。init と add はどちらもこの関数を使う(確かめ済み)
- pack のファイルの owner は core(`ownerForScaffoldPath` が `packs/` と `.claude/rules/ralph/` を core にする)。`ralph eject` したものだけが fork になる(確かめ済み)
- upgrade は `oldManifest.Meta.Packs` の pack だけを desired に入れる(`internal/cli/upgrade_v2.go:400`)。`Meta.Packs` と項を外せば、そのパスは desired にも manifest にもなく、replace planner は何もしない(`PlanCoreReplaceDesired` は desired と manifest の和だけを見る)。テストで確かめる
- doctor --strict が見るのは、core のパスの drift(`plan.Drift`)、manifest にあって disk にないパスなど 5 つ。manifest にない、desired にもないファイルはどれにも当たらない(`internal/cli/doctor.go` の `checkScaffoldIntegrity`)
- `run-verify.sh` は marker で言語を選び、`packs/languages/<lang>/verify.sh` が実行可能なら打つ。変更された verify.sh を残すと、marker がある限り打たれ続ける。それは残す理由(利用者が変えたもの)と合うので、出力で知らせるだけにする

## Affected areas

- `internal/scaffold/manifest.go`、`internal/scaffold/manifest_test.go`(manifest のすべての書き込みに効く)
- `internal/cli/pack_remove.go`(新規)、`internal/cli/pack.go`、`internal/cli/init.go`(help の 1 文)
- `internal/cli/pack_remove_test.go`(新規)
- `README.md`、`templates/base/docs/recipes/adding-a-language-pack.md`
- 共有ファイル: `README.md`(135 行。B の #340 が 136 行を触る。後から merge する側が衝突を直す)

## Visual review

- ページ: `.harness/state/plan-visual/pack-remove.html`(やさしい図 2 枚: overview、easy-after。折りたたみに詳細図 3 枚: detail-overview、detail-classify、detail-order)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: 全体と 5 枚の図を `shot` で撮って読んだ。重なり、はみ出し、矢印の外れはない(図 2 の ✓ の位置を 1 度直した)

## Design decisions

- 変更されたファイル、fork のファイルの manifest の項は外す(自律で選んだ)。残すと、desired から外れた core の項は replace planner で template-absent かつ disk の hash が違うので `plan.Drift` に入り(`classifyCore`、`internal/upgrade/replaceplan.go:369`)、`doctor --strict` の「core file hashes」が落ち、upgrade も毎回 drift と報告する。fork の項は `classifyFork` が毎回 advisory を出し続ける(同 389 行)。外せばファイルは利用者のものになり、doctor も upgrade も触らない。受け入れ条件の「doctor --strict が通る」を、変更したファイルが残る場合にも満たせる
- `--dry-run` を付ける(自律で選んだ)。消すコマンドなので、先に何が消えるかを見られる方がよい。`ralph upgrade` にも同じ名前の flag がある。足す量は出力の分岐 1 つ
- `Meta.Packs` にあるが今の binary にない pack(retired)も外せる。消すかの判断は manifest の記録した hash だけで行い、template を読まない。upgrade はこうした pack を残し続ける(`buildDesiredStateV2` の preserve)ので、外す手段はこのコマンドだけになる
- 通常のファイルでないもの(symlink など)は消さずに残す。ApplyOps は symlink を見るとすべてを止めるので、op にする前に分ける。親の連なりに symlink があるときは ApplyOps が 1 つも消さずに止める。upgrade と同じ扱いにする
- 書く順は、ファイルを消してから manifest を書く(upgrade の commit barrier と同じ)。途中で止まっても、打ち直せば disk にない項を外して終わる
- manifest の書き込みは `Manifest.Write` 自体で一時ファイルと rename にする(自律で選んだ。Codex の計画助言 [HIGH] を受け、consult-plan-packremove に範囲を聞いて判定 A)。`os.WriteFile` は先に中身を切り詰めるので、ファイルを消したあとに書き込みが途中で止まると manifest が空か途中までになり、打ち直しもできない。呼び手の 6 か所(adopt.go:174、init.go:472、pack.go:114、eject.go:102、upgrade_v2.go:704、migrate.go:958)は同じ穴を持ち、upgrade の commit barrier もその 1 つなので、remove だけに `WriteAtomic` を足す案(B)は採らない。挙動が変わるのは、manifest が symlink のとき(symlink がファイルに置き換わる)と、既存のファイルの mode が 0644 でなかったときだけ

Critical forks: None(上の判断は /consult の consult-plan-packremove に聞き、判定「進めてよい」を受けて自律で選んだ。判断の材料は repo の中にある)

## Acceptance criteria

- [x] AC-1: golang を入れた v2 の repo で `ralph pack remove golang` を打つと、`Meta.Packs` から golang が消え、`packs/languages/golang/` の下のファイルと `.claude/rules/ralph/golang.md` が消え、それらの manifest の項がなくなる。`packs/` が空なら `packs/` も消える。そのあと `checkScaffoldIntegrity` の 5 つが strict でも pass し、`checkInstalledPacks` は `Pack: golang` の行を出さない
- [x] AC-2: 外したあと `ralph upgrade`(次の世代の template)を打っても、golang のファイルは戻らず、`Meta.Packs` にも manifest にも golang は入らない
- [x] AC-3: 変更したファイル(hash が違う)と `ralph eject` したファイルは消えず、出力に名前と理由が出る。その項は manifest から外れ、doctor --strict は pass し、続く upgrade もそのファイルを変えない
- [x] AC-4: 入っていない pack(`Meta.Packs` にない)を渡すとエラーになり、disk も manifest も変わらない
- [x] AC-5: legacy の manifest では `errLegacyLayoutFailClosed` を返し、何も書かない。manifest がなければエラーにする
- [x] AC-6: `--dry-run` は消すものと残すものを出し、disk も manifest も変えない
- [x] AC-7: ほかの pack(typescript)のファイルと項は変わらない
- [x] AC-8: 手で消したあとの repo(disk にない項)でも、項を外して終わり、doctor --strict が pass する
- [x] AC-9: README の Commands の表と Language packs の節、templates の recipe に remove が書かれている
- [x] AC-10: `Manifest.Write` の rename が失敗しても、元の manifest はそのまま読め、ディレクトリに一時ファイルが残らない。成功したときの中身は今の Write と同じ toml になる
- [x] AC-11: pack remove でファイルを消したあと manifest の書き込みが失敗すると、エラーになり、元の manifest はそのまま読め(ほかの pack の項も残る)、同じコマンドを打ち直すと終わって doctor --strict が pass する

## Implementation outline

1. `Manifest.Write` を一時ファイルと rename にする(`internal/scaffold/manifest.go`、`internal/scaffold/manifest_test.go`) — verify: `test -z "$(gofmt -l internal/scaffold)" && go vet ./internal/scaffold/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/scaffold/`
2. `ralph pack remove` の本体とテスト(`internal/cli/pack_remove.go`、`internal/cli/pack_remove_test.go`、`internal/cli/pack.go`、`internal/cli/init.go` の help の 1 文) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Pack|Init|Help|Scaffold|Upgrade|Eject|Adopt'`
3. 文書(`README.md`、`templates/base/docs/recipes/adding-a-language-pack.md`) — verify: `./scripts/check-sync.sh && ./scripts/check-template-purity.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck)
- Spec compliance criteria to confirm: AC-1〜AC-11 を issue #344 の受け入れ条件 3 つと照らす
- Documentation drift to check: README の Commands の表、templates の recipe、`ralph init --help` と `ralph pack --help` の文、`scripts/check-sync.sh`
- Evidence to capture: verify の報告(`docs/reports/verify-2026-11-19-pack-remove.md`)

## Test plan

- Unit tests: 項ごとの分類(消す、残す、項だけ外す)、入っていない pack、名前の形が不正な pack、legacy、manifest なし、`--dry-run`。`Manifest.Write` の rename の失敗(元の中身が残り、一時ファイルが残らない)、書き込み先のディレクトリがないときのエラー、成功したときの読み戻し(AC-10)。remove の manifest の書き込みの失敗と打ち直し(AC-11)
- Integration tests: `initV2Project`(golang 入り)→ remove → `checkScaffoldIntegrity` strict(AC-1)、remove → 次の世代で `runUpgradeIOWithOptions`(AC-2)、upgrade で pack のファイルが更新された(`DiskHash` が付いた)あとの remove
- Regression tests: `ralph pack add` の既存のテスト、`ralph init` の help のテスト、manifest を書く init・upgrade・eject・adopt・migrate の既存のテストが通ること(`.ralph/` の中身を数えるテストに一時ファイルが残らないこと)
- Edge cases: 変更したファイル、eject した rule、symlink の pack のファイル、pack のディレクトリに利用者が足したファイル(ディレクトリは残る)、retired の pack、2 つの pack のうち 1 つだけ外す、手で消したあと
- Evidence to capture: test の報告(`docs/reports/test-2026-11-19-pack-remove.md`)

## Risks and mitigations

- 利用者が変えたファイルを消す: hash が記録と同じ core のファイルだけを消す。fork と通常でないファイルは消さない。テストで変更したファイルと eject したファイルが残ることを確かめる
- symlink を通して repo の外を消す: `upgrade.ApplyOps` の検査(パス、親の連なり、葉の Lstat)をそのまま使う
- 途中で止まり、manifest と disk がずれる: 消してから manifest を書く順にし、打ち直しで disk にない項を外せるようにする(AC-8)。manifest の書き込み自体が途中で止まっても元の manifest が残るよう、Write を一時ファイルと rename にする(AC-10、AC-11)
- `Manifest.Write` の変更が init や upgrade に波及する: 書く中身は変えず、書き方だけを変える。manifest を書く既存のテストを slice 2 の verify と /test の全体で回す。symlink の manifest がファイルに置き換わることは release ノートの候補に書く
- README の 135 行と B の 136 行の衝突: merge の直前に最新の main を取り込み、A に一言送る

## Rollout or rollback notes

- 新しいサブコマンドを足すだけで、既存のコマンドの挙動は変えない。戻すときはこの PR を revert する
- release ノートの候補: 「`ralph pack remove <language>` で入れた言語 pack を外せる。変更したファイルは残して名前を出す」「`.ralph/manifest.toml` の書き込みが一時ファイルと rename になった。manifest を symlink にしていると、ファイルに置き換わる」。A に送る

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh、rc=0、git の状態は前後で同じ)。所見 1 件: [HIGH] `Manifest.Write` は `os.WriteFile` で先に切り詰めるので、ファイルを消したあとに書き込みが止まると manifest が壊れ、打ち直せない。ユーザーの決まり(計画助言の修正提案は聞かずに直す)に従って反映した: consult-plan-packremove に範囲を聞き、判定 A で `Manifest.Write` 自体を一時ファイルと rename にする slice 1、AC-10、AC-11、Design decisions と Risks の項を足した
- [x] Plan approved — ユーザーの決まり(計画は推す案で自動承認)により 2026-11-19 に承認
- [x] Branch created
- [x] Implementation started
  - slice 1(implementer、c593c00f): `Manifest.Write` を一時ファイルと rename に。verify(gofmt、vet、`go test ./internal/scaffold/`)と呼び手の回帰テスト(`go test ./internal/cli/ -run 'Init|Upgrade|Pack|Eject|Adopt|Migrat|Scaffold|Doctor'`)が pass。self-check 5 項目 clean
  - slice 2(implementer、39bc44b4): `ralph pack remove`。verify(gofmt、vet、`go test ./internal/cli/ -run 'Pack|Init|Help|Scaffold|Upgrade|Eject|Adopt'`)が pass。self-check はコメント 1 か所を fixed、ほかは clean。計画からの足し: manifest の key を整えたあと pack の 3 か所から外れるもの(`packs/languages/golang/../../AGENTS.md` など)は何も書かずにエラーにする
  - slice 3(implementer、33cdace4): README と templates の recipe。verify(`./scripts/check-sync.sh && ./scripts/check-template-purity.sh`)が pass。self-check 5 項目 clean
  - inline の直し(8cbe2ea2): recipe の「init が選んだ」を「init の問いで選び間違えた」に。verify は同じ 2 本が pass。self-check(staged の 2 行): 不要な変更・typo・デバッグの残り・secret・範囲外のファイル、すべて clean
  - self-review の LOW 6 件の直し(implementer、42a87923): `packs` か `packs/languages` が symlink なら空ディレクトリを片付けない、断りの文に「nothing was changed」、書き込み失敗の文、dry run のディレクトリの行、番兵のコメント、テスト名。テストのファイル分け(34f91f36)
  - inline の直し(4e98b216): `doctor.go` のコメントに pack remove を足した。verify(gofmt、vet、`go test ./internal/cli/ -run 'Pack|Doctor'`)が pass。self-check 5 項目 clean
  - verify の V1 の直し(implementer、039e0d7f): 「残したものは add が上書きする」を、ファイル・symlink・ディレクトリごとの挙動に直した(4 か所とテスト)
  - test の G1 の直し(implementer、4426eb3f): 旧い rule の場所の項のテスト 2 本と、`Write` が symlink を置き換えるテスト。変異で落ちることを確かめた
  - sync-docs(cycle 1。報告 `docs/reports/sync-docs-2026-11-19-pack-remove.md`): V2 は tech-debt の「800 行を超えた 5 つのファイル」の行に、`doctor.go` の comment だけの変更を持ち越す注を足した。新しい行を 1 つ足した(`ralph pack add` が pack のパスの symlink をたどる、ディレクトリで止まる)。I1〜I5 は記録だけ。AC-1〜AC-11 のチェック欄は、verify と test の report が満たしたとする 11 件を `[x]` にした(digest は変わらない)
- [x] Review artifact created — docs/reports/self-review-2026-11-19-pack-remove.md(2 回目: 指摘 0、Merge yes)
- [x] Verification artifact created — docs/reports/verify-2026-11-19-pack-remove.md(partial pass。V1 は 039e0d7f で直した)
- [x] Test artifact created — docs/reports/test-2026-11-19-pack-remove.md(Pass、run-test.sh 56/56。G1 は 4426eb3f で埋めた。G1 のテストは通しの run-test.sh には入っておらず、絞った `go test` で pass)
- [ ] PR created
