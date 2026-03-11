package proxy

import (
	"testing"

	"github.com/ParthSareen/zuko/config"
)

func TestCheck(t *testing.T) {
	tool := config.Tool{
		AllowBare: true,
		Allow: [][]string{
			{"issue", "list"},
			{"issue", "view"},
			{"pr", "list"},
			{"pr", "view"},
			{"status"},
			{"api"},
		},
		DenyFlags: map[string][]string{
			"api": {"-X", "--method", "-f", "--raw-field"},
		},
	}

	tests := []struct {
		name    string
		args    []string
		allowed bool
	}{
		{"bare invocation", nil, true},
		{"empty args", []string{}, true},
		{"allowed simple", []string{"issue", "list"}, true},
		{"allowed with flags", []string{"issue", "list", "--state", "open"}, true},
		{"allowed with leading flags", []string{"-R", "foo/bar", "issue", "list"}, true},
		{"allowed single subcommand", []string{"status"}, true},
		{"blocked command", []string{"issue", "create", "--title", "test"}, false},
		{"blocked unknown", []string{"repo", "delete"}, false},
		{"pr view allowed", []string{"pr", "view", "123"}, true},
		{"pr merge blocked", []string{"pr", "merge", "123"}, false},
		{"api allowed", []string{"api", "/repos/foo/bar"}, true},
		{"api -X POST denied", []string{"api", "-X", "POST", "/repos/foo/bar"}, false},
		{"api --method denied", []string{"api", "--method", "DELETE", "/repos/foo/bar"}, false},
		{"api -f denied", []string{"api", "-f", "body=test"}, false},
		{"api --raw-field denied", []string{"api", "--raw-field", "body=test"}, false},
		{"completely unknown", []string{"auth", "login"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := Check(tool, tt.args)
			if allowed != tt.allowed {
				t.Errorf("Check(%v) = %v, want %v", tt.args, allowed, tt.allowed)
			}
		})
	}
}

func TestCheckBareDisabled(t *testing.T) {
	tool := config.Tool{
		AllowBare: false,
		Allow:     [][]string{{"list"}},
	}

	allowed, _ := Check(tool, nil)
	if allowed {
		t.Error("bare invocation should be blocked when allow_bare is false")
	}

	allowed, _ = Check(tool, []string{"list"})
	if !allowed {
		t.Error("'list' should be allowed")
	}
}

func TestCheckPassthrough(t *testing.T) {
	tool := config.Tool{
		AllowBare: true,
		Allow:     [][]string{{}},
		DenyFlags: map[string][]string{},
	}

	tests := []struct {
		name string
		args []string
	}{
		{"bare", nil},
		{"any subcommand", []string{"anything", "goes"}},
		{"with flags", []string{"--foo", "bar", "baz"}},
		{"deeply nested", []string{"a", "b", "c", "d"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := Check(tool, tt.args)
			if !allowed {
				t.Errorf("passthrough tool should allow %v", tt.args)
			}
		})
	}
}

func TestExtractSubcommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"simple", []string{"issue", "list"}, []string{"issue", "list"}},
		{"with flags", []string{"-R", "foo/bar", "issue", "list"}, []string{"issue", "list"}},
		{"flag with equals", []string{"--repo=foo/bar", "issue", "list"}, []string{"issue", "list"}},
		{"trailing flags", []string{"issue", "list", "--state", "open"}, []string{"issue", "list"}},
		{"empty", nil, nil},
		{"only flags", []string{"-v", "--debug"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSubcommands(tt.args)
			if len(got) != len(tt.want) {
				t.Errorf("extractSubcommands(%v) = %v, want %v", tt.args, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractSubcommands(%v)[%d] = %q, want %q", tt.args, i, got[i], tt.want[i])
				}
			}
		})
	}
}
