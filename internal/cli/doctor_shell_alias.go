package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// shellAliasEnv is the part of the process environment checkShellAliases
// depends on. It is a struct (rather than reading os.UserHomeDir/os.Getenv
// directly) so callers -- production and test alike -- can substitute
// values through a plain function instead of mutating real environment
// variables.
type shellAliasEnv struct {
	Home    string // user home directory
	Zdotdir string // $ZDOTDIR as seen by this process ("" when unset)
}

// shellAliasEnvFromOS resolves shellAliasEnv from os.UserHomeDir and
// $ZDOTDIR. This is the production resolver; doctorShellAliasEnv below wraps
// it as the package-level default that runDoctorFull actually calls.
func shellAliasEnvFromOS() (shellAliasEnv, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return shellAliasEnv{}, err
	}
	return shellAliasEnv{Home: home, Zdotdir: os.Getenv("ZDOTDIR")}, nil
}

// doctorShellAliasEnv is the environment resolver runDoctorFull hands to
// checkShellAliases. It is a package variable (rather than a direct call to
// shellAliasEnvFromOS) so internal/cli's TestMain (main_test.go) can pin it
// to a directory with no rc files, keeping every runDoctor*-based test
// hermetic against the developer's real shell rc files (self-review L7:
// without this seam, 13 pre-existing runDoctor* call sites would each open
// up to 11 files under the real $HOME).
var doctorShellAliasEnv = shellAliasEnvFromOS

// shellAliasRcCandidates returns the rc files the check scans, in order.
// Only files that exist are read; each is resolved through
// filepath.EvalSymlinks and deduplicated by its real path (on macOS a
// ~/.zshrc symlinked to ~/.config/zsh/.zshrc must count once -- the
// reported file:line still names the symlink candidate, not the resolved
// target, since dedup keeps the first candidate in list order, not the
// resolved one). The list is deliberately static -- files pulled in via
// `source` are not followed; checkShellAliases' Detail names every file it
// actually scanned, so a `source`d alias reads as outside what was
// checked, not as a silent miss. A relative $ZDOTDIR is ignored (zsh itself
// would refuse it). ~/.zshrc and ~/.config/zsh/.zshrc are still scanned even
// when $ZDOTDIR is set and zsh itself would skip them: this process's
// $ZDOTDIR is not guaranteed to match the one herdr's login-zsh pane
// resolves from /etc/zshenv, so the candidate list intentionally
// over-approximates rather than under-approximates.
func shellAliasRcCandidates(env shellAliasEnv) []string {
	var raw []string
	if env.Zdotdir != "" && filepath.IsAbs(env.Zdotdir) {
		raw = append(raw,
			filepath.Join(env.Zdotdir, ".zshrc"),
			filepath.Join(env.Zdotdir, ".zshenv"),
		)
	}
	raw = append(raw,
		filepath.Join(env.Home, ".zshrc"),
		filepath.Join(env.Home, ".config", "zsh", ".zshrc"),
		filepath.Join(env.Home, ".zshenv"),
		filepath.Join(env.Home, ".zprofile"),
		filepath.Join(env.Home, ".zsh_aliases"),
		filepath.Join(env.Home, ".bashrc"),
		filepath.Join(env.Home, ".bash_profile"),
		filepath.Join(env.Home, ".bash_aliases"),
		filepath.Join(env.Home, ".profile"),
		filepath.Join(env.Home, ".config", "fish", "config.fish"),
	)

	seen := map[string]bool{}
	var out []string
	for _, c := range raw {
		info, err := os.Stat(c)
		if err != nil || info.IsDir() {
			continue
		}
		real, evalErr := filepath.EvalSymlinks(c)
		if evalErr != nil {
			real = c
		}
		if seen[real] {
			continue
		}
		seen[real] = true
		out = append(out, c)
	}
	return out
}

// shellAliasWord is one NAME=VALUE (or fish NAME VALUE) word pair found in
// an `alias` statement by parseShellAliasLine.
type shellAliasWord struct {
	Name  string
	Value string
}

// shellAliasWords splits the text following the `alias` keyword into shell
// words. It understands three states: unquoted, single-quoted, and
// double-quoted. Unquoted and double-quoted text treat a backslash as
// escaping the next byte (the backslash itself is dropped); single-quoted
// text has no escapes. Quote characters delimit a segment and are dropped
// from the result; adjacent segments concatenate into one word regardless
// of which state produced them -- a single-quoted a, an escaped quote, and
// a single-quoted b in a row glue into the one word a'b, and `codex=`
// immediately followed by a double-quoted value is one word too. Unquoted
// whitespace ends a word. An unquoted `#` at the very start of a word ends
// the whole statement (the rest of the line is a trailing comment). An
// unquoted semicolon, pipe, or ampersand likewise ends the statement -- the
// remainder of the line is not examined (documented limitation: only one
// `alias` statement per line is read). An unterminated quote runs to the
// end of the line (a value continued on a following line is not examined).
func shellAliasWords(rest string) []string {
	var words []string
	var cur strings.Builder
	inWord := false
	flush := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}

	n := len(rest)
	i := 0
	for i < n {
		c := rest[i]
		switch {
		case c == ' ' || c == '\t':
			flush()
			i++
		case c == '\'':
			inWord = true
			i++
			for i < n && rest[i] != '\'' {
				cur.WriteByte(rest[i])
				i++
			}
			if i < n {
				i++ // skip closing quote
			}
		case c == '"':
			inWord = true
			i++
			for i < n && rest[i] != '"' {
				if rest[i] == '\\' && i+1 < n {
					cur.WriteByte(rest[i+1])
					i += 2
					continue
				}
				cur.WriteByte(rest[i])
				i++
			}
			if i < n {
				i++ // skip closing quote
			}
		case c == '\\':
			inWord = true
			if i+1 < n {
				cur.WriteByte(rest[i+1])
				i += 2
			} else {
				i++ // trailing backslash at end of line: drop it
			}
		case c == '#' && !inWord:
			flush()
			return words
		case c == ';' || c == '|' || c == '&':
			flush()
			return words
		default:
			inWord = true
			cur.WriteByte(c)
			i++
		}
	}
	flush()
	return words
}

// parseShellAliasLine returns the codex/claude alias definitions on one rc
// line. The trimmed line must start with the literal `alias` followed by a
// space or tab -- anything else (including a `#`-leading comment line)
// yields no definitions. Words starting with `-` before the first
// definition are alias options (e.g. zsh's `alias -g`) and are skipped.
// Two forms are recognized: sh/zsh `NAME=VALUE` words, of which several may
// appear in one statement (`alias codex='codex' claude='claude --model x'`
// reports both, each under its own name); and fish `alias NAME VALUE`,
// recognized when the first non-option word is exactly `codex` or `claude`
// with no `=` -- in that case the next word is the value and no further
// words on the line are examined.
func parseShellAliasLine(line string) []shellAliasWord {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "alias") {
		return nil
	}
	rest := trimmed[len("alias"):]
	if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return nil
	}

	words := shellAliasWords(rest)

	i := 0
	for i < len(words) && strings.HasPrefix(words[i], "-") {
		i++
	}
	if i >= len(words) {
		return nil
	}

	first := words[i]
	if !strings.Contains(first, "=") && (first == "codex" || first == "claude") {
		if i+1 < len(words) {
			return []shellAliasWord{{Name: first, Value: words[i+1]}}
		}
		return nil
	}

	var defs []shellAliasWord
	for _, w := range words[i:] {
		eq := strings.Index(w, "=")
		if eq < 0 {
			continue
		}
		name, value := w[:eq], w[eq+1:]
		if name == "codex" || name == "claude" {
			defs = append(defs, shellAliasWord{Name: name, Value: value})
		}
	}
	return defs
}

// shellAliasDef is one codex/claude alias definition found in an rc file,
// together with the seat-launch flags (if any) its value adds.
type shellAliasDef struct {
	Name  string // "codex" or "claude"
	File  string // candidate path as scanned (before ~ substitution)
	Line  int
	Flags []string // conflicting seat-launch flags, in order of appearance, deduplicated; empty = harmless
}

// scanShellAliasFile reads one rc file and returns every codex/claude alias
// definition found, in file order, whether or not it carries a conflicting
// flag. Each line is bounded at 1 MiB (bufio.Scanner's own 64 KiB default
// would otherwise abort the whole file on one long line). When the scanner
// still fails partway through -- the 1 MiB cap included -- the definitions
// collected before the failure are returned alongside the error rather than
// discarded, so a partial read still surfaces the flags it found.
func scanShellAliasFile(path string) ([]shellAliasDef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var defs []shellAliasDef
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		for _, w := range parseShellAliasLine(scanner.Text()) {
			defs = append(defs, shellAliasDef{
				Name:  w.Name,
				File:  path,
				Line:  lineNum,
				Flags: shellAliasConflictingFlags(w.Name, w.Value),
			})
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return defs, scanErr
	}
	return defs, nil
}

// shellAliasConflictingFlags returns every seat-launch flag ralph itself
// passes to the named driver ("codex" or "claude") that also appears in an
// alias value, in order of first appearance, deduplicated. value is the
// already-unquoted word shellAliasWords produced (quoting is handled there
// and nowhere else); it is tokenized on whitespace (strings.Fields).
//
// codex's short flags -m/-s/-a are clap aliases for --model/--sandbox/
// --ask-for-approval and collide identically with them: verified against
// codex-cli 0.154.0, repeating any of -m/-s/-a or its long form exits with
// "cannot be used multiple times". A short flag matches as the token exactly,
// or as the token's first two bytes followed by more text whose third byte
// is not '-' (e.g. -mgpt-5.5, -sread-only) -- so -m/-s/-a alone, or
// concatenated with a value, both match, while an unrelated long flag
// sharing the first two bytes (e.g. --model, --sandbox) does not, because
// those start with "--", not "-<letter>".
//
// claude has no short flags and does not error on a repeated --model or
// --permission-mode: verified against claude 2.1.274, the later value --
// ralph's own, since ralph's flags are appended after the alias expands --
// wins and the seat starts. A claude finding is therefore informational
// only; see checkShellAliases for how the two drivers' severities differ.
func shellAliasConflictingFlags(name, value string) []string {
	isShort := func(tok, letter string) bool {
		return tok == letter || (strings.HasPrefix(tok, letter) && len(tok) > 2 && tok[2] != '-')
	}

	seen := map[string]bool{}
	var flags []string
	add := func(label string) {
		if !seen[label] {
			seen[label] = true
			flags = append(flags, label)
		}
	}

	for _, tok := range strings.Fields(value) {
		switch {
		case tok == "--model", strings.HasPrefix(tok, "--model="):
			add("--model")
		case name == "codex" && isShort(tok, "-m"):
			add("--model (-m)")
		case name == "codex" && (tok == "--sandbox" || strings.HasPrefix(tok, "--sandbox=")):
			add("--sandbox")
		case name == "codex" && isShort(tok, "-s"):
			add("--sandbox (-s)")
		case name == "codex" && (tok == "--ask-for-approval" || strings.HasPrefix(tok, "--ask-for-approval=")):
			add("--ask-for-approval")
		case name == "codex" && isShort(tok, "-a"):
			add("--ask-for-approval (-a)")
		case name == "claude" && (tok == "--permission-mode" || strings.HasPrefix(tok, "--permission-mode=")):
			add("--permission-mode")
		}
	}
	return flags
}

// shellAliasUnreadableReason renders a scan error without repeating the
// file path (the caller already names the file next to this text): a
// *fs.PathError's inner Err (e.g. "permission denied") when errors.As
// matches, else the error's own text (e.g. a bufio.Scanner error).
func shellAliasUnreadableReason(err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error()
	}
	return err.Error()
}

// checkShellAliases is doctor's "Shell aliases (codex/claude)" check.
// resolveEnv supplies the home directory and $ZDOTDIR to scan from
// (production: doctorShellAliasEnv; tests: a closure or TestMain's pin).
// herdrPresent decides codex's severity: codex rejects a repeated flag
// outright, so a codex finding is a real spawn blocker only when herdr
// (the org runtime seat driver) is actually installed to expand the alias
// in a seat's pane -- warn when herdrPresent, info otherwise. claude
// accepts a repeated flag without erroring (see shellAliasConflictingFlags),
// so a claude finding is always informational, regardless of herdrPresent.
func checkShellAliases(resolveEnv func() (shellAliasEnv, error), herdrPresent bool) checkResult {
	r := checkResult{Name: "Shell aliases (codex/claude)"}

	env, err := resolveEnv()
	if err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("could not resolve home directory: %v — shell alias check skipped", err)
		return r
	}

	homePrefix := env.Home + string(filepath.Separator)
	displayPath := func(p string) string {
		if strings.HasPrefix(p, homePrefix) {
			return "~" + string(filepath.Separator) + strings.TrimPrefix(p, homePrefix)
		}
		return p
	}

	candidates := shellAliasRcCandidates(env)

	var defs []shellAliasDef
	var scannedOK []string
	var unreadable []string
	for _, c := range candidates {
		fileDefs, scanErr := scanShellAliasFile(c)
		defs = append(defs, fileDefs...)
		if scanErr != nil {
			unreadable = append(unreadable, fmt.Sprintf("%s (%s)", displayPath(c), shellAliasUnreadableReason(scanErr)))
			continue
		}
		scannedOK = append(scannedOK, displayPath(c))
	}

	var codexItems, claudeItems, harmlessItems []string
	for _, d := range defs {
		item := fmt.Sprintf("alias %s in %s:%d", d.Name, displayPath(d.File), d.Line)
		if len(d.Flags) == 0 {
			harmlessItems = append(harmlessItems, item)
			continue
		}
		full := item + " adds " + strings.Join(d.Flags, ", ")
		if d.Name == "codex" {
			codexItems = append(codexItems, full)
		} else {
			claudeItems = append(claudeItems, full)
		}
	}
	codexFound := len(codexItems) > 0
	claudeFound := len(claudeItems) > 0

	var sentences []string
	if codexFound {
		sentence := strings.Join(codexItems, "; ") +
			` — herdr expands the alias in the seat's pane and codex rejects the repeated flag ("cannot be used multiple times"), ` +
			`so ralph org spawn fails; remove the alias or start herdr from an alias-free rc (docs/recipes/codex-seat-permissions.md)`
		if !herdrPresent {
			sentence += " (herdr not installed, so no seat is affected today)"
		}
		sentences = append(sentences, sentence)
	}
	if claudeFound {
		sentence := strings.Join(claudeItems, "; ") +
			` — claude accepts the repeated flag and the value ralph org spawn passes last wins, so the seat still starts; ` +
			`the alias's other flags reach every seat`
		sentences = append(sentences, sentence)
	}
	switch {
	case codexFound || claudeFound:
		// Findings already speak for themselves; a harmless alias elsewhere
		// is not worth a separate sentence once there is a real finding.
	case len(harmlessItems) > 0:
		verb := "has"
		if len(harmlessItems) > 1 {
			verb = "have"
		}
		sentences = append(sentences, strings.Join(harmlessItems, "; ")+" "+verb+" no conflicting flags")
	case len(scannedOK) > 0:
		sentences = append(sentences, "no codex/claude alias found")
	}

	var scannedClause string
	switch {
	case len(candidates) == 0:
		scannedClause = "no shell rc file found to scan"
	case len(scannedOK) == 0:
		scannedClause = "scanned 0 shell rc file(s) (files they source are not followed)"
	default:
		scannedClause = fmt.Sprintf("scanned %d shell rc file(s): %s (files they source are not followed)",
			len(scannedOK), strings.Join(scannedOK, ", "))
	}

	var detailParts []string
	detailParts = append(detailParts, sentences...)
	if len(unreadable) > 0 {
		detailParts = append(detailParts, "could not read: "+strings.Join(unreadable, ", ")+" — aliases there were not checked")
	}
	detailParts = append(detailParts, scannedClause)
	r.Detail = strings.Join(detailParts, ". ")

	switch {
	case codexFound && herdrPresent:
		r.Status = "warn"
	case codexFound || claudeFound || len(unreadable) > 0:
		r.Status = "info"
	default:
		r.Status = "pass"
	}
	return r
}
