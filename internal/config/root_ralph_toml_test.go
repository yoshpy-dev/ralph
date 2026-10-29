package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

// TestRootRalphToml_KnownKeysAndReviewSwitchesOn reads this repository's own
// ralph.toml strictly: every key must name a Config field, and the four
// review switches must be on. Load cannot check this, since it ignores keys
// with no field (TestLoad_IgnoresRetiredOrgBudgetTable), so a misspelled
// switch would load as the default, off, without an error.
//
// The file is decoded into a zero Config, not Default(), so each switch is
// true only when the file itself sets it. A missing file fails the test:
// the file is tracked, and without it the switches fall back to off.
func TestRootRalphToml_KnownKeysAndReviewSwitchesOn(t *testing.T) {
	path := filepath.Join(repoRoot(t), "ralph.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the repository's ralph.toml: %v", err)
	}

	var cfg Config
	err = toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(&cfg)
	var strict *toml.StrictMissingError
	if errors.As(err, &strict) {
		var keys []string
		for _, e := range strict.Errors {
			line, _ := e.Position()
			keys = append(keys, fmt.Sprintf("%s (line %d)", strings.Join(e.Key(), "."), line))
		}
		t.Fatalf("%s has keys no Config field reads: %s", path, strings.Join(keys, ", "))
	}
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	switches := []struct {
		key string
		on  bool
	}{
		{"standard.plan_advisory", cfg.Standard.PlanAdvisory},
		{"standard.cross_review", cfg.Standard.CrossReview},
		{"org.review.plan_advisory", cfg.Org.Review.PlanAdvisory},
		{"org.review.cross_review", cfg.Org.Review.CrossReview},
	}
	for _, s := range switches {
		if !s.on {
			t.Errorf("%s: %s is not true", path, s.key)
		}
	}
}
