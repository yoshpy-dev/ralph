package toolchain

import (
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

// The version token of a tool's output and the version manager behind a
// resolved path (plan 2026-11-12-gate-tool-versions, D1 and D4).

// The version managers ShimOf names.
const (
	ShimMise = "mise"
	ShimAsdf = "asdf"
)

// maxTokenBytes caps a version token; a longer one is not a version.
const maxTokenBytes = 64

// maxReasonBytes caps a Measurement's Reason, and maxPathBytes its Path.
const (
	maxReasonBytes = 200
	maxPathBytes   = 512
)

// versionTokenRE is the shape of a version token: one word a gate record can
// hold as a value (D4).
var versionTokenRE = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// VersionToken returns the version in line, one line of tool's version
// output: the line itself for node ("v24.22.1"), pnpm, yarn, and npm
// ("10.10.0"); what follows "Python " for python3 ("Python 3.12.1" ->
// "3.12.1"); the third word of "go version go1.27.3 darwin/arm64" for go
// ("go1.27.3"). Surrounding spaces and a carriage return are trimmed. ok is
// false when the result is empty, longer than 64 bytes, holds no digit, or
// does not match ^[A-Za-z0-9._+-]+$ (a sentence, an error message, a control
// character).
func VersionToken(tool, line string) (token string, ok bool) {
	s := strings.TrimSpace(line)
	switch tool {
	case ToolPython3:
		if rest, found := strings.CutPrefix(s, "Python "); found {
			s = strings.TrimSpace(rest)
		}
	case ToolGo:
		if f := strings.Fields(s); len(f) >= 3 && f[0] == "go" && f[1] == "version" {
			s = f[2]
		}
	}
	if s == "" || len(s) > maxTokenBytes || !versionTokenRE.MatchString(s) || !strings.ContainsAny(s, "0123456789") {
		return "", false
	}
	return s, true
}

// ShimOf returns ShimMise when p, a path `command -v` printed, is a file in
// a mise shims directory (".../mise/shims/<tool>", which covers
// "~/.local/share/mise/shims/<tool>"), ShimAsdf when it is in asdf's
// (".../.asdf/shims/<tool>"), and "" otherwise, including for a word that is
// not an absolute path (a shell function or alias).
func ShimOf(p string) string {
	if !strings.HasPrefix(p, "/") {
		return ""
	}
	segs := strings.Split(path.Clean(p), "/")
	for i := 0; i+2 < len(segs); i++ {
		if segs[i+1] != "shims" {
			continue
		}
		switch segs[i] {
		case "mise":
			return ShimMise
		case ".asdf":
			return ShimAsdf
		}
	}
	return ""
}

// cleanText returns s with each control character and each byte of invalid
// UTF-8 replaced by "?", cut to at most limit bytes (on a character
// boundary, ending in "..." when cut), so it can go into a log line as one
// line.
func cleanText(s string, limit int) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if (r == utf8.RuneError && size <= 1) || r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			r = '?'
		}
		if b.Len()+utf8.RuneLen(r) > limit {
			out := b.String()
			for len(out)+3 > limit && out != "" {
				_, last := utf8.DecodeLastRuneInString(out)
				out = out[:len(out)-last]
			}
			return out + "..."
		}
		b.WriteRune(r)
	}
	return b.String()
}

// firstLine returns the first line of s that is not blank, trimmed of
// surrounding spaces, and "" when there is none.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}
