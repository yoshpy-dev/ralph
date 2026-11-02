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
// retiredOrgStateDirRefusalCondition (the row of docs/tech-debt/README.md
// about the refusal condition of RALPH_ORG_STATE_DIR). The README states it in
// prose, so it is compared with its backticks removed; the /org skill states
// it in one code span, which must stay on one line. The .agents copies of the
// skill are compared with these by scripts/check-skill-sync.sh, and the
// templates copies with the root ones by scripts/check-sync.sh.
func TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	const (
		prose    = "prose"     // compared with the document's backticks removed
		codeSpan = "code span" // compared as one code span, backticks included
	)
	for _, tc := range []struct {
		path string // relative to the repository root
		form string // prose or codeSpan: how the document states the condition
	}{
		{path: "README.md", form: prose},
		{path: filepath.Join(".claude", "skills", "org", "SKILL.md"), form: codeSpan},
		{path: filepath.Join("templates", "base", ".claude", "skills", "org", "SKILL.md"), form: codeSpan},
	} {
		t.Run(filepath.ToSlash(tc.path), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, tc.path))
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			text, want := string(data), retiredOrgStateDirRefusalCondition
			switch tc.form {
			case prose:
				text = strings.ReplaceAll(text, "`", "")
			case codeSpan:
				want = "`" + want + "`"
			default:
				t.Fatalf("unknown form %q", tc.form)
			}
			if !strings.Contains(text, want) {
				t.Errorf("%s does not contain %q as %s (the condition of retiredOrgStateDirRefusalCondition)",
					tc.path, want, tc.form)
			}
		})
	}
}
