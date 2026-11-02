package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs ties the
// documents that say when the org verbs that change the ledger are refused
// while RALPH_ORG_STATE_DIR is set to the one constant that the refusal, the
// doctor's warn, and the help of `ralph org escalate` share,
// retiredOrgStateDirRefusalCondition (the 397 row of docs/tech-debt/README.md).
// The README states it in prose, so it is compared with its backticks removed;
// the /org skill states it in one code span, which must stay on one line. The
// .agents copies of the skill are compared with these by
// scripts/check-skill-sync.sh, and the templates copies with the root ones by
// scripts/check-sync.sh.
func TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	for _, tc := range []struct {
		path         string
		dropBackTick bool
		want         string
	}{
		{"README.md", true, retiredOrgStateDirRefusalCondition},
		{filepath.Join(".claude", "skills", "org", "SKILL.md"), false, "`" + retiredOrgStateDirRefusalCondition + "`"},
		{filepath.Join("templates", "base", ".claude", "skills", "org", "SKILL.md"), false, "`" + retiredOrgStateDirRefusalCondition + "`"},
	} {
		t.Run(filepath.ToSlash(tc.path), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, tc.path))
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			text := string(data)
			if tc.dropBackTick {
				text = strings.ReplaceAll(text, "`", "")
			}
			if !strings.Contains(text, tc.want) {
				t.Errorf("%s does not contain the condition of retiredOrgStateDirRefusalCondition, %q (backticks removed = %v)",
					tc.path, tc.want, tc.dropBackTick)
			}
		})
	}
}
