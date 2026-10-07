# sync-docs report: org-stop-all

- Date: 2026-10-08(JST。insight event の ts は UTC の 2026-10-07)
- Plan: `docs/plans/active/2026-10-07-org-stop-all.md`
- Pipeline cycle: 2 of 2(既定の上限の最後の回)。差分は base `f423f230`(origin/main)から branch `feat/org-stop-all` まで。作業の開始時の HEAD は `8e3d951e`
- 先行 report: `docs/reports/self-review-2026-10-07-org-stop-all.md`(cycle 2。Merge 可、CRITICAL・HIGH・MEDIUM なし、新しい指摘は LOW の C2-1〜C2-6)、`docs/reports/verify-2026-10-07-org-stop-all.md`(cycle 2。pass、AC1〜AC16 Met、V2-1〜V2-6)、`docs/reports/test-2026-10-07-org-stop-all.md`(cycle 2。pass、mutation 64 件中 63 件が red、残る N37 は等価)
- cycle 1 の sync-docs(D-1〜D-4)は `2c97407c` と `223258eb` にある。この report はそれを上書きした。登録簿や plan から cycle 1 の節を指す箇所はない(`git grep` で確認)ので、付録には残していない

## Summary

verify と self-review が /sync-docs に渡した指摘(V2-1〜V2-6、C2-3、C2-4)をすべて直し、self-review の「Tech debt identified」の 2 行と test report の cycle 2 の穴を登録簿に足した。S8(後回しにした自分の pane / workspace の close が失敗したときの台帳の補償)で古くなっていたのは、`/org` skill の `stop` と `disband` の行、`disband --help` の最後の段落、leader の雛形の手順 8、登録簿の 163〜168 行の行番号、plan の進捗の 1 行だった。README、仕様 FR-2、recipe の 2 面、AGENTS.md の Repo map は S8 の細部を書く粒度ではなく、変更していない。

commit 1(`3f52d676`)は `internal/cli/org.go` の `disband` の Long と `internal/org/prompts/leader.md` の文面を変えている。どちらも文字列だけで動作は変えていないが、`internal/` のファイルなので、cycle 2 の self-review、verify、test はこの diff を見ていない。パイプラインは上限に達しているので回し直さず、`/cross-review` の入力に入れる。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md` | V2-1、V2-5。`:167` の `stop` の行の末尾に 1 節(最後の close が失敗したら座席を active に戻して終了コード 1、打ち直すか別の pane の `--all` で閉じ直せる、`--force` は戻さず警告で終了コード 0)。`:168` の `disband` の行に同じ趣旨の 1 節(その pane の座席を active に、後回しにした workspace を open に戻す)と、行末の「解散した org_id でまた `spawn` すると新しい workspace を作る」への例外(最後の close の失敗で開き直した workspace は再利用する)。`:333-335` の運用の締めの項に、最後の close が失敗したときは先に stdout に出た `stopped seat` / `disbanded org` の行より終了コードと stderr が正しい、と 1 文。`stop` のセルは 817 字から 919 字(1,639 バイトから 1,851 バイト)で、足したのは 1 節だけ。skill の本文は日本語なので、依頼の「skill の文は英語」ではなく、ファイルに合わせて日本語で書いた |
| `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 上の 3 つの写し。`scripts/sync-skills.sh` は `.agents/skills/org/SKILL.md` だけを作り直したので、`templates/base/` の 2 面は `cp` で揃えた。4 面の md5 は同じ(`c6621f43b93b68eee85bcbe0b2746634`) |
| `internal/cli/org.go` | V2-2、C2-4。`disband` の Long の最後の段落(`:1030-1034`)を「最後の close が失敗したら、その pane の座席を active に戻し、workspace を閉じようとしていたときはその workspace も open に戻して終了コード 1」に直した。pane だけを後回しにした場合(自分の pane が org の workspace の外にある)は座席だけが戻る。`ralph org disband --help` を実際のバイナリで出して折り返しを確かめた。`stop` の Long(`:741-744`)は座席だけを言っていて正確なので変えていない |
| `internal/org/prompts/leader.md` | V2-6。手順 8(`:34-42`)に 1 文。stderr に `herdr pane close` か `herdr workspace close` のコマンドが添えられているとき(台帳を読めない、または戻せなかった場合)は、打ち直しても閉じる対象が見つからないので打ち直さず、そのコマンドを添えて人に上げる。雛形のコピーは `git ls-files` で 1 本だけ(`templates/` にない) |
| `docs/tech-debt/README.md` | V2-3、C2-3。163〜168 行の `file:line` 57 件を関数名と動作に置き換えた(残る `file:line` は 0 件)。168 行の (a) から `CloseDeferredSelfPane` と `CloseDeferredSelfWorkspace` の台帳の読み取り失敗(旧 `:1003`、`:1045`)を外し、S8 の `TestOrgCloseDeferredSelf_NothingRestored_NamesTheManualClose` が届くことを書いた。166 行の F-8 の数(`stop` の行 約 920 字、`disband` の行 約 1,075 字)と、cycle 2 で `stop` と `disband` の両方が伸びたことを更新した。各行の Related の self-review、verify、test の ID は cycle 1 の付録の節に残っているので、`appendix, cycle 1:` を前に付けた。169 行に C2-1・C2-2・C2-5・C2-6 の行、170 行に cycle 2 のテストの穴の行を足した |
| `docs/plans/active/2026-10-07-org-stop-all.md` | V2-4。進捗の `:156` の 1 行だけ。Verify plan の「AC1〜AC13」は cycle 1 の時点では書き換えなかったが、S8 の承認のやり直し(`5f26af07`、digest `2f2cfde39e9f`)で AC1〜AC16 に直したことを書いた。本文のほかの行とチェックボックスは変えていない |
| `docs/insights/events/2026-10-07-org-stop-all.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 2) |
| `docs/reports/sync-docs-2026-10-07-org-stop-all.md` | この report |

## 指摘ごとの結果

| ID | 結果 | 場所 |
|----|------|------|
| V2-1 | 直した | `.claude/skills/org/SKILL.md:167`、`:168`(4 面) |
| V2-5 | 直した | `.claude/skills/org/SKILL.md:333-335`(4 面) |
| V2-2、C2-4 | 直した(tech-debt の行は足していない) | `internal/cli/org.go:1030-1034` |
| V2-6 | 直した | `internal/org/prompts/leader.md:34-42` |
| V2-3、C2-3 | 直した | `docs/tech-debt/README.md:163-168` |
| V2-4 | 直した | `docs/plans/active/2026-10-07-org-stop-all.md:156` |
| C2-1、C2-2、C2-5、C2-6 | tech-debt に 1 行 | `docs/tech-debt/README.md:169` |
| cycle 2 のテストの穴 | tech-debt に 1 行 | `docs/tech-debt/README.md:170`(`lastHerdrAgentName` の `return ""`、C2-1 と C2-2 のテストなし、本物の herdr で動かしていない、`NothingRestored` の確認が文言に依る、N37 は等価) |

依頼の項目 2(help の文)と 3(leader の雛形)は文字列だけの変更で済んだので、tech-debt に代わりの行は足していない。

## 実行した確認

| Command | Result |
|---------|--------|
| `gofmt -l internal` | 出力なし |
| `go test ./internal/cli/... ./internal/org/... -count=1` | `ok` 4 パッケージ(`internal/cli` 130.2 s、`internal/org` 11.9 s、`internal/org/driver` 1.8 s、`internal/org/protocol` 1.0 s)。leader.md の変更後。`internal/org` には雛形のテスト(`TestRenderRolePrompt_Leader_*`)が入っている |
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/run-static-verify.sh` | rc 0。shellcheck、hook の構文、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt README の plan の参照、golang verifier(`gofmt: ok`、`0 issues.`)、branch secret scan(`f423f230..3f52d676` clean。commit 2 の内容は commit 前なので範囲に入っていない) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-07-org-stop-all.md` | `2f2cfde39e9f`。plan の `- Approved:` 行と一致(進捗の節は digest の対象外) |
| 4 面の skill の md5 | すべて `c6621f43b93b68eee85bcbe0b2746634` |
| 登録簿 163〜168 行の `file:line` | 0 件(正規表現 `[A-Za-z_./]*:\d+(-\d+)?` で確認)。新しい行に書いた関数名とテスト名は `git grep` で実在を確かめた |

`go test` は leader.md と `internal/cli/org.go` を直したあとに流し、それ以降に変えたのは文書(tech-debt、plan の進捗の 1 行、この report、insight event)だけ。`gofmt`、`check-skill-sync.sh`、`check-sync.sh`、`run-static-verify.sh`、digest は、この report と insight event を書いたあとの作業ツリーで流し直した値。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `README.md`(`:124` の Commands の表、`:245` の org 節) | `stop` が pane を、`disband` が workspace を閉じること、`--all`、leader が `report` のあと `disband` を最後に打つことだけを書く。S8 の補償は書く粒度ではない。変更なし |
| `docs/specs/2026-10-07-org-multi-org-director.md` FR-2(`:36`) | 横断の `stop` / `disband`、`stop_failed`、`--force` の概要と、詳細は 2 段目の計画にあると書いている。S8 の細部は plan にある。変更なし。FR-2 のチェックボックスは PR がマージされるまで付けない |
| `docs/recipes/codex-seat-permissions.md` と `templates/base/` の写し | `cmp` で同一。後片付けの節(`:278-297`)は、操作者の shell から scratch の座席を止める手順で、自分の pane や workspace を閉じる経路ではないので S8 の補償は効かない。「どちらかが終了コード 1 なら、herdr が応答してから打ち直す」は今も正しい。reviewer 座席の probe は `reviewer.md` に依存し、この PR は `leader.md` の手順 8 だけを変えた。変更なし |
| `AGENTS.md` の Repo map(`:90` の `internal/org/` の行)、`.ralph/core/AGENTS.core.md`、`CLAUDE.md` | stop と disband の動作の粒度ではない。変更なし |
| `/org` skill の手順 8(`:258`)と運用の締め | leader の雛形の手順 8 と同じ趣旨(disband が最後のコマンド、記録と出力を済ませてから閉じる)。skill の手順 8 は打ち直しに触れないので、雛形の追記に合わせる必要はない |

## Found but left

- `disband` の skill の行の「その pane の座席を active に、後回しにした workspace を open に戻す」は、workspace 側の補償で座席を戻す条件(`HERDR_PANE_ID` が同じ org の `stopped` の座席の pane であること)を書いていない。leader が自分の workspace を閉じる通常の場合は条件を満たす。条件の細部は C2-2 の行(`docs/tech-debt/README.md:169`)にある。
- C2-4 は help の文で直したので、self-review の「Tech debt identified」の 2 行目(C2-4 を 1 行)は足していない。V2-6 の「直さないなら PR の known gap」も同じで、手順 8 の文で直した。
- stdout に先に出る `stopped seat` / `disbanded org` の行(V2-5)は、閉じる前に出力するしかないので変えていない。skill の運用の締めに、終了コードと stderr を信じる旨を書いた。
- 本物の herdr では動かしていない。help と skill と雛形の文は、コードとテスト(`TestOrgCloseDeferredSelf*`、CLI の `TestOrgStopDisband_OwnCloseFails_LedgerRestoredForRetry`、`TestOrgStopDisband_Force_OwnCloseFails_WarnsExitsZero`)の内容に照らして書いた。
- leader の雛形の規則が効く条件は、stderr に herdr のコマンドが添えられていること。`CloseDeferredSelfPane` / `CloseDeferredSelfWorkspace` がこのコマンドを添えるのは、台帳が読めない、座席・org の記録がない、補償の追記が失敗、`--force` のとき。括弧の例は台帳が読めない場合と戻せなかった場合だけを挙げていて、記録がない場合(`no seat recorded on it`。直前に `Stop` が記録するので実際にはほぼ起きない)は挙げていないが、同じ規則で足りる。
- `stop` の行のセルは 919 字になった(F-8、登録簿の 166 行に反映)。F-8 の整理(表のセルを 1〜2 文にして詳細を表の下に出す)は、4 面の写しを一緒に直す別の変更として残る。
