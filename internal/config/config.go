package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
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
	Interval int    `yaml:"interval"`
	LogLevel string `yaml:"log_level"`
	DryRun   bool   `yaml:"dry_run"`
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
	Name     string          `yaml:"name"`
	Enabled  bool            `yaml:"enabled"`
	Action   string          `yaml:"action"`
	Criteria CriteriaConfig  `yaml:"criteria"`
}

// CriteriaConfig defines filtering criteria
type CriteriaConfig struct {
	Flag       []string         `yaml:"flag,omitempty"`
	Progress   *ProgressCriteria  `yaml:"progress,omitempty"`
	Uploaded   *BytesCriteria     `yaml:"uploaded,omitempty"`
	Downloaded *BytesCriteria     `yaml:"downloaded,omitempty"`
	Relevance  *RangeCriteria     `yaml:"relevance,omitempty"`
	ActiveTime *TimeCriteria       `yaml:"active_time,omitempty"`
}

// ProgressCriteria defines progress percentage criteria
type ProgressCriteria struct {
	Min float64 `yaml:"min,omitempty"`
	Max float64 `yaml:"max,omitempty"`
}

// BytesCriteria defines byte size criteria
type BytesCriteria struct {
	Mode         string  `yaml:"mode,omitempty"`
	Min          string  `yaml:"min,omitempty"`
	Max          string  `yaml:"max,omitempty"`
	MinPercent   float64 `yaml:"min_percent,omitempty"`
	MaxPercent   float64 `yaml:"max_percent,omitempty"`
}

// RangeCriteria defines a value range criteria
type RangeCriteria struct {
	Min float64 `yaml:"min,omitempty"`
	Max float64 `yaml:"max,omitempty"`
}

// TimeCriteria defines time duration criteria
type TimeCriteria struct {
	Min string `yaml:"min,omitempty"`
	Max string `yaml:"max,omitempty"`
}

// GetInterval returns the check interval as a duration
func (a *AppConfig) GetInterval() time.Duration {
	if a.Interval <= 0 {
		return 30 * time.Minute
	}
	return time.Duration(a.Interval) * time.Minute
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
