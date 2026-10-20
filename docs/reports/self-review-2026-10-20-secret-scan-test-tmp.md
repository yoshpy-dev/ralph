# Self-review report: secret-scan-test-tmp

- Date: 2026-10-20
- Plan: `docs/plans/active/2026-10-20-secret-scan-test-tmp.md`(tech-debt 170 行の解消、パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff e333200c...HEAD`(HEAD 20fa08c5、4 ファイル、+193/-9。コードは `tests/test-secret-scan.sh` の 20 行だけ)。diff の品質だけを見た。仕様適合、テストの網羅、文書のずれ、静的解析、テストの実行は `/verify` と `/test` に任せ、このレビューでは走らせていない

## Evidence reviewed

- 全文を読んだ: `tests/test-secret-scan.sh` の 1〜60 行(`expect_exit`、新しい `capdir`、`trap`、`workdir`)と、`trap`・`mktemp`・`capdir`・`/tmp`・`set -` の全 grep、`docs/evidence/secret-scan-test-tmp-2026-10-20.md`、計画 `docs/plans/active/2026-10-20-secret-scan-test-tmp.md`
- diff を読んだ: `tests/test-secret-scan.sh` の 5 か所(`expect_exit`、`capdir`+`trap`、`expect_stderr_contains`、`expect_stderr_not_contains`)、`docs/tech-debt/README.md`(170〜171 行)、計画の f9762506 から 20fa08c5 までの差(AC の印と進捗の注だけ)
- 照らし合わせた: base の `tests/test-secret-scan.sh`(`git show e333200c:`)の `/tmp/ralph-secret-scan-test.*` 8 行と新しい版の置き換え、`docs/tech-debt/README.md` の `test-secret-scan.sh` を含む 3 行(170・171・178)、完全に閉じた行 58 件(この PR の 171 行を除く)の欄ごとの打ち消しの形(Python で数えた)、`scripts/`・`tests/`・`.github`・`.claude/hooks`・`.claude/skills` の `ralph-secret-scan` を含む行
- 手元で確かめた(読むだけ。テスト本体は走らせていない): `git diff --check e333200c...HEAD`(rc 0)、追加行の `/Users/`・`hiroki`・16 桁以上の 16 進・token の grep(なし)、171 行を `\|` 以外の `|` で分けて 5 欄、170 行の HTML コメントの `-->` が 1 つ。scratchpad の小さな sh で、同じ形の `set -eu` と `trap 'rm -rf …' EXIT HUP INT TERM` の script に TERM を送る試し(trap は消して、script は再開した)と、cwd を消した git の終了コード(128)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (M1) | maintainability(記録の主張) | tech-debt の別の行が、この PR が直した欠陥と同じものを開いたまま持つ。HEAD の 178 行(`Test gaps left by org-stop-all`)の項目 (e) は「`tests/test-secret-scan.sh` writes the fixed files `/tmp/ralph-secret-scan-test.out` and `.err` … so two overlapping runs clobber each other's output」、Impact (e) は「A spurious red when two runs overlap」、Trigger (e) は「The first overlap flake: use `mktemp` for the two files」。この PR は 171 行(元の 170 行)だけを閉じ、178 行に触っていない。register は同じ欠陥を、閉じた行と開いた行で 2 つ持つことになる。計画の Non-goals にある検索は `tests/*.sh` と `scripts/*.sh` の固定 `/tmp` パスで、register は入っていない | `docs/tech-debt/README.md` 178 行の Debt・Impact・Trigger の各欄の (e)。`grep -n 'test-secret-scan\.sh' docs/tech-debt/README.md` の 170・171・178 行。計画の Scope は「row 170」だけ | 178 行の (e) を 3 欄(Debt・Impact・Trigger)で閉じる。行は (a)〜(d) が開いているので残し、(e) の文を打ち消して「(e) RESOLVED 2026-10-20 in fix/secret-scan-test-tmp(171 行を見る)」を添える。1 欄だけだと、同じ行に「いま開いている」と「閉じた」が並ぶ。register の編集は `post-implementation-pipeline.md` の「記録だけの修正」の外なので、直すとパイプラインの再実行になる |
| LOW (L1) | maintainability(evidence の再現性) | evidence の「Before the change, two at once」の行が、旧版を `git show HEAD:tests/test-secret-scan.sh` で取ると書く。この commit(ece77f30)のあとは HEAD が直した版なので、読み手がそのまま打つと直した版を 2 つ走らせ、旧版の `FAIL: 2` は再現できない。表の上には base `e333200c` と書いてある | `docs/evidence/secret-scan-test-tmp-2026-10-20.md` 15 行と 3〜5 行 | `git show e333200c:tests/test-secret-scan.sh` にする |
| LOW (L2) | maintainability(evidence の読み取り) | 同じ evidence の TERM の行が「after it, exit 128 and the dir was empty」と書き、128 を説明しない。TERM で止まったなら 143 のはずで、128 は別の理由に読める。scratchpad の試しでは、同じ形の trap(`exit` なし)は TERM のあと dir を消すが script は止まらず再開し(rc 0、最後まで走る)、消えた `capdir` への redirect が失敗した。git は cwd が消えると `fatal: Unable to read current working directory` で 128 を返した。この run の 128 はおそらくそれで、trap のあとに再開した script が最初の git で落ちた終わり。未確認(テストは走らせていない)。つまりこの evidence が示すのは「trap が dir を消した」で、「TERM で止まった」ではない。trap の形は base の 41 行と同じで、この PR が入れたものではない | `docs/evidence/secret-scan-test-tmp-2026-10-20.md` 16 行と Notes(19 行〜)。`tests/test-secret-scan.sh` 44 行の `trap '…' EXIT HUP INT TERM`(base 41 行と同形) | Notes に 1 行足す(「128 は trap が dir を消したあと script が再開して落ちた終わりと見ている。TERM で止まったのではない」)か、「exit 128」を落とす。trap に `exit` を足す直しは別の変更で、この計画の範囲の外 |
| INFO (I1) | readability | `trap 'rm -rf "$capdir" "${workdir:-}"' …` の `${workdir:-}` は、trap を `workdir` の `mktemp` の前に置いたので要る(`set -eu` で、`workdir` の作成前に signal や `mktemp` の失敗が来ても trap が unbound variable で落ちない)。コメントはこれに触れない。次に読む人が `:-` を落とすと、その窓で `capdir` が残る | `tests/test-secret-scan.sh` 40〜45 行 | コメントに「`workdir` は次の行で作る。作る前に signal が来ても trap が落ちないよう `:-` を付ける」を足す。急がない |

## Positive notes

- 置き換えの漏れがない。base の `/tmp/ralph-secret-scan-test.*` は 8 行(redirect 1、失敗時の cat 2、`expect_stderr_contains` の grep と cat 2、`expect_stderr_not_contains` の grep と cat 2、trap 1)で、diff の `-` 側と 1 対 1。いまの script の `/tmp` は `${TMPDIR:-/tmp}` の 2 か所(43・45 行)だけで、`ralph-secret-scan-test` の語は 43 行の prefix だけ。assertion の文、期待値、順序は変わっていない
- `capdir` を `$workdir` の外に置いた判断(計画の D1)は diff で守られている。`$workdir` は `git init` した fixture で、その中に `out`/`err` を置くと各 check が見る repository に未追跡のファイルが出る。prefix `ralph-secret-scan-test.` は、scanner の `ralph-secret-scan.$$`、兄弟の `ralph-secret-scan-branch-*`(`scripts/secret-scan-branch.sh`、`tests/test-secret-scan-branch.sh`)のどれとも前が重ならず、`ralph-secret-scan*` の glob で消す・数える箇所も `scripts/`・`tests/`・`.github`・`.claude/` にない
- `trap` の順が正しい。`capdir` の `mktemp -d` が失敗すれば `set -e` で trap の前に終わり、何も作っていない。`workdir` の `mktemp` が失敗しても、trap が `rm -rf "$capdir" ""` を走らせる(`-f` で空の引数は黙って通る)。HUP・INT・TERM・EXIT の 4 つは base と同じ
- コメント(40〜42 行)は動きと合う。「per run」「outside the fixture repository」「two copies … do not overwrite or delete each other's captures」は、`capdir` が `mktemp -d` で作られ、trap が `capdir` を消すことと一致する。長さも前後に合う(最長 75 字)
- tech-debt 171 行は、完全に閉じた 58 行のうち 44 行と同じ形(Debt 欄の `~~…~~ (RESOLVED <日付> in <branch>)` と Trigger 欄の打ち消し、Impact と Why は現在形のまま)で、欄は 5 つ、170 行の `<!-- RESOLVED … -->` は先行の行(16・18・21 行ほか)と同じ置き方。閉じた旨の文は evidence と合う(単独 123/0、2 つ同時に両方 0 と `FAIL: 0`、旧版は 1 つが `FAIL: 2`、TERM のあと dir なし)。`$workdir` に置かずに別の dir にした理由も書いてある。計画の道は active を挙げていて、`/pr` の archive が直す(計画の進捗の注に経緯がある)
- evidence に `$HOME` の絶対パス、16 桁以上の 16 進、token の値、pane id はない。`git diff --check` は rc 0。SIGINT が背景の `sh` で無視されるため TERM を使った経緯も Notes に残っている
- 計画の差(f9762506 から 20fa08c5)は AC の印と進捗の注だけで、Scope・Non-goals・D1 は変わっていない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | — | cycle 1 で、M1 は register の編集で直せ、L1・L2・I1 は evidence とコメントの 1〜2 行で直せる。cap の最後の回で残るときだけ、1 行にまとめて足す | — | — |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件は、tech-debt 178 行の項目 (e) が、この PR が直した欠陥を開いたまま持つ点(M1)。LOW 2 件は、evidence の旧版の取り方が HEAD 基準で再現できない点(L1)と、TERM の行の「exit 128」が trap のあとに script が再開した終わりに見える点(L2)。コードの動きを壊すものはない。置き換えの漏れ、`capdir` の置き場所、`trap` の順、prefix の衝突、171 行の書式、evidence の個人情報には欠陥が見つからなかった)
- Follow-ups:
  - M1 は記録の文だけの直し(178 行の (e) を 3 欄で閉じる)で、コードは変わらない。tech-debt の編集は `post-implementation-pipeline.md` の「記録だけの修正」の例外(`docs/reports/` のレポートと insight event だけ)の外なので、直すとパイプラインの再実行になる。直すなら L1・L2・I1(evidence 2 か所とコメント 1 行)も同じ commit に入れると手間が少ない
  - L2 の trap に `exit` がない点(TERM を受けても止まらず、dir が消えたまま走り続ける)は、この PR が入れたものではなく base から同じ。直すなら別の変更
  - 直さずに cap(2)のまま進むなら、M1・L1・L2 は cycle 2 の self-review で残りとして 1 行の tech-debt にまとめる(今回は足していない)
  - この節でレポート以外は変えていない。insight event(`self_review`、cycle 1、medium 1、low 2)を `docs/insights/events/2026-10-20-secret-scan-test-tmp.jsonl` に足し、レポートと同じ commit に入れる
