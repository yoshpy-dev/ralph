package toolchain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/mod/semver"
)

// The versions a repository asks for (plan 2026-11-12-gate-tool-versions,
// D6): an npm version range such as package.json's engines.node, a
// .node-version or .nvmrc spec, and package.json's packageManager.

// maxRangeBytes caps the range ParseRange reads.
const maxRangeBytes = 1024

// maxVersionDigits caps one number of a version, so it fits a uint64 with
// room to add one.
const maxVersionDigits = 15

// Range is an npm version range, read by ParseRange.
type Range struct {
	// sets are the ranges joined by "||"; a version is in the range when it
	// satisfies every comparator of one set. An empty set allows every
	// version that is not a prerelease.
	sets [][]comparator
}

// comparator is one primitive comparison with a full semver, such as
// ">=v1.2.3" or "<v2.0.0-0".
type comparator struct {
	op  string // ">", ">=", "<", "<=", or "="
	ver string // a valid golang.org/x/mod/semver version with three numbers
}

// ParseRange reads s as an npm version range, the subset of node-semver's
// grammar these forms make up: an exact version ("24.15.0", "=24.15.0",
// "v24.15.0", with a prerelease or build), a partial or wildcard version
// ("24", "24.15", "24.x", "24.15.X", "*", "x", ""), a comparator (">",
// ">=", "<", "<=", "=" before a full or partial version, a space after the
// operator allowed), a caret ("^24.15.0", with npm's rules for a 0 major and
// minor) or tilde ("~24.15", "~>24.15") range, a hyphen range ("22 - 24.15",
// partial upper bounds included), comparators separated by spaces (all must
// hold), and ranges joined by "||" (one must hold). Partial versions widen as
// npm's do: "<=24.15" is "<24.16.0-0". Anything else, such as a number after
// a wildcard, a leading zero, a lone "|", or text that is not a version, is
// an error; a caller treats it as a range it cannot evaluate, not as a
// mismatch.
func ParseRange(s string) (Range, error) {
	if len(s) > maxRangeBytes {
		return Range{}, fmt.Errorf("npm range is longer than %d bytes", maxRangeBytes)
	}
	if !utf8.ValidString(s) {
		return Range{}, errors.New("npm range is not UTF-8")
	}
	var r Range
	for _, part := range strings.Split(s, "||") {
		set, err := parseComparatorSet(part)
		if err != nil {
			return Range{}, fmt.Errorf("npm range %q: %w", s, err)
		}
		r.sets = append(r.sets, set)
	}
	return r, nil
}

// Allows reports whether version, a full semver with an optional "v" or "="
// before it ("v24.22.1", "10.10.0"), is in the range, by npm's rules: a
// prerelease version is in a set only when a comparator of that set has a
// prerelease of the same major.minor.patch. The error is non-nil when
// version is not a full semver.
func (r Range) Allows(version string) (bool, error) {
	v, err := fullVersion(version)
	if err != nil {
		return false, err
	}
	for _, set := range r.sets {
		if setAllows(set, v) {
			return true, nil
		}
	}
	return false, nil
}

// setAllows reports whether v satisfies every comparator of set and, when v
// is a prerelease, set has a prerelease comparator of v's major.minor.patch.
func setAllows(set []comparator, v string) bool {
	for _, c := range set {
		if !c.allows(v) {
			return false
		}
	}
	if semver.Prerelease(v) == "" {
		return true
	}
	for _, c := range set {
		if semver.Prerelease(c.ver) != "" && versionCore(c.ver) == versionCore(v) {
			return true
		}
	}
	return false
}

func (c comparator) allows(v string) bool {
	cmp := semver.Compare(v, c.ver)
	switch c.op {
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	default:
		return cmp == 0
	}
}

// versionCore returns the "vMAJOR.MINOR.PATCH" of the valid semver v.
func versionCore(v string) string {
	c := semver.Canonical(v)
	return strings.TrimSuffix(c, semver.Prerelease(c))
}

// parseComparatorSet reads one range between "||"s: empty (any version), a
// hyphen range, or simple comparators separated by spaces.
func parseComparatorSet(part string) ([]comparator, error) {
	fields := strings.Fields(part)
	if len(fields) == 0 {
		return nil, nil
	}
	if len(fields) == 3 && fields[1] == "-" {
		return hyphenRange(fields[0], fields[2])
	}
	var set []comparator
	for i := 0; i < len(fields); i++ {
		tok := fields[i]
		if isOperator(tok) {
			if i+1 == len(fields) {
				return nil, fmt.Errorf("%q has no version after it", tok)
			}
			i++
			tok += fields[i]
		}
		cs, err := parseSimple(tok)
		if err != nil {
			return nil, err
		}
		set = append(set, cs...)
	}
	return set, nil
}

// operators are the prefixes of a simple comparator, longest first.
var operators = []string{">=", "<=", "~>", ">", "<", "=", "~", "^"}

func isOperator(tok string) bool {
	for _, op := range operators {
		if tok == op {
			return true
		}
	}
	return false
}

// parseSimple reads one comparator, caret, tilde, or (partial) version.
func parseSimple(tok string) ([]comparator, error) {
	op := ""
	for _, o := range operators {
		if strings.HasPrefix(tok, o) {
			op = o
			break
		}
	}
	p, err := parsePartial(tok[len(op):])
	if err != nil {
		return nil, err
	}
	switch op {
	case "", "=":
		return xRange(p), nil
	case "~", "~>":
		return tildeRange(p), nil
	case "^":
		return caretRange(p), nil
	default:
		return primitiveRange(op, p), nil
	}
}

// partial is a version with zero to three numbers before the first
// wildcard; pre is the prerelease, only with three numbers.
type partial struct {
	nums []uint64
	pre  string
}

// parsePartial reads a version that may stop early or use x, X, or * for
// its remaining numbers, with an optional "v" before it.
func parsePartial(s string) (partial, error) {
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return partial{}, errors.New("a version is missing")
	}
	core, build, hasBuild := strings.Cut(s, "+")
	core, pre, hasPre := strings.Cut(core, "-")
	parts := strings.Split(core, ".")
	if len(parts) > 3 {
		return partial{}, fmt.Errorf("%q has more than three numbers", s)
	}
	var p partial
	wild := false
	for _, part := range parts {
		switch {
		case part == "x" || part == "X" || part == "*":
			wild = true
		case wild:
			return partial{}, fmt.Errorf("%q has a number after a wildcard", s)
		default:
			n, err := versionNumber(part)
			if err != nil {
				return partial{}, fmt.Errorf("%q: %w", s, err)
			}
			p.nums = append(p.nums, n)
		}
	}
	if hasPre || hasBuild {
		if len(p.nums) != 3 {
			return partial{}, fmt.Errorf("%q has a prerelease or build without three numbers", s)
		}
		full := "v" + core
		if hasPre {
			full += "-" + pre
		}
		if hasBuild {
			full += "+" + build
		}
		if !semver.IsValid(full) {
			return partial{}, fmt.Errorf("%q is not a valid version", s)
		}
		if hasPre {
			p.pre = pre
		}
	}
	return p, nil
}

// versionNumber reads one number of a version: "0", or digits with no
// leading zero, at most maxVersionDigits of them.
func versionNumber(s string) (uint64, error) {
	if s == "" || len(s) > maxVersionDigits || (len(s) > 1 && s[0] == '0') {
		return 0, fmt.Errorf("%q is not a version number", s)
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("%q is not a version number", s)
		}
	}
	return strconv.ParseUint(s, 10, 64)
}

// ver returns "vM.m.p" with "-pre" when pre is not "".
func ver(m, n, p uint64, pre string) string {
	v := "v" + strconv.FormatUint(m, 10) + "." + strconv.FormatUint(n, 10) + "." + strconv.FormatUint(p, 10)
	if pre != "" {
		v += "-" + pre
	}
	return v
}

// noVersion is a set no version satisfies.
var noVersion = []comparator{{"<", "v0.0.0-0"}}

// lowerBound returns the comparators ">=" the start of p (none for a bare
// wildcard).
func lowerBound(p partial) []comparator {
	switch len(p.nums) {
	case 0:
		return nil
	case 1:
		return []comparator{{">=", ver(p.nums[0], 0, 0, "")}}
	case 2:
		return []comparator{{">=", ver(p.nums[0], p.nums[1], 0, "")}}
	default:
		return []comparator{{">=", ver(p.nums[0], p.nums[1], p.nums[2], p.pre)}}
	}
}

// xRange is "1" (>=1.0.0 <2.0.0-0), "1.2" (>=1.2.0 <1.3.0-0), "1.2.3"
// (=1.2.3), or a wildcard (any version).
func xRange(p partial) []comparator {
	switch len(p.nums) {
	case 0:
		return nil
	case 1:
		return []comparator{{">=", ver(p.nums[0], 0, 0, "")}, {"<", ver(p.nums[0]+1, 0, 0, "0")}}
	case 2:
		return []comparator{{">=", ver(p.nums[0], p.nums[1], 0, "")}, {"<", ver(p.nums[0], p.nums[1]+1, 0, "0")}}
	default:
		return []comparator{{"=", ver(p.nums[0], p.nums[1], p.nums[2], p.pre)}}
	}
}

// primitiveRange is op before p, with npm's widening of a partial version.
func primitiveRange(op string, p partial) []comparator {
	n := p.nums
	switch len(n) {
	case 0:
		if op == ">" || op == "<" {
			return noVersion
		}
		return nil
	case 3:
		return []comparator{{op, ver(n[0], n[1], n[2], p.pre)}}
	}
	switch op {
	case ">":
		if len(n) == 1 {
			return []comparator{{">=", ver(n[0]+1, 0, 0, "")}}
		}
		return []comparator{{">=", ver(n[0], n[1]+1, 0, "")}}
	case ">=":
		return lowerBound(p)
	case "<":
		if len(n) == 1 {
			return []comparator{{"<", ver(n[0], 0, 0, "0")}}
		}
		return []comparator{{"<", ver(n[0], n[1], 0, "0")}}
	default: // "<="
		if len(n) == 1 {
			return []comparator{{"<", ver(n[0]+1, 0, 0, "0")}}
		}
		return []comparator{{"<", ver(n[0], n[1]+1, 0, "0")}}
	}
}

// tildeRange allows patch changes when a minor is given, and minor changes
// when not: "~1.2.3" is >=1.2.3 <1.3.0-0, "~1" is >=1.0.0 <2.0.0-0.
func tildeRange(p partial) []comparator {
	n := p.nums
	switch len(n) {
	case 0:
		return nil
	case 1:
		return xRange(p)
	default:
		return append(lowerBound(p), comparator{"<", ver(n[0], n[1]+1, 0, "0")})
	}
}

// caretRange allows changes that keep the first non-zero number: "^1.2.3" is
// >=1.2.3 <2.0.0-0, "^0.2.3" is >=0.2.3 <0.3.0-0, "^0.0.3" is >=0.0.3
// <0.0.4-0, "^0.0" is >=0.0.0 <0.1.0-0.
func caretRange(p partial) []comparator {
	n := p.nums
	var upper string
	switch {
	case len(n) == 0:
		return nil
	case n[0] != 0 || len(n) == 1:
		upper = ver(n[0]+1, 0, 0, "0")
	case len(n) == 2 || n[1] != 0:
		upper = ver(0, n[1]+1, 0, "0")
	default:
		upper = ver(0, 0, n[2]+1, "0")
	}
	return append(lowerBound(p), comparator{"<", upper})
}

// hyphenRange is "a - b": >= the start of a, and <= b when b is full or
// below the next version b leaves open when it is partial ("1 - 2.3" is
// >=1.0.0 <2.4.0-0).
func hyphenRange(a, b string) ([]comparator, error) {
	lo, err := parsePartial(a)
	if err != nil {
		return nil, err
	}
	hi, err := parsePartial(b)
	if err != nil {
		return nil, err
	}
	set := lowerBound(lo)
	n := hi.nums
	switch len(n) {
	case 1:
		set = append(set, comparator{"<", ver(n[0]+1, 0, 0, "0")})
	case 2:
		set = append(set, comparator{"<", ver(n[0], n[1]+1, 0, "0")})
	case 3:
		set = append(set, comparator{"<=", ver(n[0], n[1], n[2], hi.pre)})
	}
	return set, nil
}

// fullVersion returns version as a golang.org/x/mod/semver version, "v"
// first, when it is a full semver (three numbers, an optional prerelease
// and build) with an optional "=" and "v" before it.
func fullVersion(version string) (string, error) {
	s := strings.TrimPrefix(strings.TrimSpace(version), "=")
	s = strings.TrimPrefix(s, "v")
	core, _, _ := strings.Cut(s, "+")
	core, _, _ = strings.Cut(core, "-")
	if strings.Count(core, ".") != 2 || !semver.IsValid("v"+s) {
		return "", fmt.Errorf("%q is not a full version", version)
	}
	return "v" + s, nil
}

// SpecResult is how a version-file spec compares with a version.
type SpecResult int

const (
	// SpecUnknown is the result that comes with an error.
	SpecUnknown SpecResult = iota
	// SpecMatch: the version is the spec's.
	SpecMatch
	// SpecMismatch: the version is not the spec's.
	SpecMismatch
	// SpecAlias: the spec names an alias (lts/*, lts/<name>, node, stable,
	// latest) that is not compared.
	SpecAlias
)

// MatchVersionSpec compares spec, the version a .node-version or .nvmrc
// asks for (see VersionFileSpec), with version, a full version such as
// node's "v24.22.1". A spec of one, two, or three numbers, with or without
// "v" ("24", "v24.15", "24.15.0"), matches a version that starts with those
// numbers; a three-number spec with a prerelease must match it exactly. An
// alias (lts/*, lts/<name>, node, stable, latest) is SpecAlias. The error is
// non-nil, with SpecUnknown, when spec is none of these or version is not a
// full version.
func MatchVersionSpec(spec, version string) (SpecResult, error) {
	s := strings.TrimSpace(spec)
	switch lower := strings.ToLower(s); {
	case lower == "node" || lower == "stable" || lower == "latest":
		return SpecAlias, nil
	case strings.HasPrefix(lower, "lts/") && len(lower) > len("lts/"):
		return SpecAlias, nil
	}
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	core, pre, hasPre := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if s == "" || len(parts) > 3 || (hasPre && len(parts) != 3) {
		return SpecUnknown, fmt.Errorf("version spec %q is not a version or a known alias", spec)
	}
	want := make([]uint64, 0, len(parts))
	for _, part := range parts {
		n, err := versionNumber(part)
		if err != nil {
			return SpecUnknown, fmt.Errorf("version spec %q is not a version or a known alias", spec)
		}
		want = append(want, n)
	}
	if hasPre && !semver.IsValid("v"+s) {
		return SpecUnknown, fmt.Errorf("version spec %q is not a valid version", spec)
	}
	v, err := fullVersion(version)
	if err != nil {
		return SpecUnknown, err
	}
	got, vpre, err := versionNumbers(v)
	if err != nil {
		return SpecUnknown, err
	}
	for i, n := range want {
		if got[i] != n {
			return SpecMismatch, nil
		}
	}
	if len(want) == 3 && pre != vpre {
		return SpecMismatch, nil
	}
	return SpecMatch, nil
}

// versionNumbers returns the three numbers and the prerelease of v, a
// version fullVersion returned.
func versionNumbers(v string) ([3]uint64, string, error) {
	var nums [3]uint64
	core := strings.TrimPrefix(versionCore(v), "v")
	for i, part := range strings.Split(core, ".") {
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil || i > 2 {
			return nums, "", fmt.Errorf("%q is not a full version", v)
		}
		nums[i] = n
	}
	return nums, strings.TrimPrefix(semver.Prerelease(v), "-"), nil
}

// VersionFileSpec returns the spec in a .node-version or .nvmrc file's
// content: its first line that is not blank once a "#" comment is cut off,
// trimmed of surrounding spaces; "" when there is none.
func VersionFileSpec(content []byte) string {
	for _, line := range strings.Split(string(content), "\n") {
		line, _, _ = strings.Cut(line, "#")
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// ParsePackageManager reads package.json's packageManager field, as corepack
// writes it: "<name>@<version>" with an optional "+<hash>" after the version
// ("pnpm@10.10.0+sha512.<hash>" is "pnpm" and "10.10.0"). The error is
// non-nil when there is no "@", the name is empty or holds anything but
// lowercase letters, digits, ".", "_", and "-", or the version is not a full
// semver (a URL, say).
func ParsePackageManager(s string) (name, version string, err error) {
	t := strings.TrimSpace(s)
	name, rest, ok := strings.Cut(t, "@")
	if !ok || name == "" || !packageNameOK(name) {
		return "", "", fmt.Errorf("packageManager %q is not <name>@<version>", s)
	}
	version, _, _ = strings.Cut(rest, "+")
	if _, err := fullVersion(version); err != nil || strings.HasPrefix(version, "v") || strings.HasPrefix(version, "=") {
		return "", "", fmt.Errorf("packageManager %q has no full version", s)
	}
	return name, version, nil
}

// packageNameOK reports whether name holds only lowercase letters, digits,
// ".", "_", and "-".
func packageNameOK(name string) bool {
	return strings.IndexFunc(name, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return false
		}
		return true
	}) < 0
}
