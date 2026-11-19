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
// depends on. It is a struct (rather than reading os.UserHomeDir/os.Getenv/
// os.Getwd directly) so callers -- production and test alike -- can
// substitute values through a plain function instead of mutating real
// environment variables or the real working directory.
type shellAliasEnv struct {
	Home    string // user home directory
	Zdotdir string // $ZDOTDIR as seen by this process ("" when unset)

	// Cwd is doctor's own working directory. A relative $ZDOTDIR is
	// resolved against it ("" skips a relative $ZDOTDIR entirely); see
	// shellAliasZdotdirBase.
	Cwd string
}

// shellAliasEnvFromOS resolves shellAliasEnv from os.UserHomeDir, $ZDOTDIR,
// and os.Getwd. This is the production resolver; doctorShellAliasEnv below
// wraps it as the package-level default that runDoctorStateDir actually calls.
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

// doctorShellAliasEnv is the environment resolver runDoctorStateDir hands to
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
	// Dir is the candidate's parent directory, as an absolute path --
	// derived by shellAliasRawParent, not filepath.Dir, so a ".." is not
	// lexically resolved away; see shellAliasRcCandidates.
	Dir string

	Reason string // shellAliasUnreadableReason(err) for the stat failure
}

// shellAliasJoinRaw concatenates dir and name with exactly one path
// separator between them, without filepath.Join's lexical cleaning -- a
// ".." or "." component already present in dir reaches the OS unresolved
// (see shellAliasRcCandidates for why that matters). Every trailing
// separator on dir is trimmed first, so a doubled one never survives; dir
// consisting only of separators (e.g. "/") trims to "", and the single
// separator this function adds back still yields the right result ("/" +
// ".zshrc" = "/.zshrc").
func shellAliasJoinRaw(dir, name string) string {
	return strings.TrimRight(dir, string(filepath.Separator)) + string(filepath.Separator) + name
}

// shellAliasRawParent returns path's parent directory by trimming the last
// path-separator-delimited component, without filepath.Dir's lexical
// cleaning -- see shellAliasRcCandidates for why a ".." must reach the OS
// unresolved. A root-level path (separator at index 0, e.g. "/.zshrc")
// keeps the separator itself as its parent, matching filepath.Dir; a path
// with no separator at all is returned unchanged (no caller passes one
// today).
func shellAliasRawParent(path string) string {
	i := strings.LastIndexByte(path, filepath.Separator)
	switch {
	case i < 0:
		return path
	case i == 0:
		return path[:1]
	default:
		return path[:i]
	}
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
// next physical line. It returns only the words of each statement; the
// reader is shellAliasStatementsWithOps, which also keeps the operators
// between them.
func shellAliasStatements(line string, stopAtComment bool) (stmts [][]string, open bool) {
	withOps, _, open := shellAliasStatementsWithOps(line, stopAtComment)
	for _, s := range withOps {
		stmts = append(stmts, s.Words)
	}
	return stmts, open
}

// shellAliasStatement is one statement of a logical line as
// shellAliasStatementsWithOps reads it.
type shellAliasStatement struct {
	// Op is the operator text that came before the statement: every
	// unquoted `;`, `|`, and `&` seen since the previous statement ended,
	// concatenated ("" for the line's first statement, then ";", "&&",
	// "||", "|", "&", and so on).
	Op string

	Words []string // the statement's words, as shellAliasStatements returns them
}

// shellAliasStatementsWithOps is shellAliasStatements' reader: it splits a
// logical line by the rules described there and also returns, for each
// statement, the operator that came before it (shellAliasStatement.Op), and
// trailingOp, the operator text left after the last statement with no
// statement after it on the line (a trailing `&&`, `||`, `|`, `;`, or `&`;
// "" when the line ends on a word or a comment). shellAliasGuardTracker
// needs both to tell whether a HERDR_PANE_ID test begins its list and which
// operator joins it to the alias statement after it.
func shellAliasStatementsWithOps(line string, stopAtComment bool) (stmts []shellAliasStatement, trailingOp string, open bool) {
	var words []string
	var cur strings.Builder
	var op strings.Builder
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
			stmts = append(stmts, shellAliasStatement{Op: op.String(), Words: words})
			words = nil
			op.Reset()
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
			return stmts, op.String(), openQuote || trailingBackslash
		case c == ';' || c == '|' || c == '&':
			flushStatement()
			op.WriteByte(c)
			i++
		default:
			inWord = true
			cur.WriteByte(c)
			i++
		}
	}
	flushStatement()
	return stmts, op.String(), openQuote || trailingBackslash
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
		if args, ok := shellAliasAliasArgs(words); ok {
			defs = append(defs, parseAliasWords(args)...)
		}
	}
	if open && len(stmts) > 0 {
		_, openAlias = shellAliasAliasArgs(stmts[len(stmts)-1])
	}
	return defs, openAlias
}

// shellAliasAliasArgs reports whether a statement's words are an alias
// statement -- `alias` as the first word once any leading
// shellAliasStatementMarkers are skipped -- and returns the words after
// `alias`. parseAliasStatements and shellAliasGuardTracker.line both read
// statements through it, so the guard tracker yields its marks for exactly
// the statements, and in the order, that parseAliasStatements yields
// assignments.
func shellAliasAliasArgs(words []string) (args []string, ok bool) {
	i := 0
	for i < len(words) && shellAliasStatementMarkers[words[i]] {
		i++
	}
	if i >= len(words) || words[i] != "alias" {
		return nil, false
	}
	return words[i+1:], true
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

// shellAliasGuard is what a statement tests about HERDR_PANE_ID, the
// variable herdr sets in the interactive shell of every pane it creates
// (herdr 0.7.5), so a seat's shell has it when it reads its rc files. An rc
// can keep a codex/claude alias out of herdr's panes by testing it (issue
// #343; docs/plans/archive/2026-11-19-doctor-alias-herdr-guard.md).
type shellAliasGuard int

const (
	// shellAliasNoGuard is any statement shellAliasGuardTest does not read
	// as a HERDR_PANE_ID test.
	shellAliasNoGuard shellAliasGuard = iota

	// shellAliasGuardSet is a `-n` test: true inside a herdr pane.
	shellAliasGuardSet

	// shellAliasGuardUnset is a `-z` test: false inside a herdr pane.
	shellAliasGuardUnset
)

// shellAliasGuardTest classifies a statement's words (after quote removal,
// as shellAliasStatements yields them, so a quoted `"[["` reads like `[[`:
// one of the misreads the doc of shellAliasGuardTracker lists as accepted).
// They are a HERDR_PANE_ID test only when they are exactly `[[ F V ]]`,
// `[ F V ]`, or `test F V`, with F one of `-n` (shellAliasGuardSet) and
// `-z` (shellAliasGuardUnset) and V one of `$HERDR_PANE_ID` and
// `${HERDR_PANE_ID}`; whether V was quoted does not matter, since herdr's
// value has no whitespace. Everything else is shellAliasNoGuard on purpose
// -- a negation (`!`), `-v`, `${HERDR_PANE_ID:-}`, another variable such as
// `$HERDR_PANE`, or any extra word -- because a guard read wrongly marks an
// alias that a pane does define as one it does not, which would hide a real
// seat-breaking alias.
func shellAliasGuardTest(words []string) shellAliasGuard {
	var flag, value string
	switch {
	case len(words) == 4 && words[0] == "[[" && words[3] == "]]",
		len(words) == 4 && words[0] == "[" && words[3] == "]":
		flag, value = words[1], words[2]
	case len(words) == 3 && words[0] == "test":
		flag, value = words[1], words[2]
	default:
		return shellAliasNoGuard
	}
	if value != "$HERDR_PANE_ID" && value != "${HERDR_PANE_ID}" {
		return shellAliasNoGuard
	}
	switch flag {
	case "-n":
		return shellAliasGuardSet
	case "-z":
		return shellAliasGuardUnset
	default:
		return shellAliasNoGuard
	}
}

// shellAliasListHeadMarkers are the leading words after which a statement
// begins a new list (`then [[ -n $HERDR_PANE_ID ]] || alias ...`), so a
// HERDR_PANE_ID test after them is at the head of its list whatever came
// before. Unlike shellAliasStatementMarkers it has no `builtin`, which
// shellAliasGuardTest's forms never take.
var shellAliasListHeadMarkers = map[string]bool{
	"then": true,
	"else": true,
	"do":   true,
	"{":    true,
}

// shellAliasIfPart is the part of an if that shellAliasGuardTracker is in.
type shellAliasIfPart int

const (
	shellAliasIfCond shellAliasIfPart = iota // between `if` (or `elif`) and `then`
	shellAliasIfThen                         // between `then` and `else`, `elif`, or `fi`
	shellAliasIfElse                         // between `else` and `fi`
)

// shellAliasIfFrame is one if of an rc file that has not reached its `fi`.
type shellAliasIfFrame struct {
	// guard is the HERDR_PANE_ID test the if's condition is, or
	// shellAliasNoGuard when the condition is anything else: another
	// command, a test joined to more by `&&`, `||`, `|`, or `&`, two or
	// more statements, or an if that has had an `elif`.
	guard shellAliasGuard

	part shellAliasIfPart
}

// shellAliasGuardMark says how a HERDR_PANE_ID test keeps one alias
// assignment from being defined in a herdr pane.
type shellAliasGuardMark struct {
	// inline is the one-line form: `[[ -n $HERDR_PANE_ID ]] || alias ...`
	// or `[[ -z $HERDR_PANE_ID ]] && alias ...`.
	inline bool

	// byIf is the if form: the alias is in the then side of an
	// `if <-z test>` or the else side of an `if <-n test>`. It is only
	// provisional until EOF; see shellAliasGuardTracker.broken.
	byIf bool
}

// shellAliasGuardTracker follows one rc file, one final logical line at a
// time, to mark the alias assignments a HERDR_PANE_ID test keeps out of
// herdr's panes (docs/plans/archive/2026-11-19-doctor-alias-herdr-guard.md).
// It reads sh/zsh syntax only. When in doubt it does not mark: a mark says
// the alias is not defined in a pane, so a wrong one would hide an alias
// that does break seats.
//
// Three forms are still misread as a guard, and zsh defines the alias in a
// pane for each: a quoted test word (`"[["`, `\[\[`), because the tokenizer
// drops the quotes before shellAliasGuardTest sees the words; an rc that
// rewrites HERDR_PANE_ID before the test (`HERDR_PANE_ID=`,
// `unset HERDR_PANE_ID`); and `if`/`fi` lines inside a heredoc or a
// multi-line string, which followIf counts as the file's own when they
// balance. They are accepted (the plan's Non-goals and Risks; real rc files
// rarely use them) and recorded in docs/tech-debt/README.md.
//
// The inline form (inlineGuarded) needs to know how the previous physical
// line ended, because scanShellAliasFile joins physical lines only for an
// unterminated alias statement: after `false &&` or `false && \`, the next
// line's `[[ -n $HERDR_PANE_ID ]] || alias codex=...` continues that list,
// and zsh defines the alias inside a pane. The if form (followIf) keeps a
// stack of the file's open ifs.
type shellAliasGuardTracker struct {
	frames []shellAliasIfFrame

	// broken is set when an if keyword came where no open if could take it:
	// a `then` with no if in its condition, an `else` or `elif` with no if
	// in its then side, or a `fi` with no if or right after a condition (see
	// followIf). scanShellAliasFile drops every byIf mark of the file when
	// broken is set or frames is not empty at EOF, and keeps the inline
	// marks: an if/fi miscount (from a heredoc, a string, or a function
	// written on one line, none of which this reader understands) would
	// otherwise misplace every mark after it.
	broken bool

	// prevOpen and prevTrailingOp are how the previous physical line that
	// held anything ended: open as shellAliasStatementsWithOps reports it
	// (an unterminated quote or a trailing backslash), and its trailingOp.
	// A blank or comment-only line does not change them, since a list left
	// open by `&&`, `||`, or `|` continues past it.
	prevOpen       bool
	prevTrailingOp string
}

// line reads one final logical line (after scanShellAliasFile's joining of
// an unterminated alias statement; only the final parse is passed in, so
// each line's if keywords count once) and returns one mark per alias
// assignment on it, in the order parseAliasStatements returns them.
func (g *shellAliasGuardTracker) line(line string) []shellAliasGuardMark {
	stmts, trailingOp, open := shellAliasStatementsWithOps(line, true)
	var marks []shellAliasGuardMark
	for k, s := range stmts {
		g.followIf(k, s)
		args, isAlias := shellAliasAliasArgs(s.Words)
		if !isAlias {
			continue
		}
		mark := shellAliasGuardMark{byIf: g.inGuardedBranch()}
		if k > 0 {
			mark.inline = g.inlineGuarded(k-1, stmts[k-1], s)
		}
		for range parseAliasWords(args) {
			marks = append(marks, mark)
		}
	}
	if len(stmts) > 0 || trailingOp != "" || open {
		g.prevOpen, g.prevTrailingOp = open, trailingOp
	}
	return marks
}

// continuesPrevLine reports whether statement k of a line continues the
// previous physical line's last statement: it is the line's first
// statement, nothing separates it from that line, and that line ended open.
// Its first word is then an argument, not a command, so `then`, `else`,
// `fi`, or `{` there is no keyword.
func (g *shellAliasGuardTracker) continuesPrevLine(k int, s shellAliasStatement) bool {
	return k == 0 && s.Op == "" && g.prevOpen
}

// prevLineEndedClosed reports whether the previous physical line ended so
// that this line's first statement begins a new list: not open, and with
// no trailing operator other than `;` or `&`. Any other trailing operator
// (`&&`, `||`, `|`, `|&`, ...) leaves the list open.
func (g *shellAliasGuardTracker) prevLineEndedClosed() bool {
	return !g.prevOpen && (g.prevTrailingOp == "" || g.prevTrailingOp == ";" || g.prevTrailingOp == "&")
}

// condClosedBeforeThen reports whether the condition before a `then` that
// begins statement k (with operator op) ended with `;` or a newline. Any
// other operator joins the condition to more, or (`&`) runs it in the
// background, where its status is 0 inside a pane too.
func (g *shellAliasGuardTracker) condClosedBeforeThen(k int, op string) bool {
	if op != ";" && op != "" {
		return false
	}
	if k > 0 {
		return true
	}
	return !g.prevOpen && (g.prevTrailingOp == "" || g.prevTrailingOp == ";")
}

// top returns the innermost open if, or nil.
func (g *shellAliasGuardTracker) top() *shellAliasIfFrame {
	if len(g.frames) == 0 {
		return nil
	}
	return &g.frames[len(g.frames)-1]
}

// condNotSingleTest records that the innermost if's condition holds more
// than its first statement, when that if is still in its condition.
func (g *shellAliasGuardTracker) condNotSingleTest() {
	if top := g.top(); top != nil && top.part == shellAliasIfCond {
		top.guard = shellAliasNoGuard
	}
}

// enterPart moves the innermost if from part from to part to, or sets
// broken when there is no if or it is in another part.
func (g *shellAliasGuardTracker) enterPart(from, to shellAliasIfPart) {
	if top := g.top(); top != nil && top.part == from {
		top.part = to
		return
	}
	g.broken = true
}

// followIf applies statement k's if keywords to the stack. Leading markers
// come first, one by one: `then` moves the innermost if from its condition
// to its then side (first marking the condition as not a single test when
// the `then` is not right after `;` or a newline that ends the condition),
// `else` moves it from its then side to its else side, and `do`, `{`, and
// `builtin` change nothing. Then by the statement's next word: `if` pushes
// an if whose guard is shellAliasGuardTest of the rest of the statement;
// `elif` turns the innermost if's guard to shellAliasNoGuard and puts it
// back in its condition; `fi` pops it. Any other statement, an alias
// included, that comes while the innermost if is in its condition makes
// that condition more than one test. This is stricter than zsh in one
// direction only: `else`, `elif`, and `fi` right after a condition (a
// syntax error in sh and zsh) set broken instead of being followed.
func (g *shellAliasGuardTracker) followIf(k int, s shellAliasStatement) {
	if g.continuesPrevLine(k, s) {
		g.condNotSingleTest()
		return
	}
	words := s.Words
	j := 0
	for ; j < len(words) && shellAliasStatementMarkers[words[j]]; j++ {
		switch words[j] {
		case "then":
			if j > 0 || !g.condClosedBeforeThen(k, s.Op) {
				g.condNotSingleTest()
			}
			g.enterPart(shellAliasIfCond, shellAliasIfThen)
		case "else":
			g.enterPart(shellAliasIfThen, shellAliasIfElse)
		}
	}
	rest := words[j:]
	first := ""
	if len(rest) > 0 {
		first = rest[0]
	}
	switch first {
	case "if":
		g.condNotSingleTest() // an if inside an if's condition makes that condition more than one test
		g.frames = append(g.frames, shellAliasIfFrame{guard: shellAliasGuardTest(rest[1:]), part: shellAliasIfCond})
	case "elif":
		if top := g.top(); top != nil && top.part == shellAliasIfThen {
			top.guard, top.part = shellAliasNoGuard, shellAliasIfCond
		} else {
			g.broken = true
		}
	case "fi":
		top := g.top()
		if top == nil || top.part == shellAliasIfCond {
			g.broken = true
		}
		if top != nil {
			g.frames = g.frames[:len(g.frames)-1]
		}
	default:
		g.condNotSingleTest()
	}
}

// inGuardedBranch reports whether the current statement is, at any depth,
// in the then side of an `if <-z test>` or the else side of an
// `if <-n test>`: a side that does not run inside a herdr pane.
func (g *shellAliasGuardTracker) inGuardedBranch() bool {
	for _, f := range g.frames {
		if (f.guard == shellAliasGuardUnset && f.part == shellAliasIfThen) ||
			(f.guard == shellAliasGuardSet && f.part == shellAliasIfElse) {
			return true
		}
	}
	return false
}

// inlineGuarded reports whether the alias statement cur, right after
// statement prev (index pi) on the same line, is kept out of a herdr pane
// by prev: `[[ -n $HERDR_PANE_ID ]] || alias ...` or
// `[[ -z $HERDR_PANE_ID ]] && alias ...`. All of these must hold:
//
//   - cur starts with `alias`, directly or after `builtin`. An alias inside
//     a brace group (`[[ -n $HERDR_PANE_ID ]] || { alias codex=...; }`) is
//     not read (plan Non-goals).
//   - prev is a HERDR_PANE_ID test once leading shellAliasListHeadMarkers
//     are skipped.
//   - the operator before cur is exactly "||" after a `-n` test and exactly
//     "&&" after a `-z` test; "|" and "&" never qualify.
//   - prev is at the head of its list: it had a leading marker, or the
//     operator before it is ";", or it is the line's first statement; and in
//     the last two cases, when it is the line's first statement, the
//     previous physical line ended closed (prevLineEndedClosed). A leading
//     marker does not count when prev continues the previous physical line
//     (`foo \` then `{ test -n "$HERDR_PANE_ID" || alias ...`), where it is
//     only an argument to foo and zsh defines the alias when foo fails.
func (g *shellAliasGuardTracker) inlineGuarded(pi int, prev, cur shellAliasStatement) bool {
	w := cur.Words
	if w[0] != "alias" && (len(w) < 2 || w[0] != "builtin" || w[1] != "alias") {
		return false
	}
	pw := prev.Words
	marker := false
	for len(pw) > 0 && shellAliasListHeadMarkers[pw[0]] {
		pw = pw[1:]
		marker = true
	}
	switch guard := shellAliasGuardTest(pw); {
	case guard == shellAliasGuardSet && cur.Op == "||":
	case guard == shellAliasGuardUnset && cur.Op == "&&":
	default:
		return false
	}
	if g.continuesPrevLine(pi, prev) {
		return false
	}
	if marker {
		return true
	}
	if prev.Op != ";" && prev.Op != "" {
		return false
	}
	return pi > 0 || g.prevLineEndedClosed()
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

	// PaneGuarded is true when a HERDR_PANE_ID test keeps the alias from
	// being defined in a herdr pane, so it never reaches a seat: the inline
	// or the if form of shellAliasGuardTracker, read in sh/zsh files only (a
	// fish file's defs stay false). scanShellAliasFile sets it at EOF from
	// paneGuard, once the file's if/fi pairs are known to match.
	PaneGuarded bool

	// paneGuard is which form marked the def; scanShellAliasFile clears its
	// byIf at EOF when the file's ifs did not match up.
	paneGuard shellAliasGuardMark
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
// up spanning. Each def's PaneGuarded comes from a shellAliasGuardTracker
// that reads every final logical line once (never the partial joins of an
// open alias statement); a file whose name ends in .fish is not read for
// guards, since fish's syntax is out of scope
// (docs/plans/archive/2026-11-19-doctor-alias-herdr-guard.md, Non-goals).
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

	readGuards := filepath.Ext(path) != ".fish"
	var guards shellAliasGuardTracker

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

		var marks []shellAliasGuardMark
		if readGuards {
			marks = guards.line(joined)
		}
		if openAlias {
			result.Unclosed = append(result.Unclosed, startLine)
		}
		for i, a := range assignments {
			// marks has one entry per assignment (both read the line's
			// statements through shellAliasAliasArgs); the bound check
			// only keeps an assignment without a mark unmarked.
			var mark shellAliasGuardMark
			if i < len(marks) {
				mark = marks[i]
			}
			result.Defs = append(result.Defs, shellAliasDef{
				Name:       a.Name,
				File:       path,
				Line:       startLine,
				Flags:      shellAliasConflictingFlags(a.Name, a.Value),
				Incomplete: openAlias,
				paneGuard:  mark,
			})
		}
	}
	if guards.broken || len(guards.frames) > 0 {
		for i := range result.Defs {
			result.Defs[i].paneGuard.byIf = false
		}
	}
	for i := range result.Defs {
		result.Defs[i].PaneGuarded = result.Defs[i].paneGuard.inline || result.Defs[i].paneGuard.byIf
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
// ralph passes --model regardless of permission mode, but only to a seat
// whose [org.role_models] model is not "" or that got --model (Spawn's
// agentArgs in internal/org/spawn.go), and likewise to a headless director
// (directorAgentArgs in internal/org/director_start.go), of which only one
// started in herdr runs through the pane's shell and its aliases (a
// launch_cmd director's script execs the agent by its absolute path); a role
// whose model is "" gets no --model, so the driver CLI, and an alias's
// model, decide the model there. For codex, --sandbox goes to edits and
// autonomous seats (internal/org/permissions.go's codexEditsArgs and
// codexAutonomousArgs both include it); --ask-for-approval goes to
// autonomous seats only (codexAutonomousArgs has it, codexEditsArgs does
// not). For claude, --permission-mode goes to edits and autonomous seats
// (permissionArgsForDriver's claude case). A guarded seat of either driver
// gets none of the three permission-class flags, so that class's alias
// value applies unopposed on a guarded seat. codex's two permission-class
// flags additionally require [org.permissions].codex_verified = true
// (internal/config/config.go's CodexVerified, default true):
// permissionArgsForDriver fails codex edits/autonomous closed when an
// operator sets it to false, and Spawn rejects such a seat outright rather
// than downgrading it, so with codex_verified = false guarded is the only
// mode a codex seat can start in, while on a default project edits and
// autonomous start too.
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

// shellAliasRemedyClause is the closing clause of a warning sentence for an
// alias of name ("codex" or "claude"): its three ways out -- remove the
// alias, start herdr from an alias-free rc, or, in a bash or zsh rc (the
// `[[` test is no fish syntax, and fish rc files are not read for guards),
// define the alias only outside herdr's panes with the inline HERDR_PANE_ID
// test that scanShellAliasFile reads as PaneGuarded, one line per alias (a
// `{ ... }` group after the test is not read that way) -- and the recipe
// that describes them.
func shellAliasRemedyClause(name string) string {
	return "remove the alias, start herdr from an alias-free rc, or, in a bash or zsh rc, define it only outside herdr's panes with " +
		"[[ -n $HERDR_PANE_ID ]] || alias " + name + "=..., one line per alias (docs/recipes/codex-seat-permissions.md)"
}

// shellAliasCodexSentence assembles the codex Detail sentence from items
// (already-formatted "alias codex in <file>:<line> adds <flags>" strings,
// one per finding a herdr pane can define) and which flag classes appeared
// among them. Pure -- no filesystem, no env -- so it is testable with plain
// slices and bools. Returns "" when items is empty. Clause order: the
// opening "rejects a flag given twice" clause (always), then
// model/sandbox/approval clauses each only when that class was found, then
// the closing shellAliasRemedyClause (always); the herdr-not-installed
// suffix is appended whenever herdrPresent is false, regardless of flag
// class, since any codex finding is worth surfacing at some severity.
func shellAliasCodexSentence(items []string, hasModel, hasSandbox, hasApproval, herdrPresent bool) string {
	if len(items) == 0 {
		return ""
	}
	clauses := []string{
		strings.Join(items, "; ") +
			` — herdr expands the alias in the seat's pane and codex rejects a flag given twice ("cannot be used multiple times")`,
	}
	if hasModel {
		clauses = append(clauses, `ralph passes --model to each seat, and to a director started in herdr, whose [org.role_models] model is not "" (or that got --model), `+
			`so those launches fail, while one whose model is "" gets no --model, so the alias's -m/--model does not collide there and the alias's model applies`)
	}
	if hasSandbox {
		clauses = append(clauses, "ralph passes --sandbox to edits and autonomous seats (modes a codex seat gets only while "+
			"[org.permissions].codex_verified is true, the default), so those spawns fail while a guarded seat silently runs with the alias's sandbox")
	}
	if hasApproval {
		clauses = append(clauses, "ralph passes --ask-for-approval to autonomous seats only (a mode a codex seat gets only while "+
			"[org.permissions].codex_verified is true, the default), so those spawns fail while edits and guarded seats silently run with the alias's approval policy")
	}
	clauses = append(clauses, shellAliasRemedyClause("codex"))
	sentence := strings.Join(clauses, "; ")
	if !herdrPresent {
		sentence += " (herdr not installed, so no seat is affected today)"
	}
	return sentence
}

// shellAliasClaudeSentence is shellAliasCodexSentence's claude counterpart.
// Unlike codex, the closing shellAliasRemedyClause and the
// herdr-not-installed suffix are appended only when hasPermissionMode is
// also true: a claude --model-only finding never warns in the first place
// (see checkShellAliases' status switch), so there is nothing for either to
// qualify when only --model was found.
func shellAliasClaudeSentence(items []string, hasModel, hasPermissionMode, herdrPresent bool) string {
	if len(items) == 0 {
		return ""
	}
	clauses := []string{
		strings.Join(items, "; ") + " — claude accepts a flag given twice and the last value wins",
	}
	if hasModel {
		clauses = append(clauses, `ralph passes --model after the alias to each seat, and to a director started in herdr, whose [org.role_models] model is not "" (or that got --model), `+
			`so ralph's value applies and the seat still starts, while one whose model is "" gets no --model, so the alias's model applies`)
	}
	if hasPermissionMode {
		clauses = append(clauses, "ralph passes --permission-mode only to edits and autonomous seats, "+
			"so a guarded seat runs with the alias's permission mode")
	}
	clauses = append(clauses, "the alias's other flags reach every seat")
	if hasPermissionMode {
		clauses = append(clauses, shellAliasRemedyClause("claude"))
	}
	sentence := strings.Join(clauses, "; ")
	if !herdrPresent && hasPermissionMode {
		sentence += " (herdr not installed, so no seat is affected today)"
	}
	return sentence
}

// shellAliasPaneGuardedSentence assembles the Detail sentence for the
// findings a HERDR_PANE_ID test keeps out of herdr's panes (shellAliasDef's
// PaneGuarded), from items formatted as the codex and claude sentences'
// items are, codex and claude alike, in file order. Returns "" when items is
// empty. It carries no herdr-not-installed suffix: with or without herdr, no
// seat gets these flags, and checkShellAliases reports them as info.
func shellAliasPaneGuardedSentence(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, "; ") +
		" — not defined in herdr's panes; the rc defines each one only where HERDR_PANE_ID is empty (a HERDR_PANE_ID test guards it), " +
		"and herdr sets HERDR_PANE_ID in the shell of each pane it opens (checked on herdr 0.7.5), " +
		"so no seat, and no director started in herdr, gets its flags"
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
// codex/claude/pane-guarded/harmless/no-alias sentences, then (each only
// when non-empty) the not-fully-parsed clause, the could-not-read clause,
// the partially-read clause, and finally scannedClause, which is always
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
// resolveEnv supplies the home directory, $ZDOTDIR, and the working
// directory a relative $ZDOTDIR is resolved against, to scan from
// (production: doctorShellAliasEnv; tests: a closure or TestMain's pin).
//
// Severity depends on the flag's class (shellAliasFlagClass), not just the
// driver: ralph passes --model to every seat, and to a director started in
// herdr, whose [org.role_models] model is not "" (Spawn's agentArgs,
// directorAgentArgs),
// which the built-in leader, implementer, and reviewer entries all have, so
// a --model finding is a real spawn blocker for codex (which rejects a
// repeated flag) on those launches and a real value override for claude
// (which accepts it and starts anyway); a role whose model is "" gets no
// --model, so there the alias's model applies, on either driver. The
// other three classes are each passed by ralph to only some permission
// modes -- codex's --sandbox to edits and autonomous, codex's
// --ask-for-approval to autonomous only, claude's --permission-mode to
// edits and autonomous -- so a guarded seat (and, for --ask-for-approval,
// an edits seat too) gets none of it and the alias's value applies
// unopposed. That makes a codex finding a pane can define (below) warn
// whenever herdr is present (herdrPresent, since only an installed herdr
// expands the alias in a seat's pane) regardless of flag class, and makes
// such a claude finding warn only for --permission-mode (a claude --model
// finding is always informational, since it never fails a spawn: ralph's
// own value wins where ralph passes --model, and where it passes none, for
// a model of "", the alias's model takes the place of the CLI's default,
// which is what "" asks for).
//
// A finding a HERDR_PANE_ID test keeps out of herdr's panes (shellAliasDef's
// PaneGuarded, on a fully read def) is informational and never warns,
// whatever its flags: herdr sets HERDR_PANE_ID in each pane's shell, so no
// seat, and no director started in herdr, gets the alias. Such findings get
// their own sentence and stay out of the codex and claude sentences, so a
// warning lists only the findings a pane can define, and a pane-guarded
// finding counts as a finding for the harmless and no-alias sentences
// (docs/plans/archive/2026-11-19-doctor-alias-herdr-guard.md). The Detail
// sentences are built by shellAliasCodexSentence, shellAliasClaudeSentence,
// and shellAliasPaneGuardedSentence; the read-status clauses by
// shellAliasDetail and shellAliasScannedClause.
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

	var codexItems, claudeItems, paneGuardedItems, harmlessItems []string
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
		if d.PaneGuarded && !d.Incomplete {
			// An incomplete def keeps the handling it had before the
			// HERDR_PANE_ID reading: its value was not fully read, so it
			// stays a finding a pane can define.
			paneGuardedItems = append(paneGuardedItems, full)
			continue
		}
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
	paneGuardedFound := len(paneGuardedItems) > 0

	var sentences []string
	if s := shellAliasCodexSentence(codexItems, codexHasModel, codexHasSandbox, codexHasApproval, herdrPresent); s != "" {
		sentences = append(sentences, s)
	}
	if s := shellAliasClaudeSentence(claudeItems, claudeHasModel, claudeHasPermissionMode, herdrPresent); s != "" {
		sentences = append(sentences, s)
	}
	if s := shellAliasPaneGuardedSentence(paneGuardedItems); s != "" {
		sentences = append(sentences, s)
	}
	switch {
	case codexFound || claudeFound || paneGuardedFound:
		// Findings already speak for themselves; a harmless or incomplete
		// alias elsewhere is not worth a separate sentence once there is a
		// real finding, pane-guarded or not.
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
	case codexFound || claudeFound || paneGuardedFound || len(couldNotRead) > 0 || len(partiallyRead) > 0 || len(notFullyParsed) > 0:
		r.Status = "info"
	default:
		r.Status = "pass"
	}
	return r
}
