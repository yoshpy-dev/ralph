package upgrade

import (
	"strings"
	"testing"
)

func TestUnifiedDiff_IdenticalInput(t *testing.T) {
	got := UnifiedDiff([]byte("a\nb\nc\n"), []byte("a\nb\nc\n"), "old", "new")
	if got != "" {
		t.Errorf("expected empty diff, got:\n%s", got)
	}
}

func TestUnifiedDiff_IdenticalEmpty(t *testing.T) {
	got := UnifiedDiff(nil, nil, "old", "new")
	if got != "" {
		t.Errorf("expected empty diff for empty inputs, got:\n%s", got)
	}
}

func TestUnifiedDiff_AddOnly(t *testing.T) {
	old := []byte("a\nb\nc\n")
	new := []byte("a\nb\nc\nd\n")
	got := UnifiedDiff(old, new, "old", "new")
	assertContains(t, got, "--- old\n")
	assertContains(t, got, "+++ new\n")
	assertContains(t, got, "+d\n")
	if strings.Contains(got, "-d") {
		t.Errorf("unexpected deletion line in add-only diff:\n%s", got)
	}
}

func TestUnifiedDiff_RemoveOnly(t *testing.T) {
	old := []byte("a\nb\nc\nd\n")
	new := []byte("a\nb\nc\n")
	got := UnifiedDiff(old, new, "old", "new")
	assertContains(t, got, "-d\n")
	if strings.Contains(got, "+d") {
		t.Errorf("unexpected addition line in remove-only diff:\n%s", got)
	}
}

func TestUnifiedDiff_Replace(t *testing.T) {
	old := []byte("alpha\nbeta\ngamma\n")
	new := []byte("alpha\nBETA\ngamma\n")
	got := UnifiedDiff(old, new, "old", "new")
	assertContains(t, got, "-beta\n")
	assertContains(t, got, "+BETA\n")
	assertContains(t, got, " alpha\n")
	assertContains(t, got, " gamma\n")
}

func TestUnifiedDiff_EmptyToNonEmpty(t *testing.T) {
	got := UnifiedDiff(nil, []byte("hello\n"), "old", "new")
	assertContains(t, got, "--- old\n")
	assertContains(t, got, "+hello\n")
	// Hunk header must use 0 as old start when the old side is empty.
	assertContains(t, got, "@@ -0,0 +1,1 @@")
}

func TestUnifiedDiff_NonEmptyToEmpty(t *testing.T) {
	got := UnifiedDiff([]byte("gone\n"), nil, "old", "new")
	assertContains(t, got, "-gone\n")
	assertContains(t, got, "@@ -1,1 +0,0 @@")
}

func TestUnifiedDiff_TrailingNewlineDifference(t *testing.T) {
	old := []byte("line\n")
	new := []byte("line") // no trailing newline
	got := UnifiedDiff(old, new, "old", "new")
	assertContains(t, got, "\\ No newline at end of file")
}

func TestUnifiedDiff_ContextWindow(t *testing.T) {
	old := []byte("1\n2\n3\n4\n5\n6\n7\n8\n9\n")
	new := []byte("1\n2\n3\n4\nFIVE\n6\n7\n8\n9\n")
	got := UnifiedDiff(old, new, "old", "new")
	// Expect 3 lines of context around the change: 2,3,4 before, 6,7,8 after.
	assertContains(t, got, " 2\n")
	assertContains(t, got, " 3\n")
	assertContains(t, got, " 4\n")
	assertContains(t, got, "-5\n")
	assertContains(t, got, "+FIVE\n")
	assertContains(t, got, " 6\n")
	assertContains(t, got, " 7\n")
	assertContains(t, got, " 8\n")
	// Lines far outside the context window should not appear.
	if strings.Contains(got, " 1\n") {
		t.Errorf("expected line 1 to be outside context window:\n%s", got)
	}
	if strings.Contains(got, " 9\n") {
		t.Errorf("expected line 9 to be outside context window:\n%s", got)
	}
}

func TestUnifiedDiff_OrderStability(t *testing.T) {
	// Running the same input twice should produce identical output so diff
	// view and evidence artifacts are reproducible.
	old := []byte("a\nb\nc\n")
	new := []byte("a\nX\nc\n")
	got1 := UnifiedDiff(old, new, "old", "new")
	got2 := UnifiedDiff(old, new, "old", "new")
	if got1 != got2 {
		t.Errorf("non-deterministic output:\n%s\n---\n%s", got1, got2)
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("expected diff to contain %q, got:\n%s", want, got)
	}
}
