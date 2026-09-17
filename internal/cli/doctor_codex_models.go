package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
)

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

	data, err := os.ReadFile(cachePath)
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

	if len(missing) > 0 {
		r.Status = "warn"
		r.Detail = fmt.Sprintf("%d codex model_pool slug(s) not found in %s: %s",
			len(missing), cachePath, strings.Join(missing, ", "))
		return r
	}

	r.Status = "pass"
	r.Detail = fmt.Sprintf("%d codex model_pool slug(s) present in %s", len(codexSlugs), cachePath)
	return r
}
