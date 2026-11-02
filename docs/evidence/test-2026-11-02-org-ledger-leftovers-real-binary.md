# org-ledger-leftovers の実物の確かめ(AC3: ほかの動詞の案内の `--state-dir`)

- Date: 2026-11-02(JST 13:27〜13:28)
- Plan: `docs/plans/active/2026-11-02-org-ledger-leftovers.md`(Test plan の Evidence、AC3)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Binary: 作業ツリーの HEAD 6c400e87 から `go build -o <S>/ralph ./cmd/ralph`(rc 0、`ralph version` は `ralph dev (unknown unknown)`)
- `<S>` は session の scratchpad の `test-gamma/`(`/private/tmp/claude-502/<project>/<session>/scratchpad/test-gamma`)。下の出力は、その絶対パスを `<S>` に置き換えただけで、ほかは打ったときのまま。

## 環境と偽物

| 項目 | 値 |
|---|---|
| 打った ralph | `<S>/ralph`(PATH の上では `<S>/bin/ralph`、同じファイルへの link)。`/opt/homebrew/bin/ralph`(6.0.0)は PATH に入れていない |
| 環境変数 | `env -i` で空にし、`HOME=<S>/home`、`PATH=<S>/bin:/usr/bin:/bin`、`GIT_CONFIG_GLOBAL=/dev/null`、`GIT_CONFIG_NOSYSTEM=1` と偽物の log の場所だけを渡した |
| git | `<S>/bin/git` は `/opt/homebrew/bin/git`(2.49.0)への link。本物 |
| gh | `<S>/bin/gh` は偽物。呼ばれた引数を `<S>/gh.log` に書き、`gh pr list` には `[]`(PR なし)を返し、ほかは exit 1。本物の gh には届かない(PATH にない) |
| origin | `<S>/origin.git`(bare)。作業用の `<S>/repo` から push したあと、origin の URL を `git@github.com:o/r.git` に変え、repo の `core.sshCommand` を `<S>/fake-ssh` にした。fake-ssh は呼ばれた引数を `<S>/ssh.log` に書き、`git-upload-pack` の要求を `git upload-pack <S>/origin.git` に渡す。ネットワークには出ない |
| 台帳 | 名指す台帳は `<S>/my ledger`(空白を含む)。feat/a を名指す `scope_reserved` を 1 行置いた。既定の台帳 `<S>/repo/.harness/state/org` には feat/default を名指す行を 1 行置いた。中身が違うので、どちらを読んだかが一覧の BRANCH の列でわかる |

origin を GitHub の URL にしたのは、cleanup が origin の URL から GitHub の owner/repo を読めないと、台帳を読む前に断るため(`cleanupReadOrigin`)。`url.<path>.insteadOf` は `git remote get-url` が展開してしまうので、ssh の側を差し替えた。ssh.log の `ssh -G -o SendEnv=GIT_PROTOCOL git@github.com` は git が ssh の種類を調べる呼び出しで、fake-ssh は exit 1 を返し、git はそのまま upload-pack に進んだ。どのケースも本物の gh やネットワークを要しなかったので、打てなかったケースはない。

## 判定

| ケース | 期待 | 結果 |
|---|---|---|
| (a) `ralph org cleanup --branch feat/none --state-dir '<S>/my ledger'` | 拒否の文が名指す `ralph org merge-order --all` に `--state-dir '<S>/my ledger'` が付き、それを写して打つと名指した台帳を読む | rc 1、stdout は空。stderr の文は `ralph org merge-order --all --state-dir '<S>/my ledger' lists the ledger's branches` で終わる。その部分を取り出して `sh -c` でそのまま打つと rc 0 で、一覧は feat/a(org a/a)の 1 行。時刻を伏せて比べると、名指した台帳を直接読んだ一覧と同じで、既定の台帳の一覧(feat/default)とは違う |
| (b) 同じものを `--state-dir` なしで | `--state-dir` を含まない | rc 1、stderr の文は `ralph org merge-order --all lists the ledger's branches`。`--state-dir` は 0 件。写して打つと既定の台帳の一覧(feat/default)と同じ |
| (c) `ralph org merge` を `--branch` なし、`--state-dir` ありで | `ralph org merge-order --state-dir <dir>` を名指す | 空白を含む台帳では `(ralph org merge-order --state-dir '<S>/my ledger' lists the ledger's branches)`。shell が特別に読む文字のないパス `<S>/plain-ledger` では quote なしの `--state-dir <S>/plain-ledger`(そのディレクトリは作られない)。`--state-dir` なしでは `(ralph org merge-order lists the ledger's branches)`。3 つとも rc 1 |
| 台帳 | どの打ち方でも変わらない | 2 つの manifest の sha256 は前後で同じ。どちらのディレクトリにも manifest.jsonl のほかのファイルはない |
| 偽物の呼ばれ方 | 本物の gh とネットワークに届かない | 偽の gh は 6 回(merge-order の 4 回は `-R` なし、cleanup の 2 回は `-R github.com/o/r`)、偽の ssh の upload-pack は 7 回(準備の ls-remote 1 回と、ralph の ls-remote 6 回)。merge の 3 回は gh も ssh も呼んでいない |

## 打ったスクリプト

`<S>/real.sh`。下の出力はこれを `sh <S>/real.sh` で 1 回打ったもの(rc 0)。

```sh
#!/bin/sh
# Real-binary check of org-ledger-leftovers AC3 (cases a, b, c). Everything it
# creates is under $S. It runs the ralph built from the worktree ($S/ralph,
# also $S/bin/ralph) under env -i with HOME=$S/home, git's global and system
# config off, and PATH=$S/bin:/usr/bin:/bin ($S/bin holds the fake gh, a link
# to the real git, and the built ralph).
S=<S>
R="$S/repo"
LEDGER="$S/my ledger"
DEFAULT="$R/.harness/state/org"

E() {
  env -i HOME="$S/home" PATH="$S/bin:/usr/bin:/bin" GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 \
    FAKE_GH_LOG="$S/gh.log" FAKE_SSH_LOG="$S/ssh.log" FAKE_SSH_BARE="$S/origin.git" "$@"
}
step() { printf '\n$ %s\n' "$*"; }
norm() { sed -E 's/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?(Z|[+-][0-9]{2}:[0-9]{2})/<time>/g'; }
# the ralph org merge-order command an error names: from its first
# "ralph org merge-order" to the " lists" after it (mergeOrderCommandIn).
cmd_in() { perl -ne 'if (/(ralph org merge-order.*?) lists/) { print "$1\n"; exit }'; }

rm -rf "$R" "$S/origin.git" "$LEDGER" "$S/gh.log" "$S/ssh.log" "$S/out"
mkdir -p "$S/out"

echo "## setup"
step "built ralph: $(E sh -c 'command -v ralph'), $(E ralph version)"
E git init --quiet --bare -b main "$S/origin.git"
E git init --quiet -b main "$R"
E git -C "$R" -c user.name=t -c user.email=t@example.com commit --quiet --allow-empty -m init
E git -C "$R" remote add origin "$S/origin.git"
E git -C "$R" push --quiet origin main
E git -C "$R" update-ref refs/heads/feat/a "$(E git -C "$R" -c user.name=t -c user.email=t@example.com commit-tree -p main -m 'work on feat/a' 'main^{tree}')"
E git -C "$R" push --quiet origin feat/a
E git -C "$R" remote set-url origin git@github.com:o/r.git
E git -C "$R" config core.sshCommand "$S/fake-ssh"
step "git -C repo remote get-url origin"; E git -C "$R" remote get-url origin
step "git -C repo ls-remote origin  (through the fake ssh, which serves origin.git)"; E git -C "$R" ls-remote origin

mkdir -p "$LEDGER" "$DEFAULT"
printf '%s\n' '{"ts":"2026-11-02T04:30:00Z","org_id":"a","seat_id":"","event":"scope_reserved","worktree":"'"$R"'","details":"paths=internal/a/ split=demo feature=a digest=0123456789ab branch=feat/a approval=per-plan"}' > "$LEDGER/manifest.jsonl"
printf '%s\n' '{"ts":"2026-11-02T04:30:00Z","org_id":"z","seat_id":"","event":"scope_reserved","worktree":"'"$R"'","details":"paths=internal/z/ split=demo feature=z digest=0123456789ab branch=feat/default approval=per-plan"}' > "$DEFAULT/manifest.jsonl"
step "cat '$LEDGER/manifest.jsonl'"; cat "$LEDGER/manifest.jsonl"
step "cat repo/.harness/state/org/manifest.jsonl"; cat "$DEFAULT/manifest.jsonl"
before_l=$(shasum -a 256 "$LEDGER/manifest.jsonl" | cut -d' ' -f1)
before_d=$(shasum -a 256 "$DEFAULT/manifest.jsonl" | cut -d' ' -f1)

cd "$R" || exit 1

echo; echo "## reference listings"
step "ralph org merge-order --all --state-dir '$LEDGER'"
E ralph org merge-order --all --state-dir "$LEDGER" > "$S/out/ref-ledger.txt" 2>&1; echo "rc=$?"; cat "$S/out/ref-ledger.txt"
step "ralph org merge-order --all   (the default ledger, repo/.harness/state/org)"
E ralph org merge-order --all > "$S/out/ref-default.txt" 2>&1; echo "rc=$?"; cat "$S/out/ref-default.txt"

echo; echo "## (a) cleanup --branch feat/none --state-dir '<ledger with a space>'"
step "ralph org cleanup --branch feat/none --state-dir '$LEDGER'"
E ralph org cleanup --branch feat/none --state-dir "$LEDGER" > "$S/out/a.stdout" 2> "$S/out/a.stderr"; echo "rc=$?"
echo "stdout: [$(cat "$S/out/a.stdout")]"; echo "stderr:"; cat "$S/out/a.stderr"
cmd_a=$(cmd_in < "$S/out/a.stderr")
step "the printed command: $cmd_a"
step "sh -c <the printed command>   (run as printed; ralph on PATH is $(E sh -c 'command -v ralph'))"
E sh -c "$cmd_a" > "$S/out/a-replay.txt" 2>&1; echo "rc=$?"; cat "$S/out/a-replay.txt"
norm < "$S/out/a-replay.txt" > "$S/out/a-replay.norm"; norm < "$S/out/ref-ledger.txt" > "$S/out/ref-ledger.norm"; norm < "$S/out/ref-default.txt" > "$S/out/ref-default.norm"
step "cmp (times normalized): replay vs the named ledger's listing"; cmp "$S/out/a-replay.norm" "$S/out/ref-ledger.norm" && echo "same"
step "cmp (times normalized): replay vs the default ledger's listing"; cmp -s "$S/out/a-replay.norm" "$S/out/ref-default.norm" && echo "same" || echo "differ"

echo; echo "## (b) the same without --state-dir"
step "ralph org cleanup --branch feat/none"
E ralph org cleanup --branch feat/none > "$S/out/b.stdout" 2> "$S/out/b.stderr"; echo "rc=$?"
echo "stdout: [$(cat "$S/out/b.stdout")]"; echo "stderr:"; cat "$S/out/b.stderr"
step "grep -c -- --state-dir <stderr>"; grep -c -- '--state-dir' "$S/out/b.stderr"
cmd_b=$(cmd_in < "$S/out/b.stderr")
step "the printed command: $cmd_b"
step "sh -c <the printed command>"
E sh -c "$cmd_b" > "$S/out/b-replay.txt" 2>&1; echo "rc=$?"; cat "$S/out/b-replay.txt"
norm < "$S/out/b-replay.txt" > "$S/out/b-replay.norm"
step "cmp (times normalized): replay vs the default ledger's listing"; cmp "$S/out/b-replay.norm" "$S/out/ref-default.norm" && echo "same"

echo; echo "## (c) merge with no --branch"
step "ralph org merge --state-dir '$LEDGER'"
E ralph org merge --state-dir "$LEDGER" > "$S/out/c.stdout" 2> "$S/out/c.stderr"; echo "rc=$?"
echo "stdout: [$(cat "$S/out/c.stdout")]"; echo "stderr:"; cat "$S/out/c.stderr"
step "ralph org merge --state-dir '$S/plain-ledger'   (a path with no character a shell reads; never created)"
E ralph org merge --state-dir "$S/plain-ledger" > "$S/out/c2.stdout" 2> "$S/out/c2.stderr"; echo "rc=$?"
echo "stdout: [$(cat "$S/out/c2.stdout")]"; echo "stderr:"; cat "$S/out/c2.stderr"
step "test -e '$S/plain-ledger'"; test -e "$S/plain-ledger" && echo "exists" || echo "absent"
step "ralph org merge   (no --state-dir)"
E ralph org merge > "$S/out/c3.stdout" 2> "$S/out/c3.stderr"; echo "rc=$?"
echo "stdout: [$(cat "$S/out/c3.stdout")]"; echo "stderr:"; cat "$S/out/c3.stderr"

echo; echo "## after"
step "shasum of the two manifests, before and after"
echo "named ledger:   $before_l $(shasum -a 256 "$LEDGER/manifest.jsonl" | cut -d' ' -f1)"
echo "default ledger: $before_d $(shasum -a 256 "$DEFAULT/manifest.jsonl" | cut -d' ' -f1)"
step "/bin/ls -A '$LEDGER' repo/.harness/state/org"; /bin/ls -A "$LEDGER"; /bin/ls -A "$DEFAULT"
step "cat gh.log   (the fake gh's calls)"; cat "$S/gh.log" 2>/dev/null || echo "(no calls)"
step "cat ssh.log   (the fake ssh's calls)"; cat "$S/ssh.log" 2>/dev/null || echo "(no calls)"
step "git -C repo status --porcelain --ignored"; E git -C "$R" status --porcelain --ignored
```

偽の gh(`<S>/bin/gh`):

```sh
#!/bin/sh
# fake gh: logs each call; `gh pr list` prints no PR, everything else fails.
printf 'gh %s\n' "$*" >> "$FAKE_GH_LOG"
case "$1 $2" in
  "pr list") printf '[]\n'; exit 0 ;;
esac
exit 1
```

偽の ssh(`<S>/fake-ssh`):

```sh
#!/bin/sh
# fake ssh for git@github.com:o/r.git: logs the call and serves the local bare
# origin's upload-pack instead of contacting any host.
printf 'ssh %s\n' "$*" >> "$FAKE_SSH_LOG"
for last; do :; done
case "$last" in
  git-upload-pack*) exec git upload-pack "$FAKE_SSH_BARE" ;;
esac
echo "fake-ssh: refused: $last" >&2
exit 1
```

## 出力

```text
## setup

$ built ralph: <S>/bin/ralph, ralph dev (unknown unknown)

$ git -C repo remote get-url origin
git@github.com:o/r.git

$ git -C repo ls-remote origin  (through the fake ssh, which serves origin.git)
30773ed22851473fb2747fc9d8f3848dfc36bcb2	HEAD
6f074d8f78b6cab4fec338b822db5691f42b93d4	refs/heads/feat/a
30773ed22851473fb2747fc9d8f3848dfc36bcb2	refs/heads/main

$ cat '<S>/my ledger/manifest.jsonl'
{"ts":"2026-11-02T04:30:00Z","org_id":"a","seat_id":"","event":"scope_reserved","worktree":"<S>/repo","details":"paths=internal/a/ split=demo feature=a digest=0123456789ab branch=feat/a approval=per-plan"}

$ cat repo/.harness/state/org/manifest.jsonl
{"ts":"2026-11-02T04:30:00Z","org_id":"z","seat_id":"","event":"scope_reserved","worktree":"<S>/repo","details":"paths=internal/z/ split=demo feature=z digest=0123456789ab branch=feat/default approval=per-plan"}

## reference listings

$ ralph org merge-order --all --state-dir '<S>/my ledger'
rc=0
origin/main 30773ed22851 (listed 2026-11-02T04:27:48Z)
merge guarantee: unknown (not checked: run ralph doctor)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	no_gate	feat/a	-	-	-	a/a	no gate record: run ralph org gate --org-id a in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-11-02T04:27:48Z and is not a guarantee: origin/main can move before a merge

$ ralph org merge-order --all   (the default ledger, repo/.harness/state/org)
rc=0
origin/main 30773ed22851 (listed 2026-11-02T04:27:49Z)
merge guarantee: unknown (not checked: run ralph doctor)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	no_gate	feat/default	-	-	-	z/z	no gate record: run ralph org gate --org-id z in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-11-02T04:27:49Z and is not a guarantee: origin/main can move before a merge

## (a) cleanup --branch feat/none --state-dir '<ledger with a space>'

$ ralph org cleanup --branch feat/none --state-dir '<S>/my ledger'
rc=1
stdout: []
stderr:
org: cleanup: no feature binding or gate record in the ledger names the branch feat/none, so nothing was removed: ralph org merge-order --all --state-dir '<S>/my ledger' lists the ledger's branches

$ the printed command: ralph org merge-order --all --state-dir '<S>/my ledger'

$ sh -c <the printed command>   (run as printed; ralph on PATH is <S>/bin/ralph)
rc=0
origin/main 30773ed22851 (listed 2026-11-02T04:27:55Z)
merge guarantee: unknown (not checked: run ralph doctor)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	no_gate	feat/a	-	-	-	a/a	no gate record: run ralph org gate --org-id a in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-11-02T04:27:55Z and is not a guarantee: origin/main can move before a merge

$ cmp (times normalized): replay vs the named ledger's listing
same

$ cmp (times normalized): replay vs the default ledger's listing
differ

## (b) the same without --state-dir

$ ralph org cleanup --branch feat/none
rc=1
stdout: []
stderr:
org: cleanup: no feature binding or gate record in the ledger names the branch feat/none, so nothing was removed: ralph org merge-order --all lists the ledger's branches

$ grep -c -- --state-dir <stderr>
0

$ the printed command: ralph org merge-order --all

$ sh -c <the printed command>
rc=0
origin/main 30773ed22851 (listed 2026-11-02T04:27:59Z)
merge guarantee: unknown (not checked: run ralph doctor)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	no_gate	feat/default	-	-	-	z/z	no gate record: run ralph org gate --org-id z in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-11-02T04:27:59Z and is not a guarantee: origin/main can move before a merge

$ cmp (times normalized): replay vs the default ledger's listing
same

## (c) merge with no --branch

$ ralph org merge --state-dir '<S>/my ledger'
rc=1
stdout: []
stderr:
org: merge: --branch is required: name the feature branch whose PR to merge (ralph org merge-order --state-dir '<S>/my ledger' lists the ledger's branches)

$ ralph org merge --state-dir '<S>/plain-ledger'   (a path with no character a shell reads; never created)
rc=1
stdout: []
stderr:
org: merge: --branch is required: name the feature branch whose PR to merge (ralph org merge-order --state-dir <S>/plain-ledger lists the ledger's branches)

$ test -e '<S>/plain-ledger'
absent

$ ralph org merge   (no --state-dir)
rc=1
stdout: []
stderr:
org: merge: --branch is required: name the feature branch whose PR to merge (ralph org merge-order lists the ledger's branches)

## after

$ shasum of the two manifests, before and after
named ledger:   15e87fe0ba9cd078633fc7990e300bd540705e7c218e47125fa7cd72d23be1b1 15e87fe0ba9cd078633fc7990e300bd540705e7c218e47125fa7cd72d23be1b1
default ledger: 3fca89fa831276ba8ccebd8f66ccf9dd9796f1f2ba89753c91fdce142fe4ee6f 3fca89fa831276ba8ccebd8f66ccf9dd9796f1f2ba89753c91fdce142fe4ee6f

$ /bin/ls -A '<S>/my ledger' repo/.harness/state/org
manifest.jsonl
manifest.jsonl

$ cat gh.log   (the fake gh's calls)
gh pr list --head feat/a --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
gh pr list --head feat/default --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
gh pr list --head feat/a --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository -R github.com/o/r
gh pr list --head feat/a --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
gh pr list --head feat/default --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository -R github.com/o/r
gh pr list --head feat/default --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository

$ cat ssh.log   (the fake ssh's calls)
ssh -G -o SendEnv=GIT_PROTOCOL git@github.com
ssh git@github.com git-upload-pack 'o/r.git'
ssh -G -o SendEnv=GIT_PROTOCOL git@github.com
ssh git@github.com git-upload-pack 'o/r.git'
ssh -G -o SendEnv=GIT_PROTOCOL git@github.com
ssh git@github.com git-upload-pack 'o/r.git'
ssh -G -o SendEnv=GIT_PROTOCOL git@github.com
ssh git@github.com git-upload-pack 'o/r.git'
ssh -G -o SendEnv=GIT_PROTOCOL git@github.com
ssh git@github.com git-upload-pack 'o/r.git'
ssh -G -o SendEnv=GIT_PROTOCOL git@github.com
ssh git@github.com git-upload-pack 'o/r.git'
ssh -G -o SendEnv=GIT_PROTOCOL git@github.com
ssh git@github.com git-upload-pack 'o/r.git'

$ git -C repo status --porcelain --ignored
?? .harness/
```

最後の `?? .harness/` は、既定の台帳を置くために作った `<S>/repo/.harness/` で、ralph が作ったものではない。

## 後片付け

scratchpad の `test-gamma/` の中だけを使い、その外のファイルは作っても消してもいない。main checkout と `~/` には触れていない。scratchpad の中身はコミットしていない。
