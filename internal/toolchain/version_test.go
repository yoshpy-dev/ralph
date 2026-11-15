package toolchain

import (
	"strings"
	"testing"
)

func TestVersionToken_ReadsEachToolsOutputShape(t *testing.T) {
	tests := []struct {
		tool, line string
		want       string
		ok         bool
	}{
		{"node", "v24.22.1", "v24.22.1", true},
		{"node", "  v24.22.1\r", "v24.22.1", true},
		{"pnpm", "10.10.0", "10.10.0", true},
		{"yarn", "4.5.0", "4.5.0", true},
		{"yarn", "1.22.22", "1.22.22", true},
		{"npm", "10.9.2", "10.9.2", true},
		{"python3", "Python 3.12.1", "3.12.1", true},
		{"python3", "Python 3.13.0rc1", "3.13.0rc1", true},
		{"python3", "3.12.1", "3.12.1", true},
		{"go", "go version go1.27.3 darwin/arm64", "go1.27.3", true},
		{"go", "go version go1.27.3 linux/amd64\r", "go1.27.3", true},
		{"node", "1.0.0+build.7", "1.0.0+build.7", true},

		{"node", "", "", false},
		{"node", "   ", "", false},
		{"node", "mise ERROR node@24 is not installed", "", false},
		{"node", "v24 .1", "", false},
		{"node", "v24\x1b[0m", "", false},
		{"node", "v24.1\x00", "", false},
		{"node", "hello", "", false},
		{"node", "v" + strings.Repeat("1", 64), "", false},
		{"python3", "Python", "", false},
		{"python3", "Python 3.12.1 (main)", "", false},
		{"go", "go version devel go1.28-abc Tue Jan 1", "", false},
		{"go", "go: downloading go1.28 (darwin/arm64)", "", false},
	}
	for _, tt := range tests {
		got, ok := VersionToken(tt.tool, tt.line)
		if got != tt.want || ok != tt.ok {
			t.Errorf("VersionToken(%q, %q) = %q, %v; want %q, %v", tt.tool, tt.line, got, ok, tt.want, tt.ok)
		}
	}
}

func TestShimOf_NamesMiseAndAsdfShimsOnly(t *testing.T) {
	tests := []struct{ path, want string }{
		{"/Users/a/.local/share/mise/shims/node", ShimMise},
		{"/opt/mise/shims/pnpm", ShimMise},
		{"/a//mise/shims//node", ShimMise},
		{"/Users/a/.asdf/shims/python3", ShimAsdf},
		{"/Users/a/.local/share/mise/installs/node/24.22.1/bin/node", ""},
		{"/usr/local/bin/node", ""},
		{"/Users/a/.local/share/mise/shims", ""},
		{"/Users/a/asdf/shims/node", ""},
		{"node", ""},
		{"alias node='/x/mise/shims/node'", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ShimOf(tt.path); got != tt.want {
			t.Errorf("ShimOf(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestCleanText_ReplacesControlCharactersAndCuts(t *testing.T) {
	tests := []struct {
		in    string
		limit int
		want  string
	}{
		{"plain text", 200, "plain text"},
		{"tab\there\nnewline", 200, "tab?here?newline"},
		{"esc\x1b[31m red", 200, "esc?[31m red"},
		{"bad\xffbyte", 200, "bad?byte"},
		{"c1\u0085char", 200, "c1?char"},
		{"日本語のテキスト", 200, "日本語のテキスト"},
		{"abcdefghij", 8, "abcde..."},
		{"日本語のテキスト", 10, "日本..."},
		{"exactly10!", 10, "exactly10!"},
	}
	for _, tt := range tests {
		got := cleanText(tt.in, tt.limit)
		if got != tt.want {
			t.Errorf("cleanText(%q, %d) = %q, want %q", tt.in, tt.limit, got, tt.want)
		}
		if len(got) > tt.limit {
			t.Errorf("cleanText(%q, %d) is %d bytes, over the limit", tt.in, tt.limit, len(got))
		}
	}
}
