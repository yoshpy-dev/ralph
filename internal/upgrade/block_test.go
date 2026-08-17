package upgrade

import (
	"bytes"
	"testing"
)

func TestUpdateManagedBlock_NormalUpdate(t *testing.T) {
	current := []byte("before\n" +
		BeginMarker("agents-md") + "\n" +
		"old line 1\n" +
		"old line 2\n" +
		EndMarker + "\n" +
		"after\n")
	want := []byte("before\n" +
		BeginMarker("agents-md") + "\n" +
		"new line 1\n" +
		EndMarker + "\n" +
		"after\n")

	got := UpdateManagedBlock(current, "agents-md", []byte("new line 1\n"))
	if got.Outcome != BlockUpdated {
		t.Fatalf("Outcome = %v, want BlockUpdated (reason=%q)", got.Outcome, got.Reason)
	}
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content = %q, want %q", got.Content, want)
	}
}

func TestUpdateManagedBlock_Unchanged(t *testing.T) {
	current := []byte("before\n" +
		BeginMarker("agents-md") + "\n" +
		"same line\n" +
		EndMarker + "\n" +
		"after\n")

	got := UpdateManagedBlock(current, "agents-md", []byte("same line\n"))
	if got.Outcome != BlockUnchanged {
		t.Fatalf("Outcome = %v, want BlockUnchanged (reason=%q)", got.Outcome, got.Reason)
	}
	if !bytes.Equal(got.Content, current) {
		t.Fatalf("Content = %q, want unchanged %q", got.Content, current)
	}
}

func TestUpdateManagedBlock_UpdateIsIdempotent(t *testing.T) {
	current := []byte("before\n" +
		BeginMarker("agents-md") + "\n" +
		"old\n" +
		EndMarker + "\n")
	managed := []byte("new\n")

	first := UpdateManagedBlock(current, "agents-md", managed)
	if first.Outcome != BlockUpdated {
		t.Fatalf("first Outcome = %v, want BlockUpdated (reason=%q)", first.Outcome, first.Reason)
	}
	second := UpdateManagedBlock(first.Content, "agents-md", managed)
	if second.Outcome != BlockUnchanged {
		t.Fatalf("second Outcome = %v, want BlockUnchanged (reason=%q)", second.Outcome, second.Reason)
	}
	if !bytes.Equal(second.Content, first.Content) {
		t.Fatalf("second Content = %q, want %q", second.Content, first.Content)
	}
}

func TestUpdateManagedBlock_AppendWhenAbsent_EmptyFile(t *testing.T) {
	got := UpdateManagedBlock(nil, "gitignore", []byte(".ralph/local/\n"))
	if got.Outcome != BlockAppended {
		t.Fatalf("Outcome = %v, want BlockAppended (reason=%q)", got.Outcome, got.Reason)
	}
	want := []byte(BeginMarker("gitignore") + "\n" +
		".ralph/local/\n" +
		EndMarker + "\n")
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content = %q, want %q", got.Content, want)
	}
}

func TestUpdateManagedBlock_AppendWhenAbsent_NonEmptyWithTrailingNewline(t *testing.T) {
	current := []byte("node_modules/\n")
	got := UpdateManagedBlock(current, "gitignore", []byte(".ralph/local/\n"))
	if got.Outcome != BlockAppended {
		t.Fatalf("Outcome = %v, want BlockAppended (reason=%q)", got.Outcome, got.Reason)
	}
	want := []byte("node_modules/\n" +
		"\n" +
		BeginMarker("gitignore") + "\n" +
		".ralph/local/\n" +
		EndMarker + "\n")
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content = %q, want %q", got.Content, want)
	}
}

func TestUpdateManagedBlock_AppendWhenAbsent_NonEmptyWithoutTrailingNewline(t *testing.T) {
	current := []byte("node_modules/")
	got := UpdateManagedBlock(current, "gitignore", []byte(".ralph/local/\n"))
	if got.Outcome != BlockAppended {
		t.Fatalf("Outcome = %v, want BlockAppended (reason=%q)", got.Outcome, got.Reason)
	}
	want := []byte("node_modules/\n" +
		"\n" +
		BeginMarker("gitignore") + "\n" +
		".ralph/local/\n" +
		EndMarker + "\n")
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content = %q, want %q", got.Content, want)
	}
}

func TestUpdateManagedBlock_AppendWhenAbsent_AlreadyEndsWithBlankLine(t *testing.T) {
	current := []byte("node_modules/\n\n")
	got := UpdateManagedBlock(current, "gitignore", []byte(".ralph/local/\n"))
	if got.Outcome != BlockAppended {
		t.Fatalf("Outcome = %v, want BlockAppended (reason=%q)", got.Outcome, got.Reason)
	}
	want := []byte("node_modules/\n" +
		"\n" +
		BeginMarker("gitignore") + "\n" +
		".ralph/local/\n" +
		EndMarker + "\n")
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content = %q, want %q", got.Content, want)
	}
}

func TestUpdateManagedBlock_Malformed(t *testing.T) {
	tests := []struct {
		name    string
		current []byte
		surface string
	}{
		{
			name:    "begin only",
			current: []byte("x\n" + BeginMarker("agents-md") + "\n" + "content\n"),
			surface: "agents-md",
		},
		{
			name:    "end only",
			current: []byte("x\n" + EndMarker + "\n" + "y\n"),
			surface: "agents-md",
		},
		{
			name: "duplicate begin",
			current: []byte(BeginMarker("agents-md") + "\n" +
				"a\n" +
				BeginMarker("agents-md") + "\n" +
				"b\n" +
				EndMarker + "\n"),
			surface: "agents-md",
		},
		{
			name: "duplicate end",
			current: []byte(BeginMarker("agents-md") + "\n" +
				"a\n" +
				EndMarker + "\n" +
				"b\n" +
				EndMarker + "\n"),
			surface: "agents-md",
		},
		{
			name: "end before begin",
			current: []byte(EndMarker + "\n" +
				"a\n" +
				BeginMarker("agents-md") + "\n"),
			surface: "agents-md",
		},
		{
			name: "surface mismatch",
			current: []byte(BeginMarker("gitignore") + "\n" +
				"a\n" +
				EndMarker + "\n"),
			surface: "agents-md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateManagedBlock(tt.current, tt.surface, []byte("new\n"))
			if got.Outcome != BlockMalformed {
				t.Fatalf("Outcome = %v, want BlockMalformed", got.Outcome)
			}
			if got.Content != nil {
				t.Fatalf("Content = %q, want nil for malformed result", got.Content)
			}
			if got.Reason == "" {
				t.Fatalf("Reason is empty, want a non-empty explanation")
			}
		})
	}
}

func TestUpdateManagedBlock_CRLFFile(t *testing.T) {
	current := []byte("before\r\n" +
		BeginMarker("agents-md") + "\r\n" +
		"old\r\n" +
		EndMarker + "\r\n" +
		"after\r\n")
	want := []byte("before\r\n" +
		BeginMarker("agents-md") + "\r\n" +
		"new\r\n" +
		EndMarker + "\r\n" +
		"after\r\n")

	got := UpdateManagedBlock(current, "agents-md", []byte("new\n"))
	if got.Outcome != BlockUpdated {
		t.Fatalf("Outcome = %v, want BlockUpdated (reason=%q)", got.Outcome, got.Reason)
	}
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content = %q, want %q", got.Content, want)
	}
}

// TestUpdateManagedBlock_PreservesBytesOutsideBlock asserts the full
// resulting byte sequence, including surrounding user content, to guard
// against accidental mutation of anything outside the marker pair.
func TestUpdateManagedBlock_PreservesBytesOutsideBlock(t *testing.T) {
	current := []byte("# My Notes\n" +
		"\n" +
		"Some user prose that must survive untouched.\n" +
		"\n" +
		BeginMarker("agents-md") + "\n" +
		"stale ralph content\n" +
		"more stale content\n" +
		EndMarker + "\n" +
		"\n" +
		"Trailing user prose, also untouched.\n")

	managed := []byte("fresh ralph content\n")

	got := UpdateManagedBlock(current, "agents-md", managed)
	if got.Outcome != BlockUpdated {
		t.Fatalf("Outcome = %v, want BlockUpdated (reason=%q)", got.Outcome, got.Reason)
	}

	want := []byte("# My Notes\n" +
		"\n" +
		"Some user prose that must survive untouched.\n" +
		"\n" +
		BeginMarker("agents-md") + "\n" +
		"fresh ralph content\n" +
		EndMarker + "\n" +
		"\n" +
		"Trailing user prose, also untouched.\n")

	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%q\nwant\n%q", got.Content, want)
	}
}

func TestBeginMarker(t *testing.T) {
	got := BeginMarker("agents-md")
	want := "<!-- BEGIN RALPH MANAGED (ralph:agents-md) -->"
	if got != want {
		t.Fatalf("BeginMarker(%q) = %q, want %q", "agents-md", got, want)
	}
}

func TestBeginMarkerStyled_Hash(t *testing.T) {
	got := BeginMarkerStyled("gitignore", BlockMarkerHash)
	want := "# BEGIN RALPH MANAGED (ralph:gitignore)"
	if got != want {
		t.Fatalf("BeginMarkerStyled(gitignore, hash) = %q, want %q", got, want)
	}
}

func TestEndMarkerStyled_Hash(t *testing.T) {
	got := EndMarkerStyled(BlockMarkerHash)
	want := "# END RALPH MANAGED"
	if got != want {
		t.Fatalf("EndMarkerStyled(hash) = %q, want %q", got, want)
	}
}

func TestUpdateManagedBlockStyled_Hash_Update(t *testing.T) {
	current := []byte("node_modules/\n" +
		BeginMarkerStyled("gitignore", BlockMarkerHash) + "\n" +
		"old-entry/\n" +
		EndMarkerStyled(BlockMarkerHash) + "\n" +
		"*.local\n")
	want := []byte("node_modules/\n" +
		BeginMarkerStyled("gitignore", BlockMarkerHash) + "\n" +
		"new-entry/\n" +
		EndMarkerStyled(BlockMarkerHash) + "\n" +
		"*.local\n")

	got := UpdateManagedBlockStyled(current, "gitignore", []byte("new-entry/\n"), BlockMarkerHash)
	if got.Outcome != BlockUpdated {
		t.Fatalf("Outcome = %v, want BlockUpdated (reason=%q)", got.Outcome, got.Reason)
	}
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content = %q, want %q", got.Content, want)
	}
}

func TestUpdateManagedBlockStyled_Hash_Append(t *testing.T) {
	current := []byte("*.local\n")
	got := UpdateManagedBlockStyled(current, "gitignore", []byte(".ralph/local/\n"), BlockMarkerHash)
	if got.Outcome != BlockAppended {
		t.Fatalf("Outcome = %v, want BlockAppended (reason=%q)", got.Outcome, got.Reason)
	}
	want := []byte("*.local\n" +
		"\n" +
		BeginMarkerStyled("gitignore", BlockMarkerHash) + "\n" +
		".ralph/local/\n" +
		EndMarkerStyled(BlockMarkerHash) + "\n")
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content = %q, want %q", got.Content, want)
	}
}

func TestUpdateManagedBlockStyled_Hash_Malformed(t *testing.T) {
	tests := []struct {
		name    string
		current []byte
	}{
		{
			name:    "begin only",
			current: []byte("x\n" + BeginMarkerStyled("gitignore", BlockMarkerHash) + "\n" + "content\n"),
		},
		{
			name:    "end only",
			current: []byte("x\n" + EndMarkerStyled(BlockMarkerHash) + "\n" + "y\n"),
		},
		{
			name: "surface mismatch",
			current: []byte(BeginMarkerStyled("agents-md", BlockMarkerHash) + "\n" +
				"a\n" +
				EndMarkerStyled(BlockMarkerHash) + "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateManagedBlockStyled(tt.current, "gitignore", []byte("new\n"), BlockMarkerHash)
			if got.Outcome != BlockMalformed {
				t.Fatalf("Outcome = %v, want BlockMalformed", got.Outcome)
			}
			if got.Content != nil {
				t.Fatalf("Content = %q, want nil for malformed result", got.Content)
			}
			if got.Reason == "" {
				t.Fatalf("Reason is empty, want a non-empty explanation")
			}
		})
	}
}

// TestUpdateManagedBlockStyled_Hash_DoesNotMatchHTMLMarkers ensures the two
// marker styles are fully independent: an HTML-style block in the file must
// not be mistaken for a hash-style block (and vice versa), so a hash-style
// call against HTML-marked content treats it as if no block exists at all.
func TestUpdateManagedBlockStyled_Hash_DoesNotMatchHTMLMarkers(t *testing.T) {
	current := []byte(BeginMarker("gitignore") + "\n" +
		"old\n" +
		EndMarker + "\n")

	got := UpdateManagedBlockStyled(current, "gitignore", []byte("new\n"), BlockMarkerHash)
	if got.Outcome != BlockAppended {
		t.Fatalf("Outcome = %v, want BlockAppended (reason=%q)", got.Outcome, got.Reason)
	}
}
