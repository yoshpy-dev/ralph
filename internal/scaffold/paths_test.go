package scaffold

import "testing"

func TestCleanLocalRelPath_Accepts(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"AGENTS.md", "AGENTS.md"},
		{".claude/rules/ralph/testing.md", ".claude/rules/ralph/testing.md"},
		{"a/./b", "a/b"},
	}
	for _, c := range cases {
		got, err := CleanLocalRelPath(c.in)
		if err != nil {
			t.Fatalf("CleanLocalRelPath(%q): unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("CleanLocalRelPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCleanLocalRelPath_Rejects(t *testing.T) {
	cases := []string{
		"",
		".",
		"..",
		"../escape.md",
		"a/../../escape.md",
		"/absolute.md",
	}
	for _, in := range cases {
		if _, err := CleanLocalRelPath(in); err == nil {
			t.Errorf("CleanLocalRelPath(%q) = nil error, want error", in)
		}
	}
}
