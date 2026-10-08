# sync-docs report: org-limits-reserve

## Cycle 1

- Date: 2026-10-08
- Plan: `docs/plans/active/2026-10-08-org-limits-reserve.md`
- Pipeline cycle: 1 of 2。差分は merge base `51855166` から、sync-docs 開始時の branch HEAD `fff1ec23`(feat/org-limits-reserve)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-08-org-limits-reserve.md`(`d8ec84da`。Merge 可、F-1〜F-11。F-1 と F-3 は `8dd19634` で直した)、
  `docs/reports/verify-2026-10-08-org-limits-reserve.md`(`fbb04f83`。pass、V-1〜V-9)、
  `docs/reports/test-2026-10-08-org-limits-reserve.md`(`fff1ec23`。pass、mutation 40 件のうち生き残りは X18 だけ、Test gaps 1〜8。テスト 2 本を `c3a95c48` で足した)

## Summary

verify が挙げた文書のずれ 9 件のうち、V-4 以外はこの cycle で閉じた。V-1・V-2・V-5 は `/org` skill、V-3 は quality-gates、V-6 と V-1 の help は `internal/cli/org.go` の文字列、V-7・V-8 は tech-debt の既存の 2 行、V-9 は plan のチェックボックスである。V-4(`{{ENVELOPE}}` と doctor の要約が `max_seats` だけを出す)は表示を変えるコードの変更なので、tech-debt の新しい行に送った。

tech-debt は、既存の 2 行(144 行目、170 行目)を更新し、新しい行を 5 つ足した(172〜176 行目)。start の `--scope` の help の表示崩れ(plan の (d))は、166 行目の (b) にすでにあるので足していない。

commit は 2 つに分けた。1 つ目は `76d1cf1c`(skill 4 面、quality-gates 2 面、help の文字列、spec、AGENTS.md)、2 つ目はこの report と同じ commit(tech-debt、plan のチェックボックス、report、insight)である。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`(203〜211、190〜195、223〜226、234〜236 行目)、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | V-1: main worktree の `ralph.toml` から 2 つの上限を読むのは「`--config` がなく、台帳の置き場所が main worktree のものであるときだけ」と書き直し、読まない場合の列挙(`--config`、`--state-dir` か env、git の外)に、台帳を git の toplevel から決めたとき(main worktree を決められない bare リポジトリの linked worktree)を足した。V-2: 「同時に打った `spawn` でも超えない」を「`spawn` どうしが同時に打たれても超えない」に直し、自分の pane か workspace を最後に閉じる `stop` / `disband` の補償は上限も予約の重なりも見ずに戻すので、1 つ超えるか重なることがあり、打ち直した `stop` / `disband` で解ける、という段落を足した。V-5: `--reserve` だけを渡した leader の `{{SCOPE}}` には予約したパスではなく既定の「未指定」の文言が入るので、範囲の説明を見せたいときは `--scope` も渡す、と足した。末尾に `/` がないパスはファイルで、`--reserve internal/auth` が守るのは `internal/auth` という名前のファイルだけ、と足した。折り返しは 4 面とも表示幅 79 以内。`./scripts/sync-skills.sh` のあと、2 つの `templates/base/` の写しは `cp` で揃えた(4 面の sha256 は `771be761…` で同じ) |
| `docs/quality/quality-gates.md`(77 行目)、`templates/base/docs/quality/quality-gates.md`(76 行目) | Envelope validation の行に `max_orgs`、`max_total_seats`、`--reserve` のパスが走っている他の org と重なること、を足した。依頼は `max_orgs` と `max_total_seats` だったが、V-3 が spawn の見る項目に予約の重なりも挙げているので、それも足した。`check-sync.sh` の KNOWN_DIFF の 2 面は同じ文にした |
| `internal/cli/org.go`(文字列だけ。挙動は変えていない) | 46 行目: 永続フラグ `--config` の説明から spawn と start だけの文を外し、「org 全体の上限をどこから読むかは `ralph org spawn --help` を見よ」の 1 つの指し示しにした(backtick は pflag が値の名前と読むので使っていない)。363 行目: `orgWideLimitsHelp` を「`ralph org spawn` and `ralph org start` check ...」で始め、start の読み手にも動詞が分かるようにした(F-11)。括弧書きの「(no --state-dir or RALPH_ORG_STATE_DIR)」は、`--state-dir` か env を使わない場合だけが main の読み元を止めるように読めたので、「--config を渡さず、台帳が main worktree のものであるときだけ読む。それ以外(--config、--state-dir か env で決めた台帳、git の toplevel から決めた台帳、git の外)は --config のファイルか ./ralph.toml を使う」に直した(V-1)。1005 行目: `ralph org status --help` に `Long` を足し、`reserved: <path>, ...` の行と `--json` の `reservation` を書いた(V-6) |
| `docs/specs/2026-10-07-org-multi-org-director.md`(41 行目) | FR-3 に「3 段目で決めたこと」の項目を 1 つ足した。FR-2 が 2 段目で決めたことを同じ行に書いて計画の名前を指すのと同じ形で、予約は任意、受け付けるのは `start` と leader の `spawn`、org ごとに 1 つ、走っている org の数え方、`--config` がないときの上限の読み元、を書き、計画 `2026-10-08-org-limits-reserve.md` を指す。FR-1 と FR-2 のチェックボックスを付けていない慣例に合わせ、FR-3 も付けていない |
| `AGENTS.md`(90 行目、managed block の外) | repo map の `internal/org/` の説明で、envelope validation に「per-org `max_seats`; org-wide `max_orgs` / `max_total_seats` and scope reservations in `reserve.go`」を足した。`reserve.go` はパスの規則、重なり、走っている org の導出を持つ新しいファイルで、探すときの入口になる |
| `docs/tech-debt/README.md` | 既存の 2 行を更新、新しい 5 行を追記(下の表) |
| `docs/plans/active/2026-10-08-org-limits-reserve.md`(151〜153 行目) | 進捗のチェックボックスだけを変えた。Review / Verification / Test artifact created にチェックを付けた(V-9)。「PR created」は `/pr` が付ける。本文は触っていない |
| `docs/insights/events/2026-10-08-org-limits-reserve.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 1) |
| `docs/reports/sync-docs-2026-10-08-org-limits-reserve.md` | この report |

### tech-debt の変更

| 行 | 変更 |
|----|------|
| 144 行目(config と state dir の解決の規則が違う、の行)。V-8 | Impact に、spawn と start が `max_orgs` と `max_total_seats` を main worktree の `ralph.toml` から読むようになったこと(`withMainWorktreeOrgLimits`)、`max_seats`・`model_pool`・`[org.roles]`・teardown の動詞は変わらないこと、main の `ralph.toml` が壊れていると linked worktree の側が正しくても spawn と start が止まること(`TestOrgStart_MainWorktreeRalphTomlLoadError`)、`--config` か `--state-dir` で避けられること、を足した。Trigger に `withMainWorktreeOrgLimits` と `MainWorktreeRoot` を足した。Related に verify の V-8 を足した。行のきっかけ(`resolveOrgConfig` か `ResolveOrgStateDir` の変更)は文字どおりには来ていない |
| 170 行目(S8 の補償の 4 件の行)。V-7 | (a) C2-1 に、`reserveAgain` が同じ ledger の写しを読む 3 つ目の読み手になったことを足した。(c) C2-5 の `verbs.go` の行数に「1,875 行(`reserveAgain` で増えた)」を足した。Trigger に `reserveAgain` を足し、「この PR は workspace の経路を変え `reserveAgain` を足したが C2-1 も C2-5 も払っていないので、きっかけは消えていない」と書いた。Related に verify の V-7 を足した |
| 172 行目(新) | 補償と上限(F-5、F-10 と plan の (a)、V-2、workspace の経路の窓)。(a) `reserveAgain` が読む前の写しで決める(同じ org_id を窓の間に `--reserve` で立て直すと、古い予約が新しい予約を上書きする)。(b) pane の経路は `reserveAgain` を呼ばず、disband で解けた予約を戻さない。(c) pane の経路が `max_total_seats` を 1 つ超えうる(`TestOrgCloseDeferredSelfPane_CloseFails_ReactivationCanExceedMaxTotalSeats` が固定している)。(d) workspace の経路の `max_orgs` +1 と予約の重なり(`TestOrgCloseDeferredSelfWorkspace_ReservationRestored_RiskWindowClearedByRetry`)。直し方と、固定しているテストを一緒に直すことを Trigger に書いた |
| 173 行目(新) | 予約の入力の静かな失敗。(a) 末尾に `/` がないとファイル(F-9)。(b) `*` はファイル名の文字(plan の (b))。(c) 読めない `scope_reserved` が repo 全体になり、本物の `.` と区別できない(F-8)。(d) パスを大文字小文字を区別して比べる(macOS の既定のファイルシステムでは `Internal/` と `internal/` が同じ場所) |
| 174 行目(新) | leader に見える情報。(a) `--reserve` だけの leader は `{{SCOPE}}` に `defaultScopeText` が入る(F-4、V-5、plan の (c))。(b) `EnvelopeSummary` と `checkOrgEnvelope` が `max_seats` だけを出す(V-4) |
| 175 行目(新) | コードの形。(a) `reserve` が `[]string` と `bool` の 2 つの意味、`scopeReservedEvent` の位置引数(F-2)。(b) `currentOrgLives` が `openOrgWorkspaces` の畳み込みを写している(F-6)。(c) `newOrgSpawnRuntime` が `newOrgRuntime` の 2 手順を写している(F-7) |
| 176 行目(新) | テストの穴。(a) X18(`reserveAgain` の「すでに予約を持つなら書かない」)。(b) 台帳の追記と読み込みの失敗の分岐(`checkCapacityAndStart` 69.2%、`idempotentRespawn` 87.5%、`reserveAgain` 75.0%、`orgReservation` 75.0%)。(c) 別プロセスの競合。(d) bare リポジトリの linked worktree を CLI から打つ場合。(e) 本物の herdr と agmsg |

新しい行はすべて、コードを関数名とテスト名で指し、`file:line` を使っていない(`[A-Za-z_./]*:\d+` で 144、170、172〜176 行目を調べて 0 件)。plan は `docs/plans/active/...` で指した(`/pr` の `archive-plan.sh` が書き換える)。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `README.md`(124 行目、247 行目) | verify が確認済みで、skill とコードに合う。124 行目は `--reserve` の規則(末尾 `/`、`.`、重なりの拒否、disband で解ける)、247 行目は 2 つの上限と拒否の文を書く。変更なし |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-3 に 1 項目を足した(上の表)。Current state の「上限は org ごとの `max_seats` だけ」(19 行目)は、2 段目が「`--org-id` が必須」を書き換えなかったのと同じく、この spec の出発点の記述なので変えていない。Rollout の「1. 共通の台帳(FR-1、進行中)」(203 行目)は 1 段目がマージされたあとも残っている。下の Found but left |
| `AGENTS.md` repo map | `internal/org/` の行を更新した(上の表)。ほかの行は、この差分が `internal/cli/` の help の文字列だけを変えたので該当なし |
| `docs/quality/quality-gates.md` と template の写し | 更新した(上の表)。Org runtime gates の ほかの行は、watchdog と品質ゲートの話で変更なし |
| `docs/specs/2026-08-01-org-runtime.md` の FR-2(`[org]` のエンベロープ設定) | `max_seats` までを挙げる 8 月の spec で、新しい 2 つのキーは multi-org spec の FR-3 が持つ。履歴の spec なので変えていない |
| `templates/base/ralph.toml`、`scripts/ralph-config.sh` と template の写し | verify が AC11 で確認済み(`defaults_sync_test.go` が揃えを固定)。この cycle は触っていない |
| `ralph org spawn --help`、`ralph org start --help`、`ralph org status --help` | 手元でビルドしたバイナリで出力を見た。spawn と start は同じ `orgWideLimitsHelp` を出し、動詞が両方書かれている。status は `Long` に `reserved:` の行と `reservation` が出る。start の `--scope` の help の崩れは 166 行目の (b) の既存の行で、この差分の範囲外 |
| `docs/recipes/`、`docs/architecture/`、`.claude/rules/`、`.ralph/`、`templates/base/docs/`(`git grep -n -i 'max_orgs\|max_total_seats\|--reserve'`) | 当たるのは README、quality-gates の 2 面だけ。recipes、architecture、rules は上限も予約も書いていない。変更なし |
| harness 内部の整合(skill、hook、rule、script、language pack) | skill は `/org` の 4 面を同じ内容に揃えた(`check-skill-sync.sh`、`check-sync.sh` が通る)。hook、rule、script、language pack はこの差分が触っていないので該当なし |

## Found but left

- spec の Rollout の「1. 共通の台帳(FR-1、進行中)」(203 行目)は、1 段目がマージされたあとも「進行中」のまま。2 段目の PR も直していない。どの時点で「進行中」を外す慣例かが spec に書かれていないので、触っていない。直すなら、1 段目と 2 段目の印を同時に整理するのが近い
- 依頼では V-5 を「`{{SCOPE}}` が空になる」と書いていたが、コードは空にせず、`prompts.go` の `defaultScopeText`(「未指定(読み取り中心で、リポジトリ規約に従うこと)」)に置き換える。skill と tech-debt には、実際の挙動(予約したパスではなく既定の「未指定」の文言が入る)で書いた
- verify の V-4(`{{ENVELOPE}}` と doctor の要約)は、174 行目の (b) に送った。出す・出さないの判断が要る
- self-review の F-3(autonomous の scope ゲートの文)と F-1(director の語)は `8dd19634` で直済み。F-2、F-6、F-7、F-8、F-9、F-10 と plan の (a)〜(c) は tech-debt に送った(172〜175 行目)。F-5 は 172 行目の (a)(と 170 行目の C2-1)に入れた。F-11 は help の文字列で直した
- test report の Test gaps 6(予約のパスの大文字小文字)は 173 行目の (d) に入れた。test の追加 2 本(`c3a95c48`)のうち、`..._ReactivationCanExceedMaxTotalSeats` は今の振る舞いの記録であることを 172 行目の (c) に書いた

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/run-static-verify.sh`(commit 1 の後、tech-debt の編集の前) | rc 0。`check-sync.sh` PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)、`check-pipeline-sync.sh` OK、`check-skill-sync.sh` OK(13 skill)、`check-template-purity.sh` PASS、tech-debt README plan references OK、gofmt ok、golangci-lint `0 issues.`、branch secret scan clean(`51855166..76d1cf1c`) |
| `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`(skill 編集の直後) | `[ok] check-skill-sync: 13 skill(s) in lock-step`、`PASS: all files in sync.`(DRIFTED 0) |
| `gofmt -l internal` | 出力なし |
| `go test ./internal/cli/... -count=1` | `ok  github.com/yoshpy-dev/ralph/internal/cli  79.399s` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-08-org-limits-reserve.md` | `1a165903b5df`(plan の `- Approved:` 行と一致。チェックボックスの変更の前後で同じ) |

tech-debt の編集の後に走らせた検査は、この report の commit の後の `./scripts/run-static-verify.sh` と `./scripts/secret-scan-branch.sh --strict` で、結果は呼び出し元への報告に書く。

## Not verified

- skill の V-1 の列挙(「`--config`、`--state-dir` か env、git の toplevel、git の外」)は、`ResolveOrgStateDir` の 5 つの source と `withMainWorktreeOrgLimits` を読んで書いた。bare リポジトリの linked worktree から CLI を打って確かめてはいない(`TestMainWorktreeRoot` の単体の場合だけ。tech-debt 176 行目の (d))。未確認です
- skill の V-2 の段落(補償は上限も予約の重なりも見ない)は、verify の V-2 と `verbs_test.go` の 2 本のテストの記述を読んで書いた。本物の herdr では起こしていない
- tech-debt の新しい行の「直し方」は、self-review と verify の提案を写したもので、実装して試してはいない
- 「上限と予約を書いた文書はほかにない」は、`max_orgs`、`max_total_seats`、`--reserve` の grep で当たった範囲の確認である。別の言い方で書かれた記述は拾えていない可能性がある
