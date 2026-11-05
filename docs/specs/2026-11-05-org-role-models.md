# org runtime の役割ごとのモデル([org.role_models])

## Summary

org runtime の役割(leader、implementer、reviewer、director と、独自の役割)ごとに、使う driver とモデルの組を `ralph.toml` の `[org.role_models]` で決められるようにする。座席を立てるときの `--driver` と `--model` はこの値を上書きする。役割の既定ができると `[org].model_pool` の仕事はほぼなくなるので、`model_pool` と `[org.roles]` は廃止する。

## Background and problem

### Current state

- `ralph org spawn`・`ralph org start`・`ralph org director start` で `--model` を省くと、`[org].model_pool` の先頭のうち `[org.roles]` が許すもの(claude なら `fable`)が警告付きで選ばれる(`internal/cli/org.go` の `resolveModelOrWarn`、`internal/org/envelope_summary.go` の `DefaultModelForDriverAndRole`)。役割ごとの既定はない。
- そのため leader の雛形(`internal/org/prompts/leader.md` の「ミッション」の 2)は「`--model` は必ず明示する」と決めている。分割計画では、director が人と座席の driver と model を決めて前文に書く(`.claude/skills/director/SKILL.md`)。
- `model_pool` には仕事が 3 つある。
  1. `--model` を省いたときの既定(先頭のエントリ)。
  2. 許可リスト。プールにない driver とモデルの組は spawn で断る(`internal/org/envelope.go` の `ValidateSpawnEnvelope`)。`[org.roles]` もプールの中のモデル名しか書けない。
  3. ほかの機能の対象リスト。doctor の codex スラッグの検査(`internal/cli/doctor_codex_models.go`)、`--probe-models`(`internal/cli/doctor.go`)、codex writable root の検査(`internal/cli/doctor_codex_writable_root.go` の `codexModelPoolModels`)、片方の CLI だけを選んだ `ralph init` で他方の行をコメントにする処理(`internal/cli/init_settings.go`)、leader の雛形の `{{ENVELOPE}}`。
- 既定のプールは 8 モデルをすべて含む。既定のままでは、許可リストとして効いているのは綴り違いや未知のスラッグを座席を立てる前に断ることだけになる。
- spawn と start は `model_pool` を作業ディレクトリの `ralph.toml` から読む。機能の org では feature worktree の写しで、leader が書き換えられる。

### Desired state

- 人は `ralph.toml` に役割ごとの driver とモデルを 1 行ずつ書く。書かない役割には組み込みの既定が入る。
- leader と director は、ふだん `--driver` と `--model` を渡さずに座席を立てる。分割計画の前文に書くのは、`ralph.toml` と違う driver やモデルにする座席だけになる。
- モデルの設定は `[org.role_models]` の 1 つにまとまり、`model_pool` と `[org.roles]` はなくなる。

## Requirements

### Functional requirements

- [ ] **FR-1 キー**: `[org.role_models]` は、役割の名前から `{ driver = "<claude|codex>", model = "<名前>" }` の組を引く表とする。役割の名前は任意(`--prompt` を渡す独自の役割も書ける)。`model` は CLI に渡す名前(claude はエイリアス、codex はスラッグ)で、`""` は「`--model` を渡さず、driver の CLI の既定のモデルで動かす」を表す。
- [ ] **FR-2 組み込みの既定**: `config.Default()` は既定の組を 2 つ持ち、実際に使われる `[org].driver_pool` に `claude` があれば claude の組を、なければ codex の組を使う。

  | 役割 | claude の組 | codex の組 |
  |------|-------------|------------|
  | leader | claude / `opus` | codex / `gpt-6.1-sol` |
  | implementer | claude / `opus` | codex / `gpt-6.1-sol` |
  | reviewer | claude / `opus` | codex / `gpt-6.1-sol` |
  | director | claude / `""`(CLI の既定) | codex / `""`(CLI の既定) |

  `ralph.toml` と `ralph.local.toml` のどちらにも書かれていない役割には、この表の値が入る。書かれた役割は、その値で置き換わる。`templates/base/ralph.toml` は既定をコメントで示し、`internal/config/defaults_sync_test.go` が `Default()` とコメントの値の一致を検査する。
- [ ] **FR-3 解決の順**: `ralph org spawn`・`ralph org start`(leader)・`ralph org director start`(director)は、driver とモデルを次の順で決める。
  - `--driver` と `--model` を両方渡せば、その値を使う。
  - `--model` だけを渡せば、役割の driver とそのモデルを使う。
  - 役割と違う `--driver` を `--model` なしで渡すと、`--model` を名指しして断る。ただし役割の `model` が `""` なら、渡した driver を CLI の既定のモデルで立てる。
  - `[org.role_models]` にエントリのない役割では、`--driver` と `--model` の両方が要る。どちらかが欠けたら、役割の名前と `[org.role_models].<role>` を名指しして断る。
  - spawn と director start の `--driver` は必須でなくなる。start の `--driver` の既定(今は `claude`)は、leader の役割の driver になる。
  - `--model` を省いたときの警告(`org: --model omitted; falling back to ...`)はなくす。spawn と start の出力行は、今と同じく driver とモデルを出す。モデルが `""` のときは、そうと分かる表示にする。
- [ ] **FR-4 読み先**: `[org.role_models]` と `[org].driver_pool` は、`max_orgs` と同じ規則で読む。`--config` がなく、台帳が main worktree のものなら、main worktree のルートの `ralph.toml`(と `ralph.local.toml`)から読む。それ以外は `--config` のファイルか `./ralph.toml` から読む。feature worktree の写しは既定を決めない。director start は今と同じく main worktree から読む。
- [ ] **FR-5 検証**: `config.Load` は次を断る。どれも、エラーにファイルと役割の名前を出す。
  - エントリに `driver` か `model` のキーがないもの。検査は `ralph.toml` と `ralph.local.toml` のそれぞれで行い、上書きしたあとの結果では行わない(go-toml/v2 は、インライン表のエントリを丸ごと置き換え、表見出しのエントリはキーごとにマージする。両方のキーを必須にすると、どちらの書き方でも結果が同じになる)。
  - 明示したエントリの `driver` が、実際に使われる `driver_pool` にないもの。
  - `model` に `[reviewer].claude_model` と同じ文字(`A-Z a-z 0-9 . _ : @ / + -`)以外を含むもの、または `-` で始まるもの(`""` は許す)。

  `--model` の値も同じ文字の規則で、herdr を呼ぶ前に断る。
- [ ] **FR-6 廃止**: `[org].model_pool` と `[org.roles]` は読まない。
  - `model_pool` が残っていても、読み込みは通る。spawn・start・director start は stderr に、もう読まないことを 1 行で出す。`ralph doctor` は warn を出す。
  - `[org.roles]` が残っていれば、spawn・start・director start は何も記録せずに断る。読み込みは通るので、`status`・`stop`・`disband` などは動く。`ralph doctor` は、もう制限しないことを warn で出す。改名した役割の古い名前のキーで spawn を断る前例(`internal/org/prompts.go` の `RetiredRoleConfigKeys`)と同じ考え方で、何の知らせもなく制限が外れることを防ぐ。
  - `[org.role_models]` に改名した役割の古い名前(`lead`)があれば、`[org.roles]` の今の扱いと同じく spawn が断り、新しい名前を示す。
- [ ] **FR-7 doctor**: 次の 4 つを変える。
  - 「Org codex model slugs」の検査は、実際に使われる `[org.role_models]` の codex のエントリのうち `model` が空でないものを対象にする。
  - `--probe-models` は、実際に使われる `[org.role_models]` の組を(重複を除いて)起動して確かめる。`model` が `""` の組は `--model` なしで起動する。
  - codex writable root の検査で「codex の座席が立ちうるか」を決める条件は、`driver_pool` に `codex` があることにする(`--driver codex --model <任意>` で立てられるため)。
  - envelope の要約(doctor の行と leader の雛形の `{{ENVELOPE}}`)は、`model_pool` の代わりに役割ごとの driver とモデルを出す。
- [ ] **FR-8 init**: 片方の CLI だけを選んだ `ralph init` は `driver_pool` だけを書き換える。`model_pool` の行をコメントにする処理と、そのエラー(`template ralph.toml has no [org].model_pool = [ line` など)は消す。
- [ ] **FR-9 手順書**: 次の文書を直す。
  - leader の雛形: 座席は `--driver` と `--model` を省いて立て、`[org.role_models]` に任せる。分割計画の前文に、その座席の driver かモデルが書いてあるときだけ渡す。「`--model` は必ず明示する」の規則は消す。
  - director の雛形と `/director` skill: 分割計画の前文に書く座席の driver とモデルは、`ralph.toml` と違うものにする座席だけにする。`ralph org start --plan` の例から `--driver` と `--model` を外す。
  - `/org` skill: 「既定の model_pool」節を `[org.role_models]` の節に置き換え、`spawn`・`start`・`director` の行の `--driver` と `--model` の説明を直す。
  - `.claude/rules/ralph/model-routing.md` の「Org runtime model receipts」節、`docs/specs/2026-08-01-org-runtime.md` の FR-2 の改訂注記、README と recipes の `model_pool` の記述を直す。skill と rule は `.claude/`・`.agents/`・`templates/base/` の写しをそろえる。

### Non-functional requirements

- [ ] **互換性**: 残った `model_pool` や `[org.roles]` で `config.Load` は失敗しない。`ralph org status`・`stop`・`disband` は、どちらが残っていても動く。
- [ ] **ロックステップ**: 既定値は `config.Default()` と `templates/base/ralph.toml` のコメントの 2 か所に置き、`defaults_sync_test.go` がずれを検出する。
- [ ] **テスト**: 解決の順(FR-3)、検証(FR-5)、廃止の扱い(FR-6)、読み先(FR-4)に Go のテストを付ける。境界として、モデルが `""` の役割、エントリのない独自の役割、`ralph.local.toml` のインラインと表見出しの両方を含める。
- [ ] **文書**: `scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る。

## Acceptance criteria

- [ ] Given `[org.role_models]` を書いていない既定の設定, when `ralph org spawn --role reviewer` を `--driver` と `--model` なしで打つ(`--dry-run`), then 座席は claude / `opus` で記録される。
- [ ] Given `driver_pool = ["codex"]` だけを書いた設定, when implementer をフラグなしで spawn する, then 座席は codex / `gpt-6.1-sol` で記録される。
- [ ] Given `reviewer = { driver = "codex", model = "gpt-6-astra" }`, when `--model gpt-6.1-sol` だけを渡して reviewer を spawn する, then 座席は codex / `gpt-6.1-sol` になる。
- [ ] Given 役割が claude / `opus`, when `--driver codex` だけを渡す, then spawn は `--model` を名指しして断り、何も記録しない。
- [ ] Given 既定の設定, when `ralph org director start` をフラグなしで打つ, then director は claude で `--model` なしで起動する。
- [ ] Given `[org.role_models]` にエントリのない独自の役割, when `--driver` か `--model` を欠いて spawn する, then 役割の名前と `[org.role_models].<role>` を名指しして断る。
- [ ] Given `ralph.local.toml` の `[org.role_models.reviewer]` に `model` だけを書いた設定, when どの `ralph org` の動詞を打っても, then `config.Load` は `ralph.local.toml` と `reviewer` を名指しして失敗する。両方のキーを書けば reviewer だけが置き換わり、ほかの役割は既定のまま。
- [ ] Given `driver_pool = ["claude"]` と `reviewer = { driver = "codex", model = "gpt-6.1-sol" }`, when 設定を読む, then `config.Load` は driver を名指しして失敗する。
- [ ] Given `model_pool` を残した `ralph.toml`, when spawn する, then spawn は通り、stderr に 1 行出る。`ralph doctor` は warn を出す。
- [ ] Given `[org.roles]` を残した `ralph.toml`, when spawn・start・director start を打つ, then 何も記録せずに断る。`ralph org stop` と `disband` は通り、`ralph doctor` は warn を出す。
- [ ] Given `--model "a b"` か `--model -x`, when spawn する, then herdr を呼ぶ前に断る。
- [ ] Given feature worktree の `ralph.toml` の `[org.role_models]` が main worktree のものと違う機能の org, when その leader が `--config` なしでフラグなしの spawn をする, then main worktree の値が使われる。
- [ ] Given `reviewer = { driver = "codex", model = "<cache にないスラッグ>" }`, when `ralph doctor` を打つ, then 「Org codex model slugs」がそのスラッグを warn で出す。
- [ ] Given `config.Default()` の既定とテンプレートのコメントの値の食い違い, when `go test ./internal/config/` を打つ, then `defaults_sync_test.go` が失敗する。

## User stories

1. As repo の持ち主, I want to `ralph.toml` に役割ごとのモデルを 1 回書く, so that 機能の org を立てるたびに leader と director がモデルを選ばずに済む。
2. As 分割計画を director と作る人, I want to 前文には既定と違う座席だけを書く, so that 計画の前文が短くなり、既定を変えるときは `ralph.toml` だけを直せばよい。
3. As codex だけを使う repo の持ち主, I want to `driver_pool = ["codex"]` と書くだけで codex の既定が入る, so that モデルの表を codex 用に書き直さずに済む。
4. As 1 台のマシンで別のモデルを試す人, I want to `ralph.local.toml` で 1 つの役割だけを上書きする, so that 共有の `ralph.toml` を変えずに試せる。

## Constraints

### In scope

- `internal/config`: `[org.role_models]` の型、既定の 2 組、検証、`model_pool` と `[org.roles]` の除去、`ralph.local.toml` の検査。
- `internal/org`: spawn の検証(`ValidateSpawnEnvelope`、`ModelAllowedForRole`、`DefaultModelForDriverAndRole`)、`RetiredRoleConfigKeys`、envelope の要約、leader と director の雛形。
- `internal/cli`: spawn・start・director start のフラグと解決、読み先、doctor の 3 つの検査、init の片方の CLI の処理。
- `templates/base/ralph.toml`、skill(`/org`、`/director`)、rule(`model-routing.md`)、README、recipes、org-runtime spec の改訂注記。

### Out of scope

- `[org.watchdog].watcher_model`(役割ではない)。
- `[reviewer]` の 4 キー(計画の助言と cross-review の相手のモデル)。
- `ralph config get` での `[org.role_models]` の取得(表は今も exit 2)。
- `ralph insights` の集計と receipt の項目(receipt には指示したモデルが今どおり残る)。
- spawn の `--leader-driver`(leader の識別子の agmsg の種類)の既定。
- 座席のモデルにコストの上限をかける仕組み。

## Impact

| Target | Impact | Severity |
|--------|--------|----------|
| `internal/config/config.go`、`local.go` | `OrgConfig` の `ModelPool` と `Roles` を除き、`RoleModels` を足す。既定の 2 組、ファイルごとの検査、`orgPoolKeysPresent` と `filterModelPoolByDrivers` の置き換え | High |
| `internal/org/envelope.go`、`envelope_summary.go`、`prompts.go`、`spawn.go` | 許可リストの検査をなくし、文字種の検査と `[org.roles]` の拒否を足す。要約と `RetiredRoleConfigKeys` の対象を変える | High |
| `internal/cli/org.go`、`org_director_headless.go` | `--driver` を任意にし、役割の既定で解決する。読み先を main worktree にそろえる | High |
| `internal/cli/doctor*.go` | 検査の対象を role_models と driver_pool に変え、残ったキーの warn を足す | Medium |
| `internal/cli/init_settings.go` | `model_pool` の処理を消す | Medium |
| `internal/org/prompts/leader.md`、`director.md` | `--driver` と `--model` の規則を変える | Medium |
| 下流の `ralph.toml` | テンプレートから写した `model_pool` が残る(警告だけ)。`[org.roles]` を書いた repo は、消すまで spawn が断られる | Medium |
| skill・rule・docs(`.claude/`・`.agents/`・`templates/base/` の写し) | `model_pool` と `[org.roles]` の記述を置き換える | Low |

## Dependencies

- codex のスラッグ `gpt-6.1-sol` が codex の `models_cache.json` にあること(2026-11-05 に手元で確認した。表示名は「GPT-6.1-Sol」、説明は「Latest workhorse model for coding and everyday work.」)。
- go-toml/v2 v2.3.0 のマージの挙動(Research findings に記録した)。

## Research findings

### Codebase analysis

- `--model` の既定: `internal/cli/org.go` の `resolveModelOrWarn` が `org.DefaultModelForDriverAndRole` を呼び、プール先頭のうち役割が許すものを警告付きで返す。spawn・start・director start が共有している。
- 許可リスト: `internal/org/envelope.go` の `ValidateSpawnEnvelope` が driver_pool、model_pool、`[org.roles]` の順に検査する。director start は座席ではないので `ModelAllowedForRole` だけを使い、`model_pool` は検査しない。
- 読み先: `applyMainWorktreeOrgLimits` が main worktree から取るのは `max_orgs`・`max_total_seats`・`[org.approval]` だけで、spawn の `model_pool` は作業ディレクトリの設定から読む。director start の `withMainWorktreeDirectorStartConfig` は `model_pool` と `[org.roles]` を main worktree から読み、コメントで「leader が書き換えられる feature worktree の写しに director のモデルを決めさせない」と理由を書いている。
- フラグ: spawn の `--driver` は必須、start の `--driver` は既定 `claude`、director start の `--driver` は必須。
- 前例: `RetiredRoleConfigKeys`(`internal/org/prompts.go`)は、改名した役割の古い名前のキーを spawn で断り、`config.Load` は止めない。読まれないキーが通ると、そこに書いた制限が効かなくなるためで、`stop` と `disband` は動き続ける。
- go-toml/v2 の挙動(2026-11-05 に一時モジュールで確かめた): 既存の map への 2 回目の `Unmarshal` は、キー(役割)ごとにマージする。インライン表のエントリ(`reviewer = { model = "opus" }`)はエントリを丸ごと置き換え、`driver` は空になる。表見出しのエントリ(`[org.role_models.reviewer]` に `model = "opus"`)はキーごとにマージし、前の `driver` が残る。go-toml/v2 は未知のキーをエラーにせず読み飛ばす。
- `model_pool` を参照する追跡ファイルは、`docs/reports`・`docs/plans/archive`・`docs/insights` を除いて約 49 ある(Go のコード 30 前後、skill・docs・evidence)。

### Best practices

- `[reviewer]` の 4 キーと `[org.watchdog].watcher_model` が、機能ごとにモデルを設定で持つ前例になっている。
- `model-routing.md` の tier 表は、標準フローの subagent に実装は `opus`、レビューは `sonnet` を当てる。org の既定は人の判断で reviewer も `opus` にした(tier 表とは別の仕組み)。

### Alternatives considered and trade-offs

| Option | Pros | Cons | Adopted |
|--------|------|------|---------|
| `model_pool` と `[org.roles]` を両方廃止し、役割の既定だけにする | 設定の考え方が 1 つで済む。既定のプールは何も制限していなかった | 綴り違いが座席の起動後まで見つからない。プールを絞ってコストを抑えていた repo はその制限を失う | Yes |
| `model_pool` を廃止し、`[org.roles]` を `--model` の許可リストとして残す | 役割ごとにコストを縛れる | 役割の表が 2 つになる | No |
| `model_pool` を許可リストとして残す | 今の検査がすべて残る | プール・許可リスト・既定の 3 つを保つ | No |
| エントリを driver とモデルの組にする | 1 行で役割の driver とモデルが決まり、`--driver` も省ける | 役割と違う driver を使うときは `--model` も要る | Yes |
| エントリを driver ごとのモデルにする | driver を替えても `--model` が要らない | 役割の driver の既定を持たず、`--driver` を毎回選ぶ | No |
| エントリをモデル名だけにする | 書く量が少ない | driver と合わないモデル名を書けてしまう | No |
| codex だけの repo の既定を `driver_pool` から導く | init の書き換えが要らない。`driver_pool` だけを直した人もそのまま動く | 組み込みの既定を 2 組保つ | Yes |
| codex だけの repo では init が `role_models` を書き換える | 組み込みの既定が 1 組で済む | あとから `driver_pool` を手で直すと既定がなくなる | No |

## Security considerations

- `--model` と設定の `model` は herdr の pane で CLI に渡る。FR-5 の文字種の検査で、空白・引用符・`;`・`$` を含む値と `-` で始まる値を断り、引数やシェルへの注入を防ぐ。
- leader(AI)は `--model` で任意のモデルを選べるようになる。ただし今の `model_pool` も leader への強い制限ではなかった。機能の org の spawn は、leader が書き換えられる feature worktree の `ralph.toml` から `model_pool` を読んでいた。FR-4 で役割の既定は main worktree から読むので、leader が既定を書き換えることはできなくなる。どのモデルを指示したかは、今どおり model receipt(`.harness/state/org/model-receipts.jsonl`)に残る。
- `[org.roles]` で役割を縛っていた repo は、気づかないうちにその制限を失わないよう、消すまで spawn が断られる(FR-6)。

## Open questions

なし。

## References

- `docs/specs/2026-08-01-org-runtime.md`(FR-2 の `[org]` エンベロープ、運用ノートの (e))
- `docs/specs/2026-10-07-org-multi-org-director.md`
- `.claude/rules/ralph/model-routing.md`
- `internal/cli/org.go`、`internal/org/envelope.go`、`internal/org/envelope_summary.go`、`internal/org/prompts.go`、`internal/config/config.go`
