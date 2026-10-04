# org-drop-qa-seat

- Status: In progress
- Owner: Claude Code
- Date: 2026-10-04
- Related request: org runtime の役割を指示役・実装役・レビュー役の 3 つに絞り、qa 座席(役割雛形)を撤去する。qa が担っていた決定論ゲート(`run-static-verify.sh` / `run-test.sh`)の再実行は reviewer の最初の手順に移し、fail なら差分レビューに進まず BLOCKED で返す。spec FR-7 の 4 フェーズの判断を改訂する。あわせて、指示役の名前 `lead` を `leader` に改める(ユーザー: 「あと、leadはleaderに変更してください。それも計画に含めて。」)
- Related issue: N/A
- Type: refactor
- Branch: refactor/org-drop-qa-seat

## Objective

org runtime の座席を、指示役(leader)・実装役(implementer)・レビュー役(reviewer)の 3 つにする。

- 実装者以外がゲートを実行し直すという qa の役目は reviewer が引き継ぐ。fail したコードにはレビューの手間をかけない点も、今と同じに保つ。
- 指示役の識別子を `lead` から `leader` に改める。旧名が残っていたら、黙って別の動作をせず、改名を案内して止める。

## 撤去の理由(2026-10-04 の会話で合意)

- qa 雛形の仕事は決定論的な 2 本のスクリプトを実行して要約するだけで、「推測で結果を上書きしない」とも書いてある。spec の「LLM は判断、決定論は機構と保証」という境界からすると、常駐の LLM 座席を 1 つ使う理由がない。
- implementer はスライスごとに検証を通してからコミットする(`implementer.md`)。qa は同じスクリプトをもう一度実行する。
- FR-7 の順序は impl → qa → reviewer の直列で、並列化による短縮はない。スター型なので、qa の結果は lead を中継して reviewer に渡る。中継が 1 回と、`max_seats` の枠が 1 つ増える。
- テストの出力で reviewer のコンテキストを埋めたくない場合は、座席内 fan-out で安いモデルのサブエージェントに任せられる。雛形はすでにこれを許可している。

## 調査で確認したこと(2026-10-04、main 11602fed)

### qa

- qa を前提に動く Go のコードはない。`RenderRolePrompt` は `prompts/<role>.md` を読むだけで、雛形がなければ `ok=false` を返す(`internal/org/prompts.go`)。spawn は雛形のない役割を「未知の役割」として扱い、`--prompt` だけを初期プロンプトにする。`--prompt` もなければ空のまま起動する(`internal/org/spawn.go:673-693`)。CLI は警告を出さない。
- spawn の入力検証は、先頭の識別子の検証(manifest に書かない単純な拒否、`spawn.go:324-341`)と、その後の envelope / 権限 / scope / 容量の検証(`reject()` で manifest に `rejected` を残す)の 2 段になっている。dry-run も同じ順で検証する。
- ゲートを機械的に強制する仕組みはない。`internal/` と `cmd/` の本番コードに `run-test.sh` / `run-static-verify.sh` の呼び出しはなく、`.claude/hooks/` にも org のゲートはない。spec FR-7 の「ゲートは hook で LLM 迂回不能とする」は未実装である。一方 `docs/quality/quality-gates.md:79` は、この行を「LLM の判断に依存せず決定論的に強制するゲート」の表に載せている。
- ゲートのスクリプトは `docs/reports/` 以外にも書き込む。`run-verify.sh:25` は `.harness/state/`、`.harness/logs/`、`docs/evidence/` を作る。3 つとも gitignore 済み(`.gitignore:45,47,58`)なので `git status` には出ない。
- 権限モード: 既定は全役割 `autonomous`。claude の `guarded` は CLI の対話の既定と同じで、ツールを使うたびに許可を求める。codex の座席は `codex_verified = true` にしない限り `guarded` に固定される(`internal/org/permissions.go`)。`templates/base/ralph.toml:81` の設定例は `reviewer = "guarded"`、`[org.roles]` の例は `reviewer = ["opus", "gpt-6-astra"]`(codex の reviewer を想定)。
- qa への参照(履歴を除く):
  - 雛形: `internal/org/prompts/qa.md`、`lead.md:11`、`reviewer.md:9,15,72`
  - Go のコメント: `internal/org/prompts.go:36`、`internal/org/spawn.go:674`
  - テスト: `internal/org/prompts_test.go:115-140`(qa 雛形の展開)、`:197`(fan-out 節の役割リスト)、`internal/org/spawn_test.go:1222-1237`(雛形 + `--prompt` の連結を qa で検査)、`:1764`(コメント)、`internal/cli/org_test.go:1491`(コメント)、`internal/cli/status_test.go:30-104`(manifest の役割文字列としての `qa`)
  - `/org` skill(4 面: `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、template の 2 つ): 154(Leaded 行)、158、164、172(fan-out の例)、185(役割リスト)、188(「上記 4 役割」)
  - `.claude/rules/ralph/agent-messaging.md:23`(+ template)、`docs/quality/quality-gates.md:79`(+ template の 78)、`README.md:232`
  - spec `docs/specs/2026-08-01-org-runtime.md`: Summary(5)、2026-09-16 改訂 (b)(10)、FR-4(50)、FR-7(53)、AC(74)、Open questions(168)

### lead

- 識別子の値は `internal/org/spawn.go:61` の `const LeadIdentity = "lead"` の 1 か所で決まる。役割名、seat id、agmsg の宛先 ID、herdr のエージェント名(`<org_id>_lead`)、雛形のファイル名(`RenderRolePrompt(role)` が `prompts/<role>.md` を読むので `lead.md`)がこの値から決まる。go:embed のパターンは `prompts/*.md` なので、`leader.md` もそのまま埋め込まれる。
- `lead` を含む Go の識別子(テストを除く): `LeadIdentity`(28)、`ensureLeadJoined`(12)、`historyLeadLineCount`(8)、`LeadDriver`(7)、`HistoryLeadLines`(6)、`LeadAgentGet`(5)、`defaultLeadDriver`(5)、`subjectIsLead`(3)、`filterLeadHistoryLines`(3)。ほかに `leadSelfSpawn`、`leadJoinErr`、`leadHistoryFromField` などの局所変数・関数がある。
- `internal/org/protocol` の typed protocol の検証は lead を固定していない。`verbs.go`(send / stop / disband / read / wait)と `report.go` も lead を特別扱いしない。`ralph org send --to X` は manifest の seat を探すので、改名後の `--to lead` は「seat not found」で失敗する。
- watch(`watch.go`)は識別子の定数を使い、lead の pane の生存確認と agmsg の履歴の送信元の判定をしている。古いバイナリで立てた org を新しいバイナリで watch すると誤判定する。
- herdr のエージェント名は `<org_id>_<seat_id>` で 32 文字が上限(`spawn.go:330-341`)。`leader` にすると、指示役の座席を立てられる org_id の上限が 26 文字から 25 文字になる。
- `ralph.toml` の `[org.roles]`(モデルプールの制限)と `[org.permissions.roles]`(権限モード)は役割名を自由なキーとして持ち、キー名を検証しない(`internal/config/config.go:44-82, 313-330`)。改名後に `lead = "guarded"` が残ると黙って効かなくなり、指示役は既定の `autonomous` で動く。`config.Load` を呼ぶのは `internal/cli/doctor.go:88`(エラーでも続行して報告)と `internal/cli/org.go:227`(org の全動詞)だけ。
- `--lead-driver` は `ralph org spawn` のフラグ(`internal/cli/org.go:327`)。`ralph org start` の help は `lead.md` と `--role lead` に言及している(`org.go:388-397`)。
- 英語の動詞としての "lead" は、単語単位(`-w`)の検索では見つからなかった。`doctor_codex_writable_root.go:516,795` の `leads back` / `leads with` は `-w lead` に掛からない。herdr 名のフィクスチャ `<org>_lead` も `_` が単語の文字なので `-w` に掛からず、go test で拾う。
- 編成パターン名 `Leaded`(Solo / Leaded / Parallel)は各面 23 か所ある。ユーザーの判断で残す。
- 文書: `AGENTS.md` と `.ralph/core/AGENTS.core.md`(+ template。`AGENTS.md` は managed block)、`README.md`、`/org` skill(地の文と description の "Lead's operating manual")、`agent-messaging.md`(`TO: lead` がプロトコルの契約)、`docs/recipes/codex-seat-permissions.md`(+ template)、`quality-gates.md`(+ template)、spec、tech-debt(履歴の行が多い)。

### 配布

- `ralph upgrade` での配布: `docs/` 配下と `ralph.toml` は seed なので、下流のコピーは書き換わらない。`.claude/skills/org/SKILL.md` と `.claude/rules/ralph/agent-messaging.md` は core なので、upgrade で置き換わる(`internal/cli/init.go` の `ownerForScaffoldPath`)。`AGENTS.md` は block で、managed block だけが置き換わる。upgrade は実行中のバイナリに埋め込まれたテンプレートから書くので、下流では core ファイルとバイナリの版がそろう。雛形はバイナリに埋め込まれ、座席は spawn した時点の雛形で動き続ける。

## Scope

### leader への改名(挙動は変えない)

- 識別子の値を `leader` にし、Go の識別子とコメントを改名する(`LeadIdentity` → `LeaderIdentity`、`ensureLeadJoined` → `ensureLeaderJoined` など)。識別子は gopls rename(または同等の型付きの rename)で直し、文字列とコメントは単語単位(`-w`)の置換に限る。置換の後に `go build ./... && go test ./...` を通す。
- `internal/org/prompts/lead.md` を `leader.md` に `git mv` する。埋め込みの雛形すべての `TO: lead` などを `leader` にする。
- CLI の help とエラー文を `leader` にする。`--lead-driver` は `--leader-driver` にする(旧フラグの扱いは次の節)。
- テストは、現行の識別子を表すフィクスチャを `leader` にする。過去のデータを表すフィクスチャ(`internal/insights/testdata/receipts.jsonl` と、それを読む insights のテスト)は `lead` のまま残す。

### 旧名 `lead` の止め方

- spawn で `--role lead` か `--id lead` を受けたら、`--prompt` の有無にかかわらず拒否し、`leader` への改名を案内する。拒否は spawn の先頭(識別子の検証の直後、manifest に書かない単純な拒否)に置く。dry-run と `ralph org start` も同じ経路を通る。
- `ralph.toml` の `[org.roles].lead` か `[org.permissions.roles].lead` があれば、spawn を同じ位置で拒否し、キーの改名を案内する。`config.Load` ではエラーにしない。読むだけの動詞や stop / disband まで止めると、古い org を片付ける前に ralph.toml を直す必要が出るため。`ralph doctor` は同じ条件で warn を出す。
- `--lead-driver` は非推奨の別名として残し、使うと警告を出す(cobra の `MarkDeprecated`)。`--lead-driver` と `--leader-driver` を両方指定して値が違えばエラーにする。どちらが指定されたかは `Flags().Changed()` で判定する。
- stop / disband / send / read / wait / status には旧名の拒否を入れない。古いバイナリで立てた org を、新しいバイナリで片付けられるようにする。

### qa の撤去: 雛形

- `internal/org/prompts/qa.md` を削除する。
- `reviewer.md` を書き換える。
  - 最初に決定論ゲート(leader の指示したコマンド、指示がなければ `./scripts/run-static-verify.sh` と `./scripts/run-test.sh`)を実行する。スクリプトの出力を正とし、推測で上書きしない。
  - ゲートの結果を 2 つに分けて扱う。チェックが落ちたら `GATE: fail` として、root cause(チェック名・ファイル・行)とレポートのパスを付けて BLOCKED を返し、差分レビューに進まない。権限や環境の問題でゲートを実行できなかったら `GATE: unrunnable` として、実行できなかった理由を付けて BLOCKED を返し、こちらも差分レビューに進まない。
  - ゲートが通ったら、差分品質と spec / plan の受け入れ基準をレビューし、ゲートの結果と合わせて所見を出す。RESULT に `GATE: pass` を入れる。
  - 書き込みの範囲を「コードと設定は変更しない。書くのは `docs/reports/` のレポートと、ゲートのスクリプトが自分で作る生成物(`.harness/state/`、`.harness/logs/`、`docs/evidence/`)だけ」にする。
  - 「QA 座席のレポートを読む」記述を外す。fan-out の例を「ゲートの実行を安いモデルの子サブエージェントに任せ、要約とレポートのパスだけ受け取る」に差し替える。
- `leader.md` を書き換える。レビューと検証(ゲートの再実行を含む)は reviewer に委譲する。reviewer の BLOCKED が `GATE: fail` なら implementer に差し戻す。`GATE: unrunnable` なら implementer には戻さず、leader が環境や権限を直してから reviewer にやり直させる(座席の権限モードを変えて spawn し直す、など)。直せなければ人に上げる。

### qa の撤去: spawn

- 撤去・改名した役割の表を 1 か所に置く(`qa` は撤去で移行先 `reviewer`、`lead` は改名で移行先 `leader`)。前の節の旧名の拒否もこの表を使う。
- `--role qa` に `--prompt` がなければ、spawn の先頭で拒否する。エラー文には、qa 雛形が撤去されたこと、ゲートの再実行は reviewer に移ったこと、独自の qa 座席が要るなら `--prompt` を渡すことを書く。dry-run でも同じく拒否する。`--prompt` があれば、従来の未知の役割と同じく `--prompt` だけで起動する。

### Go のコメントとテスト(qa)

- Go のコメント(`prompts.go:36`、`spawn.go:674`、`spawn_test.go:1764`、`org_test.go:1491`)を 3 雛形の前提に直す。
- qa 雛形の展開テストを、`RenderRolePrompt("qa")` が `ok=false` を返す回帰テストに置き換える。
- reviewer と leader の雛形のテストは、ミッション節(`markdownSection` で切り出す)に限って、決めた文言があるかを検査する。存在だけで済ませず、文言ごとに AC-2 / AC-3 の mutation で落ちることを確かめる。
- fan-out 節のテストの役割リストを implementer / reviewer にする。`spawn_test.go:1222` の連結テストを reviewer 雛形で行う。
- spawn のテスト: `--role qa` と空の `--prompt` は拒否され、エラー文に `reviewer` と `--prompt` が含まれ、manifest に何も書かれない。dry-run でも拒否される。`--role qa` に `--prompt` を付けると起動し、初期プロンプトは `--prompt` だけになる。

### 文書(qa と leader の両方)

- `/org` skill(4 面):
  - qa: Leaded 行を reviewer だけにする。fan-out の例と役割リストを 3 役割にする。reviewer の説明に「ゲートを再実行し、通れば差分と仕様適合をレビューする。ゲートを実行するので、`guarded` にすると許可待ちで止まる」を足す。「上記 4 役割」を 3 にする。未知の役割の段落に、qa 雛形は撤去済みで `--role qa` は `--prompt` なしでは拒否されること、E2E や探索的テストのような重い検証は `--prompt` で役割を立てられることを足す。
  - leader: 地の文の "Lead" と識別子の `lead` を "Leader" / `leader` にする(description を含む)。旧名の `lead` は拒否されることを一文で書く。パターン名 `Leaded` は残す。
- `agent-messaging.md`(+ template): `TO: lead` などプロトコルの契約を `leader` にし、seat id の例から `qa` を外す。
- `AGENTS.md` と `.ralph/core/AGENTS.core.md`(+ template)、`README.md`(説明と `--id leader --role leader`、`--to leader` の例、`qa` 座席の削除)、`docs/recipes/codex-seat-permissions.md`(+ template)を `leader` にする。
- `quality-gates.md`(+ template): Quality pipeline gate の行を新しい順序にし、機械的な強制ではなく雛形の指示で動いていることを書く。`ALERT to lead` などを `leader` にする。
- `templates/base/ralph.toml`: `[org.permissions.roles]` の例に、reviewer はゲートを実行するので `guarded` では許可待ちで止まる、と注記する(値の例はそのまま)。コメントの "Lead autonomy" を "Leader autonomy" にする。
- spec に「2026-10-04 改訂」の節を足す。
  - 役割は leader / implementer / reviewer の 3 種(2026-09-16 改訂 (b) を置き換える)。指示役の識別子は `lead` から `leader` に改め、旧名は案内付きで拒否する。
  - FR-7 は 3 段: impl の退出チェック → reviewer(最初にゲートを再実行し、`GATE: fail` なら BLOCKED → leader が impl に差し戻す、`GATE: unrunnable` なら leader が環境を直す)→ leader の裁定。
  - 改訂の理由、`--role qa` の拒否、FR-7 のゲートの hook 強制が未実装であること。
  - FR-4、FR-7、AC(74)の行に改訂の印を付ける。履歴の記述(Summary など)は書き換えない。
- `docs/tech-debt/README.md` に「FR-7 のゲートの hook 強制が未実装」の行を足す。既存の行の `lead` は履歴として残す。

## Non-goals

- ゲートを機械的に強制する仕組み(hook や `ralph org` の動詞)の実装。tech-debt に記録するだけにする。
- 雛形のない役割に `--prompt` もないとき、役割を問わず警告や拒否をする変更。役割が空の spawn は今も正当に使われているので、撤去・改名した役割だけを対象にする。
- `lead` を `leader` の別名として受け付けること(ユーザーの判断で、案内付きで止める)。
- 編成パターン名 `Leaded` の改名(ユーザーの判断)。
- 権限モード(`[org.permissions]`)の既定値の変更。
- 履歴の成果物(`docs/plans/archive/`、`docs/reports/`、`docs/evidence/`、`docs/research/`、`docs/insights/events/`、tech-debt の既存の行、spec の履歴の記述)の書き換え。
- `internal/cli/status_test.go` の `qa`。役割は自由な文字列で、過去の manifest に `role=qa` が残っていても `ralph status` が表示できることを検査しているので、そのまま残す。
- リリース(`/release` は手動)。

## Assumptions

- `--role qa` や `--role lead` を使う下流のスクリプトはほぼない。リポジトリ内に org の実行記録(`docs/reports/org-*`、manifest)はなく、smoke の記録(`docs/evidence/org-*-smoke-2026-08-02.txt`)は dry-run で使っただけである。
- claude の reviewer 座席は既定の `autonomous` でゲートを実行できる。
- org_id が 26 文字ちょうどの org を使っている人はいない(R10)。

## Affected areas

- 雛形: `internal/org/prompts/lead.md` → `leader.md`(改名と書き換え)、`qa.md`(削除)、`reviewer.md`、`implementer.md`(`lead` → `leader`)
- Go: `internal/org/spawn.go`、`watch.go`、`watcher.go`、`verbs.go`、`permissions.go`、`prompts.go`(撤去・改名した役割の表を置く想定)、`envelope_summary.go`、`report.go`、`statedir.go`、`internal/cli/org.go`、`internal/cli/doctor.go`(旧キーの warn)、`internal/cli/doctor_codex_writable_root.go`、`internal/config/config.go`(コメント)
- テスト: `internal/org/*_test.go`、`internal/org/driver/*_test.go`、`internal/cli/*_test.go`、`internal/config/config_test.go`、`internal/insights/insights_test.go`(過去のデータとして残す箇所の確認だけ)
- `/org` skill の 4 面、`agent-messaging.md`(+ template)、`AGENTS.md`、`.ralph/core/AGENTS.core.md`(+ template の 2 つ)、`README.md`、`docs/recipes/codex-seat-permissions.md`(+ template)、`docs/quality/quality-gates.md`(+ template)、`templates/base/ralph.toml`(コメントだけ)、`docs/specs/2026-08-01-org-runtime.md`、`docs/tech-debt/README.md`

## Design decisions

- ゲートの再実行は reviewer の最初の手順にする(依頼で決まっている)。leader が自分でスクリプトを実行する案は、leader のコンテキストをテストの出力で埋めるうえ、雛形の「自分では作業しない」原則と合わないので採らない。機構側で強制する案は Non-goals に回す。
- `--role qa` は `--prompt` がなければ拒否する(Codex plan advisory の HIGH)。古い skill を持つ下流が新しいバイナリで qa を立てたとき、指示のない座席が黙って起動するのを防ぐ。`--prompt` があるときは独自の役割として許す。互換用の別名(qa を reviewer 雛形に割り当てる)は作らない。役割が 3 つだという契約を曖昧にするため。
- ゲートの BLOCKED は `GATE: fail` と `GATE: unrunnable` に分ける(Codex plan advisory の MEDIUM)。権限や環境の問題を implementer に差し戻しても直らず、差し戻しが繰り返されるため。`GATE:` は typed protocol が許す任意のヘッダ行で、プロトコルの変更は要らない。
- 改名は Go の識別子とコメントまで行う(ユーザーの判断)。`leader` で grep すれば、値・定数・関数がすべて見つかる。
- 旧名 `lead` は案内付きで止める(ユーザーの判断)。`--role lead` と `--id lead` は `--prompt` があっても拒否する。qa と扱いを変えるのは、`lead` が指示役の宛先 ID だった名前で、同じ名前の座席があると旧版の手順で送られたメッセージがそこへ届くため。
- `ralph.toml` の旧キーは spawn で拒否し、`config.Load` ではエラーにしない(consult-advisor と合意)。旧キーが黙って外れると、`lead = "guarded"` と書いた人の指示役が `autonomous` で動く。一方 `config.Load` で止めると、古い org の片付けまで止まる。
- スライスは改名を先にする(consult-advisor と合意)。qa の作業も `leader.md` と `/org` skill を書き換えるので、先に名前をそろえておけば、後のスライスは新しい名前で書ける。
- `quality-gates.md` の Quality pipeline gate の行は、決定論的なゲートの表に置いたまま、雛形の指示で動いていると書き添える。表から外すと、下流の読者がゲートの存在自体を見落とすため。
- 1 つの PR にする(ユーザー: 「それも計画に含めて」)。レビューしやすいように、挙動を変えない改名を独立したコミットにする。
- Critical forks: None(改名の範囲・旧名の扱い・`Leaded` の扱いはユーザーが選んだ。ほかは既定の判断で足りる)

## Acceptance criteria

### leader への改名

- [x] AC-11: `LeaderIdentity == "leader"` で、`internal/org/prompts/leader.md` があり `lead.md` がない。`ralph org start` が seat id・役割ともに `leader` で座席を立てる(テスト)。agmsg の登録と、watchdog の ALERT の宛先が `leader` になる(テスト)。
- [x] AC-12: `git grep -n -P 'Lead(?!e)|\blead[A-Z_]' -- '*.go'` の結果が、Slice 0 の時点で下の例外だけになる(Go の識別子に旧名が残っていない。`Leader` と `Leaded` は掛からない)。当初は `-E 'Lead($|[^e])|\blead[A-Z_]'` と書いたが、macOS の `git grep -E` では `\b` が効かず後半が何にも掛からないことが Slice 0 で分かったので、`-P` に改めた(計画の時点の確認は BSD の `grep` で行っていて、`git grep` では確かめていなかった)。Slice 0(eb30b172)で確認済み。Slice 1 以降に残ってよいのは、旧名の拒否のための識別子(非推奨フラグの変数など)だけ。例外として、watch の state に永続化される JSON のタグ `lead_agent_get` と `history_lead_lines`(`internal/org/watch.go:230-231` とそのテストのフィクスチャ)は残す(下の「実装中の逸脱」を参照)。
- [x] AC-13: 旧名の拒否をテストで検査している。(a) `--role lead` と `--id lead` の spawn は、`--prompt` があっても、dry-run でも拒否され、エラー文に `leader` が含まれ、manifest と receipts に何も書かれない。(b) `ralph.toml` に `[org.roles].lead` か `[org.permissions.roles].lead` があると spawn が拒否され、エラー文がキーの改名を案内する。同じ設定でも `ralph org status`、`stop`、`disband` は動く。`ralph doctor` が warn を出す。(c) `--lead-driver` は警告付きで動き、`--leader-driver` と両方に違う値を渡すとエラーになる。
- [ ] AC-14: `git grep -n -w -i lead -- . ':!docs/plans' ':!docs/reports' ':!docs/evidence' ':!docs/research'` に残る行が、次の分類のどれかに入る。分類ごとのファイルと理由を verify の report に書く。
  - 旧名の拒否のコードとテスト(撤去・改名した役割の表、spawn の拒否、`--lead-driver` の別名、doctor の warn と、それぞれのテスト)
  - 過去のデータのフィクスチャ(`internal/insights/testdata/receipts.jsonl` と、それを読む `internal/insights/insights_test.go`、過去の receipts の形の行を作る `internal/cli/insights_test.go`)
  - 履歴の記録(`docs/insights/events/*.jsonl`、spec の履歴の記述と改訂の節、tech-debt の既存の行)
  - 過去の計画・レポートのファイル名と、その文言の引用(Slice 0 の時点で `internal/org/report.go:11`、`internal/cli/org.go:369-370`、`internal/org/spawn.go:59,534,769,950`、`internal/org/spawn_test.go:545`)
  - 永続化された state のキー(`watch.go` の JSON のタグ `lead_agent_get` / `history_lead_lines` と、それを使うテストのフィクスチャ。`-w lead` には掛からないが AC-12 には掛かる)
  - 文書の中の旧名の案内(`/org` skill の「旧名の `lead` は拒否される」など)
- [ ] AC-15: `agent-messaging.md`(2 面)の契約が `TO: leader` になり、`/org` skill(4 面)、`AGENTS.md` と `.ralph/core/AGENTS.core.md`(+ template)、`README.md`、`codex-seat-permissions.md`(2 面)、`quality-gates.md`(2 面)が `leader` を使う。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が green。

### qa の撤去

- [x] AC-1: `internal/org/prompts/qa.md` がなく、`RenderRolePrompt("qa", …)` が `ok=false` と空文字列を返すことをテストで検査している。
- [x] AC-2: reviewer 雛形のミッション節が、(a) 最初にゲート(leader の指示、なければ `run-static-verify.sh` と `run-test.sh`)を実行する、(b) `GATE: fail` なら root cause とレポートのパスを付けて BLOCKED を返し差分レビューに進まない、(c) `GATE: unrunnable` なら理由を付けて BLOCKED を返し差分レビューに進まない、(d) 通れば差分品質と受け入れ基準をレビューしてゲートの結果と合わせる、を含み、`QA 座席` を含まない。テストはミッション節だけを見る。red: 2 本のスクリプト名、`GATE: fail`、`GATE: unrunnable`、「差分レビューに進まない」の文言を 1 つずつ消すと、そのたびにテストが落ちる(mutation の結果を report に残す)。
- [x] AC-3: leader 雛形のミッション節が、レビューと検証を reviewer に委譲すること、`GATE: fail` を implementer に差し戻すこと、`GATE: unrunnable` は implementer に戻さず leader が直すか人に上げることを含み、`qa 座席` を含まない。red: 差し戻しの文言と `GATE: unrunnable` の扱いを 1 つずつ消すと、そのたびにテストが落ちる。
- [x] AC-4: `--role qa` と空の `--prompt` の spawn は、dry-run でも実際の spawn でも拒否され、エラー文に `reviewer` と `--prompt` が含まれ、manifest と receipts に何も書かれない。`--role qa` に `--prompt` を付けると起動し、初期プロンプトは `--prompt` だけになる。いずれもテストで検査している。
- [ ] AC-5: `git grep -n -w -i qa -- . ':!docs/plans' ':!docs/reports' ':!docs/evidence' ':!docs/research'` の結果が、次のファイルだけになる: `internal/org/prompts.go`(撤去・改名した役割の表)、`internal/org/prompts_test.go`、`internal/org/spawn_test.go`(AC-1 と AC-4 のテスト)、`internal/cli/status_test.go`(Non-goals)、`.claude/skills/org/SKILL.md` と 3 つのミラー(撤去の案内)、`docs/specs/2026-08-01-org-runtime.md`(改訂の節と履歴の記述)、`docs/tech-debt/README.md`(既存の解決済みの行と新しい行)。表を `spawn.go` に置いた場合は `prompts.go` を `spawn.go` に読み替える。
- [ ] AC-6: `/org` skill の 4 面で、Leaded 行・fan-out の例・役割リストが 3 役割になり、reviewer の説明にゲートの再実行と `guarded` の注意が入り、`--role qa` の拒否が書かれている。
- [ ] AC-7: `quality-gates.md`(root と template)の Quality pipeline gate の行が新しい順序を示し、雛形の指示で動いていることを書いている。`templates/base/ralph.toml` の `reviewer = "guarded"` の例に注記がある。`docs/tech-debt/README.md` に hook 強制の未実装の行がある。
- [ ] AC-8: spec に 2026-10-04 改訂の節があり、FR-4・FR-7・AC(74)の行に改訂の印がある。2026-09-16 改訂 (b) の「4 種」が置き換えられたことと、`lead` から `leader` への改名が改訂の節から読める。
- [ ] AC-9: reviewer 雛形の smoke を 1 回行う。scratch のディレクトリで、展開した reviewer 雛形と、わざと落ちるゲートのコマンド(例: `sh -c 'echo "--- FAIL: TestFixture (fixture_test.go:12)"; exit 1'`)を指定した TASK を、`claude -p --model sonnet` に渡す。agmsg はないので、`ralph org send` の代わりに送るメッセージを標準出力に出すよう指示する。ゲートのコマンドだけを実行できる権限にする。合格の条件は、出力に `TYPE: BLOCKED` と `GATE: fail` があり、`SEVERITY:` の付いたレビュー所見がないこと。生の出力を test report に残す。LLM の振る舞いなので 1 回の結果は保証ではなく、Known gap として扱う。

### 共通

- [ ] AC-10: `go test ./...` と `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` が green。

## Implementation outline

1. Slice 0(implementer、sonnet): leader への改名。挙動は変えない。識別子は gopls rename、文字列とコメントは `-w` の置換、`lead.md` の `git mv`、テストの追随。`go build ./... && go test ./...`。AC-11、AC-12。1 コミット(`refactor: rename the org lead identity to leader`)。
2. Slice 1(implementer、sonnet): 旧名 `lead` の止め方。撤去・改名した役割の表、spawn の拒否、`ralph.toml` の旧キーの拒否、`--lead-driver` の非推奨の別名、doctor の warn と、AC-13 のテスト。1 コミット(`feat: reject the old lead name with rename guidance`)。
3. Slice A(implementer、sonnet): qa の雛形とテスト。`qa.md` の削除、`reviewer.md` と `leader.md` の書き換え、AC-1〜AC-3 のテストと mutation、qa の Go のコメント(`qa.md` を消すと `spawn_test.go:1222` の連結テストが落ちるので、同じスライスで直す)。テストを先に赤くしてから雛形を直す。1 コミット(`refactor: drop the qa seat template from the org runtime`)。
4. Slice B(implementer、sonnet): `--role qa` の拒否(Slice 1 の表に「撤去」の種類を足す)と AC-4 のテスト。あわせて、Slice 1 の設定キーのエラー文の「default, autonomous に戻る」を、`[org.permissions].default` に戻るという書き方に直す(既定を変えている人には autonomous と限らないため)。1 コミット(`feat: reject --role qa without --prompt`)。
5. Slice C(implementer、sonnet): 文書(qa と leader の両方)。`/org` skill(`.claude` 側を直してから `./scripts/sync-skills.sh` で `.agents` 側を生成し、template の 2 面に写す)、`agent-messaging.md`、`AGENTS.md` と `AGENTS.core.md`、`README.md`、`codex-seat-permissions.md`、`quality-gates.md`、`ralph.toml` の注記、spec の改訂、tech-debt の行。1 コミット(`docs: describe the three-role org runtime and the leader rename`)。
6. pipeline: self-review → verify → test(AC-9 の smoke を含む)→ sync-docs → cross-review → PR。

## Verify plan

- Static analysis checks: `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`go vet ./...`、`gofmt -l .`。
- Spec compliance criteria to confirm: AC-1〜AC-15。AC-5、AC-12、AC-14 は grep の出力をそのまま report に残す。
- Documentation drift to check: `AGENTS.md` の Repo map(`internal/org/` の「role prompt templates」)、`CLAUDE.md`、`.claude/rules/ralph/model-routing.md` の org runtime の節、`docs/recipes/` の org 関連、`.claude/skills/org/SKILL.md` の動詞の例。
- Evidence to capture: AC-5 / AC-12 / AC-14 の grep の出力、AC-2 と AC-3 の mutation の結果。

## Test plan

- Unit tests: `go test ./internal/org/... ./internal/cli/... ./internal/config/...`。AC-1〜AC-4、AC-11、AC-13 のテスト。
- Integration tests: `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`。AC-9 の smoke。
- Regression tests: 既存の `TestOrgSpawn_UnknownRole_NoTemplateApplied`(撤去・改名した役割以外の未知の役割は従来どおり拒否されない)が通り続ける。役割が空の spawn も従来どおり通る。過去の receipts(`role=lead`)を読む insights の集計が変わらない。
- Edge cases: `--role reviewer` に `--prompt` を足したとき、reviewer 雛形の後ろに `--prompt` が連結される(`spawn_test.go:1222` を reviewer に移したテスト)。`--role qa --dry-run` と `--role lead --dry-run` の拒否。`--role QA` や `--role Lead` のような大文字は、役割の照合が大文字小文字を区別する既存の動作のまま、未知の役割として扱う。org_id が 25 文字なら leader 座席を立てられ、26 文字なら herdr の上限で拒否される。
- 新旧の版の混在: 「古い skill と新しいバイナリ」は AC-4 と AC-13 で扱う(qa や lead を立てようとすると案内付きで拒否される)。「新しい skill と古いバイナリ」は、下流では upgrade が版をそろえるので起きず、この meta-repo のマージからリリースまでの間に限られる。テストではなく Rollout の手順で扱う。
- 実機で座席を起動する smoke(herdr + agmsg)は行わない。AC-9 の `claude -p` の smoke で、雛形の指示に LLM が従うかを 1 回だけ見る。
- Evidence to capture: test report(AC-9 の生の出力を含む)。

## Risks and mitigations

- R1: 古い skill を持つ下流が新しいバイナリで `--role qa` を立てる。対策: `--prompt` なしでは案内付きで拒否する(AC-4)。PR の説明とリリースノートに、qa 雛形の撤去と移行先(reviewer)を書く。
- R2: ゲートの出力で reviewer のコンテキストが膨らむ。対策: 雛形に「ゲートは子サブエージェントに任せ、要約とレポートのパスだけ受け取る」例を置く。
- R3: 同じ座席がゲートとレビューを両方行うので、fail を言い訳して先に進むおそれがある。対策: 雛形に「スクリプトの出力を正とする」「fail ならレビューに進まない」をはっきり書き、AC-2 の文言単位のテストと AC-9 の smoke で確かめる。
- R4: reviewer を `guarded` で動かすと、ゲートの実行が許可待ちで止まる。codex の reviewer 座席は既定で `guarded` に固定される。対策: 止まらずに実行できなかった場合は `GATE: unrunnable` で leader に返し、leader が権限を直す(AC-3)。許可待ちで止まった場合は、watchdog の stall の ALERT で leader が気づく。`/org` skill と `ralph.toml` の例に注意を書く。codex の `guarded` 座席がゲートのスクリプトを許可なしで実行できるかは未確認(Open questions)。
- R5: 下流の `quality-gates.md`、`codex-seat-permissions.md`、`ralph.toml` は seed なので、古い記述(`qa`、`lead`)が残る。対策: seed の設計どおりで、リリースノートで知らせる。`ralph.toml` の旧キーは AC-13 の拒否と doctor の warn で気づける。
- R6: ユーザーが承認済みの判断(4 フェーズ)を改める。対策: spec の改訂の節に理由を残し、承認された履歴の記述は消さない。
- R7: 4 面のミラーのずれ。対策: `check-skill-sync.sh` と `check-sync.sh`。
- R8: 新旧の版の混在(Codex plan advisory の HIGH)。座席は spawn した時点の雛形で動き続けるので、更新の前に立てた座席は古い手順と古い宛先(`lead`)のまま動く。新しいバイナリの watch は古い org の指示役を見つけられない。この meta-repo では、マージからリリースまでの間、インストール済みの `ralph` が古い雛形を持つ。対策: Rollout の手順を参照。
- R9: 機械的な置換で、関係のない語や過去のデータまで書き換える。対策: 識別子は型付きの rename、文字列は `-w` の置換に限る。過去のデータのフィクスチャは残す対象として Scope に挙げる。AC-12 と AC-14 の grep で残りを分類して確かめる。
- R10: herdr のエージェント名の上限で、指示役の座席の org_id が 25 文字までになる。26 文字の org_id で動いていた org は、新しいバイナリで指示役を立てられない。対策: リリースノートに書く。既存の上限の検証がそのまま拒否するので、黙って壊れることはない。
- R11: 1 つの PR が大きくなる(改名だけで数百行)。対策: 挙動を変えない改名を Slice 0 の独立したコミットにし、AC-12 / AC-14 の grep で機械的に確かめられるようにする。self-review と cross-review は Slice 0 を「改名の漏れと巻き込み」の観点に絞って見られる。

## Rollout or rollback notes

- state や manifest の移行はない。過去の manifest にある `role=qa` や `seat_id=lead` は、`ralph status` が自由な文字列としてそのまま表示する。
- 更新の手順(PR の説明とリリースノートに書く):
  1. 動いている org があれば `ralph org disband` で片付ける。
  2. バイナリを更新する。
  3. `ralph upgrade` で core の skill と rule、`AGENTS.md` の managed block を更新する。
  4. `ralph.toml` に `[org.roles].lead` や `[org.permissions.roles].lead` があれば `leader` に改める(`ralph doctor` が教える)。
  5. org を立て直す。

  座席は spawn した時点の雛形で動くので、1 を省くと古い qa や lead の座席が残る。新しいバイナリでも古い org の stop / disband はできる。
- この meta-repo では、マージからリリースまでの間に org を動かすなら、`go build ./cmd/ralph` で作ったバイナリを使う。リリースは `/release`(手動)で、マージの後に早めに切る。
- 戻すときは PR を revert してリリースし、下流は 1〜5 を同じ順で行う(4 は `leader` を `lead` に戻す)。revert すると qa 雛形と `lead` の名前が戻り、旧名の拒否もなくなる。

## 実装中の逸脱

- 2026-10-04(Slice 0 の前): `internal/org/watch.go` の `watchPendingAlert` は、watch の state に `lead_agent_get` と `history_lead_lines` の JSON のタグで永続化される。Go のフィールド名(`LeadAgentGet`、`HistoryLeadLines`)は改名するが、タグは残す。移行なしにキー名を変えると、更新の前に書かれた state を読んだときに値が黙ってゼロになり、`history_lead_lines` の番兵 `-1` が失われるため。AC-12 と AC-14 では、このタグを「永続化された state のキー」として例外に数える。

- 2026-10-04(Slice 0、eb30b172): implementer は gopls ではなく、単語境界の perl の置換と `gofmt -w` で改名した。`go build` / `go vet` / `go test ./...` / `run-verify.sh` が green。manifest の Details に書く診断用の文字列(`lead_self=true` → `leader_self=true`、`lead_join=` → `leader_join=`、`lead_is_anomaly_subject` → `leader_is_anomaly_subject`、`agmsg_lead_joined` → `agmsg_leader_joined`)も改名した。これらを読むのは Go のテストだけで、過去の manifest や evidence に残る旧い文字列はそのまま。コミットの trailer は、委譲先のモデルに合わせて `Claude Sonnet 5.5` になっている。

- 2026-10-04(Slice A、90488a06): qa の Go のコメントもこのスライスで直した(Implementation outline の 3 を参照)。AC-2 / AC-3 の mutation は 9 通り(reviewer: 2 本のスクリプト名、`GATE: fail`、`GATE: unrunnable`、2 つの項目それぞれの「差分レビューに進まない」。leader: 「implementer 座席に差し戻」「implementer には戻さず」「`GATE: unrunnable`」)で、どれも `TestRenderRolePrompt_Reviewer_MissionRunsGateFirstAndBlocksWithoutReviewing` か `TestRenderRolePrompt_Leader_MissionRoutesGateBlocked` が落ちることを確かめ、元に戻した。

- 2026-10-04(Slice B、fbc9e539): 撤去した役割の判定は `p.Prompt == ""` の厳密な比較で、空白だけの `--prompt " "` は拒否しない(AC-4 の「空の `--prompt`」どおり)。表の `"qa"` は `internal/org/prompts.go` にだけあり、`spawn.go` のエラー文は `%q` で役割名を埋めるので、AC-5 の Go ファイルは計画どおり `prompts.go` になった。設定キーのエラー文は「full model_pool と `[org.permissions].default` に戻る」に直した。

## Open questions

- codex の `guarded` 座席が、許可を求めずにゲートのスクリプトを実行できるか。この PR では確かめず、実行できなければ R4 の `GATE: unrunnable` の経路で扱う。

## Readiness checklist

- [x] 影響範囲を grep で洗い出した(qa と lead の両方、履歴を除く)
- [x] qa を前提にした Go のコードがないこと、lead の値が定数 1 か所で決まることを確かめた
- [x] 受け入れ基準をコマンドかテストで確かめられる形にした
- [x] task worktree を作った(`.claude/worktrees/org-drop-qa-seat`、`refactor/org-drop-qa-seat`)
- [x] Codex plan advisory の 3 件を反映した
- [x] leader の改名の組み込み方を consult-advisor に確認し、指摘の 2 点(置換の手順、旧キーの拒否の位置)を反映した

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
