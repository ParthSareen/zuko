package config

func DefaultConfig() *Config {
	return &Config{
		ShimDir: "~/.config/zuko/shims",
		Tools: map[string]Tool{
			"gh": {
				AllowBare: true,
				Allow: [][]string{
					{"issue", "list"},
					{"issue", "view"},
					{"issue", "status"},
					{"pr", "list"},
					{"pr", "view"},
					{"pr", "status"},
					{"pr", "diff"},
					{"pr", "checks"},
					{"repo", "list"},
					{"repo", "view"},
					{"run", "list"},
					{"run", "view"},
					{"search", "issues"},
					{"search", "prs"},
					{"search", "repos"},
					{"search", "code"},
					{"search", "commits"},
					{"status"},
					{"api"},
					{"release", "list"},
					{"release", "view"},
					{"gist", "list"},
					{"gist", "view"},
					{"label", "list"},
					{"workflow", "list"},
					{"workflow", "view"},
				},
				DenyFlags: map[string][]string{
					"api": {"-X", "--method", "-f", "--raw-field", "-F", "--field", "--input"},
				},
			},
			"himalaya": {
				AllowBare: true,
				Allow: [][]string{
					{"account", "list"},
					{"folder", "list"},
					{"envelope", "list"},
					{"envelope", "get"},
					{"message", "read"},
					{"attachment", "download"},
					{"flag", "add"},
					{"flag", "remove"},
				},
				DenyFlags: map[string][]string{},
			},
		},
	}
}
