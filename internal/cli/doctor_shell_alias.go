package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
// without this seam, the pre-existing runDoctor* call sites would each read
// every candidate rc file under the real $HOME).
var doctorShellAliasEnv = shellAliasEnvFromOS

// shellAliasInaccessibleDir is a candidate directory whose os.Stat failed
// for a reason other than "the path cannot exist" (e.g. a permission error
// on the directory itself) -- the file(s) under it can't be ruled absent,
// so checkShellAliases must not silently read this as "no alias here".
type shellAliasInaccessibleDir struct {
	Dir    string // the candidate's parent directory, as an absolute path
	Reason string // shellAliasUnreadableReason(err) for the stat failure
}

// shellAliasRcCandidates returns the rc files the check scans, in order,
// plus any candidate directory whose stat failed inconclusively (see
// shellAliasInaccessibleDir). Only files that exist are read; each is
// resolved through filepath.EvalSymlinks and deduplicated by its real path
// (on macOS a ~/.zshrc symlinked to ~/.config/zsh/.zshrc must count once --
// the reported file:line still names the symlink candidate, not the
// resolved target, since dedup keeps the first candidate in list order, not
// the resolved one). The list is deliberately static -- files pulled in via
// `source` are not followed; checkShellAliases' Detail names every file it
// actually scanned, so a `source`d alias reads as outside what was
// checked, not as a silent miss.
//
// herdr's pane runs a login zsh, which reads .zshenv, .zprofile, .zshrc,
// and .zlogin -- in that load order -- from $ZDOTDIR, falling back to
// $HOME when $ZDOTDIR is unset. The candidate list scans those four files
// under each of $ZDOTDIR (only when set and absolute -- a relative
// $ZDOTDIR is ignored, since zsh itself would refuse it), $HOME, and
// $HOME/.config/zsh, in that order: $HOME and $HOME/.config/zsh are
// scanned even when $ZDOTDIR is set and zsh itself would skip them,
// because this process's $ZDOTDIR is not guaranteed to match the one
// herdr's login-zsh pane resolves from /etc/zshenv, so the list
// intentionally over-approximates rather than under-approximates. After
// the zsh files come ~/.zsh_aliases, the bash files (.bashrc,
// .bash_profile, .bash_login, .bash_aliases, .profile), and finally
// ~/.config/fish/config.fish.
func shellAliasRcCandidates(env shellAliasEnv) ([]string, []shellAliasInaccessibleDir) {
	var raw []string

	zshDirs := []string{}
	if env.Zdotdir != "" && filepath.IsAbs(env.Zdotdir) {
		zshDirs = append(zshDirs, env.Zdotdir)
	}
	zshDirs = append(zshDirs, env.Home, filepath.Join(env.Home, ".config", "zsh"))
	for _, dir := range zshDirs {
		for _, f := range []string{".zshenv", ".zprofile", ".zshrc", ".zlogin"} {
			raw = append(raw, filepath.Join(dir, f))
		}
	}

	raw = append(raw, filepath.Join(env.Home, ".zsh_aliases"))
	for _, f := range []string{".bashrc", ".bash_profile", ".bash_login", ".bash_aliases", ".profile"} {
		raw = append(raw, filepath.Join(env.Home, f))
	}
	raw = append(raw, filepath.Join(env.Home, ".config", "fish", "config.fish"))

	seen := map[string]bool{}
	seenDir := map[string]bool{}
	var out []string
	var inaccessible []shellAliasInaccessibleDir
	for _, c := range raw {
		info, err := os.Stat(c)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
				continue // the path cannot exist -- nothing was hidden from us
			}
			dir := filepath.Dir(c)
			if !seenDir[dir] {
				seenDir[dir] = true
				inaccessible = append(inaccessible, shellAliasInaccessibleDir{Dir: dir, Reason: shellAliasUnreadableReason(err)})
			}
			continue
		}
		if info.IsDir() {
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
	return out, inaccessible
}

// shellAliasStatementMarkers are leading words that can precede `alias` in
// a statement without preventing it from being recognized: shell control-
// flow keywords/operators that commonly appear right before a conditional
// alias definition, e.g. `if ...; then alias codex=...; fi` or
// `command -v codex >/dev/null && alias codex=...`.
var shellAliasStatementMarkers = map[string]bool{
	"then":    true,
	"else":    true,
	"do":      true,
	"{":       true,
	"builtin": true,
}

// shellAliasStatements splits one logical line (possibly several physical
// lines already joined by scanShellAliasFile) into shell statements, each a
// slice of words. Word splitting understands three states: unquoted,
// single-quoted, and double-quoted. Unquoted and double-quoted text treat a
// backslash as escaping the next byte (the backslash itself is dropped);
// single-quoted text has no escapes. Quote characters delimit a segment and
// are dropped from the result; adjacent segments concatenate into one word
// regardless of which state produced them. Unquoted whitespace ends a word.
// An unquoted `;`, `|`, or `&` ends the current statement -- a run like `&&`
// or `||` produces no empty statement in between (empty statements are
// dropped). Quotes protect all of `;`, `|`, `&`, and `#` from ending
// anything. An unquoted `#` at the very start of a word ends scanning
// entirely: the rest of the line, including any later statement, is a
// trailing comment. In unquoted or double-quoted state, a backslash
// immediately followed by a newline is a shell line-continuation: both
// bytes are dropped, joining the surrounding text seamlessly -- this is
// what lets scanShellAliasFile join two physical lines with a literal "\n"
// and have the result read as one statement. open reports whether the line
// ended before its last statement closed: either inside an unterminated
// quote, or on a bare trailing backslash with nothing after it (no
// newline) -- both are scanShellAliasFile's signal to read and append the
// next physical line.
func shellAliasStatements(line string) (stmts [][]string, open bool) {
	var words []string
	var cur strings.Builder
	inWord := false
	openQuote := false
	trailingBackslash := false

	flushWord := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	flushStatement := func() {
		flushWord()
		if len(words) > 0 {
			stmts = append(stmts, words)
			words = nil
		}
	}

	n := len(line)
	i := 0
	for i < n {
		c := line[i]
		switch {
		case c == ' ' || c == '\t':
			flushWord()
			i++
		case c == '\'':
			inWord = true
			i++
			start := i
			for i < n && line[i] != '\'' {
				i++
			}
			cur.WriteString(line[start:i])
			if i < n {
				i++ // skip closing quote
				openQuote = false
			} else {
				openQuote = true
			}
		case c == '"':
			inWord = true
			i++
			closed := false
			for i < n {
				if line[i] == '"' {
					closed = true
					i++
					break
				}
				if line[i] == '\\' && i+1 < n {
					if line[i+1] == '\n' {
						i += 2 // line continuation: drop both bytes
						continue
					}
					cur.WriteByte(line[i+1])
					i += 2
					continue
				}
				cur.WriteByte(line[i])
				i++
			}
			openQuote = !closed
		case c == '\\':
			switch {
			case i+1 < n && line[i+1] == '\n':
				i += 2 // line continuation: drop both bytes
			case i+1 < n:
				inWord = true
				cur.WriteByte(line[i+1])
				i += 2
			default:
				trailingBackslash = true
				i++
			}
		case c == '#' && !inWord:
			flushStatement()
			return stmts, openQuote || trailingBackslash
		case c == ';' || c == '|' || c == '&':
			flushStatement()
			i++
		default:
			inWord = true
			cur.WriteByte(c)
			i++
		}
	}
	flushStatement()
	return stmts, openQuote || trailingBackslash
}

// shellAliasAssignment is one NAME=VALUE (or fish NAME VALUE) pair found in
// an `alias` statement by parseAliasWords.
type shellAliasAssignment struct {
	Name  string
	Value string
}

// parseAliasStatements returns every codex/claude alias definition found on
// line, across every statement on the line (not just the first), plus
// openAlias: whether the line's last statement is itself an alias statement
// that ended unterminated (see shellAliasStatements' open return). A
// trailing unterminated statement that is NOT alias-shaped (e.g. an
// unrelated `msg=don't` with a stray apostrophe) never sets openAlias, so
// scanShellAliasFile only ever joins a following physical line onto a real
// alias statement -- an open quote on an unrelated line cannot swallow a
// later, independent alias line.
func parseAliasStatements(line string) (defs []shellAliasAssignment, openAlias bool) {
	stmts, open := shellAliasStatements(line)
	for _, words := range stmts {
		i := 0
		for i < len(words) && shellAliasStatementMarkers[words[i]] {
			i++
		}
		if i >= len(words) || words[i] != "alias" {
			continue
		}
		defs = append(defs, parseAliasWords(words[i+1:])...)
	}
	if open && len(stmts) > 0 {
		last := stmts[len(stmts)-1]
		i := 0
		for i < len(last) && shellAliasStatementMarkers[last[i]] {
			i++
		}
		openAlias = i < len(last) && last[i] == "alias"
	}
	return defs, openAlias
}

// parseAliasWords extracts codex/claude definitions from the words that
// follow the `alias` keyword within one statement. Words starting with `-`
// before the first definition are alias options (e.g. zsh's `alias -g`) and
// are skipped. Two forms are recognized: sh/zsh `NAME=VALUE` words, of
// which several may appear in one statement (`alias codex='codex'
// claude='claude --model x'` reports both, each under its own name); and
// fish `alias NAME VALUE`, recognized when the first non-option word is
// exactly `codex` or `claude` with no `=` -- in that case the next word is
// the value and no further words in the statement are examined.
func parseAliasWords(words []string) []shellAliasAssignment {
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
			return []shellAliasAssignment{{Name: first, Value: words[i+1]}}
		}
		return nil
	}

	var defs []shellAliasAssignment
	for _, w := range words[i:] {
		eq := strings.Index(w, "=")
		if eq < 0 {
			continue
		}
		name, value := w[:eq], w[eq+1:]
		if name == "codex" || name == "claude" {
			defs = append(defs, shellAliasAssignment{Name: name, Value: value})
		}
	}
	return defs
}

// shellAliasDef is one codex/claude alias definition found in an rc file,
// together with the seat-launch flags (if any) its value adds.
type shellAliasDef struct {
	Name  string   // "codex" or "claude"
	File  string   // candidate path as scanned (before ~ substitution)
	Line  int      // the alias statement's first physical line
	Flags []string // conflicting seat-launch flags, in order of appearance, deduplicated; empty = harmless

	// Incomplete is true when the alias statement's value was not fully
	// read -- an unterminated quote or trailing backslash that was still
	// open after scanShellAliasFile's 32-line continuation cap, or at EOF.
	// Flags reflects only what was actually read: an empty Flags here is
	// NOT a claim that the alias is harmless, only that nothing was found
	// in the partial value.
	Incomplete bool
}

// scanShellAliasFile reads one rc file and returns every codex/claude alias
// definition found, in file order, whether or not it carries a conflicting
// flag. opened reports whether the file was successfully opened at all --
// when opened is false, err came from os.Open and the file was never read;
// when opened is true and err is non-nil, err came from the scanner after
// some lines (possibly with alias definitions) were already read
// successfully, so defs is never discarded because of it. Each physical
// line is bounded at 1 MiB (bufio.Scanner's own 64 KiB default would
// otherwise abort the whole file on one long line). When a line's last
// statement is an unterminated alias statement (see parseAliasStatements'
// openAlias), up to 32 further physical lines are appended and re-parsed as
// one logical line before giving up; past that point, or at EOF, whatever
// definitions the statement yielded from what was actually read are kept
// and marked Incomplete. The reported Line is always the statement's first
// physical line, however many lines it ended up spanning.
func scanShellAliasFile(path string) (defs []shellAliasDef, opened bool, err error) {
	f, openErr := os.Open(path)
	if openErr != nil {
		return nil, false, openErr
	}
	defer func() { _ = f.Close() }()

	const maxContinuationLines = 32

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		startLine := lineNum
		joined := scanner.Text()

		assignments, openAlias := parseAliasStatements(joined)
		continuationLines := 0
		for openAlias && continuationLines < maxContinuationLines && scanner.Scan() {
			lineNum++
			continuationLines++
			joined += "\n" + scanner.Text()
			assignments, openAlias = parseAliasStatements(joined)
		}

		for _, a := range assignments {
			defs = append(defs, shellAliasDef{
				Name:       a.Name,
				File:       path,
				Line:       startLine,
				Flags:      shellAliasConflictingFlags(a.Name, a.Value),
				Incomplete: openAlias,
			})
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return defs, true, scanErr
	}
	return defs, true, nil
}

// shellAliasClassModel, shellAliasClassSandbox, shellAliasClassApproval, and
// shellAliasClassPermissionMode are shellAliasFlagClass's return values.
const (
	shellAliasClassModel          = "model"
	shellAliasClassSandbox        = "sandbox"
	shellAliasClassApproval       = "approval"
	shellAliasClassPermissionMode = "permission-mode"
)

// shellAliasFlagClass classifies a seat-launch flag label by which seats
// ralph actually passes it to, so checkShellAliases can describe the true
// blast radius instead of a blanket "permission" claim. Returns "" for a
// label this check never produces.
//
// ralph passes --model on every spawn regardless of permission mode
// (internal/org/spawn.go:643). For codex, --sandbox goes to edits and
// autonomous seats (internal/org/permissions.go's codexEditsArgs and
// codexAutonomousArgs both include it); --ask-for-approval goes to
// autonomous seats only (codexAutonomousArgs has it, codexEditsArgs does
// not). For claude, --permission-mode goes to edits and autonomous seats
// (permissionArgsForDriver's claude case). A guarded seat of either driver
// gets none of the three permission-class flags, so that class's alias
// value applies unopposed on a guarded seat.
func shellAliasFlagClass(label string) string {
	switch label {
	case "--model", "--model (-m)":
		return shellAliasClassModel
	case "--sandbox", "--sandbox (-s)":
		return shellAliasClassSandbox
	case "--ask-for-approval", "--ask-for-approval (-a)":
		return shellAliasClassApproval
	case "--permission-mode":
		return shellAliasClassPermissionMode
	default:
		return ""
	}
}

// shellAliasValueTokens re-tokenizes an alias value with shellAliasStatements
// (the same quote-aware reader used on rc lines), because zsh re-parses
// alias text on expansion: `alias codex='codex "--model" gpt-5'` really
// does pass --model to codex, with the inner double quotes stripped by the
// shell just like any other quoted word -- a plain strings.Fields split
// would miss it, since the quote characters would still be attached to the
// token. All statements' words are flattened in order, so an unquoted `;`,
// `|`, or `&` inside the value merely separates words rather than hiding
// anything, and an unquoted `#` that starts a word still ends the value
// early, matching real shell comment handling.
func shellAliasValueTokens(value string) []string {
	stmts, _ := shellAliasStatements(value)
	var tokens []string
	for _, words := range stmts {
		tokens = append(tokens, words...)
	}
	return tokens
}

// shellAliasConflictingFlags returns every seat-launch flag ralph itself
// passes to the named driver ("codex" or "claude") that also appears in an
// alias value, in order of first appearance, deduplicated. value is
// re-tokenized with shellAliasValueTokens rather than a plain whitespace
// split (see its doc comment for why).
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
// claude has no short form for either --model or --permission-mode (claude
// 2.1.274's --help lists eight short flags for other options, but none for
// these two), and does not error on a repeated --model or --permission-mode:
// verified against claude 2.1.274, the later value -- ralph's own, since
// ralph's flags are appended after the alias expands -- wins and the seat
// starts. See checkShellAliases and shellAliasFlagClass for how severity is
// graded once a flag is found: it depends on the flag's class and the
// seat's resolved permission mode, not on the driver alone.
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

	for _, tok := range shellAliasValueTokens(value) {
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
// file path (the caller already names the file next to this text):
// bufio.ErrTooLong (errors.Is) becomes "a line is longer than 1 MiB"; a
// *fs.PathError's inner Err (e.g. "permission denied") when errors.As
// matches; otherwise the error's own text.
func shellAliasUnreadableReason(err error) string {
	if errors.Is(err, bufio.ErrTooLong) {
		return "a line is longer than 1 MiB"
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error()
	}
	return err.Error()
}

// checkShellAliases is doctor's "Shell aliases (codex/claude)" check.
// resolveEnv supplies the home directory and $ZDOTDIR to scan from
// (production: doctorShellAliasEnv; tests: a closure or TestMain's pin).
//
// Severity depends on the flag's class (shellAliasFlagClass), not just the
// driver: ralph always passes --model on every spawn, so a --model finding
// is a real spawn blocker for codex (which rejects a repeated flag) and a
// real value override for claude (which accepts it and starts anyway). The
// other three classes are each passed by ralph to only some permission
// modes -- codex's --sandbox to edits and autonomous, codex's
// --ask-for-approval to autonomous only, claude's --permission-mode to
// edits and autonomous -- so a guarded seat (and, for --ask-for-approval,
// an edits seat too) gets none of it and the alias's value applies
// unopposed. That makes a codex finding warn whenever herdr is present
// (herdrPresent, since only an installed herdr expands the alias in a
// seat's pane) regardless of flag class, and makes a claude finding warn
// only for --permission-mode (a claude --model finding is always
// informational, since the worst case is ralph's own value winning, never
// a spawn failure or an unopposed alias).
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

	candidates, inaccessibleDirs := shellAliasRcCandidates(env)

	var couldNotRead []string
	for _, d := range inaccessibleDirs {
		couldNotRead = append(couldNotRead, fmt.Sprintf("%s (%s)", displayPath(d.Dir), d.Reason))
	}

	var defs []shellAliasDef
	var scannedOK []string
	var partiallyRead []string
	for _, c := range candidates {
		fileDefs, wasOpened, scanErr := scanShellAliasFile(c)
		defs = append(defs, fileDefs...)
		switch {
		case !wasOpened:
			couldNotRead = append(couldNotRead, fmt.Sprintf("%s (%s)", displayPath(c), shellAliasUnreadableReason(scanErr)))
		case scanErr != nil:
			partiallyRead = append(partiallyRead, fmt.Sprintf("%s (%s)", displayPath(c), shellAliasUnreadableReason(scanErr)))
			scannedOK = append(scannedOK, displayPath(c))
		default:
			scannedOK = append(scannedOK, displayPath(c))
		}
	}

	var codexItems, claudeItems, harmlessItems, notFullyParsed []string
	var codexHasModel, codexHasSandbox, codexHasApproval bool
	var claudeHasModel, claudeHasPermissionMode bool
	seenIncomplete := map[string]bool{}
	for _, d := range defs {
		item := fmt.Sprintf("alias %s in %s:%d", d.Name, displayPath(d.File), d.Line)
		if d.Incomplete {
			key := fmt.Sprintf("%s:%d", displayPath(d.File), d.Line)
			if !seenIncomplete[key] {
				seenIncomplete[key] = true
				notFullyParsed = append(notFullyParsed, key)
			}
		}
		if len(d.Flags) == 0 {
			if !d.Incomplete {
				harmlessItems = append(harmlessItems, item)
			}
			continue
		}
		full := item + " adds " + strings.Join(d.Flags, ", ")
		if d.Name == "codex" {
			codexItems = append(codexItems, full)
			for _, f := range d.Flags {
				switch shellAliasFlagClass(f) {
				case shellAliasClassModel:
					codexHasModel = true
				case shellAliasClassSandbox:
					codexHasSandbox = true
				case shellAliasClassApproval:
					codexHasApproval = true
				}
			}
		} else {
			claudeItems = append(claudeItems, full)
			for _, f := range d.Flags {
				switch shellAliasFlagClass(f) {
				case shellAliasClassModel:
					claudeHasModel = true
				case shellAliasClassPermissionMode:
					claudeHasPermissionMode = true
				}
			}
		}
	}
	codexFound := len(codexItems) > 0
	claudeFound := len(claudeItems) > 0

	var sentences []string
	if codexFound {
		clauses := []string{
			strings.Join(codexItems, "; ") +
				` — herdr expands the alias in the seat's pane and codex rejects a flag given twice ("cannot be used multiple times")`,
		}
		if codexHasModel {
			clauses = append(clauses, "ralph org spawn always passes --model, so every spawn fails")
		}
		if codexHasSandbox {
			clauses = append(clauses, "ralph passes --sandbox to edits and autonomous seats, "+
				"so those spawns fail while a guarded seat silently runs with the alias's sandbox")
		}
		if codexHasApproval {
			clauses = append(clauses, "ralph passes --ask-for-approval to autonomous seats only, "+
				"so those spawns fail while edits and guarded seats silently run with the alias's approval policy")
		}
		clauses = append(clauses, "remove the alias or start herdr from an alias-free rc (docs/recipes/codex-seat-permissions.md)")
		sentence := strings.Join(clauses, "; ")
		if !herdrPresent {
			sentence += " (herdr not installed, so no seat is affected today)"
		}
		sentences = append(sentences, sentence)
	}
	if claudeFound {
		clauses := []string{
			strings.Join(claudeItems, "; ") + " — claude accepts a flag given twice and the last value wins",
		}
		if claudeHasModel {
			clauses = append(clauses, "ralph org spawn always passes --model after the alias, so its value applies and the seat still starts")
		}
		if claudeHasPermissionMode {
			clauses = append(clauses, "ralph passes --permission-mode only to edits and autonomous seats, "+
				"so a guarded seat runs with the alias's permission mode")
		}
		clauses = append(clauses, "the alias's other flags reach every seat")
		sentence := strings.Join(clauses, "; ")
		if !herdrPresent && claudeHasPermissionMode {
			sentence += " (herdr not installed, so no seat is affected today)"
		}
		sentences = append(sentences, sentence)
	}
	switch {
	case codexFound || claudeFound:
		// Findings already speak for themselves; a harmless or incomplete
		// alias elsewhere is not worth a separate sentence once there is a
		// real finding.
	case len(harmlessItems) > 0:
		verb := "has"
		if len(harmlessItems) > 1 {
			verb = "have"
		}
		sentences = append(sentences, strings.Join(harmlessItems, "; ")+" "+verb+" no conflicting flags")
	case len(defs) > 0:
		// Every def found is incomplete; the not-fully-parsed clause below
		// names it, so no "no alias found" claim belongs here.
	case len(scannedOK) > 0:
		sentences = append(sentences, "no codex/claude alias found")
	}

	var scannedClause string
	switch {
	case len(candidates) == 0 && len(inaccessibleDirs) == 0:
		scannedClause = "no shell rc file found to scan"
	case len(scannedOK) == 0:
		scannedClause = "scanned 0 shell rc file(s) (files they source are not followed)"
	default:
		scannedClause = fmt.Sprintf("scanned %d shell rc file(s): %s (files they source are not followed)",
			len(scannedOK), strings.Join(scannedOK, ", "))
	}

	var detailParts []string
	detailParts = append(detailParts, sentences...)
	if len(notFullyParsed) > 0 {
		detailParts = append(detailParts, "not fully parsed: "+strings.Join(notFullyParsed, ", ")+" — the alias value continues past what was read")
	}
	if len(couldNotRead) > 0 {
		detailParts = append(detailParts, "could not read: "+strings.Join(couldNotRead, ", ")+" — aliases there were not checked")
	}
	if len(partiallyRead) > 0 {
		detailParts = append(detailParts, "partially read: "+strings.Join(partiallyRead, ", ")+" — aliases after that point were not checked")
	}
	detailParts = append(detailParts, scannedClause)
	r.Detail = strings.Join(detailParts, ". ")

	switch {
	case herdrPresent && (codexFound || claudeHasPermissionMode):
		r.Status = "warn"
	case codexFound || claudeFound || len(couldNotRead) > 0 || len(partiallyRead) > 0 || len(notFullyParsed) > 0:
		r.Status = "info"
	default:
		r.Status = "pass"
	}
	return r
}
