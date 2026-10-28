# sync-docs report: codex-verified-auto-merge-on

- Date: 2026-10-29(JST。insight event の ts は 2026-10-28T16:52Z)
- Plan: `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md`(issue #274。承認の digest `85ba56ccca6a` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` に足した 1 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `ae64da70`、branch は `feat/codex-verified-auto-merge-on`。run 開始時の HEAD は `495c5982`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-28-codex-verified-auto-merge-on.md`(b49fd6c2 が 1 回目、修正スライス F1 のあとの再 run は 7ca29c41、Merge yes、所見 0)、`docs/reports/verify-2026-10-28-codex-verified-auto-merge-on.md`(e802b228、pass、LOW D-1・D-2、INFO D-3〜D-5)、`docs/reports/test-2026-10-28-codex-verified-auto-merge-on.md`(9d78a1c3、pass)
- 差分の大きさ(`git diff ae64da70...HEAD --shortstat`、この run の前): 40 ファイル、+1115/-317。walkthrough を書くかどうかは `/pr` が決める

## Summary

この branch は `[org.permissions].codex_verified` と `[org.director].auto_merge` の既定を false から true に変えた(`config.Default()` と `templates/base/ralph.toml`)。false を書いた `ralph.toml` の動きは変えていない。skill 4 つの写し、recipe 2 本とその写し、`README.md`、spec、Go のコメントは、S4 までに新しい既定へ直してある。`ralph status` は `ralph.toml` が読めないとき auto_merge を false と出す(F1)。

verify が残した LOW の D-1 と D-2 を直した。どちらも `docs/tech-debt/README.md:136` の 1 行(doctor の shell alias の行)で、true が人の書く opt-in だった頃の言い方が残っていた。

そのほかの面を洗った結果、直す文書はほかに見つからなかった。「既定では guarded だけ」「director は既定で merge しない」と書いた文書は、履歴の印がついた spec の文を除いて残っていない。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(136 行、Debt item の欄) | doctor の Detail の文の引用 `("… once codex_verified = true … a guarded seat runs with the alias's value …")` を、`internal/cli/doctor_shell_alias.go:714-719` の現在の文に合わせて `("… only while [org.permissions].codex_verified is true, the default … a guarded seat silently runs with the alias's sandbox …")` に差し替えた。引用の 2 つの断片は、コードの文字列から一字違わず取った(D-1) |
| `docs/tech-debt/README.md`(136 行、Impact の欄) | 書き出しの「An operator who has already set `codex_verified = true` and assigned a `guarded` role …」を「An operator whose `[org.permissions].codex_verified` is `true` (the default since #274, whether written or left out) and who assigned a `guarded` role …」に直した。この行のほかの語は変えていない。表は別のセッションの PR と共有なので、2 か所の置換だけにした(D-2) |
| `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md` | `## Progress checklist` に「Sync-docs report created」の 1 行を足した。ほかの節は触れていない(digest は同じ) |
| `docs/reports/sync-docs-2026-10-28-codex-verified-auto-merge-on.md` | この report |
| `docs/insights/events/2026-10-28-codex-verified-auto-merge-on.jsonl` | sync_docs の event を 1 行追記(下の Verification) |

D-3(`ralph status --help` に、読めないときの false を足すか)と D-4(`internal/org/permissions.go` のエラー文の「until then … first」)は、依頼どおり直していない。どちらも PR の Notes に回る。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `docs/tech-debt/README.md:136` | 直した | 上の D-1、D-2。Debt item の「nor whether `[org.permissions].codex_verified` is really `true` on this project」は、false を書いた project があるので今も正しい |
| `docs/tech-debt/README.md:144`(b) | 変更なし(同期済み) | 「falls back to the built-in defaults (`codex_verified = true` since #274)」に S4 で直っている |
| `docs/tech-debt/README.md:241-249`、`:260`、`:268`、`:270` | 変更なし | どれも `auto_merge = true` や `codex_verified` を条件にした文、またはテンプレートの `default = "autonomous"` と codex の director が guarded に限られること(この PR で変わらない)の話 |
| `README.md:124`、`:125`、`:258`、`:266`、`:268` | 変更なし(同期済み) | `:124` は「(default true)」。`:125`、`:258`、`:266`、`:268` は `auto_merge` の true と false の両方を条件つきで説明し、既定が false だとは言っていない |
| `README.md:226`、`AGENTS.md:39`、`.ralph/core/AGENTS.core.md:28` の「CI + human merge」 | 変更なし | 開発 harness の標準フローの 10 段目で、人が PR を merge する話。`[org.director].auto_merge` は org runtime の director だけを決めるので、関係しない |
| `AGENTS.md` の Repo map、`CLAUDE.md`、`.claude/rules/ralph/`、`docs/quality/`、`docs/architecture/`、`.ralph/core/`、role prompt(`internal/org/prompts/`)、`docs/specs/2026-08-01-org-runtime.md` | 変更なし | 2 つの設定の名前を含む行が 0 件(`git grep` で確かめた)。「ralph.toml parser with defaults」のような値を言わない文だけ |
| `/org` skill(`.claude/skills/org/SKILL.md:131-134`、`:1541-1546` など、写し 3 つ) | 変更なし(同期済み) | 「`codex_verified = true`(既定)」「false では `guarded` 以外を拒否」。`check-skill-sync.sh` は 14 skill が一致 |
| `/director` skill(`.claude/skills/director/SKILL.md:813-826`、写し 3 つ) | 変更なし(同期済み) | 「`auto_merge = true`(既定)」。「2 つの値が違えば false」の規則(D4)は変えていない |
| `docs/recipes/codex-seat-permissions.md`、`docs/recipes/codex-setup.md` | 変更なし(同期済み) | 既定 true、確かめて通らなければ false。ルートとテンプレートの写しは `cmp` で byte 一致 |
| `docs/specs/2026-10-07-org-multi-org-director.md`(`:129`、`:132`、`:193`、`:310`) | 変更なし(同期済み) | 旧文は残し、2026-10-28 の印と「2026-10-28 に決めたこと」の注で新しい既定を言う。計画の AC3 が除く履歴の文 |
| `templates/base/ralph.toml` のコメント | 変更なし(同期済み) | 「true (default)」「An absent codex_verified loads as true」ほか。`TestDefaultsLockStep` の確認は `/test` が済ませた |
| `internal/cli/org.go:206-228`(`withMainWorktreeDirectorEnabled` の doc comment) | INFO D-6 | 「`ralph status` warns about before it shows the built-in defaults (statusDirectorConfig)」は、F1 のあと `AutoMerge` だけ false を出すので、`AutoMerge` についてはやや不正確。`statusDirectorConfig` 自身の doc comment は正確で、動きも `ralph status` の warning も正しい。Go のコメントなので、依頼どおり直していない |
| `internal/cli/status.go:66-71`(`ralph status --help`) | INFO D-3(未修正) | 「read by the same rule as enabled」で、読めないときの false は warning が言う |
| `internal/org/permissions.go:147`、`:152`(実行時のエラー文) | INFO D-4(未修正) | false を明示した project だけが見る条件つきの文で、偽ではない |

## Plan drift notes

- Deviation M1(`ralph status` が読めない `ralph.toml` で auto_merge を false と出す)は、計画の Progress に F1 の記録がある。計画の本文(digest の対象)には書いていないが、手を入れていない。
- 計画の Acceptance criteria は AC4 だけが `[ ]` で、PR の本文が要るため `/pr` の分。AC1〜AC3、AC5〜AC7 の `[x]` は verify と test の report が根拠になっている(verify の D-5 は、AC のチェックボックスが `[ ]` のままという指摘だった。現在は `[x]`)。Progress の残る `[ ]` は「PR created」だけ。

## Release notes

PR の本文の release ノートの項(計画の AC4)に使う文。下の枠が本文。

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
```

自動の changelog(`.goreleaser.yml` は `^docs:`、`^test:`、`^chore:` を除く)に載るのは、この branch の次の 2 件。

- `cb8d258e feat: default codex_verified and auto_merge to true`
- `958d9bff fix: show auto_merge as false when ralph status cannot read ralph.toml`

ほかの 16 件(`docs:` が 14 件、`test:` が 2 件)は載らない。skill、recipe、`README.md`、spec の更新(`f99ac82d`)と Go のコメントの更新(`920a4dcb`)は `docs:` なので、changelog からは読み取れない。`feat:` の題は既存の project への影響を言わないので、上の release ノートを手で足す。squash merge にすると、載るのは PR の題 1 件になる。

## `/pr` に渡すこと

- 上の release ノートを PR の本文に置く(AC4)。
- PR の Notes に書く未修正の INFO: D-3(`ralph status --help` に読めないときの false がない)、D-4(`internal/org/permissions.go` のエラー文の「until then … first」)、D-6(`internal/cli/org.go` の doc comment の「built-in defaults」)。いずれも文言で、動作は正しい。
- この repo にはルートの `ralph.toml` がないので、この PR のあと、この repo の `ralph org` も auto_merge true、codex の座席は autonomous の既定で動く(計画の Assumptions が受け入れた動き)。
- `docs/tech-debt/README.md` は別のセッションの PR と共有している。merge の衝突は両方の変更を残して解く。この run の変更は 136 行の 2 か所だけ。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-template-purity.sh` | rc 0。`PASS: no meta-repo-specific references found in templates.` |
| `git diff --check` | 出力なし |
| `cmp`(recipe 2 本のルートとテンプレート) | byte 一致 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md` | `85ba56ccca6a`(`- Approved:` 行の値と同じ) |
| 引用の一致 | 差し替えた 2 つの断片(`only while [org.permissions].codex_verified is true, the default`、`a guarded seat silently runs with the alias's sandbox`)は `internal/cli/doctor_shell_alias.go:714-719` の文字列と同じ |
| テスト、`go vet`、golangci-lint、`run-verify.sh` ほか | 打っていない(依頼で止められた)。この run が変えたのは tech-debt の 1 行の 2 か所、計画の Progress、この report、insight event で、コードの動きは変わらない |
| insight event | `./scripts/insights-append.sh --slug codex-verified-auto-merge-on --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で追記する(`docs/insights/events/2026-10-28-codex-verified-auto-merge-on.jsonl`) |
