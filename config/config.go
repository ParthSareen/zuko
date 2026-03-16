package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Tool struct {
	RealBinary string              `yaml:"real_binary"`
	AllowBare  bool                `yaml:"allow_bare"`
	AllowAll   bool                `yaml:"allow_all,omitempty"`
	Allow      [][]string          `yaml:"allow"`
	Locked     [][]string          `yaml:"locked,omitempty"`
	DenyFlags  map[string][]string `yaml:"deny_flags"`
}

type Config struct {
	ShimDir        string          `yaml:"shim_dir"`
	TimeoutMinutes int             `yaml:"timeout_minutes,omitempty"`
	Tools          map[string]Tool `yaml:"tools"`
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zuko")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("config not found — run 'zuko setup' first: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	(&cfg).ExpandPaths()
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}

// ExpandPaths resolves ~ in all path fields.
func (c *Config) ExpandPaths() {
	c.ShimDir = expandHome(c.ShimDir)
	for name, tool := range c.Tools {
		tool.RealBinary = expandHome(tool.RealBinary)
		c.Tools[name] = tool
	}
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
