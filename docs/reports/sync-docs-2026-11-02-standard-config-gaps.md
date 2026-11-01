# sync-docs report: standard-config-gaps

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-standard-config-gaps.md`(PR #291 の Known gaps と tech-debt 406・409・410 行)
- Branch: `fix/standard-config-gaps`。run 開始時の HEAD は `1232f0d1`、base は origin/main `d217c575`
- Pipeline cycle: 1
- 先行 report: self-review(Run 1 と Run 2。Merge: yes、LOW 11)、verify(pass、LOW 2 と INFO 4)、test(pass、`run-test.sh` rc 0 が 2 回、変異 m1 は Run 2 で落ちた)
- 制約: Go のファイル、skill、README、`internal/config/config.go` は変えていない。plan は AC6 の箱だけを `[x]` にした。digest は前後とも `11ccf2ed5a4c`

## Summary

tech-debt の 406・409・410 行を全部払った行として書き換えた。verify の V2 が挙げた 5 行(148、187(d)、198(e)、208(b)、220(b))は、古くなった名前を直し、Trigger に当たったが払わなかった旨を日付つきで足した。表の末尾に新しい行(421 行目)を足した。

コード、help、skill の文は変えていないので、pipeline の再実行は要らない。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 406・409・410 行の書き換え、148・187・198・208・220 行の注、末尾の新しい行。下の表のとおり |
| `docs/plans/active/2026-11-02-standard-config-gaps.md` | AC6 の箱を `[x]`。ほかの文は変えていない |
| `docs/reports/sync-docs-2026-11-02-standard-config-gaps.md` | この report |
| `docs/insights/events/2026-11-01-standard-config-gaps.jsonl` | `sync_docs` の行(cycle 1、pass、findings 0)。`insights-append.sh --cycle auto` で足した。UTC の日付がまだ 2026-11-01 なので既存のファイルに入った |

## 3 行の扱い

| 行 | 扱い |
|----|------|
| 406(`org_feature_test.go` の 819 行) | 払った。S1(`b71354d0`)で 3 本を `org_feature_reviewer_test.go` へ移し、819 から 647 行(移した先は 182 行)。行の 2 文目以降にある `internal/org/feature_test.go` と `internal/config/config_test.go` は前から 800 行を超えていたもので、計画の Non-goals に当たる。この 2 つは取り消し線の外に残し、「まだ 800 行を超えている」と書いた。`local_test.go` は fix/org-ledger-gaps で払ったと足した |
| 409(main の `ralph.toml` を 2 回読む) | 払った。S2(`7b9dc257`)。`resolveOrgStartConfig` が `resolveConfig` と `loadMainWorktreeConfig` で 1 回ずつ読み、`applyMainWorktreeOrgLimits` で上限と承認方式を当て、`newOrgRuntimeFromConfig` で runtime を作り、`[reviewer]` も同じ読み込みから取る。`InboxHintStateDir` は `runOrgStartPlan` が設定する。読み込みの回数は `TestOrgStartPlan_ReadsEachConfigFileOnce`、上限は S4(`bd41041a`)の `TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml`。cwd が main の root なら同じファイルを 2 回読む(役が違うので許す、AC2)と書いた |
| 410(3 つの test gap) | 全部払った。(a) は前から払ってあった。(b) は S2 で、`TestOrgStartPlan_UnreadableConfigStartsNothing`(main の `ralph.toml` と `--config` の 2 ケース)。(c) は S3(`2d4baa2d`)で、zsh のときだけ `setopt sh_word_split`、`sh`・`bash`・`dash`・`zsh -f` の 4 つとも 272/272。(b) と (c) を Debt・Impact・Trigger のそれぞれで取り消し線にし、Debt の末尾に「全部閉じた(RESOLVED 2026-11-02)」と書いた |

409 と 410 行は、取り消した文の中の `resolveOrgStartReviewer` をそのままにして(当時の記録なので書き換えない)、直した中身の文で `resolveOrgStartConfig` に置き換わったと書いた。Related に計画(`docs/plans/active/...`)を足し、410 行には test の報告も足した。

## verify V2 の 5 行

どの行も Trigger の末尾に `2026-11-02: fix/standard-config-gaps hit this trigger with ... but did not pay it: ... Carried over to the next change that meets this trigger` の形で足した(近くの行の 2026-11-01 の注と同じ形。依頼の短い形より長いが、行の中で揃えた)。Related に計画を足した。

| 行 | 古い名前の直し | Trigger の注 |
|----|----------------|--------------|
| 148(config と state dir の探索) | 2026-10-30 の注にある `resolveOrgStartReviewer` に、「since fix/standard-config-gaps, `resolveOrgStartConfig`」を括弧で足した | `withMainWorktreeOrgLimits`(読み込みと `applyMainWorktreeOrgLimits` の呼び出しに分かれた)と `resolveConfig`(新しい `loadConfigFile` を通る)に当たった。開いている部分は設定の探索で、触っていない |
| 187(d)(org-limits-reserve の test gap) | 名前は古くない | `withMainWorktreeOrgLimits` に当たった。bare repository の linked worktree の CLI テストは足していない。`MainWorktreeRoot` は変えていない |
| 198(e) T-5(org-feature-worktree の test gap) | 「`newOrgSpawnRuntimeAt` と `resolveModelOrWarn` の失敗の 2 つが打たれない」のうち、1 つ目を取り消し線にして閉じたと書いた。start --plan は `newOrgSpawnRuntimeAt` を通らず、読めない設定は `resolveOrgStartConfig` の失敗の枝になり、`TestOrgStartPlan_UnreadableConfigStartsNothing` が打つ。`resolveModelOrWarn` の枝は残した | `runOrgStartPlan` に当たった。1 つ目は払ったが、2 つ目は払っていない(枝は変えておらず、計画の Scope は読み込みの回数と文) |
| 208(b)(org-cross-watch の小さな残り) | 「`newOrgSpawnRuntimeAt`(spawn、`start <task>`、`start --plan`)」から `start --plan` を外し、`newOrgRuntimeFromConfig` で runtime を作ると書いた | `newOrgRuntimeAt` の本体に当たった(読み込みのあと `newOrgRuntimeFromConfig` を呼ぶ形)。doc は変わっておらず、まだ 2 つの呼び出し元だけを名指す。言い換えは Scope の外 |
| 220(b)(org-plan-approval の小さな残り) | 名前は古くない | `withMainWorktreeOrgLimits` に当たった。共有の規則は同じく limits だけの名前の `applyMainWorktreeOrgLimits` になった。改名は払っていない(読む回数を直す変更には要らないコード変更) |

198 と 208 と 220 の Related にも計画を足した。198 の Related は `(Progress checklist, S2)`、ほかは `(Design decisions, D1)` と書いた(計画に carry-over の節は足していない)。

## 新しい行(421 行目)

「Small leftovers of standard-config-gaps」。(a) verify V1: main の root から打つと、壊れた同じ `ralph.toml` を cwd の読み込みが先に読み、文が `org: load config: …` になってパスも値も名指さない。base も同じで、spawn と `start <task>` も同じ形。テストは subdirectory のケースだけ。(b) verify I1: `--config` の壊れたファイルの文にパスがない(`config.Load` が TOML のエラーにパスを付けない)。(c) self-review R2-L2 の (c): テストで main に `ralph.toml` を commit する 2 行の重複。(d) test の Test gaps: 前からある error の return 4 つ、上限のテストの対照が `max_orgs` だけ、外の台帳で承認方式を main から取らないことを直接見るケースがない。(e) PR #291 の Known gaps の残り: leader の通しは打っていない、live check の codex の model は既定と同じ値。

Impact は LOW。Why deferred と Trigger は項ごとに書いた。(a)(b) の Trigger は「`resolveConfig`・`loadConfigFile`・`config.Load` のエラーの次の変更、またはパスのない config error の報告」。関数名は書き、行番号は書いていない。

(d) の 4 つの return は、コードを読んで名前を付けた: `newOrgSpawnRuntimeAt` が `newOrgRuntimeAt` のエラーを返す枝、`newOrgRuntimeAt` の `org: load config:` の枝、`runOrgStartPlan` の `resolveOrgLedger` の枝、`runOrgStartPlan` の `resolveModelOrWarn` の枝。最後の 1 つは 198 行の (e) が追っているので、新しい行の Trigger は残りの 3 つだけを数えた。

## ほかの文書の食い違い

grep で、この diff の振る舞い(start --plan のエラーの文、読み込みの回数、`resolveOrgStartReviewer` の消滅)と食い違う文書を探した。見つけたものはない。

| 対象 | 結果 |
|------|------|
| `.claude/rules/ralph/model-routing.md` の start --plan の段落(と `templates/base/` の写し) | 「start --plan が読んだ値を leader のタスクの 2 行に書く」は今も正しい。読む回数は書いていない |
| `ralph org start --help`(HEAD から build して読んだ) | 承認方式と `[reviewer]` の出どころの文は「max_orgs・max_total_seats と同じ ralph.toml」のままで、1 回の読み込みで文字どおりになった。`--config` の文も合う |
| `docs/specs/`(`2026-10-07-org-multi-org-director.md` の 7 段目の決定) | 承認方式の読み方は「全 org の上限と同じ規則」で今も合う。変えた関数もエラーの文も名指していない |
| README(L131・L150)、`.claude/skills/org/SKILL.md`、AGENTS.md の repo map | `resolveOrgStartReviewer`、旧エラーの文、読み込みの回数を名指す所はない |
| `tests/test-ralph-config.sh` と zsh | `zsh -f` で動かない旨を書いた文書は tech-debt の 410 行だけで、直した |

README、`/org` skill、`internal/config/config.go` は別のセッションの #269 が変えている最中なので、触っていない。skill を変える必要も見つからなかった。

## 見つけた食い違い(lead へ)

- test の報告(`docs/reports/test-2026-11-02-standard-config-gaps.md`)の Coverage の Notes と Test gaps 2 は、打たれなかった 4 つの error の return を「`org.go:163`、`:381`、`:1207`、`:1225`(`org.FeatureRepoRoot` の失敗)」と書く。HEAD の `internal/cli/org.go` で数えると、1225 行目は `resolveModelOrWarn` の失敗の return で、`FeatureRepoRoot` の return は 1221 行目にあり、打たれなかった一覧には入っていない。tech-debt の新しい行は、行番号でなくコードを読んだ名前で書いたので、この誤りは持ち込んでいない。報告の書き直しは tester の記録の修正なので、ここでは触っていない。直すなら「パイプラインの記録だけの修正」の例外に当たりうる(verdict の行は変わらない)。この例外は cross-review の段で使うもので、それより前に直すなら test の報告を tester に直させる

## 確かめた結果

- `./scripts/run-static-verify.sh`: rc 0(`All verifiers passed.`)。`tests/test-ralph-config.sh` が未分類なので full fallback で golang を選んだ。gofmt `ok`、`0 issues.`、check-sync(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本一致)、check-template-purity、tech-debt の計画参照の検査はすべて OK。branch secret scan は `d217c575..1232f0d1` で clean。ログは `docs/evidence/verify-2026-11-01-183707.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)
- tech-debt の構造: `git show HEAD:docs/tech-debt/README.md` と行ごとに比べた。変わった行は 148、187、198、208、220、406、409、410 と、足した 421 行目だけ。変わった 9 行はどれも 5 欄に割れ、欄ごとの backtick と `~~` が偶数で、`file:line` の参照はない。ほかの行は同一
- plan の digest: `./scripts/plan-visual.sh digest` は `11ccf2ed5a4c`(承認時と同じ)
- `git status --porcelain`(コミット前): `docs/insights/events/2026-11-01-standard-config-gaps.jsonl`、`docs/plans/active/2026-11-02-standard-config-gaps.md`、`docs/tech-debt/README.md` が `M`(この report は `??`)。迷子のファイルはない
