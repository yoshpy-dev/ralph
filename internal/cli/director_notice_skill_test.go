package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// directorNoticeSkillHeading is the heading of the subsection of the
// /director skill, under its 進捗の報告 section, that tells the director what
// to do with the two note lines the test below ties to it.
const directorNoticeSkillHeading = "### 人に向けた知らせの行"

// TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines ties the two
// human-facing note lines ralph's verbs print on stderr to the /director
// skill's subsection about them (issue #318; plan
// 2026-11-13-director-notice-handling, AC7). The skill quotes a fragment of
// each line: `run 'ralph upgrade' in `, which the line of
// scaffoldVersionNotice ends with before the directory to upgrade, and
// `; run 'ralph doctor' for `, from the line of codexHookTrustNotice. The
// test builds each line with its production function, checks that the line
// holds its fragment (and, for the scaffold line, ends with it and the
// absolute root), and checks that the subsection of each of the skill's four
// copies quotes both fragments verbatim. The format strings stay in the
// production functions, so the fragments are written here: a change to
// either line's wording fails this test until the fragment here and the
// skill follow it.
func TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines(t *testing.T) {
	const (
		upgradeFragment = "run 'ralph upgrade' in "
		doctorFragment  = "; run 'ralph doctor' for "
	)

	root := t.TempDir()
	writeScaffoldVersionManifest(t, root, "[meta]\nversion = \"7.0.0\"\n")
	versionLine := scaffoldVersionNotice("org start", root, "8.1.0")

	f := newCodexHookTrustDoctorRepo(t)
	env, err := f.resolveEnv()
	if err != nil {
		t.Fatalf("resolveEnv: %v", err)
	}
	report := codexHookTrustFor(f.repo, env)
	hookLine := codexHookTrustNotice(report, codexConfigDisplayPath(env.Home, env.ConfigPath), "org spawn", "implementer")

	fragments := []struct {
		fragment string
		line     string // the line the production function returned
		from     string // the production function that words the line
	}{
		{fragment: upgradeFragment, line: versionLine, from: "scaffoldVersionNotice"},
		{fragment: doctorFragment, line: hookLine, from: "codexHookTrustNotice"},
	}
	for _, fr := range fragments {
		if fr.line == "" {
			t.Fatalf("%s returned no line for the fixture; want one holding %q", fr.from, fr.fragment)
		}
		if !strings.HasPrefix(fr.line, "note: ") {
			t.Errorf("the line of %s = %q; want it to start with %q", fr.from, fr.line, "note: ")
		}
		if !strings.Contains(fr.line, fr.fragment) {
			t.Errorf("the line of %s = %q; want it to hold %q, the fragment the /director skill quotes "+
				"(change the fragment here and in the skill's %s subsection together)",
				fr.from, fr.line, fr.fragment, directorNoticeSkillHeading)
		}
	}
	dir, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("abs %s: %v", root, err)
	}
	if want := upgradeFragment + dir; !strings.HasSuffix(versionLine, want) {
		t.Errorf("the line of scaffoldVersionNotice = %q; want it to end with %q, since the /director skill "+
			"reads the directory to upgrade as the last word of the line", versionLine, want)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	for _, path := range []string{
		filepath.Join(".claude", "skills", "director", "SKILL.md"),
		filepath.Join(".agents", "skills", "director", "SKILL.md"),
		filepath.Join("templates", "base", ".claude", "skills", "director", "SKILL.md"),
		filepath.Join("templates", "base", ".agents", "skills", "director", "SKILL.md"),
	} {
		t.Run(filepath.ToSlash(path), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, path))
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			section, ok := directorNoticeSkillSubsection(string(data), directorNoticeSkillHeading)
			if !ok {
				t.Fatalf("%s has no %q subsection", path, directorNoticeSkillHeading)
			}
			for _, fr := range fragments {
				if !strings.Contains(section, fr.fragment) {
					t.Errorf("%s: the %q subsection does not quote %q, the fragment of the line %s prints; "+
						"when that line's wording changes, update the four copies of the /director skill to match",
						path, directorNoticeSkillHeading, fr.fragment, fr.from)
				}
			}
		})
	}
}

// directorNoticeSkillSubsection returns the part of the markdown text from
// the line that is exactly heading to the next heading of level 3 or less
// ("# ", "## ", or "### "), or to the end of text, and whether heading was
// found.
func directorNoticeSkillSubsection(text, heading string) (string, bool) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != heading {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if l := lines[j]; strings.HasPrefix(l, "# ") || strings.HasPrefix(l, "## ") || strings.HasPrefix(l, "### ") {
				end = j
				break
			}
		}
		return strings.Join(lines[i:end], "\n"), true
	}
	return "", false
}
