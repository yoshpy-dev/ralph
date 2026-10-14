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
	for _, rel := range []string{
		filepath.Join(".claude", "skills", "director", "SKILL.md"),
		filepath.Join(".agents", "skills", "director", "SKILL.md"),
		filepath.Join("templates", "base", ".claude", "skills", "director", "SKILL.md"),
		filepath.Join("templates", "base", ".agents", "skills", "director", "SKILL.md"),
	} {
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
