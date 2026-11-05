package config

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

// The lock-step check of [org.role_models] (plan 2026-11-05-org-role-models,
// AC11): templates/base/ralph.toml shows both default sets of
// DefaultRoleModels as commented blocks, and roleModelsTemplateDrift compares
// each block with the set it shows. TestDefaultsLockStep in
// defaults_sync_test.go runs it as its "role_models" sub-test.
//
// templateComments cannot read the blocks: it tracks the table only from a
// real header, and both blocks sit under [org] with a commented header. So
// each block has a fixed comment line right above its commented header (its
// marker), and roleModelsTemplateDocs reads the block as the header and the
// run of `# <role> = <value>` lines right below it:
//
//	# The defaults when driver_pool includes "claude":
//	# [org.role_models]
//	# leader = { driver = "claude", model = "opus" }
//	...

// roleModelsTemplateHeader is the commented header line of each block.
const roleModelsTemplateHeader = "# [org.role_models]"

// roleModelsTemplateBlock is one of the commented blocks of [org.role_models]
// in the template.
type roleModelsTemplateBlock struct {
	label      string   // names the block in drift lines
	marker     string   // the comment line right above the block's header
	driverPool []string // the driver_pool whose DefaultRoleModels the block shows
}

// roleModelsTemplateBlocks lists the template's two blocks: the claude set
// first, then the codex set.
var roleModelsTemplateBlocks = []roleModelsTemplateBlock{
	{"the claude set", `# The defaults when driver_pool includes "claude":`, []string{"claude"}},
	{"the codex set", `# The defaults when driver_pool does not include "claude":`, []string{"codex"}},
}

// roleModelsTemplateCommentPrefix matches the "#" and the one space after it
// that a commented line of a block starts with; removing it gives the TOML
// line the block shows.
var roleModelsTemplateCommentPrefix = regexp.MustCompile(`^# ?`)

// roleModelsTemplateDocs returns, per block of roleModelsTemplateBlocks by its
// label, the TOML document the block's lines make with their comment prefix
// removed: the header and the run of commented statements
// (templateCommentStatement) right below it. problems names, as drift lines,
// a marker that is not on exactly one line, a marker whose next line is not
// roleModelsTemplateHeader, and a commented header with no marker above it,
// so a reworded marker or a third block fails instead of going unchecked.
func roleModelsTemplateDocs(template string) (docs map[string]string, problems []string) {
	lines := strings.Split(template, "\n")
	docs = map[string]string{}
	headers := map[int]bool{}
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == roleModelsTemplateHeader {
			headers[i] = true
		}
	}
	for _, block := range roleModelsTemplateBlocks {
		var at []int
		for i, line := range lines {
			if strings.TrimRight(line, " \t") == block.marker {
				at = append(at, i)
			}
		}
		if len(at) != 1 {
			problems = append(problems, fmt.Sprintf("org.role_models (%s): the template has the line %q %d times, want once, right above %q", block.label, block.marker, len(at), roleModelsTemplateHeader))
			continue
		}
		header := at[0] + 1
		if !headers[header] {
			problems = append(problems, fmt.Sprintf("org.role_models (%s): the line after %q is not %q", block.label, block.marker, roleModelsTemplateHeader))
			continue
		}
		delete(headers, header)
		doc := []string{roleModelsTemplateCommentPrefix.ReplaceAllString(lines[header], "")}
		for _, line := range lines[header+1:] {
			if !templateCommentStatement.MatchString(line) {
				break
			}
			doc = append(doc, roleModelsTemplateCommentPrefix.ReplaceAllString(line, ""))
		}
		docs[block.label] = strings.Join(doc, "\n") + "\n"
	}
	for _, i := range slices.Sorted(maps.Keys(headers)) {
		problems = append(problems, fmt.Sprintf("org.role_models: the template's line %d is %q with none of the block markers right above it", i+1, roleModelsTemplateHeader))
	}
	return docs, problems
}

// roleModelsTemplateDrift compares the template's [org.role_models] with
// defaults (DefaultRoleModels in the lock-step test). It returns one line per
// difference: the template sets [org.role_models] instead of only commenting
// it, a block is missing or malformed (roleModelsTemplateDocs), a block's
// entries fail the check Load runs on each file (checkRoleModelEntries), or a
// block's role holds another driver or model than defaults(driver_pool) has,
// is missing from the block, or is in the block only. None means the two
// agree.
func roleModelsTemplateDrift(templateData []byte, defaults func(driverPool []string) map[string]OrgRoleModel) ([]string, error) {
	var doc map[string]any
	if err := toml.Unmarshal(templateData, &doc); err != nil {
		return nil, err
	}
	var drift []string
	if tomlDocHasKey(doc, "org.role_models") {
		drift = append(drift, "org.role_models: set in the template, want it only as comments, so each role keeps the built-in default")
	}
	docs, problems := roleModelsTemplateDocs(string(templateData))
	drift = append(drift, problems...)
	for _, block := range roleModelsTemplateBlocks {
		text, ok := docs[block.label]
		if !ok {
			continue
		}
		if err := checkRoleModelEntries("templates/base/ralph.toml", []byte(text)); err != nil {
			drift = append(drift, fmt.Sprintf("org.role_models (%s): %v", block.label, err))
			continue
		}
		var parsed struct {
			Org struct {
				RoleModels map[string]OrgRoleModel `toml:"role_models"`
			} `toml:"org"`
		}
		if err := toml.Unmarshal([]byte(text), &parsed); err != nil {
			drift = append(drift, fmt.Sprintf("org.role_models (%s): the block does not read as TOML: %v", block.label, err))
			continue
		}
		got, want := parsed.Org.RoleModels, defaults(block.driverPool)
		roles := slices.Collect(maps.Keys(got))
		for role := range want {
			if _, ok := got[role]; !ok {
				roles = append(roles, role)
			}
		}
		slices.Sort(roles)
		for _, role := range roles {
			g, inTemplate := got[role]
			w, inDefault := want[role]
			switch {
			case !inTemplate:
				drift = append(drift, fmt.Sprintf("org.role_models.%s (%s): missing from the template (DefaultRoleModels = %+v)", role, block.label, w))
			case !inDefault:
				drift = append(drift, fmt.Sprintf("org.role_models.%s (%s): in the template, not in DefaultRoleModels (template = %+v)", role, block.label, g))
			case g != w:
				drift = append(drift, fmt.Sprintf("org.role_models.%s (%s): template = %+v, DefaultRoleModels = %+v", role, block.label, g, w))
			}
		}
	}
	return drift, nil
}

// roleModelsLockStepDrift is roleModelsTemplateDrift against
// DefaultRoleModels, in the form TestDefaultsLockStep's table takes. It does
// not read def: Default() holds only the claude set
// (TestDefault_RoleModelsIsTheClaudeSet), while the template shows both.
func roleModelsLockStepDrift(templateData []byte, _ Config) ([]string, error) {
	return roleModelsTemplateDrift(templateData, DefaultRoleModels)
}

// readTemplateForRoleModels returns templates/base/ralph.toml, skipping the
// test when the file is absent.
func readTemplateForRoleModels(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "templates", "base", "ralph.toml"))
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("templates/base/ralph.toml not found (%v)", err)
	}
	if err != nil {
		t.Fatalf("cannot read templates/base/ralph.toml: %v", err)
	}
	return string(data)
}

// editRoleModelsTemplateRole returns template with the commented line of role
// in block replaced by `# <role> = <value>`, or removed when value is "". It
// is an error unless the block's run of commented statements has the role on
// exactly one line.
func editRoleModelsTemplateRole(template string, block roleModelsTemplateBlock, role, value string) (string, error) {
	lines := strings.SplitAfter(template, "\n")
	start := slices.IndexFunc(lines, func(l string) bool { return strings.TrimRight(l, " \t\n") == block.marker })
	if start < 0 || start+1 >= len(lines) || strings.TrimRight(lines[start+1], " \t\n") != roleModelsTemplateHeader {
		return "", fmt.Errorf("no block after %q", block.marker)
	}
	found := 0
	for i := start + 2; i < len(lines); i++ {
		m := templateCommentStatement.FindStringSubmatch(strings.TrimSuffix(lines[i], "\n"))
		if m == nil {
			break
		}
		if m[1] != role {
			continue
		}
		found++
		if value == "" {
			lines[i] = ""
		} else {
			lines[i] = "# " + role + " = " + value + "\n"
		}
	}
	if found != 1 {
		return "", fmt.Errorf("the block after %q has %s on %d lines, want 1", block.marker, role, found)
	}
	return strings.Join(lines, ""), nil
}

// insertRoleModelsTemplateRole returns template with `# <role> = <value>`
// added as the first statement of block, right below its header.
func insertRoleModelsTemplateRole(template string, block roleModelsTemplateBlock, role, value string) (string, error) {
	old := block.marker + "\n" + roleModelsTemplateHeader + "\n"
	if n := strings.Count(template, old); n != 1 {
		return "", fmt.Errorf("the template has %q %d times, want once", old, n)
	}
	return strings.Replace(template, old, old+"# "+role+" = "+value+"\n", 1), nil
}

// changedDefaults returns DefaultRoleModels with change applied to the set of
// driverPool only.
func changedDefaults(driverPool []string, change func(map[string]OrgRoleModel)) func([]string) map[string]OrgRoleModel {
	return func(pool []string) map[string]OrgRoleModel {
		set := DefaultRoleModels(pool)
		if slices.Equal(pool, driverPool) {
			change(set)
		}
		return set
	}
}

// wantOneRoleModelsDrift fails the test unless drift is exactly one line
// starting with wantPrefix.
func wantOneRoleModelsDrift(t *testing.T, mutation string, drift []string, err error, wantPrefix string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: roleModelsTemplateDrift: %v", mutation, err)
	}
	if len(drift) != 1 || !strings.HasPrefix(drift[0], wantPrefix) {
		t.Errorf("%s: drift = %q, want one line starting with %q", mutation, drift, wantPrefix)
	}
}

// TestRoleModelsTemplateDrift_EachRoleIsComparedOnBothSides pins, for every
// role of both blocks, that each side reaches the comparison of that role and
// no other: changing the role's model in DefaultRoleModels, changing its line
// in the template, and removing its line from the template each give exactly
// one drift line naming the role and the block. The leader, implementer, and
// reviewer entries of a set are equal, so a swap of two of them would pass a
// single comparison of the real template. An extra role on either side is a
// drift too.
func TestRoleModelsTemplateDrift_EachRoleIsComparedOnBothSides(t *testing.T) {
	template := readTemplateForRoleModels(t)
	if drift, err := roleModelsTemplateDrift([]byte(template), DefaultRoleModels); err != nil || len(drift) != 0 {
		t.Fatalf("the real template must match DefaultRoleModels first (TestDefaultsLockStep): drift = %q, err = %v", drift, err)
	}

	for _, block := range roleModelsTemplateBlocks {
		for _, role := range slices.Sorted(maps.Keys(DefaultRoleModels(block.driverPool))) {
			t.Run(block.label+"/"+role, func(t *testing.T) {
				wantPrefix := "org.role_models." + role + " (" + block.label + "): "

				defaults := changedDefaults(block.driverPool, func(set map[string]OrgRoleModel) {
					entry := set[role]
					entry.Model += "-changed"
					set[role] = entry
				})
				drift, err := roleModelsTemplateDrift([]byte(template), defaults)
				wantOneRoleModelsDrift(t, "DefaultRoleModels changed", drift, err, wantPrefix+"template = ")

				other := "codex"
				if DefaultRoleModels(block.driverPool)[role].Driver == "codex" {
					other = "claude"
				}
				changed, err := editRoleModelsTemplateRole(template, block, role, `{ driver = "`+other+`", model = "`+DefaultRoleModels(block.driverPool)[role].Model+`" }`)
				if err != nil {
					t.Fatalf("templates/base/ralph.toml: %v — update this test", err)
				}
				drift, err = roleModelsTemplateDrift([]byte(changed), DefaultRoleModels)
				wantOneRoleModelsDrift(t, "template driver changed", drift, err, wantPrefix+"template = ")

				removed, err := editRoleModelsTemplateRole(template, block, role, "")
				if err != nil {
					t.Fatalf("templates/base/ralph.toml: %v — update this test", err)
				}
				drift, err = roleModelsTemplateDrift([]byte(removed), DefaultRoleModels)
				wantOneRoleModelsDrift(t, "template role removed", drift, err, wantPrefix+"missing from the template")

				drift, err = roleModelsTemplateDrift([]byte(template), changedDefaults(block.driverPool, func(set map[string]OrgRoleModel) {
					delete(set, role)
				}))
				wantOneRoleModelsDrift(t, "DefaultRoleModels role removed", drift, err, wantPrefix+"in the template, not in DefaultRoleModels")
			})
		}

		t.Run(block.label+"/an extra role", func(t *testing.T) {
			wantPrefix := "org.role_models.qa_bot (" + block.label + "): "
			added, err := insertRoleModelsTemplateRole(template, block, "qa_bot", `{ driver = "codex", model = "gpt-6.1-sol" }`)
			if err != nil {
				t.Fatalf("templates/base/ralph.toml: %v — update this test", err)
			}
			drift, err := roleModelsTemplateDrift([]byte(added), DefaultRoleModels)
			wantOneRoleModelsDrift(t, "template role added", drift, err, wantPrefix+"in the template, not in DefaultRoleModels")

			drift, err = roleModelsTemplateDrift([]byte(template), changedDefaults(block.driverPool, func(set map[string]OrgRoleModel) {
				set["qa_bot"] = OrgRoleModel{Driver: "codex", Model: "gpt-6.1-sol"}
			}))
			wantOneRoleModelsDrift(t, "DefaultRoleModels role added", drift, err, wantPrefix+"missing from the template")
		})
	}
}

// TestRoleModelsTemplateDrift_DetectsASetTableAndMalformedBlocks pins the
// drifts that are not about one role: [org.role_models] set in the template
// (even with the default values, and by uncommenting a block's header in
// place), a block whose entry fails Load's per-file check, a reworded marker,
// a marker followed by another line, and a third commented block. A block
// whose marker no longer leads to its header also leaves that header without
// a marker, so those cases give two lines. Each case replaces text that must
// occur exactly once, so a rewording of the template fails the case instead
// of making it vacuous.
func TestRoleModelsTemplateDrift_DetectsASetTableAndMalformedBlocks(t *testing.T) {
	template := readTemplateForRoleModels(t)
	claude, codex := roleModelsTemplateBlocks[0], roleModelsTemplateBlocks[1]
	for _, tc := range []struct {
		name     string
		old, new string   // old == "" appends new to the template
		want     []string // the prefix of each drift line, in order
	}{
		{
			name: "set with the default values",
			new:  "\n[org.role_models]\nleader = { driver = \"claude\", model = \"opus\" }\n",
			want: []string{"org.role_models: set in the template, want it only as comments"},
		},
		{
			name: "set as an empty table",
			new:  "\n[org.role_models]\n",
			want: []string{"org.role_models: set in the template, want it only as comments"},
		},
		{
			name: "a block's header uncommented in place",
			old:  claude.marker + "\n" + roleModelsTemplateHeader + "\n",
			new:  claude.marker + "\n[org.role_models]\n",
			want: []string{
				"org.role_models: set in the template, want it only as comments",
				"org.role_models (the claude set): the line after ",
			},
		},
		{
			name: "an entry without model",
			old:  "# director = { driver = \"claude\", model = \"\" }\n",
			new:  "# director = { driver = \"claude\" }\n",
			want: []string{"org.role_models (the claude set): templates/base/ralph.toml: [org.role_models].director must set both driver and model"},
		},
		{
			name: "an entry with a driver ralph does not know",
			old:  "# leader = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n",
			new:  "# leader = { driver = \"gemini\", model = \"gpt-6.1-sol\" }\n",
			want: []string{"org.role_models (the codex set): templates/base/ralph.toml: [org.role_models].leader driver \"gemini\" must be"},
		},
		{
			name: "a reworded marker",
			old:  claude.marker + "\n",
			new:  "# The claude defaults:\n",
			want: []string{
				"org.role_models (the claude set): the template has the line ",
				"org.role_models: the template's line ",
			},
		},
		{
			name: "a line between a marker and its header",
			old:  codex.marker + "\n" + roleModelsTemplateHeader + "\n",
			new:  codex.marker + "\n#\n" + roleModelsTemplateHeader + "\n",
			want: []string{
				"org.role_models (the codex set): the line after ",
				"org.role_models: the template's line ",
			},
		},
		{
			name: "a third commented block",
			new:  "\n" + roleModelsTemplateHeader + "\n# leader = { driver = \"claude\", model = \"opus\" }\n",
			want: []string{"org.role_models: the template's line "},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := template + tc.new
			if tc.old != "" {
				if n := strings.Count(template, tc.old); n != 1 {
					t.Fatalf("templates/base/ralph.toml contains %q %d times, want exactly 1 — update this case", tc.old, n)
				}
				mutated = strings.Replace(template, tc.old, tc.new, 1)
			}
			drift, err := roleModelsTemplateDrift([]byte(mutated), DefaultRoleModels)
			if err != nil {
				t.Fatalf("roleModelsTemplateDrift: %v", err)
			}
			if len(drift) != len(tc.want) {
				t.Fatalf("drift = %q, want %d lines starting with %q", drift, len(tc.want), tc.want)
			}
			for i, want := range tc.want {
				if !strings.HasPrefix(drift[i], want) {
					t.Errorf("drift[%d] = %q, want it to start with %q", i, drift[i], want)
				}
			}
		})
	}
}
