package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// aliasShPattern matches sh/zsh-style alias lines: `alias NAME=VALUE`, name
// glued directly to the value with no space around `=`.
var aliasShPattern = regexp.MustCompile(`^\s*alias\s+(codex|claude)=(.*)$`)

// aliasFishPattern matches fish-style alias lines: `alias NAME VALUE`, name
// and value separated by whitespace instead of `=`.
var aliasFishPattern = regexp.MustCompile(`^\s*alias\s+(codex|claude)\s+(.*)$`)

// shellAliasRcCandidates returns the rc files the check scans, in order.
// Only files that exist are read; each is resolved through
// filepath.EvalSymlinks and deduplicated by its real path (on macOS a
// ~/.zshrc symlinked to ~/.config/zsh/.zshrc must count once). The list is
// deliberately static -- files pulled in via `source` are not followed
// (documented limitation; the Detail says which files were scanned).
func shellAliasRcCandidates(home string) []string {
	var raw []string
	if zdotdir := os.Getenv("ZDOTDIR"); zdotdir != "" && filepath.IsAbs(zdotdir) {
		raw = append(raw,
			filepath.Join(zdotdir, ".zshrc"),
			filepath.Join(zdotdir, ".zshenv"),
		)
	}
	raw = append(raw,
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".config", "zsh", ".zshrc"),
		filepath.Join(home, ".zshenv"),
		filepath.Join(home, ".zprofile"),
		filepath.Join(home, ".zsh_aliases"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".bash_aliases"),
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".config", "fish", "config.fish"),
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

// shellAliasFinding is one alias line that adds a flag ralph itself passes
// when it launches a seat.
type shellAliasFinding struct {
	Name string // "codex" or "claude"
	File string // path as scanned (before ~ substitution)
	Line int
	Flag string // e.g. "--model (-m)", "--model", "--permission-mode", "--sandbox", "--ask-for-approval"
}

// scanShellAliasFile reads one rc file and returns findings plus the
// number of codex/claude alias lines seen (harmless ones included).
// Matched forms: sh/zsh `alias NAME=VALUE` (optionally quoted VALUE) and
// fish `alias NAME VALUE`; leading whitespace allowed; lines whose first
// non-space character is '#' are skipped. NAME is codex or claude.
func scanShellAliasFile(path string) (findings []shellAliasFinding, aliasLines int, err error) {
	f, openErr := os.Open(path)
	if openErr != nil {
		return nil, 0, openErr
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		var name, value string
		if m := aliasShPattern.FindStringSubmatch(line); m != nil {
			name, value = m[1], m[2]
		} else if m := aliasFishPattern.FindStringSubmatch(line); m != nil {
			name, value = m[1], m[2]
		} else {
			continue
		}

		aliasLines++
		if flag := conflictingFlag(value); flag != "" {
			findings = append(findings, shellAliasFinding{
				Name: name,
				File: path,
				Line: lineNum,
				Flag: flag,
			})
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return findings, aliasLines, scanErr
	}
	return findings, aliasLines, nil
}

// conflictingFlag reports the first seat-launch flag found in an alias
// value, tokenized on whitespace after stripping one layer of surrounding
// quotes: a token equal to --model or starting with --model= -> "--model";
// a token exactly -m, or starting with -m whose third byte is not '-'
// (e.g. -mgpt-5.5) -> "--model (-m)"; --permission-mode, --sandbox,
// --ask-for-approval (exact or with =value) -> that flag name. Returns ""
// when none is present.
func conflictingFlag(value string) string {
	v := strings.TrimSpace(value)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}

	for _, tok := range strings.Fields(v) {
		switch {
		case tok == "-m", strings.HasPrefix(tok, "-m") && len(tok) > 2 && tok[2] != '-':
			return "--model (-m)"
		case tok == "--model", strings.HasPrefix(tok, "--model="):
			return "--model"
		case tok == "--permission-mode", strings.HasPrefix(tok, "--permission-mode="):
			return "--permission-mode"
		case tok == "--sandbox", strings.HasPrefix(tok, "--sandbox="):
			return "--sandbox"
		case tok == "--ask-for-approval", strings.HasPrefix(tok, "--ask-for-approval="):
			return "--ask-for-approval"
		}
	}
	return ""
}

// checkShellAliases is doctor's "Shell aliases (codex/claude)" check.
// herdrPresent downgrades a conflict from warn to info when herdr is not
// installed (no seat can be affected today).
func checkShellAliases(herdrPresent bool) checkResult {
	r := checkResult{Name: "Shell aliases (codex/claude)"}

	home, err := os.UserHomeDir()
	if err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("could not resolve home directory: %v — shell alias check skipped", err)
		return r
	}

	candidates := shellAliasRcCandidates(home)
	var findings []shellAliasFinding
	aliasLines := 0
	scanned := 0
	for _, c := range candidates {
		fileFindings, lines, scanErr := scanShellAliasFile(c)
		if scanErr != nil {
			// Unreadable file (permissions, race with candidate listing) --
			// best-effort skip, not counted as scanned.
			continue
		}
		scanned++
		aliasLines += lines
		findings = append(findings, fileFindings...)
	}

	homePrefix := home + string(filepath.Separator)
	displayPath := func(p string) string {
		if strings.HasPrefix(p, homePrefix) {
			return "~" + string(filepath.Separator) + strings.TrimPrefix(p, homePrefix)
		}
		return p
	}

	if len(findings) > 0 {
		parts := make([]string, 0, len(findings))
		for _, f := range findings {
			parts = append(parts, fmt.Sprintf("alias %s in %s:%d adds %s", f.Name, displayPath(f.File), f.Line, f.Flag))
		}
		detail := strings.Join(parts, "; ") +
			` — herdr expands the alias in the seat's pane, so ralph org spawn's own --model/permission flags collide ` +
			`(e.g. "cannot be used multiple times"); remove the alias or start herdr from an alias-free rc ` +
			`(docs/recipes/codex-seat-permissions.md)`
		if herdrPresent {
			r.Status = "warn"
			r.Detail = detail
		} else {
			r.Status = "info"
			r.Detail = detail + " (herdr not installed, so no seat is affected today)"
		}
		return r
	}

	if aliasLines > 0 {
		r.Status = "pass"
		r.Detail = fmt.Sprintf("no conflicting codex/claude alias in %d scanned rc file(s) (%d alias line(s) seen)", scanned, aliasLines)
		return r
	}

	r.Status = "pass"
	r.Detail = fmt.Sprintf("no codex/claude alias in %d shell rc file(s)", scanned)
	return r
}
