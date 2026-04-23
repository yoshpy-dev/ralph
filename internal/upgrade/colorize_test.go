package upgrade

import (
	"strings"
	"testing"
)

func TestColorize_Empty(t *testing.T) {
	if got := Colorize(""); got != "" {
		t.Errorf("Colorize(\"\") = %q, want empty string", got)
	}
}

func TestColorize_FileHeaders(t *testing.T) {
	in := "--- local\n+++ template\n"
	got := Colorize(in)
	if !strings.Contains(got, ansiBoldRed+"--- local"+ansiReset+"\n") {
		t.Errorf("--- header not wrapped in bold-red; got:\n%q", got)
	}
	if !strings.Contains(got, ansiBoldGreen+"+++ template"+ansiReset+"\n") {
		t.Errorf("+++ header not wrapped in bold-green; got:\n%q", got)
	}
}

func TestColorize_HunkHeader(t *testing.T) {
	in := "@@ 旧 L1  →  新 L1 @@\n"
	got := Colorize(in)
	if !strings.Contains(got, ansiCyan+"@@ 旧 L1  →  新 L1 @@"+ansiReset+"\n") {
		t.Errorf("hunk header not wrapped in cyan; got:\n%q", got)
	}
}

func TestColorize_RemovedLine(t *testing.T) {
	in := " 2    │ -beta\n"
	got := Colorize(in)
	if !strings.Contains(got, ansiRed+" 2    │ -beta"+ansiReset+"\n") {
		t.Errorf("removal line not wrapped in red; got:\n%q", got)
	}
}

func TestColorize_AddedLine(t *testing.T) {
	in := "    2 │ +BETA\n"
	got := Colorize(in)
	if !strings.Contains(got, ansiGreen+"    2 │ +BETA"+ansiReset+"\n") {
		t.Errorf("addition line not wrapped in green; got:\n%q", got)
	}
}

func TestColorize_ContextLineUnchanged(t *testing.T) {
	in := " 1  1 │  alpha\n"
	got := Colorize(in)
	// Context lines must pass through without any escape sequence.
	if got != in {
		t.Errorf("context line was modified; got:\n%q", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("context line contains ANSI escape; got:\n%q", got)
	}
}

func TestColorize_NoNewlineMarker(t *testing.T) {
	in := "\\ No newline at end of file\n"
	got := Colorize(in)
	if !strings.Contains(got, ansiDimDefault+"\\ No newline at end of file"+ansiReset+"\n") {
		t.Errorf("no-newline marker not dimmed; got:\n%q", got)
	}
}

// Mixed input must preserve newline structure exactly: every original `\n` is
// retained, including the final one.
func TestColorize_PreservesNewlineStructure(t *testing.T) {
	in := "--- a\n+++ b\n@@ 旧 L1  →  新 L1 @@\n 1    │ -x\n    1 │ +y\n"
	got := Colorize(in)
	if strings.Count(got, "\n") != strings.Count(in, "\n") {
		t.Errorf("newline count drifted: in=%d got=%d\n%q", strings.Count(in, "\n"), strings.Count(got, "\n"), got)
	}
}

// Input without a trailing newline must come back without one either.
func TestColorize_NoTrailingNewlinePreserved(t *testing.T) {
	in := "--- a"
	got := Colorize(in)
	if strings.HasSuffix(got, "\n") {
		t.Errorf("trailing newline appeared where input had none; got:\n%q", got)
	}
}

// Unrecognized prefixes degrade to passthrough so future format additions do
// not silently corrupt output.
func TestColorize_UnknownLinePassthrough(t *testing.T) {
	in := "??? mystery\n"
	got := Colorize(in)
	if got != in {
		t.Errorf("unknown line was modified; got:\n%q", got)
	}
}
