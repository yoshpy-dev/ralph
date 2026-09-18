package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// codexCacheStaleAfter is the age past which the doctor slug check appends a
// stale note to its Detail: a cache this old may not reflect codex's current
// model list, so a warn against it is weak evidence that a slug was retired
// (and a pass is weak evidence that it still exists). Status never changes on
// staleness -- the check stays best-effort.
const codexCacheStaleAfter = 24 * time.Hour

// codexCacheBetweenReadAndStat is a test seam called by readCodexModelsCache
// after the cache bytes are read and immediately before the second Stat --
// only on the path where that second Stat actually follows (a failed first
// Stat returns before the seam). Tests set it to rewrite the cache in place
// so the "changed while reading" path is deterministic; production leaves it
// nil.
var codexCacheBetweenReadAndStat func()

// codexModelsCachePath resolves the path codex itself uses for its local
// model-slug cache: $CODEX_HOME/models_cache.json when CODEX_HOME is set, else
// $HOME/.codex/models_cache.json. It mirrors codex's own resolver exactly
// (codex-rs/utils/home-dir/src/lib.rs, find_codex_home -- identical at
// rust-v0.149.1 and rust-v0.154.0, checked 2026-09-17): CODEX_HOME is filtered
// only on emptiness (`!val.is_empty()`) and then used literally -- never
// trimmed or otherwise normalized -- so this check reads the same directory
// codex itself would. Kept as its own function (rather than inlined into
// checkCodexModelSlugs) so tests can pin it via t.Setenv without touching any
// other doctor check. Returns an error when CODEX_HOME is empty and the home
// directory cannot be resolved; the caller reports that as an info result
// rather than failing the check.
func codexModelsCachePath() (string, error) {
	if home := os.Getenv("CODEX_HOME"); home != "" {
		return filepath.Join(home, "models_cache.json"), nil
	}
	// os.UserHomeDir reads $HOME on unix (the same env var the plan's
	// fallback names), while also handling Windows correctly.
	uh, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(uh, ".codex", "models_cache.json"), nil
}

// readCodexModelsCache reads path through a single open handle and reports
// the file's mtime as observed consistently around the read: Stat before,
// read, Stat after. If the two Stats disagree in ModTime or Size (codex
// rewrote the cache in place while we were reading), changed is true and
// mtime is zero -- the caller must not present the bytes and any timestamp
// as belonging to the same version. If either Stat fails, mtime is zero and
// changed is false: the caller omits freshness and keeps the slug verdict.
// A rename-based refresh keeps this handle on the old inode, so contents and
// mtime stay consistent (both old) in that case.
func readCodexModelsCache(path string) (data []byte, mtime time.Time, changed bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, false, err
	}
	defer func() { _ = f.Close() }()
	before, statErr := f.Stat()
	data, err = io.ReadAll(f)
	if err != nil {
		return nil, time.Time{}, false, err
	}
	if statErr != nil {
		return data, time.Time{}, false, nil
	}
	if codexCacheBetweenReadAndStat != nil {
		codexCacheBetweenReadAndStat()
	}
	after, statErr := f.Stat()
	if statErr != nil {
		return data, time.Time{}, false, nil
	}
	if !before.ModTime().Equal(after.ModTime()) || before.Size() != after.Size() {
		return data, time.Time{}, true, nil
	}
	return data, after.ModTime(), false, nil
}

// formatCacheAge renders an age at the granularity an operator needs to tell
// "minutes ago" from "days ago": Nm below an hour, Nh below a day, Nd
// otherwise. Negative ages (clock skew, a future mtime) render as 0m.
func formatCacheAge(d time.Duration) string {
	// hoursPerDay is the display-granularity switch from Nh to Nd. It is
	// unrelated to codexCacheStaleAfter (the staleness threshold), which
	// happens to be 24h today but may change independently.
	const hoursPerDay = 24
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < hoursPerDay*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/hoursPerDay))
	}
}

// codexCacheFreshnessClause is the suffix appended to the slug check's warn
// and pass Detail. changed wins over mtime; a zero mtime with changed=false
// (Stat failed) yields "" so the Detail reads exactly as before this clause
// existed.
func codexCacheFreshnessClause(mtime time.Time, changed bool, now time.Time) string {
	if changed {
		return " (cache changed while reading; freshness unknown — re-run)"
	}
	if mtime.IsZero() {
		return ""
	}
	age := now.Sub(mtime)
	clause := fmt.Sprintf(" (cache written %s, %s ago)", mtime.UTC().Format(time.RFC3339), formatCacheAge(age))
	if age > codexCacheStaleAfter {
		clause += "; cache may be stale — launch codex once to refresh, then re-run"
	}
	return clause
}

// codexModelsCacheDoc is a minimal decode of codex's models_cache.json
// (verified shape on codex-cli 0.149.1): a top-level object with a "models"
// array of entries carrying at least a "slug". Every other field
// (display_name, visibility, ...) is ignored -- a listed slug counts as
// present regardless of its visibility value (e.g. "hide" still means the
// model is known to codex, just not offered in interactive pickers).
type codexModelsCacheDoc struct {
	Models []struct {
		Slug string `json:"slug"`
	} `json:"models"`
}

// checkCodexModelSlugs is `ralph doctor`'s "Org codex model slugs" check
// (AC-4): it validates every codex entry in [org].model_pool against
// codex's local models_cache.json without launching any process (contrast
// checkOrgModelProbes/--probe-models, which does spawn `codex exec`). This
// check always runs -- it is cheap (a single local file read) and does not
// require codex to be on PATH.
//
// Six deterministic outcomes:
//   - no codex entries in model_pool at all -> info, no cache lookup performed
//   - CODEX_HOME empty and the home directory unresolvable -> info, names
//     the resolution error (there is no cache path to look up)
//   - cache file missing -> info, names the path it looked for (codex may
//     simply not have been run locally yet -- not a project misconfiguration)
//   - cache file present but not valid JSON -> info, names the parse error
//     (the cache is best-effort data owned by codex, not ralph, so a
//     malformed cache is not itself a ralph-level warning)
//   - every pool codex slug found in the cache -> pass, listing the count
//   - one or more pool codex slugs absent from the cache -> warn, naming
//     each missing slug (the model_pool entry may be stale/retired)
//
// The warn and pass Details end with a freshness clause: normally
// "(cache written <UTC RFC3339>, <age> ago)", plus "; cache may be stale —
// launch codex once to refresh, then re-run" once that age exceeds
// codexCacheStaleAfter. If the open handle's Stat before and after the read
// disagree in ModTime or Size (an in-place rewrite that landed while this
// check was reading; a same-size rewrite within one mtime tick is not
// detected, which is acceptable for a best-effort check), the clause is
// instead "(cache changed while reading; freshness unknown — re-run)" and
// neither the mtime nor the stale note is shown (the slug verdict itself is
// still based on the bytes that were read). If Stat fails, the clause is
// omitted entirely and the Detail reads exactly as it did before this
// freshness clause existed.
func checkCodexModelSlugs(cfg config.Config) checkResult {
	r := checkResult{Name: "Org codex model slugs"}

	seen := make(map[string]bool)
	var codexSlugs []string
	for _, entry := range cfg.Org.ModelPool {
		if entry.Driver != "codex" || seen[entry.Model] {
			continue
		}
		seen[entry.Model] = true
		codexSlugs = append(codexSlugs, entry.Model)
	}
	if len(codexSlugs) == 0 {
		r.Status = "info"
		r.Detail = "no codex entries in model_pool"
		return r
	}
	sort.Strings(codexSlugs)

	cachePath, err := codexModelsCachePath()
	if err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("could not resolve codex models cache path: %v — skipping", err)
		return r
	}

	data, mtime, changed, err := readCodexModelsCache(cachePath)
	if err != nil {
		r.Status = "info"
		if errors.Is(err, fs.ErrNotExist) {
			r.Detail = fmt.Sprintf("codex models cache not found at %s — skipping", cachePath)
		} else {
			r.Detail = fmt.Sprintf("could not read codex models cache at %s: %v — skipping", cachePath, err)
		}
		return r
	}

	var doc codexModelsCacheDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("codex models cache at %s is not valid JSON: %v — skipping", cachePath, err)
		return r
	}

	cached := make(map[string]bool, len(doc.Models))
	for _, m := range doc.Models {
		cached[m.Slug] = true
	}

	var missing []string
	for _, slug := range codexSlugs {
		if !cached[slug] {
			missing = append(missing, slug)
		}
	}

	freshness := codexCacheFreshnessClause(mtime, changed, time.Now())

	if len(missing) > 0 {
		r.Status = "warn"
		r.Detail = fmt.Sprintf("%d codex model_pool slug(s) not found in %s: %s",
			len(missing), cachePath, strings.Join(missing, ", ")) + freshness
		return r
	}

	r.Status = "pass"
	r.Detail = fmt.Sprintf("%d codex model_pool slug(s) present in %s", len(codexSlugs), cachePath) + freshness
	return r
}
