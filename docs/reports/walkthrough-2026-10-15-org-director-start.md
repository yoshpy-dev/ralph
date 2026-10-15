# Walkthrough: org-director-start(8 段目の前半の 2 本目、8a-2)

- Date: 2026-10-16
- Plan: docs/plans/active/2026-10-15-org-director-start.md(/pr で archive に移す)
- Branch: feat/org-director-start(base f21df66c。この walkthrough を書いた時点で 59 commits、60 files、+16474/-476)
- 読む範囲: 差分の約 4 割は `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画で、読むのは各報告の要約だけでよい。それらを除くと 51 files、+9358/-476 になる。そのうちテストが `internal/cli/org_director_headless_test.go`(+1544)、`internal/org/director_test.go`(+1061)、`internal/org/director_stop_test.go`(+1054)、`internal/org/director_start_test.go`(+1020)ほか。ほかに `/director` と `/org` の skill の 4 面ずつ、README、仕様 FR-9 の注、`templates/base/ralph.toml`、codex の座席の権限の recipe(2 面)、tech-debt がある
- 後工程は 4 回まわした(上限は既定の 2 から 3、4 と上げた)。cross-review の ACTION_REQUIRED は cycle 1 で 1 件、cycle 2 で 2 件、cycle 3 で 2 件あり、どれも次の cycle の前に直した。cycle 4 は 0 件(下の 7)

新しいファイルは、`internal/org/director_start.go`(登録の書き足しと start)、`internal/org/director_stop.go`(stop と後始末のエラー)、`internal/cli/org_director_headless.go`(start と stop の CLI)、`internal/org/prompts/director.md`(director の雛形)、`internal/config/skill_frontmatter_test.go` と、それぞれのテスト。次の順に読むと、依存の向きに沿って追える。

## 1. 登録の記録(`internal/org/director.go`、`director_start.go` の store の関数、b9861932)

- headless の director の居場所は `director.jsonl` にだけ書き、manifest と model receipts には書かない(計画の D2、D8)。manifest に書くと `max_orgs` と `max_total_seats` に数えられ、`stop --all` と `disband --all` に拾われるため
- `attached` に `launch = herdr`、`token`、`model`、`permission_mode` を足した。新しいイベントは `workspace_created`、`pane_created`、`launched`、`stopped`。登録の状態は `starting` から `running`、最後に `stopped` になる
- 書き手はどれも `director.lock` の下で読み直してから 1 行を足す: `AttachHeadless`、`RecordWorkspace`、`RecordPane`(`recordHerdrID` は記録済みの id を断るので、id は空から値にしか変わらない)、`MarkLaunched`、`MarkStopped`、`MarkStoppedIfUnchanged`、`MarkForceStopped`
- 書き込みが途中で止まったら元の長さに切り戻す。切り戻せなければ `ErrDirectorWriteNotRestored` を返す
- 名前には起動ごとの token(crypto/rand の 12 桁の hex)を入れる(D9)。agent は `director_<token>`、workspace の label は `ralph_director_<token>`、tab の label は `director_<token>` になる。herdr の agent 名は server 全体で一意なので、台帳ごとの `d<N>` を名前にすると 2 つの repo の director がぶつかるため
- `ralph org spawn --role director` は断る(D7、`internal/org/spawn.go`)

## 2. start(`StartDirector`、8528515f。CLI は d4da1c45 と 725fff10)

- 何も書かずに断る場合を先に調べる(`checkDirectorStart`)
  - `enabled = false`
  - herdr の pane の中から打った(`HERDR_PANE_ID`)
  - driver が claude でも codex でもない、または model が空
  - codex で、role director の mode が autonomous か edits(D12)
  - mode を driver の引数に写せない
  - 今の登録がある
  - 見張りの関門が断る
- そのあと先に登録してから herdr を打つ(D3)。① `AttachHeadless`、② workspace を作って記録、③ tab を作って pane を記録、④ 雛形を描いて台帳の `prompts/director_<token>.md` に書く、⑤ agent start、⑥ `MarkLaunched` の順で、herdr を打つあいだはロックを持たない
- ② 以降が失敗したら、`abort` が作った分を閉じて `stopped` を書く。⑥ で今の登録でなくなっていたとき(人の `attach --replace`)も同じ
  - 閉じられなければ、今の登録のあいだだけ `detached` を書き、`*DirectorCleanupError` で閉じるための herdr の打ち方と stop の打ち方を出す
  - `abort` が `stopped` を書く前に stop が先に記録していれば、成功として扱う
- 見張りが動いていなければ start が起動する(D5。`ralph org start` と同じ)
- `--config` を渡した start は、その絶対パスを雛形の `ralph org` のコマンドに付ける(3243c499、cycle 1 の指摘)。その設定の `[org].agmsg_home` が相対なら断る(D13、caf1d025、cycle 2 の指摘)。人の shell の `RALPH_ORG_AGMSG_HOME` は director の pane に届かないので、設定されていれば警告を 1 行出す
- permission は `[org.permissions.roles].director`、なければ `[org.permissions].default` を読む(D6)。codex の director は guarded だけを通し、`--add-dir` は付けない。autonomous と edits は `--sandbox workspace-write` に写るが、codex は書ける root の下の `.git` を書かせない。そのため director の打つ `ralph org start --plan` が失敗する(D12)。`codexWritableRootArgs` の director の分岐は使われなくなったので消し、doctor の writable root の検査も director の workspace-write の mode を数えない(`internal/cli/doctor_codex_writable_root.go`)

## 3. stop(`StopDirector`、`recordDirectorStop`。8528515f、b5ce8dea、a2d5fc01)

- 閉じていない headless の登録なら、今の登録か detach で外れたあとか、`starting` か `running` かによらず効く(D4)。`stopped` は `detached` と別のイベントで、今の登録なら同じ write で `detached` も書く
- 何かを送る前に、記録した pane の tab と workspace の label がこの起動の名前であることを確かめる(`confirmDirectorResources`)。違えば C-c も close も打たない(AC5)。そのあと C-c を送り、pane と workspace を閉じる。herdr が知らない id は閉じ済みとして進む
- pane の中(`HERDR_PANE_ID`)や、label を確かめた workspace の中(`HERDR_WORKSPACE_ID`)からの stop は断る。閉じるとこのコマンドが記録の前に終わるため
- `--force`(D11)は、label が違う、確かめられない、閉じられないときに、まだ打っていない C-c と close を打たずに `force` 付きの `stopped` を書く。herdr が id を振り直したあとの登録を外す道で、label の探し方を出す
- 記録の段は 2 つの競争を扱う
  - D14(b888d637、cycle 3 の指摘): stop はロックなしで登録を読む。そのあと、まだ動いている start が workspace か pane を記録したら、`MarkStoppedIfUnchanged` はロックの下でその id を比べ、違えば何も書かずに断る(`ErrDirectorChangedDuringStop`)。閉じたものと、もう一度打つ stop を出す
  - AC8 の最後の文(44b80391、cycle 4 の verify の C4-V3): stop が閉じているあいだに start の後始末か別の stop が先に `stopped` を書いたら、何も足さずに exit 0 で終わり(`AlreadyStopped`)、すでに記録されていたことを出す。打っても断られる stop の打ち方は出さない
- エラーの文の頭の `org: director: stop <id>:` は 1 回だけ出す(`directorRecordStopError`)。`errors.Is` は中の sentinel をそのまま見つける

## 4. status と stop --all(d4da1c45、`internal/cli/status.go`、`internal/org/verbs_all.go`)

- `ralph org director status` と `ralph status` の `director:` の行は、headless の登録なら `starting` か `running`、pane、workspace、model を出し、`--json` にも出す
- `ralph org stop --all` は headless の director を止めない。閉じていない headless の登録があれば止め方を 1 行出し、終了コードは変えない(D10)

## 5. 雛形と skill(`internal/org/prompts/director.md`、`RenderDirectorPrompt`、9fb4f635)

- 雛形は `/director` skill を指す。最初の手順は `ralph org director status` で今の登録が自分の `id` かを確かめることで、違えば何もしない(D3 の窓を塞ぐ)。detach は打たず、止めるときは人に stop を頼む
- `/director` skill の 4 面に、2 つの立て方(今のセッションを昇格する、headless で立てる)と headless の終わり方を書いた。cycle 4 で「終える」の手順に D14 の断りと AC8 の exit 0 を足した(490a5e4c)。`/org` skill の `director` の行も同じ
- S3 の skill の description は、引用符のない値に「コロンと空白」を含み、YAML として読めなかった(cycle 3 の指摘)。言い換えて直し、tracked のすべての `SKILL.md` の frontmatter でこの型を落とすテストを足した(D15、`TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash`、54 本)。YAML の parser ではないので、見るのはこの 2 つの型だけになる

## 6. 文書

- README の director の段落、仕様 FR-9 の 8a-2 の注、AGENTS.md の repo map、`templates/base/ralph.toml` の `[org.permissions.roles].director` のコメント、codex の座席の権限の recipe(director は guarded だけ)
- tech-debt: director に関わる既存の 13 行に、headless の director が同じ `/director` skill に従うのでその行をどう扱うかを書き足した。新しい 15 行のおもなものは次のとおり
  - claude の director が autonomous で動くとき、実装しないことは雛形と skill の指示だけで守られる(D6)
  - start が herdr の workspace を作ったあと記録の前に落ちると、workspace が記録なしで残る。落ちた start の `starting` の登録は期限で外れない
  - `launch_cmd` は 8a-2b に送った
  - codex の director は guarded だけで動く(D12)。`RALPH_ORG_AGMSG_HOME` は director に届かない(D13)。どちらの前提も本物の herdr と codex では見ていない
  - stop の記録失敗の文で同じ句が 2 回出る。D14 と AC8 の競争は偽の herdr でしか確かめていない
  - 生き残った mutant

## 7. cycle ごとの直し

- cycle 1(self-review の MEDIUM 1 と LOW 6、da818b4f と dbd67a58。cross-review の 1 件、3243c499): `--config` が雛形に入らず、director が別の設定を読んでいた
- cycle 2(cross-review の 2 件、caf1d025): codex の director の workspace-write(D12)と、相対の `agmsg_home`(D13)
- cycle 3(cross-review の 2 件、b888d637): stop が動いている director を `stopped` と書きうる競争(D14)と、skill の YAML(D15)
- cycle 4(self-review の LOW 4、14057e79 と 5097b67e。verify の C4-V3、44b80391): writer を並べた doc、断りの note の重なり、stop の後始末との競争。cross-review は 0 件

## 8. 確かめたこと(evidence)

- 実機: cycle 1 の verify で、人の herdr server に触れない隔離した herdr server を立て、codex と claude の director の start と stop を打った(`docs/evidence/verify-2026-10-15-org-director-start.md`)。cycle 2 以降の変更(D12〜D14、AC8 の最後の文)は実機で見ていない
- テスト: cycle 4 は go1.26.0 と go1.25.8 の両方で `./scripts/run-test.sh` と `go test ./...` が rc 0、org と cli の `-race` で DATA RACE なし。mutation は cycle 4 の 42 件のうち 39 件が red(`docs/reports/test-2026-10-15-org-director-start.md`)
- テストは本物の herdr と osascript を打たず、repo の台帳に書かない(AC13): canary の herdr と osascript は 0 回、repo に `.harness/state/org` は作られていない
- 静的検査: HEAD 19e0661b で `./scripts/run-static-verify.sh` が rc 0(gofmt、golangci-lint 0 件、secret scan clean)
