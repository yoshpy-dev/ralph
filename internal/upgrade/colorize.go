package upgrade

import "strings"

// ANSI SGR escape sequences. Kept unexported because the formatting choice is
// an implementation detail of Colorize and should not leak into other packages.
const (
	ansiReset      = "\x1b[0m"
	ansiBoldRed    = "\x1b[1;31m"
	ansiBoldGreen  = "\x1b[1;32m"
	ansiCyan       = "\x1b[36m"
	ansiRed        = "\x1b[31m"
	ansiGreen      = "\x1b[32m"
	ansiDimDefault = "\x1b[2m"
)

// Colorize wraps a UnifiedDiff result with ANSI color escapes for terminal
// display. The function is pure: callers gate it on TTY/`NO_COLOR` themselves.
//
// Recognized line shapes:
//   - "--- ..."   → bold red    (old-file header)
//   - "+++ ..."   → bold green  (new-file header)
//   - "@@ ... @@" → cyan        (hunk header)
//   - "<gutter> │ -..." → red   (removal)
//   - "<gutter> │ +..." → green (addition)
//   - "\ ..."     → dim         (no-newline marker)
//   - context (` `) and anything else → unchanged
//
// Unrecognized lines pass through untouched so the function degrades safely
// if the diff format ever evolves ahead of this colorizer.
func Colorize(diff string) string {
	if diff == "" {
		return ""
	}

	endsWithNewline := strings.HasSuffix(diff, "\n")
	body := diff
	if endsWithNewline {
		body = body[:len(body)-1]
	}

	var b strings.Builder
	b.Grow(len(diff) + 32)

	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if code := ansiForLine(line); code != "" {
			b.WriteString(code)
			b.WriteString(line)
			b.WriteString(ansiReset)
		} else {
			b.WriteString(line)
		}
		if i < len(lines)-1 || endsWithNewline {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func ansiForLine(line string) string {
	switch {
	case strings.HasPrefix(line, "--- "):
		return ansiBoldRed
	case strings.HasPrefix(line, "+++ "):
		return ansiBoldGreen
	case strings.HasPrefix(line, "@@ "):
		return ansiCyan
	case strings.HasPrefix(line, "\\ "):
		return ansiDimDefault
	}
	// Body lines carry the gutter `<old> <new> │ <prefix><content>`. Locate
	// the separator and inspect the byte immediately after it. We must use
	// byte indexing here because `│` is a 3-byte UTF-8 rune.
	if idx := strings.Index(line, diffSeparator); idx >= 0 {
		after := idx + len(diffSeparator)
		if after < len(line) {
			switch line[after] {
			case '-':
				return ansiRed
			case '+':
				return ansiGreen
			}
		}
	}
	return ""
}
