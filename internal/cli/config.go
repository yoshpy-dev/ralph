package cli

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// ErrConfigKey is returned by `ralph config get` when its key is not a key of
// ralph.toml or names a table or an array rather than one value. ExitCode
// maps it to 2.
var ErrConfigKey = errors.New("not a key of one value in ralph.toml")

// ErrConfigLoad is returned by `ralph config get` when ralph.toml or the
// personal ralph.local.toml cannot be read, cannot be parsed, or fails
// config.Load's validation. ExitCode maps it to 3.
var ErrConfigLoad = errors.New("cannot be read or is invalid")

func newConfigCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read ralph.toml settings",
		Long: "ralph config reads the settings of ralph.toml. `ralph config get <key>`\n" +
			"prints one value, for the skills and seats that read a setting.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", "", "path to ralph.toml (default: ./ralph.toml; a file that does not exist gives the built-in defaults)")

	cmd.AddCommand(newConfigGetCmd(&configPath))

	return cmd
}

func newConfigGetCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print the value of one ralph.toml key",
		Long: "ralph config get prints the value of one key of ralph.toml: the file\n" +
			"--config names, else ./ralph.toml. The key is the dotted toml name, such\n" +
			"as standard.plan_advisory or org.review.max_cross_review_cycles. A key\n" +
			"the file does not set prints its built-in default, and so does every key\n" +
			"when the file does not exist. It prints the value on stdout: true or\n" +
			"false, a decimal number, or a string as written.\n" +
			"\n" +
			"ralph.local.toml, the git-ignored personal file, is applied on top of\n" +
			"ralph.toml key by key: a key it sets prints its value, also when\n" +
			"ralph.toml does not exist. It is read from the main worktree root of the\n" +
			"git repository the config file's directory is in, so a linked worktree\n" +
			"and a subdirectory read the same file, and from that directory itself\n" +
			"outside git. RALPH_LOCAL_CONFIG=off skips it. A ralph.local.toml that\n" +
			"cannot be read, cannot be parsed, or fails validation exits 3, like\n" +
			"ralph.toml.\n" +
			"\n" +
			"Exit codes, for the skills and seats that read a setting with it:\n" +
			"  0  the value was printed: use it.\n" +
			"  1  an older ralph without this verb exits 1 with\n" +
			"     unknown command \"config\" on stderr. Treat that, and only that, as\n" +
			"     off, and record that the setting was treated as off. Stop and report\n" +
			"     any other exit 1.\n" +
			"  2  the key is not a key of ralph.toml, or names a table or an array\n" +
			"     rather than one value: stop and report it, do not treat it as off.\n" +
			"  3  ralph.toml cannot be read, cannot be parsed, or fails validation:\n" +
			"     stop and report it, do not treat it as off.\n" +
			wrapHelpLines("A ralph.toml that sets a key only ralph.local.toml may set exits 3 too: "+
				strings.Join(config.LocalOnlyKeys(), ", ")+".", "     ", helpWrapWidth) +
			"\n" +
			"These are for a skill or seat that runs ralph config get itself, as for\n" +
			"the review switches of [standard] and [org.review]. The skills read\n" +
			"standard.max_pipeline_cycles and the four keys of [reviewer] with\n" +
			"ralph_setting of scripts/ralph-config.sh instead, which does not stop:\n" +
			"when it gets no value it goes on with the key's built-in default,\n" +
			"silently when ralph is not on PATH or is an older ralph without this\n" +
			"verb, and after one line on stderr otherwise, exits 2 and 3 included.\n" +
			"The scripts plan-visual.sh, xreview-helpers.sh,\n" +
			"detect-changed-languages.sh, and run-verify.sh read their keys with the\n" +
			"same helper: ralph_setting for plan_visual.browser, plan_visual.opener,\n" +
			"and verify.skip_branch_secret_scan, and ralph_setting_or_fail for\n" +
			"standard.base_branch and verify.base. ralph_setting_or_fail gives the\n" +
			"default only when ralph is not on PATH or is an older ralph without\n" +
			"this verb. On any other failure, exits 2 and 3 included, it writes one\n" +
			"line on stderr, prints nothing, and fails, so the script stops or takes\n" +
			"the safe side (the full verify scope) instead of using the default base.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := *configPath
			if path == "" {
				path = "ralph.toml"
			}
			cfg, err := config.Load(path)
			if err != nil {
				return fmt.Errorf("config get: %s: %w: %w", path, ErrConfigLoad, err)
			}
			value, err := configValue(cfg, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), value)
			return err
		},
	}
}

// helpWrapWidth is the width, indent included, that the help texts of
// ralph config get and ralph org escalate give wrapHelpLines.
const helpWrapWidth = 74

// wrapHelpLines breaks text at its spaces into lines of at most width
// columns, indent included, each starting with indent and ending in a line
// break, the way the rest of the help is wrapped by hand. A word longer than
// the width gets a line of its own.
func wrapHelpLines(text, indent string, width int) string {
	var b strings.Builder
	var line string
	for _, word := range strings.Fields(text) {
		if line != "" && len(indent)+len(line)+1+len(word) > width {
			b.WriteString(indent + line + "\n")
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	if line != "" {
		b.WriteString(indent + line + "\n")
	}
	return b.String()
}

// configValue returns the value of key, a dotted toml name, in cfg as text:
// it walks cfg's structs by their toml tags and formats the bool, int, or
// string it ends at (a string as it is, line breaks included). A key that is
// not a field, or that ends at or goes through a table or an array, is an
// ErrConfigKey.
func configValue(cfg config.Config, key string) (string, error) {
	v := reflect.ValueOf(cfg)
	segments := strings.Split(key, ".")
	for i, segment := range segments {
		if v.Kind() != reflect.Struct {
			return "", fmt.Errorf("config get: %s: %w: %s is %s", key, ErrConfigKey, strings.Join(segments[:i], "."), configKindName(v.Kind()))
		}
		field, ok := configField(v, segment)
		if !ok {
			return "", fmt.Errorf("config get: %s: %w: no such key", key, ErrConfigKey)
		}
		v = field
	}
	switch v.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.String:
		return v.String(), nil
	default:
		return "", fmt.Errorf("config get: %s: %w: it is %s", key, ErrConfigKey, configKindName(v.Kind()))
	}
}

// configField returns the field of the struct v whose toml tag names name.
func configField(v reflect.Value, name string) (reflect.Value, bool) {
	t := v.Type()
	for i := range t.NumField() {
		tag, _, _ := strings.Cut(t.Field(i).Tag.Get("toml"), ",")
		if tag == name && tag != "" && tag != "-" {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// configKindName names kind the way ralph.toml's reader sees it, with its
// article. A kind no field of config.Config has today is named by Go's name
// for it.
func configKindName(kind reflect.Kind) string {
	switch kind {
	case reflect.Struct, reflect.Map:
		return "a table"
	case reflect.Slice, reflect.Array:
		return "an array"
	case reflect.Bool:
		return "a boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "an integer"
	case reflect.String:
		return "a string"
	default:
		return "a value of Go kind " + kind.String()
	}
}
