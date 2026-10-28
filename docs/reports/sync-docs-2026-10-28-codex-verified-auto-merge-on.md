# sync-docs report: codex-verified-auto-merge-on

- Date: 2026-10-29(JST)
- Plan: `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md`(issue #274。承認の digest `85ba56ccca6a` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` に足した 1 行だけ)
- Pipeline cycle: 2(`.harness/state/standard-pipeline/cycle-count.json` の値)。この report は cycle 1(e3363a99)の内容を、cycle 2 の結果で置き換えた。base は `ae64da70`、branch は `feat/codex-verified-auto-merge-on`。run 開始時の HEAD は `de11b865`(この report を含む commit の SHA ではない)。cycle 1 のあとに origin/main の #282(init-pack-choice、`09dc0260`)を `aa4ec8d3` で merge している
- 先行 report(cycle 2): `docs/reports/self-review-2026-10-28-codex-verified-auto-merge-on.md`(run 3、7dbd0858、Merge yes、所見 0)、`docs/reports/verify-2026-10-28-codex-verified-auto-merge-on.md`(90f5c80a、pass、LOW D-9、INFO D-3・D-4・D-7・D-8)、`docs/reports/test-2026-10-28-codex-verified-auto-merge-on.md`(de11b865、pass)、`docs/reports/cross-review-triage-codex-verified-auto-merge-on.md`(fe22fbf2、cycle 1、ACTION_REQUIRED 1 件を F2 aea6b494 が閉じた)
- 差分の大きさ(この run の前): この PR の分は `git diff 09dc0260...HEAD --shortstat` で 42 ファイル、+1482/-343。`git diff ae64da70...HEAD` は 55 ファイル、+3165/-383 になるが、#282 の分(`ae64da70..09dc0260`)を含む。walkthrough を書くかどうかは `/pr` が決める

## Summary

この branch は `[org.permissions].codex_verified` と `[org.director].auto_merge` の既定を false から true に変えた(`config.Default()` と `templates/base/ralph.toml`)。false を書いた `ralph.toml` の動きは変えていない。cycle 1 のあと、cross-review の ACTION_REQUIRED を受けて F2(aea6b494)が入った。`resolveOrgConfig` は、`--config` がなければ `ralph.toml` を `config.Load` に渡し、`fs.ErrNotExist` のときだけ既定にする。`./ralph.toml` があるのに読めない(権限のないディレクトリへの symlink など)ときは、前の版のように既定へ倒さず、エラーを返す。

verify の LOW D-9 を直した。cycle 1 の release ノートは F2 の前に書いたので、(1) F2 の動きの項がなく、(2) changelog に載る commit を 2 件と数え、(3) 直った D-6 を PR の Notes に回す INFO に挙げていた。この report の「Release notes」と「`/pr` に渡すこと」を直してある。

F2 で変わった動きに食い違う文書を洗い直したが、直す文書は見つからなかった。ただ 1 か所、`docs/tech-debt/README.md:148` の行の Trigger に、F2 が `resolveOrgConfig` を変えたことの跡がなかった。同じ行の先例にならって 1 文を足した(下の Changes made)。

## Changes made

| File | Change |
|------|--------|
| `docs/reports/sync-docs-2026-10-28-codex-verified-auto-merge-on.md` | この report。cycle 2 の結果で書き直し、D-9 の 3 点を直した |
| `docs/tech-debt/README.md`(148 行、Trigger の欄と Related の欄) | 行の Trigger が挙げる「any change to `resolveOrgConfig`」に F2 が当たったので、Trigger の末尾に、同じ欄にある `withMainWorktreeOrgLimits` の先例(feat/org-feature-worktree)と同じ形で 1 文を足した。「`resolveOrgConfig` changed in feat/codex-verified-auto-merge-on (F2, commit aea6b494: a `./ralph.toml` that exists but cannot be read … is now an error instead of the built-in default, so it blocks the teardown verbs the way a validation error does), which met that part of this trigger too and left the open part as it was」。Related に計画の `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md`(Progress, F2)を足した。行は 5 欄のまま、`~~` は 0 個。行のほかの語は変えていない |
| `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md` | `## Progress checklist` の「Sync-docs report created」の下に、cycle 2 の 1 行を足した。ほかの節は触れていない(digest は同じ) |
| `docs/insights/events/2026-10-28-codex-verified-auto-merge-on.jsonl` | sync_docs の event を cycle 2 として 1 行追記(下の Verification) |

cycle 1(e3363a99)の変更は、`docs/tech-debt/README.md:136`(D-1、D-2)と計画の Progress の 1 行、report、insight event の 1 行で、cycle 2 では触っていない。

D-3(`ralph status --help` に、読めないときの false を足すか)、D-4(`internal/org/permissions.go` のエラー文の「until then … first」)、D-7、D-8 は、依頼どおり直していない。PR の Notes に回る。コードは触っていない。

## Surfaces checked for drift

cycle 1 の e3363a99 のあとに変わった文書は `README.md` と `templates/base/docs/recipes/adding-a-language-pack.md` だけで、どちらも #282 の `ralph init` の分(`README.md` は `ralph init` の行 1 本)。2 つの設定を言う文書は変わっていない。F2 の変更点(`os.Stat` の分岐を消し、読めない `./ralph.toml` をエラーにする)について、「ない」と「読めない」の扱いを書いた箇所を洗い直した。

| Surface | 結果 | 根拠 |
|---------|------|------|
| `internal/cli/org.go:604-612`(`resolveOrgConfig` の doc comment) | 変更なし(F2 で直っている) | 「ない」は `fs.ErrNotExist` だけ、読めない file はエラーで既定にしない、と書く |
| `--config` の help(`internal/cli/org.go:58`)、`orgWideLimitsHelp`(`:699`) | 変更なし(同期済み) | 「default: ./ralph.toml if present, else built-in defaults」「built-in defaults when it has none」。ない場合だけ既定と読め、読めない file を既定にするとは書いていない |
| `internal/cli/org.go:218-224`(`withMainWorktreeDirectorEnabled` の doc comment、D-6) | 変更なし(a91f6694 で直っている) | 「built-in Enabled with AutoMerge false」。`status.go:299` と合う |
| `internal/cli/status.go:249-261`、`org_inbox.go:62-76`、`org_merge.go:157-163` の help と doc comment | 変更なし(同期済み) | どれも「ないときは既定、読めない(does not load)ときはエラー(`ralph status` は warning)」の形 |
| `/org` skill(`.claude/skills/org/SKILL.md:190`、`:272`、`:405-413`、写し 3 つ) | 変更なし(同期済み) | `:190` は「`ralph org` の動詞はどれも設定(`--config`、省略時は cwd の `./ralph.toml`)を読む」、`:272` は「`--config` のファイル(なければ打った場所の `./ralph.toml`)」、`:406-412` は「`ralph.toml` を読めないときは終了コード 1」。F2 の前は、権限のないディレクトリへの symlink の `./ralph.toml` がこの文に反して既定になっていた |
| `/director` skill(`.claude/skills/director/SKILL.md:252-257`、`:820`、写し 3 つ) | 変更なし(同期済み) | 「読めない `ralph.toml` はエラーになる」。`:820` の「2 つの値が違えば false」(D4)も変わらない |
| `README.md`、`docs/recipes/`、`docs/specs/`、`docs/quality/`、`docs/architecture/`、`AGENTS.md`、`CLAUDE.md`、`.claude/rules/` | 変更なし | `ralph.toml` が読めないときに既定を使う、と書いた文はない。「if present」「built-in default」「unreadable」「cannot be read」「does not load」「読めない」「ない」で `git grep` した。当たりは skill の上の 2 本と spec の履歴の文だけ |
| `docs/tech-debt/README.md:148` | Trigger に跡を足した | 本文の「Config: `--config`, else only the cwd's `./ralph.toml`, else the built-in default」は、ないときの話で今も正しい。「every verb, the teardown verbs `status`/`stop`/`disband` included, goes through `newOrgRuntime` → `resolveOrgConfig`, so any `ralph.toml` validation error … blocks stopping seats」と Impact の「A broken `ralph.toml` leaves running seats reachable only after the file is fixed or a fixed copy is passed with `--config`」は、F2 のあと読めない file にも当てはまり、偽にはならない。Trigger は「any change to `resolveOrgConfig`」を挙げ、F2 が当たったので、跡を足した(上の Changes made)。行の開いた部分(設定の探索が main worktree と git の toplevel を無視すること)は F2 で変わらない |
| `docs/tech-debt/README.md:136`、`:144`、`:241-249`、`:260`、`:268`、`:270` | 変更なし(同期済み) | cycle 1 のとおり。`:136` は e3363a99 で直し、verify cycle 2 が確かめた |
| `internal/cli/status.go:66-71`(`ralph status --help`) | INFO D-3(未修正) | 「read by the same rule as enabled」で、読めないときの false は warning が言う |
| `internal/org/permissions.go:147`、`:152`(実行時のエラー文) | INFO D-4(未修正) | false を明示した project だけが見る条件つきの文で、偽ではない |
| `--config` で名指した file がないとき(`internal/config/config.go:640-645`、`TestLoad_MissingFile`) | INFO D-7(未修正) | `config.Load` の契約で、F2 の前からある。`--config` の path の打ち間違いは、既定が true になったので `ralph.toml` のない project と同じ true で動く |
| `ralph doctor`(`internal/cli/doctor.go:93-101`) | INFO D-8(未修正) | 読めない `ralph.toml` を「parse error … using defaults」と呼び、ほかの検査を既定で判定する。既定が true になったので、要らない warn が出うる。何も動かさない |

## Plan drift notes

- Deviation M1(`ralph status` が読めない `ralph.toml` で auto_merge を false と出す)と F2 は、計画の Progress に記録がある。計画の本文(digest の対象)には書いていないので、手を入れていない。
- 計画の Acceptance criteria は AC4 だけが `[ ]` で、PR の本文が要るため `/pr` の分。AC1〜AC3、AC5〜AC7 は `[x]`。Progress の残る `[ ]` は「PR created」だけ。

## Release notes

PR の本文の release ノートの項(計画の AC4)に使う文。下の枠が本文。cycle 1 の枠に、F2 の動きの項を「そのほか」に足した。

```
### [org.permissions].codex_verified と [org.director].auto_merge の既定を true にした (#274)

変わること
- ralph.toml にキーを書いていないとき、ralph.toml がないときの値が、どちらも true になった。ralph init が書く ralph.toml も、2 行とも true になる。
- codex_verified = true では、codex の座席が autonomous と edits でも起動する(false では guarded だけ)。
- auto_merge = true では、director が merge-order の ready の PR を、聞かずに ralph org merge で merge する。ralph org merge 自身は、値によらず PR、checks、既定のブランチの保護を確かめてから merge する(変わらない)。

既存の project
- ralph.toml は seed で、ralph upgrade は書き換えない(テンプレートの変化を advisory の diff として出すだけ)。false が書いてある行は変わらない。
- ralph init は codex_verified = false を v4.0.0 から、auto_merge = false を v6.0.0 から書いている。v4.0.0〜v5.x の ralph init で作った ralph.toml には auto_merge の行がないので、この版に更新すると、その project の auto_merge は true になる。
- 新しい既定にしたい project は、2 行を手で true にする。

ralph doctor
- テンプレートの既定のまま(driver_pool に codex、[org.permissions].default = "autonomous")の project では、agmsg の DB が codex の writable root にないと、「Codex sandbox (agmsg writable root)」が warn するようになる。warn は doctor の終了コードを変えない(終了コードに効くのは fail だけ)。対処は docs/recipes/codex-seat-permissions.md。

元に戻すには
1. ralph.toml の codex_verified と auto_merge を false にする。director は merge の前に毎回 auto_merge を読むので、次の merge から効く。
2. 動いている codex の座席は、spawn のときに flag が決まっている。ralph org stop で止めて spawn し直す(false では codex の座席は guarded の役でしか spawn できない)。
3. ralph org director status の auto_merge: の行と、ralph org report に出る座席の permission_mode で、効いている値を確かめる。

そのほか
- ralph status は、ralph.toml が読めないとき、director の行の auto_merge を false と出す(既定の true ではない)。stderr の warning が理由を言う。
- --config なしの ralph org の動詞と ralph status は、./ralph.toml があるのに読めない(権限のないディレクトリへの symlink、輪になった symlink など)とき、既定で動かずにエラーにする(ralph status は warning を出し、auto_merge を false と出す)。前の版は既定で動いていた。ralph.toml がないときは今までどおり既定。
```

自動の changelog(`.goreleaser.yml` は `^docs:`、`^test:`、`^chore:` を除く)に載るこの PR の commit は、次の 3 件。`git log --no-merges 09dc0260..HEAD`(この PR の分。#282 の merge `09dc0260` より後)で数えた。

- `cb8d258e feat: default codex_verified and auto_merge to true`
- `958d9bff fix: show auto_merge as false when ralph status cannot read ralph.toml`
- `aea6b494 fix: return an error for an unreadable ralph.toml instead of the defaults`

ほかの 25 件(`docs:` が 23 件、`test:` が 2 件)は載らない。run 開始時の HEAD `de11b865` までの 28 件で、この run の `docs:` の commit が 1 件加わる。skill、recipe、`README.md`、spec の更新(`f99ac82d`)と Go のコメントの更新(`920a4dcb`、`a91f6694`)は `docs:` なので、changelog からは読み取れない。`feat:` の題は既存の project への影響を言わないので、上の release ノートを手で足す。squash merge にすると、載るのは PR の題 1 件になる。

`ae64da70..HEAD` で数えると 55 件(`docs:` 44、`test:` 3、`feat:` 3、`fix:` 4、`refactor:` 1)で、`feat:` `fix:` `refactor:` の 8 件が載る。差は #282(init-pack-choice)の 5 件で、同じ release の changelog には #282 の分も載る。#282 の 5 件は次のとおり。

- `baa66655 feat: add --pack to ralph init and install no pack with --yes alone`
- `d1c756de feat: ask for language packs with skip and Recommended in ralph init`
- `f495bb23 fix: tighten the init help and pack listing from the self-review`
- `80a2eb28 fix: clarify the --yes help sentence and the AvailablePacks comment`
- `dbc5afaf refactor: build the init config once before either init path`

## `/pr` に渡すこと

- 上の release ノートを PR の本文に置く(AC4)。
- PR の Notes に書く未修正の INFO は 4 件。D-3(`ralph status --help` に、読めないときの false がない)、D-4(`internal/org/permissions.go` のエラー文の「until then … first」)、D-7(`--config` で名指した file がないときも既定、つまり true で動く。`config.Load` の契約で、この PR の前からある)、D-8(`ralph doctor` は読めない `ralph.toml` を既定で判定する)。いずれも文言か、この PR の前からの契約で、動作はこの PR の意図どおり。D-6(`internal/cli/org.go` の doc comment)は a91f6694 で直ったので、Notes から外した。
- この repo にはルートの `ralph.toml` がないので、この PR のあと、この repo の `ralph org` も auto_merge true、codex の座席は autonomous の既定で動く(計画の Assumptions が受け入れた動き)。
- `docs/tech-debt/README.md` は別のセッションの PR と共有している。merge の衝突は両方の変更を残して解く。この PR の変更は 136 行の 2 か所と、148 行の Trigger と Related の 2 か所。
- 148 行の Related に計画の `docs/plans/active/` の path を書いた。`/pr` の `archive-plan.sh` が `archive/` に書き換える。`run-static-verify.sh` は tech-debt の plan 参照の存在を見る(verify cycle 2 で OK)。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-template-purity.sh` | rc 0。`PASS: no meta-repo-specific references found in templates.` |
| `git diff --check` | 出力なし |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md` | `85ba56ccca6a`(`- Approved:` 行の値と同じ。Progress の編集の前後で同じ) |
| `docs/tech-debt/README.md:148` | 5 欄のまま(`(?<!\\) \| ` で分けて確かめた)、`~~` は 0 個。Related の計画の path は存在する |
| commit の数 | `git log --no-merges 09dc0260..HEAD`: 28 件(`docs:` 23、`test:` 2、`feat:` 1、`fix:` 2)。`ae64da70..HEAD`: 55 件(`docs:` 44、`test:` 3、`feat:` 3、`fix:` 4、`refactor:` 1) |
| テスト、`go vet`、golangci-lint、`run-verify.sh` ほか | 打っていない(依頼で止められた)。この run が変えたのは tech-debt の 1 行、計画の Progress、この report、insight event で、コードの動きは変わらない |
| insight event | `./scripts/insights-append.sh --slug codex-verified-auto-merge-on --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で追記する(`docs/insights/events/2026-10-28-codex-verified-auto-merge-on.jsonl`。`--cycle auto` は cycle-count.json の 2 を読む) |
