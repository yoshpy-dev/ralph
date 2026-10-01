# Self-review report: changed-languages-merge-base

- Date: 2026-10-01
- Plan: docs/plans/active/2026-10-01-changed-languages-merge-base.md
- Branch: fix/changed-languages-merge-base(HEAD 5361955b、コードのコミットは 5bf0d0ad)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(変えたロジックの正しさ、命名、読みやすさ、不要な変更、安全性、保守性、POSIX sh / dash での可搬性)。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。対象は `git diff main...HEAD` から `docs/plans/` を除いたもの

## Evidence reviewed

- `git diff main...HEAD --stat`: 7 ファイル、+493 / -16(plan を除くと 6 ファイル)
- `cmp scripts/detect-changed-languages.sh templates/base/scripts/detect-changed-languages.sh`: 一致。`quality-gates.md` はファイル全体では root と template が以前から異なる(`scripts/check-sync.sh:98-101` の KNOWN_DIFF)。この diff の hunk だけを比べると root と template で同じ
- `remote_default_ref` の変数(`remote`、`head_target`、`candidate`)と、呼び出し側の `current_ref`、`tracked_remote` は、127〜166 行と先頭のコメントの外では使われていない。関数は 152 行と 164 行の `$(...)` の中からしか呼ばれないので、サブシェルで動き、値が外に漏れることもない
- 兄弟のスクリプトの扱い: `scripts/secret-scan-branch.sh:164-181` は、存在を確かめた完全な ref 名(`refs/remotes/origin/<base>`)を `git merge-base` に渡し、短い名前を渡さない理由をコメントで書いている
- scratchpad の fixture を使った probe。git の設定は `GIT_CONFIG_GLOBAL=/dev/null`、`GIT_CONFIG_NOSYSTEM=1`、`HOME` を scratch に向けて切り離した。旧版は `git show main:scripts/detect-changed-languages.sh`、新版は HEAD の版、「修正案」は下の M-1 の 4 行を変えた版。macOS の `/bin/sh` と `/bin/dash` で同じ結果だった(F と G を両方で実行)
- `docs/tech-debt/README.md`: この検出器や #190 に関わる行はない。この diff が閉じる行も、無効にする行もない

### probe

| probe | fixture | 旧版 | 新版 | 修正案 |
| --- | --- | --- | --- | --- |
| A | `feature` を `--track origin/main` で作り、Go を commit(未 push)。ローカルの branch `origin/main` を feature の先端に作る | `golang`(upstream は `--abbrev-ref` で `remotes/origin/main` と出る) | `no_changes` | `golang` |
| B | push 済みの feature(upstream == HEAD)に Go の commit。tag `origin/main` を先端に作る | `no_changes`(#190 の不具合) | `no_changes` | `golang` |
| D | `origin/HEAD` を `refs/heads/main`(`refs/remotes/origin/` の外)に向ける。push 済みの feature に Go の commit | `no_changes` | `golang`(`origin/main` に落ちる) | — |
| E | remote は `central` だけで、一度も fetch していない(remote-tracking の ref がない)。`branch.main.remote=central`。main の上に未 push の Go の commit | `full`(`reason=no_merge_base:@{upstream}`) | `no_changes` | `no_changes` |
| F | remote なし、main の上に Go を commit、作業ツリーは clean | `no_changes` | `no_changes` | `no_changes` |
| G | remote は `heads` という名前だけ。main を push し、その上に未 push の Go の commit | `golang` | `no_changes` | `golang` |
| 名前 | `branch.feature.remote` を手で `r*`、`-x`、`../up.git`、URL、`a b`、`.` にする | — | どれも `golang`(ローカルの main に落ちる) | — |

修正案の版を scratch の `scripts/` に置き、`tests/test-detect-changed-languages.sh` をそのまま実行すると 48 / 48 だった。

## Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M-1 | MEDIUM | 正しさ(同じ短い名前の ref) | 存在を確かめた ref は `refs/remotes/<remote>/<branch>` や `refs/heads/main` だが、`git merge-base` に渡すのは `origin/main`、`main` といった短い名前。git は短い名前を `refs/tags/<name>`、`refs/heads/<name>`、`refs/remotes/<name>` の順に解決するので、同じ名前の tag やローカルの branch があると、確かめていない ref が base になる。base が HEAD を指すと `no_changes` になり、#190 と同じ「Go を変えたのに言語 pack が 1 つも走らない」状態に戻る。旧版も upstream がない経路では同じ性質を持っていた。一方で upstream がある経路では `--abbrev-ref` が曖昧でない名前(`remotes/origin/main`)を返していたので、そこは今回の変更で後退した(probe A と G は旧版 `golang`、新版 `no_changes`)。きっかけは、うっかり作った `origin/main` という名前のローカルの branch、`main` や `origin/main` という tag、`heads` や `tags` という名前の remote。どれも珍しいが、この repo は #169 で同じ罠を `secret-scan-branch.sh` に記録している。plan の Deviation notes には、HEAD の判定で `--short` を避けた理由として同名の tag が挙がっているが、merge-base に渡す側には同じ配慮がない | `scripts/detect-changed-languages.sh:136`(`${head_target#refs/remotes/}`)、`:143`(`%s/%s`)、`:169`、`:171`、`:179`。兄弟の扱いは `scripts/secret-scan-branch.sh:174-181`。probe A、B、G | `remote_default_ref` は `"$head_target"` と `refs/remotes/%s/%s` を出し、ローカルの段は `refs/heads/main` / `refs/heads/master` を入れる(4 行の変更)。`RALPH_VERIFY_BASE` は利用者の指定なのでそのまま渡す。merge-base が取れないときの `reason=no_merge_base:` は既定の base では完全な名前になるが、この値を読む側はない(`run-verify.sh` は表示するだけ)。回帰テストには probe A の fixture(ローカルの branch `origin/main` を先端に置く)が使える |
| L-1 | LOW | 正しさ(base が今の branch そのもの) | 追跡先の remote に remote-tracking の ref が 1 つもないと、`remote_default_ref` は何も返さず、ローカルの main に落ちる。HEAD が main の上にあると merge-base は HEAD になり、未 push の commit が見えずに `no_changes` を返す(probe E)。旧版はここで `full` だった。ただしこれは、失敗した `git rev-parse` が標準出力に `@{upstream}` をそのまま出し(rc 128)、それを base にして merge-base が失敗した結果で、意図した挙動ではない。remote のない repo で main の上に commit した場合(probe F)は旧版も新版も `no_changes` で、今回の変更とは関係なく前からある。plan の Design decisions は「ローカルの main を base にすると merge-base が HEAD になる」ことを理由にローカルの段を最後に回しているが、最後に回しても、そこへ落ちる経路が残る | `scripts/detect-changed-languages.sh:162-172`、`:179`。probe E、F | 少なくとも、先頭のコメントに「base が今の branch そのもの(例: remote のない repo の main)になると、commit 済みの変更は見えず、未 commit の変更だけが対象になる」と書く。挙動で塞ぐなら、既定の base が今の branch の `refs/heads/<branch>` と一致したときに full に落とす案がある。ただし既存のケース 1〜9(remote のない main の上の未 commit の変更)が full に変わるので、契約の変更として別 issue で判断する |
| L-2 | LOW | コメントと挙動の一致 | 先頭のコメントの「The branch's own @{upstream} is never the base」は、upstream が default branch そのものの場合に字面どおりでなくなる。main が `origin/main` を追跡しているとき、`git checkout -b feature --track origin/main` で作った branch のとき、ケース 17(main が `central/main` を追跡)のとき、base は `@{upstream}` と同じ ref になる。言いたいのは「`@{upstream}` を base の候補として読まない」こと。また、先頭のコメントも関数のコメントも、`refs/remotes/<remote>/HEAD` の指す先が `refs/remotes/<remote>/` の外にあると無視する条件を書いていない(probe D では `origin/main` に落ちた) | `scripts/detect-changed-languages.sh:17-23`、`:127-129`、`:133-134` | 「@{upstream} is never consulted」のように、読まないことを書く。HEAD の段には「(only when it exists under refs/remotes/<remote>/)」を足す |
| L-3 | LOW | 文書の言い方 | `quality-gates.md` の新しい文は「`RALPH_VERIFY_BASE` を指定したときはそれに対して差分を取り、指定しないときは default branch との merge-base に対して取る」と読める。実際には指定したときも `git merge-base HEAD "$base_ref"` を通る。base が HEAD の祖先でないとき、利用者は 2 点間の差分を想定して base を指定しうる | `docs/quality/quality-gates.md:43-46`(template も同じ)、`scripts/detect-changed-languages.sh:179` | 「the diff is taken from the merge-base of HEAD and `RALPH_VERIFY_BASE` when set, otherwise the default branch (...)」のように、両方とも merge-base だと分かる形にする |
| L-4 | LOW | テストの隔離 | 新しい fixture は `git push`、`git symbolic-ref`、branch と remote の設定を使うが、開発者の global / system の git 設定から切り離していない。同じ repo の 4 本のテスト(例: `tests/test-secret-scan-branch.sh:48-53`)は `HOME`、`GIT_CONFIG_GLOBAL`、`GIT_CONFIG_SYSTEM`、`GIT_CONFIG_NOSYSTEM` を固定している。global の `core.hooksPath` にある pre-push hook や `push.gpgSign` が fixture の push で動く。結果をすり替える設定は見つからなかった。`branch.autoSetupMerge`、`push.default`、`remote.pushDefault`、`init.defaultBranch` は、明示の `checkout -B` と `push -u <remote> <branch>` で上書きされる。起きうるのは、テストが目に見える形で落ちることだけ。既存のケースの commit も、前から同じように隔離されていない | `tests/test-detect-changed-languages.sh:116-135`、`tests/test-run-verify-scope.sh:161-181` | 2 本のテストの冒頭(`unset` の隣)に、兄弟のテストと同じ 4 変数の固定を足す |
| L-5 | LOW | テストの名前 | ケース 18 の「Detached HEAD skips the tracked-remote step and uses the local main」は、fixture では確かめられない。`make_repo_with_remote` が main を push するので、`central/main` とローカルの main は同じ commit で、`central/main` を使う版でも通る。ケース 19 の `.` を飛ばす分岐も、外しても結果は変わらない(`refs/remotes/./…` は不正な ref 名で、`remote_default_ref .` は何も返さない。`.` を外した版で `golang` を確認)。どちらのケースも、`|| true` を外したときに `set -e` で止まることは防いでいるので、残す価値はある | `tests/test-detect-changed-languages.sh:330-354`、`scripts/detect-changed-languages.sh:155`、`:163` | 名前を確かめている内容に合わせる(例: 「detached HEAD still selects golang」)。ローカルの段を使ったことまで確かめたいなら、ケース 18 で central に main より古い commit だけを置く |

CRITICAL と HIGH はない。

## 依頼された観点への回答

- 短い名前を merge-base に渡す件: 実際に起きる(M-1)。旧版は upstream のない経路で同じ性質を持っていたが、upstream のある経路は `--abbrev-ref` のおかげで影響を受けなかった。その経路については今回の変更で後退した。
- `remote_default_ref` の大域変数: 衝突はない。上の Evidence のとおり、名前はほかで使われておらず、関数はサブシェルからしか呼ばれない。
- remote の名前に変わった文字がある場合: `git remote add 'r*'` は git が拒否する。手で書いた `branch.<b>.remote` の値はどれも何も返さずにローカルの main に落ちた。git に渡す引数はすべて `refs/` で始まるので、`-x` のような値がオプションとして解釈されることもない。`case` のパターンは引用符で囲んであるので、glob の文字は文字どおりに扱われる。
- `refs/remotes/<remote>/HEAD` が外を指す場合: 正しく飛ばす(probe D)。コメントにその条件がない(L-2)。
- `branch.<b>.remote` の remote に ref がない場合: L-1。
- コメントと `quality-gates.md` の文: L-2 と L-3。
- fixture の隔離、決定性、後始末: 隔離は L-4。決定性の問題はない(時刻や並び順に依存しない。名前は `mktemp -d` か、`mktemp -d` で作った `$workdir` の下の固定名)。後始末は既存の `trap cleanup EXIT HUP INT TERM` に入る。`$workdir` の外に何も作らない。

## Positive notes

- `make_repo_with_remote` の後に `assert_upstream_is_head` で「upstream == HEAD」を確かめているので、fixture が #190 の状態を再現していないまま緑になることはない。scope 側のテストも同じ確認を持つ。
- テストの冒頭で `RALPH_VERIFY_BASE`(scope 側は `RALPH_VERIFY_SCOPE` も)を `unset` している。呼び出し元の環境から既定の base の経路を飛ばされることがない。
- HEAD の段は、指す先が `refs/remotes/<remote>/` の下にあり、かつ実在するときだけ使う。dangling の HEAD と外を指す HEAD の両方で、次の候補に正しく落ちる(ケース 14、probe D)。
- detached HEAD の判定は `git symbolic-ref --quiet HEAD` から `refs/heads/` を外す形で、`--short` が同名の tag で `heads/<name>` を返す問題を避けている。
- scope 側の fixture は、git を初期化する前の雛形を `skel` に複製してから使うので、remote 付きのケースが前半のケースの commit を引き継がない。

## Coverage gaps

- 古い git(`alpine:3.4` の 2.8.6 など)での確認はしていない。新しく使った `git symbolic-ref --quiet`、`git show-ref --verify --quiet`、`git config --get` は古くからあるので、問題はないはず。未確認です。
- Linux の dash はコンテナでは試していない。macOS の `/bin/dash` で probe F と G を実行し、`/bin/sh` と同じ結果を得た。
- テストスイートの判定は /test の担当なので実行していない。修正案の妥当性を確かめるために、scratch の写しで検出器のテストだけを 1 回実行した(48 / 48)。

## Recommendation

- Merge: 可。CRITICAL と HIGH はない。M-1 は 4 行で直り、修正案の版で既存の 48 ケースが通ったので、マージ前に直すことを勧める
- Follow-ups:
  - M-1 を直す場合は、probe A の形(ローカルの branch `origin/main` を先端に置く)の回帰ケースを 1 つ足す
  - L-1 の「base が今の branch そのもの」になる経路(remote のない repo の main など)を full に落とすかどうかは、既存の契約(ケース 1〜9)を変えるので別 issue で判断する
  - L-2〜L-5 はコメント、文書、テストの名前の直しで、挙動は変わらない
