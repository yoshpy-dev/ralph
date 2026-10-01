# Walkthrough: check-template-scaffold-green (#189)

- Date: 2026-10-01
- Plan: docs/plans/active/2026-09-30-check-template-scaffold-green.md
- Branch: fix/check-template-scaffold-green(base main d754bcd)
- Diff: plan・報告・insight を除くと 6 files、+731 / -37。うち 589 行が `tests/test-check-template.sh`

## 何を直したか

`scripts/check-template.sh` には不具合が 2 つあった。

- **settings の hook の参照の検査が fail-open だった:** settings の `command` は `./.claude/hooks/ralph-dispatch.sh PreToolUse` のように引数を含む。検査はこの文字列全体をパスとして `[ -f ]` にかけていたので、存在する dispatcher を 7 行の FAIL として誤って報告していた。しかも `| while` の subshell の中で `fail` を呼んでいたため、`status=1` が親の shell に伝わらず exit 0 になっていた。本当に hook がなくなっても、検査は通ってしまう。
- **scaffold された project の PR CI が最初の PR から赤になった:** `required_files` に、`ralph init` が配らない 3 項目(`README.md`、`docs/research/approach-comparison.md`、`docs/roadmap/harness-maturity-model.md`)が入っていた。scaffold 側の `templates/base/.github/workflows/verify.yml` の "Check template structure" は、このため exit 1 になっていた。

この PR の変更点は次の 4 つ。

- hook のパスとして、コマンドの最初の語だけを見るようにした。
- 4 つの検査ループを、subshell を作らない形に書き直した。
- `required_files` を 25 項目に減らした。
- 読めない入力があったら、黙って飛ばさずに FAIL として報告するようにした。

## 読む順

1. `scripts/check-template.sh`(template と byte 一致)
   - 冒頭: `mktemp -d` で一時ディレクトリを作る。EXIT trap で消し、INT / TERM / HUP は 130 / 143 / 129 で exit して EXIT trap を通す。
   - `required_files`: 25 項目。配らないファイルを載せない理由をコメントに書いた。issue 番号や meta-repo という語は使っていない(配布先では別の project の issue を指すため)。
   - 実行属性の検査: 存在する探索先(`.claude/hooks`、`packs`、`scripts`)だけを `find` に渡す。`.claude/hooks/local` は `-prune` で木ごと除く。`-not -path` で結果から除くだけだと、find はその下まで降りる。gitignore されたユーザーの領域に読めないディレクトリがあると、そこで find が非 0 になる(cross-review の AR-1)。`find` が非 0 で終わったら「Could not list scripts under ...」で FAIL にする。結果は一時ファイルに書き、`while IFS= read -r` で読む。空白を含むパスも 1 つのパスとして扱われる。
   - SKILL.md の検査と agent の frontmatter の検査: 同じ形。探索先のディレクトリがなければ、何も出さずに飛ばす。
   - settings の検査: `grep -o` の rc が 2 以上(読めない)なら FAIL にする。1(該当なし)は正常として扱う。`tr -d '"' | cut -d ' ' -f 1 | sort -u` で最初の語だけを取り、重複を除く。dispatcher がなくなったときの FAIL は 1 行になる。
2. `tests/test-check-template.sh`: 54 件。
   - A〜C: 一覧が golden と一致する、fixture から 1 項目ずつ消すと落ちる、root に全項目がある。
   - D: root の `CI=true sh` で FAIL がない。
   - E: 引数付きの既存 hook は通る。存在しない hook はパスだけを出して exit 1。settings が読めなければ FAIL。dispatcher が複数の event にまたがって欠けても FAIL は 1 行。
   - F1〜F10: 3 つのループの失敗が exit code に伝わること、空白を含むパス、読めないサブツリーの報告、local の木の prune(F6〜F8)、読めない `.claude/agents/` の下(F9)、`.claude/skills` 自体が読めない場合(F10)。
   - G: `go run ./cmd/ralph init --yes` で作った fresh scaffold で FAIL がない。go がなければ SKIP と数える。
   - H: 探索先がない project でも FAIL がない。
   - I: dash で、stub の find の途中に TERM / INT / HUP を送る。それぞれ exit 143 / 130 / 129 になり、一時ディレクトリが残らないこと。dash がなければ SKIP。
   - root で実行すると権限の検査が効かないので、権限を使うケースは root では SKIP する。
3. `scripts/verify.local.sh`: shellcheck の対象に `scripts/check-template.sh` を加えた。SC2044 のような警告がまた入ったら run-verify が止まる。
4. `internal/scaffold/embed_test.go`: コメントの例を、外した 3 項目から今ある項目に直した。
5. `docs/tech-debt/README.md`: #183 で記録した #189 の行を消した。

## 注意して見てほしいところ

- **exit 1 になる環境が出うる:** これまで exit 0 だった環境のうち、本当に存在しない hook を参照しているか、読めない入力がある場合だけが exit 1 になる。どちらも検知すべき状態として扱った。root と fresh scaffold は green であることを確認した。
- **`.claude/hooks/local` の扱い:** prune するので、local の木の中の `.sh` は実行属性を検査しない。これは以前の `-not -path` と同じ挙動。
- **ケース I はタイミングに頼らない:** stub の find が ready の印を作るまで待ってからシグナルを送る。`set -m` で起動するのは、そうしないと INT が無視された状態で始まるため。呼び出し元が INT を無視している場合は、INT のケースだけ SKIP する。

## 検証

| 項目 | 結果 |
|---|---|
| `bash tests/test-check-template.sh`(macOS、bash 3.2) | 54 / 0 / 0 |
| 同じスイートを ubuntu 24.04 のコンテナで 3 回(bash 5.2、dash、go 1.22、非 root) | 3 回とも 54 / 0 / 0 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` / `run-test.sh` / `run-static-verify.sh` | green |
| root と fresh scaffold で sh・dash・`bash --posix` | exit 0、FAIL なし |
| shellcheck `-S warning`、check-sync(159 IDENTICAL)、purity、cmp | clean / pass |
| mutation(cycle 1 の 8 種、cycle 2 の 9 種、Slice E の 2 種) | すべて該当ケースで落ちる |
| busybox の sh | 未導入で未確認 |

## pipeline の履歴

- plan: Codex advisory の MEDIUM 1 件(既存の 3 つのループも shellcheck SC2044 で止まる)を受け、ユーザーの決定で 3 つのループも直す範囲に入れた。
- cycle 1:
  - Slice A で 2 つの不具合とループを直した。
  - self-review は MEDIUM 1 / LOW 8 で、Slice B で全件を直した(配布されるコメントの issue 番号、読めない入力の報告、FAIL の重複、trap)。
  - verify、test、sync-docs は pass。
  - cross-review は AR-1(local の木を降りてしまう退行)。ユーザーの決定で修正し、cycle 2 として再実行した。
- cycle 2:
  - Slice C で AR-1 を直した(prune)。
  - self-review は LOW 2。Slice D でテストのない分岐を固め、コメントと FAIL の文言を直した。
  - verify は pass。test も pass だったが、塞げる穴が 2 つあると報告があり、Slice E のテストで塞いだ。
  - sync-docs はずれなし。cross-review は指摘 0 件。

## 残る gap

- busybox の sh では確認していない。
- 権限を使うケースは root で実行すると SKIP になる。CI は非 root で動くので、CI ではそれらのケースも実際に走る。
