# Sync-docs report: guard-bypass-and-hygiene

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md
- Agent: doc-maintainer subagent
- Branch: fix/guard-bypass-and-hygiene(base 2a22ba78、編集前の HEAD 37514051)
- 前段の report: `self-review-2026-10-07-guard-bypass-and-hygiene.md`(Merge 可、M-1 と L-1〜L-5)、`verify-2026-10-07-guard-bypass-and-hygiene.md`(pass、V-1〜V-9)、`test-2026-10-07-guard-bypass-and-hygiene.md`(pass、Test gaps 1〜9)

## Summary

verify の文書のずれ V-1〜V-7 と V-9 を直し、self-review の tech-debt 提案 2 行と test の Test gaps 1 行を `docs/tech-debt/README.md` に足した。V-8(AC4 の直し方の表示が plan の文言より細かい)は plan の本文が digest の対象なので直さず、Progress checklist に 1 行のメモを足した。`.codex/README.md`、`git-commit-strategy.md`、`quality-gates.md`、`definition-of-done.md` は新しい guard の挙動と食い違う記述がなく、変更していない。コードと tests は変えていない。plan の本文は変えておらず、digest は `c07adf402bdb` のまま(`./scripts/plan-visual.sh digest` で確認)。

## Files changed

| File | Change |
|------|--------|
| `docs/insights/README.md`、`templates/base/docs/insights/README.md` | insight event を書く skill の一覧に `/sync-docs` を足した。冒頭の段落(`:4`)と "Appending events"(`:118-119`)の 2 か所。root と template は `cmp` で一致。V-1 |
| `docs/tech-debt/README.md` | 下の表のとおり。127 行目(Codex の `ask`)、124 行目(コミットメッセージの guard)、128・129 行目(insight event)、157 行目(shellcheck の対象)を直し、末尾に 3 行足した。V-2〜V-6 |
| `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh` | 書き込み先の説明コメントに 2 行足した。sed fallback ではタブも 2 文字の `\t` のままなので、リダイレクトや `tee` と書き込み先の間のタブは見ない(jq の経路は見る)。コメントだけで、ロジックは変えていない。root と template は `cmp` で一致。V-7 |
| `scripts/verify.local.sh` | 冒頭コメントの test モードの説明を「every tests/test-*.sh(実行権限がないものは working tree でも index でも失敗として数える)」にした。static モードの行はこの PR の S4 で直っていて、tech-debt の plan 参照の検査を含んでいる。V-9 |
| `docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md` | `## Progress checklist` だけ更新した。Review / Verification / Test artifact created を `[x]` にし、verify と test と sync-docs の結果を 3 行のメモにした(V-8 の差はここに書いた)。本文は変えていない |
| `docs/reports/sync-docs-2026-10-07-guard-bypass-and-hygiene.md` | この report |
| `docs/insights/events/2026-10-06-guard-bypass-and-hygiene.jsonl` | `/sync-docs` の insight event を 1 行追記した(`--phase sync_docs --verdict pass --cycle auto`、cycle は 1)。ファイル名の日付は、このスラッグの既存ファイルが UTC の 10-06 で作られているため。この PR が足した手順の最初の実行 |

### `docs/tech-debt/README.md` の変更

| 行 | 変更 |
|----|------|
| 124(コミットメッセージの guard の誤検知) | Why deferred に、2026-10-07 の guard-bypass-and-hygiene も `pre_bash_guard.sh` に触れたが、この行の誤検知は plan の Non-goals に入れて意図的に残した、と足した。Trigger を「コミットメッセージの規則を直す次のタッチ」に書き直し、今回の接触は数えないと書いた。HEREDOC の形が今も deny になることは、新しい guard に payload を流して確かめた。V-4 |
| 127(Codex の `ask`) | debt item に新しい証拠を足した。codex-cli 0.160.0 の PreToolUse の入力スキーマが `permission_mode` を必須とし、値に `bypassPermissions` を含むので、この PR 以降は Codex でも bypass で `ask` を返さない。同じ実行ファイルに `unsupported permissionDecision:ask` の文字列があり、Codex はもともと `ask` を効かせていなかったとみられる(おそらく。未確認)。Codex が bypass 以外のモードで `ask` をどう扱うかと、`bypassPermissions` に当たる Codex の設定は、未確認のまま残した。Trigger は、`ask` の live-fire probe、Codex の `PreToolUse` 配線の変更、`codex exec` での予期しない block / pass の報告にし、古い Trigger(`pre_bash_guard.sh` の次のタッチ)がこの PR で発火したが `ask` は probe していないことを書いた。Related に self-review の L-2 を足した。V-3 |
| 128(HTML コメント)、129(行) | 「`/sync-docs` still writes no insight event」を直した。`/sync-docs` の 4 面に Insight event の節が入り、`tests/test-skill-insight-cycle.sh` が 5 skill × 4 面で見ていることを書いた。129 行目の Why deferred に、`/sync-docs` の手順は追加済みと足した。V-5 |
| 157(`insights-append.sh` の 2 件) | (a) を取り消し線にして RESOLVED とした。shellcheck の一覧が 9 / 36 本だったのを `scripts/*.sh` のグロブにしたこと、露出した SC2209 を直したこと、template 側の `insights-append.sh` は root と同一(`check-sync.sh` が保証)なので root の lint が覆うことを書いた。Impact と Trigger の (a) も取り消し線にした。(b)(cap を上げた `/cross-review` が `cycle-count.json` を増やさない)は開いたまま。Related に plan を足した。V-2、self-review L-5 |
| 158(新) | `pre_bash_guard.sh` が捕まえない書き込みと deny の綴り。(a) `cp`・`mv`・`install`・`sed -i`(M-1 の 5 例は旧 guard が偶然捕まえていた)、`>&`、コマンド置換を含むパス、空白を含む引用符付きのパス、行の継続。(b) `git commit -am "$(id)"`、`-m"$(id)"`、`--message "$(id)"`、`sudo` とタブ。(c) sed fallback でタブの後ろの書き込み先を見ない(L-1)。bypass では deny の規則だけが残るので (b) がそのまま素通りになる、と Impact に書いた。V-6 |
| 159(新) | 長いコマンドで guard が遅い(200 KB で 28 秒、旧 guard の jq の経路も同じ)。V-6 |
| 160(新) | test の Test gaps の要約。(a) `cat >.env<<EOF`、`/usr/bin/tee`、`<` の読み込みが続く形(mutation G8、G9、G14)と、`plan`・`acceptEdits`・`dontAsk` の未テスト。(b) `test-skill-insight-cycle.sh` が `--cycle auto` しか見ない(S2、S3)。(c) `TMPDIR` が git の work tree の中にあると `test-verify-local-hook-tests.sh` の Case 1 が落ちる。(d) マージ後でないと確かめられない 2 点(live の payload の `permission_mode`、`/pr` での実際の archive)。V-6 |

新しい行と 157 行目の plan の参照は `docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md` のままにした。`verify.local.sh` の参照の検査は実在するパスを見るので、いま archive のパスを書くと落ちる。`/pr` の `archive-plan.sh` が移動と同時に README の参照を書き換える(この PR が足した手順)。

## Drift check results

| 文書 | 結果 |
|------|------|
| `.codex/README.md:114-117`(root、template) | 変更なし。「`deny` が実際にコマンドを止める」は、deny がどのモードでも残るので今も正しい。`ask` には触れていない。root と template は `cmp` で一致 |
| `.claude/rules/ralph/git-commit-strategy.md:70`(root、template) | 変更なし。「`pre_bash_guard.sh` blocks dangerous patterns」は deny の規則のことで、変わっていない。HEREDOC の推奨形が guard に deny される問題は、124 行目の既存の行で、この PR の範囲外 |
| `docs/quality/quality-gates.md` | 変更なし。`verify.local.sh` の個々の検査を列挙していない。実行権限の検査は test モード、tech-debt の参照の検査は static モードにあり、「static と test は重ならない」の規則(`:31-37`)は保たれる |
| `docs/quality/definition-of-done.md`(root、template) | 変更なし。guard、`verify.local.sh`、insight event に触れていない |
| `docs/architecture/repo-map.md:61,66` | 変更なし。`archive-plan.sh`、`verify.local.sh`、`insights-append.sh` と `tests/test-*.sh` の glob は載っていて、新しいテスト 2 本の名前を挙げる必要がない |
| `.claude/skills/pr/SKILL.md` step 8(4 面) | 変更なし。S4 で、`archive-plan.sh` が README を書き換えたら同じコミットに入れる、と書いてある(verify AC6 で確認済み) |
| `.claude/skills/sync-docs/SKILL.md`(4 面) | 変更なし。S3 で Insight event の節が入っている。この report の insight event は、その節のコマンドで書いた |
| `internal/org/prompts/implementer.md:27` | 変更なし。`-m "$(` の deny は残っている |
| `AGENTS.md`、`CLAUDE.md`、`README.md`、`.claude/rules/ralph/post-implementation-pipeline.md` | 変更なし。パイプラインの順序と skill の一覧は変わっていない |
| plan の Non-goals(`:51`)が後続候補とした、Go のコメントと fixture に残る `docs/plans/active/` の参照(約 25 か所) | 変更なし。plan の範囲外 |

## 残したもの

- V-8(AC4 の直し方の表示が plan の文言より細かい): plan の本文は digest の対象なので直していない。Progress checklist に、`git add --chmod=+x`(untracked)と `chmod +x`(index だけが 100644)を出すことをメモした。AC4 は満たしている。
- Test gaps 1〜5 のコードとテストの追加: この pass は文書とコメントだけを扱うので足していない。tech-debt の 160 行目に、足す 1 行の形とあわせて記録した。
- self-review L-1 の正規表現の修正(`redirect_lead` と `tee_lead` が 2 文字の `\t` を受け付ける): hook の変更になり、`/self-review` から回し直すことになるので、コメントと tech-debt 158 行目(c)にとどめた。

## 確認

| コマンド | 結果 |
|----------|------|
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | PASS(13 skill) |
| `bash scripts/check-template-purity.sh` | PASS |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(shellcheck、hook の `sh -n`、settings の `jq -e`、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt README の plan 参照。すべて OK) |
| `sh tests/test-pre-bash-guard.sh` | 220 件 PASS、FAIL 0(guard のコメントだけを変えた確認) |
| `./scripts/plan-visual.sh digest <plan>` | `c07adf402bdb`(承認の行と一致) |
| `cmp`(guard、insights README の root と template) | 一致 |

テストの全体(`./scripts/run-test.sh`、`go test`)は、コードを変えていないので再実行していない。test report の結果(shell 39 ファイル 1,943 件、Go 8 パッケージ)が、この commit のコードに対してもそのまま当てはまる。

## Cycle 1 verdict

- Verdict: pass
- Verified: 上の確認表
- Not verified: Codex が `ask` をどう扱うか、live の bypass session で新しい guard が `ask` を返さないこと(どちらも tech-debt の 127 行目と 160 行目に記録)

## Cycle 2

- Date: 2026-10-07
- Agent: doc-maintainer subagent、pipeline cycle 2(`cycle-count.json` は 2)
- Branch: fix/guard-bypass-and-hygiene(編集前の HEAD 79c9664b)
- 前段の記録: self-review cycle 2(C2-M1、C2-L1、C2-L2、C2-L3)、verify cycle 2(V2-1〜V2-4)、test cycle 2(pass、guard のテスト 280 件)、cross-review triage(cycle 1 の ACTION_REQUIRED 2 件)。コードの修正は 0db1a97e(cross-review の 2 件)と ab3ee31c(C2-M1、C2-L1、C2-L3)

### Summary

tech-debt の 158 行目と 160 行目を、cycle 2 の修正後の実測に合わせて書き直した(C2-L2、V2-2、V2-3)。書く前に、旧 guard(2a22ba78)と HEAD の guard を scratchpad に置き、同じ 35 形を jq あり・なしの両方で流して判定を確かめた(140 判定)。guard、テスト、plan の本文は変えていない。plan の digest は `d8f86292d5f1` で、承認の行と一致している(`./scripts/plan-visual.sh digest`)。V2-1(plan の Risks)と V2-4(Progress checklist)は、plan の側で済んでいる(51ce09f7 と `:171`)。

### Files changed

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 158 行目(guard の取りこぼし)と 160 行目(テストの穴)。下の表のとおり |
| `docs/insights/events/2026-10-07-guard-bypass-and-hygiene.jsonl` | `/sync-docs` の event を 1 行追記(`--phase sync_docs --verdict pass --cycle auto`、cycle は 2) |
| `docs/reports/sync-docs-2026-10-07-guard-bypass-and-hygiene.md` | この節。cycle 1 の `## Verdict` は、verify と test の report にならって `## Cycle 1 verdict` に改め、最後に cycle 2 を含む `## Verdict` を置いた |

| 行 | 変更 |
|----|------|
| 158(冒頭) | 「旧 guard が通した中で、偶然 ask を返したのは M-1 の 5 例だけ」という書き方を、「後ろの `>` が部分一致に当たって偶然 ask を返した形(M-1 の 5 例と (a) の `tee` の形)」に直した。「旧 guard」は、マージ後に読んでも指せるよう「この PR より前の guard」と添えた(`origin/main` は書かなかった) |
| 158 (a) | `tee` の書き込み先が、引数の読み取りを止める文字(`&`、`<`、`\`、`;`、`\|`、`)`)の後ろにある形を足した。例は `tee out 2>&1 .env > /dev/null`、`tee < in.txt .env > /dev/null`、`tee out &>/dev/null .env > /dev/null`、`tee $(mktemp) .env > /dev/null`、`tee out\ x .env > /dev/null`。旧 guard は後ろの `>` が当たったときだけ ask を返し、`>` がなければ返さなかった。引用符の中の `;` と `\|`(`tee "a;b" .git/x > /dev/null`)は、旧 guard が jq ありでだけ ask した。0db1a97e と ab3ee31c が、改行のあと(sed 経路)、バックスラッシュとバッククォートのあと、`2>/dev/null` を挟む `tee` を戻したことも書いた |
| 158 (c) | 「`tee` の前のタブは 0db1a97e で見えるようになった。`>` のあと、`tee` のあと、`tee` の引数のあいだのタブは、sed 経路で今も見えない」に絞った。旧 guard は両経路で none だったので、退行ではない。受け入れた 2 件を足した: `tee out.txt 2>/dev/null .env` は ask になる(`.env` に書くので正しい。旧 guard は後ろに `>` がなければ ask しなかった)、`printf "a\\ntee .env"` は両経路で ask になる(書き込みはないので誤検知。旧 guard は jq ありでだけ ask した) |
| 158(Why deferred、Trigger、Related) | 退行ではない形に `tee` の形を加えた。Trigger は「guard-bypass-and-hygiene の次の変更」に書き直した(この PR の `tee_lead` の修正が前のきっかけに当たったため)。(c) の直し方は、`\t` を受け付ける位置を `>` のあと、`tee` のあと、`tee` の引数のあいだの 3 か所にした。Related に verify の cycle 2 と cross-review の triage を足した |
| 160(冒頭) | Test gaps 1〜9 のうち 1〜3 は済んだと書いた |
| 160 (a) | 3 形(`cat >.env<<EOF`、`/usr/bin/tee`、`cat >/tmp/o</repo/.git/HEAD`)を取り消し線にし、0db1a97e で済んだと書いた。テストは `tests/test-pre-bash-guard.sh` の D(`:220`、`:233`)と C(`:198`)で、cycle 2 の `/test` が G8、G9、G14 の red を確かめている。G9 の行に `-a` はないが、mutation は見分ける。残りは `plan`・`acceptEdits`・`dontAsk` と、後ろの `>` で偶然 ask だった `tee` の形がテストにないこと(今の判定は none で、固定されていない) |
| 160(Impact、Trigger、Related) | `tee` の形の判定が変わっても赤くならないことを Impact に、`tee_lead` に次に触れるときにテストへ足すことを Trigger に書いた。Related に test の cycle 2 と verify の Coverage gaps を足した |

158 行目と 160 行目の plan の参照は `docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md` のままにした(`verify.local.sh` の参照の検査は実在するパスを見る。`/pr` の `archive-plan.sh` が移動と同時に書き換える)。表の列数は 157〜160 行目とも同じで、`\|` のエスケープを保っている。

### Drift check results (cycle 2)

| 文書 | 結果 |
|------|------|
| `.codex/README.md:114-117`(root、template) | 変更なし。書いてあるのは `deny` がコマンドを止めることだけで、cycle 2 の修正は `ask` の規則だけを動かした。`cmp` で一致 |
| `pre_bash_guard.sh` のコメント(root、template) | 変更なし。37〜43 行目(sed 経路の `\n` と `\t`)と 52〜61 行目(`tee` の前に置ける文字、`<` で止める理由、`>` で止めない理由)は、probe の判定と食い違わない。33〜36 行目の「each argument of tee」は、引数の読み取りが `;` `&` `\|` `)` `<` とバックスラッシュで止まることを 52〜61 行目が書いているので、直していない(811e1452 から同じ書き方)。`cmp` で一致 |
| `post_edit_verify.sh` のコメント(root、template) | 変更なし。C2-L3 は ab3ee31c で直っている |
| `tests/test-pre-bash-guard.sh` の見出しコメント | 変更なし。C、D、H の列挙は足した行と合う(verify の cycle 2 で確認済み) |
| `docs/quality/`、`AGENTS.md`、`CLAUDE.md`、`README.md`、`.claude/rules/ralph/` | 変更なし。cycle 2 の修正は `tee_lead` の 1 行で、コマンド、契約、パイプラインの順序は動かしていない |
| plan の Progress checklist | 変更なし。cycle 2 の記録は `:171` にあり、本文には触れていない |

### 確認(cycle 2)

| コマンド | 結果 |
|----------|------|
| `./scripts/check-sync.sh` | PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | PASS(13 skill) |
| `bash scripts/check-template-purity.sh` | PASS |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(31 項目が OK。tech-debt README の plan 参照を含む) |
| `./scripts/plan-visual.sh digest <plan>` | `d8f86292d5f1`(承認の行と一致) |
| 旧 guard と HEAD の guard の probe | 35 形 × 2 つの guard × jq あり・なし。158 行目に書いた判定のすべてと一致 |

テスト(`./scripts/run-test.sh`、`go test`)は、コードとテストを変えていないので再実行していない。test の cycle 2 の結果(shell 39 ファイル 2,003 件、Go 8 パッケージ)が、この commit のコードにそのまま当てはまる。

## Cycle 2 (extra run)

- Date: 2026-10-07
- Agent: doc-maintainer subagent。cross-review cycle 2(44ab9ef7)のあと cap を 3 に上げた追加の回(`cycle-count.json` は 2 のまま、insight event の cycle も 2)
- Branch: fix/guard-bypass-and-hygiene(編集前の HEAD 15e5943a)
- 前段の記録: self-review(77e4e542、Merge 可、C3-L1〜C3-L3)、verify(2b4e8864、pass、V3-1〜V3-3)、test(15e5943a、pass、guard のテスト 324 件)、cross-review triage の cycle 2。コードの修正は 3c0ba22a、206d8335、b94a9106

### Summary

V3-1〜V3-3 を直した。書く前に、この PR より前の guard(2a22ba78)と HEAD の guard を scratchpad に置き、記録に使う 10 形を jq あり・なしで流して判定を確かめた(40 判定、`tee "a #b" .env > /dev/null` の旧 guard が jq ありだけ ask する点も含めて verify の表と一致)。guard、plan の本文は変えていない。plan の digest は `d8f86292d5f1` のままで、承認の行と一致している。C3-L1 の `` `pwd` `` の書き込み先は b94a9106 で見えるようになったので、未解決としては書いていない。C3-L3 のテストのコメントは、HEAD で「the command that the assignment prefixes」に直っている(`tests/test-pre-bash-guard.sh:203-205`)。

### Files changed

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 158 行目(V3-1、C3-L1、C3-L2)。(a) の止める文字に `#` とバッククォートを足し、例に `tee a#b .env > /dev/null`、``tee `mktemp` .env > /dev/null``、`tee "a #b" .env > /dev/null`(旧 guard は最後の 1 つを jq ありでだけ ask)を足した。修正の履歴に 3c0ba22a、206d8335、b94a9106(閉じるバッククォートで終わる書き込み先と、語の中のバッククォートを戻したこと、`#` とバッククォートで引数の読み取りを止めたこと)を書いた。`$(...)` で組んだ書き込み先は、引用符つきなら旧 guard も見なかったが、引用符なしで後ろにリダイレクトが付く `echo x > $(pwd)/.env 2>&1` は旧 guard が(後ろの `>` に当たって)ask していたので、冒頭の「偶然 ask を返した形」の列挙と Why deferred に足した。受け入れた edge case を 2 件から 3 件にし、`tee "build .env.log"`(引用符の中の空白で引数の読み取りが終わる誤検知、cycle 2 の cross-review の 2 件目)を足した。Related に self-review の C3-L1・C3-L2 と verify の V3-1 を足した |
| `tests/test-pre-bash-guard.sh` | 見出しコメントの D に、語の中のバッククォート(`` `pwd`/.git/x ``、`` `pwd`/.env ``、``tee `pwd`/.git/x``、b94a9106 の 4 行)を足した(V3-3)。コメントだけ |
| `docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md` | Progress checklist の「Implementation started」の下に、cross-review cycle 2、cap を 3 に上げた判断、3 つの修正、追加の回の self-review・verify・test の結果を 1 項目で足した(V3-2)。本文は変えていない |
| `docs/insights/events/2026-10-07-guard-bypass-and-hygiene.jsonl` | `/sync-docs` の event を 1 行追記(`--phase sync_docs --verdict pass --cycle auto`、cycle は 2) |
| `docs/reports/sync-docs-2026-10-07-guard-bypass-and-hygiene.md` | この節と、末尾の `## Verdict` の更新 |

### Drift check results (extra run)

| 文書 | 結果 |
|------|------|
| `pre_bash_guard.sh` のコメント(root、template) | 変更なし。45〜47 行目(`$(pwd)` は見ない、バッククォートは語の中に置ける)と 58〜68 行目(`tee` の引数の読み取りが `#` とバッククォートで止まる、`tee a#b .env` と ``tee `cmd` .env`` は見ない)は、上の probe の判定と食い違わない。`cmp` で一致 |
| `.codex/README.md`、`post_edit_verify.sh`、`docs/quality/`、`AGENTS.md`、`CLAUDE.md`、`README.md`、`.claude/rules/ralph/` | 変更なし。追加の回の修正は guard の文字クラスと `tee_lead` だけで、コマンド、契約、パイプラインの順序は動かしていない |
| tech-debt 160 行目(Test gaps) | 変更なし。`#`・バッククォート・`$(...)` の形をテストで固定していないことは、test の extra run が「cycle 2 の Test gaps と同じ扱い」と書いており、160 行目の (a) が覆っている |

### 確認(extra run)

| コマンド | 結果 |
|----------|------|
| `./scripts/check-sync.sh` | PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | PASS(13 skill) |
| `bash scripts/check-template-purity.sh` | PASS |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(31 項目が OK。tech-debt README の plan 参照を含む) |
| `bash tests/test-pre-bash-guard.sh` | rc 0(PASS 324、FAIL 0、SKIP 0) |
| `./scripts/plan-visual.sh digest <plan>` | `d8f86292d5f1`(承認の行と一致) |
| 旧 guard(2a22ba78)と HEAD の guard の probe | 10 形 × 2 つの guard × jq あり・なし。158 行目に書いた判定のすべてと一致 |

`./scripts/run-test.sh` と `go test` は、コードを変えていない(テストの見出しコメントだけ)ので再実行していない。test の extra run の結果(shell 39 ファイル 2,047 件、Go 8 パッケージ)がそのまま当てはまる。

## Verdict

- Verdict: pass
- Verified: cycle 1、cycle 2、cycle 2 (extra run) の確認表。cycle 2 は tech-debt の 2 行を、旧 guard と HEAD の guard の probe で確かめてから直した
- Not verified: Codex が `ask` をどう扱うか、live の bypass session で新しい guard が `ask` を返さないこと(どちらも tech-debt の 127 行目と 160 行目に記録)。GNU grep / sed での probe(test の cycle 2 が ubuntu:24.04 でテストを通している)
