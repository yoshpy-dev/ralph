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
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update scaffold files to the latest template version",
		Long: `Compares the current project files against the embedded templates and
applies a fully non-interactive upgrade: core files are replaced, managed
blocks (AGENTS.md, .gitignore) and .claude/settings.json are merged in
place, and drift or fork paths are left untouched.

Requires a v2-layout (overlay scaffold) project (.ralph/manifest.toml with
meta.layout = "v2"). Legacy (pre-v2) manifests are rejected with zero
writes — the automated migration to the v2 layout arrives in a later ralph
release (Phase 4).

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
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview upgrade actions without writing files")
	cmd.Flags().BoolVar(&diffPreview, "diff", false, "show conflict diffs without writing files (implies --dry-run)")
	cmd.Flags().StringVar(&pager, "pager", pagerAuto, "dry-run diff pager mode: auto, always, or never")

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
}

// legacyLayoutFailClosedMsg is returned whenever a ralph write operation
// (upgrade, pack add, re-init) targets a manifest that predates the overlay
// (v2) layout. The legacy interactive upgrade engine and the baseline
// mechanism it depended on were removed in Phase 3 (docs/plans/active
// /2026-08-18-overlay-scaffold-v2-p3.md, FR-13); the only remaining path for
// a legacy manifest is a clean, zero-write refusal until Phase 4 ships the
// automated legacy → v2 migration.
const legacyLayoutFailClosedMsg = "this project uses the legacy scaffold layout; the automated migration to the overlay (v2) layout arrives in a later ralph release (Phase 4); no files were changed"

// errLegacyLayoutFailClosed is the sentinel wrapped by every legacy-manifest
// refusal, so callers (and tests) can match on it with errors.Is instead of
// string-matching legacyLayoutFailClosedMsg.
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
//     (runUpgradeV2, internal/cli/upgrade_v2.go) — the sole upgrade engine
//     left after Phase 3 removed the legacy interactive conflict-resolution
//     engine and the baseline mechanism it depended on.
//   - anything else (legacy manifest, or a manifest with no layout at all):
//     fail closed with legacyLayoutFailClosedMsg. No template diffing, no
//     writes — see errLegacyLayoutFailClosed's doc comment for why.
//
// `in` is accepted for signature stability across the package's existing
// call sites even though the v2 engine never reads from it (AC-2: no
// prompts, no stdin reads, on any branch).
func runUpgradeIOWithOptions(targetDir string, opts upgradeOptions, in io.Reader, out, errOut io.Writer, colorize bool) error {
	_ = in
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
		return errLegacyLayoutFailClosed
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
