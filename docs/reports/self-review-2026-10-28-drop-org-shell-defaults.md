# Self-review report: drop-org-shell-defaults

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-drop-org-shell-defaults.md`(D1〜D3、AC1〜AC5、Progress の S1〜S3)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1。run 1 は 5de4cc6a の report、run 2 は直し 7d59e278 のあとの再 run で、この file を同じ場所で更新した)
- Run 2 の scope: 直しの commit 7d59e278(5 ファイル、+34/-51)の全体と、直しが触れていない所見(L4)の再確認。run 1 の本文は下にそのまま残し、L1〜L5 のその後は「Run 2」の節にまとめた。
- Scope: `git diff 15e301c8...HEAD`(run 1 の時点で 5 コミット、10 ファイル、+458/-315)。中心は S1 e948b85d(`internal/config/defaults_sync_test.go` の組み直しと `config_test.go` の畳み)、S2 76c7595d(`scripts/ralph-config.sh` とテンプレートの写しから 18 変数と export を削除)、S3 c56fa63c(コメントと文書)。計画ファイルと f94c072b(Progress だけ)は読んだが所見の対象にしていない。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: run 1 の所見は L1〜L5(run 2 も同じ ID で状態を追う)。run 2 で新しく立てた所見はない。

## Evidence reviewed

- `git diff 15e301c8...HEAD -- internal/config/defaults_sync_test.go` の全体。`orgTemplateProbe`(`:111-142`)、`orgTemplateDrift`(`:147`)、`valueDrift` / `listDrift`(`:178`、`:190`)、`TestDefaultsLockStep` の `org` と `shell_and_skills`(`:200-325`)、`TestOrgTemplateDrift_DetectsChangedAndMissingKeys`(`:327-`)。
- probe の読み方。`toml.Unmarshal` はポインタの field を、キーがなければ nil のままにする。`orgPoolKeysPresent`(`config.go:599`)と同じ慣用句。`OrgModelPoolEntry`(`config.go:130`)は `driver` / `model` の 2 field だけで独自の `UnmarshalTOML` がなく、`*[]OrgModelPoolEntry` で読める。probe の field 数を数えると 7 + 2 + 4 + 5 + 1 = 19 で、計画の「19 キー」と合う。`valueDrift` は nil を先に見てから `*tmpl` を引くので nil 参照はない。`listDrift` は `slices.Equal` で並びも見る。
- テンプレートの実物。`templates/base/ralph.toml` の `codex_verified = false`(`:91`)、`auto_merge = false`(`:157`)、`wake = ["pull"]`(`:183`)は、変異テストが `strings.Count == 1` で求める行頭の 1 回だけの形で、コメントや別の table に同じ行がない。
- `scripts/ralph-config.sh` の全体(88 行)と `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh`(一致)。`git diff` の削除側は 18 変数、各変数のコメント、`export RALPH_ORG_*` の 3 行で、残った `RALPH_ORG_AGMSG_HOME` の行(`:56`)と `export RALPH_CLAUDE_REVIEWER_MODEL …`(`:70`)は元のまま。
- `git grep` で、消した変数名以外の主張の残り: 「3 面」「三面」「lock-step surfaces」。当たりは `docs/specs/2026-08-01-org-runtime.md:157`(Research findings、計画が履歴として残す)と `docs/specs/2026-10-07-org-multi-org-director.md:213`(履歴)、`defaults_sync_test.go:18`、`:241`(`agmsg_home` の 3 面で正しい)だけ。
- `templates/base/scripts/` の下で `#[0-9]{2,4}` を grep: 今回の `ralph-config.sh:48` と、`xreview-helpers.sh:84` の 1 件(前からある)。
- `internal/config/config_test.go` の `TestLoad_TemplateRalphToml`。畳んだあとも `slices` は `:825` ほか、`strings` は `:205` ほかで使われていて import は余らない。
- 確かめていない: テスト、静的検査、変異の実行(S1 の報告では、テンプレートの `max_orgs` を 10 から 11 にすると `TestDefaultsLockStep/org` が落ちる)。

## 依頼された判断: `TestLoad_TemplateRalphToml` から畳んだ確認は「置き換え」か「弱め」か

置き換えで、弱めていない。

- 畳んだ確認は 4 つ。(1) `"\nauto_merge = false\n"` の文字列があること、(2) `cfg.Org.Director.AutoMerge == Default()`、(3) `"\nwake = [\"pull\"]\n"` の文字列があること、(4) `cfg.Org.Director.Wake` が `Default()` と等しいこと。
- (2) と (4) は `Load` が `Default()` から始めるので、テンプレートがキーを書いていなければ `Default()` の値が残って必ず通る。キーが書かれていて別の値のときだけ落ちる。新しい `valueDrift` / `listDrift` は同じ「別の値」を `template = …, Default() = …` で見つける。
- (1) と (3) は文字列検索で、`strings.Contains` が file 全体を探すため、`[org.director]` の外の table に同じ行を書いても通る。新しい probe は `[org.director]` の field を見るので、table を間違えると落ちる。`auto_merge=false` のように空白の書き方が違うだけで落ちることもなくなる。
- 消えた確認が受け持っていた「キーがない」は、`TestOrgTemplateDrift_DetectsChangedAndMissingKeys` の `a removed auto_merge, whose default is false` と `a removed wake` が受け持つ。どちらも `strings.Count == 1` の前提つきで、テンプレートの書き方が変わると vacuous に通らず Fatal になる。さらに 19 キーすべてに同じ検査が及ぶ。
- `wake_cmd` が空であることの確認は残っている(`config_test.go:91`)。
- 差が 1 つある。`TestLoad_TemplateRalphToml` は `ralph.toml` がなければ Fatal、新しい `org` サブテストは Skip。meta-repo ではテンプレートがあり、なければ `TestLoad_TemplateRalphToml` 自身が落ちるので、守りは減っていない。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | 下流に配られる `ralph-config.sh` の新しいコメントが、meta-repo の中でしか通じない。(a) `ralph issue #276 decides what happens to it` の `#276` は、`ralph init` した project では、その project 自身の tracker の番号になる。(b) `so this file no longer carries any` は、写しがあった頃を知らない読者には解けない履歴の言い方。(c) 末尾の `which is why it is not exported below` は、`:60-67` の `RALPH_ORG_AGMSG_HOME is deliberately NOT exported here …`(env > toml の優先で、export すると toml の `[org].agmsg_home` を隠す、という本当の理由)と同じ説明を 7 行手前で短く繰り返し、短い側は理由を落としている。 | `scripts/ralph-config.sh:46-53`(`templates/base/scripts/ralph-config.sh` と byte 一致)。前の版の同じ file に `#NN` はなく、`templates/base/scripts` で `#NN` を持つのは `xreview-helpers.sh:84` の 1 件(前から)だけ。`check-template-purity.sh` に issue 番号の pattern はなく、通る。前の版にも `internal/config` への言及が 5 つあった(だから file 全体が下流向けに清潔だったわけではない)が、`#276` は今回の追加 | 時間に依らない言い方にする。例: 「`RALPH_ORG_AGMSG_HOME` の既定値。Go の binary が環境から読む(`driver.ResolveAgmsgHome`)。ここでは export しない(下の理由)。`internal/config` のテストが `config.Default()` と突き合わせる」。issue 番号と「no longer carries」は plan と tech-debt に任せ、(c) の末尾の 1 文は削って `:60` の説明に一本化する。両方の写しに同じ直しを入れる |
| LOW | exception-handling | `shell_and_skills` サブテストが、テンプレートを読めないとき(`ErrNotExist` に限らず、どの read error でも)サブテスト全体を Skip し、テンプレートを使わない `SKILL.md` の fallback 照合まで飛ばす。`org` サブテストは `ErrNotExist` だけ Skip し、ほかは Fatal にしている。計画の Edge cases は「`.claude/skills` がないときは skill の照合だけ飛ぶ」「shell がないときは shell の照合だけ飛ぶ」と、飛ぶ範囲を狭く書いている。 | `defaults_sync_test.go:231-235`(`os.ReadFile(tomlPath)` の失敗で `t.Skipf`)と、それに依らない `:262-325` の照合。`:206-213` は `errors.Is(err, fs.ErrNotExist)` で分けている | テンプレートを使うのは `agmsg_home` の 3 面照合だけなので、その部分に限って読む。`ErrNotExist` なら `agmsg_home` の照合だけ飛ばして skill の照合へ進み、ほかの error は `org` と同じく Fatal にする。vendored の repo にテンプレートだけがない状況は実際にはまれなので、直すかどうかは軽い判断 |
| LOW | readability | `agmsg_home` の 3 面照合で、`shell_and_skills` サブテストの外側に `shellVal` / `goVal`(`:242-243`)を置き、同じサブテストのループの中で `shellVal := mustShell(t, shell, c.envVar)`(`:315`)が別の意味で同じ名前を作り直している。また `agmsg_home` の「テンプレート対 `Default()`」は `org` サブテストの `orgTemplateDrift` がすでに見ていて、同じずれが 2 つのサブテストで別の文面で報告される(`:248-260`)。shell が両方と等しければ「shell 対テンプレート」は導けるので、その分岐は他の 2 つが落ちるときにしか動かない。 | `defaults_sync_test.go:242-262`、`:315` | 3 面照合は計画 D2 が残すと決めたのでそのまま。外側を `agmsgShell` / `agmsgDefault` のように名前を分けて、ループの `shellVal` と衝突させない。テンプレート対 `Default()` の重複は、直すなら shell 対 `Default()` と shell 対テンプレートだけに減らす |
| LOW | maintainability | 「19 キー」という数が 3 つのコメントに書かれ、コードにも test にも固定されていない。`Config.Org` に 20 個目のキーを足して probe と `orgTemplateDrift` に足し忘れても、`TestDefaultsLockStep` は通り続ける(`probe` にない key は誰も見ない)。この穴は組み直し前の手書きの `check(...)` 列にもあった(足し忘れが同じ形で素通りした)ので今回の退行ではないが、計画のリスク R2(「キーが消えても気づけない」)の守りは、この 19 個の列挙の正しさに寄っている。 | `defaults_sync_test.go:10`、`:106`、`:144`(コメントの「19」)。probe の 19 field と `orgTemplateDrift` の 19 行は今は一致 | 数をコメントから外すのが最小。守るなら、`reflect` で `OrgConfig` の葉(`Roles` / `LaunchCmd` / `WakeCmd` のように shell にも既定の写しにもない除外を明示)を数えて probe の field 数と比べる 1 本のテスト。やるなら tech-debt の 1 行にして次の `[org]` キー追加(#274 など)を trigger にするのが小さい |
| LOW | maintainability | 「2 面」と書いた説明が、`org.agmsg_home` だけ 3 面のまま残る事実を言っていない。テストの冒頭コメント(`defaults_sync_test.go:15-18`)と shell の見出し(`ralph-config.sh:46-48`)は正しく例外を書いているが、`config.go` の `OrgConfig` のコメントと spec の FR-2 の印は例外なしで、FR-2 の `照合から外し` は `scripts/ralph-config.sh` が `RALPH_ORG_AGMSG_HOME` の照合にまだ使われている以上、言い過ぎ。`agmsg_home` の既定値を変える人が `config.go` のコメントだけを読むと、shell の行が要ることに気づかず、テストの失敗で初めて知る(失敗は surface 名つきなので損は小さい)。 | `internal/config/config.go:31-32`(`kept in lock-step with templates/base/ralph.toml (defaults_sync_test.go)`)、`docs/specs/2026-08-01-org-runtime.md:74`(`(2026-10-28 #273 で照合から外し、config.go と templates/base/ralph.toml の 2 面にした)`)。同 spec の注記 (a)(`:45`)は model_pool の話なので 2 面で正しい | `config.go` のコメントに「`agmsg_home` は `scripts/ralph-config.sh` の `RALPH_ORG_AGMSG_HOME` とも(#276 まで)」を足す。FR-2 の印は「`RALPH_ORG_AGMSG_HOME` を除いて」を足す。#276 が片付けば両方とも消えるので、その issue の完了条件に入れておくと忘れない |

## Run 2: 直し 7d59e278 のあと(L1〜L5 のその後と、直しの差分の確認)

読んだもの: `git show 7d59e278` の全体、直し後の `scripts/ralph-config.sh:36-80`、`cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh`(一致)、`internal/config/defaults_sync_test.go:1-306`、`internal/config/config.go:26-36`、`docs/specs/2026-08-01-org-runtime.md:74`。確かめていない: テスト、静的検査(skill の規則と依頼どおり、走らせていない)。

### 所見のその後

| ID | 状態 | 確認したこと |
| --- | --- | --- |
| L1 | 直った(7d59e278) | 両方の `ralph-config.sh` で `#276` と `no longer carries` が消え、非 export の理由は `:60-69` の 1 か所になった(run 1 の本文の `:60-67` は `:60-69` が正しい)。見出し(`:46-53`)は「下の行は export しない。理由は export の行の上の注記」と指すだけで、理由を繰り返さない。両方の写しは `cmp` で一致。`#[0-9]{2,4}` の grep は当たりなし。`templates/base/scripts` の `internal/` は、この file の `:49` の 1 件と `plan-visual.sh:401` の 1 件(別件)で、PR の前の同じ file は 5 件だった。 |
| L2 | 直った(7d59e278) | `shell_and_skills` はテンプレートを読まなくなり、`ralph.toml` がなくても `SKILL.md` の照合は走る。Skip は shell file がないとき(`:228`)と `.claude/skills` がないとき(`:260`)だけで、`:29-31` と `:253-258` のコメントと合う。`tomlPath`、`toml`、`errors`、`fs` は `org` サブテストと `TestOrgTemplateDrift_…` がまだ使うので、未使用の宣言と import は生じない。 |
| L3 | 直った(7d59e278) | 外側の `shellVal` / `goVal` は `agmsgShell` / `agmsgDefault` になり、ループの `shellVal`(`:296`)と衝突しない。テンプレート対 `Default()` の重複した報告も消えた。 |
| L4 | 残す(直しの対象外) | 「19 キー」は `defaults_sync_test.go:10`、`:107`、`:145` のコメントのままで、probe の field 数(19)と `orgTemplateDrift` の行数(19)は今も一致。組み直し前からある穴で、直しの commit は触れていない。計画の判断(`known gaps` へ)に従う。 |
| L5 | 直った(7d59e278) | `config.go:28-34` が「`AgmsgHome` の既定値は `scripts/ralph-config.sh` の `RALPH_ORG_AGMSG_HOME` とも比べる」と書き、FR-2 の印(`org-runtime.md:74`)が「`agmsg_home` のほかは」に変わった。テストの冒頭コメント(`:15-19`)、shell の見出し、`config.go`、spec の 4 か所の言い方が互いに食い違わない。 |

### 依頼された判断

- (a) L1 でテンプレート側の comment に issue の URL を使わなかったこと: 妥当。`check-template-purity.sh` の `FIXED_PATTERNS` が `yoshpy-dev` と `github.com/yoshpy-dev` を `templates/` 全体に禁じているので、URL は書けない。run 1 の L1 は番号も履歴もコメントから外す案で、issue の番号は plan とテスト(`defaults_sync_test.go:17`)に残るので、追跡できる場所は減っていない。
- (b) L3 で shell 対テンプレートの比較を落としたこと: 妥当。`shell == Default()`(`shell_and_skills`)と `template == Default()`(`org`)が両方通れば shell == テンプレートは導ける。テンプレートがなければ、前の版でもこの比較は Skip だったので、失った検査はない。失ったのは、shell とテンプレートが直接ずれたときの「shell 対テンプレート」という 1 行の文面だけで、ずれた側は `Default()` と並べて名指しされる。`-run '…/shell_and_skills'` だけを打つとテンプレートは見ないが、そのことは `:17-19` のコメントが書いている。計画 D2 は「3 面の照合は残す」と書いていて、残し方が「直接の 3 者比較」から「2 つのサブテストの合成」に変わったので、計画の Deviation notes に 1 行あると `/verify` が迷わない。
- 直しの差分の新しい問題: なし。見出しの「`internal/config/defaults_sync_test.go` … compares its default」の「its」は、文の主語(テスト)とも読めるが、run 1 の前の版も同じ言い方で、前後の文脈から `RALPH_ORG_AGMSG_HOME` と読める。所見にはしない。

## Positive notes

- テンプレートを `Load` を通さず probe に読み、nil と値のずれを分ける設計(D1)は、`Load` が `Default()` から始まるせいで「キーを消しても一致する」という穴を正しく塞いでいる。`codex_verified = false` や `auto_merge = false` のようにゼロ値が既定のキーも、ポインタだから有無が分かる。
- 変異の単体テストが、(i) `strings.Count == 1` で「書き換え前の文字列が 1 回だけ」を先に確かめ、テンプレートの書き方が変わったときに vacuous に通らず Fatal になる、(ii) ずれが 1 行だけであることと、キー名つきの前置きを固定する、(iii) `model_pool` の並び替えも入れている。計画の Test plan より厚く、handoff にない「wake の行を消す」も足してある。
- `TestDefaultsLockStep` を `t.Run` で `org` と `shell_and_skills` に分け(D3)、org の照合を `scripts/ralph-config.sh` の有無から切り離した。
- 失敗の文面が surface とキーを名指しする(`org.max_seats: template = 6, Default() = 5`)。
- shell 側の差分は削除だけで、残った行と export の 1 行に手が入っていない。2 つの写しは `cmp` で一致。
- S3 の文書は、履歴(Research findings、2026-10-07 の spec、evidence、reports)に触れず、列挙した 5 か所に止まっている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L4(未登録): 「19 キー」の数がコメントにしかなく、`Config.Org` に 20 個目のキーを足して probe と `orgTemplateDrift` に足し忘れても `TestDefaultsLockStep` が通り続ける | 新しい `[org]` キーの既定値のずれが、テンプレートとの照合から漏れる。組み直し前の手書きの列にも同じ穴があり、今回の PR が作った退行ではない | 組み直し前からの穴で、この PR の範囲(shell の写しを外し、照合を `Default()` とテンプレートに絞る)の外。run 2 の時点で pipeline は cycle 1 で、オーケストレーターは `/pr` の known gaps に書くと決めた | 次に `[org]` のキーを足す変更(#274 など)。`reflect` で `OrgConfig` の葉を数えて probe の field 数と比べる 1 本のテストか、コメントから数を外す | 本 report の L4、`docs/plans/active/2026-10-28-drop-org-shell-defaults.md` の R2 |

_(L1〜L3、L5 は 7d59e278 で直った。L4 は上のとおり。`docs/tech-debt/README.md` には、この commit では足していない。`/pr` で known gaps にするか、台帳の 1 行にするかはオーケストレーターが決める。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 1)
- Follow-ups: 残る所見は L4(動作を変えない、pre-existing)だけ。L1〜L3、L5 は 7d59e278 で直り、直しの差分に新しい所見はない。(1) L4 は `/pr` の known gaps に書くか、次の `[org]` キーの追加を trigger に台帳へ 1 行足す。(2) L3 の判断(shell 対テンプレートの直接比較を、`shell == Default()` と `template == Default()` の合成に変えた)は、計画 D2 の「3 面の照合は残す」の残し方が変わるので、計画の Deviation notes に 1 行足すと `/verify` が迷わない。(3) 7d59e278 はコードのコメントとテストの構成を変えたので、通常どおり `/verify` へ進む(record-only の例外には当たらない)。
