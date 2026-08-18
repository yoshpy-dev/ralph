package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

func newUpgradeCmd() *cobra.Command {
	var (
		dryRun      bool
		diffPreview bool
		pager       string
		yes         bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update scaffold files to the latest template version",
		Long: `Compares the current project files against the embedded templates and
applies a fully non-interactive upgrade: core files are replaced, managed
blocks (AGENTS.md, .gitignore) and .claude/settings.json are merged in
place, and drift or fork paths are left untouched.

A v2-layout (overlay scaffold) project (.ralph/manifest.toml with
meta.layout = "v2") upgrades via this non-interactive engine directly. A
legacy (pre-v2) manifest is migrated to the v2 layout first: a git-clean
work tree is required (git is the migration's rollback mechanism), a
preview is shown, and — unless --yes or --dry-run is given — a y/N
confirmation is required before any file is written. The migration then
chains directly into the same non-interactive v2 upgrade.

Exit codes: 0 success, 1 execution error, 3 completed with unresolved drift
remaining (see the upgrade report and stderr for the affected paths).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if diffPreview {
				dryRun = true
			}
			return runUpgradeWithOptions(".", upgradeOptions{
				DryRun:      dryRun,
				DiffPreview: diffPreview,
				Pager:       pager,
				Yes:         yes,
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview upgrade actions without writing files")
	cmd.Flags().BoolVar(&diffPreview, "diff", false, "show advisory diffs without writing files (implies --dry-run)")
	cmd.Flags().StringVar(&pager, "pager", pagerAuto, "dry-run diff pager mode: auto, always, or never")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the legacy migration confirmation prompt (no effect on a v2-layout project)")

	return cmd
}

const (
	pagerAuto   = "auto"
	pagerAlways = "always"
	pagerNever  = "never"
)

type upgradeOptions struct {
	DryRun      bool
	DiffPreview bool
	Pager       string
	// Yes skips the legacy-migration y/N confirmation prompt
	// (runMigrateLegacy, migrate.go). It has no effect on a v2-layout
	// project, which never prompts.
	Yes bool
}

// legacyLayoutFailClosedMsg is returned by `ralph init` (re-init) and
// `ralph pack add` whenever they target a manifest that predates the
// overlay (v2) layout: neither command performs the migration itself —
// `ralph upgrade` does (runMigrateLegacy, migrate.go) — so both remain a
// clean, zero-write refusal that points the operator at it.
const legacyLayoutFailClosedMsg = "this project uses the legacy scaffold layout; run `ralph upgrade` to migrate it to the overlay (v2) layout (a clean git work tree is required); no files were changed"

// errLegacyLayoutFailClosed is the sentinel returned as-is (never wrapped)
// by every legacy-manifest refusal outside `ralph upgrade` itself
// (internal/cli/init.go, internal/cli/pack.go), so callers (and tests) can
// match on it directly with errors.Is or ==. `ralph upgrade` no longer
// returns this sentinel: it migrates a legacy manifest instead of refusing
// it (see runUpgradeIOWithOptions below).
var errLegacyLayoutFailClosed = errors.New(legacyLayoutFailClosedMsg)

func runUpgradeWithOptions(targetDir string, opts upgradeOptions) error {
	return runUpgradeIOWithOptions(targetDir, opts, os.Stdin, os.Stdout, os.Stderr, shouldColorize(os.Stdout))
}

// shouldColorize reports whether ANSI color escapes should be emitted for the
// given output. Honors the de-facto NO_COLOR standard (https://no-color.org)
// and only enables color when writing directly to a terminal — pipes,
// redirects, and CI capture all stay plain text.
func shouldColorize(out *os.File) bool {
	if v := os.Getenv("NO_COLOR"); v != "" {
		return false
	}
	if out == nil {
		return false
	}
	return term.IsTerminal(out.Fd())
}

// runUpgradeIOWithOptions is the entry point for both the `ralph upgrade`
// command and every test in this package. There are exactly two outcomes:
//
//   - meta.layout == "v2": dispatch to the non-interactive v2 upgrade engine
//     (runUpgradeV2, internal/cli/upgrade_v2.go) — no prompts, no stdin
//     reads, on this branch.
//   - anything else (legacy manifest, or a manifest with no layout at all):
//     dispatch to the legacy-to-v2 migration flow (runMigrateLegacy,
//     internal/cli/migrate.go), which reads a y/N confirmation from `in`
//     (unless opts.Yes or opts.DryRun) before writing anything, then chains
//     into the same v2 upgrade engine.
func runUpgradeIOWithOptions(targetDir string, opts upgradeOptions, in io.Reader, out, errOut io.Writer, colorize bool) error {
	if opts.Pager == "" {
		opts.Pager = pagerAuto
	}
	if opts.DiffPreview {
		opts.DryRun = true
	}
	if err := validatePagerMode(opts.Pager); err != nil {
		return err
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	manifestPath := filepath.Join(absDir, ".ralph", "manifest.toml")
	if _, err := os.Stat(manifestPath); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no .ralph/manifest.toml found — run 'ralph init' first")
	}

	oldManifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	if oldManifest.Meta.Layout != scaffold.LayoutV2 {
		return runMigrateLegacy(absDir, manifestPath, oldManifest, opts, in, out, errOut, colorize, time.Now().UTC())
	}

	return runUpgradeV2(absDir, manifestPath, oldManifest, opts, out, errOut, colorize, time.Now().UTC())
}

func validatePagerMode(mode string) error {
	switch mode {
	case pagerAuto, pagerAlways, pagerNever:
		return nil
	default:
		return fmt.Errorf("invalid --pager %q (want auto, always, or never)", mode)
	}
}

// writef is a best-effort write for progress text. The write destination is an
// io.Writer (for testability) so the static-analyzer cannot rule out a failing
// write the way it can for os.Stdout — silence the error explicitly here
// rather than sprinkling `_, _ =` across every call site.
func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func writeDiffOutput(diff string, out, errOut io.Writer, pagerMode string) {
	if !shouldUsePager(pagerMode, out) {
		_, _ = io.WriteString(out, diff)
		return
	}
	if err := writeThroughPager(diff, out, errOut); err != nil {
		writef(errOut, "    (pager failed: %v — writing diff directly)\n", err)
		_, _ = io.WriteString(out, diff)
	}
}

func shouldUsePager(pagerMode string, out io.Writer) bool {
	switch pagerMode {
	case pagerNever:
		return false
	case pagerAlways:
		return true
	case pagerAuto:
		f, ok := out.(*os.File)
		return ok && term.IsTerminal(f.Fd())
	default:
		return false
	}
}

func writeThroughPager(diff string, out, errOut io.Writer) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -R"
	}
	parts := strings.Fields(pager)
	if len(parts) == 0 {
		return fmt.Errorf("empty pager")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = strings.NewReader(diff)
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
