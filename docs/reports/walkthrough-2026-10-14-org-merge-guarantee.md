# Walkthrough: org-merge-guarantee(7 段目の残りの後半、7c-2)

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-guarantee.md(/pr で archive に移す)
- Branch: feat/org-merge-guarantee(base a16d7c4f。この walkthrough を書いた時点で 29 commits、38 files、+7037/-90)
- 読む範囲: 差分の約半分は `docs/evidence/` と `docs/reports/` で、読むのは各報告の要約だけでよい。報告・evidence・insight・計画を除くと 21 files、+3745/-90 で、そのうちテストが `internal/org/mergeguarantee_test.go`(+1334)、`internal/cli/doctor_merge_guarantee_test.go`(+465)、`internal/cli/status_merge_guarantee_test.go`(+409)ほか。ほかに `/org` skill の 4 面、README、AGENTS.md、仕様 FR-7 の注、tech-debt の 8 行
- 後工程は 2 回まわした(既定の上限 2)。cross-review の指摘は cycle 1、cycle 2 とも 0 件。cycle 2 は self-review の LOW と test の穴を直すために回した(下の 6)

新しいファイルは `internal/org/mergeguarantee.go`(保護を読んで種類を決める、キャッシュを読み書きする)、`internal/cli/doctor_merge_guarantee.go`(doctor の検査)と、それぞれのテスト。次の順に読むと、依存の向きに沿って追える。

## 1. 種類を決める(`internal/org/mergeguarantee.go` の `CheckMergeGuarantee` と `decide`、f6bbd49c)

- 種類は 3 つ: `github_protection`(GitHub が、PR の必須の検査を最新の既定のブランチの上で通してから merge させる)、`pr_checks_only`(gate と PR 作成時の CI だけ)、`unknown`(読めなかった)
- `decide` は上から順に読み、決まった時点で止める(計画の C4)
  1. `readOrigin`: `git remote get-url origin` から host と owner/repo を読む(7c-1 の `mergeOriginFromURL` を host も返すように広げた。`internal/org/mergeorder.go`)
  2. gh が PATH になければ gh を打たずに `unknown`
  3. `readRepository`: `gh api repos/<o>/<r>` の `default_branch` と `full_name`
  4. `readMergeRules`: `rules/branches/<既定>` の有効な ruleset の規則。`strict_required_status_checks_policy` がその規則の中に必須の検査を持てば `github_protection`。`merge_queue` があり ruleset に必須の検査があっても同じ
  5. `readMergeClassic`: `branches/<既定>` の classic の保護の必須の検査の数。merge queue と classic の必須の検査の組み合わせも `github_protection`
  6. `readMergeClassicStrict`: classic に必須の検査があるときだけ `protection/required_status_checks` の `strict` を読む。admin の権限が要るので、404 `Not Found` と 403 は `unknown`(推測で `github_protection` にしない)。`Branch not protected` と `Required status checks not enabled` は strict false
- 必須の検査が 0 件の strict や merge queue は `pr_checks_only`(最新にはなるが検査が走らない)
- どの gh にも `GH_PROMPT_DISABLED=1` と `--hostname <origin の host>` を渡す(`mergeGuaranteeAPIHost`。`ssh.github.com` は `github.com`)。owner、repo、ブランチは path で escape する。git と gh の全体に 1 つの期限 20 秒をかけ、失敗はすべて `unknown` と理由の 1 文になる(error は返さない)
- `pr_checks_only` の Detail(`checksOnly`)は何を設定すればよいかを言い、merge queue は ruleset のものだけを勧める(89b981d8。doctor が読むのは ruleset の merge queue だけなので)

## 2. doctor の検査とキャッシュの書き込み(`internal/cli/doctor_merge_guarantee.go`、`internal/cli/doctor.go`、`SaveMergeGuarantee`、e3f3403b)

- doctor の Check 11c `Merge guarantee` はいつも打つ。状態は `github_protection` が `pass`、`pr_checks_only` が `warn`、`unknown` が `info` で、終了コードを変えない(計画の C3)
- 結果は `<state dir>/merge-guarantee.json` に書く(種類、理由の 1 文、host、owner/repo、gh の `full_name`、既定のブランチ、`checked_at`)。書くのは台帳のディレクトリが既にあるときだけで、`unknown` でもいつも書く(C2、C5)。一時ファイルに書いて rename する
- 書けなければ古いキャッシュを消し、消せもしなければ検査を `warn` にする(古い `github_protection` が残る経路をこの 1 つにするため)
- `internal/cli/main_test.go` の TestMain が `doctorMergeGuaranteeEnv` を、何も打たない Exec、必ず失敗する gh の探索、存在しない state dir に固定する。doctor を通すテストが本物の git と gh を打たず、repository の台帳にキャッシュを書かないため(AC8)

## 3. キャッシュを読む(`ReadMergeGuarantee`、`ForDefaultBranch`、`MergeGuarantee.Text`)

- 読む側は gh を打たない。次のどれかなら `unknown` と理由を返す: 台帳のディレクトリがない(`no_ledger`)、キャッシュがない(`not_checked`)、読めない(`unreadable`)、`checked_at` から 24 時間を過ぎた(`stale`、定数 `MergeGuaranteeMaxAge`)、今の origin の host か owner/repo が大文字小文字を無視しても違う(`origin_changed`)、origin を読めない(`origin_unreadable`)
- origin の照合で打つのはローカルの `git remote get-url origin` だけで、キャッシュが新しいときだけ打つ
- `ForDefaultBranch` は、キャッシュの既定のブランチが別のブランチなら `unknown`(`default_branch_changed`)にする。merge-order が ls-remote で読んだ既定のブランチと比べるのに使う
- `Text` は 1 行: `<kind> (<branch> of <host>/<repo>, checked <checked_at>): <detail>`、または `unknown (<理由>)`

## 4. status と merge-order の行(`internal/cli/status.go`、`internal/cli/org_merge_order.go`、`internal/org/mergeorder.go`、0e3813dc)

- `ralph status` は `org watch:` の行のあとに `merge guarantee:` の行を出す。`--json` では `merge_guarantee`
- `ralph org merge-order` は既定のブランチの行のあとに同じ行を出す。台帳に対象のブランチがなければ ls-remote を打たないので、この行が先頭になる。`MergeOrderResult.MergeGuarantee` は `ReadMergeGuarantee(...).ForDefaultBranch(res.DefaultBranch)`
- どちらも行を `mergeGuaranteeLine`(`inboxField`)に通し、キャッシュの中の改行や制御文字で行を壊せない

## 5. 文書(d7da3441、c56c1f11、sync-docs の 240b0ce8 と 97f52ffa)

- `/org` skill の 4 面(保証の節)、README の doctor と status の行と merge-order の段落、AGENTS.md の repo map、仕様 FR-7 の 7c-2 の注
- tech-debt: 既存の 2 行を更新した(gate を飛ばした PR を止めるものがないこと、merge したあとに別のブランチを `ready` に戻す手がないこと。どちらも持ち主は 8a 段目)。新しい 6 行は、ssh の別名の origin、fork の origin、`github_protection` が管理者や bypass を言わないこと、`ssh.github.com` と `github.com` の書き方を変えると `origin_changed` になること、TestMain の固定、この PR の小さい残り(244 行目の (a)〜(i))

## 6. cycle 2 の直し(59abeaec)

cycle 1 の cross-review は 0 件だったが、self-review の LOW が登録簿に載っていなかったので cycle 2 を回した。

- L3: doctor の Detail に gh の stderr の制御文字がそのまま出た。`mergeGuaranteeErrText` は空白類を畳み、1 文字ずつ `PrintableInboxText` に通し、そのあとで 300 バイトに切る(エスケープで伸びた分も数える)
- L4: strict の 403 を原因によらず「admin がない」と言っていた。404 `Not Found` と 403 を別の case にし、403 は admin のないことを原因の 1 つとだけ言う(rate limit や SSO の承認でも返りうる)
- test の Gap 1: message が想定外の 404(`Branch not found`)が `unknown` のままであることのテストがなく、`pr_checks_only` に変える mutant が生き残っていた。表に 1 行足した
- cycle 2 で新しく出た LOW(doctor の Detail の cache のエラーや path の escape、`readMergeClassicStrict` の doc の 403 の主張、呼び出し箇所ごとの escape のテスト、Detail の検査の数のテスト)は、最後の cycle なので直さず、244 行目の (g)(h)(i) に記録した

## 7. 確かめたこと(evidence)

- 本物の gh(読むだけ)で doctor と status を打った(`docs/evidence/merge-guarantee-livefire-2026-10-14.md`): この repo は `pr_checks_only`(warn)、github/docs は ruleset の strict で `github_protection`(pass、必須の検査 37 件)、cli/cli は classic の strict を admin なしで読めず `unknown`(info)、届かない GHES の host は `unknown`。origin を変えると `origin_changed`、git の外では `origin_unreadable`、台帳がなければ書かずに `no_ledger`
- テスト: go1.25.8 と go1.26.0 の両方で `go test ./internal/cli/... ./internal/org/...` が rc 0、`-race` で DATA RACE なし。mutation は cycle 1 が 74 件、cycle 2 が 31 件(`docs/reports/test-2026-10-14-org-merge-guarantee.md`)
- テストは本物の gh を打たない: PATH の先頭に置いた偽の gh は 0 回しか呼ばれず、本物の既定値で呼ぶ probe では 1 回呼ばれる(感度の確認)
