package org

// TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase ties the /director
// skill's inbox section to WaitInbox's timeout error. The skill tells a
// timeout of a bounded `ralph org wait --inbox` from a failure by finding
// inboxWaitTimeoutPhrase in stderr, and reruns the wait only on a match, so
// a change to one text without the other makes the director stop at every
// timeout. scripts/check-skill-sync.sh keeps the four copies of the skill
// identical but does not read Go, so nothing else catches that drift. It
// follows send_defaults_sync_test.go's repo-root-via-runtime.Caller pattern.
//
// If this test fails, either restore inboxWaitTimeoutPhrase in escalate.go,
// or write the new phrase, in backquotes, in the inbox section
// (受信箱の処理) of .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies (root, then with
// CLAUDE_ROOT=templates/base/.claude/skills
// CODEX_ROOT=templates/base/.agents/skills).

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// directorSkillRepoRoot resolves the repository root from this source file's
// location: internal/org/ -> ../../ (as sendDefaultsRepoRoot does).
func directorSkillRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase(t *testing.T) {
	f := newEscalateFixture(t)
	f.o.InboxPollInterval = 10 * time.Millisecond
	out := runWaitInbox(t, f.o, 20*time.Millisecond, 5*time.Second)
	if out.err == nil || !strings.Contains(out.err.Error(), inboxWaitTimeoutPhrase) {
		t.Fatalf("WaitInbox timeout error = %v; want it to contain inboxWaitTimeoutPhrase %q", out.err, inboxWaitTimeoutPhrase)
	}

	root := directorSkillRepoRoot(t)
	want := "`" + inboxWaitTimeoutPhrase + "`"
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			// A repo that vendors internal/org without the skills tree, as
			// in TestSendDefaults_DocsMatchConstant.
			t.Skipf("%s not found (%v) -- skipping the skill/error lock-step check (vendored repo?)", rel, err)
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s does not contain %s, the phrase of WaitInbox's timeout error -- see this test's doc comment for what to update", rel, want)
		}
	}
}

// directorSkillCopies are the four copies of the /director skill, relative
// to the repository root.
var directorSkillCopies = []string{
	filepath.Join(".claude", "skills", "director", "SKILL.md"),
	filepath.Join(".agents", "skills", "director", "SKILL.md"),
	filepath.Join("templates", "base", ".claude", "skills", "director", "SKILL.md"),
	filepath.Join("templates", "base", ".agents", "skills", "director", "SKILL.md"),
}

// directorSkillAutoMergeSteps are fragments of the /director skill's merge
// steps (its section merge と auto_merge), in the order the skill has them:
// with auto_merge = false the director merges and regates only when the
// human asks; with auto_merge = true it merges the first ready branch with
// ralph org merge, run with --wait-checks 60m in the background and read once
// it ended (plan D13: the verb waits for checks still pending), runs
// merge-order again and regates one branch, and merges that branch once it
// is ready again; with either value it never runs gh pr merge itself, and it
// stops a branch at a refusal or a failure of the verb and does not run it
// again unasked. The last two say that --wait-checks also waits for the
// branch's org to be disbanded after its RESULT (D5), and that the director
// does not disband an org that still runs once the time is up.
var directorSkillAutoMergeSteps = []string{
	"merge も regate も人に頼まれたときだけ打つ",
	"先頭の `ready` を `ralph org merge --branch <ブランチ>` で 1 本 merge する",
	"`--wait-checks 60m` を付けて background で打ち、終わってから終了コードと出力を読む",
	"merge-order を打ち直し、理由の欄が `ralph org regate --branch <ブランチ>` を名指すブランチの 1 本に regate を打つ",
	"そのブランチが `ready` に戻っていれば、2 と同じく merge して 3 に戻る",
	"`gh pr merge` を直接打たない。merge は `ralph org merge` でだけ打つ",
	"そのブランチはそこで止め、stderr の文(理由と次の手)をそのまま人に伝える。人に頼まれないまま打ち直さない",
	"動詞はこの断りでも、同じく `regate.lock` を外し、30 秒ごとに台帳だけを読み直して(git と gh は打たない)org の disband を待つ",
	"director は disband を打たずに、その断りを人に伝える",
}

// directorSkillOldMergeStep is a fragment of the step the skill had before
// [org.director].auto_merge, which merged with gh pr merge directly.
const directorSkillOldMergeStep = "`gh pr merge` で 1 本 merge し"

// withoutSpaces drops every space and line break from s, so a fragment of
// the skill matches however its paragraph is wrapped.
func withoutSpaces(s string) string { return strings.Join(strings.Fields(s), "") }

// TestDirectorSkill_PinsTheAutoMergeSteps pins the /director skill's merge
// steps, which meet AC11 of plan docs/plans/active/2026-10-17-org-auto-merge.md
// and FR-8's acceptance criterion (with auto_merge = false the director does
// not merge): ralph org merge does not read [org.director].auto_merge to
// refuse (plan D1), so the skill's text is all that keeps a director from
// merging unasked, and the merge of auto_merge = true carries
// --wait-checks 60m, without which it stops at checks still pending. Each
// copy must contain every fragment of
// directorSkillAutoMergeSteps, each after the one before it, compared with
// spaces and line breaks removed (withoutSpaces), and must not contain
// directorSkillOldMergeStep. scripts/check-skill-sync.sh keeps the copies
// identical but does not read Go.
//
// If this test fails after a deliberate change to the steps, put the new
// fragment in directorSkillAutoMergeSteps in the skill's order; otherwise
// restore the text in the section merge と auto_merge of
// .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies, as for the test above.
func TestDirectorSkill_PinsTheAutoMergeSteps(t *testing.T) {
	root := directorSkillRepoRoot(t)
	old := withoutSpaces(directorSkillOldMergeStep)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's merge-step check (vendored repo?)", rel, err)
		}
		text := withoutSpaces(string(data))
		at := 0
		for _, step := range directorSkillAutoMergeSteps {
			want := withoutSpaces(step)
			i := strings.Index(text[at:], want)
			if i < 0 {
				t.Errorf("%s does not contain %q after the fragments before it in directorSkillAutoMergeSteps -- see this test's doc comment for what to update", rel, step)
				continue
			}
			at += i + len(want)
		}
		if strings.Contains(text, old) {
			t.Errorf("%s still contains %q, the step that merged with gh pr merge directly -- see this test's doc comment", rel, directorSkillOldMergeStep)
		}
	}
}
