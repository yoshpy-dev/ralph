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
	Cwd     string // doctor's own working directory, for resolving a relative $ZDOTDIR ("" skips a relative $ZDOTDIR entirely; see shellAliasZdotdirBase)
}

// shellAliasEnvFromOS resolves shellAliasEnv from os.UserHomeDir, $ZDOTDIR,
// and os.Getwd. This is the production resolver; doctorShellAliasEnv below
// wraps it as the package-level default that runDoctorFull actually calls.
// A failed os.Getwd leaves Cwd empty rather than failing env resolution --
// see shellAliasZdotdirBase for what an empty Cwd means for a relative
// $ZDOTDIR.
func shellAliasEnvFromOS() (shellAliasEnv, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return shellAliasEnv{}, err
	}
	cwd, _ := os.Getwd()
	return shellAliasEnv{Home: home, Zdotdir: os.Getenv("ZDOTDIR"), Cwd: cwd}, nil
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
	Dir    string // the candidate's parent directory, as an absolute path -- derived by shellAliasRawParent, not filepath.Dir, so a ".." inside a $ZDOTDIR-derived candidate is not lexically resolved away
	Reason string // shellAliasUnreadableReason(err) for the stat failure
}

// shellAliasJoinRaw concatenates dir and name with exactly one path
// separator between them, without filepath.Join's lexical cleaning: a ".."
// or "." component already present in dir is left untouched for the OS to
// resolve at stat/open time, rather than being lexically simplified away
// before the filesystem ever sees it. A single trailing separator on dir is
// trimmed first so the result never doubles it.
func shellAliasJoinRaw(dir, name string) string {
	return strings.TrimSuffix(dir, string(filepath.Separator)) + string(filepath.Separator) + name
}

// shellAliasRawParent returns path's parent directory by trimming the last
// path-separator-delimited component, without filepath.Dir's lexical
// cleaning -- filepath.Dir would resolve a ".." component in a $ZDOTDIR-
// derived candidate (see shellAliasZdotdirBase) before the OS ever sees it,
// defeating the point of having built that candidate by concatenation.
func shellAliasRawParent(path string) string {
	if i := strings.LastIndexByte(path, filepath.Separator); i >= 0 {
		return path[:i]
	}
	return path
}

// shellAliasZdotdirBase returns the directory this process's $ZDOTDIR
// resolves to, as a raw (uncleaned) string, or "" when there is nothing to
// scan under it: $ZDOTDIR is unset, or it is relative and env.Cwd is empty
// (shellAliasEnvFromOS leaves Cwd empty when os.Getwd failed). A relative
// $ZDOTDIR is resolved against env.Cwd with shellAliasJoinRaw, never
// filepath.Join -- see shellAliasRcCandidates' doc comment for why a ".."
// inside $ZDOTDIR must reach the OS unresolved.
func shellAliasZdotdirBase(env shellAliasEnv) string {
	if env.Zdotdir == "" {
		return ""
	}
	if filepath.IsAbs(env.Zdotdir) {
		return env.Zdotdir
	}
	if env.Cwd == "" {
		return ""
	}
	return shellAliasJoinRaw(env.Cwd, env.Zdotdir)
}

// shellAliasRcCandidates returns the rc files the check scans, in order,
// plus any candidate directory whose stat failed inconclusively (see
// shellAliasInaccessibleDir). Only non-directory paths that exist become
// candidates -- a directory is silently skipped, and scanShellAliasFile (not
// this function) is what rejects a candidate that stats successfully but is
// not a regular file, see its doc comment. Each candidate is resolved
// through filepath.EvalSymlinks and deduplicated by its real path (on macOS
// a ~/.zshrc symlinked to ~/.config/zsh/.zshrc must count once -- the
// reported file:line still names the symlink candidate, not the resolved
// target, since dedup keeps the first candidate in list order, not the
// resolved one). The list is deliberately static -- files pulled in via
// `source` are not followed; checkShellAliases' Detail names every file it
// actually scanned, so a `source`d alias reads as outside what was
// checked, not as a silent miss.
//
// herdr's pane runs a login zsh, which reads .zshenv, .zprofile, .zshrc,
// and .zlogin -- in that load order -- from $ZDOTDIR, falling back to
// $HOME when $ZDOTDIR is unset. zsh resolves a relative $ZDOTDIR against
// the shell's own working directory, not against $HOME. This check has no
// way to observe the herdr pane's actual working directory (the seat's
// --cwd), so it approximates with doctor's own working directory instead
// (env.Cwd, filled by shellAliasEnvFromOS from os.Getwd -- see
// shellAliasZdotdirBase). The candidate list scans those four files under
// each of $ZDOTDIR (set, and either absolute or relative with a non-empty
// env.Cwd to resolve it against -- otherwise skipped entirely), $HOME, and
// $HOME/.config/zsh, in that order: $HOME and $HOME/.config/zsh are
// scanned even when $ZDOTDIR is set and zsh itself would skip them,
// because this process's own resolution of $ZDOTDIR is not guaranteed to
// match what herdr's login-zsh pane actually reads (a relative $ZDOTDIR
// resolved from doctor's cwd is an even weaker guarantee than an absolute
// one), so the list intentionally over-approximates rather than
// under-approximates. Every $ZDOTDIR-derived candidate is built by string
// concatenation (shellAliasJoinRaw), never filepath.Join: zsh hands
// $ZDOTDIR/.zshrc to the OS exactly as written, so a ".." inside $ZDOTDIR
// is resolved by the OS at stat/open time -- against wherever a preceding
// symlink component actually points, if any -- rather than lexically
// removed beforehand the way filepath.Join would. After the zsh files come
// ~/.zsh_aliases, the bash files (.bashrc, .bash_profile, .bash_login,
// .bash_aliases, .profile), and finally ~/.config/fish/config.fish.
func shellAliasRcCandidates(env shellAliasEnv) ([]string, []shellAliasInaccessibleDir) {
	var raw []string

	if base := shellAliasZdotdirBase(env); base != "" {
		for _, f := range []string{".zshenv", ".zprofile", ".zshrc", ".zlogin"} {
			raw = append(raw, shellAliasJoinRaw(base, f))
		}
	}
	for _, dir := range []string{env.Home, filepath.Join(env.Home, ".config", "zsh")} {
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
			dir := shellAliasRawParent(c)
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
// lines already joined by scanShellAliasFile, or an alias value re-read by
// shellAliasValueTokens) into shell statements, each a slice of words. Word
// splitting understands three states: unquoted, single-quoted, and
// double-quoted. Unquoted and double-quoted text treat a backslash as
// escaping the next byte (the backslash itself is dropped); single-quoted
// text has no escapes. Quote characters delimit a segment and are dropped
// from the result; adjacent segments concatenate into one word regardless
// of which state produced them. Unquoted whitespace -- space, tab, or a raw
// newline/carriage return, which reach this reader once a multi-line value
// or joined rc lines are re-scanned -- ends a word; inside a quote none of
// that is special, so a quoted segment that spans a join point keeps its
// embedded newline as literal content. An unquoted `;`, `|`, or `&` ends
// the current statement -- a run like `&&` or `||` produces no empty
// statement in between (empty statements are dropped). Quotes protect all
// of `;`, `|`, `&`, and `#` from ending anything. stopAtComment controls
// whether an unquoted `#` at the very start of a word ends scanning
// entirely (true: the rest of the line, including any later statement, is
// a trailing comment -- the right rule for an rc line, always read
// non-interactively; false: `#` is an ordinary word character -- the right
// rule for shellAliasValueTokens, since herdr's pane is an interactive zsh
// with interactivecomments off by default, so `#` loses its comment
// meaning there). In unquoted or double-quoted state, a backslash
// immediately followed by a newline is a shell line-continuation: both
// bytes are dropped, joining the surrounding text seamlessly -- this is
// what lets scanShellAliasFile join two physical lines with a literal "\n"
// and have the result read as one statement. open reports whether the line
// ended before its last statement closed: either inside an unterminated
// quote, or on a bare trailing backslash with nothing after it (no
// newline) -- both are scanShellAliasFile's signal to read and append the
// next physical line.
func shellAliasStatements(line string, stopAtComment bool) (stmts [][]string, open bool) {
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
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
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
		case c == '#' && !inWord && stopAtComment:
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
	stmts, open := shellAliasStatements(line, true)
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

// shellAliasFileScan is scanShellAliasFile's result.
type shellAliasFileScan struct {
	Defs []shellAliasDef

	// Unclosed holds the first physical line of every alias statement that
	// was still open (unterminated quote or trailing backslash) at
	// scanShellAliasFile's 32-line continuation cap or at EOF, INCLUDING
	// one that defined no codex/claude assignment at all (e.g. `alias
	// msg='don` swallows the rest of the file into `msg`'s value with no
	// Defs entry to carry Incomplete). checkShellAliases folds this
	// together with any Incomplete def's file:line into one "not fully
	// parsed" clause. NOT covered: an unbalanced quote that a *later*
	// line's own quote happens to close swallows the lines in between
	// without any signal here -- zsh itself would report that rc as
	// broken, and there is no local marker for it in a single statement's
	// open/closed state.
	Unclosed []int

	// Opened reports whether the file was successfully opened at all --
	// when false, Err came from the leading os.Stat (a stat failure, or a
	// synthetic "not a regular file" error for a candidate that exists but
	// is a FIFO, device, or socket) or from os.Open, and the file was
	// never read.
	Opened bool

	// Err is non-nil either when Opened is false (see above) or when the
	// scanner failed partway through; in the latter case Defs and Unclosed
	// are never discarded because of it -- they hold whatever was read
	// before the failure.
	Err error
}

// scanShellAliasFile reads one rc file and returns every codex/claude alias
// definition found, in file order, whether or not it carries a conflicting
// flag (shellAliasFileScan.Defs), plus every alias statement that was never
// fully read (shellAliasFileScan.Unclosed). path is stat'd first: a path
// that stats successfully but is not a regular file (a FIFO, device, or
// socket -- a directory candidate is already filtered out by
// shellAliasRcCandidates) is reported as an error without ever being
// opened, since opening a FIFO can block forever waiting for a writer to
// attach (mirrors readCodexUserConfig in doctor_codex_writable_root.go).
// Each physical line is bounded at 1 MiB (bufio.Scanner's own 64 KiB
// default would otherwise abort the whole file on one long line). When a
// line's last statement is an unterminated alias statement (see
// parseAliasStatements' openAlias), up to 32 further physical lines are
// appended and re-parsed as one logical line before giving up; past that
// point, or at EOF, whatever definitions the statement yielded from what
// was actually read are kept and marked Incomplete, and the statement's
// first line is recorded in Unclosed regardless of whether it yielded any
// definition. The reported Line (on a def) or line number (in Unclosed) is
// always the statement's first physical line, however many lines it ended
// up spanning.
func scanShellAliasFile(path string) shellAliasFileScan {
	info, statErr := os.Stat(path)
	if statErr != nil {
		return shellAliasFileScan{Opened: false, Err: statErr}
	}
	if !info.Mode().IsRegular() {
		return shellAliasFileScan{Opened: false, Err: errors.New("not a regular file")}
	}

	f, openErr := os.Open(path)
	if openErr != nil {
		return shellAliasFileScan{Opened: false, Err: openErr}
	}
	defer func() { _ = f.Close() }()

	const maxContinuationLines = 32

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var result shellAliasFileScan
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

		if openAlias {
			result.Unclosed = append(result.Unclosed, startLine)
		}
		for _, a := range assignments {
			result.Defs = append(result.Defs, shellAliasDef{
				Name:       a.Name,
				File:       path,
				Line:       startLine,
				Flags:      shellAliasConflictingFlags(a.Name, a.Value),
				Incomplete: openAlias,
			})
		}
	}
	result.Opened = true
	result.Err = scanner.Err()
	return result
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
// value applies unopposed on a guarded seat. codex's two permission-class
// flags additionally require [org.permissions].codex_verified = true
// (internal/config/config.go's CodexVerified, default false):
// permissionArgsForDriver fails codex edits/autonomous closed until an
// operator sets it, and Spawn rejects such a seat outright rather than
// downgrading it, so on a default project guarded is the only mode a codex
// seat can start in.
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
// anything. Unlike an rc line, an unquoted `#` does NOT end the value early
// (stopAtComment is false): herdr's pane is an interactive zsh, where
// interactivecomments is off by default, so a `#` in the expanded alias
// text is an ordinary argument, not a comment marker -- `codex #keep
// --model x` really does pass --model to codex.
func shellAliasValueTokens(value string) []string {
	stmts, _ := shellAliasStatements(value, false)
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

// shellAliasCodexSentence assembles the codex Detail sentence from items
// (already-formatted "alias codex in <file>:<line> adds <flags>" strings,
// one per finding) and which flag classes appeared among them. Pure -- no
// filesystem, no env -- so it is testable with plain slices and bools.
// Returns "" when items is empty. Clause order: the opening "rejects a
// flag given twice" clause (always), then model/sandbox/approval clauses
// each only when that class was found, then the closing "remove the
// alias" clause (always); the herdr-not-installed suffix is appended
// whenever herdrPresent is false, regardless of flag class, since any
// codex finding is worth surfacing at some severity.
func shellAliasCodexSentence(items []string, hasModel, hasSandbox, hasApproval, herdrPresent bool) string {
	if len(items) == 0 {
		return ""
	}
	clauses := []string{
		strings.Join(items, "; ") +
			` — herdr expands the alias in the seat's pane and codex rejects a flag given twice ("cannot be used multiple times")`,
	}
	if hasModel {
		clauses = append(clauses, "ralph org spawn always passes --model, so every spawn fails")
	}
	if hasSandbox {
		clauses = append(clauses, "ralph passes --sandbox to edits and autonomous seats (modes a codex seat gets only once "+
			"[org.permissions].codex_verified = true), so those spawns fail while a guarded seat silently runs with the alias's sandbox")
	}
	if hasApproval {
		clauses = append(clauses, "ralph passes --ask-for-approval to autonomous seats only (a mode a codex seat gets only once "+
			"[org.permissions].codex_verified = true), so those spawns fail while edits and guarded seats silently run with the alias's approval policy")
	}
	clauses = append(clauses, "remove the alias or start herdr from an alias-free rc (docs/recipes/codex-seat-permissions.md)")
	sentence := strings.Join(clauses, "; ")
	if !herdrPresent {
		sentence += " (herdr not installed, so no seat is affected today)"
	}
	return sentence
}

// shellAliasClaudeSentence is shellAliasCodexSentence's claude counterpart.
// Unlike codex, the herdr-not-installed suffix is appended only when
// hasPermissionMode is also true: a claude --model-only finding never
// warns in the first place (see checkShellAliases' status switch), so
// there is nothing for the suffix to qualify when herdr is absent and only
// --model was found.
func shellAliasClaudeSentence(items []string, hasModel, hasPermissionMode, herdrPresent bool) string {
	if len(items) == 0 {
		return ""
	}
	clauses := []string{
		strings.Join(items, "; ") + " — claude accepts a flag given twice and the last value wins",
	}
	if hasModel {
		clauses = append(clauses, "ralph org spawn always passes --model after the alias, so its value applies and the seat still starts")
	}
	if hasPermissionMode {
		clauses = append(clauses, "ralph passes --permission-mode only to edits and autonomous seats, "+
			"so a guarded seat runs with the alias's permission mode")
	}
	clauses = append(clauses, "the alias's other flags reach every seat")
	sentence := strings.Join(clauses, "; ")
	if !herdrPresent && hasPermissionMode {
		sentence += " (herdr not installed, so no seat is affected today)"
	}
	return sentence
}

// shellAliasScannedClause renders checkShellAliases' always-present,
// always-last Detail clause naming what was actually scanned.
// candidateCount and inaccessibleDirCount are len(candidates) and
// len(inaccessibleDirs) from shellAliasRcCandidates; scannedOK is the
// display-path list of files that were at least opened (successfully or
// partially read -- see checkShellAliases).
func shellAliasScannedClause(candidateCount, inaccessibleDirCount int, scannedOK []string) string {
	switch {
	case candidateCount == 0 && inaccessibleDirCount == 0:
		return "no shell rc file found to scan"
	case len(scannedOK) == 0:
		return "scanned 0 shell rc file(s) (files they source are not followed)"
	default:
		return fmt.Sprintf("scanned %d shell rc file(s): %s (files they source are not followed)",
			len(scannedOK), strings.Join(scannedOK, ", "))
	}
}

// shellAliasDetail joins checkShellAliases' ordered Detail pieces: the
// codex/claude/harmless/no-alias sentences, then (each only when non-empty)
// the not-fully-parsed clause, the could-not-read clause, the
// partially-read clause, and finally scannedClause, which is always
// present.
func shellAliasDetail(sentences, notFullyParsed, couldNotRead, partiallyRead []string, scannedClause string) string {
	var parts []string
	parts = append(parts, sentences...)
	if len(notFullyParsed) > 0 {
		parts = append(parts, "not fully parsed: "+strings.Join(notFullyParsed, ", ")+" — the alias value continues past what was read")
	}
	if len(couldNotRead) > 0 {
		parts = append(parts, "could not read: "+strings.Join(couldNotRead, ", ")+" — aliases there were not checked")
	}
	if len(partiallyRead) > 0 {
		parts = append(parts, "partially read: "+strings.Join(partiallyRead, ", ")+" — aliases after that point were not checked")
	}
	parts = append(parts, scannedClause)
	return strings.Join(parts, ". ")
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
// a spawn failure or an unopposed alias). The Detail sentences are built by
// shellAliasCodexSentence/shellAliasClaudeSentence; the read-status clauses
// by shellAliasDetail and shellAliasScannedClause.
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

	// notFullyParsed is built here, per candidate file, from
	// shellAliasFileScan.Unclosed -- an alias statement that never closed,
	// whether or not it yielded a codex/claude assignment (C2-2). It is
	// deduplicated by file:line since the same statement's Unclosed entry
	// and any Incomplete def it produced would otherwise name the same
	// line twice.
	var defs []shellAliasDef
	var scannedOK []string
	var partiallyRead []string
	var notFullyParsed []string
	seenUnclosed := map[string]bool{}
	for _, c := range candidates {
		scan := scanShellAliasFile(c)
		defs = append(defs, scan.Defs...)
		for _, line := range scan.Unclosed {
			key := fmt.Sprintf("%s:%d", displayPath(c), line)
			if !seenUnclosed[key] {
				seenUnclosed[key] = true
				notFullyParsed = append(notFullyParsed, key)
			}
		}
		switch {
		case !scan.Opened:
			couldNotRead = append(couldNotRead, fmt.Sprintf("%s (%s)", displayPath(c), shellAliasUnreadableReason(scan.Err)))
		case scan.Err != nil:
			partiallyRead = append(partiallyRead, fmt.Sprintf("%s (%s)", displayPath(c), shellAliasUnreadableReason(scan.Err)))
			scannedOK = append(scannedOK, displayPath(c))
		default:
			scannedOK = append(scannedOK, displayPath(c))
		}
	}

	var codexItems, claudeItems, harmlessItems []string
	var codexHasModel, codexHasSandbox, codexHasApproval bool
	var claudeHasModel, claudeHasPermissionMode bool
	for _, d := range defs {
		item := fmt.Sprintf("alias %s in %s:%d", d.Name, displayPath(d.File), d.Line)
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
	if s := shellAliasCodexSentence(codexItems, codexHasModel, codexHasSandbox, codexHasApproval, herdrPresent); s != "" {
		sentences = append(sentences, s)
	}
	if s := shellAliasClaudeSentence(claudeItems, claudeHasModel, claudeHasPermissionMode, herdrPresent); s != "" {
		sentences = append(sentences, s)
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
	case len(defs) > 0 || len(notFullyParsed) > 0:
		// Either every def found is incomplete, or an alias statement never
		// closed at all (with or without any codex/claude assignment); the
		// not-fully-parsed clause below names it, so no "no alias found"
		// claim belongs here.
	case len(scannedOK) > 0:
		sentences = append(sentences, "no codex/claude alias found")
	}

	scannedClause := shellAliasScannedClause(len(candidates), len(inaccessibleDirs), scannedOK)
	r.Detail = shellAliasDetail(sentences, notFullyParsed, couldNotRead, partiallyRead, scannedClause)

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
