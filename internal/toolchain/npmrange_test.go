package toolchain

import (
	"strings"
	"testing"
)

func TestRangeAllows_NpmRangeForms(t *testing.T) {
	tests := []struct {
		form, rng, version string
		want               bool
	}{
		// exact versions
		{"exact", "24.15.0", "v24.15.0", true},
		{"exact", "24.15.0", "24.15.1", false},
		{"exact with =", "=24.15.0", "24.15.0", true},
		{"exact with v", "v24.15.0", "24.15.0", true},
		{"exact with = and v", "=v24.15.0", "v24.15.0", true},
		{"exact ignores build", "1.2.3+build.1", "1.2.3", true},
		{"version build ignored", "1.2.3", "1.2.3+abc", true},
		// partial versions
		{"partial major", "24", "v24.0.0", true},
		{"partial major", "24", "v24.22.1", true},
		{"partial major", "24", "v25.0.0", false},
		{"partial major", "24", "v23.9.9", false},
		{"partial major excludes next prerelease", "24", "v25.0.0-rc.1", false},
		{"partial major with v", "v24", "24.1.0", true},
		{"partial minor", "24.15", "24.15.9", true},
		{"partial minor", "24.15", "24.16.0", false},
		// wildcards
		{"x minor", "24.x", "24.3.0", true},
		{"x minor", "24.x", "25.0.0", false},
		{"X patch", "24.15.X", "24.15.3", true},
		{"X patch", "24.15.X", "24.16.0", false},
		{"* any", "*", "1.2.3", true},
		{"x any", "x", "0.0.0", true},
		{"x.x.x any", "x.x.x", "99.0.0", true},
		{"empty any", "", "24.0.0", true},
		{"* excludes prereleases", "*", "1.0.0-beta", false},
		// comparators
		{">= major", ">=20", "20.0.0", true},
		{">= major", ">=20", "19.9.9", false},
		{"> major widens", ">20", "20.9.9", false},
		{"> major widens", ">20", "21.0.0", true},
		{"> minor widens", ">20.1", "20.1.9", false},
		{"> minor widens", ">20.1", "20.2.0", true},
		{"> full", ">20.1.2", "20.1.3", true},
		{"> full", ">20.1.2", "20.1.2", false},
		{"< major", "<24", "23.9.9", true},
		{"< major", "<24", "24.0.0", false},
		{"< major excludes its prerelease", "<24", "24.0.0-rc.1", false},
		{"<= major widens", "<=24", "24.9.9", true},
		{"<= major widens", "<=24", "25.0.0", false},
		{"<= minor widens", "<=24.1", "24.1.9", true},
		{"<= minor widens", "<=24.1", "24.2.0", false},
		{"<= full", "<=24.1.2", "24.1.2", true},
		{"<= full", "<=24.1.2", "24.1.3", false},
		{"= partial", "=24", "24.5.0", true},
		{"space after operator", ">= 18", "18.0.0", true},
		{"space after operator", ">= 18", "17.9.0", false},
		{"v after operator", ">=v18.0.0", "18.0.0", true},
		{"> wildcard is nothing", ">*", "1.0.0", false},
		{"< wildcard is nothing", "<*", "0.0.0", false},
		{">= wildcard is any", ">=*", "1.0.0", true},
		{"<= wildcard is any", "<=x", "1.0.0", true},
		// caret
		{"caret", "^24.15.0", "24.99.0", true},
		{"caret", "^24.15.0", "24.14.9", false},
		{"caret", "^24.15.0", "25.0.0", false},
		{"caret 0.x.y keeps the minor", "^0.2.3", "0.2.9", true},
		{"caret 0.x.y keeps the minor", "^0.2.3", "0.3.0", false},
		{"caret 0.0.y keeps the patch", "^0.0.3", "0.0.3", true},
		{"caret 0.0.y keeps the patch", "^0.0.3", "0.0.4", false},
		{"caret 0.0", "^0.0", "0.0.9", true},
		{"caret 0.0", "^0.0", "0.1.0", false},
		{"caret 0.x", "^0.x", "0.9.0", true},
		{"caret 0.x", "^0.x", "1.0.0", false},
		{"caret 0.1", "^0.1", "0.1.5", true},
		{"caret 0.1", "^0.1", "0.2.0", false},
		{"caret 1.2.x", "^1.2.x", "1.9.0", true},
		{"caret 1.2.x", "^1.2.x", "1.1.0", false},
		{"caret with space", "^ 24", "24.1.0", true},
		// tilde
		{"tilde full", "~24.15.1", "24.15.9", true},
		{"tilde full", "~24.15.1", "24.16.0", false},
		{"tilde full", "~24.15.1", "24.15.0", false},
		{"tilde minor", "~24.15", "24.15.0", true},
		{"tilde minor", "~24.15", "24.16.0", false},
		{"tilde major", "~24", "24.99.0", true},
		{"tilde major", "~24", "25.0.0", false},
		{"tilde ~>", "~>24.15", "24.15.3", true},
		{"tilde with space", "~ 24.15", "24.15.3", true},
		// space-separated AND
		{"and", ">=20 <23", "22.9.9", true},
		{"and", ">=20 <23", "23.0.0", false},
		{"and", ">=20 <23", "19.0.0", false},
		{"and of three", ">=18.0.0 <19.0.0 >=18.5.0", "18.4.0", false},
		{"and of three", ">=18.0.0 <19.0.0 >=18.5.0", "18.5.0", true},
		// ||
		{"or", "^20 || ^22 || >=24", "22.1.0", true},
		{"or", "^20 || ^22 || >=24", "21.0.0", false},
		{"or", "^20 || ^22 || >=24", "24.22.1", true},
		{"or without spaces", "18||22", "22.0.0", true},
		{"or with an empty side", "18 ||", "30.0.0", true},
		// hyphen
		{"hyphen partial", "20 - 22", "22.9.9", true},
		{"hyphen partial", "20 - 22", "23.0.0", false},
		{"hyphen partial", "20 - 22", "19.9.9", false},
		{"hyphen partial upper", "20.1.2 - 22.3", "22.3.9", true},
		{"hyphen partial upper", "20.1.2 - 22.3", "22.4.0", false},
		{"hyphen partial upper", "20.1.2 - 22.3", "20.1.1", false},
		{"hyphen full", "20.1.2 - 22.3.4", "22.3.4", true},
		{"hyphen full", "20.1.2 - 22.3.4", "22.3.5", false},
		{"hyphen open lower", "* - 22", "1.0.0", true},
		{"hyphen open upper", "20 - x", "99.0.0", true},
		{"hyphen open upper", "20 - x", "19.0.0", false},
		// prereleases
		{"prerelease same tuple", ">=24.0.0-rc.1", "24.0.0-rc.2", true},
		{"prerelease release", ">=24.0.0-rc.1", "24.0.0", true},
		{"prerelease other tuple", ">=24.0.0-rc.1", "25.0.0-rc.1", false},
		{"caret prerelease below", "^24.0.0-rc.1", "24.0.0-rc.0", false},
		{"caret prerelease", "^24.0.0-rc.1", "24.0.0-rc.1", true},
		{"exact prerelease", "24.0.0-rc.1", "24.0.0-rc.1", true},
	}
	for _, tt := range tests {
		r, err := ParseRange(tt.rng)
		if err != nil {
			t.Errorf("%s: ParseRange(%q): %v", tt.form, tt.rng, err)
			continue
		}
		got, err := r.Allows(tt.version)
		if err != nil {
			t.Errorf("%s: ParseRange(%q).Allows(%q): %v", tt.form, tt.rng, tt.version, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: ParseRange(%q).Allows(%q) = %v, want %v", tt.form, tt.rng, tt.version, got, tt.want)
		}
	}
}

func TestParseRange_RefusesFormsOutsideTheSubset(t *testing.T) {
	for _, rng := range []string{
		"24.x.1",     // a number after a wildcard
		"024",        // a leading zero
		"1.02.3",     // a leading zero
		"abc",        // not a version
		"lts/*",      // an nvm alias, not a range
		"latest",     // a dist-tag
		"^",          // an operator alone
		">=",         // an operator alone
		"1.2.3.4",    // four numbers
		"1 | 2",      // a lone |
		"1.2-beta",   // a prerelease without three numbers
		"1.x-beta",   // a prerelease after a wildcard
		"*-beta",     // a prerelease of a wildcard
		"1.2.3-",     // an empty prerelease
		"1.2.3-01",   // a leading zero in a numeric prerelease
		"1 - 2 - 3",  // two hyphens
		"1.2.3 -",    // a hyphen with no upper bound
		">=1 - 2",    // an operator in a hyphen range
		"~1.2.3 - 2", // a tilde in a hyphen range
		"1..2",       // an empty number
		"1.",         // an empty number
		"1234567890123456.0.0",
		"\xff",
		strings.Repeat("1 ", 600),
	} {
		if _, err := ParseRange(rng); err == nil {
			t.Errorf("ParseRange(%q): no error, want one (a range it cannot evaluate)", rng)
		}
	}
}

func TestRangeAllows_RefusesAVersionThatIsNotFull(t *testing.T) {
	r, err := ParseRange(">=20")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"24", "24.1", "", "abc", "v24.1.2.3", "24.01.0"} {
		if _, err := r.Allows(v); err == nil {
			t.Errorf("Allows(%q): no error, want one", v)
		}
	}
}

func TestMatchVersionSpec_ComparesByNumericPrefix(t *testing.T) {
	tests := []struct {
		spec, version string
		want          SpecResult
	}{
		{"24", "v24.22.1", SpecMatch},
		{"v24", "v24.22.1", SpecMatch},
		{"24.22", "v24.22.1", SpecMatch},
		{"24.22.1", "v24.22.1", SpecMatch},
		{"v24.22.1", "24.22.1", SpecMatch},
		{" 24.22.1 ", "v24.22.1", SpecMatch},
		{"24", "v22.1.0", SpecMismatch},
		{"24.15", "v24.22.1", SpecMismatch},
		{"24.22.0", "v24.22.1", SpecMismatch},
		{"2", "v24.0.0", SpecMismatch},
		{"24.0.0-rc.1", "v24.0.0-rc.1", SpecMatch},
		{"24.0.0", "v24.0.0-rc.1", SpecMismatch},
		{"24.0.0-rc.1", "v24.0.0", SpecMismatch},
		{"lts/*", "v24.22.1", SpecAlias},
		{"lts/iron", "v24.22.1", SpecAlias},
		{"LTS/Iron", "v24.22.1", SpecAlias},
		{"node", "v24.22.1", SpecAlias},
		{"stable", "v24.22.1", SpecAlias},
		{"latest", "garbage", SpecAlias},
	}
	for _, tt := range tests {
		got, err := MatchVersionSpec(tt.spec, tt.version)
		if err != nil || got != tt.want {
			t.Errorf("MatchVersionSpec(%q, %q) = %v, %v; want %v, nil", tt.spec, tt.version, got, err, tt.want)
		}
	}
}

func TestMatchVersionSpec_RefusesWhatItCannotCompare(t *testing.T) {
	for _, tt := range []struct{ spec, version string }{
		{"iron", "v24.22.1"},
		{"", "v24.22.1"},
		{"lts/", "v24.22.1"},
		{"24.x", "v24.22.1"},
		{">=24", "v24.22.1"},
		{"24.1.2.3", "v24.22.1"},
		{"024", "v24.22.1"},
		{"24-rc.1", "v24.22.1"},
		{"24", "garbage"},
		{"24", "24"},
	} {
		got, err := MatchVersionSpec(tt.spec, tt.version)
		if err == nil || got != SpecUnknown {
			t.Errorf("MatchVersionSpec(%q, %q) = %v, %v; want SpecUnknown and an error", tt.spec, tt.version, got, err)
		}
	}
}

func TestVersionFileSpec_TakesTheFirstLineWithoutComments(t *testing.T) {
	tests := []struct{ content, want string }{
		{"v24.15.0\n", "v24.15.0"},
		{"# pinned for CI\n\n  24  # the LTS\n25\n", "24"},
		{"lts/iron\r\n", "lts/iron"},
		{"", ""},
		{"\n\n# only a comment\n", ""},
	}
	for _, tt := range tests {
		if got := VersionFileSpec([]byte(tt.content)); got != tt.want {
			t.Errorf("VersionFileSpec(%q) = %q, want %q", tt.content, got, tt.want)
		}
	}
}

func TestParsePackageManager_ReadsNameAndVersion(t *testing.T) {
	tests := []struct{ in, name, version string }{
		{"pnpm@10.10.0", "pnpm", "10.10.0"},
		{"pnpm@10.10.0+sha512.0123456789abcdef", "pnpm", "10.10.0"},
		{"yarn@4.5.0", "yarn", "4.5.0"},
		{" npm@10.9.2 ", "npm", "10.9.2"},
		{"yarn@4.0.0-rc.1", "yarn", "4.0.0-rc.1"},
	}
	for _, tt := range tests {
		name, version, err := ParsePackageManager(tt.in)
		if err != nil || name != tt.name || version != tt.version {
			t.Errorf("ParsePackageManager(%q) = %q, %q, %v; want %q, %q, nil", tt.in, name, version, err, tt.name, tt.version)
		}
	}
	for _, in := range []string{
		"pnpm", "pnpm@", "@10.0.0", "pnpm@10", "pnpm@latest",
		"pnpm@https://example.com/pnpm.tgz", "Pnpm@10.0.0", "pnpm@v10.0.0", "",
	} {
		if name, version, err := ParsePackageManager(in); err == nil {
			t.Errorf("ParsePackageManager(%q) = %q, %q, nil; want an error", in, name, version)
		}
	}
}
