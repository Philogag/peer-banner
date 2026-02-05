package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Default values
const (
	DefaultStateFile = "bans.json"
)

// Config represents the application configuration
type Config struct {
	App       AppConfig       `yaml:"app"`
	Servers   []ServerConfig  `yaml:"servers"`
	Whitelist WhitelistConfig `yaml:"whitelist"`
	Output    OutputConfig    `yaml:"output"`
	Rules     []RuleConfig    `yaml:"rules"`
}

// AppConfig contains application-level settings
type AppConfig struct {
	Interval   int    `yaml:"interval"`
	LogLevel   string `yaml:"log_level"`
	DryRun     bool   `yaml:"dry_run"`
	StateFile  string `yaml:"state_file"`
}

// ServerConfig represents a qBittorrent server
type ServerConfig struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// WhitelistConfig contains IP whitelist
type WhitelistConfig struct {
	IPs []string `yaml:"ips"`
}

// OutputConfig defines DAT output settings
type OutputConfig struct {
	DATFile string `yaml:"dat_file"`
	Format  string `yaml:"format"`
}

// RuleConfig represents a leecher detection rule
type RuleConfig struct {
	Name         string        `yaml:"name"`
	Enabled      bool          `yaml:"enabled"`
	Action       string        `yaml:"action"`
	BanDuration  string        `yaml:"ban_duration"`
	MaxBanCount  int           `yaml:"max_ban_count"`
	Filters      []FilterConfig `yaml:"filter"`
}

// FilterConfig defines a single filter condition
type FilterConfig struct {
	Field    string `yaml:"field"`
	Operator string `yaml:"operator"` // <, >, <=, >=, include, exclude
	Value    string `yaml:"value"`
}

// GetInterval returns the check interval as a duration
func (a *AppConfig) GetInterval() time.Duration {
	if a.Interval <= 0 {
		return 30 * time.Minute
	}
	return time.Duration(a.Interval) * time.Minute
}

// GetStateFile returns the state file path
func (a *AppConfig) GetStateFile() string {
	if a.StateFile == "" {
		return DefaultStateFile
	}
	return a.StateFile
}

// GetBanDuration returns the ban duration as a duration
func (r *RuleConfig) GetBanDuration() (time.Duration, error) {
	if r.BanDuration == "" || r.BanDuration == "0" {
		return 0, nil // Permanent ban
	}
	return time.ParseDuration(r.BanDuration)
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.App.Interval <= 0 {
		cfg.App.Interval = 30
	}
	if cfg.App.LogLevel == "" {
		cfg.App.LogLevel = "info"
	}
	if cfg.Output.Format == "" {
		cfg.Output.Format = "peerbanana"
	}

	return &cfg, nil
}
