package org

import (
	"strings"
	"testing"
)

// implementerPredatingChanges is the one name the implementer template gives
// the changes that were in the git status --porcelain the seat records before
// its first edit for each TASK.
const implementerPredatingChanges = "前からあった変更"

// TestRenderRolePrompt_Implementer_RecordsPredatingChangesAtTheStartOfEachTask
// pins the term the fixes for cross-review cycle 1 #1 of plan
// 2026-10-27-org-narrow-verification rest on (cycle 2 self-review LOW 1): the
// implementer's mission runs git status --porcelain before the first edit of
// each TASK, records it, and defines the changes it shows (a staged one too)
// once as implementerPredatingChanges. The seat stays up across TASKs, so
// what an earlier TASK left uncommitted is one of them. The template calls
// them by that name only: the other names it once used (before the seat
// started, before the slice, existing changes) must not come back.
func TestRenderRolePrompt_Implementer_RecordsPredatingChangesAtTheStartOfEachTask(t *testing.T) {
	text := renderSeatPrompt(t, "implementer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "TASK ごとに、その TASK の最初の編集の前に"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the implementer mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	for _, want := range []string{
		"TASK ごとに、その TASK の最初の編集の前に `git status --porcelain` を実行し、結果を記録する",
		"この記録に出ていた変更(stage 済みのものも含む)を、以下「" + implementerPredatingChanges + "」と呼ぶ",
		"この座席は TASK をまたいで常駐するので、前の TASK がコミットせずに残した変更も" + implementerPredatingChanges + "に入る",
		"scope 外の" + implementerPredatingChanges + "は報告のうえ触らずそのままにする(ステージしない)",
		"もし scope と重なる" + implementerPredatingChanges + "を見つけたら、実装を始めず BLOCKED で leader に報告する",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the implementer mission item %q to say %q, got item:\n%s", marker, want, item)
		}
	}
	if n := strings.Count(squashSpace(text), "「"+implementerPredatingChanges+"」"); n != 1 {
		t.Errorf("expected the implementer template to define %q once, found %d definitions:\n%s", implementerPredatingChanges, n, text)
	}
	for _, old := range []string{"編集を始める前に記録した", "座席が始める前", "スライスより前", "既存の変更"} {
		if containsPhrase(text, old) {
			t.Errorf("expected the implementer template to call the changes in its record %q only, but it still says %q:\n%s", implementerPredatingChanges, old, text)
		}
	}
}

// TestRenderRolePrompt_Implementer_SelfCheckLeavesChangesThatPredateTheSlice
// pins the fix for cross-review cycle 1 #1 of plan
// 2026-10-27-org-narrow-verification, as cycle 2 self-review LOW 2 and 3
// reworded it: the implementer's mechanical self-check fixes only the slice's
// own changes and neither edits, deletes nor unstages implementerPredatingChanges
// (a staged one too). It unstages with git restore --staged naming the slice's
// own paths, never with a git reset without paths, which would unstage the
// staged changes that predate the slice too. It lists the three cases in
// which it does not commit, since a commit takes everything staged, each with
// a BLOCKED: a secret in the slice's own changes it cannot remove within
// scope, which leaves the index and the worktree (a file is deleted only when
// the slice created it); a secret in a change that predates the slice; and a
// path outside the scope among the staged changes that predate the slice. It
// touches nothing in the last two, the second's BLOCKED saying the change
// predates the slice and the third's naming the path, while a path the slice
// staged by mistake is unstaged. The BLOCKED of the first two names the file
// and line, never the value. The old wording, which cleaned up any secret in
// the staged diff, deleted any new untracked file, and counted two
// exceptions, must not come back.
func TestRenderRolePrompt_Implementer_SelfCheckLeavesChangesThatPredateTheSlice(t *testing.T) {
	text := renderSeatPrompt(t, "implementer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "コミットの前に毎回、stage した差分"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the implementer mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	steps := []string{
		"直すのはこのスライスの変更だけにする",
		implementerPredatingChanges + "は、stage 済みのものも含めて、書き換えも削除も unstage もしない",
		"unstage するときは、このスライスのパスを指定して `git restore --staged -- <このスライスのパス>...` を打ち",
		"パスを付けない `git reset` は打たない(" + implementerPredatingChanges + "の stage まで外れる)",
		"コミットしないのは次の 3 つのときで(コミットには stage してあるものがすべて入る)、どれも BLOCKED を leader に返す: " +
			"このスライスの変更にある秘密情報を scope の中で消せないとき、" +
			implementerPredatingChanges + "に秘密情報があるとき、" +
			"stage 済みの" + implementerPredatingChanges + "に scope の外のパスがあるとき",
		"1 つ目では、このスライスのパスを unstage し、作業ツリーからも出す(値を消す。その値を持つのがこのスライスで作った未追跡のファイルなら、ファイルを消す)",
		"2 つ目と 3 つ目では何も触らず、2 つ目の BLOCKED には" + implementerPredatingChanges + "だと書き、3 つ目の BLOCKED にはそのパスを書く",
		"(このスライスが誤って stage したパスなら、unstage すれば直る)",
		"1 つ目と 2 つ目の BLOCKED には秘密情報のファイルと行を書き、値は書かない",
	}
	s := squashSpace(item)
	at := -1
	for _, step := range steps {
		i := strings.Index(s, squashSpace(step))
		if i < 0 || i < at {
			t.Errorf("expected the implementer self-check item to say %q in the order %q, but it is missing or out of order, got item:\n%s", step, steps, item)
			break
		}
		at = i
	}
	for _, old := range []string{"例外は 2 つ", "例外は秘密情報だけ", "新しい未追跡のファイル"} {
		if containsPhrase(item, old) {
			t.Errorf("expected the implementer self-check item to drop the old wording %q, got item:\n%s", old, item)
		}
	}
}

// TestOrgSkill_ImplementerBulletLeavesChangesThatPredateTheSlice pins the
// /org skill side of the same fix: the ## 役割 section's implementer bullet
// says the seat fixes only the slice's own changes and, for a secret in
// implementerPredatingChanges (the changes in the git status --porcelain
// before the first edit of each TASK) or a path outside the scope staged
// among them, touches nothing, does not commit and returns a BLOCKED, as the
// implementer's self-check item does. Like
// TestOrgSkill_RoleBulletsNameEachSliceAndReadOnlyTheChecksLine, it reads the
// root copy only.
func TestOrgSkill_ImplementerBulletLeavesChangesThatPredateTheSlice(t *testing.T) {
	const rel = ".claude/skills/org/SKILL.md"
	text := readRepoFileOrSkip(t, rel)
	roles, found := markdownSection(text, "## 役割")
	if !found {
		t.Fatalf("%s: expected a '## 役割' section, found none", rel)
	}
	const marker = "- **implementer**:"
	const want = "直すのはそのスライスの変更だけで、" + implementerPredatingChanges +
		"(TASK ごとに、最初の編集の前の `git status --porcelain` に出ていた変更)の秘密情報や、" +
		"その中で stage 済みの scope の外のパスには触らず、コミットせずに BLOCKED を返す"
	item, ok := markdownItem(roles, marker)
	if !ok || !containsPhrase(item, want) {
		t.Errorf("%s: expected the ## 役割 bullet %q to say %q, got item:\n%s", rel, marker, want, item)
	}
	if containsPhrase(item, "座席が始める前") {
		t.Errorf("%s: expected the ## 役割 bullet %q to drop the old wording %q, got item:\n%s", rel, marker, "座席が始める前", item)
	}
}

// resumeLine is the body line of an implementer TASK on which the leader lists
// the paths a TASK that stopped without a commit left uncommitted, so that the
// implementer takes their changes over instead of stopping at them.
const resumeLine = "`RESUME:`"

// phrasesInOrder returns the first of phrases that text does not say after the
// phrase before it, ignoring white space as containsPhrase does, or "" when
// text says them all in that order.
func phrasesInOrder(text string, phrases []string) string {
	s := squashSpace(text)
	at := 0
	for _, p := range phrases {
		i := strings.Index(s[at:], squashSpace(p))
		if i < 0 {
			return p
		}
		at += i + len(squashSpace(p))
	}
	return ""
}

// TestRenderRolePrompt_Leader_ResendsAStoppedSliceWithAResumeLine pins the
// leader side of the org's resume of a slice that stopped without a commit
// (plan 2026-10-27-org-narrow-verification, the org form of /implement step
// 7's resume): mission item 3 has the leader that sends the same slice again,
// after the implementer stopped with a BLOCKED and no commit, list on a
// resumeLine of the TASK body (a body line, like VERIFY:, not a header) the
// paths in scope the stopped TASK left uncommitted, taken from the BLOCKED and
// git status --porcelain. After a secret stop it first checks the value is
// gone from the worktree, and when it is not, or the BLOCKED says the secret
// is in implementerPredatingChanges, it escalates a BLOCKED instead of
// resuming, without fixing the code itself.
func TestRenderRolePrompt_Leader_ResendsAStoppedSliceWithAResumeLine(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the leader template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "3. `ralph org send`"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the leader mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	steps := []string{
		"implementer がコミットせずに BLOCKED で止まり、止まった原因を取り除いて(またはスライスを狭めて)同じスライスをもう一度頼むときは、" +
			"止まった TASK がコミットせずに残した scope の中のパスを、BLOCKED と `git status --porcelain` から拾って本文の " + resumeLine + " の行に並べる",
		"(例: `RESUME: internal/foo/bar.go internal/foo/bar_test.go`。`VERIFY:` と同じく本文の行で、ヘッダにはしない)",
		"implementer はそのパスの変更を引き継いで続ける",
		"スライスを狭めるときも、残したパスはすべて並べ、外すパスの変更は元に戻すよう TASK に書く(並べなかったパスは前からあった変更として残り、implementer がまた止まる)",
		"秘密情報で止まったあとは、送る前に `git diff HEAD -- <パス>...`(未追跡のファイルは中身を読む)で、その値が作業ツリーから消えたことを確かめる",
		"消えていないとき、また BLOCKED が秘密情報を" + implementerPredatingChanges + "(止まった TASK より前の変更。implementer は触らない)にあると書いたときは、" +
			"再開せず `ralph org escalate` で BLOCKED を上げる(コードは自分で直さない)",
	}
	if missing := phrasesInOrder(item, steps); missing != "" {
		t.Errorf("expected the leader mission item %q to say %q in the order %q, but it is missing or out of order, got item:\n%s", marker, missing, steps, item)
	}
}

// TestRenderRolePrompt_Implementer_TakesOverThePathsOfAResumeLine pins the
// implementer side of the same resume: right after its mission defines
// implementerPredatingChanges, which include what an earlier TASK left
// uncommitted, it takes the uncommitted changes at the paths a resumeLine of
// the TASK body lists over as this slice's own (continuing from them, staging
// and self-checking them with the rest of the slice). A listed path outside
// the scope is not taken over: the seat starts nothing and returns a BLOCKED
// naming it. The changes at paths the line does not list stay
// implementerPredatingChanges. The report contract has the RESULT's decisions
// name the paths it took over.
func TestRenderRolePrompt_Implementer_TakesOverThePathsOfAResumeLine(t *testing.T) {
	text := renderSeatPrompt(t, "implementer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "TASK ごとに、その TASK の最初の編集の前に"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the implementer mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	steps := []string{
		"前の TASK がコミットせずに残した変更も" + implementerPredatingChanges + "に入る",
		"ただし TASK の本文に " + resumeLine + " の行があれば、その行に挙がったパスのコミットされていない変更は" + implementerPredatingChanges +
			"に入れず、このスライスの変更として引き継ぐ(そこから続け、スライスのほかの変更と一緒に stage して機械的な確かめを行う)",
		resumeLine + " の行に scope の外のパスがあれば、引き継がずに実装を始めず、そのパスを書いた BLOCKED を leader に返す",
		resumeLine + " の行に挙がっていないパスの変更は、" + implementerPredatingChanges + "のままとする",
	}
	if missing := phrasesInOrder(item, steps); missing != "" {
		t.Errorf("expected the implementer mission item %q to say %q in the order %q, but it is missing or out of order, got item:\n%s", marker, missing, steps, item)
	}

	report, found := markdownSection(text, "## レポート契約")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## レポート契約' section, got:\n%s", text)
	}
	const contractMarker = "変更したファイル一覧"
	const want = "TASK の本文の " + resumeLine + " の行のパスを引き継いだときは、決定事項や逸脱に、引き継いだパスを書きます"
	if contract, ok := markdownItem(report, contractMarker); !ok || !containsPhrase(contract, want) {
		t.Errorf("expected the implementer report contract item %q to say %q, got item:\n%s", contractMarker, want, contract)
	}
}

// TestOrgSkill_ImplementerBulletAndLeaderCycleCarryTheResumeLine pins the /org
// skill side of the same resume: the ## 役割 section's implementer bullet
// takes over the changes at the paths of a resumeLine as the slice's own and
// returns a BLOCKED for one outside the scope, and step 3 of the leader cycle
// in ## Leader 運用 2 経路 lists on that line the paths in scope a stopped
// TASK left uncommitted, after checking that the value of a secret stop is
// gone. Like TestOrgSkill_RoleBulletsNameEachSliceAndReadOnlyTheChecksLine, it
// reads the root copy only.
func TestOrgSkill_ImplementerBulletAndLeaderCycleCarryTheResumeLine(t *testing.T) {
	const rel = ".claude/skills/org/SKILL.md"
	text := readRepoFileOrSkip(t, rel)
	for _, c := range []struct{ header, marker, want string }{
		{"## 役割", "- **implementer**:",
			"TASK の本文に " + resumeLine + " の行があれば、そこに挙がったパスのコミットされていない変更は" + implementerPredatingChanges +
				"ではなく、このスライスの変更として引き継ぐ(scope の外のパスがあれば、引き継がずに BLOCKED を返す)"},
		{"## Leader 運用 2 経路", "3. `send` で TASK を委譲する",
			"implementer がコミットせずに BLOCKED で止まったスライスをもう一度頼むときは、止まった TASK がコミットせずに残した scope の中のパスを本文の " +
				resumeLine + " の行に並べる(秘密情報で止まったあとは、値が作業ツリーから消えたことを先に確かめる)"},
	} {
		section, found := markdownSection(text, c.header)
		if !found {
			t.Errorf("%s: expected a %q section, found none", rel, c.header)
			continue
		}
		if item, ok := markdownItem(section, c.marker); !ok || !containsPhrase(item, c.want) {
			t.Errorf("%s: expected the %s item %q to say %q, got item:\n%s", rel, c.header, c.marker, c.want, item)
		}
	}
}
