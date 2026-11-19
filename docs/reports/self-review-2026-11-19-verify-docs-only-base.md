# Self-review report: verify-docs-only-base

- Date: 2026-11-19(cycle 1)
- Plan: `docs/plans/active/2026-11-19-verify-docs-only-base.md`
- Branch: `fix/verify-docs-only-base`、HEAD `a50e4878`、base は `origin/main` の `a8c5e289`(issue #341)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff a8c5e289...HEAD` の 16 ファイル(+2018 −281)の diff の質だけ。コードは S1 `930e6975`(detector)、S2 `ec269cb7`(run-verify.sh)、S2b `672e5ded`(Go のテスト)、文書は S3 `5fe08eff`、実機の記録は `a50e4878`。計画書の記録の commit(`7c0edbf3`、`957bc5a1`、`09e63c43`)は Progress の説明が差分と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify` と `/test` の担当)

## Evidence reviewed

- コード: `scripts/detect-changed-languages.sh` 全 561 行(base の選び方、4 つの listing コマンド、`manifest_paths`、`is_scaffold_file`、ループの `full_reason` と `all_docs`、`emit_result`)、`scripts/run-verify.sh` 全 353 行(detector の呼び出し、`scope_docs_only` の正規化、3 つの終わり方、`verifier_hint`、`branch_scan_failed` の注記)
- 写し: `cmp` で `scripts/` と `templates/base/scripts/` の 2 本(detector、run-verify.sh)が一致。`docs/quality/quality-gates.md` の 2 つの写しは diff の外の 3 か所(Must pass の行の表現)で前から違う。この diff が足した 22 行は 2 つで同じ
- テスト(読んだだけで、走らせていない): `tests/test-run-verify-docs-only.sh` 全 529 行、`tests/test-detect-changed-languages.sh` の diff(43〜54 の新しい節、fake git)、`test-run-verify-scope.sh`、`test-run-verify-concurrency.sh`、`test-run-verify-branch-secret-scan.sh`、`test-ralph-dispatch.sh` の diff、`internal/org/gate_runverify_test.go` の diff と周辺
- 文書: `docs/quality/quality-gates.md` の diff、README の 1 文、`docs/evidence/verify-docs-only-live-2026-11-19.md`
- 周辺: `docs_only` を読むのは detector と run-verify.sh の 2 つだけ(`grep`)。`ralph pack add` は `internal/cli/pack.go:26` にある。`fetch-depth: 0` は root と template の `verify.yml` の両方にある(`quality-gates.md` の文と合う)。旧メッセージ `appears to be docs or scaffold-level` を読む場所は、過去の report 1 つのほかにない
- 自分で走らせた probe(scratch は `$TMPDIR` 配下、repo の外): dash で detector を、非 ASCII の docs と scaffold の path、既存の ref に変な文字を含む `GITHUB_BASE_REF`、ref のない同じ値の 3 つで

## Findings

この run の指摘は L1 から振る。どれも LOW で、CRITICAL、HIGH、MEDIUM はない。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | `docs/tech-debt/README.md:237`、`scripts/detect-changed-languages.sh:230` | L1. register の行 237 は「script が、検査を狭める新しい変数を読み始めたとき」を見直しの引き金にしている。この diff で `ralph org gate` が走らせる run-verify.sh の full scope が detector を呼び、detector が `GITHUB_BASE_REF` を最初に読む。行 237 の対象は `secret-scan-branch.sh` だけで、`gateScriptEnv` は `RALPH_VERIFY_` と `HARNESS_VERIFY_` だけを落とす。pack のない repository で、呼び出し側の環境に `GITHUB_BASE_REF` があると、docs-only の判定の基準が変わる(基準が HEAD 以上なら `no_changes` で `gate_passed` になりうる)。座席の環境にこの変数があることは少ないので LOW。Go のテストは fixture の環境から落としたが(`gate_runverify_test.go:85`)、本物の gate は落とさない | 行 237 の本文と引き金、`internal/org/gate_run.go:67`、detector の header(49-54)。diff は register も `gateScriptEnv` も触っていない | 行 237 の変数の例に detector を足す(`/sync-docs`)。gate で `GITHUB_BASE_REF` を断る必要があるかは、次の人が判断できるよう行に残す。コードは変えなくてよい |
| LOW | `scripts/detect-changed-languages.sh:318-321`、`:331-341`、`:496-513` | L2. 4 つの listing コマンドは `core.quotePath` の既定のまま動くので、非 ASCII の path は `"\346\227\245...md"` のように引用符つきで出る。`docs/*` も `*.md` も引用符つきの文字列に合わず、manifest の key(`[files.'日本語.sh']`)にも合わない。probe(dash): `docs/日本語.md` と、manifest に載せた `日本語.sh` を足した commit で `docs_only=false`、`first_non_docs="\346\227\245\346\234\254\350\252\236.sh"`。docs 側の挙動は旧 run-verify.sh の表も同じだった(引用符つきの文字列に当てていた)ので後退ではない。ただしこの diff は「docs だけの変更は exit 0」を約束し、manifest の照合と `[ -e "$repo_root/$file" ]` を新しく足した。どちらも引用符つきの名前を渡される。fail-closed なので exit 2 に倒れ、出る名前が読めないだけ | 上の probe の出力。`git diff --name-only main HEAD` の生の出力が引用符つき | 4 つのコマンドに `git -c core.quotePath=false` を付ける(`"`、`\`、制御文字は引用符つきのまま残り、fail-closed のまま)。`tests/test-detect-changed-languages.sh:1154-1170` と `tests/test-run-verify-docs-only.sh:463-477` の fake git は `$*` 全体に当てるので、パターンに `-c core.quotePath=false` を足す必要がある。足さないなら、header に「非 ASCII の名前は code 扱い」と書く |
| LOW | `scripts/detect-changed-languages.sh:234-248` と `:307-310`、header の 49-54 行 | L3. `GITHUB_BASE_REF` の値を出力に出さない工夫(`base_label`)は、ref がない場合の `no_base_ref:` だけにかかっている。ref があって merge-base がない場合は `no_merge_base:$base_ref` に生の値が出る。header の「such a value never reaches the output」は後者に当てはまらない。probe: origin の ref `rel+v1#x`(orphan)で `reason=no_merge_base:refs/remotes/origin/rel+v1#x`、ref のない `rel+v2#x` で `no_base_ref:GITHUB_BASE_REF`。実在する ref の名前には改行も制御文字も入らず、出力は `printf '%s'` なので害はない。守る相手のいない対策が半分だけ付いている形 | 上の probe の出力、`test-detect-changed-languages.sh` の `bad name;x` の case(ref として存在しえない値) | 両方の出力で `base_label` を使うか、whitelist をやめて header の 1 文も消す。残すなら、header を「ref がない場合」に絞る |
| LOW | `scripts/detect-changed-languages.sh:560` | L4. 最後の枝だけ `docs_only` に文字の `true` を渡し、ほかの 2 つの枝(556、558)は `"$all_docs"` を渡す。この枝に来るのは、全 path が `is_docs_file` を通ったときだけなので値は合っている(scaffold で docs でない path は `unclassified` に落ちて 556 に行く)。ただしその理由はループの構造にあり、どこにも書かれていない。`is_docs_file` と未分類の判定の間に処理を足すと、黙って `true` になる | 520-553 のループと 555-561 | `"$all_docs"` にそろえる(1 語) |
| LOW | `scripts/run-verify.sh:102-114`、`scripts/detect-changed-languages.sh:255`、`internal/org/gate_runverify_test.go:159-163` | L5. full scope で detector を毎回呼ぶが、結果の `docs_only` を読むのは `ran_any` が 0 のときだけ(`:250`)。pack のある repository の full run でも、git を数回と `ralph config get verify.base` の 1 回を余計に走らせる。`[verify].base` が読めない設定では、detector が stderr に「taking the full scope, not the automatic base」(255)を出す。full scope の run では「切り替える先」がなく、文面と合わない。plan の R5 は「pack のある repository は影響しない」と書くが、この 1 行は出る。Go のテストの comment は base を「docs-only を見分けるためだけに読む」と言うが、そのテストでは Go の verifier が走っていて、それでも読んでいる | run-verify.sh の 102-114 と 250。gate のテストの呼び出し列 `verify.base:off` が、verifier が走った run にも入っている | full scope では `ran_any -eq 0` の枝の中で detector を呼ぶ。今の形を残すなら、run-verify.sh の comment(90-96)に「verifier が走る run でも呼ぶ」と書き、Go のテストの comment の「only」を直す |
| LOW | `scripts/run-verify.sh:110`、`:113`、`:276-277` | L6. detector が失敗または不在のときの reason は `changed_detector_failed` と `changed_detector_missing` で、名前が changed scope を指す。この diff で full scope にも出るようになり、`RALPH_VERIFY_SCOPE=full` の run が「could not be read (changed_detector_missing)」と言う(`tests/test-run-verify-docs-only.sh:498` が full でも固定している)。また 277 行の「The changes are read against GITHUB_BASE_REF in CI …」は `unknown` のすべての reason で出る。`diff_failed`、`config_error:verify.base`、detector の失敗では base の話が的外れで、どこを直すかを読み手がたどれない | 276-277 と、AC5 の pin(`test-run-verify-docs-only.sh:412`) | 277 行の文は base に関わる reason(`no_base_ref*`、`no_merge_base*`、`no_diff_base`、`no_remote_default*`)のときだけ出す。reason の名前を `detector_failed` などに替えるなら、pin 2 か所と plan の言葉も一緒に |
| LOW | `tests/test-run-verify-docs-only.sh:51`、`scripts/detect-changed-languages.sh:314` | L7. 新しい test が `trap cleanup EXIT HUP INT TERM` の形(この test を含め 11 本の suite が同じ)を写している。`cleanup` は終わらないので、Ctrl-C や TERM のあと workdir を消した状態で走り続ける。detector の trap(314)も `rm -f` から `rm -rf "$tmp_dir"` に書き換えられた同じ形で、signal のあとスクリプトが続くと次の redirect が失敗して status 1 になる(旧版も `done < "$tmp_files"` で同じ)。どちらも前からの形 | `grep -l '^trap cleanup EXIT HUP INT TERM' tests/*.sh` が 11 本。正しい形は `tests/test-agent-models.sh:62-63`、`tests/test-archive-plan.sh:39-40`、`tests/test-plan-visual.sh` の `trap 'exit 1' HUP INT TERM` | 新しい test には 2 つめの `trap 'exit 1' HUP INT TERM` を足す。ほかの suite と detector は、この PR で直さず、repo 全体の見直しで(register に既存の話があれば行を足す) |
| LOW | `tests/test-detect-changed-languages.sh:1154-1170`、`tests/test-run-verify-docs-only.sh:463-477`、`:872` と `:180`、ファイルの長さ | L8. (a) 失敗を注入する fake git が 2 つの test に別々に書かれ(片方は呼び出しを log に書く)、同じ名前 `write_manifest` が違う引数の形(前者は table の見出し行、後者は path)で 2 か所にある。`tests/lib/` には `fake-ralph.sh` だけがある。(b) `test-detect-changed-languages.sh` は 848 行から 1294 行に増え、800 行の目安を 500 行近く超える。新しい 43〜54 の節(約 430 行)はそのまま別ファイルに移せる(新しい docs-only の test がその形をすでに取っている) | `wc -l`: 1294 と 529。plan の Progress は「800 行の行に書く」と言うが、diff は register を触っていない | `tests/lib/fake-git.sh` に fake git を移し、2 つの test から読む。43〜54 を `tests/test-detect-changed-languages-docs-only.sh` に移す。急がないなら register に 1 行 |
| LOW | `docs/tech-debt/README.md:232` | L9. 行 232 は「org-gate S3 から、exit code、scope、docs-only の印が run ごとの `mktemp` の file を通る」と書く。この diff が `run_non_docs_file`(印)を消したので、文が前の形のまま残る。もう 1 つ、plan の Progress は 800 行の増加を register に書くと言っていて、diff に入っていない(上の項目) | 行 232 と、`run-verify.sh` の差分(`run_non_docs_file` の削除) | `/sync-docs` で行 232 の「and the docs-only marker」を直し、検査は `tests/test-run-verify-concurrency.sh` の Case 5 に移ったと添える |

## Positive notes

- 3 値の `docs_only` を、読めなかった理由の全部(git なし、repository なし、設定の失敗、ref なし、base なし、merge-base なし、listing の失敗)で `unknown` にしている。`unknown` が `true` に倒れる経路はない。run-verify.sh 側も、detector の不在、失敗、値の違反を `unknown` に寄せ、`case` で正規化している(115-118)
- 4 つの listing コマンドから `|| true` を外し、それぞれを `if ! ... ||` で確かめている(318-321)。1 つが失敗して、ほかが docs の path だけを出していても、一部の list が `true` にならない。tmp の file は 1 つの `tmp_dir` に集めて `rm -rf` で消し、テストが残りを調べる(`assert_dir_empty`)
- docs の表を detector の 1 つだけにして、run-verify.sh の旧い表(`.claude/*` を含む)を消した。ずれる元が構造から消えている
- 出力の値は全部 `printf '%s'` で、`first_non_docs` は git が改行を引用符で包むので 1 行に収まる。`grep -Fqx --` で path の先頭が `-` でも読まれない。manifest の key の 3 つの形と backslash を含む key の扱いを test(45、46)が固定している
- 削除された scaffold の path を merge-base の manifest でも探す(`ralph upgrade` の削除の形、AC12)。存在する path は今の manifest だけを見る、という区別を test 53 が 3 つの向きで固定している
- 全 test が `GITHUB_BASE_REF` を `unset` し、新しい test は case ごとに足す。CI で `main` が入っている環境と手元で結果が変わらない。Go のテストも、fixture の環境から落とす先を 1 か所(`gateRalphStubEnv`)にまとめている
- 実機の記録(`docs/evidence/verify-docs-only-live-2026-11-19.md`)が、`git 2.49` の fetch が `origin/HEAD` を書くこと、`actions/checkout` は書かないと仮定していること、それを `symbolic-ref --delete` で合わせたことを書いている

## Coverage gaps

- テスト、shellcheck、`bash -n`、`check-sync.sh` は走らせていない(`/verify` と `/test` の担当)。plan の S1〜S3 の記録によると dash と ubuntu:24.04 で通っている
- dash で動くことは、probe した 3 つの入力の範囲でだけ見た。detector の trap が signal のあと続く挙動は、repo の既知の形として扱い、この run では probe していない
- 文書のずれは対象外。気づいた点だけ: `[verify].base` を「changed scope が言語を選ぶための revision」とだけ言う文が、README では直ったが、`internal/config/config.go:135-142`(`VerifyConfig.Base` の comment)と `templates/base/ralph.toml:99-105` にはまだある。後者は `defaults_sync` のテストが comment を固定しているので、直すときはテストの期待も見る(`/sync-docs`)
- 実際の CI(`actions/checkout`)での動きは見ていない。`origin/HEAD` を書かないという仮定は plan の Assumptions にあり、`GITHUB_BASE_REF` を先に読むのでその仮定に依らない

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 9 件のうち、コードの直しが要るものはない)
- Follow-ups: L1(register の行 237 に detector を足す)と L9(行 232 と 800 行の記録)は `/sync-docs` で。L2(`core.quotePath=false`)は、fake git のパターンの直しを含めて 1 つの小さな commit で足せる。L3、L4、L6 は detector と run-verify.sh の言い回しの直しで、まとめて 1 commit にできる。L5 は `ran_any` の枝へ移すか、comment で今の形を認めるかを選ぶ。L7 と L8 は test の整理で、急がない

## Fixes after this review

Applied by the orchestrator in commit 1c3dc457 (implementer slice S4), before
`/verify`:

- L2: the four listing commands run with `git -c core.quotePath=false`, so a
  non-ASCII path (`docs/日本語.md`, a manifest-listed `scripts/日本語.sh`) is
  read raw; a path with a control character, `"`, or `\` stays quoted and
  non-docs. New detector case 55; the git stand-in patterns were updated.
- L3: a base from `GITHUB_BASE_REF` gives `no_merge_base:refs/remotes/origin/<label>`
  (or `refs/heads/<label>`) with the same label as `no_base_ref`.
- L4: the last `emit_result` passes `"$all_docs"`.
- L7: the detector's HUP, INT, and TERM traps exit with 129, 130, and 143
  after the cleanup, as `scripts/run-verify.sh` does.

Left as they are: L5 (the full scope runs the detector even when a verifier
runs; one detector run, and a settings error prints the detector's line) and
L6 (reason names and the generic base hint) are recorded as known gaps; L1,
L8, and L9 go to `/sync-docs` (`docs/tech-debt/README.md`).
