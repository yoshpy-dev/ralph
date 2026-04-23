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
	assertContains(t, got, "│ +d\n")
	if strings.Contains(got, "│ -d") {
		t.Errorf("unexpected deletion line in add-only diff:\n%s", got)
	}
}

func TestUnifiedDiff_RemoveOnly(t *testing.T) {
	old := []byte("a\nb\nc\nd\n")
	new := []byte("a\nb\nc\n")
	got := UnifiedDiff(old, new, "old", "new")
	assertContains(t, got, "│ -d\n")
	if strings.Contains(got, "│ +d") {
		t.Errorf("unexpected addition line in remove-only diff:\n%s", got)
	}
}

func TestUnifiedDiff_Replace(t *testing.T) {
	old := []byte("alpha\nbeta\ngamma\n")
	new := []byte("alpha\nBETA\ngamma\n")
	got := UnifiedDiff(old, new, "old", "new")
	assertContains(t, got, "│ -beta\n")
	assertContains(t, got, "│ +BETA\n")
	assertContains(t, got, "│  alpha\n")
	assertContains(t, got, "│  gamma\n")
}

func TestUnifiedDiff_EmptyToNonEmpty(t *testing.T) {
	got := UnifiedDiff(nil, []byte("hello\n"), "old", "new")
	assertContains(t, got, "--- old\n")
	assertContains(t, got, "│ +hello\n")
	// Hunk header must mark the empty old side as `(空)`.
	assertContains(t, got, "@@ 旧 (空)  →  新 L1 @@")
}

func TestUnifiedDiff_NonEmptyToEmpty(t *testing.T) {
	got := UnifiedDiff([]byte("gone\n"), nil, "old", "new")
	assertContains(t, got, "│ -gone\n")
	assertContains(t, got, "@@ 旧 L1  →  新 (空) @@")
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
	assertContains(t, got, "│  2\n")
	assertContains(t, got, "│  3\n")
	assertContains(t, got, "│  4\n")
	assertContains(t, got, "│ -5\n")
	assertContains(t, got, "│ +FIVE\n")
	assertContains(t, got, "│  6\n")
	assertContains(t, got, "│  7\n")
	assertContains(t, got, "│  8\n")
	// Lines far outside the context window should not appear.
	if strings.Contains(got, "│  1\n") {
		t.Errorf("expected line 1 to be outside context window:\n%s", got)
	}
	if strings.Contains(got, "│  9\n") {
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

// Line-number gutter must show the old-side number on context+delete rows and
// the new-side number on context+add rows. Blank otherwise.
func TestUnifiedDiff_LineNumbersGutter(t *testing.T) {
	old := []byte("a\nb\nc\n")
	new := []byte("a\nX\nc\n")
	got := UnifiedDiff(old, new, "old", "new")
	// `a` is context at old line 1, new line 1.
	assertContains(t, got, " 1  1 │  a\n")
	// `b` removed at old line 2 — new gutter blank.
	assertContains(t, got, " 2    │ -b\n")
	// `X` added at new line 2 — old gutter blank.
	assertContains(t, got, "    2 │ +X\n")
	// `c` context at old line 3, new line 3.
	assertContains(t, got, " 3  3 │  c\n")
}

// Range header should collapse single-line hunks to "Lk" instead of "Lk–k".
func TestUnifiedDiff_HunkHeader_SingleLineRange(t *testing.T) {
	old := []byte("only\n")
	new := []byte("ONLY\n")
	got := UnifiedDiff(old, new, "old", "new")
	assertContains(t, got, "@@ 旧 L1  →  新 L1 @@")
}

// Files with more than 9 lines must size the gutter to 2 columns minimum.
// 100+ line files should grow the gutter to 3 columns to keep alignment.
func TestUnifiedDiff_GutterWidth_Scales(t *testing.T) {
	// Build a 120-line old file with one change near line 100.
	var oldB, newB strings.Builder
	for i := 1; i <= 120; i++ {
		if i == 100 {
			oldB.WriteString("X\n")
			newB.WriteString("Y\n")
			continue
		}
		oldB.WriteString("line\n")
		newB.WriteString("line\n")
	}
	got := UnifiedDiff([]byte(oldB.String()), []byte(newB.String()), "old", "new")
	// Width 3 → "100" right-aligned in a 3-char column.
	assertContains(t, got, "100     │ -X\n")
	assertContains(t, got, "    100 │ +Y\n")
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("expected diff to contain %q, got:\n%s", want, got)
	}
}
