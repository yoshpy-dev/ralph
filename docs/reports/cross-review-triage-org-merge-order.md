# Cross-review triage report: org-merge-order

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-order.md
- Base branch: main(merge-base ac34ccf3)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 4/4 (cap reached; cross-review cycle 3 の ACTION_REQUIRED と WORTH_CONSIDERING を直すため上限を 4 に上げた。cycle-count.json は 2 のまま)
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-14-org-merge-order.md(承認済み、digest c0b1e9230999。変わっていない。cycle 4 の決定と逸脱は `## Progress checklist` の最後の項目)
- 4 周目の理由: 下の「Cycle 3」の ACTION_REQUIRED #1(改名した repository で、注記の勧める `gh repo set-default` が効かない)を 813d6d2d で直した。注記は、改名なら `git remote set-url origin` で origin を向け直す、そうでなければ `gh repo set-default` を打つ、の両方を並べる。WORTH_CONSIDERING #2(結びつきのない org の読めない gate の記録が捨てられ、gate の記録だけのブランチが古い pass に戻る)は 1052a12a で直した。読めない記録は、結びつきがなければその org の最新の読める gate の記録の枝に付く
- Self-review report: docs/reports/self-review-2026-10-14-org-merge-order.md(cycle 4 は 2e3d272a で Merge: yes、CRITICAL・HIGH・MEDIUM なし、LOW C4-1〜C4-4。C4-2 は sync-docs の de3881e8 で直し、残りは tech-debt の行)
- Verify report: docs/reports/verify-2026-10-14-org-merge-order.md(cycle 4 は f0b34594 で pass。直す前の 999e0394 の binary を対照にして、結びつきのない org のブランチ 3 本が直す前は ready、直したあとは unreadable になることを確かめた。ずれ D8・D9 は de3881e8 で直した)
- Test report: docs/reports/test-2026-10-14-org-merge-order.md(cycle 4 は d0abfb8b で pass。mutation 29 件のうち 25 件が red。本物の gh で、改名した vuejs/vue-next を origin にすると PR の url が vuejs/core になり、古い注記の `gh repo set-default vuejs/vue-next` を gh が断り、新しい注記の `git remote set-url origin` で PR が使われることを確かめた)
- Implementation context summary: `codex exec review --base main` を ec127937 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは 232 バイト)。Codex の答えは「No actionable regressions were found against the specified merge base. Formatting and diff checks passed, and skill mirrors match. Targeted Go tests could not run because the read-only sandbox prevented temporary-directory creation.」で、指摘はない

## ACTION_REQUIRED

なし。

## WORTH_CONSIDERING

なし。

## DISMISSED

なし。

## Cycle 3

- Date: 2026-10-14
- Cycle: 3/3 (cap reached; cross-review cycle 2 の ACTION_REQUIRED を直すため上限を 3 に上げた。cycle-count.json は 2 のまま)
- Total reviewer findings: 2
- Cycle 3 counts: ACTION_REQUIRED=1, WORTH_CONSIDERING=1, DISMISSED=0

### Triage context(cycle 3)

- Active plan: docs/plans/active/2026-10-14-org-merge-order.md(承認済み、digest c0b1e9230999。変わっていない。cycle 3 の決定は `## Progress checklist` の最後から 2 番目の項目)
- 3 周目の理由: 下の「Cycle 2」の ACTION_REQUIRED #1(gh の既定の上限 30 を fork の PR が埋める)を d7599d82 で直した。`gh pr list` に `--limit 1000` を渡し、上限ぶんの PR がすべて fork なら、そのブランチの PR を使わずに注記する。同じ cycle の c885fcab で self-review の C2-1(gate の記録だけが名指すブランチの Detail に `--org-id` がない)と C2-2 を直した
- Self-review report: docs/reports/self-review-2026-10-14-org-merge-order.md(cycle 3 は a8fd9992 で Merge: yes、CRITICAL・HIGH・MEDIUM なし、LOW C3-1〜C3-5。sync-docs の f269c0db で tech-debt の行に載せた)
- Verify report: docs/reports/verify-2026-10-14-org-merge-order.md(cycle 3 は 715fc858 で pass。直す前の binary を対照にして、30 本の fork の PR があるブランチが直す前は PR なしの ready、直したあとは wrong_base と closed に分かれることを確かめた。ずれ D5〜D7 は f269c0db で直した)
- Test report: docs/reports/test-2026-10-14-org-merge-order.md(cycle 3 は d5bff097 で pass。mutation 44 件のうち 35 件が red。本物の gh で、microsoft/vscode の `--head patch-1` が `--limit` なしでは 30 本、1000 では 1000 本を新しい順に返すことを確かめた)
- Implementation context summary: `codex exec review --base main` を 999e0394 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは 1730 バイト)。Codex はテストを走らせていない。指摘は 2 件(P2 と P2)。分類と直し方は consult(consult-plan-cycle3)に聞いた

### ACTION_REQUIRED(cycle 3)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] PR の情報を捨てる前に、repository の改名を解く。GitHub の repository を改名し、origin が古い URL のままだと、git は GitHub の転送で同じ repository に届くが、PR の url は新しい名前を持つ。この比較は、2 つのコマンドが同じ repository を読んでいるのに PR をすべて捨てる。別の base に向いた gate 済みの PR が `wrong_base` でなく `ready` になり、merged・closed の PR がまた出てくる。正規の repository を割り出してから捨てる。勧めている `gh repo set-default` では、origin の古い URL は直らない | 本物の問題で、直す。PR を使わないときの分類は gh がないときと同じで、注記が `mergeOrderNoPRs` でそう言い、計画はこの劣化を認めている(PR は任意)。誤りの中身は、注記が勧める直し方が改名の場合に効かず、改名した repository では PR がずっと使われないこと。直し方は注記だけにする: gh が読む repository と origin のものが違うとき、改名ならそれに origin を向ける(`git remote set-url origin`)、そうでなければ `gh repo set-default` を打つ、の両方を並べる(改名か別の repository かは見分けられない)。正規の名前を `gh repo view` で引く案は、cycle 1 で `--repo` をやめた理由(host の解決、ssh の別名)が戻り、gh の呼び出しも 1 つ増えるので採らない(consult の判定) | internal/org/mergeorder.go(`mergeOriginRepo.check`)、internal/org/mergeorder_test.go、internal/cli/org_merge_order.go(help)、internal/cli/org_merge_order_test.go、`/org` skill の 4 面、README.md、docs/tech-debt/README.md |

### WORTH_CONSIDERING(cycle 3)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P2] gate の記録だけが名指すブランチで、新しい gate の証拠が読めないときは、そのブランチを読めないものにする。読める gate の記録で見つかり、完全な結びつきがないブランチで、そのあとの details の壊れた `gate_failed` は、ここで数えられて捨てられる。`boundBranch` に項目はないが、ブランチは古い pass の gate を持ったままなので、リモートの SHA が合えば `ready` になる。gate の記録だけのブランチはこの実装のほかの場所で扱っている。新しい読めない証拠を、関わりうるブランチに残し、古い pass に戻らないようにする | 本物だが、起きる見込みは低い。計画(Scope と C8)は「読めない記録は、前に結びつきがなければどのブランチにも付けず、注記で数を言う」と書いており、今のコードはそのとおりなので、計画に反してはいない。結びつきのない org は、結びつきの行が壊れた台帳か手で書いた台帳でしか起きない(gate は結びつきのある機能の org でしか打てない)ので、2 軸の表では「いま直す価値: 迷う」の WORTH_CONSIDERING に当たる。ただし誤りは黙って `ready` に寄る向きで、C8(読めない証拠があれば古い pass に戻らない)の趣旨に当たり、直す手間も小さい。#1 の cycle 4 に入れて直す。読めない記録は、結びつきがあればその枝に、なければその org が前に書いた読める gate の記録の枝(最新のもの)に付け、どちらもなければ今どおり付けない。計画の拡張なので Progress checklist に逸脱として書く | internal/org/mergeorder.go(`collectMergeTargets`、`mergeLedgerNotes`)、internal/org/mergeorder_test.go |

### DISMISSED(cycle 3)

なし。

## Cycle 2

- Date: 2026-10-14
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 1
- Cycle 2 counts: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context(cycle 2)

- Active plan: docs/plans/active/2026-10-14-org-merge-order.md(承認済み、digest c0b1e9230999。変わっていない。cycle 2 の逸脱は `## Progress checklist` の最後から 2 番目の項目)
- 2 周目の理由: 下の「Cycle 1」の ACTION_REQUIRED #1(gh が origin でない repository の PR を読みうる)を 15d4b947 と ee302b06 で直した。`--repo` は渡さない。gh が PR を返したら `git remote get-url origin` を 1 回だけ打ち、PR の url の owner/repo と origin の owner/repo を大文字小文字を無視して比べる。合わなければ PR をすべて捨てて注記を 1 行出し、gh をそれ以上打たない。host は比べず、owner/repo を読めない origin(ローカルのパスなど)では比べない。この 3 つの穴は tech-debt の行と help・skill・README に書いた(f1a05bc7)
- Self-review report: docs/reports/self-review-2026-10-14-org-merge-order.md(cycle 2 は 5d788e2c で Merge: yes、CRITICAL・HIGH・MEDIUM なし、LOW C2-1〜C2-6。C2-3 は sync-docs の f1a05bc7 で直し、残りは tech-debt の最後の行)
- Verify report: docs/reports/verify-2026-10-14-org-merge-order.md(cycle 2 は c8448321 で pass。ずれ D3・D4 は f1a05bc7 で直した)
- Test report: docs/reports/test-2026-10-14-org-merge-order.md(cycle 2 は 312b9df8 で pass。mutation 87 件のうち 72 件が red。本物の gh で、upstream を読む gh の PR を修正前の binary が使い、HEAD が捨てることを確かめた)
- Implementation context summary: `codex exec review --base main` を c82eb031 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは 813 バイト)。Codex はテストを走らせていない。指摘は 1 件(P2)。`lookupMergePR`(`internal/org/mergeorder.go`)は `gh pr list --head <branch> --state all` に `--limit` を渡していない。`gh pr list --help` によると既定の上限は 30 で、`--head` は `<owner>:<branch>` の形を受けない

### ACTION_REQUIRED(cycle 2)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] fork を外す前に、PR の候補を十分に取る。同じ名前の fork の PR が 30 本以上、本物の PR より新しく並ぶと、`gh pr list` は既定の上限 30 に届き、そのあと `pickMergePR` が cross-repository の PR を外す。本物の PR は結果に入らず、PR なしとして黙って返る。PR が別の base に向いている、または閉じているブランチが `ready` に見えうる。候補をページで辿るか、サーバー側で head の repository で絞ってから、番号の一番大きい PR を選ぶ | 本物の問題で、直す。gh の `--head` 単独の検索は、作成の新しい順に `--limit` まで辿る(consult が gh の source で確かめた)。fork の PR が上限を埋めると本物の PR が落ち、merged・closed・wrong_base が見えなくなる。起きる見込みは低い(slug の名前のブランチに、fork から 30 本以上)ので、2 軸の表では「いま直す価値: 迷う」の WORTH_CONSIDERING に当たる。ただし誤りは黙って `ready` に寄る向きで、merge してよい順を示すという merge-order の目的に当たり、直す手間も小さい。保守の原則で ACTION_REQUIRED に寄せた。直し方は `--limit` を大きく(1000。通常は 1 本しか返らず、gh は 1 ページ 100 件で辿るので費用は変わらない)し、上限に届いて fork でない PR が 1 本もないときは、そのブランチの PR を使わずに注記する。サーバー側で絞る `gh api` は、cycle 1 で `--repo` をやめた理由(host の解決、ssh の別名)が戻るので採らない | internal/org/mergeorder.go(`lookupMergePR`)、internal/org/mergeorder_test.go |

### WORTH_CONSIDERING(cycle 2)

なし。

### DISMISSED(cycle 2)

なし。

## Cycle 1

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-order.md
- Base branch: main(merge-base ac34ccf3)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- Cycle 1 counts: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context(cycle 1)

- Active plan: docs/plans/active/2026-10-14-org-merge-order.md(承認済み、digest c0b1e9230999。変わっていない)
- Self-review report: docs/reports/self-review-2026-10-14-org-merge-order.md(346304fc、CRITICAL・HIGH・MEDIUM なし、LOW 8、Merge: yes)
- Verify report: docs/reports/verify-2026-10-14-org-merge-order.md(a8882d6d、pass。ずれ D1・D2 は sync-docs の 93af1f87 で直した)
- Test report: docs/reports/test-2026-10-14-org-merge-order.md(e20f2616、pass。mutation 66 件のうち 58 件が red、テストの穴 M05・C12・M03・C06/C07・M09)
- Implementation context summary: `codex exec review --base main` を 93af1f87 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは 1055 バイト)。Codex はテストを走らせていない。指摘は 1 件(P2)。merge-order は origin を `git ls-remote --symref origin` で読み、PR は `gh pr list --head <branch>` を打った場所の repo で引く(`internal/org/mergeorder.go` の `lookupMergePRs`)。gh がどの repository を引くかは確かめていない

### ACTION_REQUIRED(cycle 1)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `gh pr list` に repository を渡していないので、`gh repo set-default` で origin でない repository を選んでいると、gh はその repository を引き、`listOriginBranches` は origin を読む。同じ名前のブランチの PR で、origin のブランチが merged / closed として隠れたり、関係のない PR が ready の行に付いたりする。`isCrossRepository` は PR の head と base の repository を比べるだけで、origin と比べてはいない。origin の repository を割り出して `--repo` で渡し、gh の既定が origin でない場合を回帰テストにする | 本物の問題で、直す。gh は remote が複数あるとき、`gh repo set-default` の既定(なければ remote の名前の優先順)で引く repository を決める(`gh repo set-default --help`)。fork から作業する repo では upstream を引き、origin の証拠と別の repository の PR の state を組み合わせて分類する。C5 で fork の PR を外したのと同じ種類の取り違えになる。別の repository の PR を読むと、merged / closed と誤って origin の ready のブランチを隠し、その PR の統合先で wrong_base と誤り、関係のない PR の URL を ready の行に出す。直し方は計画の Scope(PR の状態は任意で、取れなければ PR の欄を空にする)の範囲で選ぶ。選んだ直し方(`--repo` は渡さず、返った PR の url を origin と比べる)は consult の判定による。`--repo` の host は GH_HOST、認証済みの host、github.com の順に決まり、ssh の別名を解かないため | internal/org/mergeorder.go(`lookupMergePRs`、`pickMergePR`)、internal/org/mergeorder_test.go、internal/cli/org_merge_order_test.go |

### WORTH_CONSIDERING(cycle 1)

なし。

### DISMISSED(cycle 1)

なし。
