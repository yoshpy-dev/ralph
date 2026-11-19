# Cross-review triage report: org-ssh-alias-host

- Date: 2026-11-20
- Plan: docs/plans/active/2026-11-19-org-ssh-alias-host.md
- Base branch: main (32d6dc51; merge base 32d6dc51)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/3 (the third pipeline run; the cap was raised to 3 for this task after the second cross-review)
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-19-org-ssh-alias-host.md(承認 f4a9803ca80e)
- Self-review report: docs/reports/self-review-2026-11-19-org-ssh-alias-host.md(run 3、Merge: yes、LOW 9。L13・L15 は 74fef538 で直し、L14 は tech-debt に残した)
- Verify report: docs/reports/verify-2026-11-19-org-ssh-alias-host.md(run 3、pass)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`)の答えは指摘なし。原文は「No actionable regressions were found in origin resolution, caller integration, caching, or cancellation handling. Diff checks, Go formatting, and changed skill-body parity passed. Focused Go tests could not start because the read-only sandbox blocked temporary build-directory creation.」。codex はテストを打てていない。テストの根拠は run 3 の /test の報告(run-test.sh が rc 0、絞った run の 114 の subtest が PASS)。cycle 2 の ACTION_REQUIRED #1(ssh -G に小文字の host を渡していた)は 664d4771 と 04161b43 で直り、cycle 1 の ACTION_REQUIRED #1(doctor の Ctrl-C で ssh が残る)は 240324fe で直っている

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 2(履歴)

cycle 2 の triage をそのまま残す。ACTION_REQUIRED #1(ssh -G に小文字の host を渡していた)は 664d4771 と 04161b43 で直した。


- Date: 2026-11-20
- Plan: docs/plans/active/2026-11-19-org-ssh-alias-host.md
- Base branch: main (32d6dc51; merge base 32d6dc51, after the merge of origin/main into the branch at e6a98fad)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached; raised to 3 for this task, cross-review step 9 Option 1)
- Total reviewer findings: 1
- Cycle 2 after triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context

- Active plan: docs/plans/active/2026-11-19-org-ssh-alias-host.md(承認 f4a9803ca80e)
- Self-review report: docs/reports/self-review-2026-11-19-org-ssh-alias-host.md(cycle 2、Merge: yes、LOW 6)
- Verify report: docs/reports/verify-2026-11-19-org-ssh-alias-host.md(cycle 2、pass)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`)の指摘は 1 件。codex は分離した OpenSSH の設定で再現したと書き、Go のテストは read-only の sandbox で打てなかった。triage の前に手元の OpenSSH_10.2p1 で、`-F` で分離した設定を使って確かめた。`Host github.com`(`HostName ghe.example.com`)のとき `ssh -G GitHub.COM` は `hostname github.com`(当たらない)、`ssh -G github.com` は `hostname ghe.example.com`(当たる)。`Host Work-GH`(`HostName github.com`)のとき `ssh -G Work-GH` は `hostname github.com`、`ssh -G work-gh` は `hostname work-gh`(解けない)。OpenSSH の `Host` の照合は大文字小文字を区別し、出す hostname は小文字になる。gh(go-gh の `Translator`)は `u.Hostname()` を大文字小文字のまま `ssh -G` に渡すので、git と gh は origin に書かれたとおりの host で ssh の設定を引く。ralph は `mergeOriginFromURL` が小文字にした host を渡していた(D1 の「gh と同じに解く」に反する)

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `ssh -G` に渡す host は `mergeOriginFromURL` が小文字にしたものだが、OpenSSH は `Host` を大文字小文字を区別して照合する。origin が `git@GitHub.COM:o/r.git` で `Host github.com` に `HostName ghe.example.com` があると、git の `ssh GitHub.COM` は github.com に向かうのに ralph は ghe.example.com を読む。大文字を含む別名も解けないままになる。ssh には origin に書かれたとおりの host を渡し、解いた API の host だけを小文字にする | 手元で再現した(上の Triage context)。この PR が持ち込んだずれで、gh と同じに解く(D1)という計画の約束に反する。直し方は小さく、`resolveMergeAPIHost` が origin の URL から大文字小文字を保った host を取り、`mergeSSHHostName` の確かめと `ssh -G` にはそれを使い、出力の hostname と fallback だけを小文字にする。キャッシュの `host` は小文字のまま(`ReadMergeGuarantee` は大文字小文字を無視して比べる) | `internal/org/mergeorigin.go`(`resolveMergeAPIHost`、183〜186 行)、そのテスト |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe


## Cycle 1(履歴)

cycle 1 の triage をそのまま残す。ACTION_REQUIRED #1(doctor の Ctrl-C で ssh が残る)は 240324fe で直した。


- Date: 2026-11-20
- Plan: docs/plans/active/2026-11-19-org-ssh-alias-host.md
- Base branch: main (32274500; merge base e6d5d0ed, after the merge of origin/main into the branch at 0a71a930)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2 (the first run)
- Total reviewer findings: 1
- Cycle 1 after triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context

- Active plan: docs/plans/active/2026-11-19-org-ssh-alias-host.md(承認 f4a9803ca80e)
- Self-review report: docs/reports/self-review-2026-11-19-org-ssh-alias-host.md(2 回目、Merge: yes。L8・L10 は 3c369ff4 で直し、L4・L6・L7・L9 は残した)
- Verify report: docs/reports/verify-2026-11-19-org-ssh-alias-host.md(2 回目で pass)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`)の指摘は 1 件。codex は read-only の sandbox でビルドのディレクトリを作れず、テストは打てなかったと書いている。テストの根拠は /test の報告(run-test.sh が rc 0)になる。#340 は `ssh -G` を `realMergeDetachedExec`(自分のプロセスグループ)で打つ。計画助言の指摘(期限で止めても `Match exec` の子が残る)を受けた形で、期限と ctx の取り消しではグループごと止まる。ralph org merge・cleanup・regate の CLI は `signal.NotifyContext` の ctx で呼ぶので、Ctrl-C でも ctx が終わり、グループごと止まる。doctor だけが `checkMergeGuarantee(context.Background(), ...)`(`internal/cli/doctor.go:238`)で呼んでいた

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] doctor の Merge guarantee は `context.Background()` で検査を呼ぶ。`ssh -G` が `Match exec` の命令で待っている間に Ctrl-C(SIGINT)や SIGTERM で doctor を止めると、ctx は終わらず、自分のプロセスグループで動く ssh とその子は端末の割り込みも期限の片付けも受けずに残る。doctor から signal を受ける ctx を渡し(Node の検査と同じ形)、CLI の段で割り込みのテストを足す | 本物の穴で、この PR が持ち込んだ。#340 より前は doctor の git と gh が doctor と同じプロセスグループで動き、Ctrl-C で一緒に止まっていた。ssh を自分のグループに移したので、ctx を終わらせる道のない doctor だけが残る。merge・cleanup・regate は `signal.NotifyContext` を使っており当たらない。直し方は既にある形(`doctorNodeSignalContext`、`internal/cli/doctor_node.go:45-53`)をそのまま使える | `internal/cli/doctor.go`(238 行)、`internal/cli/doctor_merge_guarantee.go`、そのテスト |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
